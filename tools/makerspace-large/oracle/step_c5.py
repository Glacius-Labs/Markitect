"""Change step C5 of the large Makerspace: a tool library.

A new module `loans`: tools (`addTool`, `listTools`) and the loans of them (`borrowTool`, `returnTool`, `listMemberLoans`,
`chargeOverdueLoans`). A loan is due 14 days after the date of `borrowedAt` (`dueOn`) and is overdue on a date after
`dueOn`; each day overdue costs the tool's fee as a `lateFee` charge for the member, made by a charging run or by the
return, never twice for one day (`chargedDays`). A member with an overdue loan cannot borrow another tool and cannot
check in. Written from the requirements document after the change, with every expected value stated by hand.

What this step retires or moves: nothing. No earlier case sends one of the six new operations or the notice kind `loan`,
a member without a loan is never refused `loan_overdue`, and no case reads a table that gains a row (no new ledger account,
charge kind or counter that an earlier case lists). The generated cases (G-addTool and so on) are made from the facts.

How the cases are cut: the tool and loan records, the list order and the audit entries (`C5-tools`, `C5-borrow`,
`C5-list`); the dates (`C5-due`, `C5-overdue`, `C5-big`); the refusals of `borrowTool`, each gate on its own and then in
order (`C5-gate`, `C5-order`, `C5-limit`); the run and the return (`C5-run`, `C5-return`); the effects on other modules
(`C5-checkin`, `C5-money`).

Not tested because the requirements leave it undecided or the code cannot tell: that `memberId` is judged before `toolId`
at check 5 (both answer `not_found`), and the words of the refusal reasons.

Dates the cases rely on: 2026-03-02 is a Monday (W, the world's time), so 03-16 is a Monday, 03-17 and 03-18 Tuesday and
Wednesday; 2028 is a leap year and 2100 is not. A loan taken on 03-02 is due on 03-16. The world's rates: L1 45, L2 60, T1 30
cents a minute. Tiers: ada plus (10 %), bob basic, eve pro. Joining the laser course costs 5000 cents (a `course` charge), so
a member certified in a case has 5000 cents unbilled before anything else happens. A tool is named `Tool <id>`.
"""
from __future__ import annotations

from typing import Any

from kit import Case, world

W = "2026-03-02T09:00Z"  # Monday, the world's time

RETIRED: dict[str, str] = {}

KINDS = ["membership", "usage", "material", "course", "deposit", "pass", "locker", "guest", "lateFee", "event", "noShow"]


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


# ---- record builders (the members of each record, from the requirements) ---------------------------------------------

def tl(tid: str, fee: int, fam: str = "laser", on: bool = False) -> dict[str, Any]:
    return {"toolId": tid, "name": f"Tool {tid}", "familyId": fam, "lateFeeCentsPerDay": fee, "onLoan": on}


def ln(lid, tid, mem, borrowed, due, returned=None, overdue=False, charged=0) -> dict[str, Any]:
    return {"loanId": lid, "toolId": tid, "memberId": mem, "borrowedAt": borrowed, "dueOn": due, "returnedAt": returned,
            "status": "active" if returned is None else "returned", "overdue": overdue, "chargedDays": charged}


def chg(kind, ref_, qty, unit, disc, amt, at):
    assert (qty * unit * (100 - disc) + 50) // 100 == amt, (kind, ref_, qty, unit, disc, amt)
    return {"kind": kind, "ref": ref_, "quantity": qty, "unitPriceCents": unit, "discountPercent": disc,
            "amountCents": amt, "at": at}


def late(ref_: str, days: int, unit: int, at: str) -> dict[str, Any]:
    """The late-fee charge of a loan for `days` days: no discount, whatever the tier."""
    return chg("lateFee", ref_, days, unit, 0, days * unit, at)


def ntc(nid, mem, ref_, at, kind="loan", subject=None) -> dict[str, Any]:
    return {"noticeId": nid, "memberId": mem, "kind": kind, "subject": subject or f"Tool loan {ref_} is overdue", "ref": ref_,
            "at": at, "acknowledgedAt": None}


def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def account(mem, credit=0, open_=0, overdue=0, unbilled=0):
    return {"account": {"memberId": mem, "creditBalanceCents": credit, "openBalanceCents": open_,
                        "overdueBalanceCents": overdue, "unbilledCents": unbilled}}


def invoice(iid, mem, month, issued, due, lines, overdue=False):
    total = sum(line["amountCents"] for line in lines)
    return {"invoiceId": iid, "memberId": mem, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": 0, "paidCents": 0, "balanceCents": total, "status": "open",
            "overdue": overdue}


def je(n, at, credit, amount, memo):
    return {"entryId": f"jrn-{n}", "at": at, "debitAccount": "receivable", "creditAccount": credit, "amountCents": amount, "memo": memo}


def kinds(invoiced: dict, total: int) -> list:
    assert set(invoiced) <= set(KINDS)
    rows = [{"kind": k, "invoicedCents": invoiced.get(k, 0), "refundedCents": 0} for k in KINDS]
    assert sum(r["invoicedCents"] for r in rows) == total
    return rows


def fam(fid, cert, cons, signed, ppe):
    return {"familyId": fid, "certified": cert, "consent": cons, "waiverSigned": signed, "ppeHeld": ppe}


# ---- steps used by many cases -------------------------------------------------------------------------------------------

def add(c: Case, tid: str, fee: int, family: str = "laser", at: str = W, actor: str = "sam") -> None:
    c.ok("addTool", actor, at, {"toolId": tid, "name": f"Tool {tid}", "familyId": family, "lateFeeCentsPerDay": fee},
         {"tool": tl(tid, fee, family)})


def lend(c: Case, actor: str, at: str, mem: str, tid: str, fee: int, lid: str, due: str, family: str = "laser") -> None:
    c.ok("borrowTool", actor, at, {"memberId": mem, "toolId": tid},
         {"loan": ln(lid, tid, mem, at, due), "tool": tl(tid, fee, family, True)}, new=lid)


def back(c: Case, actor: str, at: str, lid: str, tid: str, mem: str, borrowed: str, due: str, fee: int,
         charge: dict | None = None, charged: int = 0, family: str = "laser") -> None:
    c.ok("returnTool", actor, at, {"loanId": lid},
         {"loan": ln(lid, tid, mem, borrowed, due, returned=at, charged=charged), "tool": tl(tid, fee, family), "charge": charge}, target=lid)


def run(c: Case, actor: str, at: str, charges: list) -> None:
    c.ok("chargeOverdueLoans", actor, at, {}, {"charges": charges})


def certify(c: Case, mem: str, at: str = W, course: str = "laser-101") -> None:
    c.ok("enrollInCourse", "sam", at, {"courseId": course, "memberId": mem})
    c.ok("completeCourse", "sam", at, {"courseId": course, "memberId": mem})


def unbilled(c: Case, mem: str, at: str, cents: int, open_: int = 0, overdue: int = 0) -> None:
    c.ok("getAccount", "sam", at, {"memberId": mem}, account(mem, open_=open_, overdue=overdue, unbilled=cents))


def loans_of(c: Case, actor: str, at: str, mem: str, loans: list) -> None:
    c.ok("listMemberLoans", actor, at, {"memberId": mem}, {"memberId": mem, "loans": loans})


def notices_of(c: Case, at: str, mem: str, notices: list) -> None:
    c.ok("listNotices", "sam", at, {"memberId": mem}, {"memberId": mem, "notices": notices})


def book(c: Case, rid: str, mem: str, mach: str, day: int, start: str, end: str) -> None:
    s, e = t(day, start), t(day, end)
    c.ok("createReservation", mem, W, {"memberId": mem, "machineId": mach, "start": s, "end": e},
         {"reservation": res(rid, mem, mach, s, e)}, new=rid)


