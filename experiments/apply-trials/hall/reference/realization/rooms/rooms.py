"""Rooms: the room code rule, registration and the listing. The section is {code: seats}."""
import re

from bookings.reserve import bookable
from hall import inputs
from hall.errors import Refusal

CODE = re.compile(r"[A-Z][0-9]{3}")


def normalize_code(text):
    """The room code rule, used by every command that takes a code."""
    code = text.strip().upper()
    if not CODE.fullmatch(code):
        raise Refusal("a room code must be one letter A-Z and three digits, such as A101, got %r" % text)
    return code


def existing(state, text):
    """The normalized code of an existing room."""
    code = normalize_code(text)
    if code not in state["rooms"]:
        raise Refusal("there is no room %s" % code)
    return code


def record(state, code):
    seats = state["rooms"][code]
    return {"code": code, "seats": seats, "bookable": bookable(seats)}


def add_room(state, code_text, seats_text):
    code, seats = normalize_code(code_text), inputs.room_seats(seats_text)
    if code in state["rooms"]:
        raise Refusal("the room %s already exists" % code)
    state["rooms"][code] = seats
    return record(state, code)


def listing(state):
    return [record(state, code) for code in sorted(state["rooms"])]


def validate(section):
    if not isinstance(section, dict):
        raise ValueError("rooms must be an object")
    for code, seats in section.items():
        if not CODE.fullmatch(code) or type(seats) is not int or not inputs.MIN_ROOM_SEATS <= seats <= inputs.MAX_SEATS:
            raise ValueError("invalid room %r" % code)
