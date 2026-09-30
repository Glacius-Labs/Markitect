#!/usr/bin/env python3
"""Run the repository's SHA-256-pinned Markitect release."""

from __future__ import annotations

import hashlib
import json
import os
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import zipfile
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Iterable


LOCK_NAME = "markitect.lock.yaml"
CACHE_PARTS = (".artifacts", "markitect")
MAX_ARCHIVE_FILES = 100_000
MAX_ARCHIVE_BYTES = 512 * 1024 * 1024
MAX_FILE_BYTES = 64 * 1024 * 1024
VERSION_RE = re.compile(
    r"^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"
    r"(?:-((?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)"
    r"(?:\.(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*))?"
    r"(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$"
)
FLAT_YAML_LINE_RE = re.compile(r'^([A-Za-z][A-Za-z0-9_]*): ("(?:[^"\\]|\\.)*")$')


@dataclass(frozen=True)
class Lock:
    version: str
    source: str
    sha256: str


@dataclass(frozen=True)
class ArchiveEntry:
    path: str
    info: zipfile.ZipInfo
    is_dir: bool
    mode: int


class BootstrapError(Exception):
    """A controlled bootstrap failure suitable for a concise CLI message."""


def parse_flat_quoted_yaml(data: bytes, expected_fields: tuple[str, ...], document: str) -> dict[str, str]:
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise BootstrapError(f"{document} must be UTF-8") from exc
    text = text.replace("\r\n", "\n")
    if text.endswith("\n"):
        text = text[:-1]
    if "\r" in text:
        raise BootstrapError(f"{document} must use canonical LF line endings")
    lines = text.split("\n")
    if len(lines) != len(expected_fields):
        raise BootstrapError(f"{document} must contain exactly {len(expected_fields)} fields")

    fields: dict[str, str] = {}
    for line in lines:
        match = FLAT_YAML_LINE_RE.fullmatch(line)
        if not match:
            raise BootstrapError(f"invalid {document} field syntax: {line!r}")
        key, scalar = match.groups()
        if key in fields:
            raise BootstrapError(f"duplicate {document} field: {key}")
        try:
            value = json.loads(scalar)
        except json.JSONDecodeError as exc:
            raise BootstrapError(f"invalid quoted value for {document} field {key}") from exc
        if not isinstance(value, str):
            raise BootstrapError(f"{document} field {key} must be a string")
        fields[key] = value

    if tuple(fields) != expected_fields:
        missing = [key for key in expected_fields if key not in fields]
        extra = [key for key in fields if key not in expected_fields]
        if missing or extra:
            raise BootstrapError(f"{document} fields must be exactly {', '.join(expected_fields)}")
        raise BootstrapError(f"{document} fields must use canonical order: {', '.join(expected_fields)}")
    return fields


def parse_lock(data: bytes) -> Lock:
    fields = parse_flat_quoted_yaml(data, ("version", "source", "sha256"), LOCK_NAME)

    version = fields["version"]
    if not VERSION_RE.fullmatch(version):
        raise BootstrapError(f"unsupported Markitect version: {version!r}")
    source = fields["source"]
    validate_repo_path(source)
    digest = fields["sha256"]
    if not re.fullmatch(r"[0-9a-f]{64}", digest):
        raise BootstrapError("lock sha256 must be 64 lowercase hexadecimal characters")
    return Lock(version=version, source=source, sha256=digest)


def validate_repo_path(value: str) -> PurePosixPath:
    if not value or "\x00" in value or "\\" in value or ":" in value:
        raise BootstrapError(f"unsafe repository-relative path: {value!r}")
    p = PurePosixPath(value)
    if p.is_absolute() or str(p) != value or any(part in ("", ".", "..") for part in value.split("/")):
        raise BootstrapError(f"unsafe repository-relative path: {value!r}")
    return p


def is_reparse_or_symlink(path: Path) -> bool:
    try:
        info = path.lstat()
    except FileNotFoundError:
        return False
    if stat.S_ISLNK(info.st_mode):
        return True
    attrs = getattr(info, "st_file_attributes", 0)
    reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0x400)
    return bool(attrs & reparse_flag)


def checked_repo_path(root: Path, relative: str, *, must_exist: bool = True) -> Path:
    rel = validate_repo_path(relative)
    current = root
    for part in rel.parts:
        current = current / part
        if is_reparse_or_symlink(current):
            raise BootstrapError(f"repository path traverses a symlink or reparse point: {relative}")
        if must_exist and not current.exists():
            raise BootstrapError(f"repository source does not exist: {relative}")
    return current


