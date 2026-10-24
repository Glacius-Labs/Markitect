"""Probe expectations against controlled transcripts, never a library."""

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

import probes  # noqa: E402


def case(case_id):
    return next(c for c in probes.build_probes() if c["id"] == case_id)


def index_of(c, predicate):
    return next(i for i, step in enumerate(c["steps"]) if not step.get("snapshot") and predicate(step))


def request_args(step):
    return step.get("request", {}).get("args", {})


class ProbeTests(unittest.TestCase):
    def evaluate(self, c, responses):
        captures, errors = {}, []
        for step, response in zip(c["steps"], responses):
            errors += oracle.evaluate_step(step, response, captures)["errors"]
        return errors

    def test_expected_transcripts_pass(self):
        cases = probes.build_probes()
        self.assertEqual([c["id"][:3] for c in cases], ["X01", "X02", "X03", "X04", "X05", "X06"])
        for c in cases:
            with self.subTest(c["id"]):
                self.assertEqual(self.evaluate(c, expected_transcript(c)), [])

    def test_refusals_and_invalid_requests_are_bracketed_by_state_reads(self):
        for c in probes.build_probes():
            with self.subTest(c["id"]):
                plain = [s for s in c["steps"] if not s.get("snapshot")]
                self.assertGreater(len(c["steps"]), len(plain))

    def assert_detects(self, case_id, predicate, wrong):
        c = case(case_id)
        responses = expected_transcript(c)
        position = index_of(c, predicate)
        responses[position] = wrong(copy.deepcopy(responses[position]))
        self.assertTrue(self.evaluate(c, responses), case_id)

    def test_wrong_behavior_is_detected(self):
        loan = {"loanId": "l1", "memberId": "member-x", "copyId": "copy-x", "borrowDate": "9999-12-18",
                "dueDate": "10000-01-01", "status": "open", "returnDate": None}
        self.assert_detects("X01-due-date-range", lambda s: request_args(s).get("borrowDate") == "9999-12-18",
                            lambda r: {"id": r["id"], "status": "ok", "result": {"loan": loan}})
        self.assert_detects("X02-case-variant-field", lambda s: '"MemberId"' in s.get("raw", ""),
                            lambda r: {**r, "status": "refused"})
        self.assert_detects("X03-null-argument", lambda s: request_args(s).get("memberId") == "member-null" and "name" in request_args(s),
                            lambda r: {"id": r["id"], "status": "ok", "result": {"member": {"memberId": "member-null", "name": None, "status": "active"}}})
        self.assert_detects("X04-missing-args", lambda s: "no-args" in s.get("raw", ""),
                            lambda r: {**r, "status": "refused"})
        self.assert_detects("X05-null-id-echo", lambda s: s.get("raw", "").startswith('{"id":7'),
                            lambda r: {**r, "id": 7})
        self.assert_detects("X06-year-range", lambda s: request_args(s).get("assessmentDate") == "0000-12-31",
                            lambda r: {"id": r["id"], "status": "ok", "result": {"notices": []}})

    def replay(self, directory, responses):
        data = Path(directory) / "responses.json"
        data.write_text(json.dumps(responses), encoding="utf-8")
        script = Path(directory) / "replay.py"
        script.write_text("import json,sys\nrows=json.load(open(sys.argv[1]))\n"
                          "for line,row in zip(sys.stdin,rows):\n print(json.dumps(row),flush=True)\n", encoding="utf-8")
        return [sys.executable, "-I", str(script), str(data)]

    def test_oracle_transport_runs_probe_cases(self):
        c = case("X05-null-id-echo")
        with tempfile.TemporaryDirectory() as directory:
            good = assess.run_case(c, self.replay(directory, expected_transcript(c)), directory, Limits())
            self.assertTrue(good["passed"], good["errors"])
            wrong = expected_transcript(c)
            wrong[index_of(c, lambda s: s.get("raw") == '["inspectMember"]')]["status"] = "refused"
            bad = assess.run_case(c, self.replay(directory, wrong), directory, Limits())
            self.assertFalse(bad["passed"])


if __name__ == "__main__":
    unittest.main()
