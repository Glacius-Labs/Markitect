import json
import unittest

from support import CliCase

V1_BOOKS = [{"id": "old", "title": "Old", "author": "Ann Lee", "pages": 5, "status": "unread"},
            {"id": "done", "title": "Done", "author": "Bo", "pages": 7, "status": "finished", "tags": ["x"]}]


class StorageFormats(CliCase):
    def write_v1(self):
        self.db.write_text(json.dumps(V1_BOOKS, indent=1) + "\n", encoding="utf-8")

    def stored(self):
        return json.loads(self.db.read_text(encoding="utf-8"))

    def test_new_database_is_written_as_version_2(self):
        book = self.add("a")
        self.assertEqual(self.stored(), {"schemaVersion": 2, "records": [book]})

    def test_read_only_commands_never_rewrite_a_version_1_file(self):
        self.write_v1()
        before = self.state()
        self.assertEqual(self.ids(), ["old", "done"])
        self.assertEqual(self.ids("--status", "finished", "--tag", "x", "--author", "Bo"), ["done"])
        self.assertEqual(self.ok("summary")["authors"][0], {"author": "Ann Lee", "books": 1, "finished": 0,
                                                            "pages": 5, "unknownPages": 0})
        self.ok("summary", "--author", "Bo")
        self.assertEqual(self.ok("export", "--csv", str(self.folder / "out.csv")), {"exported": 2})
        self.assertEqual(self.state(), before)

    def test_failed_mutations_leave_a_version_1_file_alone(self):
        self.write_v1()
        self.fails("duplicate", "add", "--id", "old", "--title", "T", "--author", "W", "--pages", "1")
        self.fails("not_found", "finish", "--id", "missing")
        self.fails("invalid_input", "tag", "--id", "old", "--add", "a;b")
        bad = self.folder / "bad.csv"
        bad.write_text("id,title,author,pages\nnew,T,W,0\n", encoding="utf-8")
        self.fails("invalid_input", "import", "--csv", str(bad))

    def test_every_successful_mutation_upgrades_a_version_1_file(self):
        csv_file = self.folder / "in.csv"
        csv_file.write_text("id,title,author,pages\n", encoding="utf-8")
        for args in (["add", "--id", "new", "--title", "T", "--author", "W", "--pages", "1"],
                     ["finish", "--id", "done"], ["tag", "--id", "old", "--remove", "absent"],
                     ["import", "--csv", str(csv_file)]):
            with self.subTest(command=args[0]):
                self.write_v1()
                expected = self.ok("list")["books"]
                self.ok(*args)
                stored = self.stored()
                self.assertEqual(sorted(stored), ["records", "schemaVersion"])
                self.assertEqual(stored["schemaVersion"], 2)
                if args[0] == "finish":
                    expected[1] = dict(expected[1], status="finished")
                self.assertEqual(stored["records"][:2], expected)

    def test_output_is_the_same_for_both_formats(self):
        self.write_v1()
        v1_list, v1_summary = self.ok("list"), self.ok("summary")
        self.ok("tag", "--id", "old", "--remove", "absent")
        self.assertEqual((self.ok("list"), self.ok("summary")), (v1_list, v1_summary))

    def test_other_shapes_are_malformed_and_never_overwritten(self):
        for content in ({"schemaVersion": 3, "records": []}, {"schemaVersion": "2", "records": []},
                        {"records": []}, {"schemaVersion": 2, "records": {}},
                        {"schemaVersion": 2, "records": [], "extra": 1},
                        {"schemaVersion": 2, "records": [{"id": "a"}]}):
            with self.subTest(content=content):
                self.db.write_text(json.dumps(content), encoding="utf-8")
                self.fails("storage", "list")
                self.fails("storage", "add", "--id", "b", "--title", "T", "--author", "W", "--pages", "1")


if __name__ == "__main__":
    unittest.main()
