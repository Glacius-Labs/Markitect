"""Atomic fixture grant exhaustion and replay tests; no executable is run."""
import hashlib
import json
from pathlib import Path
import shutil
import tempfile
from concurrent.futures import ThreadPoolExecutor
import unittest
from unittest.mock import patch

import native_fixture_budget
from native_fixture_budget import (CORRECTION_BASIS, CORRECTION_KEY, CORRECTION_POINTER,
                                   KEY, PROCESS_SECONDS_RESERVED, SOURCE_THREAD,
                                   FixtureBudget)


class FixtureBudgetTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        binary = root / "unexecuted-fixture.bin"
        binary.write_bytes(b"never executed")
        self.argv = [str(binary), "public-fixture-only"]
        grant = {"key":"native-s1-integration-fixtures-20261008",
                 "products":[{"name":name,"binarySha256":hashlib.sha256(binary.read_bytes()).hexdigest()}
                             for name in ("Government","Classic")],
                 "realActorStartsAuthorized":0,"providerCallsAuthorized":0,"studyCellsAuthorized":0,
                 "perProduct":{"nativeCliOrControllerStartsMaximum":8,"deterministicRoleStartsMaximum":12,
                               "maxParallel":2,"totalProcessSecondsMaximum":1200}}
        source = root / "source.json"
        source.write_text(json.dumps({"threads":[{"name":"Scientist","evidence":{
            "nativeIntegrationPreparationGrant":grant}}]}))
        document = {"sourceThreadId":"01a11367-a781-7683-a20f-46e12614dcb4",
                    "sourceCoordinationPath":str(source),
                    "sourceCoordinationSha256":hashlib.sha256(source.read_bytes()).hexdigest(),"grant":grant}
        path = root / "grant.json"
        path.write_text(json.dumps(document))
        self.budget = FixtureBudget(root / "fixture.sqlite",path,
            hashlib.sha256(path.read_bytes()).hexdigest(),{"government":str(binary),"classic":str(binary)})
        self.root = root
        self.binary = binary
        self.grant_path = path
        self.grant_sha = hashlib.sha256(path.read_bytes()).hexdigest()
        self._source_pin = (native_fixture_budget.CORRECTION_SOURCE_PATH,
                            native_fixture_budget.CORRECTION_SOURCE_SHA256)
        self.addCleanup(self._restore_source_pin)

    def _restore_source_pin(self):
        native_fixture_budget.CORRECTION_SOURCE_PATH, native_fixture_budget.CORRECTION_SOURCE_SHA256 = self._source_pin

    def _correction_request(self, ledger_path):
        source_grant = json.loads(self.grant_path.read_text())
        grant = source_grant["grant"]
        corrected = {
            "key": CORRECTION_KEY,
            "basis": {"negativeHandoffSha": CORRECTION_BASIS, "previousAllocation": KEY,
                      "previousNativeStarts": {"government": 2, "classic": 1},
                      "previousReservedSessionSeconds": {"government": 300, "classic": 150},
                      "previousNativeWrapperAttempts": {"government": 1, "classic": 0},
                      "previousDelegateAttempts": 0},
            "limits": {"government": {"maxAdditionalNativeStarts": 2,
                                       "maxAdditionalRoleInvocationsIncludingFailedWrapperStarts": 6,
                                       "maxAdditionalReservedControllerAndRoleSessionSeconds": 300,
                                       "cases": "One queue attempt; one directly related resume/replay only after the positive case succeeds. No repeated inspect."},
                       "classic": {"maxAdditionalNativeStarts": 5,
                                   "maxAdditionalRoleInvocationsIncludingFailedWrapperStarts": 6,
                                   "maxAdditionalReservedControllerAndRoleSessionSeconds": 750,
                                   "cases": "One Execute, guarded Apply after bound external review, fresh Verify, Audit, and stale Apply replay; later steps only after their prerequisites pass."},
                       "maxParallelRoles": 2, "productsRunSequentially": True,
                       "cumulativeNativeStartCeiling": {"government": 4, "classic": 6, "total": 10},
                       "cumulativeReservedSessionSecondsCeiling": {"government": 600, "classic": 900, "total": 1500},
                       "previousOverallNativeStartCeiling": 16,
                       "previousOverallReservedSessionSecondsCeiling": 2400,
                       "nativeProcessDeadlineSeconds": 38,
                       "controllerWindowSeconds": {"government": 38, "classic": 180},
                       "reservedSessionSecondsPerNativeStart": PROCESS_SECONDS_RESERVED},
            "realActorCalls": 0, "providerCalls": 0, "metadataAppServerTrees": 0,
            "studyCells": 0, "newPurchases": False, "productMutations": False}
        source = self.root / "corrected-coordinator.json"
        source.write_text(json.dumps({"threads": [{"name": "Scientist", "evidence": {
            "correctedNativeIntegrationGrant": corrected}}]}, sort_keys=True))
        native_fixture_budget.CORRECTION_SOURCE_PATH = str(source.resolve())
        native_fixture_budget.CORRECTION_SOURCE_SHA256 = hashlib.sha256(source.read_bytes()).hexdigest()
        doc = {"sourceThreadId": SOURCE_THREAD, "sourceJsonPointer": CORRECTION_POINTER,
               "sourceCoordinationPath": str(source.resolve()),
               "sourceCoordinationSha256": hashlib.sha256(source.read_bytes()).hexdigest(),
               "grant": corrected}
        correction = self.root / "correction.json"
        correction.write_text(json.dumps(doc, sort_keys=True))
        old_binding = {"path": str(self.grant_path.resolve()), "sha256": self.grant_sha,
                       "sourceKey": KEY}
        correction_binding = {"path": str(correction.resolve()),
                              "sha256": hashlib.sha256(correction.read_bytes()).hexdigest(),
                              "sourceKey": CORRECTION_KEY}
        request = {"mode": "mechanical", "nativeFixtureGrant": old_binding,
                   "nativeFixtureCorrection": correction_binding,
                   "releasedInputs": [{"path": old_binding["path"], "sha256": old_binding["sha256"]},
                                      {"path": correction_binding["path"],
                                       "sha256": correction_binding["sha256"]},
                                      {"path": str(source.resolve()),
                                       "sha256": native_fixture_budget.CORRECTION_SOURCE_SHA256}]}
        return request

    def _seed_original_history(self, path):
        budget = FixtureBudget(path, self.grant_path, self.grant_sha,
                               {"government": str(self.binary), "classic": str(self.binary)})
        for product, label in (("government", "inspect-constitution"),
                               ("government", "government-native-positive/queue"),
                               ("classic", "classic-native-positive/execute")):
            argv = [str(self.binary), label]
            budget.reserve(product, label, argv)
            budget.finish(product, label, {"argv": argv, "wallSeconds": .5})

    def _r3_request(self, product="government"):
        base_path = Path(native_fixture_budget.R3_BASE_GRANT_PATH)
        r3_path = Path(native_fixture_budget.R3_ENVELOPE_PATH)
        source_path = Path(native_fixture_budget.R3_SOURCE_PATH)
        return {"mode": "mechanical", "arm": product,
                "dispatchId": native_fixture_budget.R3_DISPATCH_IDS[product],
                "nativeFixtureGrant": {"path": str(base_path), "sha256": native_fixture_budget.R3_BASE_GRANT_SHA256,
                                       "sourceKey": KEY},
                "nativeFixtureR3Grant": {"path": str(r3_path),
                                         "sha256": native_fixture_budget.R3_ENVELOPE_SHA256,
                                         "sourceKey": native_fixture_budget.R3_KEY},
                "releasedInputs": [
                    {"path": str(base_path), "sha256": native_fixture_budget.R3_BASE_GRANT_SHA256},
                    {"path": str(r3_path), "sha256": native_fixture_budget.R3_ENVELOPE_SHA256},
                    {"path": str(source_path), "sha256": native_fixture_budget.R3_SOURCE_SHA256}]}

    @staticmethod
    def _r3_binaries():
        return {
            "government": r"C:\Users\Consiliari\.codex\worktrees\government-worker\Markitect\.artifacts\government-g5\markitect-04e225d.exe",
            "classic": r"C:\Users\Consiliari\.codex\worktrees\standard-operating-model\Markitect\.artifacts\classic-readiness\v0.14.1-r1\artifacts\markitect-v0.14.1-windows-amd64.exe"}

    def _r3_live_slot(self, path):
        snapshot = json.loads(Path(native_fixture_budget.R3_SOURCE_PATH).read_text(encoding="utf-8"))
        snapshot["fullSuiteSlot"]["owner"] = "Scientist"
        snapshot["fullSuiteSlot"]["key"] = native_fixture_budget.R3_KEY
        assigned_utc = "2026-10-08T00:20:56Z"
        snapshot["fullSuiteSlot"]["assignedUtc"] = assigned_utc
        grant = next(item for item in snapshot["threads"] if item["name"] == "Scientist")[
            "evidence"]["contractCorrectedNativeIntegrationGrant"]
        grant["slotAssignedUtc"] = assigned_utc
        grant["status"] = native_fixture_budget.R3_ACTIVE_STATUS
        path.write_text(json.dumps(snapshot, sort_keys=True), encoding="utf-8")
        return path

    def _r3_budget(self, path, request=None):
        shutil.copy2(native_fixture_budget.R3_HISTORY_PATH, path)
        return FixtureBudget(path, native_fixture_budget.R3_BASE_GRANT_PATH,
                             native_fixture_budget.R3_BASE_GRANT_SHA256,
                             self._r3_binaries(), request=request or self._r3_request())

    def test_eight_spent_slots_cannot_refill_or_replay(self):
        for i in range(8):
            self.budget.reserve("government",str(i),self.argv)
            self.budget.finish("government",str(i),{"argv":self.argv,"wallSeconds":0.1})
        with self.assertRaisesRegex(ValueError,"exhausted"):
            self.budget.reserve("government","ninth",self.argv)
        with self.assertRaisesRegex(ValueError,"no retry"):
            self.budget.reserve("government","0",self.argv)
        self.assertEqual(sum(row["reserved_seconds"] for row in self.budget.snapshot()["starts"]),1200)

    def test_racing_same_claim_admits_exactly_one(self):
        def reserve(_):
            try:
                self.budget.reserve("classic","same",self.argv)
                return True
            except ValueError:
                return False
        with ThreadPoolExecutor(max_workers=2) as workers:
            self.assertEqual(sum(workers.map(reserve,range(2))),1)
        self.assertEqual(len(self.budget.snapshot()["starts"]),1)

    def test_r2_adds_only_exact_remaining_allocation_to_original_history(self):
        path = self.root / "original-native-starts.sqlite"
        self._seed_original_history(path)
        before = FixtureBudget(path, self.grant_path, self.grant_sha,
                               {"government": str(self.binary), "classic": str(self.binary)}).snapshot()
        self.assertEqual(len(before["starts"]), 3)
        request = self._correction_request(path)
        corrected = FixtureBudget(path, self.grant_path, self.grant_sha,
                                  {"government": str(self.binary), "classic": str(self.binary)},
                                  request=request)
        labels = {"government": sorted(("government-native-corrected-r2/queue",
                                         "government-native-corrected-r2/resume")),
                  "classic": sorted(("classic-native-corrected-r2/execute",
                                     "classic-native-corrected-r2/apply",
                                     "classic-native-corrected-r2/verify",
                                     "classic-native-corrected-r2/audit",
                                     "classic-native-corrected-r2/apply-replay"))}
        for product in ("government", "classic"):
            for label in labels[product]:
                argv = [str(self.binary), label]
                corrected.reserve(product, label, argv)
                corrected.finish(product, label, {"argv": argv, "wallSeconds": .5})
            if product == "government":
                with self.assertRaisesRegex(ValueError, "exhausted"):
                    corrected.reserve("government", "government-native-corrected-r2/extra",
                                     [str(self.binary), "extra"])
        snapshot = corrected.snapshot()
        self.assertEqual(len(snapshot["starts"]), 10)
        totals = {product: (sum(row["product"] == product for row in snapshot["starts"]),
                            sum(row["reserved_seconds"] for row in snapshot["starts"]
                                if row["product"] == product))
                  for product in ("government", "classic")}
        self.assertEqual(totals, {"government": (4, 600), "classic": (6, 900)})
        with self.assertRaisesRegex(ValueError, "exhausted"):
            corrected.reserve("classic", "classic-native-corrected-r2/extra", [str(self.binary), "extra"])
        self.assertEqual(len(FixtureBudget(path, self.grant_path, self.grant_sha,
                                           {"government": str(self.binary), "classic": str(self.binary)},
                                           request=request).snapshot()["starts"]), 10)
        with self.assertRaisesRegex(ValueError, "requires its exact Request-bound correction"):
            FixtureBudget(path, self.grant_path, self.grant_sha,
                          {"government": str(self.binary), "classic": str(self.binary)})

    def test_r2_requires_original_three_rows_and_serializes_products(self):
        request = self._correction_request(self.root / "empty.sqlite")
        with self.assertRaisesRegex(ValueError, "original nonempty"):
            FixtureBudget(self.root / "empty.sqlite", self.grant_path, self.grant_sha,
                          {"government": str(self.binary), "classic": str(self.binary)}, request=request)
        path = self.root / "existing-native-starts.sqlite"
        self._seed_original_history(path)
        corrected = FixtureBudget(path, self.grant_path, self.grant_sha,
                                  {"government": str(self.binary), "classic": str(self.binary)}, request=request)
        gov_label = "government-native-corrected-r2/queue"
        gov_argv = [str(self.binary), gov_label]
        corrected.reserve("government", gov_label, gov_argv)
        with self.assertRaisesRegex(ValueError, "parallelism"):
            corrected.reserve("classic", "classic-native-corrected-r2/execute",
                              [str(self.binary), "classic-native-corrected-r2/execute"])
        corrected.finish("government", gov_label, {"argv": gov_argv, "wallSeconds": 1})
        classic_label = "classic-native-corrected-r2/execute"
        classic_argv = [str(self.binary), classic_label]
        corrected.reserve("classic", classic_label, classic_argv)
        corrected.finish("classic", classic_label, {"argv": classic_argv, "wallSeconds": 1})
        with self.assertRaisesRegex(ValueError, "sequentially"):
            corrected.reserve("government", "government-native-corrected-r2/resume",
                              [str(self.binary), "government-native-corrected-r2/resume"])

    def test_r2_rejects_replaced_original_binding_or_correction_source(self):
        path = self.root / "bound-native-starts.sqlite"
        self._seed_original_history(path)
        request = self._correction_request(path)
        altered = dict(request, nativeFixtureGrant={**request["nativeFixtureGrant"], "path": str(self.root / "copy.json")})
        (self.root / "copy.json").write_bytes(self.grant_path.read_bytes())
        with self.assertRaisesRegex(ValueError, "exact released source file"):
            FixtureBudget(path, self.grant_path, self.grant_sha,
                          {"government": str(self.binary), "classic": str(self.binary)}, request=altered)
        correction = Path(request["nativeFixtureCorrection"]["path"])
        correction.write_text(correction.read_text() + " ")
        with self.assertRaisesRegex(ValueError, "digest mismatch"):
            FixtureBudget(path, self.grant_path, self.grant_sha,
                          {"government": str(self.binary), "classic": str(self.binary)}, request=request)

    def test_r3_validator_binds_exact_grant_source_and_product_caps(self):
        request = self._r3_request("classic")
        validated = native_fixture_budget.validate_r3_grant_binding(
            request, native_fixture_budget.R3_BASE_GRANT_PATH,
            native_fixture_budget.R3_BASE_GRANT_SHA256)
        self.assertEqual(validated["grantKey"], native_fixture_budget.R3_KEY)
        self.assertEqual(validated["maxRoleStarts"], 6)
        self.assertEqual(validated["maxDeterministicDelegates"], 6)
        self.assertEqual(validated["maxNativeStarts"], 5)
        self.assertEqual(validated["maxReservedSessionSeconds"], 750)
        self.assertEqual(validated["products"], native_fixture_budget.R3_BASE_PRODUCTS)
        government = native_fixture_budget.validate_r3_grant_binding(
            self._r3_request("government"), native_fixture_budget.R3_BASE_GRANT_PATH,
            native_fixture_budget.R3_BASE_GRANT_SHA256)
        self.assertEqual((government["maxRoleStarts"], government["maxDeterministicDelegates"],
                          government["maxNativeStarts"], government["maxReservedSessionSeconds"]),
                         (6, 6, 2, 300))

        missing = dict(request)
        missing.pop("nativeFixtureR3Grant")
        with self.assertRaisesRegex(ValueError, "exact nativeFixtureR3Grant"):
            native_fixture_budget.validate_r3_grant_binding(
                missing, native_fixture_budget.R3_BASE_GRANT_PATH,
                native_fixture_budget.R3_BASE_GRANT_SHA256)
        missing_budget_request = dict(request)
        missing_budget_request.pop("nativeFixtureR3Grant")
        missing_budget_path = self.root / "r3-missing-grant.sqlite"
        shutil.copy2(native_fixture_budget.R3_HISTORY_PATH, missing_budget_path)
        before = hashlib.sha256(missing_budget_path.read_bytes()).hexdigest()
        with self.assertRaisesRegex(ValueError, "required for the corrected R3 dispatch"):
            FixtureBudget(missing_budget_path, native_fixture_budget.R3_BASE_GRANT_PATH,
                          native_fixture_budget.R3_BASE_GRANT_SHA256,
                          self._r3_binaries(), request=missing_budget_request)
        self.assertEqual(hashlib.sha256(missing_budget_path.read_bytes()).hexdigest(), before)
        altered = dict(request, nativeFixtureR3Grant={
            **request["nativeFixtureR3Grant"], "sha256": "0" * 64})
        with self.assertRaisesRegex(ValueError, "path or digest"):
            native_fixture_budget.validate_r3_grant_binding(
                altered, native_fixture_budget.R3_BASE_GRANT_PATH,
                native_fixture_budget.R3_BASE_GRANT_SHA256)
        reused_r2 = dict(request, nativeFixtureCorrection={"path": str(self.grant_path),
                                                           "sha256": self.grant_sha,
                                                           "sourceKey": CORRECTION_KEY})
        with self.assertRaisesRegex(ValueError, "cannot reuse"):
            native_fixture_budget.validate_r3_grant_binding(
                reused_r2, native_fixture_budget.R3_BASE_GRANT_PATH,
                native_fixture_budget.R3_BASE_GRANT_SHA256)

    def test_r3_worker_slot_fails_closed_without_writing_original_history(self):
        path = self.root / "r3-readonly-native-starts.sqlite"
        budget = self._r3_budget(path)
        before = hashlib.sha256(path.read_bytes()).hexdigest()
        live_path = self.root / "live-coordination-worker.json"
        state = json.loads(Path(native_fixture_budget.R3_SOURCE_PATH).read_text(encoding="utf-8"))
        state["fullSuiteSlot"]["owner"] = "Worker"
        live_path.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
        with patch.object(native_fixture_budget, "R3_COORDINATION_PATH",
                          str(live_path)):
            preflight = budget.preflight_snapshot()
            self.assertFalse(preflight["entryGate"]["ready"])
            with self.assertRaisesRegex(ValueError, "not explicitly assigned"):
                budget.reserve("government", "government-native-contract-corrected-r3/queue",
                               [self._r3_binaries()["government"], "government"])
        self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), before)
        self.assertEqual(before, native_fixture_budget.R3_HISTORY_SHA256)

    def test_r3_live_assignment_timestamp_must_match_grant_and_slot(self):
        validated = native_fixture_budget.validate_r3_grant_binding(
            self._r3_request(), native_fixture_budget.R3_BASE_GRANT_PATH,
            native_fixture_budget.R3_BASE_GRANT_SHA256)
        live_path = self._r3_live_slot(self.root / "live-coordination-timestamp.json")
        state = json.loads(live_path.read_text(encoding="utf-8"))
        grant = next(item for item in state["threads"] if item["name"] == "Scientist")[
            "evidence"]["contractCorrectedNativeIntegrationGrant"]
        grant["status"] = native_fixture_budget.R3_ACTIVE_STATUS
        grant["sentUtc"] = "2026-10-08T00:20:56Z"
        live_path.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
        with patch.object(native_fixture_budget, "R3_COORDINATION_PATH", str(live_path)):
            gate = native_fixture_budget.validate_r3_entry_gate(validated)
            self.assertEqual(gate["slotKey"], native_fixture_budget.R3_KEY)
            for status in ("Closed", "revoked", "active but unrecognized"):
                state = json.loads(live_path.read_text(encoding="utf-8"))
                status_grant = next(item for item in state["threads"] if item["name"] == "Scientist")[
                    "evidence"]["contractCorrectedNativeIntegrationGrant"]
                status_grant["status"] = status
                live_path.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
                with self.assertRaisesRegex(ValueError, "exact active assignment status"):
                    native_fixture_budget.validate_r3_entry_gate(validated)
            state = json.loads(live_path.read_text(encoding="utf-8"))
            status_grant = next(item for item in state["threads"] if item["name"] == "Scientist")[
                "evidence"]["contractCorrectedNativeIntegrationGrant"]
            status_grant["status"] = native_fixture_budget.R3_ACTIVE_STATUS
            live_path.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
            state = json.loads(live_path.read_text(encoding="utf-8"))
            state["fullSuiteSlot"]["grantKey"] = "another-grant"
            live_path.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "exactly one matching explicit grant key"):
                native_fixture_budget.validate_r3_entry_gate(validated)
            state["fullSuiteSlot"].pop("grantKey")
            state["fullSuiteSlot"]["assignedUtc"] = "2026-10-08T00:21:00Z"
            live_path.write_text(json.dumps(state, sort_keys=True), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "slotAssignedUtc must exactly match"):
                native_fixture_budget.validate_r3_entry_gate(validated)

    def test_r3_appends_only_granted_labels_and_preserves_all_five_prior_rows(self):
        path = self.root / "r3-cumulative-native-starts.sqlite"
        request = self._r3_request("government")
        budget = self._r3_budget(path, request)
        live_path = self._r3_live_slot(self.root / "live-coordination.json")
        with patch.object(native_fixture_budget, "R3_COORDINATION_PATH", str(live_path)):
            gate = native_fixture_budget.validate_r3_entry_gate(budget.r3)
            self.assertEqual((gate["slotOwner"], gate["slotKey"], gate["grantKey"]),
                             ("Scientist", native_fixture_budget.R3_KEY, native_fixture_budget.R3_KEY))
            self.assertTrue(budget.preflight_snapshot()["entryGate"]["ready"])

            binaries = self._r3_binaries()
            gov_labels = sorted(native_fixture_budget.R3_LABELS["government"])
            for label in gov_labels:
                argv = [binaries["government"], "--fixture", label]
                budget.reserve("government", label, argv)
                budget.finish("government", label, {"argv": argv, "wallSeconds": .5})
            with self.assertRaisesRegex(ValueError, "not allocated by the exact R3"):
                budget.reserve("government", "government-native-contract-corrected-r3/third",
                               [binaries["government"], "third"])

            for label in sorted(native_fixture_budget.R3_LABELS["classic"]):
                argv = [binaries["classic"], "--fixture", label]
                budget.reserve("classic", label, argv)
                budget.finish("classic", label, {"argv": argv, "wallSeconds": .5})
            with self.assertRaisesRegex(ValueError, "not allocated by the exact R3"):
                budget.reserve("classic", "classic-native-contract-corrected-r3/sixth",
                               [binaries["classic"], "sixth"])

        snapshot = budget.preflight_snapshot()
        self.assertEqual(len(snapshot["starts"]), 12)
        totals = {product: (sum(row["product"] == product for row in snapshot["starts"]),
                            sum(row["reserved_seconds"] for row in snapshot["starts"]
                                if row["product"] == product))
                  for product in ("government", "classic")}
        self.assertEqual(totals, {"government": (5, 750), "classic": (7, 1050)})
        base_allocation, base_starts, base_corrections = native_fixture_budget._history_rows(
            native_fixture_budget.R3_HISTORY_PATH)
        current_allocation, current_starts, current_corrections = native_fixture_budget._history_rows(path)
        prior_keys = {(product, label) for product, labels in native_fixture_budget.R3_PRIOR_LABELS.items()
                      for label in labels}
        self.assertEqual([row for row in current_starts if (row[0], row[1]) in prior_keys],
                         [row for row in base_starts if (row[0], row[1]) in prior_keys])
        self.assertEqual(current_allocation, base_allocation)
        self.assertEqual([row for row in current_corrections if row[0] != native_fixture_budget.R3_KEY],
                         base_corrections)
        self.assertEqual(len(current_corrections), 2)
