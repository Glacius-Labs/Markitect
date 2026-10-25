"""Case builder: explicit request sequences with explicit expected responses.

A case is the Library oracle's case form, {id, families, steps}, plus `cite`.
A step is {request | raw, expect, capture?, cite?, snapshot?}. Generated
identifiers are deterministic (w0 2.6): strict cases state them literally;
loose (probe) cases capture them and refer to them with {"$ref": name}, so a
probe never depends on the numbering rule it does not test. The builder keeps only
what the requirements make computable without realizing an operation: the
expected audit log, identifier counters, user roles, the request-time order
and which records exist (to choose the reads of a no-change snapshot).
"""

from __future__ import annotations

import json
from typing import Any, Callable

import facts
from harness import ANY, NONEMPTY


def ts(date: str, hm: str) -> str:
    return f"{date}T{hm}Z"


def _next_month_start(month: str) -> str:
    year, mon = int(month[:4]), int(month[5:7])
    year, mon = (year + 1, 1) if mon == 12 else (year, mon + 1)
    return f"{year:04d}-{mon:02d}-01T00:00Z"


CAPTURE_PATHS = {"createReservation": ["result", "reservation", "reservationId"], "scheduleMaintenance": ["result", "window", "windowId"],
                 "purchaseMaterial": ["result", "purchase", "purchaseId"], "recordPayment": ["result", "payment", "paymentId"],
                 "reportIncident": ["result", "incident", "incidentId"], "lockMachine": ["result", "lockout", "lockoutId"]}


