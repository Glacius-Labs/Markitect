"""Pre-registration: the evaluation files that judge a run are fixed in Git before it starts.

At run start (`host run` before the build, `study` once in its preflight) `observe`
reads the state of `evaluation/` in the playground's checkout. It is `registered` only
when `git status --porcelain --untracked-files=all -- evaluation` is empty; the run keeps
`{"status", "commit", "evaluationTree", "recordedAt"}` in host.json, where
`evaluationTree` is the Git tree of `evaluation/` at that commit. A changed or untracked
file (`dirty`, with its paths) or a playground outside a Git checkout (`no-checkout`) is
refused unless the run is exploratory.

`assess` judges with exactly that tree: `archive` gives its committed bytes and
`read_blob` its `config.json` (the reviewer models), never the working tree.

Reviewer models must differ from the arms' models (`clashes`): model ids are compared
casefolded, with a trailing `[...]` (such as `[1m]`) removed; the provider is not
checked. Standard library only: the assessment container imports this module too.
"""
from __future__ import annotations

import io
import json
import os
import re
import subprocess
import tarfile
from datetime import datetime, timezone
from fnmatch import fnmatch
from pathlib import Path
from typing import Any, Callable, Sequence

FOLDER = "evaluation"
GIT_TIMEOUT = 120
_OBJECT_ID = re.compile(r"[0-9a-f]{40}|[0-9a-f]{64}")
_SUFFIX = re.compile(r"\[[^\]]*\]$")

# A git runner takes the arguments after `git` and returns the finished process with
# bytes on stdout and stderr; `git_in(root)` runs real git in `root`.
Runner = Callable[[Sequence[str]], subprocess.CompletedProcess]


class RegistrationError(RuntimeError):
    """The evaluation files cannot be read from Git as registered."""


def git_in(root: Path) -> Runner:
    """Real git in `root`, never interactive. Raises RegistrationError when git is missing."""
    def run(args: Sequence[str]) -> subprocess.CompletedProcess:
        try:
            return subprocess.run(["git", "-C", str(root), *args], capture_output=True, timeout=GIT_TIMEOUT,
                                  stdin=subprocess.DEVNULL)
        except FileNotFoundError as exc:
            raise RegistrationError("git is not installed or not on PATH") from exc
        except (OSError, subprocess.SubprocessError) as exc:
            raise RegistrationError(f"git {args[0] if args else ''} failed: {exc}") from exc
    return run


def _text(data: bytes | str | None) -> str:
    return data.decode("utf-8", errors="replace") if isinstance(data, bytes) else (data or "")


def _first_line(data: bytes | str | None) -> str:
    return (_text(data).strip().splitlines() or [""])[0][:300]


def _now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def observe(root: Path, git: Runner | None = None) -> dict:
    """The pre-registration record of `root/evaluation` now: status `registered` (commit
    and tree), `dirty` (with `paths`) or `no-checkout` (with `error`)."""
    git = git or git_in(root)
    record: dict[str, Any] = {"status": "no-checkout", "commit": None, "evaluationTree": None,
                              "recordedAt": _now()}
    try:
        status = git(["status", "--porcelain", "--untracked-files=all", "--", FOLDER])
        if status.returncode != 0:
            record["error"] = f"not a Git checkout ({_first_line(status.stderr) or 'git status failed'})"
            return record
        paths = [line[3:] if len(line) > 3 else line for line in _text(status.stdout).splitlines() if line.strip()]
        ids = git(["rev-parse", "HEAD^{commit}", f"HEAD:./{FOLDER}"])
        found = _text(ids.stdout).split()
        if paths:
            record.update(status="dirty", paths=paths,
                          commit=found[0] if found and _OBJECT_ID.fullmatch(found[0]) else None)
            return record
        if ids.returncode != 0 or len(found) != 2 or not all(_OBJECT_ID.fullmatch(value) for value in found):
            record["error"] = f"{FOLDER}/ is not committed ({_first_line(ids.stderr) or 'no tree at HEAD'})"
            return record
        if not tree_exists(root, found[1], git):
            record["error"] = f"{FOLDER}/ at HEAD is not a folder"
            return record
    except RegistrationError as exc:
        record["error"] = str(exc)
        return record
    record.update(status="registered", commit=found[0], evaluationTree=found[1])
    return record


def refusal(record: dict) -> str:
    """Why `record` is not a registration, for an error message."""
    if record.get("status") == "dirty":
        paths = record.get("paths") or []
        listed = ", ".join(paths[:10]) + (f" and {len(paths) - 10} more" if len(paths) > 10 else "")
        return f"the evaluation files have uncommitted or untracked changes ({listed})"
    return f"the evaluation files are not pre-registered: {record.get('error') or record.get('status')}"


