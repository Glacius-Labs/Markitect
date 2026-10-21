"""Provider-free self-tests of assess.py and freeze.py. Run: python -B -m unittest discover -s experiments/model-assessment"""
import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

import assess

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("freeze", HERE / "run" / "freeze.py")
freeze = importlib.util.module_from_spec(spec)
spec.loader.exec_module(freeze)

DEFINITIONS = ["shop/inputs", "shop/system", "shop/list", "shop/storage"]
GAP = {"id": "public-gap", "expected": "model_gap", "definitions": DEFINITIONS,
       "issue": {"definitions": ["shop/inputs"], "terms": [r"maximum.{0,20}length"]}}
FREEDOM = {"id": "permitted-freedom", "expected": "accepted", "definitions": DEFINITIONS,
           "freedom": {"definitions": ["shop/storage"], "terms": [r"file format"]}}
CONTRADICTION = {"id": "contradiction", "expected": "model_error", "definitions": DEFINITIONS,
                 "issue": {"definitions": ["shop/inputs", "shop/list"], "terms": [r"case"]}}


def report(status, reason="", problems=(), findings=(), questions=(), sealed=None, verdict=None):
    if sealed is None:
        sealed = status == "verified" or reason == assess.PROBLEM_REASON
    if verdict is None:
        verdict = {"verified": "satisfied"}.get(status, "needs_correction" if reason == assess.PROBLEM_REASON else "")
    return {"status": "passed" if status == "verified" else "not_completed",
            "run": {"status": status, "reason": reason, "passes": [{"id": "p1", "purpose": "assess", "review": True, "sealed": sealed, "verdict": verdict}],
                    "questions": list(questions), "problems": list(problems), "findings": list(findings), "turns": [{"id": "t1"}, {"id": "t2"}],
                    "usage": {"costUSD": 0.5, "inputTokens": 100, "outputTokens": 10}}}


def finding(observed, obligation="shop/inputs", cls="model_gap", blocking=True, closed=False, fid="F1", by="."):
    return {"id": fid, "owner": ".", "raisedBy": by, "obligation": obligation, "observed": observed, "blocking": blocking,
            "closed": closed, "correction": cls}


def confirmed(*findings):
    return report("incomplete", assess.PROBLEM_REASON, findings=findings)


ABORT = {"id": "X1", "manager": ".", "category": "model", "text": "shop/inputs folds case but shop/list keeps it.", "abort": True}


class Classification(unittest.TestCase):
    def test_conclusions_follow_the_sealed_pass_and_the_finding_class(self):
        cases = {
            "accepted": report("verified"),
            "model_gap": confirmed(finding("The maximum length of `shop/inputs` is never stated.")),
            "model_error": confirmed(finding("open", fid="F1"), finding("shop/inputs contradicts shop/list", cls="model_error", fid="F2")),
        }
        for want, r in cases.items():
            with self.subTest(want):
                self.assertEqual(assess.classify(r)["conclusion"], want)
                self.assertEqual(assess.classify(r)["route"], "result")

    def test_roots_model_decision_is_a_second_route(self):
        error = report("failed", assess.MODEL_ERROR_REASON + "x", problems=[ABORT])
        gap = report("failed", assess.MODEL_ERROR_REASON + "x", problems=[dict(ABORT, text="model_gap: the maximum is missing")])
        self.assertEqual((assess.classify(error)["conclusion"], assess.classify(error)["route"]), ("model_error", "abort"))
        self.assertEqual(assess.classify(gap)["conclusion"], "model_gap")

    def test_technical_ends_are_incomplete_not_a_model_judgment(self):
        harness_error = report("verified")
        harness_error["status"], harness_error["error"] = "not_completed", "evidence: report could not be written"
        cases = {
            "deadline": report("incomplete", "the deadline was reached"),
            "an Orchestrator finding only": confirmed(finding("file.unexplained: x", by="markitect")),
            "unsealed": report("incomplete", assess.PROBLEM_REASON, findings=[finding("x")], sealed=False),
            "other failure": report("failed", "a definite error of the run's inputs"),
            "abort by a child": report("failed", assess.MODEL_ERROR_REASON + "x", problems=[dict(ABORT, manager="shop")]),
            "harness error": harness_error,
            "preparation": {"status": "failed"},
        }
        kinds = {"deadline": "limit", "an Orchestrator finding only": "unresolved", "unsealed": "technical", "other failure": "technical",
                 "abort by a child": "technical", "harness error": "technical", "preparation": "preparation"}
        for name, r in cases.items():
            with self.subTest(name):
                self.assertEqual(assess.classify(r)["conclusion"], assess.INCOMPLETE)
                self.assertEqual(assess.classify(r)["incomplete"], kinds[name])

    def test_a_finding_root_disputed_in_the_sealed_pass_is_dissent_not_confirmation(self):
        disputed = dict(finding("shop/inputs contradicts shop/list", cls="model_error", fid="F1", by="shop"), disputes=[{"by": ".", "pass": "p1"}])
        r = confirmed(disputed, finding("The maximum length of shop/inputs is never stated.", fid="F2"))
        c = assess.classify(r)
        self.assertEqual((c["conclusion"], c["confirmedFindings"]), ("model_gap", 1))
        s = assess.score(r, c, CONTRADICTION)
        self.assertEqual([x["kind"] for x in s["signals"]], ["disputed", "finding"])
        self.assertFalse(s["expectationMet"])
        self.assertEqual(s["ownerEffort"]["disputedOpen"], 1)
        only_disputed = confirmed(disputed)
        self.assertEqual(assess.classify(only_disputed)["incomplete"], "unresolved")

    def test_an_omitted_class_stays_visible(self):
        r = confirmed(finding("The maximum length of shop/inputs is never stated.", cls="unclassified"))
        c = assess.classify(r)
        self.assertEqual((c["conclusion"], c["classes"]), ("model_problem", ["unclassified"]))
        s = assess.score(r, c, GAP)
        self.assertTrue(s["blocked"] and s["issueRaised"])
        self.assertFalse(s["expectationMet"])


