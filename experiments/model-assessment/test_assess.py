"""Provider-free self-tests of assess.py. Run: python -B -m unittest discover -s experiments/model-assessment"""
import json
import unittest
from pathlib import Path

import assess

HERE = Path(__file__).resolve().parent
DEFINITIONS = ["shop/inputs", "shop/system", "shop/list", "shop/storage"]
GAP = {"id": "public-gap", "expected": "needs_clarification", "definitions": DEFINITIONS,
       "issue": {"definitions": ["shop/inputs"], "terms": [r"maximum.{0,20}length"]}}
FREEDOM = {"id": "permitted-freedom", "expected": "accepted", "definitions": DEFINITIONS,
           "freedom": {"definitions": ["shop/storage"], "terms": [r"file format"]}}
CONTRADICTION = {"id": "contradiction", "expected": "model_error", "definitions": DEFINITIONS,
                 "issue": {"definitions": ["shop/inputs", "shop/list"], "terms": [r"case"]}}


def report(status, reason="", questions=(), problems=(), findings=(), passes=None):
    if passes is None:
        passes = [{"id": "p1", "purpose": "assess", "review": True, "sealed": status != "incomplete" or reason == assess.OPEN_FINDINGS_REASON,
                   "verdict": {"verified": "satisfied"}.get(status, "needs_correction" if reason == assess.OPEN_FINDINGS_REASON else "")}]
    return {"status": "passed", "run": {"status": status, "reason": reason, "passes": passes, "questions": list(questions),
                                         "problems": list(problems), "findings": list(findings), "turns": [{"id": "t1"}, {"id": "t2"}],
                                         "usage": {"costUSD": 0.5, "inputTokens": 100, "outputTokens": 10}}}


def question(text, qid="Q1"):
    return {"id": qid, "from": ".", "to": "user", "text": text, "answered": True, "answer": "No owner decision is available."}


def finding(observed, obligation="shop/inputs", blocking=True, closed=False, fid="F1"):
    return {"id": fid, "owner": ".", "raisedBy": ".", "obligation": obligation, "observed": observed, "blocking": blocking, "closed": closed}


ABORT = {"id": "X1", "manager": ".", "category": "model", "text": "shop/inputs folds case but shop/list keeps it.", "abort": True}


class Classification(unittest.TestCase):
    def test_conclusions_follow_the_common_envelope(self):
        cases = {
            "accepted": report("verified"),
            "model_error": report("failed", assess.MODEL_ERROR_REASON + "shop/inputs contradicts shop/list", problems=[ABORT]),
            "needs_clarification": report("incomplete", assess.OPEN_FINDINGS_REASON, questions=[question("Which maximum length?")],
                                          findings=[finding("The maximum length of `shop/inputs` is never stated.")]),
        }
        for want, r in cases.items():
            with self.subTest(want):
                self.assertEqual(assess.classify(r)["conclusion"], want)

    def test_technical_or_unrouted_ends_are_incomplete_not_a_model_judgment(self):
        cases = {
            "deadline": report("incomplete", "the run deadline passed", questions=[question("Which maximum length?")]),
            "unrouted finding": report("incomplete", assess.OPEN_FINDINGS_REASON, findings=[finding("open")]),
            "other failure": report("failed", "a definite error of the run's inputs"),
            "abort by a child": report("failed", assess.MODEL_ERROR_REASON + "x", problems=[dict(ABORT, manager="shop")]),
            "preparation": {"status": "failed"},
        }
        for name, r in cases.items():
            with self.subTest(name):
                self.assertEqual(assess.classify(r)["conclusion"], assess.INCOMPLETE)
        self.assertEqual(assess.classify(cases["unrouted finding"])["incomplete"], "unrouted")
        self.assertEqual(assess.classify(cases["deadline"])["incomplete"], "technical")

    def test_an_orchestrator_finding_is_no_clarification(self):
        gate = dict(finding("file.unexplained: x"), raisedBy="markitect")
        r = report("incomplete", assess.OPEN_FINDINGS_REASON, questions=[question("Which maximum length?")], findings=[gate])
        self.assertEqual(assess.classify(r)["conclusion"], assess.INCOMPLETE)

    def test_a_verified_run_with_a_harness_error_is_not_accepted(self):
        r = report("verified")
        r["status"], r["error"] = "not_completed", "evidence: report could not be written"
        self.assertEqual(assess.classify(r)["conclusion"], assess.INCOMPLETE)


