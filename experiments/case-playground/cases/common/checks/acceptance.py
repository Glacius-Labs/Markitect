"""Published finite CLI contract checks. Run per frozen station; never a complete verdict.

    python checks/acceptance.py --case CASE --station N [--repo DIR]

This file is the shared driver; each case's checks are in `<case>.py`, a module with a
function `checks(ctx)`. In a case repository it sits beside this file (with the case's
STATIONS.json one folder up); in the playground's source tree it is
`../../<case>/checks/<case>.py` (with `../../<case>/STATIONS.json`).
"""
import argparse
import csv
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import types

HERE = Path(__file__).resolve().parent
CASE_NAME = re.compile(r"[a-z0-9][a-z0-9-]{0,62}")


class CandidateFailure(RuntimeError):
    """The application failed the public process/output contract."""


class UnknownCase(ValueError):
    """No checks or no valid station plan for the requested case."""


def invoke(repo, db, args, error=False):
    before = db.read_bytes() if db.exists() else None
    try:
        proc = subprocess.run([sys.executable, "-B", str(repo / "app.py"), "--db", str(db), *args],
                              capture_output=True, timeout=15, cwd=repo)
    except subprocess.TimeoutExpired as exc:
        raise CandidateFailure(f"application timed out: {args}") from exc
    expected = 2 if error else 0
    stdout = proc.stdout.decode("utf-8", errors="replace")
    stderr = proc.stderr.decode("utf-8", errors="replace")
    if proc.returncode != expected:
        raise CandidateFailure(f"exit {proc.returncode}, expected {expected}; args={args}; "
                               f"stderr={stderr!r}; stdout={stdout!r}")
    try:
        value = json.loads(proc.stdout.decode("utf-8"))
    except (ValueError, UnicodeError) as exc:
        raise CandidateFailure(f"invalid UTF-8 JSON response; args={args}; "
                               f"exit={proc.returncode}; stderr={stderr!r}; stdout={stdout!r}") from exc
    if not isinstance(value, dict):
        raise CandidateFailure(f"response must be a JSON object: {args}")
    if error:
        if "error" not in value:
            raise CandidateFailure(f"expected error object: {args}")
        if (db.read_bytes() if db.exists() else None) != before:
            raise CandidateFailure(f"failed command mutated stored state: {args}")
    return value


def read_output(path, kind):
    """Parse a file the application wrote: "json", "jsonl" or "csv"."""
    if not path.is_file():
        raise CandidateFailure(f"{path.name} was not written")
    try:
        text = path.read_text(encoding="utf-8")
        if kind == "csv":
            return list(csv.reader(text.splitlines(keepends=True)))
        if kind == "jsonl":
            return [json.loads(line) for line in text.splitlines() if line.strip()]
        return json.loads(text)
    except (ValueError, csv.Error) as exc:
        raise CandidateFailure(f"{path.name} is not valid UTF-8 {kind}: {exc}") from exc


def load_case(case):
    """The case's check module and its number of stations; UnknownCase otherwise."""
    if not CASE_NAME.fullmatch(case) or case in ("common", "acceptance"):
        raise UnknownCase(f"unknown case {case!r}")
    if (HERE / f"{case}.py").is_file():  # a case repository
        module_path, plan_path = HERE / f"{case}.py", HERE.parent / "STATIONS.json"
    else:  # the source tree or /in
        folder = HERE.parent.parent / case
        module_path, plan_path = folder / "checks" / f"{case}.py", folder / "STATIONS.json"
    if not module_path.is_file():
        raise UnknownCase(f"unknown case {case!r}: no {case}.py beside {HERE} and no {module_path}")
    try:
        plan = json.loads(plan_path.read_text(encoding="utf-8"))
        stations = len(plan["stations"]) if plan.get("case") == case and isinstance(plan["stations"], list) else 0
    except (OSError, ValueError, AttributeError, KeyError, TypeError):
        stations = 0
    if not stations:
        raise UnknownCase(f"case {case!r}: {plan_path} does not list the stations of case {case}")
    module = types.ModuleType(f"checks_{case.replace('-', '_')}")
    module.__file__ = str(module_path)
    # exec instead of import: no bytecode cache is written into the case repository.
    exec(compile(module_path.read_text(encoding="utf-8"), str(module_path), "exec"), module.__dict__)
    if not callable(getattr(module, "checks", None)):
        raise UnknownCase(f"case {case!r}: {module_path} defines no checks(ctx)")
    return module, stations


def assess(repo, case, station, module=None):
    module = module or load_case(case)[0]
    findings = []
    if not (repo / "app.py").is_file():
        return {"case": case, "station": station, "findings": [
            {"check": "application-entrypoint", "status": "FAIL", "detail": "required app.py is missing"},
            {"check": "released-functional-checks", "status": "NOT RUN", "detail": "application cannot launch"}],
            "limitation": "Missing entrypoint; individual functional checks did not execute."}
    with tempfile.TemporaryDirectory(prefix="playground-public-check-") as folder:
        db = Path(folder) / "state.json"
        def call(*args, error=False):
            return invoke(repo, db, args, error=error)
        def check(name, body):
            try:
                body()
                findings.append({"check": name, "status": "PASS"})
            except (AssertionError, CandidateFailure, KeyError, TypeError) as error:
                findings.append({"check": name, "status": "FAIL", "detail": str(error)})
            except Exception as error:
                findings.append({"check": name, "status": "EVALUATION_ERROR", "detail": str(error)})
        def expect(actual, wanted):
            assert actual == wanted, (actual, wanted)
        def malformed():
            saved = db.read_bytes() if db.exists() else None
            try:
                db.write_text("{broken", encoding="utf-8")
                call("list", error=True)
            finally:
                if saved is None:
                    db.unlink(missing_ok=True)
                else:
                    db.write_bytes(saved)
        module.checks(types.SimpleNamespace(repo=repo, folder=Path(folder), db=db, station=station, call=call,
                                            check=check, expect=expect, invoke=invoke, read_output=read_output))
        check("malformed-db-preserved", malformed)
    return {"case": case, "station": station, "findings": findings,
            "limitation": "Finite public checks; assessor must add meaningful cases and semantic review. Cascading failures retain individual evidence."}


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument("--case", required=True)
    parser.add_argument("--station", type=int, required=True)
    args = parser.parse_args()
    try:
        case_module, count = load_case(args.case)
    except UnknownCase as error:
        parser.error(str(error))
    if not 1 <= args.station <= count:
        parser.error(f"case {args.case} has stations 1 to {count}")
    result = assess(args.repo.resolve(), args.case, args.station, case_module)
    print(json.dumps(result, indent=2, ensure_ascii=False))
    sys.exit(1 if any(f["status"] != "PASS" for f in result["findings"]) else 0)
