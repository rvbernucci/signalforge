#!/usr/bin/env python3
"""Hydrate and build the pinned native Radeon toolchain under persistent storage."""

from __future__ import annotations

import argparse
import fcntl
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tarfile
from pathlib import Path, PurePosixPath
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

import radeon_model_cache


class NativeToolchainError(RuntimeError):
    pass


def load_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        while chunk := handle.read(1024 * 1024):
            digest.update(chunk)
    return digest.hexdigest()


def atomic_json(path: Path, payload: dict[str, Any], mode: int = 0o600) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    try:
        temporary.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        os.chmod(temporary, mode)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def ensure_directory(path: Path, mode: int = 0o700) -> None:
    if path.exists() and (not path.is_dir() or path.is_symlink()):
        raise NativeToolchainError(f"unsafe persistent directory: {path}")
    path.mkdir(parents=True, exist_ok=True)
    os.chmod(path, mode)


def remove_directory(path: Path) -> None:
    if path.is_symlink():
        raise NativeToolchainError(f"refusing symbolic-link directory: {path}")
    if path.exists():
        shutil.rmtree(path)


def safe_extract_go(archive: Path, destination: Path) -> None:
    if destination.exists():
        raise NativeToolchainError(f"extraction destination already exists: {destination}")
    destination.mkdir(parents=True, mode=0o700)
    try:
        with tarfile.open(archive, "r:gz") as bundle:
            members = bundle.getmembers()
            if not members:
                raise NativeToolchainError("Go archive is empty")
            for member in members:
                relative = PurePosixPath(member.name)
                if relative.is_absolute() or ".." in relative.parts or not relative.parts:
                    raise NativeToolchainError(f"unsafe Go archive member: {member.name}")
                if relative.parts[0] != "go":
                    raise NativeToolchainError(f"unexpected Go archive root: {member.name}")
                if member.issym() or member.islnk() or member.isdev() or member.isfifo():
                    raise NativeToolchainError(f"unsupported Go archive member type: {member.name}")
                target = destination.joinpath(*relative.parts)
                if member.isdir():
                    target.mkdir(parents=True, exist_ok=True)
                    os.chmod(target, member.mode & 0o777)
                    continue
                if not member.isfile():
                    raise NativeToolchainError(f"unsupported Go archive member: {member.name}")
                target.parent.mkdir(parents=True, exist_ok=True)
                source = bundle.extractfile(member)
                if source is None:
                    raise NativeToolchainError(f"cannot read Go archive member: {member.name}")
                with source, target.open("wb") as output:
                    shutil.copyfileobj(source, output, 1024 * 1024)
                    output.flush()
                    # The verified archive is disposable and promoted atomically after validation.
                    # Per-member fsync makes extraction pathological on Radeon persistent NFS.
                os.chmod(target, member.mode & 0o777)
    except Exception:
        remove_directory(destination)
        raise


def go_version(go_binary: Path) -> str | None:
    if not go_binary.is_file() or go_binary.is_symlink() or not os.access(go_binary, os.X_OK):
        return None
    try:
        result = subprocess.run(
            [str(go_binary), "version"],
            check=False,
            capture_output=True,
            text=True,
            timeout=10,
            env={**os.environ, "GOTOOLCHAIN": "local", "LC_ALL": "C"},
        )
    except (OSError, subprocess.TimeoutExpired):
        return None
    return result.stdout.strip() if result.returncode == 0 else None


