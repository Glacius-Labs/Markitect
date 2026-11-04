"""Hidden cases of the money modules: ledger, billing, deposits, passes, promotions, scholarships, refunds, dunning,
lockers and finance. Written from the requirements document alone; every amount is computed by hand and stated literally.

The helpers below only build records of the shapes the requirements state and check my own arithmetic with asserts
(line sums, invoice sums, trial balance totals). They know nothing about the program.
"""
from __future__ import annotations

import pathlib
import sys

HERE = pathlib.Path(__file__).resolve().parent
if str(HERE) not in sys.path:
    sys.path.insert(0, str(HERE))
from kit import Case, world, has, ref, ANY  # noqa: E402,F401

ACCOUNTS = ["cash", "creditLiability", "depositLiability", "expenseGoodwill", "expenseMaker", "expensePayout",
            "expenseVolunteer", "expenseWriteOff", "inventory", "payable", "receivable", "revenueCourse", "revenueDeposits",
            "revenueEvents", "revenueGuests", "revenueJobs", "revenueLateFees", "revenueLockers", "revenueMaterial",
            "revenueMembership", "revenuePasses", "revenueRefunds", "revenueUsage"]
KINDS = ["membership", "usage", "material", "course", "deposit", "pass", "locker", "guest", "lateFee", "event"]
T0 = "2026-03-02T09:00Z"


# ---------------------------------------------------------------- record builders


def tb(rows: dict, debits: int, credits: int) -> dict:
    """A trial balance: rows maps an account to (debitCents, creditCents); accounts not named are 0/0."""
    assert set(rows) <= set(ACCOUNTS), set(rows) - set(ACCOUNTS)
    out = []
    for a in ACCOUNTS:
        d, k = rows.get(a, (0, 0))
        out.append({"account": a, "debitCents": d, "creditCents": k, "balanceCents": abs(d - k),
                    "side": "debit" if d > k else "credit" if k > d else "none"})
    assert sum(r["debitCents"] for r in out) == debits, (sum(r["debitCents"] for r in out), debits)
    assert sum(r["creditCents"] for r in out) == credits, (sum(r["creditCents"] for r in out), credits)
    assert debits == credits
    return {"rows": out, "totalDebitCents": debits, "totalCreditCents": credits}


def je(n: int, at: str, debit: str, credit: str, amount: int, memo: str) -> dict:
    return {"entryId": f"jrn-{n}", "at": at, "debitAccount": debit, "creditAccount": credit, "amountCents": amount, "memo": memo}


def chg(kind: str, ref_: str, quantity: int, unit: int, discount: int, amount: int, at: str) -> dict:
    return {"kind": kind, "ref": ref_, "quantity": quantity, "unitPriceCents": unit, "discountPercent": discount,
            "amountCents": amount, "at": at}


def mline(tier: str, unit: int, discount: int, amount: int, at: str) -> dict:
    return chg("membership", tier, 1, unit, discount, amount, at)


def inv(iid: str, member: str, month: str, issued: str, due: str, lines: list, total: int, credit: int, paid: int,
        balance: int, status: str, overdue: bool = False) -> dict:
    assert sum(line["amountCents"] for line in lines) == total, (iid, sum(line["amountCents"] for line in lines), total)
    assert credit + paid + balance == total, (iid, credit, paid, balance, total)
    assert (status == "paid") == (balance == 0), iid
    return {"invoiceId": iid, "memberId": member, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": credit, "paidCents": paid, "balanceCents": balance,
            "status": status, "overdue": overdue}


def simple(iid: str, member: str, month: str, issued: str, due: str, lines: list, credit: int = 0) -> dict:
    """An unpaid invoice made at generation: total from the lines, credit applied, rest open (or paid at 0)."""
    total = sum(line["amountCents"] for line in lines)
    balance = total - credit
    return inv(iid, member, month, issued, due, lines, total, credit, 0, balance, "paid" if balance == 0 else "open")


def acct(member: str, credit: int, open_: int, overdue: int, unbilled: int) -> dict:
    return {"account": {"memberId": member, "creditBalanceCents": credit, "openBalanceCents": open_,
                        "overdueBalanceCents": overdue, "unbilledCents": unbilled}}


def pay(pid: str, invoice: str, member: str, amount: int, applied: int, credited: int, at: str) -> dict:
    assert applied + credited == amount
    return {"paymentId": pid, "invoiceId": invoice, "memberId": member, "amountCents": amount, "appliedCents": applied,
            "creditedCents": credited, "at": at}


def deposit(did: str, member: str, purpose: str, amount: int, status: str, taken: str, closed: str | None) -> dict:
    return {"depositId": did, "memberId": member, "purpose": purpose, "amountCents": amount, "status": status,
            "takenAt": taken, "closedAt": closed}


def passrec(pid: str, member: str, product: str, left: int, issued: str, expires: str) -> dict:
    return {"passId": pid, "memberId": member, "productId": product, "minutesLeft": left, "issuedOn": issued, "expiresOn": expires}


def promo(code: str, percent: int, scope: str, maximum: int, until: str, redemptions: int, ended: bool) -> dict:
    return {"code": code, "percent": percent, "scope": scope, "maxRedemptions": maximum, "validUntil": until,
            "redemptions": redemptions, "ended": ended}


def redemption(code: str, member: str, percent: int, at: str, used_at: str | None = None, used_on: str | None = None) -> dict:
    return {"code": code, "memberId": member, "percent": percent, "redeemedAt": at, "usedAt": used_at, "usedOn": used_on}


def award(aid: str, fund: str, member: str, percent: int, status: str) -> dict:
    return {"awardId": aid, "fundId": fund, "memberId": member, "percent": percent, "status": status}


def refund(rid: str, invoice: str, line: int, amount: int, reason: str, at: str) -> dict:
    return {"refundId": rid, "invoiceId": invoice, "lineNumber": line, "amountCents": amount, "reason": reason, "at": at}


def rental(rid: str, locker: str, member: str, status: str, started: str, ended: str | None) -> dict:
    return {"rentalId": rid, "lockerId": locker, "memberId": member, "status": status, "startedAt": started, "endedAt": ended}


def case_rec(n: int, invoice: str, member: str, noticed: str | None, fee: str | None, suspended: str | None) -> dict:
    return {"caseId": f"dun-{n}", "invoiceId": invoice, "memberId": member, "noticedAt": noticed, "feeChargedAt": fee,
            "suspendedAt": suspended}


def notice(n: int, member: str, kind: str, invoice: str, at: str) -> dict:
    subject = {"dunning": f"Invoice {invoice} is overdue",
               "suspension": f"Membership suspended: invoice {invoice} is overdue"}[kind]
    return {"noticeId": f"ntc-{n}", "memberId": member, "kind": kind, "subject": subject, "ref": invoice, "at": at,
            "acknowledgedAt": None}


def action(invoice: str, step: str) -> dict:
    return {"invoiceId": invoice, "step": step}


def resv(rid: str, member: str, machine: str, start: str, end: str, status: str, cin: str | None = None,
         cout: str | None = None, rate: int | None = None, metered: int | None = None, amount: int | None = None) -> dict:
    return {"reservationId": rid, "memberId": member, "machineId": machine, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": metered,
            "amountCents": amount}


def member_rec(mid: str, name: str, tier: str, standing: str, registered: str) -> dict:
    return {"member": {"memberId": mid, "name": name, "tier": tier, "standing": standing, "registeredAt": registered}}


def kinds(invoiced: dict, refunded: dict | None = None, total: int = 0) -> list:
    refunded = refunded or {}
    assert set(invoiced) <= set(KINDS) and set(refunded) <= set(KINDS)
    rows = [{"kind": k, "invoicedCents": invoiced.get(k, 0), "refundedCents": refunded.get(k, 0)} for k in KINDS]
    assert sum(r["invoicedCents"] for r in rows) == total, (sum(r["invoicedCents"] for r in rows), total)
    return rows


def arow(iid: str, member: str, balance: int, days: int) -> dict:
    return {"invoiceId": iid, "memberId": member, "balanceCents": balance, "daysOverdue": days}


def aged(date: str, rows: list, current: int, d30: int, d60: int, over: int) -> dict:
    assert sum(r["balanceCents"] for r in rows) == current + d30 + d60 + over
    for r in rows:
        d = r["daysOverdue"]
        bucket = 0 if d == 0 else 1 if d <= 30 else 2 if d <= 60 else 3
        assert bucket in range(4)
    return {"date": date, "rows": rows, "currentCents": current, "days1to30Cents": d30, "days31to60Cents": d60,
            "over60Cents": over}


def srow(member: str, invoice: str | None, total: int, balance: int) -> dict:
    return {"memberId": member, "invoiceId": invoice, "totalCents": total, "balanceCents": balance}


def stmt(household: str, month: str, rows: list) -> dict:
    return {"householdId": household, "month": month, "rows": rows,
            "totalCents": sum(r["totalCents"] for r in rows), "balanceCents": sum(r["balanceCents"] for r in rows)}


def adj(c: Case, at: str, debit: str, credit: str, amount: int, memo: str, n: int) -> None:
    c.ok("postAdjustment", "amy", at, {"debitAccount": debit, "creditAccount": credit, "amountCents": amount, "memo": memo},
         {"entry": je(n, at, debit, credit, amount, memo)}, new=f"jrn-{n}")


def session(c: Case, member: str, rid: str, start: str, end: str, create_at: str, out_at: str, machine: str = "L1") -> None:
    """A reservation, its check-in at the start and its check-out: only sets up (results not compared)."""
    c.ok("createReservation", member, create_at, {"memberId": member, "machineId": machine, "start": start, "end": end}, new=rid)
    c.ok("checkIn", member, start, {"reservationId": rid})
    c.ok("checkOut", member, out_at, {"reservationId": rid})


def generate(c: Case, month: str, at: str, invoices: list | None = None) -> None:
    c.ok("generateInvoices", "sam", at, {"month": month},
         None if invoices is None else {"month": month, "invoices": invoices})


def pays(c: Case, actor: str, at: str, invoice: str, amount: int, new: str, result: dict | None = None) -> None:
    c.ok("recordPayment", actor, at, {"invoiceId": invoice, "amountCents": amount}, result, new=new)


# the march data shared by many cases: world gives ada a course charge of 5000 cents at T0.
MARCH_DUE = "2026-04-15T08:00Z"
MARCH_AT = "2026-04-01T08:00Z"


def ada_course() -> dict:
    return chg("course", "laser-101", 1, 5000, 0, 5000, T0)


def march_plain(issued: str, due: str) -> list:
    """Invoices for 2026-03 when nobody has any charge but ada's course: ada 4900 + 5000, bob 2500, eve 9900."""
    return [simple("inv-1", "ada", "2026-03", issued, due, [mline("plus", 4900, 0, 4900, issued), ada_course()]),
            simple("inv-2", "bob", "2026-03", issued, due, [mline("basic", 2500, 0, 2500, issued)]),
            simple("inv-3", "eve", "2026-03", issued, due, [mline("pro", 9900, 0, 9900, issued)])]


# ---------------------------------------------------------------- ledger


def b_ledger_1() -> dict:
    c = Case("B-ledger-1", "B-ledger", "ledger: post adjustment, get trial balance, list journal entries by month")
    world(c)
    t = "2026-03-02T10:00Z"
    c.ok("getTrialBalance", "amy", t, {}, tb({}, 0, 0))
    c.ok("listJournalEntries", "amy", t, {"month": "2026-03"}, {"month": "2026-03", "entries": []})
    adj(c, t, "cash", "revenueJobs", 700, "walk-in", 1)
    adj(c, t, "expenseGoodwill", "cash", 200, "coffee", 2)
    c.ok("getTrialBalance", "amy", t, {}, tb({"cash": (700, 200), "expenseGoodwill": (200, 0), "revenueJobs": (0, 700)}, 900, 900))
    adj(c, "2026-03-31T23:59Z", "inventory", "payable", 50, "late", 3)
    adj(c, "2026-04-01T00:00Z", "payable", "cash", 20, "early", 4)
    at = "2026-04-01T00:00Z"
    c.ok("listJournalEntries", "amy", at, {"month": "2026-03"},
         {"month": "2026-03", "entries": [je(1, t, "cash", "revenueJobs", 700, "walk-in"),
                                          je(2, t, "expenseGoodwill", "cash", 200, "coffee"),
                                          je(3, "2026-03-31T23:59Z", "inventory", "payable", 50, "late")]})
    c.ok("listJournalEntries", "amy", at, {"month": "2026-04"},
         {"month": "2026-04", "entries": [je(4, at, "payable", "cash", 20, "early")]})
    c.ok("listJournalEntries", "amy", at, {"month": "2026-02"}, {"month": "2026-02", "entries": []})
    c.ok("listJournalEntries", "amy", at, {"month": "2026-05"}, {"month": "2026-05", "entries": []})
    c.refuse("postAdjustment", "amy", at, {"debitAccount": "cash", "creditAccount": "cash", "amountCents": 5, "memo": "same"},
             "same_account")
    # the refusal consumed no number
    adj(c, at, "revenueUsage", "revenueMembership", 1, "x", 5)
    c.ok("getTrialBalance", "amy", at, {},
         tb({"cash": (700, 220), "expenseGoodwill": (200, 0), "inventory": (50, 0), "payable": (20, 50),
             "revenueJobs": (0, 700), "revenueMembership": (0, 1), "revenueUsage": (1, 0)}, 971, 971))
    c.audit_log(at)
    return c.build()


def b_ledger_2() -> dict:
    c = Case("B-ledger-2", "B-ledger", "ledger: adjustment limits, refusal order, permissions; refusals change nothing")
    world(c)
    t = "2026-03-02T10:00Z"
    adj(c, t, "cash", "revenueJobs", 100000000, "max", 1)
    c.ok("getTrialBalance", "amy", t, {}, tb({"cash": (100000000, 0), "revenueJobs": (0, 100000000)}, 100000000, 100000000))
    good = {"debitAccount": "cash", "creditAccount": "revenueJobs", "amountCents": 5, "memo": "m"}
    c.refuse("postAdjustment", "amy", t, {**good, "amountCents": 100000001}, "invalid_argument")
    c.refuse("postAdjustment", "amy", t, {**good, "amountCents": 0}, "invalid_argument")
    c.refuse("postAdjustment", "amy", t, {**good, "amountCents": -5}, "invalid_argument")
    c.refuse("postAdjustment", "amy", t, {**good, "amountCents": "5"}, "invalid_argument")
    c.refuse("postAdjustment", "amy", t, {**good, "debitAccount": "bank"}, "invalid_argument")
    c.refuse("postAdjustment", "amy", t, {**good, "memo": "   "}, "invalid_argument")
    c.refuse("postAdjustment", "amy", t, {"debitAccount": "cash", "creditAccount": "revenueJobs", "amountCents": 5}, "invalid_argument")
    # argument checks come before the operation's own refusal
    c.refuse("postAdjustment", "amy", t, {**good, "creditAccount": "cash", "amountCents": 0}, "invalid_argument")
    # role check comes before the operation's own refusal
    c.refuse("postAdjustment", "sam", t, good, "forbidden")
    c.refuse("postAdjustment", "sam", t, {**good, "creditAccount": "cash"}, "forbidden")
    c.refuse("postAdjustment", "ghost", t, {**good, "creditAccount": "cash"}, "unknown_actor")
    c.refuse("postAdjustment", "ada", t, good, "forbidden")
    c.refuse("getTrialBalance", "sam", t, {}, "forbidden")
    c.refuse("listJournalEntries", "sam", t, {"month": "2026-03"}, "forbidden")
    c.refuse("listJournalEntries", "ada", t, {"month": "2026-03"}, "forbidden")
    c.refuse("listJournalEntries", "amy", t, {"month": "2026-13"}, "invalid_argument")
    c.refuse("postAdjustment", "amy", t, {**good, "creditAccount": "cash", "debitAccount": "cash"}, "same_account")
    # nothing changed; the next entry is jrn-2
    adj(c, t, "revenueRefunds", "expenseGoodwill", 5, "next", 2)
    c.ok("listJournalEntries", "amy", t, {"month": "2026-03"},
         {"month": "2026-03", "entries": [je(1, t, "cash", "revenueJobs", 100000000, "max"),
                                          je(2, t, "revenueRefunds", "expenseGoodwill", 5, "next")]})
    c.ok("getTrialBalance", "amy", t, {},
         tb({"cash": (100000000, 0), "revenueJobs": (0, 100000000), "revenueRefunds": (5, 0), "expenseGoodwill": (0, 5)},
            100000005, 100000005))
    c.audit_log(t)
    return c.build()


# ---------------------------------------------------------------- billing


