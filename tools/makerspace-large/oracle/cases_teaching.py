"""Hidden cases of the teaching cluster: rooms, instructors, classes, assessments, waitlists, payouts, events and what changed
in training. Written from the requirements document alone; every expected value is computed by hand.

  B-<module>-<n>   one module's behavior
  X-teaching-<n>   chains across modules
"""
from __future__ import annotations

from kit import Case, world, has, ref, ANY  # noqa: F401

T0 = "2026-03-02T09:00Z"

ACCOUNTS = ["cash", "creditLiability", "depositLiability", "expenseGoodwill", "expenseMaker", "expensePayout",
            "expenseVolunteer", "expenseWriteOff", "inventory", "payable", "receivable", "revenueCourse", "revenueDeposits",
            "revenueEvents", "revenueGuests", "revenueJobs", "revenueLateFees", "revenueLockers", "revenueMaterial",
            "revenueMembership", "revenuePasses", "revenueRefunds", "revenueUsage"]


# --- expected records -------------------------------------------------------------------------------------------------

def room(i, name, cap):
    return {"roomId": i, "name": name, "capacity": cap}


def hold(i, r, start, end, purpose, status="held"):
    return {"holdId": i, "roomId": r, "start": start, "end": end, "purpose": purpose, "status": status}


def klass(i, course, instructor, r, start, end, status="scheduled"):
    return {"classId": i, "courseId": course, "instructorId": instructor, "roomId": r, "start": start, "end": end, "status": status}


def enr(course, member, at, done=None):
    return {"courseId": course, "memberId": member, "status": "enrolled" if done is None else "completed",
            "enrolledAt": at, "completedAt": done}


def crs(course, family, title, fee, cap, n):
    return {"courseId": course, "familyId": family, "title": title, "feeCents": fee, "capacity": cap, "enrolledCount": n}


def cert(member, family, issued, expires, status="valid"):
    return {"memberId": member, "familyId": family, "issuedOn": issued, "expiresOn": expires, "status": status}


def qual(user, family):
    return {"userId": user, "familyId": family}


def mark(cls, member, present):
    return {"classId": cls, "memberId": member, "present": present}


def attempt(course, member, number, score, passed, at):
    return {"courseId": course, "memberId": member, "number": number, "score": score, "passed": passed, "at": at}


def waiting(course, member, at):
    return {"courseId": course, "memberId": member, "joinedAt": at}


def skipped(member, code):
    return {"memberId": member, "code": code}


def payout(i, user, month, minutes, amount, status="open"):
    return {"payoutId": i, "userId": user, "month": month, "minutes": minutes, "amountCents": amount, "status": status}


def event(i, title, r, start, end, cap, price, sold, status="open"):
    return {"eventId": i, "title": title, "roomId": r, "start": start, "end": end, "capacity": cap, "priceCents": price,
            "status": status, "sold": sold}


def ticket(i, ev, member, status="valid"):
    return {"ticketId": i, "eventId": ev, "memberId": member, "status": status}


def fund(i, budget, spent=0):
    return {"fundId": i, "budgetCents": budget, "spentCents": spent}


def award(i, fund_id, member, percent, status="active"):
    return {"awardId": i, "fundId": fund_id, "memberId": member, "percent": percent, "status": status}


def promo(code, percent, scope, maximum, until, n, ended=False):
    return {"code": code, "percent": percent, "scope": scope, "maxRedemptions": maximum, "validUntil": until,
            "redemptions": n, "ended": ended}


def redemption(code, member, percent, at, used_at=None, used_on=None):
    return {"code": code, "memberId": member, "percent": percent, "redeemedAt": at, "usedAt": used_at, "usedOn": used_on}


def line(kind, ref_, qty, unit, disc, amount, at):
    return {"kind": kind, "ref": ref_, "quantity": qty, "unitPriceCents": unit, "discountPercent": disc, "amountCents": amount, "at": at}


def invoice(i, member, month, issued, due, lines, credit=0, paid=0):
    total = sum(x["amountCents"] for x in lines)
    balance = total - credit - paid
    return {"invoiceId": i, "memberId": member, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": credit, "paidCents": paid, "balanceCents": balance,
            "status": "paid" if balance == 0 else "open", "overdue": False}


def membership(tier, fee, at):
    return line("membership", tier, 1, fee, 0, fee, at)


def notice(i, member, course, at):
    return {"noticeId": i, "memberId": member, "kind": "waitlist", "subject": f"Enrolled from the waitlist of course {course}",
            "ref": course, "at": at, "acknowledgedAt": None}


def account(member, credit, open_, overdue, unbilled):
    return {"memberId": member, "creditBalanceCents": credit, "openBalanceCents": open_, "overdueBalanceCents": overdue,
            "unbilledCents": unbilled}


def jrn(n, at, debit, credit, amount, memo):
    return {"entryId": f"jrn-{n}", "at": at, "debitAccount": debit, "creditAccount": credit, "amountCents": amount, "memo": memo}


def trial(entries):
    debit = {a: 0 for a in ACCOUNTS}
    credit = {a: 0 for a in ACCOUNTS}
    for e in entries:
        debit[e["debitAccount"]] += e["amountCents"]
        credit[e["creditAccount"]] += e["amountCents"]
    rows = []
    for a in ACCOUNTS:
        d, k = debit[a], credit[a]
        rows.append({"account": a, "debitCents": d, "creditCents": k, "balanceCents": abs(d - k),
                     "side": "debit" if d > k else "credit" if k > d else "none"})
    return {"rows": rows, "totalDebitCents": sum(debit.values()), "totalCreditCents": sum(credit.values())}


# --- B-rooms ------------------------------------------------------------------------------------------------------------

def b_rooms_1():
    c = Case("B-rooms-1", "B-rooms", "rooms: add room, hold room checks in their order, release room hold")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 12}, {"room": room("lab", "Lab", 12)})
    c.refuse("addRoom", "admin", T0, {"roomId": "lab", "name": "Again", "capacity": 5}, "already_exists")
    c.refuse("addRoom", "sam", T0, {"roomId": "r9", "name": "Nine", "capacity": 5}, "forbidden")
    c.refuse("addRoom", "admin", T0, {"roomId": "r9", "name": "Nine", "capacity": 0}, "invalid_argument")
    c.refuse("addRoom", "admin", T0, {"roomId": "r9", "name": "Nine", "capacity": 1001}, "invalid_argument")
    c.refuse("addRoom", "admin", T0, {"roomId": "r9", "name": "Nine", "capacity": "5"}, "invalid_argument")
    c.ok("addRoom", "admin", T0, {"roomId": "r2", "name": "Closet", "capacity": 1}, {"room": room("r2", "Closet", 1)})
    c.ok("addRoom", "admin", T0, {"roomId": "r3", "name": "Hall", "capacity": 1000}, {"room": room("r3", "Hall", 1000)})

    s10, e12 = "2026-03-03T10:00Z", "2026-03-03T12:00Z"
    c.ok("holdRoom", "sam", T0, {"roomId": "lab", "start": s10, "end": e12, "purpose": "Demo"},
         {"hold": hold("hld-1", "lab", s10, e12, "Demo")}, new="hld-1")
    c.ok("holdRoom", "sam", T0, {"roomId": "lab", "start": e12, "end": "2026-03-03T13:00Z", "purpose": "After"},
         {"hold": hold("hld-2", "lab", e12, "2026-03-03T13:00Z", "After")}, new="hld-2")
    c.ok("holdRoom", "admin", T0, {"roomId": "lab", "start": "2026-03-03T09:00Z", "end": s10, "purpose": "Before"},
         {"hold": hold("hld-3", "lab", "2026-03-03T09:00Z", s10, "Before")}, new="hld-3")
    # overlaps: into the end, into the start, around, inside, identical
    c.refuse("holdRoom", "sam", T0, {"roomId": "lab", "start": "2026-03-03T11:30Z", "end": "2026-03-03T12:30Z", "purpose": "x"}, "room_taken")
    c.refuse("holdRoom", "sam", T0, {"roomId": "lab", "start": "2026-03-03T08:30Z", "end": "2026-03-03T09:30Z", "purpose": "x"}, "room_taken")
    c.refuse("holdRoom", "sam", T0, {"roomId": "lab", "start": "2026-03-03T08:00Z", "end": "2026-03-03T14:00Z", "purpose": "x"}, "room_taken")
    c.refuse("holdRoom", "sam", T0, {"roomId": "lab", "start": "2026-03-03T10:30Z", "end": "2026-03-03T11:00Z", "purpose": "x"}, "room_taken")
    c.refuse("holdRoom", "sam", T0, {"roomId": "lab", "start": s10, "end": e12, "purpose": "x"}, "room_taken")
    # other room is free for the same time
    c.ok("holdRoom", "sam", T0, {"roomId": "r3", "start": s10, "end": e12, "purpose": "Other room"},
         {"hold": hold("hld-4", "r3", s10, e12, "Other room")}, new="hld-4")
    c.refuse("holdRoom", "bob", T0, {"roomId": "lab", "start": "2026-03-04T10:00Z", "end": "2026-03-04T11:00Z", "purpose": "x"}, "forbidden")
    c.refuse("holdRoom", "ghost", T0, {"roomId": "lab", "start": "2026-03-04T10:00Z", "end": "2026-03-04T11:00Z", "purpose": "x"}, "unknown_actor")
    c.refuse("holdRoom", "sam", T0, {"roomId": "nowhere", "start": "2026-03-04T10:00Z", "end": "2026-03-04T11:00Z", "purpose": "x"}, "not_found")
    c.refuse("holdRoom", "sam", T0, {"roomId": "lab", "start": "2026-03-04T10:00Z", "end": "2026-03-04T11:00Z", "purpose": "   "}, "invalid_argument")

    # now the request time is 11:00 on the day of the first hold
    ta = "2026-03-03T11:00Z"
    c.refuse("holdRoom", "sam", ta, {"roomId": "lab", "start": "2026-03-03T10:30Z", "end": "2026-03-03T11:30Z", "purpose": "x"}, "starts_in_past")
    c.refuse("holdRoom", "sam", ta, {"roomId": "lab", "start": "2026-03-03T10:30Z", "end": "2026-03-03T10:30Z", "purpose": "x"}, "invalid_interval")
    c.refuse("holdRoom", "sam", ta, {"roomId": "lab", "start": "2026-03-03T14:00Z", "end": "2026-03-03T13:00Z", "purpose": "x"}, "invalid_interval")
    # a start equal to the request time is not in the past
    c.ok("holdRoom", "sam", ta, {"roomId": "r2", "start": ta, "end": "2026-03-03T12:00Z", "purpose": "Now"},
         {"hold": hold("hld-5", "r2", ta, "2026-03-03T12:00Z", "Now")}, new="hld-5")
    # release: during the interval; again; by a member; unknown
    c.refuse("releaseRoomHold", "bob", ta, {"holdId": "hld-1"}, "forbidden")
    c.refuse("releaseRoomHold", "sam", ta, {"holdId": "hld-99"}, "not_found")
    c.ok("releaseRoomHold", "sam", ta, {"holdId": "hld-1"}, {"hold": hold("hld-1", "lab", s10, e12, "Demo", "released")})
    c.refuse("releaseRoomHold", "sam", ta, {"holdId": "hld-1"}, "invalid_state")
    # the released hold no longer blocks; hld-2 (12:00) and hld-3 (ended) still stand
    c.ok("holdRoom", "sam", ta, {"roomId": "lab", "start": ta, "end": e12, "purpose": "Reused"},
         {"hold": hold("hld-6", "lab", ta, e12, "Reused")}, new="hld-6")
    c.refuse("holdRoom", "sam", ta, {"roomId": "lab", "start": "2026-03-03T11:59Z", "end": "2026-03-03T12:01Z", "purpose": "x"}, "room_taken")
    c.audit_log(ta)
    return c.build()


def b_rooms_2():
    c = Case("B-rooms-2", "B-rooms", "rooms: get room schedule, day boundaries, order, released holds")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 10}, {"room": room("lab", "Lab", 10)})
    c.ok("addRoom", "admin", T0, {"roomId": "r2", "name": "Closet", "capacity": 2}, {"room": room("r2", "Closet", 2)})
    holds = [
        ("hld-1", "2026-03-03T23:00Z", "2026-03-04T01:00Z", "late"),
        ("hld-2", "2026-03-04T10:00Z", "2026-03-04T11:00Z", "morning"),
        ("hld-3", "2026-03-03T08:00Z", "2026-03-03T09:00Z", "early"),
        ("hld-4", "2026-03-02T22:00Z", "2026-03-03T00:00Z", "ends at midnight"),
        ("hld-5", "2026-03-05T00:00Z", "2026-03-05T01:00Z", "starts at midnight"),
    ]
    for hid, s, e, p in holds:
        c.ok("holdRoom", "sam", T0, {"roomId": "lab", "start": s, "end": e, "purpose": p}, {"hold": hold(hid, "lab", s, e, p)}, new=hid)
    # a hold that crosses midnight collides with one inside it
    c.refuse("holdRoom", "sam", T0, {"roomId": "lab", "start": "2026-03-04T00:00Z", "end": "2026-03-04T00:30Z", "purpose": "x"}, "room_taken")
    c.ok("releaseRoomHold", "sam", T0, {"holdId": "hld-2"}, {"hold": hold("hld-2", "lab", "2026-03-04T10:00Z", "2026-03-04T11:00Z", "morning", "released")})

    def sched(day, shown, room_id="lab", actor="bob"):
        c.ok("getRoomSchedule", actor, T0, {"roomId": room_id, "date": day}, {"roomId": room_id, "date": day, "holds": shown})

    h = {hid: hold(hid, "lab", s, e, p) for hid, s, e, p in holds}
    sched("2026-03-02", [h["hld-4"]])
    sched("2026-03-03", [h["hld-3"], h["hld-1"]])          # start ascending; hld-4 ends at 00:00 and is not shown
    sched("2026-03-04", [h["hld-1"]])                      # hld-2 is released; hld-5 starts at 00:00 of the next day
    sched("2026-03-05", [h["hld-5"]])
    sched("2026-03-06", [])
    sched("2026-03-03", [], room_id="r2", actor="sam")
    c.refuse("getRoomSchedule", "bob", T0, {"roomId": "nowhere", "date": "2026-03-03"}, "not_found")
    c.refuse("getRoomSchedule", "bob", T0, {"roomId": "lab", "date": "2026-02-29"}, "invalid_argument")
    c.refuse("getRoomSchedule", "bob", T0, {"roomId": "lab", "date": "2026-03-03T00:00Z"}, "invalid_argument")
    c.audit_log(T0)
    return c.build()


# --- B-instructors ------------------------------------------------------------------------------------------------------

