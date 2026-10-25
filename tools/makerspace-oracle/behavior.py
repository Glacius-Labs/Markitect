"""Strict behavior cases: every result compared in full, the audit log checked at the end.

B-* cases restate W0 per business area; D-C* cases restate a change and run
from that step on. Both are rendered for the step under test, so a later
change (for example C6's certification record) shows in every case.
"""

from __future__ import annotations

from typing import Any

import facts as f
from facts import C, W, since
from builder import Case, ts
import world as w
from world import STAFF as S, T0

MON, TUE, WED, THU, FRI, SAT, SUN = (f"2026-03-0{d}" for d in range(2, 9))
T1 = "2026-03-02T09:00Z"


def case(step: str, cid: str, family: str, cites: list[dict[str, str]]) -> Case:
    return Case(step, cid, [family], cites)


def b_members(step: str) -> list[dict[str, Any]]:
    c = case(step, "B-MEMBERS-registry", "B-MEMBERS", [
        W("6.1", "Registers a staff member or an admin."), C(4, "What changes", "`registerUser` registers a staff member, an admin or a steward"), W("6.1", "creates a user with role `member`, the given tier, standing `good`, `registeredAt` equal to the request time and a credit balance of 0."),
        W("6.1", "`already_exists` when any user, including a member, has the identifier `userId`."), W("6.2", "Setting the current tier succeeds without a change."),
        W("6.2", "Setting the current standing succeeds without a change.")])
    w.people(c)
    member = lambda mid, tier, standing="good": {"member": f.member(mid, w.MEMBERS[mid][0], tier, T0, standing)}
    c.ok("getMember", S, T1, {"memberId": "ada"}, member("ada", "plus"))
    c.ok("changeTier", S, T1, {"memberId": "ada", "tier": "pro"}, member("ada", "pro"))
    c.ok("changeTier", S, T1, {"memberId": "ada", "tier": "pro"}, member("ada", "pro"))
    c.ok("setMemberStanding", S, T1, {"memberId": "bob", "standing": "suspended"}, member("bob", "basic", "suspended"))
    c.ok("setMemberStanding", S, T1, {"memberId": "bob", "standing": "suspended"}, member("bob", "basic", "suspended"))
    c.ok("setMemberStanding", "admin", T1, {"memberId": "bob", "standing": "good"}, member("bob", "basic"))
    c.refuse("registerMember", S, T1, {"memberId": S, "name": "Sam", "tier": "basic"}, "already_exists")
    c.refuse("registerMember", S, T1, {"memberId": "ada", "name": "Ada", "tier": "basic"}, "already_exists")
    c.refuse("registerUser", "admin", T1, {"userId": "ada", "name": "Ada", "role": "admin"}, "already_exists")
    c.refuse("registerUser", "admin", T1, {"userId": "admin", "name": "Other", "role": "staff"}, "already_exists")
    c.ok("registerUser", "admin", T1, {"userId": "ana", "name": "Ana", "role": "admin"}, {"user": f.user("ana", "Ana", "admin")})
    c.ok("registerMember", "ana", T1, {"memberId": "Ada", "name": "Other Ada", "tier": "pro"}, {"member": f.member("Ada", "Other Ada", "pro", T1)})
    c.refuse("getMember", S, T1, {"memberId": "nobody"}, "not_found")
    c.refuse("getMember", S, T1, {"memberId": "ana"}, "not_found")
    c.ok("getMember", "ada", T1, {"memberId": "ada"}, member("ada", "pro"))
    c.audit_log()
    return [c.build()]


def b_training(step: str) -> list[dict[str, Any]]:
    c = case(step, "B-TRAINING-courses", "B-TRAINING", [
        W("7.1", "The course's `enrolledCount` is the number of its enrollments with status `enrolled`. A place becomes free when an enrollment is completed."),
        W("7.1", "`already_enrolled` when the member's enrollment in this course has status `enrolled`; `course_full` when `enrolledCount` equals `capacity`."),
        W("7.1", "It is created by the first enrollment; a later enrollment after completion reuses and resets it."),
        W("7.2", "If the member already has a certification for that family, whether valid, expired or revoked, it is replaced by the newly issued one."),
        C(6, "What changes", "still replaces an existing certification for that family, revoked or not, with a new valid one."),
        W("7.2", "Completing a course issues it: `issuedOn` is the date of the request time"),
        W("7.2", "the certification becomes revoked; its dates do not change."), W("4.5", "| `listCertifications` → `certifications` | `familyId` ascending |")])
    w.people(c)
    w.catalog(c)
    c.ok("addFamily", "admin", T0, {"familyId": "drill", "name": "Drills"}, {"family": f.family("drill", "Drills")})
    course = lambda cid, fam, title, fee, cap, n: {"course": f.course(cid, fam, title, fee, cap, n)}
    c.ok("addCourse", S, T0, {"courseId": "laser-101", "familyId": "laser", "title": "Laser basics", "feeCents": 3000, "capacity": 2},
         course("laser-101", "laser", "Laser basics", 3000, 2, 0))
    c.refuse("addCourse", S, T0, {"courseId": "laser-101", "familyId": "wood", "title": "X", "feeCents": 1, "capacity": 1}, "already_exists")
    c.refuse("addCourse", S, T0, {"courseId": "metal-101", "familyId": "metal", "title": "X", "feeCents": 1, "capacity": 1}, "not_found")
    for cid, fam, title in (("wood-101", "wood", "Lathe basics"), ("drill-101", "drill", "Drill basics")):
        c.ok("addCourse", S, T0, {"courseId": cid, "familyId": fam, "title": title, "feeCents": 0, "capacity": 5}, course(cid, fam, title, 0, 5, 0))
    enrolled = lambda mid, n, at=T0: {"enrollment": f.enrollment("laser-101", mid, "enrolled", at),
                                      **course("laser-101", "laser", "Laser basics", 3000, 2, n)}
    c.ok("enrollInCourse", S, T0, {"courseId": "laser-101", "memberId": "ada"}, enrolled("ada", 1))
    c.ok("enrollInCourse", S, T0, {"courseId": "laser-101", "memberId": "bob"}, enrolled("bob", 2))
    c.refuse("enrollInCourse", S, T0, {"courseId": "laser-101", "memberId": "cy"}, "course_full")
    c.refuse("enrollInCourse", S, T0, {"courseId": "laser-101", "memberId": "ada"}, "already_enrolled")
    cert = lambda mid, fam, status, issued=w.ISSUED, expires=w.EXPIRES: f.certification(step, mid, fam, issued, expires, status)
    c.ok("completeCourse", S, T1, {"courseId": "laser-101", "memberId": "ada"},
         {"enrollment": f.enrollment("laser-101", "ada", "completed", T0, T1), "certification": cert("ada", "laser", "valid")})
    c.refuse("completeCourse", S, T1, {"courseId": "laser-101", "memberId": "ada"}, "not_enrolled")
    c.refuse("completeCourse", S, T1, {"courseId": "laser-101", "memberId": "cy"}, "not_enrolled")
    c.ok("enrollInCourse", S, T1, {"courseId": "laser-101", "memberId": "cy"}, enrolled("cy", 2, T1))
    c.ok("completeCourse", S, T1, {"courseId": "laser-101", "memberId": "bob"},
         {"enrollment": f.enrollment("laser-101", "bob", "completed", T0, T1), "certification": cert("bob", "laser", "valid")})
    t2 = "2026-03-02T10:00Z"
    c.ok("enrollInCourse", S, t2, {"courseId": "laser-101", "memberId": "ada"}, enrolled("ada", 2, t2))
    for cid, fam in (("wood-101", "wood"), ("drill-101", "drill")):
        c.ok("enrollInCourse", S, t2, {"courseId": cid, "memberId": "ada"},
             {"enrollment": f.enrollment(cid, "ada", "enrolled", t2), **course(cid, fam, {"wood": "Lathe basics", "drill": "Drill basics"}[fam], 0, 5, 1)})
        c.ok("completeCourse", S, t2, {"courseId": cid, "memberId": "ada"},
             {"enrollment": f.enrollment(cid, "ada", "completed", t2, t2), "certification": cert("ada", fam, "valid")})
    c.ok("revokeCertification", S, t2, {"memberId": "ada", "familyId": "wood"}, {"certification": cert("ada", "wood", "revoked")})
    c.refuse("revokeCertification", S, t2, {"memberId": "ada", "familyId": "wood"}, "not_certified")
    c.refuse("revokeCertification", S, t2, {"memberId": "bob", "familyId": "wood"}, "not_certified")
    c.ok("listCertifications", S, t2, {"memberId": "ada"},
         {"memberId": "ada", "certifications": [cert("ada", "drill", "valid"), cert("ada", "laser", "valid"), cert("ada", "wood", "revoked")]})
    c.ok("listCertifications", S, t2, {"memberId": "cy"}, {"memberId": "cy", "certifications": []})
    c.ok("enrollInCourse", S, t2, {"courseId": "wood-101", "memberId": "ada"},
         {"enrollment": f.enrollment("wood-101", "ada", "enrolled", t2), **course("wood-101", "wood", "Lathe basics", 0, 5, 1)})
    c.ok("completeCourse", S, t2, {"courseId": "wood-101", "memberId": "ada"},
         {"enrollment": f.enrollment("wood-101", "ada", "completed", t2, t2), "certification": cert("ada", "wood", "valid")})
    t3 = "2026-03-03T09:00Z"
    c.ok("enrollInCourse", S, t2, {"courseId": "drill-101", "memberId": "cy"},
         {"enrollment": f.enrollment("drill-101", "cy", "enrolled", t2), **course("drill-101", "drill", "Drill basics", 0, 5, 1)})
    c.ok("completeCourse", S, t3, {"courseId": "drill-101", "memberId": "cy"},
         {"enrollment": f.enrollment("drill-101", "cy", "completed", t2, t3), "certification": cert("cy", "drill", "valid", "2026-03-03", "2027-03-03")})
    c.audit_log()
    return [c.build()]


