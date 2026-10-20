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
without the report its work needs gets a note and counts as empty; repeated empty turns,
repeated failed turns and a turn without an observed container stop end the run
Incomplete. A wait that nothing can resolve ends it Incomplete with the named waits.

## Envelope and verdicts

`report kind basis|result` carries the shared envelope: summary, obligation coverage
(`done`, `preserved`, `not_applicable`, `open`, with notes), the purpose payload (`plan`
in a planning Pass; `artifacts` and `evidence` otherwise), findings, closures, disputes and,
for a result, the verdict. A result covers every own obligation; a verdict contradicting
the open genuine findings within the responsibility is refused; a result is refused while
the Manager's own question is unanswered. Plans are validated (approach, method and
evidence per obligation, owners of inputs and outputs, dependencies without a cycle with
the parent-first order).

A Pass verdict is `satisfied` when every result is satisfied and no genuine finding of the
kind the Pass judges is open; a review also needs every required check passing on its
candidate.

## Findings

A finding has one owner (the raiser or a descendant) and one raiser. Findings are recorded
when submitted and cannot be withdrawn. A review finding is bound to its candidate; it
closes only through a later review submission of its raising responsibility: `resolved` on
a changed candidate, or `disproved` with evidence on the unchanged one. An ancestor
reviewer may dispute it once per candidate with reason and evidence; when only fresh
disputes keep a review from being clean, `Reassess` is true and the caller runs a new
complete Verify on the same candidate. Failed required checks are Orchestrator findings
that hold for their candidate and cannot be disputed. Work-Pass findings close through a
later work Pass of their raiser.

## Tools

`report` (above), `send` (question, answer or information to the direct parent or a child;
Root also to `user`; answers travel back the same edge, forwarded exchanges correlate with
`about`), `escalate` (a classified problem as a question to the parent, Root's to the user),
`look_up` (Definitions, owners, the derived registry view, states, plans, bases, findings,
checks, questions and the method guidance of `methods.go`), `check` (advisory File check of the caller's live area composed with the
accepted stand, write Passes) and `delegate` (a bounded task for a configured helper role).
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
`lookup.go` the look-up and check tools; `methods.go` the method guidance; `turn.go`, `tools.go` and `prompts.go` turns, tool
schemas and prompts. Tests use scripted fakes for every port.