def b_instructors_1():
    c = Case("B-instructors-1", "B-instructors", "instructors: who may be qualified, order of the refusals, listing, withdrawal")
    world(c)
    c.ok("addFamily", "admin", T0, {"familyId": "router", "name": "Routers"})
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "amy", "familyId": "laser"}, "not_eligible")
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "admin", "familyId": "laser"}, "not_eligible")
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "bob", "familyId": "laser"}, "not_eligible")
    c.refuse("qualifyInstructor", "sam", T0, {"userId": "sam", "familyId": "laser"}, "forbidden")
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "zed", "familyId": "laser"}, "not_found")
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "nope"}, "not_found")
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sue", "familyId": "laser"}, {"qualification": qual("sue", "laser")})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"}, {"qualification": qual("sam", "laser")})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "ada", "familyId": "laser"}, {"qualification": qual("ada", "laser")})
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"}, "already_exists")
    # ada holds a certification for laser only
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "ada", "familyId": "lathe"}, "not_eligible")
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "lathe"}, {"qualification": qual("sam", "lathe")})
    c.ok("listInstructors", "bob", T0, {"familyId": "laser"},
         {"familyId": "laser", "instructors": [qual("ada", "laser"), qual("sam", "laser"), qual("sue", "laser")]})
    c.ok("listInstructors", "sam", T0, {"familyId": "lathe"}, {"familyId": "lathe", "instructors": [qual("sam", "lathe")]})
    c.ok("listInstructors", "amy", T0, {"familyId": "router"}, {"familyId": "router", "instructors": []})
    c.refuse("listInstructors", "bob", T0, {"familyId": "nope"}, "not_found")
    # revoking ada's certification does not end her qualification, but she cannot be qualified again; not_eligible comes first
    c.ok("revokeCertification", "sam", T0, {"memberId": "ada", "familyId": "laser"},
         {"certification": cert("ada", "laser", "2026-03-02", "2027-03-02", "revoked")})
    c.ok("listInstructors", "bob", T0, {"familyId": "laser"},
         {"familyId": "laser", "instructors": [qual("ada", "laser"), qual("sam", "laser"), qual("sue", "laser")]})
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "ada", "familyId": "laser"}, "not_eligible")
    # withdrawal
    c.refuse("withdrawInstructor", "admin", T0, {"userId": "bob", "familyId": "laser"}, "not_instructor")
    c.refuse("withdrawInstructor", "admin", T0, {"userId": "sue", "familyId": "lathe"}, "not_instructor")
    c.refuse("withdrawInstructor", "sam", T0, {"userId": "sue", "familyId": "laser"}, "forbidden")
    c.ok("withdrawInstructor", "admin", T0, {"userId": "ada", "familyId": "laser"}, {"qualification": qual("ada", "laser")})
    c.ok("listInstructors", "bob", T0, {"familyId": "laser"}, {"familyId": "laser", "instructors": [qual("sam", "laser"), qual("sue", "laser")]})
    c.refuse("qualifyInstructor", "admin", T0, {"userId": "ada", "familyId": "laser"}, "not_eligible")
    # a staff user can be qualified again after the withdrawal
    c.ok("withdrawInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"}, {"qualification": qual("sam", "laser")})
    c.refuse("withdrawInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"}, "not_instructor")
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"}, {"qualification": qual("sam", "laser")})
    c.ok("listInstructors", "bob", T0, {"familyId": "laser"}, {"familyId": "laser", "instructors": [qual("sam", "laser"), qual("sue", "laser")]})
    c.audit_log(T0)
    return c.build()


def b_instructors_2():
    c = Case("B-instructors-2", "B-instructors", "instructors: withdrawal and scheduled classes, qualification and certification dates")
    world(c)
    c.ok("enrollInCourse", "sam", T0, {"courseId": "laser-101", "memberId": "eve"})
    c.ok("completeCourse", "sam", T0, {"courseId": "laser-101", "memberId": "eve"})
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 10})
    c.ok("addRoom", "admin", T0, {"roomId": "shop", "name": "Shop", "capacity": 10})
    c.ok("addCourse", "sam", T0, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 3000, "capacity": 4})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "lathe"})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sue", "familyId": "laser"})
    c.ok("scheduleClass", "sam", T0, {"courseId": "laser-101", "instructorId": "sam", "roomId": "lab", "start": "2026-03-10T10:00Z", "end": "2026-03-10T12:00Z"},
         {"class": klass("cls-1", "laser-101", "sam", "lab", "2026-03-10T10:00Z", "2026-03-10T12:00Z")}, new="cls-1")
    c.ok("scheduleClass", "sam", T0, {"courseId": "lathe-101", "instructorId": "sam", "roomId": "shop", "start": "2026-03-20T10:00Z", "end": "2026-03-20T11:00Z"},
         {"class": klass("cls-2", "lathe-101", "sam", "shop", "2026-03-20T10:00Z", "2026-03-20T11:00Z")}, new="cls-2")
    c.ok("scheduleClass", "sam", T0, {"courseId": "laser-101", "instructorId": "sue", "roomId": "lab", "start": "2026-03-12T10:00Z", "end": "2026-03-12T11:00Z"},
         {"class": klass("cls-3", "laser-101", "sue", "lab", "2026-03-12T10:00Z", "2026-03-12T11:00Z")}, new="cls-3")
    c.ok("cancelClass", "sam", T0, {"classId": "cls-3"}, {"class": klass("cls-3", "laser-101", "sue", "lab", "2026-03-12T10:00Z", "2026-03-12T11:00Z", "cancelled")})
    # a cancelled class does not prevent the withdrawal
    t9 = "2026-03-09T09:00Z"
    c.refuse("withdrawInstructor", "admin", t9, {"userId": "sue", "familyId": "lathe"}, "not_instructor")
    c.ok("withdrawInstructor", "admin", t9, {"userId": "sue", "familyId": "laser"}, {"qualification": qual("sue", "laser")})
    # a scheduled class that starts after the request time does, by one minute
    c.refuse("withdrawInstructor", "admin", "2026-03-10T09:59Z", {"userId": "sam", "familyId": "laser"}, "instructor_has_classes")
    # a class that starts at the request time does not; a class of another family does not matter
    t10 = "2026-03-10T10:00Z"
    c.ok("withdrawInstructor", "admin", t10, {"userId": "sam", "familyId": "laser"}, {"qualification": qual("sam", "laser")})
    c.refuse("withdrawInstructor", "admin", t10, {"userId": "sam", "familyId": "lathe"}, "instructor_has_classes")
    c.ok("listInstructors", "bob", t10, {"familyId": "laser"}, {"familyId": "laser", "instructors": []})
    # a qualification is judged on the request date: valid up to the day before expiresOn (2027-03-02)
    c.ok("qualifyInstructor", "admin", "2027-03-01T23:59Z", {"userId": "eve", "familyId": "laser"}, {"qualification": qual("eve", "laser")})
    c.refuse("qualifyInstructor", "admin", "2027-03-02T00:00Z", {"userId": "ada", "familyId": "laser"}, "not_eligible")
    c.ok("listInstructors", "bob", "2027-03-02T00:00Z", {"familyId": "laser"}, {"familyId": "laser", "instructors": [qual("eve", "laser")]})
    c.audit_log("2027-03-02T00:00Z")
    return c.build()


# --- B-classes ----------------------------------------------------------------------------------------------------------

def b_classes_1():
    c = Case("B-classes-1", "B-classes", "classes: schedule class checks in their order, room holds, listing order")
    world(c)
    for rid in ("lab", "shop", "annex"):
        c.ok("addRoom", "admin", T0, {"roomId": rid, "name": rid.title(), "capacity": 10})
    c.ok("addCourse", "sam", T0, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 3000, "capacity": 4})
    for u in ("sam", "sue", "ada"):
        c.ok("qualifyInstructor", "admin", T0, {"userId": u, "familyId": "laser"})
    c.ok("holdRoom", "sam", T0, {"roomId": "lab", "start": "2026-03-10T10:00Z", "end": "2026-03-10T12:00Z", "purpose": "demo"},
         {"hold": hold("hld-1", "lab", "2026-03-10T10:00Z", "2026-03-10T12:00Z", "demo")}, new="hld-1")

    def sc(course, ins, r, s, e, actor="sam"):
        return {"courseId": course, "instructorId": ins, "roomId": r, "start": f"2026-03-10T{s}Z", "end": f"2026-03-10T{e}Z"}

    c.refuse("scheduleClass", "sam", T0, sc("laser-101", "bob", "lab", "11:00", "11:00"), "invalid_interval")
    c.refuse("scheduleClass", "sam", T0, {"courseId": "laser-101", "instructorId": "bob", "roomId": "lab", "start": "2026-03-02T08:59Z", "end": "2026-03-02T10:00Z"}, "starts_in_past")
    c.refuse("scheduleClass", "sam", T0, {"courseId": "laser-101", "instructorId": "bob", "roomId": "lab", "start": "2026-03-02T10:00Z", "end": "2026-03-02T09:00Z"}, "invalid_interval")
    c.refuse("scheduleClass", "sam", T0, sc("laser-101", "bob", "lab", "11:00", "13:00"), "not_instructor")
    c.refuse("scheduleClass", "sam", T0, sc("lathe-101", "sam", "annex", "10:00", "11:00"), "not_instructor")
    c.refuse("scheduleClass", "bob", T0, sc("laser-101", "sam", "annex", "10:00", "11:00"), "forbidden")
    c.refuse("scheduleClass", "sam", T0, sc("laser-101", "zed", "annex", "10:00", "11:00"), "not_found")
    c.refuse("scheduleClass", "sam", T0, sc("laser-101", "sam", "nowhere", "10:00", "11:00"), "not_found")
    c.refuse("scheduleClass", "sam", T0, sc("nothing", "sam", "annex", "10:00", "11:00"), "not_found")
    c.ok("scheduleClass", "sam", T0, sc("laser-101", "sam", "shop", "10:00", "12:00"),
         {"class": klass("cls-1", "laser-101", "sam", "shop", "2026-03-10T10:00Z", "2026-03-10T12:00Z")}, new="cls-1")
    # the room is taken (by the direct hold) and the instructor is busy: room_taken comes first
    c.refuse("scheduleClass", "sam", T0, sc("laser-101", "sam", "lab", "11:30", "13:00"), "room_taken")
    c.refuse("scheduleClass", "sam", T0, sc("laser-101", "sam", "annex", "11:30", "13:00"), "instructor_busy")
    c.refuse("scheduleClass", "sam", T0, sc("laser-101", "sue", "shop", "10:00", "12:00"), "room_taken")
    # touching intervals do not overlap
    c.ok("scheduleClass", "sam", T0, sc("laser-101", "sam", "annex", "12:00", "13:00"),
         {"class": klass("cls-2", "laser-101", "sam", "annex", "2026-03-10T12:00Z", "2026-03-10T13:00Z")}, new="cls-2")
    # classes of different instructors may overlap in time, in different rooms
    c.ok("scheduleClass", "sam", T0, sc("laser-101", "sue", "annex", "10:00", "11:00"),
         {"class": klass("cls-3", "laser-101", "sue", "annex", "2026-03-10T10:00Z", "2026-03-10T11:00Z")}, new="cls-3")
    c.ok("scheduleClass", "sam", T0, sc("laser-101", "ada", "annex", "14:00", "15:00"),
         {"class": klass("cls-4", "laser-101", "ada", "annex", "2026-03-10T14:00Z", "2026-03-10T15:00Z")}, new="cls-4")
    c.ok("scheduleClass", "admin", T0, sc("laser-101", "sue", "shop", "12:00", "13:00"),
         {"class": klass("cls-5", "laser-101", "sue", "shop", "2026-03-10T12:00Z", "2026-03-10T13:00Z")}, new="cls-5")
    # start ascending, then creation order
    c.ok("listCourseClasses", "bob", T0, {"courseId": "laser-101"}, {"courseId": "laser-101", "classes": [
        klass("cls-1", "laser-101", "sam", "shop", "2026-03-10T10:00Z", "2026-03-10T12:00Z"),
        klass("cls-3", "laser-101", "sue", "annex", "2026-03-10T10:00Z", "2026-03-10T11:00Z"),
        klass("cls-2", "laser-101", "sam", "annex", "2026-03-10T12:00Z", "2026-03-10T13:00Z"),
        klass("cls-5", "laser-101", "sue", "shop", "2026-03-10T12:00Z", "2026-03-10T13:00Z"),
        klass("cls-4", "laser-101", "ada", "annex", "2026-03-10T14:00Z", "2026-03-10T15:00Z")]})
    c.ok("listCourseClasses", "sam", T0, {"courseId": "lathe-101"}, {"courseId": "lathe-101", "classes": []})
    c.refuse("listCourseClasses", "sam", T0, {"courseId": "nothing"}, "not_found")
    # the classes hold their rooms with origin class
    c.ok("getRoomSchedule", "bob", T0, {"roomId": "annex", "date": "2026-03-10"}, {"roomId": "annex", "date": "2026-03-10", "holds": [
        hold("hld-4", "annex", "2026-03-10T10:00Z", "2026-03-10T11:00Z", "class cls-3"),
        hold("hld-3", "annex", "2026-03-10T12:00Z", "2026-03-10T13:00Z", "class cls-2"),
        hold("hld-5", "annex", "2026-03-10T14:00Z", "2026-03-10T15:00Z", "class cls-4")]})
    c.audit_log(T0)
    return c.build()


