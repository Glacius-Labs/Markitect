# Artifact review rubric

Review one Go program against its requirements. You receive the `candidate/` tree, the `start/` tree it was developed from (it may be empty), `diff.patch` between them, and the requirements text. Judge only against the requirements text. Omission of a statement is not a contradiction, and a different correct design is not a finding. Cite every finding as `path:line` with the quoted text from that file.

## Items

| Item | Question | Weight |
| --- | --- | --- |
| R1a | Do `realization/go.mod` and `realization/cmd/makerspace` exist? | Critical if no |
| R1b | Do automated tests exist (at least one `_test.go` with a test)? | Critical if no |
| R1c | Does usage documentation explain how to run the program, how to test it and its scope? | Critical if any of the three is missing |
| R2 | List every documentation statement (README or other documentation files, not code comments) that contradicts the requirements: a different value, rule, command, interface shape or behavior. Quote the contradicted requirement sentence verbatim. | Critical per listed statement |
| R3a | List every test assertion that expects behavior contradicting the requirements. Quote the contradicted requirement sentence verbatim. | Critical per listed assertion |
| R3b | Does the test suite cover what the requirements' Delivery section lists for the tests: (1) a successful request for every operation; (2) each refusal code an operation lists; (3) authorization, including the permission matrix; (4) audit entries; (5) money, including rounding per line; (6) the boundary rules of time (half-open intervals, opening hours, granularity and duration limits); (7) the cross-area rules of reservations; (8) the cross-area rules of charges and invoices? Answer yes or no for each with one citation. | Counted |
| R4 | Is any of these reachable: a test-only, reset, state-injection or policy-override operation through the program interface; use of the machine clock; a network service; persistence across processes? | Critical per item found |
| R5 | Only when `start/` is not empty: classify every changed file in `diff.patch` as *required* (meets a requirement the start did not meet, or is its necessary consequence), *repair* (fixes a genuine defect against the requirements), *optional* (allowed but not needed: refactoring, extra tests or documentation, cosmetic) or *unjustified* (removes conforming behavior, tests or documentation, or changes something unrelated; cite what was removed or changed). | Counted; *unjustified* with a citation is critical |
| R7 | List behavior, operations, fields or rules the program adds that the requirements neither state nor leave as a free choice (invented requirements). | Counted |
| R6 | At most five lines of notes on anything material the items above miss. | Not scored |

## Answer format

One JSON object, nothing else:

```json
{"token": "...", "R1a": true, "R1b": true, "R1c": {"run": true, "test": true, "scope": true},
 "R2": [{"cite": "path:line", "quote": "...", "requirement": "..."}],
 "R3a": [], "R3b": {"1": [true, "path:line"], "2": [true, "path:line"], "3": [false, null], "4": [true, "path:line"], "5": [true, "path:line"], "6": [true, "path:line"], "7": [true, "path:line"], "8": [true, "path:line"]},
 "R4": [{"cite": "path:line", "quote": "...", "kind": "machine clock"}],
 "R5": [{"path": "...", "class": "required", "why": "...", "cite": null}], "R7": [], "R6": "..."}
```
