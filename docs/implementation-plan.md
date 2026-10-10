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

| Session | Role | Streams now |
|---|---|---|
| Integrator (Claude) | Roadmap and backlog, register decisions, review and merge order, repository hygiene | OPS |
| Runtime (Claude) | Exchangeable executor | RUN |
| Scientist (Claude) | Case playground and studies | PLAY |
| Cleaner (Codex) | Code and documentation cleanup | ARCH |
| QA (new) | Pipeline, test kit, runtime tests | CI, TEST |
| Bug hunter (new) | Verified findings and small fixes | BUG |
| Interface (new, after CLI-01) | One verb table for CLI and MCP | CLI, MCP, AGENT-02, TEST-02 |
| Government (Codex) | Paused until GOV-00 | GOV |

No more than six sessions change code at the same time. Research packages without code (IDEA-01, LEARN-01) can run alongside.

### Work packages

Each package has:
- an ID made of its stream and a number;
- one owner session and one branch;
- a zone;
- dependencies and acceptance criteria.

The [backlog](work-items/backlog.yaml) holds all of them. A package is done when it is merged to main and its documentation and register links are updated.

### Branch names

- New work uses `wp/<id>-<slug>` in lower case, for example `wp/ci-01-quick-fixes`. It starts from current `origin/main`. Name the worktree folder after the package ID.
- Tool prefixes such as `codex/` and `claude/` are not used for new work. Branches opened before this roadmap keep their names until they are merged.
- Throwaway experiments use `exp/<slug>`. An urgent fix outside a package uses `fix/<slug>`.
- Old branches become `archive/<name>` tags before they are deleted (OPS-03).

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
2. **The PR head is frozen while CI runs.** Nobody pushes to it; evidence goes into the PR description.
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

## Open owner decisions

| Decision | Question | Blocks | Recommendation |
|---|---|---|---|
| CLI-01 D2 | Drop the `project` noun? | CLI-02 and the rest of Wave 2's interface zone | Yes: top-level verbs such as `markitect plan` and `markitect apply` |
| CLI-01 D3 | Which brownfield path stays? | CLI-02 | One path as `adopt` stages; remove the other |
| CLI-01 D5 | Separate `--write` (persist) from `--execute` (start agents)? | CLI-02 | Yes |
| CLI-01 D7 | What does `status` show? | CLI-02 | A project overview by default; `status RUN` for one run |
| CLI-01 D8 | Read-only MCP mode or separate preview tools? | MCP-01 | A read-only server mode |
| CLI-01 D9 | Where does release tooling live? | ARCH-08 | The separate release binary |
| KG-00 | Role and form of the knowledge graph | KG-01 onward | An explanation of impact; port a reduced core; use the existing MCP surface |
| GOV-00 | Start Government v1 now or later; who owns model schema changes | GOV-01, model zone | Keep designing; start code after the interface and runtime zones settle in Wave 2 |
| OPS-02 | Where unpushed branches go | OPS-03 | Push to origin under an `archive/` prefix |
| OPS-03 | Approve the archive list | Repository hygiene | Approve the generated list |

[DEC-014](concepts/register.md#dec-014-compatibility-does-not-drive-decisions) already settles three earlier questions:
- D1: the legacy command tree is removed.
- D4: MCP tools may be renamed.
- D6: no compatibility aliases are kept.

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
