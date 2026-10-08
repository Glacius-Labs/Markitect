from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "list_open_listings.py"
VIEW = ROOT / "resources" / "list-open-listings.yaml"
FIXTURE = ROOT / "fixtures" / "listings.json"


class ListOpenListingsTests(unittest.TestCase):
    def run_script(self, items):
        with tempfile.TemporaryDirectory(prefix="list-open-listings-test-") as temp:
            proc = subprocess.run(
                [sys.executable, str(SCRIPT), "--view", str(VIEW)],
                input=json.dumps({"items": items}, separators=(",", ":")),
                text=True,
                encoding="utf-8",
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                cwd=temp,
                check=False,
            )
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertEqual(proc.stderr, "")
            self.assertEqual(list(Path(temp).iterdir()), [])
            return json.loads(proc.stdout)

    def test_empty_case_outputs_empty_items(self):
        fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
        result = self.run_script(fixture["cases"][0]["items"])
        self.assertEqual(result, {"items": []})

    def test_filters_sorts_and_preserves_complete_rows(self):
        fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
        items = fixture["cases"][1]["items"]
        result = self.run_script(items)
        rows = result["items"]

        self.assertEqual([row["id"] for row in rows], ["early", "tie-a", "tie-z", "late"])
        self.assertNotIn("closed", [row["id"] for row in rows])
        self.assertEqual(rows[1]["extra"], {"priority": 1})
        self.assertEqual(rows[2]["extra"], {"priority": 2})
        self.assertEqual(rows[3]["labels"], ["renewal"])
        self.assertEqual(rows[3]["note"], "keep me")

    def test_json_stdout_shape_is_single_object_with_items(self):
        result = self.run_script([
            {"id": "b", "status": "Open", "openedAt": "2026-01-01T00:00:00Z"},
            {"id": "a", "status": "Open", "openedAt": "2026-01-01T00:00:00Z"},
            {"id": "c", "status": "Closed", "openedAt": "2026-01-01T00:00:00Z"},
        ])
        self.assertEqual(result, {"items": [
            {"id": "a", "status": "Open", "openedAt": "2026-01-01T00:00:00Z"},
            {"id": "b", "status": "Open", "openedAt": "2026-01-01T00:00:00Z"},
        ]})


if __name__ == "__main__":
    unittest.main()
