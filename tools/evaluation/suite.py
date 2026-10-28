"""The evaluation suite: build a task, run one variant on it, collect the result.

A task is a directory: task.md (what to build), an optional model/ (the same
intent as a canonical model), start.bundle (the repository to start from) and
task.json (the start commit, the deadline and a digest over the rest). A
variant is any command; the suite appends TASK_DIR OUT_DIR [STATE_DIR], hands it
the evaluation's model in EVALUATION_MODEL, runs it under the deadline and
records what it measured itself in suite.json. How the variant works is the
variant's business.

  python -I -B suite.py task --task-md FILE [--model DIR] [--start BUNDLE] --deadline-seconds N --out NEW_DIR
  python -I -B suite.py run --task DIR --out NEW_DIR --model PROVIDER:MODEL_ID [--state DIR] [SANDBOX...] -- VARIANT...
  python -I -B suite.py chain --step DIR [--step DIR ...] [--start BUNDLE] --deadline-seconds N --model PROVIDER:MODEL_ID \
      --out NEW_DIR [SANDBOX...] -- VARIANT...

  python -I -B suite.py judge --run RUN_DIR --judge JUDGE_DIR --out NEW_DIR [--engine BIN] [--lock FILE]
  python -I -B suite.py review --run RUN_DIR --judge JUDGE_DIR --out NEW_DIR [--model PROVIDER:MODEL_ID] [SANDBOX...]
  python -I -B suite.py report --ledger FILE --out NEW_DIR

  SANDBOX: --sandbox IMAGE_ID --variant-code DIR [--login FILE] [--series FILE] [--engine BIN] [--lock FILE]

A chain step directory holds task.md and an optional model/. Each step starts
from the variant's own previous result, or from the previous start when there
was none; one state directory is handed to the variant at every step.

With --sandbox, the variant command runs inside one container per run
(sandbox.py): it receives /task /out [/state], its code is at /variant and the
access-only login at EVALUATION_LOGIN. Without it, the variant runs as a host
process: for tests only, and suite.json marks the run as not valid for counting.

`judge` runs the task's hidden judge on a run's result in a container without
network (judging.py) and writes judgment.json and the judge's own judge.json.
`review` makes the blinded code review of a result (reviewing.py); its
reviewer sessions run in the sandbox, with --variant-code naming this folder.
`report` writes one table per area for a series ledger (report.py).
"""
import argparse
import datetime
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

# Isolated mode (-I) leaves the script's directory off the path.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import judging  # noqa: E402
import report as reporting  # noqa: E402
import reviewing  # noqa: E402
import sandbox as sandboxing  # noqa: E402

FORMAT = 1
START_IDENTITY = ("Evaluation Suite", "suite@markitect.invalid")
START_DATE = "2026-10-28T00:00:00Z"
KILL_SECONDS = 15


class SuiteError(Exception):
    pass


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def sha256_file(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def git(repo, *args, env=None):
    p = subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=True, env=env)
    if p.returncode != 0:
        raise SuiteError(f"git {' '.join(args)}: {p.stderr.strip()}")
    return p.stdout.strip()


# --- Tasks --------------------------------------------------------------------

def intent_digest(task_dir):
    """One digest over task.md, model/ and start.bundle: paths and bytes."""
    h = hashlib.sha256()
    root = pathlib.Path(task_dir)
    files = [root / "task.md", root / "start.bundle", *sorted(p for p in (root / "model").rglob("*") if p.is_file())]
    for p in files:
        h.update(p.relative_to(root).as_posix().encode() + b"\0" + p.read_bytes() + b"\0")
    return h.hexdigest()


