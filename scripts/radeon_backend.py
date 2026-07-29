#!/usr/bin/env python3
"""Resolve the Radeon execution backend without installing host tooling."""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
VALID_BACKENDS = {"auto", "compose", "native"}


class BackendError(RuntimeError):
    pass


def run_version(command: list[str], timeout: float = 8) -> tuple[bool, str]:
    try:
        result = subprocess.run(
            command,
            check=False,
            capture_output=True,
            text=True,
            timeout=timeout,
            env={**os.environ, "LC_ALL": "C"},
        )
    except (OSError, subprocess.TimeoutExpired):
        return False, ""
    output = (result.stdout + "\n" + result.stderr).strip()
    return result.returncode == 0, output[:2048]


def backend_facts(manifest: dict[str, Any]) -> dict[str, Any]:
    docker = shutil.which("docker")
    engine_ready, engine_version = (
        run_version([docker, "version", "--format", "{{.Server.Version}}"])
        if docker
        else (False, "")
    )
    compose_ready, compose_version = (
        run_version([docker, "compose", "version", "--short"])
        if docker
        else (False, "")
    )
    required = manifest["execution"]["native"]["required_host_commands"]
    native_commands = {name: shutil.which(name) for name in required}
    version_commands = {
        "python3": ["python3", "--version"],
        "node": ["node", "--version"],
        "npm": ["npm", "--version"],
        "cmake": ["cmake", "--version"],
        "ninja": ["ninja", "--version"],
        "hipcc": ["hipcc", "--version"],
        "git": ["git", "--version"],
        "curl": ["curl", "--version"],
    }
    versions: dict[str, str | None] = {}
    for name, command in version_commands.items():
        if native_commands.get(name):
            ready, output = run_version(command)
            versions[name] = output.splitlines()[0] if ready and output else None
        else:
            versions[name] = None
    return {
        "compose": {
            "docker_path": docker,
            "engine_ready": engine_ready,
            "engine_version": engine_version.splitlines()[0] if engine_version else None,
            "compose_ready": compose_ready,
            "compose_version": compose_version.splitlines()[0] if compose_version else None,
        },
        "native": {
            "commands": native_commands,
            "versions": versions,
            "ready": all(native_commands.values()),
            "missing": sorted(name for name, path in native_commands.items() if not path),
        },
    }


def resolve_backend(requested: str, facts: dict[str, Any]) -> str:
    if requested not in VALID_BACKENDS:
        raise BackendError(f"unsupported execution backend: {requested}")
    compose_ready = bool(
        facts["compose"]["engine_ready"] and facts["compose"]["compose_ready"]
    )
    native_ready = bool(facts["native"]["ready"])
    if requested == "auto":
        if compose_ready:
            return "compose"
        if native_ready:
            return "native"
        raise BackendError(
            "neither Docker Compose nor the declared native Radeon toolchain is available"
        )
    if requested == "compose" and not compose_ready:
        raise BackendError("compose backend requested but Docker Engine/Compose is unavailable")
    if requested == "native" and not native_ready:
        missing = ", ".join(facts["native"]["missing"]) or "unknown"
        raise BackendError(f"native backend requested but host commands are missing: {missing}")
    return requested


def read_generated_backend(path: Path) -> str | None:
    if not path.is_file() or path.is_symlink():
        return None
    for line in path.read_text(encoding="utf-8").splitlines():
        key, separator, value = line.partition("=")
        if separator and key == "SIGNALFORGE_EXECUTION_BACKEND":
            candidate = value.strip()
            if candidate in {"compose", "native"}:
                return candidate
    return None


def default_persist_root() -> Path:
    configured = os.environ.get("SIGNALFORGE_PERSIST_ROOT", "").strip()
    if configured:
        return Path(configured).expanduser()
    if Path("/workspace").is_dir():
        return Path("/workspace/signalforge-runtime")
    return ROOT / ".signalforge" / "radeon"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--backend", choices=sorted(VALID_BACKENDS))
    parser.add_argument("--persist-root", type=Path, default=default_persist_root())
    parser.add_argument(
        "--manifest",
        type=Path,
        default=ROOT / "deploy/radeon/appliance-manifest.json",
    )
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
    requested = (
        args.backend
        or os.environ.get("SIGNALFORGE_EXECUTION_BACKEND")
        or read_generated_backend(args.persist_root / "state/generated.env")
        or manifest["execution"]["default_backend"]
    )
    facts = backend_facts(manifest)
    try:
        selected = resolve_backend(requested, facts)
    except BackendError as error:
        print(str(error), file=sys.stderr)
        return 2
    if args.json:
        print(
            json.dumps(
                {"requested": requested, "selected": selected, "facts": facts},
                indent=2,
                sort_keys=True,
            )
        )
    else:
        print(selected)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
