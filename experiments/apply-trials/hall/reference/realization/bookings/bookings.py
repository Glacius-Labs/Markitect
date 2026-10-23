"""Bookings: booking seats of a room within its bookable seats, and the listing."""
from bookings.reserve import bookable
from hall import inputs
from hall.errors import Refusal
from rooms import rooms


def booked(state, code):
    return sum(entry["seats"] for entry in state["bookings"] if entry["code"] == code)


def book(state, code_text, seats_text):
    code, seats = rooms.existing(state, code_text), inputs.booking_seats(seats_text)
    free = bookable(state["rooms"][code]) - booked(state, code)
    if seats > free:
        raise Refusal("only %d bookable seats of room %s are free" % (max(free, 0), code))
    entry = {"id": max((item["id"] for item in state["bookings"]), default=0) + 1, "code": code, "seats": seats}
    state["bookings"].append(entry)
    return dict(entry)


def listing(state, code_text=None):
    code = None if code_text is None else rooms.existing(state, code_text)
    found = [dict(entry) for entry in state["bookings"] if code is None or entry["code"] == code]
    return sorted(found, key=lambda entry: entry["id"])


def validate(section, room_section):
    if not isinstance(section, list):
        raise ValueError("bookings must be an array")
    seen = set()
    for entry in section:
        if (not isinstance(entry, dict) or set(entry) != {"id", "code", "seats"} or type(entry["id"]) is not int
                or entry["id"] < 1 or entry["id"] in seen or entry["code"] not in room_section
                or type(entry["seats"]) is not int or not 1 <= entry["seats"] <= inputs.MAX_SEATS):
            raise ValueError("invalid booking %r" % (entry,))
        seen.add(entry["id"])
    for code, seats in room_section.items():
        if sum(entry["seats"] for entry in section if entry["code"] == code) > bookable(seats):
            raise ValueError("the bookings of room %s exceed its bookable seats" % code)
