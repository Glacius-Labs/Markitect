"""Pure offline R6 terminal predicates; no fixture preparation, process, or ledger access."""
from __future__ import annotations

import importlib.util
from pathlib import Path
import unittest

PACKAGE = Path(__file__).parents[1]
spec = importlib.util.spec_from_file_location(
    "r6_checkpoint_driver", PACKAGE / "run-native-integration-r5.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


class NativeR6CheckpointTests(unittest.TestCase):
    def setUp(self):
        self.request_sha = "a" * 64
        self.process = {"path": "C:/evidence/process/process.json", "sha256": "b" * 64}
        self.queue_receipt = {
            "status": "completed",
            "dispatchId": driver.native_profiles.R6.dispatch_id,
            "requestSha256": self.request_sha,
            "elapsedSecondsAfterTerminalReceiptAvailable": 37.9,
            "terminalResult": {"path": "C:/results/r6-queue.json", "sha256": "c" * 64},
            "terminalControllerReceiptSha256": "d" * 64,
            "nativeProcess": self.process,
        }

    def test_queue_requires_both_terminal_and_driver_deadlines(self):
        self.assertTrue(driver.r6_queue_completion_is_positive(
            self.queue_receipt, self.request_sha, self.process, 37.99))
        self.assertFalse(driver.r6_queue_completion_is_positive(
            self.queue_receipt, self.request_sha, self.process, 38.01))
        late_controller = dict(self.queue_receipt,
                               elapsedSecondsAfterTerminalReceiptAvailable=38.01)
        self.assertFalse(driver.r6_queue_completion_is_positive(
            late_controller, self.request_sha, self.process, 37.0))

    def test_queue_rejects_missing_or_mismatched_terminal_evidence(self):
        for mutate in (
            lambda value: value.update(status="incomplete"),
            lambda value: value.update(requestSha256="0" * 64),
            lambda value: value.update(terminalControllerReceiptSha256="not-a-sha"),
            lambda value: value.update(nativeProcess={"path": self.process["path"], "sha256": "0" * 64}),
            lambda value: value.update(terminalResult=None),
        ):
            with self.subTest(mutate=mutate):
                receipt = dict(self.queue_receipt)
                mutate(receipt)
                self.assertFalse(driver.r6_queue_completion_is_positive(
                    receipt, self.request_sha, self.process, 10.0))

    def test_resume_requires_terminal_readback_measurement_inside_deadline(self):
        process = {"returnCode": 0, "stopReason": None, "wallSeconds": 31.8}
        completion = {"profile": "r6", "status": "completed",
                      "elapsedSecondsAfterTerminalReceiptAvailable": 37.8}
        self.assertTrue(driver.r6_resume_completion_is_positive(
            completion, process, "completed", 37.99))
        # A pre-write 37.8s sample is insufficient when the receipt readback ends late.
        self.assertFalse(driver.r6_resume_completion_is_positive(
            completion, process, "completed", 38.01))
        late_completion = dict(completion, elapsedSecondsAfterTerminalReceiptAvailable=38.01)
        self.assertFalse(driver.r6_resume_completion_is_positive(
            late_completion, process, "completed", 37.0))

    def test_incomplete_queue_never_authorizes_resume(self):
        for statuses in (("incomplete", "complete", "completed"),
                         ("completed", "incomplete", "completed"),
                         ("completed", "complete", "incomplete")):
            self.assertFalse(driver.r6_resume_authorized(*statuses))
        self.assertTrue(driver.r6_resume_authorized("completed", "complete", "completed"))

    def test_actual_input_claim_is_terminal_after_first_claim_or_receipt(self):
        self.assertTrue(driver.r6_validation_claim_available(False, False))
        self.assertFalse(driver.r6_validation_claim_available(True, False))
        self.assertFalse(driver.r6_validation_claim_available(False, True))

    def test_freeze_allows_only_its_evidence_commit_delta(self):
        prefix = "experiments/government-comparison/evidence/government-released-binding-native-20261008-r6/"
        self.assertTrue(driver.r6_source_changes_confined(
            [prefix + "preflight-freeze.json", prefix + "terminal-result.json"],
            driver.native_profiles.R6.evidence_directory))
        self.assertFalse(driver.r6_source_changes_confined(
            [prefix + "preflight-freeze.json", "experiments/government-comparison/runtime/dispatch.py"],
            driver.native_profiles.R6.evidence_directory))


if __name__ == "__main__":
    unittest.main()
