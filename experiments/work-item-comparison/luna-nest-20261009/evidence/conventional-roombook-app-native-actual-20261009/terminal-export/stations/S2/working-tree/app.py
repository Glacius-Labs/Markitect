"""Small, durable command-line room booking book."""

import argparse
from datetime import datetime
import json
import os
from pathlib import Path
import re
import tempfile


TIME_FORMAT = "%Y-%m-%dT%H:%MZ"
TIME_PATTERN = re.compile(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}Z\Z")


class AppError(Exception):
    """An invalid request, business conflict, or unusable database."""


class ArgumentParser(argparse.ArgumentParser):
    def error(self, message):
        raise AppError(message)


def parse_time(value):
    if not TIME_PATTERN.fullmatch(value):
        raise AppError("timestamps must use YYYY-MM-DDTHH:MMZ")
    try:
        return datetime.strptime(value, TIME_FORMAT)
    except ValueError as exc:
        raise AppError(f"invalid timestamp: {value}") from exc


def load_state(path):
    if not path.exists():
        return {"next_id": 1, "reservations": []}
    try:
        state = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise AppError(f"cannot read database: {exc}") from exc
    if not isinstance(state, dict) or set(state) != {"next_id", "reservations"}:
        raise AppError("database has an invalid structure")
    next_id, reservations = state["next_id"], state["reservations"]
    if type(next_id) is not int or next_id < 1 or not isinstance(reservations, list):
        raise AppError("database has an invalid structure")
    seen = set()
    max_id = 0
    for record in reservations:
        if not isinstance(record, dict) or set(record) != {"id", "room", "start", "end", "title", "status"}:
            raise AppError("database has an invalid reservation")
        record_id = record["id"]
        if type(record_id) is not int or record_id < 1 or record_id in seen:
            raise AppError("database has an invalid reservation ID")
        if (not isinstance(record["room"], str) or not record["room"]
                or not isinstance(record["title"], str) or not record["title"]
                or record["status"] != "active"):
            raise AppError("database has an invalid reservation")
        start, end = parse_time(record["start"]), parse_time(record["end"])
        if end <= start:
            raise AppError("database has an invalid reservation interval")
        seen.add(record_id)
        max_id = max(max_id, record_id)
    if next_id <= max_id:
        raise AppError("database next ID is invalid")
    return state


def save_state(path, state):
    parent = path.parent
    temp_path = None
    try:
        with tempfile.NamedTemporaryFile("w", encoding="utf-8", newline="\n",
                                         dir=parent, prefix=f".{path.name}.",
                                         suffix=".tmp", delete=False) as handle:
            temp_path = Path(handle.name)
            json.dump(state, handle, ensure_ascii=False, separators=(",", ":"))
            handle.write("\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temp_path, path)
    except OSError as exc:
        raise AppError(f"cannot write database: {exc}") from exc
    finally:
        if temp_path is not None:
            try:
                temp_path.unlink(missing_ok=True)
            except OSError:
                pass


def build_parser():
    parser = ArgumentParser(prog="app.py")
    parser.add_argument("--db", required=True, type=Path, help="UTF-8 JSON database path")
    commands = parser.add_subparsers(dest="command", required=True, parser_class=ArgumentParser)

    book = commands.add_parser("book")
    book.add_argument("--room", required=True)
    book.add_argument("--start", required=True)
    book.add_argument("--end", required=True)
    book.add_argument("--title", required=True)

    listing = commands.add_parser("list")
    listing.add_argument("--room")
    return parser


def run(args):
    state = load_state(args.db)
    if args.command == "list":
        room = args.room.strip() if args.room is not None else None
        records = [record for record in state["reservations"]
                   if room is None or record["room"] == room]
        records.sort(key=lambda item: (item["start"], item["id"]))
        return {"reservations": records}

    room, title = args.room.strip(), args.title.strip()
    if not room:
        raise AppError("room must not be empty")
    if not title:
        raise AppError("title must not be empty")
    start, end = parse_time(args.start), parse_time(args.end)
    if end <= start:
        raise AppError("end must be later than start")
    for existing in state["reservations"]:
        if existing["room"] == room:
            other_start = parse_time(existing["start"])
            other_end = parse_time(existing["end"])
            if start < other_end and other_start < end:
                raise AppError("booking overlaps an existing booking in this room")
    reservation = {
        "id": state["next_id"], "room": room, "start": args.start,
        "end": args.end, "title": title, "status": "active",
    }
    state["reservations"].append(reservation)
    state["next_id"] += 1
    save_state(args.db, state)
    return reservation


def main(argv=None):
    try:
        args = build_parser().parse_args(argv)
        result = run(args)
    except AppError as exc:
        print(json.dumps({"error": str(exc)}, ensure_ascii=False))
        return 2
    except OSError as exc:
        print(json.dumps({"error": f"database error: {exc}"}, ensure_ascii=False))
        return 2
    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
