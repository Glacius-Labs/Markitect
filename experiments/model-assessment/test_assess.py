"""Provider-free self-tests of assess.py and freeze.py. Run: python -B -m unittest discover -s experiments/model-assessment"""
import importlib.util
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

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
        sealed = reason == assess.ACCEPTED_REASON or reason.startswith(assess.PROBLEM_REASON)
    if verdict is None:
        verdict = "satisfied" if reason == assess.ACCEPTED_REASON else "needs_correction" if reason.startswith(assess.PROBLEM_REASON) else ""
    return {"status": "passed" if reason == assess.ACCEPTED_REASON else "failed" if status == "failed" else "not_completed",
            "run": {"status": status, "reason": reason, "passes": [{"id": "p1", "purpose": "assess", "review": True, "sealed": sealed, "verdict": verdict}],
                    "questions": list(questions), "problems": list(problems), "findings": list(findings), "turns": [{"id": "t1"}, {"id": "t2"}],
                    "usage": {"costUSD": 0.5, "inputTokens": 100, "outputTokens": 10}}}


def finding(observed, obligation="shop/inputs", cls="model_gap", blocking=True, closed=False, fid="F1", by="."):
    return {"id": fid, "owner": ".", "raisedBy": by, "obligation": obligation, "observed": observed, "blocking": blocking,
            "closed": closed, "correction": cls}


def confirmed(*findings):
    return report("failed", assess.PROBLEM_REASON + "F1 [shop/inputs, model_gap]", findings=findings)


ABORT = {"id": "X1", "manager": ".", "category": "model", "text": "shop/inputs folds case but shop/list keeps it.", "abort": True}


