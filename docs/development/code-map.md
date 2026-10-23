# Code map

This page lists every Go package in the repository with its layer, purpose and owning document. Read the owning document before you change a package, and update it when the package's behavior changes.

The page is generated from [codemap.yaml](../../internal/tooling/architecture/codemap.yaml). Do not edit it by hand. Change the YAML, then regenerate the page from the repository root:

```powershell
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

Tests in `internal/tooling/architecture` require every actual Go package to have one compatible map entry and an existing owning document. The [import gate](modules.md#mechanical-dependency-gate) checks all production, test and platform files against explicit responsibility edges; there is no broad maintenance or adapter exemption. The document writer additionally uses only the reviewed guarded API of `internal/tooling/guardedwrite`.

## Layers

| Layer | Meaning | Import gate layers | Packages |
|---|---|---|---|
| `apply-model` | Pure canonical v1 Model check over explicit bytes. | `apply-model` | 1 |
| `apply-snapshot` | Pure immutable repository tree values. | `apply-snapshot` | 1 |
| `apply-organization` | Pure Manager topology, context selection and settings. | `apply-organization` | 1 |
| `apply-realization` | Pure ownership, accounting and candidate rules. | `apply-realization` | 1 |
| `apply-run` | One closed complete Manager hierarchy over direct Sessions, Git, checks and records. | `apply-run` | 1 |
| `apply-project` | Checked structural input and immutable model-derived context. | `apply-project` | 1 |
| `runtime-config` | Closed static native and check profile metadata. | `runtime-config` | 1 |
| `native-harness` | Model-derived instructions and tools for concrete native Sessions. | `native-harness` | 1 |
| `apply-usecase` | Product workflow composition; unavailable at the S2 source boundary. | `apply-usecase` | 1 |
| `model-view` | Deterministic readable view and reviewed source-bound write plan. | `model-view` | 1 |
| `apply-adapter` | Concrete repository, container engine, Agent and check adapters, profile metadata and document I/O. | `apply-adapter` | 8 |
| `apply-bootstrap` | Constructs actual services and their adapters. | `apply-bootstrap` | 1 |
| `apply-command` | Thin entrypoint to product composition. | `apply-command` | 1 |
| `agent-protocol` | Frame protocol between the Orchestrator and the session helper. | `agent-protocol` | 1 |
| `agent-runtime` | Runs inside each Agent container: provider process, MCP bridge and stdio framing. | `agent-runtime` | 1 |
| `runtime-command` | Thin entrypoint of the session helper. | `runtime-command` | 1 |
| `build-data` | Version identity and embedded license notices. | `build-data` | 2 |
| `maintenance` | Explicitly separated release and repository tooling. | `maintenance` | 12 |
| `maintenance-command` | Thin entrypoints to maintenance services. | `maintenance-command` | 3 |
| `test-support` | Hermetic fixture support available only to tests. | `test-support` | 1 |
| `standalone` | Public bootstrap and source smoke with no product imports. | `standalone` | 2 |

## Canonical model

| Package | Purpose | Owning document |
|---|---|---|
| `internal/model` | Pure canonical v1 Model check over explicit bytes. | [docs/development/modules.md](modules.md) |

## Snapshots

| Package | Purpose | Owning document |
|---|---|---|
| `internal/snapshot` | Pure immutable repository tree values. | [docs/development/modules.md](modules.md) |

## Organization

| Package | Purpose | Owning document |
|---|---|---|
| `internal/organization` | Pure Manager topology, context selection and settings. | [docs/development/modules.md](modules.md) |

## Realization

| Package | Purpose | Owning document |
|---|---|---|
| `internal/realization` | Pure ownership, accounting and candidate rules. | [docs/development/modules.md](modules.md) |

## Run

| Package | Purpose | Owning document |
|---|---|---|
| `internal/run` | Complete closed Manager hierarchy with direct delegation, questions, fixed Git handoffs and durable results. | [docs/development/modules.md](modules.md) |

## Project

| Package | Purpose | Owning document |
|---|---|---|
| `internal/project` | Structural model, registry, ownership, accounting, declared instructions and checks, and checked static Manager contexts. | [internal/project/README.md](../../internal/project/README.md) |

## Runtime configuration

| Package | Purpose | Owning document |
|---|---|---|
| `internal/runtimeconfig` | Closed acquired Runtime envelopes and source-bound static native or check image observations. | [docs/development/modules.md](modules.md) |

## Native harness

| Package | Purpose | Owning document |
|---|---|---|
| `internal/harness` | Pure complete model-derived instructions for native Manager Sessions. | [internal/harness/README.md](../../internal/harness/README.md) |

## Apply

| Package | Purpose | Owning document |
|---|---|---|
| `internal/apply` | Explicit unavailable Apply boundary until S3 workflow composition. | [docs/development/modules.md](modules.md) |

## Model view

| Package | Purpose | Owning document |
|---|---|---|
| `internal/modeldocs` | Deterministic readable view and reviewed source-bound write plan. | [docs/development/modules.md](modules.md) |

## Adapters

| Package | Purpose | Owning document |
|---|---|---|
| `internal/adapters/checks` | Check profile metadata and bounded container execution. | [docs/development/modules.md](modules.md) |
| `internal/adapters/claudecode` | Claude static image metadata and retained protocol codecs; native Run Sessions are explicitly unavailable. | [internal/adapters/claudecode/README.md](../../internal/adapters/claudecode/README.md) |
| `internal/adapters/cli` | Command parsing and JSON reports for check, context, docs and explicit Apply. | [docs/development/modules.md](modules.md) |
| `internal/adapters/codexappserver` | Codex native Manager Sessions, model-derived harness loading and direct Run tools in private assignment volumes. | [internal/adapters/codexappserver/README.md](../../internal/adapters/codexappserver/README.md) |
| `internal/adapters/container` | Container engine client and the Orchestrator side of the agentd connection. | [docs/development/modules.md](modules.md) |
| `internal/adapters/documents` | Concrete repository, profile metadata and document I/O. | [docs/development/modules.md](modules.md) |
| `internal/adapters/gitrepo` | Complete repository capture, isolated Run Git copies and independently reopenable stored candidates. | [docs/development/modules.md](modules.md) |
| `internal/adapters/runstore` | Immutable terminal Run results and durable concurrent project Notes. | [internal/run/README.md](../../internal/run/README.md) |

## Composition

| Package | Purpose | Owning document |
|---|---|---|
| `internal/bootstrap` | Composes structural Project, generated views, optional static metadata and the explicit unavailable Apply boundary. | [docs/development/modules.md](modules.md) |

## Product command

| Package | Purpose | Owning document |
|---|---|---|
| `cmd/markitect` | Thin entrypoint to product composition. | [docs/development/modules.md](modules.md) |

## Agent protocol

| Package | Purpose | Owning document |
|---|---|---|
| `internal/agentd/frame` | Frame protocol between the Orchestrator and the session helper. | [docs/development/modules.md](modules.md) |

## Session helper

| Package | Purpose | Owning document |
|---|---|---|
| `internal/agentd` | Session helper inside each Agent container. | [docs/development/modules.md](modules.md) |

## Runtime command

| Package | Purpose | Owning document |
|---|---|---|
| `cmd/markitect-agentd` | Thin entrypoint of the session helper. | [docs/development/modules.md](modules.md) |

## Build data

| Package | Purpose | Owning document |
|---|---|---|
| `internal/buildinfo` | Version identity and embedded license notices. | [docs/development/modules.md](modules.md) |
| `internal/tooling/licenses` | Version identity and embedded license notices. | [docs/development/modules.md](modules.md) |

## Maintenance

| Package | Purpose | Owning document |
|---|---|---|
| `internal/tooling/architecture` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/architecturecli` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/guardedwrite` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/modulechecks` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/modulecli` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/modules/githooks` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/modules/pipelines` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/publish` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/release` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/release/install` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/releasecli` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |
| `internal/tooling/source` | Explicitly separated release and repository tooling. | [docs/development/modules.md](modules.md) |

## Maintenance commands

| Package | Purpose | Owning document |
|---|---|---|
| `cmd/markitect-check-architecture` | Thin entrypoints to maintenance services. | [docs/development/modules.md](modules.md) |
| `cmd/markitect-check-modules` | Thin entrypoints to maintenance services. | [docs/development/modules.md](modules.md) |
| `cmd/markitect-release` | Thin entrypoints to maintenance services. | [docs/development/modules.md](modules.md) |

## Test support

| Package | Purpose | Owning document |
|---|---|---|
| `internal/testkit` | Hermetic fixture support available only to tests. | [docs/development/modules.md](modules.md) |

## Standalone tools

| Package | Purpose | Owning document |
|---|---|---|
| `integration` | Public bootstrap and source smoke with no product imports. | [docs/development/modules.md](modules.md) |
| `tools/smoke` | Public bootstrap and source smoke with no product imports. | [docs/development/modules.md](modules.md) |
