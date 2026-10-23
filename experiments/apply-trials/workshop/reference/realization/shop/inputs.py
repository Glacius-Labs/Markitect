"""Shared input rules: texts, SKUs, numbers, prices, quantities and labor durations."""
import re

from shop.errors import Refusal

DIGITS = re.compile(r"[0-9]+")
SKU = re.compile(r"[A-Z]{3}-[0-9]{3}")
MAX_TEXT = 60


def text(value, label):
    value = value.strip()
    if not value:
        raise Refusal("%s must not be empty" % label)
    if len(value) > MAX_TEXT:
        raise Refusal("%s must have at most %d characters" % (label, MAX_TEXT))
    if any(0xD800 <= ord(char) <= 0xDFFF for char in value):
        raise Refusal("%s must be valid Unicode text" % label)
    return value


def sku(value):
    normalized = value.strip().upper()
    if not SKU.fullmatch(normalized):
        raise Refusal("a SKU must be three letters, a hyphen and three digits, such as BRK-101, got %r" % value)
    return normalized


def number(value, low, high, label):
    if not DIGITS.fullmatch(value):
        raise Refusal("%s must be a whole number written in digits, got %r" % (label, value))
    digits = value.lstrip("0") or "0"
    if len(digits) > 18 or not low <= int(digits) <= high:
        raise Refusal("%s must be from %d to %d, got %s" % (label, low, high, value))
    return int(digits)


def price(value):
    return number(value, 1, 1000000, "a price in cents")


def quantity(value):
    return number(value, 1, 500, "a quantity")


def minutes(value):
    result = number(value, 15, 480, "a labor duration in minutes")
    if result % 15:
        raise Refusal("a labor duration must be a multiple of 15 minutes, got %d" % result)
    return result
