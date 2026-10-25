"""Product facts of the Makerspace requirements, per step, as plain data.

Everything here restates tools/comparison/phase-c/requirements/ (w0.md and the
change documents c1.md to c9.md); `cite` records where. Nothing here realizes
an operation: these tables only shape requests and expected records.
"""

from __future__ import annotations

from typing import Any

STEPS = ("W0", "C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8", "C9")


def since(step: str, first: str) -> bool:
    """Whether `step` includes the change `first` (the steps are cumulative)."""
    return STEPS.index(step) >= STEPS.index(first)


def cite(doc: str, section: str, text: str) -> dict[str, str]:
    """A neutral requirement reference: document, heading text and a verbatim fragment."""
    return {"doc": doc, "section": section, "text": text}


def W(section: str, text: str) -> dict[str, str]:
    return cite("w0.md", section, text)


def C(n: int, section: str, text: str) -> dict[str, str]:
    return cite(f"c{n}.md", section, text)


# w0 sentences a later change replaces, with the last step at which they hold.
VOIDED = {
    "nothing else happens: no record is created or changed, no identifier number is consumed": "W0",
    "Every request to a mutation that is answered `ok` appends exactly one audit entry, even when the operation changes no data": "W0",
    "Every request to a mutation that is answered `ok` appends exactly one audit entry": "W0",
    "Reads, refusals and invalid requests never append an entry.": "W0",
    "Registers a staff member or an admin.": "C3",
    "Refusals, in this order: `member_suspended` when the member is not in good standing; `insufficient_stock`": "C4",
    "In this delivery an incident is a record only: it does not block reservations or use.": "C1",
    "Being overdue has no other effect.": "C4",
    "It is not valid on its `expiresOn` date.": "C5",
    "`expiresOn` is 365 days after `issuedOn`": "C5",
    "otherwise `expired` when the date of the request time is on or after `expiresOn`, otherwise `valid`": "C5",
    "If the member already has a certification for that family, whether valid, expired or revoked, it is replaced by the newly issued one.": "C5",
    "An expired certification can be revoked.": "C5",
    "A certification whose `expiresOn` lies after 2099 is never reported `expired`.": "C5",
    "an `expiresOn` of `2100-12-31` (365 days after `2099-12-31`)": "C5",
    "Sunday has no opening period.": "C7",
    "the slot `17:30`–`18:00` is inside opening hours and `17:30`–`18:30` is not": "C7",
    "or `null` on Sunday": "C7",
    "A reservation lasts at least one granularity step (30 minutes) and at most 240 minutes.": "C7",
    "`invalid_slot` when `start` or `end` is not on a 30-minute boundary or the length is outside 30 to 240 minutes;": "C7",
    "on Tuesday `2026-03-03`, `10:15`–`11:00` is refused with `invalid_slot`.": "C7",
}


def scope_refs(refs: list[dict[str, str]], step: str) -> list[dict[str, str]]:
    """The references in force at `step`: a change's own sentences from that change on,
    w0 sentences until a change voids them, and w0 section 11 as c7 restates it from C7."""
    scoped = []
    for ref in refs:
        if ref["doc"] != "w0.md":
            if not since(step, "C" + ref["doc"][1]):
                continue
        else:
            last = VOIDED.get(ref["text"])
            if last is not None and last != STEPS[-1] and since(step, STEPS[STEPS.index(last) + 1]):
                continue
            if ref["section"].startswith("11") and since(step, "C7"):
                ref = cite("c7.md", "Replacement text for section 11", ref["text"])
        scoped.append(dict(ref))
    return scoped


# --- Argument types (w0 2.5) ----------------------------------------------------

ID = ("identifier",)
TEXT = ("text",)
TS = ("timestamp",)
DATE = ("date",)
MONTH = ("month",)


def INT(lo: int, hi: int) -> tuple:
    return ("integer", lo, hi)


def ENUM(*values: str) -> tuple:
    return ("enum", values)


TIER = ENUM("basic", "plus", "pro")

