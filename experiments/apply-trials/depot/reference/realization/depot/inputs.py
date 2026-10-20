"""Shared input rules: numbers, quantities, reorder points, order ids and names."""
import re

from depot.errors import Refusal

DIGITS = re.compile(r"[0-9]+")
MAX_QUANTITY = 500
MAX_REORDER_POINT = 9999
MAX_NAME_LENGTH = 60


def number(text, low, high, label):
    if not DIGITS.fullmatch(text):
        raise Refusal("%s must be a whole number written in digits, got %r" % (label, text))
    value = int(text)
    if value < low or (high is not None and value > high):
        bound = "from %d to %d" % (low, high) if high is not None else "at least %d" % low
        raise Refusal("%s must be %s, got %d" % (label, bound, value))
    return value


def quantity(text):
    return number(text, 1, MAX_QUANTITY, "a quantity")


def reorder_point(text):
    return number(text, 0, MAX_REORDER_POINT, "a reorder point")


def order_id(text):
    return number(text, 1, None, "an order id")


def name(text):
    value = text.strip()
    if not value:
        raise Refusal("the name must not be empty")
    if len(value) > MAX_NAME_LENGTH:
        raise Refusal("the name must have at most %d characters" % MAX_NAME_LENGTH)
    return value
