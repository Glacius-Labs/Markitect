import csv
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

    def test_cancel_is_idempotent_preserves_history_and_frees_interval(self):
        first = self.book("A", "2026-10-09T10:00Z", "2026-10-09T11:00Z", "First")
        adjacent = self.book("A", "2026-10-09T11:00Z", "2026-10-09T12:00Z", "Adjacent")
        canceled = {**first, "status": "canceled"}
        self.assertEqual(self.call("cancel", "--id", "1"), canceled)
        after_first_cancel = self.db.read_bytes()
        self.assertEqual(self.call("cancel", "--id", "1"), canceled)
        self.assertEqual(self.db.read_bytes(), after_first_cancel)
        self.assertEqual(self.call("list"), {"reservations": [canceled, adjacent]})

        replacement = self.book("A", "2026-10-09T10:00Z", "2026-10-09T11:00Z", "Replacement")
        self.assertEqual(replacement["id"], 3)
        before_missing = self.db.read_bytes()
        missing = self.call("cancel", "--id", "999", expected=2)
        self.assertIn("error", missing)
        self.assertEqual(self.db.read_bytes(), before_missing)
        malformed_id = self.call("cancel", "--id", "one", expected=2)
        self.assertIn("error", malformed_id)
        self.assertEqual(self.db.read_bytes(), before_missing)

    def test_summary_counts_only_active_exact_minutes_in_lexical_order(self):
        self.assertEqual(self.call("summary"), {"rooms": []})
        self.book("Z room", "2026-10-09T08:00Z", "2026-10-09T09:00Z", "One hour")
        canceled_room = self.book("B room", "2026-10-09T08:00Z", "2026-10-09T08:15Z", "Canceled")
        first = self.book("A room", "2026-10-09T10:00Z", "2026-10-09T10:20Z", "Twenty")
        self.book("A room", "2026-10-09T10:20Z", "2026-10-09T11:00Z", "Forty")
        self.call("cancel", "--id", str(canceled_room["id"]))
        self.call("cancel", "--id", str(first["id"]))
        self.assertEqual(self.call("summary"), {"rooms": [
            {"room": "A room", "active": 1, "minutes": 40},
            {"room": "Z room", "active": 1, "minutes": 60},
        ]})

    def test_status_and_room_filters_compose_and_keep_sorted_order(self):
        late = self.book("A", "2026-10-09T12:00Z", "2026-10-09T13:00Z", "Late")
        early_a = self.book("A", "2026-10-09T10:00Z", "2026-10-09T11:00Z", "Early A")
        early_b = self.book("B", "2026-10-09T10:00Z", "2026-10-09T10:30Z", "Early B")
        self.call("cancel", "--id", str(early_a["id"]))

        active = self.call("list", "--status", "active")["reservations"]
        self.assertEqual([record["id"] for record in active], [early_b["id"], late["id"]])
        self.assertEqual(self.call("list", "--status", "active", "--room", " A "), {
            "reservations": [late],
        })
        self.assertEqual(self.call("list", "--status", "canceled", "--room", " A ")["reservations"][0]["id"], early_a["id"])
        self.assertEqual(self.call("list", "--room", "Unknown"), {"reservations": []})
        self.assertEqual(self.call("list", "--status", "active"), {"reservations": active})

    def test_csv_export_quotes_history_preserves_database_and_reports_failures(self):
        first = self.book("Atlas", "2026-10-09T10:00Z", "2026-10-09T11:00Z", "Planning, review\nnext")
        second = self.book("Atlas", "2026-10-09T11:00Z", "2026-10-09T11:30Z", "Canceled row")
        self.call("cancel", "--id", str(second["id"]))
        before = self.db.read_bytes()
        output = Path(self.folder.name) / "bookings.csv"
        self.assertEqual(self.call("export", "--csv", str(output)), {"exported": 2})
        with output.open(encoding="utf-8", newline="") as handle:
            rows = list(csv.DictReader(handle))
        self.assertEqual(list(rows[0]), ["id", "room", "start", "end", "title", "status"])
        self.assertEqual(rows[0]["title"], first["title"])
        self.assertEqual([row["status"] for row in rows], ["active", "canceled"])
        self.assertEqual(self.db.read_bytes(), before)

        same_file = self.call("export", "--csv", str(self.db), expected=2)
        self.assertIn("error", same_file)
        self.assertEqual(self.db.read_bytes(), before)
        directory = Path(self.folder.name) / "as-directory"
        directory.mkdir()
        failed_write = self.call("export", "--csv", str(directory), expected=2)
        self.assertIn("error", failed_write)
        self.assertEqual(self.db.read_bytes(), before)

    def test_empty_export_has_header_and_filtered_summary_uses_active_records(self):
        empty_path = Path(self.folder.name) / "empty.csv"
        self.assertEqual(self.call("export", "--csv", str(empty_path)), {"exported": 0})
        self.assertEqual(empty_path.read_text(encoding="utf-8"), "id,room,start,end,title,status\n")

        self.book("Z room", "2026-10-09T08:00Z", "2026-10-09T09:00Z", "One hour")
        canceled_room = self.book("B room", "2026-10-09T08:00Z", "2026-10-09T08:15Z", "Canceled")
        first = self.book("A room", "2026-10-09T10:00Z", "2026-10-09T10:20Z", "Twenty")
        self.book("A room", "2026-10-09T10:20Z", "2026-10-09T11:00Z", "Forty")
        self.call("cancel", "--id", str(canceled_room["id"]))
        self.call("cancel", "--id", str(first["id"]))
        self.assertEqual(self.call("summary", "--room", " A room "), {
            "rooms": [{"room": "A room", "active": 1, "minutes": 40}],
        })
        self.assertEqual(self.call("summary", "--room", "unknown"), {"rooms": []})
        self.assertEqual(self.call("summary"), {"rooms": [
            {"room": "A room", "active": 1, "minutes": 40},
            {"room": "Z room", "active": 1, "minutes": 60},
        ]})


if __name__ == "__main__":
    unittest.main()
