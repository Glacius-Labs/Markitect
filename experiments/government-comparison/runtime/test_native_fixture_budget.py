"""Atomic fixture grant exhaustion and replay tests; no executable is run."""
import hashlib
import json
from pathlib import Path
import tempfile
from concurrent.futures import ThreadPoolExecutor
import unittest

from native_fixture_budget import FixtureBudget


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