def ensure_go(
    persist_root: Path,
    manifest: dict[str, Any],
    *,
    retries: int,
    timeout_seconds: float,
) -> tuple[Path, dict[str, Any]]:
    config = manifest["go"]
    install = persist_root / config["install_relative_path"]
    binary = install / "bin/go"
    marker = install / ".signalforge-go-ready.json"
    expected_version = f"go{config['version']}"
    if marker.is_file() and not marker.is_symlink():
        try:
            receipt = load_json(marker)
        except (OSError, json.JSONDecodeError):
            receipt = {}
        observed = go_version(binary)
        if (
            receipt.get("sha256") == config["sha256"]
            and receipt.get("version") == config["version"]
            and observed
            and expected_version in observed
        ):
            return binary, receipt

    downloads = persist_root / "cache/downloads"
    ensure_directory(downloads)
    archive = downloads / config["filename"]
    partial = downloads / f".{config['filename']}.part"
    if archive.exists() and (
        not archive.is_file()
        or archive.is_symlink()
        or archive.stat().st_size != config["expected_size_bytes"]
        or sha256_file(archive) != config["sha256"]
    ):
        archive.unlink()
    if not archive.exists():
        radeon_model_cache.download_resumable(
            config["url"],
            "",
            partial,
            config["expected_size_bytes"],
            retries,
            timeout_seconds,
        )
        if sha256_file(partial) != config["sha256"]:
            partial.unlink(missing_ok=True)
            raise NativeToolchainError("Go archive SHA-256 mismatch")
        os.chmod(partial, 0o444)
        os.replace(partial, archive)

    staging_parent = persist_root / "toolchains"
    ensure_directory(staging_parent)
    for stale in staging_parent.glob(f".go-{config['version']}.extract-*"):
        remove_directory(stale)
    staging = staging_parent / f".go-{config['version']}.extract-{os.getpid()}"
    safe_extract_go(archive, staging)
    extracted = staging / "go"
    observed = go_version(extracted / "bin/go")
    if not observed or expected_version not in observed:
        remove_directory(staging)
        raise NativeToolchainError(f"extracted Go version is invalid: {observed or 'unavailable'}")
    remove_directory(install)
    os.replace(extracted, install)
    remove_directory(staging)
    receipt = {
        "schema_version": "signalforge/native-go-ready/v1",
        "version": config["version"],
        "sha256": config["sha256"],
        "archive_size_bytes": config["expected_size_bytes"],
        "binary": str(binary),
    }
    atomic_json(marker, receipt, 0o444)
    return binary, receipt


def git_source_identity(root: Path, allow_dirty: bool) -> tuple[str, bool]:
    commit = subprocess.check_output(
        ["git", "-C", str(root), "rev-parse", "HEAD"], text=True, timeout=10
    ).strip()
    dirty = bool(
        subprocess.check_output(
            ["git", "-C", str(root), "status", "--porcelain"],
            text=True,
            timeout=10,
        ).strip()
    )
    if dirty and not allow_dirty:
        raise NativeToolchainError(
            "native release build requires a clean Git worktree; commit the reviewed source first"
        )
    return commit, dirty


def verify_source_locks(manifest: dict[str, Any]) -> None:
    app = manifest["application"]
    for path_key, sha_key in (
        ("package_lock_path", "package_lock_sha256"),
        ("go_sum_path", "go_sum_sha256"),
    ):
        path = ROOT / app[path_key]
        if not path.is_file() or sha256_file(path) != app[sha_key]:
            raise NativeToolchainError(f"source lock does not match native manifest: {path}")


def native_build_environment(
    go_binary: Path,
    persist_root: Path,
    manifest: dict[str, Any],
) -> dict[str, str]:
    environment = {**os.environ}
    go_root = go_binary.parents[1]
    environment.update(
        {
            "GOROOT": str(go_root),
            "GOTOOLCHAIN": "local",
            "GOMODCACHE": str(persist_root / "cache/go-mod"),
            "GOCACHE": str(persist_root / "cache/go-build"),
            "npm_config_cache": str(persist_root / "cache/npm"),
            "GOPROXY": str(manifest["application"]["go_proxy"]),
            "GOSUMDB": str(manifest["application"]["go_sumdb"]),
            "PATH": f"{go_root / 'bin'}:{environment.get('PATH', '')}",
            "LC_ALL": "C",
        }
    )
    return environment


