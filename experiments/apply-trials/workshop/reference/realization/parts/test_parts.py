import unittest

from parts import parts
from shop.errors import Refusal
from shop.store import empty


class PartTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()

    def test_add_and_receive(self):
        self.assertEqual(parts.add_part(self.state, " brk-101 ", "Brake pad", "0125"),
                         {"sku": "BRK-101", "name": "Brake pad", "price": 125, "onHand": 0, "reserved": 0,
                          "available": 0})
        for _ in range(19):
            parts.receive(self.state, "BRK-101", "500")
        self.assertEqual(parts.receive(self.state, "BRK-101", "499")["onHand"], 9999)
        with self.assertRaises(Refusal):
            parts.receive(self.state, "BRK-101", "1")

    def test_refusals(self):
        parts.add_part(self.state, "BRK-101", "Brake pad", "125")
        for args in (("brk-101", "Other", "1"), ("BR-101", "X", "1"), ("BRK-102", "", "1"), ("BRK-102", "X", "0"),
                     ("BRK-102", "X", "1000001")):
            with self.subTest(args=args), self.assertRaises(Refusal):
                parts.add_part(self.state, *args)


if __name__ == "__main__":
    unittest.main()
