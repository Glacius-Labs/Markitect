"""Change step C3 of the large Makerspace: a fee for no-shows.

`chargeNoShows` (staff and admin, arguments `actor` and `at`) charges every reservation whose reported status is `missed`
at the request time, in ascending reservation number, unless an earlier run charged it or a lockout of its machine
overlapped its slot: kind `noShow`, ref the reservation, quantity the reserved minutes, the machine's rate at the request
time, discount 50, `at` the request time. The charge is a new kind, last in the kind order, and its invoice lines go to a
new ledger account `revenueNoShows` that sits between `revenueMembership` and `revenuePasses`. Written from the
requirements document after the change, with every expected value stated by hand.

What this step retires or moves: nothing is retired. Every case that compares a whole trial balance or `chargesByKind`
table gets one more row through `transform`: `revenueNoShows` with zeros between `revenueMembership` and `revenuePasses`
(24 rows), and `noShow` with zeros at the end of the kind rows (11 rows). Totals stay as they were.

Dates the cases rely on: 2026-03-02 is a Monday, so 03-03 Tue, 03-04 Wed, 03-05 Thu, 03-06 Fri, 03-09 Mon, 03-10 Tue,
03-30 Mon; 2026-03-07 and 03-08 are the weekend and are not used. Every reservation is booked at W (all at the start of
the world) so that later requests, which may not go back in time, can run the fee at the end of the slots.

The world's rates: L1 45, L2 60, T1 30 cents a minute. Tiers: ada plus (10 %), bob basic (0 %), eve pro (25 %). Joining
the laser course costs 5000 cents (a `course` charge at W), so a member certified in a case has 5000 cents unbilled
before anything else happens.
"""
from __future__ import annotations

from typing import Any

from kit import Case, has, world

W = "2026-03-02T09:00Z"  # Monday, the world's time

RETIRED: dict[str, str] = {}

ACCOUNTS = ["cash", "creditLiability", "depositLiability", "expenseGoodwill", "expenseMaker", "expensePayout",
            "expenseVolunteer", "expenseWriteOff", "inventory", "payable", "receivable", "revenueCourse", "revenueDeposits",
            "revenueEvents", "revenueGuests", "revenueJobs", "revenueLateFees", "revenueLockers", "revenueMaterial",
            "revenueMembership", "revenueNoShows", "revenuePasses", "revenueRefunds", "revenueUsage"]
KINDS = ["membership", "usage", "material", "course", "deposit", "pass", "locker", "guest", "lateFee", "event", "noShow"]


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


def empty_row(account: str) -> dict[str, Any]:
    return {"account": account, "debitCents": 0, "creditCents": 0, "balanceCents": 0, "side": "none"}


def transform(case: dict[str, Any]) -> dict[str, Any]:
    """A trial balance has a row more, `revenueNoShows` between `revenueMembership` and `revenuePasses`, and a charges-by-kind
    table has one more, `noShow` at the end. Both rows are empty in a case that knew nothing of the fee; the totals stay."""
    for step in case["steps"]:
        request, expect = step.get("request"), step["expect"]
        result = expect.get("result")
        if not request or not isinstance(result, dict) or not isinstance(result.get("rows"), list):
            continue
        rows = result["rows"]
        if request["op"] == "getTrialBalance" and len(rows) == 23:
            at = [r["account"] for r in rows].index("revenueMembership") + 1
            rows.insert(at, empty_row("revenueNoShows"))
        elif request["op"] == "chargesByKind" and len(rows) == 10:
            rows.append({"kind": "noShow", "invoicedCents": 0, "refundedCents": 0})
    return case


# ---- record builders (the members of each record, from the requirements) ---------------------------------------------

def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def chg(kind, ref_, qty, unit, disc, amt, at):
    assert (qty * unit * (100 - disc) + 50) // 100 == amt, (kind, ref_, qty, unit, disc, amt)
    return {"kind": kind, "ref": ref_, "quantity": qty, "unitPriceCents": unit, "discountPercent": disc,
            "amountCents": amt, "at": at}


def nsh(rid, minutes, rate, amt, at):
    """The charge a run creates for a missed reservation: the whole discount is the 50."""
    return chg("noShow", rid, minutes, rate, 50, amt, at)


def invoice(iid, mem, month, issued, due, lines, total):
    assert sum(line["amountCents"] for line in lines) == total, (iid, total)
    return {"invoiceId": iid, "memberId": mem, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": 0, "paidCents": 0, "balanceCents": total, "status": "open",
            "overdue": False}


def account(mem, credit, open_, overdue, unbilled):
    return {"account": {"memberId": mem, "creditBalanceCents": credit, "openBalanceCents": open_,
                        "overdueBalanceCents": overdue, "unbilledCents": unbilled}}


def je(n, at, debit, credit, amount, memo):
    return {"entryId": f"jrn-{n}", "at": at, "debitAccount": debit, "creditAccount": credit, "amountCents": amount, "memo": memo}


