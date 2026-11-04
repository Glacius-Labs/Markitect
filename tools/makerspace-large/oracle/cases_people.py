"""The people cluster of the large Makerspace oracle: notices, households, guardians, waivers, ppe, staffing, guests,
volunteering and access, written from the requirements document alone.

B-<module>-<n> cases check one module's behavior; X-people-<n> cases chain several modules. Every expected value is
computed by hand from the document; the helpers below only build record shapes and apply the money formula.
"""
from __future__ import annotations

from kit import Case, world, has, ref, ANY  # noqa: F401

W = "2026-03-02T09:00Z"          # the start of world(): a Monday
INVOICE_AT = "2026-04-01T10:00Z"  # a Wednesday, after March has ended
INVOICE_DUE = "2026-04-15T10:00Z"  # 14 days later, a Wednesday
FEE = {"basic": 2500, "plus": 4900, "pro": 9900}


# ---------------------------------------------------------------- record shapes


def amt(quantity: int, unit: int, discount: int) -> int:
    return (quantity * unit * (100 - discount) + 50) // 100


def line(kind: str, rf: str, quantity: int, unit: int, discount: int, at: str) -> dict:
    return {"kind": kind, "ref": rf, "quantity": quantity, "unitPriceCents": unit, "discountPercent": discount,
            "amountCents": amt(quantity, unit, discount), "at": at}


# world() enrolls ada in laser-101 at W: a course charge of 5000 cents that her invoice of March 2026 carries
COURSE = line("course", "laser-101", 1, 5000, 0, W)


def invoice(iid: str, mid: str, month: str, issued: str, due: str, lines: list, credit: int = 0, paid: int = 0,
            overdue: bool = False) -> dict:
    total = sum(item["amountCents"] for item in lines)
    balance = total - credit - paid
    return {"invoiceId": iid, "memberId": mid, "month": month, "issuedAt": issued, "dueAt": due, "lines": lines,
            "totalCents": total, "creditAppliedCents": credit, "paidCents": paid, "balanceCents": balance,
            "status": "paid" if balance == 0 else "open", "overdue": overdue}


def plain(iid: str, mid: str, tier: str, month: str, issued: str, due: str, discount: int = 0, extra: tuple = (),
          credit: int = 0) -> dict:
    return invoice(iid, mid, month, issued, due, [line("membership", tier, 1, FEE[tier], discount, issued), *extra], credit)


def account(mid: str, credit: int = 0, open_: int = 0, overdue: int = 0, unbilled: int = 0) -> dict:
    return {"account": {"memberId": mid, "creditBalanceCents": credit, "openBalanceCents": open_,
                        "overdueBalanceCents": overdue, "unbilledCents": unbilled}}


def notice(nid: str, mid: str, subject: str, at: str, ack: str | None = None) -> dict:
    return {"noticeId": nid, "memberId": mid, "kind": "manual", "subject": subject, "ref": None, "at": at,
            "acknowledgedAt": ack}


def household(hid: str, name: str, payer: str, members: list) -> dict:
    return {"householdId": hid, "name": name, "payerId": payer, "memberIds": list(members)}


def guardianship(mid: str, birth: str, guardian: str, families: list) -> dict:
    return {"memberId": mid, "birthDate": birth, "guardianId": guardian, "consentFamilyIds": list(families)}


def waiver(family: str, version: int, text: str) -> dict:
    return {"familyId": family, "version": version, "text": text}


def signature(mid: str, family: str, version: int, at: str) -> dict:
    return {"memberId": mid, "familyId": family, "version": version, "signedAt": at}


def ppe(iid: str, name: str, stock: int) -> dict:
    return {"itemId": iid, "name": name, "stock": stock}


def shift(sid: str, uid: str, start: str, end: str, status: str = "scheduled") -> dict:
    return {"shiftId": sid, "userId": uid, "start": start, "end": end, "status": status}


def gpass(gid: str, mid: str, guest: str, date: str, charged: bool, status: str = "issued") -> dict:
    return {"guestPassId": gid, "memberId": mid, "guestName": guest, "date": date, "charged": charged, "status": status}


def vol(eid: str, mid: str, date: str, minutes: int, note: str, status: str = "pending", credit: int | None = None) -> dict:
    return {"entryId": eid, "memberId": mid, "date": date, "minutes": minutes, "note": note, "status": status,
            "creditCents": credit}


def badge(bid: str, mid: str, status: str = "active", inside: bool = False) -> dict:
    return {"badgeId": bid, "memberId": mid, "status": status, "inside": inside}


def door(eid: str, kind: str, bid: str, at: str, pass_id: str | None = None) -> dict:
    return {"eventId": eid, "kind": kind, "badgeId": bid, "guestPassId": pass_id, "at": at}


def resv(rid: str, mid: str, machine: str, start: str, end: str, status: str = "reserved", cin: str | None = None,
         cout: str | None = None, rate: int | None = None, metered: int | None = None, amount: int | None = None) -> dict:
    return {"reservationId": rid, "memberId": mid, "machineId": machine, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": metered,
            "amountCents": amount}


def lockout(lid: str, machine: str, reason: str, by: str, at: str, released: str | None = None) -> dict:
    return {"lockoutId": lid, "machineId": machine, "source": "manual", "reason": reason, "appliedBy": by,
            "appliedAt": at, "releasedAt": released}


def journal(eid: str, at: str, debit: str, credit: str, amount: int, memo: str) -> dict:
    return {"entryId": eid, "at": at, "debitAccount": debit, "creditAccount": credit, "amountCents": amount, "memo": memo}


def register(c: Case, at: str, *ids: str, tier: str = "basic") -> None:
    for mid in ids:
        c.ok("registerMember", "sam", at, {"memberId": mid, "name": mid.capitalize(), "tier": tier})


# ---------------------------------------------------------------- notices


def b_notices_1() -> dict:
    c = Case("B-notices-1", "B-notices", "notices: post notice, list notices, acknowledge notice")
    t = world(c)
    t2, t3, t4 = "2026-03-02T09:30Z", "2026-03-02T10:00Z", "2026-03-02T11:00Z"
    c.ok("listNotices", "sam", t, {"memberId": "ada"}, {"memberId": "ada", "notices": []})
    c.ok("postNotice", "sam", t, {"memberId": "ada", "subject": "Welcome, Ada"},
         {"notice": notice("ntc-1", "ada", "Welcome, Ada", t)}, new="ntc-1")
    c.ok("postNotice", "amy", t2, {"memberId": "bob", "subject": " Padded  "},
         {"notice": notice("ntc-2", "bob", " Padded  ", t2)}, new="ntc-2")
    c.ok("postNotice", "sam", t2, {"memberId": "ada", "subject": "Second"},
         {"notice": notice("ntc-3", "ada", "Second", t2)}, new="ntc-3")
    c.ok("listNotices", "ada", t2, {"memberId": "ada"},
         {"memberId": "ada", "notices": [notice("ntc-1", "ada", "Welcome, Ada", t), notice("ntc-3", "ada", "Second", t2)]})
    c.refuse("listNotices", "bob", t2, {"memberId": "ada"}, "forbidden")
    # ownership (check 6) is judged before the state of the notice (check 7)
    c.refuse("acknowledgeNotice", "bob", t3, {"noticeId": "ntc-1"}, "forbidden")
    c.ok("acknowledgeNotice", "ada", t3, {"noticeId": "ntc-1"}, {"notice": notice("ntc-1", "ada", "Welcome, Ada", t, t3)})
    c.refuse("acknowledgeNotice", "bob", t3, {"noticeId": "ntc-1"}, "forbidden")
    c.refuse("acknowledgeNotice", "ada", t3, {"noticeId": "ntc-1"}, "invalid_state")
    c.refuse("acknowledgeNotice", "ada", t3, {"noticeId": "ntc-2"}, "forbidden")
    c.ok("acknowledgeNotice", "sam", t4, {"noticeId": "ntc-2"}, {"notice": notice("ntc-2", "bob", " Padded  ", t2, t4)})
    c.refuse("acknowledgeNotice", "sam", t4, {"noticeId": "ntc-2"}, "invalid_state")
    c.ok("listNotices", "ada", t4, {"memberId": "ada"},
         {"memberId": "ada", "notices": [notice("ntc-1", "ada", "Welcome, Ada", t, t3), notice("ntc-3", "ada", "Second", t2)]})
    c.ok("listNotices", "amy", t4, {"memberId": "bob"},
         {"memberId": "bob", "notices": [notice("ntc-2", "bob", " Padded  ", t2, t4)]})
    c.ok("listNotices", "sam", t4, {"memberId": "eve"}, {"memberId": "eve", "notices": []})
    c.audit_log(t4)
    return c.build()


def b_notices_2() -> dict:
    c = Case("B-notices-2", "B-notices", "notices: refusals of post notice and their order, numbering after refusals")
    t = world(c)
    early = "2026-03-02T08:59Z"
    ok_args = {"memberId": "ada", "subject": "Hi"}
    c.refuse("postNotice", "ada", t, {"memberId": "bob", "subject": "Hi"}, "forbidden")
    c.refuse("postNotice", "ada", t, {"memberId": "nobody", "subject": "Hi"}, "forbidden")  # check 4 before 5
    c.refuse("postNotice", "sam", t, {"memberId": "sam", "subject": "Hi"}, "not_found")  # a staff id names no member
    c.refuse("postNotice", "sam", t, {"memberId": "nobody", "subject": "Hi"}, "not_found")
    c.refuse("postNotice", "sam", t, {"memberId": "ada", "subject": "  \t "}, "invalid_argument")
    c.refuse("postNotice", "sam", t, {"memberId": "ada"}, "invalid_argument")
    c.refuse("postNotice", "sam", t, {"memberId": "ada", "subject": None}, "invalid_argument")
    c.refuse("postNotice", "ghost", t, {"memberId": "ada", "subject": "   "}, "invalid_argument")  # check 1 before 2
    c.refuse("postNotice", "bad id!", t, ok_args, "invalid_argument")
    c.refuse("postNotice", "ghost", t, ok_args, "unknown_actor")
    c.refuse("postNotice", "sam", early, ok_args, "time_regression")
    c.refuse("postNotice", "ghost", early, ok_args, "unknown_actor")  # check 2 before 3
    c.refuse("postNotice", "ada", early, ok_args, "time_regression")  # check 3 before 4
    c.invalid("postNotice", "sam", t, {**ok_args, "extra": 1})
    c.invalid("postNotice", "sam", t, {**ok_args, "Subject": "Hi"})
    # nothing above consumed a number or moved the request time
    c.ok("postNotice", "sam", t, {"memberId": "bob", "subject": "Real"}, {"notice": notice("ntc-1", "bob", "Real", t)},
         new="ntc-1")
    c.refuse("acknowledgeNotice", "ada", t, {"noticeId": "ntc-2"}, "not_found")  # check 5 before 6
    c.refuse("acknowledgeNotice", "bob", t, {"noticeId": "ntc-1x"}, "not_found")
    c.refuse("acknowledgeNotice", "ada", t, {"noticeId": "ntc-1"}, "forbidden")
    c.ok("listNotices", "ada", t, {"memberId": "ada"}, {"memberId": "ada", "notices": []})
    c.audit_log(t)
    return c.build()


# ---------------------------------------------------------------- households


