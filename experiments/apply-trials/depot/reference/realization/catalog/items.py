"""Catalog items and the SKU rule. The section is {SKU: {"name": str, "reorderPoint": int}}."""
import re

from depot import inputs
from depot.errors import Refusal

SKU_FORMAT = re.compile(r"[A-Z]{3}-[0-9]{3}")


def normalize_sku(text):
    sku = text.strip().upper()
    if not SKU_FORMAT.fullmatch(sku):
        raise Refusal("invalid SKU %r: expected three letters, a hyphen and three digits" % text)
    return sku


def require(state, text):
    """The normalized SKU of an existing item."""
    sku = normalize_sku(text)
    if sku not in state["items"]:
        raise Refusal("no item with SKU %s" % sku)
    return sku


def skus(state):
    return sorted(state["items"])


def record(state, sku):
    item = state["items"][sku]
    return {"sku": sku, "name": item["name"], "reorderPoint": item["reorderPoint"]}


def add_item(state, sku, name, reorder):
    key, label, point = normalize_sku(sku), inputs.name(name), inputs.reorder_point(reorder)
    if key in state["items"]:
        raise Refusal("an item with SKU %s already exists" % key)
    state["items"][key] = {"name": label, "reorderPoint": point}
    return record(state, key)


def validate(section):
    if not isinstance(section, dict):
        raise ValueError("items must be an object")
    for sku, item in section.items():
        if normalize_sku(sku) != sku or not isinstance(item, dict) or set(item) != {"name", "reorderPoint"}:
            raise ValueError("invalid item %r" % sku)
        if not isinstance(item["name"], str) or inputs.name(item["name"]) != item["name"]:
            raise ValueError("invalid name of %s" % sku)
        if type(item["reorderPoint"]) is not int or not 0 <= item["reorderPoint"] <= inputs.MAX_REORDER_POINT:
            raise ValueError("invalid reorder point of %s" % sku)
