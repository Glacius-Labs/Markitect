"""Hidden cases of the large Makerspace: the workshop floor (lockouts, inspections, parts, consumables, work orders,
escalations, compliance and what changed in bookings), written from the requirements document alone.

Every expected value below was computed by hand from the requirements. Where a helper checks arithmetic (`chg`, `inv`),
it only asserts the hand-written number against the money formula, so a slip of the author fails at build time.
"""
from __future__ import annotations

from kit import ANY, Case, has, ref, world  # noqa: F401

W = "2026-03-02T09:00Z"


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


class C(Case):
    """A case that also refuses to send a refusal back in time (it would answer time_regression)."""

    def refuse(self, op, actor, at, args, code, back: bool = False):  # type: ignore[override]
        if not back and self.last_ok is not None and at < self.last_ok:
            raise ValueError(f"{self.id}: refusal at {at} goes back from {self.last_ok}")
        super().refuse(op, actor, at, args, code)

    def invalid(self, op, actor, at, args):  # type: ignore[override]
        super().invalid(op, actor, at, args)


# ---- record builders (the members of each record, from the requirements) -------------------------------------------

def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def lock(lid, mach, source, reason, by, at, rel=None):
    return {"lockoutId": lid, "machineId": mach, "source": source, "reason": reason, "appliedBy": by,
            "appliedAt": at, "releasedAt": rel}


def plan(mach, days, last, due, failed=False):
    return {"machineId": mach, "intervalDays": days, "lastPassedOn": last, "nextDueOn": due, "failed": failed}


def chg(kind, ref_, qty, unit, disc, amt, at):
    assert (qty * unit * (100 - disc) + 50) // 100 == amt, (kind, ref_, qty, unit, disc, amt)
    return {"kind": kind, "ref": ref_, "quantity": qty, "unitPriceCents": unit, "discountPercent": disc,
            "amountCents": amt, "at": at}


def part(pid, name, cost, fam, stock):
    return {"partId": pid, "name": name, "unitCostCents": cost, "familyId": fam, "stock": stock}


def wo(wid, mach, summary, status="open", inc=None, lck=None, cost=0, by="sue", opened=None, closed=None, note=None):
    return {"workOrderId": wid, "machineId": mach, "summary": summary, "status": status, "incidentId": inc,
            "lockoutId": lck, "partsCostCents": cost, "openedBy": by, "openedAt": opened, "closedAt": closed, "note": note}


def incident(iid, mach, sev, desc, by, at, status="open", resolution=None, resolved=None):
    return {"incidentId": iid, "machineId": mach, "severity": sev, "description": desc, "reportedBy": by,
            "reportedAt": at, "status": status, "resolution": resolution, "resolvedAt": resolved}


def row(mach, mat, units):
    return {"machineId": mach, "materialId": mat, "unitsPerHour": units}


def mat(mid, name, price, stock):
    return {"materialId": mid, "name": name, "unitPriceCents": price, "stock": stock}


def notice(nid, mem, kind, subject, ref_, at):
    return {"noticeId": nid, "memberId": mem, "kind": kind, "subject": subject, "ref": ref_, "at": at, "acknowledgedAt": None}


def fam(fid, cert, cons, waiver, ppe):
    return {"familyId": fid, "certified": cert, "consent": cons, "waiverSigned": waiver, "ppeHeld": ppe}


def elig(mem, mach, date, blockers):
    return {"memberId": mem, "machineId": mach, "date": date, "eligible": not blockers, "blockers": blockers}


def invoice(iid, mem, month, issued, due, lines, total):
    assert sum(line["amountCents"] for line in lines) == total, (iid, total)
    return {"invoiceId": iid, "memberId": mem, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": 0, "paidCents": 0, "balanceCents": total, "status": "open",
            "overdue": False}


def account(mem, credit, open_, overdue, unbilled):
    return {"account": {"memberId": mem, "creditBalanceCents": credit, "openBalanceCents": open_,
                        "overdueBalanceCents": overdue, "unbilledCents": unbilled}}


def certify(c: Case, member: str, at: str) -> None:
    """Enroll and complete `laser-101` for a member (the course has two places; the world completed ada's)."""
    c.ok("enrollInCourse", "sam", at, {"courseId": "laser-101", "memberId": member})
    c.ok("completeCourse", "sam", at, {"courseId": "laser-101", "memberId": member})


# ---- B-lockouts ------------------------------------------------------------------------------------------------------

def b_lockouts_1() -> Case:
    c = C("B-lockouts-1", "B-lockouts", "lockouts page: apply lockout, list lockouts, release lockout, lockouts")
    world(c)
    a1, a2, a3 = t(2, "10:00"), t(2, "10:05"), t(2, "10:10")
    c.ok("applyLockout", "sam", a1, {"machineId": "L1", "reason": "belt"}, {"lockout": lock("lck-1", "L1", "manual", "belt", "sam", a1)}, new="lck-1")
    c.ok("applyLockout", "sue", a2, {"machineId": "L1", "reason": "jam"}, {"lockout": lock("lck-2", "L1", "manual", "jam", "sue", a2)}, new="lck-2")
    c.ok("applyLockout", "amy", a3, {"machineId": "L2", "reason": "dust"}, {"lockout": lock("lck-3", "L2", "manual", "dust", "amy", a3)}, new="lck-3")
    c.refuse("applyLockout", "sam", a3, {"machineId": "L9", "reason": "x"}, "not_found")
    c.refuse("applyLockout", "sam", a3, {"machineId": "L1", "reason": "   "}, "invalid_argument")
    c.refuse("applyLockout", "sam", a3, {"machineId": "L1"}, "invalid_argument")
    c.refuse("applyLockout", "ada", a3, {"machineId": "L1", "reason": "x"}, "forbidden")
    c.refuse("applyLockout", "ada", a3, {"machineId": "L9", "reason": "x"}, "forbidden")
    c.refuse("applyLockout", "nobody", a3, {"machineId": "L1", "reason": "x"}, "unknown_actor")
    c.refuse("listLockouts", "ada", a3, {"machineId": "L1"}, "forbidden")
    c.ok("listLockouts", "sam", a3, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "belt", "sam", a1), lock("lck-2", "L1", "manual", "jam", "sue", a2)]})
    c.ok("listLockouts", "sue", a3, {"machineId": "L2"}, {"machineId": "L2", "lockouts": [lock("lck-3", "L2", "manual", "dust", "amy", a3)]})
    c.ok("listLockouts", "amy", a3, {"machineId": "T1"}, {"machineId": "T1", "lockouts": []})
    r1 = t(2, "10:20")
    c.ok("releaseLockout", "sam", r1, {"lockoutId": "lck-1"}, {"lockout": lock("lck-1", "L1", "manual", "belt", "sam", a1, r1)})
    c.refuse("releaseLockout", "sam", t(2, "10:30"), {"lockoutId": "lck-1"}, "invalid_state")
    c.refuse("releaseLockout", "sam", t(2, "10:30"), {"lockoutId": "lck-9"}, "not_found")
    c.ok("listLockouts", "sam", t(2, "10:30"), {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "belt", "sam", a1, r1), lock("lck-2", "L1", "manual", "jam", "sue", a2)]})
    a4 = t(2, "10:30")
    c.ok("applyLockout", "sam", a4, {"machineId": "L1", "reason": "again"}, {"lockout": lock("lck-4", "L1", "manual", "again", "sam", a4)}, new="lck-4")
    c.ok("listLockouts", "sam", a4, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "belt", "sam", a1, r1), lock("lck-2", "L1", "manual", "jam", "sue", a2),
        lock("lck-4", "L1", "manual", "again", "sam", a4)]})
    c.audit_log(a4)
    return c


def b_lockouts_2() -> Case:
    c = C("B-lockouts-2", "B-lockouts", "release lockout: refusal order invalid_state, lockout_not_manual, not_lockout_holder; sources")
    world(c)
    p, f, w = t(2, "10:00"), t(2, "10:05"), t(2, "10:10")
    c.ok("setInspectionPlan", "sam", p, {"machineId": "L1", "intervalDays": 30})
    c.ok("recordInspection", "sam", f, {"machineId": "L1", "outcome": "failed", "note": "crack"},
         {"plan": plan("L1", 30, "2026-03-02", "2026-04-01", True)})
    c.ok("openWorkOrder", "sue", w, {"machineId": "L2", "summary": "Fix fan"}, new="wko-1")
    m = t(2, "10:15")
    c.ok("applyLockout", "sam", m, {"machineId": "T1", "reason": "oil"}, {"lockout": lock("lck-3", "T1", "manual", "oil", "sam", m)}, new="lck-3")
    s = t(2, "10:16")
    c.ok("applyLockout", "sue", s, {"machineId": "L1", "reason": "tag"}, {"lockout": lock("lck-4", "L1", "manual", "tag", "sue", s)}, new="lck-4")
    q = t(2, "10:20")
    c.refuse("releaseLockout", "sam", q, {"lockoutId": "lck-1"}, "lockout_not_manual")
    c.refuse("releaseLockout", "sue", q, {"lockoutId": "lck-1"}, "lockout_not_manual")
    c.refuse("releaseLockout", "amy", q, {"lockoutId": "lck-1"}, "lockout_not_manual")
    c.refuse("releaseLockout", "amy", q, {"lockoutId": "lck-2"}, "lockout_not_manual")
    c.refuse("releaseLockout", "sue", q, {"lockoutId": "lck-3"}, "not_lockout_holder")
    c.refuse("releaseLockout", "ada", q, {"lockoutId": "lck-3"}, "forbidden")
    c.refuse("releaseLockout", "ada", q, {"lockoutId": "lck-9"}, "forbidden")
    c.ok("listLockouts", "sam", q, {"machineId": "T1"}, {"machineId": "T1", "lockouts": [lock("lck-3", "T1", "manual", "oil", "sam", m)]})
    r = t(2, "10:30")
    c.ok("releaseLockout", "amy", r, {"lockoutId": "lck-3"}, {"lockout": lock("lck-3", "T1", "manual", "oil", "sam", m, r)})
    c.refuse("releaseLockout", "sue", r, {"lockoutId": "lck-3"}, "invalid_state")
    ps = t(2, "10:40")
    c.ok("recordInspection", "sam", ps, {"machineId": "L1", "outcome": "passed", "note": "ok"},
         {"plan": plan("L1", 30, "2026-03-02", "2026-04-01", False)})
    c.refuse("releaseLockout", "sue", ps, {"lockoutId": "lck-1"}, "invalid_state")
    c.ok("listLockouts", "sam", ps, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "inspection", "crack", "sam", f, ps), lock("lck-4", "L1", "manual", "tag", "sue", s)]})
    c.ok("listLockouts", "sam", ps, {"machineId": "L2"}, {"machineId": "L2", "lockouts": [
        lock("lck-2", "L2", "workOrder", "Fix fan", "sue", w)]})
    c.audit_log(ps)
    return c


# ---- B-inspections ---------------------------------------------------------------------------------------------------

def b_inspections_1() -> Case:
    c = C("B-inspections-1", "B-inspections", "inspection cycle: set plan, status, list due")
    world(c)
    n = t(2, "10:00")
    c.ok("addMachine", "admin", n, {"machineId": "L3", "name": "Laser three", "familyId": "laser", "rateCentsPerMinute": 20})
    c.refuse("getInspectionStatus", "sam", n, {"machineId": "L1"}, "no_plan")
    c.refuse("recordInspection", "sam", n, {"machineId": "L1", "outcome": "passed", "note": "ok"}, "no_plan")
    c.refuse("setInspectionPlan", "sam", n, {"machineId": "L1", "intervalDays": 0}, "invalid_argument")
    c.refuse("setInspectionPlan", "sam", n, {"machineId": "L1", "intervalDays": 3651}, "invalid_argument")
    c.refuse("setInspectionPlan", "ada", n, {"machineId": "L1", "intervalDays": 30}, "forbidden")
    c.refuse("setInspectionPlan", "sam", n, {"machineId": "L9", "intervalDays": 30}, "not_found")
    p1 = plan("L1", 30, "2026-03-02", "2026-04-01", False)
    p2 = plan("L2", 10, "2026-03-02", "2026-03-12", False)
    p3 = plan("T1", 3650, "2026-03-02", "2036-02-28", False)
    c.ok("setInspectionPlan", "sam", n, {"machineId": "L1", "intervalDays": 30}, {"plan": p1})
    c.ok("setInspectionPlan", "sue", n, {"machineId": "L2", "intervalDays": 10}, {"plan": p2})
    c.ok("setInspectionPlan", "amy", n, {"machineId": "T1", "intervalDays": 3650}, {"plan": p3})
    c.ok("getInspectionStatus", "sam", n, {"machineId": "L1"}, {"plan": p1, "due": False})
    c.ok("getInspectionStatus", "sam", t(11, "23:59"), {"machineId": "L2"}, {"plan": p2, "due": False})
    c.ok("getInspectionStatus", "sam", t(12, "00:00"), {"machineId": "L2"}, {"plan": p2, "due": True})
    c.ok("listInspectionsDue", "sam", t(12, "00:00"), {"date": "2026-03-11"}, {"date": "2026-03-11", "plans": []})
    c.ok("listInspectionsDue", "sam", t(12, "00:00"), {"date": "2026-03-12"}, {"date": "2026-03-12", "plans": [p2]})
    c.ok("listInspectionsDue", "amy", t(12, "00:00"), {"date": "2026-04-01"}, {"date": "2026-04-01", "plans": [p1, p2]})
    c.ok("listInspectionsDue", "amy", t(12, "00:00"), {"date": "2036-02-27"}, {"date": "2036-02-27", "plans": [p1, p2]})
    c.ok("listInspectionsDue", "amy", t(12, "00:00"), {"date": "2036-02-28"}, {"date": "2036-02-28", "plans": [p1, p2, p3]})
    c.refuse("listInspectionsDue", "ada", t(12, "00:00"), {"date": "2036-02-28"}, "forbidden")
    c.refuse("getInspectionStatus", "ada", t(12, "00:00"), {"machineId": "L2"}, "forbidden")
    # a changed interval moves nextDueOn and keeps lastPassedOn; the same interval changes nothing
    p2b = plan("L2", 20, "2026-03-02", "2026-03-22", False)
    c.ok("setInspectionPlan", "sam", t(14, "09:00"), {"machineId": "L2", "intervalDays": 20}, {"plan": p2b})
    c.ok("setInspectionPlan", "sam", t(14, "09:00"), {"machineId": "L2", "intervalDays": 20}, {"plan": p2b})
    c.ok("getInspectionStatus", "sam", t(14, "09:00"), {"machineId": "L2"}, {"plan": p2b, "due": False})
    c.ok("listInspectionsDue", "sam", t(14, "09:00"), {"date": "2026-03-21"}, {"date": "2026-03-21", "plans": []})
    c.ok("listInspectionsDue", "sam", t(14, "09:00"), {"date": "2026-03-22"}, {"date": "2026-03-22", "plans": [p2b]})
    c.refuse("getInspectionStatus", "sam", t(14, "09:00"), {"machineId": "L3"}, "no_plan")
    c.audit_log(t(14, "09:00"))
    return c


