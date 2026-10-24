"""Hidden behavior cases for Phase B of the single-agent comparison.

The Library oracle stays unchanged and knows only M1 and M2. Phase B's
snapshots M3 (short-loan copies), M4 (renewals) and M5 (no member suspension)
are judged by the cases built here, cumulatively, with the oracle's own case
helpers, transport, state snapshots and witness discrimination:

  carried  the oracle's m2 cases and the six Phase A probes; for M5 the steps
           that call setMemberStatus and the suspension case E05 are dropped
           and member expectations no longer require a status
  T1       loan classes, 7- and 14-day due dates, the due-date limit per class
  T2       renewals, including overdue assessment after a renewal
  T3       setMemberStatus is an unknown operation

Every new step cites the snapshot sentence it checks in its family docstring
below, and both requirement representations carry that sentence verbatim.
Writers never see this file.

Inside the assessment container (see assess_candidate.py):
  python -I /harness/phase_b_cases.py --profile m3|m4|m5 --source S --commit C --limits JSON --protected JSON
"""
import argparse
import copy
import hashlib
import json
import pathlib
import sys

HERE = pathlib.Path(__file__).resolve().parent
EVALUATOR = pathlib.Path("/evaluator")
if EVALUATOR.is_dir():
    sys.path.insert(0, str(EVALUATOR))
sys.path.insert(0, str(HERE))

import oracle  # noqa: E402
from oracle import _case, _copy, _member, _ok, _refused, _snapshot_projection, _step, _symbolic_loan  # noqa: E402
import probes  # noqa: E402

PROFILES = ("m3", "m4", "m5")
DUE = "2026-10-15"


# --- Records --------------------------------------------------------------------

def copy_class(copy_id, title, loan_class, available=True):
    record = _copy(copy_id, title, available)
    record["copy"]["loanClass"] = loan_class
    return record


def loan(symbol, member, copy_id, borrow, due, status="open", returned=None, renewals=0, capture=False):
    record = {"loanId": {"$capture" if capture else "$ref": symbol}, "memberId": member, "copyId": copy_id,
              "borrowDate": borrow, "dueDate": due, "status": status, "returnDate": returned, "renewals": renewals}
    return {"loan": record}


def new_loan(symbol, member, copy_id, borrow, due):
    return loan(symbol, member, copy_id, borrow, due, capture=True)


def notice(symbol, member, copy_id, due):
    return {"loanId": {"$ref": symbol}, "memberId": member, "copyId": copy_id, "dueDate": due}


def raw(line, request_id):
    """A literal request line expected to be invalid."""
    return {"raw": line, "expect": {"id": request_id, "status": "invalidRequest", "reason": {"$nonempty": True}}}


# --- Snapshot expansion that also follows renewals -------------------------------

def _apply(step, members, copies, loans):
    oracle._apply_expected_record(step, members, copies, loans)
    request = oracle._request_for_expectation(step)
    expected = step.get("expect", {})
    if request and request.get("op") == "renew" and expected.get("status") == "ok":
        pair = _symbolic_loan(expected.get("result", {}))
        if pair is not None:
            loans[pair[0]] = pair[1]


def expand(case):
    """oracle._expand_observation_snapshots with renewals applied to the expected state."""
    members, copies, loans, out, serial = {}, {}, {}, [], 0
    for step in case["steps"]:
        bracket = oracle._needs_observation_snapshot(step, members)
        if bracket:
            serial += 1
            out.extend(_snapshot_projection(case["id"], serial, members, copies, loans))
        out.append(step)
        _apply(step, members, copies, loans)
        if bracket:
            serial += 1
            out.extend(_snapshot_projection(case["id"], serial, members, copies, loans))
    return {**case, "steps": out}


def strip(case):
    return {**case, "steps": [s for s in case["steps"] if not s.get("snapshot")]}


def reexpand(case):
    return oracle._expand_paired_witness(case) if case["id"] == "paired-fourth-loan" else expand(case)


