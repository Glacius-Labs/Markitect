"""Provider-independent preparation and freeze mechanics for the case playground.

This module only manipulates local files and Git history. It has no facility for
starting agents, model providers, product CLIs, or semantic assessment.
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time
import uuid
from typing import Any


SCHEMA = 1
CASES = ("roombook", "readinglog")
WAVE_SIZES = (1, 3, 7, 1)


class LifecycleError(RuntimeError):
    """A requested transition would make the record ambiguous or unsafe."""


def _utc() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _json_bytes(value: Any) -> bytes:
    return (json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n").encode("utf-8")


def _write_new(path: Path, data: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("xb") as handle:
        handle.write(data)


def _git(repo: Path, *args: str, timeout: int = 30, binary: bool = False) -> str | bytes:
    return _run(["git", "-C", str(repo), *args], timeout=timeout, binary=binary)


def _git_env() -> dict[str, str]:
    env = os.environ.copy()
    env["GIT_OPTIONAL_LOCKS"] = "0"
    return env


def _run(command: list[str], *, timeout: int = 30, binary: bool = False) -> str | bytes:
    try:
        proc = subprocess.run(command, capture_output=True, timeout=timeout, check=False, env=_git_env())
    except subprocess.TimeoutExpired as exc:
        raise LifecycleError(f"command timed out after {timeout}s: {command[0]}") from exc
    if proc.returncode:
        detail = proc.stderr.decode("utf-8", errors="replace").strip()
        raise LifecycleError(f"command {command[0]} failed ({proc.returncode}): {detail}")
    return proc.stdout if binary else proc.stdout.decode("utf-8", errors="strict").strip()


def _git_check(repo: Path, *args: str) -> None:
    """Run a check-style Git command whose nonzero status is meaningful."""
    proc = subprocess.run(["git", "-C", str(repo), *args], capture_output=True,
                          timeout=30, env=_git_env())
    if proc.returncode:
        detail = proc.stderr.decode("utf-8", errors="replace").strip()
        raise LifecycleError(f"git {' '.join(args)} failed ({proc.returncode}): {detail}")


def _ensure_separate(paths: list[Path]) -> None:
    resolved = [path.resolve() for path in paths]
    for index, left in enumerate(resolved):
        for right in resolved[index + 1:]:
            if left == right or left in right.parents or right in left.parents:
                raise LifecycleError("source, repository, and audit paths must not overlap")


def _status_bytes(repo: Path, *args: str) -> bytes:
    value = _git(repo, *args, binary=True)
    assert isinstance(value, bytes)
    return value


def _sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _safe_tree(root: Path) -> list[Path]:
    """Return regular files below root, rejecting links and non-file entries."""
    if root.is_symlink():
        raise LifecycleError(f"symlink tree root is not allowed: {root}")
    files: list[Path] = []
    for current, dirs, names in os.walk(root, followlinks=False):
        here = Path(current)
        for name in list(dirs):
            item = here / name
            if item.is_symlink():
                raise LifecycleError(f"symlink is not allowed in seed/worktree: {item}")
        for name in names:
            item = here / name
            if item.is_symlink() or not item.is_file():
                raise LifecycleError(f"non-regular file is not allowed: {item}")
            files.append(item)
    return sorted(files, key=lambda p: p.relative_to(root).as_posix())


def _manifest(root: Path, *, exclude_git: bool = False) -> dict[str, str]:
    result: dict[str, str] = {}
    for path in _safe_tree(root):
        rel = path.relative_to(root)
        if exclude_git and rel.parts and rel.parts[0] == ".git":
            continue
        result[rel.as_posix()] = _sha(path.read_bytes())
    return result


def _committed_manifest(repo: Path, commit: str) -> dict[str, str]:
    raw = _git(repo, "ls-tree", "-rz", "--full-tree", commit, binary=True)
    assert isinstance(raw, bytes)
    result: dict[str, str] = {}
    for entry in raw.split(b"\0"):
        if not entry:
            continue
        metadata, name = entry.split(b"\t", 1)
        mode, kind, object_id = metadata.decode("ascii").split(" ")
        path = name.decode("utf-8", errors="strict")
        if kind != "blob" or mode not in {"100644", "100755"}:
            raise LifecycleError(f"committed tree contains unsupported entry: {path}")
        data = _git(repo, "cat-file", "blob", object_id, binary=True)
        assert isinstance(data, bytes)
        result[path] = _sha(data)
    return dict(sorted(result.items()))


def _load_json(path: Path) -> dict[str, Any]:
    if path.is_symlink():
        raise LifecycleError(f"record cannot be a symlink: {path}")
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise LifecycleError(f"cannot read JSON record {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise LifecycleError(f"JSON record must be an object: {path}")
    return value


def _load_station_plan(path: Path, case: str) -> list[list[str]]:
    data = _load_json(path)
    if data.get("case") != case or not isinstance(data.get("stations"), list):
        raise LifecycleError(f"STATIONS.json does not describe case {case}")
    stations = data["stations"]
    if len(stations) != len(WAVE_SIZES):
        raise LifecycleError("STATIONS.json must declare exactly four public waves")
    plan: list[list[str]] = []
    seen: set[str] = set()
    for index, (station, expected_size) in enumerate(zip(stations, WAVE_SIZES), 1):
        if not isinstance(station, dict) or station.get("id") != f"S{index}":
            raise LifecycleError("STATIONS.json station ids must be S1 through S4 in order")
        items = station.get("items")
        if (not isinstance(items, list) or len(items) != expected_size
                or any(not isinstance(item, str) or not item.strip() for item in items)
                or len(set(items)) != len(items) or seen.intersection(items)):
            raise LifecycleError(f"STATIONS.json S{index} must contain {expected_size} unique item ids")
        seen.update(items)
        plan.append(list(items))
    return plan


def _station_record(station: int, plan: list[list[str]], previous: str | None = None) -> dict[str, Any]:
    value: dict[str, Any] = {
        "schema": SCHEMA,
        "station": f"S{station}",
        "items": list(plan[station - 1]),
    }
    if previous is not None:
        value["previousSnapshotSha256"] = previous
    return value


def prepare(seed_root: Path, repo: Path, audit: Path, *, case: str,
            method: str, profile: dict[str, Any], pins: dict[str, Any] | None = None,
            authorization: dict[str, Any] | None = None) -> dict[str, Any]:
    """Create one clean, isolated case checkout and a separate audit directory."""
    if repo.is_symlink() or audit.is_symlink():
        raise LifecycleError("repository and audit destinations cannot be symlinks")
    if seed_root.is_symlink():
        raise LifecycleError("public seed root cannot be a symlink")
    _ensure_separate([seed_root, repo, audit])
    seed_root, repo, audit = seed_root.resolve(), repo.resolve(), audit.resolve()
    if case not in CASES:
        raise LifecycleError(f"unsupported public case: {case}")
    if not method.strip():
        raise LifecycleError("method must be a non-empty label")
    if profile is None or not isinstance(profile, dict):
        raise LifecycleError("profile must be an object")
    if repo.exists() or repo.is_symlink():
        raise LifecycleError(f"destination already exists: {repo}")
    if audit.exists() or audit.is_symlink():
        raise LifecycleError(f"audit directory already exists: {audit}")
    if repo == audit or repo in audit.parents or audit in repo.parents:
        raise LifecycleError("repository and audit paths must be separate")

    common = seed_root / "common"
    case_root = seed_root / "cases" / case
    if common.is_symlink() or case_root.is_symlink() or not common.is_dir() or not case_root.is_dir():
        raise LifecycleError(f"public seed is incomplete for {case}: {seed_root}")
    common_files = _safe_tree(common)
    case_files = _safe_tree(case_root)
    required = {"AGENTS.md", "QUALITY.md", "checks/acceptance.py"}
    present = {p.relative_to(common).as_posix() for p in common_files}
    if not required <= present:
        raise LifecycleError(f"common seed lacks required files: {sorted(required - present)}")
    case_required = {"README.md", "BACKLOG.md", "STATIONS.json"}
    present_case = {p.relative_to(case_root).as_posix() for p in case_files}
    if not case_required <= present_case:
        raise LifecycleError(f"case seed lacks required files: {sorted(case_required - present_case)}")
    common_rel = {p.relative_to(common).as_posix() for p in common_files}
    case_rel = {p.relative_to(case_root).as_posix() for p in case_files}
    collisions = common_rel & case_rel
    if collisions:
        raise LifecycleError(f"common/case seed paths overlap: {sorted(collisions)}")
    reserved = {p.split("/", 1)[0] for p in common_rel | case_rel} & {".git", ".study"}
    if reserved:
        raise LifecycleError(f"seed uses reserved actor paths: {sorted(reserved)}")
    plan_path = case_root / "STATIONS.json"
    station_plan = _load_station_plan(plan_path, case)
    run_id = uuid.uuid4().hex

    repo.parent.mkdir(parents=True, exist_ok=True)
    repo.mkdir()
    for source_root, files in ((common, common_files), (case_root, case_files)):
        for source in files:
            rel = source.relative_to(source_root)
            destination = repo / rel
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, destination)

    # The first stage is part of the immutable seed commit. Keep LF bytes stable
    # before any branch checkout; command-scoped settings would not suffice.
    _write_new(repo / ".study" / "station.json", _json_bytes(_station_record(1, station_plan)))
    _write_new(repo / ".study" / "run-id", (run_id + "\n").encode("ascii"))
    subprocess.run(["git", "init", "--initial-branch=main", str(repo)],
                   capture_output=True, timeout=30, check=True)
    _git(repo, "config", "core.autocrlf", "false")
    _git(repo, "config", "user.name", "Case Study Playground")
    _git(repo, "config", "user.email", "playground@localhost")
    _git(repo, "add", "--all")
    _git_check(repo, "diff", "--cached", "--check")
    _git(repo, "commit", "-m", "Prepare public case seed")
    seed_commit = str(_git(repo, "rev-parse", "HEAD"))
    seed_tree = str(_git(repo, "rev-parse", "HEAD^{tree}"))
    _git(repo, "switch", "-c", "codex/backlog")

    run = {
        "schema": SCHEMA,
        "runId": run_id,
        "repoPath": str(repo),
        "auditPath": str(audit),
        "case": case,
        "method": method,
        "profile": profile,
        "pins": pins or {},
        "authorization": authorization if authorization is not None else {
            "execution_authorized": False, "status": "disabled", "source": "not supplied"},
        "providerLauncher": False,
        "execution": {"status": "not_started"},
        "setup": {"status": "not_recorded", "events": []},
        "seed": {"mainCommit": seed_commit, "mainTree": seed_tree,
                 "rawGitManifest": _committed_manifest(repo, seed_commit)},
        "stationPlan": station_plan,
        "stationPlanSha256": _sha(plan_path.read_bytes()),
        "preparedAt": _utc(),
    }
    audit.mkdir(parents=True)
    _write_new(audit / "run.json", _json_bytes(run))
    return run


def _load_run(audit: Path, repo: Path | None = None) -> dict[str, Any]:
    if audit.is_symlink() or (repo is not None and repo.is_symlink()):
        raise LifecycleError("repository and audit paths cannot be symlinks")
    audit = audit.resolve()
    path = audit / "run.json"
    run = _load_json(path)
    if Path(run.get("auditPath", "")).resolve() != audit:
        raise LifecycleError("audit directory does not match this run binding")
    if repo is not None:
        repo = repo.resolve()
        _ensure_separate([repo, audit])
        if Path(run.get("repoPath", "")).resolve() != repo:
            raise LifecycleError("repository directory does not match this run binding")
        marker = repo / ".study" / "run-id"
        if marker.is_symlink() or not marker.is_file() or marker.read_text(encoding="ascii").strip() != run.get("runId"):
            raise LifecycleError("repository run identity does not match audit metadata")
    return run


def _save_run(audit: Path, run: dict[str, Any]) -> None:
    path = audit / "run.json"
    temp = path.with_name(path.name + f".{uuid.uuid4().hex}.tmp")
    _write_new(temp, _json_bytes(run))
    os.replace(temp, path)


def record_setup(audit: Path, *, owner: str, status: str, details: dict[str, Any]) -> dict[str, Any]:
    """Record externally performed setup without generating product setup."""
    if audit.is_symlink():
        raise LifecycleError("audit path cannot be a symlink")
    audit = audit.resolve()
    run = _load_run(audit)
    if status not in {"complete", "blocked", "partial"}:
        raise LifecycleError("setup status must be complete, blocked, or partial")
    event = {"owner": owner, "status": status, "details": details, "recordedAt": _utc()}
    run["setup"]["events"].append(event)
    run["setup"]["status"] = status
    _save_run(audit, run)
    return event


def record_execution(audit: Path, *, metadata: dict[str, Any]) -> dict[str, Any]:
    """Record externally supplied execution metadata; this function never launches."""
    if audit.is_symlink():
        raise LifecycleError("audit path cannot be a symlink")
    audit = audit.resolve()
    run = _load_run(audit)
    if not isinstance(metadata, dict):
        raise LifecycleError("execution metadata must be an object")
    event = {"metadata": metadata, "recordedAt": _utc()}
    run["execution"] = event
    _save_run(audit, run)
    return event


def _station_number(repo: Path, plan: list[list[str]]) -> int:
    if (repo / ".study").is_symlink():
        raise LifecycleError(".study directory cannot be a symlink")
    path = repo / ".study" / "station.json"
    if path.is_symlink():
        raise LifecycleError("station declaration cannot be a symlink")
    record = _load_json(path)
    station = record.get("station")
    if not isinstance(station, str) or len(station) != 2 or station[0] != "S" or station[1] not in "1234":
        raise LifecycleError("current station declaration is invalid")
    number = int(station[1])
    if record.get("items") != plan[number - 1]:
        raise LifecycleError("declared station items differ from the fixed public wave")
    return number


def _verify_snapshots(audit: Path, case: str, through: int) -> None:
    for number in range(1, through + 1):
        folder = audit / f"snapshot-S{number}"
        payload = folder / "snapshot.json"
        binding_path = folder / "binding.json"
        binding = _load_json(binding_path)
        if _sha(payload.read_bytes()) != binding.get("snapshotSha256"):
            raise LifecycleError(f"station S{number} snapshot no longer matches its binding")
        record = _load_json(payload)
        if record.get("case") != case or record.get("station") != f"S{number}":
            raise LifecycleError(f"station S{number} snapshot has the wrong case/station")
        if _manifest(folder / "worktree") != record.get("rawWorktreeManifest"):
            raise LifecycleError(f"station S{number} raw worktree copy no longer matches its manifest")
        main = record.get("immutableMain", {})
        if _manifest(folder / "immutable-main", exclude_git=True) != main.get("rawGitManifest"):
            raise LifecycleError(f"station S{number} immutable main copy no longer matches its manifest")
        if _sha((folder / "history.bundle").read_bytes()) != record.get("bundleSha256"):
            raise LifecycleError(f"station S{number} Git bundle no longer matches its hash")
        actor_state = folder / "actor-state"
        if (_sha((actor_state / "index.raw").read_bytes()) != record.get("branch", {}).get("indexSha256")
                or _sha((actor_state / "staged.diff").read_bytes()) != record.get("branch", {}).get("stagedDiffSha256")
                or _sha((actor_state / "unstaged.diff").read_bytes()) != record.get("branch", {}).get("unstagedDiffSha256")):
            raise LifecycleError(f"station S{number} captured Git state no longer matches its hashes")


def _git_refs(repo: Path) -> dict[str, str]:
    value = str(_git(repo, "for-each-ref", "--format=%(refname) %(objectname)"))
    return {line.split(" ", 1)[0]: line.split(" ", 1)[1]
            for line in value.splitlines() if line.strip()}


def _capture(repo: Path, audit: Path, *, final: bool, reason: str | None = None) -> dict[str, Any]:
    _ensure_separate([repo, audit])
    run = _load_run(audit, repo)
    case = run.get("case")
    plan = run.get("stationPlan")
    if case not in CASES or not isinstance(plan, list) or len(plan) != 4:
        raise LifecycleError("run metadata has an invalid case or pinned station plan")
    number = _station_number(repo, plan)
    if final:
        if not isinstance(reason, str) or not reason.strip():
            raise LifecycleError("final freeze requires a reason, including blocked/early closure")
        _verify_snapshots(audit, case, number - 1)
    if (audit / "final-freeze").exists():
        raise LifecycleError("this run is already finally frozen")
    station = f"S{number}"
    target = audit / ("final-freeze" if final else f"snapshot-{station}")
    if target.exists() or target.is_symlink():
        raise LifecycleError(f"snapshot already exists: {target}")

    branch_label = str(_git(repo, "rev-parse", "--abbrev-ref", "HEAD"))
    branch = None if branch_label == "HEAD" else branch_label
    head = str(_git(repo, "rev-parse", "HEAD"))
    head_tree = str(_git(repo, "rev-parse", "HEAD^{tree}"))
    main = str(_git(repo, "rev-parse", "main"))
    main_tree = str(_git(repo, "rev-parse", "main^{tree}"))
    refs = _git_refs(repo)
    status_raw = _status_bytes(repo, "status", "--porcelain=v2", "--untracked-files=all", "-z")
    staged_diff = _status_bytes(repo, "diff", "--cached", "--binary")
    unstaged_diff = _status_bytes(repo, "diff", "--binary")
    untracked_raw = _status_bytes(repo, "ls-files", "--others", "--exclude-standard", "-z")
    index_path = Path(str(_git(repo, "rev-parse", "--git-path", "index")))
    if not index_path.is_absolute():
        index_path = repo / index_path
    index_bytes = index_path.read_bytes() if index_path.is_file() else b""
    index_sha = _sha(index_bytes) if index_bytes else None
    raw_worktree = _manifest(repo, exclude_git=True)
    raw_git = _committed_manifest(repo, head)
    main_manifest = _committed_manifest(repo, main)

    staging = Path(tempfile.mkdtemp(prefix=f".{target.name}-", dir=audit))
    try:
        bundle = staging / "history.bundle"
        _git(repo, "bundle", "create", str(bundle), "--all", "HEAD")

        # No checkout occurs until the clone has a local autocrlf=false setting.
        main_copy = staging / "immutable-main"
        _run(["git", "clone", "--no-checkout", "--no-hardlinks", str(repo), str(main_copy)])
        _git(main_copy, "config", "core.autocrlf", "false")
        _git(main_copy, "checkout", "--detach", main)
        copied_main_manifest = _manifest(main_copy, exclude_git=True)
        if copied_main_manifest != main_manifest:
            raise LifecycleError("immutable main checkout raw bytes differ from its Git manifest")

        copy_root = staging / "worktree"
        copy_root.mkdir()
        for source in _safe_tree(repo):
            rel = source.relative_to(repo)
            if rel.parts and rel.parts[0] == ".git":
                continue
            destination = copy_root / rel
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, destination)
        copied_worktree = _manifest(copy_root)
        after_worktree = _manifest(repo, exclude_git=True)
        if copied_worktree != raw_worktree or after_worktree != raw_worktree:
            raise LifecycleError("actor worktree changed during capture or copy bytes differ")
        actor_state = staging / "actor-state"
        actor_state.mkdir()
        _write_new(actor_state / "index.raw", index_bytes)
        _write_new(actor_state / "status.porcelain-v2", status_raw)
        _write_new(actor_state / "staged.diff", staged_diff)
        _write_new(actor_state / "unstaged.diff", unstaged_diff)
        _write_new(actor_state / "untracked-paths.nul", untracked_raw)
        if (str(_git(repo, "rev-parse", "HEAD")) != head
                or str(_git(repo, "rev-parse", "main")) != main
                or _git_refs(repo) != refs
                or _status_bytes(repo, "status", "--porcelain=v2", "--untracked-files=all", "-z") != status_raw
                or _status_bytes(repo, "diff", "--cached", "--binary") != staged_diff
                or _status_bytes(repo, "diff", "--binary") != unstaged_diff
                or (index_path.read_bytes() if index_path.is_file() else b"") != index_bytes):
            raise LifecycleError("actor Git/worktree state changed during capture")

        record = {
            "schema": SCHEMA,
            "kind": "final_freeze" if final else "station_snapshot",
            "case": case,
            "station": station,
            "items": list(plan[number - 1]),
            "method": run.get("method"),
            "profile": run.get("profile"),
            "pins": run.get("pins"),
            "authorization": run.get("authorization"),
            "providerLauncher": False,
            "execution": run.get("execution"),
            "setup": run.get("setup"),
            "capturedAt": _utc(),
            "freezeReason": reason if final else None,
            "immutableMain": {"commit": main, "tree": main_tree,
                              "rawGitManifest": main_manifest},
            "branch": {"name": branch, "head": head, "tree": head_tree,
                       "refs": refs, "selectedHeadUnfinished": head != main or bool(status_raw),
                       "overallUnfinished": "unknown",
                       "headDiffersFromMain": head != main, "dirty": bool(status_raw),
                       "statusPorcelainV2Base64": base64.b64encode(status_raw).decode("ascii"),
                       "indexSha256": index_sha,
                       "stagedDiffSha256": _sha(staged_diff),
                       "unstagedDiffSha256": _sha(unstaged_diff),
                       "untrackedPathsBase64": base64.b64encode(untracked_raw).decode("ascii")},
            "rawGitManifest": raw_git,
            "rawWorktreeManifest": raw_worktree,
            "bundleSha256": _sha(bundle.read_bytes()),
            "assessment": {"status": "pending"},
        }
        _write_new(staging / "snapshot.json", _json_bytes(record))
        record["snapshotSha256"] = _sha((staging / "snapshot.json").read_bytes())
        _write_new(staging / "binding.json", _json_bytes({"snapshotSha256": record["snapshotSha256"]}))
        os.replace(staging, target)
    except BaseException as exc:
        try:
            _write_new(staging / "capture-error.json", _json_bytes({"error": repr(exc), "recordedAt": _utc()}))
        except OSError:
            pass
        raise LifecycleError(f"capture failed; partial evidence preserved at {staging}: {exc}") from exc
    return record


def snapshot(repo: Path, audit: Path) -> dict[str, Any]:
    if repo.is_symlink() or audit.is_symlink():
        raise LifecycleError("repository and audit paths cannot be symlinks")
    return _capture(repo.resolve(), audit.resolve(), final=False)


def advance(repo: Path, audit: Path) -> dict[str, Any]:
    """Declare the next wave after a frozen snapshot, leaving the delta uncommitted."""
    if repo.is_symlink() or audit.is_symlink():
        raise LifecycleError("repository and audit paths cannot be symlinks")
    repo, audit = repo.resolve(), audit.resolve()
    _ensure_separate([repo, audit])
    run = _load_run(audit, repo)
    case = run.get("case")
    plan = run.get("stationPlan")
    if case not in CASES or not isinstance(plan, list):
        raise LifecycleError("run metadata has an unknown case")
    if (audit / "final-freeze").exists():
        raise LifecycleError("this run is already finally frozen")
    number = _station_number(repo, plan)
    if number >= 4:
        raise LifecycleError("S4 is the final station; it cannot advance")
    _verify_snapshots(audit, case, number)
    frozen = audit / f"snapshot-S{number}" / "binding.json"
    binding = _load_json(frozen)
    record_path = repo / ".study" / "station.json"
    if record_path.is_symlink():
        raise LifecycleError("station declaration cannot be a symlink")
    _git_check(repo, "diff", "--cached", "--quiet")
    current = _load_json(record_path)
    old_bytes = record_path.read_bytes()
    updated = _station_record(number + 1, plan, binding.get("snapshotSha256"))
    new_bytes = _json_bytes(updated)
    transition_dir = audit / "transitions"
    transition_path = transition_dir / f"S{number}-to-S{number + 1}.json"
    if transition_dir.is_symlink() or transition_path.exists() or transition_path.is_symlink():
        raise LifecycleError(f"transition already exists: {transition_path}")
    transition_dir.mkdir(exist_ok=True)
    transition_temp = transition_dir / f".{transition_path.name}.{uuid.uuid4().hex}.tmp"
    _write_new(transition_temp, _json_bytes({
        "from": current, "to": updated,
        "oldBytesSha256": _sha(old_bytes), "newBytesSha256": _sha(new_bytes),
        "actorDelta": "left_uncommitted", "committedByPlayground": False,
        "recordedAt": _utc(),
    }))
    station_temp = record_path.with_name(f"station.json.{uuid.uuid4().hex}.tmp")
    _write_new(station_temp, new_bytes)
    os.replace(station_temp, record_path)
    os.replace(transition_temp, transition_path)
    return updated


def freeze(repo: Path, audit: Path, *, reason: str) -> dict[str, Any]:
    if repo.is_symlink() or audit.is_symlink():
        raise LifecycleError("repository and audit paths cannot be symlinks")
    return _capture(repo.resolve(), audit.resolve(), final=True, reason=reason)


def _cli() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    p = commands.add_parser("prepare")
    p.add_argument("--seed", type=Path, required=True)
    p.add_argument("--repo", type=Path, required=True)
    p.add_argument("--audit", type=Path, required=True)
    p.add_argument("--case", choices=CASES, required=True)
    p.add_argument("--method", required=True)
    p.add_argument("--profile", type=Path, required=True, help="JSON profile metadata")
    p.add_argument("--pins", type=Path, help="JSON product/model pins, if resolved")
    p.add_argument("--authorization", type=Path, help="externally supplied authorization metadata")
    for name in ("snapshot", "advance", "freeze"):
        cmd = commands.add_parser(name)
        cmd.add_argument("--repo", type=Path, required=True)
        cmd.add_argument("--audit", type=Path, required=True)
        if name == "freeze":
            cmd.add_argument("--reason", required=True)
    setup = commands.add_parser("record-setup")
    setup.add_argument("--audit", type=Path, required=True)
    setup.add_argument("--owner", required=True)
    setup.add_argument("--status", choices=("complete", "blocked", "partial"), required=True)
    setup.add_argument("--details", type=Path, required=True)
    execution = commands.add_parser("record-execution")
    execution.add_argument("--audit", type=Path, required=True)
    execution.add_argument("--metadata", type=Path, required=True,
                           help="externally supplied execution metadata; this command does not launch")
    args = parser.parse_args()
    try:
        if args.command == "prepare":
            result = prepare(args.seed, args.repo, args.audit, case=args.case, method=args.method,
                             profile=_load_json(args.profile), pins=_load_json(args.pins) if args.pins else {},
                             authorization=_load_json(args.authorization) if args.authorization else None)
        elif args.command == "snapshot":
            result = snapshot(args.repo, args.audit)
        elif args.command == "advance":
            result = advance(args.repo, args.audit)
        elif args.command == "freeze":
            result = freeze(args.repo, args.audit, reason=args.reason)
        elif args.command == "record-setup":
            result = record_setup(args.audit, owner=args.owner, status=args.status,
                                  details=_load_json(args.details))
        else:
            result = record_execution(args.audit, metadata=_load_json(args.metadata))
    except (LifecycleError, OSError, subprocess.SubprocessError) as exc:
        print(f"lifecycle error: {exc}", file=sys.stderr)
        return 2
    print(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(_cli())
