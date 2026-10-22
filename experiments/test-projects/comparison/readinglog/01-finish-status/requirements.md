# Readinglog finish and status requirements

Maintain the existing local Readinglog application for one operator. Its current
public commands are `add`, `list` and `finish`. Preserve their command names, book record
shape and default listing behavior while meeting all requirements below. The
supplied add/list application and public baseline test are the actual starting
realization, not a prepared final implementation or proof of conformance.

Use Python 3.13 and its standard library. No network or external package is
needed. Internal code, package and test organization are implementation choices
within the fixed public interface.

## Books and shared input rules

A book has exactly these fields:

| Field | Type and meaning |
| --- | --- |
| `id` | Nonempty string; unique, compared exactly and case-sensitively. |
| `title` | Nonempty string. Titles need not be unique. |
| `author` | Nonempty string. Authors need not be unique. |
| `pages` | Positive integer, never a boolean. |
| `status` | Exactly `unread` or `finished`. |

Strip surrounding whitespace from command text input. Required `id`, `title`
and `author` values must then be nonempty. Compare these normalized identifiers
exactly and case-sensitively; differently cased identifiers remain different.
Pages must be a positive integer. Apply the same normalization wherever a
command accepts these fields. Adding a book creates it with `status: unread`. Finishing a book changes only
its status to `finished`; its identifier, title, author and pages remain unchanged.
No example books are pre-created.

Missing, invalid or duplicate inputs produce an understandable JSON error
through the interface below, with no stored-data change. Expected input and
domain failures do not cause an unhandled crash. Exact error wording is free.

## Persistent state

The explicit `--db PATH` selects a UTF-8 JSON array. A missing data file is
empty state. Before using or modifying existing data, validate that its root
is an array and every record has exactly the stated book keys. Stored `id`,
`title` and `author` values are strings containing non-whitespace text; pages
is a positive integer rather than a boolean; status is `unread` or `finished`;
and stored identifiers are unique under exact comparison.

Malformed existing data fails clearly through the error interface and is never
overwritten. Preserve existing records and insertion order. Reads do not rewrite
stored records or create a missing data file. Successful changes use atomic
replacement, and a later process using the same path observes committed results.
Errors preserve the data already stored. No guarantee for concurrent independent
writers is required.

## Add

The operator supplies `--id`, `--title`, `--author` and `--pages`. Normalize and
validate them using the shared input rules, then append exactly one book with
those values and unread status. Commit it atomically and return the created
record directly.

Refuse an identifier already stored, including a duplicate reached after command
input normalization. A refusal changes no data and replaces no existing record.
Other records and their insertion order remain unchanged.

## List

Return an object with a `books` array containing every stored book, including
both unread and finished records, in insertion order. A missing data file returns
an empty array. Malformed existing data follows the shared storage and error
rules. Listing is read-only: it never changes or rewrites the database.

An optional `--status unread|finished` filter selects exactly that status while
preserving insertion order and the same `books` envelope. With no filter,
include every status as before. An invalid filter is an input error under the
shared error contract. Filtering never changes or rewrites the database.

## Finish

The operator supplies `finish --id ID`. Normalize the identifier using the
shared input rules and find that exact, case-sensitive identifier in stored
data. Change only that book's status to `finished`, preserve all other fields,
other records and their order, commit through the shared atomic storage rules,
and return the finished record directly.

Repeating finish for an already finished book succeeds idempotently and returns
that record. A missing or unknown identifier produces an understandable JSON
error under the shared interface, with no stored-data change. Validate existing
data and handle malformed storage through the same shared rules.

## Command interface

Invoke the application from `realization/` as:

```text
python -B app.py --db PATH COMMAND ...
```

| Command | Output on success |
| --- | --- |
| `add --id ID --title TITLE --author AUTHOR --pages INTEGER` | The created book record directly. |
| `finish --id ID` | The finished book record directly. |
| `list [--status unread\|finished]` | An object with a `books` array in insertion order, including every status unless filtered. |

Every success writes one JSON object to standard output and exits 0. Every
domain or input error writes exactly one JSON object to standard output with
only an `error` field containing an understandable nonempty string, and exits 2.
Stored data remains unchanged on error. Command names,
record fields and the default list envelope remain compatible with this contract.

## Delivery

Deliver working code, meaningful command-line and regression tests, and concise
documentation explaining usage, failure and state behavior, and honest
limitations. Continue this arm's actual prior application, tests, documentation and stored
data; do not replace them with a supplied solution or restart from a template.
Preserve the accepted add, list, book, input and storage contracts while
completing the current intent, including finish and filtered listing.

From `realization/`, run tests with:

```text
python -B -m unittest discover -s tests
```

Internal code and test organization remain choices within the interface above.
A successful source check alone does not prove behavior or acceptance.
