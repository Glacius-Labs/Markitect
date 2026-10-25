"""Probe families: every easy-to-miss rule, grouped so one mutant can target one family.

Probe cases are loose: setup requests assert only `status: ok`, so a fault in
another family's rule does not break them. The probed steps compare exactly
(refusal codes always; results where `check=True`). FAMILIES lists each
family's neutral references; the probe-family list sent to session I is
generated from the same table.
"""

from __future__ import annotations

from typing import Any, Callable

import facts as f
from facts import C, W, since
from builder import Case, ts
from harness import ANY
import world as w
from world import STAFF as S, T0

MON, TUE, WED, THU, FRI, SAT, SUN = (f"2026-03-0{d}" for d in range(2, 9))
T1 = "2026-03-02T09:00Z"
APR1 = "2026-04-01T09:00Z"


def has(**members: Any) -> dict[str, Any]:
    return {"$has": members}


def probe(step: str, family: str, suffix: str, capture: bool = True) -> Case:
    """A loose case citing the family's references in force at `step` (facts.scope_refs).

    Generated identifiers are captured, not predicted, except where numbering is the subject.
    """
    return Case(step, f"{family}-{suffix}", [family], FAMILY_REFS[family], strict=False, capture=capture)


def lines(kind: str, items: list[Any]) -> dict[str, Any]:
    """Only the invoice lines of one kind, in their order: a family compares its own lines."""
    return {"$filter": {"kind": kind}, "items": items}


def stamp(at: str, minutes: int) -> str:
    """at + minutes, for times inside one day (hh:mm arithmetic only)."""
    day, hm = at[:10], at[11:16]
    total = int(hm[:2]) * 60 + int(hm[3:]) + minutes
    return f"{day}T{total // 60:02d}:{total % 60:02d}Z"


def check_in(c: Case, rid: str, at: str, actor: str = S, **kw) -> None:
    c.ok("checkIn", actor, at, {"reservationId": rid}, **kw)


def check_out(c: Case, rid: str, at: str, actor: str = S, **kw) -> None:
    c.ok("checkOut", actor, at, {"reservationId": rid}, **kw)


def generate(c: Case, month: str, at: str, actor: str = S, **kw) -> dict[str, str]:
    return c.ok("generateInvoices", actor, at, {"month": month}, **kw)


def membership_line(tier: str, at: str) -> dict[str, Any]:
    return f.charge("membership", tier, 1, f.FEES[tier], 0, at)


def money(member: str, month: str, lines: list[dict[str, Any]], credit: int = 0, paid: int = 0) -> dict[str, Any]:
    """The money members of an invoice, ignoring its time members."""
    full = f.invoice("?", member, month, "?", "?", lines, credit, paid)
    return has(**{k: full[k] for k in ("memberId", "month", "lines", "totalCents", "creditAppliedCents",
                                       "paidCents", "balanceCents", "status")})


# --- Valid arguments of every operation in the authorization world ---------------------

def authz_world(c: Case) -> dict[str, Any]:
    """world() plus a reservation, a window, an incident and the March invoices; returns their captured identifiers."""
    w.world(c)
    if since(c.step, "C4"):
        c.ok("registerUser", "admin", T0, {"userId": "stu", "name": "Stu", "role": "steward"})
    res = w.reserve(c, "bob", "L1", "2026-04-02T10:00Z", "2026-04-02T10:30Z", T1)
    win = c.ok("scheduleMaintenance", S, T1, {"machineId": "W1", "start": "2026-04-03T10:00Z", "end": "2026-04-03T11:00Z", "reason": "Service"})
    inc = c.ok("reportIncident", S, T1, {"machineId": "L2", "severity": "medium", "description": "Noisy fan"})
    inv = generate(c, "2026-03", APR1)
    return {"res": res, "mnt": win, "inc": inc, "inv": inv["bob"]}


VALID: dict[str, dict[str, Any]] = {
    "registerUser": {"userId": "new-user", "name": "New User", "role": "staff"},
    "registerMember": {"memberId": "new-member", "name": "New Member", "tier": "basic"},
    "getMember": {"memberId": "bob"},
    "changeTier": {"memberId": "bob", "tier": "pro"},
    "setMemberStanding": {"memberId": "bob", "standing": "suspended"},
    "addCourse": {"courseId": "new-course", "familyId": "laser", "title": "New course", "feeCents": 100, "capacity": 5},
    "enrollInCourse": {"courseId": "laser-101", "memberId": "bob"},
    "completeCourse": {"courseId": "laser-101", "memberId": "bob"},
    "revokeCertification": {"memberId": "bob", "familyId": "laser"},
    "listCertifications": {"memberId": "bob"},
    "addFamily": {"familyId": "new-family", "name": "New family"},
    "addMachine": {"machineId": "new-machine", "name": "New machine", "familyId": "laser", "rateCentsPerMinute": 10},
    "setMachineRate": {"machineId": "L1", "rateCentsPerMinute": 50},
    "listMachines": {},
    "scheduleMaintenance": {"machineId": "W1", "start": "2026-04-06T10:00Z", "end": "2026-04-06T11:00Z", "reason": "Service"},
    "cancelMaintenance": {"windowId": "<mnt>"},
    "createReservation": {"memberId": "bob", "machineId": "L2", "start": "2026-04-02T12:00Z", "end": "2026-04-02T12:30Z"},
    "cancelReservation": {"reservationId": "<res>"},
    "checkIn": {"reservationId": "<res>"},
    "checkOut": {"reservationId": "<res>"},
    "listMemberReservations": {"memberId": "bob"},
    "getMachineSchedule": {"machineId": "L1", "date": "2026-04-02"},
    "addMaterial": {"materialId": "new-material", "name": "New material", "unitPriceCents": 10},
    "restockMaterial": {"materialId": "acrylic", "quantity": 5},
    "purchaseMaterial": {"memberId": "bob", "materialId": "acrylic", "quantity": 1},
    "listMaterials": {},
    "generateInvoices": {"month": "2026-02"},
    "listInvoices": {"memberId": "bob"},
    "recordPayment": {"invoiceId": "<inv>", "amountCents": 100},
    "issueCredit": {"memberId": "bob", "amountCents": 100, "reason": "Goodwill"},
    "getAccount": {"memberId": "bob"},
    "reportIncident": {"machineId": "L1", "severity": "low", "description": "Smoke"},
    "resolveIncident": {"incidentId": "<inc>", "resolution": "Fixed"},
    "listIncidents": {},
    "utilizationReport": {"from": "2026-03-02", "to": "2026-03-09"},
    "revenueReport": {"month": "2026-03"},
    "listAuditEntries": {},
    "lockMachine": {"incidentId": "<inc>", "start": "2026-04-07T10:00Z", "end": "2026-04-07T11:00Z"},
}


def valid(op: str, refs: dict[str, Any]) -> dict[str, Any]:
    """Valid arguments of `op` in the authorization world, with its captured identifiers."""
    return {k: refs[v[1:-1]] if isinstance(v, str) and v.startswith("<") else v for k, v in VALID[op].items()}


ACTOR_OF_ROLE = {"member": "ada", "staff": S, "admin": "admin", "steward": "stu"}


# --- Protocol ------------------------------------------------------------------------------

