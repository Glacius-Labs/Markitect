# Implemented runtime contract of Apply

This is the technical owner of the Apply strategy implemented on branch `dev/recursive-apply-simplified` for [DEC-056](../../concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace). The [concept](manager-apply-concept.md#current-agreed-concept--19-october-2026) owns the intent and the [library walkthrough](manager-apply-library-flow.md#current-library-walkthrough) shows it in sequence. The older [core](manager-apply-core.md), [execution](manager-apply-execution.md), [configuration](manager-apply-configuration.md), [port](manager-apply-port-contracts.md), [journal](manager-apply-journal.md), [architecture](manager-apply-go-architecture.md) and [rebuild](manager-apply-rebuild-plan.md) documents describe the earlier design. Where they conflict, this document and the source win.

Status: implemented on the branch, pending independent review, hosted CI and owner acceptance. Source tests and smokes are bounded evidence. They establish no real provider run, the two-Apply demonstration, release or owner acceptance. The packages own their details: [run](../../../internal/run/README.md), [apply](../../../internal/apply/README.md), [realization](../../../internal/realization/doc.go), the [Claude](../../../internal/adapters/claudecode/README.md) and [Codex](../../../internal/adapters/codexappserver/README.md) adapters.

## Team and Brief

The team is permanent and derived, not planned: Root (`.`) and one Manager per local Package, with parents from the model hierarchy. Every Manager takes part in every Verify round. `run.Config` lists Root first and every parent before its children.

Apply renders each Manager's Brief from the captured model: the applicable obligations with provenance, its owned artifact areas, its registry shard and the required checks. Own Definitions are the Manager's obligations; included Definitions describe other areas' contracts, which it uses through their owners instead of realizing them again. A Manager with no Definitions of its own (Root, or an empty intermediate Package) additionally receives the Definitions the required checks name in `requiredBy`, with the reason "required by check <name>", so it sees what the whole-project checks and the shared files it owns are for. The engine adds the mandate, contacts, owner request and decisions, findings, messages and check results. A Manager runs with its Execute and Verify profile from the project's settings; Execute and Verify use separate provider homes and Verify never resumes. The homes are runtime volumes mounted by name; Apply removes them through the Host's container engine when the run ends.

## Ownership and registry shards

Ownership is the persistent `layout` of `.markitect/execution.yaml`: artifact roots mirror the model hierarchy (`realization/lending/reminders/...` belongs to `lending/reminders`), explicit exceptions with a reason name other owners (`realization/cmd/library` belongs to `library`), the deepest mapping wins and everything else, such as `realization/go.mod`, belongs to Root. An invalid layout fails preparation. A layout grants no write authority. Without a layout the registry shards alone decide.

Each Manager creates files in its area and registers them itself in its own shard: `.markitect/registry/registry.yaml` for Root, `.markitect/registry/<package>/registry.yaml` otherwise, with `form`, `realizes` and a `why`. There are no per-file reservations. Registering a path in another Manager's shard is a blocking structural finding.

## Candidate accounting and protected inputs

`apply.Candidates.Account` runs on every candidate before Verify. Its blocking findings each name the Manager that must act: an unexplained or duplicate file, an invalid or unknown shard, a registration outside the registering Manager's area, and any canonical or control input that changed, vanished or appeared. A changed or deleted protected input quotes its original content when that is at most 8 KiB of UTF-8 text (otherwise its SHA-256 and size), so the owner can restore the exact bytes. Without a declared layout an unexplained new file is addressed to Root, which coordinates but does not own it: the Manager that created the file registers it in its own shard, and Root finds out who that is through its children. Shards are the one writable control input. Preparation runs the same accounting on the start stand before any Agent starts. Structural findings hold for exactly the candidate they were observed on.

## Reconcile cycle

`run.Reconcile` runs one bound reconciliation. It does no I/O itself.

1. **Start.** The start stand is materialized read-only as candidate 0 (`<runID>-c0`); the shared working tree is `<runID>-work`, initially the same bytes plus the mount directories of the enforced write plans (the union of their mount paths, created empty). All Execute turns mount the shared volume at `/work`. A Manager with an enforced write plan mounts it read-only unless the plan's base is writable and overlays the plan's mounts in order; without an enforced plan the whole tree is writable. Verify turns never get mounts. The engine refuses invalid mounts before any effect: clean relative paths, no `.`, no duplicates, none naming a file of the start stand, parents before descendants.
2. **Round.** A round runs `Account` and all required checks on the candidate in parallel, and one fresh Verify turn per Manager, read-only, children before parents. A parent starts when its children's turns ended. Independent branches run in parallel.
3. **Green.** A round is green when every Manager assessed it, no blocking finding is open and every check passed. A green round 1 is Verified with the start stand and runs no Execute.
4. **Execute.** Otherwise open blocking findings go to their owners and every Manager with undelivered items gets an Execute turn. In round 1 only, an owner may start as soon as its findings exist while other initial Verify turns still run on candidate 0, but only while the budget still covers that turn, the unstarted round 1 Verify turns and one full closing round. An item that restates a finding lapses when the finding is closed before the item is scheduled, counted for quiescence or taken; messages and notes stay. Later rounds pause Execute while Verify runs. Messages travel only between a Manager and its direct parent or child; only Root may ask the human. In a build from the model (DEC-057, Managers marked `Building`) a Manager's first Execute turn waits while its parent is building, has not ended its first Execute turn of the run and is running, has undelivered items or has not yet reported its round 1 Verify; a parent without work never blocks. This orders only construction dependencies: siblings and other areas stay parallel, and Reconcile runs are not ordered. The wait is recorded once as `build.wait`.
5. **Quiescence and capture.** The phase is quiescent when no Execute turn runs, no Manager has undelivered items and no owner question waits. The runtime then harvests the shared volume, which itself refuses while any container mounts it. The tree becomes candidate N (`<runID>-cN`), is materialized read-only and verified in a whole new round. Only an uncertain harvest ends the run Incomplete. An unsupported entry (a symlink, a device, a name that is no valid path) is kept out of the tree and raises, per owner of the paths (Root for an invalid name), one blocking `markitect` finding in the next round that counts all entries and names at most 20, so such a candidate is never green, the prompt size stays bounded and the owner removes the entries in the next Execute phase.
6. **End.** The run ends when a round is green (Verified), the round or agent-turn budget or the deadline ends (Incomplete), or a port or the configuration fails definitely (Failed). Candidate volumes close after their round. Once the shared volume `<runID>-work` holds the start stand, the engine never closes it (a failed materialization it closes itself): after `Reconcile` returns, in every outcome, the caller owns it; Apply closes it after a successful publication and keeps it as retained work otherwise.

**Findings.** A finding has one owner (a reporting Manager or a descendant) and one raiser. A Verify finding stays open until a later fresh Verify of its raiser closes it with a reason; a fresh reviewer that does not mention it does not close it. Failed checks and `Account` findings are raised by `markitect`. Only blocking findings go to Execute; advisory ones stay visible. Blocking means a concrete model violation, absent evidence the model requires or a failed required check.

**Owner questions.** Root's `send` to `owner` publishes a question through the Owner port and work that does not depend on the answer continues. The answer reaches Root's next turn. An unanswered question at the end leaves the run Incomplete; without an Owner port the question is refused.

**Budget and deadline.** The budget bounds Execute plus Verify turns (`agentTurns`), Verify rounds including the initial one (`rounds`), each turn (`turnTimeout`) and the whole run (`Budget.Deadline`, an absolute time Apply sets from `executionTime` and the run configuration's optional `deadline`). The deadline cancels all running turns and checks and waits for them. A turn that fails after its container stopped is retried once with a note and the items it consumed again, even when the provider kept a conversation; a second consecutive failure stops scheduling that Manager, and the run ends Incomplete if that Manager still owns an open blocking finding. A Verify turn without a valid `assess` is retried once; if that fails too, the turn's error stays in the Incomplete reason. A turn that failed before any container started is reported `Stopped` with its error, so it never makes quiescence uncertain.

## Host tools

Served to the provider over MCP by `markitect-agentd` inside the turn's container.

| Tool | Turn | Behavior |
| --- | --- | --- |
| `send {to, text}` | Execute | `to` is `parent`, a direct child path or, for Root only, `owner`. Anything else is refused with the allowed contacts. The answer returns the caller's new undelivered items. |
| `finish {status, summary}` | Execute | `status` is `done` or `waiting`. A turn that ends without it counts as done. |
| `assess {summary, satisfied, findings[], closed[]}` | Verify | Once. Findings carry `owner`, `obligation`, `observed`, `blocking`. A report that contradicts the raiser's open blocking findings, or closes a finding it did not raise, is refused with the reason and may be repeated. |

A Manager's Execute turns get mandate, Brief, contacts, owner decisions, the shared-tree note and their items until one of them finished without error; later turns resume the provider conversation with new items only. With an enforced write plan the shared-tree note says that outside the writable areas `/work` is read-only and a write there fails, and that scratch output goes under `/tmp`. The first Execute prompt of a building Manager adds that its area holds no artifacts yet.

## Ports and adapters

| Port (owner) | Adapter | Notes |
| --- | --- | --- |
| `run.Agents` | `claudecode.Agents`, `codexappserver.Agents`, selected per profile by the bootstrap router | One fresh container per turn, `/work` and `/home/agent` volumes, user `1000:1000`, the container is stopped before `Run` returns and `Stopped` reports whether the provider exit and the end of the container (a `Close` that did not have to remove it by force) were observed. Claude Code usage is accumulated from the stream and replaced by the result message's totals; a turn without them reports a lower bound and says "usage partial" in `Reason`. Known secrets are removed from tool-call arguments before the handler sees them. A profile without an adapter is refused. |
| `run.Workspaces` | `gitrepo.Workspaces` on container volumes | Materialize with read-back, Harvest, Peek, Close. |
| `run.Checks` | `checks.Runner` | One container per check on a disposable copy of the candidate; a nonzero exit fails, an unobserved result never passes. |
| `apply.Publisher` | `gitrepo` publisher | Preflight against the captured source guard, Publish, read-back in Git's clean form. |
| `run.Owner` | bootstrap file channel | See below. |
| `run.Evidence` | bootstrap file evidence | See below. |

## Configuration

`.markitect/execution.yaml` (`apiVersion: markitect.execution/v1`, `strategy: shared-workspace-v1`) is closed. Its fields are `budget {executionTime, turnTimeout, agentTurns, rounds}`, `profiles {execute, verify}`, `instructions`, per-Package `packages` overrides, optional `layout {roots, exceptions}`, optional `scope` (modeled scope selectors `{kind, property}` that add Definitions to a Manager's context) and `checks` (id, profile, directory, args, `requiredBy`, timeout). The [apply guide](../../../internal/apply/README.md#execution-settings) shows the shape and the embedded product defaults.

The run configuration is a separate closed JSON file (`formatVersion` 2) with the explicit invocation facts: `repositoryIdentity`, `productBuild`, `ownerRequest`, optional `ownerDecisions`, `stateDirectory`, `engineExecutable`, `workspaceHelper`, `harvest {maxEntries, maxBytes}`, optional `deadline` (RFC 3339, UTC) and `credentials`. It adds no project settings or profile defaults.

## Evidence and state

The state directory of the run configuration is held by one Apply at a time (`apply.lock`). It must lie outside the repository; a state directory inside, equal to or containing the repository is refused before anything is created.

```text
<state>/apply.lock
<state>/current-run                          run identity of the Apply holding the lock
<state>/runs/<runID>/events.jsonl            append-only, synced per event
<state>/runs/<runID>/transcripts/<turn>.jsonl  provider stream, secrets redacted
<state>/runs/<runID>/report.json             written once at the end
<state>/questions/<id>.json                  open owner question
<state>/answers/<id>.txt                     the owner's answer
```

Event kinds: `run.start`, `round.start`, `check`, `assessment`, `finding.raised`, `finding.closed`, `round.end`, `turn.start`, `turn.finish`, `turn.end`, `message`, `question`, `answer`, `owner.error`, `candidate`, `write.observed`, `build.wait`, `manager.disabled`, `workspace.materialize`, `workspace.close`, `home.remove`, `outcome`. A run identity with an existing directory is refused; evidence is never overwritten. A failed record or report write ends the run Incomplete. The owner answers with `markitect answer ID --text ANSWER --run-config PATH`; the answer is accepted only for a question of the running Apply (`current-run`). When a new Apply opens the channel right after taking the lock, it removes only the channel's own question `.json` and answer `.txt` files of earlier Applies.

## Publication and read-back

Only a Verified outcome is published, as the exact verified candidate against the committed source captured at the start. The publisher refuses a source that changed since capture and a change set the target cannot represent, writes the changes and reads every path back. Multi-file publication is not atomic; a partial write stays visible and is reported. Apply closes the shared work volume only after an applied or unchanged receipt; after any other receipt or a publication error it keeps the verified work and reports it as retained. The command reports `Passed` only for an applied or unchanged read-back. Apply never commits, merges or pushes. Human acceptance and a later commit follow project policy.

## Trusted and enforced

| Enforced by the code | Trusted |
| --- | --- |
| Complete accounting of start stand and every candidate; protected inputs unchanged. | Managers write only their own area. |
| Registration only in the owner's shard. | Managers create and change files only in their own area. |
| Quiescence observed by the container runtime before capture; verify only on an immutable read-only candidate; required checks on the same candidate. | Verify assessments and the Managers' judgment of obligations. |
| Parent/child-only `send`, Root-only `owner`. | The model and the human own intent; Managers do not change it. |
| Nothing publishes unless Verified, with evidence recorded. | |

The engine records `write.observed` for files of a Manager that changed although it ran no Execute turn since the previous candidate and shows them to that Manager's next Verify. This is an observation, not complete write attribution.

## Known limits

- Technical write enforcement exists only for Managers whose Execute turns carry an enforced write plan (read-only base with writable mounts); without one a Manager can write another Manager's files in the shared tree and the engine can only observe and account for the result.
- No restore of an interrupted run. The deadline, SIGTERM or a crash ends the run Incomplete with its evidence; a new Apply starts again from the committed repository, not from the incomplete shared tree.
- Every round verifies the whole team again; unchanged areas are not skipped.
- Shared-tree scalability, native provider message behavior under load and real-provider cost are unproven.
- Source tests use scripted fakes; the two-Apply library demonstration with the independent oracle is separate evidence.
