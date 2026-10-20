"""On-hand quantities. The section is {SKU: onHand}; an item without an entry has 0 on hand."""
from catalog import items
from depot import inputs
from depot.errors import Refusal

MAX_ON_HAND = 9999


def on_hand(state, sku):
    return state["stock"].get(sku, 0)


def record(state, sku):
    return {"sku": sku, "onHand": on_hand(state, sku)}


def receive(state, sku, qty):
    key, quantity = items.require(state, sku), inputs.quantity(qty)
    if on_hand(state, key) + quantity > MAX_ON_HAND:
        raise Refusal("receiving %d would raise %s above %d on hand" % (quantity, key, MAX_ON_HAND))
    state["stock"][key] = on_hand(state, key) + quantity
    return record(state, key)


def take(state, lines):
    """Take the lines of a fulfilled order out of stock."""
    for line in lines:
        if line["qty"] > on_hand(state, line["sku"]):
            raise Refusal("only %d of %s on hand" % (on_hand(state, line["sku"]), line["sku"]))
    for line in lines:
        state["stock"][line["sku"]] = on_hand(state, line["sku"]) - line["qty"]


def validate(section, catalog):
    if not isinstance(section, dict):
        raise ValueError("stock must be an object")
    for sku, held in section.items():
        if sku not in catalog or type(held) is not int or not 0 <= held <= MAX_ON_HAND:
            raise ValueError("invalid stock entry %r" % sku)