def b_equipment(step: str) -> list[dict[str, Any]]:
    c = case(step, "B-EQUIPMENT-catalog", "B-EQUIPMENT", [
        W("8.1", "Each machine belongs to one family and has a rate in cents per minute."), W("4.5", "Identifiers are ordered by ASCII byte value"),
        W("8.2", "Windows may use any minute, are not limited to opening hours and may overlap each other."),
        W("8.2", "Effect: the window gets status `cancelled`."), W("4.5", "| `getMachineSchedule` → `maintenance` | `start` ascending, then creation order |")])
    w.people(c, members=())
    c.ok("addFamily", "admin", T0, {"familyId": "laser", "name": "Laser cutters"}, {"family": f.family("laser", "Laser cutters")})
    c.ok("addFamily", "admin", T0, {"familyId": "wood", "name": "Wood lathes"}, {"family": f.family("wood", "Wood lathes")})
    c.refuse("addFamily", "admin", T0, {"familyId": "laser", "name": "Other"}, "already_exists")
    machines = [("M10", "Ten", "laser", 10), ("M9", "Nine", "laser", 9), ("Z1", "Zed", "wood", 1), ("a1", "Small", "wood", 100), ("9z", "Nine zed", "laser", 10000)]
    for mid, name, fam, rate in machines:
        c.ok("addMachine", "admin", T0, {"machineId": mid, "name": name, "familyId": fam, "rateCentsPerMinute": rate}, {"machine": f.machine(mid, name, fam, rate)})
    c.refuse("addMachine", "admin", T0, {"machineId": "M9", "name": "Again", "familyId": "laser", "rateCentsPerMinute": 5}, "already_exists")
    c.refuse("addMachine", "admin", T0, {"machineId": "X1", "name": "X", "familyId": "metal", "rateCentsPerMinute": 5}, "not_found")
    c.ok("setMachineRate", "admin", T0, {"machineId": "M9", "rateCentsPerMinute": 20}, {"machine": f.machine("M9", "Nine", "laser", 20)})
    c.refuse("setMachineRate", "admin", T0, {"machineId": "X1", "rateCentsPerMinute": 5}, "not_found")
    order = {"9z": 4, "M10": 0, "M9": 1, "Z1": 2, "a1": 3}
    listed = [f.machine(*machines[order[m]][:3], 20 if m == "M9" else machines[order[m]][3]) for m in ("9z", "M10", "M9", "Z1", "a1")]
    c.ok("listMachines", S, T0, {}, {"machines": listed})
    win = lambda wid, s, e, reason, status="scheduled": {"window": f.window(wid, "M9", s, e, reason, status)}
    c.ok("scheduleMaintenance", S, T1, {"machineId": "M9", "start": ts(TUE, "23:00"), "end": ts(WED, "02:00"), "reason": "Night"},
         lambda wid: win(wid, ts(TUE, "23:00"), ts(WED, "02:00"), "Night"))
    c.ok("scheduleMaintenance", S, T1, {"machineId": "M9", "start": ts(TUE, "10:00"), "end": ts(TUE, "12:00"), "reason": "Belt"},
         lambda wid: win(wid, ts(TUE, "10:00"), ts(TUE, "12:00"), "Belt"))
    c.ok("scheduleMaintenance", S, T1, {"machineId": "M9", "start": ts(TUE, "11:00"), "end": ts(TUE, "13:00"), "reason": "Lens"},
         lambda wid: win(wid, ts(TUE, "11:00"), ts(TUE, "13:00"), "Lens"))
    c.ok("cancelMaintenance", S, T1, {"windowId": "mnt-3"}, win("mnt-3", ts(TUE, "11:00"), ts(TUE, "13:00"), "Lens", "cancelled"))
    c.refuse("cancelMaintenance", S, T1, {"windowId": "mnt-3"}, "invalid_state")
    c.refuse("cancelMaintenance", S, T1, {"windowId": "mnt-9"}, "not_found")
    c.refuse("scheduleMaintenance", S, T1, {"machineId": "M9", "start": ts(TUE, "12:00"), "end": ts(TUE, "12:00"), "reason": "R"}, "invalid_interval")
    hours = f.opening_hours(step)[1]
    c.ok("getMachineSchedule", S, T1, {"machineId": "M9", "date": TUE}, f.schedule(step, "M9", TUE, hours, [], [
        f.window("mnt-2", "M9", ts(TUE, "10:00"), ts(TUE, "12:00"), "Belt"), f.window("mnt-1", "M9", ts(TUE, "23:00"), ts(WED, "02:00"), "Night")]))
    c.audit_log()
    return [c.build()]