def b_households_1() -> dict:
    c = Case("B-households-1", "B-households", "households: create, add, remove, get; refusals and their order, full household")
    t = world(c)
    register(c, t, "cat", "dan", "fay", "gil", "hal")
    c.ok("createHousehold", "sam", t, {"householdId": "hh1", "name": "Marsh home", "payerId": "eve"},
         {"household": household("hh1", "Marsh home", "eve", ["eve"])})
    c.refuse("createHousehold", "sam", t, {"householdId": "hh1", "name": "Other", "payerId": "eve"}, "already_exists")
    c.refuse("createHousehold", "sam", t, {"householdId": "hh2", "name": "Other", "payerId": "eve"}, "already_in_household")
    c.refuse("createHousehold", "ada", t, {"householdId": "hh2", "name": "Other", "payerId": "cat"}, "forbidden")
    c.refuse("createHousehold", "sam", t, {"householdId": "hh2", "name": "Other", "payerId": "sam"}, "not_found")
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "cat"},
         {"household": household("hh1", "Marsh home", "eve", ["cat", "eve"])})
    c.ok("addHouseholdMember", "amy", t, {"householdId": "hh1", "memberId": "ada"},
         {"household": household("hh1", "Marsh home", "eve", ["ada", "cat", "eve"])})
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "bob"},
         {"household": household("hh1", "Marsh home", "eve", ["ada", "bob", "cat", "eve"])})
    c.ok("getHousehold", "sam", t, {"householdId": "hh1"},
         {"household": household("hh1", "Marsh home", "eve", ["ada", "bob", "cat", "eve"])})
    c.refuse("getHousehold", "ada", t, {"householdId": "hh1"}, "forbidden")
    c.refuse("getHousehold", "sam", t, {"householdId": "nope"}, "not_found")
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "dan"},
         {"household": household("hh1", "Marsh home", "eve", ["ada", "bob", "cat", "dan", "eve"])})
    full = ["ada", "bob", "cat", "dan", "eve", "fay"]
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "fay"},
         {"household": household("hh1", "Marsh home", "eve", full)})
    c.refuse("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "gil"}, "household_full")
    c.refuse("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "ada"}, "already_in_household")  # first
    c.refuse("addHouseholdMember", "ada", t, {"householdId": "hh1", "memberId": "gil"}, "forbidden")
    c.refuse("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "sam"}, "not_found")
    c.ok("createHousehold", "sam", t, {"householdId": "hh2", "name": "Second home", "payerId": "gil"},
         {"household": household("hh2", "Second home", "gil", ["gil"])})
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh2", "memberId": "hal"},
         {"household": household("hh2", "Second home", "gil", ["gil", "hal"])})
    # a member of another household and a full household: the membership refusal comes first
    c.refuse("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "hal"}, "already_in_household")
    c.refuse("addHouseholdMember", "sam", t, {"householdId": "hh2", "memberId": "ada"}, "already_in_household")
    c.refuse("removeHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "eve"}, "payer_cannot_leave")
    c.refuse("removeHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "hal"}, "not_in_household")
    # the payer of another household is not in this one: not_in_household, not payer_cannot_leave
    c.refuse("removeHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "gil"}, "not_in_household")
    c.refuse("removeHouseholdMember", "ada", t, {"householdId": "hh1", "memberId": "fay"}, "forbidden")
    c.ok("removeHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "fay"},
         {"household": household("hh1", "Marsh home", "eve", ["ada", "bob", "cat", "dan", "eve"])})
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "fay"},
         {"household": household("hh1", "Marsh home", "eve", full)})  # the same household again
    c.ok("removeHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "fay"},
         {"household": household("hh1", "Marsh home", "eve", ["ada", "bob", "cat", "dan", "eve"])})
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh2", "memberId": "fay"},
         {"household": household("hh2", "Second home", "gil", ["fay", "gil", "hal"])})
    c.ok("getHousehold", "amy", t, {"householdId": "hh1"},
         {"household": household("hh1", "Marsh home", "eve", ["ada", "bob", "cat", "dan", "eve"])})
    c.audit_log(t)
    return c.build()


def b_households_2() -> dict:
    c = Case("B-households-2", "B-households",
             "households: household discount on the membership line, judged when the invoices are generated")
    t = world(c)
    register(c, t, "cat")
    c.ok("createHousehold", "sam", t, {"householdId": "hh1", "name": "Home", "payerId": "ada"},
         {"household": household("hh1", "Home", "ada", ["ada"])})
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "bob"})
    # cat belongs to no household
    c.ok("addHouseholdMember", "sam", t, {"householdId": "hh1", "memberId": "eve"},
         {"household": household("hh1", "Home", "ada", ["ada", "bob", "eve"])})
    # three members: ada plus 4900 -> 4410, bob basic 2500 -> 2250, eve pro 9900 -> 8910; cat is outside
    march = "2026-03"
    c.ok("generateInvoices", "sam", INVOICE_AT, {"month": march},
         {"month": march, "invoices": [
             plain("inv-1", "ada", "plus", march, INVOICE_AT, INVOICE_DUE, discount=10, extra=(COURSE,)),
             plain("inv-2", "bob", "basic", march, INVOICE_AT, INVOICE_DUE, discount=10),
             plain("inv-3", "cat", "basic", march, INVOICE_AT, INVOICE_DUE),
             plain("inv-4", "eve", "pro", march, INVOICE_AT, INVOICE_DUE, discount=10)]})
    c.ok("getAccount", "bob", INVOICE_AT, {"memberId": "bob"}, account("bob", open_=2250))
    c.ok("removeHouseholdMember", "sam", INVOICE_AT, {"householdId": "hh1", "memberId": "eve"},
         {"household": household("hh1", "Home", "ada", ["ada", "bob"])})
    # the household is judged when the request is made, whatever month is invoiced: two members, no discount
    t2, due2, april = "2026-05-01T10:00Z", "2026-05-15T10:00Z", "2026-04"
    c.ok("generateInvoices", "amy", t2, {"month": april},
         {"month": april, "invoices": [
             plain("inv-5", "ada", "plus", april, t2, due2),
             plain("inv-6", "bob", "basic", april, t2, due2),
             plain("inv-7", "cat", "basic", april, t2, due2),
             plain("inv-8", "eve", "pro", april, t2, due2)]})
    c.audit_log(t2)
    return c.build()


# ---------------------------------------------------------------- guardians


def b_guardians_1() -> dict:
    c = Case("B-guardians-1", "B-guardians", "guardians: declare minor, get guardianship, refusals and their order")
    t = world(c)  # the request date is 2026-03-02
    register(c, t, "mom", "kid", "teen")
    c.ok("declareMinor", "sam", t, {"memberId": "kid", "birthDate": "2012-06-15", "guardianId": "mom"},
         {"guardianship": guardianship("kid", "2012-06-15", "mom", [])})
    kid = {"guardianship": guardianship("kid", "2012-06-15", "mom", [])}
    c.ok("getGuardianship", "kid", t, {"memberId": "kid"}, kid)
    c.ok("getGuardianship", "sam", t, {"memberId": "kid"}, kid)
    c.ok("getGuardianship", "amy", t, {"memberId": "kid"}, kid)
    c.refuse("getGuardianship", "mom", t, {"memberId": "kid"}, "forbidden")  # ownership is judged on memberId
    c.refuse("getGuardianship", "bob", t, {"memberId": "kid"}, "forbidden")
    c.refuse("getGuardianship", "sam", t, {"memberId": "ada"}, "not_a_declared_minor")
    c.refuse("getGuardianship", "ada", t, {"memberId": "ada"}, "not_a_declared_minor")
    c.refuse("getGuardianship", "sam", t, {"memberId": "sam"}, "not_found")
    c.refuse("declareMinor", "sam", t, {"memberId": "kid", "birthDate": "2000-01-01", "guardianId": "kid"}, "already_exists")
    # the 18th birthday is on the day itself: adult on 2026-03-02, a minor until 2026-03-03
    c.refuse("declareMinor", "sam", t, {"memberId": "ada", "birthDate": "2008-03-02", "guardianId": "bob"}, "not_a_minor")
    c.refuse("declareMinor", "sam", t, {"memberId": "ada", "birthDate": "2000-01-01", "guardianId": "ada"}, "not_a_minor")
    c.ok("declareMinor", "sam", t, {"memberId": "teen", "birthDate": "2008-03-03", "guardianId": "mom"},
         {"guardianship": guardianship("teen", "2008-03-03", "mom", [])})
    c.refuse("declareMinor", "sam", t, {"memberId": "bob", "birthDate": "2015-01-01", "guardianId": "bob"}, "invalid_guardian")
    c.refuse("declareMinor", "sam", t, {"memberId": "bob", "birthDate": "2015-01-01", "guardianId": "kid"}, "invalid_guardian")
    c.refuse("declareMinor", "sam", t, {"memberId": "bob", "birthDate": "2015-01-01", "guardianId": "teen"}, "invalid_guardian")
    # a birth date after the request date is no error: the member is a minor
    c.ok("declareMinor", "amy", t, {"memberId": "bob", "birthDate": "2030-01-01", "guardianId": "mom"},
         {"guardianship": guardianship("bob", "2030-01-01", "mom", [])})
    c.ok("getGuardianship", "bob", t, {"memberId": "bob"}, {"guardianship": guardianship("bob", "2030-01-01", "mom", [])})
    c.refuse("declareMinor", "sam", t, {"memberId": "bob", "birthDate": "2015-01-01", "guardianId": "mom"}, "already_exists")
    c.refuse("declareMinor", "mom", t, {"memberId": "eve", "birthDate": "2015-01-01", "guardianId": "mom"}, "forbidden")
    c.refuse("declareMinor", "sam", t, {"memberId": "sam", "birthDate": "2015-01-01", "guardianId": "mom"}, "not_found")
    c.refuse("declareMinor", "sam", t, {"memberId": "eve", "birthDate": "2015-01-01", "guardianId": "sam"}, "not_found")
    c.refuse("declareMinor", "sam", t, {"memberId": "eve", "birthDate": "2026-02-29", "guardianId": "mom"}, "invalid_argument")
    c.refuse("declareMinor", "sam", t, {"memberId": "eve", "birthDate": "1999-12-31", "guardianId": "mom"}, "invalid_argument")
    c.audit_log(t)
    return c.build()


def b_guardians_2() -> dict:
    c = Case("B-guardians-2", "B-guardians", "guardians: grant consent, withdraw consent, ownership on the guardian, order")
    t = world(c)
    register(c, t, "mom", "dad", "kid")
    c.ok("declareMinor", "sam", t, {"memberId": "kid", "birthDate": "2012-06-15", "guardianId": "mom"})

    def g(*families: str) -> dict:
        return {"guardianship": guardianship("kid", "2012-06-15", "mom", list(families))}

    c.ok("grantConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "lathe"}, g("lathe"))
    c.ok("grantConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"}, g("laser", "lathe"))
    c.ok("grantConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"}, g("laser", "lathe"))
    c.refuse("grantConsent", "bob", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"}, "forbidden")
    c.refuse("grantConsent", "dad", t, {"guardianId": "dad", "memberId": "kid", "familyId": "laser"}, "not_guardian")
    c.refuse("grantConsent", "sam", t, {"guardianId": "dad", "memberId": "kid", "familyId": "laser"}, "not_guardian")
    c.refuse("grantConsent", "mom", t, {"guardianId": "mom", "memberId": "ada", "familyId": "laser"}, "not_guardian")
    c.refuse("grantConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "nope"}, "not_found")
    c.refuse("grantConsent", "sam", t, {"guardianId": "sam", "memberId": "kid", "familyId": "laser"}, "not_found")
    c.refuse("grantConsent", "sam", t, {"guardianId": "mom", "memberId": "nobody", "familyId": "laser"}, "not_found")
    c.ok("grantConsent", "sam", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"}, g("laser", "lathe"))
    c.refuse("withdrawConsent", "bob", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"}, "forbidden")
    c.refuse("withdrawConsent", "dad", t, {"guardianId": "dad", "memberId": "kid", "familyId": "laser"}, "not_guardian")
    c.refuse("withdrawConsent", "mom", t, {"guardianId": "mom", "memberId": "ada", "familyId": "laser"}, "not_guardian")
    c.ok("withdrawConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"}, g("lathe"))
    c.refuse("withdrawConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"}, "no_consent")
    # not_guardian comes before no_consent
    c.refuse("withdrawConsent", "dad", t, {"guardianId": "dad", "memberId": "kid", "familyId": "laser"}, "not_guardian")
    c.refuse("withdrawConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "nope"}, "not_found")
    c.ok("withdrawConsent", "amy", t, {"guardianId": "mom", "memberId": "kid", "familyId": "lathe"}, g())
    c.ok("getGuardianship", "kid", t, {"memberId": "kid"}, g())
    c.audit_log(t)
    return c.build()


def b_guardians_3() -> dict:
    c = Case("B-guardians-3", "B-guardians",
             "guardians: the 18th anniversary, 29 February, a former minor as guardian and as consenting guardian")
    t0 = "2026-02-27T09:00Z"
    world(c, t0)
    register(c, t0, "mom", "kid1", "kid2", "kid3", "kid4", "kid5", "kid6", "sib")
    t1, t2, t3 = "2026-02-28T10:00Z", "2026-02-28T23:59Z", "2026-03-01T00:00Z"
    # 29 February 2008: the 18th anniversary is 1 March 2026, so 28 February 2026 is still a minor's day
    c.ok("declareMinor", "sam", t1, {"memberId": "kid1", "birthDate": "2008-02-29", "guardianId": "mom"},
         {"guardianship": guardianship("kid1", "2008-02-29", "mom", [])})
    c.refuse("declareMinor", "sam", t1, {"memberId": "kid6", "birthDate": "2008-02-28", "guardianId": "mom"}, "not_a_minor")
    c.ok("declareMinor", "sam", t1, {"memberId": "sib", "birthDate": "2008-03-01", "guardianId": "mom"},
         {"guardianship": guardianship("sib", "2008-03-01", "mom", [])})
    c.refuse("declareMinor", "sam", t1, {"memberId": "kid5", "birthDate": "2012-01-01", "guardianId": "sib"}, "invalid_guardian")
    c.ok("declareMinor", "sam", t2, {"memberId": "kid2", "birthDate": "2008-02-29", "guardianId": "mom"},
         {"guardianship": guardianship("kid2", "2008-02-29", "mom", [])})
    c.refuse("declareMinor", "sam", t3, {"memberId": "kid3", "birthDate": "2008-02-29", "guardianId": "mom"}, "not_a_minor")
    # sib turned 18 at the start of 1 March: a former minor is a valid guardian
    c.ok("declareMinor", "sam", t3, {"memberId": "kid4", "birthDate": "2012-01-01", "guardianId": "sib"},
         {"guardianship": guardianship("kid4", "2012-01-01", "sib", [])})
    # consent can still be granted for a member who is not a minor any more
    c.ok("grantConsent", "mom", t3, {"guardianId": "mom", "memberId": "kid2", "familyId": "laser"},
         {"guardianship": guardianship("kid2", "2008-02-29", "mom", ["laser"])})
    c.ok("grantConsent", "sib", t3, {"guardianId": "sib", "memberId": "kid4", "familyId": "lathe"},
         {"guardianship": guardianship("kid4", "2012-01-01", "sib", ["lathe"])})
    c.audit_log(t3)
    return c.build()


# ---------------------------------------------------------------- waivers


def b_waivers_1() -> dict:
    c = Case("B-waivers-1", "B-waivers", "waivers: publish, get, sign, list; versions, refusals and their order")
    t = world(c)
    t2, t3, t4 = "2026-03-03T10:00Z", "2026-03-03T11:00Z", "2026-03-03T12:00Z"
    c.refuse("getWaiver", "ada", t, {"familyId": "laser"}, "no_waiver")
    c.refuse("signWaiver", "ada", t, {"memberId": "ada", "familyId": "laser"}, "no_waiver")
    c.refuse("publishWaiver", "sam", t, {"familyId": "laser", "text": "x"}, "forbidden")
    c.refuse("publishWaiver", "ada", t, {"familyId": "laser", "text": "x"}, "forbidden")
    c.refuse("publishWaiver", "amy", t, {"familyId": "nope", "text": "x"}, "not_found")
    c.refuse("publishWaiver", "amy", t, {"familyId": "laser", "text": "  "}, "invalid_argument")
    c.ok("publishWaiver", "amy", t, {"familyId": "laser", "text": "Laser waiver v1"}, {"waiver": waiver("laser", 1, "Laser waiver v1")})
    c.ok("getWaiver", "ada", t, {"familyId": "laser"}, {"waiver": waiver("laser", 1, "Laser waiver v1")})
    c.ok("signWaiver", "ada", t, {"memberId": "ada", "familyId": "laser"}, {"signature": signature("ada", "laser", 1, t)})
    c.refuse("signWaiver", "ada", t, {"memberId": "ada", "familyId": "laser"}, "already_signed")
    c.refuse("signWaiver", "bob", t, {"memberId": "ada", "familyId": "laser"}, "forbidden")
    c.refuse("signWaiver", "sam", t, {"memberId": "sam", "familyId": "laser"}, "not_found")
    c.ok("publishWaiver", "amy", t, {"familyId": "lathe", "text": "Lathe waiver v1"}, {"waiver": waiver("lathe", 1, "Lathe waiver v1")})
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "suspended"})
    c.ok("signWaiver", "bob", t, {"memberId": "bob", "familyId": "lathe"}, {"signature": signature("bob", "lathe", 1, t)})
    c.ok("listMemberWaivers", "ada", t, {"memberId": "ada"}, {"memberId": "ada", "signatures": [signature("ada", "laser", 1, t)]})
    c.ok("listMemberWaivers", "sam", t, {"memberId": "bob"}, {"memberId": "bob", "signatures": [signature("bob", "lathe", 1, t)]})
    c.ok("listMemberWaivers", "amy", t, {"memberId": "eve"}, {"memberId": "eve", "signatures": []})
    c.refuse("listMemberWaivers", "bob", t, {"memberId": "ada"}, "forbidden")
    # a new version replaces the waiver; the old signature stays listed but is worth nothing
    c.ok("publishWaiver", "amy", t2, {"familyId": "laser", "text": "Laser waiver v2"}, {"waiver": waiver("laser", 2, "Laser waiver v2")})
    c.ok("getWaiver", "bob", t2, {"familyId": "laser"}, {"waiver": waiver("laser", 2, "Laser waiver v2")})
    c.ok("listMemberWaivers", "ada", t2, {"memberId": "ada"}, {"memberId": "ada", "signatures": [signature("ada", "laser", 1, t)]})
    c.ok("signWaiver", "ada", t2, {"memberId": "ada", "familyId": "laser"}, {"signature": signature("ada", "laser", 2, t2)})
    c.refuse("signWaiver", "ada", t2, {"memberId": "ada", "familyId": "laser"}, "already_signed")
    c.ok("signWaiver", "sam", t3, {"memberId": "ada", "familyId": "lathe"}, {"signature": signature("ada", "lathe", 1, t3)})
    c.ok("listMemberWaivers", "ada", t3, {"memberId": "ada"},
         {"memberId": "ada", "signatures": [signature("ada", "laser", 2, t2), signature("ada", "lathe", 1, t3)]})
    c.ok("publishWaiver", "amy", t4, {"familyId": "lathe", "text": "Lathe waiver v2"}, {"waiver": waiver("lathe", 2, "Lathe waiver v2")})
    c.ok("listMemberWaivers", "ada", t4, {"memberId": "ada"},
         {"memberId": "ada", "signatures": [signature("ada", "laser", 2, t2), signature("ada", "lathe", 1, t3)]})
    c.ok("signWaiver", "ada", t4, {"memberId": "ada", "familyId": "lathe"}, {"signature": signature("ada", "lathe", 2, t4)})
    c.ok("listMemberWaivers", "ada", t4, {"memberId": "ada"},
         {"memberId": "ada", "signatures": [signature("ada", "laser", 2, t2), signature("ada", "lathe", 2, t4)]})
    c.audit_log(t4)
    return c.build()


def b_waivers_2() -> dict:
    c = Case("B-waivers-2", "B-waivers", "waivers: signing by a minor needs a guardian's consent; order no_waiver, already_signed, guardian_required")
    t = world(c)
    t2 = "2026-03-03T10:00Z"
    register(c, t, "mom", "kid")
    c.ok("declareMinor", "sam", t, {"memberId": "kid", "birthDate": "2012-06-15", "guardianId": "mom"})
    c.refuse("signWaiver", "kid", t, {"memberId": "kid", "familyId": "lathe"}, "no_waiver")  # before guardian_required
    c.ok("publishWaiver", "amy", t, {"familyId": "laser", "text": "Laser waiver v1"})
    c.refuse("signWaiver", "kid", t, {"memberId": "kid", "familyId": "laser"}, "guardian_required")
    c.refuse("signWaiver", "sam", t, {"memberId": "kid", "familyId": "laser"}, "guardian_required")
    c.ok("signWaiver", "ada", t, {"memberId": "ada", "familyId": "laser"}, {"signature": signature("ada", "laser", 1, t)})
    c.ok("grantConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "lathe"})
    c.refuse("signWaiver", "kid", t, {"memberId": "kid", "familyId": "laser"}, "guardian_required")  # consent for another family
    c.ok("grantConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"})
    c.ok("signWaiver", "kid", t, {"memberId": "kid", "familyId": "laser"}, {"signature": signature("kid", "laser", 1, t)})
    c.ok("withdrawConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"})
    c.refuse("signWaiver", "kid", t, {"memberId": "kid", "familyId": "laser"}, "already_signed")  # before guardian_required
    c.ok("publishWaiver", "amy", t2, {"familyId": "laser", "text": "Laser waiver v2"}, {"waiver": waiver("laser", 2, "Laser waiver v2")})
    c.refuse("signWaiver", "kid", t2, {"memberId": "kid", "familyId": "laser"}, "guardian_required")
    c.ok("grantConsent", "mom", t2, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"})
    c.ok("signWaiver", "kid", t2, {"memberId": "kid", "familyId": "laser"}, {"signature": signature("kid", "laser", 2, t2)})
    c.ok("listMemberWaivers", "kid", t2, {"memberId": "kid"}, {"memberId": "kid", "signatures": [signature("kid", "laser", 2, t2)]})
    c.audit_log(t2)
    return c.build()


# ---------------------------------------------------------------- ppe


def b_ppe_1() -> dict:
    c = Case("B-ppe-1", "B-ppe", "ppe: add item, require, issue, return, list; refusals, stock and order")
    t = world(c)
    c.ok("addPpeItem", "sam", t, {"itemId": "gloves", "name": "Gloves", "stock": 1}, {"item": ppe("gloves", "Gloves", 1)})
    c.ok("addPpeItem", "amy", t, {"itemId": "Mask", "name": "Mask", "stock": 5}, {"item": ppe("Mask", "Mask", 5)})
    c.ok("addPpeItem", "sam", t, {"itemId": "goggles", "name": "Goggles", "stock": 0}, {"item": ppe("goggles", "Goggles", 0)})
    c.ok("addPpeItem", "sam", t, {"itemId": "bulk", "name": "Bulk", "stock": 100000}, {"item": ppe("bulk", "Bulk", 100000)})
    c.refuse("addPpeItem", "sam", t, {"itemId": "gloves", "name": "Other", "stock": 9}, "already_exists")
    c.refuse("addPpeItem", "sam", t, {"itemId": "big", "name": "Big", "stock": 100001}, "invalid_argument")
    c.refuse("addPpeItem", "sam", t, {"itemId": "neg", "name": "Neg", "stock": -1}, "invalid_argument")
    c.refuse("addPpeItem", "ada", t, {"itemId": "own", "name": "Own", "stock": 1}, "forbidden")
    c.ok("requirePpe", "amy", t, {"familyId": "laser", "itemId": "gloves"}, {"requirement": {"familyId": "laser", "itemId": "gloves"}})
    c.ok("requirePpe", "amy", t, {"familyId": "laser", "itemId": "gloves"}, {"requirement": {"familyId": "laser", "itemId": "gloves"}})
    c.refuse("requirePpe", "sam", t, {"familyId": "laser", "itemId": "Mask"}, "forbidden")
    c.refuse("requirePpe", "amy", t, {"familyId": "nope", "itemId": "Mask"}, "not_found")
    c.refuse("requirePpe", "amy", t, {"familyId": "laser", "itemId": "nope"}, "not_found")
    c.ok("issuePpe", "sam", t, {"memberId": "ada", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 0)})
    c.refuse("issuePpe", "sam", t, {"memberId": "ada", "itemId": "gloves"}, "already_issued")  # before out_of_stock
    c.refuse("issuePpe", "sam", t, {"memberId": "bob", "itemId": "gloves"}, "out_of_stock")
    c.refuse("issuePpe", "sam", t, {"memberId": "bob", "itemId": "goggles"}, "out_of_stock")
    c.refuse("issuePpe", "ada", t, {"memberId": "ada", "itemId": "Mask"}, "forbidden")
    c.refuse("issuePpe", "sam", t, {"memberId": "sam", "itemId": "Mask"}, "not_found")
    c.refuse("issuePpe", "sam", t, {"memberId": "bob", "itemId": "nope"}, "not_found")
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "suspended"})
    c.refuse("issuePpe", "sam", t, {"memberId": "bob", "itemId": "gloves"}, "member_suspended")  # before out_of_stock
    c.refuse("issuePpe", "sam", t, {"memberId": "bob", "itemId": "Mask"}, "member_suspended")
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "good"})
    c.ok("issuePpe", "sam", t, {"memberId": "bob", "itemId": "Mask"}, {"item": ppe("Mask", "Mask", 4)})
    c.ok("issuePpe", "amy", t, {"memberId": "ada", "itemId": "Mask"}, {"item": ppe("Mask", "Mask", 3)})
    # identifiers are ordered by ASCII: an uppercase letter comes before a lowercase one
    c.ok("listMemberPpe", "ada", t, {"memberId": "ada"},
         {"memberId": "ada", "items": [ppe("Mask", "Mask", 3), ppe("gloves", "Gloves", 0)]})
    c.ok("listMemberPpe", "bob", t, {"memberId": "bob"}, {"memberId": "bob", "items": [ppe("Mask", "Mask", 3)]})
    c.refuse("listMemberPpe", "bob", t, {"memberId": "ada"}, "forbidden")
    c.ok("listMemberPpe", "sam", t, {"memberId": "eve"}, {"memberId": "eve", "items": []})
    c.refuse("returnPpe", "ada", t, {"memberId": "ada", "itemId": "gloves"}, "forbidden")
    c.ok("returnPpe", "sam", t, {"memberId": "ada", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 1)})
    c.refuse("returnPpe", "sam", t, {"memberId": "ada", "itemId": "gloves"}, "not_issued")
    c.ok("setMemberStanding", "sam", t, {"memberId": "ada", "standing": "suspended"})
    c.ok("listMemberPpe", "ada", t, {"memberId": "ada"}, {"memberId": "ada", "items": [ppe("Mask", "Mask", 3)]})  # holds on
    c.ok("returnPpe", "sam", t, {"memberId": "ada", "itemId": "Mask"}, {"item": ppe("Mask", "Mask", 4)})  # standing is not checked
    c.ok("listMemberPpe", "ada", t, {"memberId": "ada"}, {"memberId": "ada", "items": []})
    c.ok("issuePpe", "sam", t, {"memberId": "bob", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 0)})
    c.audit_log(t)
    return c.build()


