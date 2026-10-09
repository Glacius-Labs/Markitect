"""Conservative one-request Git approval policy for the owned App Server.

The broker only recognizes a small, explicit subset of Git argv rendered by
Codex. It never parses or executes shell syntax and never grants session-wide
or execution-policy permissions.
"""

from __future__ import annotations

import hashlib
import os
import re
import stat
from dataclasses import dataclass
from typing import Any


APPROVAL_METHOD = "item/commandExecution/requestApproval"
_BRANCH = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._/-]{0,99}$")
_PATH = re.compile(r"^[A-Za-z0-9._/-]+$")
_SHA256 = re.compile(r"^[0-9a-f]{64}$")
_SHELL_META = set(";&|<>`$()\r\n\x00")
_LOCAL_CORE_VALUES = {
    "repositoryformatversion": {"0"},
    "filemode": {"true", "false"},
    "bare": {"true", "false"},
    "logallrefupdates": {"true", "false"},
    "ignorecase": {"true", "false"},
    "symlinks": {"true", "false"},
}


def _is_reparse_point(metadata: os.stat_result) -> bool:
    reparse_attribute = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0x400)
    return bool(getattr(metadata, "st_file_attributes", 0) & reparse_attribute)


def _owned_repo_config_is_inert(repo_path: str) -> bool:
    """Accept only a regular .git/config containing known inert settings.

    This deliberately rejects worktree .git files, includes, filters, signing,
    merge drivers, and every unrecognized local setting. Parsing is limited to
    the tiny config subset emitted for ordinary repositories; Git is not run to
    inspect a config that could itself execute includes or helpers.
    """
    git_dir = os.path.join(repo_path, ".git")
    config_path = os.path.join(git_dir, "config")
    try:
        git_dir_metadata = os.lstat(git_dir)
        if not stat.S_ISDIR(git_dir_metadata.st_mode) or _is_reparse_point(git_dir_metadata):
            return False
        metadata = os.lstat(config_path)
        if (not stat.S_ISREG(metadata.st_mode) or _is_reparse_point(metadata)
                or metadata.st_size > 64 * 1024):
            return False
        with open(config_path, "rb") as stream:
            raw = stream.read(64 * 1024 + 1)
        if len(raw) > 64 * 1024:
            return False
        text = raw.decode("utf-8", errors="strict")
    except (OSError, UnicodeDecodeError, ValueError):
        return False

    section: str | None = None
    seen: set[tuple[str, str]] = set()
    for original in text.splitlines():
        line = original.strip()
        if not line or line.startswith(("#", ";")):
            continue
        if line.startswith("[") and line.endswith("]"):
            header = line[1:-1].strip().casefold()
            if header not in {"core", "user"}:
                return False
            section = header
            continue
        if section is None or "=" not in line or "\\" in line:
            return False
        key, value = (part.strip() for part in line.split("=", 1))
        key = key.casefold()
        if not key or not value or any(char in value for char in "\x00\r\n\"'`;#"):
            return False
        identity = (section, key)
        if identity in seen:
            return False
        seen.add(identity)
        if section == "core":
            allowed_values = _LOCAL_CORE_VALUES.get(key)
            if allowed_values is None or value.casefold() not in allowed_values:
                return False
        elif section == "user":
            if key not in {"name", "email"} or any(ord(char) < 32 for char in value):
                return False
    return True


@dataclass(frozen=True)
class ApprovalDecision:
    decision: str
    reason: str
    action: tuple[str, ...] | None = None


def _same_path(left: Any, right: Any) -> bool:
    if not isinstance(left, str) or not isinstance(right, str):
        return False
    try:
        return os.path.normcase(os.path.realpath(left)) == os.path.normcase(os.path.realpath(right))
    except (OSError, ValueError):
        return False


def _argv(command: Any) -> tuple[str, ...] | None:
    """Parse simple quoted argv while rejecting shell syntax in every token."""
    if not isinstance(command, str) or not command or len(command) > 4096:
        return None
    if any(char in _SHELL_META for char in command):
        return None
    words = _outer_argv(command)
    if not words or any("\\" in word for word in words[1:]):
        return None
    if any(not word for word in words):
        return None
    return words


def _branch_name(value: str) -> bool:
    return bool(_BRANCH.fullmatch(value) and not value.startswith(("-", "/"))
                and not value.endswith((".", "/")) and ".." not in value
                and "@{" not in value and "//" not in value
                and not value.casefold().startswith("origin/"))


def _new_branch_name(value: str) -> bool:
    return _branch_name(value) and value.casefold() not in {"main", "master", "trunk", "head"}


def _relative_paths(values: tuple[str, ...]) -> bool:
    if not values:
        return False
    for value in values:
        if (not _PATH.fullmatch(value) or value.startswith(("-", "/"))
                or value in {".", ".."} or any(part in {"", ".", ".."} for part in value.split("/"))):
            return False
    return True


