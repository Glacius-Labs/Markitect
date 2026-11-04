"""Change step C7 of the large Makerspace: prepaid minutes become packs.

A renaming and nothing else. Operations `addPassProduct`, `sellPass`, `listMemberPasses` become `addPackProduct`, `sellPack`,
`listMemberPacks`; the field `passId` becomes `packId`; the results `pass` and `passes` become `pack` and `packs`; the charge kind
`pass` becomes `pack`; the ledger account `revenuePasses` becomes `revenuePacks`; the identifiers `pas-N` become `pak-N`, the same
counter. The old names are gone without aliases. Guest passes and the words for passing an inspection or an assessment stay:
`guestPassId`, `guestPass`, `guestPasses`, `issueGuestPass`, `revokeGuestPass`, `listGuestPasses`, `pass_not_today`,
`pass_of_another_member`, charge kind `guest`, `revenueGuests`, `passMark`, `passed`, `lastPassedOn`, `assessment_not_passed`.
Written from the requirements document after the change, with every expected value stated by hand.

What this step retires or moves: nothing is retired. `transform` renames, in every earlier case, exactly the names above and nothing
else: the three operations (in requests and in the `op` of audit entries), the keys `passId`, `pass` and `passes`, the identifiers
`pas-N` wherever they stand as a whole value (record ids, charge refs, audit targets), the charge kind `pass` in charges, invoice
lines and charges-by-kind rows, and the account `revenuePasses` in trial balance rows, journal entries and adjustments. The
orders hold by themselves: `pack` takes the place of `pass` in the kind order, and `revenueNoShows` < `revenuePacks` <
`revenueRefunds` is the old place of `revenuePasses`, so no row moves. Free text of a case (a refund's reason "pass returned")
and the case ids and families keep their words. The cite of eight cases that name the old words is updated. The generated cases
come from the facts.

How the cases are cut: the renamed interface end to end (`C7-names`), the old names answering `invalidRequest` (`C7-old`), the
counter and the list order across ten (`C7-id`), a chain from sale through draw, invoice, journal, trial balance, payment and
refund (`C7-chain`), the ledger account (`C7-ledger`), and what must not be renamed (`C7-keep`).

Not tested because the requirements leave it undecided or the code cannot tell: the words of the refusal reasons and of the
reason an invalid request gets.

Dates the cases rely on: 2026-03-02 is a Monday (W, the world's time). A 30-day pack sold on 03-02 expires on 04-01, a 60-day pack
sold on 03-03 on 05-02, a 10-day pack sold on 03-02 on 03-12. The world's rates: L1 45, L2 60, T1 30 cents a minute. Tiers: ada
plus (usage 10 %), bob basic, eve pro. Joining the laser course costs 5000 cents (a `course` charge at W), so ada has 5000 cents
unbilled before anything else happens.
"""
from __future__ import annotations

import re
from typing import Any

from kit import Case, world

W = "2026-03-02T09:00Z"  # Monday, the world's time

RETIRED: dict[str, str] = {}

OPS_RENAMED = {"addPassProduct": "addPackProduct", "sellPass": "sellPack", "listMemberPasses": "listMemberPacks"}
KEYS_RENAMED = {"passId": "packId", "pass": "pack", "passes": "packs"}
OLD_ID = re.compile(r"^pas-(\d+)$")
CITE_WORDS = (("Passes", "Packs"), ("passes", "packs"), ("Pass", "Pack"), ("pass", "pack"))
CITE_CASES = {"B-passes-1", "B-passes-2", "X-money-4", "X-floor-3", "C3-ledger-1", "C3-keep-1", "C4-gate-pass", "C4-keep-pass"}


def renamed(node: Any, key: str | None = None) -> Any:
    """The node with the renamed words. Only these exact words move; `guestPassId`, `passed`, `passMark` and the rest are other words."""
    if isinstance(node, dict):
        return {KEYS_RENAMED.get(k, k): renamed(v, k) for k, v in node.items()}
    if isinstance(node, list):
        return [renamed(v, key) for v in node]
    if isinstance(node, str):
        if key == "op":
            return OPS_RENAMED.get(node, node)
        if key == "kind" and node == "pass":
            return "pack"
        if node == "revenuePasses":
            return "revenuePacks"
        found = OLD_ID.match(node)
        if found:
            return f"pak-{found.group(1)}"
    return node


