"""Offline R5 stop and checkpoint predicates; fixtures are not native evidence."""
from __future__ import annotations

import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import test_government as translation_fixtures

DRIVER_PATH = Path(__file__).parents[1] / "run-native-integration-r5.py"
spec = importlib.util.spec_from_file_location("native_r5_driver", DRIVER_PATH)
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


class NativeR5CheckpointTests(unittest.TestCase):
    def setUp(self):
        self.report = translation_fixtures.GovernmentTranslationTests()._report()
        self.report["actors"] = [actor for actor in self.report["actors"]
                                 if actor["slotId"] != "child-writer"]
        self.report.update(stage="complete", candidateCommit="b" * 40, candidateTree="c" * 40,
                           checks=[{"Name": "inventory-overflow", "Tool": "go", "ExitCode": 0,
                                    "Milliseconds": 8, "TimeoutMilliseconds": 30000}])
        self.report["promotion"].update(expectedOld="d" * 40, newCommit="b" * 40,
                                         actualActive="b" * 40, intentPath="intent.json",
                                         completionPath="completion.json")
        self.queue = {"status": "complete", "nextStep": "none", "inFlightActors": 0,
                      "jobs": [{"id": driver.government_integration.TASK_ID,
                                "state": "accepted-scoped", "nextStep": "complete",
                                "resultStatus": "completed"}]}
        self.runtime = {"expectedBase": "d" * 40, "activeRef": "refs/markitect/government/active/test",
                        "checks": [{"name": "inventory-overflow", "run": ["go", "test", "./inventory"],
                                    "timeoutSeconds": 30}],
                        "executor": {"slotId": "writer", "command": "offline"},
                        "verifier": {"slotId": "reviewer", "command": "offline"},
                        "ressorts": [{"ressort": {"name": "finance"},
                                       "runner": {"slotId": "vote-finance", "command": "offline"}}]}
        self.process = {"returnCode": 0, "stopReason": None, "wallSeconds": 1.0,
                        "controllerElapsedSeconds": 2.0}

    def checkpoint(self):
        return driver.positive_checkpoint(self.process, self.queue, self.report, self.runtime)

    def test_complete_native_scope_and_fresh_check_checkpoint_pass(self):
        result = self.checkpoint()
        self.assertEqual(result["candidateCommit"], "b" * 40)
        self.assertEqual(result["scopeIdentities"], self.report["plan"]["integrationReviews"])

    def test_native_identity_must_match_all_four_planned_fields(self):
        self.report["plan"]["integrationReviews"][0]["apiVersion"] = "markitect.other/v1alpha1"
        with self.assertRaisesRegex(ValueError, "lacks a passing review receipt"):
            self.checkpoint()
        self.report["plan"]["integrationReviews"][0] = {
            "apiVersion": "markitect.government/v1alpha1", "kind": "Other",
            "namespace": "orders", "name": "root"}
        with self.assertRaisesRegex(ValueError, "lacks a passing review receipt"):
            self.checkpoint()

    def test_partial_or_lossy_native_scope_is_rejected(self):
        self.report["plan"]["integrationReviews"][0].pop("apiVersion")
        with self.assertRaisesRegex(ValueError, "four-field Government identity"):
            self.checkpoint()
        self.report["plan"]["integrationReviews"][0] = {
            "apiVersion": "markitect.government/v1alpha1", "kind": "Area",
            "namespace": "orders", "name": "root"}
        self.report["actors"][1]["scopes"] = ["orders/root"]
        with self.assertRaisesRegex(ValueError, "serialized four-field identity"):
            self.checkpoint()

    def test_missing_review_abstention_bad_check_and_overdeadline_stop(self):
        original = copy.deepcopy(self.report)
        self.report["actors"][1]["scopes"] = []
        with self.assertRaisesRegex(ValueError, "lacks a passing review receipt"):
            self.checkpoint()
        self.report = copy.deepcopy(original)
        self.report["votes"][0]["outcome"] = "abstain"
        with self.assertRaisesRegex(ValueError, "positive final votes"):
            self.checkpoint()
        self.report = copy.deepcopy(original)
        self.report["checks"][0]["Tool"] = "git"
        with self.assertRaisesRegex(ValueError, "names/tools"):
            self.checkpoint()
        self.report = copy.deepcopy(original)
        self.report["checks"][0]["Milliseconds"] = 30001
        with self.assertRaisesRegex(ValueError, "time receipt"):
            self.checkpoint()
        self.report = copy.deepcopy(original)
        self.process["wallSeconds"] = 38.1
        with self.assertRaisesRegex(ValueError, "Queue/process"):
            self.checkpoint()
        self.process["wallSeconds"] = 1.0
        self.process["controllerElapsedSeconds"] = 38.1
        with self.assertRaisesRegex(ValueError, "Queue/process"):
            self.checkpoint()

    def test_incomplete_queue_and_nonzero_check_cannot_admit_checkpoint(self):
        self.queue["status"] = "incomplete"
        with self.assertRaisesRegex(ValueError, "Queue/process"):
            self.checkpoint()
        self.queue["status"] = "complete"
        self.report["checks"][0]["ExitCode"] = 1
        with self.assertRaisesRegex(ValueError, "exit/time receipt"):
            self.checkpoint()

    def test_resume_requires_explicit_final_assent_and_bound_promotion(self):
        self.assertEqual(self.checkpoint()["candidateCommit"], "b" * 40)
        baseline = copy.deepcopy(self.report)
        self.report["votes"][0]["evidenceId"] = "sha256:" + "0" * 64
        with self.assertRaisesRegex(ValueError, "candidate/evidence/round"):
            self.checkpoint()
        self.report = copy.deepcopy(baseline)
        self.report["promotion"]["completionPath"] = ""
        with self.assertRaisesRegex(ValueError, "promotion"):
            self.checkpoint()

    def test_resume_controller_window_includes_reservation_and_readbacks(self):
        self.assertEqual(driver.r5_resume_work_window(0), 32)
        self.assertEqual(driver.r5_resume_work_window(5), 27)
        self.assertTrue(driver.r5_resume_within_controller_deadline(38))
        self.assertFalse(driver.r5_resume_within_controller_deadline(38.001))
        with self.assertRaisesRegex(ValueError, "leaves no work time"):
            driver.r5_resume_work_window(32)

    def test_incomplete_queue_terminally_skips_resume_and_case_latch_is_one_shot(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            external, evidence = root / "external", root / "evidence"
            external.mkdir()
            evidence.mkdir()
            coordinator = root / "coordination.json"
            coordinator.write_text("{}", encoding="utf-8")
            with (patch.object(driver, "EXTERNAL", external),
                  patch.object(driver, "EVIDENCE", evidence),
                  patch.object(driver, "COORDINATION", coordinator),
                  patch.object(driver, "require_freeze"),
                  patch.object(driver, "government_queue", return_value={"status": "incomplete"}) as queue,
                  patch.object(driver, "government_resume") as resume):
                outcome = driver.run_once()
                self.assertEqual(outcome["status"], "incomplete")
                self.assertTrue(outcome["noRetry"])
                self.assertNotIn("resumePassed", outcome)
                queue.assert_called_once()
                resume.assert_not_called()
                with self.assertRaises(FileExistsError):
                    driver.run_once()
                queue.assert_called_once()
                resume.assert_not_called()

    def test_freeze_requires_reviewed_pins_and_unchanged_input_bytes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            external, evidence = root / "external", root / "evidence"
            external.mkdir()
            evidence.mkdir()
            govt = external / "government"
            (govt / "released").mkdir(parents=True)
            (govt / "released/request.json").write_bytes(b"{}")
            for path in (external / "released-r5-grant.json",
                         external / "coordinator-r5-preflight-snapshot.json",
                         external / "history-native-starts.sqlite",
                         evidence / "historical-native-starts.sqlite",
                         root / "prepare-native-r5.py",
                         evidence / "independent-preflight-review.md",
                         evidence / "host-success-contract.md",
                         root / "source-grant.json",
                         root / "public/resource-proposal.json",
                         root / "runtime/mechanical_actor.py",
                         govt / "authority.json", govt / "grant.json", govt / "protocol.json"):
                path.parent.mkdir(parents=True, exist_ok=True)
                if not path.exists():
                    path.write_bytes(b"fixture")
            (external / "government/authority.json").write_text("{}", encoding="utf-8")
            freeze_path = evidence / "preflight-freeze.json"
            inputs = [Path(driver.__file__), root / "prepare-native-r5.py",
                      evidence / "independent-preflight-review.md", evidence / "host-success-contract.md",
                      external / "released-r5-grant.json", root / "source-grant.json",
                      external / "coordinator-r5-preflight-snapshot.json",
                      external / "history-native-starts.sqlite", evidence / "historical-native-starts.sqlite",
                      root / "public/resource-proposal.json", root / "runtime/mechanical_actor.py",
                      govt / "authority.json", govt / "grant.json", govt / "protocol.json",
                      govt / "released/request.json", Path("C:/Python313/python.exe")]
            freeze = {"grantKey": driver.R5_KEY,
                      "status": "independently-reviewed-mechanical-fixture",
                      "sourceCommit": "a" * 40,
                      "runtimeSourceSha256": driver.dispatch.runtime_pins(),
                      "inputs": [driver.binding(path) for path in inputs]}
            freeze_path.write_bytes(driver.dispatch.encoded(freeze) + b"\n")
            with (patch.object(driver, "EXTERNAL", external),
                  patch.object(driver, "EVIDENCE", evidence),
                  patch.object(driver, "ROOT", root),
                  patch.object(driver, "SOURCE_GRANT", root / "source-grant.json"),
                  patch.object(driver, "SUCCESSOR_GRANT", external / "released-r5-grant.json"),
                  patch.object(driver, "EXTERNAL_HISTORY", external / "history-native-starts.sqlite"),
                  patch.object(driver, "COORDINATOR_SNAPSHOT", external / "coordinator-r5-preflight-snapshot.json"),
                  patch.object(driver, "HISTORY", evidence / "historical-native-starts.sqlite"),
                  patch.object(driver, "validate"),
                  patch.object(driver, "authority", return_value=(None, {"releasedInputs": []}, b"", {}, root / "request.json")),
                  patch("subprocess.check_output", side_effect=["".encode(), freeze_path.read_bytes(),
                                                                  "".encode(), freeze_path.read_bytes()]),
                  patch("subprocess.check_call")):
                driver.require_freeze()
                (external / "government/released/request.json").write_bytes(b"changed")
                with self.assertRaisesRegex(ValueError, "frozen R5 input changed"):
                    driver.require_freeze()


if __name__ == "__main__":
    unittest.main()
