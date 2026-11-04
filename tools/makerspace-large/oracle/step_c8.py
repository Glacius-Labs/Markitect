"""Change step C8 of the large Makerspace: shifts and supervision are removed.

The operations `addShift`, `cancelShift`, `listShifts` and `setSupervision` are gone and answer `invalidRequest`, with the shift and
supervision records, the identifier `shf-N` and the refusal codes `not_staff`, `shift_needed`, `shift_overlap`, `shift_too_long`
and `supervisor_required`. No family is supervised, so `createReservation` loses its refusal 12 and `machine_in_maintenance` is
followed directly by `machine_booked`. Nobody needs a shift for anything: `approver_not_on_shift` is gone, so `decideVolunteerTime`
(`invalid_state`, then `volunteer_cap_reached`) and `borrowTool` (`member_suspended` first) work for staff with no shift and keep
the order of their other refusals. The role `staff` and `registerUser` stay. Written from the requirements document after the
change, with every expected value stated by hand.

What this step retires or moves. Retired, because they test shifts or supervision themselves: the two staffing cases, the case of
supervision covered by two shifts, the quarter-hour shifts that cover a slot, and the case on the shift that borrowing and deciding
need. `transform` changes the other cases that only set a shift or supervision up so that something else can happen:
  - every request of `addShift`, `cancelShift`, `listShifts` and `setSupervision` goes, and with it its entry in every expected audit
    log, with the `seq` of the entries after it counted again (the numbers of the requests keep their gaps);
  - every refusal `supervisor_required` and `approver_not_on_shift` goes (a refusal changes nothing), except one step of
    `C6-gate-rules-1`, where staff with no shift borrows for a member whose certification is revoked and so is now refused
    `not_certified`; the shift identifiers `shf-N` stand in no other case;
  - in `X-floor-7` the eligibility check that was only there to stand beside the refusal `supervisor_required` goes with it;
  - the cite of a case that names a shift, a supervision or `approver_not_on_shift` is rewritten.
The generated cases of the removed operations go by themselves.

How the cases are cut: the four old names answering `invalidRequest` (`C8-old`), slots accepted with no shift on any family
(`C8-free`), the refusals of `createReservation` after `machine_in_maintenance` in order (`C8-order`), deciding volunteer time with no
shift and the order of its refusals (`C8-volunteer`), lending, renewing and returning a tool with no shift and the order of
the refusals of `borrowTool` after `tool_on_loan` (`C8-loan`), and what must stay (`C8-keep`).

Not tested because the requirements leave it undecided or the code cannot tell: the words of the refusal reasons; that the module
`staffing` and the capability "active reservations of all machines of a family" are gone, and that `shf-N` is no identifier, since
nothing in a response shows them.

Dates the cases rely on: 2026-03-02 is a Monday (W, the world's time), 03-08 a Sunday. Opening hours are 09:00 to 21:00 Monday
to Friday, 10:00 to 18:00 on Saturday. The world's rates: L1 45, L2 60, T1 30 cents a minute. Tiers: ada plus, bob basic, eve pro.
Joining the laser course costs 5000 cents (a `course` charge at W), so a member certified in a case has 5000 cents unbilled before
anything else happens. A volunteer entry at a rate of 60 cents an hour is paid one cent a minute. A tool is named `Tool <id>`.
"""
from __future__ import annotations

from typing import Any

from kit import Case, world

W = "2026-03-02T09:00Z"  # Monday, the world's time

RETIRED: dict[str, str] = {
    "B-staffing-1": "tests add shift, its refusals not_staff, shift_overlap and shift_too_long, and the shift records: the feature is removed",
    "B-staffing-2": "tests list shifts, cancel shift and set supervision, and that supervision is not judged at check-in: the feature is removed",
    "X-people-4": "tests supervision covered by two shifts, supervisor_required and shift_needed: the feature is removed",
    "C1-guards-2": "tests that shifts of 15 minutes cover a quarter-hour slot and supervisor_required for an uncovered quarter hour: removed",
    "C5-gate-shift-1": "tests that staff need a shift of their own for borrow tool and decide volunteer time: nobody needs a shift any more",
}