def b_bookings(step: str) -> list[dict[str, Any]]:
    c = case(step, "B-BOOKINGS-lifecycle", "B-BOOKINGS", [
        W("9.1", "Its stored status moves from `reserved` to `checkedIn` to `completed`, or from `reserved` to `cancelled`."),
        W("9.2", "Effect: status `checkedIn`; `checkedInAt` is the request time; `rateCentsPerMinute` is the machine's current rate."),
        W("9.2", "`meteredMinutes` is `checkedOutAt − checkedInAt` in minutes"),
        W("9.1", "the machine's reservations whose `start` falls on that date, in every status except `cancelled`"),
        W("4.5", "| `listMemberReservations` → `reservations` | `start` ascending, then creation order |"),
        W("5", "In a reservation, `checkedInAt` and `rateCentsPerMinute` are `null` until check-in; `checkedOutAt`, `meteredMinutes` and `amountCents` are `null` until check-out.")])
    w.world(c)
    r = f.reservation
    r1 = w.reserve(c, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), T1)
    r2 = w.reserve(c, "bob", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), T1)
    r3 = w.reserve(c, "ada", "L2", ts(THU, "10:00"), ts(THU, "10:30"), T1)
    r4 = w.reserve(c, "ada", "L2", ts(WED, "10:00"), ts(WED, "10:30"), T1, actor="ada")
    c.ok("cancelReservation", "bob", T1, {"reservationId": r2}, {"reservation": r(r2, "bob", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), "cancelled")})
    r5 = w.reserve(c, "cy", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), T1)
    c.refuse("cancelReservation", S, T1, {"reservationId": "res-99"}, "not_found")
    c.refuse("createReservation", S, T1, {"memberId": "ada", "machineId": "L9", "start": ts(TUE, "12:00"), "end": ts(TUE, "12:30")}, "not_found")
    c.ok("checkIn", "ada", ts(TUE, "10:00"), {"reservationId": r1},
         {"reservation": r(r1, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), "checkedIn", ts(TUE, "10:00"), None, 45)})
    c.refuse("checkIn", "ada", ts(TUE, "10:01"), {"reservationId": r1}, "invalid_state")
    out = ts(TUE, "10:37")
    c.ok("checkOut", "ada", out, {"reservationId": r1},
         {"reservation": r(r1, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), "completed", ts(TUE, "10:00"), out, 45, 37, 1499),
          "charge": f.charge("usage", r1, 37, 45, 10, out)}, cite=W("4.3", "37 minutes at 45 cents with a 10 % discount"))
    c.refuse("checkOut", "ada", out, {"reservationId": r1}, "invalid_state")
    c.refuse("cancelReservation", "ada", out, {"reservationId": r1}, "invalid_state")
    late = ts(TUE, "11:30")
    done = r(r1, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), "completed", ts(TUE, "10:00"), out, 45, 37, 1499)
    missed = r(r5, "cy", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), "missed")
    c.ok("getMachineSchedule", S, late, {"machineId": "L1", "date": TUE}, f.schedule(step, "L1", TUE, f.opening_hours(step)[1], [done, missed], []))
    c.ok("listMemberReservations", S, late, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        done, r(r4, "ada", "L2", ts(WED, "10:00"), ts(WED, "10:30")), r(r3, "ada", "L2", ts(THU, "10:00"), ts(THU, "10:30"))]})
    c.ok("listMemberReservations", "bob", late, {"memberId": "bob"}, {"memberId": "bob", "reservations": [
        r(r2, "bob", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), "cancelled")]})
    r6 = w.reserve(c, "bob", "L1", ts(WED, "10:00"), ts(WED, "10:30"), late)
    c.ok("cancelReservation", S, late, {"reservationId": r6}, {"reservation": r(r6, "bob", "L1", ts(WED, "10:00"), ts(WED, "10:30"), "cancelled")})
    r7 = w.reserve(c, "bob", "L1", ts(WED, "10:00"), ts(WED, "10:30"), late)
    c.ok("listMemberReservations", S, late, {"memberId": "bob"}, {"memberId": "bob", "reservations": [
        r(r2, "bob", "L1", ts(TUE, "11:00"), ts(TUE, "11:30"), "cancelled"), r(r6, "bob", "L1", ts(WED, "10:00"), ts(WED, "10:30"), "cancelled"),
        r(r7, "bob", "L1", ts(WED, "10:00"), ts(WED, "10:30"))]})
    c.audit_log()
    return [c.build()]


def b_materials(step: str) -> list[dict[str, Any]]:
    c = case(step, "B-MATERIALS-shop", "B-MATERIALS", [
        W("10", "Each material has a unit price and a stock count, which starts at 0."), W("10", "Effect: the stock increases by `quantity`."),
        W("10", "Effect: the stock decreases by `quantity`."), W("4.5", "| `listMaterials` → `materials` | `materialId` ascending |")])
    w.people(c)
    w.shop(c)
    c.ok("addMaterial", S, T1, {"materialId": "Resin", "name": "Resin jar", "unitPriceCents": 1500}, {"material": f.material("Resin", "Resin jar", 1500, 0)})
    c.refuse("addMaterial", S, T1, {"materialId": "acrylic", "name": "X", "unitPriceCents": 1}, "already_exists")
    c.ok("listMaterials", "ada", T1, {}, {"materials": [f.material("Resin", "Resin jar", 1500, 0), f.material("acrylic", "Acrylic sheet", 199, 100),
                                                    f.material("plywood", "Plywood panel", 37, 100)]})
    d3 = f.bulk_discount(step, 3)
    c.ok("purchaseMaterial", "ada", T1, {"memberId": "ada", "materialId": "acrylic", "quantity": 3},
         lambda pid: {"purchase": f.purchase(pid, "ada", "acrylic", 3, 199, d3, T1), "material": f.material("acrylic", "Acrylic sheet", 199, 97)})
    c.ok("restockMaterial", S, T1, {"materialId": "plywood", "quantity": 5}, {"material": f.material("plywood", "Plywood panel", 37, 105)})
    c.refuse("purchaseMaterial", S, T1, {"memberId": "ada", "materialId": "steel", "quantity": 1}, "not_found")
    c.refuse("restockMaterial", S, T1, {"materialId": "steel", "quantity": 1}, "not_found")
    c.refuse("purchaseMaterial", S, T1, {"memberId": "bob", "materialId": "acrylic", "quantity": 98}, "insufficient_stock")
    d97 = f.bulk_discount(step, 97)
    c.ok("purchaseMaterial", S, T1, {"memberId": "bob", "materialId": "acrylic", "quantity": 97},
         lambda pid: {"purchase": f.purchase(pid, "bob", "acrylic", 97, 199, d97, T1), "material": f.material("acrylic", "Acrylic sheet", 199, 0)})
    c.refuse("purchaseMaterial", S, T1, {"memberId": "bob", "materialId": "acrylic", "quantity": 1}, "insufficient_stock")
    c.ok("listMaterials", S, T1, {}, {"materials": [f.material("Resin", "Resin jar", 1500, 0), f.material("acrylic", "Acrylic sheet", 199, 0),
                                                 f.material("plywood", "Plywood panel", 37, 105)]})
    c.ok("getAccount", S, T1, {"memberId": "bob"}, {"account": f.account(step, "bob", unbilled=f.line_amount(97, 199, d97))})
    c.audit_log()
    return [c.build()]