def b_ppe_2() -> dict:
    c = Case("B-ppe-2", "B-ppe", "ppe: the gate at check-in needs every required item; a family without a requirement and reservations are not gated")
    t = world(c)
    t2, t3, t4 = "2026-03-03T10:00Z", "2026-03-03T11:00Z", "2026-03-03T11:10Z"
    c.ok("addCourse", "sam", t, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe basics", "feeCents": 0, "capacity": 5})
    c.ok("enrollInCourse", "sam", t, {"courseId": "lathe-101", "memberId": "ada"})
    c.ok("completeCourse", "sam", t, {"courseId": "lathe-101", "memberId": "ada"})
    c.ok("addPpeItem", "sam", t, {"itemId": "gloves", "name": "Gloves", "stock": 5})
    c.ok("addPpeItem", "sam", t, {"itemId": "goggles", "name": "Goggles", "stock": 5})
    c.ok("requirePpe", "amy", t, {"familyId": "laser", "itemId": "gloves"})
    c.ok("requirePpe", "amy", t, {"familyId": "laser", "itemId": "goggles"})
    # reservations are not gated by PPE
    c.ok("createReservation", "ada", t, {"memberId": "ada", "machineId": "L1", "start": "2026-03-03T10:00Z", "end": "2026-03-03T10:30Z"},
         {"reservation": resv("res-1", "ada", "L1", "2026-03-03T10:00Z", "2026-03-03T10:30Z")}, new="res-1")
    c.ok("createReservation", "ada", t, {"memberId": "ada", "machineId": "T1", "start": "2026-03-03T11:00Z", "end": "2026-03-03T11:30Z"},
         {"reservation": resv("res-2", "ada", "T1", "2026-03-03T11:00Z", "2026-03-03T11:30Z")}, new="res-2")
    c.refuse("checkIn", "ada", t2, {"reservationId": "res-1"}, "ppe_missing")
    c.ok("issuePpe", "sam", t2, {"memberId": "ada", "itemId": "goggles"}, {"item": ppe("goggles", "Goggles", 4)})
    c.refuse("checkIn", "ada", t2, {"reservationId": "res-1"}, "ppe_missing")  # every required item, not one
    c.ok("issuePpe", "sam", t2, {"memberId": "ada", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 4)})
    c.ok("checkIn", "ada", t2, {"reservationId": "res-1"},
         {"reservation": resv("res-1", "ada", "L1", "2026-03-03T10:00Z", "2026-03-03T10:30Z", "checkedIn", cin=t2, rate=45)})
    # the lathe family requires nothing; standing is judged before PPE
    c.ok("setMemberStanding", "sam", t3, {"memberId": "ada", "standing": "suspended"})
    c.ok("listMemberPpe", "ada", t3, {"memberId": "ada"}, {"memberId": "ada", "items": [ppe("gloves", "Gloves", 4), ppe("goggles", "Goggles", 4)]})
    c.refuse("checkIn", "ada", t3, {"reservationId": "res-2"}, "member_suspended")
    c.ok("setMemberStanding", "sam", t3, {"memberId": "ada", "standing": "good"})
    c.ok("checkIn", "ada", t3, {"reservationId": "res-2"},
         {"reservation": resv("res-2", "ada", "T1", "2026-03-03T11:00Z", "2026-03-03T11:30Z", "checkedIn", cin=t3, rate=30)})
    c.ok("returnPpe", "sam", t3, {"memberId": "ada", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 5)})
    c.ok("createReservation", "ada", t3, {"memberId": "ada", "machineId": "L1", "start": "2026-03-03T12:00Z", "end": "2026-03-03T12:30Z"},
         {"reservation": resv("res-3", "ada", "L1", "2026-03-03T12:00Z", "2026-03-03T12:30Z")}, new="res-3")
    # check-out is never gated: 70 metered minutes at 45 cents with 10 % off is 2835
    c.ok("checkOut", "ada", t4, {"reservationId": "res-1"},
         {"reservation": resv("res-1", "ada", "L1", "2026-03-03T10:00Z", "2026-03-03T10:30Z", "completed", cin=t2, cout=t4,
                              rate=45, metered=70, amount=2835),
          "charge": line("usage", "res-1", 70, 45, 10, t4)})
    c.audit_log(t4)
    return c.build()