def b_billing_1() -> dict:
    c = Case("B-billing-1", "B-billing", "billing: invoice lines, order, credit, due date, month end, empty month, account")
    world(c)
    c.ok("createReservation", "ada", "2026-03-02T09:30Z",
         {"memberId": "ada", "machineId": "L1", "start": "2026-03-03T10:00Z", "end": "2026-03-03T11:00Z"}, new="res-1")
    c.ok("checkIn", "ada", "2026-03-03T10:00Z", {"reservationId": "res-1"})
    usage = chg("usage", "res-1", 37, 45, 10, 1499, "2026-03-03T10:37Z")
    c.ok("checkOut", "ada", "2026-03-03T10:37Z", {"reservationId": "res-1"},
         {"reservation": resv("res-1", "ada", "L1", "2026-03-03T10:00Z", "2026-03-03T11:00Z", "completed",
                              "2026-03-03T10:00Z", "2026-03-03T10:37Z", 45, 37, 1499), "charge": usage})
    c.ok("purchaseMaterial", "sam", "2026-03-05T12:00Z", {"memberId": "bob", "materialId": "ply", "quantity": 3}, None, new="pur-1")
    c.ok("issueCredit", "amy", "2026-03-10T09:00Z", {"memberId": "ada", "amountCents": 2000, "reason": "goodwill"},
         {"memberId": "ada", "amountCents": 2000, "creditBalanceCents": 2000})
    t = "2026-03-10T09:00Z"
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, acct("ada", 2000, 0, 0, 6499))
    c.ok("getAccount", "sam", t, {"memberId": "bob"}, acct("bob", 0, 0, 0, 450))
    c.ok("getAccount", "sam", t, {"memberId": "eve"}, acct("eve", 0, 0, 0, 0))
    c.ok("listInvoices", "ada", t, {"memberId": "ada"}, {"memberId": "ada", "invoices": []})
    # one minute before the month has ended
    c.refuse("generateInvoices", "sam", "2026-03-31T23:59Z", {"month": "2026-03"}, "month_not_ended")
    c.ok("getAccount", "ada", "2026-03-31T23:59Z", {"memberId": "ada"}, acct("ada", 2000, 0, 0, 6499))
    issued, due = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    ada_lines = [mline("plus", 4900, 0, 4900, issued), chg("course", "laser-101", 1, 5000, 0, 5000, T0), usage]
    bob_lines = [mline("basic", 2500, 0, 2500, issued), chg("material", "pur-1", 3, 150, 0, 450, "2026-03-05T12:00Z")]
    invoices = [simple("inv-1", "ada", "2026-03", issued, due, ada_lines, credit=2000),
                simple("inv-2", "bob", "2026-03", issued, due, bob_lines),
                simple("inv-3", "eve", "2026-03", issued, due, [mline("pro", 9900, 0, 9900, issued)])]
    assert invoices[0]["totalCents"] == 11399 and invoices[0]["balanceCents"] == 9399
    c.ok("generateInvoices", "sam", issued, {"month": "2026-03"}, {"month": "2026-03", "invoices": invoices})
    c.refuse("generateInvoices", "sam", issued, {"month": "2026-03"}, "already_invoiced")
    c.refuse("generateInvoices", "ada", issued, {"month": "2026-04"}, "forbidden")
    c.refuse("generateInvoices", "sam", issued, {"month": "2026-04"}, "month_not_ended")
    c.ok("listInvoices", "ada", issued, {"memberId": "ada"}, {"memberId": "ada", "invoices": [invoices[0]]})
    c.ok("getAccount", "ada", issued, {"memberId": "ada"}, acct("ada", 0, 9399, 0, 0))
    c.ok("getAccount", "bob", issued, {"memberId": "bob"}, acct("bob", 0, 2950, 0, 0))
    # a month with no member to invoice is still marked invoiced
    c.ok("generateInvoices", "amy", issued, {"month": "2026-02"}, {"month": "2026-02", "invoices": []})
    c.refuse("generateInvoices", "amy", issued, {"month": "2026-02"}, "already_invoiced")
    c.audit_log(issued)
    return c.build()


def b_billing_2() -> dict:
    c = Case("B-billing-2", "B-billing", "billing: household discount is judged at the request, whatever the month; ref is the current tier")
    world(c)
    c.ok("registerMember", "sam", T0, {"memberId": "cal", "name": "Cal Cole", "tier": "basic"})
    c.ok("createHousehold", "sam", T0, {"householdId": "h1", "name": "Home", "payerId": "bob"})
    c.ok("addHouseholdMember", "sam", T0, {"householdId": "h1", "memberId": "eve"})
    c.ok("addHouseholdMember", "sam", T0, {"householdId": "h1", "memberId": "ada"})
    # April is invoiced first; the household has 3 members now
    i1, d1 = "2026-05-01T00:00Z", "2026-05-15T00:00Z"
    april = [simple("inv-1", "ada", "2026-04", i1, d1, [mline("plus", 4900, 10, 4410, i1)]),
             simple("inv-2", "bob", "2026-04", i1, d1, [mline("basic", 2500, 10, 2250, i1)]),
             simple("inv-3", "cal", "2026-04", i1, d1, [mline("basic", 2500, 0, 2500, i1)]),
             simple("inv-4", "eve", "2026-04", i1, d1, [mline("pro", 9900, 10, 8910, i1)])]
    c.ok("generateInvoices", "sam", i1, {"month": "2026-04"}, {"month": "2026-04", "invoices": april})
    c.ok("removeHouseholdMember", "sam", "2026-05-02T09:00Z", {"householdId": "h1", "memberId": "eve"})
    c.ok("changeTier", "sam", "2026-05-03T09:00Z", {"memberId": "cal", "tier": "plus"})
    # March is invoiced later: the household has 2 members now, so no discount; cal's line carries her current tier
    i2, d2 = "2026-05-04T00:00Z", "2026-05-18T00:00Z"
    march = [simple("inv-5", "ada", "2026-03", i2, d2, [mline("plus", 4900, 0, 4900, i2), ada_course()]),
             simple("inv-6", "bob", "2026-03", i2, d2, [mline("basic", 2500, 0, 2500, i2)]),
             simple("inv-7", "cal", "2026-03", i2, d2, [mline("plus", 4900, 0, 4900, i2)]),
             simple("inv-8", "eve", "2026-03", i2, d2, [mline("pro", 9900, 0, 9900, i2)])]
    c.ok("generateInvoices", "sam", i2, {"month": "2026-03"}, {"month": "2026-03", "invoices": march})
    c.audit_log(i2)
    return c.build()


def b_billing_3() -> dict:
    c = Case("B-billing-3", "B-billing", "billing: payments (partial, exact, overpaying), overdue flag, credit applied only to later invoices")
    world(c)
    generate(c, "2026-03", MARCH_AT)
    due = MARCH_DUE
    ada_lines = [mline("plus", 4900, 0, 4900, MARCH_AT), ada_course()]
    # partial payment by the member
    t = "2026-04-05T10:00Z"
    pays(c, "ada", t, "inv-1", 4000, "pay-1",
         {"payment": pay("pay-1", "inv-1", "ada", 4000, 4000, 0, t),
          "invoice": inv("inv-1", "ada", "2026-03", MARCH_AT, due, ada_lines, 9900, 0, 4000, 5900, "open"),
          "creditBalanceCents": 0})
    c.refuse("recordPayment", "bob", "2026-04-05T10:05Z", {"invoiceId": "inv-1", "amountCents": 100}, "forbidden")
    # the exact balance by staff for the member
    t = "2026-04-06T09:00Z"
    pays(c, "sam", t, "inv-2", 2500, "pay-2",
         {"payment": pay("pay-2", "inv-2", "bob", 2500, 2500, 0, t),
          "invoice": inv("inv-2", "bob", "2026-03", MARCH_AT, due, [mline("basic", 2500, 0, 2500, MARCH_AT)], 2500, 0, 2500, 0, "paid"),
          "creditBalanceCents": 0})
    c.refuse("recordPayment", "sam", "2026-04-06T09:05Z", {"invoiceId": "inv-2", "amountCents": 100}, "invoice_paid")
    c.refuse("recordPayment", "sam", "2026-04-06T09:05Z", {"invoiceId": "inv-1", "amountCents": 100000001}, "invalid_argument")
    c.refuse("recordPayment", "sam", "2026-04-06T09:05Z", {"invoiceId": "inv-1", "amountCents": 0}, "invalid_argument")
    c.refuse("recordPayment", "sam", "2026-04-06T09:05Z", {"invoiceId": "inv-9", "amountCents": 1}, "not_found")
    # an overpayment: 5900 applied, 100 credited; the refusals above used no payment number
    t = "2026-04-07T10:00Z"
    pays(c, "ada", t, "inv-1", 6000, "pay-3",
         {"payment": pay("pay-3", "inv-1", "ada", 6000, 5900, 100, t),
          "invoice": inv("inv-1", "ada", "2026-03", MARCH_AT, due, ada_lines, 9900, 0, 9900, 0, "paid"),
          "creditBalanceCents": 100})
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, acct("ada", 100, 0, 0, 0))
    # credit issued while an invoice is open does not touch that invoice
    c.ok("issueCredit", "amy", "2026-04-10T09:00Z", {"memberId": "eve", "amountCents": 500, "reason": "late desk"},
         {"memberId": "eve", "amountCents": 500, "creditBalanceCents": 500})
    eve_line = [mline("pro", 9900, 0, 9900, MARCH_AT)]
    c.ok("listInvoices", "eve", "2026-04-15T07:59Z", {"memberId": "eve"},
         {"memberId": "eve", "invoices": [inv("inv-3", "eve", "2026-03", MARCH_AT, due, eve_line, 9900, 0, 0, 9900, "open", False)]})
    c.ok("listInvoices", "eve", "2026-04-15T08:00Z", {"memberId": "eve"},
         {"memberId": "eve", "invoices": [inv("inv-3", "eve", "2026-03", MARCH_AT, due, eve_line, 9900, 0, 0, 9900, "open", True)]})
    c.ok("getAccount", "eve", "2026-04-15T08:00Z", {"memberId": "eve"}, acct("eve", 500, 9900, 9900, 0))
    # paying after the due date is allowed; the credit stays
    t = "2026-04-16T09:00Z"
    pays(c, "eve", t, "inv-3", 9900, "pay-4",
         {"payment": pay("pay-4", "inv-3", "eve", 9900, 9900, 0, t),
          "invoice": inv("inv-3", "eve", "2026-03", MARCH_AT, due, eve_line, 9900, 0, 9900, 0, "paid", False),
          "creditBalanceCents": 500})
    # credit is applied to the next invoices generated
    i2, d2 = "2026-05-01T08:00Z", "2026-05-15T08:00Z"
    april = [simple("inv-4", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2)], credit=100),
             simple("inv-5", "bob", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2)]),
             simple("inv-6", "eve", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)], credit=500)]
    assert april[0]["balanceCents"] == 4800 and april[2]["balanceCents"] == 9400
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": april})
    c.ok("getAccount", "ada", i2, {"memberId": "ada"}, acct("ada", 0, 4800, 0, 0))
    c.ok("getAccount", "eve", i2, {"memberId": "eve"}, acct("eve", 0, 9400, 0, 0))
    c.audit_log(i2)
    return c.build()


def b_billing_4() -> dict:
    c = Case("B-billing-4", "B-billing", "billing: credit capped at the total (paid at once), credit equal to the total, ledger after")
    world(c)
    c.ok("issueCredit", "amy", "2026-03-02T10:00Z", {"memberId": "bob", "amountCents": 10000, "reason": "prepaid"},
         {"memberId": "bob", "amountCents": 10000, "creditBalanceCents": 10000})
    i1 = "2026-04-01T00:00Z"
    d1 = "2026-04-15T00:00Z"
    march = [simple("inv-1", "ada", "2026-03", i1, d1, [mline("plus", 4900, 0, 4900, i1), ada_course()]),
             simple("inv-2", "bob", "2026-03", i1, d1, [mline("basic", 2500, 0, 2500, i1)], credit=2500),
             simple("inv-3", "eve", "2026-03", i1, d1, [mline("pro", 9900, 0, 9900, i1)])]
    assert march[1]["status"] == "paid" and march[1]["paidCents"] == 0 and march[1]["balanceCents"] == 0
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": march})
    c.ok("getTrialBalance", "amy", i1, {},
         tb({"creditLiability": (2500, 10000), "expenseGoodwill": (10000, 0), "receivable": (22300, 2500),
             "revenueCourse": (0, 5000), "revenueMembership": (0, 17300)}, 34800, 34800))
    c.refuse("recordPayment", "sam", i1, {"invoiceId": "inv-2", "amountCents": 100}, "invoice_paid")
    c.ok("getAccount", "bob", i1, {"memberId": "bob"}, acct("bob", 7500, 0, 0, 0))
    # credit exactly equal to the total
    c.ok("issueCredit", "amy", "2026-04-20T09:00Z", {"memberId": "ada", "amountCents": 4900, "reason": "refund of a trip"},
         {"memberId": "ada", "amountCents": 4900, "creditBalanceCents": 4900})
    i2, d2 = "2026-05-01T00:00Z", "2026-05-15T00:00Z"
    april = [simple("inv-4", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2)], credit=4900),
             simple("inv-5", "bob", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2)], credit=2500),
             simple("inv-6", "eve", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)])]
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": april})
    c.ok("getAccount", "ada", i2, {"memberId": "ada"}, acct("ada", 0, 9900, 9900, 0))
    c.ok("getAccount", "bob", i2, {"memberId": "bob"}, acct("bob", 5000, 0, 0, 0))
    c.ok("getTrialBalance", "amy", i2, {},
         tb({"creditLiability": (9900, 14900), "expenseGoodwill": (14900, 0), "receivable": (39600, 9900),
             "revenueCourse": (0, 5000), "revenueMembership": (0, 34600)}, 64400, 64400))
    c.audit_log(i2)
    return c.build()


# ---------------------------------------------------------------- deposits


def b_deposits_1() -> dict:
    c = Case("B-deposits-1", "B-deposits", "deposits: take, list, refusals in order, nothing changes by a refusal")
    world(c)
    t = "2026-03-02T10:00Z"
    dep1 = deposit("dep-1", "bob", "bench key", 3000, "held", t, None)
    c.ok("takeDeposit", "sam", t, {"memberId": "bob", "purpose": "bench key", "amountCents": 3000},
         {"deposit": dep1, "charge": chg("deposit", "dep-1", 1, 3000, 0, 3000, t)}, new="dep-1")
    t2 = "2026-03-02T10:01Z"
    dep2 = deposit("dep-2", "bob", "chair", 500, "held", t2, None)
    c.ok("takeDeposit", "amy", t2, {"memberId": "bob", "purpose": "chair", "amountCents": 500},
         {"deposit": dep2, "charge": chg("deposit", "dep-2", 1, 500, 0, 500, t2)}, new="dep-2")
    c.ok("listDeposits", "ada", t2, {"memberId": "ada"}, {"memberId": "ada", "deposits": []})
    c.ok("listDeposits", "sam", t2, {"memberId": "bob"}, {"memberId": "bob", "deposits": [dep1, dep2]})
    t3 = "2026-03-02T10:03Z"
    c.refuse("takeDeposit", "ada", t3, {"memberId": "ada", "purpose": "x", "amountCents": 100}, "forbidden")
    c.refuse("takeDeposit", "sam", t3, {"memberId": "zed", "purpose": "x", "amountCents": 100}, "not_found")
    c.refuse("takeDeposit", "sam", t3, {"memberId": "sue", "purpose": "x", "amountCents": 100}, "not_found")
    c.refuse("takeDeposit", "sam", t3, {"memberId": "bob", "purpose": "x", "amountCents": 0}, "invalid_argument")
    c.refuse("takeDeposit", "sam", t3, {"memberId": "bob", "purpose": "x", "amountCents": 100000001}, "invalid_argument")
    c.refuse("takeDeposit", "sam", t3, {"memberId": "bob", "purpose": " ", "amountCents": 100}, "invalid_argument")
    # the largest deposit is taken
    t4 = "2026-03-02T10:04Z"
    dep3 = deposit("dep-3", "eve", "max", 100000000, "held", t4, None)
    c.ok("takeDeposit", "sam", t4, {"memberId": "eve", "purpose": "max", "amountCents": 100000000},
         {"deposit": dep3, "charge": chg("deposit", "dep-3", 1, 100000000, 0, 100000000, t4)}, new="dep-3")
    # suspended members leave no deposit; standing is no reason to refuse the rest
    t5 = "2026-03-02T10:10Z"
    c.ok("setMemberStanding", "sam", t5, {"memberId": "bob", "standing": "suspended"})
    c.refuse("takeDeposit", "sam", t5, {"memberId": "bob", "purpose": "again", "amountCents": 100}, "member_suspended")
    c.ok("setMemberStanding", "sam", "2026-03-02T10:15Z", {"memberId": "bob", "standing": "good"})
    t6 = "2026-03-02T10:20Z"
    # an unpaid deposit: its charge is on no invoice yet
    c.refuse("releaseDeposit", "sam", t6, {"depositId": "dep-1"}, "deposit_unpaid")
    c.refuse("releaseDeposit", "amy", t6, {"depositId": "dep-1"}, "deposit_unpaid")
    c.refuse("forfeitDeposit", "amy", t6, {"depositId": "dep-1", "reason": "broken"}, "deposit_unpaid")
    c.refuse("releaseDeposit", "sam", t6, {"depositId": "dep-9"}, "not_found")
    c.refuse("releaseDeposit", "ada", t6, {"depositId": "dep-1"}, "forbidden")
    c.refuse("forfeitDeposit", "sam", t6, {"depositId": "dep-1", "reason": "broken"}, "forbidden")
    c.refuse("forfeitDeposit", "amy", t6, {"depositId": "dep-9", "reason": "broken"}, "not_found")
    c.refuse("forfeitDeposit", "amy", t6, {"depositId": "dep-1", "reason": ""}, "invalid_argument")
    c.refuse("listDeposits", "bob", t6, {"memberId": "ada"}, "forbidden")
    c.ok("listDeposits", "bob", t6, {"memberId": "bob"}, {"memberId": "bob", "deposits": [dep1, dep2]})
    c.audit_log(t6)
    return c.build()


