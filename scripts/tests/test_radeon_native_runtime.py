from __future__ import annotations

import importlib.util
import socket
import tempfile
import unittest
from pathlib import Path
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location(
    "radeon_native_runtime", ROOT / "scripts/radeon_native_runtime.py"
)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class RadeonNativeRuntimeTests(unittest.TestCase):
    def test_championship_uses_api_key_file_without_secret_value(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            environment = MODULE.app_environment(root, "championship", root / ".secrets")
        self.assertEqual(
            environment["SIGNALFORGE_SPECIALIST_API_KEY_FILE"],
            str(root / ".secrets/radeon-model-api-key"),
        )
        self.assertNotIn("SIGNALFORGE_SPECIALIST_API_KEY", environment)

    def test_native_commands_bind_model_and_application_to_loopback(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            command = MODULE.app_command(root / "signalforge-workspace", root, "radeon-local")
            llama_environment = MODULE.llama_environment(root, root / "llama-server")
        self.assertIn("127.0.0.1:8080", command)
        self.assertIn("http://127.0.0.1:8000/v1", command)
        self.assertEqual(llama_environment["SIGNALFORGE_MODEL_HOST"], "127.0.0.1")
        self.assertEqual(llama_environment["SIGNALFORGE_VERIFY_MODEL_HASH"], "1")

    def test_empty_native_status_is_fail_closed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            status = MODULE.native_status(Path(directory), "radeon-local")
        self.assertEqual(status["status"], "preparing")
        self.assertEqual(status["phase"], "model-runtime")
        self.assertFalse(status["processes"]["app"]["alive"])
        self.assertFalse(status["processes"]["llama"]["alive"])

    def test_port_guard_rejects_an_untracked_listener(self) -> None:
        listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.addCleanup(listener.close)
        listener.bind(("127.0.0.1", 0))
        listener.listen(1)
        port = listener.getsockname()[1]
        with self.assertRaisesRegex(MODULE.NativeRuntimeError, "untracked process"):
            MODULE.ensure_loopback_port_available(port, "application")

    def test_port_guard_allows_an_unused_loopback_port(self) -> None:
        reservation = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        reservation.bind(("127.0.0.1", 0))
        port = reservation.getsockname()[1]
        reservation.close()
        MODULE.ensure_loopback_port_available(port, "application")

    def test_wait_app_rejects_readiness_from_a_different_build(self) -> None:
        with (
            tempfile.TemporaryDirectory() as directory,
            mock.patch.object(MODULE, "read_process", return_value={"pid": 123}),
            mock.patch.object(MODULE, "process_matches", return_value=True),
            mock.patch.object(
                MODULE,
                "fetch_json",
                return_value={
                    "status": "ready",
                    "mode": "fixture",
                    "build_version": "different-build",
                },
            ),
        ):
            with self.assertRaisesRegex(MODULE.NativeRuntimeError, "identity"):
                MODULE.wait_app(
                    Path(directory),
                    timeout_seconds=1,
                    expected_build_version="expected-build",
                    expected_mode="fixture",
                    poll_seconds=0.001,
                )


if __name__ == "__main__":
    unittest.main()
