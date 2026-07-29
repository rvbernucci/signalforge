from __future__ import annotations

import hashlib
import importlib.util
import io
import json
import tempfile
import unittest
import urllib.request
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "radeon_model_cache.py"
SPEC = importlib.util.spec_from_file_location("radeon_model_cache", SCRIPT)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


def write_manifest(root: Path, payload: bytes) -> Path:
    digest = hashlib.sha256(payload).hexdigest()
    manifest = {
        "schema_version": "signalforge/model-artifact/v1",
        "manifest_version": "fixture-v1",
        "model_id": "fixture/model",
        "served_model_id": "fixture-model",
        "repository": "fixture/model",
        "revision": "0123456789abcdef",
        "filename": "fixture.gguf",
        "expected_size_bytes": len(payload),
        "sha256": digest,
        "license": {
            "id": "fixture",
            "url": "https://example.invalid/license",
            "acceptance_required": True,
        },
        "sources": {
            "existing": {"enabled": True},
            "huggingface": {
                "enabled": True,
                "url": "https://example.invalid/fixture.gguf",
            },
            "oci": {"enabled": False},
        },
        "cache": {
            "model_relative_path": "fixture.gguf",
            "partial_relative_path": ".downloads/fixture.gguf.part",
            "ready_marker_relative_path": ".ready.json",
            "lock_relative_path": ".cache.lock",
        },
    }
    path = root / "manifest.json"
    path.write_text(json.dumps(manifest), encoding="utf-8")
    return path


