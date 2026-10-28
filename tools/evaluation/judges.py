"""Judge sources: export a judge folder from a frozen revision, test a judge on its references and mutants, write reference starts.

  python -I -B judges.py export --source DIR [--rev REV] --out NEW_DIR
  python -I -B judges.py selftest --source DIR --judge JUDGE_DIR [--task TASK_DIR] --out NEW_DIR [--case NAME ...] [--engine BIN] [--lock FILE]
  python -I -B judges.py start --source DIR --step STEP [--base BUNDLE] --out NEW_BUNDLE
  python -I -B judges.py plant --plant FILE --step-out NEW_DIR [--judge JUDGE_DIR] [--judge-out NEW_DIR]

A judge source (for example tools/evaluation-judges/library-m1/) holds the judge's
`judge.json` with an `include` map from each file of the judge folder to its
path in this repository, its reference results and seeded `mutants/`.
`export` writes the self-contained judge folder `suite.py judge` runs: each
included file taken from REV (default: the working tree) and its SHA-256
recorded under `frozen`.

A single-step judge has one reference result tree, `reference/`. A series
judge names its `steps` and, per step, its reference result: `references`
maps each step to {"tree": REPO_PATH, "at": PREFIX}, a tree placed at PREFIX
in the result. It may name `patchMutants`, a folder of one subfolder per
oracle family with `STEP` and `mutant.patch` (a patch of that step's tree);
each such mutant must fail at least one case of its family.

A judge is tested on results built on a task's start: the reference must pass
every area (for a series judge, every step's reference at its step, with no
drift), and each mutant must fail where it is seeded. A mutant is a JSON file:
an optional `step`, `replace` edits of the reference ({"path", "old", "new"},
each old text found exactly once), `write` files ({path: content}) and
`expect`, any of "correctnessFails": true, "buildFails": true,
"unrequested": [paths], "protected": [paths], and for a series judge
"forgotten": true, "stale": true and "unaccounted": [paths]. A single-step
judge is tested on `--task`; a series judge builds one task per step, each
from a fixed reference start: the reference result of the step before.

`start` writes such a fixed reference start: the reference result of STEP as
one commit on BASE (default: the suite's empty start), in a one-branch bundle.
It is the start of the step after STEP when that step runs as an independent
task.

`plant` writes a planted task (tools/evaluation-judges/plants/NAME.json): the
series step (series.py) with the plant's edits applied, each exactly once, to
both task.md and the model, so that every variant sees the same defect or gap;
and its judge folder, the exported series judge with the plant record as
plant.json. Run it from the fixed reference start of the step before.
"""
import argparse
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile

# Isolated mode (-I) leaves the script's directory off the path.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import judging  # noqa: E402
import sandbox as sandboxing  # noqa: E402
import series  # noqa: E402
import suite  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
IDENTITY = ("Judge Self-Test", "judge@markitect.invalid")
DATE = "2026-10-28T00:00:00Z"
# Keys of a judge source that describe its tests, not the judge the suite runs.
SOURCE_ONLY = ("include", "references", "patchMutants")


def source_config(source):
    return json.loads((pathlib.Path(source) / "judge.json").read_text(encoding="utf-8"))


def export(source, out, rev=None, repo=REPO):
    source, out = pathlib.Path(source), pathlib.Path(out)
    if out.exists():
        raise judging.JudgeError(f"{out} exists")
    config = source_config(source)
    include = config.get("include")
    for key in SOURCE_ONLY:
        config.pop(key, None)
    if not isinstance(include, dict) or not include:
        raise judging.JudgeError(f"{source}/judge.json has no include map")
    commit = suite.git(repo, "rev-parse", rev + "^{commit}") if rev else None
    files = {}
    for dest, src in sorted(include.items()):
        if commit:
            p = subprocess.run(["git", "-C", str(repo), "show", f"{commit}:{src}"], capture_output=True)
            if p.returncode != 0:
                raise judging.JudgeError(f"{src} is not in {commit}")
            data = p.stdout
        else:
            data = (repo / src).read_bytes()
        target = out / dest
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
        files[dest] = hashlib.sha256(data).hexdigest()
    config["frozen"] = {"rev": commit or "working tree", "files": files}
    (out / "judge.json").write_text(json.dumps(config, indent=2) + "\n", encoding="utf-8")
    judging.load(out)
    return config


def references(source, repo=REPO):
    """Each step's reference result: {step: (tree, prefix)}; a single-step judge has the one step None."""
    source, config = pathlib.Path(source), source_config(source)
    if not config.get("steps"):
        return {None: (source / "reference", "")}
    refs = config.get("references") or {}
    if set(refs) != set(config["steps"]):
        raise judging.JudgeError(f"{source}/judge.json needs one reference per step")
    return {step: (repo / refs[step]["tree"], refs[step].get("at", "")) for step in config["steps"]}


