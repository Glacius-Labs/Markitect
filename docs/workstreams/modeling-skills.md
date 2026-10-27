# Modeling skills and model design (lane 7)

This lane covers how people and agents design, write and change a canonical model: the modeling skill for agents, the modeling guide, deriving a model from existing code (Extract) and the everyday model-change workflow. This page holds the lane's scope, a short state, its gaps and its open owner questions, and links to the pages that own the details.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026. It is a topic lane: it takes stock and discusses with the owner, and builds nothing until the owner says so.

## Scope

- **Owns:**
  - The [modeling guide](../design/modeling-guide.md) and its checkable side, the [quality criteria](../design/model-quality-criteria.md).
  - The modeling skill [`packaging/skills/markitect-modeling/`](../../packaging/skills/markitect-modeling/SKILL.md).
  - The analysis tool [`tools/model-analysis/`](../../tools/model-analysis/README.md): graph, references, impact, lint, lock.
  - How the Libraries are used in a project model. What the Libraries contain is shared language work.
  - Extract, if it returns, and the workflow from a change of intent to a checked model diff.
- **Elsewhere:**
  - The language, Core and the Libraries' content belong to the model-language lane (6); the compiler and `markitect check` to the compiler lane (8).
  - Automated triggers that lead to model changes belong to the continuous-change lane (10); this lane supplies the skill such a change runs through.
  - The cockpit's model-change surface ([DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration)) belongs to the cockpit lane (11).
  - Shipping the skill to users is shared with the plugin variant (lane 2). Model-writing for the evaluation portfolio follows the evaluation lane (4).

## State

- **Modeling guide:** v0.2 of 26 October, with the owner's architecture decision ([DEC-074](../concepts/register.md#dec-074-canonical-model-track-parallel-to-v1)): the Package tree is the architecture, plus dependency rules, Interfaces and placements. Its Kind names (Policy, Obligation, ValueType) are still the illustrative ones written before the Libraries 1.0.0; the skill already uses the real ones (`Practice.Rule`, `Values.Scalar`).
- **Skill and tool:** merged with [PR #771](https://github.com/Glacius-Labs/Markitect/pull/771) (W5). In two trials a fresh Sonnet agent with only the skill modeled the library program's neutral requirements: about 70 Definitions in 6 Packages, check and lint clean, an independent fidelity review found the same program, no manual repair. The tool is not run in CI and is not part of the product CLI.
- **Pilot:** the Makerspace remodel makerspace-v2 and its change chain C1–C9 are merged. In the [old-versus-new gate](../validation/v1-makerspace-gate-20261027.md) the new model built W0 and C1 with 12 Managers instead of 21 and fewer tokens, one run per cell; C2 stopped on a latent Interface defect in v2.0, which v2.1 and a new `interface` lint fixed, and then published.
- **Extract:** retired from V1 with [PR #638](https://github.com/Glacius-Labs/Markitect/pull/638). Before that, runs of 19–20 October on the library and Readinglog worked mechanically and recovered most behavior, but missed delivery obligations and the technical boundary, put policy values in prose and over-specified internal detail ([DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains)). DEC-061's round-2 clarification still waits for the owner's confirmation.
- **Everyday change workflow:** no page describes it. The guide's section 7 has four rules; the [project workflow](../guides/project-workflow.md) covers check, docs and Apply, and says a model blocker is fixed "outside the Run" without saying how.
- **Sub-sessions:** none running.

## Gaps

1. **Changing a model is untested with the skill.** Both trials wrote a new model. Nobody has measured an agent with only the skill turning a change request into a model diff, reviewing a model, or fixing a model blocker.
2. **No change workflow.** Missing are the steps from a change request to a reviewed model diff: trace to the Definitions it touches, `impact` for the owner's review, check, lint and lock, and what to do after a model blocker.
3. **Guide and Libraries drift.** The guide's Kind names and examples predate the Libraries; the skill's `reference.md` is generated from them, the guide is not.
4. **No path from existing code to a model.** With Extract retired, a project that has code but no model has no supported start.
5. **The skill reaches no user.** It is a folder in this repository; no install, plugin or CLI ships it.
6. **The tool has no gate.** `tools/model-analysis` is not in CI, so a Library or report change can break it unnoticed.

## Open owner questions

1. **Next step.** Recommendation: a change-task trial first. A fresh agent with only the skill makes one Makerspace change step (for example C3) on makerspace-v2.1, judged against the step's requirements; then write the change workflow from what it shows.
2. **Extract.** Should a model-from-code path return, and in which form? Recommendation: as a mode of the skill, which proposes a model and stops for the owner like the architecture step, not as a host command; DEC-061's criteria carry over. If it should be a command, it belongs to the commands lane (9).
3. **DEC-061 clarification.** Confirm round 2: the binding part is the contract callers rely on, and incidental details such as message wording or identifier formats are free unless the artifacts fix them. Recommendation: confirm.
4. **Shipping the skill.** Ship it with the plugin variant, the CLI distribution, or both? Recommendation: decide with lane 2; the skill itself stays here.
5. **Guide version.** Align the guide's Kind names with the Libraries 1.0.0 now, or wait for the open Library questions ([standard Libraries](../design/standard-libraries.md#answered-and-open-questions)) in lane 6? Recommendation: align now; the four open questions change only a few lines.
