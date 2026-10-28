"""The Library judge: the Library change series M1 to M5 as one series judge.

  EVALUATION_STEP=M1..M5 python3 -I -B judge.py [--contained] TASK_DIR START_BUNDLE RESULT_BUNDLE OUT_DIR

`suite.py judge --step STEP` runs it in the judge container without network,
like the Library M1 judge (../library-m1), and judges each step cumulatively,
as Phases A and B of the comparison did:

- Correctness: M1 and M2, the unchanged Library oracle with that profile, its
  opposite-profile witness and the six hidden probes; M3 to M5, every carried
  and new hidden case of tools/comparison/phase_b_cases.py (M2's cases and the
  probes, rendered for the step, plus short loans, renewals and the removal of
  suspension) and its witness; all through the oracle's transport, against the
  result's own `go build ./cmd/library`;
- Build and tests: `go build ./cmd/library` and `go test ./...` in
  `realization/`, with the comparison's primary limits;
- Scope: every changed path against the start (judgekit): `realization/` is
  requested, Markitect's registry pages and Results are bookkeeping, any other
  `.markitect/` file, `task.md` and `TASK.md` are protected inputs;
- Drift (judgekit): earlier steps' failing cases, the retired terms of
  drift.json in non-test source, and files nothing accounts for.

With --contained, and only when the suite's container sets
EVALUATION_JUDGE_CONTAINER, the judge stops every other process before it
writes judge.json.
"""
import argparse
import copy
import hashlib
import json
import os
import pathlib
import shutil
import sys
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
# Exported, the judge folder holds the tools it wraps; in the repository they sit at its root.
ROOT = HERE if (HERE / "tools").is_dir() else HERE.parents[2]
sys.path[:0] = [str(ROOT / "tools" / "comparison"), str(ROOT / "tools" / "library-oracle"),
                str(HERE if (HERE / "judgekit.py").is_file() else HERE.parent / "common")]
import assess as oracle  # noqa: E402  (tools/library-oracle/assess.py)
import judgekit  # noqa: E402
import phase_b_cases  # noqa: E402
import probes  # noqa: E402
from driver import Limits  # noqa: E402

NAME = "library"
PROGRAM = "library"
REQUESTED = "realization/"
STEPS = ["M1", "M2", "M3", "M4", "M5"]
BUILD_SECONDS, TEST_SECONDS = 300, 600
LIMITS = {"request_seconds": 2, "process_seconds": 30, "request_bytes": 16384, "output_bytes": 262144}
WITNESS = "witness-opposite-profile"


def case_ids(step):
    profile = step.lower()
    if profile in ("m1", "m2"):
        ids = [c["id"] for c in oracle.build_cases(profile)] + [c["id"] for c in probes.build_probes()]
    else:
        ids = [c["id"] for c in phase_b_cases.build_cases(profile)]
    return ids + [WITNESS]


def oracle_cases(step, program, module, cwd):
    """The step's cases and its witness against the built program."""
    profile, limits = step.lower(), Limits(**LIMITS)
    cases, errors = [], []
    if profile in ("m1", "m2"):
        manifest = {"experiment_id": "evaluation-suite", "artifact_id": NAME, "model_digest": f"library {profile} (evaluation suite task)",
                    "source_tree": str(module), "profile": profile, "command": [str(program)], "cwd": str(cwd),
                    "program_sha256": {str(program): hashlib.sha256(program.read_bytes()).hexdigest()},
                    "protected_sha256": oracle.protected_hashes(), "limits": LIMITS}
        report = oracle.assess(copy.deepcopy(manifest))
        cases = [{"id": c["id"], "passed": c["passed"], "errors": c["errors"][:3]} for c in report["cases"]]
        witness = report.get("discrimination") or {}
        errors = report.get("errors") or []
        for case in probes.build_probes():
            result = oracle.run_case(case, [str(program)], str(cwd), limits)
            cases.append({"id": result["id"], "passed": result["passed"], "errors": result["errors"][:3]})
    else:
        for case in phase_b_cases.build_cases(profile):
            result = oracle.run_case(case, [str(program)], str(cwd), limits)
            cases.append({"id": result["id"], "passed": result["passed"], "errors": result["errors"][:3]})
        witness = phase_b_cases.discrimination(profile, [str(program)], str(cwd), limits, oracle.run_case)
    cases.append({"id": WITNESS, "passed": witness.get("expected_mismatch_confirmed") is True, "errors": witness.get("errors", [])[:3]})
    return cases, errors


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
    first = judgekit.introduced({s: case_ids(s) for s in STEPS}, STEPS)
    planned, oracle_errors = case_ids(step), []
    if not (module / "go.mod").is_file():
        missing = {"exit": None, "timedOut": False, "passed": False, "log": "no realization/go.mod"}
        build, tests, cases = missing, missing, [{"id": c, "passed": False, "errors": ["not built"]} for c in planned]
    else:
        program = tmp / "bin" / (PROGRAM + (".exe" if os.name == "nt" else ""))
        shutil.copytree(module, tmp / "build")
        build = judgekit.command(["go", "build", "-o", str(program), "./cmd/" + PROGRAM], tmp / "build", BUILD_SECONDS, env)
        shutil.copytree(module, tmp / "test")
        tests = judgekit.command(["go", "test", "./..."], tmp / "test", TEST_SECONDS, env)
        if build["passed"] and program.is_file():
            cwd = tmp / "assess" / "realization"
            shutil.copytree(module, cwd)
            cases, oracle_errors = oracle_cases(step, program, module, cwd)
        else:
            cases = [{"id": c, "passed": False, "errors": ["not built"]} for c in planned]
    record["correctness"] = judgekit.correctness(cases, step, STEPS, first, planned)
    record["buildAndTests"] = {"build": build, "tests": tests}
    record["drift"] = judgekit.drift(cases, step, candidate, REQUESTED, json.loads((HERE / "drift.json").read_text(encoding="utf-8"))["retired"],
                                     STEPS, judgekit.go_accounted(module, env) if (module / "go.mod").is_file() else set())
    record["details"] = {"oracleErrors": oracle_errors}
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
