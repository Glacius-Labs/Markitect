"""Scoped file workspace for the opt-in Codex native-work execution mode."""

from __future__ import annotations

import base64
import hashlib
import json
import os
import re
import stat
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Any


MAX_FILES = 128
MAX_BYTES = 8 * 1024 * 1024
MAX_INPUT_FILES = 256
MAX_INPUT_BYTES = 24 * 1024 * 1024
WORKSPACE_API = "markitect.example.org/native-workspace/v1"
MANAGER_CONTEXT_PATH = ".markitect/manager-context.json"
CONTROL_PATHS = {".git", ".markitect/manager-context.json", "codex-response.json", "codex-response.schema.json"}


class NativeWorkError(Exception):
    """A scoped native workspace could not be safely prepared or harvested."""


class SafeDeltaRejected(NativeWorkError):
    """The workspace scan completed, but its observed delta cannot be accepted."""

    def __init__(self, message: str, native_work: dict[str, Any]) -> None:
        super().__init__(message)
        self.native_work = native_work


@dataclass(frozen=True)
class FileRecord:
    path: str
    mode: str
    content: bytes
    digest: str


@dataclass
class PreparedWorkspace:
    root: Path
    base_digest: str
    initial: dict[str, FileRecord]
    instructions: frozenset[str]
    manager_id: str
    allowed: tuple[str, ...]
    excluded: tuple[str, ...]
    foreign: tuple[dict[str, Any], ...]


def _digest(data: bytes) -> str:
    return "sha256:" + hashlib.sha256(data).hexdigest()


