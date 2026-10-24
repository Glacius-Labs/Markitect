# Delivery plan: Markitect V1

<a id="v1-implementation-organization"></a>

## Target and current state

The target is [V1 concept 1.0](concepts/v1-concept.md) ([DEC-068](concepts/register.md#dec-068-v1-concept-10)). This plan owns the steps from the current source to that target, the work streams and the integration rules. The [V1 execution architecture](design/v1-execution-architecture.md) owns concrete packages and contracts.

Main `334e7c3133ff294c0b827e623d80185e3d3ac9af` integrates the earlier S1–S3 slices. The model, compiler, snapshot, organization and realization foundations are preserved. Project owns checked preparation. One Run executor handles Assessment, Implement and Verify through the complete hierarchy with fresh Manager environments, one Git Workspace Service per Run, durable Git-bundle candidates, stored terminal results and Shared Notes. A direct Apply workflow runs fresh correction Runs under the original allowance and publishes only stored Verify acceptance after every Run has closed. Extract, Cockpit, authoring and Setup are retired. PR #672 carries the Apply deadline into every publication guard (R1).

<a id="current-qualification-evidence"></a>

The [V1 qualification record](validation/v1-qualification-20261024.md) holds the evidence so far. With Codex and normal account login it covers complete native M1 and M2 (loan limit 5 → 3) Apply, independent outside oracles, strict no-change, a seeded defect repaired by an ordinary Implement, Shared Notes across fresh Runs and the native Verify Reject → fresh Implement → fresh Verify component chain. The public Workflow's native correction branch with publication, Claude, account refresh, repetition and owner acceptance are not established. This evidence was taken on the collaboration contract that steps 2 and 3 change; it does not qualify the changed contract.

## Known deviations from concept 1.0

| Concept | Current source | Step |
|---|---|---|
| D10: questions only to the direct parent or direct children | `ask` in `internal/run/inbox.go` accepts any other Manager of the Run | 2 |
| D11: every Implement child candidate reaches the parent with its check result; invalid candidates are not integrable | `work` in `internal/run/execute.go` sends an invalid submission's findings back to the child; the parent receives only valid handoffs | 2 |
| D12: a judgment belongs to the assignment and assessed state; explicit assignment state | every active continuation requires a fresh `judge`; the Codex adapter resets the judgment per message, and a response without a judgment fails the whole Run; a late adverse finding forces Reject onto every ancestor instead of invalidating their judgments; Manager state is spread over many booleans | 2 |
| D13: tasks, not process; fixed tool set with `check`; host-known identities not requested | harness rules and host messages explain continuations, drains and late messages; no `check` tool; native Findings must name a `manager` | 2 |
| D15: closed-result validation in the Run contract | `workflow.ValidateClosed` checks Run resource names; `apply.ValidateApproval` calls it | 3 |
| D17: bounded, ordered Notes pages within the transport limit | `read_notes` returns all Notes in one tool reply ordered by random ID; the store accepts 16 MiB against a 1 MiB frame; a read error cancels the Run | 3 |
| D18: no non-functional Claude path | B1 ([PR #674](https://github.com/Glacius-Labs/Markitect/pull/674)) removed the Claude adapter and its wiring; the runtime image still installs Claude Code 2.1.301 and `codexappserver/metadata.go` still requires a `claude=` record in `runtime.env` (B2) | 3 |
| §5.5: the self-model describes the chosen architecture | Project, Workflow and Harness are owned by the `product/apply` and `product/adapters/agent` Packages; the Run Rule now states D10–D12 ahead of the code | 5 |

Documentation language ([DEC-069](concepts/register.md#dec-069-documentation-is-written-in-english)): outside the verbatim sources, about 70 documents and example files are still mostly German. They are historical design records under `docs/design/project-world`, `docs/design/product-model-refinement` and `docs/design/canonical-model`, including the normative [language specification](design/canonical-model/minimal-language/language-contract.md) and its YAML examples, plus `docs/design/work-item-comparison-20261009.md`, `docs/design/task-management-model/source-description.md`, `docs/validation/astra-assessment-2026-10-06.md` and the archived [conformance audit](development/concept-conformance-audit-20261024.md). They are translated as separate documentation work outside the two code streams. Verbatim sources that serve as a reference, first the owner's [concept v0.18](concepts/sources/v1-concept-v018-20261024.md), get an English translation in the same work. Examples that copy the core vocabulary follow its decision. The German bodies of the `markitect/core` vocabulary in `.markitect/model/markitect/core` and `internal/model/core.go` are protected model content and wait for an owner decision.

Product guides such as the [architecture](architecture.md), the [project workflow](project-workflow.md) and the adapter READMEs describe implemented behavior. They change together with the code in steps 2 and 3.

<a id="apply-rebuild-plan"></a>

## Steps

| Step | Result | Completion evidence |
|---|---|---|
| 1. Record concept 1.0 | Concept, owner sources, decision record and register entry; entry documents, this plan, the architecture and the self-model Run Rule aligned; evidence moved to the qualification record | Structural check, documentation links, generated view and independent complete-diff review |
| 2. Run collaboration contract | D10–D13: hierarchical questions, child diagnostics at the parent, judgment bound to assignment and assessed state with an explicit host-side state, the `check` tool and a fixed tool set; harness rules and host messages reduced to role, rules, context, task and answer format | Controlled-session tests for every counterexample in concept §13.3 and for V03, V05 and V17; one bounded native Codex contract proof of the changed interaction before the contract is fixed; existing real-Git vertical tests adjusted, not weakened; Run, harness and Codex adapter suites; architecture and accounting gates |
| 3. Contract fixes and provider scope | D15 closed-result validation in the Run contract; D17 bounded Notes pages after a controlled reproduction of the size conflict; D18 removal of the Claude adapter and its wiring, with guides and examples (B1), then Claude Code removed from the runtime image and the Codex metadata parser (B2) | Focused tests including a stored-Notes case above one transport frame; full build, vet, architecture and accounting; no remaining Claude product path |
| 4. Native requalification (Codex) | Frozen source after steps 2 and 3: fresh M1, M2 5 → 3, strict no-change, seeded defect, the public Verify Reject → fresh Implement → fresh Verify → publication branch and outside oracles | Bounded dispatches under the standing evidence authorization with source, limits, abort, cleanup and actual results; failures retained |
| 5. Final cleanup and acceptance package | Self-model Package cut for Project, Workflow and Harness; guides, code map and generated view; provider matrix and known limits | Full local suite, Linux, Windows and race, manual CI on the frozen head; independent architecture review; owner acceptance as a separate decision |

Steps 2 and 3 do not reopen concept decisions. A newly observed contradiction becomes an open question for the owner with its evidence. Step 4 starts only after steps 2 and 3 are integrated and reviewed together.

<a id="work-streams"></a>

## Work streams

At most two writing chats at a time. Each dispatch records the exact integrated SHA, branch and worktree, dirty state, writer and exclusive paths before writing.

| Stream | Exclusive scope | Sequence |
|---|---|---|
| A: Run contract | `internal/run`, `internal/harness`, `internal/workflow`, `internal/apply`, `internal/adapters/runstore`, the Codex adapter's tool, session and harness rendering files, `internal/adapters/gitrepo/publication.go`, their tests and real-Git vertical tests in `internal/adapters/gitrepo`, the Run and Manager-collaboration passages of `docs/project-workflow.md` and `docs/architecture.md`, and the registry shards of those Packages | Step 2 first, then the D15 and D17 parts of step 3 as separate PRs |
| B: Provider scope | `internal/adapters/claudecode`, Claude parts of `internal/runtimeconfig`, `internal/bootstrap`, `internal/adapters/container` and `internal/tooling/release/source_layout.go`, provider guides, the Claude passages of `docs/architecture.md`, examples, the README provider row and their registry entries; for B1 also `internal/adapters/codexappserver/execution_profile_test.go` (provider-mismatch case only) and the Claude sentences of `docs/project-workflow.md`; for B2 also `packaging/runtime`, `internal/adapters/codexappserver/metadata.go`, `metadata_test.go`, `metadata_decode_test.go` and the runtime.env fixture in `internal/bootstrap/main_test.go` | B1 in parallel with stream A; B2 after A1, and A3 if it changes the Codex adapter, have landed, and before step 4 |

`docs/architecture.md` and `docs/project-workflow.md` are split by passage: stream A edits the Run and Manager-collaboration passages, stream B only the Claude passages. The coordinator owns the concept, this plan, the architecture document, reviews and integration, and does not edit a writer's files. Shared files such as `internal/tooling/architecture` maps, `docs/development/code-map.md`, `.markitect/model/**` and the generated view have one writer at a time, assigned in the dispatch.

## Integration

Keep the existing development policy: focused PR, meaningful local checks, independent complete-diff review, freeze the reviewed head, merge the exact head without bypassing protection, then read Main back. Recompose and review substantive overlapping changes. Automatic CI is paused; full CI is manually available, and final or release evidence remains separate. No merge waits for hosted CI by default, and absent hosted execution is never reported green.

Use isolated worktrees from verified integrated bases, non-protected scoped branches, preserve unrelated work and revalidate ownership before changes. No parallel Go commands or source edits during a gate in the same checkout. [CONTRIBUTING](../CONTRIBUTING.md#verify-a-change) owns routine checks. New or deleted artifacts require exact registry updates. Self-model changes require the official docs preview, digest and write, not manual edits of generated views.

## Owner decisions

Settled: concept 1.0 with D1–D18 ([decision record](concepts/sources/v1-concept-decisions-20261024.md)). V1 acceptance with Codex only (D18) was proposed as a recommendation and accepted with the other proposals; the owner can revise it. The [standing evidence authorization](concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization) lets the coordinator dispatch useful bounded real proofs. Release and deployment are excluded. Pending: owner acceptance after step 5 and the timing of the Claude adapter.

## Historical references

Historical work orders are outside the normal implementation entry and grant no work or execution. The S0–S5 rebuild plan and its evidence log are in [Main `334e7c31`](https://github.com/Glacius-Labs/Markitect/blob/334e7c3133ff294c0b827e623d80185e3d3ac9af/docs/implementation-plan.md) and the [V1 qualification record](validation/v1-qualification-20261024.md). The earlier integrated plan is at [source snapshot 447ac3b](https://github.com/Glacius-Labs/Markitect/blob/447ac3b152de79b52de80513e14dc94349a060bd/docs/implementation-plan.md); [Run 08](validation/apply-assignments-luna-20261023.md) retains its own scope. The [pre-cutover archive](development/v1-source-archive-20261023.md) stays verified and is not deleted.

The following compatibility anchors route historical links only:

<a id="the-target"></a>
<a id="first-vertical-acceptance"></a>
<a id="per-turn-container-rollback"></a>
<a id="container-reuse-after-the-current-evaluation"></a>
<a id="current-checkpoint-concrete-apply-corrections-and-new-core-evidence"></a>
<a id="target-and-baseline"></a>
<a id="integrated-source-and-observed-workflow-23-october-2026"></a>
<a id="sessions"></a>
<a id="branch-names"></a>
<a id="session-and-process-contracts"></a>
<a id="knowledge-graph-assessment-and-authoring-decision"></a>
<a id="standard-operating-model-checkpoint"></a>
<a id="astra-assessment-disposition-and-next-method-checkpoint"></a>