SHIFT_OPS = {"addShift", "cancelShift", "listShifts", "setSupervision"}
GONE_CODES = {"approver_not_on_shift", "supervisor_required"}
# A refusal that is not removed but now has the code that the next check gives: (case, old code) -> new code.
CONVERTED = {("C6-gate-rules-1", "approver_not_on_shift"): "not_certified"}
# Steps that only stood beside a removed one: request id -> its op.
ALSO_REMOVED = {"X-floor-7-0061": "checkEligibility"}
CITES = {
    "B-volunteering-2": [("the shift of a staff approver, the cap", "staff and admin decide at any time, the cap")],
    "X-people-6": [("volunteering, staffing, billing", "volunteering, billing")],
    "C5-order-1": [(", while the staff actor is off shift; the refusals come approver_not_on_shift, member_suspended,",
                    "; the refusals come member_suspended,")],
    "C5-gate-machine-1": [("an inspection due, supervision required and the badge outside", "an inspection due and the badge outside")],
    "C5-gate-standing-2": [(", staff without a shift returning", ", staff returning")],
    "C5-return-1": [("staff without a shift return;", "staff return;")],
    "C6-gate-rules-1": [
        ("certification, consent, waiver and shift are not judged", "certification, consent and waiver are not judged"),
        ("unsigned, the minor's consent withdrawn and staff without a shift, the member", "unsigned and the minor's consent withdrawn, the member"),
        ("not_certified, guardian_consent_missing and approver_not_on_shift)", "not_certified and guardian_consent_missing, also for staff)"),
    ],
}


def transform(case: dict[str, Any]) -> dict[str, Any]:
    """The case without the shifts and the supervision it only set up, with the audit log it expects counted again."""
    cid = case["id"]
    steps = []
    for step in case["steps"]:
        request, expect = step.get("request"), step["expect"]
        if request is None:  # a raw line of a protocol case
            steps.append(step)
            continue
        if request["op"] in SHIFT_OPS:
            continue
        if request["id"] in ALSO_REMOVED:
            assert request["op"] == ALSO_REMOVED[request["id"]], request["id"]
            continue
        if expect["status"] == "refused" and expect.get("code") in GONE_CODES:
            code = CONVERTED.get((cid, expect["code"]))
            if code is None:
                continue
            expect["code"] = code
        steps.append(step)
    case["steps"] = steps
    for step in steps:
        result = step["expect"].get("result")
        if "request" in step and step["request"]["op"] == "listAuditEntries" and isinstance(result, dict) and isinstance(result.get("entries"), list):
            kept = [e for e in result["entries"] if e["op"] not in SHIFT_OPS]
            result["entries"] = [{**e, "seq": n} for n, e in enumerate(kept, 1)]
    for cite in case["cite"]:
        for old, new in CITES.get(cid, []):
            assert old in cite["text"], (cid, old)
            cite["text"] = cite["text"].replace(old, new)
    return case


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


# ---- record builders (the members of each record, from the requirements) ---------------------------------------------

def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def plan(mach, days, last, due):
    return {"plan": {"machineId": mach, "intervalDays": days, "lastPassedOn": last, "nextDueOn": due, "failed": False}}


def tl(tid: str, fee: int, on: bool = False) -> dict[str, Any]:
    return {"toolId": tid, "name": f"Tool {tid}", "familyId": "laser", "lateFeeCentsPerDay": fee, "onLoan": on}


def ln(lid, tid, mem, borrowed, due, returned=None, renewals=0) -> dict[str, Any]:
    return {"loanId": lid, "toolId": tid, "memberId": mem, "borrowedAt": borrowed, "dueOn": due, "returnedAt": returned,
            "status": "active" if returned is None else "returned", "overdue": False, "chargedDays": 0, "renewals": renewals}


