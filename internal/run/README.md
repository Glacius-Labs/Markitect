# Run

This package is the Orchestrator of Apply and Extract
([DEC-064](../../docs/concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams),
[accepted workflow](../../docs/design/canonical-model/manager-apply-workflow.md)). It runs
**one recursive Pass process** for every purpose: Apply's work Passes (Implement in its
`direct` strategy; Plan, Prepare and Implement in `staged`) and Verify, and Extract's own
work purposes. Only the mandate, the effects, the purpose
payload and the acceptance differ; every purpose uses the same stages, states, result
envelope, routing and gates. It does no file or process I/O itself: everything goes
through the ports in `Config`: `Agents` (one Agent turn), `Workspaces` (volumes),
`Checks` (required checks) and `Evidence` (the append-only record). A run is closed:
nobody outside the Manager tree answers during it. Callers compose Passes and publish; this package
never publishes and never commits.

## API

`Open(ctx, Config)` validates the configuration and materializes the shared work volume
`<RunID>-work` from the start stand. `Pass(PassSpec)` runs one Pass with a new identity
`p<N>` over the whole required Manager tree, then its exit gates, and returns the sealed
`PassRecord` and verdict (`satisfied` or `needs_correction`), or `Stopped` when the run
ended. `Corrections()` names the correction purposes of the open genuine findings;
`Reassess()` says whether only fresh disputes keep the last review from being clean.
`Finish(reason)` ends the run: `Verified` only when the last Pass was a clean review of
one candidate, `Failed` for a definite rejection, `Incomplete` otherwise. After Finish the
caller owns the work volume.

A turn's prompt carries the standing context (the Manager's or reviewer's Brief, contacts
and owner request) when its conversation is new, or, together with the current Pass
assignment again, when the previous turn of that conversation reported a compaction
(`TurnResult.Compacted`), came back with another conversation than it resumed or failed
after it had carried the standing context; otherwise it relies on the conversation's
history. `TurnRecord.Context` and `turn.start` record which (`brief`, `rehydrated`,
`history`) with the prompt's digest and size. `look_up brief` returns the standing context
with the current assignment (a helper's: its own turn's prompt) and `look_up definition` the
bound model's Definition; `lookup.brief` and `lookup.definition` record each answer's digest.
An Agents error wrapping `ErrContextLimit` ends the run Incomplete without repeating the
turn. `Config.Binding` is what the caller bound the run to besides
the Config's own inputs, such as an Apply's strategy, effective settings and build:
`run.start` records it as given, and the Orchestrator chooses nothing by it.

A `PassSpec` declares `Purpose`, `Mandate`, `Writes` (owned writes allowed), `Review`
(fresh independent reviewers on an immutable candidate, with `Guidance` on what a genuine
finding is), `Plans` (the plan payload; accepted plans' dependencies become start
conditions of later Passes of the named purpose; Apply's `direct` declares no planning Pass), `Corrections` (the work purposes a
finding may name, the last one the default) and `ArtifactFindingsAdvisory` (Extract's
model review).

## The base process

Every Manager, Root and leaves alike, goes through the stages of `pass.go`:

1. **start**: waiting for the parent's accepted basis (Root starts at once).
2. **blocked**: released by the parent's basis, but the dependencies of its accepted plan
   for this purpose, or after a reset the submissions its basis was bound to, are not
   accepted yet. It may answer routed questions in
   communication-only turns with a read-only project; it cannot submit, report progress,
   decide or release children.