def b_classes_2():
    c = Case("B-classes-2", "B-classes", "classes: cancel class, mark attendance checks in their order, attendance listing")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 10})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "laser-101", "memberId": "bob"})
    for n, day in (("cls-1", "10"), ("cls-2", "11"), ("cls-3", "12")):
        s, e = f"2026-03-{day}T10:00Z", f"2026-03-{day}T11:00Z"
        c.ok("scheduleClass", "sam", T0, {"courseId": "laser-101", "instructorId": "sam", "roomId": "lab", "start": s, "end": e},
             {"class": klass(n, "laser-101", "sam", "lab", s, e)}, new=n)
    t5 = "2026-03-05T09:00Z"
    c.refuse("cancelClass", "bob", t5, {"classId": "cls-2"}, "forbidden")
    c.refuse("cancelClass", "sam", t5, {"classId": "cls-99"}, "not_found")
    c.ok("cancelClass", "sam", t5, {"classId": "cls-2"}, {"class": klass("cls-2", "laser-101", "sam", "lab", "2026-03-11T10:00Z", "2026-03-11T11:00Z", "cancelled")})
    c.refuse("cancelClass", "sam", t5, {"classId": "cls-2"}, "invalid_state")
    # the room is free again for the cancelled class's interval
    c.ok("scheduleClass", "sam", t5, {"courseId": "laser-101", "instructorId": "sam", "roomId": "lab", "start": "2026-03-11T10:00Z", "end": "2026-03-11T11:00Z"},
         {"class": klass("cls-4", "laser-101", "sam", "lab", "2026-03-11T10:00Z", "2026-03-11T11:00Z")}, new="cls-4")
    c.ok("getRoomSchedule", "sam", t5, {"roomId": "lab", "date": "2026-03-11"}, {"roomId": "lab", "date": "2026-03-11", "holds": [
        hold("hld-4", "lab", "2026-03-11T10:00Z", "2026-03-11T11:00Z", "class cls-4")]})
    # one minute before the start: class_not_started comes before not_enrolled
    t_before = "2026-03-10T09:59Z"
    c.refuse("markAttendance", "sam", t_before, {"classId": "cls-1", "memberId": "bob", "present": True}, "class_not_started")
    c.refuse("markAttendance", "sam", t_before, {"classId": "cls-1", "memberId": "eve", "present": True}, "class_not_started")
    # a cancelled class: invalid_state comes first, even before its start
    c.refuse("markAttendance", "sam", t_before, {"classId": "cls-2", "memberId": "eve", "present": True}, "invalid_state")
    c.refuse("markAttendance", "sam", t_before, {"classId": "cls-2", "memberId": "bob", "present": True}, "invalid_state")
    # at the start the request is accepted; eve has no enrollment at all, ada's is completed
    t10 = "2026-03-10T10:00Z"
    c.refuse("markAttendance", "sam", t10, {"classId": "cls-1", "memberId": "eve", "present": True}, "not_enrolled")
    c.refuse("markAttendance", "bob", t10, {"classId": "cls-1", "memberId": "bob", "present": True}, "forbidden")
    c.ok("markAttendance", "sam", t10, {"classId": "cls-1", "memberId": "bob", "present": True}, {"attendance": mark("cls-1", "bob", True)})
    c.ok("markAttendance", "sam", t10, {"classId": "cls-1", "memberId": "ada", "present": True}, {"attendance": mark("cls-1", "ada", True)})
    c.ok("listClassAttendance", "sam", t10, {"classId": "cls-1"}, {"classId": "cls-1", "attendance": [mark("cls-1", "ada", True), mark("cls-1", "bob", True)]})
    # a later mark replaces the earlier one, including present
    c.ok("markAttendance", "sam", t10, {"classId": "cls-1", "memberId": "bob", "present": False}, {"attendance": mark("cls-1", "bob", False)})
    c.ok("listClassAttendance", "admin", t10, {"classId": "cls-1"}, {"classId": "cls-1", "attendance": [mark("cls-1", "ada", True), mark("cls-1", "bob", False)]})
    c.ok("listClassAttendance", "sam", t10, {"classId": "cls-2"}, {"classId": "cls-2", "attendance": []})
    c.refuse("listClassAttendance", "bob", t10, {"classId": "cls-1"}, "forbidden")
    c.refuse("listClassAttendance", "sam", t10, {"classId": "cls-99"}, "not_found")
    # a class that starts at the request time cannot be cancelled; one that starts a minute later can
    c.refuse("cancelClass", "sam", t10, {"classId": "cls-1"}, "invalid_state")
    c.ok("cancelClass", "sam", "2026-03-12T09:59Z", {"classId": "cls-3"}, {"class": klass("cls-3", "laser-101", "sam", "lab", "2026-03-12T10:00Z", "2026-03-12T11:00Z", "cancelled")})
    c.refuse("cancelClass", "sam", "2026-03-12T09:59Z", {"classId": "cls-1"}, "invalid_state")
    c.audit_log("2026-03-12T09:59Z")
    return c.build()


# --- B-assessments ------------------------------------------------------------------------------------------------------

def b_assessments_1():
    c = Case("B-assessments-1", "B-assessments", "assessments: define, record attempt checks in their order, pass mark, attempts, listing")
    world(c)
    c.ok("addCourse", "sam", T0, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 3000, "capacity": 4})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "laser-101", "memberId": "bob"})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "laser-101", "memberId": "eve"})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "lathe-101", "memberId": "bob"})
    c.refuse("recordAttempt", "sam", T0, {"courseId": "lathe-101", "memberId": "bob", "score": 50}, "no_assessment")
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "ada", "score": 50}, "no_assessment")
    c.ok("defineAssessment", "sam", T0, {"courseId": "laser-101", "passMark": 70, "maxAttempts": 2},
         {"assessment": {"courseId": "laser-101", "passMark": 70, "maxAttempts": 2}})
    c.refuse("defineAssessment", "sam", T0, {"courseId": "laser-101", "passMark": 80, "maxAttempts": 3}, "already_exists")
    c.refuse("defineAssessment", "bob", T0, {"courseId": "lathe-101", "passMark": 80, "maxAttempts": 3}, "forbidden")
    c.refuse("defineAssessment", "sam", T0, {"courseId": "nothing", "passMark": 80, "maxAttempts": 3}, "not_found")
    c.refuse("defineAssessment", "sam", T0, {"courseId": "lathe-101", "passMark": 101, "maxAttempts": 3}, "invalid_argument")
    c.refuse("defineAssessment", "sam", T0, {"courseId": "lathe-101", "passMark": -1, "maxAttempts": 3}, "invalid_argument")
    c.refuse("defineAssessment", "sam", T0, {"courseId": "lathe-101", "passMark": 50, "maxAttempts": 0}, "invalid_argument")
    c.refuse("defineAssessment", "sam", T0, {"courseId": "lathe-101", "passMark": 50, "maxAttempts": 11}, "invalid_argument")
    # the assessment applies at once to members who are already enrolled
    c.refuse("completeCourse", "sam", T0, {"courseId": "laser-101", "memberId": "bob"}, "assessment_not_passed")
    # ada's enrollment is completed: not enrolled
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "ada", "score": 50}, "not_enrolled")
    c.refuse("recordAttempt", "bob", T0, {"courseId": "laser-101", "memberId": "bob", "score": 50}, "forbidden")
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "bob", "score": 101}, "invalid_argument")
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "bob", "score": -1}, "invalid_argument")
    # bob: 69 does not pass, 70 does; a third attempt is refused although one passed
    c.ok("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "bob", "score": 69}, {"attempt": attempt("laser-101", "bob", 1, 69, False, T0)})
    c.refuse("completeCourse", "sam", T0, {"courseId": "laser-101", "memberId": "bob"}, "assessment_not_passed")
    c.ok("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "bob", "score": 70}, {"attempt": attempt("laser-101", "bob", 2, 70, True, T0)})
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "bob", "score": 100}, "attempts_exhausted")
    c.ok("completeCourse", "sam", T0, {"courseId": "laser-101", "memberId": "bob"}, {
        "enrollment": enr("laser-101", "bob", T0, T0), "certification": cert("bob", "laser", "2026-03-02", "2027-03-02")})
    # an exhausted member who is no longer enrolled: not_enrolled comes before attempts_exhausted
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "bob", "score": 100}, "not_enrolled")
    # eve: a first attempt that passed still allows a second
    c.ok("recordAttempt", "admin", T0, {"courseId": "laser-101", "memberId": "eve", "score": 100}, {"attempt": attempt("laser-101", "eve", 1, 100, True, T0)})
    c.ok("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "eve", "score": 10}, {"attempt": attempt("laser-101", "eve", 2, 10, False, T0)})
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "eve", "score": 10}, "attempts_exhausted")
    c.ok("completeCourse", "sam", T0, {"courseId": "laser-101", "memberId": "eve"}, {
        "enrollment": enr("laser-101", "eve", T0, T0), "certification": cert("eve", "laser", "2026-03-02", "2027-03-02")})
    # a pass mark of 0 is passed by every score; one attempt allowed
    c.ok("defineAssessment", "sam", T0, {"courseId": "lathe-101", "passMark": 0, "maxAttempts": 1},
         {"assessment": {"courseId": "lathe-101", "passMark": 0, "maxAttempts": 1}})
    c.ok("recordAttempt", "sam", T0, {"courseId": "lathe-101", "memberId": "bob", "score": 0}, {"attempt": attempt("lathe-101", "bob", 1, 0, True, T0)})
    c.refuse("recordAttempt", "sam", T0, {"courseId": "lathe-101", "memberId": "bob", "score": 0}, "attempts_exhausted")
    # listing: in the order recorded; members read their own
    c.ok("listAttempts", "bob", T0, {"courseId": "laser-101", "memberId": "bob"}, {"courseId": "laser-101", "memberId": "bob", "attempts": [
        attempt("laser-101", "bob", 1, 69, False, T0), attempt("laser-101", "bob", 2, 70, True, T0)]})
    c.refuse("listAttempts", "eve", T0, {"courseId": "laser-101", "memberId": "bob"}, "forbidden")
    c.ok("listAttempts", "sam", T0, {"courseId": "laser-101", "memberId": "ada"}, {"courseId": "laser-101", "memberId": "ada", "attempts": []})
    c.audit_log(T0)
    return c.build()


def b_assessments_2():
    c = Case("B-assessments-2", "B-assessments", "assessments: enrollment periods, attempt numbers and the limit restart with each enrollment")
    world(c)
    t1, t2 = "2026-03-03T09:00Z", "2026-03-04T09:00Z"
    c.ok("enrollInCourse", "sam", T0, {"courseId": "laser-101", "memberId": "bob"})
    c.ok("defineAssessment", "sam", T0, {"courseId": "laser-101", "passMark": 80, "maxAttempts": 1})
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "eve", "score": 90}, "not_enrolled")
    c.ok("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "bob", "score": 80}, {"attempt": attempt("laser-101", "bob", 1, 80, True, T0)})
    c.refuse("recordAttempt", "sam", T0, {"courseId": "laser-101", "memberId": "bob", "score": 90}, "attempts_exhausted")
    c.ok("completeCourse", "sam", t1, {"courseId": "laser-101", "memberId": "bob"}, {
        "enrollment": enr("laser-101", "bob", T0, t1), "certification": cert("bob", "laser", "2026-03-03", "2027-03-03")})
    # a new enrollment at the same time as the completion starts period 2: the pass of period 1 counts for nothing
    c.ok("enrollInCourse", "sam", t1, {"courseId": "laser-101", "memberId": "bob"}, {
        "enrollment": enr("laser-101", "bob", t1), "course": crs("laser-101", "laser", "Laser safety", 5000, 2, 1)})
    c.refuse("completeCourse", "sam", t1, {"courseId": "laser-101", "memberId": "bob"}, "assessment_not_passed")
    c.ok("recordAttempt", "sam", t1, {"courseId": "laser-101", "memberId": "bob", "score": 80}, {"attempt": attempt("laser-101", "bob", 1, 80, True, t1)})
    c.refuse("recordAttempt", "sam", t1, {"courseId": "laser-101", "memberId": "bob", "score": 80}, "attempts_exhausted")
    c.ok("completeCourse", "sam", t2, {"courseId": "laser-101", "memberId": "bob"}, {
        "enrollment": enr("laser-101", "bob", t1, t2), "certification": cert("bob", "laser", "2026-03-04", "2027-03-04")})
    # period 3: one failed attempt uses the only attempt
    c.ok("enrollInCourse", "sam", t2, {"courseId": "laser-101", "memberId": "bob"}, {
        "enrollment": enr("laser-101", "bob", t2), "course": crs("laser-101", "laser", "Laser safety", 5000, 2, 1)})
    c.ok("recordAttempt", "sam", t2, {"courseId": "laser-101", "memberId": "bob", "score": 79}, {"attempt": attempt("laser-101", "bob", 1, 79, False, t2)})
    c.refuse("recordAttempt", "sam", t2, {"courseId": "laser-101", "memberId": "bob", "score": 100}, "attempts_exhausted")
    c.refuse("completeCourse", "sam", t2, {"courseId": "laser-101", "memberId": "bob"}, "assessment_not_passed")
    c.ok("listAttempts", "bob", t2, {"courseId": "laser-101", "memberId": "bob"}, {"courseId": "laser-101", "memberId": "bob", "attempts": [
        attempt("laser-101", "bob", 1, 80, True, T0), attempt("laser-101", "bob", 1, 80, True, t1), attempt("laser-101", "bob", 1, 79, False, t2)]})
    c.audit_log(t2)
    return c.build()


# --- B-waitlists --------------------------------------------------------------------------------------------------------

def b_waitlists_1():
    c = Case("B-waitlists-1", "B-waitlists", "waitlists: join checks in their order, leave, order of the list, empty promotions")
    world(c)
    c.ok("registerMember", "sam", T0, {"memberId": "cy", "name": "Cy Dunn", "tier": "basic"})
    c.ok("registerMember", "sam", T0, {"memberId": "dan", "name": "Dan Reed", "tier": "basic"})
    c.ok("addCourse", "sam", T0, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 3000, "capacity": 4})
    # a free place: nothing to wait for
    c.refuse("joinWaitlist", "sam", T0, {"courseId": "laser-101", "memberId": "ada"}, "course_not_full")
    c.ok("enrollInCourse", "sam", T0, {"courseId": "laser-101", "memberId": "bob"})
    c.refuse("joinWaitlist", "sam", T0, {"courseId": "laser-101", "memberId": "bob"}, "already_enrolled")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "bob", "standing": "suspended"})
    c.refuse("joinWaitlist", "sam", T0, {"courseId": "laser-101", "memberId": "bob"}, "member_suspended")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "bob", "standing": "good"})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "laser-101", "memberId": "eve"})
    c.refuse("joinWaitlist", "sam", T0, {"courseId": "nothing", "memberId": "ada"}, "not_found")
    c.refuse("joinWaitlist", "ada", T0, {"courseId": "laser-101", "memberId": "cy"}, "forbidden")
    t1, t2, t3 = "2026-03-03T09:00Z", "2026-03-03T10:00Z", "2026-03-03T11:00Z"
    # a member who completed the course may join; members join for themselves
    c.ok("joinWaitlist", "ada", t1, {"courseId": "laser-101", "memberId": "ada"}, {"entry": waiting("laser-101", "ada", t1)})
    c.refuse("joinWaitlist", "ada", t1, {"courseId": "laser-101", "memberId": "ada"}, "already_waiting")
    c.ok("joinWaitlist", "cy", t2, {"courseId": "laser-101", "memberId": "cy"}, {"entry": waiting("laser-101", "cy", t2)})
    c.ok("joinWaitlist", "sam", t3, {"courseId": "laser-101", "memberId": "dan"}, {"entry": waiting("laser-101", "dan", t3)})
    c.ok("listWaitlist", "sam", t3, {"courseId": "laser-101"}, {"courseId": "laser-101", "entries": [
        waiting("laser-101", "ada", t1), waiting("laser-101", "cy", t2), waiting("laser-101", "dan", t3)]})
    c.refuse("listWaitlist", "ada", t3, {"courseId": "laser-101"}, "forbidden")
    c.ok("listWaitlist", "sam", t3, {"courseId": "lathe-101"}, {"courseId": "lathe-101", "entries": []})
    # leaving: the removed entry is returned; joining again puts the member at the end
    t4, t5 = "2026-03-04T09:00Z", "2026-03-04T10:00Z"
    c.refuse("leaveWaitlist", "sam", t4, {"courseId": "laser-101", "memberId": "bob"}, "not_waiting")
    c.refuse("leaveWaitlist", "dan", t4, {"courseId": "laser-101", "memberId": "ada"}, "forbidden")
    c.ok("leaveWaitlist", "ada", t4, {"courseId": "laser-101", "memberId": "ada"}, {"entry": waiting("laser-101", "ada", t1)})
    c.refuse("leaveWaitlist", "ada", t4, {"courseId": "laser-101", "memberId": "ada"}, "not_waiting")
    c.ok("joinWaitlist", "ada", t5, {"courseId": "laser-101", "memberId": "ada"}, {"entry": waiting("laser-101", "ada", t5)})
    c.ok("listWaitlist", "admin", t5, {"courseId": "laser-101"}, {"courseId": "laser-101", "entries": [
        waiting("laser-101", "cy", t2), waiting("laser-101", "dan", t3), waiting("laser-101", "ada", t5)]})
    # standing is not checked when leaving
    c.ok("setMemberStanding", "sam", t5, {"memberId": "cy", "standing": "suspended"})
    c.ok("leaveWaitlist", "cy", t5, {"courseId": "laser-101", "memberId": "cy"}, {"entry": waiting("laser-101", "cy", t2)})
    # nothing to promote: a full course, and a course with no entries, are not errors
    c.ok("promoteWaitlist", "sam", t5, {"courseId": "laser-101"}, {"courseId": "laser-101", "enrollments": [], "skipped": []})
    c.ok("promoteWaitlist", "sam", t5, {"courseId": "lathe-101"}, {"courseId": "lathe-101", "enrollments": [], "skipped": []})
    c.refuse("promoteWaitlist", "ada", t5, {"courseId": "laser-101"}, "forbidden")
    c.refuse("promoteWaitlist", "sam", t5, {"courseId": "nothing"}, "not_found")
    c.ok("listWaitlist", "sam", t5, {"courseId": "laser-101"}, {"courseId": "laser-101", "entries": [
        waiting("laser-101", "dan", t3), waiting("laser-101", "ada", t5)]})
    c.audit_log(t5)
    return c.build()


