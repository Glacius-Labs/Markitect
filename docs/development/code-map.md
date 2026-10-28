# Code map

This page lists every Go package in the repository with its layer, the first sentence of its doc comment and its owning document. Read the owning document before you change a package, and update it when the package's behavior changes.

The page is generated from the rule table in [rules.go](../../internal/tooling/architecture/rules.go) and the package doc comments. Do not edit it by hand. Change the table or the doc comment, then regenerate the page from the repository root:

```powershell
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

`TestRepositoryArchitecture` checks every production, test and platform file against the table: a package imports only the layers its own layer lists and the named exceptions, only test files import a test-only layer, and a standalone layer imports no Markitect package. Every Go package needs exactly one row, an existing owning document and one package doc comment. The source packager accepts exactly the table's packages below `cmd/` and `internal/`. [Architecture](../architecture.md#layers) describes the layers.

## Layers

| Layer | Meaning | May import | Packages |
|---|---|---|---|
| `model` | The model language over supplied bytes: Definitions, Kinds, links, visibility, cycles, manifest and lock. | nothing | 1 |
| `assets` | Files built into the binary: the standard Libraries and the skills. | nothing | 2 |
| `domain` | Registry pages, ownership and file accounting over a compiled model. | `model` | 1 |
| `project` | Loads and checks a project's Markitect folder from a repository state. | `model`, `assets`, `domain` | 1 |
| `service` | Questions about a loaded project: references, graph, impact with checklists, and lint. | `model`, `assets`, `domain`, `project` | 1 |
| `remote` | A Run's plan, its verdict trailers and the decision of the remote's hooks. | `model`, `assets`, `domain`, `project`, `service` | 1 |
| `protocol` | Frame protocol between the Orchestrator and the session helper. | nothing | 1 |
| `container` | The container engine client that the adapters running containers share. | `protocol` | 1 |
| `guardedwrite` | Guarded working-tree writes for bundle installation, with the Git and path helpers they share with source snapshots. | nothing | 1 |
| `build` | Version identity and embedded license notices. | nothing | 1 |
| `adapter` | Concrete Git, command-line and Run remote adapters; no adapter imports another. | `model`, `assets`, `domain`, `project`, `service`, `remote`, `build` | 4 |
| `composition` | Constructs the command line with its adapters. | `model`, `project`, `service`, `adapter` | 1 |
| `command` | Thin entrypoints to product composition. | `composition` | 1 |
| `remote-command` | Thin entrypoint of a Run's remote container. | `adapter` | 1 |
| `agent` | Runs inside each Agent container: provider process, MCP bridge and stdio framing. | `protocol` | 1 |
| `agent-command` | Thin entrypoint of the session helper. | `agent` | 1 |
| `maintenance` | The import gate and release packaging, separate from product execution. | `guardedwrite` | 2 |
| `maintenance-command` | Release maintenance entrypoint. | `maintenance`, `build` | 1 |
| `test-support` | Hermetic fixtures. Only test files may import it. It imports no Markitect package, not even from tests. | nothing | 1 |
| `standalone` | Consumer bootstrap and source smoke. It imports no Markitect package, not even from tests. | nothing | 2 |

## Exceptions

| From | To | Files | Reason |
|---|---|---|---|
| `internal/tooling/release` | `internal/tooling/architecture` | all | The packager takes its source roots from this table. |
| `internal/tooling/release` | `internal/buildinfo` | tests only | Tests compare the packaged notices with the embedded ones. |

## Canonical model

| Package | Summary | Owning document |
|---|---|---|
| `internal/model` | Package model compiles the canonical model of concept 2.0 from explicitly supplied Markdown files: Definitions with front matter, Packages as folders, Libraries of Kinds, links by path, visibility and the Package dependency graph. | [docs/guides/model-language.md#definitions](../guides/model-language.md#definitions) |

## Built-in files

| Package | Summary | Owning document |
|---|---|---|
| `internal/libraries` | Package libraries holds the standard Libraries that ship inside the markitect binary: std, rest and cli, each a folder of Kind Definitions. | [docs/guides/model-language.md#kinds-and-libraries](../guides/model-language.md#kinds-and-libraries) |
| `internal/skills` | Package skills holds the agent skills that ship inside the markitect binary: the modeling skill and the Extract skill built on it. | [docs/guides/model-language.md#commands](../guides/model-language.md#commands) |

## Realization record

| Package | Summary | Owning document |
|---|---|---|
| `internal/realization` | Package realization checks the realization record beside the model: one registry page per Package with the path patterns it owns and the files it realizes, and the ignore page. | [docs/guides/model-language.md#the-realization-record](../guides/model-language.md#the-realization-record) |

## Project

| Package | Summary | Owning document |
|---|---|---|
| `internal/project` | Package project loads a project's Markitect folder from a repository state and checks it as a whole: manifest, lock, Libraries, model, registry and required checks. | [docs/guides/model-language.md#the-projects-markitect-folder](../guides/model-language.md#the-projects-markitect-folder) |

## Analysis

| Package | Summary | Owning document |
|---|---|---|
| `internal/analysis` | Package analysis answers questions about a loaded project without starting agents: who links to a Definition, the Package dependency graph, what a change reaches and each Manager's checklist, and the lint warnings that measure the model like code. | [docs/guides/model-language.md#impact-and-checklists](../guides/model-language.md#impact-and-checklists) |

## Run remote rules

| Package | Summary | Owning document |
|---|---|---|
| `internal/runremote` | Package runremote holds the rules of a Run's Git remote (concept 2.0 §6.3): the Run's plan, the commit trailers that carry verdicts and reviews, and the decision the server-side hook makes on every push. | [internal/runremote/README.md](../../internal/runremote/README.md) |

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
| `internal/guardedwrite` | Package guardedwrite applies selected working-tree changes only while the repository, branch, HEAD and captured file bytes are unchanged. | [docs/architecture.md#distribution](../architecture.md#distribution) |

## Build data

| Package | Summary | Owning document |
|---|---|---|
| `internal/buildinfo` | Package buildinfo owns the version and the upstream license notices shared by product and release tools. | [docs/architecture.md#distribution](../architecture.md#distribution) |

## Adapters

| Package | Summary | Owning document |
|---|---|---|
| `internal/adapters/cli` | Package cli is the markitect command line: init, check, lock, refs, graph, impact, lint, version and licenses. | [docs/guides/model-language.md#commands](../guides/model-language.md#commands) |
| `internal/adapters/remotecontainer` | Package remotecontainer starts and removes a Run's remote container from the host, natively on Windows, Linux and macOS with only the Docker CLI. | [internal/runremote/README.md#the-container](../../internal/runremote/README.md#the-container) |
| `internal/adapters/remoteserver` | Package remoteserver is the Run remote inside its container: the Git smart HTTP server with one credential per agent, the pre-receive hook that applies the rules of internal/runremote, the required checks under a throwaway user, and seeding and exporting the bare repository. | [internal/runremote/README.md](../../internal/runremote/README.md) |
| `internal/adapters/repo` | Package repo reads a project's repository through Git: the working tree as Git sees it, a base commit, and the files changed between them. | [docs/guides/model-language.md#the-projects-markitect-folder](../guides/model-language.md#the-projects-markitect-folder) |

## Composition

| Package | Summary | Owning document |
|---|---|---|
| `internal/bootstrap` | Package bootstrap constructs the command line with its Git adapter. | [docs/architecture.md#product-flow](../architecture.md#product-flow) |

## Product commands

| Package | Summary | Owning document |
|---|---|---|
| `cmd/markitect` | Command markitect is the Markitect command line: init, check, lock, refs, graph, impact, lint, version and licenses. | [docs/guides/model-language.md#commands](../guides/model-language.md#commands) |

## Run remote command

| Package | Summary | Owning document |
|---|---|---|
| `cmd/markitect-remote` | Command markitect-remote is the program of a Run's remote container (internal/runremote/README.md): it seeds the bare repository, serves it to the Managers, decides every push as its pre-receive hook and exports the result. | [internal/runremote/README.md](../../internal/runremote/README.md) |

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