def b_inspections_2() -> Case:
    c = C("B-inspections-2", "B-inspections", "inspection gate, record inspection, lockout of source inspection")
    world(c)
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 29}, {"plan": plan("L1", 29, "2026-03-02", "2026-03-31", False)})
    c.ok("createReservation", "ada", W, {"memberId": "ada", "machineId": "L1", "start": t(30, "10:00"), "end": t(30, "11:00")},
         {"reservation": res("res-1", "ada", "L1", t(30, "10:00"), t(30, "11:00"))}, new="res-1")
    for hm in (("10:00", "11:00"), ("20:30", "21:00")):
        c.refuse("createReservation", "ada", W, {"memberId": "ada", "machineId": "L1", "start": t(31, hm[0]), "end": t(31, hm[1])}, "inspection_due")
    c.refuse("createReservation", "ada", W, {"memberId": "ada", "machineId": "L1", "start": "2026-04-01T10:00Z", "end": "2026-04-01T11:00Z"}, "inspection_due")
    c.ok("createReservation", "ada", W, {"memberId": "ada", "machineId": "L2", "start": t(31, "10:00"), "end": t(31, "11:00")},
         {"reservation": res("res-2", "ada", "L2", t(31, "10:00"), t(31, "11:00"))}, new="res-2")
    # a shorter interval makes the booked day due; check-in is not gated on the date
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 20}, {"plan": plan("L1", 20, "2026-03-02", "2026-03-22", False)})
    c.ok("checkIn", "ada", t(30, "10:00"), {"reservationId": "res-1"},
         {"reservation": res("res-1", "ada", "L1", t(30, "10:00"), t(30, "11:00"), "checkedIn", cin=t(30, "10:00"), rate=45)})
    c.ok("recordInspection", "sam", t(30, "10:30"), {"machineId": "L1", "outcome": "passed", "note": "ok"},
         {"plan": plan("L1", 20, "2026-03-30", "2026-04-19", False)})
    sat = {"memberId": "ada", "machineId": "L1"}
    c.ok("createReservation", "ada", t(30, "10:30"), {**sat, "start": "2026-04-18T10:00Z", "end": "2026-04-18T11:00Z"},
         {"reservation": res("res-3", "ada", "L1", "2026-04-18T10:00Z", "2026-04-18T11:00Z")}, new="res-3")
    c.refuse("createReservation", "ada", t(30, "10:30"), {**sat, "start": "2026-04-19T10:00Z", "end": "2026-04-19T11:00Z"}, "outside_opening_hours")
    c.refuse("createReservation", "ada", t(30, "10:30"), {**sat, "start": "2026-04-20T10:00Z", "end": "2026-04-20T11:00Z"}, "inspection_due")
    # a failure: failed flag, one inspection lockout, plan dates stay
    f1 = t(30, "11:00")
    c.ok("recordInspection", "sue", f1, {"machineId": "L1", "outcome": "failed", "note": "cracked"},
         {"plan": plan("L1", 20, "2026-03-30", "2026-04-19", True)})
    c.ok("getInspectionStatus", "sam", f1, {"machineId": "L1"}, {"plan": plan("L1", 20, "2026-03-30", "2026-04-19", True), "due": False})
    c.ok("listLockouts", "sam", f1, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [lock("lck-1", "L1", "inspection", "cracked", "sue", f1)]})
    c.ok("setInspectionPlan", "sam", f1, {"machineId": "L1", "intervalDays": 40}, {"plan": plan("L1", 40, "2026-03-30", "2026-05-09", True)})
    c.ok("listLockouts", "sam", f1, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [lock("lck-1", "L1", "inspection", "cracked", "sue", f1)]})
    p2 = t(30, "12:00")
    c.ok("recordInspection", "sam", p2, {"machineId": "L1", "outcome": "passed", "note": "fixed"},
         {"plan": plan("L1", 40, "2026-03-30", "2026-05-09", False)})
    c.ok("listLockouts", "sam", p2, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [lock("lck-1", "L1", "inspection", "cracked", "sue", f1, p2)]})
    c.audit_log(p2)
    return c


# ---- B-parts ---------------------------------------------------------------------------------------------------------

def b_parts_1() -> Case:
    c = C("B-parts-1", "B-parts", "parts page: add part, restock part, list parts (order by partId)")
    world(c)
    n = t(2, "10:00")
    c.ok("addPart", "sam", n, {"partId": "P-2", "name": "Belt", "unitCostCents": 1200, "familyId": "laser"}, {"part": part("P-2", "Belt", 1200, "laser", 0)})
    c.ok("addPart", "amy", n, {"partId": "P-10", "name": "Lens", "unitCostCents": 3500, "familyId": "laser"}, {"part": part("P-10", "Lens", 3500, "laser", 0)})
    c.ok("addPart", "sue", n, {"partId": "a-chuck", "name": "Chuck", "unitCostCents": 800, "familyId": "lathe"}, {"part": part("a-chuck", "Chuck", 800, "lathe", 0)})
    c.ok("addPart", "sam", n, {"partId": "P-3", "name": "Gold tube", "unitCostCents": 1000000, "familyId": "laser"}, {"part": part("P-3", "Gold tube", 1000000, "laser", 0)})
    c.refuse("addPart", "sam", n, {"partId": "P-2", "name": "Other", "unitCostCents": 1, "familyId": "lathe"}, "already_exists")
    c.refuse("addPart", "sam", n, {"partId": "P-2", "name": "Other", "unitCostCents": 1, "familyId": "ghost"}, "not_found")
    c.refuse("addPart", "sam", n, {"partId": "P-4", "name": "Free", "unitCostCents": 0, "familyId": "laser"}, "invalid_argument")
    c.refuse("addPart", "sam", n, {"partId": "P-4", "name": "Dear", "unitCostCents": 1000001, "familyId": "laser"}, "invalid_argument")
    c.ok("listParts", "sam", n, {}, {"parts": [
        part("P-10", "Lens", 3500, "laser", 0), part("P-2", "Belt", 1200, "laser", 0), part("P-3", "Gold tube", 1000000, "laser", 0),
        part("a-chuck", "Chuck", 800, "lathe", 0)]})
    r = t(2, "11:00")
    c.ok("restockPart", "sam", r, {"partId": "P-2", "quantity": 5}, {"part": part("P-2", "Belt", 1200, "laser", 5)})
    c.ok("restockPart", "sue", r, {"partId": "P-2", "quantity": 100000}, {"part": part("P-2", "Belt", 1200, "laser", 100005)})
    c.ok("restockPart", "amy", r, {"partId": "a-chuck", "quantity": 1}, {"part": part("a-chuck", "Chuck", 800, "lathe", 1)})
    c.ok("listParts", "amy", r, {}, {"parts": [
        part("P-10", "Lens", 3500, "laser", 0), part("P-2", "Belt", 1200, "laser", 100005), part("P-3", "Gold tube", 1000000, "laser", 0),
        part("a-chuck", "Chuck", 800, "lathe", 1)]})
    c.audit_log(r)
    return c


def b_parts_2() -> Case:
    c = C("B-parts-2", "B-parts", "parts: permissions, argument validation and order of checks")
    world(c)
    n = t(2, "10:00")
    c.refuse("addPart", "ada", n, {"partId": "P-1", "name": "Belt", "unitCostCents": 100, "familyId": "laser"}, "forbidden")
    c.refuse("addPart", "ada", n, {"partId": "P-1", "name": "Belt", "unitCostCents": 100, "familyId": "ghost"}, "forbidden")
    c.refuse("listParts", "ada", n, {}, "forbidden")
    c.refuse("addPart", "ghost", n, {"partId": "P-1", "name": "Belt", "unitCostCents": 100, "familyId": "laser"}, "unknown_actor")
    c.refuse("addPart", "ghost", n, {"partId": "P-1", "name": "Belt", "unitCostCents": "100", "familyId": "laser"}, "invalid_argument")
    c.refuse("addPart", "sam", n, {"partId": "P-1", "name": "   ", "unitCostCents": 100, "familyId": "laser"}, "invalid_argument")
    c.refuse("addPart", "sam", n, {"partId": "P-1", "name": None, "unitCostCents": 100, "familyId": "laser"}, "invalid_argument")
    c.refuse("addPart", "sam", n, {"partId": "bad id", "name": "Belt", "unitCostCents": 100, "familyId": "laser"}, "invalid_argument")
    c.refuse("addPart", "sam", n, {"partId": "P-1", "name": "Belt", "unitCostCents": 12.5, "familyId": "laser"}, "invalid_argument")
    c.refuse("addPart", "sam", n, {"partId": "P-1", "name": "Belt", "unitCostCents": -5, "familyId": "laser"}, "invalid_argument")
    c.refuse("addPart", "sam", n, {"partId": "P-1", "name": "Belt", "familyId": "laser"}, "invalid_argument")
    c.invalid("addPart", "sam", n, {"partId": "P-1", "name": "Belt", "unitCostCents": 100, "familyId": "laser", "stock": 5})
    c.invalid("addPart", "sam", n, {"partid": "P-1", "name": "Belt", "unitCostCents": 100, "familyId": "laser"})
    c.ok("addPart", "sam", n, {"partId": "P-1", "name": "Belt", "unitCostCents": 100, "familyId": "laser"}, {"part": part("P-1", "Belt", 100, "laser", 0)})
    c.refuse("restockPart", "sam", n, {"partId": "P-1", "quantity": 0}, "invalid_argument")
    c.refuse("restockPart", "sam", n, {"partId": "P-1", "quantity": 100001}, "invalid_argument")
    c.refuse("restockPart", "sam", n, {"partId": "P-1", "quantity": "5"}, "invalid_argument")
    c.refuse("restockPart", "sam", n, {"partId": "P-1", "quantity": 5.0}, "invalid_argument")
    c.refuse("restockPart", "sam", n, {"partId": "P-1"}, "invalid_argument")
    c.refuse("restockPart", "ghost", n, {"partId": "P-1", "quantity": 0}, "invalid_argument")
    c.refuse("restockPart", "sam", n, {"partId": "P-9", "quantity": 1}, "not_found")
    c.refuse("restockPart", "ada", n, {"partId": "P-1", "quantity": 1}, "forbidden")
    c.refuse("restockPart", "ada", n, {"partId": "P-9", "quantity": 1}, "forbidden")
    c.ok("restockPart", "sam", n, {"partId": "P-1", "quantity": 7}, {"part": part("P-1", "Belt", 100, "laser", 7)})
    c.refuse("restockPart", "ada", t(2, "09:30"), {"partId": "P-1", "quantity": 1}, "time_regression", back=True)
    c.ok("listParts", "sam", n, {}, {"parts": [part("P-1", "Belt", 100, "laser", 7)]})
    c.audit_log(n)
    return c


# ---- B-consumables ---------------------------------------------------------------------------------------------------

