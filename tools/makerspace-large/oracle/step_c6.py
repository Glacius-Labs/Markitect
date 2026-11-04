"""Change step C6 of the large Makerspace: renewing a tool loan.

One new operation, `renewLoan`, and one new field, `renewals`, on the loan record. A renewal moves `dueOn` to the date 14 days
after its current value and raises `renewals` by 1; nothing is charged, no shift is needed, certification, consent and waiver are
not judged again. The refusals at check 7 come in the order `invalid_state` (returned), `member_suspended` (suspended or paused),
`loan_overdue` (this loan, on the date of the request time), `renewal_limit` (`renewals` is 2). A member may renew only their own
loan (`loanId` joins the ownership list). Written from the requirements document after the change, with every expected value
stated by hand.

What this step retires or moves: nothing is retired. Every loan record an earlier case compares in full now has the member
`renewals` 0, so `transform` adds `"renewals": 0` to each loan record in an expected result. The generated cases (G-renewLoan and
the others) are made from the facts.

How the cases are cut: the operation and its boundaries (`C6-renew`: the example, an overdue loan, from the current `dueOn`, the
limit, after a renewal); month ends, year ends and leap years (`C6-dates`), the latest `dueOn` (`C6-big`); the refusals, each
with the one before it, and the rules not judged again (`C6-gate`); ownership and the order of checks 4 to 7 (`C6-auth`); what
the new `dueOn` does to the other modules (`C6-overdue`); what must not change (`C6-keep`); the field in every result
(`C6-field`).

Not tested because the requirements leave it undecided or the code cannot tell: the words of the refusal reasons.

Dates the cases rely on: 2026-03-02 is a Monday (W, the world's time), so 03-16 is a Monday and 03-30 a Monday; 2028 is a leap
year and 2100 is not. A loan taken on 03-02 is due on 03-16, then on 03-30, then on 04-13. The world's rates: L1 45, L2 60, T1 30
cents a minute. Tiers: ada plus, bob basic, eve pro. Joining the laser course costs 5000 cents (a `course` charge), so a member
certified in a case has 5000 cents unbilled before anything else happens. A tool is named `Tool <id>`.
"""
from __future__ import annotations

from typing import Any

from kit import Case, world

W = "2026-03-02T09:00Z"  # Monday, the world's time

RETIRED: dict[str, str] = {}

LOAN_MEMBERS = {"loanId", "toolId", "memberId", "borrowedAt", "dueOn", "chargedDays"}


def transform(case: dict[str, Any]) -> dict[str, Any]:
    """The loan record has a member more, `renewals`, 0 in every loan a case that knew nothing of renewing makes or lists."""

    def walk(node: Any) -> None:
        if isinstance(node, dict):
            if LOAN_MEMBERS <= set(node) and "renewals" not in node:
                node["renewals"] = 0
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for value in node:
                walk(value)

    for step in case["steps"]:
        walk(step["expect"].get("result"))
    return case


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


# ---- record builders (the members of each record, from the requirements) ---------------------------------------------

def tl(tid: str, fee: int, fam: str = "laser", on: bool = False) -> dict[str, Any]:
    return {"toolId": tid, "name": f"Tool {tid}", "familyId": fam, "lateFeeCentsPerDay": fee, "onLoan": on}


def ln(lid, tid, mem, borrowed, due, returned=None, overdue=False, charged=0, renewals=0) -> dict[str, Any]:
    return {"loanId": lid, "toolId": tid, "memberId": mem, "borrowedAt": borrowed, "dueOn": due, "returnedAt": returned,
            "status": "active" if returned is None else "returned", "overdue": overdue, "chargedDays": charged, "renewals": renewals}


def chg(kind, ref_, qty, unit, disc, amt, at):
    assert (qty * unit * (100 - disc) + 50) // 100 == amt, (kind, ref_, qty, unit, disc, amt)
    return {"kind": kind, "ref": ref_, "quantity": qty, "unitPriceCents": unit, "discountPercent": disc,
            "amountCents": amt, "at": at}


def late(ref_: str, days: int, unit: int, at: str) -> dict[str, Any]:
    """The late-fee charge of a loan for `days` days: no discount, whatever the tier."""
    return chg("lateFee", ref_, days, unit, 0, days * unit, at)


def ntc(nid, mem, ref_, at) -> dict[str, Any]:
    return {"noticeId": nid, "memberId": mem, "kind": "loan", "subject": f"Tool loan {ref_} is overdue", "ref": ref_,
            "at": at, "acknowledgedAt": None}


def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def account(mem, credit=0, open_=0, overdue=0, unbilled=0):
    return {"account": {"memberId": mem, "creditBalanceCents": credit, "openBalanceCents": open_,
                        "overdueBalanceCents": overdue, "unbilledCents": unbilled}}


