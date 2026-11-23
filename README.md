<h1 align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/markitect-lockup.svg">
    <img src="docs/assets/markitect-lockup-print.svg" alt="Markitect" width="360">
  </picture>
</h1>

<p align="center">
  Architect in Markdown. Agents build to spec.
</p>

<p align="center">
  <a href="https://github.com/Glacius-Labs/Markitect/actions/workflows/ci.yaml"><img src="https://github.com/Glacius-Labs/Markitect/actions/workflows/ci.yaml/badge.svg" alt="CI"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/Glacius-Labs/Markitect" alt="Go version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License: Apache-2.0"></a>
  <a href="#status"><img src="https://img.shields.io/badge/status-active%20development-orange" alt="Status: active development"></a>
</p>

<p align="center">
  <a href="#why-markitect">Why</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#how-it-works">How it works</a> ·
  <a href="#example-one-rule-changes">Example</a> ·
  <a href="#install">Install</a> ·
  <a href="#status">Status</a> ·
  <a href="#documentation">Documentation</a>
</p>

## Why Markitect

When agents build most of your project, two things go missing that used to come with writing and reviewing the code yourself: trust that the code is what you meant, and an overview of how the project stands. Reading everything again would bring both back and make you the bottleneck.

The usual fixes have no measure. A session that checks whether everything fits has nothing to fit it against. More rules and skills are more to maintain and more for an agent to overlook. Prompts, tickets and most specs in spec-driven development describe one change and go stale once it is made. What is missing is a standing statement of what the project as a whole should be.

Markitect gives the project that statement, and agents build to it.

**You architect in Markdown.** The spec is the plan of the whole project: one place you own, where what is written is binding. You decide what the project is, how it is divided and which rules hold, and an agent writes it down with you. The spec is a typed, declarative specification language in Markdown: documents with a type and ordinary links, which say what should hold, not how. The `markitect` tool checks its structure, never its meaning.

**Agents build to spec.** Code, tests, configuration and documentation are the implementations of the spec, and where they differ from it, the spec is right. Agents build to it, and their work is checked against it, so that nobody has to read the work to trust it.

**Your way of working is yours.** How your team works, how you plan, what a story or a report is and where the work lies, you describe in your spec too, with general types; skills, agents and scripts in your repository realize it. Markitect ships the language, a default module of general types to start from, and the `markitect` tool, which checks the spec, analyses it and helps reading it. The tool never writes and keeps no state.

Markitect builds on Terraform's declared configuration, static typing and the characteristics of good requirements, and a project's way of working uses the coding agents' own skills, subagents and scheduled tasks. The [vision](docs/vision.md#why-markitect-exists) gives the full reasoning.

