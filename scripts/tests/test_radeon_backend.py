from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location(
    "radeon_backend", ROOT / "scripts/radeon_backend.py"
)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


def facts(*, compose: bool, native: bool) -> dict:
    return {
        "compose": {
            "engine_ready": compose,
            "compose_ready": compose,
            "engine_version": "28.0.0" if compose else None,
            "compose_version": "2.39.0" if compose else None,
        },
        "native": {
            "ready": native,
            "missing": [] if native else ["hipcc"],
            "commands": {},
            "versions": {},
        },
    }


class RadeonBackendTests(unittest.TestCase):
    def test_auto_prefers_compose_when_it_is_healthy(self) -> None:
        self.assertEqual(MODULE.resolve_backend("auto", facts(compose=True, native=True)), "compose")

    def test_auto_falls_back_to_native_on_current_oneclick_shape(self) -> None:
        self.assertEqual(MODULE.resolve_backend("auto", facts(compose=False, native=True)), "native")

    def test_explicit_unavailable_backend_fails_closed(self) -> None:
        with self.assertRaisesRegex(MODULE.BackendError, "unavailable"):
            MODULE.resolve_backend("compose", facts(compose=False, native=True))

    def test_generated_backend_reader_accepts_only_resolved_values(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "generated.env"
            path.write_text("SIGNALFORGE_EXECUTION_BACKEND=native\n")
            self.assertEqual(MODULE.read_generated_backend(path), "native")
            path.write_text("SIGNALFORGE_EXECUTION_BACKEND=auto\n")
            self.assertIsNone(MODULE.read_generated_backend(path))


if __name__ == "__main__":
    unittest.main()
