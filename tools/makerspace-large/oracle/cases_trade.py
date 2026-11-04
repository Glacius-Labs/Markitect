"""Hidden cases of the trade cluster: hazmat, purchasing, stocktake, projects, quotes, jobs and the changes to materials.

Written from the requirements document alone. Every expected amount is computed by hand in the comment next to it.
"""
from __future__ import annotations

from typing import Any

from kit import Case, world, has, ref, ANY  # noqa: F401

T0 = "2026-03-02T09:00Z"


# --- Small builders of expected records (shapes only; every number is stated by the case) ------------------------------

def rf(c: Case, op: str, actor: str, at: str, args: dict[str, Any] | None, code: str) -> None:
    """A refusal. A refused request goes through check 3 too, so its time must not be earlier than the last accepted one."""
    if c.last_ok is not None and at < c.last_ok:
        raise ValueError(f"{c.id}: refusal at {at} is before {c.last_ok}")
    c.refuse(op, actor, at, args, code)


def mat(mid: str, name: str, price: int, stock: int) -> dict[str, Any]:
    return {"materialId": mid, "name": name, "unitPriceCents": price, "stock": stock}


def pur(pid: str, member: str, material: str, qty: int, price: int, disc: int, amount: int, at: str) -> dict[str, Any]:
    return {"purchaseId": pid, "memberId": member, "materialId": material, "quantity": qty, "unitPriceCents": price,
            "discountPercent": disc, "amountCents": amount, "at": at}


def bought(pid: str, member: str, material: str, qty: int, price: int, disc: int, amount: int, at: str,
           name: str, stock: int) -> dict[str, Any]:
    """The result of a successful purchaseMaterial: the purchase and the material after the change."""
    return {"purchase": pur(pid, member, material, qty, price, disc, amount, at), "material": mat(material, name, price, stock)}


def cls(mid: str, family: str, limit: int) -> dict[str, Any]:
    return {"materialId": mid, "familyId": family, "monthlyLimit": limit}


def oline(material: str, qty: int, cost: int, received: int = 0) -> dict[str, Any]:
    return {"materialId": material, "quantity": qty, "unitCostCents": cost, "receivedQuantity": received}


def order(oid: str, supplier: str, status: str, by: str, total: int, lines: list) -> dict[str, Any]:
    return {"orderId": oid, "supplierId": supplier, "status": status, "createdBy": by, "totalCents": total, "lines": lines}


def supplier(sid: str, name: str, status: str = "active") -> dict[str, Any]:
    return {"supplierId": sid, "name": name, "status": status}


def cnt(material: str, counted: int, diff: int | None) -> dict[str, Any]:
    return {"materialId": material, "counted": counted, "difference": diff}


def stk(sid: str, status: str, counts: list) -> dict[str, Any]:
    return {"stocktakeId": sid, "status": status, "counts": counts}


def jrn(n: int, at: str, debit: str, credit: str, amount: int, memo: str) -> dict[str, Any]:
    return {"entryId": f"jrn-{n}", "at": at, "debitAccount": debit, "creditAccount": credit, "amountCents": amount, "memo": memo}


def proj(pid: str, owner: str, title: str, collabs: list | None = None, status: str = "open",
         res: list | None = None, purchases: list | None = None) -> dict[str, Any]:
    return {"projectId": pid, "ownerId": owner, "title": title, "collaboratorIds": collabs or [], "status": status,
            "reservationIds": res or [], "purchaseIds": purchases or []}


def share(member: str, usage: int, material: int) -> dict[str, Any]:
    return {"memberId": member, "usageCents": usage, "materialCents": material}


def cost(pid: str, usage: int, material: int, by: list) -> dict[str, Any]:
    return {"projectId": pid, "usageCents": usage, "materialCents": material, "totalCents": usage + material, "byMember": by}


def qline(kind: str, ref_: str, qty: int, price: int | None = None) -> dict[str, Any]:
    return {"lineKind": kind, "ref": ref_, "quantity": qty, "unitPriceCents": price}


def quote(qid: str, customer: str, title: str, status: str, lines: list, sub: int = 0, total: int = 0,
          sent: str | None = None, valid: str | None = None) -> dict[str, Any]:
    return {"quoteId": qid, "customerId": customer, "title": title, "status": status, "lines": lines,
            "subtotalCents": sub, "totalCents": total, "sentAt": sent, "validUntil": valid}


def job(jid: str, qid: str, customer: str, maker: str, status: str, total: int, credit: int | None, paid: int) -> dict[str, Any]:
    return {"jobId": jid, "quoteId": qid, "customerId": customer, "makerId": maker, "status": status,
            "totalCents": total, "makerCreditCents": credit, "paidCents": paid}


def resv(rid: str, member: str, machine: str, start: str, end: str, status: str = "reserved", cin: str | None = None,
         cout: str | None = None, rate: int | None = None, metered: int | None = None, amount: int | None = None) -> dict[str, Any]:
    return {"reservationId": rid, "memberId": member, "machineId": machine, "start": start, "end": end, "status": status,
            "checkedInAt": cin, "checkedOutAt": cout, "rateCentsPerMinute": rate, "meteredMinutes": metered, "amountCents": amount}


def redemption(code: str, member: str, percent: int, at: str, used_at: str | None = None, used_on: str | None = None) -> dict[str, Any]:
    return {"code": code, "memberId": member, "percent": percent, "redeemedAt": at, "usedAt": used_at, "usedOn": used_on}


def account(member: str, credit: int, open_: int, overdue: int, unbilled: int) -> dict[str, Any]:
    return {"account": {"memberId": member, "creditBalanceCents": credit, "openBalanceCents": open_,
                        "overdueBalanceCents": overdue, "unbilledCents": unbilled}}


ACCOUNTS = ["cash", "creditLiability", "depositLiability", "expenseGoodwill", "expenseMaker", "expensePayout", "expenseVolunteer",
            "expenseWriteOff", "inventory", "payable", "receivable", "revenueCourse", "revenueDeposits", "revenueEvents",
            "revenueGuests", "revenueJobs", "revenueLateFees", "revenueLockers", "revenueMaterial", "revenueMembership",
            "revenuePasses", "revenueRefunds", "revenueUsage"]


def trial(debits: dict[str, int], credits: dict[str, int]) -> dict[str, Any]:
    """The trial balance with these per-account sums (stated by the case); every other account is 0 and `none`."""
    rows = []
    for name in ACCOUNTS:
        d, k = debits.get(name, 0), credits.get(name, 0)
        rows.append({"account": name, "debitCents": d, "creditCents": k, "balanceCents": abs(d - k),
                     "side": "debit" if d > k else "credit" if k > d else "none"})
    return {"rows": rows, "totalDebitCents": sum(debits.values()), "totalCreditCents": sum(credits.values())}


def certify(c: Case, member: str, at: str = T0) -> None:
    """Certify a member on family laser with the course of the shared world (capacity 2, ada's place is free again)."""
    c.ok("enrollInCourse", "sam", at, {"courseId": "laser-101", "memberId": member})
    c.ok("completeCourse", "sam", at, {"courseId": "laser-101", "memberId": member})


def make_quote(c: Case, customer: str, title: str, lines: list, at: str, decision: str | None = None,
               decided_at: str | None = None, send: bool = True) -> None:
    """Setup only: create a quote (the next number is the caller's business), add its lines, send it, maybe decide it."""
    n = sum(1 for s in c.steps if s["request"]["op"] == "createQuote" and s["expect"]["status"] == "ok") + 1
    c.ok("createQuote", "sam", at, {"customerId": customer, "title": title}, new=f"quo-{n}")
    for kind, rf_, qty in lines:
        c.ok("addQuoteLine", "sam", at, {"quoteId": f"quo-{n}", "lineKind": kind, "ref": rf_, "quantity": qty})
    if send:
        c.ok("sendQuote", "sam", at, {"quoteId": f"quo-{n}"})
    if decision:
        c.ok("decideQuote", "sam", decided_at or at, {"quoteId": f"quo-{n}", "decision": decision})


# --- hazmat ------------------------------------------------------------------------------------------------------------

def b_hazmat_1() -> Case:
    c = Case("B-hazmat-1", "B-hazmat", "hazmat: classify, replace, declassify and list; hazmat/classify-hazard, declassify-hazard, list-hazards")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "resin", "name": "Epoxy resin", "unitPriceCents": 300}, {"material": mat("resin", "Epoxy resin", 300, 0)})
    c.ok("addMaterial", "sam", T0, {"materialId": "acid", "name": "Etching acid", "unitPriceCents": 800}, {"material": mat("acid", "Etching acid", 800, 0)})
    c.ok("listHazards", "bob", T0, {}, {"hazards": []})
    # Permissions: admin only; the role check (4) comes before the existence check (5).
    rf(c, "classifyHazard", "sam", T0, {"materialId": "resin", "familyId": "laser", "monthlyLimit": 10}, "forbidden")
    rf(c, "classifyHazard", "bob", T0, {"materialId": "ghost", "familyId": "laser", "monthlyLimit": 5}, "forbidden")
    rf(c, "classifyHazard", "admin", T0, {"materialId": "ghost", "familyId": "laser", "monthlyLimit": 5}, "not_found")
    rf(c, "classifyHazard", "admin", T0, {"materialId": "resin", "familyId": "ghost", "monthlyLimit": 5}, "not_found")
    rf(c, "classifyHazard", "admin", T0, {"materialId": "resin", "familyId": "laser", "monthlyLimit": 0}, "invalid_argument")
    rf(c, "classifyHazard", "admin", T0, {"materialId": "resin", "familyId": "laser", "monthlyLimit": 10001}, "invalid_argument")
    rf(c, "classifyHazard", "admin", T0, {"materialId": "resin", "familyId": "laser", "monthlyLimit": "5"}, "invalid_argument")
    c.ok("classifyHazard", "admin", T0, {"materialId": "resin", "familyId": "laser", "monthlyLimit": 10}, {"classification": cls("resin", "laser", 10)})
    c.ok("classifyHazard", "amy", T0, {"materialId": "acid", "familyId": "lathe", "monthlyLimit": 10000}, {"classification": cls("acid", "lathe", 10000)})
    # Listed by materialId, not in the order of classification.
    c.ok("listHazards", "sam", T0, {}, {"hazards": [cls("acid", "lathe", 10000), cls("resin", "laser", 10)]})
    # A new classification replaces the old one.
    c.ok("classifyHazard", "admin", T0, {"materialId": "resin", "familyId": "lathe", "monthlyLimit": 3}, {"classification": cls("resin", "lathe", 3)})
    c.ok("classifyHazard", "admin", T0, {"materialId": "acid", "familyId": "laser", "monthlyLimit": 1}, {"classification": cls("acid", "laser", 1)})
    c.ok("listHazards", "ada", T0, {}, {"hazards": [cls("acid", "laser", 1), cls("resin", "lathe", 3)]})
    # Declassify returns the removed classification.
    rf(c, "declassifyHazard", "sam", T0, {"materialId": "acid"}, "forbidden")
    c.ok("declassifyHazard", "admin", T0, {"materialId": "acid"}, {"classification": cls("acid", "laser", 1)})
    rf(c, "declassifyHazard", "admin", T0, {"materialId": "acid"}, "not_hazardous")
    rf(c, "declassifyHazard", "admin", T0, {"materialId": "ghost"}, "not_found")
    rf(c, "declassifyHazard", "admin", T0, {"materialId": "ply"}, "not_hazardous")
    c.ok("listHazards", "admin", T0, {}, {"hazards": [cls("resin", "lathe", 3)]})
    c.audit_log(T0)
    return c


def b_hazmat_2() -> Case:
    """The gate: certification first, then the monthly limit; months, other members, other materials, certification dates."""
    c = Case("B-hazmat-2", "B-hazmat", "hazmat/hazard-gate: not certified before limit, units of the month, at the limit, new month, expiry date")
    world(c)
    certify(c, "eve")
    c.ok("addMaterial", "sam", T0, {"materialId": "resin", "name": "Epoxy resin", "unitPriceCents": 300})
    c.ok("restockMaterial", "sam", T0, {"materialId": "resin", "quantity": 100})
    c.ok("classifyHazard", "admin", T0, {"materialId": "resin", "familyId": "laser", "monthlyLimit": 10})
    # bob has no certification: not certified, even though 11 is also over the limit. Judged on the member bought for.
    rf(c, "purchaseMaterial", "bob", T0, {"memberId": "bob", "materialId": "resin", "quantity": 11}, "hazard_not_certified")
    rf(c, "purchaseMaterial", "sam", T0, {"memberId": "bob", "materialId": "resin", "quantity": 1}, "hazard_not_certified")
    rf(c, "purchaseMaterial", "bob", T0, {"memberId": "ada", "materialId": "resin", "quantity": 1}, "forbidden")
    t = "2026-03-05T10:00Z"
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 6},
         bought("pur-1", "ada", "resin", 6, 300, 0, 1800, t, "Epoxy resin", 94), new="pur-1")  # 6 x 300 = 1800
    t = "2026-03-06T10:00Z"
    # Exactly the limit is allowed (eve: 10 of 10); a member's units do not count for another member.
    c.ok("purchaseMaterial", "eve", t, {"memberId": "eve", "materialId": "resin", "quantity": 10},
         bought("pur-2", "eve", "resin", 10, 300, 0, 3000, t, "Epoxy resin", 84), new="pur-2")
    rf(c, "purchaseMaterial", "eve", t, {"memberId": "eve", "materialId": "resin", "quantity": 1}, "hazard_limit_exceeded")
    t = "2026-03-07T10:00Z"
    # Another material (unclassified) is not gated and not counted.
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "ply", "quantity": 20},
         bought("pur-3", "ada", "ply", 20, 150, 0, 3000, t, "Plywood", 80), new="pur-3")  # 20 x 150 = 3000
    t = "2026-03-20T09:00Z"
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 5}, "hazard_limit_exceeded")  # 6 + 5 = 11 > 10
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 4},
         bought("pur-4", "ada", "resin", 4, 300, 0, 1200, t, "Epoxy resin", 80), new="pur-4")  # 6 + 4 = 10, allowed
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 1}, "hazard_limit_exceeded")  # 11
    t = "2026-03-31T23:59Z"
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 1}, "hazard_limit_exceeded")  # still March
    t = "2026-04-01T00:00Z"
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 10},
         bought("pur-5", "ada", "resin", 10, 300, 0, 3000, t, "Epoxy resin", 70), new="pur-5")  # the count starts again at 0
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 1}, "hazard_limit_exceeded")
    # Certification validity is by date: ada's runs from 2026-03-02 and expires on 2027-03-02 (365 days), not valid that day.
    t = "2027-03-01T23:59Z"
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 1},
         bought("pur-6", "ada", "resin", 1, 300, 0, 300, t, "Epoxy resin", 69), new="pur-6")
    t = "2027-03-02T00:00Z"
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "resin", "quantity": 1}, "hazard_not_certified")
    rf(c, "purchaseMaterial", "eve", t, {"memberId": "eve", "materialId": "resin", "quantity": 1}, "hazard_not_certified")
    c.ok("listMaterials", "bob", t, {}, {"materials": [mat("ply", "Plywood", 150, 80), mat("resin", "Epoxy resin", 300, 69)]})
    c.audit_log(t)
    return c


