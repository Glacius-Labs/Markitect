"""Risk checks: concurrent budgets, fail-closed unknown usage, and descendant stops."""
from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import sys
import tempfile
import time
import unittest

sys.path.insert(0, str(Path(__file__).parent))
from ledger import Ledger, LimitReached
from process import bounded
import adapter
import runner

LIMITS = {"taskWallSeconds": 1200, "trialWallSeconds": 7200, "taskActorCalls": 12,
          "trialActorCalls": 72, "taskProviderTurns": 80, "trialProviderTurns": 480,
          "taskProviderTokens": 120000, "trialProviderTokens": 720000,
          "maxParallelActors": 4, "trialActiveHumanSeconds": 600}


class RuntimeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="scientist-runtime-test-")
        self.root = Path(self.temp.name)

    def tearDown(self):
        self.temp.cleanup()

    def ledger(self, **overrides):
        return Ledger(self.root / "ledger.sqlite", "trial", {**LIMITS, **overrides})

    def test_parallel_reservation_is_atomic_and_counts_setup_reviews_failures(self):
        ledger = self.ledger()
        def reserve(index):
            try:
                return ledger.reserve("setup", "review" if index % 2 else "initial-model")
            except LimitReached:
                return None
        with ThreadPoolExecutor(max_workers=12) as pool:
            calls = [value for value in pool.map(reserve, range(12)) if value]
        self.assertEqual(len(calls), 4)
        for call in calls:
            ledger.finish(call, "failed-before-provider", provider_turns=0, provider_tokens=0)
        self.assertEqual(ledger.snapshot()["actorSessions"], 4)

    def test_unknown_usage_blocks_next_call_and_cannot_refill_profile(self):
        ledger = self.ledger()
        call = ledger.reserve("1", "executor")
        ledger.finish(call, "failed", receipt={"raw": "partial"})
        self.assertIsNone(ledger.snapshot()["providerTokens"])
        with self.assertRaisesRegex(LimitReached, "unknown provider usage"):
            ledger.reserve("2", "repair")
        with self.assertRaisesRegex(ValueError, "cannot change"):
            self.ledger(trialActorCalls=73)
        with self.assertRaisesRegex(ValueError, "already finished"):
            ledger.finish(call, "completed", provider_turns=0, provider_tokens=0)

    def test_known_cap_and_human_stop_persist_across_reopen(self):
        ledger = self.ledger(taskProviderTokens=10)
        call = ledger.reserve("1", "executor")
        ledger.finish(call, "completed", provider_turns=1, provider_tokens=10)
        with self.assertRaisesRegex(LimitReached, "taskProviderTokens"):
            ledger.reserve("1", "review")
        ledger.human(600, "mandatory intervention")
        reopened = self.ledger(taskProviderTokens=10)
        self.assertTrue(reopened.snapshot()["stopped"])
        with self.assertRaises(LimitReached):
            reopened.reserve("2", "executor")

    def test_process_deadline_kills_grandchild(self):
        marker = self.root / "late-child.txt"
        child_code = "import time,pathlib;time.sleep(2);pathlib.Path(" + repr(str(marker)) + ").write_text('leaked')"
        parent_code = "import subprocess,sys,time;subprocess.Popen([sys.executable,'-c'," + repr(child_code) + "]);print('spawned',flush=True);time.sleep(30)"
        receipt = bounded([sys.executable, "-c", parent_code], str(self.root), self.root / "deadline", 0.7)
        self.assertEqual(receipt["stopReason"], "wall_deadline")
        self.assertIn("spawned", (self.root / "deadline/stdout.log").read_text())
        time.sleep(2)
        self.assertFalse(marker.exists(), "descendant survived the process-tree deadline")

    def test_stop_sentinel_kills_active_process(self):
        from threading import Timer
        stop = self.root / "STOP"
        timer = Timer(0.3, lambda: stop.touch())
        timer.start()
        try:
            receipt = bounded([sys.executable, "-c", "import time;time.sleep(30)"],
                              str(self.root), self.root / "stop", 5, stop_path=stop)
        finally:
            timer.join()
        self.assertEqual(receipt["stopReason"], "stop_requested")
        self.assertLess(receipt["wallSeconds"], 3)

    def test_preexisting_stop_never_releases_target(self):
        stop = self.root / "STOP"
        marker = self.root / "target-ran.txt"
        stop.touch()
        receipt = bounded([sys.executable, "-c", "import pathlib;pathlib.Path(" + repr(str(marker)) + ").touch()"],
                          str(self.root), self.root / "pre-stopped", 5, stop_path=stop)
        self.assertEqual(receipt["stopReason"], "stop_requested")
        self.assertFalse(marker.exists())

    def test_request_digest_binds_native_probe_and_government_stays_gap(self):
        request = {"schemaVersion": 1, "trialId": "native-smoke", "mode": "fixture", "operation": "probe",
                   "arm": "government", "condition": "greenfield", "actorRepository": str(self.root),
                   "task": None, "releasedInputs": [], "limits": {"wallSeconds": 30, "maxActorCalls": 0},
                   "evidenceDirectory": str(self.root)}
        path = self.root / "request.json"
        path.write_text(json.dumps(request), encoding="utf-8")
        result = adapter.handle(path, self.root / "result.json")
        self.assertEqual(result["requestSha256"], adapter.sha(path))
        self.assertEqual(result["status"], "readiness_gap")
        self.assertTrue(any("accepted" in gap for gap in result["gaps"]))
        self.assertFalse(any("lease-record validation" in gap for gap in result["gaps"]))
        self.assertTrue(any("common trial ledger" in gap for gap in result["gaps"]))
        self.assertEqual(result["receipts"], [])
        request["mode"] = "live"
        path.write_text(json.dumps(request), encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "Live study gate closed"):
            adapter.handle(path, self.root / "result.json")

    def test_no_provider_launch_path_before_counter_and_context_evidence(self):
        with self.assertRaisesRegex(ValueError, "Provider launch gate closed"):
            runner.require_provider_ready()


if __name__ == "__main__":
    unittest.main()
