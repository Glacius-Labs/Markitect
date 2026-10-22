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

Every required Manager receives its assignment at Pass start. Only actual declared
or previously bound prerequisites put it in **blocked**, with read-only communication
and no productive submission. Independent children can work while a parent is blocked.

In **own**, a Manager works on its contribution and may submit its complete result
as soon as its children are accepted. No universal basis or parent preapproval is
required. An optional **basis** supplies a contract another Manager actually needs;
its stopped Turn is technically checked and its parent accepts or returns it. That
acceptance releases only declared consumers. In **delegated**, the Manager handles
child proposals; **integrate** follows accepted child results. A result is checked and
decided by its parent (Root by the Orchestrator); **done** means the whole responsibility
is accepted. A child proposal wakes its parent before the child is Done.

A returned proposal goes back to its owner within the Pass. `reopen` lets a parent reopen
a Done child before closure; `revise` lets a Manager revise its accepted basis, which
resets its actual declared or bound consumers explicitly and unsettles results built on
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
proposal and stand). An explicitly needed ancestor contract binds too. Descendant results already
participate in mandatory parent integration and need no duplicate binding. A binding that
would close a cycle among all start, basis and result events, whether or not they already
happened, is refused or skipped (`dependency.unbound`). A revision or reopening of a bound
submission resets a waiter whose basis built on it, like a declared consumer, and its
restart waits for the newly accepted version (`rebound`). A waiter whose result alone built
on it keeps its basis and its children's work and integrates again: its submitted result
is invalidated, an accepted one reopened with its consumers, and the submission of a turn
running meanwhile withdrawn; that turn's helper delegations stand. Verify and Extract bind nothing. A wait
on another Manager that closes a cycle with the accepted plans' dependencies, the bindings
and the integration order could never be
satisfied, so it is refused with the cycle named; the Manager submits first what does not
need it or asks its parent, and a Pass that still cannot advance ends as stalled. Every turn records its `mode`: `work`, `communication` (blocked), `coordination`
(decisions and answers after the basis), `review` or `helper`.

## Envelope and verdicts

`report kind basis|result` carries a concise summary, shared `support`, concrete
`unresolved` gaps, the purpose payload (`plan` in planning), findings, closures,
disputes, adverse-receipt dispositions and, for a result, the verdict. The Host adds
the exact obligation `scope` and technical identities; it does not expand a receipt
into satisfied Definitions. Optional `obligations` entries localize detailed claims,
gaps or inapplicability. A verdict contradicting
the open genuine findings within the responsibility is refused; a result is refused while
the Manager's own question is unanswered. Plans are validated (approach, method and
evidence, owners of inputs and outputs, dependencies without a cycle with
the integration order).

Implement and artifact reviewers assess material conditions, exceptions and interactions
from the bound authority, explain relevant evidence and gaps, and challenge valid inputs
wrongly refused and invalid inputs wrongly accepted. No `conditions` field or
per-obligation mapping artifact is mandatory. Structural evidence validation does not
establish semantic completeness. Extract retains fixed artifacts/documented intent as
authority; its proposed model is work product.

Every parent, including Root without own Definitions, assesses composed workflows,
interfaces and shared invariants in its concise result. Relevant child receipts may be
reused after assessing their assertions; extra checks address concrete gaps rather than
routine repetition. Accepted-child labels alone are insufficient. `integration` remains
optional detail; shared `support` is sufficient structurally, while adequacy is a semantic
judgment. Integration grants no sibling writes.
Source evidence cites a real Definition in `authority`, or `pass_mandate` for the already
bound integration task where no Definition supplies it; this is no synthetic Definition
or new authority. Open integration cannot yield a satisfied result/Pass.

Implement `satisfied` means a checked implementation submission to independent Verify.
It does not establish verified conformance or permit publication.

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