def b_hazmat_3() -> Case:
    """The classification in force at the request time is judged; earlier purchases still count."""
    c = Case("B-hazmat-3", "B-hazmat", "hazmat/classify-hazard, hazard-gate: a classification gates later purchases, earlier ones count")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "dye", "name": "Dye", "unitPriceCents": 200})
    c.ok("restockMaterial", "sam", T0, {"materialId": "dye", "quantity": 100})
    t = "2026-03-03T10:00Z"
    # Not classified yet: nobody is gated; bought units are remembered.
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "dye", "quantity": 6},
         bought("pur-1", "ada", "dye", 6, 200, 0, 1200, t, "Dye", 94), new="pur-1")  # 6 x 200
    t = "2026-03-04T09:00Z"
    c.ok("classifyHazard", "admin", t, {"materialId": "dye", "familyId": "laser", "monthlyLimit": 10}, {"classification": cls("dye", "laser", 10)})
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "dye", "quantity": 1}, "hazard_not_certified")
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "dye", "quantity": 5}, "hazard_limit_exceeded")  # 6 earlier + 5
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "dye", "quantity": 4},
         bought("pur-2", "ada", "dye", 4, 200, 0, 800, t, "Dye", 90), new="pur-2")
    # Raising the limit changes only what is judged next.
    c.ok("classifyHazard", "admin", t, {"materialId": "dye", "familyId": "laser", "monthlyLimit": 12}, {"classification": cls("dye", "laser", 12)})
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "dye", "quantity": 2},
         bought("pur-3", "ada", "dye", 2, 200, 0, 400, t, "Dye", 88), new="pur-3")  # 10 + 2 = 12
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "dye", "quantity": 1}, "hazard_limit_exceeded")  # 13 > 12
    # Another family: ada has no lathe certification.
    c.ok("classifyHazard", "admin", t, {"materialId": "dye", "familyId": "lathe", "monthlyLimit": 100}, {"classification": cls("dye", "lathe", 100)})
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "dye", "quantity": 1}, "hazard_not_certified")
    # Declassified: no gate any more.
    c.ok("declassifyHazard", "admin", t, {"materialId": "dye"}, {"classification": cls("dye", "lathe", 100)})
    c.ok("listHazards", "admin", t, {}, {"hazards": []})
    c.ok("purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "dye", "quantity": 3},
         bought("pur-4", "bob", "dye", 3, 200, 0, 600, t, "Dye", 85), new="pur-4")
    c.ok("listMaterials", "admin", t, {}, {"materials": [mat("dye", "Dye", 200, 85), mat("ply", "Plywood", 150, 100)]})
    c.audit_log(t)
    return c


# --- materials ---------------------------------------------------------------------------------------------------------

def b_materials_1() -> Case:
    c = Case("B-materials-1", "B-materials", "materials: add, restock, purchase and its refusals in order")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75}, {"material": mat("foam", "Foam board", 75, 0)})
    rf(c, "addMaterial", "admin", T0, {"materialId": "ply", "name": "Other", "unitPriceCents": 10}, "already_exists")
    rf(c, "addMaterial", "ada", T0, {"materialId": "x1", "name": "X", "unitPriceCents": 10}, "forbidden")
    rf(c, "addMaterial", "sam", T0, {"materialId": "x1", "name": "X", "unitPriceCents": 0}, "invalid_argument")
    rf(c, "addMaterial", "sam", T0, {"materialId": "x1", "name": "X", "unitPriceCents": 1000001}, "invalid_argument")
    rf(c, "addMaterial", "sam", T0, {"materialId": "x1", "name": "   ", "unitPriceCents": 10}, "invalid_argument")
    c.ok("restockMaterial", "sam", T0, {"materialId": "foam", "quantity": 30}, {"material": mat("foam", "Foam board", 75, 30)})
    rf(c, "restockMaterial", "sam", T0, {"materialId": "foam", "quantity": 0}, "invalid_argument")
    rf(c, "restockMaterial", "sam", T0, {"materialId": "foam", "quantity": 100001}, "invalid_argument")
    rf(c, "restockMaterial", "sam", T0, {"materialId": "ghost", "quantity": 1}, "not_found")
    rf(c, "restockMaterial", "ada", T0, {"materialId": "foam", "quantity": 1}, "forbidden")
    c.ok("listMaterials", "bob", T0, {}, {"materials": [mat("foam", "Foam board", 75, 30), mat("ply", "Plywood", 150, 100)]})
    t = "2026-03-02T10:00Z"
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "ply", "quantity": 3},
         bought("pur-1", "ada", "ply", 3, 150, 0, 450, t, "Plywood", 97), new="pur-1")  # 3 x 150 = 450, no tier discount on materials
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "ada", "materialId": "ply", "quantity": 1}, "forbidden")
    # Staff buy for a member; exactly the whole stock is allowed.
    c.ok("purchaseMaterial", "sam", t, {"memberId": "bob", "materialId": "foam", "quantity": 30},
         bought("pur-2", "bob", "foam", 30, 75, 0, 2250, t, "Foam board", 0), new="pur-2")  # 30 x 75 = 2250
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "foam", "quantity": 1}, "insufficient_stock")
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "ply", "quantity": 0}, "invalid_argument")
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "ply", "quantity": 10001}, "invalid_argument")
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "ply", "quantity": 10000}, "insufficient_stock")
    rf(c, "purchaseMaterial", "sam", t, {"memberId": "ghost", "materialId": "ply", "quantity": 1}, "not_found")
    rf(c, "purchaseMaterial", "sam", t, {"memberId": "sue", "materialId": "ply", "quantity": 1}, "not_found")  # staff is no member
    rf(c, "purchaseMaterial", "sam", t, {"memberId": "bob", "materialId": "ghost", "quantity": 1}, "not_found")
    # A suspended member cannot buy; that comes before the stock check.
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "suspended"})
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "ply", "quantity": 1}, "member_suspended")
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "foam", "quantity": 1}, "member_suspended")
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "good"})
    c.ok("purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "ply", "quantity": 1},
         bought("pur-3", "bob", "ply", 1, 150, 0, 150, t, "Plywood", 96), new="pur-3")
    c.ok("listMaterials", "sam", t, {}, {"materials": [mat("foam", "Foam board", 75, 0), mat("ply", "Plywood", 150, 96)]})
    # Each purchase made a material charge: ada has 5000 (the course of the world) + 450, bob 2250 + 150.
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", 0, 0, 0, 5450))
    c.ok("getAccount", "bob", t, {"memberId": "bob"}, account("bob", 0, 0, 0, 2400))
    c.audit_log(t)
    return c


def b_materials_2() -> Case:
    """Redeemed promotions on purchases: oldest unused that covers materials, used once, refused purchases use none."""
    c = Case("B-materials-2", "B-materials", "materials/purchase-material, promotions/redeemed-promotions: discount, scope, oldest first, used once")
    world(c)
    for code, pct, scope in (("A", 20, "course"), ("B", 15, "both"), ("C", 30, "material"), ("D", 50, "material")):
        c.ok("addPromotion", "admin", T0, {"code": code, "percent": pct, "scope": scope, "maxRedemptions": 5, "validUntil": "2026-12-31"})
    for code in ("A", "B", "C"):
        c.ok("redeemPromotion", "ada", T0, {"memberId": "ada", "code": code})
    t1, t2, t3 = "2026-03-03T10:00Z", "2026-03-03T11:00Z", "2026-03-03T12:00Z"
    # Oldest unused that covers materials is B (A is course only): 3 x 150 = 450, 15 % off: (450 x 85 + 50) div 100 = 38300 div 100 = 383.
    c.ok("purchaseMaterial", "ada", t1, {"memberId": "ada", "materialId": "ply", "quantity": 3},
         bought("pur-1", "ada", "ply", 3, 150, 15, 383, t1, "Plywood", 97), new="pur-1")
    c.ok("listMemberPromotions", "ada", t1, {"memberId": "ada"}, {"memberId": "ada", "redemptions": [
        redemption("A", "ada", 20, T0), redemption("B", "ada", 15, T0, t1, "pur-1"), redemption("C", "ada", 30, T0)]})
    # Next is C, 30 %: (2 x 150 = 300; (300 x 70 + 50) div 100 = 21050 div 100 = 210).
    c.ok("purchaseMaterial", "ada", t2, {"memberId": "ada", "materialId": "ply", "quantity": 2},
         bought("pur-2", "ada", "ply", 2, 150, 30, 210, t2, "Plywood", 95), new="pur-2")
    # Only A is left and it does not cover materials: no discount, A stays unused.
    c.ok("purchaseMaterial", "ada", t3, {"memberId": "ada", "materialId": "ply", "quantity": 1},
         bought("pur-3", "ada", "ply", 1, 150, 0, 150, t3, "Plywood", 94), new="pur-3")
    c.ok("listMemberPromotions", "ada", t3, {"memberId": "ada"}, {"memberId": "ada", "redemptions": [
        redemption("A", "ada", 20, T0), redemption("B", "ada", 15, T0, t1, "pur-1"), redemption("C", "ada", 30, T0, t2, "pur-2")]})
    # A refused purchase uses nothing: eve's 95 units exceed the stock of 94.
    c.ok("redeemPromotion", "eve", t3, {"memberId": "eve", "code": "D"})
    rf(c, "purchaseMaterial", "eve", t3, {"memberId": "eve", "materialId": "ply", "quantity": 95}, "insufficient_stock")
    c.ok("listMemberPromotions", "eve", t3, {"memberId": "eve"}, {"memberId": "eve", "redemptions": [redemption("D", "eve", 50, t3)]})
    # 4 x 150 = 600 at 50 %: (600 x 50 + 50) div 100 = 30050 div 100 = 300.
    c.ok("purchaseMaterial", "eve", t3, {"memberId": "eve", "materialId": "ply", "quantity": 4},
         bought("pur-4", "eve", "ply", 4, 150, 50, 300, t3, "Plywood", 90), new="pur-4")
    c.ok("listMemberPromotions", "eve", t3, {"memberId": "eve"}, {"memberId": "eve", "redemptions": [redemption("D", "eve", 50, t3, t3, "pur-4")]})
    c.ok("purchaseMaterial", "eve", t3, {"memberId": "eve", "materialId": "ply", "quantity": 1},
         bought("pur-5", "eve", "ply", 1, 150, 0, 150, t3, "Plywood", 89), new="pur-5")  # D was used once
    # The charges carry the discounted amounts: ada 5000 (course) + 383 + 210 + 150, eve 300 + 150.
    c.ok("getAccount", "ada", t3, {"memberId": "ada"}, account("ada", 0, 0, 0, 5743))
    c.ok("getAccount", "eve", t3, {"memberId": "eve"}, account("eve", 0, 0, 0, 450))
    c.audit_log(t3)
    return c


# --- purchasing --------------------------------------------------------------------------------------------------------