def entry(eid, mem, date, minutes, note, status="pending", credit=None):
    return {"entry": {"entryId": eid, "memberId": mem, "date": date, "minutes": minutes, "note": note, "status": status,
                      "creditCents": credit}}


def account(mem, credit=0, unbilled=0):
    return {"account": {"memberId": mem, "creditBalanceCents": credit, "openBalanceCents": 0, "overdueBalanceCents": 0,
                        "unbilledCents": unbilled}}


def add(c: Case, tid: str, fee: int) -> None:
    c.ok("addTool", "sam", W, {"toolId": tid, "name": f"Tool {tid}", "familyId": "laser", "lateFeeCentsPerDay": fee}, {"tool": tl(tid, fee)})


def lend(c: Case, actor: str, at: str, mem: str, tid: str, fee: int, lid: str, due: str) -> None:
    c.ok("borrowTool", actor, at, {"memberId": mem, "toolId": tid}, {"loan": ln(lid, tid, mem, at, due), "tool": tl(tid, fee, True)}, new=lid)


def certify(c: Case, mem: str, at: str = W) -> None:
    c.ok("enrollInCourse", "sam", at, {"courseId": "laser-101", "memberId": mem})
    c.ok("completeCourse", "sam", at, {"courseId": "laser-101", "memberId": mem})


def book(c: Case, actor: str, rid: str, mem: str, mach: str, start: str, end: str, at: str = W) -> None:
    c.ok("createReservation", actor, at, {"memberId": mem, "machineId": mach, "start": start, "end": end},
         {"reservation": res(rid, mem, mach, start, end)}, new=rid)


# ---- C8-old: the old names are gone ------------------------------------------------------------------------------------------

def old_1() -> Case:
    c = Case("C8-old-1", "C8-old", "Invalid requests: `addShift`, `cancelShift` and `listShifts` name no operation any more, so each is answered invalidRequest, "
                                   "with the arguments and the actors that were right before, for a member, for a user that does not exist and with an argument "
                                   "missing (not invalid_argument); they change nothing, make no shift, take no number and cause no audit entry")
    world(c)
    shift = {"userId": "sue", "start": t(3, "09:00"), "end": t(3, "17:00")}
    c.invalid("addShift", "sam", W, shift)
    c.invalid("addShift", "amy", W, {"userId": "sam", "start": t(3, "09:00"), "end": t(3, "21:00")})
    c.invalid("addShift", "ada", W, shift)
    c.invalid("addShift", "ghost", W, shift)
    c.invalid("addShift", "sam", W, {"userId": "sue", "start": t(3, "09:00")})
    c.invalid("cancelShift", "sam", W, {"shiftId": "shf-1"})
    c.invalid("cancelShift", "amy", W, {"shiftId": "shf-1"})
    c.invalid("cancelShift", "sam", W, {})
    c.invalid("listShifts", "sam", W, {"date": "2026-03-03"})
    c.invalid("listShifts", "amy", W, {"date": "2026-03-03"})
    c.invalid("listShifts", "ada", W, {"date": "2026-03-03"})
    c.invalid("listShifts", "sam", W, {"date": "not a date"})
    book(c, "ada", "res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"))
    c.audit_log(W)
    return c


