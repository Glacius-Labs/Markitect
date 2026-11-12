# Code map

This page lists every Go package in the repository with its layer, the first sentence of its doc comment and its owning document. Read the owning document before you change a package, and update it when the package's behavior changes.

The page is generated from the rule table in [rules.go](../../internal/tooling/architecture/rules.go) and the package doc comments. Do not edit it by hand. Change the table or the doc comment, then regenerate the page from the repository root:

```powershell
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

`TestRepositoryArchitecture` checks every production, test and platform file against the table: a package imports only the layers its own layer lists and the named exceptions, only test files import a test-only layer, and a standalone layer imports no Markitect package. Every Go package needs exactly one row, an existing owning document and one package doc comment. [Architecture](../architecture.md#layers) describes the layers.

## Layers

| Layer | Meaning | May import | Packages |
|---|---|---|---|
| `spec` | The spec language over supplied bytes: Definitions, Kinds, links, visibility, cycles, manifest and lock. | nothing | 1 |
| `assets` | Files built into the binary: the standard Libraries, the skills and the harness of the work. | nothing | 2 |
| `harness` | harness.yaml at user and project level, resolved key by key, and the agents' front matter rendered from it. | nothing | 1 |
| `domain` | Registry pages, ownership and file accounting over a compiled spec. | `spec` | 1 |
| `project` | Loads and checks a project's Markitect folder from the files of a project folder. | `spec`, `assets`, `harness`, `domain` | 1 |
| `service` | Questions about a loaded project: what it holds, what depends on what, and vet. | `spec`, `assets`, `domain`, `project` | 1 |
| `run` | The record of what was verified, without Git: the inputs of each Definition, review.sum, status and mark. | `spec`, `assets`, `domain`, `project` | 1 |
| `guard` | Which role may write what and start whom, the hook that holds it and its entries in a project's settings, over supplied bytes. | `project` | 1 |
| `setup` | What Markitect writes into a project: its first files, the skills and the harness, the settings, the Git attributes, the ignore line and the lock. | `spec`, `assets`, `harness`, `project`, `guard` | 1 |
| `build` | Version identity and embedded license notices. | nothing | 1 |
| `adapter` | Concrete project-folder, command-line and update adapters; no adapter imports another. | `spec`, `assets`, `harness`, `domain`, `project`, `service`, `run`, `guard`, `setup`, `build` | 3 |
| `composition` | Constructs the command line with its adapters. | `spec`, `project`, `service`, `run`, `adapter` | 1 |
| `command` | Thin entrypoints to product composition. | `composition` | 1 |
| `maintenance` | The import gate and the code map, separate from product execution. | nothing | 1 |
| `test-support` | Hermetic fixtures. Only test files may import it. It imports no Markitect package, not even from tests. | nothing | 1 |
| `standalone` | The command smoke. It imports no Markitect package, not even from tests. | nothing | 1 |

## Exceptions

| From | To | Files | Reason |
|---|---|---|---|
| `internal/run` | `internal/adapters/folder` | tests only | The record is tested on real folders. |

## Spec language

| Package | Summary | Owning document |
|---|---|---|
| `internal/spec` | Package spec compiles the spec from explicitly supplied Markdown files: Definitions with front matter, Packages as folders with an anchor, Libraries of Kinds, links by path, visibility and the Package dependency graph. | [docs/guides/spec-language.md#definitions](../guides/spec-language.md#definitions) |

## Built-in files

| Package | Summary | Owning document |
|---|---|---|
| `internal/libraries` | Package libraries holds the standard Libraries that ship inside the markitect binary: std, rest and cli, each a folder of Kind Definitions. | [docs/guides/spec-language.md#kinds-and-libraries](../guides/spec-language.md#kinds-and-libraries) |
| `internal/skills` | Package skills holds what ships inside the markitect binary for agents: the harness for Claude Code, which markitect init installs into a project. | [docs/guides/spec-language.md#commands](../guides/spec-language.md#commands) |

## Harness configuration

| Package | Summary | Owning document |
|---|---|---|
| `internal/harness` | Package harness is the configuration of the harness of the work (DEC-103): one file format, harness.yaml, at two levels. | [docs/guides/project-workflow.md#strategies-and-configuration](../guides/project-workflow.md#strategies-and-configuration) |

## Realization record

| Package | Summary | Owning document |
|---|---|---|
| `internal/realization` | Package realization checks the realization record beside the spec: one registry page per Package with the path patterns it owns and the files it realizes, and the ignore file. | [docs/guides/spec-language.md#the-realization-record](../guides/spec-language.md#the-realization-record) |

## Project

| Package | Summary | Owning document |
|---|---|---|
| `internal/project` | Package project loads a project's Markitect folder from a set of files, the project folder, and checks it as a whole: manifest, lock, Libraries, spec and registry. | [docs/guides/spec-language.md#the-projects-markitect-folder](../guides/spec-language.md#the-projects-markitect-folder) |

## Analysis

| Package | Summary | Owning document |
|---|---|---|
| `internal/analysis` | Package analysis answers questions about a loaded project without starting agents: who links to a Definition, the Package dependency graph, the views of list and doc, and the warnings and notes of vet, which analyse the spec's design like code. | [docs/guides/spec-language.md#vet-warnings-and-notes](../guides/spec-language.md#vet-warnings-and-notes) |

## The record

| Package | Summary | Owning document |
|---|---|---|
| `internal/run` | Package run is the record of what was verified (DEC-099): for each Definition, .markitect/review.sum holds the inputs at which a reviewer last verified it, modelled on Go's test cache. | [internal/run/README.md](../../internal/run/README.md) |

## The agents' rights

| Package | Summary | Owning document |
|---|---|---|
| `internal/guard` | Package guard holds the agents' rights: which role may write what and start whom, the hook that holds them before a tool call (DEC-100), and its entries in a project's Claude Code settings. | [internal/guard/README.md](../../internal/guard/README.md) |

## Project setup

| Package | Summary | Owning document |
|---|---|---|
| `internal/setup` | Package setup is what Markitect writes into a project: the first files of a new one, the harness with its agents rendered from the harness configuration, the hooks and the worktree base in Claude Code's settings, the record's Git attributes, the ignore line for Claude Code's worktrees and the lock. | [docs/guides/project-workflow.md#1-set-up-the-project](../guides/project-workflow.md#1-set-up-the-project) |

## Build data

| Package | Summary | Owning document |
|---|---|---|
| `internal/buildinfo` | Package buildinfo owns the version of this build and the upstream license notices that `markitect version` and `markitect licenses` print. | [docs/architecture.md#distribution](../architecture.md#distribution) |

## Adapters

| Package | Summary | Owning document |
|---|---|---|
| `internal/adapters/cli` | Package cli is the markitect command line: install, update, init and tidy; check, vet, list and doc over the spec; status and mark over the record of what was verified (DEC-099); config over the harness configuration (DEC-103); the hook; version and licenses. | [docs/guides/spec-language.md#commands](../guides/spec-language.md#commands) |
| `internal/adapters/folder` | Package folder reads and writes a project folder without Git (DEC-097): it walks the folder and skips what the project's .gitignore files exclude, read as plain text, nested ones included. | [internal/run/README.md#the-folder](../../internal/run/README.md#the-folder) |
| `internal/adapters/upgrade` | Package upgrade finds the newest Markitect release, downloads the binary for this platform, checks it against the release's checksums and replaces the running executable. | [docs/guides/project-workflow.md#8-keep-it-current](../guides/project-workflow.md#8-keep-it-current) |

## Composition

| Package | Summary | Owning document |
|---|---|---|
| `internal/bootstrap` | Package bootstrap constructs the command line with its adapters: the project folder on disk, the record over it with its lock in the user's cache folder, and the releases update installs. | [docs/architecture.md#product-flow](../architecture.md#product-flow) |

## Product commands

| Package | Summary | Owning document |
|---|---|---|
| `cmd/markitect` | Command markitect is the Markitect command line: install, update, init, tidy, check, vet, list, doc, status, mark, version and licenses. | [docs/guides/spec-language.md#commands](../guides/spec-language.md#commands) |

## Maintenance

| Package | Summary | Owning document |
|---|---|---|
| `internal/tooling/architecture` | Package architecture holds the repository's one architecture rule table and enforces it without compiling the inspected code: every production, test and platform-specific file imports only what its package's layer allows. | [docs/architecture.md#layers](../architecture.md#layers) |

## Test support

| Package | Summary | Owning document |
|---|---|---|
| `internal/testkit` | Package testkit provides hermetic Git and file-system fixtures for tests. | [docs/architecture.md#layers](../architecture.md#layers) |

## Standalone tools

| Package | Summary | Owning document |
|---|---|---|
| `tools/smoke` | Command smoke builds markitect from the repository and drives its commands over a small project with exact assertions. | [docs/architecture.md#distribution](../architecture.md#distribution) |