def b_purchasing_1() -> Case:
    c = Case("B-purchasing-1", "B-purchasing", "purchasing: suppliers, orders, lines, submit, list and the blocked supplier at creation")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75})
    c.ok("addSupplier", "sam", T0, {"supplierId": "acme", "name": "Acme Supply"}, {"supplier": supplier("acme", "Acme Supply")})
    rf(c, "addSupplier", "admin", T0, {"supplierId": "acme", "name": "Other"}, "already_exists")
    rf(c, "addSupplier", "bob", T0, {"supplierId": "bobco", "name": "Bob Co"}, "forbidden")
    c.ok("addSupplier", "admin", T0, {"supplierId": "zeta", "name": "Zeta Parts"}, {"supplier": supplier("zeta", "Zeta Parts")})
    c.ok("addSupplier", "admin", T0, {"supplierId": "mono", "name": "Mono Ltd"})
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, {"order": order("po-1", "acme", "draft", "sam", 0, [])}, new="po-1")
    rf(c, "createOrder", "sam", T0, {"supplierId": "ghost"}, "not_found")
    rf(c, "createOrder", "bob", T0, {"supplierId": "acme"}, "forbidden")
    L1 = oline("ply", 100, 120)
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 100, "unitCostCents": 120},
         {"order": order("po-1", "acme", "draft", "sam", 12000, [L1])})  # 100 x 120
    # Lines keep the order they were added (ply before foam), not the order of the material identifiers.
    L2 = oline("foam", 10, 50)
    c.ok("addOrderLine", "admin", T0, {"orderId": "po-1", "materialId": "foam", "quantity": 10, "unitCostCents": 50},
         {"order": order("po-1", "acme", "draft", "sam", 12500, [L1, L2])})  # 12000 + 10 x 50
    rf(c, "addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 5, "unitCostCents": 9}, "duplicate_line")
    for qty, cost_ in ((0, 5), (100001, 5), (5, 0), (5, 1000001)):
        rf(c, "addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "foam", "quantity": qty, "unitCostCents": cost_}, "invalid_argument")
    rf(c, "addOrderLine", "sam", T0, {"orderId": "ghost", "materialId": "foam", "quantity": 1, "unitCostCents": 1}, "not_found")
    rf(c, "addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "ghost", "quantity": 1, "unitCostCents": 1}, "not_found")
    rf(c, "addOrderLine", "bob", T0, {"orderId": "po-1", "materialId": "foam", "quantity": 1, "unitCostCents": 1}, "forbidden")
    # An order without line cannot be submitted.
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, {"order": order("po-2", "acme", "draft", "sam", 0, [])}, new="po-2")
    rf(c, "submitOrder", "sam", T0, {"orderId": "po-2"}, "empty_order")
    rf(c, "submitOrder", "sam", T0, {"orderId": "ghost"}, "not_found")
    rf(c, "submitOrder", "bob", T0, {"orderId": "po-1"}, "forbidden")
    c.ok("submitOrder", "sam", T0, {"orderId": "po-1"}, {"order": order("po-1", "acme", "submitted", "sam", 12500, [L1, L2])})
    rf(c, "submitOrder", "sam", T0, {"orderId": "po-1"}, "invalid_state")
    # Not draft comes before a duplicate line.
    rf(c, "addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 5, "unitCostCents": 9}, "invalid_state")
    # A very large order total is plain arithmetic: 100000 x 1000000.
    c.ok("createOrder", "admin", T0, {"supplierId": "zeta"}, {"order": order("po-3", "zeta", "draft", "admin", 0, [])}, new="po-3")
    c.ok("addOrderLine", "admin", T0, {"orderId": "po-3", "materialId": "foam", "quantity": 100000, "unitCostCents": 1000000},
         {"order": order("po-3", "zeta", "draft", "admin", 100000000000, [oline("foam", 100000, 1000000)])})
    # List: creation order, every status; echoes the supplier; empty list when none.
    c.ok("listOrders", "sam", T0, {"supplierId": "acme"}, {"supplierId": "acme", "orders": [
        order("po-1", "acme", "submitted", "sam", 12500, [L1, L2]), order("po-2", "acme", "draft", "sam", 0, [])]})
    c.ok("listOrders", "admin", T0, {"supplierId": "zeta"}, {"supplierId": "zeta", "orders": [
        order("po-3", "zeta", "draft", "admin", 100000000000, [oline("foam", 100000, 1000000)])]})
    c.ok("listOrders", "admin", T0, {"supplierId": "mono"}, {"supplierId": "mono", "orders": []})
    rf(c, "listOrders", "bob", T0, {"supplierId": "acme"}, "forbidden")
    rf(c, "listOrders", "sam", T0, {"supplierId": "ghost"}, "not_found")
    # Supplier status: admin only; setting the current value succeeds; blocked stops new orders only.
    rf(c, "setSupplierStatus", "sam", T0, {"supplierId": "acme", "status": "blocked"}, "forbidden")
    rf(c, "setSupplierStatus", "admin", T0, {"supplierId": "acme", "status": "closed"}, "invalid_argument")
    rf(c, "setSupplierStatus", "admin", T0, {"supplierId": "ghost", "status": "blocked"}, "not_found")
    c.ok("setSupplierStatus", "admin", T0, {"supplierId": "acme", "status": "active"}, {"supplier": supplier("acme", "Acme Supply")})
    c.ok("setSupplierStatus", "admin", T0, {"supplierId": "acme", "status": "blocked"}, {"supplier": supplier("acme", "Acme Supply", "blocked")})
    c.ok("setSupplierStatus", "admin", T0, {"supplierId": "acme", "status": "blocked"}, {"supplier": supplier("acme", "Acme Supply", "blocked")})
    rf(c, "createOrder", "sam", T0, {"supplierId": "acme"}, "supplier_blocked")
    # The status is not judged at lines and submit: the draft of the blocked supplier goes on.
    L3 = oline("foam", 3, 7)
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-2", "materialId": "foam", "quantity": 3, "unitCostCents": 7},
         {"order": order("po-2", "acme", "draft", "sam", 21, [L3])})
    c.ok("submitOrder", "sam", T0, {"orderId": "po-2"}, {"order": order("po-2", "acme", "submitted", "sam", 21, [L3])})
    c.ok("setSupplierStatus", "admin", T0, {"supplierId": "acme", "status": "active"}, {"supplier": supplier("acme", "Acme Supply")})
    # The refused createOrder consumed no number: the next order is po-4.
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, {"order": order("po-4", "acme", "draft", "sam", 0, [])}, new="po-4")
    c.audit_log(T0)
    return c


def b_purchasing_2() -> Case:
    """Approval: state, self approval, the 100000 threshold, the blocked supplier, in the listed order."""
    c = Case("B-purchasing-2", "B-purchasing", "purchasing/approve-order: invalid_state, self_approval, approval_requires_admin, supplier_blocked")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75})
    c.ok("addSupplier", "sam", T0, {"supplierId": "acme", "name": "Acme Supply"})
    P1000 = oline("ply", 1000, 100)
    F1 = oline("foam", 1, 1)
    # po-1: exactly 100000 (1000 x 100), created by sam.
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, new="po-1")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 1000, "unitCostCents": 100})
    c.ok("submitOrder", "sam", T0, {"orderId": "po-1"})
    # po-2: 100001 (adds one unit at 1), created by sam, still draft.
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, new="po-2")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-2", "materialId": "ply", "quantity": 1000, "unitCostCents": 100})
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-2", "materialId": "foam", "quantity": 1, "unitCostCents": 1})
    # po-3: 100001, created by an admin.
    c.ok("createOrder", "admin", T0, {"supplierId": "acme"}, new="po-3")
    c.ok("addOrderLine", "admin", T0, {"orderId": "po-3", "materialId": "ply", "quantity": 1000, "unitCostCents": 100})
    c.ok("addOrderLine", "admin", T0, {"orderId": "po-3", "materialId": "foam", "quantity": 1, "unitCostCents": 1})
    c.ok("submitOrder", "admin", T0, {"orderId": "po-3"})
    # po-4: 150000 (1500 x 100) by sam. po-5: 500 (10 x 50) by sam.
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, new="po-4")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-4", "materialId": "ply", "quantity": 1500, "unitCostCents": 100})
    c.ok("submitOrder", "sam", T0, {"orderId": "po-4"})
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, new="po-5")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-5", "materialId": "foam", "quantity": 10, "unitCostCents": 50})
    c.ok("submitOrder", "sam", T0, {"orderId": "po-5"})
    # po-1: nobody approves an order they created; exactly 100000 may be approved by staff.
    rf(c, "approveOrder", "sam", T0, {"orderId": "po-1"}, "self_approval")
    rf(c, "approveOrder", "ada", T0, {"orderId": "po-1"}, "forbidden")
    rf(c, "approveOrder", "sue", T0, {"orderId": "ghost"}, "not_found")
    c.ok("approveOrder", "sue", T0, {"orderId": "po-1"}, {"order": order("po-1", "acme", "approved", "sam", 100000, [P1000])})
    rf(c, "approveOrder", "amy", T0, {"orderId": "po-1"}, "invalid_state")
    # po-2: a draft first (invalid_state), then submitted: self approval before the admin rule, then the admin rule.
    rf(c, "approveOrder", "amy", T0, {"orderId": "po-2"}, "invalid_state")
    c.ok("submitOrder", "sam", T0, {"orderId": "po-2"})
    rf(c, "approveOrder", "sam", T0, {"orderId": "po-2"}, "self_approval")  # staff creator and over 100000: self approval first
    rf(c, "approveOrder", "sue", T0, {"orderId": "po-2"}, "approval_requires_admin")  # 100001 > 100000
    c.ok("approveOrder", "amy", T0, {"orderId": "po-2"}, {"order": order("po-2", "acme", "approved", "sam", 100001, [P1000, F1])})
    # po-3 was created by the admin, who cannot approve it either; another admin can.
    rf(c, "approveOrder", "admin", T0, {"orderId": "po-3"}, "self_approval")
    c.ok("approveOrder", "amy", T0, {"orderId": "po-3"}, {"order": order("po-3", "acme", "approved", "admin", 100001, [P1000, F1])})
    # A blocked supplier blocks approval whoever acts, but it is the last rule.
    c.ok("setSupplierStatus", "admin", T0, {"supplierId": "acme", "status": "blocked"})
    rf(c, "approveOrder", "sam", T0, {"orderId": "po-4"}, "self_approval")
    rf(c, "approveOrder", "sue", T0, {"orderId": "po-4"}, "approval_requires_admin")
    rf(c, "approveOrder", "amy", T0, {"orderId": "po-4"}, "supplier_blocked")
    rf(c, "approveOrder", "sue", T0, {"orderId": "po-5"}, "supplier_blocked")
    rf(c, "approveOrder", "amy", T0, {"orderId": "po-5"}, "supplier_blocked")
    # Orders in other steps are not affected by the block.
    c.ok("listOrders", "sam", T0, {"supplierId": "acme"}, {"supplierId": "acme", "orders": [
        order("po-1", "acme", "approved", "sam", 100000, [P1000]),
        order("po-2", "acme", "approved", "sam", 100001, [P1000, F1]),
        order("po-3", "acme", "approved", "admin", 100001, [P1000, F1]),
        order("po-4", "acme", "submitted", "sam", 150000, [oline("ply", 1500, 100)]),
        order("po-5", "acme", "submitted", "sam", 500, [oline("foam", 10, 50)])]})
    c.ok("setSupplierStatus", "admin", T0, {"supplierId": "acme", "status": "active"})
    c.ok("approveOrder", "sue", T0, {"orderId": "po-5"}, {"order": order("po-5", "acme", "approved", "sam", 500, [oline("foam", 10, 50)])})
    c.ok("approveOrder", "admin", T0, {"orderId": "po-4"}, {"order": order("po-4", "acme", "approved", "sam", 150000, [oline("ply", 1500, 100)])})
    c.audit_log(T0)
    return c


def b_purchasing_3() -> Case:
    """Receiving: partial receipts, over receipt, status, stock, price unchanged, journal, blocked supplier not judged."""
    c = Case("B-purchasing-3", "B-purchasing", "purchasing/receiving, receive-order: partial receipt, over_receipt, received, stock, postings")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75})
    c.ok("addMaterial", "sam", T0, {"materialId": "glue", "name": "Glue", "unitPriceCents": 250})
    c.ok("addSupplier", "sam", T0, {"supplierId": "acme", "name": "Acme Supply"})
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, new="po-1")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 100, "unitCostCents": 120})  # 12000
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "foam", "quantity": 20, "unitCostCents": 40})  # + 800 = 12800
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, new="po-2")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-2", "materialId": "ply", "quantity": 5, "unitCostCents": 10})
    # A draft cannot receive, whatever the material (state before the line).
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-2", "materialId": "ply", "quantity": 1}, "invalid_state")
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-2", "materialId": "glue", "quantity": 1}, "invalid_state")
    c.ok("submitOrder", "sam", T0, {"orderId": "po-1"})
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 1}, "invalid_state")  # submitted
    c.ok("approveOrder", "amy", T0, {"orderId": "po-1"})
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-1", "materialId": "glue", "quantity": 1}, "not_on_order")
    rf(c, "receiveOrder", "sam", T0, {"orderId": "ghost", "materialId": "ply", "quantity": 1}, "not_found")
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-1", "materialId": "ghost", "quantity": 1}, "not_found")
    rf(c, "receiveOrder", "ada", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 1}, "forbidden")
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 0}, "invalid_argument")
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 100001}, "invalid_argument")
    t = "2026-03-03T09:00Z"
    # 40 of 100 at 120: stock 100 + 40, posts 40 x 120 = 4800.
    c.ok("receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "ply", "quantity": 40},
         {"order": order("po-1", "acme", "approved", "sam", 12800, [oline("ply", 100, 120, 40), oline("foam", 20, 40)])})
    rf(c, "receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "ply", "quantity": 61}, "over_receipt")  # 40 + 61 > 100
    c.ok("listMaterials", "sam", t, {}, {"materials": [mat("foam", "Foam board", 75, 0), mat("glue", "Glue", 250, 0), mat("ply", "Plywood", 150, 140)]})
    # 60 more completes the line (stock 200, posts 60 x 120 = 7200), but the order stays approved: foam is open.
    c.ok("receiveOrder", "admin", t, {"orderId": "po-1", "materialId": "ply", "quantity": 60},
         {"order": order("po-1", "acme", "approved", "sam", 12800, [oline("ply", 100, 120, 100), oline("foam", 20, 40)])})
    rf(c, "receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "ply", "quantity": 1}, "over_receipt")
    # The last line received: the order is received (posts 20 x 40 = 800).
    c.ok("receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "foam", "quantity": 20},
         {"order": order("po-1", "acme", "received", "sam", 12800, [oline("ply", 100, 120, 100), oline("foam", 20, 40, 20)])})
    rf(c, "receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "foam", "quantity": 1}, "invalid_state")
    rf(c, "receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "glue", "quantity": 1}, "invalid_state")  # state before line
    # The supplier's status is not judged at a receipt: block acme after the approval of po-3.
    c.ok("createOrder", "sam", t, {"supplierId": "acme"}, new="po-3")
    c.ok("addOrderLine", "sam", t, {"orderId": "po-3", "materialId": "foam", "quantity": 5, "unitCostCents": 10})
    c.ok("submitOrder", "sam", t, {"orderId": "po-3"})
    c.ok("approveOrder", "sue", t, {"orderId": "po-3"})
    c.ok("setSupplierStatus", "admin", t, {"supplierId": "acme", "status": "blocked"})
    c.ok("receiveOrder", "sam", t, {"orderId": "po-3", "materialId": "foam", "quantity": 5},
         {"order": order("po-3", "acme", "received", "sam", 50, [oline("foam", 5, 10, 5)])})  # posts 5 x 10 = 50
    # Prices of the shop did not change; stock is the sum of receipts.
    c.ok("listMaterials", "bob", t, {}, {"materials": [mat("foam", "Foam board", 75, 25), mat("glue", "Glue", 250, 0), mat("ply", "Plywood", 150, 200)]})
    c.ok("listJournalEntries", "admin", t, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, t, "inventory", "payable", 4800, "po-1"), jrn(2, t, "inventory", "payable", 7200, "po-1"),
        jrn(3, t, "inventory", "payable", 800, "po-1"), jrn(4, t, "inventory", "payable", 50, "po-3")]})
    c.audit_log(t)
    return c


# --- stocktake ---------------------------------------------------------------------------------------------------------

def b_stocktake_1() -> Case:
    c = Case("B-stocktake-1", "B-stocktake", "stocktake: start, count, replace, order of counts, close, shrink and gain, refusals")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75})
    c.ok("restockMaterial", "sam", T0, {"materialId": "foam", "quantity": 20})
    c.ok("addMaterial", "sam", T0, {"materialId": "glue", "name": "Glue", "unitPriceCents": 250})
    rf(c, "getStocktake", "sam", T0, {"stocktakeId": "stk-1"}, "not_found")
    rf(c, "startStocktake", "bob", T0, {}, "forbidden")
    c.ok("startStocktake", "sam", T0, {}, {"stocktake": stk("stk-1", "open", [])}, new="stk-1")
    rf(c, "startStocktake", "sam", T0, {}, "stocktake_running")
    rf(c, "startStocktake", "admin", T0, {}, "stocktake_running")
    rf(c, "closeStocktake", "sam", T0, {"stocktakeId": "stk-1"}, "nothing_counted")
    rf(c, "closeStocktake", "sam", T0, {"stocktakeId": "ghost"}, "not_found")
    rf(c, "closeStocktake", "bob", T0, {"stocktakeId": "stk-1"}, "forbidden")
    rf(c, "recordCount", "sam", T0, {"stocktakeId": "ghost", "materialId": "ply", "counted": 1}, "not_found")
    rf(c, "recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "ghost", "counted": 1}, "not_found")
    rf(c, "recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "ply", "counted": 100000001}, "invalid_argument")
    rf(c, "recordCount", "bob", T0, {"stocktakeId": "stk-1", "materialId": "ply", "counted": 1}, "forbidden")
    c.ok("recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "ply", "counted": 90},
         {"stocktake": stk("stk-1", "open", [cnt("ply", 90, None)])})
    # Counts are listed by materialId, whatever the order they were recorded in.
    c.ok("recordCount", "sue", T0, {"stocktakeId": "stk-1", "materialId": "foam", "counted": 25},
         {"stocktake": stk("stk-1", "open", [cnt("foam", 25, None), cnt("ply", 90, None)])})
    # A new count of a material replaces its old one: still one count per material.
    c.ok("recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "ply", "counted": 95},
         {"stocktake": stk("stk-1", "open", [cnt("foam", 25, None), cnt("ply", 95, None)])})
    c.ok("recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "glue", "counted": 0},
         {"stocktake": stk("stk-1", "open", [cnt("foam", 25, None), cnt("glue", 0, None), cnt("ply", 95, None)])})
    c.ok("getStocktake", "admin", T0, {"stocktakeId": "stk-1"},
         {"stocktake": stk("stk-1", "open", [cnt("foam", 25, None), cnt("glue", 0, None), cnt("ply", 95, None)])})
    rf(c, "getStocktake", "bob", T0, {"stocktakeId": "stk-1"}, "forbidden")
    # The stock does not change before the close.
    c.ok("listMaterials", "sam", T0, {}, {"materials": [mat("foam", "Foam board", 75, 20), mat("glue", "Glue", 250, 0), mat("ply", "Plywood", 150, 100)]})
    t = "2026-03-02T12:00Z"
    # foam: 25 - 20 = +5 (gain 5 x 75 = 375); glue: 0 - 0 = 0 (nothing posted); ply: 95 - 100 = -5 (shrink 5 x 150 = 750).
    done = stk("stk-1", "closed", [cnt("foam", 25, 5), cnt("glue", 0, 0), cnt("ply", 95, -5)])
    c.ok("closeStocktake", "sam", t, {"stocktakeId": "stk-1"}, {"stocktake": done})
    rf(c, "closeStocktake", "sam", t, {"stocktakeId": "stk-1"}, "invalid_state")
    rf(c, "recordCount", "sam", t, {"stocktakeId": "stk-1", "materialId": "ply", "counted": 1}, "invalid_state")
    c.ok("getStocktake", "sam", t, {"stocktakeId": "stk-1"}, {"stocktake": done})
    c.ok("listMaterials", "sam", t, {}, {"materials": [mat("foam", "Foam board", 75, 25), mat("glue", "Glue", 250, 0), mat("ply", "Plywood", 150, 95)]})
    c.ok("listJournalEntries", "admin", t, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, t, "inventory", "expenseWriteOff", 375, "stk-1"), jrn(2, t, "expenseWriteOff", "inventory", 750, "stk-1")]})
    # The number of a refused start was not consumed; after the close a new stocktake can start.
    c.ok("startStocktake", "sam", t, {}, {"stocktake": stk("stk-2", "open", [])}, new="stk-2")
    c.audit_log(t)
    return c