def mutants(source, repo=REPO):
    """Every seeded mutant: (name, mutant), the JSON mutants first, then the oracle-family patch mutants."""
    source, config = pathlib.Path(source), source_config(source)
    found = [(p.stem, json.loads(p.read_text(encoding="utf-8"))) for p in sorted((source / "mutants").glob("*.json"))]
    if config.get("patchMutants"):
        for folder in sorted(p for p in (repo / config["patchMutants"]).iterdir() if (p / "mutant.patch").is_file()):
            step = (folder / "STEP").read_text(encoding="utf-8").strip().upper()
            found.append((f"family-{folder.name}", {"description": f"The calibration mutant of the oracle family {folder.name}.",
                                                    "step": step, "patch": str(folder / "mutant.patch"),
                                                    "expect": {"correctnessFails": True, "failedCasePrefix": folder.name}}))
    return found


def commit_env():
    name, email = IDENTITY
    return dict(os.environ, GIT_AUTHOR_NAME=name, GIT_AUTHOR_EMAIL=email, GIT_COMMITTER_NAME=name, GIT_COMMITTER_EMAIL=email,
                GIT_AUTHOR_DATE=DATE, GIT_COMMITTER_DATE=DATE, GIT_CONFIG_NOSYSTEM="1")


def place(repo, tree, at):
    """Put the reference tree at its prefix; what it replaces goes first, so files a step deleted are gone."""
    target = repo / at if at else repo
    for top in sorted(p.name for p in tree.iterdir()):
        old = target / top
        if old.is_dir():
            shutil.rmtree(old)
        elif old.exists():
            old.unlink()
    for p in sorted(tree.rglob("*")):
        if p.is_file():
            dest = target / p.relative_to(tree)
            dest.parent.mkdir(parents=True, exist_ok=True)
            # Byte-exact: a checkout with line-ending conversion must not reach the result.
            dest.write_bytes(p.read_bytes().replace(b"\r\n", b"\n") if p.suffix in (".go", ".md", ".mod") else p.read_bytes())


def mutate(repo, mutant, at):
    for edit in mutant.get("replace", []):
        path = repo / edit["path"]
        text = path.read_text(encoding="utf-8")
        if text.count(edit["old"]) != 1:
            raise judging.JudgeError(f"{edit['path']}: the mutant's text occurs {text.count(edit['old'])} times, not once")
        path.write_text(text.replace(edit["old"], edit["new"]), encoding="utf-8", newline="\n")
    for rel, content in mutant.get("write", {}).items():
        path = repo / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8", newline="\n")
    if mutant.get("patch"):
        patch = pathlib.Path(mutant["patch"]).read_bytes().replace(b"\r\n", b"\n")
        args = ["git", "-C", str(repo), "apply", "-p1"] + ([f"--directory={at}"] if at else []) + ["-"]
        p = subprocess.run(args, input=patch, capture_output=True)
        if p.returncode != 0:
            raise judging.JudgeError(f"{mutant['patch']} does not apply: {p.stderr.decode(errors='replace').strip()}")


def materialize(source, mutant, task_dir, bundle, step=None):
    """The step's reference result, with the mutant's edits, as one commit on the task's start in a one-branch bundle."""
    task = suite.read_task(task_dir)
    tree, at = references(source)[step]
    return commit_tree(pathlib.Path(task_dir) / "start.bundle", task["start"]["commit"], tree, at, mutant, bundle, "Judge self-test result")


def commit_tree(start_bundle, start, tree, at, mutant, bundle, message):
    env = commit_env()
    with tempfile.TemporaryDirectory() as tmp:
        repo = pathlib.Path(tmp) / "repo"
        suite.git(tmp, "clone", "-q", "--no-checkout", str(pathlib.Path(start_bundle).resolve()), str(repo))
        suite.git(repo, "-c", "core.autocrlf=false", "checkout", "-q", "-B", "result", start)
        place(repo, tree, at)
        mutate(repo, mutant or {}, at)
        suite.git(repo, "-c", "core.autocrlf=false", "add", "-A", env=env)
        suite.git(repo, "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", message, env=env)
        commit = suite.git(repo, "rev-parse", "HEAD")
        suite.git(repo, "bundle", "create", str(pathlib.Path(bundle).resolve()), "refs/heads/result")
    return commit


