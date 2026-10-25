"""Offline self-tests of the Makerspace oracle: no program, no engine.

Expected transcripts are materialized from the cases themselves and replayed
through the matcher (in process) and through the reused Library transport
(tiny replay processes). They show the oracle's mechanics and its traceability,
not a realized Makerspace.
"""

from __future__ import annotations

import copy
import json
from pathlib import Path
import re
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

import behavior  # noqa: E402
import catalog  # noqa: E402
import census  # noqa: E402
import facts  # noqa: E402
import harness  # noqa: E402
import invariants  # noqa: E402
import probes  # noqa: E402

REQUIREMENTS = HERE.parents[1] / "tools" / "comparison" / "phase-c" / "requirements"
# The family list sent to the coordinator and session I on 25 October 2026; P-LIMITS added with w0 2.8.
SENT_FAMILIES = ["P-ENVELOPE", "P-ARGUMENTS", "P-CHECK-ORDER", "P-RULE-ORDER", "P-IDENTIFIERS", "P-NO-MUTATION",
                 "P-AUTHZ-ROLE", "P-AUTHZ-OWN", "P-AUTHZ-ALL", "P-AUDIT-ACCEPTED", "P-AUDIT-SILENT", "P-AUDIT-FORBIDDEN",
                 "P-MONEY-ROUNDING", "P-MONEY-USAGE", "P-MONEY-COURSE", "P-MONEY-MATERIALS", "P-MONEY-INVOICE",
                 "P-MONEY-CREDIT", "P-MONEY-REVENUE", "P-MONEY-LOCKOUT", "P-TIME-ORDER", "P-TIME-FORMAT", "P-TIME-HOURS",
                 "P-TIME-GRANULARITY", "P-TIME-INTERVALS", "P-TIME-BOOKING-STATES", "P-TIME-CERTIFICATION",
                 "P-TIME-BILLING", "P-ELIGIBILITY", "P-REPORT-UTILIZATION", "P-LIMITS"]
_CASES: dict[str, list] = {}


def cases(step: str) -> list[dict]:
    if step not in _CASES:
        _CASES[step] = catalog.build_cases(step)
    return _CASES[step]


def materialize(value, memory, counter):
    """A response that satisfies an expectation exactly (tokens filled deterministically)."""
    if isinstance(value, dict):
        token = harness._token(value)
        if token == "$nonempty":
            return "Controlled reason"
        if token == "$any":
            return "any value"
        if token in ("$record", "$same"):
            return ["state", value[token]]
        if token in ("$ref", "$calc"):
            return harness.resolve(value, memory)
        if token == "$capture":
            counter[0] += 1
            memory[("ref", value["$capture"])] = 1000 + counter[0]
            return 1000 + counter[0]
        if token == "$filter":
            where = harness.resolve(value["$filter"], memory)
            fixed = {k: v[0] if isinstance(v, list) else v for k, v in where.items()}
            return [{**fixed, **item} if isinstance(item, dict) else item
                    for item in (materialize(i, memory, counter) for i in value["items"])]
        if token == "$extends":
            tail = [materialize(item, memory, counter) for item in value["tail"]]
            for offset, item in enumerate(tail):
                if isinstance(item, dict) and item.get("seq") == {"$next": True}:
                    item["seq"] = 2 + offset + 1
            return ["state", value["$extends"]] + tail
        if token == "$has":
            return {k: materialize(v, memory, counter) for k, v in value["$has"].items()}
        return {k: materialize(v, memory, counter) for k, v in value.items()}
    if isinstance(value, list):
        return [materialize(v, memory, counter) for v in value]
    return value


def put(node, path, value):
    """Place a captured value at a capture path, creating what is missing."""
    if not path:
        return value
    head, rest = path[0], path[1:]
    if isinstance(head, str):
        node = node if isinstance(node, dict) else {}
        node[head] = put(node.get(head), rest, value)
        return node
    node = node if isinstance(node, list) else []
    for index, item in enumerate(node):
        if isinstance(item, dict) and all(item.get(k) == v for k, v in head.items()):
            node[index] = put(item, rest, value)
            return node
    node.append(put(dict(head), rest, value))
    return node


def transcript(case: dict) -> list:
    memory: dict = {}
    counter = [0]
    rows = []
    for step in case["steps"]:
        for name in step.get("capture", {}):
            memory[("ref", name)] = f"captured-{name}"
        response = materialize(step["expect"], memory, counter)
        for name, path in step.get("capture", {}).items():
            response = put(response, path, memory[("ref", name)])
        rows.append(response)
    return rows


def headings(text: str) -> list[tuple[int, int, str]]:
    found = []
    for match in re.finditer(r"^(#+) (.+)$", text, re.M):
        found.append((match.start(), len(match.group(1)), match.group(2).strip()))
    return found