def run_logged(
    command: list[str],
    *,
    cwd: Path,
    environment: dict[str, str],
    log_path: Path,
    timeout: float,
) -> None:
    log_path.parent.mkdir(parents=True, exist_ok=True)
    if log_path.exists() and (not log_path.is_file() or log_path.is_symlink()):
        raise NativeToolchainError(f"unsafe build log path: {log_path}")
    with log_path.open("a", encoding="utf-8") as log:
        os.chmod(log_path, 0o600)
        result = subprocess.run(
            command,
            cwd=cwd,
            env=environment,
            stdout=log,
            stderr=subprocess.STDOUT,
            text=True,
            timeout=timeout,
            check=False,
        )
    if result.returncode:
        raise NativeToolchainError(
            f"native build command failed with exit code {result.returncode}: {command[0]}"
        )


def build_application(
    persist_root: Path,
    manifest: dict[str, Any],
    go_binary: Path,
    *,
    allow_dirty: bool,
    timeout_seconds: float,
) -> tuple[Path, dict[str, Any]]:
    verify_source_locks(manifest)
    commit, dirty = git_source_identity(ROOT, allow_dirty)
    app = manifest["application"]
    binary = persist_root / app["binary_relative_path"]
    marker = persist_root / "state/native/app-build.json"
    if marker.is_file() and binary.is_file() and not binary.is_symlink():
        try:
            receipt = load_json(marker)
        except (OSError, json.JSONDecodeError):
            receipt = {}
        if (
            receipt.get("source_commit") == commit
            and receipt.get("package_lock_sha256") == app["package_lock_sha256"]
            and receipt.get("go_sum_sha256") == app["go_sum_sha256"]
            and receipt.get("binary_sha256") == sha256_file(binary)
        ):
            return binary, receipt

    environment = native_build_environment(go_binary, persist_root, manifest)
    logs = persist_root / "state/native/logs"
    ensure_directory(logs)
    run_logged(
        ["npm", "ci", "--no-audit", "--no-fund"],
        cwd=ROOT / "web",
        environment=environment,
        log_path=logs / "native-build.log",
        timeout=timeout_seconds,
    )
    run_logged(
        ["npm", "run", "build"],
        cwd=ROOT / "web",
        environment=environment,
        log_path=logs / "native-build.log",
        timeout=timeout_seconds,
    )
    if not (ROOT / "web/dist/index.html").is_file():
        raise NativeToolchainError("frontend build did not publish web/dist/index.html")
    ensure_directory(binary.parent)
    temporary = binary.with_name(f".{binary.name}.{os.getpid()}.tmp")
    try:
        run_logged(
            [
                str(go_binary),
                "build",
                "-trimpath",
                "-ldflags",
                f"-s -w -X main.buildCommit={commit}",
                "-o",
                str(temporary),
                app["command"],
            ],
            cwd=ROOT,
            environment=environment,
            log_path=logs / "native-build.log",
            timeout=timeout_seconds,
        )
        os.chmod(temporary, 0o555)
        os.replace(temporary, binary)
    finally:
        temporary.unlink(missing_ok=True)
    receipt = {
        "schema_version": "signalforge/native-app-build/v1",
        "source_commit": commit,
        "source_dirty": dirty,
        "go_version": manifest["go"]["version"],
        "package_lock_sha256": app["package_lock_sha256"],
        "go_sum_sha256": app["go_sum_sha256"],
        "binary_sha256": sha256_file(binary),
        "binary": str(binary),
    }
    atomic_json(marker, receipt)
    return binary, receipt


