from __future__ import annotations

import hashlib
import importlib.util
import json
import stat
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path


SCRIPT = Path(__file__).with_name("run-markitect.py")
SPEC = importlib.util.spec_from_file_location("markitect_bootstrap", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
BOOTSTRAP = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = BOOTSTRAP
SPEC.loader.exec_module(BOOTSTRAP)


def lock_text(version: str = "0.1.0-dev", source: str = "tools/markitect/release.zip", digest: str = "0" * 64) -> bytes:
    return (
        f'version: {json.dumps(version)}\n'
        f'source: {json.dumps(source)}\n'
        f'sha256: {json.dumps(digest)}\n'
    ).encode("utf-8")


def module_archive(path: Path, extra: list[tuple[str, bytes, int]] | None = None) -> str:
    files = [
        ("go.mod", b"module markitect\n\ngo 1.24.0\n", stat.S_IFREG | 0o644),
        ("go.sum", b"", stat.S_IFREG | 0o644),
        ("cmd/markitect/main.go", b"package main\n", stat.S_IFREG | 0o644),
        ("internal/app/app.go", b"package app\n", stat.S_IFREG | 0o644),
    ]
    files.extend(extra or [])
    with zipfile.ZipFile(path, "w") as archive:
        for name, data, mode in files:
            info = zipfile.ZipInfo(name)
            info.create_system = 3
            info.external_attr = mode << 16
            archive.writestr(info, data)
    return hashlib.sha256(path.read_bytes()).hexdigest()


class ParseLockTests(unittest.TestCase):
    def test_accepts_canonical_lock(self) -> None:
        lock = BOOTSTRAP.parse_lock(lock_text())
        self.assertEqual((lock.version, lock.source, lock.sha256), ("0.1.0-dev", "tools/markitect/release.zip", "0" * 64))

    def test_rejects_duplicate_unknown_trailing_and_noncanonical_fields(self) -> None:
        samples = [
            b'version: "0.1.0"\nversion: "0.1.0"\nsha256: "' + b"0" * 64 + b'"\n',
            lock_text().replace(b"source:", b"unknown:", 1),
            lock_text() + b"unexpected: true\n",
            lock_text().replace(b'"0.1.0-dev"', b"'0.1.0-dev'"),
            b'source: "tools/markitect/release.zip"\nversion: "0.1.0-dev"\nsha256: "' + b"0" * 64 + b'"\n',
        ]
        for sample in samples:
            with self.subTest(sample=sample):
                with self.assertRaises(BOOTSTRAP.BootstrapError):
                    BOOTSTRAP.parse_lock(sample)

    def test_rejects_unsupported_version_and_unsafe_source(self) -> None:
        for sample in (
            lock_text(version="latest"),
            lock_text(version="0.1.0-"),
            lock_text(source="../release.zip"),
            lock_text(source="C:/release.zip"),
            lock_text(source="tools\\release.zip"),
        ):
            with self.subTest(sample=sample):
                with self.assertRaises(BOOTSTRAP.BootstrapError):
                    BOOTSTRAP.parse_lock(sample)


class ArchiveTests(unittest.TestCase):
    def test_verified_archive_has_canonical_module_layout(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive_path = Path(directory) / "release.zip"
            digest = module_archive(archive_path)
            stream, archive, entries = BOOTSTRAP.open_verified_archive(archive_path, digest)
            try:
                self.assertIn("cmd/markitect/main.go", {entry.path for entry in entries})
                self.assertIn("internal/app/app.go", {entry.path for entry in entries})
            finally:
                archive.close()
                stream.close()

    def test_rejects_wrong_digest_before_use(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive_path = Path(directory) / "release.zip"
            module_archive(archive_path)
            with self.assertRaisesRegex(BOOTSTRAP.BootstrapError, "SHA-256"):
                BOOTSTRAP.open_verified_archive(archive_path, "f" * 64)

    def test_expanded_source_is_compared_to_the_verified_archive(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive_path = Path(directory) / "release.zip"
            digest = module_archive(archive_path)
            stream, archive, entries = BOOTSTRAP.open_verified_archive(archive_path, digest)
            source_dir = Path(directory) / "source"
            try:
                BOOTSTRAP.extract_archive(source_dir, archive, entries)
                self.assertTrue(BOOTSTRAP.verify_expanded_source(source_dir, archive, entries))
                (source_dir / "cmd/markitect/main.go").write_bytes(b"replaced")
                self.assertFalse(BOOTSTRAP.verify_expanded_source(source_dir, archive, entries))
            finally:
                archive.close()
                stream.close()

    def test_rejects_unsafe_archive_paths_and_symlinks(self) -> None:
        cases = [
            [("../escape", b"x", stat.S_IFREG | 0o644)],
            [("link", b"target", stat.S_IFLNK | 0o777)],
            [("CMD/extra.go", b"x", stat.S_IFREG | 0o644)],
        ]
        for extra in cases:
            with tempfile.TemporaryDirectory() as directory:
                archive_path = Path(directory) / "release.zip"
                digest = module_archive(archive_path, extra)
                with self.subTest(extra=extra):
                    with self.assertRaises(BOOTSTRAP.BootstrapError):
                        BOOTSTRAP.open_verified_archive(archive_path, digest)

    def test_rejects_non_module_archive(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive_path = Path(directory) / "release.zip"
            with zipfile.ZipFile(archive_path, "w") as archive:
                archive.writestr("go.mod", "module markitect\n")
            digest = hashlib.sha256(archive_path.read_bytes()).hexdigest()
            with self.assertRaisesRegex(BOOTSTRAP.BootstrapError, "canonical Go module"):
                BOOTSTRAP.open_verified_archive(archive_path, digest)


class InvocationTests(unittest.TestCase):
    def test_build_stamp_uses_strict_yaml_and_detects_replaced_binary(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            cache_dir = Path(directory)
            executable = cache_dir / BOOTSTRAP.executable_name()
            executable.write_bytes(b"original executable")
            lock = BOOTSTRAP.parse_lock(lock_text(digest="a" * 64))

            BOOTSTRAP.write_stamp(cache_dir, lock, executable)
            stamp_path = cache_dir / "build-stamp.yaml"
            self.assertTrue(stamp_path.is_file())
            self.assertFalse((cache_dir / "build-stamp.json").exists())
            self.assertEqual(BOOTSTRAP.read_build_stamp(stamp_path)["source_sha256"], lock.sha256)
            self.assertIsNotNone(BOOTSTRAP.cached_executable(cache_dir, lock))

            executable.write_bytes(b"replaced executable")
            self.assertIsNone(BOOTSTRAP.cached_executable(cache_dir, lock))

            stamp_path.write_text(
                'version: "0.1.0-dev"\nversion: "0.1.0-dev"\n'
                f'source_sha256: "{lock.sha256}"\n'
                f'executable_sha256: "{"0" * 64}"\n',
                encoding="utf-8",
            )
            self.assertIsNone(BOOTSTRAP.read_build_stamp(stamp_path))

    def test_injects_repo_without_overriding_explicit_repo(self) -> None:
        root = Path("C:/repo")
        self.assertEqual(BOOTSTRAP.command_args(["check"], root), ["check", "--repo", str(root)])
        self.assertEqual(BOOTSTRAP.command_args(["check", "--repo", "other"], root), ["check", "--repo", "other"])
        self.assertEqual(BOOTSTRAP.command_args(["inventory", "--repo=other"], root), ["inventory", "--repo=other"])
        self.assertEqual(BOOTSTRAP.command_args(["check", "--", "--repo=other"], root), ["check", "--repo", str(root), "--", "--repo=other"])
        self.assertEqual(BOOTSTRAP.command_args(["version"], root), ["version"])


if __name__ == "__main__":
    unittest.main()
