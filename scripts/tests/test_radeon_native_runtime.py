from __future__ import annotations

import importlib.util
import json
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
MANIFEST = MODULE.radeon_manifest


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
            source_root = root / "authorized-source"
            command = MODULE.app_command(
                root / "signalforge-workspace",
                root,
                "radeon-local",
                source_root,
            )
            llama_environment = MODULE.llama_environment(root, root / "llama-server")
        self.assertIn("127.0.0.1:8080", command)
        self.assertIn("http://127.0.0.1:8000/v1", command)
        self.assertIn(str(source_root / "web/dist"), command)
        self.assertIn(
            str(source_root / "fixtures/golden/financial-snapshot.json"),
            command,
        )
        self.assertEqual(llama_environment["SIGNALFORGE_MODEL_HOST"], "127.0.0.1")
        self.assertEqual(llama_environment["SIGNALFORGE_VERIFY_MODEL_HASH"], "1")

    def test_start_rejects_unresolvable_manifest_source_before_hydration(self) -> None:
        appliance = json.loads(
            (ROOT / "deploy/radeon/appliance-manifest.json").read_text(
                encoding="utf-8"
            )
        )
        appliance["application"]["source_commit"] = "f" * 40
        selection = MANIFEST.ManifestSelection(
            path=ROOT / "deploy/radeon/appliance-manifest.json",
            reference="deploy/radeon/appliance-manifest.json",
            sha256="1" * 64,
            manifest=appliance,
        )
        with (
            tempfile.TemporaryDirectory() as directory,
            mock.patch.object(MODULE, "hydrate_model") as hydrate,
        ):
            with self.assertRaisesRegex(MODULE.NativeRuntimeError, "authorized source"):
                MODULE.start(
                    Path(directory),
                    "championship",
                    ROOT / ".secrets",
                    manifest_selection=selection,
                    allow_dirty=True,
                )
        hydrate.assert_not_called()

    def test_mismatched_build_receipt_never_reaches_credential_environment(self) -> None:
        selection = MANIFEST.select_manifest(
            ROOT / "deploy/radeon/appliance-manifest.json",
            environment={},
        )
        binary_sha256 = "5" * 64
        unauthorized_toolchain = {
            "application": {
                "source_commit": "f" * 40,
                "declared_source_commit": "f" * 40,
                "appliance_manifest": selection.reference,
                "appliance_manifest_sha256": selection.sha256,
                "binary_sha256": binary_sha256,
            },
            "application_binary": "/tmp/unauthorized-signalforge",
            "llama_cpp": None,
        }
        stale_status = {
            "status": "ready",
            "phase": "ready",
            "toolchain": {"application": unauthorized_toolchain["application"]},
        }
        with (
            tempfile.TemporaryDirectory() as directory,
            mock.patch.object(MODULE, "native_status", return_value=stale_status),
            mock.patch.object(MODULE, "read_process", return_value=None),
            mock.patch.object(
                MODULE.radeon_native_toolchain,
                "prepare",
                return_value=unauthorized_toolchain,
            ),
            mock.patch.object(MODULE, "app_environment") as app_environment,
        ):
            with self.assertRaisesRegex(MODULE.NativeRuntimeError, "release authority"):
                MODULE.start(
                    Path(directory),
                    "fixture",
                    ROOT / ".secrets",
                    manifest_selection=selection,
                    allow_dirty=True,
                )
        app_environment.assert_not_called()

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

    def test_wait_app_rejects_runtime_or_model_identity_mismatch(self) -> None:
        with (
            tempfile.TemporaryDirectory() as directory,
            mock.patch.object(MODULE, "read_process", return_value={"pid": 123}),
            mock.patch.object(MODULE, "process_matches", return_value=True),
            mock.patch.object(
                MODULE,
                "fetch_json",
                return_value={
                    "status": "ready",
                    "mode": "live",
                    "build_version": "expected-build",
                    "identities": {
                        "runtime": "sha256:wrong-runtime",
                        "model": "sha256:expected-model",
                    },
                },
            ),
        ):
            with self.assertRaisesRegex(MODULE.NativeRuntimeError, "identity"):
                MODULE.wait_app(
                    Path(directory),
                    timeout_seconds=1,
                    expected_build_version="expected-build",
                    expected_mode="live",
                    expected_runtime_identity="sha256:expected-runtime",
                    expected_model_identity="sha256:expected-model",
                    poll_seconds=0.001,
                )

    def test_wait_app_rejects_application_identity_mismatch(self) -> None:
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
                    "build_version": "expected-build",
                    "identities": {"application": "sha256:wrong-application"},
                },
            ),
        ):
            with self.assertRaisesRegex(MODULE.NativeRuntimeError, "identity"):
                MODULE.wait_app(
                    Path(directory),
                    timeout_seconds=1,
                    expected_build_version="expected-build",
                    expected_mode="fixture",
                    expected_application_identity="sha256:expected-application",
                    poll_seconds=0.001,
                )


if __name__ == "__main__":
    unittest.main()