class RadeonModelCacheTests(unittest.TestCase):
    def test_existing_source_is_verified_and_reused_idempotently(self) -> None:
        payload = b"GGUF-fixture-payload"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = write_manifest(root, payload)
            source = root / "source.gguf"
            source.write_bytes(payload)
            cache = root / "cache"
            state = root / "state.json"

            first = MODULE.hydrate(
                manifest_path=manifest,
                cache_dir=cache,
                source="existing",
                token_file=None,
                existing_file=source,
                license_accepted=True,
                retries=1,
                timeout_seconds=1,
                state_path=state,
            )
            source.unlink()
            second = MODULE.hydrate(
                manifest_path=manifest,
                cache_dir=cache,
                source="auto",
                token_file=None,
                existing_file=None,
                license_accepted=False,
                retries=1,
                timeout_seconds=1,
                state_path=state,
            )

            self.assertEqual(first["cache"], "hydrated")
            self.assertEqual(second["cache"], "reused")
            self.assertEqual((cache / "fixture.gguf").read_bytes(), payload)
            self.assertEqual(json.loads(state.read_text())["status"], "ready")
            self.assertEqual((cache / "fixture.gguf").stat().st_mode & 0o777, 0o444)

    def test_corrupted_cache_is_rejected_and_recovered_from_existing_source(self) -> None:
        payload = b"correct-model"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = write_manifest(root, payload)
            source = root / "source.gguf"
            source.write_bytes(payload)
            cache = root / "cache"
            arguments = dict(
                manifest_path=manifest,
                cache_dir=cache,
                source="existing",
                token_file=None,
                existing_file=source,
                license_accepted=True,
                retries=1,
                timeout_seconds=1,
                state_path=None,
            )
            MODULE.hydrate(**arguments)
            model = cache / "fixture.gguf"
            model.chmod(0o644)
            model.write_bytes(b"broken-model!")

            recovered = MODULE.hydrate(**arguments)

            self.assertEqual(recovered["cache"], "hydrated")
            self.assertEqual(model.read_bytes(), payload)
            self.assertTrue((cache / ".ready.json").is_file())

    def test_wrong_source_hash_never_publishes_ready_marker(self) -> None:
        payload = b"correct-model"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = write_manifest(root, payload)
            source = root / "source.gguf"
            source.write_bytes(b"wrong--model")
            cache = root / "cache"

            with self.assertRaises(MODULE.ModelCacheError):
                MODULE.hydrate(
                    manifest_path=manifest,
                    cache_dir=cache,
                    source="existing",
                    token_file=None,
                    existing_file=source,
                    license_accepted=True,
                    retries=1,
                    timeout_seconds=1,
                )

            self.assertFalse((cache / ".ready.json").exists())
            self.assertFalse((cache / "fixture.gguf").exists())
            self.assertFalse((cache / ".downloads/fixture.gguf.part").exists())

    def test_resume_starts_at_existing_partial_offset(self) -> None:
        payload = b"0123456789abcdefghijklmnopqrstuvwxyz"
        with tempfile.TemporaryDirectory() as directory:
            partial = Path(directory) / "model.part"
            partial.write_bytes(payload[:11])
            observed_offsets: list[int] = []

            def opener(
                _url: str,
                _token: str | None,
                offset: int,
                _timeout: float,
            ) -> tuple[io.BytesIO, int]:
                observed_offsets.append(offset)
                return io.BytesIO(payload[offset:]), 206

            MODULE.download_resumable(
                "https://example.invalid/model",
                "not-printed",
                partial,
                len(payload),
                retries=2,
                timeout_seconds=1,
                opener=opener,
            )

            self.assertEqual(observed_offsets, [11])
            self.assertEqual(partial.read_bytes(), payload)

    def test_server_without_range_support_restarts_partial_safely(self) -> None:
        payload = b"complete-payload"
        with tempfile.TemporaryDirectory() as directory:
            partial = Path(directory) / "model.part"
            partial.write_bytes(b"partial")

            def opener(
                _url: str,
                _token: str | None,
                _offset: int,
                _timeout: float,
            ) -> tuple[io.BytesIO, int]:
                return io.BytesIO(payload), 200

            MODULE.download_resumable(
                "https://example.invalid/model",
                "not-printed",
                partial,
                len(payload),
                retries=1,
                timeout_seconds=1,
                opener=opener,
            )

            self.assertEqual(partial.read_bytes(), payload)

    def test_license_and_file_mounted_token_are_fail_closed(self) -> None:
        payload = b"model"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = write_manifest(root, payload)
            with self.assertRaisesRegex(MODULE.ModelCacheError, "license acceptance"):
                MODULE.hydrate(
                    manifest_path=manifest,
                    cache_dir=root / "cache",
                    source="huggingface",
                    token_file=root / "missing-token",
                    existing_file=None,
                    license_accepted=False,
                    retries=1,
                    timeout_seconds=1,
                )

    def test_manifest_rejects_cache_path_traversal(self) -> None:
        payload = b"model"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = write_manifest(root, payload)
            data = json.loads(manifest.read_text())
            data["cache"]["model_relative_path"] = "../escape.gguf"
            manifest.write_text(json.dumps(data), encoding="utf-8")

            with self.assertRaisesRegex(MODULE.ModelCacheError, "unsafe"):
                MODULE.load_manifest(manifest)

    def test_symlink_cache_directory_is_rejected(self) -> None:
        payload = b"model"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = write_manifest(root, payload)
            target = root / "target"
            target.mkdir()
            cache = root / "cache"
            cache.symlink_to(target, target_is_directory=True)

            with self.assertRaisesRegex(MODULE.ModelCacheError, "unsafe"):
                MODULE.hydrate(
                    manifest_path=manifest,
                    cache_dir=cache,
                    source="existing",
                    token_file=None,
                    existing_file=root / "missing",
                    license_accepted=True,
                    retries=1,
                    timeout_seconds=1,
                )

    def test_cross_host_redirect_drops_provider_authorization(self) -> None:
        request = urllib.request.Request(
            "https://huggingface.co/model",
            headers={"Authorization": "Bearer synthetic", "Range": "bytes=10-"},
        )
        redirected = MODULE.SafeRedirectHandler().redirect_request(
            request,
            None,
            302,
            "Found",
            {},
            "https://cdn-lfs.hf.co/signed-object",
        )
        assert redirected is not None
        self.assertIsNone(redirected.get_header("Authorization"))
        self.assertEqual(redirected.get_header("Range"), "bytes=10-")

    def test_same_host_redirect_preserves_provider_authorization(self) -> None:
        request = urllib.request.Request(
            "https://huggingface.co/model",
            headers={"Authorization": "Bearer synthetic"},
        )
        redirected = MODULE.SafeRedirectHandler().redirect_request(
            request,
            None,
            302,
            "Found",
            {},
            "https://huggingface.co/model?download=true",
        )
        assert redirected is not None
        self.assertEqual(redirected.get_header("Authorization"), "Bearer synthetic")


if __name__ == "__main__":
    unittest.main()
