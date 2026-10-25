"""The shared starting world of most cases, with fully stated expected results.

Monday 2026-03-02 (w0 9.1's example dates: Tuesday 2026-03-03, Saturday
2026-03-07). Setup requests are made by staff or admins only, so a probe of
member permissions never depends on them.
"""

from __future__ import annotations

import facts as f
from builder import Case

T0 = "2026-03-02T08:00Z"
ISSUED = "2026-03-02"
EXPIRES = "2027-03-02"  # 365 days after 2026-03-02 (w0 7.2)
STAFF = "sam"
MEMBERS = {"ada": ("Ada", "plus"), "bob": ("Bob", "basic"), "cy": ("Cy", "pro")}
MACHINES = {"L1": ("Laser one", "laser", 45), "L2": ("Laser two", "laser", 30), "W1": ("Lathe one", "wood", 20)}
LASER_FEE = 3000


def people(c: Case, at: str = T0, members=("ada", "bob", "cy")) -> None:
    c.ok("registerUser", "admin", at, {"userId": STAFF, "name": "Sam", "role": "staff"},
         {"user": f.user(STAFF, "Sam", "staff")})
    for mid in members:
        name, tier = MEMBERS[mid]
        c.ok("registerMember", STAFF, at, {"memberId": mid, "name": name, "tier": tier},
             {"member": f.member(mid, name, tier, at)})


def catalog(c: Case, at: str = T0, machines=("L1", "L2", "W1")) -> None:
    c.ok("addFamily", "admin", at, {"familyId": "laser", "name": "Laser cutters"}, {"family": f.family("laser", "Laser cutters")})
    c.ok("addFamily", "admin", at, {"familyId": "wood", "name": "Wood lathes"}, {"family": f.family("wood", "Wood lathes")})
    for mid in machines:
        name, fam, rate = MACHINES[mid]
        c.ok("addMachine", "admin", at, {"machineId": mid, "name": name, "familyId": fam, "rateCentsPerMinute": rate},
             {"machine": f.machine(mid, name, fam, rate)})


def certify(c: Case, members, at: str = T0) -> None:
    """Enroll and complete laser-101 one member at a time (charge 3000 each, w0 7.1)."""
    c.ok("addCourse", STAFF, at, {"courseId": "laser-101", "familyId": "laser", "title": "Laser basics",
                                  "feeCents": LASER_FEE, "capacity": 10},
         {"course": f.course("laser-101", "laser", "Laser basics", LASER_FEE, 10, 0)})
    for mid in members:
        c.ok("enrollInCourse", STAFF, at, {"courseId": "laser-101", "memberId": mid},
             {"enrollment": f.enrollment("laser-101", mid, "enrolled", at),
              "course": f.course("laser-101", "laser", "Laser basics", LASER_FEE, 10, 1)})
        c.ok("completeCourse", STAFF, at, {"courseId": "laser-101", "memberId": mid},
             {"enrollment": f.enrollment("laser-101", mid, "completed", at, at),
              "certification": f.certification(c.step, mid, "laser", at[:10], EXPIRES if at[:10] == ISSUED else None,
                                               "valid")})


def shop(c: Case, at: str = T0) -> None:
    c.ok("addMaterial", STAFF, at, {"materialId": "acrylic", "name": "Acrylic sheet", "unitPriceCents": 199},
         {"material": f.material("acrylic", "Acrylic sheet", 199, 0)})
    c.ok("addMaterial", STAFF, at, {"materialId": "plywood", "name": "Plywood panel", "unitPriceCents": 37},
         {"material": f.material("plywood", "Plywood panel", 37, 0)})
    c.ok("restockMaterial", STAFF, at, {"materialId": "acrylic", "quantity": 100}, {"material": f.material("acrylic", "Acrylic sheet", 199, 100)})
    c.ok("restockMaterial", STAFF, at, {"materialId": "plywood", "quantity": 100}, {"material": f.material("plywood", "Plywood panel", 37, 100)})


def world(c: Case, at: str = T0, members=("ada", "bob", "cy"), certified=("ada", "bob", "cy"), materials: bool = True) -> None:
    people(c, at, members)
    catalog(c, at)
    certify(c, certified, at)
    if materials:
        shop(c, at)


def reserve(c: Case, member_id: str, machine_id: str, start: str, end: str, at: str, actor: str = STAFF, *, check=None, cite=None) -> str:
    """createReservation answered ok with its full expected record."""
    return c.ok("createReservation", actor, at, {"memberId": member_id, "machineId": machine_id, "start": start, "end": end},
                lambda rid: {"reservation": f.reservation(rid, member_id, machine_id, start, end)}, check=check, cite=cite)
