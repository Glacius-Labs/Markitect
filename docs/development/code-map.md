# Code map

This page lists every Go package in the repository with its layer, purpose and owning document. Read the owning document before you change a package, and update it when the package's behavior changes.

The page is generated from [codemap.yaml](../../src/internal/tooling/architecture/codemap.yaml). Do not edit it by hand. Change the YAML, then regenerate the page from the repository root:

```powershell
go test ./src/internal/tooling/architecture -run TestCodeMapPage -update
```

Tests in `src/internal/tooling/architecture` fail when a Go package is missing from the map, when an entry names a directory that is no longer a package, when an owning document is missing, or when this page is stale. Each layer must agree with the coarse layer of the [import gate](modules.md#mechanical-dependency-gate); the gate does not yet enforce the finer layers.

## Layers

| Layer | Meaning | Import gate layers | Packages |
|---|---|---|---|
| `core` | Structural compiler, snapshots and project-model views. No providers, Git processes or writes. | `core`, `host`, `module` | 5 |
| `infrastructure` | Git and working-tree access: fixed snapshots and guarded writes. | `infrastructure`, `host` | 2 |
| `application` | Current product use cases and their command and MCP surfaces. | `host` | 8 |
| `runtime` | Runs Manager, review and verify roles in owned workspaces. | `host` | 6 |
| `legacy` | The published v0.13 Project/Domain surfaces. DEC-014 allows removing them. | `host`, `module` | 26 |
| `module` | Independent capability packages that import only Core. | `module` | 0 |
| `cli` | Thin entrypoints that delegate to Host. | `cli` | 9 |
| `tooling` | Maintainer tooling: import gate, release, publication and notices. | `tooling` | 4 |
| `bootstrap` | Standalone public bootstrap without Markitect imports. | `bootstrap` | 1 |
| `harness` | Executable tests, test support and example programs; not product code. | `harness-tests`, `harness-runtime`, `testkit` | 5 |
| `experiment` | Bounded pilots kept with their evaluation. | `harness-runtime` | 1 |
| `fixture` | Adopting-project code used as test input; it may not import Markitect. | `fixture` | 1 |

## Deterministic core

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/core` | Compiles explicitly supplied Schemas and Definitions into a pure structural model. | [src/internal/core/README.md](../../src/internal/core/README.md) |
| `src/internal/core/snapshot` | Fixed sets of source files and deterministic comparisons over them. | [docs/source-snapshots.md](../source-snapshots.md#current-value) |
| `src/internal/host/projectbriefing` | Records deterministic briefings for accepted model changes from committed history. | [docs/design/project-world/operation-scopes-and-model-briefings.md](../design/project-world/operation-scopes-and-model-briefings.md) |
| `src/internal/host/projectcoverage` | Inventories repository paths and classifies how the project model covers them. | [docs/project-operations.md](../project-operations.md#ordinary-work-item) |
| `src/internal/modules/projectmodel` | Derives management, artifact and impact views from the compiled project model. | [docs/project-workflow.md](../project-workflow.md#model-and-readiness) |

## Infrastructure

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/host/guardedwrite` | Applies selected working-tree changes only while repository, branch, HEAD and captured bytes are unchanged. | [docs/design/host-write-identity.md](../design/host-write-identity.md#authority-and-object-identity) |
| `src/internal/infrastructure/source` | Loads fixed snapshots from the working tree or a Git commit with hardened Git processes. | [docs/source-snapshots.md](../source-snapshots.md#adapter-and-use-case-ownership) |

## Product application

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/host/mcp` | Local stdio MCP server over the project operations. | [docs/project-operations.md](../project-operations.md#mcp-server-and-tool-groups) |
| `src/internal/host/projectadoption` | Brownfield adoption that checks fixed-snapshot evidence and prepares model-only changes. | [docs/project-workflow.md](../project-workflow.md#existing-repositories) |
| `src/internal/host/projectapp` | Shared application facade that the project CLI and MCP server call. | [docs/architecture.md](../architecture.md) |
| `src/internal/host/projectcli` | The `markitect project` command line. | [docs/project-operations.md](../project-operations.md#cli-setup-and-current-roots) |
| `src/internal/host/projectexplore` | Stores explorations, open decisions, scope readiness and Apply receipts. | [docs/project-workflow.md](../project-workflow.md#model-and-readiness) |
| `src/internal/host/projectonboarding` | Writes project-local guidance and skills for Codex and Claude contributors. | [docs/project-workflow.md](../project-workflow.md#connect-and-inspect) |
| `src/internal/host/projectwork` | Loads the selected project model and writes model changes under guard. | [docs/project-workflow.md](../project-workflow.md#model-and-readiness) |
| `src/internal/host/releasecli` | Command interface of `markitect-release` for preparing or publishing a GitHub release. | [docs/operations.md](../operations.md#release-operations) |

## Execution runtime

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/host/agentexec` | Provider-neutral subprocess boundary for one role invocation and its receipt. | [src/internal/host/agentexec/README.md](../../src/internal/host/agentexec/README.md) |
| `src/internal/host/codexappserver` | Transport to the native Codex App Server for Manager, review and verify roles. | [docs/provider-adapters.md](../provider-adapters.md#inner-role-execution-and-workspace-boundary) |
| `src/internal/host/exchangecli` | Reference bring-your-own executor that hands each role invocation to an external party through request and response files. | [docs/provider-adapters.md](../provider-adapters.md#bring-your-own-executor) |
| `src/internal/host/projectrun` | Runs, resumes, verifies and applies bounded Manager work over fixed snapshots. | [docs/project-workflow.md](../project-workflow.md#execute-verify-and-apply) |
| `src/internal/host/projectsetup` | Builds the project-local agent runtime proposal for `project setup`. | [docs/project-operations.md](../project-operations.md#cli-setup-and-current-roots) |
| `src/internal/host/projectworkspace` | Owned candidate workspaces for bounded runs. | [docs/project-operations.md](../project-operations.md#candidate-workspaces-and-recovery) |

## Legacy and compatibility

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/host` | Host root with the v0.13 Project/Domain commands and their composition. | [docs/development/modules.md](modules.md#responsibility-map) |
| `src/internal/host/artifactcli` | Command interface of `markitect-check-artifacts` for managed-artifact accounting. | [docs/usage.md](../usage.md#managed-artifact-accounting) |
| `src/internal/host/authoring` | Source YAML model for v0.13 Projects, Domains and resources. | [docs/usage.md](../usage.md#project-and-resource-model) |
| `src/internal/host/authoring/contentpackage` | Reads and builds deterministic offline content-package archives. | [docs/content-packages.md](../content-packages.md) |
| `src/internal/host/azuredevopscli` | Command protocol of the Azure DevOps metadata adapter. | [src/cmd/markitect-adapter-azure-devops/README.md](../../src/cmd/markitect-adapter-azure-devops/README.md) |
| `src/internal/host/cli` | Top-level `markitect` command dispatcher with the legacy verbs. | [docs/usage.md](../usage.md#commands) |
| `src/internal/host/compat/v0_13/consumers/agentrules` | Historical v0.13 projection of Codex and Claude entrypoints. | [src/internal/host/compat/v0_13/consumers/agentrules/README.md](../../src/internal/host/compat/v0_13/consumers/agentrules/README.md) |
| `src/internal/host/compat/v0_13/consumers/agentrules/links` | Rewrites Markdown link targets for the v0.13 agent-rules consumer. | [src/internal/host/compat/v0_13/consumers/agentrules/README.md](../../src/internal/host/compat/v0_13/consumers/agentrules/README.md) |
| `src/internal/host/compat/v0_13/consumers/artifactcoverage` | Historical v0.13 evaluation of managed-artifact ownership. | [src/internal/host/compat/v0_13/consumers/artifactcoverage/README.md](../../src/internal/host/compat/v0_13/consumers/artifactcoverage/README.md) |
| `src/internal/host/compat/v0_13/consumers/azuredevops` | Historical v0.13 offline comparison of Azure DevOps Git metadata. | [src/internal/host/compat/v0_13/consumers/azuredevops/README.md](../../src/internal/host/compat/v0_13/consumers/azuredevops/README.md) |
| `src/internal/host/compat/v0_13/consumers/dotnet` | Historical v0.13 comparison of MSBuild project files. | [src/internal/host/compat/v0_13/consumers/dotnet/README.md](../../src/internal/host/compat/v0_13/consumers/dotnet/README.md) |
| `src/internal/host/compat/v0_13/consumers/githooks` | Historical v0.13 bounded Git hook checks. | [src/internal/host/compat/v0_13/consumers/githooks/README.md](../../src/internal/host/compat/v0_13/consumers/githooks/README.md) |
| `src/internal/host/compat/v0_13/consumers/github` | Historical v0.13 offline comparison of GitHub repository metadata. | [src/internal/host/compat/v0_13/consumers/github/README.md](../../src/internal/host/compat/v0_13/consumers/github/README.md) |
| `src/internal/host/compat/v0_13/consumers/markdown` | Historical v0.13 Markdown views of the semantic model. | [src/internal/host/compat/v0_13/consumers/markdown/README.md](../../src/internal/host/compat/v0_13/consumers/markdown/README.md) |
| `src/internal/host/compat/v0_13/consumers/markdown/links` | Rewrites Markdown link targets for the v0.13 Markdown consumer. | [src/internal/host/compat/v0_13/consumers/markdown/README.md](../../src/internal/host/compat/v0_13/consumers/markdown/README.md) |
| `src/internal/host/compat/v0_13/consumers/pipelines` | Historical v0.13 bounded pipeline checks. | [src/internal/host/compat/v0_13/consumers/pipelines/README.md](../../src/internal/host/compat/v0_13/consumers/pipelines/README.md) |
| `src/internal/host/compat/v0_13/consumers/projections` | Historical v0.13 projection plan, apply and verify API. | [src/internal/host/compat/v0_13/consumers/projections/README.md](../../src/internal/host/compat/v0_13/consumers/projections/README.md) |
| `src/internal/host/compat/v0_13/kernel` | Historical v0.13 Domain, resource and policy kernel (Go package name `core`). | [src/internal/host/compat/v0_13/README.md](../../src/internal/host/compat/v0_13/README.md) |
| `src/internal/host/dotnetcli` | Command protocol of the .NET reference adapter. | [src/cmd/markitect-adapter-dotnet/README.md](../../src/cmd/markitect-adapter-dotnet/README.md) |
| `src/internal/host/embedded` | Tool-shipped authoring guidance compiled as a Markitect context. | [docs/architecture.md](../architecture.md#authoring-and-queries) |
| `src/internal/host/githubcli` | Command protocol of the GitHub repository-metadata adapter. | [src/cmd/markitect-adapter-github/README.md](../../src/cmd/markitect-adapter-github/README.md) |
| `src/internal/host/inputs` | Resolves explicitly declared non-Markitect input files for a v0.13 graph. | [docs/architecture.md](../architecture.md#fixed-inputs-and-evidence) |
| `src/internal/host/modulecli` | Command interface of `markitect-check-modules`. | [docs/usage.md](../usage.md#independent-module-checks-current-source) |
| `src/internal/host/projectionengine` | Pure evaluation of explicit projection contracts. | [src/internal/host/projectionengine/README.md](../../src/internal/host/projectionengine/README.md) |
| `src/internal/modules/adoption/capture` | Selective-capture handoff for v0.13 Copy Me adoption. | [src/internal/modules/adoption/README.md](../../src/internal/modules/adoption/README.md) |
| `src/internal/modules/adoption/review` | Copy Me review over a selective-adoption handoff. | [src/internal/modules/adoption/README.md](../../src/internal/modules/adoption/README.md) |

## Executables

| Package | Purpose | Owning document |
|---|---|---|
| `src/cmd/markitect` | The `markitect` executable. | [docs/usage.md](../usage.md#commands) |
| `src/cmd/markitect-adapter-azure-devops` | Legacy Azure DevOps metadata adapter executable. | [src/cmd/markitect-adapter-azure-devops/README.md](../../src/cmd/markitect-adapter-azure-devops/README.md) |
| `src/cmd/markitect-adapter-dotnet` | Legacy .NET reference adapter executable. | [src/cmd/markitect-adapter-dotnet/README.md](../../src/cmd/markitect-adapter-dotnet/README.md) |
| `src/cmd/markitect-adapter-github` | Legacy GitHub repository-metadata adapter executable. | [src/cmd/markitect-adapter-github/README.md](../../src/cmd/markitect-adapter-github/README.md) |
| `src/cmd/markitect-check-architecture` | Runs the import gate. | [docs/development/modules.md](modules.md#mechanical-dependency-gate) |
| `src/cmd/markitect-check-artifacts` | Runs managed-artifact accounting. | [docs/usage.md](../usage.md#managed-artifact-accounting) |
| `src/cmd/markitect-check-modules` | Runs the independent Module checks used by the pre-commit hook. | [docs/usage.md](../usage.md#independent-module-checks-current-source) |
| `src/cmd/markitect-exchange-executor` | The reference bring-your-own executor executable. | [docs/provider-adapters.md](../provider-adapters.md#bring-your-own-executor) |
| `src/cmd/markitect-release` | Prepares or publishes a GitHub release. | [docs/operations.md](../operations.md#release-operations) |

## Tooling

| Package | Purpose | Owning document |
|---|---|---|
| `src/internal/tooling/architecture` | Import gate and code map checks. | [docs/development/modules.md](modules.md#mechanical-dependency-gate) |
| `src/internal/tooling/licenses` | Third-party notices embedded in the CLI. | [CONTRIBUTING.md](../../CONTRIBUTING.md#release-work) |
| `src/internal/tooling/publish` | Prepares and, on request, publishes an immutable GitHub release. | [integration/README.md](../../integration/README.md#owner-publication) |
| `src/internal/tooling/release` | Builds the reproducible source archive and release bundle. | [integration/README.md](../../integration/README.md#release-contents-and-verification) |

## Bootstrap

| Package | Purpose | Owning document |
|---|---|---|
| `integration` | Single-file bootstrap copied into consumer repositories to build and run a pinned release. | [integration/README.md](../../integration/README.md) |

## Harnesses

| Package | Purpose | Owning document |
|---|---|---|
| `examples/selective-adoption` | Replays a bounded public Markitect report for the selective-adoption example. | [examples/selective-adoption/README.md](../../examples/selective-adoption/README.md#public-report-replay) |
| `examples/selective-adoption/pathspell` | Canonical path spelling for the replay, including Windows short names. | [examples/selective-adoption/README.md](../../examples/selective-adoption/README.md#public-report-replay) |
| `src/harness/engineering-discovery` | Evidence helper for the engineering-discovery example. | [examples/engineering-discovery/README.md](../../examples/engineering-discovery/README.md) |
| `src/harness/examples` | Executable tests over the example fixtures. | [examples/README.md](../../examples/README.md) |
| `src/internal/testkit` | Hermetic Git and temporary-directory fixtures that only tests import. | [CONTRIBUTING.md](../../CONTRIBUTING.md#verify-a-change) |

## Experiments

| Package | Purpose | Owning document |
|---|---|---|
| `experiments/mcp-pilot` | Small local MCP stdio pilot kept with its evaluation. | [experiments/mcp-pilot/README.md](../../experiments/mcp-pilot/README.md) |

## Fixtures

| Package | Purpose | Owning document |
|---|---|---|
| `examples/documentation/docs/implementation/src` | Ordinary Go source used as a declared artifact input. | [examples/documentation/README.md](../../examples/documentation/README.md) |
