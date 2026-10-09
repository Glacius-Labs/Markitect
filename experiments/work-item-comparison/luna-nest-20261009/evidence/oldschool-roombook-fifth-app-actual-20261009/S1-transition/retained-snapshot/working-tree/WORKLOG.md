# Work log

- Work item: S1 / R01, released by `.study/station.json`.
- Implementation: added `app.py` with JSON command errors, strict UTC-minute/date and required-text validation, normalized case-sensitive room filtering, half-open overlap checks, increasing IDs, start/id list ordering, strict stored-data validation, and UTF-8 atomic replacement. No unreleased commands were added.
- Tests: added subprocess CLI regression tests for cross-process persistence, normalization, sorting, overlap rejection and unchanged state bytes, adjacency, separate rooms, invalid input, and malformed-state preservation.
- Docs: README marks the current S1 command surface, provides run/test commands, documents state behavior, and states the single-writer limitation.
- Checks and repairs: the first unit run exposed an incorrect test expectation that list order followed ID; corrected it to expect start-time ordering, then all five unit tests passed. S1 acceptance checks all passed. Both checks passed again after integration to `main` at merge SHA `a765087e31b81e8f09dfbd784bd9edc78e9933ed`.
- Review: self-review completed; no helpers were started because S1 is a single bounded work item.
- Integration: feature commit `bedccfa` merged into isolated `main` with merge commit `a765087e31b81e8f09dfbd784bd9edc78e9933ed`.
- Unresolved decisions: none known. Human acceptance remains separate from these finite automated checks.
