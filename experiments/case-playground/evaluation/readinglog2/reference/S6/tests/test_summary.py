import unittest

from support import CliCase


class Summary(CliCase):
    def test_empty_database(self):
        self.assertEqual(self.ok("summary"), {"authors": []})
        self.assertFalse(self.db.exists())

    def test_counts_all_entries_and_orders_authors_by_code_point(self):
        self.add("a", author="anne", pages="10")
        self.add("b", author="Zed", pages="7")
        self.add("c", author="anne", pages="5")
        self.ok("finish", "--id", "c")
        before = self.state()
        self.assertEqual(self.ok("summary"), {"authors": [
            {"author": "Zed", "entries": 1, "finished": 0, "pages": 7, "unknownPages": 0},
            {"author": "anne", "entries": 2, "finished": 1, "pages": 15, "unknownPages": 0}]})
        self.assertEqual(self.state(), before)

    def test_author_filter_selects_one_exact_author(self):
        self.add("a", author="Ann Lee", pages="10")
        self.add("b", author="ann lee", pages="3")
        self.ok("finish", "--id", "a")
        self.assertEqual(self.ok("summary", "--author", " Ann  Lee "),
                         {"authors": [{"author": "Ann Lee", "entries": 1, "finished": 1, "pages": 10, "unknownPages": 0}]})
        self.assertEqual(self.ok("summary", "--author", "Unknown"), {"authors": []})
        self.fails("invalid_input", "summary", "--author", "")


if __name__ == "__main__":
    unittest.main()