def b_stocktake_2() -> Case:
    """Closing arithmetic over two stocktakes; a closed stocktake and an open one without the count freeze nothing."""
    c = Case("B-stocktake-2", "B-stocktake", "stocktake/close-stocktake, stocktake-freeze: difference against the stock at closing, counted 0, zero, large")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75})
    c.ok("restockMaterial", "sam", T0, {"materialId": "foam", "quantity": 8})
    c.ok("addMaterial", "sam", T0, {"materialId": "glue", "name": "Glue", "unitPriceCents": 1000000})
    c.ok("startStocktake", "sam", T0, {}, new="stk-1")
    c.ok("recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "ply", "counted": 0})
    # Counted 0 against 100: difference -100, the shrink is 100 x 150 = 15000, the stock becomes 0.
    c.ok("closeStocktake", "sam", T0, {"stocktakeId": "stk-1"}, {"stocktake": stk("stk-1", "closed", [cnt("ply", 0, -100)])})
    # A closed stocktake freezes nothing: the refusal is the stock, not the freeze.
    rf(c, "purchaseMaterial", "ada", T0, {"memberId": "ada", "materialId": "ply", "quantity": 1}, "insufficient_stock")
    c.ok("restockMaterial", "sam", T0, {"materialId": "ply", "quantity": 40}, {"material": mat("ply", "Plywood", 150, 40)})
    t = "2026-03-02T10:00Z"
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "ply", "quantity": 10},
         bought("pur-1", "ada", "ply", 10, 150, 0, 1500, t, "Plywood", 30), new="pur-1")
    c.ok("startStocktake", "sam", t, {}, {"stocktake": stk("stk-2", "open", [])}, new="stk-2")
    c.ok("recordCount", "sam", t, {"stocktakeId": "stk-2", "materialId": "foam", "counted": 8})
    c.ok("recordCount", "sam", t, {"stocktakeId": "stk-2", "materialId": "glue", "counted": 100000000})
    # ply has no count yet: an open stocktake alone does not freeze it.
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "ply", "quantity": 1},
         bought("pur-2", "ada", "ply", 1, 150, 0, 150, t, "Plywood", 29), new="pur-2")
    c.ok("recordCount", "sam", t, {"stocktakeId": "stk-2", "materialId": "ply", "counted": 33})
    rf(c, "purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "ply", "quantity": 1}, "stocktake_open")
    # foam 8 - 8 = 0; glue 100000000 - 0 = +100000000 (x 1000000 = 100000000000000); ply 33 - 29 = +4 (x 150 = 600).
    c.ok("closeStocktake", "admin", t, {"stocktakeId": "stk-2"},
         {"stocktake": stk("stk-2", "closed", [cnt("foam", 8, 0), cnt("glue", 100000000, 100000000), cnt("ply", 33, 4)])})
    c.ok("listJournalEntries", "admin", t, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, T0, "expenseWriteOff", "inventory", 15000, "stk-1"),
        jrn(2, t, "inventory", "expenseWriteOff", 100000000000000, "stk-2"),
        jrn(3, t, "inventory", "expenseWriteOff", 600, "stk-2")]})
    c.ok("listMaterials", "admin", t, {}, {"materials": [mat("foam", "Foam board", 75, 8), mat("glue", "Glue", 1000000, 100000000), mat("ply", "Plywood", 150, 33)]})
    c.audit_log(t)
    return c


# --- projects ----------------------------------------------------------------------------------------------------------

