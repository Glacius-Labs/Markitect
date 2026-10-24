"""Supplementary hidden behavior probes for the single-agent comparison.

The Library oracle stays unchanged. These six cases exercise stated Library
obligations the oracle does not reach, in the oracle's own case format and
through its own transport, step evaluation and state snapshots:

  X01 due-date range: a due date after 9999-12-31 is refused, 9999-12-31 is not
  X02 a field whose name differs from a stated one only in case is invalid
  X03 a null required argument is a refusal
  X04 a request without an args object is invalid
  X05 a non-string or missing id is echoed as null
  X06 year 0000 is not a valid date; years 0001 and 9999 are

The obligations are stated in the borrow Definition and the callable interface,
and verbatim in arm B's requirements. Writers never see this file.

Inside the assessment container (see assess_candidate.py):
  python -I /harness/probes.py --source S --commit C --limits JSON --protected JSON
"""
import argparse
import hashlib
import json
import pathlib
import sys

EVALUATOR = pathlib.Path("/evaluator")
if str(EVALUATOR) not in sys.path and EVALUATOR.is_dir():
    sys.path.insert(0, str(EVALUATOR))

from oracle import _case, _copy, _expand_observation_snapshots, _member, _new_loan, _ok, _refused, _step  # noqa: E402


def _invalid(request_id):
    return {"id": request_id, "status": "invalidRequest", "reason": {"$nonempty": True}}


def _raw(line, request_id):
    return {"raw": line, "expect": _invalid(request_id)}


def build_probes():
    cases = []

    c = "X01-due-date-range"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-x", "name": "Xena"}, _ok(_member("member-x", "Xena")))]
    s += [_step(c, 2, "registerCopy", {"copyId": "copy-x", "title": "Late Book"}, _ok(_copy("copy-x", "Late Book")))]
    s += [_step(c, 3, "borrow", {"memberId": "member-x", "copyId": "copy-x", "borrowDate": "9999-12-18"}, _refused())]
    s += [_step(c, 4, "borrow", {"memberId": "member-x", "copyId": "copy-x", "borrowDate": "9999-12-17"},
                _ok(_new_loan("loan-x", "member-x", "copy-x", "9999-12-17", "9999-12-31")), {"loan-x": "result.loan.loanId"})]
    s += [_step(c, 5, "inspectCopy", {"copyId": "copy-x"}, _ok(_copy("copy-x", "Late Book", False)))]
    cases.append(_case(c, ["X01"], s))

    c = "X02-case-variant-field"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-x", "name": "Xena"}, _ok(_member("member-x", "Xena")))]
    s += [_raw('{"id":"case-arg","op":"inspectMember","args":{"memberId":"member-x","MemberId":"member-x"}}', "case-arg")]
    s += [_raw('{"id":"case-envelope","op":"inspectMember","Op":"inspectMember","args":{"memberId":"member-x"}}', "case-envelope")]
    s += [_raw('{"id":"case-register","op":"registerMember","args":{"memberId":"member-y","name":"Yara","Name":"Yara"}}', "case-register")]
    s += [_step(c, 2, "inspectMember", {"memberId": "member-y"}, _refused())]
    s += [_step(c, 3, "inspectMember", {"memberId": "member-x"}, _ok(_member("member-x", "Xena")))]
    cases.append(_case(c, ["X02"], s))

    c = "X03-null-argument"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-x", "name": "Xena"}, _ok(_member("member-x", "Xena")))]
    s += [_step(c, 2, "registerCopy", {"copyId": "copy-x", "title": "Null Book"}, _ok(_copy("copy-x", "Null Book")))]
    s += [_step(c, 3, "registerMember", {"memberId": "member-null", "name": None}, _refused())]
    s += [_step(c, 4, "inspectMember", {"memberId": "member-null"}, _refused())]
    s += [_step(c, 5, "borrow", {"memberId": "member-x", "copyId": "copy-x", "borrowDate": None}, _refused())]
    s += [_step(c, 6, "setMemberStatus", {"memberId": "member-x", "status": None}, _refused())]
    s += [_step(c, 7, "inspectCopy", {"copyId": "copy-x"}, _ok(_copy("copy-x", "Null Book")))]
    cases.append(_case(c, ["X03"], s))

    c = "X04-missing-args"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-x", "name": "Xena"}, _ok(_member("member-x", "Xena")))]
    s += [_raw('{"id":"no-args","op":"inspectMember"}', "no-args")]
    s += [_raw('{"id":"null-args","op":"inspectMember","args":null}', "null-args")]
    s += [_step(c, 2, "inspectMember", {"memberId": "member-x"}, _ok(_member("member-x", "Xena")))]
    cases.append(_case(c, ["X04"], s))

    c = "X05-null-id-echo"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-x", "name": "Xena"}, _ok(_member("member-x", "Xena")))]
    s += [_raw('{"id":7,"op":"inspectMember","args":{"memberId":"member-x"}}', None)]
    s += [_raw('{"op":"inspectMember","args":{"memberId":"member-x"}}', None)]
    s += [_raw('["inspectMember"]', None)]
    s += [_step(c, 2, "inspectMember", {"memberId": "member-x"}, _ok(_member("member-x", "Xena")))]
    cases.append(_case(c, ["X05"], s))

    c = "X06-year-range"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-x", "name": "Xena"}, _ok(_member("member-x", "Xena")))]
    s += [_step(c, 2, "registerCopy", {"copyId": "copy-x", "title": "Old Book"}, _ok(_copy("copy-x", "Old Book")))]
    s += [_step(c, 3, "borrow", {"memberId": "member-x", "copyId": "copy-x", "borrowDate": "0000-06-01"}, _refused())]
    s += [_step(c, 4, "assessOverdue", {"assessmentDate": "0000-12-31"}, _refused())]
    s += [_step(c, 5, "assessOverdue", {"assessmentDate": "0001-01-01"}, _ok({"notices": []}))]
    s += [_step(c, 6, "assessOverdue", {"assessmentDate": "9999-12-31"}, _ok({"notices": []}))]
    s += [_step(c, 7, "inspectCopy", {"copyId": "copy-x"}, _ok(_copy("copy-x", "Old Book")))]
    cases.append(_case(c, ["X06"], s))

    return [_expand_observation_snapshots(case) for case in cases]