# ---- steps used by many cases -------------------------------------------------------------------------------------------

def add(c: Case, tid: str, fee: int, family: str = "laser", at: str = W, actor: str = "sam") -> None:
    c.ok("addTool", actor, at, {"toolId": tid, "name": f"Tool {tid}", "familyId": family, "lateFeeCentsPerDay": fee},
         {"tool": tl(tid, fee, family)})


def lend(c: Case, actor: str, at: str, mem: str, tid: str, fee: int, lid: str, due: str, family: str = "laser") -> None:
    c.ok("borrowTool", actor, at, {"memberId": mem, "toolId": tid},
         {"loan": ln(lid, tid, mem, at, due), "tool": tl(tid, fee, family, True)}, new=lid)


def renew(c: Case, actor: str, at: str, lid: str, tid: str, mem: str, borrowed: str, due: str, n: int) -> None:
    """A renewal that is answered ok: the loan is active, not overdue at `at`, nothing charged, `due` its new dueOn, `n` its count."""
    c.ok("renewLoan", actor, at, {"loanId": lid}, {"loan": ln(lid, tid, mem, borrowed, due, renewals=n)})


def back(c: Case, actor: str, at: str, lid: str, tid: str, mem: str, borrowed: str, due: str, fee: int,
         charge: dict | None = None, charged: int = 0, family: str = "laser", renewals: int = 0) -> None:
    c.ok("returnTool", actor, at, {"loanId": lid},
         {"loan": ln(lid, tid, mem, borrowed, due, returned=at, charged=charged, renewals=renewals), "tool": tl(tid, fee, family),
          "charge": charge}, target=lid)


def run(c: Case, actor: str, at: str, charges: list) -> None:
    c.ok("chargeOverdueLoans", actor, at, {}, {"charges": charges})


def certify(c: Case, mem: str, at: str = W, course: str = "laser-101") -> None:
    c.ok("enrollInCourse", "sam", at, {"courseId": course, "memberId": mem})
    c.ok("completeCourse", "sam", at, {"courseId": course, "memberId": mem})


def unbilled(c: Case, mem: str, at: str, cents: int) -> None:
    c.ok("getAccount", "sam", at, {"memberId": mem}, account(mem, unbilled=cents))


def loans_of(c: Case, actor: str, at: str, mem: str, loans: list) -> None:
    c.ok("listMemberLoans", actor, at, {"memberId": mem}, {"memberId": mem, "loans": loans})


def notices_of(c: Case, at: str, mem: str, notices: list) -> None:
    c.ok("listNotices", "sam", at, {"memberId": mem}, {"memberId": mem, "notices": notices})


def book(c: Case, rid: str, mem: str, mach: str, day: int, start: str, end: str) -> None:
    s, e = t(day, start), t(day, end)
    c.ok("createReservation", mem, W, {"memberId": mem, "machineId": mach, "start": s, "end": e},
         {"reservation": res(rid, mem, mach, s, e)}, new=rid)


def stand(c: Case, at: str, mem: str, standing: str) -> None:
    c.ok("setMemberStanding", "sam", at, {"memberId": mem, "standing": standing})


# ---- C6-renew: the operation and its boundaries ---------------------------------------------------------------------------

