"""Matcher and invariant-check mechanics on controlled values; no program."""

from __future__ import annotations

import copy
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))

import facts as f  # noqa: E402
import harness  # noqa: E402
import invariants  # noqa: E402


def check(expect, response, memory=None):
    return harness.evaluate_step({"expect": expect}, response, {} if memory is None else memory)["errors"]


class MatcherTests(unittest.TestCase):
    def test_members_are_exact(self):
        self.assertEqual(check({"a": 1}, {"a": 1}), [])
        self.assertTrue(check({"a": 1}, {"a": 1, "b": 2}))
        self.assertTrue(check({"a": 1, "b": 2}, {"a": 1}))

    def test_json_types_are_strict(self):
        self.assertTrue(check({"a": 1}, {"a": True}))
        self.assertTrue(check({"a": 1}, {"a": 1.0}))
        self.assertTrue(check({"a": None}, {"a": 0}))
        self.assertTrue(check({"a": []}, {"a": None}))

    def test_lists_are_ordered(self):
        self.assertTrue(check({"a": [1, 2]}, {"a": [2, 1]}))
        self.assertTrue(check({"a": [1]}, {"a": [1, 1]}))

    def test_tokens(self):
        self.assertEqual(check({"r": harness.NONEMPTY}, {"r": " x "}), [])
        self.assertTrue(check({"r": harness.NONEMPTY}, {"r": " \t"}))
        self.assertTrue(check({"r": harness.NONEMPTY}, {"r": 5}))
        self.assertEqual(check({"r": harness.ANY}, {"r": {"deep": [1]}}), [])
        self.assertTrue(check({"r": harness.ANY}, {}))
        self.assertEqual(check({"$has": {"a": 1}}, {"a": 1, "b": 2}), [])
        self.assertTrue(check({"$has": {"a": 1}}, {"b": 2}))

    def test_record_same_and_extends(self):
        memory: dict = {}
        self.assertEqual(check({"x": {"$record": "k"}}, {"x": [{"seq": 1}]}, memory), [])
        self.assertEqual(check({"x": {"$same": "k"}}, {"x": [{"seq": 1}]}, memory), [])
        self.assertTrue(check({"x": {"$same": "k"}}, {"x": [{"seq": 2}]}, memory))
        self.assertTrue(check({"x": {"$same": "missing"}}, {"x": 1}, memory))
        tail = [{"seq": {"$next": True}, "op": "a"}]
        self.assertEqual(check({"x": {"$extends": "k", "tail": tail}}, {"x": [{"seq": 1}, {"seq": 2, "op": "a"}]}, memory), [])
        self.assertTrue(check({"x": {"$extends": "k", "tail": tail}}, {"x": [{"seq": 1}, {"seq": 3, "op": "a"}]}, memory))
        self.assertTrue(check({"x": {"$extends": "k", "tail": tail}}, {"x": [{"seq": 9}, {"seq": 2, "op": "a"}]}, memory))
        self.assertTrue(check({"x": {"$extends": "k", "tail": []}}, {"x": [{"seq": 1}, {"seq": 2}]}, memory))

    def test_non_object_response(self):
        self.assertTrue(harness.evaluate_step({"expect": {}}, [1], {})["errors"])

    def test_family_summary_counts_failed_families(self):
        rows = harness.family_summary([{"families": ["A"], "passed": True}, {"families": ["A", "B"], "passed": False}])
        self.assertEqual(rows, {"A": {"cases": 2, "failed": 1}, "B": {"cases": 1, "failed": 1}})


def req(op, actor="sam", at="2026-03-02T09:00Z", **args):
    return {"request": {"id": "q", "op": op, "args": {"actor": actor, "at": at, **args}}}


def ok(result):
    return {"id": "q", "status": "ok", "result": result}


G = "2026-04-01T09:00Z"