class Classification(unittest.TestCase):
    def test_conclusions_follow_the_sealed_pass_and_the_finding_class(self):
        cases = {
            "accepted": report("incomplete", assess.ACCEPTED_REASON),
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
        harness_error = report("incomplete", assess.ACCEPTED_REASON)
        harness_error["status"], harness_error["error"] = "not_completed", "evidence: report could not be written"
        cases = {
            "deadline": report("incomplete", "the deadline was reached"),
            "an Orchestrator finding only": confirmed(finding("file.unexplained: x", by="markitect")),
            "unsealed": report("failed", assess.PROBLEM_REASON + "x", findings=[finding("x")], sealed=False),
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

    def test_only_the_exact_limit_reasons_are_a_limit(self):
        for reason, kind in (("the Pass budget of 2 ended", "limit"), ("the agent turn budget of 40 ended", "limit"),
                             ("the deadline was reached", "limit"), ("the run was cancelled", "limit"),
                             ("the agent turn budget of 4 does not cover a basis and a result of each of the 3 managers in one Pass", "technical"),
                             ('manager "shop" stopped after 3 failed turns: context deadline exceeded', "technical")):
            with self.subTest(reason):
                self.assertEqual(assess.classify(report("incomplete", reason))["incomplete"], kind)

    def test_an_omitted_class_stays_visible(self):
        r = confirmed(finding("The maximum length of shop/inputs is never stated.", cls="unclassified"))
        c = assess.classify(r)
        self.assertEqual((c["conclusion"], c["classes"]), ("model_problem", ["unclassified"]))
        s = assess.score(r, c, GAP)
        self.assertTrue(s["blocked"] and s["issueRaised"])
        self.assertFalse(s["expectationMet"])

    def test_only_the_production_assessment_envelope_can_conclude(self):
        accepted = report("incomplete", assess.ACCEPTED_REASON)
        legacy = report("verified", sealed=True, verdict="satisfied")
        wrong_purpose = report("incomplete", assess.ACCEPTED_REASON)
        wrong_purpose["run"]["passes"][0]["purpose"] = "verify"
        failed_evidence = confirmed(finding("The maximum length of shop/inputs is missing."))
        failed_evidence["error"] = "evidence write failed"
        for r in (legacy, wrong_purpose, failed_evidence):
            with self.subTest(r):
                self.assertEqual(assess.classify(r)["conclusion"], assess.INCOMPLETE)
        self.assertEqual(assess.classify(accepted)["conclusion"], assess.ACCEPTED)


class Scoring(unittest.TestCase):
    def test_a_sourced_gap_finding_meets_the_gap_control(self):
        r = confirmed(finding("[[inputs]] refers to a maximum name length that no Definition states; 40 and 100 both conform."))
        s = assess.score(r, assess.classify(r), GAP)
        self.assertTrue(s["expectationMet"] and s["blocked"] and s["issueRaised"])
        self.assertFalse(s["missedIssue"])
        self.assertEqual(s["falseAlarmCandidates"], [])
        self.assertEqual(s["ownerEffort"]["confirmedFindings"], 1)

    def test_accepting_the_gap_misses_it(self):
        r = report("incomplete", assess.ACCEPTED_REASON)
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
        r = report("incomplete", assess.ACCEPTED_REASON, findings=[finding("The file format is free.", obligation="shop/storage", blocking=False)])
        s = assess.score(r, assess.classify(r), FREEDOM)
        self.assertTrue(s["expectationMet"])
        self.assertFalse(s["falseAbort"])
        self.assertEqual(s["falseAlarmCandidates"], [])
        self.assertEqual(s["ownerEffort"]["notes"], 1)

    def test_a_disproved_finding_is_disagreement_not_owner_effort(self):
        r = report("incomplete", assess.ACCEPTED_REASON, findings=[dict(finding("The maximum is unstated."), closed=True, closedAs="disproved", disputes=[{"by": "."}])])
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
        r = report("incomplete", "the deadline was reached", findings=[finding("The format is open.", obligation="shop/storage")])
        s = assess.score(r, assess.classify(r), GAP)
        self.assertIsNone(s["expectationMet"])
        self.assertFalse(s["missedIssue"] or s["blocked"])
        # An unfinished attempt confirms nothing: no owner effort and no false alarm.
        self.assertEqual(([x["kind"] for x in s["signals"]], s["falseAlarmCandidates"], s["ownerEffort"]["confirmedFindings"]), (["open"], [], 0))

    def test_failed_harness_process_overrides_a_complete_report(self):
        for envelope, control in ((report("incomplete", assess.ACCEPTED_REASON), FREEDOM),
                                  (confirmed(finding("The maximum length is missing.")), GAP)):
            for code in (0, 1, 124):
                with self.subTest(conclusion=assess.classify(envelope)["conclusion"], code=code), tempfile.TemporaryDirectory() as root:
                    root = Path(root)
                    (root / "stdout").write_text(json.dumps(envelope) + "\n" + ("PASS" if code == 0 else "FAIL") + "\n", encoding="utf-8")
                    (root / "timing").write_text(f"exit={code}\nwall_seconds=2\n", encoding="utf-8")
                    (root / "control").write_text(json.dumps(control), encoding="utf-8")
                    with patch("builtins.print"):
                        self.assertEqual(assess.main(["assess.py", "attempt", "--report", str(root / "stdout"),
                                                     "--control", str(root / "control"), "--timing", str(root / "timing"),
                                                     "--output", str(root / "score")]), 0)
                    result = json.loads((root / "score").read_text(encoding="utf-8"))
                    if code == 0:
                        self.assertEqual(result["classification"]["conclusion"], assess.classify(envelope)["conclusion"])
                    else:
                        self.assertEqual(result["classification"]["incomplete"], "technical")
                        self.assertEqual(result["classification"]["harnessExit"], str(code))
                        self.assertIsNone(result["expectationMet"])
                        self.assertFalse(result["blocked"] or result["falseAbort"])
                        self.assertEqual(result["ownerEffort"]["confirmedFindings"], 0)

    def test_the_class_of_the_finding_on_the_issue_decides(self):
        r = confirmed(finding("The maximum length of shop/inputs is never stated.", fid="F1"),
                      finding("shop/storage contradicts shop/system.", obligation="shop/storage", cls="model_error", fid="F2"))
        s = assess.score(r, assess.classify(r), GAP)
        self.assertEqual((s["classification"]["conclusion"], s["issueClasses"]), ("model_error", ["model_gap"]))
        self.assertTrue(s["expectationMet"])
        self.assertEqual(s["falseAlarmCandidates"], ["F2"])

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
        # F1 is raised in p1 and disproved in p2; only a finding on its Definition after p2 is raised again.
        run = {"findings": [dict(finding("The limit is unstated.", fid="F1"), closed=True, closedAs="disproved", closedPass="p2", **{"pass": "p1"}),
                            dict(finding("Another statement of it.", fid="F2"), **{"pass": "p2"}),
                            dict(finding("The limit is still unstated.", fid="F3"), **{"pass": "p3"}),
                            dict(finding("Much later.", fid="F4"), **{"pass": "p10"}),
                            dict(finding("Another point.", obligation="shop/list", fid="F5"), **{"pass": "p3"})]}
        self.assertEqual(assess.process([], run)["reraisedAfterDisproof"], ["F3", "F4"])

    def test_non_object_evidence_lines_are_skipped(self):
        events = [{"kind": "turn.start", "manager": ".", "turn": "r-p1-root-r1"}]
        with tempfile.TemporaryDirectory() as root:
            self.write(root, "r-p1-root-r1", [{"kind": "provider-error", "data": "boom"}, {"kind": "tool_call", "data": "not an object"}, "123",
                                              {"kind": "tool_call", "data": {"id": 1, "name": "send", "arguments": "x"}}])
            self.assertEqual(assess.user_question_attempts(root, events), [])
            self.assertEqual(assess.jsonl(Path(root) / "transcripts" / "r-p1-root-r1.jsonl")[0], {"kind": "provider-error", "data": "boom"})
            self.assertEqual(len(assess.jsonl(Path(root) / "transcripts" / "r-p1-root-r1.jsonl")), 3)

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
        envelope = report("incomplete", "credential unavailable")
        envelope.update(operation="model-assessment", outcome="incomplete")
        for indent in (None, 2):
            for end in ("PASS", "FAIL"):
                with self.subTest(indent=indent, end=end):
                    text = "=== RUN TestModelAssessmentHarness\n" + json.dumps(envelope, indent=indent) + "\n" + end + "\n"
                    decoded = assess.load_report(text)
                    self.assertEqual(decoded, envelope)
                    self.assertEqual(assess.classify(decoded)["incomplete"], "technical")
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
    def make_freeze(self, root):
        """Real tiny Git inputs; fake build bytes, with only the CLI check stubbed."""
        root = Path(root)
        source, packet, build = root / "source", root / "packet", root / "build"
        kit = source / "experiments"
        shutil.copytree(HERE, kit / "model-assessment", ignore=shutil.ignore_patterns("__pycache__"))
        (packet / ".markitect" / "model" / "shop").mkdir(parents=True)
        (packet / ".markitect" / "model" / "shop" / "inputs.yaml").write_text("id: inputs\nbody: Names are trimmed.\n", encoding="utf-8")
        heads = []
        for repo in (source, packet):
            subprocess.run(["git", "init", "-q", str(repo)], check=True, capture_output=True)
            freeze.git(repo, "add", ".")
            freeze.git(repo, "-c", "user.name=Freeze test", "-c", "user.email=freeze@example.invalid", "commit", "-qm", "fixture")
            heads.append(freeze.git(repo, "rev-parse", "HEAD"))
        build.mkdir()
        for name in ("assessment.test", "markitect", "runtime.env"):
            (build / name).write_bytes(b"frozen test bytes\n")
        (build / "build.txt").write_text("sha=" + heads[0] + "\n", encoding="utf-8")
        spec = {"build": str(build), "kit": str(kit), "controls": [{"id": "complete", "packet": {"repo": str(packet), "head": heads[1]},
                                                                    "expectation": {"expected": "accepted"}}]}
        spec_path, frozen = root / "spec.json", root / "frozen"
        spec_path.write_text(json.dumps(spec), encoding="utf-8")
        with patch.object(freeze, "check", return_value=(0, "passed", '{"status":"passed"}')):
            freeze.freeze(spec_path, frozen)
        return frozen, spec, spec_path

    def test_verification_rejects_changed_execution_metadata_and_added_kit_files(self):
        with tempfile.TemporaryDirectory() as root:
            frozen, spec, _ = self.make_freeze(root)
            self.assertEqual(freeze.verify(frozen, "complete"), [])
            for name in ("build.txt", "runtime.env"):
                path = Path(spec["build"]) / name
                original = path.read_bytes()
                path.write_bytes(original + b"changed\n")
                self.assertTrue(any(name in p for p in freeze.verify(frozen, "complete")), name)
                path.write_bytes(original)
            added = frozen / "kit" / "model-assessment" / "unexpected.py"
            added.write_text("raise RuntimeError('different imported code')\n", encoding="utf-8")
            self.assertIn("kit file model-assessment/unexpected.py was added after freeze", freeze.verify(frozen, "complete"))
            freeze.ledger_path(frozen).unlink()
            self.assertIn("the shared series ledger is missing", freeze.verify(frozen, "complete"))

    def test_successor_freezes_share_the_live_ledger_without_resetting_attempts(self):
        with tempfile.TemporaryDirectory() as root:
            first, spec, spec_path = self.make_freeze(root)
            ledger = freeze.ledger_path(first)
            ledger.write_text("complete C-01 status=done charged=600 cap=600\n", encoding="utf-8")
            spec["ledgerFrom"] = str(ledger)
            spec_path.write_text(json.dumps(spec), encoding="utf-8")
            second = Path(root) / "successor"
            with patch.object(freeze, "check", return_value=(0, "passed", '{"status":"passed"}')):
                freeze.freeze(spec_path, second)
            self.assertEqual(freeze.ledger_path(second), ledger)
            self.assertFalse((second / "ledger.txt").exists())
            with ledger.open("a", encoding="utf-8") as stream:
                stream.write("complete C-02 status=done charged=600 cap=600\n")
            self.assertEqual(len(freeze.ledger_path(second).read_text(encoding="utf-8").splitlines()), 2)
            self.assertEqual(freeze.ledger_path(second).parent / "ledger.lock", ledger.parent / "ledger.lock")

    def test_a_successor_refuses_unresolved_cleanup(self):
        with tempfile.TemporaryDirectory() as root:
            first, spec, spec_path = self.make_freeze(root)
            ledger = freeze.ledger_path(first)
            ledger.write_text("complete C-01 status=cleanup_failed charged=600 cap=600\n", encoding="utf-8")
            spec["ledgerFrom"] = str(ledger)
            spec_path.write_text(json.dumps(spec), encoding="utf-8")
            with patch.object(freeze, "check", return_value=(0, "passed", '{"status":"passed"}')):
                with self.assertRaisesRegex(SystemExit, "unfinished attempt"):
                    freeze.freeze(spec_path, Path(root) / "successor")
            self.assertIn("status=cleanup_failed", ledger.read_text(encoding="utf-8"))

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
        for cid in ("../outside", "/absolute", "C:\\outside", "", ".", "a\nb", None, "x" * 65):
            with self.subTest(cid), self.assertRaises(SystemExit):
                freeze.validate(cid, {"expected": "accepted"})
        self.assertEqual(freeze.verify(Path("unused-freeze"), "../outside"), ["invalid control id"])
        with self.assertRaises(SystemExit):
            freeze.validate("x", {"expected": "needs_clarification"})
        with self.assertRaises(SystemExit):
            freeze.validate("x", {"expected": "model_gap", "issue": {"terms": ["("]}})
        freeze.validate("x", {"expected": "model_gap", "issue": {"terms": ["max"]}})


class Cleanup(unittest.TestCase):
    def test_missing_observation_live_containers_and_failed_cleanup_cannot_claim_cessation(self):
        bash = shutil.which("bash")
        if os.name == "nt":
            git_bash = Path(os.environ.get("ProgramFiles", "C:/Program Files")) / "Git/bin/bash.exe"
            bash = str(git_bash) if git_bash.is_file() else None
        if not bash:
            self.skipTest("Bash is needed for the attempt cleanup controls")
        source = (HERE / "run" / "attempt.sh").read_text(encoding="utf-8")
        cleanup = "cleanup() {" + source.split("cleanup() {", 1)[1].split("\nfinish() {", 1)[0]
        for name, docker, helper, expected in (
                ("unavailable", "return 1", 0, "alive_after=unknown cleanup_confirmed=0"),
                ("alive", "echo container-id", 0, "alive_after=1 cleanup_confirmed=0"),
                ("helper-failed", "return 0", 1, "alive_after=0 cleanup_confirmed=0"),
                ("observed-stopped", "return 0", 0, "alive_after=0 cleanup_confirmed=1")):
            with self.subTest(name), tempfile.TemporaryDirectory() as root:
                script = ("set -u\nATTEMPT=attempt; CLEANED=0; CLEANUP_OK=0; START=$(date +%s); MARKITECT_STORE=unit-test\n"
                          "mkdir -p attempt/host attempt/frozen\ntouch attempt/host/host.yaml\n"
                          f"printf 'exit {helper}\\n' > attempt/frozen/ra_cleanup.sh\n"
                          "docker() { " + docker + "; }\n" + cleanup + "\ncleanup\ncat attempt/timing.txt\n")
                result = subprocess.run([bash, "-c", script], cwd=root, capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn(expected, result.stdout)


class Series(unittest.TestCase):
    def test_series_counts_conclusions_agreement_and_false_aborts_per_control(self):
        met = assess.score(report("incomplete", assess.ACCEPTED_REASON), assess.classify(report("incomplete", assess.ACCEPTED_REASON)), FREEDOM)
        alarm_report = confirmed(finding("format", obligation="shop/storage"))
        alarm = assess.score(alarm_report, assess.classify(alarm_report), FREEDOM)
        summary = assess.series([met, met, alarm])["permitted-freedom"]
        self.assertEqual((summary["attempts"], summary["met"], summary["falseAborts"]), (3, 2, 1))
        self.assertEqual(summary["conclusions"], {"accepted": 2, "model_gap": 1})
        self.assertFalse(summary["agreement"])

    def test_the_ledger_counts_attempts_that_were_never_assessed(self):
        met = assess.score(report("incomplete", assess.ACCEPTED_REASON), assess.classify(report("incomplete", assess.ACCEPTED_REASON)), FREEDOM)
        ledger = ("permitted-freedom C-01 status=done conclusion=accepted charged=200\n"
                  "permitted-freedom C-05 status=running charged=600\n"
                  "public-gap C-02 status=done conclusion=bind-failed charged=12\n")
        summary = assess.series([met], ledger)
        self.assertEqual((summary["permitted-freedom"]["ledgerAttempts"], summary["permitted-freedom"]["unassessed"]), (2, 1))
        self.assertEqual((summary["public-gap"]["attempts"], summary["public-gap"]["unassessed"]), (0, 1))


if __name__ == "__main__":
    unittest.main()
