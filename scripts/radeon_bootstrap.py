#!/usr/bin/env python3
"""Prepare an existing Radeon workspace without installing host tooling."""

from __future__ import annotations

import argparse
import getpass
import os
import stat
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def atomic_secret(path: Path, value: str, mode: int = 0o600) -> None:
    if "\n" in value or "\r" in value:
        raise ValueError("secret must be one line")
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    try:
        with temporary.open("w", encoding="utf-8") as handle:
            handle.write(value)
            handle.flush()
            os.fsync(handle.fileno())
        os.chmod(temporary, mode)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def ensure_private_directory(path: Path) -> None:
    if path.exists() and (not path.is_dir() or path.is_symlink()):
        raise ValueError(f"unsafe directory: {path}")
    path.mkdir(parents=True, exist_ok=True)
    os.chmod(path, 0o700)


def ensure_placeholder(path: Path, mode: int) -> None:
    if path.exists() and (not path.is_file() or path.is_symlink()):
        raise ValueError(f"unsafe secret path: {path}")
    if not path.exists():
        atomic_secret(path, "", mode)
    os.chmod(path, mode)


def has_nonempty_file(path: Path) -> bool:
    return path.is_file() and not path.is_symlink() and path.stat().st_size > 0


def prompt_secret(label: str) -> str:
    first = getpass.getpass(f"{label}: ").strip()
    if not first:
        raise ValueError(f"{label} cannot be empty")
    return first


def resolve_persist_root(explicit: Path | None) -> Path:
    if explicit is not None:
        return explicit.expanduser()
    configured = os.environ.get("SIGNALFORGE_PERSIST_ROOT", "").strip()
    if configured:
        return Path(configured).expanduser()
    if Path("/workspace").is_dir():
        return Path("/workspace/signalforge-runtime")
    return ROOT / ".signalforge" / "radeon"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--profile", choices=("fixture", "radeon-local", "championship"), default="radeon-local")
    parser.add_argument(
        "--backend",
        choices=("auto", "compose", "native"),
        default=os.environ.get("SIGNALFORGE_EXECUTION_BACKEND", "auto"),
    )
    parser.add_argument("--persist-root", type=Path)
    parser.add_argument("--accept-gemma-license", action="store_true")
    parser.add_argument("--noninteractive", action="store_true")
    parser.add_argument("--skip-network-check", action="store_true")
    args = parser.parse_args()

    persist_root = resolve_persist_root(args.persist_root)
    secrets = ROOT / ".secrets"
    ensure_private_directory(secrets)
    for directory in (
        persist_root,
        persist_root / "data",
        persist_root / "models",
        persist_root / "state",
        persist_root / "state" / "native",
        persist_root / "state" / "native" / "logs",
        persist_root / "toolchains",
        persist_root / "bin",
        persist_root / "cache",
    ):
        ensure_private_directory(directory)
    ensure_placeholder(secrets / "hf-token", 0o600)
    ensure_placeholder(secrets / "radeon-model-api-key", 0o644)
    ensure_placeholder(secrets / "grafana-admin-password", 0o644)

    needs_model = args.profile in {"radeon-local", "championship"}
    marker = persist_root / "models" / ".signalforge-model-ready.json"
    accepted = args.accept_gemma_license or os.environ.get("SIGNALFORGE_ACCEPT_GEMMA_LICENSE") == "yes"
    if needs_model and not marker.is_file():
        if not accepted:
            if args.noninteractive or not sys.stdin.isatty():
                print(
                    "Bootstrap requires explicit Gemma license acceptance. "
                    "Re-run with --accept-gemma-license after reviewing the upstream terms.",
                    file=sys.stderr,
                )
                return 2
            response = input("Have you reviewed and accepted the upstream Gemma license? [yes/NO] ").strip()
            accepted = response == "yes"
            if not accepted:
                print("Gemma license was not accepted; bootstrap stopped.", file=sys.stderr)
                return 2
        token_path = secrets / "hf-token"
        if not has_nonempty_file(token_path):
            if args.noninteractive or not sys.stdin.isatty():
                print(
                    "Bootstrap requires a non-empty file-mounted Hugging Face read token at "
                    f"{token_path}.",
                    file=sys.stderr,
                )
                return 2
            atomic_secret(token_path, prompt_secret("Hugging Face read token"), 0o600)

    if args.profile == "championship" and not has_nonempty_file(secrets / "radeon-model-api-key"):
        if args.noninteractive or not sys.stdin.isatty():
            print(
                "Championship bootstrap requires a non-empty Radeon API key file at "
                f"{secrets / 'radeon-model-api-key'}.",
                file=sys.stderr,
            )
            return 2
        atomic_secret(
            secrets / "radeon-model-api-key",
            prompt_secret("Radeon API key"),
            0o644,
        )

    report = persist_root / "state" / "preflight.json"
    environment = persist_root / "state" / "generated.env"
    command = [
        sys.executable,
        str(ROOT / "scripts" / "radeon_preflight.py"),
        "--profile",
        args.profile,
        "--persist-root",
        str(persist_root),
        "--secrets-dir",
        str(secrets),
        "--report",
        str(report),
        "--env-output",
        str(environment),
        "--model-source",
        os.environ.get("SIGNALFORGE_MODEL_SOURCE", "huggingface"),
        "--backend",
        args.backend,
    ]
    if accepted:
        command.append("--license-accepted")
    if not args.skip_network_check and not marker.is_file():
        command.append("--check-network")
    result = subprocess.run(command, check=False)
    if result.returncode:
        return result.returncode
    print("Bootstrap prepared only existing host capabilities; no duplicate tooling was installed.")
    print(f"Persistent runtime root: {persist_root}")
    print(f"Generated non-secret environment: {environment}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