def b_deposits_2() -> dict:
    c = Case("B-deposits-2", "B-deposits", "deposits: paid by credit at invoicing, released into credit, forfeited; postings")
    world(c)
    c.ok("issueCredit", "amy", "2026-03-02T10:00Z", {"memberId": "bob", "amountCents": 10000, "reason": "prepaid"})
    t1, t2 = "2026-03-02T10:05Z", "2026-03-02T10:10Z"
    c.ok("takeDeposit", "sam", t1, {"memberId": "bob", "purpose": "tool loan", "amountCents": 2000}, None, new="dep-1")
    c.ok("takeDeposit", "sam", t2, {"memberId": "eve", "purpose": "tool loan", "amountCents": 2000}, None, new="dep-2")
    i1, d1 = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    dl1, dl2 = chg("deposit", "dep-1", 1, 2000, 0, 2000, t1), chg("deposit", "dep-2", 1, 2000, 0, 2000, t2)
    march = [simple("inv-1", "ada", "2026-03", i1, d1, [mline("plus", 4900, 0, 4900, i1), ada_course()]),
             simple("inv-2", "bob", "2026-03", i1, d1, [mline("basic", 2500, 0, 2500, i1), dl1], credit=4500),
             simple("inv-3", "eve", "2026-03", i1, d1, [mline("pro", 9900, 0, 9900, i1), dl2])]
    assert march[1]["status"] == "paid" and march[2]["totalCents"] == 11900
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": march})
    # bob's deposit was paid by credit alone: released at once; credit 10000 - 4500 + 2000 = 7500
    t3 = "2026-04-01T00:05Z"
    c.ok("releaseDeposit", "sam", t3, {"depositId": "dep-1"},
         {"deposit": deposit("dep-1", "bob", "tool loan", 2000, "released", t1, t3), "creditBalanceCents": 7500})
    c.refuse("releaseDeposit", "sam", t3, {"depositId": "dep-1"}, "invalid_state")
    c.refuse("forfeitDeposit", "amy", t3, {"depositId": "dep-1", "reason": "late"}, "invalid_state")
    # eve's invoice is open: her deposit is not paid
    c.refuse("forfeitDeposit", "amy", t3, {"depositId": "dep-2", "reason": "lost"}, "deposit_unpaid")
    t4 = "2026-04-02T09:00Z"
    pays(c, "eve", t4, "inv-3", 11900, "pay-1")
    t5 = "2026-04-02T09:05Z"
    c.ok("forfeitDeposit", "amy", t5, {"depositId": "dep-2", "reason": "lost"},
         {"deposit": deposit("dep-2", "eve", "tool loan", 2000, "forfeited", t2, t5)})
    c.ok("getAccount", "eve", t5, {"memberId": "eve"}, acct("eve", 0, 0, 0, 0))
    c.ok("getAccount", "bob", t5, {"memberId": "bob"}, acct("bob", 7500, 0, 0, 0))
    c.refuse("releaseDeposit", "sam", t5, {"depositId": "dep-2"}, "invalid_state")
    c.ok("listDeposits", "eve", t5, {"memberId": "eve"},
         {"memberId": "eve", "deposits": [deposit("dep-2", "eve", "tool loan", 2000, "forfeited", t2, t5)]})
    c.ok("listJournalEntries", "amy", t5, {"month": "2026-04"},
         {"month": "2026-04", "entries": [
             je(2, i1, "receivable", "revenueMembership", 4900, "inv-1"),
             je(3, i1, "receivable", "revenueCourse", 5000, "inv-1"),
             je(4, i1, "receivable", "revenueMembership", 2500, "inv-2"),
             je(5, i1, "receivable", "depositLiability", 2000, "inv-2"),
             je(6, i1, "creditLiability", "receivable", 4500, "inv-2"),
             je(7, i1, "receivable", "revenueMembership", 9900, "inv-3"),
             je(8, i1, "receivable", "depositLiability", 2000, "inv-3"),
             je(9, t3, "depositLiability", "creditLiability", 2000, "dep-1"),
             je(10, t4, "cash", "receivable", 11900, "pay-1"),
             je(11, t5, "depositLiability", "revenueDeposits", 2000, "dep-2")]})
    c.ok("getTrialBalance", "amy", t5, {},
         tb({"cash": (11900, 0), "creditLiability": (4500, 12000), "depositLiability": (4000, 4000),
             "expenseGoodwill": (10000, 0), "receivable": (26300, 16400), "revenueCourse": (0, 5000),
             "revenueDeposits": (0, 2000), "revenueMembership": (0, 17300)}, 56700, 56700))
    c.audit_log(t5)
    return c.build()


# ---------------------------------------------------------------- passes


def b_passes_1() -> dict:
    c = Case("B-passes-1", "B-passes", "passes: products, sale dates and charge, list, refusals")
    world(c)
    prod = {"productId": "p60", "name": "Sixty minutes", "minutes": 60, "priceCents": 3000, "validDays": 30}
    c.ok("addPassProduct", "amy", T0, dict(prod), {"product": prod})
    c.refuse("addPassProduct", "amy", T0, {**prod, "name": "Other", "priceCents": 1}, "already_exists")
    c.refuse("addPassProduct", "sam", T0, {**prod, "productId": "p-x"}, "forbidden")
    c.refuse("addPassProduct", "amy", T0, {**prod, "productId": "p-x", "minutes": 0}, "invalid_argument")
    c.refuse("addPassProduct", "amy", T0, {**prod, "productId": "p-x", "minutes": 100001}, "invalid_argument")
    c.refuse("addPassProduct", "amy", T0, {**prod, "productId": "p-x", "priceCents": 0}, "invalid_argument")
    c.refuse("addPassProduct", "amy", T0, {**prod, "productId": "p-x", "priceCents": 100000001}, "invalid_argument")
    c.refuse("addPassProduct", "amy", T0, {**prod, "productId": "p-x", "validDays": 0}, "invalid_argument")
    c.refuse("addPassProduct", "amy", T0, {**prod, "productId": "p-x", "validDays": 3651}, "invalid_argument")
    big = {"productId": "pmax", "name": "Largest", "minutes": 100000, "priceCents": 100000000, "validDays": 3650}
    c.ok("addPassProduct", "amy", T0, dict(big), {"product": big})
    year = {"productId": "year", "name": "Year", "minutes": 120, "priceCents": 9000, "validDays": 365}
    c.ok("addPassProduct", "amy", T0, dict(year), {"product": year})
    day = {"productId": "day", "name": "Day", "minutes": 10, "priceCents": 1, "validDays": 1}
    c.ok("addPassProduct", "amy", T0, dict(day), {"product": day})
    # ada: 60 minutes for 30 days from 2026-03-02
    t1 = "2026-03-02T11:00Z"
    p1 = passrec("pas-1", "ada", "p60", 60, "2026-03-02", "2026-04-01")
    c.ok("sellPass", "sam", t1, {"memberId": "ada", "productId": "p60"},
         {"pass": p1, "charge": chg("pass", "pas-1", 1, 3000, 0, 3000, t1)}, new="pas-1")
    # the largest product, 3650 days: 2036-02-28 (leap days 2028, 2032 and 2036 lie between)
    t2 = "2026-03-02T23:59Z"
    p2 = passrec("pas-2", "ada", "pmax", 100000, "2026-03-02", "2036-02-28")
    c.ok("sellPass", "amy", t2, {"memberId": "ada", "productId": "pmax"},
         {"pass": p2, "charge": chg("pass", "pas-2", 1, 100000000, 0, 100000000, t2)}, new="pas-2")
    # a member buys her own pass; 30 days from 2026-03-03 is 2026-04-02
    t3 = "2026-03-03T00:00Z"
    p3 = passrec("pas-3", "bob", "p60", 60, "2026-03-03", "2026-04-02")
    c.ok("sellPass", "bob", t3, {"memberId": "bob", "productId": "p60"},
         {"pass": p3, "charge": chg("pass", "pas-3", 1, 3000, 0, 3000, t3)}, new="pas-3")
    t4 = "2026-03-03T00:05Z"
    p4 = passrec("pas-4", "bob", "year", 120, "2026-03-03", "2027-03-03")
    c.ok("sellPass", "sam", t4, {"memberId": "bob", "productId": "year"},
         {"pass": p4, "charge": chg("pass", "pas-4", 1, 9000, 0, 9000, t4)}, new="pas-4")
    t5 = "2026-03-03T00:10Z"
    c.refuse("sellPass", "ada", t5, {"memberId": "bob", "productId": "p60"}, "forbidden")
    c.ok("setMemberStanding", "sam", t5, {"memberId": "eve", "standing": "suspended"})
    c.refuse("sellPass", "eve", t5, {"memberId": "eve", "productId": "p60"}, "member_suspended")
    c.refuse("sellPass", "sam", t5, {"memberId": "bob", "productId": "nope"}, "not_found")
    c.refuse("sellPass", "sam", t5, {"memberId": "zed", "productId": "p60"}, "not_found")
    c.refuse("sellPass", "sam", t5, {"memberId": "sue", "productId": "p60"}, "not_found")
    # one day valid, sold in the last minute of March: the date of the request time is the 31st
    t6 = "2026-03-31T23:59Z"
    p5 = passrec("pas-5", "ada", "day", 10, "2026-03-31", "2026-04-01")
    c.ok("sellPass", "sam", t6, {"memberId": "ada", "productId": "day"},
         {"pass": p5, "charge": chg("pass", "pas-5", 1, 1, 0, 1, t6)}, new="pas-5")
    c.ok("listMemberPasses", "ada", t6, {"memberId": "ada"}, {"memberId": "ada", "passes": [p1, p2, p5]})
    c.ok("listMemberPasses", "sam", t6, {"memberId": "bob"}, {"memberId": "bob", "passes": [p3, p4]})
    c.ok("listMemberPasses", "sam", t6, {"memberId": "eve"}, {"memberId": "eve", "passes": []})
    c.refuse("listMemberPasses", "ada", t6, {"memberId": "bob"}, "forbidden")
    c.audit_log(t6)
    return c.build()


def b_passes_2() -> dict:
    c = Case("B-passes-2", "B-passes", "passes: the draw at check-out: expiry date, two passes in order, partial cover, standing")
    world(c)
    for pid, minutes, price, days in (("pa", 25, 500, 30), ("pb", 100, 1000, 30), ("pc", 20, 400, 61)):
        c.ok("addPassProduct", "amy", "2026-04-01T08:00Z",
             {"productId": pid, "name": pid, "minutes": minutes, "priceCents": price, "validDays": days})
    s = "2026-04-01T09:00Z"
    # pas-1 is the pass that expires last; pas-2 and pas-3 expire on 2026-05-01
    c.ok("sellPass", "sam", s, {"memberId": "ada", "productId": "pc"}, None, new="pas-1")
    c.ok("sellPass", "sam", s, {"memberId": "ada", "productId": "pa"}, None, new="pas-2")
    c.ok("sellPass", "sam", s, {"memberId": "ada", "productId": "pb"}, None, new="pas-3")
    for rid, day in (("res-1", "2026-04-30"), ("res-2", "2026-05-01"), ("res-3", "2026-05-04")):
        c.ok("createReservation", "ada", "2026-04-01T09:30Z",
             {"memberId": "ada", "machineId": "L1", "start": f"{day}T10:00Z", "end": f"{day}T11:00Z"}, new=rid)
    # res-1: 50 metered minutes on 2026-04-30; the member is suspended during the session, which does not matter
    c.ok("checkIn", "ada", "2026-04-30T10:00Z", {"reservationId": "res-1"})
    c.ok("setMemberStanding", "sam", "2026-04-30T10:30Z", {"memberId": "ada", "standing": "suspended"})
    out = "2026-04-30T10:50Z"
    c.ok("checkOut", "ada", out, {"reservationId": "res-1"},
         {"reservation": resv("res-1", "ada", "L1", "2026-04-30T10:00Z", "2026-04-30T11:00Z", "completed",
                              "2026-04-30T10:00Z", out, 45, 50, 0),
          "charge": chg("usage", "res-1", 0, 45, 10, 0, out)})
    c.ok("setMemberStanding", "sam", "2026-04-30T11:00Z", {"memberId": "ada", "standing": "good"})
    c.ok("listMemberPasses", "ada", "2026-04-30T11:00Z", {"memberId": "ada"},
         {"memberId": "ada", "passes": [passrec("pas-1", "ada", "pc", 20, "2026-04-01", "2026-06-01"),
                                        passrec("pas-2", "ada", "pa", 0, "2026-04-01", "2026-05-01"),
                                        passrec("pas-3", "ada", "pb", 75, "2026-04-01", "2026-05-01")]})
    # res-2 on 2026-05-01: pas-2 and pas-3 are not valid on their expiry date, pas-1 gives its 20 minutes; billable 30
    c.ok("checkIn", "ada", "2026-05-01T10:00Z", {"reservationId": "res-2"})
    out = "2026-05-01T10:50Z"
    c.ok("checkOut", "ada", out, {"reservationId": "res-2"},
         {"reservation": resv("res-2", "ada", "L1", "2026-05-01T10:00Z", "2026-05-01T11:00Z", "completed",
                              "2026-05-01T10:00Z", out, 45, 50, 1215),
          "charge": chg("usage", "res-2", 30, 45, 10, 1215, out)})
    # res-3: nothing is valid any more: 10 metered minutes are all billable, 405 cents
    c.ok("checkIn", "ada", "2026-05-04T10:00Z", {"reservationId": "res-3"})
    out = "2026-05-04T10:10Z"
    c.ok("checkOut", "ada", out, {"reservationId": "res-3"},
         {"reservation": resv("res-3", "ada", "L1", "2026-05-04T10:00Z", "2026-05-04T11:00Z", "completed",
                              "2026-05-04T10:00Z", out, 45, 10, 405),
          "charge": chg("usage", "res-3", 10, 45, 10, 405, out)})
    c.ok("listMemberPasses", "ada", out, {"memberId": "ada"},
         {"memberId": "ada", "passes": [passrec("pas-1", "ada", "pc", 0, "2026-04-01", "2026-06-01"),
                                        passrec("pas-2", "ada", "pa", 0, "2026-04-01", "2026-05-01"),
                                        passrec("pas-3", "ada", "pb", 75, "2026-04-01", "2026-05-01")]})
    c.audit_log(out)
    return c.build()


# ---------------------------------------------------------------- promotions


def b_promotions_1() -> dict:
    c = Case("B-promotions-1", "B-promotions", "promotions: add, end, redeem; the five refusals in their order; date boundary")
    world(c)
    t = "2026-03-02T09:10Z"
    spring = promo("SPRING", 20, "course", 2, "2026-03-31", 0, False)
    c.ok("addPromotion", "amy", t, {"code": "SPRING", "percent": 20, "scope": "course", "maxRedemptions": 2, "validUntil": "2026-03-31"},
         {"promotion": spring})
    c.refuse("addPromotion", "amy", t, {"code": "SPRING", "percent": 5, "scope": "both", "maxRedemptions": 1, "validUntil": "2026-12-31"},
             "already_exists")
    c.refuse("addPromotion", "sam", t, {"code": "X1", "percent": 5, "scope": "both", "maxRedemptions": 1, "validUntil": "2026-12-31"},
             "forbidden")
    base = {"code": "X1", "percent": 5, "scope": "both", "maxRedemptions": 1, "validUntil": "2026-12-31"}
    c.refuse("addPromotion", "amy", t, {**base, "percent": 0}, "invalid_argument")
    c.refuse("addPromotion", "amy", t, {**base, "percent": 101}, "invalid_argument")
    c.refuse("addPromotion", "amy", t, {**base, "maxRedemptions": 0}, "invalid_argument")
    c.refuse("addPromotion", "amy", t, {**base, "scope": "all"}, "invalid_argument")
    c.refuse("addPromotion", "amy", t, {**base, "validUntil": "2026-02-30"}, "invalid_argument")
    c.ok("addPromotion", "amy", t, {"code": "SHORT", "percent": 15, "scope": "material", "maxRedemptions": 1, "validUntil": "2026-03-05"},
         {"promotion": promo("SHORT", 15, "material", 1, "2026-03-05", 0, False)})
    c.ok("addPromotion", "amy", t, {"code": "GONE", "percent": 10, "scope": "both", "maxRedemptions": 5, "validUntil": "2026-03-10"},
         {"promotion": promo("GONE", 10, "both", 5, "2026-03-10", 0, False)})
    # redemptions
    t1 = "2026-03-02T10:00Z"
    c.ok("redeemPromotion", "ada", t1, {"memberId": "ada", "code": "SPRING"},
         {"redemption": redemption("SPRING", "ada", 20, t1), "promotion": promo("SPRING", 20, "course", 2, "2026-03-31", 1, False)})
    c.ok("listMemberPromotions", "ada", t1, {"memberId": "ada"}, {"memberId": "ada", "redemptions": [redemption("SPRING", "ada", 20, t1)]})
    c.refuse("redeemPromotion", "ada", "2026-03-02T10:05Z", {"memberId": "ada", "code": "SPRING"}, "already_redeemed")
    c.refuse("redeemPromotion", "ada", "2026-03-02T10:05Z", {"memberId": "bob", "code": "SPRING"}, "forbidden")
    c.refuse("redeemPromotion", "ada", "2026-03-02T10:05Z", {"memberId": "ada", "code": "NOPE"}, "not_found")
    t2 = "2026-03-02T10:10Z"
    c.ok("redeemPromotion", "sam", t2, {"memberId": "bob", "code": "SPRING"},
         {"redemption": redemption("SPRING", "bob", 20, t2), "promotion": promo("SPRING", 20, "course", 2, "2026-03-31", 2, False)})
    # exhausted comes before already redeemed
    t3 = "2026-03-02T10:15Z"
    c.refuse("redeemPromotion", "eve", t3, {"memberId": "eve", "code": "SPRING"}, "promotion_exhausted")
    c.refuse("redeemPromotion", "ada", t3, {"memberId": "ada", "code": "SPRING"}, "promotion_exhausted")
    # ended
    t4 = "2026-03-02T10:20Z"
    ended = promo("GONE", 10, "both", 5, "2026-03-10", 0, True)
    c.ok("endPromotion", "amy", t4, {"code": "GONE"}, {"promotion": ended})
    c.refuse("endPromotion", "amy", t4, {"code": "GONE"}, "invalid_state")
    c.refuse("endPromotion", "sam", t4, {"code": "GONE"}, "forbidden")
    c.refuse("endPromotion", "amy", t4, {"code": "NOPE"}, "not_found")
    # suspended comes before ended; ended comes before everything after it
    c.ok("setMemberStanding", "sam", t4, {"memberId": "eve", "standing": "suspended"})
    c.refuse("redeemPromotion", "sam", t4, {"memberId": "eve", "code": "GONE"}, "member_suspended")
    c.ok("setMemberStanding", "sam", t4, {"memberId": "eve", "standing": "good"})
    c.refuse("redeemPromotion", "eve", t4, {"memberId": "eve", "code": "GONE"}, "promotion_ended")
    # the date validUntil is allowed, the day after is not
    t5 = "2026-03-05T23:59Z"
    c.ok("redeemPromotion", "ada", t5, {"memberId": "ada", "code": "SHORT"},
         {"redemption": redemption("SHORT", "ada", 15, t5), "promotion": promo("SHORT", 15, "material", 1, "2026-03-05", 1, False)})
    t6 = "2026-03-06T00:00Z"
    c.refuse("redeemPromotion", "bob", t6, {"memberId": "bob", "code": "SHORT"}, "promotion_expired")
    # ended and expired: ended is reported
    c.refuse("redeemPromotion", "bob", "2026-03-11T09:00Z", {"memberId": "bob", "code": "GONE"}, "promotion_ended")
    t7 = "2026-03-11T09:00Z"
    c.ok("listMemberPromotions", "sam", t7, {"memberId": "ada"},
         {"memberId": "ada", "redemptions": [redemption("SPRING", "ada", 20, t1), redemption("SHORT", "ada", 15, t5)]})
    c.ok("listMemberPromotions", "eve", t7, {"memberId": "eve"}, {"memberId": "eve", "redemptions": []})
    c.refuse("listMemberPromotions", "bob", t7, {"memberId": "ada"}, "forbidden")
    c.audit_log(t7)
    return c.build()