# --- Operation index (w0 14, c2, c4) --------------------------------------------
# name: (area, kind, member, staff, admin, audit target, arguments)
# Targets: ("arg", a) | ("args", a, b) | ("new",); arguments in listed order.

_W0_OPS: dict[str, tuple] = {
    "registerUser": ("Members", "mutation", "none", "none", "all", ("arg", "userId"),
                     [("userId", ID), ("name", TEXT), ("role", ENUM("staff", "admin"))]),
    "registerMember": ("Members", "mutation", "none", "all", "all", ("arg", "memberId"),
                       [("memberId", ID), ("name", TEXT), ("tier", TIER)]),
    "getMember": ("Members", "read", "own", "all", "all", None, [("memberId", ID)]),
    "changeTier": ("Members", "mutation", "none", "all", "all", ("arg", "memberId"),
                   [("memberId", ID), ("tier", TIER)]),
    "setMemberStanding": ("Members", "mutation", "none", "all", "all", ("arg", "memberId"),
                          [("memberId", ID), ("standing", ENUM("good", "suspended"))]),
    "addCourse": ("Training", "mutation", "none", "all", "all", ("arg", "courseId"),
                  [("courseId", ID), ("familyId", ID), ("title", TEXT),
                   ("feeCents", INT(0, 1000000)), ("capacity", INT(1, 1000))]),
    "enrollInCourse": ("Training", "mutation", "own", "all", "all", ("args", "courseId", "memberId"),
                       [("courseId", ID), ("memberId", ID)]),
    "completeCourse": ("Training", "mutation", "none", "all", "all", ("args", "courseId", "memberId"),
                       [("courseId", ID), ("memberId", ID)]),
    "revokeCertification": ("Training", "mutation", "none", "all", "all", ("args", "memberId", "familyId"),
                            [("memberId", ID), ("familyId", ID)]),
    "listCertifications": ("Training", "read", "own", "all", "all", None, [("memberId", ID)]),
    "addFamily": ("Equipment", "mutation", "none", "none", "all", ("arg", "familyId"),
                  [("familyId", ID), ("name", TEXT)]),
    "addMachine": ("Equipment", "mutation", "none", "none", "all", ("arg", "machineId"),
                   [("machineId", ID), ("name", TEXT), ("familyId", ID), ("rateCentsPerMinute", INT(1, 10000))]),
    "setMachineRate": ("Equipment", "mutation", "none", "none", "all", ("arg", "machineId"),
                       [("machineId", ID), ("rateCentsPerMinute", INT(1, 10000))]),
    "listMachines": ("Equipment", "read", "all", "all", "all", None, []),
    "scheduleMaintenance": ("Equipment", "mutation", "none", "all", "all", ("new",),
                            [("machineId", ID), ("start", TS), ("end", TS), ("reason", TEXT)]),
    "cancelMaintenance": ("Equipment", "mutation", "none", "all", "all", ("arg", "windowId"), [("windowId", ID)]),
    "createReservation": ("Bookings", "mutation", "own", "all", "all", ("new",),
                          [("memberId", ID), ("machineId", ID), ("start", TS), ("end", TS)]),
    "cancelReservation": ("Bookings", "mutation", "own", "all", "all", ("arg", "reservationId"), [("reservationId", ID)]),
    "checkIn": ("Bookings", "mutation", "own", "all", "all", ("arg", "reservationId"), [("reservationId", ID)]),
    "checkOut": ("Bookings", "mutation", "own", "all", "all", ("arg", "reservationId"), [("reservationId", ID)]),
    "listMemberReservations": ("Bookings", "read", "own", "all", "all", None, [("memberId", ID)]),
    "getMachineSchedule": ("Bookings", "read", "all", "all", "all", None, [("machineId", ID), ("date", DATE)]),
    "addMaterial": ("Materials", "mutation", "none", "all", "all", ("arg", "materialId"),
                    [("materialId", ID), ("name", TEXT), ("unitPriceCents", INT(1, 1000000))]),
    "restockMaterial": ("Materials", "mutation", "none", "all", "all", ("arg", "materialId"),
                        [("materialId", ID), ("quantity", INT(1, 100000))]),
    "purchaseMaterial": ("Materials", "mutation", "own", "all", "all", ("new",),
                         [("memberId", ID), ("materialId", ID), ("quantity", INT(1, 10000))]),
    "listMaterials": ("Materials", "read", "all", "all", "all", None, []),
    "generateInvoices": ("Billing", "mutation", "none", "all", "all", ("arg", "month"), [("month", MONTH)]),
    "listInvoices": ("Billing", "read", "own", "all", "all", None, [("memberId", ID)]),
    "recordPayment": ("Billing", "mutation", "own", "all", "all", ("new",),
                      [("invoiceId", ID), ("amountCents", INT(1, 100000000))]),
    "issueCredit": ("Billing", "mutation", "none", "none", "all", ("arg", "memberId"),
                    [("memberId", ID), ("amountCents", INT(1, 100000000)), ("reason", TEXT)]),
    "getAccount": ("Billing", "read", "own", "all", "all", None, [("memberId", ID)]),
    "reportIncident": ("Safety", "mutation", "none", "all", "all", ("new",),
                       [("machineId", ID), ("severity", ENUM("low", "medium", "high")), ("description", TEXT)]),
    "resolveIncident": ("Safety", "mutation", "none", "all", "all", ("arg", "incidentId"),
                        [("incidentId", ID), ("resolution", TEXT)]),
    "listIncidents": ("Safety", "read", "none", "all", "all", None, []),
    "utilizationReport": ("Reporting", "read", "none", "all", "all", None, [("from", DATE), ("to", DATE)]),
    "revenueReport": ("Reporting", "read", "none", "all", "all", None, [("month", MONTH)]),
    "listAuditEntries": ("Audit", "read", "none", "none", "all", None, []),
}