def renew_1() -> Case:
    c = Case("C6-renew-1", "C6-renew", "Renew loan, example, Overdue loans: a loan with dueOn 2026-03-16 renewed at 2026-03-16T23:59 (on its last day, not overdue) has dueOn "
                                       "2026-03-30 and renewals 1, chargedDays 0, still active; nothing is charged (the account, the notices and the journal are as before) "
                                       "and the tool stays onLoan")
    world(c)
    add(c, "drill", 200)
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    e = t(16, "23:59")
    renew(c, "ada", e, "lon-1", "drill", "ada", a, "2026-03-30", 1)
    loans_of(c, "ada", e, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-30", renewals=1)])
    c.ok("listTools", "bob", e, {}, {"tools": [tl("drill", 200, on=True)]})
    unbilled(c, "ada", e, 5000)
    notices_of(c, e, "ada", [])
    c.ok("listJournalEntries", "amy", e, {"month": "2026-03"}, {"month": "2026-03", "entries": []})
    c.audit_log(e)
    return c


def renew_2() -> Case:
    c = Case("C6-renew-2", "C6-renew", "Renew loan, example, loan_overdue: the same request one minute later, at 2026-03-17T00:00, is refused loan_overdue, as it is later "
                                       "that day, three days later and for staff and admin; the loans stay as they were (dueOn 2026-03-16, renewals 0) and no entry is written")
    world(c)
    add(c, "drill", 200)
    add(c, "saw", 150)
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    lend(c, "amy", a, "ada", "saw", 150, "lon-2", "2026-03-16")
    c.refuse("renewLoan", "ada", t(17, "00:00"), {"loanId": "lon-1"}, "loan_overdue")
    c.refuse("renewLoan", "sam", t(17, "23:59"), {"loanId": "lon-2"}, "loan_overdue")
    c.refuse("renewLoan", "amy", t(20, "09:00"), {"loanId": "lon-1"}, "loan_overdue")
    loans_of(c, "ada", t(20, "09:00"), "ada", [ln("lon-1", "drill", "ada", a, "2026-03-16", overdue=True),
                                              ln("lon-2", "saw", "ada", a, "2026-03-16", overdue=True)])
    c.audit_log(t(20, "09:00"))
    return c


def renew_3() -> Case:
    c = Case("C6-renew-3", "C6-renew", "Renew loan, Loan: the new dueOn is 14 days after the current dueOn, not after the request time: renewed on 03-05 the loan is due 03-30, "
                                       "renewed again on 03-06 it is due 04-13; the third renewal is refused renewal_limit, for the member, staff and admin alike, and the loan "
                                       "stays as it was; returned, the loan keeps renewals 2 and dueOn 04-13, is refused invalid_state, and the next loan of the tool starts "
                                       "with renewals 0 and dueOn 04-03")
    world(c)
    add(c, "drill", 200)
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    renew(c, "ada", t(5, "09:00"), "lon-1", "drill", "ada", a, "2026-03-30", 1)
    renew(c, "ada", t(6, "09:00"), "lon-1", "drill", "ada", a, "2026-04-13", 2)
    r3 = t(7, "09:00")
    c.refuse("renewLoan", "ada", r3, {"loanId": "lon-1"}, "renewal_limit")
    c.refuse("renewLoan", "sam", r3, {"loanId": "lon-1"}, "renewal_limit")
    c.refuse("renewLoan", "amy", r3, {"loanId": "lon-1"}, "renewal_limit")
    loans_of(c, "ada", r3, "ada", [ln("lon-1", "drill", "ada", a, "2026-04-13", renewals=2)])
    r = t(20, "10:00")
    back(c, "sam", r, "lon-1", "drill", "ada", a, "2026-04-13", 200, renewals=2)
    c.refuse("renewLoan", "ada", r, {"loanId": "lon-1"}, "invalid_state")
    c.refuse("renewLoan", "sam", r, {"loanId": "lon-1"}, "invalid_state")
    b = t(20, "11:00")
    lend(c, "amy", b, "ada", "drill", 200, "lon-2", "2026-04-03")
    loans_of(c, "ada", b, "ada", [ln("lon-1", "drill", "ada", a, "2026-04-13", returned=r, renewals=2), ln("lon-2", "drill", "ada", b, "2026-04-03")])
    c.audit_log(b)
    return c


def renew_4() -> Case:
    c = Case("C6-renew-4", "C6-renew", "Renew loan, Overdue loans: a renewed loan is judged by its new dueOn: renewed on 03-16 to 03-30, it can be renewed again at 03-30T23:59 (to 04-13), and "
                                       "at 03-31T00:00 the first loan is refused renewal_limit (not overdue, two renewals) while the other, renewed once to 03-30, is refused "
                                       "loan_overdue; another loan of the member being overdue does not make the first one loan_overdue")
    world(c)
    add(c, "drill", 200)
    add(c, "saw", 150)
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    lend(c, "amy", a, "ada", "saw", 150, "lon-2", "2026-03-16")
    m = t(16, "09:00")
    renew(c, "ada", m, "lon-1", "drill", "ada", a, "2026-03-30", 1)
    renew(c, "ada", m, "lon-2", "saw", "ada", a, "2026-03-30", 1)
    renew(c, "ada", t(30, "23:59"), "lon-1", "drill", "ada", a, "2026-04-13", 2)
    z = t(31, "00:00")
    c.refuse("renewLoan", "ada", z, {"loanId": "lon-1"}, "renewal_limit")
    c.refuse("renewLoan", "ada", z, {"loanId": "lon-2"}, "loan_overdue")
    loans_of(c, "ada", z, "ada", [ln("lon-1", "drill", "ada", a, "2026-04-13", renewals=2), ln("lon-2", "saw", "ada", a, "2026-03-30", overdue=True, renewals=1)])
    c.audit_log(z)
    return c


# ---- C6-dates, C6-big: dates ----------------------------------------------------------------------------------------------

def dates_1() -> Case:
    c = Case("C6-dates-1", "C6-dates", "Renew loan, Loan, Time: 14 days after the current dueOn over a non-leap February (2026-02-20 gives 2026-03-06; 2027-02-15 gives 2027-03-01), a year end "
                                       "(2026-12-25 gives 2027-01-08, then 2027-01-22) and a leap February (2028-02-20 gives 2028-03-05, then 2028-03-19)")
    s = "2026-02-06T09:00Z"
    world(c, s)
    add(c, "drill", 200, at=s)
    # a non-leap February
    renew_at = "2026-02-10T09:00Z"
    lend(c, "amy", s, "ada", "drill", 200, "lon-1", "2026-02-20")
    renew(c, "ada", renew_at, "lon-1", "drill", "ada", s, "2026-03-06", 1)
    r1 = "2026-02-12T09:00Z"
    back(c, "sam", r1, "lon-1", "drill", "ada", s, "2026-03-06", 200, renewals=1)
    # a year end
    b2 = "2026-12-11T09:00Z"
    lend(c, "amy", b2, "ada", "drill", 200, "lon-2", "2026-12-25")
    renew(c, "ada", "2026-12-12T09:00Z", "lon-2", "drill", "ada", b2, "2027-01-08", 1)
    renew(c, "ada", "2026-12-13T09:00Z", "lon-2", "drill", "ada", b2, "2027-01-22", 2)
    r2 = "2026-12-14T09:00Z"
    back(c, "sam", r2, "lon-2", "drill", "ada", b2, "2027-01-22", 200, renewals=2)
    # the second February of the case, not a leap year
    certify(c, "ada", "2027-02-01T09:00Z")  # the certification of 2026-02-06 has expired
    b3 = "2027-02-01T10:00Z"
    lend(c, "amy", b3, "ada", "drill", 200, "lon-3", "2027-02-15")
    renew(c, "ada", "2027-02-02T09:00Z", "lon-3", "drill", "ada", b3, "2027-03-01", 1)
    r3 = "2027-02-03T09:00Z"
    back(c, "sam", r3, "lon-3", "drill", "ada", b3, "2027-03-01", 200, renewals=1)
    # a leap February
    certify(c, "ada", "2028-02-06T09:00Z")
    b4 = "2028-02-06T10:00Z"
    lend(c, "amy", b4, "ada", "drill", 200, "lon-4", "2028-02-20")
    renew(c, "ada", "2028-02-10T09:00Z", "lon-4", "drill", "ada", b4, "2028-03-05", 1)
    end = "2028-02-11T09:00Z"
    renew(c, "ada", end, "lon-4", "drill", "ada", b4, "2028-03-19", 2)
    loans_of(c, "ada", end, "ada", [ln("lon-1", "drill", "ada", s, "2026-03-06", returned=r1, renewals=1),
                                    ln("lon-2", "drill", "ada", b2, "2027-01-22", returned=r2, renewals=2),
                                    ln("lon-3", "drill", "ada", b3, "2027-03-01", returned=r3, renewals=1),
                                    ln("lon-4", "drill", "ada", b4, "2028-03-19", renewals=2)])
    c.audit_log(end)
    return c


def big_1() -> Case:
    c = Case("C6-big-1", "C6-big", "Computed times, Renew loan: a loan taken at 2099-12-31T23:59 is due 2100-01-14; two renewals at once give 2100-01-28 and 2100-02-11, the latest dueOn; "
                                   "a third is refused renewal_limit; the loan is not overdue, so a charging run finds nothing")
    e = "2099-12-31T23:59Z"
    world(c, e)
    add(c, "drill", 200, at=e)
    lend(c, "amy", e, "ada", "drill", 200, "lon-1", "2100-01-14")
    renew(c, "ada", e, "lon-1", "drill", "ada", e, "2100-01-28", 1)
    renew(c, "ada", e, "lon-1", "drill", "ada", e, "2100-02-11", 2)
    c.refuse("renewLoan", "ada", e, {"loanId": "lon-1"}, "renewal_limit")
    loans_of(c, "ada", e, "ada", [ln("lon-1", "drill", "ada", e, "2100-02-11", renewals=2)])
    run(c, "sam", e, [])
    c.audit_log(e)
    return c


# ---- C6-gate: the refusals, and what is not judged again ------------------------------------------------------------------

def gate_order_1() -> Case:
    c = Case("C6-gate-order-1", "C6-gate", "Renew loan, refusals in order: a returned loan of a suspended member is invalid_state (not member_suspended); a suspended member's loan "
                                           "that is overdue is member_suspended (not loan_overdue), and one with two renewals too (not renewal_limit); once good, an overdue "
                                           "loan is loan_overdue; a loan with two renewals is renewal_limit while not overdue and loan_overdue once it is overdue")
    world(c)
    certify(c, "bob")
    certify(c, "eve")
    add(c, "a", 200)
    add(c, "b", 150)
    add(c, "c", 100)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "bob", "b", 150, "lon-2", "2026-03-16")
    lend(c, "amy", x, "eve", "c", 100, "lon-3", "2026-03-16")
    renew(c, "sam", t(3, "09:00"), "lon-3", "c", "eve", x, "2026-03-30", 1)
    renew(c, "sam", t(4, "09:00"), "lon-3", "c", "eve", x, "2026-04-13", 2)
    y = t(5, "09:00")
    stand(c, y, "bob", "suspended")
    stand(c, y, "eve", "suspended")
    c.refuse("renewLoan", "sam", y, {"loanId": "lon-2"}, "member_suspended")
    c.refuse("renewLoan", "sam", y, {"loanId": "lon-3"}, "member_suspended")
    z = t(6, "09:00")
    back(c, "sam", z, "lon-2", "b", "bob", x, "2026-03-16", 150)
    c.refuse("renewLoan", "sam", z, {"loanId": "lon-2"}, "invalid_state")
    stand(c, z, "eve", "good")
    c.refuse("renewLoan", "sam", z, {"loanId": "lon-3"}, "renewal_limit")
    v = t(17, "09:00")
    stand(c, v, "ada", "suspended")
    c.refuse("renewLoan", "sam", v, {"loanId": "lon-1"}, "member_suspended")
    stand(c, v, "ada", "good")
    c.refuse("renewLoan", "sam", v, {"loanId": "lon-1"}, "loan_overdue")
    c.refuse("renewLoan", "sam", v, {"loanId": "lon-3"}, "renewal_limit")
    w = t(14, "09:00", 4)
    c.refuse("renewLoan", "sam", w, {"loanId": "lon-3"}, "loan_overdue")
    c.refuse("renewLoan", "sam", w, {"loanId": "lon-2"}, "invalid_state")
    c.audit_log(w)
    return c


