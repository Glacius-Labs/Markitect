"""Targeted remaining-start authorization checks, no Actor/provider calls."""
from contextlib import closing
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
from remaining_probe import corrected_config, reserve_last


class RemainingProbeTests(unittest.TestCase):
    def test_builtin_provider_and_protection_flags_preserved(self):
        config = corrected_config()
        self.assertEqual(config["model_provider"], "openai")
        self.assertFalse(any(key.startswith("model_providers.") or key.endswith("base_url") for key in config))
        for key in ("features.shell_tool", "features.unified_exec", "features.multi_agent", "features.hooks", "features.plugins"):
            self.assertFalse(config[key])

    def test_unknown_old_usage_preserved_and_changing_actor_root_cannot_refill(self):
        with tempfile.TemporaryDirectory(prefix="scientist-final-quota-") as temporary:
            path = Path(temporary) / "budget.sqlite"
            previous = {"number": 1, "status": "failed", "requestSha256": "old", "usage": {"agentTurnsStarted": 0, "reportedInputPlusOutputTokens": None}}
            reserve_last(path, previous, "new", "actor-two")
            with self.assertRaisesRegex(ValueError, "authorization exhausted"):
                reserve_last(path, previous, "third", "different-root")
            with closing(sqlite3.connect(path)) as db:
                rows = db.execute("SELECT number,result FROM attempts ORDER BY number").fetchall()
            self.assertEqual([row[0] for row in rows], [1, 2])
            self.assertIsNone(json.loads(rows[0][1])["usage"]["reportedInputPlusOutputTokens"])

    def test_other_prerequisite_is_rejected_before_reserving(self):
        with tempfile.TemporaryDirectory(prefix="scientist-final-prerequisite-") as temporary:
            with self.assertRaisesRegex(ValueError, "preserved pre-turn config failure"):
                reserve_last(Path(temporary) / "budget.sqlite", {"number": 1, "status": "completed"}, "new", "actor")


if __name__ == "__main__":
    unittest.main()
