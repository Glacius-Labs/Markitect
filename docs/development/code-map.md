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
| `agent-port` | How the Run engine starts a maker or checker for a Manager and observes how it ends. | nothing | 1 |
| `agent-fake` | Scripted makers and checkers that work on a Run's remote with real Git, for tests. Only test files may import it. | `agent-port` | 1 |
| `run` | One Run per change: who works, makers and checkers along the tree, the Result and landing. | `model`, `assets`, `domain`, `project`, `service`, `remote`, `agent-port` | 1 |
| `protocol` | Frame protocol between the Orchestrator and the session helper. | nothing | 1 |
| `container` | The container engine client that the adapters running containers share. | `protocol` | 1 |
| `harness` | What a maker or checker receives: its role, the model it knows, its task and the tool contract. | `model`, `assets`, `domain`, `project`, `service`, `remote`, `agent-port` | 1 |
| `agents` | One maker or checker in its container: login, clone, provider turns and the Manager tools, with the Claude Code and Codex adapters. | `model`, `assets`, `domain`, `project`, `remote`, `agent-port`, `protocol`, `container`, `harness`, `agents` | 3 |
| `guardedwrite` | Guarded working-tree writes for bundle installation, with the Git and path helpers they share with source snapshots. | nothing | 1 |
| `build` | Version identity and embedded license notices. | nothing | 1 |
| `adapter` | Concrete Git, command-line, desktop and Run remote adapters; no adapter imports another. | `model`, `assets`, `domain`, `project`, `service`, `remote`, `agent-port`, `run`, `build` | 6 |
| `composition` | Constructs the command line with its adapters. | `model`, `project`, `service`, `remote`, `agent-port`, `container`, `agents`, `run`, `adapter` | 1 |
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
| `internal/agents` | `internal/adapters/remotecontainer` | tests only | The real-engine test starts real Managers against a real Run remote. |
| `internal/run` | `internal/adapters/remoteserver` | tests only | The engine's scripted Runs push through the Run remote's real hooks. |
| `internal/run` | `internal/adapters/rungit` | tests only | The engine's scripted Runs use the host's real Git. |
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

## Agent port

| Package | Summary | Owning document |
|---|---|---|
| `internal/agentport` | Package agentport is the boundary between the Run engine and the agents it starts (concept 2.0 §6.2, §12): the engine starts a maker or a checker for one Manager with its role, task, checklist, tools and deadline, and observes how it ends. | [internal/agentport/README.md](../../internal/agentport/README.md) |

## Scripted agents

| Package | Summary | Owning document |
|---|---|---|
| `internal/agentport/agentfake` | Package agentfake is a scripted implementation of agentport.Agents for tests: every start of a Manager's maker or checker runs the next scripted Step, which works on the Run's remote with real Git and calls the host's tools as an agent would. | [internal/agentport/README.md](../../internal/agentport/README.md) |

## Run engine

| Package | Summary | Owning document |
|---|---|---|
| `internal/run` | Package run is the Run engine of concept 2.0 (§6, §7): one Run per change. | [internal/run/README.md](../../internal/run/README.md) |

## Agent protocol

| Package | Summary | Owning document |
|---|---|---|
| `internal/agentd/frame` | Package frame is the wire format between the Markitect Orchestrator and the session helper markitect-agentd. | [internal/adapters/container/README.md](../../internal/adapters/container/README.md) |

## Container client

| Package | Summary | Owning document |
|---|---|---|
| `internal/adapters/container` | Package container is the Orchestrator's client of the container engine. | [internal/adapters/container/README.md](../../internal/adapters/container/README.md) |

## Manager harness

| Package | Summary | Owning document |
|---|---|---|
| `internal/harness` | Package harness renders what one maker or checker of a Manager receives (concept 2.0 §5, §6.2): its role and repository, the model it knows, its task with the checklist, and the answer format, together with the tools it answers through. | [internal/agents/README.md#harness](../../internal/agents/README.md#harness) |

## Manager agents

| Package | Summary | Owning document |
|---|---|---|
| `internal/agents` | Package agents starts and drives one agent of a Manager (concept 2.0 §6.2, §9): a maker or a checker, each in its own hardened Linux container that clones the Run's remote, with a provider CLI logged in through an access-only copy of the account login. | [internal/agents/README.md](../../internal/agents/README.md) |
| `internal/agents/claude` | Package claude is the Claude Code adapter of the agent runtime: the launch contract of the pinned CLI in a Manager container, the access-only copy of the account login, and the reader of its stream-json output with the failure classes the evaluation voids. | [internal/agents/README.md#claude-code](../../internal/agents/README.md#claude-code) |
| `internal/agents/codex` | Package codex is the Codex adapter of the agent runtime: the launch contract of the pinned Codex CLI's exec mode in a Manager container, the access-only copy of the ChatGPT account login, and the reader of its JSON events with the failure classes the evaluation voids. | [internal/agents/README.md#codex](../../internal/agents/README.md#codex) |

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
| `internal/adapters/cli` | Package cli is the markitect command line: init, check, lock, refs, graph, impact and lint, which start no agent; apply and verify, which start a Run; desktop, which opens the app over these commands; version and licenses. | [docs/guides/model-language.md#commands](../guides/model-language.md#commands) |
| `internal/adapters/desktop` | Package desktop is Markitect Desktop, the app over the command line that users know as Markitect: one service per user that serves the embedded UI and its JSON routes on a loopback address, and the window that shows it. | [internal/adapters/desktop/README.md](../../internal/adapters/desktop/README.md) |
| `internal/adapters/remotecontainer` | Package remotecontainer starts and removes a Run's remote container from the host, natively on Windows, Linux and macOS with only the Docker CLI. | [internal/runremote/README.md#the-container](../../internal/runremote/README.md#the-container) |
| `internal/adapters/remoteserver` | Package remoteserver is the Run remote inside its container: the Git smart HTTP server with one credential per agent, the pre-receive hook that applies the rules of internal/runremote, the required checks under a throwaway user, and seeding and exporting the bare repository. | [internal/runremote/README.md](../../internal/runremote/README.md) |
| `internal/adapters/repo` | Package repo reads a project's repository through Git: the working tree as Git sees it, a base commit, and the files changed between them. | [docs/guides/model-language.md#the-projects-markitect-folder](../guides/model-language.md#the-projects-markitect-folder) |
| `internal/adapters/rungit` | Package rungit is the host's own Git work for a Run (concept 2.0 §9): a private bare repository that reads the target branch, makes the start bundle, imports what the Run's remote holds, merges and writes the Result commit, and pushes to the target repository. | [internal/run/README.md#host-git](../../internal/run/README.md#host-git) |

## Composition

| Package | Summary | Owning document |
|---|---|---|
| `internal/bootstrap` | Package bootstrap constructs the command line with its adapters: Git for the model commands, and for apply and verify the Run engine with the host's Git, the Run remote in a container and the provider adapters. | [docs/architecture.md#product-flow](../architecture.md#product-flow) |

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
