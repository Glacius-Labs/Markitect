"""Atomic fixture grant exhaustion and replay tests; no executable is run."""
import hashlib
import json
from pathlib import Path
import tempfile
from concurrent.futures import ThreadPoolExecutor
import unittest

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