def tb(rows: dict, total: int) -> dict:
    """A trial balance of 24 rows: rows maps an account to (debitCents, creditCents); accounts not named are 0/0."""
    assert set(rows) <= set(ACCOUNTS), set(rows) - set(ACCOUNTS)
    out = []
    for a in ACCOUNTS:
        d, k = rows.get(a, (0, 0))
        out.append({"account": a, "debitCents": d, "creditCents": k, "balanceCents": abs(d - k),
                    "side": "debit" if d > k else "credit" if k > d else "none"})
    assert sum(r["debitCents"] for r in out) == total and sum(r["creditCents"] for r in out) == total
    return {"rows": out, "totalDebitCents": total, "totalCreditCents": total}


def kinds(invoiced: dict, refunded: dict | None, total: int) -> list:
    refunded = refunded or {}
    assert set(invoiced) <= set(KINDS) and set(refunded) <= set(KINDS)
    rows = [{"kind": k, "invoicedCents": invoiced.get(k, 0), "refundedCents": refunded.get(k, 0)} for k in KINDS]
    assert sum(r["invoicedCents"] for r in rows) == total
    return rows


# ---- steps used by many cases -------------------------------------------------------------------------------------------

def certify(c: Case, mem: str) -> None:
    c.ok("enrollInCourse", "sam", W, {"courseId": "laser-101", "memberId": mem})
    c.ok("completeCourse", "sam", W, {"courseId": "laser-101", "memberId": mem})


def book(c: Case, rid: str, mem: str, mach: str, day: int, start: str, end: str, month: int = 3) -> None:
    s, e = t(day, start, month), t(day, end, month)
    c.ok("createReservation", mem, W, {"memberId": mem, "machineId": mach, "start": s, "end": e},
         {"reservation": res(rid, mem, mach, s, e)}, new=rid)


def run(c: Case, actor: str, at: str, charges: list) -> None:
    c.ok("chargeNoShows", actor, at, {}, {"charges": charges})


def unbilled(c: Case, mem: str, at: str, cents: int) -> None:
    c.ok("getAccount", "sam", at, {"memberId": mem}, account(mem, 0, 0, 0, cents))


def lock(c: Case, lid: str, mach: str, applied: str, released: str | None = None, day: int = 0) -> None:
    """A manual lockout applied at `applied` and released at `released` (clock times of `day` of March)."""
    c.ok("applyLockout", "sam", t(day, applied), {"machineId": mach, "reason": f"{lid} on {mach}"}, new=lid)
    if released is not None:
        c.ok("releaseLockout", "sam", t(day, released), {"lockoutId": lid})


# ---- C3-run: what a run charges ------------------------------------------------------------------------------------------

def run_1() -> Case:
    c = Case("C3-run-1", "C3-run", "Charge no-shows, no-show fee, example: 60 minutes at 45 gives 1350, 15 minutes gives 338, 240 minutes gives 5400; "
                                    "a reservation whose end is the request time is taken, one minute before it is not; nothing is due gives an empty list; "
                                    "a missed reservation causes no charge by itself; a charged one is not charged again")
    world(c)
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    book(c, "res-2", "ada", "L1", 3, "11:00", "11:15")
    book(c, "res-3", "ada", "L1", 4, "09:00", "13:00")
    run(c, "sam", t(3, "10:59"), [])
    unbilled(c, "ada", t(3, "11:00"), 5000)
    run(c, "sam", t(3, "11:00"), [nsh("res-1", 60, 45, 1350, t(3, "11:00"))])
    unbilled(c, "ada", t(3, "11:00"), 6350)
    run(c, "sam", t(3, "11:14"), [])
    run(c, "sam", t(3, "11:15"), [nsh("res-2", 15, 45, 338, t(3, "11:15"))])
    run(c, "sam", t(3, "11:16"), [])
    unbilled(c, "ada", t(3, "11:16"), 6688)
    c.ok("listMemberReservations", "ada", t(3, "11:16"), {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "missed"),
        res("res-2", "ada", "L1", t(3, "11:00"), t(3, "11:15"), "missed"),
        res("res-3", "ada", "L1", t(4, "09:00"), t(4, "13:00"))]})
    run(c, "sam", t(4, "13:00"), [nsh("res-3", 240, 45, 5400, t(4, "13:00"))])
    run(c, "sam", t(4, "13:01"), [])
    unbilled(c, "ada", t(4, "13:01"), 12088)
    c.audit_log(t(4, "13:01"))
    return c


def run_2() -> Case:
    c = Case("C3-run-2", "C3-run", "Charge no-shows, order and List order: the charges come in ascending reservation number compared as a number, "
                                    "res-9 before res-10, not by slot time (res-1 has the latest slot) and not as text; an admin may run it")
    world(c)
    for n in range(1, 13):  # res-1 has the latest slot, res-12 the earliest: 11:45-12:00 down to 09:00-09:15
        start = 9 * 60 + 15 * (12 - n)
        book(c, f"res-{n}", "ada", "L1", 3, f"{start // 60:02d}:{start % 60:02d}", f"{(start + 15) // 60:02d}:{(start + 15) % 60:02d}")
    run(c, "amy", t(3, "12:00"), [nsh(f"res-{n}", 15, 45, 338, t(3, "12:00")) for n in range(1, 13)])
    unbilled(c, "ada", t(3, "12:00"), 9056)
    run(c, "amy", t(3, "12:00"), [])
    c.audit_log(t(3, "12:00"))
    return c


