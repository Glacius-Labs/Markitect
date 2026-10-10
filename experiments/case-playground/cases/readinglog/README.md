# Readinglog

This is an existing small Python 3.13 standard-library CLI. The baseline supports add and list over a JSON array. Preserve its command names, record shape and default listing semantics while completing BACKLOG.md. No network or external package is needed.

`python -B app.py --db PATH COMMAND ...`

Legacy commands:

- `add --id ID --title TITLE --author AUTHOR --pages INTEGER` creates a record `{id,title,author,pages,status:"unread"}` and returns it. IDs are nonempty and case-sensitive, unique; strip surrounding whitespace on text input, nonempty text required, pages must be a positive integer. Invalid or duplicate input returns `{"error":"..."}` and exit 2 without mutation.
- `list` returns `{"books":[...]}` in insertion order, including every status. Optional `--status unread|finished` is added by the backlog; the no-filter behavior must remain unchanged.

Backlog commands:

- `finish --id ID` sets status to `finished`, returns the record, and succeeds idempotently when repeated. A missing ID is an error.
- `import --csv PATH` imports UTF-8 CSV with exactly the header `id,title,author,pages`. Validate the entire file first, including required text, positive integer pages, duplicate IDs within the file or against existing records. On any invalid row reject the entire import without changing data. Success returns `{"imported":COUNT}` and appends unread books in input order. Empty file with a valid header imports zero; incorrect/missing/extra columns fail. Use the same normalization and case-sensitive identity as add.
- `summary` returns `{"authors":[{"author":NAME,"books":COUNT,"finished":COUNT,"pages":TOTAL},...]}` ordered lexically by author. Count all books and all their pages; finished counts only finished records.

All successes write one JSON object and exit 0; all domain/input errors write one error object and exit 2. Persist successful changes via atomic replacement; missing data file is empty state and malformed existing data must not be overwritten. A second process on the same data observes committed results. No concurrent multiwriter guarantee is required. Keep existing tests and add meaningful CLI/regression tests, usage, failure behavior and honest limitations.