def b_projects_1() -> Case:
    c = Case("B-projects-1", "B-projects", "projects: create, collaborators, close, ownership judged on the project, read cost")
    world(c)
    c.ok("createProject", "ada", T0, {"projectId": "p1", "ownerId": "ada", "title": "Lamp"}, {"project": proj("p1", "ada", "Lamp")})
    rf(c, "createProject", "bob", T0, {"projectId": "p9", "ownerId": "ada", "title": "Mine now"}, "forbidden")
    rf(c, "createProject", "ada", T0, {"projectId": "p1", "ownerId": "ada", "title": "Again"}, "already_exists")
    rf(c, "createProject", "bob", T0, {"projectId": "p9", "ownerId": "ghost", "title": "Nobody"}, "not_found")
    rf(c, "createProject", "sam", T0, {"projectId": "p9", "ownerId": "sue", "title": "Staff owner"}, "not_found")  # staff is no member
    rf(c, "createProject", "ada", T0, {"projectId": "p9", "ownerId": "ada", "title": "   "}, "invalid_argument")
    rf(c, "createProject", "ada", T0, {"projectId": "bad id", "ownerId": "ada", "title": "X"}, "invalid_argument")
    c.ok("createProject", "sam", T0, {"projectId": "p2", "ownerId": "bob", "title": "Shelf"}, {"project": proj("p2", "bob", "Shelf")})
    # Collaborators stay in ascending order whatever the order they were added in.
    c.ok("addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "eve"}, {"project": proj("p1", "ada", "Lamp", ["eve"])})
    c.ok("addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "bob"}, {"project": proj("p1", "ada", "Lamp", ["bob", "eve"])})
    rf(c, "addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "ada"}, "already_collaborator")  # the owner
    rf(c, "addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "eve"}, "already_collaborator")
    rf(c, "addCollaborator", "ada", T0, {"projectId": "ghost", "memberId": "eve"}, "not_found")
    rf(c, "addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "ghost"}, "not_found")
    rf(c, "addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "sue"}, "not_found")
    # A collaborator is not the owner; ownership is judged on the project alone, also when the member added is the actor.
    rf(c, "addCollaborator", "eve", T0, {"projectId": "p1", "memberId": "ada"}, "forbidden")
    rf(c, "addCollaborator", "bob", T0, {"projectId": "p1", "memberId": "bob"}, "forbidden")
    rf(c, "addCollaborator", "ada", T0, {"projectId": "p2", "memberId": "ada"}, "forbidden")
    c.ok("addCollaborator", "sam", T0, {"projectId": "p2", "memberId": "ada"}, {"project": proj("p2", "bob", "Shelf", ["ada"])})
    # Close: owner or staff; final.
    rf(c, "closeProject", "eve", T0, {"projectId": "p1"}, "forbidden")
    c.ok("closeProject", "ada", T0, {"projectId": "p1"}, {"project": proj("p1", "ada", "Lamp", ["bob", "eve"], "closed")})
    rf(c, "closeProject", "ada", T0, {"projectId": "p1"}, "invalid_state")
    rf(c, "closeProject", "sam", T0, {"projectId": "p1"}, "invalid_state")
    # A closed project takes no collaborator; that is judged before the duplicate rule.
    rf(c, "addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "ada"}, "project_closed")
    rf(c, "addCollaborator", "sam", T0, {"projectId": "p1", "memberId": "sam"}, "not_found")
    rf(c, "addCollaborator", "sam", T0, {"projectId": "p1", "memberId": "eve"}, "project_closed")
    # A closed project can still be read; a collaborator reads nothing; the owner reads own, staff all.
    zero = cost("p1", 0, 0, [])
    c.ok("getProjectCost", "ada", T0, {"projectId": "p1"}, zero)
    rf(c, "getProjectCost", "eve", T0, {"projectId": "p1"}, "forbidden")
    c.ok("getProjectCost", "bob", T0, {"projectId": "p2"}, cost("p2", 0, 0, []))
    rf(c, "getProjectCost", "ada", T0, {"projectId": "p2"}, "forbidden")
    c.ok("getProjectCost", "sam", T0, {"projectId": "p2"}, cost("p2", 0, 0, []))
    rf(c, "getProjectCost", "sam", T0, {"projectId": "ghost"}, "not_found")
    # The owner's standing is not judged when creating a project.
    c.ok("setMemberStanding", "sam", T0, {"memberId": "eve", "standing": "suspended"})
    c.ok("createProject", "eve", T0, {"projectId": "p3", "ownerId": "eve", "title": "Table"}, {"project": proj("p3", "eve", "Table")})
    c.audit_log(T0)
    return c


def b_projects_2() -> Case:
    """Attachments: the order of refusals, reservations in several statuses, purchases, ownership, the closed project."""
    c = Case("B-projects-2", "B-projects", "projects/attachment: project_closed, not_project_member, already_attached, ownership on the project")
    world(c)
    certify(c, "eve")
    c.ok("addMaterial", "sam", T0, {"materialId": "glue", "name": "Glue", "unitPriceCents": 250})
    c.ok("restockMaterial", "sam", T0, {"materialId": "glue", "quantity": 20})
    d3, d4, d5 = "2026-03-03", "2026-03-04", "2026-03-05"
    c.ok("createReservation", "sam", T0, {"memberId": "ada", "machineId": "L1", "start": f"{d3}T10:00Z", "end": f"{d3}T11:00Z"},
         {"reservation": resv("res-1", "ada", "L1", f"{d3}T10:00Z", f"{d3}T11:00Z")}, new="res-1")
    c.ok("createReservation", "sam", T0, {"memberId": "ada", "machineId": "L2", "start": f"{d3}T14:00Z", "end": f"{d3}T15:00Z"}, new="res-2")
    c.ok("createReservation", "sam", T0, {"memberId": "ada", "machineId": "L1", "start": f"{d4}T10:00Z", "end": f"{d4}T11:00Z"}, new="res-3")
    c.ok("createReservation", "sam", T0, {"memberId": "eve", "machineId": "L1", "start": f"{d3}T12:00Z", "end": f"{d3}T13:00Z"}, new="res-4")
    c.ok("purchaseMaterial", "sam", T0, {"memberId": "ada", "materialId": "ply", "quantity": 4}, new="pur-1")  # 4 x 150 = 600
    c.ok("purchaseMaterial", "sam", T0, {"memberId": "bob", "materialId": "glue", "quantity": 1}, new="pur-2")  # 250
    c.ok("purchaseMaterial", "sam", T0, {"memberId": "eve", "materialId": "glue", "quantity": 2}, new="pur-3")  # 2 x 250 = 500
    c.ok("createProject", "ada", T0, {"projectId": "p1", "ownerId": "ada", "title": "Lamp"})
    c.ok("addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "bob"})
    c.ok("createProject", "eve", T0, {"projectId": "p2", "ownerId": "eve", "title": "Chair"})
    # Reservations are listed in the order they were attached.
    c.ok("attachReservation", "ada", T0, {"projectId": "p1", "reservationId": "res-3"}, {"project": proj("p1", "ada", "Lamp", ["bob"], res=["res-3"])})
    c.ok("attachReservation", "ada", T0, {"projectId": "p1", "reservationId": "res-1"}, {"project": proj("p1", "ada", "Lamp", ["bob"], res=["res-3", "res-1"])})
    rf(c, "attachReservation", "ada", T0, {"projectId": "p1", "reservationId": "res-1"}, "already_attached")
    rf(c, "attachReservation", "ada", T0, {"projectId": "p1", "reservationId": "res-4"}, "not_project_member")
    c.ok("attachReservation", "eve", T0, {"projectId": "p2", "reservationId": "res-4"}, {"project": proj("p2", "eve", "Chair", res=["res-4"])})
    # Not a member comes before already attached, in both directions.
    rf(c, "attachReservation", "ada", T0, {"projectId": "p1", "reservationId": "res-4"}, "not_project_member")
    rf(c, "attachReservation", "sam", T0, {"projectId": "p2", "reservationId": "res-1"}, "not_project_member")
    rf(c, "attachReservation", "ada", T0, {"projectId": "p2", "reservationId": "res-2"}, "forbidden")  # ada does not own p2
    rf(c, "attachReservation", "ada", T0, {"projectId": "p1", "reservationId": "res-99"}, "not_found")
    rf(c, "attachReservation", "ada", T0, {"projectId": "ghost", "reservationId": "res-2"}, "not_found")
    c.ok("attachReservation", "ada", T0, {"projectId": "p1", "reservationId": "res-2"}, {"project": proj("p1", "ada", "Lamp", ["bob"], res=["res-3", "res-1", "res-2"])})
    # Any status can be attached: a cancelled reservation too.
    c.ok("cancelReservation", "ada", T0, {"reservationId": "res-2"}, {"reservation": resv("res-2", "ada", "L2", f"{d3}T14:00Z", f"{d3}T15:00Z", "cancelled")})
    # Purchases, in the order attached; bob is a collaborator, so his purchase can be attached, but only by the owner.
    c.ok("attachPurchase", "ada", T0, {"projectId": "p1", "purchaseId": "pur-2"}, {"project": proj("p1", "ada", "Lamp", ["bob"], res=["res-3", "res-1", "res-2"], purchases=["pur-2"])})
    c.ok("attachPurchase", "sam", T0, {"projectId": "p1", "purchaseId": "pur-1"},
         {"project": proj("p1", "ada", "Lamp", ["bob"], res=["res-3", "res-1", "res-2"], purchases=["pur-2", "pur-1"])})
    rf(c, "attachPurchase", "ada", T0, {"projectId": "p1", "purchaseId": "pur-3"}, "not_project_member")
    rf(c, "attachPurchase", "ada", T0, {"projectId": "p1", "purchaseId": "pur-1"}, "already_attached")
    rf(c, "attachPurchase", "bob", T0, {"projectId": "p1", "purchaseId": "pur-2"}, "forbidden")  # a collaborator acts as no member
    rf(c, "attachPurchase", "ada", T0, {"projectId": "p1", "purchaseId": "pur-99"}, "not_found")
    c.ok("attachPurchase", "eve", T0, {"projectId": "p2", "purchaseId": "pur-3"}, {"project": proj("p2", "eve", "Chair", res=["res-4"], purchases=["pur-3"])})
    rf(c, "attachPurchase", "ada", T0, {"projectId": "p1", "purchaseId": "pur-3"}, "not_project_member")
    rf(c, "attachPurchase", "sam", T0, {"projectId": "p2", "purchaseId": "pur-1"}, "not_project_member")
    # Cost: no reservation is completed, so usage is 0; the shares are by member, ascending: ada then bob.
    c.ok("getProjectCost", "ada", T0, {"projectId": "p1"}, cost("p1", 0, 850, [share("ada", 0, 600), share("bob", 0, 250)]))
    c.ok("getProjectCost", "eve", T0, {"projectId": "p2"}, cost("p2", 0, 500, [share("eve", 0, 500)]))
    # Closing is final: attachments are refused with project_closed before the member rule.
    c.ok("closeProject", "ada", T0, {"projectId": "p1"})
    rf(c, "attachReservation", "ada", T0, {"projectId": "p1", "reservationId": "res-4"}, "project_closed")
    rf(c, "attachPurchase", "sam", T0, {"projectId": "p1", "purchaseId": "pur-3"}, "project_closed")
    c.ok("getProjectCost", "sam", T0, {"projectId": "p1"}, cost("p1", 0, 850, [share("ada", 0, 600), share("bob", 0, 250)]))
    c.audit_log(T0)
    return c


# --- quotes ------------------------------------------------------------------------------------------------------------

def b_quotes_1() -> Case:
    c = Case("B-quotes-1", "B-quotes", "quotes: customers, lines, send pricing, validity, decide, refusals in order")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 236})
    c.ok("addCustomer", "sam", T0, {"customerId": "cu1", "name": "Cora Customer"}, {"customer": {"customerId": "cu1", "name": "Cora Customer"}})
    rf(c, "addCustomer", "admin", T0, {"customerId": "cu1", "name": "Other"}, "already_exists")
    rf(c, "addCustomer", "bob", T0, {"customerId": "cu2", "name": "Bob"}, "forbidden")
    rf(c, "createQuote", "sam", T0, {"customerId": "ghost", "title": "Sign"}, "not_found")
    rf(c, "createQuote", "bob", T0, {"customerId": "cu1", "title": "Sign"}, "forbidden")
    c.ok("createQuote", "sam", T0, {"customerId": "cu1", "title": "Sign"}, {"quote": quote("quo-1", "cu1", "Sign", "draft", [])}, new="quo-1")
    c.ok("createQuote", "sam", T0, {"customerId": "cu1", "title": "Empty"}, {"quote": quote("quo-2", "cu1", "Empty", "draft", [])}, new="quo-2")
    c.ok("createQuote", "sam", T0, {"customerId": "cu1", "title": "Spare"}, new="quo-3")
    rf(c, "sendQuote", "sam", T0, {"quoteId": "quo-2"}, "empty_quote")
    # Lines: unit prices stay null in a draft; the same ref can be on several lines; lines keep their order.
    a = qline("machine", "L1", 20)
    c.ok("addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L1", "quantity": 20}, {"quote": quote("quo-1", "cu1", "Sign", "draft", [a])})
    b = qline("material", "foam", 7)
    c.ok("addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "material", "ref": "foam", "quantity": 7}, {"quote": quote("quo-1", "cu1", "Sign", "draft", [a, b])})
    d = qline("machine", "L1", 10)
    c.ok("addQuoteLine", "admin", T0, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L1", "quantity": 10}, {"quote": quote("quo-1", "cu1", "Sign", "draft", [a, b, d])})
    rf(c, "addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "machine", "ref": "foam", "quantity": 1}, "unknown_line_ref")  # a material id on a machine line
    rf(c, "addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "material", "ref": "L1", "quantity": 1}, "unknown_line_ref")
    rf(c, "addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L9", "quantity": 1}, "unknown_line_ref")  # not not_found
    rf(c, "addQuoteLine", "sam", T0, {"quoteId": "ghost", "lineKind": "machine", "ref": "L1", "quantity": 1}, "not_found")
    rf(c, "addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L1", "quantity": 0}, "invalid_argument")
    rf(c, "addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L1", "quantity": 100001}, "invalid_argument")
    rf(c, "addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "tool", "ref": "L1", "quantity": 1}, "invalid_argument")
    rf(c, "addQuoteLine", "bob", T0, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L1", "quantity": 1}, "forbidden")
    # quo-3: lathe T1 (rate 30), 10 minutes: subtotal 300, total (300 x 120 + 50) div 100 = 36050 div 100 = 360.
    c.ok("addQuoteLine", "sam", T0, {"quoteId": "quo-3", "lineKind": "machine", "ref": "T1", "quantity": 10})
    # Send: prices captured at the request time. subtotal = 20 x 45 + 7 x 236 + 10 x 45 = 900 + 1652 + 450 = 3002.
    # total = (3002 x 120 + 50) div 100 = 360290 div 100 = 3602. 30 days after 2026-03-02 is 2026-04-01.
    t1 = "2026-03-02T15:00Z"
    sent1 = quote("quo-1", "cu1", "Sign", "sent", [qline("machine", "L1", 20, 45), qline("material", "foam", 7, 236), qline("machine", "L1", 10, 45)],
                  3002, 3602, t1, "2026-04-01")
    rf(c, "sendQuote", "bob", t1, {"quoteId": "quo-1"}, "forbidden")
    c.ok("sendQuote", "sam", t1, {"quoteId": "quo-1"}, {"quote": sent1})
    rf(c, "sendQuote", "sam", t1, {"quoteId": "quo-1"}, "invalid_state")
    rf(c, "addQuoteLine", "sam", t1, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L9", "quantity": 1}, "invalid_state")  # state before the ref
    c.ok("getQuote", "admin", t1, {"quoteId": "quo-1"}, {"quote": sent1})
    rf(c, "getQuote", "bob", t1, {"quoteId": "quo-1"}, "forbidden")
    rf(c, "getQuote", "sam", t1, {"quoteId": "ghost"}, "not_found")
    t2 = "2026-03-05T09:00Z"
    sent3 = quote("quo-3", "cu1", "Spare", "sent", [qline("machine", "T1", 10, 30)], 300, 360, t2, "2026-04-04")
    c.ok("sendQuote", "sam", t2, {"quoteId": "quo-3"}, {"quote": sent3})
    # The last valid day of quo-1 is 2026-04-01.
    t3 = "2026-04-01T23:59Z"
    rf(c, "decideQuote", "sam", t3, {"quoteId": "quo-2", "decision": "accepted"}, "invalid_state")  # a draft
    rf(c, "decideQuote", "sam", t3, {"quoteId": "quo-1", "decision": "approved"}, "invalid_argument")
    rf(c, "decideQuote", "bob", t3, {"quoteId": "quo-1", "decision": "accepted"}, "forbidden")
    accepted = quote("quo-1", "cu1", "Sign", "accepted", sent1["lines"], 3002, 3602, t1, "2026-04-01")
    c.ok("decideQuote", "sam", t3, {"quoteId": "quo-1", "decision": "accepted"}, {"quote": accepted})
    rf(c, "decideQuote", "sam", t3, {"quoteId": "quo-1", "decision": "rejected"}, "invalid_state")
    # A rejection is never refused for age: quo-3 is valid until 2026-04-04 and is rejected on 2026-04-20.
    t4 = "2026-04-20T09:00Z"
    c.ok("decideQuote", "admin", t4, {"quoteId": "quo-3", "decision": "rejected"}, {"quote": quote("quo-3", "cu1", "Spare", "rejected", sent3["lines"], 300, 360, t2, "2026-04-04")})
    c.ok("getQuote", "sam", t4, {"quoteId": "quo-2"}, {"quote": quote("quo-2", "cu1", "Empty", "draft", [])})
    c.audit_log(t4)
    return c


def b_quotes_2() -> Case:
    """Pricing rounded once, half up; the validity date across month, year, leap day and the end of the calendar."""
    c = Case("B-quotes-2", "B-quotes", "quotes/quote-pricing, quote-validity: surcharge rounded once, last valid day, day after, 30-day dates")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "pin", "name": "Pin", "unitPriceCents": 1})
    c.ok("addCustomer", "sam", T0, {"customerId": "cu1", "name": "Cora Customer"})
    # subtotal -> total: 3 -> (3 x 120 + 50) div 100 = 410 div 100 = 4; 2 -> 290 div 100 = 2; 4 -> 530 div 100 = 5;
    # 1 -> 170 div 100 = 1; 5 -> 650 div 100 = 6.
    for n, (title, qty) in enumerate((("Three", 3), ("Two", 2), ("Four", 4), ("One", 1), ("Five", 5)), start=1):
        c.ok("createQuote", "sam", T0, {"customerId": "cu1", "title": title}, new=f"quo-{n}")
        c.ok("addQuoteLine", "sam", T0, {"quoteId": f"quo-{n}", "lineKind": "material", "ref": "pin", "quantity": qty})

    def sent(n: int, title: str, qty: int, total: int, at: str, valid: str, status: str = "sent") -> dict[str, Any]:
        return {"quote": quote(f"quo-{n}", "cu1", title, status, [qline("material", "pin", qty, 1)], qty, total, at, valid)}

    # 2026-12-15 + 30 days = 2027-01-14.
    t1 = "2026-12-15T10:00Z"
    c.ok("sendQuote", "sam", t1, {"quoteId": "quo-1"}, sent(1, "Three", 3, 4, t1, "2027-01-14"))
    t2 = "2027-01-14T23:59Z"  # the last valid day
    c.ok("decideQuote", "sam", t2, {"quoteId": "quo-1", "decision": "accepted"}, sent(1, "Three", 3, 4, t1, "2027-01-14", "accepted"))
    # 2027-02-10 + 30 days = 2027-03-12 (February has 28 days).
    t3 = "2027-02-10T12:00Z"
    c.ok("sendQuote", "sam", t3, {"quoteId": "quo-4"}, sent(4, "One", 1, 1, t3, "2027-03-12"))
    c.ok("sendQuote", "sam", t3, {"quoteId": "quo-5"}, sent(5, "Five", 5, 6, t3, "2027-03-12"))
    t4 = "2027-03-12T23:59Z"
    c.ok("decideQuote", "sam", t4, {"quoteId": "quo-5", "decision": "accepted"}, sent(5, "Five", 5, 6, t3, "2027-03-12", "accepted"))
    # The day after the last valid day: expired. The quote stays sent; a rejection is still possible.
    t5 = "2027-03-13T00:00Z"
    rf(c, "decideQuote", "sam", t5, {"quoteId": "quo-4", "decision": "accepted"}, "quote_expired")
    c.ok("getQuote", "sam", t5, {"quoteId": "quo-4"}, sent(4, "One", 1, 1, t3, "2027-03-12"))
    c.ok("decideQuote", "sam", t5, {"quoteId": "quo-4", "decision": "rejected"}, sent(4, "One", 1, 1, t3, "2027-03-12", "rejected"))
    # 2028-02-01 + 30 days = 2028-03-02 (leap year: February has 29 days). Subtotal 2 gives total 2.
    t6 = "2028-02-01T08:00Z"
    c.ok("sendQuote", "sam", t6, {"quoteId": "quo-2"}, sent(2, "Two", 2, 2, t6, "2028-03-02"))
    t7 = "2028-03-02T23:59Z"
    c.ok("decideQuote", "sam", t7, {"quoteId": "quo-2", "decision": "accepted"}, sent(2, "Two", 2, 2, t6, "2028-03-02", "accepted"))
    # At the end of the calendar the validity lies in 2100: 2099-12-31 + 30 days = 2100-01-30. Subtotal 4 gives total 5.
    t8 = "2099-12-31T23:59Z"
    c.ok("sendQuote", "sam", t8, {"quoteId": "quo-3"}, sent(3, "Four", 4, 5, t8, "2100-01-30"))
    c.audit_log(t8)
    return c


# --- jobs --------------------------------------------------------------------------------------------------------------

def b_jobs_1() -> Case:
    """Start job: the order of refusals and the maker gate."""
    c = Case("B-jobs-1", "B-jobs", "jobs/start-job, maker-gate: quote_not_accepted, already_started, member_suspended, maker_not_certified")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "pin", "name": "Pin", "unitPriceCents": 1})
    c.ok("addCustomer", "sam", T0, {"customerId": "cu1", "name": "Cora Customer"})
    make_quote(c, "cu1", "Q1", [("machine", "L1", 10), ("machine", "L2", 5)], T0, "accepted")  # quo-1, two laser machines
    make_quote(c, "cu1", "Q2", [("machine", "T1", 5)], T0, "accepted")  # quo-2, a lathe
    make_quote(c, "cu1", "Q3", [("material", "pin", 4)], T0, "accepted")  # quo-3, materials only; total 5
    make_quote(c, "cu1", "Q4", [], T0, send=False)  # quo-4, a draft
    make_quote(c, "cu1", "Q5", [("material", "pin", 1)], T0, "rejected")  # quo-5
    make_quote(c, "cu1", "Q6", [("material", "pin", 1)], T0)  # quo-6, sent
    make_quote(c, "cu1", "Q7", [("machine", "L1", 1)], T0, "accepted")  # quo-7
    make_quote(c, "cu1", "Q8", [("machine", "L1", 1)], T0, "accepted")  # quo-8
    for q in ("quo-4", "quo-5", "quo-6"):
        rf(c, "startJob", "sam", T0, {"quoteId": q, "makerId": "ada"}, "quote_not_accepted")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "bob", "standing": "suspended"})
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-4", "makerId": "bob"}, "quote_not_accepted")  # before the standing
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-1", "makerId": "bob"}, "member_suspended")  # before the certification
    c.ok("setMemberStanding", "sam", T0, {"memberId": "bob", "standing": "good"})
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-1", "makerId": "bob"}, "maker_not_certified")
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-2", "makerId": "ada"}, "maker_not_certified")  # ada is certified on laser, not lathe
    # Materials only: no certification needed. 4 pins: subtotal 4, total (4 x 120 + 50) div 100 = 5.
    c.ok("startJob", "sam", T0, {"quoteId": "quo-3", "makerId": "bob"},
         {"job": job("job-1", "quo-3", "cu1", "bob", "started", 5, None, 0)}, new="job-1")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "eve", "standing": "suspended"})
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-3", "makerId": "eve"}, "already_started")  # before the standing
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-3", "makerId": "ada"}, "already_started")
    # Two machines of one family: the certification is needed once. Subtotal 10 x 45 + 5 x 60 = 750, total (750 x 120 + 50) div 100 = 900.
    c.ok("startJob", "admin", T0, {"quoteId": "quo-1", "makerId": "ada"},
         {"job": job("job-2", "quo-1", "cu1", "ada", "started", 900, None, 0)}, new="job-2")
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-1", "makerId": "ada"}, "already_started")
    rf(c, "startJob", "sam", T0, {"quoteId": "ghost", "makerId": "ada"}, "not_found")
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-7", "makerId": "ghost"}, "not_found")
    rf(c, "startJob", "sam", T0, {"quoteId": "quo-7", "makerId": "sue"}, "not_found")  # staff is no member
    rf(c, "startJob", "ada", T0, {"quoteId": "quo-7", "makerId": "ada"}, "forbidden")
    # The certification is judged by date: ada's expires on 2027-03-02. Quote 7: 1 minute at 45, total (45 x 120 + 50) div 100 = 54.
    t = "2027-03-01T23:59Z"
    c.ok("startJob", "sam", t, {"quoteId": "quo-7", "makerId": "ada"},
         {"job": job("job-3", "quo-7", "cu1", "ada", "started", 54, None, 0)}, new="job-3")
    t = "2027-03-02T00:00Z"
    rf(c, "startJob", "sam", t, {"quoteId": "quo-8", "makerId": "ada"}, "maker_not_certified")
    c.ok("listJobs", "sam", t, {"customerId": "cu1"}, {"customerId": "cu1", "jobs": [
        job("job-1", "quo-3", "cu1", "bob", "started", 5, None, 0), job("job-2", "quo-1", "cu1", "ada", "started", 900, None, 0),
        job("job-3", "quo-7", "cu1", "ada", "started", 54, None, 0)]})
    c.audit_log(t)
    return c


def b_jobs_2() -> Case:
    """Complete job, maker credit rounding, payments up to the total, list jobs, ledger entries."""
    c = Case("B-jobs-2", "B-jobs", "jobs/maker-credit, complete-job, record-job-payment, list-jobs")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 236})
    c.ok("addMaterial", "sam", T0, {"materialId": "pin", "name": "Pin", "unitPriceCents": 1})
    for cid in ("cu1", "cu2", "cu3"):
        c.ok("addCustomer", "sam", T0, {"customerId": cid, "name": f"Customer {cid}"})
    make_quote(c, "cu1", "Big", [("machine", "L1", 30), ("material", "foam", 7)], T0, "accepted")  # quo-1: 3602
    make_quote(c, "cu1", "Small", [("material", "pin", 4)], T0, "accepted")  # quo-2: 5
    make_quote(c, "cu2", "Tiny", [("material", "pin", 1)], T0, "accepted")  # quo-3: 1
    c.ok("startJob", "sam", T0, {"quoteId": "quo-1", "makerId": "ada"}, {"job": job("job-1", "quo-1", "cu1", "ada", "started", 3602, None, 0)}, new="job-1")
    c.ok("startJob", "sam", T0, {"quoteId": "quo-2", "makerId": "bob"}, {"job": job("job-2", "quo-2", "cu1", "bob", "started", 5, None, 0)}, new="job-2")
    c.ok("startJob", "sam", T0, {"quoteId": "quo-3", "makerId": "eve"}, {"job": job("job-3", "quo-3", "cu2", "eve", "started", 1, None, 0)}, new="job-3")
    # A payment needs a completed job, whatever the amount.
    rf(c, "recordJobPayment", "sam", T0, {"jobId": "job-2", "amountCents": 1}, "job_not_completed")
    rf(c, "recordJobPayment", "sam", T0, {"jobId": "job-2", "amountCents": 99999}, "job_not_completed")
    rf(c, "completeJob", "ada", T0, {"jobId": "job-1"}, "forbidden")
    rf(c, "completeJob", "sam", T0, {"jobId": "ghost"}, "not_found")
    t = "2026-03-10T09:00Z"
    # Maker credit: (3602 x 30 + 50) div 100 = 108110 div 100 = 1081.
    c.ok("completeJob", "sam", t, {"jobId": "job-1"}, {"job": job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 0)})
    rf(c, "completeJob", "sam", t, {"jobId": "job-1"}, "invalid_state")
    c.ok("getAccount", "ada", t, {"memberId": "ada"}, account("ada", 1081, 0, 0, 5000))
    # Payments: up to the total, exactly the total accepted, then nothing more.
    c.ok("recordJobPayment", "sam", t, {"jobId": "job-1", "amountCents": 3000}, {"job": job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 3000)})
    rf(c, "recordJobPayment", "sam", t, {"jobId": "job-1", "amountCents": 603}, "overpayment")  # 3000 + 603 = 3603 > 3602
    rf(c, "recordJobPayment", "sam", t, {"jobId": "job-1", "amountCents": 100000000}, "overpayment")
    rf(c, "recordJobPayment", "sam", t, {"jobId": "job-1", "amountCents": 0}, "invalid_argument")
    rf(c, "recordJobPayment", "sam", t, {"jobId": "job-1", "amountCents": 100000001}, "invalid_argument")
    rf(c, "recordJobPayment", "ada", t, {"jobId": "job-1", "amountCents": 1}, "forbidden")
    rf(c, "recordJobPayment", "sam", t, {"jobId": "ghost", "amountCents": 1}, "not_found")
    c.ok("recordJobPayment", "admin", t, {"jobId": "job-1", "amountCents": 602}, {"job": job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 3602)})
    rf(c, "recordJobPayment", "sam", t, {"jobId": "job-1", "amountCents": 1}, "overpayment")  # fully paid
    # job-2: total 5, credit (5 x 30 + 50) div 100 = 200 div 100 = 2 (1.5 rounds up). A suspended maker can complete.
    c.ok("setMemberStanding", "sam", t, {"memberId": "bob", "standing": "suspended"})
    c.ok("completeJob", "sam", t, {"jobId": "job-2"}, {"job": job("job-2", "quo-2", "cu1", "bob", "completed", 5, 2, 0)})
    # job-3: total 1, credit (1 x 30 + 50) div 100 = 80 div 100 = 0: no balance change and no entry, but the job is completed.
    c.ok("completeJob", "sam", t, {"jobId": "job-3"}, {"job": job("job-3", "quo-3", "cu2", "eve", "completed", 1, 0, 0)})
    c.ok("recordJobPayment", "sam", t, {"jobId": "job-2", "amountCents": 5}, {"job": job("job-2", "quo-2", "cu1", "bob", "completed", 5, 2, 5)})
    c.ok("recordJobPayment", "sam", t, {"jobId": "job-3", "amountCents": 1}, {"job": job("job-3", "quo-3", "cu2", "eve", "completed", 1, 0, 1)})
    c.ok("getAccount", "bob", t, {"memberId": "bob"}, account("bob", 2, 0, 0, 0))
    c.ok("getAccount", "eve", t, {"memberId": "eve"}, account("eve", 0, 0, 0, 0))
    c.ok("listJobs", "sam", t, {"customerId": "cu1"}, {"customerId": "cu1", "jobs": [
        job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 3602), job("job-2", "quo-2", "cu1", "bob", "completed", 5, 2, 5)]})
    c.ok("listJobs", "admin", t, {"customerId": "cu2"}, {"customerId": "cu2", "jobs": [job("job-3", "quo-3", "cu2", "eve", "completed", 1, 0, 1)]})
    c.ok("listJobs", "admin", t, {"customerId": "cu3"}, {"customerId": "cu3", "jobs": []})
    rf(c, "listJobs", "bob", t, {"customerId": "cu1"}, "forbidden")
    rf(c, "listJobs", "sam", t, {"customerId": "ghost"}, "not_found")
    c.ok("listJournalEntries", "admin", t, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, t, "expenseMaker", "creditLiability", 1081, "job-1"), jrn(2, t, "cash", "revenueJobs", 3000, "job-1"),
        jrn(3, t, "cash", "revenueJobs", 602, "job-1"), jrn(4, t, "expenseMaker", "creditLiability", 2, "job-2"),
        jrn(5, t, "cash", "revenueJobs", 5, "job-2"), jrn(6, t, "cash", "revenueJobs", 1, "job-3")]})
    c.audit_log(t)
    return c


