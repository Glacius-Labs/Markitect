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
> **Early access.** The tool and the skills of the standard are built on Main, and the [concept](docs/concept/README.md) describes them. No release of them exists yet: the latest release and the plugin in the marketplace are still the earlier 0.34.0, of an earlier concept. Until the first release, build the tool from source; the plugin on Main is a development version, whose launcher starts that build ([install](#install)). No skill has run on a real project yet, and commands, files and skills will change. Try it on a small project you can throw away, read the [limits](#limits), and tell us in [Issues](https://github.com/Glacius-Labs/Markitect/issues) what worked and what broke.

## Quick start

Build the tool from source, with Go 1.27.1 or newer, and put Go's `bin` folder on your `PATH`:

```bash
go install github.com/Glacius-Labs/Markitect/cmd/markitect@main
```

For a new project, make an empty folder, run `git init` in it, and then:

1. **Set it up.** `markitect init` writes `.markitect/` with the folders of the work, the spec folder `docs/spec/` with the four templates `project`, `product`, `architecture` and `factory`, the skills of the standard in `.claude/skills/` and `.agents/skills/`, and an entry in `AGENTS.md`. It commits nothing.
2. **See what is left to fill.** `markitect check` lists the sections the templates leave empty: the work still to do. A further template goes into a folder of its own, such as `markitect apply company company`.
3. **Start with Norman.** In a new session of Claude Code or Codex in the folder, start the skill `norman`. It tells you what is critical and what waits for you, and starts the session that fits your role: for a new project the Architect, who works out the spec as a whole with you and leads you through planning, and for an existing one extracts its spec.

The plugin in [`plugin/`](plugin/) only fetches the tool for Claude Code ([the plugin](docs/guides/plugin.md)). The [language reference](docs/guides/language-reference.md) lists every command, and the [library](docs/guides/library.md#the-skills-of-the-standard) the skills.

## How it works

The [concept](docs/concept/README.md) describes the whole; this is the short form.

1. **You change the spec** with an agent, the Architect. A document is an ordinary Markdown file: a title, a type line that links its type, and the sections the type asks for.
2. **`markitect status` makes the list.** From the spec and the state of what was confirmed, it lists the documents that changed, those suspect because a document they link to, or that links to them, changed, and the files no document explains. It only shows.
3. **Agents do the work.** A builder realizes a planned story on its branch, the spec change and the code together, and a reviewer with fresh context checks it and confirms each document with evidence, `ok`, `fail` or `skip`, which `markitect confirm` binds to the state it judged.
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
changed: lending/loan-limit.md@3f85f260: self
  look at: docs/spec/lending/loan-limit.md, src/loans.go, tests/loans_test.go, tests/loans_test.go:5
suspect: lending/borrow.md@fde1ae8e: link:lending/loan-limit.md
  look at: docs/spec/lending/borrow.md, src/loans.go, tests/loans_test.go, tests/loans_test.go:5
1 changed, 1 suspect, 0 unexplained
```

A build has the builder realize the changed rule, and the reviewer judges the suspect documents against it, mostly without a builder. Documents the change does not reach are not listed.

## Install

The tool runs anywhere Go runs, on Windows, Linux and macOS; the skills are written for Claude Code and Codex alike.

- **From source,** as in the [quick start](#quick-start): `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main`. Each project pins the tool's version in `.markitect/version`, and `markitect version` says whether the tool on your `PATH` is that version.
- **The Claude Code plugin** in [`plugin/`](plugin/) brings the tool and nothing else: its launcher downloads the binary of the plugin's version on first use and checks it against the release's checksums. The plugin on Main is a development version, which has no release binary: load the repository's `plugin/` folder as a local plugin, and its launcher starts the next `markitect` on the PATH after the plugin's own `bin/`, the build from source ([the plugin](docs/guides/plugin.md#the-launcher)).

## Status

The tool and the skills of the standard are built on Main and covered by tests and a command smoke. No skill has run on a project yet: the roadmap runs a finding around the circle on a test project, extracts the spec of an existing project, and then runs a lab test and a proof on a project that is not ours ([roadmap](docs/roadmap.md#current-steps)).

### Limits

- **Not run yet.** The skills of the standard have not run on a real project, for Claude Code or for Codex.
- **Your machine, your rights.** In a session the agents run with your permissions, so they may stop to ask. Don't let them read untrusted input, such as issue text or web pages, without an isolated machine with limited rights around them.
- **No spending or time limit.** A Builder works until its story lands or is aborted, with at most five review rounds. Nothing else limits the time or the quota it uses. You stop it.
- **Confirmed is not proven.** An empty `markitect status` means every document was confirmed at its current state, with evidence. It reaches one step along declared links, and a verdict can be wrong; control samples, which come later, will measure how often.

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
