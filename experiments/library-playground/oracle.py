"""Independent, standard-library oracle for the library JSONL contract.

Expected policy values are frozen in expectations/*.json.  The cases drive only
the public callable interface; this module contains no library implementation.
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any


_ROOT = Path(__file__).resolve().parent


def expectation_profile(profile: str) -> dict[str, Any]:
    """Load a deliberately authored M1 or M2 policy expectation."""
    if profile not in ("m1", "m2"):
        raise ValueError("profile must be 'm1' or 'm2'")
    return json.loads((_ROOT / "expectations" / f"{profile}.json").read_text(encoding="utf-8"))


def _step(case_id: str, index: int, op: str, args: dict[str, Any], expected: dict[str, Any], capture: dict[str, str] | None = None) -> dict[str, Any]:
    result: dict[str, Any] = {
        "request": {"id": f"{case_id}-{index:03d}", "op": op, "args": args},
        "expect": {"id": f"{case_id}-{index:03d}", **expected},
    }
    if capture:
        result["capture"] = capture
    return result


def _ok(result: dict[str, Any]) -> dict[str, Any]:
    return {"status": "ok", "result": result}


def _refused() -> dict[str, Any]:
    return {"status": "refused", "reason": {"$nonempty": True}}


def _member(member_id: str, name: str, status: str = "active") -> dict[str, Any]:
    return {"member": {"memberId": member_id, "name": name, "status": status}}


def _copy(copy_id: str, title: str, available: bool = True) -> dict[str, Any]:
    return {"copy": {"copyId": copy_id, "title": title, "available": available}}


def _loan(symbol: str, member: str, copy: str, borrow_date: str, due_date: str, status: str = "open", return_date: str | None = None) -> dict[str, Any]:
    return {"loan": {"loanId": {"$ref": symbol}, "memberId": member, "copyId": copy,
                      "borrowDate": borrow_date, "dueDate": due_date, "status": status,
                      "returnDate": return_date}}


def _new_loan(symbol: str, member: str, copy: str, borrow_date: str, due_date: str) -> dict[str, Any]:
    """Expected shape for a newly generated opaque ID in its capture step."""
    value = _loan(symbol, member, copy, borrow_date, due_date)
    value["loan"]["loanId"] = {"$capture": symbol}
    return value


def _case(case_id: str, families: list[str], steps: list[dict[str, Any]]) -> dict[str, Any]:
    return {"id": case_id, "families": families, "steps": steps}


def build_cases(profile: str) -> list[dict[str, Any]]:
    """Return finite, explicit operation sequences for one frozen profile."""
    policy = expectation_profile(profile)
    limit = policy["maxOpenLoans"]
    days = policy["loanPeriodDays"]
    if (limit, days) not in ((5, 14), (3, 14)):
        raise ValueError("only the independently frozen M1 (5/14) and M2 (3/14) profiles are supported")
    due = "2026-10-15"
    cases: list[dict[str, Any]] = []

    # E01 membership lifecycle, duplicate protection, idempotence and reads.
    c = "E01-membership"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 2, "registerMember", {"memberId": "member-a", "name": "Changed"}, _refused())]
    s += [_step(c, 3, "inspectMember", {"memberId": "member-a"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 4, "setMemberStatus", {"memberId": "member-a", "status": "suspended"}, _ok(_member("member-a", "Ada", "suspended")))]
    s += [_step(c, 5, "setMemberStatus", {"memberId": "member-a", "status": "suspended"}, _ok(_member("member-a", "Ada", "suspended")))]
    s += [_step(c, 6, "setMemberStatus", {"memberId": "member-a", "status": "active"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 7, "setMemberStatus", {"memberId": "absent", "status": "active"}, _refused())]
    s += [_step(c, 8, "inspectMember", {"memberId": "absent"}, _refused())]
    s += [_step(c, 9, "inspectMember", {"memberId": "member-a"}, _ok(_member("member-a", "Ada")))]
    cases.append(_case(c, ["E01"], s))

    # E02 copies, duplicate protection, equal titles and derived availability.
    c = "E02-copies"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 2, "registerMember", {"memberId": "member-b", "name": "Bea"}, _ok(_member("member-b", "Bea")))]
    s += [_step(c, 3, "registerCopy", {"copyId": "copy-a", "title": "Shared title"}, _ok(_copy("copy-a", "Shared title")))]
    s += [_step(c, 4, "registerCopy", {"copyId": "copy-b", "title": "Shared title"}, _ok(_copy("copy-b", "Shared title")))]
    s += [_step(c, 5, "registerCopy", {"copyId": "copy-a", "title": "Overwrite"}, _refused())]
    s += [_step(c, 6, "borrow", {"memberId": "member-a", "copyId": "copy-a", "borrowDate": "2026-10-01"}, _ok(_new_loan("loan-a", "member-a", "copy-a", "2026-10-01", due)), {"loan-a": "result.loan.loanId"})]
    s += [_step(c, 7, "borrow", {"memberId": "member-b", "copyId": "copy-a", "borrowDate": "2026-10-01"}, _refused())]
    s += [_step(c, 8, "inspectCopy", {"copyId": "copy-a"}, _ok(_copy("copy-a", "Shared title", False)))]
    s += [_step(c, 9, "inspectLoan", {"loanId": {"$ref": "loan-a"}}, _ok(_loan("loan-a", "member-a", "copy-a", "2026-10-01", due)))]
    s += [_step(c, 10, "inspectOpenLoans", {"memberId": "member-a"}, _ok({"memberId": "member-a", "count": 1, "loans": [_loan("loan-a", "member-a", "copy-a", "2026-10-01", due)["loan"]]}))]
    s += [_step(c, 11, "inspectCopy", {"copyId": "copy-b"}, _ok(_copy("copy-b", "Shared title")))]
    s += [_step(c, 14, "borrow", {"memberId": "member-b", "copyId": "copy-b", "borrowDate": "2026-10-01"}, _ok(_new_loan("loan-b", "member-b", "copy-b", "2026-10-01", due)), {"loan-b": "result.loan.loanId"})]
    s += [_step(c, 12, "return", {"loanId": {"$ref": "loan-a"}, "returnDate": "2026-10-02"}, _ok({"outcome": "returned", **_loan("loan-a", "member-a", "copy-a", "2026-10-01", due, "closed", "2026-10-02")}))]
    s += [_step(c, 13, "inspectCopy", {"copyId": "copy-a"}, _ok(_copy("copy-a", "Shared title")))]
    cases.append(_case(c, ["E02", "E03", "E07"], s))

    # E03 loan shape, stable ID, count, copy state, duplicate borrow refusal.
    c = "E03-successful-loan"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    for i in range(1, 3):
        s += [_step(c, i + 1, "registerCopy", {"copyId": f"copy-{i}", "title": f"Book {i}"}, _ok(_copy(f"copy-{i}", f"Book {i}")))]
    s += [_step(c, 4, "borrow", {"memberId": "member-a", "copyId": "copy-1", "borrowDate": "2026-10-01"}, _ok(_new_loan("loan-a", "member-a", "copy-1", "2026-10-01", due)), {"loan-a": "result.loan.loanId"})]
    s += [_step(c, 5, "borrow", {"memberId": "member-a", "copyId": "copy-1", "borrowDate": "2026-10-01"}, _refused())]
    s += [_step(c, 6, "inspectOpenLoans", {"memberId": "member-a"}, _ok({"memberId": "member-a", "count": 1, "loans": [_loan("loan-a", "member-a", "copy-1", "2026-10-01", due)["loan"]]}))]
    s += [_step(c, 7, "inspectCopy", {"copyId": "copy-1"}, _ok(_copy("copy-1", "Book 1", False)))]
    s += [_step(c, 8, "inspectCopy", {"copyId": "copy-2"}, _ok(_copy("copy-2", "Book 2")))]
    cases.append(_case(c, ["E03"], s))

    # E04 limit boundary: open overdue loans count, closed loans release a slot.
    c = "E04-admission"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    for i in range(1, limit + 3):
        s += [_step(c, i + 1, "registerCopy", {"copyId": f"copy-{i}", "title": f"Book {i}"}, _ok(_copy(f"copy-{i}", f"Book {i}")))]
    for i in range(1, limit + 1):
        s += [_step(c, limit + i + 2, "borrow", {"memberId": "member-a", "copyId": f"copy-{i}", "borrowDate": "2026-10-01"}, _ok(_new_loan(f"loan-{i}", "member-a", f"copy-{i}", "2026-10-01", due)), {f"loan-{i}": "result.loan.loanId"})]
    s += [_step(c, 2 * limit + 3, "borrow", {"memberId": "member-a", "copyId": f"copy-{limit + 1}", "borrowDate": "2026-10-01"}, _refused())]
    s += [_step(c, 2 * limit + 4, "inspectCopy", {"copyId": f"copy-{limit + 1}"}, _ok(_copy(f"copy-{limit + 1}", f"Book {limit + 1}")))]
    s += [_step(c, 2 * limit + 5, "inspectOpenLoans", {"memberId": "member-a"}, _ok({"memberId": "member-a", "count": limit, "loans": [_loan(f"loan-{i}", "member-a", f"copy-{i}", "2026-10-01", due)["loan"] for i in range(1, limit + 1)]}))]
    s += [_step(c, 2 * limit + 6, "assessOverdue", {"assessmentDate": "2026-10-16"}, _ok({"notices": [{"loanId": {"$ref": f"loan-{i}"}, "memberId": "member-a", "copyId": f"copy-{i}", "dueDate": due} for i in range(1, limit + 1)]}))]
    s += [_step(c, 2 * limit + 7, "return", {"loanId": {"$ref": "loan-1"}, "returnDate": "2026-10-16"}, _ok({"outcome": "returned", **_loan("loan-1", "member-a", "copy-1", "2026-10-01", due, "closed", "2026-10-16")}))]
    s += [_step(c, 2 * limit + 8, "borrow", {"memberId": "member-a", "copyId": f"copy-{limit + 1}", "borrowDate": "2026-10-16"}, _ok(_new_loan("loan-new", "member-a", f"copy-{limit + 1}", "2026-10-16", "2026-10-30")), {"loan-new": "result.loan.loanId"})]
    s += [_step(c, 2 * limit + 9, "inspectOpenLoans", {"memberId": "member-a"}, _ok({"memberId": "member-a", "count": limit, "loans": [_loan(f"loan-{i}", "member-a", f"copy-{i}", "2026-10-01", due)["loan"] for i in range(2, limit + 1)] + [_loan("loan-new", "member-a", f"copy-{limit + 1}", "2026-10-16", "2026-10-30")["loan"]]}))]
    s += [_step(c, 2 * limit + 10, "borrow", {"memberId": "member-a", "copyId": f"copy-{limit + 2}", "borrowDate": "2026-10-16"}, _refused())]
    s += [_step(c, 2 * limit + 11, "inspectCopy", {"copyId": f"copy-{limit + 2}"}, _ok(_copy(f"copy-{limit + 2}", f"Book {limit + 2}")))]
    s += [_step(c, 2 * limit + 12, "registerMember", {"memberId": "member-b", "name": "Bea"}, _ok(_member("member-b", "Bea")))]
    s += [_step(c, 2 * limit + 13, "borrow", {"memberId": "member-b", "copyId": f"copy-{limit + 2}", "borrowDate": "2026-10-16"}, _ok(_new_loan("other-member-loan", "member-b", f"copy-{limit + 2}", "2026-10-16", "2026-10-30")), {"other-member-loan": "result.loan.loanId"})]
    s += [_step(c, 2 * limit + 14, "inspectOpenLoans", {"memberId": "member-b"}, _ok({"memberId": "member-b", "count": 1, "loans": [_loan("other-member-loan", "member-b", f"copy-{limit + 2}", "2026-10-16", "2026-10-30")["loan"]]}))]
    cases.append(_case(c, ["E04", "E07", "E10"], s))

    # E05 suspension preserves existing loan and return rights.
    c = "E05-suspension"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 2, "registerCopy", {"copyId": "copy-a", "title": "Book"}, _ok(_copy("copy-a", "Book")))]
    s += [_step(c, 3, "registerCopy", {"copyId": "copy-b", "title": "Book 2"}, _ok(_copy("copy-b", "Book 2")))]
    s += [_step(c, 4, "borrow", {"memberId": "member-a", "copyId": "copy-a", "borrowDate": "2026-10-01"}, _ok(_new_loan("loan-a", "member-a", "copy-a", "2026-10-01", due)), {"loan-a": "result.loan.loanId"})]
    s += [_step(c, 5, "setMemberStatus", {"memberId": "member-a", "status": "suspended"}, _ok(_member("member-a", "Ada", "suspended")))]
    s += [_step(c, 6, "borrow", {"memberId": "member-a", "copyId": "copy-b", "borrowDate": "2026-10-01"}, _refused())]
    s += [_step(c, 7, "inspectLoan", {"loanId": {"$ref": "loan-a"}}, _ok(_loan("loan-a", "member-a", "copy-a", "2026-10-01", due)))]
    s += [_step(c, 8, "inspectCopy", {"copyId": "copy-b"}, _ok(_copy("copy-b", "Book 2")))]
    s += [_step(c, 9, "inspectMember", {"memberId": "member-a"}, _ok(_member("member-a", "Ada", "suspended")))]
    s += [_step(c, 10, "return", {"loanId": {"$ref": "loan-a"}, "returnDate": "2026-10-20"}, _ok({"outcome": "returned", **_loan("loan-a", "member-a", "copy-a", "2026-10-01", due, "closed", "2026-10-20")}))]
    s += [_step(c, 11, "setMemberStatus", {"memberId": "member-a", "status": "active"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 12, "borrow", {"memberId": "member-a", "copyId": "copy-b", "borrowDate": "2026-10-21"}, _ok(_new_loan("loan-b", "member-a", "copy-b", "2026-10-21", "2026-11-04")), {"loan-b": "result.loan.loanId"})]
    cases.append(_case(c, ["E05", "E07"], s))

    # E06 calendar arithmetic including leap day and year boundary.
    c = "E06-calendar"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    for i, date in enumerate(("2026-10-01", "2028-02-20", "2026-12-25"), 1):
        s += [_step(c, i + 1, "registerCopy", {"copyId": f"copy-{i}", "title": f"Book {i}"}, _ok(_copy(f"copy-{i}", f"Book {i}")))]
        due_date = ("2026-10-15", "2028-03-05", "2027-01-08")[i - 1]
        s += [_step(c, i + 4, "borrow", {"memberId": "member-a", "copyId": f"copy-{i}", "borrowDate": date}, _ok(_new_loan(f"loan-{i}", "member-a", f"copy-{i}", date, due_date)), {f"loan-{i}": "result.loan.loanId"})]
    cases.append(_case(c, ["E06"], s))

    # E07 and E08 return boundaries, repeat behavior and old-return isolation.
    c = "E08-old-return"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 2, "registerCopy", {"copyId": "copy-a", "title": "Book"}, _ok(_copy("copy-a", "Book")))]
    s += [_step(c, 3, "borrow", {"memberId": "member-a", "copyId": "copy-a", "borrowDate": "2026-10-01"}, _ok(_new_loan("old", "member-a", "copy-a", "2026-10-01", due)), {"old": "result.loan.loanId"})]
    s += [_step(c, 4, "return", {"loanId": {"$ref": "old"}, "returnDate": "2026-10-02"}, _ok({"outcome": "returned", **_loan("old", "member-a", "copy-a", "2026-10-01", due, "closed", "2026-10-02")}))]
    s += [_step(c, 5, "borrow", {"memberId": "member-a", "copyId": "copy-a", "borrowDate": "2026-10-03"}, _ok(_new_loan("new", "member-a", "copy-a", "2026-10-03", "2026-10-17")), {"new": "result.loan.loanId"})]
    s += [_step(c, 6, "return", {"loanId": {"$ref": "old"}, "returnDate": "2026-10-04"}, _ok({"outcome": "alreadyReturned", **_loan("old", "member-a", "copy-a", "2026-10-01", due, "closed", "2026-10-02")}))]
    s += [_step(c, 7, "return", {"loanId": {"$ref": "old"}, "returnDate": "2026-09-30"}, _refused())]
    s += [_step(c, 8, "inspectLoan", {"loanId": {"$ref": "new"}}, _ok(_loan("new", "member-a", "copy-a", "2026-10-03", "2026-10-17")))]
    s += [_step(c, 9, "inspectCopy", {"copyId": "copy-a"}, _ok(_copy("copy-a", "Book", False)))]
    s += [_step(c, 10, "return", {"loanId": {"$ref": "new"}, "returnDate": "2026-10-03"}, _ok({"outcome": "returned", **_loan("new", "member-a", "copy-a", "2026-10-03", "2026-10-17", "closed", "2026-10-03")}))]
    s += [_step(c, 11, "return", {"loanId": {"$unusedLoan": "missing"}, "returnDate": "2026-10-04"}, _refused())]
    cases.append(_case(c, ["E07", "E08", "E09"], s))

    # E09 reads known/unknown records and the valid empty list.
    c = "E09-reading"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 2, "inspectOpenLoans", {"memberId": "member-a"}, _ok({"memberId": "member-a", "count": 0, "loans": []}))]
    s += [_step(c, 3, "inspectLoan", {"loanId": {"$unusedLoan": "unknown"}}, _refused())]
    s += [_step(c, 4, "inspectCopy", {"copyId": "unknown"}, _refused())]
    s += [_step(c, 5, "inspectMember", {"memberId": "unknown"}, _refused())]
    s += [_step(c, 6, "inspectMember", {"memberId": "member-a"}, _ok(_member("member-a", "Ada")))]
    cases.append(_case(c, ["E09"], s))

    # E10 strict overdue date, suspended member inclusion, returned exclusion, repeat read.
    c = "E10-overdue"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-a", "name": "Ada"}, _ok(_member("member-a", "Ada")))]
    s += [_step(c, 2, "registerMember", {"memberId": "member-b", "name": "Bea"}, _ok(_member("member-b", "Bea")))]
    for i in range(1, 4):
        s += [_step(c, i + 2, "registerCopy", {"copyId": f"copy-{i}", "title": f"Book {i}"}, _ok(_copy(f"copy-{i}", f"Book {i}")))]
    for i, member in ((1, "member-a"), (2, "member-b"), (3, "member-a")):
        s += [_step(c, i + 5, "borrow", {"memberId": member, "copyId": f"copy-{i}", "borrowDate": "2026-10-01"}, _ok(_new_loan(f"loan-{i}", member, f"copy-{i}", "2026-10-01", due)), {f"loan-{i}": "result.loan.loanId"})]
    s += [_step(c, 9, "setMemberStatus", {"memberId": "member-b", "status": "suspended"}, _ok(_member("member-b", "Bea", "suspended")))]
    s += [_step(c, 10, "return", {"loanId": {"$ref": "loan-3"}, "returnDate": "2026-10-20"}, _ok({"outcome": "returned", **_loan("loan-3", "member-a", "copy-3", "2026-10-01", due, "closed", "2026-10-20")}))]
    s += [_step(c, 11, "assessOverdue", {"assessmentDate": "2026-10-15"}, _ok({"notices": []}))]
    notices = [{"loanId": {"$ref": f"loan-{i}"}, "memberId": member, "copyId": f"copy-{i}", "dueDate": due} for i, member in ((1, "member-a"), (2, "member-b"))]
    s += [_step(c, 12, "assessOverdue", {"assessmentDate": "2026-10-16"}, _ok({"notices": notices}))]
    s += [_step(c, 13, "assessOverdue", {"assessmentDate": "2026-10-16"}, _ok({"notices": notices}))]
    s += [_step(c, 14, "inspectOpenLoans", {"memberId": "member-a"}, _ok({"memberId": "member-a", "count": 1, "loans": [_loan("loan-1", "member-a", "copy-1", "2026-10-01", due)["loan"]]}))]
    s += [_step(c, 15, "inspectOpenLoans", {"memberId": "member-b"}, _ok({"memberId": "member-b", "count": 1, "loans": [_loan("loan-2", "member-b", "copy-2", "2026-10-01", due)["loan"]]}))]
    cases.append(_case(c, ["E05", "E10"], s))

    # E11 required-argument validation: each of the ten operations receives
    # each required field once missing and once with a deliberately wrong type.
    c = "E11-invalid-input"; s = []; n = 4
    s.append(_step(c, 1, "registerMember", {"memberId": "valid-id", "name": "Valid Member"}, _ok(_member("valid-id", "Valid Member"))))
    s.append(_step(c, 2, "registerCopy", {"copyId": "valid-copy", "title": "Valid Copy"}, _ok(_copy("valid-copy", "Valid Copy"))))
    s.append(_step(c, 3, "registerCopy", {"copyId": "validation-copy", "title": "Validation Copy"}, _ok(_copy("validation-copy", "Validation Copy"))))
    s.append(_step(c, 4, "borrow", {"memberId": "valid-id", "copyId": "valid-copy", "borrowDate": "2026-10-01"}, _ok(_new_loan("valid-loan", "valid-id", "valid-copy", "2026-10-01", due)), {"valid-loan": "result.loan.loanId"}))
    required = {
        "registerMember": {"memberId": "new-member", "name": "Valid Name"},
        "setMemberStatus": {"memberId": "valid-id", "status": "active"},
        "inspectMember": {"memberId": "valid-id"},
        "registerCopy": {"copyId": "new-copy", "title": "Valid Title"},
        "inspectCopy": {"copyId": "valid-copy"},
        "borrow": {"memberId": "valid-id", "copyId": "validation-copy", "borrowDate": "2026-10-01"},
        "return": {"loanId": {"$ref": "valid-loan"}, "returnDate": "2026-10-01"},
        "inspectLoan": {"loanId": {"$ref": "valid-loan"}},
        "inspectOpenLoans": {"memberId": "valid-id"},
        "assessOverdue": {"assessmentDate": "2026-10-01"},
    }
    for op, args in required.items():
        for key in args:
            n += 1; missing = dict(args); del missing[key]
            s.append(_step(c, n, op, missing, _refused()))
            n += 1; wrong = dict(args); wrong[key] = 17
            s.append(_step(c, n, op, wrong, _refused()))
    for op, args in (("registerMember", {"memberId": "", "name": "Ada"}),
                     ("registerMember", {"memberId": "new-member", "name": "  "}),
                     ("setMemberStatus", {"memberId": "valid-id", "status": "inactive"}),
                     ("registerCopy", {"copyId": "", "title": "Book"}),
                     ("registerCopy", {"copyId": "new-copy", "title": " \t"}),
                     ("borrow", {"memberId": "valid-id", "copyId": "validation-copy", "borrowDate": "2026-02-29"}),
                     ("borrow", {"memberId": "valid-id", "copyId": "validation-copy", "borrowDate": "not-a-date"}),
                     ("return", {"loanId": {"$ref": "valid-loan"}, "returnDate": "2026-09-30"}),
                     ("assessOverdue", {"assessmentDate": "2026-13-01"}),
                     ("inspectMember", {"memberId": "missing-member"}),
                     ("inspectCopy", {"copyId": "missing-copy"}),
                     ("inspectLoan", {"loanId": {"$unusedLoan": "missing-loan"}}),
                     ("inspectOpenLoans", {"memberId": "missing-member"})):
        n += 1; s.append(_step(c, n, op, args, _refused()))
    # Valid requests after every invalid input remain processable.
    n += 1; s.append(_step(c, n, "registerMember", {"memberId": "valid-after-errors", "name": "Still works"}, _ok(_member("valid-after-errors", "Still works"))))
    cases.append(_case(c, ["E11"], s))

    # E12 contract-level normal flow and independent request correlation IDs.
    c = "E12-callable"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "Case", "name": "Exact"}, _ok(_member("Case", "Exact")))]
    s += [_step(c, 2, "registerMember", {"memberId": "case", "name": "Different"}, _ok(_member("case", "Different")))]
    s += [_step(c, 3, "inspectMember", {"memberId": "Case"}, _ok(_member("Case", "Exact")))]
    # Opaque identifiers and display values preserve all supplied characters.
    # Padded/unpadded and composed/decomposed identifiers denote distinct records.
    for index, member_id, name in ((20, " padded-member ", " \tAda \t"),
                                    (21, "padded-member", " Plain Ada "),
                                    (22, "caf\u00e9", "Composed"),
                                    (23, "cafe\u0301", "Decomposed")):
        s += [_step(c, index, "registerMember", {"memberId": member_id, "name": name}, _ok(_member(member_id, name)))]
    for index, copy_id, title in ((24, " padded-copy ", " \tExact title \t"),
                                  (25, "padded-copy", "Plain title")):
        s += [_step(c, index, "registerCopy", {"copyId": copy_id, "title": title}, _ok(_copy(copy_id, title)))]
    s += [_step(c, 26, "borrow", {"memberId": " padded-member ", "copyId": " padded-copy ", "borrowDate": "2026-10-01"}, _ok(_new_loan("padded-loan", " padded-member ", " padded-copy ", "2026-10-01", due)), {"padded-loan": "result.loan.loanId"})]
    s += [_step(c, 27, "inspectMember", {"memberId": " padded-member "}, _ok(_member(" padded-member ", " \tAda \t")))]
    s += [_step(c, 28, "inspectMember", {"memberId": "padded-member"}, _ok(_member("padded-member", " Plain Ada ")))]
    s += [_step(c, 29, "inspectCopy", {"copyId": "padded-copy"}, _ok(_copy("padded-copy", "Plain title")))]
    s += [_step(c, 30, "inspectOpenLoans", {"memberId": "padded-member"}, _ok({"memberId": "padded-member", "count": 0, "loans": []}))]
    s += [{"raw": "{malformed json", "expect": {"id": None, "status": "invalidRequest", "reason": {"$nonempty": True}}}]
    s += [{"raw": "{\"id\":\"bad-op\",\"op\":\"invented\",\"args\":{}}", "expect": {"id": "bad-op", "status": "invalidRequest", "reason": {"$nonempty": True}}}]
    s += [{"raw": "{\"id\":\"same-id\",\"op\":\"registerMember\",\"args\":{\"memberId\":\"transport-a\",\"name\":\"Transport A\"}}", "expect": {"id": "same-id", "status": "ok", "result": _member("transport-a", "Transport A")}}]
    s += [{"raw": "{\"id\":\"same-id\",\"op\":\"inspectMember\",\"args\":{\"memberId\":\"transport-a\"}}", "expect": {"id": "same-id", "status": "ok", "result": _member("transport-a", "Transport A")}}]
    s += [{"raw": "{\"id\":\"same-id\",\"op\":\"registerMember\",\"args\":{\"memberId\":\"transport-a\",\"name\":\"Transport A\"}}", "expect": {"id": "same-id", "status": "refused", "reason": {"$nonempty": True}}}]
    s += [{"raw": "{\"id\":\"unknown-field\",\"op\":\"inspectMember\",\"args\":{\"memberId\":\"Case\",\"extra\":true}}", "expect": {"id": "unknown-field", "status": "invalidRequest", "reason": {"$nonempty": True}}}]
    s += [{"raw": "{\"id\":\"bad-envelope\",\"op\":\"inspectMember\",\"args\":[]}", "expect": {"id": "bad-envelope", "status": "invalidRequest", "reason": {"$nonempty": True}}}]
    s += [{"raw": "{\"id\":\"extra-envelope\",\"op\":\"inspectMember\",\"args\":{\"memberId\":\"Case\"},\"extra\":true}", "expect": {"id": "extra-envelope", "status": "invalidRequest", "reason": {"$nonempty": True}}}]
    s += [_step(c, 10, "inspectMember", {"memberId": "Case"}, _ok(_member("Case", "Exact")))]
    cases.append(_case(c, ["E12"], s))

    # Same fourth-loan sequence and identifiers for both policy versions.
    c = "paired-fourth-loan"; s = []
    s += [_step(c, 1, "registerMember", {"memberId": "member-limit", "name": "Limit Case"}, _ok(_member("member-limit", "Limit Case")))]
    for i in range(1, 7):
        s += [_step(c, i + 1, "registerCopy", {"copyId": f"copy-{i}", "title": f"Copy {i}"}, _ok(_copy(f"copy-{i}", f"Copy {i}")))]
    for i in range(1, 4):
        s += [_step(c, i + 7, "borrow", {"memberId": "member-limit", "copyId": f"copy-{i}", "borrowDate": "2026-10-01"}, _ok(_new_loan(f"loan-{i}", "member-limit", f"copy-{i}", "2026-10-01", due)), {f"loan-{i}": "result.loan.loanId"})]
    if limit == 5:
        fourth = _step(c, 11, "borrow", {"memberId": "member-limit", "copyId": "copy-4", "borrowDate": "2026-10-01"}, _ok(_new_loan("loan-4", "member-limit", "copy-4", "2026-10-01", due)), {"loan-4": "result.loan.loanId"})
        count = 4; available = False
    else:
        fourth = _step(c, 11, "borrow", {"memberId": "member-limit", "copyId": "copy-4", "borrowDate": "2026-10-01"}, _refused())
        count = 3; available = True
    fourth["witness"] = "fourth-admission"
    s.append(fourth)
    count_step = _step(c, 12, "inspectOpenLoans", {"memberId": "member-limit"}, _ok({"memberId": "member-limit", "count": count, "loans": [_loan(f"loan-{i}", "member-limit", f"copy-{i}", "2026-10-01", due)["loan"] for i in range(1, count + 1)]}))
    count_step["witness"] = "open-loan-count"
    s.append(count_step)
    availability_step = _step(c, 13, "inspectCopy", {"copyId": "copy-4"}, _ok(_copy("copy-4", "Copy 4", available)))
    availability_step["witness"] = "copy-4-availability"
    s.append(availability_step)
    for i in range(1, 4):
        s.append(_step(c, 13 + i, "inspectLoan", {"loanId": {"$ref": f"loan-{i}"}}, _ok(_loan(f"loan-{i}", "member-limit", f"copy-{i}", "2026-10-01", due))))
    cases.append(_case(c, ["E03", "E04", "paired-fourth-loan"], s))
    return [_expand_paired_witness(case) if case["id"] == "paired-fourth-loan"
            else _expand_observation_snapshots(case) for case in cases]


def witness_case(profile: str) -> dict[str, Any]:
    """Return the paired discrimination case with its step witness labels."""
    return next(case for case in build_cases(profile) if case["id"] == "paired-fourth-loan")


_SNAPSHOT_DATES = ("2026-10-15", "2026-10-16")
_UNKNOWN_IDS = {
    "member": "__oracle_unknown_member__",
    "copy": "__oracle_unknown_copy__",
    "loan": "__oracle_unknown_loan__",
}


def _snapshot_projection(case_id: str, serial: int, members: dict[str, dict[str, Any]],
                         copies: dict[str, dict[str, Any]], loans: dict[str, dict[str, Any]],
                         inspected_loans: set[str] | None = None) -> list[dict[str, Any]]:
    """Make public reads from explicitly expected records at one point in a case."""
    rows: list[dict[str, Any]] = []

    def add(op: str, args: dict[str, Any], expected: dict[str, Any]) -> None:
        ordinal = serial * 10_000 + len(rows) + 1
        request_id = f"{case_id}-snapshot-{serial:04d}-{len(rows) + 1:04d}"
        row = _step(case_id, ordinal, op, args, expected)
        row["request"]["id"] = request_id
        row["expect"]["id"] = request_id
        row["snapshot"] = True
        rows.append(row)

    open_loans = [loan for loan in loans.values() if loan["status"] == "open"]
    for member_id in sorted(members):
        add("inspectMember", {"memberId": member_id}, _ok({"member": members[member_id]}))
    for copy_id in sorted(copies):
        add("inspectCopy", {"copyId": copy_id}, _ok({"copy": copies[copy_id]}))
    for loan_symbol in sorted(loans):
        if inspected_loans is not None and loan_symbol not in inspected_loans:
            continue
        loan = loans[loan_symbol]
        add("inspectLoan", {"loanId": {"$ref": loan_symbol}}, _ok({"loan": loan}))
    for member_id in sorted(members):
        member_loans = [loan for loan in open_loans if loan["memberId"] == member_id]
        add("inspectOpenLoans", {"memberId": member_id}, _ok({
            "memberId": member_id,
            "count": len(member_loans),
            "loans": member_loans,
        }))
    for date in _SNAPSHOT_DATES:
        overdue = [loan for loan in open_loans if loan["dueDate"] < date]
        notices = [{"loanId": loan["loanId"], "memberId": loan["memberId"],
                    "copyId": loan["copyId"], "dueDate": loan["dueDate"]} for loan in overdue]
        add("assessOverdue", {"assessmentDate": date}, _ok({"notices": notices}))
    # These reserved probes remain unknown after every observed operation.
    add("inspectMember", {"memberId": _UNKNOWN_IDS["member"]}, _refused())
    add("inspectCopy", {"copyId": _UNKNOWN_IDS["copy"]}, _refused())
    add("inspectLoan", {"loanId": {"$unusedLoan": _UNKNOWN_IDS["loan"]}}, _refused())
    add("inspectOpenLoans", {"memberId": _UNKNOWN_IDS["member"]}, _refused())
    return rows


def _request_for_expectation(step: dict[str, Any]) -> dict[str, Any] | None:
    if "request" in step:
        return step["request"]
    if "raw" in step:
        try:
            value = json.loads(step["raw"])
            return value if isinstance(value, dict) else None
        except (TypeError, json.JSONDecodeError):
            return None
    return None


def _symbolic_loan(record: dict[str, Any]) -> tuple[str, dict[str, Any]] | None:
    loan = record.get("loan") if isinstance(record, dict) else None
    if not isinstance(loan, dict):
        return None
    loan = dict(loan)
    loan_id = loan.get("loanId")
    if isinstance(loan_id, dict) and set(loan_id) == {"$capture"}:
        symbol = loan_id["$capture"]
        loan["loanId"] = {"$ref": symbol}
        return symbol, loan
    if isinstance(loan_id, dict) and set(loan_id) == {"$ref"}:
        return loan_id["$ref"], loan
    if isinstance(loan_id, str) and loan_id:
        return loan_id, loan
    return None


def _apply_expected_record(step: dict[str, Any], members: dict[str, dict[str, Any]],
                           copies: dict[str, dict[str, Any]], loans: dict[str, dict[str, Any]]) -> None:
    """Advance only from an explicitly authored successful result record."""
    request = _request_for_expectation(step)
    expected = step.get("expect", {})
    if request is None or expected.get("status") != "ok":
        return
    op = request.get("op")
    result = expected.get("result", {})
    if op in ("registerMember", "setMemberStatus") and isinstance(result.get("member"), dict):
        record = result["member"]
        members[record["memberId"]] = record
    if op == "registerCopy" and isinstance(result.get("copy"), dict):
        record = result["copy"]
        copies[record["copyId"]] = record
    if op in ("borrow", "return"):
        pair = _symbolic_loan(result)
        if pair is not None:
            symbol, record = pair
            loans[symbol] = record
    # Availability follows the explicit expected open/closed loan records.
    for copy_id, record in copies.items():
        available = not any(loan["copyId"] == copy_id and loan["status"] == "open" for loan in loans.values())
        if record.get("available") is not available:
            copies[copy_id] = {**record, "available": available}


def _needs_observation_snapshot(step: dict[str, Any], members: dict[str, dict[str, Any]]) -> bool:
    request = _request_for_expectation(step)
    expected = step.get("expect", {})
    status = expected.get("status")
    if status in ("refused", "invalidRequest"):
        return True
    if request is None or status != "ok":
        return False
    op = request.get("op")
    if op in {"inspectMember", "inspectCopy", "inspectLoan", "inspectOpenLoans", "assessOverdue"}:
        return True
    if op == "setMemberStatus":
        member = expected.get("result", {}).get("member", {})
        prior = members.get(member.get("memberId"))
        return prior is not None and prior.get("status") == member.get("status")
    if op == "return":
        return expected.get("result", {}).get("outcome") == "alreadyReturned"
    return False


def _expand_observation_snapshots(case: dict[str, Any]) -> dict[str, Any]:
    """Bracket observations/refusals with reads derived from expected records."""
    members: dict[str, dict[str, Any]] = {}
    copies: dict[str, dict[str, Any]] = {}
    loans: dict[str, dict[str, Any]] = {}
    expanded: list[dict[str, Any]] = []
    serial = 0
    for step in case["steps"]:
        bracket = _needs_observation_snapshot(step, members)
        if bracket:
            serial += 1
            expanded.extend(_snapshot_projection(case["id"], serial, members, copies, loans))
        expanded.append(step)
        _apply_expected_record(step, members, copies, loans)
        if bracket:
            serial += 1
            expanded.extend(_snapshot_projection(case["id"], serial, members, copies, loans))
    return {**case, "steps": expanded}


def _expand_paired_witness(case: dict[str, Any]) -> dict[str, Any]:
    """Bracket the fourth request with identical public requests in both profiles.

    The optional fourth loan is observed in the full open-loan list and notices.
    Only the original three opaque IDs are used for direct loan inspection.
    """
    members: dict[str, dict[str, Any]] = {}
    copies: dict[str, dict[str, Any]] = {}
    loans: dict[str, dict[str, Any]] = {}
    expanded = []
    for step in case["steps"]:
        fourth = step.get("witness") == "fourth-admission"
        if fourth:
            expanded.extend(_snapshot_projection(case["id"], 1, members, copies, loans))
        expanded.append(step)
        _apply_expected_record(step, members, copies, loans)
        if fourth:
            expanded.extend(_snapshot_projection(case["id"], 2, members, copies, loans,
                                                 {"loan-1", "loan-2", "loan-3"}))
    return {**case, "steps": expanded}


def _resolve(value: Any, captures: dict[str, str]) -> Any:
    if isinstance(value, dict):
        if set(value) == {"$unusedLoan"}:
            candidate = value["$unusedLoan"]
            while candidate in captures.values():
                candidate += "_"
            return candidate
        if set(value) == {"$ref"}:
            symbol = value["$ref"]
            if symbol not in captures:
                raise ValueError(f"uncaptured symbol {symbol!r}")
            return captures[symbol]
        return {k: _resolve(v, captures) for k, v in value.items()}
    if isinstance(value, list):
        return [_resolve(v, captures) for v in value]
    return value


def resolve_request(step: dict[str, Any], captures: dict[str, str]) -> dict[str, Any]:
    """Return the public request with previously captured opaque IDs filled in."""
    if "request" not in step:
        raise ValueError("raw input step has no parsed request")
    return _resolve(step["request"], captures)


def _matches(expected: Any, actual: Any, path: str, errors: list[str], assertions: list[int]) -> None:
    assertions[0] += 1
    if isinstance(expected, dict) and expected == {"$nonempty": True}:
        if not isinstance(actual, str) or not actual.strip():
            errors.append(f"{path}: expected a nonempty string")
        return
    if isinstance(expected, dict) and set(expected) == {"$capture"}:
        if not isinstance(actual, str) or not actual:
            errors.append(f"{path}: expected a nonempty captured identifier")
        return
    if isinstance(expected, dict):
        if not isinstance(actual, dict):
            errors.append(f"{path}: expected object, got {type(actual).__name__}")
            return
        for key, value in expected.items():
            if key not in actual:
                errors.append(f"{path}.{key}: missing required field")
            else:
                _matches(value, actual[key], f"{path}.{key}", errors, assertions)
        return
    if isinstance(expected, list):
        if not isinstance(actual, list):
            errors.append(f"{path}: expected list, got {type(actual).__name__}")
            return
        # Loan and notice collections have contract-defined unordered semantics.
        if all(isinstance(x, dict) and "loanId" in x for x in expected + actual):
            exp_by_id = {x["loanId"]: x for x in expected if isinstance(x.get("loanId"), str)}
            act_by_id = {x["loanId"]: x for x in actual if isinstance(x.get("loanId"), str)}
            if len(exp_by_id) != len(expected) or len(act_by_id) != len(actual):
                errors.append(f"{path}: loan IDs must be strings and unique")
                return
            if set(exp_by_id) != set(act_by_id):
                errors.append(f"{path}: expected IDs {sorted(exp_by_id)}, got {sorted(act_by_id)}")
                return
            for item_id in exp_by_id:
                _matches(exp_by_id[item_id], act_by_id[item_id], f"{path}[{item_id}]", errors, assertions)
            return
        if len(expected) != len(actual):
            errors.append(f"{path}: expected {len(expected)} items, got {len(actual)}")
            return
        for i, (left, right) in enumerate(zip(expected, actual)):
            _matches(left, right, f"{path}[{i}]", errors, assertions)
        return
    # Python considers True == 1; JSON contract types must remain distinct.
    if type(expected) is not type(actual) or expected != actual:
        errors.append(f"{path}: expected {expected!r}, got {actual!r}")


def evaluate_step(step: dict[str, Any], response: Any, captures: dict[str, str]) -> dict[str, Any]:
    """Check one parsed response against independent expectations.

    Return the error list and number of field/shape assertions performed. The
    response may contain extensions; required fields and JSON types are strict.
    Loan-bearing lists are unordered and unique.
    """
    errors: list[str] = []
    assertions = [0]
    if not isinstance(response, dict):
        return {"errors": ["response: expected JSON object"], "assertions": 1}
    try:
        expected = _resolve(step["expect"], captures)
    except (KeyError, ValueError) as exc:
        return {"errors": [f"expectation: {exc}"], "assertions": 0}
    _matches(expected, response, "response", errors, assertions)
    if expected.get("status") in ("refused", "invalidRequest"):
        assertions[0] += 1
        if "result" in response:
            errors.append("response.result: a refusal/invalidRequest must have no result")
    capture = step.get("capture", {})
    for symbol, dotted_path in capture.items():
        value: Any = response
        try:
            for part in dotted_path.split("."):
                value = value[part]
        except (KeyError, TypeError):
            errors.append(f"capture {symbol}: missing {dotted_path}")
            continue
        if not isinstance(value, str) or not value:
            errors.append(f"capture {symbol}: expected nonempty string at {dotted_path}")
        elif symbol in captures:
            errors.append(f"capture {symbol}: symbol was already captured")
        elif value in captures.values():
            errors.append(f"capture {symbol}: loan identifier duplicates an earlier capture")
        else:
            captures[symbol] = value
    return {"errors": errors, "assertions": assertions[0]}