def billing_transcript(step="C3"):
    """A small consistent history: registration, purchase, check-out, invoice and the final log."""
    usage = f.charge("usage", "res-1", 10, 45, 10, "2026-03-03T10:10Z")
    material = f.charge("material", "pur-1", 10, 199, f.bulk_discount(step, 10), "2026-03-02T09:00Z")
    inv = f.invoice("inv-1", "ada", "2026-03", G, "2026-04-15T09:00Z", [f.charge("membership", "plus", 1, 4900, 0, G), material, usage])
    entries = [f.audit_entry(step, n, at, actor, role, op, target) for n, (at, actor, role, op, target) in enumerate([
        ("2026-03-02T09:00Z", "admin", "admin", "registerUser", "sam"), ("2026-03-02T09:00Z", "sam", "staff", "registerMember", "ada"),
        ("2026-03-02T09:00Z", "sam", "staff", "purchaseMaterial", "pur-1"), ("2026-03-02T09:00Z", "sam", "staff", "createReservation", "res-1"),
        ("2026-03-03T10:00Z", "sam", "staff", "checkIn", "res-1"), ("2026-03-03T10:10Z", "sam", "staff", "checkOut", "res-1"),
        (G, "sam", "staff", "generateInvoices", "2026-03")], 1)]
    res = f.reservation("res-1", "ada", "L1", "2026-03-03T10:00Z", "2026-03-03T10:30Z")
    return [(req("registerUser", "admin", userId="sam", name="Sam", role="staff"), ok({"user": f.user("sam", "Sam", "staff")})),
            (req("registerMember", memberId="ada", name="Ada", tier="plus"), ok({"member": f.member("ada", "Ada", "plus", "2026-03-02T09:00Z")})),
            (req("purchaseMaterial", memberId="ada", materialId="acrylic", quantity=10),
             ok({"purchase": f.purchase("pur-1", "ada", "acrylic", 10, 199, f.bulk_discount(step, 10), "2026-03-02T09:00Z"), "material": {}})),
            (req("createReservation", memberId="ada", machineId="L1", start=res["start"], end=res["end"]), ok({"reservation": res})),
            (req("checkIn", at="2026-03-03T10:00Z", reservationId="res-1"), ok({"reservation": res})),
            (req("checkOut", at="2026-03-03T10:10Z", reservationId="res-1"), ok({"reservation": res, "charge": usage})),
            (req("generateInvoices", at=G, month="2026-03"), ok({"month": "2026-03", "invoices": [inv]})),
            (req("listInvoices", at=G, memberId="ada"), ok({"memberId": "ada", "invoices": [inv]})),
            (req("getAccount", at=G, memberId="ada"), ok({"account": f.account(step, "ada", open_=inv["balanceCents"])})),
            (req("listAuditEntries", "admin", at=G), ok({"entries": entries}))]


