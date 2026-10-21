# Model assessment trial

Session C's method trial for the mandatory Model assessment ([handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-c-model-assessment-before-realization), [workflow](../../docs/design/canonical-model/manager-apply-workflow.md#mandatory-model-assessment-and-closed-runs)). After the deterministic Model check and File check, does a read-only Agent team find the material gaps and contradictions of the bound model, with sources, before any realization? And does it accept complete models that leave optional details out, state an unusual but valid design or delegate choices explicitly ([conformance mandate](../../docs/design/canonical-model/manager-apply-workflow.md#conformance-mandate-and-observation-kinds))? The isolated harness exercises the product's assessment-only entry and adds no Command or additional production gate. [contract.md](contract.md) states the method and contract for Session A, which owns the product integration. A positive conclusion is no completeness proof and does not replace independent artifact Verify.

**Current successor scope, 21 October 2026:** method review and provider-free harness preparation only. Provider evaluation is paused and requires new explicit authority; earlier startup wording and the ceilings below do not grant it. The harness uses A's production assessment integrated in [#566](https://github.com/Glacius-Labs/Markitect/pull/566). Build/freeze and wiring evidence wait for B's reviewed integrated controls.

**Everything here is operator material.** Agents receive only a packet: the frozen control's repository. Expectations, this guide, the scorer and the ledger stay outside every packet and mount.

## Method

**Preconditions.** Every packet passes `markitect check` (Model check and strict File check) when the series is frozen, and the harness's preparation runs both again before any turn; a failing stand is refused before work, as in Apply.

**Treatment.** One review Pass of the one Pass process (`apply.AssessCaptured(ctx, req)`) on the captured stand, driven by the grant-gated `TestModelAssessmentHarness` in `internal/bootstrap` with the product's real preparation and adapters. Execution uses the built-in `assessSpec`; [assessment-pass.json](assessment-pass.json) is its hashed reference copy, not another execution specification. `TestAssessCapturedRunsOneReadOnlyReviewWithoutChecksOrPublication` checks that reference's mandate and guidance against the actual prompts sent to the model's reviewers:

- The team is the model's Manager tree: Root and one reviewer per Package, so its size follows the model. Every reviewer starts fresh and reads the stand read-only.
- The test is materiality: is the model usable, materially consistent and decisive enough for its realization? Not: is it as detailed as possible, or does it follow a preferred architecture? Each reviewer asks four questions about its own obligations, each covered with an inspected model source: are the outcomes users or other areas rely on stated, derivable or explicitly delegated; do statements contradict; are responsibilities and contracts usable; and which freedom is permitted.
- Derive the material conditions, exceptions and interactions from the complete bound statements and explicit purpose. A list of Definition IDs or an author's condition list is not complete semantic coverage. Parents independently challenge acceptance as well as findings and assess the actual fit of their children's contracts. A missing material public decision cannot become implementation freedom merely through that label.
- The common observation kinds, applied to a model:
  - A *model problem* is a genuine finding with its basis and class. `model_gap` covers a missing material decision or an unusable contract; `model_error` covers a contradiction or an erroneous statement.
  - A *suspected* problem is investigated inside the Pass, by lookup and questions along the tree. It ends as a model problem, is resolved, or keeps the reviewer working. It is never parked as a note.
  - A *suggestion* is at most a non-blocking note: missing optional detail, an ordinary implementation choice, an unusual but consistent design, delegated freedom, style or preference.
- Parents assess interfaces and cross-area consistency. They also judge the basis and class of their children's findings: they accept or return them, and dispute only an unfounded finding. A real problem with the wrong class is returned for the class to be corrected, never disputed, so that a disproof cannot remove a real gap. Root confirms the conclusion in its integrated result and disputes every open finding it does not confirm. While only fresh disputes keep a Pass from being clean, a new complete Pass runs on the unchanged stand, as in Apply, within the packet's Pass budget and the attempt's deadline. A resolved or disproved finding is not raised again on the same statements without new evidence.
- The classes are the corrections `[model_error, model_gap, unclassified]`. The tool's default is the last, so an omitted class stays visible as `unclassified` instead of passing for a gap.
- No required check, probe, write or publication runs: a check judges artifacts, not the model.

**Closed Run.** No owner channel, no earlier answers and no owner decisions reach the team; a question to the user is refused. Questions stay inside the Manager tree. A material decision the model lacks is a model problem that ends the Run, and the corrected model belongs to a new Run.

**Conclusions.** [assess.py](assess.py) classifies each attempt from the common envelope alone. These are payload classifications, not new lifecycle or Pass verdicts. A finding is *confirmed* when it is open, genuine, raised by the team and not disputed in the last Pass:

