"""Change step C4 of the large Makerspace: pausing a membership.

`setMemberStanding` also accepts `paused`. A paused member is not in good standing: every operation that refuses a member
who is not in good standing answers `member_suspended` for a paused member, at its place in the operation's order, and
what a suspended member can still do, a paused member can do. The monthly fee of a paused member is 500 cents whatever the
tier, judged at the request time of `generateInvoices` whatever month is invoiced; a household discount applies to the 500
cents and a paused member counts as a member of the household. A suspended member keeps paying the tier's fee, and a dunning
run's suspension step turns a paused member into a suspended one. Written from the requirements document after the change,
with every expected value stated by hand.

What this step retires or moves: nothing. No earlier case sends the value `paused`, and none depends on a paused member
being refused as invalid; a suspended member behaves as before.

How the cases are cut: the 14 operations that refuse a member who is not in good standing (issue PPE, take deposit, sell
pass, redeem promotion, rent locker, issue guest pass, enroll in course, purchase material, check in, create reservation,
enter door, join waitlist, sell ticket, start job) have one case each (`C4-gate-*`), and each pauses the member where the
refusal would otherwise be a different one or none, so the refusal's place in the order shows. The 13 operations that do not
check standing (cancel reservation, check out, record payment, sign waiver, declare minor, return PPE, leave door, admit
guest, issue badge, log volunteer time, complete course, complete job, create project) have one case each too (`C4-open-*`).
The fee, its discount, its postings and the dunning rule follow (`C4-fee-*`, `C4-dun-1`), then the two reads and the waitlist
promotion (`C4-read-*`, `C4-promote-1`).

Dates the cases rely on: 2026-03-02 is a Monday (W, the world's time), so 03-03 is a Tuesday, 03-10 Tuesday, 03-31 Tuesday.
Invoices of March are made on 2026-04-01T08:00Z (I) and are due 2026-04-15T08:00Z (D). The world's rates: L1 45, L2 60, T1
30 cents a minute. Tiers: ada plus (fee 4900, 10 %), bob basic (2500, 0 %), eve pro (9900, 25 %). Joining the laser course
costs 5000 cents (a `course` charge at W), so ada's invoice of March carries it.
"""
from __future__ import annotations

from typing import Any

from kit import Case, world

W = "2026-03-02T09:00Z"  # Monday, the world's time
I = "2026-04-01T08:00Z"  # invoices of March
D = "2026-04-15T08:00Z"  # and their due time

RETIRED: dict[str, str] = {}

ACCOUNTS = ["cash", "creditLiability", "depositLiability", "expenseGoodwill", "expenseMaker", "expensePayout",
            "expenseVolunteer", "expenseWriteOff", "inventory", "payable", "receivable", "revenueCourse", "revenueDeposits",
            "revenueEvents", "revenueGuests", "revenueJobs", "revenueLateFees", "revenueLockers", "revenueMaterial",
            "revenueMembership", "revenueNoShows", "revenuePasses", "revenueRefunds", "revenueUsage"]
KINDS = ["membership", "usage", "material", "course", "deposit", "pass", "locker", "guest", "lateFee", "event", "noShow"]
NAMES = {"ada": ("Ada Byron", "plus"), "bob": ("Bob Stone", "basic"), "eve": ("Eve Marsh", "pro")}


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


# ---- record builders (the members of each record, from the requirements) ---------------------------------------------

def mrec(mem: str, standing: str, tier: str | None = None) -> dict[str, Any]:
    name, base = NAMES[mem]
    return {"member": {"memberId": mem, "name": name, "tier": tier or base, "standing": standing, "registeredAt": W}}


def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def chg(kind, ref_, qty, unit, disc, amt, at):
    assert (qty * unit * (100 - disc) + 50) // 100 == amt, (kind, ref_, qty, unit, disc, amt)
    return {"kind": kind, "ref": ref_, "quantity": qty, "unitPriceCents": unit, "discountPercent": disc,
            "amountCents": amt, "at": at}


def mline(tier, unit, disc, amt, at):
    return chg("membership", tier, 1, unit, disc, amt, at)


def course_line():
    return chg("course", "laser-101", 1, 5000, 0, 5000, W)


def invoice(iid, mem, month, issued, due, lines, credit=0, paid=0):
    total = sum(line["amountCents"] for line in lines)
    balance = total - credit - paid
    return {"invoiceId": iid, "memberId": mem, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": credit, "paidCents": paid, "balanceCents": balance,
            "status": "paid" if balance == 0 else "open", "overdue": False}


def account(mem, credit=0, open_=0, overdue=0, unbilled=0):
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


def kinds(invoiced: dict, total: int) -> list:
    assert set(invoiced) <= set(KINDS)
    rows = [{"kind": k, "invoicedCents": invoiced.get(k, 0), "refundedCents": 0} for k in KINDS]
    assert sum(r["invoicedCents"] for r in rows) == total
    return rows


def pay(pid, inv_id, mem, amount, at):
    return {"paymentId": pid, "invoiceId": inv_id, "memberId": mem, "amountCents": amount, "appliedCents": amount,
            "creditedCents": 0, "at": at}


def deposit(did, mem, purpose, amount, taken):
    return {"depositId": did, "memberId": mem, "purpose": purpose, "amountCents": amount, "status": "held",
            "takenAt": taken, "closedAt": None}


def passrec(pid, mem, product, left, issued, expires):
    return {"passId": pid, "memberId": mem, "productId": product, "minutesLeft": left, "issuedOn": issued, "expiresOn": expires}


def promo(code, percent, scope, maximum, until, n, ended=False):
    return {"code": code, "percent": percent, "scope": scope, "maxRedemptions": maximum, "validUntil": until,
            "redemptions": n, "ended": ended}


def redemption(code, mem, percent, at):
    return {"code": code, "memberId": mem, "percent": percent, "redeemedAt": at, "usedAt": None, "usedOn": None}


def rental(rid, locker, mem, status, started, ended=None):
    return {"rentalId": rid, "lockerId": locker, "memberId": mem, "status": status, "startedAt": started, "endedAt": ended}


def gpass(gid, mem, guest, date, charged=False, status="issued"):
    return {"guestPassId": gid, "memberId": mem, "guestName": guest, "date": date, "charged": charged, "status": status}


def enr(course, mem, at, done=None):
    return {"courseId": course, "memberId": mem, "status": "enrolled" if done is None else "completed",
            "enrolledAt": at, "completedAt": done}


def crs(course, family, title, fee, cap, n):
    return {"courseId": course, "familyId": family, "title": title, "feeCents": fee, "capacity": cap, "enrolledCount": n}


def cert(mem, family, issued, expires):
    return {"memberId": mem, "familyId": family, "issuedOn": issued, "expiresOn": expires, "status": "valid"}


def waiting(course, mem, at):
    return {"courseId": course, "memberId": mem, "joinedAt": at}


def skipped(mem, code):
    return {"memberId": mem, "code": code}


def mat(mid, name, price, stock):
    return {"materialId": mid, "name": name, "unitPriceCents": price, "stock": stock}


def bought(pid, mem, mid, qty, price, amount, at, name, stock):
    return {"purchase": {"purchaseId": pid, "memberId": mem, "materialId": mid, "quantity": qty, "unitPriceCents": price,
                         "discountPercent": 0, "amountCents": amount, "at": at},
            "material": mat(mid, name, price, stock)}


def event(eid, title, room, start, end, cap, price, sold, status="open"):
    return {"eventId": eid, "title": title, "roomId": room, "start": start, "end": end, "capacity": cap, "priceCents": price,
            "status": status, "sold": sold}


def ticket(tid, eid, mem, status="valid"):
    return {"ticketId": tid, "eventId": eid, "memberId": mem, "status": status}


def ppe(iid, name, stock):
    return {"itemId": iid, "name": name, "stock": stock}


def badge(bid, mem, status="active", inside=False):
    return {"badgeId": bid, "memberId": mem, "status": status, "inside": inside}


def door(eid, kind, bid, at, pass_id=None):
    return {"eventId": eid, "kind": kind, "badgeId": bid, "guestPassId": pass_id, "at": at}


def job(jid, qid, customer, maker, status, total, credit, paid):
    return {"jobId": jid, "quoteId": qid, "customerId": customer, "makerId": maker, "status": status,
            "totalCents": total, "makerCreditCents": credit, "paidCents": paid}


def project(pid, owner, title):
    return {"projectId": pid, "ownerId": owner, "title": title, "collaboratorIds": [], "status": "open",
            "reservationIds": [], "purchaseIds": []}


def fam(fid, certified, consent, waiver, ppe_held):
    return {"familyId": fid, "certified": certified, "consent": consent, "waiverSigned": waiver, "ppeHeld": ppe_held}


def elig(mem, mach, date, blockers):
    return {"memberId": mem, "machineId": mach, "date": date, "eligible": not blockers, "blockers": blockers}


def notice(n, mem, kind, inv_id, at):
    subject = {"dunning": f"Invoice {inv_id} is overdue", "suspension": f"Membership suspended: invoice {inv_id} is overdue"}[kind]
    return {"noticeId": f"ntc-{n}", "memberId": mem, "kind": kind, "subject": subject, "ref": inv_id, "at": at,
            "acknowledgedAt": None}


# ---- steps used by many cases -------------------------------------------------------------------------------------------

def rf(c: Case, op: str, actor: str, at: str, args: dict, code: str) -> None:
    """A refusal; it goes through the request-time check too, so it may not go back in time."""
    if c.last_ok is not None and at < c.last_ok:
        raise ValueError(f"{c.id}: refusal at {at} goes back from {c.last_ok}")
    c.refuse(op, actor, at, args, code)


def stand(c: Case, mem: str, standing: str, at: str, tier: str | None = None, actor: str = "sam") -> None:
    """Set the standing and compare the member that comes back."""
    c.ok("setMemberStanding", actor, at, {"memberId": mem, "standing": standing}, mrec(mem, standing, tier))


def reg(c: Case, mem: str, name: str, tier: str) -> None:
    NAMES[mem] = (name, tier)
    c.ok("registerMember", "sam", W, {"memberId": mem, "name": name, "tier": tier}, mrec(mem, "good"))


def certify(c: Case, mem: str, at: str = W) -> None:
    c.ok("enrollInCourse", "sam", at, {"courseId": "laser-101", "memberId": mem})
    c.ok("completeCourse", "sam", at, {"courseId": "laser-101", "memberId": mem})