def gate_paused_1() -> Case:
    c = Case("C6-gate-paused-1", "C6-gate", "Renew loan, Membership: a paused member is refused member_suspended like a suspended one, for the member and for staff, though the loan "
                                            "is not overdue and has no renewal; a member in good standing is not affected; reinstated, the paused and the suspended member renew")
    world(c)
    certify(c, "bob")
    certify(c, "eve")
    add(c, "a", 200)
    add(c, "b", 150)
    add(c, "c", 100)
    x = t(2, "10:00")
    lend(c, "amy", x, "bob", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "eve", "b", 150, "lon-2", "2026-03-16")
    lend(c, "amy", x, "ada", "c", 100, "lon-3", "2026-03-16")
    y = t(5, "09:00")
    stand(c, y, "bob", "paused")
    stand(c, y, "eve", "suspended")
    c.refuse("renewLoan", "bob", y, {"loanId": "lon-1"}, "member_suspended")
    c.refuse("renewLoan", "sam", y, {"loanId": "lon-1"}, "member_suspended")
    c.refuse("renewLoan", "eve", y, {"loanId": "lon-2"}, "member_suspended")
    renew(c, "ada", y, "lon-3", "c", "ada", x, "2026-03-30", 1)
    z = t(6, "09:00")
    stand(c, z, "bob", "good")
    renew(c, "bob", z, "lon-1", "a", "bob", x, "2026-03-30", 1)
    c.refuse("renewLoan", "eve", z, {"loanId": "lon-2"}, "member_suspended")
    stand(c, z, "eve", "good")
    renew(c, "eve", z, "lon-2", "b", "eve", x, "2026-03-30", 1)
    c.audit_log(z)
    return c


