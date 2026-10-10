import csv
import json
import unittest

from support import CliCase


class OptionalPages(CliCase):
    def test_add_without_or_with_empty_pages_stores_null(self):
        for entry_id, extra in (("a", []), ("b", ["--pages", ""]), ("c", ["--pages", " \t "])):
            with self.subTest(extra=extra):
                entry = self.ok("add", "--id", entry_id, "--title", "T", "--author", "W", *extra)
                self.assertIsNone(entry["pages"])
        self.assertEqual([entry["pages"] for entry in self.ok("list")["entries"]], [None, None, None])
        self.assertEqual(json.loads(self.db.read_text(encoding="utf-8"))["records"][0]["pages"], None)
        self.assertEqual(self.logged(), [("add", ["a"]), ("add", ["b"]), ("add", ["c"])])

    def test_given_pages_must_still_be_positive_integers(self):
        for pages in ("0", "-3", "x", "1.0"):
            with self.subTest(pages=pages):
                self.fails("invalid_input", "add", "--id", "a", "--title", "T", "--author", "W", "--pages", pages)

    def test_import_reads_an_empty_cell_as_unknown(self):
        path = self.folder / "in.csv"
        path.write_text('id,title,author,pages\na,T,W,\nb,T,W," "\nc,T,W,4\n', encoding="utf-8")
        self.assertEqual(self.ok("import", "--csv", str(path)), {"imported": 3})
        self.assertEqual([entry["pages"] for entry in self.ok("list")["entries"]], [None, None, 4])
        path.write_text("id,title,author,pages\nd,T,W,\ne,T,W,0\n", encoding="utf-8")
        self.fails("invalid_input", "import", "--csv", str(path))

    def test_summary_totals_known_pages_and_counts_unknown_ones(self):
        self.add("a", author="Ann", pages="10")
        self.ok("add", "--id", "b", "--title", "T", "--author", "Ann")
        self.ok("add", "--id", "c", "--title", "T", "--author", "Bo")
        self.ok("finish", "--id", "b")
        expected = [{"author": "Ann", "entries": 2, "finished": 1, "pages": 10, "unknownPages": 1},
                    {"author": "Bo", "entries": 1, "finished": 0, "pages": 0, "unknownPages": 1}]
        self.assertEqual(self.ok("summary"), {"authors": expected})
        self.assertEqual(self.ok("summary", "--author", "Bo"), {"authors": expected[1:]})

    def test_export_writes_an_empty_cell_and_other_commands_keep_null(self):
        self.ok("add", "--id", "a", "--title", "T", "--author", "W")
        self.ok("finish", "--id", "a")
        self.assertIsNone(self.ok("tag", "--id", "a", "--add", "x")["pages"])
        path = self.folder / "out.csv"
        self.ok("export", "--csv", str(path))
        with path.open(encoding="utf-8", newline="") as handle:
            rows = list(csv.reader(handle))
        self.assertEqual(rows[1], ["a", "T", "W", "", "finished", "x"])
        self.assertEqual(self.ids("--status", "finished", "--tag", "x"), ["a"])

    def test_unknown_pages_are_readable_in_both_storage_formats(self):
        entry = {"id": "a", "title": "T", "author": "W", "pages": None, "status": "unread"}
        for content in ([entry], {"schemaVersion": 2, "records": [dict(entry, tags=[])]}):
            with self.subTest(content=content):
                self.db.write_text(json.dumps(content), encoding="utf-8")
                before = self.state()
                self.assertEqual(self.ok("list")["entries"], [dict(entry, tags=[])])
                self.assertEqual(self.ok("summary")["authors"][0]["unknownPages"], 1)
                self.assertEqual(self.state(), before)


if __name__ == "__main__":
    unittest.main()
