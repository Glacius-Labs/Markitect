import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

APP = Path(__file__).resolve().parents[1] / "app.py"


class BookingCliTests(unittest.TestCase):
    def setUp(self):
        self.folder = tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.db = Path(self.folder.name) / "state.json"

    def call(self, *arguments, expected=0):
        result = subprocess.run(
            [sys.executable, "-B", str(APP), "--db", str(self.db), *arguments],
            cwd=APP.parent, capture_output=True, text=True, check=False,
        )
        self.assertEqual(result.returncode, expected, (result.stdout, result.stderr))
        self.assertEqual(result.stderr, "")
        return json.loads(result.stdout)

    def book(self, room, start, end, title="Planning"):
        return self.call("book", "--room", room, "--start", start, "--end", end, "--title", title)

    def test_normalizes_and_persists_between_processes(self):
        created = self.book(" Atlas ", "2026-10-09T10:00Z", "2026-10-09T11:00Z", "  Planning  ")
        self.assertEqual(created, {
            "id": 1, "room": "Atlas", "start": "2026-10-09T10:00Z",
            "end": "2026-10-09T11:00Z", "title": "Planning", "status": "active",
        })
        self.assertEqual(self.call("list"), {"reservations": [created]})
        self.assertEqual(self.call("list", "--room", " Atlas "), {"reservations": [created]})

    def test_half_open_intervals_and_other_rooms(self):
        self.book("A", "2026-10-09T10:00Z", "2026-10-09T11:00Z")
        adjacent = self.book("A", "2026-10-09T11:00Z", "2026-10-09T11:30Z", "Next")
        other_room = self.book("B", "2026-10-09T10:30Z", "2026-10-09T11:30Z", "Elsewhere")
        self.assertEqual([item["id"] for item in self.call("list")["reservations"]], [1, 3, 2])
        self.assertEqual(adjacent["id"], 2)
        self.assertEqual(other_room["id"], 3)

    def test_overlaps_and_invalid_values_preserve_bytes_and_id_sequence(self):
        self.book("A", "2026-10-09T10:00Z", "2026-10-09T11:00Z")
        before = self.db.read_bytes()
        cases = [
            ("A", "2026-10-09T10:30Z", "2026-10-09T11:30Z", "Overlap"),
            ("A", "2026-10-09T10:00Z", "2026-10-09T10:00Z", "Zero"),
            (" ", "2026-10-09T12:00Z", "2026-10-09T13:00Z", "Room"),
            ("A", "2026-02-30T10:00Z", "2026-03-01T10:00Z", "Date"),
            ("A", "2026-10-09T12:00+00:00", "2026-10-09T13:00Z", "Offset"),
        ]
        for room, start, end, title in cases:
            with self.subTest(room=room, start=start, title=title):
                result = self.call("book", "--room", room, "--start", start, "--end", end,
                                   "--title", title, expected=2)
                self.assertIn("error", result)
                self.assertEqual(self.db.read_bytes(), before)
        self.assertEqual(self.book("A", "2026-10-09T11:00Z", "2026-10-09T12:00Z")["id"], 2)

    def test_list_sorts_by_start_then_id(self):
        self.book("A", "2026-10-09T12:00Z", "2026-10-09T13:00Z", "Later")
        self.book("A", "2026-10-09T10:00Z", "2026-10-09T11:00Z", "First")
        self.book("B", "2026-10-09T10:00Z", "2026-10-09T10:30Z", "Same time")
        values = self.call("list")["reservations"]
        self.assertEqual([(item["start"], item["id"]) for item in values], [
            ("2026-10-09T10:00Z", 2), ("2026-10-09T10:00Z", 3), ("2026-10-09T12:00Z", 1),
        ])

    def test_malformed_database_is_not_overwritten(self):
        self.db.write_bytes(b"{broken\xff")
        before = self.db.read_bytes()
        result = self.call("list", expected=2)
        self.assertIn("error", result)
        self.assertEqual(self.db.read_bytes(), before)


if __name__ == "__main__":
    unittest.main()