def cited_text_in_section(cite: dict) -> bool:
    text = (REQUIREMENTS / cite["doc"]).read_text(encoding="utf-8")
    heads = headings(text)
    for i, (pos, level, title) in enumerate(heads):
        if title == cite["section"] or title.startswith(cite["section"] + " "):
            end = next((p for p, lv, _ in heads[i + 1:] if lv <= level), len(text))
            if cite["text"] in text[pos:end]:
                return True
    return False


class StructureTests(unittest.TestCase):
    def test_family_list_matches_the_sent_list(self):
        self.assertEqual([name for name, *_ in probes.FAMILIES], SENT_FAMILIES)

    def test_every_case_and_cited_step_traces_to_a_verbatim_sentence(self):
        cites = {}
        for step in facts.STEPS:
            for case in cases(step):
                self.assertTrue(case["cite"], case["id"])
                for cite in case["cite"] + [c for s in case["steps"] for c in s.get("cite", [])]:
                    cites[json.dumps(cite, sort_keys=True)] = cite
        missing = [c for c in cites.values() if not cited_text_in_section(c)]
        self.assertEqual(missing, [])

    def test_steps_are_cumulative_and_identifiers_unique(self):
        previous: set[str] = set()
        for step in facts.STEPS:
            ids = [case["id"] for case in cases(step)]
            self.assertEqual(len(ids), len(set(ids)), step)
            self.assertTrue(previous <= set(ids), f"{step} drops {previous - set(ids)}")
            previous = set(ids)

    def test_every_operation_and_policy_area_is_probed(self):
        for step in ("W0", "C2", "C4", "C8"):
            ops = facts.ops(step)
            requested = {(s.get("request") or {}).get("op") for case in probes.build_probes(step) for s in case["steps"]}
            self.assertEqual(set(ops) - requested, set(), step)

    def test_spec_examples_and_formulas(self):
        self.assertEqual(facts.line_amount(37, 45, 10), 1499)
        self.assertEqual([facts.line_amount(9, 199, 0), facts.line_amount(10, 199, 5), facts.line_amount(25, 37, 12)], [1791, 1891, 814])
        self.assertEqual([facts.bulk_discount("C3", q) for q in (1, 9, 10, 24, 25, 10000)], [0, 0, 5, 5, 12, 12])
        self.assertEqual(facts.bulk_discount("C2", 25), 0)
        self.assertEqual([facts.percent(18, 720), facts.percent(9, 720), facts.percent(30, 0), facts.percent(21, 840)], [3, 1, 0, 3])
        self.assertEqual(facts.opening_hours("W0")[5], ("10:00", "18:00"))
        self.assertIsNone(facts.opening_hours("C7")[6])
        self.assertEqual(facts.opening_hours("C8")[6], ("12:00", "16:00"))
        self.assertEqual(facts.ops("W0")["listAuditEntries"]["perms"], {"member": "none", "staff": "none", "admin": "all"})
        self.assertEqual(sum(1 for p in facts.ops("C4").values() if p["perms"]["steward"] == "all"), 9)
        self.assertEqual(len(facts.ops("W0")), 37)
        self.assertEqual(len(facts.ops("C2")), 38)

    def test_records_follow_the_step(self):
        self.assertIn("expiresOn", facts.certification("C5", "a", "f", "2026-03-02", "2027-03-02", "valid"))
        self.assertNotIn("expiresOn", facts.certification("C6", "a", "f", "2026-03-02", "2027-03-02", "valid"))
        self.assertNotIn("blocked", facts.account("C4", "a"))
        self.assertIn("blocked", facts.account("C5", "a"))
        self.assertEqual(set(facts.audit_entry("W0", 1, "t", "a", "admin", "op", "x")), {"seq", "at", "actor", "op", "target"})
        self.assertIn("lockouts", facts.schedule("C2", "m", "d", None, [], []))
        self.assertIn("lockoutMinutes", facts.util_row("C2", "m", 1, 0, 0))


