"""Assess a frozen candidate on scratch copies; the candidate itself is never changed.

Every check gets its own fresh copy without `.git` and runs through
`codex_agent.run_as_agent` (as the agent user inside the container, as the current
user in test mode) with a fresh empty HOME and TMPDIR, Python without user site or
PYTHON* variables, and Git without global or system config, so nothing the agent left
in its own environment takes part. Raw output goes to `out_dir`; the runner saves the
returned summary (`checks.json` per station, `final.json`).
"""
from __future__ import annotations

import json
import os
import re
import shutil
import stat
import sys
import tempfile
from contextlib import contextmanager
from pathlib import Path
from typing import Any, Iterator

from . import codex_agent
from .lifecycle import load_station_plan, read_text as _read

ACCEPTANCE_TIMEOUT = 600
OWN_TESTS_TIMEOUT = 900
PRODUCT_TIMEOUT = 600
GIT_TIMEOUT = 120
SWEEP_SECONDS = 30  # codex_agent.kill_all_agent_processes bound after every command
# Worst-case durations, used for the host's safety timeout.
STATION_BOUND_SECONDS = ACCEPTANCE_TIMEOUT + SWEEP_SECONDS
FINAL_BOUND_SECONDS = (ACCEPTANCE_TIMEOUT + 2 * OWN_TESTS_TIMEOUT + 3 * GIT_TIMEOUT + PRODUCT_TIMEOUT
                       + 7 * SWEEP_SECONDS)
MARKITECT_INSTALLED = Path("/usr/local/bin/markitect")
SCRATCH_GIT = ["git", "-c", "user.name=Assessment", "-c", "user.email=assessment@playground.invalid",
               "-c", "core.autocrlf=false"]