def b_consumables_1() -> Case:
    c = C("B-consumables-1", "B-consumables", "consumables page: set, list, clear consumption rows")
    world(c)
    n = t(2, "10:00")
    c.ok("addMaterial", "sam", n, {"materialId": "acr", "name": "Acrylic", "unitPriceCents": 30})
    c.ok("setConsumption", "sam", n, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 6}, {"row": row("L1", "ply", 6)})
    c.ok("setConsumption", "sue", n, {"machineId": "L1", "materialId": "acr", "unitsPerHour": 25}, {"row": row("L1", "acr", 25)})
    c.ok("setConsumption", "amy", n, {"machineId": "L2", "materialId": "ply", "unitsPerHour": 1000}, {"row": row("L2", "ply", 1000)})
    c.ok("listConsumption", "sam", n, {"machineId": "L1"}, {"machineId": "L1", "rows": [row("L1", "acr", 25), row("L1", "ply", 6)]})
    c.ok("listConsumption", "sam", n, {"machineId": "L2"}, {"machineId": "L2", "rows": [row("L2", "ply", 1000)]})
    c.ok("listConsumption", "sam", n, {"machineId": "T1"}, {"machineId": "T1", "rows": []})
    c.ok("setConsumption", "sam", n, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 8}, {"row": row("L1", "ply", 8)})
    c.ok("setConsumption", "sam", n, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 8}, {"row": row("L1", "ply", 8)})
    c.ok("listConsumption", "amy", n, {"machineId": "L1"}, {"machineId": "L1", "rows": [row("L1", "acr", 25), row("L1", "ply", 8)]})
    c.refuse("setConsumption", "sam", n, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 0}, "invalid_argument")
    c.refuse("setConsumption", "sam", n, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 1001}, "invalid_argument")
    c.refuse("setConsumption", "sam", n, {"machineId": "L9", "materialId": "ply", "unitsPerHour": 5}, "not_found")
    c.refuse("setConsumption", "sam", n, {"machineId": "L1", "materialId": "zzz", "unitsPerHour": 5}, "not_found")
    c.refuse("setConsumption", "ada", n, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 5}, "forbidden")
    c.refuse("listConsumption", "ada", n, {"machineId": "L1"}, "forbidden")
    c.refuse("clearConsumption", "sam", n, {"machineId": "T1", "materialId": "ply"}, "no_consumption")
    c.refuse("clearConsumption", "sam", n, {"machineId": "L2", "materialId": "acr"}, "no_consumption")
    c.refuse("clearConsumption", "sam", n, {"machineId": "T1", "materialId": "zzz"}, "not_found")
    c.ok("clearConsumption", "sam", n, {"machineId": "L1", "materialId": "ply"}, {"machineId": "L1", "materialId": "ply"})
    c.refuse("clearConsumption", "sam", n, {"machineId": "L1", "materialId": "ply"}, "no_consumption")
    c.ok("listConsumption", "sam", n, {"machineId": "L1"}, {"machineId": "L1", "rows": [row("L1", "acr", 25)]})
    c.audit_log(n)
    return c


def b_consumables_2() -> Case:
    c = C("B-consumables-2", "B-consumables", "consumption: units rounded up, drawn is at most the stock, 0 minutes, ascending materialId")
    world(c)
    c.ok("addMaterial", "sam", W, {"materialId": "acr", "name": "Acrylic", "unitPriceCents": 30})
    c.ok("restockMaterial", "sam", W, {"materialId": "acr", "quantity": 50})
    c.ok("addMaterial", "sam", W, {"materialId": "glue", "name": "Glue", "unitPriceCents": 10})
    c.ok("restockMaterial", "sam", W, {"materialId": "glue", "quantity": 3})
    c.ok("setConsumption", "sam", W, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 6})
    c.ok("setConsumption", "sam", W, {"machineId": "L1", "materialId": "glue", "unitsPerHour": 60})
    c.ok("setConsumption", "sam", W, {"machineId": "L1", "materialId": "acr", "unitsPerHour": 25})
    slots = [("res-1", "10:00", "11:00"), ("res-2", "12:00", "13:00"), ("res-3", "14:00", "15:00"), ("res-4", "15:00", "16:00")]
    for rid, s, e in slots:
        c.ok("createReservation", "ada", W, {"memberId": "ada", "machineId": "L1", "start": t(3, s), "end": t(3, e)},
             {"reservation": res(rid, "ada", "L1", t(3, s), t(3, e))}, new=rid)
    # 60 minutes: acr 25 units, glue 60 units but stock 3, ply 6 units
    c.ok("checkIn", "ada", t(3, "10:00"), {"reservationId": "res-1"})
    c.ok("checkOut", "ada", t(3, "11:00"), {"reservationId": "res-1"}, {
        "reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "completed", t(3, "10:00"), t(3, "11:00"), 45, 60, 2430),
        "charge": chg("usage", "res-1", 60, 45, 10, 2430, t(3, "11:00"))})
    # 1 minute: acr 1 unit, glue stock is 0, ply 1 unit
    c.ok("checkIn", "ada", t(3, "12:00"), {"reservationId": "res-2"})
    c.ok("checkOut", "ada", t(3, "12:01"), {"reservationId": "res-2"}, {
        "reservation": res("res-2", "ada", "L1", t(3, "12:00"), t(3, "13:00"), "completed", t(3, "12:00"), t(3, "12:01"), 45, 1, 41),
        "charge": chg("usage", "res-2", 1, 45, 10, 41, t(3, "12:01"))})
    # 0 minutes: a usage charge of 0 cents, no material is drawn
    c.ok("checkIn", "ada", t(3, "14:00"), {"reservationId": "res-3"})
    c.ok("checkOut", "ada", t(3, "14:00"), {"reservationId": "res-3"}, {
        "reservation": res("res-3", "ada", "L1", t(3, "14:00"), t(3, "15:00"), "completed", t(3, "14:00"), t(3, "14:00"), 45, 0, 0),
        "charge": chg("usage", "res-3", 0, 45, 10, 0, t(3, "14:00"))})
    # 11 minutes: acr (11 x 25 + 59) div 60 = 5, ply (11 x 6 + 59) div 60 = 2
    c.ok("checkIn", "ada", t(3, "15:00"), {"reservationId": "res-4"})
    c.ok("checkOut", "ada", t(3, "15:11"), {"reservationId": "res-4"}, {
        "reservation": res("res-4", "ada", "L1", t(3, "15:00"), t(3, "16:00"), "completed", t(3, "15:00"), t(3, "15:11"), 45, 11, 446),
        "charge": chg("usage", "res-4", 11, 45, 10, 446, t(3, "15:11"))})
    c.ok("listMaterials", "ada", t(3, "15:11"), {}, {"materials": [
        mat("acr", "Acrylic", 30, 19), mat("glue", "Glue", 10, 0), mat("ply", "Plywood", 150, 91)]})
    # unbilled: course 5000 + (2430 + 750 + 30 + 900) + (41 + 30 + 150) + 0 + (446 + 150 + 300) = 10227
    c.ok("getAccount", "ada", t(3, "15:11"), {"memberId": "ada"}, account("ada", 0, 0, 0, 10227))
    c.audit_log(t(3, "15:11"))
    return c


# ---- B-workorders ----------------------------------------------------------------------------------------------------

def b_workorders_1() -> Case:
    c = C("B-workorders-1", "B-workorders", "work orders: open, link incident, use parts, close; refusal order")
    world(c)
    c.ok("addPart", "sam", W, {"partId": "belt", "name": "Belt", "unitCostCents": 1200, "familyId": "laser"})
    c.ok("restockPart", "sam", W, {"partId": "belt", "quantity": 5})
    c.ok("addPart", "sam", W, {"partId": "chuck", "name": "Chuck", "unitCostCents": 800, "familyId": "lathe"})
    c.ok("restockPart", "sam", W, {"partId": "chuck", "quantity": 3})
    c.ok("reportIncident", "sam", W, {"machineId": "L1", "severity": "high", "description": "sparks"}, new="inc-1")
    c.ok("reportIncident", "sam", W, {"machineId": "L2", "severity": "low", "description": "smell"}, new="inc-2")
    o = t(2, "10:00")
    c.refuse("openWorkOrder", "ada", o, {"machineId": "L1", "summary": "Replace belt"}, "forbidden")
    c.refuse("openWorkOrder", "sue", o, {"machineId": "L9", "summary": "Replace belt"}, "not_found")
    c.refuse("openWorkOrder", "sue", o, {"machineId": "L1", "summary": " "}, "invalid_argument")
    c.ok("openWorkOrder", "sue", o, {"machineId": "L1", "summary": "Replace belt"}, {
        "workOrder": wo("wko-1", "L1", "Replace belt", lck="lck-1", by="sue", opened=o)}, new="wko-1")
    k = t(2, "10:10")
    c.refuse("linkIncident", "sue", k, {"workOrderId": "wko-1", "incidentId": "inc-2"}, "incident_other_machine")
    c.refuse("linkIncident", "sue", k, {"workOrderId": "wko-1", "incidentId": "inc-9"}, "not_found")
    c.refuse("linkIncident", "sue", k, {"workOrderId": "wko-9", "incidentId": "inc-1"}, "not_found")
    c.ok("linkIncident", "sue", k, {"workOrderId": "wko-1", "incidentId": "inc-1"}, {
        "workOrder": wo("wko-1", "L1", "Replace belt", inc="inc-1", lck="lck-1", by="sue", opened=o)})
    c.refuse("linkIncident", "sue", k, {"workOrderId": "wko-1", "incidentId": "inc-1"}, "already_linked")
    c.refuse("linkIncident", "sue", k, {"workOrderId": "wko-1", "incidentId": "inc-2"}, "incident_other_machine")
    u = t(2, "10:20")
    c.refuse("useParts", "sue", u, {"workOrderId": "wko-1", "partId": "chuck", "quantity": 4}, "part_wrong_family")
    c.refuse("useParts", "sue", u, {"workOrderId": "wko-1", "partId": "belt", "quantity": 6}, "insufficient_parts")
    c.refuse("useParts", "sue", u, {"workOrderId": "wko-1", "partId": "belt", "quantity": 0}, "invalid_argument")
    c.refuse("useParts", "sue", u, {"workOrderId": "wko-1", "partId": "belt", "quantity": 1001}, "invalid_argument")
    c.refuse("useParts", "sue", u, {"workOrderId": "wko-1", "partId": "nope", "quantity": 1}, "not_found")
    c.ok("useParts", "sue", u, {"workOrderId": "wko-1", "partId": "belt", "quantity": 2}, {
        "workOrder": wo("wko-1", "L1", "Replace belt", inc="inc-1", lck="lck-1", cost=2400, by="sue", opened=o),
        "part": part("belt", "Belt", 1200, "laser", 3)})
    c.ok("useParts", "amy", u, {"workOrderId": "wko-1", "partId": "belt", "quantity": 3}, {
        "workOrder": wo("wko-1", "L1", "Replace belt", inc="inc-1", lck="lck-1", cost=6000, by="sue", opened=o),
        "part": part("belt", "Belt", 1200, "laser", 0)})
    c.refuse("useParts", "sue", u, {"workOrderId": "wko-1", "partId": "belt", "quantity": 1}, "insufficient_parts")
    c.ok("listParts", "sue", u, {}, {"parts": [part("belt", "Belt", 1200, "laser", 0), part("chuck", "Chuck", 800, "lathe", 3)]})
    z = t(2, "10:30")
    c.refuse("closeWorkOrder", "sue", z, {"workOrderId": "wko-1", "note": "done"}, "incident_open")
    c.ok("resolveIncident", "sam", z, {"incidentId": "inc-1", "resolution": "fixed"})
    c.ok("closeWorkOrder", "sue", z, {"workOrderId": "wko-1", "note": "done"}, {
        "workOrder": wo("wko-1", "L1", "Replace belt", "closed", "inc-1", "lck-1", 6000, "sue", o, z, "done")})
    c.refuse("closeWorkOrder", "sue", z, {"workOrderId": "wko-1", "note": "again"}, "invalid_state")
    c.refuse("useParts", "sue", z, {"workOrderId": "wko-1", "partId": "chuck", "quantity": 4}, "invalid_state")
    c.refuse("linkIncident", "sue", z, {"workOrderId": "wko-1", "incidentId": "inc-2"}, "invalid_state")
    c.ok("listWorkOrders", "sue", z, {"machineId": "L1"}, {"machineId": "L1", "workOrders": [
        wo("wko-1", "L1", "Replace belt", "closed", "inc-1", "lck-1", 6000, "sue", o, z, "done")]})
    c.ok("listWorkOrders", "sue", z, {"machineId": "L2"}, {"machineId": "L2", "workOrders": []})
    c.refuse("listWorkOrders", "ada", z, {"machineId": "L1"}, "forbidden")
    c.audit_log(z)
    return c