def b_waitlists_2():
    c = Case("B-waitlists-2", "B-waitlists", "waitlists: promotion in join order, skipped entries, stop when full, notices")
    world(c)
    c.ok("registerMember", "sam", T0, {"memberId": "ann", "name": "Ann Lowe", "tier": "basic"})
    c.ok("registerMember", "sam", T0, {"memberId": "dan", "name": "Dan Reed", "tier": "basic"})
    c.ok("addCourse", "sam", T0, {"courseId": "w2", "familyId": "laser", "title": "Welding two", "feeCents": 1000, "capacity": 2})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "w2", "memberId": "ada"})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "w2", "memberId": "eve"})
    t1 = "2026-03-03T09:00Z"
    c.ok("joinWaitlist", "ann", t1, {"courseId": "w2", "memberId": "ann"})
    c.ok("joinWaitlist", "bob", "2026-03-03T09:05Z", {"courseId": "w2", "memberId": "bob"})
    c.ok("joinWaitlist", "dan", "2026-03-03T09:10Z", {"courseId": "w2", "memberId": "dan"})
    c.ok("setMemberStanding", "sam", "2026-03-03T09:15Z", {"memberId": "ann", "standing": "suspended"})
    t2 = "2026-03-04T09:00Z"
    c.ok("completeCourse", "sam", t2, {"courseId": "w2", "memberId": "ada"}, {
        "enrollment": enr("w2", "ada", T0, t2), "certification": cert("ada", "laser", "2026-03-04", "2027-03-04")})
    # ann is skipped, bob enrolled, the course is full: dan is not tried and stays
    c.ok("promoteWaitlist", "sam", t2, {"courseId": "w2"}, {"courseId": "w2", "enrollments": [enr("w2", "bob", t2)],
                                                          "skipped": [skipped("ann", "member_suspended")]})
    c.ok("listWaitlist", "sam", t2, {"courseId": "w2"}, {"courseId": "w2", "entries": [
        waiting("w2", "ann", t1), waiting("w2", "dan", "2026-03-03T09:10Z")]})
    c.ok("promoteWaitlist", "sam", t2, {"courseId": "w2"}, {"courseId": "w2", "enrollments": [], "skipped": []})
    c.ok("getAccount", "bob", t2, {"memberId": "bob"}, {"account": account("bob", 0, 0, 0, 1000)})
    c.ok("listNotices", "bob", t2, {"memberId": "bob"}, {"memberId": "bob", "notices": [notice("ntc-1", "bob", "w2", t2)]})
    # two free places; dan enrolls directly and keeps his entry; promotion skips it
    t3 = "2026-03-05T09:00Z"
    c.ok("completeCourse", "sam", t3, {"courseId": "w2", "memberId": "eve"})
    c.ok("completeCourse", "sam", t3, {"courseId": "w2", "memberId": "bob"})
    c.ok("enrollInCourse", "sam", t3, {"courseId": "w2", "memberId": "dan"}, {
        "enrollment": enr("w2", "dan", t3), "course": crs("w2", "laser", "Welding two", 1000, 2, 1)})
    c.ok("promoteWaitlist", "sam", t3, {"courseId": "w2"}, {"courseId": "w2", "enrollments": [], "skipped": [
        skipped("ann", "member_suspended"), skipped("dan", "already_enrolled")]})
    c.ok("listWaitlist", "sam", t3, {"courseId": "w2"}, {"courseId": "w2", "entries": [
        waiting("w2", "ann", t1), waiting("w2", "dan", "2026-03-03T09:10Z")]})
    # ann is reinstated: she is enrolled, the course is full, dan is not tried (and is not in skipped)
    t4 = "2026-03-05T10:00Z"
    c.ok("setMemberStanding", "sam", t4, {"memberId": "ann", "standing": "good"})
    c.ok("promoteWaitlist", "sam", t4, {"courseId": "w2"}, {"courseId": "w2", "enrollments": [enr("w2", "ann", t4)], "skipped": []})
    c.ok("listWaitlist", "sam", t4, {"courseId": "w2"}, {"courseId": "w2", "entries": [waiting("w2", "dan", "2026-03-03T09:10Z")]})
    c.ok("listNotices", "ann", t4, {"memberId": "ann"}, {"memberId": "ann", "notices": [notice("ntc-2", "ann", "w2", t4)]})
    c.ok("listNotices", "dan", t4, {"memberId": "dan"}, {"memberId": "dan", "notices": []})
    c.audit_log(t4)
    return c.build()


# --- B-payouts ----------------------------------------------------------------------------------------------------------

def b_payouts_1():
    c = Case("B-payouts-1", "B-payouts", "payouts: instructor rates, generation, month bounds, once only, cancelled classes, month of the start")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 10})
    for u in ("sam", "sue", "ada"):
        c.ok("qualifyInstructor", "admin", T0, {"userId": u, "familyId": "laser"})
    c.refuse("setInstructorRate", "admin", T0, {"userId": "bob", "centsPerHour": 4500}, "not_instructor")
    c.refuse("setInstructorRate", "admin", T0, {"userId": "amy", "centsPerHour": 4500}, "not_instructor")
    c.refuse("setInstructorRate", "sam", T0, {"userId": "sam", "centsPerHour": 4500}, "forbidden")
    c.refuse("setInstructorRate", "admin", T0, {"userId": "sam", "centsPerHour": 0}, "invalid_argument")
    c.refuse("setInstructorRate", "admin", T0, {"userId": "sam", "centsPerHour": 1000001}, "invalid_argument")
    c.refuse("setInstructorRate", "admin", T0, {"userId": "zed", "centsPerHour": 100}, "not_found")
    c.ok("setInstructorRate", "admin", T0, {"userId": "sam", "centsPerHour": 1000}, {"rate": {"userId": "sam", "centsPerHour": 1000}})
    c.ok("setInstructorRate", "admin", T0, {"userId": "sam", "centsPerHour": 4500}, {"rate": {"userId": "sam", "centsPerHour": 4500}})
    c.ok("setInstructorRate", "admin", T0, {"userId": "sue", "centsPerHour": 6000}, {"rate": {"userId": "sue", "centsPerHour": 6000}})
    # sam: 90 + 45 minutes; a cancelled class of 120 minutes does not count. sue: one class that starts on the last day of March
    # and ends in April. ada has no rate.
    def sched(i, ins, s, e, day=None):
        c.ok("scheduleClass", "sam", T0, {"courseId": "laser-101", "instructorId": ins, "roomId": "lab", "start": s, "end": e},
             {"class": klass(i, "laser-101", ins, "lab", s, e)}, new=i)

    sched("cls-1", "sam", "2026-03-10T10:00Z", "2026-03-10T11:30Z")
    sched("cls-2", "sam", "2026-03-11T10:00Z", "2026-03-11T10:45Z")
    sched("cls-3", "sam", "2026-03-20T10:00Z", "2026-03-20T12:00Z")
    c.ok("cancelClass", "sam", T0, {"classId": "cls-3"})
    sched("cls-4", "sue", "2026-03-31T23:30Z", "2026-04-01T01:00Z")
    sched("cls-5", "ada", "2026-03-15T10:00Z", "2026-03-15T11:00Z")

    c.refuse("generatePayouts", "sam", "2026-03-31T23:59Z", {"month": "2026-03"}, "forbidden")
    c.refuse("generatePayouts", "admin", "2026-03-31T23:59Z", {"month": "2026-03"}, "month_not_ended")
    c.refuse("generatePayouts", "admin", "2026-03-31T23:59Z", {"month": "2026-13"}, "invalid_argument")
    t1 = "2026-04-01T00:00Z"
    p1 = payout("pyt-1", "sam", "2026-03", 135, 10125)
    p2 = payout("pyt-2", "sue", "2026-03", 90, 9000)
    c.ok("generatePayouts", "admin", t1, {"month": "2026-03"}, {"month": "2026-03", "payouts": [p1, p2]})
    c.refuse("generatePayouts", "admin", t1, {"month": "2026-03"}, "already_generated")
    # a later rate does not change a payout that exists
    c.ok("setInstructorRate", "admin", t1, {"userId": "sam", "centsPerHour": 9999}, {"rate": {"userId": "sam", "centsPerHour": 9999}})
    c.ok("listPayouts", "sam", t1, {"userId": "sam"}, {"userId": "sam", "payouts": [p1]})
    c.ok("listPayouts", "admin", t1, {"userId": "sue"}, {"userId": "sue", "payouts": [p2]})
    c.ok("listPayouts", "sam", t1, {"userId": "ada"}, {"userId": "ada", "payouts": []})
    c.refuse("listPayouts", "bob", t1, {"userId": "sam"}, "forbidden")
    c.refuse("listPayouts", "sam", t1, {"userId": "zed"}, "not_found")
    # April: the class that ended in April counts only in March. The month is generated although no payout results.
    c.refuse("generatePayouts", "admin", t1, {"month": "2026-04"}, "month_not_ended")
    t2 = "2026-05-01T00:00Z"
    c.ok("generatePayouts", "admin", t2, {"month": "2026-04"}, {"month": "2026-04", "payouts": []})
    c.refuse("generatePayouts", "admin", t2, {"month": "2026-04"}, "already_generated")
    # settle
    c.refuse("settlePayout", "sam", t2, {"payoutId": "pyt-1"}, "forbidden")
    c.refuse("settlePayout", "admin", t2, {"payoutId": "pyt-9"}, "not_found")
    c.ok("settlePayout", "admin", t2, {"payoutId": "pyt-1"}, {"payout": payout("pyt-1", "sam", "2026-03", 135, 10125, "settled")})
    c.refuse("settlePayout", "admin", t2, {"payoutId": "pyt-1"}, "invalid_state")
    c.ok("listPayouts", "admin", t2, {"userId": "sam"}, {"userId": "sam", "payouts": [payout("pyt-1", "sam", "2026-03", 135, 10125, "settled")]})
    c.audit_log(t2)
    return c.build()


def b_payouts_2():
    c = Case("B-payouts-2", "B-payouts", "payouts: rounding once for the month, zero payouts, ledger postings")
    world(c)
    c.ok("registerUser", "admin", T0, {"userId": "sal", "name": "Sal Staff", "role": "staff"})
    c.ok("registerUser", "admin", T0, {"userId": "tom", "name": "Tom Staff", "role": "staff"})
    c.ok("registerUser", "admin", T0, {"userId": "ula", "name": "Ula Staff", "role": "staff"})
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 10})
    rates = {"sal": 29, "sam": 30, "sue": 89, "tom": 90, "ula": 90}
    for u, r in rates.items():
        c.ok("qualifyInstructor", "admin", T0, {"userId": u, "familyId": "laser"})
        c.ok("setInstructorRate", "admin", T0, {"userId": u, "centsPerHour": r})
    n = 0

    def one_minute(who, hhmm):
        nonlocal n
        n += 1
        cid = f"cls-{n}"
        s = f"2026-03-12T{hhmm}Z"
        h, m = int(hhmm[:2]), int(hhmm[3:])
        e = f"2026-03-12T{h:02d}:{m + 1:02d}Z"
        c.ok("scheduleClass", "sam", T0, {"courseId": "laser-101", "instructorId": who, "roomId": "lab", "start": s, "end": e},
             {"class": klass(cid, "laser-101", who, "lab", s, e)}, new=cid)

    one_minute("sal", "10:00")   # 1 minute at 29: (29 + 30) div 60 = 0
    one_minute("sam", "11:00")   # 1 minute at 30: 60 div 60 = 1
    one_minute("sue", "12:00")   # 1 minute at 89: 119 div 60 = 1
    one_minute("tom", "13:00")   # 1 minute at 90: 120 div 60 = 2
    one_minute("ula", "14:00")   # three minutes at 90 rounded once: (270 + 30) div 60 = 5, not 3 x 2
    one_minute("ula", "15:00")
    one_minute("ula", "16:00")
    t1 = "2026-04-01T00:00Z"
    c.ok("generatePayouts", "admin", t1, {"month": "2026-03"}, {"month": "2026-03", "payouts": [
        payout("pyt-1", "sal", "2026-03", 1, 0), payout("pyt-2", "sam", "2026-03", 1, 1), payout("pyt-3", "sue", "2026-03", 1, 1),
        payout("pyt-4", "tom", "2026-03", 1, 2), payout("pyt-5", "ula", "2026-03", 3, 5)]})
    # a payout of 0 posts nothing and consumes no entry number; the others post in creation order
    entries = [jrn(1, t1, "expensePayout", "payable", 1, "pyt-2"), jrn(2, t1, "expensePayout", "payable", 1, "pyt-3"),
               jrn(3, t1, "expensePayout", "payable", 2, "pyt-4"), jrn(4, t1, "expensePayout", "payable", 5, "pyt-5")]
    c.ok("listJournalEntries", "admin", t1, {"month": "2026-04"}, {"month": "2026-04", "entries": entries})
    c.ok("settlePayout", "admin", t1, {"payoutId": "pyt-1"}, {"payout": payout("pyt-1", "sal", "2026-03", 1, 0, "settled")})
    c.ok("listJournalEntries", "admin", t1, {"month": "2026-04"}, {"month": "2026-04", "entries": entries})
    c.ok("settlePayout", "admin", t1, {"payoutId": "pyt-5"}, {"payout": payout("pyt-5", "ula", "2026-03", 3, 5, "settled")})
    entries.append(jrn(5, t1, "payable", "cash", 5, "pyt-5"))
    c.ok("listJournalEntries", "admin", t1, {"month": "2026-04"}, {"month": "2026-04", "entries": entries})
    c.ok("listJournalEntries", "admin", t1, {"month": "2026-03"}, {"month": "2026-03", "entries": []})
    c.audit_log(t1)
    return c.build()