_LOCK_MACHINE = ("Safety", "mutation", "none", "all", "all", ("new",),
                 [("incidentId", ID), ("start", TS), ("end", TS)])

# c4: "A steward has permission "all" for exactly these operations".
STEWARD_ALL = ("scheduleMaintenance", "cancelMaintenance", "reportIncident", "resolveIncident",
               "listIncidents", "lockMachine", "listMachines", "getMachineSchedule", "listMaterials")

# Operations that create records with generated identifiers (w0 2.6, c2).
CREATES = {"createReservation": "res", "scheduleMaintenance": "mnt", "purchaseMaterial": "pur",
           "generateInvoices": "inv", "recordPayment": "pay", "reportIncident": "inc", "lockMachine": "lock"}


def ops(step: str) -> dict[str, dict[str, Any]]:
    """The operation index of a step, with a steward column from C4."""
    table = dict(_W0_OPS)
    if since(step, "C2"):
        table["lockMachine"] = _LOCK_MACHINE
    result = {}
    for name, (area, kind, member, staff, admin, target, args) in table.items():
        perms = {"member": member, "staff": staff, "admin": admin}
        if since(step, "C4"):
            perms["steward"] = "all" if name in STEWARD_ALL else "none"
        if name == "registerUser" and since(step, "C4"):
            args = [("userId", ID), ("name", TEXT), ("role", ENUM("staff", "admin", "steward"))]
        result[name] = {"area": area, "kind": kind, "perms": perms, "target": target,
                        "args": [("actor", ID), ("at", TS)] + list(args)}
    return result


def roles(step: str) -> tuple[str, ...]:
    return ("member", "staff", "admin", "steward") if since(step, "C4") else ("member", "staff", "admin")


def audit_target(op_info: dict[str, Any], args: dict[str, Any], new_id: str | None) -> str:
    kind = op_info["target"]
    if kind[0] == "arg":
        return args[kind[1]]
    if kind[0] == "args":
        return f"{args[kind[1]]}/{args[kind[2]]}"
    if new_id is None:
        raise ValueError("a 'new' audit target needs the generated identifier")
    return new_id


# --- Policies (w0 4.3, 4.4, 6.2; c3, c8) -----------------------------------------

FEES = {"basic": 2500, "plus": 4900, "pro": 9900}
USAGE_DISCOUNT = {"basic": 0, "plus": 10, "pro": 25}


