"""Prepare, snapshot, advance and freeze one case repository.

Only local files and Git history are touched: no agents, providers or product CLIs.

Inside the run container the runner is root and the case repo belongs to the agent.
Root never runs Git on that repo: Git runs as the repo owner, with hooks, fsmonitor,
external diff and textconv switched off, and only reads. An optional `sweep` callable
(the runner passes its kill-all-agent-processes) then makes sure nothing of the owner
still runs before root copies plain files (never following symlinks, size-capped) and
checks out `main` from its own clone of the bundle.

The current station comes from the audit directory (one record per advance), not from
the repository, so an agent that edits or deletes `.study/station.json` cannot break
the capture; the declaration state is recorded.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tempfile
import time
import uuid
from typing import Any, BinaryIO, Callable

from .manifest import CASES

SCHEMA = 1
COMMON_REQUIRED = {"AGENTS.md", "QUALITY.md", "checks/acceptance.py"}
CASE_REQUIRED = {"README.md", "BACKLOG.md", "STATIONS.json"}
# Command-scope config wins over every config file: raw bytes, and nothing the
# repository configures (hooks, fsmonitor) runs. Diffs also pass --no-ext-diff --no-textconv.
GIT_CONFIG = (("core.autocrlf", "false"), ("core.fsmonitor", "false"), ("core.hooksPath", os.devnull),
              ("commit.gpgsign", "false"), ("core.longpaths", "true"))  # longpaths: manual use on Windows
RAW_ATTRIBUTES = b"* -text -eol -ident -filter -working-tree-encoding\n"
FILE_CAP = 64 << 20  # larger files are listed by size, not copied
TREE_CAP = 1 << 30   # per copied tree
SEED_AUTHOR = ("user.name=Case Study Playground", "user.email=playground@localhost")
AGENT_IDENTITY = ("Case Study Agent", "agent@localhost")


class LifecycleError(RuntimeError):
    """A requested transition would make the record ambiguous or unsafe."""


# --- small helpers (also used by other modules) --------------------------------------

def utc() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _json_bytes(value: Any) -> bytes:
    # ASCII escapes keep non-UTF-8 file names (surrogates) representable.
    return (json.dumps(value, ensure_ascii=True, indent=2, sort_keys=True) + "\n").encode("ascii")


def _write_new(path: Path, data: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("xb") as handle:
        handle.write(data)


def sha_file(path: Path) -> str:
    with path.open("rb") as handle:
        return hashlib.file_digest(handle, "sha256").hexdigest()


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""


def _load_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise LifecycleError(f"cannot read JSON record {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise LifecycleError(f"JSON record must be an object: {path}")
    return value


def match_owner(path: Path, reference: Path) -> None:
    """Root writes into an agent-owned repo; give the new entry the reference owner."""
    if not hasattr(os, "geteuid") or os.geteuid() != 0:
        return
    info = reference.stat()
    os.chown(path, info.st_uid, info.st_gid, follow_symlinks=False)


def open_plain(path: Path, owner: int | None = None) -> BinaryIO | None:
    """Open a regular file without following a final symlink (and without blocking on a
    FIFO); None for anything else or, with `owner`, for a file another user owns."""
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_NONBLOCK", 0) | getattr(os, "O_BINARY", 0)
    try:
        fd = os.open(path, flags)
    except OSError:
        return None
    info = os.fstat(fd)
    if not stat.S_ISREG(info.st_mode) or (owner is not None and info.st_uid != owner):
        os.close(fd)
        return None
    return os.fdopen(fd, "rb")


# --- git -----------------------------------------------------------------------------

def git_env(base: dict[str, str] | None = None) -> dict[str, str]:
    """`base` (default: this process's environment) plus the command-scope Git config."""
    env = dict(os.environ if base is None else base)
    env.update(GIT_OPTIONAL_LOCKS="0", GIT_TERMINAL_PROMPT="0")
    count = int(env.get("GIT_CONFIG_COUNT") or 0)
    for key, value in GIT_CONFIG:
        env[f"GIT_CONFIG_KEY_{count}"] = key
        env[f"GIT_CONFIG_VALUE_{count}"] = value
        count += 1
    env["GIT_CONFIG_COUNT"] = str(count)
    return env


def _owner(path: Path) -> int | None:
    """The uid Git must run as: the owner of `path` when root works on someone else's tree."""
    if not hasattr(os, "geteuid") or os.geteuid() != 0:
        return None
    return path.stat().st_uid or None


def _run(command: list[str], *, timeout: int = 300, stdin: bytes | None = None,
         as_owner_of: Path | None = None) -> bytes:
    kwargs: dict[str, Any] = {"env": git_env()}
    uid = _owner(as_owner_of) if as_owner_of is not None else None
    if uid is not None:
        import pwd  # POSIX only
        entry = pwd.getpwuid(uid)
        clean = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "LANG": "C.UTF-8", "HOME": entry.pw_dir,
                 "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull}
        kwargs = {"env": git_env(clean), "user": uid, "group": entry.pw_gid, "extra_groups": [],
                  "cwd": as_owner_of}
    if stdin is None:
        kwargs["stdin"] = subprocess.DEVNULL
    try:
        proc = subprocess.run(command, input=stdin, capture_output=True, timeout=timeout, check=False, **kwargs)
    except subprocess.TimeoutExpired as exc:
        raise LifecycleError(f"command timed out after {timeout}s: {' '.join(command[:4])}") from exc
    if proc.returncode:
        detail = proc.stderr.decode("utf-8", errors="replace").strip()
        raise LifecycleError(f"{' '.join(command[:4])} failed ({proc.returncode}): {detail}")
    return proc.stdout


def git_bytes(repo: Path, *args: str, timeout: int = 300) -> bytes:
    """Run git in repo (as its owner when root) and return raw stdout."""
    return _run(["git", "-C", str(repo), *args], timeout=timeout, as_owner_of=Path(repo))


def git(repo: Path, *args: str, timeout: int = 300) -> str:
    """Run git in repo and return stripped stdout; raise LifecycleError on failure."""
    return git_bytes(repo, *args, timeout=timeout).decode("utf-8", errors="surrogateescape").strip()


# --- trees and manifests -------------------------------------------------------------

def _entries(root: Path, *, exclude_git: bool = False) -> dict[str, tuple[str, Path]]:
    """Map POSIX relative paths to (kind, path); kind is file, symlink or special."""
    found: dict[str, tuple[str, Path]] = {}
    for current, dirs, names in os.walk(root, followlinks=False):
        here = Path(current)
        if exclude_git and here == root:
            dirs[:] = [name for name in dirs if name != ".git"]
            names = [name for name in names if name != ".git"]
        for name in list(dirs):
            if (here / name).is_symlink():
                dirs.remove(name)
                names.append(name)
        for name in names:
            item = here / name
            kind = "symlink" if item.is_symlink() else "file" if item.is_file() else "special"
            found[item.relative_to(root).as_posix()] = (kind, item)
    return dict(sorted(found.items()))


def _manifest(root: Path, *, exclude_git: bool = False) -> dict[str, str]:
    """Raw byte hashes; symlinks as `symlink:<target>`, other non-files as `special`."""
    result: dict[str, str] = {}
    for rel, (kind, path) in _entries(root, exclude_git=exclude_git).items():
        if kind == "file":
            result[rel] = sha_file(path)
        elif kind == "symlink":
            result[rel] = "symlink:" + os.readlink(path)
        else:
            result[rel] = "special"
    return result


def _files_only(manifest: dict[str, str]) -> dict[str, str]:
    """Entries that are copied byte for byte (symlinks/gitlinks/specials are only listed)."""
    return {rel: value for rel, value in manifest.items()
            if value != "special" and not value.startswith(("symlink:", "gitlink:"))}


def copy_tree(source: Path, target: Path, *, exclude_git: bool = False, owner: int | None = None) -> dict[str, str]:
    """Copy the regular files under `source` to `target`; return the raw manifest.

    Symlinks are listed as `symlink:<target>` and never followed, other non-files as
    `special`, files over FILE_CAP (or past TREE_CAP for the tree) as `skipped:<bytes>`
    and files that cannot be opened safely (or, with `owner`, belong to someone else)
    as `unreadable`.
    """
    manifest: dict[str, str] = {}
    total = 0
    target.mkdir(parents=True, exist_ok=True)
    for rel, (kind, path) in _entries(source, exclude_git=exclude_git).items():
        if kind == "symlink":
            manifest[rel] = "symlink:" + os.readlink(path)
            continue
        handle = open_plain(path, owner) if kind == "file" else None
        if handle is None:
            manifest[rel] = "unreadable" if kind == "file" else "special"
            continue
        with handle:
            size = os.fstat(handle.fileno()).st_size
            if size > FILE_CAP or total + size > TREE_CAP:
                manifest[rel] = f"skipped:{size}"
                continue
            destination = target / rel
            destination.parent.mkdir(parents=True, exist_ok=True)
            digest = hashlib.sha256()
            with destination.open("xb") as out:
                for block in iter(lambda: handle.read(1 << 20), b""):
                    digest.update(block)
                    out.write(block)
            manifest[rel] = digest.hexdigest()
            total += size
    return manifest


def _committed_manifest(repo: Path, commit: str) -> dict[str, str]:
    raw = git_bytes(repo, "ls-tree", "-rz", "--full-tree", commit)
    entries: list[tuple[str, str, str]] = []
    for entry in raw.split(b"\0"):
        if entry:
            metadata, name = entry.split(b"\t", 1)
            mode, _kind, object_id = metadata.decode("ascii").split(" ")
            entries.append((name.decode("utf-8", errors="surrogateescape"), mode, object_id))
    blobs = _read_blobs(repo, sorted({oid for _path, mode, oid in entries if mode != "160000"}))
    result: dict[str, str] = {}
    for path, mode, oid in entries:
        if mode == "160000":
            result[path] = "gitlink:" + oid
        elif mode == "120000":
            result[path] = "symlink:" + blobs[oid].decode("utf-8", errors="surrogateescape")
        else:
            result[path] = hashlib.sha256(blobs[oid]).hexdigest()
    return dict(sorted(result.items()))


def _read_blobs(repo: Path, object_ids: list[str]) -> dict[str, bytes]:
    if not object_ids:
        return {}
    out = _run(["git", "-C", str(repo), "cat-file", "--batch"],
               stdin=("\n".join(object_ids) + "\n").encode("ascii"), as_owner_of=repo)
    blobs: dict[str, bytes] = {}
    position = 0
    for oid in object_ids:
        header_end = out.index(b"\n", position)
        header = out[position:header_end].decode("ascii").split()
        if len(header) != 3:
            raise LifecycleError(f"cannot read Git object {oid}: {' '.join(header)}")
        size = int(header[2])
        blobs[oid] = out[header_end + 1:header_end + 1 + size]
        position = header_end + 1 + size + 1
    return blobs


# --- seed and station plan -----------------------------------------------------------

def _seed_files(root: Path) -> list[str]:
    files = []
    for rel, (kind, path) in _entries(root).items():
        if "__pycache__" in rel.split("/"):
            continue
        if kind != "file":
            raise LifecycleError(f"seed may only contain regular files: {path}")
        files.append(rel)
    return files


def load_station_plan(path: Path, case: str) -> list[list[str]]:
    """The case's public waves from STATIONS.json: item ids per station S1..SN, in order."""
    data = _load_json(path)
    if data.get("case") != case or not isinstance(data.get("stations"), list) or not data["stations"]:
        raise LifecycleError(f"STATIONS.json does not describe case {case} with at least one station")
    plan: list[list[str]] = []
    seen: set[str] = set()
    for index, station in enumerate(data["stations"], 1):
        if not isinstance(station, dict) or station.get("id") != f"S{index}":
            raise LifecycleError(f"STATIONS.json station ids must be S1 through S{len(data['stations'])} in order")
        items = station.get("items")
        if (not isinstance(items, list) or not items
                or any(not isinstance(item, str) or not item.strip() for item in items)
                or len(set(items)) != len(items) or seen.intersection(items)):
            raise LifecycleError(f"STATIONS.json S{index} must contain unique item ids not used before")
        seen.update(items)
        plan.append(list(items))
    return plan


def _station_record(station: int, plan: list[list[str]]) -> dict[str, Any]:
    return {"schema": SCHEMA, "station": f"S{station}", "items": list(plan[station - 1])}


# --- prepare -------------------------------------------------------------------------

def prepare(seed_root: Path, repo: Path, audit: Path, *, case: str, method: str) -> dict[str, Any]:
    """Create a fresh case repo (seed commit on main, S1 released) and its audit directory.

    Seed = `<seed_root>/common` + `<seed_root>/<case>`; nothing else (e.g. the task prompt)
    is copied.
    """
    seed_root, repo, audit = Path(seed_root).resolve(), Path(repo).resolve(), Path(audit).resolve()
    if case not in CASES:
        raise LifecycleError(f"unsupported case: {case}")
    if not method.strip():
        raise LifecycleError("method must be a non-empty label")
    for path in (repo, audit):
        if path.exists():
            raise LifecycleError(f"destination already exists: {path}")

    common, case_root = seed_root / "common", seed_root / case
    if not (common.is_dir() and case_root.is_dir()):
        raise LifecycleError(f"seed is incomplete for {case}: {seed_root}")
    common_files, case_files = _seed_files(common), _seed_files(case_root)
    if not COMMON_REQUIRED <= set(common_files):
        raise LifecycleError(f"common seed lacks {sorted(COMMON_REQUIRED - set(common_files))}")
    if not CASE_REQUIRED <= set(case_files):
        raise LifecycleError(f"case seed lacks {sorted(CASE_REQUIRED - set(case_files))}")
    collisions = set(common_files) & set(case_files)
    if collisions:
        raise LifecycleError(f"common/case seed paths overlap: {sorted(collisions)}")
    reserved = {rel.split("/", 1)[0] for rel in common_files + case_files} & {".git", ".study"}
    if reserved:
        raise LifecycleError(f"seed uses reserved paths: {sorted(reserved)}")
    plan = load_station_plan(case_root / "STATIONS.json", case)

    repo.mkdir(parents=True)
    for source_root, files in ((common, common_files), (case_root, case_files)):
        for rel in files:
            destination = repo / rel
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source_root / rel, destination)
    # S1 is part of the seed commit; LF bytes are written directly.
    _write_new(repo / ".study" / "station.json", _json_bytes(_station_record(1, plan)))
    _run(["git", "init", "--initial-branch=main", str(repo)])
    git(repo, "config", "core.autocrlf", "false")
    git(repo, "config", "user.name", AGENT_IDENTITY[0])
    git(repo, "config", "user.email", AGENT_IDENTITY[1])
    git(repo, "add", "--all")
    git(repo, "diff", "--cached", "--check")
    git(repo, "-c", SEED_AUTHOR[0], "-c", SEED_AUTHOR[1], "commit", "-q", "-m", "Prepare public case seed")
    seed_commit = git(repo, "rev-parse", "HEAD")

    run = {
        "schema": SCHEMA,
        "case": case,
        "method": method,
        "seed": {"mainCommit": seed_commit, "rawGitManifest": _committed_manifest(repo, seed_commit)},
        "stationPlan": plan,
        "preparedAt": utc(),
    }
    audit.mkdir(parents=True)
    _write_new(audit / "run.json", _json_bytes(run))
    return run


