#!/usr/bin/env python3
"""Fail-closed, secret-safe Radeon Cloud appliance preflight."""

from __future__ import annotations

import argparse
import json
import os
import platform
import re
import shutil
import socket
import ssl
import stat
import subprocess
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

import radeon_backend


DEFAULT_MANIFEST = ROOT / "deploy" / "radeon" / "appliance-manifest.json"
DEFAULT_MODEL_MANIFEST = ROOT / "deploy" / "radeon" / "model-manifest.json"
MAC_PATH_PATTERN = re.compile(r"^/(?:Users|Volumes)/")
GFX_PATTERN = re.compile(r"\bgfx[0-9a-f]+\b", re.IGNORECASE)


def load_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def atomic_write(path: Path, payload: str, mode: int) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    try:
        temporary.write_text(payload, encoding="utf-8")
        os.chmod(temporary, mode)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def run_safe(command: list[str], timeout: float = 5) -> dict[str, Any]:
    try:
        result = subprocess.run(
            command,
            check=False,
            capture_output=True,
            text=True,
            timeout=timeout,
            env={**os.environ, "LC_ALL": "C"},
        )
    except (OSError, subprocess.TimeoutExpired) as error:
        return {"available": False, "returncode": None, "output": "", "error": type(error).__name__}
    output = (result.stdout + "\n" + result.stderr).strip()
    return {
        "available": True,
        "returncode": result.returncode,
        "output": output[:8192],
        "error": None,
    }


def normalize_architecture(machine: str) -> str:
    normalized = machine.strip().lower()
    if normalized in {"x86_64", "x64"}:
        return "amd64"
    if normalized in {"aarch64", "arm64"}:
        return "arm64"
    return normalized or "unknown"


def version_at_least(observed: str | None, required: tuple[int, ...]) -> bool:
    if not observed:
        return False
    match = re.search(r"([0-9]+(?:\.[0-9]+)+)", observed)
    if not match:
        return False
    parts = tuple(int(item) for item in match.group(1).split("."))
    width = max(len(parts), len(required))
    return parts + (0,) * (width - len(parts)) >= required + (0,) * (width - len(required))


def memory_total_bytes() -> int | None:
    path = Path("/proc/meminfo")
    if path.is_file():
        for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
            if line.startswith("MemTotal:"):
                return int(line.split()[1]) * 1024
    return None


def rocm_version() -> str | None:
    candidates = (
        Path("/opt/rocm/.info/version"),
        Path("/opt/rocm/.info/version-dev"),
        Path("/opt/rocm/share/rocmcmakebuildtools/version"),
    )
    for candidate in candidates:
        if candidate.is_file():
            value = candidate.read_text(encoding="utf-8", errors="replace").strip()
            if value:
                return value.splitlines()[0]
    result = run_safe(["rocminfo"], timeout=8)
    if result["available"] and result["returncode"] == 0:
        match = re.search(r"ROCm(?: Runtime)? Version\s*:?\s*([0-9][^\s]*)", result["output"])
        if match:
            return match.group(1)
    return None


def gpu_facts() -> dict[str, Any]:
    result = run_safe(["rocminfo"], timeout=10)
    output = result["output"] if result["available"] and result["returncode"] == 0 else ""
    architectures = sorted({match.lower() for match in GFX_PATTERN.findall(output)})
    marketing_names = sorted(
        {
            line.split(":", 1)[1].strip()
            for line in output.splitlines()
            if "Marketing Name:" in line and line.split(":", 1)[1].strip()
        }
    )
    smi = run_safe(["rocm-smi", "--showmeminfo", "vram", "--json"], timeout=10)
    vram_bytes: int | None = None
    if smi["available"] and smi["returncode"] == 0:
        numbers = [
            int(value)
            for value in re.findall(r'"VRAM Total Memory \(B\)"\s*:\s*"?([0-9]+)', smi["output"])
        ]
        if numbers:
            vram_bytes = max(numbers)
    return {
        "architectures": architectures,
        "marketing_names": marketing_names,
        "vram_bytes": vram_bytes,
        "rocminfo_available": bool(output),
        "rocm_smi_available": bool(smi["available"] and smi["returncode"] == 0),
    }