def old_2() -> Case:
    c = Case("C8-old-2", "C8-old", "Invalid requests: `setSupervision` names no operation any more, so it is answered invalidRequest, from the admin and from staff, "
                                   "for a family that exists and one that does not, with the flag true, false or of the wrong type and with an argument missing "
                                   "(not not_found, forbidden or invalid_argument); no family is supervised, so a slot is accepted with no shift at all")
    world(c)
    c.invalid("setSupervision", "amy", W, {"familyId": "laser", "required": True})
    c.invalid("setSupervision", "amy", W, {"familyId": "laser", "required": False})
    c.invalid("setSupervision", "sam", W, {"familyId": "laser", "required": True})
    c.invalid("setSupervision", "ada", W, {"familyId": "laser", "required": True})
    c.invalid("setSupervision", "amy", W, {"familyId": "nope", "required": True})
    c.invalid("setSupervision", "amy", W, {"familyId": "laser", "required": "yes"})
    c.invalid("setSupervision", "amy", W, {"familyId": "laser"})
    c.invalid("setSupervision", "ghost", W, {"familyId": "lathe", "required": True})
    book(c, "ada", "res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"))
    c.audit_log(W)
    return c


# ---- C8-free: no family is supervised -----------------------------------------------------------------------------------------

def free_1() -> Case:
    c = Case("C8-free-1", "C8-free", "Create reservation, Reservation rules: setSupervision is invalidRequest, and with no shift at all a slot on a laser machine is accepted: on a Tuesday morning, on a "
                                     "Saturday until closing, on a Monday a year ahead, by the member, by staff and by the admin; no reservation is refused "
                                     "for lack of cover")
    world(c)
    c.invalid("setSupervision", "amy", W, {"familyId": "laser", "required": True})
    book(c, "ada", "res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00"))
    book(c, "ada", "res-2", "ada", "L2", t(7, "17:00"), t(7, "18:00"))
    book(c, "ada", "res-3", "ada", "L2", t(1, "10:00", 3, 2027), t(1, "11:00", 3, 2027))
    book(c, "sam", "res-4", "ada", "L2", t(4, "09:00"), t(4, "09:15"))
    book(c, "amy", "res-5", "ada", "L1", t(5, "20:00"), t(5, "21:00"))
    c.ok("listMemberReservations", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "10:00"), t(3, "11:00")), res("res-4", "ada", "L2", t(4, "09:00"), t(4, "09:15")),
        res("res-5", "ada", "L1", t(5, "20:00"), t(5, "21:00")), res("res-2", "ada", "L2", t(7, "17:00"), t(7, "18:00")),
        res("res-3", "ada", "L2", t(1, "10:00", 3, 2027), t(1, "11:00", 3, 2027))]})
    c.audit_log(W)
    return c


def free_2() -> Case:
    c = Case("C8-free-2", "C8-free", "Create reservation, Check in, Eligibility: setSupervision is invalidRequest, so on the lathe family too a slot with no shift is "
                                     "accepted and the member checks in at the start, and the eligibility check agrees: no blocker for a certified member, "
                                     "not_certified alone for one who is not")
    world(c)
    c.invalid("setSupervision", "amy", W, {"familyId": "lathe", "required": True})
    c.ok("addCourse", "sam", W, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 3000, "capacity": 2})
    c.ok("enrollInCourse", "sam", W, {"courseId": "lathe-101", "memberId": "eve"})
    c.ok("completeCourse", "sam", W, {"courseId": "lathe-101", "memberId": "eve"})
    c.ok("checkEligibility", "sam", W, {"memberId": "eve", "machineId": "T1", "date": "2026-03-03"}, {
        "memberId": "eve", "machineId": "T1", "date": "2026-03-03", "eligible": True, "blockers": []})
    c.ok("checkEligibility", "sam", W, {"memberId": "bob", "machineId": "T1", "date": "2026-03-03"}, {
        "memberId": "bob", "machineId": "T1", "date": "2026-03-03", "eligible": False, "blockers": ["not_certified"]})
    book(c, "eve", "res-1", "eve", "T1", t(3, "12:00"), t(3, "13:00"))
    c.ok("checkIn", "eve", t(3, "12:00"), {"reservationId": "res-1"}, {"reservation": res(
        "res-1", "eve", "T1", t(3, "12:00"), t(3, "13:00"), "checkedIn", cin=t(3, "12:00"), rate=30)})
    c.audit_log(t(3, "12:00"))
    return c


