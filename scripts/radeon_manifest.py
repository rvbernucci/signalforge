#!/usr/bin/env python3
"""Resolve and validate one immutable Radeon appliance manifest authority."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import stat
from pathlib import Path
from typing import Any, Mapping, NamedTuple


ROOT = Path(__file__).resolve().parents[1]
MANIFEST_DIRECTORY = ROOT / "deploy" / "radeon"
DEFAULT_MANIFEST = MANIFEST_DIRECTORY / "appliance-manifest.json"
MANIFEST_ENV = "SIGNALFORGE_APPLIANCE_MANIFEST"
MANIFEST_SHA_ENV = "SIGNALFORGE_APPLIANCE_MANIFEST_SHA256"
GENERATED_ENV_KEYS = frozenset(
    {
        "SIGNALFORGE_ACCEPT_GEMMA_LICENSE",
        MANIFEST_ENV,
        MANIFEST_SHA_ENV,
        "SIGNALFORGE_APPLICATION_ARTIFACT_IDENTITY",
        "SIGNALFORGE_APP_IMAGE",
        "SIGNALFORGE_EXECUTION_BACKEND",
        "SIGNALFORGE_LLAMA_ROCM_IMAGE",
        "SIGNALFORGE_MODEL_ARTIFACT_IDENTITY",
        "SIGNALFORGE_MODEL_SOURCE",
        "SIGNALFORGE_PERSIST_ROOT",
        "SIGNALFORGE_RENDER_GID",
        "SIGNALFORGE_RUNTIME_IDENTITY",
        "SIGNALFORGE_VIDEO_GID",
    }
)
DIGEST_IMAGE = re.compile(r"^[a-z0-9./:_-]+@sha256:[0-9a-f]{64}$")
SOURCE_COMMIT = re.compile(r"^[0-9a-f]{7,40}$")
SHA256 = re.compile(r"^[0-9a-f]{64}$")


class ManifestError(RuntimeError):
    pass


class ManifestSelection(NamedTuple):
    path: Path
    reference: str
    sha256: str
    manifest: dict[str, Any]


def _optional(value: object) -> str | None:
    if value is None:
        return None
    normalized = str(value).strip()
    return normalized or None


def _safe_path(value: str | Path) -> Path:
    candidate = Path(value).expanduser()
    if not candidate.is_absolute():
        candidate = ROOT / candidate
    lexical = Path(os.path.abspath(candidate))
    manifest_root = MANIFEST_DIRECTORY.resolve(strict=True)
    if not lexical.is_relative_to(manifest_root):
        raise ManifestError("appliance manifest must remain under deploy/radeon")
    current = manifest_root
    for component in lexical.relative_to(manifest_root).parts:
        current = current / component
        if current.is_symlink():
            raise ManifestError("appliance manifest path must not contain symbolic links")
    if not lexical.is_file():
        raise ManifestError(f"appliance manifest is missing or not a regular file: {lexical}")
    if not lexical.name.startswith("appliance-manifest") or lexical.suffix != ".json":
        raise ManifestError("appliance manifest must use the appliance-manifest*.json convention")
    return lexical


def manifest_reference(path: Path) -> str:
    return path.relative_to(ROOT).as_posix()


def component_path(reference: object, field: str) -> Path:
    value = _optional(reference)
    if not value:
        raise ManifestError(f"{field} is required")
    candidate = Path(value)
    if candidate.is_absolute() or ".." in candidate.parts:
        raise ManifestError(f"{field} must be a repository-relative path")
    lexical = Path(os.path.abspath(ROOT / candidate))
    if not lexical.is_relative_to(MANIFEST_DIRECTORY.resolve(strict=True)):
        raise ManifestError(f"{field} must remain under deploy/radeon")
    current = ROOT
    for component in candidate.parts:
        current = current / component
        if current.is_symlink():
            raise ManifestError(f"{field} must not contain symbolic links")
    if not lexical.is_file():
        raise ManifestError(f"{field} is missing or not a regular file")
    return lexical


def _require_mapping(value: object, field: str) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise ManifestError(f"appliance manifest field must be an object: {field}")
    return value


def validate_manifest(manifest: dict[str, Any]) -> None:
    if manifest.get("schema_version") != "signalforge/radeon-appliance/v2":
        raise ManifestError("unsupported Radeon appliance manifest schema")
    if manifest.get("platform") != "linux/amd64":
        raise ManifestError("Radeon appliance platform must be linux/amd64")
    if not _optional(manifest.get("appliance_version")):
        raise ManifestError("Radeon appliance version is required")

    application = _require_mapping(manifest.get("application"), "application")
    runtime = _require_mapping(manifest.get("runtime"), "runtime")
    execution = _require_mapping(manifest.get("execution"), "execution")
    utility_images = _require_mapping(manifest.get("utility_images"), "utility_images")

    for field, value in (
        ("application.image", application.get("image")),
        ("runtime.image", runtime.get("image")),
        *(
            (f"utility_images.{name}", image)
            for name, image in utility_images.items()
        ),
    ):
        if not isinstance(value, str) or not DIGEST_IMAGE.fullmatch(value):
            raise ManifestError(f"{field} must be an immutable sha256 image reference")
    if not isinstance(application.get("source_commit"), str) or not SOURCE_COMMIT.fullmatch(
        application["source_commit"]
    ):
        raise ManifestError("application.source_commit must be a 7-40 character Git commit")
    if not _optional(application.get("release")):
        raise ManifestError("application.release is required")
    if execution.get("default_backend") not in {"auto", "compose", "native"}:
        raise ManifestError("execution.default_backend is invalid")
    _require_mapping(execution.get("compose"), "execution.compose")
    _require_mapping(execution.get("native"), "execution.native")
    _require_mapping(manifest.get("gpu"), "gpu")
    _require_mapping(manifest.get("profiles"), "profiles")
    _require_mapping(
        manifest.get("first_run_network_destinations"),
        "first_run_network_destinations",
    )
    component_path(manifest.get("model_manifest"), "model_manifest")
    native = _require_mapping(execution.get("native"), "execution.native")
    component_path(native.get("toolchain_manifest"), "execution.native.toolchain_manifest")
    if not isinstance(manifest.get("minimum_free_bytes"), int) or manifest["minimum_free_bytes"] <= 0:
        raise ManifestError("minimum_free_bytes must be a positive integer")


def read_generated_environment(path: Path) -> dict[str, str]:
    if not path.is_file() or path.is_symlink():
        return {}
    details = path.stat()
    if details.st_uid != os.geteuid() or stat.S_IMODE(details.st_mode) & 0o077:
        raise ManifestError(
            "generated environment must be owner-only and owned by the current user"
        )
    result: dict[str, str] = {}
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        if "\r" in raw_line:
            raise ManifestError("generated environment contains a carriage return")
        if not raw_line or raw_line.startswith("#"):
            continue
        key, separator, value = raw_line.partition("=")
        if not separator:
            raise ManifestError("generated environment contains an invalid line")
        if key not in GENERATED_ENV_KEYS:
            raise ManifestError(f"generated environment contains an unapproved key: {key}")
        if key in result:
            raise ManifestError(f"generated environment repeats {key}")
        result[key] = value
    return result


def _one_authority(values: list[tuple[str, str | None]], label: str) -> str | None:
    populated = [(source, value) for source, value in values if value]
    if not populated:
        return None
    distinct = {value for _, value in populated}
    if len(distinct) != 1:
        sources = ", ".join(source for source, _ in populated)
        raise ManifestError(f"conflicting {label} authorities: {sources}")
    return populated[0][1]


def select_manifest(
    explicit_manifest: str | Path | None = None,
    explicit_sha256: str | None = None,
    *,
    environment: Mapping[str, str] | None = None,
    generated_environment: Mapping[str, str] | None = None,
) -> ManifestSelection:
    environment = environment if environment is not None else os.environ
    generated_environment = generated_environment or {}
    manifest_values = [
        ("command line", _optional(explicit_manifest)),
        ("environment", _optional(environment.get(MANIFEST_ENV))),
        ("generated state", _optional(generated_environment.get(MANIFEST_ENV))),
    ]

    resolved: list[tuple[str, str | None]] = []
    for source, value in manifest_values:
        resolved.append((source, str(_safe_path(value)) if value else None))
    selected_path = _one_authority(resolved, "appliance manifest")
    path = Path(selected_path) if selected_path else _safe_path(DEFAULT_MANIFEST)

    expected_sha = _one_authority(
        [
            ("command line", _optional(explicit_sha256)),
            ("environment", _optional(environment.get(MANIFEST_SHA_ENV))),
            ("generated state", _optional(generated_environment.get(MANIFEST_SHA_ENV))),
        ],
        "appliance manifest SHA-256",
    )
    if expected_sha and not SHA256.fullmatch(expected_sha):
        raise ManifestError("appliance manifest SHA-256 must contain 64 lowercase hex characters")

    payload = path.read_bytes()
    observed_sha = hashlib.sha256(payload).hexdigest()
    if expected_sha and observed_sha != expected_sha:
        raise ManifestError("appliance manifest bytes changed after authority selection")
    try:
        parsed = json.loads(payload)
    except json.JSONDecodeError as error:
        raise ManifestError("appliance manifest is not valid JSON") from error
    if not isinstance(parsed, dict):
        raise ManifestError("appliance manifest root must be an object")
    validate_manifest(parsed)
    return ManifestSelection(
        path=path,
        reference=manifest_reference(path),
        sha256=observed_sha,
        manifest=parsed,
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", type=Path)
    parser.add_argument("--manifest-sha256")
    parser.add_argument("--generated-env", type=Path)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    generated = (
        read_generated_environment(args.generated_env)
        if args.generated_env
        else {}
    )
    try:
        selection = select_manifest(
            args.manifest,
            args.manifest_sha256,
            generated_environment=generated,
        )
    except (ManifestError, OSError) as error:
        print(str(error), file=os.sys.stderr)
        return 2
    result = {
        "path": selection.reference,
        "sha256": selection.sha256,
        "appliance_version": selection.manifest["appliance_version"],
        "application_image": selection.manifest["application"]["image"],
    }
    if args.json:
        print(json.dumps(result, indent=2, sort_keys=True))
    else:
        print(f"{selection.reference}@sha256:{selection.sha256}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
