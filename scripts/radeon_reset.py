#!/usr/bin/env python3
"""Remove bounded Radeon runtime state only after an explicit confirmation."""

from __future__ import annotations

import argparse
import os
import shutil
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def safe_root(path: Path) -> Path:
    expanded = path.expanduser()
    if expanded.is_symlink():
        raise ValueError(f"refusing symbolic-link runtime root: {expanded}")
    resolved = expanded.resolve()
    forbidden = {
        Path("/"),
        Path("/workspace"),
        Path.home().resolve(),
        Path(__file__).resolve().parents[1],
    }
    if resolved in forbidden or len(resolved.parts) < 3:
        raise ValueError(f"refusing unsafe runtime root: {resolved}")
    if resolved.name != "signalforge-runtime" and ".signalforge" not in resolved.parts:
        raise ValueError("runtime root lacks a SignalForge path marker")
    return resolved


def remove_path(path: Path) -> None:
    if path.is_symlink():
        raise ValueError(f"refusing symbolic link: {path}")
    if path.is_dir():
        shutil.rmtree(path)
    elif path.exists():
        path.unlink()


def clean(root: Path) -> None:
    for relative in ("data", "state"):
        remove_path(root / relative)
        (root / relative).mkdir(parents=True, mode=0o700)


def reset(root: Path) -> None:
    remove_path(root)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--persist-root", type=Path)
    parser.add_argument("--mode", choices=("clean", "reset"), required=True)
    parser.add_argument("--confirm", default=os.environ.get("CONFIRM", ""))
    args = parser.parse_args()
    try:
        default_root = (
            Path("/workspace/signalforge-runtime")
            if Path("/workspace").is_dir()
            else ROOT / ".signalforge" / "radeon"
        )
        root = safe_root(args.persist_root or default_root)
        expected = "clean-signalforge-state" if args.mode == "clean" else "delete-signalforge-runtime"
        if args.confirm != expected:
            print(f"Refusing {args.mode}; set CONFIRM={expected} explicitly.", file=sys.stderr)
            return 2
        if args.mode == "clean":
            clean(root)
            print(f"Removed disposable state under {root}; retained model cache.")
        else:
            reset(root)
            print(f"Removed complete SignalForge runtime root: {root}")
        return 0
    except ValueError as error:
        print(str(error), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