3. **own**: works on its own contribution and submits its **basis** with `report`.
4. **basis**: the basis is checked technically after the turn ended (writers stopped):
   the submitter's stable area is composed with the accepted stand and File checked; a
   finding addressed to the submitter returns it to the owner (the whole stand is
   checked at the Pass's exit). The direct parent then accepts or returns it with
   `report kind accept|return`; Root's basis is accepted by the Orchestrator against the
   declared Pass requirements.
5. **delegated**: the basis is accepted and the children are released; the Manager's own
   area is read-only, it decides its children's proposals and answers questions. A
   submitted child proposal wakes it before the child is Done.
6. **integrate**: every child's result is accepted (at once for a leaf); the Manager
   integrates and submits its complete **result** with every own obligation covered and a
   verdict.
7. **result** and **done**: the result is checked and decided like the basis; Done means
   the complete responsibility is accepted.

A returned proposal goes back to its owner within the Pass. `reopen` lets a parent reopen
a Done child before closure; `revise` lets a Manager revise its accepted basis, which
resets its children and declared dependents explicitly and unsettles results built on
it. The Pass closes when every Manager is Done and nothing runs; its exit gates harvest
the quiescent shared tree, require it to equal the accepted stand and run the Model check
and the File check; any failure ends the run Failed without a repair Pass.

States per Pass are `ready`, `working`, `waiting` and `done`, recorded as
`manager.state_changed` with the stage and what a Manager waits for. A turn that ends
without the report its work needs gets a note and counts as empty; so does a turn whose
only report repeats an earlier progress summary of the Pass, which is not continued
automatically. Repeated empty turns, repeated failed turns and a turn without an observed
container stop end the run Incomplete. A wait that nothing can resolve ends it Incomplete
with the named waits.

`report kind wait` names an own open question, a child, or the accepted basis or result
(`stage`) of any other Manager of the Pass. Only an actual event wakes a waiting Manager:
the accepted submission it named, an answer, a message or a decision it must make; an
unchanged wait starts no turn. In a Pass with `BindWaits` (Implement of Apply's `direct`,
which has no planning Pass before it), the accepted submission a wait named, or one already accepted
when the wait is reported (recorded as report kind `bind`), becomes a **bound dependency**
of the event the waiter works toward, its basis or its result (`dependency.bound`, with
proposal and stand). Only Managers outside the waiter's own line bind: ancestors and
descendants are already related by the parent-first and integration order. A binding that
would close a cycle among all start, basis and result events, whether or not they already
happened, is refused or skipped (`dependency.unbound`). A revision or reopening of a bound
submission resets a waiter whose basis built on it, like a declared consumer, and its
restart waits for the newly accepted version (`rebound`). A waiter whose result alone built
on it keeps its basis and its children's work and integrates again: its submitted result
is invalidated, an accepted one reopened with its consumers, and the submission of a turn
running meanwhile withdrawn; that turn's helper delegations stand. Verify and Extract bind nothing. A wait
on another Manager that closes a cycle with the accepted plans' dependencies, the bindings
and the parent-first and integration order could never be
satisfied, so it is refused with the cycle named; the Manager submits first what does not
need it or asks its parent, and a Pass that still cannot advance ends as stalled. Every turn records its `mode`: `work`, `communication` (blocked), `coordination`
(decisions and answers after the basis), `review` or `helper`.

## Envelope and verdicts

`report kind basis|result` carries the shared envelope: summary, obligation coverage
(`done`, `preserved`, `not_applicable`, `open`, with notes and evidence items), the purpose
payload (`plan` in a planning Pass; `artifacts` and `evidence` otherwise), findings,
closures, disputes, dispositions of adverse probe receipts and, for a result, the verdict. A result covers every own obligation; a verdict contradicting
the open genuine findings within the responsibility is refused; a result is refused while
the Manager's own question is unanswered. Plans are validated (approach, method and
evidence per obligation, owners of inputs and outputs, dependencies without a cycle with
the parent-first order).

A Pass verdict is `satisfied` when Root's integrated result is satisfied and no genuine
finding of the kind the Pass judges is open; a review also needs every required check
passing on its candidate and every adverse probe receipt on it disposed of independently. In every Pass no accepted result may leave an own obligation
`open`. A
result's `satisfied` is refused while a finding raised within its responsibility stays
open, and its `needs_correction` rests on an open finding. A finding an ancestor raised is
closed only in that ancestor's later report, so it does not refuse a child's `satisfied`;
it holds the Pass back until its raiser closes it, and a child's `needs_correction` that
rested on it no longer does once it is closed. `PassResult.Corrections` names the correction purposes of
the open genuine findings that kind of Pass judges: work findings after a work Pass, all
after a review.

## Findings

A finding has one owner (the raiser or a descendant) and one raiser. A submission's
findings, closures and disputes take effect when it is accepted and are then never
withdrawn. A review finding is bound to its candidate; it closes only through a later
review submission of its raising responsibility, with reason and evidence: `resolved` on
a changed candidate, or `disproved` on the unchanged one. Another reviewer whose
responsibility holds its raiser or owner may dispute it once per candidate with reason and
evidence; when only fresh
disputes keep a review from being clean, `Reassess` is true and the caller runs a new
complete Verify on the same candidate. Failed required checks are Orchestrator findings
that hold for their candidate and cannot be disputed. Work-Pass findings close through a
later work Pass of their raiser.

## Probe receipts and evidence

`check` with a `probe` runs one behavioral probe file through the `Checks` port with the
execution profile of a required check: the profile's image, executable, fixed arguments,
environment and limits, the caller's arguments, no network and a disposable copy. The probe
names its obligation (own or a descendant's), claim and expected outcome; they are recorded
(`probe.requested`) before it runs. It runs on a stable stand: the candidate in a review,
the accepted stand in a Pass without writes, a child's checked proposal, or in a Pass with
writes the caller's live area composed with the accepted stand. The file is added to that
stand in the check's directory and never replaces a project file. The Orchestrator keeps
the receipt (`probe.receipt`, `Outcome.Probes`): the binding, the probe bytes and digest,
the invocation and what the Runtime observed. Its outcome is the probe's own assertion
verdict: `passed` for an observed zero exit of exactly that request, `failed` for an
observed nonzero exit of the probe's own assertions, `error` for everything else (not
started, timed out, uncertain, mismatched, not executable with exit 126 or 127, or killed
by a signal with exit 128 or above), which never counts as a pass or as an expected
refusal. `probe.rerun` repeats the exact probe of an error receipt on the same candidate.
A probe runs only while its turn keeps a fifth of its timeout for the report, and it ends
before that point; late in a turn it is refused rather than letting the timeout fail the
turn.
A helper runs no probe; `look_up topic probe` returns a whole receipt.

