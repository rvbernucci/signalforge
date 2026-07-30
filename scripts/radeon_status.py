#!/usr/bin/env python3
"""Report safe appliance phase, health, and immutable identities."""

from __future__ import annotations

import argparse
import json
import os
import re
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

import radeon_backend
import radeon_manifest
import radeon_native_runtime
import radeon_native_toolchain


GIT_COMMIT = re.compile(r"^[0-9a-f]{40}$")
SHA256 = re.compile(r"^[0-9a-f]{64}$")


def load_json(path: Path) -> dict[str, Any] | None:
    if not path.is_file() or path.is_symlink():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None


def atomic_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    try:
        temporary.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        os.chmod(temporary, 0o600)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


def parse_compose_ps(value: str) -> list[dict[str, Any]]:
    stripped = value.strip()
    if not stripped:
        return []
    try:
        parsed = json.loads(stripped)
        if isinstance(parsed, list):
            return [item for item in parsed if isinstance(item, dict)]
        if isinstance(parsed, dict):
            return [parsed]
    except json.JSONDecodeError:
        pass
    services: list[dict[str, Any]] = []
    for line in stripped.splitlines():
        try:
            parsed = json.loads(line)
        except json.JSONDecodeError:
            continue
        if isinstance(parsed, dict):
            services.append(parsed)
    return services


def compose_ps(profile: str) -> tuple[list[dict[str, Any]], str | None]:
    environment = {**os.environ, "SIGNALFORGE_PROFILE": profile}
    command = [str(ROOT / "scripts" / "radeon_compose.sh"), "current", "ps", "--format", "json"]
    try:
        result = subprocess.run(
            command,
            cwd=ROOT,
            env=environment,
            check=False,
            capture_output=True,
            text=True,
            timeout=20,
        )
    except (OSError, subprocess.TimeoutExpired) as error:
        return [], type(error).__name__
    if result.returncode:
        return [], (result.stderr.strip() or "docker compose ps failed")[:512]
    return parse_compose_ps(result.stdout), None


def fetch_json(url: str, timeout: float = 2) -> dict[str, Any] | None:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as response:
            if response.status != 200:
                return None
            return json.load(response)
    except (OSError, urllib.error.URLError, json.JSONDecodeError):
        return None


def service_summary(services: list[dict[str, Any]]) -> list[dict[str, Any]]:
    result: list[dict[str, Any]] = []
    for service in services:
        result.append(
            {
                "service": service.get("Service") or service.get("Name"),
                "state": service.get("State"),
                "health": service.get("Health"),
                "exit_code": service.get("ExitCode"),
                "image": service.get("Image"),
            }
        )
    return sorted(result, key=lambda item: str(item["service"]))


def expected_app_service(profile: str) -> str:
    return {
        "fixture": "signalforge",
        "radeon-local": "signalforge-local",
        "championship": "signalforge-championship",
    }[profile]


def bind_native_identity(
    status: dict[str, Any],
    appliance: dict[str, Any],
    manifest_reference: str,
    manifest_sha256: str,
) -> dict[str, Any]:
    toolchain = status.get("toolchain") or {}
    application_receipt = toolchain.get("application") or {}
    source_commit = application_receipt.get("source_commit")
    binary_sha256 = application_receipt.get("binary_sha256")
    health_identity = (
        (status.get("application_health") or {})
        .get("identities", {})
        .get("application")
    )
    receipt_complete = bool(
        isinstance(source_commit, str)
        and GIT_COMMIT.fullmatch(source_commit)
        and isinstance(binary_sha256, str)
        and SHA256.fullmatch(binary_sha256)
    )
    identity_matches = bool(
        receipt_complete and health_identity == f"sha256:{binary_sha256}"
    )
    selection = radeon_manifest.ManifestSelection(
        path=ROOT / manifest_reference,
        reference=manifest_reference,
        sha256=manifest_sha256,
        manifest=appliance,
    )
    try:
        resolved_source_commit = radeon_native_toolchain.resolve_source_commit(
            ROOT,
            appliance["application"]["source_commit"],
        )
        authority_error = radeon_native_toolchain.application_authority_error(
            application_receipt,
            selection,
            resolved_source_commit,
        )
    except radeon_native_toolchain.NativeToolchainError:
        resolved_source_commit = None
        authority_error = "application-source-authority-mismatch"
    if status.get("status") == "ready" and not receipt_complete:
        status["status"] = "preparing"
        status["phase"] = "application-identity-missing"
    elif status.get("status") == "ready" and authority_error:
        status["status"] = "preparing"
        status["phase"] = authority_error
    elif status.get("status") == "ready" and not identity_matches:
        status["status"] = "preparing"
        status["phase"] = "application-identity-mismatch"
    status["manifest_authority"] = {
        "path": manifest_reference,
        "sha256": manifest_sha256,
    }
    status["identities"] = {
        "appliance_version": appliance["appliance_version"],
        "appliance_manifest": manifest_reference,
        "appliance_manifest_sha256": manifest_sha256,
        "declared_application_image": appliance["application"]["image"],
        "declared_application_source_commit": appliance["application"]["source_commit"],
        "resolved_application_source_commit": resolved_source_commit,
        "executed_application_source_commit": (
            source_commit if receipt_complete else None
        ),
        "executed_application_binary_sha256": (
            binary_sha256 if receipt_complete else None
        ),
        "llama_cpp_revision": appliance["execution"]["native"]["llama_cpp_revision"],
    }
    return status


