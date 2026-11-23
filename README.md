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

**Agents build to spec.** `markitect status` computes from the spec and a state of what was confirmed which documents a change reaches, so the list of work does not depend on an agent's memory. A builder agent makes code, tests, configuration and documentation match the spec, and a reviewer agent with fresh context confirms each document with evidence, so nobody has to read the work to trust it.

**Your team works in one circle around the spec.** What anyone notices lands as a report and becomes a finding; what follows from what was decided is recorded without a question; what needs a decision reaches people in a short briefing, and they decide in conversation. People plan with an agent; agents build, review and merge; QA accepts. That way of working stands in the spec too, and skills in your repository realize it, for Claude Code and Codex alike.

Markitect builds on Terraform's declared state, requirements traceability with suspect links, and quality management, and it works with the coding agents' own skills, subagents and scheduled tasks. The [vision](docs/vision.md#why-markitect-exists) gives the full reasoning.

> [!IMPORTANT]
> **Early access.** The [concept](docs/concept/README.md) describes the direction. The tool and the plugin on Main still implement the previous concept, with reports, settings, control samples, a write table, a landing check and the plugin's jobs, until the [roadmap](docs/roadmap.md#where-main-is) replaces them. No release of them exists yet: the latest release and the plugin in the marketplace are still the earlier 0.34.0. Until the first release, build the tool from source; the plugin on Main is a development version, whose launcher starts that build ([install](#install)). The jobs have not run on a real project yet, and commands, files and skills will change. Try it on a small project you can throw away, read the [limits](#limits), and tell us in [Issues](https://github.com/Glacius-Labs/Markitect/issues) what worked and what broke.

## Quick start

This is the tool and the plugin as they are on Main today. Build the tool from source, with Go 1.27.1 or newer, and put Go's `bin` folder on your `PATH`:

```bash
go install github.com/Glacius-Labs/Markitect/cmd/markitect@main
```

For a new project, make an empty folder, run `git init` in it, and then:

1. **Set it up.** `markitect init` writes `.markitect/` and the spec folder, `docs/spec/`, and nothing else. It commits nothing.
2. **Start the spec.** `markitect apply software` copies the template of a software project with the packages of types it needs; `markitect apply company` does the same for a company. `markitect check` then lists the empty sections: the work still to do.

With the plugin, Claude Code takes it from there:

3. **Architect the project.** `/markitect-architect`, in a folder without `.markitect/`, runs `init` and `apply` itself. It walks the first design with you, area by area, asks what only you can know, and proposes the rest for you to accept.
4. **Build it.** Add a `## Tests` section to `.markitect/settings.md` with the command that runs your tests, since `init` writes none, then run `/markitect-build`. A counter-checker writes checks from the spec alone, a builder realizes what `markitect status` lists, and a fresh reviewer confirms each document, on a branch of the job's own. The branch lands only when `status` shows nothing new against the target and the tests pass.

The [language reference](docs/guides/language-reference.md) lists every command, and [the plugin](docs/guides/plugin.md) every skill.

## How it works

The direction, as the [concept](docs/concept/README.md) describes it; the tool and the plugin on Main still work the previous way until the [roadmap](docs/roadmap.md#where-main-is) replaces them.

1. **You change the spec** with an agent, the Architect. A document is an ordinary Markdown file: a title, a type line that links its type, and the sections the type asks for.
2. **`markitect status` makes the list.** From the spec and the state of what was confirmed, it lists the documents that changed, those suspect because a document they link to, or that links to them, changed, and the files no document explains. It only shows.
3. **Agents do the work.** A builder realizes a planned story on its branch, the spec change and the code together, and a reviewer with fresh context checks it and confirms each document with evidence, `ok`, `fail` or `skip`, which the tool binds to the state it judged.
4. **What anyone notices becomes a finding.** Reports land in `.markitect/context/`; the Triager files findings; what follows from what was decided is recorded without a question, and what needs a decision reaches you in the briefing.
5. **The way of working is part of the spec.** Its `factory` part says who does what, and the skills in the repository realize it.

The state, `.markitect/state.jsonl`, works like Go's test cache: a verdict holds until one of its inputs changes, so a later change puts back on the list only what it reaches. No `markitect` command starts an agent or uses Git.

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

A build has the builder realize the changed rule, and the reviewer judges the suspect documents against it, mostly without a builder. Documents the change does not reach are not listed.

## Install

The current plugin needs Claude Code for its jobs; the tool runs anywhere Go runs, on Windows, Linux and macOS.

- **From source,** as in the [quick start](#quick-start): `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main`. Each project pins the tool's version in `.markitect/version`, and `markitect version` says whether the tool on your `PATH` is that version.
- **The Claude Code plugin** in [`plugin/`](plugin/) brings the skills and agents and the tool itself: its launcher downloads the binary of the plugin's version on first use and checks it against the release's checksums. The plugin on Main is a development version, which has no release binary: load the repository's `plugin/` folder as a local plugin, and its launcher starts the next `markitect` on the PATH after the plugin's own `bin/`, the build from source ([the plugin](docs/guides/plugin.md#the-launcher)).

## Status

The tool and the plugin on Main implement the previous concept and are covered by tests and a command smoke. The roadmap rebuilds them for the circle around the spec, with the skills of a standard way of working, then runs a lab test and a proof on a project that is not ours ([roadmap](docs/roadmap.md#current-steps)).

### Limits

- **Claude Code only, for now.** The current plugin runs on Claude Code; the skills of the standard will hold for Claude Code and Codex alike.
- **Your machine, your rights.** In a session the agents run with your permissions, so they may stop to ask. Don't let them read untrusted input, such as issue text or web pages, without an isolated machine with limited rights around them.
- **No spending or time limit.** A build runs until nothing is left on the list, or until three rounds in a row leave it no shorter. You stop it.
- **Confirmed is not proven.** An empty `markitect status` means every document was confirmed at its current state, with evidence. It reaches one step along declared links, and a verdict can be wrong; control samples measure how often.

## Documentation

| Page | Covers |
| --- | --- |
| [Language reference](docs/guides/language-reference.md) | The spec language and every command |
| [Project reference](docs/guides/project-reference.md) | The project, the state, `status`, `confirm`, `move`, the history and `compare` |
| [The plugin](docs/guides/plugin.md) | The launcher that fetches the tool, and its start hooks |
| [Library](docs/guides/library.md), [Signals](docs/guides/signals.md) | The packages, templates and skills, and the signals of a spec |
| [Concept](docs/concept/README.md) | The one description of the product |
| [Vision](docs/vision.md) | Why Markitect exists and what it leaves to people |
| [Decisions](docs/decisions.md) | What holds today, and the open questions |
| [Roadmap](docs/roadmap.md) | Where Main is and what comes next |

## Contributing

Markitect is developed as an ordinary Go project, with tests and documentation. [CONTRIBUTING.md](CONTRIBUTING.md) says how to make, check, review and land a change. Report defects and propose changes in [GitHub Issues](https://github.com/Glacius-Labs/Markitect/issues); [SUPPORT.md](SUPPORT.md) says where questions go. Report vulnerabilities privately, as [SECURITY.md](SECURITY.md) describes.

## License

Apache-2.0. See [LICENSE](LICENSE) and the [third-party notices](internal/buildinfo/notices.md).
