# Architecture

How the source is composed, where each responsibility lives and which contracts hold between the packages. This branch builds [concept 4.0](concepts/v4-concept.md), the target ([DEC-098](concepts/register.md#dec-098-concept-40-is-the-target)). T1 built the spec language and its commands; [the record](../internal/run/README.md) (`status`, `mark`) says what a driver's agents have to verify and records their verdicts, and the tool steers no work and uses no Git ([DEC-096](concepts/register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](concepts/register.md#dec-097-markitect-uses-no-git), [DEC-099](concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)).

This page owns the package contracts. The [model language guide](guides/model-language.md) owns the language, the `.markitect/` layout and the commands; the [code map](development/code-map.md) lists every package with its owning document. V1's Run, Apply, adapter and state architecture is kept at the base commit of T1: [architecture of V1](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/architecture.md).

## Product flow

`cmd/markitect → internal/bootstrap → CLI adapter → Project → analysis`.

- `cmd/markitect` only calls `bootstrap.Run`. Bootstrap constructs the CLI with the folder adapter, the engine of the record and the releases `update` installs.
- The CLI adapter (`internal/adapters/cli`) parses the arguments, opens the project folder through its own `Folders` port and renders Markdown or JSON. It calls Project to load and check the project, and analysis for `list`, `doc` and `vet`. `init`, `update` and `tidy` write through the same port.
- The folder adapter (`internal/adapters/folder`) implements that port without Git: the project folder is the nearest one with the manifest, and its files are what a walk finds without what its `.gitignore` files exclude, read as text.
- Project (`internal/project`) reads one set of files through its `Repository` port, so the same load runs over the folder and over a copy of its spec, such as the one a certificate of 0.20.0 holds, which the migration reads. It parses the manifest and lock, takes the standard Libraries from the binary and other Libraries from `.markitect/libraries/`, compiles the model and accounts for every file. A project with compile errors still loads as far as its files allow, so `list`, `doc` and `vet` can run over it.
- Analysis (`internal/analysis`) answers questions about a loaded project: what it holds, who links to what, what depends on what, and the warnings of vet.

None of these commands starts an agent, a container or a Command, and none needs credentials. The [commands](guides/model-language.md#commands) section gives their options and exit codes.

`status`, `mark` and `hook` run `cmd/markitect → internal/bootstrap → CLI adapter → the record`. Bootstrap builds [the record](../internal/run/README.md) (`internal/run`) over the folder adapter, with the lock of each project's record in the user's cache folder; the record itself, `.markitect/review.sum`, lies in the project. The driver, by default the owner's own Claude Code session through the skills of [DEC-100](concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role), such as `/markitect-build` and `/markitect-review`, starts the agents and makes every Git step: nothing in the tool starts or drives an agent, which calls `markitect` like any other tool. `update` also runs the migration that writes the record from the newest certificate of 0.20.0.

The default driver needs Claude Code on the user's own login, and no container: its agents are subagents of the owner's session, with the user's rights, the builder in a worktree of its own and a reviewer alone in the owner's folder ([concept 4.0 §7](concepts/v4-concept.md#7-how-sure-we-are)).

## Layers

All Go packages use one root module. One rule table in [`internal/tooling/architecture/rules.go`](../internal/tooling/architecture/rules.go) gives every package a layer and its owning document, lists which layers each layer may import and names the few exceptions. The [import gate](#import-gate) enforces it, and the generated [code map](development/code-map.md) shows the table. The source packager accepts exactly the table's packages below `cmd/` and `internal/`.

| Responsibility | Packages | Depends on |
| --- | --- | --- |
| Model language: Definitions, Kinds, links, visibility, cycles, manifest and lock, over supplied bytes | `internal/model` | No other Markitect package |
| Files built into the binary: the standard Libraries `std`, `rest` and `cli`, the design skill, and the harness of the work: its skills, agents and routine prompts | `internal/libraries`, `internal/skills` | No other Markitect package |
| Registry pages, ownership by path pattern, file accounting and stale realization | `internal/realization` | Model |
| Loading and checking a project's `.markitect/` folder from one set of files | `internal/project` | Model, the built-in files, realization; its own `Repository` port |
| What a project holds and who links to what, the Package graph, vet | `internal/analysis` | Project and the packages below it |
| The record: the inputs of each Definition, `review.sum`, `status`, `mark`, the hook and the migration from certificates | `internal/run` | Project and the packages below it |
| Project-folder, command-line and release adapters; no adapter imports another | `internal/adapters/folder`, `internal/adapters/cli`, `internal/adapters/upgrade` | Their inward consumers and build data |
| Composition | `internal/bootstrap` | The adapters and the record |
| Entrypoints | `cmd/markitect` | Bootstrap |
| Guarded working-tree writes for bundle installation; version and license notices | `internal/guardedwrite`, `internal/buildinfo` | No other Markitect package |
| Maintenance | `internal/tooling/*`, `cmd/markitect-release` | The guarded writer and build data; separate from product execution |
| Test support, test files only | `internal/testkit` | No Markitect package |
| Standalone: consumer bootstrap and source smoke | `packaging/bootstrap`, `tools/smoke` | No Markitect package |

The pure packages (`model`, `realization`, `analysis`) read no files and start no process; their callers hand them bytes. `model` and `realization` return every error as a diagnostic with a code, which the [model language guide](guides/model-language.md) lists.

### Import gate

`TestRepositoryArchitecture` in `internal/tooling/architecture` inspects all production, test and platform Go files; `go test ./...` runs it, and `go test ./internal/tooling/architecture` runs it alone. It checks unknown and removed packages, layer directions and named exceptions, table completeness, owning-document links, external-test self imports and test-only `testkit` use. Third-party imports remain permitted: this is a Markitect dependency check, not a proof of I/O purity.

Every Go package has one row in the rule table with its layer and owning document, and one package doc comment whose first sentence is its summary. The generated code map is byte-checked. Regenerate it with:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

### Adding, changing or removing a package

A new package root needs an actual consumer and a distinct responsibility. Land each contract with a caller and a concrete implementation, never as an interface-only wave. Add, move or remove the package's row in the rule table, give it a package doc comment and regenerate the code map. Source packaging takes its roots from the same table; update its data selection and tests for a new embedded asset. Keep one root module and reject nested modules. Delete obsolete source, commands and gates together ([DEC-032](concepts/register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)). Producers own value types; consumers own ports.

## Distribution

Source packaging, bundling and guarded installation are independent maintenance services. A release is a tag `vX.Y.Z` that the [release workflow](../.github/workflows/release.yaml) turns into a GitHub release: binaries for six systems, their checksums, the source bundle and a provenance file ([DEC-095](concepts/register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself); [release operations](../packaging/README.md#release-operations)). Between releases Main's source declares a development version such as `0.0.0-dev` (`internal/buildinfo`, which also compares versions: `Newer`, `IsDev`); the retired versions `0.1.0` to `0.14.1` are never reused, and a release sets its own. `internal/adapters/upgrade` is the one package that uses the network: for `markitect update` it finds the newest tag, downloads the binary for the system, checks it against the release's checksums and replaces the running executable. The Claude Code plugin in `plugin/`, listed by the marketplace repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins) at the tag of a release, fetches the same binary through its launcher; a test has the launcher fetch the file names that the updater uses, and the workflow builds files under those names. `markitect version` adds the source revision: the VCS revision of a build in a checkout, marked `-modified` for a changed tree, or the commit in the module's pseudo-version for `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main`. Source archives use the root `cmd/` and `internal/` layout and include the Markdown of the standard Libraries and skills that the binary embeds. The maintenance-only `release.Input` keeps fixed source files, modes and commit provenance for packaging and installation. The source smoke packages the source, tests the standalone bootstrap and drives the model commands over a small project; it publishes nothing and starts no agent. See [distribution](../packaging/README.md) and its [release operations](../packaging/README.md#release-operations).

Earlier product generations are listed in the [archive index](archive.md).
