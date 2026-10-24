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
| D10–D13: hierarchical questions, child diagnostics at the parent, assignment-bound judgments, tasks not process | Resolved in source by A1 ([PR #683](https://github.com/Glacius-Labs/Markitect/pull/683)) with controlled tests and a bounded [native contract proof](validation/a1-native-contract-proof-20261024.md) on f76826c3; the later wording delta of host messages is requalified in step 4 | 2 |
| D15: closed-result validation in the Run contract | `workflow.ValidateClosed` checks Run resource names; `apply.ValidateApproval` calls it | 3 |
| D17: bounded, ordered Notes pages within the transport limit | `read_notes` returns all Notes in one tool reply ordered by random ID; the store accepts 16 MiB against a 1 MiB frame; a read error cancels the Run | 3 |
| D18: no non-functional Claude path | Resolved in source: B1 ([PR #674](https://github.com/Glacius-Labs/Markitect/pull/674)) removed the Claude adapter and its wiring, B2 ([PR #678](https://github.com/Glacius-Labs/Markitect/pull/678)) removed Claude Code from the runtime image recipe and the `claude=` metadata record; the changed image is built and qualified in step 4 | 3 |
| §5.5: the self-model describes the chosen architecture | Project, Workflow and Harness are owned by the `product/apply` and `product/adapters/agent` Packages; the Run Rule now states D10–D12 ahead of the code | 5 |

Documentation language ([DEC-069](concepts/register.md#dec-069-documentation-is-written-in-english)): [PR #679](https://github.com/Glacius-Labs/Markitect/pull/679) translated the current canonical-model documents, including the normative [language specification](design/canonical-model/minimal-language/language-contract.md) and the exchange example, the archived [conformance audit](development/concept-conformance-audit-20261024.md) and, as a new reference translation, [concept v0.18](concepts/sources/v1-concept-v018-20261024-en.md). Three groups are still German. First, the historical design records under `docs/design/project-world`, `docs/design/product-model-refinement`, `docs/design/work-item-comparison-20261009.md`, `docs/design/task-management-model` and `docs/validation/astra-assessment-2026-10-06.md`; R2 archives them, and archived records are not translated ([DEC-071](concepts/register.md#dec-071-repository-structure-and-archive)). Second, the digest-pinned probe fixtures under `docs/design/canonical-model/minimal-language`, handled as the [repository structure](development/repository-structure.md) document classifies them. Third, the `markitect/core` vocabulary bodies in `internal/model/core.go` and `.markitect/model/markitect/core` and the examples that copy them, translated in step 3 ([owner decision](concepts/sources/english-documentation-20261024.md#follow-up-owner-decision-core-vocabulary)); as recorded there, this is planned as a new core version with unchanged meaning. Two links in deferred records still point at old German headings of the consolidation page and are fixed when those records are translated or archived.

Product guides such as the [architecture](architecture.md), the [project workflow](project-workflow.md) and the adapter READMEs describe implemented behavior. They change together with the code in steps 2 and 3.

## Technical decisions from implementation

Decisions taken within concept 1.0 while implementing; they do not change product semantics.

- **Bounded no-progress stop (A1):** an open assignment is continued with a message that names the task. After two continued responses in a row with no Run tool action and no change of the working state, the Run ends technically incomplete with a diagnostic that names the Manager. This keeps concept §13.2 – a single response end is never a failure and never demands a judgment – while honoring §20 finite limits and honest results.
- **`judge` is a Run tool (A1):** like every other Manager tool, its semantics live in Run (concept §14.2); the provider adapter only maps it.
- **Failure classes (A1):** port or provider failures that stop a Run are reported as failed; deadline, budget, observation and no-progress stops stay incomplete (concept §20.1).
- **Review mandates (A1):** Assessment and Verify child handoffs carry no check result; reviewers run `check` themselves when they need one.
- **Host messages (A1):** every refusal and diagnostic names the concrete reason and a next step that the tools accept (concept §3.7).

<a id="apply-rebuild-plan"></a>

## Steps

| Step | Result | Completion evidence |
|---|---|---|
| 1. Record concept 1.0 | Concept, owner sources, decision record and register entry; entry documents, this plan, the architecture and the self-model Run Rule aligned; evidence moved to the qualification record | Structural check, documentation links, generated view and independent complete-diff review |
| 2. Run collaboration contract | D10–D13: hierarchical questions, child diagnostics at the parent, judgment bound to assignment and assessed state with an explicit host-side state, the `check` tool and a fixed tool set; harness rules and host messages reduced to role, rules, context, task and answer format | Controlled-session tests for every counterexample in concept §13.3 and for V03, V05 and V17; one bounded native Codex contract proof of the changed interaction before the contract is fixed; existing real-Git vertical tests adjusted, not weakened; Run, harness and Codex adapter suites; architecture and accounting gates |
| 3. Remaining source changes before requalification | D15 closed-result validation in the Run contract; D17 bounded Notes pages after a controlled reproduction of the size conflict; D18 removal of the Claude adapter and its wiring, with guides and examples (B1), then Claude Code removed from the runtime image and the Codex metadata parser (B2); the `markitect/core` vocabulary in English as a new core version with unchanged meaning (C1); the repository structure: inventory and target layout (R1), then its execution (R2) | Focused tests including a stored-Notes case above one transport frame; full build, vet, architecture and accounting; no remaining Claude product path; model, snapshot, organization and realization suites with updated core identity and vectors, and an independent check that every translated body keeps its meaning; for R2 the target layout of R1, no broken links or anchors, complete accounting, source packaging and the full build and test suite unchanged in behavior |
| 4. Native requalification (Codex) | Frozen source after steps 2 and 3: fresh M1, M2 5 → 3, strict no-change, seeded defect, the public Verify Reject → fresh Implement → fresh Verify → publication branch and outside oracles | Bounded dispatches under the standing evidence authorization with source, limits, abort, cleanup and actual results; failures retained |
| 5. Final cleanup and acceptance package | Self-model Package cut for Project, Workflow and Harness; guides, code map and generated view; provider matrix and known limits | Full local suite, Linux, Windows and race, manual CI on the frozen head; independent architecture review; owner acceptance as a separate decision |

Steps 2 and 3 do not reopen concept decisions. A newly observed contradiction becomes an open question for the owner with its evidence. Step 4 starts only after steps 2 and 3 are integrated and reviewed together.

## Repository structure

The owner wants a repository that is state of the art, tidy and well organized. Two slices in step 3 deliver it, before the native requalification, so that step 4 qualifies the final layout. R1 is done ([PR #680](https://github.com/Glacius-Labs/Markitect/pull/680), [repository structure](development/repository-structure.md)), and the owner accepted its six recommendations ([DEC-071](concepts/register.md#dec-071-repository-structure-and-archive)). The ruleset protecting `archive/**` refs is active.

- **R1 – inventory and target layout (read-only, Opus):** classify every top-level directory and every document and experiment as current, archive or delete. Today `docs/` holds about 540 files, about 270 of them design records, and `experiments/` about 500 files. R1 also proposes a target layout: where fixtures, examples, packaging, tooling commands and scripts live, how the documentation is grouped, and how archived material stays traceable, for example through a durable archive ref plus an index page instead of files on Main. It also writes a one-page repository guide and reconciles or replaces the existing [repository layout](repository-layout.md) page, so that there is one repository guide. The owner sees the target layout before R2 moves anything. R1 can start at any time because it changes nothing, and its result also tells the documentation translation which historical records still need translating.
- **R2 – execution (writer, Sonnet):** apply the work list of the repository structure document mechanically: archive point and refs, moves, archival, deletions, registry and self-model paths including the changed tag Rule, code map, source packaging, CI, links and anchors. Behavior stays unchanged. It runs directly after A1 has landed, alone in the affected paths, and before C1.

Two current layout choices are deliberate:

- **No `src/` folder.** Markitect is a Go module, and Go places `cmd/` and `internal/` at the module root; a `src/` folder is not idiomatic Go. R1 keeps the Go layout and tidies the rest of the top level around it.
- **`.markitect/` in Markitect's own repository.** It holds Markitect's canonical self-model, registry shards and ignore rules. Markitect develops itself model-first with its own product, and every file in the repository is accounted for there (concept §5.5, §6). R1 keeps it and explains it in the repository guide.

<a id="work-streams"></a>

## Work streams

Writers run in parallel as far as is useful and their exclusive scopes are disjoint ([DEC-070](concepts/register.md#dec-070-parallel-writers-where-useful)). Each dispatch records the exact integrated SHA, branch and worktree, dirty state, writer and exclusive paths before writing.

| Stream | Exclusive scope | Sequence |
|---|---|---|
| A: Run contract | `internal/run`, `internal/harness`, `internal/workflow`, `internal/apply`, `internal/adapters/runstore`, the Codex adapter's tool, session and harness rendering files, `internal/adapters/gitrepo/publication.go`, their tests and real-Git vertical tests in `internal/adapters/gitrepo`, the Run and Manager-collaboration passages of `docs/project-workflow.md` and `docs/architecture.md`, and the registry shards of those Packages | Step 2 first, then the D15 and D17 parts of step 3 as separate PRs |
| B: Provider scope | `internal/adapters/claudecode`, Claude parts of `internal/runtimeconfig`, `internal/bootstrap`, `internal/adapters/container` and `internal/tooling/release/source_layout.go`, provider guides, the Claude passages of `docs/architecture.md`, examples, the README provider row and their registry entries; for B1 also `internal/adapters/codexappserver/execution_profile_test.go` (provider-mismatch case only) and the Claude sentences of `docs/project-workflow.md`; for B2 also `packaging/runtime`, `internal/adapters/codexappserver/metadata.go`, `metadata_test.go`, `metadata_decode_test.go` and the runtime.env fixture in `internal/bootstrap/main_test.go` | B1 and B2 done |
| C: Core vocabulary | `internal/model/core.go` with `core_fixture_test.go` and `canonical_core_vector_test.go`, `internal/organization/core_fixture_test.go`, `.markitect/model/markitect/core`, `.markitect/README.md`, the generated model view, under `docs/design/canonical-model/minimal-language` the files `exchange-example/canonical/core.json`, `exchange-example/dependencies/core/**`, `bootstrap/core.yaml`, `identities/core-1.0.0.yaml` (and its successor) and `identities/probe.json`, and tests that load the exchange-example core and pin its digest (`internal/project`, `internal/bootstrap`, `internal/adapters/gitrepo`, `internal/adapters/codexappserver`) | C1 after A1 and R2 have landed, on the paths R2 leaves (for example `testdata/`); before step 4; it runs when no other writer touches its files |
| R: Repository structure | R1 reads everything and writes only its inventory and target-layout document; R2 owns the moved paths, `.markitect/registry/**`, the self-model paths, `docs/development/code-map.md`, `internal/tooling/release/source_layout.go` and link updates, as assigned in its dispatch | R1 now, in parallel; R1 done; R2 directly after A1 has landed, before C1, A2, A3 and step 4 |

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
