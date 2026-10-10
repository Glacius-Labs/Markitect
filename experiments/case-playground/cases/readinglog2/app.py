"""Readinglog: keep a reading list in one JSON file (see README.md)."""
import argparse
import contextlib
from datetime import datetime, timezone
import json
from pathlib import Path
import sys
import unicodedata

FIELDS = ("id", "title", "author", "pages", "status")
STATUSES = ("unread", "finished")


class CommandError(Exception):
    """A domain or input error, reported as an error object with exit code 2."""

    def __init__(self, code, message):
        super().__init__(message)
        self.code = code


class Parser(argparse.ArgumentParser):
    """Report usage errors through the error contract instead of exiting."""

    def error(self, message):
        raise CommandError("invalid_input", message)


def normalize(value):
    """NFC, strip, and collapse internal whitespace runs to one space."""
    return " ".join(unicodedata.normalize("NFC", value).split())


def text_value(name, value):
    text = normalize(value)
    if not text:
        raise CommandError("invalid_input", f"{name} must not be empty")
    return text


def pages_value(value):
    text = normalize(value)
    if not (text.isascii() and text.isdigit()) or int(text) < 1:
        raise CommandError("invalid_input", f"pages must be a positive integer, got {value!r}")
    return int(text)


def valid_record(record):
    return (isinstance(record, dict) and set(record) == set(FIELDS)
            and all(isinstance(record[key], str) and record[key] for key in ("id", "title", "author"))
            and type(record["pages"]) is int and record["pages"] >= 1
            and record["status"] in STATUSES)


def load(db):
    """Return the stored books in insertion order; a missing file is an empty list."""
    try:
        books = json.loads(db.read_text(encoding="utf-8"))
    except FileNotFoundError:
        return []
    except (OSError, ValueError) as error:
        raise CommandError("storage", f"cannot read database {db}: {error}") from error
    if not isinstance(books, list) or not all(valid_record(book) for book in books):
        raise CommandError("storage", f"database {db} is malformed")
    if len({book["id"] for book in books}) != len(books):
        raise CommandError("storage", f"database {db} contains a repeated id")
    return books


def save(db, books):
    """Write a temporary file next to the database, then atomically replace it."""
    temporary = db.with_name(db.name + ".tmp")
    try:
        db.parent.mkdir(parents=True, exist_ok=True)
        temporary.write_text(json.dumps(books, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        temporary.replace(db)
    except OSError as error:
        with contextlib.suppress(OSError):
            temporary.unlink(missing_ok=True)
        raise CommandError("storage", f"cannot write database {db}: {error}") from error


def append_log(db, op, ids):
    """Append the audit line of one successful change (README, rule R1)."""
    entry = {"op": op, "ids": ids, "at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")}
    try:
        with open(db.with_name(db.name + ".log"), "a", encoding="utf-8", newline="\n") as log:
            log.write(json.dumps(entry, ensure_ascii=False) + "\n")
    except OSError as error:
        raise CommandError("storage", f"cannot append to the audit log: {error}") from error


def commit(db, books, op, ids):
    """Save a successful change and record it in the audit log."""
    save(db, books)
    append_log(db, op, ids)


def add(books, args):
    book = {"id": text_value("id", args.id), "title": text_value("title", args.title),
            "author": text_value("author", args.author), "pages": pages_value(args.pages),
            "status": "unread"}
    if any(existing["id"] == book["id"] for existing in books):
        raise CommandError("duplicate", f"id {book['id']!r} already exists")
    books.append(book)
    return book


def build_parser():
    parser = Parser(prog="app.py", description="Keep a reading list in one JSON file.")
    parser.add_argument("--db", required=True, type=Path, help="database file")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("list", help="list every book in insertion order")
    add_command = commands.add_parser("add", help="add an unread book")
    for field in ("id", "title", "author", "pages"):
        add_command.add_argument("--" + field, required=True)
    return parser


def main(argv=None):
    try:
        args = build_parser().parse_args(argv)
        books = load(args.db)
        if args.command == "add":
            result = add(books, args)
            commit(args.db, books, "add", [result["id"]])
        else:
            result = {"books": books}
    except CommandError as error:
        print(json.dumps({"error": {"code": error.code, "message": str(error)}}))
        return 2
    print(json.dumps(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
