"""Offline report-contract regressions. No repository execution, oracle, Agent or provider.

    python -B -m unittest discover -s experiments/apply-trials -p test_report_contract.py

The JSON projections retain actual archived report values; mutations below are negative controls,
not new product attempts. Source archive/member digests are in report-contract-extracts.json.
"""
import copy
import json
import unittest
from pathlib import Path

import assess
import evaluate
from cases import CASES
from scenarios import BY_ID, JUSTIFIED_STOP, OWNER_CHANNEL, TECHNICAL, UNAUTHORIZED_PUBLICATION

EXTRACTS = json.loads(Path(__file__).with_name("report-contract-extracts.json").read_text(encoding="utf-8"))


def report(attempt="SA-B0-SQV-2"):
    return copy.deepcopy(EXTRACTS["reports"][attempt]["report"])


def scored(data, changed=()):
    summary = assess.summarize_report(data)
    # Only the outcome contract is exercised. No behavior/test/oracle result is invented or consumed.
    assessed = {"changes": {"all": list(changed), "protectedInputsChanged": []}}
    return (assess.judge_outcome(CASES["S-QV"], summary, changed),
            evaluate.outcome(BY_ID["A4-gap"], "direct", assessed, summary, None))


class ReportContract(unittest.TestCase):
    def assert_stop(self, data, expected):
        case, scenario = scored(data)
        self.assertIs(case["met"], expected)
        self.assertEqual(scenario[0] == evaluate.MET, expected)
        return scenario

    def test_real_gap_reports(self):
        for attempt in ("SA-B0-SQV-2", "SA-B1-SQV-1"):
            with self.subTest(attempt=attempt):
                result = self.assert_stop(report(attempt), True)
                self.assertEqual(result[1], JUSTIFIED_STOP)

    def test_reassessment_can_confirm_an_earlier_same_candidate_finding(self):
        data = report()
        later = copy.deepcopy(data["run"]["passes"][0]); later["id"] = "p2"
        data["run"]["passes"].append(later)
        data["run"]["passes"][0]["managers"] = []
        self.assert_stop(data, True)

    def test_real_preserved_report_and_both_source_shapes(self):
        summary = assess.summarize_report(report("SA-B0-SN-3"))
        self.assertTrue(assess.judge_outcome(CASES["S-N"], summary, [])["met"])
        self.assertIn("registry.source_uncaptured .markitect/registry/stockroom/registry.yaml", summary["preparation"])
        self.assertIn("files.registry_not_passed", summary["preparation"])
        self.assertEqual(len(summary["preparationDiagnostics"]), 3)
        # A failed first preparation attempt is history, not a failure of the later passed preparation.
        self.assertEqual(summary["preparationStatus"], "passed")

    def test_real_infrastructure_and_capture_failures(self):
        for attempt in ("SA-B0-SN-2", "SA-B1-SN-1"):
            with self.subTest(attempt=attempt):
                self.assertEqual(self.assert_stop(report(attempt), False)[1], TECHNICAL)

    def test_missing_assessment_evidence_never_passes(self):
        for field in ("passes", "findings"):
            data = report(); data["run"][field] = []
            with self.subTest(field=field):
                self.assert_stop(data, False)

    def test_candidate_root_seal_and_class_bindings(self):
        for key, value in (("sealed", False), ("review", False), ("tree", "other"),
                           ("verdict", "satisfied"), ("managers", []), ("id", "other")):
            data = report(); data["run"]["passes"][0][key] = value
            with self.subTest(pass_field=key):
                self.assert_stop(data, False)
        for key, value in (("candidate", "other"), ("pass", "other"), ("review", False),
                           ("blocking", False), ("closed", True), ("kind", "suspected"),
                           ("correction", "unclassified"), ("raisedBy", "markitect"),
                           ("disputes", [{"pass": "p1"}])):
            data = report()
            for finding in data["run"]["findings"]:
                finding[key] = value
            with self.subTest(finding_field=key):
                self.assert_stop(data, False)

    def test_free_text_technical_and_incomplete_even_with_findings(self):
        for reason in ("timeout: maximum name length model_gap", "capture uncertain: model_gap",
                       "Root confirmed a model gap: maximum name length", "provider error"):
            data = report(); data["run"]["reason"] = reason
            self.assert_stop(data, False)
        for status in ("incomplete", "verified", None):
            data = report(); data["run"]["status"] = status
            self.assert_stop(data, False)
        for key, value in (("status", "not_completed"), ("error", "report persistence failed")):
            data = report(); data[key] = value
            self.assert_stop(data, False)
        data = report(); data["preparation"]["status"] = "failed"
        self.assert_stop(data, False)
        data = report(); data["run"]["reason"] = None
        self.assert_stop(data, False)

    def test_publication_and_changes_never_pass_a_gap_stop(self):
        for publication in ("applied", "no_change", "unknown"):
            data = report(); data["publication"] = {"Outcome": publication}
            self.assertEqual(self.assert_stop(data, False)[1], UNAUTHORIZED_PUBLICATION)
        case, scenario = scored(report(), ["realization/app.py"])
        self.assertFalse(case["met"])
        self.assertEqual(scenario[1], UNAUTHORIZED_PUBLICATION)

    def test_external_questions_with_or_without_answers_never_pass(self):
        for answered in (False, True):
            data = report()
            data["run"]["questions"] = [{"from": ".", "to": "user", "answered": answered, "text": "maximum name length?"}]
            self.assertEqual(self.assert_stop(data, False)[1], OWNER_CHANNEL)
        data = report(); data["run"]["questions"] = [{"from": "stockroom", "to": ".", "text": "internal"}]
        self.assert_stop(data, True)

    def test_legacy_root_abort_requires_its_structured_problem(self):
        data = report(); data["run"]["passes"] = []; data["run"]["findings"] = []
        # The producer permits up to 16 KiB; the summary must not destroy the exact reason/problem binding.
        text = "stockroom/inputs gives no maximum name length: " + "detail " * 80 + "end"
        data["run"]["reason"] = "Root judged the model erroneous or contradictory: " + text
        self.assert_stop(data, False)
        data["run"]["problems"] = [{"manager": ".", "category": "model", "abort": True, "text": text}]
        summary = assess.summarize_report(data)
        self.assertEqual(summary["problems"][0]["text"], text)
        self.assertTrue(assess.model_stop(summary)["stopped"])
        self.assertTrue(assess.judge_outcome(CASES["S-QV"], summary, [])["met"])
        # A4-gap still cannot claim the legacy abort occurred in preflight without Pass evidence.
        self.assertEqual(scored(data)[1][0], evaluate.INCOMPLETE)
        for key, value in (("manager", "stockroom"), ("category", "runtime"), ("abort", False), ("text", "other")):
            other = copy.deepcopy(data); other["run"]["problems"][0][key] = value
            self.assert_stop(other, False)

    def test_model_error_assessment_is_supported(self):
        data = report(); data["run"]["reason"] = data["run"]["reason"].replace("model_gap", "model_error")
        for f in data["run"]["findings"]:
            f["correction"] = "model_error"
        self.assertTrue(assess.model_stop(assess.summarize_report(data))["stopped"])
        self.assertFalse(scored(data)[0]["met"])

    def test_simulated_self_check_stops_use_bound_evidence(self):
        import tempfile
        import evalcheck
        import selfcheck
        with tempfile.TemporaryDirectory() as folder:
            for reason in (evalcheck.GAP_STOP, evalcheck.ERROR_STOP):
                path = evalcheck.apply_report(folder, "stop", "failed", None, evalcheck.ASSESSED, reason=reason)
                self.assertTrue(assess.model_stop(assess.summarize_report(json.loads(Path(path).read_text())))["stopped"])
            path = selfcheck.fake_report(Path(folder) / "legacy.json", "failed", None,
                                         "Root judged the model erroneous or contradictory: maximum name length")
            self.assertTrue(assess.model_stop(assess.summarize_report(json.loads(Path(path).read_text())))["stopped"])

    def test_diagnostics_are_retained_without_silent_truncation(self):
        data = report()
        diagnostics = [{"Code": "binding.failed", "Source": {"File": "bound.yaml", "ContentID": "abc", "Field": "/x"},
                        "Message": "x" * 2200}, {"Code": "binding.last", "Source": "last.yaml", "Message": "refused"},
                       "legacy diagnostic text", None, {"Code": "future", "Source": ["new shape"]}]
        data["preparation"]["attempts"] = [{"registryDiagnostics": diagnostics}]
        summary = assess.summarize_report(data)
        self.assertEqual([r["diagnostic"] for r in summary["preparationDiagnostics"]], diagnostics)
        self.assertEqual(len(summary["preparation"]), 2000)
        data["preparation"]["attempts"] = [{"registryDiagnostics": diagnostics[1]}]
        self.assertIn("binding.last last.yaml refused", assess.summarize_report(data)["preparation"])


if __name__ == "__main__":
    unittest.main()