# --- capture -------------------------------------------------------------------------

def _current_station(audit: Path) -> int:
    transitions = audit / "transitions"
    return 1 + (len(list(transitions.glob("S*-to-S*.json"))) if transitions.is_dir() else 0)


def _declaration_state(copy_root: Path, plan: list[list[str]], number: int) -> dict[str, Any]:
    """State of `.study/station.json`, read from root's own worktree copy."""
    path = copy_root / ".study" / "station.json"
    if path.is_symlink() or not path.is_file():
        return {"status": "missing", "sha256": None}
    data = path.read_bytes()
    try:
        value = json.loads(data.decode("utf-8"))
        intact = (isinstance(value, dict) and value.get("station") == f"S{number}"
                  and value.get("items") == plan[number - 1])
    except (UnicodeError, ValueError):
        intact = False
    return {"status": "intact" if intact else "modified", "sha256": hashlib.sha256(data).hexdigest()}


def _read_git_state(repo: Path) -> dict[str, Any]:
    """Read-only Git state of the actor repo; failures are collected, not raised."""
    errors: list[str] = []

    def read(*args: str, quiet: bool = False) -> bytes | None:
        try:
            return git_bytes(repo, *args)
        except LifecycleError as exc:
            if not quiet:
                errors.append(str(exc))
            return None

    def text(value: bytes | None) -> str | None:
        return (value or b"").decode("utf-8", errors="surrogateescape").strip() or None

    refs = text(read("for-each-ref", "--format=%(refname) %(objectname)")) or ""
    return {
        "head": text(read("rev-parse", "--verify", "--quiet", "HEAD^{commit}", quiet=True)),
        "main": text(read("rev-parse", "--verify", "--quiet", "refs/heads/main^{commit}", quiet=True)),
        "branch": text(read("symbolic-ref", "--quiet", "--short", "HEAD", quiet=True)),
        "refs": dict(line.split(" ", 1) for line in refs.splitlines() if " " in line),
        "status": read("status", "--porcelain=v2", "--untracked-files=all", "-z") or b"",
        "staged": read("diff", "--cached", "--binary", "--no-ext-diff", "--no-textconv") or b"",
        "unstaged": read("diff", "--binary", "--no-ext-diff", "--no-textconv") or b"",
        "untracked": read("ls-files", "--others", "--exclude-standard", "-z") or b"",
        "errors": errors,
    }


