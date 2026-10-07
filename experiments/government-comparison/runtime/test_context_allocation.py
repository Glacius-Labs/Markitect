"""Focused synthetic tests for the one-session diagnostic context allocation."""
from contextlib import closing
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
import context_allocation as allocation
from ledger import LimitReached
from measurement_profile import LEGACY, OBSERVED, limits_sha, profile_sha

LIMITS = {
    "taskWallSeconds": 1200, "trialWallSeconds": 7200,
    "taskActorCalls": 12, "trialActorCalls": 72,
    "taskProviderTurns": 80, "trialProviderTurns": 480,
    "taskProviderTokens": 120000, "trialProviderTokens": 720000,
    "maxParallelActors": 4, "maxTransportRetriesPerCall": 1,
    "maxSemanticRepairRoundsPerTask": 2, "trialActiveHumanSeconds": 600,
    "newPurchases": False,
}


class ContextAllocationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="context-allocation-test-")
        self.root = Path(self.temp.name).resolve()
        self.old_path = allocation.ALLOCATION_PATH
        allocation.ALLOCATION_PATH = self.root / "external-allocation.sqlite"
        self.path = allocation.ALLOCATION_PATH

    def tearDown(self):
        allocation.ALLOCATION_PATH = self.old_path
        self.temp.cleanup()

    def make_ledger(self, *, path=None, trial_id=None, binding=None, profile_id=LEGACY):
        return allocation.ContextAllocationLedger(
            path or self.path,
            allocation.ALLOCATION_ID if trial_id is None else trial_id,
            dict(LIMITS),
            binding=allocation.allocation_binding() if binding is None else binding,
            profile_id=profile_id,
        )

    @staticmethod
    def approval(ledger):
        return {
            "status": "approved",
            "trialId": allocation.ALLOCATION_ID,
            "ledgerPath": str(Path(ledger.path).resolve()),
            "fromProfileSha256": profile_sha(LEGACY),
            "toProfileSha256": profile_sha(OBSERVED),
            "limitsSha256": limits_sha(ledger.limits),
            "decisionRef": "decision://synthetic-context-allocation/v2",
        }

    @staticmethod
    def authority(profile_id=OBSERVED):
        return {"grantId": "synthetic-one-session-grant", "protocolSha256": "a" * 64,
                "profileSha256": profile_sha(profile_id), "limitsSha256": limits_sha(LIMITS)}

    def initialize_v2(self, ledger, *, authority=None):
        ledger.adopt_profile(OBSERVED, self.approval(ledger))
        authority = authority or self.authority()
        ledger.bind_dispatch(authority, maximum=1)
        return authority

    def test_binding_and_single_context_session_preserve_predecessors_and_exact_dispatch(self):
        binding = allocation.allocation_binding()
        historical_hashes = dict(binding["predecessors"]["ledgerSha256After"])
        self.assertTrue(binding["predecessors"]["historicalLedgersUnchanged"])
        ledger = self.make_ledger(binding=binding)
        authority = self.initialize_v2(ledger)

        first = ledger.snapshot()
        self.assertEqual(first["measurementProfile"]["profileId"], OBSERVED)
        self.assertEqual(first["profileHistory"][-1]["profile_id"], OBSERVED)
        self.assertEqual([row["profile_id"] for row in first["profileHistory"]], [LEGACY, OBSERVED])
        self.assertEqual(first["profileHistory"][-1]["profile_sha"], profile_sha(OBSERVED))
        self.assertEqual(first["profileHistory"][-1]["prior_sha"], profile_sha(LEGACY))
        self.assertEqual(first["actorSessions"], 0)
        self.assertEqual(first["predecessorActorSessions"], 3)
        self.assertEqual(first["cumulativeActorSessions"], 3)
        self.assertIsNone(first["cumulativeProviderTokens"])
        self.assertIsNone(first["cumulativeProviderRequests"])
        self.assertEqual(first["knownCumulativeTokenSubtotal"], 10009)
        with closing(sqlite3.connect(self.path)) as db:
            self.assertEqual(db.execute("SELECT stopped FROM trial").fetchone()[0], 0)
            self.assertEqual(db.execute("SELECT id,binding FROM context_allocation").fetchall(),
                             [(allocation.ALLOCATION_ID, json.dumps(binding, sort_keys=True))])
            self.assertEqual(db.execute("SELECT authority_sha,profile_sha,ceiling FROM dispatch_authority_history").fetchall(),
                             [(limits_sha(authority), profile_sha(OBSERVED), 1)])

        dispatch_id = "synthetic-context-access-1"
        request = b'{"kind":"synthetic-context-access"}'
        command = ["synthetic-adapter", "--never-run"]
        attempt = ledger.reserve_dispatch(dispatch_id, "b" * 64, request, command,
                                          "context-access", "context-access", 1)
        self.assertTrue(ledger.claim_dispatch(dispatch_id, "launched"))
        result = {"status": "completed", "receipts": [{"source": "synthetic-only"}]}
        result = ledger.complete_dispatch(dispatch_id, result, 2, 9)

        resumed = self.make_ledger(binding=binding, profile_id=OBSERVED)
        resumed.bind_dispatch(authority, maximum=1)
        self.assertEqual(resumed.dispatch_record(dispatch_id)["attempt"], attempt)
        self.assertEqual(resumed.complete_dispatch(dispatch_id, {"status": "ignored", "receipts": []}, None, None), result)
        self.assertFalse(resumed.claim_dispatch(dispatch_id, "launched"))
        with self.assertRaisesRegex(LimitReached, "already booked; use resume"):
            resumed.reserve_dispatch(dispatch_id, "c" * 64, b'{"different":true}', command,
                                     "context-access", "context-access", 1)
        with self.assertRaisesRegex(LimitReached, "grant session limit"):
            resumed.reserve_dispatch("synthetic-context-access-2", "d" * 64, request, command,
                                     "context-access", "context-access", 1)
        with closing(sqlite3.connect(self.path)) as db:
            self.assertEqual(db.execute("SELECT stopped FROM trial").fetchone()[0], 0)
            self.assertEqual(db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0], 1)
            self.assertEqual(db.execute("SELECT COUNT(*) FROM dispatch_authority_history").fetchone()[0], 1)
        final = resumed.snapshot()
        self.assertEqual(final["actorSessions"], 1)
        self.assertEqual(final["cumulativeActorSessions"], 4)
        self.assertIsNone(final["cumulativeProviderTokens"])
        self.assertIsNone(final["cumulativeProviderRequests"])
        self.assertEqual(final["knownCumulativeTokenSubtotal"], 10018)
        after_binding = allocation.allocation_binding()
        self.assertEqual(after_binding["predecessors"]["ledgerSha256After"], historical_hashes)
        self.assertTrue(after_binding["predecessors"]["historicalLedgersUnchanged"])
        self.assertEqual(after_binding["predecessors"]["attempts"], binding["predecessors"]["attempts"])
        self.assertTrue(all(row["finished"] for row in after_binding["predecessors"]["attempts"]))

    def test_unknown_local_tokens_block_another_reservation_and_keep_null_cumulative_total(self):
        binding = allocation.allocation_binding()
        ledger = self.make_ledger(binding=binding)
        self.initialize_v2(ledger)
        attempt = ledger.reserve("context-access", "context-access")
        ledger.finish(attempt, "completed", provider_turns=None, provider_tokens=None,
                      receipt={"synthetic": "unknown-local-usage"})
        with self.assertRaisesRegex(LimitReached, "new context usage tokens unknown"):
            ledger.reserve("context-access", "context-access")
        snapshot = ledger.snapshot()
        self.assertEqual(snapshot["actorSessions"], 1)
        self.assertEqual(snapshot["cumulativeActorSessions"], 4)
        self.assertIsNone(snapshot["providerTokens"])
        self.assertIsNone(snapshot["cumulativeProviderTokens"])
        self.assertIsNone(snapshot["cumulativeProviderRequests"])
        self.assertEqual(snapshot["knownCumulativeTokenSubtotal"], 10009)

    def test_missing_or_changed_predecessor_identity_trial_and_path_reject_before_database_creation(self):
        good = allocation.allocation_binding()
        altered_hash = json.loads(json.dumps(good))
        altered_hash["predecessors"]["ledgerSha256After"][next(iter(altered_hash["predecessors"]["ledgerSha256After"]))] = "0" * 64
        missing_counter = json.loads(json.dumps(good))
        del missing_counter["predecessors"]["actorStartsConsumed"]
        altered_attempt_tokens = json.loads(json.dumps(good))
        altered_attempt_tokens["predecessors"]["attempts"][0]["tokens"] = 10010
        altered_attempt_requests = json.loads(json.dumps(good))
        altered_attempt_requests["predecessors"]["attempts"][0]["providerRequests"] = 99
        missing_attempt = json.loads(json.dumps(good))
        missing_attempt["predecessors"]["attempts"].pop()
        missing_runner_field = json.loads(json.dumps(good))
        runner_attempt = next(row for row in missing_runner_field["predecessors"]["attempts"]
                              if row["grant"] == "selected-runner-one-start")
        del runner_attempt["providerRequests"]
        cases = (
            ("altered-hash.sqlite", good, altered_hash, allocation.ALLOCATION_ID, self.path),
            ("missing-counter.sqlite", good, missing_counter, allocation.ALLOCATION_ID, self.path),
            ("altered-attempt-tokens.sqlite", good, altered_attempt_tokens, allocation.ALLOCATION_ID, self.path),
            ("altered-attempt-requests.sqlite", good, altered_attempt_requests, allocation.ALLOCATION_ID, self.path),
            ("missing-attempt.sqlite", good, missing_attempt, allocation.ALLOCATION_ID, self.path),
            ("missing-runner-field.sqlite", good, missing_runner_field, allocation.ALLOCATION_ID, self.path),
            ("wrong-trial.sqlite", good, good, "another-trial", self.path),
            ("wrong-path-target.sqlite", good, good, allocation.ALLOCATION_ID, self.root / "other.sqlite"),
        )
        for name, _, supplied_binding, trial_id, path in cases:
            with self.subTest(name=name):
                target = Path(path)
                if target.exists():
                    target.unlink()
                with self.assertRaisesRegex(ValueError, "fixed allocation identity/path/predecessors required"):
                    self.make_ledger(path=target, trial_id=trial_id, binding=supplied_binding)
                self.assertFalse(target.exists(), "invalid identity must fail before SQLite creates a file")

    def test_new_ledger_cannot_be_opened_as_allocation_recovery_without_profile_binding(self):
        with self.assertRaisesRegex(ValueError, "allocation resume requires existing ledger"):
            self.make_ledger(profile_id=None)
        self.assertFalse(self.path.exists(), "recovery-only open must not create an allocation ledger")

    def test_corrupt_durable_binding_rejects_reservation_before_creating_attempt(self):
        ledger = self.make_ledger()
        self.initialize_v2(ledger)
        with ledger.transaction() as db:
            corrupted = json.dumps({"decisionId": "synthetic-corruption"}, sort_keys=True)
            db.execute("UPDATE context_allocation SET binding=? WHERE id=?",
                       (corrupted, allocation.ALLOCATION_ID))

        with self.assertRaisesRegex(ValueError, "durable additional allocation predecessors changed"):
            ledger.reserve("context-access", "context-access")
        with closing(sqlite3.connect(self.path)) as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0], 0)
            self.assertEqual(db.execute("SELECT COUNT(*) FROM tasks").fetchone()[0], 0)

    def test_reopen_cannot_change_binding_or_grant_and_profile_migration_is_explicit(self):
        binding = allocation.allocation_binding()
        ledger = self.make_ledger(binding=binding)
        with self.assertRaisesRegex(ValueError, "explicit measurement profile binding mismatch"):
            allocation.ContextAllocationLedger(self.path, allocation.ALLOCATION_ID, dict(LIMITS),
                                               binding=binding, profile_id=OBSERVED)
        self.assertEqual(ledger.profile_id, LEGACY)
        self.initialize_v2(ledger)

        changed_binding = json.loads(json.dumps(binding))
        changed_binding["resourceWindow"] = "replacement-window"
        with self.assertRaisesRegex(ValueError, "fixed allocation identity/path/predecessors required"):
            allocation.ContextAllocationLedger(self.path, allocation.ALLOCATION_ID, dict(LIMITS),
                                               binding=changed_binding, profile_id=OBSERVED)

        reopened = self.make_ledger(binding=binding, profile_id=OBSERVED)
        with self.assertRaisesRegex(ValueError, "fixed additional allocation authority cannot change or refill"):
            reopened.bind_dispatch(self.authority().copy() | {"grantId": "different-grant"}, maximum=1)
        with closing(sqlite3.connect(self.path)) as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM context_allocation").fetchone()[0], 1)
            self.assertEqual(db.execute("SELECT COUNT(*) FROM measurement_profiles").fetchone()[0], 2)
            self.assertEqual(db.execute("SELECT stopped FROM trial").fetchone()[0], 0)


if __name__ == "__main__":
    unittest.main()