def book(c: Case, rid: str, mem: str, mach: str, day: int, start: str, end: str, at: str = W) -> None:
    s, e = t(day, start), t(day, end)
    c.ok("createReservation", mem, at, {"memberId": mem, "machineId": mach, "start": s, "end": e},
         {"reservation": res(rid, mem, mach, s, e)}, new=rid)


def make_quote(c: Case, n: int, customer: str, title: str, lines: list, decision: str | None = "accepted") -> None:
    c.ok("createQuote", "sam", W, {"customerId": customer, "title": title}, new=f"quo-{n}")
    for kind, ref_, qty in lines:
        c.ok("addQuoteLine", "sam", W, {"quoteId": f"quo-{n}", "lineKind": kind, "ref": ref_, "quantity": qty})
    if decision:
        c.ok("sendQuote", "sam", W, {"quoteId": f"quo-{n}"})
        c.ok("decideQuote", "sam", W, {"quoteId": f"quo-{n}", "decision": decision})


def unbilled(c: Case, mem: str, at: str, cents: int, actor: str = "sam") -> None:
    c.ok("getAccount", actor, at, {"memberId": mem}, account(mem, unbilled=cents))


# ---- C4-standing: the value, and what pausing does not do ---------------------------------------------------------------

def standing_1() -> Case:
    c = Case("C4-standing-1", "C4-standing", "Set member standing, Member: a third value `paused`, set from any standing to any, the current one succeeds without a "
                                              "change (and is audited), every other value stays invalid_argument; the member record carries it")
    world(c)
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    c.ok("getMember", "bob", a, {"memberId": "bob"}, mrec("bob", "paused"))
    c.ok("getMember", "sam", a, {"memberId": "bob"}, mrec("bob", "paused"))
    stand(c, "bob", "paused", a)
    stand(c, "bob", "suspended", a, actor="amy")
    stand(c, "bob", "paused", a, actor="amy")
    stand(c, "bob", "good", a)
    stand(c, "bob", "paused", a)
    stand(c, "bob", "suspended", a)
    stand(c, "bob", "suspended", a)
    stand(c, "bob", "paused", a)
    for bad in ("Paused", "PAUSED", "pause", " paused", "paused ", "away", ""):
        c.refuse("setMemberStanding", "sam", a, {"memberId": "bob", "standing": bad}, "invalid_argument")
    c.refuse("setMemberStanding", "sam", a, {"memberId": "bob", "standing": None}, "invalid_argument")
    c.refuse("setMemberStanding", "sam", a, {"memberId": "bob", "standing": 5}, "invalid_argument")
    c.refuse("setMemberStanding", "bob", a, {"memberId": "bob", "standing": "good"}, "forbidden")
    c.refuse("setMemberStanding", "ada", a, {"memberId": "bob", "standing": "paused"}, "forbidden")
    c.refuse("setMemberStanding", "sam", a, {"memberId": "ghost", "standing": "paused"}, "not_found")
    c.refuse("setMemberStanding", "sam", a, {"memberId": "sue", "standing": "paused"}, "not_found")
    c.ok("getMember", "bob", a, {"memberId": "bob"}, mrec("bob", "paused"))
    c.ok("getMember", "sam", a, {"memberId": "ada"}, mrec("ada", "good"))
    c.audit_log(a)
    return c


def standing_2() -> Case:
    c = Case("C4-standing-2", "C4-standing", "Set member standing: pausing cancels nothing (a reservation, an enrollment, a ticket, a rental and a deposit stay, with their "
                                              "charges); change tier on a paused member keeps the standing in the member it returns")
    world(c)
    certify(c, "bob")
    c.ok("addCourse", "sam", W, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe basics", "feeCents": 3000, "capacity": 4})
    c.ok("enrollInCourse", "bob", W, {"courseId": "lathe-101", "memberId": "bob"})
    book(c, "res-1", "bob", "L1", 3, "10:00", "11:00")
    c.ok("addRoom", "amy", W, {"roomId": "lab", "name": "Lab", "capacity": 10})
    c.ok("createEvent", "sam", W, {"eventId": "fair", "title": "Fair", "roomId": "lab", "start": t(10, "10:00"), "end": t(10, "12:00"),
                                   "capacity": 5, "priceCents": 1500})
    c.ok("sellTicket", "bob", W, {"eventId": "fair", "memberId": "bob"}, None, new="tkt-1")
    c.ok("addLocker", "amy", W, {"lockerId": "K1", "monthlyCents": 1500})
    c.ok("rentLocker", "bob", W, {"memberId": "bob", "lockerId": "K1"}, None, new="rnt-1")
    c.ok("takeDeposit", "sam", W, {"memberId": "bob", "purpose": "chair", "amountCents": 500}, None, new="dep-1")
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    c.ok("listMemberReservations", "bob", a, {"memberId": "bob"}, {"memberId": "bob", "reservations": [res("res-1", "bob", "L1", t(3, "10:00"), t(3, "11:00"))]})
    c.ok("listEventTickets", "sam", a, {"eventId": "fair"}, {"eventId": "fair", "tickets": [ticket("tkt-1", "fair", "bob")]})
    c.ok("listDeposits", "bob", a, {"memberId": "bob"}, {"memberId": "bob", "deposits": [deposit("dep-1", "bob", "chair", 500, W)]})
    unbilled(c, "bob", a, 5000 + 3000 + 1500 + 500)  # the laser course of the certification, the lathe course, the ticket, the deposit
    c.ok("completeCourse", "sam", a, {"courseId": "lathe-101", "memberId": "bob"},
         {"enrollment": enr("lathe-101", "bob", W, a), "certification": cert("bob", "lathe", "2026-03-02", "2027-03-02")})
    c.ok("endLockerRental", "bob", a, {"rentalId": "rnt-1"}, {"rental": rental("rnt-1", "K1", "bob", "ended", W, a)})
    c.ok("changeTier", "sam", a, {"memberId": "bob", "tier": "pro"}, mrec("bob", "paused", "pro"))
    c.ok("getMember", "bob", a, {"memberId": "bob"}, mrec("bob", "paused", "pro"))
    c.audit_log(a)
    return c


# ---- C4-gate: the 14 operations that refuse a member who is not in good standing ---------------------------------------

def gate_ppe() -> Case:
    c = Case("C4-gate-ppe", "C4-gate", "Issue PPE, Membership: a paused member is refused member_suspended, before already_issued and before out_of_stock; nothing is issued")
    world(c)
    c.ok("addPpeItem", "sam", W, {"itemId": "gloves", "name": "Gloves", "stock": 1}, {"item": ppe("gloves", "Gloves", 1)})
    c.ok("addPpeItem", "sam", W, {"itemId": "goggles", "name": "Goggles", "stock": 0}, {"item": ppe("goggles", "Goggles", 0)})
    c.ok("issuePpe", "sam", W, {"memberId": "ada", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 0)})
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    stand(c, "bob", "paused", a)
    rf(c, "issuePpe", "sam", a, {"memberId": "bob", "itemId": "gloves"}, "member_suspended")  # the stock is 0
    rf(c, "issuePpe", "amy", a, {"memberId": "bob", "itemId": "goggles"}, "member_suspended")  # the stock is 0
    rf(c, "issuePpe", "sam", a, {"memberId": "ada", "itemId": "gloves"}, "member_suspended")  # she holds them
    c.ok("listMemberPpe", "ada", a, {"memberId": "ada"}, {"memberId": "ada", "items": [ppe("gloves", "Gloves", 0)]})
    c.ok("listMemberPpe", "bob", a, {"memberId": "bob"}, {"memberId": "bob", "items": []})
    stand(c, "bob", "good", a)
    rf(c, "issuePpe", "sam", a, {"memberId": "bob", "itemId": "gloves"}, "out_of_stock")
    stand(c, "ada", "good", a)
    rf(c, "issuePpe", "sam", a, {"memberId": "ada", "itemId": "gloves"}, "already_issued")
    c.audit_log(a)
    return c


def gate_deposit() -> Case:
    c = Case("C4-gate-deposit", "C4-gate", "Take deposit, Membership: a paused member is refused member_suspended; no deposit and no charge exist, no number is used")
    world(c)
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    rf(c, "takeDeposit", "sam", a, {"memberId": "bob", "purpose": "chair", "amountCents": 500}, "member_suspended")
    rf(c, "takeDeposit", "amy", a, {"memberId": "bob", "purpose": "chair", "amountCents": 500}, "member_suspended")
    c.ok("listDeposits", "bob", a, {"memberId": "bob"}, {"memberId": "bob", "deposits": []})
    unbilled(c, "bob", a, 0)
    b = t(2, "11:00")
    stand(c, "bob", "good", b)
    c.ok("takeDeposit", "sam", b, {"memberId": "bob", "purpose": "chair", "amountCents": 500},
         {"deposit": deposit("dep-1", "bob", "chair", 500, b), "charge": chg("deposit", "dep-1", 1, 500, 0, 500, b)}, new="dep-1")
    unbilled(c, "bob", b, 500)
    c.audit_log(b)
    return c


def gate_pass() -> Case:
    c = Case("C4-gate-pass", "C4-gate", "Sell pass, Membership: a paused member is refused member_suspended, whoever acts; no pass and no charge exist, no number is used")
    world(c)
    c.ok("addPassProduct", "amy", W, {"productId": "p60", "name": "Sixty", "minutes": 60, "priceCents": 3000, "validDays": 30})
    a = t(2, "10:00")
    stand(c, "eve", "paused", a)
    rf(c, "sellPass", "eve", a, {"memberId": "eve", "productId": "p60"}, "member_suspended")
    rf(c, "sellPass", "sam", a, {"memberId": "eve", "productId": "p60"}, "member_suspended")
    rf(c, "sellPass", "amy", a, {"memberId": "eve", "productId": "p60"}, "member_suspended")
    c.ok("listMemberPasses", "eve", a, {"memberId": "eve"}, {"memberId": "eve", "passes": []})
    unbilled(c, "eve", a, 0)
    b = t(2, "11:00")
    stand(c, "eve", "good", b)
    c.ok("sellPass", "eve", b, {"memberId": "eve", "productId": "p60"},
         {"pass": passrec("pas-1", "eve", "p60", 60, "2026-03-02", "2026-04-01"), "charge": chg("pass", "pas-1", 1, 3000, 0, 3000, b)}, new="pas-1")
    c.audit_log(b)
    return c


