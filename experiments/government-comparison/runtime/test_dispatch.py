"""Focused deterministic mechanics tests; no native Actor or inference invocation."""
import concurrent.futures
import importlib.util
import json
import os
from pathlib import Path
import sys
import subprocess
import tempfile
import threading
import time
import unittest
from unittest.mock import patch

from adapter import handle
import dispatch as dispatch_module
from dispatch import Authority, LIVE_GAPS, base_result, command, digest, encoded, execution_sha, mechanical_pin, runtime_pins, validate_listing, tool_observations
from ledger import LimitReached
from process import bounded
from measurement_profile import LEGACY, OBSERVED, profile_sha, limits_sha

ROOT = Path(tempfile.mkdtemp(prefix="markitect-s1-mechanics-", dir=os.environ.get("S1_MECHANICS_PARENT")))
COMMON = json.loads(Path(__file__).parents[1].joinpath("public/resource-proposal.json").read_bytes())["commonLimits"]


class DispatchTests(unittest.TestCase):
    def setUp(self):
        self.root = ROOT / self._testMethodName
        self.root.mkdir()
        self.actor = self.root / "actor"
        self.actor.mkdir()
        self.results = self.root / "results"
        self.results.mkdir()
        self.fixture = self.root / "deterministic.py"
        self.fixture.write_bytes(Path(__file__).with_name("mechanical_actor.py").read_bytes())
        self.card = self.root / "public-card.txt"
        self.card.write_text("synthetic-public-card\n", encoding="utf-8")
        self.requests = []

    def request(self, name="one", case="success", purpose="task", arm="conventional", wall=5):
        card = {"path": str(self.card), "sha256": digest(self.card.read_bytes())}
        r = {"schemaVersion": 1, "mode": "mechanical", "operation": "run_task", "trialId": self._testMethodName,
             "dispatchId": name, "arm": arm, "condition": "greenfield", "actorRepository": str(self.actor),
             "evidenceDirectory": str(self.root / (name + "-evidence")), "limits": COMMON,
             "wallSeconds": wall, "task": {"id": "public-task", "card": card}, "purpose": purpose,
             "releasedInputs": [card], "prompt": card, "mechanicalFixture": str(self.fixture), "mechanicalCase": case}
        path = self.root / (name + ".json")
        path.write_bytes(encoded(r))
        self.requests.append((r, path))
        return r, path

    def authorize(self, maximum=8, threshold=10000, profile_id=LEGACY, label="", successor=None, allocation=False):
        trial_id = self._testMethodName
        if allocation:
            from context_allocation import ALLOCATION_ID, allocation_binding, DECISION
            maximum, profile_id, trial_id = 1, OBSERVED, ALLOCATION_ID
            owned = self.actor / "actor-own.txt"
            owned.write_text("synthetic-owned-context\n")
            reads = [str(self.card), str(owned)]
            for r, path in self.requests:
                r.update(trialId=trial_id, purpose="context-access", smokeKind="effective-context-only",
                         contextAllocationDecisionId=ALLOCATION_ID, allowedReadPaths=reads,
                         actorOwnedInputs=[{"path": str(owned), "sha256": digest(owned.read_bytes())}])
        pin = digest(encoded(mechanical_pin()))
        p = {"status": "frozen", "mode": "mechanical", "commonLimits": COMMON,
             "runnerPinSha256": pin, "runtimeSourceSha256": runtime_pins(),
             "wrapperPythonSha256": digest(Path(sys.executable).read_bytes())}
        if profile_id == OBSERVED:
            for r, path in self.requests:
                r.update(measurementProfileId=OBSERVED, measurementProfileSha256=profile_sha(OBSERVED))
                path.write_bytes(encoded(r))
            adoption = {"status": "approved", "trialId": trial_id,
                        "ledgerPath": str((self.root / "trial.sqlite").resolve()), "fromProfileSha256": profile_sha(LEGACY),
                        "toProfileSha256": profile_sha(OBSERVED), "limitsSha256": limits_sha(COMMON),
                        "decisionRef": "synthetic-profile-adoption-test"}
            adoption_path = self.root / (label + "adoption.json")
            adoption_path.write_bytes(encoded(adoption))
            p.update(measurementProfileId=OBSERVED, measurementProfileSha256=profile_sha(OBSERVED),
                     profileAdoption={"path": str(adoption_path), "sha256": digest(encoded(adoption))})
        if allocation:
            p.update(contextAllocation=allocation_binding(), allowedReadPaths=reads, contextOnly=True)
        self.protocol_path = self.root / (label + "protocol.json")
        self.protocol_path.write_bytes(encoded(p))
        g = {"schemaVersion": 1, "status": "approved", "purpose": "s1-mechanics", "mode": "mechanical",
             "trialId": trial_id, "notBefore": time.time() - 1, "expiresAt": time.time() + 600,
             "protocolSha256": digest(encoded(p)), "profileSha256": digest(encoded(COMMON)), "runnerPinSha256": pin,
             "ledgerPath": str((self.root / "trial.sqlite").resolve()), "resultDirectory": str(self.results), "maxActorSessions": maximum,
             "maxSessionWallSeconds": 10, "retrospectiveTokenThreshold": threshold,
             "authorizedRequests": [{"dispatchId": r["dispatchId"], "executionSha256": execution_sha(r),
                                      "initialRequestSha256": digest(path.read_bytes())} for r, path in self.requests]}
        if profile_id == OBSERVED:
            g.update(measurementProfileId=OBSERVED, measurementProfileSha256=profile_sha(OBSERVED))
        if successor:
            g.update(successor)
        if allocation:
            g.update({key: DECISION[key] for key in ("maxParallelSessions", "wrapperAgentTurns", "wrapperRetries",
                     "children", "continuations", "semanticRepairs", "newPurchases", "cumulativeSessionCeiling")})
            g.update(contextAllocation=allocation_binding(), maxAdditionalActorSessions=1,
                     maxSessionWallSeconds=180, allowedReadPaths=reads)
        self.grant_path = self.root / (label + "grant.json")
        self.grant_path.write_bytes(encoded(g))
        self.authority = Authority(self.grant_path, digest(encoded(g)), self.protocol_path, digest(encoded(p)))
        self.ledger = self.authority.ledger()

    def call(self, path, result="result.json"):
        return handle(path, self.results / result, authority=self.authority)

    def operation(self, request, operation):
        changed = dict(request, operation=operation)
        path = self.root / (request["dispatchId"] + "-" + operation + ".json")
        path.write_bytes(encoded(changed))
        return path

    def launches(self):
        p = self.actor / "launches.jsonl"
        return len(p.read_text().splitlines()) if p.exists() else 0

    def reserve(self, r, path):
        return self.ledger.reserve_dispatch(r["dispatchId"], execution_sha(r), path.read_bytes(),
                                          command(r, self.authority), r["task"]["id"], r["purpose"], 8)

    def test_completion_and_result_recovery(self):
        r, path = self.request()
        self.authorize()
        result = self.call(path)
        self.assertEqual(result["status"], "completed")
        self.assertEqual(result["usage"]["reportedInputPlusOutputTokens"], 5)
        self.assertEqual(self.ledger.snapshot()["providerTurns"], 1)
        (self.results / "result.json").unlink()  # Derived result copy may be lost after DB completion.
        resume = self.operation(r, "resume")
        recovered = self.call(resume)
        self.assertEqual(recovered["initialRequestSha256"], digest(path.read_bytes()))
        self.assertEqual(recovered["requestSha256"], digest(resume.read_bytes()))
        self.assertEqual(self.launches(), 1)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 1)

    def test_missing_usage_blocks_next_role(self):
        _, first = self.request(case="no_usage", purpose="setup")
        _, second = self.request("two", purpose="review")
        self.authorize()
        self.assertEqual(self.call(first)["status"], "incomplete")
        self.assertEqual(self.call(second)["status"], "blocked")
        self.assertIsNone(self.ledger.snapshot()["providerTokens"])
        self.assertEqual(self.launches(), 1)

    def test_v2_unknown_requests_known_tokens_complete_and_admit_next(self):
        _, one = self.request(case="unknown_requests", purpose="setup")
        _, two = self.request("two", case="unknown_requests", purpose="review")
        self.authorize(profile_id=OBSERVED)
        result = self.call(one)
        self.assertEqual(result["status"], "completed")
        self.assertIsNone(result["usage"]["providerRequests"])
        self.assertEqual(result["usage"]["reportedInputPlusOutputTokens"], 5)
        self.assertEqual(self.call(two)["status"], "completed")
        self.assertEqual(self.ledger.snapshot()["providerTokens"], 10)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 2)
        self.assertIsNone(self.ledger.snapshot()["providerTurns"])

    def test_additional_allocation_dispatch_duplicate_resume_and_new_grant_no_refill(self):
        import context_allocation
        with patch.object(context_allocation, "ALLOCATION_PATH", (self.root / "trial.sqlite").resolve()):
            r, path = self.request(case="unknown_requests")
            self.authorize(allocation=True)
            result = self.call(path)
            self.assertEqual(result["status"], "completed")
            self.assertEqual(result["cumulativeAccounting"]["knownTokenSubtotal"], 10014)
            self.assertIsNone(result["cumulativeAccounting"]["allHistoryTokens"])
            self.assertEqual(self.call(path)["status"], "blocked")
            resumed = self.call(self.operation(r, "resume"))
            self.assertEqual(resumed["attemptId"], result["attemptId"])
            self.assertEqual(self.launches(), 1)
            self.assertEqual(self.ledger.snapshot()["cumulativeActorSessions"], 4)
            changed = json.loads(self.grant_path.read_bytes())
            changed["expiresAt"] += 1
            self.grant_path.write_bytes(encoded(changed))
            replacement = Authority(self.grant_path, digest(encoded(changed)), self.protocol_path, self.authority.protocol_sha)
            with self.assertRaisesRegex(ValueError, "authority cannot change or refill"):
                replacement.ledger()
            self.assertEqual(self.launches(), 1)

    def test_additional_allocation_unknown_new_usage_remains_incomplete_and_blocks(self):
        import context_allocation
        with patch.object(context_allocation, "ALLOCATION_PATH", (self.root / "trial.sqlite").resolve()):
            _, path = self.request(case="no_usage")
            self.authorize(allocation=True)
            result = self.call(path)
            self.assertEqual(result["status"], "incomplete")
            self.assertIsNone(result["cumulativeAccounting"]["newWindowTokens"])
            with self.assertRaisesRegex(LimitReached, "new context usage tokens unknown"):
                self.ledger.reserve("next", "context-access")
            self.assertEqual(self.launches(), 1)

    def test_additional_allocation_resume_empty_or_unbooked_file_is_read_only(self):
        import context_allocation
        path = (self.root / "trial.sqlite").resolve()
        with patch.object(context_allocation, "ALLOCATION_PATH", path):
            path.write_bytes(b"")
            with self.assertRaisesRegex(ValueError, "existing exact dispatch"):
                context_allocation.ContextAllocationLedger(path, context_allocation.ALLOCATION_ID, COMMON,
                    binding=context_allocation.allocation_binding(), profile_id=None, resume_dispatch_id="one")
            self.assertEqual(path.read_bytes(), b"")
            r, request_path = self.request()
            self.authorize(allocation=True)
            before = digest(path.read_bytes())
            with self.assertRaisesRegex(ValueError, "existing exact dispatch"):
                self.call(self.operation(r, "resume"))
            self.assertEqual(digest(path.read_bytes()), before)
            self.assertEqual(self.launches(), 0)

    def test_v2_unknown_tokens_still_block_and_profile_mismatch_rejected(self):
        _, one = self.request(case="no_usage")
        _, two = self.request("two")
        self.authorize(profile_id=OBSERVED)
        self.assertEqual(self.call(one)["status"], "incomplete")
        self.assertEqual(self.call(two)["status"], "blocked")
        value = json.loads(two.read_bytes())
        value["measurementProfileId"] = "government-special-profile"
        with self.assertRaises(ValueError):
            self.authority.validate(encoded(value))
        self.assertEqual(self.launches(), 1)

    def test_terminal_resume_preserves_original_profile_after_adoption(self):
        r, path = self.request()
        self.authorize()
        initial = self.call(path)
        approval = {"status": "approved", "trialId": self._testMethodName,
                    "ledgerPath": self.ledger.path, "fromProfileSha256": profile_sha(LEGACY),
                    "toProfileSha256": profile_sha(OBSERVED), "limitsSha256": limits_sha(COMMON),
                    "decisionRef": "synthetic-no-reset-resume-test"}
        self.ledger.adopt_profile(OBSERVED, approval)
        resumed = self.call(self.operation(r, "resume"))
        self.assertEqual(resumed["measurementProfileId"], LEGACY)
        self.assertEqual(initial["attemptId"], resumed["attemptId"])
        self.assertEqual(self.launches(), 1)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 1)
        with self.assertRaisesRegex(ValueError, "profile binding mismatch"):
            self.authority.ledger()

    def test_successor_grant_preserves_history_and_both_resumes(self):
        old_request, old_path = self.request()
        self.authorize(maximum=1)
        old_authority = self.authority
        first = self.call(old_path)
        old_key = self.ledger.authority_key
        with self.ledger.transaction() as db:
            original = db.execute("SELECT * FROM dispatch_authority").fetchall()
            old_attempt = db.execute("SELECT * FROM attempts").fetchall()
            old_dispatch = db.execute("SELECT * FROM dispatches").fetchall()
            original_trial = db.execute("SELECT * FROM trial").fetchall()
        new_request, new_path = self.request("two", case="unknown_requests", purpose="review")
        self.requests = [(new_request, new_path)]
        self.authorize(maximum=2, profile_id=OBSERVED, label="successor-", successor={
            "priorAuthoritySha256": old_key, "priorActorSessions": 1, "additionalActorSessions": 1})
        self.assertEqual(self.call(new_path)["status"], "completed")
        self.assertEqual(self.call(self.operation(new_request, "resume"))["measurementProfileId"], OBSERVED)
        with self.ledger.transaction() as db:
            self.assertEqual(db.execute("SELECT * FROM dispatch_authority").fetchall(), original)
            self.assertEqual(db.execute("SELECT * FROM attempts WHERE id=?", (first["attemptId"],)).fetchall(), old_attempt)
            self.assertEqual(db.execute("SELECT * FROM dispatches WHERE id='one'").fetchall(), old_dispatch)
            self.assertEqual(db.execute("SELECT * FROM trial").fetchall(), original_trial)
            self.assertEqual(db.execute("SELECT ceiling FROM dispatch_authority_history ORDER BY sequence").fetchall(), [(1,), (2,)])
        recovered = handle(self.operation(old_request, "resume"), self.results / "old-resume.json", authority=old_authority)
        self.assertEqual(recovered["attemptId"], first["attemptId"])
        self.assertEqual(recovered["measurementProfileId"], LEGACY)
        self.assertEqual(self.launches(), 2)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 2)
        with self.assertRaisesRegex(ValueError, "recovery-only"):
            old_authority.ledger("resume").reserve("other", "task")
        with self.assertRaisesRegex(ValueError, "ceiling"):
            self.ledger.reserve_dispatch("extra", "x", b"{}", [], "public-task", "child", 3)

    def test_new_grant_requires_explicit_successor_allocation(self):
        _, path = self.request()
        self.authorize(maximum=1)
        self.call(path)
        original_ledger = self.ledger
        before = original_ledger.snapshot()
        _, second = self.request("two")
        self.requests = [self.requests[-1]]
        with self.assertRaisesRegex(ValueError, "explicit cumulative successor"):
            self.authorize(maximum=2, profile_id=OBSERVED, label="unallocated-")
        self.assertEqual(self.launches(), 1)
        self.assertEqual(original_ledger.snapshot(), before)
        with original_ledger.transaction() as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM dispatch_authority_history").fetchone()[0], 1)

    def test_shared_stop_aborts_active_process(self):
        r, path = self.request(case="sleep")
        self.authorize()
        with concurrent.futures.ThreadPoolExecutor(1) as pool:
            running = pool.submit(self.call, path, "running.json")
            end = time.monotonic() + 5
            while not self.launches() and time.monotonic() < end:
                time.sleep(.01)
            self.assertEqual(self.launches(), 1)
            self.assertEqual(self.call(self.operation(r, "stop"), "stop.json")["status"], "stopped")
            result = running.result(timeout=8)
        self.assertEqual(result["status"], "stopped")
        self.assertEqual(result["process"]["stopReason"], "trial_stopped")
        self.assertEqual(self.ledger.snapshot()["remainingActiveCalls"], 0)

    def test_deadline_kills_descendants(self):
        _, path = self.request(case="child", wall=1)
        self.authorize()
        result = self.call(path)
        self.assertEqual(self.launches(), 1)
        self.assertEqual(result["process"]["stopReason"], "wall_deadline")
        time.sleep(2.2)
        self.assertFalse((self.actor / "escaped-child").exists())

    def test_concurrent_reservations_share_parallel_limit(self):
        requests = [self.request(str(n)) for n in range(8)]
        self.authorize()
        barrier = threading.Barrier(8)
        def reserve(item):
            barrier.wait()
            try:
                return self.reserve(*item)
            except LimitReached:
                return None
        with concurrent.futures.ThreadPoolExecutor(8) as pool:
            reservations = list(pool.map(reserve, requests))
        self.assertEqual(sum(x is not None for x in reservations), COMMON["maxParallelActors"])
        self.assertEqual(self.launches(), 0)

    def test_same_request_concurrent_dispatch_launches_once(self):
        _, path = self.request()
        self.authorize()
        with concurrent.futures.ThreadPoolExecutor(2) as pool:
            results = list(pool.map(lambda n: self.call(path, str(n) + ".json"), range(2)))
        self.assertEqual(sorted(r["status"] for r in results), ["blocked", "completed"])
        self.assertEqual(self.launches(), 1)

    def test_interrupted_booking_is_consumed_without_relaunch(self):
        r, path = self.request()
        self.authorize()
        self.reserve(r, path)
        result = self.call(self.operation(r, "resume"))
        self.assertEqual(result["status"], "incomplete")
        self.assertFalse(self.ledger.claim_dispatch(r["dispatchId"], "launching"))
        self.assertEqual(self.launches(), 0)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 1)
        self.assertEqual(self.call(path)["status"], "blocked")

    def test_launch_without_receipt_stays_active(self):
        r, path = self.request()
        self.authorize()
        self.reserve(r, path)
        self.ledger.claim_dispatch(r["dispatchId"], "launching")
        self.assertEqual(self.call(self.operation(r, "resume"))["status"], "blocked")
        self.assertEqual(self.ledger.snapshot()["remainingActiveCalls"], 1)
        self.assertEqual(self.launches(), 0)

    def test_interrupted_recovery_can_finish_without_refill(self):
        r, path = self.request()
        self.authorize()
        self.reserve(r, path)
        self.ledger.claim_dispatch(r["dispatchId"], "recovering")  # Previous two-step recovery crash fixture.
        self.assertEqual(self.call(self.operation(r, "resume"))["status"], "incomplete")
        self.assertEqual(self.ledger.snapshot()["remainingActiveCalls"], 0)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 1)
        self.assertEqual(self.launches(), 0)

    def test_recover_terminal_process_receipt_without_launch(self):
        r, path = self.request()
        self.authorize()
        self.reserve(r, path)
        self.ledger.claim_dispatch(r["dispatchId"], "launching")
        directory = Path(r["evidenceDirectory"])
        directory.mkdir()
        (directory / "request.json").write_bytes(path.read_bytes())
        (directory / "released-inputs").mkdir()
        (directory / "released-inputs/0").write_bytes(self.card.read_bytes())
        bounded(command(r, self.authority), str(self.actor), directory / "process", 5)
        resume = self.operation(r, "resume")
        self.assertEqual(self.call(resume)["status"], "completed")
        self.assertEqual(self.call(resume)["status"], "completed")
        self.assertEqual(self.launches(), 1)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 1)

    def test_retrospective_threshold_is_not_hard_cap(self):
        _, path = self.request()
        self.authorize(threshold=4)
        result = self.call(path)
        self.assertEqual(result["status"], "stopped")
        self.assertEqual(result["usage"]["reportedInputPlusOutputTokens"], 5)
        self.assertEqual(result["accountingStopReason"], "retrospective_session_token_threshold")

    def test_native_missing_products_do_not_reserve(self):
        _, path = self.request(arm="government")
        self.authorize()
        result = self.call(path)
        self.assertEqual(result["status"], "readiness_gap")
        self.assertTrue(any("accepted for preparation" in gap for gap in result["gaps"]))
        self.assertFalse(any("lease-record validation" in gap for gap in result["gaps"]))
        self.assertTrue(any("common trial ledger" in gap for gap in result["gaps"]))
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 0)

    def test_controller_process_receipts_are_append_only_without_actor_booking(self):
        r, path = self.request()
        self.authorize()
        argv = [str(Path(sys.executable).resolve()), "synthetic-controller"]
        self.ledger.reserve_controller_dispatch(r["dispatchId"], execution_sha(r), path.read_bytes(), argv,
                                                r["task"]["id"], r["purpose"],
                                                self.authority.grant["maxActorSessions"])
        self.assertTrue(self.ledger.claim_dispatch(r["dispatchId"], "launching"))
        recorded = self.ledger.record_controller_process(r["dispatchId"], "synthetic-step", time.time(),
                                                         {"status": "completed", "argv": argv,
                                                          "stdoutSha256": "1" * 64})
        snapshot = self.ledger.snapshot()
        self.assertEqual(snapshot["actorSessions"], 0)
        self.assertEqual(len(snapshot["controllerProcesses"]), 1)
        self.assertEqual(snapshot["controllerProcesses"][0]["action"], "synthetic-step")
        self.assertEqual(snapshot["controllerProcesses"][0]["receipt_sha256"], recorded["receiptSha256"])
        with self.assertRaisesRegex(ValueError, "immutable receipt"):
            self.ledger.record_controller_process(r["dispatchId"], "synthetic-step", time.time(),
                                                  {"status": "completed", "argv": argv})

    def test_government_translation_failure_finalizes_incomplete_with_valid_process_receipts(self):
        r, path = self.request(arm="government")
        auth_path = self.root / "native-role-auth.json"
        auth_raw = b'{"synthetic":"released role authorization"}'
        auth_path.write_bytes(auth_raw)
        binding = {"path": str(auth_path.resolve()), "sha256": digest(auth_raw)}
        r["product"] = {"government": {"roleAuthorization": binding}}
        r["releasedInputs"].append(binding)
        raw = encoded(r)
        path.write_bytes(raw)
        captured = {item["path"]: Path(item["path"]).read_bytes() for item in r["releasedInputs"]}
        self.authorize()
        evidence = Path(r["evidenceDirectory"])
        result_path = self.results / "government-incomplete.json"

        def write_bundle(path, **_kwargs):
            path = Path(path)
            path.write_bytes(b"synthetic controller bootstrap")
            return path, digest(path.read_bytes())

        def bounded_spy(argv, cwd, directory, _wall, **_kwargs):
            directory = Path(directory)
            directory.mkdir(parents=True)
            stdout = directory / "stdout.log"
            stderr = directory / "stderr.log"
            stdout.write_bytes(b'{"queue":"native result fixture"}')
            stderr.write_bytes(b"")
            receipts = [{"path": str(target.resolve()), "sha256": digest(target.read_bytes())}
                        for target in (stdout, stderr)]
            process = {"argv": argv, "cwd": cwd, "returnCode": 0, "wallSeconds": 0.01,
                       "receipts": receipts}
            (directory / "process.json").write_bytes(encoded(process))
            return {"returnCode": 0}

        with patch("government.bind_request", return_value={}), \
             patch("native_controller.validate_native_fixture_grant", return_value={}), \
             patch("government_roles.preflight_authorization"), \
             patch("government.plan_request", return_value={"argv": [str(self.fixture), "government"]}), \
             patch("native_controller.write_bundle", side_effect=write_bundle), \
             patch.object(dispatch_module, "bounded", side_effect=bounded_spy), \
             patch("government.translate_queue_result", side_effect=ValueError("missing evidence identity")):
            result = dispatch_module.dispatch_government(path, result_path, self.authority, r, raw,
                                                         captured, dispatch_module.base_result(r, raw))

        self.assertEqual(result["status"], "incomplete")
        self.assertIsNone(result["candidateCommit"])
        self.assertTrue(any("missing evidence identity" in gap for gap in result["gaps"]))
        kinds = {item["kind"] for item in result["receipts"]}
        self.assertTrue({"controller-process", "controller-bootstrap",
                         "released-input-snapshot"}.issubset(kinds))
        controller = self.ledger.snapshot()["controllerRuns"]
        self.assertEqual(len(controller), 1)
        self.assertEqual(controller[0]["status"], "incomplete")
        self.assertIsNotNone(controller[0]["end"])
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 0)

    def test_no_mode_only_launch_or_arbitrary_mechanical_program(self):
        r, path = self.request()
        with self.assertRaises(ValueError):
            handle(path, self.root / "result.json")
        self.authorize()
        self.fixture.write_text("raise Exception('not authorized')")
        with self.assertRaises(ValueError):
            self.call(path)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 0)

    def test_authority_digest_request_and_no_refill(self):
        r, path = self.request()
        self.authorize()
        with self.assertRaises(ValueError):
            Authority(self.grant_path, "0" * 64, self.protocol_path, self.authority.protocol_sha)
        path.write_bytes(path.read_bytes() + b"\n")
        with self.assertRaises(ValueError):
            self.call(path)
        self.authority.grant_sha = "different"
        with self.assertRaises(ValueError):
            self.authority.ledger()
        self.assertEqual(self.launches(), 0)

    def test_harness_adapter_path_and_grant_session_cap(self):
        _, first = self.request(purpose="setup")
        _, second = self.request("two", purpose="child")
        self.authorize(maximum=1)
        parent = Path(__file__).parents[1]
        sys.path.insert(0, str(parent))
        spec = importlib.util.spec_from_file_location("s1_harness", parent / "harness.py")
        harness = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(harness)
        result = harness.dispatch(first, self.results / "result.json", grant_path=self.grant_path,
                                  grant_sha256=self.authority.grant_sha, protocol_path=self.protocol_path,
                                  protocol_sha256=self.authority.protocol_sha)
        self.assertEqual(result["status"], "completed")
        self.assertEqual(self.call(second)["status"], "blocked")
        self.assertEqual(self.launches(), 1)

    def test_public_adapter_cli_forwards_exact_operator_authority(self):
        _, path = self.request(purpose="setup")
        self.authorize()
        result_path = self.results / "cli-result.json"
        process = subprocess.run([sys.executable, str(Path(__file__).with_name("adapter.py")),
                                  "--request", str(path), "--result", str(result_path),
                                  "--grant", str(self.grant_path), "--grant-sha256", self.authority.grant_sha,
                                  "--protocol", str(self.protocol_path), "--protocol-sha256", self.authority.protocol_sha],
                                 capture_output=True, timeout=10, shell=False)
        (self.root / "cli-stdout.log").write_bytes(process.stdout)
        (self.root / "cli-stderr.log").write_bytes(process.stderr)
        self.assertEqual(process.returncode, 0, process.stderr)
        self.assertEqual(json.loads(result_path.read_bytes())["status"], "completed")
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 1)
        self.assertEqual(self.launches(), 1)

    def test_result_paths_are_confined_before_reservation(self):
        _, path = self.request()
        self.authorize()
        for output in (self.grant_path, self.root / "config.toml", self.actor / "result.json", path):
            with self.subTest(output=output), self.assertRaises(ValueError):
                handle(path, output, authority=self.authority)
        self.assertEqual(self.ledger.snapshot()["actorSessions"], 0)

    def test_observed_usage_cannot_decrease(self):
        r, path = self.request()
        self.authorize()
        attempt = self.reserve(r, path)
        self.ledger.observe(attempt, 1, 10)
        self.ledger.observe(attempt, None, 2)
        self.assertEqual(self.ledger.snapshot()["providerTokens"], 10)
        with self.assertRaisesRegex(ValueError, "cannot decrease"):
            self.ledger.complete_dispatch(r["dispatchId"], {"status": "completed", "receipts": []}, None, 2)

    def test_expired_grant_cannot_start_but_can_recover(self):
        r, path = self.request()
        self.authorize()
        self.reserve(r, path)
        self.authority.grant["expiresAt"] = time.time() - 1
        with self.assertRaisesRegex(ValueError, "not currently valid"):
            self.call(path)
        self.assertEqual(self.call(self.operation(r, "resume"))["status"], "incomplete")
        self.assertEqual(self.launches(), 0)

    def test_listing_requires_exact_successful_metadata_contents(self):
        metadata = self.root / "listing.json"
        value = {"status": "advertised", "accountType": "chatgpt", "requestedHighAdvertised": True,
                 "rpcErrors": [], "processReturnCode": 0, "processStopReason": None,
                 "targetEntries": [{"id": "gpt-6.1-sol", "model": "gpt-6.1-sol",
                                    "supportedReasoningEfforts": [{"reasoningEffort": "high"}]}]}
        def validate(v):
            metadata.write_bytes(encoded(v))
            validate_listing({"path": str(metadata), "sha256": digest(encoded(v))})
        validate(value)
        for key, wrong in (("status", "unknown"), ("accountType", "apiKey"), ("processReturnCode", 1),
                           ("targetEntries", []), ("rpcErrors", ["failed"])):
            with self.subTest(key=key), self.assertRaises(ValueError):
                validate(dict(value, **{key: wrong}))

    def test_cumulative_wall_and_human_time_stop_running_work(self):
        r, path = self.request()
        self.authorize()
        self.reserve(r, path)
        self.ledger.human(COMMON["trialActiveHumanSeconds"], "synthetic intervention timing")
        self.assertEqual(self.ledger.running_bound(r["task"]["id"])[1], "trial_stopped")
        with self.ledger.transaction() as db:
            db.execute("UPDATE trial SET stopped=0")  # Synthetic test fixture only, no historical database.
            db.execute("UPDATE tasks SET start=?", (time.time() - COMMON["taskWallSeconds"] - 1,))
        self.assertEqual(self.ledger.running_bound(r["task"]["id"])[1], "cumulative_wall_deadline")

    def test_native_events_are_classified_without_running_native_actor(self):
        values = [{"type": "item.started", "item": {"type": "command_execution"}}]
        self.assertTrue(tool_observations(values, "forbidden")["unexpectedToolActivity"])
        self.assertFalse(tool_observations(values, "ordinary-tools")["unexpectedToolActivity"])
        values.append({"type": "item.completed", "item": {"type": "collab_tool_call"}})
        self.assertTrue(tool_observations(values, "ordinary-tools")["unexpectedToolActivity"])

    def test_active_known_subtotals_detect_parallel_overshoot(self):
        one, first = self.request()
        two, second = self.request("two")
        self.authorize()
        a, b = self.reserve(one, first), self.reserve(two, second)
        per_call = COMMON["taskProviderTokens"] // 2 + 1
        self.ledger.observe(a, 1, per_call)
        self.ledger.observe(b, 1, per_call)
        self.assertEqual(self.ledger.running_bound(one["task"]["id"])[1], "retrospective_taskProviderTokens")
        self.assertEqual(self.ledger.snapshot()["providerTokens"], COMMON["taskProviderTokens"] + 2)
        self.assertFalse(self.ledger.snapshot()["hardProviderTurnTokenCap"])

    def test_live_mode_does_not_assert_inference_before_launch(self):
        r, _ = self.request()
        r["mode"] = "live"
        self.assertFalse(base_result(r, encoded(r))["inferencePerformed"])
        self.authorize()
        grant = json.loads(self.grant_path.read_bytes())
        protocol = json.loads(self.protocol_path.read_bytes())
        grant.update(mode="live", purpose="s1-public-smoke", acceptedObservabilityGaps=LIVE_GAPS)
        protocol["mode"] = "live"
        self.protocol_path.write_bytes(encoded(protocol))
        grant["protocolSha256"] = digest(encoded(protocol))
        self.grant_path.write_bytes(encoded(grant))
        for switch in (False, "true", 1):
            with self.subTest(switch=switch), self.assertRaisesRegex(ValueError, "explicit operator live switch"):
                Authority(self.grant_path, digest(encoded(grant)), self.protocol_path, digest(encoded(protocol)), allow_live=switch)
        self.assertEqual(self.launches(), 0)

    def test_unresolved_cumulative_binding_cannot_be_approved_by_status_only(self):
        self.request()
        self.authorize()
        grant = json.loads(self.grant_path.read_bytes())
        grant["cumulativeLedgerBinding"] = {"status": "blocked-legacy-diagnostic-schema"}
        self.grant_path.write_bytes(encoded(grant))
        with self.assertRaisesRegex(ValueError, "superseded cumulative migration binding"):
            Authority(self.grant_path, digest(encoded(grant)), self.protocol_path, self.authority.protocol_sha)
        self.assertEqual(self.launches(), 0)

    def test_missing_partial_or_fresh_cumulative_binding_fails_closed(self):
        from context_allocation import validate_allocation
        for binding in (None, {}, {"status": "ready"}, {"status": "ready", "existingLedgerPath": str(self.root / "fresh.sqlite")}):
            with self.subTest(binding=binding):
                g = {"trialId": "fresh", "ledgerPath": str(self.root / "fresh.sqlite"), "maxActorSessions": 1}
                p = {}
                if binding is not None:
                    g["contextAllocation"] = binding
                    p["contextAllocation"] = binding
                with self.assertRaisesRegex(ValueError, "complete fixed additional context"):
                    validate_allocation(g, p)
                self.assertFalse((self.root / "fresh.sqlite").exists())

    def test_metadata_client_rejects_exec_argv_before_subprocess(self):
        import runner
        import selected_metadata
        pin = json.loads(Path(__file__).with_name("runner-pin.json").read_bytes())
        bad = {"runnerPin": pin, "argv": [pin["path"], "exec", "--model", "gpt-6.1-sol"],
               "initialize": {"method": "initialize", "id": 0}}
        path = self.root / "bad-metadata.json"
        path.write_bytes(encoded(bad))
        with patch("selected_metadata.subprocess.Popen") as launch:
            with self.assertRaisesRegex(ValueError, "exact pinned metadata argv"):
                selected_metadata.client(path)
            launch.assert_not_called()

    def test_prompt_bytes_are_captured_and_mutable_actor_inputs_rejected(self):
        _, path = self.request()
        self.authorize()
        r, captured = self.authority.validate(path.read_bytes())
        expected = self.card.read_bytes()
        self.card.write_text("changed after validation")
        self.assertEqual(captured[r["prompt"]["path"]], expected)
        owned = self.actor / "mutable-card.txt"
        owned.write_bytes(expected)
        r["releasedInputs"][0]["path"] = str(owned)
        r["task"]["card"]["path"] = str(owned)
        r["prompt"]["path"] = str(owned)
        self.authority.grant["authorizedRequests"][0]["executionSha256"] = execution_sha(r)
        self.authority.grant["authorizedRequests"][0]["initialRequestSha256"] = digest(encoded(r))
        with self.assertRaisesRegex(ValueError, "separate from mutable Actor"):
            self.authority.validate(encoded(r))


if __name__ == "__main__":
    print("Mechanical evidence root: " + str(ROOT), flush=True)
    unittest.main(verbosity=2)
