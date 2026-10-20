import unittest

from catalog import items
from depot.errors import Refusal
from depot.store import empty
from orders import orderbook
from stock import ledger


class OrderTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()
        for sku in ("BOL-006", "NUT-010"):
            items.add_item(self.state, sku, "Part", "0")
            ledger.receive(self.state, sku, "10")

    def test_place_reserves_stock(self):
        order = orderbook.place(self.state, ["bol-006:4", "NUT-010:10"])
        self.assertEqual(order, {"id": 1, "status": "open",
                                 "lines": [{"sku": "BOL-006", "qty": 4}, {"sku": "NUT-010", "qty": 10}]})
        self.assertEqual((orderbook.reserved(self.state, "BOL-006"), orderbook.available(self.state, "BOL-006")), (4, 6))
        self.assertEqual(orderbook.place(self.state, ["BOL-006:6"])["id"], 2)

    def test_place_refuses_more_than_available(self):
        orderbook.place(self.state, ["BOL-006:8"])
        with self.assertRaises(Refusal):
            orderbook.place(self.state, ["BOL-006:3"])
        self.assertEqual(len(self.state["orders"]), 1)

    def test_place_refuses_invalid_lines(self):
        for lines in ([], ["BOL-006"], ["BOL-006:0"], ["XYZ-999:1"], ["BOL-006:1", "bol-006:1"], ["BOL-006:1"] * 11):
            with self.subTest(lines=lines), self.assertRaises(Refusal):
                orderbook.place(self.state, lines)
        self.assertEqual(self.state["orders"], [])

    def test_fulfil_takes_stock_and_releases_the_reservation(self):
        orderbook.place(self.state, ["BOL-006:4"])
        self.assertEqual(orderbook.fulfil(self.state, "1")["status"], "fulfilled")
        self.assertEqual(ledger.on_hand(self.state, "BOL-006"), 6)
        self.assertEqual(orderbook.reserved(self.state, "BOL-006"), 0)
        with self.assertRaises(Refusal):
            orderbook.fulfil(self.state, "1")

    def test_cancel_releases_and_repeats_without_change(self):
        orderbook.place(self.state, ["BOL-006:4"])
        self.assertEqual(orderbook.cancel(self.state, "1")["status"], "cancelled")
        self.assertEqual(orderbook.cancel(self.state, "001")["status"], "cancelled")
        self.assertEqual((ledger.on_hand(self.state, "BOL-006"), orderbook.available(self.state, "BOL-006")), (10, 10))
        with self.assertRaises(Refusal):
            orderbook.fulfil(self.state, "1")

    def test_listing_filters_by_status(self):
        orderbook.place(self.state, ["BOL-006:1"])
        orderbook.place(self.state, ["NUT-010:1"])
        orderbook.cancel(self.state, "2")
        self.assertEqual([order["id"] for order in orderbook.listing(self.state, "open")], [1])
        self.assertEqual([order["id"] for order in orderbook.listing(self.state)], [1, 2])
        with self.assertRaises(Refusal):
            orderbook.listing(self.state, "closed")


if __name__ == "__main__":
    unittest.main()
