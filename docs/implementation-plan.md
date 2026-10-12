# Roadmap

Updated 12 October 2026 against main `d083c8f37b6bb1e09a9c5a7947c421651a7e755b`, with complete [Main CI](https://github.com/Glacius-Labs/Markitect/actions/runs/38231391134). The [previous integration record](validation/integration-queue-20261011.md) closes the six-PR queue and its scoped follow-ups. The [role-round record](validation/integration-round-20261012.md) accounts for integrated TEST-03, CLI-03, CLI-04 and TEST-02, the subsequent TEST-04/ARCH-08/PLAY-06 group and the two completed Bughunter episodes. The owner has authorized continuous backlog-wave restart; the current reservations are RUN-02 and ARCH-09. The [Claude-to-Codex handoff](development/session-handoff.md) preserves the earlier interrupted-session snapshot.

This page owns the current direction: how parallel work is organized, the streams and waves, the code zones, and the open owner decisions. The [backlog](work-items/backlog.yaml) is the single owner of each work package's status. The [concept register](concepts/register.md) records the decisions this roadmap relies on. Source changes do not change a published release.

## Where we stand

**On main:**
- The model-first product: canonical model and compiler, coverage, impact, Manager execution with independent review, verification and guarded Apply.
- The [concept record](concepts/README.md).
- Exchangeable process execution and per-role configuration (RUN-01, #104), the top-level verb table (CLI-02, #161), and the reduced knowledge core and explain/trace surface (KG-02, #184).
- The case playground, provider-free container smoke, shared source/bundle smoke driver, hermetic test kit and Linux race gate.
- Markitect's own recursive Manager model and generated project view (ARCH-07, #203), stable view (#216) and verified short-name contributor entrypoint (#222).
- Read-only accepted briefing history (#205), guided previews (#221), structured private child assessment filtering (#223), smaller typed MCP schemas (#218), and the reviewed pinned-image inventory (#219).
- Historical-context projection of private briefing event details (#225), read-only Context/list callers with atomic accepted-history dismissal (#229), LF-stable Windows playground fixtures (#228), delegated DEC-023/hypothesis provenance (#226) and the prepared KG-04 protocol (#227). KG-04 measurement remains NOT RUN.
- TEST-03's shared isolated process fixtures and parallel E2E selection (#232) retain all default-suite assertions; CLI-03's application composition/read facade and CLI boundary/size check (#231) preserve the briefing semantics above.
- CLI-04's Doctor Git-version prerequisite (#235) reports unsupported Git before model or provider inspection and retains CLI/MCP diagnostics without project writes.
- TEST-02's complete outer CLI/MCP contract matrix (#237) covers help/schema, rejected arguments, successful operations and missing-write authorization, with explicit surface completeness and target isolation before Apply.

**Remaining older product line:** The canonical projection alpha has been removed (ARCH-04). The Project/Domain line and its explicit legacy gates remain until ARCH-09; distribution separation is ARCH-08. Current contributor instructions, CI and hooks also check the self-hosted model. Do not reuse the old survey's code-size estimate as a current measurement.

**CI:**
- Linux quality and race jobs are the required PR gates. The ordinary suite runs once per quality job; fixed-revision dogfood does not repeat it.
- Windows is nightly, manual and release validation; it does not block ordinary merges (DEC-013).
- The original six PR heads and their scoped follow-ups passed required Linux Quality/Race and relevant provider-free smoke. Main CI is tracked separately for each merge; exact candidate results and local Windows failures are recorded in the integration record. They do not establish provider or method-study acceptance.
- Freeze each PR head while its required Linux checks run. Recheck the current head and compile its merge onto current main before integration.

**Open work outside main:**
- bounded investigation of the recorded Windows Git timeout and workspace alias failure, retaining their original source and diagnostic limits;
- KG-04 measurement after its prepared Shop/Markitect protocol: independent expected lists remain unfrozen and execution needs separate authority;
- historical Government and research candidates, preserved as references. They are not combined with current main or scheduled ahead of the backlog's gates.

**Product Readiness program:** closed. P01 to P10 are complete. Its records were distilled into the [Product Readiness lessons survey](work-items/surveys/product-readiness-lessons-20261010.md) and the [A01 validation record](validation/a01-native-smoke-20261010.md); git history keeps the originals. Its open follow-ups now live in the backlog: RUN-04 replaces P11, AGENT-03 replaces R03-F01, and the KG packages replace KG01 to KG03.

## Guiding decisions

- **A clean, stable and testable main comes first.**
- **Compatibility is not a driver.** Nobody else uses Markitect yet ([DEC-014](concepts/register.md#dec-014-compatibility-does-not-drive-decisions)).
- **Linux first for tests and the playground.** Windows support is revisited later ([DEC-013](concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)).
- **The method is evaluated separately from any single runtime, and the product supplies the method** ([DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime)).
- **Earlier plans and branches are inputs, not the plan.** Their ideas are checked against current main and the vision first (IDEA-01).

## How we work

### Sessions

A session is a long-running chat for one role. The backlog's `owner` field names the role, and any fresh session can take a role over. After the previous queue closed, the owner requested fresh sessions on 12 October. QA delivered TEST-03 and preserved its separate failed Windows diagnostic run; Interface delivered CLI-03, CLI-04 and TEST-02; its assigned implementation round is complete. Bughunter completed its first bounded read-only round. The existing integrator owns canonical status and integration. The interrupted Claude sessions' snapshot is in [Session continuation](development/session-handoff.md); the [branch survey](work-items/surveys/branches-and-worktrees-20261010.md) preserves the earlier archive inventory. A package's `active` state is not evidence that its old chat is still running.

On 12 October the owner instructed: **“Beginne die Wellen selbstständig neu.”** The integrator continuously selects dependency-ready work within accepted roadmap intent, explicitly assigns its scope and zone, coordinates independent review and normal integration, and verifies complete Main CI at the actual full merge SHA. After a package or group closes, it rechecks the DAG and starts the next eligible assignment through existing role chats, or a new role chat when needed. A green intermediate Main is a checkpoint, not a reason to wait for another owner session-start request. `proposed` alone means written scope not yet started; it neither bans assignment nor grants product approval. Intent, `needs_owner`, parked work, credentials/cost, provider/study execution, publication and cleanup retain their actual gates. Unaccepted helper recovery remains a proposal. Prior reviews are reused only for unchanged exact sources; fresh changed candidates get their own review.

| Role | Streams | Starts |
|---|---|---|
| integrator | OPS, register decisions, review and merge order, IDEA-01, LEARN-01 | Coordinate current reservations, independent reviews and normal PR integration; verify final Main |
| interface | CLI, MCP-01, TEST-02, AGENT-02, AGENT-03 | CLI-03, CLI-04 and TEST-02 integrated; no current implementation reservation, later packages require their own assignment |
| scientist | PLAY | PLAY-06 integrated with complete actual-Main CI; writer released, no protocol-writing or study scope while PLAY-07/CI-04 owner gates remain |
| kg | KG | KG-04 protocol is prepared; measurement waits for separate authority |
| bughunter | BUG | Both bounded read-only episodes complete; new Windows lock cause unisolated, P2 test cleanup issue recorded, no fixwriter; helper recovery remains unaccepted |
| qa | CI, TEST | TEST-04 integrated with complete actual-Main CI; the existing chat now holds the separate Runtime role, no concurrent QA/TEST-05 writer |
| cleaner | ARCH, AGENT-01 | ARCH-08 integrated; ARCH-09 active on `dev/legacy-removal` in `legacy-removal`, started from Main D083; exclusive Legacy scope and coordinated checker/module/model/CI crosspaths |
| runtime | RUN, MCP-02, ARCH-06 | RUN-02 active on `dev/asynchronous-executor` in `asynchronous-executor`, started from Main D083 after verified TEST-04 integration; existing chat reused, no duplicate writer |
| government | GOV | After Wave 2 and the explicit GOV-01 dependencies (DEC-019) |

No more than six sessions change code at the same time. Each session uses subagents and workflows inside its own packages as it sees fit.

The first autonomous group is integrated: ARCH-08 as `4b41d29a219836472e17a50fb17fa5d5e98ce2cd`, TEST-04 as `f7827e4ff7708bedfbfe872a94f6c519829323c8`, and PLAY-06 as current Main `d083c8f37b6bb1e09a9c5a7947c421651a7e755b`. Each actual merge has complete successful push Quality/Race/source-artifact CI. The [integration record](validation/integration-round-20261012.md#autonomous-wave-and-next-reservations) keeps literal heads, compositions, tested hosted checkouts and actual Main results distinct. The separate scheduled Windows run on F782 failed a process-lock test; independent read-only diagnosis leaves its primary cause unisolated and confirms a test failure-cleanup issue. It does not change the successful push-run verdict or establish Windows acceptance.

Two implementation writers are now assigned: Runtime RUN-02 and Cleaner ARCH-09, each from full Main D083 in a verified fresh checkout. RUN-02 stays within the endorsed asynchronous executor boundary and bounded public surfaces; ARCH-09 removes the precise legacy inventory while retaining current product/Shop/method harness and historical data. The shared legacy artifact manifest is edited only for Runtime's exact existing-package files on its branch and migrated/retired only after equivalent current-model checks on the Legacy branch; the integrator composes both with the then-current schema. Cleaner owns shared codemap/code-map/model/CI migration. No required-coverage waiver, placeholder or import exception is allowed. Backlog, roadmap, register and source-evidence records remain integrator-owned. The completed TEST-04 real-kill fixture proves blocked/uncertain/no-replay, not positive continuation or new recovery. A possible test-only Windows follow-up has no concurrent writer reservation. Provider or method studies, KG-04 measurements, Government, publication and live worktree/resource cleanup retain their gates.

### Starting a session

Start every session with the same prompt; only the role changes:

For this restart, also read the [role handoff and initial queue](development/session-handoff.md#role-starts). Reuse an existing candidate worktree only after verifying its branch, full SHA, local changes and ownership; use a fresh branch from current main for new packages. No new chat is dispatched by this document itself.

```text
You are the <role> session for Markitect. Read current AGENTS.md, docs/implementation-plan.md, docs/work-items/backlog.yaml and docs/development/session-handoff.md, including your role's initial queue. Fetch origin and identify the checkout path, branch, full SHA and local changes before writing. Resume an existing candidate only after confirming ownership; start new packages from current origin/main. Read the owning Manager's context and each package's inputs and survey. Work in dependency order inside the integrator-assigned zone, with one PR per package or slice and its ID in the title. Freeze a pushed head while required Linux CI runs; fixes may be pushed when only non-required Windows CI remains. The integrator alone updates the backlog, roadmap and concept register; other roles report evidence and decisions in their PRs. Use subagents for independent workstreams. Preserve uncommitted work, never stash or force-push a reviewed branch, and distinguish source checks, provider runs, method studies and human acceptance. Ask the owner only for decisions outside existing authority.
```

The integrator merges pull requests in dependency order, updates the backlog and the register, and assigns zones.

### Work packages

Each package has:
- an ID made of its stream and a number;
- one owner session and one branch;
- a zone;
- dependencies and acceptance criteria.

The [backlog](work-items/backlog.yaml) holds all of them. A package is done when it is merged to main and its documentation and register links are updated.

### Branch names

New branches start from current `origin/main`. Each has a meaningful lower-case name and a prefix that says what kind of work it holds:

| Prefix | Use | Example |
|---|---|---|
| `dev/` | Development: features, refactoring, tests, CI | `dev/linux-ci-gate`, `dev/verb-table` |
| `fix/` | A bug fix | `fix/impact-order-independence` |
| `docs/` | Documentation only | `docs/roadmap` |
| `exp/` | A throwaway experiment | `exp/graph-explain-spike` |
| `archive/` | A preserved old branch or a snapshot of work in progress | `archive/codex/knowledge-graph` |

- The name says what the work is.
- It contains no tool name (`codex/`, `claude/`), no date, no version suffix and no number.
- The package ID goes into the pull-request title and into the backlog's `branch` field, which links the two.
- Branches with the old `codex/` and `claude/` prefixes are earlier work. They are archived step by step (OPS-03).

### Zones

A zone is a code area in which only one active package changes code at a time. The integrator assigns zones; a package that waits for its zone is blocked.

| Zone | Paths | Packages |
|---|---|---|
| runtime | `src/internal/host/{projectrun,projectsetup,agentexec,codexappserver,projectworkspace}` and the exchange executor | RUN, TEST-03 to TEST-05, ARCH-05, ARCH-06 |
| interface | `src/internal/host/{cli,projectcli,mcp,projectapp,projectonboarding}` | CLI, MCP-01, TEST-02, AGENT-02, AGENT-03, ARCH-08 |
| model | `src/internal/core`, `src/internal/modules/projectmodel`, `src/internal/host/{projectwork,projectcoverage,projectbriefing}` | KG, GOV-01, IDEA-02 to IDEA-04 |
| legacy | the host root package, `compat`, canonical, records, authoring and related packages | ARCH-02, ARCH-04, ARCH-09 |
| ci | `.github/workflows`, `markitect.yaml`, `scripts`, `.githooks` | CI, ARCH-07 |
| docs-entry | `AGENTS.md`, `README.md`, `docs/README.md`, this page, the backlog | OPS, AGENT-01 |
| free | everything else, including new packages, `experiments/case-playground` and new documents | the rest |

### Integration

1. **One pull request per package or slice.** Its title starts with the package ID.
2. **The PR head is frozen while the required Linux CI runs.** Nobody pushes to it during that run, and evidence goes into the PR description. Fixes may be pushed while a run that is not required (Windows) is still in progress, because only Linux is required ([DEC-013](concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)).
3. **Linux CI is the required gate** (DEC-013). The integrator merges in dependency order, one pull request at a time, with merge commits.
4. **The backlog stays current.** After each merge, the package's status is updated in the same or the next integrator pull request.
5. **Decisions come first.** A decision that changes direction is recorded in the register before code relies on it.
6. **Compile before merge.** The integrator first compiles each pull request merged onto current main (`go vet ./...`); two pull requests that pass alone can still break main together.
7. **Stacked pull requests target main.** A pull request whose base is another branch is retargeted to `main` before it merges.
8. **Worktree hygiene.** Never use `git stash`, because it is shared across worktrees; use patch files. Check `git show --stat` before every push. [CONTRIBUTING](../CONTRIBUTING.md#pull-requests-and-parallel-work) has the full rules.

## Waves

### Wave 0: secure and unblock

- OPS-01: this roadmap.
- OPS-02: secure work that exists only on this machine.
- OPS-04: contribution rules.
- RUN-01: executor increment 1 (pull request #92).
- PLAY-01: bring the playground onto main.
- CI-01: quick fixes.
- Owner decisions CLI-01, KG-00 and GOV-00.

### Wave 1: foundations, in parallel

- CI-02: Linux-first pipeline.
- TEST-01: hermetic test kit.
- Architecture:
  - ARCH-01: code map.
  - ARCH-02 and ARCH-03: separate the product from the legacy package.
  - ARCH-04: remove the canonical alpha.
  - ARCH-10: documentation per module.
- AGENT-01: slim agent entry files.
- PLAY-02 to PLAY-05: ground truth, architecture page, parameters, hardening.
- BUG-01: bug hunt.
- IDEA-01 and LEARN-01: research inputs.
- MCP-02: App Server path review.
- KG-01: only if KG-00 says go.
- OPS-03: archive branches.

### Wave 2: one interface, a clean runtime

- **Interface zone:** CLI-02, CLI-03, MCP-01, CLI-04, TEST-02, AGENT-02 and ARCH-08 are integrated. Cleaner now owns ARCH-09 under a separate Legacy reservation.
- **Runtime zone, remaining sequence:** RUN-02 → ARCH-05 → ARCH-06 after integrated TEST-03/TEST-04. The existing QA chat now owns RUN-02 as Runtime; the sequence stays serialized. PLAY-06 is also integrated. TEST-05 needs an explicitly documented later slot or nonconflicting slice; RUN-05/RUN-06 are separate packages with their own risk and reservation checks.
- **Also in this wave:**
  - CI-03: one smoke driver.
  - CI-04: container runtime tier.
  - PLAY-06: per-role models in studies.
  - KG-02.

### Wave 3: Markitect on Markitect, legacy removed

- ARCH-07: Markitect uses its own model and current verbs.
- ARCH-08: release tooling separated.
- ARCH-09: the Project/Domain line removed.
- RUN-03: Claude Code as an inner executor.
- GOV-01: Government code after the interface/runtime conditions in [DEC-019](concepts/register.md#dec-019-government-code-starts-after-wave-2) and its actual backlog dependencies; the old evaluation and dirty v1 basis remain unaccepted inputs.
- AGENT-03 remains parked in the backlog; wave placement does not activate it.
- CI-05 remains parked for the owner's Windows-support decision.

### Wave 4: evidence and extensions

These wait for the conditions in [DEC-011](concepts/register.md#dec-011-detailed-planning-of-future-ideas-waits), a stable version and case-study numbers, unless the owner decides otherwise:
- PLAY-07: first method study over a change series.
- KG-03: the knowledge graph as a variant.
- GOV-02; GOV-01's code-start conditions are separately governed by DEC-019.
- IDEA-02 to IDEA-04: the register's enhancements ENH-001 to ENH-003.
- Follow-ups selected from LEARN-01.

## Owner decisions

Markitect's product owner approved the roadmap and its recommendations on 10 October 2026:
- [DEC-017](concepts/register.md#dec-017-command-model-top-level-verbs-from-one-verb-table): commands.
  - Top-level verbs come from one verb table, and the `project` noun is dropped.
  - One brownfield path remains, as `adopt` stages.
  - `--write` and `--execute` are separate.
  - `status` shows a project overview.
  - MCP gets a read-only mode.
  - Release tooling lives in its own binary.
- [DEC-014](concepts/register.md#dec-014-compatibility-does-not-drive-decisions): the legacy tree is removed, tools may be renamed, and no aliases are kept.
- [DEC-018](concepts/register.md#dec-018-the-knowledge-graph-explains-impact): the knowledge graph explains impact. A reduced core is ported, and the existing MCP surface is used.
- [DEC-019](concepts/register.md#dec-019-government-code-starts-after-wave-2): Government code starts after Wave 2; design work may continue.
- OPS-02 and OPS-03: earlier work is preserved as archive references, and the archive list is approved.

Decisions still to come are listed per package as `needs_owner` in the backlog, for example the protocol of the first method study (PLAY-07).

## Releases and historical evidence

The latest published release is v0.14.1. It preserves the earlier Project/Domain CLI and an experimental canonical Projection alpha; it does not include the current model-first project workflow. Source changes do not change a pinned release. See the [production assessment](production-assessment.md) for exact release assets and evidence.

The prior roadmap, including dated release summaries, proof dispositions, architecture assessments, and their original pinned references, is preserved in the [dated historical roadmap](history/implementation-plan-before-product-integration-20261009.md). Use its linked validation reports for evidence details; those records are not current work status.

## Historical roadmap links

These retained anchors route existing references to the corresponding dated historical sections.

<a id="astra-assessment-disposition-and-next-method-checkpoint"></a>
The [Astra disposition](history/implementation-plan-before-product-integration-20261009.md#astra-assessment-disposition-and-next-method-checkpoint) is historical; current work lives in the [backlog](work-items/backlog.yaml).

<a id="standard-operating-model-checkpoint"></a>
The [operating-model checkpoint](history/implementation-plan-before-product-integration-20261009.md#standard-operating-model-checkpoint) is historical; current work lives in the [backlog](work-items/backlog.yaml).

<a id="read-only-policy-failure-analysis-unreleased-source"></a>
The [policy-failure analysis](history/implementation-plan-before-product-integration-20261009.md#read-only-policy-failure-analysis-unreleased-source) is historical.

<a id="v0100--canonical-engineering-model-published"></a>
The [v0.10.0 release record](history/implementation-plan-before-product-integration-20261009.md#v0100--canonical-engineering-model-published) is historical.

<a id="knowledge-graph-assessment-and-authoring-decision"></a>
The [knowledge-graph assessment](history/implementation-plan-before-product-integration-20261009.md#knowledge-graph-assessment-and-authoring-decision) is historical; current knowledge-graph work is KG-00 to KG-03 in the [backlog](work-items/backlog.yaml).
