from __future__ import annotations

import unittest
from pathlib import Path
import sys
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from shop.commerce.cancellation import ReservationInvariantError, cancel_order
from shop.inventory.reservations import status_for_order
from shop.orders.order import InvalidTransition, status
from support import database


class CancellationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.connection = database()

    def tearDown(self) -> None:
        self.connection.close()

    def test_cancellation_changes_order_and_releases_reservation(self) -> None:
        self.assertTrue(cancel_order(self.connection, "order-1"))
        self.assertEqual(status(self.connection, "order-1"), "cancelled")
        self.assertEqual(status_for_order(self.connection, "order-1"), "released")

    def test_repeated_cancellation_is_idempotent(self) -> None:
        self.assertTrue(cancel_order(self.connection, "order-1"))
        self.assertFalse(cancel_order(self.connection, "order-1"))
        self.assertEqual(status_for_order(self.connection, "order-1"), "released")

    def test_shipped_order_is_not_cancelled(self) -> None:
        self.connection.execute("UPDATE orders SET status = 'shipped' WHERE id = 'order-1'")
        with self.assertRaises(InvalidTransition):
            cancel_order(self.connection, "order-1")
        self.assertEqual(status(self.connection, "order-1"), "shipped")
        self.assertEqual(status_for_order(self.connection, "order-1"), "active")

    def test_release_failure_rolls_back_the_order_change(self) -> None:
        with patch("shop.commerce.cancellation.release_for_order", side_effect=RuntimeError("injected")):
            with self.assertRaisesRegex(RuntimeError, "injected"):
                cancel_order(self.connection, "order-1")
        self.assertEqual(status(self.connection, "order-1"), "confirmed")
        self.assertEqual(status_for_order(self.connection, "order-1"), "active")

    def test_missing_reservation_rolls_back_the_order_change(self) -> None:
        self.connection.execute("DELETE FROM reservations WHERE order_id = 'order-1'")
        with self.assertRaisesRegex(ReservationInvariantError, "released 0"):
            cancel_order(self.connection, "order-1")
        self.assertEqual(status(self.connection, "order-1"), "confirmed")
        self.assertEqual(
            self.connection.execute("SELECT COUNT(*) FROM reservations WHERE order_id = 'order-1'").fetchone()[0],
            0,
        )

    def test_pre_released_reservation_rolls_back_the_order_change(self) -> None:
        self.connection.execute("UPDATE reservations SET status = 'released' WHERE order_id = 'order-1'")
        with self.assertRaisesRegex(ReservationInvariantError, "released 0"):
            cancel_order(self.connection, "order-1")
        self.assertEqual(status(self.connection, "order-1"), "confirmed")
        self.assertEqual(status_for_order(self.connection, "order-1"), "released")

    def test_duplicate_active_reservations_roll_back_the_order_change(self) -> None:
        self.connection.execute("INSERT INTO reservations VALUES ('reservation-2', 'order-1', 2, 'active')")
        with self.assertRaisesRegex(ReservationInvariantError, "released 2"):
            cancel_order(self.connection, "order-1")
        self.assertEqual(status(self.connection, "order-1"), "confirmed")
        self.assertEqual(
            self.connection.execute("SELECT COUNT(*) FROM reservations WHERE order_id = 'order-1' AND status = 'active'").fetchone()[0],
            2,
        )


if __name__ == "__main__":
    unittest.main()