def run_3() -> Case:
    c = Case("C3-run-3", "C3-run", "Charge no-shows: the machine's rate at the request time, not the rate when the slot was booked; discount 50 whatever the "
                                    "member's tier (plus, pro, basic); the charge goes to the reservation's member, and rounds once per line")
    world(c)
    certify(c, "eve")
    certify(c, "bob")
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    book(c, "res-2", "eve", "L1", 3, "11:00", "11:45")
    book(c, "res-3", "bob", "L2", 3, "10:00", "10:30")
    c.ok("setMachineRate", "amy", t(3, "08:00"), {"machineId": "L1", "rateCentsPerMinute": 47})
    c.ok("setMachineRate", "amy", t(3, "08:05"), {"machineId": "L2", "rateCentsPerMinute": 111})
    run(c, "sue", t(3, "12:00"), [nsh("res-1", 60, 47, 1410, t(3, "12:00")), nsh("res-2", 45, 47, 1058, t(3, "12:00")),
                                    nsh("res-3", 30, 111, 1665, t(3, "12:00"))])
    unbilled(c, "ada", t(3, "12:00"), 6410)
    unbilled(c, "eve", t(3, "12:00"), 6058)
    unbilled(c, "bob", t(3, "12:00"), 6665)
    c.ok("setMachineRate", "amy", t(3, "12:10"), {"machineId": "L1", "rateCentsPerMinute": 90})
    run(c, "sue", t(3, "12:20"), [])
    unbilled(c, "ada", t(3, "12:20"), 6410)
    c.audit_log(t(3, "12:20"))
    return c


def run_4() -> Case:
    c = Case("C3-run-4", "C3-run", "Reservations, Charge no-shows: only a missed reservation is charged; cancelled, completed, checked-in and still-reserved ones are not; "
                                    "the member's standing is not checked; the reservation stays missed with amountCents null and still cannot be checked in or cancelled")
    world(c)
    certify(c, "eve")
    book(c, "res-1", "ada", "L1", 3, "09:00", "10:00")
    book(c, "res-2", "ada", "L1", 3, "10:00", "11:00")
    book(c, "res-3", "ada", "L1", 3, "11:00", "12:00")
    book(c, "res-4", "ada", "L1", 3, "12:00", "13:00")
    book(c, "res-5", "ada", "L1", 3, "14:00", "15:00")
    book(c, "res-6", "eve", "L2", 3, "09:00", "10:00")
    c.ok("cancelReservation", "ada", W, {"reservationId": "res-2"}, {"reservation": res("res-2", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "cancelled")})
    c.ok("setMemberStanding", "sam", t(3, "08:00"), {"memberId": "eve", "standing": "suspended"})
    c.ok("checkIn", "ada", t(3, "11:00"), {"reservationId": "res-3"}, {"reservation": res(
        "res-3", "ada", "L1", t(3, "11:00"), t(3, "12:00"), "checkedIn", cin=t(3, "11:00"), rate=45)})
    c.ok("checkOut", "ada", t(3, "11:20"), {"reservationId": "res-3"}, {
        "reservation": res("res-3", "ada", "L1", t(3, "11:00"), t(3, "12:00"), "completed", t(3, "11:00"), t(3, "11:20"), 45, 20, 810),
        "charge": chg("usage", "res-3", 20, 45, 10, 810, t(3, "11:20"))})
    c.ok("checkIn", "ada", t(3, "12:00"), {"reservationId": "res-4"}, {"reservation": res(
        "res-4", "ada", "L1", t(3, "12:00"), t(3, "13:00"), "checkedIn", cin=t(3, "12:00"), rate=45)})
    run(c, "sam", t(3, "13:30"), [nsh("res-1", 60, 45, 1350, t(3, "13:30")), nsh("res-6", 60, 60, 1800, t(3, "13:30"))])
    unbilled(c, "ada", t(3, "13:30"), 7160)
    unbilled(c, "eve", t(3, "13:30"), 6800)
    c.ok("listMemberReservations", "ada", t(3, "13:30"), {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "09:00"), t(3, "10:00"), "missed"),
        res("res-2", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "cancelled"),
        res("res-3", "ada", "L1", t(3, "11:00"), t(3, "12:00"), "completed", t(3, "11:00"), t(3, "11:20"), 45, 20, 810),
        res("res-4", "ada", "L1", t(3, "12:00"), t(3, "13:00"), "checkedIn", cin=t(3, "12:00"), rate=45),
        res("res-5", "ada", "L1", t(3, "14:00"), t(3, "15:00"))]})
    c.ok("listMemberReservations", "sam", t(3, "13:30"), {"memberId": "eve"}, {"memberId": "eve", "reservations": [
        res("res-6", "eve", "L2", t(3, "09:00"), t(3, "10:00"), "missed")]})
    c.refuse("checkIn", "ada", t(3, "13:31"), {"reservationId": "res-1"}, "invalid_state")
    c.refuse("cancelReservation", "ada", t(3, "13:31"), {"reservationId": "res-1"}, "invalid_state")
    run(c, "sam", t(3, "15:00"), [nsh("res-5", 60, 45, 1350, t(3, "15:00"))])
    run(c, "sam", t(3, "15:01"), [])
    c.audit_log(t(3, "15:01"))
    return c


