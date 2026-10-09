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
    cursor = connection.execute(
        """
        UPDATE orders
        SET status = ?
        WHERE id = ?
          AND ? IN ('confirmed', 'shipped', 'cancelled')
          AND status IN ('confirmed', 'shipped', 'cancelled')
          AND (status <> 'cancelled' OR ? = 'cancelled')
          AND (status <> 'shipped' OR ? = 'shipped')
        """,
        (value, order_id, value, value, value),
    )
    if cursor.rowcount != 1:
        row = connection.execute("SELECT status FROM orders WHERE id = ?", (order_id,)).fetchone()
        if row is None:
            raise KeyError(f"unknown order: {order_id}")
        raise InvalidTransition(f"cannot change order in {row[0]!r} state to {value!r}")
