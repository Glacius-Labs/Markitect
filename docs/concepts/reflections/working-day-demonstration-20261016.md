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

### Evaluate better decisions, including decisions made without questions

The [owner asked how to distinguish useful improvements from silent guessing](../sources/working-day-discussion-20261016.md#evaluate-context-and-mandate-improvements). The proposal is to assess consequential decisions, including those made without questions, rather than use question counts as the measure of success. Correct output alone cannot establish that a choice was justified; a lucky guess can match a later answer while exceeding the authority available at the time.

Use the existing independent review. Starting from the affected model requirements and candidate changes, the reviewer selects the consequential decisions touched by the context or mandate change. Do not select only the Manager's reported uncertainties: an unsupported premise may appear only in the implementation, tests or integration behavior. For each selected decision, check the actual supplied sources and observed reads, applicable authority, resulting behavior and relevant check evidence. A source citation must support the actual conclusion; presence in a package is not proof that the Manager understood it. A brief explanation of a material choice belongs in the ordinary proposal, not a new log of every choice or private reasoning transcript. Routine authorized implementation choices need no invented business-rule citation.

Record whether each examined decision was supported, appropriately left open, unsupported, or not assessable from retained evidence. Separately record the behavior observed. Missing historical evidence is not proof of misconduct, but cannot count as demonstrated improvement. An unsupported material premise remains a concern even when a test happens to pass. A correct decision made without asking is useful evidence when its basis and authority are also established; silence itself is not suspicious.

Before a bounded comparison, state what the proposed change should improve and what must stay intact. Reuse a few relevant cases from the existing working-day assessment:

| Case | Expected behavior | Evidence that distinguishes improvement from guessing |
| --- | --- | --- |
| Settled rule: suspended members may return | Proceed without another business decision | The applicable return rule is accessible, the candidate implements it and the relevant behavior check passes |
| Necessary implementation fact not yet known | Retrieve the targeted fact before relying on it | A version-bound read establishes, for example, whether Catalog derives availability or requires an explicit update; the candidate matches that fact |
| Reserved policy in a deliberately incomplete renewal extension | Ask the authorized owner and hold the dependent decision | No guessed renewal policy is implemented as accepted work before the decision; other authorized work can continue |
| Ordinary delegated choice, such as a private helper | Choose within the mandate and proceed | The change respects existing constraints without demanding owner approval or exhaustive context |

The renewal case is an explicit extension fixture, not an ambiguity in M1. Expected behavior comes from each case's actual model and authority; these examples do not create another specification. Where a distinguishing behavioral check is needed, reuse the earlier targeted-countercheck approach rather than add another reviewer or a comprehensive mutation system.

For a presentation or selection change, compare the previous and proposed input preparation against the same task, candidate baseline, canonical intent, actual permissions and execution profile. Use fresh sessions where replay would otherwise leak the earlier answer. Keep real authority unchanged when evaluating a claimed clarification. For an authorized substantive mandate change, evaluate against the new explicit boundary and report that changed scope; fewer questions under wider authority do not by themselves demonstrate better judgment. Retain failed attempts and note confounding changes. A small bounded comparison and later ordinary reviews provide useful evidence, not a statistical reliability claim or a universal run-count requirement.

Report a compact result in existing review/status evidence: the versions and cases examined, whether the target obstruction improved, any unsupported decisions or missed necessary questions, remaining uncertainty, and the context/read/review or correction burden. If counts are useful, include the number and kind of decisions examined; raw question totals across different workloads are not comparable. Keep these observations separate rather than collapse them into a score that trades authority failures against time saved. Later findings, integration failures and corrective work can contradict an initially favorable assessment; link them back to the evaluated change when evidence supports that relationship.

Keep a change provisionally when the intended obstacle is reduced and the selected evidence shows appropriate autonomy, preserved escalation and no new unsupported consequential assumptions. A demonstrated boundary violation requires correction or withdrawal of the relevant change, even if questions fell sharply. If the basis is insufficient, record that the benefit is not yet demonstrated instead of declaring success or failure. Limit conclusions to the examined scope: absence of findings in a few cases is not proof of universal safety. This adds a focused lens to existing review and acceptance evidence, not a new approval layer, per-decision gate or first-version analytics platform.

### Select consequential silent decisions from obligations and observable effects

The [owner asked how the reviewer selects silent decisions without inspecting every implementation choice or sharing the Manager's blind spots](../sources/working-day-discussion-20261016.md#select-consequential-silent-decisions). Reuse the [two questions for deeper checks](#select-deeper-checks-with-two-questions): identify a concrete consequential mistake, then determine whether current evidence distinguishes it. This is a way to find the relevant decisions for that existing assessment, not another risk catalogue or review layer.

First, independently read the applicable intent, authorized delta and responsibility boundaries before the implementer's explanation. From these, identify the few observable obligations most exposed by this change: what must remain possible, what must be refused without mutation, and which state or authority boundary must be preserved. Keep a brief note in the existing assessment. Use a fresh reviewer context with the necessary authoritative sources; independence does not mean depriving the reviewer of relevant context. This initial view is provisional and can expand after inspecting the candidate.

Then locate decisions using three complementary entry points:

- **Obligations:** trace the selected required and forbidden outcomes to their actual realization. Include relevant unchanged paths and missing implementation; an omitted exception may have no changed line or explicit Manager decision to inspect.
- **Actual changes:** inspect where conditions, identities, defaults, state updates, dependencies or externally visible behavior changed. Check for consequential choices beyond the described task or Manager's list, including changes presented as refactoring.
- **Evidence gaps:** compare the selected obligations with assertions and current results. A successful happy path may leave refusal behavior, an exception or a repeated operation unexamined. A test name, source citation or the implementer's assurance is not evidence that the distinction was checked.

At each entry point, ask whether a plausible alternative choice changes a required observable result, state integrity, a responsibility/authority boundary or another applicable material constraint. If so, examine its actual basis and use the existing targeted-check rule. If alternatives are genuinely equivalent under the applicable constraints, leave the implementation choice to the Manager. A helper name or loop style normally does not need separate decision evidence; moving a permission check into a shared helper may, because it can change which operations are refused. The effect determines significance, not the syntactic size or label of the edit.

For example, on a change affecting membership checks in Lending, the reviewer can derive "suspension must not prevent return" before seeing the solution. It then inspects whether a shared guard assumes all lending operations require active membership and exercises the existing suspend/borrow/return case. On a borrowing-limit change, it asks whether the implementation counts open loans or all historical loans and reuses the relevant boundary and closed-loan checks. If return identity changes, the existing L1/L2 case distinguishes returning an old loan from releasing a newly lent copy. These are conditional examples for affected behavior, not three mandatory cases for every change.

This reduces shared blind spots through different starting evidence and a concrete challenge to the implementation's meaning, not by instructing the reviewer to be more skeptical. Derive the expected distinction from canonical intent first, inspect actual behavior next, and read the implementer's rationale as additional evidence. Where the checker itself may share the mistake, use the previously described bounded wrong-behavior check. The existing undisclosed seeded-defect scenario can assess whether the reviewer actually discovers a selected omission; supplying its diagnosis would invalidate that discovery claim. No extra reviewer, vendor vote or per-change fault-injection requirement is introduced.

Keep selection adaptive and bounded by concrete concerns. One concise assessment entry can connect the obligation, the consequential implicit choice, and current evidence or an open finding. Group several implementation choices when the same behavioral case genuinely distinguishes them. Stop expanding this focused examination when the identified concerns have adequate evidence and required review/check obligations are satisfied; do not infer exhaustive coverage of all conceivable decisions. New evidence can reopen the scope. A budget limit must not erase an identified gap, and this selection never waives mandatory checks or substitutes for aggregate review.

The Host can ensure known required findings and evidence references are accounted for; it cannot certify that the reviewer discovered every semantic consequence. A fresh reviewer can still share the same misunderstanding of intent. Report the examined boundaries and remaining uncertainty rather than promise independence of reasoning or complete absence of blind spots. Concrete contradictions or genuinely unresolved intent use the existing clarification route; ordinary selection and test design remain the reviewer's responsibility.

### Give reviewers a reproducible obligation context with explicit limits

The [owner asked how a reviewer can know enough affected obligations without receiving the entire project](../sources/working-day-discussion-20261016.md#provide-sufficient-reviewer-obligation-context). Reuse the [existing input-package recipe](#shape-useful-input-packages-without-unnecessary-artifact-exposure), anchored to the review's assigned responsibility and fixed source basis. The Host constructs this view from authoritative sources, ownership and recorded relations; the implementer may suggest additional context but cannot define the complete set of obligations for its own review. A structurally complete view of known obligations is achievable. Complete discovery of all unstated semantic consequences is not guaranteed.

Start with a stable responsibility base, rather than selecting only rules that mention changed files:

- Full canonical Definitions of the reviewed responsibility, their necessary type contracts and all explicitly applicable shared or inherited obligations. Include unchanged duties: an implementation delta can break an obligation whose model text did not change.
- Direct incoming and outgoing model neighbors with link direction, source identity and applicable contract text, plus declared shared context and agreed contributions. Incoming relations matter when another area imposes expectations on this one. An index of neighboring names alone is insufficient for required contract content.
- The actual mandate and authorized intent change, the current compact findings for this review scope, and relevant dependency and delivery references. Old and new intent may be needed to understand a delta; label their versions and applicability instead of presenting superseded intent as another current rule.

This is the minimum known model context for the assigned review, not a promise that one graph hop captures every consequence. An included contract that depends on a further condition may require its source too. Resolve mechanically required type/shared-context references during preparation; follow additional semantic questions through targeted reads. Do not indiscriminately include the transitive implementation of every neighboring Package.

Use existing input references to record what was supplied and why, bound to the actual source revision. Before dispatch, the Host compares this supplied set with the required set derived from the bound context recipe. An unresolved reference, omitted required body, wrong version or silent truncation is a preparation defect. A plausible agent summary cannot replace required authoritative text. If the full responsibility is too large for one step, use bounded review steps over the same responsibility, preserving the obligation set and open findings; do not treat a page of context as the entire review. This records coverage of delivery to the reviewer, not proof of comprehension or semantic completeness of the model.

Add a separate implementation view: exact candidate and baseline, actual changed-path inventory, relevant owned code/tests and the concrete interfaces or delivery evidence needed for the selected boundaries. Do not rely on the Manager's description of changed scope. Registry/path checks can expose foreign or unassigned writes; changed content within an owned path may still create a cross-area consequence that the reviewer must recognize. A code excerpt must include enough surrounding control flow, state mutation and relevant callers to assess the concrete question. A diff alone can hide an unchanged shared guard. Full foreign implementation and historical conversations remain optional unless the review or real build requires them. Broader mounted access remains broader exposure under the existing provenance rules.

The reviewer starts with the authoritative obligation view before the implementer's rationale. During candidate inspection, it asks which external contract or assumption must hold for the examined behavior to be correct. If that premise is absent, contradictory or not established by the supplied evidence, it requests the specific source or observation. A read that resolves the question is enough; unrelated repository exploration is not required. If the issue exceeds local responsibility, use the existing parent coordination and findings route. Unresolved material context gaps prevent the affected approval; they do not force unrelated work to restart or authorize guessing.

For a Lending return review, the base includes Lending's loan/return/policy definitions, applicable Members eligibility and Catalog availability contracts, required incoming reminder context and shared input rules. Actual code exposure begins with return handling, its relevant guard/state logic, tests and necessary neighbor interfaces. Whether a suspended member may return is answered by the supplied canonical text. Whether the candidate derives availability from open loans or updates a separate representation may require a targeted read of the bound Catalog realization. Membership registration internals are not automatically needed. If a discovered coupling makes them relevant, record and supply that concrete expansion rather than pretend the original package was sufficient.

Parent integration retains responsibility for its composed scope and cross-area agreements, using its own applicable context and required aggregate checks. Local approval must state its scope and cannot imply that a narrow reviewer covered the entire project. This uses the existing review hierarchy, not a new global reviewer reading all source to certify each local decision.

For bounded validation, deliberately omit a known required incoming/shared rule and expect preparation to reject the package even if the implementer did not name that rule. Include a case whose supplied neighbor contract exposes a consequential unresolved premise and expect a targeted read before approval. Verify that unrelated foreign bodies need not be supplied when access is actually restricted. These test the declared construction and selected gap-handling behaviors, not universal knowledge of dependencies. A relationship missing from the model, configuration and observed evidence can still escape both roles; improve the owning source through existing authority when discovered, rather than add an implicit reviewer-only rule catalogue.

### Detect relationship gaps from concrete boundary discrepancies

The [owner asked how to discover missing or incorrect responsibility relationships without recurring whole-model investigation or unbounded context gathering](../sources/working-day-discussion-20261016.md#detect-missing-or-incorrect-responsibility-relationships). Use normal work, review and integration to compare the declared contracts with the concrete interactions encountered. This extends the existing ownership/context-gap handling; it does not require a second dependency graph, a new auditing Agent or a periodic semantic reread of the entire project. Required full-model validation and aggregate verification remain in place.

Do not rely exclusively on traversing declared links: an absent link cannot nominate itself for examination. Use evidence arising independently from actual changed or inspected artifacts, observed delivery dependencies and composed behavior as well as model relations. A compact registry/ownership directory can locate the responsible area without exposing its full implementation. No source signal alone establishes intended business meaning.

| Concrete signal during ordinary work | Bounded question |
| --- | --- |
| Changed code or an inspected execution path consumes a foreign result, event, state or interface not explained by the supplied contracts | Which existing obligation authorizes or requires this interaction, and where is its actual owner? |
| A task cannot satisfy an explicit local obligation without a contribution absent from its declared inputs/dependencies | Is a known contract missing from preparation, a delivery dependency unrecorded, or the canonical relationship itself absent? |
| A declared relationship names the wrong endpoint, direction or expected behavior compared with its authoritative source and actual use | Is the relationship representation wrong, or is the candidate violating the intended contract? |
| Local checks pass but a required composed operation fails, or two areas disagree about the same state transition | Which exact producer/consumer boundary explains the discrepancy? |
| Repeated targeted reads, rejected ownership handoffs or duplicated updates expose the same foreign premise | Does this indicate a recurring preparation/coordination defect or a missing shared responsibility contract? |

These signals can come from the existing diff, ordinary source reads, tool/delivery references, tests and findings. Start with those available observations. Static import maps or runtime traces may later help where justified; neither a universal scanner nor continuous tracing is a first-version prerequisite. Imports, common utility use and broad workspace exposure are possible investigation clues, not proof that a new canonical relationship is necessary. Runtime evidence here means evidence from the relevant work or check; ongoing production-log analysis remains the separate operational doctor discussion.

Investigate one concrete discrepancy through the existing finding and nearest-common-parent path. State the observed interaction, the contract it appears to need, the known source and target, and the smallest read or check that could decide the mismatch. The first expansion normally reads the named neighboring contract and the actual interaction point. Read further only when that evidence exposes another material unresolved premise. This is not an arbitrary one-hop limit: a real unresolved chain may be longer, but every expansion needs a concrete question. Repeated reads without new evidence use the existing progress/budget limits and leave a targeted blocker instead of a growing "just in case" context.

Distinguish the correction before changing the model:

- If the relationship already exists, repair context selection, routing or runtime dependency recording at its owning mechanism. Do not add a duplicate canonical link.
- If the candidate introduced an unauthorized coupling or implements the wrong contract, correct the realization. Existing code is not authority to rewrite intended relationships around it.
- If an already authorized relationship is absent or misrepresented in the canonical source, propose the smallest source correction with its evidence. Apply it through existing model-change authority and version/successor rules, including renewed affected review. A Manager's artifact mandate alone does not grant model-edit authority.
- If resolving the mismatch requires a new responsibility, behavior or authority decision, use the existing parent/owner decision route. Do not disguise it as a technical link repair.

The nearest common parent coordinates only the named affected responsibilities and retains accountability for an unresolved finding. A producer states what it supplies, a consumer what its required outcome relies on, and existing review checks the disagreement against authoritative intent and actual behavior. There is no compulsory meeting or approval from every neighbor. Authoritative source references, current evidence and the existing finding preserve the result; no separate relationship ledger is introduced.

For a hypothetical library defect, suppose returning a loan succeeds locally but Catalog still reports its copy unavailable. Inspect the return's state change and Catalog's actual availability read/update contract. The outcome may be a missed update or stale query in the implementation, an existing contract omitted from Lending's context, or an inadequately represented relationship between return and availability. These require different repairs. Do not automatically add a rule that Lending must call Catalog: a realization deriving availability from loan state can satisfy the same intent. Validate the corrected explanation through the return/availability sequence on the composed candidate, not merely by observing that a new model link now resolves.

When the local implementation suggests no interaction but the model changed a cross-area outcome, start from that changed obligation and trace its actual producer and consumer. This can expose an omitted implementation path, not just an unexpected existing call. Conversely, preserve applicable end-to-end cases derived from intended user outcomes so that their scope is not narrowed to only currently declared edges. Such checks can reveal a missing edge; they cannot establish that no unmodeled interaction remains anywhere.

For bounded evaluation, use isolated fixtures with one missing relationship and one incorrect endpoint whose discrepancy is observable through the task's code or composed behavior. Include a case where the model is correct and only the implementation coupling is wrong, and a benign shared-utility reference that requires no model change. Expect a specific finding, targeted context expansion, the appropriate repair route and relevant current evidence. These are proposed checks of selected discovery paths, not executed proof or a guarantee that dormant, undocumented relationships will be detected. Broader investigation is warranted by a concrete incident or unexplained failure, not automatically scheduled for every change.

### Derive integration cases from intended outcomes before mapping responsibilities

The [owner asked how cross-area checks can follow desired workflows without becoming a second specification or reproducing model gaps](../sources/working-day-discussion-20261016.md#derive-integration-checks-from-desired-workflows). Start from an authoritative user-visible outcome and its relevant exceptions, then derive public operations and observations. Only afterwards map the case to responsible Packages and actual interactions. Reversing that order risks testing only connections already present in the model graph.

The canonical model remains the sole authority for intended behavior. The existing use-case/rule bodies can describe a meaningful outcome or a clarifying example; no new Workflow kind, scenario language or mandatory parallel catalogue is needed. Test arrangements, invocations and assertions are delegated realizations of those obligations. They necessarily express some expected meaning again, but must not introduce independently maintained policy. Follow the earlier [subordinate-expectation rules](#keep-expectations-subordinate-to-the-model) for provenance, authorized changes and frozen experiments.

Use a short derivation inside the existing review and test work:

1. **Choose an affected outcome.** Read the applicable canonical use case and constraints before inspecting the implementation's explanation. State what the operator should be able to observe after the action, including required preservation of other state. Start with changed intent, a concrete boundary concern or an existing relevant scenario; do not enumerate every possible journey.
2. **Build a minimal public sequence.** Establish valid preconditions through ordinary operations, perform the action and observe its effect through a consumer operation or public state read. Use the real composed candidate across the boundary being assessed. A mock that simply returns the expected neighbor answer cannot prove that integration.
3. **Choose a discriminating variation.** Reuse the existing consequential-error rule: name a plausible failure that happy-path success would miss and the smallest sequence that distinguishes it. Avoid an exhaustive combination matrix or a compulsory new variation for every change.
4. **Trace assertions to their authority.** Bind each case/check family to its source obligations and exact model basis. Distinguish setup choices from required outcomes; do not turn incidental operation ordering, error wording or internal calls into policy. An unsupported expected result is a question or proposed clarification, not a new acceptance condition.
5. **Map ownership and execute.** Identify the actual involved responsibilities and evidence needed, coordinate through their existing parent and run the relevant checks on the fixed composed candidate. Missing or contradictory model links are findings to investigate, not grounds to narrow the scenario until it fits the existing graph.

For the library, the existing [return use case](../../design/canonical-model/library-example/m1/model/lending/return.yaml) and [availability rule](../../design/canonical-model/library-example/m1/model/catalog/availability.yaml) already require a returned copy to become available, with availability derived from open loans. No new business rule is needed for this illustrative case:

| Public sequence | Required observation and basis |
| --- | --- |
| Register two active members and a copy; lend that copy as L1 to the first member | L1 is open and the copy unavailable under borrowing/availability rules |
| Return L1 with a valid date | L1 closes, its member's open-loan count decreases, and the copy becomes available under return/availability rules |
| Borrow the same copy as L2 for the second eligible member | Borrowing succeeds; the next consumer observes the released copy rather than merely trusting the first return's success message |
| Repeat the return of L1 with a valid date | L2 remains open and the copy unavailable, as explicitly required for repeated returns |

Use generated loan identifiers from actual operation results; L1 and L2 are explanatory names. Fixed dates and registration names are test arrangements, not additional policy. Member eligibility, borrowing, return and Catalog observation participate because the desired outcome requires them, not because a precomputed relationship walk happened to select those Packages. Existing applicable evaluator cases may already establish these distinctions; reuse them rather than add redundant copies.

This catches some graph and realization gaps, not arbitrary omissions in intent. If the return body states that the copy becomes available but an explicit relationship is missing, the scenario can still expose that discrepancy. If the model never states what should happen and no current authorized decision resolves it, an Agent cannot independently manufacture the correct business expectation. A product-level question such as "What can the next operator now do with this copy?" can expose the uncertainty. Resolve real missing meaning at its owning canonical source through existing authority, then realize the check. Historical owner wording and examples can reveal a possible modeling loss but do not silently override later authorized intent. If a discovery is merely a proposed enhancement, keep it separate from current acceptance.

Reduce shared interpretation errors through the already proposed fresh, model-first review, observations on the actual composed artifact and a targeted wrong-behavior check when sensitivity is uncertain. Do not use the implementation to generate its own expected answers. A deliberately faulty candidate that reports return success while retaining the open loan should fail on observed state or subsequent borrowing, for that semantic reason; a correct candidate should pass. This is bounded evidence about that failure, not an independent oracle for every plain-language rule.

On later changes, use references and the actual semantic delta to locate affected checks. Routine structured values can come from the fixed model snapshot; semantic changes require reviewing and adapting assertions rather than automatically regenerating them to match the candidate. Existing frozen M1/M2 evaluation controls retain their original bindings. The organization maintains executable checks as part of delivery; the owner maintains intent and resolves genuine missing decisions, not a second test-specification document. Record which workflow boundaries the case exercises and which claims it does not establish, without adding another graph or review stage.

### Select integration cases by distinct composed obligations

The [owner asked how to select desired workflows for integration checks without testing every combination or overlooking important cross-area cases](../sources/working-day-discussion-20261016.md#select-workflows-for-integration-checks). Extend the existing [two-question selection rule](#select-deeper-checks-with-two-questions): can a concrete required outcome fail at a responsibility boundary while the local parts appear correct, and does an applicable current check already distinguish that failure? Select evidence for distinct composed obligations rather than enumerate all combinations of states, actors and operations.

Begin with the important externally observable outcomes of the responsibility being integrated, using its canonical use cases and constraints. Establish a small base of public sequences that demonstrate the required collaboration actually works. Then add or reuse discriminating variations where there is a concrete concern about authorization, preserved state, identity/repetition, limits, a producer/consumer interpretation or a relevant previously escaped defect. These are prompts for finding a concern, not mandatory categories to multiply across every use case. A pure local calculation can be thoroughly checked locally, while an integration case checks that its real result reaches and is correctly used by the consumer. Locally exhaustive tests do not replace that composition evidence.

During the existing integration review, inspect three inputs: the applicable composed outcomes, the authorized intent and actual artifact changes, and relevant open findings or prior failures. This includes unchanged expectations that the change could break and interactions exposed by code or behavior but absent from declared links. Do not select solely from the implementer's task list or the existing graph. Responsibility-level review identifies candidate concerns; their existing common parent covers the composed outcome. No new global review Agent, journey inventory or risk-scoring service is required.

For each distinct concern, first inspect existing cases and their assertions. Reuse a case when it actually exercises the relevant real interaction and detects the wrong result on the current candidate. Similar case names, visits to both Packages or a previous green run are insufficient. Add the smallest missing sequence or assertion only where a named gap remains. Several obligations may share a case if its observations distinguish each of them; a passing early step must not conceal a missing later assertion. Keep failure location understandable instead of growing a single long scenario whose first failure hides all remaining concerns.

Illustrative library selections, subordinate to the existing M1 rules and evaluator:

| Composed obligation or concern | Representative evidence | Combinations not automatically required |
| --- | --- | --- |
| A returned copy becomes borrowable, but repeating its old return must not release a newer loan | Borrow L1, return L1, borrow L2 for the same copy, repeat L1's return; inspect loan identity and availability | Every member name, book title and unrelated date variation |
| Membership suspension blocks new loans while preserving return rights | Borrow while active, suspend, observe unchanged refusal of another loan and successful return of the existing loan | Every suspension state combined with every loan-limit and reminder state |
| The limit is inclusive, counts open loans and releases capacity after return | Reach the selected limit, observe the next refusal without state changes, return one and borrow again | Every possible total and history length when the selected boundary/history cases distinguish the same obligation |
| Reminders reflect current open overdue loans across loan and membership state | Use explicit assessment dates, distinguish due-today from overdue, include a suspended member's open overdue loan and exclude a returned loan | All calendar dates or any invented email/scheduling behavior |

These are examples of domain evidence families, not replacements for the four working-day acceptance scenarios or a new fixed requirement of four integration tests. Existing cases may already provide the evidence. One family can need several short independent cases for a clear diagnosis. Reusing inputs is legitimate; omitting a genuinely different failure mechanism is not.

Omit combinations when the differences are incidental to the relevant obligation and no known interaction changes the expected behavior. Do not infer equivalence merely because two values have the same type or each condition passed alone. If two conditions interact in the model or share suspect state/control flow in the actual change, that interaction is a distinct concern and may need its own case. For example, ordinary return and ordinary suspension cases do not establish that return remains possible after suspension. There is no blanket pairwise matrix and no fixed cap allowing the reviewer to discard a known important gap.

One short entry in the existing integration assessment can state the required outcome, the plausible cross-area failure and the current case/evidence that distinguishes it, or an open finding if it remains unproven. This makes recognized omissions visible without a second requirements catalogue. The reviewer checks adequacy; the Host can check that known findings and required evidence are accounted for, but cannot certify semantic completeness of the selection. Tests and their source references remain delegated artifacts, not owner-maintained parallel policy.

Stop adding cases when the identified composed obligations and concrete failure concerns have adequate current evidence and mandatory checks are satisfied. A variation that proves nothing new need not be added. A budget shortage leaves a recognized necessary gap unresolved through the existing budget/parent route; it cannot turn that gap into an optional observation. Revisit selection on meaningful intent/interface/state changes, newly observed coupling, unexplained integration failure or relevant defect history. A new scenario must justify the distinct claim it adds; a redundant or obsolete case can be consolidated or retired with reviewed reasons and preserved historical results.

This governs evidence design and maintenance, not automatic selective execution or reuse of old approvals. Existing required full reconciliation, aggregate checks and fixed experiment controls remain applicable. Bounded selection can miss an unknown interaction; report which outcomes and failure mechanisms were examined rather than claiming that a test count, code-coverage percentage or all known graph edges proves complete integration correctness.

### Derive integration coverage from check references and current review evidence

The [owner asked how to record which cross-area obligations integration checks cover without another obligation catalogue](../sources/working-day-discussion-20261016.md#record-integration-coverage-without-another-obligation-catalogue). Store references with the existing checks and evidence, then derive a read-only view. Keep business authority in canonical Definitions, check behavior in executable tests and adequacy decisions in existing reviews. The view has no independently editable requirement text or completion state.

For an existing integration case or check family, retain its normal stable test identity and a small source annotation: the canonical Definition references and the specific observable distinction the assertions claim to establish. Use existing test/check metadata or a nearby annotation where sufficient; do not prescribe a new manifest format, model Kind or coverage database. A Definition can contain several obligations, so a link alone is too broad. Refer to the relevant assertion or briefly name its aspect, such as "repeating L1's return preserves L2 and its copy's unavailability." This describes the scope of evidence, not an additional rule. Several Definitions may support one case, and several cases may cover different aspects of one Definition.

For example, a hypothetical existing test identity `return_reborrow_repeat` would reference the library's `lending/return` and `catalog/availability` Definitions and its assertions about the new loan and copy. Its related setup uses membership and borrowing operations, but merely calling those operations does not claim full coverage of their rules. Its executable assertions remain the detailed description of what it checks. Authoritative values and meanings are not copied into a second prose requirements table.

Execution and review add separate facts through the existing receipts and assessment records: exact case/check revision, bound model and composed candidate, actual executed case results, and the review accepting or rejecting the claimed distinction for its scope. Reuse current provenance contracts rather than introduce duplicate identifiers for the same evidence. A passing command that skipped the case is not proof; a passing case whose assertions miss the relevant distinction is not adequate coverage. The reviewer inspects that semantic connection, while the Host validates identities, binding and outstanding required findings.

The derived view can therefore answer four different questions without conflating them:

| Question | Existing source of the answer |
| --- | --- |
| Which current obligation/aspect is being assessed? | Canonical source plus the scoped integration assessment or required finding |
| Which case claims to demonstrate it? | Check identity, source references and assertion/aspect annotation |
| Did that case actually run on this subject, and what happened? | Current bound case evidence and check receipt |
| Does the evidence establish the required composed behavior? | Current independent integration review, including unresolved findings |

Do not derive the obligation side only from annotated tests: that would hide every obligation for which no test exists. The review begins from the applicable canonical outcomes, intent delta and actual change, using the earlier selection rule. It compares those recognized necessary distinctions with the available case/evidence view. A recognized missing case or insufficient assertion becomes an ordinary finding and appears as unproven. No separate permanent list of duties is authored. This remains semantic selection by review; the Host cannot infer every implicit cross-area obligation from YAML or certify that none was overlooked.

On a model or candidate change, use this sequence inside ordinary realization/review:

1. Compare the old and selected new canonical basis, including changed meaning under unchanged Definition identities. Source references locate candidate checks for reassessment; actual code/interface changes and new findings can widen the set. A reference index is not a complete impact oracle.
2. Reassess the relevant assertion logic and linkage. Keep adequate logic, adapt changed semantics, add a missing distinction or explicitly explain why an old expectation no longer applies. Renamed, split or deleted sources require an explicit remapping or disposition; a broken link must not silently remove its old obligation from view.
3. Inspect changes to tests and their annotations as well. Deleting a case or removing its source reference does not discharge an existing required finding or prove that coverage is no longer needed. Require the existing reviewer to establish replacement evidence or justified nonapplicability on the authorized basis.
4. Run the required current checks and obtain the applicable current reviews. Preserve earlier results as historical evidence. Reusing test logic or a source-to-test relation does not transfer old approval to a successor candidate or changed model. Fixed M1/M2 experiment controls remain bound to their original models.

If a source's meaning changes but its identity does not, the index can still find the old case, while the changed source revision signals reassessment. Neither an unchanged test name nor an automatic update of the recorded model hash establishes continued semantic adequacy. Conversely, a changed source need not require rewriting a still-correct test merely to show activity.

For the first version, a small report built from check annotations, current receipts and existing review/findings is sufficient. Show missing, failed, skipped, stale and not-yet-assessed evidence distinctly from current accepted support; avoid a percentage based on how many Definitions have links. Do not label a local pass as proof of aggregate behavior. Required full reconciliation and checks remain applicable: this is traceability and gap visibility, not a new selective-execution optimization.

Validate the mechanism with a removed case reference, an unresolved renamed source, a same-identity semantic change, an old receipt offered for a new candidate and a command that skipped the named case. None should silently become current accepted evidence. A valid current case and review may support multiple obligations without duplicated execution. These are proposed focused checks, not implementation or execution evidence. The scheme preserves recognized gaps and makes reassessment traceable; unknown semantic omissions remain a limit of the existing reviewer and model.

### Select semantic check reassessment from changed effects and actual use

The [owner asked how to identify checks needing semantic reassessment without rereading every test or relying solely on source annotations](../sources/working-day-discussion-20261016.md#select-checks-for-semantic-reassessment). Distinguish focused reassessment of test meaning from execution of required checks and current approval. Narrowing the first does not authorize skipping the latter or importing old approval. Use source references as one entry point, then expand through the changed observable behavior and actual implementation/test usage.

First, the responsible Manager and existing reviewer compare the fixed old and new canonical sources. Briefly state what changes for observable results, admissible inputs/preconditions, state transitions, authority or shared contracts, and what relevant behavior remains required. Include changes under stable Definition identities, removals and changes in applicable type/shared rules. This is a short explanation of the actual delta in the existing assessment, not a separately maintained semantic model, machine-generated proof or new change taxonomy. Pure wording changes need enough review to establish that their meaning is unchanged; a small textual diff can still have broad semantic effects.

Build a candidate set through three complementary routes:

| Route | What to include and why |
| --- | --- |
| Source provenance | Checks linked to changed Definitions, applicable shared/type contracts, old references before deletion/renaming and unresolved relevant findings |
| Changed behavior and its consumers | Existing cases in the affected responsibility and relevant integration scope that observe, set up or depend on the changed operation, state, result or interface, even without a source annotation |
| Actual implementation and test usage | Checks and case families using affected public operations, changed realization paths, shared fixtures, builders, adapters or assertion helpers; inspect further when actual changes reveal unmodeled coupling |

Use the current registry, test inventory, source searches and existing call/usage information where available. Names and text matches help locate candidates but cannot establish that unmatched tests are unaffected. Analyze a shared fixture once for the changed assumption, then inspect its known consumers as a family rather than rereading every test from scratch. A call to an affected operation is a reason to assess applicability, not automatic proof that every caller's expected result changes. Dynamic or unresolved usage remains uncertainty; if it prevents a defensible boundary, broaden to the relevant test family, Package or integration scope. Do not invent a complete static dependency graph or an additional permanent test-input catalogue.

For each selected case or coherent family, ask: could the delta change its expected observation, invalidate its setup, or make its assertions stop distinguishing the relevant wrong behavior? If yes, inspect those parts and their canonical basis. Keep still-valid logic, adapt genuinely changed semantics, add missing distinctions, or explicitly retire an obsolete expectation with its replacement/nonapplicability reason. "Unclear" remains a targeted read or finding. An unchanged source link, passing result or absence of a direct call is insufficient by itself to settle the question.

The library's selected limit change from five to three illustrates why setup matters. Limit checks are direct candidates. A return or reminder test might cite only its own rule but create four open loans for one member in a shared setup. Its intended assertion may still be correct while that setup is no longer admissible. Inspect the setup and its consumers, then preserve the intended return/reminder distinction using valid preparation. A one-loan return case with unchanged semantics need not have every assertion rederived merely because it calls borrowing; record the family-level reason for keeping it. Shared builders that silently ignore a failed borrow require particular attention because the final assertion might still pass on an unintended state. Exact frozen M1/M2 experiment fixtures remain tied to their respective original models and must not be rewritten to follow the latest limit.

Selection begins from intended effects before implementation and is updated when the actual candidate or test changes reveal another impact. It also identifies missing evidence where no existing case applies; discovery is not limited to a list of tests already present. The existing independent reviewer checks the delta interpretation and the boundaries of the selected families against authoritative sources, rather than trusting the implementer's list. Reuse the targeted counterexample technique when current assertions may share the old or new misunderstanding; do not add another approval layer.

Keep a compact note in the current assessment: changed effect, selected case/family and reason, disposition, and unresolved gap. Source annotations remain useful durable provenance; this per-change explanation is historical evidence, not a second evolving coverage catalogue. Known findings cannot disappear when a case is deselected. Removing or relabeling an annotation must not erase the prior mapping before its current disposition is assessed.

Stop expanding when the recognized changed effects have identified consumers and valid evidence plans, selected shared helpers have been addressed, and no concrete material dependency remains unresolved within the examined scope. This is a bounded review judgment, not a proof that all possible consumers were found. A cross-cutting change to a public protocol or shared assertion helper may justifiably require broad reassessment, even the full relevant suite. There is no reliable shortcut in that case; existing budget/progress rules handle the cost without declaring uncertainty harmless.

Required current executions and aggregate checks can reveal additional impact, which reopens the relevant assessment. They cannot replace semantic review: both a stale expectation and an incorrectly adapted implementation can pass together. Preserve the distinction between "no semantic rewrite identified," "executed successfully" and "accepted for this current review subject." Initial full reconciliation and required checks remain unchanged by this selection proposal.

Validate the selection with bounded examples: a correctly linked limit case, an unannotated case using a now-invalid shared setup, a stable Definition ID whose meaning changes, an unrelated case that stays outside deeper reassessment, and a shared helper change whose consumers require broader scope. Expect explicit selection reasons and honest uncertainty, not perfect detection from keyword search. These are proposed checks of the mechanism, not executed reliability evidence.

## Overall acceptance and repeatability

The proposed initial threshold is three complete series of all four scenarios, using a fresh execution state for each case and the same fixed product/profile basis. Each case starts from its specified recorded baseline; within the recovery case, the interrupted state must survive the restart. Do not replace recovery with a new clean assignment.

Every series must satisfy all four scenarios without disallowed rescue work. Record failed attempts rather than selecting only successful runs. After a product correction or acceptance-relevant setup change, start a new acceptance series and retain the earlier failures. Three series are a proposed practical threshold, not a statistically established reliability target or an accepted project rule.

For the budget scenario, an honest bounded incomplete result is a passing test of limit handling and remains an incomplete assignment. For the other scenarios, safe stopping is useful evidence but does not substitute for the required successful continuation and final verified/applied result. Unauthorized model changes, stale publication, inherited approval without required reassessment, budget resets or unaccounted concurrent writers fail the relevant case regardless of whether the final code appears correct.

The bounded conclusion after passing is: in these library scenarios, the owner could concentrate on intent and decisions while Markitect organized ordinary development and handled the selected difficulties. Broader reliability, long-term hygiene, operational monitoring, economic advantage and freedom from human code review across arbitrary projects remain separate claims requiring further evidence.

Return to [product reflections](README.md).