def gate_promotion() -> Case:
    c = Case("C4-gate-promotion", "C4-gate", "Redeem promotion, Membership: a paused member is refused member_suspended, before promotion_ended and before promotion_exhausted; "
                                             "the refusals redeem nothing and the promotion's count stays")
    world(c)
    c.ok("addPromotion", "amy", W, {"code": "GONE", "percent": 10, "scope": "both", "maxRedemptions": 5, "validUntil": "2026-03-10"})
    c.ok("endPromotion", "amy", W, {"code": "GONE"})
    c.ok("addPromotion", "amy", W, {"code": "SPRING", "percent": 20, "scope": "course", "maxRedemptions": 1, "validUntil": "2026-03-31"})
    c.ok("addPromotion", "amy", W, {"code": "OPEN", "percent": 15, "scope": "material", "maxRedemptions": 3, "validUntil": "2026-12-31"})
    c.ok("redeemPromotion", "ada", W, {"memberId": "ada", "code": "SPRING"})
    a = t(2, "10:00")
    stand(c, "eve", "paused", a)
    stand(c, "ada", "paused", a)
    rf(c, "redeemPromotion", "eve", a, {"memberId": "eve", "code": "GONE"}, "member_suspended")  # it is ended
    rf(c, "redeemPromotion", "eve", a, {"memberId": "eve", "code": "SPRING"}, "member_suspended")  # it is used up
    rf(c, "redeemPromotion", "sam", a, {"memberId": "eve", "code": "OPEN"}, "member_suspended")
    rf(c, "redeemPromotion", "ada", a, {"memberId": "ada", "code": "SPRING"}, "member_suspended")  # she has it
    c.ok("listMemberPromotions", "eve", a, {"memberId": "eve"}, {"memberId": "eve", "redemptions": []})
    b = t(2, "11:00")
    stand(c, "eve", "good", b)
    c.ok("redeemPromotion", "eve", b, {"memberId": "eve", "code": "OPEN"},
         {"redemption": redemption("OPEN", "eve", 15, b), "promotion": promo("OPEN", 15, "material", 3, "2026-12-31", 1)})
    rf(c, "redeemPromotion", "eve", b, {"memberId": "eve", "code": "GONE"}, "promotion_ended")
    c.audit_log(b)
    return c


def gate_locker() -> Case:
    c = Case("C4-gate-locker", "C4-gate", "Rent locker, Membership: a paused member is refused member_suspended, before locker_taken and before rental_limit; rentals she had stay")
    world(c)
    for lid, cents in (("K1", 1500), ("K2", 2000), ("K3", 3000)):
        c.ok("addLocker", "amy", W, {"lockerId": lid, "monthlyCents": cents})
    t1, t2 = t(2, "10:00"), t(2, "10:05")
    c.ok("rentLocker", "ada", t1, {"memberId": "ada", "lockerId": "K1"}, {"rental": rental("rnt-1", "K1", "ada", "active", t1)}, new="rnt-1")
    c.ok("rentLocker", "ada", t2, {"memberId": "ada", "lockerId": "K2"}, {"rental": rental("rnt-2", "K2", "ada", "active", t2)}, new="rnt-2")
    a = t(2, "10:10")
    stand(c, "ada", "paused", a)
    stand(c, "bob", "paused", a)
    rf(c, "rentLocker", "ada", a, {"memberId": "ada", "lockerId": "K3"}, "member_suspended")  # two rentals: rental_limit
    rf(c, "rentLocker", "bob", a, {"memberId": "bob", "lockerId": "K1"}, "member_suspended")  # taken: locker_taken
    rf(c, "rentLocker", "bob", a, {"memberId": "bob", "lockerId": "K3"}, "member_suspended")  # free
    rf(c, "rentLocker", "sam", a, {"memberId": "bob", "lockerId": "K3"}, "member_suspended")
    b = t(2, "10:20")
    stand(c, "bob", "good", b)
    rf(c, "rentLocker", "bob", b, {"memberId": "bob", "lockerId": "K1"}, "locker_taken")
    c.ok("rentLocker", "bob", b, {"memberId": "bob", "lockerId": "K3"}, {"rental": rental("rnt-3", "K3", "bob", "active", b)}, new="rnt-3")
    c.audit_log(b)
    return c


def gate_guest() -> Case:
    c = Case("C4-gate-guest", "C4-gate", "Issue guest pass, Membership: a paused member is refused member_suspended, before date_in_past and before guests_not_allowed; no pass exists")
    world(c)
    c.ok("setGuestAllowance", "amy", W, {"tier": "plus", "perMonth": 2, "feeCents": 500})
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    stand(c, "ada", "paused", a)
    rf(c, "issueGuestPass", "sam", a, {"memberId": "bob", "guestName": "Bea", "date": "2026-03-01"}, "member_suspended")  # in the past
    rf(c, "issueGuestPass", "bob", a, {"memberId": "bob", "guestName": "Bea", "date": "2026-06-10"}, "member_suspended")  # basic: no allowance
    rf(c, "issueGuestPass", "ada", a, {"memberId": "ada", "guestName": "Gus", "date": "2026-06-10"}, "member_suspended")
    rf(c, "issueGuestPass", "sam", a, {"memberId": "ada", "guestName": "Gus", "date": "2026-06-10"}, "member_suspended")
    c.ok("listGuestPasses", "ada", a, {"memberId": "ada"}, {"memberId": "ada", "guestPasses": []})
    unbilled(c, "ada", a, 5000)
    b = t(2, "11:00")
    stand(c, "ada", "good", b)
    c.ok("issueGuestPass", "ada", b, {"memberId": "ada", "guestName": "Gus", "date": "2026-06-10"},
         {"guestPass": gpass("gst-1", "ada", "Gus", "2026-06-10")}, new="gst-1")
    stand(c, "bob", "good", b)
    rf(c, "issueGuestPass", "bob", b, {"memberId": "bob", "guestName": "Bea", "date": "2026-03-01"}, "date_in_past")
    c.audit_log(b)
    return c


def gate_enroll() -> Case:
    c = Case("C4-gate-enroll", "C4-gate", "Enroll in course, Membership: a paused member is refused member_suspended, before guardian_consent_missing, already_enrolled and course_full; "
                                          "no enrollment and no charge exist")
    world(c)
    c.ok("addCourse", "sam", W, {"courseId": "solo", "familyId": "laser", "title": "Solo", "feeCents": 0, "capacity": 1})
    c.ok("addCourse", "sam", W, {"courseId": "open", "familyId": "laser", "title": "Open", "feeCents": 3000, "capacity": 5})
    c.ok("enrollInCourse", "bob", W, {"courseId": "solo", "memberId": "bob"})
    c.ok("declareMinor", "sam", W, {"memberId": "eve", "birthDate": "2012-05-01", "guardianId": "ada"})
    a = t(2, "10:00")
    for mem in ("ada", "bob", "eve"):
        stand(c, mem, "paused", a)
    rf(c, "enrollInCourse", "eve", a, {"courseId": "solo", "memberId": "eve"}, "member_suspended")  # a minor without consent; full
    rf(c, "enrollInCourse", "bob", a, {"courseId": "solo", "memberId": "bob"}, "member_suspended")  # already enrolled
    rf(c, "enrollInCourse", "ada", a, {"courseId": "solo", "memberId": "ada"}, "member_suspended")  # full
    rf(c, "enrollInCourse", "ada", a, {"courseId": "open", "memberId": "ada"}, "member_suspended")  # free
    rf(c, "enrollInCourse", "sam", a, {"courseId": "open", "memberId": "ada"}, "member_suspended")
    unbilled(c, "ada", a, 5000)
    b = t(2, "11:00")
    stand(c, "eve", "good", b)
    rf(c, "enrollInCourse", "eve", b, {"courseId": "solo", "memberId": "eve"}, "guardian_consent_missing")
    stand(c, "ada", "good", b)
    c.ok("enrollInCourse", "ada", b, {"courseId": "open", "memberId": "ada"},
         {"enrollment": enr("open", "ada", b), "course": crs("open", "laser", "Open", 3000, 5, 1)})
    unbilled(c, "ada", b, 8000)
    c.audit_log(b)
    return c


def gate_material() -> Case:
    c = Case("C4-gate-material", "C4-gate", "Purchase material, Membership: a paused member is refused member_suspended, before hazard_not_certified and before insufficient_stock; "
                                            "the stock and the account stay, no number is used")
    world(c)
    c.ok("addMaterial", "sam", W, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75})
    c.ok("restockMaterial", "sam", W, {"materialId": "foam", "quantity": 5})
    c.ok("classifyHazard", "amy", W, {"materialId": "ply", "familyId": "laser", "monthlyLimit": 10})
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    rf(c, "purchaseMaterial", "bob", a, {"memberId": "bob", "materialId": "ply", "quantity": 1}, "member_suspended")  # a hazard bob is not certified for
    rf(c, "purchaseMaterial", "sam", a, {"memberId": "bob", "materialId": "foam", "quantity": 6}, "member_suspended")  # more than the stock
    rf(c, "purchaseMaterial", "bob", a, {"memberId": "bob", "materialId": "foam", "quantity": 1}, "member_suspended")
    c.ok("listMaterials", "bob", a, {}, {"materials": [mat("foam", "Foam board", 75, 5), mat("ply", "Plywood", 150, 100)]})
    unbilled(c, "bob", a, 0)
    b = t(2, "11:00")
    stand(c, "bob", "good", b)
    rf(c, "purchaseMaterial", "bob", b, {"memberId": "bob", "materialId": "ply", "quantity": 1}, "hazard_not_certified")
    c.ok("purchaseMaterial", "bob", b, {"memberId": "bob", "materialId": "foam", "quantity": 2},
         bought("pur-1", "bob", "foam", 2, 75, 150, b, "Foam board", 3), new="pur-1")
    unbilled(c, "bob", b, 150)
    c.audit_log(b)
    return c


