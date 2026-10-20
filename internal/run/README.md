# Run

This package is the Orchestrator of Apply and Extract
([DEC-064](../../docs/concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams),
[accepted workflow](../../docs/design/canonical-model/manager-apply-workflow.md)). It runs
**one recursive Pass process** for every purpose: Plan, Prepare, Implement and Verify for
Apply, and Extract's own work purposes. Only the mandate, the effects, the purpose
payload and the acceptance differ; every purpose uses the same stages, states, result
envelope, routing and gates. It does no file or process I/O itself: everything goes
through the ports in `Config`: `Agents` (one Agent turn), `Workspaces` (volumes),
`Checks` (required checks), `Owner` (the AI that invoked Markitect, optional) and
`Evidence` (the append-only record). Callers compose Passes and publish; this package
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

A `PassSpec` declares `Purpose`, `Mandate`, `Writes` (owned writes allowed), `Review`
(fresh independent reviewers on an immutable candidate, with `Guidance` on what a genuine
finding is), `Plans` (the plan payload; accepted plans' dependencies become start
conditions of later Passes of the named purpose), `Corrections` (the work purposes a
finding may name, the last one the default) and `ArtifactFindingsAdvisory` (Extract's
model review).

## The base process

Every Manager, Root and leaves alike, goes through the stages of `pass.go`:

1. **start**: waiting for the parent's accepted basis (Root starts at once).
2. **blocked**: released by the parent's basis, but the dependencies of its accepted plan
   for this purpose are not accepted yet. It may answer routed questions in
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
unchanged wait starts no turn. A wait on another Manager that closes a cycle with the
accepted plans' dependencies and the parent-first and integration order could never be
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
child's `needs_correction` rests on an open finding, so a finding its raiser closes later
no longer holds the Pass back. `PassResult.Corrections` names the correction purposes of
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
finding that cites it (a violation), a reasoned `expectation_error` for a failed assertion,
or a `rerun` that replaces an `error` by a passing exact rerun of it. A failed assertion is
never disposed of by running it again; an error without a passing rerun leaves its claim
without evidence, an open obligation with a genuine finding. Dispositions take effect with
their submission's acceptance, and an expectation-error or rerun disposition falls when
that submission is reopened or invalidated; a parent's acceptance is the independent
assessment of a child's own disproof. Root's disproof of its own receipt in the same Pass
awaits a fresh review: the Pass verdict needs correction and `Reassess` runs a new Verify of
the unchanged candidate, whose Root reviewer confirms it or raises a finding. A done claim
with an undisposed adverse receipt of its own and a satisfied result with one within the
responsibility are refused.

## Tools

`report` (above), `send` (question, answer or information to the direct parent or a child;
Root also to `user`; answers travel back the same edge, forwarded exchanges correlate with
`about`), `escalate` (a classified problem as a question to the parent, Root's to the user),
`look_up` (Definitions, owners, the derived registry view, states, plans, bases, findings,
checks, probe receipts, questions and the method guidance of `methods.go`), `check`
(advisory File check of the caller's live area composed with the accepted stand in write
Passes, or a probe when the project declares a check) and `delegate` (a bounded task for a
configured helper role).
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
routing and the owner channel; `helpers.go` delegation; `review.go` candidates and checks;
`lookup.go` the look-up and check tools; `probes.go` probes, receipts, evidence binding and
dispositions; `methods.go` the method guidance; `turn.go`, `tools.go` and `prompts.go` turns, tool
schemas and prompts. Tests use scripted fakes for every port.