def _checkout_main(bundle: Path, target: Path, commit: str) -> dict[str, str]:
    """Root's own clone of the bundle with `main` checked out as raw blob bytes."""
    _run(["git", "clone", "--quiet", "--no-checkout", str(bundle), str(target)])
    info = target / ".git" / "info"
    info.mkdir(parents=True, exist_ok=True)
    (info / "attributes").write_bytes(RAW_ATTRIBUTES)  # in-tree .gitattributes must not convert
    git(target, "checkout", "--quiet", "--detach", commit)
    manifest = _committed_manifest(target, commit)
    if _files_only(_manifest(target, exclude_git=True)) != _files_only(manifest):
        raise LifecycleError("immutable main checkout raw bytes differ from its Git manifest")
    return manifest


def _capture(repo: Path, audit: Path, *, final: bool, reason: str | None,
             sweep: Callable[[], Any] | None) -> dict[str, Any]:
    repo, audit = Path(repo).resolve(), Path(audit).resolve()
    run = _load_json(audit / "run.json")
    plan = run.get("stationPlan")
    if not isinstance(plan, list) or not plan:
        raise LifecycleError("run.json has no valid station plan")
    if not repo.is_dir():  # a broken or missing .git is recorded, not fatal
        raise LifecycleError(f"repository directory missing: {repo}")
    if final and (not isinstance(reason, str) or not reason.strip()):
        raise LifecycleError("final freeze requires a reason, including blocked/early closure")
    if (audit / "final-freeze").exists():
        raise LifecycleError("this run is already finally frozen")
    number = _current_station(audit)
    station = f"S{number}"
    target = audit / ("final-freeze" if final else f"snapshot-{station}")
    if target.exists() or target.is_symlink():
        raise LifecycleError(f"snapshot already exists: {target}")

    staging = Path(tempfile.mkdtemp(prefix=f".{target.name}-", dir=audit))
    scratch = Path(tempfile.mkdtemp(prefix="mpg-capture-"))  # the repo owner writes the bundle here
    try:
        # 1. Git reads, as the repo owner.
        state = _read_git_state(repo)
        errors: list[str] = state["errors"]
        owner = _owner(repo)
        if owner is not None:
            os.chown(scratch, owner, -1)
        produced = scratch / "history.bundle"
        if state["refs"] or state["head"]:
            try:
                git_bytes(repo, "bundle", "create", str(produced), "--all", *(["HEAD"] if state["head"] else []))
            except LifecycleError as exc:
                errors.append(str(exc))
        # 2. Nothing of the owner may run while root copies its files.
        if sweep is not None:
            sweep()
        # 3. Root copies plain files and works on its own clone of the bundle.
        bundle: Path | None = staging / "history.bundle"
        source = open_plain(produced, owner)
        if source is None:
            bundle = None
        else:
            with source, bundle.open("xb") as out:
                shutil.copyfileobj(source, out)
        worktree = copy_tree(repo, staging / "worktree", exclude_git=True, owner=owner)
        main_manifest = None
        if bundle is not None and state["main"]:
            main_manifest = _checkout_main(bundle, staging / "immutable-main", state["main"])
        else:
            errors.append("no main branch to check out" if bundle is not None else "no Git bundle")
        actor_state = staging / "actor-state"
        for name, key in (("status.porcelain-v2", "status"), ("staged.diff", "staged"),
                          ("unstaged.diff", "unstaged"), ("untracked-paths.nul", "untracked")):
            _write_new(actor_state / name, state[key])
        dirty = bool(state["status"])
        record = {
            "schema": SCHEMA,
            "kind": "final_freeze" if final else "station_snapshot",
            "case": run.get("case"),
            "method": run.get("method"),
            "station": station,
            "items": list(plan[number - 1]),
            "capturedAt": utc(),
            "freezeReason": reason if final else None,
            "stationDeclaration": _declaration_state(staging / "worktree", plan, number),
            "immutableMain": {"commit": state["main"] if main_manifest is not None else None,
                              "rawGitManifest": main_manifest},
            "branch": {"name": state["branch"], "head": state["head"], "refs": state["refs"],
                       "headDiffersFromMain": state["head"] != state["main"], "dirty": dirty,
                       "selectedHeadUnfinished": state["head"] != state["main"] or dirty},
            "rawWorktreeManifest": worktree,
            "bundleSha256": sha_file(bundle) if bundle is not None else None,
            "errors": errors,
        }
        _write_new(staging / "snapshot.json", _json_bytes(record))
        os.replace(staging, target)
    except BaseException as exc:
        try:
            _write_new(staging / "capture-error.json", _json_bytes({"error": repr(exc), "recordedAt": utc()}))
        except OSError:
            pass
        raise LifecycleError(f"capture failed; partial evidence preserved at {staging}: {exc}") from exc
    finally:
        shutil.rmtree(scratch, ignore_errors=True)
    main_dir = target / "immutable-main"
    return {"kind": record["kind"], "station": station, "dir": str(target),
            "immutableMain": str(main_dir) if main_manifest is not None else None,
            "worktree": str(target / "worktree"),
            "bundle": str(target / "history.bundle") if bundle is not None else None, "record": record}


