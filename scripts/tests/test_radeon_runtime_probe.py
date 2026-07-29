from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "radeon_runtime_probe.py"
SPEC = importlib.util.spec_from_file_location("radeon_runtime_probe", SCRIPT)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class RadeonRuntimeProbeTests(unittest.TestCase):
    def test_probe_requires_health_and_exact_model_identity(self) -> None:
        calls: list[str] = []

        def fetcher(url: str, _timeout: float) -> dict:
            calls.append(url)
            if url.endswith("/health"):
                return {"status": "ok"}
            return {"data": [{"id": "signalforge-gemma4-26b-q4"}]}

        receipt = MODULE.wait_ready(
            "http://runtime:8000",
            "signalforge-gemma4-26b-q4",
            timeout_seconds=1,
            request_timeout_seconds=0.1,
            poll_seconds=0.001,
            fetcher=fetcher,
        )

        self.assertEqual(receipt["status"], "ready")
        self.assertEqual(receipt["attempts"], 1)
        self.assertEqual(len(calls), 2)

    def test_probe_rejects_open_endpoint_serving_wrong_model(self) -> None:
        def fetcher(url: str, _timeout: float) -> dict:
            if url.endswith("/health"):
                return {"status": "ok"}
            return {"data": [{"id": "wrong-model"}]}

        with self.assertRaisesRegex(TimeoutError, "model identity"):
            MODULE.wait_ready(
                "http://runtime:8000",
                "expected-model",
                timeout_seconds=0.01,
                request_timeout_seconds=0.01,
                poll_seconds=0.001,
                fetcher=fetcher,
            )

    def test_atomic_receipt_is_world_readable_but_not_writable(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "ready.json"
            MODULE.atomic_json(path, {"status": "ready"})
            self.assertEqual(path.stat().st_mode & 0o777, 0o444)
            self.assertIn('"status": "ready"', path.read_text())


if __name__ == "__main__":
    unittest.main()
