import unittest

from roombook_summary import summarize


def booking(identifier, room, start, end, status="active"):
    return {
        "id": identifier,
        "room": room,
        "start": start,
        "end": end,
        "title": f"Booking {identifier}",
        "status": status,
    }


class RoomSummaryTests(unittest.TestCase):
    def test_rooms_are_in_lexical_order(self):
        state = {"bookings": [
            booking(1, "Zeta", "2026-10-09T08:00Z", "2026-10-09T09:00Z"),
            booking(2, "Alpha", "2026-10-09T08:00Z", "2026-10-09T08:30Z"),
            booking(3, "Middle", "2026-10-09T08:00Z", "2026-10-09T08:15Z"),
        ]}

        self.assertEqual(summarize(state), {"rooms": [
            {"room": "Alpha", "active": 1, "minutes": 30},
            {"room": "Middle", "active": 1, "minutes": 15},
            {"room": "Zeta", "active": 1, "minutes": 60},
        ]})

    def test_room_filter_trims_whitespace_and_unknown_room_is_empty(self):
        state = {"bookings": [
            booking(1, "Studio", "2026-10-09T08:00Z", "2026-10-09T09:00Z"),
            booking(2, "studio", "2026-10-09T08:00Z", "2026-10-09T08:10Z"),
        ]}

        self.assertEqual(summarize(state, "  Studio \t"), {"rooms": [
            {"room": "Studio", "active": 1, "minutes": 60},
        ]})
        self.assertEqual(summarize(state, "Missing"), {"rooms": []})
        self.assertEqual(summarize(state, "studio"), {"rooms": [
            {"room": "studio", "active": 1, "minutes": 10},
        ]})

    def test_canceled_bookings_do_not_affect_active_count_or_minutes(self):
        state = {"bookings": [
            booking(1, "Studio", "2026-10-09T08:00Z", "2026-10-09T08:25Z", "canceled"),
            booking(2, "Studio", "2026-10-09T09:00Z", "2026-10-09T09:12Z"),
        ]}

        self.assertEqual(summarize(state), {"rooms": [
            {"room": "Studio", "active": 1, "minutes": 12},
        ]})

    def test_adjacent_bookings_are_counted_and_minutes_are_summed_exactly(self):
        state = {"bookings": [
            booking(1, "Studio", "2026-10-09T23:40Z", "2026-10-10T00:00Z"),
            booking(2, "Studio", "2026-10-10T00:00Z", "2026-10-10T00:07Z"),
        ]}

        self.assertEqual(summarize(state), {"rooms": [
            {"room": "Studio", "active": 2, "minutes": 27},
        ]})


if __name__ == "__main__":
    unittest.main()
