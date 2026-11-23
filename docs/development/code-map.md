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
| `library` | The library of packages and templates, and the reference values and question catalogs built into the tool. | nothing | 1 |
| `signals` | The statements of a spec, its signals and the topic view, over a parsed spec. | `language` | 1 |
| `project` | A project over a supplied file system: the file .markitect/spec, the project loaded with its spec and the difference between the specs of two projects. | `language` | 1 |
| `command-line` | The command line: its arguments, the project it works in, the commands that read its spec, version and licenses, and the text and JSON output. It writes nothing. | `build`, `language`, `library`, `signals`, `project` | 1 |
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
| `internal/library` | Package library is the library of Markitect: the packages of types and the templates that a person copies into a spec by hand, and per package its reference values and its question catalog, which the tool reads and never copies. | [docs/guides/library.md](../guides/library.md) |

## Signals

| Package | Summary | Owning document |
|---|---|---|
| `internal/signals` | Package signals finds where reading a spec pays off: it splits the prose of the documents into statements, finds the marker words in them, computes the eight signals of the spec itself and gathers the topic view of a document, a type or a folder. | [docs/guides/signals.md](../guides/signals.md) |

## Project

| Package | Summary | Owning document |
|---|---|---|
| `internal/project` | Package project is a Markitect project, over a file system rooted at the project folder: the file .markitect/spec that makes a folder a project and names its spec folder, the project loaded with its spec, and the difference between the specs of two projects. | [docs/guides/project-reference.md](../guides/project-reference.md) |

## Command line

| Package | Summary | Owning document |
|---|---|---|
| `internal/cli` | Package cli is the command line of markitect: it parses the arguments, finds the project they work in, reads its spec and prints what the language finds, as text or as JSON. | [docs/guides/language-reference.md#commands](../guides/language-reference.md#commands) |

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
| `tools/smoke` | Command smoke builds markitect from the repository and drives its commands over a small project that it builds by hand, the way a person does: an empty folder, the packages and templates of the library copied into a spec folder, and the file .markitect/spec. | [docs/architecture.md#distribution](../architecture.md#distribution) |