def b_workorders_2() -> Case:
    c = C("B-workorders-2", "B-workorders", "work orders hold their own lockout; a manual lockout stays; closing releases exactly one")
    world(c)
    m = t(2, "10:00")
    c.ok("applyLockout", "sam", m, {"machineId": "L1", "reason": "manual"}, new="lck-1")
    a, b = t(2, "10:10"), t(2, "10:20")
    c.ok("openWorkOrder", "sue", a, {"machineId": "L1", "summary": "Fix A"}, {"workOrder": wo("wko-1", "L1", "Fix A", lck="lck-2", by="sue", opened=a)}, new="wko-1")
    c.ok("openWorkOrder", "amy", b, {"machineId": "L1", "summary": "Fix B"}, {"workOrder": wo("wko-2", "L1", "Fix B", lck="lck-3", by="amy", opened=b)}, new="wko-2")
    c.ok("listLockouts", "sam", b, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "manual", "sam", m), lock("lck-2", "L1", "workOrder", "Fix A", "sue", a),
        lock("lck-3", "L1", "workOrder", "Fix B", "amy", b)]})
    c1 = t(2, "10:30")
    c.ok("closeWorkOrder", "sue", c1, {"workOrderId": "wko-1", "note": "ok A"}, {
        "workOrder": wo("wko-1", "L1", "Fix A", "closed", None, "lck-2", 0, "sue", a, c1, "ok A")})
    c.ok("listLockouts", "sam", c1, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "manual", "sam", m), lock("lck-2", "L1", "workOrder", "Fix A", "sue", a, c1),
        lock("lck-3", "L1", "workOrder", "Fix B", "amy", b)]})
    ada_l1 = {"memberId": "ada", "machineId": "L1", "start": t(3, "10:00"), "end": t(3, "11:00")}
    c.refuse("createReservation", "ada", c1, ada_l1, "machine_locked_out")
    c.ok("createReservation", "ada", c1, {"memberId": "ada", "machineId": "L2", "start": t(3, "13:00"), "end": t(3, "14:00")},
         {"reservation": res("res-1", "ada", "L2", t(3, "13:00"), t(3, "14:00"))}, new="res-1")
    c.refuse("closeWorkOrder", "sue", c1, {"workOrderId": "wko-1", "note": "again"}, "invalid_state")
    c2 = t(2, "10:40")
    c.ok("closeWorkOrder", "amy", c2, {"workOrderId": "wko-2", "note": "ok B"}, {
        "workOrder": wo("wko-2", "L1", "Fix B", "closed", None, "lck-3", 0, "amy", b, c2, "ok B")})
    c.refuse("createReservation", "ada", c2, ada_l1, "machine_locked_out")
    c.ok("listWorkOrders", "sam", c2, {"machineId": "L1"}, {"machineId": "L1", "workOrders": [
        wo("wko-1", "L1", "Fix A", "closed", None, "lck-2", 0, "sue", a, c1, "ok A"),
        wo("wko-2", "L1", "Fix B", "closed", None, "lck-3", 0, "amy", b, c2, "ok B")]})
    r = t(2, "10:50")
    c.ok("releaseLockout", "sam", r, {"lockoutId": "lck-1"})
    c.ok("createReservation", "ada", r, ada_l1, {"reservation": res("res-2", "ada", "L1", t(3, "10:00"), t(3, "11:00"))}, new="res-2")
    c.ok("listLockouts", "sam", r, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "manual", "sam", m, r), lock("lck-2", "L1", "workOrder", "Fix A", "sue", a, c1),
        lock("lck-3", "L1", "workOrder", "Fix B", "amy", b, c2)]})
    c.audit_log(r)
    return c


# ---- B-escalations ---------------------------------------------------------------------------------------------------

def b_escalations_1() -> Case:
    c = C("B-escalations-1", "B-escalations", "escalation policy and the refusal order of escalate incident")
    world(c)
    n = t(2, "10:00")
    c.refuse("setEscalationPolicy", "sam", n, {"familyId": "laser", "minSeverity": "medium"}, "forbidden")
    c.refuse("setEscalationPolicy", "amy", n, {"familyId": "ghost", "minSeverity": "medium"}, "not_found")
    c.refuse("setEscalationPolicy", "amy", n, {"familyId": "laser", "minSeverity": "critical"}, "invalid_argument")
    c.ok("setEscalationPolicy", "amy", n, {"familyId": "laser", "minSeverity": "high"}, {"policy": {"familyId": "laser", "minSeverity": "high"}})
    c.ok("setEscalationPolicy", "amy", n, {"familyId": "laser", "minSeverity": "medium"}, {"policy": {"familyId": "laser", "minSeverity": "medium"}})
    c.ok("setEscalationPolicy", "amy", n, {"familyId": "laser", "minSeverity": "medium"}, {"policy": {"familyId": "laser", "minSeverity": "medium"}})
    for mach, sev, desc, iid in (("L1", "low", "scratch", "inc-1"), ("L1", "medium", "smoke", "inc-2"), ("L1", "high", "fire", "inc-3"),
                                 ("T1", "high", "jam", "inc-4"), ("L2", "medium", "noise", "inc-5")):
        c.ok("reportIncident", "sam", n, {"machineId": mach, "severity": sev, "description": desc}, new=iid)
    c.ok("resolveIncident", "sam", n, {"incidentId": "inc-5", "resolution": "n/a"})
    c.refuse("escalateIncident", "sam", n, {"incidentId": "inc-1"}, "below_policy")
    c.refuse("escalateIncident", "sam", n, {"incidentId": "inc-4"}, "no_policy")
    c.refuse("escalateIncident", "sam", n, {"incidentId": "inc-5"}, "invalid_state")
    c.refuse("escalateIncident", "sam", n, {"incidentId": "inc-9"}, "not_found")
    c.refuse("escalateIncident", "ada", n, {"incidentId": "inc-3"}, "forbidden")
    c.ok("listEscalations", "sam", n, {}, {"escalations": []})
    c.ok("listWorkOrders", "sam", n, {"machineId": "L1"}, {"machineId": "L1", "workOrders": []})
    c.ok("listLockouts", "sam", n, {"machineId": "L1"}, {"machineId": "L1", "lockouts": []})
    h = t(2, "10:10")
    c.ok("setEscalationPolicy", "amy", h, {"familyId": "laser", "minSeverity": "high"})
    c.refuse("escalateIncident", "sam", h, {"incidentId": "inc-2"}, "below_policy")
    c.refuse("escalateIncident", "sam", h, {"incidentId": "inc-5"}, "invalid_state")
    e1 = t(2, "10:20")
    c.ok("escalateIncident", "sam", e1, {"incidentId": "inc-3"}, {"escalation": {
        "escalationId": "esc-1", "incidentId": "inc-3", "machineId": "L1", "workOrderId": "wko-1", "cancelledReservationIds": [], "at": e1}},
         new="esc-1")
    c.refuse("escalateIncident", "sam", e1, {"incidentId": "inc-3"}, "already_escalated")
    c.ok("resolveIncident", "sam", e1, {"incidentId": "inc-3", "resolution": "done"})
    c.refuse("escalateIncident", "sam", e1, {"incidentId": "inc-3"}, "invalid_state")
    c.ok("setEscalationPolicy", "amy", e1, {"familyId": "laser", "minSeverity": "medium"})
    e2 = t(2, "10:30")
    c.ok("escalateIncident", "sue", e2, {"incidentId": "inc-2"}, {"escalation": {
        "escalationId": "esc-2", "incidentId": "inc-2", "machineId": "L1", "workOrderId": "wko-2", "cancelledReservationIds": [], "at": e2}},
         new="esc-2")
    c.refuse("escalateIncident", "sam", e2, {"incidentId": "inc-4"}, "no_policy")
    c.ok("listEscalations", "sam", e2, {}, {"escalations": [
        {"escalationId": "esc-1", "incidentId": "inc-3", "machineId": "L1", "workOrderId": "wko-1", "cancelledReservationIds": [], "at": e1},
        {"escalationId": "esc-2", "incidentId": "inc-2", "machineId": "L1", "workOrderId": "wko-2", "cancelledReservationIds": [], "at": e2}]})
    c.ok("listIncidents", "sam", e2, {}, {"incidents": [
        incident("inc-1", "L1", "low", "scratch", "sam", n),
        incident("inc-2", "L1", "medium", "smoke", "sam", n),
        incident("inc-3", "L1", "high", "fire", "sam", n, "resolved", "done", e1),
        incident("inc-4", "T1", "high", "jam", "sam", n),
        incident("inc-5", "L2", "medium", "noise", "sam", n, "resolved", "n/a", n)]})
    c.ok("listWorkOrders", "sam", e2, {"machineId": "L1"}, {"machineId": "L1", "workOrders": [
        wo("wko-1", "L1", "Escalated incident inc-3", inc="inc-3", lck="lck-1", by="sam", opened=e1),
        wo("wko-2", "L1", "Escalated incident inc-2", inc="inc-2", lck="lck-2", by="sue", opened=e2)]})
    c.ok("listLockouts", "sam", e2, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        has(lockoutId="lck-1", machineId="L1", source="workOrder", appliedBy="sam", appliedAt=e1, releasedAt=None),
        has(lockoutId="lck-2", machineId="L1", source="workOrder", appliedBy="sue", appliedAt=e2, releasedAt=None)]})
    c.audit_log(e2)
    return c


def b_escalations_2() -> Case:
    c = C("B-escalations-2", "B-escalations", "escalation effects: cancellations by start, notices share the counter, other machines untouched")
    world(c)
    certify(c, "bob", W)
    c.ok("setEscalationPolicy", "amy", W, {"familyId": "laser", "minSeverity": "low"})
    c.ok("postNotice", "sam", W, {"memberId": "bob", "subject": "hello"}, new="ntc-1")
    plan_ = [("res-1", "ada", "L1", "14:00", "15:00"), ("res-2", "bob", "L1", "10:00", "11:00"),
             ("res-3", "ada", "L1", "12:00", "13:00"), ("res-4", "bob", "L2", "12:00", "13:00")]
    for rid, mem, mach, s, e in plan_:
        c.ok("createReservation", mem, W, {"memberId": mem, "machineId": mach, "start": t(4, s), "end": t(4, e)},
             {"reservation": res(rid, mem, mach, t(4, s), t(4, e))}, new=rid)
    c.ok("setMemberStanding", "sam", W, {"memberId": "bob", "standing": "suspended"})
    c.ok("reportIncident", "sam", t(3, "08:00"), {"machineId": "L1", "severity": "medium", "description": "smoke"}, new="inc-1")
    e = t(3, "09:00")
    c.ok("escalateIncident", "sue", e, {"incidentId": "inc-1"}, {"escalation": {
        "escalationId": "esc-1", "incidentId": "inc-1", "machineId": "L1", "workOrderId": "wko-1",
        "cancelledReservationIds": ["res-2", "res-3", "res-1"], "at": e}}, new="esc-1")
    c.ok("listNotices", "bob", e, {"memberId": "bob"}, {"memberId": "bob", "notices": [
        notice("ntc-1", "bob", "manual", "hello", None, W),
        notice("ntc-2", "bob", "cancellation", "Reservation res-2 was cancelled", "res-2", e)]})
    c.ok("listNotices", "ada", e, {"memberId": "ada"}, {"memberId": "ada", "notices": [
        notice("ntc-3", "ada", "cancellation", "Reservation res-3 was cancelled", "res-3", e),
        notice("ntc-4", "ada", "cancellation", "Reservation res-1 was cancelled", "res-1", e)]})
    c.ok("listMemberReservations", "sam", e, {"memberId": "bob"}, {"memberId": "bob", "reservations": [
        res("res-2", "bob", "L1", t(4, "10:00"), t(4, "11:00"), "cancelled"),
        res("res-4", "bob", "L2", t(4, "12:00"), t(4, "13:00"))]})
    c.ok("listMemberReservations", "sam", e, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-3", "ada", "L1", t(4, "12:00"), t(4, "13:00"), "cancelled"),
        res("res-1", "ada", "L1", t(4, "14:00"), t(4, "15:00"), "cancelled")]})
    c.ok("getMachineSchedule", "sam", e, {"machineId": "L1", "date": "2026-03-04"}, {
        "machineId": "L1", "date": "2026-03-04", "openingHours": {"open": "09:00", "close": "21:00"}, "reservations": [], "maintenance": []})
    c.audit_log(e)
    return c


# ---- B-compliance ----------------------------------------------------------------------------------------------------

