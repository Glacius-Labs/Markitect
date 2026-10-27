# Code map

This page lists every Go package in the repository with its layer, the first sentence of its doc comment and its owning document. Read the owning document before you change a package, and update it when the package's behavior changes.

The page is generated from the rule table in [rules.go](../../internal/tooling/architecture/rules.go) and the package doc comments. Do not edit it by hand. Change the table or the doc comment, then regenerate the page from the repository root:

```powershell
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

`TestRepositoryArchitecture` checks every production, test and platform file against the table: a package imports only the layers its own layer lists and the named exceptions, only test files import a test-only layer, a standalone layer imports no Markitect package, and the document writer `internal/adapters/documents` uses only the reviewed guarded API of `internal/guardedwrite`. Every Go package needs exactly one row, an existing owning document and one package doc comment. The source packager accepts exactly the table's packages below `cmd/` and `internal/`. [Architecture](../architecture.md#layers) describes the layers.

## Layers

| Layer | Meaning | May import | Packages |
|---|---|---|---|
| `model` | The canonical language check over supplied bytes. | nothing | 1 |
| `snapshot` | Immutable repository tree values. | nothing | 1 |
| `domain` | Manager topology, ownership and accounting over model and snapshot values. | `model`, `snapshot` | 2 |
| `project` | Checked structural preparation that every command starts from. | `model`, `snapshot`, `domain` | 1 |
| `service` | Run execution, static runtime metadata, harness instructions and the model view over a checked Project. | `model`, `snapshot`, `domain`, `project` | 4 |
| `usecase` | Command preflight, the sequence of closed Runs and stored-candidate publication. | `model`, `project`, `service` | 2 |
| `protocol` | Frame protocol between the Orchestrator and the session helper. | nothing | 1 |
| `container` | The container engine client that the adapters running containers share. | `protocol` | 1 |
| `guardedwrite` | Guarded working-tree writes for publication, docs --write and bundle installation, with the Git and path helpers they share with source snapshots. | nothing | 1 |
| `build` | Version identity and embedded license notices. | nothing | 1 |
| `adapter` | Concrete repository, Session, check, record, runner, command-line and document adapters; no adapter imports another. | `model`, `snapshot`, `domain`, `project`, `service`, `usecase`, `container`, `guardedwrite`, `build` | 7 |
| `composition` | Constructs the actual services and their adapters. | `model`, `project`, `service`, `usecase`, `container`, `guardedwrite`, `adapter` | 1 |
| `command` | Thin entrypoints to product composition. | `composition` | 2 |
| `agent` | Runs inside each Agent container: provider process, MCP bridge and stdio framing. | `protocol` | 1 |
| `agent-command` | Thin entrypoint of the session helper. | `agent` | 1 |
| `maintenance` | The import gate and release packaging, separate from product execution. | `guardedwrite` | 2 |
| `maintenance-command` | Release maintenance entrypoint. | `maintenance`, `build` | 1 |
| `test-support` | Hermetic fixtures. Only test files may import it. It imports no Markitect package, not even from tests. | nothing | 1 |
| `standalone` | Consumer bootstrap and source smoke. It imports no Markitect package, not even from tests. | nothing | 2 |

## Exceptions

| From | To | Files | Reason |
|---|---|---|---|
| `cmd/markitect-runner` | `internal/adapters/runner` | all | The runner image entrypoint starts the runner adapter directly. |
| `internal/adapters/codexappserver` | `internal/agentd/frame` | all | Codex Sessions use the frame values that the container client exposes. |
| `internal/apply` | `internal/workflow` | all | Apply runs the sequence of closed Runs. |
| `internal/tooling/release` | `internal/tooling/architecture` | all | The packager takes its source roots from this table. |
| `internal/adapters/codexappserver` | `internal/adapters/gitrepo` | tests only | Hierarchy tests run Codex Sessions over real Run Git copies. |
| `internal/adapters/codexappserver` | `internal/adapters/runstore` | tests only | Hierarchy tests read the stored Run records. |
| `internal/adapters/gitrepo` | `internal/adapters/checks` | tests only | The real-Git vertical tests compose workspaces with declared checks. |
| `internal/adapters/gitrepo` | `internal/adapters/runstore` | tests only | The real-Git vertical tests store and read back Run records. |
| `internal/bootstrap` | `internal/snapshot` | tests only | Composition tests build snapshot check values. |
| `internal/tooling/release` | `internal/buildinfo` | tests only | Tests compare the packaged notices with the embedded ones. |

## Canonical model

| Package | Summary | Owning document |
|---|---|---|
| `internal/model` | Package model checks explicitly supplied canonical model sources. | [docs/architecture.md#foundations-and-project](../architecture.md#foundations-and-project) |

## Snapshots

| Package | Summary | Owning document |
|---|---|---|
| `internal/snapshot` | Package snapshot defines fixed generic file trees from explicitly supplied values. | [docs/architecture.md#foundations-and-project](../architecture.md#foundations-and-project) |

## Organization and realization

| Package | Summary | Owning document |
|---|---|---|
| `internal/organization` | Package organization derives local Manager organization from checked model values and resolves explicitly supplied execution settings. | [docs/architecture.md#foundations-and-project](../architecture.md#foundations-and-project) |
| `internal/realization` | Package realization describes artifact ownership and file accounting over explicitly supplied model and snapshot values. | [docs/architecture.md#foundations-and-project](../architecture.md#foundations-and-project) |

## Project

| Package | Summary | Owning document |
|---|---|---|
| `internal/project` | Package project prepares the checked structural input that every command starts from: model, registry, accounting, declared instructions and checks, and the static Manager contexts. | [internal/project/README.md](../../internal/project/README.md) |

## Services

| Package | Summary | Owning document |
|---|---|---|
| `internal/harness` | Package harness renders fixed checked project inputs without acquisition or execution. | [internal/harness/README.md](../../internal/harness/README.md) |
| `internal/modeldocs` | Package modeldocs prepares a generated canonical v1 view and an explicit source/target-bound write plan. | [docs/architecture.md#product-flow](../architecture.md#product-flow) |
| `internal/run` | Package run executes one complete modeled hierarchy and returns only after candidate retention, resource closure and terminal record readback. | [internal/run/README.md](../../internal/run/README.md) |
| `internal/runtimeconfig` | Package runtimeconfig prepares closed static native/check profile metadata. | [docs/architecture.md#foundations-and-project](../architecture.md#foundations-and-project) |

## Apply

| Package | Summary | Owning document |
|---|---|---|
| `internal/apply` | Package apply composes checked capture, closed Runs and durable publication. | [internal/apply/README.md](../../internal/apply/README.md) |
| `internal/workflow` | Package workflow selects fully closed Runs. | [internal/apply/README.md](../../internal/apply/README.md) |

## Agent protocol

| Package | Summary | Owning document |
|---|---|---|
| `internal/agentd/frame` | Package frame is the wire format between the Markitect Orchestrator and the session helper markitect-agentd. | [internal/adapters/container/README.md](../../internal/adapters/container/README.md) |

## Container client

| Package | Summary | Owning document |
|---|---|---|
| `internal/adapters/container` | Package container is the Orchestrator's client of the container engine. | [internal/adapters/container/README.md](../../internal/adapters/container/README.md) |

## Guarded write

| Package | Summary | Owning document |
|---|---|---|
| `internal/guardedwrite` | Package guardedwrite applies selected working-tree changes only while the repository, branch, HEAD and captured file bytes are unchanged. | [docs/architecture.md#state-and-evidence-storage](../architecture.md#state-and-evidence-storage) |

## Build data

| Package | Summary | Owning document |
|---|---|---|
| `internal/buildinfo` | Package buildinfo owns the version and the upstream license notices shared by product and release tools. | [docs/architecture.md#distribution](../architecture.md#distribution) |

## Adapters

| Package | Summary | Owning document |
|---|---|---|
| `internal/adapters/checks` | Package checks prepares pinned container-check metadata and observes bounded exact-candidate checks. | [internal/adapters/checks/README.md](../../internal/adapters/checks/README.md) |
| `internal/adapters/cli` | Package cli presents preparation and Apply. | [docs/architecture.md#adapters](../architecture.md#adapters) |
| `internal/adapters/codexappserver` | Package codexappserver runs Codex native Manager Sessions in Agent containers, loads the model-derived harness into them, maps the Run tools and prepares static profile metadata from an explicitly selected runtime image. | [internal/adapters/codexappserver/README.md](../../internal/adapters/codexappserver/README.md) |
| `internal/adapters/documents` | Package documents implements the narrow generated-view target observation and guarded write port. | [docs/architecture.md#adapters](../architecture.md#adapters) |
| `internal/adapters/gitrepo` | Package gitrepo acquires explicit repository sources and publishes guarded working-tree changes. | [docs/architecture.md#adapters](../architecture.md#adapters) |
| `internal/adapters/runner` | Package runner starts Apply in the Linux runner image and returns its verified result as a branch (DEC-077). | [docs/design/linux-runner.md](../design/linux-runner.md) |
| `internal/adapters/runstore` | Package runstore stores closed Run records, an index of completed Verify approvals and independently appended Notes. | [internal/run/README.md](../../internal/run/README.md) |

## Composition

| Package | Summary | Owning document |
|---|---|---|
| `internal/bootstrap` | Package bootstrap constructs concrete adapters for the implemented product. | [docs/architecture.md#product-flow](../architecture.md#product-flow) |

## Product commands

| Package | Summary | Owning document |
|---|---|---|
| `cmd/markitect` | Command markitect is the Markitect command line: check, context, docs, apply, version and licenses. | [docs/architecture.md#product-flow](../architecture.md#product-flow) |
| `cmd/markitect-runner` | Command markitect-runner is the entrypoint of the Linux runner image (docs/design/linux-runner.md). | [docs/design/linux-runner.md](../design/linux-runner.md) |

## Session helper

| Package | Summary | Owning document |
|---|---|---|
| `internal/agentd` | Package agentd is the session helper that runs inside every Agent container. | [internal/adapters/container/README.md](../../internal/adapters/container/README.md) |

## Session helper command

| Package | Summary | Owning document |
|---|---|---|
| `cmd/markitect-agentd` | Command markitect-agentd is the session helper inside every Markitect Agent container. | [internal/adapters/container/README.md](../../internal/adapters/container/README.md) |

## Maintenance

| Package | Summary | Owning document |
|---|---|---|
| `internal/tooling/architecture` | Package architecture holds the repository's one architecture rule table and enforces it without compiling the inspected code: every production, test and platform-specific file imports only what its package's layer allows. | [docs/architecture.md#layers](../architecture.md#layers) |
| `internal/tooling/release` | Package release builds a portable, reproducible Markitect source archive and a source-bound bundle, and installs a verified bundle into a consumer. | [docs/architecture.md#distribution](../architecture.md#distribution) |

## Maintenance command

| Package | Summary | Owning document |
|---|---|---|
| `cmd/markitect-release` | Command markitect-release prepares local Markitect distributions: source packaging, bundling and bundle installation, plus the version and license notices. | [docs/architecture.md#distribution](../architecture.md#distribution) |

## Test support

| Package | Summary | Owning document |
|---|---|---|
| `internal/testkit` | Package testkit provides hermetic Git and file-system fixtures for tests. | [docs/architecture.md#layers](../architecture.md#layers) |

## Standalone tools

| Package | Summary | Owning document |
|---|---|---|
| `packaging/bootstrap` | run-markitect.go is copied into a consumer repository and invoked with `go run .markitect/bootstrap/run.go ...`. | [packaging/README.md](../../packaging/README.md) |
| `tools/smoke` | Command smoke checks Markitect's packaged distribution end to end. | [docs/architecture.md#distribution](../architecture.md#distribution) |
