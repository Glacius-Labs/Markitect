"""The change series of the portfolio: each step's task.md and model, as chain steps.

  python -I -B series.py steps --series library|makerspace [--steps M1,M2,...] --out NEW_DIR

A step directory holds what `suite.py task` and `suite.py chain --step` read:
`task.md`, the ordinary documentation every variant receives, and `model/`,
the same intent as a concept 2.0 model: the content of a target project's
`.markitect/` folder (manifest, lock, execution configuration and `model/`).

  library      M1 to M5   task.md: tools/comparison/packet/requirements-mN.md
                          model:   tools/evaluation-tasks/library/mN
  makerspace   W0 to C9   task.md: tools/comparison/makerspace-v2/spec/requirements-STEP.md
                          model:   tools/evaluation-tasks/makerspace/STEP

The step names are the steps of the series judges in tools/evaluation-judges
(`suite.py judge --step`, `judge-chain --steps`). Nothing here is hidden
material: every file goes to every variant.
"""
import argparse
import json
import pathlib
import shutil
import sys

REPO = pathlib.Path(__file__).resolve().parents[2]
SERIES = {
    "library": [(f"M{i}", f"tools/comparison/packet/requirements-m{i}.md", f"tools/evaluation-tasks/library/m{i}") for i in range(1, 6)],
    "makerspace": [(s, f"tools/comparison/makerspace-v2/spec/requirements-{s.lower()}.md", f"tools/evaluation-tasks/makerspace/{s.lower()}")
                   for s in ("W0", "C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8", "C9")],
}


class SeriesError(Exception):
    pass


def steps(series):
    if series not in SERIES:
        raise SeriesError(f"no series {series!r}; known: {', '.join(SERIES)}")
    return [name for name, _, _ in SERIES[series]]


def sources(series, step, repo=REPO):
    """(task.md, model folder) of one step."""
    for name, task_md, model in SERIES.get(series, []):
        if name == step:
            return repo / task_md, repo / model
    raise SeriesError(f"{step} is not a step of {series}")


def write_step(series, step, out, repo=REPO):
    """One step directory: task.md and model/."""
    out = pathlib.Path(out)
    if out.exists():
        raise SeriesError(f"{out} exists")
    task_md, model = sources(series, step, repo)
    out.mkdir(parents=True)
    shutil.copyfile(task_md, out / "task.md")
    shutil.copytree(model, out / "model")
    return out


def write_steps(series, out, only=None):
    out = pathlib.Path(out)
    if out.exists():
        raise SeriesError(f"{out} exists")
    chosen = only or steps(series)
    unknown = [s for s in chosen if s not in steps(series)]
    if unknown:
        raise SeriesError(f"not steps of {series}: {', '.join(unknown)}")
    return [str(write_step(series, s, out / s)) for s in chosen]


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
    s = sub.add_parser("steps")
    s.add_argument("--series", required=True)
    s.add_argument("--steps", help="comma-separated steps (default: every step, in order)")
    s.add_argument("--out", required=True)
    a = p.parse_args(argv)
    try:
        written = write_steps(a.series, a.out, [x.strip() for x in a.steps.split(",")] if a.steps else None)
    except SeriesError as e:
        print(f"series: {e}", file=sys.stderr)
        return 2
    print(json.dumps(written, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