def run_5() -> Case:
    c = Case("C3-run-5", "C3-run", "Charge no-shows, Permissions: staff all, admin all, a member is refused and nothing is charged or audited; "
                                    "the audit target is the fixed string noShows, also for a run that charges nothing")
    world(c)
    certify(c, "bob")
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    book(c, "res-2", "bob", "L2", 3, "10:00", "10:15")
    c.refuse("chargeNoShows", "ada", t(3, "11:00"), {}, "forbidden")
    c.refuse("chargeNoShows", "bob", t(3, "11:00"), {}, "forbidden")
    unbilled(c, "ada", t(3, "11:00"), 5000)
    run(c, "amy", t(3, "11:00"), [nsh("res-1", 60, 45, 1350, t(3, "11:00")), nsh("res-2", 15, 60, 450, t(3, "11:00"))])
    run(c, "sue", t(3, "11:01"), [])
    unbilled(c, "bob", t(3, "11:01"), 5450)
    c.audit_log(t(3, "11:01"))
    return c


# ---- C3-lock: a machine that could not have been used --------------------------------------------------------------------

def lock_1() -> Case:
    c = Case("C3-lock-1", "C3-lock", "Lockouts interface, No-show fee: the lockout [appliedAt, releasedAt) overlaps the slot only when appliedAt < end and start < releasedAt; "
                                      "released at the start minute or earlier, no overlap; released a minute into the slot or later, overlap; "
                                      "a reservation locked out is not charged by this run or by a later one")
    world(c)
    for n, day in ((1, 3), (2, 4), (3, 5), (4, 6)):
        book(c, f"res-{n}", "ada", "L1", day, "10:00", "11:00")
    lock(c, "lck-1", "L1", "09:00", "10:00", 3)  # released at the start: no overlap
    lock(c, "lck-2", "L1", "09:00", "10:01", 4)  # released a minute into the slot: overlap
    lock(c, "lck-3", "L1", "09:00", "09:59", 5)  # released before the start: no overlap
    lock(c, "lck-4", "L1", "09:00", "10:30", 6)  # released inside the slot: overlap
    run(c, "sam", t(6, "11:00"), [nsh("res-1", 60, 45, 1350, t(6, "11:00")), nsh("res-3", 60, 45, 1350, t(6, "11:00"))])
    run(c, "sam", t(6, "12:00"), [])
    unbilled(c, "ada", t(6, "12:00"), 7700)
    c.audit_log(t(6, "12:00"))
    return c


def lock_2() -> Case:
    c = Case("C3-lock-2", "C3-lock", "Lockouts interface, No-show fee: a lockout applied a minute before the end of the slot overlaps it; applied at the end minute "
                                      "or later it does not (appliedAt < end)")
    world(c)
    for n, day in ((1, 3), (2, 4), (3, 5)):
        book(c, f"res-{n}", "ada", "L1", day, "10:00", "11:00")
    lock(c, "lck-1", "L1", "10:59", "12:00", 3)
    lock(c, "lck-2", "L1", "11:00", "12:00", 4)
    lock(c, "lck-3", "L1", "11:01", "12:00", 5)
    run(c, "sam", t(5, "12:00"), [nsh("res-2", 60, 45, 1350, t(5, "12:00")), nsh("res-3", 60, 45, 1350, t(5, "12:00"))])
    unbilled(c, "ada", t(5, "12:00"), 7700)
    c.audit_log(t(5, "12:00"))
    return c


def lock_3() -> Case:
    c = Case("C3-lock-3", "C3-lock", "Time, intervals; Lockouts interface: a lockout applied and released one minute apart is the interval [appliedAt, releasedAt) and overlaps by the "
                                      "formula: at the start minute, inside, and ending at the end minute it does; ending at the start minute or applied at the end minute it does not")
    world(c)
    for n, day in ((1, 3), (2, 4), (3, 5), (4, 6), (5, 9)):
        book(c, f"res-{n}", "ada", "L1", day, "10:00", "11:00")
    lock(c, "lck-1", "L1", "10:00", "10:01", 3)
    lock(c, "lck-2", "L1", "10:30", "10:31", 4)
    lock(c, "lck-3", "L1", "10:59", "11:00", 5)
    lock(c, "lck-4", "L1", "09:59", "10:00", 6)
    lock(c, "lck-5", "L1", "11:00", "11:01", 9)
    run(c, "sam", t(9, "12:00"), [nsh("res-4", 60, 45, 1350, t(9, "12:00")), nsh("res-5", 60, 45, 1350, t(9, "12:00"))])
    unbilled(c, "ada", t(9, "12:00"), 7700)
    c.audit_log(t(9, "12:00"))
    return c