def b_billing(step: str) -> list[dict[str, Any]]:
    c = case(step, "B-BILLING-worked-example", "B-BILLING", [
        W("11.1", "Worked example: member `ada` on tier `plus` uses a machine with rate 45 from `2026-03-03T10:00Z` to `10:37`."),
        W("11.1", "`totalCents` 6399, due `2026-04-15T08:00Z`"), W("4.5", "| `listInvoices` → `invoices` | `month` ascending |"),
        W("11.2", "exceeds the invoice's balance becomes credit on the member's account."),
        W("13", "Summarizes billing for one month.")])
    feb = "2026-02-02T09:00Z"
    c.ok("registerUser", "admin", feb, {"userId": S, "name": "Sam", "role": "staff"}, {"user": f.user(S, "Sam", "staff")})
    c.ok("registerMember", S, feb, {"memberId": "ada", "name": "Ada", "tier": "plus"}, {"member": f.member("ada", "Ada", "plus", feb)})
    c.ok("addFamily", "admin", feb, {"familyId": "laser", "name": "Laser cutters"}, {"family": f.family("laser", "Laser cutters")})
    c.ok("addMachine", "admin", feb, {"machineId": "L1", "name": "Laser one", "familyId": "laser", "rateCentsPerMinute": 45},
         {"machine": f.machine("L1", "Laser one", "laser", 45)})
    c.ok("addCourse", S, feb, {"courseId": "laser-101", "familyId": "laser", "title": "Laser basics", "feeCents": 0, "capacity": 5},
         {"course": f.course("laser-101", "laser", "Laser basics", 0, 5, 0)})
    c.ok("enrollInCourse", S, feb, {"courseId": "laser-101", "memberId": "ada"},
         {"enrollment": f.enrollment("laser-101", "ada", "enrolled", feb), "course": f.course("laser-101", "laser", "Laser basics", 0, 5, 1)})
    c.ok("completeCourse", S, feb, {"courseId": "laser-101", "memberId": "ada"},
         {"enrollment": f.enrollment("laser-101", "ada", "completed", feb, feb),
          "certification": f.certification(step, "ada", "laser", "2026-02-02", "2027-02-02", "valid")})
    bob_at = "2026-02-10T09:00Z"
    c.ok("registerMember", S, bob_at, {"memberId": "bob", "name": "Bob", "tier": "basic"}, {"member": f.member("bob", "Bob", "basic", bob_at)})
    g1 = "2026-03-01T08:00Z"
    due1 = "2026-03-15T08:00Z"
    feb_ada = f.invoice("inv-1", "ada", "2026-02", g1, due1, [f.charge("membership", "plus", 1, 4900, 0, g1), f.charge("course", "laser-101", 1, 0, 0, feb)])
    feb_bob = f.invoice("inv-2", "bob", "2026-02", g1, due1, [f.charge("membership", "basic", 1, 2500, 0, g1)])
    c.ok("generateInvoices", S, g1, {"month": "2026-02"}, {"month": "2026-02", "invoices": [feb_ada, feb_bob]})
    rid = w.reserve(c, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), T1)
    paid_ada = f.invoice("inv-1", "ada", "2026-02", g1, due1, feb_ada["lines"], paid=4900)
    c.ok("recordPayment", "ada", T1, {"invoiceId": "inv-1", "amountCents": 4900},
         lambda pid: {"payment": f.payment(pid, "inv-1", "ada", 4900, 4900, T1), "invoice": paid_ada, "creditBalanceCents": 0})
    paid_bob = f.invoice("inv-2", "bob", "2026-02", g1, due1, feb_bob["lines"], paid=2500)
    c.ok("recordPayment", S, T1, {"invoiceId": "inv-2", "amountCents": 3000},
         lambda pid: {"payment": f.payment(pid, "inv-2", "bob", 3000, 2500, T1), "invoice": paid_bob, "creditBalanceCents": 500})
    c.refuse("recordPayment", S, T1, {"invoiceId": "inv-9", "amountCents": 1}, "not_found")
    c.ok("checkIn", S, ts(TUE, "10:00"), {"reservationId": rid},
         {"reservation": f.reservation(rid, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), "checkedIn", ts(TUE, "10:00"), None, 45)})
    out = ts(TUE, "10:37")
    usage = f.charge("usage", rid, 37, 45, 10, out)
    c.ok("checkOut", S, out, {"reservationId": rid},
         {"reservation": f.reservation(rid, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), "completed", ts(TUE, "10:00"), out, 45, 37, 1499),
          "charge": usage})
    c.ok("getAccount", "ada", out, {"memberId": "ada"}, {"account": f.account(step, "ada", unbilled=1499)})
    c.refuse("generateInvoices", S, "2026-03-31T23:59Z", {"month": "2026-03"}, "month_not_ended")
    g2 = "2026-04-01T08:00Z"
    due2 = "2026-04-15T08:00Z"
    mar_ada = f.invoice("inv-3", "ada", "2026-03", g2, due2, [f.charge("membership", "plus", 1, 4900, 0, g2), usage])
    mar_bob = f.invoice("inv-4", "bob", "2026-03", g2, due2, [f.charge("membership", "basic", 1, 2500, 0, g2)], credit=500)
    c.ok("generateInvoices", S, g2, {"month": "2026-03"}, {"month": "2026-03", "invoices": [mar_ada, mar_bob]})
    c.refuse("generateInvoices", S, g2, {"month": "2026-03"}, "already_invoiced")
    c.ok("listInvoices", "ada", g2, {"memberId": "ada"}, {"memberId": "ada", "invoices": [paid_ada, mar_ada]})
    c.ok("getAccount", S, g2, {"memberId": "bob"}, {"account": f.account(step, "bob", open_=2000)})
    c.ok("getAccount", S, g2, {"memberId": "ada"}, {"account": f.account(step, "ada", open_=6399)})
    c.ok("revenueReport", S, g2, {"month": "2026-03"}, {"month": "2026-03", "invoicesGenerated": True, "invoiceCount": 2, "membershipCents": 7400,
                                                        "usageCents": 1499, "materialCents": 0, "courseCents": 0, "invoicedCents": 8899,
                                                        "creditAppliedCents": 500, "outstandingCents": 8399, "paymentsCents": 7900, "creditsIssuedCents": 0})
    c.ok("revenueReport", S, g2, {"month": "2026-02"}, {"month": "2026-02", "invoicesGenerated": True, "invoiceCount": 2, "membershipCents": 7400,
                                                        "usageCents": 0, "materialCents": 0, "courseCents": 0, "invoicedCents": 7400,
                                                        "creditAppliedCents": 0, "outstandingCents": 0, "paymentsCents": 0, "creditsIssuedCents": 0})
    c.ok("issueCredit", "admin", g2, {"memberId": "ada", "amountCents": 250, "reason": "Sorry"}, {"memberId": "ada", "amountCents": 250, "creditBalanceCents": 250})
    over = "2026-04-15T08:00Z"
    c.ok("listInvoices", S, over, {"memberId": "bob"}, {"memberId": "bob", "invoices": [
        paid_bob, f.invoice("inv-4", "bob", "2026-03", g2, due2, mar_bob["lines"], credit=500, overdue=True)]})
    c.ok("getAccount", S, over, {"memberId": "ada"}, {"account": f.account(step, "ada", credit=250, open_=6399, overdue=6399)})
    c.audit_log()
    return [c.build()]