def file_sha256(path: Path) -> str:
    digest = hashlib.sha256()
    try:
        with path.open("rb") as stream:
            for chunk in iter(lambda: stream.read(1024 * 1024), b""):
                digest.update(chunk)
    except OSError as exc:
        raise BootstrapError(f"cannot read {path}: {exc}") from exc
    return digest.hexdigest()


def archive_entries(archive: zipfile.ZipFile) -> list[ArchiveEntry]:
    infos = archive.infolist()
    if len(infos) > MAX_ARCHIVE_FILES:
        raise BootstrapError("source archive exceeds entry-count limit")
    total = 0
    entries: list[ArchiveEntry] = []
    names: set[str] = set()
    folded: dict[str, str] = {}
    portable_paths: dict[str, tuple[str, bool]] = {}
    for info in infos:
        raw = info.filename
        is_dir = info.is_dir()
        name = raw[:-1] if is_dir and raw.endswith("/") else raw
        if not name:
            raise BootstrapError("source archive contains an empty path")
        validate_repo_path(name)
        if name in names:
            raise BootstrapError(f"source archive contains duplicate path: {name}")
        names.add(name)
        key = name.casefold()
        prior = folded.get(key)
        if prior is not None and prior != name:
            raise BootstrapError(f"source archive contains case-fold collision: {prior} and {name}")
        folded[key] = name

        unix_mode = (info.external_attr >> 16) & 0xFFFF if info.create_system == 3 else 0
        file_type = stat.S_IFMT(unix_mode)
        dos_attrs = info.external_attr & 0xFFFF
        if dos_attrs & 0x400:
            raise BootstrapError(f"source archive contains a reparse entry: {name}")
        if file_type == stat.S_IFLNK:
            raise BootstrapError(f"source archive contains a symlink: {name}")
        if file_type not in (0, stat.S_IFREG, stat.S_IFDIR):
            raise BootstrapError(f"source archive contains a special file: {name}")
        if is_dir != (file_type == stat.S_IFDIR) and file_type != 0:
            raise BootstrapError(f"source archive entry type mismatch: {name}")
        if not is_dir:
            if info.flag_bits & 0x1:
                raise BootstrapError(f"source archive contains an encrypted file: {name}")
            if info.file_size < 0 or info.file_size > MAX_FILE_BYTES:
                raise BootstrapError(f"source archive file exceeds size limit: {name}")
            total += info.file_size
            if total > MAX_ARCHIVE_BYTES:
                raise BootstrapError("source archive exceeds expanded-size limit")
        mode = 0o755 if unix_mode & 0o111 else 0o644
        entries.append(ArchiveEntry(path=name, info=info, is_dir=is_dir, mode=mode))
        parts = name.split("/")
        for i in range(len(parts)):
            prefix = "/".join(parts[: i + 1])
            prefix_is_dir = i < len(parts) - 1 or is_dir
            key = prefix.casefold()
            prior = portable_paths.get(key)
            if prior is not None and (prior[0] != prefix or prior[1] != prefix_is_dir):
                raise BootstrapError(f"source archive contains case-fold collision: {prior[0]} and {prefix}")
            portable_paths[key] = (prefix, prefix_is_dir)

    validate_archive_tree(entries)
    return entries


def validate_archive_tree(entries: Iterable[ArchiveEntry]) -> None:
    types: dict[str, bool] = {}
    for entry in entries:
        parts = entry.path.split("/")
        for i in range(len(parts)):
            prefix = "/".join(parts[: i + 1])
            is_dir = i < len(parts) - 1 or entry.is_dir
            previous = types.get(prefix)
            if previous is not None and previous != is_dir:
                raise BootstrapError(f"source archive file/directory conflict: {prefix}")
            types[prefix] = is_dir

    required = {
        "go.mod": False,
        "cmd": True,
        "cmd/markitect": True,
        "cmd/markitect/main.go": False,
        "internal": True,
    }
    for name, want_dir in required.items():
        if types.get(name) is not want_dir:
            raise BootstrapError(f"source archive is not a canonical Go module: missing {name}")