def file_hashes(paths):
    return {name: hashlib.sha256(path.read_bytes()).hexdigest() for name, path in paths.items()}


def protected_paths(evaluator, harness):
    paths = {"evaluator/" + n: evaluator / n for n in ("assess.py", "driver.py", "oracle.py")}
    paths["harness/probes.py"] = harness / "probes.py"
    return paths


def main():
    import assess  # the oracle's assessment plumbing from /evaluator
    from driver import Limits

    p = argparse.ArgumentParser()
    p.add_argument("--source", required=True)
    p.add_argument("--commit", required=True)
    p.add_argument("--limits", required=True)
    p.add_argument("--protected", required=True, help="evaluator and probe hashes computed on the Host")
    a = p.parse_args()
    expected = json.loads(a.protected)
    paths = protected_paths(EVALUATOR, pathlib.Path(__file__).resolve().parent)
    report = {"commit": a.commit, "source": a.source, "protected_sha256": expected, "cases": [], "errors": []}
    with open("/out/library", "rb") as f:
        report["program_sha256"] = hashlib.sha256(f.read()).hexdigest()
    if file_hashes(paths) != expected:
        report["errors"].append("evaluator or probe bytes differ from the Host-computed hashes")
    else:
        limits = Limits(**json.loads(a.limits))
        for case in build_probes():
            result = assess.run_case(case, ["/out/library"], "/assess/realization", limits)
            report["cases"].append(result)
        if file_hashes(paths) != expected:
            report["errors"].append("evaluator or probe bytes changed during the probes")
    report["passed"] = not report["errors"] and len(report["cases"]) == 6 and all(c["passed"] for c in report["cases"])
    json.dump(report, sys.stdout)
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