def b_promotions_2() -> dict:
    c = Case("B-promotions-2", "B-promotions", "promotions: oldest unused redemption by scope, used by a purchase or an enrollment; refusals use none")
    world(c)
    c.ok("addCourse", "sam", T0, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe basics", "feeCents": 12000, "capacity": 5})
    t = "2026-03-02T09:20Z"
    for code, pct, scope, until in (("A", 20, "course", "2026-12-31"), ("B", 15, "both", "2026-12-31"),
                                    ("C", 30, "material", "2026-12-31"), ("D", 40, "material", "2026-12-31"),
                                    ("E", 50, "material", "2026-03-02")):
        c.ok("addPromotion", "amy", t, {"code": code, "percent": pct, "scope": scope, "maxRedemptions": 10, "validUntil": until})
    for minute, code in (("10:00", "A"), ("10:01", "B"), ("10:02", "C")):
        c.ok("redeemPromotion", "ada", f"2026-03-02T{minute}Z", {"memberId": "ada", "code": code})
    ply = lambda stock: {"material": {"materialId": "ply", "name": "Plywood", "unitPriceCents": 150, "stock": stock}}  # noqa: E731
    # A does not cover materials: B (15 %) is the oldest unused one that does
    t1 = "2026-03-02T10:10Z"
    c.ok("purchaseMaterial", "ada", t1, {"memberId": "ada", "materialId": "ply", "quantity": 2},
         {"purchase": {"purchaseId": "pur-1", "memberId": "ada", "materialId": "ply", "quantity": 2, "unitPriceCents": 150,
                       "discountPercent": 15, "amountCents": 255, "at": t1}, **ply(98)}, new="pur-1")
    c.ok("listMemberPromotions", "ada", t1, {"memberId": "ada"},
         {"memberId": "ada", "redemptions": [redemption("A", "ada", 20, "2026-03-02T10:00Z"),
                                             redemption("B", "ada", 15, "2026-03-02T10:01Z", t1, "pur-1"),
                                             redemption("C", "ada", 30, "2026-03-02T10:02Z")]})
    c.ok("getAccount", "ada", t1, {"memberId": "ada"}, acct("ada", 0, 0, 0, 5255))
    # now A is the oldest unused one that covers courses: 12000 at 20 % is 9600
    t2 = "2026-03-02T10:20Z"
    c.ok("enrollInCourse", "ada", t2, {"courseId": "lathe-101", "memberId": "ada"},
         {"enrollment": {"courseId": "lathe-101", "memberId": "ada", "status": "enrolled", "enrolledAt": t2, "completedAt": None},
          "course": {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe basics", "feeCents": 12000, "capacity": 5,
                     "enrolledCount": 1}})
    c.ok("getAccount", "ada", t2, {"memberId": "ada"}, acct("ada", 0, 0, 0, 14855))
    t3 = "2026-03-02T10:30Z"
    c.ok("purchaseMaterial", "ada", t3, {"memberId": "ada", "materialId": "ply", "quantity": 4},
         {"purchase": {"purchaseId": "pur-2", "memberId": "ada", "materialId": "ply", "quantity": 4, "unitPriceCents": 150,
                       "discountPercent": 30, "amountCents": 420, "at": t3}, **ply(94)}, new="pur-2")
    t4 = "2026-03-02T10:40Z"
    c.ok("purchaseMaterial", "ada", t4, {"memberId": "ada", "materialId": "ply", "quantity": 1},
         {"purchase": {"purchaseId": "pur-3", "memberId": "ada", "materialId": "ply", "quantity": 1, "unitPriceCents": 150,
                       "discountPercent": 0, "amountCents": 150, "at": t4}, **ply(93)}, new="pur-3")
    c.ok("redeemPromotion", "ada", "2026-03-02T10:50Z", {"memberId": "ada", "code": "D"})
    # a refused purchase uses nothing
    c.refuse("purchaseMaterial", "ada", "2026-03-02T11:00Z", {"memberId": "ada", "materialId": "ply", "quantity": 1000}, "insufficient_stock")
    c.ok("listMemberPromotions", "ada", "2026-03-02T11:00Z", {"memberId": "ada"},
         {"memberId": "ada", "redemptions": [redemption("A", "ada", 20, "2026-03-02T10:00Z", t2, "lathe-101"),
                                             redemption("B", "ada", 15, "2026-03-02T10:01Z", t1, "pur-1"),
                                             redemption("C", "ada", 30, "2026-03-02T10:02Z", t3, "pur-2"),
                                             redemption("D", "ada", 40, "2026-03-02T10:50Z")]})
    t5 = "2026-03-02T11:10Z"
    c.ok("purchaseMaterial", "ada", t5, {"memberId": "ada", "materialId": "ply", "quantity": 1},
         {"purchase": {"purchaseId": "pur-4", "memberId": "ada", "materialId": "ply", "quantity": 1, "unitPriceCents": 150,
                       "discountPercent": 40, "amountCents": 90, "at": t5}, **ply(92)}, new="pur-4")
    c.ok("getAccount", "ada", t5, {"memberId": "ada"}, acct("ada", 0, 0, 0, 15515))
    # a redemption stays usable after its promotion ended and after its validUntil date
    c.ok("redeemPromotion", "bob", "2026-03-02T11:20Z", {"memberId": "bob", "code": "E"})
    c.ok("endPromotion", "amy", "2026-03-02T11:25Z", {"code": "E"})
    t6 = "2026-03-10T09:00Z"
    c.ok("purchaseMaterial", "bob", t6, {"memberId": "bob", "materialId": "ply", "quantity": 3},
         {"purchase": {"purchaseId": "pur-5", "memberId": "bob", "materialId": "ply", "quantity": 3, "unitPriceCents": 150,
                       "discountPercent": 50, "amountCents": 225, "at": t6}, **ply(89)}, new="pur-5")
    c.ok("listMemberPromotions", "bob", t6, {"memberId": "bob"},
         {"memberId": "bob", "redemptions": [redemption("E", "bob", 50, "2026-03-02T11:20Z", t6, "pur-5")]})
    c.audit_log(t6)
    return c.build()


# ---------------------------------------------------------------- scholarships


def b_scholarships_1() -> dict:
    c = Case("B-scholarships-1", "B-scholarships", "scholarships: funds, awards, revoke, award again, list")
    world(c)
    t = "2026-03-02T09:10Z"
    c.ok("addFund", "amy", t, {"fundId": "f1", "budgetCents": 10000}, {"fund": {"fundId": "f1", "budgetCents": 10000, "spentCents": 0}})
    c.refuse("addFund", "amy", t, {"fundId": "f1", "budgetCents": 5}, "already_exists")
    c.refuse("addFund", "sam", t, {"fundId": "f2", "budgetCents": 5}, "forbidden")
    c.refuse("addFund", "amy", t, {"fundId": "f2", "budgetCents": 0}, "invalid_argument")
    c.refuse("addFund", "amy", t, {"fundId": "f2", "budgetCents": 100000001}, "invalid_argument")
    c.ok("addFund", "amy", t, {"fundId": "f2", "budgetCents": 100000000},
         {"fund": {"fundId": "f2", "budgetCents": 100000000, "spentCents": 0}})
    t1 = "2026-03-02T10:00Z"
    c.ok("awardScholarship", "amy", t1, {"fundId": "f1", "memberId": "bob", "percent": 30},
         {"award": award("aws-1", "f1", "bob", 30, "active")}, new="aws-1")
    c.refuse("awardScholarship", "amy", t1, {"fundId": "f2", "memberId": "bob", "percent": 10}, "already_awarded")
    c.refuse("awardScholarship", "amy", t1, {"fundId": "nope", "memberId": "bob", "percent": 10}, "not_found")
    c.refuse("awardScholarship", "amy", t1, {"fundId": "f1", "memberId": "zed", "percent": 10}, "not_found")
    c.refuse("awardScholarship", "amy", t1, {"fundId": "f1", "memberId": "eve", "percent": 0}, "invalid_argument")
    c.refuse("awardScholarship", "amy", t1, {"fundId": "f1", "memberId": "eve", "percent": 101}, "invalid_argument")
    c.refuse("awardScholarship", "sam", t1, {"fundId": "f1", "memberId": "eve", "percent": 10}, "forbidden")
    c.ok("awardScholarship", "amy", t1, {"fundId": "f1", "memberId": "eve", "percent": 100},
         {"award": award("aws-2", "f1", "eve", 100, "active")}, new="aws-2")
    t2 = "2026-03-02T10:10Z"
    c.ok("revokeScholarship", "amy", t2, {"awardId": "aws-1"}, {"award": award("aws-1", "f1", "bob", 30, "revoked")})
    c.refuse("revokeScholarship", "amy", t2, {"awardId": "aws-1"}, "invalid_state")
    c.refuse("revokeScholarship", "amy", t2, {"awardId": "aws-9"}, "not_found")
    c.refuse("revokeScholarship", "sam", t2, {"awardId": "aws-2"}, "forbidden")
    # a revoked award frees the member; the refusals above consumed no number
    c.ok("awardScholarship", "amy", t2, {"fundId": "f1", "memberId": "bob", "percent": 40},
         {"award": award("aws-3", "f1", "bob", 40, "active")}, new="aws-3")
    # standing is not checked
    c.ok("setMemberStanding", "sam", t2, {"memberId": "ada", "standing": "suspended"})
    c.ok("awardScholarship", "amy", t2, {"fundId": "f2", "memberId": "ada", "percent": 10},
         {"award": award("aws-4", "f2", "ada", 10, "active")}, new="aws-4")
    c.ok("listAwards", "bob", t2, {"memberId": "bob"},
         {"memberId": "bob", "awards": [award("aws-1", "f1", "bob", 30, "revoked"), award("aws-3", "f1", "bob", 40, "active")]})
    c.ok("listAwards", "sam", t2, {"memberId": "eve"}, {"memberId": "eve", "awards": [award("aws-2", "f1", "eve", 100, "active")]})
    c.ok("listAwards", "sam", t2, {"memberId": "ada"}, {"memberId": "ada", "awards": [award("aws-4", "f2", "ada", 10, "active")]})
    c.refuse("listAwards", "bob", t2, {"memberId": "eve"}, "forbidden")
    c.audit_log(t2)
    return c.build()


def b_scholarships_2() -> dict:
    c = Case("B-scholarships-2", "B-scholarships",
             "scholarships: course discount: award covered at the budget, award vs promotion, only the winner is used, fee 0")
    world(c)
    c.ok("registerMember", "sam", T0, {"memberId": "cal", "name": "Cal Cole", "tier": "basic"})
    for cid, fee in (("k1", 12000), ("k2", 12000), ("k3", 12000), ("k4", 12000), ("free", 0)):
        c.ok("addCourse", "sam", T0, {"courseId": cid, "familyId": "lathe", "title": cid, "feeCents": fee, "capacity": 5})
    s = "2026-03-02T09:05Z"
    for fid, budget in (("fa", 7200), ("fb", 3600), ("fc", 3600), ("fd", 100000)):
        c.ok("addFund", "amy", s, {"fundId": fid, "budgetCents": budget})
    for aid, fid, mid, pct in (("aws-1", "fa", "ada", 30), ("aws-2", "fb", "bob", 30), ("aws-3", "fc", "eve", 30), ("aws-4", "fd", "cal", 50)):
        c.ok("awardScholarship", "amy", s, {"fundId": fid, "memberId": mid, "percent": pct}, None, new=aid)
    for code, pct in (("pq", 25), ("pr", 30), ("ps", 40)):
        c.ok("addPromotion", "amy", s, {"code": code, "percent": pct, "scope": "course", "maxRedemptions": 10, "validUntil": "2026-12-31"})
    r = "2026-03-02T09:10Z"
    for mid, code in (("ada", "pq"), ("bob", "pr"), ("eve", "ps"), ("cal", "pq")):
        c.ok("redeemPromotion", "sam", r, {"memberId": mid, "code": code})

    def enroll(minute: str, course: str, member: str, result: dict | None = None) -> None:
        c.ok("enrollInCourse", "sam", f"2026-03-02T{minute}Z", {"courseId": course, "memberId": member}, result)

    # ada: the award 30 % covers 3600 of 7200, then exactly the remaining 3600; then the promotion; then nothing
    enroll("10:00", "k1", "ada", {"enrollment": {"courseId": "k1", "memberId": "ada", "status": "enrolled",
                                                 "enrolledAt": "2026-03-02T10:00Z", "completedAt": None},
                                  "course": {"courseId": "k1", "familyId": "lathe", "title": "k1", "feeCents": 12000,
                                             "capacity": 5, "enrolledCount": 1}})
    enroll("10:01", "k2", "ada")
    enroll("10:02", "k3", "ada")
    enroll("10:03", "k4", "ada")
    # bob: equal percent, the award wins and the promotion stays; then the award is spent and the promotion is used
    enroll("10:10", "k1", "bob")
    enroll("10:11", "k2", "bob")
    # eve: the higher promotion wins and the fund is untouched; the second course still gets the award
    enroll("10:20", "k1", "eve")
    enroll("10:21", "k2", "eve")
    c.ok("revokeScholarship", "amy", "2026-03-02T10:40Z", {"awardId": "aws-3"})
    enroll("10:41", "k3", "eve")
    # cal: the higher award wins and the promotion stays unused; a free course uses neither
    c.ok("enrollInCourse", "sam", "2026-03-02T10:42Z", {"courseId": "k1", "memberId": "cal"})
    c.ok("enrollInCourse", "sam", "2026-03-02T10:43Z", {"courseId": "k2", "memberId": "cal"})
    c.ok("enrollInCourse", "sam", "2026-03-02T10:44Z", {"courseId": "free", "memberId": "cal"})
    t = "2026-03-02T11:00Z"
    c.ok("listMemberPromotions", "sam", t, {"memberId": "ada"},
         {"memberId": "ada", "redemptions": [redemption("pq", "ada", 25, r, "2026-03-02T10:02Z", "k3")]})
    c.ok("listMemberPromotions", "sam", t, {"memberId": "bob"},
         {"memberId": "bob", "redemptions": [redemption("pr", "bob", 30, r, "2026-03-02T10:11Z", "k2")]})
    c.ok("listMemberPromotions", "sam", t, {"memberId": "eve"},
         {"memberId": "eve", "redemptions": [redemption("ps", "eve", 40, r, "2026-03-02T10:20Z", "k1")]})
    c.ok("listMemberPromotions", "sam", t, {"memberId": "cal"}, {"memberId": "cal", "redemptions": [redemption("pq", "cal", 25, r)]})
    i1, d1 = "2026-04-01T00:00Z", "2026-04-15T00:00Z"

    def course(cid: str, discount: int, amount: int, minute: str) -> dict:
        return chg("course", cid, 1, 12000 if cid != "free" else 0, discount, amount, f"2026-03-02T{minute}Z")

    ada = [mline("plus", 4900, 0, 4900, i1), ada_course(), course("k1", 30, 8400, "10:00"), course("k2", 30, 8400, "10:01"),
           course("k3", 25, 9000, "10:02"), course("k4", 0, 12000, "10:03")]
    bob = [mline("basic", 2500, 0, 2500, i1), course("k1", 30, 8400, "10:10"), course("k2", 30, 8400, "10:11")]
    cal = [mline("basic", 2500, 0, 2500, i1), course("k1", 50, 6000, "10:42"), course("k2", 50, 6000, "10:43"),
           course("free", 0, 0, "10:44")]
    eve = [mline("pro", 9900, 0, 9900, i1), course("k1", 40, 7200, "10:20"), course("k2", 30, 8400, "10:21"),
           course("k3", 0, 12000, "10:41")]
    invoices = [simple("inv-1", "ada", "2026-03", i1, d1, ada), simple("inv-2", "bob", "2026-03", i1, d1, bob),
                simple("inv-3", "cal", "2026-03", i1, d1, cal), simple("inv-4", "eve", "2026-03", i1, d1, eve)]
    assert [i["totalCents"] for i in invoices] == [47700, 19300, 14500, 37500]
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": invoices})
    c.audit_log(i1)
    return c.build()


# ---------------------------------------------------------------- refunds


def _paid_march(c: Case) -> None:
    """March invoiced at 2026-04-01T00:00Z (ada 4900 + 5000, bob 2500, eve 9900); ada and bob pay in full on 04-02."""
    world(c)
    generate(c, "2026-03", "2026-04-01T00:00Z")
    pays(c, "ada", "2026-04-02T09:00Z", "inv-1", 9900, "pay-1")
    pays(c, "sam", "2026-04-02T09:05Z", "inv-2", 2500, "pay-2")


def b_refunds_1() -> dict:
    c = Case("B-refunds-1", "B-refunds", "refunds: the three refusals in order, partial refunds bounded by earlier ones, credit, nothing else changes")
    _paid_march(c)
    t = "2026-04-03T09:00Z"
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-3", "lineNumber": 5, "amountCents": 99999, "reason": "x"}, "invoice_not_paid")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 3, "amountCents": 999999, "reason": "x"}, "invalid_line")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 5001, "reason": "x"}, "refund_too_large")
    c.ok("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 2000, "reason": "late arrival"},
         {"refund": refund("rfd-1", "inv-1", 2, 2000, "late arrival", t), "creditBalanceCents": 2000}, new="rfd-1")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 3001, "reason": "x"}, "refund_too_large")
    c.ok("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 3000, "reason": "rest"},
         {"refund": refund("rfd-2", "inv-1", 2, 3000, "rest", t), "creditBalanceCents": 5000}, new="rfd-2")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 1, "reason": "x"}, "refund_too_large")
    # another line of the same invoice has its own bound
    c.ok("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 4900, "reason": "fee"},
         {"refund": refund("rfd-3", "inv-1", 1, 4900, "fee", t), "creditBalanceCents": 9900}, new="rfd-3")
    ok = {"invoiceId": "inv-2", "lineNumber": 1, "amountCents": 1, "reason": "x"}
    c.refuse("issueRefund", "sam", t, ok, "forbidden")
    c.refuse("issueRefund", "ada", t, ok, "forbidden")
    c.refuse("issueRefund", "amy", t, {**ok, "lineNumber": 0}, "invalid_argument")
    c.refuse("issueRefund", "amy", t, {**ok, "lineNumber": 1001}, "invalid_argument")
    c.refuse("issueRefund", "amy", t, {**ok, "amountCents": 0}, "invalid_argument")
    c.refuse("issueRefund", "amy", t, {**ok, "reason": "  "}, "invalid_argument")
    c.refuse("issueRefund", "amy", t, {**ok, "invoiceId": "inv-99"}, "not_found")
    c.ok("listRefunds", "sam", t, {"invoiceId": "inv-1"},
         {"invoiceId": "inv-1", "refunds": [refund("rfd-1", "inv-1", 2, 2000, "late arrival", t),
                                             refund("rfd-2", "inv-1", 2, 3000, "rest", t),
                                             refund("rfd-3", "inv-1", 1, 4900, "fee", t)]})
    c.ok("listRefunds", "ada", t, {"invoiceId": "inv-1"},
         {"invoiceId": "inv-1", "refunds": [refund("rfd-1", "inv-1", 2, 2000, "late arrival", t),
                                             refund("rfd-2", "inv-1", 2, 3000, "rest", t),
                                             refund("rfd-3", "inv-1", 1, 4900, "fee", t)]})
    c.ok("listRefunds", "sam", t, {"invoiceId": "inv-2"}, {"invoiceId": "inv-2", "refunds": []})
    c.refuse("listRefunds", "bob", t, {"invoiceId": "inv-1"}, "forbidden")
    # the standing is not checked
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "suspended"})
    c.ok("issueRefund", "amy", t, ok, {"refund": refund("rfd-4", "inv-2", 1, 1, "x", t), "creditBalanceCents": 1}, new="rfd-4")
    # the invoice, its lines and its status do not change
    issued, due = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    c.ok("listInvoices", "ada", t, {"memberId": "ada"},
         {"memberId": "ada", "invoices": [inv("inv-1", "ada", "2026-03", issued, due, [mline("plus", 4900, 0, 4900, issued), ada_course()],
                                              9900, 0, 9900, 0, "paid")]})
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, acct("ada", 9900, 0, 0, 0))
    c.ok("getTrialBalance", "amy", t, {},
         tb({"cash": (12400, 0), "creditLiability": (0, 9901), "receivable": (22300, 12400), "revenueCourse": (0, 5000),
             "revenueMembership": (0, 17300), "revenueRefunds": (9901, 0)}, 44601, 44601))
    c.audit_log(t)
    return c.build()


