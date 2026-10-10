import unittest

from support import CliCase


class Summary(CliCase):
    def test_empty_database(self):
        self.assertEqual(self.ok("summary"), {"authors": []})
        self.assertFalse(self.db.exists())

    def test_counts_all_books_and_orders_authors_by_code_point(self):
        self.add("a", author="anne", pages="10")
        self.add("b", author="Zed", pages="7")
        self.add("c", author="anne", pages="5")
        self.ok("finish", "--id", "c")
        before = self.state()
        self.assertEqual(self.ok("summary"), {"authors": [
            {"author": "Zed", "books": 1, "finished": 0, "pages": 7},
            {"author": "anne", "books": 2, "finished": 1, "pages": 15}]})
        self.assertEqual(self.state(), before)


if __name__ == "__main__":
    unittest.main()
