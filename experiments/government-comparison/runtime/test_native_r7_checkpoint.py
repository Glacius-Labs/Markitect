"""Pure offline R7 admission and terminal predicates; no process or ledger access."""
from __future__ import annotations

import importlib.util
from pathlib import Path
import unittest

PACKAGE = Path(__file__).parents[1]
spec = importlib.util.spec_from_file_location(
    "r7_checkpoint_driver", PACKAGE / "run-native-integration-r5.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


class NativeR7CheckpointTests(unittest.TestCase):
    def setUp(self):
        profile = driver.native_profiles.R7
        self.request_sha = "a" * 64
        self.process = {"path": "C:/evidence/process/process.json", "sha256": "b" * 64}
        self.queue_receipt = {
            "status": "completed", "dispatchId": profile.dispatch_id,
            "requestSha256": self.request_sha,
            "elapsedSecondsAfterTerminalReceiptAvailable": 37.8,
            "terminalResult": {"path": "C:/results/r7-queue.json", "sha256": "c" * 64},
            "terminalControllerReceiptSha256": "d" * 64,
            "nativeProcess": self.process,
        }

    def test_closed_r7_profile_and_thirteen_start_history_are_distinct(self):
        r7 = driver.native_profiles.R7
        self.assertEqual(r7.name, "r7")
        self.assertEqual(r7.key, "government-check-receipt-native-20261008-r7")
        self.assertEqual(r7.dispatch_id, "government-native-check-receipt-r7")
        self.assertEqual(r7.marker, "nativeFixtureR7Grant")
        self.assertEqual(r7.metadata_field, "fixtureR7SourceGrant")
        self.assertEqual(r7.task_id, "positive-overflow-release-r7")
        self.assertEqual(r7.history_starts, 13)
        self.assertEqual(r7.history_sha,
                         "c0a91051a16f5d2d3a7754789c2ad02b67b6b2f57f32ac4bd16562700b8fcf1a")
        self.assertEqual(driver.native_profiles.R6.history_starts, 12)

    def test_success_contract_requirements_keep_r6_addendum_out_of_r7(self):
        self.assertEqual(driver.r6_required_success_contracts("r7"),
                         ("host-success-contract.md",))
        self.assertEqual(driver.r6_required_success_contracts("r6"),
                         ("host-success-contract.md", "independent-preflight-a1-addendum.md"))

    def test_r7_queue_requires_bridge_and_driver_deadlines(self):
        self.assertTrue(driver.r6_queue_completion_is_positive(
            self.queue_receipt, self.request_sha, self.process, 37.99, "r7"))
        self.assertFalse(driver.r6_queue_completion_is_positive(
            self.queue_receipt, self.request_sha, self.process, 38.01, "r7"))
        late_bridge = dict(self.queue_receipt,
                           elapsedSecondsAfterTerminalReceiptAvailable=38.01)
        self.assertFalse(driver.r6_queue_completion_is_positive(
            late_bridge, self.request_sha, self.process, 37.0, "r7"))

    def test_r7_resume_requires_receipt_readback_inside_38_seconds(self):
        process = {"returnCode": 0, "stopReason": None, "wallSeconds": 31.9}
        completion = {"profile": "r7", "status": "completed",
                      "elapsedSecondsAfterTerminalReceiptAvailable": 37.7}
        self.assertTrue(driver.r6_resume_completion_is_positive(
            completion, process, "completed", 37.99, "r7"))
        self.assertFalse(driver.r6_resume_completion_is_positive(
            completion, process, "completed", 38.01, "r7"))
        self.assertFalse(driver.r6_resume_completion_is_positive(
            completion, process, "completed", 20.0, "r6"))

    def test_first_admission_claim_closes_any_retry(self):
        self.assertTrue(driver.r6_validation_claim_available(False, False))
        self.assertFalse(driver.r6_validation_claim_available(True, False))
        self.assertFalse(driver.r6_validation_claim_available(False, True))

    def test_incomplete_outer_or_native_queue_never_authorizes_resume(self):
        self.assertFalse(driver.r6_resume_authorized("incomplete", "complete", "completed"))
        self.assertFalse(driver.r6_resume_authorized("completed", "incomplete", "completed"))
        self.assertFalse(driver.r6_resume_authorized("completed", "complete", "incomplete"))
        self.assertTrue(driver.r6_resume_authorized("completed", "complete", "completed"))

    def test_freeze_delta_is_confined_to_r7_evidence(self):
        prefix = "experiments/government-comparison/evidence/government-check-receipt-native-20261008-r7/"
        self.assertTrue(driver.r6_source_changes_confined(
            [prefix + "preflight-freeze.json", prefix + "terminal-result.json"],
            driver.native_profiles.R7.evidence_directory))
        self.assertFalse(driver.r6_source_changes_confined(
            [prefix + "preflight-freeze.json", "experiments/government-comparison/runtime/dispatch.py"],
            driver.native_profiles.R7.evidence_directory))


if __name__ == "__main__":
    unittest.main()
