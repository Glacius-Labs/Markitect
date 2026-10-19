# Recursive Apply implementation and library demonstration — 19 October 2026

**Scope:** the implementation of [DEC-056](../concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) under the owner's [dispatch](../concepts/sources/recursive-apply-implementation-20261019.md), merged as [PR #463](https://github.com/Glacius-Labs/Markitect/pull/463) (Main `ce0a56e3`), and its real library demonstration with the [harness](../../experiments/recursive-apply-demo/README.md). Implementation work, local tests, hosted CI, the product demonstration and human acceptance are reported separately. This record is evidence for these runs only; it is not owner acceptance of the product or of the generated library.

## Verdict

The simplified concept works as implemented. `markitect apply` realized Library M1 from the model alone and then reconciled the actual A1 to M2. Both Applies were verified by fresh recursive Verify, required checks and the independent oracle, and published with read-back in **932 seconds**, inside the 30-minute hard limit, with no owner question and no human repair. The generated library is good but not flawless (independent source audit **B+**). The run also showed that trust without technical enforcement is not enough to keep a parent out of its children's areas. **Overall grade of this run: B+.** The result is strong evidence that the mechanism works, but only one bounded data point for quality and scalability.

## Implementation (separate from the demonstration)

| Item | Observation |
| --- | --- |
| Time | About 25 minutes of analysis, then 1 h 56 min from the contract commit `7337879c` (11:05 UTC) to the merge (13:01 UTC), including reviews, fixes and the demonstration. |
| Organization | Contract first (Agents port, `run.Reconcile`), then five parallel Sonnet work packages in separate worktrees: engine, Claude adapter, Codex adapter, project controls/layout/accounting, Host composition. A documentation pass followed. 16 Sonnet subagents, including the source audit, used about 3.4 M tokens. The model choice follows the owner's dispatch; the exact subagent model ID behind the `sonnet` alias was not observed. |
| Size | 298 files changed, +12.2k / −36.9k lines. The sequential task machine, deliveries, reviewer/expectations machinery, reservations, rejection counters, journal binding/restore, SQLite journal adapter and repository control were removed without compatibility. |
| Independent review | One early engine review, three reviewers over the complete diff, one delta review of all fixes. Reported defects: 7 + 8 + 9 + 5 + 4. Every real defect was fixed, among them stale finding items, round-1 budget reserve, volume leaks, `Stopped` truthfulness, usage of aborted turns, token redaction in the Codex adapter, Root's Brief without obligations, layout case aliases, protected-input restore, retention of verified work after a failed publication, destructive owner-channel cleanup and the Failed status mapping. |
| Defects found only by real runs | Python check executable must be a regular file, not a symlink (harness configuration). The `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` setting aborts the CLI in the image because bubblewrap is missing, so it is not set. Provider-home volumes were never removed (product defect, fixed in `33b03690`). The library Manager re-implemented sibling domain logic in its own area (smoke run); the Brief now states that included Definitions belong to their owners. |
| Local tests | `go vet`, `go build`, `go test ./...` green on Windows and on Linux (WSL). Architecture gate, `markitect check` (working tree and HEAD), modules, documentation links and script tests are green. The race detector ran only in hosted CI, because the local toolchains have no C compiler. |
| Hosted CI | Quality and race detector green on the exact PR head `b74a5726` and on Main `ce0a56e3`. One earlier red run was caused by a race in the adapters' test fakes, fixed in `b74a5726`. The hosted Windows job did not run for this PR or push; Windows is covered only by the local run. |

## Demonstration

**Setup.** Product `63aecf79`. Every Execute and Verify turn used Sonnet `claude-sonnet-5-5` with high effort, observed in each provider init. No Codex or OpenAI inference ran. The target started from the M1 model, dependencies and `execution.yaml` only, with no artifacts and no prepared internal APIs. The persistent layout was root `realization` with the exception `realization/cmd/library` → `library`. The two owner decisions from earlier runs were bound. The required checks were `go test ./...` and the independent library oracle as a check image. One `timeout` outside all prompts enforced 27.5 minutes for both Applies plus cleanup within 30 minutes. Each Apply had a 20-minute budget, capped by a shared absolute deadline.

| | A1 (M1 → A1) | A2 (`maxOpenLoans` 5 → 3, against actual A1) |
| --- | --- | --- |
| Wall clock of `markitect apply` | 652 s | 280 s |
| Engine time / Verify rounds | 615 s / 3 | 211 s / 2 |
| Agent turns: Execute / Verify; peak in parallel | 14 / 18; 8 | 6 / 12; 6 |
| Agent seconds: Execute / Verify | 1009 / 469 | 180 / 309 |
| Parent/child messages; owner questions | 13; 0 | 2; 0 |
| Findings (blocking) | 30 (29), all blocking ones closed; 1 advisory open | 8 (7), all closed |
| Published change | 63-entry tree: realization and 6 registry shards | 7 files: lending code, test, README and shard; library tests and README |
| Required checks on the final candidate | `go test` and oracle green | `go test` and oracle green |
| Independent oracle after the run | m1 12/12 with discrimination | m2 12/12 with discrimination; A2 as m1 10/12, as expected |
| Cost reported by Claude Code | 6.92 USD | 1.80 USD |

The total of **932 s** covers projection, A1, the owner's commit and model change, A2 and cleanup. The external script did not orchestrate any Agent work. Reported cost is Claude Code's own estimate under a subscription token, not billed money.