def open_verified_archive(path: Path, expected_sha256: str) -> tuple[object, zipfile.ZipFile, list[ArchiveEntry]]:
    try:
        stream = path.open("rb")
    except OSError as exc:
        raise BootstrapError(f"cannot open Markitect source archive: {exc}") from exc
    try:
        digest = hashlib.sha256()
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
        if digest.hexdigest() != expected_sha256:
            raise BootstrapError("Markitect source archive SHA-256 does not match the lock")
        stream.seek(0)
        archive = zipfile.ZipFile(stream, "r")
        entries = archive_entries(archive)
        return stream, archive, entries
    except (OSError, zipfile.BadZipFile) as exc:
        stream.close()
        if isinstance(exc, BootstrapError):
            raise
        raise BootstrapError(f"invalid Markitect source archive: {exc}") from exc
    except Exception:
        stream.close()
        raise


def expected_directories(entries: Iterable[ArchiveEntry]) -> set[str]:
    dirs: set[str] = set()
    for entry in entries:
        parts = entry.path.split("/")
        for i in range(1, len(parts)):
            dirs.add("/".join(parts[:i]))
        if entry.is_dir:
            dirs.add(entry.path)
    return dirs


def verify_expanded_source(source_dir: Path, archive: zipfile.ZipFile, entries: list[ArchiveEntry]) -> bool:
    expected_files = {entry.path: entry for entry in entries if not entry.is_dir}
    expected_dirs = expected_directories(entries)
    actual_files: set[str] = set()
    actual_dirs: set[str] = set()
    if not source_dir.exists():
        return False
    if is_reparse_or_symlink(source_dir) or not source_dir.is_dir():
        raise BootstrapError(f"unsafe expanded source directory: {source_dir}")
    for current, dirnames, filenames in os.walk(source_dir, topdown=True, followlinks=False):
        current_path = Path(current)
        rel_dir = current_path.relative_to(source_dir).as_posix()
        for name in list(dirnames):
            child = current_path / name
            if is_reparse_or_symlink(child) or not child.is_dir():
                raise BootstrapError(f"unsafe entry in expanded source: {child}")
            rel = name if rel_dir == "." else f"{rel_dir}/{name}"
            actual_dirs.add(rel)
        for name in filenames:
            child = current_path / name
            if is_reparse_or_symlink(child) or not child.is_file():
                raise BootstrapError(f"unsafe entry in expanded source: {child}")
            rel = name if rel_dir == "." else f"{rel_dir}/{name}"
            actual_files.add(rel)
    if actual_files != set(expected_files) or actual_dirs != expected_dirs:
        return False
    for rel, entry in expected_files.items():
        target = source_dir.joinpath(*rel.split("/"))
        if target.stat().st_size != entry.info.file_size:
            return False
        with archive.open(entry.info, "r") as zipped, target.open("rb") as expanded:
            while True:
                expected = zipped.read(1024 * 1024)
                if not expected:
                    break
                actual = expanded.read(len(expected))
                if actual != expected:
                    return False
            if expanded.read(1):
                return False
        if os.name != "nt" and target.stat().st_mode & 0o111 != entry.mode & 0o111:
            return False
    return True


def create_cache_path(root: Path, digest: str) -> Path:
    current = root
    for part in (*CACHE_PARTS, digest):
        current = current / part
        if is_reparse_or_symlink(current):
            raise BootstrapError(f"Markitect cache path traverses a symlink or reparse point: {current}")
        if current.exists():
            if not current.is_dir():
                raise BootstrapError(f"Markitect cache component is not a directory: {current}")
        else:
            try:
                current.mkdir()
            except OSError as exc:
                raise BootstrapError(f"cannot create Markitect cache directory {current}: {exc}") from exc
    return current


def validate_directory_path(path: Path, *, create: bool) -> Path:
    """Validate existing ancestors and optionally create a real directory path."""
    absolute = Path(os.path.abspath(path))
    anchor = Path(absolute.anchor)
    current = anchor
    for part in absolute.parts[1:]:
        current = current / part
        if is_reparse_or_symlink(current):
            raise BootstrapError(f"build cache path traverses a symlink or reparse point: {current}")
        if current.exists():
            if not current.is_dir():
                raise BootstrapError(f"build cache path component is not a directory: {current}")
            continue
        if create:
            try:
                current.mkdir()
            except OSError as exc:
                raise BootstrapError(f"cannot create build cache directory {current}: {exc}") from exc
            if is_reparse_or_symlink(current) or not current.is_dir():
                raise BootstrapError(f"unsafe build cache directory after creation: {current}")
    return absolute