def line_amount(quantity: int, unit: int, discount: int) -> int:
    """w0 4.3: (quantity x unitPriceCents x (100 - discountPercent) + 50) div 100."""
    return (quantity * unit * (100 - discount) + 50) // 100


def bulk_discount(step: str, quantity: int) -> int:
    """c3 table; 0 before C3."""
    if not since(step, "C3"):
        return 0
    return 0 if quantity <= 9 else 5 if quantity <= 24 else 12


def opening_hours(step: str) -> dict[int, tuple[str, str] | None]:
    """Weekday (Monday 0) to (opens, closes); w0 4.4 and from C8 the c8 table."""
    if since(step, "C8"):
        week = {d: ("08:00", "22:00") for d in range(4)}
        week.update({4: ("08:00", "20:00"), 5: ("09:00", "17:00"), 6: ("12:00", "16:00")})
        return week
    week = {d: ("09:00", "21:00") for d in range(5)}
    week.update({5: ("10:00", "18:00"), 6: None})
    return week


def granularity(step: str) -> int:
    return 15 if since(step, "C8") else 30


# --- Records (w0 5 and the changes) ----------------------------------------------

def user(uid: str, name: str, role: str) -> dict[str, Any]:
    return {"userId": uid, "name": name, "role": role}


def member(mid: str, name: str, tier: str, registered: str, standing: str = "good") -> dict[str, Any]:
    return {"memberId": mid, "name": name, "tier": tier, "standing": standing, "registeredAt": registered}


def family(fid: str, name: str) -> dict[str, Any]:
    return {"familyId": fid, "name": name}


def course(cid: str, fid: str, title: str, fee: int, capacity: int, enrolled: int) -> dict[str, Any]:
    return {"courseId": cid, "familyId": fid, "title": title, "feeCents": fee, "capacity": capacity,
            "enrolledCount": enrolled}


def enrollment(cid: str, mid: str, status: str, enrolled_at: str, completed_at: str | None = None) -> dict[str, Any]:
    return {"courseId": cid, "memberId": mid, "status": status, "enrolledAt": enrolled_at, "completedAt": completed_at}


def certification(step: str, mid: str, fid: str, issued: str, expires: str | None, status: str) -> dict[str, Any]:
    """c6 removes expiresOn and the status expired."""
    record = {"memberId": mid, "familyId": fid, "issuedOn": issued, "expiresOn": expires, "status": status}
    if since(step, "C6"):
        del record["expiresOn"]
    return record


def machine(mid: str, name: str, fid: str, rate: int) -> dict[str, Any]:
    return {"machineId": mid, "name": name, "familyId": fid, "rateCentsPerMinute": rate}


def window(wid: str, mid: str, start: str, end: str, reason: str, status: str = "scheduled") -> dict[str, Any]:
    return {"windowId": wid, "machineId": mid, "start": start, "end": end, "reason": reason, "status": status}


def reservation(rid: str, mid: str, machine_id: str, start: str, end: str, status: str = "reserved",
                checked_in: str | None = None, checked_out: str | None = None, rate: int | None = None,
                minutes: int | None = None, amount: int | None = None) -> dict[str, Any]:
    return {"reservationId": rid, "memberId": mid, "machineId": machine_id, "start": start, "end": end,
            "status": status, "checkedInAt": checked_in, "checkedOutAt": checked_out, "rateCentsPerMinute": rate,
            "meteredMinutes": minutes, "amountCents": amount}


def charge(kind: str, ref: str, quantity: int, unit: int, discount: int, at: str) -> dict[str, Any]:
    return {"kind": kind, "ref": ref, "quantity": quantity, "unitPriceCents": unit, "discountPercent": discount,
            "amountCents": line_amount(quantity, unit, discount), "at": at}


def material(mid: str, name: str, unit: int, stock: int) -> dict[str, Any]:
    return {"materialId": mid, "name": name, "unitPriceCents": unit, "stock": stock}


def purchase(pid: str, mid: str, material_id: str, quantity: int, unit: int, discount: int, at: str) -> dict[str, Any]:
    return {"purchaseId": pid, "memberId": mid, "materialId": material_id, "quantity": quantity,
            "unitPriceCents": unit, "discountPercent": discount, "amountCents": line_amount(quantity, unit, discount),
            "at": at}