def snapshot(repo: Path, audit: Path, *, sweep: Callable[[], Any] | None = None) -> dict[str, Any]:
    """Capture the current station: bundle, immutable main checkout, worktree copy, Git state."""
    return _capture(repo, audit, final=False, reason=None, sweep=sweep)


def freeze(repo: Path, audit: Path, *, reason: str, sweep: Callable[[], Any] | None = None) -> dict[str, Any]:
    """Final capture; afterwards no snapshot or advance is possible."""
    return _capture(repo, audit, final=True, reason=reason, sweep=sweep)


# --- advance -------------------------------------------------------------------------

def advance(repo: Path, audit: Path) -> dict[str, Any]:
    """Release the next wave by rewriting only `.study/station.json` (left uncommitted)."""
    repo, audit = Path(repo).resolve(), Path(audit).resolve()
    plan = _load_json(audit / "run.json").get("stationPlan")
    if not isinstance(plan, list) or not plan:
        raise LifecycleError("run.json has no valid station plan")
    if (audit / "final-freeze").exists():
        raise LifecycleError("this run is already finally frozen")
    number = _current_station(audit)
    if number >= len(plan):
        raise LifecycleError(f"S{len(plan)} is the final station; it cannot advance")
    if not (audit / f"snapshot-S{number}").is_dir():
        raise LifecycleError(f"S{number} must be snapshotted before advancing")
    if not (repo / ".git").exists():  # never write into an unrelated folder
        raise LifecycleError(f"not a case repository (no .git): {repo}")

    study = repo / ".study"
    if study.is_symlink() or (study.exists() and not study.is_dir()):
        study.unlink()  # the harness owns this path; never write through a link
    if not study.exists():
        study.mkdir()
        match_owner(study, repo)
    updated = _station_record(number + 1, plan)
    temp = study / f".station.json.{uuid.uuid4().hex}.tmp"
    _write_new(temp, _json_bytes(updated))
    match_owner(temp, repo)
    os.replace(temp, study / "station.json")
    _write_new(audit / "transitions" / f"S{number}-to-S{number + 1}.json",
               _json_bytes({"from": f"S{number}", "to": f"S{number + 1}", "recordedAt": utc()}))
    return updated


