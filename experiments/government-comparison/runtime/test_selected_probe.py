"""Finite grant, metadata gate, and reactive observer regressions; no providers."""
from contextlib import closing
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
import identity_probe
import remaining_probe
import selected_probe as probe


class SelectedProbeTests(unittest.TestCase):
    def test_separate_grant_preserves_unknown_history_and_rejects_refill(self):
        with tempfile.TemporaryDirectory() as directory:
            database = Path(directory) / "grant.sqlite"
            history = [{"number": number, "status": "failed", "usage": {"tokens": None}} for number in (1, 2)]
            probe.reserve(database, history, "first")
            with self.assertRaisesRegex(ValueError, "exhausted"):
                probe.reserve(database, history, "different-root-request")
            with closing(sqlite3.connect(database)) as db:
                row = db.execute("SELECT history,end FROM grants").fetchone()
            self.assertEqual(json.loads(row[0]), history)
            self.assertIsNone(row[1])

    def test_historical_launch_paths_are_closed_before_any_side_effect(self):
        for function in (identity_probe.run, remaining_probe.run):
            with self.assertRaisesRegex(ValueError, "Historical two-start grant exhausted"):
                function("does-not-exist", "no-runner")

    def test_current_catalog_requires_exact_target_high_and_normal_auth_metadata(self):
        values = [{"id": 1, "result": {"account": {"type": "chatgpt"}}},
                  {"id": 2, "result": {"data": [{"model": "gpt-6.1-sol", "supportedReasoningEfforts": [
                      {"reasoningEffort": "high"}]}], "nextCursor": None}}]
        self.assertEqual(probe.evaluate_metadata(values)["status"], "advertised")
        values[1]["result"]["data"][0]["model"] = "gpt-6-sol"
        self.assertEqual(probe.evaluate_metadata(values)["status"], "rejected")
        values[1]["result"]["nextCursor"] = "unread-page"
        self.assertEqual(probe.evaluate_metadata(values)["status"], "unknown")
        self.assertEqual(probe.evaluate_metadata([{"id": 2, "error": {"code": 403}}])["status"], "rejected")

    def test_reactive_monitor_and_usage_semantics(self):
        self.assertEqual(probe.stop_reason([{"type": "item.started", "item": {"type": "command_execution"}}]), "unexpected_tool_activity")
        self.assertEqual(probe.stop_reason([{"type": "turn.started"}] * 2), "unexpected_second_agent_turn")
        usage = [{"type": "turn.completed", "usage": {"input_tokens": 9000, "cached_input_tokens": 8000, "output_tokens": 1000}}]
        self.assertEqual(probe.stop_reason(usage), "retrospective_new_token_threshold")
        self.assertIsNone(probe.stop_reason([{"type": "turn.failed"}]))

    def test_unknown_or_missing_exact_high_cannot_reach_start_reservation(self):
        value = {"status": "advertised", "requestedHighAdvertised": True, "targetEntries": [
            {"model": "gpt-6.1-sol", "supportedReasoningEfforts": [{"reasoningEffort": "high"}]}]}
        probe.require_advertised(value)
        for changed in ({**value, "status": "unknown"}, {**value, "targetEntries": []},
                        {**value, "requestedHighAdvertised": False}):
            with self.assertRaisesRegex(ValueError, "does not affirm exact target/high"):
                probe.require_advertised(changed)

    def test_corrected_profile_preserves_selected_model_provider_without_custom_provider(self):
        settings = probe.config()
        self.assertEqual(settings["model_provider"], "openai")
        self.assertEqual(settings["model_reasoning_effort"], "high")
        self.assertEqual(settings["approval_policy"], "never")
        self.assertFalse(any(key.startswith("model_providers.") for key in settings))


if __name__ == "__main__":
    unittest.main()
