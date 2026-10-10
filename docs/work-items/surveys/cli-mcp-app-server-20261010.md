# Survey: CLI, MCP and App Server

AI-generated, read-only, 10 October 2026, against main `02c7e529`. Input for CLI-01 to CLI-03, MCP-01, MCP-02, TEST-02, AGENT-02 and ARCH-08. Decisions are in register DEC-014 and DEC-017.

## Two command trees

**Current product:** `markitect project <action>`. Actions are defined in `src/internal/host/projectcli/options.go` (`actionSpecs`); output is JSON.

| Group | Actions |
|---|---|
| Setup | `schema`, `init`, `onboard`, `setup`, `doctor`, `mcp` (missing from `--help`) |
| Inspect | `check`, `index`, `coverage`, `context`, `impact`, `document` |
| Work item and model | `explore`, `readiness`, `edit` |
| Execute | `plan`; `cleanup` and `reconcile` are `plan` with an operation; `run`, `resume`, `repair`, `status`; `verify` (`--run` or `--revision`); `apply` (preview without `--write`); `deliver` |
| Briefings | `brief`, `briefings`, `dismiss` |

Brownfield has two parallel paths:
- `brownfield --brownfield-action start|begin|context|propose|integrate|iterate|run|resolve|plan|apply-adoption|resume`, plus an unlisted `record-adoption`;
- the pipeline `discover` → `distill` → `resolve` → `adopt`.

**Legacy tree:** `markitect.yaml`, YAML output, defined in `src/internal/host/cli/distribution.go` (`commandFlags`).
- Top-level commands: `check`, `verify`, `inventory`, `model`, `context`, `impact`, `find`, `explain`, `review`, `render`, `format`, `schema`, `authoring`.
- `--action` style: `reconcile` (observe, plan, apply, verify) and `projection` (observe, verify, apply).
- `canonical`: about 22 actions.
- `prepare` and `copy-me`.

**Tooling:** `version`, `licenses`, `pack`, `package`, `bundle`, `install`. Separate binaries: `markitect-release`, `markitect-check-{architecture,artifacts,modules}` and `markitect-adapter-{github,azure-devops,dotnet}`.

Eleven verbs exist in both trees with different meanings: check, verify, context, impact, schema, reconcile, plan, apply, adopt, resolve, resume. Markitect's own CI makes 43 legacy `markitect` calls.

**Inconsistencies:**
- `project` takes a positional action; legacy commands take an `--action` flag.
- `--repo` is required in `project` but defaults to the current directory in legacy.
- Output formats differ: JSON in `project`, YAML in legacy.
- `init` and `document` write without `--expect`.
- `--write` both persists and authorizes execution, so the CLI cannot run a non-persisting full verify; MCP can.
- The usage text of `plan` omits `--exploration` and `--scope`.

## MCP

Tools are registered in `src/internal/host/mcp/tools.go` (10) and `src/internal/host/projectcli/mcp.go` (15).

**Mapping to the CLI:**
- Most tools are `project_X` for `project X`.
- `project_plan` covers plan, cleanup and reconcile.
- `project_verify` is `verify --run`; `project_full_verify` is `verify --revision`.
- `project_preflight` is `apply` without `--write`; `project_apply` is `apply --write`.
- `project_brownfield_run` is `brownfield --brownfield-action run`.
- No MCP twin exists for `schema`, `brief`, `briefings`, `dismiss` or the discover/distill/resolve/adopt pipeline.

**Schema size:** `tools/list` is about 307 KB, roughly 75 to 80 thousand tokens. About 280 KB of that is output schemas. The largest are `project_brownfield` at 63 KB and `project_deliver` at 47 KB. The run report schema repeats in run, resume and repair.

**Read-only:** Preview and write share one tool, so every such tool reports `readOnlyHint:false`. There is no read-only server mode.

**Argument-name drift between CLI flags and MCP fields:**
- `--expect` ↔ `expectedDigest`; on apply, `expectedVerificationDigest`
- `--exploration` ↔ `explorationId`; `--scope` ↔ `scopeId`
- `--manager` ↔ `managerId` or `managers`
- `--base` ↔ `baseRevision`; `--since` ↔ `sinceRevision`
- `--source-repo` ↔ `sourceRoot`; `--session` ↔ `sessionId`
- `--branch`, `--head`, `--worktree` ↔ `targetBranch`, `expectedHead`, `expectedWorktree`
- `--input FILE` ↔ inline objects
- `--write` ↔ `write` or `executeAuthorized`

`docs/mcp-evaluation.md` still describes the old read-only pilot.

## App Server

"App Server" here means only the transport for inner roles: `src/internal/host/codexappserver`, which runs `codex app-server --listen stdio://`.
- **Wiring:** through `projectrun` (config, transport invoker, native journal, helper, workspace recovery) and through the doctor check in `projectsetup`.
- **No product app server:** `projectapp` is only the facade that CLI and MCP share.
- **Product branches:** `codex/product-app-server-20261009`, `codex/product-mcp-20261009` and `codex/product-workspaces-20261009` add nothing that is not on main.

## Draft verb model

The verb names below are the survey's draft, recorded for the CLI-01 specification. DEC-017 fixes the direction: top-level verbs, one verb table, a separate `--execute`, a read-only MCP mode, and no `project_` prefix.

| Verb | Today |
|---|---|
| `init`, `onboard`, `doctor`, `schema`, `edit`, `explore` | same `project` actions |
| `config` | `setup` |
| `status` | new overview; `status RUN` for one run |
| `check [--coverage]` | `check` and `coverage` |
| `model` | `index` |
| `context MANAGER`, `impact BASE [REV]` | same |
| `docs` | `document` |
| `ready` | `readiness` |
| `plan [--kind cleanup\|reconcile]` | `plan`, `cleanup`, `reconcile` |
| `run`, `resume`, `repair`, `verify`, `apply`, `deliver` | same |
| `brief [list\|dismiss]` | `brief`, `briefings`, `dismiss` |
| `adopt <stage>` | brownfield |
| `mcp [--read-only]` | `project mcp` |
| `version`, `licenses` | same |

**Shared rules for CLI and MCP:**
- MCP tool name equals the verb.
- Each flag is the kebab-case form of its MCP field.
- `--write` only persists; `--execute` starts agents.
- Every write goes through preview, then digest, then write.
- JSON output by default.
- `--repo` defaults to the current directory.
- Read-only status is visible to clients.
- Help lists every verb.

## Collision zones

- **Top-level CLI:** `src/internal/host/cli/{main.go,distribution.go,cli_dispatch.go,cli_options.go}`.
- **projectcli and mcp:** `src/internal/host/projectcli/*` (help text in `run.go`; `runAction` is 484 lines) and `src/internal/host/mcp/{tools.go,schema.go}`. Tool-name-keyed error mapping is in `tools.go`.
- **Onboarding:** `src/internal/host/projectonboarding/{capabilities.go,render.go}`.
- **Harness and examples:** `src/harness/examples/project_world_workflow_test.go` and `examples/project-world`.
- **CI:** `.github/workflows/ci.yaml`.
- **Docs:** `README.md`, `CONTRIBUTING.md`, `docs/{usage,project-workflow,project-operations,provider-adapters,architecture,repository-layout,mcp-evaluation}.md` and `docs/design/project-world/*`.
