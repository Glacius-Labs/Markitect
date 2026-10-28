"""Judge sources: export a judge folder from a frozen revision, and test a judge on its reference and mutants.

  python -I -B judges.py export --source DIR [--rev REV] --out NEW_DIR
  python -I -B judges.py selftest --source DIR --judge JUDGE_DIR --task TASK_DIR --out NEW_DIR [--engine BIN] [--lock FILE]

A judge source (for example judges/library-m1/) holds the judge's
`judge.json` with an `include` map from each file of the judge folder to its
path in this repository, a `reference/` result tree and seeded `mutants/`.
`export` writes the self-contained judge folder `suite.py judge` runs: each
included file taken from REV (default: the working tree) and its SHA-256
recorded under `frozen`.

A judge is tested on results built from the task's start: the reference must
pass every area, and each mutant must fail where it is seeded. A mutant is a
JSON file: `replace` edits of the reference ({"path", "old", "new"}, each old
text found exactly once), `write` files ({path: content}) and `expect`, any of
"correctnessFails": true, "buildFails": true, "unrequested": [paths] and
"protected": [paths].
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
import suite  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
IDENTITY = ("Judge Self-Test", "judge@markitect.invalid")
DATE = "2026-10-28T00:00:00Z"


def export(source, out, rev=None, repo=REPO):
    source, out = pathlib.Path(source), pathlib.Path(out)
    if out.exists():
        raise judging.JudgeError(f"{out} exists")
    config = json.loads((source / "judge.json").read_text(encoding="utf-8"))
    include = config.pop("include", None)
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


def materialize(source, mutant, task_dir, bundle):
    """The reference result, with the mutant's edits, as one commit on the task's start in a one-branch bundle."""
    task = suite.read_task(task_dir)
    name, email = IDENTITY
    env = dict(os.environ, GIT_AUTHOR_NAME=name, GIT_AUTHOR_EMAIL=email, GIT_COMMITTER_NAME=name, GIT_COMMITTER_EMAIL=email,
               GIT_AUTHOR_DATE=DATE, GIT_COMMITTER_DATE=DATE, GIT_CONFIG_NOSYSTEM="1")
    with tempfile.TemporaryDirectory() as tmp:
        repo = pathlib.Path(tmp) / "repo"
        suite.git(tmp, "clone", "-q", "--no-checkout", str((pathlib.Path(task_dir) / "start.bundle").resolve()), str(repo))
        suite.git(repo, "-c", "core.autocrlf=false", "checkout", "-q", "-B", "result", task["start"]["commit"])
        reference = pathlib.Path(source) / "reference"
        for p in sorted(reference.rglob("*")):
            if p.is_file():
                target = repo / p.relative_to(reference)
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(p, target)
        for edit in (mutant or {}).get("replace", []):
            path = repo / edit["path"]
            text = path.read_text(encoding="utf-8")
            if text.count(edit["old"]) != 1:
                raise judging.JudgeError(f"{edit['path']}: the mutant's text occurs {text.count(edit['old'])} times, not once")
            path.write_text(text.replace(edit["old"], edit["new"]), encoding="utf-8", newline="\n")
        for rel, content in (mutant or {}).get("write", {}).items():
            path = repo / rel
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content, encoding="utf-8", newline="\n")
        suite.git(repo, "-c", "core.autocrlf=false", "add", "-A", env=env)
        suite.git(repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "Judge self-test result", env=env)
        commit = suite.git(repo, "rev-parse", "HEAD")
        suite.git(repo, "bundle", "create", str(pathlib.Path(bundle).resolve()), "refs/heads/result")
    return commit


def verdict(record, mutant):
    """Why the judgment does not show what the reference or the mutant should show; empty when it does."""
    if not record["valid"]:
        return ["the judgment is not valid: " + "; ".join(record["problems"])]
    areas = record["areas"]
    correctness, build, tests, scope = (areas["correctness"], areas["buildAndTests"]["build"], areas["buildAndTests"]["tests"],
                                        areas["scope"])
    if mutant is None:
        wrong = []
        if correctness["failed"]:
            wrong.append("cases failed: " + ", ".join(correctness["failedCases"]))
        if not build["passed"] or not tests["passed"]:
            wrong.append("its own build or tests failed")
        if scope["unrequested"] or scope["protectedTouched"]:
            wrong.append("scope findings: " + ", ".join(scope["unrequested"] + scope["protectedTouched"]))
        return wrong
    expect, wrong = mutant.get("expect") or {}, []
    if not expect:
        wrong.append("the mutant expects nothing")
    if expect.get("correctnessFails") and not correctness["failed"]:
        wrong.append("every case passed")
    if expect.get("buildFails") and build["passed"]:
        wrong.append("the build passed")
    wrong += [f"{p} is not unrequested" for p in expect.get("unrequested", []) if p not in scope["unrequested"]]
    wrong += [f"{p} is not a protected input touched" for p in expect.get("protected", []) if p not in scope["protectedTouched"]]
    return wrong


def selftest(source, judge_dir, task_dir, out, engine):
    source, out = pathlib.Path(source), pathlib.Path(out)
    if out.exists():
        raise judging.JudgeError(f"{out} exists")
    out.mkdir(parents=True)
    cases = [("reference", None)] + [(p.stem, json.loads(p.read_text(encoding="utf-8"))) for p in sorted((source / "mutants").glob("*.json"))]
    summary = {"format": judging.FORMAT, "judge": str(pathlib.Path(judge_dir).resolve()),
               "judgeSha256": judging.folder_digest(judge_dir), "task": str(pathlib.Path(task_dir).resolve()), "cases": []}
    for name, mutant in cases:
        (out / name).mkdir()
        commit = materialize(source, mutant, task_dir, out / name / "result.bundle")
        record = judging.run_judge(engine, judge_dir, task_dir, out / name / "result.bundle", commit)
        (out / name / "judgment.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
        wrong = verdict(record, mutant)
        summary["cases"].append({"case": name, "expected": "pass" if mutant is None else mutant.get("expect"), "ok": not wrong, "wrong": wrong})
    summary["passed"] = all(c["ok"] for c in summary["cases"])
    (out / "selftest.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    return summary


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
    e = sub.add_parser("export")
    e.add_argument("--source", required=True)
    e.add_argument("--rev", help="the frozen revision to take every included file from (default: the working tree)")
    e.add_argument("--out", required=True)
    s = sub.add_parser("selftest")
    for name in ("--source", "--judge", "--task", "--out"):
        s.add_argument(name, required=True)
    s.add_argument("--engine", default=os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker")
    s.add_argument("--lock", help="exclusive engine lock file shared with the other runners")
    a = p.parse_args(argv)
    try:
        if a.cmd == "export":
            result = export(a.source, a.out, a.rev)
        else:
            with sandboxing.engine_lock(a.lock, "judge self-test", str(pathlib.Path(a.out).resolve())):
                result = selftest(a.source, a.judge, a.task, a.out, [a.engine])
    except (judging.JudgeError, suite.SuiteError, sandboxing.SandboxError) as e:
        print(f"judges: {e}", file=sys.stderr)
        return 2
    print(json.dumps(result if a.cmd == "export" else {"passed": result["passed"], "cases": result["cases"]}, indent=2))
    return 0 if a.cmd == "export" or result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
