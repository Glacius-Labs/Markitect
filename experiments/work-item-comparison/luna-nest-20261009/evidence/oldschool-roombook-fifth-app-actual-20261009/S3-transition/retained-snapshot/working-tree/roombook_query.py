"""Query helpers for ordered Roombook reservation records."""

from __future__ import annotations

from collections.abc import Iterable, Mapping
from typing import Any


_VALID_STATUSES = frozenset({"active", "canceled"})


def filter_reservations(
    records: Iterable[Mapping[str, Any]],
    room: str | None = None,
    status: str | None = None,
) -> list[Mapping[str, Any]]:
    """Filter already-ordered records by optional exact room and status.

    The room filter is trimmed before case-sensitive comparison. The helper
    preserves the input order and never sorts records. ``status`` may be
    ``"active"``, ``"canceled"``, or ``None``; any other value raises
    ``ValueError``.
    """
    if status is not None and status not in _VALID_STATUSES:
        raise ValueError("status must be 'active' or 'canceled'")

    normalized_room = room.strip() if room is not None else None
    return [
        record
        for record in records
        if (normalized_room is None or record["room"] == normalized_room)
        and (status is None or record["status"] == status)
    ]
