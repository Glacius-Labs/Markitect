# Apply Pass process trials — 20 October 2026

**Scope:** first real runs of the recursive Pass process of [DEC-064](../concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams) (PR #530) on the stockroom (level 1) and depot (level 2) trial cases of the Apply trial kit (`dev/apply-trial-runs` at `8652adb6`). Each packet's execution settings were translated outside the repository to `recursive-passes-v1` with ownership areas (`./` for Root, the delivery directory for its project Package, `realization/<package>/` for each Package) and a budget of 12 Passes. Runs used the [demonstration harness](../../experiments/recursive-apply-demo/README.md) in WSL Ubuntu with Docker 29.4.1, Claude Code 2.1.301, effort high. Source checks, these runs, Main integration and owner acceptance are reported separately. No owner acceptance is recorded.

## Verdict

The Pass process works as the concept intends:
- Plan → Prepare → Implement → Verify runs end to end without owner help. Parents accept or return their children's plans and results with concrete reasons.
- Drift is repaired with regression tests, and a model-only start produces a complete verified realization.
- A contradictory model ends the run within one Root turn.
- With Sonnet, the six-Manager depot verified in about 12 minutes.

The last builds also showed a complete correction cycle in a real run: Verify raised a genuine finding, a correction Implement fixed it, and a fresh Verify closed it with evidence.

Three things keep it from being called stable:
- **Slow.** Runs take 2–4× the time of the previous engine.
- **Overreach.** Managers change stands that already conform.
- **Serial waits.** In a multi-Package tree a weaker model plans serial dependencies, and its blocked Managers use up the turn budget.

The evidence covers two small Python projects and two models of one family, mostly in single runs per build.

## Changes driven by the runs

| Build | Change | Why |
| --- | --- | --- |
| a6887967 | The Plan and Prepare mandates state what their result is; a Pass without writes says it is read-only for every Manager. | On 2072c1e7 every leaf read "submit your complete result" in the Plan Pass as "implement now", wrote patches under /tmp and escalated read-only turns; S-D and S-Q used all 60 turns. |
| a6887967 | Work Passes never skip a forward Pass. | S-N on 2072c1e7 went Plan → Implement without Prepare. |
| eb36d3cb, 0fd4a86a | Apply's Root guidance: end the run for an erroneous or contradictory model, never ask the user to change it. | Both S-X runs before it asked the user first and aborted only after an owner answer; an unattended run would have waited until its deadline. |
| 0db7f008 | Plan and Implement change only what an observed model violation or a missing required realization or evidence needs; optional hardening stays a note. | Conforming stands received optional hardening, refactors and extra tests (S-D 16 paths, D-N 10 paths). |

## Runs

Expected outcomes are those of the trial kit; "paths" counts changed files of the publication.

| Case | Build | Model | Outcome | Engine seconds | Turns | Owner answers | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| S-N conforming | 2072c1e7 | Haiku | verified, applied, 6 paths | 1114 | 42 | 1 | Stuck in Plan until the answer; Prepare skipped. |
| S-D drift | 2072c1e7 | Haiku | incomplete, turn budget | – | 60 | 1 | Stuck in Plan. |
| S-Q gap, model only | 2072c1e7 | Haiku | incomplete, turn budget | – | 60 | 0 | Stuck in Plan. |
| S-X contradiction | 2072c1e7 | Haiku | model defect, nothing published | 168 | 6 | 1 | Asked the user before aborting. |
| S-N | a6887967 | Haiku | verified, applied, 8 paths | 824 | 25 | 0 | Fixed a real reference defect, plus optional hardening. |
| S-N | a6887967 | Sonnet | verified, applied, 6 paths | 514 | 25 | 0 | Same real defect fixed. |
| S-D | a6887967 | Haiku | expected outcome met, 6 paths | 902 | 30 | 0 | Both faults repaired with regression tests. |
| S-X | a6887967 | Haiku | model defect, nothing published | 334 | 9 | 1 | Asked the user before aborting. |
| S-Q | a6887967 | Haiku | incomplete, deadline in Verify | 1192 | 26 | 0 | Short deadline of 21 minutes. |
| S-N | eb36d3cb | Haiku | verified, applied, 6 paths | 840 | 25 | 0 | Real defect fixed. |
| S-D | eb36d3cb | Haiku | expected outcome met, 16 paths | 1018 | 26 | 0 | Faults repaired; also restructured the code into a package. |
| S-X | eb36d3cb | Haiku | model defect, nothing published | 20 | 1 | 0 | One Root turn. |
| S-Q | eb36d3cb | Haiku | verified, applied, 15 paths | 969 | 24 | 0 | Complete realization from the model; the gap was decided within the model, not reported. |
| D-N conforming | eb36d3cb | Sonnet | verified, applied, 10 paths | 768 | 75 | 0 | Optional hardening and about 350 lines of tests. |
| D-N | eb36d3cb | Haiku | incomplete, turn budget 120 in Implement | 1262 | 120 | 1 | Serial plan dependencies; blocked Managers kept escalating. |
| S-N | 0db7f008 | Haiku | verified, applied, 5 paths | 1151 | 36 | 0 | Verify raised the real defect as genuine findings; a correction Implement fixed it and a fresh Verify closed them; optional points stayed notes. |
| D-N | 0db7f008 | Sonnet | verified, applied, 6 paths | 710 | 77 | 0 | Only tests and a registry changed; the real defect stayed in place (`--name -Bolt` still refused). |

The trial kit's stockroom and depot references refuse a valid name that starts with a hyphen (`add-item --sku ABC-123 --name -Bolt`), which the model allows. Every Pass-process S-N run found and fixed it. The previous engine's no-change runs had left it in place, so the kit's "verified, unchanged" expectation does not hold for these references; the trial session corrects them.

## Topics and proposals

1. **Time.** The Plan Pass alone takes three to six minutes on these small projects, and every proposal costs a turn of the submitter and one of its parent. Proposal: let a leaf submit its Plan result together with its basis, as the same proposal, and measure turn latency per container start before tuning prompts.
2. **Overreach on conforming stands.** Partly real defects, partly optional work: hardening, restructuring, tests as planned evidence. With the 0db7f008 mandates, S-N kept optional points as notes and Verify caught the real defect, but the Sonnet D-N run added only tests and left the real defect in place. Repeat S-N, S-D and D-N several times before judging, and add a planted-conformance case to the kit.
3. **Plan dependencies in multi-Package trees.** Haiku planned chains such as stock → depot's result, and blocked Managers spent turns asking for writes. Proposals:
   - refuse dependencies on another Manager's result when its basis would do;
   - bound or exclude communication-only turns from the turn budget;
   - say in a blocked turn's prompt why it waits and that it cannot write.
4. **Model gap (S-Q).** Under DEC-064 a gap may be clarified through the user, and Root decided it within the model instead of asking. Whether that is right is the owner's reading; the kit still expects a model defect after DEC-063.
5. **Models.** Sonnet handled every case it ran faster and verified the six-Manager depot; Haiku manages one Package reliably but not the serial depot plan. Proposal: run Sonnet by default for multi-Package trees until topic 3 is addressed.
6. **Not yet observed in real runs:** helpers, disputes and disproved findings, correction cycles after a failed Verify, Extract on the Pass process, trees deeper than two levels.