def b_safety(step: str) -> list[dict[str, Any]]:
    c = case(step, "B-SAFETY-incidents", "B-SAFETY", [
        W("12", "Effect: creates an incident with status `open`, `reportedBy` the actor, `reportedAt` the request time, and `resolution` and `resolvedAt` `null`."),
        W("12", "Effect: status `resolved`, `resolution` as supplied, `resolvedAt` the request time."), W("4.5", "| `listIncidents` → `incidents` | creation order |"),
        W("12", "In this delivery an incident is a record only: it does not block reservations or use.") if not since(step, "C2") else
        C(2, "Changes to existing operations and rules", "An incident itself still blocks nothing; only its lockouts do.")])
    w.world(c)
    one = f.incident("inc-1", "L1", "high", "Smoke from the fan", S, T1)
    two = f.incident("inc-2", "L2", "low", "Loose panel", "admin", T1)
    c.ok("reportIncident", S, T1, {"machineId": "L1", "severity": "high", "description": "Smoke from the fan"}, lambda iid: {"incident": one})
    c.ok("reportIncident", "admin", T1, {"machineId": "L2", "severity": "low", "description": "Loose panel"}, lambda iid: {"incident": two})
    t2 = "2026-03-02T10:00Z"
    fixed = f.incident("inc-1", "L1", "high", "Smoke from the fan", S, T1, "Replaced fan", t2)
    c.ok("resolveIncident", S, t2, {"incidentId": "inc-1", "resolution": "Replaced fan"}, {"incident": fixed})
    c.refuse("resolveIncident", S, t2, {"incidentId": "inc-1", "resolution": "Again"}, "invalid_state")
    c.refuse("resolveIncident", S, t2, {"incidentId": "inc-9", "resolution": "R"}, "not_found")
    c.refuse("reportIncident", S, t2, {"machineId": "L9", "severity": "low", "description": "D"}, "not_found")
    c.ok("listIncidents", S, t2, {}, {"incidents": [fixed, two]})
    w.reserve(c, "ada", "L2", ts(TUE, "10:00"), ts(TUE, "10:30"), t2)
    c.audit_log()
    return [c.build()]


# --- Changes ---------------------------------------------------------------------------------

def d_c1(step: str) -> list[dict[str, Any]]:
    c = case(step, "D-C1-refusal-entries", "D-C1", [
        C(1, "What changes", "Example: member `ada` sends `listAuditEntries` at `2026-03-05T10:00Z`."),
        C(1, "What changes", "The audit entry record (section 5) is now: `seq`, `at`, `actor`, `actorRole`, `op`, `target` (`null` in refusal entries), `outcome` (`accepted` or `refused`)."),
        C(1, "What is not recorded", "Invalid requests.")])
    w.people(c)
    at = "2026-03-05T10:00Z"
    c.refuse("listAuditEntries", "ada", at, {}, "forbidden")
    c.refuse("getMember", "ada", at, {"memberId": "bob"}, "forbidden")
    c.refuse("setMachineRate", S, at, {"machineId": "L1", "rateCentsPerMinute": 5}, "forbidden")
    c.refuse("registerUser", "ada", "2026-03-02T07:00Z", {"userId": "u", "name": "U", "role": "staff"}, "time_regression")
    c.refuse("listAuditEntries", "ghost", at, {}, "unknown_actor")
    c.refuse("getMember", "ada", at, {"memberId": "nobody"}, "not_found")
    c.invalid('{"id":"q","op":"listAuditEntries","args":{"actor":"ada","at":"%s","x":1}}' % at, "q")
    c.ok("changeTier", S, "2026-03-02T09:00Z", {"memberId": "bob", "tier": "pro"}, {"member": f.member("bob", "Bob", "pro", T0)})
    c.audit_log()
    return [c.build()]