class Scoring(unittest.TestCase):
    def test_a_sourced_gap_finding_meets_the_gap_control(self):
        r = confirmed(finding("[[inputs]] refers to a maximum name length that no Definition states; 40 and 100 both conform."))
        s = assess.score(r, assess.classify(r), GAP)
        self.assertTrue(s["expectationMet"] and s["blocked"] and s["issueRaised"])
        self.assertFalse(s["missedIssue"])
        self.assertEqual(s["falseAlarmCandidates"], [])
        self.assertEqual(s["ownerEffort"]["confirmedFindings"], 1)

    def test_accepting_the_gap_misses_it(self):
        r = report("verified")
        s = assess.score(r, assess.classify(r), GAP)
        self.assertFalse(s["expectationMet"])
        self.assertTrue(s["missedIssue"])

    def test_a_block_on_another_issue_misses_the_gap_and_is_a_false_alarm(self):
        r = confirmed(finding("The file format is open.", obligation="shop/storage"))
        s = assess.score(r, assess.classify(r), GAP)
        self.assertFalse(s["expectationMet"])
        self.assertTrue(s["blocked"] and s["missedIssue"])
        self.assertEqual(s["falseAlarmCandidates"], ["F1"])

    def test_a_gap_classed_as_an_error_blocks_but_misses_the_class(self):
        r = confirmed(finding("The maximum length of shop/inputs is never stated.", cls="model_error"))
        s = assess.score(r, assess.classify(r), GAP)
        self.assertTrue(s["blocked"] and s["issueRaised"])
        self.assertFalse(s["missedIssue"])
        self.assertFalse(s["expectationMet"])

    def test_blocking_on_permitted_freedom_is_a_false_abort(self):
        r = confirmed(finding("Which file format should [[storage]] use?", obligation="shop/storage"))
        s = assess.score(r, assess.classify(r), FREEDOM)
        self.assertTrue(s["falseAbort"])
        self.assertFalse(s["expectationMet"])
        self.assertTrue(all(x["citesFreedom"] for x in s["signals"]))

    def test_a_note_is_no_false_alarm_and_an_accepted_freedom_meets_its_control(self):
        r = report("verified", findings=[finding("The file format is free.", obligation="shop/storage", blocking=False)])
        s = assess.score(r, assess.classify(r), FREEDOM)
        self.assertTrue(s["expectationMet"])
        self.assertFalse(s["falseAbort"])
        self.assertEqual(s["falseAlarmCandidates"], [])
        self.assertEqual(s["ownerEffort"]["notes"], 1)

    def test_a_disproved_finding_is_disagreement_not_owner_effort(self):
        r = report("verified", findings=[dict(finding("The maximum is unstated."), closed=True, closedAs="disproved", disputes=[{"by": "."}])])
        s = assess.score(r, assess.classify(r), FREEDOM)
        self.assertTrue(s["expectationMet"])
        self.assertEqual(s["ownerEffort"]["confirmedFindings"], 0)
        self.assertEqual((s["process"]["disputes"], s["process"]["disproved"]), (1, 1))

    def test_a_sourced_contradiction_meets_the_contradiction_control(self):
        for r in (confirmed(finding("shop/inputs folds case, shop/list keeps it.", cls="model_error")),
                  report("failed", assess.MODEL_ERROR_REASON + "contradiction", problems=[ABORT])):
            s = assess.score(r, assess.classify(r), CONTRADICTION)
            self.assertTrue(s["expectationMet"])
            self.assertEqual(s["uncitedBlocking"], [])

    def test_an_incomplete_attempt_is_not_scored_as_a_judgment(self):
        r = report("incomplete", "the deadline was reached")
        s = assess.score(r, assess.classify(r), GAP)
        self.assertIsNone(s["expectationMet"])
        self.assertFalse(s["missedIssue"] or s["blocked"])

    def test_terms_decide_the_match_and_tags_name_known_observations(self):
        control = dict(GAP, tags={"M2": {"terms": [r"unicode"]}, "invented: error wording": {"terms": [r"error (text|wording)"]}})
        r = confirmed(finding("shop/inputs: Unicode or ASCII trimming is open.", fid="F1"),
                      finding("shop/inputs: the exact error wording must be fixed.", fid="F2"))
        s = assess.score(r, assess.classify(r), control)
        self.assertEqual([x["tags"] for x in s["signals"]], [["M2"], ["invented: error wording"]])
        self.assertFalse(s["issueRaised"])
        self.assertEqual(s["falseAlarmCandidates"], ["F1", "F2"])

    def test_citations_need_the_identity_or_a_marked_local_id(self):
        self.assertEqual(assess.cited("see shop/inputs and [[list]]", DEFINITIONS), ["shop/inputs", "shop/list"])
        self.assertEqual(assess.cited("the list of inputs is long", DEFINITIONS), [])
        self.assertEqual(assess.cited("shop/inputs-extra", DEFINITIONS), [])


