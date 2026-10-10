"""Hidden holdout checks of the readinglog2 case (EVALUATION.md, "Hidden evaluation files").

    python3 -I holdout.py --repo DIR --station N

Runs every holdout released up to station N against `DIR/app.py`, each on fresh
temporary data, and prints `{"station": N, "checks": [{"id", "status", "item", "rule",
"detail", "source"}]}`. Exits 0 even when checks fail.

A holdout tests only what the public README.md and BACKLOG.md of the case already say:
hidden cases, never hidden requirements. Each one names its backlog item ("baseline"
for the seed's add/list), its project rule and, in `source`, the public sentence it
derives from. Matters the public text leaves open (reference/README.md, "Left open")
are not tested. Standard library only.
"""
import argparse
import csv
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time
import unicodedata

STATIONS = 6
APP_TIMEOUT = 30
IMPORT_HEADER = "id,title,author,pages\n"
EXPORT_HEADER = ["id", "title", "author", "pages", "status", "tags"]
AT_FORMAT = re.compile(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z")
LOG_KEYS = {"op", "ids", "at"}

R1_SOURCE = ("README R1: Every successful mutating command appends exactly one JSON line to <db>.log: "
             "{op: COMMAND, ids: [IDS in processing order], at: UTC ISO-8601}.")
R1_NOOP = ("README R1: A successful mutating command saves the database and logs its line even when no book "
           "actually changed.")
R1_READ_ONLY = ("README R1: Failed commands and read-only commands append nothing. README Storage: "
                "Read-only commands never create or change the file.")
R2_SOURCE = ("README R2: Every domain/input error prints {error: {code, message}} and exits 2 without any mutation. "
             "This includes usage errors.")
R2_TABLE = "README R2: Codes come from the error-code table. A new code must be added to that table."
R3_SOURCE = ("README R3: All text input, from arguments or files, is Unicode NFC normalized, stripped, and internal "
             "whitespace runs collapse to one space. IDs are case-sensitive after normalization.")
R3_PATHS = "README R3: File paths such as --db are used exactly as given."
MALFORMED = ("README Storage: a file that is not valid UTF-8 JSON, does not have the expected shape or holds an "
             "invalid record is malformed. Every command then fails with storage, and the file is never overwritten.")

EMITTED_CODES = set()  # every error code the CLI printed during this run (for the R2 table check)


class Fail(Exception):
    """The candidate violates a public requirement."""


def shown(value, limit=240):
    text = value if isinstance(value, str) else repr(value)
    return text if len(text) <= limit else text[:limit] + "..."


def expect(actual, wanted, what):
    if actual != wanted:
        raise Fail(f"{what}: got {shown(repr(actual))}, expected {shown(repr(wanted))}")


def normalize(text):
    return " ".join(unicodedata.normalize("NFC", text).split())


def content(db):
    """The bytes of the database and of its log (None when missing)."""
    log = log_path(db)
    return db.read_bytes() if db.exists() else None, log.read_bytes() if log.exists() else None


def stamp(db):
    """The content plus the database's modification time: a rewrite with equal bytes still shows."""
    return content(db), db.stat().st_mtime_ns if db.exists() else None


def log_path(db):
    return db.with_name(db.name + ".log")


class Trial:
    """One holdout's fresh temporary folder and the CLI calls it makes there."""

    def __init__(self, repo, station, folder):
        self.repo, self.station, self.folder = repo, station, folder
        self.db = folder / "state.json"
        self.key = "entries" if station >= 6 else "books"
        self.started = time.time()

    # --- running the CLI ---------------------------------------------------------------

    def call(self, args, db=None):
        db = self.db if db is None else db
        command = [sys.executable, "-B", str(self.repo / "app.py"), "--db", str(db), *args]
        env = dict(os.environ, PYTHONIOENCODING="utf-8")
        try:
            proc = subprocess.run(command, cwd=self.repo, capture_output=True, timeout=APP_TIMEOUT, env=env)
        except subprocess.TimeoutExpired:
            raise Fail(f"timed out after {APP_TIMEOUT}s: {shown(list(args))}") from None
        try:
            value = json.loads(proc.stdout.decode("utf-8"))
        except (UnicodeError, ValueError):
            raise Fail(f"stdout is not one UTF-8 JSON object: args={shown(list(args))} exit={proc.returncode} "
                       f"stdout={shown(proc.stdout.decode('utf-8', 'replace'))} "
                       f"stderr={shown(proc.stderr.decode('utf-8', 'replace')[-300:])}") from None
        if not isinstance(value, dict):
            raise Fail(f"response is not a JSON object: args={shown(list(args))} value={shown(value)}")
        error = value.get("error")
        if proc.returncode != 0 and isinstance(error, dict) and isinstance(error.get("code"), str):
            EMITTED_CODES.add(error["code"])
        return proc.returncode, value

    def ok(self, *args, db=None):
        """A successful command: exit 0 and its JSON object."""
        status, value = self.call(args, db)
        if status != 0:
            raise Fail(f"exit {status}, expected 0: args={shown(list(args))} response={shown(value)}")
        return value

    def fails(self, code, *args, db=None):
        """A domain or input error: exit 2, the error object with CODE, no mutation (R2)."""
        db = self.db if db is None else db
        before = content(db)
        status, value = self.call(args, db)
        error = value.get("error")
        if (status != 2 or not isinstance(error, dict) or not isinstance(error.get("code"), str)
                or not isinstance(error.get("message"), str)):
            raise Fail(f"expected exit 2 with {{'error': {{'code', 'message'}}}}: args={shown(list(args))} "
                       f"exit={status} response={shown(value)}")
        if error["code"] != code:
            raise Fail(f"error code {error['code']!r}, expected {code!r}: args={shown(list(args))}")
        if content(db) != before:
            raise Fail(f"failed command changed or created the database or its log: args={shown(list(args))}")
        return error

    def readonly(self, *args, db=None):
        """A read-only command: success, and the database and its log stay untouched (R1, Storage)."""
        db = self.db if db is None else db
        before = stamp(db)
        value = self.ok(*args, db=db)
        if stamp(db) != before:
            raise Fail(f"read-only command changed, rewrote or created the database or its log: "
                       f"args={shown(list(args))}")
        return value

    # --- shortcuts ---------------------------------------------------------------------

    def add(self, book_id, title="Title", author="Writer", pages="100", db=None):
        args = ["add", "--id", book_id, "--title", title, "--author", author]
        if pages is not None:
            args += ["--pages", pages]
        return self.ok(*args, db=db)

    def list(self, *filters, db=None, legacy=False):
        key = "books" if legacy else self.key
        value = self.ok("list", *filters, db=db)
        expect(set(value), {key}, f"keys of the list response {shown(list(filters))}")
        if not isinstance(value[key], list):
            raise Fail(f"list response {key!r} is not a list")
        return value[key]

    def ids(self, *filters, db=None):
        return [book["id"] for book in self.list(*filters, db=db)]

    def rec(self, book_id, title="Title", author="Writer", pages=100, status="unread", tags=()):
        """The expected record at this station (records gain `tags` with B08)."""
        record = {"id": book_id, "title": title, "author": author, "pages": pages, "status": status}
        if self.station >= 3:
            record["tags"] = list(tags)
        return record

    def line(self, author, count, finished, pages, unknown=0, legacy=False):
        """The expected summary line at this station (B03, B13 unknownPages, B15 entries)."""
        key = "books" if legacy or self.station < 6 else "entries"
        line = {"author": author, key: count, "finished": finished, "pages": pages}
        if self.station >= 5:
            line["unknownPages"] = unknown
        return line

    def csv_file(self, name, text, encoding="utf-8"):
        path = self.folder / name
        path.write_bytes(text.encode(encoding) if isinstance(text, str) else text)
        return path

    def import_text(self, text, name="in.csv", db=None):
        return self.ok("import", "--csv", str(self.csv_file(name, text)), db=db)

    def export_rows(self, name="out.csv", db=None, readonly=False):
        target = self.folder / name
        value = (self.readonly if readonly else self.ok)("export", "--csv", str(target), db=db)
        return value, read_csv(target)

    def write_db(self, records, version, db=None):
        db = self.db if db is None else db
        data = records if version == 1 else {"schemaVersion": 2, "records": records}
        db.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        return db

    def stored(self, db=None):
        db = self.db if db is None else db
        try:
            return json.loads(db.read_text(encoding="utf-8"))
        except (OSError, ValueError) as error:
            raise Fail(f"cannot read the stored database {db.name}: {error}") from None

    def stored_v2_records(self, db=None):
        data = self.stored(db)
        if not (isinstance(data, dict) and set(data) == {"schemaVersion", "records"}):
            raise Fail(f"stored file is not {{schemaVersion, records}}: {shown(data)}")
        expect(data["schemaVersion"], 2, "stored schemaVersion")
        if not isinstance(data["records"], list):
            raise Fail("stored records is not a list")
        return data["records"]

    def log_lines(self, db=None):
        """Every audit line, each checked against the R1 format."""
        log = log_path(self.db if db is None else db)
        if not log.exists():
            return []
        try:
            text = log.read_text(encoding="utf-8")
        except UnicodeError as error:
            raise Fail(f"the audit log is not UTF-8: {error}") from None
        return [self.log_entry(line) for line in text.splitlines() if line.strip()]

    def log_entry(self, text):
        try:
            entry = json.loads(text)
        except ValueError:
            raise Fail(f"audit line is not JSON: {shown(text)}") from None
        if not isinstance(entry, dict) or set(entry) != LOG_KEYS:
            raise Fail(f"audit line must have exactly op, ids and at: {shown(text)}")
        if not isinstance(entry["op"], str) or not (isinstance(entry["ids"], list)
                                                    and all(isinstance(i, str) for i in entry["ids"])):
            raise Fail(f"audit line op must be a string and ids a list of strings: {shown(text)}")
        at = entry["at"]
        if not (isinstance(at, str) and AT_FORMAT.fullmatch(at)):
            raise Fail(f"audit line at is not UTC ISO-8601 with Z: {shown(text)}")
        try:
            moment = datetime.strptime(at[:19], "%Y-%m-%dT%H:%M:%S").replace(tzinfo=timezone.utc).timestamp()
        except ValueError:
            raise Fail(f"audit line at is not a valid time: {shown(text)}") from None
        if not self.started - 5 <= moment <= time.time() + 5:
            raise Fail(f"audit line at is not the current UTC time: {shown(text)}")
        return entry

    def ops(self, db=None):
        return [(entry["op"], entry["ids"]) for entry in self.log_lines(db)]

    def readme(self):
        path = self.repo / "README.md"
        if not path.is_file():
            raise Fail("README.md is missing")
        return path.read_text(encoding="utf-8", errors="replace")

    def readme_mentions(self, *patterns):
        text = self.readme()
        missing = [pattern for pattern in patterns if not re.search(pattern, text)]
        if missing:
            raise Fail(f"README.md does not mention {missing}")


def read_csv(path):
    if not path.is_file():
        raise Fail(f"{path.name} was not written")
    try:
        text = path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as error:
        raise Fail(f"{path.name} is not UTF-8: {error}") from None
    return list(csv.reader(text.removeprefix("\ufeff").splitlines(keepends=True)))


def error_codes_in_readme(text):
    """Codes in the first column of every Markdown table whose header names a code."""
    codes, block = set(), []
    for raw in text.splitlines() + [""]:
        line = raw.strip()
        if line.startswith("|"):
            block.append(line)
            continue
        if block and any("code" in cell.lower() for cell in block[0].strip("|").split("|")):
            for row in block[1:]:
                cell = row.strip("|").split("|")[0].strip().strip("`").strip()
                if cell and not set(cell) <= set("-: "):
                    codes.add(cell)
        block = []
    return codes


# --- registry -------------------------------------------------------------------------

HOLDOUTS = []


def holdout(check_id, since, item, rule, source, until=None, final=False):
    """Register a check released at station `since` (and retired after `until`, when a
    later public decision replaced what it tests)."""
    def register(body):
        HOLDOUTS.append({"id": check_id, "since": since, "until": until, "item": item, "rule": rule,
                         "source": source, "final": final, "body": body})
        return body
    return register


def released(station):
    selected = [h for h in HOLDOUTS if h["since"] <= station and (h["until"] is None or station <= h["until"])]
    return [h for h in selected if not h["final"]] + [h for h in selected if h["final"]]


# --- baseline: add and list (seed README) ----------------------------------------------

BAD_PAGES = ("0", "-3", "1.5", "+5", "1_000", "abc", "12 3", "\u0663", "\uff11\uff12")


@holdout("base-pages-validation", 1, "baseline", "R2",
         "README Records: pages is a page count, at least 1, given as ASCII digits. "
         "BACKLOG B13: a given value must still be a positive integer.")
def _(t):
    for bad in BAD_PAGES[:3]:
        t.fails("invalid_input", "add", "--id", "x", "--title", "T", "--author", "A", "--pages", bad)
    t.add("keep")
    for bad in BAD_PAGES:
        t.fails("invalid_input", "add", "--id", "x", "--title", "T", "--author", "A", "--pages", bad)


@holdout("base-pages-required", 1, "baseline", "R2",
         "README Usage: add --id ID --title TITLE --author AUTHOR --pages PAGES; pages is an integer of at least 1 "
         "(until B13 makes pages optional).", until=4)
def _(t):
    t.fails("invalid_input", "add", "--id", "x", "--title", "T", "--author", "A")
    t.add("keep")
    t.fails("invalid_input", "add", "--id", "x", "--title", "T", "--author", "A", "--pages", "")
    t.fails("invalid_input", "add", "--id", "x", "--title", "T", "--author", "A", "--pages", " \t")


@holdout("base-id-exact-after-normalization", 1, "baseline", "R3",
         "README R3: Comparisons are exact after normalization: Dune and dune are different IDs, while e-acute "
         "written as one character or as e plus a combining accent is the same ID.")
def _(t):
    t.add("Dune")
    t.add("dune")
    t.add("caf\u00e9")
    t.fails("duplicate", "add", "--id", " cafe\u0301 ", "--title", "T", "--author", "A", "--pages", "1")
    expect(t.ids(), ["Dune", "dune", "caf\u00e9"], "stored IDs")


@holdout("base-db-path-exact", 1, "baseline", "R3", R3_PATHS + " README R1: <db>.log is the database path with "
         ".log appended.")
def _(t):
    name = "my  reading cafe\u0301.json"
    t.add("a", db=t.folder / name)
    names = set(os.listdir(t.folder))
    for wanted in (name, name + ".log"):
        if wanted not in names:
            raise Fail(f"{wanted!r} was not written; folder holds {sorted(names)}")
    for unwanted in (normalize(name), normalize(name) + ".log"):
        if unwanted in names:
            raise Fail(f"the path was normalized to {unwanted!r}")


@holdout("base-list-read-only", 1, "baseline", "R1", R1_READ_ONLY)
def _(t):
    expect(t.readonly("list"), {t.key: []}, "list of a missing database")
    t.add("a")
    t.readonly("list")


# --- S1: B01 finish and list --status ----------------------------------------------------

@holdout("b01-finish-record-and-log", 1, "B01", "R1",
         "B01: finish --id ID marks the book as finished and prints the updated record. " + R1_SOURCE)
def _(t):
    t.add("a", title="First", author="Ann", pages="10")
    t.add("b", title="Second", author="Bob", pages="20")
    expect(t.ok("finish", "--id", "b"), t.rec("b", "Second", "Bob", 20, "finished"), "finish response")
    expect(t.list(), [t.rec("a", "First", "Ann", 10), t.rec("b", "Second", "Bob", 20, "finished")], "stored books")
    expect(t.ops(), [("add", ["a"]), ("add", ["b"]), ("finish", ["b"])], "audit log")


@holdout("b01-finish-repeat-logs-again", 1, "B01", "R1",
         "B01: Finishing a book that is already finished succeeds again with the same record. " + R1_NOOP)
def _(t):
    t.add("a")
    first = t.ok("finish", "--id", "a")
    expect(t.ok("finish", "--id", "a"), first, "repeated finish response")
    expect(t.ops(), [("add", ["a"]), ("finish", ["a"]), ("finish", ["a"])], "audit log")


@holdout("b01-finish-unknown-id", 1, "B01", "R2",
         "B01: A missing ID is an error. README error codes: not_found: No book has the given ID. " + R2_SOURCE)
def _(t):
    t.fails("not_found", "finish", "--id", "ghost")
    t.add("a")
    t.fails("not_found", "finish", "--id", "ghost")


@holdout("b01-finish-normalized-id", 1, "B01", "R3", R3_SOURCE)
def _(t):
    t.add("caf\u00e9")
    expect(t.ok("finish", "--id", "\u00a0cafe\u0301\t"), t.rec("caf\u00e9", status="finished"), "finish response")
    t.fails("not_found", "finish", "--id", "CAF\u00c9")


@holdout("b01-finish-empty-id", 1, "B01", "R3",
         "README R3: A value that must not be empty is invalid when it is empty after normalization. "
         "README error codes: invalid_input.")
def _(t):
    t.add("a")
    t.fails("invalid_input", "finish", "--id", " \t\u00a0")


@holdout("b01-usage-errors", 1, "B01", "R2", R2_SOURCE)
def _(t):
    t.add("a")
    t.fails("invalid_input", "finish")
    t.fails("invalid_input", "finish", "--id")
    t.fails("invalid_input", "list", "--status")


@holdout("b01-status-filter-order", 1, "B01", None,
         "B01: list --status unread|finished lists only books with that status, in insertion order; list without "
         "--status stays unchanged.")
def _(t):
    for book_id in "abcd":
        t.add(book_id)
    t.ok("finish", "--id", "d")
    t.ok("finish", "--id", "b")
    expect(t.ids("--status", "finished"), ["b", "d"], "finished books")
    expect(t.ids("--status", "unread"), ["a", "c"], "unread books")
    expect(t.list(), [t.rec("a"), t.rec("b", status="finished"), t.rec("c"), t.rec("d", status="finished")],
           "list without --status")


@holdout("b01-status-invalid", 1, "B01", "R2",
         "B01: any other status value is invalid. README R3: Comparisons are exact after normalization.")
def _(t):
    for bad in ("done", "", "Finished", "UNREAD", "finish"):
        t.fails("invalid_input", "list", "--status", bad)
    t.add("a")
    for bad in ("done", "Finished"):
        t.fails("invalid_input", "list", "--status", bad)


@holdout("b01-status-normalized", 1, "B01", "R3",
         "README R3: This covers every value that becomes or selects book data: ... statuses and filter values.")
def _(t):
    t.add("a")
    t.add("b")
    t.ok("finish", "--id", "b")
    expect(t.ids("--status", " finished\u00a0"), ["b"], "list --status ' finished '")
    expect(t.ids("--status", "\tunread "), ["a"], "list --status ' unread '")


@holdout("b01-list-status-read-only", 1, "B01", "R1", R1_READ_ONLY)
def _(t):
    t.readonly("list", "--status", "finished")
    t.add("a")
    t.ok("finish", "--id", "a")
    t.readonly("list", "--status", "finished")
    t.readonly("list", "--status", "unread")


@holdout("b01-log-append-only", 1, "B01", "R1", "README R1: The tool only appends to the log. It never reads or "
         "rewrites it.")
def _(t):
    t.add("a")
    log = log_path(t.db)
    with open(log, "ab") as handle:
        handle.write(b"a foreign line that is not JSON\n")
    before = log.read_bytes()
    t.ok("finish", "--id", "a")
    after = log.read_bytes()
    if not after.startswith(before):
        raise Fail("finish rewrote the existing log instead of appending")
    added = after[len(before):].decode("utf-8").splitlines()
    expect(len(added), 1, "lines appended by finish")
    entry = t.log_entry(added[0])
    expect((entry["op"], entry["ids"]), ("finish", ["a"]), "appended line")


@holdout("b01-finish-malformed-db", 1, "B01", "R2", MALFORMED)
def _(t):
    for content in ("{broken", '[{"id": "a"}]'):
        t.db.write_text(content, encoding="utf-8")
        t.fails("storage", "finish", "--id", "a")


@holdout("b01-readme-documents-finish", 1, "B01", None,
         "BACKLOG: Each item includes its tests and docs. B01: finish --id ID; list --status unread|finished.")
def _(t):
    t.readme_mentions(r"\bfinish\b", r"--status\b")


# --- S2: B02 import ---------------------------------------------------------------------

@holdout("b02-import-file-order", 2, "B02", None,
         "B02: On success the books are appended as unread in file order and the command prints {imported: COUNT}.")
def _(t):
    t.add("z", title="Zeta", author="Zed", pages="5")
    text = IMPORT_HEADER + 'c,Gamma,Ann,30\na,"Alpha, the first",Bob,10\nb,"Say ""hi""",Ann,20\n'
    expect(t.import_text(text), {"imported": 3}, "import response")
    expect(t.list(), [t.rec("z", "Zeta", "Zed", 5), t.rec("c", "Gamma", "Ann", 30),
                      t.rec("a", "Alpha, the first", "Bob", 10), t.rec("b", 'Say "hi"', "Ann", 20)], "stored books")


@holdout("b02-import-log", 2, "B02", "R1", R1_SOURCE + " ids lists the IDs ... in the order it processed them.")
def _(t):
    t.add("z")
    t.import_text(IMPORT_HEADER + "c,Gamma,Ann,30\na,Alpha,Bob,10\nb,Beta,Ann,20\n")
    expect(t.ops(), [("add", ["z"]), ("import", ["c", "a", "b"])], "audit log")


@holdout("b02-import-header-only", 2, "B02", "R1",
         "B02: A file with only the header imports nothing and succeeds. README R1: ids ... is [] when there were "
         "none. " + R1_NOOP)
def _(t):
    t.add("z")
    expect(t.import_text(IMPORT_HEADER), {"imported": 0}, "import response")
    expect(t.ids(), ["z"], "stored IDs")
    expect(t.ops(), [("add", ["z"]), ("import", [])], "audit log")


@holdout("b02-import-empty-lines", 2, "B02", None, "B02: Empty lines are ignored.")
def _(t):
    expect(t.import_text(IMPORT_HEADER + "\na,Alpha,Ann,10\n\n\nb,Beta,Ann,20\n\n"), {"imported": 2},
           "import response")
    expect(t.ids(), ["a", "b"], "stored IDs")


@holdout("b02-import-normalization", 2, "B02", "R3", R3_SOURCE + " Whitespace means any Unicode whitespace: "
         "spaces, tabs, line breaks, no-break spaces.")
def _(t):
    text = (IMPORT_HEADER + ' cafe\u0301 ,"  The \t Long\u00a0 Title ",A\u030asa  Berg , 20 \n'
            + 'n2,"Part\none",Ann,7\n')
    expect(t.import_text(text), {"imported": 2}, "import response")
    expect(t.list(), [t.rec("caf\u00e9", "The Long Title", "\u00c5sa Berg", 20), t.rec("n2", "Part one", "Ann", 7)],
           "stored books")


@holdout("b02-import-duplicates", 2, "B02", "R3",
         "B02: no ID may appear twice in the file or exist already. README error codes: duplicate. " + R3_SOURCE)
def _(t):
    t.add("caf\u00e9")
    for index, rows in enumerate(("x,One,Ann,1\n x\u00a0,Two,Ann,2\n", "y,One,Ann,1\ncafe\u0301,Two,Ann,2\n",
                                  "d,One,Ann,1\nd,Two,Ann,2\n")):
        path = t.csv_file(f"dup{index}.csv", IMPORT_HEADER + rows)
        t.fails("duplicate", "import", "--csv", str(path))
    expect(t.ids(), ["caf\u00e9"], "stored IDs")


@holdout("b02-import-all-or-nothing", 2, "B02", "R2",
         "B02: Every row follows the rules of add ... The import is all or nothing: if any row is invalid, nothing "
         "is imported. A row with too few or too many columns is invalid input.")
def _(t):
    bad_rows = ("b,Beta,Ann,0", "b, \t,Ann,10", "b,Beta,\u00a0,10", " ,Beta,Ann,10", "b,Beta,Ann,ten", "b,Beta,Ann",
                "b,Beta,Ann,10,extra")
    for index, row in enumerate(bad_rows):
        path = t.csv_file(f"bad{index}.csv", IMPORT_HEADER + "a,Alpha,Ann,10\n" + row + "\n")
        if index < 2:
            t.fails("invalid_input", "import", "--csv", str(path))  # a missing database stays missing
    t.add("z")
    for index in range(len(bad_rows)):
        t.fails("invalid_input", "import", "--csv", str(t.folder / f"bad{index}.csv"))
    expect(t.ids(), ["z"], "stored IDs")


@holdout("b02-import-pages-validation", 2, "B02", "R2",
         "B02: Every row follows the rules of add. README Records: pages ... at least 1, given as ASCII digits. "
         "B13: a given value must still be a positive integer.")
def _(t):
    t.add("z")
    for index, bad in enumerate(BAD_PAGES):
        path = t.csv_file(f"pages{index}.csv", IMPORT_HEADER + f"a,Alpha,Ann,1\nb,Beta,Ann,{bad}\n")
        t.fails("invalid_input", "import", "--csv", str(path))
    expect(t.ids(), ["z"], "stored IDs")


@holdout("b02-import-empty-pages-invalid", 2, "B02", "R2",
         "B02: Every row follows the rules of add (pages are required until B13).", until=4)
def _(t):
    t.add("z")
    for index, cell in enumerate(("", " ")):
        path = t.csv_file(f"nopages{index}.csv", IMPORT_HEADER + f"a,Alpha,Ann,{cell}\n")
        t.fails("invalid_input", "import", "--csv", str(path))


@holdout("b02-import-bad-file", 2, "B02", "R2",
         "B02: A missing or unreadable file, a missing or different header ... is invalid input. README error "
         "codes: invalid_input ... an input file that cannot be read or has the wrong format.")
def _(t):
    t.add("z")
    (t.folder / "folder.csv").mkdir()
    files = [t.folder / "missing.csv", t.folder / "folder.csv",
             t.csv_file("empty.csv", ""),
             t.csv_file("short-header.csv", "id,title,author\na,Alpha,Ann\n"),
             t.csv_file("order.csv", "title,id,author,pages\nAlpha,a,Ann,10\n"),
             t.csv_file("extra-header.csv", "id,title,author,pages,status\n"),
             t.csv_file("case-header.csv", "ID,Title,Author,Pages\na,Alpha,Ann,10\n"),
             t.csv_file("no-header.csv", "a,Alpha,Ann,10\n"),
             t.csv_file("latin1.csv", IMPORT_HEADER + "a,Caf\u00e9,Ann,10\n", encoding="latin-1")]
    for path in files:
        t.fails("invalid_input", "import", "--csv", str(path))
    expect(t.ids(), ["z"], "stored IDs")


@holdout("b02-import-path-exact", 2, "B02", "R3", R3_PATHS)
def _(t):
    path = t.csv_file("in  put cafe\u0301.csv", IMPORT_HEADER + "a,Alpha,Ann,10\n")
    expect(t.ok("import", "--csv", str(path)), {"imported": 1}, "import response")


@holdout("b02-import-usage", 2, "B02", "R2", R2_SOURCE)
def _(t):
    t.add("z")
    t.fails("invalid_input", "import")
    t.fails("invalid_input", "import", "--csv")


@holdout("b02-import-malformed-db", 2, "B02", "R2", MALFORMED)
def _(t):
    path = t.csv_file("in.csv", IMPORT_HEADER + "a,Alpha,Ann,10\n")
    t.db.write_text("{broken", encoding="utf-8")
    t.fails("storage", "import", "--csv", str(path))


# --- S2: B03 summary, B04 docs -------------------------------------------------------------

@holdout("b03-summary-order-and-counts", 2, "B03", "R3",
         "B03: one object per author, ordered by author name in plain code-point order. books and pages count all "
         "of the author's books, finished only the finished ones. " + R3_SOURCE)
def _(t):
    for book_id, author, pages in (("1", "b", "10"), ("2", "\u00c4", "5"), ("3", "B", "7"), ("4", "a", "1"),
                                   ("5", "b", "20"), ("6", "Z", "3"), ("7", "E\u0301mile  Zola", "2"),
                                   ("8", " \u00c9mile Zola ", "4")):
        t.add(book_id, author=author, pages=pages)
    t.ok("finish", "--id", "1")
    t.ok("finish", "--id", "3")
    t.ok("finish", "--id", "8")
    expect(t.ok("summary"), {"authors": [t.line("B", 1, 1, 7), t.line("Z", 1, 0, 3), t.line("a", 1, 0, 1),
                                         t.line("b", 2, 1, 30), t.line("\u00c4", 1, 0, 5),
                                         t.line("\u00c9mile Zola", 2, 1, 6)]}, "summary")


@holdout("b03-summary-read-only", 2, "B03", "R1", "B03: An empty database gives {authors: []}. " + R1_READ_ONLY)
def _(t):
    expect(t.readonly("summary"), {"authors": []}, "summary of a missing database")
    t.add("a")
    t.readonly("summary")


@holdout("b03-summary-usage", 2, "B03", "R2", R2_SOURCE + " " + MALFORMED)
def _(t):
    t.add("a")
    t.fails("invalid_input", "summary", "--bogus", "x")
    t.db.write_text("{broken", encoding="utf-8")
    t.fails("storage", "summary")


@holdout("b04-readme-documents-commands", 2, "B04", None,
         "B04: README usage, failure and stored-state docs for them (import and summary).")
def _(t):
    t.readme_mentions(r"\bimport\b", r"--csv\b", r"\bsummary\b")


# --- S3: B05 list --author --------------------------------------------------------------

@holdout("b05-author-filter", 3, "B05", "R3",
         "B05: list --author NAME lists only that author's books, in insertion order, and combines with --status. "
         + R3_SOURCE)
def _(t):
    for book_id, author in (("1", "Ann Lee"), ("2", "Bob"), ("3", "ann lee"), ("4", "Ann Lee"), ("5", "Zo\u00eb")):
        t.add(book_id, author=author)
    t.ok("finish", "--id", "4")
    expect(t.ids("--author", " Ann\u00a0\tLee "), ["1", "4"], "list --author ' Ann  Lee '")
    expect(t.ids("--author", "ann lee"), ["3"], "list --author 'ann lee'")
    expect(t.ids("--author", "Zoe\u0308"), ["5"], "list --author with a combining accent")
    expect(t.ids("--author", "Ann Lee", "--status", "finished"), ["4"], "list --author --status finished")
    expect(t.ids("--status", "unread", "--author", "Ann Lee"), ["1"], "list --status unread --author")
    expect(t.ids("--author", "Nobody"), [], "list --author of an unknown author")


@holdout("b05-author-filter-read-only", 3, "B05", "R1", R1_READ_ONLY)
def _(t):
    t.add("1", author="Ann")
    t.readonly("list", "--author", "Ann")
    t.readonly("list", "--author", "Ann", "--status", "unread")


# --- S3: B06 export -----------------------------------------------------------------------

@holdout("b06-export-content", 3, "B06", None,
         "B06: export --csv PATH writes every book to a UTF-8 CSV file with the header id,title,author,pages,status, "
         "in insertion order and with standard CSV quoting, and prints {exported: COUNT}.")
def _(t):
    t.add("b2", title='Say "hi", then go', author="\u00c5sa", pages="20")
    t.add("a1", title="Plain", author="Ann", pages="10")
    t.add("c3", title="Comma, inside", author="Zo\u00eb", pages="3")
    t.ok("finish", "--id", "a1")
    value, rows = t.export_rows()
    expect(value, {"exported": 3}, "export response")
    expect(rows, [EXPORT_HEADER, ["b2", 'Say "hi", then go', "\u00c5sa", "20", "unread", ""],
                  ["a1", "Plain", "Ann", "10", "finished", ""], ["c3", "Comma, inside", "Zo\u00eb", "3", "unread", ""]],
           "exported rows")


@holdout("b06-export-replaces-file", 3, "B06", None, "B06: export --csv PATH writes every book to a UTF-8 CSV file.")
def _(t):
    t.add("a", title="Alpha", author="Ann", pages="1")
    (t.folder / "out.csv").write_text("stale,content\nmore,stale,rows\n", encoding="utf-8")
    expect(t.export_rows()[1], [EXPORT_HEADER, ["a", "Alpha", "Ann", "1", "unread", ""]], "exported rows")


@holdout("b06-export-read-only", 3, "B06", "R1",
         "B06: It never changes the database. README R1: A command that only reads books is read-only ..., even "
         "when it writes some other file.")
def _(t):
    t.add("a")
    t.ok("finish", "--id", "a")
    t.export_rows(readonly=True)
    t.export_rows("again.csv", readonly=True)
    expect(t.ops(), [("add", ["a"]), ("finish", ["a"])], "audit log")


@holdout("b06-export-empty-database", 3, "B06", "R1",
         "B09: export ... empty database. " + R1_READ_ONLY)
def _(t):
    value, rows = t.export_rows(readonly=True)
    expect((value, rows), ({"exported": 0}, [EXPORT_HEADER]), "export of a missing database")


@holdout("b06-export-failed", 3, "B06", "R2",
         "B06: If the destination cannot be written, the command fails with the new error code export_failed. "
         + R2_SOURCE)
def _(t):
    t.add("a")
    (t.folder / "adir").mkdir()
    (t.folder / "plain").write_text("x", encoding="utf-8")
    for target in (t.folder / "adir", t.folder / "plain" / "out.csv", t.folder / "missing" / "out.csv"):
        t.fails("export_failed", "export", "--csv", str(target))


@holdout("b06-error-code-table", 3, "B06", "R2", R2_TABLE + " B06: the new error code export_failed.")
def _(t):
    codes = error_codes_in_readme(t.readme())
    if "export_failed" not in codes:
        raise Fail(f"the README error-code table lacks export_failed; it lists {sorted(codes)}")


@holdout("b06-export-path-exact", 3, "B06", "R3", R3_PATHS)
def _(t):
    t.add("a")
    name = "out  cafe\u0301.csv"
    t.ok("export", "--csv", str(t.folder / name))
    names = set(os.listdir(t.folder))
    if name not in names or normalize(name) in names:
        raise Fail(f"export did not write exactly {name!r}; folder holds {sorted(names)}")


@holdout("b06-export-usage", 3, "B06", "R2", R2_SOURCE + " " + MALFORMED)
def _(t):
    t.add("a")
    t.fails("invalid_input", "export")
    t.fails("invalid_input", "export", "--csv")
    t.db.write_text("{broken", encoding="utf-8")
    t.fails("storage", "export", "--csv", str(t.folder / "out.csv"))


# --- S3: B07 summary --author -------------------------------------------------------------

@holdout("b07-summary-author", 3, "B07", "R3",
         "B07: summary --author NAME reports only that author; an unknown author gives {authors: []}. summary "
         "without --author stays unchanged. " + R3_SOURCE)
def _(t):
    for book_id, author, pages in (("1", "Ann Lee", "10"), ("2", "Bob", "5"), ("3", "Ann Lee", "7"),
                                   ("4", "ann lee", "1")):
        t.add(book_id, author=author, pages=pages)
    t.ok("finish", "--id", "3")
    expect(t.readonly("summary", "--author", " Ann \u00a0Lee"), {"authors": [t.line("Ann Lee", 2, 1, 17)]},
           "summary --author ' Ann  Lee'")
    expect(t.readonly("summary", "--author", "ann lee"), {"authors": [t.line("ann lee", 1, 0, 1)]},
           "summary --author 'ann lee'")
    expect(t.readonly("summary", "--author", "Nobody"), {"authors": []}, "summary --author of an unknown author")
    expect(t.readonly("summary"), {"authors": [t.line("Ann Lee", 2, 1, 17), t.line("Bob", 1, 0, 5),
                                               t.line("ann lee", 1, 0, 1)]}, "summary without --author")


@holdout("b07-summary-author-usage", 3, "B07", "R2", R2_SOURCE)
def _(t):
    t.add("a")
    t.fails("invalid_input", "summary", "--author")


# --- S3: B08 tags -------------------------------------------------------------------------

@holdout("b08-tag-record-and-log", 3, "B08", "R1",
         "B08: tag --id ID --add TAG or tag --id ID --remove TAG changes one book's tags and prints the updated "
         "record. " + R1_SOURCE)
def _(t):
    t.add("a", title="T", author="A", pages="5")
    t.add("b")
    expect(t.ok("tag", "--id", "a", "--add", "classic"), t.rec("a", "T", "A", 5, tags=["classic"]), "tag --add")
    expect(t.ok("tag", "--id", "a", "--remove", "classic"), t.rec("a", "T", "A", 5), "tag --remove")
    expect(t.ops(), [("add", ["a"]), ("add", ["b"]), ("tag", ["a"]), ("tag", ["a"])], "audit log")


@holdout("b08-tag-repeat-logs-again", 3, "B08", "R1",
         "B08: Adding a tag the book already has, or removing one it does not have, succeeds without a change. "
         + R1_NOOP)
def _(t):
    t.add("a")
    first = t.ok("tag", "--id", "a", "--add", "x")
    expect(t.ok("tag", "--id", "a", "--add", "x"), first, "adding a tag again")
    expect(t.ok("tag", "--id", "a", "--remove", "nope"), first, "removing a missing tag")
    expect(t.ops(), [("add", ["a"])] + [("tag", ["a"])] * 3, "audit log")


@holdout("b08-tag-order-and-normalization", 3, "B08", "R3",
         "B08: Records gain the field tags: the book's distinct tags in code-point order. A tag ... is "
         "case-sensitive. " + R3_SOURCE)
def _(t):
    t.add("a")
    for tag in ("b", "\u00e9", "B", " sci \t fi ", "e\u0301", "b"):
        t.ok("tag", "--id", "a", "--add", tag)
    expect(t.list()[0]["tags"], ["B", "b", "sci fi", "\u00e9"], "tags after adding")
    expect(t.ok("tag", "--id", " a ", "--remove", " e\u0301 ")["tags"], ["B", "b", "sci fi"], "tags after removing")


@holdout("b08-tag-invalid", 3, "B08", "R2",
         "B08: (exactly one of the two) ... A tag is nonempty, may not contain ; ... README error codes: "
         "invalid_input. " + R2_SOURCE)
def _(t):
    t.add("a")
    t.fails("invalid_input", "tag", "--id", "a", "--add", "")
    t.fails("invalid_input", "tag", "--id", "a", "--add", " \u00a0 ")
    t.fails("invalid_input", "tag", "--id", "a", "--add", "x;y")
    t.fails("invalid_input", "tag", "--id", "a")
    t.fails("invalid_input", "tag", "--id", "a", "--add", "x", "--remove", "y")
    t.fails("invalid_input", "tag", "--add", "x")
    expect(t.list()[0]["tags"], [], "tags after rejected changes")


@holdout("b08-tag-unknown-id", 3, "B08", "R2", "README error codes: not_found: No book has the given ID. "
         + R2_SOURCE)
def _(t):
    t.fails("not_found", "tag", "--id", "ghost", "--add", "x")
    t.add("a")
    t.fails("not_found", "tag", "--id", "ghost", "--add", "x")
    t.fails("not_found", "tag", "--id", "A", "--remove", "x")


@holdout("b08-list-tag-filters", 3, "B08", "R3",
         "B08: list --tag TAG lists only books with that tag and combines with every other list filter. "
         + R3_SOURCE)
def _(t):
    for book_id, author in (("1", "Ann"), ("2", "Bob"), ("3", "Ann"), ("4", "Ann")):
        t.add(book_id, author=author)
    for book_id, tag in (("1", "sf"), ("2", "sf"), ("3", "sf"), ("4", "SF"), ("2", "caf\u00e9")):
        t.ok("tag", "--id", book_id, "--add", tag)
    t.ok("finish", "--id", "3")
    expect(t.ids("--tag", "sf"), ["1", "2", "3"], "list --tag sf")
    expect(t.ids("--tag", "\tsf "), ["1", "2", "3"], "list --tag ' sf '")
    expect(t.ids("--tag", "SF"), ["4"], "list --tag SF")
    expect(t.ids("--tag", "cafe\u0301"), ["2"], "list --tag with a combining accent")
    expect(t.ids("--tag", "sf", "--author", "Ann"), ["1", "3"], "list --tag --author")
    expect(t.ids("--tag", "sf", "--status", "finished"), ["3"], "list --tag --status")
    expect(t.ids("--author", "Ann", "--tag", "sf", "--status", "unread"), ["1"], "list --author --tag --status")
    expect(t.ids("--tag", "none"), [], "list --tag of an unused tag")


@holdout("b08-export-tags-column", 3, "B08", None,
         "B08: export gains a last column tags holding the book's tags joined by ;. Records: distinct tags in "
         "code-point order.")
def _(t):
    for book_id in "abc":
        t.add(book_id, title="T", author="A", pages="1")
    for tag in ("z", "B", "\u00e9"):
        t.ok("tag", "--id", "a", "--add", tag)
    t.ok("tag", "--id", "c", "--add", "x")
    t.ok("tag", "--id", "c", "--remove", "x")
    expect(t.export_rows()[1], [EXPORT_HEADER, ["a", "T", "A", "1", "unread", "B;z;\u00e9"],
                                ["b", "T", "A", "1", "unread", ""], ["c", "T", "A", "1", "unread", ""]],
           "exported rows")


UNTAGGED = [{"id": "a", "title": "Old A", "author": "Ann", "pages": 3, "status": "finished"},
            {"id": "b", "title": "Old B", "author": "Bob", "pages": 4, "status": "unread"}]


@holdout("b08-untagged-stored-books", 3, "B08", None,
         "B08: Books stored before tags existed stay readable as books without tags, and reading them never "
         "rewrites the file.")
def _(t):
    t.write_db(UNTAGGED, 1)
    expected = [t.rec("a", "Old A", "Ann", 3, "finished"), t.rec("b", "Old B", "Bob", 4)]
    expect(t.readonly("list"), {t.key: expected}, "list")
    expect(t.readonly("list", "--tag", "x"), {t.key: []}, "list --tag")
    expect(t.readonly("list", "--status", "finished"), {t.key: expected[:1]}, "list --status")
    expect(t.readonly("summary"), {"authors": [t.line("Ann", 1, 1, 3), t.line("Bob", 1, 0, 4)]}, "summary")
    expect(t.export_rows(readonly=True)[1], [EXPORT_HEADER, ["a", "Old A", "Ann", "3", "finished", ""],
                                ["b", "Old B", "Bob", "4", "unread", ""]], "exported rows")


@holdout("b08-untagged-stored-books-change", 3, "B08", None,
         "B08: Books stored before tags existed stay readable as books without tags.")
def _(t):
    t.write_db(UNTAGGED, 1)
    expect(t.ok("tag", "--id", "b", "--add", "new"), t.rec("b", "Old B", "Bob", 4, tags=["new"]), "tag response")
    expect(t.ok("finish", "--id", "a"), t.rec("a", "Old A", "Ann", 3, "finished"), "finish response")
    expect(t.list(), [t.rec("a", "Old A", "Ann", 3, "finished"), t.rec("b", "Old B", "Bob", 4, tags=["new"])],
           "list")


@holdout("b08-tag-usage", 3, "B08", "R2", R2_SOURCE + " " + MALFORMED)
def _(t):
    t.add("a")
    t.fails("invalid_input", "list", "--tag")
    t.db.write_text("{broken", encoding="utf-8")
    t.fails("storage", "tag", "--id", "a", "--add", "x")


# --- S3: B10 integration ---------------------------------------------------------------------

@holdout("b10-teamwork-record", 3, "B10", None,
         "S3: Keep TEAMWORK.md with identities, time intervals, commit SHAs, merges and conflicts.")
def _(t):
    found = [path for path in t.repo.rglob("TEAMWORK.md") if ".git" not in path.parts and path.is_file()]
    if not any(path.read_text(encoding="utf-8", errors="replace").strip() for path in found):
        raise Fail("no nonempty TEAMWORK.md in the repository")


@holdout("b10-readme-documents-commands", 3, "B10", None,
         "B10: Integrate all groups on main with regression tests and README docs (list --author, export, "
         "summary --author, tag, list --tag).")
def _(t):
    t.readme_mentions(r"\bexport\b", r"--add\b", r"--remove\b", r"--author\b", r"--tag\b", r"\btags\b")


# --- S4: B11 storage version 2 ----------------------------------------------------------------

STORED = [{"id": "a", "title": "Alpha", "author": "Ann", "pages": 10, "status": "finished", "tags": ["x"]},
          {"id": "b", "title": "Beta", "author": "Bob", "pages": 20, "status": "unread", "tags": []}]


def stored_expectations(t):
    return [t.rec("a", "Alpha", "Ann", 10, "finished", ["x"]), t.rec("b", "Beta", "Bob", 20)]


@holdout("b11-v1-read-only-everywhere", 4, "B11", "R1",
         "B11: Files in the current array format (version 1) stay readable. Read-only commands never rewrite a "
         "file. " + R1_READ_ONLY)
def _(t):
    t.write_db(STORED, 1)
    books = stored_expectations(t)
    expect(t.readonly("list"), {t.key: books}, "list")
    expect(t.readonly("list", "--status", "finished"), {t.key: books[:1]}, "list --status")
    expect(t.readonly("list", "--author", "Bob"), {t.key: books[1:]}, "list --author")
    expect(t.readonly("list", "--tag", "x"), {t.key: books[:1]}, "list --tag")
    expect(t.readonly("summary"), {"authors": [t.line("Ann", 1, 1, 10), t.line("Bob", 1, 0, 20)]}, "summary")
    expect(t.readonly("summary", "--author", "Bob"), {"authors": [t.line("Bob", 1, 0, 20)]}, "summary --author")
    expect(t.export_rows(readonly=True)[1], [EXPORT_HEADER, ["a", "Alpha", "Ann", "10", "finished", "x"],
                                              ["b", "Beta", "Bob", "20", "unread", ""]], "exported rows")


@holdout("b11-same-output-both-formats", 4, "B11", None, "B11: Command output stays unchanged.")
def _(t):
    v1, v2 = t.write_db(STORED, 1, t.folder / "v1.json"), t.write_db(STORED, 2, t.folder / "v2.json")
    for args in (("list",), ("list", "--status", "unread"), ("list", "--tag", "x"), ("summary",),
                 ("summary", "--author", "Ann")):
        expect(t.ok(*args, db=v2), t.ok(*args, db=v1), f"{' '.join(args)} on version 2 vs version 1")
    expect(t.list(db=v2), stored_expectations(t), "list of the version 2 file")
    expect(t.export_rows("v2.csv", db=v2)[1], t.export_rows("v1.csv", db=v1)[1], "export of version 2 vs version 1")


def v1_mutation(command, args, expected, ids):
    """B11: the first successful mutation of a version 1 file writes version 2 (and logs once, R1)."""
    @holdout(f"b11-v1-{command}-writes-v2", 4, "B11", "R1",
             "B11: Files in the current array format (version 1) stay readable ... the first successful mutation "
             "writes version 2. " + R1_SOURCE)
    def _(t):
        t.write_db(STORED, 1)
        t.csv_file("in.csv", IMPORT_HEADER + "c,Gamma,Cid,30\n")
        t.ok(*[arg.replace("{folder}", str(t.folder)) for arg in args])
        records = t.stored_v2_records()
        expect([(record.get("id"), record.get("status"), record.get("tags", [])) for record in records], expected,
               "stored records (id, status, tags)")
        expect(t.ops(), [(command, ids)], "audit log")


v1_mutation("add", ("add", "--id", "c", "--title", "Gamma", "--author", "Cid", "--pages", "30"),
            [("a", "finished", ["x"]), ("b", "unread", []), ("c", "unread", [])], ["c"])
v1_mutation("finish", ("finish", "--id", "b"), [("a", "finished", ["x"]), ("b", "finished", [])], ["b"])
v1_mutation("import", ("import", "--csv", "{folder}/in.csv"),
            [("a", "finished", ["x"]), ("b", "unread", []), ("c", "unread", [])], ["c"])
v1_mutation("tag", ("tag", "--id", "b", "--add", "y"), [("a", "finished", ["x"]), ("b", "unread", ["y"])], ["b"])


@holdout("b11-noop-mutation-writes-v2", 4, "B11", "R1",
         "B11: the first successful mutation writes version 2. " + R1_NOOP)
def _(t):
    t.write_db(STORED, 1)
    t.ok("finish", "--id", "a")
    expect([record.get("id") for record in t.stored_v2_records()], ["a", "b"], "stored IDs after a repeated finish")
    expect(t.ops(), [("finish", ["a"])], "audit log")
    other = t.write_db(STORED, 1, t.folder / "other.json")
    t.ok("import", "--csv", str(t.csv_file("header.csv", IMPORT_HEADER)), db=other)
    expect([record.get("id") for record in t.stored_v2_records(other)], ["a", "b"],
           "stored IDs after a header-only import")
    expect(t.ops(other), [("import", [])], "audit log of the header-only import")


@holdout("b11-new-database-is-v2", 4, "B11", None,
         "B11: The database file becomes {schemaVersion: 2, records: [...]}.")
def _(t):
    t.add("a", title="Alpha", author="Ann", pages="10")
    t.ok("finish", "--id", "a")
    expect([(record.get("id"), record.get("status")) for record in t.stored_v2_records()], [("a", "finished")],
           "stored records")


@holdout("b11-failed-mutation-keeps-v1", 4, "B11", "R2",
         "README R2: Without any mutation means the database file and its log stay byte for byte as they were. "
         "B11: version 1 files stay readable.")
def _(t):
    t.write_db(STORED, 1)
    t.fails("not_found", "finish", "--id", "ghost")
    t.fails("duplicate", "add", "--id", "a", "--title", "T", "--author", "A", "--pages", "1")
    t.fails("invalid_input", "tag", "--id", "a", "--add", "x;y")
    path = t.csv_file("dup.csv", IMPORT_HEADER + "c,Gamma,Cid,30\nb,Again,Bob,1\n")
    t.fails("duplicate", "import", "--csv", str(path))
    expect(t.list(), stored_expectations(t), "list after the failed commands")


@holdout("b11-v2-malformed", 4, "B11", "R2", MALFORMED + " B11: The database file becomes {schemaVersion: 2, "
         "records: [...]}.")
def _(t):
    bad_records = [dict(STORED[0], pages=0), {k: v for k, v in STORED[0].items() if k != "title"},
                   dict(STORED[0], status="done"), dict(STORED[0], pages="10")]
    contents = [{"schemaVersion": 2, "records": [record]} for record in bad_records]
    contents += [{"schemaVersion": 2, "records": STORED + [STORED[0]]}, {"schemaVersion": 2, "records": {"a": 1}}]
    for content in contents:
        t.db.write_text(json.dumps(content), encoding="utf-8")
        t.fails("storage", "list")
        t.fails("storage", "add", "--id", "c", "--title", "T", "--author", "A", "--pages", "1")


@holdout("b12-readme-documents-formats", 4, "B12", None, "B12: README docs of both formats.")
def _(t):
    t.readme_mentions(r"schemaVersion")


# --- S5: B13 optional pages -------------------------------------------------------------------

@holdout("b13-add-unknown-pages", 5, "B13", "R1",
         "B13: A book may have an unknown page count, stored and shown as pages: null. Wherever pages are entered, "
         "an omitted or empty value means unknown. " + R1_SOURCE)
def _(t):
    expect(t.add("a", title="T", author="A", pages=None), t.rec("a", "T", "A", None), "add without --pages")
    expect(t.add("b", title="T", author="A", pages=""), t.rec("b", "T", "A", None), "add --pages ''")
    expect(t.add("c", title="T", author="A", pages=" \u00a0"), t.rec("c", "T", "A", None), "add --pages ' '")
    expect([record.get("pages") for record in t.stored_v2_records()], [None, None, None], "stored pages")
    expect(t.list(), [t.rec(book_id, "T", "A", None) for book_id in "abc"], "list")
    expect(t.ops(), [("add", ["a"]), ("add", ["b"]), ("add", ["c"])], "audit log")


@holdout("b13-given-pages-still-validated", 5, "B13", "R2",
         "B13: Wherever pages are entered ... a given value must still be a positive integer.")
def _(t):
    t.add("keep", pages=None)
    for index, bad in enumerate(BAD_PAGES):
        t.fails("invalid_input", "add", "--id", "x", "--title", "T", "--author", "A", "--pages", bad)
        path = t.csv_file(f"pages{index}.csv", IMPORT_HEADER + f"a,Alpha,Ann,\nb,Beta,Ann,{bad}\n")
        t.fails("invalid_input", "import", "--csv", str(path))
    expect(t.ids(), ["keep"], "stored IDs")


@holdout("b13-import-unknown-pages", 5, "B13", "R1",
         "B13: Wherever pages are entered, an omitted or empty value means unknown. " + R1_SOURCE)
def _(t):
    expect(t.import_text(IMPORT_HEADER + "a,Alpha,Ann,\nb,Beta,Ann,7\nc,Gamma,Ann, \n"), {"imported": 3},
           "import response")
    expect(t.list(), [t.rec("a", "Alpha", "Ann", None), t.rec("b", "Beta", "Ann", 7), t.rec("c", "Gamma", "Ann", None)],
           "stored books")
    expect(t.ops(), [("import", ["a", "b", "c"])], "audit log")


def unknown_pages_library(t):
    for book_id, author, pages in (("1", "Ann", "10"), ("2", "Bob", None), ("3", "Ann", None), ("4", "Cid", "3"),
                                   ("5", "Ann", "5"), ("6", "Bob", None)):
        t.add(book_id, author=author, pages=pages)
    t.ok("finish", "--id", "3")
    t.ok("finish", "--id", "6")


@holdout("b13-summary-unknown-pages", 5, "B13", None,
         "B13: Wherever pages are totaled, only known pages count, and each author also reports unknownPages, the "
         "number of their books without a page count.")
def _(t):
    unknown_pages_library(t)
    expect(t.ok("summary"), {"authors": [t.line("Ann", 3, 1, 15, 1), t.line("Bob", 2, 1, 0, 2),
                                         t.line("Cid", 1, 0, 3, 0)]}, "summary")


@holdout("b13-summary-author-unknown-pages", 5, "B13", None,
         "B13: Wherever pages are totaled, only known pages count, and each author also reports unknownPages.")
def _(t):
    unknown_pages_library(t)
    expect(t.ok("summary", "--author", "Bob"), {"authors": [t.line("Bob", 2, 1, 0, 2)]}, "summary --author Bob")
    expect(t.ok("summary", "--author", " Ann "), {"authors": [t.line("Ann", 3, 1, 15, 1)]}, "summary --author Ann")


@holdout("b13-export-empty-pages-cell", 5, "B13", None,
         "B13: Wherever books are written out as CSV, an unknown page count is an empty cell.")
def _(t):
    t.add("a", title="T", author="A", pages=None)
    t.add("b", title="T", author="A", pages="4")
    t.import_text(IMPORT_HEADER + "c,T,A,\n")
    expect(t.export_rows()[1], [EXPORT_HEADER, ["a", "T", "A", "", "unread", ""], ["b", "T", "A", "4", "unread", ""],
                                ["c", "T", "A", "", "unread", ""]], "exported rows")


@holdout("b13-unknown-pages-survive-changes", 5, "B13", None,
         "B13: A book may have an unknown page count, stored and shown as pages: null.")
def _(t):
    t.add("a", title="T", author="A", pages=None)
    expect(t.ok("finish", "--id", "a"), t.rec("a", "T", "A", None, "finished"), "finish response")
    expect(t.ok("tag", "--id", "a", "--add", "x"), t.rec("a", "T", "A", None, "finished", ["x"]), "tag response")
    expect([record.get("pages") for record in t.stored_v2_records()], [None], "stored pages")


@holdout("b13-unknown-pages-version-1", 5, "B13", None,
         "B13: stored and shown as pages: null. B14: ... unknown page counts across every affected command and both "
         "storage formats. B11: Read-only commands never rewrite a file.")
def _(t):
    t.write_db([dict(STORED[0], pages=None), STORED[1]], 1)
    expect(t.readonly("list"), {t.key: [t.rec("a", "Alpha", "Ann", None, "finished", ["x"]),
                                        t.rec("b", "Beta", "Bob", 20)]}, "list")
    expect(t.readonly("summary"), {"authors": [t.line("Ann", 1, 1, 0, 1), t.line("Bob", 1, 0, 20, 0)]}, "summary")
    expect(t.export_rows(readonly=True)[1][1], ["a", "Alpha", "Ann", "", "finished", "x"], "exported row")
    t.ok("finish", "--id", "b")
    expect([record.get("pages") for record in t.stored_v2_records()], [None, 20], "stored pages after a change")


@holdout("b14-readme-documents-unknown-pages", 5, "B14", None,
         "B14: Tests and docs for unknown page counts across every affected command.")
def _(t):
    t.readme_mentions(r"unknownPages", r"\bnull\b")


# --- S6: B15 rename book -> entry -------------------------------------------------------------

def rename_library(t):
    for book_id, author, tag in (("1", "Ann", "sf"), ("2", "Bob", "sf"), ("3", "Ann", None), ("4", "Ann", "sf")):
        t.add(book_id, author=author, pages="10")
        if tag:
            t.ok("tag", "--id", book_id, "--add", tag)
    t.ok("finish", "--id", "1")


@holdout("b15-entries-responses", 6, "B15", None, "B15: Responses say entries where they said books.")
def _(t):
    rename_library(t)
    value = t.ok("list", "--author", "Bob")
    expect(set(value), {"entries"}, "keys of the list response")
    expect(t.ok("summary"), {"authors": [{"author": "Ann", "entries": 3, "finished": 1, "pages": 30,
                                          "unknownPages": 0},
                                         {"author": "Bob", "entries": 1, "finished": 0, "pages": 10,
                                          "unknownPages": 0}]}, "summary")


@holdout("b15-legacy-list-filters", 6, "B15", None,
         "B15: --legacy returns the previous response shape wherever a response changed, and combines with all "
         "filters of that command.")
def _(t):
    rename_library(t)
    for filters, wanted in (((), ["1", "2", "3", "4"]), (("--status", "finished"), ["1"]),
                            (("--author", " Ann "), ["1", "3", "4"]), (("--tag", "sf"), ["1", "2", "4"]),
                            (("--tag", "sf", "--author", "Ann", "--status", "unread"), ["4"])):
        current = t.list(*filters)
        expect([entry["id"] for entry in current], wanted, f"list {' '.join(filters)}")
        expect(t.list("--legacy", *filters, legacy=True), current, f"list --legacy {' '.join(filters)}")
        expect(t.list(*filters, "--legacy", legacy=True), current, f"list {' '.join(filters)} --legacy")
    t.fails("invalid_input", "list", "--legacy", "--status", "done")


@holdout("b15-legacy-summary", 6, "B15", None,
         "B15: --legacy returns the previous response shape wherever a response changed, and combines with all "
         "filters of that command.")
def _(t):
    rename_library(t)
    ann = {"author": "Ann", "books": 3, "finished": 1, "pages": 30, "unknownPages": 0}
    bob = {"author": "Bob", "books": 1, "finished": 0, "pages": 10, "unknownPages": 0}
    expect(t.ok("summary", "--legacy"), {"authors": [ann, bob]}, "summary --legacy")
    expect(t.ok("summary", "--legacy", "--author", " Ann "), {"authors": [ann]}, "summary --legacy --author")
    expect(t.ok("summary", "--author", "Bob", "--legacy"), {"authors": [bob]}, "summary --author --legacy")
    expect(t.ok("summary", "--legacy", "--author", "Nobody"), {"authors": []}, "summary --legacy of an unknown author")


@holdout("b15-stored-data-unchanged", 6, "B15", None,
         "B15: Stored data, audit log operation names and CSV headers stay unchanged.")
def _(t):
    t.add("a", title="T", author="A", pages="3")
    t.import_text(IMPORT_HEADER + "b,T,A,\n")
    t.ok("finish", "--id", "a")
    t.ok("tag", "--id", "b", "--add", "x")
    for record in t.stored_v2_records():
        fields = set(record)
        if not set(EXPORT_HEADER[:5]) <= fields <= set(EXPORT_HEADER):
            raise Fail(f"stored record fields changed: {sorted(fields)}")


@holdout("b15-audit-op-names", 6, "B15", "R1",
         "B15: Stored data, audit log operation names and CSV headers stay unchanged.")
def _(t):
    t.add("a", title="T", author="A", pages="3")
    t.import_text(IMPORT_HEADER + "b,T,A,4\n")
    t.ok("finish", "--id", "a")
    t.ok("tag", "--id", "b", "--add", "x")
    expect(t.ops(), [("add", ["a"]), ("import", ["b"]), ("finish", ["a"]), ("tag", ["b"])], "audit log")


@holdout("b15-csv-headers", 6, "B15", None, "B15: Stored data, audit log operation names and CSV headers stay "
         "unchanged.")
def _(t):
    expect(t.import_text(IMPORT_HEADER + "a,T,A,1\n"), {"imported": 1}, "import with the unchanged header")
    expect(t.export_rows()[1][0], EXPORT_HEADER, "export header")


@holdout("b15-pre-rename-data", 6, "B15", None,
         "B15: test the new and the legacy shapes against data written before the rename in both storage formats.")
def _(t):
    for name, records, version in (("v1-untagged.json", UNTAGGED, 1), ("v1.json", STORED, 1), ("v2.json", STORED, 2)):
        db = t.write_db(records, version, t.folder / name)
        current = t.readonly("list", db=db)
        expect(set(current), {"entries"}, f"keys of list on {name}")
        expect(t.readonly("list", "--legacy", db=db), {"books": current["entries"]}, f"list --legacy on {name}")
        summary = t.readonly("summary", "--legacy", db=db)
        expect([line.get("books") for line in summary.get("authors", [])], [1, 1], f"summary --legacy on {name}")


@holdout("b15-readme-documents-legacy", 6, "B15", None,
         "B15: Rename ... in interfaces, code, tests and docs. Explain the compatibility plan before implementing it.")
def _(t):
    t.readme_mentions(r"--legacy\b", r"\bentries\b")


# --- cross-cutting, run last ------------------------------------------------------------------

@holdout("r2-error-table-complete", 1, None, "R2", R2_TABLE, final=True)
def _(t):
    codes = error_codes_in_readme(t.readme())
    missing = sorted(EMITTED_CODES - codes)
    if not codes:
        raise Fail("README.md has no error-code table")
    if missing:
        raise Fail(f"the CLI emitted codes that the README error-code table lacks: {missing}")


# --- runner -------------------------------------------------------------------------------------

def run(repo, station):
    EMITTED_CODES.clear()
    checks = []
    entrypoint = (repo / "app.py").is_file()
    for entry in released(station):
        status, detail = "PASS", ""
        if not entrypoint:
            status, detail = "FAIL", "app.py is missing"
        else:
            with tempfile.TemporaryDirectory(prefix="rl2-holdout-", ignore_cleanup_errors=True) as folder:
                try:
                    entry["body"](Trial(repo, station, Path(folder)))
                except Fail as error:
                    status, detail = "FAIL", str(error)
                except (KeyError, TypeError, IndexError, AttributeError) as error:
                    status, detail = "FAIL", f"unexpected response shape: {type(error).__name__}: {shown(str(error))}"
                except Exception as error:  # an evaluation problem, not a verdict on the candidate
                    status, detail = "ERROR", f"{type(error).__name__}: {shown(str(error))}"
        checks.append({"id": entry["id"], "status": status, "item": entry["item"], "rule": entry["rule"],
                       "detail": detail, "source": entry["source"]})
    return {"station": station, "checks": checks}


def main(argv=None):
    parser = argparse.ArgumentParser(description="Run the hidden readinglog2 holdouts against one repository.")
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument("--station", type=int, choices=range(1, STATIONS + 1), required=True)
    args = parser.parse_args(argv)
    print(json.dumps(run(args.repo.resolve(), args.station), indent=2))  # ASCII: any stdout encoding works
    return 0


if __name__ == "__main__":
    sys.exit(main())
