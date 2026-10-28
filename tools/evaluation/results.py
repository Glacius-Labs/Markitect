"""The Result check: in every Result, every checklist item has a verdict with evidence (concept 2.0 §15, "Nothing skipped").

  python -I -B results.py RUN_DIR

A variant that keeps a checklist returns, besides result.bundle, one Result per
Run it ran, as OUT_DIR/results/RUN.json in this small contract. The 2.0 arm's
Result page (concept 2.0 §6.4) fills it; a variant without a checklist, such as
K, returns none, and the check does not apply to it.

  {"format": 1, "run": "ID", "command": "apply" or "verify" or ...,
   "outcome": "landed" or "not-landed",
   "requiredChecks": {"ran": true, "wholeRepository": true},
   "items": [{"manager": "billing", "item": "billing/invoice", "part": "own" or "inherited" or "finding",
              "verdict": "conforms" or "changed" or "model-problem", "evidence": "...",
              "checker": "billing/checker", "approved": true}],
   "notParticipating": [{"package": "...", "reason": "..."}],
   "findings": [{"definition": "...", "observed": "...", "expected": "..."}]}

An item is skipped when it has no verdict, unsupported when its evidence is
empty, and unapproved when no checker approved it. A Result whose required
checks did not run on the whole repository is incomplete. The run is checked
against its own record too: a delivered change needs a landed Result, and a
landed Result needs a delivered change. `findings` carries what a `verify`
found; after a series, an independent `verify --full` with no findings is the
2.0 arm's own no-drift result.
"""
import json
import pathlib
import sys

FORMAT = 1
VERDICTS = ("conforms", "changed", "model-problem")
OUTCOMES = ("landed", "not-landed")


def check(result):
    """One Result's skipped, unsupported and unapproved items, and its format problems."""
    if not isinstance(result, dict) or result.get("format") != FORMAT:
        return {"problems": [f"not a Result of format {FORMAT}"]}
    problems = []
    if result.get("outcome") not in OUTCOMES:
        problems.append("outcome must be landed or not-landed")
    items = result.get("items")
    if not isinstance(items, list) or not all(isinstance(i, dict) for i in items):
        problems.append("items must be a list of objects")
        items = []
    elif not items and result.get("outcome") == "landed":
        problems.append("a landed Result has no checklist items")
    name = lambda i: f"{i.get('manager')}:{i.get('item')}"  # noqa: E731
    checks = result.get("requiredChecks") or {}
    findings = result.get("findings")
    return {"run": result.get("run"), "command": result.get("command"), "outcome": result.get("outcome"), "items": len(items),
            "skipped": [name(i) for i in items if i.get("verdict") not in VERDICTS],
            "unsupported": [name(i) for i in items if i.get("verdict") in VERDICTS and not str(i.get("evidence") or "").strip()],
            "unapproved": [name(i) for i in items if not (i.get("checker") and i.get("approved") is True)],
            "checksIncomplete": not (checks.get("ran") is True and checks.get("wholeRepository") is True),
            "findings": len(findings) if isinstance(findings, list) else 0, "problems": problems}


def read(path):
    try:
        return json.loads(pathlib.Path(path).read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return None


def check_run(run_dir):
    """Every Result a run returned, checked; None when the run returned none."""
    run_dir = pathlib.Path(run_dir)
    folder = run_dir / "variant" / "results"
    files = sorted(p for p in folder.glob("*.json") if p.is_file() and not p.is_symlink()) if folder.is_dir() else []
    if not files:
        return None
    checked = [{"file": p.name, **check(read(p))} for p in files]
    record = read(run_dir / "suite.json") or {}
    landed_change = bool((record.get("result") or {}).get("changedPaths"))
    landed = any(c.get("outcome") == "landed" for c in checked)
    problems = [f"{c['file']}: {p}" for c in checked for p in c["problems"]]
    if landed_change and not landed:
        problems.append("the run delivered a change, but no Result landed")
    if landed and not landed_change:
        problems.append("a Result landed, but the run delivered no change")
    complete = not problems and not any(c["skipped"] or c["unsupported"] or c["unapproved"] or c["checksIncomplete"] for c in checked)
    return {"results": checked, "problems": problems, "complete": complete}


def main(argv=None):
    argv = sys.argv[1:] if argv is None else argv
    if len(argv) != 1:
        print(__doc__.split("\n")[2], file=sys.stderr)
        return 2
    result = check_run(argv[0])
    print(json.dumps(result, indent=2))
    return 0 if result is None or result["complete"] else 1


if __name__ == "__main__":
    sys.exit(main())
