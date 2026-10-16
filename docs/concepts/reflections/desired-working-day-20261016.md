# A working day with delegated realization

Date: 16 October 2026. The desired user experience from the [owner discussion](../sources/working-day-discussion-20261016.md), which the owner asked to keep, and the scope of a first usable version. The interaction choices are assistant proposals. The [Apply concept](../../design/canonical-model/manager-apply-concept.md) and its contracts own the rules; this note keeps the picture and the reasons. It builds on [trust through organized agent work](organization-and-trust-20261016.md).

The person changes and approves intent. The organization realizes it, handles ordinary implementation and review, and asks for a decision only when its authority or the model is insufficient. The person can inspect progress without coordinating every Agent exchange.

## Morning model change

The owner lowers the library's admission limit from five open loans to three; existing loans stay intact and returnable. Editing a draft starts nothing. An explicit instruction starts Apply against the selected model revision and the existing realization. A short acknowledgement states the change, revision, scope and existing authority without asking again for permissions already granted.

The root delegates outcomes: lending owns borrowing, while membership, catalog and reminders get their assessment. One task writes at a time, and ready tasks advance while another waits. The owner sees a compact status such as:

> In progress: lower the admission limit to three. Lending is updating its realization; related return and reminder behavior is being assessed. No owner decision is currently needed.

The task tree, actual prompts, candidate differences and evidence are available on demand.

## Decision authority and escalation

Two questions stay separate: who may resolve an issue, and when the owner needs to be told. A question can need owner authority without an immediate interruption; a serious execution failure can need prompt notice although the business rule is clear.

| Situation | Responsible action | Library example |
| --- | --- | --- |
| The answer already exists | Retrieve the model passage or decision and follow it. | A suspended member may still return a book. |
| The choice is within implementation freedom | The Manager decides, explains material choices and submits them to review. | How the borrowing check is structured. |
| Several areas must coordinate | The nearest common parent coordinates within its mandate. | Lending and members agree how the membership status is supplied. |
| The choice exceeds the mandate | Escalate to the authorized decision owner, ultimately the owner. | An exception allowing a loan beyond the limit. |

Hierarchy alone grants no authority over intended behavior; the root cannot invent a missing rule because its children disagree. Before escalating, a Manager asks whether knowledge or a decision is missing, which rule or freedom permits the choice, and whether binding meaning is preserved. Several valid implementations need no owner decision; several materially different policies without a selection rule may. The reviewer checks authority as well as the result: an unauthorized assumption is a finding.

### Derive the Lending mandate from its sources