class ExpectationTests(unittest.TestCase):
    def test_materialized_transcripts_satisfy_every_case(self):
        for step in facts.STEPS:
            for case in cases(step):
                memory: dict = {}
                for index, (s, response) in enumerate(zip(case["steps"], transcript(case))):
                    result = harness.evaluate_step(s, response, memory)
                    self.assertEqual(result["errors"], [], f"{step} {case['id']} step {index}")

    def test_every_case_detects_a_changed_response(self):
        for case in cases("C8"):
            responses = transcript(case)
            index = next((i for i, s in enumerate(case["steps"]) if s["expect"].get("status") == "refused"), len(responses) - 1)
            wrong = copy.deepcopy(responses[index])
            if wrong.get("status") == "refused":
                wrong["code"] = "not_a_code"
            else:
                wrong["status"] = "refused"
            memory: dict = {}
            for s, response in zip(case["steps"][:index], responses[:index]):
                harness.evaluate_step(s, response, memory)
            self.assertTrue(harness.evaluate_step(case["steps"][index], wrong, memory)["errors"], case["id"])

    def test_strict_cases_satisfy_the_invariants(self):
        for step in facts.STEPS:
            for case in behavior.build_behavior(step):
                pairs = list(zip(case["steps"], transcript(case)))
                for name, check in invariants.CHECKS.items():
                    self.assertEqual(check({**case, "step": step}, pairs), [], f"{step} {case['id']} {name}")

    def test_census_selects_only_cases_the_change_leaves_alone(self):
        def render(case):
            return [(s.get("request"), s.get("raw"), s["expect"]) for s in case["steps"]]
        for step, start in (("C3", "C2"), ("C7", "C6"), ("C9", "C8")):
            before = {case["id"]: case for case in cases(start)}
            selected = census.census_cases(step)
            self.assertTrue(selected)
            for case in selected:
                self.assertIn(case["id"], before)
                self.assertEqual(render(case), render(before[case["id"]]), f"{step} {case['id']}")
        self.assertEqual(len(census.census_cases("C7")), len(cases("C7")))
        self.assertTrue(any(census._bulk(c) for c in cases("C3")))
        self.assertEqual(census.census_cases("C2"), [])

    def test_census_compare_ignores_reason_only_where_allowed(self):
        a = [{"id": "x", "status": "refused", "code": "forbidden", "reason": "One"}]
        b = [{"id": "x", "status": "refused", "code": "forbidden", "reason": "Two"}]
        self.assertEqual(census.compare("C3", a, b), [])
        self.assertEqual(len(census.compare("C9", a, b)), 1)
        self.assertEqual(len(census.compare("C7", a, [dict(b[0], code="not_found")])), 1)


REPLAY = r"""
import json, sys
rows = json.load(open(sys.argv[1], encoding="utf-8"))
for line, row in zip(sys.stdin, rows):
    sys.stdout.write(row + "\n")
    sys.stdout.flush()
"""


class ReplayTests(unittest.TestCase):
    """The reused Library transport carries a materialized transcript; a planted fault fails the case."""

    def replay(self, case, responses, checks=None):
        with tempfile.TemporaryDirectory() as directory:
            script = Path(directory) / "replay.py"
            script.write_text(REPLAY, encoding="utf-8")
            rows = Path(directory) / "rows.json"
            rows.write_text(json.dumps([json.dumps(r, ensure_ascii=False) for r in responses]), encoding="utf-8")
            limits = harness.Limits(**harness.LIMITS)
            return harness.run_case(case, [sys.executable, "-I", str(script), str(rows)], directory, limits,
                                    checks=catalog.CHECKS if checks is None else checks)

    def test_driver_is_the_library_oracles(self):
        self.assertEqual(Path(harness.driver.__file__).resolve(), (HERE.parent / "library-oracle" / "driver.py").resolve())

    def test_cases_pass_through_the_transport(self):
        chosen = {"B-BILLING-worked-example", "P-ENVELOPE-lines", "P-AUDIT-FORBIDDEN-refusal-entries", "D-C2-lockout"}
        for case in cases("C4"):
            if case["id"] in chosen:
                result = self.replay(case, transcript(case))
                self.assertTrue(result["passed"], (case["id"], result["errors"][:3]))

    def test_planted_fault_fails_at_its_step(self):
        case = next(c for c in cases("W0") if c["id"] == "B-BILLING-worked-example")
        responses = transcript(case)
        index = next(i for i, s in enumerate(case["steps"]) if (s.get("request") or {}).get("op") == "generateInvoices"
                     and s["expect"]["status"] == "ok" and s["request"]["args"]["month"] == "2026-03")
        responses[index]["result"]["invoices"][0]["totalCents"] += 1
        result = self.replay(case, responses)
        self.assertFalse(result["passed"])
        self.assertEqual(result["steps"][0]["index"], index)

    def test_checks_only_cases_fail_only_on_their_check(self):
        step = {"request": {"id": "a", "op": "listMachines", "args": {}}, "expect": {"id": "a", "status": "ok", "result": {"machines": []}}}
        case = {"id": "x", "families": ["F"], "steps": [step], "invariants": ["NONE"], "checks_only": True}
        wrong = [{"id": "a", "status": "refused", "code": "forbidden", "reason": "x"}]
        result = self.replay(case, wrong, {"NONE": lambda case, transcript: []})
        self.assertTrue(result["passed"])
        self.assertTrue(result["step_mismatches"])
        self.assertFalse(self.replay({**case, "checks_only": False}, wrong, {"NONE": lambda case, transcript: []})["passed"])
        scenario = invariants.scenario("C2")
        self.assertEqual([c["families"] for c in scenario][0], ["INV-SCENARIO"])
        self.assertTrue(all(c.get("checks_only") and c["invariants"] == c["families"] for c in scenario[1:]))

    def test_invariant_scenario_runs_its_check(self):
        case = next(c for c in cases("C2") if c["id"] == "INV-AUDIT-ORDER-scenario")
        result = self.replay(case, transcript(case))
        self.assertFalse(result["passed"])  # "any value" placeholders are no audit log
        self.assertTrue(any(e.startswith("invariant INV-AUDIT-ORDER") for e in result["errors"]), result["errors"][:3])


if __name__ == "__main__":
    unittest.main()
