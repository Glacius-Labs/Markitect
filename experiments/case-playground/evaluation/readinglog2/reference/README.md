# readinglog2 reference implementations

Hidden. Never staged into a run or shown to reviewers. `S1` to `S6` are the state of
`main` after each wave: `app.py`, `README.md`, `tests/` and, from S3, a `TEAMWORK.md`
placeholder (one author, no real team). Each tree satisfies every public requirement
released up to its wave and nothing later.

Validate from `experiments/case-playground`:

```
python -I -B cases/common/checks/acceptance.py --repo evaluation/readinglog2/reference/S3 --case readinglog2 --station 3
cd evaluation/readinglog2/reference/S3 && python -B -m unittest discover -s tests
```

Station n passes all public checks on `S<n>`; the seed baseline fails the S1 checks.

## Left open by the public text

The references make a choice here, but README.md and BACKLOG.md do not, so holdouts
must not test these:

- CSV import: whitespace or a byte-order mark in the header row; which error wins when
  a file has several bad rows.
- CSV import once B13 is released: a row or a header without the `pages` column (an
  omitted value means unknown, yet a row with too few columns is invalid; the reference
  rejects both as `invalid_input`).
- Lines holding only whitespace in an import file ("Empty lines are ignored" does not
  say whether they are empty; the reference rejects them as rows with too few columns).
- `--pages` given with no value at all once B13 is released (the reference treats it as
  a usage error, `invalid_input`).
- An empty `--author` or `--tag` filter value, such as `""` or only whitespace (the
  reference rejects it as `invalid_input`).
- Version 2 records without `tags` (B08 keeps books stored before tags existed readable,
  but version 2 came after tags; the reference reads them as books without tags).
- `list --tag` with a value containing `;` (the reference rejects it as `invalid_input`).
- Which error wins when one call has several problems, such as an invalid tag for an
  unknown ID.
- Extra top-level keys next to `schemaVersion` and `records` (the reference treats
  them as malformed).
- Exporting onto the database or its log (the reference refuses with `export_failed`).
- A partial export file left behind by a failed write; CSV line endings.
- `--legacy` on commands whose response did not change.
- The wording of error messages.