def p_envelope(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-ENVELOPE", "lines")
    w.world(c)
    a = '{"actor":"admin","at":"%s"}' % T1
    c.ok("listMachines", "admin", T1)
    for line in ("not json", "", "   ", "[1,2]", '"listMachines"', "42", "null", "{"):
        c.invalid(line, None)
    for line in ('{"op":"listMachines","args":%s}' % a, '{"id":7,"op":"listMachines","args":%s}' % a,
                 '{"id":"","op":"listMachines","args":%s}' % a, '{"id":null,"op":"listMachines","args":%s}' % a,
                 '{"id":["x"],"op":"listMachines","args":%s}' % a):
        c.invalid(line, None)
    for rid, line in (("e1", '{"id":"e1","args":%s}' % a), ("e2", '{"id":"e2","op":17,"args":%s}' % a),
                      ("e3", '{"id":"e3","op":"ListMachines","args":%s}' % a),
                      ("e4", '{"id":"e4","op":"noSuchOperation","args":%s}' % a),
                      ("e5", '{"id":"e5","op":"listMachines"}'), ("e6", '{"id":"e6","op":"listMachines","args":[]}'),
                      ("e7", '{"id":"e7","op":"listMachines","args":null}'), ("e8", '{"id":"e8","op":"listMachines","args":"x"}'),
                      ("e9", '{"id":"e9","op":"listMachines","args":%s,"extra":1}' % a),
                      ("e10", '{"id":"e10","op":"listMachines","args":{"actor":"admin","at":"%s","memberId":"ada"}}' % T1),
                      ("e11", '{"id":"e11","op":"getMember","args":{"actor":"admin","at":"%s","memberid":"ada"}}' % T1),
                      ("e12", '{"id":"e12","op":"registerMember","args":{"actor":"admin","at":"%s","memberId":"zed","name":"Zed","tier":"basic","Tier":"basic"}}' % T1),
                      ("e13", '{"id":"e13","op":"createReservation","args":{"actor":"sam","at":"%s","memberId":"ada","machineId":"L1","start":"2026-03-03T10:00Z","end":"2026-03-03T11:00Z","note":"x"}}' % T1),
                      ("e14", '{"id":"e14","op":"listMachines","args":{"actor":"admin","at":"2026-03-02T09:00Z","bogus":1}}')):
        c.invalid(line, rid)
    # Nothing of the invalid requests happened: no member, no time (the counter is P-IDENTIFIERS').
    c.refuse("getMember", "admin", T1, {"memberId": "zed"}, "not_found")
    c.ok("createReservation", S, "2026-03-02T09:30Z", {"memberId": "ada", "machineId": "L1", "start": ts(TUE, "10:00"),
                                                         "end": ts(TUE, "11:00")})
    # The same id twice is two requests, never a replay.
    later = "2026-03-02T09:40Z"
    c.ok("registerMember", S, later, {"memberId": "dup-a", "name": "Dup A", "tier": "basic"}, rid="dup")
    c.ok("registerMember", S, later, {"memberId": "dup-b", "name": "Dup B", "tier": "basic"},
         {"member": has(memberId="dup-b")}, check=True, rid="dup")
    c.refuse("registerMember", S, later, {"memberId": "dup-a", "name": "Dup A", "tier": "basic"}, "already_exists", rid="dup")
    text = "x" * 60000
    c.ok("reportIncident", S, later, {"machineId": "L1", "severity": "low", "description": text},
         lambda iid: {"incident": has(incidentId=iid, description=text)}, check=True)
    return [c.build()]


def _base_value(kind: tuple) -> Any:
    return {"identifier": "x1", "text": "Text", "timestamp": "2026-03-03T10:00Z", "date": "2026-03-03",
            "month": "2026-03"}.get(kind[0], kind[1] if kind[0] == "integer" else kind[1][0] if kind[0] == "enum" else None)


def _wrong_types(kind: tuple) -> list[Any]:
    return ["5", 1.5, True] if kind[0] == "integer" else [17, True, {"x": 1}]


def p_arguments(step: str) -> list[dict[str, Any]]:
    ops = f.ops(step)
    c = probe(step, "P-ARGUMENTS", "required")
    at = T1
    for op, info in ops.items():
        base = {name: _base_value(kind) for name, kind in info["args"]}
        base["actor"], base["at"] = "admin", at
        for name, kind in info["args"]:
            variants = [None, "missing"] + _wrong_types(kind)
            for value in variants:
                payload = dict(base)
                if value == "missing":
                    del payload[name]
                else:
                    payload[name] = value
                rid = c._rid()
                c.steps.append({"request": {"id": rid, "op": op, "args": payload},
                                "expect": c._refused(rid, "invalid_argument")})
    c.ok("listMachines", "admin", at, {}, {"machines": []}, check=True)

    d = probe(step, "P-ARGUMENTS", "formats")
    d.ok("addFamily", "admin", T1, {"familyId": "fam", "name": "Family"})
    for mid in ("", "x" * 65, "-ada", ".ada", "_ada", "ada!", "a d", "adé", "ada\n", "ad/a"):
        d.refuse("registerMember", "admin", T1, {"memberId": mid, "name": "Name", "tier": "basic"}, "invalid_argument")
    for mid in ("x" * 64, "9lives", "a.b_c-D", "ada", "Ada"):
        d.ok("registerMember", "admin", T1, {"memberId": mid, "name": "Name", "tier": "basic"},
             {"member": has(memberId=mid)}, check=True)
    for name in ("", " ", "\t\n", " ", "　 "):
        d.refuse("registerMember", "admin", T1, {"memberId": "texty", "name": name, "tier": "basic"}, "invalid_argument")
    d.ok("registerMember", "admin", T1, {"memberId": "spaced", "name": " Spaced \t Name ", "tier": "basic"},
         {"member": has(name=" Spaced \t Name ")}, check=True)
    for rate in (0, 10001, -5, 10.0, 2.5):
        d.refuse("addMachine", "admin", T1, {"machineId": "m-x", "name": "M", "familyId": "fam", "rateCentsPerMinute": rate}, "invalid_argument")
    for raw in ("1e3", "-0", "1E1", "100.0"):
        d.refuse_raw("addMachine", "admin", T1, '"machineId":"m-x","name":"M","familyId":"fam","rateCentsPerMinute":%s' % raw, "invalid_argument")
    d.refuse_raw("addCourse", "admin", T1, '"courseId":"c-neg","familyId":"fam","title":"T","feeCents":-0,"capacity":5', "invalid_argument",
                 cite=W("2.5", "including `-0`, is out of range"))
    for mid, rate in (("m-1", 1), ("m-10000", 10000)):
        d.ok("addMachine", "admin", T1, {"machineId": mid, "name": "M", "familyId": "fam", "rateCentsPerMinute": rate},
             {"machine": has(rateCentsPerMinute=rate)}, check=True)
    for fee, cap, good in ((0, 1000, True), (1000000, 1, True), (1000001, 5, False), (5, 0, False), (5, 1001, False), (-1, 5, False)):
        args = {"courseId": f"course-{fee}-{cap}", "familyId": "fam", "title": "T", "feeCents": fee, "capacity": cap}
        if good:
            d.ok("addCourse", "admin", T1, args, {"course": has(feeCents=fee, capacity=cap)}, check=True)
        else:
            d.refuse("addCourse", "admin", T1, args, "invalid_argument")
    d.ok("addMaterial", "admin", T1, {"materialId": "bead", "name": "Bead", "unitPriceCents": 1000000},
         {"material": has(unitPriceCents=1000000)}, check=True)
    for price in (0, 1000001):
        d.refuse("addMaterial", "admin", T1, {"materialId": "bead2", "name": "Bead", "unitPriceCents": price}, "invalid_argument")
    for qty in (0, 100001):
        d.refuse("restockMaterial", "admin", T1, {"materialId": "bead", "quantity": qty}, "invalid_argument")
    d.ok("restockMaterial", "admin", T1, {"materialId": "bead", "quantity": 100000}, {"material": has(stock=100000)}, check=True)
    for qty in (0, 10001):
        d.refuse("purchaseMaterial", "admin", T1, {"memberId": "ada", "materialId": "bead", "quantity": qty}, "invalid_argument")
    for amount in (0, 100000001, 100000000000000000000):
        d.refuse("issueCredit", "admin", T1, {"memberId": "ada", "amountCents": amount, "reason": "R"}, "invalid_argument")
        d.refuse("recordPayment", "admin", T1, {"invoiceId": "inv-1", "amountCents": amount}, "invalid_argument")
    d.ok("issueCredit", "admin", T1, {"memberId": "ada", "amountCents": 100000000, "reason": "R"},
         {"memberId": "ada", "amountCents": 100000000, "creditBalanceCents": 100000000}, check=True)
    for tier in ("Plus", "gold", "", "basic "):
        d.refuse("changeTier", "admin", T1, {"memberId": "ada", "tier": tier}, "invalid_argument")
    d.refuse("setMemberStanding", "admin", T1, {"memberId": "ada", "standing": "Suspended"}, "invalid_argument")
    d.refuse("reportIncident", "admin", T1, {"machineId": "m-1", "severity": "critical", "description": "D"}, "invalid_argument")
    d.refuse("registerUser", "admin", T1, {"userId": "u-member", "name": "U", "role": "member"}, "invalid_argument")
    d.refuse("registerUser", "admin", T1, {"userId": "u-admin", "name": "U", "role": "Admin"}, "invalid_argument")
    if since(step, "C4"):
        d.ok("registerUser", "admin", T1, {"userId": "u-steward", "name": "U", "role": "steward"},
             {"user": f.user("u-steward", "U", "steward")}, check=True, cite=C(4, "What changes", "accepts `role` from the enumeration `staff`, `admin` or `steward`"))
    else:
        d.refuse("registerUser", "admin", T1, {"userId": "u-steward", "name": "U", "role": "steward"}, "invalid_argument")
    return [c.build(), d.build()]


def p_check_order(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-CHECK-ORDER", "checks")
    w.world(c)
    bobs = w.reserve(c, "bob", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), T0)
    c.ok("changeTier", S, T1, {"memberId": "bob", "tier": "basic"})
    c.refuse("getMember", "ghost", T1, {"memberId": 17}, "invalid_argument")
    c.refuse("listMachines", "ghost", T0, {}, "unknown_actor")
    c.refuse("listAuditEntries", "ghost", T1, {}, "unknown_actor")
    c.refuse("registerUser", "ada", T0, {"userId": "u1", "name": "U", "role": "staff"}, "time_regression")
    c.refuse("getMember", "admin", T0, {"memberId": "nobody"}, "time_regression")
    c.refuse("changeTier", "ada", T1, {"memberId": "nobody", "tier": "pro"}, "forbidden")
    c.refuse("scheduleMaintenance", "ada", T1, {"machineId": "L1", "start": ts(TUE, "12:00"), "end": ts(TUE, "11:00"), "reason": "R"}, "forbidden")
    c.refuse("cancelReservation", "ada", T1, {"reservationId": "res-99"}, "not_found")
    c.refuse("checkIn", "ada", T1, {"reservationId": bobs}, "forbidden")
    c.refuse("createReservation", S, T1, {"memberId": "nobody", "machineId": "L1", "start": ts(TUE, "10:15"), "end": ts(TUE, "11:00")}, "not_found")
    c.refuse("createReservation", S, T1, {"memberId": "ada", "machineId": "nope", "start": ts(TUE, "10:15"), "end": ts(TUE, "11:00")}, "not_found")
    for actor, op, args in ((S, "getMember", {"memberId": S}), (S, "getMember", {"memberId": "admin"}),
                            (S, "createReservation", {"memberId": S, "machineId": "L2", "start": ts(TUE, "12:00"), "end": ts(TUE, "12:30")}),
                            ("admin", "purchaseMaterial", {"memberId": "admin", "materialId": "acrylic", "quantity": 1}),
                            ("ada", "listInvoices", {"memberId": S}), (S, "changeTier", {"memberId": S, "tier": "pro"})):
        c.refuse(op, actor, T1, args, "not_found")
    return [c.build()]


def p_rule_order(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-RULE-ORDER", "operations")
    w.world(c)
    t = T1
    c.ok("addCourse", S, t, {"courseId": "wood-101", "familyId": "wood", "title": "Lathe basics", "feeCents": 0, "capacity": 1})
    c.ok("enrollInCourse", S, t, {"courseId": "wood-101", "memberId": "bob"})
    c.ok("setMemberStanding", S, t, {"memberId": "bob", "standing": "suspended"})
    c.refuse("enrollInCourse", S, t, {"courseId": "wood-101", "memberId": "bob"}, "member_suspended")
    c.ok("setMemberStanding", S, t, {"memberId": "bob", "standing": "good"})
    c.refuse("enrollInCourse", S, t, {"courseId": "wood-101", "memberId": "bob"}, "already_enrolled")
    c.refuse("enrollInCourse", S, t, {"courseId": "wood-101", "memberId": "cy"}, "course_full")
    ada = w.reserve(c, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), t)
    c.refuse("scheduleMaintenance", S, t, {"machineId": "L1", "start": "2026-03-02T08:00Z", "end": "2026-03-02T07:00Z", "reason": "R"}, "invalid_interval")
    c.refuse("scheduleMaintenance", S, t, {"machineId": "L1", "start": "2026-03-02T08:30Z", "end": ts(TUE, "10:30"), "reason": "R"}, "starts_in_past")
    res = {"memberId": "ada", "machineId": "L1"}
    c.refuse("createReservation", S, t, {**res, "start": ts(TUE, "10:15"), "end": ts(TUE, "10:15")}, "invalid_interval")
    c.refuse("createReservation", S, t, {**res, "start": "2026-03-02T08:10Z", "end": "2026-03-02T08:40Z"}, "invalid_slot")
    c.refuse("createReservation", S, t, {**res, "start": "2026-03-02T08:00Z", "end": "2026-03-02T08:30Z"}, "starts_in_past")
    c.ok("setMemberStanding", S, t, {"memberId": "cy", "standing": "suspended"})
    c.refuse("createReservation", S, t, {"memberId": "cy", "machineId": "L1", "start": ts(TUE, "07:00"), "end": ts(TUE, "07:30")}, "outside_opening_hours")
    c.refuse("createReservation", S, t, {"memberId": "cy", "machineId": "W1", "start": ts(TUE, "10:00"), "end": ts(TUE, "10:30")}, "member_suspended")
    c.ok("setMemberStanding", S, t, {"memberId": "cy", "standing": "good"})
    c.ok("scheduleMaintenance", S, t, {"machineId": "W1", "start": ts(TUE, "12:00"), "end": ts(TUE, "13:00"), "reason": "R"})
    c.refuse("createReservation", S, t, {"memberId": "ada", "machineId": "W1", "start": ts(TUE, "12:00"), "end": ts(TUE, "12:30")}, "not_certified")
    w.reserve(c, "bob", "L2", ts(TUE, "13:00"), ts(TUE, "13:30"), t)
    c.ok("scheduleMaintenance", S, t, {"machineId": "L2", "start": ts(TUE, "14:30"), "end": ts(TUE, "15:00"), "reason": "R"})
    c.refuse("createReservation", S, t, {"memberId": "ada", "machineId": "L2", "start": ts(TUE, "13:00"), "end": ts(TUE, "15:00")}, "machine_in_maintenance")
    w.reserve(c, "bob", "L2", ts(TUE, "10:00"), ts(TUE, "11:00"), t)
    c.refuse("createReservation", S, t, {"memberId": "ada", "machineId": "L2", "start": ts(TUE, "10:00"), "end": ts(TUE, "10:30")}, "machine_booked")
    if since(step, "C2"):
        inc = c.ok("reportIncident", S, t, {"machineId": "L2", "severity": "high", "description": "Sparks"})
        c.ok("lockMachine", S, t, {"incidentId": inc, "start": ts(WED, "10:00"), "end": ts(WED, "11:00")})
        w.reserve(c, "bob", "L2", ts(WED, "11:30"), ts(WED, "12:00"), t)
        c.refuse("createReservation", S, t, {"memberId": "ada", "machineId": "L2", "start": ts(WED, "10:30"), "end": ts(WED, "12:00")}, "machine_locked",
                 cite=C(2, "Changes to existing operations and rules", "It is checked after `machine_in_maintenance` and before `machine_booked`."))
        c.ok("scheduleMaintenance", S, t, {"machineId": "L2", "start": ts(WED, "12:30"), "end": ts(WED, "13:30"), "reason": "R"})
        c.ok("lockMachine", S, t, {"incidentId": inc, "start": ts(WED, "13:00"), "end": ts(WED, "14:00")})
        c.refuse("createReservation", S, t, {"memberId": "ada", "machineId": "L2", "start": ts(WED, "13:00"), "end": ts(WED, "13:30")}, "machine_in_maintenance")
        done = c.ok("reportIncident", S, t, {"machineId": "L1", "severity": "low", "description": "Dust"})
        c.ok("resolveIncident", S, t, {"incidentId": done, "resolution": "Cleaned"})
        c.refuse("lockMachine", S, t, {"incidentId": done, "start": "2026-03-02T08:00Z", "end": "2026-03-02T07:00Z"}, "invalid_state")
        c.refuse("lockMachine", S, t, {"incidentId": inc, "start": "2026-03-02T08:00Z", "end": "2026-03-02T07:00Z"}, "invalid_interval")
    # checkIn: invalid_state before too early, too early before suspended, suspended before not certified.
    cancelled = w.reserve(c, "cy", "L2", ts(THU, "10:00"), ts(THU, "10:30"), t)
    c.ok("cancelReservation", S, t, {"reservationId": cancelled})
    c.refuse("checkIn", S, t, {"reservationId": cancelled}, "invalid_state")
    c.ok("setMemberStanding", S, t, {"memberId": "ada", "standing": "suspended"})
    c.refuse("checkIn", S, t, {"reservationId": ada}, "check_in_too_early")
    c.refuse("purchaseMaterial", S, t, {"memberId": "ada", "materialId": "acrylic", "quantity": 1000}, "member_suspended")
    c.ok("revokeCertification", S, t, {"memberId": "ada", "familyId": "laser"})
    c.refuse("checkIn", S, ts(TUE, "10:00"), {"reservationId": ada}, "member_suspended")
    cases = [c.build()]
    if since(step, "C5"):
        d = probe(step, "P-RULE-ORDER", "overdue")
        w.world(d)
        generate(d, "2026-03", "2026-04-01T00:00Z")
        late = "2026-05-15T00:01Z"
        d.ok("setMemberStanding", S, late, {"memberId": "cy", "standing": "suspended"})
        d.refuse("createReservation", S, late, {"memberId": "cy", "machineId": "W1", "start": "2026-05-18T10:00Z", "end": "2026-05-18T10:30Z"}, "member_suspended")
        d.refuse("purchaseMaterial", S, late, {"memberId": "cy", "materialId": "acrylic", "quantity": 1000}, "member_suspended")
        d.ok("setMemberStanding", S, late, {"memberId": "cy", "standing": "good"})
        d.refuse("createReservation", S, late, {"memberId": "cy", "machineId": "W1", "start": "2026-05-18T10:00Z", "end": "2026-05-18T10:30Z"}, "account_overdue",
                 cite=C(5, "What changes", "The new refusal `account_overdue` is checked after `member_suspended` and before `not_certified`."))
        d.refuse("purchaseMaterial", S, late, {"memberId": "cy", "materialId": "acrylic", "quantity": 1000}, "account_overdue",
                 cite=C(5, "What changes", "the new refusal `account_overdue` is checked after `member_suspended` and before `insufficient_stock`."))
        cases.append(d.build())
    return cases


def p_identifiers(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-IDENTIFIERS", "counters", capture=False)
    w.people(c, members=())
    for mid, name in (("bob", "Bob"), ("Zed", "Zed"), ("ada", "Ada")):
        c.ok("registerMember", S, T0, {"memberId": mid, "name": name, "tier": "basic"})
    w.catalog(c)
    w.certify(c, ("ada", "bob"))
    w.shop(c)
    for day in range(11):
        start = f"2026-05-{day + 4:02d}T10:00Z"
        c.ok("scheduleMaintenance", S, T1, {"machineId": "W1", "start": start, "end": f"2026-05-{day + 4:02d}T11:00Z", "reason": "R"},
             lambda wid: {"window": has(windowId=wid)}, check=True)
    one = c.ok("createReservation", S, T1, {"memberId": "ada", "machineId": "L1", "start": ts(TUE, "10:00"), "end": ts(TUE, "10:30")},
               lambda rid: {"reservation": has(reservationId=rid)}, check=True)
    c.refuse("createReservation", S, T1, {"memberId": "bob", "machineId": "L1", "start": ts(TUE, "10:00"), "end": ts(TUE, "10:30")}, "machine_booked")
    c.refuse("createReservation", S, T1, {"memberId": "nobody", "machineId": "L2", "start": ts(TUE, "10:00"), "end": ts(TUE, "10:30")}, "not_found")
    c.invalid('{"id":"bad","op":"createReservations","args":{"actor":"sam","at":"%s","memberId":"bob","machineId":"L2","start":"%s","end":"%s"}}'
              % (T1, ts(TUE, "10:00"), ts(TUE, "10:30")), "bad")
    c.ok("createReservation", S, T1, {"memberId": "bob", "machineId": "L2", "start": ts(TUE, "10:00"), "end": ts(TUE, "10:30")},
         lambda rid: {"reservation": has(reservationId=rid)}, check=True)
    c.ok("reportIncident", S, T1, {"machineId": "L1", "severity": "low", "description": "D"}, lambda iid: {"incident": has(incidentId=iid)}, check=True)
    c.ok("purchaseMaterial", S, T1, {"memberId": "ada", "materialId": "acrylic", "quantity": 1}, lambda pid: {"purchase": has(purchaseId=pid), "material": has(materialId="acrylic")}, check=True)
    c.refuse("purchaseMaterial", S, T1, {"memberId": "ada", "materialId": "acrylic", "quantity": 1000}, "insufficient_stock")
    c.ok("purchaseMaterial", S, T1, {"memberId": "bob", "materialId": "acrylic", "quantity": 1}, lambda pid: {"purchase": has(purchaseId=pid), "material": has(materialId="acrylic")}, check=True)
    ids = generate(c, "2026-03", APR1, result=lambda m: {"month": "2026-03", "invoices": [has(invoiceId=m[x], memberId=x) for x in ("Zed", "ada", "bob")]},
                   check=True, cite=W("11.1", "Members are processed in ascending `memberId` order."))
    assert ids == {"Zed": "inv-1", "ada": "inv-2", "bob": "inv-3"}
    c.ok("recordPayment", S, APR1, {"invoiceId": "inv-1", "amountCents": 2500}, lambda pid: {"payment": has(paymentId=pid), "invoice": has(invoiceId="inv-1"), "creditBalanceCents": 0}, check=True)
    c.refuse("recordPayment", S, APR1, {"invoiceId": "inv-1", "amountCents": 1}, "invoice_paid")
    c.ok("recordPayment", S, APR1, {"invoiceId": "inv-2", "amountCents": 1}, lambda pid: {"payment": has(paymentId=pid), "invoice": has(invoiceId="inv-2"), "creditBalanceCents": 0}, check=True)
    if since(step, "C2"):
        inc = "inc-1"
        for start, end in (("2026-04-06T10:00Z", "2026-04-06T11:00Z"), ("2026-04-06T12:00Z", "2026-04-06T13:00Z")):
            c.ok("lockMachine", S, APR1, {"incidentId": inc, "start": start, "end": end},
                 lambda lid: {"lockout": has(lockoutId=lid), "cancelledReservations": []}, check=True,
                 cite=C(2, "Lockouts", "Lockout identifiers are generated like the others in section 2.6: `lock-1`, `lock-2`, … with their own counter."))
    return [c.build()]


def p_no_mutation(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-NO-MUTATION", "refusals")
    w.world(c)
    t = T1
    ada = w.reserve(c, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), t)
    done = w.reserve(c, "bob", "L2", ts(MON, "09:30"), ts(MON, "10:00"), t)
    c.ok("addCourse", S, t, {"courseId": "tiny", "familyId": "laser", "title": "Tiny", "feeCents": 500, "capacity": 1})
    c.ok("enrollInCourse", S, t, {"courseId": "tiny", "memberId": "bob"})
    inc = c.ok("reportIncident", S, t, {"machineId": "L1", "severity": "low", "description": "D"})
    c.ok("resolveIncident", S, t, {"incidentId": inc, "resolution": "R"})
    spare = c.ok("scheduleMaintenance", S, t, {"machineId": "W1", "start": ts(THU, "10:00"), "end": ts(THU, "11:00"), "reason": "Spare"})
    c.ok("cancelMaintenance", S, t, {"windowId": spare})
    refusals = [
        ("registerMember", S, {"memberId": S, "name": "Sam", "tier": "basic"}, "already_exists"),
        ("registerMember", S, {"memberId": "ada", "name": "Other Ada", "tier": "pro"}, "already_exists"),
        ("registerUser", "admin", {"userId": "ada", "name": "Ada", "role": "staff"}, "already_exists"),
        ("changeTier", "ada", {"memberId": "ada", "tier": "pro"}, "forbidden"),
        ("setMemberStanding", S, {"memberId": "ada", "standing": "inactive"}, "invalid_argument"),
        ("enrollInCourse", S, {"courseId": "tiny", "memberId": "cy"}, "course_full"),
        ("completeCourse", S, {"courseId": "tiny", "memberId": "cy"}, "not_enrolled"),
        ("revokeCertification", S, {"memberId": "ada", "familyId": "wood"}, "not_certified"),
        ("addMachine", "admin", {"machineId": "L1", "name": "Other", "familyId": "wood", "rateCentsPerMinute": 99}, "already_exists"),
        ("setMachineRate", S, {"machineId": "L1", "rateCentsPerMinute": 99}, "forbidden"),
        ("scheduleMaintenance", S, {"machineId": "L1", "start": ts(TUE, "10:30"), "end": ts(TUE, "12:00"), "reason": "R"}, "reservation_conflict"),
        ("cancelMaintenance", S, {"windowId": spare}, "invalid_state"),
        ("createReservation", S, {"memberId": "bob", "machineId": "L1", "start": ts(TUE, "10:30"), "end": ts(TUE, "11:30")}, "machine_booked"),
        ("checkIn", S, {"reservationId": ada}, "check_in_too_early"),
        ("checkOut", S, {"reservationId": ada}, "invalid_state"),
        ("addMaterial", S, {"materialId": "acrylic", "name": "Other", "unitPriceCents": 1}, "already_exists"),
        ("purchaseMaterial", S, {"memberId": "ada", "materialId": "acrylic", "quantity": 101}, "insufficient_stock"),
        ("restockMaterial", S, {"materialId": "acrylic", "quantity": 0}, "invalid_argument"),
        ("reportIncident", S, {"machineId": "nope", "severity": "low", "description": "D"}, "not_found"),
        ("resolveIncident", S, {"incidentId": inc, "resolution": "Again"}, "invalid_state"),
        ("generateInvoices", S, {"month": "2026-03"}, "month_not_ended"),
        ("issueCredit", S, {"memberId": "ada", "amountCents": 100, "reason": "R"}, "forbidden"),
    ]
    if since(step, "C2"):
        live = c.ok("reportIncident", S, t, {"machineId": "L1", "severity": "high", "description": "Live"})
        refusals.append(("lockMachine", S, {"incidentId": live, "start": ts(TUE, "10:00"), "end": ts(TUE, "10:00")}, "invalid_interval"))
    for op, actor, args, code in refusals:
        c.refuse(op, actor, t, args, code, snapshot=True)
    check_in(c, done, ts(MON, "09:30"))
    check_out(c, done, ts(MON, "09:45"))
    c.refuse("cancelReservation", S, ts(MON, "09:45"), {"reservationId": done}, "invalid_state", snapshot=True)
    march = generate(c, "2026-03", APR1)
    c.refuse("generateInvoices", S, APR1, {"month": "2026-03"}, "already_invoiced", snapshot=True)
    c.ok("issueCredit", "admin", APR1, {"memberId": "cy", "amountCents": 999999, "reason": "Cover"})
    c.ok("recordPayment", S, APR1, {"invoiceId": march["bob"], "amountCents": 999999})
    c.refuse("recordPayment", S, APR1, {"invoiceId": march["bob"], "amountCents": 5}, "invoice_paid", snapshot=True)
    c.refuse("registerUser", "admin", APR1, {"userId": "admin", "name": "X", "role": "staff"}, "already_exists")
    c.ok("registerUser", "admin", APR1, {"userId": "still-admin", "name": "S", "role": "staff"},
         {"user": f.user("still-admin", "S", "staff")}, check=True)
    return [c.build()]


# --- Authorization ---------------------------------------------------------------------------

def _authz_case(step: str, family: str) -> Case:
    c = probe(step, family, "matrix")
    authz_world(c)
    return c


def p_authz_role(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-AUTHZ-ROLE", "matrix")
    refs = authz_world(c)
    for op, info in f.ops(step).items():
        for role, perm in info["perms"].items():
            if perm == "none":
                c.refuse(op, ACTOR_OF_ROLE[role], APR1, valid(op, refs), "forbidden")
    return [c.build()]


def p_authz_own(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-AUTHZ-OWN", "ownership")
    w.world(c)
    march = generate(c, "2026-03", APR1)
    day = "2026-04-01"
    own1 = w.reserve(c, "ada", "L1", ts(day, "10:00"), ts(day, "10:30"), APR1)
    bob1 = w.reserve(c, "bob", "L2", ts(day, "10:00"), ts(day, "10:30"), APR1)
    own2 = w.reserve(c, "ada", "L1", ts(day, "11:00"), ts(day, "11:30"), APR1)
    bob2 = w.reserve(c, "bob", "L2", ts(day, "11:00"), ts(day, "11:30"), APR1)
    for op in ("getMember", "listCertifications", "listMemberReservations", "listInvoices", "getAccount"):
        c.refuse(op, "ada", APR1, {"memberId": "bob"}, "forbidden")
        c.ok(op, "ada", APR1, {"memberId": "ada"})
    c.refuse("enrollInCourse", "ada", APR1, {"courseId": "laser-101", "memberId": "bob"}, "forbidden")
    c.ok("enrollInCourse", "ada", APR1, {"courseId": "laser-101", "memberId": "ada"})
    c.refuse("purchaseMaterial", "ada", APR1, {"memberId": "bob", "materialId": "acrylic", "quantity": 1}, "forbidden")
    c.ok("purchaseMaterial", "ada", APR1, {"memberId": "ada", "materialId": "acrylic", "quantity": 1})
    slot = {"machineId": "L1", "start": ts(day, "13:00"), "end": ts(day, "13:30")}
    c.refuse("createReservation", "ada", APR1, {"memberId": "bob", **slot}, "forbidden")
    c.ok("createReservation", "ada", APR1, {"memberId": "ada", **slot})
    c.refuse("cancelReservation", "ada", APR1, {"reservationId": bob2}, "forbidden")
    c.ok("cancelReservation", "ada", APR1, {"reservationId": own2})
    c.refuse("recordPayment", "ada", APR1, {"invoiceId": march["bob"], "amountCents": 100}, "forbidden")
    c.ok("recordPayment", "ada", APR1, {"invoiceId": march["ada"], "amountCents": 100})
    c.refuse("checkIn", "ada", ts(day, "10:00"), {"reservationId": bob1}, "forbidden")
    check_in(c, own1, ts(day, "10:00"), "ada")
    c.refuse("checkOut", "ada", ts(day, "10:10"), {"reservationId": bob1}, "forbidden")
    check_out(c, own1, ts(day, "10:10"), "ada")
    return [c.build()]


def p_authz_all(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-AUTHZ-ALL", "permitted")
    w.world(c)
    for actor, day, tag in ((S, MON, "s"), ("admin", TUE, "a")):
        t = ts(MON, "09:00") if actor == S else ts(MON, "09:30")
        if actor == "admin":
            c.ok("registerUser", "admin", t, {"userId": "ann", "name": "Ann", "role": "staff"})
            c.ok("addFamily", "admin", t, {"familyId": "metal", "name": "Metal"})
            c.ok("addMachine", "admin", t, {"machineId": "M1", "name": "Mill", "familyId": "metal", "rateCentsPerMinute": 12})
            c.ok("setMachineRate", "admin", t, {"machineId": "M1", "rateCentsPerMinute": 13})
            c.ok("issueCredit", "admin", t, {"memberId": "bob", "amountCents": 10, "reason": "R"})
            c.ok("listAuditEntries", "admin", t)
        c.ok("registerMember", actor, t, {"memberId": f"new-{tag}", "name": "New", "tier": "basic"})
        for op, args in (("getMember", {"memberId": "bob"}), ("changeTier", {"memberId": "bob", "tier": "basic"}),
                         ("setMemberStanding", {"memberId": "bob", "standing": "good"}),
                         ("addCourse", {"courseId": f"wood-{tag}", "familyId": "wood", "title": "Lathe", "feeCents": 0, "capacity": 5}),
                         ("enrollInCourse", {"courseId": f"wood-{tag}", "memberId": "bob"}),
                         ("completeCourse", {"courseId": f"wood-{tag}", "memberId": "bob"}),
                         ("revokeCertification", {"memberId": "bob", "familyId": "wood"}),
                         ("listCertifications", {"memberId": "bob"}), ("listMachines", {}),
                         ("listMemberReservations", {"memberId": "bob"}), ("getMachineSchedule", {"machineId": "L1", "date": day}),
                         ("addMaterial", {"materialId": f"wax-{tag}", "name": "Wax", "unitPriceCents": 50}),
                         ("restockMaterial", {"materialId": f"wax-{tag}", "quantity": 10}),
                         ("purchaseMaterial", {"memberId": "bob", "materialId": f"wax-{tag}", "quantity": 1}), ("listMaterials", {}),
                         ("listIncidents", {}), ("utilizationReport", {"from": MON, "to": TUE}), ("revenueReport", {"month": "2026-03"})):
            c.ok(op, actor, t, args)
        wid = c.ok("scheduleMaintenance", actor, t, {"machineId": "W1", "start": ts(THU, "10:00"), "end": ts(THU, "11:00"), "reason": "R"})
        c.ok("cancelMaintenance", actor, t, {"windowId": wid})
        gone = w.reserve(c, "bob", "L1", ts(day, "10:00"), ts(day, "10:30"), t, actor)
        c.ok("cancelReservation", actor, t, {"reservationId": gone})
        inc = c.ok("reportIncident", actor, t, {"machineId": "L2", "severity": "low", "description": "D"})
        c.ok("resolveIncident", actor, t, {"incidentId": inc, "resolution": "R"})
        if since(step, "C2"):
            open_inc = c.ok("reportIncident", actor, t, {"machineId": "W1", "severity": "low", "description": "D"})
            c.ok("lockMachine", actor, t, {"incidentId": open_inc, "start": ts(FRI, "10:00" if actor == S else "12:00"),
                                           "end": ts(FRI, "11:00" if actor == S else "13:00")})
    for op, args in (("listMachines", {}), ("getMachineSchedule", {"machineId": "L1", "date": TUE}), ("listMaterials", {})):
        c.ok(op, "ada", ts(MON, "09:30"), args)
    if since(step, "C4"):
        t = ts(MON, "09:30")
        c.ok("registerUser", "admin", t, {"userId": "stu", "name": "Stu", "role": "steward"})
        wid = c.ok("scheduleMaintenance", "stu", t, {"machineId": "W1", "start": ts(THU, "14:00"), "end": ts(THU, "15:00"), "reason": "R"})
        c.ok("cancelMaintenance", "stu", t, {"windowId": wid})
        inc = c.ok("reportIncident", "stu", t, {"machineId": "W1", "severity": "low", "description": "D"})
        c.ok("lockMachine", "stu", t, {"incidentId": inc, "start": ts(FRI, "14:00"), "end": ts(FRI, "15:00")})
        c.ok("resolveIncident", "stu", t, {"incidentId": inc, "resolution": "R"})
        for op, args in (("listIncidents", {}), ("listMachines", {}), ("getMachineSchedule", {"machineId": "L1", "date": TUE}), ("listMaterials", {})):
            c.ok(op, "stu", t, args)
    s_res = w.reserve(c, "bob", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), ts(MON, "09:30"), S)
    a_res = w.reserve(c, "bob", "L2", ts(TUE, "12:00"), ts(TUE, "12:30"), ts(MON, "09:30"), "admin")
    check_in(c, s_res, ts(TUE, "11:00"), S)
    check_out(c, s_res, ts(TUE, "11:10"), S)
    check_in(c, a_res, ts(TUE, "12:00"), "admin")
    check_out(c, a_res, ts(TUE, "12:10"), "admin")
    march = generate(c, "2026-03", APR1, S)
    generate(c, "2026-02", APR1, "admin")
    for actor in (S, "admin"):
        c.ok("listInvoices", actor, APR1, {"memberId": "bob"})
        c.ok("recordPayment", actor, APR1, {"invoiceId": march["bob"], "amountCents": 100})
        c.ok("getAccount", actor, APR1, {"memberId": "bob"})
    return [c.build()]


# --- Audit -------------------------------------------------------------------------------------

def p_audit_accepted(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-AUDIT-ACCEPTED", "log")
    w.world(c)
    t = T1
    c.ok("changeTier", S, t, {"memberId": "ada", "tier": "plus"})
    c.ok("setMemberStanding", S, t, {"memberId": "bob", "standing": "good"})
    c.ok("setMachineRate", "admin", t, {"machineId": "L2", "rateCentsPerMinute": 31})
    c.ok("revokeCertification", S, t, {"memberId": "cy", "familyId": "laser"})
    wid = c.ok("scheduleMaintenance", S, t, {"machineId": "W1", "start": ts(TUE, "10:00"), "end": ts(TUE, "11:00"), "reason": "R"})
    c.ok("cancelMaintenance", S, t, {"windowId": wid})
    own = c.ok("createReservation", "ada", t, {"memberId": "ada", "machineId": "L1", "start": ts(MON, "10:00"), "end": ts(MON, "10:30")})
    other = w.reserve(c, "bob", "L2", ts(MON, "10:00"), ts(MON, "10:30"), t)
    c.ok("cancelReservation", "bob", t, {"reservationId": other})
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "acrylic", "quantity": 2})
    inc = c.ok("reportIncident", S, t, {"machineId": "L1", "severity": "low", "description": "D"})
    c.ok("resolveIncident", S, t, {"incidentId": inc, "resolution": "R"})
    c.ok("issueCredit", "admin", t, {"memberId": "bob", "amountCents": 500, "reason": "R"})
    if since(step, "C2"):
        inc2 = c.ok("reportIncident", S, t, {"machineId": "L2", "severity": "high", "description": "D"})
        w.reserve(c, "ada", "L2", ts(TUE, "10:00"), ts(TUE, "10:30"), t)
        c.ok("lockMachine", S, t, {"incidentId": inc2, "start": ts(TUE, "09:00"), "end": ts(TUE, "11:00")})
    if since(step, "C4"):
        c.ok("registerUser", "admin", t, {"userId": "stu", "name": "Stu", "role": "steward"})
        wid = c.ok("scheduleMaintenance", "stu", t, {"machineId": "W1", "start": ts(WED, "10:00"), "end": ts(WED, "11:00"), "reason": "R"})
        c.ok("cancelMaintenance", "stu", t, {"windowId": wid})
        inc3 = c.ok("reportIncident", "stu", t, {"machineId": "W1", "severity": "low", "description": "D"})
        c.ok("resolveIncident", "stu", t, {"incidentId": inc3, "resolution": "R"})
    check_in(c, own, ts(MON, "10:00"), "ada")
    check_out(c, own, ts(MON, "10:10"), "ada")
    march = generate(c, "2026-03", APR1)
    generate(c, "2026-02", APR1)
    c.ok("recordPayment", "ada", APR1, {"invoiceId": march["ada"], "amountCents": 100})
    c.audit_log()
    return [c.build()]


def p_audit_silent(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-AUDIT-SILENT", "reads-and-refusals")
    w.world(c)
    res = w.reserve(c, "bob", "L1", ts(TUE, "10:00"), ts(TUE, "10:30"), T1)
    inc = c.ok("reportIncident", S, T1, {"machineId": "L1", "severity": "low", "description": "D"})
    c.ok("resolveIncident", S, T1, {"incidentId": inc, "resolution": "R"})
    mark = c.audit_mark("before")
    for op, args in (("getMember", {"memberId": "bob"}), ("listCertifications", {"memberId": "bob"}), ("listMachines", {}),
                     ("getMachineSchedule", {"machineId": "L1", "date": TUE}), ("listMemberReservations", {"memberId": "bob"}),
                     ("listMaterials", {}), ("listInvoices", {"memberId": "bob"}), ("getAccount", {"memberId": "bob"}),
                     ("listIncidents", {}), ("utilizationReport", {"from": MON, "to": TUE}), ("revenueReport", {"month": "2026-03"}),
                     ("listAuditEntries", {})):
        c.ok(op, "admin", T1, args)
    for op in ("getMember", "listCertifications", "listMemberReservations", "listInvoices", "getAccount"):
        c.ok(op, "ada", T1, {"memberId": "ada"})
    c.refuse("getMember", "admin", T1, {"memberId": 17}, "invalid_argument")
    c.refuse("listMachines", "ghost", T1, {}, "unknown_actor")
    c.refuse("registerUser", "ada", "2026-03-02T08:59Z", {"userId": "u", "name": "U", "role": "staff"}, "time_regression",
             cite=C(1, "What is not recorded", "A request that fails an earlier check than the permission checks.") if since(step, "C1") else None)
    c.refuse("getMember", "admin", T1, {"memberId": "nobody"}, "not_found")
    c.refuse("cancelReservation", "ada", T1, {"reservationId": "res-99"}, "not_found")
    c.refuse("registerMember", S, T1, {"memberId": "ada", "name": "Ada", "tier": "plus"}, "already_exists")
    c.refuse("createReservation", S, T1, {"memberId": "ada", "machineId": "L1", "start": ts(TUE, "10:00"), "end": ts(TUE, "10:30")}, "machine_booked")
    c.refuse("purchaseMaterial", S, T1, {"memberId": "bob", "materialId": "acrylic", "quantity": 1000}, "insufficient_stock")
    c.refuse("generateInvoices", S, T1, {"month": "2026-03"}, "month_not_ended")
    c.refuse("resolveIncident", S, T1, {"incidentId": inc, "resolution": "Again"}, "invalid_state")
    c.refuse("checkOut", S, T1, {"reservationId": res}, "invalid_state")
    c.invalid("not json", None)
    c.invalid('{"id":"x1","op":"registerMembers","args":{"actor":"sam","at":"%s","memberId":"q","name":"Q","tier":"basic"}}' % T1, "x1")
    c.audit_since("before", mark)
    return [c.build()]


def p_audit_forbidden(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-AUDIT-FORBIDDEN", "refusal-entries")
    w.world(c)
    if since(step, "C4"):
        c.ok("registerUser", "admin", T0, {"userId": "stu", "name": "Stu", "role": "steward"})
    res = w.reserve(c, "bob", "L1", ts(TUE, "10:00"), ts(TUE, "10:30"), T1)
    mark = c.audit_mark("before")
    c.refuse("listAuditEntries", "ada", T1, {}, "forbidden")
    c.refuse("issueCredit", S, T1, {"memberId": "bob", "amountCents": 100, "reason": "R"}, "forbidden")
    c.refuse("cancelReservation", "ada", T1, {"reservationId": res}, "forbidden")
    c.refuse("getMember", "ada", T1, {"memberId": "bob"}, "forbidden")
    if since(step, "C4"):
        c.refuse("addMachine", "stu", T1, {"machineId": "X1", "name": "X", "familyId": "laser", "rateCentsPerMinute": 5}, "forbidden",
                 cite=C(4, "What changes", "A steward's `forbidden` refusals are recorded as C1 states, also with `actorRole` `steward`."))
    c.refuse("registerUser", "ada", T1, {"userId": "u", "name": "U", "role": "staff"}, "forbidden")
    c.ok("changeTier", S, T1, {"memberId": "bob", "tier": "basic"})
    c.audit_since("before", mark)
    return [c.build()]


# --- Money ---------------------------------------------------------------------------------------

def _session(c: Case, rid: Any, day: str, start: str, minutes: int, *, rate: int, discount: int) -> None:
    """Check in at the reservation start, check out after `minutes`; compare only the amounts."""
    check_in(c, rid, ts(day, start))
    amount = f.line_amount(minutes, rate, discount)
    check_out(c, rid, stamp(ts(day, start), minutes), result={"reservation": has(amountCents=amount), "charge": has(amountCents=amount)},
              check=True)


def p_money_rounding(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-MONEY-ROUNDING", "half-up")
    w.world(c, fee=0)
    c.ok("addMachine", "admin", T1, {"machineId": "R5", "name": "Rate five", "familyId": "laser", "rateCentsPerMinute": 5})
    c.ok("addMachine", "admin", T1, {"machineId": "R6", "name": "Rate six", "familyId": "laser", "rateCentsPerMinute": 6})
    # 1 x 6 x 75 = 4.5 -> 5; 1 x 5 x 90 = 4.5 -> 5; 3 x 5 x 90 = 13.5 -> 14 (never 3 x 5); 37 x 45 x 90 -> 1499 (w0 4.3).
    plan = [("cy", "R6", "10:00", 1, 6, 25), ("ada", "R5", "10:30", 1, 5, 10), ("ada", "R5", "11:30", 3, 5, 10),
            ("ada", "L1", "12:30", 37, 45, 10), ("cy", "R6", "13:30", 3, 6, 25)]
    ids = [w.reserve(c, member, machine, ts(TUE, start), stamp(ts(TUE, start), 30), T1) for member, machine, start, *_ in plan]
    for rid, (_, _, start, minutes, rate, discount) in zip(ids, plan):
        _session(c, rid, TUE, start, minutes, rate=rate, discount=discount)
    buy_at = ts(TUE, "14:30")
    d10 = f.bulk_discount(step, 10)
    c.ok("purchaseMaterial", S, buy_at, {"memberId": "bob", "materialId": "acrylic", "quantity": 10},
         {"purchase": has(amountCents=f.line_amount(10, 199, d10)), "material": ANY},
         check=True, cite=C(3, "What changes", "rounded half up once per line, as before"))
    totals = W("4.3", "Totals are sums of rounded line amounts and are never rounded again.")
    c.ok("getAccount", S, buy_at, {"memberId": "ada"}, {"account": has(unbilledCents=5 + 14 + 1499)}, check=True, cite=totals)
    c.ok("getAccount", S, buy_at, {"memberId": "cy"}, {"account": has(unbilledCents=5 + 14)}, check=True, cite=totals)
    return [c.build()]


def p_money_usage(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-MONEY-USAGE", "charges")
    w.world(c, fee=0)
    t = T1
    r1 = w.reserve(c, "bob", "L1", ts(TUE, "10:00"), ts(TUE, "10:30"), t)
    r2 = w.reserve(c, "ada", "L2", ts(TUE, "10:00"), ts(TUE, "10:30"), t)
    r3 = w.reserve(c, "cy", "L2", ts(TUE, "11:00"), ts(TUE, "11:30"), t)
    r4 = w.reserve(c, "bob", "L1", ts(TUE, "12:00"), ts(TUE, "12:30"), t)
    r5 = w.reserve(c, "ada", "L1", ts(TUE, "20:00"), ts(TUE, "20:30"), t)
    w.reserve(c, "cy", "L1", ts(TUE, "14:00"), ts(TUE, "14:30"), t)  # never checked in: missed
    gone = w.reserve(c, "bob", "L2", ts(TUE, "15:00"), ts(TUE, "15:30"), t)
    c.ok("cancelReservation", S, t, {"reservationId": gone})
    c.ok("setMachineRate", "admin", ts(TUE, "09:00"), {"machineId": "L1", "rateCentsPerMinute": 50})
    check_in(c, r1, ts(TUE, "10:00"), result={"reservation": has(rateCentsPerMinute=50, amountCents=None)}, check=True)
    check_in(c, r2, ts(TUE, "10:00"))
    c.ok("setMachineRate", "admin", ts(TUE, "10:05"), {"machineId": "L1", "rateCentsPerMinute": 60})
    c.ok("changeTier", S, ts(TUE, "10:05"), {"memberId": "ada", "tier": "pro"})

    usage: dict[str, list] = {"ada": [], "bob": [], "cy": []}

    def out(rid, member, at, minutes, rate, discount):
        charge = f.charge("usage", rid, minutes, rate, discount, at)
        check_out(c, rid, at, result={"reservation": has(meteredMinutes=minutes, amountCents=charge["amountCents"]),
                                       "charge": charge}, check=True)
        usage[member].append(charge)

    out(r1, "bob", ts(TUE, "10:10"), 10, 50, 0)
    out(r2, "ada", ts(TUE, "10:20"), 20, 30, 25)
    c.ok("changeTier", S, ts(TUE, "10:21"), {"memberId": "ada", "tier": "plus"})
    check_in(c, r3, ts(TUE, "11:00"))
    out(r3, "cy", ts(TUE, "11:00"), 0, 30, 25)
    check_in(c, r4, ts(TUE, "12:00"))
    out(r4, "bob", ts(TUE, "13:15"), 75, 60, 0)
    c.ok("getAccount", S, ts(TUE, "14:30"), {"memberId": "cy"}, {"account": has(unbilledCents=0)}, check=True,
         cite=W("9.1", "A missed reservation can no longer be checked in or cancelled and causes no charge."))
    check_in(c, r5, ts(TUE, "20:00"))
    out(r5, "ada", ts(TUE, "21:10"), 70, 60, 10)
    c.ok("setMachineRate", "admin", ts(TUE, "21:45"), {"machineId": "L1", "rateCentsPerMinute": 99})
    c.ok("getAccount", S, ts(TUE, "21:45"), {"memberId": "bob"}, {"account": has(unbilledCents=500 + 4500)}, check=True)
    c.ok("getAccount", S, ts(TUE, "21:45"), {"memberId": "ada"}, {"account": has(unbilledCents=450 + 3780)}, check=True)
    generate(c, "2026-03", APR1)
    for member in ("ada", "bob", "cy"):
        c.ok("listInvoices", S, APR1, {"memberId": member}, {"memberId": member, "invoices": [has(lines=lines("usage", usage[member]))]}, check=True,
             cite=W("9.2", "A charge of 0 cents is still created."))
    return [c.build()]


def p_money_course(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-MONEY-COURSE", "charges")
    w.people(c)
    w.catalog(c)
    c.ok("addCourse", S, T1, {"courseId": "wood-101", "familyId": "wood", "title": "Lathe basics", "feeCents": 0, "capacity": 5})
    c.ok("addCourse", S, T1, {"courseId": "laser-pro", "familyId": "laser", "title": "Laser pro", "feeCents": 4550, "capacity": 5})
    times = [ts(MON, hm) for hm in ("10:00", "11:00", "12:00", "13:00", "14:00")]
    c.ok("enrollInCourse", S, times[0], {"courseId": "laser-pro", "memberId": "cy"})
    c.ok("enrollInCourse", S, times[1], {"courseId": "wood-101", "memberId": "ada"})
    c.ok("completeCourse", S, times[2], {"courseId": "wood-101", "memberId": "ada"})
    c.ok("enrollInCourse", S, times[3], {"courseId": "wood-101", "memberId": "ada"},
         cite=W("7.1", "It is created by the first enrollment; a later enrollment after completion reuses and resets it."))
    c.ok("enrollInCourse", S, times[4], {"courseId": "laser-pro", "memberId": "ada"})
    c.ok("getAccount", S, times[4], {"memberId": "cy"}, {"account": has(unbilledCents=4550)}, check=True)
    c.ok("getAccount", S, times[4], {"memberId": "ada"}, {"account": has(unbilledCents=4550)}, check=True)
    course = {"cy": [f.charge("course", "laser-pro", 1, 4550, 0, times[0])],
              "ada": [f.charge("course", "wood-101", 1, 0, 0, times[1]), f.charge("course", "wood-101", 1, 0, 0, times[3]),
                      f.charge("course", "laser-pro", 1, 4550, 0, times[4])],
              "bob": []}
    generate(c, "2026-03", APR1)
    for member in ("ada", "bob", "cy"):
        c.ok("listInvoices", S, APR1, {"memberId": member}, {"memberId": member, "invoices": [has(lines=lines("course", course[member]))]}, check=True)
    return [c.build()]


def p_money_materials(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-MONEY-MATERIALS", "purchases")
    w.people(c)
    w.shop(c)
    c.ok("addMaterial", S, T1, {"materialId": "bead", "name": "Bead", "unitPriceCents": 3})
    c.ok("restockMaterial", S, T1, {"materialId": "bead", "quantity": 100000})
    stock = {"acrylic": 100, "plywood": 100, "bead": 100000}
    price = {"acrylic": 199, "plywood": 37, "bead": 3}
    names = {"acrylic": "Acrylic sheet", "plywood": "Plywood panel", "bead": "Bead"}
    bought: dict[str, list] = {"ada": [], "bob": [], "cy": []}
    for member, mat, qty in (("ada", "acrylic", 9), ("ada", "acrylic", 10), ("bob", "plywood", 24), ("bob", "plywood", 25),
                             ("cy", "acrylic", 5), ("cy", "acrylic", 5), ("bob", "bead", 10000), ("ada", "plywood", 51)):
        stock[mat] -= qty
        d = f.bulk_discount(step, qty)
        pid = c.ok("purchaseMaterial", S, T1, {"memberId": member, "materialId": mat, "quantity": qty},
                   lambda pid, member=member, mat=mat, qty=qty, d=d: {
                       "purchase": f.purchase(pid, member, mat, qty, price[mat], d, T1),
                       "material": f.material(mat, names[mat], price[mat], stock[mat])}, check=True)
        bought[member].append(f.charge("material", pid, qty, price[mat], d, T1))
    c.refuse("purchaseMaterial", S, T1, {"memberId": "ada", "materialId": "plywood", "quantity": 1}, "insufficient_stock")
    generate(c, "2026-03", APR1)
    for member in ("ada", "bob", "cy"):
        c.ok("listInvoices", S, APR1, {"memberId": member}, {"memberId": member, "invoices": [has(lines=lines("material", bought[member]))]}, check=True,
             cite=C(3, "What changes", "The material charge carries the same `discountPercent` and `amountCents` as the purchase."))
    return [c.build()]


def p_money_invoice(step: str) -> list[dict[str, Any]]:
    """Invoice construction; the material lines are taken from the purchases, so the materials rules do not matter."""
    c = probe(step, "P-MONEY-INVOICE", "construction")
    feb_reg, feb_buy = "2026-02-10T09:00Z", "2026-02-15T09:00Z"
    c.ok("registerMember", "admin", feb_reg, {"memberId": "fay", "name": "Fay", "tier": "basic"})
    c.ok("addMaterial", "admin", feb_reg, {"materialId": "felt", "name": "Felt", "unitPriceCents": 50})
    c.ok("restockMaterial", "admin", feb_reg, {"materialId": "felt", "quantity": 10})

    def buy(member, material, quantity, at, name, actor=S):
        return c.ok("purchaseMaterial", actor, at, {"memberId": member, "materialId": material, "quantity": quantity},
                    {"purchase": has(amountCents={"$capture": name}), "material": ANY}, check=True)

    felt = buy("fay", "felt", 2, feb_buy, "felt", actor="admin")
    w.people(c)
    w.shop(c)
    sheet = buy("ada", "acrylic", 2, "2026-03-10T10:00Z", "sheet")
    c.ok("changeTier", S, "2026-03-15T10:00Z", {"memberId": "bob", "tier": "pro"})
    c.ok("setMemberStanding", S, "2026-03-15T10:00Z", {"memberId": "cy", "standing": "suspended"})
    c.ok("issueCredit", "admin", "2026-03-20T10:00Z", {"memberId": "ada", "amountCents": 1000, "reason": "R"})
    c.ok("issueCredit", "admin", "2026-03-20T10:00Z", {"memberId": "cy", "amountCents": 20000, "reason": "R"})
    c.ok("registerMember", S, "2026-03-31T23:59Z", {"memberId": "dee", "name": "Dee", "tier": "basic"})
    apr = "2026-04-01T00:00Z"
    gen = "2026-04-01T09:00Z"
    c.ok("registerMember", S, apr, {"memberId": "eve", "name": "Eve", "tier": "plus"})
    panel = buy("ada", "plywood", 1, apr, "panel")

    def with_material(member, tier, month, at, ref, name, bought, credit=0):
        total = {"$calc": ["+", f.FEES[tier], {"$ref": name}]}
        return has(memberId=member, month=month, lines=[membership_line(tier, at), has(kind="material", ref=ref, at=bought)],
                   totalCents=total, creditAppliedCents=credit, paidCents=0, balanceCents={"$calc": ["-", total, credit]}, status="open")

    march = {"ada": with_material("ada", "plus", "2026-03", gen, sheet, "sheet", "2026-03-10T10:00Z", credit=1000),
             "bob": money("bob", "2026-03", [membership_line("pro", gen)]),
             "cy": money("cy", "2026-03", [membership_line("pro", gen)], credit=9900),
             "dee": money("dee", "2026-03", [membership_line("basic", gen)]),
             "fay": money("fay", "2026-03", [membership_line("basic", gen)])}
    mar = generate(c, "2026-03", gen, result=lambda m: {"month": "2026-03", "invoices": [march[x] for x in ("ada", "bob", "cy", "dee", "fay")]}, check=True,
                   cite=W("11.1", "All of the member's unbilled charges whose `at` lies in month M follow"))
    c.ok("getAccount", S, gen, {"memberId": "ada"}, {"account": has(creditBalanceCents=0)}, check=True)
    c.ok("getAccount", S, gen, {"memberId": "cy"}, {"account": has(creditBalanceCents=10100, openBalanceCents=0)}, check=True)
    generate(c, "2026-01", gen, result={"month": "2026-01", "invoices": []}, check=True,
             cite=W("11.1", "Effect: creates the invoices as described above and marks the month as invoiced, even when there is no member to invoice."))
    c.refuse("generateInvoices", S, gen, {"month": "2026-01"}, "already_invoiced")
    zero = {k: 0 for k in ("invoiceCount", "membershipCents", "usageCents", "materialCents", "courseCents", "invoicedCents",
                           "creditAppliedCents", "outstandingCents", "paymentsCents", "creditsIssuedCents")}
    c.ok("revenueReport", S, gen, {"month": "2026-01"}, {"month": "2026-01", "invoicesGenerated": True, **zero}, check=True,
         cite=W("13", "`invoicesGenerated`: whether invoices for the month have been generated;"))
    fe = generate(c, "2026-02", gen, result=lambda m: {"month": "2026-02", "invoices": [with_material("fay", "basic", "2026-02", gen, felt, "felt", feb_buy)]}, check=True)
    c.ok("listInvoices", S, gen, {"memberId": "fay"}, {"memberId": "fay", "invoices": [has(invoiceId=fe["fay"], month="2026-02"), has(invoiceId=mar["fay"], month="2026-03")]},
         check=True, cite=W("4.5", "| `listInvoices` → `invoices` | `month` ascending |"))
    may = "2026-05-01T09:00Z"
    april = {"ada": with_material("ada", "plus", "2026-04", may, panel, "panel", apr),
             "bob": money("bob", "2026-04", [membership_line("pro", may)]),
             "cy": money("cy", "2026-04", [membership_line("pro", may)], credit=9900),
             "dee": money("dee", "2026-04", [membership_line("basic", may)]),
             "eve": money("eve", "2026-04", [membership_line("plus", may)]),
             "fay": money("fay", "2026-04", [membership_line("basic", may)])}
    generate(c, "2026-04", may, result=lambda m: {"month": "2026-04", "invoices": [april[x] for x in ("ada", "bob", "cy", "dee", "eve", "fay")]}, check=True)
    c.ok("getAccount", S, may, {"memberId": "cy"}, {"account": has(creditBalanceCents=200)}, check=True)
    return [c.build()]


def p_money_credit(step: str) -> list[dict[str, Any]]:
    """Payment and credit fields only; invoice balances are read first, so fees and charges do not matter."""
    c = probe(step, "P-MONEY-CREDIT", "payments")
    w.people(c, members=("ada", "bob"))
    w.shop(c)
    c.ok("purchaseMaterial", S, "2026-03-10T10:00Z", {"memberId": "ada", "materialId": "acrylic", "quantity": 1})
    march = generate(c, "2026-03", APR1)
    a1, b1 = march["ada"], march["bob"]
    for member, inv, name in (("ada", a1, "A"), ("bob", b1, "B")):
        c.ok("listInvoices", S, APR1, {"memberId": member}, {"memberId": member, "invoices": [has(invoiceId=inv, balanceCents={"$capture": name})]}, check=True)
    A, B = {"$ref": "A"}, {"$ref": "B"}
    left = {"$calc": ["-", A, 1000]}
    pay_at = "2026-04-02T10:00Z"
    c.ok("recordPayment", S, pay_at, {"invoiceId": a1, "amountCents": 1000},
         lambda pid: {"payment": f.payment(pid, a1, "ada", 1000, 1000, pay_at),
                      "invoice": has(invoiceId=a1, paidCents=1000, balanceCents=left, status="open"), "creditBalanceCents": 0}, check=True)
    c.ok("issueCredit", "admin", pay_at, {"memberId": "ada", "amountCents": 500, "reason": "Goodwill"},
         {"memberId": "ada", "amountCents": 500, "creditBalanceCents": 500}, check=True)
    c.ok("listInvoices", S, pay_at, {"memberId": "ada"}, {"memberId": "ada", "invoices": [has(invoiceId=a1, balanceCents=left)]}, check=True)
    c.ok("getAccount", S, pay_at, {"memberId": "ada"}, {"account": has(creditBalanceCents=500, openBalanceCents=left, unbilledCents=0)}, check=True)
    big = 100000000
    credited = {"$calc": ["-", big, left]}
    credit = {"$calc": ["+", 500, credited]}
    c.ok("recordPayment", "ada", pay_at, {"invoiceId": a1, "amountCents": big},
         lambda pid: {"payment": {"paymentId": pid, "invoiceId": a1, "memberId": "ada", "amountCents": big, "appliedCents": left,
                                  "creditedCents": credited, "at": pay_at},
                      "invoice": has(invoiceId=a1, paidCents=A, balanceCents=0, status="paid"), "creditBalanceCents": credit}, check=True)
    c.refuse("recordPayment", S, pay_at, {"invoiceId": a1, "amountCents": 1}, "invoice_paid")
    c.ok("recordPayment", S, pay_at, {"invoiceId": b1, "amountCents": B},
         lambda pid: {"payment": {"paymentId": pid, "invoiceId": b1, "memberId": "bob", "amountCents": B, "appliedCents": B,
                                  "creditedCents": 0, "at": pay_at},
                      "invoice": has(status="paid", balanceCents=0), "creditBalanceCents": 0}, check=True)
    c.ok("purchaseMaterial", S, pay_at, {"memberId": "ada", "materialId": "acrylic", "quantity": 1},
         {"purchase": has(amountCents={"$capture": "P"}), "material": ANY}, check=True)
    c.ok("getAccount", S, pay_at, {"memberId": "ada"}, {"account": has(creditBalanceCents=credit, openBalanceCents=0, unbilledCents={"$ref": "P"})}, check=True)
    may = "2026-05-01T09:00Z"
    generate(c, "2026-04", may)
    c.ok("listInvoices", S, may, {"memberId": "ada"}, {"memberId": "ada", "invoices": {"$filter": {"month": "2026-04"}, "items": [has(totalCents={"$capture": "T"})]}}, check=True)
    c.ok("listInvoices", S, may, {"memberId": "ada"}, {"memberId": "ada", "invoices": {"$filter": {"month": "2026-04"}, "items": [
        has(creditAppliedCents={"$ref": "T"}, balanceCents=0, status="paid")]}}, check=True,
         cite=W("11.1", "`creditAppliedCents` is the smaller of the member's credit balance and `totalCents`."))
    c.ok("getAccount", S, may, {"memberId": "ada"}, {"account": has(creditBalanceCents={"$calc": ["-", credit, {"$ref": "T"}]}, openBalanceCents=0, unbilledCents=0)}, check=True)
    c.ok("issueCredit", "admin", may, {"memberId": "bob", "amountCents": big, "reason": "R"})
    jun = "2026-06-01T09:00Z"
    june = generate(c, "2026-05", jun)
    c.refuse("recordPayment", S, jun, {"invoiceId": june["bob"], "amountCents": 100}, "invoice_paid")
    return [c.build()]


def p_money_revenue(step: str) -> list[dict[str, Any]]:
    """Report sums against the invoices' own captured lines, so the fee and charge rules do not matter."""
    c = probe(step, "P-MONEY-REVENUE", "report")
    w.world(c, fee=0)
    r = w.reserve(c, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "10:30"), T1)
    lock_credit = 0
    if since(step, "C2"):
        inc = c.ok("reportIncident", S, T1, {"machineId": "L2", "severity": "high", "description": "D"})
        w.reserve(c, "cy", "L2", ts(WED, "10:00"), ts(WED, "10:30"), T1)
        c.ok("lockMachine", S, T1, {"incidentId": inc, "start": ts(WED, "09:00"), "end": ts(WED, "12:00")})
        lock_credit = 1000
    check_in(c, r, ts(TUE, "10:00"))
    check_out(c, r, ts(TUE, "10:10"))
    c.ok("purchaseMaterial", S, ts(TUE, "11:00"), {"memberId": "bob", "materialId": "acrylic", "quantity": 3})
    c.ok("issueCredit", "admin", ts(THU, "10:00"), {"memberId": "cy", "amountCents": 1000, "reason": "R"})
    march = generate(c, "2026-03", APR1)

    def read(member, **members):
        c.ok("listInvoices", S, APR1, {"memberId": member}, {"memberId": member, "invoices": [has(**members)]}, check=True)

    for member in ("ada", "bob", "cy"):
        read(member, lines=lines("membership", [has(amountCents={"$capture": "M-" + member})]), balanceCents={"$capture": "BAL-" + member})
    read("ada", lines=lines("usage", [has(amountCents={"$capture": "U"})]))
    read("bob", lines=lines("material", [has(amountCents={"$capture": "MAT"})]))
    read("cy", creditAppliedCents={"$capture": "CR"})
    ref = lambda name: {"$ref": name}
    c.ok("recordPayment", S, "2026-04-02T10:00Z", {"invoiceId": march["ada"], "amountCents": {"$calc": ["+", ref("BAL-ada"), 100]}})
    c.ok("recordPayment", S, "2026-04-02T10:00Z", {"invoiceId": march["bob"], "amountCents": 1000})
    c.ok("issueCredit", "admin", "2026-04-03T10:00Z", {"memberId": "ada", "amountCents": 250, "reason": "R"})
    membership = {"$calc": ["+", ref("M-ada"), ref("M-bob"), ref("M-cy")]}
    report = {"month": "2026-03", "invoicesGenerated": True, "invoiceCount": 3, "membershipCents": membership, "usageCents": ref("U"),
              "materialCents": ref("MAT"), "courseCents": 0, "invoicedCents": {"$calc": ["+", membership, ref("U"), ref("MAT")]},
              "creditAppliedCents": ref("CR"), "outstandingCents": {"$calc": ["+", {"$calc": ["-", ref("BAL-bob"), 1000]}, ref("BAL-cy")]},
              "paymentsCents": 0, "creditsIssuedCents": 1000 + lock_credit}
    c.ok("revenueReport", S, "2026-04-03T10:00Z", {"month": "2026-03"}, report, check=True)
    april = {"month": "2026-04", "invoicesGenerated": False, "invoiceCount": 0, "membershipCents": 0, "usageCents": 0,
             "materialCents": 0, "courseCents": 0, "invoicedCents": 0, "creditAppliedCents": 0, "outstandingCents": 0,
             "paymentsCents": {"$calc": ["+", ref("BAL-ada"), 100, 1000]}, "creditsIssuedCents": 250}
    c.ok("revenueReport", S, "2026-04-03T10:00Z", {"month": "2026-04"}, april, check=True,
         cite=W("13", "For a month without generated invoices, the invoice figures are 0 and `invoicesGenerated` is `false`; payments and issued credits are still reported."))
    return [c.build()]


def p_money_lockout(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-MONEY-LOCKOUT", "credit")
    w.world(c)
    r1 = w.reserve(c, "ada", "L1", ts(WED, "10:00"), ts(WED, "10:30"), T1)
    r2 = w.reserve(c, "bob", "L1", ts(WED, "11:00"), ts(WED, "11:30"), T1)
    r3 = w.reserve(c, "ada", "L1", ts(WED, "12:00"), ts(WED, "12:30"), T1)
    r4 = w.reserve(c, "cy", "L1", ts(WED, "13:00"), ts(WED, "13:30"), T1)
    c.ok("cancelReservation", S, T1, {"reservationId": r4})
    c.ok("setMemberStanding", S, T1, {"memberId": "bob", "standing": "suspended"})
    inc = c.ok("reportIncident", S, T1, {"machineId": "L1", "severity": "high", "description": "Fire"})
    c.ok("lockMachine", S, T1, {"incidentId": inc, "start": ts(WED, "09:00"), "end": ts(WED, "12:30")},
         lambda lid: {"lockout": has(lockoutId=lid, cancelledReservationIds=[r1, r2, r3]),
                      "cancelledReservations": [has(reservationId=x, status="cancelled") for x in (r1, r2, r3)]}, check=True)
    for member, credit in (("ada", 2000), ("bob", 1000), ("cy", 0)):
        c.ok("getAccount", S, T1, {"memberId": member}, {"account": has(creditBalanceCents=credit)}, check=True)
    generate(c, "2026-03", APR1)
    for member, credit in (("ada", 2000), ("bob", 1000), ("cy", 0)):
        c.ok("listInvoices", S, APR1, {"memberId": member}, {"memberId": member, "invoices": [has(creditAppliedCents=credit)]}, check=True)
    return [c.build()]


# --- Time ------------------------------------------------------------------------------------------

def p_time_order(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-TIME-ORDER", "every-operation")
    c.refuse("listMachines", "ghost", "2050-01-01T00:00Z", {}, "unknown_actor")
    c.ok("listMachines", "admin", "2026-03-02T07:00Z", {}, {"machines": []}, check=True,
         cite=W("4.4", "Before the first `ok` response, any valid `at` is accepted."))
    refs = authz_world(c)
    early = "2026-04-01T08:59Z"
    for op in f.ops(step):
        c.refuse(op, "admin", early, valid(op, refs), "time_regression")
    c.ok("changeTier", S, APR1, {"memberId": "bob", "tier": "basic"})
    c.refuse("getMember", "admin", "2026-04-01T10:00Z", {"memberId": "nobody"}, "not_found")
    c.ok("listMachines", "admin", "2026-04-01T09:30Z")
    # An invalid request with a later time; how it is answered is P-ENVELOPE's, here only the time point matters.
    c.raw('{"id":"late","op":"listMachines","args":{"actor":"admin","at":"2026-04-01T11:00Z","bogus":1}}', {"$has": {"id": "late"}},
          cite=W("2.4", "An invalid request changes nothing: no state, no audit entry, no identifier counter and no request-time order (section 4.4)."))
    c.ok("listMachines", "admin", "2026-04-01T09:45Z")
    c.refuse("listAuditEntries", "ada", "2026-04-01T12:00Z", {}, "forbidden")
    c.ok("listMachines", "admin", "2026-04-01T09:50Z")
    c.ok("registerMember", S, "2026-04-01T10:00Z", {"memberId": "late", "name": "Late", "tier": "basic"})
    c.refuse("listMachines", "admin", "2026-04-01T09:59Z", {}, "time_regression")
    c.ok("listMaterials", "admin", "2026-04-01T10:30Z")
    c.refuse("registerMember", S, "2026-04-01T10:29Z", {"memberId": "later", "name": "L", "tier": "basic"}, "time_regression")
    c.ok("registerMember", S, "2026-04-01T10:30Z", {"memberId": "later", "name": "L", "tier": "basic"})
    c.refuse("generateInvoices", S, "2026-04-01T13:00Z", {"month": "2026-05"}, "month_not_ended")
    c.ok("listMachines", "admin", "2026-04-01T10:31Z")
    return [c.build()]


def p_time_format(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-TIME-FORMAT", "values")
    start = "2000-01-01T00:00Z"
    c.ok("listMachines", "admin", start, {}, {"machines": []}, check=True)
    c.ok("addFamily", "admin", start, {"familyId": "fam", "name": "F"})
    c.ok("addMachine", "admin", start, {"machineId": "M1", "name": "M", "familyId": "fam", "rateCentsPerMinute": 1})
    for bad in ("2026-03-02T09:00", "2026-03-02T09:00:00Z", "2026-03-02 09:00Z", "2026-03-02t09:00z", "2026-02-29T10:00Z",
                "2026-03-02T24:00Z", "2026-03-02T09:60Z", "1999-12-31T23:59Z", "2100-01-01T00:00Z", "2026-3-02T09:00Z",
                "2026-03-32T09:00Z", "2026-04-31T10:00Z", "26-03-02T09:00Z"):
        c.refuse("scheduleMaintenance", "admin", start, {"machineId": "M1", "start": bad, "end": "2026-03-03T09:00Z", "reason": "R"}, "invalid_argument")
        c.refuse("listMachines", "admin", bad, {}, "invalid_argument")
    for s, e in (("2028-02-29T10:00Z", "2028-02-29T11:00Z"), ("2099-12-31T23:58Z", "2099-12-31T23:59Z")):
        c.ok("scheduleMaintenance", "admin", start, {"machineId": "M1", "start": s, "end": e, "reason": "R"},
             lambda wid, s=s, e=e: {"window": has(start=s, end=e)}, check=True)
    for bad in ("2026-02-29", "1999-12-31", "2100-01-01", "2026-3-03", "2026-03-03T00:00Z", "20260303", "2026-13-01"):
        c.refuse("getMachineSchedule", "admin", start, {"machineId": "M1", "date": bad}, "invalid_argument")
        c.refuse("utilizationReport", "admin", start, {"from": bad, "to": "2026-03-09"}, "invalid_argument")
    for good in ("2028-02-29", "2000-01-01", "2099-12-31"):
        c.ok("getMachineSchedule", "admin", start, {"machineId": "M1", "date": good}, {**_sched_has(good)}, check=True)
    for bad in ("2026-13", "2026-00", "1999-12", "2100-01", "2026-3", "2026-03-01", "202603"):
        c.refuse("revenueReport", "admin", start, {"month": bad}, "invalid_argument")
        c.refuse("generateInvoices", "admin", start, {"month": bad}, "invalid_argument")
    for good in ("2000-01", "2099-12"):
        c.ok("revenueReport", "admin", start, {"month": good}, has(month=good), check=True)
    c.ok("listMachines", "admin", "2099-12-31T23:59Z", {}, {"machines": [has(machineId="M1")]}, check=True)
    return [c.build()]


def _sched_has(date: str) -> dict[str, Any]:
    return has(machineId="M1", date=date)


def _weekday(date: str) -> int:
    import datetime
    return datetime.date.fromisoformat(date).weekday()


def p_time_hours(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-TIME-HOURS", "boundaries")
    w.world(c)
    if since(step, "C8"):
        slots = [(TUE, "08:00", "08:30", True), (TUE, "07:30", "08:00", False), (TUE, "21:30", "22:00", True), (TUE, "21:30", "22:30", False),
                 (THU, "21:30", "22:00", True), (FRI, "19:30", "20:00", True), (FRI, "19:30", "20:30", False), (FRI, "08:00", "08:30", True),
                 (SAT, "09:00", "09:30", True), (SAT, "08:30", "09:00", False), (SAT, "16:30", "17:00", True), (SAT, "16:30", "17:30", False),
                 (SAT, "17:30", "18:00", False), (SUN, "12:00", "12:30", True), (SUN, "11:30", "12:30", False), (SUN, "15:30", "16:00", True),
                 (SUN, "15:30", "16:30", False), (TUE, "23:00", "23:30", False)]
    else:
        slots = [(TUE, "09:00", "09:30", True), (TUE, "08:30", "09:00", False), (TUE, "08:30", "09:30", False), (TUE, "20:30", "21:00", True),
                 (TUE, "20:30", "21:30", False), (TUE, "21:00", "21:30", False), (FRI, "20:30", "21:00", True),
                 (SAT, "10:00", "10:30", True), (SAT, "09:30", "10:00", False), (SAT, "17:30", "18:00", True), (SAT, "17:30", "18:30", False),
                 (SUN, "12:00", "12:30", False), (SUN, "10:00", "10:30", False), (TUE, "23:00", "23:30", False)]
    for date, start, end, inside in slots:
        args = {"memberId": "ada", "machineId": "L1", "start": ts(date, start), "end": ts(date, end)}
        if inside:
            c.ok("createReservation", S, T1, args, lambda rid: {"reservation": has(reservationId=rid, status="reserved")}, check=True)
        else:
            c.refuse("createReservation", S, T1, args, "outside_opening_hours")
    hours = f.opening_hours(step)
    for date in (MON, TUE, FRI, SAT, SUN):
        h = hours[_weekday(date)]
        c.ok("getMachineSchedule", S, T1, {"machineId": "L2", "date": date},
             has(openingHours=None if h is None else {"open": h[0], "close": h[1]}), check=True)
    return [c.build()]


def p_time_granularity(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-TIME-GRANULARITY", "slots")
    w.world(c)
    if since(step, "C8"):
        rows = [("bob", "L2", "10:15", "10:30", None), ("bob", "L2", "10:10", "10:30", "invalid_slot"), ("bob", "L2", "11:00", "11:15", None),
                ("bob", "L2", "11:30", "12:15", None), ("ada", "L1", "12:00", "16:00", None), ("bob", "L2", "12:30", "16:45", "invalid_slot"),
                ("bob", "L2", "16:30", "16:35", "invalid_slot"), ("bob", "L2", "16:37", "17:00", "invalid_slot"),
                ("bob", "L2", "17:00", "17:00", "invalid_interval"), ("bob", "L2", "17:15", "17:00", "invalid_interval")]
    else:
        rows = [("ada", "L1", "10:15", "11:00", "invalid_slot"), ("ada", "L1", "10:00", "10:45", "invalid_slot"),
                ("ada", "L1", "10:01", "10:31", "invalid_slot"), ("ada", "L1", "10:00", "10:15", "invalid_slot"),
                ("ada", "L1", "10:00", "14:30", "invalid_slot"), ("ada", "L1", "10:00", "14:00", None), ("ada", "L1", "15:00", "15:30", None),
                ("ada", "L1", "16:00", "16:00", "invalid_interval"), ("ada", "L1", "16:30", "16:00", "invalid_interval"),
                ("ada", "L1", "16:15", "16:15", "invalid_interval"), ("bob", "L2", "10:30", "11:00", None)]
    for member, machine, start, end, code in rows:
        args = {"memberId": member, "machineId": machine, "start": ts(TUE, start), "end": ts(TUE, end)}
        if code is None:
            c.ok("createReservation", S, T1, args, lambda rid: {"reservation": has(reservationId=rid)}, check=True)
        else:
            c.refuse("createReservation", S, T1, args, code)
    return [c.build()]


def p_time_intervals(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-TIME-INTERVALS", "half-open")
    w.world(c)
    t = T1
    ok = lambda m, mach, s, e: c.ok("createReservation", S, t, {"memberId": m, "machineId": mach, "start": ts(TUE, s), "end": ts(TUE, e)},
                                    lambda rid: {"reservation": has(reservationId=rid)}, check=True)
    no = lambda m, mach, s, e, code: c.refuse("createReservation", S, t, {"memberId": m, "machineId": mach, "start": ts(TUE, s), "end": ts(TUE, e)}, code)
    ok("ada", "L1", "10:00", "11:00")
    ok("bob", "L1", "11:00", "12:00")
    ok("cy", "L1", "09:00", "10:00")
    no("cy", "L1", "10:30", "11:30", "machine_booked")
    no("cy", "L1", "11:30", "12:30", "machine_booked")
    ok("ada", "L2", "11:00", "11:30")
    no("ada", "L2", "10:30", "11:00", "member_double_booked")
    maint = lambda mach, s, e: c.ok("scheduleMaintenance", S, t, {"machineId": mach, "start": s, "end": e, "reason": "R"},
                                    lambda wid: {"window": has(windowId=wid)}, check=True)
    maint("L1", ts(TUE, "12:00"), ts(TUE, "13:00"))
    c.refuse("scheduleMaintenance", S, t, {"machineId": "L1", "start": ts(TUE, "11:59"), "end": ts(TUE, "12:30"), "reason": "R"}, "reservation_conflict")
    ok("cy", "L1", "13:00", "13:30")
    no("bob", "L1", "12:30", "13:00", "machine_in_maintenance")
    gone = c.ok("scheduleMaintenance", S, t, {"machineId": "L2", "start": ts(TUE, "15:00"), "end": ts(TUE, "16:00"), "reason": "R"})
    c.ok("cancelMaintenance", S, t, {"windowId": gone})
    ok("ada", "L2", "15:00", "15:30")
    w1 = [maint("W1", "2026-03-02T22:00Z", ts(TUE, "00:00")), maint("W1", ts(TUE, "23:00"), ts(WED, "01:00")), maint("W1", ts(WED, "00:00"), ts(WED, "01:00"))]
    c.ok("getMachineSchedule", S, t, {"machineId": "W1", "date": TUE},
         has(maintenance=[f.window(w1[1], "W1", ts(TUE, "23:00"), ts(WED, "01:00"), "R")]), check=True)
    c.ok("getMachineSchedule", S, t, {"machineId": "W1", "date": WED},
         has(maintenance=[f.window(w1[1], "W1", ts(TUE, "23:00"), ts(WED, "01:00"), "R"), f.window(w1[2], "W1", ts(WED, "00:00"), ts(WED, "01:00"), "R")]),
         check=True)
    c.ok("getMachineSchedule", S, t, {"machineId": "L2", "date": TUE}, has(maintenance=[]), check=True)
    if since(step, "C2"):
        inc = c.ok("reportIncident", S, t, {"machineId": "L2", "severity": "high", "description": "D"})
        w.reserve(c, "cy", "L2", ts(WED, "12:00"), ts(WED, "12:30"), t)
        lock = lambda s, e, cancelled: c.ok("lockMachine", S, t, {"incidentId": inc, "start": s, "end": e},
                                            lambda lid: {"lockout": has(lockoutId=lid), "cancelledReservations": cancelled}, check=True)
        l1 = lock(ts(WED, "10:00"), ts(WED, "11:00"), [])
        l2 = lock(ts(WED, "11:30"), ts(WED, "12:00"), [])
        l3 = lock(ts(WED, "23:00"), ts(THU, "01:00"), [])
        c.ok("createReservation", S, t, {"memberId": "ada", "machineId": "L2", "start": ts(WED, "11:00"), "end": ts(WED, "11:30")},
             lambda rid: {"reservation": has(reservationId=rid)}, check=True)
        c.ok("createReservation", S, t, {"memberId": "ada", "machineId": "L2", "start": ts(WED, "09:30"), "end": ts(WED, "10:00")},
             lambda rid: {"reservation": has(reservationId=rid)}, check=True)
        c.refuse("createReservation", S, t, {"memberId": "bob", "machineId": "L2", "start": ts(WED, "10:30"), "end": ts(WED, "11:00")}, "machine_locked")
        lk = lambda lid, s, e: f.lockout(lid, inc, "L2", s, e, [])
        c.ok("getMachineSchedule", S, t, {"machineId": "L2", "date": WED},
             has(lockouts=[lk(l1, ts(WED, "10:00"), ts(WED, "11:00")), lk(l2, ts(WED, "11:30"), ts(WED, "12:00")), lk(l3, ts(WED, "23:00"), ts(THU, "01:00"))]), check=True)
        c.ok("getMachineSchedule", S, t, {"machineId": "L2", "date": THU}, has(lockouts=[lk(l3, ts(WED, "23:00"), ts(THU, "01:00"))]), check=True)
    return [c.build()]


def p_time_booking_states(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-TIME-BOOKING-STATES", "boundaries")
    w.world(c)
    r1 = w.reserve(c, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "10:30"), T1)
    r2 = w.reserve(c, "bob", "L2", ts(TUE, "10:00"), ts(TUE, "10:30"), T1)
    r3 = w.reserve(c, "cy", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), T1)
    c.refuse("checkIn", S, ts(TUE, "09:59"), {"reservationId": r1}, "check_in_too_early")
    check_in(c, r1, ts(TUE, "10:00"), result={"reservation": has(status="checkedIn", checkedInAt=ts(TUE, "10:00"))}, check=True)
    c.ok("listMemberReservations", S, ts(TUE, "10:29"), {"memberId": "bob"}, has(reservations=[has(reservationId=r2, status="reserved")]), check=True)
    c.ok("listMemberReservations", S, ts(TUE, "10:30"), {"memberId": "bob"}, has(reservations=[has(reservationId=r2, status="missed")]), check=True)
    c.refuse("checkIn", S, ts(TUE, "10:30"), {"reservationId": r2}, "invalid_state")
    c.refuse("cancelReservation", S, ts(TUE, "10:30"), {"reservationId": r2}, "invalid_state")
    c.ok("createReservation", S, ts(TUE, "10:30"), {"memberId": "bob", "machineId": "L2", "start": ts(TUE, "10:30"), "end": ts(TUE, "11:00")},
         lambda rid: {"reservation": has(reservationId=rid)}, check=True)
    c.refuse("createReservation", S, ts(TUE, "10:31"), {"memberId": "bob", "machineId": "L1", "start": ts(TUE, "10:30"), "end": ts(TUE, "11:00")}, "starts_in_past")
    check_out(c, r1, ts(TUE, "10:40"))
    c.refuse("scheduleMaintenance", S, ts(TUE, "11:10"), {"machineId": "L1", "start": ts(TUE, "11:10"), "end": ts(TUE, "11:40"), "reason": "R"}, "reservation_conflict")
    c.ok("scheduleMaintenance", S, ts(TUE, "11:10"), {"machineId": "L2", "start": ts(TUE, "11:10"), "end": ts(TUE, "12:00"), "reason": "R"},
         lambda wid: {"window": has(windowId=wid)}, check=True)
    c.refuse("scheduleMaintenance", S, ts(TUE, "11:10"), {"machineId": "L2", "start": ts(TUE, "11:09"), "end": ts(TUE, "12:00"), "reason": "R"}, "starts_in_past")
    check_in(c, r3, ts(TUE, "11:10"), result={"reservation": has(status="checkedIn")}, check=True)
    a = c.ok("scheduleMaintenance", S, ts(TUE, "11:10"), {"machineId": "W1", "start": ts(TUE, "12:00"), "end": ts(TUE, "13:00"), "reason": "R"})
    b = c.ok("scheduleMaintenance", S, ts(TUE, "11:10"), {"machineId": "W1", "start": ts(TUE, "12:00"), "end": ts(TUE, "12:30"), "reason": "R"})
    c.ok("cancelMaintenance", S, ts(TUE, "11:59"), {"windowId": a}, lambda _: {"window": has(windowId=a, status="cancelled")}, check=True)
    c.refuse("cancelMaintenance", S, ts(TUE, "12:00"), {"windowId": b}, "invalid_state")
    c.refuse("cancelMaintenance", S, ts(TUE, "12:00"), {"windowId": a}, "invalid_state")
    if since(step, "C2"):
        inc = c.ok("reportIncident", S, ts(TUE, "12:00"), {"machineId": "L2", "severity": "high", "description": "D"})
        c.refuse("lockMachine", S, ts(TUE, "12:00"), {"incidentId": inc, "start": ts(TUE, "11:59"), "end": ts(TUE, "12:30")}, "starts_in_past")
        c.ok("lockMachine", S, ts(TUE, "12:00"), {"incidentId": inc, "start": ts(TUE, "12:00"), "end": ts(TUE, "12:30")},
             lambda lid: {"lockout": has(lockoutId=lid), "cancelledReservations": []}, check=True)
        late = w.reserve(c, "bob", "L1", ts(TUE, "13:00"), ts(TUE, "13:30"), ts(TUE, "12:00"))
        inc2 = c.ok("reportIncident", S, ts(TUE, "13:10"), {"machineId": "L1", "severity": "high", "description": "D"})
        c.ok("lockMachine", S, ts(TUE, "13:10"), {"incidentId": inc2, "start": ts(TUE, "13:10"), "end": ts(TUE, "14:00")},
             lambda lid: {"lockout": has(lockoutId=lid, cancelledReservationIds=[late]),
                          "cancelledReservations": [has(reservationId=late, status="cancelled")]}, check=True,
             cite=C(2, "New operation", "This includes a reservation whose start has already passed but which is not yet checked in and not yet missed."))
    return [c.build()]


def p_time_certification(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-TIME-CERTIFICATION", "validity")
    w.world(c)
    expiry = not since(step, "C6")
    first = w.reserve(c, "ada", "L1", "2027-03-01T10:00Z", "2027-03-01T10:30Z", T1, check=True)
    for member, start in (("bob", "2027-03-02T10:00Z"), ("cy", "2028-03-02T10:00Z")):
        args = {"memberId": member, "machineId": "L1", "start": start, "end": stamp(start, 30)}
        if expiry:
            c.refuse("createReservation", S, T1, args, "not_certified")
        else:
            c.ok("createReservation", S, T1, args, lambda rid: {"reservation": has(reservationId=rid)}, check=True)
    check_in(c, first, "2027-03-01T10:00Z", result={"reservation": has(status="checkedIn")}, check=True)
    check_out(c, first, "2027-03-01T10:05Z")
    noon = "2027-03-01T12:00Z"
    c.ok("enrollInCourse", S, noon, {"courseId": "laser-101", "memberId": "ada"})
    c.ok("completeCourse", S, noon, {"courseId": "laser-101", "memberId": "ada"},
         lambda _: {"enrollment": has(status="completed"),
                    "certification": f.certification(step, "ada", "laser", "2027-03-01", "2028-02-29", "valid")}, check=True,
         cite=W("4.4", "so `2027-03-01` plus 365 days is `2028-02-29`.") if expiry else
         C(6, "What changes", "completing a course still issues the certification with `issuedOn` the date of the request time"))
    c.ok("createReservation", S, noon, {"memberId": "ada", "machineId": "L1", "start": "2028-02-28T10:00Z", "end": "2028-02-28T10:30Z"},
         lambda rid: {"reservation": has(reservationId=rid)}, check=True)
    leap = {"memberId": "ada", "machineId": "L1", "start": "2028-02-29T10:00Z", "end": "2028-02-29T10:30Z"}
    if expiry:
        c.refuse("createReservation", S, noon, leap, "not_certified")
    else:
        c.ok("createReservation", S, noon, leap, lambda rid: {"reservation": has(reservationId=rid)}, check=True)
    bob = lambda status: {"memberId": "bob", "certifications": [f.certification(step, "bob", "laser", w.ISSUED, w.EXPIRES, status)]}
    c.ok("listCertifications", S, "2027-03-01T23:59Z", {"memberId": "bob"}, bob("valid"), check=True)
    c.ok("listCertifications", S, "2027-03-02T00:00Z", {"memberId": "bob"}, bob("expired" if expiry else "valid"), check=True)
    c.ok("revokeCertification", S, "2027-03-02T00:00Z", {"memberId": "bob", "familyId": "laser"},
         {"certification": f.certification(step, "bob", "laser", w.ISSUED, w.EXPIRES, "revoked")}, check=True,
         cite=W("7.2", "An expired certification can be revoked.") if expiry else C(6, "What changes", "refused with `not_certified` when the member has no certification for the family or it is already revoked, as before"))
    c.ok("listCertifications", S, "2027-03-02T00:00Z", {"memberId": "bob"}, bob("revoked"), check=True)
    return [c.build()]


def p_time_billing(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-TIME-BILLING", "dates")
    w.world(c)
    eve = "2026-03-31T23:59Z"
    apr = "2026-04-01T00:00Z"
    p1 = c.ok("purchaseMaterial", S, eve, {"memberId": "ada", "materialId": "acrylic", "quantity": 1})
    c.refuse("generateInvoices", S, eve, {"month": "2026-03"}, "month_not_ended")
    c.ok("purchaseMaterial", S, apr, {"memberId": "ada", "materialId": "plywood", "quantity": 1})
    march = generate(c, "2026-03", apr)
    c.ok("listInvoices", S, apr, {"memberId": "ada"}, {"memberId": "ada", "invoices": [
        has(invoiceId=march["ada"], issuedAt=apr, dueAt="2026-04-15T00:00Z", overdue=False, balanceCents={"$capture": "A"},
            lines=lines("material", [has(ref=p1, at=eve)]))]}, check=True,
         cite=W("11.1", "A charge belongs to the month of its `at`."))
    A = {"$ref": "A"}
    for at, overdue in (("2026-04-14T23:59Z", False), ("2026-04-15T00:00Z", True)):
        c.ok("listInvoices", S, at, {"memberId": "ada"}, {"memberId": "ada", "invoices": [has(invoiceId=march["ada"], overdue=overdue)]}, check=True)
        c.ok("getAccount", S, at, {"memberId": "ada"}, {"account": has(overdueBalanceCents=A if overdue else 0, openBalanceCents=A)}, check=True)
    due = "2026-04-15T00:00Z"
    c.ok("listInvoices", S, due, {"memberId": "bob"}, {"memberId": "bob", "invoices": [has(balanceCents={"$capture": "B"})]}, check=True)
    c.ok("recordPayment", S, due, {"invoiceId": march["bob"], "amountCents": {"$ref": "B"}})
    c.ok("listInvoices", S, due, {"memberId": "bob"}, {"memberId": "bob", "invoices": [has(status="paid", overdue=False)]}, check=True)
    block = since(step, "C5")
    edge, after = "2026-05-15T00:00Z", "2026-05-15T00:01Z"
    day = "2026-05-18"
    c.ok("createReservation", S, edge, {"memberId": "ada", "machineId": "L1", "start": ts(day, "10:00"), "end": ts(day, "10:30")},
         lambda rid: {"reservation": has(reservationId=rid)}, check=True,
         cite=C(5, "What changes", "At exactly `dueAt` plus 30 days the member is not yet blocked; one minute later the member is blocked.") if block else
         W("11.1", "Being overdue has no other effect."))
    c.ok("purchaseMaterial", S, edge, {"memberId": "ada", "materialId": "acrylic", "quantity": 1})
    if block:
        c.ok("getAccount", S, edge, {"memberId": "ada"}, {"account": has(blocked=False)}, check=True)
    attempts = [(S, {"memberId": "ada", "machineId": "L1", "start": ts(day, "11:00"), "end": ts(day, "11:30")}),
                ("ada", {"memberId": "ada", "machineId": "L1", "start": ts(day, "12:00"), "end": ts(day, "12:30")})]
    for actor, args in attempts:
        if block:
            c.refuse("createReservation", actor, after, args, "account_overdue")
        else:
            c.ok("createReservation", actor, after, args)
    if block:
        c.refuse("purchaseMaterial", "admin", after, {"memberId": "ada", "materialId": "acrylic", "quantity": 1}, "account_overdue",
                 cite=C(5, "What changes", "The block applies whoever acts: a member for themselves, staff or an admin."))
        c.ok("getAccount", S, after, {"memberId": "ada"}, {"account": has(blocked=True)}, check=True)
        c.ok("recordPayment", S, after, {"invoiceId": march["ada"], "amountCents": 100})
        c.ok("issueCredit", "admin", after, {"memberId": "ada", "amountCents": 100000, "reason": "R"})
        c.refuse("createReservation", S, after, attempts[0][1], "account_overdue",
                 cite=C(5, "What changes", "A partial payment that leaves a balance does not end the block."))
        c.ok("recordPayment", S, after, {"invoiceId": march["ada"], "amountCents": {"$calc": ["-", A, 100]}})
        c.ok("createReservation", S, after, attempts[0][1], lambda rid: {"reservation": has(reservationId=rid)}, check=True)
        c.ok("getAccount", S, after, {"memberId": "ada"}, {"account": has(blocked=False)}, check=True)
    else:
        c.ok("purchaseMaterial", "admin", after, {"memberId": "ada", "materialId": "acrylic", "quantity": 1})
        c.ok("recordPayment", S, after, {"invoiceId": march["ada"], "amountCents": 100},
             lambda pid: {"payment": has(appliedCents=100, creditedCents=0),
                          "invoice": has(paidCents=100, balanceCents={"$calc": ["-", A, 100]}, overdue=True),
                          "creditBalanceCents": 0}, check=True, cite=W("11.2", "Paying after the due date is allowed."))
    gen = "2026-05-25T10:00Z"
    generate(c, "2026-04", gen)
    c.ok("listInvoices", S, gen, {"memberId": "ada"}, {"memberId": "ada", "invoices": {"$filter": {"month": "2026-04"}, "items": [
        has(issuedAt=gen, dueAt="2026-06-08T10:00Z")]}}, check=True, cite=W("4.4", '"N days after timestamp t" is `t + N × 1440` minutes.'))
    return [c.build()]


def p_eligibility(step: str) -> list[dict[str, Any]]:
    c = probe(step, "P-ELIGIBILITY", "standing-and-certification")
    w.world(c)
    day, nxt = "2026-04-01", "2026-04-02"
    b1 = w.reserve(c, "bob", "L1", ts(day, "10:00"), ts(day, "10:30"), T1)
    b2 = w.reserve(c, "bob", "L1", ts(day, "11:00"), ts(day, "11:30"), T1)
    a3 = w.reserve(c, "ada", "L1", ts(day, "12:00"), ts(day, "12:30"), T1)
    c.ok("addCourse", S, T1, {"courseId": "wood-101", "familyId": "wood", "title": "Lathe basics", "feeCents": 0, "capacity": 5})
    c.ok("enrollInCourse", S, T1, {"courseId": "wood-101", "memberId": "bob"})
    march = generate(c, "2026-03", APR1)
    c.ok("setMemberStanding", S, APR1, {"memberId": "bob", "standing": "suspended"})
    c.refuse("createReservation", S, APR1, {"memberId": "bob", "machineId": "L2", "start": ts(day, "13:00"), "end": ts(day, "13:30")}, "member_suspended")
    c.refuse("enrollInCourse", S, APR1, {"courseId": "laser-101", "memberId": "bob"}, "member_suspended")
    c.refuse("purchaseMaterial", S, APR1, {"memberId": "bob", "materialId": "acrylic", "quantity": 1}, "member_suspended")
    c.ok("listMemberReservations", "bob", APR1, {"memberId": "bob"},
         has(reservations=[has(reservationId=b1, status="reserved"), has(reservationId=b2, status="reserved")]), check=True)
    c.ok("recordPayment", "bob", APR1, {"invoiceId": march["bob"], "amountCents": 100})
    c.ok("getMember", "bob", APR1, {"memberId": "bob"}, {"member": has(standing="suspended")}, check=True)
    c.ok("cancelReservation", "bob", APR1, {"reservationId": b2})
    c.ok("completeCourse", S, APR1, {"courseId": "wood-101", "memberId": "bob"}, cite=W("7.1", "Standing is not checked."))
    c.refuse("checkIn", S, ts(day, "10:00"), {"reservationId": b1}, "member_suspended")
    c.ok("setMemberStanding", S, ts(day, "10:01"), {"memberId": "bob", "standing": "good"})
    check_in(c, b1, ts(day, "10:01"))
    c.ok("setMemberStanding", S, ts(day, "10:05"), {"memberId": "bob", "standing": "suspended"})
    check_out(c, b1, ts(day, "10:10"), result={"reservation": has(checkedInAt=ts(day, "10:01"), meteredMinutes=9), "charge": has(quantity=9)},
              check=True, cite=[W("9.2", "A charge of 0 cents is still created. The reservation's `amountCents` is the charge's amount. Standing is not checked."),
                                W("9.2", "`meteredMinutes` is `checkedOutAt − checkedInAt` in minutes")])
    c.ok("revokeCertification", S, ts(day, "10:10"), {"memberId": "ada", "familyId": "laser"})
    c.ok("listMemberReservations", S, ts(day, "10:10"), {"memberId": "ada"}, has(reservations=[has(reservationId=a3, status="reserved")]), check=True)
    c.refuse("createReservation", S, ts(day, "10:10"), {"memberId": "ada", "machineId": "L2", "start": ts(nxt, "10:00"), "end": ts(nxt, "10:30")}, "not_certified")
    c.refuse("createReservation", S, ts(day, "10:10"), {"memberId": "cy", "machineId": "W1", "start": ts(nxt, "10:00"), "end": ts(nxt, "10:30")}, "not_certified")
    c.refuse("checkIn", S, ts(day, "12:00"), {"reservationId": a3}, "not_certified")
    c.ok("setMemberStanding", S, ts(day, "12:00"), {"memberId": "bob", "standing": "good"})
    c.ok("createReservation", S, ts(day, "12:00"), {"memberId": "bob", "machineId": "L2", "start": ts(nxt, "10:00"), "end": ts(nxt, "10:30")},
         lambda rid: {"reservation": has(reservationId=rid)}, check=True,
         cite=W("6.2", "Reinstating good standing removes these restrictions; all other rules still apply."))
    c.ok("enrollInCourse", S, ts(day, "12:00"), {"courseId": "laser-101", "memberId": "bob"})
    c.ok("purchaseMaterial", S, ts(day, "12:00"), {"memberId": "bob", "materialId": "acrylic", "quantity": 1})
    return [c.build()]


# --- Reporting -------------------------------------------------------------------------------------

def p_report_utilization(step: str) -> list[dict[str, Any]]:
    u = f.util_row
    empty = probe(step, "P-REPORT-UTILIZATION", "empty-and-range")
    empty.ok("utilizationReport", "admin", T1, {"from": MON, "to": "2026-03-09"},
             {"from": MON, "to": "2026-03-09", "machines": [], "totals": u(step, None, 0, 0, 0)}, check=True)
    for frm, to in ((TUE, TUE), (WED, TUE), ("2026-01-01", "2027-01-03")):
        empty.refuse("utilizationReport", "admin", T1, {"from": frm, "to": to}, "invalid_interval")
    empty.ok("utilizationReport", "admin", T1, {"from": "2026-01-01", "to": "2027-01-02"}, has(machines=[]), check=True)

    c = probe(step, "P-REPORT-UTILIZATION", "minutes")
    w.world(c)
    c8 = since(step, "C8")
    c.ok("scheduleMaintenance", S, T0, {"machineId": "L1", "start": ts(TUE, "08:00"), "end": ts(TUE, "10:00"), "reason": "R"})
    c.ok("scheduleMaintenance", S, T0, {"machineId": "L1", "start": ts(TUE, "09:30"), "end": ts(TUE, "10:30"), "reason": "R"})
    gone = c.ok("scheduleMaintenance", S, T0, {"machineId": "L1", "start": ts(TUE, "12:00"), "end": ts(TUE, "13:00"), "reason": "R"})
    c.ok("cancelMaintenance", S, T0, {"windowId": gone})
    c.ok("scheduleMaintenance", S, T0, {"machineId": "L1", "start": ts(TUE, "22:00"), "end": ts(TUE, "23:00"), "reason": "Night"})
    c.ok("scheduleMaintenance", S, T0, {"machineId": "L2", "start": ts(THU, "00:00"), "end": ts(THU, "19:30"), "reason": "Rebuild"})
    a = w.reserve(c, "ada", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), T0)
    b = w.reserve(c, "bob", "L1", ts(TUE, "20:00"), ts(TUE, "20:30"), T0)
    cy = w.reserve(c, "cy", "L1", ts(TUE, "15:00"), ts(TUE, "15:30"), T0)
    l2 = w.reserve(c, "bob", "L2", ts(TUE, "10:00"), ts(TUE, "10:30"), T0)
    sat = w.reserve(c, "ada", "L2", ts(SAT, "16:00"), ts(SAT, "16:30"), T0)
    thu = w.reserve(c, "ada", "L2", ts(THU, "20:00"), ts(THU, "20:30"), T0)
    l2_used = 21 if c8 else 18  # 18 x 100 / 720 = 2.5 -> 3; 21 x 100 / 840 = 2.5 -> 3 (half up)
    check_in(c, l2, ts(TUE, "10:00"))
    check_out(c, l2, stamp(ts(TUE, "10:00"), l2_used))
    check_in(c, a, ts(TUE, "11:04"))  # metered from check-in, not from the reservation start
    check_out(c, a, ts(TUE, "11:22"))
    check_in(c, cy, ts(TUE, "15:00"))
    check_in(c, b, ts(TUE, "20:00"))
    c.ok("addMachine", "admin", ts(TUE, "22:00"), {"machineId": "L9", "name": "Laser nine", "familyId": "laser", "rateCentsPerMinute": 9})
    check_out(c, b, ts(WED, "00:10"))
    day_open = 840 if c8 else 720
    l1_maint = 150 if c8 else 90
    rows = [u(step, "L1", day_open, l1_maint, 18 + 240), u(step, "L2", day_open, 0, l2_used), u(step, "L9", day_open, 0, 0), u(step, "W1", day_open, 0, 0)]
    tot = u(step, None, 4 * day_open, l1_maint, 258 + l2_used)
    c.ok("utilizationReport", S, ts(WED, "00:10"), {"from": TUE, "to": WED}, {"from": TUE, "to": WED, "machines": rows, "totals": tot}, check=True)
    rows = [u(step, "L1", day_open, 0, 10), u(step, "L2", day_open, 0, 0), u(step, "L9", day_open, 0, 0), u(step, "W1", day_open, 0, 0)]
    c.ok("utilizationReport", S, ts(WED, "00:10"), {"from": WED, "to": THU},
         {"from": WED, "to": THU, "machines": rows, "totals": u(step, None, 4 * day_open, 0, 10)}, check=True)
    check_in(c, thu, ts(THU, "20:00"))
    check_out(c, thu, ts(FRI, "02:00"))
    l2_thu = 690 if c8 else 630  # Thursday opening minutes before 19:30
    rows = [u(step, "L1", day_open, 0, 0), u(step, "L2", day_open, l2_thu, 240), u(step, "L9", day_open, 0, 0), u(step, "W1", day_open, 0, 0)]
    c.ok("utilizationReport", S, ts(FRI, "02:00"), {"from": THU, "to": FRI},
         {"from": THU, "to": FRI, "machines": rows, "totals": u(step, None, 4 * day_open, l2_thu, 240)}, check=True,
         cite=W("13", "It can exceed 100."))
    check_in(c, sat, ts(SAT, "16:00"))
    check_out(c, sat, ts(SUN, "00:30"))
    sun_open = 240 if c8 else 0
    rows = [u(step, "L1", sun_open, 0, 0), u(step, "L2", sun_open, 0, 30), u(step, "L9", sun_open, 0, 0), u(step, "W1", sun_open, 0, 0)]
    c.ok("utilizationReport", S, ts(SUN, "00:30"), {"from": SUN, "to": "2026-03-09"},
         {"from": SUN, "to": "2026-03-09", "machines": rows, "totals": u(step, None, 4 * sun_open, 0, 30)}, check=True)
    week = 4800 if c8 else 4080
    sat_used = 510
    l2_week = l2_used + 360 + sat_used
    rows = [u(step, "L1", week, l1_maint, 268), u(step, "L2", week, l2_thu, l2_week), u(step, "L9", week, 0, 0), u(step, "W1", week, 0, 0)]
    c.ok("utilizationReport", S, ts(SUN, "00:30"), {"from": MON, "to": "2026-03-09"},
         {"from": MON, "to": "2026-03-09", "machines": rows, "totals": u(step, None, 4 * week, l1_maint + l2_thu, 268 + l2_week)}, check=True)
    return [empty.build(), c.build()]


# --- Registry --------------------------------------------------------------------------------------

FAMILIES: list[tuple[str, str, list[dict[str, str]], Callable[[str], list[dict[str, Any]]]]] = [
    ("P-ENVELOPE", "W0", [W("2.2", "A request is a UTF-8 JSON object with exactly three members:"),
                          W("2.2", "two requests with the same `id` are two separate requests."),
                          W("2.4", "the line is not valid JSON, is empty, or is not a JSON object;"),
                          W("2.4", "The response's `id` is the request's `id` when the line is a JSON object whose `id` member is a nonempty string; otherwise it is `null`."),
                          W("2.4", "An invalid request changes nothing: no state, no audit entry, no identifier counter and no request-time order (section 4.4)."),
                          W("2.1", "Request lines up to 64 KiB must be accepted.")], p_envelope),
    ("P-ARGUMENTS", "W0", [W("2.2", "All arguments are required; no operation has optional arguments."),
                           W("2.5", "whose first character is a letter or digit"),
                           W("2.5", "It is stored and returned exactly as supplied, without trimming."),
                           W("2.5", "(`10.0` and `1e3` are not integers)"),
                           W("2.5", "A JSON string equal to one of the listed values, compared exactly."),
                           W("2.5", "An argument that is missing, `null`, of the wrong JSON type or outside its format or range is refused with `invalid_argument`."),
                           W("3.1", "every argument is present and valid by section 2.5")], p_arguments),
    ("P-CHECK-ORDER", "W0", [W("3.1", "The first check that fails decides the refusal and its code."),
                             W("3.1", "An argument that names a member must name a user whose role is `member`; the identifier of a staff or admin user does not name a member.")], p_check_order),
    ("P-RULE-ORDER", "W0", [W("7.1", "Refusals, in this order: `member_suspended` when the member is not in good standing; `already_enrolled`"),
                            W("8.2", "Refusals, in this order: `invalid_interval` when `end` is not after `start`; `starts_in_past`"),
                            W("9.1", "Refusals, in this order:"),
                            W("9.2", "Refusals, in this order: `invalid_state` when the reported status is not `reserved`"),
                            W("10", "Refusals, in this order: `member_suspended` when the member is not in good standing; `insufficient_stock`")], p_rule_order),
    ("P-IDENTIFIERS", "W0", [W("2.6", "Each kind has its own counter."), W("2.6", "Refused and invalid requests consume no number."),
                             W("2.6", "in decimal without leading zeros")], p_identifiers),
    ("P-NO-MUTATION", "W0", [W("3.1", "nothing else happens: no record is created or changed, no identifier number is consumed"),
                             C(1, "What changes", "Section 3.1, last paragraph, now reads: when a check fails, nothing else happens, except that a `forbidden` refusal appends its audit entry (section 4.2).")], p_no_mutation),
    ("P-AUTHZ-ROLE", "W0", [W("4.1", "**none:** the role may not use the operation."),
                            W("3.1", "the actor's role may use the operation at all"),
                            W("14", "This table is the permission matrix (section 4.1) and the audit target list (section 4.2)."),
                            C(4, "What changes", "For every other operation a steward's permission is \"none\", and the request is refused with `forbidden` at check 4 of section 3.1.")], p_authz_role),
    ("P-AUTHZ-OWN", "W0", [W("4.1", "**own:** the role may use the operation only for records that belong to the actor."),
                           W("4.1", "the operation has a `memberId` argument and it equals the actor's identifier;"),
                           W("4.1", "the operation has a `reservationId` argument and the reservation's member is the actor;"),
                           W("4.1", "the operation has an `invoiceId` argument and the invoice's member is the actor."),
                           W("3.1", "**Ownership:**")], p_authz_own),
    ("P-AUTHZ-ALL", "W0", [W("4.1", "**all:** the role may use the operation for any record."),
                           W("14", "This table is the permission matrix (section 4.1) and the audit target list (section 4.2)."),
                           C(2, "New operation", "Operation index row: `lockMachine` | Safety | mutation | member none | staff all | admin all"),
                           C(4, "What changes", "A steward has permission \"all\" for exactly these operations:")], p_authz_all),
    ("P-AUDIT-ACCEPTED", "W0", [W("4.2", "Every request to a mutation that is answered `ok` appends exactly one audit entry, even when the operation changes no data"),
                                W("14", "In the target column, a name is the value of that argument or new identifier;"),
                                C(1, "What changes", "every request to a mutation answered `ok` appends exactly one entry with outcome `accepted`"),
                                C(1, "What changes", "**Content of an accepted entry:** as before, plus `actorRole` and `outcome` `accepted`."),
                                C(2, "New operation", "The request writes exactly one audit entry, for `lockMachine`."),
                                C(4, "What changes", "a steward's accepted mutations are recorded like everyone's, with `actorRole` `steward`.")], p_audit_accepted),
    ("P-AUDIT-SILENT", "W0", [W("4.2", "Reads, refusals and invalid requests never append an entry."),
                              C(1, "What changes", "Reads answered `ok`, other refusals and invalid requests append nothing."),
                              C(1, "What is not recorded", "Refusals with any other code, including `invalid_argument`, `unknown_actor`, `time_regression`, `not_found` and every operation rule code.")], p_audit_silent),
    ("P-AUDIT-FORBIDDEN", "W0", [W("4.2", "Reads, refusals and invalid requests never append an entry."),
                                 C(1, "What changes", "Every request refused with `forbidden` appends one audit entry."),
                                 C(1, "What changes", "**Content of a refusal entry:**")], p_audit_forbidden),
    ("P-MONEY-ROUNDING", "W0", [W("4.3", "This is the exact value rounded half up to a whole cent. Rounding happens once per line. Totals are sums of rounded line amounts and are never rounded again."),
                                W("4.3", "37 minutes at 45 cents with a 10 % discount")], p_money_rounding),
    ("P-MONEY-USAGE", "W0", [W("9.2", "charges them at the rate captured at check-in, with the member's tier discount"),
                             W("9.2", "`discountPercent` the discount of the member's tier at check-out"),
                             W("8.1", "The rate is captured at check-in, so the new rate applies to check-ins after the change."),
                             W("6.2", "The usage discount applies only to usage charges, never to course or material charges."),
                             W("4.3", "A charge's amount is fixed when the charge is created."),
                             W("9.1", "There is no fee and no credit.")], p_money_usage),
    ("P-MONEY-COURSE", "W0", [W("7.1", "A course with fee 0 creates a charge of 0 cents."),
                              W("7.1", "`discountPercent` 0, `at` the request time")], p_money_course),
    ("P-MONEY-MATERIALS", "W0", [W("10", "A purchase is created with the material's current unit price"),
                                 W("10", "Stock never goes below 0."),
                                 C(3, "What changes", "Quantities of separate purchases, also by the same member or of the same material, are never added together."),
                                 C(3, "What changes", "The member's membership tier plays no part in it.")], p_money_materials),
    ("P-MONEY-INVOICE", "W0", [W("11.1", "every member whose `registeredAt` is before the first minute of the month after M receives one invoice, whatever the member's standing."),
                               W("11.1", "`ref` the member's current tier"),
                               W("11.1", "`creditAppliedCents` is the smaller of the member's credit balance and `totalCents`."),
                               W("6.2", "The fee is charged once for every invoiced month (section 11.1), without proration."),
                               W("6.2", "and still receives the monthly fee"),
                               W("11.1", "Months can be generated in any order.")], p_money_invoice),
    ("P-MONEY-CREDIT", "W0", [W("11.2", "`appliedCents` is the smaller of `amountCents` and the invoice's `balanceCents`"),
                              W("11.2", "it is never applied to an invoice that already exists"),
                              W("11.2", "A payment record is created with `memberId` the invoice's member (whoever acts)"),
                              W("11.2", "`openBalanceCents` is the sum of `balanceCents` over the member's invoices with status `open`;")], p_money_credit),
    ("P-MONEY-REVENUE", "W0", [W("13", "`paymentsCents`: the sum of `amountCents` of all payments whose `at` lies in the month, for any invoice, including the parts that became credit;"),
                               W("13", "Credit from overpayments is not included."),
                               C(2, "Changes to existing operations and rules", "`creditsIssuedCents` also includes lockout credits whose `lockMachine` request time lies in the month.")], p_money_revenue),
    ("P-MONEY-LOCKOUT", "C2", [C(2, "New operation", "the member's credit balance increases by 1000. This happens whatever the member's standing."),
                               C(2, "What stays as it is", "`cancelReservation` by a member, staff or admin gives no credit, as before.")], p_money_lockout),
    ("P-TIME-ORDER", "W0", [W("4.4", "The request time of every request must not be earlier than the request time of the most recent request answered `ok`; an equal time is allowed."),
                            W("4.4", "Refused and invalid requests do not move this point."),
                            C(1, "What changes", "the request-time order is still set only by requests answered `ok` (section 4.4)")], p_time_order),
    ("P-TIME-FORMAT", "W0", [W("2.5", "naming a valid UTC calendar date and time from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`"),
                             W("2.5", "`2026-02-29` is not valid; `2028-02-29` is."),
                             W("2.5", "A JSON string `YYYY-MM` from `2000-01` to `2099-12`."),
                             W("2.5", "there are no seconds")], p_time_format),
    ("P-TIME-HOURS", "W0", [W("4.4", "A reservation ending exactly at closing time is inside."), W("4.4", "Sunday has no opening period."),
                            W("9.1", "the slot `17:30`–`18:00` is inside opening hours and `17:30`–`18:30` is not"),
                            W("9.1", "or `null` on Sunday"),
                            C(8, "What changes", "so `getMachineSchedule` always returns an `openingHours` object and never `null`."),
                            C(8, "What changes", "| Monday to Thursday | 08:00 | 22:00 | 840 |"), C(8, "What changes", "| Friday | 08:00 | 20:00 | 720 |"),
                            C(8, "What changes", "| Saturday | 09:00 | 17:00 | 480 |"), C(8, "What changes", "| Sunday | 12:00 | 16:00 | 240 |")], p_time_hours),
    ("P-TIME-GRANULARITY", "W0", [W("4.4", "A reservation lasts at least one granularity step (30 minutes) and at most 240 minutes."),
                                  W("9.1", "`invalid_slot` when `start` or `end` is not on a 30-minute boundary or the length is outside 30 to 240 minutes;"),
                                  W("9.1", "on Tuesday `2026-03-03`, `10:15`–`11:00` is refused with `invalid_slot`."),
                                  C(8, "What changes", "`invalid_slot` now applies when `start` or `end` is not on a 15-minute boundary or the length is outside 15 to 240 minutes."),
                                  C(8, "What changes", "On Tuesday `2026-03-03`, `10:15`–`10:30` is accepted and `10:10`–`10:30` is refused with `invalid_slot`.")], p_time_granularity),
    ("P-TIME-INTERVALS", "W0", [W("4.4", "intervals that only touch, such as 10:00–11:00 and 11:00–12:00, do not overlap."),
                                W("9.1", "Only active reservations block a machine or a member."),
                                W("8.2", "A cancelled window no longer blocks reservations and no longer counts in schedules or reports."),
                                W("9.1", "the machine's scheduled windows that overlap the interval from `00:00` of that date to `00:00` of the next date."),
                                C(2, "Changes to existing operations and rules", "the machine's lockouts that overlap the interval from `00:00` of the date to `00:00` of the next date")], p_time_intervals),
    ("P-TIME-BOOKING-STATES", "W0", [W("9.1", "a stored `reserved` reservation whose `end` is at or before the request time is reported as `missed`"),
                                     W("9.2", "`check_in_too_early` when the request time is before `start`"),
                                     W("8.2", "`invalid_state` when the window is already cancelled or its start is at or before the request time."),
                                     C(2, "New operation", "This includes a reservation whose start has already passed but which is not yet checked in and not yet missed.")], p_time_booking_states),
    ("P-TIME-CERTIFICATION", "W0", [W("7.2", "It is not valid on its `expiresOn` date."),
                                    W("7.2", "`expiresOn` is 365 days after `issuedOn`"),
                                    W("7.2", "otherwise `expired` when the date of the request time is on or after `expiresOn`, otherwise `valid`"),
                                    C(6, "What changes", "a certification is valid on a date D when it is not revoked and `issuedOn ≤ D`."),
                                    C(6, "What changes", "The field `expiresOn` is removed and must not appear in any result.")], p_time_certification),
    ("P-TIME-BILLING", "W0", [W("11.1", "`issuedAt` is the request time; `dueAt` is 14 days after `issuedAt`."),
                              W("11.1", "at a request time when its status is `open` and the request time is at or after `dueAt`."),
                              W("11.1", "`month_not_ended` when the request time is before the first minute of the following month"),
                              C(5, "What changes", "that is, later than `dueAt` plus 43200 minutes."),
                              C(5, "What changes", "Credit is never applied to an existing invoice, so issuing credit does not end it either.")], p_time_billing),
    ("P-ELIGIBILITY", "W0", [W("6.2", "A suspended member cannot make reservations, check in, buy materials or enroll in courses. Suspension does not cancel existing reservations or enrollments."),
                             W("6.2", "A suspended member can still cancel reservations, check out, see their records and pay invoices"),
                             W("7.2", "Existing reservations stay, but check-in requires a valid certification (section 9.2)."),
                             W("9.1", "the member holds a certification for the machine's family that is valid on the reservation date")], p_eligibility),
    ("P-REPORT-UTILIZATION", "W0", [W("13", "Every machine gets the full count, regardless of when it was added."),
                                    W("13", "Overlapping windows count once."),
                                    W("13", "Opening hours do not limit this count. Reservations that are checked in but not checked out are not counted."),
                                    W("13", "rounded half up to an integer, or 0 when `availableMinutes` is 0. It can exceed 100."),
                                    W("13", "`invalid_interval` when `to` is not after `from` or the range is longer than 366 days."),
                                    W("13", "With no machines, all totals are 0."),
                                    C(8, "What changes", "the `openMinutes` of the utilization report, for every date, including dates before the change.")], p_report_utilization),
]

FAMILY_REFS = {name: refs for name, _, refs, _ in FAMILIES}


def build_probes(step: str) -> list[dict[str, Any]]:
    cases: list[dict[str, Any]] = []
    for name, first, _, make in FAMILIES:
        if since(step, first):
            cases.extend(make(step))
    return cases