def transform(case: dict[str, Any]) -> dict[str, Any]:
    """Every name the change renames reads with its new name in an earlier case; the behaviour and everything else stay."""
    case["steps"] = [renamed(step) for step in case["steps"]]
    if case["id"] in CITE_CASES:
        for cite in case["cite"]:
            text = cite["text"]
            for old, new in CITE_WORDS:
                text = text.replace(old, new)
            cite["text"] = text
    return case


ACCOUNTS = ["cash", "creditLiability", "depositLiability", "expenseGoodwill", "expenseMaker", "expensePayout",
            "expenseVolunteer", "expenseWriteOff", "inventory", "payable", "receivable", "revenueCourse", "revenueDeposits",
            "revenueEvents", "revenueGuests", "revenueJobs", "revenueLateFees", "revenueLockers", "revenueMaterial",
            "revenueMembership", "revenueNoShows", "revenuePacks", "revenueRefunds", "revenueUsage"]
KINDS = ["membership", "usage", "material", "course", "deposit", "pack", "locker", "guest", "lateFee", "event", "noShow"]


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


# ---- record builders (the members of each record, from the requirements) ---------------------------------------------

def product(pid, name, minutes, price, days):
    return {"productId": pid, "name": name, "minutes": minutes, "priceCents": price, "validDays": days}


def pack(pid, mem, prod, left, issued, expires):
    return {"packId": pid, "memberId": mem, "productId": prod, "minutesLeft": left, "issuedOn": issued, "expiresOn": expires}


def gpass(gid, mem, guest, date, charged, status="issued"):
    return {"guestPassId": gid, "memberId": mem, "guestName": guest, "date": date, "charged": charged, "status": status}


def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def chg(kind, ref_, qty, unit, disc, amt, at):
    assert (qty * unit * (100 - disc) + 50) // 100 == amt, (kind, ref_, qty, unit, disc, amt)
    return {"kind": kind, "ref": ref_, "quantity": qty, "unitPriceCents": unit, "discountPercent": disc,
            "amountCents": amt, "at": at}


def invoice(iid, mem, month, issued, due, lines, total):
    assert sum(line["amountCents"] for line in lines) == total, (iid, total)
    return {"invoiceId": iid, "memberId": mem, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": 0, "paidCents": 0, "balanceCents": total, "status": "open",
            "overdue": False}


def account(mem, unbilled):
    return {"account": {"memberId": mem, "creditBalanceCents": 0, "openBalanceCents": 0, "overdueBalanceCents": 0,
                        "unbilledCents": unbilled}}


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


P20 = product("p20", "Twenty", 20, 1000, 30)


# ---- C7-names: the renamed interface end to end ----------------------------------------------------------------------------

def names_1() -> Case:
    c = Case("C7-names-1", "C7-names", "Add pack product, Sell pack, List member packs, Pack, Charge: the three operations under their new names with the "
                                       "result members `product`, `pack`, `packs` and the field `packId`; the charge has kind `pack` and ref `pak-N`; "
                                       "the audit entry has the new `op` and, for a sale, the new `packId` as target")
    world(c)
    c.refuse("addPackProduct", "sam", W, dict(P20), "forbidden")
    c.ok("addPackProduct", "amy", W, dict(P20), {"product": P20})
    c.refuse("addPackProduct", "amy", W, {**P20, "name": "Other"}, "already_exists")
    c.ok("sellPack", "ada", W, {"memberId": "ada", "productId": "p20"}, {
        "pack": pack("pak-1", "ada", "p20", 20, "2026-03-02", "2026-04-01"), "charge": chg("pack", "pak-1", 1, 1000, 0, 1000, W)}, new="pak-1")
    c.ok("sellPack", "sam", W, {"memberId": "bob", "productId": "p20"}, {
        "pack": pack("pak-2", "bob", "p20", 20, "2026-03-02", "2026-04-01"), "charge": chg("pack", "pak-2", 1, 1000, 0, 1000, W)}, new="pak-2")
    c.refuse("sellPack", "ada", W, {"memberId": "bob", "productId": "p20"}, "forbidden")
    c.refuse("sellPack", "sam", W, {"memberId": "bob", "productId": "nope"}, "not_found")
    c.ok("listMemberPacks", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "packs": [
        pack("pak-1", "ada", "p20", 20, "2026-03-02", "2026-04-01")]})
    c.ok("listMemberPacks", "sam", W, {"memberId": "bob"}, {"memberId": "bob", "packs": [
        pack("pak-2", "bob", "p20", 20, "2026-03-02", "2026-04-01")]})
    c.ok("listMemberPacks", "amy", W, {"memberId": "eve"}, {"memberId": "eve", "packs": []})
    c.refuse("listMemberPacks", "ada", W, {"memberId": "bob"}, "forbidden")
    c.ok("getAccount", "ada", W, {"memberId": "ada"}, account("ada", 6000))
    c.audit_log(W)
    return c