# ---- C8-order: the refusals of createReservation after the gates that stay ------------------------------------------------------

def order_1() -> Case:
    c = Case("C8-order-1", "C8-order", "Create reservation, refusals in order, with setSupervision invalidRequest: for one slot that is in a maintenance window, overlaps another member's reservation of the "
                                       "machine and the member's own reservation of another machine, and on a machine whose inspection is due, the refusals come "
                                       "inspection_due, machine_in_maintenance, machine_booked, member_double_booked as each is cured, with no refusal between "
                                       "machine_in_maintenance and machine_booked; the slot is then accepted")
    world(c)
    certify(c, "eve")
    c.invalid("setSupervision", "amy", W, {"familyId": "laser", "required": True})
    s, e = t(3, "11:00"), t(3, "13:00")
    book(c, "eve", "res-1", "eve", "L1", t(3, "10:00"), t(3, "11:30"))
    book(c, "ada", "res-2", "ada", "L2", t(3, "11:00"), t(3, "12:00"))
    c.ok("scheduleMaintenance", "sam", W, {"machineId": "L1", "start": t(3, "12:00"), "end": t(3, "13:00"), "reason": "service"}, new="mnt-1")
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 1}, plan("L1", 1, "2026-03-02", "2026-03-03"))
    slot = {"memberId": "ada", "machineId": "L1", "start": s, "end": e}
    c.refuse("createReservation", "ada", W, slot, "inspection_due")
    c.ok("setInspectionPlan", "sam", W, {"machineId": "L1", "intervalDays": 30}, plan("L1", 30, "2026-03-02", "2026-04-01"))
    c.refuse("createReservation", "ada", W, slot, "machine_in_maintenance")
    c.ok("cancelMaintenance", "sam", W, {"windowId": "mnt-1"})
    c.refuse("createReservation", "ada", W, slot, "machine_booked")
    c.ok("cancelReservation", "eve", W, {"reservationId": "res-1"})
    c.refuse("createReservation", "ada", W, slot, "member_double_booked")
    c.ok("cancelReservation", "ada", W, {"reservationId": "res-2"})
    c.ok("createReservation", "ada", W, slot, {"reservation": res("res-3", "ada", "L1", s, e)}, new="res-3")
    c.audit_log(W)
    return c


# ---- C8-volunteer: deciding volunteer time with no shift ----------------------------------------------------------------------

def volunteer_1() -> Case:
    c = Case("C8-volunteer-1", "C8-volunteer", "Decide volunteer time: staff with no shift decide, on a Monday morning and in the small hours of a Sunday; the approval credits the "
                                               "member and posts to the ledger as before, a rejection leaves the credit null; there is no refusal for being off shift")
    world(c)
    c.ok("setVolunteerRate", "amy", W, {"centsPerHour": 1500}, {"centsPerHour": 1500})
    c.ok("logVolunteerTime", "ada", W, {"memberId": "ada", "date": "2026-03-02", "minutes": 100, "note": "Bench build"},
         entry("vol-1", "ada", "2026-03-02", 100, "Bench build"), new="vol-1")
    c.ok("logVolunteerTime", "ada", W, {"memberId": "ada", "date": "2026-03-01", "minutes": 30, "note": "Sweeping"},
         entry("vol-2", "ada", "2026-03-01", 30, "Sweeping"), new="vol-2")
    c.ok("decideVolunteerTime", "sam", W, {"entryId": "vol-1", "decision": "approved"},
         entry("vol-1", "ada", "2026-03-02", 100, "Bench build", "approved", 2500))
    c.ok("getAccount", "ada", W, {"memberId": "ada"}, account("ada", credit=2500, unbilled=5000))
    n = t(8, "03:00")
    c.ok("decideVolunteerTime", "sue", n, {"entryId": "vol-2", "decision": "rejected"},
         entry("vol-2", "ada", "2026-03-01", 30, "Sweeping", "rejected", None))
    c.ok("listVolunteerTime", "sam", n, {"memberId": "ada"}, {"memberId": "ada", "entries": [
        entry("vol-1", "ada", "2026-03-02", 100, "Bench build", "approved", 2500)["entry"],
        entry("vol-2", "ada", "2026-03-01", 30, "Sweeping", "rejected", None)["entry"]]})
    c.ok("listJournalEntries", "amy", n, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        {"entryId": "jrn-1", "at": W, "debitAccount": "expenseVolunteer", "creditAccount": "creditLiability", "amountCents": 2500, "memo": "vol-1"}]})
    c.audit_log(n)
    return c


