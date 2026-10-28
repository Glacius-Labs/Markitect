"""The Makerspace judge: the comparison's Phase C assessment (tools/comparison/assess_candidate.py) as one series judge.

  EVALUATION_STEP=W0..C9 python3 -I -B judge.py [--contained] TASK_DIR START_BUNDLE RESULT_BUNDLE OUT_DIR

`suite.py judge --step STEP` runs it in the judge container without network,
with the task, the start bundle and the result bundle read-only, and reads
OUT_DIR/judge.json. Inside one container it does what assess_candidate.py does
for the profiles w0 and c1-c9 in several:

- Correctness: the hidden Makerspace oracle (tools/makerspace-oracle) at the
  step, which holds every earlier step's cases rendered for this step, against
  the result's own `go build ./cmd/makerspace`; at C3, C7 and C9 also the
  oracle's change census against the start's program, one case per census case;
- Build and tests: the result's `go build ./cmd/makerspace` and `go test ./...`
  in `realization/`, with the Phase C own-check limit;
- Scope: every changed path against the start. The task asks for
  `realization/` only; Markitect's registry pages and Results are its
  bookkeeping; any other `.markitect/` file, `task.md` and `TASK.md` are
  protected inputs, unless a changed file is a copy of the task's own model file;
- Drift (judgekit): earlier steps' failing cases, the retired terms of
  drift.json in non-test source, and files nothing accounts for.

The result's code runs as the judge's user; with --contained the judge stops
every other process in the container before it writes judge.json, only when
the suite's container says so in EVALUATION_JUDGE_CONTAINER.
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
sys.path[:0] = [str(ROOT / "tools" / "makerspace-oracle"), str(HERE if (HERE / "judgekit.py").is_file() else HERE.parent / "common")]
import catalog  # noqa: E402
import census as change_census  # noqa: E402
import facts  # noqa: E402
import harness  # noqa: E402
import judgekit  # noqa: E402

NAME = "makerspace"
PROGRAM = "makerspace"
REQUESTED = "realization/"
STEPS = list(facts.STEPS)
# The Phase C own-check limit for both arms (addendum C), for the build and for the tests.
BUILD_SECONDS, TEST_SECONDS = 300, 300


def retired():
    return json.loads((HERE / "drift.json").read_text(encoding="utf-8"))["retired"]


def oracle_cases(step, program, cwd, start_program):
    """The step's cumulative cases, and its change census against the start's program where the step has one."""
    limits = harness.Limits(**harness.LIMITS)
    report = harness.assess(step, [str(program)], str(cwd), limits)
    cases = [{"id": c["id"], "passed": c["passed"], "families": c["families"], "errors": c["errors"][:3]} for c in report["cases"]]
    details = {"failedFamilies": report["failed_families"], "casesSha256": report["cases_sha256"]}
    if step in change_census.CENSUS:
        if start_program is None:
            details["census"] = "skipped: the start has no program that builds"
        else:
            result = change_census.run(step, [str(start_program)], [str(program)], str(cwd), limits)
            changed = {c["id"]: c for c in result["changed_cases"]}
            for case_id in sorted(result["cases"]):
                cases.append({"id": f"CENSUS-{step}-{case_id}", "passed": case_id not in changed, "families": ["CENSUS"],
                              "errors": [json.dumps(d)[:300] for d in changed[case_id]["differences"][:3]] if case_id in changed else []})
            details["census"] = {"startStep": result["start_step"], "changedCases": len(result["changed_cases"])}
    return cases, details


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
    candidate, start_tree = tmp / "candidate", tmp / "start"
    judgekit.checkout(repo, commit, candidate)
    module = candidate / REQUESTED.rstrip("/")
    env = judgekit.go_env(tmp)
    first = judgekit.introduced({s: [c["id"] for c in catalog.build_cases(s)] for s in STEPS}, STEPS)
    planned = [c["id"] for c in catalog.build_cases(step)]
    if step in change_census.CENSUS:
        planned += [f"CENSUS-{step}-{c['id']}" for c in change_census.census_cases(step)]
    details = {}
    if not (module / "go.mod").is_file():
        missing = {"exit": None, "timedOut": False, "passed": False, "log": "no realization/go.mod"}
        build_record, tests, cases = missing, missing, [{"id": c, "passed": False, "errors": ["not built"]} for c in planned]
    else:
        program, build_record = build(module, tmp, "result", env)
        shutil.copytree(module, tmp / "test")
        tests = judgekit.command(["go", "test", "./..."], tmp / "test", TEST_SECONDS, env)
        if build_record["passed"] and program.is_file():
            start_program = None
            if step in change_census.CENSUS:
                judgekit.checkout(repo, start, start_tree)
                if (start_tree / REQUESTED / "go.mod").is_file():
                    built, start_build = build(start_tree / REQUESTED.rstrip("/"), tmp, "start", env)
                    start_program = built if start_build["passed"] and built.is_file() else None
            cwd = tmp / "assess" / "realization"
            shutil.copytree(module, cwd)
            cases, details = oracle_cases(step, program, cwd, start_program)
        else:
            cases = [{"id": c, "passed": False, "errors": ["not built"]} for c in planned]
    record["correctness"] = judgekit.correctness(cases, step, STEPS, first)
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
