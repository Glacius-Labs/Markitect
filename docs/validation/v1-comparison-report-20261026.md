# V1 comparison report: Markitect against one agent

Date: 26 October 2026. Second of the two final reports the [owner directive](../concepts/sources/v1-acceptance-directive-20261025.md) asks for. It answers the product-value question across the three predeclared phases of the [comparison protocol](v1-comparison-protocol-20261024.md); the technical question is the [technical acceptance](v1-technical-acceptance-20261026.md). Each phase is read by its own rule and never pooled. Figures come from the phase records: [Phase A](v1-comparison-results-20261025.md), [Phase B](v1-comparison-results-phase-b-20261025.md), [Phase C](v1-comparison-results-phase-c-20261026.md).

## Verdict

**The data do not support "Markitect V1 is better than one agent".** No phase showed a quality advantage under its predeclared rule, and Phase C, the large project, counts against Markitect. Markitect cost 9 to 10.5 times the time and 9 to 42 times the tokens throughout. The added Manager organization has not shown its economic advantage. By the owner's own standard, this does not by itself discard the canonical model or the architecture.

| Phase | Project | Reading (predeclared rule) | M / B time | M / B tokens |
| --- | --- | --- | --- | --- |
| A | Library, greenfield and seeded repair, 6 units per arm | No quality advantage shown as approved; a tendency for M only under a seed list the coordinator changed after the first S4 run (disclosed) | about 9× | about 24× |
| B | Library, five chains of three cross-area changes and one unchanged re-application | No quality advantage shown: both arms conformed in all 20 counted runs, p = 1.0 | about 10.5× | about 9.2× |
| C | Makerspace, 21 Managers (20 after C7), three chains of a fresh build, eight changes and one unchanged re-application | Counts against Markitect: missed probe families M 339, B 10; p = 0.95 | about 9.3× | about 42× |

## Quality

- **Where both arms delivered, quality was equal or slightly in M's favor, descriptively.** On its 19 deliverable Phase C candidates M missed no hidden obligation; B missed 10 probe families, all from one audit defect it carried through chain 1. In Phase A, M removed a stale seeded comment in all three repairs, B in one. Neither difference is predeclared evidence, and N is small.
- **Change discipline favored B.** M failed the strict no-change gate (C9) in both Phase C chains where it delivered C9, through test-only edits; B passed it every time. M changed 1.9 to 2.5 times as many lines per step in Phase B, and its reviewers classed more of its changes as optional.
- **Among quality error types**, no unsupported success, no false blocker, no model blocker, no regression and no cross-cutting invariant failure occurred in any phase; only B failed behavior families (chain 1 of Phase C). The error type that decided Phase C is M's steps without a candidate, below.

## Reliability

- **One agent completed every step** of every phase: 30 of 30 in Phase C, 20 of 20 in Phase B, all of Phase A.
- **Markitect completed every Library Apply** (38 of 38 counted Applies on F and F′, plus P4), but only **19 of 30 Phase C steps**:
  - chain 1: 10 of 10; chain 2: 9 of 10, with one step lost after three provider-capacity failures;
  - chain 3: 0 of 10. After the fresh build failed, the input rule made every later step a fresh build again. Nine stopped on run limits (the per-message clock on an actively coordinating Root seven times, a correction pair started without enough message budget once, the message allowance used up by hierarchical question relay once) and one on a defect: open questions to a handed-off Manager triggered the Run-wide no-progress stop.
- The owner-approved fix plan targets these causes (WP1–WP4, plus WP5 added by the coordinator for the open-question defect; WP3b, ordering by dependency, stays deferred). Whether the fixes remove the failures is not yet shown: their native controls have not run, and none of the fixes is on the source the study ran on.

## Cost

- Markitect's cost is mostly **fixed per Apply, not proportional to the change.** At Makerspace scale, Assessment (9–15 M tokens) and Verify (14–17 M) run the whole hierarchy at every step, and a change step costs 41–55 M tokens. An unchanged re-application (C9) still cost 44–45 M tokens, against 0.3–0.5 M for one agent.
- One agent's cost follows the change: 0.3–2.9 M tokens per Phase C change step.
- Over Phase C, M used 1.63 B tokens and 37,167 s; B used 38.6 M tokens and 3,995 s.

## What the data support

Supported:

- At Library size, one Codex agent with ordinary documentation matches Markitect's quality at about a tenth of the time and a tenth to a twenty-fourth of the tokens.
- At Makerspace size, Markitect V1 on the frozen source is less reliable than one agent and costs about forty times the tokens; under the predeclared rule this counts against the claim.
- Markitect's cost structure is dominated by full-hierarchy Runs at every step, independent of the size of the change.

Not supported:

- That Markitect V1 is better than one agent, in any phase.
- That Markitect's domain quality is worse when it delivers: on its deliverable candidates it missed nothing, but that comparison is descriptive and was requested after the data.
- That the reliability failures are removed by the fix plan: that is a forecast until its native controls run.
- Any claim beyond this model (gpt-6-sol, high effort), these two projects and these backlogs.

## Limits

- N is small in every phase (Phase C: 3 chains per arm), and Phase C's chain 3 is one failure regime.
- Two post-data decisions affected scores, each disclosed and recorded with its direction: Phase A's changed seed list (in M's favor) and the continuation after Phase C's stop at chain 2 step C5 (least favorable to M).
- The change documents and specs were precise; real tickets are vaguer. The projects were designed by the study, not taken from practice.
- Quiet-machine and background-load deviations are listed in the Phase C record; they affect cost, not quality.

## What comes next

- The fix plan targets the reliability causes found here; its native controls run before a new frozen source F.
- An exploratory probe ([owner approval](../concepts/sources/v1-consistency-probe-approval-20261026.md), tracked in the [delivery plan](../implementation-plan.md#product-fixes-before-the-next-study-owner-approval-26-october)) tests the remaining core claim without Markitect: whether an agent that maintains its own documentation from tickets drifts over time, compared with one that receives a re-rendered spec.
- Whether a further study is worth its cost is the owner's decision.
