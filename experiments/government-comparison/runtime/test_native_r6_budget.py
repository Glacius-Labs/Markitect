"""Offline R6 profile grant/ledger checks; no experiment processes are run."""
import hashlib
import json
from pathlib import Path
import shutil
import tempfile
import unittest
from unittest.mock import patch

import government_native_profile as native_profile
import native_fixture_budget as budget


class NativeR6BudgetTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.profile = native_profile.profile("r6")
        self.request = self._request()

    def _request(self):
        base = Path(budget.R3_BASE_GRANT_PATH)
        envelope = self.profile.envelope_path
        source = self.profile.snapshot_path
        executable = Path(
            r"C:\Users\Consiliari\.codex\worktrees\government-worker\Markitect\.artifacts\government-g5\markitect-04e225d.exe")
        return {
            "mode": "mechanical", "arm": "government", "dispatchId": self.profile.dispatch_id,
            "task": {"id": self.profile.task_id},
            "nativeFixtureGrant": {"path": str(base), "sha256": budget.R3_BASE_GRANT_SHA256,
                                   "sourceKey": budget.KEY},
            self.profile.marker: {"path": str(envelope), "sha256": self.profile.envelope_sha,
                                  "sourceKey": self.profile.key},
            "releasedInputs": [
                {"path": str(base), "sha256": budget.R3_BASE_GRANT_SHA256},
                {"path": str(envelope), "sha256": self.profile.envelope_sha},
                {"path": str(source), "sha256": self.profile.snapshot_sha}],
            "product": {"government": {"executable": {
                "path": str(executable), "sha256": budget.R5_BINARY_SHA256,
                "sourceCommit": budget.R5_SOURCE_SHA}}}}

    def _admitted(self):
        return budget.validate_r6_grant_binding(
            self.request, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256)

    def _live_copy(self, name, *, grant_change=None, slot_change=None):
        state = json.loads(self.profile.snapshot_path.read_text(encoding="utf-8"))
        scientist = next(item for item in state["threads"] if item["name"] == "Scientist")
        grant = scientist["evidence"]["governmentReleasedBindingNativeGrant"]
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

    def test_r6_exact_profile_grant_pins_and_new_zero_extra_bounds(self):
        admitted = self._admitted()
        self.assertEqual(admitted["profileName"], "r6")
        self.assertEqual(admitted["dispatchId"], self.profile.dispatch_id)
        self.assertEqual(admitted["grantKey"], self.profile.key)
        self.assertEqual(admitted["sourceCoordinationSha256"], self.profile.snapshot_sha)
        self.assertEqual(admitted["product"]["sourceSha"], budget.R5_SOURCE_SHA)
        self.assertEqual(admitted["product"]["binarySha256"], budget.R5_BINARY_SHA256)
        self.assertEqual(admitted["product"]["delegateSha256"], budget.R5_DELEGATE_SHA256)
        self.assertEqual(admitted["maxNativeStarts"], 2)
        self.assertEqual(admitted["maxWrapperAttempts"], 6)
        self.assertEqual(admitted["maxDeterministicDelegates"], 6)
        self.assertEqual(admitted["maxActualInputValidations"], 1)
        self.assertEqual(admitted["maxFreshStaticCasePreparations"], 1)
        self.assertIsNone(admitted["grant"]["historicalRealUsage"]["totalTokens"])

    def test_r6_profile_rejects_r5_mixed_markers_wrong_dispatch_and_wrong_task(self):
        self.request["nativeFixtureR5Grant"] = self.request.pop(self.profile.marker)
        with self.assertRaisesRegex(ValueError, "exact separate R5 or R6"):
            budget.validate_profile_grant_binding(
                self.request, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256,
                self.profile)
        self.request.pop("nativeFixtureR5Grant")
        self.request[self.profile.marker] = {
            "path": str(self.profile.envelope_path), "sha256": self.profile.envelope_sha,
            "sourceKey": self.profile.key}
        self.request["dispatchId"] = native_profile.profile("r5").dispatch_id
        with self.assertRaisesRegex(ValueError, "R6 grant is restricted to the exact Government dispatch"):
            self._admitted()
        self.request["dispatchId"] = self.profile.dispatch_id
        self.request["task"]["id"] = "positive-overflow-release"
        with self.assertRaisesRegex(ValueError, "exact released task identity"):
            self._admitted()

    def test_live_gate_requires_frozen_grant_and_full_slot_equality(self):
        admitted = self._admitted()
        with patch.object(budget, "R5_COORDINATION_PATH", str(self.profile.snapshot_path)):
            result = budget.validate_r6_entry_gate(admitted)
        self.assertEqual(result["slotKey"], self.profile.key)
        bad_cases = (
            ("grant-time.json", {"sentUtc": "2026-10-08T02:29:00Z"}, None),
            ("grant-status.json", {"status": "Closed"}, None),
            ("slot-time.json", None, {"assignedUtc": "2026-10-08T02:28:00Z"}),
            ("slot-status.json", None, {"status": "Released"}),
            ("slot-owner.json", None, {"owner": "Worker"}),
            ("slot-key.json", None, {"key": native_profile.profile("r5").key}),
        )
        for name, grant_change, slot_change in bad_cases:
            with self.subTest(name=name):
                path = self._live_copy(name, grant_change=grant_change, slot_change=slot_change)
                with patch.object(budget, "R5_COORDINATION_PATH", str(path)):
                    with self.assertRaises(ValueError):
                        budget.validate_r6_entry_gate(admitted)

    def test_constructor_reads_the_immutable_twelve_start_prefix(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(self.profile.history_path, path)
        original = hashlib.sha256(path.read_bytes()).hexdigest()
        instance = self._budget(path)
        self.assertEqual(instance.profile, self.profile)
        self.assertEqual(instance.r6["grantKey"], self.profile.key)
        self.assertEqual(len(budget._history_rows(path)[1]), 12)
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), original)

    def test_queue_append_uses_r6_identity_and_resume_still_requires_full_positive_proof(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(self.profile.history_path, path)
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        instance = self._budget(path)
        live = self._live_copy("live.json")
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
        self.assertEqual(len(starts), 13)
        self.assertEqual([row[1] for row in starts if row[1].startswith(f"{self.profile.dispatch_id}/")], [label])
        self.assertEqual([row[0] for row in corrections if row[0] == self.profile.key], [self.profile.key])
        self.assertEqual(hashlib.sha256(self.profile.history_path.read_bytes()).hexdigest(),
                         budget.R5_HISTORY_SHA256)
        self.assertNotEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)

    def test_old_r5_label_and_classic_cannot_consume_r6_profile(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(self.profile.history_path, path)
        instance = self._budget(path)
        live = self._live_copy("live.json")
        gov = self.request["product"]["government"]["executable"]["path"]
        classic = str(Path(
            r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe"))
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        with patch.object(budget, "R5_COORDINATION_PATH", str(live)):
            with self.assertRaisesRegex(ValueError, "label is not allocated"):
                instance.reserve("government", "government-native-scope-r5/queue", [gov])
            with self.assertRaisesRegex(ValueError, "label is not allocated"):
                instance.reserve("classic", f"{self.profile.dispatch_id}/queue", [classic])
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)


if __name__ == "__main__":
    unittest.main()
