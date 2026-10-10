"""Readinglog: keep a reading list in one JSON file (see README.md)."""
import argparse
import contextlib
import csv
from datetime import datetime, timezone
import json
from pathlib import Path
import sys
import unicodedata

FIELDS = ("id", "title", "author", "pages", "status")
STATUSES = ("unread", "finished")
CSV_COLUMNS = ["id", "title", "author", "pages"]


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


def status_value(value):
    status = normalize(value)
    if status not in STATUSES:
        raise CommandError("invalid_input", f"status must be one of {', '.join(STATUSES)}, got {value!r}")
    return status


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


def find(books, raw_id):
    book_id = text_value("id", raw_id)
    for book in books:
        if book["id"] == book_id:
            return book
    raise CommandError("not_found", f"no book with id {book_id!r}")


def new_book(book_id, title, author, pages):
    return {"id": text_value("id", book_id), "title": text_value("title", title),
            "author": text_value("author", author), "pages": pages_value(pages), "status": "unread"}


def read_csv_rows(path):
    """The data rows of an import file; empty lines are skipped."""
    try:
        with open(path, encoding="utf-8", newline="") as handle:
            rows = [row for row in csv.reader(handle) if row]
    except (OSError, ValueError, csv.Error) as error:
        raise CommandError("invalid_input", f"cannot read CSV file {path}: {error}") from error
    if not rows or rows[0] != CSV_COLUMNS:
        raise CommandError("invalid_input", f"the CSV header must be exactly {','.join(CSV_COLUMNS)}")
    return rows[1:]


# Mutating commands change `books` in place and return (result, processed ids).

def add(books, args):
    book = new_book(args.id, args.title, args.author, args.pages)
    if any(existing["id"] == book["id"] for existing in books):
        raise CommandError("duplicate", f"id {book['id']!r} already exists")
    books.append(book)
    return book, [book["id"]]


def finish(books, args):
    book = find(books, args.id)
    book["status"] = "finished"
    return book, [book["id"]]


def import_csv(books, args):
    """All or nothing: every row is validated before any book is appended."""
    taken = {book["id"] for book in books}
    imported = []
    for number, row in enumerate(read_csv_rows(args.csv), 1):
        if len(row) != len(CSV_COLUMNS):
            raise CommandError("invalid_input", f"row {number}: expected {len(CSV_COLUMNS)} columns, got {len(row)}")
        try:
            book = new_book(*row)
        except CommandError as error:
            raise CommandError(error.code, f"row {number}: {error}") from None
        if book["id"] in taken:
            raise CommandError("duplicate", f"row {number}: id {book['id']!r} is already taken")
        taken.add(book["id"])
        imported.append(book)
    books.extend(imported)
    return {"imported": len(imported)}, [book["id"] for book in imported]


# Read-only commands return the result and never save.

def list_books(books, args):
    if args.status is not None:
        status = status_value(args.status)
        books = [book for book in books if book["status"] == status]
    return {"books": books}


def summary(books, args):
    authors = {}
    for book in books:
        entry = authors.setdefault(book["author"], {"author": book["author"], "books": 0, "finished": 0, "pages": 0})
        entry["books"] += 1
        entry["finished"] += book["status"] == "finished"
        entry["pages"] += book["pages"]
    return {"authors": [authors[name] for name in sorted(authors)]}


MUTATING = {"add": add, "finish": finish, "import": import_csv}
READ_ONLY = {"list": list_books, "summary": summary}


def build_parser():
    parser = Parser(prog="app.py", description="Keep a reading list in one JSON file.")
    parser.add_argument("--db", required=True, type=Path, help="database file")
    commands = parser.add_subparsers(dest="command", required=True)
    list_command = commands.add_parser("list", help="list books in insertion order")
    list_command.add_argument("--status", help="only books with this status (unread or finished)")
    add_command = commands.add_parser("add", help="add an unread book")
    for field in ("id", "title", "author", "pages"):
        add_command.add_argument("--" + field, required=True)
    finish_command = commands.add_parser("finish", help="mark a book as finished")
    finish_command.add_argument("--id", required=True)
    import_command = commands.add_parser("import", help="add books from a CSV file, all or nothing")
    import_command.add_argument("--csv", required=True, type=Path)
    commands.add_parser("summary", help="books, finished books and pages per author")
    return parser


def main(argv=None):
    try:
        args = build_parser().parse_args(argv)
        books = load(args.db)
        if args.command in MUTATING:
            result, ids = MUTATING[args.command](books, args)
            commit(args.db, books, args.command, ids)
        else:
            result = READ_ONLY[args.command](books, args)
    except CommandError as error:
        print(json.dumps({"error": {"code": error.code, "message": str(error)}}))
        return 2
    print(json.dumps(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
