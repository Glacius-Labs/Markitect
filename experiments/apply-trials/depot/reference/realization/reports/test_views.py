import unittest

from catalog import items
from depot.store import empty
from orders import orderbook
from reports import views
from stock import ledger


class ViewTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()
        items.add_item(self.state, "NUT-010", "Nut", "5")
        items.add_item(self.state, "BOL-006", "Bolt", "20")
        ledger.receive(self.state, "BOL-006", "12")

    def test_stock_view_shows_reservations(self):
        orderbook.place(self.state, ["BOL-006:2"])
        self.assertEqual(views.stock_view(self.state), [
            {"sku": "BOL-006", "name": "Bolt", "reorderPoint": 20, "onHand": 12, "reserved": 2, "available": 10},
            {"sku": "NUT-010", "name": "Nut", "reorderPoint": 5, "onHand": 0, "reserved": 0, "available": 0}])

    def test_reorder_lists_items_below_their_point_by_shortfall(self):
        self.assertEqual(views.reorder_view(self.state), [
            {"sku": "BOL-006", "name": "Bolt", "available": 12, "reorderPoint": 20, "shortfall": 8},
            {"sku": "NUT-010", "name": "Nut", "available": 0, "reorderPoint": 5, "shortfall": 5}])

    def test_reorder_counts_open_orders(self):
        orderbook.place(self.state, ["BOL-006:10"])
        self.assertEqual(views.reorder_view(self.state)[0]["available"], 2)


if __name__ == "__main__":
    unittest.main()
