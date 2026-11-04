"""Where a step's reference and its hidden cases meet, after being written independently.

  python -I -B meet.py --step STEP

Builds the step's reference and the step before's, and answers two questions. Does the step's reference pass every case of
the step? A failure is a mistake in the reference or in the case, to be settled from the rendered requirements. And can the
step's new and changed cases tell the change at all? A case that also passes on the step before's program is listed: it is
right only if it holds something that must stay. It starts no container and no agent.
"""
import argparse
import os
import pathlib
import subprocess
import sys
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import catalog  # noqa: E402


def build(step, out):
    program = pathlib.Path(out) / (f"makerspace-{step}" + (".exe" if os.name == "nt" else ""))
    done = subprocess.run(["go", "build", "-o", str(program), "./cmd/makerspace"], cwd=HERE.parent / "reference" / step.lower(),
                          capture_output=True, text=True, errors="replace")
    if done.returncode != 0:
        raise SystemExit(f"the reference of {step} does not build:\n{done.stderr[-2000:]}")
    return program


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--step", required=True, choices=catalog.STEPS[1:])
    p.add_argument("--errors", type=int, default=2)
    a = p.parse_args(argv)
    previous = catalog.STEPS[catalog.STEPS.index(a.step) - 1]
    with tempfile.TemporaryDirectory(prefix="ms-meet-", ignore_cleanup_errors=True) as tmp:
        now, before = build(a.step, tmp), build(previous, tmp)
        report = catalog.assess([str(now)], tmp, step=a.step)
        failed = [c for c in report["cases"] if not c["passed"]]
        print(f"{a.step} on its reference: passed {report['passed_cases']} of {report['planned_cases']}")
        for case in failed:
            print(" FAIL", case["id"])
            for error in case["errors"][:a.errors]:
                print("     ", error[:500])
        changed = catalog.changed_cases(a.step)
        old = catalog.assess([str(before)], tmp, step=a.step, ids=changed)
        same = [c["id"] for c in old["cases"] if c["passed"]]
        print(f"{a.step}'s new or changed cases on the program of {previous}: {len(changed) - len(same)} of {len(changed)} fail, as they should")
        for name in same:
            print(" also passes before the change:", name)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
