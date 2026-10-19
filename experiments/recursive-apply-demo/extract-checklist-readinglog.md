# Extract assessment checklist (Readinglog)

A fixed rubric for an extracted Readinglog model, the second project of the Extract bench. The source is the supplied Python application of `experiments/test-projects/readinglog/template/realization`: `app.py` and one baseline test, with no README. The reference is the human [00-baseline model](../test-projects/readinglog/snapshots/00-baseline/model/readinglog). Part of that reference states intent that the two files do not show. Those items are marked **N/D** (not derivable). They are scored only when the model states them anyway, and then as invented (wrong) unless the artifacts support them.

Score and list over-specifications exactly as in [the library checklist](extract-checklist.md): covered 2, partial 1, missing 0, wrong counted separately; categories X1–X6.

## Obligations

| ID | Obligation | Evidence |
| --- | --- | --- |
| L01 | A local command-line Readinglog records books through `add` and `list` | app.py |
| L02 | Invocation from `realization/` as `python app.py --db PATH COMMAND ...`; standard library only, no network or external package | app.py imports and parser |
| L03 | A book has exactly the fields id, title, author, pages, status; status is `unread` or `finished` | `load` |
| L04 | Command text is stripped; id, title and author must then be nonempty; pages is a positive integer | `main` |
| L05 | IDs are unique and compared exactly (case-sensitive, after stripping); names and authors need not be unique | `main`, test |
| L06 | `add` creates exactly one unread book, appends it after the existing records and returns the created record | `main`, test |
| L07 | `add` refuses a duplicate or invalid input without changing the stored data | `main`, test |
| L08 | `list` returns an object with `books` holding every stored record in insertion order; a missing file gives an empty list; `list` writes nothing | `main`, `load`, test |
| L09 | Storage is a UTF-8 JSON array at the `--db` path; a missing file is empty state | `load`, `save` |
| L10 | Existing data is validated before use (array root, exact keys, nonempty text, positive non-boolean integer pages, known status, unique IDs); malformed data fails without being overwritten | `load` |
| L11 | A successful mutation replaces the file atomically (temporary file, then replace) | `save` |
| L12 | Success prints one JSON object on standard output and exits 0 | `main` |
| L13 | A domain or input error prints one JSON object with an `error` string on standard output and exits 2, leaving the data unchanged | `main`, test |
| L14 | Command-line usage errors (missing or unknown arguments) are reported by the argument parser, not as a JSON object; the model states this as observed behavior or as an open point, not as the JSON error rule | argparse in `main` |
| L15 | The tests run as Python unittest from `realization/` (for example `python -m unittest discover -s tests`) | tests/ |
| L16 | Internal code and test organization and error wording are realization choices | DEC-061 default |
| N/D | Python 3.13 exactly; usage and limitation documentation; "a successful source check does not prove acceptance"; no concurrent-writer guarantee | reference only |

## Generality questions

Score each yes or no.

| ID | Question |
| --- | --- |
| G1 | The structure gives the application and its tests (directly in `realization/`) a Package owner: a layout exception that maps `realization` to that Package, or an explicit Root ownership with registrations that link the Package's Definitions |
| G2 | No owner question about a matter the code settles (there is no document, so no source conflict exists) |
| G3 | A proposed check runs the Python tests with a profile the Host offers, or no check is proposed and the summary says why |
| G4 | No statement contradicts the code |

## Assessor answer

One JSON object: `{"obligations": {"L01": {"score": 2, ...}, ...}, "wrong": [...], "invented_nd": [...], "overspecification": [...], "generality": {"G1": true, ...}, "summary": "..."}`.