def b_compliance_1() -> Case:
    c = C("B-compliance-1", "B-compliance", "compliance summary: one row per family, flags and the minor rule")
    world(c)
    n = t(2, "10:00")
    c.ok("complianceSummary", "sam", n, {"memberId": "ada"}, {"memberId": "ada", "standing": "good", "minor": False, "families": [
        fam("laser", True, True, True, True), fam("lathe", False, True, True, True)]})
    c.ok("addFamily", "admin", n, {"familyId": "Zeta", "name": "Zeta tools"})
    c.ok("publishWaiver", "amy", n, {"familyId": "laser", "text": "v1"})
    c.ok("complianceSummary", "ada", n, {"memberId": "ada"}, {"memberId": "ada", "standing": "good", "minor": False, "families": [
        fam("Zeta", False, True, True, True), fam("laser", True, True, False, True), fam("lathe", False, True, True, True)]})
    c.ok("signWaiver", "ada", n, {"memberId": "ada", "familyId": "laser"})
    c.ok("addPpeItem", "sam", n, {"itemId": "gl", "name": "Glasses", "stock": 2})
    c.ok("addPpeItem", "sam", n, {"itemId": "ap", "name": "Apron", "stock": 1})
    c.ok("requirePpe", "amy", n, {"familyId": "laser", "itemId": "gl"})
    c.ok("requirePpe", "amy", n, {"familyId": "lathe", "itemId": "ap"})
    c.ok("complianceSummary", "sam", n, {"memberId": "ada"}, {"memberId": "ada", "standing": "good", "minor": False, "families": [
        fam("Zeta", False, True, True, True), fam("laser", True, True, True, False), fam("lathe", False, True, True, False)]})
    c.ok("issuePpe", "sam", n, {"memberId": "ada", "itemId": "gl"})
    c.ok("publishWaiver", "amy", n, {"familyId": "laser", "text": "v2"})
    c.ok("complianceSummary", "ada", n, {"memberId": "ada"}, {"memberId": "ada", "standing": "good", "minor": False, "families": [
        fam("Zeta", False, True, True, True), fam("laser", True, True, False, True), fam("lathe", False, True, True, False)]})
    # bob is a minor until his 18th birthday on 2026-03-03
    c.ok("declareMinor", "sam", n, {"memberId": "bob", "birthDate": "2008-03-03", "guardianId": "ada"})
    c.ok("complianceSummary", "bob", n, {"memberId": "bob"}, {"memberId": "bob", "standing": "good", "minor": True, "families": [
        fam("Zeta", False, False, True, True), fam("laser", False, False, False, False), fam("lathe", False, False, True, False)]})
    c.ok("grantConsent", "ada", n, {"guardianId": "ada", "memberId": "bob", "familyId": "laser"})
    c.ok("setMemberStanding", "sam", n, {"memberId": "bob", "standing": "suspended"})
    c.ok("complianceSummary", "amy", t(2, "23:59"), {"memberId": "bob"}, {"memberId": "bob", "standing": "suspended", "minor": True, "families": [
        fam("Zeta", False, False, True, True), fam("laser", False, True, False, False), fam("lathe", False, False, True, False)]})
    c.ok("complianceSummary", "amy", t(3, "00:00"), {"memberId": "bob"}, {"memberId": "bob", "standing": "suspended", "minor": False, "families": [
        fam("Zeta", False, True, True, True), fam("laser", False, True, False, False), fam("lathe", False, True, True, False)]})
    c.refuse("complianceSummary", "ada", t(3, "00:00"), {"memberId": "bob"}, "forbidden")
    c.refuse("complianceSummary", "sam", t(3, "00:00"), {"memberId": "ghost"}, "not_found")
    c.refuse("complianceSummary", "sam", t(3, "00:00"), {"memberId": "sam"}, "not_found")
    c.refuse("complianceSummary", "ada", t(3, "00:00"), {"memberId": "sam"}, "not_found")
    # certification of ada is valid until 2027-03-01 and expired on 2027-03-02
    c.ok("complianceSummary", "ada", "2027-03-01T23:59Z", {"memberId": "ada"}, {"memberId": "ada", "standing": "good", "minor": False, "families": [
        fam("Zeta", False, True, True, True), fam("laser", True, True, False, True), fam("lathe", False, True, True, False)]})
    c.ok("complianceSummary", "ada", "2027-03-02T00:00Z", {"memberId": "ada"}, {"memberId": "ada", "standing": "good", "minor": False, "families": [
        fam("Zeta", False, True, True, True), fam("laser", False, True, False, True), fam("lathe", False, True, True, False)]})
    c.audit_log("2027-03-02T00:00Z")
    return c


def b_compliance_2() -> Case:
    c = C("B-compliance-2", "B-compliance", "check eligibility: blockers in order, dates, permissions and the order of checks")
    world(c)
    n = t(2, "10:00")
    c.ok("checkEligibility", "sam", n, {"memberId": "bob", "machineId": "L1", "date": "2026-03-03"}, elig("bob", "L1", "2026-03-03", ["not_certified"]))
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L1", "date": "2026-03-03"}, elig("ada", "L1", "2026-03-03", []))
    c.ok("checkEligibility", "bob", n, {"memberId": "bob", "machineId": "L2", "date": "2026-03-03"}, elig("bob", "L2", "2026-03-03", ["not_certified"]))
    for d, b in (("2026-03-02", []), ("2026-03-01", ["not_certified"]), ("2000-01-01", ["not_certified"]), ("2027-03-01", []),
                 ("2027-03-02", ["not_certified"]), ("2099-12-31", ["not_certified"])):
        c.ok("checkEligibility", "amy", n, {"memberId": "ada", "machineId": "L2", "date": d}, elig("ada", "L2", d, b))
    c.ok("setInspectionPlan", "sam", n, {"machineId": "L1", "intervalDays": 29})
    for d, b in (("2026-03-30", []), ("2026-03-31", ["inspection_due"]), ("2026-04-01", ["inspection_due"])):
        c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L1", "date": d}, elig("ada", "L1", d, b))
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L2", "date": "2026-03-31"}, elig("ada", "L2", "2026-03-31", []))
    c.ok("applyLockout", "sam", n, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L1", "date": "2026-03-30"}, elig("ada", "L1", "2026-03-30", ["machine_locked_out"]))
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L1", "date": "2026-03-31"},
         elig("ada", "L1", "2026-03-31", ["machine_locked_out", "inspection_due"]))
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L1", "date": "2026-03-01"},
         elig("ada", "L1", "2026-03-01", ["not_certified", "machine_locked_out"]))
    c.ok("setMemberStanding", "sam", n, {"memberId": "ada", "standing": "suspended"})
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L1", "date": "2026-03-31"},
         elig("ada", "L1", "2026-03-31", ["member_suspended", "machine_locked_out", "inspection_due"]))
    c.ok("setMemberStanding", "sam", n, {"memberId": "ada", "standing": "good"})
    c.ok("revokeCertification", "sam", n, {"memberId": "ada", "familyId": "laser"})
    c.ok("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L2", "date": "2026-03-03"}, elig("ada", "L2", "2026-03-03", ["not_certified"]))
    c.refuse("checkEligibility", "ada", n, {"memberId": "bob", "machineId": "L1", "date": "2026-03-03"}, "forbidden")
    c.refuse("checkEligibility", "ada", n, {"memberId": "sam", "machineId": "L1", "date": "2026-03-03"}, "not_found")
    c.refuse("checkEligibility", "ada", n, {"memberId": "ada", "machineId": "L9", "date": "2026-03-03"}, "not_found")
    c.refuse("checkEligibility", "bob", n, {"memberId": "ada", "machineId": "L9", "date": "2026-03-03"}, "not_found")
    c.refuse("checkEligibility", "sam", n, {"memberId": "ada", "machineId": "L1", "date": "2026-02-30"}, "invalid_argument")
    c.refuse("checkEligibility", "sam", n, {"memberId": "ada", "machineId": "L1"}, "invalid_argument")
    c.refuse("checkEligibility", "ghost", n, {"memberId": "ada", "machineId": "L1", "date": "2026-03-03"}, "unknown_actor")
    c.audit_log(n)
    return c


# ---- B-bookings ------------------------------------------------------------------------------------------------------

def b_bookings_1() -> Case:
    c = C("B-bookings-1", "B-bookings", "create reservation: interval, granularity, start, opening hours; touching slots; schedule")
    world(c)

    def book(start, end, mach="L1"):
        return {"memberId": "ada", "machineId": mach, "start": start, "end": end}

    c.refuse("createReservation", "ada", W, book(t(3, "10:00"), t(3, "10:00")), "invalid_interval")
    c.refuse("createReservation", "ada", W, book(t(3, "11:00"), t(3, "10:00")), "invalid_interval")
    c.refuse("createReservation", "ada", W, book(t(3, "10:10"), t(3, "10:10")), "invalid_interval")
    c.refuse("createReservation", "ada", W, book(t(3, "10:15"), t(3, "11:00")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(3, "10:00"), t(3, "10:15")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(3, "10:00"), t(3, "10:45")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(3, "10:00"), t(3, "14:30")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(2, "08:15"), t(2, "08:45")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(2, "08:30"), t(2, "09:30")), "starts_in_past")
    c.refuse("createReservation", "ada", W, book(t(3, "08:30"), t(3, "09:30")), "outside_opening_hours")
    c.refuse("createReservation", "ada", W, book(t(3, "20:30"), t(3, "21:30")), "outside_opening_hours")
    c.refuse("createReservation", "ada", W, book(t(3, "23:30"), t(4, "00:30")), "outside_opening_hours")
    c.refuse("createReservation", "ada", W, book(t(7, "17:30"), t(7, "18:30")), "outside_opening_hours")
    c.refuse("createReservation", "ada", W, book(t(7, "09:30"), t(7, "10:30")), "outside_opening_hours")
    c.refuse("createReservation", "ada", W, book(t(8, "10:00"), t(8, "11:00")), "outside_opening_hours")
    c.refuse("createReservation", "ada", W, {"memberId": "ada", "machineId": "L1", "start": t(3, "10:00")}, "invalid_argument")
    c.refuse("createReservation", "bob", W, book(t(3, "10:00"), t(3, "11:00")), "forbidden")
    c.refuse("createReservation", "ada", W, book(t(3, "10:00"), t(3, "11:00"), "L9"), "not_found")
    c.refuse("createReservation", "ada", W, {"memberId": "ghost", "machineId": "L1", "start": t(3, "10:00"), "end": t(3, "11:00")}, "not_found")
    c.refuse("createReservation", "sam", W, {"memberId": "sue", "machineId": "L1", "start": t(3, "10:00"), "end": t(3, "11:00")}, "not_found")
    c.ok("createReservation", "ada", W, book(t(2, "09:00"), t(2, "10:00")), {"reservation": res("res-1", "ada", "L1", t(2, "09:00"), t(2, "10:00"))}, new="res-1")
    c.ok("createReservation", "ada", W, book(t(3, "10:00"), t(3, "14:00")), {"reservation": res("res-2", "ada", "L1", t(3, "10:00"), t(3, "14:00"))}, new="res-2")
    c.ok("createReservation", "ada", W, book(t(3, "14:00"), t(3, "14:30")), {"reservation": res("res-3", "ada", "L1", t(3, "14:00"), t(3, "14:30"))}, new="res-3")
    c.ok("createReservation", "ada", W, book(t(3, "20:30"), t(3, "21:00")), {"reservation": res("res-4", "ada", "L1", t(3, "20:30"), t(3, "21:00"))}, new="res-4")
    c.ok("createReservation", "ada", W, book(t(7, "17:30"), t(7, "18:00")), {"reservation": res("res-5", "ada", "L1", t(7, "17:30"), t(7, "18:00"))}, new="res-5")
    c.ok("createReservation", "ada", W, book(t(7, "10:00"), t(7, "10:30")), {"reservation": res("res-6", "ada", "L1", t(7, "10:00"), t(7, "10:30"))}, new="res-6")
    c.refuse("createReservation", "ada", W, book(t(3, "13:30"), t(3, "14:30")), "machine_booked")
    c.refuse("createReservation", "ada", W, book(t(3, "13:30"), t(3, "14:30"), "L2"), "member_double_booked")
    c.ok("cancelReservation", "ada", W, {"reservationId": "res-3"}, {"reservation": res("res-3", "ada", "L1", t(3, "14:00"), t(3, "14:30"), "cancelled")})
    c.ok("createReservation", "ada", W, book(t(3, "14:00"), t(3, "14:30")), {"reservation": res("res-7", "ada", "L1", t(3, "14:00"), t(3, "14:30"))}, new="res-7")
    c.ok("listMemberReservations", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(2, "09:00"), t(2, "10:00")),
        res("res-2", "ada", "L1", t(3, "10:00"), t(3, "14:00")),
        res("res-3", "ada", "L1", t(3, "14:00"), t(3, "14:30"), "cancelled"),
        res("res-7", "ada", "L1", t(3, "14:00"), t(3, "14:30")),
        res("res-4", "ada", "L1", t(3, "20:30"), t(3, "21:00")),
        res("res-6", "ada", "L1", t(7, "10:00"), t(7, "10:30")),
        res("res-5", "ada", "L1", t(7, "17:30"), t(7, "18:00"))]})
    c.ok("getMachineSchedule", "ada", W, {"machineId": "L1", "date": "2026-03-03"}, {
        "machineId": "L1", "date": "2026-03-03", "openingHours": {"open": "09:00", "close": "21:00"}, "reservations": [
            res("res-2", "ada", "L1", t(3, "10:00"), t(3, "14:00")), res("res-7", "ada", "L1", t(3, "14:00"), t(3, "14:30")),
            res("res-4", "ada", "L1", t(3, "20:30"), t(3, "21:00"))], "maintenance": []})
    c.ok("getMachineSchedule", "ada", W, {"machineId": "L1", "date": "2026-03-07"}, {
        "machineId": "L1", "date": "2026-03-07", "openingHours": {"open": "10:00", "close": "18:00"}, "reservations": [
            res("res-6", "ada", "L1", t(7, "10:00"), t(7, "10:30")), res("res-5", "ada", "L1", t(7, "17:30"), t(7, "18:00"))], "maintenance": []})
    c.ok("getMachineSchedule", "ada", W, {"machineId": "L1", "date": "2026-03-08"}, {
        "machineId": "L1", "date": "2026-03-08", "openingHours": None, "reservations": [], "maintenance": []})
    c.audit_log(W)
    return c


