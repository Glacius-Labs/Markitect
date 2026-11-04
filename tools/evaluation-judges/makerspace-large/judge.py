"""The large Makerspace judge: the Makerspace judge's areas over the large project's hidden cases.

  EVALUATION_STEP=W0 python3 -I -B judge.py [--contained] TASK_DIR START_BUNDLE RESULT_BUNDLE OUT_DIR

`suite.py judge --step STEP` runs it in the judge container without network, with the task, the start bundle and the
result bundle read-only, and reads OUT_DIR/judge.json. Inside one container it judges:

- Correctness: the hidden cases of tools/makerspace-large/oracle at W0, which hold Makerspace's own 54 cases unchanged,
  one generated case per new operation and the cases written by hand, against the result's own `go build ./cmd/makerspace`;
- Build and tests: the result's `go build ./cmd/makerspace` and `go test ./...` in `realization/`;
- Scope: every changed path against the start. The task asks for `realization/` only; Markitect's registry pages and
  Results are its bookkeeping; any other `.markitect/` file, `task.md` and `TASK.md` are protected inputs, unless a changed
  file is a copy of the task's own model file;
- Drift (judgekit): the retired terms of drift.json, of which there are none yet, and files nothing accounts for.

The result's code runs as the judge's user; with --contained the judge stops every other process in the container before
it writes judge.json, only when the suite's container says so in EVALUATION_JUDGE_CONTAINER.
"""
import argparse
import json
import os
import pathlib
import shutil
import sys
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
# Exported, the judge folder holds the tools it wraps; in the repository they sit at its root.
ROOT = HERE if (HERE / "tools").is_dir() else HERE.parents[2]
sys.path[:0] = [str(ROOT / "tools" / "makerspace-large" / "oracle"), str(HERE if (HERE / "judgekit.py").is_file() else HERE.parent / "common")]
import catalog  # noqa: E402
import judgekit  # noqa: E402

NAME = "makerspace-large"
PROGRAM = "makerspace"
REQUESTED = "realization/"
STEPS = ["W0"]
# A program five times the size may have five times the tests, and they run on this limit.
BUILD_SECONDS, TEST_SECONDS = 300, 900
WORKERS = 4


def retired():
    return json.loads((HERE / "drift.json").read_text(encoding="utf-8"))["retired"]


def oracle_cases(program, cwd):
    report = catalog.assess([str(program)], str(cwd), workers=WORKERS)
    cases = [{"id": c["id"], "passed": c["passed"], "families": c["families"], "errors": c["errors"][:3]} for c in report["cases"]]
    return cases, {"failedFamilies": report["failed_families"], "casesSha256": report["cases_sha256"]}


def build(module, tmp, label, env):
    program = tmp / "bin" / label / (PROGRAM + (".exe" if os.name == "nt" else ""))
    shutil.copytree(module, tmp / ("build-" + label))
    return program, judgekit.command(["go", "build", "-o", str(program), "./cmd/" + PROGRAM], tmp / ("build-" + label), BUILD_SECONDS, env)


def judge(task, start_bundle, result_bundle, tmp):
    step = judgekit.judged_step(STEPS)
    changed = judgekit.frozen_changed(HERE)
    if changed:
        raise RuntimeError("judge files differ from the ones exported: " + ", ".join(changed))
    repo, start, commit = judgekit.fetch(task, start_bundle, result_bundle, tmp)
    record = {"format": judgekit.FORMAT, "judge": NAME, "step": step, "commit": commit, "start": start}
    record["scope"] = judgekit.scope(task, repo, start, commit, REQUESTED)
    candidate = tmp / "candidate"
    judgekit.checkout(repo, commit, candidate)
    module = candidate / REQUESTED.rstrip("/")
    env = judgekit.go_env(tmp)
    planned = [c["id"] for c in catalog.old_cases() + catalog.new_cases()]
    first = judgekit.introduced({"W0": planned}, STEPS)
    details = {}
    if not (module / "go.mod").is_file():
        missing = {"exit": None, "timedOut": False, "passed": False, "log": "no realization/go.mod"}
        build_record, tests, cases = missing, missing, [{"id": c, "passed": False, "errors": ["not built"]} for c in planned]
    else:
        program, build_record = build(module, tmp, "result", env)
        shutil.copytree(module, tmp / "test")
        tests = judgekit.command(["go", "test", "./..."], tmp / "test", TEST_SECONDS, env)
        if build_record["passed"] and program.is_file():
            cwd = tmp / "assess" / "realization"
            shutil.copytree(module, cwd)
            cases, details = oracle_cases(program, cwd)
        else:
            cases = [{"id": c, "passed": False, "errors": ["not built"]} for c in planned]
    record["correctness"] = judgekit.correctness(cases, step, STEPS, first, planned)
    record["buildAndTests"] = {"build": build_record, "tests": tests}
    record["drift"] = judgekit.drift(cases, step, candidate, REQUESTED, retired(), STEPS,
                                     judgekit.go_accounted(module, env) if (module / "go.mod").is_file() else set())
    record["details"] = details
    return record


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--contained", action="store_true", help="inside the judge container: stop all other processes before writing")
    for name in ("task", "start", "result", "out"):
        p.add_argument(name)
    a = p.parse_args(argv)
    with tempfile.TemporaryDirectory() as tmp:
        try:
            record = judge(pathlib.Path(a.task), pathlib.Path(a.start).resolve(), pathlib.Path(a.result).resolve(), pathlib.Path(tmp))
        except (OSError, ValueError, KeyError, RuntimeError) as e:
            record = {"format": judgekit.FORMAT, "judge": NAME, "error": f"{type(e).__name__}: {e}"}
        return judgekit.write(pathlib.Path(a.out), record, a.contained)


if __name__ == "__main__":
    sys.exit(main())
