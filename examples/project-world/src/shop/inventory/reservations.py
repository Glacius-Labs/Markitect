from __future__ import annotations

import sqlite3


def release_for_order(connection: sqlite3.Connection, order_id: str) -> int:
    cursor = connection.execute(
        "UPDATE reservations SET status = 'released' WHERE order_id = ? AND status = 'active'",
        (order_id,),
    )
    return cursor.rowcount


def status_for_order(connection: sqlite3.Connection, order_id: str) -> str:
    row = connection.execute("SELECT status FROM reservations WHERE order_id = ?", (order_id,)).fetchone()
    if row is None:
        raise KeyError(f"reservation missing for order: {order_id}")
    return str(row[0])