class Scoring(unittest.TestCase):
    def test_a_sourced_question_about_the_gap_meets_the_gap_control(self):
        r = report("incomplete", assess.OPEN_FINDINGS_REASON, questions=[question("[[inputs]] refers to a maximum name length that no Definition states.")],
                   findings=[finding("shop/inputs: the maximum length is open; 40 or 100 would both conform.")])
        s = assess.score(r, assess.classify(r), GAP)
        self.assertTrue(s["expectationMet"])
        self.assertTrue(s["issueRaised"])
        self.assertFalse(s["missedIssue"])
        self.assertEqual(s["falseAlarmCandidates"], [])
        self.assertEqual(s["ownerEffort"]["questions"], 1)

    def test_accepting_the_gap_misses_it(self):
        r = report("verified")
        s = assess.score(r, assess.classify(r), GAP)
        self.assertFalse(s["expectationMet"])
        self.assertTrue(s["missedIssue"])

    def test_a_clarification_about_something_else_is_no_detection(self):
        r = report("incomplete", assess.OPEN_FINDINGS_REASON, questions=[question("Which file format should storage use?")],
                   findings=[finding("The file format is open.", obligation="shop/storage")])
        s = assess.score(r, assess.classify(r), GAP)
        self.assertFalse(s["expectationMet"])
        self.assertTrue(s["missedIssue"])
        self.assertEqual(sorted(s["falseAlarmCandidates"]), ["F1", "Q1"])

    def test_a_question_about_permitted_freedom_is_a_false_alarm(self):
        r = report("incomplete", assess.OPEN_FINDINGS_REASON, questions=[question("Which file format should [[storage]] use?")],
                   findings=[finding("The file format is open.", obligation="shop/storage")])
        s = assess.score(r, assess.classify(r), FREEDOM)
        self.assertFalse(s["expectationMet"])
        self.assertEqual(sorted(s["falseAlarmCandidates"]), ["F1", "Q1"])
        self.assertTrue(all(x["citesFreedom"] for x in s["signals"]))

    def test_a_note_is_no_false_alarm_and_an_accepted_freedom_meets_its_control(self):
        r = report("verified", findings=[finding("The file format is free.", obligation="shop/storage", blocking=False)])
        s = assess.score(r, assess.classify(r), FREEDOM)
        self.assertTrue(s["expectationMet"])
        self.assertEqual(s["falseAlarmCandidates"], [])
        self.assertEqual(s["ownerEffort"]["notes"], 1)

    def test_a_sourced_model_error_meets_the_contradiction_control(self):
        r = report("failed", assess.MODEL_ERROR_REASON + "contradiction", problems=[ABORT])
        s = assess.score(r, assess.classify(r), CONTRADICTION)
        self.assertTrue(s["expectationMet"])
        self.assertEqual(s["uncitedBlocking"], [])

    def test_an_incomplete_attempt_is_not_scored_as_a_judgment(self):
        r = report("incomplete", "the run deadline passed")
        s = assess.score(r, assess.classify(r), GAP)
        self.assertIsNone(s["expectationMet"])
        self.assertFalse(s["missedIssue"])

    def test_citations_need_the_identity_or_a_marked_local_id(self):
        self.assertEqual(assess.cited("see shop/inputs and [[list]]", DEFINITIONS), ["shop/inputs", "shop/list"])
        self.assertEqual(assess.cited("the list of inputs is long", DEFINITIONS), [])
        self.assertEqual(assess.cited("shop/inputs-extra", DEFINITIONS), [])


class Report(unittest.TestCase):
    def test_the_report_is_read_between_the_harness_lines(self):
        text = '{\n "status": "failed",\n "run": {"status": "verified"}\n}\nPASS\n'
        self.assertEqual(assess.load_report(text)["run"]["status"], "verified")
        self.assertEqual(assess.load_report("model assessment exit 1\nFAIL\n"), {})
        self.assertEqual(assess.load_report(""), {})


