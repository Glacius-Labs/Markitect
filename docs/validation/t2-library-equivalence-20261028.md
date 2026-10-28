# T2: Library models, old and new equivalent

**Verdict:** the five Library steps in the [concept 2.0](../concepts/v2-concept.md) form say the same as their concept 1.0 originals, after one optional fix that was applied. Step T2 of the [concept 2.0 build](../implementation-plan.md#concept-20-build).

| | |
|---|---|
| Date | 28 October 2026 |
| Old | M1 [testdata/library/m1](../../testdata/library/m1/model.yaml); M2 its loan-limit edit (5 → 3); M3 to M5 the overlays in [tools/comparison/phase-b](../../tools/comparison/phase-b) composed as `requirements.py model_files` does |
| New | [tools/evaluation-tasks/library](../../tools/evaluation-tasks/README.md) `m1` to `m5` |
| Reviewer | An independent Claude Opus 5.5 subagent that did not write the conversion; it read all 22 old M1 Definitions beside the new files, diffed every old overlay and every new step, and resolved every link by script |
| Hidden material | The Library judge and requirements are unchanged |

## Result

| Step | Change | Verdict |
|---|---|---|
| `m1` | the small library | equivalent after fix 1 |
| `m2` | loan limit 5 → 3 | equivalent |
| `m3` | short-loan copies, 7-day period | equivalent |
| `m4` | one renewal per loan | equivalent |
| `m5` | no suspension, `setMemberStatus` removed | equivalent |

Each new delta changes exactly what the old step changed and nothing else. Every link resolves in every step; Root names only Packages; the Package edges are lending → catalog, lending → members, lending/reminders → members (m1 to m4) and the implicit child-to-parent edges, so no step has a cycle.

## Fixes

1. **Positive values** (optional, applied to every step): the old local Kind `LendingPolicy` required its values to be positive integers. The new [loan policy](../../tools/evaluation-tasks/library/m1/model/lending/loan-policy.md) now says each value is "a required positive integer".

## How the form changed, with the same information

- **No cycles.** In M1, members and catalog linked lending and lending linked them back. [Eligibility](../../tools/evaluation-tasks/library/m1/model/members/eligibility.md) and [availability](../../tools/evaluation-tasks/library/m1/model/catalog/availability.md) now state lending's duty in prose; borrow, return and renew link them, so lending's Manager still receives them.
- **Root is the project.** The old `library` Package became Root. The [callable interface](../../tools/evaluation-tasks/library/m1/model/callable-interface.md) names the Package and use case of each operation instead of linking it, and every Feature names its operation in return.
- **No typed fields.** The `LendingPolicy` Kind and its Properties became one Rule with a table of the values and their meanings, verbatim.
- **Additions without new meaning:** the Feature-to-operation sentences, and that the loan policy is the single source of its values.

## Check on T1

The reviewer could not run the 2.0 `markitect check`, which was not built yet. Every step compiles with no error on T1's branch `dev/awesome-chaum-554c4d` at `e3c922ee`, with the lock that branch's `markitect lock` writes; Root's link to the grandchild Package `lending/reminders` is accepted, as the reviewer expected. To be confirmed on `main` once T1 merges.
