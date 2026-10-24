"""Phase B cases against controlled transcripts, never a library."""

import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest

ORACLE = Path(__file__).resolve().parents[1] / "library-oracle"
sys.path.insert(0, str(ORACLE))

import assess  # noqa: E402  (the oracle's assessment plumbing)
from driver import Limits  # noqa: E402
import oracle  # noqa: E402
from test_oracle import expected_transcript  # noqa: E402

import phase_b_cases  # noqa: E402


def find(cases, case_id):
    return next(c for c in cases if c["id"] == case_id)


def args(step):
    return step.get("request", {}).get("args", {})


def op(step):
    request = oracle._request_for_expectation(step) or {}
    return request.get("op")


def contains_status(value):
    if isinstance(value, dict):
        member = value.get("member")
        return (isinstance(member, dict) and "status" in member) or any(contains_status(v) for v in value.values())
    if isinstance(value, list):
        return any(contains_status(v) for v in value)
    return False


class PhaseBCaseTests(unittest.TestCase):
    def evaluate(self, case, responses):
        captures, errors = {}, []
        for step, response in zip(case["steps"], responses):
            errors += oracle.evaluate_step(step, response, captures)["errors"]
        return errors

    def test_expected_transcripts_pass_for_every_profile(self):
        for profile in phase_b_cases.PROFILES:
            for case in phase_b_cases.build_cases(profile):
                with self.subTest(profile=profile, case=case["id"]):
                    self.assertEqual(self.evaluate(case, expected_transcript(case)), [])

    def test_families_accumulate_and_the_removal_is_applied(self):
        ids = {p: {c["id"] for c in phase_b_cases.build_cases(p)} for p in phase_b_cases.PROFILES}
        carried = {c["id"] for c in oracle.build_cases("m2")}
        self.assertTrue(carried <= ids["m3"] and carried <= ids["m4"])
        self.assertTrue({"T1-loan-class", "T1-short-loans", "T1-due-date-range-by-class"} <= ids["m3"])
        self.assertFalse(any(i.startswith("T2") for i in ids["m3"]))
        self.assertIn("T2-renew-suspended", ids["m4"])
        self.assertNotIn("T2-renew-suspended", ids["m5"])
        self.assertNotIn("E05-suspension", ids["m5"])
        self.assertIn("T3-no-member-status", ids["m5"])
        for case in phase_b_cases.build_cases("m5"):
            with self.subTest(case=case["id"]):
                calls = [s for s in case["steps"] if op(s) == "setMemberStatus"]
                self.assertTrue(all(s["expect"]["status"] == "invalidRequest" for s in calls))
                self.assertFalse(any(contains_status(s["expect"]) for s in case["steps"]))
        self.assertFalse(any(contains_status(s["expect"]) for s in phase_b_cases.witness("m5")["steps"]))

    def assert_detects(self, profile, case_id, predicate, wrong):
        case = find(phase_b_cases.build_cases(profile), case_id)
        responses = expected_transcript(case)
        index = next(i for i, s in enumerate(case["steps"]) if not s.get("snapshot") and predicate(s))
        responses[index] = wrong(copy.deepcopy(responses[index]))
        self.assertTrue(self.evaluate(case, responses), case_id)

    def test_wrong_behavior_is_detected(self):
        def due(value):
            def change(r):
                r["result"]["loan"]["dueDate"] = value
                return r
            return change
        # T1: a short copy lent for fourteen days; a standard copy past 9999-12-31 admitted.
        self.assert_detects("m3", "T1-short-loans", lambda s: args(s).get("copyId") == "short-1" and op(s) == "borrow", due("2026-10-15"))
        self.assert_detects("m3", "T1-due-date-range-by-class",
                            lambda s: op(s) == "borrow" and args(s).get("copyId") == "std-1" and args(s).get("borrowDate") == "9999-12-24",
                            lambda r: {"id": r["id"], "status": "ok", "result": {"loan": {"loanId": "x", "dueDate": "10000-01-07"}}})
        # T2: overdue assessment that ignores the renewed due date (composition).
        self.assert_detects("m4", "T2-renew-once", lambda s: args(s).get("assessmentDate") == "2026-10-16",
                            lambda r: {**r, "result": {"notices": [{"loanId": "fixed-opaque-loan-1", "memberId": "member-a",
                                                                   "copyId": "std-1", "dueDate": "2026-10-15"}]}})
        # T2: a suspended member's renewal admitted; a second renewal admitted.
        self.assert_detects("m4", "T2-renew-suspended", lambda s: op(s) == "renew" and args(s).get("renewalDate") == "2026-10-05",
                            lambda r: {"id": r["id"], "status": "ok", "result": {"loan": {}}})
        self.assert_detects("m4", "T2-renew-once", lambda s: op(s) == "renew" and args(s).get("renewalDate") == "2026-10-20",
                            lambda r: {"id": r["id"], "status": "ok", "result": {"loan": {}}})
        # T2: the renewal count missing from a loan record.
        def drop_renewals(r):
            del r["result"]["loan"]["renewals"]
            return r
        self.assert_detects("m4", "T2-renew-once", lambda s: op(s) == "inspectLoan", drop_renewals)
        # T3: setMemberStatus still working.
        self.assert_detects("m5", "T3-no-member-status", lambda s: op(s) == "setMemberStatus",
                            lambda r: {"id": r["id"], "status": "ok", "result": {"member": {"memberId": "member-a", "name": "Ada"}}})

    def test_discrimination_needs_the_three_loan_limit(self):
        for profile in ("m3", "m5"):
            with self.subTest(profile=profile), tempfile.TemporaryDirectory() as directory:
                three = phase_b_cases.strip(oracle.witness_case("m2"))
                if profile == "m5":
                    three = phase_b_cases.no_suspension(three)
                data = Path(directory) / "responses.json"
                data.write_text(json.dumps(expected_transcript(three)), encoding="utf-8")
                script = Path(directory) / "replay.py"
                script.write_text("import json,sys\nrows=json.load(open(sys.argv[1]))\n"
                                  "for line,row in zip(sys.stdin,rows):\n print(json.dumps(row),flush=True)\n", encoding="utf-8")
                command = [sys.executable, "-I", str(script), str(data)]
                result = phase_b_cases.discrimination(profile, command, directory, Limits(), assess.run_case)
                self.assertTrue(result["expected_mismatch_confirmed"], result["errors"][:3])

    def test_oracle_transport_runs_phase_b_cases(self):
        case = find(phase_b_cases.build_cases("m4"), "T2-renew-once")
        with tempfile.TemporaryDirectory() as directory:
            data = Path(directory) / "responses.json"
            data.write_text(json.dumps(expected_transcript(case)), encoding="utf-8")
            script = Path(directory) / "replay.py"
            script.write_text("import json,sys\nrows=json.load(open(sys.argv[1]))\n"
                              "for line,row in zip(sys.stdin,rows):\n print(json.dumps(row),flush=True)\n", encoding="utf-8")
            result = assess.run_case(case, [sys.executable, "-I", str(script), str(data)], directory, Limits())
            self.assertTrue(result["passed"], result["errors"][:3])


if __name__ == "__main__":
    unittest.main()
