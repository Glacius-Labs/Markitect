import json
import unittest

from support import CliCase


class Tags(CliCase):
    def test_add_and_remove_are_idempotent_sorted_and_logged(self):
        self.add("a")
        self.assertEqual(self.ok("tag", "--id", "a", "--add", " sci \t fi ")["tags"], ["sci fi"])
        self.assertEqual(self.ok("tag", "--id", "a", "--add", "Classic")["tags"], ["Classic", "sci fi"])
        self.assertEqual(self.ok("tag", "--id", "a", "--add", "classic")["tags"], ["Classic", "classic", "sci fi"])
        self.assertEqual(self.ok("tag", "--id", "a", "--add", "Classic")["tags"], ["Classic", "classic", "sci fi"])
        self.assertEqual(self.ok("tag", "--id", "a", "--remove", "missing")["tags"], ["Classic", "classic", "sci fi"])
        self.assertEqual(self.ok("tag", "--id", "a", "--remove", "sci fi")["tags"], ["Classic", "classic"])
        self.assertEqual(self.logged()[1:], [("tag", ["a"])] * 6)
        self.assertEqual(self.ok("list")["books"][0]["tags"], ["Classic", "classic"])

    def test_invalid_tagging_changes_nothing(self):
        self.fails("not_found", "tag", "--id", "a", "--add", "x")
        self.add("a")
        for args in (["--add", "a;b"], ["--add", " "], ["--add", "x", "--remove", "y"], []):
            with self.subTest(args=args):
                self.fails("invalid_input", "tag", "--id", "a", *args)
        self.fails("invalid_input", "list", "--tag", "")

    def test_books_stored_without_tags_stay_readable_and_are_not_rewritten(self):
        old = [{"id": "o", "title": "Old", "author": "W", "pages": 3, "status": "finished"}]
        self.db.write_text(json.dumps(old), encoding="utf-8")
        before = self.state()
        self.assertEqual(self.ok("list")["books"], [dict(old[0], tags=[])])
        self.ok("list", "--tag", "x")
        self.ok("summary")
        self.ok("export", "--csv", str(self.folder / "out.csv"))
        self.assertEqual(self.state(), before)
        self.assertEqual(self.ok("tag", "--id", "o", "--add", "x")["tags"], ["x"])
        self.assertEqual(self.ok("list")["books"], [dict(old[0], tags=["x"])])

    def test_imported_books_start_without_tags(self):
        path = self.folder / "in.csv"
        path.write_text("id,title,author,pages\nb,T,W,1\n", encoding="utf-8")
        self.ok("import", "--csv", str(path))
        self.assertEqual(self.ok("list")["books"][0]["tags"], [])


if __name__ == "__main__":
    unittest.main()
