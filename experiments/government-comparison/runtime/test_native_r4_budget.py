"""Offline R4 grant and additive-ledger checks; no product/delegate process runs."""
import hashlib
import json
from pathlib import Path
import shutil
import tempfile
import unittest
from unittest.mock import patch

import native_fixture_budget as budget


class NativeR4BudgetTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.request = self._request()

    def _request(self):
        base = Path(budget.R3_BASE_GRANT_PATH)
        envelope = Path(budget.R4_ENVELOPE_PATH)
        source = Path(budget.R4_SOURCE_PATH)
        executable = Path(
            r"C:\Users\Consiliari\.codex\worktrees\government-worker\Markitect\.artifacts\government-g5\markitect-04e225d.exe")
        return {
            "mode": "mechanical", "arm": "government", "dispatchId": budget.R4_DISPATCH_ID,
            # The Request pins the disposable actor fixture revision, distinct from the grant's source base.
            "baseCommit": "20c0b4c85135bc5d76fe4d337ff5450b8d7178bf",
            "nativeFixtureGrant": {"path": str(base), "sha256": budget.R3_BASE_GRANT_SHA256,
                                   "sourceKey": budget.KEY},
            "nativeFixtureR4Grant": {"path": str(envelope), "sha256": budget.R4_ENVELOPE_SHA256,
                                     "sourceKey": budget.R4_KEY},
            "releasedInputs": [
                {"path": str(base), "sha256": budget.R3_BASE_GRANT_SHA256},
                {"path": str(envelope), "sha256": budget.R4_ENVELOPE_SHA256},
                {"path": str(source), "sha256": budget.R4_SOURCE_SHA256}],
            "product": {"government": {"executable": {
                "path": str(executable), "sha256": budget.R4_BINARY_SHA256,
                "sourceCommit": budget.R4_SOURCE_SHA}}}}

    def _validated(self):
        return budget.validate_r4_grant_binding(
            self.request, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256)

    def _live_state(self, path, *, owner="Scientist", key=None, status=None, slot_time=None, grant_time=None):
        state = json.loads(Path(budget.R4_SOURCE_PATH).read_text(encoding="utf-8"))
        slot = state["fullSuiteSlot"]
        slot["owner"] = owner
        slot.pop("assignmentKey", None)
        slot.pop("grantKey", None)
        slot["key"] = key if key is not None else budget.R4_KEY
        assigned = slot_time or "2026-10-08T01:30:00Z"
        slot["assignedUtc"] = assigned
        grant = next(t for t in state["threads"] if t["name"] == "Scientist")[
            "evidence"]["governmentSerializationNativeGrant"]
        grant["status"] = status or budget.R4_ACTIVE_STATUS
        grant["sentUtc"] = "2026-10-08T01:30:00Z"
        grant["slotAssignedUtc"] = grant_time or assigned
        path.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
        return path

    def test_real_r4_grant_binds_exact_source_product_delegate_and_request(self):
        admitted = self._validated()
        self.assertEqual(admitted["grantKey"], budget.R4_KEY)
        self.assertEqual(admitted["sourceCoordinationSha256"], budget.R4_SOURCE_SHA256)
        self.assertEqual(admitted["maxNativeStarts"], 2)
        self.assertEqual(admitted["maxWrapperAttempts"], 6)
        self.assertEqual(admitted["maxDeterministicDelegates"], 6)
        self.assertEqual(admitted["maxRoleStarts"], 6)
        self.assertEqual(admitted["maxRoleParallel"], 2)
        self.assertEqual(admitted["maxRoleProcessSeconds"], 38)
        self.assertNotEqual(self.request["baseCommit"], admitted["grant"]["baseSha"])
        self.assertEqual(admitted["product"]["delegateSha256"], budget.R4_DELEGATE_SHA256)

    def test_r4_binding_rejects_wrong_dispatch_closed_grant_and_changed_product_pins(self):
        mutations = [
            ("dispatchId", "government-native-contract-corrected-r3"),
            ("arm", "classic"),
            ("mode", "live"),
        ]
        for field, value in mutations:
            with self.subTest(field=field):
                request = dict(self.request, **{field: value})
                with self.assertRaises(ValueError):
                    budget.validate_r4_grant_binding(
                        request, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256)
        for field in ("nativeFixtureCorrection", "nativeFixtureR3Grant"):
            request = dict(self.request, **{field: {"reused": True}})
            with self.assertRaisesRegex(ValueError, "cannot reuse"):
                budget.validate_r4_grant_binding(
                    request, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256)
        changed = json.loads(json.dumps(self.request))
        changed["product"]["government"]["executable"]["sourceCommit"] = "0" * 40
        with self.assertRaisesRegex(ValueError, "exact Government executable"):
            budget.validate_r4_grant_binding(
                changed, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256)

    def test_r4_fresh_check_receipts_match_go_json_gate_result_fields_and_bounds(self):
        # Runtime check definitions use lower-case JSON tags; Go GateResult has yaml-only tags,
        # so encoding/json emits the exported field names below.
        definitions = [{"name": "unit", "tool": "go"}, {"name": "policy", "tool": "go"}]
        native_gate_results = [
            {"Name": "unit", "Tool": "go", "ExitCode": 0, "Milliseconds": 12,
             "TimeoutMilliseconds": 1000},
            {"Name": "policy", "Tool": "go", "ExitCode": 0, "Milliseconds": 20,
             "TimeoutMilliseconds": 2000}]
        self.assertTrue(budget._validate_r4_fresh_check_receipts(definitions, native_gate_results))
        wrong_case = [{"name": "unit", "tool": "go", "exitCode": 0,
                       "milliseconds": 12, "timeoutMilliseconds": 1000}, native_gate_results[1]]
        with self.assertRaisesRegex(ValueError, "configured check names/tools"):
            budget._validate_r4_fresh_check_receipts(definitions, wrong_case)
        nonzero = json.loads(json.dumps(native_gate_results))
        nonzero[0]["ExitCode"] = 1
        with self.assertRaisesRegex(ValueError, "passing bounded check"):
            budget._validate_r4_fresh_check_receipts(definitions, nonzero)
        over_timeout = json.loads(json.dumps(native_gate_results))
        over_timeout[0]["Milliseconds"] = 1001
        with self.assertRaisesRegex(ValueError, "passing bounded check"):
            budget._validate_r4_fresh_check_receipts(definitions, over_timeout)

    def test_r4_entry_gate_requires_exact_live_status_owner_key_and_activation_time(self):
        admitted = self._validated()
        live = self._live_state(self.root / "live-r4.json")
        with patch.object(budget, "R4_COORDINATION_PATH", str(live)):
            gate = budget.validate_r4_entry_gate(admitted)
            self.assertEqual((gate["slotOwner"], gate["slotKey"], gate["grantKey"]),
                             ("Scientist", budget.R4_KEY, budget.R4_KEY))
            for status in ("Prepared", "Revoked", "Closed", "Active but arbitrary"):
                bad = self._live_state(self.root / f"status-{hash(status)}.json", status=status)
                with patch.object(budget, "R4_COORDINATION_PATH", str(bad)):
                    with self.assertRaisesRegex(ValueError, "exact active assignment status"):
                        budget.validate_r4_entry_gate(admitted)
            for kwargs, message in (({"owner": "Worker"}, "not explicitly assigned"),
                                    ({"key": "another-grant"}, "exactly one matching"),
                                    ({"slot_time": "2026-10-08T01:31:00Z",
                                      "grant_time": "2026-10-08T01:30:00Z"}, "slotAssignedUtc must exactly match"),
                                    ({"grant_time": "2026-10-08T01:31:00Z"}, "slotAssignedUtc must exactly match")):
                with self.subTest(kwargs=kwargs):
                    bad = self._live_state(self.root / f"bad-{len(list(self.root.iterdir()))}.json", **kwargs)
                    with patch.object(budget, "R4_COORDINATION_PATH", str(bad)):
                        with self.assertRaisesRegex(ValueError, message):
                            budget.validate_r4_entry_gate(admitted)

    def test_r4_budget_requires_actual_translated_acceptance_before_resume(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(budget.R4_HISTORY_PATH, path)
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        binary = self.request["product"]["government"]["executable"]["path"]
        # FixtureBudget also validates both source-grant products; these are hashes only, never run.
        classic = r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe"
        native_budget = budget.FixtureBudget(
            path, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256,
            {"government": binary, "classic": classic}, request=self.request)
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)
        worker_live = self._live_state(self.root / "live-r4-worker.json", owner="Worker")
        with patch.object(budget, "R4_COORDINATION_PATH", str(worker_live)):
            with self.assertRaisesRegex(ValueError, "not explicitly assigned"):
                native_budget.reserve("government", "government-native-serialization-r4/queue",
                                     [binary, "government", "--action", "queue"])
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)
        live = self._live_state(self.root / "live-r4-active.json")
        argv = [binary, "government", "--action", "queue"]
        with patch.object(budget, "R4_COORDINATION_PATH", str(live)):
            with self.assertRaisesRegex(ValueError, "queue followed only by its associated resume"):
                native_budget.reserve("government", "government-native-serialization-r4/resume", argv)
            self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)
            native_budget.reserve("government", "government-native-serialization-r4/queue", argv)
            with self.assertRaisesRegex(ValueError, "completed successful queue process"):
                native_budget.reserve("government", "government-native-serialization-r4/resume", argv)
            native_budget.finish("government", "government-native-serialization-r4/queue",
                                 {"argv": argv, "returnCode": 0, "wallSeconds": 1})
            with self.assertRaisesRegex(ValueError, "exact Request roleAuthorization binding"):
                native_budget.reserve("government", "government-native-serialization-r4/resume",
                                      [binary, "government", "--action", "resume"])
        snapshot = native_budget.snapshot()
        added = [row for row in snapshot["starts"] if row["label"] in budget.R4_LABELS]
        self.assertEqual(len(added), 1)
        self.assertEqual(sum(row["reserved_seconds"] for row in added), 150)
        self.assertEqual(len(snapshot["starts"]), 12)

    def test_r4_resume_requires_successful_completed_queue_process_receipt(self):
        path = self.root / "native-starts-failed-queue.sqlite"
        shutil.copy2(budget.R4_HISTORY_PATH, path)
        binary = self.request["product"]["government"]["executable"]["path"]
        classic = r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe"
        native_budget = budget.FixtureBudget(
            path, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256,
            {"government": binary, "classic": classic}, request=self.request)
        live = self._live_state(self.root / "live-r4-failed-queue.json")
        queue_argv = [binary, "government", "--action", "queue"]
        resume_argv = [binary, "government", "--action", "resume"]
        with patch.object(budget, "R4_COORDINATION_PATH", str(live)):
            native_budget.reserve("government", "government-native-serialization-r4/queue", queue_argv)
            native_budget.finish("government", "government-native-serialization-r4/queue",
                                 {"argv": queue_argv, "returnCode": 2, "wallSeconds": 1})
            with self.assertRaisesRegex(ValueError, "successful queue process"):
                native_budget.reserve("government", "government-native-serialization-r4/resume", resume_argv)

    def test_r4_request_without_r4_binding_fails_closed_at_budget_construction(self):
        path = self.root / "native-starts.sqlite"
        shutil.copy2(budget.R4_HISTORY_PATH, path)
        request = dict(self.request)
        request.pop("nativeFixtureR4Grant")
        with self.assertRaisesRegex(ValueError, "exact R4 fixture grant is required"):
            budget.FixtureBudget(path, budget.R3_BASE_GRANT_PATH, budget.R3_BASE_GRANT_SHA256,
                                 {"government": self.request["product"]["government"]["executable"]["path"],
                                  "classic": r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe"},
                                 request=request)


if __name__ == "__main__":
    unittest.main()
