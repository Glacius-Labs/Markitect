from __future__ import annotations

import unittest
from pathlib import Path
import sys
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from shop.commerce.cancellation import ReservationInvariantError, cancel_order
from shop.inventory.reservations import release_for_order, status_for_order
from shop.orders.order import InvalidTransition, set_status, status
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
        self.assertEqual(
            self.connection.execute("SELECT quantity FROM reservations WHERE order_id = 'order-1'").fetchone()[0],
            3,
        )
        self.assertEqual(
            self.connection.execute("SELECT COALESCE(SUM(quantity), 0) FROM reservations WHERE status = 'active'").fetchone()[0],
            0,
        )

    def test_release_keeps_recorded_quantity_out_of_active_reserved_total(self) -> None:
        self.assertEqual(
            self.connection.execute("SELECT COALESCE(SUM(quantity), 0) FROM reservations WHERE status = 'active'").fetchone()[0],
            3,
        )
        self.assertEqual(release_for_order(self.connection, "order-1"), 1)
        self.assertEqual(
            self.connection.execute("SELECT quantity, status FROM reservations WHERE order_id = 'order-1'").fetchone(),
            (3, "released"),
        )
        self.assertEqual(
            self.connection.execute("SELECT COALESCE(SUM(quantity), 0) FROM reservations WHERE status = 'active'").fetchone()[0],
            0,
        )
        self.assertEqual(release_for_order(self.connection, "order-1"), 0)
        self.assertEqual(
            self.connection.execute("SELECT quantity, status FROM reservations WHERE order_id = 'order-1'").fetchone(),
            (3, "released"),
        )

    def test_set_status_rejects_unknown_state_without_mutation(self) -> None:
        with self.assertRaises(InvalidTransition):
            set_status(self.connection, "order-1", "not-a-status")
        self.assertEqual(status(self.connection, "order-1"), "confirmed")

    def test_cancelled_order_is_terminal_but_cancelled_is_idempotent(self) -> None:
        set_status(self.connection, "order-1", "cancelled")
        for value in ("confirmed", "shipped"):
            with self.subTest(value=value):
                with self.assertRaises(InvalidTransition):
                    set_status(self.connection, "order-1", value)
                self.assertEqual(status(self.connection, "order-1"), "cancelled")
        set_status(self.connection, "order-1", "cancelled")
        self.assertEqual(status(self.connection, "order-1"), "cancelled")

    def test_set_status_preserves_unknown_order_key_error(self) -> None:
        with self.assertRaisesRegex(KeyError, "unknown order: missing"):
            set_status(self.connection, "missing", "confirmed")

    def test_shipped_order_is_not_cancelled(self) -> None:
        self.connection.execute("UPDATE orders SET status = 'shipped' WHERE id = 'order-1'")
        with self.assertRaises(InvalidTransition):
            cancel_order(self.connection, "order-1")
        self.assertEqual(status(self.connection, "order-1"), "shipped")
        self.assertEqual(status_for_order(self.connection, "order-1"), "active")

    def test_set_status_cannot_cancel_shipped_order_or_change_reservation(self) -> None:
        self.connection.execute("UPDATE orders SET status = 'shipped' WHERE id = 'order-1'")
        with self.assertRaises(InvalidTransition):
            set_status(self.connection, "order-1", "cancelled")
        self.assertEqual(status(self.connection, "order-1"), "shipped")
        self.assertEqual(
            self.connection.execute("SELECT quantity, status FROM reservations WHERE order_id = 'order-1'").fetchone(),
            (3, "active"),
        )
        self.assertEqual(
            self.connection.execute("SELECT COALESCE(SUM(quantity), 0) FROM reservations WHERE status = 'active'").fetchone()[0],
            3,
        )

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