class Evidence(unittest.TestCase):
    def write(self, root, turn, lines):
        path = Path(root) / "transcripts" / f"{turn}.jsonl"
        path.parent.mkdir(parents=True, exist_ok=True)
        # A string is written raw, such as a line a killed writer truncated.
        path.write_text("".join((line if isinstance(line, str) else json.dumps(line)) + "\n" for line in lines), encoding="utf-8")

    def test_delivery_counts_full_reference_only_and_missing_definitions(self):
        control = {"definitionTexts": {"shop/inputs": {"package": "shop", "body": "Names are trimmed.\nAt most the maximum."},
                                       "shop/list": {"package": "shop", "body": "Lists every item."},
                                       "shop/storage": {"package": "shop", "body": "One file."},
                                       "other/x": {"package": "other", "body": "Not the reviewer's."}}}
        events = [{"kind": "turn.start", "pass": "p1", "manager": "shop", "turn": "r-p1-shop-r1", "data": {"role": "reviewer"}},
                  {"kind": "turn.start", "pass": "p1", "manager": "shop", "turn": "r-p1-shop-r2", "data": {"role": "reviewer"}}]
        prompt = "shop/inputs (Software.Rule): Inputs\nNames are   trimmed. At most the maximum.\nshop/list (Software.Rule): List\n"
        with tempfile.TemporaryDirectory() as root:
            self.write(root, "r-p1-shop-r1", [{"kind": "sent", "data": {"type": "user", "message": {"content": [{"type": "text", "text": prompt}]}}}])
            d = assess.delivery(root, events, control)
        self.assertEqual((d["reviewers"], d["expected"], d["full"], d["referenceOnly"]), (1, 3, 1, 1))
        self.assertEqual(d["missing"], ["p1 shop shop/list: body not sent", "p1 shop shop/storage: not sent"])

    def test_attempted_questions_to_the_user_are_counted(self):
        events = [{"kind": "turn.start", "manager": ".", "turn": "r-p1-root-r1"}, {"kind": "turn.start", "manager": "shop", "turn": "r-p1-shop-r1"}]
        with tempfile.TemporaryDirectory() as root:
            self.write(root, "r-p1-root-r1", [{"kind": "tool_call", "data": {"id": 1, "name": "send", "arguments": {"to": "user", "kind": "question", "text": "Which?"}}},
                                              {"kind": "tool_call", "data": {"id": 2, "name": "send", "arguments": {"to": "shop", "kind": "question", "text": "Settled?"}}},
                                              {"kind": "tool_call", "data": {"id": 3, "name": "escalate", "arguments": {"category": "model", "text": "Decide."}}}])
            # A child's escalate goes to its parent, inside the tree.
            self.write(root, "r-p1-shop-r1", [{"kind": "tool_call", "data": {"id": 1, "name": "escalate", "arguments": {"category": "model", "text": "Settled?"}}},
                                              '{"kind": "tool_call", "data": {"id": 2, "na'])
            self.assertEqual(assess.user_question_attempts(root, events), [{"turn": "r-p1-root-r1", "tool": "send", "text": "Which?"},
                                                                            {"turn": "r-p1-root-r1", "tool": "escalate", "text": "Decide."}])

    def test_truncated_evidence_is_skipped_or_read_as_no_run(self):
        with tempfile.TemporaryDirectory() as root:
            path = Path(root) / "events.jsonl"
            path.write_text('{"kind": "run.start"}\n{"kind": "turn.st', encoding="utf-8")
            self.assertEqual(assess.jsonl(path), [{"kind": "run.start"}])
        self.assertEqual(assess.load_report('{\n "run": {"status": "ver\n}\nFAIL\n'), {})

    def test_a_finding_raised_again_after_its_disproof_is_counted(self):
        run = {"findings": [dict(finding("The limit is unstated.", fid="F1"), closed=True, closedAs="disproved", **{"pass": "p1"}),
                            dict(finding("The limit is still unstated.", fid="F2"), **{"pass": "p2"}),
                            dict(finding("Another point.", obligation="shop/list", fid="F3"), **{"pass": "p2"})]}
        self.assertEqual(assess.process([], run)["reraisedAfterDisproof"], ["F2"])

    def test_returns_coverage_and_internal_questions_come_from_the_evidence(self):
        events = [
            {"kind": "proposal.submitted", "data": {"proposal": "P2", "kind": "result", "envelope": {"obligations": [
                {"definition": "shop/inputs", "status": "done", "evidence": [{"source": ".markitect/model/shop/inputs.yaml"}]},
                {"definition": "shop/list", "status": "done", "evidence": [{"source": "realization/app.py"}]}]}}},
            {"kind": "result.returned", "manager": "shop", "data": {"proposal": "P2", "by": ".", "reason": "unsupported"}},
            {"kind": "proposal.submitted", "data": {"proposal": "P3", "kind": "result", "envelope": {"obligations": [
                {"definition": "shop/inputs", "status": "done", "evidence": [{"source": ".markitect/model/shop/inputs.yaml"}]}]}}},
            {"kind": "result.accepted", "data": {"proposal": "P3"}},
        ]
        run = {"questions": [{"from": "shop", "to": "."}], "findings": []}
        p = assess.process(events, run)
        self.assertEqual(p["returned"], [{"manager": "shop", "by": ".", "reason": "unsupported"}])
        self.assertEqual(p["internalQuestions"], 1)
        self.assertEqual(p["acceptedResultCoverage"], {"obligations": 1, "withModelSource": 1, "open": 0})

    def test_the_report_is_read_between_the_harness_lines(self):
        text = '{\n "status": "failed",\n "run": {"status": "verified"}\n}\nPASS\n'
        self.assertEqual(assess.load_report(text)["run"]["status"], "verified")
        self.assertEqual(assess.load_report("model assessment exit 1\nFAIL\n"), {})
        self.assertEqual(assess.load_report(""), {})


