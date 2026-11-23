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
| `build` | Version identity and embedded license notices. | nothing | 1 |
| `language` | The spec language over a supplied file system: documents, types, sections, links and check. | nothing | 1 |
| `library` | The library built into the tool: packages, templates, the skills of the standard, reference values and question catalogs. | nothing | 1 |
| `editing` | The change sets of the commands that write a spec: new and apply, over supplied file systems. | `language` | 1 |
| `signals` | The statements of a spec, its signals and the topic view, over a parsed spec. | `language` | 1 |
| `project` | A project over a supplied file system: the .markitect/ layout, the project loaded with its spec and the comparison of the specs of two projects. | `language` | 1 |
| `command-line` | The command line: its arguments, the project it works in, init with its templates, skills and entry in AGENTS.md, version and licenses, the spec folder it reads and writes, and its text and JSON output. | `build`, `language`, `library`, `editing`, `signals`, `project` | 1 |
| `entrypoint` | The thin entrypoint of markitect. | `command-line` | 1 |
| `maintenance` | The import gate and the code map, separate from product execution. | nothing | 1 |
| `test-support` | Hermetic fixtures. Only test files may import it. It imports no Markitect package, not even from tests. | nothing | 1 |
| `standalone` | The command smoke. It imports no Markitect package, not even from tests. | nothing | 1 |

## Build data

| Package | Summary | Owning document |
|---|---|---|
| `internal/buildinfo` | Package buildinfo owns the version of this build and the upstream license notices that `markitect version` and `markitect licenses` print. | [docs/architecture.md#distribution](../architecture.md#distribution) |

## Language

| Package | Summary | Owning document |
|---|---|---|
| `internal/lang` | Package lang is the spec language over supplied files: it parses documents, with their title, type line, sections and links, loads a spec from a file system and checks it. | [docs/guides/language-reference.md#documents](../guides/language-reference.md#documents) |

## Library

| Package | Summary | Owning document |
|---|---|---|
| `internal/library` | Package library is the library built into the tool: the packages of types, the templates that apply and init copy into a spec, the skills of the standard that init copies into a project, and per package its reference values and its question catalog, which the tool reads and never copies. | [docs/guides/library.md](../guides/library.md) |

## Editing

| Package | Summary | Owning document |
|---|---|---|
| `internal/edit` | Package edit computes the changes of the commands that write a spec: new and apply. | [docs/guides/language-reference.md#commands-that-write](../guides/language-reference.md#commands-that-write) |

## Signals

| Package | Summary | Owning document |
|---|---|---|
| `internal/signals` | Package signals finds where reading a spec pays off: it splits the prose of the documents into statements, finds the marker words in them, computes the eight signals of the spec itself and gathers the topic view of a document, a type or a folder. | [docs/guides/signals.md](../guides/signals.md) |

## Project

| Package | Summary | Owning document |
|---|---|---|
| `internal/project` | Package project is a Markitect project, over a file system rooted at the project folder: the layout of .markitect/, the project loaded with its spec, and the comparison of the specs of two projects. | [docs/guides/project-reference.md](../guides/project-reference.md) |

## Command line

| Package | Summary | Owning document |
|---|---|---|
| `internal/cli` | Package cli is the command line of markitect: it parses the arguments, finds the project they work in, sets one up with init, reads its spec, prints what the language finds, as text or as JSON, and applies the changes of the commands that write. | [docs/guides/language-reference.md#commands](../guides/language-reference.md#commands) |

## Entrypoint

| Package | Summary | Owning document |
|---|---|---|
| `cmd/markitect` | Command markitect is the Markitect tool: show, table, topic, questions and compare to look; check to check; new and apply to change the spec; and init, version and licenses to set up. | [docs/guides/language-reference.md#commands](../guides/language-reference.md#commands) |

## Maintenance

| Package | Summary | Owning document |
|---|---|---|
| `internal/tooling/architecture` | Package architecture holds the repository's one architecture rule table and enforces it without compiling the inspected code: every production, test and platform-specific file imports only what its package's layer allows. | [docs/architecture.md#layers](../architecture.md#layers) |

## Test support

| Package | Summary | Owning document |
|---|---|---|
| `internal/testkit` | Package testkit provides file-system fixtures for tests. | [docs/architecture.md#layers](../architecture.md#layers) |

## Standalone tools

| Package | Summary | Owning document |
|---|---|---|
| `tools/smoke` | Command smoke builds markitect from the repository and drives its commands over a small fresh project with exact assertions. | [docs/architecture.md#distribution](../architecture.md#distribution) |