def lock_4() -> Case:
    c = Case("C3-lock-4", "C3-lock", "Time, intervals: a lockout applied and released in the same minute is the interval [m, m); by the formula it overlaps a slot that "
                                      "contains the minute strictly inside, and not one that starts or ends at that minute")
    world(c)
    for n, day in ((1, 3), (2, 4), (3, 5)):
        book(c, f"res-{n}", "ada", "L1", day, "10:00", "11:00")
    lock(c, "lck-1", "L1", "10:30", "10:30", 3)
    lock(c, "lck-2", "L1", "10:00", "10:00", 4)
    lock(c, "lck-3", "L1", "11:00", "11:00", 5)
    run(c, "sam", t(5, "12:00"), [nsh("res-2", 60, 45, 1350, t(5, "12:00")), nsh("res-3", 60, 45, 1350, t(5, "12:00"))])
    unbilled(c, "ada", t(5, "12:00"), 7700)
    c.audit_log(t(5, "12:00"))
    return c


def lock_5() -> Case:
    c = Case("C3-lock-5", "C3-lock", "Lockouts interface: a lockout that is not released overlaps every slot that ends after its appliedAt, also a slot after it was applied; "
                                      "once released, a later slot is free, and the slot it overlapped stays uncharged for every later run")
    world(c)
    for n, day in ((1, 3), (2, 4), (3, 5)):
        book(c, f"res-{n}", "ada", "L1", day, "10:00", "11:00")
    c.ok("applyLockout", "sam", t(3, "11:00"), {"machineId": "L1", "reason": "belt"}, new="lck-1")
    run(c, "sam", t(4, "12:00"), [nsh("res-1", 60, 45, 1350, t(4, "12:00"))])
    c.ok("releaseLockout", "sam", t(5, "09:00"), {"lockoutId": "lck-1"})
    run(c, "sam", t(5, "11:00"), [nsh("res-3", 60, 45, 1350, t(5, "11:00"))])
    run(c, "sam", t(5, "12:00"), [])
    unbilled(c, "ada", t(5, "12:00"), 7700)
    c.ok("listMemberReservations", "ada", t(5, "12:00"), {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "missed"),
        res("res-2", "ada", "L1", t(4, "10:00"), t(4, "11:00"), "missed"),
        res("res-3", "ada", "L1", t(5, "10:00"), t(5, "11:00"), "missed")]})
    c.audit_log(t(5, "12:00"))
    return c


def lock_6() -> Case:
    c = Case("C3-lock-6", "C3-lock", "Lockouts interface, Lockouts: a lockout of any source counts: a failed inspection and an open work order lock the machine and stop the fee; "
                                      "a lockout of another machine does not")
    world(c)
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 30})
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    book(c, "res-2", "ada", "L1", 4, "10:00", "11:00")
    book(c, "res-3", "ada", "L1", 5, "10:00", "11:00")
    book(c, "res-4", "ada", "L2", 6, "10:00", "11:00")
    book(c, "res-5", "ada", "L2", 9, "10:00", "11:00")
    c.ok("recordInspection", "sam", t(3, "10:30"), {"machineId": "L1", "outcome": "failed", "note": "crack"})
    c.ok("recordInspection", "sam", t(3, "11:30"), {"machineId": "L1", "outcome": "passed", "note": "fixed"})
    c.ok("openWorkOrder", "sam", t(4, "10:30"), {"machineId": "L1", "summary": "Replace tube"}, new="wko-1")
    c.ok("closeWorkOrder", "sam", t(4, "11:30"), {"workOrderId": "wko-1", "note": "done"})
    lock(c, "lck-3", "L2", "10:30", "10:45", 5)  # another machine: res-3 on L1 is free
    lock(c, "lck-4", "L2", "10:30", "10:45", 6)  # res-4 is on L2
    lock(c, "lck-5", "L1", "10:30", "10:45", 9)  # another machine: res-5 on L2 is free
    run(c, "sam", t(9, "12:00"), [nsh("res-3", 60, 45, 1350, t(9, "12:00")), nsh("res-5", 60, 60, 1800, t(9, "12:00"))])
    unbilled(c, "ada", t(9, "12:00"), 8150)
    c.audit_log(t(9, "12:00"))
    return c


# ---- C3-esc: an escalation and the fee ----------------------------------------------------------------------------------------

