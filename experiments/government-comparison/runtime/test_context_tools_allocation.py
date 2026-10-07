"""Focused synthetic contracts for the corrected-tools successor allocation."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import context_tools_allocation as allocation
from ledger import LimitReached
from measurement_profile import LEGACY, OBSERVED, limits_sha, profile_sha
from dispatch import Authority, digest, encoded, mechanical_pin, runtime_pins


LIMITS = {
    "taskWallSeconds": 1200, "trialWallSeconds": 7200,
    "taskActorCalls": 12, "trialActorCalls": 72,
    "taskProviderTurns": 80, "trialProviderTurns": 480,
    "taskProviderTokens": 120000, "trialProviderTokens": 720000,
    "maxParallelActors": 4, "maxTransportRetriesPerCall": 1,
    "maxSemanticRepairRoundsPerTask": 2, "trialActiveHumanSeconds": 600,
    "newPurchases": False,
}


def predecessor_history():
    tokens = (None, None, 10009, 20501)
    return {
        "actorStartsConsumed": 4,
        "historicalStartsRemaining": 0,
        "attempts": [
            {"grant": f"synthetic-start-{index + 1}", "tokens": amount, "finished": True}
            for index, amount in enumerate(tokens)
        ],
        "allHistoryTokens": None,
        "knownTokenSubtotal": 30510,
        "unknownTokenAttempts": 2,
        "ledgerSha256Before": {},
        "ledgerSha256After": {
            "first-ledger": allocation.DECISION["historicalLedgerSha256"][0],
            "second-ledger": allocation.DECISION["historicalLedgerSha256"][1],
            "fourth-ledger": allocation.DECISION["historicalLedgerSha256"][2],
        },
        "historicalLedgersUnchanged": True,
    }


class ContextToolsAllocationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="context-tools-allocation-test-")
        self.root = Path(self.temp.name).resolve()
        self.old_path = allocation.ALLOCATION_PATH
        allocation.ALLOCATION_PATH = self.root / "successor.sqlite"
        self.history_patch = patch.object(allocation, "read_context_history", side_effect=predecessor_history)
        self.history_patch.start()
        self.addCleanup(self.history_patch.stop)

    def tearDown(self):
        allocation.ALLOCATION_PATH = self.old_path
        self.temp.cleanup()

    def binding(self):
        return allocation.allocation_binding()

    def make_ledger(self, *, binding=None, profile_id=LEGACY, path=None, trial_id=None):
        return allocation.ContextToolsAllocationLedger(
            path or allocation.ALLOCATION_PATH,
            allocation.ALLOCATION_ID if trial_id is None else trial_id,
            dict(LIMITS),
            binding=self.binding() if binding is None else binding,
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
            "decisionRef": "decision://synthetic-context-tools-allocation",
        }

    def valid_authority(self):
        binding = self.binding()
        return {
            "contextAllocation": binding,
            "trialId": allocation.ALLOCATION_ID,
            "ledgerPath": str(allocation.ALLOCATION_PATH),
            "maxActorSessions": 1,
            "maxAdditionalActorSessions": 1,
            "cumulativeSessionCeiling": 5,
            "authorizedRequests": [{"requestSha256": "a" * 64}],
            "maxParallelSessions": 1,
            "maxSessionWallSeconds": 180,
            "retrospectiveTokenThreshold": 50000,
            "wrapperAgentTurns": allocation.DECISION["wrapperAgentTurns"],
            "wrapperRetries": allocation.DECISION["wrapperRetries"],
            "children": allocation.DECISION["children"],
            "continuations": allocation.DECISION["continuations"],
            "semanticRepairs": allocation.DECISION["semanticRepairs"],
            "newPurchases": allocation.DECISION["newPurchases"],
        }

    def test_history_binding_preserves_four_starts_unknown_total_and_three_hashes(self):
        binding = self.binding()
        history = binding["predecessors"]
        self.assertEqual([row["tokens"] for row in history["attempts"]], [None, None, 10009, 20501])
        self.assertEqual(history["knownTokenSubtotal"], 30510)
        self.assertIsNone(history["allHistoryTokens"])
        self.assertEqual(history["actorStartsConsumed"], 4)
        self.assertEqual(history["historicalStartsRemaining"], 0)
        self.assertEqual(len(history["ledgerSha256After"]), 3)
        self.assertEqual(binding["cumulativeSessionCeiling"], 5)

    def test_grant_and_protocol_require_exact_fixed_threshold_quota_identity_and_history(self):
        grant = self.valid_authority()
        protocol = {"contextAllocation": copy.deepcopy(grant["contextAllocation"])}
        self.assertEqual(allocation.validate_allocation(grant, protocol), self.binding())

        mutations = (
            ("threshold", lambda g, p: g.update(retrospectiveTokenThreshold=10000)),
            ("quota", lambda g, p: g.update(maxAdditionalActorSessions=2)),
            ("identity", lambda g, p: g.update(trialId="other-allocation")),
            ("history", lambda g, p: g["contextAllocation"]["predecessors"].update(knownTokenSubtotal=30511)),
        )
        for label, mutate in mutations:
            with self.subTest(label=label):
                changed_grant, changed_protocol = copy.deepcopy(grant), copy.deepcopy(protocol)
                mutate(changed_grant, changed_protocol)
                with self.assertRaisesRegex(ValueError, "complete fixed corrected-tools allocation binding required"):
                    allocation.validate_allocation(changed_grant, changed_protocol)

    def test_explicit_profile_adoption_allows_one_session_and_preserves_cumulative_unknown(self):
        binding = self.binding()
        ledger = self.make_ledger(binding=binding)
        self.assertEqual(ledger.snapshot()["measurementProfile"]["profileId"], LEGACY)
        ledger.adopt_profile(OBSERVED, self.approval(ledger))

        attempt = ledger.reserve("synthetic-context-task", "context-access")
        ledger.finish(attempt, "completed", provider_turns=1, provider_tokens=10,
                      receipt={"synthetic": True})
        snapshot = ledger.snapshot()
        self.assertEqual(snapshot["predecessorActorSessions"], 4)
        self.assertEqual(snapshot["actorSessions"], 1)
        self.assertEqual(snapshot["cumulativeActorSessions"], 5)
        self.assertEqual(snapshot["contextAllocation"]["cumulativeSessionCeiling"], 5)
        self.assertIsNone(snapshot["cumulativeProviderTokens"])
        self.assertEqual(snapshot["knownCumulativeTokenSubtotal"], 30520)

        with self.assertRaisesRegex(LimitReached, "allocation exhausted|already booked"):
            ledger.reserve("synthetic-context-task", "context-access")
        self.assertEqual(ledger.snapshot()["actorSessions"], 1)

    def test_invalid_predecessor_or_identity_path_fails_before_creating_database(self):
        good = self.binding()
        changed_history = copy.deepcopy(good)
        changed_history["predecessors"]["attempts"][3]["tokens"] = 20502
        targets = (
            (self.root / "wrong-history.sqlite", changed_history, allocation.ALLOCATION_ID),
            (self.root / "wrong-trial.sqlite", good, "other-allocation"),
            (self.root / "wrong-path.sqlite", good, allocation.ALLOCATION_ID),
        )
        for index, (target, binding, trial_id) in enumerate(targets):
            if index == 2:
                target = self.root / "wrong-path.sqlite"
                with self.assertRaisesRegex(ValueError, "fixed allocation identity/path/predecessors required"):
                    self.make_ledger(binding=binding, path=target)
            else:
                with self.assertRaisesRegex(ValueError, "fixed allocation identity/path/predecessors required"):
                    self.make_ledger(binding=binding, path=target, trial_id=trial_id)
            self.assertFalse(target.exists(), "invalid allocation must fail before SQLite creates a file")

    def test_resume_of_missing_successor_is_rejected_without_creating_it(self):
        with self.assertRaisesRegex(ValueError, "allocation resume requires existing ledger"):
            self.make_ledger(profile_id=None)
        self.assertFalse(allocation.ALLOCATION_PATH.exists())

    def test_dispatcher_selects_fixed_successor_and_does_not_expand_generic_threshold(self):
        results = self.root / "results"
        results.mkdir()
        adoption = self.root / "adoption.json"
        adoption.write_bytes(encoded({"status": "approved", "trialId": allocation.ALLOCATION_ID,
            "ledgerPath": str(allocation.ALLOCATION_PATH), "fromProfileSha256": profile_sha(LEGACY),
            "toProfileSha256": profile_sha(OBSERVED), "limitsSha256": limits_sha(LIMITS),
            "decisionRef": "synthetic-only"}))
        pin_sha = digest(encoded(mechanical_pin()))
        protocol = {"schemaVersion": 1, "status": "frozen", "mode": "mechanical", "commonLimits": LIMITS,
            "measurementProfileId": OBSERVED, "measurementProfileSha256": profile_sha(OBSERVED),
            "runtimeSourceSha256": runtime_pins(), "wrapperPythonSha256": digest(Path(sys.executable).read_bytes()),
            "runnerPinSha256": pin_sha, "contextAllocation": self.binding(),
            "profileAdoption": {"path": str(adoption), "sha256": digest(adoption.read_bytes())}}
        grant = {**self.valid_authority(), "schemaVersion": 1, "status": "approved", "mode": "mechanical",
            "purpose": "s1-mechanics", "profileSha256": limits_sha(LIMITS),
            "measurementProfileId": OBSERVED, "measurementProfileSha256": profile_sha(OBSERVED),
            "runnerPinSha256": pin_sha, "resultDirectory": str(results), "notBefore": 1, "expiresAt": 9999999999}
        protocol_path, grant_path = self.root / "protocol.json", self.root / "grant.json"
        def authority():
            protocol_path.write_bytes(encoded(protocol))
            grant["protocolSha256"] = digest(protocol_path.read_bytes())
            grant_path.write_bytes(encoded(grant))
            return Authority(grant_path, digest(grant_path.read_bytes()), protocol_path, digest(protocol_path.read_bytes()))
        selected = authority()
        self.assertIs(selected.context_ledger_class, allocation.ContextToolsAllocationLedger)
        ledger = selected.ledger()
        self.assertEqual(ledger.snapshot()["cumulativeActorSessions"], 4)
        # Removing the exact allocation cannot grant the 50k threshold generically.
        del grant["contextAllocation"]
        del protocol["contextAllocation"]
        with self.assertRaisesRegex(ValueError, "finite positive bound required"):
            authority()


if __name__ == "__main__":
    unittest.main()