# ---- C7-old: the old names are gone ---------------------------------------------------------------------------------------

def old_1() -> Case:
    c = Case("C7-old-1", "C7-old", "Invalid requests: `addPassProduct`, `sellPass` and `listMemberPasses` name no operation any more, so each is answered "
                                   "invalidRequest, with the arguments and the actors that were right before; they change nothing, create no product and "
                                   "no pack, take no number and cause no audit entry; there are no aliases")
    world(c)
    c.ok("addPackProduct", "amy", W, dict(P20), {"product": P20})
    c.ok("sellPack", "ada", W, {"memberId": "ada", "productId": "p20"}, {
        "pack": pack("pak-1", "ada", "p20", 20, "2026-03-02", "2026-04-01"), "charge": chg("pack", "pak-1", 1, 1000, 0, 1000, W)}, new="pak-1")
    c.invalid("addPassProduct", "amy", W, product("p21", "Twentyone", 21, 1100, 30))
    c.invalid("sellPass", "ada", W, {"memberId": "ada", "productId": "p20"})
    c.invalid("sellPass", "sam", W, {"memberId": "bob", "productId": "p20"})
    c.invalid("listMemberPasses", "ada", W, {"memberId": "ada"})
    c.invalid("listMemberPasses", "sam", W, {"memberId": "bob"})
    c.refuse("sellPack", "sam", W, {"memberId": "bob", "productId": "p21"}, "not_found")  # addPassProduct made no product
    c.ok("listMemberPacks", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "packs": [
        pack("pak-1", "ada", "p20", 20, "2026-03-02", "2026-04-01")]})
    c.ok("listMemberPacks", "sam", W, {"memberId": "bob"}, {"memberId": "bob", "packs": []})
    c.ok("sellPack", "ada", W, {"memberId": "ada", "productId": "p20"}, {
        "pack": pack("pak-2", "ada", "p20", 20, "2026-03-02", "2026-04-01"), "charge": chg("pack", "pak-2", 1, 1000, 0, 1000, W)}, new="pak-2")
    c.ok("getAccount", "ada", W, {"memberId": "ada"}, account("ada", 7000))
    c.audit_log(W)
    return c


# ---- C7-id: the counter and the list order -----------------------------------------------------------------------------------

def id_1() -> Case:
    c = Case("C7-id-1", "C7-id", "Initial state, List order, List member packs: the identifiers are `pak-1`, `pak-2`, … from one counter, in decimal without "
                                 "leading zeros; a refused sale and a guest pass take no pack number; the list is in ascending pack number, `pak-10` after "
                                 "`pak-9`, not by text")
    world(c)
    c.ok("setGuestAllowance", "amy", W, {"tier": "plus", "perMonth": 5, "feeCents": 0}, {"allowance": {"tier": "plus", "perMonth": 5, "feeCents": 0}})
    c.ok("addPackProduct", "amy", W, product("p5", "Five", 5, 100, 10), {"product": product("p5", "Five", 5, 100, 10)})
    for n in range(1, 11):
        c.ok("sellPack", "ada", W, {"memberId": "ada", "productId": "p5"}, {
            "pack": pack(f"pak-{n}", "ada", "p5", 5, "2026-03-02", "2026-03-12"), "charge": chg("pack", f"pak-{n}", 1, 100, 0, 100, W)}, new=f"pak-{n}")
        if n == 3:
            c.ok("issueGuestPass", "ada", W, {"memberId": "ada", "guestName": "Gus", "date": "2026-03-10"},
                 {"guestPass": gpass("gst-1", "ada", "Gus", "2026-03-10", False)}, new="gst-1")
        if n == 6:
            c.refuse("sellPack", "ada", W, {"memberId": "ada", "productId": "nope"}, "not_found")
    c.ok("sellPack", "sam", W, {"memberId": "bob", "productId": "p5"}, {
        "pack": pack("pak-11", "bob", "p5", 5, "2026-03-02", "2026-03-12"), "charge": chg("pack", "pak-11", 1, 100, 0, 100, W)}, new="pak-11")
    c.ok("listMemberPacks", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "packs": [
        pack(f"pak-{n}", "ada", "p5", 5, "2026-03-02", "2026-03-12") for n in range(1, 11)]})
    c.ok("listMemberPacks", "bob", W, {"memberId": "bob"}, {"memberId": "bob", "packs": [
        pack("pak-11", "bob", "p5", 5, "2026-03-02", "2026-03-12")]})
    c.ok("getAccount", "ada", W, {"memberId": "ada"}, account("ada", 6000))
    c.audit_log(W)
    return c