# ---------------------------------------------------------------- staffing


def b_staffing_1() -> dict:
    c = Case("B-staffing-1", "B-staffing", "staffing: add shift, its refusals in order, length 720, touching and cancelled shifts")
    t = world(c)
    d3 = "2026-03-03T"
    c.refuse("addShift", "sam", t, {"userId": "ada", "start": d3 + "09:00Z", "end": d3 + "09:00Z"}, "not_staff")  # before invalid_interval
    c.refuse("addShift", "sam", t, {"userId": "sam", "start": d3 + "09:00Z", "end": d3 + "09:00Z"}, "invalid_interval")
    c.refuse("addShift", "sam", t, {"userId": "sam", "start": "2026-03-02T08:00Z", "end": "2026-03-02T08:00Z"}, "invalid_interval")
    c.refuse("addShift", "sam", t, {"userId": "sam", "start": "2026-03-02T08:59Z", "end": "2026-03-02T10:00Z"}, "starts_in_past")
    c.refuse("addShift", "sam", t, {"userId": "sam", "start": "2026-03-02T08:00Z", "end": "2026-03-02T21:00Z"}, "starts_in_past")  # also too long
    c.refuse("addShift", "sam", t, {"userId": "sam", "start": d3 + "08:00Z", "end": d3 + "20:01Z"}, "shift_too_long")  # 721 minutes
    c.ok("addShift", "sam", t, {"userId": "sam", "start": d3 + "08:00Z", "end": d3 + "20:00Z"},
         {"shift": shift("shf-1", "sam", d3 + "08:00Z", d3 + "20:00Z")}, new="shf-1")  # exactly 720 minutes
    c.refuse("addShift", "sam", t, {"userId": "sam", "start": d3 + "19:00Z", "end": d3 + "22:00Z"}, "shift_overlap")
    c.refuse("addShift", "sam", t, {"userId": "sam", "start": d3 + "10:00Z", "end": d3 + "22:01Z"}, "shift_too_long")  # before overlap
    c.ok("addShift", "sam", t, {"userId": "sam", "start": d3 + "20:00Z", "end": d3 + "22:00Z"},
         {"shift": shift("shf-2", "sam", d3 + "20:00Z", d3 + "22:00Z")}, new="shf-2")  # touching is no overlap
    c.ok("addShift", "amy", t, {"userId": "sue", "start": d3 + "09:00Z", "end": d3 + "11:00Z"},
         {"shift": shift("shf-3", "sue", d3 + "09:00Z", d3 + "11:00Z")}, new="shf-3")  # another user may overlap
    c.ok("addShift", "sam", t, {"userId": "amy", "start": d3 + "09:00Z", "end": d3 + "10:00Z"},
         {"shift": shift("shf-4", "amy", d3 + "09:00Z", d3 + "10:00Z")}, new="shf-4")  # an admin may work a shift
    c.ok("addShift", "sam", t, {"userId": "sue", "start": W, "end": "2026-03-02T10:00Z"},
         {"shift": shift("shf-5", "sue", W, "2026-03-02T10:00Z")}, new="shf-5")  # start at the request time
    c.refuse("addShift", "ada", t, {"userId": "sam", "start": d3 + "09:00Z", "end": d3 + "10:00Z"}, "forbidden")
    c.refuse("addShift", "sam", t, {"userId": "ghost", "start": d3 + "09:00Z", "end": d3 + "10:00Z"}, "not_found")
    c.refuse("addShift", "sam", t, {"userId": "sam", "start": "2026-03-03 09:00", "end": d3 + "10:00Z"}, "invalid_argument")
    c.ok("listShifts", "sam", t, {"date": "2026-03-03"},
         {"date": "2026-03-03", "shifts": [shift("shf-1", "sam", d3 + "08:00Z", d3 + "20:00Z"),
                                           shift("shf-3", "sue", d3 + "09:00Z", d3 + "11:00Z"),
                                           shift("shf-4", "amy", d3 + "09:00Z", d3 + "10:00Z"),
                                           shift("shf-2", "sam", d3 + "20:00Z", d3 + "22:00Z")]})
    c.audit_log(t)
    return c.build()


def b_staffing_2() -> dict:
    c = Case("B-staffing-2", "B-staffing",
             "staffing: list shifts over midnight, cancel shift, set supervision; supervision is not judged at check-in")
    t = world(c)
    d3, d4 = "2026-03-03T", "2026-03-04T"
    c.ok("addShift", "sam", t, {"userId": "sam", "start": d3 + "22:00Z", "end": d4 + "04:00Z"}, new="shf-1")
    c.ok("addShift", "sam", t, {"userId": "sue", "start": d4 + "00:00Z", "end": d4 + "06:00Z"}, new="shf-2")
    c.ok("addShift", "sam", t, {"userId": "sam", "start": d3 + "12:00Z", "end": d3 + "22:00Z"}, new="shf-3")
    c.ok("addShift", "sam", t, {"userId": "sue", "start": d3 + "14:00Z", "end": d4 + "00:00Z"}, new="shf-4")
    c.ok("addShift", "sam", t, {"userId": "amy", "start": d3 + "12:00Z", "end": d3 + "13:00Z"}, new="shf-5")
    s1, s2 = shift("shf-1", "sam", d3 + "22:00Z", d4 + "04:00Z"), shift("shf-2", "sue", d4 + "00:00Z", d4 + "06:00Z")
    s3, s4 = shift("shf-3", "sam", d3 + "12:00Z", d3 + "22:00Z"), shift("shf-4", "sue", d3 + "14:00Z", d4 + "00:00Z")
    s5 = shift("shf-5", "amy", d3 + "12:00Z", d3 + "13:00Z")
    # a shift that ends at 00:00 does not reach the next date; equal starts keep the creation order
    c.ok("listShifts", "amy", t, {"date": "2026-03-03"}, {"date": "2026-03-03", "shifts": [s3, s5, s4, s1]})
    c.ok("listShifts", "sam", t, {"date": "2026-03-04"}, {"date": "2026-03-04", "shifts": [s1, s2]})
    c.ok("listShifts", "sam", t, {"date": "2026-03-05"}, {"date": "2026-03-05", "shifts": []})
    c.ok("listShifts", "sam", t, {"date": "2026-03-02"}, {"date": "2026-03-02", "shifts": []})
    c.refuse("listShifts", "ada", t, {"date": "2026-03-03"}, "forbidden")
    c.ok("cancelShift", "sam", t, {"shiftId": "shf-5"}, {"shift": dict(s5, status="cancelled")})
    c.ok("listShifts", "sam", t, {"date": "2026-03-03"}, {"date": "2026-03-03", "shifts": [s3, s4, s1]})
    c.refuse("cancelShift", "sam", t, {"shiftId": "shf-5"}, "invalid_state")
    c.refuse("cancelShift", "sam", t, {"shiftId": "shf-9"}, "not_found")
    c.refuse("cancelShift", "ada", t, {"shiftId": "shf-3"}, "forbidden")
    # a shift that starts at the request time cannot be cancelled
    c.ok("addShift", "sam", t, {"userId": "sue", "start": W, "end": "2026-03-02T10:00Z"},
         {"shift": shift("shf-6", "sue", W, "2026-03-02T10:00Z")}, new="shf-6")
    c.refuse("cancelShift", "sam", t, {"shiftId": "shf-6"}, "invalid_state")
    # a cancelled shift no longer overlaps: sue's shf-4 (14:00 to 00:00) is gone, so 13:00 to 15:00 fits
    c.refuse("addShift", "sam", t, {"userId": "sue", "start": d3 + "13:00Z", "end": d3 + "15:00Z"}, "shift_overlap")
    c.ok("cancelShift", "amy", t, {"shiftId": "shf-4"}, {"shift": dict(s4, status="cancelled")})
    c.ok("addShift", "sam", t, {"userId": "sue", "start": d3 + "13:00Z", "end": d3 + "15:00Z"},
         {"shift": shift("shf-7", "sue", d3 + "13:00Z", d3 + "15:00Z")}, new="shf-7")
    # set supervision
    c.refuse("setSupervision", "sam", t, {"familyId": "laser", "required": True}, "forbidden")
    c.refuse("setSupervision", "amy", t, {"familyId": "nope", "required": True}, "not_found")
    c.refuse("setSupervision", "amy", t, {"familyId": "laser", "required": "true"}, "invalid_argument")
    c.ok("setSupervision", "amy", t, {"familyId": "laser", "required": True}, {"supervision": {"familyId": "laser", "required": True}})
    c.ok("setSupervision", "amy", t, {"familyId": "laser", "required": True}, {"supervision": {"familyId": "laser", "required": True}})
    c.ok("setSupervision", "amy", t, {"familyId": "laser", "required": False}, {"supervision": {"familyId": "laser", "required": False}})
    # a reservation made without supervision stays; turning supervision on does not touch it and check-in does not judge it
    r1 = ("res-1", "ada", "L1", d3 + "10:00Z", d3 + "10:30Z")
    c.ok("createReservation", "ada", t, {"memberId": "ada", "machineId": "L1", "start": r1[3], "end": r1[4]}, {"reservation": resv(*r1)}, new="res-1")
    c.ok("setSupervision", "amy", t, {"familyId": "laser", "required": True})
    c.refuse("createReservation", "ada", t, {"memberId": "ada", "machineId": "L2", "start": d3 + "11:00Z", "end": d3 + "11:30Z"}, "supervisor_required")
    c.ok("checkIn", "ada", d3 + "10:00Z", {"reservationId": "res-1"}, {"reservation": resv(*r1, status="checkedIn", cin=d3 + "10:00Z", rate=45)})
    c.ok("setSupervision", "amy", d3 + "10:00Z", {"familyId": "laser", "required": False})
    c.ok("createReservation", "ada", d3 + "10:00Z", {"memberId": "ada", "machineId": "L2", "start": d3 + "11:00Z", "end": d3 + "11:30Z"},
         {"reservation": resv("res-2", "ada", "L2", d3 + "11:00Z", d3 + "11:30Z")}, new="res-2")
    c.audit_log(d3 + "10:00Z")
    return c.build()


# ---------------------------------------------------------------- guests


def b_guests_1() -> dict:
    c = Case("B-guests-1", "B-guests", "guests: free and charged passes by the month of the date, revoke, refusals and their order")
    t = world(c)
    c.ok("setGuestAllowance", "amy", t, {"tier": "plus", "perMonth": 2, "feeCents": 500},
         {"allowance": {"tier": "plus", "perMonth": 2, "feeCents": 500}})
    c.refuse("issueGuestPass", "sam", t, {"memberId": "bob", "guestName": "Bea", "date": "2026-06-10"}, "guests_not_allowed")
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "suspended"})
    c.refuse("issueGuestPass", "sam", t, {"memberId": "bob", "guestName": "Bea", "date": "2026-03-01"}, "member_suspended")
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "good"})
    c.refuse("issueGuestPass", "sam", t, {"memberId": "bob", "guestName": "Bea", "date": "2026-03-01"}, "date_in_past")  # before the allowance check
    c.refuse("issueGuestPass", "bob", t, {"memberId": "ada", "guestName": "Bea", "date": "2026-06-10"}, "forbidden")
    c.refuse("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Bea", "date": "2026-02-30"}, "invalid_argument")
    c.refuse("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": " ", "date": "2026-06-10"}, "invalid_argument")
    # the example of the document: two free passes, a third charged 500
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gus", "date": "2026-06-10"},
         {"guestPass": gpass("gst-1", "ada", "Gus", "2026-06-10", False)}, new="gst-1")
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gwen", "date": "2026-06-11"},
         {"guestPass": gpass("gst-2", "ada", "Gwen", "2026-06-11", False)}, new="gst-2")
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", unbilled=5000))  # the course charge of the world
    c.ok("issueGuestPass", "sam", t, {"memberId": "ada", "guestName": "Gil", "date": "2026-06-12"},
         {"guestPass": gpass("gst-3", "ada", "Gil", "2026-06-12", True)}, new="gst-3")
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", unbilled=5500))
    c.ok("revokeGuestPass", "ada", t, {"guestPassId": "gst-1"}, {"guestPass": gpass("gst-1", "ada", "Gus", "2026-06-10", False, "revoked")})
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gaby", "date": "2026-06-20"},
         {"guestPass": gpass("gst-4", "ada", "Gaby", "2026-06-20", True)}, new="gst-4")
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gio", "date": "2026-07-01"},
         {"guestPass": gpass("gst-5", "ada", "Gio", "2026-07-01", False)}, new="gst-5")
    # revoking takes no charge back and keeps `charged`
    c.ok("revokeGuestPass", "amy", t, {"guestPassId": "gst-3"}, {"guestPass": gpass("gst-3", "ada", "Gil", "2026-06-12", True, "revoked")})
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", unbilled=6000))
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gus", "date": "2026-06-25"},
         {"guestPass": gpass("gst-6", "ada", "Gus", "2026-06-25", True)}, new="gst-6")  # gst-2 and gst-4 count
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gem", "date": "2026-03-02"},
         {"guestPass": gpass("gst-7", "ada", "Gem", "2026-03-02", False)}, new="gst-7")  # today is allowed
    c.refuse("revokeGuestPass", "ada", t, {"guestPassId": "gst-1"}, "invalid_state")
    c.refuse("revokeGuestPass", "bob", t, {"guestPassId": "gst-2"}, "forbidden")
    c.refuse("revokeGuestPass", "ada", t, {"guestPassId": "gst-9"}, "not_found")
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", unbilled=6500))
    t2 = "2026-03-03T09:00Z"
    c.refuse("revokeGuestPass", "ada", t2, {"guestPassId": "gst-7"}, "invalid_state")  # its date is before the date of the request
    c.ok("revokeGuestPass", "ada", t2, {"guestPassId": "gst-5"}, {"guestPass": gpass("gst-5", "ada", "Gio", "2026-07-01", False, "revoked")})
    c.ok("listGuestPasses", "ada", t2, {"memberId": "ada"},
         {"memberId": "ada", "guestPasses": [
             gpass("gst-1", "ada", "Gus", "2026-06-10", False, "revoked"), gpass("gst-2", "ada", "Gwen", "2026-06-11", False),
             gpass("gst-3", "ada", "Gil", "2026-06-12", True, "revoked"), gpass("gst-4", "ada", "Gaby", "2026-06-20", True),
             gpass("gst-5", "ada", "Gio", "2026-07-01", False, "revoked"), gpass("gst-6", "ada", "Gus", "2026-06-25", True),
             gpass("gst-7", "ada", "Gem", "2026-03-02", False)]})
    c.refuse("listGuestPasses", "bob", t2, {"memberId": "ada"}, "forbidden")
    c.ok("listGuestPasses", "sam", t2, {"memberId": "bob"}, {"memberId": "bob", "guestPasses": []})
    c.audit_log(t2)
    return c.build()


