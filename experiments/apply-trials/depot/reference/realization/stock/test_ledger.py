import unittest

from catalog import items
from depot.errors import Refusal
from depot.store import empty
from stock import ledger


class LedgerTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()
        items.add_item(self.state, "BOL-006", "Bolt", "0")

    def test_new_item_has_nothing_on_hand(self):
        self.assertEqual(ledger.record(self.state, "BOL-006"), {"sku": "BOL-006", "onHand": 0})

    def test_receive_adds(self):
        ledger.receive(self.state, "BOL-006", "40")
        self.assertEqual(ledger.receive(self.state, "bol-006", "2"), {"sku": "BOL-006", "onHand": 42})

    def test_receive_checks_the_item_and_quantity(self):
        for sku, qty in (("NUT-010", "1"), ("BOL-006", "0"), ("BOL-006", "501"), ("BOL-006", "-1")):
            with self.subTest(sku=sku, qty=qty), self.assertRaises(Refusal):
                ledger.receive(self.state, sku, qty)
        self.assertEqual(ledger.on_hand(self.state, "BOL-006"), 0)

    def test_receive_stops_at_the_limit(self):
        for _ in range(19):
            ledger.receive(self.state, "BOL-006", "500")
        self.assertEqual(ledger.receive(self.state, "BOL-006", "499")["onHand"], 9999)
        with self.assertRaises(Refusal):
            ledger.receive(self.state, "BOL-006", "1")

    def test_take_removes_fulfilled_lines(self):
        ledger.receive(self.state, "BOL-006", "10")
        ledger.take(self.state, [{"sku": "BOL-006", "qty": 4}])
        self.assertEqual(ledger.on_hand(self.state, "BOL-006"), 6)


if __name__ == "__main__":
    unittest.main()
