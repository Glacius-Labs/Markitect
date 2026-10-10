# Verb table

Specification for backlog CLI-01, 10 October 2026, against main `a98921cc`. It implements register [DEC-017](../concepts/register.md#dec-017-command-model-top-level-verbs-from-one-verb-table) and the command consequences of [DEC-014](../concepts/register.md#dec-014-compatibility-does-not-drive-decisions). Its input is the [CLI, MCP and App Server survey](../work-items/surveys/cli-mcp-app-server-20261010.md).

This page is the target contract. CLI-02 implements the verbs, the tools and the `markitect-legacy` split; CLI-03, MCP-01, TEST-02, AGENT-02 and ARCH-08 in the [backlog](../work-items/backlog.yaml) complete it. [Usage](../usage.md) and the [Project workflow](../project-workflow.md) describe the commands as they exist.

## Summary

- `markitect project <action>` becomes `markitect <verb>`. The `project` noun is gone, and no aliases remain.
- One verb table in code drives CLI help, CLI dispatch and MCP registration.
- An MCP tool has the verb's name. Its fields are the verb's arguments: a flag `--some-name` is the field `someName`.
- `--write` only persists. `--execute` starts agents or the project's configured checks.
- Every write follows preview, then digest, then `--write --expect DIGEST`. Executions bind to a persisted record instead, such as the plan that `run` executes.
- `status` shows a project overview; `status RUN` shows one run.
- `mcp --read-only` serves a read-only tool set.
- Existing-project adoption has one path: `adopt STAGE`.
- The legacy command tree and the distribution commands leave the product binary.

## The verb table

`D` stands for the digest a write is bound to; see `--expect` in [Arguments](#arguments). Effects are defined in [Effects](#effects-read-write-execute). Every verb is also an MCP tool of the same name, except those marked "CLI only".

### Set up

| Verb | Synopsis | Effect | Replaces |
|---|---|---|---|
| `init` | `init --name NAME [--expect D --write]` | write | `project init`; gains `--expect` |
| `config` | `config (--input FILE \| --provider P --model M [profile flags]) [--expect D --write]` | write | `project setup`, with the flags from RUN-01 (pull request #104) |
| `onboard` | `onboard --provider codex\|claude\|both [--document-path PATH] [--expect D --write]` | write | `project onboard` |
| `doctor` | `doctor --provider P [--provider-executable PATH]` | read | `project doctor` |
| `mcp` | `mcp [--read-only]` | server; CLI only | `project mcp` |

### Model

| Verb | Synopsis | Effect | Replaces |
|---|---|---|---|
| `schema` | `schema` | read | `project schema`; new as an MCP tool |
| `check` | `check [--revision R] [--coverage]` | read | `project check` and `project coverage` |
| `model` | `model [--revision R]` | read | `project index` |
| `context` | `context MANAGER [--revision R]` | read | `project context` |
| `impact` | `impact --since R1 --revision R2` | read | `project impact` |
| `docs` | `docs [--revision R] \| docs --expect D --write` | write | `project document`; gains `--expect`; a write uses the working tree |
| `edit` | `edit --input FILE [--revision R] [--expect D --write]` | write | `project edit` |

### Work items

| Verb | Synopsis | Effect | Replaces |
|---|---|---|---|
| `explore` | `explore [--exploration ID \| --input FILE] [--revision R] [--expect D --write]` | write | `project explore` |
| `ready` | `ready --exploration ID --scope ID [--acknowledge --actor A --authority TEXT --decision-ref REF --acknowledged-at TIME] [--revision R] [--expect D --write]` | write | `project readiness` |
| `brief` | `brief --since R1 --revision R2 --provenance TEXT [--expect D --write]` | write | `project brief` |
| | `brief list [--manager ID]` | read | `project briefings` |
| | `brief dismiss --event ID --manager ID --expect D --write` | write | `project dismiss` |

`brief` is new as an MCP tool.

### Delivery

| Verb | Synopsis | Effect | Replaces |
|---|---|---|---|
| `plan` | `plan --goal TEXT [--operation apply\|cleanup\|reconcile] [--manager ID ...] [--revision R] [--since R0] [--exploration ID --scope ID] [--input FILE] [--expect D --write]` | write | `project plan`, `cleanup` and `reconcile`; gains `--expect` |
| `run` | `run PLAN --execute` | execute | `project run --plan ID --write` |
| `resume` | `resume RUN --execute` | execute | `project resume` |
| `repair` | `repair RUN --execute` | execute | `project repair`. Without `--execute` it is a usage error; inspect with `status RUN` |
| `status` | `status [RUN]` | read | `project status`, plus the new [overview](#status) |
| `verify` | `verify RUN --execute` | execute | `project verify --run`; MCP `project_verify` |
| | `verify --revision R --execute [--write]` | execute, optional write | `project verify --revision`; MCP `project_full_verify` |
| `apply` | `apply --plan ID --run ID --candidate ID [--branch B --head C --worktree DIGEST --expect D --write]` | write | `project apply`; MCP `project_preflight` and `project_apply` |
| `deliver` | `deliver --exploration ID --scope ID [--run ID] --execute --write` | execute and write | `project deliver` |

Without `--write`, `apply` returns the preflight: the verification digest, branch, head and worktree digest that the write must repeat.

`plan --input FILE` takes a model mutation, the same record `edit` takes; it replaces MCP `modelEdit`. The plan carries the mutation as a draft in its initial candidate, so the run delivers the model change and its implementation together. `edit` does not cover this, because `edit --write` changes the working tree and a plan needs a clean basis. `--input` is not allowed with `--exploration` or with `--operation cleanup|reconcile`, as today.

### Adoption

| Verb | Synopsis | Effect | Replaces |
|---|---|---|---|
| `adopt` | `adopt STAGE [--session ID] [--source-repo PATH] [--revision R] [--input FILE] [--expect D] [--write] [--execute]` | per stage, see [Adoption stages](#adoption-stages) | `project brownfield`, `discover`, `distill`, `resolve`, `adopt`; MCP `project_brownfield` and `project_brownfield_run` |

### Information

| Verb | Synopsis | Effect |
|---|---|---|
| `version` | `version` | CLI only |
| `licenses` | `licenses` | CLI only |
| `help` | `help [VERB]` | CLI only |

## Arguments

Each argument has one name. The CLI flag is its kebab-case form, and the MCP field is its camelCase form. A repeatable flag is an array field of the same name.

| Flag | MCP field | Meaning | Today |
|---|---|---|---|
| `--repo PATH` | none | Project root. Defaults to the current directory. MCP fixes it with `mcp --repo` and never takes it from tool arguments. | required in `project` |
| `--revision R` | `revision` | The fixed commit a verb reads, verifies or plans against. Reads without it use the working tree and report `provisional`. | MCP `baseRevision` on plan |
| `--since R` | `since` | The older revision for change impact (`impact`, `plan`, `brief`). | `--base`, `baseRevision` on impact; `sinceRevision` |
| `--manager ID` | `manager` | Manager ID. Repeatable on `plan`; `context` takes it as its operand. | `managerId`, `managers` |
| `--goal TEXT` | `goal` | Bounded plan goal. | same |
| `--operation KIND` | `operation` | `apply` (default), `cleanup` or `reconcile`. | separate `cleanup` and `reconcile` verbs |
| `--exploration ID` | `exploration` | Work-item exploration. | `explorationId` |
| `--scope ID` | `scope` | Scope within an exploration. | `scopeId` |
| `PLAN`, `--plan ID` | `plan` | Plan ID. | `planId` |
| `RUN`, `--run ID` | `run` | Run ID. | `runId` |
| `--candidate ID` | `candidate` | Integrated candidate. | `candidateId` |
| `--branch B`, `--head C`, `--worktree DIGEST` | `branch`, `head`, `worktree` | Apply freshness guards from the preflight. | `targetBranch`, `expectedHead`, `expectedWorktree` |
| `--expect D` | `expect` | The digest a write is bound to; the write fails as stale when the state behind it changed. Usually the digest of the verb's own preview. For `plan` it is the preview's `previewDigest`, which leaves out the run identity that persisting assigns. For `brief dismiss` it is the `stateDigest` from `brief list`. For an `adopt` stage write it is the session digest (compare and swap), and `adopt run --write --execute` requires the run preview's `previewDigest`. | `expectedDigest`, `expectedVerificationDigest` |
| `--write` | `write` | Persist. | `write`, and `executeAuthorized` on plan and deliver |
| `--execute` | `execute` | Start agents or configured checks. | new; today part of `--write` |
| `--input FILE` | `input` | One typed JSON record. The CLI reads it from a `.json` file under `.markitect/drafts/` or `.markitect/runs/`, as every record input today; MCP takes the same object inline. For `adopt`, the record holds the stage's payload under the stage's key, such as `{"start": {...}}`. | MCP `record`, `mutation`, `modelEdit`, `options`, `request`, `input` |
| `--session ID` | `session` | Adoption session. | `sessionId` |
| `--source-repo PATH` | none | Adoption source repository. Defaults to `--repo`. CLI only: MCP `adopt` uses the server root, because tool arguments cannot redirect the root. | `sourceRoot` |
| `--acknowledge` | `acknowledge` | Acknowledge the exact proposed structure, with `--actor`, `--authority`, `--decision-ref` and `--acknowledged-at` (fields `actor`, `authority`, `decisionRef`, `acknowledgedAt`). | `--acknowledge-structure`; MCP `acknowledgement` object |
| `--coverage` | `coverage` | `check` evaluates whole-repository coverage regardless of the project's coverage mode. | `project coverage` |
| `--read-only` | none | `mcp` only. | new |

The remaining verb-specific flags keep their names and gain matching fields: `--name`, `--provider`, `--model`, `--effort`, `--codex-profile`, `--windows-sandbox-backend`, `--provider-executable`, `--provider-arg` (repeatable), `--provider-version`, `--cost-mode`, `--input-micros-per-million`, `--output-micros-per-million`, `--max-cost-micros`, `--document-path`, `--provenance` and `--event`. Today's nested MCP `options` objects for setup, doctor and onboard become these flat fields.

**Operands.** A verb takes a positional operand only when it acts on one record named by one ID: `context MANAGER`, `run PLAN`, `resume RUN`, `repair RUN`, `status [RUN]` and `verify RUN`. Verbs that need several IDs use flags only. `adopt STAGE` and `brief [list|dismiss]` take a sub-verb. In MCP, an operand is an ordinary field: `manager`, `plan`, `run`, and `action` for every sub-verb (the `adopt` stage; for `brief`, `create`, `list` or `dismiss`, default `create`). Each operand has one CLI form; there is no flag alias for it.

## Effects: read, write, execute

Every verb, and every `adopt` stage, has one effect class in the table.

- **read** never changes project state. It starts nothing except Git and, for `doctor` and the `config` preview, a bounded `--version` probe of the provider executable.
- **write** previews by default. The preview is read-only and returns a digest. `--write --expect D` persists exactly what `D` was taken from; if that changed, the write fails as stale. `--expect` is valid only together with `--write` or `--execute`.
- **execute** starts agents or the project's configured checks, and only with `--execute`. Without it, the verb fails with exit code 2 and names `--execute` and the read-only verb to inspect first. What an execution records about itself (its run journal, attempts and receipts, and the evidence of `verify RUN`) is part of the execution and needs no `--write`.
- **execute and write** verbs need both flags. `deliver` runs agents and applies the result to the target branch. `verify --revision` persists its standalone verification record only with `--write`; without it, it runs the checks and reports.

This separates the two authorities that `--write` carried until now. The CLI can now run a full verification without persisting it, as MCP already could.

Each execution is bound to a persisted, digest-bound record instead of a preview digest: `run` to the plan from `plan --write`, `resume`, `repair` and `verify` to the run, and `deliver` to the scope acknowledged with `ready --write`. That is the preview step for those verbs.

The plan record keeps its `executeAuthorized` field for CLI-02. It then means "persisted by an authorized caller". The authority to start agents is `--execute` on `run`, `resume`, `repair` and `deliver`. Renaming the field is runtime work and not needed for CLI-02.

## Output, exit codes and errors

- Every verb writes one JSON object to stdout. Exceptions: `version`, `licenses` and `help` write text, and `mcp` writes protocol messages only.
- `docs` returns JSON with the document path, its digest and its content, so that it can follow preview, digest, then write like every other write.
- Exit code 0: success. Exit code 1: the verb completed with a non-conforming or blocked outcome, such as check findings, coverage gaps, missing doctor prerequisites, failed verification, an apply that did not apply, or a blocked delivery. Exit code 2: a usage error or a failed operation. These are today's `project` codes.
- Errors go to stderr as `markitect VERB: message`.

## Help

- `markitect --help` and `markitect help` list every verb from the table, grouped as above, with a one-line summary.
- `markitect help VERB` and `markitect VERB --help` print the synopsis, each argument with its help text, the effect class and whether the verb is an MCP tool.
- All help text comes from the table. That fixes today's drift, such as `plan` not listing `--exploration` and `--scope`, and `mcp` missing from `project --help`.
- `markitect` without arguments prints the verb list to stderr and exits with 2.

## MCP

- `tools/list` is generated from the verb table. Every verb not marked "CLI only" is one tool with the verb's name and no prefix.
- A tool's input fields are the verb's arguments, without `repo`. Today's tools `project_full_verify`, `project_preflight` and `project_brownfield_run` are folded into `verify`, `apply` and `adopt`.
- Outcomes and diagnostics name the verb in their `operation` field. Recovery hints name verbs, for example `status`, instead of `project_status`.
- Mutating calls stay serialized, as today: a second mutating call while one is active returns `busy`.
- **Annotations.** `readOnlyHint` is true for read verbs, and for every tool in read-only mode. It is false for write and execute verbs in normal mode. MCP-01 settles the remaining hints.
- **Read-only mode.** `markitect mcp --read-only` registers:
  - every read verb; `doctor` without its `providerExecutable` field, so a client cannot choose the program it probes;
  - every write verb as a preview-only tool, without the `write` and `expect` fields; `config` also without `providerExecutable`;
  - `brief` with `action` limited to `create` (preview only) and `list`; `dismiss` has no preview;
  - `adopt` with its read stages and the preview of every write stage, including `adopt run`, without the `write`, `expect` and `execute` fields; `adopt apply` has no preview of its own (`adopt plan` is its preview), so it is left out.

  It does not register `run`, `resume`, `repair`, `verify` or `deliver`. The server's `initialize` instructions state that it is read-only. A read-only server cannot write or start anything, whatever the client sends.
- Smaller schemas are MCP-01's work. The table's argument names must not change to achieve them.

## Status

`status RUN` returns today's run status report.

`status` without an operand returns a project overview. It is read-only and quick, and it uses only already-recorded state and the compiled model:
- project name, selected revision and whether it is provisional;
- model check status and finding counts, plus coverage conformity when the project's coverage mode is `full`;
- whether runtime configuration exists, and its digest;
- each run with its plan ID, status, last update and the next verb that applies, such as `resume`, `repair`, `verify` or `apply`;
- explorations with their scopes and readiness state;
- pending briefings per Manager;
- open adoption sessions with their current stage.

## Adoption stages

One path remains: the **session ledger** behind today's `project brownfield`. It is the newer path, has the MCP tools, and is the path the generated onboarding skills already name. It reuses the pipeline's record types and its adoption plan and apply code.

Two capabilities exist today only in the file pipeline (`discover`, `distill`, `resolve`, `adopt`). They move into ledger stages:
- `adopt start` takes a discovery request and runs discovery itself. Today `start` needs a sealed Discovery that no ledger stage or MCP tool can create.
- `adopt resolve` takes the human choices and builds and seals the Resolution in the host, as the pipeline's `resolve` does. Today the ledger makes the caller assemble the sealed record.

| Stage | Does | Effect | Today |
|---|---|---|---|
| `start` | Discovers the source and creates a session; needs `--revision` and `--input` (discovery request and scope statuses). | write | `discover` plus `brownfield start` |
| `begin` | Adds a reverse iteration for a Manager. | write | `brownfield begin` |
| `context` | Shows a Manager's proposal or integration context. | read | `brownfield context` |
| `propose` | Records a Manager proposal. | write | `brownfield propose` |
| `integrate` | Records the integration of child proposals. | write | `brownfield integrate` |
| `iterate` | Begin, propose and optionally integrate in one step. | write | `brownfield iterate` |
| `run` | Runs a Manager agent for the propose or integrate phase and records the result in the session. | execute and write | `brownfield run`; MCP `project_brownfield_run` |
| `resolve` | Builds and seals the Resolution from human choices. | write | `resolve` (pipeline) and `brownfield resolve` |
| `plan` | Shows the adoption plan. | read | `brownfield plan`; pipeline `adopt` without `--write` |
| `apply` | Applies the reviewed plan to the target model and records the receipt; needs `--expect` and the plan digest. | write | `brownfield apply-adoption`; pipeline `adopt --write` |
| `status` | Session overview with readiness, scopes, conflicts and coverage. | read | `brownfield resume` |

All stages except `start` need `--session`. Writes compare and swap against the session digest, as today.

**Removed:**
- the pipeline's one-shot `distill` and `distill --generate`; agent-assisted proposals come from `adopt run`, per Manager, journaled and retryable;
- the pipeline's own output records (discovery, distillation, resolution and adoption plan files) as a user-facing interface. `.markitect/drafts/` itself stays: `--input` files and adoption sessions live there;
- `record-adoption`, which was already rejected.

The pipeline-only code in `projectadoption` that becomes unused, such as `GenerateDistillation`, is removed with it.

## Convenience verbs

A convenience verb saves real steps; a verb that only renames a flag is not one.
- `status` without an operand composes the overview.
- `deliver` composes plan, run, verify, preflight and apply for an acknowledged scope.

`cleanup` and `reconcile` do not remain as verbs. They are `plan --operation cleanup|reconcile`.

## What leaves the product binary

- **The legacy tree.** Today the product binary still dispatches the Project/Domain commands: `check`, `verify`, `model`, `context`, `impact`, `schema`, `format`, `canonical`, `reconcile`, `projection` and the rest. Six of these names collide with the new verbs, and Markitect's own CI, hooks and scripts still call them.

  CLI-02 therefore moves the legacy dispatcher unchanged into a temporary binary, `markitect-legacy`. The CI, hook and script calls, and the legacy command path in `AGENTS.md`, switch to it in the same merge. ARCH-07 then moves Markitect onto its own model and the current verbs, and ARCH-09 deletes `markitect-legacy`.

  Without this step there is a cycle: the verbs cannot take their names while the legacy tree holds them, and the legacy tree goes only after ARCH-07, which needs the verbs.

  A pinned v0.14.1 release cannot take this role. It rejects the current `markitect.yaml` (`unknown field "timeoutSeconds"`), and `bundle` must package the current source.
- **Distribution commands.** `pack`, `package`, `bundle` and `install` belong to the legacy dispatcher, so CLI-02 moves them into `markitect-legacy` too. Between CLI-02 and ARCH-08, `ci.yaml` and `release.yaml` call them as `go run ./src/cmd/markitect-legacy`. The installed-bundle smoke in `release.yaml` runs legacy commands through `.markitect/bootstrap/run.go`; it switches with them. ARCH-08 then moves them to `markitect-release` or removes them. `version` and `licenses` stay as information verbs.
- **Other binaries.** `markitect-check-artifacts`, `markitect-check-modules` and the three `markitect-adapter-*` binaries belong to the legacy line. They stay separate and unchanged until ARCH-07 and ARCH-09 retire them. `markitect-check-architecture` and `markitect-release` are not affected.
- **The `project` noun** and every `project_` tool name, with no aliases (DEC-014).

## Settled details

DEC-017 delegated these details to this specification:
1. **Brownfield path.** The session ledger stays, as `adopt STAGE`. Discovery and host-built resolution move into its `start` and `resolve` stages. The file pipeline goes.
2. **Verb names.** As in [the verb table](#the-verb-table): `config` for setup, `model` for index, `docs` for document, `ready` for readiness, `brief [list|dismiss]` for the three briefing actions, and `adopt` stages as listed.
3. **Convenience verbs.** `status` (overview) and `deliver`. `cleanup` and `reconcile` become `plan --operation`.

It also settles:

4. One argument name per meaning across CLI and MCP, including `--since` for every older revision and `expect` for every digest guard.
5. `--repo` defaults to the current directory. MCP takes the root only from `mcp --repo`.
6. Operands are used only for single-record verbs; every operand is also a named MCP field.
7. `verify` is an execute verb under D5, because it starts the configured verifier agent; its configured checks run under the same `--execute`. It is absent from read-only MCP.
8. An execution's own journal needs no `--write`; results persisted into the project do.
9. `init`, `docs` and `plan` gain the preview digest and `--expect`. `docs` output becomes JSON.
10. `repair` without `--execute` no longer doubles as `status`.
11. Read-only MCP keeps previews of write verbs and omits execute verbs.
12. The legacy tree, including `pack`, `package`, `bundle` and `install`, moves to a temporary `markitect-legacy` binary in CLI-02 and is deleted by ARCH-09.
13. `version` and `licenses` stay in the product binary; ARCH-08 takes the other distribution commands out of `markitect-legacy`.
14. `plan --input` replaces MCP `modelEdit`, so the CLI gains a capability only MCP had.

None of these changes DEC-017's direction.

## Notes for the implementing packages

**Table shape.** One Go table, with one entry per verb:
- name, help group and one-line summary;
- effect class;
- optional operand;
- arguments, each with name, kind (string, bool, repeated, record), whether required, and help text;
- whether the verb is an MCP tool;
- the handler that calls the application facade.

MCP registration iterates the table. The `mcp` package then registers no tools of its own and stays a pure adapter. For CLI-02 the table can stay in `projectcli`; package names are CLI-03's concern.

**Input types.** Several MCP tools decode runtime structs directly today: `projectrun.PlanRequest`, `ApplyRequest` and `DeliverRequest`. The new field names live in input types in the interface zone, which the handlers map onto the runtime requests. CLI-02 does not change runtime JSON tags or persisted records.

**Parity test (CLI-02).** It walks the table and fails when:
- a verb has no handler or no help entry;
- an MCP tool exists that is not in the table;
- a tool's input fields differ from the verb's arguments, apart from `repo` and the read-only reductions.

TEST-02 adds per-verb help, bad-flag, happy-path and no-write-without-flag cases on top of this table.

**Touches outside the interface zone.** The integrator should schedule these:
- **ci and docs-entry:** the switch of CI, release, hook and script calls and `AGENTS.md` to `markitect-legacy`, in the same merge as CLI-02. This includes `package` in `ci.yaml`, and `bundle`, `install` and the bootstrap smoke in `release.yaml`.
- **runtime:** a read-only run listing in `projectrun` for the `status` overview.
- **runtime, only if the facade cannot do it:** comparing the plan preview digest before persisting.
- **free (`projectadoption`):** the discovery and resolution composition for `adopt start` and `adopt resolve`, and removal of the pipeline-only code.
- **docs:** `README.md`, `CONTRIBUTING.md`, Usage, Project workflow, Project operations, Provider adapters, Architecture and `docs/design/project-world/*`. Generated agent guidance belongs to AGENT-02.

**Order.** CLI-02 starts from main after RUN-01 (pull request #104), which brought the setup flags this table takes over as `config`.
