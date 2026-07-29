#!/usr/bin/env python3
"""Hydrate and verify the pinned SignalForge model cache without exposing credentials."""

from __future__ import annotations

import argparse
import contextlib
import fcntl
import hashlib
import json
import os
import shutil
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import BinaryIO, Callable


CHUNK_BYTES = 8 * 1024 * 1024
USER_AGENT = "SignalForge-Radeon-Appliance/1"


class ModelCacheError(RuntimeError):
    """A safe, operator-actionable model cache failure."""


class SafeRedirectHandler(urllib.request.HTTPRedirectHandler):
    """Prevent a provider token from crossing host boundaries on signed redirects."""

    def redirect_request(self, request, fp, code, message, headers, new_url):
        redirected = super().redirect_request(request, fp, code, message, headers, new_url)
        if redirected is None:
            return None
        old_host = urllib.parse.urlsplit(request.full_url).netloc.lower()
        new_host = urllib.parse.urlsplit(new_url).netloc.lower()
        if old_host != new_host:
            redirected.remove_header("Authorization")
        return redirected


def load_manifest(path: Path) -> dict[str, object]:
    data = json.loads(path.read_text(encoding="utf-8"))
    required = {
        "schema_version",
        "manifest_version",
        "model_id",
        "served_model_id",
        "revision",
        "filename",
        "expected_size_bytes",
        "sha256",
        "license",
        "sources",
        "cache",
    }
    missing = sorted(required - data.keys())
    if missing:
        raise ModelCacheError(f"model manifest is missing fields: {', '.join(missing)}")
    if data["schema_version"] != "signalforge/model-artifact/v1":
        raise ModelCacheError("unsupported model manifest schema")
    expected_hash = str(data["sha256"])
    if len(expected_hash) != 64 or any(character not in "0123456789abcdef" for character in expected_hash):
        raise ModelCacheError("model manifest SHA-256 is invalid")
    if int(data["expected_size_bytes"]) <= 0:
        raise ModelCacheError("model manifest expected size must be positive")
    cache = data["cache"]
    if not isinstance(cache, dict):
        raise ModelCacheError("model manifest cache contract is invalid")
    for field in (
        "model_relative_path",
        "partial_relative_path",
        "ready_marker_relative_path",
        "lock_relative_path",
    ):
        candidate = Path(str(cache.get(field, "")))
        if not candidate.parts or candidate.is_absolute() or ".." in candidate.parts:
            raise ModelCacheError(f"model manifest cache path is unsafe: {field}")
    return data