def invoice(iid: str, mid: str, month: str, issued: str, due: str, lines: list[dict[str, Any]],
            credit: int = 0, paid: int = 0, overdue: bool = False) -> dict[str, Any]:
    total = sum(line["amountCents"] for line in lines)
    balance = total - credit - paid
    return {"invoiceId": iid, "memberId": mid, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": credit, "paidCents": paid, "balanceCents": balance,
            "status": "paid" if balance == 0 else "open", "overdue": overdue}


def payment(pid: str, iid: str, mid: str, amount: int, applied: int, at: str) -> dict[str, Any]:
    return {"paymentId": pid, "invoiceId": iid, "memberId": mid, "amountCents": amount, "appliedCents": applied,
            "creditedCents": amount - applied, "at": at}


def account(step: str, mid: str, credit: int = 0, open_: int = 0, overdue: int = 0, unbilled: int = 0,
            blocked: bool = False) -> dict[str, Any]:
    """c5 adds blocked."""
    record = {"memberId": mid, "creditBalanceCents": credit, "openBalanceCents": open_,
              "overdueBalanceCents": overdue, "unbilledCents": unbilled}
    if since(step, "C5"):
        record["blocked"] = blocked
    return record


def incident(iid: str, mid: str, severity: str, description: str, by: str, at: str,
             resolution: str | None = None, resolved_at: str | None = None) -> dict[str, Any]:
    return {"incidentId": iid, "machineId": mid, "severity": severity, "description": description,
            "reportedBy": by, "reportedAt": at, "status": "resolved" if resolution is not None else "open",
            "resolution": resolution, "resolvedAt": resolved_at}


def lockout(lid: str, iid: str, mid: str, start: str, end: str, cancelled: list[str]) -> dict[str, Any]:
    return {"lockoutId": lid, "incidentId": iid, "machineId": mid, "start": start, "end": end,
            "cancelledReservationIds": cancelled}


def audit_entry(step: str, seq: int, at: str, actor: str, role: str, op: str, target: str | None,
                outcome: str = "accepted") -> dict[str, Any]:
    """c1 adds actorRole and outcome."""
    entry: dict[str, Any] = {"seq": seq, "at": at, "actor": actor, "op": op, "target": target}
    if since(step, "C1"):
        entry["actorRole"] = role
        entry["outcome"] = outcome
    return entry


def schedule(step: str, machine_id: str, date: str, hours: tuple[str, str] | None,
             reservations: list[dict[str, Any]], maintenance: list[dict[str, Any]],
             lockouts: list[dict[str, Any]] | None = None) -> dict[str, Any]:
    """getMachineSchedule result; c2 adds lockouts."""
    result = {"machineId": machine_id, "date": date,
              "openingHours": None if hours is None else {"open": hours[0], "close": hours[1]},
              "reservations": reservations, "maintenance": maintenance}
    if since(step, "C2"):
        result["lockouts"] = list(lockouts or [])
    return result


def percent(used: int, available: int) -> int:
    """w0 13: (used x 100 x 2 + available) div (available x 2), or 0."""
    return 0 if available == 0 else (used * 100 * 2 + available) // (available * 2)


def util_row(step: str, machine_id: str | None, open_: int, maintenance: int, used: int,
             lockout_minutes: int = 0, blocked: int | None = None) -> dict[str, Any]:
    """One utilizationReport row (machine_id None for totals).

    `blocked` is the open minutes covered by maintenance or a lockout, each counted
    once (c2); it defaults to the maintenance minutes.
    """
    available = open_ - (maintenance if blocked is None else blocked)
    row: dict[str, Any] = {} if machine_id is None else {"machineId": machine_id}
    row.update({"openMinutes": open_, "maintenanceMinutes": maintenance, "availableMinutes": available,
                "usedMinutes": used, "utilizationPercent": percent(used, available)})
    if since(step, "C2"):
        row["lockoutMinutes"] = lockout_minutes
    return row