def b_guests_2() -> dict:
    c = Case("B-guests-2", "B-guests", "guests: allowance of 0 per month, the tier at the request time, replacing an allowance, ranges, invoice lines")
    t = world(c)
    c.ok("setGuestAllowance", "amy", t, {"tier": "basic", "perMonth": 0, "feeCents": 0}, {"allowance": {"tier": "basic", "perMonth": 0, "feeCents": 0}})
    c.ok("setGuestAllowance", "amy", t, {"tier": "plus", "perMonth": 1, "feeCents": 700}, {"allowance": {"tier": "plus", "perMonth": 1, "feeCents": 700}})
    c.refuse("setGuestAllowance", "sam", t, {"tier": "pro", "perMonth": 1, "feeCents": 1}, "forbidden")
    c.refuse("setGuestAllowance", "ada", t, {"tier": "pro", "perMonth": 1, "feeCents": 1}, "forbidden")
    c.refuse("setGuestAllowance", "amy", t, {"tier": "gold", "perMonth": 1, "feeCents": 1}, "invalid_argument")
    c.refuse("setGuestAllowance", "amy", t, {"tier": "pro", "perMonth": 101, "feeCents": 1}, "invalid_argument")
    c.refuse("setGuestAllowance", "amy", t, {"tier": "pro", "perMonth": -1, "feeCents": 1}, "invalid_argument")
    c.refuse("setGuestAllowance", "amy", t, {"tier": "pro", "perMonth": 1, "feeCents": 1000001}, "invalid_argument")
    # perMonth 0: every pass is charged, here with a fee of 0 cents, and the charge of 0 cents is still created
    c.ok("issueGuestPass", "bob", t, {"memberId": "bob", "guestName": "Bea", "date": "2026-03-02"},
         {"guestPass": gpass("gst-1", "bob", "Bea", "2026-03-02", True)}, new="gst-1")
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gus", "date": "2026-06-10"},
         {"guestPass": gpass("gst-2", "ada", "Gus", "2026-06-10", False)}, new="gst-2")
    c.refuse("issueGuestPass", "eve", t, {"memberId": "eve", "guestName": "Eli", "date": "2026-06-10"}, "guests_not_allowed")  # pro has none
    c.ok("changeTier", "sam", t, {"memberId": "eve", "tier": "plus"})
    c.ok("issueGuestPass", "eve", t, {"memberId": "eve", "guestName": "Eli", "date": "2026-06-10"},
         {"guestPass": gpass("gst-3", "eve", "Eli", "2026-06-10", False)}, new="gst-3")
    # the tier of the request decides, the passes already issued count by member: basic has 0 per month
    c.ok("changeTier", "sam", t, {"memberId": "ada", "tier": "basic"})
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gil", "date": "2026-06-11"},
         {"guestPass": gpass("gst-4", "ada", "Gil", "2026-06-11", True)}, new="gst-4")
    c.ok("setGuestAllowance", "amy", t, {"tier": "plus", "perMonth": 5, "feeCents": 100}, {"allowance": {"tier": "plus", "perMonth": 5, "feeCents": 100}})
    c.ok("setGuestAllowance", "amy", t, {"tier": "pro", "perMonth": 100, "feeCents": 1000000}, {"allowance": {"tier": "pro", "perMonth": 100, "feeCents": 1000000}})
    c.ok("listGuestPasses", "ada", t, {"memberId": "ada"},
         {"memberId": "ada", "guestPasses": [gpass("gst-2", "ada", "Gus", "2026-06-10", False), gpass("gst-4", "ada", "Gil", "2026-06-11", True)]})
    march = "2026-03"
    zero = lambda gid: line("guest", gid, 1, 0, 0, W)  # noqa: E731
    c.ok("generateInvoices", "sam", INVOICE_AT, {"month": march},
         {"month": march, "invoices": [
             plain("inv-1", "ada", "basic", march, INVOICE_AT, INVOICE_DUE, extra=(COURSE, zero("gst-4"))),
             plain("inv-2", "bob", "basic", march, INVOICE_AT, INVOICE_DUE, extra=(zero("gst-1"),)),
             plain("inv-3", "eve", "plus", march, INVOICE_AT, INVOICE_DUE)]})
    c.audit_log(INVOICE_AT)
    return c.build()


# ---------------------------------------------------------------- volunteering