def device_facts() -> dict[str, Any]:
    render_nodes = sorted(Path("/dev/dri").glob("renderD*")) if Path("/dev/dri").is_dir() else []
    card_nodes = sorted(Path("/dev/dri").glob("card*")) if Path("/dev/dri").is_dir() else []
    render = render_nodes[0] if render_nodes else (card_nodes[0] if card_nodes else None)
    kfd = Path("/dev/kfd")

    def describe(path: Path | None) -> dict[str, Any]:
        if path is None or not path.exists():
            return {"path": str(path) if path else None, "exists": False, "gid": None, "mode": None}
        details = path.stat()
        return {
            "path": str(path),
            "exists": True,
            "gid": details.st_gid,
            "mode": stat.filemode(details.st_mode),
        }

    return {"kfd": describe(kfd), "render": describe(render), "dri_exists": Path("/dev/dri").is_dir()}


def docker_facts() -> dict[str, Any]:
    binary = shutil.which("docker")
    if binary is None:
        return {
            "installed": False,
            "engine_ready": False,
            "compose_ready": False,
            "engine_version": None,
            "compose_version": None,
        }
    engine = run_safe([binary, "version", "--format", "{{.Server.Version}}"], timeout=8)
    compose = run_safe([binary, "compose", "version", "--short"], timeout=8)
    return {
        "installed": True,
        "engine_ready": engine["returncode"] == 0,
        "compose_ready": compose["returncode"] == 0,
        "engine_version": engine["output"].splitlines()[0] if engine["returncode"] == 0 else None,
        "compose_version": compose["output"].splitlines()[0] if compose["returncode"] == 0 else None,
    }


