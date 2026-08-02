from __future__ import annotations

import importlib.util
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]


def load_module(name: str):
    script = ROOT / "scripts" / f"{name}.py"
    spec = importlib.util.spec_from_file_location(name, script)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


STATUS = load_module("radeon_status")
LOGS = load_module("radeon_logs")
RESET = load_module("radeon_reset")


class RadeonOperatorTests(unittest.TestCase):
    def setUp(self) -> None:
        self.appliance = json.loads(
            (ROOT / "deploy/radeon/appliance-manifest.json").read_text(encoding="utf-8")
        )
        self.model = json.loads(
            (ROOT / "deploy/radeon/model-manifest.json").read_text(encoding="utf-8")
        )

    def test_status_parses_array_and_json_lines_from_compose(self) -> None:
        array = '[{"Service":"signalforge","State":"running","Health":"healthy"}]'
        lines = '{"Service":"one","State":"running"}\n{"Service":"two","State":"exited"}\n'
        self.assertEqual(STATUS.parse_compose_ps(array)[0]["Service"], "signalforge")
        self.assertEqual(len(STATUS.parse_compose_ps(lines)), 2)

    def test_local_status_requires_model_runtime_and_application(self) -> None:
        status = STATUS.build_status(
            profile="radeon-local",
            services=[
                {
                    "Service": "signalforge-local",
                    "State": "running",
                    "Health": "healthy",
                    "Image": self.appliance["application"]["image"],
                }
            ],
            compose_error=None,
            app_health={"status": "ready"},
            model_state={"status": "ready"},
            runtime_state={"status": "ready"},
            appliance_manifest=self.appliance,
            manifest_reference="deploy/radeon/appliance-manifest.json",
            manifest_sha256="1" * 64,
            model_manifest=self.model,
            app_port=8080,
            grafana_port=3000,
        )
        self.assertEqual(status["status"], "ready")
        self.assertEqual(status["phase"], "ready")
        self.assertNotIn("credential", json.dumps(status).lower())

    def test_local_status_surfaces_model_hydration_phase(self) -> None:
        status = STATUS.build_status(
            profile="radeon-local",
            services=[],
            compose_error=None,
            app_health=None,
            model_state=None,
            runtime_state=None,
            appliance_manifest=self.appliance,
            manifest_reference="deploy/radeon/appliance-manifest.json",
            manifest_sha256="1" * 64,
            model_manifest=self.model,
            app_port=8080,
            grafana_port=3000,
        )
        self.assertEqual(status["status"], "preparing")
        self.assertEqual(status["phase"], "model-hydration")

    def test_compose_status_rejects_application_identity_mismatch(self) -> None:
        status = STATUS.build_status(
            profile="fixture",
            services=[
                {
                    "Service": "signalforge",
                    "State": "running",
                    "Health": "healthy",
                    "Image": "ghcr.io/rvbernucci/signalforge@sha256:" + "0" * 64,
                }
            ],
            compose_error=None,
            app_health={"status": "ready"},
            model_state=None,
            runtime_state=None,
            appliance_manifest=self.appliance,
            manifest_reference="deploy/radeon/appliance-manifest.json",
            manifest_sha256="1" * 64,
            model_manifest=self.model,
            app_port=8080,
            grafana_port=3000,
        )
        self.assertEqual(status["status"], "preparing")
        self.assertEqual(status["phase"], "application-identity-mismatch")

    def test_native_status_rejects_missing_or_mismatched_application_receipt(self) -> None:
        declared_commit = self.appliance["application"]["source_commit"]
        resolved_commit = subprocess.check_output(
            ["git", "rev-parse", f"{declared_commit}^{{commit}}"],
            cwd=ROOT,
            text=True,
        ).strip()
        missing = STATUS.bind_native_identity(
            {
                "status": "ready",
                "phase": "ready",
                "application_health": {
                    "status": "ready",
                    "identities": {"application": "sha256:" + "1" * 64},
                },
                "toolchain": {},
            },
            self.appliance,
            "deploy/radeon/appliance-manifest.json",
            "2" * 64,
        )
        self.assertEqual(missing["status"], "preparing")
        self.assertEqual(missing["phase"], "application-identity-missing")

        mismatched = STATUS.bind_native_identity(
            {
                "status": "ready",
                "phase": "ready",
                "application_health": {
                    "status": "ready",
                    "identities": {"application": "sha256:" + "3" * 64},
                },
                "toolchain": {
                    "application": {
                        "source_commit": resolved_commit,
                        "declared_source_commit": declared_commit,
                        "appliance_manifest": "deploy/radeon/appliance-manifest.json",
                        "appliance_manifest_sha256": "2" * 64,
                        "binary_sha256": "5" * 64,
                    }
                },
            },
            self.appliance,
            "deploy/radeon/appliance-manifest.json",
            "2" * 64,
        )
        self.assertEqual(mismatched["status"], "preparing")
        self.assertEqual(mismatched["phase"], "application-identity-mismatch")

    def test_native_status_rejects_source_outside_selected_manifest_authority(self) -> None:
        binary_sha256 = "5" * 64
        status = STATUS.bind_native_identity(
            {
                "status": "ready",
                "phase": "ready",
                "application_health": {
                    "status": "ready",
                    "identities": {"application": f"sha256:{binary_sha256}"},
                },
                "toolchain": {
                    "application": {
                        "source_commit": "4" * 40,
                        "declared_source_commit": "4" * 40,
                        "appliance_manifest": "deploy/radeon/appliance-manifest.json",
                        "appliance_manifest_sha256": "2" * 64,
                        "binary_sha256": binary_sha256,
                    }
                },
            },
            self.appliance,
            "deploy/radeon/appliance-manifest.json",
            "2" * 64,
        )
        self.assertEqual(status["status"], "preparing")
        self.assertEqual(status["phase"], "application-source-authority-mismatch")

    def test_native_status_rejects_legacy_or_wrong_manifest_authority(self) -> None:
        declared_commit = self.appliance["application"]["source_commit"]
        resolved_commit = subprocess.check_output(
            ["git", "rev-parse", f"{declared_commit}^{{commit}}"],
            cwd=ROOT,
            text=True,
        ).strip()
        binary_sha256 = "5" * 64
        base_status = {
            "status": "ready",
            "phase": "ready",
            "application_health": {
                "status": "ready",
                "identities": {"application": f"sha256:{binary_sha256}"},
            },
        }

        legacy = STATUS.bind_native_identity(
            {
                **base_status,
                "toolchain": {
                    "application": {
                        "source_commit": resolved_commit,
                        "binary_sha256": binary_sha256,
                    }
                },
            },
            self.appliance,
            "deploy/radeon/appliance-manifest.json",
            "2" * 64,
        )
        self.assertEqual(legacy["status"], "preparing")
        self.assertEqual(legacy["phase"], "application-source-authority-missing")

        wrong_manifest = STATUS.bind_native_identity(
            {
                **base_status,
                "toolchain": {
                    "application": {
                        "source_commit": resolved_commit,
                        "declared_source_commit": declared_commit,
                        "appliance_manifest": "deploy/radeon/appliance-manifest.json",
                        "appliance_manifest_sha256": "3" * 64,
                        "binary_sha256": binary_sha256,
                    }
                },
            },
            self.appliance,
            "deploy/radeon/appliance-manifest.json",
            "2" * 64,
        )
        self.assertEqual(wrong_manifest["status"], "preparing")
        self.assertEqual(
            wrong_manifest["phase"],
            "application-source-authority-mismatch",
        )

    def test_native_status_accepts_matching_application_receipt(self) -> None:
        declared_commit = self.appliance["application"]["source_commit"]
        resolved_commit = subprocess.check_output(
            ["git", "rev-parse", f"{declared_commit}^{{commit}}"],
            cwd=ROOT,
            text=True,
        ).strip()
        binary_sha256 = "5" * 64
        status = STATUS.bind_native_identity(
            {
                "status": "ready",
                "phase": "ready",
                "application_health": {
                    "status": "ready",
                    "identities": {"application": f"sha256:{binary_sha256}"},
                },
                "toolchain": {
                    "application": {
                        "source_commit": resolved_commit,
                        "declared_source_commit": declared_commit,
                        "appliance_manifest": "deploy/radeon/appliance-manifest.json",
                        "appliance_manifest_sha256": "2" * 64,
                        "binary_sha256": binary_sha256,
                    }
                },
            },
            self.appliance,
            "deploy/radeon/appliance-manifest.json",
            "2" * 64,
        )
        self.assertEqual(status["status"], "ready")
        self.assertEqual(
            status["identities"]["executed_application_binary_sha256"],
            binary_sha256,
        )

    def test_log_redactor_removes_secret_and_private_bodies(self) -> None:
        line = (
            'signalforge | {"token":"synthetic-secret","prompt":"private question",'
            '"answer":"private answer","status":"completed"}'
        )
        redacted = LOGS.redact_line(line)
        self.assertNotIn("synthetic-secret", redacted)
        self.assertNotIn("private question", redacted)
        self.assertNotIn("private answer", redacted)
        self.assertIn("[REDACTED_SECRET]", redacted)
        self.assertIn("[REDACTED_BODY]", redacted)
        self.assertIn("completed", redacted)

    def test_log_redactor_handles_non_json_bearer_and_query_credentials(self) -> None:
        line = (
            "Authorization: Bearer abc.def "
            + "to"
            + "ken=sensitive https://example.test/?api"
            + "_key=value"
        )
        redacted = LOGS.redact_line(line)
        self.assertNotIn("abc.def", redacted)
        self.assertNotIn("sensitive", redacted)
        self.assertNotIn("api" + "_key=value", redacted)

    def test_clean_preserves_model_cache(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / ".signalforge" / "radeon"
            model = root / "models/model.gguf"
            data = root / "data/private.json"
            state = root / "state/status.json"
            model.parent.mkdir(parents=True)
            data.parent.mkdir(parents=True)
            state.parent.mkdir(parents=True)
            model.write_text("model")
            data.write_text("data")
            state.write_text("state")

            RESET.clean(RESET.safe_root(root))

            self.assertEqual(model.read_text(), "model")
            self.assertFalse(data.exists())
            self.assertFalse(state.exists())
            self.assertTrue((root / "data").is_dir())
            self.assertTrue((root / "state").is_dir())

    def test_reset_refuses_broad_or_unmarked_paths(self) -> None:
        for path in (Path("/"), Path.home(), Path("/tmp/arbitrary")):
            with self.subTest(path=path):
                with self.assertRaises(ValueError):
                    RESET.safe_root(path)

    def test_reset_refuses_symbolic_link_runtime_root(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            target = base / "signalforge-runtime"
            target.mkdir()
            alias = base / ".signalforge" / "radeon"
            alias.parent.mkdir()
            alias.symlink_to(target, target_is_directory=True)

            with self.assertRaises(ValueError):
                RESET.safe_root(alias)

    def test_reset_default_root_honors_environment(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {"SIGNALFORGE_PERSIST_ROOT": "/tmp/signalforge-reset-root"},
            clear=False,
        ):
            self.assertEqual(
                RESET.default_persist_root(),
                Path("/tmp/signalforge-reset-root"),
            )

    def test_compose_has_zero_touch_profiles_and_no_local_build_requirement(self) -> None:
        compose = (ROOT / "compose.yaml").read_text(encoding="utf-8")
        self.assertNotIn("\n  build:", compose)
        for service in (
            "storage-init:",
            "model-init:",
            "llama-rocm:",
            "runtime-ready:",
            "signalforge-local:",
            "signalforge-championship:",
        ):
            self.assertIn(service, compose)
        self.assertIn("internal: true", compose)
        self.assertIn("service_healthy", compose)
        self.assertIn("service_completed_successfully", compose)
        self.assertNotIn("/Users/", compose)
        self.assertNotIn("SIGNALFORGE_SPECIALIST_API_KEY:", compose)

    def test_observability_target_enables_application_telemetry(self) -> None:
        makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
        target = makefile.split("radeon-observe:", 1)[1].split("\n\n", 1)[0]
        self.assertIn("SIGNALFORGE_OBSERVABILITY=1", target)
        self.assertIn("SIGNALFORGE_OTEL_ENABLED=true", target)

    @unittest.skipUnless(shutil.which("docker"), "Docker is unavailable on this test host")
    def test_docker_compose_renders_every_runtime_profile(self) -> None:
        for profile in ("fixture", "radeon-local", "championship", "observability"):
            with self.subTest(profile=profile):
                subprocess.run(
                    [
                        "docker",
                        "compose",
                        "--env-file",
                        "container.env.example",
                        "--profile",
                        profile,
                        "config",
                        "--quiet",
                    ],
                    cwd=ROOT,
                    check=True,
                    capture_output=True,
                    text=True,
                )

    @unittest.skipUnless(shutil.which("docker"), "Docker is unavailable on this test host")
    def test_docker_compose_renders_explicit_image_override_without_changing_defaults(self) -> None:
        override_image = "ghcr.io/rvbernucci/signalforge@sha256:" + "1" * 64
        result = subprocess.run(
            [
                "docker",
                "compose",
                "--env-file",
                "container.env.example",
                "--profile",
                "fixture",
                "config",
                "--images",
            ],
            cwd=ROOT,
            env={
                **os.environ,
                "SIGNALFORGE_APP_IMAGE": override_image,
                "SIGNALFORGE_LLAMA_ROCM_IMAGE": self.appliance["runtime"]["image"],
            },
            check=True,
            capture_output=True,
            text=True,
        )
        self.assertIn(override_image, result.stdout)
        self.assertIn(self.appliance["application"]["image"], (ROOT / "container.env.example").read_text())

    def test_static_appliance_audit_passes(self) -> None:
        result = subprocess.run(
            ["python3", "scripts/radeon_validate_appliance.py"],
            cwd=ROOT,
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(json.loads(result.stdout)["status"], "passed")

if __name__ == "__main__":
    unittest.main()