class Case:
    def __init__(self, step: str, cid: str, families: list[str], cites: list[dict[str, str]], strict: bool = True,
                 capture: bool | None = None):
        cites = facts.scope_refs(cites, step)
        if not cites:
            raise ValueError(f"{cid}: every case cites the requirement it checks")
        self.step, self.id, self.families, self.cites, self.strict = step, cid, list(families), cites, strict
        self.capture = (not strict) if capture is None else capture
        self.ops = facts.ops(step)
        self.steps: list[dict[str, Any]] = []
        self.invariants: list[str] = []
        self.n = 0
        self.roles = {"admin": "admin"}
        self.registered: dict[str, str] = {}
        self.machines: list[str] = []
        self.dates: set[str] = set()
        self.audit: list[dict[str, Any]] = []
        self.counters: dict[str, int] = {}
        self.last_ok: str | None = None
        self.snapshots = 0

    # --- Request construction -----------------------------------------------------

    def _rid(self) -> str:
        self.n += 1
        return f"{self.id}-{self.n:04d}"

    def _append(self, step: dict[str, Any], cite: dict[str, str] | list | None) -> dict[str, Any]:
        cites = facts.scope_refs(cite if isinstance(cite, list) else [cite], self.step) if cite else []
        if cites:
            step["cite"] = cites
        self.steps.append(step)
        return step

    def peek(self, prefix: str, offset: int = 1) -> str:
        """The identifier the next creation of this kind must get (w0 2.6)."""
        return f"{prefix}-{self.counters.get(prefix, 0) + offset}"

    def _take(self, prefix: str) -> str:
        self.counters[prefix] = self.counters.get(prefix, 0) + 1
        return f"{prefix}-{self.counters[prefix]}"

    def eligible(self, month: str) -> list[str]:
        """w0 11.1: members registered before the first minute of the month after M, by memberId."""
        cutoff = _next_month_start(month)
        return sorted(m for m, at in self.registered.items() if at < cutoff)

    def _audit(self, at: str, actor: str, op: str, target: str | None, outcome: str) -> None:
        self.audit.append(facts.audit_entry(self.step, len(self.audit) + 1, at, actor, self.roles[actor],
                                            op, target, outcome))

    def _track(self, op: str, args: dict[str, Any], ids: list[str], at: str) -> None:
        if op == "registerUser":
            self.roles[args["userId"]] = args["role"]
        elif op == "registerMember":
            self.roles[args["memberId"]] = "member"
            self.registered[args["memberId"]] = at
        elif op == "addMachine":
            self.machines.append(args["machineId"])
        for key in ("start", "end"):
            if op in ("createReservation", "scheduleMaintenance", "lockMachine") and key in args:
                self.dates.add(args[key][:10])

    def _check_time(self, at: str, code: str | None) -> None:
        """Authoring guard: only time_regression probes may go back in time (w0 4.4)."""
        regress = self.last_ok is not None and at < self.last_ok
        if regress != (code == "time_regression") and code not in ("invalid_argument", "unknown_actor"):
            raise ValueError(f"{self.id}: at {at} after last ok {self.last_ok} does not fit code {code}")

    def ok(self, op: str, actor: str, at: str, args: dict[str, Any] | None = None, result: Any = None, *,
           check: bool | None = None, cite=None, rid: str | None = None) -> Any:
        """A request answered ok. Returns the generated identifier(s), if the operation creates records.

        `result` may be a callable taking the new identifier (generateInvoices: a
        mapping memberId -> invoiceId). In a loose case only `check=True` steps
        compare the result; the others assert `status: ok`.
        """
        args = dict(args or {})
        info = self.ops[op]
        self._check_time(at, None)
        prefix = facts.CREATES.get(op)
        captures: dict[str, list] = {}
        if op == "generateInvoices":
            members = self.eligible(args["month"])
            if self.capture:
                symbols = {m: self._take("cap-inv") for m in members}
                captures = {s: ["result", "invoices", {"memberId": m}, "invoiceId"] for m, s in symbols.items()}
                ids = [{"$ref": s} for s in symbols.values()]
            else:
                ids = [self._take("inv") for _ in members]
            new: Any = dict(zip(members, ids))
        elif prefix and self.capture:
            symbol = self._take("cap-" + prefix)
            captures = {symbol: CAPTURE_PATHS[op]}
            ids = [{"$ref": symbol}]
            new = ids[0]
        elif prefix:
            ids = [self._take(prefix)]
            new = ids[0]
        else:
            ids, new = [], None
        if callable(result):
            result = result(new)
        compare = self.strict if check is None else check
        rid = rid or self._rid()
        request = {"id": rid, "op": op, "args": {"actor": actor, "at": at, **args}}
        expected = result if compare and result is not None else ANY
        step = {"request": request, "expect": {"id": rid, "status": "ok", "result": expected}}
        if captures:
            step["capture"] = captures
        self._append(step, cite)
        if info["kind"] == "mutation":
            self._audit(at, actor, op, facts.audit_target(info, args, ids[0] if ids else None), "accepted")
        self._track(op, args, ids, at)
        self.last_ok = at
        return new

    def refuse(self, op: str, actor: str, at: str, args: dict[str, Any] | None, code: str, *, cite=None,
               snapshot: bool = False, rid: str | None = None) -> None:
        """A request refused with `code`; optionally bracketed by an unchanged-state snapshot."""
        self._check_time(at, code)
        rid = rid or self._rid()
        request = {"id": rid, "op": op, "args": {"actor": actor, "at": at, **(args or {})}}
        self._refusal({"request": request, "expect": self._refused(rid, code)}, actor, op, at, code, cite, snapshot)

    def refuse_raw(self, op: str, actor: str, at: str, args_json: str, code: str, *, cite=None) -> None:
        """A refusal whose arguments need literal JSON text (for example 1e3 or -0)."""
        rid = self._rid()
        raw = ('{"id":%s,"op":%s,"args":{"actor":%s,"at":%s,%s}}'
               % (json.dumps(rid), json.dumps(op), json.dumps(actor), json.dumps(at), args_json))
        self._refusal({"raw": raw, "expect": self._refused(rid, code)}, actor, op, at, code, cite, False)

    @staticmethod
    def _refused(rid: str, code: str) -> dict[str, Any]:
        return {"id": rid, "status": "refused", "code": code, "reason": NONEMPTY}

    def _refusal(self, step, actor, op, at, code, cite, snapshot) -> None:
        if snapshot:
            label = self._snapshot_label()
            self._snapshot(label, at, "$record")
        self._append(step, cite)
        if code == "forbidden" and facts.since(self.step, "C1"):
            self._audit(at, actor, op, None, "refused")
        if snapshot:
            self._snapshot(label, at, "$same")

    def invalid(self, raw: str, echo: str | None, *, cite=None) -> None:
        self._append({"raw": raw, "expect": {"id": echo, "status": "invalidRequest", "reason": NONEMPTY}}, cite)

    def raw(self, line: str, expect: Any, *, cite=None) -> None:
        """A literal line whose response is compared only as far as `expect` says (for example its id)."""
        self._append({"raw": line, "expect": expect}, cite)

    def ok_raw(self, raw: str, rid: str, result: Any = None, *, cite=None) -> None:
        """A literal line answered ok that does not change state (a read)."""
        self._append({"raw": raw, "expect": {"id": rid, "status": "ok", "result": ANY if result is None else result}}, cite)

    # --- Reads used as observations ----------------------------------------------

    def read_at(self, at: str | None) -> str:
        """A time for an observation that never regresses (w0 4.4)."""
        if at is not None:
            return at
        if self.last_ok is None:
            raise ValueError("no request answered ok yet")
        return self.last_ok

    def audit_log(self, at: str | None = None, *, cite=None) -> None:
        """Compare the whole expected log (admin read)."""
        self.ok("listAuditEntries", "admin", self.read_at(at), {}, {"entries": [dict(e) for e in self.audit]},
                check=True, cite=cite)

    def audit_mark(self, key: str, at: str | None = None) -> int:
        self.ok("listAuditEntries", "admin", self.read_at(at), {}, {"entries": {"$record": key}}, check=True)
        return len(self.audit)

    def audit_since(self, key: str, mark: int, at: str | None = None, *, cite=None) -> None:
        """The log equals the recorded one plus the entries expected since `mark`, numbered after it."""
        tail = [{**entry, "seq": {"$next": True}} for entry in self.audit[mark:]]
        self.ok("listAuditEntries", "admin", self.read_at(at), {}, {"entries": {"$extends": key, "tail": tail}},
                check=True, cite=cite)

    def _snapshot_label(self) -> str:
        self.snapshots += 1
        return f"snap{self.snapshots}"

    def _snapshot(self, label: str, at: str, mode: str) -> None:
        """Admin reads of every known record; '$same' compares with the '$record' pass."""
        at = at if self.last_ok is None or at >= self.last_ok else self.last_ok
        reads: list[tuple[str, dict[str, Any]]] = [("listMachines", {}), ("listMaterials", {}), ("listIncidents", {})]
        for member_id in sorted(self.registered):
            for op in ("getMember", "listCertifications", "listMemberReservations", "listInvoices", "getAccount"):
                reads.append((op, {"memberId": member_id}))
        for machine_id in self.machines:
            for date in sorted(self.dates):
                reads.append(("getMachineSchedule", {"machineId": machine_id, "date": date}))
        for index, (op, args) in enumerate(reads):
            rid = self._rid()
            request = {"id": rid, "op": op, "args": {"actor": "admin", "at": at, **args}}
            self.steps.append({"request": request, "snapshot": True,
                               "expect": {"id": rid, "status": "ok", "result": {mode: f"{label}-{index}"}}})
        self.last_ok = at

    def build(self) -> dict[str, Any]:
        case = {"id": self.id, "families": self.families, "cite": self.cites, "steps": self.steps}
        if self.invariants:
            case["invariants"] = list(self.invariants)
        return case


def build_all(builders: list[Callable[[str], list[dict[str, Any]]]], step: str) -> list[dict[str, Any]]:
    cases: list[dict[str, Any]] = []
    for make in builders:
        cases.extend(make(step))
    return cases
