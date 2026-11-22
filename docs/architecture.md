# Architecture

How the source is composed, where each responsibility lives and which contracts hold between the packages. Main builds the [concept](concept/README.md). The tool checks and analyses a project's spec, computes from the state what is changed, suspect or unexplained, binds verdicts, and checks the steps and the landing of a job; it starts no agent, steers no work and uses no Git ([DEC-097](decisions.md#dec-097-markitect-uses-no-git); [where Main is](roadmap.md#where-main-is)). The jobs are skills and agents of the [plugin](guides/plugin.md), run by a session or a runner.

This page owns the package contracts and the repository layout. The [language reference](guides/language-reference.md) owns the language and the commands, the [project reference](guides/project-reference.md) the project and its state; the [code map](development/code-map.md) lists every package with its owning document.

## Product flow

`cmd/markitect → internal/cli → internal/state, internal/signals, internal/edit, internal/library → internal/lang`.

- `cmd/markitect` only calls `cli.Run` with the arguments and the output streams.
- The command line (`internal/cli`) parses the arguments, finds the project, the nearest folder upward that holds `.markitect/version` or the one `-C` names, and prints text or JSON. It writes `.markitect/` and the spec folder for `init`, carries out the change sets of `new`, `move` and `apply` all or nothing, checking first, writing through temporary files and undoing every step when one fails, writes the state and the history for `mark` and `move` under a lock in the user's cache folder, and prints the version and the license notices from `internal/buildinfo`.
- The project and its state (`internal/state`) load a project through the `os.DirFS` of its folder: the layout of `.markitect/`, the spec, the walk of the files without what the ignore files exclude, their hashes, the state with `status`, `mark` and `move`, the history, the reports, settings and control samples, the write table with the step and landing checks, `compare` and the figures. It reads and returns; the command line writes.
- The signals (`internal/signals`) read a parsed spec: its statements, the eight signals and the topic view.
- The change sets (`internal/edit`) compute what `new`, `move` and `apply` write, over supplied file systems.
- The library (`internal/library`) is built into the binary: the packages of types, the templates, the types the tool owns, the reference values and the question catalogs.
- The language (`internal/lang`) parses and checks the documents of the file system it is handed and writes nothing.

None of the commands starts an agent or a process, uses the network or needs credentials. The [language reference](guides/language-reference.md#commands) gives their arguments and exit codes.

## Layers

All Go packages use one root module. One rule table in [`internal/tooling/architecture/rules.go`](../internal/tooling/architecture/rules.go) gives every package a layer and its owning document, lists which layers each layer may import and names the exceptions, of which there are none today. The [import gate](#import-gate) enforces it, and the generated [code map](development/code-map.md) shows the table.

| Responsibility | Packages | Depends on |
| --- | --- | --- |
| Version and license notices | `internal/buildinfo` | No other Markitect package |
| The spec language over a supplied file system: documents, types, sections, links and `check` ([language reference](guides/language-reference.md)) | `internal/lang` | No other Markitect package |
| The library built into the tool: packages, templates and the types the tool owns, of reports, settings and control samples ([library](guides/library.md)) | `internal/library` | No other Markitect package |
| The change sets of `new`, `move` and `apply` over supplied file systems: files to move and files to write | `internal/edit` | The language |
| The statements, signals and topic view over a parsed spec ([signals](guides/signals.md)) | `internal/signals` | The language |
| The project over a supplied file system: the `.markitect/` layout, the project loaded with its spec, the walk of its files, their hashes, the state with `status`, `mark` and `move`, the history, the reports, settings and control samples ([reports](guides/reports.md)), the write table with the step and landing checks ([rights](guides/rights.md)), `compare`, the change signals and the figures of the factory ([project reference](guides/project-reference.md)) | `internal/state` | The language, the library and the change sets |
| The command line: arguments, the project it works in, `init`, `version` and `licenses`, the spec folder and the state it reads and writes, the lock of `mark`, and the output | `internal/cli` | Every layer above, and build data |
| The entrypoint, `markitect` | `cmd/markitect` | The command line |
| Maintenance: the import gate and the code map | `internal/tooling/architecture` | No other Markitect package; separate from product execution |
| Test support, test files only: temporary folders | `internal/testkit` | No Markitect package |
| Standalone: the command smoke | `tools/smoke` | No Markitect package |

The language, the library, the change sets, the signals and the project read no file of their own accord and start no process; their callers hand them file systems.

### Import gate

`TestRepositoryArchitecture` in `internal/tooling/architecture` inspects all production, test and platform Go files; `go test ./...` runs it, and `go test ./internal/tooling/architecture` runs it alone. It checks unknown and removed packages, layer directions and named exceptions, table completeness, owning-document links, external-test self imports and test-only `testkit` use. Third-party imports remain permitted: this is a Markitect dependency check, not a proof of I/O purity.

Every Go package has one row in the rule table with its layer and owning document, and one package doc comment whose first sentence is its summary. The generated code map is byte-checked. Regenerate it with:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

### Adding, changing or removing a package

A new package root needs an actual consumer and a distinct responsibility. Land each contract with a caller and a concrete implementation, never as an interface-only wave. Add, move or remove the package's row in the rule table, give it a package doc comment and regenerate the code map. Keep one root module and reject nested modules. Delete obsolete source, commands and gates together ([DEC-139](decisions.md#dec-139-quality-simplicity-and-stability-over-effort-and-compatibility)). Producers own value types; consumers own ports.

## Distribution

A release is a tag `vX.Y.Z` that the [release workflow](../.github/workflows/release.yaml) turns into a GitHub release: binaries for six systems, their checksums and a provenance file ([releasing](releasing.md)). Between releases Main's source declares a development version such as `0.35.0-dev` (`internal/buildinfo`); the retired versions `0.1.0` to `0.14.1` are never reused, and a release sets its own. The tool itself uses no network. The Claude Code plugin in `plugin/`, listed by the marketplace repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins) at the tag of a release, fetches the binary of its own version through its launcher ([the plugin](guides/plugin.md#the-launcher)); the launcher's tests serve a release under the file names the workflow builds. `markitect version` adds the source revision: the VCS revision of a build in a checkout, marked `-modified` for a changed tree, the commit in the module's pseudo-version for `go install github.com/Glacius-Labs/Markitect/cmd/markitect@COMMIT`, or the commit the release workflow injects. The command smoke, `tools/smoke`, builds `markitect` from the repository and drives its commands over a small fresh project; it publishes nothing and starts no agent.

## Repository layout

One Go module at the root, with `cmd/` and `internal/` below it.

| Path | Holds |
|---|---|
| `cmd/` | The Go entrypoint `markitect` |
| `internal/` | The Go packages, including the library (`internal/library`) that the binary embeds; the [code map](development/code-map.md) lists every package |
| `plugin/` | The Claude Code plugin `markitect`, with its skills, agents, launcher and start hooks ([plugin](guides/plugin.md)) |
| `tools/` | Independent tools that import no product code: the command smoke |
| `scripts/` | Maintainer scripts: the documentation link check and the CI test and flake reports |
| `docs/` | This documentation |
| `.github/` | The CI, release and flake-report workflows, all started by hand or by a tag |

The architecture gate fails on a Go package without a row in the rule table, and `scripts/check-docs.py` on broken local links.