def gate_checkin() -> Case:
    c = Case("C4-gate-checkin", "C4-gate", "Check in, Membership: a paused member is refused member_suspended after invalid_state and check_in_too_early and before guardian_consent_missing, "
                                           "whichever other gate also fails; the reservation stays")
    world(c)
    certify(c, "bob")
    book(c, "res-1", "bob", "L1", 3, "10:00", "11:00")
    book(c, "res-2", "bob", "L1", 3, "12:00", "13:00")
    c.ok("cancelReservation", "bob", W, {"reservationId": "res-2"})
    d = t(2, "09:30")
    c.ok("revokeCertification", "sam", d, {"memberId": "bob", "familyId": "laser"})
    c.ok("declareMinor", "sam", d, {"memberId": "bob", "birthDate": "2012-05-01", "guardianId": "ada"})
    c.ok("publishWaiver", "amy", d, {"familyId": "laser", "text": "Laser waiver"})
    c.ok("addPpeItem", "sam", d, {"itemId": "gl", "name": "Safety glasses", "stock": 1})
    c.ok("requirePpe", "amy", d, {"familyId": "laser", "itemId": "gl"})
    c.ok("applyLockout", "sam", d, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    stand(c, "bob", "paused", d)
    s1 = t(2, "10:00")
    rf(c, "checkIn", "bob", s1, {"reservationId": "res-2"}, "invalid_state")  # cancelled
    rf(c, "checkIn", "bob", s1, {"reservationId": "res-1"}, "check_in_too_early")
    n = t(3, "10:00")
    rf(c, "checkIn", "bob", n, {"reservationId": "res-1"}, "member_suspended")
    rf(c, "checkIn", "sam", n, {"reservationId": "res-1"}, "member_suspended")
    c.ok("listMemberReservations", "bob", n, {"memberId": "bob"}, {"memberId": "bob", "reservations": [
        res("res-1", "bob", "L1", t(3, "10:00"), t(3, "11:00")), res("res-2", "bob", "L1", t(3, "12:00"), t(3, "13:00"), "cancelled")]})
    stand(c, "bob", "good", n)
    rf(c, "checkIn", "bob", n, {"reservationId": "res-1"}, "guardian_consent_missing")
    c.audit_log(n)
    return c


def gate_reservation_1() -> Case:
    c = Case("C4-gate-reservation-1", "C4-gate", "Create reservation, Reservation rules rule 3, Membership: a paused member is refused member_suspended on a free slot, whoever books; "
                                                 "no reservation exists and no number is used; the reservation she had stays")
    world(c)
    slot = {"memberId": "ada", "machineId": "L1", "start": t(3, "10:00"), "end": t(3, "11:00")}
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    rf(c, "createReservation", "ada", a, slot, "member_suspended")
    rf(c, "createReservation", "sam", a, slot, "member_suspended")
    rf(c, "createReservation", "amy", a, slot, "member_suspended")
    c.ok("listMemberReservations", "ada", a, {"memberId": "ada"}, {"memberId": "ada", "reservations": []})
    b = t(2, "11:00")
    stand(c, "ada", "good", b)
    c.ok("createReservation", "ada", b, slot, {"reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"))}, new="res-1")
    d = t(2, "12:00")
    stand(c, "ada", "paused", d)
    other = {"memberId": "ada", "machineId": "L2", "start": t(3, "14:00"), "end": t(3, "15:00")}
    rf(c, "createReservation", "ada", d, other, "member_suspended")
    c.ok("listMemberReservations", "ada", d, {"memberId": "ada"}, {"memberId": "ada", "reservations": [res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"))]})
    e = t(2, "13:00")
    stand(c, "ada", "good", e)
    c.ok("createReservation", "ada", e, other, {"reservation": res("res-2", "ada", "L2", t(3, "14:00"), t(3, "15:00"))}, new="res-2")
    c.audit_log(e)
    return c


def gate_reservation_2() -> Case:
    c = Case("C4-gate-reservation-2", "C4-gate", "Create reservation, Reservation rules: the paused member's refusal comes after the checks of the slot (invalid_interval, invalid_slot, "
                                                 "starts_in_past, outside_opening_hours) and before every gate after it, though each of them also fails")
    world(c)
    book(c, "res-1", "ada", "L1", 3, "13:00", "14:00")  # clear of the maintenance window below, which may not overlap a reservation
    n = t(2, "10:00")
    c.ok("declareMinor", "sam", n, {"memberId": "bob", "birthDate": "2012-05-01", "guardianId": "ada"})
    stand(c, "bob", "paused", n)
    c.ok("publishWaiver", "amy", n, {"familyId": "laser", "text": "Laser waiver"})
    c.ok("applyLockout", "sam", n, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("setInspectionPlan", "sam", n, {"machineId": "L1", "intervalDays": 1})
    c.ok("scheduleMaintenance", "sam", n, {"machineId": "L1", "start": t(3, "10:00"), "end": t(3, "11:00"), "reason": "service"}, new="mnt-1")
    c.ok("setSupervision", "amy", n, {"familyId": "laser", "required": True})

    def bob(start, end):
        return {"memberId": "bob", "machineId": "L1", "start": start, "end": end}

    slot = bob(t(3, "10:00"), t(3, "12:00"))
    rf(c, "createReservation", "bob", n, bob(t(3, "11:00"), t(3, "10:00")), "invalid_interval")
    rf(c, "createReservation", "bob", n, bob(t(3, "10:10"), t(3, "12:00")), "invalid_slot")
    rf(c, "createReservation", "bob", n, bob(t(2, "09:00"), t(2, "10:00")), "starts_in_past")
    rf(c, "createReservation", "bob", n, bob(t(3, "08:30"), t(3, "09:30")), "outside_opening_hours")
    rf(c, "createReservation", "bob", n, slot, "member_suspended")
    rf(c, "createReservation", "sam", n, slot, "member_suspended")
    stand(c, "bob", "good", n)
    rf(c, "createReservation", "bob", n, slot, "guardian_consent_missing")
    c.ok("grantConsent", "ada", n, {"guardianId": "ada", "memberId": "bob", "familyId": "laser"})
    rf(c, "createReservation", "bob", n, slot, "not_certified")
    c.audit_log(n)
    return c


def gate_door() -> Case:
    c = Case("C4-gate-door", "C4-gate", "Enter door, Membership: a paused member is refused member_suspended after invalid_state and already_inside and before outside_opening_hours; "
                                        "no door event is made")
    world(c)
    for mem, bid in (("ada", "bdg-1"), ("bob", "bdg-2"), ("eve", "bdg-3")):
        c.ok("issueBadge", "sam", W, {"memberId": mem}, {"badge": badge(bid, mem)}, new=bid)
    c.ok("enterDoor", "bob", W, {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "bob", inside=True), "event": door("dor-1", "enter", "bdg-2", W)})
    c.ok("revokeBadge", "sam", W, {"badgeId": "bdg-3"})
    a = t(2, "10:00")
    for mem in ("ada", "bob", "eve"):
        stand(c, mem, "paused", a)
    rf(c, "enterDoor", "bob", a, {"badgeId": "bdg-2"}, "already_inside")
    rf(c, "enterDoor", "eve", a, {"badgeId": "bdg-3"}, "invalid_state")
    rf(c, "enterDoor", "ada", a, {"badgeId": "bdg-1"}, "member_suspended")
    rf(c, "enterDoor", "sam", a, {"badgeId": "bdg-1"}, "member_suspended")
    late = t(2, "21:00")
    rf(c, "enterDoor", "ada", late, {"badgeId": "bdg-1"}, "member_suspended")  # the door is closed
    c.ok("listDoorEvents", "sam", late, {"date": "2026-03-02"}, {"date": "2026-03-02", "events": [door("dor-1", "enter", "bdg-2", W)]})
    stand(c, "ada", "good", late)
    rf(c, "enterDoor", "ada", late, {"badgeId": "bdg-1"}, "outside_opening_hours")
    nxt = t(3, "09:00")
    c.ok("enterDoor", "ada", nxt, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-2", "enter", "bdg-1", nxt)})
    c.audit_log(nxt)
    return c


def gate_waitlist() -> Case:
    c = Case("C4-gate-waitlist", "C4-gate", "Join waitlist, Membership: a paused member is refused member_suspended, before already_enrolled, course_not_full and already_waiting; "
                                            "no entry is made, and the entry she had keeps its place")
    world(c)
    reg(c, "cy", "Cy Dunn", "basic")
    c.ok("addCourse", "sam", W, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 3000, "capacity": 4})
    c.ok("enrollInCourse", "sam", W, {"courseId": "laser-101", "memberId": "bob"})
    c.ok("enrollInCourse", "sam", W, {"courseId": "laser-101", "memberId": "eve"})
    t1 = t(3, "09:00")
    c.ok("joinWaitlist", "ada", t1, {"courseId": "laser-101", "memberId": "ada"}, {"entry": waiting("laser-101", "ada", t1)})
    a = t(3, "10:00")
    for mem in ("ada", "cy", "eve"):
        stand(c, mem, "paused", a)
    rf(c, "joinWaitlist", "cy", a, {"courseId": "laser-101", "memberId": "cy"}, "member_suspended")  # the course is full
    rf(c, "joinWaitlist", "eve", a, {"courseId": "laser-101", "memberId": "eve"}, "member_suspended")  # she is enrolled
    rf(c, "joinWaitlist", "sam", a, {"courseId": "lathe-101", "memberId": "cy"}, "member_suspended")  # it has a free place
    rf(c, "joinWaitlist", "ada", a, {"courseId": "laser-101", "memberId": "ada"}, "member_suspended")  # she is waiting
    c.ok("listWaitlist", "sam", a, {"courseId": "laser-101"}, {"courseId": "laser-101", "entries": [waiting("laser-101", "ada", t1)]})
    b = t(3, "11:00")
    stand(c, "cy", "good", b)
    c.ok("joinWaitlist", "cy", b, {"courseId": "laser-101", "memberId": "cy"}, {"entry": waiting("laser-101", "cy", b)})
    c.ok("listWaitlist", "sam", b, {"courseId": "laser-101"}, {"courseId": "laser-101", "entries": [waiting("laser-101", "ada", t1), waiting("laser-101", "cy", b)]})
    c.audit_log(b)
    return c


def gate_ticket() -> Case:
    c = Case("C4-gate-ticket", "C4-gate", "Sell ticket, Membership: a paused member is refused member_suspended, before event_closed, already_ticketed and event_sold_out; "
                                          "no ticket exists and the event's sold count stays")
    world(c)
    c.ok("addRoom", "amy", W, {"roomId": "lab", "name": "Lab", "capacity": 10})
    fs, fe, us, ue = t(10, "10:00"), t(10, "12:00"), t(11, "10:00"), t(11, "12:00")
    c.ok("createEvent", "sam", W, {"eventId": "fair", "title": "Fair", "roomId": "lab", "start": fs, "end": fe, "capacity": 2, "priceCents": 1500})
    c.ok("createEvent", "sam", W, {"eventId": "full", "title": "Full", "roomId": "lab", "start": us, "end": ue, "capacity": 1, "priceCents": 800})
    c.ok("createEvent", "sam", W, {"eventId": "gone", "title": "Gone", "roomId": "lab", "start": t(12, "10:00"), "end": t(12, "12:00"), "capacity": 3, "priceCents": 500})
    c.ok("sellTicket", "bob", W, {"eventId": "fair", "memberId": "bob"}, None, new="tkt-1")
    c.ok("sellTicket", "ada", W, {"eventId": "full", "memberId": "ada"}, None, new="tkt-2")
    c.ok("cancelEvent", "sam", W, {"eventId": "gone"})
    a = t(2, "10:00")
    stand(c, "eve", "paused", a)
    stand(c, "bob", "paused", a)
    rf(c, "sellTicket", "eve", a, {"eventId": "gone", "memberId": "eve"}, "member_suspended")  # cancelled
    rf(c, "sellTicket", "eve", a, {"eventId": "full", "memberId": "eve"}, "member_suspended")  # sold out
    rf(c, "sellTicket", "bob", a, {"eventId": "fair", "memberId": "bob"}, "member_suspended")  # she holds one
    rf(c, "sellTicket", "sam", a, {"eventId": "fair", "memberId": "eve"}, "member_suspended")  # free
    c.ok("listEventTickets", "sam", a, {"eventId": "fair"}, {"eventId": "fair", "tickets": [ticket("tkt-1", "fair", "bob")]})
    unbilled(c, "eve", a, 0)
    b = t(2, "11:00")
    stand(c, "eve", "good", b)
    rf(c, "sellTicket", "eve", b, {"eventId": "gone", "memberId": "eve"}, "event_closed")
    c.ok("sellTicket", "eve", b, {"eventId": "fair", "memberId": "eve"},
         {"ticket": ticket("tkt-3", "fair", "eve"), "event": event("fair", "Fair", "lab", fs, fe, 2, 1500, 2)}, new="tkt-3")
    unbilled(c, "eve", b, 1500)
    c.audit_log(b)
    return c


def gate_job() -> Case:
    c = Case("C4-gate-job", "C4-gate", "Start job, Maker gate, Membership: a paused maker is refused member_suspended after quote_not_accepted and already_started and before maker_not_certified; "
                                       "no job exists and no number is used")
    world(c)
    c.ok("addMaterial", "sam", W, {"materialId": "pin", "name": "Pin", "unitPriceCents": 1})
    c.ok("addCustomer", "sam", W, {"customerId": "cu1", "name": "Cora Customer"})
    make_quote(c, 1, "cu1", "Pins", [("material", "pin", 4)])  # total 5
    make_quote(c, 2, "cu1", "Laser", [("machine", "L1", 1)])  # total 54
    make_quote(c, 3, "cu1", "More pins", [("material", "pin", 4)])  # total 5
    make_quote(c, 4, "cu1", "Draft", [], decision=None)
    c.ok("startJob", "sam", W, {"quoteId": "quo-3", "makerId": "ada"}, {"job": job("job-1", "quo-3", "cu1", "ada", "started", 5, None, 0)}, new="job-1")
    a = t(2, "10:00")
    for mem in ("ada", "bob", "eve"):
        stand(c, mem, "paused", a)
    rf(c, "startJob", "sam", a, {"quoteId": "quo-4", "makerId": "bob"}, "quote_not_accepted")
    rf(c, "startJob", "sam", a, {"quoteId": "quo-3", "makerId": "ada"}, "already_started")
    rf(c, "startJob", "sam", a, {"quoteId": "quo-1", "makerId": "bob"}, "member_suspended")  # materials only
    rf(c, "startJob", "sam", a, {"quoteId": "quo-2", "makerId": "bob"}, "member_suspended")  # bob is not certified
    rf(c, "startJob", "amy", a, {"quoteId": "quo-2", "makerId": "ada"}, "member_suspended")  # ada is
    c.ok("listJobs", "sam", a, {"customerId": "cu1"}, {"customerId": "cu1", "jobs": [job("job-1", "quo-3", "cu1", "ada", "started", 5, None, 0)]})
    b = t(2, "11:00")
    stand(c, "bob", "good", b)
    rf(c, "startJob", "sam", b, {"quoteId": "quo-2", "makerId": "bob"}, "maker_not_certified")
    c.ok("startJob", "sam", b, {"quoteId": "quo-1", "makerId": "bob"}, {"job": job("job-2", "quo-1", "cu1", "bob", "started", 5, None, 0)}, new="job-2")
    c.audit_log(b)
    return c


# ---- C4-open: the 13 operations that do not check standing and stay open to a paused member --------------------------------

def open_cancel() -> Case:
    c = Case("C4-open-cancel", "C4-open", "Cancel reservation: a paused member (and staff for her) can cancel; pausing had cancelled nothing")
    world(c)
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    book(c, "res-2", "ada", "L2", 3, "12:00", "13:00")
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    c.ok("listMemberReservations", "ada", a, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00")), res("res-2", "ada", "L2", t(3, "12:00"), t(3, "13:00"))]})
    c.ok("cancelReservation", "ada", a, {"reservationId": "res-1"}, {"reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "cancelled")})
    c.ok("cancelReservation", "sam", a, {"reservationId": "res-2"}, {"reservation": res("res-2", "ada", "L2", t(3, "12:00"), t(3, "13:00"), "cancelled")})
    c.audit_log(a)
    return c


def open_checkout() -> Case:
    c = Case("C4-open-checkout", "C4-open", "Check out: a paused member checks out, with the discount of her tier (10 %, not the fee's 500); the usage charge is on her invoice beside the 500")
    world(c)
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    cin, cout = t(3, "10:00"), t(3, "10:30")
    c.ok("checkIn", "ada", cin, {"reservationId": "res-1"}, {"reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "checkedIn", cin=cin, rate=45)})
    stand(c, "ada", "paused", t(3, "10:10"))
    c.ok("checkOut", "ada", cout, {"reservationId": "res-1"}, {
        "reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "completed", cin=cin, cout=cout, rate=45, met=30, amt=1215),
        "charge": chg("usage", "res-1", 30, 45, 10, 1215, cout)})
    unbilled(c, "ada", cout, 6215)
    c.ok("generateInvoices", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 500, 0, 500, I), course_line(), chg("usage", "res-1", 30, 45, 10, 1215, cout)]),
        invoice("inv-2", "bob", "2026-03", I, D, [mline("basic", 2500, 0, 2500, I)]),
        invoice("inv-3", "eve", "2026-03", I, D, [mline("pro", 9900, 0, 9900, I)])]})
    c.audit_log(I)
    return c


def open_payment() -> Case:
    c = Case("C4-open-payment", "C4-open", "Record payment: a paused member pays an invoice; she also reads her account, invoices, reservations and notices")
    world(c)
    c.ok("generateInvoices", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 4900, 0, 4900, I), course_line()]),
        invoice("inv-2", "bob", "2026-03", I, D, [mline("basic", 2500, 0, 2500, I)]),
        invoice("inv-3", "eve", "2026-03", I, D, [mline("pro", 9900, 0, 9900, I)])]})
    a = t(2, "09:00", 4)
    stand(c, "ada", "paused", a)
    c.ok("getAccount", "ada", a, {"memberId": "ada"}, account("ada", open_=9900))
    c.ok("listInvoices", "ada", a, {"memberId": "ada"}, {"memberId": "ada", "invoices": [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 4900, 0, 4900, I), course_line()])]})
    c.ok("listMemberReservations", "ada", a, {"memberId": "ada"}, {"memberId": "ada", "reservations": []})
    c.ok("listNotices", "ada", a, {"memberId": "ada"}, {"memberId": "ada", "notices": []})
    paid = invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 4900, 0, 4900, I), course_line()], paid=9900)
    c.ok("recordPayment", "ada", a, {"invoiceId": "inv-1", "amountCents": 9900},
         {"payment": pay("pay-1", "inv-1", "ada", 9900, a), "invoice": paid, "creditBalanceCents": 0}, new="pay-1")
    c.ok("getAccount", "ada", a, {"memberId": "ada"}, account("ada"))
    c.audit_log(a)
    return c


