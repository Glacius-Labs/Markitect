"""Only targeted counter/session/observer risk checks; no provider execution."""
import json
from contextlib import closing
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
from identity_probe import reserve, summarize
from process import bounded


class IdentityProbeTests(unittest.TestCase):
    def test_failed_start_consumes_attempt_and_prevents_second_probe(self):
        with tempfile.TemporaryDirectory(prefix="scientist-probe-limit-") as directory:
            database = Path(directory) / "budget.sqlite"
            reserve(database, 1)
            with self.assertRaises(ValueError):
                reserve(database, 1)
            with self.assertRaises(ValueError):
                reserve(database, 2)
            with closing(sqlite3.connect(database)) as db:
                with db:
                    db.execute("UPDATE attempts SET end=1,result=?", (json.dumps({"status": "failed", "usage": {"reportedInputPlusOutputTokens": None}}),))
            with self.assertRaisesRegex(ValueError, "executable first probe"):
                reserve(database, 2)

    def test_agent_usage_is_not_provider_request_count_or_double_added(self):
        usage = summarize([{"type": "turn.started"}, {"type": "turn.completed", "usage": {
            "input_tokens": 9000, "cached_input_tokens": 8000, "output_tokens": 1000,
            "reasoning_output_tokens": 500}}])
        self.assertEqual(usage["reportedInputPlusOutputTokens"], 10000)
        self.assertIsNone(usage["providerRequests"])
        self.assertIsNone(summarize([])["reportedInputPlusOutputTokens"])

    def test_observer_threshold_stops_process_without_retry(self):
        with tempfile.TemporaryDirectory(prefix="scientist-probe-observer-") as directory:
            root = Path(directory)
            result = bounded([sys.executable, "-c", "import time;print('counter',flush=True);time.sleep(30)"],
                             str(root), root / "receipt", 5,
                             poll_stop=lambda path: "retrospective_token_threshold" if (path / "stdout.log").stat().st_size else None)
            self.assertEqual(result["stopReason"], "retrospective_token_threshold")
            self.assertEqual(result["automaticRetries"], 0)


if __name__ == "__main__":
    unittest.main()