# --- B-events -----------------------------------------------------------------------------------------------------------

def b_events_1():
    c = Case("B-events-1", "B-events", "events: create event checks in their order, sell ticket checks in their order")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 10})
    c.ok("addRoom", "admin", T0, {"roomId": "annex", "name": "Annex", "capacity": 3})
    sd, ed = "2026-03-10T10:00Z", "2026-03-10T12:00Z"

    def mk(i, r, s, e, cap, price, title="Event"):
        return {"eventId": i, "title": title, "roomId": r, "start": s, "end": e, "capacity": cap, "priceCents": price}

    # a capacity equal to the room's is accepted
    c.ok("createEvent", "sam", T0, mk("big", "lab", "2026-03-20T10:00Z", "2026-03-20T12:00Z", 10, 500, "Big"),
         {"event": event("big", "Big", "lab", "2026-03-20T10:00Z", "2026-03-20T12:00Z", 10, 500, 0)})
    c.refuse("createEvent", "sam", T0, mk("tiny", "annex", sd, ed, 4, 500), "capacity_too_large")
    c.ok("createEvent", "admin", T0, mk("open-day", "annex", sd, ed, 2, 1500, "Open day"),
         {"event": event("open-day", "Open day", "annex", sd, ed, 2, 1500, 0)})
    c.refuse("createEvent", "sam", T0, mk("open-day", "annex", ed, sd, 2, 1500), "already_exists")
    c.refuse("createEvent", "sam", T0, mk("e2", "annex", "2026-03-02T08:00Z", "2026-03-02T08:00Z", 4, 1), "invalid_interval")
    c.refuse("createEvent", "sam", T0, mk("e2", "annex", "2026-03-02T08:00Z", "2026-03-02T10:00Z", 4, 1), "starts_in_past")
    c.refuse("createEvent", "sam", T0, mk("e2", "annex", "2026-03-10T11:00Z", "2026-03-10T13:00Z", 4, 1), "capacity_too_large")
    c.refuse("createEvent", "sam", T0, mk("e2", "annex", "2026-03-10T11:00Z", "2026-03-10T13:00Z", 2, 1), "room_taken")
    c.refuse("createEvent", "bob", T0, mk("e2", "annex", "2026-03-11T11:00Z", "2026-03-11T13:00Z", 2, 1), "forbidden")
    c.refuse("createEvent", "sam", T0, mk("e2", "nowhere", "2026-03-11T11:00Z", "2026-03-11T13:00Z", 2, 1), "not_found")
    c.refuse("createEvent", "sam", T0, mk("e2", "annex", "2026-03-11T11:00Z", "2026-03-11T13:00Z", 0, 1), "invalid_argument")
    c.refuse("createEvent", "sam", T0, mk("e2", "annex", "2026-03-11T11:00Z", "2026-03-11T13:00Z", 2, 1000001), "invalid_argument")
    c.ok("createEvent", "sam", T0, mk("after", "annex", ed, "2026-03-10T13:00Z", 1, 0, "After"),
         {"event": event("after", "After", "annex", ed, "2026-03-10T13:00Z", 1, 0, 0)})
    c.ok("getRoomSchedule", "bob", T0, {"roomId": "annex", "date": "2026-03-10"}, {"roomId": "annex", "date": "2026-03-10", "holds": [
        hold("hld-2", "annex", sd, ed, "event open-day"), hold("hld-3", "annex", ed, "2026-03-10T13:00Z", "event after")]})
    # tickets: member_suspended comes before event_closed
    c.ok("setMemberStanding", "sam", T0, {"memberId": "eve", "standing": "suspended"})
    c.ok("cancelEvent", "sam", T0, {"eventId": "after"}, {"event": event("after", "After", "annex", ed, "2026-03-10T13:00Z", 1, 0, 0, "cancelled"), "tickets": []})
    c.refuse("sellTicket", "sam", T0, {"eventId": "after", "memberId": "eve"}, "member_suspended")
    c.refuse("sellTicket", "sam", T0, {"eventId": "open-day", "memberId": "eve"}, "member_suspended")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "eve", "standing": "good"})
    c.refuse("sellTicket", "sam", T0, {"eventId": "after", "memberId": "ada"}, "event_closed")
    c.refuse("sellTicket", "sam", T0, {"eventId": "nothing", "memberId": "ada"}, "not_found")
    c.refuse("sellTicket", "ada", T0, {"eventId": "open-day", "memberId": "bob"}, "forbidden")
    c.ok("sellTicket", "ada", T0, {"eventId": "open-day", "memberId": "ada"}, {
        "ticket": ticket("tkt-1", "open-day", "ada"), "event": event("open-day", "Open day", "annex", sd, ed, 2, 1500, 1)}, new="tkt-1")
    c.refuse("sellTicket", "sam", T0, {"eventId": "open-day", "memberId": "ada"}, "already_ticketed")
    c.ok("sellTicket", "sam", T0, {"eventId": "open-day", "memberId": "bob"}, {
        "ticket": ticket("tkt-2", "open-day", "bob"), "event": event("open-day", "Open day", "annex", sd, ed, 2, 1500, 2)}, new="tkt-2")
    c.refuse("sellTicket", "sam", T0, {"eventId": "open-day", "memberId": "eve"}, "event_sold_out")
    c.refuse("sellTicket", "sam", T0, {"eventId": "open-day", "memberId": "ada"}, "already_ticketed")
    # the start decides: one minute before it the event is sold out, at it the event is closed, before already_ticketed
    c.refuse("sellTicket", "sam", "2026-03-10T09:59Z", {"eventId": "open-day", "memberId": "eve"}, "event_sold_out")
    c.refuse("sellTicket", "sam", "2026-03-10T10:00Z", {"eventId": "open-day", "memberId": "eve"}, "event_closed")
    c.refuse("sellTicket", "sam", "2026-03-10T10:00Z", {"eventId": "open-day", "memberId": "ada"}, "event_closed")
    c.ok("listEventTickets", "sam", "2026-03-10T10:00Z", {"eventId": "open-day"}, {"eventId": "open-day", "tickets": [
        ticket("tkt-1", "open-day", "ada"), ticket("tkt-2", "open-day", "bob")]})
    c.ok("listEventTickets", "sam", "2026-03-10T10:00Z", {"eventId": "big"}, {"eventId": "big", "tickets": []})
    c.refuse("listEventTickets", "ada", "2026-03-10T10:00Z", {"eventId": "open-day"}, "forbidden")
    c.audit_log("2026-03-10T10:00Z")
    return c.build()


def b_events_2():
    c = Case("B-events-2", "B-events", "events: cancel ticket and cancel event, the 1440-minute limit, credit, price 0, journal")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "hall", "name": "Hall", "capacity": 20})
    fs, fe = "2026-03-10T18:00Z", "2026-03-10T20:00Z"
    ps, pe = "2026-03-12T10:00Z", "2026-03-12T11:00Z"
    qs, qe = "2026-03-15T10:00Z", "2026-03-15T11:00Z"
    c.ok("createEvent", "sam", T0, {"eventId": "fair", "title": "Fair", "roomId": "hall", "start": fs, "end": fe, "capacity": 5, "priceCents": 1200})
    c.ok("createEvent", "sam", T0, {"eventId": "free", "title": "Free", "roomId": "hall", "start": ps, "end": pe, "capacity": 5, "priceCents": 0})
    c.ok("createEvent", "sam", T0, {"eventId": "quiet", "title": "Quiet", "roomId": "hall", "start": qs, "end": qe, "capacity": 5, "priceCents": 800})
    t1 = "2026-03-03T10:00Z"
    c.ok("sellTicket", "sam", t1, {"eventId": "fair", "memberId": "ada"}, new="tkt-1")
    c.ok("sellTicket", "sam", t1, {"eventId": "fair", "memberId": "bob"}, new="tkt-2")
    c.ok("sellTicket", "sam", t1, {"eventId": "free", "memberId": "eve"}, new="tkt-3")
    c.ok("sellTicket", "sam", t1, {"eventId": "free", "memberId": "ada"}, new="tkt-4")
    c.refuse("cancelTicket", "bob", t1, {"ticketId": "tkt-1"}, "forbidden")
    c.refuse("cancelTicket", "bob", t1, {"ticketId": "tkt-99"}, "not_found")
    # exactly 1440 minutes before the start: accepted
    t2 = "2026-03-09T18:00Z"
    c.ok("cancelTicket", "bob", t2, {"ticketId": "tkt-2"}, {"ticket": ticket("tkt-2", "fair", "bob", "cancelled"),
                                                            "event": event("fair", "Fair", "hall", fs, fe, 5, 1200, 1)})
    # one minute later: too late; an already cancelled ticket is invalid_state before that
    t3 = "2026-03-09T18:01Z"
    c.refuse("cancelTicket", "ada", t3, {"ticketId": "tkt-1"}, "too_late")
    c.refuse("cancelTicket", "bob", t3, {"ticketId": "tkt-2"}, "invalid_state")
    # credit comes from the price; the charge stays unbilled and unchanged
    c.ok("getAccount", "bob", t3, {"memberId": "bob"}, {"account": account("bob", 1200, 0, 0, 1200)})
    c.ok("getAccount", "ada", t3, {"memberId": "ada"}, {"account": account("ada", 0, 0, 0, 6200)})
    # the whole event: the 1440-minute limit does not apply; only valid tickets are cancelled
    t4 = "2026-03-10T17:59Z"
    c.ok("cancelEvent", "sam", t4, {"eventId": "fair"}, {"event": event("fair", "Fair", "hall", fs, fe, 5, 1200, 0, "cancelled"),
                                                         "tickets": [ticket("tkt-1", "fair", "ada", "cancelled")]})
    c.refuse("cancelEvent", "sam", t4, {"eventId": "fair"}, "invalid_state")
    c.refuse("cancelEvent", "ada", t4, {"eventId": "quiet"}, "forbidden")
    c.ok("getAccount", "ada", t4, {"memberId": "ada"}, {"account": account("ada", 1200, 0, 0, 6200)})
    c.ok("listJournalEntries", "admin", t4, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, t2, "revenueRefunds", "creditLiability", 1200, "tkt-2"), jrn(2, t4, "revenueRefunds", "creditLiability", 1200, "tkt-1")]})
    # price 0: cancelling gives no credit and no posting
    t5 = "2026-03-11T10:00Z"
    c.ok("cancelTicket", "eve", t5, {"ticketId": "tkt-3"}, {"ticket": ticket("tkt-3", "free", "eve", "cancelled"),
                                                            "event": event("free", "Free", "hall", ps, pe, 5, 0, 1)})
    c.ok("getAccount", "eve", t5, {"memberId": "eve"}, {"account": account("eve", 0, 0, 0, 0)})
    c.refuse("cancelTicket", "ada", "2026-03-11T10:01Z", {"ticketId": "tkt-4"}, "too_late")
    c.ok("listJournalEntries", "admin", "2026-03-11T10:01Z", {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, t2, "revenueRefunds", "creditLiability", 1200, "tkt-2"), jrn(2, t4, "revenueRefunds", "creditLiability", 1200, "tkt-1")]})
    # an event that has started cannot be cancelled and its tickets are too late
    t6 = "2026-03-12T10:00Z"
    c.refuse("cancelEvent", "sam", t6, {"eventId": "free"}, "invalid_state")
    c.refuse("cancelTicket", "ada", t6, {"ticketId": "tkt-4"}, "too_late")
    c.ok("cancelEvent", "admin", t6, {"eventId": "quiet"}, {"event": event("quiet", "Quiet", "hall", qs, qe, 5, 800, 0, "cancelled"), "tickets": []})
    c.ok("listEventTickets", "sam", t6, {"eventId": "free"}, {"eventId": "free", "tickets": [
        ticket("tkt-3", "free", "eve", "cancelled"), ticket("tkt-4", "free", "ada")]})
    c.ok("listEventTickets", "sam", t6, {"eventId": "fair"}, {"eventId": "fair", "tickets": [
        ticket("tkt-1", "fair", "ada", "cancelled"), ticket("tkt-2", "fair", "bob", "cancelled")]})
    c.audit_log(t6)
    return c.build()


# --- B-training ---------------------------------------------------------------------------------------------------------

def b_training_1():
    c = Case("B-training-1", "B-training", "training: enroll in course and complete course, order of refusals, places, re-enrollment")
    world(c)
    c.ok("addCourse", "sam", T0, {"courseId": "solo", "familyId": "laser", "title": "Solo", "feeCents": 0, "capacity": 1},
         {"course": crs("solo", "laser", "Solo", 0, 1, 0)})
    c.refuse("addCourse", "sam", T0, {"courseId": "solo", "familyId": "laser", "title": "Again", "feeCents": 0, "capacity": 1}, "already_exists")
    c.refuse("addCourse", "sam", T0, {"courseId": "x", "familyId": "laser", "title": "X", "feeCents": 1000001, "capacity": 1}, "invalid_argument")
    c.refuse("addCourse", "sam", T0, {"courseId": "x", "familyId": "laser", "title": "X", "feeCents": 0, "capacity": 0}, "invalid_argument")
    c.refuse("addCourse", "sam", T0, {"courseId": "x", "familyId": "nope", "title": "X", "feeCents": 0, "capacity": 1}, "not_found")
    c.refuse("addCourse", "bob", T0, {"courseId": "x", "familyId": "laser", "title": "X", "feeCents": 0, "capacity": 1}, "forbidden")
    # suspended and full: member_suspended first
    c.ok("setMemberStanding", "sam", T0, {"memberId": "eve", "standing": "suspended"})
    c.ok("enrollInCourse", "bob", T0, {"courseId": "solo", "memberId": "bob"}, {
        "enrollment": enr("solo", "bob", T0), "course": crs("solo", "laser", "Solo", 0, 1, 1)})
    c.refuse("enrollInCourse", "eve", T0, {"courseId": "solo", "memberId": "eve"}, "member_suspended")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "eve", "standing": "good"})
    c.refuse("enrollInCourse", "eve", T0, {"courseId": "solo", "memberId": "eve"}, "course_full")
    # already enrolled comes before full; suspended before already enrolled
    c.refuse("enrollInCourse", "bob", T0, {"courseId": "solo", "memberId": "bob"}, "already_enrolled")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "bob", "standing": "suspended"})
    c.refuse("enrollInCourse", "bob", T0, {"courseId": "solo", "memberId": "bob"}, "member_suspended")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "bob", "standing": "good"})
    c.refuse("enrollInCourse", "eve", T0, {"courseId": "solo", "memberId": "bob"}, "forbidden")
    c.refuse("enrollInCourse", "eve", T0, {"courseId": "nothing", "memberId": "eve"}, "not_found")
    c.refuse("enrollInCourse", "sam", T0, {"courseId": "solo", "memberId": "sam"}, "not_found")
    # completion
    c.refuse("completeCourse", "sam", T0, {"courseId": "solo", "memberId": "eve"}, "not_enrolled")
    c.refuse("completeCourse", "bob", T0, {"courseId": "solo", "memberId": "bob"}, "forbidden")
    c.ok("completeCourse", "sam", T0, {"courseId": "solo", "memberId": "bob"}, {
        "enrollment": enr("solo", "bob", T0, T0), "certification": cert("bob", "laser", "2026-03-02", "2027-03-02")})
    c.refuse("completeCourse", "sam", T0, {"courseId": "solo", "memberId": "bob"}, "not_enrolled")
    # a place became free; the completed member is full-refused until it is taken, then re-enrolls after another completion
    c.ok("enrollInCourse", "eve", T0, {"courseId": "solo", "memberId": "eve"}, {
        "enrollment": enr("solo", "eve", T0), "course": crs("solo", "laser", "Solo", 0, 1, 1)})
    c.refuse("enrollInCourse", "bob", T0, {"courseId": "solo", "memberId": "bob"}, "course_full")
    t1 = "2026-03-10T09:00Z"
    c.ok("completeCourse", "sam", t1, {"courseId": "solo", "memberId": "eve"}, {
        "enrollment": enr("solo", "eve", T0, t1), "certification": cert("eve", "laser", "2026-03-10", "2027-03-10")})
    # the record is reset: enrolledAt is the new time, completedAt null again
    c.ok("enrollInCourse", "bob", t1, {"courseId": "solo", "memberId": "bob"}, {
        "enrollment": enr("solo", "bob", t1), "course": crs("solo", "laser", "Solo", 0, 1, 1)})
    # standing is not checked when completing; the new certification replaces the old one
    c.ok("setMemberStanding", "sam", t1, {"memberId": "bob", "standing": "suspended"})
    t2 = "2026-03-11T09:00Z"
    c.ok("completeCourse", "sam", t2, {"courseId": "solo", "memberId": "bob"}, {
        "enrollment": enr("solo", "bob", t1, t2), "certification": cert("bob", "laser", "2026-03-11", "2027-03-11")})
    c.ok("listCertifications", "bob", t2, {"memberId": "bob"}, {"memberId": "bob", "certifications": [cert("bob", "laser", "2026-03-11", "2027-03-11")]})
    c.audit_log(t2)
    return c.build()