# ---- C5-tools: the tools ------------------------------------------------------------------------------------------------

def tools_1() -> Case:
    c = Case("C5-tools-1", "C5-tools", "Add tool, List tools, List order, Initial state: no tool exists at the start; staff and admin add tools, a member may not; "
                                       "the tools come by toolId in ASCII order (X1, x-1, x.1, x1, x10, x_1), not in the order added; an existing toolId is refused "
                                       "already_exists and the first tool stays as it was; an unknown family is not_found; a member lists tools; the audit target is toolId")
    world(c)
    c.ok("listTools", "bob", W, {}, {"tools": []})
    for actor, tid, fee, family in [("sam", "x1", 200, "laser"), ("sam", "x_1", 150, "lathe"), ("amy", "X1", 20, "laser"),
                                    ("sue", "x10", 100000, "lathe"), ("amy", "x-1", 1, "laser"), ("sam", "x.1", 75, "laser")]:
        add(c, tid, fee, family, actor=actor)
    c.refuse("addTool", "sam", W, {"toolId": "x1", "name": "Other", "familyId": "lathe", "lateFeeCentsPerDay": 999}, "already_exists")
    c.refuse("addTool", "amy", W, {"toolId": "y1", "name": "Tool y1", "familyId": "nofam", "lateFeeCentsPerDay": 5}, "not_found")
    c.refuse("addTool", "bob", W, {"toolId": "y1", "name": "Tool y1", "familyId": "laser", "lateFeeCentsPerDay": 5}, "forbidden")
    expected = [tl("X1", 20), tl("x-1", 1), tl("x.1", 75), tl("x1", 200), tl("x10", 100000, "lathe"), tl("x_1", 150, "lathe")]
    c.ok("listTools", "bob", W, {}, {"tools": expected})
    c.ok("listTools", "sam", W, {}, {"tools": expected})
    c.ok("listTools", "amy", W, {}, {"tools": expected})
    c.audit_log(W)
    return c