def b_refunds_2() -> dict:
    c = Case("B-refunds-2", "B-refunds", "refunds: a zero line has nothing left; lines are bounded one by one; invalid_line at the edge")
    world(c)
    c.ok("addPassProduct", "amy", T0, {"productId": "pp", "name": "Thirty", "minutes": 30, "priceCents": 3000, "validDays": 60})
    c.ok("sellPass", "sam", "2026-03-03T08:00Z", {"memberId": "ada", "productId": "pp"}, None, new="pas-1")
    # 20 metered minutes, all drawn from the pass: a usage line of 0 cents
    session(c, "ada", "res-1", "2026-03-04T10:00Z", "2026-03-04T11:00Z", "2026-03-03T08:30Z", "2026-03-04T10:20Z")
    i1, d1 = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    lines = [mline("plus", 4900, 0, 4900, i1), ada_course(), chg("pass", "pas-1", 1, 3000, 0, 3000, "2026-03-03T08:00Z"),
             chg("usage", "res-1", 0, 45, 10, 0, "2026-03-04T10:20Z")]
    march = [simple("inv-1", "ada", "2026-03", i1, d1, lines),
             simple("inv-2", "bob", "2026-03", i1, d1, [mline("basic", 2500, 0, 2500, i1)]),
             simple("inv-3", "eve", "2026-03", i1, d1, [mline("pro", 9900, 0, 9900, i1)])]
    assert march[0]["totalCents"] == 12900
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": march})
    t = "2026-04-02T09:00Z"
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 4, "amountCents": 1, "reason": "x"}, "invoice_not_paid")
    pays(c, "ada", t, "inv-1", 12900, "pay-1")
    t = "2026-04-03T09:00Z"
    # the usage line is 0 cents: whatever is asked is too much; line 5 does not exist
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 4, "amountCents": 1, "reason": "x"}, "refund_too_large")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 5, "amountCents": 1, "reason": "x"}, "invalid_line")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 1000, "amountCents": 1, "reason": "x"}, "invalid_line")
    c.ok("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 3, "amountCents": 3000, "reason": "pass returned"},
         {"refund": refund("rfd-1", "inv-1", 3, 3000, "pass returned", t), "creditBalanceCents": 3000}, new="rfd-1")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 3, "amountCents": 1, "reason": "x"}, "refund_too_large")
    # the membership line in two parts: 100, then exactly what is left, then nothing
    c.ok("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 100, "reason": "p1"},
         {"refund": refund("rfd-2", "inv-1", 1, 100, "p1", t), "creditBalanceCents": 3100}, new="rfd-2")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 4801, "reason": "x"}, "refund_too_large")
    c.ok("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 4800, "reason": "p2"},
         {"refund": refund("rfd-3", "inv-1", 1, 4800, "p2", t), "creditBalanceCents": 7900}, new="rfd-3")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 1, "amountCents": 1, "reason": "x"}, "refund_too_large")
    c.ok("listRefunds", "ada", t, {"invoiceId": "inv-1"},
         {"invoiceId": "inv-1", "refunds": [refund("rfd-1", "inv-1", 3, 3000, "pass returned", t),
                                             refund("rfd-2", "inv-1", 1, 100, "p1", t),
                                             refund("rfd-3", "inv-1", 1, 4800, "p2", t)]})
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, acct("ada", 7900, 0, 0, 0))
    c.audit_log(t)
    return c.build()


# ---------------------------------------------------------------- dunning


def b_dunning_1() -> dict:
    c = Case("B-dunning-1", "B-dunning", "dunning: ladder validation, no ladder, days overdue, notice then fee and suspension, nothing twice")
    world(c)
    generate(c, "2026-03", MARCH_AT)
    pays(c, "eve", "2026-04-02T09:00Z", "inv-3", 9900, "pay-1")
    t = "2026-04-02T09:05Z"
    c.refuse("runDunning", "sam", t, {}, "no_ladder")
    c.refuse("setDunningLadder", "sam", t, {"noticeDays": 3, "feeDays": 7, "feeCents": 500, "suspendDays": 14}, "forbidden")
    lad = {"noticeDays": 3, "feeDays": 7, "feeCents": 500, "suspendDays": 14}
    c.refuse("setDunningLadder", "amy", t, {**lad, "feeDays": 3}, "invalid_ladder")
    c.refuse("setDunningLadder", "amy", t, {**lad, "suspendDays": 7}, "invalid_ladder")
    c.refuse("setDunningLadder", "amy", t, {**lad, "noticeDays": 7}, "invalid_ladder")
    c.refuse("setDunningLadder", "amy", t, {**lad, "suspendDays": 6}, "invalid_ladder")
    c.refuse("setDunningLadder", "amy", t, {**lad, "noticeDays": 0}, "invalid_argument")
    c.refuse("setDunningLadder", "amy", t, {**lad, "feeCents": 0}, "invalid_argument")
    c.refuse("setDunningLadder", "amy", t, {**lad, "feeCents": 1000001}, "invalid_argument")
    c.refuse("setDunningLadder", "amy", t, {**lad, "suspendDays": 3651}, "invalid_argument")
    c.ok("setDunningLadder", "amy", t, dict(lad), {"ladder": dict(lad)})
    c.refuse("runDunning", "ada", t, {}, "forbidden")
    # inv-1 and inv-2 are due on 2026-04-15T08:00Z. 2 days and 23:59 hours: no step
    c.ok("runDunning", "sam", "2026-04-18T07:59Z", {}, {"actions": []})
    t3 = "2026-04-18T08:00Z"
    c.ok("runDunning", "sam", t3, {}, {"actions": [action("inv-1", "notice"), action("inv-2", "notice")]})
    c.ok("runDunning", "amy", t3, {}, {"actions": []})
    c.ok("listNotices", "ada", t3, {"memberId": "ada"}, {"memberId": "ada", "notices": [notice(1, "ada", "dunning", "inv-1", t3)]})
    c.ok("listNotices", "bob", t3, {"memberId": "bob"}, {"memberId": "bob", "notices": [notice(2, "bob", "dunning", "inv-2", t3)]})
    c.ok("listDunningCases", "ada", t3, {"memberId": "ada"}, {"memberId": "ada", "cases": [case_rec(1, "inv-1", "ada", t3, None, None)]})
    c.ok("listDunningCases", "sam", t3, {"memberId": "eve"}, {"memberId": "eve", "cases": []})
    c.refuse("listDunningCases", "bob", t3, {"memberId": "ada"}, "forbidden")
    # 15 days: the fee and the suspension in one run, in this order, per invoice
    t4 = "2026-04-30T08:00Z"
    c.ok("runDunning", "sam", t4, {}, {"actions": [action("inv-1", "fee"), action("inv-1", "suspend"),
                                                  action("inv-2", "fee"), action("inv-2", "suspend")]})
    c.ok("runDunning", "sam", t4, {}, {"actions": []})
    c.ok("getMember", "sam", t4, {"memberId": "ada"}, member_rec("ada", "Ada Byron", "plus", "suspended", T0))
    c.ok("getMember", "sam", t4, {"memberId": "bob"}, member_rec("bob", "Bob Stone", "basic", "suspended", T0))
    c.ok("getMember", "sam", t4, {"memberId": "eve"}, member_rec("eve", "Eve Marsh", "pro", "good", T0))
    c.ok("listNotices", "ada", t4, {"memberId": "ada"},
         {"memberId": "ada", "notices": [notice(1, "ada", "dunning", "inv-1", t3), notice(3, "ada", "suspension", "inv-1", t4)]})
    c.ok("listNotices", "bob", t4, {"memberId": "bob"},
         {"memberId": "bob", "notices": [notice(2, "bob", "dunning", "inv-2", t3), notice(4, "bob", "suspension", "inv-2", t4)]})
    c.ok("listDunningCases", "ada", t4, {"memberId": "ada"}, {"memberId": "ada", "cases": [case_rec(1, "inv-1", "ada", t3, t4, t4)]})
    c.ok("listDunningCases", "bob", t4, {"memberId": "bob"}, {"memberId": "bob", "cases": [case_rec(2, "inv-2", "bob", t3, t4, t4)]})
    # the late fee is a charge of its own; the overdue invoice does not change
    c.ok("getAccount", "ada", t4, {"memberId": "ada"}, acct("ada", 0, 9900, 9900, 500))
    c.audit_log(t4)
    return c.build()


def b_dunning_2() -> dict:
    c = Case("B-dunning-2", "B-dunning", "dunning: several runs, a new ladder applies to later runs, steps once per invoice, reinstated stays reinstated")
    world(c)
    generate(c, "2026-03", MARCH_AT)
    c.ok("setDunningLadder", "amy", "2026-04-02T09:00Z", {"noticeDays": 1, "feeDays": 2, "feeCents": 700, "suspendDays": 5})
    t1 = "2026-04-16T08:00Z"
    c.ok("runDunning", "sam", t1, {}, {"actions": [action("inv-1", "notice"), action("inv-2", "notice"), action("inv-3", "notice")]})
    # the new ladder replaces the old; cases stay
    lad = {"noticeDays": 1, "feeDays": 3, "feeCents": 900, "suspendDays": 10}
    c.ok("setDunningLadder", "amy", "2026-04-16T09:00Z", dict(lad), {"ladder": dict(lad)})
    c.ok("listDunningCases", "ada", "2026-04-16T09:00Z", {"memberId": "ada"},
         {"memberId": "ada", "cases": [case_rec(1, "inv-1", "ada", t1, None, None)]})
    pays(c, "eve", "2026-04-17T09:00Z", "inv-3", 9900, "pay-1")
    t2 = "2026-04-18T08:00Z"
    c.ok("runDunning", "sam", t2, {}, {"actions": [action("inv-1", "fee"), action("inv-2", "fee")]})
    # 9 days: the old ladder would suspend at 5; the new one does not yet
    c.ok("runDunning", "sam", "2026-04-24T08:00Z", {}, {"actions": []})
    t3 = "2026-04-25T08:00Z"
    c.ok("runDunning", "sam", t3, {}, {"actions": [action("inv-1", "suspend"), action("inv-2", "suspend")]})
    c.ok("setMemberStanding", "sam", "2026-04-25T09:00Z", {"memberId": "ada", "standing": "good"})
    pays(c, "ada", "2026-04-26T09:00Z", "inv-1", 9900, "pay-2")
    pays(c, "bob", "2026-04-26T09:05Z", "inv-2", 2500, "pay-3")
    # paying reinstates nobody; paid invoices are not visited; the reinstated member is not suspended again
    c.ok("getMember", "sam", "2026-04-26T09:10Z", {"memberId": "ada"}, member_rec("ada", "Ada Byron", "plus", "good", T0))
    c.ok("getMember", "sam", "2026-04-26T09:10Z", {"memberId": "bob"}, member_rec("bob", "Bob Stone", "basic", "suspended", T0))
    c.ok("runDunning", "sam", "2026-05-05T08:00Z", {}, {"actions": []})
    t4 = "2026-05-05T08:00Z"
    c.ok("getMember", "sam", t4, {"memberId": "ada"}, member_rec("ada", "Ada Byron", "plus", "good", T0))
    c.ok("listDunningCases", "ada", t4, {"memberId": "ada"}, {"memberId": "ada", "cases": [case_rec(1, "inv-1", "ada", t1, t2, t3)]})
    c.ok("listDunningCases", "bob", t4, {"memberId": "bob"}, {"memberId": "bob", "cases": [case_rec(2, "inv-2", "bob", t1, t2, t3)]})
    c.ok("listDunningCases", "eve", t4, {"memberId": "eve"}, {"memberId": "eve", "cases": [case_rec(3, "inv-3", "eve", t1, None, None)]})
    c.ok("listNotices", "ada", t4, {"memberId": "ada"},
         {"memberId": "ada", "notices": [notice(1, "ada", "dunning", "inv-1", t1), notice(4, "ada", "suspension", "inv-1", t3)]})
    c.ok("listNotices", "eve", t4, {"memberId": "eve"}, {"memberId": "eve", "notices": [notice(3, "eve", "dunning", "inv-3", t1)]})
    # the fee was charged at the ladder of the second run: 900 cents
    c.ok("getAccount", "ada", t4, {"memberId": "ada"}, acct("ada", 0, 0, 0, 900))
    c.ok("getAccount", "bob", t4, {"memberId": "bob"}, acct("bob", 0, 0, 0, 900))
    c.ok("getAccount", "eve", t4, {"memberId": "eve"}, acct("eve", 0, 0, 0, 0))
    c.audit_log(t4)
    return c.build()