def _git_action(argv: tuple[str, ...] | None, repo_path: str,
                git_executable: str) -> tuple[str, ...] | None:
    if (argv is None or len(argv) < 2 or not _same_path(argv[0], git_executable)
            or any(char in _SHELL_META or "\\" in char for arg in argv[1:] for char in arg)):
        return None
    # Every Git call disables repository hooks and configured fsmonitor
    # commands, which can otherwise execute arbitrary project/user code.
    required_prefix = ("-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false")
    if argv[1:5] != required_prefix or len(argv) < 6:
        return None
    subcommand, args = argv[5], argv[6:]
    if subcommand == "branch" and len(args) == 1 and _new_branch_name(args[0]):
        return argv
    if subcommand == "switch":
        if len(args) == 1 and _branch_name(args[0]):
            return argv
        if len(args) == 2 and args[0] in {"-c", "--create"} and _new_branch_name(args[1]):
            return argv
    if subcommand == "add":
        if args == ("--all",):
            return argv
        paths = args[1:] if args[:1] == ("--",) else args
        if _relative_paths(paths) and _paths_stay_in_repo(paths, repo_path):
            return argv
    if subcommand == "commit":
        if (len(args) == 2 and args[0] in {"-m", "--message"}
                and args[1].strip() and not args[1].startswith("-")):
            return argv
        if len(args) == 1 and args[0].startswith("--message=") and args[0][10:].strip():
            return argv
    if subcommand == "merge":
        if len(args) == 2 and args[0] in {"--ff-only", "--no-ff"} and _branch_name(args[1]):
            return argv
    return None


def _paths_stay_in_repo(paths: tuple[str, ...], repo_path: str) -> bool:
    root = os.path.normcase(os.path.realpath(repo_path))
    try:
        for path in paths:
            candidate = os.path.normcase(os.path.realpath(os.path.join(root, path.replace("/", os.sep))))
            if os.path.commonpath((root, candidate)) != root:
                return False
    except (OSError, ValueError):
        return False
    return True


def _outer_argv(command: Any) -> tuple[str, ...] | None:
    """Split only quoted argv fields while preserving Windows path separators."""
    if not isinstance(command, str) or not command or len(command) > 8192:
        return None
    words: list[str] = []
    current: list[str] = []
    quote: str | None = None
    active = False
    for char in command:
        if char in "\r\n\x00":
            return None
        if quote is not None:
            if char == quote:
                quote = None
            else:
                current.append(char)
            active = True
            continue
        if char in {"'", '"'}:
            quote = char
            active = True
        elif char.isspace():
            if active:
                words.append("".join(current))
                current = []
                active = False
        else:
            current.append(char)
            active = True
    if quote is not None:
        return None
    if active:
        words.append("".join(current))
    return tuple(words) if words else None


def _hash_file(path: str) -> str | None:
    try:
        digest = hashlib.sha256()
        with open(path, "rb") as stream:
            for chunk in iter(lambda: stream.read(1024 * 1024), b""):
                digest.update(chunk)
        return digest.hexdigest()
    except OSError:
        return None


def validate_git_approval_shell(value: Any) -> dict[str, str]:
    """Validate and canonicalize the exact PowerShell and Git binary pins."""
    fields = {"executable", "pinSha256", "gitExecutable", "gitPinSha256"}
    if not isinstance(value, dict) or set(value) != fields:
        raise ValueError("runtimeOptions.gitApprovalShell requires executable, pinSha256, gitExecutable, and gitPinSha256")
    result: dict[str, str] = {}
    for path_key, hash_key, basename in (
            ("executable", "pinSha256", "pwsh.exe"),
            ("gitExecutable", "gitPinSha256", "git.exe")):
        executable = value.get(path_key)
        pin = value.get(hash_key)
        if (not isinstance(executable, str) or not os.path.isabs(executable)
                or os.path.basename(executable).casefold() != basename):
            raise ValueError(f"runtimeOptions.gitApprovalShell.{path_key} must be an absolute {basename} path")
        if not isinstance(pin, str) or not _SHA256.fullmatch(pin) or not os.path.isfile(executable):
            raise ValueError(f"runtimeOptions.gitApprovalShell requires an existing {basename} and lowercase SHA-256 pin")
        if _hash_file(executable) != pin:
            raise ValueError(f"runtimeOptions.gitApprovalShell {basename} does not match its SHA-256 pin")
        result[path_key] = os.path.realpath(executable)
        result[hash_key] = pin
    return result


def _pinned_shell_action(command: Any, shell: dict[str, str], repo_path: str) -> tuple[str, ...] | None:
    try:
        pinned = validate_git_approval_shell(shell)
    except ValueError:
        return None
    executable = pinned["executable"]
    git_executable = pinned["gitExecutable"]
    argv = _outer_argv(command)
    if not argv:
        return None
    if not _same_path(argv[0], executable):
        return _git_action(_argv(command), repo_path, git_executable)
    index = 1
    switches: set[str] = set()
    while index < len(argv) and argv[index] in {"-NoProfile", "-NonInteractive"}:
        if argv[index] in switches:
            return None
        switches.add(argv[index])
        index += 1
    if "-NoProfile" not in switches:
        return None
    if index >= len(argv) or argv[index] != "-Command" or index + 2 != len(argv):
        return None
    inner = argv[index + 1]
    # PowerShell treats a quoted executable path by itself as a string. Permit
    # exactly the call operator required to invoke the pinned Git executable.
    if not inner.startswith("& ") or len(inner) <= 2 or inner[2].isspace():
        return None
    return _git_action(_argv(inner[2:]), repo_path, git_executable)


