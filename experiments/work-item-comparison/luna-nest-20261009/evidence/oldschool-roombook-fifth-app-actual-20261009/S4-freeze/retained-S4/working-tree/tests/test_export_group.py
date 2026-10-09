import csv
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from roombook_export import ExportError, export_csv  # noqa: E402


class CsvExportTests(unittest.TestCase):
    def setUp(self):
        self.folder = tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.root = Path(self.folder.name)
        self.database = self.root / "state.json"
        self.database.write_bytes(b'{"reservations":[]}\n')

    def test_csv_escapes_commas_and_embedded_newlines_and_preserves_order(self):
        records = [
            {
                "id": 2,
                "room": "North, Wing",
                "start": "2026-10-09T11:00Z",
                "end": "2026-10-09T12:00Z",
                "title": "Review, plan\nand decide",
                "status": "active",
            },
            {
                "id": 1,
                "room": "South",
                "start": "2026-10-09T10:00Z",
                "end": "2026-10-09T11:00Z",
                "title": 'A "quoted" title',
                "status": "canceled",
            },
        ]
        output = self.root / "bookings.csv"

        self.assertEqual(export_csv(self.database, output, records), 2)
        with output.open(encoding="utf-8", newline="") as handle:
            rows = list(csv.reader(handle))

        self.assertEqual(rows, [
            ["id", "room", "start", "end", "title", "status"],
            ["2", "North, Wing", "2026-10-09T11:00Z", "2026-10-09T12:00Z", "Review, plan\nand decide", "active"],
            ["1", "South", "2026-10-09T10:00Z", "2026-10-09T11:00Z", 'A "quoted" title', "canceled"],
        ])

    def test_empty_export_writes_header_and_returns_zero(self):
        output = self.root / "empty.csv"

        self.assertEqual(export_csv(self.database, output, []), 0)
        self.assertEqual(output.read_text(encoding="utf-8"), "id,room,start,end,title,status\n")

    def test_export_keeps_canceled_history_and_does_not_change_database_bytes(self):
        original = b'{"reservations":[{"id":1,"status":"canceled"}]}\n'
        self.database.write_bytes(original)
        output = self.root / "history.csv"
        record = {
            "id": 1,
            "room": "Atlas",
            "start": "2026-10-09T10:00Z",
            "end": "2026-10-09T11:00Z",
            "title": "Past booking",
            "status": "canceled",
        }

        self.assertEqual(export_csv(self.database, output, [record]), 1)
        self.assertIn(",canceled\n", output.read_text(encoding="utf-8"))
        self.assertEqual(self.database.read_bytes(), original)

    def test_database_destination_alias_is_rejected_before_truncation(self):
        original = self.database.read_bytes()

        with self.assertRaisesRegex(ExportError, "must not be the database file"):
            export_csv(self.database, self.database, [])

        self.assertEqual(self.database.read_bytes(), original)

    def test_hard_link_alias_is_rejected_when_supported(self):
        alias = self.root / "database-alias.json"
        try:
            alias.hardlink_to(self.database)
        except (OSError, NotImplementedError) as error:
            self.skipTest(f"hard links unavailable: {error}")
        original = self.database.read_bytes()

        with self.assertRaisesRegex(ExportError, "must not be the database file"):
            export_csv(self.database, alias, [])

        self.assertEqual(self.database.read_bytes(), original)

    def test_output_failure_is_reported_clearly(self):
        missing_parent_output = self.root / "missing" / "export.csv"

        with self.assertRaisesRegex(ExportError, "cannot write CSV export"):
            export_csv(self.database, missing_parent_output, [])


if __name__ == "__main__":
    unittest.main()