A coverage entry's evidence items each support one claim: a receipt (the caller's own,
for that Definition, passed), which brings its own claim and supports one item of a
submission, or an inspected source file with claim, location and reasoning. In a review
every `done` or `preserved` obligation needs at least one item; a suite total is no item,
and whether the items cover the Definition's claims is the parent's judgment. Items bind to the stand of the submission: the candidate or the unchanged accepted
stand at once, in a Pass with writes the checked stand after the writers stopped, where a
receipt of another stand returns the submission to its owner. A parent's proposal view
shows the child's evidence and every probe it ran in the Pass, adverse ones included.

An adverse receipt on a review's candidate blocks until it is disposed of: a genuine
finding that cites it (a violation), a reasoned `expectation_error` when the probe itself was
wrong (a failed assertion, or an error within the probe's own execution, such as an exceeded
time limit, together with a passing receipt of the same obligation on the candidate),
or a `rerun` that replaces an `error` by a passing exact rerun of it (a repetition always
repeats the original probe, and one passing repetition also disposes of the repetitions
that erred). A failed assertion is
never disposed of by running it again; an error outside the probe's own execution (not
started or uncertain) only by a passing exact rerun; and an error that points at the
candidate, such as a hang, is a genuine finding. One disposition of an error also disposes
of the erring repetitions of its probe within the responsibility. Dispositions take effect with
their submission's acceptance, and an expectation-error or rerun disposition falls when
that submission is reopened or invalidated; a parent's acceptance is the independent
assessment of a child's own disproof. The reviewer of an obligation's owner may also judge
an ancestor's adverse receipt about that obligation, as an independent context, so Root
returns or reopens a child's proposal for its own failed probe of the child's claim.
Root's disproof of its own receipt in the same Pass awaits a fresh review: the Pass verdict needs correction and `Reassess` runs a new Verify of
the unchanged candidate, whose Root reviewer confirms it or raises a finding. A done claim
with an undisposed adverse receipt of its own and a satisfied result with one within the
responsibility are refused.

## Model assessment

A review Pass with `Assesses` is the Model assessment of the bound model before any
realization: its fresh read-only reviewers run the same stages, but no gate or required check
runs on its candidate and no probe is offered, and its clean seal verifies no candidate of its
own (a candidate a clean Verify verified just before stays verified). A seal that needs
correction with a confirmed finding, open, genuine, raised by the team and not disputed in
the Pass, ends the run Failed with `ModelAssessmentReason`, since no Pass of the run can
correct the model; findings disputed in the Pass are left to `Reassess`. The caller supplies
the mandate, the guidance and the correction purposes, which name the classes of a model
problem.

## Tools

`report` (above), `send` (question, answer or information to the direct parent or a child;
answers travel back the same edge, forwarded exchanges correlate with `about`; Root's message
beyond the tree is refused with the route that applies), `escalate` (a classified problem as
a question to the parent; Root has nobody above it and reports a model problem instead),
`look_up` (Definitions, owners, the derived registry view, states, plans, bases, findings,
checks, probe receipts, questions and the method guidance of `methods.go`), `check`
(advisory File check of the caller's live area composed with the accepted stand in write
Passes, or a probe when the project declares a check; not in a Model assessment) and
`delegate` (a bounded task for a configured helper role).
Helpers run one turn each in their own container with their own home, read the whole
stand, write only their delegated, existing, disjoint subset of the Manager's area and
report to their Manager only; while they work their subset is read-only for the Manager.

## Turns, homes and mounts

Work turns of a Manager continue one conversation across the work Passes (`<RunID>-<slug>-m`);
every review Pass has fresh reviewer homes (`<RunID>-<pass>-<slug>-r`) and never resumes
across Passes. A turn may write only in a Pass with writes while its Manager works on its
own basis or result: its `WritePlan` overlays directories and exact files of `/work` over a
read-only base; every other turn is read-only. Turn identities are
`<RunID>-<pass>-<slug>-<m|r|h><n>`.

## Files

`run.go` and `agents.go` hold the contract, `workspace.go` and `checks.go` the
consumer-owned ports. `engine.go` validates and ends the run; `pass.go` holds the base
process; `proposals.go` the `report` tool, envelopes and technical checks; `messages.go`
routing along the tree; `helpers.go` delegation; `review.go` candidates and checks;
`lookup.go` the look-up and check tools; `probes.go` probes, receipts, evidence binding and
dispositions; `methods.go` the method guidance; `turn.go`, `tools.go` and `prompts.go` turns, tool
schemas and prompts. Tests use scripted fakes for every port.
