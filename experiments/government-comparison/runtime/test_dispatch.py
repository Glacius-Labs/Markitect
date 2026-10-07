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

from adapter import handle
from dispatch import Authority, LIVE_GAPS, base_result, command, digest, encoded, execution_sha, mechanical_pin, runtime_pins, validate_listing, tool_observations
from ledger import LimitReached
from process import bounded

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

    def authorize(self, maximum=8, threshold=10000):
        pin = digest(encoded(mechanical_pin()))
        p = {"status": "frozen", "mode": "mechanical", "commonLimits": COMMON,
             "runnerPinSha256": pin, "runtimeSourceSha256": runtime_pins(),
             "wrapperPythonSha256": digest(Path(sys.executable).read_bytes())}
        self.protocol_path = self.root / "protocol.json"
        self.protocol_path.write_bytes(encoded(p))
        g = {"schemaVersion": 1, "status": "approved", "purpose": "s1-mechanics", "mode": "mechanical",
             "trialId": self._testMethodName, "notBefore": time.time() - 1, "expiresAt": time.time() + 600,
             "protocolSha256": digest(encoded(p)), "profileSha256": digest(encoded(COMMON)), "runnerPinSha256": pin,
             "ledgerPath": str(self.root / "trial.sqlite"), "resultDirectory": str(self.results), "maxActorSessions": maximum,
             "maxSessionWallSeconds": 10, "retrospectiveTokenThreshold": threshold,
             "authorizedRequests": [{"dispatchId": r["dispatchId"], "executionSha256": execution_sha(r),
                                      "initialRequestSha256": digest(path.read_bytes())} for r, path in self.requests]}
        self.grant_path = self.root / "grant.json"
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
        self.assertEqual(self.call(path)["status"], "readiness_gap")
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
