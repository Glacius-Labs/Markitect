"""Change step C1 of the large Makerspace: bookings in 15-minute steps.

The request makes the booking granularity 15 minutes: `start` and `end` fall on :00, :15, :30 or :45, a reservation lasts
15 to 240 minutes, and `createReservation` keeps its refusals and their order. Written from the requirements document after
the change, with every expected value stated by hand.

What this step retires or moves (see RETIRED and transform):
  * P-TIME-GRANULARITY-slots states the 30-minute grid in most of its rows, so it goes.
  * B-bookings-1 and X-floor-1 send quarter-hour slots expecting `invalid_slot`; those rows are moved.
"""
from __future__ import annotations

from typing import Any

from kit import Case, world

W = "2026-03-02T09:00Z"  # Monday, the world's time

RETIRED = {
    "P-TIME-GRANULARITY-slots": "its rows state the 30-minute grid: 10:15-11:00, 10:00-10:45 and 10:00-10:15 are refused there and "
                                "accepted now, and the rows after them book the same machine and member; the new C1-grid, C1-length "
                                "and C1-hours cases hold the grid, the limits and what stays refused",
}


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def book(start, end, mach="L1", mem="ada"):
    return {"memberId": mem, "machineId": mach, "start": start, "end": end}


def transform(case: dict[str, Any]) -> dict[str, Any]:
    """Rows that sent a quarter-hour slot and expected `invalid_slot` now send the nearest slot that is still off the grid,
    or expect the refusal that follows invalid_slot in the order."""
    cid = case["id"]
    if cid not in ("B-bookings-1", "X-floor-1"):
        return case
    for step in case["steps"]:
        request = step.get("request")
        if not request or request["op"] != "createReservation":
            continue
        args = request["args"]
        pair = (args.get("start"), args.get("end"))
        if step["expect"].get("code") != "invalid_slot":
            continue
        if pair == (t(2, "08:15"), t(2, "08:45")):
            # a quarter-hour slot is on the grid now; it starts before the request time, which is the next refusal
            step["expect"]["code"] = "starts_in_past"
        elif cid == "B-bookings-1" and pair == (t(3, "10:15"), t(3, "11:00")):
            args["start"] = t(3, "10:10")
        elif cid == "B-bookings-1" and pair == (t(3, "10:00"), t(3, "10:15")):
            args["end"] = t(3, "10:10")
        elif cid == "B-bookings-1" and pair == (t(3, "10:00"), t(3, "10:45")):
            args["end"] = t(3, "10:40")
    return case


# ---- C1-grid: the new boundaries ---------------------------------------------------------------------------------------

def grid_1() -> Case:
    c = Case("C1-grid-1", "C1-grid", "Create reservation, first example and refusal 2: a slot that starts or ends on :15 or :45 is accepted, "
                                       "as is a 15-minute slot; granularity 15")
    world(c)
    c.ok("createReservation", "ada", W, book(t(3, "10:15"), t(3, "11:00")), {"reservation": res("res-1", "ada", "L1", t(3, "10:15"), t(3, "11:00"))}, new="res-1")
    c.ok("createReservation", "ada", W, book(t(3, "11:00"), t(3, "11:15")), {"reservation": res("res-2", "ada", "L1", t(3, "11:00"), t(3, "11:15"))}, new="res-2")
    c.ok("createReservation", "ada", W, book(t(3, "11:15"), t(3, "11:45")), {"reservation": res("res-3", "ada", "L1", t(3, "11:15"), t(3, "11:45"))}, new="res-3")
    c.ok("createReservation", "ada", W, book(t(3, "12:45"), t(3, "13:00")), {"reservation": res("res-4", "ada", "L1", t(3, "12:45"), t(3, "13:00"))}, new="res-4")
    c.ok("listMemberReservations", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "10:15"), t(3, "11:00")),
        res("res-2", "ada", "L1", t(3, "11:00"), t(3, "11:15")),
        res("res-3", "ada", "L1", t(3, "11:15"), t(3, "11:45")),
        res("res-4", "ada", "L1", t(3, "12:45"), t(3, "13:00"))]})
    c.audit_log(W)
    return c


