# Work log

- S1 / R01: booking/list validation, half-open overlaps, ordering, durable state, JSON errors, tests, and docs; merged to `main` at `a765087`.
- S2 / R02-R04: cancellation/history/interval reuse, active-only exact-minute summary, regressions, and run/failure/state/limitation docs; merged at `7b3f782`.
- S3 / R05-R11: status/room filters, CSV export, filtered summary, three overlapping helper groups, integrated tests/docs, and TEAMWORK evidence; merged at `1073832`.
- S4 / R12: documented response compatibility before code changes, renamed the default list envelope to `bookings`, added temporary `list --legacy`, and changed internal state/code/tests/docs to booking terminology while retaining the raw JSON storage key.
- S4 compatibility test uses a pre-change JSON file, checks both list interfaces and composed filters without byte changes, exercises book/cancel/summary/export, and confirms returned fields, IDs, and storage compatibility after mutation.
- Checks and repair: initial test filter used lowercase `atlas`; corrected it to `Atlas`. The full 25-test suite and 13 S4 public checks passed after merge at `1cc7b37acd20905add3defd2d4c910d8d08547a0`. Logs: `.work/logs/post-merge-unit-s4.log`, `.work/logs/post-merge-acceptance-s4.log`; first failing attempt: `.work/logs/unit-s4-first.log`.
- Integration: feature commit `c1dbc33` merged via `--no-ff` to isolated `main` at `1cc7b37acd20905add3defd2d4c910d8d08547a0`.
- Unresolved decisions: none known. Human acceptance remains separate from these finite checks.
