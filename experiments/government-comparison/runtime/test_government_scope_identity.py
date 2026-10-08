"""Offline report-identity regressions; no product, Actor, or provider processes."""
import copy
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).parent))
import government
import test_government as fixture_module
from test_government import write


def scope(identity):
    return json.dumps([identity[key] for key in
                       ("apiVersion", "kind", "namespace", "name")], separators=(",", ":"))


class GovernmentScopeIdentityTests(unittest.TestCase):
    def setUp(self):
        self.fixture = fixture_module.GovernmentTranslationTests()
        self.fixture.setUp()
        self.addCleanup(self.fixture.tearDown)
        self.report = self.fixture._report()
        self.configured = {"writer": ("execute", "executor"),
                           "child-writer": ("execute", "executor"),
                           "reviewer": ("review", "verifier"),
                           "vote-finance": ("vote", "verifier")}

    def validate(self, report=None):
        return government._validate_run_report(
            self.report if report is None else report, "government-run-abc", self.configured,
            require_acceptance=True, root_review_slots={"reviewer"})

    def review(self, report=None):
        return next(actor for actor in (self.report if report is None else report)["actors"]
                    if actor["phase"] == "review")

    def test_native_four_field_scope_covers_matching_planned_review(self):
        identity = {"apiVersion": "markitect.government/v1alpha1", "kind": "Area",
                    "namespace": "inventory", "name": "root"}
        native = '["markitect.government/v1alpha1","Area","inventory","root"]'
        self.report["plan"]["integrationReviews"] = [identity]
        self.review()["scopes"] = [native]
        self.assertIn("government-decision", self.validate())

    def test_json_escaping_and_formatting_preserve_exact_identity(self):
        identity = self.report["plan"]["integrationReviews"][0]
        identity["name"] = "caf\u00e9"
        self.review()["scopes"] = [json.dumps(list(identity.values()), ensure_ascii=True, indent=2)]
        self.assertIn("government-decision", self.validate())
        self.review()["scopes"] = [scope({**identity, "name": identity["name"] + " "})]
        with self.assertRaisesRegex(ValueError, "complete four-field"):
            self.validate()

    def test_matching_but_syntactically_invalid_identities_are_rejected(self):
        identity = self.report["plan"]["integrationReviews"][0]
        malformed = {"apiVersion": ["v", "a/b/c", ".a/v1", "a./v1", "a/_v1", "a/\u00e9", "a" * 254 + "/v1", "a/" + "v" * 64],
                     "kind": ["Area ", " Area", "A/B", "_Area"],
                     "namespace": ["bad/ns", "bad ns", "bad\nns"],
                     "name": ["a b", 'a/b,"c\\d', "\u00b2", "a\x00b"]}
        for key, values in malformed.items():
            for value in values:
                with self.subTest(field=key, value=value):
                    bad = {**identity, key: value}
                    self.report["plan"]["integrationReviews"] = [bad]
                    self.review()["scopes"] = [scope(bad)]
                    with self.assertRaisesRegex(ValueError, "complete four-field"):
                        self.validate()

    def test_empty_namespace_is_a_valid_complete_identity(self):
        identity = self.report["plan"]["integrationReviews"][0]
        identity["namespace"] = ""
        self.review()["scopes"] = [scope(identity)]
        self.report["cabinet"][0]["ressort"]["namespace"] = ""
        self.report["votes"][0]["ressort"]["namespace"] = ""
        self.assertIn("government-decision", self.validate())

    def test_wrong_api_kind_namespace_or_name_never_covers_review(self):
        identity = self.report["plan"]["integrationReviews"][0]
        for key in identity:
            with self.subTest(field=key):
                self.review()["scopes"] = [scope({**identity, key: identity[key] + "-different"})]
                with self.assertRaisesRegex(ValueError, "planned integration scope"):
                    self.validate()

    def test_partial_or_malformed_planned_identity_is_rejected(self):
        identity = self.report["plan"]["integrationReviews"][0]
        bad = [None, [], "orders/root", {**identity, "unexpected": "field"}]
        for key in identity:
            bad += [{k: v for k, v in identity.items() if k != key}, {**identity, key: None},
                    {**identity, key: 1}, {**identity, key: True}]
            if key != "namespace":
                bad += [{**identity, key: ""}, {**identity, key: " "}]
        for value in bad:
            with self.subTest(identity=value):
                self.report["plan"]["integrationReviews"] = [value]
                with self.assertRaisesRegex(ValueError, "complete four-field"):
                    self.validate()

    def test_malformed_native_scopes_never_fall_back_to_short_names(self):
        malformed = ["orders/root", "not-json", "null", "{}", '"orders/root"',
                     '[]', '["orders","root"]', '["v","Area","orders","root","extra"]',
                     '["v","Area",null,"root"]', '["v","Area",false,"root"]',
                     '["v","Area",1,"root"]', '["","Area","orders","root"]',
                     '["v","","orders","root"]', '["v","Area","orders",""]']
        for value in malformed:
            with self.subTest(scope=value):
                self.review()["scopes"] = [value]
                with self.assertRaisesRegex(ValueError, "four-field"):
                    self.validate()
        for value in (None, {}, "orders/root", [None], [1], [True]):
            with self.subTest(scopes=value):
                self.review()["scopes"] = value
                with self.assertRaisesRegex(ValueError, "reviewed scopes"):
                    self.validate()

    def test_every_planned_review_requires_coverage_by_passing_review(self):
        identity = self.report["plan"]["integrationReviews"][0]
        other = {**identity, "name": "other"}
        self.report["plan"]["integrationReviews"].append(other)
        with self.assertRaisesRegex(ValueError, "planned integration scope"):
            self.validate()
        self.review()["scopes"].append(scope(other))
        self.assertIn("government-decision", self.validate())
        self.review()["scopes"] = []
        with self.assertRaisesRegex(ValueError, "planned integration scope"):
            self.validate()

    def test_non_review_actor_scope_cannot_supply_missing_review_coverage(self):
        self.report["actors"][0]["scopes"] = self.review()["scopes"]
        self.review()["scopes"] = []
        with self.assertRaisesRegex(ValueError, "planned integration scope"):
            self.validate()

    def test_vote_must_match_all_four_selected_cabinet_identity_fields(self):
        identity = self.report["votes"][0]["ressort"]
        for key in identity:
            with self.subTest(field=key):
                report = copy.deepcopy(self.report)
                report["votes"][0]["ressort"][key] += "-different"
                with self.assertRaisesRegex(ValueError, "outside or duplicated"):
                    self.validate(report)

    def test_cabinet_and_vote_require_complete_identities(self):
        for collection in ("cabinet", "votes"):
            identity = self.report[collection][0]["ressort"]
            for key in identity:
                for value in (None, False, 1):
                    with self.subTest(collection=collection, field=key, value=value):
                        report = copy.deepcopy(self.report)
                        report[collection][0]["ressort"][key] = value
                        with self.assertRaisesRegex(ValueError, "complete four-field"):
                            self.validate(report)
                with self.subTest(collection=collection, missing=key):
                    report = copy.deepcopy(self.report)
                    del report[collection][0]["ressort"][key]
                    with self.assertRaisesRegex(ValueError, "complete four-field"):
                        self.validate(report)

    def test_existing_review_vote_receipt_and_decision_requirements_still_reject(self):
        mutations = [
            (lambda r: r["votes"].clear(), "one explicit vote"),
            (lambda r: r["votes"][0].update(outcome="objection"), "positive final votes"),
            (lambda r: r["votes"][0].update(evidenceId="different"), "candidate/evidence/round"),
            (lambda r: r["votes"][0].update(priorMandate={}), "authority differs"),
            (lambda r: r["votes"][0]["provenance"].update(slotId="other"), "selected native verifier"),
            (lambda r: r["decision"].update(voteIds=[]), "vote set"),
            (lambda r: r.update(decision=None), "complete native AcceptanceDecision"),
            (lambda r: self.review(r)["result"]["Receipt"].update(runId="other"), "role/receipt"),
            (lambda r: self.review(r)["result"]["Receipt"].update(inputDigest="other"), "response/receipt"),
            (lambda r: self.review(r)["result"]["Response"].update(outcome="failed"), "passing review"),
            (lambda r: self.review(r)["result"]["Response"].update(uncertainty=["unknown"]), "passing review"),
            (lambda r: self.review(r)["result"]["Response"].update(verifierObservations=[]), "passing review"),
            (lambda r: r["actors"][-1]["result"]["Response"].update(verifierObservations=[]), "passing native verifier"),
        ]
        for mutate, error in mutations:
            with self.subTest(error=error):
                report = copy.deepcopy(self.report)
                mutate(report)
                with self.assertRaisesRegex(ValueError, error):
                    self.validate(report)

    def test_queue_and_resume_use_the_same_full_identity_validation(self):
        stdout, value = self.fixture._queue()
        for operation in ("run_task", "resume"):
            with self.subTest(operation=operation):
                self.fixture.request["operation"] = operation
                if operation == "resume":
                    self.fixture.request["product"]["government"]["queueDirectory"] = value["queueDirectory"]
                result = government.translate_queue_result(
                    self.fixture.request, self.fixture._request_bytes(), stdout, 0)
                self.assertEqual(result["status"], "completed")
                self.assertIsNone(result["candidateCommit"])
                self.assertIsNone(result["usage"]["providerRequests"])
                path = Path(value["jobs"][0]["reportPath"])
                report = json.loads(path.read_bytes())
                report["plan"]["integrationReviews"][0]["kind"] = "DifferentArea"
                value["jobs"][0]["reportDigest"] = write(path, report)
                queue = Path(value["queueDirectory"])
                write(queue / "queue-report-00000001.json", value)
                write(stdout, value)
                with self.assertRaisesRegex(ValueError, "planned integration scope"):
                    government.translate_queue_result(
                        self.fixture.request, self.fixture._request_bytes(), stdout, 0)
                report["plan"]["integrationReviews"][0]["kind"] = "Area"
                value["jobs"][0]["reportDigest"] = write(path, report)
                write(queue / "queue-report-00000001.json", value)
                write(stdout, value)


if __name__ == "__main__":
    unittest.main(verbosity=2)
