# Work log

- S1 / R01: booking/list behavior, validation, half-open conflicts, ordering, durable state, JSON errors, tests, and documentation; merged at `a765087`.
- S2 / R02-R04: idempotent cancel, history retention, interval reuse, active-only exact-minute summary, regression tests, operations/state/failure/limitation documentation; merged at `7b3f782`.
- S3 / R05-R11: status/room list filters, CSV export, room-filtered summaries, focused helper tests, integrated CLI tests, README updates, and actual teamwork evidence.
- Team: query, export, and summary helpers owned disjoint files and overlapped 08:14:33-08:14:45 UTC. `TEAMWORK.md` records identities, requests, intervals, contribution blob IDs, review/repairs, and merge evidence.
- Checks and repair: one test fixture abbreviated room `A room` to `A`; corrected the input. The integrated 24-test suite and 12 S3 acceptance checks passed on `main` merge `1073832a50d802cccbe5a4155c911897d90df4d1`. Focused helper tests also passed (4 query, 6 export, 4 summary). Logs are under `.work/logs/`.
- Integration: S3 feature commit `4e1fcca7ec6134fb54984a0f382e155801d8cc41` merged with `--no-ff` to isolated `main` at `1073832a50d802cccbe5a4155c911897d90df4d1`; no conflicts.
- Unresolved decisions: none known. Human acceptance remains separate from automated checks.
