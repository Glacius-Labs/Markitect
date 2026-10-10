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
| `add --id ID --title TITLE --author AUTHOR [--pages PAGES]` | Adds an unread entry and prints the new record. Without `--pages`, or with an empty value, the page count is unknown (`null`). |
| `finish --id ID` | Marks the entry as finished and prints the updated record. Repeating it succeeds with the same record. |
| `list [--status unread\|finished] [--author NAME] [--tag TAG] [--legacy]` | Prints the entries in insertion order: `{"entries": [RECORD, ...]}`. Each given filter narrows the list; an entry must match all of them. See [Compatibility](#compatibility) for `--legacy`. |
| `import --csv PATH` | Adds the entries of a CSV file, all or nothing, and prints `{"imported": COUNT}`. See [CSV import](#csv-import). |
| `summary [--author NAME] [--legacy]` | Prints `{"authors": [{"author": NAME, "entries": COUNT, "finished": COUNT, "pages": TOTAL, "unknownPages": COUNT}, ...]}`, one object per author in code-point order of the name. `entries` counts all of the author's entries, `finished` only the finished ones, `pages` sums the known page counts and `unknownPages` counts the entries without one. `--author` reports only that author; an unknown author gives `{"authors": []}`. |
| `tag --id ID (--add TAG \| --remove TAG)` | Adds or removes one tag and prints the updated record. Adding a tag the entry has, or removing one it lacks, succeeds without a change. |
| `export --csv PATH` | Writes every entry to a CSV file and prints `{"exported": COUNT}`. See [CSV export](#csv-export). |

Every successful command prints exactly one JSON object on standard output and exits 0.
Errors are described under [R2 Error contract](#r2-error-contract).

```
$ python -B app.py --db reading.json add --id dune --title "Dune" --author "Frank Herbert" --pages 412
{"id": "dune", "title": "Dune", "author": "Frank Herbert", "pages": 412, "status": "unread", "tags": []}
$ python -B app.py --db reading.json list
{"entries": [{"id": "dune", "title": "Dune", "author": "Frank Herbert", "pages": 412, "status": "unread", "tags": []}]}
```

## Records

An entry is a JSON object with exactly these fields:

| Field | Type | Meaning |
|---|---|---|
| `id` | string | Nonempty and unique. Identifies the entry in later commands. |
| `title` | string | Nonempty. |
| `author` | string | Nonempty. |
| `pages` | integer or `null` | Page count, at least 1, given as ASCII digits, for example `412`. `null` when the page count is unknown. |
| `status` | string | `unread` or `finished`. New entries are `unread`. |
| `tags` | list of strings | The entry's distinct tags in code-point order, `[]` for new entries. A tag is nonempty, case-sensitive and may not contain `;`. |

Entries keep the order in which they were added (insertion order). Listings use that
order.

## CSV import

`import --csv PATH` reads a UTF-8 CSV file whose header row is exactly
`id,title,author,pages`:

```
id,title,author,pages
dune,Dune,Frank Herbert,412
"emma","Emma, a novel",Jane Austen,474
```

- Every row follows the rules of `add` and is normalized the same way. An empty `pages`
  cell means the page count is unknown. Empty lines are skipped.
- The whole file is checked before anything is stored. One bad row (invalid value, too
  few or too many columns, an ID that exists already or appears twice in the file)
  rejects the whole import and changes nothing.
- On success the entries are appended as unread, in file order, and the audit line lists
  their IDs in that order. A file with only the header imports nothing and succeeds.
- A missing or unreadable file, or a missing or different header, fails with
  `invalid_input`.
- The import file has no tags column; imported entries start without tags.

## CSV export

`export --csv PATH` writes every entry in insertion order to a UTF-8 CSV file with the
header `id,title,author,pages,status,tags`. Text is quoted the standard CSV way (fields
with commas or quotes are enclosed in quotes, quotes are doubled). `tags` holds the
entry's tags joined by `;`, empty when it has none. An unknown page count is an empty
`pages` cell. Export only reads the database: it never changes it and writes no audit
line. A destination that cannot be written, including the database or its log, fails
with `export_failed`.

## Compatibility

Earlier versions called an entry a book. The rename changed two response keys:
`list` returned `{"books": [...]}` and each `summary` author object counted `books`.
For one compatibility period both commands accept `--legacy`, which returns exactly
the previous shape and combines with all of the command's filters:

```
$ python -B app.py --db reading.json list --legacy --status finished
{"books": [...]}
$ python -B app.py --db reading.json summary --legacy --author "Frank Herbert"
{"authors": [{"author": "Frank Herbert", "books": 1, "finished": 0, "pages": 412, "unknownPages": 0}]}
```

Nothing else changed: stored data in both formats, record fields, audit log operation
names (`add`, `finish`, `import`, `tag`), the CSV headers and the other responses stay as
they were, and data written before the rename is read without being rewritten.
`--legacy` will be removed after the compatibility period.

## Project rules

These rules hold for every command, including commands added later. Backlog items do
not repeat them.

### R1 Audit log

Every successful mutating command appends exactly one JSON line to `<db>.log`:
`{"op": COMMAND, "ids": [IDS in processing order], "at": UTC ISO-8601}`. Failed
commands and read-only commands append nothing.

- `<db>.log` is the database path with `.log` appended: `--db data/reading.json` logs to
  `data/reading.json.log`.
- A command is mutating when it can change stored entries (`add`, `finish`, `import`,
  `tag`). A command that only reads entries is read-only (`list`, `summary`, `export`),
  even when it writes some other file.
- `op` is the command name. `ids` lists the IDs of the entries the command processed, in
  the order it processed them, and is `[]` when there were none. `at` is the UTC time
  with a `Z` suffix, for example `2026-10-10T08:15:30Z`.
- A successful mutating command saves the database and logs its line even when no entry
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

- This covers every value that becomes or selects entry data: IDs, titles, authors,
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
| `not_found` | No entry has the given ID. |
| `storage` | The database file cannot be read or written, or it is malformed. |
| `export_failed` | The export destination cannot be written. |

## Storage

- The database is one UTF-8 JSON file. The current format, version 2, is an object
  with exactly two fields, the entry records in insertion order and the format version:

  ```
  {"schemaVersion": 2, "records": [RECORD, ...]}
  ```

- Version 1, a plain array of entry records (`[RECORD, ...]`), stays readable. Read-only
  commands never rewrite such a file; the first successful mutation saves it as
  version 2. Command output is the same for both formats. Any other shape, including
  another `schemaVersion`, is malformed.
- A missing file is an empty database. Read-only commands never create or change the
  file. The first successful change creates it in version 2.
- Every change is saved by writing a temporary file next to the database and then
  atomically replacing the database with it. Another process sees either the old or
  the new file, never a partial one, and a second process on the same database sees
  every committed change.
- Both formats may hold `"pages": null` for an unknown page count.
- Entries stored before tags existed have no `tags` field. They read as entries without
  tags; reading never rewrites them, and the next save stores `tags` for every entry.
- A file that exists but is not valid UTF-8 JSON, does not have the expected shape or
  holds an invalid record (missing or extra fields, wrong types, empty text, pages
  below 1, an unknown status, invalid tags, a repeated ID) is malformed. Every command
  then fails with `storage`, and the file is never overwritten.

## Limitations

- One writer at a time: there is no locking between processes that change the same
  database concurrently.
- Saving the database and appending the log line are two steps. A crash between them
  can leave a saved change without its log line.

## Teamwork

[TEAMWORK.md](TEAMWORK.md) records who built the tags, query, export and reporting
changes and how they were merged.

## Development

```
python -B -m unittest discover -s tests
```

The tests run the CLI as a separate process on temporary files.