**A1 timeline.**
- **0–57 s:** initial Verify on the empty stand. Every area reported its missing realization, and the checks failed because `realization/` did not exist. Execute started early: Root at 11 s, the others as their findings arrived.
- **Execute until 364 s:** Root's first turn took 353 s and was the critical path. Catalog and lending/reminders finished within 80 s. Members and lending first did nothing on Root's instruction, then delivered in later turns.
- **Round 2 (367–480 s):** both checks passed. Accounting blocked five registrations in library's shard that lay outside its area (`registry.outside_area`).
- **Repair (480–516 s):** library moved its files into `realization/library/**`.
- **Round 3 (528–614 s):** green, followed by publication.

**A2.** The initial Verify found the oracle failing and the policy at 5. Lending changed the constant, its tests and README. Library updated tests that asserted 5. Round 2 was green. Members, catalog and reminders only verified and changed nothing.

**Separate real runs**, not part of the timed demonstration:
- A plumbing smoke (A1 only, product `38d86aa8`): verified in 322 s, 3.59 USD.
- A no-change Apply on the accepted A2 with product `33b03690`: 6 Verify turns, 0 Execute turns, one green round, `no_change`, no volume left, 183 s, 0.54 USD.

## Artifact quality (independent source audit of A2)

Overall **B+**. Fidelity A−: the complete callable contract and both owner decisions are realized; the lowered limit leaves existing loans untouched; overdue assessment is pure. Structure and ownership B+: every area's code lies in its mirrored area, and the program composes all four domain packages. Tests B+: 90–100 % coverage, with the M2 boundary tested at three levels. Registry B+: one entry per file in the right shard. A2 minimality A.

Gaps that fresh Verify did not block:
- The catalog README misdescribes how availability is enforced.
- `lending/reminders` has no README.
- Date, identifier and name validation is copied across packages.
- Registry links for cross-area rules (`members/eligibility`, `catalog/availability`) are missing.
- No test pins that overdue assessment changes no state.
- Dead helper code is left behind.
- An envelope with a duplicate key answers with `id: null`.

None of these affects the oracle's behavior. They show the limits of model review by the authors' own peers.

## Against the concept

**Held:**
- The permanent hierarchy and ownership fixed before Apply.
- Parallel Execute in one shared volume mounted by separate containers, with no private writer copies, merges or parent updates.
- Communication only between direct parent and child, through 15 messages. No owner question was needed.
- Quiescence observed through the container ends, including all subprocesses, and the harvest refusing a mounted volume.
- One immutable candidate per round, and fresh Verify sessions in separate homes without resume, with children before parents.
- Findings kept until their raiser closed them, and required checks on the same candidate.
- Unchanged areas assessed without manufactured work, both in A2 and in the no-change Apply.
- Guarded publication with read-back.
- No new process machinery was added.

**Did not hold in behavior, though the mechanism caught it:**
- In A1, Root's first turn wrote child-owned code (`realization/internal/...`, `realization/cmd`) and told members, catalog and lending that no action was needed. It gave as its reason that "no Package paths are assigned": its Brief showed only Root's own areas. Members and lending followed that instruction at first.
- Library wrote Root's `go.mod` in parallel with Root, with a different module path.
- Root corrected itself after 3.5 minutes, and accounting refused library's misplaced registrations. The final layout is clean.
- `write.observed` did not flag these writes, because the owners also ran in that phase. Attribution from transcripts is approximate, because Agents mostly write through Bash.

## Problems and risks, ranked

1. **Ownership is trust only.** A parent can overwrite or pre-empt its children, and the final state only looks right because later rounds repaired it. The cheapest next step is to show each parent its children's areas in the Brief. Technical enforcement remains open.
2. **Verify is not as strict as an independent audit.** Wrong documentation, a missing README and duplicated helpers passed. Whether such documentation and structure gaps should block is a model and policy question; they are not obviously "concrete model violations".
3. **Evidence is narrow.** One timed run on a small library, with one model, after a plumbing smoke on the same model, and a Brief clarification informed by that smoke. There is no variance across repetitions and no scalability data.
4. **Cost and redundancy.** About 7 USD for a small A1 (estimate). The initial Verify of an empty project and the whole-tree re-verification add turns. Parents often restate their children's findings.
5. **Not exercised for real:**
   - An owner question during a run.
   - Restore of an interrupted Apply, which is not implemented: a new Apply starts from the committed repository.
   - The Codex adapter, which has fake tests only.
   - The hosted Windows CI job.
6. **Demonstration and Main differ.** The timed run used `63aecf79`. Later commits fixed cleanup, robustness and a test fake; the no-change run covered `33b03690`.

## Final assessment

| Aspect | Grade | Reason |
| --- | --- | --- |
| Implementation work | A− | Complete replacement in under two hours of parallel work with independent review loops and green gates. One product defect surfaced only in a real run; one CI flake came from a test fake. |
| Product mechanism in the run | A− | First-attempt success of both Applies within budget, with every concept guarantee observed. Ownership violations happened and were repaired, not prevented. |
| Generated artifacts | B+ | Behavior complete; documentation and structural gaps remain. |
| Strength of the evidence | C+ | n = 1, small example, estimated cost only, untested owner question and restore. |
| **Overall** | **B+** | The simplified strategy is the right direction and works end to end. Quality control of ownership and documentation is the next lever, not more process. |

Owner acceptance of PR #463, A1 and A2 is not recorded here.
