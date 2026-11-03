"""The hidden cases of the large Makerspace, and the program's assessment against them.

  python -I -B catalog.py cases                          how many cases, per class and family; builds every case
  python -I -B catalog.py run --program PATH --cwd DIR   runs them against a built program, prints the failures

Makerspace's own 54 cases at W0 stay as they are and must pass unchanged. On top: G, generated per new operation; B and X,
written by hand from the requirements document alone, one module each cluster (cases_*.py).
"""
from __future__ import annotations

import argparse
import concurrent.futures
import importlib.util
import json
import pathlib
import sys
from typing import Any

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import kit  # noqa: E402,F401  (puts the old oracle on the path)

CLUSTERS = ("people", "money", "floor", "teaching", "trade")


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


def build_cases() -> list[dict[str, Any]]:
    return old_cases() + new_cases()


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


def assess(command: list[str], cwd: str, only: str | None = None, workers: int = 8) -> dict[str, Any]:
    import harness
    limits = harness.Limits(**harness.LIMITS)
    cases = old_cases() + new_cases()
    if only:
        cases = [c for c in cases if c["id"].startswith(only)]
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
    p.add_argument("cmd", choices=["cases", "run"])
    p.add_argument("--program")
    p.add_argument("--cwd", default=".")
    p.add_argument("--only", help="only cases whose id starts with this")
    p.add_argument("--workers", type=int, default=8)
    p.add_argument("--errors", type=int, default=3, help="how many errors to print per failed case")
    a = p.parse_args(argv)
    if a.cmd == "cases":
        old, new = old_cases(), new_cases()
        classes: dict[str, int] = {}
        for c in new:
            key = c["id"].split("-")[0]
            classes[key] = classes.get(key, 0) + 1
        print(json.dumps({"old": len(old), **classes, "total": len(old) + len(new), "requests": sum(len(c["steps"]) for c in old + new)}))
        ids = [c["id"] for c in old + new]
        dupes = sorted({i for i in ids if ids.count(i) > 1})
        if dupes:
            print("duplicate case ids:", dupes)
            return 1
        return 0
    report = assess([a.program], a.cwd, a.only, a.workers)
    for case in report["cases"]:
        if not case["passed"]:
            print(case["id"], case["families"])
            for error in case["errors"][:a.errors]:
                print("   ", error[:600])
    print(f"passed {report['passed_cases']} of {report['planned_cases']}")
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
