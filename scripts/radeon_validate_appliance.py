#!/usr/bin/env python3
"""Validate the static zero-touch Radeon appliance contract without Docker or network."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Any

import radeon_manifest


ROOT = Path(__file__).resolve().parents[1]
DIGEST_IMAGE = re.compile(r"^[a-z0-9./:_-]+@sha256:[0-9a-f]{64}$")


def result(check_id: str, passed: bool, detail: str) -> dict[str, str]:
    return {"id": check_id, "status": "passed" if passed else "failed", "detail": detail}


def resolve_git_commit(root: Path, declared_commit: str) -> str | None:
    resolved = subprocess.run(
        ["git", "-C", str(root), "rev-parse", "--verify", f"{declared_commit}^{{commit}}"],
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    candidate = resolved.stdout.strip()
    if resolved.returncode or not re.fullmatch(r"[0-9a-f]{40}", candidate):
        return None
    return candidate if candidate.startswith(declared_commit) else None


def git_blob_sha256(root: Path, commit: str, path: str) -> str | None:
    blob = subprocess.run(
        ["git", "-C", str(root), "show", f"{commit}:{path}"],
        check=False,
        capture_output=True,
        timeout=30,
    )
    if blob.returncode:
        return None
    return hashlib.sha256(blob.stdout).hexdigest()


def validate(root: Path, manifest_path: Path | None = None) -> dict[str, Any]:
    selection = radeon_manifest.select_manifest(
        manifest_path or root / "deploy/radeon/appliance-manifest.json",
        environment={},
    )
    appliance = selection.manifest
    default_manifest_path = root / "deploy/radeon/appliance-manifest.json"
    default_appliance = json.loads(default_manifest_path.read_text(encoding="utf-8"))
    is_default = selection.path == default_manifest_path.resolve()
    model_manifest_path = radeon_manifest.component_path(
        appliance["model_manifest"],
        "model_manifest",
    )
    native_manifest_path = radeon_manifest.component_path(
        appliance["execution"]["native"]["toolchain_manifest"],
        "execution.native.toolchain_manifest",
    )
    model = json.loads(
        model_manifest_path.read_text(encoding="utf-8")
    )
    native = json.loads(
        native_manifest_path.read_text(encoding="utf-8")
    )
    compose = (root / "compose.yaml").read_text(encoding="utf-8")
    makefile = (root / "Makefile").read_text(encoding="utf-8")
    environment = (root / "container.env.example").read_text(encoding="utf-8")
    preflight = (root / "scripts/radeon_preflight.py").read_text(encoding="utf-8")
    preflight_shell = (root / "scripts/radeon_preflight.sh").read_text(encoding="utf-8")
    startup = (root / "scripts/radeon_up.sh").read_text(encoding="utf-8")
    stage = (root / "scripts/stage_gemma_model.sh").read_text(encoding="utf-8")
    checks: list[dict[str, str]] = []

    checks.append(
        result(
            "platform",
            appliance.get("platform") == "linux/amd64",
            "appliance platform is linux/amd64",
        )
    )
    for identity in ("application", "runtime"):
        image = appliance[identity]["image"]
        checks.append(
            result(
                f"pinned-{identity}-image",
                bool(DIGEST_IMAGE.fullmatch(image)),
                f"{identity} image uses an immutable sha256 digest",
            )
        )
        if is_default:
            identity_consistent = image in compose and image in environment
            identity_detail = (
                f"{identity} rollback identity is consistent across manifest, Compose, "
                "and environment example"
            )
        else:
            variable = {
                "application": "SIGNALFORGE_APP_IMAGE",
                "runtime": "SIGNALFORGE_LLAMA_ROCM_IMAGE",
            }[identity]
            identity_consistent = (
                variable in compose
                and variable in preflight
                and "SIGNALFORGE_APPLIANCE_MANIFEST" in preflight_shell
                and "--manifest-sha256" in startup
            )
            identity_detail = (
                f"{identity} candidate identity is transported through the hash-bound "
                "generated environment"
            )
        checks.append(
            result(
                f"compose-{identity}-identity",
                identity_consistent,
                identity_detail,
            )
        )
    checks.append(
        result(
            "rollback-default-preserved",
            default_appliance["application"]["image"] in compose
            and default_appliance["application"]["image"] in environment
            and default_appliance["runtime"]["image"] in compose
            and default_appliance["runtime"]["image"] in environment,
            "accepted rollback remains the static Compose and environment default",
        )
    )
    checks.append(
        result(
            "candidate-source-commit",
            is_default or bool(re.fullmatch(r"[0-9a-f]{40}", appliance["application"]["source_commit"])),
            "candidate application source commit is complete",
        )
    )
    for identity, image in appliance["utility_images"].items():
        checks.append(
            result(
                f"pinned-utility-{identity}",
                bool(DIGEST_IMAGE.fullmatch(image)),
                f"{identity} utility image uses an immutable sha256 digest",
            )
        )
    checks.append(
        result(
            "model-sha",
            bool(re.fullmatch(r"[0-9a-f]{64}", model["sha256"])),
            "model artifact has a complete SHA-256",
        )
    )
    native_runtime = (root / "scripts/radeon_native_runtime.py").read_text(encoding="utf-8")
    checks.append(
        result(
            "model-manifest-consistency",
            model["served_model_id"] in compose
            and model["filename"] in compose
            and "radeon_model_cache.py" in compose
            and "radeon_manifest.component_path" in native_runtime
            and 'appliance["model_manifest"]' in native_runtime
            and "radeon_model_cache.hydrate" in native_runtime
            and 'model_manifest["cache"]["model_relative_path"]' in native_runtime
            and 'model_manifest["served_model_id"]' in native_runtime,
            "Compose and native backends consume the same model identity",
        )
    )
    checks.append(
        result(
            "native-go-pin",
            native["go"]["version"] == "1.25.12"
            and native["go"]["sha256"]
            == "234828b7a89e0e303d2556310ee549fbcf253d28de937bac3da13d6294262ac1",
            "native Go toolchain is version- and SHA-pinned",
        )
    )
    checks.append(
        result(
            "native-llama-pin",
            native["llama_cpp"]["revision"]
            == appliance["execution"]["native"]["llama_cpp_revision"]
            == "305ba519ab61cdff8044922cba2347826a04453f",
            "native llama.cpp build uses the reviewed revision",
        )
    )
    checks.append(
        result(
            "native-source-locks",
            native["application"]["package_lock_sha256"]
            == hashlib.sha256((root / native["application"]["package_lock_path"]).read_bytes()).hexdigest()
            and native["application"]["go_sum_sha256"]
            == hashlib.sha256((root / native["application"]["go_sum_path"]).read_bytes()).hexdigest(),
            "native npm and Go dependency locks match the manifest",
        )
    )
    resolved_source_commit = resolve_git_commit(
        root,
        appliance["application"]["source_commit"],
    )
    checks.append(
        result(
            "native-authorized-source-available",
            resolved_source_commit is not None,
            "selected application source commit resolves locally without a runtime fetch",
        )
    )
    checks.append(
        result(
            "native-authorized-source-locks",
            resolved_source_commit is not None
            and git_blob_sha256(
                root,
                resolved_source_commit,
                native["application"]["package_lock_path"],
            )
            == native["application"]["package_lock_sha256"]
            and git_blob_sha256(
                root,
                resolved_source_commit,
                native["application"]["go_sum_path"],
            )
            == native["application"]["go_sum_sha256"],
            "dependency locks in the selected source commit match the native toolchain authority",
        )
    )
    native_toolchain = (
        root / "scripts/radeon_native_toolchain.py"
    ).read_text(encoding="utf-8")
    native_status = (root / "scripts/radeon_status.py").read_text(encoding="utf-8")
    checks.append(
        result(
            "native-source-authority-wiring",
            "--manifest \"$SIGNALFORGE_APPLIANCE_MANIFEST\"" in startup
            and "--manifest-sha256 \"$SIGNALFORGE_APPLIANCE_MANIFEST_SHA256\"" in startup
            and "resolve_source_commit" in native_runtime
            and "materialize_source" in native_toolchain
            and "application_authority_error" in native_runtime
            and "application-source-authority-mismatch" in native_status,
            "native startup, build, and status share the selected manifest source authority",
        )
    )
    checks.append(
        result(
            "native-selected-source-assets",
            "source_root / \"web/dist\"" in native_runtime
            and "source_root / \"fixtures/productscope/technology20-catalog.json\""
            in native_runtime,
            "native runtime assets come from the same selected source tree as the binary",
        )
    )
    checks.append(
        result(
            "native-regional-downloads",
            native["go"]["url"].startswith("https://dl.google.com/")
            and native["application"]["go_proxy"] == "https://goproxy.cn,direct"
            and native["application"]["go_sumdb"]
            == "sum.golang.org https://sum.golang.google.cn",
            "native dependency transport matches the Radeon region and remains lock-verified",
        )
    )
    network = appliance["first_run_network_destinations"]
    checks.append(
        result(
            "backend-scoped-network",
            set(network) == {"compose", "native", "model"}
            and "gh-proxy.org:443" in network["native"]
            and "hf-mirror.com:443" in network["model"],
            "first-run connectivity is scoped to the selected backend and model requirement",
        )
    )
    checks.append(
        result(
            "model-transport-authority",
            model["sources"]["huggingface"]["url"].startswith("https://hf-mirror.com/")
            and model["sources"]["huggingface"]["canonical_url"].startswith(
                "https://huggingface.co/"
            ),
            "regional model transport preserves the canonical Hugging Face locator",
        )
    )
    checks.append(
        result(
            "no-local-build",
            "\n  build:" not in compose and "--no-build" in makefile,
            "Compose judge startup never requires a local image build",
        )
    )
    checks.append(
        result(
            "backend-auto",
            "SIGNALFORGE_EXECUTION_BACKEND=auto" in environment
            and "radeon_native_runtime.py" in (root / "scripts/radeon_up.sh").read_text(),
            "operator startup selects Compose or native without installing host tooling",
        )
    )
    checks.append(
        result(
            "internal-local-network",
            "signalforge:" in compose and "internal: true" in compose,
            "local application and runtime share an internal network",
        )
    )
    for profile in ("fixture", "radeon-local", "championship", "observability"):
        checks.append(
            result(
                f"profile-{profile}",
                profile in compose,
                f"{profile} profile is declared",
            )
        )
    for target in (
        "radeon-bootstrap",
        "radeon-preflight",
        "radeon-up",
        "radeon-status",
        "radeon-logs",
        "radeon-observe",
        "radeon-down",
        "radeon-clean",
        "radeon-reset",
    ):
        checks.append(
            result(
                f"make-{target}",
                re.search(rf"(?m)^{re.escape(target)}(?:\s*:[^\n]*)?$", makefile) is not None,
                f"Make target {target} is available",
            )
        )
    checks.append(
        result(
            "stage-file-secret",
            "SIGNALFORGE_HF_TOKEN_FILE" in stage and "${HF_TOKEN" not in stage,
            "staging consumes a file-mounted token rather than an environment secret",
        )
    )
    checks.append(
        result(
            "no-mac-path",
            "/Users/" not in compose
            and "/Users/" not in environment
            and "/Users/" not in json.dumps(appliance)
            and "/Users/" not in json.dumps(native),
            "runtime contracts contain no Mac path",
        )
    )
    checks.append(
        result(
            "no-credential-env",
            "SIGNALFORGE_SPECIALIST_API_KEY=" not in compose
            and "HF_TOKEN=" not in compose
            and "HF_TOKEN=" not in environment,
            "credential values are absent from environment contracts",
        )
    )
    checks.append(
        result(
            "native-file-secret",
            "SIGNALFORGE_SPECIALIST_API_KEY_FILE" in (
                root / "scripts/radeon_native_runtime.py"
            ).read_text()
            and "SIGNALFORGE_SPECIALIST_API_KEY\"" not in (
                root / "scripts/radeon_native_runtime.py"
            ).read_text(),
            "native championship passes only the API key file path",
        )
    )
    tracked = subprocess.check_output(
        ["git", "-C", str(root), "ls-files"], text=True
    ).splitlines()
    checks.append(
        result(
            "no-tracked-secrets",
            not [
                name
                for name in tracked
                if name.startswith(".secrets/")
                or (name.startswith(".env.") and name != ".env.example")
            ],
            "no runtime secret or private environment file is tracked",
        )
    )
    scripts = [
        "scripts/stage_gemma_model.sh",
        "scripts/radeon_compose.sh",
        "scripts/radeon_preflight.sh",
        "scripts/radeon_up.sh",
        "scripts/radeon_logs.sh",
        "scripts/radeon_down.sh",
    ]
    syntax = subprocess.run(
        ["bash", "-n", *scripts],
        cwd=root,
        check=False,
        capture_output=True,
        text=True,
    )
    checks.append(result("shell-syntax", syntax.returncode == 0, "operator shell scripts parse"))
    failed = [item for item in checks if item["status"] == "failed"]
    return {
        "schema_version": "signalforge/radeon-appliance-static-audit/v1",
        "status": "failed" if failed else "passed",
        "appliance_version": appliance["appliance_version"],
        "manifest_authority": {
            "path": selection.reference,
            "sha256": selection.sha256,
        },
        "checks": checks,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=ROOT)
    parser.add_argument("--manifest", type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    try:
        report = validate(args.root.resolve(), args.manifest)
    except (radeon_manifest.ManifestError, OSError, KeyError, json.JSONDecodeError) as error:
        print(f"Radeon appliance validation failed: {error}", file=sys.stderr)
        return 2
    encoded = json.dumps(report, indent=2, sort_keys=True) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(encoded, encoding="utf-8")
    print(encoded, end="")
    return 0 if report["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