def d_c2(step: str) -> list[dict[str, Any]]:
    c = case(step, "D-C2-lockout", "D-C2", [
        C(2, "New operation", "Every reservation of that machine whose reported status is `reserved` and whose interval overlaps the lockout becomes `cancelled`."),
        C(2, "New operation", "Reservations that are checked in, completed, missed or already cancelled are not changed."),
        C(2, "New operation", "Result: `lockout` and `cancelledReservations`, the reservations cancelled by this request (after the change), in creation order."),
        C(2, "Lockouts", "A lockout cannot be changed, cancelled or ended early; resolving its incident does not end it."),
        C(2, "What stays as it is", "Scheduling maintenance that overlaps a lockout is allowed, and a lockout never causes `reservation_conflict`."),
        C(2, "Changes to existing operations and rules", "so a minute covered by both maintenance and a lockout appears in both `maintenanceMinutes` and `lockoutMinutes` but is subtracted from `availableMinutes` once."),
        C(2, "Changes to existing operations and rules", "and their metered minutes count in full, also those inside the lockout."),
        C(2, "Lockouts", "A lockout belongs to one incident and locks that incident's machine for an interval `[start, end)`."),
        C(2, "Changes to existing operations and rules", "the minutes of the range that lie inside opening hours and inside at least one lockout of the machine, overlapping lockouts counting once"),
        C(2, "Changes to existing operations and rules", "ordered by `start` ascending, then creation order")])
    w.world(c)
    r = f.reservation
    slots = [("A", "ada", "10:00", "10:30"), ("B", "bob", "09:00", "09:30"), ("D", "bob", "11:00", "11:30"), ("E", "cy", "11:30", "12:00"),
             ("F", "ada", "12:00", "12:30"), ("G", "cy", "10:30", "11:00")]
    ids = {k: w.reserve(c, m, "L1", ts(WED, s), ts(WED, e), T1) for k, m, s, e in slots}
    rec = {k: (m, ts(WED, s), ts(WED, e)) for k, m, s, e in slots}
    other = w.reserve(c, "ada", "L2", ts(WED, "11:00"), ts(WED, "11:30"), T1)
    c.ok("cancelReservation", S, T1, {"reservationId": ids["E"]}, {"reservation": r(ids["E"], "cy", "L1", *rec["E"][1:], "cancelled")})
    c.ok("checkIn", S, ts(WED, "09:00"), {"reservationId": ids["B"]}, {"reservation": r(ids["B"], "bob", "L1", *rec["B"][1:], "checkedIn", ts(WED, "09:00"), None, 45)})
    b_done = r(ids["B"], "bob", "L1", *rec["B"][1:], "completed", ts(WED, "09:00"), ts(WED, "09:20"), 45, 20, 900)
    c.ok("checkOut", S, ts(WED, "09:20"), {"reservationId": ids["B"]}, {"reservation": b_done, "charge": f.charge("usage", ids["B"], 20, 45, 0, ts(WED, "09:20"))})
    c.ok("checkIn", S, ts(WED, "10:00"), {"reservationId": ids["A"]}, {"reservation": r(ids["A"], "ada", "L1", *rec["A"][1:], "checkedIn", ts(WED, "10:00"), None, 45)})
    c.ok("setMemberStanding", S, ts(WED, "10:10"), {"memberId": "cy", "standing": "suspended"}, {"member": f.member("cy", "Cy", "pro", T0, "suspended")})
    inc = f.incident("inc-1", "L1", "high", "Burning smell", S, ts(WED, "10:10"))
    c.ok("reportIncident", S, ts(WED, "10:10"), {"machineId": "L1", "severity": "high", "description": "Burning smell"}, lambda iid: {"incident": inc})
    at = ts(WED, "10:15")
    cancelled = [r(ids["D"], "bob", "L1", *rec["D"][1:], "cancelled"), r(ids["G"], "cy", "L1", *rec["G"][1:], "cancelled")]
    lock1 = f.lockout("lock-1", "inc-1", "L1", at, ts(WED, "12:00"), [ids["D"], ids["G"]])
    c.ok("lockMachine", S, at, {"incidentId": "inc-1", "start": at, "end": ts(WED, "12:00")}, lambda lid: {"lockout": lock1, "cancelledReservations": cancelled})
    c.ok("getAccount", S, at, {"memberId": "bob"}, {"account": f.account(step, "bob", credit=1000, unbilled=3000 + 900)})
    c.ok("getAccount", S, at, {"memberId": "cy"}, {"account": f.account(step, "cy", credit=1000, unbilled=3000)})
    c.ok("getAccount", S, at, {"memberId": "ada"}, {"account": f.account(step, "ada", unbilled=3000)})
    c.ok("getMachineSchedule", S, at, {"machineId": "L2", "date": WED},
         f.schedule(step, "L2", WED, f.opening_hours(step)[2], [r(other, "ada", "L2", ts(WED, "11:00"), ts(WED, "11:30"))], [], []))
    w.reserve(c, "bob", "L2", ts(WED, "10:30"), ts(WED, "11:00"), at)
    c.refuse("createReservation", S, at, {"memberId": "ada", "machineId": "L1", "start": ts(WED, "11:00"), "end": ts(WED, "11:30")}, "machine_locked")
    c.ok("scheduleMaintenance", S, at, {"machineId": "L1", "start": ts(WED, "11:00"), "end": ts(WED, "11:30"), "reason": "Inspect"},
         lambda wid: {"window": f.window(wid, "L1", ts(WED, "11:00"), ts(WED, "11:30"), "Inspect")})
    fixed = f.incident("inc-1", "L1", "high", "Burning smell", S, ts(WED, "10:10"), "Cleaned", ts(WED, "10:20"))
    c.ok("resolveIncident", S, ts(WED, "10:20"), {"incidentId": "inc-1", "resolution": "Cleaned"}, {"incident": fixed})
    c.refuse("createReservation", S, ts(WED, "10:20"), {"memberId": "bob", "machineId": "L1", "start": ts(WED, "11:30"), "end": ts(WED, "12:00")}, "machine_locked")
    c.refuse("lockMachine", S, ts(WED, "10:20"), {"incidentId": "inc-1", "start": ts(THU, "10:00"), "end": ts(THU, "11:00")}, "invalid_state")
    inc2 = f.incident("inc-2", "L1", "medium", "Wobble", S, ts(WED, "10:20"))
    c.ok("reportIncident", S, ts(WED, "10:20"), {"machineId": "L1", "severity": "medium", "description": "Wobble"}, lambda iid: {"incident": inc2})
    lock2 = f.lockout("lock-2", "inc-2", "L1", ts(THU, "20:30"), ts(THU, "22:30"), [])
    lock3 = f.lockout("lock-3", "inc-2", "L1", ts(THU, "10:00"), ts(THU, "11:00"), [])
    lock4 = f.lockout("lock-4", "inc-2", "L1", ts(THU, "10:30"), ts(THU, "12:00"), [])
    for lock in (lock2, lock3, lock4):
        c.ok("lockMachine", S, ts(WED, "10:20"), {"incidentId": "inc-2", "start": lock["start"], "end": lock["end"]},
             lambda lid, lock=lock: {"lockout": lock, "cancelledReservations": []})
    end = ts(WED, "12:40")
    a_done = r(ids["A"], "ada", "L1", *rec["A"][1:], "completed", ts(WED, "10:00"), end, 45, 160, f.line_amount(160, 45, 10))
    c.ok("checkOut", S, end, {"reservationId": ids["A"]}, {"reservation": a_done, "charge": f.charge("usage", ids["A"], 160, 45, 10, end)})
    f_missed = r(ids["F"], "ada", "L1", *rec["F"][1:], "missed")
    c.ok("getMachineSchedule", S, end, {"machineId": "L1", "date": WED}, f.schedule(step, "L1", WED, f.opening_hours(step)[2], [b_done, a_done, f_missed],
         [f.window("mnt-1", "L1", ts(WED, "11:00"), ts(WED, "11:30"), "Inspect")], [lock1]))
    c.ok("getMachineSchedule", S, end, {"machineId": "L1", "date": THU}, f.schedule(step, "L1", THU, f.opening_hours(step)[3], [], [], [lock3, lock4, lock2]))
    open_ = 840 if since(step, "C8") else 720
    locked = 105 + 120 + (90 if since(step, "C8") else 30)  # Wednesday, Thursday 10:00-12:00 once, Thursday 20:30 to closing
    u = f.util_row
    rows = [u(step, "L1", 2 * open_, 30, 180, locked, locked), u(step, "L2", 2 * open_, 0, 0), u(step, "W1", 2 * open_, 0, 0)]
    c.ok("utilizationReport", S, end, {"from": WED, "to": FRI},
         {"from": WED, "to": FRI, "machines": rows, "totals": u(step, None, 6 * open_, 30, 180, locked, locked)})
    c.audit_log()
    return [c.build()]