# --- The M5 transform --------------------------------------------------------------

def _calls_set_status(step):
    request = oracle._request_for_expectation(step)
    return bool(request) and request.get("op") == "setMemberStatus"


def _without_status(value):
    if isinstance(value, dict):
        out = {k: _without_status(v) for k, v in value.items()}
        member = out.get("member")
        if isinstance(member, dict):
            out["member"] = {k: v for k, v in member.items() if k != "status"}
        return out
    if isinstance(value, list):
        return [_without_status(v) for v in value]
    return value


def no_suspension(case):
    """M5 removes suspension: drop setMemberStatus steps and member status expectations."""
    steps = [s for s in strip(case)["steps"] if not _calls_set_status(s)]
    return {**case, "steps": [{**s, "expect": _without_status(s["expect"])} for s in steps]}


# --- New families -------------------------------------------------------------------

def t1_cases():
    """Short-loan copies (M3).

    register-copy: "Without a loan class the copy is standard; a loan class other
    than exactly standard or short is refused without changing any state."
    inspect-copy: "return its identifier, title, loan class and current availability".
    loan-policy: "plus the value of loanPeriodDays for a standard copy, or plus the
    value of shortLoanPeriodDays for a short copy" (14 and 7).
    borrow: "the new loan's due date falls on or before 9999-12-31".
    assess-overdue: "due dates ... strictly earlier than the assessment date".
    """
    cases = []
    c = "T1-loan-class"; s = []
    s += [_step(c, 1, "registerCopy", {"copyId": "short-1", "title": "Quick", "loanClass": "short"}, _ok(copy_class("short-1", "Quick", "short")))]
    s += [_step(c, 2, "registerCopy", {"copyId": "std-1", "title": "Slow"}, _ok(copy_class("std-1", "Slow", "standard")))]
    s += [_step(c, 3, "registerCopy", {"copyId": "std-2", "title": "Slow", "loanClass": "standard"}, _ok(copy_class("std-2", "Slow", "standard")))]
    s += [_step(c, 4, "registerCopy", {"copyId": "bad-1", "title": "Bad", "loanClass": "Short"}, _refused())]
    s += [_step(c, 5, "registerCopy", {"copyId": "bad-2", "title": "Bad", "loanClass": ""}, _refused())]
    s += [_step(c, 6, "registerCopy", {"copyId": "bad-3", "title": "Bad", "loanClass": "reference"}, _refused())]
    s += [_step(c, 7, "inspectCopy", {"copyId": "short-1"}, _ok(copy_class("short-1", "Quick", "short")))]
    for i, cid in ((8, "bad-1"), (9, "bad-2"), (10, "bad-3")):
        s += [_step(c, i, "inspectCopy", {"copyId": cid}, _refused())]
    cases.append(_case(c, ["T1"], s))

    c = "T1-short-loans"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 2, "registerCopy", {"copyId": "short-1", "title": "Quick", "loanClass": "short"}, _ok(copy_class("short-1", "Quick", "short")))]
    s += [_step(c, 3, "registerCopy", {"copyId": "std-1", "title": "Slow"}, _ok(copy_class("std-1", "Slow", "standard")))]
    s += [_step(c, 4, "borrow", {"memberId": "member-a", "copyId": "short-1", "borrowDate": "2026-10-01"},
                _ok(_new_class_loan("loan-s", "member-a", "short-1", "2026-10-01", "2026-10-08")), {"loan-s": "result.loan.loanId"})]
    s += [_step(c, 5, "borrow", {"memberId": "member-a", "copyId": "std-1", "borrowDate": "2026-10-01"},
                _ok(_new_class_loan("loan-d", "member-a", "std-1", "2026-10-01", DUE)), {"loan-d": "result.loan.loanId"})]
    s += [_step(c, 6, "assessOverdue", {"assessmentDate": "2026-10-08"}, _ok({"notices": []}))]
    s += [_step(c, 7, "assessOverdue", {"assessmentDate": "2026-10-09"}, _ok({"notices": [notice("loan-s", "member-a", "short-1", "2026-10-08")]}))]
    s += [_step(c, 8, "assessOverdue", {"assessmentDate": "2026-10-16"}, _ok({"notices": [notice("loan-s", "member-a", "short-1", "2026-10-08"),
                                                                                       notice("loan-d", "member-a", "std-1", DUE)]}))]
    cases.append(_case(c, ["T1"], s))

    c = "T1-due-date-range-by-class"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 2, "registerCopy", {"copyId": "short-1", "title": "Quick", "loanClass": "short"}, _ok(copy_class("short-1", "Quick", "short")))]
    s += [_step(c, 3, "registerCopy", {"copyId": "short-2", "title": "Quick", "loanClass": "short"}, _ok(copy_class("short-2", "Quick", "short")))]
    s += [_step(c, 4, "registerCopy", {"copyId": "std-1", "title": "Slow"}, _ok(copy_class("std-1", "Slow", "standard")))]
    s += [_step(c, 5, "borrow", {"memberId": "member-a", "copyId": "short-1", "borrowDate": "9999-12-25"}, _refused())]
    s += [_step(c, 6, "borrow", {"memberId": "member-a", "copyId": "short-1", "borrowDate": "9999-12-24"},
                _ok(_new_class_loan("loan-s", "member-a", "short-1", "9999-12-24", "9999-12-31")), {"loan-s": "result.loan.loanId"})]
    s += [_step(c, 7, "borrow", {"memberId": "member-a", "copyId": "std-1", "borrowDate": "9999-12-24"}, _refused())]
    s += [_step(c, 8, "borrow", {"memberId": "member-a", "copyId": "std-1", "borrowDate": "9999-12-17"},
                _ok(_new_class_loan("loan-d", "member-a", "std-1", "9999-12-17", "9999-12-31")), {"loan-d": "result.loan.loanId"})]
    s += [_step(c, 9, "inspectCopy", {"copyId": "short-2"}, _ok(copy_class("short-2", "Quick", "short")))]
    cases.append(_case(c, ["T1"], s))
    return cases