An optional `Budget.WorkDeadline` precedes the whole-run deadline and cancels an active
non-Verify Pass when it expires. Incomplete work is not sealed or sent to Verify. Verify
can use the remaining window; a subsequent correction must still meet the original
work cutoff. The existing deadline, turn, Pass and probe limits remain. The invocation
may bind this cutoff explicitly; setup does not synthesize a reserve.

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
is standalone: include all imports, setup and helpers in its content; another probe's
files or process state do not carry over. Each Agent Turn also uses a fresh container.
Its conversation and provider-home files persist, but tool/REPL variables and `/tmp`
do not. Save a recipe in the provider home or send its complete content again.
The Runtime bounds checks and reserves time for a report; it refuses a check that
cannot fit. The Manager reports progress and continues in a later Turn. The prompt
does not prescribe probe counts or require the Agent to schedule Runtime cleanup.
The probe names its obligation (own or a descendant's), claim and expected outcome; they are recorded
(`probe.requested`) before it runs. It runs on a stable stand: the candidate in a review,
the accepted stand in a Pass without writes, a child's checked proposal, or in a Pass with
writes the caller's live area composed with the accepted stand. The file is added to that
stand in the check's directory and never replaces a project file. With `existing: true`,
`path` selects a captured test file unchanged, `content` is omitted, and omitted `args`
uses the configured check invocation. Its original bytes and digest are retained and
the executed tree equals the stand; its actual assertions still require assessment.
The Orchestrator keeps
the receipt (`probe.receipt`, `Outcome.Probes`): the binding, the probe bytes and digest,
the invocation and what the Runtime observed. Its outcome is the probe's own assertion
verdict: `passed` for an observed zero exit of exactly that request, `failed` for an
observed nonzero exit of the probe's own assertions, `error` for everything else (not
started, timed out, uncertain, mismatched, not executable with exit 126 or 127, or killed
by a signal with exit 128 or above), which never counts as a pass or as an expected
refusal. `probe.rerun` repeats the exact probe of an error receipt on the same candidate.
Implement and artifact-review `done`/`preserved` claims require typed evidence.
`kind: public_behavior` cites a passing receipt within the responsibility with a nonempty `probe.boundary`.
A passing execution may support several assessed claims and Definitions. Explain
its actual support and gaps; the retained invocation and assertions remain unchanged. A failed receipt belongs in a genuine finding (or its
reasoned disposition), not in coverage evidence. A failure of an own public
contract is reported under that own obligation; coordinate a dependency's required
implementation change through the parent, preserving the dependency's write owner.
The parent independently assesses that cross-area cause and records the necessary
finding for the actual correction owner in its subtree; the caller's contract
finding alone does not route the dependency's implementation work.
A parent may cite a descendant's adverse receipt in that finding after assessing
the causal link independently. The receipt retains its original requester,
obligation, candidate and outcome; it is not relabelled as coverage evidence.
An omitted evidence `boundary` inherits that recorded value; a supplied different value
is refused. The stored evidence retains the boundary, through the actual public entry and relevant parsing/transport/
persistence. `kind: static` cites source, location and reasoning explaining why execution
is inapplicable. Neither private-validator execution, source reassurance for executable
behavior nor a suite total closes a public behavior claim. Missing required execution
stays open. One receipt may support both local claims and integration. Sources/receipts bind to the submitted stand after writers stopped. Tooling
checks structure and bindings; Managers/reviewers judge authority, actual entry use,
assertions and semantic adequacy. Repeating a boundary label proves no such adequacy.

Fresh independent reviewers receive the bound model and immutable candidate directly.
There is no empty model-only workspace, pre-candidate basis artifact, parent preapproval
or global waiting barrier. They derive their own expectations before assessing author
assertions; the Host enforces independent conversations, participation and candidate
binding rather than candidate-blind preregistration. Model assessment remains its own
mandatory review of intent. Optional contract dependencies use the same mechanism.

Relied-on probe files are delivered in full within the bounded assignment budget;
larger files require `look_up probe`. That lookup records the exact delivered bytes'
digest. A valid report submission ends probing/helper work in that Turn and explicitly
asks the provider to end the Turn; a submitted report never substitutes for its normal
terminal event and observed stop. NIGHT's bounded operator validation is described in
[the NIGHT handoff](../../docs/development/night-apply-stability-20261021.md).

A probe runs only while its turn keeps a fifth of its timeout for the report, and it ends
before that point; late in a turn it is refused rather than letting the timeout fail the
turn.
A helper runs no probe; `look_up topic probe` returns a whole receipt.

Evidence items support assessed conclusions through a passing receipt or an inspected
source file with claim, location and reasoning. Shared support is retained once; a
receipt can support multiple conclusions without mandatory duplicate entries. A bare
satisfied result or suite total is insufficient, and concrete unresolved required gaps
cannot be satisfied. Whether the evidence covers the bound scope is the reviewer's and
parent's judgment. Items bind to the stand of the submission: the candidate or the unchanged accepted
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
For a demonstrated script setup failure before candidate execution, the existing
`expectation_error` disposition may include `setup` evidence and the `passing` corrected
script receipt. The candidate, claim, expectation, boundary, check and invocation must
match; the observed failed exit and corrected bytes are retained. `setup` is explicitly
the reviewer's semantic assessment, not a Host certification of nonexecution. Ordinary
review and parent integration assess it; it does not force another whole Verify solely
to disprove that broken script, including at Root. Hangs, uncertain stops, Runtime
non-starts and genuine findings do not use this exception; required-check failures stay
blocking. Other Root disproofs of its own receipt in the same Pass await a fresh review:
the Pass verdict needs correction and `Reassess` runs a new Verify of
the unchanged candidate, whose Root reviewer confirms it or raises a finding. A done claim
with an undisposed adverse receipt of its own and a satisfied result with one within the
responsibility are refused.

## Finding kinds

A finding is a `Deviation` (blocks, states `Expected`, names a correction purpose),
`Suspected` (only in a review; blocks and states `Expected`, but names no correction and is
never handed to a work Pass: `Reassess` holds while only suspicions and fresh disputes keep a
review from being clean), `ModelProblem` (only in
a Model assessment) or `Suggestion` (never blocks, no correction). Only the declared effect
is checked. A suspicion closes as `disproved` or as `confirmed` with the deviation of the same owner
and obligation that the same submission raises. A blocking review finding that repeats a
finding a review of the unchanged candidate disproved, same owner and obligation text, is
refused without a failed probe receipt of its Pass.

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
