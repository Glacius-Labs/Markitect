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
# Response key for a collection of entries; `--legacy` restores the pre-rename key for
# one compatibility period (README, "Compatibility").
ENTRIES_KEY, LEGACY_ENTRIES_KEY = "entries", "books"


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
    """Entries stored before tags existed have no `tags` field and are still valid."""
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
    """Return the stored entries in insertion order; a missing file is an empty list.

    Both storage formats are read. Entries without tags get `tags: []` in memory only;
    reading never rewrites the file.
    """
    try:
        entries = stored_records(json.loads(db.read_text(encoding="utf-8")))
    except FileNotFoundError:
        return []
    except (OSError, ValueError) as error:
        raise CommandError("storage", f"cannot read database {db}: {error}") from error
    if not isinstance(entries, list) or not all(valid_record(entry) for entry in entries):
        raise CommandError("storage", f"database {db} is malformed")
    if len({entry["id"] for entry in entries}) != len(entries):
        raise CommandError("storage", f"database {db} contains a repeated id")
    for entry in entries:
        entry.setdefault("tags", [])
    return entries


def save(db, entries):
    """Write the current format (version 2) to a temporary file next to the database, then
    atomically replace it. Only successful mutations save, so they upgrade version 1 files."""
    temporary = db.with_name(db.name + ".tmp")
    content = {"schemaVersion": SCHEMA_VERSION, "records": entries}
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


def commit(db, entries, op, ids):
    """Save a successful change and record it in the audit log."""
    save(db, entries)
    append_log(db, op, ids)


def find(entries, raw_id):
    entry_id = text_value("id", raw_id)
    for entry in entries:
        if entry["id"] == entry_id:
            return entry
    raise CommandError("not_found", f"no entry with id {entry_id!r}")


