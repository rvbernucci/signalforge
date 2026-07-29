from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "radeon_bootstrap.py"
SPEC = importlib.util.spec_from_file_location("radeon_bootstrap", SCRIPT)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class RadeonBootstrapTests(unittest.TestCase):
    def test_atomic_secret_writes_private_single_line_file(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "secret"
            MODULE.atomic_secret(path, "synthetic-test-value")
            self.assertEqual(path.read_text(), "synthetic-test-value")
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)

    def test_atomic_secret_rejects_multiline_values(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(ValueError, "one line"):
                MODULE.atomic_secret(Path(directory) / "secret", "line1\nline2")

    def test_private_directory_rejects_symlink(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            target = root / "target"
            target.mkdir()
            link = root / "link"
            link.symlink_to(target, target_is_directory=True)
            with self.assertRaisesRegex(ValueError, "unsafe"):
                MODULE.ensure_private_directory(link)

    def test_placeholder_preserves_existing_secret_value(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "secret"
            path.write_text("preserve-me")
            MODULE.ensure_placeholder(path, 0o600)
            self.assertEqual(path.read_text(), "preserve-me")
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)

    def test_persist_root_prefers_explicit_then_environment(self) -> None:
        with mock.patch.dict(
            "os.environ",
            {"SIGNALFORGE_PERSIST_ROOT": "/tmp/signalforge-from-environment"},
            clear=False,
        ):
            self.assertEqual(
                MODULE.resolve_persist_root(None),
                Path("/tmp/signalforge-from-environment"),
            )
            self.assertEqual(
                MODULE.resolve_persist_root(Path("~/signalforge-explicit")),
                Path("~/signalforge-explicit").expanduser(),
            )


if __name__ == "__main__":
    unittest.main()