def _new_class_loan(symbol, member, copy_id, borrow, due):
    """A new loan before M4 carries no renewal count."""
    record = new_loan(symbol, member, copy_id, borrow, due)
    del record["loan"]["renewals"]
    return record


def t2_cases(profile):
    """Renewals (M4).

    renew: "Admit the renewal only when the loan is open, its member is registered
    and active under the eligibility rule, the renewal date is on or after the
    borrowing date and on or before the current due date, the loan has been
    renewed fewer times than the value of maxRenewals in the lending policy, and
    the new due date falls on or before 9999-12-31. Success moves the due date
    forward by the loan period of the loan's copy ... counted from the current due
    date, adds one to the loan's renewal count and returns the loan."
    loan-policy: "each renewal replaces the loan's due date with the later one,
    which is then the loan's current due date."
    eligibility: "Suspension prevents new borrowing and renewal" (M4 only).
    callable-interface: loan records carry "`renewals` (integer, `0` until renewed)";
    `renew` takes `loanId`, `renewalDate`; unknown fields are invalid.
    """
    cases = []
    m = ("member-a", "Ada")

    def start(c, copies):
        s = [_step(c, 1, "registerMember", {"memberId": m[0], "name": m[1]}, _ok(_member(*m)))]
        for i, (cid, cls) in enumerate(copies, 2):
            args = {"copyId": cid, "title": "Book " + cid}
            if cls == "short":
                args["loanClass"] = "short"
            s += [_step(c, i, "registerCopy", args, _ok(copy_class(cid, "Book " + cid, cls)))]
        return s

    def borrow(c, n, symbol, cid, date, due):
        return _step(c, n, "borrow", {"memberId": m[0], "copyId": cid, "borrowDate": date},
                     _ok(new_loan(symbol, m[0], cid, date, due)), {symbol: "result.loan.loanId"})

    def renew(c, n, symbol, date, expected):
        return _step(c, n, "renew", {"loanId": {"$ref": symbol}, "renewalDate": date}, expected)

    c = "T2-renew-once"; s = start(c, [("std-1", "standard")])
    s += [borrow(c, 10, "loan-1", "std-1", "2026-10-01", DUE)]
    s += [renew(c, 11, "loan-1", "2026-10-10", _ok(loan("loan-1", m[0], "std-1", "2026-10-01", "2026-10-29", renewals=1)))]
    s += [_step(c, 12, "inspectLoan", {"loanId": {"$ref": "loan-1"}}, _ok(loan("loan-1", m[0], "std-1", "2026-10-01", "2026-10-29", renewals=1)))]
    s += [_step(c, 13, "inspectOpenLoans", {"memberId": m[0]}, _ok({"memberId": m[0], "count": 1,
                                                                     "loans": [loan("loan-1", m[0], "std-1", "2026-10-01", "2026-10-29", renewals=1)["loan"]]}))]
    s += [_step(c, 14, "inspectCopy", {"copyId": "std-1"}, _ok(copy_class("std-1", "Book std-1", "standard", False)))]
    s += [renew(c, 15, "loan-1", "2026-10-20", _refused())]
    s += [_step(c, 16, "assessOverdue", {"assessmentDate": "2026-10-16"}, _ok({"notices": []}))]
    s += [_step(c, 17, "assessOverdue", {"assessmentDate": "2026-10-29"}, _ok({"notices": []}))]
    s += [_step(c, 18, "assessOverdue", {"assessmentDate": "2026-10-30"}, _ok({"notices": [notice("loan-1", m[0], "std-1", "2026-10-29")]}))]
    cases.append(_case(c, ["T2", "composition"], s))

    c = "T2-renew-window"; s = start(c, [("std-1", "standard"), ("std-2", "standard"), ("std-3", "standard")])
    s += [borrow(c, 10, "loan-1", "std-1", "2026-10-01", DUE), borrow(c, 11, "loan-2", "std-2", "2026-10-01", DUE),
          borrow(c, 12, "loan-3", "std-3", "2026-10-05", "2026-10-19")]
    s += [renew(c, 13, "loan-1", DUE, _ok(loan("loan-1", m[0], "std-1", "2026-10-01", "2026-10-29", renewals=1)))]
    s += [renew(c, 14, "loan-2", "2026-10-16", _refused())]
    s += [renew(c, 15, "loan-3", "2026-10-04", _refused())]
    s += [renew(c, 16, "loan-3", "2026-02-30", _refused())]
    s += [_step(c, 17, "inspectLoan", {"loanId": {"$ref": "loan-2"}}, _ok(loan("loan-2", m[0], "std-2", "2026-10-01", DUE)))]
    cases.append(_case(c, ["T2"], s))

    if profile == "m4":
        c = "T2-renew-suspended"; s = start(c, [("std-1", "standard")])
        s += [borrow(c, 10, "loan-1", "std-1", "2026-10-01", DUE)]
        s += [_step(c, 11, "setMemberStatus", {"memberId": m[0], "status": "suspended"}, _ok(_member(m[0], m[1], "suspended")))]
        s += [renew(c, 12, "loan-1", "2026-10-05", _refused())]
        s += [_step(c, 13, "setMemberStatus", {"memberId": m[0], "status": "active"}, _ok(_member(*m)))]
        s += [renew(c, 14, "loan-1", "2026-10-05", _ok(loan("loan-1", m[0], "std-1", "2026-10-01", "2026-10-29", renewals=1)))]
        cases.append(_case(c, ["T2"], s))

    c = "T2-renew-closed-and-invalid"; s = start(c, [("std-1", "standard")])
    s += [borrow(c, 10, "loan-1", "std-1", "2026-10-01", DUE)]
    s += [_step(c, 11, "return", {"loanId": {"$ref": "loan-1"}, "returnDate": "2026-10-03"},
                _ok({"outcome": "returned", **loan("loan-1", m[0], "std-1", "2026-10-01", DUE, "closed", "2026-10-03")}))]
    s += [renew(c, 12, "loan-1", "2026-10-03", _refused())]
    s += [_step(c, 13, "renew", {"loanId": {"$unusedLoan": "missing-loan"}, "renewalDate": "2026-10-03"}, _refused())]
    s += [_step(c, 14, "renew", {"loanId": {"$ref": "loan-1"}}, _refused())]
    s += [_step(c, 15, "renew", {"loanId": 17, "renewalDate": "2026-10-03"}, _refused())]
    cases.append(_case(c, ["T2"], s))

    # An otherwise valid renewal of an open loan with an unknown field is invalid.
    c = "T2-renew-unknown-field"; s = start(c, [("std-1", "standard")])
    s += [borrow(c, 10, "loan-1", "std-1", "2026-10-01", DUE)]
    s += [_step(c, 11, "renew", {"loanId": {"$ref": "loan-1"}, "renewalDate": "2026-10-05", "extra": True},
                {"status": "invalidRequest", "reason": {"$nonempty": True}})]
    s += [_step(c, 12, "inspectLoan", {"loanId": {"$ref": "loan-1"}}, _ok(loan("loan-1", m[0], "std-1", "2026-10-01", DUE)))]
    cases.append(_case(c, ["T2"], s))

    c = "T2-renew-short-and-range"; s = start(c, [("short-1", "short"), ("std-1", "standard"), ("short-2", "short")])
    s += [borrow(c, 10, "loan-s", "short-1", "2026-10-01", "2026-10-08")]
    s += [renew(c, 11, "loan-s", "2026-10-05", _ok(loan("loan-s", m[0], "short-1", "2026-10-01", DUE, renewals=1)))]
    s += [_step(c, 12, "return", {"loanId": {"$ref": "loan-s"}, "returnDate": "2026-10-06"},
                _ok({"outcome": "returned", **loan("loan-s", m[0], "short-1", "2026-10-01", DUE, "closed", "2026-10-06", renewals=1)}))]
    s += [borrow(c, 13, "loan-d", "std-1", "9999-12-17", "9999-12-31")]
    s += [renew(c, 14, "loan-d", "9999-12-20", _refused())]
    s += [borrow(c, 15, "loan-s2", "short-1", "9999-12-17", "9999-12-24")]
    s += [renew(c, 16, "loan-s2", "9999-12-20", _ok(loan("loan-s2", m[0], "short-1", "9999-12-17", "9999-12-31", renewals=1)))]
    s += [borrow(c, 17, "loan-s3", "short-2", "9999-12-20", "9999-12-27")]
    s += [renew(c, 18, "loan-s3", "9999-12-27", _refused())]
    cases.append(_case(c, ["T2"], s))
    return cases


