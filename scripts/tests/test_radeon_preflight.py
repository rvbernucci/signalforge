from __future__ import annotations

import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "radeon_preflight.py"
SPEC = importlib.util.spec_from_file_location("radeon_preflight", SCRIPT)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


def base_facts(root: Path) -> dict:
    return {
        "platform": {"system": "linux", "architecture": "amd64"},
        "devices": {
            "kfd": {"path": "/dev/kfd", "exists": True, "gid": 992, "mode": "crw-rw----"},
            "render": {
                "path": "/dev/dri/renderD128",
                "exists": True,
                "gid": 993,
                "mode": "crw-rw----",
            },
            "dri_exists": True,
        },
        "gpu": {
            "architectures": ["gfx1100"],
            "marketing_names": ["AMD Radeon PRO W7900"],
            "vram_bytes": 48 * 1024**3,
            "rocminfo_available": True,
            "rocm_smi_available": True,
        },
        "rocm_version": "7.2.1",
        "host": {
            "ram_bytes": 128 * 1024**3,
            "cpu_count": 32,
            "disk_total_bytes": 100 * 1024**3,
            "disk_free_bytes": 80 * 1024**3,
        },
        "docker": {
            "installed": True,
            "engine_ready": True,
            "compose_ready": True,
            "engine_version": "28.0.0",
            "compose_version": "2.39.0",
        },
        "execution_backends": {
            "compose": {
                "docker_path": "/usr/bin/docker",
                "engine_ready": True,
                "engine_version": "28.0.0",
                "compose_ready": True,
                "compose_version": "2.39.0",
            },
            "native": {
                "commands": {
                    name: f"/usr/bin/{name}"
                    for name in ("git", "curl", "python3", "node", "npm", "cmake", "ninja", "hipcc")
                },
                "versions": {
                    "python3": "Python 3.12.11",
                    "node": "v22.17.0",
                    "npm": "10.9.2",
                    "cmake": "cmake version 3.31.0",
                    "ninja": "1.12.1",
                    "hipcc": "HIP version: 7.2.1",
                    "git": "git version 2.49.0",
                    "curl": "curl 8.12.1",
                },
                "ready": True,
                "missing": [],
            },
        },
        "persistent_root": {
            "path": str(root / "runtime"),
            "exists": True,
            "is_symlink": False,
        },
        "model_cache_ready": False,
        "secrets": {
            "directory": {"path": str(root / ".secrets"), "exists": True, "mode": 0o700},
            "hf_token": {
                "path": str(root / ".secrets/hf-token"),
                "exists": True,
                "regular": True,
                "size_bytes": 32,
                "mode": 0o600,
            },
            "radeon_api_key": {
                "path": str(root / ".secrets/radeon-model-api-key"),
                "exists": True,
                "regular": True,
                "size_bytes": 32,
                "mode": 0o644,
            },
            "grafana_password": {
                "path": str(root / ".secrets/grafana-admin-password"),
                "exists": True,
                "regular": True,
                "size_bytes": 32,
                "mode": 0o644,
            },
        },
        "network": [
            {"destination": "ghcr.io:443", "reachable": True, "error": None},
        ],
    }


