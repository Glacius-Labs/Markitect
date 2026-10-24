"""Mechanical checks of one rubric answer before any critical finding counts.

A critical finding counts only when this check finds it valid and a second,
independent reviewer confirms it. Valid means: the cited file exists in the
candidate and the quote occurs within three lines of the cited line; for R2
and R3a the quoted requirement occurs verbatim in the requirements text; for
R1a and R1b the claimed absence is true on disk.

  python -I -B rubric_check.py --token-dir DIR --requirements FILE --review REVIEW.json
"""
import argparse
import json
import pathlib
import re
import sys


def normalize(text):
    return " ".join(str(text).split())


def cited(candidate, cite, quote):
    match = re.fullmatch(r"(.+):(\d+)", str(cite or ""))
    if not match:
        return "the citation is not path:line"
    path = (candidate / match.group(1)).resolve()
    if candidate.resolve() not in path.parents or not path.is_file():
        return "the cited file is not in the candidate"
    lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
    line = int(match.group(2))
    window = " ".join(lines[max(0, line - 4):line + 3])
    return None if normalize(quote) and normalize(quote) in normalize(window) else "the quote is not at the cited line"


def candidate_criticals(review):
    """Every finding the rubric weighs as critical, with its item."""
    out = []
    for key in ("R1a", "R1b"):
        if review.get(key) is False:
            out.append((key, {"claim": "missing"}))
    for part, present in (review.get("R1c") or {}).items():
        if present is False:
            out.append(("R1c", {"claim": "documentation lacks " + part}))
    for key in ("R2", "R3a", "R4"):
        out += [(key, finding) for finding in review.get(key) or []]
    out += [("R5", finding) for finding in review.get("R5") or [] if finding.get("class") == "unjustified"]
    return out


def check(token_dir, requirements, review):
    candidate = pathlib.Path(token_dir) / "candidate"
    text = normalize(pathlib.Path(requirements).read_text(encoding="utf-8"))
    results = []
    for item, finding in candidate_criticals(review):
        reason = None
        if item == "R1a":
            present = (candidate / "realization/go.mod").is_file() and (candidate / "realization/cmd/library").is_dir()
            reason = "the module and entry point exist" if present else None
        elif item == "R1b":
            tests = [p for p in candidate.rglob("*_test.go") if re.search(r"(?m)^func Test", p.read_text(encoding="utf-8", errors="replace"))]
            reason = "tests exist" if tests else None
        elif item in ("R2", "R3a", "R4") or finding.get("cite"):
            reason = cited(candidate, finding.get("cite"), finding.get("quote", ""))
            if reason is None and item in ("R2", "R3a") and normalize(finding.get("requirement", "")) not in text:
                reason = "the quoted requirement is not verbatim in the requirements"
            if reason is None and item in ("R2", "R3a") and not normalize(finding.get("requirement", "")):
                reason = "no requirement is quoted"
        elif item == "R5":
            reason = "an unjustified change needs a citation"
        results.append({"item": item, "finding": finding, "valid": reason is None, "reason": reason})
    return {"token": review.get("token"), "criticals": results,
            "valid_criticals": sum(r["valid"] for r in results)}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--token-dir", required=True)
    p.add_argument("--requirements", required=True)
    p.add_argument("--review", required=True)
    a = p.parse_args(argv)
    result = check(a.token_dir, a.requirements, json.loads(pathlib.Path(a.review).read_text(encoding="utf-8")))
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