# ---- C7-chain: sale, draw, invoice, ledger, report ----------------------------------------------------------------------------

def chain_1() -> Case:
    c = Case("C7-chain-1", "C7-chain", "Sell pack, Pack draw, Check out, Charges and invoices, Postings, Get trial balance, Charges by kind: a pack sold, drawn "
                                       "at check-out (30 of 40 minutes, 10 billable), invoiced with a line of kind `pack` and ref `pak-1`, posted to "
                                       "`revenuePacks`, shown in the trial balance row `revenuePacks` and in the `pack` row of the kinds report")
    world(c)
    c.ok("addPackProduct", "amy", W, product("p30", "Thirty", 30, 3000, 60), {"product": product("p30", "Thirty", 30, 3000, 60)})
    s = t(3, "08:00")
    c.ok("sellPack", "sam", s, {"memberId": "ada", "productId": "p30"}, {
        "pack": pack("pak-1", "ada", "p30", 30, "2026-03-03", "2026-05-02"), "charge": chg("pack", "pak-1", 1, 3000, 0, 3000, s)}, new="pak-1")
    c.ok("createReservation", "ada", t(3, "08:30"), {"memberId": "ada", "machineId": "L1", "start": t(4, "10:00"), "end": t(4, "11:00")},
         {"reservation": res("res-1", "ada", "L1", t(4, "10:00"), t(4, "11:00"))}, new="res-1")
    c.ok("checkIn", "ada", t(4, "10:00"), {"reservationId": "res-1"}, {"reservation": res(
        "res-1", "ada", "L1", t(4, "10:00"), t(4, "11:00"), "checkedIn", cin=t(4, "10:00"), rate=45)})
    use = chg("usage", "res-1", 10, 45, 10, 405, t(4, "10:40"))
    c.ok("checkOut", "ada", t(4, "10:40"), {"reservationId": "res-1"}, {
        "reservation": res("res-1", "ada", "L1", t(4, "10:00"), t(4, "11:00"), "completed", t(4, "10:00"), t(4, "10:40"), 45, 40, 405),
        "charge": use})
    c.ok("listMemberPacks", "ada", t(4, "10:40"), {"memberId": "ada"}, {"memberId": "ada", "packs": [
        pack("pak-1", "ada", "p30", 0, "2026-03-03", "2026-05-02")]})
    i1, d1 = t(1, "00:00", 4), t(15, "00:00", 4)
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", i1, d1, [chg("membership", "plus", 1, 4900, 0, 4900, i1), chg("course", "laser-101", 1, 5000, 0, 5000, W),
                                                     chg("pack", "pak-1", 1, 3000, 0, 3000, s), use], 13305),
        invoice("inv-2", "bob", "2026-03", i1, d1, [chg("membership", "basic", 1, 2500, 0, 2500, i1)], 2500),
        invoice("inv-3", "eve", "2026-03", i1, d1, [chg("membership", "pro", 1, 9900, 0, 9900, i1)], 9900)]})
    c.ok("listJournalEntries", "amy", i1, {"month": "2026-04"}, {"month": "2026-04", "entries": [
        je(1, i1, "receivable", "revenueMembership", 4900, "inv-1"), je(2, i1, "receivable", "revenueCourse", 5000, "inv-1"),
        je(3, i1, "receivable", "revenuePacks", 3000, "inv-1"), je(4, i1, "receivable", "revenueUsage", 405, "inv-1"),
        je(5, i1, "receivable", "revenueMembership", 2500, "inv-2"), je(6, i1, "receivable", "revenueMembership", 9900, "inv-3")]})
    c.ok("getTrialBalance", "amy", i1, {}, tb({"receivable": (25705, 0), "revenueCourse": (0, 5000), "revenueMembership": (0, 17300),
                                              "revenuePacks": (0, 3000), "revenueUsage": (0, 405)}, 25705))
    c.ok("chargesByKind", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds(
        {"membership": 17300, "usage": 405, "course": 5000, "pack": 3000}, None, 25705), "totalCents": 25705})
    c.audit_log(i1)
    return c