class RadeonPreflightTests(unittest.TestCase):
    def setUp(self) -> None:
        self.manifest = json.loads(
            (ROOT / "deploy/radeon/appliance-manifest.json").read_text(encoding="utf-8")
        )
        self.model_manifest = json.loads(
            (ROOT / "deploy/radeon/model-manifest.json").read_text(encoding="utf-8")
        )
        self.manifest["first_run_network_destinations"] = {
            "compose": ["ghcr.io:443"],
            "native": ["gh-proxy.org:443"],
            "model": ["hf-mirror.com:443"],
        }

    def test_local_profile_passes_on_supported_synthetic_radeon_host(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            checks = MODULE.evaluate(
                base_facts(Path(directory)),
                self.manifest,
                profile="radeon-local",
                license_accepted=True,
                model_source="huggingface",
                check_network=True,
            )
        self.assertFalse([item for item in checks if item["status"] == "failed"])

    def test_gpu_probe_can_read_architecture_after_verbose_cpu_agents(self) -> None:
        command = [
            sys.executable,
            "-c",
            "print('x' * 9000 + ' Name: gfx1100')",
        ]
        default = MODULE.run_safe(command, output_limit=8192)
        gpu_probe = MODULE.run_safe(command, output_limit=128 * 1024)

        self.assertNotIn("gfx1100", default["output"])
        self.assertIn("gfx1100", gpu_probe["output"])

    def test_championship_requires_nonempty_radeon_api_key(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            facts = base_facts(Path(directory))
            facts["secrets"]["radeon_api_key"]["size_bytes"] = 0
            checks = MODULE.evaluate(
                facts,
                self.manifest,
                profile="championship",
                license_accepted=True,
                model_source="huggingface",
                check_network=True,
            )
        failed = {item["id"] for item in checks if item["status"] == "failed"}
        self.assertIn("radeon-api-key", failed)

    def test_missing_device_group_and_wrong_gpu_fail_closed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            facts = base_facts(Path(directory))
            facts["devices"]["render"]["gid"] = None
            facts["gpu"]["architectures"] = ["gfx9999"]
            checks = MODULE.evaluate(
                facts,
                self.manifest,
                profile="radeon-local",
                license_accepted=True,
                model_source="huggingface",
                check_network=True,
            )
        failed = {item["id"] for item in checks if item["status"] == "failed"}
        self.assertEqual({"device-groups", "gpu-architecture"} & failed, {"device-groups", "gpu-architecture"})

    def test_mac_runtime_path_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            facts = base_facts(Path(directory))
            facts["persistent_root"]["path"] = "/Users/developer/models"
            checks = MODULE.evaluate(
                facts,
                self.manifest,
                profile="fixture",
                license_accepted=False,
                model_source="huggingface",
                check_network=False,
            )
        failed = {item["id"] for item in checks if item["status"] == "failed"}
        self.assertIn("persistent-root", failed)

    def test_cached_model_does_not_require_hf_token_or_new_license_input(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            facts = base_facts(Path(directory))
            facts["model_cache_ready"] = True
            facts["secrets"]["hf_token"]["exists"] = False
            facts["secrets"]["hf_token"]["size_bytes"] = 0
            checks = MODULE.evaluate(
                facts,
                self.manifest,
                profile="radeon-local",
                license_accepted=False,
                model_source="huggingface",
                check_network=False,
            )
        failed = {item["id"] for item in checks if item["status"] == "failed"}
        self.assertNotIn("hf-token", failed)
        self.assertNotIn("license", failed)

    def test_fixture_allows_secret_directory_to_be_absent(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            facts = base_facts(Path(directory))
            facts["secrets"]["directory"] = {
                "path": str(Path(directory) / ".secrets"),
                "exists": False,
                "mode": None,
            }
            facts["secrets"]["hf_token"]["exists"] = False
            facts["secrets"]["radeon_api_key"]["exists"] = False
            facts["secrets"]["grafana_password"]["exists"] = False
            checks = MODULE.evaluate(
                facts,
                self.manifest,
                profile="fixture",
                license_accepted=False,
                model_source="huggingface",
                check_network=False,
            )
        failed = {item["id"] for item in checks if item["status"] == "failed"}
        self.assertNotIn("secret-directory", failed)

    def test_native_network_scope_excludes_compose_and_adds_model_only_when_needed(self) -> None:
        native_fixture = MODULE.required_network_destinations(
            self.manifest,
            profile="fixture",
            backend="native",
            model_cache_ready=False,
        )
        native_local = MODULE.required_network_destinations(
            self.manifest,
            profile="radeon-local",
            backend="native",
            model_cache_ready=False,
        )
        self.assertEqual(native_fixture, ["gh-proxy.org:443"])
        self.assertEqual(native_local, ["gh-proxy.org:443", "hf-mirror.com:443"])
        self.assertNotIn("ghcr.io:443", native_local)

    def test_generated_environment_contains_only_derived_nonsecrets(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            generated = MODULE.generated_environment(
                base_facts(root),
                self.manifest,
                self.model_manifest,
                MODULE.radeon_manifest.select_manifest(environment={}),
                persist_root=root / "runtime",
                profile="radeon-local",
                license_accepted=True,
                model_source="huggingface",
                execution_backend="compose",
            )
        self.assertIn("SIGNALFORGE_RENDER_GID=993", generated)
        self.assertIn("SIGNALFORGE_VIDEO_GID=992", generated)
        self.assertIn("SIGNALFORGE_ACCEPT_GEMMA_LICENSE=yes", generated)
        self.assertIn(
            "SIGNALFORGE_APPLIANCE_MANIFEST=deploy/radeon/appliance-manifest.json",
            generated,
        )
        self.assertRegex(
            generated,
            r"SIGNALFORGE_APPLIANCE_MANIFEST_SHA256=[0-9a-f]{64}",
        )
        self.assertIn("SIGNALFORGE_APPLICATION_ARTIFACT_IDENTITY=ghcr.io/", generated)
        self.assertIn("SIGNALFORGE_MODEL_ARTIFACT_IDENTITY=sha256:", generated)
        self.assertIn("SIGNALFORGE_RUNTIME_IDENTITY=rocm/llama.cpp@sha256:", generated)
        self.assertNotIn("TOKEN", generated)
        self.assertNotIn("API_KEY", generated)

    def test_offline_first_run_is_a_warning_not_a_false_pass(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            checks = MODULE.evaluate(
                base_facts(Path(directory)),
                self.manifest,
                profile="radeon-local",
                license_accepted=True,
                model_source="huggingface",
                check_network=False,
            )
        network = next(item for item in checks if item["id"] == "first-run-network")
        self.assertEqual(network["status"], "warning")

    def test_outdated_compose_is_rejected_without_installing_a_replacement(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            facts = base_facts(Path(directory))
            facts["execution_backends"]["compose"]["compose_version"] = "2.19.1"
            checks = MODULE.evaluate(
                facts,
                self.manifest,
                profile="fixture",
                license_accepted=False,
                model_source="huggingface",
                check_network=False,
                requested_backend="compose",
            )
        compose = next(item for item in checks if item["id"] == "docker-compose")
        self.assertEqual(compose["status"], "failed")

    def test_auto_backend_uses_native_when_oneclick_has_no_container_runtime(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            facts = base_facts(Path(directory))
            facts["execution_backends"]["compose"].update(
                {
                    "docker_path": None,
                    "engine_ready": False,
                    "engine_version": None,
                    "compose_ready": False,
                    "compose_version": None,
                }
            )
            checks = MODULE.evaluate(
                facts,
                self.manifest,
                profile="radeon-local",
                license_accepted=True,
                model_source="huggingface",
                check_network=True,
                requested_backend="auto",
            )
        failed = [item for item in checks if item["status"] == "failed"]
        selected = next(item for item in checks if item["id"] == "execution-backend")
        self.assertFalse(failed)
        self.assertIn("selected native", selected["detail"])

    def test_explicit_native_fails_closed_when_required_host_tool_is_missing(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            facts = base_facts(Path(directory))
            facts["execution_backends"]["native"]["ready"] = False
            facts["execution_backends"]["native"]["missing"] = ["hipcc"]
            facts["execution_backends"]["native"]["commands"]["hipcc"] = None
            checks = MODULE.evaluate(
                facts,
                self.manifest,
                profile="radeon-local",
                license_accepted=True,
                model_source="huggingface",
                check_network=True,
                requested_backend="native",
            )
        failed = {item["id"] for item in checks if item["status"] == "failed"}
        self.assertIn("execution-backend", failed)


if __name__ == "__main__":
    unittest.main()