# ---------------------------------------------------------------- lockers


def b_lockers_1() -> dict:
    c = Case("B-lockers-1", "B-lockers", "lockers: rent refusals in order, the limit of two, end, bill the current month once")
    world(c)
    t = "2026-03-02T09:10Z"
    c.ok("addLocker", "amy", t, {"lockerId": "K1", "monthlyCents": 1500}, {"locker": {"lockerId": "K1", "monthlyCents": 1500}})
    c.ok("addLocker", "amy", t, {"lockerId": "K2", "monthlyCents": 2000}, {"locker": {"lockerId": "K2", "monthlyCents": 2000}})
    c.ok("addLocker", "amy", t, {"lockerId": "K3", "monthlyCents": 3000}, {"locker": {"lockerId": "K3", "monthlyCents": 3000}})
    c.refuse("addLocker", "amy", t, {"lockerId": "K1", "monthlyCents": 9}, "already_exists")
    c.refuse("addLocker", "sam", t, {"lockerId": "K9", "monthlyCents": 9}, "forbidden")
    c.refuse("addLocker", "amy", t, {"lockerId": "K9", "monthlyCents": 0}, "invalid_argument")
    c.refuse("addLocker", "amy", t, {"lockerId": "K9", "monthlyCents": 1000001}, "invalid_argument")
    t1 = "2026-03-02T10:00Z"
    c.ok("rentLocker", "ada", t1, {"memberId": "ada", "lockerId": "K1"},
         {"rental": rental("rnt-1", "K1", "ada", "active", t1, None)}, new="rnt-1")
    c.refuse("rentLocker", "bob", "2026-03-02T10:05Z", {"memberId": "bob", "lockerId": "K1"}, "locker_taken")
    t2 = "2026-03-02T10:10Z"
    c.ok("rentLocker", "ada", t2, {"memberId": "ada", "lockerId": "K2"},
         {"rental": rental("rnt-2", "K2", "ada", "active", t2, None)}, new="rnt-2")
    c.refuse("rentLocker", "ada", "2026-03-02T10:15Z", {"memberId": "ada", "lockerId": "K3"}, "rental_limit")
    # taken comes before the limit
    c.refuse("rentLocker", "ada", "2026-03-02T10:16Z", {"memberId": "ada", "lockerId": "K1"}, "locker_taken")
    # suspended comes before taken
    c.ok("setMemberStanding", "sam", "2026-03-02T10:20Z", {"memberId": "bob", "standing": "suspended"})
    c.refuse("rentLocker", "bob", "2026-03-02T10:21Z", {"memberId": "bob", "lockerId": "K1"}, "member_suspended")
    c.ok("setMemberStanding", "sam", "2026-03-02T10:25Z", {"memberId": "bob", "standing": "good"})
    c.refuse("rentLocker", "bob", "2026-03-02T10:26Z", {"memberId": "bob", "lockerId": "K1"}, "locker_taken")
    c.refuse("rentLocker", "ada", "2026-03-02T10:27Z", {"memberId": "bob", "lockerId": "K3"}, "forbidden")
    c.refuse("rentLocker", "sam", "2026-03-02T10:27Z", {"memberId": "bob", "lockerId": "K9"}, "not_found")
    c.refuse("rentLocker", "sam", "2026-03-02T10:27Z", {"memberId": "sue", "lockerId": "K3"}, "not_found")
    # end: only the owner or staff; twice is a state error
    c.refuse("endLockerRental", "bob", "2026-03-02T10:30Z", {"rentalId": "rnt-1"}, "forbidden")
    c.refuse("endLockerRental", "ada", "2026-03-02T10:30Z", {"rentalId": "rnt-9"}, "not_found")
    t3 = "2026-03-02T10:31Z"
    c.ok("endLockerRental", "ada", t3, {"rentalId": "rnt-1"}, {"rental": rental("rnt-1", "K1", "ada", "ended", t1, t3)})
    c.refuse("endLockerRental", "ada", "2026-03-02T10:32Z", {"rentalId": "rnt-1"}, "invalid_state")
    # the ended rental frees the limit and the locker
    t4 = "2026-03-02T10:33Z"
    c.ok("rentLocker", "ada", t4, {"memberId": "ada", "lockerId": "K3"},
         {"rental": rental("rnt-3", "K3", "ada", "active", t4, None)}, new="rnt-3")
    t5 = "2026-03-02T10:34Z"
    c.ok("rentLocker", "bob", t5, {"memberId": "bob", "lockerId": "K1"},
         {"rental": rental("rnt-4", "K1", "bob", "active", t5, None)}, new="rnt-4")
    # a suspended member is billed all the same
    c.ok("setMemberStanding", "sam", "2026-03-02T10:40Z", {"memberId": "ada", "standing": "suspended"})
    b = "2026-03-15T12:00Z"
    c.refuse("billLockers", "sue", b, {"month": "2026-04"}, "not_current_month")
    c.refuse("billLockers", "ada", b, {"month": "2026-03"}, "forbidden")
    c.ok("billLockers", "sue", b, {"month": "2026-03"},
         {"month": "2026-03", "charges": [chg("locker", "rnt-4", 1, 1500, 0, 1500, b), chg("locker", "rnt-2", 1, 2000, 0, 2000, b),
                                          chg("locker", "rnt-3", 1, 3000, 0, 3000, b)]})
    c.refuse("billLockers", "sue", "2026-03-15T12:30Z", {"month": "2026-03"}, "already_billed")
    # in April, March is not the current month (checked before the month was billed already)
    a = "2026-04-02T09:00Z"
    c.refuse("billLockers", "sue", a, {"month": "2026-03"}, "not_current_month")
    c.ok("billLockers", "sue", a, {"month": "2026-04"},
         {"month": "2026-04", "charges": [chg("locker", "rnt-4", 1, 1500, 0, 1500, a), chg("locker", "rnt-2", 1, 2000, 0, 2000, a),
                                          chg("locker", "rnt-3", 1, 3000, 0, 3000, a)]})
    for rid in ("rnt-2", "rnt-3", "rnt-4"):
        c.ok("endLockerRental", "sam", "2026-04-03T09:00Z", {"rentalId": rid})
    m = "2026-05-01T00:00Z"
    c.ok("billLockers", "amy", m, {"month": "2026-05"}, {"month": "2026-05", "charges": []})
    c.refuse("billLockers", "amy", m, {"month": "2026-05"}, "already_billed")
    c.audit_log(m)
    return c.build()


def b_lockers_2() -> dict:
    c = Case("B-lockers-2", "B-lockers", "lockers: the whole rent is billed, month end boundary, invoice lines and postings, ended rentals not billed")
    world(c)
    t = "2026-03-02T09:10Z"
    c.ok("addLocker", "amy", t, {"lockerId": "La", "monthlyCents": 1500})
    c.ok("addLocker", "amy", t, {"lockerId": "Lb", "monthlyCents": 2000})
    c.ok("rentLocker", "sam", "2026-03-02T10:00Z", {"memberId": "ada", "lockerId": "La"}, None, new="rnt-1")
    c.ok("rentLocker", "ada", "2026-03-20T10:00Z", {"memberId": "ada", "lockerId": "Lb"}, None, new="rnt-2")
    last = "2026-03-31T23:59Z"
    c.ok("billLockers", "sue", last, {"month": "2026-03"},
         {"month": "2026-03", "charges": [chg("locker", "rnt-1", 1, 1500, 0, 1500, last), chg("locker", "rnt-2", 1, 2000, 0, 2000, last)]})
    c.refuse("billLockers", "sue", last, {"month": "2026-03"}, "already_billed")
    i1, d1 = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    c.refuse("billLockers", "sue", i1, {"month": "2026-03"}, "not_current_month")
    c.ok("endLockerRental", "ada", i1, {"rentalId": "rnt-2"})
    march = [simple("inv-1", "ada", "2026-03", i1, d1, [mline("plus", 4900, 0, 4900, i1), ada_course(),
                                                         chg("locker", "rnt-1", 1, 1500, 0, 1500, last),
                                                         chg("locker", "rnt-2", 1, 2000, 0, 2000, last)]),
             simple("inv-2", "bob", "2026-03", i1, d1, [mline("basic", 2500, 0, 2500, i1)]),
             simple("inv-3", "eve", "2026-03", i1, d1, [mline("pro", 9900, 0, 9900, i1)])]
    assert march[0]["totalCents"] == 13400
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": march})
    c.ok("getTrialBalance", "amy", i1, {},
         tb({"receivable": (25800, 0), "revenueCourse": (0, 5000), "revenueLockers": (0, 3500), "revenueMembership": (0, 17300)},
            25800, 25800))
    # April: only the rental still active is billed, at the request time
    c.ok("billLockers", "sue", i1, {"month": "2026-04"}, {"month": "2026-04", "charges": [chg("locker", "rnt-1", 1, 1500, 0, 1500, i1)]})
    c.ok("endLockerRental", "ada", "2026-04-02T09:00Z", {"rentalId": "rnt-1"})
    m = "2026-05-01T00:00Z"
    c.ok("billLockers", "sue", m, {"month": "2026-05"}, {"month": "2026-05", "charges": []})
    c.refuse("billLockers", "sue", m, {"month": "2026-05"}, "already_billed")
    c.audit_log(m)
    return c.build()


# ---------------------------------------------------------------- finance


def _rich_march(c: Case) -> None:
    """All kinds a March can have here: membership, course, deposit, pass, material, locker, with credit on eve's invoice."""
    world(c)
    c.ok("addPassProduct", "amy", "2026-03-02T09:10Z", {"productId": "pp", "name": "Twenty", "minutes": 20, "priceCents": 2000, "validDays": 30})
    c.ok("addLocker", "amy", "2026-03-02T09:20Z", {"lockerId": "La", "monthlyCents": 1500})
    c.ok("takeDeposit", "sam", "2026-03-02T10:00Z", {"memberId": "bob", "purpose": "key", "amountCents": 1000}, None, new="dep-1")
    c.ok("sellPass", "sam", "2026-03-02T10:10Z", {"memberId": "ada", "productId": "pp"}, None, new="pas-1")
    c.ok("purchaseMaterial", "sam", "2026-03-02T10:20Z", {"memberId": "eve", "materialId": "ply", "quantity": 2}, None, new="pur-1")
    c.ok("rentLocker", "sam", "2026-03-02T10:30Z", {"memberId": "ada", "lockerId": "La"}, None, new="rnt-1")
    c.ok("issueCredit", "amy", "2026-03-15T09:00Z", {"memberId": "eve", "amountCents": 700, "reason": "goodwill"})
    c.ok("billLockers", "sam", "2026-03-31T12:00Z", {"month": "2026-03"}, None)


def _rich_invoices() -> list:
    i1, d1 = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    ada = [mline("plus", 4900, 0, 4900, i1), ada_course(), chg("pass", "pas-1", 1, 2000, 0, 2000, "2026-03-02T10:10Z"),
           chg("locker", "rnt-1", 1, 1500, 0, 1500, "2026-03-31T12:00Z")]
    bob = [mline("basic", 2500, 0, 2500, i1), chg("deposit", "dep-1", 1, 1000, 0, 1000, "2026-03-02T10:00Z")]
    eve = [mline("pro", 9900, 0, 9900, i1), chg("material", "pur-1", 2, 150, 0, 300, "2026-03-02T10:20Z")]
    return [simple("inv-1", "ada", "2026-03", i1, d1, ada), simple("inv-2", "bob", "2026-03", i1, d1, bob),
            simple("inv-3", "eve", "2026-03", i1, d1, eve, credit=700)]