def b_bookings_2() -> Case:
    c = C("B-bookings-2", "B-bookings", "check in and check out: boundaries, captured rate, tier discount at check-out, lockout, missed")
    world(c)
    for rid, mach, s, e in (("res-1", "L1", "10:00", "11:00"), ("res-2", "L2", "12:00", "13:00"), ("res-3", "L1", "14:00", "15:00"),
                            ("res-4", "L1", "16:00", "17:00")):
        c.ok("createReservation", "ada", W, {"memberId": "ada", "machineId": mach, "start": t(3, s), "end": t(3, e)},
             {"reservation": res(rid, "ada", mach, t(3, s), t(3, e))}, new=rid)
    r1 = ("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"))
    c.refuse("checkIn", "ada", t(3, "09:59"), {"reservationId": "res-1"}, "check_in_too_early")
    c.refuse("checkIn", "bob", t(3, "09:59"), {"reservationId": "res-1"}, "forbidden")
    c.ok("checkIn", "ada", t(3, "10:00"), {"reservationId": "res-1"}, {"reservation": res(*r1, "checkedIn", cin=t(3, "10:00"), rate=45)})
    c.refuse("checkIn", "ada", t(3, "10:00"), {"reservationId": "res-1"}, "invalid_state")
    c.refuse("cancelReservation", "ada", t(3, "10:00"), {"reservationId": "res-1"}, "invalid_state")
    c.refuse("checkOut", "ada", t(3, "10:00"), {"reservationId": "res-3"}, "invalid_state")
    c.ok("applyLockout", "sam", t(3, "10:10"), {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("checkOut", "ada", t(3, "10:37"), {"reservationId": "res-1"}, {
        "reservation": res(*r1, "completed", t(3, "10:00"), t(3, "10:37"), 45, 37, 1499),
        "charge": chg("usage", "res-1", 37, 45, 10, 1499, t(3, "10:37"))})
    c.refuse("checkOut", "ada", t(3, "10:40"), {"reservationId": "res-1"}, "invalid_state")
    r2 = ("res-2", "ada", "L2", t(3, "12:00"), t(3, "13:00"))
    c.ok("checkIn", "ada", t(3, "12:00"), {"reservationId": "res-2"}, {"reservation": res(*r2, "checkedIn", cin=t(3, "12:00"), rate=60)})
    c.ok("setMachineRate", "amy", t(3, "12:10"), {"machineId": "L2", "rateCentsPerMinute": 100}, {"machine": {
        "machineId": "L2", "name": "Laser two", "familyId": "laser", "rateCentsPerMinute": 100}})
    c.ok("changeTier", "sam", t(3, "12:20"), {"memberId": "ada", "tier": "pro"}, {"member": {
        "memberId": "ada", "name": "Ada Byron", "tier": "pro", "standing": "good", "registeredAt": W}})
    c.ok("checkOut", "ada", t(3, "12:30"), {"reservationId": "res-2"}, {
        "reservation": res(*r2, "completed", t(3, "12:00"), t(3, "12:30"), 60, 30, 1350),
        "charge": chg("usage", "res-2", 30, 60, 25, 1350, t(3, "12:30"))})
    r3 = ("res-3", "ada", "L1", t(3, "14:00"), t(3, "15:00"))
    c.refuse("checkIn", "ada", t(3, "14:00"), {"reservationId": "res-3"}, "machine_locked_out")
    c.ok("releaseLockout", "sam", t(3, "14:10"), {"lockoutId": "lck-1"})
    c.ok("checkIn", "ada", t(3, "14:59"), {"reservationId": "res-3"}, {"reservation": res(*r3, "checkedIn", cin=t(3, "14:59"), rate=45)})
    c.ok("setMemberStanding", "sam", t(3, "15:10"), {"memberId": "ada", "standing": "suspended"})
    c.ok("checkOut", "ada", t(3, "15:30"), {"reservationId": "res-3"}, {
        "reservation": res(*r3, "completed", t(3, "14:59"), t(3, "15:30"), 45, 31, 1046),
        "charge": chg("usage", "res-3", 31, 45, 25, 1046, t(3, "15:30"))})
    c.refuse("checkIn", "ada", t(3, "17:00"), {"reservationId": "res-4"}, "invalid_state")
    c.refuse("cancelReservation", "sam", t(3, "17:00"), {"reservationId": "res-4"}, "invalid_state")
    c.ok("listMemberReservations", "ada", t(3, "17:00"), {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res(*r1, "completed", t(3, "10:00"), t(3, "10:37"), 45, 37, 1499),
        res(*r2, "completed", t(3, "12:00"), t(3, "12:30"), 60, 30, 1350),
        res(*r3, "completed", t(3, "14:59"), t(3, "15:30"), 45, 31, 1046),
        res("res-4", "ada", "L1", t(3, "16:00"), t(3, "17:00"), "missed")]})
    c.audit_log(t(3, "17:00"))
    return c


# ---- X-floor ---------------------------------------------------------------------------------------------------------

def x_floor_1() -> Case:
    c = C("X-floor-1", "X-floor", "create reservation: every gate in the listed order, removed one at a time")
    world(c)
    certify(c, "bob", W)
    c.ok("createReservation", "bob", W, {"memberId": "bob", "machineId": "L2", "start": t(3, "10:00"), "end": t(3, "11:00")},
         {"reservation": res("res-1", "bob", "L2", t(3, "10:00"), t(3, "11:00"))}, new="res-1")
    c.ok("createReservation", "ada", W, {"memberId": "ada", "machineId": "L1", "start": t(3, "11:00"), "end": t(3, "12:00")},
         {"reservation": res("res-2", "ada", "L1", t(3, "11:00"), t(3, "12:00"))}, new="res-2")
    n = t(2, "10:00")
    c.ok("revokeCertification", "sam", n, {"memberId": "bob", "familyId": "laser"})
    c.ok("declareMinor", "sam", n, {"memberId": "bob", "birthDate": "2012-05-01", "guardianId": "ada"})
    c.ok("setMemberStanding", "sam", n, {"memberId": "bob", "standing": "suspended"})
    c.ok("publishWaiver", "amy", n, {"familyId": "laser", "text": "Laser waiver"})
    c.ok("applyLockout", "sam", n, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("setInspectionPlan", "sam", n, {"machineId": "L1", "intervalDays": 1}, {"plan": plan("L1", 1, "2026-03-02", "2026-03-03", False)})
    c.ok("scheduleMaintenance", "sam", n, {"machineId": "L1", "start": t(3, "10:00"), "end": t(3, "11:00"), "reason": "service"}, new="mnt-1")
    c.ok("setSupervision", "amy", n, {"familyId": "laser", "required": True})

    def bob(start, end, mach="L1"):
        return {"memberId": "bob", "machineId": mach, "start": start, "end": end}

    slot = bob(t(3, "10:00"), t(3, "12:00"))
    c.refuse("createReservation", "bob", n, bob(t(3, "11:00"), t(3, "10:00")), "invalid_interval")
    c.refuse("createReservation", "bob", n, bob(t(3, "10:00"), t(3, "10:00")), "invalid_interval")
    c.refuse("createReservation", "bob", n, bob(t(3, "10:10"), t(3, "12:00")), "invalid_slot")
    c.refuse("createReservation", "bob", n, bob(t(2, "08:15"), t(2, "08:45")), "invalid_slot")
    c.refuse("createReservation", "bob", n, bob(t(2, "09:00"), t(2, "10:00")), "starts_in_past")
    c.refuse("createReservation", "bob", n, bob(t(3, "08:30"), t(3, "09:30")), "outside_opening_hours")
    c.refuse("createReservation", "bob", n, bob(t(3, "20:30"), t(3, "21:30")), "outside_opening_hours")
    c.refuse("createReservation", "bob", n, slot, "member_suspended")
    c.ok("setMemberStanding", "sam", n, {"memberId": "bob", "standing": "good"})
    c.refuse("createReservation", "bob", n, slot, "guardian_consent_missing")
    c.ok("grantConsent", "ada", n, {"guardianId": "ada", "memberId": "bob", "familyId": "laser"})
    c.refuse("createReservation", "bob", n, slot, "not_certified")
    certify(c, "bob", n)
    c.refuse("createReservation", "bob", n, slot, "waiver_missing")
    c.ok("signWaiver", "bob", n, {"memberId": "bob", "familyId": "laser"})
    c.refuse("createReservation", "bob", n, slot, "machine_locked_out")
    c.ok("releaseLockout", "sam", n, {"lockoutId": "lck-1"})
    c.refuse("createReservation", "bob", n, slot, "inspection_due")
    c.ok("setInspectionPlan", "sam", n, {"machineId": "L1", "intervalDays": 30}, {"plan": plan("L1", 30, "2026-03-02", "2026-04-01", False)})
    c.refuse("createReservation", "bob", n, slot, "machine_in_maintenance")
    c.ok("cancelMaintenance", "sam", n, {"windowId": "mnt-1"})
    c.refuse("createReservation", "bob", n, slot, "supervisor_required")
    c.ok("addShift", "sue", n, {"userId": "sue", "start": t(3, "09:00"), "end": t(3, "13:00")}, new="shf-1")
    c.refuse("createReservation", "bob", n, slot, "machine_booked")
    c.ok("cancelReservation", "ada", n, {"reservationId": "res-2"})
    c.refuse("createReservation", "bob", n, slot, "member_double_booked")
    c.ok("cancelReservation", "bob", n, {"reservationId": "res-1"})
    c.ok("createReservation", "bob", n, slot, {"reservation": res("res-3", "bob", "L1", t(3, "10:00"), t(3, "12:00"))}, new="res-3")
    c.audit_log(n)
    return c


def x_floor_2() -> Case:
    c = C("X-floor-2", "X-floor", "check in: every gate in the listed order, removed one at a time")
    world(c)
    certify(c, "bob", W)
    c.ok("createReservation", "bob", W, {"memberId": "bob", "machineId": "L1", "start": t(3, "10:00"), "end": t(3, "11:00")},
         {"reservation": res("res-1", "bob", "L1", t(3, "10:00"), t(3, "11:00"))}, new="res-1")
    c.ok("createReservation", "bob", W, {"memberId": "bob", "machineId": "L1", "start": t(2, "09:30"), "end": t(2, "10:00")},
         {"reservation": res("res-2", "bob", "L1", t(2, "09:30"), t(2, "10:00"))}, new="res-2")
    c.ok("createReservation", "bob", W, {"memberId": "bob", "machineId": "L1", "start": t(3, "12:00"), "end": t(3, "13:00")},
         {"reservation": res("res-3", "bob", "L1", t(3, "12:00"), t(3, "13:00"))}, new="res-3")
    c.ok("cancelReservation", "bob", W, {"reservationId": "res-3"})
    d = t(2, "09:30")
    c.ok("revokeCertification", "sam", d, {"memberId": "bob", "familyId": "laser"})
    c.ok("declareMinor", "sam", d, {"memberId": "bob", "birthDate": "2012-05-01", "guardianId": "ada"})
    c.ok("setMemberStanding", "sam", d, {"memberId": "bob", "standing": "suspended"})
    c.ok("publishWaiver", "amy", d, {"familyId": "laser", "text": "w"})
    c.ok("addPpeItem", "sam", d, {"itemId": "gl", "name": "Safety glasses", "stock": 1})
    c.ok("requirePpe", "amy", d, {"familyId": "laser", "itemId": "gl"})
    c.ok("applyLockout", "sam", d, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("issueBadge", "sam", d, {"memberId": "bob"}, new="bdg-1")
    c.ok("setInspectionPlan", "sam", d, {"machineId": "L1", "intervalDays": 1})
    c.ok("setSupervision", "amy", d, {"familyId": "laser", "required": True})
    s1 = t(2, "10:00")
    c.refuse("checkIn", "bob", s1, {"reservationId": "res-2"}, "invalid_state")
    c.refuse("checkIn", "bob", s1, {"reservationId": "res-3"}, "invalid_state")
    c.refuse("checkIn", "bob", s1, {"reservationId": "res-1"}, "check_in_too_early")
    c.refuse("checkIn", "ada", s1, {"reservationId": "res-1"}, "forbidden")
    c.refuse("checkIn", "bob", s1, {"reservationId": "res-9"}, "not_found")
    n = t(3, "10:00")
    one = {"reservationId": "res-1"}
    c.refuse("checkIn", "bob", n, one, "member_suspended")
    c.ok("setMemberStanding", "sam", n, {"memberId": "bob", "standing": "good"})
    c.refuse("checkIn", "bob", n, one, "guardian_consent_missing")
    c.ok("grantConsent", "ada", n, {"guardianId": "ada", "memberId": "bob", "familyId": "laser"})
    c.refuse("checkIn", "bob", n, one, "not_certified")
    certify(c, "bob", n)
    c.refuse("checkIn", "bob", n, one, "waiver_missing")
    c.ok("signWaiver", "bob", n, {"memberId": "bob", "familyId": "laser"})
    c.refuse("checkIn", "bob", n, one, "ppe_missing")
    c.ok("issuePpe", "sam", n, {"memberId": "bob", "itemId": "gl"})
    c.refuse("checkIn", "bob", n, one, "machine_locked_out")
    c.ok("releaseLockout", "sam", n, {"lockoutId": "lck-1"})
    c.refuse("checkIn", "bob", n, one, "not_on_premises")
    c.ok("enterDoor", "bob", n, {"badgeId": "bdg-1"})
    c.ok("checkIn", "bob", n, one, {"reservation": res("res-1", "bob", "L1", t(3, "10:00"), t(3, "11:00"), "checkedIn", cin=n, rate=45)})
    c.audit_log(n)
    return c


def x_floor_3() -> Case:
    c = C("X-floor-3", "X-floor", "check out with passes and consumables, and the lines of the next invoice")
    world(c)
    c.ok("addMaterial", "sam", W, {"materialId": "acrylic", "name": "Acrylic", "unitPriceCents": 30})
    c.ok("restockMaterial", "sam", W, {"materialId": "acrylic", "quantity": 15})
    c.ok("setConsumption", "sam", W, {"machineId": "L1", "materialId": "ply", "unitsPerHour": 6})
    c.ok("setConsumption", "sam", W, {"machineId": "L1", "materialId": "acrylic", "unitsPerHour": 25})
    c.ok("addPassProduct", "amy", W, {"productId": "p20", "name": "Twenty", "minutes": 20, "priceCents": 1000, "validDays": 30})
    c.ok("addPassProduct", "amy", W, {"productId": "p120", "name": "Big", "minutes": 120, "priceCents": 2000, "validDays": 365})
    c.ok("sellPass", "ada", W, {"memberId": "ada", "productId": "p20"}, {
        "pass": {"passId": "pas-1", "memberId": "ada", "productId": "p20", "minutesLeft": 20, "issuedOn": "2026-03-02", "expiresOn": "2026-04-01"},
        "charge": chg("pass", "pas-1", 1, 1000, 0, 1000, W)}, new="pas-1")
    for rid, s, e in (("res-1", "10:00", "11:00"), ("res-2", "11:00", "12:00"), ("res-3", "13:00", "14:00")):
        c.ok("createReservation", "ada", W, {"memberId": "ada", "machineId": "L1", "start": t(3, s), "end": t(3, e)},
             {"reservation": res(rid, "ada", "L1", t(3, s), t(3, e))}, new=rid)
    # session A: 50 metered minutes, the pass draws 20, 30 are billable; acrylic 21 units but stock 15; ply 5 units
    c.ok("checkIn", "ada", t(3, "10:00"), {"reservationId": "res-1"})
    c.ok("checkOut", "ada", t(3, "10:50"), {"reservationId": "res-1"}, {
        "reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"), "completed", t(3, "10:00"), t(3, "10:50"), 45, 50, 1215),
        "charge": chg("usage", "res-1", 30, 45, 10, 1215, t(3, "10:50"))})
    # session B: 0 metered minutes
    c.ok("checkIn", "ada", t(3, "11:00"), {"reservationId": "res-2"})
    c.ok("checkOut", "ada", t(3, "11:00"), {"reservationId": "res-2"}, {
        "reservation": res("res-2", "ada", "L1", t(3, "11:00"), t(3, "12:00"), "completed", t(3, "11:00"), t(3, "11:00"), 45, 0, 0),
        "charge": chg("usage", "res-2", 0, 45, 10, 0, t(3, "11:00"))})
    c.ok("sellPass", "ada", t(3, "12:00"), {"memberId": "ada", "productId": "p120"}, {
        "pass": {"passId": "pas-2", "memberId": "ada", "productId": "p120", "minutesLeft": 120, "issuedOn": "2026-03-03", "expiresOn": "2027-03-03"},
        "charge": chg("pass", "pas-2", 1, 2000, 0, 2000, t(3, "12:00"))}, new="pas-2")
    # session C: 11 metered minutes, wholly covered by the pass, the materials are still drawn on 11 minutes
    c.ok("checkIn", "ada", t(3, "13:00"), {"reservationId": "res-3"})
    c.ok("checkOut", "ada", t(3, "13:11"), {"reservationId": "res-3"}, {
        "reservation": res("res-3", "ada", "L1", t(3, "13:00"), t(3, "14:00"), "completed", t(3, "13:00"), t(3, "13:11"), 45, 11, 0),
        "charge": chg("usage", "res-3", 0, 45, 10, 0, t(3, "13:11"))})
    c.ok("listMemberPasses", "ada", t(3, "13:11"), {"memberId": "ada"}, {"memberId": "ada", "passes": [
        {"passId": "pas-1", "memberId": "ada", "productId": "p20", "minutesLeft": 0, "issuedOn": "2026-03-02", "expiresOn": "2026-04-01"},
        {"passId": "pas-2", "memberId": "ada", "productId": "p120", "minutesLeft": 109, "issuedOn": "2026-03-03", "expiresOn": "2027-03-03"}]})
    c.ok("listMaterials", "ada", t(3, "13:11"), {}, {"materials": [mat("acrylic", "Acrylic", 30, 0), mat("ply", "Plywood", 150, 93)]})
    inv_at, due = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    ada_lines = [
        chg("membership", "plus", 1, 4900, 0, 4900, inv_at),
        chg("course", "laser-101", 1, 5000, 0, 5000, W),
        chg("pass", "pas-1", 1, 1000, 0, 1000, W),
        chg("usage", "res-1", 30, 45, 10, 1215, t(3, "10:50")),
        chg("material", "res-1", 15, 30, 0, 450, t(3, "10:50")),
        chg("material", "res-1", 5, 150, 0, 750, t(3, "10:50")),
        chg("usage", "res-2", 0, 45, 10, 0, t(3, "11:00")),
        chg("pass", "pas-2", 1, 2000, 0, 2000, t(3, "12:00")),
        chg("usage", "res-3", 0, 45, 10, 0, t(3, "13:11")),
        chg("material", "res-3", 2, 150, 0, 300, t(3, "13:11"))]
    c.ok("generateInvoices", "sam", inv_at, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        invoice("inv-1", "ada", "2026-03", inv_at, due, ada_lines, 15615),
        invoice("inv-2", "bob", "2026-03", inv_at, due, [chg("membership", "basic", 1, 2500, 0, 2500, inv_at)], 2500),
        invoice("inv-3", "eve", "2026-03", inv_at, due, [chg("membership", "pro", 1, 9900, 0, 9900, inv_at)], 9900)]})
    c.ok("getAccount", "ada", inv_at, {"memberId": "ada"}, account("ada", 0, 15615, 0, 0))
    c.audit_log(inv_at)
    return c