def gate_loans_1() -> Case:
    c = Case("C6-gate-loans-1", "C6-gate", "Renew loan, loan_overdue, Overdue loans: only this loan counts: with lon-1 overdue the member renews lon-2 (03-17, then 03-25 after "
                                           "the original dueOn 03-24 has passed), while lon-1 is refused loan_overdue and the member is still refused another tool; a charging "
                                           "run on 03-25 charges lon-1 alone (9 days) and posts one notice, as the renewed lon-2 is not overdue; the third renewal is renewal_limit")
    world(c)
    add(c, "a", 200)
    add(c, "b", 150)
    add(c, "c", 100)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "a", 200, "lon-1", "2026-03-16")
    y = t(10, "10:00")
    lend(c, "amy", y, "ada", "b", 150, "lon-2", "2026-03-24")
    d = t(17, "09:00")
    loans_of(c, "ada", d, "ada", [ln("lon-1", "a", "ada", x, "2026-03-16", overdue=True), ln("lon-2", "b", "ada", y, "2026-03-24")])
    c.refuse("borrowTool", "amy", d, {"memberId": "ada", "toolId": "c"}, "loan_overdue")
    c.refuse("renewLoan", "ada", d, {"loanId": "lon-1"}, "loan_overdue")
    renew(c, "ada", d, "lon-2", "b", "ada", y, "2026-04-07", 1)
    e = t(25, "09:00")
    run(c, "sam", e, [late("lon-1", 9, 200, e)])
    notices_of(c, e, "ada", [ntc("ntc-1", "ada", "lon-1", e)])
    renew(c, "ada", e, "lon-2", "b", "ada", y, "2026-04-21", 2)
    c.refuse("renewLoan", "ada", e, {"loanId": "lon-2"}, "renewal_limit")
    loans_of(c, "ada", e, "ada", [ln("lon-1", "a", "ada", x, "2026-03-16", overdue=True, charged=9), ln("lon-2", "b", "ada", y, "2026-04-21", renewals=2)])
    unbilled(c, "ada", e, 6800)
    c.audit_log(e)
    return c