def _portable_path(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value or "\\" in value or "\x00" in value:
        raise NativeWorkError(f"{label} is not a safe relative path")
    path = PurePosixPath(value)
    if path.is_absolute() or any(part in {"", ".", ".."} for part in value.split("/")):
        raise NativeWorkError(f"{label} is not a safe relative path")
    if re.match(r"^[A-Za-z]:", value) or any(ord(char) < 32 for char in value):
        raise NativeWorkError(f"{label} is not a safe relative path")
    return path.as_posix()


def _mode(value: Any, label: str) -> str:
    if not isinstance(value, str) or value not in {"0600", "0644", "0755"}:
        raise NativeWorkError(f"{label} has an unsupported file mode")
    return value


def _decode_artifact(item: Any, label: str) -> FileRecord:
    if not isinstance(item, dict) or set(item) != {"path", "mode", "digest", "content"}:
        raise NativeWorkError(f"{label} has an unsupported shape")
    path = _portable_path(item["path"], f"{label} path")
    mode_value = item["mode"]
    digest = item["digest"]
    if label == "native instruction":
        mode = {"100644": "0644", "100755": "0755"}.get(mode_value, "")
        if not isinstance(digest, str) or re.fullmatch(r"[0-9a-f]{64}", digest) is None:
            raise NativeWorkError(f"{label} digest is malformed")
        digest = "sha256:" + digest
    else:
        mode = _mode(mode_value, label)
    if not mode:
        raise NativeWorkError(f"{label} has an unsupported file mode")
    if not isinstance(digest, str) or re.fullmatch(r"sha256:[0-9a-f]{64}", digest) is None:
        raise NativeWorkError(f"{label} digest is malformed")
    try:
        content = base64.b64decode(item["content"], validate=True)
    except (ValueError, TypeError) as exc:
        raise NativeWorkError(f"{label} bytes are malformed") from exc
    if _digest(content) != digest:
        raise NativeWorkError(f"{label} digest does not match supplied bytes")
    if label == "native instruction":
        try:
            content.decode("utf-8", errors="strict")
        except UnicodeDecodeError as exc:
            raise NativeWorkError(f"{label} is not valid UTF-8") from exc
    return FileRecord(path, mode, content, digest)


def _check_unique_paths(records: list[FileRecord]) -> None:
    exact: set[str] = set()
    folded: dict[str, str] = {}
    for record in records:
        if record.path in exact:
            raise NativeWorkError(f"duplicate native workspace path: {record.path}")
        exact.add(record.path)
        key = record.path.casefold()
        prior = folded.get(key)
        if prior is not None and prior != record.path:
            raise NativeWorkError(f"case-alias collision in native workspace: {prior} and {record.path}")
        folded[key] = record.path


def _control_path(path: str) -> bool:
    lower = path.casefold()
    return (
        lower == ".git" or lower.startswith(".git/")
        or lower == ".markitect" or lower.startswith(".markitect/")
        or lower in {item.casefold() for item in CONTROL_PATHS}
    )


def _manifest_digest(files: dict[str, FileRecord]) -> str:
    manifest = [
        {"path": item.path, "mode": item.mode, "digest": item.digest}
        for item in sorted(files.values(), key=lambda entry: entry.path)
    ]
    encoded = json.dumps(manifest, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return _digest(encoded)


def prepare(invocation: dict[str, Any], root: Path) -> PreparedWorkspace:
    """Create a fresh candidate tree from verified request artifacts/instructions."""
    request = invocation["request"]
    context = request["context"]
    if request.get("role") != "executor" or not isinstance(context, dict) or context.get("kind") != "projectrun-task/v1":
        raise NativeWorkError("native-work is supported only for projectrun Manager executor tasks")
    if context.get("phase") not in {"work", "integrate", "repair"}:
        raise NativeWorkError("native-work requires a work, integrate, or repair phase")
    workspace = context.get("nativeWorkspace")
    if not isinstance(workspace, dict) or set(workspace) != {"apiVersion", "instructions"}:
        raise NativeWorkError("native-work requires a valid Host nativeWorkspace manifest")
    if workspace.get("apiVersion") != WORKSPACE_API or not isinstance(workspace.get("instructions"), list) or len(workspace["instructions"]) > 128:
        raise NativeWorkError("nativeWorkspace manifest is unsupported or malformed")

    records = [_decode_artifact(item, "request artifact") for item in request["artifacts"]]
    instruction_records = [_decode_artifact(item, "native instruction") for item in workspace["instructions"]]
    if any(_control_path(item.path) for item in records + instruction_records):
        raise NativeWorkError("native workspace input collides with a protected control-plane path")
    _check_unique_paths(records + instruction_records)
    if len(records) + len(instruction_records) > MAX_INPUT_FILES:
        raise NativeWorkError("native workspace inputs exceed the 256-file bound")
    if sum(len(item.content) for item in records + instruction_records) > MAX_INPUT_BYTES:
        raise NativeWorkError("native workspace inputs exceed the 24 MiB bound")
    if root.exists() or root.is_symlink():
        if root.is_symlink() or not root.is_dir():
            raise NativeWorkError("native candidate workspace must be a real directory")
        if any(root.iterdir()):
            raise NativeWorkError("native candidate workspace must start empty")
    root.mkdir(parents=True, exist_ok=True)

    initial: dict[str, FileRecord] = {}
    for record in records + instruction_records:
        target = root.joinpath(*record.path.split("/"))
        _write_new_file(root, target, record.path, record.content, record.mode)
        initial[record.path] = record

    manager_id = context.get("managerId")
    if not isinstance(manager_id, str) or not manager_id:
        raise NativeWorkError("native-work Manager identity is missing")
    manager_context = {key: value for key, value in context.items() if key not in {"nativeWorkspace", "responseSchema"}}
    context_bytes = json.dumps(manager_context, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode("utf-8")
    context_record = FileRecord(MANAGER_CONTEXT_PATH, "0644", context_bytes, _digest(context_bytes))
    _write_new_file(root, root / MANAGER_CONTEXT_PATH, MANAGER_CONTEXT_PATH, context_bytes, "0644")
    initial[MANAGER_CONTEXT_PATH] = context_record

    allowed = _scopes(context.get("allowedWritePaths"), "allowedWritePaths")
    excluded = _scopes(context.get("excludedWritePaths", []), "excludedWritePaths")
    foreign_value = context.get("foreignOwnership", [])
    if not isinstance(foreign_value, list) or any(not isinstance(item, dict) or not isinstance(item.get("path"), str) or not isinstance(item.get("owner"), str) for item in foreign_value):
        raise NativeWorkError("native-work foreignOwnership metadata is malformed")
    foreign = []
    for item in foreign_value:
        normalized = dict(item)
        normalized["path"] = _portable_path(item["path"].rstrip("/"), "foreign ownership path")
        foreign.append(normalized)
    return PreparedWorkspace(
        root=root,
        base_digest=_manifest_digest(initial),
        initial=initial,
        instructions=frozenset(item.path for item in instruction_records),
        manager_id=manager_id,
        allowed=allowed,
        excluded=excluded,
        foreign=tuple(foreign),
    )


def _scopes(value: Any, label: str) -> tuple[str, ...]:
    if not isinstance(value, list) or len(value) > 1024:
        raise NativeWorkError(f"native-work {label} is malformed")
    result: list[str] = []
    for scope in value:
        if isinstance(scope, str):
            raw = scope
        elif isinstance(scope, dict) and isinstance(scope.get("path"), str):
            raw = scope["path"]
        else:
            raise NativeWorkError(f"native-work {label} is malformed")
        result.append(_portable_path(raw.rstrip("/"), f"native-work {label} entry"))
    return tuple(result)


def _write_new_file(root: Path, target: Path, relative: str, content: bytes, mode: str) -> None:
    current = root
    for part in relative.split("/")[:-1]:
        current = current / part
        if current.exists() and (current.is_symlink() or not current.is_dir()):
            raise NativeWorkError(f"native workspace path has a directory collision: {relative}")
        current.mkdir(exist_ok=True)
    if target.exists() or target.is_symlink():
        raise NativeWorkError(f"native workspace path collision: {relative}")
    flags = os.O_WRONLY | os.O_CREAT | os.O_EXCL
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    fd = os.open(target, flags, int(mode, 8))
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(content)
    except BaseException:
        try:
            os.close(fd)
        except OSError:
            pass
        raise
    os.chmod(target, int(mode, 8))


def _reparse(st: os.stat_result) -> bool:
    return bool(getattr(st, "st_file_attributes", 0) & getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0x400))


def _posix_mode(mode_bits: int, path: str) -> str:
    if mode_bits not in {0o600, 0o644, 0o755}:
        raise NativeWorkError(f"native-work candidate has unsupported POSIX mode {mode_bits:04o}: {path}")
    return f"{mode_bits:04o}"


def _read_candidate(workspace: PreparedWorkspace) -> dict[str, FileRecord]:
    files: dict[str, FileRecord] = {}
    folded: dict[str, str] = {}
    total = 0
    for directory, dirnames, filenames in os.walk(workspace.root, topdown=True, followlinks=False):
        base = Path(directory)
        for name in list(dirnames):
            full = base / name
            st = full.lstat()
            if stat.S_ISLNK(st.st_mode) or _reparse(st) or not stat.S_ISDIR(st.st_mode):
                raise NativeWorkError(f"native-work candidate contains a reparse point or non-directory: {full.relative_to(workspace.root).as_posix()}")
        for name in filenames:
            full = base / name
            relative = full.relative_to(workspace.root).as_posix()
            normalized = _portable_path(relative, "candidate path")
            if normalized.casefold() in folded and folded[normalized.casefold()] != normalized:
                raise NativeWorkError(f"case-alias collision in candidate: {folded[normalized.casefold()]} and {normalized}")
            folded[normalized.casefold()] = normalized
            st = full.lstat()
            if stat.S_ISLNK(st.st_mode) or _reparse(st) or not stat.S_ISREG(st.st_mode):
                raise NativeWorkError(f"native-work candidate contains a link or non-regular file: {normalized}")
            if st.st_nlink != 1:
                raise NativeWorkError(f"native-work candidate contains a hard link: {normalized}")
            raw = full.read_bytes()
            total += st.st_size
            if st.st_size > MAX_BYTES or total > MAX_INPUT_BYTES + MAX_BYTES:
                raise NativeWorkError("native-work candidate exceeds the scan size bound")
            mode_bits = stat.S_IMODE(st.st_mode)
            original = workspace.initial.get(normalized)
            if os.name == "nt":
                # Windows chmod/stat does not preserve POSIX execute bits.
                # Keep Host-supplied modes for existing files. New files use
                # the documented non-executable mode; Windows has no reliable
                # portable observation for the executable bit.
                mode = original.mode if original is not None else "0644"
            else:
                # 0600 is a deliberate supported private-file mode from the
                # request/candidate contract, not a normalization fallback.
                # Other modes (such as 0664 under umask 002) are kept as
                # observed so harvest rejects them as an unsafe delta.
                try:
                    mode = _posix_mode(mode_bits, normalized)
                except NativeWorkError:
                    mode = f"{mode_bits:04o}"
            files[normalized] = FileRecord(normalized, mode, raw, _digest(raw))
            if len(files) > MAX_INPUT_FILES + MAX_FILES + 1:
                raise NativeWorkError("native-work candidate exceeds the total scan file bound")
    return files


def _matches(path: str, scope: str) -> bool:
    path_key = path.casefold()
    scope_key = scope.rstrip("/").casefold()
    return path_key == scope_key or path_key.startswith(scope_key + "/")


def _authorize(workspace: PreparedWorkspace, path: str) -> None:
    if _control_path(path):
        raise NativeWorkError(f"native-work candidate attempted a control-plane write: {path}")
    if any(_matches(path, scope) for scope in workspace.excluded):
        raise NativeWorkError(f"native-work candidate write is explicitly excluded: {path}")
    matching_foreign = [item for item in workspace.foreign if _matches(path, item["path"])]
    if matching_foreign:
        owner = max(matching_foreign, key=lambda item: len(item["path"].rstrip("/"))).get("owner")
        if owner != workspace.manager_id:
            raise NativeWorkError(f"native-work candidate write belongs to foreign Manager {owner}: {path}")
    if not any(_matches(path, scope) for scope in workspace.allowed):
        raise NativeWorkError(f"native-work candidate write is outside allowedWritePaths: {path}")


def harvest(workspace: PreparedWorkspace, declared_files: Any, tool_calls: int) -> tuple[list[dict[str, str]], str, str, list[str]]:
    """Validate final bytes and require the provider's JSON claims to match disk."""
    final = _read_candidate(workspace)
    changed: list[FileRecord] = []
    deleted: list[str] = []
    reasons: list[str] = []
    for path, original in workspace.initial.items():
        current = final.get(path)
        if current is None:
            deleted.append(path)
            reasons.append(f"native-work candidate deleted an existing input: {path}")
            continue
        if path in workspace.instructions or path == MANAGER_CONTEXT_PATH:
            if current.content != original.content or current.mode != original.mode:
                reasons.append(f"native-work candidate modified immutable instructions/context: {path}")
        elif current.content != original.content or current.mode != original.mode:
            try:
                _authorize(workspace, path)
            except NativeWorkError as exc:
                reasons.append(str(exc))

    for path, current in final.items():
        original = workspace.initial.get(path)
        if original is None or current.content != original.content or current.mode != original.mode:
            if current.mode not in {"0644", "0755"}:
                reasons.append(f"native-work candidate uses unsupported repository mode {current.mode}: {path}")
            try:
                _authorize(workspace, path)
            except NativeWorkError as exc:
                reasons.append(str(exc))
            try:
                current.content.decode("utf-8", errors="strict")
            except UnicodeDecodeError as exc:
                reasons.append(f"native-work candidate produced unsupported non-UTF-8 bytes: {path}")
            changed.append(current)
    if sum(len(item.content) for item in changed) > MAX_BYTES or len(changed) > MAX_FILES:
        reasons.append("native-work candidate delta exceeds its file or byte bound")
    changed.sort(key=lambda item: item.path)
    changed_paths = sorted({item.path for item in changed} | set(deleted))
    final_digest = _manifest_digest(final)
    delta_payload = json.dumps(
        [{"path": item.path, "mode": item.mode, "digest": item.digest} for item in changed]
        + [{"path": path, "deleted": True} for path in sorted(deleted)],
        ensure_ascii=False, sort_keys=True, separators=(",", ":"),
    ).encode("utf-8")
    delta_digest = _digest(delta_payload)
    native_work = receipt(workspace.base_digest, final_digest, delta_digest, changed_paths, tool_calls)
    if len(changed_paths) > MAX_FILES:
        reasons.append("native-work candidate delta exceeds the 128-file bound")
    expected: list[dict[str, str]] = []
    for item in changed:
        try:
            content = item.content.decode("utf-8", errors="strict")
        except UnicodeDecodeError:
            continue
        expected.append({"path": item.path, "mode": item.mode, "content": content})
    if reasons:
        raise SafeDeltaRejected("; ".join(dict.fromkeys(reasons)), native_work)
    if not isinstance(declared_files, list):
        raise SafeDeltaRejected("native-work provider candidateFiles is malformed", native_work)
    declared: dict[str, tuple[str, str]] = {}
    for item in declared_files:
        if not isinstance(item, dict) or set(item) != {"path", "mode", "content"}:
            raise SafeDeltaRejected("native-work provider candidateFiles is malformed", native_work)
        path = _portable_path(item["path"], "provider candidate path")
        if path in declared:
            raise SafeDeltaRejected("native-work provider candidateFiles contains a duplicate path", native_work)
        mode = _mode(item["mode"], "provider candidate")
        if not isinstance(item["content"], str):
            raise SafeDeltaRejected("native-work provider candidate content is malformed", native_work)
        declared[path] = (mode, item["content"])
    if declared != {item["path"]: (item["mode"], item["content"]) for item in expected}:
        raise SafeDeltaRejected("native-work provider candidateFiles disagrees with the observed workspace delta", native_work)
    return expected, final_digest, delta_digest, [item["path"] for item in expected]


def receipt(base_digest: str, final_digest: str, delta_digest: str, paths: list[str], tool_calls: int) -> dict[str, Any]:
    return {
        "workspaceBaseDigest": base_digest,
        "workspaceFinalDigest": final_digest,
        "deltaDigest": delta_digest,
        "changedPaths": paths,
        "toolCalls": tool_calls,
        "helperStarts": 0,
        "helperAccounting": "disabled",
    }
