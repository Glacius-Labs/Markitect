# Roadmap

Updated 10 October 2026, on main `02c7e529`, after pull requests #89 to #91.

This page owns the current direction: how parallel work is organized, the streams and waves, the code zones, and the open owner decisions. The [backlog](work-items/backlog.yaml) is the single owner of each work package's status. The [concept register](concepts/register.md) records the decisions this roadmap relies on. Source changes do not change a published release.

## Where we stand

**On main:**
- The model-first product: canonical model and compiler, coverage, impact, Manager execution with independent review, verification and guarded Apply.
- The [concept record](concepts/README.md).

**Still dependent on older product lines:** About half of the production Go code serves the earlier Project/Domain line and the canonical projection alpha. Markitect's own CI, hooks and agent instructions still run on them.

**CI:**
- Each job runs the Go suite twice.
- The Windows job takes 50 to 70 minutes.
- New pushes cancel runs on main.

**Open work outside main:**
- the exchangeable executor (pull request #92);
- the rebuilt case playground (a local branch);
- the knowledge-graph branch (old layout);
- a started Government v1 (docs plus uncommitted code).

**Product Readiness program:** P01 to P10 are complete. Its records stay in [work-items/product-readiness](work-items/product-readiness/README.md) as history. Its open follow-ups now live in the backlog: AGENT-03 replaces R03-F01, and the KG packages replace KG01 to KG03.

## Guiding decisions

- **A clean, stable and testable main comes first.**
- **Compatibility is not a driver.** Nobody else uses Markitect yet ([DEC-014](concepts/register.md#dec-014-compatibility-does-not-drive-decisions)).
- **Linux first for tests and the playground.** Windows support is revisited later ([DEC-013](concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)).
- **The method is evaluated separately from any single runtime, and the product supplies the method** ([DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime)).
- **Earlier plans and branches are inputs, not the plan.** Their ideas are checked against current main and the vision first (IDEA-01).

## How we work

### Sessions

A session is a long-running chat for one role. The backlog's `owner` field names the role, and any fresh session can take a role over. The earlier sessions have ended. Their work is preserved and listed in the [branch survey](work-items/surveys/branches-and-worktrees-20261010.md).

| Role | Streams | Starts |
|---|---|---|
| integrator | OPS, register decisions, review and merge order, IDEA-01, LEARN-01 | running |
| qa | CI, TEST | Wave 0 |
| cleaner | ARCH, AGENT-01 | Wave 0 |
| runtime | RUN, MCP-02, ARCH-06 | Wave 0 |
| scientist | PLAY | Wave 0 |
| interface | CLI, MCP-01, TEST-02, AGENT-02, AGENT-03 | Wave 0 (specification first) |
| bughunter | BUG | Wave 1 |
| kg | KG | Wave 1 |
| government | GOV | after Wave 2 (DEC-019) |

No more than six sessions change code at the same time. Each session uses subagents and workflows inside its own packages as it sees fit.

### Starting a session

Start every session with the same prompt; only the role changes:

```text
You are the <role> session for Markitect, working in this repository. Read AGENTS.md, docs/implementation-plan.md (the roadmap) and docs/work-items/backlog.yaml. Work on the packages whose owner is <role>, in wave and dependency order, starting with those marked ready. For each package, first read its inputs and the matching survey under docs/work-items/surveys/, so you do not repeat earlier analysis. Follow the roadmap's rules: a branch with a meaningful prefix from current origin/main, changes only inside the package's zone, one pull request per package with the package ID in the title, and no pushes to a pull-request head while its required Linux CI runs (fixes may be pushed while only the Windows job is still running). Do not edit the backlog, the roadmap or the concept register; report status, findings and needed owner decisions in the pull request. Use subagents and workflows inside your packages as you see fit. Ask the owner when a decision is genuinely theirs.
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
3. **Linux CI is the required gate** (DEC-013). The integrator merges in dependency order with merge commits and deletes merged branches.
4. **The backlog stays current.** After each merge, the package's status is updated in the same or the next integrator pull request.
5. **Decisions come first.** A decision that changes direction is recorded in the register before code relies on it.

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

- **Interface zone, in sequence:** CLI-02 → CLI-03 and MCP-01 → TEST-02 → AGENT-02.
- **Runtime zone, in sequence:** TEST-04 → RUN-02 → TEST-03 → ARCH-05 → ARCH-06. TEST-05 runs alongside where it does not conflict.
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
- AGENT-03.
- CI-05: revisit Windows.

### Wave 4: evidence and extensions

These wait for the conditions in [DEC-011](concepts/register.md#dec-011-detailed-planning-of-future-ideas-waits), a stable version and case-study numbers, unless the owner decides otherwise:
- PLAY-07: first method study over a change series.
- KG-03: the knowledge graph as a variant.
- GOV-01 and GOV-02.
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