def open_waiver() -> Case:
    c = Case("C4-open-waiver", "C4-open", "Sign waiver: a paused member signs the family's waiver")
    world(c)
    c.ok("publishWaiver", "amy", W, {"familyId": "laser", "text": "Laser waiver"})
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    c.ok("signWaiver", "bob", a, {"memberId": "bob", "familyId": "laser"},
         {"signature": {"memberId": "bob", "familyId": "laser", "version": 1, "signedAt": a}})
    c.refuse("signWaiver", "bob", a, {"memberId": "bob", "familyId": "laser"}, "already_signed")
    c.audit_log(a)
    return c


def open_minor() -> Case:
    c = Case("C4-open-minor", "C4-open", "Declare minor: a paused member is declared a minor, and a paused member is the guardian")
    world(c)
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    stand(c, "ada", "paused", a)
    c.ok("declareMinor", "sam", a, {"memberId": "bob", "birthDate": "2012-06-15", "guardianId": "ada"},
         {"guardianship": {"memberId": "bob", "birthDate": "2012-06-15", "guardianId": "ada", "consentFamilyIds": []}})
    c.ok("getGuardianship", "bob", a, {"memberId": "bob"},
         {"guardianship": {"memberId": "bob", "birthDate": "2012-06-15", "guardianId": "ada", "consentFamilyIds": []}})
    c.audit_log(a)
    return c


def open_ppe() -> Case:
    c = Case("C4-open-ppe", "C4-open", "Return PPE: a paused member's items are returned; the stock grows back")
    world(c)
    c.ok("addPpeItem", "sam", W, {"itemId": "gloves", "name": "Gloves", "stock": 3})
    c.ok("issuePpe", "sam", W, {"memberId": "bob", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 2)})
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    c.ok("listMemberPpe", "bob", a, {"memberId": "bob"}, {"memberId": "bob", "items": [ppe("gloves", "Gloves", 2)]})
    c.ok("returnPpe", "sam", a, {"memberId": "bob", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 3)})
    c.ok("listMemberPpe", "bob", a, {"memberId": "bob"}, {"memberId": "bob", "items": []})
    c.audit_log(a)
    return c


