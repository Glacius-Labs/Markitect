# Core authoring exercise

Date: 2026-09-30. This bounded usability assessment used the repository's isolated minimal example. It is not an A/B performance study or an external acceptance decision. [Measurement](measurement.md) owns the protocol; the [roadmap](implementation-plan.md) owns remaining product work.

## Task and independent expectation

A fresh Luna High agent received an isolated copy of the minimal example and the compiled `authoring` guidance. The task was to require a rollback plan with concrete steps, a trigger, verification, and a responsible owner before an assessment recommends action. The agent could inspect the example and invoke the binary, but could not inspect the parent implementation or the hidden oracle.

The oracle was recorded before the first run. It expected an edit to the existing Rule and its generated Markdown, with exactly four affected resources: `sample/Rule/rollback-review`, `sample/Agent/rollback-reviewer`, `sample/Skill/rollback-review`, and `sample/Workflow/rollback-review`. A separate reviewer compared the actual committed diff, fixed context, and impact with those expectations.

## Results, including the failed attempt

| Run | Fixed baseline → candidate | Outcome |
|---|---|---|
| Initial example | `dc0b01ef643a14e654f01aeb21a2ecc8c04af51d` → `97d8f718c810d807769ac92682deefb7e1edcdb7` | Correct semantic owner and requirement; failed the two-file boundary. Formatting changed six resource YAML files plus Project YAML and the Rule view, so impact included the full graph. |
| Canonical example, fresh agent | `0842ed82d05bf6e1fd23ebd464f9e55109885969` → `5bca5374d2d998730c96567093948e7b7ffa6b7d` | Passed the bounded oracle: exactly two changed paths and four affected resources; fixed contexts carried the requirement into the reviewer and entrypoints. |

The example YAML was valid but not in canonical format. The product normalized that example before using it as an authoring baseline and added a regression requiring a no-change format check. The failed attempt is retained. Unknown-input or Project-change invalidation was not weakened to reduce the result size.

An early development binary assembled a namespaced identity incorrectly; integration tests found it and a corrected binary was supplied before final evidence. Both final runs used tool SHA-256 `fffda090efd4b06548e63d5150c7dcb1763ee2299ce520457fadd59eaeb9ceaa`. Subsequent source-package/bootstrap and repository-identity changes create a different tool identity and cannot reuse this evidence.

The results above record the fixture and sibling-view contract at their fixed 2026-09-30 commits. The current example uses canonical YAML under `.markitect/areas/` and opts into views under `docs/markitect/`; the dated run is not a replay of that current layout.

## Verified boundary

The second actor formatted, rendered, structurally checked, and compiled fixed context and impact. The independent reviewer checked the exact two-file diff, requirement placement, and wording, then reran fixed impact and Rule context. At the time, an unavailable repository-check adapter was reported as unavailable rather than passed. The Rule named an assessment requirement; no rollback was executed or authorized by the exercise.

Raw actor reports, command output, and the pre-recorded oracle remain in excluded development artifacts. They are local evidence, not hosted CI attestations. No review record was fabricated without a configured question and completed report. Token counts, runtime conduct, operational sufficiency, human acceptance, and sustained savings are not claimed.

## Product learnings

- Establish canonical formatting in an example or baseline before measuring a small edit; formatting changes are real inputs.
- Use `find` for discovery, `explain` for direct relationships and area ownership, and fixed `context`/`impact` for transitive meaning and change scope.
- Declare project verification commands explicitly. Core authoring guidance does not supply project-owned gates or authority.
- Keep failures and tool changes in the evidence; a corrected run does not retroactively pass the earlier attempt.
