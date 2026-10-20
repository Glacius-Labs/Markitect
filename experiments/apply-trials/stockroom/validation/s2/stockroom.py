"""Stock rules of model 0.2.0 (oracle validation only, not a delivered realization)."""
import re

MAX_QUANTITY, MAX_ON_HAND, MAX_NAME_LENGTH, MAX_REORDER = 500, 9999, 60, 9999
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


def parse_number(text, low, high, label):
    if not DIGITS.fullmatch(text) or len(text.lstrip("0")) > 9 or not low <= int(text) <= high:
        raise StockError("the %s must be a whole number from %d to %d" % (label, low, high))
    return int(text)


def record(data, sku):
    item = data[sku]
    return {"sku": sku, "name": item["name"], "onHand": item["onHand"], "reorderPoint": item["reorderPoint"]}


def existing(data, sku):
    if sku not in data:
        raise StockError("no item with SKU %s" % sku)
    return data[sku]


def add_item(data, sku, name, reorder):
    key, label = normalize_sku(sku), parse_name(name)
    point = parse_number(reorder, 0, MAX_REORDER, "reorder point")
    if key in data:
        raise StockError("an item with SKU %s already exists" % key)
    data[key] = {"name": label, "onHand": 0, "reorderPoint": point}
    return record(data, key)


def receive(data, sku, qty):
    key = normalize_sku(sku)
    item, quantity = existing(data, key), parse_number(qty, 1, MAX_QUANTITY, "quantity")
    if item["onHand"] + quantity > MAX_ON_HAND:
        raise StockError("too much stock")
    item["onHand"] += quantity
    return record(data, key)


def issue(data, sku, qty):
    key = normalize_sku(sku)
    item, quantity = existing(data, key), parse_number(qty, 1, MAX_QUANTITY, "quantity")
    if quantity > item["onHand"]:
        raise StockError("not enough stock")
    item["onHand"] -= quantity
    return record(data, key)


def list_items(data, low=False):
    return [record(data, key) for key in sorted(data) if not low or data[key]["onHand"] < data[key]["reorderPoint"]]