def new_entry(entry_id, title, author, pages):
    return {"id": text_value("id", entry_id), "title": text_value("title", title),
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


# Mutating commands change `entries` in place and return (result, processed ids).

def add(entries, args):
    entry = new_entry(args.id, args.title, args.author, args.pages)
    if any(existing["id"] == entry["id"] for existing in entries):
        raise CommandError("duplicate", f"id {entry['id']!r} already exists")
    entries.append(entry)
    return entry, [entry["id"]]


def finish(entries, args):
    entry = find(entries, args.id)
    entry["status"] = "finished"
    return entry, [entry["id"]]


def import_csv(entries, args):
    """All or nothing: every row is validated before any entry is appended."""
    taken = {entry["id"] for entry in entries}
    imported = []
    for number, row in enumerate(read_csv_rows(args.csv), 1):
        if len(row) != len(CSV_COLUMNS):
            raise CommandError("invalid_input", f"row {number}: expected {len(CSV_COLUMNS)} columns, got {len(row)}")
        try:
            entry = new_entry(*row)
        except CommandError as error:
            raise CommandError(error.code, f"row {number}: {error}") from None
        if entry["id"] in taken:
            raise CommandError("duplicate", f"row {number}: id {entry['id']!r} is already taken")
        taken.add(entry["id"])
        imported.append(entry)
    entries.extend(imported)
    return {"imported": len(imported)}, [entry["id"] for entry in imported]


def tag(entries, args):
    name = tag_value(args.add if args.add is not None else args.remove)
    entry = find(entries, args.id)
    tags = set(entry["tags"])
    if args.add is not None:
        tags.add(name)
    else:
        tags.discard(name)
    entry["tags"] = sorted(tags)
    return entry, [entry["id"]]


# Read-only commands return the result and never save.

def matching(entries, args):
    """Apply the optional filters of `list` and `summary`; all given filters must match."""
    if getattr(args, "status", None) is not None:
        status = status_value(args.status)
        entries = [entry for entry in entries if entry["status"] == status]
    if getattr(args, "author", None) is not None:
        author = text_value("author", args.author)
        entries = [entry for entry in entries if entry["author"] == author]
    if getattr(args, "tag", None) is not None:
        name = tag_value(args.tag)
        entries = [entry for entry in entries if name in entry["tags"]]
    return entries


def entries_key(args):
    return LEGACY_ENTRIES_KEY if args.legacy else ENTRIES_KEY


def list_entries(entries, args):
    return {entries_key(args): matching(entries, args)}


def summary(entries, args):
    """Per author: all entries, finished entries, the sum of known pages and how many
    entries have an unknown page count."""
    count_key = entries_key(args)
    authors = {}
    for entry in matching(entries, args):
        line = authors.setdefault(entry["author"], {"author": entry["author"], count_key: 0, "finished": 0,
                                                    "pages": 0, "unknownPages": 0})
        line[count_key] += 1
        line["finished"] += entry["status"] == "finished"
        if entry["pages"] is None:
            line["unknownPages"] += 1
        else:
            line["pages"] += entry["pages"]
    return {"authors": [authors[name] for name in sorted(authors)]}


def export_csv(entries, args):
    if args.csv.resolve() in (args.db.resolve(), args.db.with_name(args.db.name + ".log").resolve()):
        raise CommandError("export_failed", "the export file may not be the database or its log")
    try:
        with open(args.csv, "w", encoding="utf-8", newline="") as handle:
            writer = csv.writer(handle)
            writer.writerow(EXPORT_COLUMNS)
            for entry in entries:
                pages = "" if entry["pages"] is None else entry["pages"]
                writer.writerow([entry["id"], entry["title"], entry["author"], pages, entry["status"],
                                 TAG_SEPARATOR.join(entry["tags"])])
    except OSError as error:
        raise CommandError("export_failed", f"cannot write {args.csv}: {error}") from error
    return {"exported": len(entries)}


MUTATING = {"add": add, "finish": finish, "import": import_csv, "tag": tag}
READ_ONLY = {"list": list_entries, "summary": summary, "export": export_csv}


def build_parser():
    parser = Parser(prog="app.py", description="Keep a reading list in one JSON file.")
    parser.add_argument("--db", required=True, type=Path, help="database file")
    commands = parser.add_subparsers(dest="command", required=True)
    legacy_help = f"use the pre-rename response key {LEGACY_ENTRIES_KEY!r} (compatibility period)"
    list_command = commands.add_parser("list", help="list entries in insertion order")
    list_command.add_argument("--status", help="only entries with this status (unread or finished)")
    list_command.add_argument("--author", help="only entries by this author")
    list_command.add_argument("--tag", help="only entries with this tag")
    list_command.add_argument("--legacy", action="store_true", help=legacy_help)
    add_command = commands.add_parser("add", help="add an unread entry")
    for field in ("id", "title", "author"):
        add_command.add_argument("--" + field, required=True)
    add_command.add_argument("--pages", help="page count; omit it when unknown")
    finish_command = commands.add_parser("finish", help="mark an entry as finished")
    finish_command.add_argument("--id", required=True)
    import_command = commands.add_parser("import", help="add entries from a CSV file, all or nothing")
    import_command.add_argument("--csv", required=True, type=Path)
    summary_command = commands.add_parser("summary", help="entries, finished entries and pages per author")
    summary_command.add_argument("--author", help="only this author")
    summary_command.add_argument("--legacy", action="store_true", help=legacy_help)
    export_command = commands.add_parser("export", help="write every entry to a CSV file")
    export_command.add_argument("--csv", required=True, type=Path)
    tag_command = commands.add_parser("tag", help="add or remove one tag of an entry")
    tag_command.add_argument("--id", required=True)
    change = tag_command.add_mutually_exclusive_group(required=True)
    change.add_argument("--add", metavar="TAG")
    change.add_argument("--remove", metavar="TAG")
    return parser


def main(argv=None):
    try:
        args = build_parser().parse_args(argv)
        entries = load(args.db)
        if args.command in MUTATING:
            result, ids = MUTATING[args.command](entries, args)
            commit(args.db, entries, args.command, ids)
        else:
            result = READ_ONLY[args.command](entries, args)
    except CommandError as error:
        print(json.dumps({"error": {"code": error.code, "message": str(error)}}))
        return 2
    print(json.dumps(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
