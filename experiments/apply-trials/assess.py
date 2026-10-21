"""Assess one Apply trial result against its case's answer key; see README.md.

    python -B assess.py CASE --repo REPO [--report APPLY_JSON] [--python CMD] [--output FILE]

REPO is the target repository after Apply: HEAD is the committed start and
the working tree holds whatever Apply published. The assessment compares the
outcome with the expectation, runs the case's oracle profile on the result,
decides per fault whether it was repaired and whether a test now guards it,
runs the case's artifact checks, lists every change outside the faults and
the case's allowed paths, and summarizes passes (rounds before DEC-064), findings, turns and time
from the Apply report. False findings, the content of a model-defect report
and human interventions stay the evaluator's judgment; the report lists the
material for it. Operator material: never part of an Agent's repository.
"""
import argparse
import fnmatch
import io
import json
import re
import shutil
import subprocess
import sys
import tempfile
import zipfile
from datetime import datetime
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

import oracle  # noqa: E402
from cases import APPLIED, CASES, DEFECT, GAP, PROJECTS, REFUSED, UNCHANGED  # noqa: E402
from materialize import GIT, model_case  # noqa: E402
from trial_kit import python_command  # noqa: E402

TEST_COMMANDS = {
    "stockroom": ["-B", "-m", "unittest", "discover", "-s", "tests"],
    "depot": ["-B", "-m", "unittest", "discover", "-s", ".", "-p", "test_*.py"],
    "hall": ["-B", "-m", "unittest", "discover", "-s", ".", "-p", "test_*.py"],
    "fleet": ["-B", "-m", "unittest", "discover", "-s", ".", "-p", "test_*.py"],
}


def test_command(project):
    return TEST_COMMANDS["fleet" if project.startswith("fleet-") else project]
TEST_TIMEOUT = 300


# -- trees ---------------------------------------------------------------------------

def git_bytes(repo, *args):
    return subprocess.run([*GIT, "-C", str(repo), *args], capture_output=True, check=True).stdout


def export_head(repo, folder):
    """Write the committed start (HEAD) of REPO into FOLDER."""
    archive = git_bytes(repo, "archive", "--format=zip", "HEAD")
    with zipfile.ZipFile(io.BytesIO(archive)) as packed:
        packed.extractall(folder)
    return Path(folder)


def changed_paths(repo):
    """Paths whose working-tree state differs from HEAD, including untracked and deleted files."""
    entries = git_bytes(repo, "status", "--porcelain=v1", "-z", "-uall").decode("utf-8").split("\0")
    changed, skip = [], False
    for entry in entries:
        if skip:
            skip = False
            continue
        if not entry:
            continue
        changed.append(entry[3:])
        skip = entry[0] in "RC"  # a rename carries its source as the next entry
    return sorted(changed)


def matches(path, patterns):
    return any(fnmatch.fnmatchcase(path, pattern) for pattern in patterns)


def owner_of(project, path):
    """The Manager whose area holds PATH under the project's layout (the deepest Package; Root outside it)."""
    layout = PROJECTS[project]
    for base, fallback in (("realization/", layout["exception"]), (".markitect/registry/", ".")):
        if path.startswith(base):
            rest = path[len(base):]
            owners = [package for package in layout["packages"] if rest.startswith(package + "/")]
            return max(owners, key=len) if owners else fallback
    return "."


def by_owner(project, paths):
    grouped = {}
    for path in paths:
        grouped.setdefault(owner_of(project, path), []).append(path)
    return grouped


# -- static artifact checks ---------------------------------------------------------------

def read(realization, name):
    path = Path(realization) / name
    return path.read_text(encoding="utf-8", errors="replace") if path.is_file() else None


def static_readme_commands(realization, case):
    text = read(realization, "README.md")
    if text is None:
        return "realization/README.md does not exist"
    if case.project.startswith("fleet-"):
        import fleet
        words = ("credit", "debit", "balances", "totals", "--account", "--amount") + tuple(
            r for r, *_ in fleet.ledgers(case.project))
    else:
        words = {"stockroom": ("add-item", "receive", "issue", "list", "--sku", "--name", "--qty"),
                 "depot": ("add-item", "receive", "place-order", "fulfil-order", "cancel-order", "orders", "items",
                           "reorder", "--line", "--status"),
                 "hall": ("add-room", "rooms", "book", "bookings", "--code", "--seats")}[case.project]
    missing = [word for word in words if word not in text]
    return "the README does not mention %s" % missing if missing else ""


