#!/usr/bin/env python3
"""Start and supervise the zero-touch native Radeon runtime."""

from __future__ import annotations

import argparse
import json
import os
import signal
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

import radeon_model_cache
import radeon_native_toolchain
import radeon_runtime_probe


class NativeRuntimeError(RuntimeError):
    pass


def load_json(path: Path) -> dict[str, Any] | None:
    if not path.is_file() or path.is_symlink():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None


def atomic_json(path: Path, payload: dict[str, Any], mode: int = 0o600) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    try:
        temporary.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        os.chmod(temporary, mode)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def ensure_private_directory(path: Path) -> None:
    if path.exists() and (not path.is_dir() or path.is_symlink()):
        raise NativeRuntimeError(f"unsafe native runtime directory: {path}")
    path.mkdir(parents=True, exist_ok=True)
    os.chmod(path, 0o700)


def process_alive(pid: int) -> bool:
    if pid <= 1:
        return False
    try:
        os.kill(pid, 0)
        return True
    except (OSError, ProcessLookupError):
        return False


def process_cmdline(pid: int) -> str | None:
    path = Path(f"/proc/{pid}/cmdline")
    if not path.is_file():
        return None
    try:
        return path.read_bytes().replace(b"\0", b" ").decode("utf-8", errors="replace").strip()
    except OSError:
        return None


def process_matches(record: dict[str, Any]) -> bool:
    pid = int(record.get("pid", 0))
    if not process_alive(pid):
        return False
    expected = [
        str(value)
        for value in (record.get("identity"), record.get("launcher_identity"))
        if value
    ]
    observed = process_cmdline(pid)
    return bool(expected and observed and any(value in observed for value in expected))


def process_record_path(persist_root: Path, name: str) -> Path:
    return persist_root / f"state/native/{name}.process.json"


def ensure_loopback_port_available(port: int, component: str) -> None:
    if port < 1 or port > 65535:
        raise NativeRuntimeError(f"invalid {component} loopback port: {port}")
    probe = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        probe.bind(("127.0.0.1", port))
    except OSError as error:
        raise NativeRuntimeError(
            f"{component} loopback port 127.0.0.1:{port} is already occupied "
            "by an untracked process"
        ) from error
    finally:
        probe.close()


def read_process(persist_root: Path, name: str) -> dict[str, Any] | None:
    return load_json(process_record_path(persist_root, name))


def open_log(path: Path):
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and (not path.is_file() or path.is_symlink()):
        raise NativeRuntimeError(f"unsafe native log path: {path}")
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
    os.chmod(path, 0o600)
    return os.fdopen(descriptor, "a", encoding="utf-8", buffering=1)


def start_process(
    persist_root: Path,
    name: str,
    command: list[str],
    identity: str,
    environment: dict[str, str],
    launcher_identity: str | None = None,
) -> dict[str, Any]:
    record_path = process_record_path(persist_root, name)
    existing = load_json(record_path)
    if existing and process_matches(existing):
        return existing
    if existing and process_alive(int(existing.get("pid", 0))):
        raise NativeRuntimeError(f"refusing to replace unverified live process for {name}")
    record_path.unlink(missing_ok=True)
    log_path = persist_root / f"state/native/logs/{name}.log"
    log = open_log(log_path)
    try:
        process = subprocess.Popen(
            command,
            cwd=ROOT,
            env=environment,
            stdin=subprocess.DEVNULL,
            stdout=log,
            stderr=subprocess.STDOUT,
            start_new_session=True,
            close_fds=True,
            text=True,
        )
    finally:
        log.close()
    record = {
        "schema_version": "signalforge/native-process/v1",
        "name": name,
        "pid": process.pid,
        "identity": identity,
        "launcher_identity": launcher_identity,
        "log_path": str(log_path),
        "started_monotonic_ns": time.monotonic_ns(),
    }
    atomic_json(record_path, record)
    return record


