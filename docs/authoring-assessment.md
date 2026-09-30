# Core authoring exercise

Date: 2026-09-30. This is a bounded usability assessment, not an A/B performance study or customer acceptance. [Measurement](measurement.md) owns the protocol; [the plan](implementation-plan.md) owns remaining product work.

## Task and independent expectation

A fresh Luna High agent received an isolated copy of the minimal example and the compiled `authoring` guidance. Its task was to require a rollback plan containing concrete steps, a trigger, verification and a responsible owner before an assessment recommends action. The agent could inspect the consumer and invoke the binary, but could not inspect the parent implementation or the hidden oracle.

The oracle was recorded before the first run. It expected an edit to the existing Rule and its generated Markdown, with exactly four affected resources: `sample/Rule/rollback-review`, `sample/Agent/rollback-reviewer`, `sample/Skill/rollback-review`, and `sample/Workflow/rollback-review`. A separate reviewer compared the actual committed diff, fixed context and impact with those expectations.

## Results, including the failed attempt

| Run | Fixed baseline → candidate | Outcome |
|---|---|---|
| Initial example | `dc0b01ef643a14e654f01aeb21a2ecc8c04af51d` → `97d8f718c810d807769ac92682deefb7e1edcdb7` | Correct semantic owner and requirement; failed the two-file boundary. Formatting changed six resource YAML files plus Project YAML and the Rule view, so impact included the full graph. |
| Canonical example, fresh agent | `0842ed82d05bf6e1fd23ebd464f9e55109885969` → `5bca5374d2d998730c96567093948e7b7ffa6b7d` | Passed the bounded oracle: exactly two changed paths and four affected resources; fixed contexts carry the requirement into the reviewer and entrypoints. |

The example's handwritten YAML was valid but not yet in the formatter's canonical layout. The product fix normalizes that example before it becomes an authoring baseline and adds a regression requiring a no-change format check. The first failed run is retained. Unknown-input or Project-change invalidation was not weakened to make the result smaller. A read-only format check of the existing Konfyra pilot returned no changes.

An early development binary also assembled a namespaced identity incorrectly; integration tests caught it and a corrected binary was supplied before final evidence. Both runs' final fixed evidence used tool SHA-256 `fffda090efd4b06548e63d5150c7dcb1763ee2299ce520457fadd59eaeb9ceaa`. Subsequent source-package/bootstrap and repository-identity changes create a different binary and must not reuse that tool identity.

## Verified boundary

The second actor ran formatting, rendering, structural checks and fixed context/impact. The independent reviewer checked the exact two-file diff, requirement placement and wording, and reran fixed impact and Rule context. The report correctly treats the generic profile's unavailable repository gate adapter as unavailable, not passed. The Rule names an assessment requirement; no rollback was executed or authorized by the test.

The raw actor reports, command outputs and oracle remain in the development worktree's excluded `.artifacts/authoring-*` directories. They are local evidence, not authenticated CI attestations. No review record was fabricated without a configured question and actual completed report. No token counts, provider-runtime certification, operational sufficiency, human acceptance or sustained savings are claimed.

## Consequences for daily use

- Establish canonical formatting in a migration or template baseline before measuring small edits. Incidental formatting is still a real input change.
- Use `find` to discover candidates, `explain` for direct relationships and area ownership, then fixed `context`/`impact` for transitive meaning and change scope.
- Keep consumer-specific checks explicit. Core authoring guides the sequence but cannot supply an unknown repository's gate adapter or delivery authority.
- Keep real failures and tool changes in evidence. A corrected attempt is useful evidence of refinement, not a retroactive pass for the first attempt.
