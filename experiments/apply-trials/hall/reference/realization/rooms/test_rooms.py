import unittest

from bookings.reserve import RESERVE
from hall.errors import Refusal
from hall.store import empty
from rooms import rooms


class RoomTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()

    def test_codes_are_normalized(self):
        self.assertEqual(rooms.normalize_code(" a101 "), "A101")
        for text in ("A10", "AB101", "1101", "A1011", "A-101", "", "Ä101", "A10１"):
            with self.subTest(text=text), self.assertRaises(Refusal):
                rooms.normalize_code(text)

    def test_add_room_returns_the_record_with_bookable_seats(self):
        self.assertEqual(rooms.add_room(self.state, " b202", "40"),
                         {"code": "B202", "seats": 40, "bookable": 40 - RESERVE})

    def test_seats_and_duplicates_are_refused(self):
        for seats in ("2", "501", "0", "-3", "+3", "3.0", " 5", ""):
            with self.subTest(seats=seats), self.assertRaises(Refusal):
                rooms.add_room(self.state, "A101", seats)
        rooms.add_room(self.state, "A101", "003")
        with self.assertRaises(Refusal):
            rooms.add_room(self.state, "a101", "10")
        self.assertEqual(self.state["rooms"], {"A101": 3})

    def test_listing_is_sorted_by_code(self):
        rooms.add_room(self.state, "C300", "10")
        rooms.add_room(self.state, "A100", "500")
        self.assertEqual([entry["code"] for entry in rooms.listing(self.state)], ["A100", "C300"])


if __name__ == "__main__":
    unittest.main()
