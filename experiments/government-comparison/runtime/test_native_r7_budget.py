"""Offline R7 grant and additive-ledger checks; no experiment process is run."""
import hashlib
import json
from pathlib import Path
import shutil
import tempfile
import unittest
from unittest.mock import patch

import government_native_profile as native_profile
import native_fixture_budget as budget


class NativeR7BudgetTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.profile = native_profile.profile("r7")
        self.request = self._request()

    def _request(self):
        base = Path(budget.R3_BASE_GRANT_PATH)
        executable = Path(
            r"C:\Users\Consiliari\.codex\worktrees\government-worker\Markitect\.artifacts\government-g5\markitect-04e225d.exe")
        return {
            "mode": "mechanical", "arm": "government", "dispatchId": self.profile.dispatch_id,
            "task": {"id": self.profile.task_id},
            "nativeFixtureGrant": {"path": str(base), "sha256": budget.R3_BASE_GRANT_SHA256,
                                   "sourceKey": budget.KEY},
            self.profile.marker: {"path": str(self.profile.envelope_path),
                                  "sha256": self.profile.envelope_sha, "sourceKey": self.profile.key},
            "releasedInputs": [
                {"path": str(base), "sha256": budget.R3_BASE_GRANT_SHA256},
                {"path": str(self.profile.envelope_path), "sha256": self.profile.envelope_sha},
                {"path": str(self.profile.snapshot_path), "sha256": self.profile.snapshot_sha}],
            "product": {"government": {"executable": {
                "path": str(executable), "sha256": budget.R5_BINARY_SHA256,
                "sourceCommit": budget.R5_SOURCE_SHA}}}}

    def _admitted(self):
        return budget.validate_r7_grant_binding(
            self.request, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256)

    def _live_copy(self, name, *, grant_change=None, slot_change=None):
        state = json.loads(self.profile.snapshot_path.read_text(encoding="utf-8"))
        scientist = next(item for item in state["threads"] if item["name"] == "Scientist")
        grant = scientist["evidence"]["governmentCheckReceiptNativeGrant"]
        if grant_change:
            grant.update(grant_change)
        if slot_change:
            state["fullSuiteSlot"].update(slot_change)
        path = self.root / name
        path.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
        return path

    def _budget(self, path):
        binary = self.request["product"]["government"]["executable"]["path"]
        classic = Path(
            r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe")
        return budget.FixtureBudget(
            path, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256,
            {"government": binary, "classic": str(classic)}, request=self.request)

    def test_r7_grant_binds_exact_profiles_task_products_python_and_quota_history(self):
        admitted = self._admitted()
        self.assertEqual(admitted["profileName"], "r7")
        self.assertEqual(admitted["dispatchId"], self.profile.dispatch_id)
        self.assertEqual(admitted["grantKey"], self.profile.key)
        self.assertEqual(admitted["sourceCoordinationSha256"], self.profile.snapshot_sha)
        self.assertEqual(admitted["product"]["sourceSha"], budget.R5_SOURCE_SHA)
        self.assertEqual(admitted["product"]["binarySha256"], budget.R5_BINARY_SHA256)
        self.assertEqual(admitted["product"]["delegateSha256"], budget.R5_DELEGATE_SHA256)
        self.assertEqual(admitted["historicalConsumed"], {
            "nativeStarts": 13, "wrapperAttempts": 13, "delegates": 10,
            "reservedSessionSeconds": 1950})
        self.assertEqual(admitted["cumulativeMaxNativeStarts"], 15)
        self.assertEqual(admitted["cumulativeMaxWrapperAttempts"], 19)
        self.assertEqual(admitted["cumulativeMaxDelegates"], 16)
        self.assertEqual(admitted["cumulativeMaxReservedSessionSeconds"], 2250)
        self.assertEqual(admitted["maxActualInputValidations"], 1)
        self.assertEqual(admitted["maxFreshStaticCasePreparations"], 1)
        self.assertIsNone(admitted["grant"]["historicalRealUsage"]["totalTokens"])

    def test_r7_rejects_mixed_closed_profiles_wrong_dispatch_and_wrong_task(self):
        self.request["nativeFixtureR6Grant"] = self.request.pop(self.profile.marker)
        with self.assertRaisesRegex(ValueError, "exact separate R5, R6 or R7"):
            self._admitted()
        self.request.pop("nativeFixtureR6Grant")
        self.request[self.profile.marker] = {"path": str(self.profile.envelope_path),
                                             "sha256": self.profile.envelope_sha,
                                             "sourceKey": self.profile.key}
        self.request["dispatchId"] = native_profile.profile("r6").dispatch_id
        with self.assertRaisesRegex(ValueError, "R7 grant is restricted"):
            self._admitted()
        self.request["dispatchId"] = self.profile.dispatch_id
        self.request["task"]["id"] = "positive-overflow-release-r6"
        with self.assertRaisesRegex(ValueError, "exact released task identity"):
            self._admitted()

    def test_live_entry_gate_requires_exact_frozen_grant_slot_key_and_time(self):
        admitted = self._admitted()
        with patch.object(budget, "R5_COORDINATION_PATH", str(self.profile.snapshot_path)):
            live = budget.validate_r7_entry_gate(admitted)
        self.assertEqual(live["slotOwner"], "Scientist")
        self.assertEqual(live["slotKey"], self.profile.key)
        negatives = (
            ("grant-sent.json", {"sentUtc": "2026-10-08T04:14:00Z"}, None),
            ("grant-status.json", {"status": "Closed"}, None),
            ("slot-time.json", None, {"assignedUtc": "2026-10-08T04:11:00Z"}),
            ("slot-status.json", None, {"status": "Released"}),
            ("slot-owner.json", None, {"owner": "Worker"}),
            ("slot-key.json", None, {"key": native_profile.profile("r6").key}),
        )
        for name, grant_change, slot_change in negatives:
            with self.subTest(name=name):
                changed = self._live_copy(name, grant_change=grant_change, slot_change=slot_change)
                with patch.object(budget, "R5_COORDINATION_PATH", str(changed)):
                    with self.assertRaises(ValueError):
                        budget.validate_r7_entry_gate(admitted)

    def test_budget_opens_exact_13_row_prefix_and_exposes_r7_grant(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(self.profile.history_path, path)
        original = hashlib.sha256(path.read_bytes()).hexdigest()
        instance = self._budget(path)
        self.assertEqual(instance.profile, self.profile)
        self.assertEqual(instance.r7["grantKey"], self.profile.key)
        self.assertEqual(len(budget._history_rows(path)[1]), 13)
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), original)

    def test_r7_queue_append_is_additive_and_resume_requires_complete_queue_proof(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(self.profile.history_path, path)
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        instance = self._budget(path)
        live = self._live_copy("live-r7.json")
        argv = [self.request["product"]["government"]["executable"]["path"], "government", "--action", "queue"]
        label = f"{self.profile.dispatch_id}/queue"
        with patch.object(budget, "R5_COORDINATION_PATH", str(live)):
            instance.reserve("government", label, argv)
            with self.assertRaisesRegex(ValueError, "already claimed"):
                instance.reserve("government", label, argv)
            instance.finish("government", label, {"argv": argv, "returnCode": 0, "status": "complete"})
            with self.assertRaisesRegex(ValueError, "roleAuthorization binding"):
                instance.reserve("government", f"{self.profile.dispatch_id}/resume", argv)
        allocation, starts, corrections = budget._history_rows(path)
        self.assertEqual(len(starts), 14)
        self.assertEqual([row[1] for row in starts if row[1].startswith(f"{self.profile.dispatch_id}/")], [label])
        self.assertEqual(len(corrections), 5)
        self.assertEqual([row[0] for row in corrections if row[0] == self.profile.key], [self.profile.key])
        self.assertEqual(hashlib.sha256(self.profile.history_path.read_bytes()).hexdigest(),
                         self.profile.history_sha)
        self.assertNotEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)

    def test_r5_r6_labels_classic_and_wrong_r7_dispatch_cannot_consume(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(self.profile.history_path, path)
        instance = self._budget(path)
        live = self._live_copy("live-profile.json")
        gov = self.request["product"]["government"]["executable"]["path"]
        classic = str(Path(
            r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe"))
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        with patch.object(budget, "R5_COORDINATION_PATH", str(live)):
            for label in ("government-native-scope-r5/queue", "government-native-released-binding-r6/queue"):
                with self.subTest(label=label), self.assertRaisesRegex(ValueError, "label is not allocated"):
                    instance.reserve("government", label, [gov])
            with self.assertRaisesRegex(ValueError, "label is not allocated"):
                instance.reserve("classic", f"{self.profile.dispatch_id}/queue", [classic])
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)


if __name__ == "__main__":
    unittest.main()
