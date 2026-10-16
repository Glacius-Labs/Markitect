# Demonstrating a usable working day

Date: 16 October 2026. Proposed acceptance procedure preserved at the [owner's request](../sources/working-day-discussion-20261016.md#preserve-the-working-day-demonstration). This develops the [first usable version boundary](first-usable-version-20261016.md#usable-working-day). It is not an executed result, an accepted release gate or evidence of general reliability.

The question is whether the owner can remain responsible for intent and decisions while Markitect organizes implementation, review, correction and continuation. Correct artifacts alone do not establish a usable working day if the owner must keep scheduling Agents or reconstructing lost work.

## Fix the starting point and expectations

The prerequisite is the separate [technical-core demonstration](first-usable-version-20261016.md#technical-core): real Managers realize library M1 as A1, then apply M2's change from five permitted open loans to three against that actual implementation to produce A2. This procedure starts only after that prerequisite has been demonstrated; this document does not claim it has already happened.

Before execution, record:

- The exact model, starting artifacts and registry, Markitect build, execution profile and configuration.
- The assignment, mandates, shared budget, time limits and protected review/integration capacity.
- Independently authored business expectations and the allowed owner interactions.
- The intended fault or interruption, its insertion point and the evidence required for each scenario.

Use four separate scenarios from a recorded baseline. Use real Manager and reviewer execution for the working-day evidence. Controlled fault injection makes the scenario reproducible; fake-adapter tests can support implementation checks but cannot substitute for this demonstration. Injection must preserve the normal candidate and review contracts, including content bindings.

Expected behavior is fixed before the run and checked against the actual resulting artifacts. Agent-written tests and reports are evidence to inspect, not their own acceptance authority. Preserve all attempts, including failures and manual interventions.

## Four scenarios

| Scenario | Setup | Passing behavior |
| --- | --- | --- |
| Question and continuation | Prepare a new, explicitly scoped one-time loan-renewal extension. Fix its other rules in advance, but reserve the decision whether overdue loans may be renewed for the owner. Arrange for useful independent candidate work to exist before the answer. | The organization identifies the missing authority, asks a concrete question with a proposed model edit, binds the answer to that question, observes the actual authorized model change and continues the same assignment in a successor Apply. Suitable work survives; previous approval does not. Required assessment covers the current result. |
| Internal correction | Introduce a controlled candidate defect that permits another loan when the current limit has already been reached, before independent review. | The reviewer independently identifies the violation, the responsible Manager corrects it, and renewed assessment verifies the correction and relevant existing behavior. The owner does not explain the defect, schedule review or repair the code. |
| Interruption and recovery | Interrupt execution after candidate work exists but before its completion is unambiguously confirmed. Record the actual injection point. | On resumption, Markitect reconciles observed execution and artifact state before allowing another writer. It preserves useful work and consumed budget, avoids blind duplicate effects and competing writers, and continues through required assessment. If activity remains uncertain, it stays visibly blocked rather than pretending recovery succeeded. Such a safe block alone does not pass the complete recovery scenario. |
| Insufficient budget | Give the assignment deliberately insufficient unprotected capacity to finish. Keep completion reserves and limits explicit. | Markitect stops within the configured bounds, retains candidates and findings, and reports the unfinished work and any concrete request for additional capacity. It neither spends protected verification capacity on extra implementation nor waives checks or resets consumption. |

Run the cases separately so that failures have an identifiable cause. The renewal case is a new extension: the existing [library M1](../../design/canonical-model/library-example/README.md) excludes renewals and must not be portrayed as ambiguous. The prepared extension explicitly changes that scope. An unresolved business decision belongs to the recorded question/assignment; it must not be treated as permission to invent final model semantics.

## Follow the question through to the result

An example question in the prepared renewal scenario is:

> **Decision needed:** May an overdue loan be renewed?
>
> **Recommendation:** No.
>
> **Proposed model change:** Renewal is permitted only when the request date is on or before the existing due date. Refusal leaves the loan unchanged.
>
> **Work impact:** The eligibility condition waits for this decision. Completed independent work is retained.

The actual question also identifies its source basis and exact proposed model patch. The owner answers that specific question with “Accept the recommendation.” Record these as separate observable steps:

1. The answer is associated with the correct question and source revision.
2. The authorized patch actually changes the model. An answer alone does not imply a successful edit.
3. A successor Apply captures the actual new model while retaining the assignment identity and consumed budget.
4. Existing work receives a current suitability judgment: Keep, Adapt, Replace or Unresolved. Record the retained candidate identity and its use so that rebuilding everything cannot masquerade as reuse.
5. Required reviews and checks assess the current inputs and candidate. Earlier approvals cannot authorize the new result.
6. Guarded publication and read-back establish which artifacts were actually applied.

For the first version, the owner or calling coding agent may apply the exact authorized patch outside artifact Apply and invoke continuation. That explicit convenience limit does not permit recreating child assignments, manually assigning reviewers or transferring candidate files by hand.

Independent behavior checks include a permitted timely renewal, refusal of an overdue renewal without state change, refusal of a second renewal and preservation of existing borrowing and return rules. The prepared scenario must define the other renewal semantics, including the new due date, before execution; the example question is not a complete feature specification.

## Allowed human intervention

Record each intervention with its reason and effect, not only a total count. Required owner decisions are expected behavior; routine management performed by the owner is a failure of this acceptance objective.

| Allowed first-version interaction | Intervention that fails the affected scenario |
| --- | --- |
| Start an assignment and inspect its compact status. | Remind Managers which routine step to take next or manually route their work. |
| Answer a genuinely reserved business question. | Re-explain a rule already available in the selected model to rescue the run. |
| Apply the exact authorized model patch. | Repair implementation or tests manually so the scenario can finish. |
| Invoke supported continuation or restart. | Recreate child assignments, select reviewers or reconstruct lost candidates. |
| Explicitly grant additional budget when requested. | Reset the budget ledger, bypass checks or silently enlarge authority. |

Any requested budget extension starts a recorded continuation; it does not turn the original insufficient-budget outcome into a completed assignment. Fault injection by the evaluator is separately recorded as experiment setup and is not an owner intervention. Subsequent evaluator coaching or repair is not setup.

Independent evaluation of the final result is part of this demonstration, not a requirement that the owner review every routine production diff. No unrecorded human assistance is allowed behind the scenes.

## Status and evidence

An observer should be able to answer the following from the compact status and linked evidence without reconstructing Agent transcripts:

- What is finished, and what remains?
- Which owner decision is needed, and what depends on it?
- What is blocked or uncertain?
- What budget has been consumed, allocated, protected and left available?
- What is locally checked, checked as a whole and actually applied?

Applied artifacts do not establish merge, release, deployment or correct production operation. Preserve those boundaries in the report.

A small result record per scenario is sufficient: fixed starting basis, injection details, observed events and candidate references, owner interactions, budget/time consumption, independent behavior results and a pass/fail judgment against each criterion. Evidence must distinguish what was actually observed from an Agent's claim. A convenient status report cannot replace the underlying evidence.

## A small independent assessment

This proposal answers the [owner's follow-up](../sources/working-day-discussion-20261016.md#keep-independent-assessment-small). Reuse the proposed [external Playground](../../design/canonical-model/manager-apply-playground.md) and [independent library evaluator](../../design/canonical-model/library-example/evaluation-contract.md). Those documents own the runner, interface, isolation, frozen inputs and business-case families. Extend them with the four working-day cases when implementing this proposal; do not build a second orchestration system or duplicate their specifications here. These designs do not yet establish an implemented evaluator or a passed experiment.

### Three complementary evidence sources

| Evidence | Smallest useful check | Boundary |
| --- | --- | --- |
| Actual library behavior | Build the captured artifact tree and call its public JSON Lines interface from the existing proposed external Python standard-library evaluator. Assert returned values and observable state before/after operations against independently authored expectations. | Generated tests and self-reports are not the oracle. An API trace establishes only the behavior exercised; retain the existing source inspection for obligations it cannot observe. |
| Actual artifact and execution history | Compare model patches, candidate/source identities and consistent Host history exports. Cross-check important transitions against the runner's actual invocations, provider/session lifecycle observations and publication read-back. | A Host event is stronger than an Agent's narrative but is still produced by the system under test. A history entry saying "verified" or "reconciled" is not sufficient by itself. Hashes bind bytes, not correctness. |
| A short independent judgment | Inspect the question, its authority/model effect, the review finding and the compact final status using a fixed checklist. Record reasons and links. | Semantic clarity and usefulness cannot be established by checking that fields are nonempty. A separate Agent may assist, but its approval alone does not establish acceptance. |

Use one frozen evidence bundle per scenario: source/model snapshots, relevant history export, external observations, behavior traces, owner-intervention record and verdict. Build from captured artifacts after writers are quiescent. Keep evaluator inputs outside worker write scope and use the Playground's declared enforcement; hashes alone are not isolation. Reuse its consistent history-export mechanism rather than copying a live database or adding another event store.

The independent evaluator is an experiment role, not another Manager. It can invoke the public assignment/answer/resume operations and the declared injection, but cannot route internal work, fabricate review findings, edit the result or rescue execution. Requirements and the public interface are visible to workers; assessment data must introduce no hidden business obligations.

### Concrete checks for the four scenarios

| Scenario | Independent observations and assertions |
| --- | --- |
| Question and continuation | Check the real question/answer association and the actual before/after model patch against the approved edit. Require the same assignment, a new run bound to the new model, preserved consumption and a retained candidate present in the successor's starting inputs. Check new required assessments against the actual current subject. Run timely/overdue/second-renewal cases and unchanged library cases. The renewal operation and remaining semantics require a separately frozen extension to the public contract; do not pretend M1's ten-operation interface already supports them. |
| Internal correction | Capture the defective candidate and show that the external limit case fails for the intended reason before correction. On the corrected artifact, the same case must pass: after three open loans, a fourth request is refused, the count remains three and the requested copy stays available. Inspect the independently produced review finding, its candidate binding, actual correction delta and renewed review. The external evaluator's pre-check remains outside the reviewer's input; otherwise it demonstrates following a supplied finding, not independent detection. If another check finds the defect before the reviewer, record that useful result but do not count it as evidence of reviewer detection. |
| Interruption and recovery | Record the controlled stop point and actual process/session identities outside the product history. Retain the interrupted workspace and state. Before another writer starts, require evidence of the previous writer's terminal state or effective loss of write authority, plus reconciliation of candidate/publication state. Check observed dispatch/write-authority intervals, retained candidate bytes and monotonically accounted consumption across resumption; then assess the final result. A finished artifact alone cannot reveal overlapping writers. If the chosen environment cannot establish these observations, mark the recovery criterion unproven. Evidence is limited to the injected interruption point, not every possible crash window. |
| Insufficient budget | Compare actual captured invocations with the frozen accounting unit and declared reservations/settlements. Check that new work stops at the applicable admission boundary, protected completion capacity is not allocated to ordinary implementation, and restart/successor does not reset consumption. Read back retained work and verify incomplete status and remaining need. Observe through confirmed quiescence, not merely a short quiet interval. Session/step accounting is not proof of a monetary spending ceiling unless actual cost accounting supports that claim. |

Reuse all applicable baseline business cases; only add cases for the new extension. For example, the borrowing-limit witness uses normal member/copy registration and three successful loans, records observable state, attempts the fourth loan and compares state again. A refusal message without unchanged state fails. This is a small concrete trace, not a reference implementation of the library.

For process checks, assert a few required relationships rather than a complete prescribed timeline: answer before the authorized model transition, current input/candidate bindings for assessment, no new writer while prior authority is unresolved, and no reset of shared consumption. Do not require a particular delegation order when several valid orders exist. Reuse product exports as data without importing product scheduling, approval or accounting code as the independent expected-result implementation.

### Keep human judgment bounded

Use three short questions for the evaluator's inspection:

1. Did the owner question identify a genuinely reserved decision, offer a concrete authorized change and describe its effect on ongoing work?
2. Did the independent review finding identify the seeded violation and require an actual correction, with evidence tied to the relevant candidate?
3. Does the compact status accurately convey decisions, blockers, budget and the distinction between local checks, aggregate verification and applied artifacts?

Confirm fresh reviewer execution from actual session/context preparation and candidate bindings. A different role label or session ID alone does not prove relevant context or fresh execution; even fresh execution cannot prove absence of correlated reasoning errors. The seeded defect provides a bounded demonstration of detection.

Record this evaluation after capture, without coaching the active organization. A one-time independent inspection of the acceptance procedure, its expectations and resulting evidence is compatible with removing routine owner code review. It does not require another continuing team, scoring model or full transcript reading for every run. A contradiction discovered during inspection remains a finding; it cannot be dismissed because mechanical checks passed.

### Small implementation and checks of the checker

The proposed addition is four explicit scenario drivers plus a few assertion functions and data files in the planned Playground, reusing its existing proposed manifest, capture and report shapes. Each driver contains preparation, public invocations, the declared injection and assertions. No scenario DSL, generic fault-injection platform, dashboard, duplicate Manager loop or new approval authority is needed. Select one concrete recovery interruption point first and disclose the untested windows.

Before relying on the added assertions, exercise a handful of controlled negative evidence copies outside real execution: a mismatched model/candidate binding, an old approval presented for the successor, missing writer-termination/fencing evidence and reset budget consumption. Each must prevent a pass for its intended reason. Reuse the library evaluator's existing proposed wrong-result checks, including mutation on refusal. These checks establish that the checker discriminates selected failures; they are not real working-day runs.

For every criterion report `passed`, `failed` or `not proven`, with its evidence reference. Missing or inconsistent evidence prevents overall acceptance, but distinguish a deficient capture/evaluator from an observed product defect. Re-evaluating a frozen bundle should require no new Agent execution. Bind the evaluator and case hashes for the repeatability series as well as the product/profile; preserve earlier results if the evaluator itself is corrected.

## Keep expectations subordinate to the model

This clarification answers the [owner's concern about a second specification](../sources/working-day-discussion-20261016.md#avoid-a-second-specification). The canonical model owns business intent. Tests and expectation sets are realizations of that intent, with provenance and revision bindings. Their maintenance is part of delegated work, not another requirements document the owner must keep synchronized. A failing test is evidence to investigate, not authority to override a newer authorized model.

Independent assessment still has a cost: a check must express enough of an obligation to distinguish correct and incorrect behavior. Eliminating every repeated expression would also remove much of its ability to detect mistakes. The aim is one source of authority and bounded, reviewed derivation, not zero test maintenance or a claim that model-to-test generation is infallible.

### Distinguish routine checks from frozen experiments

| Situation | Treatment |
| --- | --- |
| An explicit structured value changes | For routine project checks, read the value from the exact selected model snapshot and supply it as test data to a separately reviewed check. Do not read the generated implementation's constant, ask the program for its intended policy, or use a mutable "latest model". |
| The business meaning stays the same | Reuse the applicable check logic and test arrangements. A loan-limit check can attempt N successful loans and one refusal, then verify unchanged state on refusal. Its semantics still require independent review against the model. |
| The business meaning changes | Have the responsible realization work update affected tests and expectations, with fresh review from the changed model obligations. For example, a per-member limit becoming a per-household limit changes the check logic, not just a number. |
| Markitect itself is being evaluated | Keep the bounded experiment's independently authored M1/M2 expectations frozen and bound to those exact models. The five/three witnesses in the existing evaluator remain deliberate controls. They must not silently follow whichever model or implementation a failed run produced. |

The proposal to parameterize routine checks does not change the [existing library evaluation contract](../../design/canonical-model/library-example/evaluation-contract.md#independent-inputs-and-oracle). Its fixed profiles establish a particular historical result and are not a second living specification for every future library version. A later deliberate experiment has new bindings; old evidence is preserved rather than updated to appear current.

Reading a structured limit from the selected canonical model shares that input and its interpretation with the realization path; it is less independent evidence of model loading than the frozen M1/M2 witness. It can still independently test whether the resulting program enforces the selected limit. Keep these claims separate. Do not build a second generic model compiler or semantic interpreter into the evaluator to eliminate this tradeoff.

### A small maintenance rule

Each check or check family needs a reference to the model obligation it assesses and a binding to the model used for the result. Reuse existing assessment/registry metadata where possible; do not add another manually maintained requirements catalogue or a new Core construct for this purpose.

When intent changes:

1. Identify potentially affected checks from the changed obligations, their relevant relationships and the actual change. References help selection but are not proof of complete impact discovery; uncertain impact remains visible and requires broader assessment.
2. Reuse applicable logic and update structured parameters from the fixed new basis. If semantics changed, explicitly adapt or replace the affected checks; an obsolete expectation is retired with a reason and history.
3. Review the test change against the authorized model delta, separately from the implementation's explanation. A failed assertion alone is never permission to weaken or remove it. Required current assessments still run; this is not an approval-reuse or check-skipping policy.
4. Freeze the reviewed expectations for the next assessed attempt. If a defect in the evaluator is discovered, preserve the failed attempt and record a new assessment revision instead of rewriting the original result.

The owner resolves missing business meaning or authority, not routine expected-value updates. If a test introduces a new business rule with no model basis, either remove that unsupported requirement or propose an explicit model change for authorization. Useful owner-authored business examples may live with their canonical rule; they should not become a mandatory parallel catalogue of executable specifications.

Independence comes from deriving and reviewing checks against intent before using implementation output as evidence, protecting their accepted basis during an attempt, and executing them externally. It does not require different business rules. For plain-language obligations, automatic interpretation remains fallible: a fresh reviewer, a few deliberately chosen counterexamples and visible uncertainty provide bounded assurance without promising a universal semantic oracle.

For the working-day scenarios, stable process obligations such as continued assignment identity and no budget reset belong to Markitect's own contracts. Library loan-policy changes do not normally change those checks. Actual limits and expected authorized model patches are scenario inputs. This keeps the small process suite separate from changing library business values without maintaining another library specification.

## Reduce shared misunderstandings without more reviewers

This proposal answers the [owner's concern about correlated interpretation errors](../sources/working-day-discussion-20261016.md#reduce-shared-model-misunderstandings). Implementation, tests and a fresh reviewer can all misread the same rule. Their agreement is not independent proof of the intended meaning. No finite review chain eliminates that risk; improve the evidence produced by the existing Manager/reviewer team instead of adding permanent approval layers.

### Derive a discriminating case before reading the solution

For an affected rule with consequential ambiguity or a plausible common mistake, have the existing reviewer first read the canonical rule, its relevant relationships and authorized delta. Before reading the implementer's rationale or solution, record a short assessment note: the obligation, one plausible wrong interpretation and an observable case that distinguishes them. Then inspect the candidate, tests and their evidence. This is one review with an initial intent-reading phase, not another reviewer or a separate policy authority.

Keep the note small and tied to existing assessment records; do not request another prose specification for every Definition. Do not hide relevant model context to manufacture independence. If both phases share a session, the first must genuinely precede exposure to the solution. This reduces anchoring on the implementer's explanation but does not establish statistical independence or eliminate shared model biases.

The current library already provides useful cases without inventing new intent:

| Canonical meaning | Plausible wrong interpretation | Distinguishing observation |
| --- | --- | --- |
| [Suspension](../../design/canonical-model/library-example/m1/model/members/eligibility.yaml) prevents new borrowing and preserves return rights. | A suspended member cannot perform any lending operation. | Borrow while active, suspend, then attempt a new loan and return the existing loan. New borrowing refuses without changes; the return succeeds, closes exactly that loan and releases its copy. |
| The [loan limit](../../design/canonical-model/library-example/m1/model/lending/loan-policy.yaml) counts current open loans, excludes closed loans and is inclusive of the permitted resulting count. | Count all historical loans, or refuse the loan that would exactly reach the limit. | Reach N open loans, observe that request N succeeded, return one, then borrow another available copy. That request succeeds; the next over-limit request refuses unchanged. |
| A repeated [return](../../design/canonical-model/library-example/m1/model/lending/return.yaml) refers to a loan transaction. | Returning an old loan releases its copy even after that copy was lent again. | Return loan L1, lend the same copy as L2, then repeat L1's return. L2 stays open, its copy stays unavailable and L1's original return date is preserved. |

These cases challenge different meanings using normal public operations and actual state. A happy-path success, a refusal string or high code coverage does not establish that the check distinguishes the wrong meaning.

### Show that a check rejects the chosen mistake

For selected consequential rules, use a small controlled counterexample to the checker: for instance, a transcript/program stub that refuses a suspended member's return, or a disposable candidate copy that closes L2 when returning L1. The relevant check must fail for that semantic difference, not because the stub crashed, failed to build or violated an unrelated protocol detail. Correct behavior must also pass, so an always-failing checker is not useful evidence.

Reuse the existing proposed evaluator self-checks and seeded review scenario. Do not add a generic mutation-testing platform or a target mutation score. A few representative mistakes at authorization boundaries, state transitions, limits and previously troublesome rules are more useful for this first version than a required artificial mutation of every rule. Disposable variants stay outside the actual applied artifacts; retain their identities and outcomes separately.

The seeded-review experiment and checker self-check answer different questions: the former asks whether the reviewer discovers an undisclosed defect, while the latter asks whether a check rejects a declared wrong behavior. Supplying the seed's diagnosis to the reviewer cannot count as independent discovery. If a chosen wrong variant survives, inspect the test and its model basis; do not simply add another approval. This demonstrates sensitivity to those selected mistakes only.

### Put genuine clarification in the canonical rule

If the relevant model statements already decide the case, the organization must resolve it from that context without asking the owner again. If two materially different behaviors remain permitted by genuinely unclear intent, use the existing decision route. The owner chooses business meaning; the authorized clarification or useful example belongs with the canonical rule. Ordinary v1 body text is sufficient; no mandatory example language, duplicated rule catalogue or new Core construct is needed.

An Agent-proposed example is still an interpretation until checked against existing authority or explicitly authorized as new meaning. Recording it beside a rule does not make it independent owner evidence. Examples should clarify discriminating boundaries and reference structured policy values where possible, rather than duplicate every value or enumerate a complete implementation.

### A bounded review result

For the selected rule, the existing review should be able to cite: canonical basis, distinguishing input/state, expected outcome, observed candidate behavior and the selected wrong-behavior check where used. These are evidence fields in an ordinary assessment, not new universal gates. Routine checks can remain simple; deeper challenge is concentrated where a wrong interpretation would matter or has previously escaped.

If implementation and assessment both infer the same wrong meaning and no authoritative example or discriminating observation challenges it, the result can still be wrong. Changing model vendors or adding votes does not remove that limitation. Honest evidence states which interpretations were challenged and which uncertainty remains. Repeated runs establish repeatability of the selected evidence, not immunity to a systematic misunderstanding. The library working-day acceptance remains bounded by its chosen cases.

### Select deeper checks with two questions

The [owner asked how to choose rules](../sources/working-day-discussion-20261016.md#select-rules-for-targeted-counterchecks) without adding a complex process. Use a short judgment within the existing review of affected obligations, not a risk score, another approval or a permanent classification of every model Definition. All mandatory checks and ordinary independent review still apply.

**First: is there a concrete reason to challenge this rule's interpretation?** One of the following is enough:

- A plausible error would cross an authority boundary, damage or incorrectly change existing state, or violate a central business constraint while an ordinary successful example could still look correct.
- The actual change exposes a plausible alternative meaning with different behavior: an exception, a limit boundary, identity across repeated operations or an interaction between responsibilities. Complexity or novelty alone is not the reason; name the potential mistake.
- A relevant previous failure or current contradiction shows that the existing checks may miss this behavior. This is a reason to inspect the present rule, not an automatic new business rule or permanent escalation requirement.

Start from the model before reading the solution, as described above. Update the judgment if candidate inspection exposes a new concrete concern. Do not invent arbitrary mistakes merely to fill a checklist; for a consequential boundary, identify at least the forbidden outcome that the assessment needs to exclude. If intent itself is unclear, use the existing clarification route rather than silently choosing an interpretation.

**Second: do the applicable checks already distinguish the relevant wrong behavior?** If yes, reuse the check and execute it against the current candidate, with current assessment bindings. If no, add the smallest discriminating case to the ordinary test/review work. Several rules may share a single case. A previous green run or test name is not enough to establish present sensitivity; inspect its assertions and applicability. Reuse of check logic is not reuse of old approval.

| Library change | Proportionate action |
| --- | --- |
| Correct a spelling mistake in usage documentation, with no change of meaning or executable interface. | Ordinary content review and applicable document checks. No fabricated behavioral counterexample. |
| Change the numeric loan limit while keeping its meaning. | Run the existing inclusive-boundary/open-loan/refusal-state cases with the new selected value. Add no new countercase if they already distinguish the relevant mistakes. Frozen M1/M2 experiment controls retain their original bindings. |
| Change member-status behavior affecting borrowing and return. | Exercise the suspension case across membership and lending; a successful active-member loan alone does not cover this boundary. |
| Change repeated-return handling or loan/copy identity. | Exercise L1 return, L2 borrowing of the same copy and repeated L1 return, checking L2 and availability. |
| Change which records count toward the loan limit. | Reassess the meaning of the existing limit cases; add a case for the newly relevant distinction if it is not covered. |

Testing the checker with a deliberately wrong variant remains a further bounded technique, not a requirement for every selected rule. Use it when the chosen check's ability to catch an important mistake is uncertain, a relevant defect previously escaped, or the frozen acceptance experiment expressly requires it. Otherwise an applicable, reviewed discriminating case with actual current results can be sufficient. Do not require fresh artificial mutations after every numeric or documentation change.

The existing reviewer owns this adequacy judgment; the responsible Manager arranges any missing case within the assignment. For selected cases, one short entry in the existing assessment is enough: reason, distinguishing case/evidence, and remaining gap. Example: "Repeated return could close the newer loan; existing L1/L2 case exercises that distinction and passes on the current candidate." Ordinary cases need no separate risk form beyond their normal review result.

Begin with one case per distinct concern and add another only when it addresses a named remaining gap. This is a starting heuristic, not a limit that permits uncovered obligations. Stop expanding when the selected wrong behavior is distinguished and applicable required checks are satisfied. If necessary evidence cannot be obtained within authority or budget, retain the unresolved obligation and use the existing parent/budget/decision path. Do not relabel the rule low-risk merely to finish, and do not ask the owner to make routine test-selection decisions.

### Keep budget pressure from weakening assessment

The [owner asked how to prevent permissive selection under pressure](../sources/working-day-discussion-20261016.md#prevent-budget-driven-underassessment) without another controller. The proposal separates two judgments: which evidence is needed for the current obligation, and whether the assignment can afford to obtain it. Lack of time or budget answers the second question; it cannot answer the first. This extends the existing [protected-budget proposal](desired-working-day-20261016.md#share-an-assignment-budget-and-protect-completion-work), not the Manager hierarchy.

Use the existing review and history to keep the decision explicit:

1. **Identify consequential concerns early.** During the reviewer's model-first reading, record the affected boundary, plausible wrong behavior and intended distinguishing evidence before implementation overruns become a reason to negotiate completion. The Manager includes assessment cost in its plan and reserves capacity. Early assessment is provisional: newly discovered concerns must still be added. Do not hide the budget from the reviewer or require every future case to be known up front.
2. **Keep recognized needs open until resolved.** Once the reviewer identifies necessary evidence, preserve that requirement in the existing finding/assessment history. An unused budget reserve, an expired deadline or a shorter remaining allocation cannot silently turn it into "unnecessary". Genuine reclassification is allowed with a factual reason: the concern is demonstrably outside the actual change and applicability, an existing current check covers the same distinction, or an authorized scope change removes the obligation. Preserve the earlier judgment and the reason/evidence for the change. "Too expensive" and "the code looks simple" do not resolve an identified behavioral concern. A Manager cannot override the reviewer's unresolved finding merely to finish.
3. **Make completion depend on evidence, not a label.** The existing Host/verification path should reject a success/publication attempt with unresolved required findings, missing required assessments or stale/mismatched evidence. A successful command must also have exercised the intended assertion; an unrelated green test cannot discharge the concern. A semantic claim that an obligation no longer applies remains a reviewed judgment, not a deterministic proof. The Host can enforce the recorded bindings and completeness, but cannot detect every false justification or concern omitted from the start.
4. **Keep honest incompletion an expected outcome.** When remaining capacity cannot cover the smallest adequate assessment, retain work and return an explicit incomplete result or use the existing authority/budget path. A controlled stop is not completed delivery, but is correct handling of the limit. Instructions and reports must not treat a green result, throughput or speed as success regardless of omitted evidence. Budget extension does not waive review. An authorized narrower scope must be stated as such and must still meet its own obligations; it cannot masquerade as completion of the original request.

No new risk register or mandatory per-rule form is needed. For a selected concern, the ordinary review entry can remain: concern, required case or equivalent existing check, current evidence, unresolved remainder. The selected-case record may be supplemented as new evidence arrives; it is not an immutable up-front test plan.

For example, a change to repeated-return handling has an identified risk of closing a newer loan. The reviewer requires the L1-return/L2-borrow/L1-return case. If the remaining budget cannot support that case and the required reassessment, the Manager may use an already applicable test, simplify the realization while preserving intent, seek a bounded extension, or stop incomplete. It may not substitute "ordinary return passes" or delete the concern. The report should say which work exists and which result is still unproven, rather than implying that a review-budget shortage makes the behavior safe.

Keep implementation and assessment allocations distinct within the same assignment budget. Only release protected capacity when work actually removes the remaining need or a justified plan still covers it, as already proposed. Reviewers must have a supported incomplete outcome rather than needing to produce approval to close their task. This removes a procedural incentive to manufacture certainty; it does not assert that an LLM has human financial incentives or that instructions guarantee unbiased judgment.

A small extension to the planned insufficient-budget scenario can exercise this boundary without a fifth scenario: establish a necessary distinguishing case before exhausting the free allowance, then attempt continuation with too little remaining capacity. Passing requires either valid current evidence obtained within the remaining allowance or retained incomplete work with the requirement still visible. No automatic downgrade, waived check, reset consumption or publication based solely on a pass label is allowed. Reuse existing negative evidence checks to verify rejection of missing/stale required evidence. This is proposed test coverage, not an observed result.

Residual risk remains when Manager and reviewer both omit the need or accept the same false applicability argument. Without a separate semantic authority, runtime guards cannot prove that judgment correct. The earlier discriminating cases, current evidence and visible revisions reduce this risk; the claim remains bounded rather than promising that pressure can never affect judgment.

### Record required evidence through existing findings

The [owner asked for a concrete record](../sources/working-day-discussion-20261016.md#record-required-evidence-and-its-resolution) that lets the Host detect missing proof without another task or rule system. Reuse the proposed [review and check contracts](../../design/canonical-model/manager-apply-port-contracts.md): `Finding`, `ReviewSubject`, `EvidenceRef`, `CheckReceipt`, accepted deliveries and the existing assignment journal. This is a proposed refinement of finding identity and resolution, not a new API already implemented. The Architect's current uncommitted journal/execution drafts were also consulted; their lifecycle details remain owned there.

A necessary countercheck is an ordinary blocking review finding whose requested outcome is evidence of an existing obligation. It is not a new model rule or a second work item. Implementation, adding a test and running it remain work within the responsible task, delegated through the normal Manager tree where needed. Nonblocking observations must not silently replace an already identified necessary finding.

The finding needs only the information already present or naturally referenced in ordinary review:

| Information | Reuse and meaning |
| --- | --- |
| Stable finding reference | Host-recorded identity, derived from the existing event identity plus item index if sufficient. Repeated submissions reference it rather than creating an unrelated replacement. |
| Responsibility and original basis | Existing assignment/task, issuing reviewer, model Definition references, input binding and affected scope; original review subject/candidate when available. An early model-first concern may precede a candidate, so record that fact rather than inventing candidate evidence. |
| Missing evidence | A short concrete requested outcome: the distinction to demonstrate, with its canonical basis. This explains assessment of a rule; it has no independent business authority. |
| Resolution reference | A later recorded review resolves that finding for an exact current subject, with the relevant evidence references or a justified applicability decision. |

Open/resolved status is a projection of these records, not a separately editable checkbox or another task lifecycle. Use existing journal storage and actor/session provenance. The Manager may propose that the concern is satisfied; the authorized independent reviewer assesses that proposal. The Host accepts a structurally valid resolution from the bound reviewer session, not an Agent-supplied claim to be the reviewer.

For example, an illustrative record can read:

```text
Finding F17, owned by the existing Lending task
Basis: current model's lending/return; observed on candidate C7
Need: return L1, lend the same copy as L2, repeat L1's return;
      demonstrate that L2 remains open and the copy unavailable.

Later review R12, subject C8 with its current input binding:
  resolves F17 using Host check receipt K9 and its L1/L2 case result;
  reviewer confirms that the observed assertions establish the requested distinction.
```

These labels illustrate references, not prescribed schema or new command names. The initial finding stays tied to C7; the later resolution explicitly concerns C8. `K9` must refer to execution on C8's immutable materialization under the allowed check specification, with actual termination/output evidence. A command-level pass alone does not establish that the L1/L2 case ran instead of being skipped. The reviewer cites the actual case output/assertions; if that evidence is unavailable, the requirement remains unproven. Reuse the existing check report format where available rather than creating a universal test language.

### Host acceptance rules for finding resolution

At delivery acceptance and final publication readiness, the Host uses its recorded history and required delivery tree, not only the latest Agent's list of findings:

1. **No disappearance by omission.** Every required finding in the task, relevant child deliveries and predecessor lineage has a current disposition. Missing from a subsequent report does not mean resolved. Task recreation, restart and successor runs cannot hide unresolved requirements. Parent views reference child records rather than copying their text into another tracker.
2. **References must resolve to real records.** Required receipts, immutable evidence payloads, review records and their authorized actor/session provenance exist and are consistent. A prose assertion that tests passed does not create a check receipt. Unknown, failed, cancelled or skipped required execution cannot satisfy a successful-execution requirement.
3. **Evidence must fit its subject.** Check the full input binding, candidate/registry, task/scope and review purpose required at that assessment boundary. Child deliveries retain their exact subjects and composition provenance; the aggregate has its own required integration/final evidence. Do not require every child receipt to share the aggregate hash, and do not use a child pass as aggregate approval. A new candidate or changed basis needs current required assessment, even if an earlier finding has a historical resolution.
4. **Resolution must be explicit.** The ordinary review either establishes the requested evidence or explains why the obligation is genuinely inapplicable to the current authorized scope/basis, with relevant references. Existing adequate checks are evidence of satisfaction, not an exemption. A changed model/scope must cite the actual authorized transition. Neither budget exhaustion nor a generic final pass is a resolution. Initial policy still repeats required checks/reviews; no approval-reuse shortcut is introduced.

One current check can support several findings and one finding can cite several receipts. Reference the same evidence rather than duplicate execution or storage merely to achieve one-to-one bookkeeping. This does not waive check requirements attached to different subjects or review purposes.

The deterministic guard can establish that declared requirements have authorized, current, well-formed dispositions and evidence. It cannot prove that a natural-language finding was complete, that an assertion expresses the intended business meaning, or that a reviewer's inapplicability argument is true. Those remain the existing review's semantic responsibility. Reports should not describe this guard as proof that no necessary check was ever missed.

For the first implementation, refine the existing finding/review payloads and their replay projection only as needed for these references. Do not create a separate verification backlog, per-rule workflow, policy interpreter or UI status store. Focused contract checks can exercise omission of F17, an invented receipt, a C7 receipt offered for C8 and an unrelated final pass; none may discharge F17. A valid current receipt plus the authorized finding-specific review should resolve it, and replay/restart must preserve the same result. These are proposed checks, not executed evidence.

### Reassess open findings in a successor Apply

The [owner asked how to retain open findings without preserving obsolete blockers](../sources/working-day-discussion-20261016.md#reassess-open-findings-after-model-changes). Carry forward the original finding reference and history, then assess its applicability against the successor's actual authorized model/configuration and retained work. The finding is evidence of a concern on its original basis, not a rule that outranks later intent. Old failure and new applicability are separate facts.

At the existing safe successor handoff, derive the predecessor's unresolved findings from the assignment history, including relevant children. Carry them independently of whether their candidate files are retained; discarding a candidate does not prove its underlying obligation disappeared. New tasks reference these findings and their predecessor provenance. If an old responsibility no longer exists, route the applicability question through the existing parent/root rather than dropping it. No semantic migration is done by the journal adapter.

Use four ordinary applicability judgments in the current review, not a new finding workflow:

| Judgment | Current action | Example |
| --- | --- | --- |
| Still applies | Retain the required outcome; obtain current evidence before resolving it. | Changing the loan limit does not remove the requirement that a repeated L1 return leave L2 open. |
| Applies with adjustment | Preserve the original finding, append the current basis and revised required outcome, then verify it. | A loan-limit finding from the five-loan model now needs the corresponding three-loan boundary case. Do not require the obsolete expectation that a fourth loan succeeds. |
| No longer applies | Record a reviewed, source-bound reason why the old requested outcome is not an obligation of the new authorized scope. It stops blocking that scope without becoming a claim that the old candidate was correct. | An authorized withdrawal of the proposed renewal feature removes a pending requirement about allowing timely renewal. Check the remaining model obligations and removal of unintended renewal behavior separately. |
| Unclear | Keep the unresolved scope visible and investigate using existing context, ownership and authority. Ask the owner only for genuinely missing intent or authority. | A change removes a Definition but leaves a conflicting obligation in another applicable rule. Deletion alone does not establish that the requirement disappeared. |

The responsible Manager can propose the mapping; the existing independent reviewer validates its semantic grounds. A short entry is sufficient: original finding, successor input binding, applicability judgment, reason/model delta, current requested evidence where needed, and the review reference. The current subject/candidate is included when assessment reaches that stage. Do not edit the historical finding in place, and do not require the owner to approve routine applicability decisions.

For the same underlying concern, keep the finding identity and append the revised applicability/evidence requirement. If the new obligation is materially different, record an explicitly linked replacement finding in the same review mechanism. The old concern must have a reviewed disposition and the replacement must remain required until resolved; a "replaced" label alone cannot close both. Existing adequate checks can satisfy the current requirement but still need the current evidence and review required by the first-version policy.

For example, R1/M1 has F21: "The borrowing boundary is not verified; a sixth loan may be admitted." R2/M2 changes the limit from five to three. The current assessment links F21 to M2 and asks for three successful loans, a refused fourth, unchanged state on refusal and the applicable existing-loan checks. F21 remains open until those current requirements are supported. The original M1 result is preserved; changing intent does not retroactively pass it.

The Host checks that every inherited required finding has an explicit current disposition, references the actual successor basis, and has authorized review/evidence records appropriate to that disposition. It does not infer obsolescence from a changed model digest, renamed/deleted file, absent task or omitted report entry. An applicability judgment is bound to its reviewed basis; it is not a permanent exemption for later runs. Ordinary full reconciliation of the current model remains necessary to discover new obligations as well as handle inherited concerns.

Unclear or still-open required findings block acceptance of the affected delivery and any aggregate success that requires it. They need not freeze unrelated permitted work. Findings reviewed as no longer applicable do not require executing obsolete behavioral tests merely to achieve a green report. Their historical records remain visible, and the successor still needs its required local/integration/final assessment. No previous approval transfers automatically.

Extend the existing question/model-change scenario with a small representative set: one unchanged finding, one adjusted boundary finding and one genuinely withdrawn obligation. Assert that all original identities remain traceable, required current checks cannot be omitted, the obsolete expectation no longer blocks after valid review, and an unexplained disappearance still blocks. A deleted definition or discarded candidate alone must not close a finding. This is a proposed extension of existing acceptance coverage, not another test platform or an observed outcome.

### Keep current finding context small without losing history

The [owner asked how successive runs avoid context growth](../sources/working-day-discussion-20261016.md#bound-finding-context-across-apply-runs). Keep the full recorded history in the existing journal; supply the current relevant view to each role. The proposed [journal contract](../../design/canonical-model/manager-apply-journal.md#one-authoritative-history) already derives state from committed events and loads large payloads on demand. Extend that existing projection and context construction rather than adding a finding database, summarizing Agent or semantic-search service.

There are three distinct representations:

| Representation | Purpose |
| --- | --- |
| Original events, assessments and evidence | Preserve the original finding, every reviewed applicability/resolution decision and exact supporting records. Historical failures and replaced interpretations remain inspectable. |
| Current derived finding view | One effective entry per finding identity for the selected assignment/run basis: responsible scope, open/currently resolved/no-longer-applicable or requiring reassessment, current requested evidence, latest applicable reason and source references. These are readable projections of existing judgments, not another editable lifecycle. |
| Role context | The current entries relevant to the actual responsibility, dependencies and review purpose, with a bounded account of recent relevant changes and links for deeper inspection. No complete chronology by default. |

Build the view from original typed records and explicit causal/supersession links at a pinned journal revision. Do not choose a disposition merely because it has the newest timestamp. Do not recursively summarize earlier summaries: that can lose the qualification which made a finding important. The concise required outcome and grounds should be the recorded assessment text or a faithful excerpt with a direct link, not a new unreviewed interpretation that becomes authoritative. Any optional generated prose is a presentation aid; the records remain the authority for finding state.

A compact current entry could be:

```text
F21 | Lending | R4 / current input binding | still open
Need: after three open loans, refuse a fourth without changing state.
Reason: admission must enforce the current open-loan limit.
Latest change: adapted from the five-loan case by assessment A18.
Evidence still missing: current boundary-case execution and review.
Sources: model rule; original F21; A18; relevant candidate/check records.
```

R4 does not receive the complete R1/R2/R3 discussion merely because F21 originated there. A Manager receives current obligations and blockers it must act on. A parent receives a compact account of relevant child blockers and delivery references, loading child detail when integration or coordination requires it. The reviewer receives the canonical grounds and current required outcome for its model-first phase; implementation rationale and earlier approval prose do not replace that independent reading.

Do not send resolved or genuinely inapplicable historical findings by default. Retrieve their reason/evidence when a new change affects the grounds of that disposition, when a regression or contradiction appears, or when the role requests the detail. Selection uses recorded scope, model references, dependency/provenance links and actual changes, not recency or wording similarity alone. Unknown ownership or impact routes to the existing parent/root for clarification rather than disappearing. Old resolved status supplies neither current approval nor an exemption: the first-version full current assessments/checks still run, and a disposition whose applicability is unsettled is shown as requiring reassessment.

### Bound the prompt, not the set of obligations

The Host keeps the complete required set in its projection and acceptance guards even when a particular prompt contains only a bounded portion. Necessary open or unclear findings cannot be dropped by a top-N cutoff, token threshold, low relevance score or age. If the current scoped set is too large, use explicit pages or bounded work steps under the same task/responsibility, exposing total counts, remaining entries and continuation references at the pinned revision. Delivery cannot be accepted merely because the Agent handled the first page. A fetched entry counts as seen, not resolved.

All findings needed for a particular implementation or review step must be available before that dependent action; merely offering an unread link is not sufficient context for an affected decision. If adequate context cannot fit, split the work through existing delegation/steps or stop with a visible context/coordination gap. This does not narrow the task's full reconciliation responsibility or create a second scheduler. New relevant records after the pinned view require refreshed context before dependent acceptance, rather than mixing pages from different states.

Use one entry per stable finding, and follow explicit reviewed replacement links rather than repeating every historical wording. Do not merge apparently similar findings solely by text similarity; differences in scope or required evidence may matter. Refer to shared receipts instead of copying their outputs into every entry. A changed-since-last-view digest is useful as an addition to the current view, never as the only basis for a fresh session.

For the first implementation, a query/render over the existing journal projection and on-demand evidence reads are enough. A rebuildable performance cache may be added if needed; it cannot authorize work or become another source of status. Stored history can grow without occupying the prompt, but genuinely growing unresolved work cannot be made safe by compression alone: Managers must resolve it, distribute it or expose the resulting limit.

Extend the existing successor/replay checks with an old unresolved finding among many resolved entries, a current applicability change, explicit replacement links and a page boundary. Verify that the old blocker remains in the Host's required set, current context links to its actual reason, old approval is not promoted, and omitted/unread pages cannot turn incomplete work into an accepted delivery. Rebuilding the view must preserve the same state and source references. These remain proposed bounded checks, not an executed scale or context-quality result.

### Select finding context from recorded responsibility

The [owner asked how the Host selects context without domain reasoning](../sources/working-day-discussion-20261016.md#select-finding-context-without-domain-reasoning). The Host cannot guarantee discovery of an unrecorded semantic dependency. Managers and reviewers identify affected responsibilities and obligations; the Host carries and checks those explicit relationships. This is the same division already used for task scope, model references, findings and accepted deliveries, not a new relevance authority.

For the first version, avoid attempting a minimal finding set for each individual action. Give an active Manager the compact current set for its full task responsibility, plus recorded dependencies and applicable inherited concerns. The saving comes primarily from excluding obsolete chronology and unrelated responsibilities, not from guessing which of that Manager's unresolved findings can be hidden for the next step. Detailed evidence remains on demand. This deliberately favors a modest amount of extra current context over a new semantic routing mechanism.

At each dispatch, use the existing context construction:

1. Pin the current input binding, task/review subject and committed journal revision. Read the task's complete responsibility, parent/predecessor links, declared dependencies and delivery references. A Manager's free-text statement of its next action cannot narrow those obligations.
2. Include all current open, unclear or requiring-reassessment findings assigned to that responsibility, including inherited predecessor findings even when their original candidate was discarded. Include recorded cross-responsibility requirements and applicable parent constraints. For coordination/integration, include child blocker entries and delivery references needed by that parent purpose; no child transcript dump is required.
3. Use exact recorded model/subject/ownership references to conservatively add potentially related entries where needed. Structural matches may widen context; absence of a file overlap or a reference in a short step description cannot remove an existing requirement. A model link makes a relationship inspectable, not a deterministic proof that every statement at the target applies transitively. Do not recursively include the whole graph as if every reachable finding were binding.
4. Render the concise current entries with their grounds and source references. Record which finding/assessment references were actually supplied in the existing step-context payload and which revision they describe. Check dependent dispatch/acceptance against current required records and refresh or reconcile context when relevant bindings or findings change. Do not mix pages from different revisions or claim a new finding was present in an earlier prompt.
5. Derive unresolved obligations for acceptance independently of the prompt contents. Providing, reading or acknowledging an entry never resolves it; current finding-specific evidence and review still do.

The issuing reviewer or accountable Manager records affected scope when raising/triaging a concern through existing task/message/finding records. The independent reviewer validates semantic applicability as already proposed. No extra permission to resolve business intent is created. Host validates that referenced tasks/Definitions/scopes exist and that routing respects authority. It does not infer that a finding is irrelevant because a Manager omitted it from its next request.

If scope is missing, conflicts with the actual change, or points to a removed responsibility, expose the uncertainty to the existing common parent/root. Do not silently discard the finding or classify it as unrelated. The parent resolves ownership from available context and records the relationship; genuine missing business authority follows the normal owner-question path. Dependent work waits while that uncertainty is material. Clearly independent work may continue under its existing mandate.

For example, the Membership reviewer identifies a concern about returning loans while a member is suspended. The finding references the membership eligibility rule and the affected Lending return behavior. The current model already connects those definitions. The Host can route the recorded concern to the ordinary Lending responsibility and make it visible to their coordinating parent without understanding the word "suspended". If the finding is recorded only under Membership and the relevant relationship is neither captured nor structurally surfaced, the Host cannot promise to discover that semantic omission. Full model/context review and integration review remain necessary; their role is not replaced by reference matching.

If the complete compact task set still exceeds the context limit, use the existing explicit paging/planning/delegation mechanisms. Read-only preparation may inspect pages; a step that changes or accepts an affected result must have its relevant obligations available, not just an aggregate count. The Manager can propose a bounded work step or child task with explicit references, but this cannot erase the parent responsibility or suppress unresolved blockers. If an adequately scoped action cannot be established, report the context gap instead of silently dropping entries. Fine-grained automatic context minimization can wait for demonstrated need.

Keep routing evidence in existing context and journal records: task identity, current revision, included finding references and explicit dependency/ownership references. No second graph editor, required per-step risk form, semantic ranking threshold or separate context approver is proposed. A deterministic membership check can establish completeness relative to the recorded required set. It cannot prove completeness of the original semantic mapping or that an Agent actually understood its supplied context.

Bounded contract cases can verify that a step naming only one file still receives its task's other open findings, a recorded cross-Package finding reaches the receiving responsibility, an unresolved owner mapping is surfaced rather than dropped, and an old context revision cannot silently authorize work against newly changed required evidence. These are proposed extensions to existing context/replay checks, not claims of implemented routing.

### Detect incorrect or incomplete finding ownership locally

The [owner asked how to detect misrouting and additional affected Managers](../sources/working-day-discussion-20261016.md#detect-incorrect-or-incomplete-finding-ownership) without repeated whole-project inspection. Check ownership while the existing receiving Manager interprets the task and while the existing reviewer/parent assesses the result. The [Apply concept](../../design/canonical-model/manager-apply-concept.md) already supplies direct incoming/outgoing model references, registry ownership, targeted additional reads and sibling coordination through the nearest common parent. These are sufficient starting points; another routing reviewer is not proposed.

Distinguish the source of a rule, the owner of an artifact and the responsibilities affected by a behavior. A Membership rule can constrain Lending code; the rule's Package does not automatically own the repair. Finding scope and requested outcome must be considered alongside actual artifact ownership, not inferred from where the rule or error message was found.

During normal task interpretation, the receiving Manager asks three short questions:

1. Can my current responsibility and mandate produce the requested outcome, using the owned artifacts and contracts I actually have?
2. Does the case depend on another responsibility's output, state transition or contract? Check the cited Definitions, their direct relevant neighbors and concrete artifact/call evidence encountered in the work.
3. Would a local correction and local passing check establish the requested behavior, or could a known consumer/producer boundary still violate it?

Use only the actual context and targeted supporting reads needed to answer these questions. If a concrete dependency points further away, follow that dependency with an explicit reason; direct neighbors are a starting scope, not a claim that impact can never extend further. No arbitrary one-hop cutoff establishes semantic safety, and no transitive graph scan is required for every finding.

| Signal | Response within existing responsibilities |
| --- | --- |
| The cited artifact belongs to another Manager or the proposed write crosses an ownership boundary. | Host exposes/enforces the recorded ownership. The receiving Manager requests coordination instead of editing foreign files. This is a structural mismatch, not proof of which semantic fix is correct. |
| The required result depends on a neighbor's contract or behavior. | Record the concrete dependency and request the necessary contribution through the nearest common parent. Merely referencing a neighbor does not automatically require work or approval from it. |
| A local test passes but the composed behavior fails. | Keep the finding open at the coordinating scope, inspect the failing boundary and involve the specific responsibility indicated by the evidence. Do not require every Manager to reread the project. |
| Model links, registry links or responsibility mapping are insufficient or contradictory. | Expose the gap and perform bounded investigation through the parent. Missing links do not prove absence of impact. Any actual change of canonical intent follows the existing authority path. |

For example, a finding about returns by suspended members may initially reach Membership because its eligibility rule is cited. Membership checks that the status it supplies is correct and points to the Lending return operation where status is consumed. If Lending wrongly rejects the return, Lending owns that repair; Membership need not receive an unnecessary implementation task. If the return succeeds but Catalog still reports the copy unavailable, the concrete availability contract and registry identify a possible additional contribution. Whether Catalog needs changing depends on the actual realization: a shared observation alone is not permission to assume a second faulty component.

Keep one finding reference. The parent records corrected accountability and any needed child contributions using the existing task/dependency/delivery records. A consumer may request an outcome but receives no write authority over the producer. A receiving Manager's "not mine" response cannot close or orphan the finding: unresolved routing remains accountable to the existing parent until a recorded valid assignment exists. Changed task identity or a rejected handoff does not remove it from Host acceptance checks. Recurring disagreement uses the existing bounded stagnation/coordination path rather than endless forwarding.

When several responsibilities must contribute, retain the finding at their coordinating scope with references to their ordinary tasks and local evidence. Each existing reviewer assesses its own subject; the existing integration review assesses the composed requested outcome. For the library case, that can be the actual suspend/return/availability sequence. A local pass alone cannot discharge the cross-responsibility concern. There is no requirement for every neighboring Manager to co-sign, and no separate duplicate finding queue per participant.

The first proposal can be corrected as new evidence arrives. Record the reason and affected references in the current finding assessment; do not require a permanent dependency map to be complete before any useful work begins. Host validation covers valid scope references, ownership, recorded dependencies and absence of orphaned unresolved findings. Semantic completeness remains fallible: a missing relation which neither local reasoning nor integration evidence exposes may still be missed. The proposal reduces repeated project-wide reading; it does not prove perfect impact discovery.

Extend the existing routing checks with a deliberately misassigned finding and a case needing two contributions: the first receiver must expose the mismatch, the same finding remains traceable through parent coordination, and acceptance requires the resulting current local/integration evidence. Include a valid case where a cited neighbor needs no change, to avoid turning every relationship into compulsory delegation. These are proposed checks, not executed evidence.

### Keep parent coordination bounded and prevent forwarding loops

The [owner asked how to avoid a common-parent bottleneck](../sources/working-day-discussion-20261016.md#bound-parent-coordination-without-new-structures). Use the nearest existing common parent for cross-responsibility decisions, with normal child delegation through the hierarchy. Do not escalate every case to root or introduce a coordinator role, finding board or peer-management authority. Parent accountability for the combined outcome remains; routine transport and progress recording need not invoke the parent Agent again.

Use five rules within the existing task/finding mechanism:

1. **Send one decision-ready request.** The requesting Manager supplies the finding reference, observed mismatch, affected contract/ownership references, outcome needed from the neighbor and what currently blocks progress. It proposes a responsibility/order where supported. The parent reads this bounded issue and fetches further evidence only when necessary; it does not reproduce every child's investigation.
2. **Decide contributions together.** In one coordination step, the parent records which existing task owns each required contribution, the dependencies/order, expected delivery evidence and bounded allocations. One existing Manager may remain the primary requester who tracks its needed result; this gives it no authority to assign siblings or edit their artifacts. The finding can remain accountable at the parent until its cross-area outcome is verified. These facts belong in existing tasks and dependency/delivery records, not a separate coordination plan format.
3. **Let the Host carry out the recorded routing.** Delivery availability, ordinary status updates and already authorized dependency releases are recorded/routed mechanically under that decision, within current authority and budget guards. They do not require a new parent inference for every message. New scope, conflicting contracts, changed ownership, failed dependencies or insufficient authority return for a real parent decision. This does not bypass required local/integration review or the initial one-writer scheduling policy.
4. **Reject forwarding without progress.** A Manager may contest a concrete assignment with evidence of the ownership mismatch and the missing prerequisite; "not mine" is not completion or permission to redirect the sibling. Once the parent has recorded a decision, reversing it requires a new concrete reason or correcting an identified error. If both sides wait on each other or the same routing recurs without new evidence, the parent performs the existing bounded progress/coordination assessment instead of sending the unchanged request around again. It can commission a focused diagnosis within existing read/write mandates, revise the order or stop incomplete. No fresh budget or review-round allowance arises from reassignment.
5. **Integrate the result at the existing boundary.** Children perform their normal work and local review. Parent composition and independent integration review assess the combined outcome using actual delivery evidence. Compatible findings on the same subject can share a review/check; do not create another whole review merely for each finding. Batch decision-ready issues where useful, but do not wait for an arbitrary batch size when a dependency already blocks work.

For the library return example, the parent can record: Lending investigates/corrects the return operation; Catalog contributes only if its actual availability contract needs a change; Membership's demonstrated correct status behavior needs no new implementation. If Catalog work is necessary, record its contract outcome and the order in which Lending consumes the reviewed delivery. The Host advances those existing dependencies as evidence arrives. The parent returns for a genuine conflict or the required integration, rather than repeatedly asking whether either Manager is ready. The final suspend/return/availability trace still has to pass on the composed result.

The first version can observe coordination cost from its existing journal: parent invocations, waiting on parent decisions, repeated reassignment of the same finding, and whether each intervention produced new evidence or a changed actionable decision. These are diagnostic facts, not a new score, automated blame system or incentive to skip coordination. A repeated structural cycle in explicit dependencies can be surfaced mechanically; deciding the right technical order remains Manager work. The existing execution contract's bounded correction/progress rules remain authoritative; this proposal adds no conflicting fixed retry count.

Some parent work is unavoidable: many genuinely coupled changes create real coordination demand. If that persists, first simplify the concrete request, contract or task boundaries within current authority. Broader changes to modeled responsibility are explicit design decisions, not automatic runtime reorganization. Lack of progress is reported honestly; a technical dependency cycle must not be disguised as an owner business decision or external waiting.

Extend the existing multi-contribution routing case with ordinary delivery notifications and a contested handoff: recorded dependency progress should not require a new parent semantic decision for each update, while repeated unsupported forwarding must remain unresolved and trigger bounded coordination rather than silent closure. Required integration evidence, ownership and budgets stay enforced. These are proposed observations/checks, not demonstrated throughput improvements.

### Release a dependency from an accepted delivery

The [owner asked how the Host recognizes a fulfilled delivery dependency](../sources/working-day-discussion-20261016.md#recognize-a-delivery-that-enables-continuation) without deciding domain meaning. Use the existing Host-issued `DeliveryID` and its exact subject, scope and evidence from the [port contract](../../design/canonical-model/manager-apply-port-contracts.md). An Agent's completion message, a file appearing or a successful command is not an accepted delivery. This proposal makes the dependency's references explicit; it does not add a general predicate language or another acceptance role.

The parent records the agreement in existing tasks and dependency records: the consumer task/step, the designated producer task, the versioned requested outcome/constraints and their model basis, and the required delivery/review boundary. The outcome and constraints have one owner in that recorded task/coordination decision; the dependency points there instead of copying a separate specification. When the producer delivers, the dependency binds the actual accepted delivery reference, not an alias such as "latest Catalog output".

The producer's normal independent review assesses whether the candidate meets that recorded outcome and its full applicable responsibility. Required checks, Definition assessments and finding-specific resolutions support the judgment. The Host then issues an accepted delivery only when its ordinary structural/evidence guards pass. If the existing verdict does not explicitly cover the requested contribution, a generic pass cannot supply that missing semantic assessment. A clarification or changed outcome returns to the existing coordination path.

To make the dependent step ready, the Host can check these facts mechanically:

1. The dependency still refers to the current authorized task/coordination revision and the designated producer, with no unresolved prerequisite or blocking question for that step.
2. A Host-issued accepted delivery exists for that producer and recorded scope, backed by the required authorized reviews, real check receipts and dispositions of required findings. Its input/contract bindings match the dependency's expected basis. Partial, failed or unknown execution does not qualify.
3. The actual immutable candidate and registry referenced by that delivery are available and consistent. The recorded consumer starting candidate can compose/use them under the existing ownership and base rules. Conflicts are returned for coordination; the Host does not choose a semantic merge.
4. The consumer's actual materialized inputs/context reference that exact delivery and the resulting composition, including relevant constraints/findings. Merely delivering a notification or the delivery ID in prose is insufficient if the consumer still works against older bytes.
5. Normal dispatch guards still hold: current run/controller authority, budget, execution limits, exclusive writing and no unresolved external effect. "Dependency satisfied" makes work eligible; it does not bypass scheduling or automatically start another writer.

Persist the accepted delivery and dependency/consumer binding through the existing journal transitions before dependent dispatch, and make repeated notifications idempotent through existing IDs/replay. Recovery derives the same readiness from those records and reconciles actual workspace/session state before effects. It does not blindly repeat a dispatch because the producer notification arrived twice.

An illustrative library sequence is:

```text
Parent decision P6: Lending task L waits for the reviewed contribution of
Catalog task C under the recorded availability contract and model basis.

Catalog proposes candidate C8; its required review/checks cover that outcome.
Host accepts delivery D12 for task C, referring to C8 and its evidence.
Host binds L's dependency to D12 and prepares the compatible consumer inputs.
If all ordinary dispatch guards hold, L may continue with those exact inputs.

Later integration still checks return -> loan closed -> copy available.
```

These labels are illustrative references, not new API names. Catalog contributes only if actual responsibility requires it; the example does not mandate a particular implementation of availability. The producer boundary should describe a useful contribution that can be independently assessed before the consumer uses it. If its supposed acceptance already depends on the unfinished consumer result, the parent must resolve that dependency cycle or revise the legitimate work order. It must not mark the producer done merely to unblock the cycle.

Readiness is not proof that the complete consumer flow works. The consumer can still discover a contract mismatch, and the existing parent/integration review then handles it with current findings. An accepted component delivery does not automatically resolve the overall cross-area finding, pass aggregate verification or authorize publication. The Host checks the existence and consistency of the semantic judgment; it does not independently prove that judgment correct.

The chosen delivery remains immutable. A new producer candidate, revised coordination decision or successor basis must not silently replace it in an already prepared consumer context. If the selected input/contract changes, explicitly rebind/reconcile the consumer and obtain the current assessments required by the initial policy. Earlier deliveries and consumer results remain historical, not automatically current approval.

Bounded dependency cases should reject a producer's self-reported completion, a receipt for the wrong subject, a delivery from the wrong task/contract revision and a consumer workspace still containing older input. A valid accepted delivery with compatible actual inputs should enable eligibility; duplicate notification/replay should not create duplicate execution. Keep these checks within the existing delivery, context, composition and dispatch contracts rather than building another workflow engine. These are proposed checks, not executed results.

### Correct an accepted delivery without discarding downstream work

The [owner asked about changes or defects discovered after downstream work began](../sources/working-day-discussion-20261016.md#correct-an-accepted-delivery-with-active-consumers). Preserve the accepted delivery's immutable content, original acceptance and consumer provenance. Record a later finding, a selected replacement and their effect on current eligibility in the existing history. The old acceptance is a historical fact, not a permanent guarantee. This refines the existing correction/dependency contracts; it is not a new incremental correctness engine.

Distinguish two situations first:

- **A newer candidate or delivery exists.** Its existence alone neither invalidates a still-applicable selected delivery nor silently upgrades its consumers. A recorded decision selects a replacement when required by the current outcome/basis.
- **A recorded blocking defect or incompatible changed requirement concerns the selected delivery.** Prevent it and dependent results from serving as current completion/publication evidence until the concern is assessed and resolved. Use existing finding/assessment records to derive that eligibility; do not overwrite the old delivery or erase its original review. If a suspected defect is disproved, record that reviewed resolution and reassess the current guards rather than pretending no concern was raised.

Handle correction through the normal Manager tree:

1. **Identify potentially affected consumers.** Follow actual consumed `DeliveryID` references and composition provenance, including downstream deliveries and the aggregate. These links establish potential impact, not that every consumer contains a defect. The affected Managers/reviewers judge the scope using the actual finding and contracts. Missing or uncertain dependency evidence broadens assessment through the parent; it cannot justify an automatic "unaffected" result.
2. **Stop unsafe progression, preserve candidates.** Do not dispatch new work that requires the known-invalid premise or accept/publish affected results. Reconcile an already active step through existing stop/terminal-state rules before changing its inputs or starting another writer. Retain any output with its actual old input binding as a candidate for inspection, not a current accepted result. Explicitly independent work may continue within authority and limits; no silent mid-session replacement is allowed.
3. **Repair at the owner and obtain a new delivery.** The responsible producer corrects the actual defect and undergoes required review/checks. An independently accepted replacement receives its own identity and links to the earlier delivery/finding. Reuse the existing shared budget and bounded correction rules; changing task or delivery IDs cannot reset them.
4. **Reconcile consumer work against the replacement.** Keep, Adapt, Replace or Unresolved applies to the retained work. Explicitly compose/rebind the new producer delivery with the consumer's actual candidate and registry. The consumer may retain almost all code or propose justified no-change work, but must obtain required current assessment for changed inputs/assumptions. Producer acceptance alone does not restore downstream approval. If a replacement conflicts with local work, the existing parent coordinates it rather than the Host choosing a semantic merge.
5. **Re-establish the required aggregate evidence.** Local, integration and final verification use their correct current subjects and composition references before publication. No blanket restart from an empty repository and no blanket reuse of old approvals are implied. The initial full reconciliation/required-check policy remains in force; this proposal saves implementation work, not mandatory assessment through speculative semantic invalidation.

For example, Lending has already built candidate L7 using Catalog delivery D12. A finding shows that D12 can report a returned copy as unavailable. Membership registration work remains available; L7 is also retained rather than deleted. Catalog repairs the identified behavior and delivers D13 after current review. Lending explicitly incorporates D13, keeps its still-suitable return implementation and adjusts only what the changed contract/behavior requires. The suspend/return/availability trace and required current reviews assess the new composition. L7's prior pass cannot approve that changed composition, and D13's pass alone cannot close the whole cross-area finding.

A correction within the same fixed model/configuration and valid acquired source can use ordinary bounded rework or replacement tasks in the current Apply. A changed intent/configuration or invalidated acquisition basis requires the existing successor/new-acquisition route, retaining assignment consumption and candidates. Do not create a new Apply solely to reset attempts, and do not mutate a fixed run's input basis to avoid a successor.

If publication has already happened, retain that actual historical Applied result and add the discovered defect. A subsequent authorized repair acquires the real current source. Do not pretend that old publication never happened, perform an implicit rollback or claim a deployed system was repaired by changing local artifacts. Uncertain/partial publication still follows the existing reconciliation rules.

Use the existing defect/recovery evidence cases to observe D12, a dependent L7, a later blocking finding and D13: verify that affected acceptance cannot use the old premise, L7 bytes remain available, the replacement is explicitly selected and materialized, and current downstream/integration evidence is required. Also verify that an unused newer delivery does not silently replace D12 merely because it is newer. These are proposed bounded checks, not an implemented guarantee of complete impact discovery.

### Gate active work by recorded delivery dependencies

The [owner asked which ongoing work must pause after a delivery defect](../sources/working-day-discussion-20261016.md#pause-dependent-work-without-domain-reasoning). The Host can conservatively determine exposure to a recorded delivery, not prove the semantic impact of its defect. For the first version, gate at the task/step and concrete delivery boundary rather than infer whether a particular function happens to avoid the faulty behavior. Use existing findings, input/context bindings, composition provenance and dispatch guards; no new scheduling or invalidation service is needed.

Start from a recorded blocking concern tied to the actual delivery and authorized finding/assessment provenance. An arbitrary log line or an Agent's unsupported claim to change authority is not a control transition. A suspected defect may justify a visible precautionary block pending assessment; it is not silently presented as a confirmed semantic failure.

Derive potentially exposed work from recorded consumption: direct consumers of the delivery, tasks whose candidate composition contains it, and downstream consumers of those results. Include relevant supplied artifact context and observed additional reads rather than trusting only a manually declared dependency list. Broad workspaces or missing read/provenance boundaries can force a broader exposure set. Do not claim precise independence when the actual environment does not establish it. This follows concrete data/provenance links, not a transitive inference that every related model statement has the same meaning.

| Situation | Host action |
| --- | --- |
| An ordinary pending/active step consumes the affected delivery or a derived result. | Block new dependent dispatch and acceptance/publication on that premise. For an active step, request a bounded stop and reconcile it through the existing execution contract; preserve its candidate and actual old bindings. |
| The step has no recorded exposure, its required inputs are established, and no unresolved scope or shared-state uncertainty includes it. | It may remain eligible under the normal mandate, context, budget and writer guards. This is independence within the recorded execution basis, not a proof against every hidden semantic dependency. |
| Exposure, ownership, effect state or the defect's scope is uncertain. | Block the potentially affected action and route clarification through the existing parent/review. Do not turn missing information into an "unaffected" decision. |
| The step is explicitly commissioned to diagnose or repair the faulty delivery. | Permit that bounded correction work under the appropriate existing task/role grant. It may inspect/use the defective candidate as evidence, but cannot treat its old acceptance as proof that the repaired result or its consumers are correct. |

The last distinction prevents a deadlock where blocking all use of D12 also prevents its owner from fixing D12. It uses the recorded task purpose and authority, not an unrestricted "ignore dependency checks" flag. Consumer continuation requires a compatible accepted replacement or a current reviewed applicability/resolution decision that addresses the concern; a Manager's claim "my next line does not use that function" is insufficient. Required current assessment remains mandatory.

Commit the blocking finding/current eligibility before further dependent admission. Recheck eligibility at dispatch and acceptance/publication boundaries so that a completion racing with the finding cannot silently preserve a stale permission. An already executing external process cannot be made retroactively idle. A stop acknowledgement alone does not prove termination: retain its execution/resource reservations until observed termination or effective loss of authority is established. Do not start a replacement writer or replace its inputs during that uncertainty. A late result is retained as old-basis evidence, not automatically accepted for the current result.

The first version allows one writer. Consequently, an otherwise independent writer may still have to wait for the active writer to stop and be reconciled. Distinguish a dependency block from resource scheduling; "unaffected" does not mean "runs concurrently now". These are eligibility decisions within existing run/task states, not new global Paused states or an external-wait label for an internal dependency problem.

For the library example, Lending's L7 consumes faulty Catalog D12 and cannot proceed as if that prerequisite were sound. An aggregate built from L7 also cannot be accepted. Catalog receives a normal bounded repair task. A Membership task using only its established independent inputs can remain eligible, provided the actual workspace/context isolation and global writer state support that claim. If its workspace also contains D12 and usage cannot be separated reliably, investigate or conservatively block it rather than assert independence from the Package name. After accepted D13 is explicitly composed and bound, Lending can reuse L7's useful work and obtain current review.

Limit selective exceptions in the first version. Coarse exposure-based blocking is easier to inspect than a mechanism trying to prove that a consumer used only harmless parts of a defective delivery. If conservative blocking becomes a measured cost, improve explicit task/input boundaries first. Unknown consequences still broaden assessment and the selected full reconciliation policy remains unchanged.

Extend existing dispatch/recovery checks with direct and downstream exposure, a demonstrably separate task, a commissioned producer repair, uncertain exposure, and a finding arriving during a live step. Verify no new dependent dispatch or stale acceptance, preservation of candidate work, bounded reconciliation, and no second writer while the first remains uncertain. These are proposed bounded checks, not an implemented guarantee of semantic impact detection.

### Record supplied inputs rather than every file read

The [owner asked for reliable provenance without file-operation surveillance](../sources/working-day-discussion-20261016.md#record-work-inputs-without-file-operation-surveillance). Record the bounded information made available to execution and its actual additions, not an unverifiable claim about which facts the Agent mentally relied on. This produces a conservative exposure set: an available input may never have influenced the result, but it cannot be excluded merely because the Agent says it did not use it. Precise causal reliance is not established.

Use the existing `InputBinding`, `StepRequest`, `WorkspaceSpec`, session/history references, tool replies and immutable candidate composition. A compact input record for a step references:

- The fixed model/configuration/mandate basis and actual prompt/context/messages.
- The exact starting candidate and registry, its contributing delivery references and the actual readable artifact scope made available to the session.
- The selected tools/profile and relevant declared environment/dependency inputs already bound by execution contracts.
- Any continued session or supplied prior history, rather than pretending a resumed Agent starts with only the latest prompt.
- Later information replies with their actual content/source revision and the session/step to which they were supplied.

These are references in existing records, not another user-maintained manifest or dependency specification. The Host creates them from what it prepares and actually sends. One snapshot can cover thousands of files; individual opens within that fixed readable snapshot do not need separate causal tracking. Relevant additional information outside it goes through the existing Host-mediated reads/messages/tools, with the actual reply durably recorded before it becomes an execution input. A path, URL or label such as "latest Catalog" is not sufficient provenance without the observed version/content.

For example:

```text
Step S18 / Lending
Fixed inputs: I4
Workspace base: C10, composed from deliveries D12 and D5
Context: K7; continued session/history: H3
Additional input: R31, exact returned source content and basis recorded
Result: candidate C11, linked to S18 and its recorded additions
```

If the whole repository snapshot was readable, its relevant represented deliveries are potential inputs, even when only one Package was assigned. Do not infer a narrower dependency set from the task title. Smaller actually enforced input/workspace boundaries can reduce conservative pauses later; they need not be introduced as a fine-grained tracing platform in the first version. Host-mediated requests provide targeted expansion when needed.

A continued session retains provenance from previous supplied inputs and observed additions. Keep a reference to that lineage rather than repeatedly placing its full text in the next prompt. Provider context compaction does not prove that an old dependency stopped influencing the session. Fresh reviewers still start with independently prepared context as required by the existing protocol; citing a fresh role name cannot erase reused session history.

Each result links to its producing step and exact input references; each composition links to its selected deliveries. Reverse lookup of consumers can be derived from those existing records, or cached rebuildably if needed. This is a small provenance relation that follows actual execution, not a manually curated domain dependency graph. Multiple uses of the same immutable input reference need not duplicate its bytes. No separate Agent report of "files I used" is authoritative.

### State the limits of the execution boundary

Reliable exclusion requires a real observable/enforced boundary. If a shell or provider can silently read arbitrary local files, browse external sources or insert unobserved context, an otherwise complete Host manifest cannot prove that those inputs were absent. Ordinary scoped instructions and snapshot hashes are not enforcement. Use the executor's supported controls and recorded tool/transport results where available; otherwise report broader possible exposure or unknown input provenance and do not grant selective-independence claims from missing records.

The first version can prefer frozen workspace inputs plus recorded explicit additions, with supported tool access and bounded execution. It need not monitor every file read. Conversely, it cannot promise unrestricted ambient access, no observation of that access and exact dependency isolation at the same time. Actual changed-artifact capture, ownership checks and protected-input verification at freeze remain required; avoiding read surveillance does not remove those write/result checks.

Additional context does not silently change the run's canonical model/configuration. A recorded clarification can inform same-basis work; changed intent or execution authority follows the existing authorized model/amendment and successor rules. Exact bytes establish what was supplied, not whether those bytes are true, authoritative or a correct interpretation.

For defect handling, a supplied D12, an explicitly read D12-derived artifact or continued history containing it remains potential exposure. If an input reply is missing or its basis cannot be established, record an evidence gap rather than assume independence. A known broad snapshot can support conservative assessment without claiming per-file usage; unknown uncontrolled exposure cannot justify a precise unaffected verdict.

Bounded contract checks can verify snapshot-to-step-to-result links, recording of an additional reply before delivery, preserved exposure across session continuation, and conservative handling of a broad or unobserved input scope. They should not assert deterministic reproduction of Agent reasoning or complete semantic influence. This is proposed execution bookkeeping using existing records, not an implemented tracing system.

### Shape useful input packages without unnecessary artifact exposure

The [owner asked how to cut input packages](../sources/working-day-discussion-20261016.md#shape-manager-input-packages) so that Managers can reason well without depending on every other delivery. Start from the [existing context recipe](../../design/canonical-model/manager-apply-concept.md): complete own Definitions, direct incoming/outgoing neighbors, declared shared context, current obligations and a compact routing directory. Refine the actual artifact exposure separately; reducing prompt text alone does not narrow a readable workspace.

Distinguish the Host's complete acquired source/aggregate candidate from the view supplied to one role. The Host still needs the whole fixed source and registry for accounting, composition and publication guards. A role can receive a reproducible subset identified by that source snapshot, selected content and actual access boundary. Merely naming the aggregate base in Host metadata does not mean the role read all its bytes. Conversely, if the complete base is mounted/readable to the session, it remains conservative exposure even when the prompt describes only one Package. This clarifies input provenance rather than creating another canonical model or task scope.

Use four simple parts:

| Part | Initial content | Expansion rule |
| --- | --- | --- |
| Own responsibility | Task/mandate, full own model Definitions and necessary type contracts, applicable shared rules, complete compact current finding set, owned artifacts and relevant local tests/docs. | Do not strip applicable obligations to meet a token target. A large responsibility uses the existing bounded steps/delegation mechanism. |
| Neighbor contracts | Direct incoming/outgoing referenced Definitions and needed type contracts, with the reason/direction of each link; agreed contribution/interface references. | Follow further references when a concrete question requires them. One-hop inclusion is a useful starting point, not a semantic completeness guarantee. |
| Necessary realization inputs | Exact accepted deliveries or immutable artifact portions needed to call, compile, test or understand the neighbor interaction; required module/build/lock/shared-file inputs under their actual owners. | Add further source or evidence by targeted recorded reads. A path view carries its originating delivery reference; it is not a separately accepted delivery or a proof that defects elsewhere in that delivery are irrelevant. |
| Orientation | Package/owner identities, purposes and references sufficient to locate other areas, plus relevant task/dependency status. | Load another area's actual artifacts, reports and detailed history only for an identified work or review need. A directory entry does not imply that all listed delivery contents were supplied. |

For Lending, this normally means its loan/borrow/return/policy Definitions and owned realization, the Membership eligibility/member contracts, Catalog copy/availability contracts and relevant shared input rules. Include direct incoming reminder references as required by the current context policy; do not hide them because reminders are a child area. Give Lending the concrete interfaces/dependencies required by its actual implementation, but not automatically Membership's full registration implementation, all Catalog source, reminder transcripts or every historical candidate.

Canonical neighboring rules and delivered implementation are different input classes. Learning from the model that suspended members may return loans does not by itself consume a specific Membership implementation delivery. Once Lending is given that implementation, its interface artifact or a delivery-derived report, retain the concrete provenance. A shortened report is not a way to wash away a delivery dependency.

Record a short inclusion reason alongside the existing input references: own responsibility, explicit model neighbor, shared obligation, required build/input, current dependency/finding or targeted additional read. These are explanations of the existing context construction, not a new manually maintained dependency catalogue or ranking system. The Manager requests a needed source by identity with its concrete question; permitted read expansion uses existing Host tools and does not need a new parent decision each time. Reading it grants no write authority or business-policy decision.

Apply the same distinction to review and integration. A local reviewer gets the actual candidate/diff and sufficient neighboring contracts/evidence in a fresh context. Parent coordination starts from relevant agreements, blockers and accepted delivery references, then loads exact details for composition and verification. Required aggregate checks operate on the full composed result; their wider input provenance remains visible. Do not replace a real project build with stubs or omit dependencies merely to make the input set look small.

For the first version, do not require every Package to become a separately buildable module or publish a new contract-only delivery format. Reuse actual interfaces and existing artifact ownership. If the current code layout requires broader source for a meaningful build/test, supply that broader fixed input and report the broader potential exposure. Narrowing can later follow real repeated coupling costs; input bookkeeping must not force premature architecture changes.

The safe first optimization is removing unrelated history and avoiding unsolicited foreign artifacts. Smaller actual workspace views are useful where supported and inspectable, but remain an execution capability to establish, not a claim implied by scoped prompts. The Host still preserves complete model reconciliation and aggregate verification. Existing delivery-level defect gating stays conservative: a narrower artifact view improves what was supplied, but does not automatically prove that a known defect in its source delivery cannot matter.

Test the package construction with a Lending example: required eligibility/return and incoming reminder context remains present, a task-specific Catalog input is exactly bound, unrelated foreign artifact bodies are absent when access really is restricted, and a targeted read expands the recorded basis. Include a broad-build case whose exposure is honestly broader. These are proposed preparation/context checks, not proof that minimal context or isolated builds have already been implemented.

### Detect consequential context gaps during ordinary work

The [owner asked how a Manager notices missing context](../sources/working-day-discussion-20261016.md#notice-missing-decision-context) before making a plausible wrong decision. Self-confidence cannot establish context sufficiency. A Manager can recognize an unsupported premise or contradictory evidence, but may fail to notice a missing concept entirely. Use a short premise check in existing task interpretation and review, backed by structural context checks and the earlier discriminating cases; do not promise complete detection or add another review role.

Before a consequential decision about business behavior, an ownership boundary or another area's contract, ask:

1. What supports this decision: an actual canonical statement, an observed current artifact/contract, or explicit implementation freedom in the mandate?
2. What must be true outside my own changes for the result to be correct, and is that premise established on the selected basis?
3. If a plausible alternative were true, would my action differ? What is the smallest source read or observation that distinguishes the alternatives?

The third question bounds investigation: retrieve information that could change the decision rather than asking for the whole project "to be safe". This is not a requirement to justify every local variable or algorithm choice with a model citation. Ordinary local choices within the mandate can proceed. Consequential assumptions about business meaning, authority or foreign behavior cannot become facts merely because they seem conventional.

| Observable signal | Appropriate next action |
| --- | --- |
| A required own rule, declared shared rule or direct referenced contract should be in the prepared context but is missing/unresolvable. | Host context construction should detect the structural omission against its bound inclusion recipe. Repair/refuse the deficient preparation rather than asking the Agent to guess. Presence checks cannot prove that the model itself has every necessary relationship. |
| The Manager's intended action relies on an unobserved foreign behavior, side effect, error case or ownership claim. | Name that premise and request the exact relevant contract, source or current evidence through the existing recorded read tools. |
| Model, implementation, test output or supplied reports disagree. | Preserve the discrepancy and investigate its concrete basis. Do not resolve it by selecting the most convenient statement or by automatically treating current code as desired intent. |
| The alternative interpretations would change authorized business behavior and available sources do not decide between them. | Use the existing parent/owner decision route for genuinely missing authority; do not invent a new rule in code or tests. |

For a library example, suppose a Manager is changing return handling in an existing realization and is about to add a second stored availability flag. The model already requires availability to reflect open loans. The missing fact may be how the selected Catalog implementation currently realizes that contract: does its query derive availability from loan state, or does a maintained representation need updating? Read that specific contract/source and exercise the return/availability observation before assuming a second state update is needed. This hypothetical implementation question does not reopen M1's settled business rule or imply that the modeled library already has a particular storage design.

Conversely, whether suspended members may return loans is already explicitly decided by M1. The Manager should use the supplied eligibility/return rules, not escalate that question again. If those required sources were omitted from its package, that is a context-preparation defect. If it read them but misunderstood them, the suspension/return countercase and existing independent review address a reasoning error rather than an absence of context.

Resolve a recognized material gap before the dependent write, commitment or acceptance. Bounded reads and diagnostic checks may continue under existing authority; unrelated work remains possible. A technical information request is not automatically a question for the owner. New context is recorded against the actual fixed source basis; discovering changed intent still follows the authorized successor route.

Only material unresolved premises need a short entry in the existing task/finding record: "To do X I need to establish Y; current evidence is Z; request/read A will distinguish the alternatives." Once resolved, cite the actual source or observation. Do not create a separate assumption register, confidence score or per-step checklist ceremony. If the gap cannot be resolved within current authority/budget, retain it as an incomplete obligation rather than filling it with a plausible default.

The existing reviewer starts from the canonical obligation and asks which foreign premise the actual candidate relies on, then checks its evidence and a discriminating case where warranted. Integration examines the resulting boundary behavior. This gives a second opportunity to expose an unstated assumption without requiring another full project read or additional review layer. Agreement still does not prove the absence of shared blind spots.

Small preparation/workflow cases can omit a declared required context item, leave a concrete implementation premise unavailable, or supply contradictory bound evidence. The expected observations differ: structural preparation failure, targeted retrieval before the dependent action, or an explicit unresolved discrepancy. Include a normal in-mandate choice that proceeds without needless retrieval to avoid rewarding perpetual hesitation. These are proposed cases, not a guarantee that an Agent recognizes every missing dependency.

### Prefer authorized action over exhaustive certainty

The [owner asked how to prevent endless context loading and questions](../sources/working-day-discussion-20261016.md#avoid-endless-context-loading-and-questions). Make deciding within the mandate the default duty. An implementation choice does not need a uniquely best answer before work can proceed. Apply the existing [decision-authority rules](desired-working-day-20261016.md#decision-authority-and-escalation); this clarification adds no permission gate or separate research workflow.

The earlier question "would another fact change my action?" needs a materiality boundary. More information might always suggest a different implementation. Additional context is necessary when it could change compliance with intent, authority, an applicable nonfunctional requirement or a consequential interaction with another area, not merely because it might reveal a more elegant option. Mandatory context and checks remain mandatory irrespective of this judgment.

| Situation | Default response |
| --- | --- |
| Several local implementations satisfy known obligations and remain inside the mandate. | Choose a simple maintainable option consistent with the observed project conventions, implement it and use normal review/checks. Do not ask the owner to choose helper layout, local names or another routine preference. |
| A concrete missing fact could alter correctness, ownership or a relevant contract. | Perform the smallest targeted read or bounded diagnostic check that can resolve it. A harmless diagnostic experiment can establish implementation behavior; it cannot invent business policy. |
| An existing cross-area contract or technical disagreement needs coordination. | Use the nearest authorized parent and existing tasks. Technical difficulty does not automatically become an owner policy question. |
| A material business decision or additional authority is genuinely absent. | Escalate that exact decision with the existing concrete question format. If only execution capacity is missing, request bounded additional capacity where justified or stop incomplete; do not fabricate a business decision. |

Before discretionary context expansion, the Manager should be able to state briefly: "I need fact X to distinguish consequence Y from Z; source/check A is the next useful observation." Record this in the ordinary request/task explanation where material, not in another form requiring approval. "I want to understand the whole project better" is not a sufficient reason to delay an otherwise actionable task. A broad exploratory assignment can explicitly grant broader investigation; this default concerns bounded realization work.

After a targeted read/check, decide the next action: proceed within the mandate, inspect a newly evidenced concrete dependency, coordinate a conflict, or record a remaining blocker. Do not repeat substantially the same search or collect unrelated files without new evidence or an identified remaining distinction. Use existing step/time/tool limits, protected completion reserves and stagnation handling to bound investigation. No universal fixed number of reads is proposed: it would be either wasteful for simple cases or unsafe for necessary deeper investigation.

For a permissible local design choice, prefer an existing pattern where it meets the current obligations; otherwise use a simple reversible implementation within the granted scope. Reversibility helps select among authorized options but does not authorize a new business rule, a foreign-file write or an unverified external side effect. Material design choices can be explained in the normal work proposal; no owner approval is required merely because alternatives exist.

In the library example, extracting a private helper for the borrowing-limit check is ordinary implementation freedom unless an applicable project rule says otherwise. Whether the current Catalog already derives availability from loan state is a concrete implementation fact worth reading before adding another update path. Whether an extra loan should be allowed above the modeled limit is not a local optimization. The existing limit rule must be followed; a requested exception would require authorized new intent. Whether suspended members may return is already settled by M1 and should not be asked again.

The existing reviewer assesses both sides: unsupported consequential assumptions are findings, but so are unnecessary escalations that offload an already delegated decision or repeated investigation without progress. Do not reward a raw minimum question/read count; that would encourage hiding genuine uncertainty. Record whether additional context resolved a relevant premise and whether owner intervention was actually required, using existing evidence. These are bounded observations, not new agent-ranking metrics or automatic sanctions.

Include both a routine in-mandate choice that completes without an owner question and a genuinely missing contract fact that triggers a targeted read in the existing working-day assessment. Also preserve the true reserved-decision case. Passing means appropriate autonomy, not either zero questions or maximal investigation. This is a proposed behavior to validate, not a claim that prompting alone reliably prevents hesitation.

### Assess questions using the information available at the time

The [owner asked how to judge escalation without hindsight or punishing caution](../sources/working-day-discussion-20261016.md#review-questions-without-hindsight). Assess whether asking was reasonable under the Manager's actual information, mandate and constraints when it asked. The later answer, successful implementation or absence of harm cannot by itself show that the question was unnecessary. Conversely, a lucky guess does not establish that an unsupported decision was authorized.

Reuse the question's bound model/mandate version, task/step context, observed additional reads, known contradictions, remaining limits and the Manager's short stated missing premise. Preserve what was actually supplied and what could reasonably have been retrieved at that point. This is a concise decision basis, not a request for private reasoning transcripts or another decision log. If that basis is missing, report limited assessability rather than assume that the Manager had today's knowledge.

The existing reviewer first examines that contemporaneous basis without relying on the owner's later answer or resulting fix. Ask:

1. What decision was genuinely open, and what materially different consequences were reasonably apparent then?
2. Did the Manager already have authority to choose, or did the known sources reserve the choice elsewhere?
3. Was there a concrete, reasonably accessible existing rule or bounded observation that would have resolved the issue within the task's actual constraints?
4. Was the recipient and timing appropriate: local decision, organizational information/coordination, owner authority, immediate interruption or grouped question?

Then inspect the later outcome separately to identify lessons about context or contracts. Where outcome information was already visible to the reviewer, explicitly base the judgment on the earlier evidence; do not claim perfect blinding. Choosing to ask and choosing to interrupt immediately are separate judgments, as in the existing mandate/notification distinction.

| Judgment | Required grounds and consequence |
| --- | --- |
| Required owner decision | The material choice exceeded the then-current mandate or reserved authority. The question was necessary even if the owner answered immediately or selected the Manager's recommendation. |
| Reasonable clarification under uncertainty | Known ambiguity, contradictory evidence, unavailable necessary context or a consequential boundary made proceeding unjustified. A later obvious answer does not invalidate that caution. Address any context/contract defect separately. |
| Avoidable or misrouted question | Cite the then-applicable authority or concrete available source and the proportionate alternative the Manager could have taken. For example, make the delegated local choice, read the supplied rule or ask the existing technical parent instead of the owner. "It turned out to be easy" is not a reason. |
| Not assessable from retained evidence | State which historical input/authority evidence is missing. Do not treat the gap as proof of either good or poor Manager judgment. |

Avoidability concerns a specific missed opportunity, not an obligation to search every accessible file. A source somewhere in the repository is not necessarily a reasonably discoverable answer at that time. Consider the supplied references, actual access, known source relevance, budget/deadline and the possible consequence of guessing. Low remaining budget never grants missing business authority, but can make further investigation infeasible. Distinguish a justified stop/capacity request from an invented business-policy question.

Library examples make the difference concrete:

- Asking whether suspended members may return loans is avoidable when the explicit eligibility/return rules were supplied and no contradictory applicable statement was known. The reviewer cites those exact sources rather than its later understanding.
- If the required rule was omitted and a relevant read failed, caution may have been justified; the preparation/retrieval failure needs correction. Whether to contact the parent or owner is a separate routing judgment. Do not blame the Manager merely because the evaluator can now see the missing rule.
- In the deliberately incomplete new renewal extension, asking whether overdue loans may be renewed is required when that choice was reserved for the owner. The owner's quick acceptance of "no" does not mean the Manager was authorized to decide it.
- Asking the owner to choose a private helper layout is avoidable where architecture constraints and implementation freedom were already clear. A material public-interface or architectural choice must be judged against its actual mandate, not dismissed as a mere naming preference.

Keep this assessment inside the existing work/review and retrospective evidence. It must not become a prerequisite review before the Manager is allowed to ask a necessary question. Normally, an avoidable past question is process feedback and a chance to improve supplied context, defaults or mandate clarity; it does not invalidate otherwise correct artifacts or reopen the owner's authoritative answer. An unresolved authority/correctness problem can still be a blocking finding, based on that concrete problem rather than punishment for having asked.

Use repeated comparable evidence to improve the mechanism, with observation, explanation, experience and binding rules kept distinct. No automatic "too many questions" score, penalty for cautious language, mandatory question quota or promotion of a single episode into a rule is proposed. A good question can demonstrate that the authority boundary worked. The practical criterion is whether the Manager used the authority and evidence reasonably available at the time while exposing material uncertainty.

For bounded evaluation, compare otherwise similar cases with different initial authority/context: an explicitly delegated choice, a supplied settled rule, a genuinely reserved decision and a failed required information retrieval. The reviewer should distinguish them even when the eventual answer is identical. Record missing basis as unknown and assess notification timing separately. These are proposed evidence cases, not a guarantee against all hindsight bias.

### Improve context and mandates without turning feedback into policy

This proposal answers the [follow-up about learning from question assessments](../sources/working-day-discussion-20261016.md#improve-context-and-mandates-from-question-assessments). It applies the existing [distinction between observation, explanation, experience and binding rule](experience-without-implicit-rules-20261016.md#four-distinct-kinds-of-knowledge). These are different meanings, not automatic promotion stages. An avoidable question is evidence about one situation, not authority to prohibit similar questions or widen a Manager's mandate.

Start from the contemporaneous assessment above. Record the observed question and supplied basis, then the proposed cause separately: an omitted source, poor presentation, unclear rendering of existing authority, a genuine missing decision, or an unsupported diagnosis. If several explanations remain plausible, retain that uncertainty. A later successful answer does not establish the cause of the earlier question.

Choose the smallest improvement that addresses the supported cause:

| Supported cause | Small improvement | Authority boundary |
| --- | --- | --- |
| A source required by the existing context contract was missing | Correct preparation or routing for the affected responsibility and check the omission | This realizes an existing obligation; it does not create a business rule |
| The right source was supplied but difficult to locate among unrelated material | Make its reference and purpose easier to find; remove demonstrated redundancy | Preserve applicable content and provenance; avoid copying another authoritative rule into prompt prose |
| Existing implementation freedom was unclear in the derived mandate | Show the relevant delegation and its limits more clearly, with its authoritative source | Clarification preserves the set of permitted decisions; it cannot create a new default, exception or permission |
| Authority or intended behavior was actually missing | Propose the concrete change at the owning model or configuration source | Use existing decision authority; reserved decisions still require the owner |
| No context or mandate defect is established | Retain scoped feedback or an optional experience hint for relevant future work | Do not force a configuration change to make every assessment produce an action |

The practical test for a claimed clarification is: would a previously unauthorized decision become authorized, or a previously allowed choice become forbidden? If so, this is a substantive change, even if presented as better wording. A mandatory default can also narrow discretion without using the word "must". Review it through the existing authority path. The derived mandate continues to refer to its model, configuration, registry and assignment sources; it is not a second place to maintain those rules.

Use existing assessment and task records for a short improvement proposal: evidence, suspected cause, affected scope, the exact proposed change and how to tell whether it helped. No new improvement backlog, learning Agent, scoring service or approval round is needed. An authorized preparation defect can be fixed after one clear example. Uncertain diagnoses justify a bounded, reversible trial or further observation, not an arbitrary requirement for several incidents. Repetition can support a broader proposal, but never supplies authority by itself.

Review the change in the normal workflow, checking both the problem case and a nearby case where asking remains necessary. For example, clearer access to the existing suspension/return rule should let Lending proceed with a return without another business decision. A later extension with a genuinely undecided renewal policy must still lead to a question. These are hypothetical examples, not claims of completed runs. For a small presentation change, a focused comparison of the prepared inputs and ordinary reviewer assessment can suffice; a repeatable evaluator case is useful when a deterministic omission was fixed. Do not build a new test suite for every rewording.

Success means that the relevant source is available and discoverable, an authorized decision can proceed, and legitimate escalation still works. Also inspect whether the change unnecessarily broadens input exposure or burden. Fewer questions alone are not success: silence may hide guessing. If later use does not support the diagnosis, narrow, revise or withdraw the improvement while preserving the original observation. Do not make a permanent checklist or mandatory check out of every useful example.

For the library, suppose review establishes that Lending's prepared context omitted the already applicable rule allowing returns for suspended members. Correct that selection and retain a source reference. If the source was already present but buried, improve orientation instead of adding all Members implementation files. If it was clear and available, use individual feedback rather than automatically expanding every future package. None of these observations authorizes a new instruction such as "Never ask about member status" or "Lending may decide all membership policy."

An experience hint can suggest inspecting the relevant contract before escalating a similar issue, with its scope and evidence. A reviewer cannot reject a different valid approach merely for not following that hint. Only an explicitly authorized change to the owning source becomes binding. Apply source changes through the existing version and successor-run rules; do not silently replace the basis of running work or carry old approval to the changed basis. This proposal can begin with ordinary records and focused corrections; the deferred experience storage and selection mechanism is not a first-version prerequisite.

## Overall acceptance and repeatability

The proposed initial threshold is three complete series of all four scenarios, using a fresh execution state for each case and the same fixed product/profile basis. Each case starts from its specified recorded baseline; within the recovery case, the interrupted state must survive the restart. Do not replace recovery with a new clean assignment.

Every series must satisfy all four scenarios without disallowed rescue work. Record failed attempts rather than selecting only successful runs. After a product correction or acceptance-relevant setup change, start a new acceptance series and retain the earlier failures. Three series are a proposed practical threshold, not a statistically established reliability target or an accepted project rule.

For the budget scenario, an honest bounded incomplete result is a passing test of limit handling and remains an incomplete assignment. For the other scenarios, safe stopping is useful evidence but does not substitute for the required successful continuation and final verified/applied result. Unauthorized model changes, stale publication, inherited approval without required reassessment, budget resets or unaccounted concurrent writers fail the relevant case regardless of whether the final code appears correct.

The bounded conclusion after passing is: in these library scenarios, the owner could concentrate on intent and decisions while Markitect organized ordinary development and handled the selected difficulties. Broader reliability, long-term hygiene, operational monitoring, economic advantage and freedom from human code review across arbitrary projects remain separate claims requiring further evidence.

Return to [product reflections](README.md).