def volunteer_2() -> Case:
    c = Case("C8-volunteer-2", "C8-volunteer", "Decide volunteer time, refusals in order: with no shift, an entry that is not pending is invalid_state, also when approving it "
                                               "would pass the cap of 1200 minutes; a pending entry that would pass the cap is volunteer_cap_reached; reaching 1200 "
                                               "exactly is allowed; a rejection is never held by the cap")
    world(c)
    c.ok("setVolunteerRate", "amy", W, {"centsPerHour": 60}, {"centsPerHour": 60})
    for eid, minutes in (("vol-1", 720), ("vol-2", 480), ("vol-3", 1)):
        c.ok("logVolunteerTime", "ada", W, {"memberId": "ada", "date": "2026-03-01", "minutes": minutes, "note": f"n{minutes}"},
             entry(eid, "ada", "2026-03-01", minutes, f"n{minutes}"), new=eid)
    c.ok("decideVolunteerTime", "sam", W, {"entryId": "vol-1", "decision": "approved"}, entry("vol-1", "ada", "2026-03-01", 720, "n720", "approved", 720))
    c.ok("decideVolunteerTime", "sam", W, {"entryId": "vol-2", "decision": "approved"}, entry("vol-2", "ada", "2026-03-01", 480, "n480", "approved", 480))
    c.refuse("decideVolunteerTime", "sue", W, {"entryId": "vol-3", "decision": "approved"}, "volunteer_cap_reached")
    c.refuse("decideVolunteerTime", "sam", W, {"entryId": "vol-1", "decision": "approved"}, "invalid_state")
    c.refuse("decideVolunteerTime", "sam", W, {"entryId": "vol-1", "decision": "rejected"}, "invalid_state")
    c.ok("decideVolunteerTime", "sue", W, {"entryId": "vol-3", "decision": "rejected"}, entry("vol-3", "ada", "2026-03-01", 1, "n1", "rejected", None))
    c.refuse("decideVolunteerTime", "sue", W, {"entryId": "vol-3", "decision": "approved"}, "invalid_state")
    c.ok("getAccount", "ada", W, {"memberId": "ada"}, account("ada", credit=1200, unbilled=5000))
    c.audit_log(W)
    return c


# ---- C8-loan: lending a tool with no shift --------------------------------------------------------------------------------------

def loan_1() -> Case:
    c = Case("C8-loan-1", "C8-loan", "Borrow tool, Renew loan, Return tool: staff with no shift lend a tool late on a Sunday and just after midnight, renew a loan and take "
                                     "a tool back in the night; nothing refuses them for a shift")
    world(c)
    certify(c, "bob")
    add(c, "drill", 200)
    add(c, "saw", 150)
    a, b = t(8, "23:30"), t(9, "00:15")
    lend(c, "sam", a, "ada", "drill", 200, "lon-1", "2026-03-22")
    lend(c, "sue", b, "bob", "saw", 150, "lon-2", "2026-03-23")
    r = t(9, "03:00")
    c.ok("renewLoan", "sue", r, {"loanId": "lon-1"}, {"loan": ln("lon-1", "drill", "ada", a, "2026-04-05", renewals=1)})
    z = t(9, "04:00")
    c.ok("returnTool", "sam", z, {"loanId": "lon-2"}, {"loan": ln("lon-2", "saw", "bob", b, "2026-03-23", returned=z), "tool": tl("saw", 150),
                                                        "charge": None})
    c.ok("listTools", "sue", z, {}, {"tools": [tl("drill", 200, True), tl("saw", 150)]})
    c.audit_log(z)
    return c