def b_training_2():
    c = Case("B-training-2", "B-training", "training: certification dates, status at the request time, revoke, replacement, listing order")
    world(c)
    c.ok("registerMember", "sam", T0, {"memberId": "cy", "name": "Cy Dunn", "tier": "basic"})
    c.ok("addCourse", "sam", T0, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 3000, "capacity": 4})
    # eve gets lathe first, then laser; the list is by familyId
    c.ok("enrollInCourse", "sam", T0, {"courseId": "lathe-101", "memberId": "eve"})
    c.ok("completeCourse", "sam", T0, {"courseId": "lathe-101", "memberId": "eve"}, {
        "enrollment": enr("lathe-101", "eve", T0, T0), "certification": cert("eve", "lathe", "2026-03-02", "2027-03-02")})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "laser-101", "memberId": "eve"})
    c.ok("completeCourse", "sam", T0, {"courseId": "laser-101", "memberId": "eve"}, {
        "enrollment": enr("laser-101", "eve", T0, T0), "certification": cert("eve", "laser", "2026-03-02", "2027-03-02")})
    c.ok("listCertifications", "eve", T0, {"memberId": "eve"}, {"memberId": "eve", "certifications": [
        cert("eve", "laser", "2026-03-02", "2027-03-02"), cert("eve", "lathe", "2026-03-02", "2027-03-02")]})
    c.ok("listCertifications", "sam", T0, {"memberId": "bob"}, {"memberId": "bob", "certifications": []})
    c.refuse("listCertifications", "bob", T0, {"memberId": "ada"}, "forbidden")
    # 365 days after 2027-03-01 is 2028-02-29
    t1 = "2027-03-01T10:00Z"
    c.ok("enrollInCourse", "sam", t1, {"courseId": "laser-101", "memberId": "cy"})
    c.ok("completeCourse", "sam", t1, {"courseId": "laser-101", "memberId": "cy"}, {
        "enrollment": enr("laser-101", "cy", t1, t1), "certification": cert("cy", "laser", "2027-03-01", "2028-02-29")})
    # valid until the day before expiresOn, expired on it
    c.ok("listCertifications", "ada", "2027-03-01T23:59Z", {"memberId": "ada"}, {"memberId": "ada", "certifications": [
        cert("ada", "laser", "2026-03-02", "2027-03-02", "valid")]})
    t2 = "2027-03-02T00:00Z"
    c.ok("listCertifications", "ada", t2, {"memberId": "ada"}, {"memberId": "ada", "certifications": [
        cert("ada", "laser", "2026-03-02", "2027-03-02", "expired")]})
    # an expired certification can be revoked; revoked wins over expired; a revoked or missing one cannot be revoked
    c.ok("revokeCertification", "sam", t2, {"memberId": "ada", "familyId": "laser"}, {"certification": cert("ada", "laser", "2026-03-02", "2027-03-02", "revoked")})
    c.refuse("revokeCertification", "sam", t2, {"memberId": "ada", "familyId": "laser"}, "not_certified")
    c.refuse("revokeCertification", "sam", t2, {"memberId": "bob", "familyId": "laser"}, "not_certified")
    c.refuse("revokeCertification", "sam", t2, {"memberId": "ada", "familyId": "lathe"}, "not_certified")
    c.refuse("revokeCertification", "ada", t2, {"memberId": "ada", "familyId": "laser"}, "forbidden")
    c.refuse("revokeCertification", "sam", t2, {"memberId": "ada", "familyId": "nope"}, "not_found")
    # a new completion replaces a revoked one and an expired one; the lathe one stays expired
    t3 = "2027-03-03T09:00Z"
    c.ok("enrollInCourse", "sam", t3, {"courseId": "laser-101", "memberId": "ada"})
    c.ok("completeCourse", "sam", t3, {"courseId": "laser-101", "memberId": "ada"}, {
        "enrollment": enr("laser-101", "ada", t3, t3), "certification": cert("ada", "laser", "2027-03-03", "2028-03-02")})
    c.ok("enrollInCourse", "sam", t3, {"courseId": "laser-101", "memberId": "eve"})
    c.ok("completeCourse", "sam", t3, {"courseId": "laser-101", "memberId": "eve"}, {
        "enrollment": enr("laser-101", "eve", t3, t3), "certification": cert("eve", "laser", "2027-03-03", "2028-03-02")})
    c.ok("listCertifications", "eve", t3, {"memberId": "eve"}, {"memberId": "eve", "certifications": [
        cert("eve", "laser", "2027-03-03", "2028-03-02", "valid"), cert("eve", "lathe", "2026-03-02", "2027-03-02", "expired")]})
    c.ok("listCertifications", "sam", t3, {"memberId": "ada"}, {"memberId": "ada", "certifications": [cert("ada", "laser", "2027-03-03", "2028-03-02", "valid")]})
    c.audit_log(t3)
    return c.build()


# --- X-teaching ---------------------------------------------------------------------------------------------------------

def x_teaching_1():
    c = Case("X-teaching-1", "X-teaching", "complete course: attendance gate, then assessment gate; a class before the enrollment does not count; "
                                         "a re-enrollment starts a new period")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 10})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"})
    c.ok("addCourse", "sam", T0, {"courseId": "laser-201", "familyId": "laser", "title": "Laser advanced", "feeCents": 0, "capacity": 3})
    c.ok("defineAssessment", "sam", T0, {"courseId": "laser-201", "passMark": 50, "maxAttempts": 3})
    classes = [("cls-1", "2026-03-03T10:00Z", "2026-03-03T10:30Z"), ("cls-2", "2026-03-03T10:30Z", "2026-03-03T11:30Z"),
               ("cls-3", "2026-03-04T10:00Z", "2026-03-04T11:00Z"), ("cls-4", "2026-03-06T10:00Z", "2026-03-06T11:00Z"),
               ("cls-5", "2026-03-07T10:00Z", "2026-03-07T11:00Z")]
    for cid, s, e in classes:
        c.ok("scheduleClass", "sam", T0, {"courseId": "laser-201", "instructorId": "sam", "roomId": "lab", "start": s, "end": e},
             {"class": klass(cid, "laser-201", "sam", "lab", s, e)}, new=cid)
    course = lambda n: crs("laser-201", "laser", "Laser advanced", 0, 3, n)
    te = "2026-03-03T10:30Z"
    c.refuse("completeCourse", "sam", te, {"courseId": "laser-201", "memberId": "eve"}, "not_enrolled")
    c.ok("enrollInCourse", "bob", te, {"courseId": "laser-201", "memberId": "bob"}, {"enrollment": enr("laser-201", "bob", te), "course": course(1)})
    # cls-1 began before the enrollment: it does not count, whatever its marks
    c.ok("markAttendance", "sam", te, {"classId": "cls-1", "memberId": "bob", "present": False}, {"attendance": mark("cls-1", "bob", False)})
    # cls-2 began at the enrollment time: it counts. Attendance is checked before the assessment
    c.refuse("completeCourse", "sam", te, {"courseId": "laser-201", "memberId": "bob"}, "attendance_missing")
    c.ok("markAttendance", "sam", te, {"classId": "cls-2", "memberId": "bob", "present": False}, {"attendance": mark("cls-2", "bob", False)})
    c.refuse("completeCourse", "sam", te, {"courseId": "laser-201", "memberId": "bob"}, "attendance_missing")
    c.ok("markAttendance", "sam", te, {"classId": "cls-2", "memberId": "bob", "present": True}, {"attendance": mark("cls-2", "bob", True)})
    c.refuse("completeCourse", "sam", te, {"courseId": "laser-201", "memberId": "bob"}, "assessment_not_passed")
    c.ok("recordAttempt", "sam", te, {"courseId": "laser-201", "memberId": "bob", "score": 40}, {"attempt": attempt("laser-201", "bob", 1, 40, False, te)})
    c.refuse("completeCourse", "sam", te, {"courseId": "laser-201", "memberId": "bob"}, "assessment_not_passed")
    # cls-3 starts: a class that has started but not ended already requires attendance
    t4 = "2026-03-04T10:00Z"
    c.refuse("completeCourse", "sam", t4, {"courseId": "laser-201", "memberId": "bob"}, "attendance_missing")
    c.ok("recordAttempt", "sam", t4, {"courseId": "laser-201", "memberId": "bob", "score": 55}, {"attempt": attempt("laser-201", "bob", 2, 55, True, t4)})
    c.refuse("completeCourse", "sam", t4, {"courseId": "laser-201", "memberId": "bob"}, "attendance_missing")
    c.ok("markAttendance", "sam", t4, {"classId": "cls-3", "memberId": "bob", "present": True}, {"attendance": mark("cls-3", "bob", True)})
    t5 = "2026-03-04T10:30Z"
    c.ok("completeCourse", "sam", t5, {"courseId": "laser-201", "memberId": "bob"}, {
        "enrollment": enr("laser-201", "bob", te, t5), "certification": cert("bob", "laser", "2026-03-04", "2027-03-04")})
    # a new enrollment at the same time starts period 2: cls-3 began before it, the pass of period 1 does not count
    c.ok("enrollInCourse", "bob", t5, {"courseId": "laser-201", "memberId": "bob"}, {"enrollment": enr("laser-201", "bob", t5), "course": course(1)})
    c.refuse("completeCourse", "sam", t5, {"courseId": "laser-201", "memberId": "bob"}, "assessment_not_passed")
    c.ok("recordAttempt", "sam", t5, {"courseId": "laser-201", "memberId": "bob", "score": 50}, {"attempt": attempt("laser-201", "bob", 1, 50, True, t5)})
    # cls-4 is cancelled before it starts, cls-5 starts after the request time: neither requires attendance
    c.ok("cancelClass", "sam", "2026-03-05T09:00Z", {"classId": "cls-4"}, {"class": klass("cls-4", "laser-201", "sam", "lab", "2026-03-06T10:00Z", "2026-03-06T11:00Z", "cancelled")})
    t6 = "2026-03-06T10:00Z"
    c.ok("completeCourse", "sam", t6, {"courseId": "laser-201", "memberId": "bob"}, {
        "enrollment": enr("laser-201", "bob", t5, t6), "certification": cert("bob", "laser", "2026-03-06", "2027-03-06")})
    c.ok("listCertifications", "bob", t6, {"memberId": "bob"}, {"memberId": "bob", "certifications": [cert("bob", "laser", "2026-03-06", "2027-03-06")]})
    c.ok("listAttempts", "bob", t6, {"courseId": "laser-201", "memberId": "bob"}, {"courseId": "laser-201", "memberId": "bob", "attempts": [
        attempt("laser-201", "bob", 1, 40, False, te), attempt("laser-201", "bob", 2, 55, True, t4), attempt("laser-201", "bob", 1, 50, True, t5)]})
    c.ok("listCourseClasses", "bob", t6, {"courseId": "laser-201"}, {"courseId": "laser-201", "classes": [
        klass("cls-1", "laser-201", "sam", "lab", "2026-03-03T10:00Z", "2026-03-03T10:30Z"),
        klass("cls-2", "laser-201", "sam", "lab", "2026-03-03T10:30Z", "2026-03-03T11:30Z"),
        klass("cls-3", "laser-201", "sam", "lab", "2026-03-04T10:00Z", "2026-03-04T11:00Z"),
        klass("cls-4", "laser-201", "sam", "lab", "2026-03-06T10:00Z", "2026-03-06T11:00Z", "cancelled"),
        klass("cls-5", "laser-201", "sam", "lab", "2026-03-07T10:00Z", "2026-03-07T11:00Z")]})
    c.audit_log(t6)
    return c.build()


