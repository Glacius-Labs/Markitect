"""Depot checks: black-box expectations of snapshot d1.

Each check cites the Definitions of every Package whose rule it observes, so a
failure points at the responsibilities involved. Error wording and the
organization of the data file are free; only the envelope, exit code,
unchanged data and the specified records are checked.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from trial_kit import (Checks, ascii_output, expect_equal, fail, no_crash, unusable_change,  # noqa: E402
                       unusable_query)

CHECKS = Checks("depot", {"d1": {"snapshot": "d1"}})
D1 = ["d1"]
check = CHECKS.check


def add(env, sku, name="Part", reorder=None, refuse=False):
    args = ["add-item", "--sku", sku, "--name", name] + ([] if reorder is None else ["--reorder", str(reorder)])
    return env.refuse(*args) if refuse else env.ok(*args)


def receive(env, sku, qty, refuse=False):
    args = ["receive", "--sku", sku, "--qty", str(qty)]
    return env.refuse(*args) if refuse else env.ok(*args)


def place(env, *lines, refuse=False):
    args = ["place-order"] + [part for line in lines for part in ("--line", line)]
    return env.refuse(*args) if refuse else env.ok(*args)


def by_id(env, command, order_id, refuse=False):
    args = [command, "--id", str(order_id)]
    return env.refuse(*args) if refuse else env.ok(*args)


def collection(env, command, field, *extra):
    result = env.ok(command, *extra)
    if set(result) != {field} or not isinstance(result[field], list):
        fail("%s must return exactly {\"%s\": [...]}, got %s" % (command, field, result))
    return result[field]


def items(env):
    return collection(env, "items", "items")


def reorder(env):
    return collection(env, "reorder", "items")


def orders(env, *extra):
    return collection(env, "orders", "orders", *extra)


def order(order_id, status, *lines):
    return {"id": order_id, "status": status, "lines": [{"sku": sku, "qty": qty} for sku, qty in lines]}


def stocked(env, sku, held, reorder_point=0):
    add(env, sku, "Part " + sku, reorder_point)
    if held:
        receive(env, sku, held)


def figures(env, sku):
    entry = next((entry for entry in items(env) if entry.get("sku") == sku), None)
    if entry is None:
        fail("items has no entry for %s" % sku)
    return entry["onHand"], entry["reserved"], entry["available"]


@check("empty-state", D1, ["depot/storage", "reports/stock-view", "reports/reorder", "orders/list-orders"],
       "A missing data file is an empty depot for every query")
def _(env, p):
    expect_equal(items(env), [], "items")
    expect_equal(orders(env), [], "orders")
    expect_equal(reorder(env), [], "reorder")


@check("add-item-record", D1, ["catalog/add-item", "catalog/item", "catalog/sku", "depot/inputs"],
       "add-item returns the catalog record with normalized SKU and name and the reorder point")
def _(env, p):
    expect_equal(add(env, " bol-006 ", "  Bolt M6 ", "0007"), {"sku": "BOL-006", "name": "Bolt M6", "reorderPoint": 7},
                 "added record")
    expect_equal(add(env, "NUT-010", "Nut"), {"sku": "NUT-010", "name": "Nut", "reorderPoint": 0}, "default point")


@check("item-inputs", D1, ["catalog/sku", "catalog/add-item", "depot/inputs"],
       "Malformed or duplicate SKUs, invalid names and reorder points are refused")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    for sku in ("BO-006", "BOLT-006", "BOL-06", "BOL006", "B0L-006", "BOL_006", "ÄOL-006", "", " bol-006 "):
        add(env, sku, "Other", refuse=True)
    for name in ("", "   ", "x" * 61):
        add(env, "NUT-010", name, refuse=True)
    for point in ("-1", "10000", "x", "1.5", "", " 3"):
        add(env, "NUT-010", "Nut", point, refuse=True)
    expect_equal(add(env, "NUT-010", "é" * 60, "9999")["reorderPoint"], 9999, "60 code points, point 9999")


@check("receive-record", D1, ["stock/receive", "stock/on-hand", "catalog/sku", "depot/inputs"],
       "receive adds to the item found by its normalized SKU and returns the stock record")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    expect_equal(receive(env, " bol-006", 500), {"sku": "BOL-006", "onHand": 500}, "receive 500")
    expect_equal(receive(env, "BOL-006", "0007"), {"sku": "BOL-006", "onHand": 507}, "receive 0007")
    for qty in ("0", "501", "-1", "+1", "1.0", " 5", "", "５"):
        receive(env, "BOL-006", qty, refuse=True)
    receive(env, "XYZ-999", 1, refuse=True)
    receive(env, "BOL_006", 1, refuse=True)


@check("receive-cap", D1, ["stock/limits", "stock/receive"], "On hand may reach exactly 9999, never more")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    for _ in range(19):
        receive(env, "BOL-006", 500)
    expect_equal(receive(env, "BOL-006", 499)["onHand"], 9999, "on hand at the limit")
    receive(env, "BOL-006", 1, refuse=True)
    expect_equal(figures(env, "BOL-006")[0], 9999, "on hand after the refused receipt")


@check("place-order-record", D1, ["orders/place-order", "orders/order", "catalog/sku"],
       "place-order returns the open order with normalized lines in the given order and increasing ids")
def _(env, p):
    stocked(env, "BOL-006", 10)
    stocked(env, "NUT-010", 10)
    expect_equal(place(env, " nut-010 :3", "bol-006:0004"), order(1, "open", ("NUT-010", 3), ("BOL-006", 4)),
                 "first order")
    expect_equal(place(env, "BOL-006:1")["id"], 2, "second id")
    expect_equal(orders(env), [order(1, "open", ("NUT-010", 3), ("BOL-006", 4)), order(2, "open", ("BOL-006", 1))],
                 "orders")


@check("place-order-lines", D1, ["orders/place-order", "orders/order", "catalog/sku", "depot/inputs"],
       "Malformed, unknown, duplicate, missing or too many lines refuse the whole order")
def _(env, p):
    for sku in ("BOL-006", "NUT-010"):
        stocked(env, sku, 50)
    for lines in (["BOL-006"], ["BOL-006:"], [":5"], ["BOL-006:0"], ["BOL-006:501"], ["BOL-006:x"], ["BOL-006:5:1"],
                  ["XYZ-999:1"], ["BOL-006:1", " bol-006:2"], ["NUT-010:1", "BOL-006 1"]):
        place(env, *lines, refuse=True)
    env.refuse("place-order")
    many = ["%s-%03d:1" % ("ITM", number) for number in range(1, 12)]
    for line in many:
        stocked(env, line.split(":")[0], 1)
    place(env, *many, refuse=True)
    expect_equal(place(env, *many[:10])["id"], 1, "ten lines are allowed and the refusals used no id")


@check("place-order-availability", D1, ["orders/availability", "orders/place-order", "stock/on-hand"],
       "A line may take at most the available quantity: on hand minus what open orders reserve")
def _(env, p):
    stocked(env, "BOL-006", 10)
    place(env, "BOL-006:6")
    place(env, "BOL-006:5", refuse=True)
    expect_equal(place(env, "BOL-006:4")["id"], 2, "exactly the available quantity")
    place(env, "BOL-006:1", refuse=True)
    stocked(env, "NUT-010", 0)
    place(env, "NUT-010:1", refuse=True)


@check("place-order-atomic", D1, ["orders/place-order", "orders/availability", "depot/storage"],
       "When one line does not fit, no order is created and nothing changes")
def _(env, p):
    stocked(env, "BOL-006", 10)
    stocked(env, "NUT-010", 2)
    place(env, "BOL-006:5", "NUT-010:3", refuse=True)
    place(env, "NUT-010:3", "BOL-006:5", refuse=True)
    expect_equal(orders(env), [], "orders after refusals")
    expect_equal(figures(env, "BOL-006"), (10, 0, 10), "BOL-006 figures")
    expect_equal(place(env, "BOL-006:5", "NUT-010:2")["id"], 1, "the next order gets id 1")


@check("fulfil-order", D1, ["orders/fulfil-order", "stock/on-hand", "orders/availability", "depot/inputs"],
       "Fulfilling takes the lines out of stock once and releases the reservation")
def _(env, p):
    stocked(env, "BOL-006", 10)
    stocked(env, "NUT-010", 5)
    place(env, "BOL-006:4", "NUT-010:5")
    expect_equal(by_id(env, "fulfil-order", "01"), order(1, "fulfilled", ("BOL-006", 4), ("NUT-010", 5)), "fulfilled")
    expect_equal(figures(env, "BOL-006"), (6, 0, 6), "BOL-006 after fulfilment")
    expect_equal(figures(env, "NUT-010"), (0, 0, 0), "NUT-010 after fulfilment")
    by_id(env, "fulfil-order", 1, refuse=True)
    by_id(env, "cancel-order", 1, refuse=True)
    for bad in ("0", "-1", "x", "", "2"):
        by_id(env, "fulfil-order", bad, refuse=True)


@check("cancel-order", D1, ["orders/cancel-order", "orders/availability", "stock/on-hand"],
       "Cancelling releases the reservation without touching stock; repeating it changes nothing")
def _(env, p):
    stocked(env, "BOL-006", 10)
    place(env, "BOL-006:7")
    expect_equal(by_id(env, "cancel-order", 1), order(1, "cancelled", ("BOL-006", 7)), "cancelled")
    expect_equal(figures(env, "BOL-006"), (10, 0, 10), "after cancelling")
    expect_equal(by_id(env, "cancel-order", "001"), order(1, "cancelled", ("BOL-006", 7)), "cancelled again")
    by_id(env, "fulfil-order", 1, refuse=True)
    for bad in ("0", "x", "9"):
        by_id(env, "cancel-order", bad, refuse=True)
    expect_equal(place(env, "BOL-006:10")["id"], 2, "the released stock can be ordered")


@check("list-orders", D1, ["orders/list-orders", "orders/order"], "orders lists by id and filters by status")
def _(env, p):
    stocked(env, "BOL-006", 10)
    for qty in (1, 2, 3):
        place(env, "BOL-006:%d" % qty)
    by_id(env, "fulfil-order", 2)
    by_id(env, "cancel-order", 3)
    expect_equal([item["id"] for item in orders(env)], [1, 2, 3], "all orders")
    expect_equal([item["id"] for item in orders(env, "--status", "open")], [1], "open")
    expect_equal([item["id"] for item in orders(env, "--status", "fulfilled")], [2], "fulfilled")
    expect_equal([item["id"] for item in orders(env, "--status", "cancelled")], [3], "cancelled")
    for status in ("closed", "Open", ""):
        env.refuse("orders", "--status", status)


@check("stock-view", D1, ["reports/stock-view", "orders/availability", "stock/on-hand", "catalog/item"],
       "items shows every item by SKU with on hand, reserved and available")
def _(env, p):
    stocked(env, "NUT-010", 9, 4)
    stocked(env, "BOL-006", 10, 2)
    add(env, "CAP-001", "Cap")
    place(env, "BOL-006:3", "NUT-010:1")
    place(env, "BOL-006:2")
    by_id(env, "fulfil-order", 2)
    place(env, "NUT-010:5")
    by_id(env, "cancel-order", 3)
    expect_equal(items(env), [
        {"sku": "BOL-006", "name": "Part BOL-006", "reorderPoint": 2, "onHand": 8, "reserved": 3, "available": 5},
        {"sku": "CAP-001", "name": "Cap", "reorderPoint": 0, "onHand": 0, "reserved": 0, "available": 0},
        {"sku": "NUT-010", "name": "Part NUT-010", "reorderPoint": 4, "onHand": 9, "reserved": 1, "available": 8}],
        "items")


@check("reorder-report", D1, ["reports/reorder", "orders/availability", "catalog/item"],
       "reorder lists items whose available quantity is below the point, by shortfall then SKU")
def _(env, p):
    stocked(env, "AAA-001", 10, 12)  # shortfall 2
    stocked(env, "BBB-002", 10, 10)  # not below its point
    stocked(env, "CCC-003", 0, 5)  # shortfall 5
    stocked(env, "DDD-004", 10, 8)  # becomes shortfall 5 through an open order
    stocked(env, "EEE-005", 10, 9)  # an order that is fulfilled and one that is cancelled
    place(env, "DDD-004:7")
    place(env, "EEE-005:3")
    by_id(env, "fulfil-order", 2)
    place(env, "EEE-005:7")
    by_id(env, "cancel-order", 3)
    expect_equal(reorder(env), [
        {"sku": "CCC-003", "name": "Part CCC-003", "available": 0, "reorderPoint": 5, "shortfall": 5},
        {"sku": "DDD-004", "name": "Part DDD-004", "available": 3, "reorderPoint": 8, "shortfall": 5},
        {"sku": "AAA-001", "name": "Part AAA-001", "available": 10, "reorderPoint": 12, "shortfall": 2},
        {"sku": "EEE-005", "name": "Part EEE-005", "available": 7, "reorderPoint": 9, "shortfall": 2}], "reorder")


@check("persist", D1, ["depot/storage", "catalog/add-item", "stock/receive", "orders/place-order"],
       "A later process sees every part's committed changes in one UTF-8 JSON file")
def _(env, p):
    stocked(env, "BOL-006", 5)
    place(env, "BOL-006:2")
    expect_equal(figures(env, "BOL-006"), (5, 2, 3), "figures in a later process")
    env.stored_json()


@check("reads-untouched", D1, ["depot/storage", "reports/stock-view", "reports/reorder", "orders/list-orders"],
       "Queries never write the data file")
def _(env, p):
    stocked(env, "BOL-006", 5, 9)
    place(env, "BOL-006:2")
    env.reads_leave_data_untouched([["items"], ["reorder"], ["orders"], ["orders", "--status", "open"]])


@check("malformed-kept", D1, ["depot/storage", "depot/interface"], "Malformed data fails clearly and is kept")
def _(env, p):
    for content in (b"{not json", b"\xff\xfe\x00"):
        env.write_db(content)
        env.refuse("items")
        add(env, "BOL-006", "Bolt", refuse=True)
        if env.raw() != content:
            fail("the malformed data file was changed")


@check("missing-directory", D1, ["depot/storage", "depot/interface"],
       "A missing data directory is not created: changes fail and queries are empty")
def _(env, p):
    env.db = env.root / "absent" / "depot.json"
    expect_equal(items(env), [], "items without the data directory")
    add(env, "BOL-006", "Bolt", refuse=True)
    if (env.root / "absent").exists():
        fail("the missing data directory was created")


@check("edge-robustness", D1, ["depot/interface", "depot/inputs", "depot/storage"],
       "Extreme inputs fail through the error object, never with a crash")
def _(env, p):
    add(env, "BOL-006", "Bolt")
    receive(env, "BOL-006", "9" * 5000, refuse=True)
    expect_equal(receive(env, "BOL-006", "0" * 5000 + "7")["onHand"], 7, "receive 5000 leading zeros and 7")
    by_id(env, "fulfil-order", "9" * 5000, refuse=True)
    place(env, "BOL-006:" + "9" * 5000, refuse=True)
    no_crash(env, "add-item", "--sku", "NUT-010", "--name", "Nut \udcff")  # undecodable byte: the model is silent
    items(env)
    expect_equal(ascii_output(env, "add-item", "--sku", "CAF-001", "--name", "Café")["name"], "Café",
                 "a non-ASCII name with an ASCII-only stdout")
    env.write_db("[" * 100000)
    env.refuse("items")
    unusable_query(env, "items")
    unusable_change(env, "add-item", "--sku", "BOL-006", "--name", "Bolt")


@check("usage-errors", D1, ["depot/interface"], "Command-line errors use the error object and exit 2")
def _(env, p):
    for args in (["receive", "--sku", "BOL-006"], ["ship", "--id", "1"], ["items", "--bogus"], [], ["place-order"],
                 ["fulfil-order"], ["add-item", "--name", "Bolt"]):
        env.refuse(*args)


@check("no-leftovers", D1, ["depot/storage"], "Atomic replacement leaves no temporary files")
def _(env, p):
    stocked(env, "BOL-006", 9)
    place(env, "BOL-006:4")
    by_id(env, "fulfil-order", 1)
    place(env, "BOL-006:9", refuse=True)
    found = env.leftover_temp_files()
    if found:
        fail("files left beside the data file: %s" % found)
