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

**Agents build to spec.** `markitect status` computes from the spec and a state of what was confirmed which documents a change reaches, so the list of work does not depend on an agent's memory. A builder agent makes code, tests, configuration and documentation match the spec for that list, and a reviewer agent with fresh context confirms each document with evidence. Checks derived from the spec alone and planted control samples measure how well that checking works, so nobody has to read the work to trust it.

Markitect builds on Terraform's declared state, requirements traceability with suspect links, and quality management, and it works with Claude Code's own plugins, skills, subagents and scheduled tasks. The [vision](docs/vision.md#why-markitect-exists) gives the full reasoning.

> [!IMPORTANT]
> **Early access.** The tool and the plugin on Main are rebuilt to the current [concept](docs/concept/README.md), with tests, and no release of them exists yet: the latest release and the plugin in the marketplace are still the earlier 0.34.0. Until the first release, build the tool from source; the plugin on Main is a development version, whose launcher starts that build ([install](#install)). The jobs have not run on a real project yet, and commands, files and skills may still change. Try it on a small project you can throw away, read the [limits](#limits), and tell us in [Issues](https://github.com/Glacius-Labs/Markitect/issues) what worked and what broke.

## Quick start

Build the tool from source, with Go 1.27.1 or newer, and put Go's `bin` folder on your `PATH`:

```bash
go install github.com/Glacius-Labs/Markitect/cmd/markitect@main
```

For a new project, make an empty folder, run `git init` in it, and then:

1. **Set it up.** `markitect init` writes `.markitect/` and the spec folder, `docs/spec/`, and nothing else. It commits nothing.
2. **Start the spec.** `markitect apply software` copies the template of a software project with the packages of types it needs; `markitect apply company` does the same for a company. `markitect check` then lists the empty sections: the work still to do.

With the plugin, Claude Code takes it from there:

3. **Architect the project.** `/markitect-architect`, in a folder without `.markitect/`, runs `init` and `apply` itself. It walks the first design with you, area by area, asks what only you can know, and proposes the rest for you to accept.
4. **Build it.** Name the command that runs your tests in the Tests section of `.markitect/settings.md`, then run `/markitect-build`. A counter-checker writes checks from the spec alone, a builder realizes what `markitect status` lists, and a fresh reviewer confirms each document, on a branch of the job's own. The branch lands only when `status` shows nothing new against the target and the tests pass.

The [language reference](docs/guides/language-reference.md) lists every command, and [the plugin](docs/guides/plugin.md) every skill.

## How it works

1. **You change the spec,** by hand or with `/markitect-architect`. A document is an ordinary Markdown file: a title, a type line that links its type, and the sections the type asks for.
2. **`markitect status` makes the list.** From the spec and the state of what was confirmed, it lists the documents that changed, those suspect because a document they link to, or that links to them, changed, and the files no document explains.
3. **A job does the work.** The build job realizes the changed documents in code, tests, configuration and documentation, and a reviewer with fresh context confirms each one with evidence, `ok`, `fail` or `skip`, which `markitect mark` binds to the state it judged. It stops when three rounds in a row leave the list no shorter.
4. **What needs a decision becomes a report.** A gap or a contradiction in the spec is a report in `.markitect/reports/`; triage answers it or escalates it to you, and the document waits on it in `status`.
5. **The checking is measured.** Control samples plant deviations and check whether a blind review finds them, and `markitect figures` computes how the factory runs from the reports, the samples and the history.

The state, `.markitect/state.jsonl`, works like Go's test cache: a verdict holds until one of its inputs changes, so a later change puts back on the list only what it reaches. No `markitect` command starts an agent or uses Git. The [concept](docs/concept/README.md) describes the jobs and the roles.

## Example: one rule changes

A small library allows each member five open loans. The limit lives in one rule of the spec:

```markdown
# Loan limit

Type: [Rule](../packages/core/rule.md)

## Applies to
[Borrow](borrow.md)

## Rule
A member has at most five open loans.
```

You change five to three. `markitect status` lists the rule as changed and every document that links to it, or that it links to, as suspect, such as borrowing:

```text
changed: lending/loan-limit.md@327da543: self
  look at: docs/spec/lending/loan-limit.md, src/loans.go, src/loans.go:3
suspect: lending/borrow.md@f0861ccb: link:lending/loan-limit.md
  look at: docs/spec/lending/borrow.md, src/loans.go, src/loans.go:3
1 changed, 1 suspect, 0 unexplained
```

`/markitect-build` has the builder realize the changed rule, and the reviewer judges the suspect documents against it, mostly without a builder. Documents the change does not reach are not listed.

## Install

Markitect needs Claude Code for its jobs; the tool runs anywhere Go runs, on Windows, Linux and macOS.

- **From source,** as in the [quick start](#quick-start): `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main`. Each project pins the tool's version in `.markitect/version`, and `markitect version` says whether the tool on your `PATH` is that version.
- **The Claude Code plugin** in [`plugin/`](plugin/) brings the skills and agents and the tool itself: its launcher downloads the binary of the plugin's version on first use and checks it against the release's checksums. The plugin on Main is a development version, which has no release binary: load the repository's `plugin/` folder as a local plugin, and its launcher starts the next `markitect` on the PATH after the plugin's own `bin/`, the build from source ([the plugin](docs/guides/plugin.md#the-launcher)).

## Status

The rebuilt tool and plugin are covered by tests and a command smoke. Their jobs come next on a committed test project, then a lab test and a proof on a project that is not ours ([roadmap](docs/roadmap.md#current-steps)).

### Limits

- **Claude Code only.** Another provider comes once Claude's version is reliable.
- **Your machine, your rights.** In a session the agents run with your permissions, so they may stop to ask. Don't let them read untrusted input, such as issue text or web pages, without an isolated machine with limited rights around them.
- **No spending or time limit.** A build runs until nothing is left on the list, or until three rounds in a row leave it no shorter. You stop it.
- **Confirmed is not proven.** An empty `markitect status` means every document was confirmed at its current state, with evidence. It reaches one step along declared links, and a verdict can be wrong; control samples measure how often.

## Documentation

| Page | Covers |
| --- | --- |
| [Language reference](docs/guides/language-reference.md) | The spec language and every command |
| [Project reference](docs/guides/project-reference.md) | The project, the state, `status`, `mark`, `move`, the history and `compare` |
| [The plugin](docs/guides/plugin.md) | The skills, the agents, the launcher and how firm its checks are |
| [Library](docs/guides/library.md), [Signals](docs/guides/signals.md), [Reports](docs/guides/reports.md), [Rights](docs/guides/rights.md) | The packages and templates, the signals of a spec, the reports, settings and control samples, and the landing rule |
| [Concept](docs/concept/README.md) | The one description of the product |
| [Vision](docs/vision.md) | Why Markitect exists and what it leaves to people |
| [Decisions](docs/decisions.md) | What holds today, and the open questions |
| [Roadmap](docs/roadmap.md) | Where Main is and what comes next |

## Contributing

Markitect is developed as an ordinary Go project, with tests and documentation. [CONTRIBUTING.md](CONTRIBUTING.md) says how to make, check, review and land a change. Report defects and propose changes in [GitHub Issues](https://github.com/Glacius-Labs/Markitect/issues); [SUPPORT.md](SUPPORT.md) says where questions go. Report vulnerabilities privately, as [SECURITY.md](SECURITY.md) describes.

## License

Apache-2.0. See [LICENSE](LICENSE) and the [third-party notices](internal/buildinfo/notices.md).
