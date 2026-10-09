from __future__ import annotations

from datetime import datetime
from typing import Any


def summarize(state: dict[str, Any], room: str | None = None) -> dict[str, Any]:
    """Return active booking totals, optionally for one normalized room name.

    The state is expected to have passed the application's stored-data validation.
    Room matching trims only the filter's surrounding whitespace and remains
    case-sensitive, matching the booking and listing commands.
    """
    normalized_room = room.strip() if room is not None else None
    totals: dict[str, dict[str, int]] = {}

    for record in state["bookings"]:
        if record["status"] != "active":
            continue
        if normalized_room is not None and record["room"] != normalized_room:
            continue

        start = datetime.strptime(record["start"], "%Y-%m-%dT%H:%MZ")
        end = datetime.strptime(record["end"], "%Y-%m-%dT%H:%MZ")
        room_totals = totals.setdefault(record["room"], {"active": 0, "minutes": 0})
        room_totals["active"] += 1
        duration = end - start
        room_totals["minutes"] += duration.days * 24 * 60 + duration.seconds // 60

    return {
        "rooms": [
            {"room": name, **totals[name]}
            for name in sorted(totals)
        ]
    }
