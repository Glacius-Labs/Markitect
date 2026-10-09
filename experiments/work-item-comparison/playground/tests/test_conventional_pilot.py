"""Provider-free controller integration checks; no installed Codex dispatch."""
import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
from datetime import timedelta

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
spec = importlib.util.spec_from_file_location("pilot_controller_under_test", ROOT / "run_conventional_pilot.py")
pilot = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pilot)


class PilotTests(unittest.TestCase):
    def test_completed_scripted_turns_capture_all_four_public_stations(self):
        plan = {"id": "offline-controller-fixture", "model": "gpt-6-luna", "effort": "high",
                "jobWallSeconds": 14400, "turnWallSeconds": 5400, "startAllowance": 256}
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            with patch.object(pilot, "run_turn", side_effect=lambda *args: {
                    "state": "completed", "runId": "offline-" + args[-1], "nativeSessionId": "scripted-same"}):
                result = pilot.trajectory("roombook", "codex-cli", root, plan, Path(sys.executable),
                                          "offline fixture", "source-fixture", pilot.now() + timedelta(hours=8))
            self.assertEqual(result["status"], "trajectory_completed")
            self.assertEqual([row["station"] for row in result["stations"]], ["S1", "S2", "S3", "S4"])
            self.assertTrue((root / "roombook/audit/final-freeze/immutable-main").is_dir())
            self.assertEqual(result["startAccounting"]["outerAttempts"], 4)

    def test_mcp_observation_failure_is_recorded_without_replay_or_stop(self):
        with tempfile.TemporaryDirectory() as temporary:
            audit = Path(temporary)
            with patch.object(pilot, "mcp_status", side_effect=RuntimeError("scripted observation failure")):
                receipt = pilot.observe_mcp(audit / "config.json", "owned-run", audit, "S1")
            self.assertEqual(receipt["status"], "observation_failed")
            self.assertEqual(receipt["startsConsumed"], 0)
            self.assertTrue((audit / "operator-mcp-S1.error.json").exists())

    def test_known_native_tool_setup_failure_stops_after_terminal_first_turn(self):
        plan = {"id": "offline-controller-fixture", "model": "gpt-6-luna", "effort": "high",
                "jobWallSeconds": 14400, "turnWallSeconds": 5400, "startAllowance": 64}
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            def terminal_with_setup_error(config, audit, parent, expiry, station):
                events = audit / "conventional-execution/offline-native/events.jsonl"
                events.parent.mkdir(parents=True)
                events.write_bytes(b'{"event":{"raw":"helper_unknown_error: setup refresh had errors"}}\n')
                return {"state":"completed", "runId":"offline-native", "nativeSessionId":"scripted-same"}
            with patch.object(pilot, "run_turn", side_effect=terminal_with_setup_error) as run:
                result = pilot.trajectory("readinglog", "codex-app-server", root, plan, Path(sys.executable),
                                          "offline fixture", "source-fixture", pilot.now() + timedelta(hours=8))
            self.assertEqual(result["status"], "blocked_by_native_tool_setup")
            self.assertEqual(run.call_count, 1)
            self.assertEqual(len(result["stations"]), 1)
            self.assertTrue((root / "readinglog/audit/final-freeze/immutable-main").is_dir())
