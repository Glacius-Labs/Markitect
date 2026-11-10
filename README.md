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

**You architect in Markdown.** The spec is the plan of the whole project: one place you own, where what is written is binding. You decide what the project is, how it is divided and which rules hold, and an agent writes it down with you. The spec is typed prose: Definitions of a Kind, in Packages, with links that work like imports. The `markitect` tool checks its structure, never its meaning, as a compiler checks code.

**Agents build to spec.** `markitect status` computes from the spec and the record of what was verified which Definitions a change reaches through their declared links, so the list of work does not depend on an agent's memory. A builder agent makes code, tests, configuration and documentation match the spec for that list. A reviewer agent with fresh context verifies every Definition on it against the spec and records a verdict with its evidence.

Markitect builds on requirements traceability, architecture metrics and the conventions of Go's tools, and it works with Claude Code's own skills, subagents, hooks and worktrees. The [vision](docs/vision.md#why-markitect-exists) gives the full reasoning.

> [!IMPORTANT]
> **Early access.** The latest release runs the whole way of working on Claude Code, with tests. Real runs on real projects are only beginning, and commands, files and skills may still change. Try it on a small project you can throw away, read the [limits](#limits), and tell us in [Issues](https://github.com/Glacius-Labs/Markitect/issues) what worked and what broke.

## Quick start

Install the Claude Code plugin:

```bash
claude plugin marketplace add Glacius-Labs/Claude-Plugins
claude plugin install markitect@glacius-labs
```

For a new project, make an empty folder, run `git init` in it and open Claude Code there.

1. **Set it up.** Ask Claude to set up Markitect. It runs `markitect init`, which writes `.markitect/` with the first page of the spec, `AGENTS.md` and `CLAUDE.md`, and in `.claude/` the skills, the four agents and the hooks that hold their rights. It commits nothing. If Claude Code's sandbox blocks the plugin's first download, let that call run outside it.
2. **Architect the project.** Run `/markitect-architect`. If the session does not offer it yet, run `/reload-skills` or start a new session. It asks what the project is, one question at a time, and writes the spec with you. Read the Markdown diff, or go through it with your session, and commit it together with what `init` wrote.
3. **Build it.** Run `/markitect-build`. Your session starts a builder in a worktree of its own, and a fresh reviewer verifies each Definition. When nothing is left to verify, your session commits the work with its record and merges it into your branch. It pushes nothing.

In a repository that already has code, `/markitect-architect` drafts a first spec from the code with you. The [project workflow](docs/guides/project-workflow.md) covers every step, option and error.

## How it works

1. **You change the spec,** by hand or with `/markitect-architect`, which asks first when something is open.
2. **`markitect status` makes the list.** It lists the Definitions the change reaches: those it edits, those that link to them, and those whose files changed.
3. **A builder agent does the work.** It changes code, tests and documentation for what the list holds and keeps the registry current. A hook keeps it out of the spec and the record.
4. **A fresh reviewer verifies it.** It reads the spec, the list and the code, and treats what the builder tells it as claims to check. It records each item: `ok` with its evidence, `fail`, `skip`, or `spec-problem` with the question for you.
5. **The builder repairs what failed** and has it verified again. It stops when three rounds in a row leave the list no shorter than it has been.
6. **Nothing left.** Your session commits the work with the record and merges it into your branch.

The record, `.markitect/review.sum`, works like Go's test cache: a verdict holds until one of its inputs changes, so a later change puts back on the list only what it reaches. Every file of the project is listed with the Definitions it builds, or explicitly ignored.

Other skills review, clean up, hunt bugs, audit the spec, set up routines and configure the agents. No `markitect` command starts an agent or uses Git. The [concept](docs/concept.md) describes the skills and the four roles, and the [spec language guide](docs/guides/spec-language.md#commands) lists every command.

## Example: one rule changes

A small library allows each member five open loans. The limit lives in one Rule of the spec (excerpt from the [Library example](https://github.com/Glacius-Labs/Markitect/blob/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks/library/m1/model/lending/loan-policy.md)):

```markdown
---
kind: std/Rule
---
# Loan policy

…

| Value | Setting | Meaning |
| --- | --- | --- |
| Loan limit | `5` | The maximum resulting count of currently open loans per member when admitting a new loan. … |

For [[lending/borrow]], the resulting current open-loan count of the member must not exceed the loan limit.
```

You change `5` to `3`. `markitect status` lists the Rule and every Definition that links to it, such as borrowing, returning and the overdue reminders, each with the files that build them. The `catalog` and `members` Packages are not reached. `/markitect-build` has the builder work on that list, and the reviewer then checks each item against the spec.

## Install

Markitect needs Claude Code. Pick one way to get the `markitect` command:

- **The Claude Code plugin,** as in the [quick start](#quick-start). It provides `markitect` inside Claude Code only and downloads the binary of its own version on first use. Update it with `claude plugin update markitect@glacius-labs`.
- **A binary** from the [releases](https://github.com/Glacius-Labs/Markitect/releases). Check it against the checksums file, name it `markitect` and put it on your `PATH`. Then run `markitect install` once, so that a Claude Code session in any folder knows what Markitect is.
- **From source,** with Go 1.27.1 or newer: `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main`, then `markitect install`.

`markitect update` installs the newest release over a downloaded binary and brings a project's skills, agents, hooks and lock in line. It commits nothing. With the plugin, update the plugin, then run `markitect update` in the project; it leaves the plugin's binary alone.

## Status

Markitect is built, released and covered by tests. An earlier form of the build and review flow passed three small real changes, and the parallel workflow passed two real runs on a project of 6 Definitions. A real project, planted defects and reliability on Sonnet over a series of changes come next ([roadmap](docs/roadmap.md#current-steps)).

### Limits

- **Claude Code only.** Another provider comes once Claude's version is reliable.
- **Your machine, your rights.** The agents run in your own Claude Code session with your permissions, so they may stop to ask. Don't let them read untrusted input, such as issue text or web pages, without an isolated machine with limited rights around them.
- **No spending or time limit.** A build runs until nothing is left to verify, or until three rounds in a row leave the list no shorter. You stop it.
- **Verified is not proven.** An empty `markitect status` means a fresh agent judged every Definition against the spec, with evidence. It is complete for the declared links only, and a verdict can be wrong.

## Documentation

| Page | Covers |
| --- | --- |
| [Project workflow](docs/guides/project-workflow.md) | Using Markitect on a project, step by step, with troubleshooting |
| [Spec language](docs/guides/spec-language.md) | Definitions, Kinds, links, the registry, every command and diagnostic |
| [What makes a good spec](docs/guides/good-spec.md) | Designing a spec like code |
| [Concept](docs/concept.md) | The one description of the product |
| [Vision](docs/vision.md) | Why Markitect exists and what it leaves to people |
| [Decisions](docs/decisions.md) | What holds today, and the open questions |
| [Roadmap](docs/roadmap.md) | Where Main is and what comes next |

## Contributing

Markitect is developed as an ordinary Go project, with tests and documentation. [CONTRIBUTING.md](CONTRIBUTING.md) says how to make, check, review and land a change. Report defects and propose changes in [GitHub Issues](https://github.com/Glacius-Labs/Markitect/issues); [SUPPORT.md](SUPPORT.md) says where questions go. Report vulnerabilities privately, as [SECURITY.md](SECURITY.md) describes.

## License

Apache-2.0. See [LICENSE](LICENSE) and the [third-party notices](internal/buildinfo/notices.md).
