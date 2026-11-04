"""The generated hidden cases, one per new operation: its arguments, its permissions and its referenced records.

Written from the parsed pages (facts.json), so they hold every operation to the form its page states. None of them
sends a request that should succeed: each step is an invalid request or a refusal, so nothing here depends on an
operation's effects, and the audit log at the end must hold the setup's entries alone.
"""
from __future__ import annotations

from typing import Any

from kit import Case, OPS

AT = "2026-06-01T10:00Z"
EARLIER = "2026-05-01T10:00Z"
# The 37 operations of the old program have their own cases.
OLD = {"registerUser", "registerMember", "getMember", "changeTier", "setMemberStanding", "addCourse", "enrollInCourse", "completeCourse",
       "revokeCertification", "listCertifications", "addFamily", "addMachine", "setMachineRate", "listMachines", "scheduleMaintenance",
       "cancelMaintenance", "createReservation", "cancelReservation", "checkIn", "checkOut", "listMemberReservations", "getMachineSchedule",
       "addMaterial", "restockMaterial", "purchaseMaterial", "listMaterials", "generateInvoices", "listInvoices", "recordPayment",
       "issueCredit", "getAccount", "reportIncident", "resolveIncident", "listIncidents", "utilizationReport", "revenueReport",
       "listAuditEntries"}
VALID = {"identifier": "g-none", "text": "some text", "timestamp": "2026-07-01T10:00Z", "date": "2026-07-01", "month": "2026-07", "flag": True}
WRONG_TYPE = {"identifier": 5, "text": 5, "timestamp": 5, "date": 5, "month": 5, "flag": "yes", "integer": "5", "enum": 5}
BAD_VALUE = {"identifier": "bad id!", "text": "   ", "timestamp": "2026-13-01T10:00Z", "date": "2026-02-30", "month": "2026-13"}


def valid(arg: dict[str, Any]) -> Any:
    if arg["type"] == "integer":
        return arg["min"]
    if arg["type"] == "enum":
        return arg["values"][0]
    return VALID[arg["type"]]


def actor_for(op: dict[str, Any]) -> str:
    """A role the operation allows without ownership, for the checks past the role check."""
    perms = op["permissions"]
    if perms.get("admin") == "all":
        return "admin"
    if perms.get("staff") == "all":
        return "g-staff"
    return "g-member"


def case_of(wire: str, op: dict[str, Any]) -> dict[str, Any]:
    c = Case(f"G-{wire}", "G", cite=wire)
    c.ok("registerUser", "admin", AT, {"userId": "g-staff", "name": "G Staff", "role": "staff"})
    c.ok("registerMember", "admin", AT, {"memberId": "g-member", "name": "G Member", "tier": "basic"})
    args = {a["name"]: valid(a) for a in op["args"]}
    actor = actor_for(op)
    for arg in op["args"]:
        name = arg["name"]
        without = {k: v for k, v in args.items() if k != name}
        c.refuse(wire, actor, AT, without, "invalid_argument")
        c.refuse(wire, actor, AT, {**args, name: None}, "invalid_argument")
        c.refuse(wire, actor, AT, {**args, name: WRONG_TYPE[arg["type"]]}, "invalid_argument")
        if arg["type"] == "integer":
            if arg["min"] > 0:
                c.refuse(wire, actor, AT, {**args, name: arg["min"] - 1}, "invalid_argument")
            c.refuse(wire, actor, AT, {**args, name: arg["max"] + 1}, "invalid_argument")
        elif arg["type"] == "enum":
            c.refuse(wire, actor, AT, {**args, name: "g-not-a-value"}, "invalid_argument")
        elif arg["type"] in BAD_VALUE:
            c.refuse(wire, actor, AT, {**args, name: BAD_VALUE[arg["type"]]}, "invalid_argument")
    c.invalid(wire, actor, AT, {**args, "gExtra": 1})
    c.refuse(wire, "bad id!", AT, args, "invalid_argument")
    c.refuse(wire, "g-nobody", AT, args, "unknown_actor")
    c.refuse(wire, actor, EARLIER, args, "time_regression")
    for role, user in (("member", "g-member"), ("staff", "g-staff"), ("admin", "admin")):
        if role not in op["permissions"]:
            c.refuse(wire, user, AT, args, "forbidden")
    refs = [a["name"] for a in op["args"] if "ref" in a]
    if refs:
        c.refuse(wire, actor, AT, args, "not_found")
    c.audit_log(AT)
    return c.build()


def build_cases() -> list[dict[str, Any]]:
    return [case_of(wire, op) for wire, op in sorted(OPS.items()) if wire not in OLD]
