from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "radeon_manifest.py"
SPEC = importlib.util.spec_from_file_location("radeon_manifest", SCRIPT)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class RadeonManifestTests(unittest.TestCase):
    def test_default_preserves_accepted_rollback(self) -> None:
        selection = MODULE.select_manifest(environment={})
        self.assertEqual(
            selection.manifest["application"]["image"],
            "ghcr.io/rvbernucci/signalforge@sha256:"
            "cbac58cf3e62df0404e9ef1cfc7db6aec49e491e4beb5e1f214d6d562fad814b",
        )
        self.assertEqual(selection.reference, "deploy/radeon/appliance-manifest.json")

    def test_explicit_candidate_selects_only_candidate_digest(self) -> None:
        selection = MODULE.select_manifest(
            ROOT / "deploy/radeon/appliance-manifest.vnext.json",
            environment={},
        )
        self.assertEqual(
            selection.manifest["application"]["image"],
            "ghcr.io/rvbernucci/signalforge@sha256:"
            "2537c832a43b3e71e2352d18ae959803c2b2a218133517e96c4526ee0aeb3ab3",
        )
        self.assertEqual(selection.manifest["application"]["source_commit"], "ce4f2cabf0981bec09cf80c805864515f42fa41c")

    def test_conflicting_cli_environment_and_generated_authorities_fail(self) -> None:
        with self.assertRaisesRegex(MODULE.ManifestError, "conflicting appliance manifest"):
            MODULE.select_manifest(
                ROOT / "deploy/radeon/appliance-manifest.vnext.json",
                environment={
                    MODULE.MANIFEST_ENV: "deploy/radeon/appliance-manifest.json",
                },
            )
        with self.assertRaisesRegex(MODULE.ManifestError, "conflicting appliance manifest"):
            MODULE.select_manifest(
                environment={
                    MODULE.MANIFEST_ENV: "deploy/radeon/appliance-manifest.json",
                },
                generated_environment={
                    MODULE.MANIFEST_ENV: "deploy/radeon/appliance-manifest.vnext.json",
                },
            )

    def test_changed_manifest_bytes_fail_expected_hash(self) -> None:
        selection = MODULE.select_manifest(environment={})
        with self.assertRaisesRegex(MODULE.ManifestError, "bytes changed"):
            MODULE.select_manifest(
                selection.path,
                "0" * 64,
                environment={},
            )

    def test_missing_outside_and_symlink_manifests_fail(self) -> None:
        with self.assertRaisesRegex(MODULE.ManifestError, "missing"):
            MODULE.select_manifest(
                "deploy/radeon/appliance-manifest.missing.json",
                environment={},
            )
        with tempfile.TemporaryDirectory(dir=ROOT / "deploy/radeon") as directory:
            root = Path(directory)
            outside = Path(tempfile.gettempdir()) / "signalforge-outside-manifest.json"
            outside.write_text("{}")
            link = root / "appliance-manifest-link.json"
            link.symlink_to(outside)
            try:
                with self.assertRaisesRegex(MODULE.ManifestError, "symbolic"):
                    MODULE.select_manifest(link, environment={})
            finally:
                outside.unlink(missing_ok=True)

    def test_mutable_image_and_wrong_platform_fail(self) -> None:
        manifest = json.loads(
            (ROOT / "deploy/radeon/appliance-manifest.json").read_text()
        )
        manifest["application"]["image"] = "ghcr.io/rvbernucci/signalforge:latest"
        with self.assertRaisesRegex(MODULE.ManifestError, "immutable"):
            MODULE.validate_manifest(manifest)
        manifest["application"]["image"] = (
            "ghcr.io/rvbernucci/signalforge@sha256:" + "1" * 64
        )
        manifest["platform"] = "linux/arm64"
        with self.assertRaisesRegex(MODULE.ManifestError, "linux/amd64"):
            MODULE.validate_manifest(manifest)

    def test_component_manifest_cannot_escape_repository_authority(self) -> None:
        manifest = json.loads(
            (ROOT / "deploy/radeon/appliance-manifest.json").read_text()
        )
        manifest["model_manifest"] = "../../outside.json"
        with self.assertRaisesRegex(MODULE.ManifestError, "repository-relative"):
            MODULE.validate_manifest(manifest)

    def test_generated_environment_reader_accepts_only_manifest_authority(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "generated.env"
            path.write_text(
                "SIGNALFORGE_EXECUTION_BACKEND=native\n"
                "SIGNALFORGE_APPLIANCE_MANIFEST=deploy/radeon/appliance-manifest.vnext.json\n"
                f"SIGNALFORGE_APPLIANCE_MANIFEST_SHA256={'2' * 64}\n"
            )
            path.chmod(0o600)
            values = MODULE.read_generated_environment(path)
        self.assertEqual(
            values[MODULE.MANIFEST_ENV],
            "deploy/radeon/appliance-manifest.vnext.json",
        )
        self.assertEqual(values[MODULE.MANIFEST_SHA_ENV], "2" * 64)

    def test_generated_environment_rejects_unsafe_mode_and_unknown_keys(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "generated.env"
            path.write_text("SIGNALFORGE_EXECUTION_BACKEND=native\n")
            path.chmod(0o644)
            with self.assertRaisesRegex(MODULE.ManifestError, "owner-only"):
                MODULE.read_generated_environment(path)
            path.write_text("UNAPPROVED=value\n")
            path.chmod(0o600)
            with self.assertRaisesRegex(MODULE.ManifestError, "unapproved key"):
                MODULE.read_generated_environment(path)

    def test_shell_loader_treats_command_substitution_as_literal_data(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            marker = root / "executed"
            generated = root / "generated.env"
            generated.write_text(
                f"SIGNALFORGE_APP_IMAGE=$(touch {marker})\n"
                "SIGNALFORGE_EXECUTION_BACKEND=native\n"
            )
            generated.chmod(0o600)
            command = (
                'source "$1"; signalforge_load_generated_env "$2"; '
                'printf "%s" "$SIGNALFORGE_APP_IMAGE"'
            )
            result = subprocess.run(
                [
                    "bash",
                    "-c",
                    command,
                    "signalforge-generated-env-test",
                    str(ROOT / "scripts/radeon_generated_env.sh"),
                    str(generated),
                ],
                check=False,
                capture_output=True,
                text=True,
                env={**os.environ},
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, f"$(touch {marker})")
            self.assertFalse(marker.exists())

    def test_shell_loader_rejects_unknown_keys_and_public_mode(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            generated = root / "generated.env"
            generated.write_text("UNAPPROVED=value\n")
            generated.chmod(0o600)
            command = 'source "$1"; signalforge_load_generated_env "$2"'
            unknown = subprocess.run(
                [
                    "bash",
                    "-c",
                    command,
                    "signalforge-generated-env-test",
                    str(ROOT / "scripts/radeon_generated_env.sh"),
                    str(generated),
                ],
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(unknown.returncode, 0)
            self.assertIn("unapproved key", unknown.stderr)
            generated.write_text("SIGNALFORGE_EXECUTION_BACKEND=native\n")
            generated.chmod(0o644)
            public = subprocess.run(
                [
                    "bash",
                    "-c",
                    command,
                    "signalforge-generated-env-test",
                    str(ROOT / "scripts/radeon_generated_env.sh"),
                    str(generated),
                ],
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(public.returncode, 0)
            self.assertIn("owner-only", public.stderr)

    def test_direct_compose_wrapper_rejects_conflicting_persisted_authority(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            runtime = root / "runtime"
            state = runtime / "state"
            state.mkdir(parents=True)
            candidate = MODULE.select_manifest(
                ROOT / "deploy/radeon/appliance-manifest.vnext.json",
                environment={},
            )
            generated = state / "generated.env"
            generated.write_text(
                f"{MODULE.MANIFEST_ENV}={candidate.reference}\n"
                f"{MODULE.MANIFEST_SHA_ENV}={candidate.sha256}\n"
                f"SIGNALFORGE_APP_IMAGE={candidate.manifest['application']['image']}\n"
                "SIGNALFORGE_EXECUTION_BACKEND=compose\n"
            )
            generated.chmod(0o600)
            bin_dir = root / "bin"
            bin_dir.mkdir()
            docker = bin_dir / "docker"
            docker.write_text(
                "#!/usr/bin/env bash\n"
                'if [[ "$1 $2" == "compose version" ]]; then exit 0; fi\n'
                "exit 0\n"
            )
            docker.chmod(0o755)
            result = subprocess.run(
                [
                    str(ROOT / "scripts/radeon_compose.sh"),
                    "current",
                    "config",
                    "--quiet",
                ],
                cwd=ROOT,
                env={
                    **os.environ,
                    "PATH": f"{bin_dir}:{os.environ['PATH']}",
                    "SIGNALFORGE_PERSIST_ROOT": str(runtime),
                    MODULE.MANIFEST_ENV: "deploy/radeon/appliance-manifest.json",
                },
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.returncode, 2)
            self.assertIn("conflicting appliance manifest", result.stderr)


if __name__ == "__main__":
    unittest.main()