def chain_2() -> Case:
    c = Case("C7-chain-2", "C7-chain", "Issue refund, Charges by kind, Postings, Get trial balance: the pack line of an invoice is refundable like any line; "
                                       "the refund shows in the `pack` row of the kinds report, posts debit `revenueRefunds` credit `creditLiability`, and "
                                       "leaves `revenuePacks` as it was")
    world(c)
    c.ok("addPackProduct", "amy", W, product("p30", "Thirty", 30, 3000, 60), {"product": product("p30", "Thirty", 30, 3000, 60)})
    s = t(3, "08:00")
    c.ok("sellPack", "sam", s, {"memberId": "ada", "productId": "p30"}, {
        "pack": pack("pak-1", "ada", "p30", 30, "2026-03-03", "2026-05-02"), "charge": chg("pack", "pak-1", 1, 3000, 0, 3000, s)}, new="pak-1")
    i1, d1 = t(1, "00:00", 4), t(15, "00:00", 4)
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", i1, d1, [chg("membership", "plus", 1, 4900, 0, 4900, i1), chg("course", "laser-101", 1, 5000, 0, 5000, W),
                                                     chg("pack", "pak-1", 1, 3000, 0, 3000, s)], 12900),
        invoice("inv-2", "bob", "2026-03", i1, d1, [chg("membership", "basic", 1, 2500, 0, 2500, i1)], 2500),
        invoice("inv-3", "eve", "2026-03", i1, d1, [chg("membership", "pro", 1, 9900, 0, 9900, i1)], 9900)]})
    p = t(2, "09:00", 4)
    c.ok("recordPayment", "sam", p, {"invoiceId": "inv-1", "amountCents": 12900}, None, new="pay-1")
    r = t(2, "10:00", 4)
    c.ok("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 3, "amountCents": 3000, "reason": "returned"}, {
        "refund": {"refundId": "rfd-1", "invoiceId": "inv-1", "lineNumber": 3, "amountCents": 3000, "reason": "returned", "at": r},
        "creditBalanceCents": 3000}, new="rfd-1")
    c.refuse("issueRefund", "amy", r, {"invoiceId": "inv-1", "lineNumber": 3, "amountCents": 1, "reason": "again"}, "refund_too_large")
    c.ok("chargesByKind", "sam", r, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds(
        {"membership": 17300, "course": 5000, "pack": 3000}, {"pack": 3000}, 25300), "totalCents": 25300})
    c.ok("listJournalEntries", "amy", r, {"month": "2026-04"}, {"month": "2026-04", "entries": [
        je(1, i1, "receivable", "revenueMembership", 4900, "inv-1"), je(2, i1, "receivable", "revenueCourse", 5000, "inv-1"),
        je(3, i1, "receivable", "revenuePacks", 3000, "inv-1"),
        je(4, i1, "receivable", "revenueMembership", 2500, "inv-2"), je(5, i1, "receivable", "revenueMembership", 9900, "inv-3"),
        je(6, p, "cash", "receivable", 12900, "pay-1"), je(7, r, "revenueRefunds", "creditLiability", 3000, "rfd-1")]})
    c.ok("getTrialBalance", "amy", r, {}, tb({"cash": (12900, 0), "creditLiability": (0, 3000), "receivable": (25300, 12900),
                                             "revenueCourse": (0, 5000), "revenueMembership": (0, 17300), "revenuePacks": (0, 3000),
                                             "revenueRefunds": (3000, 0)}, 41200))
    c.audit_log(r)
    return c


# ---- C7-ledger: the account --------------------------------------------------------------------------------------------------

