import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


APP = Path(__file__).resolve().parents[1] / "app.py"


class CliTests(unittest.TestCase):
    def setUp(self):
        self.folder = tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.db = Path(self.folder.name) / "state.json"

    def call(self, *args, status=0):
        result = subprocess.run(
            [sys.executable, "-B", str(APP), "--db", str(self.db), *args],
            capture_output=True, text=True, cwd=APP.parent, check=False,
        )
        self.assertEqual(result.returncode, status, result.stderr)
        self.assertEqual(result.stderr, "")
        return json.loads(result.stdout)

    def book(self, room, start, end, title="Meeting"):
        return self.call("book", "--room", room, "--start", start, "--end", end, "--title", title)

    def test_validation_overlap_and_error_do_not_mutate_state(self):
        first = self.book(" Room A ", "2026-10-09T10:00Z", "2026-10-09T11:00Z", "  Planning  ")
        before = self.db.read_bytes()
        for bad in (
            ("Room A", "2026-10-09T10:59Z", "2026-10-09T11:30Z", "Overlap"),
            ("Room A", "2026-10-09T12:00Z", "2026-10-09T12:00Z", "Zero length"),
            ("Room A", "2026-02-30T10:00Z", "2026-03-01T10:00Z", "Bad date"),
            ("Room A", "2026-10-09T10:00+00:00", "2026-10-09T11:00Z", "Offset"),
            ("   ", "2026-10-09T12:00Z", "2026-10-09T13:00Z", "Empty room"),
            ("Room A", "2026-10-09T12:00Z", "2026-10-09T13:00Z", "   "),
        ):
            with self.subTest(bad=bad):
                self.assertIn("error", self.call(
                    "book", "--room", bad[0], "--start", bad[1], "--end", bad[2], "--title", bad[3], status=2))
                self.assertEqual(self.db.read_bytes(), before)
        self.assertEqual(first["id"], 1)

    def test_adjacent_different_rooms_filter_sort_and_restart(self):
        later = self.book("B", "2026-10-09T11:00Z", "2026-10-09T12:00Z", "Later")
        adjacent = self.book("A", "2026-10-09T11:00Z", "2026-10-09T12:00Z", "Adjacent")
        earlier = self.book("A", "2026-10-09T09:00Z", "2026-10-09T10:00Z", "Earlier")
        self.assertEqual([r["id"] for r in self.call("list")["reservations"]], [3, 1, 2])
        self.assertEqual([r["id"] for r in self.call("list", "--room", " A ")["reservations"]], [3, 2])
        self.assertEqual((later["id"], adjacent["id"], earlier["id"]), (1, 2, 3))

    def test_malformed_database_is_preserved(self):
        self.db.write_text("{broken", encoding="utf-8")
        before = self.db.read_bytes()
        self.assertIn("error", self.call("list", status=2))
        self.assertEqual(self.db.read_bytes(), before)

    def test_write_failure_is_reported_without_creating_database(self):
        missing_parent_db = self.db.parent / "missing" / "state.json"
        result = subprocess.run(
            [sys.executable, "-B", str(APP), "--db", str(missing_parent_db),
             "book", "--room", "A", "--start", "2026-10-09T10:00Z",
             "--end", "2026-10-09T11:00Z", "--title", "No write"],
            capture_output=True, text=True, cwd=APP.parent, check=False,
        )
        self.assertEqual(result.returncode, 2)
        self.assertEqual(result.stderr, "")
        self.assertIn("error", json.loads(result.stdout))
        self.assertFalse(missing_parent_db.exists())


if __name__ == "__main__":
    unittest.main()