def borrow_1() -> Case:
    c = Case("C5-borrow-1", "C5-borrow", "Borrow tool, Return tool, Tool, Loan: the loan is lon-1 with its own counter, active, dueOn 14 days after borrowedAt, chargedDays 0, and the "
                                        "tool is onLoan; a refusal consumes no loan number; borrowing creates no charge, no notice and no journal entry; a returned tool is "
                                        "free again and another member borrows it as lon-3; the audit targets are the new loanId and the loanId")
    world(c)
    add(c, "drill", 200)
    add(c, "saw", 150)
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    c.ok("listTools", "bob", a, {}, {"tools": [tl("drill", 200, on=True), tl("saw", 150)]})
    unbilled(c, "ada", a, 5000)
    notices_of(c, a, "ada", [])
    c.ok("listJournalEntries", "amy", a, {"month": "2026-03"}, {"month": "2026-03", "entries": []})
    c.refuse("borrowTool", "amy", a, {"memberId": "ada", "toolId": "drill"}, "tool_on_loan")
    lend(c, "amy", a, "ada", "saw", 150, "lon-2", "2026-03-16")
    r = t(10, "09:00")
    back(c, "sam", r, "lon-1", "drill", "ada", a, "2026-03-16", 200)
    loans_of(c, "ada", r, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-16", returned=r), ln("lon-2", "saw", "ada", a, "2026-03-16")])
    certify(c, "eve", r)
    lend(c, "amy", r, "eve", "drill", 200, "lon-3", "2026-03-24")
    c.ok("listTools", "sam", r, {}, {"tools": [tl("drill", 200, on=True), tl("saw", 150, on=True)]})
    notices_of(c, r, "ada", [])
    c.audit_log(r)
    return c


# ---- C5-due: the due date -----------------------------------------------------------------------------------------------

def due_1() -> Case:
    c = Case("C5-due-1", "C5-due", "Loan, dueOn: 14 days after the date of borrowedAt, over a month end, a year end, a non-leap February and a leap February: "
                                   "2026-02-20 gives 2026-03-06, 2026-12-25T23:59 gives 2027-01-08, 2027-02-15 gives 2027-03-01, 2028-02-20 gives 2028-03-05")
    s = "2026-02-20T09:00Z"
    world(c, s)
    add(c, "drill", 200, at=s)
    spans = [(s, "lon-1", "2026-03-06"), ("2026-12-25T23:59Z", "lon-2", "2027-01-08"), ("2027-02-15T10:00Z", "lon-3", "2027-03-01"),
             ("2028-02-20T00:00Z", "lon-4", "2028-03-05")]
    loans = []
    for at, lid, due in spans:
        if lid == "lon-4":
            certify(c, "ada", at)  # the certification of 2026-02-20 has expired
        lend(c, "amy", at, "ada", "drill", 200, lid, due)
        back(c, "amy", at, lid, "drill", "ada", at, due, 200)
        loans.append(ln(lid, "drill", "ada", at, due, returned=at))
    loans_of(c, "ada", "2028-02-20T00:00Z", "ada", loans)
    c.audit_log("2028-02-20T00:00Z")
    return c


def due_2() -> Case:
    c = Case("C5-due-2", "C5-due", "Computed times, Loan: a loan taken on 2099-12-31 is due on 2100-01-14 and one taken on 2099-12-18 on 2100-01-01, in the date format with a "
                                   "four-digit year; a loan whose dueOn lies after 2099 is never overdue, so nothing is due to charge")
    a = "2099-12-18T09:00Z"
    e = "2099-12-31T23:59Z"
    world(c, a)
    add(c, "drill", 200, at=a)
    add(c, "saw", 150, at=a)
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2100-01-01")
    lend(c, "amy", e, "ada", "saw", 150, "lon-2", "2100-01-14")
    loans_of(c, "ada", e, "ada", [ln("lon-1", "drill", "ada", a, "2100-01-01"), ln("lon-2", "saw", "ada", e, "2100-01-14")])
    run(c, "sam", e, [])
    c.audit_log(e)
    return c


# ---- C5-overdue: overdue, days overdue, the charge ----------------------------------------------------------------------

def overdue_1() -> Case:
    c = Case("C5-overdue-1", "C5-overdue", "Overdue loans, Late-fee charge, example: dueOn 2026-03-16 is not overdue, a run on 03-19 charges 3 days (600), on 03-20 1 day (200), "
                                          "a return on 03-22 the remaining 2 days (400); chargedDays follows; a second run the same day charges nothing; the notice kind loan is "
                                          "posted once, in the first charging run, for the loan's member with the subject of the kind; a return posts none; a returned loan is "
                                          "no longer overdue; the charges are unbilled on the member's account")
    world(c)
    add(c, "drill", 200)
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    e = t(16, "23:59")
    loans_of(c, "ada", e, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-16")])
    run(c, "sam", e, [])
    d3 = t(19, "08:00")
    loans_of(c, "ada", d3, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-16", overdue=True)])
    run(c, "sam", d3, [late("lon-1", 3, 200, d3)])
    loans_of(c, "ada", d3, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-16", overdue=True, charged=3)])
    notices_of(c, d3, "ada", [ntc("ntc-1", "ada", "lon-1", d3)])
    run(c, "amy", t(19, "09:00"), [])
    d4 = t(20, "08:00")
    run(c, "sue", d4, [late("lon-1", 1, 200, d4)])
    notices_of(c, d4, "ada", [ntc("ntc-1", "ada", "lon-1", d3)])
    d6 = t(22, "08:00")
    back(c, "sam", d6, "lon-1", "drill", "ada", a, "2026-03-16", 200, charge=late("lon-1", 2, 200, d6), charged=6)
    unbilled(c, "ada", d6, 6200)
    run(c, "sam", t(22, "09:00"), [])
    notices_of(c, t(22, "09:00"), "ada", [ntc("ntc-1", "ada", "lon-1", d3)])
    c.audit_log(t(22, "09:00"))
    return c


def overdue_2() -> Case:
    c = Case("C5-overdue-2", "C5-overdue", "Overdue loans, days overdue: the days are calendar days from dueOn to the date of the request time, not 24-hour periods: a loan taken at "
                                          "23:59 on 03-02 (dueOn 03-16) is not overdue at 03-16T23:59 and has 1 day at 03-17T00:00, still 1 at 03-17T23:59, 2 at 03-18T00:00; "
                                          "a loan taken at 00:00 on 03-03 (dueOn 03-17) is not overdue on 03-17 and has 1 day on 03-18; the notices come in the run that first charges")
    world(c)
    add(c, "drill", 200)
    add(c, "saw", 150)
    a, b = t(2, "23:59"), t(3, "00:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    lend(c, "amy", b, "ada", "saw", 150, "lon-2", "2026-03-17")
    x1 = t(16, "23:59")
    loans_of(c, "ada", x1, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-16"), ln("lon-2", "saw", "ada", b, "2026-03-17")])
    run(c, "sam", x1, [])
    x2 = t(17, "00:00")
    run(c, "sam", x2, [late("lon-1", 1, 200, x2)])
    run(c, "sam", t(17, "23:59"), [])
    x4 = t(18, "00:00")
    run(c, "sam", x4, [late("lon-1", 1, 200, x4), late("lon-2", 1, 150, x4)])
    loans_of(c, "ada", x4, "ada", [ln("lon-1", "drill", "ada", a, "2026-03-16", overdue=True, charged=2),
                                   ln("lon-2", "saw", "ada", b, "2026-03-17", overdue=True, charged=1)])
    notices_of(c, x4, "ada", [ntc("ntc-1", "ada", "lon-1", x2), ntc("ntc-2", "ada", "lon-2", x4)])
    c.audit_log(x4)
    return c


def overdue_3() -> Case:
    c = Case("C5-overdue-3", "C5-overdue", "Overdue loans, days overdue, Time: days between dates over a leap February and a year end: dueOn 2028-02-24 has 6 days on 2028-03-01 and 7 on "
                                          "2028-03-02; dueOn 2029-01-03 has 2 days on 2029-01-05; a member with the overdue loan is refused loan_overdue, and may borrow again "
                                          "once it is returned")
    s = "2028-02-10T09:00Z"
    world(c, s)
    add(c, "drill", 100, at=s)
    add(c, "saw", 250, at=s)
    lend(c, "amy", s, "ada", "drill", 100, "lon-1", "2028-02-24")
    m1 = "2028-03-01T09:00Z"
    c.refuse("borrowTool", "amy", m1, {"memberId": "ada", "toolId": "saw"}, "loan_overdue")
    run(c, "sam", m1, [late("lon-1", 6, 100, m1)])
    m2 = "2028-03-02T09:00Z"
    back(c, "sam", m2, "lon-1", "drill", "ada", s, "2028-02-24", 100, charge=late("lon-1", 1, 100, m2), charged=7)
    d1 = "2028-12-20T09:00Z"
    lend(c, "amy", d1, "ada", "saw", 250, "lon-2", "2029-01-03")
    d2 = "2029-01-05T09:00Z"
    run(c, "sam", d2, [late("lon-2", 2, 250, d2)])
    c.audit_log(d2)
    return c


def big_1() -> Case:
    c = Case("C5-big-1", "C5-big", "Late-fee charge, Money, Integer limits: a tool at the highest fee, 100000 cents a day, kept from 2000-01-01 (dueOn 2000-01-15) to 2099-12-31 is "
                                   "36510 days overdue and the charge is 3,651,000,000 cents, beyond 32 bits; chargedDays holds the days; the return then charges nothing")
    s = "2000-01-01T00:00Z"
    e = "2099-12-31T23:59Z"
    world(c, s)
    add(c, "gold", 100000, at=s)
    lend(c, "amy", s, "ada", "gold", 100000, "lon-1", "2000-01-15")
    run(c, "sam", e, [late("lon-1", 36510, 100000, e)])
    loans_of(c, "ada", e, "ada", [ln("lon-1", "gold", "ada", s, "2000-01-15", overdue=True, charged=36510)])
    unbilled(c, "ada", e, 3651005000)
    back(c, "sam", e, "lon-1", "gold", "ada", s, "2000-01-15", 100000, charge=None, charged=36510)
    c.audit_log(e)
    return c


# ---- C5-limit, C5-order, C5-gate: the refusals of borrowTool -------------------------------------------------------------

def limit_1() -> Case:
    c = Case("C5-limit-1", "C5-limit", "Borrow tool, loan_limit: a member with 3 active loans is refused the 4th with the tool free and no number consumed; the limit is per member "
                                       "(eve borrows); a returned loan no longer counts, and the limit is reached again with the next loan; List member loans: all loans of "
                                       "the member in creation order, active and returned, none of another member")
    world(c)
    certify(c, "eve")
    for tid, fee in (("a", 100), ("b", 150), ("c", 200), ("d", 250), ("e", 300)):
        add(c, tid, fee)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "a", 100, "lon-1", "2026-03-16")
    lend(c, "amy", x, "ada", "b", 150, "lon-2", "2026-03-16")
    lend(c, "amy", x, "ada", "c", 200, "lon-3", "2026-03-16")
    c.refuse("borrowTool", "amy", x, {"memberId": "ada", "toolId": "d"}, "loan_limit")
    lend(c, "amy", x, "eve", "d", 250, "lon-4", "2026-03-16")
    y = t(5, "10:00")
    back(c, "sam", y, "lon-2", "b", "ada", x, "2026-03-16", 150)
    lend(c, "amy", y, "ada", "e", 300, "lon-5", "2026-03-19")
    c.refuse("borrowTool", "amy", y, {"memberId": "ada", "toolId": "b"}, "loan_limit")
    loans_of(c, "ada", y, "ada", [ln("lon-1", "a", "ada", x, "2026-03-16"), ln("lon-2", "b", "ada", x, "2026-03-16", returned=y),
                                  ln("lon-3", "c", "ada", x, "2026-03-16"), ln("lon-5", "e", "ada", y, "2026-03-19")])
    loans_of(c, "sam", y, "eve", [ln("lon-4", "d", "eve", x, "2026-03-16")])
    c.audit_log(y)
    return c


def order_1() -> Case:
    c = Case("C5-order-1", "C5-order", "Borrow tool, refusals in order: bob is suspended, a minor without consent, uncertified, without the signed waiver, wants a tool that is on loan, and "
                                       "has an overdue loan, while the staff actor is off shift; the refusals come approver_not_on_shift, member_suspended, "
                                       "guardian_consent_missing, not_certified, waiver_missing, tool_on_loan, loan_overdue as each is cured; the late return charges the fee to bob")
    world(c)
    certify(c, "bob")
    for tid, fee in (("ta", 200), ("tb", 150), ("tc", 100)):
        add(c, tid, fee)
    a = t(2, "10:00")
    lend(c, "amy", a, "bob", "ta", 200, "lon-1", "2026-03-16")
    lend(c, "amy", a, "ada", "tb", 150, "lon-2", "2026-03-16")
    x = t(20, "10:00")
    c.ok("setMemberStanding", "sam", x, {"memberId": "bob", "standing": "suspended"})
    c.ok("declareMinor", "sam", x, {"memberId": "bob", "birthDate": "2012-05-01", "guardianId": "ada"})
    c.ok("revokeCertification", "sam", x, {"memberId": "bob", "familyId": "laser"})
    c.ok("publishWaiver", "amy", x, {"familyId": "laser", "text": "Laser waiver"})
    on_loan = {"memberId": "bob", "toolId": "tb"}
    c.refuse("borrowTool", "sam", x, on_loan, "approver_not_on_shift")
    c.ok("addShift", "amy", x, {"userId": "sam", "start": x, "end": t(20, "18:00")}, new="shf-1")
    c.refuse("borrowTool", "sam", x, on_loan, "member_suspended")
    c.ok("setMemberStanding", "sam", x, {"memberId": "bob", "standing": "good"})
    c.refuse("borrowTool", "sam", x, on_loan, "guardian_consent_missing")
    c.ok("grantConsent", "ada", x, {"guardianId": "ada", "memberId": "bob", "familyId": "laser"})
    c.refuse("borrowTool", "sam", x, on_loan, "not_certified")
    certify(c, "bob", x)
    c.refuse("borrowTool", "sam", x, on_loan, "waiver_missing")
    c.ok("signWaiver", "bob", x, {"memberId": "bob", "familyId": "laser"})
    c.refuse("borrowTool", "sam", x, on_loan, "tool_on_loan")
    free = {"memberId": "bob", "toolId": "tc"}
    c.refuse("borrowTool", "sam", x, free, "loan_overdue")
    back(c, "sam", x, "lon-1", "ta", "bob", a, "2026-03-16", 200, charge=late("lon-1", 4, 200, x), charged=4)
    lend(c, "sam", x, "bob", "tc", 100, "lon-3", "2026-04-03")
    unbilled(c, "bob", x, 10800)
    c.audit_log(x)
    return c


def order_2() -> Case:
    c = Case("C5-order-2", "C5-order", "Borrow tool, refusals in order: tool_on_loan comes before loan_overdue and loan_limit, loan_overdue before loan_limit (ada holds 3 loans, all "
                                       "overdue, and is refused loan_overdue, not loan_limit); on dueOn itself the limit applies but not loan_overdue; the member is free to "
                                       "borrow again only when no loan is overdue")
    world(c)
    for tid, fee in (("a", 200), ("b", 150), ("c", 100), ("d", 300)):
        add(c, tid, fee)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "ada", "b", 150, "lon-2", "2026-03-16")
    lend(c, "amy", x, "ada", "c", 100, "lon-3", "2026-03-16")
    y = t(16, "09:00")
    certify(c, "eve", y)
    c.refuse("borrowTool", "amy", y, {"memberId": "ada", "toolId": "d"}, "loan_limit")
    c.refuse("borrowTool", "amy", y, {"memberId": "ada", "toolId": "a"}, "tool_on_loan")
    c.refuse("borrowTool", "amy", y, {"memberId": "eve", "toolId": "a"}, "tool_on_loan")
    z = t(17, "09:00")
    c.refuse("borrowTool", "amy", z, {"memberId": "ada", "toolId": "d"}, "loan_overdue")
    c.refuse("borrowTool", "amy", z, {"memberId": "ada", "toolId": "a"}, "tool_on_loan")
    back(c, "sam", z, "lon-1", "a", "ada", x, "2026-03-16", 200, charge=late("lon-1", 1, 200, z), charged=1)
    c.refuse("borrowTool", "amy", z, {"memberId": "ada", "toolId": "d"}, "loan_overdue")
    back(c, "sam", z, "lon-2", "b", "ada", x, "2026-03-16", 150, charge=late("lon-2", 1, 150, z), charged=1)
    back(c, "sam", z, "lon-3", "c", "ada", x, "2026-03-16", 100, charge=late("lon-3", 1, 100, z), charged=1)
    lend(c, "amy", z, "ada", "d", 300, "lon-4", "2026-03-31")
    c.audit_log(z)
    return c


def gate_overdue_1() -> Case:
    c = Case("C5-gate-overdue-1", "C5-gate", "Borrow tool, loan_overdue, Overdue loans: judged on the date of the request time: on dueOn (03-16, even at 23:59) ada may borrow another "
                                             "tool, one day later she is refused loan_overdue; another member's overdue loan does not matter: eve borrows")
    world(c)
    certify(c, "eve")
    for tid, fee in (("a", 200), ("b", 150), ("c", 100)):
        add(c, tid, fee)
    lend(c, "amy", t(2, "10:00"), "ada", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", t(16, "23:59"), "ada", "b", 150, "lon-2", "2026-03-30")
    g = t(17, "00:00")
    c.refuse("borrowTool", "amy", g, {"memberId": "ada", "toolId": "c"}, "loan_overdue")
    lend(c, "amy", g, "eve", "c", 100, "lon-3", "2026-03-31")
    c.audit_log(g)
    return c


def standing_1() -> Case:
    c = Case("C5-gate-standing-1", "C5-gate", "Borrow tool, Membership: a paused member and a suspended member are refused member_suspended though certified and with nothing else wrong; "
                                              "no loan number is used; reinstated, they borrow (lon-1, lon-2)")
    world(c)
    certify(c, "bob")
    certify(c, "eve")
    add(c, "a", 200)
    add(c, "b", 150)
    c.ok("setMemberStanding", "sam", W, {"memberId": "bob", "standing": "paused"})
    c.ok("setMemberStanding", "sam", W, {"memberId": "eve", "standing": "suspended"})
    c.refuse("borrowTool", "amy", W, {"memberId": "bob", "toolId": "a"}, "member_suspended")
    c.refuse("borrowTool", "amy", W, {"memberId": "eve", "toolId": "a"}, "member_suspended")
    c.ok("setMemberStanding", "sam", W, {"memberId": "bob", "standing": "good"})
    c.ok("setMemberStanding", "sam", W, {"memberId": "eve", "standing": "good"})
    lend(c, "amy", W, "bob", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", W, "eve", "b", 150, "lon-2", "2026-03-16")
    c.audit_log(W)
    return c


def standing_2() -> Case:
    c = Case("C5-gate-standing-2", "C5-gate", "Return tool, Charge overdue loans, Membership: standing is not checked: a charging run charges the loans of a suspended and a paused member and "
                                              "posts their notices, and both return their tools, staff without a shift returning; the late fee is on the member's account, the "
                                              "standing is unchanged")
    world(c)
    certify(c, "bob")
    certify(c, "eve")
    for tid, fee in (("a", 200), ("b", 150), ("c", 100)):
        add(c, tid, fee)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "bob", "b", 150, "lon-2", "2026-03-16")
    lend(c, "amy", x, "eve", "c", 100, "lon-3", "2026-03-16")
    y = t(17, "09:00")
    c.ok("setMemberStanding", "sam", y, {"memberId": "ada", "standing": "suspended"})
    c.ok("setMemberStanding", "sam", y, {"memberId": "bob", "standing": "paused"})
    run(c, "sam", y, [late("lon-1", 1, 200, y), late("lon-2", 1, 150, y), late("lon-3", 1, 100, y)])
    notices_of(c, y, "ada", [ntc("ntc-1", "ada", "lon-1", y)])
    notices_of(c, y, "bob", [ntc("ntc-2", "bob", "lon-2", y)])
    notices_of(c, y, "eve", [ntc("ntc-3", "eve", "lon-3", y)])
    unbilled(c, "ada", y, 5200)
    unbilled(c, "bob", y, 5150)
    z = t(18, "09:00")
    back(c, "sue", z, "lon-1", "a", "ada", x, "2026-03-16", 200, charge=late("lon-1", 1, 200, z), charged=2)
    back(c, "sue", z, "lon-2", "b", "bob", x, "2026-03-16", 150, charge=late("lon-2", 1, 150, z), charged=2)
    c.ok("getMember", "sam", z, {"memberId": "ada"}, {"member": {"memberId": "ada", "name": "Ada Byron", "tier": "plus", "standing": "suspended", "registeredAt": W}})
    c.ok("getMember", "sam", z, {"memberId": "bob"}, {"member": {"memberId": "bob", "name": "Bob Stone", "tier": "basic", "standing": "paused", "registeredAt": W}})
    unbilled(c, "ada", z, 5400)
    c.audit_log(z)
    return c


def consent_1() -> Case:
    c = Case("C5-gate-consent-1", "C5-gate", "Borrow tool, guardian consent: a minor without consent for the tool's family is refused guardian_consent_missing (consent for another family "
                                             "does not help); the same minor borrows a tool of the family the guardian consented to, and the lathe tool once the lathe consent is "
                                             "granted")
    world(c)
    add(c, "drill", 200)
    add(c, "chisel", 300, "lathe")
    c.ok("addCourse", "sam", W, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 0, "capacity": 5})
    c.ok("declareMinor", "sam", W, {"memberId": "bob", "birthDate": "2012-05-01", "guardianId": "ada"})
    for family in ("laser", "lathe"):
        c.ok("grantConsent", "ada", W, {"guardianId": "ada", "memberId": "bob", "familyId": family})
    certify(c, "bob")
    certify(c, "bob", W, "lathe-101")
    c.ok("withdrawConsent", "ada", W, {"guardianId": "ada", "memberId": "bob", "familyId": "lathe"})
    x = t(2, "10:00")
    c.refuse("borrowTool", "amy", x, {"memberId": "bob", "toolId": "chisel"}, "guardian_consent_missing")
    lend(c, "amy", x, "bob", "drill", 200, "lon-1", "2026-03-16")
    c.ok("grantConsent", "ada", x, {"guardianId": "ada", "memberId": "bob", "familyId": "lathe"})
    lend(c, "amy", x, "bob", "chisel", 300, "lon-2", "2026-03-16", "lathe")
    c.audit_log(x)
    return c


def consent_2() -> Case:
    c = Case("C5-gate-consent-2", "C5-gate", "Borrow tool, guardian consent: the date of the request time decides: a member born 2008-03-10 is a minor on 2026-03-09 (refused "
                                             "guardian_consent_missing at 23:59) and an adult on 2026-03-10 (borrows at 00:00), consent or not")
    world(c)
    add(c, "drill", 200)
    c.ok("declareMinor", "sam", W, {"memberId": "eve", "birthDate": "2008-03-10", "guardianId": "ada"})
    c.ok("grantConsent", "ada", W, {"guardianId": "ada", "memberId": "eve", "familyId": "laser"})
    certify(c, "eve")
    c.ok("withdrawConsent", "ada", W, {"guardianId": "ada", "memberId": "eve", "familyId": "laser"})
    c.refuse("borrowTool", "amy", t(9, "23:59"), {"memberId": "eve", "toolId": "drill"}, "guardian_consent_missing")
    lend(c, "amy", t(10, "00:00"), "eve", "drill", 200, "lon-1", "2026-03-24")
    c.audit_log(t(10, "00:00"))
    return c


def cert_1() -> Case:
    c = Case("C5-gate-cert-1", "C5-gate", "Borrow tool, certification: the certification must be for the tool's family (a laser certification does not serve a lathe tool); a revoked "
                                          "certification is no certification; certified again, the member borrows")
    world(c)
    add(c, "drill", 200)
    add(c, "chisel", 300, "lathe")
    x = t(2, "10:00")
    c.refuse("borrowTool", "amy", x, {"memberId": "ada", "toolId": "chisel"}, "not_certified")
    c.ok("revokeCertification", "sam", x, {"memberId": "ada", "familyId": "laser"})
    c.refuse("borrowTool", "amy", x, {"memberId": "ada", "toolId": "drill"}, "not_certified")
    certify(c, "ada", x)
    lend(c, "amy", x, "ada", "drill", 200, "lon-1", "2026-03-16")
    c.audit_log(x)
    return c


def cert_2() -> Case:
    c = Case("C5-gate-cert-2", "C5-gate", "Borrow tool, certification, Certifications: a certification of 2026-03-02 is valid until 2027-03-01 and not on 2027-03-02 (expiresOn); the date "
                                          "of the request time decides; certified again, the member borrows")
    world(c)
    add(c, "drill", 200)
    e1, e2 = "2027-03-01T23:59Z", "2027-03-02T00:00Z"
    lend(c, "amy", e1, "ada", "drill", 200, "lon-1", "2027-03-15")
    back(c, "amy", e1, "lon-1", "drill", "ada", e1, "2027-03-15", 200)
    c.refuse("borrowTool", "amy", e2, {"memberId": "ada", "toolId": "drill"}, "not_certified")
    certify(c, "ada", e2)
    lend(c, "amy", e2, "ada", "drill", 200, "lon-2", "2027-03-16")
    c.audit_log(e2)
    return c


def waiver_1() -> Case:
    c = Case("C5-gate-waiver-1", "C5-gate", "Borrow tool, waiver gate: a family without a waiver needs no signature; once the tool's family has one, an unsigned member is refused "
                                           "waiver_missing and a signature of another family does not help; a newer version needs a new signature; the lathe tool is judged on "
                                           "the lathe waiver alone")
    world(c)
    add(c, "drill", 200)
    add(c, "chisel", 300, "lathe")
    c.ok("addCourse", "sam", W, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 0, "capacity": 5})
    certify(c, "ada", W, "lathe-101")
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "drill", 200, "lon-1", "2026-03-16")
    back(c, "amy", x, "lon-1", "drill", "ada", x, "2026-03-16", 200)
    c.ok("publishWaiver", "amy", x, {"familyId": "laser", "text": "Laser waiver"})
    c.refuse("borrowTool", "amy", x, {"memberId": "ada", "toolId": "drill"}, "waiver_missing")
    c.ok("publishWaiver", "amy", x, {"familyId": "lathe", "text": "Lathe waiver"})
    c.ok("signWaiver", "ada", x, {"memberId": "ada", "familyId": "lathe"})
    c.refuse("borrowTool", "amy", x, {"memberId": "ada", "toolId": "drill"}, "waiver_missing")
    c.ok("signWaiver", "ada", x, {"memberId": "ada", "familyId": "laser"})
    lend(c, "amy", x, "ada", "drill", 200, "lon-2", "2026-03-16")
    back(c, "amy", x, "lon-2", "drill", "ada", x, "2026-03-16", 200)
    c.ok("publishWaiver", "amy", x, {"familyId": "laser", "text": "Laser waiver, new"})
    c.refuse("borrowTool", "amy", x, {"memberId": "ada", "toolId": "drill"}, "waiver_missing")
    c.ok("signWaiver", "ada", x, {"memberId": "ada", "familyId": "laser"})
    lend(c, "amy", x, "ada", "drill", 200, "lon-3", "2026-03-16")
    lend(c, "amy", x, "ada", "chisel", 300, "lon-4", "2026-03-16", "lathe")
    c.audit_log(x)
    return c


def waiver_2() -> Case:
    c = Case("C5-gate-waiver-2", "C5-gate", "Borrow tool, waiver gate, lapse (step 2): a signature of 2026-03-03 counts on 2027-03-02 and has lapsed on 2027-03-03 (365 days after the date "
                                           "of signedAt), so the borrower is refused waiver_missing; signing again lets the member borrow")
    world(c)
    add(c, "drill", 200)
    c.ok("publishWaiver", "amy", W, {"familyId": "laser", "text": "Laser waiver"})
    c.ok("signWaiver", "eve", t(3, "09:00"), {"memberId": "eve", "familyId": "laser"})
    certify(c, "eve", "2026-04-01T09:00Z")
    b1, b2 = "2027-03-02T23:59Z", "2027-03-03T00:00Z"
    lend(c, "amy", b1, "eve", "drill", 200, "lon-1", "2027-03-16")
    back(c, "amy", b1, "lon-1", "drill", "eve", b1, "2027-03-16", 200)
    c.refuse("borrowTool", "amy", b2, {"memberId": "eve", "toolId": "drill"}, "waiver_missing")
    c.ok("signWaiver", "eve", b2, {"memberId": "eve", "familyId": "laser"})
    lend(c, "amy", b2, "eve", "drill", 200, "lon-2", "2027-03-17")
    c.audit_log(b2)
    return c


def shift_1() -> Case:
    c = Case("C5-gate-shift-1", "C5-gate", "Borrow tool, Decide volunteer time (on shift): staff need a shift of their own that has started and not ended: a shift starting at the request time "
                                          "counts, one ending at the request time does not, another person's shift does not, a cancelled shift counts for nothing; an admin needs "
                                          "none; returning a tool needs none")
    world(c)
    certify(c, "bob")
    certify(c, "eve")
    for tid, fee in (("drill", 200), ("saw", 150), ("sander", 100), ("router", 250)):
        add(c, tid, fee)
    c.ok("addShift", "amy", W, {"userId": "sam", "start": W, "end": t(2, "13:00")}, new="shf-1")
    c.ok("addShift", "amy", W, {"userId": "sue", "start": t(2, "11:00"), "end": t(2, "15:00")}, new="shf-2")
    lend(c, "sam", W, "ada", "drill", 200, "lon-1", "2026-03-16")
    c.refuse("borrowTool", "sue", t(2, "10:59"), {"memberId": "bob", "toolId": "saw"}, "approver_not_on_shift")
    lend(c, "sue", t(2, "11:00"), "bob", "saw", 150, "lon-2", "2026-03-16")
    lend(c, "sam", t(2, "12:59"), "eve", "sander", 100, "lon-3", "2026-03-16")
    w = t(2, "13:00")
    c.refuse("borrowTool", "sam", w, {"memberId": "eve", "toolId": "router"}, "approver_not_on_shift")
    lend(c, "amy", w, "eve", "router", 250, "lon-4", "2026-03-16")
    back(c, "sam", w, "lon-1", "drill", "ada", W, "2026-03-16", 200)
    c.ok("addShift", "amy", t(2, "13:30"), {"userId": "sam", "start": t(2, "14:00"), "end": t(2, "15:00")}, new="shf-3")
    c.ok("cancelShift", "amy", t(2, "13:30"), {"shiftId": "shf-3"})
    u = t(2, "14:00")
    c.refuse("borrowTool", "sam", u, {"memberId": "ada", "toolId": "drill"}, "approver_not_on_shift")
    lend(c, "sue", u, "ada", "drill", 200, "lon-5", "2026-03-16")
    c.audit_log(u)
    return c


def machine_rules_1() -> Case:
    c = Case("C5-gate-machine-1", "C5-gate", "Borrow tool: the rules for machines do not apply to tools: with PPE required for the family and not held, both laser machines locked out and in "
                                             "maintenance, an inspection due, supervision required and the badge outside, ada borrows a laser tool")
    world(c)
    add(c, "drill", 200)
    c.ok("addPpeItem", "sam", W, {"itemId": "gl", "name": "Glasses", "stock": 1})
    c.ok("requirePpe", "amy", W, {"familyId": "laser", "itemId": "gl"})
    c.ok("applyLockout", "sam", W, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("applyLockout", "sam", W, {"machineId": "L2", "reason": "lens"}, new="lck-2")
    c.ok("scheduleMaintenance", "sam", W, {"machineId": "L1", "start": t(3, "09:00"), "end": t(3, "18:00"), "reason": "service"}, new="mnt-1")
    c.ok("scheduleMaintenance", "sam", W, {"machineId": "L2", "start": t(3, "09:00"), "end": t(3, "18:00"), "reason": "service"}, new="mnt-2")
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 1})
    c.ok("setSupervision", "amy", W, {"familyId": "laser", "required": True})
    c.ok("issueBadge", "sam", W, {"memberId": "ada"}, new="bdg-1")
    x = t(3, "10:00")
    lend(c, "amy", x, "ada", "drill", 200, "lon-1", "2026-03-17")
    c.audit_log(x)
    return c


# ---- C5-run, C5-return: the two ways a late fee is charged --------------------------------------------------------------

def run_1() -> Case:
    c = Case("C5-run-1", "C5-run", "Charge overdue loans, List order, Posting notices: ten overdue loans of four members come in ascending loan number as a number (lon-9 before lon-10), "
                                   "not by tool, member or text order; each loan's notice takes the next notice number after a manual notice, in that order, and the member of "
                                   "the loan gets it; a second run charges nothing")
    world(c)
    c.ok("registerMember", "sam", W, {"memberId": "gus", "name": "Gus Hale", "tier": "basic"})
    for mem in ("bob", "eve", "gus"):
        certify(c, mem)
    owner = {1: "ada", 2: "bob", 3: "eve", 4: "gus", 5: "ada", 6: "bob", 7: "eve", 8: "gus", 9: "ada", 10: "bob"}
    fees = {n: 100 + 10 * n for n in owner}
    tid = {n: "t-" + "jihgfedcba"[n - 1] for n in owner}  # the tool ids fall as the loan numbers rise
    for n in owner:
        add(c, tid[n], fees[n])
    x = t(2, "10:00")
    for n in owner:
        lend(c, "amy", x, owner[n], tid[n], fees[n], f"lon-{n}", "2026-03-16")
    y = t(18, "09:00")
    c.ok("postNotice", "sam", y, {"memberId": "bob", "subject": "Hello"}, new="ntc-1")
    z = t(18, "10:00")
    run(c, "sam", z, [late(f"lon-{n}", 2, fees[n], z) for n in owner])
    notices_of(c, z, "ada", [ntc(f"ntc-{n + 1}", "ada", f"lon-{n}", z) for n in (1, 5, 9)])
    manual = {"noticeId": "ntc-1", "memberId": "bob", "kind": "manual", "subject": "Hello", "ref": None, "at": y, "acknowledgedAt": None}
    notices_of(c, z, "bob", [manual] + [ntc(f"ntc-{n + 1}", "bob", f"lon-{n}", z) for n in (2, 6, 10)])
    notices_of(c, z, "eve", [ntc(f"ntc-{n + 1}", "eve", f"lon-{n}", z) for n in (3, 7)])
    notices_of(c, z, "gus", [ntc(f"ntc-{n + 1}", "gus", f"lon-{n}", z) for n in (4, 8)])
    run(c, "sam", z, [])
    c.audit_log(z)
    return c


def return_1() -> Case:
    c = Case("C5-return-1", "C5-return", "Return tool: on dueOn no charge (charge null); a loan that is returned is refused invalid_state with nothing changed; a member is forbidden, an unknown "
                                         "loan not_found; staff without a shift return; a loan fully charged by a run the same day returns with charge null; no notice is posted "
                                         "by a return; the tool is free to borrow again")
    world(c)
    certify(c, "eve")
    add(c, "drill", 200)
    add(c, "saw", 150)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "drill", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "eve", "saw", 150, "lon-2", "2026-03-16")
    d = t(16, "12:00")
    back(c, "sue", d, "lon-2", "saw", "eve", x, "2026-03-16", 150)
    c.refuse("returnTool", "sue", d, {"loanId": "lon-2"}, "invalid_state")
    c.refuse("returnTool", "eve", d, {"loanId": "lon-1"}, "forbidden")
    c.refuse("returnTool", "sue", d, {"loanId": "lon-9"}, "not_found")
    c.ok("listTools", "sam", d, {}, {"tools": [tl("drill", 200, on=True), tl("saw", 150)]})
    r = t(19, "08:00")
    run(c, "sam", r, [late("lon-1", 3, 200, r)])
    r2 = t(19, "09:00")
    back(c, "sam", r2, "lon-1", "drill", "ada", x, "2026-03-16", 200, charge=None, charged=3)
    lend(c, "amy", r2, "eve", "saw", 150, "lon-3", "2026-04-02")
    back(c, "amy", r2, "lon-3", "saw", "eve", r2, "2026-04-02", 150)
    notices_of(c, r2, "ada", [ntc("ntc-1", "ada", "lon-1", r)])
    notices_of(c, r2, "eve", [])
    c.audit_log(r2)
    return c