def open_door() -> Case:
    c = Case("C4-open-door", "C4-open", "Leave door: a paused member who is inside leaves, with a door event")
    world(c)
    c.ok("issueBadge", "sam", W, {"memberId": "ada"}, {"badge": badge("bdg-1", "ada")}, new="bdg-1")
    c.ok("enterDoor", "ada", W, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-1", "enter", "bdg-1", W)})
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    c.ok("leaveDoor", "ada", a, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada"), "event": door("dor-2", "leave", "bdg-1", a)})
    c.ok("listDoorEvents", "sam", a, {"date": "2026-03-02"}, {"date": "2026-03-02", "events": [door("dor-1", "enter", "bdg-1", W), door("dor-2", "leave", "bdg-1", a)]})
    c.audit_log(a)
    return c


def open_guest() -> Case:
    c = Case("C4-open-guest", "C4-open", "Admit guest: a paused sponsor who is inside admits the guest of her pass")
    world(c)
    c.ok("setGuestAllowance", "amy", W, {"tier": "plus", "perMonth": 5, "feeCents": 0})
    c.ok("issueGuestPass", "ada", W, {"memberId": "ada", "guestName": "Gus", "date": "2026-03-02"}, None, new="gst-1")
    c.ok("issueBadge", "sam", W, {"memberId": "ada"}, {"badge": badge("bdg-1", "ada")}, new="bdg-1")
    c.ok("enterDoor", "ada", W, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-1", "enter", "bdg-1", W)})
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    c.ok("admitGuest", "ada", a, {"guestPassId": "gst-1", "badgeId": "bdg-1"}, {"event": door("dor-2", "guest", "bdg-1", a, "gst-1")})
    c.audit_log(a)
    return c


def open_badge() -> Case:
    c = Case("C4-open-badge", "C4-open", "Issue badge: a badge is issued to a paused member")
    world(c)
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    c.ok("issueBadge", "sam", a, {"memberId": "bob"}, {"badge": badge("bdg-1", "bob")}, new="bdg-1")
    c.refuse("issueBadge", "amy", a, {"memberId": "bob"}, "badge_active")
    c.audit_log(a)
    return c


def open_volunteer() -> Case:
    c = Case("C4-open-volunteer", "C4-open", "Log volunteer time: a paused member logs time for herself")
    world(c)
    c.ok("setVolunteerRate", "amy", W, {"centsPerHour": 1500})
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    c.ok("logVolunteerTime", "ada", a, {"memberId": "ada", "date": "2026-03-02", "minutes": 100, "note": "Bench build"},
         {"entry": {"entryId": "vol-1", "memberId": "ada", "date": "2026-03-02", "minutes": 100, "note": "Bench build",
                    "status": "pending", "creditCents": None}}, new="vol-1")
    c.audit_log(a)
    return c


def open_course() -> Case:
    c = Case("C4-open-course", "C4-open", "Complete course: a paused member's enrollment is completed and certifies her")
    world(c)
    c.ok("addCourse", "sam", W, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe basics", "feeCents": 0, "capacity": 4})
    c.ok("enrollInCourse", "bob", W, {"courseId": "lathe-101", "memberId": "bob"})
    a = t(2, "10:00")
    stand(c, "bob", "paused", a)
    c.ok("completeCourse", "sam", a, {"courseId": "lathe-101", "memberId": "bob"},
         {"enrollment": enr("lathe-101", "bob", W, a), "certification": cert("bob", "lathe", "2026-03-02", "2027-03-02")})
    c.ok("listCertifications", "bob", a, {"memberId": "bob"}, {"memberId": "bob", "certifications": [cert("bob", "lathe", "2026-03-02", "2027-03-02")]})
    c.audit_log(a)
    return c


def open_job() -> Case:
    c = Case("C4-open-job", "C4-open", "Complete job: a paused maker completes the job and earns the maker credit (30 % of 5 is 2)")
    world(c)
    c.ok("addMaterial", "sam", W, {"materialId": "pin", "name": "Pin", "unitPriceCents": 1})
    c.ok("addCustomer", "sam", W, {"customerId": "cu1", "name": "Cora Customer"})
    make_quote(c, 1, "cu1", "Pins", [("material", "pin", 4)])
    c.ok("startJob", "sam", W, {"quoteId": "quo-1", "makerId": "ada"}, {"job": job("job-1", "quo-1", "cu1", "ada", "started", 5, None, 0)}, new="job-1")
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    c.ok("completeJob", "sam", a, {"jobId": "job-1"}, {"job": job("job-1", "quo-1", "cu1", "ada", "completed", 5, 2, 0)})
    c.ok("getAccount", "ada", a, {"memberId": "ada"}, account("ada", credit=2, unbilled=5000))
    c.audit_log(a)
    return c


def open_project() -> Case:
    c = Case("C4-open-project", "C4-open", "Create project: a paused member creates a project of her own")
    world(c)
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    c.ok("createProject", "ada", a, {"projectId": "p1", "ownerId": "ada", "title": "Lamp"}, {"project": project("p1", "ada", "Lamp")})
    c.audit_log(a)
    return c


def open_misc() -> Case:
    c = Case("C4-open-misc", "C4-open", "Leave waitlist, End locker rental: a paused member leaves a waiting list and ends a rental, since neither checks standing")
    world(c)
    c.ok("enrollInCourse", "sam", W, {"courseId": "laser-101", "memberId": "bob"})
    c.ok("enrollInCourse", "sam", W, {"courseId": "laser-101", "memberId": "eve"})
    t1 = t(3, "09:00")
    c.ok("joinWaitlist", "ada", t1, {"courseId": "laser-101", "memberId": "ada"}, {"entry": waiting("laser-101", "ada", t1)})
    c.ok("addLocker", "amy", t1, {"lockerId": "K1", "monthlyCents": 1500})
    c.ok("rentLocker", "ada", t1, {"memberId": "ada", "lockerId": "K1"}, None, new="rnt-1")
    a = t(3, "10:00")
    stand(c, "ada", "paused", a)
    c.ok("leaveWaitlist", "ada", a, {"courseId": "laser-101", "memberId": "ada"}, {"entry": waiting("laser-101", "ada", t1)})
    c.ok("endLockerRental", "ada", a, {"rentalId": "rnt-1"}, {"rental": rental("rnt-1", "K1", "ada", "ended", t1, a)})
    c.audit_log(a)
    return c


# ---- C4-keep: what a paused member keeps paying and drawing, as a suspended one does ------------------------------------

def keep_noshow() -> Case:
    c = Case("C4-keep-noshow", "C4-keep", "Charge no-shows: the run does not check standing, so the paused member's missed reservation is charged (half of 60 minutes at 45)")
    world(c)
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    stand(c, "ada", "paused", t(2, "10:00"))
    run = t(3, "11:00")
    c.ok("chargeNoShows", "sam", run, {}, {"charges": [chg("noShow", "res-1", 60, 45, 50, 1350, run)]})
    c.ok("listMemberReservations", "ada", run, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "missed")]})
    unbilled(c, "ada", run, 6350)
    c.audit_log(run)
    return c


def keep_locker() -> Case:
    c = Case("C4-keep-locker", "C4-keep", "Bill lockers: standing plays no part, so the paused member's locker is billed in full")
    world(c)
    c.ok("addLocker", "amy", W, {"lockerId": "K1", "monthlyCents": 1500})
    r = t(2, "10:00")
    c.ok("rentLocker", "ada", r, {"memberId": "ada", "lockerId": "K1"}, {"rental": rental("rnt-1", "K1", "ada", "active", r)}, new="rnt-1")
    stand(c, "ada", "paused", t(2, "10:30"))
    last = t(31, "23:59")
    c.ok("billLockers", "sue", last, {"month": "2026-03"}, {"month": "2026-03", "charges": [chg("locker", "rnt-1", 1, 1500, 0, 1500, last)]})
    unbilled(c, "ada", last, 6500)
    c.audit_log(last)
    return c


def keep_pass() -> Case:
    c = Case("C4-keep-pass", "C4-keep", "Pass draw: standing plays no part, so a paused member's pass covers the metered minutes at check-out (20 of 30 minutes)")
    world(c)
    c.ok("addPassProduct", "amy", W, {"productId": "p30", "name": "Thirty", "minutes": 30, "priceCents": 3000, "validDays": 60})
    c.ok("sellPass", "ada", W, {"memberId": "ada", "productId": "p30"}, None, new="pas-1")
    book(c, "res-1", "ada", "L1", 3, "10:00", "11:00")
    cin, cout = t(3, "10:00"), t(3, "10:20")
    c.ok("checkIn", "ada", cin, {"reservationId": "res-1"})
    stand(c, "ada", "paused", t(3, "10:10"))
    c.ok("checkOut", "ada", cout, {"reservationId": "res-1"}, {
        "reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "completed", cin=cin, cout=cout, rate=45, met=20, amt=0),
        "charge": chg("usage", "res-1", 0, 45, 10, 0, cout)})
    c.ok("listMemberPasses", "ada", cout, {"memberId": "ada"}, {"memberId": "ada", "passes": [passrec("pas-1", "ada", "p30", 10, "2026-03-02", "2026-05-01")]})
    c.audit_log(cout)
    return c


# ---- C4-fee: the monthly fee of a paused member ---------------------------------------------------------------------------

def fee_1() -> Case:
    c = Case("C4-fee-1", "C4-fee", "Generate invoices, Membership, Tier, Postings, Revenue report, Charges by kind: a paused member's membership line is 500 cents whatever the tier "
                                    "(plus, basic, pro), ref the current tier, with the invoice made; a suspended member keeps the full fee; the line posts to revenueMembership at 500")
    world(c)
    reg(c, "cy", "Cy Dunn", "basic")
    reg(c, "dan", "Dan Reed", "pro")
    a = t(2, "10:00")
    stand(c, "ada", "paused", a)
    stand(c, "bob", "suspended", a)
    stand(c, "cy", "paused", a)
    stand(c, "dan", "paused", a)
    lines = [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 500, 0, 500, I), course_line()]),
        invoice("inv-2", "bob", "2026-03", I, D, [mline("basic", 2500, 0, 2500, I)]),
        invoice("inv-3", "cy", "2026-03", I, D, [mline("basic", 500, 0, 500, I)]),
        invoice("inv-4", "dan", "2026-03", I, D, [mline("pro", 500, 0, 500, I)]),
        invoice("inv-5", "eve", "2026-03", I, D, [mline("pro", 9900, 0, 9900, I)])]
    c.ok("generateInvoices", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "invoices": lines})
    c.ok("getMember", "sam", I, {"memberId": "ada"}, mrec("ada", "paused"))
    c.ok("getMember", "sam", I, {"memberId": "bob"}, mrec("bob", "suspended"))
    c.ok("revenueReport", "sam", I, {"month": "2026-03"}, {
        "month": "2026-03", "invoicesGenerated": True, "invoiceCount": 5, "membershipCents": 13900, "usageCents": 0, "materialCents": 0,
        "courseCents": 5000, "invoicedCents": 18900, "creditAppliedCents": 0, "outstandingCents": 18900, "paymentsCents": 0, "creditsIssuedCents": 0})
    c.ok("chargesByKind", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds({"membership": 13900, "course": 5000}, 18900), "totalCents": 18900})
    c.ok("listJournalEntries", "amy", I, {"month": "2026-04"}, {"month": "2026-04", "entries": [
        je(1, I, "receivable", "revenueMembership", 500, "inv-1"), je(2, I, "receivable", "revenueCourse", 5000, "inv-1"),
        je(3, I, "receivable", "revenueMembership", 2500, "inv-2"), je(4, I, "receivable", "revenueMembership", 500, "inv-3"),
        je(5, I, "receivable", "revenueMembership", 500, "inv-4"), je(6, I, "receivable", "revenueMembership", 9900, "inv-5")]})
    c.ok("getTrialBalance", "amy", I, {}, tb({"receivable": (18900, 0), "revenueCourse": (0, 5000), "revenueMembership": (0, 13900)}, 18900))
    c.audit_log(I)
    return c