def empty_start(out):
    """A bundle with one empty root commit of fixed identity and date, for a new project."""
    with tempfile.TemporaryDirectory() as tmp:
        git(tmp, "init", "-q", "-b", "start")
        name, email = START_IDENTITY
        env = dict(os.environ, GIT_AUTHOR_NAME=name, GIT_AUTHOR_EMAIL=email, GIT_COMMITTER_NAME=name,
                   GIT_COMMITTER_EMAIL=email, GIT_AUTHOR_DATE=START_DATE, GIT_COMMITTER_DATE=START_DATE)
        git(tmp, "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "Start", env=env)
        git(tmp, "bundle", "create", str(pathlib.Path(out).resolve()), "start")


def bundle_head(bundle):
    """The one commit a complete start bundle carries."""
    with tempfile.TemporaryDirectory() as tmp:
        git(tmp, "init", "-q", "--bare")
        git(tmp, "bundle", "verify", str(pathlib.Path(bundle).resolve()))
        heads = {line.split()[0] for line in git(tmp, "bundle", "list-heads", str(pathlib.Path(bundle).resolve())).splitlines()}
    if len(heads) != 1:
        raise SuiteError(f"{bundle} must carry exactly one commit, it carries {len(heads)}")
    return heads.pop()


def make_task(task_md, model, start, deadline, out):
    out = pathlib.Path(out)
    if out.exists():
        raise SuiteError(f"{out} exists")
    if deadline <= 0:
        raise SuiteError("the deadline must be positive")
    out.mkdir(parents=True)
    shutil.copyfile(task_md, out / "task.md")
    if model:
        shutil.copytree(model, out / "model")
    if start:
        shutil.copyfile(start, out / "start.bundle")
    else:
        empty_start(out / "start.bundle")
    record = {"format": FORMAT, "start": {"commit": bundle_head(out / "start.bundle"), "bundleSha256": sha256_file(out / "start.bundle")},
              "deadlineSeconds": deadline, "intentSha256": intent_digest(out)}
    (out / "task.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return record


def read_task(task_dir):
    task_dir = pathlib.Path(task_dir)
    try:
        record = json.loads((task_dir / "task.json").read_text(encoding="utf-8"))
    except (OSError, ValueError) as e:
        raise SuiteError(f"{task_dir} has no readable task.json: {e}")
    if record.get("format") != FORMAT:
        raise SuiteError(f"{task_dir}: task format {record.get('format')} is not {FORMAT}")
    if intent_digest(task_dir) != record["intentSha256"] or sha256_file(task_dir / "start.bundle") != record["start"]["bundleSha256"]:
        raise SuiteError(f"{task_dir} does not match its task.json")
    return record


# --- Running a variant --------------------------------------------------------

def kill_tree(proc):
    """Kill the variant and everything it started. On POSIX the whole process group goes, also
    after a normal exit; on Windows taskkill follows the tree only while the variant still runs."""
    if os.name == "nt":
        if proc.poll() is None:
            subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
    else:
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    try:
        proc.wait(timeout=KILL_SECONDS)
    except subprocess.TimeoutExpired:
        pass


def start_variant(argv, env, stdout, stderr, deadline, on_deadline=None):
    """Run the variant command under the deadline; on_deadline() runs first when the deadline expires."""
    kw = {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP} if os.name == "nt" else {"start_new_session": True}
    started = time.monotonic()
    with open(stdout, "wb") as out, open(stderr, "wb") as err:
        proc = subprocess.Popen(argv, env=env, stdout=out, stderr=err, stdin=subprocess.DEVNULL, **kw)
        try:
            proc.wait(timeout=deadline)
            timed_out = False
        except subprocess.TimeoutExpired:
            timed_out = True
            if on_deadline:
                on_deadline()
        kill_tree(proc)
    return {"exitCode": proc.returncode, "timedOut": timed_out, "wallSeconds": round(time.monotonic() - started, 1)}


def collect_result(task_dir, start_commit, bundle):
    """The candidate a result bundle carries, checked against the task's start."""
    if bundle.is_symlink() or not bundle.is_file():
        return None, "no result.bundle"
    with tempfile.TemporaryDirectory() as tmp:
        try:
            git(tmp, "init", "-q", "--bare")
            git(tmp, "fetch", "-q", str((pathlib.Path(task_dir) / "start.bundle").resolve()), f"+{start_commit}:refs/suite/start")
            git(tmp, "bundle", "verify", str(bundle.resolve()))
            heads = {line.split()[0] for line in git(tmp, "bundle", "list-heads", str(bundle.resolve())).splitlines()}
            if len(heads) != 1:
                return None, f"result.bundle carries {len(heads)} commits, not one"
            commit = heads.pop()
            git(tmp, "fetch", "-q", str(bundle.resolve()), f"+{commit}:refs/suite/result")
            if subprocess.run(["git", "-C", tmp, "merge-base", "--is-ancestor", start_commit, commit]).returncode != 0:
                return None, f"the result {commit} does not descend from the start {start_commit}"
            changed = git(tmp, "diff", "--name-status", "--no-renames", start_commit, commit).splitlines()
            return {"commit": commit, "tree": git(tmp, "rev-parse", f"{commit}^{{tree}}"), "bundleSha256": sha256_file(bundle),
                    "changedPaths": changed}, None
        except SuiteError as e:
            return None, str(e)


def reported(path):
    if path.is_symlink() or not path.is_file():
        return None
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except ValueError:
        return {"unreadable": True}
    return value if isinstance(value, dict) else {"unreadable": True}


def model_check(model, report):
    """Whether the variant reports the evaluation's model; None when it reports none."""
    if not report or not report.get("provider") or not report.get("model"):
        return None
    return f"{report['provider']}:{report['model']}" == model


def run(task_dir, variant, out, model, state=None, sandbox=None):
    """Run one variant on one task under the evaluation's model, in the sandbox when one is given.

    Writes OUT/suite.json; the variant writes OUT/variant/."""
    task_dir, out = pathlib.Path(task_dir), pathlib.Path(out)
    task = read_task(task_dir)
    if out.exists():
        raise SuiteError(f"{out} exists")
    if not variant:
        raise SuiteError("no variant command")
    if ":" not in model:
        raise SuiteError(f"the model {model!r} must be PROVIDER:MODEL_ID")
    out.mkdir(parents=True)
    (out / "variant").mkdir()
    record = {"format": FORMAT, "task": {"path": str(task_dir.resolve()), "intentSha256": task["intentSha256"], "start": task["start"]["commit"],
                                         "deadlineSeconds": task["deadlineSeconds"]},
              "variant": {"argv": list(variant)}, "model": model, "conditions": "sandbox" if sandbox else "host",
              "startedUtc": now()}
    stdout, stderr, deadline = out / "variant.stdout.log", out / "variant.stderr.log", task["deadlineSeconds"]
    with tempfile.TemporaryDirectory() as tmp:
        # The variant works on its own copy; the task stays untouched for the evaluation.
        copy = pathlib.Path(tmp) / "task"
        shutil.copytree(task_dir, copy)
        if sandbox:
            timing, record["sandbox"] = sandbox.run(
                copy, out / "variant", pathlib.Path(state).resolve() if state else None, variant, model, deadline, out,
                lambda argv, stop: start_variant(argv, dict(os.environ), stdout, stderr, deadline, stop))
            record.update(timing)
        else:
            argv = [*variant, str(copy), str((out / "variant").resolve())] + ([str(pathlib.Path(state).resolve())] if state else [])
            record.update(start_variant(argv, dict(os.environ, EVALUATION_MODEL=model), stdout, stderr, deadline))
    record["endedUtc"] = now()
    # Only a sandboxed run whose sandbox stood up and left nothing behind ran under the evaluation's conditions.
    record["validForCounting"] = (bool(sandbox) and "error" not in record["sandbox"] and "container" in record["sandbox"]
                                  and not record["sandbox"]["residue"])
    try:
        # The variant got a copy, but could still reach the original by path.
        read_task(task_dir)
        record["result"], record["resultError"] = collect_result(task_dir, task["start"]["commit"], out / "variant" / "result.bundle")
    except SuiteError as e:
        record["result"], record["resultError"] = None, f"the task changed during the run: {e}"
    record["delivered"] = record["result"] is not None
    record["reported"] = reported(out / "variant" / "run.json")
    record["modelMatches"] = model_check(model, record["reported"])
    (out / "suite.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return record


def chain_start(task_dir, step_record, out):
    """The next step's start: this step's result, or this step's start when there was none, as one complete bundle."""
    task = read_task(task_dir)
    start = task["start"]["commit"]
    commit = step_record["result"]["commit"] if step_record["delivered"] else start
    with tempfile.TemporaryDirectory() as tmp:
        git(tmp, "init", "-q", "--bare")
        git(tmp, "fetch", "-q", str((pathlib.Path(task_dir) / "start.bundle").resolve()), f"+{start}:refs/suite/start")
        if step_record["delivered"]:
            result = pathlib.Path(step_record["resultBundle"])
            git(tmp, "fetch", "-q", str(result.resolve()), f"+{commit}:refs/suite/result")
        git(tmp, "update-ref", "refs/heads/start", commit)
        git(tmp, "bundle", "create", str(pathlib.Path(out).resolve()), "refs/heads/start")


def chain(steps, start, deadline, variant, out, model, sandbox=None):
    out = pathlib.Path(out)
    if out.exists():
        raise SuiteError(f"{out} exists")
    if not steps:
        raise SuiteError("a chain needs at least one step")
    out.mkdir(parents=True)
    state = out / "state"
    state.mkdir()
    records, error = [], None
    for i, step in enumerate(steps, 1):
        step = pathlib.Path(step)
        task_dir, run_dir = out / f"step-{i:02d}" / "task", out / f"step-{i:02d}" / "run"
        try:
            if i == 1:
                make_task(step / "task.md", step / "model" if (step / "model").is_dir() else None, start, deadline, task_dir)
            else:
                with tempfile.TemporaryDirectory() as tmp:
                    bundle = pathlib.Path(tmp) / "start.bundle"
                    chain_start(out / f"step-{i - 1:02d}" / "task", records[-1], bundle)
                    make_task(step / "task.md", step / "model" if (step / "model").is_dir() else None, bundle, deadline, task_dir)
            record = run(task_dir, variant, run_dir, model, state, sandbox)
        except SuiteError as e:
            # The chain cannot go on, but every step that ran stays on record.
            error = {"step": i, "error": str(e)}
            break
        record["resultBundle"] = str(run_dir / "variant" / "result.bundle")
        records.append(record)
        if "error" in record.get("sandbox", {}):
            # A sandbox that did not stand up is no step of the variant's; the chain stops rather than run on.
            error = {"step": i, "error": "sandbox: " + record["sandbox"]["error"]}
            break
    summary = {"format": FORMAT, "model": model, "conditions": "sandbox" if sandbox else "host",
               "steps": [{"step": i, "delivered": r["delivered"], "timedOut": r["timedOut"], "wallSeconds": r["wallSeconds"],
                          "modelMatches": r["modelMatches"], "validForCounting": r["validForCounting"],
                          "result": r["result"]["commit"] if r["result"] else None}
                         for i, r in enumerate(records, 1)],
               "stopped": error}
    (out / "chain.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    return summary


def judged_run(run_dir):
    """A run's record, its task and its result bundle, checked to be what the run judged."""
    run_dir = pathlib.Path(run_dir)
    try:
        record = json.loads((run_dir / "suite.json").read_text(encoding="utf-8"))
    except (OSError, ValueError) as e:
        raise SuiteError(f"{run_dir} has no readable suite.json: {e}")
    if not record.get("delivered"):
        raise SuiteError(f"{run_dir} delivered no result: there is nothing to judge")
    task_dir = pathlib.Path(record["task"]["path"])
    if read_task(task_dir)["intentSha256"] != record["task"]["intentSha256"]:
        raise SuiteError(f"{task_dir} is not the task the run had")
    bundle = run_dir / "variant" / "result.bundle"
    if bundle.is_symlink() or not bundle.is_file() or sha256_file(bundle) != record["result"]["bundleSha256"]:
        raise SuiteError(f"{bundle} is not the result the run delivered")
    return record, task_dir, bundle


def judge(run_dir, judge_dir, out, engine):
    """Run the task's judge on one run's result; writes OUT/judgment.json and the judge's OUT/judge.json."""
    out = pathlib.Path(out)
    record, task_dir, bundle = judged_run(run_dir)
    if out.exists():
        raise SuiteError(f"{out} exists")
    judgment = judging.run_judge(engine, judge_dir, task_dir, bundle, record["result"]["commit"])
    judgment["run"] = str(pathlib.Path(run_dir).resolve())
    out.mkdir(parents=True)
    output = judgment.pop("output")
    if output is not None:
        (out / "judge.json").write_text(json.dumps(output, indent=2) + "\n", encoding="utf-8")
    (out / "judgment.json").write_text(json.dumps(judgment, indent=2) + "\n", encoding="utf-8")
    return judgment


def sandbox_from(a):
    """The sandbox the arguments name, checked before anything is written; None in host mode."""
    if not a.sandbox:
        if a.variant_code or a.login or a.lock:
            raise SuiteError("--variant-code, --login and --lock need --sandbox")
        return None
    if not a.variant_code:
        raise SuiteError("--sandbox needs --variant-code")
    provider = a.model.partition(":")[0]
    login = a.login or (str(pathlib.Path.home() / ".claude" / ".credentials.json") if provider == "claude" else None)
    if not login:
        raise SuiteError(f"--sandbox needs --login for {provider}")
    return sandboxing.Sandbox([a.engine], a.sandbox, sandboxing.load_series(a.series), a.variant_code, login, provider)


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    variant = []
    if "--" in argv:
        i = argv.index("--")
        argv, variant = argv[:i], argv[i + 1:]
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="cmd", required=True)
    t = sub.add_parser("task")
    t.add_argument("--task-md", required=True)
    t.add_argument("--model")
    t.add_argument("--start")
    t.add_argument("--deadline-seconds", type=int, required=True)
    t.add_argument("--out", required=True)
    r = sub.add_parser("run")
    r.add_argument("--task", required=True)
    r.add_argument("--out", required=True)
    r.add_argument("--state")
    r.add_argument("--model", required=True, help="the evaluation's model as PROVIDER:MODEL_ID")
    c = sub.add_parser("chain")
    c.add_argument("--step", action="append", required=True)
    c.add_argument("--start")
    c.add_argument("--deadline-seconds", type=int, required=True)
    c.add_argument("--out", required=True)
    c.add_argument("--model", required=True, help="the evaluation's model as PROVIDER:MODEL_ID")
    j = sub.add_parser("judge")
    v = sub.add_parser("review")
    for p in (j, v):
        p.add_argument("--run", required=True)
        p.add_argument("--judge", required=True, help="an exported judge folder (judges.py export)")
        p.add_argument("--out", required=True)
    j.add_argument("--engine", default=os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker")
    j.add_argument("--lock", help="exclusive engine lock file shared with the other runners")
    v.add_argument("--model", help="the reviewers' model as PROVIDER:MODEL_ID (default: the run's model)")
    v.add_argument("--deadline-seconds", type=int, default=reviewing.DEADLINE, help="each reviewer session's time")
    o = sub.add_parser("report")
    o.add_argument("--ledger", required=True)
    o.add_argument("--out", required=True)
    for p in (r, c, v):
        p.add_argument("--sandbox", metavar="IMAGE_ID", help="run the variant in the evaluation sandbox of this image")
        p.add_argument("--variant-code", help="the variant's code, mounted read-only at /variant")
        p.add_argument("--login", help="the provider login to copy access-only (default: the user's Claude login)")
        p.add_argument("--series", default=str(sandboxing.DEFAULT_SERIES), help="the series' safety limit and egress")
        p.add_argument("--engine", default=os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker")
        p.add_argument("--lock", help="exclusive engine lock file shared with the other runners")
    a = parser.parse_args(argv)
    try:
        if a.cmd == "task":
            result = make_task(a.task_md, a.model, a.start, a.deadline_seconds, a.out)
        elif a.cmd == "report":
            data = reporting.write(a.ledger, a.out)
            result = {"launches": len(data["launches"]), "report": str(pathlib.Path(a.out) / "report.md")}
        elif a.cmd == "judge":
            with sandboxing.engine_lock(a.lock, "evaluation judge", str(pathlib.Path(a.out).resolve())):
                judgment = judge(a.run, a.judge, a.out, [a.engine])
            result = {"valid": judgment["valid"], "problems": judgment["problems"], "areas": judgment["areas"] and {
                "correctness": {k: judgment["areas"]["correctness"][k] for k in ("planned", "passed", "failed", "failedCases")},
                "build": judgment["areas"]["buildAndTests"]["build"]["passed"], "tests": judgment["areas"]["buildAndTests"]["tests"]["passed"],
                "unrequested": judgment["areas"]["scope"]["unrequested"], "protectedTouched": judgment["areas"]["scope"]["protectedTouched"]}}
        elif a.cmd == "review":
            record, _, _ = judged_run(a.run)
            a.model = a.model or record["model"]
            box = sandbox_from(a)
            reviewer = reviewing.REVIEWER if box else [sys.executable, "-I", "-B", str(pathlib.Path(__file__).resolve().parent / "reviewer.py")]
            with sandboxing.engine_lock(a.lock, "evaluation review", str(pathlib.Path(a.out).resolve())):
                review = reviewing.review(a.run, a.judge, a.out, a.model, box, reviewer, a.deadline_seconds)
            result = {k: review[k] for k in ("valid", "problems", "criticals", "guesses")}
        else:
            box = sandbox_from(a)
            with sandboxing.engine_lock(a.lock, "evaluation suite", str(pathlib.Path(a.out).resolve())):
                if a.cmd == "run":
                    record = run(a.task, variant, a.out, a.model, a.state, box)
                    result = {k: record[k] for k in ("delivered", "timedOut", "exitCode", "wallSeconds", "resultError", "modelMatches",
                                                     "conditions", "validForCounting", "reported")}
                else:
                    result = chain(a.step, a.start, a.deadline_seconds, variant, a.out, a.model, box)
    except (SuiteError, sandboxing.SandboxError, judging.JudgeError, ValueError) as e:
        print(f"suite: {e}", file=sys.stderr)
        return 2
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
