from __future__ import annotations

import argparse
import json
import os
import re
import sys
import tempfile
from datetime import datetime
from pathlib import Path
from typing import Any


TIMESTAMP = re.compile(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}Z\Z")


class CommandError(Exception):
    """An expected input, conflict, or stored-data error."""


class JsonArgumentParser(argparse.ArgumentParser):
    def error(self, message: str) -> None:
        raise CommandError(message)


def parser_for_cli() -> argparse.ArgumentParser:
    parser = JsonArgumentParser(prog="app.py")
    parser.add_argument("--db", required=True, metavar="PATH")
    commands = parser.add_subparsers(dest="command", required=True, parser_class=JsonArgumentParser)

    book = commands.add_parser("book")
    book.add_argument("--room", required=True)
    book.add_argument("--start", required=True)
    book.add_argument("--end", required=True)
    book.add_argument("--title", required=True)

    listing = commands.add_parser("list")
    listing.add_argument("--room")

    cancel = commands.add_parser("cancel")
    cancel.add_argument("--id", type=int, required=True)

    commands.add_parser("summary")
    return parser


def normalize_text(value: str, field: str) -> str:
    result = value.strip()
    if not result:
        raise CommandError(f"{field} must not be empty")
    return result


def parse_timestamp(value: str, field: str) -> datetime:
    if not TIMESTAMP.fullmatch(value):
        raise CommandError(f"{field} must use YYYY-MM-DDTHH:MMZ")
    try:
        return datetime.strptime(value, "%Y-%m-%dT%H:%MZ")
    except ValueError as error:
        raise CommandError(f"{field} is not a valid UTC calendar time") from error


def empty_state() -> dict[str, Any]:
    return {"reservations": []}


def load_state(path: Path) -> dict[str, Any]:
    if not path.exists():
        return empty_state()
    try:
        raw = path.read_text(encoding="utf-8")
        state = json.loads(raw)
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        raise CommandError(f"cannot read database: {error}") from error
    if not isinstance(state, dict) or set(state) != {"reservations"}:
        raise CommandError("database has an invalid structure")
    reservations = state["reservations"]
    if not isinstance(reservations, list):
        raise CommandError("database reservations must be a list")
    seen_ids: set[int] = set()
    for record in reservations:
        if not isinstance(record, dict) or set(record) != {"id", "room", "start", "end", "title", "status"}:
            raise CommandError("database contains an invalid reservation")
        identifier = record["id"]
        if isinstance(identifier, bool) or not isinstance(identifier, int) or identifier < 1 or identifier in seen_ids:
            raise CommandError("database contains an invalid reservation ID")
        seen_ids.add(identifier)
        if not all(isinstance(record[key], str) for key in ("room", "start", "end", "title", "status")):
            raise CommandError("database contains invalid reservation fields")
        if not record["room"].strip() or not record["title"].strip() or record["status"] not in ("active", "canceled"):
            raise CommandError("database contains an invalid reservation")
        start = parse_timestamp(record["start"], "stored start")
        end = parse_timestamp(record["end"], "stored end")
        if end <= start:
            raise CommandError("database contains an invalid interval")
    return state


def save_state(path: Path, state: dict[str, Any]) -> None:
    temporary: str | None = None
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        with tempfile.NamedTemporaryFile(
            mode="w", encoding="utf-8", newline="\n", dir=path.parent,
            prefix=f".{path.name}.", suffix=".tmp", delete=False,
        ) as handle:
            temporary = handle.name
            json.dump(state, handle, ensure_ascii=False, indent=2)
            handle.write("\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
        temporary = None
    except OSError as error:
        raise CommandError(f"cannot save database: {error}") from error
    finally:
        if temporary is not None:
            try:
                os.unlink(temporary)
            except OSError:
                pass


def list_reservations(state: dict[str, Any], room: str | None) -> dict[str, Any]:
    records = state["reservations"]
    if room is not None:
        normalized_room = room.strip()
        records = [record for record in records if record["room"] == normalized_room]
    records = sorted(records, key=lambda record: (record["start"], record["id"]))
    return {"reservations": records}


def create_booking(state: dict[str, Any], arguments: argparse.Namespace) -> dict[str, Any]:
    room = normalize_text(arguments.room, "room")
    title = normalize_text(arguments.title, "title")
    start = parse_timestamp(arguments.start, "start")
    end = parse_timestamp(arguments.end, "end")
    if end <= start:
        raise CommandError("end must be later than start")

    for existing in state["reservations"]:
        if existing["room"] != room or existing["status"] != "active":
            continue
        existing_start = parse_timestamp(existing["start"], "stored start")
        existing_end = parse_timestamp(existing["end"], "stored end")
        if start < existing_end and existing_start < end:
            raise CommandError(f"reservation overlaps active reservation {existing['id']}")

    identifier = max((record["id"] for record in state["reservations"]), default=0) + 1
    record = {
        "id": identifier,
        "room": room,
        "start": arguments.start,
        "end": arguments.end,
        "title": title,
        "status": "active",
    }
    state["reservations"].append(record)
    return record


def cancel_booking(state: dict[str, Any], identifier: int) -> tuple[dict[str, Any], bool]:
    for record in state["reservations"]:
        if record["id"] == identifier:
            changed = record["status"] == "active"
            if record["status"] == "active":
                record["status"] = "canceled"
            return record, changed
    raise CommandError(f"reservation {identifier} does not exist")


def summarize(state: dict[str, Any]) -> dict[str, Any]:
    totals: dict[str, dict[str, int]] = {}
    for record in state["reservations"]:
        if record["status"] != "active":
            continue
        start = parse_timestamp(record["start"], "stored start")
        end = parse_timestamp(record["end"], "stored end")
        room = totals.setdefault(record["room"], {"active": 0, "minutes": 0})
        room["active"] += 1
        duration = end - start
        room["minutes"] += duration.days * 24 * 60 + duration.seconds // 60
    return {
        "rooms": [
            {"room": name, **totals[name]}
            for name in sorted(totals)
        ]
    }


def run(arguments: argparse.Namespace) -> dict[str, Any]:
    database = Path(arguments.db)
    state = load_state(database)
    if arguments.command == "list":
        return list_reservations(state, arguments.room)
    if arguments.command == "summary":
        return summarize(state)
    if arguments.command == "cancel":
        record, changed = cancel_booking(state, arguments.id)
        if changed:
            save_state(database, state)
        return record
    record = create_booking(state, arguments)
    save_state(database, state)
    return record


def main(argv: list[str] | None = None) -> int:
    try:
        result = run(parser_for_cli().parse_args(argv))
    except CommandError as error:
        print(json.dumps({"error": str(error)}, ensure_ascii=False))
        return 2
    except OSError as error:
        print(json.dumps({"error": f"database operation failed: {error}"}, ensure_ascii=False))
        return 2
    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