def d_c3(step: str) -> list[dict[str, Any]]:
    c = case(step, "D-C3-bulk-tiers", "D-C3", [
        C(3, "What changes", "| 9 units at 199 | `(9 × 199 × 100 + 50) div 100` | 1791 |"), C(3, "What changes", "| 10 units at 199 | `(10 × 199 × 95 + 50) div 100` | 1891 |"),
        C(3, "What changes", "| 25 units at 37 | `(25 × 37 × 88 + 50) div 100` | 814 |"),
        C(3, "What must not change", "The tier usage discount never applies to materials, and the bulk discount never applies to usage."),
        C(3, "What must not change", "the revenue report's `materialCents` is still the sum of the material line amounts")])
    w.world(c, certified=("ada",))
    buys = [("ada", "acrylic", 9, 199, 0, 1791), ("ada", "acrylic", 10, 199, 5, 1891), ("bob", "plywood", 25, 37, 12, 814), ("cy", "plywood", 25, 37, 12, 814)]
    stock = {"acrylic": 100, "plywood": 100}
    names = {"acrylic": "Acrylic sheet", "plywood": "Plywood panel"}
    lines: dict[str, list] = {"ada": [], "bob": [], "cy": []}
    for member, mat, qty, price, disc, amount in buys:
        stock[mat] -= qty
        assert f.line_amount(qty, price, disc) == amount
        c.ok("purchaseMaterial", S, T1, {"memberId": member, "materialId": mat, "quantity": qty},
             lambda pid, member=member, mat=mat, qty=qty, price=price, disc=disc: {
                 "purchase": f.purchase(pid, member, mat, qty, price, disc, T1), "material": f.material(mat, names[mat], price, stock[mat])})
        lines[member].append(f.charge("material", c.peek("pur", 0), qty, price, disc, T1))
    rid = w.reserve(c, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), T1)
    c.ok("checkIn", S, ts(TUE, "10:00"), {"reservationId": rid},
         {"reservation": f.reservation(rid, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), "checkedIn", ts(TUE, "10:00"), None, 45)})
    out = ts(TUE, "10:37")
    usage = f.charge("usage", rid, 37, 45, 10, out)
    c.ok("checkOut", S, out, {"reservationId": rid}, {"reservation": f.reservation(rid, "ada", "L1", ts(TUE, "10:00"), ts(TUE, "11:00"), "completed",
                                                                               ts(TUE, "10:00"), out, 45, 37, 1499), "charge": usage})
    g = "2026-04-01T08:00Z"
    due = "2026-04-15T08:00Z"
    m = lambda tier: f.charge("membership", tier, 1, f.FEES[tier], 0, g)
    invoices = [f.invoice("inv-1", "ada", "2026-03", g, due, [m("plus"), f.charge("course", "laser-101", 1, 3000, 0, T0)] + lines["ada"] + [usage]),
                f.invoice("inv-2", "bob", "2026-03", g, due, [m("basic")] + lines["bob"]),
                f.invoice("inv-3", "cy", "2026-03", g, due, [m("pro")] + lines["cy"])]
    c.ok("generateInvoices", S, g, {"month": "2026-03"}, {"month": "2026-03", "invoices": invoices})
    c.ok("revenueReport", S, g, {"month": "2026-03"}, {"month": "2026-03", "invoicesGenerated": True, "invoiceCount": 3, "membershipCents": 17300,
                                                       "usageCents": 1499, "materialCents": 5310, "courseCents": 3000, "invoicedCents": 27109,
                                                       "creditAppliedCents": 0, "outstandingCents": 27109, "paymentsCents": 0, "creditsIssuedCents": 0})
    c.audit_log()
    return [c.build()]


def d_c4(step: str) -> list[dict[str, Any]]:
    c = case(step, "D-C4-steward", "D-C4", [
        C(4, "What changes", "A steward is a user, not a member: a steward's identifier does not name a member, and a steward has no membership, invoices or credit."),
        C(4, "What changes", "when a steward reports an incident, `reportedBy` is the steward's identifier."),
        C(4, "What changes", "a steward may lock a machine with `lockMachine`, and the lockout credits it grants (C2) take effect as for staff.")])
    w.world(c)
    c.ok("registerUser", "admin", T1, {"userId": "stu", "name": "Stu", "role": "steward"}, {"user": f.user("stu", "Stu", "steward")})
    c.refuse("getMember", "admin", T1, {"memberId": "stu"}, "not_found")
    c.refuse("registerMember", S, T1, {"memberId": "stu", "name": "Stu", "tier": "basic"}, "already_exists")
    c.refuse("createReservation", S, T1, {"memberId": "stu", "machineId": "L1", "start": ts(WED, "10:00"), "end": ts(WED, "10:30")}, "not_found")
    rid = w.reserve(c, "ada", "L1", ts(WED, "10:00"), ts(WED, "10:30"), T1)
    inc = f.incident("inc-1", "L1", "high", "Sparks", "stu", T1)
    c.ok("reportIncident", "stu", T1, {"machineId": "L1", "severity": "high", "description": "Sparks"}, lambda iid: {"incident": inc})
    lock = f.lockout("lock-1", "inc-1", "L1", ts(WED, "09:00"), ts(WED, "12:00"), [rid])
    c.ok("lockMachine", "stu", T1, {"incidentId": "inc-1", "start": ts(WED, "09:00"), "end": ts(WED, "12:00")},
         lambda lid: {"lockout": lock, "cancelledReservations": [f.reservation(rid, "ada", "L1", ts(WED, "10:00"), ts(WED, "10:30"), "cancelled")]})
    c.ok("getAccount", "admin", T1, {"memberId": "ada"}, {"account": f.account(step, "ada", credit=1000, unbilled=3000)})
    for op, args in (("getAccount", {"memberId": "ada"}), ("issueCredit", {"memberId": "ada", "amountCents": 5, "reason": "R"}),
                     ("revenueReport", {"month": "2026-03"}), ("listAuditEntries", {}), ("addMachine", {"machineId": "X", "name": "X", "familyId": "laser", "rateCentsPerMinute": 5})):
        c.refuse(op, "stu", T1, args, "forbidden")
    c.ok("resolveIncident", "stu", T1, {"incidentId": "inc-1", "resolution": "Fixed"},
         {"incident": f.incident("inc-1", "L1", "high", "Sparks", "stu", T1, "Fixed", T1)})
    c.ok("listIncidents", "stu", T1, {}, {"incidents": [f.incident("inc-1", "L1", "high", "Sparks", "stu", T1, "Fixed", T1)]})
    c.audit_log()
    return [c.build()]


