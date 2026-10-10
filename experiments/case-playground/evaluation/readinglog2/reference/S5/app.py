"""Readinglog: keep a reading list in one JSON file (see README.md)."""
import argparse
import contextlib
import csv
from datetime import datetime, timezone
import json
from pathlib import Path
import sys
import unicodedata

FIELDS = ("id", "title", "author", "pages", "status", "tags")
STATUSES = ("unread", "finished")
CSV_COLUMNS = ["id", "title", "author", "pages"]
EXPORT_COLUMNS = CSV_COLUMNS + ["status", "tags"]
TAG_SEPARATOR = ";"
SCHEMA_VERSION = 2


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
    """A positive integer, or None (unknown) when the value is omitted or empty."""
    text = normalize(value or "")
    if not text:
        return None
    if not (text.isascii() and text.isdigit()) or int(text) < 1:
        raise CommandError("invalid_input", f"pages must be a positive integer, got {value!r}")
    return int(text)


def status_value(value):
    status = normalize(value)
    if status not in STATUSES:
        raise CommandError("invalid_input", f"status must be one of {', '.join(STATUSES)}, got {value!r}")
    return status


def tag_value(value):
    tag = text_value("tag", value)
    if TAG_SEPARATOR in tag:
        raise CommandError("invalid_input", f"a tag may not contain {TAG_SEPARATOR!r}")
    return tag


def valid_tags(tags):
    return (isinstance(tags, list)
            and all(isinstance(tag, str) and tag and TAG_SEPARATOR not in tag for tag in tags)
            and len(set(tags)) == len(tags))


def valid_record(record):
    """Books stored before tags existed have no `tags` field and are still valid."""
    return (isinstance(record, dict) and set(record) in (set(FIELDS), set(FIELDS) - {"tags"})
            and all(isinstance(record[key], str) and record[key] for key in ("id", "title", "author"))
            and (record["pages"] is None or (type(record["pages"]) is int and record["pages"] >= 1))
            and record["status"] in STATUSES
            and valid_tags(record.get("tags", [])))


def stored_records(data):
    """The record list of a version 2 object or a version 1 array; None for any other shape."""
    if isinstance(data, list):
        return data
    if (isinstance(data, dict) and set(data) == {"schemaVersion", "records"}
            and type(data["schemaVersion"]) is int and data["schemaVersion"] == SCHEMA_VERSION):
        return data["records"]
    return None


def load(db):
    """Return the stored books in insertion order; a missing file is an empty list.

    Both storage formats are read. Books without tags get `tags: []` in memory only;
    reading never rewrites the file.
    """
    try:
        books = stored_records(json.loads(db.read_text(encoding="utf-8")))
    except FileNotFoundError:
        return []
    except (OSError, ValueError) as error:
        raise CommandError("storage", f"cannot read database {db}: {error}") from error
    if not isinstance(books, list) or not all(valid_record(book) for book in books):
        raise CommandError("storage", f"database {db} is malformed")
    if len({book["id"] for book in books}) != len(books):
        raise CommandError("storage", f"database {db} contains a repeated id")
    for book in books:
        book.setdefault("tags", [])
    return books


def save(db, books):
    """Write the current format (version 2) to a temporary file next to the database, then
    atomically replace it. Only successful mutations save, so they upgrade version 1 files."""
    temporary = db.with_name(db.name + ".tmp")
    content = {"schemaVersion": SCHEMA_VERSION, "records": books}
    try:
        db.parent.mkdir(parents=True, exist_ok=True)
        temporary.write_text(json.dumps(content, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
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
            "author": text_value("author", author), "pages": pages_value(pages), "status": "unread",
            "tags": []}


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


def tag(books, args):
    name = tag_value(args.add if args.add is not None else args.remove)
    book = find(books, args.id)
    tags = set(book["tags"])
    if args.add is not None:
        tags.add(name)
    else:
        tags.discard(name)
    book["tags"] = sorted(tags)
    return book, [book["id"]]


# Read-only commands return the result and never save.

def matching(books, args):
    """Apply the optional filters of `list` and `summary`; all given filters must match."""
    if getattr(args, "status", None) is not None:
        status = status_value(args.status)
        books = [book for book in books if book["status"] == status]
    if getattr(args, "author", None) is not None:
        author = text_value("author", args.author)
        books = [book for book in books if book["author"] == author]
    if getattr(args, "tag", None) is not None:
        name = tag_value(args.tag)
        books = [book for book in books if name in book["tags"]]
    return books


def list_books(books, args):
    return {"books": matching(books, args)}


def summary(books, args):
    """Per author: all books, finished books, the sum of known pages and how many books
    have an unknown page count."""
    authors = {}
    for book in matching(books, args):
        entry = authors.setdefault(book["author"], {"author": book["author"], "books": 0, "finished": 0,
                                                    "pages": 0, "unknownPages": 0})
        entry["books"] += 1
        entry["finished"] += book["status"] == "finished"
        if book["pages"] is None:
            entry["unknownPages"] += 1
        else:
            entry["pages"] += book["pages"]
    return {"authors": [authors[name] for name in sorted(authors)]}


def export_csv(books, args):
    if args.csv.resolve() in (args.db.resolve(), args.db.with_name(args.db.name + ".log").resolve()):
        raise CommandError("export_failed", "the export file may not be the database or its log")
    try:
        with open(args.csv, "w", encoding="utf-8", newline="") as handle:
            writer = csv.writer(handle)
            writer.writerow(EXPORT_COLUMNS)
            for book in books:
                pages = "" if book["pages"] is None else book["pages"]
                writer.writerow([book["id"], book["title"], book["author"], pages, book["status"],
                                 TAG_SEPARATOR.join(book["tags"])])
    except OSError as error:
        raise CommandError("export_failed", f"cannot write {args.csv}: {error}") from error
    return {"exported": len(books)}


MUTATING = {"add": add, "finish": finish, "import": import_csv, "tag": tag}
READ_ONLY = {"list": list_books, "summary": summary, "export": export_csv}


def build_parser():
    parser = Parser(prog="app.py", description="Keep a reading list in one JSON file.")
    parser.add_argument("--db", required=True, type=Path, help="database file")
    commands = parser.add_subparsers(dest="command", required=True)
    list_command = commands.add_parser("list", help="list books in insertion order")
    list_command.add_argument("--status", help="only books with this status (unread or finished)")
    list_command.add_argument("--author", help="only books by this author")
    list_command.add_argument("--tag", help="only books with this tag")
    add_command = commands.add_parser("add", help="add an unread book")
    for field in ("id", "title", "author"):
        add_command.add_argument("--" + field, required=True)
    add_command.add_argument("--pages", help="page count; omit it when unknown")
    finish_command = commands.add_parser("finish", help="mark a book as finished")
    finish_command.add_argument("--id", required=True)
    import_command = commands.add_parser("import", help="add books from a CSV file, all or nothing")
    import_command.add_argument("--csv", required=True, type=Path)
    summary_command = commands.add_parser("summary", help="books, finished books and pages per author")
    summary_command.add_argument("--author", help="only this author")
    export_command = commands.add_parser("export", help="write every book to a CSV file")
    export_command.add_argument("--csv", required=True, type=Path)
    tag_command = commands.add_parser("tag", help="add or remove one tag of a book")
    tag_command.add_argument("--id", required=True)
    change = tag_command.add_mutually_exclusive_group(required=True)
    change.add_argument("--add", metavar="TAG")
    change.add_argument("--remove", metavar="TAG")
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