def ledger_1() -> Case:
    c = Case("C7-ledger-1", "C7-ledger", "Ledger account, Get trial balance, Post adjustment: `revenuePacks` is an account on either side and sits between "
                                         "`revenueNoShows` and `revenueRefunds` in the 24 rows; `revenuePasses` is no account any more and is refused "
                                         "invalid_argument on either side, without taking an entry number")
    world(c)
    c.ok("getTrialBalance", "amy", W, {}, tb({}, 0))
    a = t(3, "09:00")
    c.refuse("postAdjustment", "amy", a, {"debitAccount": "revenuePasses", "creditAccount": "cash", "amountCents": 100, "memo": "old"}, "invalid_argument")
    c.refuse("postAdjustment", "amy", a, {"debitAccount": "cash", "creditAccount": "revenuePasses", "amountCents": 100, "memo": "old"}, "invalid_argument")
    c.ok("postAdjustment", "amy", a, {"debitAccount": "revenuePacks", "creditAccount": "cash", "amountCents": 700, "memo": "a"},
         {"entry": je(1, a, "revenuePacks", "cash", 700, "a")}, new="jrn-1")
    c.ok("postAdjustment", "amy", a, {"debitAccount": "cash", "creditAccount": "revenuePacks", "amountCents": 200, "memo": "b"},
         {"entry": je(2, a, "cash", "revenuePacks", 200, "b")}, new="jrn-2")
    c.refuse("postAdjustment", "amy", a, {"debitAccount": "revenuePacks", "creditAccount": "revenuePacks", "amountCents": 5, "memo": "c"}, "same_account")
    c.ok("postAdjustment", "amy", a, {"debitAccount": "revenueNoShows", "creditAccount": "revenuePacks", "amountCents": 50, "memo": "d"},
         {"entry": je(3, a, "revenueNoShows", "revenuePacks", 50, "d")}, new="jrn-3")
    c.ok("postAdjustment", "amy", a, {"debitAccount": "revenuePacks", "creditAccount": "revenueRefunds", "amountCents": 30, "memo": "e"},
         {"entry": je(4, a, "revenuePacks", "revenueRefunds", 30, "e")}, new="jrn-4")
    c.ok("getTrialBalance", "amy", a, {}, tb({"cash": (200, 700), "revenueNoShows": (50, 0), "revenuePacks": (730, 250),
                                             "revenueRefunds": (0, 30)}, 980))
    c.ok("listJournalEntries", "amy", a, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        je(1, a, "revenuePacks", "cash", 700, "a"), je(2, a, "cash", "revenuePacks", 200, "b"),
        je(3, a, "revenueNoShows", "revenuePacks", 50, "d"), je(4, a, "revenuePacks", "revenueRefunds", 30, "e")]})
    c.audit_log(a)
    return c


# ---- C7-keep: what is not renamed --------------------------------------------------------------------------------------------