def x_teaching_2():
    c = Case("X-teaching-2", "X-teaching", "enroll in course: scholarship award against a redeemed promotion; higher percent wins, equal goes "
                                         "to the award, only the winner is used, a fund that cannot cover it, a course of fee 0")
    world(c)
    for m, tier in (("cy", "plus"), ("dan", "basic"), ("fay", "plus")):
        c.ok("registerMember", "sam", T0, {"memberId": m, "name": m.title(), "tier": tier})
    for cid, title, fee in (("pottery", "Pottery", 12000), ("weaving", "Weaving", 12000), ("free-intro", "Intro", 0), ("welding", "Welding", 12000)):
        c.ok("addCourse", "sam", T0, {"courseId": cid, "familyId": "laser", "title": title, "feeCents": fee, "capacity": 10},
             {"course": crs(cid, "laser", title, fee, 10, 0)})
    for fid, budget in (("FA", 10000), ("FB", 10000), ("FC", 3599), ("FD", 3600), ("FF", 3600)):
        c.ok("addFund", "admin", T0, {"fundId": fid, "budgetCents": budget}, {"fund": fund(fid, budget)})
    for n, (fid, m) in enumerate((("FA", "cy"), ("FB", "bob"), ("FC", "eve"), ("FD", "dan"), ("FF", "fay")), 1):
        c.ok("awardScholarship", "admin", T0, {"fundId": fid, "memberId": m, "percent": 30}, {"award": award(f"aws-{n}", fid, m, 30)}, new=f"aws-{n}")
    for code, pct, scope in (("P30", 30, "course"), ("P25", 25, "both"), ("P20", 20, "course"), ("P40", 40, "both")):
        c.ok("addPromotion", "admin", T0, {"code": code, "percent": pct, "scope": scope, "maxRedemptions": 10, "validUntil": "2026-12-31"},
             {"promotion": promo(code, pct, scope, 10, "2026-12-31", 0)})
    for code, pct, scope, m in (("P30", 30, "course", "cy"), ("P25", 25, "both", "bob"), ("P20", 20, "course", "eve"), ("P40", 40, "both", "fay")):
        c.ok("redeemPromotion", m, T0, {"memberId": m, "code": code}, {"redemption": redemption(code, m, pct, T0),
                                                                       "promotion": promo(code, pct, scope, 10, "2026-12-31", 1)})
    pot = lambda n: crs("pottery", "laser", "Pottery", 12000, 10, n)
    wea = lambda n: crs("weaving", "laser", "Weaving", 12000, 10, n)
    e1, e2, e3, e4 = "2026-03-03T09:00Z", "2026-03-04T09:00Z", "2026-03-04T09:30Z", "2026-03-05T09:00Z"
    # pottery, all at e1
    for n, m in enumerate(("cy", "bob", "eve", "dan", "fay"), 1):
        c.ok("enrollInCourse", m, e1, {"courseId": "pottery", "memberId": m}, {"enrollment": enr("pottery", m, e1), "course": pot(n)})
    c.ok("listMemberPromotions", "cy", e1, {"memberId": "cy"}, {"memberId": "cy", "redemptions": [redemption("P30", "cy", 30, T0)]})
    c.ok("listMemberPromotions", "bob", e1, {"memberId": "bob"}, {"memberId": "bob", "redemptions": [redemption("P25", "bob", 25, T0)]})
    c.ok("listMemberPromotions", "eve", e1, {"memberId": "eve"}, {"memberId": "eve", "redemptions": [redemption("P20", "eve", 20, T0, e1, "pottery")]})
    c.ok("listMemberPromotions", "fay", e1, {"memberId": "fay"}, {"memberId": "fay", "redemptions": [redemption("P40", "fay", 40, T0, e1, "pottery")]})
    c.ok("listMemberPromotions", "dan", e1, {"memberId": "dan"}, {"memberId": "dan", "redemptions": []})
    # weaving, all at e2
    for n, m in enumerate(("dan", "fay", "eve", "cy"), 1):
        c.ok("enrollInCourse", m, e2, {"courseId": "weaving", "memberId": m}, {"enrollment": enr("weaving", m, e2), "course": wea(n)})
    # cy: a course of fee 0 uses neither source; then the fund cannot cover a third course, the promotion wins
    c.ok("enrollInCourse", "cy", e3, {"courseId": "free-intro", "memberId": "cy"}, {
        "enrollment": enr("free-intro", "cy", e3), "course": crs("free-intro", "laser", "Intro", 0, 10, 1)})
    c.ok("listMemberPromotions", "cy", e3, {"memberId": "cy"}, {"memberId": "cy", "redemptions": [redemption("P30", "cy", 30, T0)]})
    c.ok("getAccount", "cy", e3, {"memberId": "cy"}, {"account": account("cy", 0, 0, 0, 16800)})
    c.ok("enrollInCourse", "cy", e4, {"courseId": "welding", "memberId": "cy"}, {
        "enrollment": enr("welding", "cy", e4), "course": crs("welding", "laser", "Welding", 12000, 10, 1)})
    c.ok("listMemberPromotions", "cy", e4, {"memberId": "cy"}, {"memberId": "cy", "redemptions": [redemption("P30", "cy", 30, T0, e4, "welding")]})
    # the invoices show the discounts decided
    inv_at, due = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    c.ok("generateInvoices", "sam", inv_at, {"month": "2026-03"})
    c.ok("listInvoices", "bob", inv_at, {"memberId": "bob"}, {"memberId": "bob", "invoices": [invoice("inv-2", "bob", "2026-03", inv_at, due, [
        membership("basic", 2500, inv_at), line("course", "pottery", 1, 12000, 30, 8400, e1)])]})
    c.ok("listInvoices", "cy", inv_at, {"memberId": "cy"}, {"memberId": "cy", "invoices": [invoice("inv-3", "cy", "2026-03", inv_at, due, [
        membership("plus", 4900, inv_at), line("course", "pottery", 1, 12000, 30, 8400, e1), line("course", "weaving", 1, 12000, 30, 8400, e2),
        line("course", "free-intro", 1, 0, 0, 0, e3), line("course", "welding", 1, 12000, 30, 8400, e4)])]})
    c.ok("listInvoices", "dan", inv_at, {"memberId": "dan"}, {"memberId": "dan", "invoices": [invoice("inv-4", "dan", "2026-03", inv_at, due, [
        membership("basic", 2500, inv_at), line("course", "pottery", 1, 12000, 30, 8400, e1), line("course", "weaving", 1, 12000, 0, 12000, e2)])]})
    c.ok("listInvoices", "eve", inv_at, {"memberId": "eve"}, {"memberId": "eve", "invoices": [invoice("inv-5", "eve", "2026-03", inv_at, due, [
        membership("pro", 9900, inv_at), line("course", "pottery", 1, 12000, 20, 9600, e1), line("course", "weaving", 1, 12000, 0, 12000, e2)])]})
    c.ok("listInvoices", "fay", inv_at, {"memberId": "fay"}, {"memberId": "fay", "invoices": [invoice("inv-6", "fay", "2026-03", inv_at, due, [
        membership("plus", 4900, inv_at), line("course", "pottery", 1, 12000, 40, 7200, e1), line("course", "weaving", 1, 12000, 30, 8400, e2)])]})
    c.audit_log(inv_at)
    return c.build()


def x_teaching_3():
    c = Case("X-teaching-3", "X-teaching", "a full course through the waiting list: promotion in join order with discounts and the charge, "
                                         "skipped members with their codes, the notice")
    world(c)
    for m, tier in (("cy", "plus"), ("dan", "basic"), ("fay", "basic")):
        c.ok("registerMember", "sam", T0, {"memberId": m, "name": m.title(), "tier": tier})
    c.ok("addCourse", "sam", T0, {"courseId": "pottery", "familyId": "laser", "title": "Pottery", "feeCents": 10000, "capacity": 2})
    c.ok("addFund", "admin", T0, {"fundId": "FW", "budgetCents": 50000})
    c.ok("awardScholarship", "admin", T0, {"fundId": "FW", "memberId": "cy", "percent": 20}, new="aws-1")
    c.ok("addPromotion", "admin", T0, {"code": "P50", "percent": 50, "scope": "course", "maxRedemptions": 5, "validUntil": "2026-12-31"})
    c.ok("redeemPromotion", "ada", T0, {"memberId": "ada", "code": "P50"})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "pottery", "memberId": "bob"})
    c.ok("enrollInCourse", "sam", T0, {"courseId": "pottery", "memberId": "eve"})
    # join order: dan, cy, fay, ada
    j = {"dan": "2026-03-03T09:00Z", "cy": "2026-03-03T09:05Z", "fay": "2026-03-03T09:10Z", "ada": "2026-03-03T09:15Z"}
    for m, at in j.items():
        c.ok("joinWaitlist", m, at, {"courseId": "pottery", "memberId": m}, {"entry": waiting("pottery", m, at)})
    c.ok("setMemberStanding", "sam", "2026-03-03T10:00Z", {"memberId": "dan", "standing": "suspended"})
    entries = lambda *ms: {"courseId": "pottery", "entries": [waiting("pottery", m, j[m]) for m in ms]}
    c.ok("listWaitlist", "sam", "2026-03-03T10:00Z", {"courseId": "pottery"}, entries("dan", "cy", "fay", "ada"))
    # the course is full: nothing happens
    c.ok("promoteWaitlist", "sam", "2026-03-03T10:00Z", {"courseId": "pottery"}, {"courseId": "pottery", "enrollments": [], "skipped": []})
    # one place is free: dan is skipped, cy is enrolled with the award (20 %: 8000), the course is full again
    t1 = "2026-03-05T10:00Z"
    c.ok("completeCourse", "sam", t1, {"courseId": "pottery", "memberId": "bob"}, {
        "enrollment": enr("pottery", "bob", T0, t1), "certification": cert("bob", "laser", "2026-03-05", "2027-03-05")})
    c.ok("promoteWaitlist", "sam", t1, {"courseId": "pottery"}, {"courseId": "pottery", "enrollments": [enr("pottery", "cy", t1)],
                                                                "skipped": [skipped("dan", "member_suspended")]})
    c.ok("listWaitlist", "sam", t1, {"courseId": "pottery"}, entries("dan", "fay", "ada"))
    c.ok("listNotices", "cy", t1, {"memberId": "cy"}, {"memberId": "cy", "notices": [notice("ntc-1", "cy", "pottery", t1)]})
    c.ok("getAccount", "cy", t1, {"memberId": "cy"}, {"account": account("cy", 0, 0, 0, 8000)})
    # two places free: dan skipped again, fay (enrolled directly meanwhile) skipped, ada enrolled with the promotion (50 %: 5000)
    t2 = "2026-03-07T10:00Z"
    c.ok("completeCourse", "sam", t2, {"courseId": "pottery", "memberId": "eve"})
    c.ok("completeCourse", "sam", t2, {"courseId": "pottery", "memberId": "cy"})
    c.ok("enrollInCourse", "sam", t2, {"courseId": "pottery", "memberId": "fay"}, {
        "enrollment": enr("pottery", "fay", t2), "course": crs("pottery", "laser", "Pottery", 10000, 2, 1)})
    c.ok("promoteWaitlist", "sam", t2, {"courseId": "pottery"}, {"courseId": "pottery", "enrollments": [enr("pottery", "ada", t2)], "skipped": [
        skipped("dan", "member_suspended"), skipped("fay", "already_enrolled")]})
    c.ok("listWaitlist", "sam", t2, {"courseId": "pottery"}, entries("dan", "fay"))
    c.ok("listMemberPromotions", "ada", t2, {"memberId": "ada"}, {"memberId": "ada", "redemptions": [redemption("P50", "ada", 50, T0, t2, "pottery")]})
    c.ok("listNotices", "ada", t2, {"memberId": "ada"}, {"memberId": "ada", "notices": [notice("ntc-2", "ada", "pottery", t2)]})
    c.ok("listNotices", "dan", t2, {"memberId": "dan"}, {"memberId": "dan", "notices": []})
    c.ok("listNotices", "fay", t2, {"memberId": "fay"}, {"memberId": "fay", "notices": []})
    # the charges: ada 5000 (laser-101) + 5000, cy 8000, dan none
    inv_at, due = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    c.ok("generateInvoices", "sam", inv_at, {"month": "2026-03"})
    c.ok("listInvoices", "ada", inv_at, {"memberId": "ada"}, {"memberId": "ada", "invoices": [invoice("inv-1", "ada", "2026-03", inv_at, due, [
        membership("plus", 4900, inv_at), line("course", "laser-101", 1, 5000, 0, 5000, T0), line("course", "pottery", 1, 10000, 50, 5000, t2)])]})
    c.ok("listInvoices", "cy", inv_at, {"memberId": "cy"}, {"memberId": "cy", "invoices": [invoice("inv-3", "cy", "2026-03", inv_at, due, [
        membership("plus", 4900, inv_at), line("course", "pottery", 1, 10000, 20, 8000, t1)])]})
    c.ok("listInvoices", "dan", inv_at, {"memberId": "dan"}, {"memberId": "dan", "invoices": [invoice("inv-4", "dan", "2026-03", inv_at, due, [
        membership("basic", 2500, inv_at)])]})
    c.ok("listInvoices", "fay", inv_at, {"memberId": "fay"}, {"memberId": "fay", "invoices": [invoice("inv-6", "fay", "2026-03", inv_at, due, [
        membership("basic", 2500, inv_at), line("course", "pottery", 1, 10000, 0, 10000, t2)])]})
    c.audit_log(inv_at)
    return c.build()


def x_teaching_4():
    c = Case("X-teaching-4", "X-teaching", "a class and an event compete for one room: cancelling one frees it for the other; "
                                         "holds of classes and events cannot be released directly")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "hall", "name": "Hall", "capacity": 20})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sam", "familyId": "laser"})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "sue", "familyId": "laser"})
    fs, fe = "2026-03-10T18:00Z", "2026-03-10T20:00Z"
    c.ok("createEvent", "sam", T0, {"eventId": "fair", "title": "Fair", "roomId": "hall", "start": fs, "end": fe, "capacity": 10, "priceCents": 500},
         {"event": event("fair", "Fair", "hall", fs, fe, 10, 500, 0)})

    def sc(ins, s, e):
        return {"courseId": "laser-101", "instructorId": ins, "roomId": "hall", "start": f"2026-03-10T{s}Z", "end": f"2026-03-10T{e}Z"}

    # the event holds the hall: a class during it is refused; a class that starts when it ends is accepted
    c.refuse("scheduleClass", "sam", T0, sc("sam", "19:00", "21:00"), "room_taken")
    c.ok("scheduleClass", "sam", T0, sc("sam", "20:00", "21:00"),
         {"class": klass("cls-1", "laser-101", "sam", "hall", "2026-03-10T20:00Z", "2026-03-10T21:00Z")}, new="cls-1")
    c.refuse("createEvent", "sam", T0, {"eventId": "fair2", "title": "Fair two", "roomId": "hall", "start": "2026-03-10T20:30Z", "end": "2026-03-10T22:00Z",
                                         "capacity": 30, "priceCents": 0}, "capacity_too_large")
    c.refuse("createEvent", "sam", T0, {"eventId": "fair2", "title": "Fair two", "roomId": "hall", "start": "2026-03-10T20:30Z", "end": "2026-03-10T22:00Z",
                                         "capacity": 10, "priceCents": 0}, "room_taken")
    c.refuse("createEvent", "sam", T0, {"eventId": "fair", "title": "Fair two", "roomId": "hall", "start": "2026-03-10T20:30Z", "end": "2026-03-10T22:00Z",
                                         "capacity": 30, "priceCents": 0}, "already_exists")
    c.ok("getRoomSchedule", "bob", T0, {"roomId": "hall", "date": "2026-03-10"}, {"roomId": "hall", "date": "2026-03-10", "holds": [
        hold("hld-1", "hall", fs, fe, "event fair"), hold("hld-2", "hall", "2026-03-10T20:00Z", "2026-03-10T21:00Z", "class cls-1")]})
    # holds made for a class or an event cannot be released directly
    c.refuse("releaseRoomHold", "sam", T0, {"holdId": "hld-1"}, "hold_not_direct")
    c.refuse("releaseRoomHold", "sam", T0, {"holdId": "hld-2"}, "hold_not_direct")
    c.refuse("holdRoom", "sam", T0, {"roomId": "hall", "start": "2026-03-10T19:00Z", "end": "2026-03-10T20:30Z", "purpose": "x"}, "room_taken")
    # cancelling the event frees the room for a class during it
    t1 = "2026-03-05T09:00Z"
    c.ok("cancelEvent", "sam", t1, {"eventId": "fair"}, {"event": event("fair", "Fair", "hall", fs, fe, 10, 500, 0, "cancelled"), "tickets": []})
    c.ok("scheduleClass", "sam", t1, sc("sue", "18:00", "20:00"),
         {"class": klass("cls-2", "laser-101", "sue", "hall", fs, fe)}, new="cls-2")
    # now the class holds the hall against an event
    fair3 = {"eventId": "fair3", "title": "Fair three", "roomId": "hall", "start": "2026-03-10T18:30Z", "end": "2026-03-10T19:30Z", "capacity": 10, "priceCents": 0}
    c.refuse("createEvent", "sam", t1, fair3, "room_taken")
    # cancelling the class frees it for the event; the hold of the cancelled class cannot be released again
    t2 = "2026-03-06T09:00Z"
    c.ok("cancelClass", "sam", t2, {"classId": "cls-2"}, {"class": klass("cls-2", "laser-101", "sue", "hall", fs, fe, "cancelled")})
    c.ok("createEvent", "sam", t2, fair3, {"event": event("fair3", "Fair three", "hall", "2026-03-10T18:30Z", "2026-03-10T19:30Z", 10, 0, 0)})
    c.refuse("releaseRoomHold", "sam", t2, {"holdId": "hld-3"}, "invalid_state")
    c.refuse("releaseRoomHold", "sam", t2, {"holdId": "hld-1"}, "invalid_state")
    c.refuse("releaseRoomHold", "sam", t2, {"holdId": "hld-4"}, "hold_not_direct")
    # a direct hold can be released
    c.ok("holdRoom", "sam", t2, {"roomId": "hall", "start": "2026-03-10T21:00Z", "end": "2026-03-10T22:00Z", "purpose": "cleaning"},
         {"hold": hold("hld-5", "hall", "2026-03-10T21:00Z", "2026-03-10T22:00Z", "cleaning")}, new="hld-5")
    c.ok("releaseRoomHold", "sam", t2, {"holdId": "hld-5"}, {"hold": hold("hld-5", "hall", "2026-03-10T21:00Z", "2026-03-10T22:00Z", "cleaning", "released")})
    c.ok("getRoomSchedule", "sam", t2, {"roomId": "hall", "date": "2026-03-10"}, {"roomId": "hall", "date": "2026-03-10", "holds": [
        hold("hld-4", "hall", "2026-03-10T18:30Z", "2026-03-10T19:30Z", "event fair3"),
        hold("hld-2", "hall", "2026-03-10T20:00Z", "2026-03-10T21:00Z", "class cls-1")]})
    c.audit_log(t2)
    return c.build()