The mandate is a readable view assembled from existing sources, not another authored policy ([configuration](../../design/canonical-model/manager-apply-configuration.md#the-mandate-section)). For lending:

> **Assignment:** Realize lending as code, tests and documentation, including reminders through its child Manager.
>
> **Binding basis:** Your local Definitions and connected obligations: borrowing policy, membership eligibility, copy availability and returns, with their original passages.
>
> **Decision freedom:** Internal implementation and tests within the architecture and interfaces. No additional business restrictions.
>
> **Write authority:** Your artifacts; new paths reserved through the Host; changes to other artifacts requested through the common parent.
>
> **Limits:** Artifacts only; the model, configuration and your authority stay unchanged. Caps and required reviews apply.
>
> **Escalation:** Retrieve missing context first and coordinate dependencies internally. Present missing business decisions, contradictions or needed mandate changes with a concrete proposal.

`loan-policy` owns `spec.maxOpenLoans`. The mandate refers to it instead of repeating five, so changing the canonical value to three needs no second edit.

### Resolve conflicts through applicability and authority

No blanket order such as "model over assignment over configuration" decides conflicts; it could silently discard an authorized request to change the model. Each source keeps its responsibility. A conflict exists only when applicable requirements cannot hold together in the same case, so first check scope, conditions and revisions. Determinate cases are resolved with existing authority and recorded briefly. A real conflict becomes a question naming the conflicting passages, the concrete case, the blocked work, a proposed correction at the proper source and the actor who may decide.

"Change the model to three loans and realize it" already states the owner's decision: the model operation changes the model and a new Apply follows. "Just limit the code to three" during an Apply bound to five is not implementation freedom; it needs clarification. Artifact Apply stays model-read-only in both cases.

## A question requiring the owner

If a necessary decision is missing, the responsible Manager routes a concise question through its parent. The authored M1 resolves its own rules; questions arise only from genuinely open intent, such as a later renewal extension.

### A question the owner can answer directly

| Part | Content |
| --- | --- |
| Decision | One concrete question that needs the owner's judgment. |
| Reason | The missing or conflicting passages and why the mandate cannot resolve them. |
| Recommendation and alternatives | A recommended answer, its consequences and meaningful alternatives. |
| Exact model change | The proposed wording and the passages or structured values it replaces, as a complete edit. |
| Effect on work | Which tasks wait, which continue and what accepting triggers. |

Example, for a deliberately inconsistent future draft that both preserves existing loans and caps every member at three:

**Decision needed: existing loans above the new limit.** **Recommendation A:** apply the limit when admitting new loans; existing loans stay and returns remain possible. **Proposed replacement:** "A new loan is allowed only if the resulting number of open loans is at most three. Lowering the limit leaves existing loans unchanged. Returns remain permitted regardless of the current loan limit." **Alternative B:** require existing loans to be reduced, which still needs decisions on deadlines and enforcement. **Work impact:** lending waits; independent work continues. **Responses:** accept A, elaborate B, decide differently, defer.

Accepting A authorizes exactly the displayed edit, without a second confirmation. Silence is never consent, and a required unanswered question prevents aggregate completion.

## Continue the assignment across model changes

| Level | What persists or changes |
| --- | --- |
| Assignment | The owner's request stays the same across questions and continuation runs. It grants no extra authority. |
| Apply run | Each run binds its own model, configuration and source. New intent ends the run and starts a continuation linked to it. |
| Existing work | Exact candidates and their registry can become starting material, with their findings visible. That is not approval. |
| Approval and evidence | Reviews and checks keep their original subjects. Required assessment runs again on the new basis. |

The answer is bound to the question version; the run quiesces and finishes Incomplete, awaiting the model change; the owner commits the edit; the continuation captures the new model and selects retained work; Managers reassess; guarded publication follows. Each step is reported only when observed ([journal](../../design/canonical-model/manager-apply-journal.md#questions-answers-and-model-changes)).

### A simple rule for carrying work forward

The Host checks that the retained work is traceable, that no uncontrolled writer remains, and that ownership and composition are valid. The responsible Manager then records Keep, Adapt, Replace or Unresolved with a short reason, and a fresh reviewer assesses it. Keep is a positive, reviewable claim, not a shortcut. All required reviews and checks run again: reuse the work, justify its current suitability and establish approval anew.

## Ordinary review stays inside the organization

If a new eligibility check accidentally blocks returns for suspended members, the review finding goes back to lending; correction, checks and renewed review follow within the limits. The owner can see "review found a return regression; correction in progress" without having to answer. Interruption is warranted only when a decision exceeds authority, a model ambiguity remains, work cannot converge or a limit stops progress.

### Detect stalled work and bound internal correction

After an unsuccessful attempt, the Manager records what remains unresolved, what improved or was learned, and what the next attempt does differently with an observable target. A new prompt or a promise to try again is not progress. After the second rejected submission the parent assesses that plan before the third ([execution](../../design/canonical-model/manager-apply-execution.md#admission-amendments-and-finite-correction)). If no justified attempt fits, the affected work ends honestly with its candidates and evidence.

## Share an assignment budget and protect completion work

The assignment has one budget; delegation creates no extra capacity. A completion reserve keeps room for the required reviews and checks: writing work is admitted only while the reserve stays covered ([execution](../../design/canonical-model/manager-apply-execution.md#one-ledger-with-protected-completion-capacity)). Waiting for the owner costs nothing. When capacity runs out, the run ends Incomplete with retained work, and a concrete extension request states what the capacity is for and what happens without it. Budget pressure never weakens a rule or drops a review. The earlier proposal of bounded allocations per Manager is kept in the [north star](../../design/canonical-model/manager-realization-north-star.md#simplified-for-the-first-version) as a later option.

## A compact organization status report

One screen with four stable blocks: overall state, required decisions, progress, and problems or limits, each linking to changes, evidence, the task tree and history. The heading names the assignment, model revision, run and observation time. Reports are projections of recorded work and show stale or missing observations as such.

| Displayed state | Evidence and scope |
| --- | --- |
| Locally verified (lokal geprüft) | Local assessment, independent review and checks support the exact area candidate. |
| Aggregate verified (insgesamt geprüft) | Deliveries, integration reviews, checks and the final evidence check support the exact aggregate, with no required finding open. |
| Applied (übernommen) | Guarded publication and read-back confirm the result. Not a merge, release or deployment. |
| Deployed (ausgerollt) | A separate authorized deployment with an observed outcome. Apply cannot establish it. |

Status derives from observed events, never from an implementer's claim. One question has one identity across all blocks. Counts keep their scope ("four of five areas locally verified"), not percentages. Unknown cost, progress or completion time stays unknown.

## The first usable version

The first version is a small organization that does delegated work reliably within its limits; occasional manual operation is acceptable. It delivers:

| Capability | Smallest useful delivery |
| --- | --- |
| A clear assignment | One local project, a fixed basis, explicit authority and a shared budget, started from the CLI. |
| Autonomous delegation | Package Managers, Manager plus independent reviewer, sequential writing and internal coordination. |
| Complete assessment and publication | Local and integration reviews, required checks, the final evidence check, guarded publication into the working tree and read-back; the owner's commit accepts the result. |
| Questions and continuation | A source-bound question with a proposed edit, a bound answer and a continuation run on the changed model. The owner or calling agent applies the approved edit in the first version. |
| Bounded consumption and honest failure | Finite rounds and execution time, the completion reserve, progress assessment after repeated failure and controlled stopping. |
| Inspectability and recovery | Compact status, open decisions, retained candidates and evidence, reconciliation before resuming. |

Deferred: cockpit and push notifications, further team variants, parallel writing, automatic reuse of approvals, experience storage, an operational doctor, merge, release and deployment, Government and budget optimization. Fixed input bindings, artifact ownership, independent review, publication guards and honest outcomes are never deferred: the organization must not change the model silently, approve its own work, publish stale results, lose retained work or retry endlessly.

The technical core is shown first by the two library Applies; the [working-day demonstration](working-day-demonstration-20261016.md) then tests questions, correction, interruption and budget limits in separate scenarios.

Return to [product reflections](README.md).
