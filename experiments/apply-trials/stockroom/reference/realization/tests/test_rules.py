import unittest

import stockroom
from stockroom import StockError


class InputTests(unittest.TestCase):
    def test_sku_is_trimmed_and_uppercased(self):
        self.assertEqual(stockroom.normalize_sku("  abc-123 "), "ABC-123")

    def test_sku_format_is_enforced(self):
        for text in ("AB-123", "ABCD-123", "ABC-12", "ABC123", "AB1-123", "ABC-12X", ""):
            with self.subTest(text=text), self.assertRaises(StockError):
                stockroom.normalize_sku(text)

    def test_name_is_trimmed_and_limited(self):
        self.assertEqual(stockroom.parse_name("  Bolt M6 "), "Bolt M6")
        self.assertEqual(stockroom.parse_name("x" * 60), "x" * 60)
        for text in ("", "   ", "x" * 61):
            with self.subTest(text=text), self.assertRaises(StockError):
                stockroom.parse_name(text)

    def test_quantity_accepts_digits_from_1_to_1000(self):
        self.assertEqual(stockroom.parse_quantity("1"), 1)
        self.assertEqual(stockroom.parse_quantity("0007"), 7)
        self.assertEqual(stockroom.parse_quantity("1000"), 1000)

    def test_quantity_refuses_other_text(self):
        for text in ("0", "1001", "-1", "+1", "1.0", "1e3", " 1", "", "one"):
            with self.subTest(text=text), self.assertRaises(StockError):
                stockroom.parse_quantity(text)


class StockTests(unittest.TestCase):
    def setUp(self):
        self.data = {}
        stockroom.add_item(self.data, "ABC-123", "Bolt")

    def test_add_item_starts_at_zero(self):
        self.assertEqual(stockroom.add_item(self.data, "XYZ-001", "Nut"), {"sku": "XYZ-001", "name": "Nut", "onHand": 0})

    def test_duplicate_sku_is_refused_after_normalization(self):
        with self.assertRaises(StockError):
            stockroom.add_item(self.data, " abc-123 ", "Other")
        self.assertEqual(self.data, {"ABC-123": {"name": "Bolt", "onHand": 0}})

    def test_receive_adds(self):
        self.assertEqual(stockroom.receive(self.data, "ABC-123", "40")["onHand"], 40)
        self.assertEqual(stockroom.receive(self.data, "abc-123", "2")["onHand"], 42)

    def test_receive_up_to_the_on_hand_limit(self):
        for _ in range(9):
            stockroom.receive(self.data, "ABC-123", "1000")
        self.assertEqual(stockroom.receive(self.data, "ABC-123", "999")["onHand"], 9999)
        with self.assertRaises(StockError):
            stockroom.receive(self.data, "ABC-123", "1")
        self.assertEqual(self.data["ABC-123"]["onHand"], 9999)

    def test_issue_subtracts(self):
        stockroom.receive(self.data, "ABC-123", "10")
        self.assertEqual(stockroom.issue(self.data, "ABC-123", "3")["onHand"], 7)

    def test_issue_all_leaves_zero(self):
        stockroom.receive(self.data, "ABC-123", "10")
        self.assertEqual(stockroom.issue(self.data, "ABC-123", "10")["onHand"], 0)

    def test_issue_more_than_on_hand_is_refused(self):
        stockroom.receive(self.data, "ABC-123", "10")
        with self.assertRaises(StockError):
            stockroom.issue(self.data, "ABC-123", "11")
        self.assertEqual(self.data["ABC-123"]["onHand"], 10)

    def test_unknown_sku_is_refused(self):
        for command in (stockroom.receive, stockroom.issue):
            with self.subTest(command=command.__name__), self.assertRaises(StockError):
                command(self.data, "XYZ-999", "1")

    def test_list_is_sorted_by_sku(self):
        stockroom.add_item(self.data, "AAA-001", "Anchor")
        self.assertEqual([item["sku"] for item in stockroom.list_items(self.data)], ["AAA-001", "ABC-123"])


if __name__ == "__main__":
    unittest.main()
