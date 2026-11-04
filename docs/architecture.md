# Architecture

How the source is composed, where each responsibility lives and which contracts hold between the packages. No concept is the target right now: the current realization version was removed and a new one is being designed ([DEC-094](concepts/register.md#dec-094-a-fresh-start-for-realization)). What stays is the model language and the model commands, and [Markitect Desktop](../internal/adapters/desktop/README.md) with its chats in containers on the user's token ([agent sessions](../internal/agents/README.md)). The host is native on Windows, Linux and macOS.

This page owns the package contracts. The [model language guide](guides/model-language.md) owns the language, the `.markitect/` layout and the commands; the [code map](development/code-map.md) lists every package with its owning document. V1's Run, Apply, adapter and state architecture is kept at the base commit of T1: [architecture of V1](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/architecture.md); the Command engine of concept 2.0 is kept at the [pre-native archive point](archive.md).

## Product flow

`cmd/markitect → internal/bootstrap → CLI adapter → Project → analysis`.

- `cmd/markitect` only calls `bootstrap.Run`. Bootstrap constructs the CLI with its Git adapter and nothing else.
- The CLI adapter (`internal/adapters/cli`) parses the arguments, opens the repository through its own `Git` port and renders Markdown or JSON. It calls Project to load and check the project, and analysis for `refs`, `graph`, `impact` and `lint`. `init` and `lock` write through the same port.
- The repo adapter (`internal/adapters/repo`) implements that port with Git: the working tree as Git sees it (tracked files and untracked files Git does not ignore), a base commit, and the files changed between them.
- Project (`internal/project`) reads one repository state through its `Repository` port, so the same load runs over the working tree and over a base commit. It parses the manifest and lock, takes the standard Libraries from the binary and other Libraries from `.markitect/libraries/`, compiles the model, accounts for every file and reads the required checks. A project with compile errors still loads as far as its files allow, so `refs`, `graph` and `lint` can run over it.
- Analysis (`internal/analysis`) answers questions about a loaded project: references, the Package graph, impact (the changed and reached Definitions and files) and lint warnings.

None of these commands starts an agent or a container, and none needs credentials. The [commands](guides/model-language.md#commands) section gives their options and exit codes.

`desktop` runs `cmd/markitect → internal/bootstrap → CLI adapter → desktop adapter`. The CLI adapter parses the flags and calls its `Desktop` port; bootstrap implements it with [Markitect Desktop](../internal/adapters/desktop/README.md) (`internal/adapters/desktop`), a loopback service that serves the embedded UI. It reads projects through Project like `check`, and gets Git and `init` from bootstrap: the repo adapter, and the CLI's own `init`. It starts no agent and no container. The native app runs `cmd/markitect-desktop → internal/bootstrap → desktop adapter` with the same service; its shell, Wails windows with a tray and the platform's folder dialog, lives in the entry point and builds only with the `native` tag.

Only the chats of the app need Docker ([DEC-083](concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers)). The host mounts no folder of the user's into a container, and the user's token reaches only the chat session's environment.

## Layers

All Go packages use one root module. One rule table in [`internal/tooling/architecture/rules.go`](../internal/tooling/architecture/rules.go) gives every package a layer and its owning document, lists which layers each layer may import and names the few exceptions. The [import gate](#import-gate) enforces it, and the generated [code map](development/code-map.md) shows the table. The source packager accepts exactly the table's packages below `cmd/` and `internal/`.

| Responsibility | Packages | Depends on |
| --- | --- | --- |
| Model language: Definitions, Kinds, links, visibility, cycles, manifest and lock, over supplied bytes | `internal/model` | No other Markitect package |
| Files built into the binary: the standard Libraries `std`, `rest` and `cli`, and the modeling and Extract skills | `internal/libraries`, `internal/skills` | No other Markitect package |
| Registry pages, ownership by path pattern, file accounting and stale realization | `internal/realization` | Model |
| Loading and checking a project's `.markitect/` folder from one repository state | `internal/project` | Model, the built-in files, realization; its own `Repository` port |
| References, graph, impact with the reached Definitions and files, lint | `internal/analysis` | Project and the packages below it |
| Git, command-line and desktop adapters; no adapter imports another | `internal/adapters/repo`, `internal/adapters/cli`, `internal/adapters/desktop` (Markitect Desktop's service and embedded UI) | Their inward consumers and build data |
| Composition | `internal/bootstrap` | The adapters, the container client and the agent chats |
| Entrypoints | `cmd/markitect`; `cmd/markitect-desktop`, Markitect Desktop's native app | Bootstrap |
| Container engine client, frame protocol and chat session helper | `internal/adapters/container`, `internal/agentd/frame`, `internal/agentd`, `cmd/markitect-agentd` | The frame protocol only |
| What a chat agent starts with: its role, its skill and the conversation so far | `internal/harness` | The built-in files |
| The app's chats on the user's token, with the Claude Code and Codex adapters ([agent sessions](../internal/agents/README.md)) | `internal/agents`, `internal/agents/claude`, `internal/agents/codex` | The harness, the container client and the frame protocol |
| Guarded working-tree writes for bundle installation; version and license notices | `internal/guardedwrite`, `internal/buildinfo` | No other Markitect package |
| Maintenance | `internal/tooling/*`, `cmd/markitect-release` | The guarded writer and build data; separate from product execution |
| Test support, test files only | `internal/testkit` | No Markitect package |
| Standalone: consumer bootstrap and source smoke | `packaging/bootstrap`, `tools/smoke` | No Markitect package |

The pure packages (`model`, `realization`, `analysis`, `harness`) read no files and start no process; their callers hand them bytes. `model` and `realization` return every error as a diagnostic with a code, which the [model language guide](guides/model-language.md) lists. The container client, the frame protocol and the session helper serve the chats of the [agent sessions](../internal/agents/README.md).

### Import gate

`TestRepositoryArchitecture` in `internal/tooling/architecture` inspects all production, test and platform Go files; `go test ./...` runs it, and `go test ./internal/tooling/architecture` runs it alone. It checks unknown and removed packages, layer directions and named exceptions, table completeness, owning-document links, external-test self imports and test-only `testkit` use. Third-party imports remain permitted: this is a Markitect dependency check, not a proof of I/O purity.

Every Go package has one row in the rule table with its layer and owning document, and one package doc comment whose first sentence is its summary. The generated code map is byte-checked. Regenerate it with:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

### Adding, changing or removing a package

A new package root needs an actual consumer and a distinct responsibility. Land each contract with a caller and a concrete implementation, never as an interface-only wave. Add, move or remove the package's row in the rule table, give it a package doc comment and regenerate the code map. Source packaging takes its roots from the same table; update its data selection and tests for a new embedded asset. Keep one root module and reject nested modules. Delete obsolete source, commands and gates together ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)). Producers own value types; consumers own ports.

## Distribution

Source packaging, bundling and guarded installation are independent maintenance services; Main has no release publisher. Main's source declares the version `0.0.0-dev` (`internal/buildinfo`); the retired versions `0.1.0` to `0.14.1` are never reused, and a release sets its own. `markitect version` adds the source revision: the VCS revision of a build in a checkout, marked `-modified` for a changed tree, or the commit in the module's pseudo-version for `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main`. Source archives use the root `cmd/` and `internal/` layout and include the Markdown of the standard Libraries and skills that the binary embeds. The maintenance-only `release.Input` keeps fixed source files, modes and commit provenance for packaging and installation. The source smoke packages the source, tests the standalone bootstrap and drives the model commands over a small project; it publishes nothing and starts no agent. See [distribution](../packaging/README.md) and its [release operations](../packaging/README.md#release-operations).

Earlier product generations are listed in the [archive index](archive.md).