def gate_rules_1() -> Case:
    c = Case("C6-gate-rules-1", "C6-gate", "Renew loan, certification, consent, waiver and shift are not judged again: with the certification revoked, a new waiver unsigned, the minor's "
                                           "consent withdrawn and staff without a shift, the member, staff and admin renew (while borrowing another tool is still refused "
                                           "not_certified, guardian_consent_missing and approver_not_on_shift)")
    world(c)
    add(c, "drill", 200)
    add(c, "saw", 150)
    add(c, "rasp", 100)
    c.ok("declareMinor", "sam", W, {"memberId": "bob", "birthDate": "2012-05-01", "guardianId": "ada"})
    c.ok("grantConsent", "ada", W, {"guardianId": "ada", "memberId": "bob", "familyId": "laser"})
    certify(c, "bob")
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "drill", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "bob", "saw", 150, "lon-2", "2026-03-16")
    y = t(3, "09:00")
    c.ok("withdrawConsent", "ada", y, {"guardianId": "ada", "memberId": "bob", "familyId": "laser"})
    c.ok("revokeCertification", "sam", y, {"memberId": "ada", "familyId": "laser"})
    c.ok("revokeCertification", "sam", y, {"memberId": "bob", "familyId": "laser"})
    c.ok("publishWaiver", "amy", y, {"familyId": "laser", "text": "Laser waiver"})
    c.refuse("borrowTool", "amy", y, {"memberId": "ada", "toolId": "rasp"}, "not_certified")
    c.refuse("borrowTool", "amy", y, {"memberId": "bob", "toolId": "rasp"}, "guardian_consent_missing")
    c.refuse("borrowTool", "sam", y, {"memberId": "ada", "toolId": "rasp"}, "approver_not_on_shift")
    z = t(4, "09:00")
    renew(c, "ada", z, "lon-1", "drill", "ada", x, "2026-03-30", 1)
    renew(c, "bob", z, "lon-2", "saw", "bob", x, "2026-03-30", 1)
    renew(c, "sam", z, "lon-1", "drill", "ada", x, "2026-04-13", 2)
    renew(c, "amy", z, "lon-2", "saw", "bob", x, "2026-04-13", 2)
    c.audit_log(z)
    return c


# ---- C6-auth: ownership and the order of checks 4 to 7 ----------------------------------------------------------------------

def auth_1() -> Case:
    c = Case("C6-auth-1", "C6-auth", "Renew loan, Authorization, Order of checks: a member renews only a loan of their own (loanId joins the ownership list), staff and admin any; another "
                                     "member's loan is forbidden at check 6 before invalid_state, renewal_limit and loan_overdue; an unknown loan is not_found at check 5 for "
                                     "a member too; a bad identifier is invalid_argument; returnTool stays forbidden for a member at check 4, own loan or not; the owner meets "
                                     "invalid_state, loan_overdue and renewal_limit at check 7")
    world(c)
    certify(c, "bob")
    for tid, fee in (("a", 200), ("b", 150), ("c", 100), ("d", 250)):
        add(c, tid, fee)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "ada", "b", 150, "lon-2", "2026-03-16")
    lend(c, "amy", x, "ada", "c", 100, "lon-3", "2026-03-16")
    b = t(3, "09:00")
    lend(c, "amy", b, "bob", "d", 250, "lon-4", "2026-03-17")
    renew(c, "ada", t(3, "10:00"), "lon-3", "c", "ada", x, "2026-03-30", 1)
    renew(c, "ada", t(4, "09:00"), "lon-3", "c", "ada", x, "2026-04-13", 2)
    r = t(5, "09:00")
    back(c, "sam", r, "lon-1", "a", "ada", x, "2026-03-16", 200)
    f = t(5, "10:00")
    c.refuse("renewLoan", "bob", f, {"loanId": "lon-1"}, "forbidden")  # returned
    c.refuse("renewLoan", "bob", f, {"loanId": "lon-3"}, "forbidden")  # two renewals
    c.refuse("renewLoan", "bob", f, {"loanId": "lon-2"}, "forbidden")  # would be allowed
    c.refuse("renewLoan", "ada", f, {"loanId": "lon-4"}, "forbidden")
    c.refuse("renewLoan", "bob", f, {"loanId": "lon-9"}, "not_found")
    c.refuse("renewLoan", "sam", f, {"loanId": "lon-9"}, "not_found")
    c.refuse("renewLoan", "sam", f, {"loanId": "bad id!"}, "invalid_argument")
    c.refuse("returnTool", "ada", f, {"loanId": "lon-2"}, "forbidden")
    c.refuse("returnTool", "bob", f, {"loanId": "lon-4"}, "forbidden")
    renew(c, "amy", t(6, "09:00"), "lon-4", "d", "bob", b, "2026-03-31", 1)
    renew(c, "sam", t(7, "09:00"), "lon-4", "d", "bob", b, "2026-04-14", 2)
    o = t(17, "09:00")
    c.refuse("renewLoan", "bob", o, {"loanId": "lon-2"}, "forbidden")
    c.refuse("renewLoan", "ada", o, {"loanId": "lon-2"}, "loan_overdue")
    c.refuse("renewLoan", "ada", o, {"loanId": "lon-1"}, "invalid_state")
    c.refuse("renewLoan", "ada", o, {"loanId": "lon-3"}, "renewal_limit")
    c.audit_log(o)
    return c


