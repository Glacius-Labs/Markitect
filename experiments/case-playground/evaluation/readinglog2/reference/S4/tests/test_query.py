import unittest

from support import CliCase


class ListFilters(CliCase):
    def setUp(self):
        super().setUp()
        self.add("1", author="Ann Lee")
        self.add("2", author="ann lee")
        self.add("3", author="Ann Lee")
        self.add("4", author="Bo")
        self.ok("finish", "--id", "3")
        self.ok("tag", "--id", "3", "--add", "classic")
        self.ok("tag", "--id", "1", "--add", "classic")
        self.ok("tag", "--id", "4", "--add", "classic")

    def test_author_filter_is_exact_after_normalization(self):
        self.assertEqual(self.ids("--author", " Ann \t Lee "), ["1", "3"])
        self.assertEqual(self.ids("--author", "ann lee"), ["2"])
        self.assertEqual(self.ids("--author", "Nobody"), [])
        self.fails("invalid_input", "list", "--author", "  ")

    def test_filters_combine_and_keep_insertion_order_across_processes(self):
        self.assertEqual(self.ids("--author", "Ann Lee", "--status", "unread"), ["1"])
        self.assertEqual(self.ids("--tag", "classic"), ["1", "3", "4"])
        self.assertEqual(self.ids("--tag", "classic", "--status", "finished", "--author", "Ann Lee"), ["3"])
        self.assertEqual(self.ids("--tag", "Classic"), [])
        self.assertEqual(self.ids(), ["1", "2", "3", "4"])


if __name__ == "__main__":
    unittest.main()
