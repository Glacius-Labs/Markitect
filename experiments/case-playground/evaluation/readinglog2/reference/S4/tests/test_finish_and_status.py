import unittest

from support import CliCase


class FinishAndStatus(CliCase):
    def test_finish_is_idempotent_and_logged_every_time(self):
        self.add("a")
        first = self.ok("finish", "--id", " a ")
        self.assertEqual(first["status"], "finished")
        self.assertEqual(self.ok("finish", "--id", "a"), first)
        self.assertEqual(self.ok("list")["books"], [first])
        self.assertEqual(self.logged(), [("add", ["a"]), ("finish", ["a"]), ("finish", ["a"])])

    def test_finish_errors_change_nothing(self):
        self.fails("not_found", "finish", "--id", "a")
        self.assertFalse(self.db.exists())
        self.add("a")
        self.fails("not_found", "finish", "--id", "A")
        self.fails("invalid_input", "finish", "--id", " \t ")
        self.fails("invalid_input", "finish")

    def test_status_filter_keeps_insertion_order(self):
        for book_id in ("c", "a", "b"):
            self.add(book_id)
        self.ok("finish", "--id", "b")
        self.ok("finish", "--id", "c")
        self.assertEqual(self.ids("--status", " finished "), ["c", "b"])
        self.assertEqual(self.ids("--status", "unread"), ["a"])
        self.assertEqual(self.ids(), ["c", "a", "b"])
        self.fails("invalid_input", "list", "--status", "Finished")
        self.assertEqual(len(self.logged()), 5)


if __name__ == "__main__":
    unittest.main()