# ---- C6-overdue: what the new dueOn does in other modules ---------------------------------------------------------------------

def overdue_1() -> Case:
    c = Case("C6-overdue-1", "C6-overdue", "Overdue loans, Check in, Borrow tool, Charge overdue loans, Return tool: a loan renewed on its last day to 03-30 is not overdue on 03-17: ada "
                                           "checks in, borrows another tool and a charging run finds nothing; on 03-30 it is still not overdue; on 03-31 the run charges 1 day "
                                           "(from the new dueOn) and posts the notice, check-in and borrowing are refused loan_overdue; the return on 04-01 charges the 1 day left "
                                           "and keeps renewals 1")
    world(c)
    add(c, "drill", 200)
    add(c, "saw", 150)
    add(c, "rasp", 100)
    book(c, "res-1", "ada", "L1", 17, "10:00", "11:00")
    book(c, "res-2", "ada", "L1", 31, "10:00", "11:00")
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    renew(c, "ada", t(16, "23:59"), "lon-1", "drill", "ada", a, "2026-03-30", 1)
    d1 = t(17, "10:00")
    run(c, "sam", d1, [])
    notices_of(c, d1, "ada", [])
    c.ok("checkIn", "ada", d1, {"reservationId": "res-1"}, {"reservation": res("res-1", "ada", "L1", t(17, "10:00"), t(17, "11:00"), "checkedIn", cin=d1, rate=45)})
    lend(c, "amy", d1, "ada", "saw", 150, "lon-2", "2026-03-31")
    loans_of(c, "ada", d1, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-30", renewals=1), ln("lon-2", "saw", "ada", d1, "2026-03-31")])
    run(c, "sam", t(30, "23:59"), [])
    d2 = t(31, "08:00")
    run(c, "sam", d2, [late("lon-1", 1, 200, d2)])
    notices_of(c, d2, "ada", [ntc("ntc-1", "ada", "lon-1", d2)])
    d3 = t(31, "10:00")
    c.refuse("checkIn", "ada", d3, {"reservationId": "res-2"}, "loan_overdue")
    c.refuse("borrowTool", "amy", d3, {"memberId": "ada", "toolId": "rasp"}, "loan_overdue")
    r = t(1, "08:00", 4)
    back(c, "sam", r, "lon-1", "drill", "ada", a, "2026-03-30", 200, charge=late("lon-1", 1, 200, r), charged=2, renewals=1)
    loans_of(c, "ada", r, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-30", returned=r, charged=2, renewals=1),
                                  ln("lon-2", "saw", "ada", d1, "2026-03-31", overdue=True)])
    unbilled(c, "ada", r, 5400)
    c.audit_log(r)
    return c