def clear_cache(cache_dir: Path) -> None:
    if not cache_dir.exists():
        cache_dir.mkdir()
        return
    if is_reparse_or_symlink(cache_dir) or not cache_dir.is_dir():
        raise BootstrapError(f"unsafe Markitect cache directory: {cache_dir}")
    for current, dirs, files in os.walk(cache_dir, topdown=True, followlinks=False):
        for name in dirs + files:
            child = Path(current) / name
            if is_reparse_or_symlink(child):
                raise BootstrapError(f"refusing to clear cache containing a reparse point: {child}")
    shutil.rmtree(cache_dir)
    cache_dir.mkdir()


def extract_archive(source_dir: Path, archive: zipfile.ZipFile, entries: list[ArchiveEntry]) -> None:
    source_dir.mkdir()
    for entry in entries:
        target = source_dir.joinpath(*entry.path.split("/"))
        if entry.is_dir:
            target.mkdir(parents=True, exist_ok=True)
            continue
        target.parent.mkdir(parents=True, exist_ok=True)
        with archive.open(entry.info, "r") as zipped, target.open("xb") as expanded:
            shutil.copyfileobj(zipped, expanded, length=1024 * 1024)
        target.chmod(entry.mode)
    if not verify_expanded_source(source_dir, archive, entries):
        raise BootstrapError("expanded Markitect source did not match its verified archive")


def executable_name() -> str:
    return "markitect.exe" if os.name == "nt" else "markitect"


def read_build_stamp(stamp_path: Path) -> dict[str, str] | None:
    if is_reparse_or_symlink(stamp_path):
        raise BootstrapError(f"unsafe Markitect build stamp: {stamp_path}")
    if not stamp_path.exists():
        return None
    if not stamp_path.is_file():
        raise BootstrapError(f"unsafe Markitect build stamp: {stamp_path}")
    try:
        fields = parse_flat_quoted_yaml(
            stamp_path.read_bytes(),
            ("version", "source_sha256", "executable_sha256"),
            "Markitect build-stamp.yaml",
        )
    except (OSError, BootstrapError):
        return None
    if not re.fullmatch(r"[0-9a-f]{64}", fields["source_sha256"]):
        return None
    if not re.fullmatch(r"[0-9a-f]{64}", fields["executable_sha256"]):
        return None
    return fields


def cached_executable(cache_dir: Path, lock: Lock) -> Path | None:
    executable = cache_dir / executable_name()
    stamp_path = cache_dir / "build-stamp.yaml"
    stamp = read_build_stamp(stamp_path)
    if is_reparse_or_symlink(executable):
        raise BootstrapError(f"unsafe cached Markitect executable: {executable}")
    if stamp is None or not executable.exists():
        return None
    if not executable.is_file():
        raise BootstrapError(f"unsafe cached Markitect executable: {executable}")
    if stamp["version"] != lock.version or stamp["source_sha256"] != lock.sha256:
        return None
    if file_sha256(executable) != stamp["executable_sha256"]:
        return None
    return executable


def write_stamp(cache_dir: Path, lock: Lock, executable: Path) -> None:
    stamp = {
        "version": lock.version,
        "source_sha256": lock.sha256,
        "executable_sha256": file_sha256(executable),
    }
    target = cache_dir / "build-stamp.yaml"
    fd, temp_name = tempfile.mkstemp(prefix="build-stamp-", dir=cache_dir)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="\n") as stream:
            for key in ("version", "source_sha256", "executable_sha256"):
                stream.write(f"{key}: {json.dumps(stamp[key])}\n")
        os.replace(temp_name, target)
    finally:
        try:
            os.unlink(temp_name)
        except FileNotFoundError:
            pass