def x_floor_4() -> Case:
    c = C("X-floor-4", "X-floor", "failed inspection locks the machine, a passed one releases it, a manual lockout stays")
    world(c)
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 30}, {"plan": plan("L1", 30, "2026-03-02", "2026-04-01", False)})
    c.ok("applyLockout", "sam", W, {"machineId": "L1", "reason": "belt"}, {"lockout": lock("lck-1", "L1", "manual", "belt", "sam", W)}, new="lck-1")
    f1 = t(3, "09:00")
    c.ok("recordInspection", "sue", f1, {"machineId": "L1", "outcome": "failed", "note": "cracked lens"},
         {"plan": plan("L1", 30, "2026-03-02", "2026-04-01", True)})
    two = [lock("lck-1", "L1", "manual", "belt", "sam", W), lock("lck-2", "L1", "inspection", "cracked lens", "sue", f1)]
    c.ok("listLockouts", "sam", f1, {"machineId": "L1"}, {"machineId": "L1", "lockouts": two})
    f2 = t(3, "10:00")
    c.ok("recordInspection", "sam", f2, {"machineId": "L1", "outcome": "failed", "note": "still cracked"},
         {"plan": plan("L1", 30, "2026-03-02", "2026-04-01", True)})
    c.ok("listLockouts", "sam", f2, {"machineId": "L1"}, {"machineId": "L1", "lockouts": two})
    p1 = t(10, "09:00")
    c.ok("recordInspection", "sue", p1, {"machineId": "L1", "outcome": "passed", "note": "ok"},
         {"plan": plan("L1", 30, "2026-03-10", "2026-04-09", False)})
    c.ok("listLockouts", "sam", p1, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "belt", "sam", W), lock("lck-2", "L1", "inspection", "cracked lens", "sue", f1, p1)]})
    ada = {"memberId": "ada", "machineId": "L1"}
    c.refuse("createReservation", "ada", p1, {**ada, "start": t(11, "10:00"), "end": t(11, "11:00")}, "machine_locked_out")
    rel = t(10, "09:30")
    c.ok("releaseLockout", "amy", rel, {"lockoutId": "lck-1"}, {"lockout": lock("lck-1", "L1", "manual", "belt", "sam", W, rel)})
    c.ok("createReservation", "ada", rel, {**ada, "start": t(11, "10:00"), "end": t(11, "11:00")},
         {"reservation": res("res-1", "ada", "L1", t(11, "10:00"), t(11, "11:00"))}, new="res-1")
    f3 = t(10, "10:00")
    c.ok("recordInspection", "sam", f3, {"machineId": "L1", "outcome": "failed", "note": "chipped"},
         {"plan": plan("L1", 30, "2026-03-10", "2026-04-09", True)})
    c.ok("listLockouts", "sam", f3, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "belt", "sam", W, rel), lock("lck-2", "L1", "inspection", "cracked lens", "sue", f1, p1),
        lock("lck-3", "L1", "inspection", "chipped", "sam", f3)]})
    c.refuse("createReservation", "ada", f3, {**ada, "start": t(11, "11:00"), "end": t(11, "12:00")}, "machine_locked_out")
    c.ok("createReservation", "ada", f3, {"memberId": "ada", "machineId": "L2", "start": t(11, "12:00"), "end": t(11, "13:00")},
         {"reservation": res("res-2", "ada", "L2", t(11, "12:00"), t(11, "13:00"))}, new="res-2")
    p2 = t(10, "10:30")
    c.ok("recordInspection", "sam", p2, {"machineId": "L1", "outcome": "passed", "note": "fixed"},
         {"plan": plan("L1", 30, "2026-03-10", "2026-04-09", False)})
    c.ok("createReservation", "ada", p2, {**ada, "start": t(11, "11:00"), "end": t(11, "12:00")},
         {"reservation": res("res-3", "ada", "L1", t(11, "11:00"), t(11, "12:00"))}, new="res-3")
    c.refuse("createReservation", "ada", p2, {**ada, "start": t(9, "10:00", 4), "end": t(9, "11:00", 4)}, "inspection_due")
    c.ok("createReservation", "ada", p2, {**ada, "start": t(8, "10:00", 4), "end": t(8, "11:00", 4)},
         {"reservation": res("res-4", "ada", "L1", t(8, "10:00", 4), t(8, "11:00", 4))}, new="res-4")
    c.ok("listLockouts", "sam", p2, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "manual", "belt", "sam", W, rel), lock("lck-2", "L1", "inspection", "cracked lens", "sue", f1, p1),
        lock("lck-3", "L1", "inspection", "chipped", "sam", f3, p2)]})
    c.audit_log(p2)
    return c


def x_floor_5() -> Case:
    c = C("X-floor-5", "X-floor", "a work order with incident and parts, closed against the lockout it holds and an inspection lockout")
    world(c)
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 30})
    c.ok("addPart", "sam", W, {"partId": "belt", "name": "Belt", "unitCostCents": 1200, "familyId": "laser"})
    c.ok("restockPart", "sam", W, {"partId": "belt", "quantity": 4})
    i0 = t(2, "10:00")
    c.ok("reportIncident", "sam", i0, {"machineId": "L1", "severity": "medium", "description": "arc fault"}, new="inc-1")
    f = t(2, "10:05")
    c.ok("recordInspection", "sue", f, {"machineId": "L1", "outcome": "failed", "note": "arc"},
         {"plan": plan("L1", 30, "2026-03-02", "2026-04-01", True)})
    o = t(2, "10:10")
    c.ok("openWorkOrder", "sue", o, {"machineId": "L1", "summary": "Replace tube"}, {
        "workOrder": wo("wko-1", "L1", "Replace tube", lck="lck-2", by="sue", opened=o)}, new="wko-1")
    k = t(2, "10:11")
    c.ok("linkIncident", "sue", k, {"workOrderId": "wko-1", "incidentId": "inc-1"}, {
        "workOrder": wo("wko-1", "L1", "Replace tube", inc="inc-1", lck="lck-2", by="sue", opened=o)})
    u = t(2, "10:15")
    c.ok("useParts", "sue", u, {"workOrderId": "wko-1", "partId": "belt", "quantity": 3}, {
        "workOrder": wo("wko-1", "L1", "Replace tube", inc="inc-1", lck="lck-2", cost=3600, by="sue", opened=o),
        "part": part("belt", "Belt", 1200, "laser", 1)})
    p = t(2, "10:20")
    c.ok("recordInspection", "sam", p, {"machineId": "L1", "outcome": "passed", "note": "fixed"},
         {"plan": plan("L1", 30, "2026-03-02", "2026-04-01", False)})
    ada = {"memberId": "ada", "machineId": "L1", "start": t(3, "10:00"), "end": t(3, "11:00")}
    c.refuse("createReservation", "ada", p, ada, "machine_locked_out")
    c.ok("listLockouts", "sam", p, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "inspection", "arc", "sue", f, p), lock("lck-2", "L1", "workOrder", "Replace tube", "sue", o)]})
    c.refuse("closeWorkOrder", "sue", t(2, "10:25"), {"workOrderId": "wko-1", "note": "done"}, "incident_open")
    r = t(2, "10:30")
    c.ok("resolveIncident", "sam", r, {"incidentId": "inc-1", "resolution": "replaced"})
    z = t(2, "10:35")
    c.ok("closeWorkOrder", "sue", z, {"workOrderId": "wko-1", "note": "done"}, {
        "workOrder": wo("wko-1", "L1", "Replace tube", "closed", "inc-1", "lck-2", 3600, "sue", o, z, "done")})
    c.ok("createReservation", "ada", z, ada, {"reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"))}, new="res-1")
    c.ok("listLockouts", "sam", z, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        lock("lck-1", "L1", "inspection", "arc", "sue", f, p), lock("lck-2", "L1", "workOrder", "Replace tube", "sue", o, z)]})
    c.ok("listWorkOrders", "sam", z, {"machineId": "L1"}, {"machineId": "L1", "workOrders": [
        wo("wko-1", "L1", "Replace tube", "closed", "inc-1", "lck-2", 3600, "sue", o, z, "done")]})
    c.ok("listIncidents", "sam", z, {}, {"incidents": [incident("inc-1", "L1", "medium", "arc fault", "sam", i0, "resolved", "replaced", r)]})
    c.ok("listParts", "sam", z, {}, {"parts": [part("belt", "Belt", 1200, "laser", 1)]})
    c.audit_log(z)
    return c