def static_readme_facts(realization, case):
    text = read(realization, "README.md") or ""
    present = {"stockroom": ("--qty", "onHand"), "depot": ("fulfil-order",)}.get(case.project, ())
    absent = {"stockroom": ("--quantity", "on_hand"), "depot": ("fulfill-order",)}.get(case.project, ())
    problems = ["missing %r" % word for word in present if word not in text]
    problems += ["still documents %r" % word for word in absent if word in text]
    return "; ".join(problems)


def static_readme_s2(realization, case):
    text = read(realization, "README.md") or ""
    missing = [word for word in ("--reorder", "--low", "reorderPoint") if word not in text]
    return "the README does not mention %s" % missing if missing else ""


def static_tests_present(realization, case):
    found = [path for path in Path(realization).rglob("test_*.py") if "__pycache__" not in path.parts]
    return "" if found else "no test_*.py module in the realization"


def static_config_consistent(realization, case):
    text = read(realization, "limits.json")
    if text is None:
        return ""  # the values may live elsewhere; the oracle judges the behavior
    try:
        values = json.loads(text)
    except ValueError as error:
        return "limits.json is not JSON: %s" % error
    expected = {"maxQuantity": 1000, "maxOnHand": 9999, "maxNameLength": 60}
    wrong = {key: values.get(key) for key in expected if key in values and values[key] != expected[key]}
    return "limits.json disagrees with the model: %s" % wrong if wrong else ""


def static_readme_name_40(realization, case):
    """The README states the owner's name maximum of 40 and no longer the earlier 60 (S-N40)."""
    lines = [line for line in (read(realization, "README.md") or "").splitlines() if "name" in line.lower()]
    stated = [line for line in lines if re.search(r"(?<![0-9])40(?![0-9])", line)]
    stale = [line for line in lines if re.search(r"(?<![0-9])60(?![0-9])", line)]
    if not stated or stale:
        return "the README's name lines do not state 40 or still state 60: %s" % lines[:5]
    return ""


def static_readme_limits(realization, case):
    """The README does not point at a limits file the realization no longer has (S-O moves the limits)."""
    text = read(realization, "README.md") or ""
    if "limits.json" in text and not (Path(realization) / "limits.json").is_file():
        return "the README still refers to limits.json, which the realization no longer has"
    return ""


def static_registry_owner(realization, case):
    """The reports module is registered by the reports shard and by no other shard (depot fault m1)."""
    shards = Path(realization).parent / ".markitect" / "registry"
    target = "realization/reports/views.py"
    holders = sorted(path.relative_to(shards).as_posix() for path in shards.rglob("*.yaml")
                     if target in path.read_text(encoding="utf-8", errors="replace"))
    return "" if holders == ["reports/registry.yaml"] else "%s is registered in %s" % (target, holders or "no shard")


STATIC = {
    "readme-commands": static_readme_commands,
    "readme-facts": static_readme_facts,
    "readme-s2": static_readme_s2,
    "tests-present": static_tests_present,
    "config-consistent": static_config_consistent,
    "readme-limits": static_readme_limits,
    "readme-name-40": static_readme_name_40,
    "registry-owner": static_registry_owner,
}


def run_static(names, realization, case):
    return {name: STATIC[name](realization, case) for name in names}


# -- tests ---------------------------------------------------------------------------

def run_tests(realization, project, python):
    done = subprocess.run([*python, *test_command(project)], cwd=realization, capture_output=True, text=True,
                          encoding="utf-8", errors="replace", timeout=TEST_TIMEOUT)
    return done.returncode == 0, (done.stdout + done.stderr)[-1500:]


def regression_test(result_realization, fault, project, python):
    """Put the fault's code edits back into a copy of the result and see whether its tests notice."""
    code = [edit for edit in fault.edits if "/tests/" not in "/" + edit.path and "test_" not in Path(edit.path).name
            and not edit.path.endswith(".md") and edit.path.startswith("realization/")]
    if not code:
        return {"status": "not-applicable", "detail": "the fault has no code edit"}
    with tempfile.TemporaryDirectory(prefix="trial-regression-") as root:
        copy = Path(root) / "realization"
        shutil.copytree(result_realization, copy, ignore=shutil.ignore_patterns("__pycache__"))
        for edit in code:
            path = copy / Path(edit.path).relative_to("realization")
            text = path.read_text(encoding="utf-8") if path.is_file() else ""
            if text.count(edit.old) != edit.count:
                return {"status": "not-applicable",
                        "detail": "the repaired text of %s is not found as the fault replaced it" % edit.path}
            path.write_text(text.replace(edit.old, edit.new), encoding="utf-8", newline="\n")
        passed, output = run_tests(copy, project, python)
    return {"status": "absent" if passed else "present",
            "detail": "the tests still pass with the fault put back" if passed else "the tests fail with the fault put back"}