def build_markitect(source_dir: Path, cache_dir: Path, lock: Lock, go: str) -> Path:
    executable = cache_dir / executable_name()
    cache_dir = validate_directory_path(cache_dir, create=False)
    source_dir = validate_directory_path(source_dir, create=False)
    if not cache_dir.is_dir() or not source_dir.is_dir():
        raise BootstrapError("Markitect source and cache directories must exist before building")
    build_env = os.environ.copy()
    shared_cache_root = validate_directory_path(cache_dir.parent, create=True)
    for key, local_name in (("GOCACHE", "go-build"), ("GOTMPDIR", "go-tmp")):
        override = build_env.get(key)
        if override:
            override_path = Path(override)
            if not override_path.is_absolute():
                override_path = source_dir / override_path
            validate_directory_path(override_path, create=False)
        else:
            local_path = validate_directory_path(shared_cache_root / local_name, create=True)
            build_env[key] = str(local_path)

    fd, temporary_name = tempfile.mkstemp(prefix="markitect-build-", dir=cache_dir)
    os.close(fd)
    os.unlink(temporary_name)
    try:
        command = [
            go,
            "build",
            "-buildvcs=false",
            "-trimpath",
            "-ldflags",
            f"-X main.version={lock.version}",
            "-o",
            temporary_name,
            "./cmd/markitect",
        ]
        result = subprocess.run(command, cwd=source_dir, check=False, env=build_env)
        if result.returncode != 0:
            raise BootstrapError(f"Go failed to build pinned Markitect source (exit {result.returncode})")
        built = Path(temporary_name)
        if not built.is_file() or is_reparse_or_symlink(built):
            raise BootstrapError("Go build did not produce a regular executable")
        os.replace(built, executable)
        if os.name != "nt":
            executable.chmod(executable.stat().st_mode | 0o111)
        write_stamp(cache_dir, lock, executable)
        return executable
    finally:
        try:
            os.unlink(temporary_name)
        except FileNotFoundError:
            pass


def command_args(args: list[str], repo_root: Path) -> list[str]:
    if not args or args[0] == "version":
        return args
    before_separator = args[1:]
    for i, arg in enumerate(before_separator):
        if arg == "--":
            before_separator = before_separator[:i]
            break
        if arg == "--repo" or arg.startswith("--repo="):
            return args
    return [args[0], "--repo", str(repo_root), *args[1:]]


def discover_repo_root(script_path: Path) -> Path:
    for candidate in (script_path.resolve().parent, *script_path.resolve().parent.parents):
        lock = candidate / LOCK_NAME
        if lock.exists():
            if is_reparse_or_symlink(lock) or not lock.is_file():
                raise BootstrapError(f"unsafe lock file: {lock}")
            return candidate
    raise BootstrapError(f"could not find {LOCK_NAME} in the script's parent directories")


def prepare(root: Path) -> Path:
    lock_path = checked_repo_path(root, LOCK_NAME)
    if not lock_path.is_file():
        raise BootstrapError(f"lock path is not a regular file: {lock_path}")
    lock = parse_lock(lock_path.read_bytes())
    archive_path = checked_repo_path(root, lock.source)
    if not archive_path.is_file():
        raise BootstrapError(f"Markitect source is not a regular file: {lock.source}")
    go = shutil.which("go")
    if go is None:
        raise BootstrapError("Go is required to run the pinned Markitect release")

    cache_dir = create_cache_path(root, lock.sha256)
    source_dir = cache_dir / "source"
    with _verified_archive_context(archive_path, lock.sha256) as (archive, entries):
        if not verify_expanded_source(source_dir, archive, entries):
            clear_cache(cache_dir)
            source_dir = cache_dir / "source"
            extract_archive(source_dir, archive, entries)
        executable = cached_executable(cache_dir, lock)
        if executable is not None:
            return executable
        return build_markitect(source_dir, cache_dir, lock, go)


class _verified_archive_context:
    def __init__(self, path: Path, expected_sha256: str):
        self.path = path
        self.expected_sha256 = expected_sha256
        self.stream = None
        self.archive = None
        self.entries = None

    def __enter__(self):
        self.stream, self.archive, self.entries = open_verified_archive(self.path, self.expected_sha256)
        return self.archive, self.entries

    def __exit__(self, exc_type, exc, traceback):
        if self.archive is not None:
            self.archive.close()
        if self.stream is not None:
            self.stream.close()


def run(args: list[str], script_path: Path | None = None) -> int:
    script_path = script_path or Path(__file__)
    try:
        root = discover_repo_root(script_path)
        executable = prepare(root)
        forwarded = command_args(args, root)
        if not forwarded:
            return subprocess.run([str(executable)], cwd=root, check=False).returncode
        return subprocess.run([str(executable), *forwarded], cwd=root, check=False).returncode
    except BootstrapError as exc:
        print(f"markitect bootstrap: {exc}", file=sys.stderr)
        return 2
    except (OSError, zipfile.BadZipFile, RuntimeError, NotImplementedError) as exc:
        print(f"markitect bootstrap: {exc}", file=sys.stderr)
        return 2


def main(argv: list[str] | None = None) -> int:
    return run(list(sys.argv[1:] if argv is None else argv))


if __name__ == "__main__":
    raise SystemExit(main())
