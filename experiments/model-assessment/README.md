# Model assessment trial

Session C's method trial for the mandatory Model assessment ([handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-c-model-assessment-before-realization), [workflow](../../docs/design/canonical-model/manager-apply-workflow.md#mandatory-model-assessment-and-closed-runs)). After the deterministic Model check and File check, does a read-only Agent team find the material gaps and contradictions of the bound model, with sources, before any realization, without inventing requirements or objecting to permitted freedom? The harness is isolated evaluation material: no Command, no Apply step, no production hook. [contract.md](contract.md) states the method and contract for Session A, which owns the product integration. A positive conclusion is no completeness proof and does not replace independent artifact Verify.

**Everything here is operator material.** Agents receive only a packet: the frozen control's repository. Expectations, this guide, the scorer and the ledger stay outside every packet and mount.

## Method

**Preconditions.** Every packet passes `markitect check` (Model check and strict File check) when the series is frozen, and the harness's preparation runs both again before any turn; a failing stand is refused before work, as in Apply.

**Treatment.** One review Pass of the one Pass process (`apply.AssessCaptured`) on the captured stand, driven by the grant-gated `TestModelAssessmentHarness` in `internal/bootstrap` with the product's real preparation and adapters. Its purpose, mandate and guidance are frozen in [assessment-pass.json](assessment-pass.json):

- The team is the model's Manager tree: Root and one reviewer per Package, so its size follows the model. Every reviewer starts fresh and reads the stand read-only.
- Each reviewer assesses its own obligations against four questions: are public decisions stated, derivable or explicitly delegated; do statements contradict; are responsibilities and contracts usable; and which freedom is permitted. It covers each obligation with an inspected model source.
- Parents assess interfaces and cross-area consistency and challenge their children's findings and claims: they accept, return or dispute them. While only fresh disputes keep a Pass from being clean, a new complete Pass runs on the unchanged stand, as in Apply, within the packet's Pass budget and the attempt's deadline.
- Root confirms the conclusion in its integrated result. A confirmed finding carries its class, `model_gap` or `model_error`, in the finding's correction field.
- No required check, probe, write or publication runs: a check judges artifacts, not the model.

**Closed Run.** No owner channel, no earlier answers and no owner decisions reach the team; a question to the user is refused. Questions stay inside the Manager tree. A material decision the model lacks is a finding that ends the Run, and the corrected model belongs to a new Run.

**Conclusions.** [assess.py](assess.py) classifies each attempt from the common envelope alone. These are payload classifications, not new lifecycle or Pass verdicts:

| Conclusion | Observed in the outcome |
| --- | --- |
| `accepted` | The run verified: the assessment Pass sealed `Satisfied` and the harness reported no error. |
| `model_gap` | Root's integrated result sealed the Pass `NeedsCorrection` for open genuine findings of the team, none classed `model_error`. |
| `model_error` | As above with at least one `model_error` finding, or Root's model-error decision ended the run (route `abort`). |
| `incomplete` | Anything else: timeout, budget, a failed turn, an unsealed Pass, a harness error, a preparation refusal. Technical incompleteness is never an acceptance or a rejection of the model. |

