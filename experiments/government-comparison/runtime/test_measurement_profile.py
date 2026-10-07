"""Focused regression tests for append-only measurement-profile adoption; no provider calls."""
from contextlib import closing
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import time
import unittest

sys.path.insert(0, str(Path(__file__).parent))
from ledger import Ledger, LimitReached
from measurement_profile import LEGACY, OBSERVED, limits_sha, profile, profile_sha

COMMON_LIMITS = {
    "taskWallSeconds": 1200,
    "trialWallSeconds": 7200,
    "taskActorCalls": 12,
    "trialActorCalls": 72,
    "taskProviderTurns": 80,
    "trialProviderTurns": 480,
    "taskProviderTokens": 120000,
    "trialProviderTokens": 720000,
    "maxParallelActors": 4,
    "maxTransportRetriesPerCall": 1,
    "maxSemanticRepairRoundsPerTask": 2,
    "trialActiveHumanSeconds": 600,
    "newPurchases": False,
}


class MeasurementProfileTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="measurement-profile-test-")
        self.root = Path(self.temp.name).resolve()

    def tearDown(self):
        self.temp.cleanup()

    def make_ledger(self, name="trial", *, limits=None, profile_id=LEGACY):
        return Ledger(self.root / f"{name}.sqlite", name, dict(limits or COMMON_LIMITS), profile_id=profile_id)

    @staticmethod
    def approval(ledger, **changes):
        with closing(sqlite3.connect(ledger.path)) as db:
            trial_id = db.execute("SELECT id FROM trial").fetchone()[0]
        value = {
            "status": "approved",
            "trialId": trial_id,
            "ledgerPath": str(Path(ledger.path).resolve()),
            "fromProfileSha256": profile_sha(LEGACY),
            "toProfileSha256": profile_sha(OBSERVED),
            "limitsSha256": limits_sha(ledger.limits),
            "decisionRef": "decision://synthetic-test/approved-v2",
        }
        value.update(changes)
        return value

    def capture_legacy_state(self, ledger):
        with closing(sqlite3.connect(ledger.path)) as db:
            return {
                "trial": db.execute("SELECT id,limits,start,stopped FROM trial").fetchall(),
                "tasks": db.execute("SELECT id,start FROM tasks ORDER BY id").fetchall(),
                "attempts": db.execute("SELECT * FROM attempts ORDER BY id").fetchall(),
                "human": db.execute("SELECT seconds,note FROM human ORDER BY rowid").fetchall(),
                "dispatchAuthority": db.execute("SELECT value FROM dispatch_authority ORDER BY rowid").fetchall(),
                "dispatches": db.execute("SELECT * FROM dispatches ORDER BY id").fetchall(),
            }

    def make_pre_migration_ledger(self, path, trial_id, authority):
        """Create the actual seven-column dispatch schema used before profile migration."""
        attempt_id = "synthetic-old-attempt"
        execution_id = "synthetic-old-terminal-dispatch"
        result = {"status": "readiness_gap", "receipts": [{"source": "synthetic-old-ledger"}]}
        with closing(sqlite3.connect(path)) as db:
            db.executescript("""
                CREATE TABLE trial(id TEXT PRIMARY KEY, limits TEXT, start REAL, stopped INTEGER);
                CREATE TABLE tasks(id TEXT PRIMARY KEY, start REAL);
                CREATE TABLE attempts(id TEXT PRIMARY KEY, task TEXT, purpose TEXT, start REAL,
                    end REAL, status TEXT, turns INTEGER, tokens INTEGER, receipt TEXT);
                CREATE TABLE human(seconds REAL, note TEXT);
                CREATE TABLE dispatch_authority(value TEXT);
                CREATE TABLE dispatches(id TEXT PRIMARY KEY, execution_sha TEXT, attempt TEXT UNIQUE,
                    phase TEXT, request BLOB, command TEXT, result TEXT);
            """)
            db.execute("INSERT INTO trial VALUES(?,?,?,0)",
                       (trial_id, json.dumps(COMMON_LIMITS, sort_keys=True), time.time() - 30))
            db.execute("INSERT INTO tasks VALUES(?,?)", ("old-task", time.time() - 20))
            db.execute("INSERT INTO attempts VALUES(?,?,?,?,?,?,?,?,?)",
                       (attempt_id, "old-task", "executor", time.time() - 20, time.time() - 10,
                        "completed", 4, 37, json.dumps({"synthetic": "preserve-me"}, sort_keys=True)))
            db.execute("INSERT INTO human VALUES(?,?)", (13, "pre-migration synthetic note"))
            db.execute("INSERT INTO dispatch_authority VALUES(?)",
                       (json.dumps(authority, sort_keys=True),))
            db.execute("INSERT INTO dispatches VALUES(?,?,?,?,?,?,?)",
                       (execution_id, "c" * 64, attempt_id, "finished", b'{"synthetic":true}',
                        json.dumps(["synthetic-adapter", "--never-run"]), json.dumps(result, sort_keys=True)))
            db.commit()
            self.assertEqual([row[1] for row in db.execute("PRAGMA table_info(dispatches)")],
                             ["id", "execution_sha", "attempt", "phase", "request", "command", "result"])
            self.assertFalse(db.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='measurement_profiles'").fetchone())
            self.assertFalse(db.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='dispatch_authority_history'").fetchone())
        return execution_id, attempt_id, result

    def capture_original_rows(self, path):
        with closing(sqlite3.connect(path)) as db:
            return {
                "trial": db.execute("SELECT id,limits,start,stopped FROM trial").fetchall(),
                "tasks": db.execute("SELECT id,start FROM tasks ORDER BY id").fetchall(),
                "attempts": db.execute("SELECT * FROM attempts ORDER BY id").fetchall(),
                "human": db.execute("SELECT seconds,note FROM human ORDER BY rowid").fetchall(),
                "dispatchAuthority": db.execute("SELECT value FROM dispatch_authority ORDER BY rowid").fetchall(),
                "dispatches": db.execute("SELECT id,execution_sha,attempt,phase,request,command,result FROM dispatches ORDER BY id").fetchall(),
            }

    def test_pre_migration_seven_column_schema_adopts_rebinds_and_recovers_without_rewriting_rows(self):
        path = self.root / "pre-migration.sqlite"
        old_authority = {"grantId": "synthetic-old-grant", "protocolSha256": "a" * 64,
                         "profileSha256": profile_sha(LEGACY), "limitsSha256": limits_sha(COMMON_LIMITS)}
        execution_id, attempt_id, terminal_result = self.make_pre_migration_ledger(path, "pre-migration", old_authority)
        original = self.capture_original_rows(path)

        ledger = Ledger(path, "pre-migration", COMMON_LIMITS, profile_id=LEGACY)
        ledger.adopt_profile(OBSERVED, self.approval(ledger))
        self.assertEqual(self.capture_original_rows(path), original,
                         "profile adoption must preserve the original seven-column dispatch rows verbatim")

        old_key = limits_sha(old_authority)
        ledger.bind_dispatch(old_authority, maximum=1)
        successor = {"grantId": "synthetic-successor-grant", "protocolSha256": "b" * 64,
                     "profileSha256": profile_sha(OBSERVED), "limitsSha256": limits_sha(COMMON_LIMITS)}
        ledger.bind_dispatch(successor, maximum=2, transition={
            "priorAuthoritySha256": old_key,
            "priorActorSessions": 1,
            "additionalActorSessions": 1,
        })
        self.assertEqual(self.capture_original_rows(path), original,
                         "history migration and successor binding must not rewrite source rows")
        with closing(sqlite3.connect(path)) as db:
            columns = [row[1] for row in db.execute("PRAGMA table_info(dispatches)")]
            self.assertEqual(columns, ["id", "execution_sha", "attempt", "phase", "request", "command", "result",
                                       "measurement_profile", "authority_sha"])
            history = db.execute("SELECT sequence,authority_sha,profile_sha,ceiling FROM dispatch_authority_history ORDER BY sequence").fetchall()
            self.assertEqual(history, [(1, old_key, profile_sha(LEGACY), 1),
                                       (2, limits_sha(successor), profile_sha(OBSERVED), 2)])

        # A recovery-only reopen may read the old terminal result and cannot book new work.
        recovered = Ledger(path, "pre-migration", COMMON_LIMITS, profile_id=None)
        record = recovered.dispatch_record(execution_id)
        self.assertEqual(record["phase"], "finished")
        self.assertEqual(json.loads(record["result"]), terminal_result)
        self.assertFalse(recovered.claim_dispatch(execution_id, "launched"))
        self.assertEqual(recovered.complete_dispatch(execution_id, {"status": "ignored"}, None, None), terminal_result)
        with self.assertRaisesRegex(ValueError, "recovery-only ledger cannot reserve"):
            recovered.reserve("new-task", "executor")
        self.assertEqual(original["attempts"][0][0], attempt_id)
        self.assertEqual(self.capture_original_rows(path), original)

    def test_adoption_preserves_finished_stopped_human_dispatch_and_common_limits(self):
        ledger = self.make_ledger()
        first = ledger.reserve("setup", "initial-model")
        ledger.finish(first, "failed-before-provider", provider_turns=0, provider_tokens=0,
                      receipt={"synthetic": "failed-start"})
        second = ledger.reserve("task-1", "executor")
        ledger.finish(second, "completed", provider_turns=3, provider_tokens=19,
                      receipt={"synthetic": "finished"})
        authority = {"grantId": "synthetic-grant", "protocolSha256": "a" * 64,
                     "profileSha256": profile_sha(LEGACY), "limitsSha256": limits_sha(COMMON_LIMITS)}
        ledger.bind_dispatch(authority)
        execution_id = "synthetic-terminal-dispatch"
        attempt = ledger.reserve_dispatch(execution_id, "b" * 64, b'{"synthetic":true}',
                                          ["fake-adapter", "--never-run"], "task-2", "review", 5)
        self.assertTrue(ledger.claim_dispatch(execution_id, "launched"))
        final_result = {"status": "readiness_gap", "receipts": [{"kind": "synthetic-only"}]}
        self.assertEqual(ledger.complete_dispatch(execution_id, final_result, 2, 29), final_result)
        ledger.human(11, "synthetic oversight entry")
        ledger.stop()
        before = self.capture_legacy_state(ledger)
        self.assertEqual(before["trial"][0][1], json.dumps(COMMON_LIMITS, sort_keys=True))
        self.assertEqual(before["trial"][0][3], 1)
        self.assertEqual(len(before["attempts"]), 3)
        self.assertEqual(len(before["human"]), 1)
        approval = self.approval(ledger)

        ledger.adopt_profile(OBSERVED, approval)

        self.assertEqual(ledger.profile_id, OBSERVED, "the adopting legacy object updates only after commit")
        after = self.capture_legacy_state(ledger)
        self.assertEqual(after, before, "adoption must not rewrite existing rows, limits, counters, stop, or authority")
        snapshot = ledger.snapshot()
        self.assertEqual(snapshot["measurementProfile"]["profileId"], OBSERVED)
        self.assertEqual([row["profile_id"] for row in snapshot["profileHistory"]], [LEGACY, OBSERVED])
        self.assertEqual(snapshot["profileHistory"][1]["prior_sha"], profile_sha(LEGACY))
        recorded_approval = json.loads(snapshot["profileHistory"][1]["approval"])
        for key, value in approval.items():
            self.assertEqual(recorded_approval[key], value)
        self.assertEqual(recorded_approval["approvalSha256"], limits_sha(approval))
        self.assertTrue(recorded_approval["migrationSourceSha256"])

        # A completed dispatch remains terminal after recovery: only record reads/idempotent result reads succeed.
        reopened = Ledger(ledger.path, "trial", COMMON_LIMITS, profile_id=OBSERVED)
        self.assertEqual(reopened.dispatch_record(execution_id)["phase"], "finished")
        self.assertFalse(reopened.claim_dispatch(execution_id, "launched"))
        self.assertEqual(reopened.complete_dispatch(execution_id, final_result, None, None), final_result)
        with self.assertRaisesRegex(LimitReached, "already booked; use resume"):
            reopened.reserve_dispatch(execution_id, "b" * 64, b'{"synthetic":true}',
                                      ["fake-adapter", "--never-run"], "task-2", "review", 5)
        # Recovery with None explicitly reads the already-adopted profile; no work is launched.
        recovered = Ledger(ledger.path, "trial", COMMON_LIMITS, profile_id=None)
        self.assertEqual(recovered.snapshot()["measurementProfile"]["profileId"], OBSERVED)
        with self.assertRaisesRegex(ValueError, "explicit measurement profile binding mismatch"):
            Ledger(ledger.path, "trial", COMMON_LIMITS, profile_id=LEGACY)
        self.assertEqual(attempt, before["dispatches"][0][2])

    def test_unknown_provider_requests_are_allowed_only_after_explicit_v2_adoption(self):
        limits = {**COMMON_LIMITS, "taskProviderTurns": 1, "trialProviderTurns": 1,
                  "taskProviderTokens": 100, "trialProviderTokens": 100}
        legacy = self.make_ledger("legacy-unknown-request", limits=limits)
        old_attempt = legacy.reserve("task", "executor")
        legacy.finish(old_attempt, "completed", provider_turns=None, provider_tokens=2)
        with self.assertRaisesRegex(LimitReached, "unknown provider usage"):
            legacy.reserve("task", "review")

        observed = self.make_ledger("observed-unknown-request", limits=limits)
        old_attempt = observed.reserve("task", "executor")
        observed.finish(old_attempt, "completed", provider_turns=None, provider_tokens=2)
        observed.adopt_profile(OBSERVED, self.approval(observed))
        next_attempt = observed.reserve("task", "review")
        self.assertTrue(next_attempt)
        self.assertEqual(observed.snapshot()["measurementProfile"]["profileId"], OBSERVED)

    def test_known_provider_request_count_is_not_a_v2_cap_but_token_cap_remains(self):
        limits = {**COMMON_LIMITS, "taskProviderTurns": 1, "trialProviderTurns": 1,
                  "taskProviderTokens": 10, "trialProviderTokens": 10}
        ledger = self.make_ledger("known-request-over-cap", limits=limits)
        attempt = ledger.reserve("task", "executor")
        ledger.finish(attempt, "completed", provider_turns=1, provider_tokens=2)
        ledger.adopt_profile(OBSERVED, self.approval(ledger))
        self.assertTrue(ledger.reserve("task", "review"), "v2 reports but does not enforce ProviderTurns as a cap")

        token_limited = self.make_ledger("token-cap", limits=limits)
        attempt = token_limited.reserve("task", "executor")
        token_limited.finish(attempt, "completed", provider_turns=None, provider_tokens=10)
        token_limited.adopt_profile(OBSERVED, self.approval(token_limited))
        with self.assertRaisesRegex(LimitReached, "ProviderTokens"):
            token_limited.reserve("task", "review")

    def test_unknown_finished_tokens_block_in_both_profiles(self):
        for name, profile_id in (("legacy-unknown-token", LEGACY), ("observed-unknown-token", OBSERVED)):
            with self.subTest(profile=profile_id):
                ledger = self.make_ledger(name)
                attempt = ledger.reserve("task", "executor")
                ledger.finish(attempt, "failed", provider_turns=3, provider_tokens=None)
                if profile_id == OBSERVED:
                    ledger.adopt_profile(OBSERVED, self.approval(ledger))
                with self.assertRaisesRegex(LimitReached, "tokens unknown"):
                    ledger.reserve("task", "review")
                self.assertIsNone(ledger.snapshot()["providerTokens"])

    def test_active_attempt_blocks_migration_and_rolls_back_without_history_change(self):
        ledger = self.make_ledger()
        active = ledger.reserve("task", "executor")
        before = ledger.snapshot()
        with self.assertRaisesRegex(ValueError, "active/ambiguous attempts"):
            ledger.adopt_profile(OBSERVED, self.approval(ledger))
        after = ledger.snapshot()
        self.assertEqual(after["profileHistory"], before["profileHistory"])
        self.assertEqual(after["attempts"], before["attempts"])
        self.assertIsNone(after["attempts"][0]["end"])
        self.assertTrue(active)

    def test_failed_atomic_insert_rolls_back_and_keeps_legacy_profile(self):
        ledger = self.make_ledger()
        attempt = ledger.reserve("task", "executor")
        ledger.finish(attempt, "completed", provider_turns=1, provider_tokens=3)
        with ledger.transaction() as db:
            db.execute("CREATE TRIGGER reject_v2 BEFORE INSERT ON measurement_profiles WHEN NEW.sequence=2 BEGIN SELECT RAISE(ABORT, 'synthetic migration fault'); END")
        with self.assertRaises(sqlite3.IntegrityError):
            ledger.adopt_profile(OBSERVED, self.approval(ledger))
        self.assertEqual(ledger.profile_id, LEGACY)
        snapshot = ledger.snapshot()
        self.assertEqual(snapshot["measurementProfile"]["profileId"], LEGACY)
        self.assertEqual([row["profile_id"] for row in snapshot["profileHistory"]], [LEGACY])
        self.assertEqual(snapshot["attempts"][0]["tokens"], 3)

    def test_bad_approval_identity_path_limits_and_profile_digests_are_rejected(self):
        mutations = (
            {"status": "pending"},
            {"trialId": "another-trial"},
            {"ledgerPath": "C:/other/ledger.sqlite"},
            {"fromProfileSha256": "0" * 64},
            {"toProfileSha256": "1" * 64},
            {"limitsSha256": "2" * 64},
            {"decisionRef": ""},
        )
        for index, mutation in enumerate(mutations):
            with self.subTest(mutation=mutation):
                ledger = self.make_ledger(f"bad-approval-{index}")
                approval = self.approval(ledger, **mutation)
                before = ledger.snapshot()["profileHistory"]
                with self.assertRaisesRegex(ValueError, "approval identity/digests mismatch"):
                    ledger.adopt_profile(OBSERVED, approval)
                self.assertEqual(ledger.snapshot()["profileHistory"], before)
                self.assertEqual(ledger.snapshot()["measurementProfile"]["profileId"], LEGACY)

    def test_duplicate_adoption_wrong_target_constructor_identity_and_limit_refill_rejected(self):
        ledger = self.make_ledger("duplicate")
        approval = self.approval(ledger)
        ledger.adopt_profile(OBSERVED, approval)
        history = ledger.snapshot()["profileHistory"]
        with self.assertRaisesRegex(ValueError, "only explicit successive"):
            ledger.adopt_profile(OBSERVED, approval)
        self.assertEqual(ledger.snapshot()["profileHistory"], history)

        with self.assertRaisesRegex(ValueError, "explicit measurement profile binding mismatch"):
            Ledger(ledger.path, "duplicate", COMMON_LIMITS, profile_id=LEGACY)
        with self.assertRaisesRegex(ValueError, "explicit measurement profile binding mismatch"):
            Ledger(self.root / "unadopted-explicit-v2.sqlite", "unadopted-explicit-v2",
                   COMMON_LIMITS, profile_id=OBSERVED)
        with self.assertRaisesRegex(ValueError, "identity/profile cannot change or refill"):
            Ledger(ledger.path, "wrong-trial", COMMON_LIMITS, profile_id=None)
        changed_limits = {**COMMON_LIMITS, "taskProviderTokens": COMMON_LIMITS["taskProviderTokens"] + 1}
        with self.assertRaisesRegex(ValueError, "identity/profile cannot change or refill"):
            Ledger(ledger.path, "duplicate", changed_limits, profile_id=None)
        with self.assertRaisesRegex(ValueError, "explicit successive v1 to v2"):
            other = self.make_ledger("wrong-target")
            other.adopt_profile(LEGACY, self.approval(other))

    def test_all_arms_share_identical_observed_profile_digest_and_limits(self):
        profiles = []
        for arm in ("conventional", "classic", "government"):
            ledger = self.make_ledger(f"same-profile-{arm}")
            ledger.adopt_profile(OBSERVED, self.approval(ledger))
            snapshot = ledger.snapshot()
            profiles.append((snapshot["measurementProfile"]["profileId"],
                             snapshot["profileHistory"][-1]["profile_sha"],
                             snapshot["profileHistory"][-1]["approval"]))
            self.assertEqual(snapshot["measurementProfile"]["arms"], ["conventional", "classic", "government"])
            self.assertEqual(ledger.limits, COMMON_LIMITS)
        self.assertEqual({(item[0], item[1]) for item in profiles}, {(OBSERVED, profile_sha(OBSERVED))})
        self.assertEqual([profile(OBSERVED)["profileId"] for _ in profiles], [OBSERVED] * 3)


if __name__ == "__main__":
    unittest.main()