def b_finance_1() -> dict:
    c = Case("B-finance-1", "B-finance", "finance: charges by kind, finance summary, revenue report beside the new kinds")
    _rich_march(c)
    t = "2026-03-31T12:00Z"
    zero = kinds({}, None, 0)
    c.ok("chargesByKind", "sam", t, {"month": "2026-03"}, {"month": "2026-03", "rows": zero, "totalCents": 0})
    c.ok("financeSummary", "amy", t, {"month": "2026-03"},
         {"month": "2026-03", "invoicedCents": 0, "refundedCents": 0, "paymentsCents": 0, "creditsIssuedCents": 700,
          "journalEntries": 1, "trialBalanced": True})
    c.ok("revenueReport", "sam", t, {"month": "2026-03"},
         {"month": "2026-03", "invoicesGenerated": False, "invoiceCount": 0, "membershipCents": 0, "usageCents": 0,
          "materialCents": 0, "courseCents": 0, "invoicedCents": 0, "creditAppliedCents": 0, "outstandingCents": 0,
          "paymentsCents": 0, "creditsIssuedCents": 700})
    invoices = _rich_invoices()
    assert [i["totalCents"] for i in invoices] == [13400, 3500, 10200] and invoices[2]["balanceCents"] == 9500
    i1 = "2026-04-01T00:00Z"
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": invoices})
    by_kind = {"membership": 17300, "material": 300, "course": 5000, "deposit": 1000, "pass": 2000, "locker": 1500}
    c.ok("chargesByKind", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds(by_kind, None, 27100), "totalCents": 27100})
    pays(c, "ada", "2026-04-05T09:00Z", "inv-1", 13400, "pay-1")
    pays(c, "sam", "2026-04-05T09:05Z", "inv-2", 3500, "pay-2")
    r = "2026-04-05T10:00Z"
    c.ok("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 3, "amountCents": 500, "reason": "pass"}, None, new="rfd-1")
    c.ok("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 4, "amountCents": 1500, "reason": "locker"}, None, new="rfd-2")
    c.ok("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 1000, "reason": "course"}, None, new="rfd-3")
    # refunds are counted in the month of the invoice, whenever they were issued
    c.ok("chargesByKind", "sam", r, {"month": "2026-03"},
         {"month": "2026-03", "rows": kinds(by_kind, {"course": 1000, "pass": 500, "locker": 1500}, 27100), "totalCents": 27100})
    c.ok("chargesByKind", "amy", r, {"month": "2026-04"}, {"month": "2026-04", "rows": kinds({}, None, 0), "totalCents": 0})
    c.refuse("chargesByKind", "ada", r, {"month": "2026-03"}, "forbidden")
    c.refuse("financeSummary", "sam", r, {"month": "2026-03"}, "forbidden")
    c.ok("financeSummary", "amy", r, {"month": "2026-03"},
         {"month": "2026-03", "invoicedCents": 27100, "refundedCents": 3000, "paymentsCents": 0, "creditsIssuedCents": 700,
          "journalEntries": 1, "trialBalanced": True})
    c.ok("financeSummary", "amy", r, {"month": "2026-04"},
         {"month": "2026-04", "invoicedCents": 0, "refundedCents": 0, "paymentsCents": 16900, "creditsIssuedCents": 0,
          "journalEntries": 14, "trialBalanced": True})
    # the revenue report names four kinds only; credit applied, outstanding and payments take every line
    c.ok("revenueReport", "sam", r, {"month": "2026-03"},
         {"month": "2026-03", "invoicesGenerated": True, "invoiceCount": 3, "membershipCents": 17300, "usageCents": 0,
          "materialCents": 300, "courseCents": 5000, "invoicedCents": 22600, "creditAppliedCents": 700,
          "outstandingCents": 9500, "paymentsCents": 0, "creditsIssuedCents": 700})
    c.ok("revenueReport", "sam", r, {"month": "2026-04"},
         {"month": "2026-04", "invoicesGenerated": False, "invoiceCount": 0, "membershipCents": 0, "usageCents": 0,
          "materialCents": 0, "courseCents": 0, "invoicedCents": 0, "creditAppliedCents": 0, "outstandingCents": 0,
          "paymentsCents": 16900, "creditsIssuedCents": 0})
    c.ok("getTrialBalance", "amy", r, {},
         tb({"cash": (16900, 0), "creditLiability": (700, 3700), "depositLiability": (0, 1000), "expenseGoodwill": (700, 0),
             "receivable": (27100, 17600), "revenueCourse": (0, 5000), "revenueLockers": (0, 1500), "revenueMaterial": (0, 300),
             "revenueMembership": (0, 17300), "revenuePasses": (0, 2000), "revenueRefunds": (3000, 0)}, 48400, 48400))
    c.audit_log(r)
    return c.build()


def b_finance_2() -> dict:
    c = Case("B-finance-2", "B-finance", "finance: aged receivables with its buckets at their edges, household statement")
    world(c)
    sam = "sam"
    c.ok("agedReceivables", sam, T0, {"date": "2026-03-02"}, aged("2026-03-02", [], 0, 0, 0, 0))
    c.ok("createHousehold", sam, T0, {"householdId": "h1", "name": "Home", "payerId": "bob"})
    c.ok("addHouseholdMember", sam, T0, {"householdId": "h1", "memberId": "ada"})
    c.ok("householdStatement", sam, T0, {"householdId": "h1", "month": "2026-03"},
         stmt("h1", "2026-03", [srow("ada", None, 0, 0), srow("bob", None, 0, 0)]))
    generate(c, "2026-03", MARCH_AT)
    pays(c, "ada", "2026-04-02T09:00Z", "inv-1", 9900, "pay-1")
    pays(c, "sam", "2026-04-02T09:05Z", "inv-2", 1000, "pay-2")
    # cal joins after March; the household then has 3 members and gets the discount on the April membership lines
    c.ok("registerMember", sam, "2026-04-10T10:00Z", {"memberId": "cal", "name": "Cal Cole", "tier": "basic"})
    c.ok("addHouseholdMember", sam, "2026-04-10T10:00Z", {"householdId": "h1", "memberId": "cal"})
    i2, d2 = "2026-05-01T08:00Z", "2026-05-15T08:00Z"
    april = [simple("inv-4", "ada", "2026-04", i2, d2, [mline("plus", 4900, 10, 4410, i2)]),
             simple("inv-5", "bob", "2026-04", i2, d2, [mline("basic", 2500, 10, 2250, i2)]),
             simple("inv-6", "cal", "2026-04", i2, d2, [mline("basic", 2500, 10, 2250, i2)]),
             simple("inv-7", "eve", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)])]
    c.ok("generateInvoices", sam, i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": april})
    pays(c, "ada", "2026-05-02T09:00Z", "inv-4", 4410, "pay-3")
    pays(c, "eve", "2026-05-02T09:05Z", "inv-7", 9900, "pay-4")
    c.ok("removeHouseholdMember", sam, "2026-06-10T09:00Z", {"householdId": "h1", "memberId": "ada"})
    c.ok("addHouseholdMember", sam, "2026-06-11T09:00Z", {"householdId": "h1", "memberId": "eve"})
    t = "2026-06-20T10:00Z"
    # open: inv-2 (bob, 1500, due 04-15), inv-3 (eve, 9900, due 04-15), inv-5 (bob, 2250, due 05-15), inv-6 (cal, 2250, due 05-15)
    c.ok("agedReceivables", sam, t, {"date": "2026-04-15"},
         aged("2026-04-15", [arow("inv-2", "bob", 1500, 0), arow("inv-3", "eve", 9900, 0), arow("inv-5", "bob", 2250, 0),
                             arow("inv-6", "cal", 2250, 0)], 15900, 0, 0, 0))
    c.ok("agedReceivables", sam, t, {"date": "2026-05-15"},
         aged("2026-05-15", [arow("inv-2", "bob", 1500, 30), arow("inv-3", "eve", 9900, 30), arow("inv-5", "bob", 2250, 0),
                             arow("inv-6", "cal", 2250, 0)], 4500, 11400, 0, 0))
    c.ok("agedReceivables", "amy", t, {"date": "2026-05-16"},
         aged("2026-05-16", [arow("inv-2", "bob", 1500, 31), arow("inv-3", "eve", 9900, 31), arow("inv-5", "bob", 2250, 1),
                             arow("inv-6", "cal", 2250, 1)], 0, 4500, 11400, 0))
    c.ok("agedReceivables", sam, t, {"date": "2026-06-14"},
         aged("2026-06-14", [arow("inv-2", "bob", 1500, 60), arow("inv-3", "eve", 9900, 60), arow("inv-5", "bob", 2250, 30),
                             arow("inv-6", "cal", 2250, 30)], 0, 4500, 11400, 0))
    c.ok("agedReceivables", sam, t, {"date": "2026-06-15"},
         aged("2026-06-15", [arow("inv-2", "bob", 1500, 61), arow("inv-3", "eve", 9900, 61), arow("inv-5", "bob", 2250, 31),
                             arow("inv-6", "cal", 2250, 31)], 0, 0, 4500, 11400))
    c.refuse("agedReceivables", "ada", t, {"date": "2026-06-15"}, "forbidden")
    # the household has bob, cal and eve now: ada, who left, is not in the rows; cal has no March invoice
    c.ok("householdStatement", sam, t, {"householdId": "h1", "month": "2026-03"},
         stmt("h1", "2026-03", [srow("bob", "inv-2", 2500, 1500), srow("cal", None, 0, 0), srow("eve", "inv-3", 9900, 9900)]))
    c.ok("householdStatement", sam, t, {"householdId": "h1", "month": "2026-04"},
         stmt("h1", "2026-04", [srow("bob", "inv-5", 2250, 2250), srow("cal", "inv-6", 2250, 2250), srow("eve", "inv-7", 9900, 0)]))
    c.ok("householdStatement", "amy", t, {"householdId": "h1", "month": "2026-02"},
         stmt("h1", "2026-02", [srow("bob", None, 0, 0), srow("cal", None, 0, 0), srow("eve", None, 0, 0)]))
    c.refuse("householdStatement", sam, t, {"householdId": "h9", "month": "2026-03"}, "not_found")
    c.refuse("householdStatement", "bob", t, {"householdId": "h1", "month": "2026-03"}, "forbidden")
    c.audit_log(t)
    return c.build()


# ---------------------------------------------------------------- chains across modules


def x_money_1() -> dict:
    c = Case("X-money-1", "X-money",
             "the money chain through the journal: charges of several kinds, credit, invoices of three members, payments that overpay and part-pay, trial balance after each step")
    world(c)
    c.ok("createReservation", "ada", "2026-03-02T09:30Z",
         {"memberId": "ada", "machineId": "L1", "start": "2026-03-03T10:00Z", "end": "2026-03-03T11:00Z"}, new="res-1")
    c.ok("checkIn", "ada", "2026-03-03T10:00Z", {"reservationId": "res-1"})
    usage = chg("usage", "res-1", 37, 45, 10, 1499, "2026-03-03T10:37Z")
    c.ok("checkOut", "ada", "2026-03-03T10:37Z", {"reservationId": "res-1"}, has(charge=usage))
    c.ok("purchaseMaterial", "sam", "2026-03-05T12:00Z", {"memberId": "bob", "materialId": "ply", "quantity": 3}, None, new="pur-1")
    g = "2026-03-10T09:00Z"
    c.ok("issueCredit", "amy", g, {"memberId": "ada", "amountCents": 2000, "reason": "goodwill"})
    c.ok("getTrialBalance", "amy", g, {},
         tb({"creditLiability": (0, 2000), "expenseGoodwill": (2000, 0)}, 2000, 2000))
    issued, due = "2026-04-01T08:00Z", "2026-04-15T08:00Z"
    ada = [mline("plus", 4900, 0, 4900, issued), ada_course(), usage]
    bob = [mline("basic", 2500, 0, 2500, issued), chg("material", "pur-1", 3, 150, 0, 450, "2026-03-05T12:00Z")]
    invoices = [simple("inv-1", "ada", "2026-03", issued, due, ada, credit=2000),
                simple("inv-2", "bob", "2026-03", issued, due, bob),
                simple("inv-3", "eve", "2026-03", issued, due, [mline("pro", 9900, 0, 9900, issued)])]
    c.ok("generateInvoices", "sam", issued, {"month": "2026-03"}, {"month": "2026-03", "invoices": invoices})
    c.ok("getTrialBalance", "amy", issued, {},
         tb({"creditLiability": (2000, 2000), "expenseGoodwill": (2000, 0), "receivable": (24249, 2000),
             "revenueCourse": (0, 5000), "revenueMaterial": (0, 450), "revenueMembership": (0, 17300), "revenueUsage": (0, 1499)},
            28249, 28249))
    # the journal order across the invoices: lines of one invoice, then its credit entry, before the next invoice
    c.ok("listJournalEntries", "amy", issued, {"month": "2026-04"},
         {"month": "2026-04", "entries": [
             je(2, issued, "receivable", "revenueMembership", 4900, "inv-1"),
             je(3, issued, "receivable", "revenueCourse", 5000, "inv-1"),
             je(4, issued, "receivable", "revenueUsage", 1499, "inv-1"),
             je(5, issued, "creditLiability", "receivable", 2000, "inv-1"),
             je(6, issued, "receivable", "revenueMembership", 2500, "inv-2"),
             je(7, issued, "receivable", "revenueMaterial", 450, "inv-2"),
             je(8, issued, "receivable", "revenueMembership", 9900, "inv-3")]})
    c.ok("listJournalEntries", "amy", issued, {"month": "2026-03"}, {"month": "2026-03", "entries": [je(1, g, "expenseGoodwill", "creditLiability", 2000, "ada")]})
    # ada pays 10000 on a balance of 9399: 9399 applied, 601 credited
    p = "2026-04-05T10:00Z"
    ada_after = inv("inv-1", "ada", "2026-03", issued, due, ada, 11399, 2000, 9399, 0, "paid")
    pays(c, "ada", p, "inv-1", 10000, "pay-1",
         {"payment": pay("pay-1", "inv-1", "ada", 10000, 9399, 601, p), "invoice": ada_after, "creditBalanceCents": 601})
    c.ok("getTrialBalance", "amy", p, {},
         tb({"cash": (10000, 0), "creditLiability": (2000, 2601), "expenseGoodwill": (2000, 0), "receivable": (24249, 11399),
             "revenueCourse": (0, 5000), "revenueMaterial": (0, 450), "revenueMembership": (0, 17300), "revenueUsage": (0, 1499)},
            38249, 38249))
    # bob pays exactly (one entry), eve part-pays
    pays(c, "sam", "2026-04-05T10:10Z", "inv-2", 2950, "pay-2",
         {"payment": pay("pay-2", "inv-2", "bob", 2950, 2950, 0, "2026-04-05T10:10Z"),
          "invoice": inv("inv-2", "bob", "2026-03", issued, due, bob, 2950, 0, 2950, 0, "paid"), "creditBalanceCents": 0})
    pays(c, "eve", "2026-04-05T10:20Z", "inv-3", 5000, "pay-3",
         {"payment": pay("pay-3", "inv-3", "eve", 5000, 5000, 0, "2026-04-05T10:20Z"),
          "invoice": inv("inv-3", "eve", "2026-03", issued, due, [mline("pro", 9900, 0, 9900, issued)], 9900, 0, 5000, 4900, "open"),
          "creditBalanceCents": 0})
    t = "2026-04-05T10:30Z"
    c.ok("getTrialBalance", "amy", t, {},
         tb({"cash": (17950, 0), "creditLiability": (2000, 2601), "expenseGoodwill": (2000, 0), "receivable": (24249, 19349),
             "revenueCourse": (0, 5000), "revenueMaterial": (0, 450), "revenueMembership": (0, 17300), "revenueUsage": (0, 1499)},
            46199, 46199))
    c.ok("listJournalEntries", "amy", t, {"month": "2026-04"},
         {"month": "2026-04", "entries": [
             je(2, issued, "receivable", "revenueMembership", 4900, "inv-1"),
             je(3, issued, "receivable", "revenueCourse", 5000, "inv-1"),
             je(4, issued, "receivable", "revenueUsage", 1499, "inv-1"),
             je(5, issued, "creditLiability", "receivable", 2000, "inv-1"),
             je(6, issued, "receivable", "revenueMembership", 2500, "inv-2"),
             je(7, issued, "receivable", "revenueMaterial", 450, "inv-2"),
             je(8, issued, "receivable", "revenueMembership", 9900, "inv-3"),
             je(9, p, "cash", "receivable", 9399, "pay-1"),
             je(10, p, "cash", "creditLiability", 601, "pay-1"),
             je(11, "2026-04-05T10:10Z", "cash", "receivable", 2950, "pay-2"),
             je(12, "2026-04-05T10:20Z", "cash", "receivable", 5000, "pay-3")]})
    # the invariants: receivable equals the invoice balances (4900); creditLiability equals the credit balances (601)
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, acct("ada", 601, 0, 0, 0))
    c.ok("getAccount", "bob", t, {"memberId": "bob"}, acct("bob", 0, 0, 0, 0))
    c.ok("getAccount", "eve", t, {"memberId": "eve"}, acct("eve", 0, 4900, 0, 0))
    c.audit_log(t)
    return c.build()


def x_money_2() -> dict:
    c = Case("X-money-2", "X-money", "the household discount on the membership line, on and off as the household grows and shrinks")
    world(c)
    c.ok("registerMember", "sam", T0, {"memberId": "cal", "name": "Cal Cole", "tier": "basic"})
    c.ok("createHousehold", "sam", T0, {"householdId": "h1", "name": "Home", "payerId": "bob"})
    c.ok("addHouseholdMember", "sam", T0, {"householdId": "h1", "memberId": "eve"})
    # two members: no discount yet (nothing is invoiced); the third makes it
    c.ok("addHouseholdMember", "sam", "2026-03-15T09:00Z", {"householdId": "h1", "memberId": "ada"},
         {"household": {"householdId": "h1", "name": "Home", "payerId": "bob", "memberIds": ["ada", "bob", "eve"]}})
    i1, d1 = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    r1 = [simple("inv-1", "ada", "2026-03", i1, d1, [mline("plus", 4900, 10, 4410, i1), ada_course()]),
          simple("inv-2", "bob", "2026-03", i1, d1, [mline("basic", 2500, 10, 2250, i1)]),
          simple("inv-3", "cal", "2026-03", i1, d1, [mline("basic", 2500, 0, 2500, i1)]),
          simple("inv-4", "eve", "2026-03", i1, d1, [mline("pro", 9900, 10, 8910, i1)])]
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": r1})
    c.ok("removeHouseholdMember", "sam", "2026-04-10T09:00Z", {"householdId": "h1", "memberId": "eve"},
         {"household": {"householdId": "h1", "name": "Home", "payerId": "bob", "memberIds": ["ada", "bob"]}})
    i2, d2 = "2026-05-01T00:00Z", "2026-05-15T00:00Z"
    r2 = [simple("inv-5", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2)]),
          simple("inv-6", "bob", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2)]),
          simple("inv-7", "cal", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2)]),
          simple("inv-8", "eve", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)])]
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": r2})
    c.ok("addHouseholdMember", "sam", "2026-05-10T09:00Z", {"householdId": "h1", "memberId": "eve"})
    c.ok("addHouseholdMember", "sam", "2026-05-10T09:05Z", {"householdId": "h1", "memberId": "cal"},
         {"household": {"householdId": "h1", "name": "Home", "payerId": "bob", "memberIds": ["ada", "bob", "cal", "eve"]}})
    i3, d3 = "2026-06-01T00:00Z", "2026-06-15T00:00Z"
    r3 = [simple("inv-9", "ada", "2026-05", i3, d3, [mline("plus", 4900, 10, 4410, i3)]),
          simple("inv-10", "bob", "2026-05", i3, d3, [mline("basic", 2500, 10, 2250, i3)]),
          simple("inv-11", "cal", "2026-05", i3, d3, [mline("basic", 2500, 10, 2250, i3)]),
          simple("inv-12", "eve", "2026-05", i3, d3, [mline("pro", 9900, 10, 8910, i3)])]
    c.ok("generateInvoices", "sam", i3, {"month": "2026-05"}, {"month": "2026-05", "invoices": r3})
    t = "2026-06-01T00:05Z"
    # the statement shows the members of the household now, with each one's invoice for the month
    c.ok("householdStatement", "sam", t, {"householdId": "h1", "month": "2026-03"},
         stmt("h1", "2026-03", [srow("ada", "inv-1", 9410, 9410), srow("bob", "inv-2", 2250, 2250), srow("cal", "inv-3", 2500, 2500),
                                srow("eve", "inv-4", 8910, 8910)]))
    c.ok("householdStatement", "sam", t, {"householdId": "h1", "month": "2026-04"},
         stmt("h1", "2026-04", [srow("ada", "inv-5", 4900, 4900), srow("bob", "inv-6", 2500, 2500), srow("cal", "inv-7", 2500, 2500),
                                srow("eve", "inv-8", 9900, 9900)]))
    c.ok("householdStatement", "sam", t, {"householdId": "h1", "month": "2026-05"},
         stmt("h1", "2026-05", [srow("ada", "inv-9", 4410, 4410), srow("bob", "inv-10", 2250, 2250), srow("cal", "inv-11", 2250, 2250),
                                srow("eve", "inv-12", 8910, 8910)]))
    # 18070 + 19800 + 17820 of membership, 5000 of course
    c.ok("getTrialBalance", "amy", t, {}, tb({"receivable": (60690, 0), "revenueCourse": (0, 5000), "revenueMembership": (0, 55690)}, 60690, 60690))
    c.audit_log(t)
    return c.build()