def loan_2() -> Case:
    c = Case("C8-loan-2", "C8-loan", "Borrow tool, refusals in order: staff with no shift lend, and a member who has three loans is refused loan_limit while none is late; "
                                     "with all three late the refusals are tool_on_loan for a tool on loan, loan_overdue (not loan_limit) for a free one, and "
                                     "not_certified (before both) for a lathe tool; a member with no loan is lent a free tool by the same staff")
    world(c)
    certify(c, "bob")
    for tid, fee in (("ta", 200), ("tb", 150), ("tc", 100), ("td", 50)):
        add(c, tid, fee)
    c.ok("addTool", "sam", W, {"toolId": "te", "name": "Tool te", "familyId": "lathe", "lateFeeCentsPerDay": 40}, None)
    due = "2026-03-16"
    lend(c, "sam", W, "ada", "ta", 200, "lon-1", due)
    lend(c, "sue", W, "ada", "tb", 150, "lon-2", due)
    lend(c, "sam", W, "ada", "tc", 100, "lon-3", due)
    c.refuse("borrowTool", "ada", W, {"memberId": "ada", "toolId": "td"}, "forbidden")
    c.refuse("borrowTool", "sam", W, {"memberId": "ada", "toolId": "nope"}, "not_found")
    c.refuse("borrowTool", "sam", W, {"memberId": "ada", "toolId": "td"}, "loan_limit")
    c.refuse("borrowTool", "sue", W, {"memberId": "ada", "toolId": "ta"}, "tool_on_loan")
    c.refuse("borrowTool", "sam", W, {"memberId": "ada", "toolId": "te"}, "not_certified")
    c.refuse("borrowTool", "sam", W, {"memberId": "bob", "toolId": "ta"}, "tool_on_loan")
    d = t(17, "09:00")
    c.refuse("borrowTool", "sue", d, {"memberId": "ada", "toolId": "ta"}, "tool_on_loan")
    c.refuse("borrowTool", "sam", d, {"memberId": "ada", "toolId": "td"}, "loan_overdue")
    c.refuse("borrowTool", "sue", d, {"memberId": "ada", "toolId": "te"}, "not_certified")
    lend(c, "sam", d, "bob", "td", 50, "lon-4", "2026-03-31")
    c.audit_log(d)
    return c


# ---- C8-keep: what stays --------------------------------------------------------------------------------------------------------

