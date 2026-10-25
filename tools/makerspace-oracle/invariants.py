"""Cross-area invariants checked over a whole transcript.

A scenario case asserts only statuses and refusal codes per step; the named
checks then relate the observed responses to each other. They test relations
the requirements fix across areas (no charge without an audit entry, a locked
machine keeps no reservation in the lockout, bulk tiers never touch usage or
membership amounts), independent of any single expected value.
"""

from __future__ import annotations

from typing import Any

import facts as f
from facts import C, W, since
from builder import Case, ts
import world as w
from world import STAFF as S, T0

MON, TUE, WED, THU = "2026-03-02", "2026-03-03", "2026-03-04", "2026-03-05"
T1 = "2026-03-02T09:00Z"

Transcript = list[tuple[dict[str, Any], Any]]


def _request(step: dict[str, Any]) -> dict[str, Any] | None:
    return step.get("request")


def _ok(response: Any) -> bool:
    return isinstance(response, dict) and response.get("status") == "ok"


def _invoices(transcript: Transcript) -> list[dict[str, Any]]:
    """Every invoice object seen, latest version last."""
    seen = []
    for step, response in transcript:
        if not _ok(response):
            continue
        result = response.get("result", {})
        for item in result.get("invoices", []) if isinstance(result.get("invoices"), list) else []:
            seen.append(item)
        if isinstance(result.get("invoice"), dict):
            seen.append(result["invoice"])
    return seen


def _reservations(transcript: Transcript):
    for step, response in transcript:
        if not _ok(response):
            continue
        result = response.get("result", {})
        if isinstance(result.get("reservation"), dict):
            yield step, result["reservation"]
        for key in ("reservations", "cancelledReservations"):
            for item in result.get(key, []) if isinstance(result.get(key), list) else []:
                yield step, item


def _final_log(transcript: Transcript) -> list[dict[str, Any]]:
    step, response = transcript[-1]
    request = _request(step) or {}
    if request.get("op") != "listAuditEntries" or not _ok(response):
        raise ValueError("the scenario must end with an accepted listAuditEntries")
    return response["result"]["entries"]


