# Backlog: Readinglog

Only `.study/station.json` releases work. All requirements are public: README.md
describes the tool, its project rules, error codes and storage, and this backlog the
planned changes. Keep the working `add`/`list` baseline and stored books working. Each
item includes its tests and docs and is integrated into `main`; choose the internal
organization freely.

## S1: one item

- B01: `finish --id ID` marks the book as finished and prints the updated record.
  Finishing a book that is already finished succeeds again with the same record. A
  missing ID is an error. `list --status unread|finished` lists only books with that
  status, in insertion order; any other status value is invalid. `list` without
  `--status` stays unchanged.

## S2: three items

- B02 (B01): `import --csv PATH` adds books from a UTF-8 CSV file whose header row is
  exactly `id,title,author,pages`. Every row follows the rules of `add`, and no ID may
  appear twice in the file or exist already. The import is all or nothing: if any row
  is invalid, nothing is imported. On success the books are appended as unread in file
  order and the command prints `{"imported": COUNT}`. A file with only the header
  imports nothing and succeeds. Empty lines are ignored. A missing or unreadable file,
  a missing or different header, or a row with too few or too many columns is invalid
  input.
- B03 (B01): `summary` prints `{"authors": [{"author": NAME, "books": COUNT,
  "finished": COUNT, "pages": TOTAL}, ...]}`, one object per author, ordered by author
  name in plain code-point order. `books` and `pages` count all of the author's books,
  `finished` only the finished ones. An empty database gives `{"authors": []}`.
- B04 (B02, B03): Integrate: regression tests for the baseline and the new commands,
  and README usage, failure and stored-state docs for them.

## S3: six items, actual team wave

At least two executing agents must overlap in time on independent groups, and both
contributions are merged. Native roles, prompts, planning and branches are yours. Keep
TEAMWORK.md with identities, time intervals, commit SHAs, merges and conflicts.

- B05 (B04; query group): `list --author NAME` lists only that author's books, in
  insertion order, and combines with `--status`.
- B06 (B04; export group): `export --csv PATH` writes every book to a UTF-8 CSV file
  with the header `id,title,author,pages,status`, in insertion order and with standard
  CSV quoting, and prints `{"exported": COUNT}`. It never changes the database. If the
  destination cannot be written, the command fails with the new error code
  `export_failed`.
- B07 (B04; reporting group): `summary --author NAME` reports only that author; an
  unknown author gives `{"authors": []}`. `summary` without `--author` stays unchanged.
- B08 (B04; tags group): Books get tags. `tag --id ID --add TAG` or
  `tag --id ID --remove TAG` (exactly one of the two) changes one book's tags and
  prints the updated record. Adding a tag the book already has, or removing one it
  does not have, succeeds without a change. Records gain the field `tags`: the book's
  distinct tags in code-point order, `[]` for new books. A tag is nonempty, may not
  contain `;` and is case-sensitive. `list --tag TAG` lists only books with that tag
  and combines with every other `list` filter. `export` gains a last column `tags`
  holding the book's tags joined by `;`. Books stored before tags existed stay readable
  as books without tags, and reading them never rewrites the file.
- B09 (B05, B06, B07, B08): Tests per group: filter combinations and order across
  processes; export quoting, tags column, empty database, unchanged database and an
  unwritable destination; summary for one author and for unknown authors; repeated
  tagging, unknown IDs, invalid tags and books stored without tags.
- B10 (B09): Integrate all groups on `main` with regression tests and README docs, and
  complete TEAMWORK.md with the real team, merge and conflict evidence. No architecture
  is prescribed.

## S4: storage refactoring

- B11 (B10): Storage format version 2. The database file becomes
  `{"schemaVersion": 2, "records": [...]}`. Files in the current array format
  (version 1) stay readable. Read-only commands never rewrite a file; the first
  successful mutation writes version 2. Command output stays unchanged.
- B12 (B11): Tests that run every command against version 1 files, and README docs of
  both formats.

## S5: late decision change

- B13 (B12): Pages become optional. A book may have an unknown page count, stored and
  shown as `"pages": null`. Wherever pages are entered, an omitted or empty value
  means unknown; a given value must still be a positive integer. Wherever pages are
  totaled, only known pages count, and each author also reports `unknownPages`, the
  number of their books without a page count. Wherever books are written out as CSV,
  an unknown page count is an empty cell.
- B14 (B13): Tests and docs for unknown page counts across every affected command and
  both storage formats.

## S6: final domain rename

- B15 (B14): Rename the active domain term book to entry in interfaces, code, tests and
  docs. Responses say `entries` where they said `books`. For one compatibility period,
  `--legacy` returns the previous response shape wherever a response changed, and
  combines with all filters of that command. Stored data, audit log operation names
  and CSV headers stay unchanged. Explain the compatibility plan before implementing
  it, then test the new and the legacy shapes against data written before the rename
  in both storage formats. Historic notes and deliberate aliases are fine; a text
  search alone is not acceptance.

15 items in six waves (1/3/6/2/2/1). Do not implement unreleased waves.