def t3_cases():
    """Removal of member suspension (M5).

    callable-interface: its operations table no longer lists `setMemberStatus`,
    and "A malformed JSON line, an invalid request envelope or an unknown operation
    returns `status: invalidRequest`". Member records carry `memberId`, `name`.
    eligibility: "membership never blocks borrowing, renewal or return once the
    member is registered".
    """
    c = "T3-no-member-status"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok({"member": {"memberId": "member-a", "name": "Ada"}}))]
    s += [_step(c, 2, "registerCopy", {"copyId": "std-1", "title": "Slow"}, _ok(copy_class("std-1", "Slow", "standard")))]
    s += [raw('{"id":"no-status","op":"setMemberStatus","args":{"memberId":"member-a","status":"suspended"}}', "no-status")]
    s += [_step(c, 3, "inspectMember", {"memberId": "member-a"}, _ok({"member": {"memberId": "member-a", "name": "Ada"}}))]
    s += [_step(c, 4, "borrow", {"memberId": "member-a", "copyId": "std-1", "borrowDate": "2026-10-01"},
                _ok(new_loan("loan-1", "member-a", "std-1", "2026-10-01", DUE)), {"loan-1": "result.loan.loanId"})]
    s += [_step(c, 5, "renew", {"loanId": {"$ref": "loan-1"}, "renewalDate": "2026-10-05"},
                _ok(loan("loan-1", "member-a", "std-1", "2026-10-01", "2026-10-29", renewals=1)))]
    return [_case(c, ["T3"], s)]