def b_volunteering_1() -> dict:
    c = Case("B-volunteering-1", "B-volunteering", "volunteering: rate, credit rounding, decisions, ledger postings, refusals and order")
    t = world(c)
    c.refuse("setVolunteerRate", "sam", t, {"centsPerHour": 1500}, "forbidden")
    c.refuse("setVolunteerRate", "amy", t, {"centsPerHour": 100001}, "invalid_argument")
    c.ok("setVolunteerRate", "amy", t, {"centsPerHour": 1500}, {"centsPerHour": 1500})
    c.ok("setVolunteerRate", "amy", t, {"centsPerHour": 1500}, {"centsPerHour": 1500})
    c.ok("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 100, "note": "Bench build"},
         {"entry": vol("vol-1", "ada", "2026-03-02", 100, "Bench build")}, new="vol-1")
    c.refuse("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-03", "minutes": 10, "note": "Tomorrow"}, "date_in_future")
    c.refuse("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 0, "note": "None"}, "invalid_argument")
    c.refuse("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 721, "note": "Long"}, "invalid_argument")
    c.refuse("logVolunteerTime", "bob", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 10, "note": "Other"}, "forbidden")
    c.refuse("decideVolunteerTime", "ada", t, {"entryId": "vol-1", "decision": "approved"}, "forbidden")
    c.refuse("decideVolunteerTime", "sam", t, {"entryId": "vol-1", "decision": "approved"}, "approver_not_on_shift")
    c.refuse("decideVolunteerTime", "amy", t, {"entryId": "vol-9", "decision": "approved"}, "not_found")
    c.refuse("decideVolunteerTime", "amy", t, {"entryId": "vol-1", "decision": "pending"}, "invalid_argument")
    # 100 minutes at 1500 cents per hour: (100 * 1500 + 30) div 60 = 2500
    c.ok("decideVolunteerTime", "amy", t, {"entryId": "vol-1", "decision": "approved"},
         {"entry": vol("vol-1", "ada", "2026-03-02", 100, "Bench build", "approved", 2500)})
    c.refuse("decideVolunteerTime", "amy", t, {"entryId": "vol-1", "decision": "approved"}, "invalid_state")
    c.refuse("decideVolunteerTime", "sam", t, {"entryId": "vol-1", "decision": "rejected"}, "invalid_state")  # before the shift check
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", credit=2500, unbilled=5000))
    # 30 minutes at 1 cent per hour: (30 * 1 + 30) div 60 = 1
    c.ok("setVolunteerRate", "amy", t, {"centsPerHour": 1}, {"centsPerHour": 1})
    c.ok("logVolunteerTime", "sam", t, {"memberId": "ada", "date": "2026-03-01", "minutes": 30, "note": "Sweeping"},
         {"entry": vol("vol-2", "ada", "2026-03-01", 30, "Sweeping")}, new="vol-2")
    c.ok("decideVolunteerTime", "amy", t, {"entryId": "vol-2", "decision": "approved"},
         {"entry": vol("vol-2", "ada", "2026-03-01", 30, "Sweeping", "approved", 1)})
    # rate 0: approved with credit 0, nothing posted
    c.ok("setVolunteerRate", "amy", t, {"centsPerHour": 0}, {"centsPerHour": 0})
    c.ok("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 45, "note": "Free help"},
         {"entry": vol("vol-3", "ada", "2026-03-02", 45, "Free help")}, new="vol-3")
    c.ok("decideVolunteerTime", "amy", t, {"entryId": "vol-3", "decision": "approved"},
         {"entry": vol("vol-3", "ada", "2026-03-02", 45, "Free help", "approved", 0)})
    # 7 minutes at 100 cents per hour: (7 * 100 + 30) div 60 = 12
    c.ok("setVolunteerRate", "amy", t, {"centsPerHour": 100}, {"centsPerHour": 100})
    c.ok("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 7, "note": "Labels"},
         {"entry": vol("vol-4", "ada", "2026-03-02", 7, "Labels")}, new="vol-4")
    c.ok("decideVolunteerTime", "amy", t, {"entryId": "vol-4", "decision": "approved"},
         {"entry": vol("vol-4", "ada", "2026-03-02", 7, "Labels", "approved", 12)})
    # the rate in force at the approval pays: 20 minutes at 3000 cents per hour is (20 * 3000 + 30) div 60 = 1000
    c.ok("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 20, "note": "Painting"},
         {"entry": vol("vol-5", "ada", "2026-03-02", 20, "Painting")}, new="vol-5")
    c.ok("setVolunteerRate", "amy", t, {"centsPerHour": 3000}, {"centsPerHour": 3000})
    c.ok("decideVolunteerTime", "amy", t, {"entryId": "vol-5", "decision": "approved"},
         {"entry": vol("vol-5", "ada", "2026-03-02", 20, "Painting", "approved", 1000)})
    c.ok("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 10, "note": "Unwanted"},
         {"entry": vol("vol-6", "ada", "2026-03-02", 10, "Unwanted")}, new="vol-6")
    c.ok("decideVolunteerTime", "amy", t, {"entryId": "vol-6", "decision": "rejected"},
         {"entry": vol("vol-6", "ada", "2026-03-02", 10, "Unwanted", "rejected")})
    c.refuse("decideVolunteerTime", "amy", t, {"entryId": "vol-6", "decision": "approved"}, "invalid_state")
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", credit=3513, unbilled=5000))  # 2500 + 1 + 0 + 12 + 1000
    c.ok("listVolunteerTime", "ada", t, {"memberId": "ada"},
         {"memberId": "ada", "entries": [
             vol("vol-1", "ada", "2026-03-02", 100, "Bench build", "approved", 2500), vol("vol-2", "ada", "2026-03-01", 30, "Sweeping", "approved", 1),
             vol("vol-3", "ada", "2026-03-02", 45, "Free help", "approved", 0), vol("vol-4", "ada", "2026-03-02", 7, "Labels", "approved", 12),
             vol("vol-5", "ada", "2026-03-02", 20, "Painting", "approved", 1000), vol("vol-6", "ada", "2026-03-02", 10, "Unwanted", "rejected")]})
    c.ok("listVolunteerTime", "bob", t, {"memberId": "bob"}, {"memberId": "bob", "entries": []})
    c.refuse("listVolunteerTime", "bob", t, {"memberId": "ada"}, "forbidden")
    c.ok("listJournalEntries", "amy", t, {"month": "2026-03"},
         {"month": "2026-03", "entries": [
             journal("jrn-1", t, "expenseVolunteer", "creditLiability", 2500, "vol-1"), journal("jrn-2", t, "expenseVolunteer", "creditLiability", 1, "vol-2"),
             journal("jrn-3", t, "expenseVolunteer", "creditLiability", 12, "vol-4"), journal("jrn-4", t, "expenseVolunteer", "creditLiability", 1000, "vol-5")]})
    c.audit_log(t)
    return c.build()


def b_volunteering_2() -> dict:
    c = Case("B-volunteering-2", "B-volunteering",
             "volunteering: the shift of a staff approver, the cap of 1200 approved minutes per month of the entry's date")
    t = world(c)
    t10, t1159, t12 = "2026-03-02T10:00Z", "2026-03-02T11:59Z", "2026-03-02T12:00Z"
    mar, feb = "2026-03-01", "2026-02-28"
    c.ok("setVolunteerRate", "amy", t, {"centsPerHour": 60}, {"centsPerHour": 60})  # one cent per minute
    c.ok("addShift", "sam", t, {"userId": "sue", "start": t10, "end": t12}, new="shf-1")
    logs = (("ada", mar, 720), ("ada", mar, 380), ("ada", mar, 100), ("ada", mar, 1), ("ada", feb, 720),
            ("ada", mar, 720), ("bob", mar, 720), ("bob", mar, 10))
    for number, (who, date, minutes) in enumerate(logs, 1):
        c.ok("logVolunteerTime", who, t, {"memberId": who, "date": date, "minutes": minutes, "note": f"n{minutes}"}, new=f"vol-{number}")
    c.refuse("decideVolunteerTime", "sue", t, {"entryId": "vol-1", "decision": "approved"}, "approver_not_on_shift")  # starts at 10:00
    c.ok("decideVolunteerTime", "sue", t10, {"entryId": "vol-1", "decision": "approved"}, {"entry": vol("vol-1", "ada", mar, 720, "n720", "approved", 720)})
    c.ok("decideVolunteerTime", "sue", t10, {"entryId": "vol-2", "decision": "approved"}, {"entry": vol("vol-2", "ada", mar, 380, "n380", "approved", 380)})
    c.ok("decideVolunteerTime", "sue", t10, {"entryId": "vol-3", "decision": "approved"}, {"entry": vol("vol-3", "ada", mar, 100, "n100", "approved", 100)})  # 1200 exactly
    c.refuse("decideVolunteerTime", "sue", t10, {"entryId": "vol-4", "decision": "approved"}, "volunteer_cap_reached")
    c.refuse("decideVolunteerTime", "sue", t10, {"entryId": "vol-6", "decision": "approved"}, "volunteer_cap_reached")
    c.refuse("decideVolunteerTime", "amy", t10, {"entryId": "vol-4", "decision": "approved"}, "volunteer_cap_reached")  # an admin has the cap too
    # the month of the entry's date counts, not the month of the request; other members have their own cap
    c.ok("decideVolunteerTime", "sue", t10, {"entryId": "vol-5", "decision": "approved"}, {"entry": vol("vol-5", "ada", feb, 720, "n720", "approved", 720)})
    c.ok("decideVolunteerTime", "sue", t10, {"entryId": "vol-7", "decision": "approved"}, {"entry": vol("vol-7", "bob", mar, 720, "n720", "approved", 720)})
    c.ok("decideVolunteerTime", "sue", t10, {"entryId": "vol-4", "decision": "rejected"}, {"entry": vol("vol-4", "ada", mar, 1, "n1", "rejected")})
    c.ok("decideVolunteerTime", "sue", t1159, {"entryId": "vol-8", "decision": "approved"}, {"entry": vol("vol-8", "bob", mar, 10, "n10", "approved", 10)})
    c.refuse("decideVolunteerTime", "sue", t12, {"entryId": "vol-3", "decision": "rejected"}, "invalid_state")  # before the shift check
    c.refuse("decideVolunteerTime", "sue", t12, {"entryId": "vol-6", "decision": "rejected"}, "approver_not_on_shift")  # the shift ends at 12:00
    c.ok("decideVolunteerTime", "amy", t12, {"entryId": "vol-6", "decision": "rejected"}, {"entry": vol("vol-6", "ada", mar, 720, "n720", "rejected")})
    c.ok("getAccount", "ada", t12, {"memberId": "ada"}, account("ada", credit=1920, unbilled=5000))
    c.ok("getAccount", "bob", t12, {"memberId": "bob"}, account("bob", credit=730))
    c.ok("listVolunteerTime", "ada", t12, {"memberId": "ada"},
         {"memberId": "ada", "entries": [
             vol("vol-1", "ada", mar, 720, "n720", "approved", 720), vol("vol-2", "ada", mar, 380, "n380", "approved", 380),
             vol("vol-3", "ada", mar, 100, "n100", "approved", 100), vol("vol-4", "ada", mar, 1, "n1", "rejected"),
             vol("vol-5", "ada", feb, 720, "n720", "approved", 720), vol("vol-6", "ada", mar, 720, "n720", "rejected")]})
    c.audit_log(t12)
    return c.build()


# ---------------------------------------------------------------- access


def b_access_1() -> dict:
    c = Case("B-access-1", "B-access", "access: badges, enter and leave door, opening hours, the order of enter door refusals, door events")
    t = world(c)
    c.ok("issueBadge", "sam", t, {"memberId": "ada"}, {"badge": badge("bdg-1", "ada")}, new="bdg-1")
    c.refuse("issueBadge", "sam", t, {"memberId": "ada"}, "badge_active")
    c.refuse("issueBadge", "ada", t, {"memberId": "bob"}, "forbidden")
    c.refuse("issueBadge", "sam", t, {"memberId": "sam"}, "not_found")
    c.refuse("leaveDoor", "ada", t, {"badgeId": "bdg-1"}, "not_inside")
    c.ok("enterDoor", "ada", t, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-1", "enter", "bdg-1", t)})  # 09:00 opens
    c.refuse("enterDoor", "ada", t, {"badgeId": "bdg-1"}, "already_inside")
    c.refuse("enterDoor", "bob", t, {"badgeId": "bdg-1"}, "forbidden")
    c.refuse("enterDoor", "ada", t, {"badgeId": "bdg-9"}, "not_found")
    c.ok("leaveDoor", "ada", t, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada"), "event": door("dor-2", "leave", "bdg-1", t)})
    c.ok("enterDoor", "sam", t, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-3", "enter", "bdg-1", t)})
    # revoking an inside badge: outside, revoked, no door event
    c.ok("revokeBadge", "sam", t, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", "revoked")})
    c.refuse("enterDoor", "ada", t, {"badgeId": "bdg-1"}, "invalid_state")
    c.refuse("leaveDoor", "ada", t, {"badgeId": "bdg-1"}, "invalid_state")
    c.refuse("revokeBadge", "sam", t, {"badgeId": "bdg-1"}, "invalid_state")
    c.refuse("revokeBadge", "ada", t, {"badgeId": "bdg-1"}, "forbidden")
    c.ok("issueBadge", "amy", t, {"memberId": "ada"}, {"badge": badge("bdg-2", "ada")}, new="bdg-2")
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "suspended"})
    c.ok("issueBadge", "sam", t, {"memberId": "bob"}, {"badge": badge("bdg-3", "bob")}, new="bdg-3")  # standing is not checked
    c.refuse("enterDoor", "bob", t, {"badgeId": "bdg-3"}, "member_suspended")
    # the last open minute and the closing time of a Monday
    c.ok("enterDoor", "ada", "2026-03-02T20:59Z", {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "ada", inside=True), "event": door("dor-4", "enter", "bdg-2", "2026-03-02T20:59Z")})
    c.ok("leaveDoor", "ada", "2026-03-02T20:59Z", {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "ada"), "event": door("dor-5", "leave", "bdg-2", "2026-03-02T20:59Z")})
    c.refuse("enterDoor", "ada", "2026-03-02T21:00Z", {"badgeId": "bdg-2"}, "outside_opening_hours")
    c.refuse("enterDoor", "bob", "2026-03-02T21:00Z", {"badgeId": "bdg-3"}, "member_suspended")  # before the opening hours
    c.refuse("enterDoor", "ada", "2026-03-03T08:59Z", {"badgeId": "bdg-2"}, "outside_opening_hours")
    c.ok("enterDoor", "ada", "2026-03-03T09:00Z", {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "ada", inside=True), "event": door("dor-6", "enter", "bdg-2", "2026-03-03T09:00Z")})
    c.ok("leaveDoor", "ada", "2026-03-03T09:00Z", {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "ada"), "event": door("dor-7", "leave", "bdg-2", "2026-03-03T09:00Z")})
    # Saturday 2026-03-07 is open from 10:00 to 18:00; leaving is never gated by the hours
    c.refuse("enterDoor", "ada", "2026-03-07T09:59Z", {"badgeId": "bdg-2"}, "outside_opening_hours")
    c.ok("enterDoor", "ada", "2026-03-07T10:00Z", {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "ada", inside=True), "event": door("dor-8", "enter", "bdg-2", "2026-03-07T10:00Z")})
    c.ok("leaveDoor", "ada", "2026-03-07T18:00Z", {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "ada"), "event": door("dor-9", "leave", "bdg-2", "2026-03-07T18:00Z")})
    c.refuse("enterDoor", "ada", "2026-03-07T18:00Z", {"badgeId": "bdg-2"}, "outside_opening_hours")
    c.refuse("enterDoor", "ada", "2026-03-08T12:00Z", {"badgeId": "bdg-2"}, "outside_opening_hours")  # Sunday
    c.ok("listDoorEvents", "sam", "2026-03-08T12:00Z", {"date": "2026-03-02"},
         {"date": "2026-03-02", "events": [door("dor-1", "enter", "bdg-1", t), door("dor-2", "leave", "bdg-1", t), door("dor-3", "enter", "bdg-1", t),
                                           door("dor-4", "enter", "bdg-2", "2026-03-02T20:59Z"), door("dor-5", "leave", "bdg-2", "2026-03-02T20:59Z")]})
    c.ok("listDoorEvents", "amy", "2026-03-08T12:00Z", {"date": "2026-03-03"},
         {"date": "2026-03-03", "events": [door("dor-6", "enter", "bdg-2", "2026-03-03T09:00Z"), door("dor-7", "leave", "bdg-2", "2026-03-03T09:00Z")]})
    c.ok("listDoorEvents", "sam", "2026-03-08T12:00Z", {"date": "2026-03-07"},
         {"date": "2026-03-07", "events": [door("dor-8", "enter", "bdg-2", "2026-03-07T10:00Z"), door("dor-9", "leave", "bdg-2", "2026-03-07T18:00Z")]})
    c.ok("listDoorEvents", "sam", "2026-03-08T12:00Z", {"date": "2026-03-08"}, {"date": "2026-03-08", "events": []})
    c.refuse("listDoorEvents", "ada", "2026-03-08T12:00Z", {"date": "2026-03-02"}, "forbidden")
    c.audit_log()
    return c.build()


def b_access_2() -> dict:
    c = Case("B-access-2", "B-access", "access: admit guest, the order of its refusals, ownership on the badge, what it does not check")
    t = world(c)
    late = "2026-03-02T21:30Z"
    c.ok("setGuestAllowance", "amy", t, {"tier": "plus", "perMonth": 5, "feeCents": 0})
    c.ok("setGuestAllowance", "amy", t, {"tier": "basic", "perMonth": 1, "feeCents": 0})
    passes = (("ada", "Gus", "2026-03-02"), ("ada", "Gwen", "2026-03-03"), ("ada", "Gil", "2026-03-02"),
              ("bob", "Bea", "2026-03-02"), ("ada", "Gem", "2026-03-02"), ("ada", "Gia", "2026-03-02"))
    for number, (who, name, date) in enumerate(passes, 1):
        c.ok("issueGuestPass", who, t, {"memberId": who, "guestName": name, "date": date}, new=f"gst-{number}")
    c.ok("revokeGuestPass", "ada", t, {"guestPassId": "gst-3"})
    c.ok("issueBadge", "sam", t, {"memberId": "ada"}, {"badge": badge("bdg-1", "ada")}, new="bdg-1")
    c.ok("issueBadge", "sam", t, {"memberId": "bob"}, {"badge": badge("bdg-2", "bob")}, new="bdg-2")
    # invalid_state, pass_not_today, pass_of_another_member, sponsor_not_inside, guest_already_admitted, in this order
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-3", "badgeId": "bdg-1"}, "invalid_state")  # a revoked pass
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-2", "badgeId": "bdg-1"}, "pass_not_today")  # before sponsor_not_inside
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-4", "badgeId": "bdg-1"}, "pass_of_another_member")  # before sponsor_not_inside
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-1", "badgeId": "bdg-1"}, "sponsor_not_inside")
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-9", "badgeId": "bdg-1"}, "not_found")
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-1", "badgeId": "bdg-9"}, "not_found")
    c.ok("enterDoor", "ada", t, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-1", "enter", "bdg-1", t)})
    c.ok("admitGuest", "ada", t, {"guestPassId": "gst-1", "badgeId": "bdg-1"}, {"event": door("dor-2", "guest", "bdg-1", t, "gst-1")})
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-1", "badgeId": "bdg-1"}, "guest_already_admitted")
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-3", "badgeId": "bdg-1"}, "invalid_state")
    c.refuse("admitGuest", "ada", t, {"guestPassId": "gst-4", "badgeId": "bdg-1"}, "pass_of_another_member")  # ada is inside now
    # ownership is judged on the badge alone: bob owns bdg-2, not bdg-1
    c.refuse("admitGuest", "bob", t, {"guestPassId": "gst-4", "badgeId": "bdg-1"}, "forbidden")
    c.refuse("admitGuest", "bob", t, {"guestPassId": "gst-4", "badgeId": "bdg-2"}, "sponsor_not_inside")
    c.ok("enterDoor", "bob", t, {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "bob", inside=True), "event": door("dor-3", "enter", "bdg-2", t)})
    c.ok("admitGuest", "sam", t, {"guestPassId": "gst-4", "badgeId": "bdg-2"}, {"event": door("dor-4", "guest", "bdg-2", t, "gst-4")})
    # standing is not checked
    c.ok("setMemberStanding", "sam", t, {"memberId": "ada", "standing": "suspended"})
    c.ok("admitGuest", "ada", t, {"guestPassId": "gst-5", "badgeId": "bdg-1"}, {"event": door("dor-5", "guest", "bdg-1", t, "gst-5")})
    # opening hours are not checked: 21:30 is after closing
    c.ok("admitGuest", "amy", late, {"guestPassId": "gst-6", "badgeId": "bdg-1"}, {"event": door("dor-6", "guest", "bdg-1", late, "gst-6")})
    # a revoked badge: revoking an inside badge sets inside false
    c.ok("revokeBadge", "sam", late, {"badgeId": "bdg-2"}, {"badge": badge("bdg-2", "bob", "revoked")})
    c.ok("issueGuestPass", "sam", late, {"memberId": "bob", "guestName": "Bo", "date": "2026-03-02"}, new="gst-7")
    c.refuse("admitGuest", "sam", late, {"guestPassId": "gst-7", "badgeId": "bdg-2"}, "invalid_state")
    c.ok("listDoorEvents", "sam", late, {"date": "2026-03-02"},
         {"date": "2026-03-02", "events": [door("dor-1", "enter", "bdg-1", t), door("dor-2", "guest", "bdg-1", t, "gst-1"), door("dor-3", "enter", "bdg-2", t),
                                           door("dor-4", "guest", "bdg-2", t, "gst-4"), door("dor-5", "guest", "bdg-1", t, "gst-5"),
                                           door("dor-6", "guest", "bdg-1", late, "gst-6")]})
    c.audit_log(late)
    return c.build()


# ---------------------------------------------------------------- chains across modules