def check_audit_order(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """w0 4.2 / c1: the log is exactly the accepted mutations (and from C1 the forbidden refusals), in order."""
    step_name = case["step"]
    ops = f.ops(step_name)
    roles = {"admin": "admin"}
    expected = []
    for step, response in transcript[:-1]:
        request = _request(step)
        if request is None or not isinstance(response, dict):
            continue
        args = request["args"]
        if _ok(response):
            if request["op"] == "registerUser":
                roles[args["userId"]] = args["role"]
            if request["op"] == "registerMember":
                roles[args["memberId"]] = "member"
            if ops[request["op"]]["kind"] == "mutation":
                expected.append((request["op"], args["actor"], args["at"], "accepted"))
        elif response.get("code") == "forbidden" and since(step_name, "C1"):
            expected.append((request["op"], args["actor"], args["at"], "refused"))
    log = _final_log(transcript)
    errors = []
    if len(log) != len(expected):
        errors.append(f"log has {len(log)} entries, the transcript implies {len(expected)}")
    for index, (entry, (op, actor, at, outcome)) in enumerate(zip(log, expected), 1):
        if entry.get("seq") != index or entry.get("op") != op or entry.get("actor") != actor or entry.get("at") != at:
            errors.append(f"entry {index} {entry!r} does not match {op} by {actor} at {at}")
        if since(step_name, "C1") and (entry.get("outcome") != outcome or entry.get("actorRole") != roles.get(actor)):
            errors.append(f"entry {index}: outcome/actorRole {entry.get('outcome')}/{entry.get('actorRole')} expected {outcome}/{roles.get(actor)}")
        if outcome == "refused" and entry.get("target") is not None:
            errors.append(f"entry {index}: a refusal entry has target null")
    return errors


def check_charge_audit(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """No charge without the audit entry of the request that created it (w0 4.2, 7.1, 9.2, 10, 11.1, 14)."""
    log = _final_log(transcript)
    keys = {(e.get("op"), e.get("target"), e.get("at")) for e in log}
    errors = []
    for invoice in _invoices(transcript):
        for line in invoice.get("lines", []):
            kind, ref, at = line.get("kind"), line.get("ref"), line.get("at")
            want = {"usage": ("checkOut", ref), "material": ("purchaseMaterial", ref),
                    "course": ("enrollInCourse", f"{ref}/{invoice.get('memberId')}"),
                    "membership": ("generateInvoices", invoice.get("month"))}.get(kind)
            if want is None or (want[0], want[1], at) not in keys:
                errors.append(f"{invoice.get('invoiceId')}: {kind} line {ref} at {at} has no matching audit entry")
    return sorted(set(errors))


def check_money(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """w0 4.3 and 11.1: line formula, totals, balances, status, non-negative integers."""
    errors = []
    for invoice in _invoices(transcript):
        lines = invoice.get("lines", [])
        for line in lines:
            if line.get("amountCents") != f.line_amount(line["quantity"], line["unitPriceCents"], line["discountPercent"]):
                errors.append(f"{invoice['invoiceId']}: line {line.get('ref')} amount {line.get('amountCents')} breaks the formula")
        if not lines or lines[0].get("kind") != "membership" or lines[0].get("unitPriceCents") != f.FEES.get(lines[0].get("ref")):
            errors.append(f"{invoice['invoiceId']}: the first line is not the membership line of the tier fee")
        total = sum(line.get("amountCents", 0) for line in lines)
        if invoice.get("totalCents") != total:
            errors.append(f"{invoice['invoiceId']}: totalCents {invoice.get('totalCents')} is not the line sum {total}")
        balance = invoice.get("totalCents", 0) - invoice.get("creditAppliedCents", 0) - invoice.get("paidCents", 0)
        if invoice.get("balanceCents") != balance or (invoice.get("status") == "paid") != (balance == 0):
            errors.append(f"{invoice['invoiceId']}: balance or status inconsistent")
        for key in ("totalCents", "creditAppliedCents", "paidCents", "balanceCents"):
            if type(invoice.get(key)) is not int or invoice[key] < 0:
                errors.append(f"{invoice['invoiceId']}: {key} is not a non-negative integer")
    for step, response in transcript:
        if _ok(response) and isinstance(response["result"].get("account"), dict):
            for key, value in response["result"]["account"].items():
                if key.endswith("Cents") and (type(value) is not int or value < 0):
                    errors.append(f"account {key} {value!r} is not a non-negative integer")
    return errors


def check_discounts(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """w0 6.2 and c3: usage by tier at check-out, materials by the bulk table only, courses and membership 0."""
    step_name = case["step"]
    tiers: dict[str, list[tuple[str, str]]] = {}
    errors = []
    for step, response in transcript:
        request = _request(step)
        if request and _ok(response) and request["op"] in ("registerMember", "changeTier"):
            tiers.setdefault(request["args"]["memberId"], []).append((request["args"]["at"], request["args"]["tier"]))
        if request and _ok(response) and request["op"] == "purchaseMaterial":
            purchase = response["result"]["purchase"]
            if purchase.get("discountPercent") != f.bulk_discount(step_name, purchase.get("quantity", 0)):
                errors.append(f"{purchase.get('purchaseId')}: discount {purchase.get('discountPercent')} is not the bulk tier of {purchase.get('quantity')}")

    def tier_at(member: str, at: str) -> str | None:
        current = None
        for when, tier in tiers.get(member, []):
            if when <= at:
                current = tier
        return current

    for invoice in _invoices(transcript):
        for line in invoice.get("lines", []):
            kind, d = line.get("kind"), line.get("discountPercent")
            if kind == "usage":
                tier = tier_at(invoice["memberId"], line["at"])
                if tier is None or d != f.USAGE_DISCOUNT[tier]:
                    errors.append(f"{invoice['invoiceId']}: usage line {line['ref']} discount {d} is not the {tier} tier discount")
            elif kind == "material" and d != f.bulk_discount(step_name, line.get("quantity", 0)):
                errors.append(f"{invoice['invoiceId']}: material line {line['ref']} discount {d}")
            elif kind in ("course", "membership") and d != 0:
                errors.append(f"{invoice['invoiceId']}: {kind} line {line['ref']} discount {d} is not 0")
    return sorted(set(errors))


def _overlap(a: dict[str, Any], b: dict[str, Any]) -> bool:
    return a["start"] < b["end"] and b["start"] < a["end"]


def check_no_overlap(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """w0 9.1 rules 5-7, c2: active reservations never overlap each other or maintenance; reserved ones never a lockout."""
    errors = []
    for step, response in transcript:
        if not _ok(response):
            continue
        result = response["result"]
        if not isinstance(result.get("reservations"), list):
            continue
        active = [r for r in result["reservations"] if r.get("status") in ("reserved", "checkedIn")]
        for i, a in enumerate(active):
            for b in active[i + 1:]:
                if _overlap(a, b):
                    errors.append(f"{a['reservationId']} and {b['reservationId']} are active and overlap")
            for window in result.get("maintenance", []) if isinstance(result.get("maintenance"), list) else []:
                if _overlap(a, window):
                    errors.append(f"{a['reservationId']} is active and overlaps {window.get('windowId')}")
            for lock in result.get("lockouts", []) if isinstance(result.get("lockouts"), list) else []:
                if a.get("status") == "reserved" and _overlap(a, lock):
                    errors.append(f"{a['reservationId']} is reserved inside lockout {lock.get('lockoutId')}")
    return sorted(set(errors))


def check_lockouts(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """c2: a lockout cancels exactly overlapping reserved reservations of its machine, and none is booked into it later."""
    errors = []
    locks: list[dict[str, Any]] = []
    for step, response in transcript:
        request = _request(step)
        if not request or not _ok(response):
            continue
        result = response["result"]
        if request["op"] == "lockMachine":
            lock = result["lockout"]
            locks.append(lock)
            ids = [r.get("reservationId") for r in result.get("cancelledReservations", [])]
            if ids != lock.get("cancelledReservationIds"):
                errors.append(f"{lock['lockoutId']}: cancelledReservations differ from cancelledReservationIds")
            for r in result.get("cancelledReservations", []):
                if r.get("status") != "cancelled" or r.get("machineId") != lock.get("machineId") or not _overlap(r, lock):
                    errors.append(f"{lock['lockoutId']}: {r.get('reservationId')} is not a cancelled overlapping reservation of the machine")
        if request["op"] == "createReservation":
            new = result["reservation"]
            for lock in locks:
                if lock.get("machineId") == new.get("machineId") and _overlap(new, lock):
                    errors.append(f"{new['reservationId']} was booked inside {lock['lockoutId']}")
    return errors


def _minute(t: str) -> int:
    return int(t[11:13]) * 60 + int(t[14:16])


def _hm(hm: str) -> int:
    return int(hm[:2]) * 60 + int(hm[3:])


def check_slots(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """w0 4.4 and c8: every reservation follows the granularity, the length limits and the opening hours of its start date."""
    import datetime
    step_name = case["step"]
    grain = f.granularity(step_name)
    hours = f.opening_hours(step_name)
    errors = []
    for _, r in _reservations(transcript):
        start, end = r.get("start", ""), r.get("end", "")
        if start[:10] != end[:10]:
            errors.append(f"{r.get('reservationId')} spans days")
            continue
        s, e = _minute(start), _minute(end)
        day = hours[datetime.date.fromisoformat(start[:10]).weekday()]
        if s % grain or e % grain or not grain <= e - s <= 240:
            errors.append(f"{r.get('reservationId')} breaks granularity or length")
        if day is None or s < _hm(day[0]) or e > _hm(day[1]):
            errors.append(f"{r.get('reservationId')} lies outside opening hours")
    return sorted(set(errors))


def check_counters(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """w0 2.6: generated identifiers of each kind are consecutive in creation order."""
    fields = {"createReservation": ("reservation", "reservationId", "res"), "scheduleMaintenance": ("window", "windowId", "mnt"),
              "purchaseMaterial": ("purchase", "purchaseId", "pur"), "recordPayment": ("payment", "paymentId", "pay"),
              "reportIncident": ("incident", "incidentId", "inc"), "lockMachine": ("lockout", "lockoutId", "lock")}
    seen: dict[str, list[str]] = {}
    for step, response in transcript:
        request = _request(step)
        if not request or not _ok(response):
            continue
        if request["op"] in fields:
            record, key, prefix = fields[request["op"]]
            seen.setdefault(prefix, []).append(response["result"][record][key])
        if request["op"] == "generateInvoices":
            seen.setdefault("inv", []).extend(i["invoiceId"] for i in response["result"]["invoices"])
    errors = []
    for prefix, ids in seen.items():
        if ids != [f"{prefix}-{n}" for n in range(1, len(ids) + 1)]:
            errors.append(f"{prefix} identifiers are {ids}")
    return errors


def check_accounts(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """w0 11.2: getAccount right after listInvoices of the same member at the same time agrees with the invoices."""
    errors = []
    for (s1, r1), (s2, r2) in zip(transcript, transcript[1:]):
        q1, q2 = _request(s1), _request(s2)
        if not (q1 and q2 and _ok(r1) and _ok(r2) and q1["op"] == "listInvoices" and q2["op"] == "getAccount"
                and q1["args"]["memberId"] == q2["args"]["memberId"] and q1["args"]["at"] == q2["args"]["at"]):
            continue
        invoices = r1["result"]["invoices"]
        account = r2["result"]["account"]
        open_ = sum(i["balanceCents"] for i in invoices if i["status"] == "open")
        overdue = sum(i["balanceCents"] for i in invoices if i["overdue"])
        if account.get("openBalanceCents") != open_ or account.get("overdueBalanceCents") != overdue:
            errors.append(f"{q1['args']['memberId']} at {q1['args']['at']}: account {account} disagrees with its invoices")
        months = [i["month"] for i in invoices]
        if months != sorted(months):
            errors.append(f"{q1['args']['memberId']}: invoices are not in month order")
    return errors


def check_billing_once(case: dict[str, Any], transcript: Transcript) -> list[str]:
    """w0 11.1: a charge is invoiced once, on the invoice of the month of its `at`."""
    placed: dict[tuple, str] = {}
    errors = []
    for invoice in _invoices(transcript):
        for line in invoice.get("lines", [])[1:]:
            key = (invoice.get("memberId"), line.get("kind"), line.get("ref"), line.get("at"), line.get("amountCents"))
            if placed.setdefault(key, invoice["invoiceId"]) != invoice["invoiceId"]:
                errors.append(f"{key} appears on {placed[key]} and {invoice['invoiceId']}")
            if line.get("at", "")[:7] != invoice.get("month"):
                errors.append(f"{invoice['invoiceId']}: line {line.get('ref')} at {line.get('at')} is outside {invoice.get('month')}")
    return sorted(set(errors))


CHECKS = {"INV-AUDIT-ORDER": check_audit_order, "INV-CHARGE-AUDIT": check_charge_audit, "INV-MONEY": check_money,
          "INV-DISCOUNTS": check_discounts, "INV-NO-OVERLAP": check_no_overlap, "INV-LOCKOUTS": check_lockouts,
          "INV-SLOTS": check_slots, "INV-COUNTERS": check_counters, "INV-ACCOUNTS": check_accounts,
          "INV-BILLING-ONCE": check_billing_once}

CITES = [W("4.2", "Every request to a mutation that is answered `ok` appends exactly one audit entry"),
         W("4.3", "Totals are sums of rounded line amounts and are never rounded again."),
         W("9.1", "Only active reservations block a machine or a member."),
         W("11.1", "These charges are no longer unbilled."),
         C(2, "What changes", "A lockout cancels the reservations that have not started being used in that window"),
         C(3, "What must not change", "The bulk discount is a materials rule only.")]


def scenario(step: str) -> list[dict[str, Any]]:
    """A month in the life of the workshop, every area interleaved; statuses only, then one check per case."""
    c = Case(step, "INV-scenario", ["INV"], CITES, strict=False)
    w.world(c)
    if since(step, "C4"):
        c.ok("registerUser", "admin", T0, {"userId": "stu", "name": "Stu", "role": "steward"})
    c.ok("addCourse", S, T1, {"courseId": "wood-101", "familyId": "wood", "title": "Lathe basics", "feeCents": 1250, "capacity": 4})
    c.ok("changeTier", S, T1, {"memberId": "bob", "tier": "plus"})
    r1 = c.ok("createReservation", "ada", T1, {"memberId": "ada", "machineId": "L1", "start": ts(TUE, "10:00"), "end": ts(TUE, "11:00")})
    r2 = w.reserve(c, "bob", "L2", ts(TUE, "10:00"), ts(TUE, "10:30"), T1)
    w.reserve(c, "cy", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), T1)
    c.ok("purchaseMaterial", "ada", T1, {"memberId": "ada", "materialId": "acrylic", "quantity": 12})
    c.ok("purchaseMaterial", S, T1, {"memberId": "bob", "materialId": "plywood", "quantity": 30})
    c.ok("purchaseMaterial", S, T1, {"memberId": "cy", "materialId": "acrylic", "quantity": 3})
    c.ok("scheduleMaintenance", S, T1, {"machineId": "W1", "start": ts(WED, "10:00"), "end": ts(WED, "12:00"), "reason": "Bearing"})
    c.refuse("listAuditEntries", "ada", T1, {}, "forbidden")
    c.refuse("cancelReservation", "bob", T1, {"reservationId": r1}, "forbidden")
    c.refuse("createReservation", S, T1, {"memberId": "cy", "machineId": "L1", "start": ts(TUE, "10:30"), "end": ts(TUE, "11:00")}, "machine_booked")
    c.ok("listMemberReservations", S, T1, {"memberId": "ada"})
    c.ok("getMachineSchedule", S, T1, {"machineId": "L1", "date": TUE})
    c.ok("checkIn", "ada", ts(TUE, "10:00"), {"reservationId": r1})
    c.ok("checkIn", S, ts(TUE, "10:00"), {"reservationId": r2})
    c.ok("changeTier", S, ts(TUE, "10:20"), {"memberId": "ada", "tier": "pro"})
    c.ok("checkOut", S, ts(TUE, "10:31"), {"reservationId": r2})
    c.ok("checkOut", "ada", ts(TUE, "10:45"), {"reservationId": r1})
    noon = ts(TUE, "12:00")
    w.reserve(c, "ada", "L2", ts(WED, "10:00"), ts(WED, "10:30"), noon)
    r5 = w.reserve(c, "bob", "L2", ts(WED, "11:00"), ts(WED, "11:30"), noon)
    if since(step, "C2"):
        inc = c.ok("reportIncident", S, noon, {"machineId": "L2", "severity": "high", "description": "Sparks"})
        c.ok("lockMachine", S, noon, {"incidentId": inc, "start": ts(WED, "09:00"), "end": ts(WED, "10:45")})
        c.refuse("createReservation", S, noon, {"memberId": "cy", "machineId": "L2", "start": ts(WED, "10:00"), "end": ts(WED, "10:30")}, "machine_locked")
    if since(step, "C4"):
        inc2 = c.ok("reportIncident", "stu", noon, {"machineId": "W1", "severity": "low", "description": "Squeak"})
        c.ok("lockMachine", "stu", noon, {"incidentId": inc2, "start": ts(THU, "10:00"), "end": ts(THU, "11:00")})
        c.refuse("getAccount", "stu", noon, {"memberId": "ada"}, "forbidden")
    c.ok("getMachineSchedule", S, noon, {"machineId": "L2", "date": WED})
    c.ok("listMemberReservations", S, noon, {"memberId": "ada"})
    c.ok("enrollInCourse", S, noon, {"courseId": "wood-101", "memberId": "cy"})
    c.ok("completeCourse", S, noon, {"courseId": "wood-101", "memberId": "cy"})
    c.ok("checkIn", S, ts(WED, "11:00"), {"reservationId": r5})
    c.ok("checkOut", S, ts(WED, "11:20"), {"reservationId": r5})
    c.ok("issueCredit", "admin", ts(WED, "12:00"), {"memberId": "cy", "amountCents": 700, "reason": "Sorry"})
    apr1, apr2, apr20 = "2026-04-01T09:00Z", "2026-04-02T09:00Z", "2026-04-20T09:00Z"
    c.ok("generateInvoices", S, apr1, {"month": "2026-03"})
    for member in ("ada", "bob", "cy"):
        c.ok("listInvoices", S, apr1, {"memberId": member})
        c.ok("getAccount", S, apr1, {"memberId": member})
    c.ok("recordPayment", "ada", apr2, {"invoiceId": "inv-1", "amountCents": 100000})
    c.ok("recordPayment", S, apr2, {"invoiceId": "inv-2", "amountCents": 1000})
    c.ok("revenueReport", S, apr2, {"month": "2026-03"})
    c.ok("utilizationReport", S, apr2, {"from": MON, "to": "2026-03-09"})
    for member in ("ada", "bob", "cy"):
        c.ok("listInvoices", S, apr20, {"memberId": member})
        c.ok("getAccount", S, apr20, {"memberId": member})
    may1 = "2026-05-01T09:00Z"
    c.ok("purchaseMaterial", S, "2026-04-21T09:00Z", {"memberId": "ada", "materialId": "plywood", "quantity": 25})
    w.reserve(c, "bob", "L1", "2026-05-04T10:00Z", "2026-05-04T10:30Z", may1)
    c.ok("generateInvoices", S, may1, {"month": "2026-04"})
    for member in ("ada", "bob", "cy"):
        c.ok("listInvoices", S, may1, {"memberId": member})
        c.ok("getAccount", S, may1, {"memberId": member})
    c.ok("listAuditEntries", "admin", may1)
    base = c.build()
    # The statuses are judged once (INV-SCENARIO); each check fails only its own family.
    checks = [{**base, "id": f"{name}-scenario", "families": [name], "invariants": [name], "step": step, "checks_only": True}
              for name in CHECKS]
    return [{**base, "id": "INV-SCENARIO-statuses", "families": ["INV-SCENARIO"]}] + checks