def fee_2() -> Case:
    c = Case("C4-fee-2", "C4-fee", "Generate invoices, Membership: the fee is judged by the standing at the request time of generate invoices, whatever month is invoiced "
                                    "(paused in the month and reinstated before: full; paused only at the request: 500), and ref is the member's current tier")
    world(c)
    stand(c, "bob", "paused", t(2, "10:00"))
    stand(c, "bob", "good", t(20, "10:00"))
    stand(c, "ada", "paused", t(1, "07:00", 4))
    stand(c, "eve", "paused", t(1, "07:00", 4))
    c.ok("changeTier", "sam", t(1, "07:01", 4), {"memberId": "eve", "tier": "basic"}, mrec("eve", "paused", "basic"))
    c.ok("generateInvoices", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 500, 0, 500, I), course_line()]),
        invoice("inv-2", "bob", "2026-03", I, D, [mline("basic", 2500, 0, 2500, I)]),
        invoice("inv-3", "eve", "2026-03", I, D, [mline("basic", 500, 0, 500, I)])]})
    stand(c, "ada", "good", t(20, "10:00", 4))
    stand(c, "bob", "paused", t(30, "10:00", 4))
    i2, d2 = t(1, "08:00", 5), t(15, "08:00", 5)
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": [
        invoice("inv-4", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2)]),
        invoice("inv-5", "bob", "2026-04", i2, d2, [mline("basic", 500, 0, 500, i2)]),
        invoice("inv-6", "eve", "2026-04", i2, d2, [mline("basic", 500, 0, 500, i2)])]})
    stand(c, "eve", "good", t(20, "10:00", 5), tier="basic")
    i3, d3 = t(1, "08:00", 6), t(15, "08:00", 6)
    c.ok("generateInvoices", "sam", i3, {"month": "2026-05"}, {"month": "2026-05", "invoices": [
        invoice("inv-7", "ada", "2026-05", i3, d3, [mline("plus", 4900, 0, 4900, i3)]),
        invoice("inv-8", "bob", "2026-05", i3, d3, [mline("basic", 500, 0, 500, i3)]),
        invoice("inv-9", "eve", "2026-05", i3, d3, [mline("basic", 2500, 0, 2500, i3)])]})
    c.audit_log(i3)
    return c


def fee_3() -> Case:
    c = Case("C4-fee-3", "C4-fee", "Household discount: it applies to the 500 cents of a paused member, a paused member counts as a member (so the other two of a household of 3 "
                                    "get it), a household of 2 gets none, and the count is judged at the request time")
    world(c)
    reg(c, "cy", "Cy Dunn", "basic")
    reg(c, "dan", "Dan Reed", "pro")
    c.ok("createHousehold", "sam", W, {"householdId": "home", "name": "Marsh home", "payerId": "eve"})
    c.ok("addHouseholdMember", "sam", W, {"householdId": "home", "memberId": "ada"})
    c.ok("addHouseholdMember", "sam", W, {"householdId": "home", "memberId": "bob"})
    c.ok("createHousehold", "sam", W, {"householdId": "duo", "name": "Duo", "payerId": "cy"})
    c.ok("addHouseholdMember", "sam", W, {"householdId": "duo", "memberId": "dan"})
    a = t(2, "10:00")
    stand(c, "eve", "paused", a)  # the payer of the household of 3
    stand(c, "cy", "paused", a)  # a member of the household of 2
    c.ok("generateInvoices", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 4900, 10, 4410, I), course_line()]),
        invoice("inv-2", "bob", "2026-03", I, D, [mline("basic", 2500, 10, 2250, I)]),
        invoice("inv-3", "cy", "2026-03", I, D, [mline("basic", 500, 0, 500, I)]),
        invoice("inv-4", "dan", "2026-03", I, D, [mline("pro", 9900, 0, 9900, I)]),
        invoice("inv-5", "eve", "2026-03", I, D, [mline("pro", 500, 10, 450, I)])]})
    c.ok("removeHouseholdMember", "sam", t(10, "10:00", 4), {"householdId": "home", "memberId": "bob"})
    i2, d2 = t(1, "08:00", 5), t(15, "08:00", 5)
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": [
        invoice("inv-6", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2)]),
        invoice("inv-7", "bob", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2)]),
        invoice("inv-8", "cy", "2026-04", i2, d2, [mline("basic", 500, 0, 500, i2)]),
        invoice("inv-9", "dan", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)]),
        invoice("inv-10", "eve", "2026-04", i2, d2, [mline("pro", 500, 0, 500, i2)])]})
    c.audit_log(i2)
    return c


def fee_credit() -> Case:
    c = Case("C4-fee-credit", "C4-fee", "Generate invoices, Postings: credit is applied to a paused member's invoice of 500 (credit 700 pays it, 200 is left); the entries are "
                                         "500 to revenueMembership and 500 of credit applied")
    world(c)
    c.ok("issueCredit", "amy", W, {"memberId": "eve", "amountCents": 700, "reason": "goodwill"})
    stand(c, "eve", "paused", t(2, "10:00"))
    c.ok("generateInvoices", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 4900, 0, 4900, I), course_line()]),
        invoice("inv-2", "bob", "2026-03", I, D, [mline("basic", 2500, 0, 2500, I)]),
        invoice("inv-3", "eve", "2026-03", I, D, [mline("pro", 500, 0, 500, I)], credit=500)]})
    c.ok("getAccount", "eve", I, {"memberId": "eve"}, account("eve", credit=200))
    c.ok("listJournalEntries", "amy", I, {"month": "2026-03"}, {"month": "2026-03", "entries": [je(1, W, "expenseGoodwill", "creditLiability", 700, "eve")]})
    c.ok("listJournalEntries", "amy", I, {"month": "2026-04"}, {"month": "2026-04", "entries": [
        je(2, I, "receivable", "revenueMembership", 4900, "inv-1"), je(3, I, "receivable", "revenueCourse", 5000, "inv-1"),
        je(4, I, "receivable", "revenueMembership", 2500, "inv-2"), je(5, I, "receivable", "revenueMembership", 500, "inv-3"),
        je(6, I, "creditLiability", "receivable", 500, "inv-3")]})
    c.ok("getTrialBalance", "amy", I, {}, tb({"creditLiability": (500, 700), "expenseGoodwill": (700, 0), "receivable": (12900, 500),
                                             "revenueCourse": (0, 5000), "revenueMembership": (0, 7900)}, 14100))
    c.audit_log(I)
    return c


