"""The evaluation suite: build a task, run one variant on it, collect the result.

A task is a directory: task.md (what to build), an optional model/ (the same
intent as a canonical model), start.bundle (the repository to start from) and
task.json (the start commit, the deadline and a digest over the rest). A
variant is any command; the suite appends TASK_DIR OUT_DIR [STATE_DIR], hands it
the evaluation's model in EVALUATION_MODEL, runs it under the deadline and
records what it measured itself in suite.json. How the variant works is the
variant's business.

  python -I -B suite.py task --task-md FILE [--model DIR] [--start BUNDLE] --deadline-seconds N --out NEW_DIR
  python -I -B suite.py run --task DIR --out NEW_DIR --model PROVIDER:MODEL_ID [--state DIR] -- VARIANT...
  python -I -B suite.py chain --step DIR [--step DIR ...] [--start BUNDLE] --deadline-seconds N --model PROVIDER:MODEL_ID \
      --out NEW_DIR -- VARIANT...

A chain step directory holds task.md and an optional model/. Each step starts
from the variant's own previous result, or from the previous start when there
was none; one state directory is handed to the variant at every step.
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
    if proc.poll() is not None:
        return
    if os.name == "nt":
        subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
    else:
        os.killpg(proc.pid, signal.SIGKILL)
    try:
        proc.wait(timeout=KILL_SECONDS)
    except subprocess.TimeoutExpired:
        pass


def start_variant(argv, env, stdout, stderr, deadline):
    kw = {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP} if os.name == "nt" else {"start_new_session": True}
    started = time.monotonic()
    with open(stdout, "wb") as out, open(stderr, "wb") as err:
        proc = subprocess.Popen(argv, env=env, stdout=out, stderr=err, stdin=subprocess.DEVNULL, **kw)
        try:
            code, timed_out = proc.wait(timeout=deadline), False
        except subprocess.TimeoutExpired:
            kill_tree(proc)
            code, timed_out = proc.returncode, True
    return {"exitCode": code, "timedOut": timed_out, "wallSeconds": round(time.monotonic() - started, 1)}


def collect_result(task_dir, start_commit, bundle):
    """The candidate a result bundle carries, checked against the task's start."""
    if not bundle.is_file():
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
    if not path.is_file():
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


def run(task_dir, variant, out, model, state=None):
    """Run one variant on one task under the evaluation's model. Writes OUT/suite.json; the variant writes OUT/variant/."""
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
              "variant": {"argv": list(variant)}, "model": model, "startedUtc": now()}
    with tempfile.TemporaryDirectory() as tmp:
        # The variant works on its own copy; the task stays untouched for the evaluation.
        copy = pathlib.Path(tmp) / "task"
        shutil.copytree(task_dir, copy)
        argv = [*variant, str(copy), str((out / "variant").resolve())] + ([str(pathlib.Path(state).resolve())] if state else [])
        env = dict(os.environ, EVALUATION_MODEL=model)
        record.update(start_variant(argv, env, out / "variant.stdout.log", out / "variant.stderr.log", task["deadlineSeconds"]))
    record["endedUtc"] = now()
    record["result"], record["resultError"] = collect_result(task_dir, task["start"]["commit"], out / "variant" / "result.bundle")
    record["delivered"] = record["result"] is not None
    record["reported"] = reported(out / "variant" / "run.json")
    record["modelMatches"] = model_check(model, record["reported"])
    (out / "suite.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return record


def chain_start(task_dir, step_record, out):
    """The next step's start: this step's result, or this step's start when there was none, as one complete bundle."""
    commit = step_record["result"]["commit"] if step_record["delivered"] else step_record["task"]["start"]
    with tempfile.TemporaryDirectory() as tmp:
        git(tmp, "init", "-q", "--bare")
        git(tmp, "fetch", "-q", str((pathlib.Path(task_dir) / "start.bundle").resolve()), "+refs/*:refs/suite/start/*")
        if step_record["delivered"]:
            result = pathlib.Path(step_record["resultBundle"])
            git(tmp, "fetch", "-q", str(result.resolve()), f"+{commit}:refs/suite/result")
        git(tmp, "update-ref", "refs/heads/start", commit)
        git(tmp, "bundle", "create", str(pathlib.Path(out).resolve()), "refs/heads/start")


def chain(steps, start, deadline, variant, out, model):
    out = pathlib.Path(out)
    if out.exists():
        raise SuiteError(f"{out} exists")
    if not steps:
        raise SuiteError("a chain needs at least one step")
    out.mkdir(parents=True)
    state = out / "state"
    state.mkdir()
    records = []
    for i, step in enumerate(steps, 1):
        step = pathlib.Path(step)
        task_dir, run_dir = out / f"step-{i:02d}" / "task", out / f"step-{i:02d}" / "run"
        if i == 1:
            make_task(step / "task.md", step / "model" if (step / "model").is_dir() else None, start, deadline, task_dir)
        else:
            with tempfile.TemporaryDirectory() as tmp:
                bundle = pathlib.Path(tmp) / "start.bundle"
                chain_start(out / f"step-{i - 1:02d}" / "task", records[-1], bundle)
                make_task(step / "task.md", step / "model" if (step / "model").is_dir() else None, bundle, deadline, task_dir)
        record = run(task_dir, variant, run_dir, model, state)
        record["resultBundle"] = str(run_dir / "variant" / "result.bundle")
        records.append(record)
    summary = {"format": FORMAT, "model": model,
               "steps": [{"step": i, "delivered": r["delivered"], "timedOut": r["timedOut"], "wallSeconds": r["wallSeconds"],
                          "modelMatches": r["modelMatches"], "result": r["result"]["commit"] if r["result"] else None}
                         for i, r in enumerate(records, 1)]}
    (out / "chain.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    return summary


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
    a = parser.parse_args(argv)
    try:
        if a.cmd == "task":
            result = make_task(a.task_md, a.model, a.start, a.deadline_seconds, a.out)
        elif a.cmd == "run":
            record = run(a.task, variant, a.out, a.model, a.state)
            result = {k: record[k] for k in ("delivered", "timedOut", "exitCode", "wallSeconds", "resultError", "modelMatches")}
        else:
            result = chain(a.step, a.start, a.deadline_seconds, variant, a.out, a.model)
    except SuiteError as e:
        print(f"suite: {e}", file=sys.stderr)
        return 2
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
