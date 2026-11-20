# Reports and actions

**Everything that calls for a change is a report, and every change is an action** ([DEC-128](../decisions.md#dec-128-reports-and-actions-hold-everything-that-needs-a-change)). The loop is the improvement loop of ISO 9001, clause 10.2, nonconformity and corrective action, without approvals.

1. **Report.** Every signal that a change is needed becomes a report: an idea of the owner, a bug report, an incident, a customer's wish, a log, a finding of a job, or a problem with the spec itself. A report says what is wrong or wanted, with evidence, and proposes no solution. It occurred, as an error, or it is a risk.
2. **Triage.** A job sorts the report, rates it and sets the switch ([below](#triage)).
3. **Action.** A change on a branch: the spec, the artifacts or both. It includes its prevention.
4. **Landing.** By the one landing rule, the spec together with its realization ([§8](the-factory.md#the-landing-rule)).
5. **Effectiveness.** A report is closed when its action has landed, or with the reason why nothing is done. A report that recurs at the same place shows that the action did not work.

## The report

A report is a document of the language in `.markitect/reports/`, one file per report, of the report type the tool owns. `check` checks its form like any document.

| Section | Holds | Written by |
|---|---|---|
| Source | Who or what raised it: a job and its role, intake, or the owner | The job, or a conversation for the owner |
| Place | The document, artifact or path it concerns | The job |
| Finding | What is wrong or wanted, with evidence | The job |
| Occurred or risk | Whether it happened or may happen | The job |
| Violated property | One of the five properties, or correct ([§5](is-the-spec-good.md#what-is-built-and-the-questions-raised)) | The job |
| Rating | Impact and likelihood | Triage |
| Actions | The branch that answers it, or the proposals for the owner | Triage |
| Prevention | The fixed question or counter-check the action adds | The job that does the action |
| Closing | How it was closed, and why | Triage |

Agents report in their answer, as they give a verdict, and the job writes the file, so no builder or reviewer needs to write in `.markitect/`. The source "owner" is set only by a conversation, never by a job. Reports are work, so they are not part of the spec. A report about a gap or a contradiction in the spec stops building at the document it concerns until it is settled, and `status` shows that document as waiting on the report. Any other report, such as a bug report, blocks nothing: its action repairs the document.

## Intake

Signals from outside are foreign text: an issue, a ticket, a log, a mail. An intake job reads them isolated and writes only a report of fixed form. Their raw text never reaches an agent that may change something. A ticket system is read through a connector, and the report keeps its form.

## Triage

Triage sorts each report: valid, a duplicate, which attaches to the earlier report and counts there as a repetition, or needing nothing, with the reason. It rates the report by impact and likelihood, which orders the work. Then the switch:

- **The spec already says what is right:** a defect of an artifact. A build job repairs it.
- **The spec does not say it, but the answer follows from what was decided,** such as a missing link or a gap whose answer the spec implies: an architect job changes the spec, and the build job realizes it on the same branch.
- **A new decision is needed:** the report escalates to the owner ([below](#the-owners-decisions)).

## Prevention

An action does not only repair the place. It asks where the gap should have shown earlier and adds a fixed question to the project's catalog or a counter-check, so that the same gap shows earlier next time. A question or counter-check that never finds anything again goes.

## The owner's decisions

The owner is needed only for new decisions, and then as a choice ([DEC-129](../decisions.md#dec-129-the-owner-makes-only-new-decisions-and-the-system-learns-which)). A new decision is:

- something someone will rely on that nobody decided;
- a contradiction between two decisions;
- the change of a decision;
- and, beside these, a job that stops without progress, or a report that recurs at the same place.

The report escalates: the runner notifies the owner on its way, and agents prepare two or three proposals, each with its effect, none of them built. The owner chooses in a conversation, and only the chosen proposal is built, on a branch. A decision made lands as a Decision in the spec.

**The system learns which decisions the owner wants to make.** Each decision of the owner is recorded in its report with its kind, area, proposals, recommendation, choice and comment. From what the owner says, such as "don't ask me that again" or "I always want to see this", and from how they behave, which the briefing shows as a proposal, the system learns which kinds the owner keeps and which they hand over. A handed-over kind is decided with the recommendation and appears afterwards under "decided for you" in the briefing ([§10](overview.md#briefing)); a report reverses it, and what the owner does not comment on counts as agreed. Wishes from outside go to the owner until that kind is handed over. The learned rules live in `.markitect/`, not in the spec, since they concern the work, and only a conversation changes them.

**The owner's ideas** become reports with the owner as source. An architect job prepares the spec change, unbuilt. In the briefing the owner sees how it would change the spec, by comparing its branch with the main branch, and discusses it or takes it. Only then does the build job build it. The owner states their own intent here and reads no artifacts.

## A change past the spec

A decision made in passing, in any session, is caught by its effect, not by recording the chat ([DEC-130](../decisions.md#dec-130-a-change-past-the-spec-is-back-filled)). A change past the spec shows in `status` as changed documents or unexplained files, on a branch before it lands or on the main branch after a direct push. Where it came from is the switch:

- **From a job,** the build job brings the artifacts back to the spec.
- **From a conversation,** by a person or by Claude for them, it becomes a report "past the spec", with the diff and the commit messages as evidence, and the build job waits for triage.
  - **It fills a gap,** where the spec says nothing: an architect job writes the back-fill, as a Decision with the reason taken from the commits where it was a choice; the counter-checker derives counter-checks from the new text, the reviewer confirms, and only then does the branch land.
  - **It contradicts the spec,** where a document says otherwise or earlier counter-checks fail: a new decision for the owner, to keep it and change the spec, or to have the build job revert it.

So no slip silently becomes intent. Loosely like drift in Terraform, a change is kept by changing the spec, or reverted. A back-fill appears under "decided for you", and a report reverses it. The work of the architect is that of a first spec of an existing project, for one diff. A decision that changes no file, a no or a trade-off, leaves no drift: only the conversation that makes it records it, by changing the spec.