class InvariantTests(unittest.TestCase):
    """Every check passes a consistent history and catches one planted inconsistency."""

    def run_checks(self, transcript, step="C3"):
        case = {"step": step}
        return {name: check(case, transcript) for name, check in invariants.CHECKS.items()}

    def test_consistent_history_passes(self):
        for step in ("W0", "C3", "C8"):
            self.assertEqual({k: v for k, v in self.run_checks(billing_transcript(step), step).items() if v}, {})

    def plant(self, edit):
        transcript = copy.deepcopy(billing_transcript())
        edit(transcript)
        return transcript

    def test_each_check_catches_its_fault(self):
        def invoice(t):
            return t[6][1]["result"]["invoices"][0]
        faults = {
            "INV-AUDIT-ORDER": lambda t: t[-1][1]["result"]["entries"].pop(),
            "INV-CHARGE-AUDIT": lambda t: t[-1][1]["result"]["entries"].pop(5),
            "INV-MONEY": lambda t: invoice(t)["lines"][2].__setitem__("amountCents", 999),
            "INV-DISCOUNTS": lambda t: t[2][1]["result"]["purchase"].__setitem__("discountPercent", 10),
            "INV-SLOTS": lambda t: t[3][1]["result"]["reservation"].__setitem__("end", "2026-03-03T10:45Z"),
            "INV-COUNTERS": lambda t: t[3][1]["result"]["reservation"].__setitem__("reservationId", "res-2"),
            "INV-ACCOUNTS": lambda t: t[8][1]["result"]["account"].__setitem__("openBalanceCents", 1),
            "INV-BILLING-ONCE": lambda t: invoice(t)["lines"][1].__setitem__("at", "2026-02-27T09:00Z"),
        }
        for name, edit in faults.items():
            found = self.run_checks(self.plant(edit))[name]
            self.assertTrue(found, name)

    def test_overlap_and_lockout_checks(self):
        a = f.reservation("res-1", "ada", "L1", "2026-03-04T10:00Z", "2026-03-04T11:00Z")
        b = f.reservation("res-2", "bob", "L1", "2026-03-04T10:30Z", "2026-03-04T11:30Z")
        lock = f.lockout("lock-1", "inc-1", "L1", "2026-03-04T10:00Z", "2026-03-04T12:00Z", ["res-1"])
        sched = [(req("getMachineSchedule", machineId="L1", date="2026-03-04"),
                  ok(f.schedule("C2", "L1", "2026-03-04", ("09:00", "21:00"), [a, b], [], [lock])))]
        self.assertTrue(invariants.check_no_overlap({"step": "C2"}, sched))
        cancelled = dict(a, status="cancelled")
        good = [(req("lockMachine", incidentId="inc-1", start=lock["start"], end=lock["end"]), ok({"lockout": lock, "cancelledReservations": [cancelled]}))]
        self.assertEqual(invariants.check_lockouts({"step": "C2"}, good), [])
        bad = copy.deepcopy(good)
        bad[0][1]["result"]["cancelledReservations"][0]["status"] = "reserved"
        self.assertTrue(invariants.check_lockouts({"step": "C2"}, bad))
        later = good + [(req("createReservation", memberId="cy", machineId="L1", start=b["start"], end=b["end"]), ok({"reservation": b}))]
        self.assertTrue(invariants.check_lockouts({"step": "C2"}, later))

    def test_identical_lines_of_different_members_are_not_double_billing(self):
        course = f.charge("course", "laser-101", 1, 3000, 0, "2026-03-02T08:00Z")
        a = f.invoice("inv-1", "ada", "2026-03", G, "2026-04-15T09:00Z", [f.charge("membership", "plus", 1, 4900, 0, G), course])
        b = f.invoice("inv-2", "bob", "2026-03", G, "2026-04-15T09:00Z", [f.charge("membership", "basic", 1, 2500, 0, G), course])
        t = [(req("generateInvoices", at=G, month="2026-03"), ok({"month": "2026-03", "invoices": [a, b]}))]
        self.assertEqual(invariants.check_billing_once({"step": "W0"}, t), [])
        again = f.invoice("inv-3", "ada", "2026-03", G, "2026-04-15T09:00Z", [f.charge("membership", "plus", 1, 4900, 0, G), course])
        t.append((req("listInvoices", at=G, memberId="ada"), ok({"memberId": "ada", "invoices": [again]})))
        self.assertTrue(invariants.check_billing_once({"step": "W0"}, t))

    def test_forbidden_refusals_enter_the_order_from_c1(self):
        refused = (req("listAuditEntries", "ada", at=G), {"id": "q", "status": "refused", "code": "forbidden", "reason": "No"})
        for step, logged in (("W0", False), ("C1", True)):
            t = billing_transcript(step)
            t.insert(-1, refused)
            self.assertEqual(bool(invariants.check_audit_order({"step": step}, t)), logged, step)
            if logged:
                t[-1][1]["result"]["entries"].append(f.audit_entry(step, 8, G, "ada", "member", "listAuditEntries", None, "refused"))
                self.assertEqual(invariants.check_audit_order({"step": step}, t), [])
                t[-1][1]["result"]["entries"][-1]["actorRole"] = "staff"
                self.assertTrue(invariants.check_audit_order({"step": step}, t))

if __name__ == "__main__":
    unittest.main()