def d_c5(step: str) -> list[dict[str, Any]]:
    c = case(step, "D-C5-overdue-block", "D-C5", [
        C(5, "What changes", "Example: an invoice issued at `2026-04-01T08:00Z` is due at `2026-04-15T08:00Z`."),
        C(5, "What changes", "the account record gets a new field `blocked`, a boolean that states whether the member is blocked at the request time."),
        C(5, "What stays as it is", "A blocked member's standing stays `good` unless staff suspend the member"),
        C(5, "What stays as it is", "Check-in, check-out, cancellation, course enrollment, payments, credits and invoice generation are not blocked.")])
    w.world(c, certified=("ada",))
    g = "2026-04-01T08:00Z"
    due = "2026-04-15T08:00Z"
    ada_lines = [f.charge("membership", "plus", 1, 4900, 0, g), f.charge("course", "laser-101", 1, 3000, 0, T0)]
    c.ok("generateInvoices", S, g, {"month": "2026-03"}, {"month": "2026-03", "invoices": [
        f.invoice("inv-1", "ada", "2026-03", g, due, ada_lines), f.invoice("inv-2", "bob", "2026-03", g, due, [f.charge("membership", "basic", 1, 2500, 0, g)]),
        f.invoice("inv-3", "cy", "2026-03", g, due, [f.charge("membership", "pro", 1, 9900, 0, g)])]})
    day = "2026-05-18"
    edge, late = "2026-05-15T08:00Z", "2026-05-15T08:01Z"
    r1 = w.reserve(c, "ada", "L1", ts(day, "10:00"), ts(day, "10:30"), edge)
    r2 = w.reserve(c, "ada", "L1", ts(day, "11:00"), ts(day, "11:30"), edge)
    c.refuse("createReservation", "ada", late, {"memberId": "ada", "machineId": "L1", "start": ts(day, "12:00"), "end": ts(day, "12:30")}, "account_overdue")
    c.refuse("purchaseMaterial", S, late, {"memberId": "ada", "materialId": "acrylic", "quantity": 1}, "account_overdue")
    c.ok("getMember", S, late, {"memberId": "ada"}, {"member": f.member("ada", "Ada", "plus", T0)})
    c.ok("getAccount", S, late, {"memberId": "ada"}, {"account": f.account(step, "ada", open_=7900, overdue=7900, blocked=True)})
    c.ok("cancelReservation", "ada", late, {"reservationId": r2}, {"reservation": f.reservation(r2, "ada", "L1", ts(day, "11:00"), ts(day, "11:30"), "cancelled")})
    c.ok("enrollInCourse", S, late, {"courseId": "laser-101", "memberId": "ada"},
         {"enrollment": f.enrollment("laser-101", "ada", "enrolled", late), "course": f.course("laser-101", "laser", "Laser basics", 3000, 10, 1)})
    c.ok("issueCredit", "admin", late, {"memberId": "ada", "amountCents": 500, "reason": "R"}, {"memberId": "ada", "amountCents": 500, "creditBalanceCents": 500})
    c.ok("checkIn", "ada", ts(day, "10:00"), {"reservationId": r1},
         {"reservation": f.reservation(r1, "ada", "L1", ts(day, "10:00"), ts(day, "10:30"), "checkedIn", ts(day, "10:00"), None, 45)})
    out = ts(day, "10:30")
    usage = f.charge("usage", r1, 30, 45, 10, out)
    c.ok("checkOut", "ada", out, {"reservationId": r1},
         {"reservation": f.reservation(r1, "ada", "L1", ts(day, "10:00"), ts(day, "10:30"), "completed", ts(day, "10:00"), out, 45, 30, usage["amountCents"]),
          "charge": usage})
    c.ok("getAccount", S, out, {"memberId": "ada"}, {"account": f.account(step, "ada", credit=500, open_=7900, overdue=7900, unbilled=3000 + usage["amountCents"], blocked=True)})
    paid = f.invoice("inv-1", "ada", "2026-03", g, due, ada_lines, paid=7900)
    c.ok("recordPayment", "ada", out, {"invoiceId": "inv-1", "amountCents": 7900},
         lambda pid: {"payment": f.payment(pid, "inv-1", "ada", 7900, 7900, out), "invoice": paid, "creditBalanceCents": 500})
    w.reserve(c, "ada", "L1", ts(day, "12:00"), ts(day, "12:30"), out)
    c.ok("getAccount", S, out, {"memberId": "ada"}, {"account": f.account(step, "ada", credit=500, unbilled=3000 + usage["amountCents"])})
    c.audit_log()
    return [c.build()]


def d_c6(step: str) -> list[dict[str, Any]]:
    c = case(step, "D-C6-no-expiry", "D-C6", [
        C(6, "What changes", "A certification stays valid from its issue until it is revoked."),
        C(6, "What changes", "the fields are now `memberId`, `familyId`, `issuedOn` and `status` (`valid` or `revoked`)."),
        C(6, "What changes", "still replaces an existing certification for that family, revoked or not, with a new valid one.")])
    w.world(c, certified=("ada",))
    cert = lambda issued, status: {"memberId": "ada", "familyId": "laser", "issuedOn": issued, "status": status}
    c.ok("listCertifications", S, T1, {"memberId": "ada"}, {"memberId": "ada", "certifications": [cert(w.ISSUED, "valid")]})
    w.reserve(c, "ada", "L1", "2028-03-02T10:00Z", "2028-03-02T10:30Z", T1)
    later = "2028-01-03T09:00Z"
    c.ok("listCertifications", S, later, {"memberId": "ada"}, {"memberId": "ada", "certifications": [cert(w.ISSUED, "valid")]})
    c.ok("revokeCertification", S, later, {"memberId": "ada", "familyId": "laser"}, {"certification": cert(w.ISSUED, "revoked")})
    c.refuse("createReservation", S, later, {"memberId": "ada", "machineId": "L1", "start": "2028-03-03T10:00Z", "end": "2028-03-03T10:30Z"}, "not_certified")
    c.ok("enrollInCourse", S, later, {"courseId": "laser-101", "memberId": "ada"},
         {"enrollment": f.enrollment("laser-101", "ada", "enrolled", later), "course": f.course("laser-101", "laser", "Laser basics", 3000, 10, 1)})
    c.ok("completeCourse", S, later, {"courseId": "laser-101", "memberId": "ada"},
         {"enrollment": f.enrollment("laser-101", "ada", "completed", later, later), "certification": cert("2028-01-03", "valid")})
    w.reserve(c, "ada", "L1", "2030-03-01T10:00Z", "2030-03-01T10:30Z", later)
    c.audit_log()
    return [c.build()]


def d_c8(step: str) -> list[dict[str, Any]]:
    c = case(step, "D-C8-hours-and-steps", "D-C8", [
        C(8, "What changes", "Examples: on Sunday `2026-03-08`, `12:00`–`12:15` is accepted and `11:45`–`12:15` is refused with `outside_opening_hours`."),
        C(8, "What changes", "On Friday `2026-03-06`, `19:45`–`20:00` is inside opening hours and `19:45`–`20:15` is not."),
        C(8, "What changes", "On Tuesday `2026-03-03`, `10:15`–`10:30` is accepted and `10:10`–`10:30` is refused with `invalid_slot`."),
        C(8, "What changes", "including dates before the change")])
    w.world(c)
    sun = w.reserve(c, "ada", "L1", ts(SUN, "12:00"), ts(SUN, "12:15"), T1)
    c.refuse("createReservation", S, T1, {"memberId": "ada", "machineId": "L1", "start": ts(SUN, "11:45"), "end": ts(SUN, "12:15")}, "outside_opening_hours")
    w.reserve(c, "ada", "L1", ts(FRI, "19:45"), ts(FRI, "20:00"), T1)
    c.refuse("createReservation", S, T1, {"memberId": "bob", "machineId": "L1", "start": ts(FRI, "19:45"), "end": ts(FRI, "20:15")}, "outside_opening_hours")
    w.reserve(c, "ada", "L1", ts(TUE, "10:15"), ts(TUE, "10:30"), T1)
    c.refuse("createReservation", S, T1, {"memberId": "bob", "machineId": "L1", "start": ts(TUE, "10:10"), "end": ts(TUE, "10:30")}, "invalid_slot")
    c.ok("getMachineSchedule", S, T1, {"machineId": "L1", "date": SUN},
         f.schedule(step, "L1", SUN, ("12:00", "16:00"), [f.reservation(sun, "ada", "L1", ts(SUN, "12:00"), ts(SUN, "12:15"))], []))
    u = f.util_row
    c.ok("utilizationReport", S, "2026-03-03T09:00Z", {"from": MON, "to": TUE},
         {"from": MON, "to": TUE, "machines": [u(step, m, 840, 0, 0) for m in ("L1", "L2", "W1")], "totals": u(step, None, 2520, 0, 0)})
    c.audit_log()
    return [c.build()]


BEHAVIOR = [("W0", b_members), ("W0", b_training), ("W0", b_equipment), ("W0", b_bookings), ("W0", b_materials), ("W0", b_billing),
            ("W0", b_safety), ("C1", d_c1), ("C2", d_c2), ("C3", d_c3), ("C4", d_c4), ("C5", d_c5), ("C6", d_c6), ("C8", d_c8)]


def build_behavior(step: str) -> list[dict[str, Any]]:
    cases: list[dict[str, Any]] = []
    for first, make in BEHAVIOR:
        if since(step, first):
            cases.extend(make(step))
    return cases
