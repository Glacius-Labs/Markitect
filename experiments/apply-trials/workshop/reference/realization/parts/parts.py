"""Parts and their stock: on hand, reserved by open jobs, and available."""
from shop import inputs
from shop.errors import Refusal

MAX_ON_HAND = 9999


def existing(state, sku_text):
    key = inputs.sku(sku_text)
    if key not in state["parts"]:
        raise Refusal("there is no part %s" % key)
    return key


def record(state, key):
    part = state["parts"][key]
    return {"sku": key, "name": part["name"], "price": part["price"], "onHand": part["onHand"],
            "reserved": part["reserved"], "available": part["onHand"] - part["reserved"]}


def add_part(state, sku_text, name_text, price_text):
    key, name, price = inputs.sku(sku_text), inputs.text(name_text, "a part's name"), inputs.price(price_text)
    if key in state["parts"]:
        raise Refusal("the part %s already exists" % key)
    state["parts"][key] = {"name": name, "price": price, "onHand": 0, "reserved": 0}
    return record(state, key)


def receive(state, sku_text, qty_text):
    key, quantity = existing(state, sku_text), inputs.quantity(qty_text)
    part = state["parts"][key]
    if part["onHand"] + quantity > MAX_ON_HAND:
        raise Refusal("receiving %d would raise %s above %d on hand" % (quantity, key, MAX_ON_HAND))
    part["onHand"] += quantity
    return record(state, key)


def reserve(state, key, quantity):
    """Reserve units for an open job; the caller has checked the job."""
    part = state["parts"][key]
    if quantity > part["onHand"] - part["reserved"]:
        raise Refusal("only %d units of %s are available" % (part["onHand"] - part["reserved"], key))
    part["reserved"] += quantity


def consume(state, key, quantity):
    """Take reserved units out of stock when their job closes."""
    part = state["parts"][key]
    part["onHand"] -= quantity
    part["reserved"] -= quantity


def price(state, key):
    return state["parts"][key]["price"]


def available(state, key):
    part = state["parts"][key]
    return part["onHand"] - part["reserved"]


def listing(state):
    return [record(state, key) for key in sorted(state["parts"])]


def validate(state):
    section = state["parts"]
    if not isinstance(section, dict):
        raise ValueError("parts must be an object")
    for key, part in section.items():
        if (inputs.sku(key) != key or not isinstance(part, dict)
                or set(part) != {"name", "price", "onHand", "reserved"}
                or not isinstance(part["name"], str) or inputs.text(part["name"], "a name") != part["name"]
                or type(part["price"]) is not int or not 1 <= part["price"] <= 1000000
                or type(part["onHand"]) is not int or type(part["reserved"]) is not int
                or not 0 <= part["reserved"] <= part["onHand"] <= MAX_ON_HAND):
            raise ValueError("invalid part %r" % key)