class Declaration(unittest.TestCase):
    def test_the_frozen_pass_declaration_is_a_valid_assessment(self):
        declaration = json.loads((HERE / "assessment-pass.json").read_text(encoding="utf-8"))
        self.assertEqual(assess.preflight(declaration), [])

    def test_the_mandate_names_no_control_specific_issue_and_no_owner_channel(self):
        # The treatment is written from the assessment questions, not from the controls' answers.
        text = (HERE / "assessment-pass.json").read_text(encoding="utf-8").lower()
        for word in ("name length", "sku", "stockroom", "maximum name", "case-insensitive", "uppercase", "partial", "shortfall"):
            self.assertNotIn(word, text)
        self.assertIn("never a question to the user", text)

    def test_invalid_declarations_are_refused(self):
        valid = {"purpose": "assess", "mandate": "m", "guidance": "g", "corrections": ["model_error", "model_gap", "unclassified"]}
        self.assertEqual(assess.preflight(valid), [])
        for change in ({"purpose": "verify"}, {"mandate": " "}, {"corrections": ["model_error", "model_gap"]}, {"corrections": ["owner-decision"]}):
            with self.subTest(change):
                self.assertTrue(assess.preflight(dict(valid, **change)))


class Freeze(unittest.TestCase):
    def test_bodies_of_block_folded_and_plain_definitions(self):
        self.assertEqual(freeze.body_of("id: a\nkind: K\nbody: |-\n  One line\n  two.\nspec: {}\n"), "One line\ntwo.")
        self.assertEqual(freeze.body_of("id: a\nbody: >\n  Folded\n  text.\n"), "Folded\ntext.")
        self.assertEqual(freeze.body_of("id: a\nbody: Plain text.\n"), "Plain text.")
        self.assertEqual(freeze.body_of('id: a\nbody: "Say \\"no\\" twice."\n'), 'Say "no" twice.')
        self.assertEqual(freeze.body_of("id: a\nbody: 'It''s free.'\n"), "It's free.")
        self.assertEqual(freeze.body_of("id: a\n"), "")
        with self.assertRaises(SystemExit):
            freeze.body_of('id: a\nbody: "unterminated\n')

    def test_the_build_records_its_images(self):
        text = ("sha=abc\nimage markitect-runtime:ra-abc sha256:" + "1" * 64 + "\n"
                "image markitect-runtime-go:ra-abc sha256:" + "2" * 64 + "\npython_image=sha256:" + "3" * 64 + "\n")
        self.assertEqual(freeze.images(text), [("markitect-runtime:ra-abc", "sha256:" + "1" * 64), ("markitect-runtime-go:ra-abc", "sha256:" + "2" * 64),
                                               ("sha256:" + "3" * 64, "sha256:" + "3" * 64)])

    def test_definition_texts_and_identities_of_a_packet(self):
        with tempfile.TemporaryDirectory() as root:
            packet = Path(root)
            (packet / ".markitect" / "model" / "shop").mkdir(parents=True)
            (packet / ".markitect" / "model" / "shop" / "inputs.yaml").write_text("id: inputs\nbody: |-\n  Names are trimmed.\n", encoding="utf-8")
            (packet / ".markitect" / "dependencies" / "software" / "model" / "software").mkdir(parents=True)
            (packet / ".markitect" / "dependencies" / "software" / "model" / "software" / "rule.yaml").write_text("id: rule\n", encoding="utf-8")
            self.assertEqual(freeze.definitions(packet), ["shop/inputs", "software/rule"])
            self.assertEqual(freeze.definition_texts(packet), {"shop/inputs": {"package": "shop", "body": "Names are trimmed."}})

    def test_invalid_expectations_are_refused(self):
        with self.assertRaises(SystemExit):
            freeze.validate("x", {"expected": "needs_clarification"})
        with self.assertRaises(SystemExit):
            freeze.validate("x", {"expected": "model_gap", "issue": {"terms": ["("]}})
        freeze.validate("x", {"expected": "model_gap", "issue": {"terms": ["max"]}})


class Series(unittest.TestCase):
    def test_series_counts_conclusions_agreement_and_false_aborts_per_control(self):
        met = assess.score(report("verified"), assess.classify(report("verified")), FREEDOM)
        alarm_report = confirmed(finding("format", obligation="shop/storage"))
        alarm = assess.score(alarm_report, assess.classify(alarm_report), FREEDOM)
        summary = assess.series([met, met, alarm])["permitted-freedom"]
        self.assertEqual((summary["attempts"], summary["met"], summary["falseAborts"]), (3, 2, 1))
        self.assertEqual(summary["conclusions"], {"accepted": 2, "model_gap": 1})
        self.assertFalse(summary["agreement"])

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