def grid_2() -> Case:
    c = Case("C1-grid-2", "C1-grid", "Create reservation, refusal 2 (stays): a start or end off the quarter hour is invalid_slot, "
                                       "for 1, 5, 10, 14, 16 and 29 minutes past; the old 30-minute slots stay valid")
    world(c)
    for start, end in (("10:10", "11:00"), ("10:00", "10:50"), ("10:01", "10:30"), ("10:00", "10:29"), ("10:14", "10:30"),
                       ("10:15", "10:31"), ("10:16", "10:45"), ("10:44", "11:15"), ("10:46", "11:15"), ("10:59", "11:30"),
                       ("10:05", "10:30"), ("10:00", "10:10"), ("10:00", "10:05"), ("10:45", "11:01"), ("10:20", "10:50")):
        c.refuse("createReservation", "ada", W, book(t(3, start), t(3, end)), "invalid_slot")
    c.ok("createReservation", "ada", W, book(t(3, "10:30"), t(3, "11:00")), {"reservation": res("res-1", "ada", "L1", t(3, "10:30"), t(3, "11:00"))}, new="res-1")
    c.ok("createReservation", "ada", W, book(t(3, "12:00"), t(3, "16:00")), {"reservation": res("res-2", "ada", "L1", t(3, "12:00"), t(3, "16:00"))}, new="res-2")
    c.audit_log(W)
    return c


