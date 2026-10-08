from __future__ import annotations

import sqlite3


def database() -> sqlite3.Connection:
    connection = sqlite3.connect(":memory:", isolation_level=None)
    connection.executescript(
        """
        CREATE TABLE orders (id TEXT PRIMARY KEY, status TEXT NOT NULL);
        CREATE TABLE reservations (
            id TEXT PRIMARY KEY,
            order_id TEXT NOT NULL REFERENCES orders(id),
            quantity INTEGER NOT NULL CHECK(quantity > 0),
            status TEXT NOT NULL
        );
        INSERT INTO orders VALUES ('order-1', 'confirmed');
        INSERT INTO reservations VALUES ('reservation-1', 'order-1', 3, 'active');
        """
    )
    return connection