def esc_1() -> Case:
    c = Case("C3-esc-1", "C3-esc", "Escalation effects, No-show fee: a reservation the escalation cancels is cancelled and not missed; one it leaves because it has started "
                                    "goes missed but its machine was locked out by the work order, so no fee, also after the work order is closed; another machine is charged")
    world(c)
    certify(c, "eve")
    certify(c, "bob")
    book(c, "res-1", "ada", "L1", 3, "09:00", "10:00")
    book(c, "res-2", "eve", "L1", 3, "11:00", "12:00")
    book(c, "res-3", "bob", "L2", 3, "09:00", "10:00")
    c.ok("setEscalationPolicy", "amy", W, {"familyId": "laser", "minSeverity": "medium"})
    n = t(3, "09:30")
    c.ok("reportIncident", "sam", n, {"machineId": "L1", "severity": "medium", "description": "fire"}, new="inc-1")
    c.ok("escalateIncident", "sue", n, {"incidentId": "inc-1"}, {"escalation": {
        "escalationId": "esc-1", "incidentId": "inc-1", "machineId": "L1", "workOrderId": "wko-1",
        "cancelledReservationIds": ["res-2"], "at": n}}, new="esc-1")
    run(c, "sam", t(3, "13:00"), [nsh("res-3", 60, 60, 1800, t(3, "13:00"))])
    c.ok("listMemberReservations", "sam", t(3, "13:00"), {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "09:00"), t(3, "10:00"), "missed")]})
    c.ok("listMemberReservations", "sam", t(3, "13:00"), {"memberId": "eve"}, {"memberId": "eve", "reservations": [
        res("res-2", "eve", "L1", t(3, "11:00"), t(3, "12:00"), "cancelled")]})
    c.ok("resolveIncident", "sam", t(3, "13:10"), {"incidentId": "inc-1", "resolution": "fixed"})
    c.ok("closeWorkOrder", "sue", t(3, "13:20"), {"workOrderId": "wko-1", "note": "done"})
    run(c, "sam", t(3, "13:30"), [])
    unbilled(c, "ada", t(3, "13:30"), 5000)
    unbilled(c, "eve", t(3, "13:30"), 5000)
    unbilled(c, "bob", t(3, "13:30"), 6800)
    c.audit_log(t(3, "13:30"))
    return c


# ---- C3-bill: the charge on invoices, in the ledger and in the reports ----------------------------------------------------------

def bill_1() -> Case:
    c = Case("C3-bill-1", "C3-bill", "Charges and invoices, Postings, Get account: the charge is unbilled until the invoice of the month of its at, which is the month of the run "
                                      "whatever month the slot was in; the invoice line is kind noShow after the membership line, posted debit receivable credit revenueNoShows "
                                      "with the invoice as memo; the trial balance has 24 rows")
    world(c)
    certify(c, "eve")
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    book(c, "res-2", "eve", "L2", 30, "10:00", "10:15")
    m1, d1 = t(1, "00:00", 4), t(15, "00:00", 4)
    c.ok("generateInvoices", "sam", m1, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", m1, d1, [chg("membership", "plus", 1, 4900, 0, 4900, m1), chg("course", "laser-101", 1, 5000, 0, 5000, W)], 9900),
        invoice("inv-2", "bob", "2026-03", m1, d1, [chg("membership", "basic", 1, 2500, 0, 2500, m1)], 2500),
        invoice("inv-3", "eve", "2026-03", m1, d1, [chg("membership", "pro", 1, 9900, 0, 9900, m1), chg("course", "laser-101", 1, 5000, 0, 5000, W)], 14900)]})
    r = t(2, "10:00", 4)
    run(c, "sam", r, [nsh("res-1", 60, 45, 1350, r), nsh("res-2", 15, 60, 450, r)])
    c.ok("getAccount", "ada", r, {"memberId": "ada"}, account("ada", 0, 9900, 0, 1350))
    c.ok("getAccount", "eve", r, {"memberId": "eve"}, account("eve", 0, 14900, 0, 450))
    m2, d2 = t(1, "00:00", 5), t(15, "00:00", 5)
    c.ok("generateInvoices", "sam", m2, {"month": "2026-04"}, {"month": "2026-04", "invoices": [
        invoice("inv-4", "ada", "2026-04", m2, d2, [chg("membership", "plus", 1, 4900, 0, 4900, m2), nsh("res-1", 60, 45, 1350, r)], 6250),
        invoice("inv-5", "bob", "2026-04", m2, d2, [chg("membership", "basic", 1, 2500, 0, 2500, m2)], 2500),
        invoice("inv-6", "eve", "2026-04", m2, d2, [chg("membership", "pro", 1, 9900, 0, 9900, m2), nsh("res-2", 15, 60, 450, r)], 10350)]})
    c.ok("getAccount", "ada", m2, {"memberId": "ada"}, account("ada", 0, 16150, 9900, 0))
    c.ok("listJournalEntries", "amy", m2, {"month": "2026-05"}, {"month": "2026-05", "entries": [
        je(6, m2, "receivable", "revenueMembership", 4900, "inv-4"), je(7, m2, "receivable", "revenueNoShows", 1350, "inv-4"),
        je(8, m2, "receivable", "revenueMembership", 2500, "inv-5"),
        je(9, m2, "receivable", "revenueMembership", 9900, "inv-6"), je(10, m2, "receivable", "revenueNoShows", 450, "inv-6")]})
    c.ok("getTrialBalance", "amy", m2, {}, tb({"receivable": (46400, 0), "revenueCourse": (0, 10000), "revenueMembership": (0, 34600),
                                              "revenueNoShows": (0, 1800)}, 46400))
    c.ok("chargesByKind", "sam", m2, {"month": "2026-04"}, {"month": "2026-04", "rows": kinds({"membership": 17300, "noShow": 1800}, None, 19100), "totalCents": 19100})
    c.ok("chargesByKind", "sam", m2, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds({"membership": 17300, "course": 10000}, None, 27300), "totalCents": 27300})
    c.audit_log(m2)
    return c