def length_1() -> Case:
    c = Case("C1-length-1", "C1-length", "Time, durations: the maximum stays 240, so 10:00-14:15 is invalid_slot; a 15-minute and a 240-minute "
                                         "reservation are accepted")
    world(c)
    c.refuse("createReservation", "ada", W, book(t(5, "10:00"), t(5, "14:15")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(5, "10:15"), t(5, "14:30")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(3, "14:30"), t(3, "18:45")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(5, "10:00"), t(5, "18:00")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(3, "08:15"), t(3, "12:30")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(5, "10:00"), t(5, "10:00")), "invalid_interval")
    c.refuse("createReservation", "ada", W, book(t(5, "10:15"), t(5, "10:15")), "invalid_interval")
    c.refuse("createReservation", "ada", W, book(t(5, "10:15"), t(5, "10:00")), "invalid_interval")
    c.ok("createReservation", "ada", W, book(t(3, "10:00"), t(3, "14:00")), {"reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "14:00"))}, new="res-1")
    c.ok("createReservation", "ada", W, book(t(3, "14:15"), t(3, "14:30")), {"reservation": res("res-2", "ada", "L1", t(3, "14:15"), t(3, "14:30"))}, new="res-2")
    c.ok("createReservation", "ada", W, book(t(4, "10:15"), t(4, "14:15")), {"reservation": res("res-3", "ada", "L1", t(4, "10:15"), t(4, "14:15"))}, new="res-3")
    c.audit_log(W)
    return c


# ---- C1-hours: opening hours meet the new slots ------------------------------------------------------------------------

def hours_1() -> Case:
    c = Case("C1-hours-1", "C1-hours", "Opening hours (unstated): 17:45-18:00 on a Saturday is accepted and 17:45-18:15 is outside_opening_hours; "
                                       "the same at every edge of every day")
    world(c)
    out = "outside_opening_hours"
    c.ok("createReservation", "ada", W, book(t(7, "17:45"), t(7, "18:00")), {"reservation": res("res-1", "ada", "L1", t(7, "17:45"), t(7, "18:00"))}, new="res-1")
    c.refuse("createReservation", "ada", W, book(t(7, "17:45"), t(7, "18:15")), out)
    c.refuse("createReservation", "ada", W, book(t(7, "17:15"), t(7, "18:15")), out)
    c.ok("createReservation", "ada", W, book(t(7, "10:00"), t(7, "10:15")), {"reservation": res("res-2", "ada", "L1", t(7, "10:00"), t(7, "10:15"))}, new="res-2")
    c.refuse("createReservation", "ada", W, book(t(7, "09:45"), t(7, "10:00")), out)
    c.refuse("createReservation", "ada", W, book(t(7, "09:45"), t(7, "10:15")), out)
    c.ok("createReservation", "ada", W, book(t(3, "09:00"), t(3, "09:15")), {"reservation": res("res-3", "ada", "L1", t(3, "09:00"), t(3, "09:15"))}, new="res-3")
    c.refuse("createReservation", "ada", W, book(t(3, "08:45"), t(3, "09:00")), out)
    c.refuse("createReservation", "ada", W, book(t(3, "08:45"), t(3, "09:15")), out)
    c.ok("createReservation", "ada", W, book(t(3, "20:45"), t(3, "21:00")), {"reservation": res("res-4", "ada", "L1", t(3, "20:45"), t(3, "21:00"))}, new="res-4")
    c.refuse("createReservation", "ada", W, book(t(3, "20:45"), t(3, "21:15")), out)
    c.refuse("createReservation", "ada", W, book(t(3, "23:45"), t(4, "00:15")), out)
    c.refuse("createReservation", "ada", W, book(t(8, "10:15"), t(8, "10:30")), out)
    c.refuse("createReservation", "ada", W, book(t(8, "12:00"), t(8, "12:15")), out)
    c.audit_log(W)
    return c


# ---- C1-touch: time, intervals -----------------------------------------------------------------------------------------

def touch_1() -> Case:
    c = Case("C1-touch-1", "C1-touch", "Time, intervals (unstated): 10:00-10:15 and 10:15-10:30 only touch and do not conflict, for the machine "
                                       "and for the member; a 15-minute overlap does conflict")
    world(c)
    c.ok("createReservation", "ada", W, book(t(3, "10:00"), t(3, "10:15")), {"reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "10:15"))}, new="res-1")
    c.ok("createReservation", "ada", W, book(t(3, "10:15"), t(3, "10:30")), {"reservation": res("res-2", "ada", "L1", t(3, "10:15"), t(3, "10:30"))}, new="res-2")
    c.refuse("createReservation", "ada", W, book(t(3, "10:00"), t(3, "10:30")), "machine_booked")
    c.refuse("createReservation", "ada", W, book(t(3, "10:15"), t(3, "10:45")), "machine_booked")
    c.refuse("createReservation", "ada", W, book(t(3, "10:00"), t(3, "10:15"), "L2"), "member_double_booked")
    c.refuse("createReservation", "ada", W, book(t(3, "10:15"), t(3, "10:30"), "L2"), "member_double_booked")
    c.ok("createReservation", "ada", W, book(t(3, "10:30"), t(3, "10:45"), "L2"), {"reservation": res("res-3", "ada", "L2", t(3, "10:30"), t(3, "10:45"))}, new="res-3")
    c.ok("createReservation", "ada", W, book(t(3, "10:45"), t(3, "11:00")), {"reservation": res("res-4", "ada", "L1", t(3, "10:45"), t(3, "11:00"))}, new="res-4")
    c.refuse("createReservation", "ada", W, book(t(3, "10:30"), t(3, "11:00")), "machine_booked")
    c.ok("cancelReservation", "ada", W, {"reservationId": "res-2"}, {"reservation": res("res-2", "ada", "L1", t(3, "10:15"), t(3, "10:30"), "cancelled")})
    c.ok("createReservation", "ada", W, book(t(3, "10:15"), t(3, "10:30")), {"reservation": res("res-5", "ada", "L1", t(3, "10:15"), t(3, "10:30"))}, new="res-5")
    c.ok("listMemberReservations", "ada", W, {"memberId": "ada"}, {"memberId": "ada", "reservations": [
        res("res-1", "ada", "L1", t(3, "10:00"), t(3, "10:15")),
        res("res-2", "ada", "L1", t(3, "10:15"), t(3, "10:30"), "cancelled"),
        res("res-5", "ada", "L1", t(3, "10:15"), t(3, "10:30")),
        res("res-3", "ada", "L2", t(3, "10:30"), t(3, "10:45")),
        res("res-4", "ada", "L1", t(3, "10:45"), t(3, "11:00"))]})
    c.audit_log(W)
    return c


# ---- C1-order: createReservation keeps its refusals and their order ----------------------------------------------------

def order_1() -> Case:
    c = Case("C1-order-1", "C1-order", "Create reservation: refusals keep their order; invalid_slot still follows invalid_interval and precedes "
                                       "starts_in_past, outside_opening_hours, member_suspended and not_certified; a quarter-hour slot passes it")
    world(c)
    c.refuse("createReservation", "ada", W, book(t(3, "10:07"), t(3, "10:07")), "invalid_interval")
    c.refuse("createReservation", "ada", W, book(t(3, "10:07"), t(3, "10:03")), "invalid_interval")
    c.refuse("createReservation", "ada", W, book(t(2, "08:10"), t(2, "08:40")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(2, "08:15"), t(2, "08:45")), "starts_in_past")
    c.refuse("createReservation", "ada", W, book(t(2, "08:45"), t(2, "09:15")), "starts_in_past")
    c.ok("createReservation", "ada", W, book(t(2, "09:00"), t(2, "09:15")), {"reservation": res("res-1", "ada", "L1", t(2, "09:00"), t(2, "09:15"))}, new="res-1")
    c.refuse("createReservation", "ada", W, book(t(3, "08:10"), t(3, "08:40")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(3, "08:15"), t(3, "08:45")), "outside_opening_hours")
    c.refuse("createReservation", "bob", W, book(t(3, "10:10"), t(3, "10:40"), mem="bob"), "invalid_slot")
    c.refuse("createReservation", "bob", W, book(t(3, "10:15"), t(3, "10:30"), mem="bob"), "not_certified")
    c.ok("setMemberStanding", "sam", W, {"memberId": "ada", "standing": "suspended"})
    c.refuse("createReservation", "ada", W, book(t(3, "10:10"), t(3, "10:40")), "invalid_slot")
    c.refuse("createReservation", "ada", W, book(t(3, "08:45"), t(3, "09:15")), "outside_opening_hours")
    c.refuse("createReservation", "ada", W, book(t(3, "10:15"), t(3, "10:30")), "member_suspended")
    c.audit_log(W)
    return c


def past_1() -> Case:
    c = Case("C1-past-1", "C1-order", "Create reservation, refusal 3: a quarter-hour start is in the past when it is before the request time and "
                                      "is accepted at the request time itself; a request time between steps does not matter")
    world(c)
    a, b, d, e = t(3, "10:07"), t(3, "10:30"), t(3, "10:45"), t(3, "10:46")
    c.ok("createReservation", "ada", a, book(t(3, "10:15"), t(3, "10:30")), {"reservation": res("res-1", "ada", "L1", t(3, "10:15"), t(3, "10:30"))}, new="res-1")
    c.refuse("createReservation", "ada", a, book(t(3, "10:00"), t(3, "10:15")), "starts_in_past")
    c.refuse("createReservation", "ada", a, book(t(3, "10:07"), t(3, "10:30"), "L2"), "invalid_slot")
    c.ok("createReservation", "ada", a, book(t(3, "10:30"), t(3, "10:45")), {"reservation": res("res-2", "ada", "L1", t(3, "10:30"), t(3, "10:45"))}, new="res-2")
    c.ok("createReservation", "ada", d, book(t(3, "10:45"), t(3, "11:00")), {"reservation": res("res-3", "ada", "L1", t(3, "10:45"), t(3, "11:00"))}, new="res-3")
    c.refuse("createReservation", "ada", e, book(t(3, "10:45"), t(3, "11:00")), "starts_in_past")
    c.refuse("createReservation", "ada", e, book(t(3, "11:00"), t(3, "11:20")), "invalid_slot")
    c.audit_log(e)
    return c


# ---- C1-life: what a 15-minute reservation goes through ----------------------------------------------------------------

def life_1() -> Case:
    c = Case("C1-life-1", "C1-life", "Reservations, check in, check out: a 15-minute reservation is missed exactly at its end, is checked in and "
                                     "out with 15 metered minutes billed at the captured rate and the tier discount")
    world(c)
    c.ok("createReservation", "ada", W, book(t(3, "10:15"), t(3, "10:30")), {"reservation": res("res-1", "ada", "L1", t(3, "10:15"), t(3, "10:30"))}, new="res-1")
    c.ok("createReservation", "ada", W, book(t(3, "10:30"), t(3, "10:45")), {"reservation": res("res-2", "ada", "L1", t(3, "10:30"), t(3, "10:45"))}, new="res-2")
    c.ok("createReservation", "ada", W, book(t(3, "10:45"), t(3, "11:00")), {"reservation": res("res-3", "ada", "L1", t(3, "10:45"), t(3, "11:00"))}, new="res-3")
    c.ok("checkIn", "ada", t(3, "10:15"), {"reservationId": "res-1"}, {"reservation": res("res-1", "ada", "L1", t(3, "10:15"), t(3, "10:30"), "checkedIn", cin=t(3, "10:15"), rate=45)})
    c.ok("checkOut", "ada", t(3, "10:30"), {"reservationId": "res-1"}, {
        "reservation": res("res-1", "ada", "L1", t(3, "10:15"), t(3, "10:30"), "completed", t(3, "10:15"), t(3, "10:30"), 45, 15, 608),
        "charge": {"kind": "usage", "ref": "res-1", "quantity": 15, "unitPriceCents": 45, "discountPercent": 10, "amountCents": 608, "at": t(3, "10:30")}})
    c.ok("checkIn", "ada", t(3, "10:30"), {"reservationId": "res-2"}, {"reservation": res("res-2", "ada", "L1", t(3, "10:30"), t(3, "10:45"), "checkedIn", cin=t(3, "10:30"), rate=45)})
    soon = [res("res-1", "ada", "L1", t(3, "10:15"), t(3, "10:30"), "completed", t(3, "10:15"), t(3, "10:30"), 45, 15, 608),
            res("res-2", "ada", "L1", t(3, "10:30"), t(3, "10:45"), "checkedIn", cin=t(3, "10:30"), rate=45)]
    c.ok("listMemberReservations", "ada", t(3, "10:59"), {"memberId": "ada"}, {"memberId": "ada", "reservations": soon + [
        res("res-3", "ada", "L1", t(3, "10:45"), t(3, "11:00"))]})
    c.ok("listMemberReservations", "ada", t(3, "11:00"), {"memberId": "ada"}, {"memberId": "ada", "reservations": soon + [
        res("res-3", "ada", "L1", t(3, "10:45"), t(3, "11:00"), "missed")]})
    c.refuse("checkIn", "ada", t(3, "11:00"), {"reservationId": "res-3"}, "invalid_state")
    c.refuse("cancelReservation", "ada", t(3, "11:00"), {"reservationId": "res-3"}, "invalid_state")
    c.audit_log(t(3, "11:00"))
    return c


# ---- C1-guards: rules that work on any minute stay as they are ---------------------------------------------------------

def maintenance_1() -> Case:
    c = Case("C1-guards-1", "C1-guards", "Schedule maintenance (stays): a window on any minute blocks the quarter-hour slots it overlaps and "
                                         "not the ones it only touches")
    world(c)
    c.ok("scheduleMaintenance", "sam", W, {"machineId": "L1", "start": t(3, "10:00"), "end": t(3, "10:15"), "reason": "belt"}, new="mnt-1")
    c.ok("scheduleMaintenance", "sam", W, {"machineId": "L1", "start": t(3, "10:35"), "end": t(3, "10:50"), "reason": "lens"}, new="mnt-2")
    c.refuse("createReservation", "ada", W, book(t(3, "10:00"), t(3, "10:15")), "machine_in_maintenance")
    c.ok("createReservation", "ada", W, book(t(3, "10:15"), t(3, "10:30")), {"reservation": res("res-1", "ada", "L1", t(3, "10:15"), t(3, "10:30"))}, new="res-1")
    c.refuse("createReservation", "ada", W, book(t(3, "10:30"), t(3, "10:45")), "machine_in_maintenance")
    c.refuse("createReservation", "ada", W, book(t(3, "10:45"), t(3, "11:00")), "machine_in_maintenance")
    c.ok("createReservation", "ada", W, book(t(3, "11:00"), t(3, "11:15")), {"reservation": res("res-2", "ada", "L1", t(3, "11:00"), t(3, "11:15"))}, new="res-2")
    c.audit_log(W)
    return c


def cover_1() -> Case:
    c = Case("C1-guards-2", "C1-guards", "Supervision (stays): shifts of 15 minutes piece together to cover a quarter-hour slot, and a slot with an "
                                         "uncovered quarter hour is supervisor_required")
    world(c)
    c.ok("setSupervision", "amy", W, {"familyId": "laser", "required": True})
    c.ok("addShift", "sue", W, {"userId": "sue", "start": t(3, "10:00"), "end": t(3, "10:15")}, new="shf-1")
    c.ok("addShift", "sam", W, {"userId": "sam", "start": t(3, "10:15"), "end": t(3, "10:30")}, new="shf-2")
    c.refuse("createReservation", "ada", W, book(t(3, "09:45"), t(3, "10:15")), "supervisor_required")
    c.refuse("createReservation", "ada", W, book(t(3, "10:15"), t(3, "10:45")), "supervisor_required")
    c.refuse("createReservation", "ada", W, book(t(3, "10:30"), t(3, "10:45")), "supervisor_required")
    c.ok("createReservation", "ada", W, book(t(3, "10:00"), t(3, "10:30")), {"reservation": res("res-1", "ada", "L1", t(3, "10:00"), t(3, "10:30"))}, new="res-1")
    c.ok("addShift", "sue", W, {"userId": "sue", "start": t(3, "10:30"), "end": t(3, "10:45")}, new="shf-3")
    c.ok("createReservation", "ada", W, book(t(3, "10:30"), t(3, "10:45")), {"reservation": res("res-2", "ada", "L1", t(3, "10:30"), t(3, "10:45"))}, new="res-2")
    c.audit_log(W)
    return c


def build_cases() -> list[dict[str, Any]]:
    cases = [grid_1(), grid_2(), length_1(), hours_1(), touch_1(), order_1(), past_1(), life_1(), maintenance_1(), cover_1()]
    return [c.build() for c in cases]
