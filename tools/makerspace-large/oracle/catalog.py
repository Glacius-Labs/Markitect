"""The hidden cases of the large Makerspace, and the program's assessment against them.

  python -I -B catalog.py cases [--step STEP]                          how many cases, per class and family; builds every case
  python -I -B catalog.py run [--step STEP] --program PATH --cwd DIR   runs them against a built program, prints the failures
  python -I -B catalog.py changed --step STEP                          the cases a step adds or changes, against the step before

At W0, Makerspace's own 54 cases stay as they are and must pass unchanged. On top: G, generated per new operation; B and X,
written by hand from the requirements document alone, one module each cluster (cases_*.py).

A change step's cases build on the step before's, through its file `step_cN.py`:

  RETIRED      {case id: why}: the cases this step's change makes wrong. Nothing else goes, since behaviour nobody asked to
               change must stay, so a case that fails on the step's reference without a reason here is a bug in one of them.
  transform(c) optional: the case as it must read now, for a change that only moves its expected values, such as one more row
               in a table every chain reads. A new dict, or the same one.
  build_cases() the new cases, under the step's operations (kit.OPS).

G is generated again from each step's facts, so an operation's changed form changes its case, and a removed one's goes.
"""
from __future__ import annotations

import argparse
import concurrent.futures
import copy
import importlib.util
import json
import pathlib
import sys
from typing import Any

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import kit  # noqa: E402,F401  (puts the old oracle on the path)

CLUSTERS = ("people", "money", "floor", "teaching", "trade")
STEPS = ("W0", "C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8")
_built: dict[str, list[dict[str, Any]]] = {}


def hand_cases() -> list[dict[str, Any]]:
    out: list[dict[str, Any]] = []
    for cluster in CLUSTERS:
        path = HERE / f"cases_{cluster}.py"
        if path.is_file():
            spec = importlib.util.spec_from_file_location(f"cases_{cluster}", path)
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            out += module.build_cases()
    return out


def step_module(step: str) -> Any:
    path = HERE / f"step_{step.lower()}.py"
    spec = importlib.util.spec_from_file_location(f"step_{step.lower()}", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def build_cases(step: str = "W0") -> list[dict[str, Any]]:
    """Every case at a step, in a stable order: what the step before had, and what this step adds."""
    if step not in STEPS:
        raise ValueError(f"{step} is not a step")
    if step in _built:
        return _built[step]
    import generated
    kit.use_step(step)
    if step == "W0":
        cases = old_cases() + new_cases()
    else:
        module = step_module(step)
        retired = getattr(module, "RETIRED", {})
        transform = getattr(module, "transform", lambda c: c)
        before = build_cases(STEPS[STEPS.index(step) - 1])
        kit.use_step(step)
        known = {c["id"] for c in before}
        missing = sorted(set(retired) - known)
        if missing:
            raise ValueError(f"{step} retires cases that do not exist: {missing}")
        kept = [transform(copy.deepcopy(c)) for c in before if c["id"] not in retired and not c["id"].startswith("G-")]
        cases = kept + generated.build_cases() + module.build_cases()
    ids = [c["id"] for c in cases]
    twice = sorted({i for i in ids if ids.count(i) > 1})
    if twice:
        raise ValueError(f"{step}: duplicate case ids: {twice}")
    _built[step] = cases
    return cases


def changed_cases(step: str) -> list[str]:
    """The cases this step adds or changes against the step before: on the program of the step before, these show whether
    a case can tell the change at all."""
    before = {c["id"]: c for c in build_cases(STEPS[STEPS.index(step) - 1])}
    return [c["id"] for c in build_cases(step) if before.get(c["id"]) != c]


def old_cases() -> list[dict[str, Any]]:
    """Makerspace's own cases at W0, from its oracle, by file path, since this file has its name."""
    path = kit.OLD / "catalog.py"
    spec = importlib.util.spec_from_file_location("makerspace_catalog", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.build_cases("W0")


def old_checks() -> dict[str, Any]:
    path = kit.OLD / "catalog.py"
    spec = importlib.util.spec_from_file_location("makerspace_catalog", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.CHECKS


def new_cases() -> list[dict[str, Any]]:
    import generated
    return generated.build_cases() + hand_cases()


def assess(command: list[str], cwd: str, only: str | None = None, workers: int = 8, step: str = "W0", ids: list[str] | None = None) -> dict[str, Any]:
    import harness
    limits = harness.Limits(**harness.LIMITS)
    cases = build_cases(step)
    if only:
        cases = [c for c in cases if c["id"].startswith(only)]
    if ids is not None:
        chosen = set(ids)
        cases = [c for c in cases if c["id"] in chosen]
    checks = old_checks()

    def one(case):
        return harness.run_case(case, command, cwd, limits, checks=checks)

    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        results = list(pool.map(one, cases))
    return {"cases": results, "families": harness.family_summary(results),
            "failed_families": sorted(n for n, r in harness.family_summary(results).items() if r["failed"]),
            "planned_cases": len(cases), "passed_cases": sum(r["passed"] for r in results), "cases_sha256": harness.data_hash(cases),
            "passed": bool(cases) and all(r["passed"] for r in results)}


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("cmd", choices=["cases", "run", "changed"])
    p.add_argument("--step", default="W0", choices=STEPS)
    p.add_argument("--changed-only", action="store_true", help="with run: only the cases this step adds or changes")
    p.add_argument("--program")
    p.add_argument("--cwd", default=".")
    p.add_argument("--only", help="only cases whose id starts with this")
    p.add_argument("--workers", type=int, default=8)
    p.add_argument("--errors", type=int, default=3, help="how many errors to print per failed case")
    a = p.parse_args(argv)
    if a.cmd == "changed":
        print("\n".join(changed_cases(a.step)))
        return 0
    if a.cmd == "cases":
        cases = build_cases(a.step)
        old = {c["id"] for c in old_cases()}
        classes: dict[str, int] = {}
        for c in cases:
            key = "old" if c["id"] in old else c["id"].split("-")[0]
            classes[key] = classes.get(key, 0) + 1
        print(json.dumps({"step": a.step, **classes, "total": len(cases), "requests": sum(len(c["steps"]) for c in cases)}))
        return 0
    report = assess([a.program], a.cwd, a.only, a.workers, a.step, changed_cases(a.step) if a.changed_only else None)
    for case in report["cases"]:
        if not case["passed"]:
            print(case["id"], case["families"])
            for error in case["errors"][:a.errors]:
                print("   ", error[:600])
    print(f"passed {report['passed_cases']} of {report['planned_cases']}")
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