# --- CLI -----------------------------------------------------------------------------

def _cli(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Manual case lifecycle (prepare/snapshot/advance/freeze).")
    commands = parser.add_subparsers(dest="command", required=True)
    p = commands.add_parser("prepare")
    p.add_argument("--seed", type=Path, required=True, help="cases/ folder with common/ and <case>/")
    p.add_argument("--repo", type=Path, required=True)
    p.add_argument("--audit", type=Path, required=True)
    p.add_argument("--case", choices=CASES, required=True)
    p.add_argument("--method", required=True)
    for name in ("snapshot", "advance", "freeze"):
        cmd = commands.add_parser(name)
        cmd.add_argument("--repo", type=Path, required=True)
        cmd.add_argument("--audit", type=Path, required=True)
        if name == "freeze":
            cmd.add_argument("--reason", required=True)
    args = parser.parse_args(argv)
    try:
        if args.command == "prepare":
            result = prepare(args.seed, args.repo, args.audit, case=args.case, method=args.method)
        elif args.command == "snapshot":
            result = snapshot(args.repo, args.audit)
        elif args.command == "advance":
            result = advance(args.repo, args.audit)
        else:
            result = freeze(args.repo, args.audit, reason=args.reason)
    except (LifecycleError, OSError) as exc:
        print(f"lifecycle error: {exc}", file=sys.stderr)
        return 2
    print(json.dumps(result, ensure_ascii=True, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(_cli())