def list_1() -> Case:
    c = Case("C5-list-1", "C5-list", "List member loans, Order of checks: a member lists only their own loans (another member is forbidden), staff and admin any; a member without loans gets an "
                                     "empty list; a staff user's identifier is no member (not_found); a member actor is forbidden for borrowTool before the records are looked up")
    world(c)
    certify(c, "eve")
    add(c, "a", 200)
    add(c, "b", 150)
    x = t(2, "10:00")
    lend(c, "amy", x, "ada", "a", 200, "lon-1", "2026-03-16")
    lend(c, "amy", x, "eve", "b", 150, "lon-2", "2026-03-16")
    own_ada = [ln("lon-1", "a", "ada", x, "2026-03-16")]
    loans_of(c, "ada", x, "ada", own_ada)
    c.refuse("listMemberLoans", "eve", x, {"memberId": "ada"}, "forbidden")
    loans_of(c, "sam", x, "eve", [ln("lon-2", "b", "eve", x, "2026-03-16")])
    loans_of(c, "amy", x, "ada", own_ada)
    loans_of(c, "bob", x, "bob", [])
    c.refuse("listMemberLoans", "sam", x, {"memberId": "sam"}, "not_found")
    c.refuse("listMemberLoans", "sam", x, {"memberId": "zed"}, "not_found")
    c.refuse("borrowTool", "amy", x, {"memberId": "sam", "toolId": "a"}, "not_found")
    c.refuse("borrowTool", "ada", x, {"memberId": "ada", "toolId": "nonesuch"}, "forbidden")
    c.refuse("returnTool", "ada", x, {"loanId": "nonesuch"}, "forbidden")
    run(c, "sam", x, [])
    c.audit_log(x)
    return c


