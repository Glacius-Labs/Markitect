"""The Library M1 judge: the comparison's M1 assessment (tools/comparison/assess_candidate.py) as a judge plug-in.

  python3 -I -B judge.py [--contained] TASK_DIR START_BUNDLE RESULT_BUNDLE OUT_DIR

`suite.py judge` runs it in the judge container without network, with the
task, the start bundle and the result bundle read-only, and reads
OUT_DIR/judge.json. Inside one container it does what assess_candidate.py does
in several:

- Correctness: the unchanged Library oracle (tools/library-oracle, profile m1:
  its cases and the opposite-profile witness) and the six hidden probes of
  tools/comparison/probes.py, through the oracle's transport, against the
  result's own `go build ./cmd/library`;
- Build and tests: the result's `go build ./cmd/library` and `go test ./...`
  in `realization/`, with the comparison's primary time limits;
- Scope: assess_candidate's census of every changed path against the start.
  The task asks for `realization/` only, so a path outside it, and a compiled
  executable anywhere, is unrequested. A changed `task.md`, `TASK.md` or
  Markitect model file touches a protected input, unless it is a copy of the
  task's own model file.

The result's code is agent-produced. It runs as the judge's user, so with
--contained the judge stops every other process in the container and empties
OUT_DIR before it writes judge.json: nothing the result started can write the
judgment. It does so only when the suite's container says so in
EVALUATION_JUDGE_CONTAINER; anywhere else the same call would stop every
process of the user.
"""
import argparse
import hashlib
import json
import os
import pathlib
import shutil
import signal
import subprocess
import sys
import tempfile
import time

HERE = pathlib.Path(__file__).resolve().parent
# Exported, the judge folder holds the tools it wraps; in the repository they sit at its root.
ROOT = HERE if (HERE / "tools").is_dir() else HERE.parents[2]
sys.path[:0] = [str(ROOT / "tools" / "comparison"), str(ROOT / "tools" / "library-oracle")]
import assess as oracle  # noqa: E402  (tools/library-oracle/assess.py)
import assess_candidate  # noqa: E402
import probes  # noqa: E402
from driver import Limits  # noqa: E402

FORMAT = 1
NAME = "library-m1"
PROFILE = "m1"
PROGRAM = "library"
REQUESTED = "realization/"
TASK_INPUTS = ("task.md", "TASK.md")
# The comparison's primary own-check limit; the oracle's limits from its container harness.
BUILD_SECONDS, TEST_SECONDS = 300, 600
LIMITS = {"request_seconds": 2, "process_seconds": 30, "request_bytes": 16384, "output_bytes": 262144}
LOG_TAIL = 4000
WITNESS = "witness-opposite-profile"


def git(repo, *args):
    p = subprocess.run(["git", "-c", "core.autocrlf=false", "-C", str(repo), *args], capture_output=True)
    if p.returncode != 0:
        raise RuntimeError(f"git {' '.join(args[:2])}: {p.stderr.decode(errors='replace').strip()}")
    return p.stdout.decode()


def command(argv, cwd, seconds, env):
    """Run a step of the result's own code in its own process group; the group goes when it ends."""
    kw = {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP} if os.name == "nt" else {"start_new_session": True}
    started = time.monotonic()
    proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, **kw)
    try:
        out, _ = proc.communicate(timeout=seconds)
        timed_out = False
    except subprocess.TimeoutExpired:
        # Also when the step ended but something it started still holds its output.
        timed_out = True
    if os.name == "nt":
        if proc.poll() is None:
            subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
    else:
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    if timed_out:
        try:
            out, _ = proc.communicate(timeout=10)
        except subprocess.TimeoutExpired:
            # A process that left the group still holds the output: give up on reading it.
            proc.kill()
            out = b"(output not read: a process outside the step still held it)"
    log = out.decode(errors="replace")
    return {"exit": None if timed_out else proc.returncode, "timedOut": timed_out, "passed": not timed_out and proc.returncode == 0,
            "seconds": round(time.monotonic() - started, 1), "log": log[-LOG_TAIL:]}