# -- Apply report ----------------------------------------------------------------------

def parse_time(text):
    return datetime.fromisoformat(text.replace("Z", "+00:00")) if text and not text.startswith("0001") else None


def summarize_report(report):
    run = report.get("run") or {}
    turns = run.get("turns") or []
    starts = [parse_time(turn.get("started")) for turn in turns]
    ends = [parse_time(turn.get("ended")) for turn in turns]
    starts, ends = [t for t in starts if t], [t for t in ends if t]
    findings = run.get("findings") or []
    preparation = report.get("preparation") or {}
    diagnostics = [item for attempt in preparation.get("attempts") or [] for key, items in attempt.items()
                   if key.endswith("Diagnostics") for item in items or []]
    return {
        "status": report.get("status"), "runStatus": run.get("status"), "reason": run.get("reason", ""),
        "error": report.get("error", ""), "preparationStatus": preparation.get("status"),
        "preparation": " ".join([attempt.get("error", "") for attempt in preparation.get("attempts") or []]
                                + ["%s %s %s" % (item.get("Code", ""), (item.get("Source") or {}).get("File", ""),
                                                 item.get("Message", "")) for item in diagnostics])[:2000],
        "publication": (report.get("publication") or {}).get("Outcome"), "retainedWork": report.get("retainedWork", ""),
        "rounds": [{"number": item.get("number"), "green": item.get("green"),
                    "checks": {check.get("name"): check.get("passed") for check in item.get("checks") or []}}
                   for item in run.get("rounds") or []],
        "passes": [{"id": item.get("id"), "purpose": item.get("purpose"), "review": item.get("review"),
                    "verdict": item.get("verdict"), "checks": {check.get("name"): check.get("passed")
                                                              for check in item.get("checks") or []}}
                   for item in run.get("passes") or []],
        "questions": [{"from": item.get("from"), "to": item.get("to"), "answered": item.get("answered"),
                       "text": (item.get("text") or "")[:400]} for item in run.get("questions") or []],
        "problems": [{"manager": item.get("manager"), "category": item.get("category"), "abort": item.get("abort"),
                      "text": (item.get("text") or "")[:400]} for item in run.get("problems") or []],
        "turns": {kind: sum(1 for turn in turns if turn.get("kind") == kind)
                  for kind in sorted({turn.get("kind") for turn in turns if turn.get("kind")})},
        "engineSeconds": round((max(ends) - min(starts)).total_seconds()) if starts and ends else None,
        "costUSD": round((run.get("usage") or {}).get("costUSD", 0.0), 2),
        "findings": [{"id": item.get("id"), "owner": item.get("owner"), "raisedBy": item.get("raisedBy"),
                      "blocking": item.get("blocking"), "closed": item.get("closed"),
                      "pass": item.get("pass", item.get("round")),
                      "obligation": (item.get("obligation") or "")[:300], "observed": (item.get("observed") or "")[:300]}
                     for item in findings],
    }


# How Root ends a run over a model defect: the model_defect tool (DEC-063, Incomplete) or the
# Root abort of the pass process (DEC-064, Failed).
DEFECT_REASONS = ("model defect reported by Root", "Root judged the model erroneous or contradictory")


def judge_outcome(case, summary, changed):
    if summary is None:
        return {"met": None, "detail": "no Apply report supplied"}
    expected = case.expected
    if expected == APPLIED:
        met = summary["runStatus"] == "verified" and summary["publication"] == "applied" and bool(changed)
    elif expected == UNCHANGED:
        met = summary["runStatus"] == "verified" and summary["publication"] == "no_change" and not changed
    else:
        quiet = summary["publication"] in (None, "") and not changed
        stopped = summary["runStatus"] in ("incomplete", "failed") and summary["reason"].startswith(DEFECT_REASONS)
        asked = any(question["to"] == "user" for question in summary.get("questions") or [])
        # A refused start fails its preparation and creates no run; a missing run alone can also be a
        # runtime, lock or run-config failure.
        refused = summary["status"] == "failed" and summary["preparationStatus"] == "failed" and not summary["runStatus"]
        met = quiet and {DEFECT: stopped, GAP: stopped or asked, REFUSED: refused}[expected]
    detail = "run %s, publication %s, %d changed paths" % (summary["runStatus"], summary["publication"], len(changed))
    result = {"met": met, "detail": detail}
    if expected in (DEFECT, GAP, REFUSED):
        text = " ".join([summary["reason"], summary.get("error") or "", summary["preparation"]]
                        + [question["text"] for question in summary.get("questions") or [] if question["to"] == "user"]).lower()
        named = [[term for term in group if term.lower() in text] for group in case.defect_terms]
        result["defectNamed"] = all(named) if case.defect_terms else None
        result["defectTerms"] = named
    return result