# ---- C5-checkin, C5-money: what the loans reach in other modules --------------------------------------------------------

def checkin_1() -> Case:
    c = Case("C5-checkin-1", "C5-checkin", "Check in, overdue loans: ada checks in on dueOn, then her loan is overdue: check-out and reserving are not gated, check-in is refused loan_overdue "
                                           "(the reservation stays), another member's check-in goes through, and once the tool is returned her check-in goes through")
    world(c)
    certify(c, "eve")
    add(c, "drill", 200)
    book(c, "res-1", "ada", "L1", 16, "20:00", "21:00")
    book(c, "res-2", "ada", "L1", 17, "10:00", "11:00")
    book(c, "res-3", "eve", "L2", 17, "10:00", "11:00")
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    b = t(16, "20:50")
    c.ok("checkIn", "ada", b, {"reservationId": "res-1"}, {"reservation": res("res-1", "ada", "L1", t(16, "20:00"), t(16, "21:00"), "checkedIn", cin=b, rate=45)})
    z = t(17, "00:00")
    c.ok("checkOut", "ada", z, {"reservationId": "res-1"},
         {"reservation": res("res-1", "ada", "L1", t(16, "20:00"), t(16, "21:00"), "completed", cin=b, cout=z, rate=45, met=190, amt=7695),
          "charge": chg("usage", "res-1", 190, 45, 10, 7695, z)})
    d = t(17, "10:00")
    c.refuse("checkIn", "ada", d, {"reservationId": "res-2"}, "loan_overdue")
    c.ok("checkIn", "eve", d, {"reservationId": "res-3"}, {"reservation": res("res-3", "eve", "L2", t(17, "10:00"), t(17, "11:00"), "checkedIn", cin=d, rate=60)})
    c.ok("createReservation", "ada", d, {"memberId": "ada", "machineId": "L1", "start": t(18, "10:00"), "end": t(18, "11:00")},
         {"reservation": res("res-4", "ada", "L1", t(18, "10:00"), t(18, "11:00"))}, new="res-4")
    e = t(17, "10:30")
    back(c, "sam", e, "lon-1", "drill", "ada", a, "2026-03-16", 200, charge=late("lon-1", 1, 200, e), charged=1)
    c.ok("checkIn", "ada", e, {"reservationId": "res-2"}, {"reservation": res("res-2", "ada", "L1", t(17, "10:00"), t(17, "11:00"), "checkedIn", cin=e, rate=45)})
    c.audit_log(e)
    return c


