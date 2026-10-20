"""Read-only views over catalog, stock and orders."""
from catalog import items
from orders import orderbook
from stock import ledger


def stock_view(state):
    return [dict(items.record(state, sku), onHand=ledger.on_hand(state, sku),
                 reserved=orderbook.reserved(state, sku), available=orderbook.available(state, sku))
            for sku in items.skus(state)]


def reorder_view(state):
    entries = []
    for sku in items.skus(state):
        item = items.record(state, sku)
        available = orderbook.available(state, sku)
        if available < item["reorderPoint"]:
            entries.append({"sku": sku, "name": item["name"], "available": available,
                            "reorderPoint": item["reorderPoint"], "shortfall": item["reorderPoint"] - available})
    return sorted(entries, key=lambda entry: (-entry["shortfall"], entry["sku"]))