def go_env(tmp):
    env = dict(os.environ)
    env.update(HOME=str(tmp), GOENV="off", GOCACHE=str(tmp / "gocache"), GOPATH=str(tmp / "gopath"), GOTMPDIR=str(tmp),
               GOFLAGS="-buildvcs=false", GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off")
    return env


def checkout(repo, commit, dest):
    """The commit's exact files, without Git."""
    dest.mkdir(parents=True)
    env = dict(os.environ, GIT_INDEX_FILE=str(dest.parent / (dest.name + ".index")))
    for args in (["read-tree", commit], ["checkout-index", "-a", "-f"]):
        p = subprocess.run(["git", "-c", "core.autocrlf=false", "-c", "core.symlinks=false", "--git-dir", str(repo), "--work-tree", str(dest), *args],
                           capture_output=True, env=env)
        if p.returncode != 0:
            raise RuntimeError(f"git {args[0]}: {p.stderr.decode(errors='replace').strip()}")


def frozen_protected():
    """The oracle's protected hashes as exported with the judge, else as the files are now."""
    try:
        files = json.loads((HERE / "judge.json").read_text(encoding="utf-8"))["frozen"]["files"]
    except (OSError, ValueError, KeyError, TypeError):
        return oracle.protected_hashes()
    prefix = "tools/library-oracle/"
    return {k[len(prefix):]: v for k, v in files.items()
            if k.startswith(prefix) and (k[len(prefix):] in ("assess.py", "driver.py", "oracle.py") or k[len(prefix):].startswith("expectations/"))}


def planned_cases():
    return [c["id"] for c in oracle.build_cases(PROFILE)] + [WITNESS] + [c["id"] for c in probes.build_probes()]


def correctness(program, module, tmp):
    """The oracle's cases, its opposite-profile witness and the probes against the built program."""
    cwd = tmp / "assess" / "realization"
    shutil.copytree(module, cwd)
    manifest = {"experiment_id": "evaluation-suite", "artifact_id": NAME, "model_digest": "library m1 (evaluation suite task)",
                "source_tree": str(module), "profile": PROFILE, "command": [str(program)], "cwd": str(cwd),
                "program_sha256": {str(program): hashlib.sha256(program.read_bytes()).hexdigest()},
                "protected_sha256": frozen_protected(), "limits": LIMITS}
    report = oracle.assess(json.loads(json.dumps(manifest)))
    cases = [{"id": c["id"], "passed": c["passed"], "errors": c["errors"][:3]} for c in report["cases"]]
    witness = report.get("discrimination") or {}
    cases.append({"id": WITNESS, "passed": witness.get("expected_mismatch_confirmed") is True, "errors": witness.get("errors", [])[:3]})
    for case in probes.build_probes():
        result = oracle.run_case(case, [str(program)], str(cwd), Limits(**LIMITS))
        cases.append({"id": result["id"], "passed": result["passed"], "errors": result["errors"][:3]})
    return cases, report.get("errors") or []


def model_copy(task, repo, commit, path):
    """Whether a changed Markitect model file is byte-equal to the task's own model file."""
    for prefix, origin in ((".markitect/model/", "model/model/"), (".markitect/model.yaml", "model/model.yaml"),
                           (".markitect/model-lock.yaml", "model/model-lock.yaml")):
        if path == prefix or (prefix.endswith("/") and path.startswith(prefix)):
            original = task / (origin + path[len(prefix):] if prefix.endswith("/") else origin)
            if not original.is_file():
                return False
            try:
                return subprocess.run(["git", "-C", str(repo), "cat-file", "blob", f"{commit}:{path}"], capture_output=True).stdout == original.read_bytes()
            except OSError:
                return False
    return False


def scope(task, repo, start, commit):
    census = assess_candidate.census(repo, start, commit)
    changed, unrequested, protected = [], [], []
    for category, entries in census.items():
        for entry in entries:
            path = entry["path"]
            changed.append({"status": entry["status"], "path": path, "category": category})
            if category == "protected" or path in TASK_INPUTS:
                if not (entry["status"] != "D" and model_copy(task, repo, commit, path)):
                    protected.append(path)
            elif category == "build_residue" or (category == "comparable" and not path.startswith(REQUESTED)):
                unrequested.append(path)
    key = lambda p: p if isinstance(p, str) else p["path"]  # noqa: E731
    return {"changedPaths": sorted(changed, key=key), "unrequested": sorted(unrequested), "protectedTouched": sorted(protected)}, census


def judge(task, start_bundle, result_bundle, tmp):
    spec = json.loads((task / "task.json").read_text(encoding="utf-8"))
    start = spec["start"]["commit"]
    repo = tmp / "repo"
    git(tmp, "init", "-q", "--bare", str(repo))
    git(repo, "fetch", "-q", str(start_bundle), f"+{start}:refs/judge/start")
    heads = {line.split()[0] for line in git(repo, "bundle", "list-heads", str(result_bundle)).splitlines()}
    if len(heads) != 1:
        raise RuntimeError(f"the result bundle carries {len(heads)} commits, not one")
    commit = heads.pop()
    git(repo, "fetch", "-q", str(result_bundle), f"+{commit}:refs/judge/result")
    record = {"format": FORMAT, "judge": NAME, "commit": commit, "start": start}
    record["scope"], census = scope(task, repo, start, commit)
    candidate = tmp / "candidate"
    checkout(repo, commit, candidate)
    module = candidate / REQUESTED.rstrip("/")
    env = go_env(tmp)
    planned = planned_cases()
    oracle_errors = []
    if not (module / "go.mod").is_file():
        missing = {"exit": None, "timedOut": False, "passed": False, "log": "no realization/go.mod"}
        build, tests, cases = missing, missing, [{"id": c, "passed": False, "errors": ["not built"]} for c in planned]
    else:
        program = tmp / "bin" / (PROGRAM + (".exe" if os.name == "nt" else ""))
        shutil.copytree(module, tmp / "build")
        build = command(["go", "build", "-o", str(program), "./cmd/" + PROGRAM], tmp / "build", BUILD_SECONDS, env)
        shutil.copytree(module, tmp / "test")
        tests = command(["go", "test", "./..."], tmp / "test", TEST_SECONDS, env)
        if build["passed"] and program.is_file():
            cases, oracle_errors = correctness(program, module, tmp)
        else:
            cases = [{"id": c, "passed": False, "errors": ["not built"]} for c in planned]
    failed = [c["id"] for c in cases if not c["passed"]]
    record["correctness"] = {"planned": len(planned), "passed": len(cases) - len(failed), "failed": len(failed),
                             "failedCases": failed, "cases": cases}
    record["buildAndTests"] = {"build": build, "tests": tests}
    record["details"] = {"census": census, "scopePatterns": assess_candidate.scope_findings(module) if module.is_dir() else {},
                         "oracleErrors": oracle_errors}
    return record


def stop_everything_else():
    """Kill every other process this user runs in the container: whatever the result started is gone.

    Only in the suite's judge container, which sets EVALUATION_JUDGE_CONTAINER: anywhere else the same call
    would stop every process of the user."""
    if not os.environ.get("EVALUATION_JUDGE_CONTAINER"):
        return
    try:
        os.kill(-1, signal.SIGKILL)
    except (ProcessLookupError, PermissionError):
        pass


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--contained", action="store_true", help="inside the judge container: stop all other processes before writing")
    for name in ("task", "start", "result", "out"):
        p.add_argument(name)
    a = p.parse_args(argv)
    out = pathlib.Path(a.out)
    with tempfile.TemporaryDirectory() as tmp:
        try:
            record = judge(pathlib.Path(a.task), pathlib.Path(a.start).resolve(), pathlib.Path(a.result).resolve(), pathlib.Path(tmp))
        except (OSError, ValueError, KeyError, RuntimeError) as e:
            record = {"format": FORMAT, "judge": NAME, "error": f"{type(e).__name__}: {e}"}
        if a.contained:
            stop_everything_else()
            for entry in out.iterdir():
                shutil.rmtree(entry) if entry.is_dir() and not entry.is_symlink() else entry.unlink()
    (out / "judge.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return 0 if "error" not in record else 1


if __name__ == "__main__":
    sys.exit(main())