def tree_exists(root: Path, tree: str, git: Runner | None = None) -> bool:
    git = git or git_in(root)
    if not isinstance(tree, str) or not _OBJECT_ID.fullmatch(tree):
        return False
    done = git(["cat-file", "-t", tree])
    return done.returncode == 0 and _text(done.stdout).strip() == "tree"


def read_blob(root: Path, tree: str, path: str, git: Runner | None = None) -> bytes:
    """The committed bytes of `path` inside `tree`."""
    git = git or git_in(root)
    done = git(["cat-file", "blob", f"{tree}:{path}"])
    if done.returncode != 0:
        raise RegistrationError(f"{path} is not in the registered evaluation tree {tree} "
                                f"({_first_line(done.stderr) or 'git cat-file failed'})")
    return done.stdout


def read_json(root: Path, tree: str, path: str, git: Runner | None = None) -> Any:
    try:
        return json.loads(read_blob(root, tree, path, git).decode("utf-8"))
    except (UnicodeError, ValueError) as exc:
        raise RegistrationError(f"{path} in the registered evaluation tree {tree} is not valid JSON: {exc}") from exc


def archive(root: Path, tree: str, git: Runner | None = None) -> bytes:
    """A tar archive of `tree` with the committed bytes (no line-ending conversion)."""
    git = git or git_in(root)
    done = git(["-c", "core.autocrlf=false", "-c", "core.eol=lf", "archive", "--format=tar", tree])
    if done.returncode != 0:
        raise RegistrationError(f"git archive of the registered evaluation tree {tree} failed "
                                f"({_first_line(done.stderr)})")
    return done.stdout


def extract(data: bytes, target: Path, keep: Callable[[tuple[str, ...]], bool]) -> list[str]:
    """Write the regular files of a tar archive whose path parts `keep` accepts below
    `target`; returns their paths. Links and anything leaving `target` are skipped."""
    written = []
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:") as tar:
        for member in tar:
            parts = tuple(part for part in member.name.split("/") if part not in ("", "."))
            if not member.isfile() or not parts or ".." in parts or not keep(parts):
                continue
            path = target.joinpath(*parts)
            path.parent.mkdir(parents=True, exist_ok=True)
            handle = tar.extractfile(member)
            path.write_bytes(handle.read() if handle is not None else b"")
            os.chmod(path, 0o755 if member.mode & 0o111 else 0o644)
            written.append("/".join(parts))
    return written


def ignored(parts: tuple[str, ...], patterns: Sequence[str]) -> bool:
    """True when any path part matches one of `patterns` (shutil.ignore_patterns semantics)."""
    return any(fnmatch(part, pattern) for part in parts for pattern in patterns)


# --- reviewer independence -----------------------------------------------------------------

def model_key(model: Any) -> str | None:
    """A model id compared casefolded and without a trailing `[...]`; None when empty."""
    if not isinstance(model, str) or not model.strip():
        return None
    return _SUFFIX.sub("", model.strip()).strip().casefold() or None


def arm_models(manifest: dict | None, roles: list | None = None) -> dict[str, str]:
    """The arms' models of one run, key -> where it comes from: `agent.model`,
    `markitect.innerModel` and, when given, the model of every Markitect role."""
    found: dict[str, str] = {}

    def add(model: Any, source: str) -> None:
        key = model_key(model)
        if key is not None:
            found.setdefault(key, f"{source} {model}")

    manifest = manifest or {}
    add((manifest.get("agent") or {}).get("model"), "agent.model")
    add((manifest.get("markitect") or {}).get("innerModel"), "markitect.innerModel")
    for role in roles or []:
        if isinstance(role, dict):
            label = " ".join(str(role[key]) for key in ("role", "manager") if role.get(key))
            add(role.get("model"), f"role {label or '?'}")
    return found


def clashes(config: dict, names: Sequence[str], arms: dict[str, str]) -> list[dict]:
    """Selected reviewers whose model is also an arm's model."""
    found = []
    for name in names:
        model = ((config.get("reviewers") or {}).get(name) or {}).get("model")
        key = model_key(model)
        if key is not None and key in arms:
            found.append({"reviewer": name, "model": model, "arm": arms[key]})
    return found


def describe(found: list[dict]) -> str:
    return "; ".join(f"reviewer {c['reviewer']} uses model {c['model']}, as does {c['arm']}" for c in found)
