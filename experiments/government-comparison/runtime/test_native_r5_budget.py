"""Offline R5 grant and additive-ledger checks; no experimental processes run."""
import hashlib
import json
from pathlib import Path
import shutil
import sqlite3
import tempfile
import unittest
from unittest.mock import patch

import native_fixture_budget as budget


class NativeR5BudgetTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.request = self._request()

    def _request(self):
        base = Path(budget.R3_BASE_GRANT_PATH)
        envelope = Path(budget.R5_ENVELOPE_PATH)
        source = Path(budget.R5_SOURCE_PATH)
        executable = Path(
            r"C:\Users\Consiliari\.codex\worktrees\government-worker\Markitect\.artifacts\government-g5\markitect-04e225d.exe")
        return {
            "mode": "mechanical", "arm": "government", "dispatchId": budget.R5_DISPATCH_ID,
            "nativeFixtureGrant": {"path": str(base), "sha256": budget.R3_BASE_GRANT_SHA256,
                                   "sourceKey": budget.KEY},
            "nativeFixtureR5Grant": {"path": str(envelope), "sha256": budget.R5_ENVELOPE_SHA256,
                                     "sourceKey": budget.R5_KEY},
            "releasedInputs": [
                {"path": str(base), "sha256": budget.R3_BASE_GRANT_SHA256},
                {"path": str(envelope), "sha256": budget.R5_ENVELOPE_SHA256},
                {"path": str(source), "sha256": budget.R5_SOURCE_SHA256}],
            "product": {"government": {"executable": {
                "path": str(executable), "sha256": budget.R5_BINARY_SHA256,
                "sourceCommit": budget.R5_SOURCE_SHA}}}}

    def _admitted(self):
        return budget.validate_r5_grant_binding(
            self.request, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256)

    def _live_state(self, path, *, owner="Scientist", key=None, status=None,
                    assigned=None, grant_assigned=None, slot_status=None):
        state = json.loads(Path(budget.R5_SOURCE_PATH).read_text(encoding="utf-8"))
        slot = state["fullSuiteSlot"]
        slot["owner"] = owner
        slot["key"] = budget.R5_KEY if key is None else key
        slot["assignedUtc"] = assigned or "2026-10-08T02:00:58Z"
        if slot_status is not None:
            slot["status"] = slot_status
        grant = next(item for item in state["threads"] if item["name"] == "Scientist")[
            "evidence"]["governmentScopeNativeGrant"]
        grant["status"] = status or budget.R5_ACTIVE_STATUS
        grant["sentUtc"] = "2026-10-08T02:01:52Z"
        grant["slotAssignedUtc"] = grant_assigned or slot["assignedUtc"]
        target = Path(path)
        target.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
        return target

    def test_released_r5_grant_binds_exact_source_product_delegate_python_and_limits(self):
        admitted = self._admitted()
        self.assertEqual(admitted["grantKey"], budget.R5_KEY)
        self.assertEqual(admitted["sourceCoordinationSha256"], budget.R5_SOURCE_SHA256)
        self.assertEqual(admitted["product"]["sourceSha"], budget.R5_SOURCE_SHA)
        self.assertEqual(admitted["product"]["binarySha256"], budget.R5_BINARY_SHA256)
        self.assertEqual(admitted["product"]["delegateSha256"], budget.R5_DELEGATE_SHA256)
        self.assertEqual(admitted["maxNativeStarts"], 2)
        self.assertEqual(admitted["maxWrapperAttempts"], 6)
        self.assertEqual(admitted["maxDeterministicDelegates"], 6)
        self.assertEqual(admitted["reservedSecondsPerNativeStart"], 150)
        self.assertEqual(admitted["nativeProcessDeadlineSeconds"], 38)
        self.assertEqual(admitted["cumulativeMaxNativeStarts"], 14)
        self.assertIsNone(admitted["grant"]["historicalRealUsage"]["totalTokens"])

    def test_request_dispatch_binding_and_closed_r4_grant_fail_closed(self):
        self.request["dispatchId"] = budget.R4_DISPATCH_ID
        with self.assertRaisesRegex(ValueError, "exact Government dispatch"):
            self._admitted()
        self.request["dispatchId"] = budget.R5_DISPATCH_ID
        self.request["nativeFixtureR4Grant"] = {"closed": True}
        with self.assertRaisesRegex(ValueError, "closed correction, R3, or R4"):
            self._admitted()

    def test_live_slot_gate_requires_exact_owner_key_status_and_assignment_time(self):
        admitted = self._admitted()
        good = self._live_state(self.root / "live.json")
        with patch.object(budget, "R5_COORDINATION_PATH", str(good)):
            result = budget.validate_r5_entry_gate(admitted)
        self.assertEqual(result["slotOwner"], "Scientist")
        self.assertEqual(result["slotKey"], budget.R5_KEY)
        for overrides in (
            {"owner": "Worker"}, {"key": "closed-key"}, {"status": "Closed"},
            {"assigned": "2026-10-08T02:01:00Z"},
            {"grant_assigned": "2026-10-08T02:01:00Z"},
            {"status": "Closed"}, {"slot_status": "Released"},
        ):
            bad = self._live_state(self.root / (str(len(list(self.root.iterdir()))) + ".json"), **overrides)
            with patch.object(budget, "R5_COORDINATION_PATH", str(bad)):
                with self.assertRaises(ValueError):
                    budget.validate_r5_entry_gate(admitted)

    def test_fixture_budget_reads_exact_twelve_start_prefix_without_mutating_it(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(budget.R5_HISTORY_PATH, path)
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        binary = self.request["product"]["government"]["executable"]["path"]
        classic = Path(r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe")
        instance = budget.FixtureBudget(
            path, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256,
            {"government": binary, "classic": str(classic)}, request=self.request)
        self.assertIsNotNone(instance.r5)
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)
        self.assertEqual(len(budget._history_rows(path)[1]), 12)

    def test_queue_reservation_is_additive_and_resume_requires_complete_queue_proof(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(budget.R5_HISTORY_PATH, path)
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        binary = self.request["product"]["government"]["executable"]["path"]
        classic = Path(r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe")
        live = self._live_state(self.root / "live-active.json")
        instance = budget.FixtureBudget(
            path, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256,
            {"government": binary, "classic": str(classic)}, request=self.request)
        argv = [binary, "government", "--action", "queue"]
        with patch.object(budget, "R5_COORDINATION_PATH", str(live)):
            instance.reserve("government", "government-native-scope-r5/queue", argv)
            with self.assertRaisesRegex(ValueError, "already claimed"):
                instance.reserve("government", "government-native-scope-r5/queue", argv)
            instance.finish("government", "government-native-scope-r5/queue",
                            {"argv": argv, "returnCode": 0, "status": "complete"})
            with self.assertRaisesRegex(ValueError, "roleAuthorization binding"):
                instance.reserve("government", "government-native-scope-r5/resume", argv)
        allocation, starts, corrections = budget._history_rows(path)
        self.assertEqual(len(starts), 13)
        self.assertEqual([row[1] for row in starts if row[1].startswith("government-native-scope-r5/")],
                         ["government-native-scope-r5/queue"])
        self.assertEqual(len(corrections), 4)
        self.assertEqual([row[0] for row in corrections if row[0] == budget.R5_KEY], [budget.R5_KEY])
        self.assertEqual(hashlib.sha256(budget.R5_HISTORY_PATH.read_bytes()).hexdigest(),
                         budget.R5_HISTORY_SHA256)
        self.assertNotEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)

    def test_wrong_product_and_label_do_not_consume_r5_allocation(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(budget.R5_HISTORY_PATH, path)
        binary = self.request["product"]["government"]["executable"]["path"]
        classic = Path(r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe")
        live = self._live_state(self.root / "live-active.json")
        instance = budget.FixtureBudget(
            path, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256,
            {"government": binary, "classic": str(classic)}, request=self.request)
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        with patch.object(budget, "R5_COORDINATION_PATH", str(live)):
            with self.assertRaisesRegex(ValueError, "label is not allocated"):
                instance.reserve("government", "government-native-serialization-r5/queue", [binary])
            with self.assertRaisesRegex(ValueError, "label is not allocated"):
                instance.reserve("classic", "government-native-scope-r5/queue", [str(classic)])
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)


if __name__ == "__main__":
    unittest.main()
