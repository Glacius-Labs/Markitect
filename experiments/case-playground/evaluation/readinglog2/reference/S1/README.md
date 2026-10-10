# Readinglog

Readinglog keeps a personal reading list in one JSON file. It is a small command-line
tool in Python (3.11 or newer, standard library only, no network). Planned work is in
[BACKLOG.md](BACKLOG.md).

## Usage

```
python -B app.py --db PATH COMMAND [OPTIONS]
```

`--db PATH` names the database file. It is created by the first successful change.

| Command | Result |
|---|---|
| `add --id ID --title TITLE --author AUTHOR --pages PAGES` | Adds an unread book and prints the new record. |
| `finish --id ID` | Marks the book as finished and prints the updated record. Repeating it succeeds with the same record. |
| `list [--status unread\|finished]` | Prints the books in insertion order, optionally only those with the given status: `{"books": [RECORD, ...]}`. |

Every successful command prints exactly one JSON object on standard output and exits 0.
Errors are described under [R2 Error contract](#r2-error-contract).

```
$ python -B app.py --db books.json add --id dune --title "Dune" --author "Frank Herbert" --pages 412
{"id": "dune", "title": "Dune", "author": "Frank Herbert", "pages": 412, "status": "unread"}
$ python -B app.py --db books.json list
{"books": [{"id": "dune", "title": "Dune", "author": "Frank Herbert", "pages": 412, "status": "unread"}]}
```

## Records

A book is a JSON object with exactly these fields:

| Field | Type | Meaning |
|---|---|---|
| `id` | string | Nonempty and unique. Identifies the book in later commands. |
| `title` | string | Nonempty. |
| `author` | string | Nonempty. |
| `pages` | integer | Page count, at least 1. Given as ASCII digits, for example `412`. |
| `status` | string | `unread` or `finished`. New books are `unread`. |

Books keep the order in which they were added (insertion order). Listings use that
order.

## Project rules

These rules hold for every command, including commands added later. Backlog items do
not repeat them.

### R1 Audit log

Every successful mutating command appends exactly one JSON line to `<db>.log`:
`{"op": COMMAND, "ids": [IDS in processing order], "at": UTC ISO-8601}`. Failed
commands and read-only commands append nothing.

- `<db>.log` is the database path with `.log` appended: `--db data/books.json` logs to
  `data/books.json.log`.
- A command is mutating when it can change stored books (`add`, `finish`). A command
  that only reads books is read-only (`list`), even when it writes some other file.
- `op` is the command name. `ids` lists the IDs of the books the command processed, in
  the order it processed them, and is `[]` when there were none. `at` is the UTC time
  with a `Z` suffix, for example `2026-10-10T08:15:30Z`.
- A successful mutating command saves the database and logs its line even when no book
  actually changed.
- The tool only appends to the log. It never reads or rewrites it.

Example line: `{"op": "add", "ids": ["dune"], "at": "2026-10-10T08:15:30Z"}`

### R2 Error contract

Every domain/input error prints `{"error": {"code": CODE, "message": TEXT}}` and exits
2 without any mutation. Codes come from the [error-code table](#error-codes). A new
code must be added to that table.

- This includes usage errors: an unknown command or option, a missing argument or a
  value that is not allowed.
- "Without any mutation" means the database file and its log stay byte for byte as
  they were, and a missing database file is not created.
- `TEXT` explains the problem for people. Scripts should rely on `CODE` only.

### R3 Text normalization

All text input, from arguments or files, is Unicode NFC normalized, stripped, and
internal whitespace runs collapse to one space. IDs are case-sensitive after
normalization.

- This covers every value that becomes or selects book data: IDs, titles, authors,
  page counts, statuses and filter values. File paths such as `--db` are used exactly
  as given.
- Whitespace means any Unicode whitespace: spaces, tabs, line breaks, no-break spaces.
- A value that must not be empty is invalid when it is empty after normalization.
- Comparisons are exact after normalization: `Dune` and `dune` are different IDs, while
  `é` written as one character or as `e` plus a combining accent is the same ID.

## Error codes

| Code | Meaning |
|---|---|
| `invalid_input` | A command, option or value is missing, empty after normalization or not allowed, for example pages that are not a positive integer. Also used for an input file that cannot be read or has the wrong format. |
| `duplicate` | An ID is already taken: it exists in the database or appears twice in the same input. |
| `not_found` | No book has the given ID. |
| `storage` | The database file cannot be read or written, or it is malformed. |

## Storage

- The database is one UTF-8 JSON file holding an array of book records in insertion
  order.
- A missing file is an empty database. Read-only commands never create or change the
  file.
- Every change is saved by writing a temporary file next to the database and then
  atomically replacing the database with it. Another process sees either the old or
  the new file, never a partial one, and a second process on the same database sees
  every committed change.
- A file that exists but is not valid UTF-8 JSON, does not have the expected shape or
  holds an invalid record (missing or extra fields, wrong types, empty text, pages
  below 1, an unknown status, a repeated ID) is malformed. Every command then fails
  with `storage`, and the file is never overwritten.

## Limitations

- One writer at a time: there is no locking between processes that change the same
  database concurrently.
- Saving the database and appending the log line are two steps. A crash between them
  can leave a saved change without its log line.

## Development

```
python -B -m unittest discover -s tests
```

The tests run the CLI as a separate process on temporary files.
