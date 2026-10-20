"""Orders and the availability rule. The section is a list of {"id", "status", "lines"}."""
from catalog import items
from depot import inputs
from depot.errors import Refusal
from stock import ledger

STATUSES = ("open", "fulfilled", "cancelled")
MAX_LINES = 10


def reserved(state, sku):
    """Units of the item held by open orders."""
    return sum(line["qty"] for order in state["orders"] if order["status"] == "open"
               for line in order["lines"] if line["sku"] == sku)


def available(state, sku):
    return ledger.on_hand(state, sku) - reserved(state, sku)


def record(order):
    return {"id": order["id"], "status": order["status"],
            "lines": [{"sku": line["sku"], "qty": line["qty"]} for line in order["lines"]]}


def parse_lines(state, texts):
    if not 1 <= len(texts) <= MAX_LINES:
        raise Refusal("an order has from 1 to %d lines" % MAX_LINES)
    lines, seen = [], set()
    for text in texts:
        sku_text, colon, qty_text = text.partition(":")
        if not colon:
            raise Refusal("a line is SKU:QTY, got %r" % text)
        sku, quantity = items.require(state, sku_text), inputs.quantity(qty_text)
        if sku in seen:
            raise Refusal("%s appears in two lines" % sku)
        seen.add(sku)
        lines.append((sku, quantity))
    return lines


def place(state, texts):
    lines = parse_lines(state, texts)
    for sku, quantity in lines:
        if quantity > available(state, sku):
            raise Refusal("only %d of %s available" % (available(state, sku), sku))
    order = {"id": max((order["id"] for order in state["orders"]), default=0) + 1, "status": "open",
             "lines": [{"sku": sku, "qty": quantity} for sku, quantity in lines]}
    state["orders"].append(order)
    return record(order)


def find(state, text):
    wanted = inputs.order_id(text)
    for order in state["orders"]:
        if order["id"] == wanted:
            return order
    raise Refusal("no order %d" % wanted)


def fulfil(state, text):
    order = find(state, text)
    if order["status"] != "open":
        raise Refusal("order %d is %s, not open" % (order["id"], order["status"]))
    ledger.take(state, order["lines"])
    order["status"] = "fulfilled"
    return record(order)


def cancel(state, text):
    order = find(state, text)
    if order["status"] == "fulfilled":
        raise Refusal("order %d is fulfilled and cannot be cancelled" % order["id"])
    order["status"] = "cancelled"
    return record(order)


def listing(state, status=None):
    if status is not None and status not in STATUSES:
        raise Refusal("the status is one of %s, got %r" % (", ".join(STATUSES), status))
    return [record(order) for order in sorted(state["orders"], key=lambda order: order["id"])
            if status is None or order["status"] == status]


def validate(section, catalog):
    if not isinstance(section, list):
        raise ValueError("orders must be an array")
    ids = set()
    for order in section:
        if not isinstance(order, dict) or set(order) != {"id", "status", "lines"}:
            raise ValueError("invalid order")
        if type(order["id"]) is not int or order["id"] < 1 or order["id"] in ids or order["status"] not in STATUSES:
            raise ValueError("invalid order id or status")
        ids.add(order["id"])
        lines = order["lines"]
        if not isinstance(lines, list) or not 1 <= len(lines) <= MAX_LINES:
            raise ValueError("invalid lines of order %d" % order["id"])
        skus = [line.get("sku") for line in lines if isinstance(line, dict)]
        if len(set(skus)) != len(lines) or any(sku not in catalog for sku in skus):
            raise ValueError("invalid line items of order %d" % order["id"])
        for line in lines:
            if set(line) != {"sku", "qty"} or type(line["qty"]) is not int or not 1 <= line["qty"] <= inputs.MAX_QUANTITY:
                raise ValueError("invalid line of order %d" % order["id"])
