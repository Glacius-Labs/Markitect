"""The case kit of the large Makerspace oracle: explicit requests with explicit expected responses.

A case is the Makerspace oracle's form, {id, families, cite, steps}, run by its harness. Results are compared in
full when an expected result is given. Generated identifiers are stated literally, since their numbering is a rule of
the program. The kit keeps the expected audit log, so a case can compare it at any point with `audit_log`.

  c = Case("B-lockouts-1", "B-lockouts")
  world(c)
  c.ok("applyLockout", "sam", T1, {"machineId": "L1", "reason": "belt"}, {"lockout": {...}}, new="lck-1")
  c.refuse("releaseLockout", "sam", T2, {"lockoutId": "lck-9"}, "not_found")
  c.audit_log(T2)
"""
from __future__ import annotations

import json
import pathlib
import re
import sys
from typing import Any

HERE = pathlib.Path(__file__).resolve().parent
OLD = HERE.parent.parent / "makerspace-oracle"
for path in (HERE, OLD):
    if path.is_dir() and str(path) not in sys.path:
        sys.path.insert(0, str(path))

ANY = {"$any": True}
NONEMPTY = {"$nonempty": True}
FACTS_FILE = HERE / "facts.json"
OPS: dict[str, dict[str, Any]] = json.loads(FACTS_FILE.read_text(encoding="utf-8")) if FACTS_FILE.is_file() else {}
FIXED = re.compile(r"the fixed string `([^`]+)`")
ARGS = re.compile(r"`([^`]+)`")


def has(**members: Any) -> dict[str, Any]:
    """An object with at least these members, compared exactly."""
    return {"$has": members}


def ref(name: str) -> dict[str, str]:
    return {"$ref": name}


def audit_target(op: dict[str, Any], args: dict[str, Any], new: str | None) -> str:
    """The `target` of the operation's audit entry, read from its page's Audit target line."""
    text = op["audit"]
    fixed = FIXED.search(text)
    if fixed:
        return fixed.group(1)
    if "new" in text.split("`")[0]:
        if new is None:
            raise ValueError(f"{op['wire']}: its audit target is a new identifier: pass new=")
        return new
    names = ARGS.findall(text)
    if not names:
        raise ValueError(f"{op['wire']}: audit target not understood: {text}")
    name = names[0]
    if "/" in name:
        return "/".join(str(args[part]) for part in name.split("/"))
    return str(args[name])


class Case:
    def __init__(self, cid: str, family: str | list[str], cite: str = "") -> None:
        self.id, self.families = cid, [family] if isinstance(family, str) else list(family)
        self.cite = cite
        self.steps: list[dict[str, Any]] = []
        self.audit: list[dict[str, Any]] = []
        self.n = 0
        self.last_ok: str | None = None

    def _rid(self) -> str:
        self.n += 1
        return f"{self.id}-{self.n:04d}"

    def _guard(self, at: str, refused: bool) -> None:
        if self.last_ok is not None and at < self.last_ok and not refused:
            raise ValueError(f"{self.id}: {at} goes back from {self.last_ok}")

    def ok(self, op: str, actor: str, at: str, args: dict[str, Any] | None = None, result: Any = None, *, new: str | None = None,
           capture: dict[str, list] | None = None, target: str | None = None) -> None:
        """A request answered ok. `result` is compared in full; without it only the status is. `new` is a new identifier."""
        args = dict(args or {})
        self._guard(at, False)
        rid = self._rid()
        step: dict[str, Any] = {"request": {"id": rid, "op": op, "args": {"actor": actor, "at": at, **args}},
                                "expect": {"id": rid, "status": "ok", "result": ANY if result is None else result}}
        if capture:
            step["capture"] = capture
        self.steps.append(step)
        self.last_ok = at
        info = OPS.get(op)
        if info is None:
            raise ValueError(f"{self.id}: no operation {op}")
        if info["mutation"]:
            self.audit.append({"seq": len(self.audit) + 1, "at": at, "actor": actor, "op": op,
                               "target": target if target is not None else audit_target(info, args, new)})

    def refuse(self, op: str, actor: str, at: str, args: dict[str, Any] | None, code: str) -> None:
        """A request refused with `code`: nothing happens, no audit entry, the request-time order does not move."""
        self._guard(at, True)
        rid = self._rid()
        self.steps.append({"request": {"id": rid, "op": op, "args": {"actor": actor, "at": at, **(args or {})}},
                           "expect": {"id": rid, "status": "refused", "code": code, "reason": NONEMPTY}})

    def invalid(self, op: str, actor: str, at: str, args: dict[str, Any]) -> None:
        """A request with a member that is not an argument: an invalid request, nothing happens."""
        rid = self._rid()
        self.steps.append({"request": {"id": rid, "op": op, "args": {"actor": actor, "at": at, **args}},
                           "expect": {"id": rid, "status": "invalidRequest", "reason": NONEMPTY}})

    def audit_log(self, at: str | None = None) -> None:
        """Compare the whole expected log."""
        when = at or self.last_ok
        if when is None:
            raise ValueError(f"{self.id}: no time for the audit log")
        self.ok("listAuditEntries", "admin", when, {}, {"entries": [dict(e) for e in self.audit]})

    def build(self) -> dict[str, Any]:
        return {"id": self.id, "families": self.families, "cite": [{"section": "requirements", "text": self.cite}] if self.cite else [],
                "steps": self.steps}