def station(candidate_dir: Path, case: str, station: int, in_dir: Path, out_dir: Path) -> dict:
    """Run the public checks released up to `station` on a scratch copy."""
    out_dir = Path(out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    return _acceptance(Path(candidate_dir).resolve(), case, station, Path(in_dir).resolve(), out_dir)


def final(candidate_dir: Path, case: str, method: str, in_dir: Path, out_dir: Path,
          *, markitect_cmd: list[str] | None = None, station: int | None = None) -> dict:
    """Checks of the last station, the candidate's own tests and, for Markitect, conformance.

    `passed`/`total`/`status` come from the public checks only; own tests and
    conformance are recorded beside them and do not change that verdict.
    `station` defaults to the last station of the case's STATIONS.json.
    `markitect_cmd` overrides the product command (tests use a fake).
    """
    candidate_dir, in_dir, out_dir = Path(candidate_dir).resolve(), Path(in_dir).resolve(), Path(out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    if station is None:
        station = len(load_station_plan(in_dir / "cases" / case / "STATIONS.json", case))
    checks = _acceptance(candidate_dir, case, station, in_dir, out_dir)
    result = {"passed": checks["passed"], "total": checks["total"], "status": checks["status"],
              "method": method, "checks": checks, "ownTests": _own_tests(candidate_dir, out_dir),
              "conformance": None}
    if method == "markitect":
        result["conformance"] = _conformance(candidate_dir, in_dir, out_dir, markitect_cmd)
    return result


def scratch_copy(source: Path, target: Path) -> Path:
    """Copy `source` to `target` without any `.git` entry; symlinks stay links."""
    shutil.copytree(source, target, symlinks=True,
                    ignore=lambda _dir, names: [n for n in names if n == ".git"])
    return target


@contextmanager
def _scratch(candidate_dir: Path) -> Iterator[tuple[Path, dict]]:
    """A scratch copy of the candidate plus the environment every check runs with."""
    root = Path(tempfile.mkdtemp(prefix="mpg-assess-"))
    try:
        copy = scratch_copy(candidate_dir, root / "candidate")
        home = root / "home"
        (home / "tmp").mkdir(parents=True)
        codex_agent.give_to_agent(root, recursive=True)
        env = {"HOME": str(home), "TMPDIR": str(home / "tmp"), "CODEX_HOME": str(home / ".codex"),
               "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1"}
        yield copy, env
    finally:
        _rmtree(root)


def _run(cmd: list[str], cwd: Path, timeout: float, out_dir: Path, name: str, env: dict) -> dict:
    stdout, stderr = out_dir / f"{name}.stdout.txt", out_dir / f"{name}.stderr.txt"
    run = codex_agent.run_as_agent(cmd, cwd, timeout, stdout, stderr, extra_env=env)
    # No agent runs during assessment; anything the checked code left behind is swept
    # here so it is not counted against (or interfering with) the next station.
    leftovers = codex_agent.kill_all_agent_processes() + (run.get("killedOnTimeout") or 0)
    return {"exitCode": run.get("exitCode"), "timedOut": bool(run.get("timedOut")),
            "seconds": run.get("seconds"), "stdout": stdout.name, "stderr": stderr.name,
            "leftoverProcessesKilled": leftovers}


def _acceptance(candidate_dir: Path, case: str, number: int, in_dir: Path, out_dir: Path) -> dict:
    result: dict[str, Any] = {"case": case, "station": number, "passed": None, "total": None,
                              "status": "error", "byStatus": {}, "findings": [], "error": None}
    if not candidate_dir.is_dir():
        result["error"] = f"candidate directory missing: {candidate_dir}"
        return result
    script = in_dir / "cases" / "common" / "checks" / "acceptance.py"
    with _scratch(candidate_dir) as (copy, env):
        run = _run([sys.executable, "-I", "-B", str(script), "--repo", str(copy), "--case", case,
                    "--station", str(number)], copy, ACCEPTANCE_TIMEOUT, out_dir, "acceptance", env)
    result.update(run)
    try:
        findings = json.loads(_read(out_dir / run["stdout"]))["findings"]
        statuses = [item["status"] for item in findings]
    except (ValueError, KeyError, TypeError):
        result["error"] = "acceptance timed out" if run["timedOut"] else "acceptance output could not be parsed"
        return result
    result["findings"] = findings
    result["byStatus"] = {name: statuses.count(name) for name in sorted(set(statuses))}
    result["passed"], result["total"] = statuses.count("PASS"), len(statuses)
    result["status"] = "pass" if run["exitCode"] == 0 and result["passed"] == result["total"] else "fail"
    return result


def _own_tests(candidate_dir: Path, out_dir: Path) -> dict:
    """`unittest discover` from the root, plus a non-package `tests/` folder, which
    root discovery does not enter. Recorded, not judged."""
    runs = []
    with _scratch(candidate_dir) as (copy, env):
        starts = ["."]
        if (copy / "tests").is_dir() and not (copy / "tests" / "__init__.py").exists():
            starts.append("tests")
        for index, start in enumerate(starts, 1):
            # -s -E (not -I): the repo root must stay importable for its tests.
            run = _run([sys.executable, "-B", "-s", "-E", "-m", "unittest", "discover", "-s", start], copy,
                       OWN_TESTS_TIMEOUT, out_dir, f"own-tests-{index}", env)
            runs.append({"start": start, **run, **_unittest_counts(_read(out_dir / run["stderr"]))})
    summary: dict[str, Any] = {"runs": runs}
    for key in ("ran", "failures", "errors", "skipped"):
        values = [run[key] for run in runs]
        summary[key] = None if None in values else sum(values)
    if any(run["timedOut"] or run["ran"] is None for run in runs):
        summary["status"] = "error"
    elif summary["failures"] or summary["errors"] or any(run["exitCode"] not in (0, 5) for run in runs):
        summary["status"] = "fail"  # exit 5 means "no tests ran" (Python 3.12+)
    else:
        summary["status"] = "pass" if summary["ran"] else "no-tests"
    return summary


def _unittest_counts(text: str) -> dict:
    ran = re.findall(r"^Ran (\d+) tests? in", text, re.M)
    if not ran:
        return {"ran": None, "failures": None, "errors": None, "skipped": None}
    counts = {"ran": int(ran[-1]), "failures": 0, "errors": 0, "skipped": 0}
    summary = re.findall(r"^(?:OK|FAILED|NO TESTS RAN)(?: \(([^)]*)\))?\s*$", text, re.M)
    for part in (summary[-1] if summary else "").split(","):
        name, _, value = part.strip().partition("=")
        if name in counts and value.isdigit():
            counts[name] = int(value)
    return counts


def _conformance(candidate_dir: Path, in_dir: Path, out_dir: Path, markitect_cmd: list[str] | None) -> dict:
    """`markitect project check` on a scratch Git repository holding the frozen files
    (the product reads the project through Git). `errorSource` says whose failure an
    `error` is: "harness" (binary or scratch repository) or "product" (the command)."""
    result: dict[str, Any] = {"command": "markitect project check", "status": "error", "exitCode": None,
                              "reportStatus": None, "coverageConforming": None, "findings": None, "error": None,
                              "errorSource": None}
    command = markitect_cmd or _markitect_command(in_dir)
    if not command:
        result.update(error="Markitect binary not found", errorSource="harness")
        return result
    with _scratch(candidate_dir) as (copy, env):
        for index, args in enumerate((["init", "-q", "--initial-branch=main", "."], ["add", "-A"],
                                      ["commit", "-q", "--allow-empty", "-m", "Frozen candidate"]), 1):
            git = _run([*SCRATCH_GIT, *args], copy, GIT_TIMEOUT, out_dir, f"conformance-git-{index}", env)
            if git["exitCode"] != 0:
                result.update(error=f"scratch git {args[0]} failed", errorSource="harness")
                return result
        run = _run([*command, "project", "check", "--repo", str(copy)], copy, PRODUCT_TIMEOUT,
                   out_dir, "conformance", env)
    result.update(run)
    if run["timedOut"] or run["exitCode"] is None:
        result.update(error="markitect project check did not finish", errorSource="product")
        return result
    try:
        report = json.loads(_read(out_dir / run["stdout"]))
        result.update(reportStatus=report.get("status"), findings=len(report.get("findings") or []),
                      coverageConforming=(report.get("coverage") or {}).get("conforming"))
    except (ValueError, AttributeError):
        result.update(error=_read(out_dir / run["stderr"]).strip()[:2000] or "no JSON report", errorSource="product")
    result["status"] = "pass" if run["exitCode"] == 0 else "fail"
    return result


def _markitect_command(in_dir: Path) -> list[str] | None:
    for path in (MARKITECT_INSTALLED, in_dir / "bin" / "markitect"):
        if path.is_file():
            return [str(path)]
    found = shutil.which("markitect")
    return [found] if found else None


def _rmtree(path: Path) -> None:
    """Remove a scratch tree, including read-only Git objects on Windows."""
    def retry(func, target, _error):
        os.chmod(target, stat.S_IWRITE)
        func(target)
    try:
        if sys.version_info >= (3, 12):
            shutil.rmtree(path, onexc=retry)
        else:
            shutil.rmtree(path, onerror=retry)
    except OSError:
        pass