# --- cross-module chains -----------------------------------------------------------------------------------------------

def x_trade_1() -> Case:
    """purchaseMaterial: every refusal in order, removed one cause at a time, then a promotion used once."""
    c = Case("X-trade-1", "X-trade", "materials/purchase-material: member_suspended, hazard_not_certified, hazard_limit_exceeded, stocktake_open, insufficient_stock")
    world(c)
    c.ok("addCourse", "sam", T0, {"courseId": "lathe-101", "familyId": "lathe", "title": "Lathe safety", "feeCents": 2000, "capacity": 5})
    c.ok("addMaterial", "sam", T0, {"materialId": "acid", "name": "Etching acid", "unitPriceCents": 333})
    c.ok("restockMaterial", "sam", T0, {"materialId": "acid", "quantity": 2})
    c.ok("classifyHazard", "admin", T0, {"materialId": "acid", "familyId": "lathe", "monthlyLimit": 5})
    c.ok("addPromotion", "admin", T0, {"code": "M", "percent": 25, "scope": "material", "maxRedemptions": 10, "validUntil": "2026-12-31"})
    c.ok("redeemPromotion", "bob", T0, {"memberId": "bob", "code": "M"})
    c.ok("startStocktake", "sam", T0, {}, new="stk-1")
    c.ok("recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "acid", "counted": 2})
    c.ok("setMemberStanding", "sam", T0, {"memberId": "bob", "standing": "suspended"})
    # The request: bob buys 6 acid. Suspended, not certified, over the limit, counted, short of stock: each in turn.
    buy = {"memberId": "bob", "materialId": "acid", "quantity": 6}
    rf(c, "purchaseMaterial", "bob", T0, buy, "member_suspended")
    c.ok("setMemberStanding", "sam", T0, {"memberId": "bob", "standing": "good"})
    rf(c, "purchaseMaterial", "bob", T0, buy, "hazard_not_certified")
    c.ok("enrollInCourse", "sam", T0, {"courseId": "lathe-101", "memberId": "bob"})
    c.ok("completeCourse", "sam", T0, {"courseId": "lathe-101", "memberId": "bob"})
    rf(c, "purchaseMaterial", "bob", T0, buy, "hazard_limit_exceeded")  # 6 > 5
    c.ok("classifyHazard", "admin", T0, {"materialId": "acid", "familyId": "lathe", "monthlyLimit": 10}, {"classification": cls("acid", "lathe", 10)})
    rf(c, "purchaseMaterial", "bob", T0, buy, "stocktake_open")
    # Closing: counted 2 against the stock 2, difference 0, nothing posted.
    c.ok("closeStocktake", "sam", T0, {"stocktakeId": "stk-1"}, {"stocktake": stk("stk-1", "closed", [cnt("acid", 2, 0)])})
    rf(c, "purchaseMaterial", "bob", T0, buy, "insufficient_stock")  # 2 < 6
    c.ok("restockMaterial", "sam", T0, {"materialId": "acid", "quantity": 10}, {"material": mat("acid", "Etching acid", 333, 12)})
    # None of the refused requests used the promotion: 6 x 333 = 1998, 25 % off: (1998 x 75 + 50) div 100 = 149900 div 100 = 1499.
    c.ok("purchaseMaterial", "bob", T0, buy, bought("pur-1", "bob", "acid", 6, 333, 25, 1499, T0, "Etching acid", 6), new="pur-1")
    c.ok("listMemberPromotions", "bob", T0, {"memberId": "bob"}, {"memberId": "bob", "redemptions": [redemption("M", "bob", 25, T0, T0, "pur-1")]})
    # Used once: the next purchase has no discount. 2 x 333 = 666.
    c.ok("purchaseMaterial", "bob", T0, {"memberId": "bob", "materialId": "acid", "quantity": 2},
         bought("pur-2", "bob", "acid", 2, 333, 0, 666, T0, "Etching acid", 4), new="pur-2")
    rf(c, "purchaseMaterial", "bob", T0, {"memberId": "bob", "materialId": "acid", "quantity": 3}, "hazard_limit_exceeded")  # 6 + 2 + 3 = 11
    # A new month: the count starts again, so 4 more are allowed (4 + 0 <= 10) and take the last of the stock; 4 x 333 = 1332.
    t = "2026-04-01T00:00Z"
    c.ok("purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "acid", "quantity": 4},
         bought("pur-3", "bob", "acid", 4, 333, 0, 1332, t, "Etching acid", 0), new="pur-3")
    rf(c, "purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "acid", "quantity": 1}, "insufficient_stock")
    # The empty stocktake close posted nothing in either month; bob's charges: 2000 (course) + 1499 + 666 + 1332.
    c.ok("listJournalEntries", "admin", t, {"month": "2026-03"}, {"month": "2026-03", "entries": []})
    c.ok("getAccount", "bob", t, {"memberId": "bob"}, account("bob", 0, 0, 0, 5497))
    c.audit_log(t)
    return c


def x_trade_2() -> Case:
    """A stocktake freezes purchase, restock and receipt of a counted material only; consumption still draws; closing."""
    c = Case("X-trade-2", "X-trade", "stocktake/stocktake-freeze, close-stocktake, purchasing/receiving, consumables/consumption")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75})
    c.ok("restockMaterial", "sam", T0, {"materialId": "foam", "quantity": 50})
    c.ok("addMaterial", "sam", T0, {"materialId": "glue", "name": "Glue", "unitPriceCents": 250})
    c.ok("restockMaterial", "sam", T0, {"materialId": "glue", "quantity": 10})
    c.ok("setConsumption", "sam", T0, {"machineId": "L1", "materialId": "foam", "unitsPerHour": 60})
    c.ok("addSupplier", "sam", T0, {"supplierId": "acme", "name": "Acme Supply"})
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, new="po-1")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 10, "unitCostCents": 100})
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "foam", "quantity": 10, "unitCostCents": 50})  # total 1500
    c.ok("submitOrder", "sam", T0, {"orderId": "po-1"})
    c.ok("approveOrder", "amy", T0, {"orderId": "po-1"})
    c.ok("createReservation", "sam", T0, {"memberId": "ada", "machineId": "L1", "start": "2026-03-03T10:00Z", "end": "2026-03-03T11:00Z"}, new="res-1")
    c.ok("startStocktake", "sam", T0, {}, new="stk-1")
    c.ok("recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "ply", "counted": 90})
    c.ok("recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "foam", "counted": 40})
    # ply and foam are counted, hence frozen; the freeze is judged before the stock but after the other rules.
    rf(c, "purchaseMaterial", "ada", T0, {"memberId": "ada", "materialId": "ply", "quantity": 1}, "stocktake_open")
    rf(c, "purchaseMaterial", "bob", T0, {"memberId": "bob", "materialId": "foam", "quantity": 1}, "stocktake_open")
    rf(c, "purchaseMaterial", "bob", T0, {"memberId": "bob", "materialId": "ply", "quantity": 101}, "stocktake_open")  # not insufficient_stock
    rf(c, "restockMaterial", "sam", T0, {"materialId": "ply", "quantity": 5}, "stocktake_open")
    rf(c, "restockMaterial", "sam", T0, {"materialId": "foam", "quantity": 5}, "stocktake_open")
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 5}, "stocktake_open")
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-1", "materialId": "foam", "quantity": 5}, "stocktake_open")
    rf(c, "receiveOrder", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 11}, "over_receipt")  # before the freeze
    # glue has no count: it goes on while the stocktake is open.
    c.ok("purchaseMaterial", "ada", T0, {"memberId": "ada", "materialId": "glue", "quantity": 2},
         bought("pur-1", "ada", "glue", 2, 250, 0, 500, T0, "Glue", 8), new="pur-1")
    c.ok("restockMaterial", "sam", T0, {"materialId": "glue", "quantity": 5}, {"material": mat("glue", "Glue", 250, 13)})
    # Replacing a count keeps the material frozen.
    c.ok("recordCount", "sam", T0, {"stocktakeId": "stk-1", "materialId": "foam", "counted": 45},
         {"stocktake": stk("stk-1", "open", [cnt("foam", 45, None), cnt("ply", 90, None)])})
    rf(c, "restockMaterial", "sam", T0, {"materialId": "foam", "quantity": 5}, "stocktake_open")
    # A session draws foam while it is counted: 30 metered minutes at 60 units an hour give (30 x 60 + 59) div 60 = 30 units.
    t1, t2 = "2026-03-03T10:00Z", "2026-03-03T10:30Z"
    c.ok("checkIn", "ada", t1, {"reservationId": "res-1"})
    c.ok("checkOut", "ada", t2, {"reservationId": "res-1"})
    c.ok("listMaterials", "sam", t2, {}, {"materials": [mat("foam", "Foam board", 75, 20), mat("glue", "Glue", 250, 13), mat("ply", "Plywood", 150, 100)]})
    # Closing compares each count with the stock at that moment: foam 45 - 20 = +25 (x 75 = 1875), ply 90 - 100 = -10 (x 150 = 1500).
    t3 = "2026-03-03T12:00Z"
    c.ok("closeStocktake", "sam", t3, {"stocktakeId": "stk-1"}, {"stocktake": stk("stk-1", "closed", [cnt("foam", 45, 25), cnt("ply", 90, -10)])})
    # The freeze is over for everything.
    c.ok("purchaseMaterial", "ada", t3, {"memberId": "ada", "materialId": "ply", "quantity": 5},
         bought("pur-2", "ada", "ply", 5, 150, 0, 750, t3, "Plywood", 85), new="pur-2")
    c.ok("restockMaterial", "sam", t3, {"materialId": "foam", "quantity": 5}, {"material": mat("foam", "Foam board", 75, 50)})
    # Receipts: foam 10 at 50 posts 500 (the order stays approved), ply 10 at 100 posts 1000 (the order is received).
    c.ok("receiveOrder", "sam", t3, {"orderId": "po-1", "materialId": "foam", "quantity": 10},
         {"order": order("po-1", "acme", "approved", "sam", 1500, [oline("ply", 10, 100), oline("foam", 10, 50, 10)])})
    c.ok("receiveOrder", "sam", t3, {"orderId": "po-1", "materialId": "ply", "quantity": 10},
         {"order": order("po-1", "acme", "received", "sam", 1500, [oline("ply", 10, 100, 10), oline("foam", 10, 50, 10)])})
    c.ok("listMaterials", "bob", t3, {}, {"materials": [mat("foam", "Foam board", 75, 60), mat("glue", "Glue", 250, 13), mat("ply", "Plywood", 150, 95)]})
    c.ok("listJournalEntries", "admin", t3, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, t3, "inventory", "expenseWriteOff", 1875, "stk-1"), jrn(2, t3, "expenseWriteOff", "inventory", 1500, "stk-1"),
        jrn(3, t3, "inventory", "payable", 500, "po-1"), jrn(4, t3, "inventory", "payable", 1000, "po-1")]})
    # ada's account: course 5000 (world), glue 500, usage (30 x 45 x 90 + 50) div 100 = 121550 div 100 = 1215,
    # foam drawn 30 x 75 = 2250 (no discount), ply 750.
    c.ok("getAccount", "ada", t3, {"memberId": "ada"}, account("ada", 0, 0, 0, 9715))
    c.audit_log(t3)
    return c