def world(c: Case, at: str = "2026-03-02T09:00Z") -> str:
    """The shared start: staff `sam` and `sue`, admin `amy`, members `ada` (plus), `bob` (basic), `eve` (pro); family `laser`
    with machines `L1` (rate 45) and `L2` (rate 60); family `lathe` with machine `T1` (rate 30); course `laser-101` (fee 5000,
    capacity 2) with `ada` certified on it; material `ply` (price 150, stock 100). No other record exists. Returns `at`."""
    c.ok("registerUser", "admin", at, {"userId": "sam", "name": "Sam Staff", "role": "staff"})
    c.ok("registerUser", "admin", at, {"userId": "sue", "name": "Sue Staff", "role": "staff"})
    c.ok("registerUser", "admin", at, {"userId": "amy", "name": "Amy Admin", "role": "admin"})
    c.ok("registerMember", "sam", at, {"memberId": "ada", "name": "Ada Byron", "tier": "plus"})
    c.ok("registerMember", "sam", at, {"memberId": "bob", "name": "Bob Stone", "tier": "basic"})
    c.ok("registerMember", "sam", at, {"memberId": "eve", "name": "Eve Marsh", "tier": "pro"})
    c.ok("addFamily", "admin", at, {"familyId": "laser", "name": "Laser cutters"})
    c.ok("addFamily", "admin", at, {"familyId": "lathe", "name": "Lathes"})
    c.ok("addMachine", "admin", at, {"machineId": "L1", "name": "Laser one", "familyId": "laser", "rateCentsPerMinute": 45})
    c.ok("addMachine", "admin", at, {"machineId": "L2", "name": "Laser two", "familyId": "laser", "rateCentsPerMinute": 60})
    c.ok("addMachine", "admin", at, {"machineId": "T1", "name": "Lathe one", "familyId": "lathe", "rateCentsPerMinute": 30})
    c.ok("addCourse", "sam", at, {"courseId": "laser-101", "familyId": "laser", "title": "Laser safety", "feeCents": 5000, "capacity": 2})
    c.ok("enrollInCourse", "sam", at, {"courseId": "laser-101", "memberId": "ada"})
    c.ok("completeCourse", "sam", at, {"courseId": "laser-101", "memberId": "ada"})
    c.ok("addMaterial", "sam", at, {"materialId": "ply", "name": "Plywood", "unitPriceCents": 150})
    c.ok("restockMaterial", "sam", at, {"materialId": "ply", "quantity": 100})
    return at
