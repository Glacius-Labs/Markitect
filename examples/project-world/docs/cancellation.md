# Cancel an order

A confirmed order can be cancelled before shipment. Cancelling also releases the order's active inventory reservation in the same SQLite transaction. A repeated cancellation is a no-op; a shipped order remains unchanged.

The finite checks in `tests/test_cancellation.py` cover successful cancellation, retry behavior, rejection after shipment, and rollback when reservation release fails. These checks do not cover a production database, concurrent distributed services, or every possible order state.
