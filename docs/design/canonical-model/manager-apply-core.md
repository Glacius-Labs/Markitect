# Core protocol of the first Apply

This contract owns the core of Apply:
- the step lifecycle;
- the Manager task machine and its step outcomes;
- the Orchestrator tools;
- messages and routing;
- issues;
- deliveries and composition;
- the run while it waits.

Other documents cite these rules by ID instead of restating them. The [concept](manager-apply-concept.md) owns purpose, roles and review responsibilities. The [execution contract](manager-apply-execution.md) owns budget, admission accounting, rejection counting, finding states and the run state machine. The [port contracts](manager-apply-port-contracts.md) own the Go interfaces and their values.

Status: target for the first Apply. The owner decisions behind S6, T3, T10, M2, D3 and W2 are recorded in [DEC-041](../../concepts/register.md#dec-041-core-protocol-choices). The other rules complete the frozen concept ([DEC-040](../../concepts/register.md#dec-040-concept-freeze-until-the-core-runs)) within [DEC-039](../../concepts/register.md#dec-039-deterministic-manager-process).

The owner-accepted [team extension](../../concepts/register.md#dec-043-project-teams-review-arbitration-and-product-defaults) updates T7/T8 and finding resolution below. It adds named independent reviewers and a dispute-only reviewArbiter without changing the task topology, sequential admission, artifact authority or publication gates. Current single-reviewer source is not evidence that this extension is implemented.

## Steps

- **S1** A step is one Agent turn in its own container. The Orchestrator does these things in order:
  1. Materialize the step's workspace version (D3).
  2. Start the container with the role's mounts.
  3. Dispatch the turn and serve its Orchestrator tools.
  4. Stop the container when the turn ends.
- **S2** A confirmed stop ends all activity inside the container. If termination is not observed, the step stays uncertain, and its workspace admits no further step until the uncertainty is reconciled. A confirmed stop does not settle Orchestrator-side or external effects; those are reconciled as the [execution contract](manager-apply-execution.md#admission-amendments-and-finite-correction) requires.
- **S3** A Manager's conversation continues across steps: the next step resumes the provider conversation from the retained provider home. Both pinned CLIs support this ([runtime capabilities](../../validation/runtime-capabilities-20261017.md)). If resume fails, the step starts a fresh conversation with an Orchestrator-built handoff. The handoff contains the task state, the open issues addressed to the Manager, its last report and its own unaccepted changes. Reviews always start fresh.
- **S4** After a writing step, the Orchestrator compares the workspace with the step's start version and checks ownership, reservations and protected inputs. It records the result as the Manager's new working version. Own unaccepted changes survive between steps only in these records.
- **S5** Steps run one at a time. Concurrency is a later optimization within the same rules.
- **S6** The first Apply stops every container when its step ends, and keeps no container warm between steps. It therefore needs no pause barrier and no basis change inside a running container, and every step is a charged session start. The owner chose this for the first version and kept warm retention for later, once measurements show a need (DEC-041).

## Manager tasks

Each Manager of the topology has exactly one task per run. The Orchestrator creates every task when the run starts Running. No message, report or Agent output can create, merge or remove a task.

| State | Meaning |
| --- | --- |
| `queued` | The operation has been propagated; Explore has not run yet. |
| `exploring` | The Manager's Explore step is admitted or running. |
| `explored` | The Explore result is confirmed. The task waits for the write barrier. |
| `ready` | The task has admissible work: its first work step, a received message, rework after a rejection, or integration. |
| `running` | A work step of this task is running. |
| `blocked` | The task waits for something named: a reply, a delivery, its children or an owner answer. |
| `reviewing` | A submitted candidate is under local, integration or intake review. |
| `done` | The Orchestrator accepted the task's complete responsibility. |
| `incomplete` | The task cannot finish in this run. |

**Explore and the write barrier**
- **T1** Explore runs children before parents. Each Manager has one read-only Explore step. A parent's Explore becomes admissible once all its direct children's Explore results are confirmed, and its input contains their reports.
- **T2** The Orchestrator admits no writing step until every task is explored. A task counts as explored once its Explore result is confirmed, including a T3 no-change submission, whatever its later state. If the captured source has unexplained files, the root's Explore proposes their owners, and an independent intake review assesses those proposals before the barrier opens. Accepted assignments are recorded as D6 describes. A rejected proposal leaves its file unexplained and becomes an issue addressed to the root. The root can propose again in a later work step through `assign_owner`, under a new intake review. Intake rejections count against the root's rejection limit under their own review purpose. Publication still requires conforming coverage.
- **T3** An Explore step ends with `report`, or with a `complete` no-change `submit` when the Manager finds nothing to change. A no-change submission is reviewed like any other. Its review may run before the write barrier, because nothing is written.

**Admission and step outcomes**
- **T4** One admission queue holds all admissible steps: Explore steps, review steps and work steps. The Orchestrator admits the item that became admissible first. Ties break first in favor of reply steps, so that a question is answered before the asking step's submission is reviewed ([DEC-049](../../concepts/register.md#dec-049-review-cases-are-recorded-before-the-work-they-assess)), then by Manager-path order, then by kind: Explore, review, work. Agents cannot set priorities.
- **T5** A work step ends with exactly one outcome:
  - `submit`, with a purpose (`local` or `integration`) and a scope (`limited` or `complete`). A no-change result is a `complete` submission without changes.
  - `wait`, naming what the task waits for.
- **T6** A step without a valid outcome is recorded as failed and stays charged. The task returns to the state before the step: `queued` for Explore, `ready` for work. An uncertain step goes through S2 first.

**Review**
- **T7** On a submission, the Orchestrator freezes one candidate and moves the task to `reviewing`. Every active named reviewer in the bound team assesses that subject in two steps: first a reviewer records expectations from the model, then a reviewer assesses the candidate and its evidence against those recorded expectations. Each step starts a fresh reviewer conversation (S3), so the second step receives the expectations as recorded, not through a resumed conversation. Where the configuration enables review batching, one such review covers several pending no-change submissions in the same two steps, each with its own verdict bound to its subject. Batching never removes a subject or lowers the protected review floor. Every reviewer assesses the same frozen subject independently, with its own recorded expectations and fresh conversations; one reviewer's conversation is not another's input. Collect all required assessments before deriving a submission result. Once the write barrier is open, a reviewer records the expectations before the task's writing step for a purpose, unless cases for that purpose still stand; the review of a submission against them has only its candidate step ([DEC-049](../../concepts/register.md#dec-049-review-cases-are-recorded-before-the-work-they-assess)). Unless its assessment failed (T8), a review's recorded expectations stand for the task's later submissions with the same purpose, whatever their scope. Such a submission skips the expectations step while the task's open findings, other than those the reviews against these expectations raised or resolved, and the owner's decisions are unchanged. The expectations step does not see the scope, and the Manager's steps show the standing cases ([DEC-046](../../concepts/register.md#dec-046-a-withheld-pass-keeps-its-recorded-expectations), [DEC-047](../../concepts/register.md#dec-047-recorded-review-cases-stand-until-their-inputs-change), [DEC-048](../../concepts/register.md#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose)). An explicitly disputed blocking finding invokes the configured read-only reviewArbiter under the [execution rules](manager-apply-execution.md#review-conflicts-and-arbitration); the task stays `reviewing` until that assessment completes or an owner decision blocks it. No arbiter creates another Manager task.
- **T8** The complete review group, including any required conflict clarification, ends in one of four ways:
  - **Pass:** the Orchestrator accepts the delivery (D1) only when all required named assessments are complete and valid, every blocking finding addressed to the task for this review purpose is resolved or not applicable with current independent evidence, and required checks pass. This includes findings from earlier candidates. Reviewer preference or a missing response supplies no acceptance. The arbiter may settle a disputed finding without the original reviewer personally agreeing; its recorded ruling is part of this group result, not a waiver of a model obligation. If another open blocking issue covers the subject (I3), the Orchestrator withholds acceptance instead, and the task returns to `ready` with no rejection counted.
    - After a `limited` delivery, the task returns to `ready`.
    - After a `complete` local delivery, a Manager without children is `done`. A Manager with children becomes `ready` for integration if all its children are `done`, and otherwise `blocked` on them.
    - After a `complete` integration delivery, the task is `done`.
  - **Changes requested:** the submission is rejected at most once for this review purpose, regardless of how many reviewers or findings reject it; the rejection counts as the execution contract defines, and the task returns to `ready` with the new findings. At the rejection limit, the task becomes `incomplete`.
  - **Needs decision:** the reviewer raises an owner question, and the task becomes `blocked` on it. The owner's answer starts a fresh review of the same frozen candidate, unless the answer requires a model change (W3).
  - **Failed assessment**, an invalid verdict, or a failed review step: the task stays `reviewing`, and a fresh review of the same frozen candidate is admitted. It is charged, and no rejection counts. An uncertain review step goes through S2 first.
- **T9** A Manager with children submits its `integration` candidate only when three conditions hold:
  - every child is `done`;
  - its own `local` responsibility has an accepted `complete` delivery;
  - the integration step has received the composed aggregate of its subtree.

  That step may change only the parent's own artifacts.

**Waking and reopening**
- **T10** A `blocked` task becomes `ready` when what it named arrives: the reply to its question, an accepted delivery of the named producer, the completion of its children, or the owner's answer. The exception is the answer to a needs-decision verdict, which starts a fresh review instead (T8). A message addressed to a `blocked` task also makes it `ready`, so that a waiting Manager can still answer.
- **T11** A question addressed to a `done` task admits one reply step, and the task stays `done`. A reply step ends after its `send(reply)`, and T6 does not apply to it. If that step changes no file (S4) and raises no blocking issue, nothing else happens. A blocking issue addressed to a `done` task, or a reply step that changes files, returns the task to `ready`. In that case:
  - every ancestor whose accepted integration contains that task's delivery returns to `blocked` on it;
  - its integration needs a fresh integration review once the task is `done` again;
  - counters carry on.

  The same applies during Verifying, when a correctable finding returns the run to Running.
- **T12** If no step is admissible, none is running and no task waits for the owner, the Orchestrator dispatches the lowest common parent of the blocked tasks once for that set of recorded waits. If the same waits recur after that step, those tasks become `incomplete`.

**Completion and incomplete work**
- **T13** Completion:
  - A Manager without children is `done` when its `local` complete delivery is accepted.
  - A Manager with children is `done` only when its `integration` delivery is also accepted.
  - When the root is `done`, the run moves to Verifying.
- **T14** A task becomes `incomplete` in these cases:
  - at the rejection limit (T8);
  - through an unresolvable wait (T12);
  - when required work no longer fits the budget, as the execution contract defines.

  When any task becomes `incomplete`, the Orchestrator admits no new work, lets running steps end or reconciles them, and finishes the run Incomplete with all work retained. An answer that requires a model change also finishes the run Incomplete (W3).

<a id="host-tools"></a>

## Orchestrator tools

Each role sees only its own closed catalogue. Here reviewer includes each named independent reviewer and the conflict-only reviewArbiter, with read-only candidate access. The arbiter uses review capabilities and records a bound resolution; it cannot write artifacts, alter configuration or launch another arbiter. Exact new payload/schema fields are implementation work. The [port contracts](manager-apply-port-contracts.md#session-and-communication-port) own how a tool call reaches the Orchestrator and how its answer returns.

| Tool | Roles | Effect |
| --- | --- | --- |
| `send(kind, to, body)` | Manager | Sends a message (M1 to M5). |
| `report(assessment)` | Manager | Ends an Explore step with the initial assessment (T3). |
| `submit(purpose, scope, proposal)` | Manager | Ends a step with a candidate or a no-change result (T5). |
| `wait(on)` | Manager | Ends a step and names what the task waits for (T5). |
| `read(ref)` | Manager, reviewer | Returns a Definition, artifact, delivery report or transcript by identity, and records what was supplied. |
| `reserve_path(path, form)` | Manager | Reserves a new artifact path within the creation areas and exclusive ownership. |
| `register(path, realizes, why)` | Manager | Proposes a registry change for an owned, reserved or intake-assigned path. |
| `assign_owner(path, manager, reason)` | Manager | As the common parent, decides the owner of an artifact that several descendants' Definitions require, proposes an owner for an unexplained file during intake (T2), or transfers a record between descendants (D6). |
| `request_check(check)` | Manager, reviewer | Runs an allowed project check on the current candidate and returns its receipt. |
| `record_expectations(cases)` | Reviewer | Records the expected behavior and discriminating cases from the model before the candidate is supplied. |
| `submit_review(proposal)` | Reviewer | Submits a review verdict with findings and evidence, assessing every recorded expectation case by its name. |

## Messages

A message has one of three kinds:

| Kind | Purpose |
| --- | --- |
| `question` | Asks for information, an interface change or a decision. It names an addressee: a Manager path, `parent` or `owner`. |
| `reply` | Answers one question by its ID. |
| `hint` | A parent advises a direct child, separately from the operation and the model. |

- **M1** Messages travel along tree edges, and the Orchestrator records every hop. A message cannot change the operation, a task's scope or any authority.
- **M2** When a question names a Manager, the Orchestrator forwards it along the tree path without dispatching the Managers in between. Each of them sees the forwarded message in its next step's context. The reply returns to the asker the same way. The common parent is dispatched only when it is the addressee, or under T12.
- **M3** A question addressed to `parent` goes to the direct parent. The parent then does one of four things:
  - it replies;
  - it names a better addressee anywhere in the tree, and M2 forwards the question;
  - it passes the question to its own parent;
  - it sends the question to all its relevant direct branches, or to all of them if no sound narrowing exists.

  The Orchestrator records every recipient and outstanding reply, so silence is never agreement.
- **M4** A question addressed to `owner` travels up the tree to the root as M2 forwards, and from there to the owner through the application. Other admissible work continues.
- **M5** If the asker's next work depends on a question, the asker's task is `blocked` until the reply arrives (T10).

## Issues

Findings and questions share one issue shape:
- an ID and a kind (`finding` or `question`);
- a severity (`blocking` or `advisory`);
- who raised it, and to whom it is addressed: the responsible Manager or the owner;
- its subject: the Definitions concerned and, optionally, a candidate or delivery;
- its basis and its request, which is the evidence needed or the question asked;
- a state (`open`, `resolved` or `not-applicable`) and how it was resolved.

The [journal](manager-apply-journal.md) stores findings and questions as records of this shape.

- **I1** A reviewer raises a finding. Only an independent review, including a bound arbiter ruling, with current evidence resolves it, marks it not applicable or grounds a severity correction, as the [execution contract](manager-apply-execution.md#evidence-need-survives-budget-pressure) defines. Preserve the original report and append the resolution. An unchanged closed dispute reopens only with new concrete evidence.
- **I2** A Manager or a reviewer raises a question. It is resolved by a reply from its addressee: a Manager acting within its mandate, or the owner through the application. The Orchestrator binds every reply to the question's version.
- **I3** An open blocking issue blocks acceptance of its subject and of every aggregate that contains it.
- **I4** A Definition disposition `unresolved` needs a linked open issue.
- **I5** Open issues carry into a continuation run.

## Deliveries and composition

- **D1** A delivery is an Orchestrator-accepted candidate. It consists of:
  - the owned file delta and the registry delta;
  - the aggregate version it was made on;
  - its purpose and scope;
  - its review and check evidence.
- **D2** The Orchestrator keeps one accepted aggregate tree per run, which starts as the captured source (version zero). To accept a delivery, the Orchestrator composes the delivery's owned delta onto the current aggregate. Two conditions apply:
  - every changed path is owned or reserved by the delivering Manager;
  - each such path still has, in the aggregate, the content the delivery was made on.

  Registry records follow the same rule. Exclusive ownership makes this replay safe without a common base. A failed condition is an Orchestrator error that stops the run. Composition never merges content or chooses between owners.
- **D3** Each step starts from the aggregate current at its admission, plus the Manager's own unaccepted changes replayed by the D2 rule. The step records the aggregate version it saw. A dependent task can implement against a producer's result once that result is accepted, because every later step sees it.
- **D4** A local review is renewed only when a delivery that its candidate used has changed (DEC-041). The candidate used the producers its task waited on (T5) and those its submission names as dependencies. The captured source counts as version zero of every producer. A producer has changed when it has a delivery accepted after the aggregate version that the candidate's last step saw. Any other later change leaves the local review valid, because interactions between areas are the integration review's question. An integration review always assesses the exact composed aggregate of its subtree, and it is renewed whenever that aggregate changes.
- **D5** A parent never writes child-owned paths. A correction to child content is an issue addressed to the owning child, which reopens it if it is `done` (T11).
- **D6** Ownership decisions are recorded by the Orchestrator as registry changes outside D2. These are `assign_owner` decisions, accepted intake assignments, transfers, and records passing to an ancestor. A transfer of a path is refused while another Manager has unaccepted changes on it. A decision takes effect in the aggregate when it is recorded, and later deliveries are composed under the new ownership. Each decision also becomes a blocking issue addressed to the new owner, which must register and assess the path; a `done` owner reopens under T11.

## While a run waits

- **W1** When nothing is admissible, no effect is active or uncertain, and the run waits for the owner (a question or the approval of publication), the run is Waiting. No container runs and nothing is charged.
- **W2** While a run waits, the Orchestrator process may exit (DEC-041). The [verbs](manager-apply-go-architecture.md#source-layout) `answer` and `resume` continue the run: `answer` records the owner's answer under a short exclusive control session, and `resume` reopens the run, reconciles first and continues. Recovery after an Orchestrator crash needs its own evidence before this is relied on.
- **W3** An answer that requires a model change finishes the run Incomplete, as the [execution contract](manager-apply-execution.md#assignments-and-runs) requires.

## Minimum cost

Take N Managers, of which P have children. A run without rework or batching needs at least:
- N Explore steps;
- two review steps for each of the N + P review subjects;
- one work step per Manager;
- one integration step per parent.

That is about 4N + 3P steps. For library M1 (N = 6, P = 2) it is 30. For 15 Packages (N = 16, P = 5) it is 79.
- T3 saves the work step of every Manager that needs no change.
- Review batching (T7) shares review steps between no-change subjects.
- Under S6, every step is also a session start.

## Not settled here

- **Computed scope:** the concept's [computed scope](manager-apply-concept.md#toward-computed-scope) gives Managers outside the scope no task. That conflicts with DEC-039's complete participation. The [architecture review](apply-architecture-review-20261017.md#f4-complete-participation-costs-a-lot-and-does-not-scale-yet) proposes reuse verified by the Orchestrator instead. It is a later extension and needs an owner decision.
- **Integration census:** a disposition for every explicit cross-Package reference in the integration review of its lowest common ancestor ([review F5](apply-architecture-review-20261017.md#f5-the-validation-cannot-yet-show-the-promised-benefit)). This is a proposal for after the first Package runs.
