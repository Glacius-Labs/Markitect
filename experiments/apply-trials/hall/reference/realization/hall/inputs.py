"""Shared input rules: numbers and the seats of rooms and bookings."""
import re

from hall.errors import Refusal

DIGITS = re.compile(r"[0-9]+")
MIN_ROOM_SEATS = 3
MAX_SEATS = 500


def number(text, low, high, label):
    if not DIGITS.fullmatch(text):
        raise Refusal("%s must be a whole number written in digits, got %r" % (label, text))
    digits = text.lstrip("0") or "0"  # leading zeros do not change the value
    if len(digits) > 18:
        raise Refusal("%s is too large" % label)
    value = int(digits)
    if not low <= value <= high:
        raise Refusal("%s must be from %d to %d, got %d" % (label, low, high, value))
    return value


def room_seats(text):
    return number(text, MIN_ROOM_SEATS, MAX_SEATS, "the seats of a room")


def booking_seats(text):
    return number(text, 1, MAX_SEATS, "the seats of a booking")
