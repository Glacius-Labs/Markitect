"""Stock rules under the overspecified model (oracle validation only, not a delivered realization)."""
import re

from limits import MAX_NAME_LENGTH, MAX_ON_HAND, MAX_QUANTITY

SKU_FORMAT = re.compile(r"[A-Z]{3}-[0-9]{3}")
DIGITS = re.compile(r"[0-9]+")


class StockError(Exception):
    """A refused command: invalid input or a violated stock rule."""


def normalize_sku(text):
    sku = text.strip().upper()
    if not SKU_FORMAT.fullmatch(sku):
        raise StockError("invalid SKU %r" % text)
    return sku


def parse_name(text):
    name = text.strip()
    if not name or len(name) > MAX_NAME_LENGTH or any(0xD800 <= ord(char) <= 0xDFFF for char in name):
        raise StockError("invalid name")
    return name


def parse_quantity(text):
    if not DIGITS.fullmatch(text) or len(text.lstrip("0")) > 9 or not 1 <= int(text) <= MAX_QUANTITY:
        raise StockError("invalid quantity %r" % text)
    return int(text)


def record(data, sku):
    return {"sku": sku, "name": data[sku]["name"], "onHand": data[sku]["onHand"]}


def existing(data, sku):
    if sku not in data:
        raise StockError("unknown SKU: %s" % sku)
    return data[sku]


def add_item(data, sku, name):
    key, label = normalize_sku(sku), parse_name(name)
    if key in data:
        raise StockError("an item with SKU %s already exists" % key)
    data[key] = {"name": label, "onHand": 0}
    return record(data, key)


def receive(data, sku, qty):
    key = normalize_sku(sku)
    item = existing(data, key)
    quantity = parse_quantity(qty)
    if item["onHand"] + quantity > MAX_ON_HAND:
        raise StockError("too much stock")
    item["onHand"] += quantity
    return record(data, key)


def issue(data, sku, qty):
    key = normalize_sku(sku)
    item = existing(data, key)
    quantity = parse_quantity(qty)
    if quantity > item["onHand"]:
        raise StockError("not enough stock")
    item["onHand"] -= quantity
    return record(data, key)


def list_items(data):
    return [record(data, key) for key in sorted(data)]