def x_trade_3() -> Case:
    """A purchase order from draft to received, the blocked supplier at creation and approval, the journal, the trial balance."""
    c = Case("X-trade-3", "X-trade", "purchasing: approval rules, partial receipts, ledger postings, trial balance")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 75})
    c.ok("addSupplier", "sam", T0, {"supplierId": "acme", "name": "Acme Supply"})
    c.ok("addSupplier", "sam", T0, {"supplierId": "zeta", "name": "Zeta Parts"})
    # po-1: ply 800 x 130 = 104000 plus foam 100 x 40 = 4000, total 108000.
    c.ok("createOrder", "sam", T0, {"supplierId": "acme"}, new="po-1")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "ply", "quantity": 800, "unitCostCents": 130})
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-1", "materialId": "foam", "quantity": 100, "unitCostCents": 40},
         {"order": order("po-1", "acme", "draft", "sam", 108000, [oline("ply", 800, 130), oline("foam", 100, 40)])})
    c.ok("submitOrder", "sam", T0, {"orderId": "po-1"})
    c.ok("createOrder", "sam", T0, {"supplierId": "zeta"}, new="po-2")
    c.ok("addOrderLine", "sam", T0, {"orderId": "po-2", "materialId": "foam", "quantity": 10, "unitCostCents": 20})
    c.ok("submitOrder", "sam", T0, {"orderId": "po-2"})
    # Over 100000: not by the staff creator, not by staff at all.
    rf(c, "approveOrder", "sam", T0, {"orderId": "po-1"}, "self_approval")
    rf(c, "approveOrder", "sue", T0, {"orderId": "po-1"}, "approval_requires_admin")
    c.ok("approveOrder", "amy", T0, {"orderId": "po-1"}, {"order": order("po-1", "acme", "approved", "sam", 108000, [oline("ply", 800, 130), oline("foam", 100, 40)])})
    # A supplier blocked at the approval blocks it, and no order can be created for it; unblocked, the approval goes through.
    c.ok("setSupplierStatus", "admin", T0, {"supplierId": "zeta", "status": "blocked"})
    rf(c, "approveOrder", "sue", T0, {"orderId": "po-2"}, "supplier_blocked")
    rf(c, "createOrder", "sam", T0, {"supplierId": "zeta"}, "supplier_blocked")
    c.ok("setSupplierStatus", "admin", T0, {"supplierId": "zeta", "status": "active"})
    c.ok("approveOrder", "sue", T0, {"orderId": "po-2"}, {"order": order("po-2", "zeta", "approved", "sam", 200, [oline("foam", 10, 20)])})
    t = "2026-03-04T09:00Z"
    ply_done = oline("ply", 800, 130, 800)
    c.ok("receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "ply", "quantity": 300},  # 300 x 130 = 39000
         {"order": order("po-1", "acme", "approved", "sam", 108000, [oline("ply", 800, 130, 300), oline("foam", 100, 40)])})
    c.ok("receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "ply", "quantity": 500},  # 500 x 130 = 65000
         {"order": order("po-1", "acme", "approved", "sam", 108000, [ply_done, oline("foam", 100, 40)])})
    c.ok("receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "foam", "quantity": 60},  # 60 x 40 = 2400
         {"order": order("po-1", "acme", "approved", "sam", 108000, [ply_done, oline("foam", 100, 40, 60)])})
    rf(c, "receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "foam", "quantity": 41}, "over_receipt")  # 60 + 41 > 100
    c.ok("receiveOrder", "sam", t, {"orderId": "po-1", "materialId": "foam", "quantity": 40},  # 40 x 40 = 1600
         {"order": order("po-1", "acme", "received", "sam", 108000, [ply_done, oline("foam", 100, 40, 100)])})
    c.ok("receiveOrder", "sam", t, {"orderId": "po-2", "materialId": "foam", "quantity": 10},  # 10 x 20 = 200
         {"order": order("po-2", "zeta", "received", "sam", 200, [oline("foam", 10, 20, 10)])})
    c.ok("listOrders", "admin", t, {"supplierId": "acme"}, {"supplierId": "acme", "orders": [
        order("po-1", "acme", "received", "sam", 108000, [ply_done, oline("foam", 100, 40, 100)])]})
    # Stock 100 + 800 and 100 + 10; the shop's prices are untouched.
    c.ok("listMaterials", "sam", t, {}, {"materials": [mat("foam", "Foam board", 75, 110), mat("ply", "Plywood", 150, 900)]})
    c.ok("listJournalEntries", "admin", t, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, t, "inventory", "payable", 39000, "po-1"), jrn(2, t, "inventory", "payable", 65000, "po-1"),
        jrn(3, t, "inventory", "payable", 2400, "po-1"), jrn(4, t, "inventory", "payable", 1600, "po-1"),
        jrn(5, t, "inventory", "payable", 200, "po-2")]})
    # 39000 + 65000 + 2400 + 1600 + 200 = 108200 on both sides.
    c.ok("getTrialBalance", "admin", t, {}, trial({"inventory": 108200}, {"payable": 108200}))
    c.audit_log(t)
    return c


def x_trade_4() -> Case:
    """A quote captures rates at sending and is not changed by a later rate change; validity on its last day and after."""
    c = Case("X-trade-4", "X-trade", "quotes/quote-pricing, quote-validity, equipment rate, jobs/maker-gate")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 236})
    c.ok("addCustomer", "sam", T0, {"customerId": "cu1", "name": "Cora Customer"})
    c.ok("createQuote", "sam", T0, {"customerId": "cu1", "title": "Sign"}, new="quo-1")
    c.ok("addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L1", "quantity": 30})
    c.ok("addQuoteLine", "sam", T0, {"quoteId": "quo-1", "lineKind": "material", "ref": "foam", "quantity": 7})
    # A draft captures nothing: the rate set before sending is the one captured.
    c.ok("setMachineRate", "admin", T0, {"machineId": "L1", "rateCentsPerMinute": 50},
         {"machine": {"machineId": "L1", "name": "Laser one", "familyId": "laser", "rateCentsPerMinute": 50}})
    c.ok("getQuote", "sam", T0, {"quoteId": "quo-1"}, {"quote": quote("quo-1", "cu1", "Sign", "draft", [qline("machine", "L1", 30), qline("material", "foam", 7)])})
    # Subtotal 30 x 50 + 7 x 236 = 1500 + 1652 = 3152; total (3152 x 120 + 50) div 100 = 378290 div 100 = 3782.
    t1 = "2026-03-02T15:00Z"
    sent1 = quote("quo-1", "cu1", "Sign", "sent", [qline("machine", "L1", 30, 50), qline("material", "foam", 7, 236)], 3152, 3782, t1, "2026-04-01")
    c.ok("sendQuote", "sam", t1, {"quoteId": "quo-1"}, {"quote": sent1})
    # A later change of the rate changes nothing of the sent quote.
    t2 = "2026-03-03T09:00Z"
    c.ok("setMachineRate", "admin", t2, {"machineId": "L1", "rateCentsPerMinute": 80})
    c.ok("getQuote", "sam", t2, {"quoteId": "quo-1"}, {"quote": sent1})
    # A quote sent afterwards uses the new rate: 30 x 80 + 1652 = 4052; total (4052 x 120 + 50) div 100 = 486290 div 100 = 4862.
    c.ok("createQuote", "sam", t2, {"customerId": "cu1", "title": "Sign again"}, new="quo-2")
    c.ok("addQuoteLine", "sam", t2, {"quoteId": "quo-2", "lineKind": "machine", "ref": "L1", "quantity": 30})
    c.ok("addQuoteLine", "sam", t2, {"quoteId": "quo-2", "lineKind": "material", "ref": "foam", "quantity": 7})
    sent2 = quote("quo-2", "cu1", "Sign again", "sent", [qline("machine", "L1", 30, 80), qline("material", "foam", 7, 236)], 4052, 4862, t2, "2026-04-02")
    c.ok("sendQuote", "sam", t2, {"quoteId": "quo-2"}, {"quote": sent2})
    rf(c, "startJob", "sam", t2, {"quoteId": "quo-1", "makerId": "ada"}, "quote_not_accepted")
    # quo-1 is valid until 2026-04-01 and quo-2 until 2026-04-02. Accepting quo-1 on its last day works.
    t3 = "2026-04-01T23:59Z"
    c.ok("decideQuote", "sam", t3, {"quoteId": "quo-1", "decision": "accepted"},
         {"quote": quote("quo-1", "cu1", "Sign", "accepted", sent1["lines"], 3152, 3782, t1, "2026-04-01")})
    # quo-2 is accepted the day after its last valid day: expired, and it stays sent until it is rejected.
    t4 = "2026-04-03T00:00Z"
    rf(c, "decideQuote", "sam", t4, {"quoteId": "quo-2", "decision": "accepted"}, "quote_expired")
    c.ok("getQuote", "sam", t4, {"quoteId": "quo-2"}, {"quote": sent2})
    c.ok("decideQuote", "sam", t4, {"quoteId": "quo-2", "decision": "rejected"},
         {"quote": quote("quo-2", "cu1", "Sign again", "rejected", sent2["lines"], 4052, 4862, t2, "2026-04-02")})
    rf(c, "startJob", "sam", t4, {"quoteId": "quo-2", "makerId": "ada"}, "quote_not_accepted")
    # The job takes the total of the quote as sent.
    c.ok("startJob", "sam", t4, {"quoteId": "quo-1", "makerId": "ada"},
         {"job": job("job-1", "quo-1", "cu1", "ada", "started", 3782, None, 0)}, new="job-1")
    c.audit_log(t4)
    return c