def checkin_2() -> Case:
    c = Case("C5-checkin-2", "C5-checkin", "Check in, order: loan_overdue is the last refusal: with the waiver renewed, PPE required, a lockout, the badge outside and a loan overdue, check-in is "
                                           "refused waiver_missing, ppe_missing, machine_locked_out, not_on_premises and only then loan_overdue; the return lets it through")
    world(c)
    add(c, "drill", 200)
    c.ok("publishWaiver", "amy", W, {"familyId": "laser", "text": "Laser waiver"})
    c.ok("signWaiver", "ada", W, {"memberId": "ada", "familyId": "laser"})
    book(c, "res-1", "ada", "L1", 17, "10:00", "11:00")
    c.ok("issueBadge", "sam", W, {"memberId": "ada"}, new="bdg-1")
    a = t(2, "10:00")
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    x = t(17, "09:00")
    c.ok("applyLockout", "sam", x, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("publishWaiver", "amy", x, {"familyId": "laser", "text": "Laser waiver, new"})
    c.ok("addPpeItem", "sam", x, {"itemId": "gl", "name": "Glasses", "stock": 1})
    c.ok("requirePpe", "amy", x, {"familyId": "laser", "itemId": "gl"})
    y = t(17, "10:00")
    one = {"reservationId": "res-1"}
    c.refuse("checkIn", "ada", y, one, "waiver_missing")
    c.ok("signWaiver", "ada", y, {"memberId": "ada", "familyId": "laser"})
    c.refuse("checkIn", "ada", y, one, "ppe_missing")
    c.ok("issuePpe", "sam", y, {"memberId": "ada", "itemId": "gl"})
    c.refuse("checkIn", "ada", y, one, "machine_locked_out")
    c.ok("releaseLockout", "sam", y, {"lockoutId": "lck-1"})
    c.refuse("checkIn", "ada", y, one, "not_on_premises")
    c.ok("enterDoor", "ada", y, {"badgeId": "bdg-1"})
    c.refuse("checkIn", "ada", y, one, "loan_overdue")
    back(c, "sam", y, "lon-1", "drill", "ada", a, "2026-03-16", 200, charge=late("lon-1", 1, 200, y), charged=1)
    c.ok("checkIn", "ada", y, one, {"reservation": res("res-1", "ada", "L1", t(17, "10:00"), t(17, "11:00"), "checkedIn", cin=y, rate=45)})
    c.audit_log(y)
    return c


def eligibility_1() -> Case:
    c = Case("C5-checkin-3", "C5-checkin", "Eligibility, Family compliance: loan_overdue is a check-in rule and stays out of the blockers: with an overdue loan ada is eligible for L1 and her "
                                           "compliance summary is unchanged")
    world(c)
    add(c, "drill", 200)
    lend(c, "amy", t(2, "10:00"), "ada", "drill", 200, "lon-1", "2026-03-16")
    y = t(18, "09:00")
    loans_of(c, "ada", y, "ada", [ln("lon-1", "drill", "ada", t(2, "10:00"), "2026-03-16", overdue=True)])
    c.ok("checkEligibility", "ada", y, {"memberId": "ada", "machineId": "L1", "date": "2026-03-19"},
         {"memberId": "ada", "machineId": "L1", "date": "2026-03-19", "eligible": True, "blockers": []})
    c.ok("complianceSummary", "ada", y, {"memberId": "ada"}, {"memberId": "ada", "standing": "good", "minor": False, "families": [
        fam("laser", True, True, True, True), fam("lathe", False, True, True, True)]})
    c.audit_log(y)
    return c


def march(c: Case) -> tuple[str, str, str]:
    """The world, a loan of ada's from 03-02 charged for 15 days on 03-31, and the invoices of March made on 04-01 (inv-1 ada, inv-2 bob,
    inv-3 eve). Returns the time of the loan, of the run and of the invoices."""
    world(c)
    add(c, "drill", 200)
    a, m, i = t(2, "10:00"), t(31, "10:00"), "2026-04-01T08:00Z"
    lend(c, "amy", a, "ada", "drill", 200, "lon-1", "2026-03-16")
    run(c, "sam", m, [late("lon-1", 15, 200, m)])
    c.ok("generateInvoices", "sam", i, {"month": "2026-03"})
    return a, m, i


MARCH_DUE = "2026-04-15T08:00Z"


def march_invoice(m: str, i: str, overdue: bool = False) -> dict:
    return invoice("inv-1", "ada", "2026-03", i, MARCH_DUE, [chg("membership", "plus", 1, 4900, 0, 4900, i), chg("course", "laser-101", 1, 5000, 0, 5000, W),
                                                              late("lon-1", 15, 200, m)], overdue)


def invoice_1() -> Case:
    c = Case("C5-money-1", "C5-money", "Charges and invoices, Postings, Charges by kind, Revenue report: the late fees are invoiced in the month of their at: 15 days (3000) on March's invoice "
                                       "after the membership and course lines, the remaining 5 days (1000) charged by the return in April on April's; each posts to revenueLateFees "
                                       "when invoiced, nothing before; the lateFee row of the charges by kind counts them, the four sums of the revenue report do not")
    a, m, i = march(c)
    c.ok("listJournalEntries", "amy", i, {"month": "2026-03"}, {"month": "2026-03", "entries": []})
    c.ok("listInvoices", "ada", i, {"memberId": "ada"}, {"memberId": "ada", "invoices": [march_invoice(m, i)]})
    c.ok("listJournalEntries", "amy", i, {"month": "2026-04"}, {"month": "2026-04", "entries": [
        je(1, i, "revenueMembership", 4900, "inv-1"), je(2, i, "revenueCourse", 5000, "inv-1"), je(3, i, "revenueLateFees", 3000, "inv-1"),
        je(4, i, "revenueMembership", 2500, "inv-2"), je(5, i, "revenueMembership", 9900, "inv-3")]})
    c.ok("chargesByKind", "sam", i, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds({"membership": 17300, "course": 5000, "lateFee": 3000}, 25300), "totalCents": 25300})
    c.ok("revenueReport", "sam", i, {"month": "2026-03"}, {"month": "2026-03", "invoicesGenerated": True, "invoiceCount": 3, "membershipCents": 17300, "usageCents": 0,
                                                           "materialCents": 0, "courseCents": 5000, "invoicedCents": 22300, "creditAppliedCents": 0,
                                                           "outstandingCents": 25300, "paymentsCents": 0, "creditsIssuedCents": 0})
    r = t(5, "10:00", 4)
    back(c, "sam", r, "lon-1", "drill", "ada", a, "2026-03-16", 200, charge=late("lon-1", 5, 200, r), charged=20)
    unbilled(c, "ada", r, 1000, open_=12900)
    i2 = "2026-05-01T08:00Z"
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"})
    c.ok("listInvoices", "ada", i2, {"memberId": "ada"}, {"memberId": "ada", "invoices": [
        march_invoice(m, i, overdue=True),
        invoice("inv-4", "ada", "2026-04", i2, "2026-05-15T08:00Z", [chg("membership", "plus", 1, 4900, 0, 4900, i2), late("lon-1", 5, 200, r)])]})
    c.ok("chargesByKind", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "rows": kinds({"membership": 17300, "lateFee": 1000}, 18300), "totalCents": 18300})
    c.ok("revenueReport", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoicesGenerated": True, "invoiceCount": 3, "membershipCents": 17300, "usageCents": 0,
                                                            "materialCents": 0, "courseCents": 0, "invoicedCents": 17300, "creditAppliedCents": 0,
                                                            "outstandingCents": 18300, "paymentsCents": 0, "creditsIssuedCents": 0})
    unbilled(c, "ada", i2, 0, open_=18800, overdue=12900)
    c.audit_log(i2)
    return c


