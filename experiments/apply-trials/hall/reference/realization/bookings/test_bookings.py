import unittest

from bookings import bookings
from bookings.reserve import RESERVE, bookable
from hall.errors import Refusal
from hall.store import empty
from rooms import rooms


class BookingTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()
        rooms.add_room(self.state, "A101", "10")
        rooms.add_room(self.state, "B202", "3")

    def test_reserve_is_within_the_model(self):
        self.assertIn(RESERVE, (0, 1, 2))
        self.assertEqual(bookable(10), 10 - RESERVE)

    def test_book_fills_exactly_the_bookable_seats(self):
        self.assertEqual(bookings.book(self.state, " a101", "1"), {"id": 1, "code": "A101", "seats": 1})
        self.assertEqual(bookings.book(self.state, "A101", str(bookable(10) - 1))["id"], 2)
        with self.assertRaises(Refusal):
            bookings.book(self.state, "A101", "1")
        self.assertEqual(len(self.state["bookings"]), 2)

    def test_other_rooms_do_not_count(self):
        bookings.book(self.state, "A101", str(bookable(10)))
        self.assertEqual(bookings.book(self.state, "B202", str(bookable(3)))["code"], "B202")

    def test_unknown_rooms_and_bad_seats_are_refused(self):
        for code, seats in (("C303", "1"), ("A10", "1"), ("A101", "0"), ("A101", "501"), ("A101", "x")):
            with self.subTest(code=code, seats=seats), self.assertRaises(Refusal):
                bookings.book(self.state, code, seats)
        self.assertEqual(self.state["bookings"], [])

    def test_listing_filters_by_room(self):
        bookings.book(self.state, "A101", "2")
        bookings.book(self.state, "B202", "1")
        self.assertEqual([entry["id"] for entry in bookings.listing(self.state)], [1, 2])
        self.assertEqual(bookings.listing(self.state, "b202"), [{"id": 2, "code": "B202", "seats": 1}])
        with self.assertRaises(Refusal):
            bookings.listing(self.state, "C303")


if __name__ == "__main__":
    unittest.main()