def x_people_1() -> dict:
    c = Case("X-people-1", "X-people",
             "guardians, waivers, training, bookings: a minor through consent, waiver signature, enrollment, reservation and check-in")
    t = world(c)
    t2, t3 = "2026-03-03T09:00Z", "2026-03-03T10:00Z"
    start, end = "2026-03-03T10:00Z", "2026-03-03T11:00Z"
    reserve = {"memberId": "kid", "machineId": "L1", "start": start, "end": end}
    register(c, t, "mom", "kid")
    c.ok("declareMinor", "sam", t, {"memberId": "kid", "birthDate": "2012-06-15", "guardianId": "mom"},
         {"guardianship": guardianship("kid", "2012-06-15", "mom", [])})
    c.ok("publishWaiver", "amy", t, {"familyId": "laser", "text": "Laser waiver v1"}, {"waiver": waiver("laser", 1, "Laser waiver v1")})
    c.refuse("signWaiver", "kid", t, {"memberId": "kid", "familyId": "laser"}, "guardian_required")
    c.refuse("enrollInCourse", "kid", t, {"courseId": "laser-101", "memberId": "kid"}, "guardian_consent_missing")
    c.refuse("createReservation", "kid", t, reserve, "guardian_consent_missing")  # before not_certified and waiver_missing
    c.ok("grantConsent", "mom", t, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"},
         {"guardianship": guardianship("kid", "2012-06-15", "mom", ["laser"])})
    c.refuse("createReservation", "kid", t, reserve, "not_certified")
    course = {"courseId": "laser-101", "familyId": "laser", "title": "Laser safety", "feeCents": 5000, "capacity": 2, "enrolledCount": 1}
    c.ok("enrollInCourse", "kid", t, {"courseId": "laser-101", "memberId": "kid"},
         {"enrollment": {"courseId": "laser-101", "memberId": "kid", "status": "enrolled", "enrolledAt": t, "completedAt": None}, "course": course})
    c.refuse("enrollInCourse", "kid", t, {"courseId": "laser-101", "memberId": "kid"}, "already_enrolled")
    c.refuse("createReservation", "kid", t, reserve, "not_certified")  # enrolled is not certified
    c.ok("completeCourse", "sam", t, {"courseId": "laser-101", "memberId": "kid"},
         {"enrollment": {"courseId": "laser-101", "memberId": "kid", "status": "completed", "enrolledAt": t, "completedAt": t},
          "certification": {"memberId": "kid", "familyId": "laser", "issuedOn": "2026-03-02", "expiresOn": "2027-03-02", "status": "valid"}})
    c.refuse("createReservation", "kid", t, reserve, "waiver_missing")
    c.ok("signWaiver", "kid", t, {"memberId": "kid", "familyId": "laser"}, {"signature": signature("kid", "laser", 1, t)})
    c.ok("createReservation", "kid", t, reserve, {"reservation": resv("res-1", "kid", "L1", start, end)}, new="res-1")
    c.refuse("checkIn", "kid", t2, {"reservationId": "res-1"}, "check_in_too_early")
    c.ok("withdrawConsent", "mom", t2, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"},
         {"guardianship": guardianship("kid", "2012-06-15", "mom", [])})
    c.ok("setMemberStanding", "sam", t3, {"memberId": "kid", "standing": "suspended"})
    c.refuse("checkIn", "kid", t3, {"reservationId": "res-1"}, "member_suspended")  # before guardian_consent_missing
    c.ok("setMemberStanding", "sam", t3, {"memberId": "kid", "standing": "good"})
    c.refuse("checkIn", "kid", t3, {"reservationId": "res-1"}, "guardian_consent_missing")
    c.ok("grantConsent", "mom", t3, {"guardianId": "mom", "memberId": "kid", "familyId": "laser"},
         {"guardianship": guardianship("kid", "2012-06-15", "mom", ["laser"])})
    c.ok("checkIn", "kid", t3, {"reservationId": "res-1"}, {"reservation": resv("res-1", "kid", "L1", start, end, "checkedIn", cin=t3, rate=45)})
    c.ok("getAccount", "kid", t3, {"memberId": "kid"}, account("kid", unbilled=5000))
    c.audit_log(t3)
    return c.build()


def x_people_2() -> dict:
    c = Case("X-people-2", "X-people", "waivers, bookings, billing: a new waiver version blocks the check-in of an existing reservation until signed")
    t = world(c)
    t2, t3, t4 = "2026-03-02T15:00Z", "2026-03-03T10:00Z", "2026-03-03T10:37Z"
    s1, e1, s2, e2 = "2026-03-03T10:00Z", "2026-03-03T11:00Z", "2026-03-03T12:00Z", "2026-03-03T13:00Z"
    c.ok("publishWaiver", "amy", t, {"familyId": "laser", "text": "Laser waiver v1"}, {"waiver": waiver("laser", 1, "Laser waiver v1")})
    c.ok("signWaiver", "ada", t, {"memberId": "ada", "familyId": "laser"}, {"signature": signature("ada", "laser", 1, t)})
    c.ok("createReservation", "ada", t, {"memberId": "ada", "machineId": "L1", "start": s1, "end": e1},
         {"reservation": resv("res-1", "ada", "L1", s1, e1)}, new="res-1")
    c.ok("publishWaiver", "amy", t2, {"familyId": "laser", "text": "Laser waiver v2"}, {"waiver": waiver("laser", 2, "Laser waiver v2")})
    c.ok("listMemberReservations", "ada", t2, {"memberId": "ada"}, {"memberId": "ada", "reservations": [resv("res-1", "ada", "L1", s1, e1)]})
    c.refuse("checkIn", "ada", t3, {"reservationId": "res-1"}, "waiver_missing")
    c.refuse("createReservation", "ada", t3, {"memberId": "ada", "machineId": "L2", "start": s2, "end": e2}, "waiver_missing")
    c.ok("listMemberWaivers", "ada", t3, {"memberId": "ada"}, {"memberId": "ada", "signatures": [signature("ada", "laser", 1, t)]})
    c.ok("signWaiver", "ada", t3, {"memberId": "ada", "familyId": "laser"}, {"signature": signature("ada", "laser", 2, t3)})
    c.refuse("signWaiver", "ada", t3, {"memberId": "ada", "familyId": "laser"}, "already_signed")
    c.ok("checkIn", "ada", t3, {"reservationId": "res-1"}, {"reservation": resv("res-1", "ada", "L1", s1, e1, "checkedIn", cin=t3, rate=45)})
    # 37 minutes at 45 cents with the plus discount of 10 %: 1499
    c.ok("checkOut", "ada", t4, {"reservationId": "res-1"},
         {"reservation": resv("res-1", "ada", "L1", s1, e1, "completed", cin=t3, cout=t4, rate=45, metered=37, amount=1499),
          "charge": line("usage", "res-1", 37, 45, 10, t4)})
    c.ok("getAccount", "ada", t4, {"memberId": "ada"}, account("ada", unbilled=6499))
    c.ok("createReservation", "ada", t4, {"memberId": "ada", "machineId": "L2", "start": s2, "end": e2},
         {"reservation": resv("res-2", "ada", "L2", s2, e2)}, new="res-2")
    c.audit_log(t4)
    return c.build()