def fee_refund() -> Case:
    c = Case("C4-fee-refund", "C4-fee", "Issue refund: line 1 of a paused member's paid invoice is 500 cents, so only 500 can be refunded (in parts), and not a cent more")
    world(c)
    stand(c, "ada", "paused", t(2, "10:00"))
    c.ok("generateInvoices", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 500, 0, 500, I), course_line()]),
        invoice("inv-2", "bob", "2026-03", I, D, [mline("basic", 2500, 0, 2500, I)]),
        invoice("inv-3", "eve", "2026-03", I, D, [mline("pro", 9900, 0, 9900, I)])]})
    p = t(2, "09:00", 4)
    c.ok("recordPayment", "ada", p, {"invoiceId": "inv-1", "amountCents": 5500}, None, new="pay-1")
    r = t(3, "09:00", 4)

    def refund(n, amount, reason):
        return {"refund": {"refundId": f"rfd-{n}", "invoiceId": "inv-1", "lineNumber": 1, "amountCents": amount, "reason": reason, "at": r}}

    rf(c, "issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 501, "reason": "x"}, "refund_too_large")
    c.ok("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 200, "reason": "p1"},
         {**refund(1, 200, "p1"), "creditBalanceCents": 200}, new="rfd-1")
    rf(c, "issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 301, "reason": "x"}, "refund_too_large")
    c.ok("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 300, "reason": "p2"},
         {**refund(2, 300, "p2"), "creditBalanceCents": 500}, new="rfd-2")
    rf(c, "issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 1, "reason": "x"}, "refund_too_large")
    c.ok("getAccount", "ada", r, {"memberId": "ada"}, account("ada", credit=500))
    c.ok("listJournalEntries", "amy", r, {"month": "2026-04"}, {"month": "2026-04", "entries": [
        je(1, I, "receivable", "revenueMembership", 500, "inv-1"), je(2, I, "receivable", "revenueCourse", 5000, "inv-1"),
        je(3, I, "receivable", "revenueMembership", 2500, "inv-2"), je(4, I, "receivable", "revenueMembership", 9900, "inv-3"),
        je(5, p, "cash", "receivable", 5500, "pay-1"), je(6, r, "revenueRefunds", "creditLiability", 200, "rfd-1"),
        je(7, r, "revenueRefunds", "creditLiability", 300, "rfd-2")]})
    c.audit_log(r)
    return c


# ---- C4-dun, C4-read, C4-promote ----------------------------------------------------------------------------------------

def dun_1() -> Case:
    c = Case("C4-dun-1", "C4-dun", "Dunning run, step 3: a paused member is not already suspended, so the suspension step makes her suspended (notice and fee steps leave her paused); "
                                   "her next invoice has the tier's full fee beside the late fee")
    world(c)
    stand(c, "ada", "paused", t(1, "07:00", 4))
    c.ok("generateInvoices", "sam", I, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", I, D, [mline("plus", 500, 0, 500, I), course_line()]),
        invoice("inv-2", "bob", "2026-03", I, D, [mline("basic", 2500, 0, 2500, I)]),
        invoice("inv-3", "eve", "2026-03", I, D, [mline("pro", 9900, 0, 9900, I)])]})
    p = t(2, "09:00", 4)
    c.ok("recordPayment", "sam", p, {"invoiceId": "inv-2", "amountCents": 2500}, None, new="pay-1")
    c.ok("recordPayment", "sam", p, {"invoiceId": "inv-3", "amountCents": 9900}, None, new="pay-2")
    lad = {"noticeDays": 3, "feeDays": 7, "feeCents": 300, "suspendDays": 14}
    c.ok("setDunningLadder", "amy", p, dict(lad), {"ladder": dict(lad)})
    t3, t7, t14 = t(18, "08:00", 4), t(22, "08:00", 4), t(29, "08:00", 4)
    c.ok("runDunning", "sam", t3, {}, {"actions": [{"invoiceId": "inv-1", "step": "notice"}]})
    c.ok("getMember", "sam", t3, {"memberId": "ada"}, mrec("ada", "paused"))
    c.ok("runDunning", "sam", t7, {}, {"actions": [{"invoiceId": "inv-1", "step": "fee"}]})
    c.ok("getMember", "sam", t7, {"memberId": "ada"}, mrec("ada", "paused"))
    c.ok("runDunning", "sam", t14, {}, {"actions": [{"invoiceId": "inv-1", "step": "suspend"}]})
    c.ok("getMember", "sam", t14, {"memberId": "ada"}, mrec("ada", "suspended"))
    c.ok("runDunning", "sam", t14, {}, {"actions": []})
    c.ok("listDunningCases", "ada", t14, {"memberId": "ada"}, {"memberId": "ada", "cases": [
        {"caseId": "dun-1", "invoiceId": "inv-1", "memberId": "ada", "noticedAt": t3, "feeChargedAt": t7, "suspendedAt": t14}]})
    c.ok("listNotices", "ada", t14, {"memberId": "ada"}, {"memberId": "ada", "notices": [
        notice(1, "ada", "dunning", "inv-1", t3), notice(2, "ada", "suspension", "inv-1", t14)]})
    c.ok("getAccount", "ada", t14, {"memberId": "ada"}, account("ada", open_=5500, overdue=5500, unbilled=300))
    i2, d2 = t(1, "08:00", 5), t(15, "08:00", 5)
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": [
        invoice("inv-4", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2), chg("lateFee", "inv-1", 1, 300, 0, 300, t7)]),
        invoice("inv-5", "bob", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2)]),
        invoice("inv-6", "eve", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)])]})
    c.audit_log(i2)
    return c


def read_eligibility() -> Case:
    c = Case("C4-read-eligibility", "C4-read", "Check eligibility, Eligibility blocker 1: a paused member has the blocker member_suspended, first in the order, as a suspended one has; "
                                               "good standing removes it")
    world(c)
    n = t(2, "10:00")
    stand(c, "ada", "paused", n)
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L2", "date": "2026-03-03"}, elig("ada", "L2", "2026-03-03", ["member_suspended"]))
    c.ok("checkEligibility", "sam", n, {"memberId": "ada", "machineId": "L2", "date": "2026-03-03"}, elig("ada", "L2", "2026-03-03", ["member_suspended"]))
    c.ok("checkEligibility", "bob", n, {"memberId": "bob", "machineId": "L1", "date": "2026-03-03"}, elig("bob", "L1", "2026-03-03", ["not_certified"]))
    stand(c, "bob", "paused", n)
    c.ok("checkEligibility", "bob", n, {"memberId": "bob", "machineId": "L1", "date": "2026-03-03"},
         elig("bob", "L1", "2026-03-03", ["member_suspended", "not_certified"]))
    c.ok("setInspectionPlan", "sam", n, {"machineId": "L1", "intervalDays": 29})
    c.ok("applyLockout", "sam", n, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("checkEligibility", "amy", n, {"memberId": "bob", "machineId": "L1", "date": "2026-03-31"},
         elig("bob", "L1", "2026-03-31", ["member_suspended", "not_certified", "machine_locked_out", "inspection_due"]))
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L1", "date": "2026-03-31"},
         elig("ada", "L1", "2026-03-31", ["member_suspended", "machine_locked_out", "inspection_due"]))
    stand(c, "ada", "suspended", n)
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L2", "date": "2026-03-03"}, elig("ada", "L2", "2026-03-03", ["member_suspended"]))
    stand(c, "ada", "good", n)
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L2", "date": "2026-03-03"}, elig("ada", "L2", "2026-03-03", []))
    c.audit_log(n)
    return c


def read_compliance() -> Case:
    c = Case("C4-read-compliance", "C4-read", "Compliance summary: `standing` answers paused for a paused member (and still good and suspended), the rest of the summary as before")
    world(c)
    n = t(2, "10:00")
    rows = [fam("laser", True, True, True, True), fam("lathe", False, True, True, True)]
    bob_rows = [fam("laser", False, True, True, True), fam("lathe", False, True, True, True)]
    c.ok("complianceSummary", "ada", n, {"memberId": "ada"}, {"memberId": "ada", "standing": "good", "minor": False, "families": rows})
    stand(c, "ada", "paused", n)
    c.ok("complianceSummary", "ada", n, {"memberId": "ada"}, {"memberId": "ada", "standing": "paused", "minor": False, "families": rows})
    c.ok("complianceSummary", "sam", n, {"memberId": "ada"}, {"memberId": "ada", "standing": "paused", "minor": False, "families": rows})
    stand(c, "ada", "suspended", n)
    c.ok("complianceSummary", "amy", n, {"memberId": "ada"}, {"memberId": "ada", "standing": "suspended", "minor": False, "families": rows})
    stand(c, "ada", "paused", n)
    stand(c, "bob", "paused", n)
    c.ok("complianceSummary", "bob", n, {"memberId": "bob"}, {"memberId": "bob", "standing": "paused", "minor": False, "families": bob_rows})
    c.ok("complianceSummary", "eve", n, {"memberId": "eve"}, {"memberId": "eve", "standing": "good", "minor": False, "families": bob_rows})
    c.audit_log(n)
    return c


def promote_1() -> Case:
    c = Case("C4-promote-1", "C4-promote", "Waitlist promotion, Skipped entry: a paused member on the list is skipped with member_suspended and keeps her place; once reinstated "
                                           "she is enrolled in the next free place")
    world(c)
    reg(c, "ann", "Ann Lowe", "basic")
    reg(c, "dan", "Dan Reed", "basic")
    c.ok("addCourse", "sam", W, {"courseId": "w2", "familyId": "laser", "title": "Welding two", "feeCents": 1000, "capacity": 2})
    c.ok("enrollInCourse", "sam", W, {"courseId": "w2", "memberId": "ada"})
    c.ok("enrollInCourse", "sam", W, {"courseId": "w2", "memberId": "eve"})
    t1, t2_, t3_ = t(3, "09:00"), t(3, "09:05"), t(3, "09:10")
    c.ok("joinWaitlist", "ann", t1, {"courseId": "w2", "memberId": "ann"}, {"entry": waiting("w2", "ann", t1)})
    c.ok("joinWaitlist", "bob", t2_, {"courseId": "w2", "memberId": "bob"}, {"entry": waiting("w2", "bob", t2_)})
    c.ok("joinWaitlist", "dan", t3_, {"courseId": "w2", "memberId": "dan"}, {"entry": waiting("w2", "dan", t3_)})
    stand(c, "ann", "paused", t(3, "09:15"))
    p1 = t(4, "09:00")
    c.ok("completeCourse", "sam", p1, {"courseId": "w2", "memberId": "ada"})
    c.ok("promoteWaitlist", "sam", p1, {"courseId": "w2"}, {"courseId": "w2", "enrollments": [enr("w2", "bob", p1)],
                                                          "skipped": [skipped("ann", "member_suspended")]})
    c.ok("listWaitlist", "sam", p1, {"courseId": "w2"}, {"courseId": "w2", "entries": [waiting("w2", "ann", t1), waiting("w2", "dan", t3_)]})
    c.ok("promoteWaitlist", "sam", p1, {"courseId": "w2"}, {"courseId": "w2", "enrollments": [], "skipped": []})
    p2 = t(5, "09:00")
    c.ok("completeCourse", "sam", p2, {"courseId": "w2", "memberId": "eve"})
    # one place is free again: ann is skipped, dan (next in line) is enrolled and the course is full
    c.ok("promoteWaitlist", "sam", p2, {"courseId": "w2"}, {"courseId": "w2", "enrollments": [enr("w2", "dan", p2)],
                                                          "skipped": [skipped("ann", "member_suspended")]})
    c.ok("listWaitlist", "sam", p2, {"courseId": "w2"}, {"courseId": "w2", "entries": [waiting("w2", "ann", t1)]})
    p3 = t(6, "09:00")
    c.ok("completeCourse", "sam", p3, {"courseId": "w2", "memberId": "bob"})
    stand(c, "ann", "good", p3)
    c.ok("promoteWaitlist", "sam", p3, {"courseId": "w2"}, {"courseId": "w2", "enrollments": [enr("w2", "ann", p3)], "skipped": []})
    c.ok("listWaitlist", "sam", p3, {"courseId": "w2"}, {"courseId": "w2", "entries": []})
    c.audit_log(p3)
    return c


def build_cases() -> list[dict[str, Any]]:
    cases = [standing_1(), standing_2(),
             gate_ppe(), gate_deposit(), gate_pass(), gate_promotion(), gate_locker(), gate_guest(), gate_enroll(), gate_material(),
             gate_checkin(), gate_reservation_1(), gate_reservation_2(), gate_door(), gate_waitlist(), gate_ticket(), gate_job(),
             open_cancel(), open_checkout(), open_payment(), open_waiver(), open_minor(), open_ppe(), open_door(), open_guest(),
             open_badge(), open_volunteer(), open_course(), open_job(), open_project(), open_misc(),
             keep_noshow(), keep_locker(), keep_pass(),
             fee_1(), fee_2(), fee_3(), fee_credit(), fee_refund(), dun_1(),
             read_eligibility(), read_compliance(), promote_1()]
    return [c.build() for c in cases]
