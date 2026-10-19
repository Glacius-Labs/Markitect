# Extract on the library: first real runs — 19 October 2026

**Scope:** the first real runs of the bounded [Extract](../design/canonical-model/manager-apply-runtime.md#extract-first-bounded-implementation) (case 3 of [DEC-057](../concepts/register.md#dec-057-three-fundamental-starting-situations)), run on the library realization that the demonstration RA-D2 ([repetitions record](recursive-apply-repetitions-20261019.md)) published as A2. Each run was followed by a round-trip Apply with the extracted model. Implementation, local tests and these product runs are reported separately. Hosted CI was not run (CI is manual during development). No human acceptance is recorded.

## Verdict

**The mechanism works end to end:**
- From code alone, Extract proposed the same five Packages and the same layout exception as the human model.
- It derived and registered a model, verified it, and published it with read-back in 7–9 minutes.
- The round-trip Apply then found nothing to change.

**The model is not yet adoptable as binding intent.**
- The behavior is recovered well: roughly 90 % of the use-case and rule obligations, including the limit of 3 and the 14-day period.
- Project-level obligations are mostly missing: delivery (tests, documentation), the technical boundary (Go module, standard library only, build and test commands, no test-only operations) and the non-goals.
- Many details of the Go code became binding rules. Examples are ordering, the order in which refusals are checked, wire-parser corner cases, internal views and wiring, and a concurrency claim.
- Fresh Verify caught one unsupported statement, but did not block this over-specification, although its guidance names that as blocking.

The independent assessor estimated 75–80 % of the original's binding obligations recovered. The extract adds about as many incidental binding statements. An owner would need roughly an hour of editing to adopt it.

## Setup

| Item | Value |
| --- | --- |
| Product | Build of `120272dd` (branch `dev/extract-library`): Extract, DEC-060 and the direction-neutral first Execute item. The review fixes made after these runs were not exercised by them: owner questions across both phases, the guard on every candidate and the turn split between phases. |
| Agents | Sonnet `claude-sonnet-5-5`, high effort, for every Execute and Verify turn; the provider reported that model for every turn. No Codex or OpenAI inference. |
| Start | Only `realization/` of RA-D2's published A2, plus the imported vocabulary, a manifest and lock without local Definitions, and `extract-execution.yaml`. That file declares the layout root `realization` with no exceptions, no checks, and a budget of 20 min, 120 turns and 4 rounds. Nothing of the original model was copied. |
| Limits | `timeout` 27.5 min outside all prompts for the Extract; the round trip ran afterwards under its own 21-minute deadline |
| Adoption | Scripted commit of the published model and registry, standing in for the owner's adoption ([handoff](../design/canonical-model/commands.md#adopt-the-model-and-hand-findings-to-apply)); no Extract finding was passed to the round trip |
| Harness | `experiments/recursive-apply-demo/ra_extract.sh` and `ra_extract_project.sh` |

## Attempts

| Attempt | Extract | Structure phase | Model phase | Round trip | Cost estimate |
| --- | --- | --- | --- | --- | --- |
| RX-1 | passed, 433 s | 147 s, 1 round, 2 turns | 254 s, 1 round, 14 Execute / 6 Verify | verified, no change, 245 s, 1 advisory finding | 3.36 + 1.67 USD |
| RX-2 | not started (harness error, no Agent turn) | | | | 0 |
| RX-3 | passed, 513 s | 100 s, 1 round, 2 turns | 384 s, 2 rounds, 14 Execute / 12 Verify | verified, no change, 264 s, no finding | 3.31 + 1.50 USD |

**RX-1 is contaminated.** The harness passed it the demonstration's owner request ("Apply the committed Library model") and its two owner decisions. Those decisions name `library/callable-interface`, a Definition of the original model, and state the unknown-field and 9999-12-31 rules. The extracted model then used that name and stated both rules. The harness was fixed: an Extract and its round trip get their own owner request and no owner decisions.

**RX-2** stopped at once: the run configuration rejects an empty `ownerDecisions` list. The harness now omits the field.

**RX-3 is the clean run** and the basis of the assessment below. In both RX-1 and RX-3:
- No owner question was asked.
- At most 4 Agent turns ran in parallel.
- The Managers used the `check` tool 18–21 times per Extract.

Costs are Claude Code's per-turn estimates; billing for the runtime token was not observed.

## Observed behavior (RX-3)

- **Structure phase:** Root alone proposed `catalog`, `lending`, `lending/reminders`, `library` and `members`, each with an overview Definition, and the exception `realization/cmd/library` → `library`. Verify raised one advisory finding: a garbled sentence in an overview.
- **Model phase:**
  - Root registered `realization/README.md` and `go.mod` and sent each child a basis.
  - The children started after Root's first turn, and reminders after lending's.
  - Root used the `check` tool and relayed a broken link to its owner.
- **Verify rounds:** In round 1, library and Root blocked an unsupported statement: the id-echo rule claimed more than `envelope.go` does. Library corrected it, and round 2 was green.
- **Open advisory findings:** two remained open:
  - **Notice order:** the model does not say whether the order of reminder notices is part of the contract.
  - **Eligibility:** lending applies the active-member rule itself instead of members' check. This is the DEC-060 duplication of the repetitions audit. It was not recorded as a realization deviation, so `deviations` is empty.
- **Publication:** 47 files changed, all in `.markitect/model`, `.markitect/registry` and the layout exceptions of `execution.yaml`. Every one of the 17 realization files is registered exactly once.
- **Round trip:** six Managers, one Verify round, no change. Without declared checks it shows that Apply Managers find the artifacts consistent with the extracted model. It is not evidence of model quality, because the model was derived from these artifacts.

## Model quality (independent comparison with M2)

A separate Sonnet agent compared each extracted model with the human-written M2 and the code. This is the same model family as the authors, and only part of the code was read in full.

| Aspect | Result (RX-3; RX-1 alike) |
| --- | --- |
| Structure | Same five Packages and the same layout exception; registry complete and plausible, links coarse |
| Behavior rules | About 90 % recovered and correct: limit 3, 14 days, refusals leave state unchanged, eligibility, availability, overdue boundary, inputs, operation table |
| Delivery, technical boundary, non-goals | Under 40 % recovered, although `README.md`, `go.mod` and the tests state most of them |
| Policy | Written as numbers in the prose of several Definitions instead of one structured policy, as the original's `LendingPolicy` Kind does. An M1 → M2 style change would touch about five Definitions. |
| Over-specification | Fixed ordering and refusal precedence; wire-parser corner cases; internal views, wiring and Go APIs; a concurrency claim; availability reframed as a flag; "basis" paragraphs citing file names |
| Contradictions | None found |

## Assessment

| Aspect | Grade | Reason |
| --- | --- | --- |
| Mechanism | A− | Both phases, enforced write plans, check tool, Verify, guarded publication and read-back worked in two real runs. The second real run had a repair round. Owner questions and an interrupted run were not exercised. |
| Model quality | C+ | The behavior is faithful, but the model describes the code more than the intent and misses the project-level obligations |
| Strength of the evidence | C | One clean run (RX-3) and one contaminated run, one small example, one model; the assessor is of the same model family |

## Next levers

1. **What a model contains.** The Extract tasks ask for meaning, behavior, rules, interfaces and relationships per area. Nothing makes a Manager responsible for project-level delivery, the technical boundary or non-goals, and nothing tells the model to separate intent from realization choices. This is a concept question for the owner: should Root's model phase own the project-level obligations, and how should intent be told apart from incidental realization?
2. **Adoption subject.** The owner adopts the model. A readable adoption subject that lists binding statements which look like realization choices would put the pruning where the [handoff](../design/canonical-model/commands.md#adopt-the-model-and-hand-findings-to-apply) already places the decision.
3. **Checks.** Extract may not propose checks, so the adopted project has none until the owner adds them. Whether Extract should propose checks from the documented build and test commands is a decision for the owner.
4. **Deviations.** The Managers recorded a known cross-area defect only as an advisory finding, not as a realization deviation, so the handoff to Apply would not carry it.
