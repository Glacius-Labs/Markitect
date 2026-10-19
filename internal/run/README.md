# Run

This package is the reconciliation engine of Apply ([DEC-056](../../docs/concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace),
[concept](../../docs/design/canonical-model/manager-apply-concept.md#current-agreed-concept--19-october-2026)).
`Reconcile(ctx, Config)` runs one bound reconciliation: fresh Verify of the starting
stand, parallel Execute in one shared working tree, observed quiescence, one immutable
candidate per closing Verify round, and repair through the owners until a round is green
or the budget ends. It does no file or process I/O itself. Everything goes through the
ports in `Config`: `Agents` (one Agent turn), `Workspaces` (volumes), `Checks` (required
checks), `Owner` (the human, optional) and `Evidence` (the append-only record). Apply
prepares the team, briefs, checks and accounting from one complete capture, passes them in
and publishes a verified candidate; this package never publishes and never commits.

## Cycle

1. Candidate 0 is the start stand, materialized read-only as `<RunID>-c0`; the shared
   working tree is `<RunID>-work`, created with the union of the mount paths of all enforced
   write plans as empty directories (candidate volumes need none). Later candidates are `<RunID>-c<N>`. Candidate volumes
   close after their round. Once the shared volume holds the start stand, the engine never
   closes it: after `Reconcile` returns, in every outcome, the caller owns it and closes it
   after a successful publication or keeps it as retained work. A shared volume whose
   materialization failed is closed by the engine.
2. A round runs `Account` and all required checks on the candidate (checks in parallel),
   then one fresh Verify turn per Manager, children before parents, independent branches
   in parallel. A parent starts when all its children's Verify turns of the round ended.
3. A round is green when every Manager assessed it, no blocking finding is open and every
   check passed. Round 1 green is Verified with the start stand as candidate and no Execute.
4. Otherwise the open blocking findings go to their owners and the Execute phase runs: a
   Manager that is not running and has undelivered items gets an Execute turn. In round 1
   only, owners may start as soon as their findings are known while other initial Verify
   turns continue on candidate 0. In later rounds Execute is paused for the whole round.
   A Manager with `Write.Enforced` runs its Execute turns read-only unless `Write.Base`, with
   `Write.Mounts` overlaid; without it the whole tree is writable and ownership is trusted.
   Verify turns are always read-only without mounts. Mount paths are validated before any
   effect: clean relative paths below `/work`, no `.`, no duplicates, none naming a file of the
   start stand, each before the mounts of its descendants.
   A Manager with `Building` (build from the model) does not start its first Execute turn
   while its parent is `Building` and has not ended its first Execute turn of the run (finished,
   waiting or failed) and is not disabled. So that every such parent has a first turn, round 1
   seeds each `Building` Manager that has a `Building` child with one initial Execute item
   (prepare the foundations the children need within its own area, for Root the project
   skeleton and shared prerequisites, then coordinate): it starts at once, in parallel with the
   initial Verify, and a parent waiting for a child ends its turn with `finish` waiting, which
   counts as ended. Siblings and other areas stay parallel. The wait is recorded once per
   Manager as `build.wait` with the parent, and the first Execute prompt of a building Manager
   says that its area holds no artifacts yet. Reconcile runs (`Building` false) are not ordered.
5. The phase is quiescent when no Execute turn runs, no Manager has undelivered items and
   no owner question waits. The shared volume is then harvested (`Workspaces.Harvest`
   itself refuses to call a volume complete while a container mounts it) and the tree
   becomes the next candidate.

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
read-only mounts and the writable overlays nested inside them) and its items until one of them finished without error; later turns
resume the conversation and get only new items. A turn that fails after its container
stopped gets one retry with a note and the items it consumed again, even when the adapter
reported a conversation; the second consecutive failure stops scheduling that Manager. An item that restates a finding lapses once
the finding is closed; messages and notes stay. An early round 1 Execute turn starts only while
the budget covers it, the unstarted round 1 Verify turns and one closing round.

## Stopping and trust

The deadline cancels all running turns and checks and waits for them. The agent turn budget
and the round budget end the run Incomplete. Any turn whose container end was not observed
(`TurnResult.Stopped` false), an uncertain harvest or a failed evidence record also ends it
Incomplete, because quiescence, capture or the record is uncertain; nothing may then be
published. A definite error of the configuration or a port is Failed; a check that
cannot run on a candidate, such as a missing check directory before the first Execute,
is a failed check. An unsupported harvest entry (a symlink, a device or an invalid name)
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
Verify rounds and the findings; `execute.go` schedules Execute and routes messages;
`turn.go`, `tools.go` and `prompts.go` hold turn handling, tool schemas and prompts. Tests
use scripted fakes for every port and run with `-race`.