def x_money_3() -> dict:
    c = Case("X-money-3", ["X-money", "B-deposits"],
             "a deposit taken, invoiced, refused release while unpaid, paid, released into credit, used by the next invoice; postings")
    world(c)
    t0 = "2026-03-02T10:00Z"
    c.ok("takeDeposit", "sam", t0, {"memberId": "bob", "purpose": "bench key", "amountCents": 3000}, None, new="dep-1")
    c.refuse("releaseDeposit", "sam", t0, {"depositId": "dep-1"}, "deposit_unpaid")
    i1, d1 = "2026-04-01T08:00Z", "2026-04-15T08:00Z"
    march = [simple("inv-1", "ada", "2026-03", i1, d1, [mline("plus", 4900, 0, 4900, i1), ada_course()]),
             simple("inv-2", "bob", "2026-03", i1, d1, [mline("basic", 2500, 0, 2500, i1), chg("deposit", "dep-1", 1, 3000, 0, 3000, t0)]),
             simple("inv-3", "eve", "2026-03", i1, d1, [mline("pro", 9900, 0, 9900, i1)])]
    assert march[1]["totalCents"] == 5500
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": march})
    c.ok("getTrialBalance", "amy", i1, {},
         tb({"depositLiability": (0, 3000), "receivable": (25300, 0), "revenueCourse": (0, 5000), "revenueMembership": (0, 17300)},
            25300, 25300))
    c.refuse("releaseDeposit", "sam", i1, {"depositId": "dep-1"}, "deposit_unpaid")
    # a suspended member can pay; paying all but one cent still leaves the deposit unpaid
    c.ok("setMemberStanding", "sam", "2026-04-02T08:00Z", {"memberId": "bob", "standing": "suspended"})
    pays(c, "bob", "2026-04-02T09:00Z", "inv-2", 5499, "pay-1")
    c.refuse("releaseDeposit", "sam", "2026-04-02T09:01Z", {"depositId": "dep-1"}, "deposit_unpaid")
    pays(c, "bob", "2026-04-02T09:05Z", "inv-2", 1, "pay-2")
    r = "2026-04-03T10:00Z"
    c.ok("releaseDeposit", "sam", r, {"depositId": "dep-1"},
         {"deposit": deposit("dep-1", "bob", "bench key", 3000, "released", t0, r), "creditBalanceCents": 3000})
    c.refuse("releaseDeposit", "sam", r, {"depositId": "dep-1"}, "invalid_state")
    c.ok("listDeposits", "bob", r, {"memberId": "bob"},
         {"memberId": "bob", "deposits": [deposit("dep-1", "bob", "bench key", 3000, "released", t0, r)]})
    c.ok("getTrialBalance", "amy", r, {},
         tb({"cash": (5500, 0), "creditLiability": (0, 3000), "depositLiability": (3000, 3000), "receivable": (25300, 5500),
             "revenueCourse": (0, 5000), "revenueMembership": (0, 17300)}, 33800, 33800))
    # the released deposit is credit: the next invoice uses 2500 of the 3000
    i2, d2 = "2026-05-01T08:00Z", "2026-05-15T08:00Z"
    april = [simple("inv-4", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2)]),
             simple("inv-5", "bob", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2)], credit=2500),
             simple("inv-6", "eve", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)])]
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": april})
    c.ok("getAccount", "bob", i2, {"memberId": "bob"}, acct("bob", 500, 0, 0, 0))
    c.ok("getTrialBalance", "amy", i2, {},
         tb({"cash": (5500, 0), "creditLiability": (2500, 3000), "depositLiability": (3000, 3000), "receivable": (42600, 8000),
             "revenueCourse": (0, 5000), "revenueMembership": (0, 34600)}, 53600, 53600))
    c.ok("listJournalEntries", "amy", i2, {"month": "2026-04"},
         {"month": "2026-04", "entries": [
             je(1, i1, "receivable", "revenueMembership", 4900, "inv-1"),
             je(2, i1, "receivable", "revenueCourse", 5000, "inv-1"),
             je(3, i1, "receivable", "revenueMembership", 2500, "inv-2"),
             je(4, i1, "receivable", "depositLiability", 3000, "inv-2"),
             je(5, i1, "receivable", "revenueMembership", 9900, "inv-3"),
             je(6, "2026-04-02T09:00Z", "cash", "receivable", 5499, "pay-1"),
             je(7, "2026-04-02T09:05Z", "cash", "receivable", 1, "pay-2"),
             je(8, r, "depositLiability", "creditLiability", 3000, "dep-1")]})
    c.audit_log(i2)
    return c.build()


def x_money_4() -> dict:
    c = Case("X-money-4", ["X-money", "B-passes"],
             "a pass bought and drawn at check-out: a usage line of 0 cents posts nothing; the pass line, a payment that overpays")
    world(c)
    c.ok("addPassProduct", "amy", "2026-03-02T09:05Z", {"productId": "p30", "name": "Thirty", "minutes": 30, "priceCents": 3000, "validDays": 60})
    s = "2026-03-03T08:00Z"
    c.ok("sellPass", "sam", s, {"memberId": "ada", "productId": "p30"},
         {"pass": passrec("pas-1", "ada", "p30", 30, "2026-03-03", "2026-05-02"), "charge": chg("pass", "pas-1", 1, 3000, 0, 3000, s)},
         new="pas-1")
    # 20 metered minutes, all from the pass; then 25 metered minutes, 10 from the pass and 15 billable: 15 x 45 less 10 % is 608
    c.ok("createReservation", "ada", "2026-03-03T08:30Z",
         {"memberId": "ada", "machineId": "L1", "start": "2026-03-04T10:00Z", "end": "2026-03-04T11:00Z"}, new="res-1")
    c.ok("createReservation", "ada", "2026-03-03T08:31Z",
         {"memberId": "ada", "machineId": "L1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z"}, new="res-2")
    c.ok("checkIn", "ada", "2026-03-04T10:00Z", {"reservationId": "res-1"})
    u1 = chg("usage", "res-1", 0, 45, 10, 0, "2026-03-04T10:20Z")
    c.ok("checkOut", "ada", "2026-03-04T10:20Z", {"reservationId": "res-1"},
         {"reservation": resv("res-1", "ada", "L1", "2026-03-04T10:00Z", "2026-03-04T11:00Z", "completed", "2026-03-04T10:00Z",
                              "2026-03-04T10:20Z", 45, 20, 0), "charge": u1})
    c.ok("checkIn", "ada", "2026-03-05T10:00Z", {"reservationId": "res-2"})
    u2 = chg("usage", "res-2", 15, 45, 10, 608, "2026-03-05T10:25Z")
    c.ok("checkOut", "ada", "2026-03-05T10:25Z", {"reservationId": "res-2"},
         {"reservation": resv("res-2", "ada", "L1", "2026-03-05T10:00Z", "2026-03-05T11:00Z", "completed", "2026-03-05T10:00Z",
                              "2026-03-05T10:25Z", 45, 25, 608), "charge": u2})
    c.ok("listMemberPasses", "ada", "2026-03-05T10:25Z", {"memberId": "ada"},
         {"memberId": "ada", "passes": [passrec("pas-1", "ada", "p30", 0, "2026-03-03", "2026-05-02")]})
    i1, d1 = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    ada = [mline("plus", 4900, 0, 4900, i1), ada_course(), chg("pass", "pas-1", 1, 3000, 0, 3000, s), u1, u2]
    march = [simple("inv-1", "ada", "2026-03", i1, d1, ada),
             simple("inv-2", "bob", "2026-03", i1, d1, [mline("basic", 2500, 0, 2500, i1)]),
             simple("inv-3", "eve", "2026-03", i1, d1, [mline("pro", 9900, 0, 9900, i1)])]
    assert march[0]["totalCents"] == 13508
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": march})
    # six entries: the usage line of 0 cents posts nothing and takes no number
    c.ok("listJournalEntries", "amy", i1, {"month": "2026-04"},
         {"month": "2026-04", "entries": [
             je(1, i1, "receivable", "revenueMembership", 4900, "inv-1"),
             je(2, i1, "receivable", "revenueCourse", 5000, "inv-1"),
             je(3, i1, "receivable", "revenuePasses", 3000, "inv-1"),
             je(4, i1, "receivable", "revenueUsage", 608, "inv-1"),
             je(5, i1, "receivable", "revenueMembership", 2500, "inv-2"),
             je(6, i1, "receivable", "revenueMembership", 9900, "inv-3")]})
    c.ok("getTrialBalance", "amy", i1, {},
         tb({"receivable": (25908, 0), "revenueCourse": (0, 5000), "revenueMembership": (0, 17300), "revenuePasses": (0, 3000),
             "revenueUsage": (0, 608)}, 25908, 25908))
    p = "2026-04-02T09:00Z"
    pays(c, "ada", p, "inv-1", 14000, "pay-1",
         {"payment": pay("pay-1", "inv-1", "ada", 14000, 13508, 492, p), "invoice": inv("inv-1", "ada", "2026-03", i1, d1, ada, 13508, 0, 13508, 0, "paid"),
          "creditBalanceCents": 492})
    c.ok("getTrialBalance", "amy", p, {},
         tb({"cash": (14000, 0), "creditLiability": (0, 492), "receivable": (25908, 13508), "revenueCourse": (0, 5000),
             "revenueMembership": (0, 17300), "revenuePasses": (0, 3000), "revenueUsage": (0, 608)}, 39908, 39908))
    c.ok("getAccount", "ada", p, {"memberId": "ada"}, acct("ada", 492, 0, 0, 0))
    c.audit_log(p)
    return c.build()


def x_money_5() -> dict:
    c = Case("X-money-5", ["X-money", "B-refunds"],
             "a refund bounded by earlier refunds, given as credit, used by the next invoice; refund of a line paid by credit; charges by kind")
    world(c)
    i1, d1 = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    generate(c, "2026-03", i1)
    pays(c, "ada", "2026-04-02T09:00Z", "inv-1", 9900, "pay-1")
    t = "2026-04-02T10:00Z"
    c.ok("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 3000, "reason": "course moved"},
         {"refund": refund("rfd-1", "inv-1", 2, 3000, "course moved", t), "creditBalanceCents": 3000}, new="rfd-1")
    c.refuse("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 2001, "reason": "more"}, "refund_too_large")
    c.ok("issueRefund", "amy", t, {"invoiceId": "inv-1", "lineNumber": 2, "amountCents": 2000, "reason": "rest"},
         {"refund": refund("rfd-2", "inv-1", 2, 2000, "rest", t), "creditBalanceCents": 5000}, new="rfd-2")
    # the next invoice takes 4900 of the 5000; the invoice paid by credit alone can itself be refunded
    i2, d2 = "2026-05-01T00:00Z", "2026-05-15T00:00Z"
    april = [simple("inv-4", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2)], credit=4900),
             simple("inv-5", "bob", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2)]),
             simple("inv-6", "eve", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)])]
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": april})
    c.ok("getAccount", "ada", i2, {"memberId": "ada"}, acct("ada", 100, 0, 0, 0))
    r = "2026-05-01T09:00Z"
    c.ok("issueRefund", "amy", r, {"invoiceId": "inv-4", "lineNumber": 1, "amountCents": 4900, "reason": "fee waived"},
         {"refund": refund("rfd-3", "inv-4", 1, 4900, "fee waived", r), "creditBalanceCents": 5000}, new="rfd-3")
    c.refuse("issueRefund", "amy", r, {"invoiceId": "inv-4", "lineNumber": 1, "amountCents": 1, "reason": "again"}, "refund_too_large")
    c.refuse("issueRefund", "amy", r, {"invoiceId": "inv-5", "lineNumber": 1, "amountCents": 1, "reason": "open"}, "invoice_not_paid")
    c.ok("chargesByKind", "sam", r, {"month": "2026-03"},
         {"month": "2026-03", "rows": kinds({"membership": 17300, "course": 5000}, {"course": 5000}, 22300), "totalCents": 22300})
    c.ok("chargesByKind", "sam", r, {"month": "2026-04"},
         {"month": "2026-04", "rows": kinds({"membership": 17300}, {"membership": 4900}, 17300), "totalCents": 17300})
    c.ok("getTrialBalance", "amy", r, {},
         tb({"cash": (9900, 0), "creditLiability": (4900, 9900), "receivable": (39600, 14800), "revenueCourse": (0, 5000),
             "revenueMembership": (0, 34600), "revenueRefunds": (9900, 0)}, 64300, 64300))
    c.ok("listJournalEntries", "amy", r, {"month": "2026-04"},
         {"month": "2026-04", "entries": [
             je(1, i1, "receivable", "revenueMembership", 4900, "inv-1"),
             je(2, i1, "receivable", "revenueCourse", 5000, "inv-1"),
             je(3, i1, "receivable", "revenueMembership", 2500, "inv-2"),
             je(4, i1, "receivable", "revenueMembership", 9900, "inv-3"),
             je(5, "2026-04-02T09:00Z", "cash", "receivable", 9900, "pay-1"),
             je(6, t, "revenueRefunds", "creditLiability", 3000, "rfd-1"),
             je(7, t, "revenueRefunds", "creditLiability", 2000, "rfd-2")]})
    c.ok("listJournalEntries", "amy", r, {"month": "2026-05"},
         {"month": "2026-05", "entries": [
             je(8, i2, "receivable", "revenueMembership", 4900, "inv-4"),
             je(9, i2, "creditLiability", "receivable", 4900, "inv-4"),
             je(10, i2, "receivable", "revenueMembership", 2500, "inv-5"),
             je(11, i2, "receivable", "revenueMembership", 9900, "inv-6"),
             je(12, r, "revenueRefunds", "creditLiability", 4900, "rfd-3")]})
    c.audit_log(r)
    return c.build()


def x_money_6() -> dict:
    c = Case("X-money-6", ["X-money", "B-dunning"],
             "dunning through notice, fee and suspension in one run; a reservation refused; paying reinstates nobody; the fee on the next invoice")
    world(c)
    generate(c, "2026-03", MARCH_AT)
    c.ok("setDunningLadder", "amy", "2026-04-02T09:00Z", {"noticeDays": 3, "feeDays": 7, "feeCents": 500, "suspendDays": 14})
    pays(c, "eve", "2026-04-02T09:10Z", "inv-3", 9900, "pay-1")
    t = "2026-04-30T08:00Z"
    c.ok("runDunning", "sam", t, {},
         {"actions": [action("inv-1", "notice"), action("inv-1", "fee"), action("inv-1", "suspend"),
                      action("inv-2", "notice"), action("inv-2", "fee"), action("inv-2", "suspend")]})
    c.ok("listNotices", "ada", t, {"memberId": "ada"},
         {"memberId": "ada", "notices": [notice(1, "ada", "dunning", "inv-1", t), notice(2, "ada", "suspension", "inv-1", t)]})
    c.ok("listNotices", "bob", t, {"memberId": "bob"},
         {"memberId": "bob", "notices": [notice(3, "bob", "dunning", "inv-2", t), notice(4, "bob", "suspension", "inv-2", t)]})
    c.ok("listDunningCases", "ada", t, {"memberId": "ada"}, {"memberId": "ada", "cases": [case_rec(1, "inv-1", "ada", t, t, t)]})
    c.ok("getMember", "sam", t, {"memberId": "ada"}, member_rec("ada", "Ada Byron", "plus", "suspended", T0))
    slot = {"machineId": "L1", "start": "2026-05-04T14:00Z", "end": "2026-05-04T15:00Z"}
    c.refuse("createReservation", "ada", "2026-04-30T09:00Z", {"memberId": "ada", **slot}, "member_suspended")
    c.refuse("createReservation", "sam", "2026-04-30T09:00Z", {"memberId": "bob", **slot}, "member_suspended")
    p = "2026-04-30T09:10Z"
    ada_lines = [mline("plus", 4900, 0, 4900, MARCH_AT), ada_course()]
    pays(c, "ada", p, "inv-1", 9900, "pay-2",
         {"payment": pay("pay-2", "inv-1", "ada", 9900, 9900, 0, p),
          "invoice": inv("inv-1", "ada", "2026-03", MARCH_AT, MARCH_DUE, ada_lines, 9900, 0, 9900, 0, "paid"),
          "creditBalanceCents": 0})
    pays(c, "bob", "2026-04-30T09:15Z", "inv-2", 2500, "pay-3")
    # paid, and still suspended
    c.refuse("createReservation", "ada", "2026-04-30T09:20Z", {"memberId": "ada", **slot}, "member_suspended")
    c.ok("getMember", "sam", "2026-04-30T09:30Z", {"memberId": "bob"}, member_rec("bob", "Bob Stone", "basic", "suspended", T0))
    c.ok("runDunning", "sam", "2026-04-30T09:30Z", {}, {"actions": []})
    # the late fees (500 each, created at the run) are in April and on the next invoices
    i2, d2 = "2026-05-01T00:00Z", "2026-05-15T00:00Z"
    fee1, fee2 = chg("lateFee", "inv-1", 1, 500, 0, 500, t), chg("lateFee", "inv-2", 1, 500, 0, 500, t)
    april = [simple("inv-4", "ada", "2026-04", i2, d2, [mline("plus", 4900, 0, 4900, i2), fee1]),
             simple("inv-5", "bob", "2026-04", i2, d2, [mline("basic", 2500, 0, 2500, i2), fee2]),
             simple("inv-6", "eve", "2026-04", i2, d2, [mline("pro", 9900, 0, 9900, i2)])]
    assert [i["totalCents"] for i in april] == [5400, 3000, 9900]
    c.ok("generateInvoices", "sam", i2, {"month": "2026-04"}, {"month": "2026-04", "invoices": april})
    c.ok("getTrialBalance", "amy", i2, {},
         tb({"cash": (22300, 0), "receivable": (40600, 22300), "revenueCourse": (0, 5000), "revenueLateFees": (0, 1000),
             "revenueMembership": (0, 34600)}, 62900, 62900))
    # reinstating by staff is what lifts the suspension
    c.ok("setMemberStanding", "sam", "2026-05-01T09:00Z", {"memberId": "ada", "standing": "good"})
    r = "2026-05-01T09:05Z"
    c.ok("createReservation", "ada", r, {"memberId": "ada", **slot},
         {"reservation": resv("res-1", "ada", "L1", slot["start"], slot["end"], "reserved")}, new="res-1")
    c.refuse("createReservation", "sam", r, {"memberId": "bob", **slot, "machineId": "T1"}, "member_suspended")
    c.audit_log(r)
    return c.build()


def build_cases() -> list:
    return [b_ledger_1(), b_ledger_2(), b_billing_1(), b_billing_2(), b_billing_3(), b_billing_4(), b_deposits_1(),
            b_deposits_2(), b_passes_1(), b_passes_2(), b_promotions_1(), b_promotions_2(), b_scholarships_1(),
            b_scholarships_2(), b_refunds_1(), b_refunds_2(), b_dunning_1(), b_dunning_2(), b_lockers_1(), b_lockers_2(),
            b_finance_1(), b_finance_2(), x_money_1(), x_money_2(), x_money_3(), x_money_4(), x_money_5(), x_money_6()]