| Conclusion | Observed in the outcome |
| --- | --- |
| `accepted` | The assessment Pass sealed `Satisfied`, the assessment-only Run ended `Incomplete` with `AssessmentAccepted` and the harness reported no error. This verifies no artifact. |
| `model_gap` | Root's integrated result sealed the Pass `NeedsCorrection` with confirmed findings, at least one `model_gap` and none `model_error`. |
| `model_error` | As above with at least one confirmed `model_error`. As a second route, Root's model-error decision (`problem`, `abort`) ended the run: `model_error` unless its free text names `model_gap`. |
| `model_problem` | Confirmed findings whose class was omitted (`unclassified`): the Run is blocked, the class unknown. |
| `incomplete` | Anything else, never an acceptance or a rejection of the model: `limit` (deadline or budget), `unresolved` (a sealed negative Pass without a confirmed finding), `technical` (a failed or uncertain turn, an unsealed Pass, a harness error) or `preparation`. |

The production assessment ends a confirmed problem `Failed` with `run.ModelAssessmentReason`. A model-only acceptance ends `Incomplete` with `apply.AssessmentAccepted` because the entry realizes and verifies nothing. The scorer recognizes those exact production boundaries; it does not treat any other incomplete Run as acceptance ([contract](contract.md#production-integration)).

**Controls.** Session B's independently checked model controls: public gap, permitted freedom, contradiction and complete model, three fresh repetitions each. The complete and freedom controls should be versions that omit optional details or state an unconventional but valid choice; both must be accepted while the gap and contradiction controls are still rejected. Each control's expectation is transcribed at freeze into a hidden control file, together with the packet's Definition identities and bodies:

- the conclusion;
- the terms and Definitions of the known issue and of the permitted freedom;
- named observations, such as known minor ambiguities and invented requirements.

A signal concerns a target by its terms when the target has any, otherwise by a cited Definition.

**Measures.** Per attempt `assess.py attempt` records:

- the conclusion, whether it blocks, and whether it meets the expectation, including the class of the finding on the known issue (an unrelated finding's class does not decide it);
- matches and misses: whether a confirmed finding matches the known issue, missed issues, and false aborts (a block on a control that expects acceptance);
- false-alarm candidates (confirmed findings that do not match the known issue), with the signals that cite the permitted freedom or concern one of the control's named observations;
- confirmed findings that cite no Definition of the model;
- disagreement: returned results, disputes, disputed findings left open, disproved findings and findings raised again after a disproof;
- internal questions, and attempted questions to the user (`send` to the user or Root's `escalate`, both refused);
- coverage of accepted results by model sources, and the actual delivery of each reviewer's own Definitions in its first sent prompt: identity with full body, reference only, or missing;
- owner effort: confirmed findings and their length, plus notes;
- turns, usage and wall time.

`assess.py series` aggregates per control, including agreement across repetitions and attempts the ledger holds but nobody assessed. Matching is an aid: the evaluator reads every signal and decides whether it is a sourced, relevant model problem, a false alarm, an unfounded requirement or a fair suggestion, and whether its outcomes and its reason that the model cannot settle it are concrete.

**Independence.**
- **Origin.** The mandate and guidance were written from the handoff's assessment questions before Session B delivered its controls, and they have never run against a control.
- **First revision.** They were revised for the owner's closed-Run policy (#560: no user questions; classed findings).
- **Second revision.** Revised again for the conformance mandate (#561: materiality, observation kinds, suggestions, the explicit `unclassified` class), following the workflow's own wording. The second revision came after the author had read Session B's draft assessment key for the earlier control versions (its required findings, minor ambiguities and invented requirements). It takes no example from that key, but this is disclosed.
- **Review correction.** An independent review then found that the dispute rule could disprove a real but misclassified gap. A wrong class is now returned instead of disputed. This was a rule defect found by review, not by a result.
- **No control names.** The treatment names no control-specific issue; a self-test guards this.
- **Earlier cases.** The author also knew the earlier Apply trial cases: the unstated maximum of S-QV, the shortfall policy of S-Q and the contradiction of S-X. The mandate's generic outcome categories (results, acceptance or refusal) overlap them.
- **Unseen controls.** The strengthened complete and freedom versions B prepares after this revision remain unseen until the freeze transcribes them.
- **No tuning on results.** The series runs the treatment unchanged. Changing it after seeing results is development and needs new, unseen controls before it can be called validated.

## Running a series

In WSL Ubuntu with Docker, after the owner's start signal and only while no other provider series uses the host:

```text
MARKITECT_SOURCE_REPO=CHECKOUT bash experiments/model-assessment/run/build.sh SHA
python3 -B experiments/model-assessment/run/freeze.py freeze SPEC FROZEN
bash FROZEN/kit/model-assessment/run/attempt.sh FROZEN CONTROL C-01
python3 -B FROZEN/kit/model-assessment/assess.py series --ledger "$(python3 -B FROZEN/kit/model-assessment/run/freeze.py ledger FROZEN)" FROZEN/attempts/*/out/assessment.json
```

`freeze.py`'s docstring defines SPEC:
- the build;
- the experiments directory of the clean clone it was built from;
- the ledger of an earlier freeze of the same series, which all successor freezes share with its lock;
- per control, the packet (a materialization command or an existing repository with its head) and its expectation.

The freeze copies that directory as the kit, refuses an invalid Pass declaration (`assess.py preflight`), builds and checks the packets, writes the hidden control files and a manifest of every hash. It binds the complete kit file set, build declaration and runtime environment as well as the binaries and images. The ceilings hold for the series: successor freezes use the same live ledger and lock, so attempts through an earlier freeze still count. `freeze.py ledger FROZEN` prints that shared path; use it with `series --ledger`, and pass the earlier freezes' assessment files too, or their attempts count as unassessed. Do not reuse an older freeze whose scripts copied ledgers; prepare a new freeze carrying all consumed attempts instead.

`attempt.sh` runs one attempt:
1. It recomputes every frozen hash, the exact kit file set and the build's image IDs (`freeze.py verify`).
2. Under a lock on the ledger it reads the manifest's ceilings, model and effort, and refuses beyond 12 attempts or 90 summed minutes. It also refuses while any container with a `markitect.store` label runs.
3. It writes the attempt's ledger line before anything starts, marked running and charged with the whole cap of at most 10 minutes. The operator deadline lies 90 s before the cap, so cleanup fits.
4. It binds the manifest's model (Sonnet) and effort (high), and refuses a bound profile that differs from them or still carries the demonstration's Go sentence. It runs the attempt with its own store label.
5. On every exit it cleans up only that label and replaces the line with the actual charge, labelled `interrupted` after INT or TERM. A failed cleanup or unavailable container observation records `status=cleanup_failed` and blocks further attempts. An unavailable Docker observation is `alive_after=unknown`.

A killed attempt keeps its pessimistic line. Resolve and observe its resource cessation before marking a running or cleanup-failed line `status=abandoned`; retain its charge. Name attempts neutrally (letters, digits, hyphens) and interleave the controls in an order fixed before the first attempt.

`ASSESS_SMOKE_NO_CREDENTIAL=1` runs the same script without a provider: the bound token is removed before the run, so every turn fails at its credential read. Use it only on a separate smoke freeze, whose ledger is not the series'.

## Files

| Path | Purpose |
| --- | --- |
| [contract.md](contract.md) | The method and contract for the mandatory gate, and what production must add. |
| [assessment-pass.json](assessment-pass.json) | Hashed reference of the built-in treatment: purpose, mandate, guidance and the classes of a confirmed finding. |
| [assess.py](assess.py) | Preflight of the Pass declaration, classification, delivery and scoring against a hidden control, and the series summary. |
| [test_assess.py](test_assess.py) | Provider-free self-tests: `python -B -m unittest discover -s experiments/model-assessment`. |
| [run/build.sh](run/build.sh) | Builds the candidate's images and CLI with the trial kit and compiles the harness binary. |
| [run/freeze.py](run/freeze.py) | Freezes kit, packets, hidden expectations and the manifest before any provider work, and verifies them before each attempt. |
| [run/attempt.sh](run/attempt.sh) | One attempt within the ceilings in a closed Run, with cleanup, scoring and the ledger entry. |

## Limits

- One fixed mandate, one model family (Sonnet), small controls and three repetitions per control: a diagnostic sample, not a reliability rate or a comparison of assessment designs.
- Signal matching is textual. A false alarm, an unfounded requirement and source quality are the evaluator's judgment on the listed signals. The class of Root's abort route is read from its free text.
- Delivery compares whitespace-normalized bodies with the sent prompt text; structured fields are not compared, and a delivered body is no evidence that an Agent read or followed it.
- Complete IDs, body delivery and source references do not show that every material clause was assessed. Technical context delivery, semantic model quality and harness wiring are separate evidence claims.
- The assessment sees the model through the same Briefs as Apply; a gap that the Briefs hide from a Manager is a finding about the Briefs, not about the team. The Briefs still name the project's required checks, which do not run.
- The Runtime profile comes from the demonstration harness (`ra_bind.sh`), whose Agent system prompt frames the team as applying a model. The attempt removes only its Go sentence, which states a technology the model may not.