def dunning_1() -> Case:
    c = Case("C5-money-2", "C5-money", "Dunning run, Posting notices: the dunning run's late fee is per invoice (ref the invoice, 700 cents) and is not touched by the late fee of a tool loan on "
                                       "the same invoice; its notices take the numbers after the loan notice (ntc-2, ntc-3, ntc-4); the account shows the invoice open and overdue "
                                       "and the dunning fee unbilled")
    a, m, i = march(c)
    c.ok("setDunningLadder", "amy", i, {"noticeDays": 1, "feeDays": 2, "feeCents": 700, "suspendDays": 30})
    y = "2026-04-17T08:00Z"
    c.ok("runDunning", "sam", y, {}, {"actions": [{"invoiceId": "inv-1", "step": "notice"}, {"invoiceId": "inv-1", "step": "fee"},
                                                  {"invoiceId": "inv-2", "step": "notice"}, {"invoiceId": "inv-2", "step": "fee"},
                                                  {"invoiceId": "inv-3", "step": "notice"}, {"invoiceId": "inv-3", "step": "fee"}]})
    c.ok("getAccount", "sam", y, {"memberId": "ada"}, account("ada", open_=12900, overdue=12900, unbilled=700))
    notices_of(c, y, "ada", [ntc("ntc-1", "ada", "lon-1", m), ntc("ntc-2", "ada", "inv-1", y, "dunning", "Invoice inv-1 is overdue")])
    notices_of(c, y, "bob", [ntc("ntc-3", "bob", "inv-2", y, "dunning", "Invoice inv-2 is overdue")])
    notices_of(c, y, "eve", [ntc("ntc-4", "eve", "inv-3", y, "dunning", "Invoice inv-3 is overdue")])
    c.audit_log(y)
    return c


def build_cases() -> list[dict[str, Any]]:
    makers = [tools_1, borrow_1, due_1, due_2, overdue_1, overdue_2, overdue_3, big_1, limit_1, order_1, order_2, gate_overdue_1, standing_1,
              standing_2, consent_1, consent_2, cert_1, cert_2, waiver_1, waiver_2, shift_1, machine_rules_1, run_1, return_1, list_1,
              checkin_1, checkin_2, eligibility_1, invoice_1, dunning_1]
    return [maker().build() for maker in makers]