def build_cases(profile):
    """All hidden cases of a Phase B profile, carried and new, expanded with snapshots."""
    if profile not in PROFILES:
        raise ValueError("profile must be m3, m4 or m5")
    cases = [strip(c) for c in oracle.build_cases("m2")] + [strip(c) for c in probes.build_probes()]
    cases += t1_cases()
    if profile in ("m4", "m5"):
        cases += t2_cases(profile)
    if profile == "m5":
        cases = [no_suspension(c) for c in cases if c["id"] != "E05-suspension"] + t3_cases()
    return [reexpand(c) for c in cases]


def witness(profile):
    """The opposite-limit witness: M1's five-loan sequence, transformed like the profile."""
    case = strip(oracle.witness_case("m1"))
    return no_suspension(case) if profile == "m5" else case


def discrimination(profile, command, cwd, limits, run_case):
    """As the oracle's: the five-loan witness must fail exactly at the fourth loan."""
    case = copy.deepcopy(witness(profile))
    for step in case["steps"]:
        if step.get("witness") == "open-loan-count":
            step["expect"]["result"].pop("loans", None)
    result = run_case(case, command, cwd, limits, keep_going=True)
    labels = {step["witness"] for step in result["steps"] if step["errors"]}
    from_steps = sum(len(step["errors"]) for step in result["steps"])
    result["expected_mismatch_confirmed"] = (
        result["executed_steps"] == result["planned_steps"]
        and labels == {"fourth-admission", "open-loan-count", "copy-4-availability"}
        and len(result["errors"]) == from_steps and result["trace"]["exit_code"] == 0)
    return result