def bill_2() -> Case:
    c = Case("C3-bill-2", "C3-bill", "Revenue report, Finance summary, Charges by kind, Issue refund: a no-show line is in chargesByKind (11 rows, noShow last) and in the finance summary's "
                                      "invoicedCents, left out of the revenue report's four kind sums and invoicedCents but counted in its outstandingCents, and refundable like any line")
    world(c)
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    run(c, "sam", t(3, "12:00"), [nsh("res-1", 60, 45, 1350, t(3, "12:00"))])
    m1, d1 = t(1, "00:00", 4), t(15, "00:00", 4)
    c.ok("chargesByKind", "sam", t(3, "12:00"), {"month": "2026-03"}, {"month": "2026-03", "rows": kinds({}, None, 0), "totalCents": 0})
    c.ok("generateInvoices", "sam", m1, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", m1, d1, [chg("membership", "plus", 1, 4900, 0, 4900, m1), chg("course", "laser-101", 1, 5000, 0, 5000, W),
                                                     nsh("res-1", 60, 45, 1350, t(3, "12:00"))], 11250),
        invoice("inv-2", "bob", "2026-03", m1, d1, [chg("membership", "basic", 1, 2500, 0, 2500, m1)], 2500),
        invoice("inv-3", "eve", "2026-03", m1, d1, [chg("membership", "pro", 1, 9900, 0, 9900, m1)], 9900)]})
    c.ok("revenueReport", "sam", m1, {"month": "2026-03"}, {
        "month": "2026-03", "invoicesGenerated": True, "invoiceCount": 3, "membershipCents": 17300, "usageCents": 0, "materialCents": 0,
        "courseCents": 5000, "invoicedCents": 22300, "creditAppliedCents": 0, "outstandingCents": 23650, "paymentsCents": 0, "creditsIssuedCents": 0})
    c.ok("chargesByKind", "sam", m1, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds(
        {"membership": 17300, "course": 5000, "noShow": 1350}, None, 23650), "totalCents": 23650})
    c.ok("financeSummary", "amy", m1, {"month": "2026-03"}, {
        "month": "2026-03", "invoicedCents": 23650, "refundedCents": 0, "paymentsCents": 0, "creditsIssuedCents": 0, "journalEntries": 0, "trialBalanced": True})
    p = t(2, "09:00", 4)
    c.ok("recordPayment", "sam", p, {"invoiceId": "inv-1", "amountCents": 11250}, None, new="pay-1")
    r = t(2, "10:00", 4)
    c.ok("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 3, "amountCents": 1350, "reason": "waived"}, {
        "refund": {"refundId": "rfd-1", "invoiceId": "inv-1", "lineNumber": 3, "amountCents": 1350, "reason": "waived", "at": r},
        "creditBalanceCents": 1350}, new="rfd-1")
    c.refuse("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 3, "amountCents": 1, "reason": "again"}, "refund_too_large")
    c.ok("chargesByKind", "sam", r, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds(
        {"membership": 17300, "course": 5000, "noShow": 1350}, {"noShow": 1350}, 23650), "totalCents": 23650})
    c.ok("financeSummary", "amy", r, {"month": "2026-03"}, {
        "month": "2026-03", "invoicedCents": 23650, "refundedCents": 1350, "paymentsCents": 0, "creditsIssuedCents": 0, "journalEntries": 0, "trialBalanced": True})
    c.ok("listJournalEntries", "amy", r, {"month": "2026-04"}, {"month": "2026-04", "entries": [
        je(1, m1, "receivable", "revenueMembership", 4900, "inv-1"), je(2, m1, "receivable", "revenueCourse", 5000, "inv-1"),
        je(3, m1, "receivable", "revenueNoShows", 1350, "inv-1"),
        je(4, m1, "receivable", "revenueMembership", 2500, "inv-2"), je(5, m1, "receivable", "revenueMembership", 9900, "inv-3"),
        je(6, p, "cash", "receivable", 11250, "pay-1"), je(7, r, "revenueRefunds", "creditLiability", 1350, "rfd-1")]})
    c.ok("getAccount", "ada", r, {"memberId": "ada"}, account("ada", 1350, 0, 0, 0))
    c.audit_log(r)
    return c