def start(source, step, out, base=None):
    """The reference result of STEP as one commit on BASE (default: the suite's empty start): a fixed reference start."""
    out = pathlib.Path(out)
    if out.exists():
        raise judging.JudgeError(f"{out} exists")
    refs = references(source)
    if step not in refs:
        raise judging.JudgeError(f"{step} is not a step of {source}")
    tree, at = refs[step]
    with tempfile.TemporaryDirectory() as tmp:
        if base is None:
            base = pathlib.Path(tmp) / "empty.bundle"
            suite.empty_start(base)
        commit = commit_tree(base, suite.bundle_head(base), tree, at, None, pathlib.Path(tmp) / "result.bundle", f"Reference result of {step}")
        # A start bundle carries its one commit on the branch `start`, like the suite's own starts.
        with tempfile.TemporaryDirectory() as repo:
            suite.git(repo, "init", "-q", "--bare")
            suite.git(repo, "fetch", "-q", str(pathlib.Path(tmp) / "result.bundle"), f"+{commit}:refs/heads/start")
            suite.git(repo, "bundle", "create", str(out.resolve()), "refs/heads/start")
    return {"step": step, "commit": commit, "bundle": str(out)}


def verdict(record, mutant, series=False):
    """Why the judgment does not show what the reference or the mutant should show; empty when it does."""
    if not record["valid"]:
        return ["the judgment is not valid: " + "; ".join(record["problems"])]
    areas = record["areas"]
    correctness, build, tests, scope = (areas["correctness"], areas["buildAndTests"]["build"], areas["buildAndTests"]["tests"],
                                        areas["scope"])
    drift = areas.get("drift") or {}
    if mutant is None:
        wrong = []
        if correctness["failed"]:
            wrong.append("cases failed: " + ", ".join(correctness["failedCases"]))
        if not build["passed"] or not tests["passed"]:
            wrong.append("its own build or tests failed")
        if scope["unrequested"] or scope["protectedTouched"]:
            wrong.append("scope findings: " + ", ".join(scope["unrequested"] + scope["protectedTouched"]))
        if series and (drift.get("forgottenRules") or drift.get("staleFiles") or drift.get("unaccounted") != []):
            wrong.append("drift findings: " + json.dumps(drift))
        return wrong
    expect, wrong = mutant.get("expect") or {}, []
    if not expect:
        wrong.append("the mutant expects nothing")
    if expect.get("correctnessFails") and not correctness["failed"]:
        wrong.append("every case passed")
    if expect.get("failedCasePrefix") and not any(c.startswith(expect["failedCasePrefix"]) for c in correctness["failedCases"]):
        wrong.append(f"no case of {expect['failedCasePrefix']} failed")
    if expect.get("buildFails") and build["passed"]:
        wrong.append("the build passed")
    wrong += [f"{p} is not unrequested" for p in expect.get("unrequested", []) if p not in scope["unrequested"]]
    wrong += [f"{p} is not a protected input touched" for p in expect.get("protected", []) if p not in scope["protectedTouched"]]
    if expect.get("forgotten") and not drift.get("forgottenRules"):
        wrong.append("no earlier step's case failed")
    if expect.get("stale") and not drift.get("staleFiles"):
        wrong.append("no stale file was found")
    wrong += [f"{p} is not unaccounted" for p in expect.get("unaccounted", []) if p not in (drift.get("unaccounted") or [])]
    return wrong


def series_tasks(source, out):
    """One task per step, each from a fixed reference start: an empty start for the first, the step before's reference after."""
    steps, tasks = source_config(source)["steps"], {}
    for i, step in enumerate(steps):
        folder = out / "tasks" / step
        folder.mkdir(parents=True)
        (folder / "task.md").write_text(f"Judge self-test of step {step}.\n", encoding="utf-8")
        bundle = None
        if i:
            bundle = folder / "start.bundle"
            start(source, steps[i - 1], bundle)
        suite.make_task(folder / "task.md", None, bundle, 600, folder / "task")
        tasks[step] = folder / "task"
    return tasks


