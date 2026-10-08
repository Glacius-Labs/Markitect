# Cancel an order

A confirmed order can be cancelled before shipment only when exactly one active inventory reservation exists. Cancelling releases that reservation in the same SQLite transaction. A missing, previously released, or duplicate active reservation rolls the order update back; a repeated cancellation is a no-op, and a shipped order remains unchanged.

The finite checks in `tests/test_cancellation.py` cover successful cancellation, retry behavior, rejection after shipment, and rollback when reservation release fails. These checks do not cover a production database, concurrent distributed services, or every possible order state.
