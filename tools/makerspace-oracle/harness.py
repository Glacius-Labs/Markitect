"""Run the hidden Makerspace cases through the Library oracle's transport.

Only `driver.py` of tools/library-oracle is reused, loaded unchanged by file
path (Process, Limits, TransportFailure). Case runner, matcher and report are
here because the Library's are bound to library records.

Inside the assessment container (mounts as tools/comparison/assess_candidate.py
uses them: the Library oracle at /evaluator, this folder at /harness):
  python -I /harness/harness.py --step W0..C9 --source S --commit C --limits JSON --protected JSON
Locally, against an already built program:
  python -B tools/makerspace-oracle/harness.py --step W0 --program PATH --cwd DIR
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import sys
from typing import Any, Callable

HERE = Path(__file__).resolve().parent
if str(HERE) not in sys.path:
    sys.path.insert(0, str(HERE))  # python -I does not add the script directory


def _library_dir() -> Path:
    for candidate in (HERE.parent / "library-oracle", Path("/evaluator")):
        if (candidate / "driver.py").is_file():
            return candidate
    raise FileNotFoundError("tools/library-oracle/driver.py is not available")


def _load_driver():
    spec = importlib.util.spec_from_file_location("library_driver", _library_dir() / "driver.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


driver = _load_driver()
Limits = driver.Limits
TransportFailure = driver.TransportFailure

# Long cumulative cases need more room than the Library's 30 s and 256 KiB.
LIMITS = {"request_seconds": 2, "process_seconds": 120, "request_bytes": 131072, "output_bytes": 8388608}

ANY = {"$any": True}
NONEMPTY = {"$nonempty": True}


# --- Matching -----------------------------------------------------------------------

def _token(value: Any) -> str | None:
    """The expectation token of a dict ({"$extends": key, "tail": [...]} carries one plain member)."""
    if isinstance(value, dict):
        tokens = [k for k in value if isinstance(k, str) and k.startswith("$")]
        if tokens:
            return tokens[0]
    return None


def _same(expected: Any, actual: Any) -> bool:
    """Strict JSON equality: True is not 1, 1 is not 1.0."""
    if type(expected) is not type(actual):
        return False
    if isinstance(expected, dict):
        return expected.keys() == actual.keys() and all(_same(expected[k], actual[k]) for k in expected)
    if isinstance(expected, list):
        return len(expected) == len(actual) and all(_same(a, b) for a, b in zip(expected, actual))
    return expected == actual


def _match(expected: Any, actual: Any, path: str, errors: list[str], memory: dict[str, Any], count: list[int]) -> None:
    count[0] += 1
    token = _token(expected)
    if token == "$any":
        return
    if token == "$nonempty":
        if not isinstance(actual, str) or not any(not ch.isspace() for ch in actual):
            errors.append(f"{path}: expected a nonempty string")
        return
    if token == "$record":
        memory[expected["$record"]] = copy.deepcopy(actual)
        return
    if token == "$same":
        stored = memory.get(expected["$same"], KeyError)
        if stored is KeyError:
            errors.append(f"{path}: nothing recorded as {expected['$same']!r}")
        elif not _same(stored, actual):
            errors.append(f"{path}: changed since {expected['$same']!r}: before {stored!r}, after {actual!r}")
        return
    if token == "$extends":
        stored = memory.get(expected["$extends"], KeyError)
        if stored is KeyError or not isinstance(stored, list):
            errors.append(f"{path}: no list recorded as {expected['$extends']!r}")
            return
        if not isinstance(actual, list) or not _same(stored, actual[:len(stored)]):
            errors.append(f"{path}: does not start with the list recorded as {expected['$extends']!r}")
            return
        tail = copy.deepcopy(expected["tail"])
        for offset, item in enumerate(tail):
            if isinstance(item, dict) and item.get("seq") == {"$next": True}:
                item["seq"] = len(stored) + offset + 1
        _match(tail, actual[len(stored):], f"{path}[{len(stored)}:]", errors, memory, count)
        return
    if token == "$has":
        if not isinstance(actual, dict):
            errors.append(f"{path}: expected object, got {type(actual).__name__}")
            return
        for key, value in expected["$has"].items():
            if key not in actual:
                errors.append(f"{path}.{key}: missing member")
            else:
                _match(value, actual[key], f"{path}.{key}", errors, memory, count)
        return
    if token is not None:
        raise ValueError(f"unknown expectation token {token}")
    if isinstance(expected, dict):
        if not isinstance(actual, dict):
            errors.append(f"{path}: expected object, got {type(actual).__name__}")
            return
        for key in sorted(expected.keys() - actual.keys()):
            errors.append(f"{path}.{key}: missing member")
        for key in sorted(actual.keys() - expected.keys()):
            errors.append(f"{path}.{key}: unexpected member")
        for key in expected.keys() & actual.keys():
            _match(expected[key], actual[key], f"{path}.{key}", errors, memory, count)
        return
    if isinstance(expected, list):
        if not isinstance(actual, list):
            errors.append(f"{path}: expected list, got {type(actual).__name__}")
        elif len(expected) != len(actual):
            errors.append(f"{path}: expected {len(expected)} items, got {len(actual)}: {actual!r}")
        else:
            for i, (left, right) in enumerate(zip(expected, actual)):
                _match(left, right, f"{path}[{i}]", errors, memory, count)
        return
    if type(expected) is not type(actual) or expected != actual:
        errors.append(f"{path}: expected {expected!r}, got {actual!r}")


def evaluate_step(step: dict[str, Any], response: Any, memory: dict[str, Any]) -> dict[str, Any]:
    """Check one parsed response; responses have exactly the stated members (README of the requirements)."""
    errors: list[str] = []
    count = [0]
    if not isinstance(response, dict):
        return {"errors": ["response: expected a JSON object"], "assertions": 1}
    _match(step["expect"], response, "response", errors, memory, count)
    return {"errors": errors, "assertions": count[0]}


# --- Running ------------------------------------------------------------------------

Check = Callable[[dict[str, Any], list[tuple[dict[str, Any], Any]]], list[str]]


def run_case(case: dict[str, Any], command: list[str], cwd: str, limits, keep_going: bool = False,
             checks: dict[str, Check] | None = None) -> dict[str, Any]:
    """One fresh process per case. The report shape is the Library oracle's run_case shape.

    A `checks_only` case (an invariant check over a scenario) records step
    mismatches without failing on them; only its named checks decide.
    """
    result = {"id": case["id"], "families": case["families"], "planned_steps": len(case["steps"]),
              "executed_steps": 0, "assertions": 0, "errors": [], "steps": [], "passed": False}
    checks_only = bool(case.get("checks_only"))
    keep_going = keep_going or checks_only
    if checks_only:
        result["step_mismatches"] = []
    memory: dict[str, Any] = {}
    transcript: list[tuple[dict[str, Any], Any]] = []
    process = None
    try:
        process = driver.Process(command, cwd, limits)
        for index, step in enumerate(case["steps"]):
            response = process.request(step.get("request"), step.get("raw"))
            transcript.append((step, response))
            checked = evaluate_step(step, response, memory)
            result["executed_steps"] += 1
            result["assertions"] += checked["assertions"]
            if checked["errors"]:
                result["steps"].append({"index": index, "errors": checked["errors"]})
                target = result["step_mismatches"] if checks_only else result["errors"]
                target.extend(f"step {index}: {error}" for error in checked["errors"])
                if not keep_going:
                    break
        process.finish()
        result["assertions"] += 1  # Normal EOF, no extra response, exit zero.
        if result["executed_steps"] == result["planned_steps"]:
            for name in case.get("invariants", []):
                try:
                    found = (checks or {})[name](case, transcript)
                except Exception as exc:  # a malformed response is a failed check, not a crashed run
                    found = [f"could not check: {type(exc).__name__}: {exc}"]
                result["assertions"] += 1
                result["errors"].extend(f"invariant {name}: {error}" for error in found)
    except (TransportFailure, OSError, KeyError, ValueError) as exc:
        result["errors"].append(type(exc).__name__ + ": " + str(exc))
    finally:
        if process is not None:
            process.close()
            result["trace"] = process.trace()
    result["passed"] = not result["errors"] and result["executed_steps"] == result["planned_steps"]
    return result


def family_summary(results: list[dict[str, Any]]) -> dict[str, dict[str, int]]:
    """Per hidden case family: cases and failed cases (a failed family is a missed obligation)."""
    summary: dict[str, dict[str, int]] = {}
    for case in results:
        for name in case["families"]:
            row = summary.setdefault(name, {"cases": 0, "failed": 0})
            row["cases"] += 1
            row["failed"] += 0 if case["passed"] else 1
    return dict(sorted(summary.items()))


def assess(step: str, command: list[str], cwd: str, limits) -> dict[str, Any]:
    import catalog
    cases = catalog.build_cases(step)
    report: dict[str, Any] = {"schema": "markitect.makerspace-assessment/v1", "step": step,
                              "cases_sha256": data_hash(cases), "cases": []}
    for case in cases:
        report["cases"].append(run_case(case, command, cwd, limits, checks=catalog.CHECKS))
    report["families"] = family_summary(report["cases"])
    report["failed_families"] = sorted(name for name, row in report["families"].items() if row["failed"])
    report["planned_cases"] = len(cases)
    report["passed_cases"] = sum(case["passed"] for case in report["cases"])
    report["passed"] = bool(cases) and report["passed_cases"] == len(cases)
    return report


def data_hash(data: Any) -> str:
    return hashlib.sha256(json.dumps(data, sort_keys=True, ensure_ascii=False, separators=(",", ":")).encode()).hexdigest()


def protected_paths(library: Path, here: Path) -> dict[str, Path]:
    """The evaluator bytes an assessment depends on: the reused transport and this oracle."""
    paths = {"evaluator/driver.py": library / "driver.py"}
    for path in sorted(here.glob("*.py")):
        if not path.name.startswith("test_"):
            paths["harness/" + path.name] = path
    return paths


def file_hashes(paths: dict[str, Path]) -> dict[str, str]:
    return {name: hashlib.sha256(path.read_bytes()).hexdigest() for name, path in paths.items()}


def main(argv: list[str] | None = None) -> int:
    import facts
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--step", choices=facts.STEPS, required=True)
    p.add_argument("--program", default="/out/makerspace")
    p.add_argument("--cwd", default="/assess/realization")
    p.add_argument("--source", default="")
    p.add_argument("--commit", default="")
    p.add_argument("--limits", default=json.dumps(LIMITS))
    p.add_argument("--protected", help="evaluator hashes computed on the Host; checked before and after")
    p.add_argument("--census-start", help="the start step's program: also run the step's change census")
    a = p.parse_args(argv)
    paths = protected_paths(_library_dir(), HERE)
    report: dict[str, Any] = {"step": a.step, "commit": a.commit, "source": a.source, "errors": []}
    with open(a.program, "rb") as stream:
        report["program_sha256"] = hashlib.sha256(stream.read()).hexdigest()
    expected = json.loads(a.protected) if a.protected else None
    report["protected_sha256"] = expected or file_hashes(paths)
    if expected is not None and file_hashes(paths) != expected:
        report["errors"].append("evaluator or harness bytes differ from the Host-computed hashes")
    else:
        limits = Limits(**json.loads(a.limits))
        report.update(assess(a.step, [a.program], a.cwd, limits))
        if a.census_start:
            import census
            report["census"] = census.run(a.step, [a.census_start], [a.program], a.cwd, limits)
        if expected is not None and file_hashes(paths) != expected:
            report["errors"].append("evaluator or harness bytes changed during the assessment")
    report["passed"] = not report["errors"] and report.get("passed") is True
    json.dump(report, sys.stdout)
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
