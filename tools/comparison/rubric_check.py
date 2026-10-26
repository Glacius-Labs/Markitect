"""Mechanical checks of one rubric answer before any critical finding counts.

A critical finding counts only when this check finds it valid and a second,
independent reviewer confirms it. Valid means: the cited file exists in the
candidate (or, for a removal, in the start tree) and the quote occurs within
three lines of the cited line, or the quote occurs in diff.patch; for R2 and
R3a the quoted requirement occurs verbatim in the requirements text; for R1a
and R1b the claimed absence is true on disk. With --s3, every R5 file
classified optional or unjustified is a critical finding too, valid when the
file appears in diff.patch.

  python -I -B rubric_check.py --token-dir DIR --requirements FILE --review REVIEW.json [--s3] [--entry PATH]

--entry names the program's main package for R1a; the default is the
Library's, and Phase C passes realization/cmd/makerspace.
"""
import argparse
import json
import pathlib
import re
import sys


def normalize(text):
    return " ".join(str(text).split())


def cited(roots, cite, quote, diff=""):
    """None when the quote is at the cited line of a file in one of the roots, or in the diff."""
    if not normalize(quote):
        return "no quote is given"
    match = re.fullmatch(r"(.+):(\d+)", str(cite or ""))
    if not match:
        return "the citation is not path:line"
    for root in roots:
        path = (root / match.group(1)).resolve()
        if root.resolve() in path.parents and path.is_file():
            lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
            line = int(match.group(2))
            if normalize(quote) in normalize(" ".join(lines[max(0, line - 4):line + 3])):
                return None
    diff_lines = " ".join(line[1:] for line in diff.splitlines() if line[:1] in "+- " and not line.startswith(("+++", "---")))
    return None if normalize(quote) in normalize(diff_lines) else "the quote is not at the cited line"


def in_diff(diff, path):
    return ("+++ b/" + path) in diff or ("--- a/" + path) in diff


def candidate_criticals(review, s3=False):
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
    classes = ("optional", "unjustified") if s3 else ("unjustified",)
    out += [("R5", finding) for finding in review.get("R5") or [] if finding.get("class") in classes]
    return out


LIBRARY_ENTRY = "realization/cmd/library"


def check(token_dir, requirements, review, s3=False, entry=LIBRARY_ENTRY):
    token = pathlib.Path(token_dir)
    candidate = token / "candidate"
    roots = (candidate, token / "start")
    diff_path = token / "diff.patch"
    diff = diff_path.read_text(encoding="utf-8", errors="replace") if diff_path.is_file() else ""
    text = normalize(pathlib.Path(requirements).read_text(encoding="utf-8"))
    results = []
    for item, finding in candidate_criticals(review, s3):
        reason = None
        if item == "R1a":
            present = (candidate / "realization/go.mod").is_file() and (candidate / entry).is_dir()
            reason = "the module and entry point exist" if present else None
        elif item == "R1b":
            tests = [p for p in candidate.rglob("*_test.go") if re.search(r"(?m)^func Test", p.read_text(encoding="utf-8", errors="replace"))]
            reason = "tests exist" if tests else None
        elif item in ("R2", "R3a", "R4"):
            reason = cited((candidate,), finding.get("cite"), finding.get("quote", ""))
            if reason is None and item in ("R2", "R3a") and not normalize(finding.get("requirement", "")):
                reason = "no requirement is quoted"
            elif reason is None and item in ("R2", "R3a") and normalize(finding.get("requirement", "")) not in text:
                reason = "the quoted requirement is not verbatim in the requirements"
        elif item == "R5":
            if not in_diff(diff, str(finding.get("path", ""))):
                reason = "the file is not changed in diff.patch"
            elif finding.get("class") == "unjustified":
                # A removal exists only in the start tree or the diff.
                reason = cited(roots, finding.get("cite"), finding.get("quote", ""), diff) if finding.get("cite") else "an unjustified change needs a citation"
        results.append({"item": item, "finding": finding, "valid": reason is None, "reason": reason})
    return {"token": review.get("token"), "criticals": results,
            "valid_criticals": sum(r["valid"] for r in results)}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--token-dir", required=True)
    p.add_argument("--requirements", required=True)
    p.add_argument("--review", required=True)
    p.add_argument("--s3", action="store_true", help="S3 with a non-conforming start: optional changes are critical too")
    p.add_argument("--entry", default=LIBRARY_ENTRY, help="the program's main package for R1a, relative to the candidate")
    a = p.parse_args(argv)
    result = check(a.token_dir, a.requirements, json.loads(pathlib.Path(a.review).read_text(encoding="utf-8")), a.s3, a.entry)
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