def keep_1() -> Case:
    c = Case("C8-keep-1", "C8-keep", "Registry, Register user, Authorization: the role `staff` stays and `registerUser` registers staff and admins, only for the admin; "
                                     "the identifier is shared with members; a new staff user has no shift and still registers a member, changes a standing and "
                                     "decides volunteer time")
    world(c)
    c.ok("registerUser", "amy", W, {"userId": "kim", "name": "Kim Staff", "role": "staff"}, {"user": {"userId": "kim", "name": "Kim Staff", "role": "staff"}})
    c.ok("registerUser", "amy", W, {"userId": "ned", "name": "Ned Admin", "role": "admin"}, {"user": {"userId": "ned", "name": "Ned Admin", "role": "admin"}})
    c.refuse("registerUser", "sam", W, {"userId": "ola", "name": "Ola Staff", "role": "staff"}, "forbidden")
    c.refuse("registerUser", "ada", W, {"userId": "ola", "name": "Ola Staff", "role": "staff"}, "forbidden")
    c.refuse("registerUser", "amy", W, {"userId": "sam", "name": "Sam Again", "role": "staff"}, "already_exists")
    c.refuse("registerUser", "amy", W, {"userId": "ada", "name": "Ada Again", "role": "staff"}, "already_exists")
    c.refuse("registerUser", "amy", W, {"userId": "zed", "name": "Zed", "role": "member"}, "invalid_argument")
    c.ok("registerMember", "kim", W, {"memberId": "joy", "name": "Joy Reed", "tier": "basic"})
    c.ok("setMemberStanding", "kim", W, {"memberId": "joy", "standing": "suspended"})
    c.ok("logVolunteerTime", "ada", W, {"memberId": "ada", "date": "2026-03-02", "minutes": 45, "note": "Labels"},
         entry("vol-1", "ada", "2026-03-02", 45, "Labels"), new="vol-1")
    c.ok("decideVolunteerTime", "kim", W, {"entryId": "vol-1", "decision": "approved"}, entry("vol-1", "ada", "2026-03-02", 45, "Labels", "approved", 0))
    c.audit_log(W)
    return c


def keep_2() -> Case:
    c = Case("C8-keep-2", "C8-keep", "Authorization: what staff may not do stays refused for them (set the volunteer rate, add a family, a machine or an adjustment) "
                                     "and what a member may not do stays refused (decide volunteer time, lend or return a tool), without any shift in the program; "
                                     "forbidden comes before not_found")
    world(c)
    c.refuse("setVolunteerRate", "sam", W, {"centsPerHour": 60}, "forbidden")
    c.refuse("addFamily", "sam", W, {"familyId": "press", "name": "Presses"}, "forbidden")
    c.refuse("addMachine", "sue", W, {"machineId": "P1", "name": "Press one", "familyId": "laser", "rateCentsPerMinute": 10}, "forbidden")
    c.refuse("postAdjustment", "sam", W, {"debitAccount": "cash", "creditAccount": "revenueMaterial", "amountCents": 5, "memo": "x"}, "forbidden")
    c.ok("setVolunteerRate", "amy", W, {"centsPerHour": 60}, {"centsPerHour": 60})
    c.ok("logVolunteerTime", "ada", W, {"memberId": "ada", "date": "2026-03-02", "minutes": 30, "note": "Sorting"},
         entry("vol-1", "ada", "2026-03-02", 30, "Sorting"), new="vol-1")
    c.refuse("decideVolunteerTime", "ada", W, {"entryId": "vol-1", "decision": "approved"}, "forbidden")
    c.refuse("decideVolunteerTime", "bob", W, {"entryId": "vol-1", "decision": "approved"}, "forbidden")
    c.refuse("decideVolunteerTime", "ada", W, {"entryId": "vol-9", "decision": "approved"}, "forbidden")
    c.refuse("decideVolunteerTime", "sam", W, {"entryId": "vol-9", "decision": "approved"}, "not_found")
    c.refuse("decideVolunteerTime", "sam", W, {"entryId": "vol-1", "decision": "pending"}, "invalid_argument")
    c.ok("decideVolunteerTime", "sam", W, {"entryId": "vol-1", "decision": "approved"}, entry("vol-1", "ada", "2026-03-02", 30, "Sorting", "approved", 30))
    add(c, "drill", 200)
    c.refuse("borrowTool", "ada", W, {"memberId": "ada", "toolId": "drill"}, "forbidden")
    lend(c, "sam", W, "ada", "drill", 200, "lon-1", "2026-03-16")
    c.refuse("returnTool", "ada", W, {"loanId": "lon-1"}, "forbidden")
    c.audit_log(W)
    return c


def build_cases() -> list[dict[str, Any]]:
    cases = [old_1(), old_2(), free_1(), free_2(), order_1(), volunteer_1(), volunteer_2(), loan_1(), loan_2(), keep_1(), keep_2()]
    return [c.build() for c in cases]