def x_trade_5() -> Case:
    """A job from an accepted quote through completion: credit in balance, journal and the next invoice, payments."""
    c = Case("X-trade-5", "X-trade", "jobs: start, maker credit into the balance and journal, payments, invoice credit")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "foam", "name": "Foam board", "unitPriceCents": 236})
    c.ok("addCustomer", "sam", T0, {"customerId": "cu1", "name": "Cora Customer"})
    # Subtotal 30 x 45 + 7 x 236 = 1350 + 1652 = 3002, total (3002 x 120 + 50) div 100 = 3602 (the spec example).
    t1 = "2026-03-02T15:00Z"
    c.ok("createQuote", "sam", t1, {"customerId": "cu1", "title": "Sign"}, new="quo-1")
    c.ok("addQuoteLine", "sam", t1, {"quoteId": "quo-1", "lineKind": "machine", "ref": "L1", "quantity": 30})
    c.ok("addQuoteLine", "sam", t1, {"quoteId": "quo-1", "lineKind": "material", "ref": "foam", "quantity": 7})
    c.ok("sendQuote", "sam", t1, {"quoteId": "quo-1"}, {"quote": quote(
        "quo-1", "cu1", "Sign", "sent", [qline("machine", "L1", 30, 45), qline("material", "foam", 7, 236)], 3002, 3602, t1, "2026-04-01")})
    c.ok("decideQuote", "sam", "2026-03-02T16:00Z", {"quoteId": "quo-1", "decision": "accepted"})
    t2 = "2026-03-03T09:00Z"
    c.ok("startJob", "sam", t2, {"quoteId": "quo-1", "makerId": "ada"}, {"job": job("job-1", "quo-1", "cu1", "ada", "started", 3602, None, 0)}, new="job-1")
    rf(c, "recordJobPayment", "sam", t2, {"jobId": "job-1", "amountCents": 100}, "job_not_completed")
    rf(c, "startJob", "sam", t2, {"quoteId": "quo-1", "makerId": "bob"}, "already_started")
    c.ok("getAccount", "ada", t2, {"memberId": "ada"}, account("ada", 0, 0, 0, 5000))
    # The maker's standing is not judged at completion. Credit (3602 x 30 + 50) div 100 = 1081.
    t3 = "2026-03-05T09:00Z"
    c.ok("setMemberStanding", "sam", t3, {"memberId": "ada", "standing": "suspended"})
    c.ok("completeJob", "sam", t3, {"jobId": "job-1"}, {"job": job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 0)})
    c.ok("getAccount", "ada", t3, {"memberId": "ada"}, account("ada", 1081, 0, 0, 5000))
    c.ok("recordJobPayment", "sam", t3, {"jobId": "job-1", "amountCents": 1000}, {"job": job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 1000)})
    c.ok("recordJobPayment", "sam", t3, {"jobId": "job-1", "amountCents": 2000}, {"job": job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 3000)})
    rf(c, "recordJobPayment", "sam", t3, {"jobId": "job-1", "amountCents": 603}, "overpayment")
    c.ok("recordJobPayment", "sam", t3, {"jobId": "job-1", "amountCents": 602}, {"job": job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 3602)})
    rf(c, "recordJobPayment", "sam", t3, {"jobId": "job-1", "amountCents": 1}, "overpayment")
    c.ok("listJournalEntries", "admin", t3, {"month": "2026-03"}, {"month": "2026-03", "entries": [
        jrn(1, t3, "expenseMaker", "creditLiability", 1081, "job-1"), jrn(2, t3, "cash", "revenueJobs", 1000, "job-1"),
        jrn(3, t3, "cash", "revenueJobs", 2000, "job-1"), jrn(4, t3, "cash", "revenueJobs", 602, "job-1")]})
    c.ok("getTrialBalance", "admin", t3, {}, trial({"expenseMaker": 1081, "cash": 3602}, {"creditLiability": 1081, "revenueJobs": 3602}))
    # The credit applies to the next invoice, whatever the standing: March invoice of ada, membership 4900 + course 5000 = 9900,
    # minus the credit 1081 = 8819 open.
    t4 = "2026-04-01T08:00Z"
    c.ok("generateInvoices", "sam", t4, {"month": "2026-03"})
    c.ok("getAccount", "ada", t4, {"memberId": "ada"}, account("ada", 0, 8819, 0, 0))
    c.ok("listJobs", "sam", t4, {"customerId": "cu1"}, {"customerId": "cu1", "jobs": [job("job-1", "quo-1", "cu1", "ada", "completed", 3602, 1081, 3602)]})
    c.audit_log(t4)
    return c


def x_trade_6() -> Case:
    """A project with a collaborator: reservations in several statuses and purchases, the cost by member."""
    c = Case("X-trade-6", "X-trade", "projects/project-cost, attachment with bookings and materials")
    world(c)
    c.ok("addMaterial", "sam", T0, {"materialId": "glue", "name": "Glue", "unitPriceCents": 250})
    c.ok("restockMaterial", "sam", T0, {"materialId": "glue", "quantity": 20})
    d3, d4, d5 = "2026-03-03", "2026-03-04", "2026-03-05"
    for i, slot in enumerate(((d3, "L1", "10:00", "11:00"), (d3, "L2", "14:00", "15:00"), (d4, "L1", "10:00", "11:00"), (d5, "L2", "10:00", "11:00")), start=1):
        c.ok("createReservation", "sam", T0, {"memberId": "ada", "machineId": slot[1], "start": f"{slot[0]}T{slot[2]}Z", "end": f"{slot[0]}T{slot[3]}Z"},
             new=f"res-{i}")
    c.ok("createProject", "ada", T0, {"projectId": "p1", "ownerId": "ada", "title": "Lamp"})
    c.ok("addCollaborator", "ada", T0, {"projectId": "p1", "memberId": "bob"})
    for r in ("res-1", "res-2", "res-3", "res-4"):
        c.ok("attachReservation", "ada", T0, {"projectId": "p1", "reservationId": r})
    c.ok("cancelReservation", "ada", T0, {"reservationId": "res-4"})
    c.ok("getProjectCost", "ada", T0, {"projectId": "p1"}, cost("p1", 0, 0, []))
    # res-1: 37 minutes at 45 with the plus discount 10 %: (37 x 45 x 90 + 50) div 100 = 149900 div 100 = 1499.
    ta, tb = f"{d3}T10:00Z", f"{d3}T10:37Z"
    c.ok("checkIn", "ada", ta, {"reservationId": "res-1"})
    c.ok("getProjectCost", "ada", ta, {"projectId": "p1"}, cost("p1", 0, 0, []))  # checked in adds nothing yet
    c.ok("checkOut", "ada", tb, {"reservationId": "res-1"}, {
        "reservation": resv("res-1", "ada", "L1", f"{d3}T10:00Z", f"{d3}T11:00Z", "completed", ta, tb, 45, 37, 1499),
        "charge": {"kind": "usage", "ref": "res-1", "quantity": 37, "unitPriceCents": 45, "discountPercent": 10, "amountCents": 1499, "at": tb}})
    t = f"{d3}T11:00Z"
    c.ok("purchaseMaterial", "ada", t, {"memberId": "ada", "materialId": "ply", "quantity": 4}, new="pur-1")  # 4 x 150 = 600
    c.ok("purchaseMaterial", "bob", t, {"memberId": "bob", "materialId": "glue", "quantity": 1}, new="pur-2")  # 250
    c.ok("attachPurchase", "ada", t, {"projectId": "p1", "purchaseId": "pur-2"})
    c.ok("attachPurchase", "ada", t, {"projectId": "p1", "purchaseId": "pur-1"},
         {"project": proj("p1", "ada", "Lamp", ["bob"], res=["res-1", "res-2", "res-3", "res-4"], purchases=["pur-2", "pur-1"])})
    # The example of the requirements: usage 1499, material 600 + 250, total 2349; bob has a share without usage.
    c.ok("getProjectCost", "ada", t, {"projectId": "p1"}, cost("p1", 1499, 850, [share("ada", 1499, 600), share("bob", 0, 250)]))
    rf(c, "getProjectCost", "bob", t, {"projectId": "p1"}, "forbidden")
    # res-2 is checked in at 14:00; a rate change before check-out does not change its captured rate 60.
    tc, td, te = f"{d3}T14:00Z", f"{d3}T14:10Z", f"{d3}T14:20Z"
    c.ok("checkIn", "ada", tc, {"reservationId": "res-2"})
    c.ok("setMachineRate", "admin", td, {"machineId": "L2", "rateCentsPerMinute": 100})
    c.ok("getProjectCost", "sam", td, {"projectId": "p1"}, cost("p1", 1499, 850, [share("ada", 1499, 600), share("bob", 0, 250)]))
    # 20 minutes at 60 with 10 %: (20 x 60 x 90 + 50) div 100 = 108050 div 100 = 1080. Usage 1499 + 1080 = 2579.
    c.ok("checkOut", "ada", te, {"reservationId": "res-2"}, {
        "reservation": resv("res-2", "ada", "L2", f"{d3}T14:00Z", f"{d3}T15:00Z", "completed", tc, te, 60, 20, 1080),
        "charge": {"kind": "usage", "ref": "res-2", "quantity": 20, "unitPriceCents": 60, "discountPercent": 10, "amountCents": 1080, "at": te}})
    c.ok("getProjectCost", "ada", te, {"projectId": "p1"}, cost("p1", 2579, 850, [share("ada", 2579, 600), share("bob", 0, 250)]))
    # Account of ada: course 5000 (world) + usage 1499 + ply 600 + usage 1080 = 8179; bob only the glue.
    c.ok("getAccount", "ada", te, {"memberId": "ada"}, account("ada", 0, 0, 0, 8179))
    c.ok("getAccount", "bob", te, {"memberId": "bob"}, account("bob", 0, 0, 0, 250))
    c.ok("closeProject", "ada", te, {"projectId": "p1"},
         {"project": proj("p1", "ada", "Lamp", ["bob"], "closed", ["res-1", "res-2", "res-3", "res-4"], ["pur-2", "pur-1"])})
    c.ok("getProjectCost", "sam", te, {"projectId": "p1"}, cost("p1", 2579, 850, [share("ada", 2579, 600), share("bob", 0, 250)]))
    c.audit_log(te)
    return c


def b_projects_3():
    c = Case("B-projects-3", "B-projects", "projects: each share of the cost is the share of the member whose record it is, not the owner's")
    t = world(c)
    c.ok("enrollInCourse", "sam", t, {"courseId": "laser-101", "memberId": "bob"})
    c.ok("completeCourse", "sam", t, {"courseId": "laser-101", "memberId": "bob"})
    c.ok("createProject", "ada", t, {"projectId": "p1", "ownerId": "ada", "title": "Frame"})
    c.ok("addCollaborator", "ada", t, {"projectId": "p1", "memberId": "bob"})
    start, end = "2026-03-03T10:00Z", "2026-03-03T10:30Z"
    c.ok("createReservation", "bob", t, {"memberId": "bob", "machineId": "L1", "start": start, "end": end}, new="res-1")
    c.ok("checkIn", "bob", start, {"reservationId": "res-1"})
    c.ok("checkOut", "bob", end, {"reservationId": "res-1"})
    # bob, a collaborator, on tier basic: 30 minutes at 45 with no discount is 1350
    c.ok("attachReservation", "ada", end, {"projectId": "p1", "reservationId": "res-1"})
    c.ok("getProjectCost", "ada", end, {"projectId": "p1"},
         {"projectId": "p1", "usageCents": 1350, "materialCents": 0, "totalCents": 1350, "byMember": [{"memberId": "bob", "usageCents": 1350, "materialCents": 0}]})
    # 2 of ply at 150 is 300, ada's own
    c.ok("purchaseMaterial", "ada", end, {"memberId": "ada", "materialId": "ply", "quantity": 2}, new="pur-1")
    c.ok("attachPurchase", "ada", end, {"projectId": "p1", "purchaseId": "pur-1"})
    c.ok("getProjectCost", "ada", end, {"projectId": "p1"},
         {"projectId": "p1", "usageCents": 1350, "materialCents": 300, "totalCents": 1650,
          "byMember": [{"memberId": "ada", "usageCents": 0, "materialCents": 300}, {"memberId": "bob", "usageCents": 1350, "materialCents": 0}]})
    c.audit_log(end)
    return c


def build_cases() -> list:
    makers = [b_hazmat_1, b_hazmat_2, b_hazmat_3, b_materials_1, b_materials_2, b_purchasing_1, b_purchasing_2, b_purchasing_3,
              b_stocktake_1, b_stocktake_2, b_projects_1, b_projects_2, b_projects_3, b_quotes_1, b_quotes_2, b_jobs_1, b_jobs_2,
              x_trade_1, x_trade_2, x_trade_3, x_trade_4, x_trade_5, x_trade_6]
    return [make().build() for make in makers]
