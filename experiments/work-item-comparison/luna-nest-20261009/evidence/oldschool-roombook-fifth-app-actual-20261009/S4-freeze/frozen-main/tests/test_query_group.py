import unittest

from roombook_query import filter_bookings


class BookingQueryTests(unittest.TestCase):
    def setUp(self):
        # Deliberately not sorted by timestamp or ID: the helper must retain
        # the caller's established list order.
        self.records = [
            {"id": 8, "room": "Atlas", "status": "canceled"},
            {"id": 3, "room": "Birch", "status": "active"},
            {"id": 6, "room": "Atlas", "status": "active"},
            {"id": 2, "room": "Atlas", "status": "canceled"},
        ]

    def test_status_and_trimmed_room_filters_compose(self):
        result = filter_bookings(self.records, room=" Atlas ", status="canceled")
        self.assertEqual([record["id"] for record in result], [8, 2])

        active = filter_bookings(self.records, room="Atlas", status="active")
        self.assertEqual([record["id"] for record in active], [6])

    def test_unknown_or_case_mismatched_room_returns_no_records(self):
        self.assertEqual(filter_bookings(self.records, room="Missing"), [])
        self.assertEqual(filter_bookings(self.records, room="atlas"), [])

    def test_filtering_and_unfiltered_results_preserve_input_order(self):
        self.assertEqual(
            [record["id"] for record in filter_bookings(self.records)],
            [8, 3, 6, 2],
        )
        self.assertEqual(
            [record["id"] for record in filter_bookings(self.records, status="active")],
            [3, 6],
        )

    def test_invalid_status_fails_clearly_even_for_empty_input(self):
        with self.assertRaisesRegex(ValueError, "status must be 'active' or 'canceled'"):
            filter_bookings([], status="pending")


if __name__ == "__main__":
    unittest.main()