The harness ends a confirmed problem `Incomplete` with the reason `the model assessment confirmed a material model problem`, because the caller of `run.Finish` cannot end a sealed Pass `Failed`; the production gate maps it to `Failed` ([contract](contract.md#what-production-must-add)).

**Controls.** Session B's independently checked model controls: public gap, permitted freedom, contradiction and complete model, each with three fresh repetitions. Their expectation (the conclusion; the terms and Definitions of the known issue and of the permitted freedom; named observations such as known minor ambiguities and invented requirements) is transcribed at freeze into a hidden control file, together with the packet's Definition identities and bodies. A signal concerns a target by its terms when the target has any, otherwise by a cited Definition.

**Measures.** Per attempt `assess.py attempt` records:

- the conclusion, whether it blocks, and whether it meets the expectation, including the class;
- whether a confirmed finding matches the known issue, missed issues, false aborts (a block on a control that expects acceptance), and false-alarm candidates (confirmed findings that do not match the known issue), with signals that cite the permitted freedom and the control's named observations each signal concerns;
- confirmed findings that cite no Definition of the model;
- disagreement: returned results, disputes and disproved findings;
- internal questions, and attempted questions to the user (refused);
- coverage of accepted results by model sources, and the actual delivery of each reviewer's own Definitions in its first sent prompt: identity with full body, reference only, or missing;
- owner effort: confirmed findings and their length, plus notes;
- turns, usage and wall time.

`assess.py series` aggregates per control, including agreement across repetitions and attempts the ledger holds but nobody assessed. Matching is an aid: the evaluator reads every signal and decides whether it is a sourced, relevant finding, a false alarm or an unfounded requirement, and whether its outcomes and its reason that the model cannot settle it are concrete.

**Independence.** The mandate and guidance were written from the handoff's assessment questions before Session B delivered its controls and were never run against a control. They were revised once for the owner's closed-Run policy of 21 October 2026 (no user questions; classed findings), again without any result. They name no control-specific issue (a self-test guards this). Their author knew the earlier Apply trial cases, such as the unstated maximum of S-QV, the shortfall policy of S-Q and the contradiction of S-X, and the generic categories of public decisions in the mandate (values, limits, outputs, refusals) overlap them. The series runs the treatment unchanged. Changing it after seeing results is development and needs new, unseen controls before it can be called validated.

## Running a series

In WSL Ubuntu with Docker, after the owner's start signal and only while no other provider series uses the host:

```text
MARKITECT_SOURCE_REPO=CHECKOUT bash experiments/model-assessment/run/build.sh SHA
python3 -B experiments/model-assessment/run/freeze.py freeze SPEC FROZEN
bash FROZEN/kit/model-assessment/run/attempt.sh FROZEN CONTROL C-01
python3 -B FROZEN/kit/model-assessment/assess.py series --ledger FROZEN/ledger.txt FROZEN/attempts/*/out/assessment.json
```

`freeze.py`'s docstring defines SPEC: the build, the experiments directory of the clean clone it was built from, and per control the packet (a materialization command or an existing repository with its head) and its expectation. The freeze copies that directory as the kit, builds and checks the packets, writes the hidden control files and a manifest of every hash.

`attempt.sh` first recomputes every frozen hash (`freeze.py verify`). Under a lock on the ledger it refuses beyond 12 attempts or 90 summed minutes, refuses while any container with a `markitect.store` label runs, and writes the attempt's ledger line before anything starts, marked running and charged with the whole cap of at most 10 minutes. The operator deadline lies 90 s before the cap, so cleanup fits. It binds Sonnet with effort high and runs the attempt with its own store label. On every exit it cleans up only that label and replaces the line with the actual charge. A killed attempt keeps its pessimistic line, and a running line refuses further attempts until the operator marks it `status=abandoned`. Name attempts neutrally and interleave the controls in an order fixed before the first attempt.

`ASSESS_SMOKE_NO_CREDENTIAL=1` runs the same script without a provider: the bound token is removed before the run, so every turn fails at its credential read. Use it only on a separate smoke freeze, whose ledger is not the series'.

## Files

| Path | Purpose |
| --- | --- |
| [contract.md](contract.md) | The method and contract for the mandatory gate, and what production must add. |
| [assessment-pass.json](assessment-pass.json) | The frozen treatment: purpose, mandate, guidance and the classes of a confirmed finding. |
| [assess.py](assess.py) | Preflight of the Pass declaration, classification, delivery and scoring against a hidden control, and the series summary. |
| [test_assess.py](test_assess.py) | Provider-free self-tests: `python -B -m unittest discover -s experiments/model-assessment`. |
| [run/build.sh](run/build.sh) | Builds the candidate's images and CLI with the trial kit and compiles the harness binary. |
| [run/freeze.py](run/freeze.py) | Freezes kit, packets, hidden expectations and the manifest before any provider work, and verifies them before each attempt. |
| [run/attempt.sh](run/attempt.sh) | One attempt within the ceilings in a closed Run, with cleanup, scoring and the ledger entry. |

## Limits

- One fixed mandate, one model family (Sonnet), small controls and three repetitions per control: a diagnostic sample, not a reliability rate or a comparison of assessment designs.
- Signal matching is textual. A false alarm, an unfounded requirement and source quality are the evaluator's judgment on the listed signals.
- Delivery compares whitespace-normalized bodies with the sent prompt text; structured fields are not compared, and a delivered body is no evidence that an Agent read or followed it.
- The assessment sees the model through the same Briefs as Apply; a gap that the Briefs hide from a Manager is a finding about the Briefs, not about the team. The Briefs still name the project's required checks, which do not run.
- The Runtime profile keeps the product's Agent system prompt ("a team that applies a repository's canonical model"); the attempt removes only the demonstration's Go sentence, which states a technology the model may not.