> [!IMPORTANT]
> **Early access.** The tool on Main is described by the [concept](docs/concept/README.md). No release of it exists yet: the latest release and the plugin in the marketplace are still the earlier 0.34.0, of an earlier concept. Until the first release, build the tool from source; the plugin on Main is a development version, whose launcher starts that build ([install](#install)). Commands and files may still change. Try it on a small project you can throw away, read the [limits](#limits), and tell us in [Issues](https://github.com/Glacius-Labs/Markitect/issues) what worked and what broke.

## Quick start

Build the tool from source, with Go 1.27.1 or newer, and put Go's `bin` folder on your `PATH`:

```bash
go install github.com/Glacius-Labs/Markitect/cmd/markitect@main
```

A project is set up by hand. In an empty folder:

1. **Name the spec folder.** Write the file `.markitect/spec` with the one line `docs/spec`.
2. **Copy the default.** From a clone of this repository, copy the module `internal/library/core/`, six general types such as Rule and Role, to `docs/spec/core/`.
3. **Write your modules.** Each module is a folder with a `README.md` that says what it is for and lists under `Contents` what other modules may link, such as `docs/spec/lending/`. Write its documents with your agent.
4. **Check it.** `markitect check` names what is off: a missing section, a link to no document, a link into what a module keeps internal. Fix it until `check` reports nothing.

The [project reference](docs/guides/project-reference.md#setting-up-a-project-by-hand) gives the details, the [library](docs/guides/library.md) the default, and the [language reference](docs/guides/language-reference.md) modules and every command. The plugin in [`plugin/`](plugin/) only fetches the tool for Claude Code ([the plugin](docs/guides/plugin.md)).

## How it works

The [concept](docs/concept/README.md) describes the whole; this is the short form.

1. **You write the spec** with an agent. A document is an ordinary Markdown file: a title, a type line that links its type, and the sections the type asks for. Folders group documents into modules, and each module's README says what the others may link.
2. **`markitect check` checks its structure:** titles, types, missing or empty sections, references and values that do not fit their property, links that point to no document, links into what a module keeps internal, and cycles between modules. It never judges meaning.
3. **The tool helps reading it.** `show` gives a document with what it links to and what links to it, `table` every object of a type, `topic` everything about a document, a type or a folder with the signals that say where reading pays off, and `questions` the competency questions of the default's types.
4. **Agents build to it,** in the way of working your spec describes. A change to the spec is seen in the repository's diff like every change, and `markitect diff` shows it document by document against another state of the project.

No `markitect` command writes a file or starts an agent.

## Example: one rule changes

A small library allows each member five open loans. The limit lives in one rule of the spec:

```markdown
# Loan limit

Type: [Rule](../core/rule.md)

## Applies to
[Borrow](borrow.md)

## Rule
A member has at most five open loans.
```

`markitect show lending/loan-limit` shows what relies on it, with the section the link stands in:

```text
lending/loan-limit.md
Title: Loan limit
Type: core/rule.md
Sections:
  Applies to
  Rule
Links to:
  lending/borrow.md
Linked from:
  lending/borrow.md (Refused when)
```

You change five to three. Borrowing names the limit in its answer to a refused call, so it is the document to read again, and the agent that realizes the change starts from both. A link with a typo, `[loan limit](loan-limt.md)`, would not pass `check`:

```text
lending/borrow.md:15: link.unresolved: no document at "loan-limt.md"
1 error in 31 documents
```

## Install

The tool runs anywhere Go runs, on Windows, Linux and macOS.

- **From source,** as in the [quick start](#quick-start): `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main`. `markitect version` names the version and build of the tool on your `PATH`.
- **The Claude Code plugin** in [`plugin/`](plugin/) brings the tool and nothing else: its launcher downloads the binary of the plugin's version on first use and checks it against the release's checksums. The plugin on Main is a development version, which has no release binary: load the repository's `plugin/` folder as a local plugin, and its launcher starts the next `markitect` on the PATH after the plugin's own `bin/`, the build from source ([the plugin](docs/guides/plugin.md#the-launcher)).

## Status

The tool and the default are built on Main and covered by tests and a command smoke. Markitect ships no way of working yet; the [roadmap](docs/roadmap.md#next) names what comes next.

### Limits

- **Set up by hand.** Nothing creates a project or copies the default for you.
- **Structure only.** `check` sees titles, types, sections and links. Whether the spec says what you meant, readers and use show, and whether the project realizes it, your way of working has to check.
- **Your machine, your rights.** Agents run with your permissions. Don't let them read untrusted input, such as issue text or web pages, without an isolated machine with limited rights around them.

## Documentation

| Page | Covers |
| --- | --- |
| [Language reference](docs/guides/language-reference.md) | The spec language and every command |
| [Project reference](docs/guides/project-reference.md) | The project, setting one up by hand, `version` and `diff` |
| [The plugin](docs/guides/plugin.md) | The launcher that fetches the tool, and its start hooks |
| [Library](docs/guides/library.md), [Signals](docs/guides/signals.md) | The default, and the signals of a spec |
| [Concept](docs/concept/README.md) | The one description of the product |
| [Vision](docs/vision.md) | Why Markitect exists and what it leaves to people |
| [Decisions](docs/decisions.md) | What holds today, and the open questions |
| [Roadmap](docs/roadmap.md) | Where Main is and what comes next |

## Contributing

Markitect is developed as an ordinary Go project, with tests and documentation. [CONTRIBUTING.md](CONTRIBUTING.md) says how to make, check, review and land a change. Report defects and propose changes in [GitHub Issues](https://github.com/Glacius-Labs/Markitect/issues); [SUPPORT.md](SUPPORT.md) says where questions go. Report vulnerabilities privately, as [SECURITY.md](SECURITY.md) describes.

## License

Apache-2.0. See [LICENSE](LICENSE) and the [third-party notices](internal/buildinfo/notices.md).
