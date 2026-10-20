# Cockpit run inspector on recorded runs — 20 October 2026

**Scope:** the first slice of the [cockpit](../design/cockpit.md): `markitect cockpit` over recorded run evidence, implemented in the [cockpit adapter](../../internal/adapters/cockpit/README.md). This record separates source checks, UI behavior, real history inspection and what remains unproved. It records no live observation, provider run or owner acceptance.

## What was inspected

Nine recorded Pass-process runs of the [Apply Pass process trials](apply-pass-process-20261020.md), copied read-only from their WSL state directories to a local directory outside the repository. Their events, transcripts and reports were produced by development builds of the DEC-064 implementation before its Main integration; the cockpit read them without conversion. Agent text and transcripts stay outside the repository.

| Attempt | Run | Build | Model | Records | Recorded outcome | events.jsonl SHA-256 prefix |
| --- | --- | --- | --- | --- | --- | --- |
| S-N | 20261020T123344Z-66475687 | 0db7f008 | Haiku | 330 | verified, applied | da69e3f1ca9b8c56 |
| D-N | 20261020T123344Z-49406489 | 0db7f008 | Sonnet | 621 | verified, applied | c5d44fedfa2ffb50 |
| D-N | 20261020T121805Z-b5f64c31 | eb36d3cb | Haiku | 823 | incomplete, turn budget | 9271e4e046b76e36 |
| S-X | 20261020T113103Z-1cbd0747 | a6887967 | Haiku | 77 | failed, model defect | b4e7521f99802d59 |
| S-X | 20261020T121805Z-254fcbf1 | eb36d3cb | Haiku | 14 | failed, model defect | 8a0fdd7c347b6506 |
| S-D | 20261020T111537Z-37d03dae | 2072c1e7 | Haiku | 346 | incomplete, turn budget | 4a5d8033d6348025 |
| S-N | 20261020T111537Z-353a9270 | 2072c1e7 | Haiku | 339 | verified, applied | b273ed3fed45e489 |
| S-Q | 20261020T111537Z-c0f1eb2f | 2072c1e7 | Haiku | 297 | incomplete, turn budget | 0be92f6956409931 |
| S-D | 20261020T113103Z-d9bfd13d | a6887967 | Haiku | 262 | verified, applied | 70d452cf3a25493b |

## Acceptance scenarios

| Scenario | Evidence | What the inspector shows |
| --- | --- | --- |
| Accepted and returned results | Real: every run; S-N (2072c1e7) returns a basis and a result | Submission, technical check and parent or Orchestrator decision as separate steps with the decider's reason. |
| Correlated question and answer | Real: S-X 1cbd0747 (stockroom → Root Q1, Root → owner Q2, answered); D-N Haiku (eleven questions, two to the owner, one open at the end) | Question and answer with route, `about` correlation, recorded send only. |
| Lapsed question | Synthetic test only | A lapsed question is not actionable and leaves the account. No real run recorded a lapse. |
| Simultaneous independent branches | Real: D-N Sonnet Implement p3 | Several Managers working in parallel lanes; the timeline keeps them apart. |
| No-change result | Synthetic test only | The account states that nothing was written and the guard held. No Pass-process run published no_change. |
| Correction and fresh Verify on a new candidate | Real: S-N 66475687 | Verify p4 needs correction on candidate 3dc41cc0; Implement p5 corrects F1–F3; fresh Verify p6 on candidate c536584f closes them as resolved. |
| Interruption or unknown effects | Real: D-N Haiku (turn budget), S-Q 2072c1e7; synthetic: record without outcome | Incomplete outcome with its reason; a record without outcome or report says that it does not show whether the run still runs. |
| Publish with exact read-back | Real: S-N, D-N Sonnet | Receipt outcome and each path's read-back state; the receipt names no candidate digest, which the view states. |

At the final boundary the views of the verified runs give the outcome, the Run outcome mapped from the report's command status (these reports predate the `outcome` field; the view says so) and the read-back per path. Model-defect runs show Root's problem and the abort reason; nothing was published.

## Observed producer limitation

The record of a waiting Manager can be outdated. `recordStateLocked` records a change only when the state or the wait text differs from the Manager's fields, but the release and dependency paths set the new wait text into those fields first. A changed wait with an unchanged Waiting state is therefore not recorded. In D-N Sonnet, orders was released at record 322 after Root's basis was accepted, yet its last recorded wait still names Root's basis; its accepted plan needs the results of catalog and stock. In D-N Haiku, orders still shows a wait for catalog's result after catalog's result was accepted at record 526; its plan also needs stock's result, which never came.

The inspector does not present such a wait as current. It marks it outdated, names the record that fulfilled it and lists the unaccepted dependencies of the accepted plan instead, and lists the limitation among the gaps. Correcting the producer is Apply core work; it bears on the wait analysis of #532.

## Checks and UI behavior

- `go test ./internal/adapters/cockpit ./internal/tooling/release ./internal/tooling/architecture`, `go vet` of the changed packages, the architecture gate, `markitect check` and the guarded model view write passed locally on Windows. The tests use synthetic records in the run engine's event shapes and cover boundary isolation, the scenarios above, unreadable and unknown records, partial tails, replaced files, foreign hosts and loopback-only listening.
- Headless Chrome on Windows, driven through the DevTools protocol outside the repository: light and dark themes, replay with moving arrows, selection pausing replay with an announcement, record and exchange steps by keyboard, the shortcut list, the transcript dialog cut at the boundary, reduced motion with still arrows and the motion toggle off, 1000 px with a detail drawer and 390 px with the organization as a list and no horizontal page scroll. No browser errors were logged.

## Not established

Live observation of an active run, reconnection and owner actions belong to the next slices. No provider run, hosted CI, release or owner assessment is part of this record. The owner has not yet used the inspector.
