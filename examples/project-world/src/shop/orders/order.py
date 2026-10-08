from __future__ import annotations

import sqlite3


class InvalidTransition(ValueError):
    """Raised when an order lifecycle transition is not allowed."""


def status(connection: sqlite3.Connection, order_id: str) -> str:
    row = connection.execute("SELECT status FROM orders WHERE id = ?", (order_id,)).fetchone()
    if row is None:
        raise KeyError(f"unknown order: {order_id}")
    return str(row[0])


def set_status(connection: sqlite3.Connection, order_id: str, value: str) -> None:
    cursor = connection.execute("UPDATE orders SET status = ? WHERE id = ?", (value, order_id))
    if cursor.rowcount != 1:
        raise KeyError(f"unknown order: {order_id}")