def x_people_3() -> dict:
    c = Case("X-people-3", "X-people", "ppe, lockouts, access, bookings: PPE and premises at check-in; ppe_missing, machine_locked_out, not_on_premises in order")
    t = world(c)
    t2, t3 = "2026-03-03T10:00Z", "2026-03-03T10:20Z"
    s, e = "2026-03-03T10:00Z", "2026-03-03T11:00Z"
    c.ok("addPpeItem", "sam", t, {"itemId": "gloves", "name": "Gloves", "stock": 1})
    c.ok("addPpeItem", "sam", t, {"itemId": "goggles", "name": "Goggles", "stock": 2})
    c.ok("requirePpe", "amy", t, {"familyId": "laser", "itemId": "gloves"})
    c.ok("requirePpe", "amy", t, {"familyId": "laser", "itemId": "goggles"})
    c.ok("createReservation", "ada", t, {"memberId": "ada", "machineId": "L1", "start": s, "end": e},
         {"reservation": resv("res-1", "ada", "L1", s, e)}, new="res-1")
    c.ok("issueBadge", "sam", t, {"memberId": "ada"}, {"badge": badge("bdg-1", "ada")}, new="bdg-1")
    c.ok("applyLockout", "sam", t, {"machineId": "L1", "reason": "belt"}, {"lockout": lockout("lck-1", "L1", "belt", "sam", t)}, new="lck-1")
    # no PPE, a lockout and a badge outside: ppe_missing decides
    c.refuse("checkIn", "ada", t2, {"reservationId": "res-1"}, "ppe_missing")
    c.ok("issuePpe", "sam", t2, {"memberId": "ada", "itemId": "gloves"}, {"item": ppe("gloves", "Gloves", 0)})
    c.refuse("checkIn", "ada", t2, {"reservationId": "res-1"}, "ppe_missing")  # goggles are still missing
    c.ok("issuePpe", "sam", t2, {"memberId": "ada", "itemId": "goggles"}, {"item": ppe("goggles", "Goggles", 1)})
    c.refuse("checkIn", "ada", t2, {"reservationId": "res-1"}, "machine_locked_out")
    c.ok("releaseLockout", "sam", t2, {"lockoutId": "lck-1"}, {"lockout": lockout("lck-1", "L1", "belt", "sam", t, t2)})
    c.refuse("checkIn", "ada", t2, {"reservationId": "res-1"}, "not_on_premises")
    c.ok("enterDoor", "ada", t2, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-1", "enter", "bdg-1", t2)})
    c.ok("checkIn", "ada", t2, {"reservationId": "res-1"}, {"reservation": resv("res-1", "ada", "L1", s, e, "checkedIn", cin=t2, rate=45)})
    # check-out is never gated by the premises: she leaves first; 20 minutes at 45 cents with 10 % off is 810
    c.ok("leaveDoor", "ada", t2, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada"), "event": door("dor-2", "leave", "bdg-1", t2)})
    c.ok("checkOut", "ada", t3, {"reservationId": "res-1"},
         {"reservation": resv("res-1", "ada", "L1", s, e, "completed", cin=t2, cout=t3, rate=45, metered=20, amount=810),
          "charge": line("usage", "res-1", 20, 45, 10, t3)})
    c.ok("getAccount", "ada", t3, {"memberId": "ada"}, account("ada", unbilled=5810))
    c.audit_log(t3)
    return c.build()


def x_people_4() -> dict:
    c = Case("X-people-4", "X-people", "staffing, bookings: supervision covered by two shifts, supervisor_required and shift_needed")
    t = world(c)
    d3, d4 = "2026-03-03T", "2026-03-04T"
    c.ok("setSupervision", "amy", t, {"familyId": "laser", "required": True}, {"supervision": {"familyId": "laser", "required": True}})
    r1 = {"memberId": "ada", "machineId": "L1", "start": d3 + "12:00Z", "end": d3 + "13:30Z"}
    c.refuse("createReservation", "ada", t, r1, "supervisor_required")  # no shift at all
    c.ok("addShift", "sam", t, {"userId": "sam", "start": d3 + "09:00Z", "end": d3 + "13:00Z"},
         {"shift": shift("shf-1", "sam", d3 + "09:00Z", d3 + "13:00Z")}, new="shf-1")
    c.refuse("createReservation", "ada", t, r1, "supervisor_required")  # 13:00 to 13:30 is not covered
    c.ok("addShift", "sam", t, {"userId": "sue", "start": d3 + "12:00Z", "end": d3 + "18:00Z"},
         {"shift": shift("shf-2", "sue", d3 + "12:00Z", d3 + "18:00Z")}, new="shf-2")
    c.ok("createReservation", "ada", t, r1, {"reservation": resv("res-1", "ada", "L1", d3 + "12:00Z", d3 + "13:30Z")}, new="res-1")  # pieced from two shifts
    c.refuse("createReservation", "ada", t, {"memberId": "ada", "machineId": "L2", "start": d3 + "17:30Z", "end": d3 + "18:30Z"}, "supervisor_required")
    c.ok("createReservation", "ada", t, {"memberId": "ada", "machineId": "L2", "start": d3 + "17:30Z", "end": d3 + "18:00Z"},
         {"reservation": resv("res-2", "ada", "L2", d3 + "17:30Z", d3 + "18:00Z")}, new="res-2")  # ends exactly at the end of the shift
    s1, s2 = shift("shf-1", "sam", d3 + "09:00Z", d3 + "13:00Z"), shift("shf-2", "sue", d3 + "12:00Z", d3 + "18:00Z")
    # the second shift alone would leave 13:00 to 13:30 of res-1 and all of res-2 uncovered
    c.refuse("cancelShift", "sam", t, {"shiftId": "shf-2"}, "shift_needed")
    c.ok("cancelShift", "sam", t, {"shiftId": "shf-1"}, {"shift": dict(s1, status="cancelled")})  # shf-2 still covers res-1
    c.refuse("cancelShift", "sam", t, {"shiftId": "shf-2"}, "shift_needed")
    c.ok("cancelReservation", "ada", t, {"reservationId": "res-1"}, {"reservation": resv("res-1", "ada", "L1", d3 + "12:00Z", d3 + "13:30Z", "cancelled")})
    c.refuse("cancelShift", "sam", t, {"shiftId": "shf-2"}, "shift_needed")  # res-2 still needs it
    c.ok("cancelReservation", "sam", t, {"reservationId": "res-2"}, {"reservation": resv("res-2", "ada", "L2", d3 + "17:30Z", d3 + "18:00Z", "cancelled")})
    c.ok("cancelShift", "amy", t, {"shiftId": "shf-2"}, {"shift": dict(s2, status="cancelled")})
    # a reservation that is not covered now does not stop a cancellation
    c.ok("setSupervision", "amy", t, {"familyId": "laser", "required": False})
    c.ok("createReservation", "ada", t, {"memberId": "ada", "machineId": "L1", "start": d4 + "10:00Z", "end": d4 + "11:00Z"},
         {"reservation": resv("res-3", "ada", "L1", d4 + "10:00Z", d4 + "11:00Z")}, new="res-3")
    c.ok("addShift", "sam", t, {"userId": "sam", "start": d4 + "10:30Z", "end": d4 + "12:00Z"},
         {"shift": shift("shf-3", "sam", d4 + "10:30Z", d4 + "12:00Z")}, new="shf-3")
    c.ok("setSupervision", "amy", t, {"familyId": "laser", "required": True})
    c.ok("cancelShift", "sam", t, {"shiftId": "shf-3"}, {"shift": shift("shf-3", "sam", d4 + "10:30Z", d4 + "12:00Z", "cancelled")})
    # supervision is checked before the machine and the member are found booked
    c.refuse("createReservation", "ada", t, {"memberId": "ada", "machineId": "L1", "start": d4 + "10:00Z", "end": d4 + "10:30Z"}, "supervisor_required")
    c.audit_log(t)
    return c.build()


def x_people_5() -> dict:
    c = Case("X-people-5", "X-people", "guests, billing, access: a guest pass over the allowance, its invoice line and the door")
    t = world(c)
    day, t2, t3 = "2026-03-10T10:00Z", INVOICE_AT, "2026-04-02T10:00Z"
    c.ok("setGuestAllowance", "amy", t, {"tier": "plus", "perMonth": 1, "feeCents": 500}, {"allowance": {"tier": "plus", "perMonth": 1, "feeCents": 500}})
    c.ok("setGuestAllowance", "amy", t, {"tier": "basic", "perMonth": 0, "feeCents": 300}, {"allowance": {"tier": "basic", "perMonth": 0, "feeCents": 300}})
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gus", "date": "2026-03-10"},
         {"guestPass": gpass("gst-1", "ada", "Gus", "2026-03-10", False)}, new="gst-1")
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gwen", "date": "2026-03-10"},
         {"guestPass": gpass("gst-2", "ada", "Gwen", "2026-03-10", True)}, new="gst-2")  # over the allowance of 1
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gil", "date": "2026-04-02"},
         {"guestPass": gpass("gst-3", "ada", "Gil", "2026-04-02", False)}, new="gst-3")  # April: no pass counts yet
    c.ok("issueGuestPass", "ada", t, {"memberId": "ada", "guestName": "Gaby", "date": "2026-03-12"},
         {"guestPass": gpass("gst-4", "ada", "Gaby", "2026-03-12", True)}, new="gst-4")
    c.ok("revokeGuestPass", "ada", t, {"guestPassId": "gst-4"}, {"guestPass": gpass("gst-4", "ada", "Gaby", "2026-03-12", True, "revoked")})
    c.ok("issueGuestPass", "bob", t, {"memberId": "bob", "guestName": "Bea", "date": "2026-03-10"},
         {"guestPass": gpass("gst-5", "bob", "Bea", "2026-03-10", True)}, new="gst-5")  # perMonth 0: charged
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", unbilled=6000))
    c.ok("getAccount", "bob", t, {"memberId": "bob"}, account("bob", unbilled=300))
    c.ok("issueBadge", "sam", day, {"memberId": "ada"}, {"badge": badge("bdg-1", "ada")}, new="bdg-1")
    c.ok("issueBadge", "sam", day, {"memberId": "bob"}, {"badge": badge("bdg-2", "bob")}, new="bdg-2")
    c.refuse("admitGuest", "ada", day, {"guestPassId": "gst-1", "badgeId": "bdg-1"}, "sponsor_not_inside")
    c.ok("enterDoor", "ada", day, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-1", "enter", "bdg-1", day)})
    c.refuse("admitGuest", "ada", day, {"guestPassId": "gst-3", "badgeId": "bdg-1"}, "pass_not_today")
    c.refuse("admitGuest", "ada", day, {"guestPassId": "gst-5", "badgeId": "bdg-1"}, "pass_of_another_member")
    c.refuse("admitGuest", "ada", day, {"guestPassId": "gst-4", "badgeId": "bdg-1"}, "invalid_state")
    c.ok("admitGuest", "ada", day, {"guestPassId": "gst-1", "badgeId": "bdg-1"}, {"event": door("dor-2", "guest", "bdg-1", day, "gst-1")})
    c.refuse("admitGuest", "ada", day, {"guestPassId": "gst-1", "badgeId": "bdg-1"}, "guest_already_admitted")
    c.ok("admitGuest", "ada", day, {"guestPassId": "gst-2", "badgeId": "bdg-1"}, {"event": door("dor-3", "guest", "bdg-1", day, "gst-2")})
    c.ok("listDoorEvents", "sam", day, {"date": "2026-03-10"},
         {"date": "2026-03-10", "events": [door("dor-1", "enter", "bdg-1", day), door("dor-2", "guest", "bdg-1", day, "gst-1"), door("dor-3", "guest", "bdg-1", day, "gst-2")]})
    # the charges become invoice lines of March: ada's course charge first (same time, created earlier), then her two
    # charged passes (the revoked one too); bob's one
    march = "2026-03"
    ada_lines = (COURSE, line("guest", "gst-2", 1, 500, 0, t), line("guest", "gst-4", 1, 500, 0, t))
    c.ok("generateInvoices", "sam", t2, {"month": march},
         {"month": march, "invoices": [
             plain("inv-1", "ada", "plus", march, t2, INVOICE_DUE, extra=ada_lines),
             plain("inv-2", "bob", "basic", march, t2, INVOICE_DUE, extra=(line("guest", "gst-5", 1, 300, 0, t),)),
             plain("inv-3", "eve", "pro", march, t2, INVOICE_DUE)]})
    c.ok("getAccount", "ada", t2, {"memberId": "ada"}, account("ada", open_=10900))
    # the pass of April admits its guest on its day, with the invoice still open
    c.ok("admitGuest", "ada", t3, {"guestPassId": "gst-3", "badgeId": "bdg-1"}, {"event": door("dor-4", "guest", "bdg-1", t3, "gst-3")})
    c.ok("listDoorEvents", "sam", t3, {"date": "2026-04-02"}, {"date": "2026-04-02", "events": [door("dor-4", "guest", "bdg-1", t3, "gst-3")]})
    c.audit_log(t3)
    return c.build()


def x_people_6() -> dict:
    c = Case("X-people-6", "X-people",
             "volunteering, staffing, billing, ledger: volunteer time approved into credit, seen in the account, applied to the next invoice")
    t = world(c)
    t10, t2, t3 = "2026-03-02T10:00Z", INVOICE_AT, "2026-04-02T10:00Z"
    c.ok("setVolunteerRate", "amy", t, {"centsPerHour": 1500}, {"centsPerHour": 1500})
    c.ok("addShift", "sam", t, {"userId": "sue", "start": t10, "end": "2026-03-02T12:00Z"},
         {"shift": shift("shf-1", "sue", t10, "2026-03-02T12:00Z")}, new="shf-1")
    c.ok("logVolunteerTime", "ada", t, {"memberId": "ada", "date": "2026-03-02", "minutes": 100, "note": "Workshop cleanup"},
         {"entry": vol("vol-1", "ada", "2026-03-02", 100, "Workshop cleanup")}, new="vol-1")
    c.refuse("decideVolunteerTime", "sam", t, {"entryId": "vol-1", "decision": "approved"}, "approver_not_on_shift")
    c.refuse("decideVolunteerTime", "sue", t, {"entryId": "vol-1", "decision": "approved"}, "approver_not_on_shift")  # the shift starts at 10:00
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", unbilled=5000))  # the course charge of the world
    c.ok("decideVolunteerTime", "sue", t10, {"entryId": "vol-1", "decision": "approved"},
         {"entry": vol("vol-1", "ada", "2026-03-02", 100, "Workshop cleanup", "approved", 2500)})
    c.ok("getAccount", "ada", t10, {"memberId": "ada"}, account("ada", credit=2500, unbilled=5000))
    # March invoice of 9900 for ada (membership 4900, course 5000): 2500 of credit is applied, the balance is 7400
    march = "2026-03"
    c.ok("generateInvoices", "sam", t2, {"month": march},
         {"month": march, "invoices": [
             plain("inv-1", "ada", "plus", march, t2, INVOICE_DUE, extra=(COURSE,), credit=2500),
             plain("inv-2", "bob", "basic", march, t2, INVOICE_DUE),
             plain("inv-3", "eve", "pro", march, t2, INVOICE_DUE)]})
    c.ok("getAccount", "ada", t2, {"memberId": "ada"}, account("ada", open_=7400))
    c.ok("listJournalEntries", "amy", t2, {"month": "2026-03"},
         {"month": "2026-03", "entries": [journal("jrn-1", t10, "expenseVolunteer", "creditLiability", 2500, "vol-1")]})
    c.ok("listJournalEntries", "amy", t2, {"month": "2026-04"},
         {"month": "2026-04", "entries": [
             journal("jrn-2", t2, "receivable", "revenueMembership", 4900, "inv-1"), journal("jrn-3", t2, "receivable", "revenueCourse", 5000, "inv-1"),
             journal("jrn-4", t2, "creditLiability", "receivable", 2500, "inv-1"),
             journal("jrn-5", t2, "receivable", "revenueMembership", 2500, "inv-2"), journal("jrn-6", t2, "receivable", "revenueMembership", 9900, "inv-3")]})
    # a payment of 8000 on the balance of 7400: 7400 applied, 600 becomes credit
    paid = invoice("inv-1", "ada", march, t2, INVOICE_DUE, [line("membership", "plus", 1, 4900, 0, t2), COURSE], credit=2500, paid=7400)
    c.ok("recordPayment", "ada", t3, {"invoiceId": "inv-1", "amountCents": 8000},
         {"payment": {"paymentId": "pay-1", "invoiceId": "inv-1", "memberId": "ada", "amountCents": 8000, "appliedCents": 7400, "creditedCents": 600, "at": t3},
          "invoice": paid, "creditBalanceCents": 600}, new="pay-1")
    c.ok("getAccount", "ada", t3, {"memberId": "ada"}, account("ada", credit=600))
    c.audit_log(t3)
    return c.build()


def x_people_7() -> dict:
    c = Case("X-people-7", "X-people", "access, billing: enter door is refused while an invoice is overdue and allowed after payment")
    t = world(c)
    due = INVOICE_DUE  # 2026-04-15T10:00Z, a Wednesday
    t1, t2, t3, t4, t5 = "2026-04-15T09:59Z", due, "2026-04-15T10:01Z", "2026-04-15T21:00Z", "2026-04-16T09:00Z"
    march = "2026-03"
    c.ok("issueBadge", "sam", t, {"memberId": "ada"}, {"badge": badge("bdg-1", "ada")}, new="bdg-1")
    c.ok("generateInvoices", "sam", INVOICE_AT, {"month": march},
         {"month": march, "invoices": [
             plain("inv-1", "ada", "plus", march, INVOICE_AT, due, extra=(COURSE,)), plain("inv-2", "bob", "basic", march, INVOICE_AT, due),
             plain("inv-3", "eve", "pro", march, INVOICE_AT, due)]})
    # one minute before the due date the invoice is not overdue yet; it totals 9900: membership 4900, course 5000
    c.ok("getAccount", "ada", t1, {"memberId": "ada"}, account("ada", open_=9900))
    c.ok("enterDoor", "ada", t1, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-1", "enter", "bdg-1", t1)})
    c.ok("leaveDoor", "ada", t1, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada"), "event": door("dor-2", "leave", "bdg-1", t1)})
    # at the due date it is overdue
    c.ok("getAccount", "ada", t2, {"memberId": "ada"}, account("ada", open_=9900, overdue=9900))
    c.refuse("enterDoor", "ada", t2, {"badgeId": "bdg-1"}, "account_overdue")
    part = invoice("inv-1", "ada", march, INVOICE_AT, due, [line("membership", "plus", 1, 4900, 0, INVOICE_AT), COURSE], paid=1000, overdue=True)
    c.ok("recordPayment", "ada", t2, {"invoiceId": "inv-1", "amountCents": 1000},
         {"payment": {"paymentId": "pay-1", "invoiceId": "inv-1", "memberId": "ada", "amountCents": 1000, "appliedCents": 1000, "creditedCents": 0, "at": t2},
          "invoice": part, "creditBalanceCents": 0}, new="pay-1")
    c.ok("getAccount", "ada", t3, {"memberId": "ada"}, account("ada", open_=8900, overdue=8900))
    c.refuse("enterDoor", "ada", t3, {"badgeId": "bdg-1"}, "account_overdue")  # a part payment is not enough
    c.refuse("enterDoor", "ada", t4, {"badgeId": "bdg-1"}, "outside_opening_hours")  # the hours come before the account
    done = invoice("inv-1", "ada", march, INVOICE_AT, due, [line("membership", "plus", 1, 4900, 0, INVOICE_AT), COURSE], paid=9900)
    c.ok("recordPayment", "sam", t4, {"invoiceId": "inv-1", "amountCents": 8900},
         {"payment": {"paymentId": "pay-2", "invoiceId": "inv-1", "memberId": "ada", "amountCents": 8900, "appliedCents": 8900, "creditedCents": 0, "at": t4},
          "invoice": done, "creditBalanceCents": 0}, new="pay-2")
    c.refuse("recordPayment", "ada", t4, {"invoiceId": "inv-1", "amountCents": 100}, "invoice_paid")
    c.ok("getAccount", "ada", t4, {"memberId": "ada"}, account("ada"))
    c.ok("enterDoor", "ada", t5, {"badgeId": "bdg-1"}, {"badge": badge("bdg-1", "ada", inside=True), "event": door("dor-3", "enter", "bdg-1", t5)})
    c.audit_log(t5)
    return c.build()


def build_cases() -> list:
    return [b_notices_1(), b_notices_2(), b_households_1(), b_households_2(), b_guardians_1(), b_guardians_2(), b_guardians_3(),
            b_waivers_1(), b_waivers_2(), b_ppe_1(), b_ppe_2(), b_staffing_1(), b_staffing_2(), b_guests_1(), b_guests_2(),
            b_volunteering_1(), b_volunteering_2(), b_access_1(), b_access_2(),
            x_people_1(), x_people_2(), x_people_3(), x_people_4(), x_people_5(), x_people_6(), x_people_7()]
