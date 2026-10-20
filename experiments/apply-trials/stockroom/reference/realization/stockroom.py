"""Stock rules: SKU, name and quantity inputs, items and their on-hand limits.

The data is a dict from normalized SKU to {"name": str, "onHand": int}. Every
function validates completely before it changes the data, so a refused command
leaves it untouched.
"""
import json
import re
from pathlib import Path

LIMITS = json.loads(Path(__file__).with_name("limits.json").read_text(encoding="utf-8"))
SKU_FORMAT = re.compile(r"[A-Z]{3}-[0-9]{3}")
DIGITS = re.compile(r"[0-9]+")


class StockError(Exception):
    """A refused command: invalid input or a violated stock rule."""


def normalize_sku(text):
    sku = text.strip().upper()
    if not SKU_FORMAT.fullmatch(sku):
        raise StockError("invalid SKU %r: expected three letters, a hyphen and three digits" % text)
    return sku


def parse_name(text):
    name = text.strip()
    if not name:
        raise StockError("the name must not be empty")
    if len(name) > LIMITS["maxNameLength"]:
        raise StockError("the name must have at most %d characters" % LIMITS["maxNameLength"])
    return name


def parse_quantity(text):
    if not DIGITS.fullmatch(text):
        raise StockError("invalid quantity %r: expected a whole number written in digits" % text)
    quantity = int(text)
    if not 1 <= quantity <= LIMITS["maxQuantity"]:
        raise StockError("the quantity must be from 1 to %d" % LIMITS["maxQuantity"])
    return quantity


def record(data, sku):
    return {"sku": sku, "name": data[sku]["name"], "onHand": data[sku]["onHand"]}


def existing(data, sku):
    if sku not in data:
        raise StockError("no item with SKU %s" % sku)
    return data[sku]


def add_item(data, sku, name):
    key, label = normalize_sku(sku), parse_name(name)
    if key in data:
        raise StockError("an item with SKU %s already exists" % key)
    data[key] = {"name": label, "onHand": 0}
    return record(data, key)


def receive(data, sku, qty):
    key = normalize_sku(sku)
    item, quantity = existing(data, key), parse_quantity(qty)
    if item["onHand"] + quantity > LIMITS["maxOnHand"]:
        raise StockError("receiving %d would raise %s above %d on hand" % (quantity, key, LIMITS["maxOnHand"]))
    item["onHand"] += quantity
    return record(data, key)


def issue(data, sku, qty):
    key = normalize_sku(sku)
    item, quantity = existing(data, key), parse_quantity(qty)
    if quantity > item["onHand"]:
        raise StockError("cannot issue %d of %s: only %d on hand" % (quantity, key, item["onHand"]))
    item["onHand"] -= quantity
    return record(data, key)


def list_items(data):
    return [record(data, key) for key in sorted(data)]