def secret_metadata(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {"path": str(path), "exists": False, "regular": False, "size_bytes": 0, "mode": None}
    details = path.lstat()
    return {
        "path": str(path),
        "exists": True,
        "regular": stat.S_ISREG(details.st_mode) and not path.is_symlink(),
        "size_bytes": details.st_size,
        "mode": stat.S_IMODE(details.st_mode),
    }


def model_ready(cache_dir: Path, model_manifest: dict[str, Any]) -> bool:
    cache = model_manifest["cache"]
    marker = cache_dir / cache["ready_marker_relative_path"]
    model = cache_dir / cache["model_relative_path"]
    if not marker.is_file() or marker.is_symlink() or not model.is_file() or model.is_symlink():
        return False
    try:
        payload = json.loads(marker.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return False
    return (
        payload.get("verified") is True
        and payload.get("manifest_version") == model_manifest["manifest_version"]
        and payload.get("sha256") == model_manifest["sha256"]
        and model.stat().st_size == model_manifest["expected_size_bytes"]
    )


def network_facts(destinations: list[str], timeout: float) -> list[dict[str, Any]]:
    context = ssl.create_default_context()
    results: list[dict[str, Any]] = []
    for destination in destinations:
        host, raw_port = destination.rsplit(":", 1)
        port = int(raw_port)
        reachable = False
        error: str | None = None
        try:
            with socket.create_connection((host, port), timeout=timeout) as connection:
                with context.wrap_socket(connection, server_hostname=host):
                    reachable = True
        except (OSError, ssl.SSLError) as failure:
            error = type(failure).__name__
        results.append({"destination": destination, "reachable": reachable, "error": error})
    return results


def required_network_destinations(
    manifest: dict[str, Any],
    *,
    profile: str,
    backend: str | None,
    model_cache_ready: bool,
) -> list[str]:
    configured = manifest["first_run_network_destinations"]
    if isinstance(configured, list):
        return configured
    selected = list(configured.get(backend or "", []))
    if profile in {"radeon-local", "championship"} and not model_cache_ready:
        selected.extend(configured.get("model", []))
    return sorted(set(selected))


def collect_facts(
    manifest: dict[str, Any],
    model_manifest: dict[str, Any],
    persist_root: Path,
    secrets_dir: Path,
    check_network: bool,
    network_timeout: float,
    profile: str,
    requested_backend: str,
) -> dict[str, Any]:
    if persist_root.exists() and (not persist_root.is_dir() or persist_root.is_symlink()):
        disk_path = persist_root.parent
    else:
        persist_root.mkdir(parents=True, exist_ok=True)
        disk_path = persist_root
    disk = shutil.disk_usage(disk_path)
    parent_mode = stat.S_IMODE(secrets_dir.stat().st_mode) if secrets_dir.is_dir() else None
    execution_backends = radeon_backend.backend_facts(manifest)
    try:
        selected_backend = radeon_backend.resolve_backend(requested_backend, execution_backends)
    except radeon_backend.BackendError:
        selected_backend = None
    cache_ready = model_ready(persist_root / "models", model_manifest)
    destinations = required_network_destinations(
        manifest,
        profile=profile,
        backend=selected_backend,
        model_cache_ready=cache_ready,
    )
    return {
        "platform": {
            "system": platform.system().lower(),
            "architecture": normalize_architecture(platform.machine()),
        },
        "devices": device_facts(),
        "gpu": gpu_facts(),
        "rocm_version": rocm_version(),
        "host": {
            "ram_bytes": memory_total_bytes(),
            "cpu_count": os.cpu_count(),
            "disk_total_bytes": disk.total,
            "disk_free_bytes": disk.free,
        },
        "docker": docker_facts(),
        "execution_backends": execution_backends,
        "persistent_root": {
            "path": str(persist_root.resolve()),
            "exists": persist_root.is_dir(),
            "is_symlink": persist_root.is_symlink(),
        },
        "model_cache_ready": cache_ready,
        "secrets": {
            "directory": {
                "path": str(secrets_dir),
                "exists": secrets_dir.is_dir(),
                "mode": parent_mode,
            },
            "hf_token": secret_metadata(secrets_dir / "hf-token"),
            "radeon_api_key": secret_metadata(secrets_dir / "radeon-model-api-key"),
            "grafana_password": secret_metadata(secrets_dir / "grafana-admin-password"),
        },
        "network": (
            network_facts(destinations, network_timeout)
            if check_network
            else []
        ),
    }


def check(check_id: str, passed: bool, detail: str, *, warning: bool = False) -> dict[str, str]:
    return {
        "id": check_id,
        "status": "warning" if warning else ("passed" if passed else "failed"),
        "detail": detail,
    }


def evaluate(
    facts: dict[str, Any],
    manifest: dict[str, Any],
    *,
    profile: str,
    license_accepted: bool,
    model_source: str,
    check_network: bool,
    requested_backend: str = "auto",
) -> list[dict[str, str]]:
    checks: list[dict[str, str]] = []
    platform_facts = facts["platform"]
    checks.append(
        check(
            "platform",
            platform_facts["system"] == "linux" and platform_facts["architecture"] == "amd64",
            f"observed {platform_facts['system']}/{platform_facts['architecture']}; required linux/amd64",
        )
    )
    persistent_path = facts["persistent_root"]["path"]
    checks.append(
        check(
            "persistent-root",
            facts["persistent_root"]["exists"]
            and not facts["persistent_root"]["is_symlink"]
            and not MAC_PATH_PATTERN.match(persistent_path),
            f"persistent root {persistent_path}",
        )
    )
    checks.append(
        check(
            "disk",
            facts["host"]["disk_free_bytes"] >= manifest["minimum_free_bytes"],
            f"{facts['host']['disk_free_bytes']} bytes free; {manifest['minimum_free_bytes']} required",
        )
    )
    backend_facts = facts.get("execution_backends")
    if backend_facts is None:
        docker = facts["docker"]
        backend_facts = {
            "compose": {
                "engine_ready": docker["engine_ready"],
                "engine_version": docker["engine_version"],
                "compose_ready": docker["compose_ready"],
                "compose_version": docker["compose_version"],
            },
            "native": {"ready": False, "missing": ["native facts unavailable"], "commands": {}, "versions": {}},
        }
    try:
        selected_backend = radeon_backend.resolve_backend(requested_backend, backend_facts)
    except radeon_backend.BackendError:
        selected_backend = None
    checks.append(
        check(
            "execution-backend",
            selected_backend is not None,
            (
                f"selected {selected_backend} from requested {requested_backend}"
                if selected_backend
                else f"no usable backend for requested {requested_backend}"
            ),
        )
    )
    if selected_backend == "compose":
        compose = backend_facts["compose"]
        checks.append(
            check(
                "docker-engine",
                compose["engine_ready"] and version_at_least(compose["engine_version"], (24, 0)),
                f"existing Docker Engine {compose['engine_version'] or 'unavailable'}; minimum 24.0",
            )
        )
        checks.append(
            check(
                "docker-compose",
                compose["compose_ready"] and version_at_least(compose["compose_version"], (2, 20)),
                f"Docker Compose {compose['compose_version'] or 'unavailable'}; minimum 2.20",
            )
        )
    elif selected_backend == "native":
        native = backend_facts["native"]
        checks.append(
            check(
                "native-host-tools",
                native["ready"],
                (
                    "all declared native host tools are available"
                    if native["ready"]
                    else f"missing native tools: {', '.join(native['missing'])}"
                ),
            )
        )
        versions = native.get("versions", {})
        checks.append(
            check(
                "native-python",
                version_at_least(versions.get("python3"), (3, 11)),
                f"Python {versions.get('python3') or 'unavailable'}; minimum 3.11",
            )
        )
        checks.append(
            check(
                "native-node",
                version_at_least(versions.get("node"), (20, 19)),
                f"Node {versions.get('node') or 'unavailable'}; minimum 20.19",
            )
        )
        checks.append(
            check(
                "native-cmake",
                version_at_least(versions.get("cmake"), (3, 20)),
                f"CMake {versions.get('cmake') or 'unavailable'}; minimum 3.20",
            )
        )

    needs_gpu = profile in {"radeon-local", "championship"}
    if needs_gpu:
        devices = facts["devices"]
        checks.append(check("device-kfd", devices["kfd"]["exists"], "/dev/kfd is available"))
        checks.append(
            check("device-dri", devices["dri_exists"] and devices["render"]["exists"], "DRM render device is available")
        )
        observed_architectures = set(facts["gpu"]["architectures"])
        allowed = set(manifest["gpu"]["allowed_architectures"])
        checks.append(
            check(
                "gpu-architecture",
                bool(observed_architectures & allowed),
                f"observed {sorted(observed_architectures) or ['unknown']}; allowed {sorted(allowed)}",
            )
        )
        checks.append(
            check(
                "rocm",
                bool(facts["rocm_version"]),
                f"ROCm version {facts['rocm_version'] or 'not detected'}",
            )
        )
        checks.append(
            check(
                "device-groups",
                devices["kfd"]["gid"] is not None and devices["render"]["gid"] is not None,
                f"kfd gid={devices['kfd']['gid']}; render gid={devices['render']['gid']}",
            )
        )
        if not facts["model_cache_ready"]:
            checks.append(
                check(
                    "license",
                    license_accepted,
                    "explicit Gemma license acceptance recorded for first hydration",
                )
            )
            if model_source == "huggingface":
                token = facts["secrets"]["hf_token"]
                checks.append(
                    check(
                        "hf-token",
                        token["exists"] and token["regular"] and token["size_bytes"] > 0,
                        "file-mounted Hugging Face token is present and non-empty",
                    )
                )

    secret_directory = facts["secrets"]["directory"]
    secret_directory_safe = (
        not secret_directory["exists"]
        or (
            secret_directory["mode"] is not None
            and secret_directory["mode"] & 0o077 == 0
        )
    )
    checks.append(
        check(
            "secret-directory",
            secret_directory_safe,
            "secret directory is absent or has no group/world access",
        )
    )
    for name in ("hf_token", "radeon_api_key", "grafana_password"):
        metadata = facts["secrets"][name]
        if metadata["exists"]:
            checks.append(
                check(
                    f"secret-permissions-{name.replace('_', '-')}",
                    metadata["regular"]
                    and metadata["mode"] is not None
                    and metadata["mode"] & 0o022 == 0,
                    f"{name} is regular and not group/world writable",
                )
            )
    if profile == "championship":
        api_key = facts["secrets"]["radeon_api_key"]
        checks.append(
            check(
                "radeon-api-key",
                api_key["exists"] and api_key["regular"] and api_key["size_bytes"] > 0,
                "file-mounted Radeon API key is present and non-empty",
            )
        )
    if check_network:
        failed = [item["destination"] for item in facts["network"] if not item["reachable"]]
        checks.append(
            check(
                "first-run-network",
                not failed,
                "all declared first-run destinations reachable"
                if not failed
                else f"unreachable destinations: {', '.join(failed)}",
            )
        )
    else:
        checks.append(
            check(
                "first-run-network",
                True,
                "network check explicitly skipped; only valid for an already hydrated and pulled appliance",
                warning=not facts["model_cache_ready"] and needs_gpu,
            )
        )
    return checks


def shell_value(value: str) -> str:
    if "\n" in value or "\r" in value:
        raise ValueError("environment value contains a newline")
    return value


def generated_environment(
    facts: dict[str, Any],
    manifest: dict[str, Any],
    model_manifest: dict[str, Any],
    *,
    persist_root: Path,
    profile: str,
    license_accepted: bool,
    model_source: str,
    execution_backend: str,
) -> str:
    devices = facts["devices"]
    values = {
        "SIGNALFORGE_ACCEPT_GEMMA_LICENSE": "yes" if license_accepted else "no",
        "SIGNALFORGE_APPLICATION_ARTIFACT_IDENTITY": manifest["application"]["image"],
        "SIGNALFORGE_APP_IMAGE": manifest["application"]["image"],
        "SIGNALFORGE_EXECUTION_BACKEND": execution_backend,
        "SIGNALFORGE_LLAMA_ROCM_IMAGE": manifest["runtime"]["image"],
        "SIGNALFORGE_MODEL_ARTIFACT_IDENTITY": (
            "not-required" if profile == "fixture" else "sha256:" + model_manifest["sha256"]
        ),
        "SIGNALFORGE_MODEL_SOURCE": model_source,
        "SIGNALFORGE_PERSIST_ROOT": str(persist_root.resolve()),
        "SIGNALFORGE_RENDER_GID": str(devices["render"]["gid"] or 109),
        "SIGNALFORGE_RUNTIME_IDENTITY": (
            manifest["application"]["image"]
            if profile == "fixture"
            else manifest["runtime"]["image"]
        ),
        "SIGNALFORGE_VIDEO_GID": str(devices["kfd"]["gid"] or 44),
    }
    return "".join(f"{key}={shell_value(str(value))}\n" for key, value in sorted(values.items()))


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", type=Path, default=DEFAULT_MANIFEST)
    parser.add_argument("--model-manifest", type=Path, default=DEFAULT_MODEL_MANIFEST)
    parser.add_argument("--profile", choices=("fixture", "radeon-local", "championship"), default="radeon-local")
    parser.add_argument("--persist-root", type=Path)
    parser.add_argument("--secrets-dir", type=Path, default=ROOT / ".secrets")
    parser.add_argument("--report", type=Path)
    parser.add_argument("--env-output", type=Path)
    parser.add_argument("--model-source", choices=("existing", "huggingface", "oci"), default="huggingface")
    parser.add_argument(
        "--backend",
        choices=("auto", "compose", "native"),
        default=os.environ.get("SIGNALFORGE_EXECUTION_BACKEND", "auto"),
    )
    parser.add_argument("--license-accepted", action="store_true")
    parser.add_argument("--check-network", action="store_true")
    parser.add_argument("--network-timeout", type=float, default=4)
    args = parser.parse_args()

    manifest = load_json(args.manifest)
    model_manifest = load_json(args.model_manifest)
    default_root = (
        Path(manifest["persistent_root_default"])
        if Path("/workspace").is_dir()
        else ROOT / ".signalforge" / "radeon"
    )
    persist_root = (args.persist_root or default_root).expanduser()
    facts = collect_facts(
        manifest,
        model_manifest,
        persist_root,
        args.secrets_dir,
        args.check_network,
        args.network_timeout,
        args.profile,
        args.backend,
    )
    checks = evaluate(
        facts,
        manifest,
        profile=args.profile,
        license_accepted=args.license_accepted,
        model_source=args.model_source,
        check_network=args.check_network,
        requested_backend=args.backend,
    )
    try:
        selected_backend = radeon_backend.resolve_backend(
            args.backend, facts["execution_backends"]
        )
    except radeon_backend.BackendError:
        selected_backend = None
    failed = [item for item in checks if item["status"] == "failed"]
    report = {
        "schema_version": "signalforge/radeon-preflight/v1",
        "profile": args.profile,
        "requested_backend": args.backend,
        "selected_backend": selected_backend,
        "status": "failed" if failed else "passed",
        "appliance_version": manifest["appliance_version"],
        "facts": facts,
        "checks": checks,
    }
    encoded = json.dumps(report, indent=2, sort_keys=True) + "\n"
    if args.report:
        atomic_write(args.report, encoded, 0o600)
    if args.env_output and not failed:
        atomic_write(
            args.env_output,
            generated_environment(
                facts,
                manifest,
                model_manifest,
                persist_root=persist_root,
                profile=args.profile,
                license_accepted=args.license_accepted,
                model_source=args.model_source,
                execution_backend=selected_backend,
            ),
            0o600,
        )

    print(f"SignalForge Radeon preflight: {report['status']} ({args.profile})")
    for item in checks:
        symbol = {"passed": "OK", "warning": "WARN", "failed": "FAIL"}[item["status"]]
        print(f"[{symbol}] {item['id']}: {item['detail']}")
    if args.report:
        print(f"Machine report: {args.report}")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
