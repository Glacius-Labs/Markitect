"""Change step C2 of the large Makerspace: a waiver signature lasts a year.

A signature's lapse date is the date 365 days after the date of its `signedAt`; it counts up to the day before and not on
the lapse date. A member has signed a family's waiver when the latest signature (the one recorded last) has the current
version and has not lapsed on the date of the request time; the date of a slot plays no part. Written from the requirements
document after the change, with every expected value stated by hand.

What this step retires or moves: nothing. No earlier case sends a request after a signature's lapse date and expects it to
count. B-compliance-1 reads `complianceSummary` for ada on 2027-03-02, a year after her signature, but a version 2 has been
published by then, so `waiverSigned` is `false` in it either way.

Dates the cases rely on: 2026-03-02 is a Monday, so 2027-03-02 is a Tuesday; 2028 is a leap year, so 365 days is not a
calendar year across 29 February 2028.
"""
from __future__ import annotations

from typing import Any

from kit import Case, world

W = "2026-03-02T09:00Z"  # Monday, the world's time


def t(day: int, hm: str, month: int = 3, year: int = 2026) -> str:
    return f"{year:04d}-{month:02d}-{day:02d}T{hm}Z"


RETIRED: dict[str, str] = {}


def res(rid, mem, mach, start, end, status="reserved", cin=None, cout=None, rate=None, met=None, amt=None):
    return {"reservationId": rid, "memberId": mem, "machineId": mach, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": met, "amountCents": amt}


def book(mem, start, end, mach="L1"):
    return {"memberId": mem, "machineId": mach, "start": start, "end": end}


def waiver(family: str, version: int, text: str) -> dict:
    return {"familyId": family, "version": version, "text": text}


def signature(mid: str, family: str, version: int, at: str) -> dict:
    return {"memberId": mid, "familyId": family, "version": version, "signedAt": at}


def fam(fid, cert, cons, signed, ppe):
    return {"familyId": fid, "certified": cert, "consent": cons, "waiverSigned": signed, "ppeHeld": ppe}


def elig(mem, mach, date, blockers):
    return {"memberId": mem, "machineId": mach, "date": date, "eligible": not blockers, "blockers": blockers}


def publish(c: Case, family: str, version: int, at: str = W) -> None:
    c.ok("publishWaiver", "amy", at, {"familyId": family, "text": f"{family} v{version}"}, {"waiver": waiver(family, version, f"{family} v{version}")})


def sign(c: Case, mem: str, family: str, version: int, at: str) -> None:
    c.ok("signWaiver", mem, at, {"memberId": mem, "familyId": family}, {"signature": signature(mem, family, version, at)})


def member(c: Case, mid: str, name: str, tier: str = "basic") -> None:
    c.ok("registerMember", "sam", W, {"memberId": mid, "name": name, "tier": tier})


def laser_flag(c: Case, mem: str, at: str, laser: bool, lathe: bool = True) -> None:
    """The compliance summary of a member without certification, consent trouble or PPE: only the waiver flags vary."""
    c.ok("complianceSummary", "sam", at, {"memberId": mem}, {"memberId": mem, "standing": "good", "minor": False, "families": [
        fam("laser", False, True, laser, True), fam("lathe", False, True, lathe, True)]})


def lapse_world(c: Case) -> None:
    """The world, a published laser waiver, and eve (pro tier): she signs it on 2026-03-03 at 09:00, so it lapses on 2027-03-03,
    and is certified on 2026-04-01, so her certification is valid through 2027-03-31 and does not get in the way of the gate."""
    world(c)
    publish(c, "laser", 1)
    sign(c, "eve", "laser", 1, t(3, "09:00"))
    c.ok("enrollInCourse", "sam", t(1, "09:00", 4), {"courseId": "laser-101", "memberId": "eve"})
    c.ok("completeCourse", "sam", t(1, "09:00", 4), {"courseId": "laser-101", "memberId": "eve"})


# ---- C2-lapse: the lapse date, as the family compliance row shows it -----------------------------------------------------

def lapse_1() -> Case:
    c = Case("C2-lapse-1", "C2-lapse", "Waiver gate, example; Family compliance, waiverSigned: a signature of 2026-03-02T09:00Z counts at 2027-03-01T23:59Z "
                                       "and not at 2027-03-02T00:00Z; each family on its own; a family without a waiver needs none; a new signature counts again")
    world(c)
    publish(c, "laser", 1)
    sign(c, "bob", "laser", 1, W)
    publish(c, "lathe", 1, t(1, "10:00", 12))
    sign(c, "bob", "lathe", 1, t(1, "10:00", 12))
    laser_flag(c, "bob", "2027-03-01T23:59Z", True)
    laser_flag(c, "bob", "2027-03-02T00:00Z", False)
    laser_flag(c, "bob", "2027-03-02T08:00Z", False)
    sign(c, "bob", "laser", 1, "2027-03-02T10:00Z")
    laser_flag(c, "bob", "2027-03-02T10:00Z", True)
    laser_flag(c, "bob", "2027-11-30T23:59Z", True, True)
    laser_flag(c, "bob", "2027-12-01T00:00Z", True, False)
    c.ok("complianceSummary", "sam", "2027-12-01T00:00Z", {"memberId": "eve"}, {"memberId": "eve", "standing": "good", "minor": False, "families": [
        fam("laser", False, True, False, True), fam("lathe", False, True, False, True)]})
    c.audit_log("2027-12-01T00:00Z")
    return c


def lapse_2() -> Case:
    c = Case("C2-lapse-2", "C2-lapse", "Waiver gate: a signature lapses on the date 365 days after the date of signedAt, whatever the time of day of "
                                       "signing; signed at 00:00, 09:00 and 23:59 on 2026-03-03, all three count at 2027-03-02T23:59Z and none at 2027-03-03T00:00Z")
    world(c)
    member(c, "gus", "Gus Hale")
    publish(c, "laser", 1)
    sign(c, "gus", "laser", 1, t(3, "00:00"))
    sign(c, "bob", "laser", 1, t(3, "09:00"))
    sign(c, "eve", "laser", 1, t(3, "23:59"))
    for who in ("gus", "bob", "eve"):
        laser_flag(c, who, "2027-03-02T23:59Z", True)
    for who in ("gus", "bob", "eve"):
        laser_flag(c, who, "2027-03-03T00:00Z", False)
    c.audit_log("2027-03-03T00:00Z")
    return c


def lapse_3() -> Case:
    c = Case("C2-lapse-3", "C2-lapse", "Waiver gate: 365 days is not a calendar year across 29 February 2028; signed 2027-02-28 lapses 2028-02-28, signed "
                                       "2027-03-01 lapses 2028-02-29, signed 2028-02-29 lapses 2029-02-28")
    world(c)
    member(c, "gus", "Gus Hale")
    publish(c, "laser", 1)
    sign(c, "eve", "laser", 1, "2027-02-28T09:00Z")
    sign(c, "bob", "laser", 1, "2027-03-01T09:00Z")
    laser_flag(c, "eve", "2028-02-27T23:59Z", True)
    laser_flag(c, "bob", "2028-02-27T23:59Z", True)
    laser_flag(c, "eve", "2028-02-28T00:00Z", False)
    laser_flag(c, "bob", "2028-02-28T00:00Z", True)
    laser_flag(c, "bob", "2028-02-28T23:59Z", True)
    laser_flag(c, "bob", "2028-02-29T00:00Z", False)
    sign(c, "gus", "laser", 1, "2028-02-29T09:00Z")
    laser_flag(c, "gus", "2029-02-27T23:59Z", True)
    laser_flag(c, "gus", "2029-02-28T00:00Z", False)
    c.audit_log("2029-02-28T00:00Z")
    return c


def late_1() -> Case:
    c = Case("C2-late-1", "C2-lapse", "Computed times: a signature made in 2099 never lapses at a request time (lapse date after 2099); one made on "
                                      "2098-12-31 lapses on 2099-12-31")
    world(c)
    publish(c, "laser", 1)
    sign(c, "bob", "laser", 1, "2098-12-31T10:00Z")
    sign(c, "eve", "laser", 1, "2099-01-01T00:00Z")
    laser_flag(c, "bob", "2099-12-30T23:59Z", True)
    laser_flag(c, "eve", "2099-12-30T23:59Z", True)
    laser_flag(c, "bob", "2099-12-31T00:00Z", False)
    laser_flag(c, "eve", "2099-12-31T00:00Z", True)
    laser_flag(c, "eve", "2099-12-31T23:59Z", True)
    laser_flag(c, "bob", "2099-12-31T23:59Z", False)
    c.audit_log("2099-12-31T23:59Z")
    return c


# ---- C2-gate: the gate at createReservation and checkIn, on the date of the request -------------------------------------------

def gate_1() -> Case:
    c = Case("C2-gate-1", "C2-gate", "Create reservation, refusal 8: a lapsed signature refuses waiver_missing whatever the slot's date, and one that lapses "
                                     "tomorrow lets through a slot next month; the day before the lapse date it still counts; signing again lets it through")
    lapse_world(c)
    s1, s2, s3 = (t(30, "10:00", 3, 2027), t(30, "10:30", 3, 2027)), (t(3, "10:00", 3, 2027), t(3, "10:30", 3, 2027)), (t(4, "10:00", 3, 2027), t(4, "10:30", 3, 2027))
    c.ok("createReservation", "eve", "2027-03-02T12:00Z", book("eve", *s1), {"reservation": res("res-1", "eve", "L1", *s1)}, new="res-1")
    c.ok("createReservation", "eve", "2027-03-02T12:00Z", book("eve", *s2), {"reservation": res("res-2", "eve", "L1", *s2)}, new="res-2")
    c.ok("createReservation", "eve", "2027-03-02T23:59Z", book("eve", *s3), {"reservation": res("res-3", "eve", "L1", *s3)}, new="res-3")
    late = "2027-03-03T00:00Z"
    c.refuse("createReservation", "eve", late, book("eve", t(5, "10:00", 3, 2027), t(5, "10:30", 3, 2027)), "waiver_missing")
    c.refuse("createReservation", "eve", late, book("eve", t(30, "11:00", 3, 2027), t(30, "11:30", 3, 2027)), "waiver_missing")
    c.refuse("createReservation", "eve", "2027-03-03T12:00Z", book("eve", t(4, "11:00", 3, 2027), t(4, "11:30", 3, 2027)), "waiver_missing")
    sign(c, "eve", "laser", 1, "2027-03-03T12:00Z")
    s4 = (t(5, "10:00", 3, 2027), t(5, "10:30", 3, 2027))
    c.ok("createReservation", "eve", "2027-03-03T12:00Z", book("eve", *s4), {"reservation": res("res-4", "eve", "L1", *s4)}, new="res-4")
    c.ok("listMemberWaivers", "eve", "2027-03-03T12:00Z", {"memberId": "eve"}, {"memberId": "eve", "signatures": [signature("eve", "laser", 1, "2027-03-03T12:00Z")]})
    c.audit_log("2027-03-03T12:00Z")
    return c


def gate_2() -> Case:
    c = Case("C2-gate-2", "C2-gate", "Waiver gate, existing reservations; Check in, refusal 6; Check out: a lapse leaves reservations in place and check-in is "
                                     "judged on the date of the check-in; check-out and cancelling are not stopped; signing again lets check-in through")
    lapse_world(c)
    first, second, third = (t(2, "20:30", 3, 2027), t(2, "21:00", 3, 2027)), (t(3, "10:00", 3, 2027), t(3, "10:30", 3, 2027)), (t(3, "11:00", 3, 2027), t(3, "11:30", 3, 2027))
    c.ok("createReservation", "eve", "2027-03-02T12:00Z", book("eve", *first), {"reservation": res("res-1", "eve", "L1", *first)}, new="res-1")
    c.ok("createReservation", "eve", "2027-03-02T12:00Z", book("eve", *second), {"reservation": res("res-2", "eve", "L1", *second)}, new="res-2")
    c.ok("createReservation", "eve", "2027-03-02T12:00Z", book("eve", *third), {"reservation": res("res-3", "eve", "L1", *third)}, new="res-3")
    c.ok("checkIn", "eve", first[0], {"reservationId": "res-1"}, {"reservation": res("res-1", "eve", "L1", *first, "checkedIn", cin=first[0], rate=45)})
    out = "2027-03-03T00:00Z"
    c.ok("checkOut", "eve", out, {"reservationId": "res-1"}, {
        "reservation": res("res-1", "eve", "L1", *first, "completed", first[0], out, 45, 210, 7088),
        "charge": {"kind": "usage", "ref": "res-1", "quantity": 210, "unitPriceCents": 45, "discountPercent": 25, "amountCents": 7088, "at": out}})
    c.refuse("checkIn", "eve", second[0], {"reservationId": "res-2"}, "waiver_missing")
    c.ok("cancelReservation", "eve", second[0], {"reservationId": "res-3"}, {"reservation": res("res-3", "eve", "L1", *third, "cancelled")})
    sign(c, "eve", "laser", 1, second[0])
    c.ok("checkIn", "eve", second[0], {"reservationId": "res-2"}, {"reservation": res("res-2", "eve", "L1", *second, "checkedIn", cin=second[0], rate=45)})
    c.audit_log(second[0])
    return c


# ---- C2-sign: already_signed and guardian_required at a lapse -------------------------------------------------------------------

def sign_1() -> Case:
    c = Case("C2-sign-1", "C2-sign", "Sign waiver, already_signed: judged as the gate defines having signed; before the lapse date it is still already_signed, "
                                     "from the lapse date a member can sign the same version again, the new signedAt starts a new year; standing is not checked")
    world(c)
    publish(c, "laser", 1)
    sign(c, "ada", "laser", 1, W)
    sign(c, "bob", "laser", 1, W)
    c.ok("setMemberStanding", "sam", W, {"memberId": "bob", "standing": "suspended"})
    c.refuse("signWaiver", "ada", "2027-03-01T23:59Z", {"memberId": "ada", "familyId": "laser"}, "already_signed")
    c.refuse("signWaiver", "bob", "2027-03-01T23:59Z", {"memberId": "bob", "familyId": "laser"}, "already_signed")
    again = "2027-03-02T00:00Z"
    sign(c, "ada", "laser", 1, again)
    sign(c, "bob", "laser", 1, again)
    c.refuse("signWaiver", "ada", again, {"memberId": "ada", "familyId": "laser"}, "already_signed")
    c.refuse("signWaiver", "bob", again, {"memberId": "bob", "familyId": "laser"}, "already_signed")
    c.refuse("signWaiver", "ada", "2028-02-29T23:59Z", {"memberId": "ada", "familyId": "laser"}, "already_signed")
    sign(c, "ada", "laser", 1, "2028-03-01T00:00Z")
    c.ok("listMemberWaivers", "ada", "2028-03-01T00:00Z", {"memberId": "ada"}, {"memberId": "ada", "signatures": [signature("ada", "laser", 1, "2028-03-01T00:00Z")]})
    c.audit_log("2028-03-01T00:00Z")
    return c


def sign_2() -> Case:
    c = Case("C2-sign-2", "C2-sign", "Sign waiver, guardian_required: a minor without consent who signs again after a lapse is refused as on a first signature; "
                                     "before the lapse already_signed still comes first")
    world(c)
    member(c, "mom", "Mom Stone")
    member(c, "kid", "Kid Stone")
    c.ok("declareMinor", "sam", W, {"memberId": "kid", "birthDate": "2012-06-15", "guardianId": "mom"})
    publish(c, "laser", 1)
    c.ok("grantConsent", "mom", W, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"})
    sign(c, "kid", "laser", 1, W)
    c.ok("withdrawConsent", "mom", W, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"})
    c.refuse("signWaiver", "kid", "2027-03-01T23:59Z", {"memberId": "kid", "familyId": "laser"}, "already_signed")
    c.refuse("signWaiver", "kid", "2027-03-02T00:00Z", {"memberId": "kid", "familyId": "laser"}, "guardian_required")
    c.refuse("signWaiver", "sam", "2027-03-02T00:00Z", {"memberId": "kid", "familyId": "laser"}, "guardian_required")
    c.ok("grantConsent", "mom", "2027-03-02T00:00Z", {"guardianId": "mom", "memberId": "kid", "familyId": "laser"})
    sign(c, "kid", "laser", 1, "2027-03-02T00:00Z")
    c.audit_log("2027-03-02T00:00Z")
    return c


# ---- C2-list: the latest signature is the one recorded last ---------------------------------------------------------------------

def list_1() -> Case:
    c = Case("C2-list-1", "C2-list", "List member waivers: the latest signature per family is the one recorded last, so it shows the new signedAt; a lapsed "
                                     "signature is still listed")
    world(c)
    publish(c, "laser", 1)
    publish(c, "lathe", 1)
    sign(c, "ada", "laser", 1, W)
    sign(c, "ada", "lathe", 1, "2026-09-01T10:00Z")
    c.ok("listMemberWaivers", "ada", "2027-03-02T00:00Z", {"memberId": "ada"}, {"memberId": "ada", "signatures": [
        signature("ada", "laser", 1, W), signature("ada", "lathe", 1, "2026-09-01T10:00Z")]})
    sign(c, "ada", "laser", 1, "2027-03-02T10:00Z")
    c.ok("listMemberWaivers", "ada", "2027-03-02T10:00Z", {"memberId": "ada"}, {"memberId": "ada", "signatures": [
        signature("ada", "laser", 1, "2027-03-02T10:00Z"), signature("ada", "lathe", 1, "2026-09-01T10:00Z")]})
    c.ok("listMemberWaivers", "sam", "2027-09-01T10:00Z", {"memberId": "ada"}, {"memberId": "ada", "signatures": [
        signature("ada", "laser", 1, "2027-03-02T10:00Z"), signature("ada", "lathe", 1, "2026-09-01T10:00Z")]})
    sign(c, "ada", "lathe", 1, "2027-09-01T10:00Z")
    c.ok("listMemberWaivers", "amy", "2027-09-01T10:00Z", {"memberId": "ada"}, {"memberId": "ada", "signatures": [
        signature("ada", "laser", 1, "2027-03-02T10:00Z"), signature("ada", "lathe", 1, "2027-09-01T10:00Z")]})
    c.audit_log("2027-09-01T10:00Z")
    return c


# ---- C2-elig: the blocker is judged at the request time ---------------------------------------------------------------------------

def elig_1() -> Case:
    c = Case("C2-elig-1", "C2-elig", "Eligibility, blocker 4: waiver_missing is judged as createReservation does, at the request time, so the date argument plays "
                                     "no part: a far-future date is not blocked while the signature counts, a past date is blocked once it has lapsed")
    world(c)
    publish(c, "laser", 1)
    sign(c, "ada", "laser", 1, W)
    counts = "2027-03-01T23:59Z"
    c.ok("checkEligibility", "ada", counts, {"memberId": "ada", "machineId": "L1", "date": "2027-03-01"}, elig("ada", "L1", "2027-03-01", []))
    c.ok("checkEligibility", "ada", counts, {"memberId": "ada", "machineId": "L1", "date": "2026-03-05"}, elig("ada", "L1", "2026-03-05", []))
    c.ok("checkEligibility", "ada", counts, {"memberId": "ada", "machineId": "L1", "date": "2099-12-31"}, elig("ada", "L1", "2099-12-31", ["not_certified"]))
    lapsed = "2027-03-02T00:00Z"
    c.ok("checkEligibility", "ada", lapsed, {"memberId": "ada", "machineId": "L1", "date": "2027-03-01"}, elig("ada", "L1", "2027-03-01", ["waiver_missing"]))
    c.ok("checkEligibility", "ada", lapsed, {"memberId": "ada", "machineId": "L1", "date": "2026-03-05"}, elig("ada", "L1", "2026-03-05", ["waiver_missing"]))
    c.ok("checkEligibility", "ada", lapsed, {"memberId": "ada", "machineId": "L1", "date": "2099-12-31"},
         elig("ada", "L1", "2099-12-31", ["not_certified", "waiver_missing"]))
    c.ok("checkEligibility", "ada", lapsed, {"memberId": "ada", "machineId": "T1", "date": "2027-03-01"}, elig("ada", "T1", "2027-03-01", ["not_certified"]))
    sign(c, "ada", "laser", 1, lapsed)
    c.ok("checkEligibility", "ada", lapsed, {"memberId": "ada", "machineId": "L1", "date": "2027-03-01"}, elig("ada", "L1", "2027-03-01", []))
    c.audit_log(lapsed)
    return c


# ---- C2-keep: what the year does not touch ------------------------------------------------------------------------------------------

def keep_1() -> Case:
    c = Case("C2-keep-1", "C2-keep", "Publish waiver, Get waiver (unchanged): the year counts from the signature and not from the waiver, so a signature made "
                                     "more than a year after the waiver was published counts; a new version still needs a new signature, however recent the old one is")
    world(c)
    publish(c, "laser", 1)
    sign(c, "bob", "laser", 1, "2027-03-05T10:00Z")
    laser_flag(c, "bob", "2027-03-05T10:00Z", True)
    c.ok("getWaiver", "bob", "2027-03-05T10:00Z", {"familyId": "laser"}, {"waiver": waiver("laser", 1, "laser v1")})
    laser_flag(c, "bob", "2028-03-03T23:59Z", True)
    laser_flag(c, "bob", "2028-03-04T00:00Z", False)
    publish(c, "laser", 2, "2028-03-04T00:05Z")
    sign(c, "bob", "laser", 2, "2028-03-04T00:10Z")
    laser_flag(c, "bob", "2028-03-04T00:10Z", True)
    publish(c, "laser", 3, "2028-03-04T00:15Z")
    laser_flag(c, "bob", "2028-03-04T00:15Z", False)
    c.ok("getWaiver", "bob", "2028-03-04T00:15Z", {"familyId": "laser"}, {"waiver": waiver("laser", 3, "laser v3")})
    c.ok("listMemberWaivers", "bob", "2028-03-04T00:15Z", {"memberId": "bob"}, {"memberId": "bob", "signatures": [signature("bob", "laser", 2, "2028-03-04T00:10Z")]})
    c.audit_log("2028-03-04T00:15Z")
    return c


def build_cases() -> list[dict[str, Any]]:
    cases = [lapse_1(), lapse_2(), lapse_3(), late_1(), gate_1(), gate_2(), sign_1(), sign_2(), list_1(), elig_1(), keep_1()]
    return [c.build() for c in cases]