# -- assessment ------------------------------------------------------------------------

def assess(case_id, repo, report_path=None, python=None, previous=None):
    """PREVIOUS is the realization of the previous model version for upgrade checks; a model change on
    an earlier result (start `previous` with a snapshot) uses the committed start by default."""
    case = CASES[case_id]
    python = python or python_command("")
    repo = Path(repo).resolve()
    profile = case.profile
    if not case.snapshot:
        model = model_case(repo)
        profile = model.profile if model else ""
    changed = changed_paths(repo)
    fault_paths = sorted({edit.path for fault in case.faults for edit in fault.edits})
    protected = [path for path in changed if path.startswith(".markitect/") and not path.startswith(".markitect/registry/")]
    unexpected = [path for path in changed if path not in fault_paths and not matches(path, case.allowed)]
    summary = summarize_report(json.loads(Path(report_path).read_text(encoding="utf-8"))) if report_path else None
    result = {"case": case.id, "title": case.title, "question": case.question, "expected": case.expected,
              "profile": profile, "repo": str(repo), "outcome": judge_outcome(case, summary, changed),
              "changes": {"all": changed, "byOwner": by_owner(case.project, changed),
                          "protectedInputsChanged": protected, "faultFiles": fault_paths,
                          "outsideFaultsAndAllowed": unexpected},
              "report": summary, "measure": list(case.measure)}
    realization = repo / "realization"
    if case.expected == DEFECT or not profile:
        result["behavior"] = None
        return result
    with tempfile.TemporaryDirectory(prefix="trial-start-") as folder:
        start = export_head(repo, folder)
        if previous is None and case.start == "previous" and case.snapshot and (start / "realization" / "app.py").is_file():
            previous = start / "realization"
        behavior = oracle.assess(case.project, profile, realization, python, previous)
    summary_counts = behavior["summary"]
    result["behavior"] = {"passed": summary_counts["failed"] == 0 and summary_counts["skipped"] == 0,
                          "summary": summary_counts,
                          "failed": {item["id"]: item["detail"] for item in behavior["checks"] if item["status"] == "fail"},
                          "skipped": {item["id"]: item["detail"] for item in behavior["checks"] if item["status"] == "skip"}}
    status = {item["id"]: item["status"] for item in behavior["checks"]}
    static = run_static(sorted(set(case.static) | {name for fault in case.faults for name in fault.static}),
                        realization, case)
    result["static"] = static
    tests_pass, tests_output = run_tests(realization, case.project, python) if (realization / "app.py").is_file() else (False, "")
    result["tests"] = {"passed": tests_pass, "tail": "" if tests_pass else tests_output}
    faults = []
    for fault in case.faults:
        open_checks = [name for name in fault.detected_by if status.get(name) != "pass"]
        open_static = [name for name in fault.static if static.get(name)]
        entry = {"id": fault.id, "owner": fault.owner, "kind": fault.kind, "summary": fault.summary,
                 "repaired": not open_checks and not open_static, "openChecks": open_checks + open_static}
        if tests_pass:
            entry["regressionTest"] = regression_test(realization, fault, case.project, python)
            entry["regressionTestRequired"] = fault.regression_test == "required"
        faults.append(entry)
    result["faults"] = faults
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("case", choices=sorted(CASES))
    parser.add_argument("--repo", required=True)
    parser.add_argument("--report", help="the JSON report that markitect apply printed")
    parser.add_argument("--previous", help="realization of the previous model version, for upgrade checks")
    parser.add_argument("--python", default="", help="interpreter command (default: this interpreter)")
    parser.add_argument("--output", help="also write the assessment to this file")
    args = parser.parse_args(argv)
    result = assess(args.case, args.repo, args.report, python_command(args.python),
                    Path(args.previous).resolve() if args.previous else None)
    text = json.dumps(result, indent=2)  # ASCII escapes: details can quote undecodable test input
    print(text)
    if args.output:
        Path(args.output).write_text(text + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    sys.exit(main())
