# Core protocol of the first Apply

This contract owns the core of Apply:
- the step lifecycle;
- the Manager task machine and its step outcomes;
- messages and routing;
- issues;
- deliveries and composition;
- the Host process while a run waits.

Other documents cite these rules by ID instead of restating them. The [concept](manager-apply-concept.md) owns purpose, roles and review responsibilities. The [execution contract](manager-apply-execution.md) owns budget, admission accounting, rejection counting, finding states and the run state machine. The [port contracts](manager-apply-port-contracts.md) own the Go interfaces and their values.

Status: target for the first Apply. The owner selected the choices behind T3, T10, M2 and D3 in [DEC-041](../../concepts/register.md#dec-041-core-protocol-choices). The other rules complete the frozen concept ([DEC-040](../../concepts/register.md#dec-040-concept-freeze-until-the-core-runs)) within [DEC-039](../../concepts/register.md#dec-039-deterministic-manager-process).

## Steps

- **S1** A step is one Agent turn in its own container. The Host does these things in order:
  1. Materialize the step's workspace version (D3).
  2. Start the container with the role's mounts.
  3. Dispatch the turn and serve its Host tools.
  4. Stop the container when the turn ends.
- **S2** A confirmed stop ends all activity inside the container. If termination is not observed, the step stays uncertain, and its workspace admits no further step until the uncertainty is reconciled. A confirmed stop does not settle Host-side or external effects. Those are reconciled as the [execution contract](manager-apply-execution.md#admission-amendments-and-finite-correction) requires.
- **S3** A Manager's conversation continues across steps: the next step resumes the provider conversation from the retained provider home. Both pinned CLIs support this ([runtime capabilities](../../validation/runtime-capabilities-20261017.md)). If resume fails, the step starts a fresh conversation with a Host-built handoff. The handoff contains the task state, the open issues addressed to the Manager, its last report and its own unaccepted changes. Reviews always start fresh.
- **S4** After a writing step, the Host compares the workspace with the step's start version and checks ownership, reservations and protected inputs. It records the result as the Manager's new working version. Own unaccepted changes survive between steps only in these records.
- **S5** Steps run one at a time. Concurrency is a later optimization within the same rules.
- **S6** There is no warm retention, no pause barrier and no basis change inside a running container. The idle policy of the execution contract therefore has a warm limit of zero, and every step is a charged session start.

## Manager tasks

Each Manager of the topology has exactly one task per run. The Host creates every task when the run starts Running. No message, report or Agent output can create, merge or remove a task.

| State | Meaning |
| --- | --- |
| `queued` | The operation has been propagated; Explore has not run yet. |
| `exploring` | The Manager's Explore step is admitted or running. |
| `explored` | The Explore result is confirmed. The task waits for the write barrier. |
| `ready` | The task has admissible work: its first work step, a received message that needs action, rework after a rejection, or integration. |
| `running` | A work step of this task is running. |
| `blocked` | The task waits for something named: a reply, a delivery, its children or an owner answer. |
| `reviewing` | A submitted candidate is under local or integration review. |
| `done` | The Host accepted the task's complete responsibility. |
| `incomplete` | The task cannot finish in this run. |

- **T1** Explore runs children before parents, with siblings in stable Manager-path order. Each Manager has one read-only Explore step. A parent's Explore input contains its direct children's Explore reports.
- **T2** The Host admits no writing step until every task is `explored` or later.
- **T3** An Explore step ends with `report`, or with a `complete` no-change `submit` when the Manager finds nothing to change. A no-change submission is reviewed like any other, and that review may run before the write barrier because nothing is written.
- **T4** The Host admits the oldest `ready` task first and breaks ties by Manager-path order. Agents cannot set priorities.
- **T5** A work step ends with exactly one outcome:
  - `submit`, with a purpose (`local` or `integration`) and a scope (`limited` or `complete`). A no-change result is a `complete` submission without changes.
  - `wait`, naming what the task waits for.

  A step without a valid outcome is recorded as failed. The task returns to `ready`, and the step stays charged.
- **T6** On a submission, the Host freezes the candidate and moves the task to `reviewing`. A fresh reviewer conversation assesses it in two steps: first expectations from the model, then the candidate and its evidence. Where the configuration enables review batching, one such review covers several pending no-change submissions in the same two steps, each with its own verdict bound to its subject. Batching never removes a subject or lowers the protected review floor.
- **T7** A review ends in one of three ways:
  - **Pass:** the Host accepts the delivery (D2). After a `limited` delivery, the task returns to `ready` or `blocked`.
  - **Changes requested:** the rejection counts as the execution contract defines, and the task returns to `ready` with the new findings. At the rejection limit, the task becomes `incomplete`.
  - **Needs decision:** the reviewer raises an owner question, and the task becomes `blocked` on it.
- **T8** A Manager with children submits its `integration` candidate only when three conditions hold: every child is `done`, the Manager's `local` responsibility has an accepted `complete` delivery, and the integration step has received the composed aggregate of its subtree. That step may change only the parent's own artifacts.
- **T9** Completion:
  - A Manager without children is `done` when its `local` complete delivery is accepted.
  - A Manager with children is also `done` only when its `integration` delivery is accepted.
  - When the root is `done`, the run moves to Verifying.
- **T10** When a task becomes `incomplete`, the Host admits no new work. It lets running steps end or reconciles them, and finishes the run Incomplete with all work retained.

## Host tools

Each role sees only its own closed catalogue. The [port contracts](manager-apply-port-contracts.md#session-and-communication-port) own how a tool call reaches the Host and how its answer returns.

| Tool | Roles | Effect |
| --- | --- | --- |
| `send(kind, to, body)` | Manager | Sends a message (M1–M5). |
| `report(assessment)` | Manager | Ends an Explore step with the initial assessment (T3). |
| `submit(purpose, scope, proposal)` | Manager | Ends a step with a candidate or a no-change result (T5). |
| `wait(on)` | Manager | Ends a step and names what the task waits for (T5). |
| `read(ref)` | Manager, reviewer | Returns a Definition, artifact, delivery report or transcript by identity and records what was supplied. |
| `reserve_path(path, form)` | Manager | Reserves a new artifact path within the creation areas and exclusive ownership. |
| `register(path, realizes, why)` | Manager | Proposes a registry change for an owned, reserved or intake-assigned path. |
| `assign_owner(path, manager, reason)` | Manager | As the common parent, decides the owner of an artifact that several descendants' Definitions require. It also assigns an unexplained file during intake, or transfers a record between descendants. |
| `request_check(check)` | Manager, reviewer | Runs an allowed project check on the current candidate and returns its receipt. |
| `record_expectations(cases)` | Reviewer | Records the expected behavior and discriminating cases from the model before the candidate is supplied. |
| `submit_review(proposal)` | Reviewer | Submits a review verdict with findings and evidence. |

## Messages

A message has one of three kinds:

| Kind | Purpose |
| --- | --- |
| `question` | Asks for information, an interface change or a decision. It names an addressee: a Manager path, `parent` or `owner`. |
| `reply` | Answers one question by its ID. |
| `hint` | A parent advises a direct child, separately from the operation and the model. |

- **M1** Messages travel along tree edges, and the Host records every hop. A message cannot change the operation, a task's scope or any authority.
- **M2** When a question names a Manager, the Host forwards it along the tree path without dispatching the Managers in between. Each of them sees the forwarded message in its next step's context. The common parent is dispatched only when it is the addressee, or when a dispute or a circular wait reaches it.
- **M3** A question addressed to `parent` goes to the direct parent. The parent replies or names a better addressee in its subtree, and the Host forwards the question there. This is the bounded inquiry of the concept.
- **M4** A question addressed to `owner` goes to the owner through the application. Other admissible work continues.
- **M5** If the asker's next work depends on a question, the asker's task is `blocked` until a reply arrives.

## Issues

Findings and questions share one issue shape:
- an ID and a kind (`finding` or `question`);
- a severity (`blocking` or `advisory`);
- who raised it, and to whom it is addressed: the responsible Manager or the owner;
- its subject: the Definitions concerned and, optionally, a candidate or delivery;
- its basis and its request, which is the evidence needed or the question asked;
- a state (`open`, `resolved` or `not-applicable`) and how it was resolved.

The [journal](manager-apply-journal.md) stores findings and questions as records of this shape.

- **I1** A reviewer raises a finding. Only an independent review with current evidence resolves it or marks it not applicable, as the [execution contract](manager-apply-execution.md#evidence-need-survives-budget-pressure) defines.
- **I2** A Manager or a reviewer raises a question. It is resolved by a reply from its addressee: a Manager acting within its mandate, or the owner through the application. The Host binds every reply to the question's version.
- **I3** An open blocking issue blocks acceptance of its subject and of every aggregate that contains it.
- **I4** A Definition disposition `unresolved` needs a linked open issue.
- **I5** Open issues carry into a continuation run.

## Deliveries and composition

- **D1** A delivery is a Host-accepted candidate. It consists of:
  - the owned file delta and the registry delta;
  - the aggregate version it was made on;
  - its purpose and scope;
  - its review and check evidence.
- **D2** The Host keeps one accepted aggregate tree per run, which starts as the captured source. To accept a delivery, the Host composes the delivery's owned delta onto the current aggregate. Two conditions apply:
  - every changed path is owned or reserved by the delivering Manager;
  - each such path still has, in the aggregate, the content the delivery was made on.

  Registry records follow the same rule. Exclusive ownership makes this replay safe without a common base. A failed condition is a Host error that stops the run. Composition never merges content or chooses between owners.
- **D3** Each step starts from the aggregate current at its admission, plus the Manager's own unaccepted changes replayed by the D2 rule. The step records the aggregate version it saw. A dependent task can implement against a producer's result once that result is accepted, because every later step sees it.
- **D4** A local review stays valid when its delivery is composed onto a newer aggregate. The exception: the review is renewed when the aggregate now holds a different version of a delivery whose result the subject used, according to its step records. An integration review always assesses the exact composed aggregate of its subtree.
- **D5** A parent never writes child-owned paths. A correction to child content is an issue addressed to the owning child.

## While a run waits

- **W1** When no task is admissible and at least one task is `blocked` on an owner question or approval, the run is Waiting. No container runs, and nothing is charged.
- **W2** While a run waits, the Host process may exit. The [verbs](manager-apply-go-architecture.md#source-layout) `answer` and `resume` continue the run: `answer` records the owner's answer under a short exclusive control session, and `resume` reopens the run, reconciles first and continues. Recovery after a Host crash needs its own evidence before this is relied on.
- **W3** An answer that requires a model change finishes the run Incomplete, as the [execution contract](manager-apply-execution.md#assignments-and-runs) requires.

## Minimum cost

Take N Managers, of which P have children. A run without rework needs at least:
- N Explore steps;
- two review steps for each of the N + P review subjects;
- one work step per Manager;
- one integration step per parent.

That is about 4N + 3P steps. For library M1 (N = 6, P = 2) it is 30. For 15 Packages (N = 16, P = 5) it is 79. T3 saves the work step of every Manager that needs no change. Under S6, every step is also a session start.

## Not settled here

- **Computed scope:** the concept's [computed scope](manager-apply-concept.md#toward-computed-scope) gives Managers outside the scope no task. That conflicts with DEC-039's complete participation. The [architecture review](apply-architecture-review-20261017.md#f4-complete-participation-costs-a-lot-and-does-not-scale-yet) proposes reuse verified by the Host instead. It is a later extension and needs an owner decision.
- **Integration census:** a disposition for every explicit cross-Package reference in the integration review of its lowest common ancestor ([review F5](apply-architecture-review-20261017.md#f5-the-validation-cannot-yet-show-the-promised-benefit)). This is a proposal for after the first Package runs.
