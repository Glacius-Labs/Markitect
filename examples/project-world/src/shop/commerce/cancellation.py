from __future__ import annotations

import sqlite3

from shop.inventory.reservations import release_for_order
from shop.orders.order import InvalidTransition, set_status, status


def cancel_order(connection: sqlite3.Connection, order_id: str) -> bool:
    """Cancel a confirmed order and release its reservation atomically.

    Returns False when the same cancellation was already completed. A shipped
    order is rejected, and any failure rolls both updates back.
    """
    connection.execute("BEGIN IMMEDIATE")
    try:
        current = status(connection, order_id)
        if current == "cancelled":
            connection.commit()
            return False
        if current != "confirmed":
            raise InvalidTransition(f"cannot cancel order in {current!r} state")
        set_status(connection, order_id, "cancelled")
        release_for_order(connection, order_id)
        connection.commit()
        return True
    except Exception:
        connection.rollback()
        raise
