from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path


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


if __name__ == "__main__":
    unittest.main()