def keep_1() -> Case:
    c = Case("C7-keep-1", "C7-keep", "Guest pass, Issue guest pass, List guest passes, Charge kind row, Postings: guest passes keep `guestPassId`, `guestPass`, "
                                     "`guestPasses`, the identifiers `gst-N`, the charge kind `guest` and the account `revenueGuests`, beside a pack with "
                                     "`packId`, `pak-N`, kind `pack` and `revenuePacks`; the two counters are separate")
    world(c)
    c.ok("setGuestAllowance", "amy", W, {"tier": "plus", "perMonth": 1, "feeCents": 500}, {"allowance": {"tier": "plus", "perMonth": 1, "feeCents": 500}})
    c.ok("addPackProduct", "amy", W, dict(P20), {"product": P20})
    c.ok("sellPack", "ada", W, {"memberId": "ada", "productId": "p20"}, {
        "pack": pack("pak-1", "ada", "p20", 20, "2026-03-02", "2026-04-01"), "charge": chg("pack", "pak-1", 1, 1000, 0, 1000, W)}, new="pak-1")
    c.ok("issueGuestPass", "ada", W, {"memberId": "ada", "guestName": "Gus", "date": "2026-03-10"},
         {"guestPass": gpass("gst-1", "ada", "Gus", "2026-03-10", False)}, new="gst-1")
    c.ok("issueGuestPass", "ada", W, {"memberId": "ada", "guestName": "Gwen", "date": "2026-03-11"},
         {"guestPass": gpass("gst-2", "ada", "Gwen", "2026-03-11", True)}, new="gst-2")
    c.ok("sellPack", "ada", W, {"memberId": "ada", "productId": "p20"}, {
        "pack": pack("pak-2", "ada", "p20", 20, "2026-03-02", "2026-04-01"), "charge": chg("pack", "pak-2", 1, 1000, 0, 1000, W)}, new="pak-2")
    c.ok("listGuestPasses", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "guestPasses": [
        gpass("gst-1", "ada", "Gus", "2026-03-10", False), gpass("gst-2", "ada", "Gwen", "2026-03-11", True)]})
    c.ok("listMemberPacks", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "packs": [
        pack("pak-1", "ada", "p20", 20, "2026-03-02", "2026-04-01"), pack("pak-2", "ada", "p20", 20, "2026-03-02", "2026-04-01")]})
    c.ok("revokeGuestPass", "ada", W, {"guestPassId": "gst-2"}, {"guestPass": gpass("gst-2", "ada", "Gwen", "2026-03-11", True, "revoked")})
    c.ok("getAccount", "ada", W, {"memberId": "ada"}, account("ada", 7500))
    i1, d1 = t(1, "00:00", 4), t(15, "00:00", 4)
    c.ok("generateInvoices", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", i1, d1, [chg("membership", "plus", 1, 4900, 0, 4900, i1), chg("course", "laser-101", 1, 5000, 0, 5000, W),
                                                     chg("pack", "pak-1", 1, 1000, 0, 1000, W), chg("guest", "gst-2", 1, 500, 0, 500, W),
                                                     chg("pack", "pak-2", 1, 1000, 0, 1000, W)], 12400),
        invoice("inv-2", "bob", "2026-03", i1, d1, [chg("membership", "basic", 1, 2500, 0, 2500, i1)], 2500),
        invoice("inv-3", "eve", "2026-03", i1, d1, [chg("membership", "pro", 1, 9900, 0, 9900, i1)], 9900)]})
    c.ok("listJournalEntries", "amy", i1, {"month": "2026-04"}, {"month": "2026-04", "entries": [
        je(1, i1, "receivable", "revenueMembership", 4900, "inv-1"), je(2, i1, "receivable", "revenueCourse", 5000, "inv-1"),
        je(3, i1, "receivable", "revenuePacks", 1000, "inv-1"), je(4, i1, "receivable", "revenueGuests", 500, "inv-1"),
        je(5, i1, "receivable", "revenuePacks", 1000, "inv-1"),
        je(6, i1, "receivable", "revenueMembership", 2500, "inv-2"), je(7, i1, "receivable", "revenueMembership", 9900, "inv-3")]})
    c.ok("getTrialBalance", "amy", i1, {}, tb({"receivable": (24800, 0), "revenueCourse": (0, 5000), "revenueGuests": (0, 500),
                                              "revenueMembership": (0, 17300), "revenuePacks": (0, 2000)}, 24800))
    c.ok("chargesByKind", "sam", i1, {"month": "2026-03"}, {"month": "2026-03", "rows": kinds(
        {"membership": 17300, "course": 5000, "pack": 2000, "guest": 500}, None, 24800), "totalCents": 24800})
    c.audit_log(i1)
    return c