def atomic_write_json(path: Path, payload: dict[str, object], mode: int = 0o444) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    try:
        with temporary.open("w", encoding="utf-8") as handle:
            json.dump(payload, handle, indent=2, sort_keys=True)
            handle.write("\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.chmod(temporary, mode)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        while chunk := handle.read(CHUNK_BYTES):
            digest.update(chunk)
    return digest.hexdigest()


def validate_model(path: Path, manifest: dict[str, object]) -> tuple[bool, str]:
    if not path.is_file() or path.is_symlink():
        return False, "model file is absent or not a regular file"
    actual_size = path.stat().st_size
    expected_size = int(manifest["expected_size_bytes"])
    if actual_size != expected_size:
        return False, f"model size mismatch: expected {expected_size}, observed {actual_size}"
    actual_hash = sha256_file(path)
    if actual_hash != manifest["sha256"]:
        return False, "model SHA-256 mismatch"
    return True, "verified"


def ready_payload(manifest: dict[str, object], source: str) -> dict[str, object]:
    return {
        "schema_version": "signalforge/model-ready/v1",
        "manifest_version": manifest["manifest_version"],
        "model_id": manifest["model_id"],
        "served_model_id": manifest["served_model_id"],
        "revision": manifest["revision"],
        "filename": manifest["filename"],
        "expected_size_bytes": manifest["expected_size_bytes"],
        "sha256": manifest["sha256"],
        "source": source,
        "verified": True,
    }


def marker_matches(path: Path, manifest: dict[str, object]) -> bool:
    if not path.is_file() or path.is_symlink():
        return False
    try:
        marker = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return False
    expected = ready_payload(manifest, str(marker.get("source", "unknown")))
    return all(marker.get(key) == value for key, value in expected.items() if key != "source")


def read_token(path: Path | None) -> str:
    if path is None:
        raise ModelCacheError("a file-mounted Hugging Face read token is required")
    if not path.is_file() or path.is_symlink():
        raise ModelCacheError("the Hugging Face token file is absent or unsafe")
    token = path.read_text(encoding="utf-8").strip()
    if not token:
        raise ModelCacheError("the Hugging Face token file is empty")
    return token


def ensure_license(manifest: dict[str, object], accepted: bool) -> None:
    license_data = manifest["license"]
    assert isinstance(license_data, dict)
    if license_data.get("acceptance_required") and not accepted:
        raise ModelCacheError(
            "Gemma license acceptance is required before hydration; "
            "run the documented bootstrap with explicit acceptance"
        )


def copy_existing(source: Path, partial: Path) -> None:
    if not source.is_file() or source.is_symlink():
        raise ModelCacheError("existing model source is absent or unsafe")
    partial.parent.mkdir(parents=True, exist_ok=True)
    temporary = partial.with_name(f"{partial.name}.copy-{os.getpid()}")
    try:
        with source.open("rb") as reader, temporary.open("wb") as writer:
            shutil.copyfileobj(reader, writer, CHUNK_BYTES)
            writer.flush()
            os.fsync(writer.fileno())
        os.replace(temporary, partial)
    finally:
        temporary.unlink(missing_ok=True)


def open_download(
    url: str,
    token: str | None,
    offset: int,
    timeout_seconds: float,
) -> tuple[BinaryIO, int]:
    headers = {"User-Agent": USER_AGENT}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if offset:
        headers["Range"] = f"bytes={offset}-"
    request = urllib.request.Request(url, headers=headers)
    opener = urllib.request.build_opener(SafeRedirectHandler())
    response = opener.open(request, timeout=timeout_seconds)
    return response, response.status


def download_resumable(
    url: str,
    token: str,
    partial: Path,
    expected_size: int,
    retries: int,
    timeout_seconds: float,
    opener: Callable[[str, str | None, int, float], tuple[BinaryIO, int]] = open_download,
) -> None:
    partial.parent.mkdir(parents=True, exist_ok=True)
    if partial.exists() and (not partial.is_file() or partial.is_symlink()):
        raise ModelCacheError("partial download path is unsafe")
    if partial.exists() and partial.stat().st_size > expected_size:
        partial.unlink()

    last_error: Exception | None = None
    for attempt in range(1, retries + 1):
        offset = partial.stat().st_size if partial.exists() else 0
        if offset == expected_size:
            return
        try:
            response, status = opener(url, token, offset, timeout_seconds)
            with contextlib.closing(response):
                if offset and status != 206:
                    partial.unlink(missing_ok=True)
                    offset = 0
                mode = "ab" if offset and status == 206 else "wb"
                with partial.open(mode) as handle:
                    while chunk := response.read(CHUNK_BYTES):
                        handle.write(chunk)
                    handle.flush()
                    os.fsync(handle.fileno())
            observed = partial.stat().st_size
            if observed > expected_size:
                partial.unlink()
                raise ModelCacheError("download exceeded the pinned model size")
            if observed == expected_size:
                return
            last_error = ModelCacheError(
                f"download ended early at {observed} of {expected_size} bytes"
            )
        except (OSError, urllib.error.URLError, ModelCacheError) as error:
            last_error = error
        if attempt < retries:
            time.sleep(min(2 ** (attempt - 1), 8))
    raise ModelCacheError(f"model download failed after {retries} attempts: {last_error}")


def publish_verified(
    partial: Path,
    model: Path,
    marker: Path,
    manifest: dict[str, object],
    source: str,
) -> None:
    valid, reason = validate_model(partial, manifest)
    if not valid:
        marker.unlink(missing_ok=True)
        partial.unlink(missing_ok=True)
        raise ModelCacheError(f"downloaded model was rejected: {reason}")
    model.parent.mkdir(parents=True, exist_ok=True)
    os.chmod(partial, 0o444)
    os.replace(partial, model)
    atomic_write_json(marker, ready_payload(manifest, source))


def hydrate(
    *,
    manifest_path: Path,
    cache_dir: Path,
    source: str,
    token_file: Path | None,
    existing_file: Path | None,
    license_accepted: bool,
    retries: int,
    timeout_seconds: float,
    state_path: Path | None = None,
) -> dict[str, object]:
    manifest = load_manifest(manifest_path)
    cache = manifest["cache"]
    assert isinstance(cache, dict)
    model = cache_dir / str(cache["model_relative_path"])
    partial = cache_dir / str(cache["partial_relative_path"])
    marker = cache_dir / str(cache["ready_marker_relative_path"])
    lock = cache_dir / str(cache["lock_relative_path"])
    if cache_dir.exists() and (not cache_dir.is_dir() or cache_dir.is_symlink()):
        raise ModelCacheError("model cache directory is unsafe")
    cache_dir.mkdir(parents=True, exist_ok=True)
    os.chmod(cache_dir, 0o755)
    lock.parent.mkdir(parents=True, exist_ok=True)

    with lock.open("a+b") as lock_handle:
        fcntl.flock(lock_handle.fileno(), fcntl.LOCK_EX)
        if marker_matches(marker, manifest):
            valid, reason = validate_model(model, manifest)
            if valid:
                result = {
                    "schema_version": "signalforge/model-init-status/v1",
                    "status": "ready",
                    "cache": "reused",
                    "model_id": manifest["served_model_id"],
                    "manifest_version": manifest["manifest_version"],
                    "sha256": manifest["sha256"],
                    "size_bytes": manifest["expected_size_bytes"],
                }
                if state_path:
                    atomic_write_json(state_path, result)
                return result
            marker.unlink(missing_ok=True)
            model.unlink(missing_ok=True)

        if model.exists():
            valid, _ = validate_model(model, manifest)
            if valid:
                atomic_write_json(marker, ready_payload(manifest, "verified-existing-cache"))
                result = {
                    "schema_version": "signalforge/model-init-status/v1",
                    "status": "ready",
                    "cache": "repaired-marker",
                    "model_id": manifest["served_model_id"],
                    "manifest_version": manifest["manifest_version"],
                    "sha256": manifest["sha256"],
                    "size_bytes": manifest["expected_size_bytes"],
                }
                if state_path:
                    atomic_write_json(state_path, result)
                return result
            model.unlink()
            marker.unlink(missing_ok=True)

        ensure_license(manifest, license_accepted)
        sources = manifest["sources"]
        assert isinstance(sources, dict)
        selected = source
        if source == "auto":
            selected = "existing" if existing_file else "huggingface"
        source_config = sources.get(selected)
        if not isinstance(source_config, dict) or not source_config.get("enabled"):
            raise ModelCacheError(f"model source is unavailable under the reviewed manifest: {selected}")

        if selected == "existing":
            if existing_file is None:
                raise ModelCacheError("existing source requires --existing-file")
            copy_existing(existing_file, partial)
        elif selected == "huggingface":
            token = read_token(token_file)
            download_resumable(
                str(source_config["url"]),
                token,
                partial,
                int(manifest["expected_size_bytes"]),
                retries,
                timeout_seconds,
            )
        elif selected == "oci":
            raise ModelCacheError("OCI model hydration is disabled by the current rights decision")
        else:
            raise ModelCacheError(f"unsupported model source: {selected}")

        publish_verified(partial, model, marker, manifest, selected)
        result = {
            "schema_version": "signalforge/model-init-status/v1",
            "status": "ready",
            "cache": "hydrated",
            "source": selected,
            "model_id": manifest["served_model_id"],
            "manifest_version": manifest["manifest_version"],
            "sha256": manifest["sha256"],
            "size_bytes": manifest["expected_size_bytes"],
        }
        if state_path:
            atomic_write_json(state_path, result)
        return result


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", type=Path, required=True)
    parser.add_argument("--cache-dir", type=Path, required=True)
    parser.add_argument(
        "--source",
        choices=("auto", "existing", "huggingface", "oci"),
        default="auto",
    )
    parser.add_argument("--token-file", type=Path)
    parser.add_argument("--existing-file", type=Path)
    parser.add_argument("--license-accepted", action="store_true")
    parser.add_argument("--license-acceptance", choices=("yes", "no"), default="no")
    parser.add_argument("--retries", type=int, default=5)
    parser.add_argument("--timeout-seconds", type=float, default=60)
    parser.add_argument("--state", type=Path)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.retries < 1 or args.retries > 10:
        print("model cache error: retries must be between 1 and 10", file=sys.stderr)
        return 2
    if args.timeout_seconds <= 0 or args.timeout_seconds > 600:
        print("model cache error: timeout must be between 0 and 600 seconds", file=sys.stderr)
        return 2
    try:
        result = hydrate(
            manifest_path=args.manifest,
            cache_dir=args.cache_dir,
            source=args.source,
            token_file=args.token_file,
            existing_file=args.existing_file,
            license_accepted=args.license_accepted or args.license_acceptance == "yes",
            retries=args.retries,
            timeout_seconds=args.timeout_seconds,
            state_path=args.state,
        )
    except (ModelCacheError, OSError, json.JSONDecodeError) as error:
        print(f"model cache error: {error}", file=sys.stderr)
        return 1
    print(
        f"model cache {result['status']}: {result['model_id']} "
        f"({result['cache']}, sha256={str(result['sha256'])[:12]}...)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
