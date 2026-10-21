# Model assessment trial

A bounded experiment of the [handoff of 21 October 2026](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-c-model-assessment-before-realization), Session C: after the deterministic Model check and File check, does a small read-only Agent team find what the bound model leaves undecided or contradicts, before any realization, without inventing requirements or questioning permitted freedom? It is no product gate, no Command and no Apply step. A positive conclusion is no completeness proof and replaces neither Apply's later questions nor its independent Verify.

**Everything here is operator material.** Agents receive only a packet: the frozen control's repository. Expectations, this guide, the scorer and the ledger stay outside every packet and mount.

## Method

**Preconditions.** Every packet passes `markitect check` (Model check and strict File check) when the series is frozen, and the harness's preparation runs both again before any turn; a failing stand is refused before work, as in Apply.

**Treatment.** One review Pass of the one Pass process (`apply.AssessCaptured`) on the captured stand, driven by the grant-gated `TestModelAssessmentHarness` in `internal/bootstrap` with the product's real preparation and adapters. Its purpose, mandate and guidance are frozen in [assessment-pass.json](assessment-pass.json):

- The team is the model's Manager tree: Root and one reviewer per Package, so its size follows the model. Every reviewer starts fresh and reads the stand read-only.
- Each reviewer assesses its own obligations against four questions: are public decisions stated or explicitly delegated, do statements contradict, are responsibilities and contracts sufficient, and which freedom is permitted. It covers each obligation with an inspected model source.
- Parents assess interfaces and cross-area consistency and challenge their children's findings and claims: they accept, return or dispute them. A finding kept open only by a fresh dispute gets one new complete Pass on the unchanged stand, as in Apply, within the packet's Pass budget.
- Root concludes: a model-error decision (`problem`, category `model`, `abort`), a question to the user per missing public decision, or a satisfied result.
- No required check, probe, write or publication runs: a check judges artifacts, not the model.

**Owner.** Every question of Root reaches the owner channel. The attempt script answers each with one fixed deferral that settles nothing ("No owner decision is available during this assessment. Do not assume an answer. …"), so the team concludes within the bound model. An owner clarification or a model change belongs to the owner after the trial and needs new checks and a new assessment binding; no Agent edits authority inputs.

**Conclusions.** [assess.py](assess.py) classifies each attempt from the common envelope alone. These are payload classifications, not new lifecycle or Pass verdicts:

| Conclusion | Observed in the outcome |
| --- | --- |
| `accepted` | The run verified: the assessment Pass was clean, no genuine finding open. |
| `needs_clarification` | Root asked the user, and the sealed last Pass needs correction for an open genuine finding. |
| `model_error` | Root's model-error decision ended the run Failed. |
| `incomplete` | Anything else: timeout, budget, a failed turn, an unfinished Pass, findings left open without a question to the user, a preparation refusal. Technical incompleteness is never an acceptance or a rejection of the model. |

**Controls.** Session B's independently checked model controls: public gap, permitted freedom, contradiction and complete model, each with three fresh repetitions. Their expectation (the conclusion, the Definitions and terms of the known issue, the Definitions of the permitted freedom) is transcribed at freeze into a hidden control file.

**Measures.** Per attempt `assess.py attempt` records the conclusion and whether it meets the expectation; whether a blocking signal (question, genuine finding, model error) matches the known issue, and missed issues; false-alarm candidates (blocking signals that do not match it) and signals that cite the permitted freedom; blocking signals that cite no Definition of the model; returned results and disputes (disagreement); coverage of accepted results by model sources; questions and their length, genuine findings and notes (owner effort); turns, usage and wall time. `assess.py series` aggregates per control, including agreement across repetitions. Matching is an aid: the evaluator reads every signal and decides whether it is a sourced, relevant finding, a false alarm or an unfounded requirement, and whether its interpretations and consequences are concrete.

**Independence.** The mandate and guidance were written from the handoff's assessment questions before Session B delivered its controls and were never run against a control; they name no control-specific issue (a self-test guards this). Their author knew the earlier Apply trial cases, such as the unstated maximum of S-QV and the contradiction of S-X. The series runs them unchanged. Changing the treatment after seeing results is development: such a change needs new, unseen controls before it can be called validated.

## Running a series

In WSL Ubuntu with Docker, after the owner's start signal and only while no other provider series uses the host:

```text
MARKITECT_SOURCE_REPO=CHECKOUT bash experiments/model-assessment/run/build.sh SHA
python3 -B experiments/model-assessment/run/freeze.py SPEC FROZEN
bash FROZEN/kit/model-assessment/run/attempt.sh FROZEN CONTROL C-01
python3 -B FROZEN/kit/model-assessment/assess.py series FROZEN/attempts/*/out/assessment.json
```

`freeze.py`'s docstring defines SPEC: the build, the source kit and per control the packet (a materialization command or an existing repository with its head) and its expectation. The freeze copies the kit, builds and checks the packets, writes the hidden control files and a manifest of every hash. `attempt.sh` verifies the frozen Pass, harness and packet, refuses beyond 12 attempts or 90 summed minutes, caps an attempt at 10 minutes (an operator deadline 90 s before the cap leaves time for cleanup), refuses while any container with a `markitect.store` label runs, binds Sonnet with effort high, runs one attempt with its own store label, cleans up only that label and appends the attempt to the ledger even when it failed to start. Name attempts neutrally and interleave the controls in an order fixed before the first attempt.

## Files

| Path | Purpose |
| --- | --- |
| [assessment-pass.json](assessment-pass.json) | The frozen treatment: purpose, mandate, guidance and correction purpose of the assessment Pass. |
| [assess.py](assess.py) | Preflight of the Pass declaration, classification, scoring against a hidden control and the series summary. |
| [test_assess.py](test_assess.py) | Provider-free self-tests: `python -B -m unittest discover -s experiments/model-assessment`. |
| [run/build.sh](run/build.sh) | Builds the candidate's images and CLI with the trial kit and compiles the harness binary. |
| [run/freeze.py](run/freeze.py) | Freezes kit, packets, hidden expectations and the manifest before any provider work. |
| [run/attempt.sh](run/attempt.sh) | One attempt within the ceilings, with the deferring owner, cleanup, scoring and the ledger entry. |

## Limits

- One fixed mandate, one model family (Sonnet), small controls and three repetitions per control: a diagnostic sample, not a reliability rate or a comparison of assessment designs.
- The deferring owner tests whether a question reaches the owner, not how a team uses an answer.
- Signal matching is textual. A false alarm, an unfounded requirement and source quality are the evaluator's judgment on the listed signals.
- The assessment sees the model through the same Briefs as Apply; a gap that the Briefs hide from a Manager is a finding about the Briefs, not about the team.
