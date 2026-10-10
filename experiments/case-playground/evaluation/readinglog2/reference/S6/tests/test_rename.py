import csv
import json
import unittest

from support import CliCase

# Written before the rename: one entry without tags (version 1 era), one with tags.
OLD_RECORDS = [{"id": "a", "title": "A", "author": "Ann", "pages": 10, "status": "finished"},
               {"id": "b", "title": "B", "author": "Ann", "pages": None, "status": "unread", "tags": ["x"]},
               {"id": "c", "title": "C", "author": "Bo", "pages": 3, "status": "finished", "tags": ["x"]}]
FILTERS = ([], ["--status", "finished"], ["--author", "Ann"], ["--tag", "x"],
           ["--status", "finished", "--tag", "x", "--author", "Bo"])


class Rename(CliCase):
    def write(self, content):
        self.db.write_text(json.dumps(content), encoding="utf-8")

    def test_new_and_legacy_shapes_on_pre_rename_data_in_both_formats(self):
        for content in (OLD_RECORDS, {"schemaVersion": 2, "records": OLD_RECORDS}):
            with self.subTest(version=1 if isinstance(content, list) else 2):
                self.write(content)
                before = self.state()
                for args in FILTERS:
                    current = self.ok("list", *args)
                    self.assertEqual(list(current), ["entries"])
                    self.assertEqual(self.ok("list", "--legacy", *args), {"books": current["entries"]})
                self.assertEqual(self.ids("--status", "finished", "--tag", "x"), ["c"])
                for args in ([], ["--author", "Ann"], ["--author", "Nobody"]):
                    current = self.ok("summary", *args)["authors"]
                    legacy = self.ok("summary", "--legacy", *args)["authors"]
                    self.assertEqual(legacy, [{("books" if key == "entries" else key): value
                                               for key, value in line.items()} for line in current])
                self.assertEqual(self.ok("summary", "--author", "Ann"), {"authors": [
                    {"author": "Ann", "entries": 2, "finished": 1, "pages": 10, "unknownPages": 1}]})
                self.assertEqual(self.state(), before)

    def test_stored_data_audit_operations_and_csv_headers_are_unchanged(self):
        self.write(OLD_RECORDS)
        path = self.folder / "in.csv"
        path.write_text("id,title,author,pages\nd,D,Ann,5\n", encoding="utf-8")
        self.ok("import", "--csv", str(path))
        self.add("e")
        self.ok("finish", "--id", "e")
        self.ok("tag", "--id", "e", "--add", "y")
        self.assertEqual([op for op, _ in self.logged()], ["import", "add", "finish", "tag"])
        stored = json.loads(self.db.read_text(encoding="utf-8"))
        self.assertEqual(sorted(stored), ["records", "schemaVersion"])
        self.assertEqual(stored["records"][0], dict(OLD_RECORDS[0], tags=[]))
        out = self.folder / "out.csv"
        self.ok("export", "--csv", str(out))
        with out.open(encoding="utf-8", newline="") as handle:
            self.assertEqual(next(csv.reader(handle)), ["id", "title", "author", "pages", "status", "tags"])


if __name__ == "__main__":
    unittest.main()
