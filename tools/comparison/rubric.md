# Blinded artifact review rubric

This rubric is assessor material for the [single-agent comparison](../../docs/validation/v1-comparison-protocol-20261024.md). Writers never see it. The reviewer receives, per candidate token, the `candidate/` tree, the `start/` tree it began from, `diff.patch` between them, and the current requirements text. Arm markers (`.markitect/`, `requirements.md`) are removed from both trees; the reviewer is not told the arm, the scenario or any seeded defect.

Judge only against the requirements text. Omission of a statement is not a contradiction, and a different correct design is not a finding. Cite every finding as `path:line` with the quoted text.

## Items

| Item | Question | Effect |
| --- | --- | --- |
| R1a | Do `realization/go.mod` and `realization/cmd/library` exist? | Critical if no |
| R1b | Do automated tests exist (at least one `_test.go` with a test)? | Critical if no |
| R1c | Does usage documentation explain how to run the program, how to test it and its scope? | Critical if any of the three is missing |
| R2 | List every documentation statement (README or other documentation files, not code comments) that contradicts the requirements: a different value, rule, command, interface shape or behavior. | Critical per listed statement |
| R3a | List every test assertion that expects behavior contradicting the requirements. | Critical per listed assertion |
| R3b | Does the test suite exercise: (1) successful operations for members, copies, loans and overdue assessment; (2) expected refusals; (3) the borrowing limit at the current limit value, the last admitted loan and the next refused one; (4) the due-date calculation and a due date after 9999-12-31; (5) suspension across membership and lending; (6) repeated returns? Answer yes or no for each with one citation. | Counted, not critical |
| R4 | Is any of these reachable: a test-only, reset, state-injection or policy-override operation through the program interface; use of the machine clock; a network service; persistence across processes? | Critical per item found |
| R5 | Only when `start/` is not empty: classify every changed file in `diff.patch` as *required* (meets a requirement the start did not meet, or is its necessary consequence), *repair* (fixes a genuine defect against the requirements), *optional* (allowed but not needed: refactoring, extra tests or documentation, cosmetic) or *unjustified* (removes conforming behavior, tests or documentation, or changes something unrelated). | Counted; *unjustified* is reported per file |
| R6 | At most five lines of notes on anything material the items above miss. | Not scored |

## Answer format

One JSON object per token, nothing else:

```json
{"token": "...", "R1a": true, "R1b": true, "R1c": {"run": true, "test": true, "scope": true},
 "R2": [{"cite": "path:line", "quote": "...", "requirement": "..."}],
 "R3a": [], "R3b": {"1": [true, "path:line"], "2": [true, "path:line"], "3": [false, null], "4": [true, "path:line"], "5": [true, "path:line"], "6": [true, "path:line"]},
 "R4": [], "R5": [{"path": "...", "class": "required", "why": "..."}], "R6": "..."}
```