def x_floor_6() -> Case:
    c = C("X-floor-6", "X-floor", "escalate incident: work order, lockout, cancellations in order, notices; a session goes on")
    world(c)
    certify(c, "eve", W)
    certify(c, "bob", W)
    for rid, mem, mach, d, s, e in (
            ("res-1", "eve", "L1", 2, "09:30", "10:00"), ("res-2", "ada", "L1", 3, "09:00", "10:00"),
            ("res-3", "eve", "L1", 3, "14:00", "15:00"), ("res-4", "bob", "L1", 3, "11:00", "12:00"),
            ("res-5", "eve", "L1", 3, "10:00", "11:00"), ("res-6", "bob", "L1", 3, "15:00", "16:00"),
            ("res-7", "bob", "L2", 3, "12:00", "13:00")):
        c.ok("createReservation", mem, W, {"memberId": mem, "machineId": mach, "start": t(d, s), "end": t(d, e)},
             {"reservation": res(rid, mem, mach, t(d, s), t(d, e))}, new=rid)
    c.ok("cancelReservation", "bob", W, {"reservationId": "res-6"})
    c.ok("setEscalationPolicy", "amy", W, {"familyId": "laser", "minSeverity": "medium"})
    c.ok("checkIn", "ada", t(3, "09:00"), {"reservationId": "res-2"})
    for iid, mach, sev, desc, at in (("inc-1", "L1", "medium", "fire", "09:30"), ("inc-2", "L1", "low", "scratch", "09:31"),
                                     ("inc-3", "T1", "medium", "jam", "09:32"), ("inc-4", "T1", "low", "rattle", "09:33")):
        c.ok("reportIncident", "sam", t(3, at), {"machineId": mach, "severity": sev, "description": desc}, new=iid)
    c.ok("resolveIncident", "sam", t(3, "09:34"), {"incidentId": "inc-4", "resolution": "ignore"})
    q = t(3, "09:40")
    c.refuse("escalateIncident", "sue", q, {"incidentId": "inc-2"}, "below_policy")
    c.refuse("escalateIncident", "sue", q, {"incidentId": "inc-3"}, "no_policy")
    c.refuse("escalateIncident", "sue", q, {"incidentId": "inc-4"}, "invalid_state")
    c.refuse("escalateIncident", "ada", q, {"incidentId": "inc-1"}, "forbidden")
    c.ok("listWorkOrders", "sam", q, {"machineId": "L1"}, {"machineId": "L1", "workOrders": []})
    c.ok("listNotices", "sam", q, {"memberId": "bob"}, {"memberId": "bob", "notices": []})
    n = t(3, "10:00")
    c.ok("escalateIncident", "sue", n, {"incidentId": "inc-1"}, {"escalation": {
        "escalationId": "esc-1", "incidentId": "inc-1", "machineId": "L1", "workOrderId": "wko-1",
        "cancelledReservationIds": ["res-4", "res-3"], "at": n}}, new="esc-1")
    c.ok("listNotices", "bob", n, {"memberId": "bob"}, {"memberId": "bob", "notices": [
        notice("ntc-1", "bob", "cancellation", "Reservation res-4 was cancelled", "res-4", n)]})
    c.ok("listNotices", "eve", n, {"memberId": "eve"}, {"memberId": "eve", "notices": [
        notice("ntc-2", "eve", "cancellation", "Reservation res-3 was cancelled", "res-3", n)]})
    c.ok("listNotices", "ada", n, {"memberId": "ada"}, {"memberId": "ada", "notices": []})
    c.ok("listWorkOrders", "sam", n, {"machineId": "L1"}, {"machineId": "L1", "workOrders": [
        wo("wko-1", "L1", "Escalated incident inc-1", inc="inc-1", lck="lck-1", by="sue", opened=n)]})
    c.ok("listLockouts", "sam", n, {"machineId": "L1"}, {"machineId": "L1", "lockouts": [
        has(lockoutId="lck-1", machineId="L1", source="workOrder", appliedBy="sue", appliedAt=n, releasedAt=None)]})
    c.ok("listMemberReservations", "sam", n, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-2", "ada", "L1", t(3, "09:00"), t(3, "10:00"), "checkedIn", cin=t(3, "09:00"), rate=45)]})
    c.ok("listMemberReservations", "sam", n, {"memberId": "eve"}, {"memberId": "eve", "reservations": [
        res("res-1", "eve", "L1", t(2, "09:30"), t(2, "10:00"), "missed"),
        res("res-5", "eve", "L1", t(3, "10:00"), t(3, "11:00")),
        res("res-3", "eve", "L1", t(3, "14:00"), t(3, "15:00"), "cancelled")]})
    c.ok("listMemberReservations", "sam", n, {"memberId": "bob"}, {"memberId": "bob", "reservations": [
        res("res-4", "bob", "L1", t(3, "11:00"), t(3, "12:00"), "cancelled"),
        res("res-7", "bob", "L2", t(3, "12:00"), t(3, "13:00")),
        res("res-6", "bob", "L1", t(3, "15:00"), t(3, "16:00"), "cancelled")]})
    c.ok("listIncidents", "sam", n, {}, {"incidents": [
        incident("inc-1", "L1", "medium", "fire", "sam", t(3, "09:30")),
        incident("inc-2", "L1", "low", "scratch", "sam", t(3, "09:31")),
        incident("inc-3", "T1", "medium", "jam", "sam", t(3, "09:32")),
        incident("inc-4", "T1", "low", "rattle", "sam", t(3, "09:33"), "resolved", "ignore", t(3, "09:34"))]})
    c.refuse("checkIn", "eve", n, {"reservationId": "res-5"}, "machine_locked_out")
    c.refuse("createReservation", "eve", n, {"memberId": "eve", "machineId": "L1", "start": t(4, "10:00"), "end": t(4, "11:00")}, "machine_locked_out")
    c.ok("createReservation", "eve", n, {"memberId": "eve", "machineId": "L2", "start": t(4, "10:00"), "end": t(4, "11:00")},
         {"reservation": res("res-8", "eve", "L2", t(4, "10:00"), t(4, "11:00"))}, new="res-8")
    c.refuse("escalateIncident", "sue", n, {"incidentId": "inc-1"}, "already_escalated")
    # the session that is checked in goes on: no refusal at check-out, metering to 10:20 = 80 minutes
    c.ok("checkOut", "ada", t(3, "10:20"), {"reservationId": "res-2"}, {
        "reservation": res("res-2", "ada", "L1", t(3, "09:00"), t(3, "10:00"), "completed", t(3, "09:00"), t(3, "10:20"), 45, 80, 3240),
        "charge": chg("usage", "res-2", 80, 45, 10, 3240, t(3, "10:20"))})
    c.ok("listEscalations", "sam", t(3, "10:20"), {}, {"escalations": [{
        "escalationId": "esc-1", "incidentId": "inc-1", "machineId": "L1", "workOrderId": "wko-1",
        "cancelledReservationIds": ["res-4", "res-3"], "at": n}]})
    c.audit_log(t(3, "10:20"))
    return c


def x_floor_7() -> Case:
    c = C("X-floor-7", "X-floor", "check eligibility agrees with the answers of create reservation, in several states")
    world(c)

    def create(member, mach, day, code=None, rid=None, month=3, year=2026):
        args = {"memberId": member, "machineId": mach, "start": t(day, "10:00", month, year), "end": t(day, "11:00", month, year)}
        if code:
            c.refuse("createReservation", member, W, args, code)
        else:
            c.ok("createReservation", member, W, args, {"reservation": res(rid, member, mach, args["start"], args["end"])}, new=rid)

    def look(mach, date, blockers, member="eve"):
        c.ok("checkEligibility", "sam", W, {"memberId": member, "machineId": mach, "date": date}, elig(member, mach, date, blockers))

    look("L1", "2026-03-03", ["not_certified"])
    create("eve", "L1", 3, "not_certified")
    c.ok("declareMinor", "sam", W, {"memberId": "eve", "birthDate": "2008-03-10", "guardianId": "ada"})
    look("L1", "2026-03-09", ["guardian_consent_missing", "not_certified"])
    create("eve", "L1", 9, "guardian_consent_missing")
    look("L1", "2026-03-10", ["not_certified"])
    create("eve", "L1", 10, "not_certified")
    c.ok("setMemberStanding", "sam", W, {"memberId": "eve", "standing": "suspended"})
    look("L1", "2026-03-09", ["member_suspended", "guardian_consent_missing", "not_certified"])
    create("eve", "L1", 9, "member_suspended")
    c.ok("setMemberStanding", "sam", W, {"memberId": "eve", "standing": "good"})
    c.ok("publishWaiver", "amy", W, {"familyId": "laser", "text": "w"})
    c.ok("applyLockout", "sam", W, {"machineId": "L1", "reason": "belt"}, new="lck-1")
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 10}, {"plan": plan("L1", 10, "2026-03-02", "2026-03-12", False)})
    look("L1", "2026-03-09", ["guardian_consent_missing", "not_certified", "waiver_missing", "machine_locked_out"])
    look("L1", "2026-03-11", ["not_certified", "waiver_missing", "machine_locked_out"])
    look("L1", "2026-03-12", ["not_certified", "waiver_missing", "machine_locked_out", "inspection_due"])
    look("L2", "2026-03-12", ["not_certified", "waiver_missing"])
    create("eve", "L1", 11, "not_certified")
    c.ok("grantConsent", "ada", W, {"guardianId": "ada", "memberId": "eve", "familyId": "laser"})
    look("L1", "2026-03-09", ["not_certified", "waiver_missing", "machine_locked_out"])
    certify(c, "eve", W)
    look("L1", "2026-03-11", ["waiver_missing", "machine_locked_out"])
    create("eve", "L1", 11, "waiver_missing")
    c.ok("signWaiver", "eve", W, {"memberId": "eve", "familyId": "laser"})
    look("L1", "2026-03-11", ["machine_locked_out"])
    create("eve", "L1", 11, "machine_locked_out")
    c.ok("releaseLockout", "sam", W, {"lockoutId": "lck-1"})
    look("L1", "2026-03-11", [])
    create("eve", "L1", 11, None, "res-1")
    look("L1", "2026-03-12", ["inspection_due"])
    create("eve", "L1", 12, "inspection_due")
    look("L1", "2026-03-01", ["not_certified"])
    # the rules of the slot are not judged by the eligibility check
    c.ok("scheduleMaintenance", "sam", W, {"machineId": "L1", "start": t(11, "15:00"), "end": t(11, "16:00"), "reason": "service"}, new="mnt-1")
    look("L1", "2026-03-11", [])
    c.refuse("createReservation", "eve", W, {"memberId": "eve", "machineId": "L1", "start": t(11, "15:00"), "end": t(11, "16:00")}, "machine_in_maintenance")
    look("L2", "2026-03-11", [])
    create("eve", "L2", 11, "member_double_booked")
    # the certification is valid on 2027-03-01 and not on its expiry date 2027-03-02
    look("L2", "2027-03-01", [])
    create("eve", "L2", 1, None, "res-2", 3, 2027)
    look("L2", "2027-03-02", ["not_certified"])
    create("eve", "L2", 2, "not_certified", None, 3, 2027)
    c.ok("setSupervision", "amy", W, {"familyId": "laser", "required": True})
    look("L1", "2026-03-11", [])
    c.refuse("createReservation", "eve", W, {"memberId": "eve", "machineId": "L1", "start": t(11, "12:00"), "end": t(11, "13:00")}, "supervisor_required")
    c.audit_log(W)
    return c


def build_cases() -> list:
    makers = [b_lockouts_1, b_lockouts_2, b_inspections_1, b_inspections_2, b_parts_1, b_parts_2, b_consumables_1, b_consumables_2,
              b_workorders_1, b_workorders_2, b_escalations_1, b_escalations_2, b_compliance_1, b_compliance_2, b_bookings_1, b_bookings_2,
              x_floor_1, x_floor_2, x_floor_3, x_floor_4, x_floor_5, x_floor_6, x_floor_7]
    return [make().build() for make in makers]