class Process(unittest.TestCase):
    def test_returns_disputes_and_coverage_come_from_the_events(self):
        events = [
            {"kind": "proposal.submitted", "data": {"proposal": "P2", "kind": "result", "envelope": {"obligations": [
                {"definition": "shop/inputs", "status": "done", "evidence": [{"source": ".markitect/model/shop/inputs.yaml"}]},
                {"definition": "shop/list", "status": "done", "evidence": [{"source": "realization/app.py"}]}]}}},
            {"kind": "result.returned", "manager": "shop", "data": {"proposal": "P2", "by": ".", "reason": "unsupported"}},
            {"kind": "proposal.submitted", "data": {"proposal": "P3", "kind": "result", "envelope": {"obligations": [
                {"definition": "shop/inputs", "status": "done", "evidence": [{"source": ".markitect/model/shop/inputs.yaml"}]}]}}},
            {"kind": "result.accepted", "data": {"proposal": "P3"}},
            {"kind": "finding.disputed", "data": {}},
        ]
        p = assess.process(events)
        self.assertEqual(p["returned"], [{"manager": "shop", "by": ".", "reason": "unsupported"}])
        self.assertEqual(p["disputes"], 1)
        self.assertEqual(p["acceptedResultCoverage"], {"obligations": 1, "withModelSource": 1, "open": 0})


class Declaration(unittest.TestCase):
    def test_the_frozen_pass_declaration_is_a_valid_assessment(self):
        declaration = json.loads((HERE / "assessment-pass.json").read_text(encoding="utf-8"))
        self.assertEqual(assess.preflight(declaration), [])

    def test_the_mandate_names_no_control_specific_issue(self):
        # The treatment is written from the assessment questions, not from the controls' answers.
        text = (HERE / "assessment-pass.json").read_text(encoding="utf-8").lower()
        for word in ("name length", "sku", "stockroom", "maximum name", "case-insensitive", "uppercase"):
            self.assertNotIn(word, text)

    def test_invalid_declarations_are_refused(self):
        self.assertTrue(assess.preflight({"purpose": "verify", "mandate": "m", "guidance": "g", "corrections": ["x"]}))
        self.assertTrue(assess.preflight({"purpose": "assess", "mandate": " ", "guidance": "g", "corrections": ["x"]}))
        self.assertTrue(assess.preflight({"purpose": "assess", "mandate": "m", "guidance": "g", "corrections": []}))


class Series(unittest.TestCase):
    def test_series_counts_conclusions_and_agreement_per_control(self):
        met = assess.score(report("verified"), assess.classify(report("verified")), FREEDOM)
        alarm_report = report("incomplete", assess.OPEN_FINDINGS_REASON, questions=[question("Which format?")], findings=[finding("format")])
        alarm = assess.score(alarm_report, assess.classify(alarm_report), FREEDOM)
        summary = assess.series([met, met, alarm])["permitted-freedom"]
        self.assertEqual(summary["attempts"], 3)
        self.assertEqual(summary["met"], 2)
        self.assertEqual(summary["conclusions"], {"accepted": 2, "needs_clarification": 1})
        self.assertFalse(summary["agreement"])
        self.assertEqual(summary["questions"], 1)

    def test_the_ledger_counts_attempts_that_were_never_assessed(self):
        met = assess.score(report("verified"), assess.classify(report("verified")), FREEDOM)
        ledger = ("permitted-freedom C-01 status=done conclusion=accepted charged=200\n"
                  "permitted-freedom C-05 status=running charged=600\n"
                  "public-gap C-02 status=done conclusion=bind-failed charged=12\n")
        summary = assess.series([met], ledger)
        self.assertEqual((summary["permitted-freedom"]["ledgerAttempts"], summary["permitted-freedom"]["unassessed"]), (2, 1))
        self.assertEqual((summary["public-gap"]["attempts"], summary["public-gap"]["unassessed"]), (0, 1))


if __name__ == "__main__":
    unittest.main()
