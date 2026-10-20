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


## First hardening cycle

The owner requested evidence-driven Apply hardening after PR #530. The prepared candidate starts at Main `780dccb7eda4f10e70d1f5513f5e43bd031346b1`; it preserves the recursive Pass engine, ownership, checks, fresh Verify and finite budgets. It does not change the default model or the trial stream's fixtures. Main `93c29e63ae802bc4fda25f13849c547de995786a` subsequently clarified that the current phase is assessment and concept refinement. The candidate is preserved as draft PR #536 for that assessment, not an adopted change or an instruction to integrate or run providers. The current delivery plan remains unchanged.

### Observation and challenged assumption

The Sonnet Depot attempt `att5s-D-N`, product `0db7f008197de9ca46fd029305a86827cf37f0d7`, run `20261020T123344Z-49406489`, published a verified candidate with no findings. It changed five test files and a registry, leaving `app.py` unchanged. Both the project's 51 tests and the outside oracle's 21 cases passed. Yet the model permits a trimmed, nonempty name of at most 60 codepoints, and the public command refused `add-item --sku ABC-123 --name -Bolt` in argument parsing, before domain validation. The captured post-run probe observed exit 2 and `argument --name: expected one argument`.

The preserved Depot and Root reviewer transcripts show actual source inspection and manual CLI checks. Neither tried this permitted input class; the outside oracle omitted it too. This was a coverage blind spot, not a known finding waived by publication. Fresh contexts and passing suites did not ensure independent test hypotheses. Per-Definition `done` labels and global evidence strings admitted an assessment broader than its observations.

The Haiku Depot attempt `att4-D-N`, run `20261020T121805Z-b5f64c31`, waited for Depot's complete result while Depot needed that child's result to finish. Communication-only turns could not repair the dependency and spent the finite turn budget. Main already has `planCycleLocked`, which includes implicit parent-first and child-integration edges, and read-only blocked-turn guidance. This cycle therefore adds regression evidence for that existing guard instead of inventing a second dependency mechanism or removing the budget.

### Prepared implementation candidate (not adopted)

- Plan names distinct claims, permitted/rejected input classes and public-boundary evidence. Verify derives its checklist from the model before reading the candidate or existing assertions, then exercises permitted unusual inputs, boundaries and invalid inputs through the public interface.
- The common report envelope accepts per-Definition observations: `claim`, `method`, `expected`, `observed`. Every Review basis/result entry marked `done` or `preserved` requires at least one complete observation. A global passing-suite count cannot substitute for it. The parent's proposal view includes these observations for semantic assessment. An unsupported claim must be investigated or remain open with the required finding.
- Protocol regressions reject missing or blank observation fields for both success statuses at both submission stages, and confirm that the parent sees the submitted evidence. Plan regressions cover parent/ancestor results, descendant basis/results, valid ancestor bases and a valid other-branch result dependency.

This structural gate proves that a claim has an inspectable record. It cannot prove that a command ran, that the observation is true, that all semantic cases were considered or that the model is fully realized. Independent inspection and real behavioral evaluation remain necessary. No new generic test generator, review role or coverage percentage is introduced.

### Recheck and decision criteria

Source validation and independent complete-diff review belong to this candidate. Real provider improvement is still unproved. The proposed finite recheck uses one pinned build, one pinned trial snapshot and Haiku: (1) the preserved faulty Depot reference, without naming the defect to its Agents, (2) a second Apply on its corrected candidate, and (3) Stockroom's two-fault drift case. Freeze public-behavior assertions outside Agent context before running; first establish that they reject the faulty reference and accept a corrected control. Preserve attempts, questions, turns, findings, candidate/publication digests and changed paths.

Success requires discovery and repair of the actual Depot violation, independently passing public-boundary checks, confirmation by fresh Verify and independent closure of any raised genuine finding, no optional changes on the conforming repeat, both Stockroom faults repaired, no ownership/accounting violation and no unresolved dependency. A timeout, missed defect, optional rewrite or unsupported success is retained as a failed observation; it does not trigger an unlimited rerun. This provider series needs its own finite owner grant; the earlier grant belonged to the finished Claude implementation session.

### Local validation observations

The eight-case public-name checker was frozen outside the repository at `/home/user/mk-hardening-20261020/check_public_names.py` (SHA-256 `14150f13c61f2841c82236573648e4c581409b18ca4adf78981443c0a849f03b`). On temporary copies in native WSL Python 3.12.3, it passed 6/8 cases on `att5s-D-N` and 8/8 on the corrected `att4s-D-N` control. The two failures are `-Bolt` and `--Bolt`; ordinary names, `-9`, `-`, and three invalid-name cases distinguish the input classes. The source `app.py` hashes are respectively `faa7ad9f050c0d60c0fa71e7bb4979c6efe64955d128c530f1b5a1ff3ff25e2b` and `a2953679e20e49d09f97539406350f8c47f2d5480d42247800a3918cc84c24d2`. The results are preserved beside the checker in `control-results.json`. This reproduces the defect and validates the discriminator; it is neither a new Apply nor evidence of improved Agent detection.

Before the authority-source review fix, `go test ./... -count=1 -timeout=20m` on the hardening working candidate in native Windows passed every package except `internal/agentd`: three bridge tests failed (including Windows Unix-socket `connect: An invalid argument was supplied`). These are recorded as failures, not a green full suite. After that fix, `go test ./internal/run ./internal/apply -count=1 -timeout=5m`, `go vet ./internal/run ./internal/apply`, the actual Model/File preparation (`markitect check --repo .`) and the changed-page documentation link check passed. The delivery PR binds the frozen source revision and any subsequent platform validation; these working-candidate results are not a claim of a final green full suite. Independent review found that shared observation guidance must not treat an Extract model proposal as its own authority; the candidate corrects this by keeping shared guidance source-aware and giving Extract fixed-artifact/documented-intent expectations, with a focused workflow assertion. Real provider improvement remains pending the finite recheck.


The frozen candidate source `0a40316f95b58d04d53e8f3762a62c449cc70201` passed the complete Linux `go test ./... -count=1 -timeout=20m` suite under Go 1.27.1 in image `sha256:bfa95066afac0f7d52b9d3b268082f647089f5c8b7790e3500eb1f49414faaf0`, with a writable disposable source clone for testkit fixtures and `--init`, as used by the real runtime adapter. Two earlier incorrectly configured Linux harness attempts (read-only fixtures, then no init/reaping) failed and remain preserved alongside the final successful log and binding under `/home/user/mk-hardening-20261020/`. Git status stayed clean. The three native Windows agentd failures reproduced unchanged at baseline `780dccb7eda4f10e70d1f5513f5e43bd031346b1`. These are source-validation outcomes; no provider improvement, Main integration or owner acceptance follows from them.


## Concept disposition after owner assessment

The owner has adopted the [evidence-driven hardening refinement](../design/canonical-model/manager-apply-workflow.md#evidence-driven-hardening-adopted-20-october-2026) and requested the [new Claude implementation/retest handoff](../development/claude-apply-hardening-handoff-20261020.md). This follows the assessment-only checkpoint recorded above. PR #536's fields/tests are reusable partial work, not the complete adopted solution; new receipts and active counterexample/parent assessment still need implementation and real evaluation. All earlier source and Run results retain their exact original bindings. The workflow's disposition table assesses every previous proposal, distinguishing existing guards, accepted corrections, conditional batching, rejected blanket budget exemption, unchanged model default and the S-Q clarification rule. No new provider Run is reported here.