def overdue_2() -> Case:
    c = Case("C6-overdue-2", "C6-overdue", "Charge overdue loans, Late-fee charge, Posting notices: of two loans due 03-16, the one renewed to 03-30 is not charged on 03-17 or 03-30, while "
                                           "the other is (1 day, then 13 days); on 03-31 both are charged 1 day in loan order and the renewed loan's notice is posted then, "
                                           "once; chargedDays follows each loan's own days")
    world(c)
    add(c, "drill", 200)
    add(c, "saw", 150)
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    lend(c, "amy", a, "ada", "saw", 150, "lon-2", "2026-03-16")
    renew(c, "ada", t(16, "09:00"), "lon-1", "drill", "ada", a, "2026-03-30", 1)
    d1 = t(17, "09:00")
    run(c, "sam", d1, [late("lon-2", 1, 150, d1)])
    notices_of(c, d1, "ada", [ntc("ntc-1", "ada", "lon-2", d1)])
    d2 = t(30, "09:00")
    run(c, "sam", d2, [late("lon-2", 13, 150, d2)])
    notices_of(c, d2, "ada", [ntc("ntc-1", "ada", "lon-2", d1)])
    d3 = t(31, "09:00")
    run(c, "sam", d3, [late("lon-1", 1, 200, d3), late("lon-2", 1, 150, d3)])
    notices_of(c, d3, "ada", [ntc("ntc-1", "ada", "lon-2", d1), ntc("ntc-2", "ada", "lon-1", d3)])
    loans_of(c, "ada", d3, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-30", overdue=True, charged=1, renewals=1),
                                   ln("lon-2", "saw", "ada", a, "2026-03-16", overdue=True, charged=15)])
    unbilled(c, "ada", d3, 7450)
    c.audit_log(d3)
    return c


# ---- C6-keep, C6-field: what must not change, and the field ------------------------------------------------------------------

def keep_1() -> Case:
    c = Case("C6-keep-1", "C6-keep", "Renew loan, Tool, Borrow tool: a renewal is no new loan: the tool stays onLoan and cannot be borrowed by another member, the member still has 3 active "
                                     "loans and is refused loan_limit, the next loan takes the next number (lon-4) with renewals 0, the other loans and the list order are untouched, "
                                     "and nothing is charged or posted")
    world(c)
    certify(c, "eve")
    for tid, fee in (("a", 200), ("b", 150), ("c", 100), ("d", 250)):
        add(c, tid, fee)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "ada", "b", 150, "lon-2", "2026-03-16")
    lend(c, "amy", x, "ada", "c", 100, "lon-3", "2026-03-16")
    y = t(5, "09:00")
    renew(c, "ada", y, "lon-2", "b", "ada", x, "2026-03-30", 1)
    c.ok("listTools", "bob", y, {}, {"tools": [tl("a", 200, on=True), tl("b", 150, on=True), tl("c", 100, on=True), tl("d", 250)]})
    c.refuse("borrowTool", "amy", y, {"memberId": "ada", "toolId": "d"}, "loan_limit")
    c.refuse("borrowTool", "amy", y, {"memberId": "eve", "toolId": "b"}, "tool_on_loan")
    z = t(6, "09:00")
    back(c, "sam", z, "lon-1", "a", "ada", x, "2026-03-16", 200)
    lend(c, "amy", z, "ada", "d", 250, "lon-4", "2026-03-20")
    loans_of(c, "ada", z, "ada", [ln("lon-1", "a", "ada", x, "2026-03-16", returned=z), ln("lon-2", "b", "ada", x, "2026-03-30", renewals=1),
                                  ln("lon-3", "c", "ada", x, "2026-03-16"), ln("lon-4", "d", "ada", z, "2026-03-20")])
    c.ok("listTools", "bob", z, {}, {"tools": [tl("a", 200), tl("b", 150, on=True), tl("c", 100, on=True), tl("d", 250, on=True)]})
    notices_of(c, z, "ada", [])
    unbilled(c, "ada", z, 5000)
    c.audit_log(z)
    return c


def field_1() -> Case:
    c = Case("C6-field-1", "C6-field", "Loan, renewals: the field is in the loan of every result that carries one: borrowTool (0), listMemberLoans (0, then 1 after a renewal, also for staff), "
                                       "renewLoan (1) and returnTool (1), and in the list again once the loan is returned")
    world(c)
    add(c, "drill", 200)
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    loans_of(c, "sam", a, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-16")])
    renew(c, "sam", t(3, "09:00"), "lon-1", "drill", "ada", a, "2026-03-30", 1)
    loans_of(c, "sam", t(3, "09:00"), "ada", [ln("lon-1", "drill", "ada", a, "2026-03-30", renewals=1)])
    r = t(4, "09:00")
    back(c, "sam", r, "lon-1", "drill", "ada", a, "2026-03-30", 200, renewals=1)
    loans_of(c, "ada", r, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-30", returned=r, renewals=1)])
    c.audit_log(r)
    return c


def build_cases() -> list[dict[str, Any]]:
    makers = [renew_1, renew_2, renew_3, renew_4, dates_1, big_1, gate_order_1, gate_paused_1, gate_loans_1, gate_rules_1, auth_1,
              overdue_1, overdue_2, keep_1, field_1]
    return [maker().build() for maker in makers]
