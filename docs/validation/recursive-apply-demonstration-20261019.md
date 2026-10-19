# Recursive Apply implementation and library demonstration — 19 October 2026

**Scope:** this record covers the implementation of [DEC-056](../concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) under the owner's [dispatch](../concepts/sources/recursive-apply-implementation-20261019.md), merged as [PR #463](https://github.com/Glacius-Labs/Markitect/pull/463), and its real library demonstration with the [harness](../../experiments/recursive-apply-demo/README.md). The timed demonstration ran on the pre-merge build `63aecf79`, not on the merged Main `ce0a56e3`; the differences are listed under risk 6. Implementation work, local tests, hosted CI and the product demonstration are reported separately. No human acceptance exists: neither the product change nor the generated library has been accepted by the owner. This record is evidence for these runs only.

## Verdict

The simplified concept works as implemented. `markitect apply` realized Library M1 from the model and its execution controls, with no application artifacts or prepared internal APIs. It then reconciled the actual A1 to M2. Both Applies passed fresh recursive Verify and the required checks, including the library oracle, and were published with read-back. Everything took **932 seconds**, inside the 30-minute hard limit, with no owner question and no human repair.

The generated library is good but not flawless (source audit by a further Sonnet agent: **B+**). The run also showed that trust without technical enforcement does not keep Managers inside their responsibilities. **Overall grade for how this run went: B+.** The evidence is encouraging but narrow (n = 1, one model, one small example, oracle-informed setup); it is not yet proof of quality or scalability.

## Implementation (separate from the demonstration)

| Item | Observation |
| --- | --- |
| Time | About 25 minutes of analysis, then 1 h 56 min from the contract commit `7337879c` (11:05 UTC) to the merge (13:01 UTC), including reviews, fixes and the demonstration. |
| Organization | Contract first (Agents port, `run.Reconcile`). Then five parallel Sonnet work packages in separate worktrees: engine, Claude adapter, Codex adapter, project controls/layout/accounting, Host composition. A documentation pass followed. 16 Sonnet subagents, including the source audit, used about 3.4 M tokens; their cost was not measured. The model choice follows the owner's dispatch; the exact model ID behind the subagents' `sonnet` alias was not observed. |
| Size | PR #463: 295 files, +12,187 / −36,907 lines. Removed without compatibility: the sequential task machine, deliveries, the reviewer/expectations machinery, reservations, rejection counters, journal binding/restore, the SQLite journal adapter and repository control. |
| Review | One early engine review, three reviews over the complete diff and one delta review of all fixes, all by Sonnet agents separate from the authors. Reported defects: 7 + 8 + 9 + 5 + 4. Every real defect was fixed, among them stale finding items, round-1 budget reserve, volume leaks, `Stopped` truthfulness, usage of aborted turns, token redaction in the Codex adapter, Root's Brief without obligations, layout case aliases, protected-input restore, retention of verified work after a failed publication, destructive owner-channel cleanup and the Failed status mapping. |
| Defects found only by real runs | The Python check executable must be a regular file, not a symlink (harness configuration). `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` aborts the CLI in the image because bubblewrap is missing, so it is not set. Provider-home volumes were never removed (product defect, fixed in `33b03690`). In the plumbing smoke the library Manager re-implemented sibling domain logic in its own area; the Brief now states that included Definitions belong to their owners. |
| Local tests | `go vet`, `go build` and `go test ./...` green on Windows and Linux (WSL). Architecture gate, `markitect check` (working tree and HEAD), modules, documentation links and script tests green. The race detector ran only in hosted CI, because the local toolchains have no C compiler. |
| Hosted CI | Quality and race detector green on the exact PR head `b74a5726` and on Main `ce0a56e3`. One earlier Quality run failed on `TestProviderEndingWithoutResultIsIncomplete`: the test fake chose at random between a buffered line and end of stream. This was a nondeterministic fake, fixed in `b74a5726`. The hosted Windows job did not run for this PR or its merge; Windows is covered only locally. |

## Demonstration

