# Progress

- Station: S4; released work item R12 complete and integrated into isolated `main`.
- Feature commit: `c1dbc33`; no-ff merge commit: `1cc7b37acd20905add3defd2d4c910d8d08547a0`.
- Default `list` returns `bookings`; temporary `list --legacy` returns the old `reservations` envelope. In-memory state, active code, tests, and docs use booking terminology. Existing raw JSON is adapted on read and written back under its established storage key only when a mutation succeeds.
- Compatibility coverage begins with pre-change JSON; default/legacy outputs and composed filters match and do not rewrite bytes. Book, cancel, summary, and export then operate on that data; returned fields, ID progression, and raw storage schema are checked.
- Checks on merge SHA `1cc7b37acd20905add3defd2d4c910d8d08547a0`: `C:/Python313/python.exe -B -m unittest discover -s tests -v`, exit 0 (25 tests), log `.work/logs/post-merge-unit-s4.log`; `C:/Python313/python.exe -B checks/acceptance.py --repo . --case roombook --station 4`, exit 0 (13 checks), log `.work/logs/post-merge-acceptance-s4.log`.
- Initial compatibility test used lowercase `atlas` despite case-sensitive matching; corrected it to `Atlas`. Review found no remaining S4 issue. Human acceptance was not performed.
- Dispatcher state: `.study/station.json` is the S4 release change and remains unstaged.
- Next: verify the final evidence commit, then return the completed backlog result.
