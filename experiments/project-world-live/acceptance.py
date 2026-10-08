"""Independent finite acceptance checks over the actual Shop implementation.

This file stays outside the adopting checkout and is not supplied to managers.
It deliberately builds its own database instead of importing the fixture tests.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import sqlite3
import sys


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", type=Path, required=True)
    args = parser.parse_args()
    sys.path.insert(0, str(args.repo.resolve() / "src"))
    from shop.commerce.cancellation import cancel_order
    from shop.orders.order import InvalidTransition

    checked: list[str] = []
    for initial in ("confirmed", "packing"):
        for reservations in (0, 1, 2):
            connection = database(initial, reservations)
            try:
                if reservations == 1:
                    assert cancel_order(connection, "acceptance-order") is True
                    assert state(connection) == ("cancelled", 0, 1)
                    assert cancel_order(connection, "acceptance-order") is False
                    assert state(connection) == ("cancelled", 0, 1)
                else:
                    try:
                        cancel_order(connection, "acceptance-order")
                    except Exception:
                        pass
                    else:
                        raise AssertionError("invalid reservation cardinality was accepted")
                    assert state(connection) == (initial, reservations, 0)
                checked.append(f"{initial}/reservations={reservations}")
            finally:
                connection.close()
    connection = database("shipped", 1)
    try:
        try:
            cancel_order(connection, "acceptance-order")
        except InvalidTransition:
            pass
        else:
            raise AssertionError("shipped cancellation was accepted")
        assert state(connection) == ("shipped", 1, 0)
        checked.append("shipped/rejected")
    finally:
        connection.close()
    print(json.dumps({"status": "passed", "cases": checked}))


def database(order_state: str, reservations: int) -> sqlite3.Connection:
    connection = sqlite3.connect(":memory:", isolation_level=None)
    connection.executescript(
        "CREATE TABLE orders (id TEXT PRIMARY KEY, status TEXT NOT NULL);"
        "CREATE TABLE reservations (id TEXT PRIMARY KEY, order_id TEXT NOT NULL,"
        " quantity INTEGER NOT NULL, status TEXT NOT NULL);"
    )
    connection.execute("INSERT INTO orders VALUES (?, ?)", ("acceptance-order", order_state))
    for number in range(reservations):
        connection.execute("INSERT INTO reservations VALUES (?, ?, ?, ?)",
                           (f"reservation-{number}", "acceptance-order", 4, "active"))
    return connection


def state(connection: sqlite3.Connection) -> tuple[str, int, int]:
    order_state = connection.execute("SELECT status FROM orders WHERE id = ?",
                                     ("acceptance-order",)).fetchone()[0]
    active = connection.execute("SELECT count(*) FROM reservations WHERE status = 'active'").fetchone()[0]
    released = connection.execute("SELECT count(*) FROM reservations WHERE status = 'released'").fetchone()[0]
    return order_state, active, released


if __name__ == "__main__":
    main()