def x_teaching_5():
    c = Case("X-teaching-5", "X-teaching", "an event with tickets sold, cancelled in time and too late, the whole event cancelled: credit, "
                                         "charges, invoices with credit applied, journal and trial balance")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "hall", "name": "Hall", "capacity": 50})
    es, ee = "2026-03-10T18:00Z", "2026-03-10T21:00Z"
    ev = lambda sold, status="open": event("fair", "Fair", "hall", es, ee, 2, 1200, sold, status)
    c.ok("createEvent", "sam", T0, {"eventId": "fair", "title": "Fair", "roomId": "hall", "start": es, "end": ee, "capacity": 2, "priceCents": 1200},
         {"event": ev(0)})
    ta, tb = "2026-03-03T10:00Z", "2026-03-03T10:05Z"
    c.ok("sellTicket", "ada", ta, {"eventId": "fair", "memberId": "ada"}, {"ticket": ticket("tkt-1", "fair", "ada"), "event": ev(1)}, new="tkt-1")
    c.ok("sellTicket", "bob", tb, {"eventId": "fair", "memberId": "bob"}, {"ticket": ticket("tkt-2", "fair", "bob"), "event": ev(2)}, new="tkt-2")
    c.refuse("sellTicket", "eve", tb, {"eventId": "fair", "memberId": "eve"}, "event_sold_out")
    c.refuse("sellTicket", "ada", tb, {"eventId": "fair", "memberId": "ada"}, "already_ticketed")
    # exactly 1440 minutes before the start: ada may cancel; a minute later bob may not
    t1 = "2026-03-09T18:00Z"
    c.ok("cancelTicket", "ada", t1, {"ticketId": "tkt-1"}, {"ticket": ticket("tkt-1", "fair", "ada", "cancelled"), "event": ev(1)})
    c.refuse("cancelTicket", "bob", "2026-03-09T18:01Z", {"ticketId": "tkt-2"}, "too_late")
    c.ok("getAccount", "ada", t1, {"memberId": "ada"}, {"account": account("ada", 1200, 0, 0, 6200)})
    # the place is free again and ada may buy another ticket
    t2 = "2026-03-09T18:30Z"
    c.ok("sellTicket", "ada", t2, {"eventId": "fair", "memberId": "ada"}, {"ticket": ticket("tkt-3", "fair", "ada"), "event": ev(2)}, new="tkt-3")
    c.refuse("sellTicket", "eve", t2, {"eventId": "fair", "memberId": "eve"}, "event_sold_out")
    # the whole event: the valid tickets in creation order
    t3 = "2026-03-10T09:00Z"
    c.ok("cancelEvent", "sam", t3, {"eventId": "fair"}, {"event": ev(0, "cancelled"), "tickets": [
        ticket("tkt-2", "fair", "bob", "cancelled"), ticket("tkt-3", "fair", "ada", "cancelled")]})
    c.refuse("sellTicket", "eve", t3, {"eventId": "fair", "memberId": "eve"}, "event_closed")
    c.refuse("cancelTicket", "bob", t3, {"ticketId": "tkt-2"}, "invalid_state")
    c.ok("listEventTickets", "sam", t3, {"eventId": "fair"}, {"eventId": "fair", "tickets": [
        ticket("tkt-1", "fair", "ada", "cancelled"), ticket("tkt-2", "fair", "bob", "cancelled"), ticket("tkt-3", "fair", "ada", "cancelled")]})
    c.ok("getAccount", "ada", t3, {"memberId": "ada"}, {"account": account("ada", 2400, 0, 0, 7400)})
    c.ok("getAccount", "bob", t3, {"memberId": "bob"}, {"account": account("bob", 1200, 0, 0, 1200)})
    c.ok("getAccount", "eve", t3, {"memberId": "eve"}, {"account": account("eve", 0, 0, 0, 0)})
    march = [jrn(1, t1, "revenueRefunds", "creditLiability", 1200, "tkt-1"), jrn(2, t3, "revenueRefunds", "creditLiability", 1200, "tkt-2"),
             jrn(3, t3, "revenueRefunds", "creditLiability", 1200, "tkt-3")]
    c.ok("listJournalEntries", "admin", t3, {"month": "2026-03"}, {"month": "2026-03", "entries": march})
    # the invoices: the cancelled tickets' charges are still on them and the credit is applied
    inv_at, due = "2026-04-01T00:00Z", "2026-04-15T00:00Z"
    ia = invoice("inv-1", "ada", "2026-03", inv_at, due, [
        membership("plus", 4900, inv_at), line("course", "laser-101", 1, 5000, 0, 5000, T0),
        line("event", "tkt-1", 1, 1200, 0, 1200, ta), line("event", "tkt-3", 1, 1200, 0, 1200, t2)], credit=2400)
    ib = invoice("inv-2", "bob", "2026-03", inv_at, due, [membership("basic", 2500, inv_at), line("event", "tkt-2", 1, 1200, 0, 1200, tb)], credit=1200)
    ie = invoice("inv-3", "eve", "2026-03", inv_at, due, [membership("pro", 9900, inv_at)])
    assert (ia["totalCents"], ia["balanceCents"], ib["totalCents"], ib["balanceCents"], ie["balanceCents"]) == (12300, 9900, 3700, 2500, 9900)
    c.ok("generateInvoices", "sam", inv_at, {"month": "2026-03"}, {"month": "2026-03", "invoices": [ia, ib, ie]})
    april = [jrn(4, inv_at, "receivable", "revenueMembership", 4900, "inv-1"), jrn(5, inv_at, "receivable", "revenueCourse", 5000, "inv-1"),
             jrn(6, inv_at, "receivable", "revenueEvents", 1200, "inv-1"), jrn(7, inv_at, "receivable", "revenueEvents", 1200, "inv-1"),
             jrn(8, inv_at, "creditLiability", "receivable", 2400, "inv-1"),
             jrn(9, inv_at, "receivable", "revenueMembership", 2500, "inv-2"), jrn(10, inv_at, "receivable", "revenueEvents", 1200, "inv-2"),
             jrn(11, inv_at, "creditLiability", "receivable", 1200, "inv-2"),
             jrn(12, inv_at, "receivable", "revenueMembership", 9900, "inv-3")]
    c.ok("listJournalEntries", "admin", inv_at, {"month": "2026-04"}, {"month": "2026-04", "entries": april})
    c.ok("getTrialBalance", "admin", inv_at, {}, trial(march + april))
    c.ok("getAccount", "ada", inv_at, {"memberId": "ada"}, {"account": account("ada", 0, 9900, 0, 0)})
    c.audit_log(inv_at)
    return c.build()


def x_teaching_6():
    c = Case("X-teaching-6", "X-teaching", "payouts over a month of classes: minutes, rounding, once only, the class that ends in the next "
                                         "month, settled, in the journal and the trial balance")
    world(c)
    c.ok("addRoom", "admin", T0, {"roomId": "lab", "name": "Lab", "capacity": 10})
    for u, name in (("sal", "Sal Staff"), ("tom", "Tom Staff")):
        c.ok("registerUser", "admin", T0, {"userId": u, "name": name, "role": "staff"})
    rates = {"sal": 29, "sam": 4500, "sue": 90, "tom": 30}
    for u, r in rates.items():
        c.ok("qualifyInstructor", "admin", T0, {"userId": u, "familyId": "laser"})
        c.ok("setInstructorRate", "admin", T0, {"userId": u, "centsPerHour": r})
    c.ok("qualifyInstructor", "admin", T0, {"userId": "ada", "familyId": "laser"})
    plan = [("sam", "2026-03-10T10:00Z", "2026-03-10T11:30Z"), ("sam", "2026-03-11T10:00Z", "2026-03-11T10:45Z"),
            ("sam", "2026-03-20T10:00Z", "2026-03-20T12:00Z"), ("sam", "2026-03-31T23:30Z", "2026-04-01T01:00Z"),
            ("sal", "2026-03-12T10:00Z", "2026-03-12T10:01Z"), ("sue", "2026-03-13T10:00Z", "2026-03-13T10:01Z"),
            ("tom", "2026-03-14T10:00Z", "2026-03-14T10:01Z"), ("ada", "2026-03-15T10:00Z", "2026-03-15T11:00Z"),
            ("sam", "2026-04-02T10:00Z", "2026-04-02T11:00Z")]
    for n, (who, s, e) in enumerate(plan, 1):
        c.ok("scheduleClass", "sam", T0, {"courseId": "laser-101", "instructorId": who, "roomId": "lab", "start": s, "end": e},
             {"class": klass(f"cls-{n}", "laser-101", who, "lab", s, e)}, new=f"cls-{n}")
    c.ok("cancelClass", "sam", T0, {"classId": "cls-3"})
    # tom is withdrawn after his class; his rate stays and he is paid
    t0 = "2026-03-31T23:59Z"
    c.ok("withdrawInstructor", "admin", t0, {"userId": "tom", "familyId": "laser"}, {"qualification": qual("tom", "laser")})
    c.refuse("generatePayouts", "admin", t0, {"month": "2026-03"}, "month_not_ended")
    t1 = "2026-04-01T00:00Z"
    # sam: 90 + 45 + 90 = 225 minutes at 4500: (225 x 4500 + 30) div 60 = 16875; the cancelled class does not count and the April class
    # belongs to April. sal: (29 + 30) div 60 = 0. sue: (90 + 30) div 60 = 2. tom: (30 + 30) div 60 = 1. ada has no rate.
    p = {"sal": payout("pyt-1", "sal", "2026-03", 1, 0), "sam": payout("pyt-2", "sam", "2026-03", 225, 16875),
         "sue": payout("pyt-3", "sue", "2026-03", 1, 2), "tom": payout("pyt-4", "tom", "2026-03", 1, 1)}
    c.ok("generatePayouts", "admin", t1, {"month": "2026-03"}, {"month": "2026-03", "payouts": [p["sal"], p["sam"], p["sue"], p["tom"]]})
    c.refuse("generatePayouts", "admin", t1, {"month": "2026-03"}, "already_generated")
    entries = [jrn(1, t1, "expensePayout", "payable", 16875, "pyt-2"), jrn(2, t1, "expensePayout", "payable", 2, "pyt-3"),
               jrn(3, t1, "expensePayout", "payable", 1, "pyt-4")]
    c.ok("listJournalEntries", "admin", t1, {"month": "2026-04"}, {"month": "2026-04", "entries": entries})
    # settling: sam's payout posts payable to cash; a payout of 0 can be settled and posts nothing
    t2 = "2026-04-02T09:00Z"
    sam_settled = dict(p["sam"], status="settled")
    c.ok("settlePayout", "admin", t2, {"payoutId": "pyt-2"}, {"payout": sam_settled})
    c.ok("settlePayout", "admin", t2, {"payoutId": "pyt-1"}, {"payout": dict(p["sal"], status="settled")})
    c.refuse("settlePayout", "admin", t2, {"payoutId": "pyt-2"}, "invalid_state")
    entries.append(jrn(4, t2, "payable", "cash", 16875, "pyt-2"))
    # April's payouts: the class of 2026-04-02 only; the class that began in March does not count again
    t3 = "2026-05-01T00:00Z"
    p5 = payout("pyt-5", "sam", "2026-04", 60, 4500)
    c.ok("generatePayouts", "admin", t3, {"month": "2026-04"}, {"month": "2026-04", "payouts": [p5]})
    entries.append(jrn(5, t3, "expensePayout", "payable", 4500, "pyt-5"))
    c.ok("listPayouts", "admin", t3, {"userId": "sam"}, {"userId": "sam", "payouts": [sam_settled, p5]})
    c.ok("listJournalEntries", "admin", t3, {"month": "2026-04"}, {"month": "2026-04", "entries": entries[:4]})
    c.ok("listJournalEntries", "admin", t3, {"month": "2026-05"}, {"month": "2026-05", "entries": entries[4:]})
    c.ok("getTrialBalance", "admin", t3, {}, trial(entries))
    c.audit_log(t3)
    return c.build()


def build_cases() -> list:
    return [b_rooms_1(), b_rooms_2(), b_instructors_1(), b_instructors_2(), b_classes_1(), b_classes_2(), b_assessments_1(), b_assessments_2(),
            b_waitlists_1(), b_waitlists_2(), b_payouts_1(), b_payouts_2(), b_events_1(), b_events_2(), b_training_1(), b_training_2(),
            x_teaching_1(), x_teaching_2(), x_teaching_3(), x_teaching_4(), x_teaching_5(), x_teaching_6()]
