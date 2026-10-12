# Code map

This page lists every Go package in the repository with its layer, purpose and owning document. Read the owning document before you change a package, and update it when the package's behavior changes.

The page is generated from [codemap.yaml](../../src/internal/tooling/architecture/codemap.yaml). Do not edit it by hand. Change the YAML, then regenerate the page from the repository root:

```powershell
go test ./src/internal/tooling/architecture -run TestCodeMapPage -update
```

Tests in `src/internal/tooling/architecture` fail when a Go package is missing from the map, when an entry names a directory that is no longer a package, when an owning document is missing, or when this page is stale. Each layer must agree with the coarse layer of the [import gate](modules.md#mechanical-dependency-gate). The gate also enforces two rules of the finer layers: `core`, `infrastructure`, `application` and `runtime` packages may not import `legacy` packages, and they may use only the guarded API of `src/internal/host/guardedwrite`.

## Layers

| Layer | Meaning | Import gate layers | Packages |
|---|---|---|---|
| `core` | Structural compiler, snapshots and project-model views. No providers, Git processes or writes. | `core`, `host`, `module` | 5 |
| `infrastructure` | Git and working-tree access: fixed snapshots and guarded writes. | `infrastructure`, `host` | 2 |
| `application` | Current product use cases and their command and MCP surfaces. | `host` | 13 |
| `runtime` | Runs Manager, review and verify roles in owned workspaces. | `host` | 8 |
| `legacy` | The published v0.13 Project/Domain surfaces. DEC-014 allows removing them. | `host`, `module` | 0 |
| `module` | Independent capability packages that import only Core. | `module` | 2 |
| `cli` | Thin entrypoints that delegate to Host. | `cli` | 5 |
| `tooling` | Maintainer tooling: import gate, release, publication and notices. | `tooling` | 5 |
| `bootstrap` | Standalone public bootstrap without Markitect imports. | `bootstrap` | 1 |
| `harness` | Executable tests, test support and example programs; not product code. | `harness-tests`, `harness-runtime`, `testkit` | 2 |
| `experiment` | Bounded pilots kept with their evaluation. | `harness-runtime` | 1 |
| `fixture` | Adopting-project code used as test input; it may not import Markitect. | `fixture` | 0 |

## Deterministic core

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/core` | Compiles explicitly supplied Schemas and Definitions into a pure structural model. | [src/internal/core/README.md](../../src/internal/core/README.md) |
| `src/internal/core/snapshot` | Fixed sets of source files and deterministic comparisons over them. | [docs/source-snapshots.md](../source-snapshots.md#current-value) |
| `src/internal/host/projectbriefing` | Records deterministic briefings for accepted model changes from committed history. | [docs/architecture.md](../architecture.md#core-layer) |
| `src/internal/host/projectcoverage` | Inventories repository paths and classifies how the project model covers them. | [docs/architecture.md](../architecture.md#core-layer) |
| `src/internal/modules/projectmodel` | Derives management, artifact and impact views from the compiled project model. | [docs/architecture.md](../architecture.md#core-layer) |

## Infrastructure

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/host/guardedwrite` | Applies selected working-tree changes only while repository, branch, HEAD and captured bytes are unchanged. | [docs/architecture.md](../architecture.md#infrastructure-layer) |
| `src/internal/infrastructure/source` | Loads fixed snapshots from the working tree or a Git commit with hardened Git processes. | [docs/source-snapshots.md](../source-snapshots.md#adapter-and-use-case-ownership) |

## Product application

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/host/architecturecli` | Host command adapter for the repository architecture gate. | [docs/development/modules.md](modules.md#mechanical-dependency-gate) |
| `src/internal/host/buildinfo` | Shared immutable source and build version for product and release tooling. | [integration/README.md](../../integration/README.md#current-source-release-tooling) |
| `src/internal/host/mcp` | Local stdio MCP server over the project operations. | [docs/project-operations.md](../project-operations.md#mcp-server-and-tool-groups) |
| `src/internal/host/modulechecks` | Host composition for the independent Git hooks and pipeline checks. | [CONTRIBUTING.md](../../CONTRIBUTING.md#verify-a-change) |
| `src/internal/host/modulecli` | Command adapter for the independent Git hooks and pipeline checks. | [CONTRIBUTING.md](../../CONTRIBUTING.md#verify-a-change) |
| `src/internal/host/projectadoption` | Brownfield adoption that checks fixed-snapshot evidence and prepares model-only changes. | [docs/project-workflow.md](../project-workflow.md#existing-repositories) |
| `src/internal/host/projectapp` | Shared application facade that the project CLI and MCP server call. | [docs/architecture.md](../architecture.md#application-layer) |
| `src/internal/host/projectcli` | The `markitect` verb table, its command line and its MCP tools. | [docs/project-operations.md](../project-operations.md#cli-setup-and-current-roots) |
| `src/internal/host/projectexplore` | Stores explorations, open decisions, scope readiness and Apply receipts. | [docs/project-workflow.md](../project-workflow.md#model-and-readiness) |
| `src/internal/host/projectonboarding` | Writes project-local guidance and skills for Codex and Claude contributors. | [docs/project-workflow.md](../project-workflow.md#connect-and-inspect) |
| `src/internal/host/projectwork` | Loads the selected project model and writes model changes under guard. | [docs/project-workflow.md](../project-workflow.md#model-and-readiness) |
| `src/internal/host/releasecli` | Command interface of `markitect-release` for preparing or publishing a GitHub release. | [docs/operations.md](../operations.md#release-operations) |
| `src/internal/host/releaseinstall` | Plans and applies complete verified bundle pins through guarded writes. | [integration/README.md](../../integration/README.md#current-source-release-tooling) |

## Execution runtime

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/host/agentexec` | Provider-neutral subprocess boundary for one role invocation and its receipt. | [src/internal/host/agentexec/README.md](../../src/internal/host/agentexec/README.md) |
| `src/internal/host/codexappserver` | Transport to the native Codex App Server for Manager, review and verify roles. | [docs/provider-adapters.md](../provider-adapters.md#inner-role-execution-and-workspace-boundary) |
| `src/internal/host/exchangecli` | Reference bring-your-own executor that hands each role invocation to an external party through request and response files. | [docs/provider-adapters.md](../provider-adapters.md#bring-your-own-executor) |
| `src/internal/host/projectconfig` | Loads and validates the explicit runtime configuration and finite role policies. | [docs/project-operations.md](../project-operations.md#cli-setup-and-current-roots) |
| `src/internal/host/projectexecutor` | Executes process and native role transports with private protocol evidence and injected helper policy. | [docs/provider-adapters.md](../provider-adapters.md#inner-role-execution-and-workspace-boundary) |
| `src/internal/host/projectrun` | Runs, resumes, verifies and applies bounded Manager work over fixed snapshots. | [docs/project-workflow.md](../project-workflow.md#execute-verify-and-apply) |
| `src/internal/host/projectsetup` | Builds the project-local agent runtime proposal for `project setup`. | [docs/project-operations.md](../project-operations.md#cli-setup-and-current-roots) |
| `src/internal/host/projectworkspace` | Owned candidate workspaces for bounded runs. | [docs/project-operations.md](../project-operations.md#candidate-workspaces-and-recovery) |

## Modules

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/modules/githooks` | Pure checks for explicitly configured Git hook entrypoints. | [docs/development/modules.md](modules.md#go-capability-packages) |
| `src/internal/modules/pipelines` | Pure checks for configured pipeline references to project checks. | [docs/development/modules.md](modules.md#go-capability-packages) |

## Executables

| Package | Purpose | Owning document |
|---|---|---|
| `src/cmd/markitect` | The `markitect` executable with the top-level verbs. | [docs/project-operations.md](../project-operations.md#cli-setup-and-current-roots) |
| `src/cmd/markitect-check-architecture` | Runs the import gate. | [docs/development/modules.md](modules.md#mechanical-dependency-gate) |
| `src/cmd/markitect-check-modules` | Runs the independent Module checks used by the pre-commit hook. | [CONTRIBUTING.md](../../CONTRIBUTING.md#verify-a-change) |
| `src/cmd/markitect-exchange-executor` | The reference bring-your-own executor executable. | [docs/provider-adapters.md](../provider-adapters.md#bring-your-own-executor) |
| `src/cmd/markitect-release` | Prepares or publishes a GitHub release. | [docs/operations.md](../operations.md#release-operations) |

## Tooling

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/tooling/architecture` | Import gate and code map checks. | [docs/development/modules.md](modules.md#mechanical-dependency-gate) |
| `src/internal/tooling/licenses` | Third-party notices embedded in the CLI. | [CONTRIBUTING.md](../../CONTRIBUTING.md#release-work) |
| `src/internal/tooling/publish` | Prepares and, on request, publishes an immutable GitHub release. | [integration/README.md](../../integration/README.md#owner-publication) |
| `src/internal/tooling/release` | Builds the reproducible source archive and release bundle. | [integration/README.md](../../integration/README.md#release-contents-and-verification) |
| `tools/smoke` | Smoke driver that packages the source and checks the built tools end to end on every OS. | [CONTRIBUTING.md](../../CONTRIBUTING.md#verify-a-change) |

## Bootstrap

| Package | Purpose | Owning document |
|---|---|---|
| `integration` | Single-file bootstrap copied into consumer repositories to build and run a pinned release. | [integration/README.md](../../integration/README.md) |

## Harnesses

| Package | Purpose | Owning document |
|---|---|---|
| `src/harness/examples` | Executable tests over the example fixtures. | [examples/README.md](../../examples/README.md) |
| `src/internal/testkit` | Hermetic Git and temporary-directory fixtures that only tests import. | [CONTRIBUTING.md](../../CONTRIBUTING.md#verify-a-change) |

## Experiments

| Package | Purpose | Owning document |
|---|---|---|
| `experiments/mcp-pilot` | Small local MCP stdio pilot kept with its evaluation. | [experiments/mcp-pilot/README.md](../../experiments/mcp-pilot/README.md) |
