import csv
import unittest

from support import CliCase

HEADER = ["id", "title", "author", "pages", "status", "tags"]


class Export(CliCase):
    def export(self, name="out.csv"):
        path = self.folder / name
        before = self.state()
        result = self.ok("export", "--csv", str(path))
        self.assertEqual(self.state(), before)
        with path.open(encoding="utf-8", newline="") as handle:
            return result, list(csv.reader(handle))

    def test_empty_database_writes_only_the_header(self):
        self.assertEqual(self.export(), ({"exported": 0}, [HEADER]))
        self.assertFalse(self.db.exists())

    def test_quotes_text_and_joins_tags_in_insertion_order(self):
        self.add("b", title='Say "hi", then go', author="Ann Lee", pages="12")
        self.add("a", title="Plain")
        self.ok("finish", "--id", "b")
        self.ok("tag", "--id", "b", "--add", "z, last")
        self.ok("tag", "--id", "b", "--add", "first")
        result, rows = self.export()
        self.assertEqual(result, {"exported": 2})
        self.assertEqual(rows, [HEADER, ["b", 'Say "hi", then go', "Ann Lee", "12", "finished", "first;z, last"],
                                ["a", "Plain", "Writer", "100", "unread", ""]])
        self.assertIn('"Say ""hi"", then go"', (self.folder / "out.csv").read_text(encoding="utf-8"))

    def test_unwritable_destination_fails_with_export_failed(self):
        self.add("a")
        for target in (self.folder / "missing" / "out.csv", self.folder, self.db):
            with self.subTest(target=target):
                self.fails("export_failed", "export", "--csv", str(target))
        self.fails("invalid_input", "export")


if __name__ == "__main__":
    unittest.main()