def selftest(source, judge_dir, task_dir, out, engine, only=None):
    source, out = pathlib.Path(source), pathlib.Path(out)
    if out.exists():
        raise judging.JudgeError(f"{out} exists")
    steps = source_config(source).get("steps")
    if not steps and not task_dir:
        raise judging.JudgeError("a single-step judge is tested on --task")
    out.mkdir(parents=True)
    cases = [(f"reference-{s}" if s else "reference", None, s) for s in (steps or [None])]
    cases += [(name, m, m.get("step") or (steps[-1] if steps else None)) for name, m in mutants(source)]
    if only:
        unknown = set(only) - {name for name, _, _ in cases}
        if unknown:
            raise judging.JudgeError("no such self-test case: " + ", ".join(sorted(unknown)))
        cases = [c for c in cases if c[0] in only]
    tasks = series_tasks(source, out) if steps else {None: pathlib.Path(task_dir)}
    summary = {"format": judging.FORMAT, "judge": str(pathlib.Path(judge_dir).resolve()), "judgeSha256": judging.folder_digest(judge_dir),
               "tasks": {str(k): str(v.resolve()) for k, v in tasks.items()}, "cases": []}
    for name, mutant, step in cases:
        (out / name).mkdir()
        commit = materialize(source, mutant, tasks[step], out / name / "result.bundle", step)
        record = judging.run_judge(engine, judge_dir, tasks[step], out / name / "result.bundle", commit, step)
        (out / name / "judgment.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
        wrong = verdict(record, mutant, bool(steps))
        summary["cases"].append({"case": name, "step": step, "expected": "pass" if mutant is None else mutant.get("expect"),
                                 "ok": not wrong, "wrong": wrong})
    summary["passed"] = all(c["ok"] for c in summary["cases"])
    (out / "selftest.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    return summary


def edit_once(path, old, new):
    text = path.read_text(encoding="utf-8")
    if text.count(old) != 1:
        raise judging.JudgeError(f"{path}: the plant's text occurs {text.count(old)} times, not once: {old[:60]!r}")
    path.write_text(text.replace(old, new), encoding="utf-8", newline="\n")


def plant(plant_file, step_out, judge=None, judge_out=None):
    """A planted task: the series step with the plant's edits in both its task.md and its model, and its judge folder.

    The judge folder is the exported series judge (when given) plus plant.json, so a harmless gap's landed change
    is judged at its step like any other while the suite scores the outcome."""
    plant_file, step_out = pathlib.Path(plant_file), pathlib.Path(step_out)
    record = json.loads(plant_file.read_text(encoding="utf-8"))
    if not record.get("model") or not record.get("task"):
        raise judging.JudgeError(f"{plant_file}: a plant edits both the model and task.md, so every variant sees the same defect")
    with tempfile.TemporaryDirectory() as tmp:
        shutil.copyfile(plant_file, pathlib.Path(tmp) / "plant.json")
        judging.load_plant(tmp)
    try:
        series.write_step(record["series"], record["step"], step_out)
    except series.SeriesError as e:
        raise judging.JudgeError(str(e))
    for edit in record["model"]:
        edit_once(step_out / "model" / edit["path"], edit["old"], edit["new"])
    for edit in record["task"]:
        edit_once(step_out / "task.md", edit["old"], edit["new"])
    if judge_out:
        judge_out = pathlib.Path(judge_out)
        if judge_out.exists():
            raise judging.JudgeError(f"{judge_out} exists")
        if judge:
            if judging.load(judge)["steps"] and record["step"] not in judging.load(judge)["steps"]:
                raise judging.JudgeError(f"{judge} does not judge the step {record['step']}")
            shutil.copytree(judge, judge_out)
        else:
            judge_out.mkdir(parents=True)
        shutil.copyfile(plant_file, judge_out / "plant.json")
    return {"step": record["step"], "kind": record["kind"], "expected": record["expected"], "stepDir": str(step_out),
            "judge": str(judge_out) if judge_out else None}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
    e = sub.add_parser("export")
    e.add_argument("--source", required=True)
    e.add_argument("--rev", help="the frozen revision to take every included file from (default: the working tree)")
    e.add_argument("--out", required=True)
    s = sub.add_parser("selftest")
    for name in ("--source", "--judge", "--out"):
        s.add_argument(name, required=True)
    s.add_argument("--task", help="the task a single-step judge is tested on; a series judge builds its own")
    s.add_argument("--case", action="append", help="run only this self-test case (repeatable)")
    s.add_argument("--engine", default=os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker")
    s.add_argument("--lock", help="exclusive engine lock file shared with the other runners")
    t = sub.add_parser("start")
    t.add_argument("--source", required=True)
    t.add_argument("--step", required=True)
    t.add_argument("--base", help="the bundle to build on (default: the suite's empty start)")
    t.add_argument("--out", required=True)
    n = sub.add_parser("plant")
    n.add_argument("--plant", required=True, help="a planted task record (tools/evaluation-judges/plants/NAME.json)")
    n.add_argument("--step-out", required=True, help="the planted chain step to write: task.md and model/")
    n.add_argument("--judge", help="the exported series judge to copy into the judge folder")
    n.add_argument("--judge-out", help="the judge folder to write: the judge, if given, and plant.json")
    a = p.parse_args(argv)
    try:
        if a.cmd == "export":
            result = export(a.source, a.out, a.rev)
        elif a.cmd == "plant":
            result = plant(a.plant, a.step_out, a.judge, a.judge_out)
        elif a.cmd == "start":
            result = start(a.source, a.step, a.out, a.base)
        else:
            with sandboxing.engine_lock(a.lock, "judge self-test", str(pathlib.Path(a.out).resolve())):
                result = selftest(a.source, a.judge, a.task, a.out, [a.engine], a.case)
    except (judging.JudgeError, suite.SuiteError, sandboxing.SandboxError) as e:
        print(f"judges: {e}", file=sys.stderr)
        return 2
    print(json.dumps(result if a.cmd != "selftest" else {"passed": result["passed"], "cases": result["cases"]}, indent=2))
    return 0 if a.cmd != "selftest" or result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