def ledger_1() -> Case:
    c = Case("C3-ledger-1", "C3-ledger", "Ledger account, Get trial balance, Post adjustment: the trial balance has 24 rows, revenueNoShows between revenueMembership and "
                                          "revenuePasses, also without entries; post adjustment accepts the account on either side and refuses it on both")
    world(c)
    c.ok("getTrialBalance", "amy", W, {}, tb({}, 0))
    a = t(3, "09:00")
    c.ok("postAdjustment", "amy", a, {"debitAccount": "revenueNoShows", "creditAccount": "cash", "amountCents": 700, "memo": "a"},
         {"entry": je(1, a, "revenueNoShows", "cash", 700, "a")}, new="jrn-1")
    c.ok("postAdjustment", "amy", a, {"debitAccount": "cash", "creditAccount": "revenueNoShows", "amountCents": 200, "memo": "b"},
         {"entry": je(2, a, "cash", "revenueNoShows", 200, "b")}, new="jrn-2")
    c.refuse("postAdjustment", "amy", a, {"debitAccount": "revenueNoShows", "creditAccount": "revenueNoShows", "amountCents": 5, "memo": "c"}, "same_account")
    c.ok("getTrialBalance", "amy", a, {}, tb({"cash": (200, 700), "revenueNoShows": (700, 200)}, 900))
    c.ok("listJournalEntries", "amy", a, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        je(1, a, "revenueNoShows", "cash", 700, "a"), je(2, a, "cash", "revenueNoShows", 200, "b")]})
    c.audit_log(a)
    return c


# ---- C3-keep: what the fee does not touch -------------------------------------------------------------------------------------------

def keep_1() -> Case:
    c = Case("C3-keep-1", "C3-keep", "Pass draw, Consumption: the run draws no pass minutes and no stock, because both happen only at check-out; it creates the noShow charge only")
    world(c)
    c.ok("addPassProduct", "amy", W, {"productId": "p20", "name": "Twenty", "minutes": 20, "priceCents": 1000, "validDays": 30})
    c.ok("sellPass", "ada", W, {"memberId": "ada", "productId": "p20"}, {
        "pass": {"passId": "pas-1", "memberId": "ada", "productId": "p20", "minutesLeft": 20, "issuedOn": "2026-03-02", "expiresOn": "2026-04-01"},
        "charge": chg("pass", "pas-1", 1, 1000, 0, 1000, W)}, new="pas-1")
    c.ok("setConsumption", "sam", W, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 6})
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    run(c, "sam", t(3, "12:00"), [nsh("res-1", 60, 45, 1350, t(3, "12:00"))])
    c.ok("listMemberPasses", "ada", t(3, "12:00"), {"memberId": "ada"}, {"memberId": "ada", "passes": [
        {"passId": "pas-1", "memberId": "ada", "productId": "p20", "minutesLeft": 20, "issuedOn": "2026-03-02", "expiresOn": "2026-04-01"}]})
    c.ok("listMaterials", "ada", t(3, "12:00"), {}, {"materials": [{"materialId": "ply", "name": "Plywood", "unitPriceCents": 150, "stock": 100}]})
    unbilled(c, "ada", t(3, "12:00"), 7350)
    c.audit_log(t(3, "12:00"))
    return c


def keep_2() -> Case:
    c = Case("C3-keep-2", "C3-keep", "Project cost: an attached missed reservation adds nothing, before and after the run charges it; only the completed reservation counts")
    world(c)
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    book(c, "res-2", "ada", "L1", 3, "11:00", "12:00")
    c.ok("createProject", "ada", W, {"projectId": "pj-1", "ownerId": "ada", "title": "Sign"})
    c.ok("attachReservation", "ada", W, {"projectId": "pj-1", "reservationId": "res-1"})
    c.ok("attachReservation", "ada", W, {"projectId": "pj-1", "reservationId": "res-2"})
    c.ok("checkIn", "ada", t(3, "11:00"), {"reservationId": "res-2"})
    c.ok("checkOut", "ada", t(3, "11:20"), {"reservationId": "res-2"})
    cost = {"projectId": "pj-1", "usageCents": 810, "materialCents": 0, "totalCents": 810,
            "byMember": [{"memberId": "ada", "usageCents": 810, "materialCents": 0}]}
    c.ok("getProjectCost", "ada", t(3, "11:30"), {"projectId": "pj-1"}, cost)
    run(c, "sam", t(3, "12:00"), [nsh("res-1", 60, 45, 1350, t(3, "12:00"))])
    c.ok("getProjectCost", "ada", t(3, "12:00"), {"projectId": "pj-1"}, cost)
    c.audit_log(t(3, "12:00"))
    return c


def build_cases() -> list[dict[str, Any]]:
    cases = [run_1(), run_2(), run_3(), run_4(), run_5(), lock_1(), lock_2(), lock_3(), lock_4(), lock_5(), lock_6(), esc_1(),
             bill_1(), bill_2(), ledger_1(), keep_1(), keep_2()]
    return [c.build() for c in cases]
