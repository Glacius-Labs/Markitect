# Run

This package is the reconciliation engine of Apply ([DEC-056](../../docs/concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace),
[DEC-058](../../docs/concepts/register.md#dec-058-apply-always-executes-before-independent-verify),
[concept](../../docs/design/canonical-model/manager-apply-concept.md#current-agreed-concept--19-october-2026)).
`Reconcile(ctx, Config)` runs one bound reconciliation: Execute in one shared working tree
starting at Root, observed quiescence, one immutable candidate, fresh Verify with the
required checks, and repair through the owners with a new candidate and a new whole round
until a round is green or the budget ends. It does no file or process I/O itself. Everything goes through the
ports in `Config`: `Agents` (one Agent turn), `Workspaces` (volumes), `Checks` (required
checks), `Owner` (the human, optional) and `Evidence` (the append-only record). Apply
prepares the team, briefs, checks and accounting from one complete capture, passes them in
and publishes a verified candidate; this package never publishes and never commits.

## Cycle

Every Apply runs the same order (DEC-058): Execute and integration first, then fresh Verify
of one captured candidate. There is no Verify, check or `Account` before the first Execute
turn and no separate route for absent or unchanged artifacts.

1. The shared working tree `<RunID>-work` is materialized from `Start`, created with the union
   of the mount paths of all enforced write plans as empty directories. Candidates are
   `<RunID>-c<N>` for Verify round N and close after their round. Once the shared volume holds
   the start stand, the engine never closes it: after `Reconcile` returns, in every outcome, the
   caller owns it and closes it after a successful publication or keeps it as retained work. A
   shared volume whose materialization failed is closed by the engine.
2. The Execute phase starts at Root, which gets the initial mandate item at once. Every other
   Manager gets its initial item when its parent's first Execute turn has ended (finished,
   waiting or failed) or the parent is disabled, so a parent first supplies the foundation its
   children need and tells them what is available; its messages to them during that turn are
   queued behind the initial item for their first turn. A Manager that is not released runs no
   turn. Siblings then run in parallel. Every Manager takes part; its first turn may conclude
   that no change is needed. The initial item tells a parent to supply its children's
   foundation within its own area, because they start when its turn ends, and a child that its
   parent's foundation is in place. The release of each Manager is recorded once as
   `manager.release`, with the parent. A Manager that is not running and has undelivered items
   gets an Execute turn.
3. When a child's Execute turn finishes with status `done` (a `finish` call; a turn that ends
   without one reports nothing), its summary goes to its direct parent as the item "Your child X
   reports done: ..." (`child.report`), so the parent integrates its children's results. Items that
   arrive while the parent runs are batched into its next turn.
4. A Manager with `Write.Enforced` runs its Execute turns read-only unless `Write.Base`, with
   `Write.Mounts` overlaid; without it the whole tree is writable and ownership is trusted.
   Verify turns are always read-only without mounts. Mount paths are validated before any
   effect: clean relative paths below `/work`, no `.`, no duplicates, none naming a file of the
   start stand, each before the mounts of its descendants.
5. The phase is quiescent when no Execute turn runs, no Manager has undelivered items and
   no owner question waits. The shared volume is then harvested (`Workspaces.Harvest`
   itself refuses to call a volume complete while a container mounts it) and the tree
   becomes the candidate of round 1. If Execute changed nothing it equals `Start`.
6. Round N runs `Account` and all required checks on the candidate (checks in parallel),
   then one fresh Verify turn per Manager, children before parents, independent branches
   in parallel. A parent starts when all its children's Verify turns of the round ended. The
   Execute phase before round N and its turns carry the number N (`<RunID>-r<N>-<manager>-x<k>`).
7. A round is green when every Manager assessed it, no blocking finding is open and every
   check passed; the outcome is Verified with that candidate, which is `Start` when nothing
   changed. Otherwise the open blocking findings go to their owners, Execute resumes (the
   phase of round N+1), a new candidate is captured and a whole new round runs, until a round
   is green or the budget ends. Execute is paused for the whole of every round.
8. Budget: `Rounds` counts Verify rounds. `AgentTurns` counts Execute and Verify turns and must
   cover at least one Execute and one Verify turn of every Manager, or the run ends Incomplete
   before any effect. Every Execute turn is only started while the budget also leaves room for
   one whole Verify round after it; the run ends Incomplete when it cannot.

## Findings

A finding has one owner (the Manager that must act) and one raiser. A Verify finding stays
open until a later fresh Verify of its raiser closes it with a reason; a fresh reviewer that
does not mention it does not close it. Failed required checks and `Account` findings are
raised by `markitect` and hold for exactly the candidate they were observed on. Only
blocking findings go to Execute; advisory ones stay visible in Verify prompts. The Verify
tool refuses a report whose `satisfied` contradicts the raiser's open blocking findings.

## Host tools

- Verify: `assess {summary, satisfied, findings[{owner, obligation, observed, blocking}], closed[{id, reason}]}`,
  once. A finding's owner is the reporting Manager or a descendant; closed IDs are open
  findings it raised. An invalid call is refused with the reason and may be repeated. A turn
  that ends without a valid report is retried once with a fresh turn; then the run is Incomplete.
- Execute: `send {to, text}` where `to` is `parent`, a direct child path or, for Root only,
  `owner`; anything else is refused with the allowed contacts. An answer also returns the
  caller's new undelivered items. `finish {status: done|waiting, summary}` ends the turn;
  a turn that ends without it counts as done. Without an `Owner`, a question is refused.

A Manager's Execute turns get the mandate, its brief, contacts, owner decisions, the
shared-tree note (with an enforced plan: outside its writable areas `/work` is read-only, a
write there fails, scratch output goes under `/tmp`; for a writable base it lists the
read-only mounts and the writable overlays nested inside them) and its items, the initial
mandate item first, until one of them finished without error; later turns resume the
conversation and get only new items. A turn that fails after its container
stopped gets one retry with a note and the items it consumed again, even when the adapter
reported a conversation; the second consecutive failure stops scheduling that Manager. An item that restates a finding lapses once
the finding is closed; messages and notes stay.

## Stopping and trust

The deadline cancels all running turns and checks and waits for them. The agent turn budget
and the round budget end the run Incomplete. Any turn whose container end was not observed
(`TurnResult.Stopped` false), an uncertain harvest or a failed evidence record also ends it
Incomplete, because quiescence, capture or the record is uncertain; nothing may then be
published. A definite error of the configuration or a port is Failed; a check that
cannot run on a candidate, such as a missing check directory, is a failed check. An unsupported harvest entry (a symlink, a device or an invalid name)
is not uncertain: the candidate is the harvested tree and one blocking `markitect` finding per
owner goes to the owner of the paths (Root for an invalid name). It counts all entries and
names at most 20 of them, so their number never grows a prompt, and the candidate cannot be
green while they stand.

Managers are trusted to write only their own area. After each capture the engine compares
the new candidate with the previous one and records `write.observed` for files of a Manager
that changed although it ran no Execute turn, and shows them to that Manager's next Verify.
This is an observation, not enforcement or complete write attribution. Registrations
outside a Manager's area are rejected by `Account`, which Apply supplies.

## Files

`reconcile.go` and `agents.go` hold the contract, `workspace.go` and `checks.go` the
consumer-owned ports. `engine.go` validates, drives rounds and captures; `verify.go` runs
Verify rounds and the findings; `execute.go` schedules Execute, releases children and routes
messages; `turn.go`, `tools.go` and `prompts.go` hold turn handling, tool schemas and prompts. Tests
use scripted fakes for every port and run with `-race`.
