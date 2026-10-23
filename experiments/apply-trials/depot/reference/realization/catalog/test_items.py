import unittest

from catalog import items
from depot.errors import Refusal
from depot.store import empty


class SkuTests(unittest.TestCase):
    def test_sku_is_trimmed_and_uppercased(self):
        self.assertEqual(items.normalize_sku("  bol-006 "), "BOL-006")

    def test_sku_format_is_enforced(self):
        for text in ("BO-006", "BOLT-006", "BOL-06", "BOL006", "B0L-006", ""):
            with self.subTest(text=text), self.assertRaises(Refusal):
                items.normalize_sku(text)


class AddItemTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()

    def test_add_item_returns_the_catalog_record(self):
        self.assertEqual(items.add_item(self.state, "bol-006", " Bolt ", "5"),
                         {"sku": "BOL-006", "name": "Bolt", "reorderPoint": 5})

    def test_duplicate_sku_is_refused(self):
        items.add_item(self.state, "BOL-006", "Bolt", "0")
        with self.assertRaises(Refusal):
            items.add_item(self.state, " bol-006", "Other", "0")
        self.assertEqual(list(self.state["items"]), ["BOL-006"])

    def test_require_names_an_existing_item(self):
        items.add_item(self.state, "BOL-006", "Bolt", "0")
        self.assertEqual(items.require(self.state, "bol-006"), "BOL-006")
        with self.assertRaises(Refusal):
            items.require(self.state, "NUT-010")


if __name__ == "__main__":
    unittest.main()