def build_status(
    *,
    profile: str,
    services: list[dict[str, Any]],
    compose_error: str | None,
    app_health: dict[str, Any] | None,
    model_state: dict[str, Any] | None,
    runtime_state: dict[str, Any] | None,
    appliance_manifest: dict[str, Any],
    manifest_reference: str,
    manifest_sha256: str,
    model_manifest: dict[str, Any],
    app_port: int,
    grafana_port: int,
) -> dict[str, Any]:
    summarized = service_summary(services)
    expected = expected_app_service(profile)
    app_service = next((item for item in summarized if item["service"] == expected), None)
    expected_image = appliance_manifest["application"]["image"]
    application_image_matches = bool(
        app_service and app_service["image"] == expected_image
    )
    app_ready = (
        app_service is not None
        and app_service["state"] == "running"
        and application_image_matches
        and app_health is not None
        and app_health.get("status") == "ready"
    )
    local_required = profile in {"radeon-local", "championship"}
    model_ready = bool(model_state and model_state.get("status") == "ready")
    runtime_ready = bool(runtime_state and runtime_state.get("status") == "ready")
    ready = not compose_error and app_ready and (
        not local_required or (model_ready and runtime_ready)
    )
    if ready:
        phase = "ready"
    elif compose_error:
        phase = "compose-unavailable"
    elif app_service is not None and not application_image_matches:
        phase = "application-identity-mismatch"
    elif local_required and not model_ready:
        phase = "model-hydration"
    elif local_required and not runtime_ready:
        phase = "model-load-or-runtime-health"
    else:
        phase = "application-startup"
    return {
        "schema_version": "signalforge/radeon-appliance-status/v1",
        "status": "ready" if ready else "preparing",
        "phase": phase,
        "profile": profile,
        "compose_error": compose_error,
        "services": summarized,
        "application_health": app_health,
        "model_state": model_state,
        "runtime_state": runtime_state,
        "manifest_authority": {
            "path": manifest_reference,
            "sha256": manifest_sha256,
        },
        "identities": {
            "appliance_version": appliance_manifest["appliance_version"],
            "application_image": appliance_manifest["application"]["image"],
            "runtime_image": appliance_manifest["runtime"]["image"],
            "model_id": model_manifest["served_model_id"],
            "model_revision": model_manifest["revision"],
            "model_sha256": model_manifest["sha256"],
        },
        "urls": {
            "workspace": f"http://127.0.0.1:{app_port}",
            "grafana": f"http://127.0.0.1:{grafana_port}",
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--profile", choices=("fixture", "radeon-local", "championship"), default="radeon-local")
    parser.add_argument("--backend", choices=("auto", "compose", "native"), default="auto")
    parser.add_argument("--persist-root", type=Path)
    parser.add_argument("--manifest", type=Path)
    parser.add_argument("--manifest-sha256")
    parser.add_argument("--wait-seconds", type=float, default=0)
    parser.add_argument("--poll-seconds", type=float, default=3)
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    persist_root = args.persist_root or radeon_backend.default_persist_root()
    try:
        generated_manifest = radeon_manifest.read_generated_environment(
            persist_root / "state/generated.env"
        )
        manifest_selection = radeon_manifest.select_manifest(
            args.manifest,
            args.manifest_sha256,
            generated_environment=generated_manifest,
        )
    except (radeon_manifest.ManifestError, OSError) as error:
        print(f"SignalForge appliance: unavailable ({error})", file=sys.stderr)
        return 2
    appliance = manifest_selection.manifest
    model_manifest_path = radeon_manifest.component_path(
        appliance["model_manifest"],
        "model_manifest",
    )
    model = json.loads(
        model_manifest_path.read_text(encoding="utf-8")
    )
    requested_backend = args.backend
    if requested_backend == "auto":
        requested_backend = (
            radeon_backend.read_generated_backend(
                persist_root / "state/generated.env",
                generated_manifest,
            )
            or "auto"
        )
    try:
        backend = radeon_backend.resolve_backend(
            requested_backend, radeon_backend.backend_facts(appliance)
        )
    except radeon_backend.BackendError as error:
        print(f"SignalForge appliance: unavailable ({error})", file=sys.stderr)
        return 1
    app_port = int(os.environ.get("SIGNALFORGE_APP_PORT", "8080"))
    grafana_port = int(os.environ.get("SIGNALFORGE_GRAFANA_PORT", "3000"))
    if backend == "native":
        deadline = time.monotonic() + args.wait_seconds
        while True:
            status = radeon_native_runtime.native_status(persist_root, args.profile)
            status = bind_native_identity(
                status,
                appliance,
                manifest_selection.reference,
                manifest_selection.sha256,
            )
            status["identities"].update(
                {
                    "model_id": model["served_model_id"],
                    "model_revision": model["revision"],
                    "model_sha256": model["sha256"],
                }
            )
            if status["status"] == "ready" or time.monotonic() >= deadline:
                break
            time.sleep(args.poll_seconds)
        report = args.report or persist_root / "state/status.json"
        try:
            atomic_json(report, status)
        except OSError as error:
            print(
                f"Warning: status report could not be written ({type(error).__name__}).",
                file=sys.stderr,
            )
        print(
            f"SignalForge appliance: {status['status']} "
            f"(backend=native, profile={status['profile']}, phase={status['phase']})"
        )
        for name, process in status["processes"].items():
            if name == "llama" and args.profile == "fixture":
                continue
            print(f"- {name}: {'running' if process['alive'] else 'stopped'}")
        print(f"Workspace: {status['workspace_url']}")
        print(
            f"Model: {status['identities']['model_id']} @ "
            f"{status['identities']['model_revision'][:12]}"
        )
        print("Safe next command: make radeon-logs")
        return 0 if status["status"] == "ready" else 1

    deadline = time.monotonic() + args.wait_seconds
    status: dict[str, Any]
    while True:
        services, compose_error = compose_ps(args.profile)
        status = build_status(
            profile=args.profile,
            services=services,
            compose_error=compose_error,
            app_health=fetch_json(f"http://127.0.0.1:{app_port}/health/ready"),
            model_state=load_json(persist_root / "state/model-init.json"),
            runtime_state=load_json(persist_root / "state/runtime-ready.json"),
            appliance_manifest=appliance,
            manifest_reference=manifest_selection.reference,
            manifest_sha256=manifest_selection.sha256,
            model_manifest=model,
            app_port=app_port,
            grafana_port=grafana_port,
        )
        if status["status"] == "ready" or time.monotonic() >= deadline:
            break
        time.sleep(args.poll_seconds)
    report = args.report or persist_root / "state/status.json"
    try:
        atomic_json(report, status)
    except OSError as error:
        print(f"Warning: status report could not be written ({type(error).__name__}).", file=sys.stderr)
    print(
        f"SignalForge appliance: {status['status']} "
        f"(backend=compose, profile={status['profile']}, phase={status['phase']})"
    )
    for service in status["services"]:
        health = f", health={service['health']}" if service["health"] else ""
        print(f"- {service['service']}: {service['state']}{health}")
    print(f"Workspace: {status['urls']['workspace']}")
    print(f"Model: {status['identities']['model_id']} @ {status['identities']['model_revision'][:12]}")
    if os.environ.get("SIGNALFORGE_OBSERVABILITY") == "1":
        print(f"Grafana: {status['urls']['grafana']}")
    print("Safe next command: make radeon-logs")
    return 0 if status["status"] == "ready" else 1


if __name__ == "__main__":
    raise SystemExit(main())