def stop_process(persist_root: Path, name: str, timeout_seconds: float = 20) -> bool:
    path = process_record_path(persist_root, name)
    record = load_json(path)
    if not record:
        path.unlink(missing_ok=True)
        return False
    pid = int(record.get("pid", 0))
    if not process_alive(pid):
        path.unlink(missing_ok=True)
        return False
    if not process_matches(record):
        raise NativeRuntimeError(f"refusing to signal process with mismatched identity: {name}")
    try:
        os.killpg(pid, signal.SIGTERM)
    except ProcessLookupError:
        path.unlink(missing_ok=True)
        return False
    deadline = time.monotonic() + timeout_seconds
    while process_alive(pid) and time.monotonic() < deadline:
        time.sleep(0.2)
    if process_alive(pid):
        try:
            os.killpg(pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    path.unlink(missing_ok=True)
    return True


def base_environment(persist_root: Path) -> dict[str, str]:
    allowed = (
        "PATH",
        "HOME",
        "LANG",
        "LC_ALL",
        "LD_LIBRARY_PATH",
        "ROCM_PATH",
        "HIP_PATH",
        "HSA_OVERRIDE_GFX_VERSION",
    )
    environment = {key: os.environ[key] for key in allowed if key in os.environ}
    environment.setdefault("PATH", "/usr/local/bin:/usr/bin:/bin")
    environment.setdefault("HOME", str(persist_root / "state/native/home"))
    environment["LC_ALL"] = "C"
    return environment


def llama_environment(persist_root: Path, llama_binary: Path) -> dict[str, str]:
    environment = base_environment(persist_root)
    environment.update(
        {
            "LLAMA_SERVER_BIN": str(llama_binary),
            "SIGNALFORGE_MODEL_HOST": "127.0.0.1",
            "SIGNALFORGE_MODEL_PORT": os.environ.get("SIGNALFORGE_MODEL_PORT", "8000"),
            "SIGNALFORGE_VERIFY_MODEL_HASH": "1",
            "SIGNALFORGE_CONTEXT_SIZE": os.environ.get("SIGNALFORGE_CONTEXT_SIZE", "32768"),
            "SIGNALFORGE_PARALLEL_SLOTS": os.environ.get("SIGNALFORGE_PARALLEL_SLOTS", "4"),
            "SIGNALFORGE_BATCH_SIZE": os.environ.get("SIGNALFORGE_BATCH_SIZE", "2048"),
            "SIGNALFORGE_UBATCH_SIZE": os.environ.get("SIGNALFORGE_UBATCH_SIZE", "512"),
            "SIGNALFORGE_FLASH_ATTN": os.environ.get("SIGNALFORGE_FLASH_ATTN", "auto"),
            "SIGNALFORGE_CACHE_TYPE_K": os.environ.get("SIGNALFORGE_CACHE_TYPE_K", "f16"),
            "SIGNALFORGE_CACHE_TYPE_V": os.environ.get("SIGNALFORGE_CACHE_TYPE_V", "f16"),
            "SIGNALFORGE_CONT_BATCHING": os.environ.get("SIGNALFORGE_CONT_BATCHING", "1"),
            "SIGNALFORGE_KV_UNIFIED": os.environ.get("SIGNALFORGE_KV_UNIFIED", "1"),
            "HSA_OVERRIDE_GFX_VERSION": os.environ.get("HSA_OVERRIDE_GFX_VERSION", "11.0.0"),
        }
    )
    return environment


def app_environment(persist_root: Path, profile: str, secrets_dir: Path) -> dict[str, str]:
    environment = base_environment(persist_root)
    environment.update(
        {
            "SIGNALFORGE_SPECIALIST_API_ENABLED": "true" if profile == "championship" else "false",
            "SIGNALFORGE_OTEL_ENABLED": os.environ.get("SIGNALFORGE_OTEL_ENABLED", "false"),
            "SIGNALFORGE_OTEL_INSECURE": os.environ.get("SIGNALFORGE_OTEL_INSECURE", "true"),
            "SIGNALFORGE_OTEL_ALLOW_PRIVATE_NETWORK": "true",
            "OTEL_EXPORTER_OTLP_ENDPOINT": os.environ.get(
                "OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318"
            ),
        }
    )
    if profile == "championship":
        environment.update(
            {
                "SIGNALFORGE_SPECIALIST_API_PROVIDER": os.environ.get(
                    "SIGNALFORGE_SPECIALIST_API_PROVIDER", "radeon-vllm"
                ),
                "SIGNALFORGE_SPECIALIST_API_BASE_URL": os.environ.get(
                    "SIGNALFORGE_SPECIALIST_API_BASE_URL",
                    "https://radeon.anruicloud.com/api/v1",
                ),
                "SIGNALFORGE_SPECIALIST_API_KEY_FILE": str(
                    secrets_dir / "radeon-model-api-key"
                ),
                "SIGNALFORGE_SPECIALIST_TEXT_MODEL": os.environ.get(
                    "SIGNALFORGE_SPECIALIST_TEXT_MODEL", "DeepSeek-V4-Flash"
                ),
                "SIGNALFORGE_SPECIALIST_VISION_MODEL": os.environ.get(
                    "SIGNALFORGE_SPECIALIST_VISION_MODEL", "Qwen3.6-35B-A3B"
                ),
                "SIGNALFORGE_SPECIALIST_API_TIMEOUT": os.environ.get(
                    "SIGNALFORGE_SPECIALIST_API_TIMEOUT", "90s"
                ),
            }
        )
    return environment


def app_command(binary: Path, persist_root: Path, profile: str) -> list[str]:
    data = persist_root / "data"
    common = [
        str(binary),
        "--listen",
        f"127.0.0.1:{os.environ.get('SIGNALFORGE_APP_PORT', '8080')}",
        "--static-dir",
        str(ROOT / "web/dist"),
        "--catalog",
        str(ROOT / "fixtures/productscope/technology20-catalog.json"),
        "--case-db",
        str(data / "cases.db"),
        "--audit-dir",
        str(data / "audit"),
        "--trace-dir",
        str(data / "traces"),
        "--event-log",
        str(data / "logs/events.jsonl"),
    ]
    if profile == "fixture":
        return common + [
            "--mode",
            "fixture",
            "--fixture",
            str(ROOT / "fixtures/workspace/golden-case.json"),
            "--event-delay",
            "0",
        ]
    return common + [
        "--mode",
        "live",
        "--base-url",
        f"http://127.0.0.1:{os.environ.get('SIGNALFORGE_MODEL_PORT', '8000')}/v1",
        "--model",
        "signalforge-gemma4-26b-q4",
        "--snapshot",
        str(ROOT / "fixtures/golden/financial-snapshot.json"),
        "--retrieval",
        str(ROOT / "fixtures/retrieval/golden-eval.json"),
        "--price-inputs",
        str(ROOT / "fixtures/golden/market-price-inputs.json"),
    ]


def fetch_json(url: str, timeout: float = 3) -> dict[str, Any] | None:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as response:
            if response.status != 200:
                return None
            return json.load(response)
    except (OSError, urllib.error.URLError, json.JSONDecodeError):
        return None


def wait_app(
    persist_root: Path,
    timeout_seconds: float,
    expected_build_version: str,
    expected_mode: str,
    poll_seconds: float = 1,
) -> dict[str, Any]:
    deadline = time.monotonic() + timeout_seconds
    port = os.environ.get("SIGNALFORGE_APP_PORT", "8080")
    while time.monotonic() < deadline:
        record = read_process(persist_root, "app")
        if not record or not process_matches(record):
            raise NativeRuntimeError("SignalForge application exited before readiness")
        health = fetch_json(f"http://127.0.0.1:{port}/health/ready")
        if health and health.get("status") == "ready":
            if (
                health.get("build_version") != expected_build_version
                or health.get("mode") != expected_mode
            ):
                raise NativeRuntimeError(
                    "SignalForge readiness identity does not match the launched binary"
                )
            return health
        time.sleep(poll_seconds)
    raise NativeRuntimeError("SignalForge application did not become ready before timeout")


def hydrate_model(
    persist_root: Path,
    model_manifest: Path,
    secrets_dir: Path,
) -> dict[str, Any]:
    existing = os.environ.get("SIGNALFORGE_EXISTING_MODEL_FILE", "").strip()
    return radeon_model_cache.hydrate(
        manifest_path=model_manifest,
        cache_dir=persist_root / "models",
        source=os.environ.get("SIGNALFORGE_MODEL_SOURCE", "huggingface"),
        token_file=secrets_dir / "hf-token",
        existing_file=Path(existing) if existing else None,
        license_accepted=os.environ.get("SIGNALFORGE_ACCEPT_GEMMA_LICENSE") == "yes",
        retries=int(os.environ.get("SIGNALFORGE_MODEL_DOWNLOAD_RETRIES", "5")),
        timeout_seconds=float(
            os.environ.get("SIGNALFORGE_MODEL_DOWNLOAD_TIMEOUT_SECONDS", "60")
        ),
        state_path=persist_root / "state/model-init.json",
    )


def native_status(persist_root: Path, profile: str) -> dict[str, Any]:
    app_record = read_process(persist_root, "app")
    llama_record = read_process(persist_root, "llama")
    app_alive = bool(app_record and process_matches(app_record))
    llama_required = profile in {"radeon-local", "championship"}
    llama_alive = bool(llama_record and process_matches(llama_record))
    app_port = os.environ.get("SIGNALFORGE_APP_PORT", "8080")
    model_port = os.environ.get("SIGNALFORGE_MODEL_PORT", "8000")
    app_health = fetch_json(f"http://127.0.0.1:{app_port}/health/ready") if app_alive else None
    model_health = fetch_json(f"http://127.0.0.1:{model_port}/health") if llama_alive else None
    ready = (
        app_alive
        and bool(app_health and app_health.get("status") == "ready")
        and (not llama_required or (llama_alive and model_health is not None))
    )
    return {
        "schema_version": "signalforge/radeon-native-status/v1",
        "backend": "native",
        "profile": profile,
        "status": "ready" if ready else "preparing",
        "phase": (
            "ready"
            if ready
            else "model-runtime"
            if llama_required and not llama_alive
            else "application"
        ),
        "processes": {
            "app": {"alive": app_alive, "record": app_record},
            "llama": {"alive": llama_alive, "record": llama_record},
        },
        "application_health": app_health,
        "model_health": model_health,
        "toolchain": load_json(persist_root / "state/native/toolchain-status.json"),
        "model_state": load_json(persist_root / "state/model-init.json"),
        "runtime_state": load_json(persist_root / "state/runtime-ready.json"),
        "workspace_url": f"http://127.0.0.1:{app_port}",
    }


def start(
    persist_root: Path,
    profile: str,
    secrets_dir: Path,
    *,
    allow_dirty: bool,
) -> dict[str, Any]:
    for directory in (
        persist_root,
        persist_root / "data",
        persist_root / "data/logs",
        persist_root / "data/audit",
        persist_root / "data/traces",
        persist_root / "models",
        persist_root / "state/native/logs",
        persist_root / "state/native/home",
    ):
        ensure_private_directory(directory)
    current = native_status(persist_root, profile)
    if current["status"] == "ready":
        return current
    for name in ("app", "llama"):
        if read_process(persist_root, name):
            stop_process(persist_root, name)
    model_manifest_path = ROOT / "deploy/radeon/model-manifest.json"
    toolchain_manifest_path = ROOT / "deploy/radeon/native-toolchain-manifest.json"
    model_manifest = json.loads(model_manifest_path.read_text(encoding="utf-8"))
    toolchain_manifest = json.loads(toolchain_manifest_path.read_text(encoding="utf-8"))
    atomic_json(
        persist_root / "state/native/startup.json",
        {"status": "preparing", "phase": "model-hydration", "profile": profile},
    )
    if profile in {"radeon-local", "championship"}:
        hydrate_model(persist_root, model_manifest_path, secrets_dir)
    atomic_json(
        persist_root / "state/native/startup.json",
        {"status": "preparing", "phase": "native-build", "profile": profile},
    )
    toolchain = radeon_native_toolchain.prepare(
        persist_root,
        profile,
        toolchain_manifest,
        allow_dirty=allow_dirty,
        retries=int(os.environ.get("SIGNALFORGE_MODEL_DOWNLOAD_RETRIES", "5")),
        download_timeout_seconds=float(
            os.environ.get("SIGNALFORGE_MODEL_DOWNLOAD_TIMEOUT_SECONDS", "60")
        ),
        build_timeout_seconds=float(
            os.environ.get("SIGNALFORGE_NATIVE_BUILD_TIMEOUT_SECONDS", "1800")
        ),
    )
    try:
        if profile in {"radeon-local", "championship"}:
            llama_binary = Path(toolchain["llama_cpp"]["binary"])
            model_file = persist_root / "models" / model_manifest["cache"]["model_relative_path"]
            atomic_json(
                persist_root / "state/native/startup.json",
                {"status": "preparing", "phase": "model-load", "profile": profile},
            )
            ensure_loopback_port_available(
                int(os.environ.get("SIGNALFORGE_MODEL_PORT", "8000")),
                "local model",
            )
            start_process(
                persist_root,
                "llama",
                [str(ROOT / "scripts/serve_llama_rocm.sh"), str(model_file)],
                str(llama_binary),
                llama_environment(persist_root, llama_binary),
                str(ROOT / "scripts/serve_llama_rocm.sh"),
            )
            runtime_receipt = radeon_runtime_probe.wait_ready(
                f"http://127.0.0.1:{os.environ.get('SIGNALFORGE_MODEL_PORT', '8000')}",
                model_manifest["served_model_id"],
                timeout_seconds=float(
                    os.environ.get("SIGNALFORGE_NATIVE_MODEL_READY_TIMEOUT_SECONDS", "900")
                ),
                request_timeout_seconds=3,
                poll_seconds=2,
            )
            radeon_runtime_probe.atomic_json(
                persist_root / "state/runtime-ready.json", runtime_receipt
            )
        atomic_json(
            persist_root / "state/native/startup.json",
            {"status": "preparing", "phase": "application-startup", "profile": profile},
        )
        app_binary = Path(toolchain["application_binary"])
        ensure_loopback_port_available(
            int(os.environ.get("SIGNALFORGE_APP_PORT", "8080")),
            "application",
        )
        start_process(
            persist_root,
            "app",
            app_command(app_binary, persist_root, profile),
            str(app_binary),
            app_environment(persist_root, profile, secrets_dir),
        )
        wait_app(
            persist_root,
            float(os.environ.get("SIGNALFORGE_NATIVE_APP_READY_TIMEOUT_SECONDS", "120")),
            str(toolchain["application"]["source_commit"]),
            "fixture" if profile == "fixture" else "live",
        )
    except Exception:
        for name in ("app", "llama"):
            try:
                stop_process(persist_root, name)
            except NativeRuntimeError:
                pass
        raise
    result = native_status(persist_root, profile)
    atomic_json(persist_root / "state/native/startup.json", result)
    return result


def stop(persist_root: Path) -> dict[str, Any]:
    stopped: list[str] = []
    for name in ("app", "llama"):
        if stop_process(persist_root, name):
            stopped.append(name)
    result = {
        "schema_version": "signalforge/radeon-native-stop/v1",
        "status": "stopped",
        "processes": stopped,
    }
    atomic_json(persist_root / "state/native/last-stop.json", result)
    return result


def print_logs(persist_root: Path, tail: int) -> None:
    for name in ("llama", "app", "native-build"):
        path = persist_root / f"state/native/logs/{name}.log"
        if not path.is_file() or path.is_symlink():
            continue
        print(f"==> {name}.log <==")
        lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
        for line in lines[-tail:]:
            print(line)


def main() -> int:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)
    for command in ("up", "status", "down", "logs"):
        child = subparsers.add_parser(command)
        child.add_argument("--persist-root", type=Path, required=True)
        if command in {"up", "status"}:
            child.add_argument(
                "--profile",
                choices=("fixture", "radeon-local", "championship"),
                required=True,
            )
        if command == "up":
            child.add_argument("--secrets-dir", type=Path, default=ROOT / ".secrets")
            child.add_argument("--allow-dirty", action="store_true")
        if command == "logs":
            child.add_argument("--tail", type=int, default=200)
    args = parser.parse_args()
    persist_root = args.persist_root.expanduser().resolve()
    try:
        if args.command == "up":
            result = start(
                persist_root,
                args.profile,
                args.secrets_dir,
                allow_dirty=args.allow_dirty,
            )
        elif args.command == "status":
            result = native_status(persist_root, args.profile)
        elif args.command == "down":
            result = stop(persist_root)
        else:
            print_logs(persist_root, max(1, min(args.tail, 5000)))
            return 0
    except (
        NativeRuntimeError,
        radeon_model_cache.ModelCacheError,
        radeon_native_toolchain.NativeToolchainError,
        OSError,
        subprocess.SubprocessError,
        TimeoutError,
    ) as error:
        print(f"Native Radeon runtime failed: {error}", file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result.get("status") in {"ready", "stopped"} else 1


if __name__ == "__main__":
    raise SystemExit(main())