def keep_2() -> Case:
    c = Case("C7-keep-2", "C7-keep", "Admit guest: `pass_not_today` and `pass_of_another_member` are refusal codes of guest passes and stay, in their order "
                                     "before `sponsor_not_inside`; the field is `guestPassId`")
    world(c)
    c.ok("setGuestAllowance", "amy", W, {"tier": "plus", "perMonth": 5, "feeCents": 0}, {"allowance": {"tier": "plus", "perMonth": 5, "feeCents": 0}})
    c.ok("setGuestAllowance", "amy", W, {"tier": "basic", "perMonth": 5, "feeCents": 0}, {"allowance": {"tier": "basic", "perMonth": 5, "feeCents": 0}})
    c.ok("issueGuestPass", "ada", W, {"memberId": "ada", "guestName": "Gus", "date": "2026-03-03"},
         {"guestPass": gpass("gst-1", "ada", "Gus", "2026-03-03", False)}, new="gst-1")
    c.ok("issueGuestPass", "bob", W, {"memberId": "bob", "guestName": "Bea", "date": "2026-03-02"},
         {"guestPass": gpass("gst-2", "bob", "Bea", "2026-03-02", False)}, new="gst-2")
    c.ok("issueGuestPass", "ada", W, {"memberId": "ada", "guestName": "Gil", "date": "2026-03-02"},
         {"guestPass": gpass("gst-3", "ada", "Gil", "2026-03-02", False)}, new="gst-3")
    c.ok("issueBadge", "sam", W, {"memberId": "ada"}, {"badge": {"badgeId": "bdg-1", "memberId": "ada", "status": "active", "inside": False}}, new="bdg-1")
    c.refuse("admitGuest", "ada", W, {"guestPassId": "gst-1", "badgeId": "bdg-1"}, "pass_not_today")
    c.refuse("admitGuest", "ada", W, {"guestPassId": "gst-2", "badgeId": "bdg-1"}, "pass_of_another_member")
    c.refuse("admitGuest", "ada", W, {"guestPassId": "gst-3", "badgeId": "bdg-1"}, "sponsor_not_inside")
    c.ok("enterDoor", "ada", W, {"badgeId": "bdg-1"}, {"badge": {"badgeId": "bdg-1", "memberId": "ada", "status": "active", "inside": True},
                                                       "event": {"eventId": "dor-1", "kind": "enter", "badgeId": "bdg-1", "guestPassId": None, "at": W}})
    c.ok("admitGuest", "ada", W, {"guestPassId": "gst-3", "badgeId": "bdg-1"}, {
        "event": {"eventId": "dor-2", "kind": "guest", "badgeId": "bdg-1", "guestPassId": "gst-3", "at": W}})
    c.audit_log(W)
    return c


def keep_3() -> Case:
    c = Case("C7-keep-3", "C7-keep", "Define assessment, Record attempt, Complete course: `passMark`, the attempt member `passed` and the refusal "
                                     "`assessment_not_passed` are the words of an assessment and stay")
    world(c)
    c.ok("enrollInCourse", "sam", W, {"courseId": "laser-101", "memberId": "bob"})
    c.ok("defineAssessment", "sam", W, {"courseId": "laser-101", "passMark": 70, "maxAttempts": 2},
         {"assessment": {"courseId": "laser-101", "passMark": 70, "maxAttempts": 2}})
    c.ok("recordAttempt", "sam", W, {"courseId": "laser-101", "memberId": "bob", "score": 69}, {"attempt": {
        "courseId": "laser-101", "memberId": "bob", "number": 1, "score": 69, "passed": False, "at": W}})
    c.refuse("completeCourse", "sam", W, {"courseId": "laser-101", "memberId": "bob"}, "assessment_not_passed")
    c.ok("recordAttempt", "sam", W, {"courseId": "laser-101", "memberId": "bob", "score": 70}, {"attempt": {
        "courseId": "laser-101", "memberId": "bob", "number": 2, "score": 70, "passed": True, "at": W}})
    c.audit_log(W)
    return c


def keep_4() -> Case:
    c = Case("C7-keep-4", "C7-keep", "Inspection plan, Record inspection: `lastPassedOn` and the outcome `passed` are the words of an inspection and stay")
    world(c)
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 30}, {"plan": {
        "machineId": "L1", "intervalDays": 30, "lastPassedOn": "2026-03-02", "nextDueOn": "2026-04-01", "failed": False}})
    n = t(10, "10:00")
    c.ok("recordInspection", "sam", n, {"machineId": "L1", "outcome": "passed", "note": "ok"}, {"plan": {
        "machineId": "L1", "intervalDays": 30, "lastPassedOn": "2026-03-10", "nextDueOn": "2026-04-09", "failed": False}})
    c.ok("getInspectionStatus", "sam", n, {"machineId": "L1"}, {"plan": {
        "machineId": "L1", "intervalDays": 30, "lastPassedOn": "2026-03-10", "nextDueOn": "2026-04-09", "failed": False}, "due": False})
    c.audit_log(n)
    return c


def build_cases() -> list[dict[str, Any]]:
    cases = [names_1(), old_1(), id_1(), chain_1(), chain_2(), ledger_1(), keep_1(), keep_2(), keep_3(), keep_4()]
    return [c.build() for c in cases]