def build_llama(
    persist_root: Path,
    manifest: dict[str, Any],
    *,
    timeout_seconds: float,
) -> tuple[Path, dict[str, Any]]:
    config = manifest["llama_cpp"]
    source = persist_root / config["source_relative_path"]
    binary = persist_root / config["server_relative_path"]
    marker = persist_root / "state/native/llama-build.json"
    if marker.is_file() and binary.is_file() and not binary.is_symlink():
        try:
            receipt = load_json(marker)
        except (OSError, json.JSONDecodeError):
            receipt = {}
        if (
            receipt.get("revision") == config["revision"]
            and receipt.get("binary_sha256") == sha256_file(binary)
        ):
            return binary, receipt

    environment = {
        **os.environ,
        "LLAMA_CPP_DIR": str(source),
        "LLAMA_CPP_REPOSITORY": config["repository"],
        "BUILD_JOBS": os.environ.get("BUILD_JOBS", str(max(1, min(os.cpu_count() or 1, 32)))),
        "LC_ALL": "C",
    }
    logs = persist_root / "state/native/logs"
    ensure_directory(logs)
    run_logged(
        [str(ROOT / "scripts/build_llama_rocm.sh")],
        cwd=ROOT,
        environment=environment,
        log_path=logs / "native-build.log",
        timeout=timeout_seconds,
    )
    observed = subprocess.check_output(
        ["git", "-C", str(source), "rev-parse", "HEAD"], text=True, timeout=10
    ).strip()
    if observed != config["revision"] or not binary.is_file() or not os.access(binary, os.X_OK):
        raise NativeToolchainError("llama.cpp native build identity is invalid")
    receipt = {
        "schema_version": "signalforge/native-llama-build/v1",
        "repository": config["repository"],
        "revision": config["revision"],
        "binary_sha256": sha256_file(binary),
        "binary": str(binary),
    }
    atomic_json(marker, receipt)
    return binary, receipt


def prepare(
    persist_root: Path,
    profile: str,
    manifest: dict[str, Any],
    *,
    allow_dirty: bool,
    retries: int,
    download_timeout_seconds: float,
    build_timeout_seconds: float,
) -> dict[str, Any]:
    ensure_directory(persist_root)
    lock_path = persist_root / "state/native/toolchain.lock"
    ensure_directory(lock_path.parent)
    with lock_path.open("a+b") as lock:
        fcntl.flock(lock.fileno(), fcntl.LOCK_EX)
        go_binary, go_receipt = ensure_go(
            persist_root,
            manifest,
            retries=retries,
            timeout_seconds=download_timeout_seconds,
        )
        app_binary, app_receipt = build_application(
            persist_root,
            manifest,
            go_binary,
            allow_dirty=allow_dirty,
            timeout_seconds=build_timeout_seconds,
        )
        llama_receipt = None
        if profile in {"radeon-local", "championship"}:
            _, llama_receipt = build_llama(
                persist_root, manifest, timeout_seconds=build_timeout_seconds
            )
        result = {
            "schema_version": "signalforge/native-toolchain-status/v1",
            "status": "ready",
            "profile": profile,
            "go": go_receipt,
            "application": app_receipt,
            "llama_cpp": llama_receipt,
            "application_binary": str(app_binary),
        }
        atomic_json(persist_root / "state/native/toolchain-status.json", result)
        return result


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--persist-root", type=Path, required=True)
    parser.add_argument("--profile", choices=("fixture", "radeon-local", "championship"), required=True)
    parser.add_argument(
        "--manifest",
        type=Path,
        default=ROOT / "deploy/radeon/native-toolchain-manifest.json",
    )
    parser.add_argument("--allow-dirty", action="store_true")
    parser.add_argument("--retries", type=int, default=5)
    parser.add_argument("--download-timeout-seconds", type=float, default=60)
    parser.add_argument("--build-timeout-seconds", type=float, default=1800)
    args = parser.parse_args()
    try:
        result = prepare(
            args.persist_root.expanduser().resolve(),
            args.profile,
            load_json(args.manifest),
            allow_dirty=args.allow_dirty,
            retries=args.retries,
            download_timeout_seconds=args.download_timeout_seconds,
            build_timeout_seconds=args.build_timeout_seconds,
        )
    except (
        NativeToolchainError,
        radeon_model_cache.ModelCacheError,
        OSError,
        subprocess.SubprocessError,
        tarfile.TarError,
    ) as error:
        print(f"Native toolchain failed: {error}", file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