**Setup.**
- **Product and model:** build `63aecf79`. Every Execute and Verify turn used Sonnet `claude-sonnet-5-5` with high effort, observed in each provider init. No Codex or OpenAI inference ran.
- **Inputs:** the target started with the M1 model, its dependencies and a prepared `execution.yaml`: layout root `realization`, exception `realization/cmd/library` → `library`, budget, and the required checks `go test ./...` and the library oracle as a check image.
- **Agent input set by the harness:** the Host profile's system prompt (including "use only the Go standard library") and the owner request.
- **Owner decisions:** two decisions were bound (unknown argument fields → `invalidRequest`; a due date beyond 9999-12-31 → atomic refusal). The owner made them in earlier runs when Agents asked; both concern oracle edge cases.
- **Oracle feedback:** the oracle check showed the Agents failing case IDs and short build or error output, but not the expected values.
- **Time limit:** one `timeout` outside all prompts enforced 27.5 minutes for both Applies, then cleanup within 30 minutes. Each Apply had a 20-minute budget, capped by a shared absolute deadline.
- **Owner step:** scripted as instructed. It committed the published A1 unchanged, changed only `maxOpenLoans`, and committed M2.

| | A1 (M1 → A1) | A2 (`maxOpenLoans` 5 → 3, against actual A1) |
| --- | --- | --- |
| Wall clock of `markitect apply` | 652 s | 280 s |
| Engine time / Verify rounds | 615 s / 3 | 211 s / 2 |
| Agent turns: Execute / Verify; peak in parallel | 14 / 18; 8 | 6 / 12; 6 |
| Agent seconds: Execute / Verify | 1009 / 469 | 180 / 309 |
| Parent/child messages; owner questions | 13; 0 | 2; 0 |
| Findings (blocking) | 30 (29), all blocking ones closed; 1 advisory open | 8 (7), all closed |
| Published paths | 24: 18 realization files and 6 registry shards | 7: lending code, test, README and shard; library tests and README |
| Required checks on the final candidate | `go test` and oracle green | `go test` and oracle green |
| Oracle rerun after publication | m1 12/12 with discrimination | m2 12/12 with discrimination; A2 under m1: 10/12, as expected |
| Cost estimate reported by Claude Code | 6.92 USD | 1.80 USD |

The total of **932 s** covers projection, A1, the scripted owner step, A2 and cleanup. The script did not orchestrate any Agent work, check, capture or publication.

The post-publication oracle rerun uses the same cases as the required check, in the stricter hardened assessment harness of `experiments/runtime-baseline`; it is not a second, independent judgement. Its results are in the attempt directory outside the repository.

The cost figures are Claude Code's per-turn `total_cost_usd` estimates for the runtime turns only. Billing for the OAuth runtime token was not observed.

**A1 timeline.**
- **0–57 s:** initial Verify on the empty stand. Every area reported its missing realization, and the checks failed because `realization/` did not exist. Execute started early: Root at 11 s, the others as their findings arrived.
- **Execute until 364 s:** Root's first turn took 353 s and was the critical path. Catalog and lending/reminders finished within 80 s. Members and lending first did nothing on Root's instruction, then delivered in later turns.
- **Round 2 (367–480 s):** both checks passed. Accounting blocked five registrations in library's shard that lay outside its area (`registry.outside_area`).
- **Repair (480–516 s):** library moved its files into `realization/library/**`.
- **Round 3 (528–614 s):** green, followed by publication.

**A2.** The initial Verify found the oracle failing on its admission cases and the code limit at 5. Root asked lending to change the policy. Lending changed the constant, its tests and its README, and library updated tests that asserted 5. Round 2 was green. Members, catalog and reminders only verified and changed nothing.

**Separate real runs**, not part of the timed demonstration (evidence in their attempt directories outside the repository):
- A plumbing smoke: A1 only, product `38d86aa8`, verified in 322 s, 3.59 USD estimate.
- A no-change Apply on the published A2, committed by the same script, with product `33b03690`: 6 Verify turns, 0 Execute turns, one green round, `no_change`, no volume left, 183 s, 0.54 USD estimate.

## Artifact quality (source audit of A2)

The audit was done by a separate Sonnet agent, the same model family as the authors; it is not a human or cross-model review. `go vet` and `go test` pass.

Overall **B+**:
- **Fidelity A−:** the complete callable contract and both owner decisions are realized; the lowered limit leaves existing loans untouched; overdue assessment is pure.
- **Structure and ownership B+:** every area's code lies in its mirrored area, and the program composes all four domain packages.
- **Tests B+:** 90–100 % coverage of the domain packages (the `cmd/library` entry point is uncovered), with the M2 boundary tested at three levels.
- **Registry B+:** one entry per file in the right shard.
- **A2 minimality A.**