class ScopedGitApprovalBroker:
    """Authorize only exact, safe Git approvals for one owned App Server turn."""

    def __init__(self, repo_path: str, thread_id: str, turn_id: str,
                 git_approval_shell: dict[str, str] | None = None):
        self.repo_path = os.path.realpath(repo_path)
        self.thread_id = thread_id
        self.turn_id = turn_id
        self.git_approval_shell = dict(git_approval_shell) if git_approval_shell else None
        self._seen: set[tuple[type, Any]] = set()

    def decide(self, request: dict[str, Any], started_item: dict[str, Any] | None) -> ApprovalDecision:
        params = request.get("params")
        if request.get("method") != APPROVAL_METHOD or not isinstance(params, dict):
            return ApprovalDecision("decline", "not the exact command approval request method")
        if set(request) - {"id", "method", "params", "jsonrpc"}:
            return ApprovalDecision("decline", "server request contains unknown fields")
        if request.get("jsonrpc", "2.0") != "2.0":
            return ApprovalDecision("decline", "unsupported JSON-RPC version")
        known_fields = {
            "kind", "threadId", "turnId", "itemId", "startedAtMs", "approvalId",
            "environmentId", "reason", "networkApprovalContext", "command", "cwd",
            "commandActions", "additionalPermissions", "proposedExecpolicyAmendment",
            "proposedNetworkPolicyAmendments", "availableDecisions",
        }
        if set(params) - known_fields:
            return ApprovalDecision("decline", "approval request contains unknown fields")
        request_id = request.get("id")
        if type(request_id) not in (int, str) or not str(request_id):
            return ApprovalDecision("decline", "missing or invalid server request id")
        key = (type(request_id), request_id)
        if key in self._seen:
            return ApprovalDecision("decline", "duplicate server request id")
        self._seen.add(key)

        if params.get("threadId") != self.thread_id or params.get("turnId") != self.turn_id:
            return ApprovalDecision("decline", "thread or turn id does not match the owned turn")
        item_id = params.get("itemId")
        if not isinstance(item_id, str) or not item_id or not isinstance(started_item, dict):
            return ApprovalDecision("decline", "approval is not bound to an observed command item")
        item = started_item.get("item")
        if (not isinstance(item, dict) or item.get("type") != "commandExecution"
                or item.get("id") != item_id
                or started_item.get("threadId") != self.thread_id
                or started_item.get("turnId") != self.turn_id):
            return ApprovalDecision("decline", "item id does not match the observed command item")
        if (params.get("kind") != "command"
                or type(params.get("startedAtMs")) is not int or params["startedAtMs"] < 0):
            return ApprovalDecision("decline", "unsupported approval kind or missing start timestamp")
        if params.get("environmentId") is not None:
            return ApprovalDecision("decline", "approval targets a non-local or unknown environment")
        command = params.get("command")
        if not isinstance(command, str) or command != item.get("command"):
            return ApprovalDecision("decline", "approval command does not exactly match the observed item")
        if ("commandActions" in params and params["commandActions"] is not None
                and not isinstance(params["commandActions"], list)):
            return ApprovalDecision("decline", "command action classification has an unknown shape")
        if not _same_path(params.get("cwd"), self.repo_path) or not _same_path(item.get("cwd"), self.repo_path):
            return ApprovalDecision("decline", "command cwd is outside or differs from the owned repository")

        if params.get("approvalId") is not None and (
                not isinstance(params.get("approvalId"), str) or not params["approvalId"]):
            return ApprovalDecision("decline", "invalid approval id")
        if (params.get("networkApprovalContext") is not None
                or params.get("proposedExecpolicyAmendment") is not None
                or params.get("proposedNetworkPolicyAmendments") is not None
                or params.get("additionalPermissions") is not None):
            return ApprovalDecision("decline", "request includes network, permission, or policy escalation")
        # The current App Server schema makes availableDecisions optional.
        # If supplied, accept only a well-formed set that offers one-time
        # accept; other offered choices never change the selected decision.
        decisions = params.get("availableDecisions")
        if decisions is not None and (
                not isinstance(decisions, list)
                or any(type(decision) is not str for decision in decisions)
                or "accept" not in decisions
                or any(decision not in {"accept", "acceptForSession", "decline", "cancel"}
                       for decision in decisions)):
            return ApprovalDecision("decline", "request includes malformed or unsupported available decisions")

        if not _owned_repo_config_is_inert(self.repo_path):
            return ApprovalDecision("decline", "owned repository Git config is missing, non-regular, or outside the inert allowlist")

        action = (_pinned_shell_action(command, self.git_approval_shell, self.repo_path)
                  if self.git_approval_shell else None)
        if action is None:
            return ApprovalDecision("decline", "command is unknown or outside the scoped Git allowlist")
        return ApprovalDecision("accept", "exact single Git argv action is scoped to the owned repository", action)
