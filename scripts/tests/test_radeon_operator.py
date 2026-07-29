from __future__ import annotations

import importlib.util
import json
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
                    "Image": "pinned",
                }
            ],
            compose_error=None,
            app_health={"status": "ready"},
            model_state={"status": "ready"},
            runtime_state={"status": "ready"},
            appliance_manifest=self.appliance,
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
            model_manifest=self.model,
            app_port=8080,
            grafana_port=3000,
        )
        self.assertEqual(status["status"], "preparing")
        self.assertEqual(status["phase"], "model-hydration")

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
