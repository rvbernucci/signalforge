from __future__ import annotations

import importlib.util
import io
import json
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location(
    "radeon_native_toolchain", ROOT / "scripts/radeon_native_toolchain.py"
)
assert SPEC and SPEC.loader
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class RadeonNativeToolchainTests(unittest.TestCase):
    def test_manifest_pins_go_and_llama_cpp(self) -> None:
        manifest = json.loads(
            (ROOT / "deploy/radeon/native-toolchain-manifest.json").read_text()
        )
        self.assertEqual(manifest["go"]["version"], "1.25.12")
        self.assertEqual(
            manifest["go"]["sha256"],
            "234828b7a89e0e303d2556310ee549fbcf253d28de937bac3da13d6294262ac1",
        )
        self.assertEqual(
            manifest["go"]["url"],
            "https://dl.google.com/go/go1.25.12.linux-amd64.tar.gz",
        )
        self.assertEqual(manifest["application"]["go_proxy"], "https://goproxy.cn,direct")
        self.assertEqual(
            manifest["application"]["go_sumdb"],
            "sum.golang.org https://sum.golang.google.cn",
        )
        self.assertEqual(
            manifest["llama_cpp"]["revision"],
            "305ba519ab61cdff8044922cba2347826a04453f",
        )

    def test_safe_go_extraction_publishes_only_go_tree(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive = root / "go.tar.gz"
            payload = b"#!/bin/sh\necho go version go1.25.12 linux/amd64\n"
            with tarfile.open(archive, "w:gz") as bundle:
                info = tarfile.TarInfo("go/bin/go")
                info.size = len(payload)
                info.mode = 0o755
                bundle.addfile(info, io.BytesIO(payload))
            destination = root / "extract"
            with mock.patch.object(MODULE.os, "fsync") as fsync:
                MODULE.safe_extract_go(archive, destination)
            binary = destination / "go/bin/go"
            self.assertTrue(binary.is_file())
            self.assertEqual(binary.stat().st_mode & 0o777, 0o755)
            fsync.assert_not_called()

    def test_safe_go_extraction_rejects_path_traversal(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive = root / "unsafe.tar.gz"
            with tarfile.open(archive, "w:gz") as bundle:
                info = tarfile.TarInfo("../escape")
                info.size = 1
                bundle.addfile(info, io.BytesIO(b"x"))
            with self.assertRaisesRegex(MODULE.NativeToolchainError, "unsafe"):
                MODULE.safe_extract_go(archive, root / "extract")
            self.assertFalse((root / "escape").exists())

    def test_materialized_application_source_comes_from_selected_commit(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repository = root / "repository"
            repository.mkdir()
            subprocess.run(["git", "init", "--quiet"], cwd=repository, check=True)
            subprocess.run(
                ["git", "config", "user.email", "test@example.invalid"],
                cwd=repository,
                check=True,
            )
            subprocess.run(
                ["git", "config", "user.name", "SignalForge Test"],
                cwd=repository,
                check=True,
            )
            source = repository / "source.txt"
            source.write_text("authorized\n", encoding="utf-8")
            subprocess.run(["git", "add", "source.txt"], cwd=repository, check=True)
            subprocess.run(
                ["git", "commit", "--quiet", "-m", "authorized source"],
                cwd=repository,
                check=True,
            )
            selected_commit = subprocess.check_output(
                ["git", "rev-parse", "HEAD"],
                cwd=repository,
                text=True,
            ).strip()
            source.write_text("unselected working tree\n", encoding="utf-8")

            destination = root / "materialized"
            MODULE.materialize_source(repository, selected_commit, destination)

            self.assertEqual(
                (destination / "source.txt").read_text(encoding="utf-8"),
                "authorized\n",
            )

    def test_safe_source_extraction_rejects_symbolic_links(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive = root / "unsafe.tar"
            with tarfile.open(archive, "w") as bundle:
                info = tarfile.TarInfo("link")
                info.type = tarfile.SYMTYPE
                info.linkname = "/etc/passwd"
                bundle.addfile(info)
            with self.assertRaisesRegex(MODULE.NativeToolchainError, "unsupported"):
                MODULE.safe_extract_source(archive, root / "extract")

    def test_source_lock_hashes_match_the_native_manifest(self) -> None:
        manifest = json.loads(
            (ROOT / "deploy/radeon/native-toolchain-manifest.json").read_text()
        )
        MODULE.verify_source_locks(manifest)

    def test_native_environment_uses_manifest_bound_go_mirrors(self) -> None:
        manifest = json.loads(
            (ROOT / "deploy/radeon/native-toolchain-manifest.json").read_text()
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            go_binary = root / "go/bin/go"
            environment = MODULE.native_build_environment(go_binary, root, manifest)
        self.assertEqual(environment["GOPROXY"], "https://goproxy.cn,direct")
        self.assertEqual(
            environment["GOSUMDB"],
            "sum.golang.org https://sum.golang.google.cn",
        )

    def test_remove_directory_rejects_symlink_and_removes_stale_tree(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            stale = root / ".go-1.25.12.extract-interrupted"
            stale.mkdir()
            (stale / "partial").write_text("incomplete")
            MODULE.remove_directory(stale)
            self.assertFalse(stale.exists())

            target = root / "target"
            target.mkdir()
            link = root / "unsafe-link"
            link.symlink_to(target, target_is_directory=True)
            with self.assertRaisesRegex(MODULE.NativeToolchainError, "symbolic-link"):
                MODULE.remove_directory(link)


if __name__ == "__main__":
    unittest.main()