Gaps that fresh Verify did not block:
- The catalog README misdescribes how availability is enforced.
- `lending/reminders` has no README.
- Date, identifier and name validation is copied across packages.
- Registry links for cross-area rules (`members/eligibility`, `catalog/availability`) are missing.
- No test pins that overdue assessment changes no state.
- Dead helper code is left behind.
- An envelope with a duplicate key answers with `id: null`.

None of these changes the oracle's verdict. They show the limits of Verify by peers of the same model.

## Against the concept

**Held:**
- The permanent hierarchy and ownership were fixed before Apply.
- Execute ran in parallel in one shared volume mounted by separate containers, with no private writer copies, merges or parent updates.
- Communication went only between direct parent and child, through 15 messages. No owner question was needed.
- Quiescence was observed through the container ends, including all subprocesses, and the harvest refuses a volume that is still mounted.
- There was one immutable candidate per round, and fresh Verify sessions ran in separate homes without resume, children before parents.
- Findings were kept until their raiser closed them, and the required checks ran on the same candidate.
- Unchanged areas were assessed without manufactured work, both in A2 and in the no-change Apply.
- Publication was guarded, with read-back.
- No new process machinery was added.

**Did not hold in behavior, though later rounds repaired it.** Evidence: the Execute transcripts, kept outside the repository, and the run's messages and findings.
- In A1, Root's first turn wrote domain code for its children's responsibilities into a Root-owned directory (`realization/internal/...`). It told members, catalog and lending that no action was needed, saying that "no Package paths are assigned"; its Brief showed only Root's own areas. Members and lending followed that instruction at first. Root corrected itself after about 3.5 minutes.
- Library wrote `realization/go.mod`, which Root owns, in parallel with Root and with a different module path. It also placed and registered its own code in Root's area, until accounting refused the registrations and library moved the files. The final layout is clean.
- `write.observed` flagged none of this, because the owners also ran in that phase. Attribution from transcripts is approximate, because Agents mostly write through Bash.

## Problems and risks, ranked

1. **Ownership is trust only.** A parent can pre-empt its children, and a child can write a parent's file; the final state was correct only because later rounds repaired it. The cheapest next step is to show each parent its children's areas in the Brief. Technical enforcement remains open.
2. **Verify is not as strict as an outside audit.** Wrong documentation, a missing README and duplicated helpers passed. Whether such documentation and structure gaps should block is a model and policy question; they are not obviously "concrete model violations".
3. **Evidence is narrow and partly test-informed.**
   - One timed run on a small library, with one model, after a plumbing smoke on the same model and a Brief clarification informed by that smoke.
   - The two bound owner decisions and the oracle case IDs visible to the Agents both came from or point to the oracle.
   - There is no variance across repetitions and no scalability data.
4. **Cost and redundancy.** About 7 USD for a small A1 (estimate). The initial Verify of an empty project and the whole-tree re-verification add turns, and parents often restate their children's findings.
5. **Not exercised for real:**
   - an owner question during a run;
   - restore of an interrupted Apply (not implemented: a new Apply starts from the committed repository);
   - the Codex adapter (fake tests only);
   - the hosted Windows CI job.
6. **The demonstration build differs from Main.** After `63aecf79`, five things changed:
   - `33b03690` removed provider homes, and the no-change run covered it.
   - `ff54f52f` closes a failed work volume, gives a Manager whose earlier turn left no conversation the full first prompt, and bounds quoted protected inputs.
   - `b74a5726` fixed a test fake.
   - Main's owner documentation was merged in twice.
   - None of these was exercised in a timed two-Apply run.

## Final assessment

| Aspect | Grade | Reason |
| --- | --- | --- |
| Implementation work | A− | Complete replacement in under two hours of parallel work, with review loops and green gates. One product defect surfaced only in a real run; one CI failure came from a nondeterministic test fake. |
| Product mechanism in the run | A− | Both Applies succeeded at the first attempt within budget, and every concept guarantee was observed. Responsibility violations happened and were repaired, not prevented. |
| Generated artifacts | B+ | Behavior complete; documentation and structural gaps remain (same-family audit). |
| Strength of the evidence | C | n = 1, small example, oracle-informed setup, estimated cost only, owner question and restore untested, pre-merge build. |
| **Overall run** | **B+** | The simplified strategy is the right direction and works end to end. The next levers are quality control of ownership and documentation and broader evidence, not more process. |
