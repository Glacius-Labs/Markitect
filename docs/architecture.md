# Architecture

How the source is composed, where each responsibility lives and which contracts hold between the packages. Main builds the [concept](concept/README.md). The tool checks a project's model, analyses it and helps reading it; it never writes, keeps no state, starts no agent and steers no work ([DEC-154](decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads), [DEC-097](decisions.md#dec-097-the-language-and-the-tool-neither-need-nor-know-version-control); [where Main is](roadmap.md#where-main-is)). The default is a model of plain files in the [plugin](guides/plugin.md)'s folder `plugin/default/`, which a person copies into a project and which is no part of the binary; the plugin fetches the tool and holds the default.

This page owns the package contracts and the repository layout. The [language reference](guides/language-reference.md) owns the language and the commands, the [project reference](guides/project-reference.md) the project; the [code map](development/code-map.md) lists every package with its owning document.

## Product flow

`cmd/markitect → internal/cli → internal/project, internal/signals, internal/library → internal/lang`.

- `cmd/markitect` only calls `cli.Run` with the arguments and the output streams.
- The command line (`internal/cli`) parses the arguments, finds the project, the nearest folder upward that holds `.markitect/model` or the one `-C` names, makes sure its model folder lies inside it, runs the command, and prints text or JSON. It prints the version and the license notices from `internal/buildinfo`. It writes no file.
- The project (`internal/project`) loads a project through the `os.DirFS` of its folder: the file `.markitect/model`, the model it names, the ignore file `.markitect/ignore`, and the difference between two projects that `diff` prints: their models and, for each module, its files. It reads and returns.
- The signals (`internal/signals`) read a parsed model: its statements, the eight signals and the topic view.
- The library (`internal/library`) builds into the binary the reference values and question catalog of the default's modules, today those of `core`; the default itself lies in `plugin/default/`.
- The language (`internal/lang`) parses and checks the documents of the file system it is handed, and reads the path patterns under `Realized in` that say where each module is built; it writes nothing.

None of the commands writes a file, starts an agent or a process, uses the network or needs credentials. The [language reference](guides/language-reference.md#commands) gives their arguments and exit codes.

## Layers

All Go packages use one root Go module. One rule table in [`internal/tooling/architecture/rules.go`](../internal/tooling/architecture/rules.go) gives every package a layer and its owning document, lists which layers each layer may import and names the exceptions, of which there are none today. The [import gate](#import-gate) enforces it, and the generated [code map](development/code-map.md) shows the table.

| Responsibility | Packages | Depends on |
| --- | --- | --- |
| Version and license notices | `internal/buildinfo` | No other Markitect package |
| The modeling language over a supplied file system: documents, types, sections, links, modules with their path patterns, and `check` ([language reference](guides/language-reference.md)) | `internal/lang` | No other Markitect package |
| The library: the reference values and question catalog of the default's modules, built into the tool ([library](guides/library.md)) | `internal/library` | No other Markitect package |
| The statements, signals and topic view over a parsed model ([signals](guides/signals.md)) | `internal/signals` | The language |
| The project over a supplied file system: the file `.markitect/model`, the project loaded with its model, its ignore file, and the difference between two projects: their models and, for each module, its files ([project reference](guides/project-reference.md)) | `internal/project` | The language |
| The command line: arguments, the project it works in, the commands that read it, `version` and `licenses`, and the text and JSON output; it writes nothing | `internal/cli` | Every layer above, and build data |
| The entrypoint, `markitect` | `cmd/markitect` | The command line |
| Maintenance: the import gate and the code map | `internal/tooling/architecture` | No other Markitect package; separate from product execution |
| Test support, test files only: temporary folders | `internal/testkit` | No Markitect package |
| Standalone: the command smoke | `tools/smoke` | No Markitect package |

The language, the signals and the project read no file of their own accord and start no process; their callers hand them file systems.

### Import gate

`TestRepositoryArchitecture` in `internal/tooling/architecture` inspects all production, test and platform Go files; `go test ./...` runs it, and `go test ./internal/tooling/architecture` runs it alone. It checks unknown and removed packages, layer directions and named exceptions, table completeness, owning-document links, external-test self imports and test-only `testkit` use. Third-party imports remain permitted: this is a Markitect dependency check, not a proof of I/O purity.

Every Go package has one row in the rule table with its layer and owning document, and one package doc comment whose first sentence is its summary. The generated code map is byte-checked. Regenerate it with:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

### Adding, changing or removing a package

A new package root needs an actual consumer and a distinct responsibility. Land each contract with a caller and a concrete implementation, never as an interface-only wave. Add, move or remove the package's row in the rule table, give it a package doc comment and regenerate the code map. Keep one root Go module and reject nested Go modules. Delete obsolete source, commands and gates together ([DEC-139](decisions.md#dec-139-quality-simplicity-and-stability-over-effort-and-compatibility)). Producers own value types; consumers own ports.

## Distribution

A release is a tag `vX.Y.Z` that the [release workflow](../.github/workflows/release.yaml) turns into a GitHub release: binaries for six systems, their checksums and a provenance file ([releasing](releasing.md)). Between releases Main's source declares a development version such as `0.35.0-dev` (`internal/buildinfo`); the retired versions `0.1.0` to `0.14.1` are never reused, and a release sets its own. The tool itself uses no network. The Claude Code plugin in `plugin/`, listed by the marketplace repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins) at the tag of a release, fetches the binary of its own version through its launcher ([the plugin](guides/plugin.md#the-launcher)); the launcher's tests serve a release under the file names the workflow builds. `markitect version` adds the source revision: the VCS revision of a build in a checkout, marked `-modified` for a changed tree, the commit in the module's pseudo-version for `go install github.com/Glacius-Labs/Markitect/cmd/markitect@COMMIT`, or the commit the release workflow injects. The command smoke, `tools/smoke`, builds `markitect` from the repository and drives its commands over a small fresh project; it publishes nothing and starts no agent.

## Repository layout

One Go module at the root, with `cmd/` and `internal/` below it.

| Path | Holds |
|---|---|
| `cmd/` | The Go entrypoint `markitect` |
| `internal/` | The Go packages, including the library (`internal/library`), whose reference values and question catalog of the default's module `core` the binary embeds; the [code map](development/code-map.md) lists every package |
| `plugin/` | The Claude Code plugin `markitect`: the launcher that fetches the tool, its start hooks, and the default in `plugin/default/` ([plugin](guides/plugin.md), [library](guides/library.md)) |
| `tools/` | Independent tools that import no product code: the command smoke |
| `scripts/` | Maintainer scripts: the documentation link check and the CI test and flake reports |
| `docs/` | This documentation |
| `.github/` | The CI, release and flake-report workflows, all started by hand or by a tag |

The architecture gate fails on a Go package without a row in the rule table, and `scripts/check-docs.py` on broken local links.
