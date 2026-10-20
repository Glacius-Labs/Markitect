"""Summarize trial attempts. Usage: collect.py [PREFIX] [--json | --aggregate | --first]

Reads ~/mk-tr (or TRIALS_WORK); prefers out/assess-final.json unless --first is given.
"""
import json
import os
import sys
from pathlib import Path

ROOT = Path(os.environ.get("TRIALS_WORK", Path.home() / "mk-tr"))


def kv(path):
    values = {}
    if path.is_file():
        for line in path.read_text().split():
            if "=" in line:
                key, value = line.split("=", 1)
                values[key] = value
    return values


def row(attempt):
    final = attempt / "out" / "assess-final.json"
    assess_path = final if final.is_file() and "--first" not in sys.argv else attempt / "out" / "assess.json"
    timing = kv(attempt / "timing.txt")
    binding = kv(attempt / "binding.txt")
    if not assess_path.is_file():
        return {"attempt": attempt.name, "case": binding.get("case", "?"), "error": "no assessment",
                "apply_exit": timing.get("apply_exit"), "wall": timing.get("wall_seconds")}
    try:
        a = json.loads(assess_path.read_text())
    except ValueError:
        return {"attempt": attempt.name, "case": binding.get("case", "?"), "error": "unreadable assessment",
                "apply_exit": timing.get("apply_exit"), "wall": timing.get("wall_seconds")}
    report = a.get("report") or {}
    behavior = a.get("behavior") or {}
    faults = a.get("faults") or []
    findings = report.get("findings") or []
    return {
        "attempt": attempt.name, "case": a["case"], "expected": a["expected"], "lanes": binding.get("lanes"),
        "met": a["outcome"].get("met"), "defectNamed": a["outcome"].get("defectNamed"),
        "run": report.get("runStatus"), "publication": report.get("publication"),
        "rounds": len(report.get("rounds") or []), "green": [r["green"] for r in report.get("rounds") or []],
        "turns": report.get("turns"), "engine": report.get("engineSeconds"), "wall": int(timing.get("wall_seconds", 0) or 0),
        "behavior": "%s/%s" % ((behavior.get("summary") or {}).get("passed"), (behavior.get("summary") or {}).get("total"))
                    if behavior else "-",
        "behaviorFailed": sorted((behavior.get("failed") or {}).keys()) if behavior else [],
        "faults": "%d/%d" % (sum(1 for f in faults if f["repaired"]), len(faults)) if faults else "-",
        "unrepaired": [f["id"] for f in faults if not f["repaired"]],
        "regression": {f["id"]: (f.get("regressionTest") or {}).get("status") for f in faults},
        "static": {k: v for k, v in (a.get("static") or {}).items() if v},
        "tests": (a.get("tests") or {}).get("passed"),
        "changed": len(a["changes"]["all"]), "outside": a["changes"]["outsideFaultsAndAllowed"],
        "protected": a["changes"]["protectedInputsChanged"], "byOwner": {k: len(v) for k, v in a["changes"].get("byOwner", {}).items()},
        "findings": len(findings), "blocking": sum(1 for f in findings if f.get("blocking")),
        "reason": (report.get("reason") or report.get("error") or "")[:400],
    }


def aggregate(rows):
    """Per case: runs, expectation met, full behavior, faults, rounds, time and changes."""
    import statistics
    cases = {}
    for r in rows:
        if "error" in r:
            continue
        cases.setdefault(r["case"], []).append(r)
    print("| case | runs | met | behavior full | faults repaired | rounds | wall s median (min-max) | turns x/v mean | changed files | outside |")
    print("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
    for case, items in sorted(cases.items()):
        walls = [r["wall"] for r in items]
        faults = [r["faults"] for r in items if r["faults"] != "-"]
        repaired = sum(int(f.split("/")[0]) for f in faults)
        total = sum(int(f.split("/")[1]) for f in faults)
        full = sum(1 for r in items if r["behavior"] != "-" and r["behavior"].split("/")[0] == r["behavior"].split("/")[1])
        behaved = sum(1 for r in items if r["behavior"] != "-")
        print("| %s | %d | %d | %s | %s | %s | %d (%d-%d) | %.1f/%.1f | %s | %s |" % (
            case, len(items), sum(1 for r in items if r["met"]), "%d/%d" % (full, behaved) if behaved else "-",
            "%d/%d" % (repaired, total) if total else "-", ",".join(str(r["rounds"]) for r in items),
            statistics.median(walls), min(walls), max(walls),
            statistics.mean((r["turns"] or {}).get("execute", 0) for r in items),
            statistics.mean((r["turns"] or {}).get("verify", 0) for r in items),
            ",".join(str(r["changed"]) for r in items), ",".join(str(len(r["outside"])) for r in items)))


def main(argv):
    prefix = next((arg for arg in argv[1:] if not arg.startswith("--")), "TR-")
    rows = [row(path) for path in sorted(ROOT.glob(prefix + "*")) if path.is_dir()]
    if "--json" in argv:
        print(json.dumps(rows, indent=1))
        return
    if "--aggregate" in argv:
        aggregate(rows)
        return
    print("| attempt | case | met | run/pub | rounds | turns x/v | engine s | wall s | behavior | faults | outside | blocking/findings |")
    print("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
    for r in rows:
        if "error" in r:
            print("| %s | %s | ERROR %s | exit %s | | | | %s | | | | |" % (r["attempt"], r["case"], r["error"], r["apply_exit"], r["wall"]))
            continue
        turns = r["turns"] or {}
        print("| %s | %s | %s | %s/%s | %d %s | %s/%s | %s | %s | %s %s | %s %s | %d | %d/%d |" % (
            r["attempt"], r["case"], r["met"], r["run"], r["publication"], r["rounds"], r["green"],
            turns.get("execute"), turns.get("verify"), r["engine"], r["wall"], r["behavior"],
            ",".join(r["behaviorFailed"]), r["faults"], ",".join(r["unrepaired"]), len(r["outside"]),
            r["blocking"], r["findings"]))


if __name__ == "__main__":
    main(sys.argv)