def main():
    import assess  # the oracle's assessment plumbing from /evaluator
    from driver import Limits

    p = argparse.ArgumentParser()
    p.add_argument("--profile", choices=PROFILES, required=True)
    p.add_argument("--source", required=True)
    p.add_argument("--commit", required=True)
    p.add_argument("--limits", required=True)
    p.add_argument("--protected", required=True, help="evaluator and harness hashes computed on the Host")
    a = p.parse_args()
    paths = probes.protected_paths(EVALUATOR, HERE)
    paths["harness/phase_b_cases.py"] = HERE / "phase_b_cases.py"
    for name in ("expectations/m1.json", "expectations/m2.json"):
        paths["evaluator/" + name] = EVALUATOR / name
    expected = json.loads(a.protected)
    report = {"profile": a.profile, "commit": a.commit, "source": a.source, "protected_sha256": expected, "cases": [], "errors": []}
    with open("/out/library", "rb") as f:
        report["program_sha256"] = hashlib.sha256(f.read()).hexdigest()
    if probes.file_hashes(paths) != expected:
        report["errors"].append("evaluator or harness bytes differ from the Host-computed hashes")
    else:
        limits = Limits(**json.loads(a.limits))
        cases = build_cases(a.profile)
        report["planned_cases"] = len(cases)
        for case in cases:
            report["cases"].append(assess.run_case(case, ["/out/library"], "/assess/realization", limits))
        report["discrimination"] = discrimination(a.profile, ["/out/library"], "/assess/realization", limits, assess.run_case)
        if probes.file_hashes(paths) != expected:
            report["errors"].append("evaluator or harness bytes changed during the cases")
    report["passed"] = (not report["errors"] and bool(report["cases"]) and all(c["passed"] for c in report["cases"])
                        and report.get("discrimination", {}).get("expected_mismatch_confirmed") is True)
    json.dump(report, sys.stdout)
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
