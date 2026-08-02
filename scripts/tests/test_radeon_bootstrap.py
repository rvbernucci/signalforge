from __future__ import annotations

import importlib.util
import os
import subprocess
import sys
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

    def test_conflicting_manifest_authorities_fail_before_bootstrap(self) -> None:
        with (
            tempfile.TemporaryDirectory() as directory,
            tempfile.TemporaryDirectory(dir=ROOT / "deploy/radeon") as manifest_directory,
        ):
            override = Path(manifest_directory) / "appliance-manifest-override.json"
            override.write_text(
                (ROOT / "deploy/radeon/appliance-manifest.json").read_text(
                    encoding="utf-8"
                ),
                encoding="utf-8",
            )
            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPT),
                    "--profile",
                    "fixture",
                    "--manifest",
                    str(override),
                    "--persist-root",
                    directory,
                    "--noninteractive",
                    "--skip-network-check",
                ],
                cwd=ROOT,
                env={
                    **os.environ,
                    "SIGNALFORGE_APPLIANCE_MANIFEST": (
                        "deploy/radeon/appliance-manifest.json"
                    ),
                },
                check=False,
                capture_output=True,
                text=True,
            )
        self.assertEqual(result.returncode, 2)
        self.assertIn("conflicting appliance manifest", result.stderr)


if __name__ == "__main__":
    unittest.main()
