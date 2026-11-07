<h1 align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/markitect-lockup.svg">
    <img src="docs/assets/markitect-lockup-print.svg" alt="Markitect" width="360">
  </picture>
</h1>

<p align="center">
  Markitect keeps one spec of your project in Markdown, checks it like code,<br>
  and has an agent build every change while a fresh agent checks it.
</p>

<p align="center">
  <a href="https://github.com/Glacius-Labs/Markitect/actions/workflows/ci.yaml"><img src="https://github.com/Glacius-Labs/Markitect/actions/workflows/ci.yaml/badge.svg" alt="CI"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/Glacius-Labs/Markitect" alt="Go version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License: Apache-2.0"></a>
  <a href="#status"><img src="https://img.shields.io/badge/status-active%20development-orange" alt="Status: active development"></a>
</p>

<p align="center">
  <a href="#why-markitect">Why</a> ·
  <a href="#how-it-works">How it works</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#example-one-rule-changes">Example</a> ·
  <a href="#status">Status</a> ·
  <a href="#documentation">Documentation</a>
</p>

> [!IMPORTANT]
> Main carries the build of [concept 4.0](docs/concepts/v4-concept.md), Markitect's target. Markitect is in active development, and its releases are builds of that concept. Every command on this page exists on `main`.

## Why Markitect

When AI agents build most of a project, its design, rules, documentation, code, tests and configuration drift apart. A decision about behavior has to reach the code, the tests, the documentation and the agents' own instructions. An agent finishes its task and misses a rule two folders away. Consistency passes take hours and find more each time.

Markitect moves the knowledge of what belongs together out of the agents' memory and into one place you own: the spec. You describe the project once, as typed documentation in Markdown. Code, tests, configuration and documentation are built from it. When you want something different, you change the spec; Markitect says which Definitions the change puts back on the list to verify, an agent brings the project in line, and a fresh agent verifies each one and records its verdict, until nothing is left.

It is a way of agentic coding, not a platform: the agents are ordinary Claude Code agents in your own session and your own folder, and the tool keeps a few clear guarantees around them.

## Features

- **Typed documentation.** One Markdown file per Definition, its Kind in the front matter, links by path such as `[[members/member]]`. Projects define their own Kinds; the standard Libraries `std`, `rest` and `cli` ship in the binary.
- **A compiler for intent.** `markitect check` resolves every Kind and link, keeps internal Definitions inside their subtree and refuses dependency cycles between Packages, as Go does. If the spec does not compile, `markitect status` gives no list until it does.
- **Every file accounted for.** One registry page per Package names the paths it owns and lists each file with the Definitions it builds. A file that is neither registered nor ignored is an error.
- **A cache of what was verified.** Like `go test`, which runs a test again only when something it depends on changed, Markitect records for each Definition the state at which a reviewer verified it: its text, its Kind and links, its files and the evidence. `markitect status` lists every Definition whose state changed, with what changed and where to look. `markitect vet` measures the spec like code: coupling, size, how far a change reaches.
- **A builder and a fresh reviewer.** `/markitect-build` has your Claude Code session start a builder agent, which starts a reviewer agent with fresh context. The reviewer verifies each item and records `ok`, `fail`, `skip` or `spec-problem` with `markitect mark`, and a `fail` stays on the list until the builder has repaired it.
- **Verdicts bound to what was read.** A verdict names the item as `status` printed it, such as `members/limit@3f9a1c`, and `mark` refuses it when anything changed since. A hook lets only the reviewer record, and keeps every agent out of the spec and the record.
- **The record, committed.** `.markitect/review.sum` holds one line per Definition, and your session commits it with the work on your branch. Nothing is pushed. Markitect itself uses no Git and steers no work.

## How it works

```mermaid
flowchart LR
    Spec["You change<br/>the spec"] --> Status["status<br/>what is<br/>to verify"]
    Status --> Build["builder<br/>makes the project<br/>match"]
    Build --> Review["reviewer<br/>verifies and records<br/>every item"]
    Review -->|"fail stays<br/>on the list"| Status
    Review --> Land["record committed<br/>with the work on<br/>your branch"]
```

1. **You change the spec,** by hand or with your coding agent through the modeling skill, which asks first when something is open. You read the Markdown diff.
2. **You run `/markitect-build` in Claude Code.** Your session starts a builder agent.
3. **`markitect status` makes the list.** It lists the Definitions the change reaches: those it edits, those that link to them, and those whose files changed, each with where to look. A spec that does not compile gives no list.
4. **A builder agent does the work,** in your folder. It changes the code, tests and documentation for what the list holds and keeps the registry current.
5. **A fresh reviewer verifies it.** The builder starts the reviewer, which sees only the spec, the list and the code. It records each item with `ok` and its evidence, `fail` with why, `skip` with why, or `spec-problem` with the question for you. The builder repairs what failed and has it verified again.
6. **Nothing left.** When `status` lists nothing, or only what needs you, your session commits the work with the record on your branch.

`/markitect-review` runs a reviewer alone over what `status` lists, or over everything with `--all`, and changes no code. `/markitect-build` then repairs what failed.

## Quick start

### Install

Markitect needs Claude Code; the tool itself uses no Git, and your session commits with whatever you use. Pick one way to get the `markitect` command.

**With the Claude Code plugin.** One step, and nothing else to install:

```bash
claude plugin marketplace add Glacius-Labs/Claude-Plugins
claude plugin install markitect@glacius-labs
```

The plugin puts `markitect` on the `PATH` of Claude Code's shell and tells every session what Markitect is. It downloads the binary of its own version the first time, checking it against the release's checksums, and a plugin update brings the matching binary: `claude plugin update markitect@glacius-labs`. That `markitect` exists inside Claude Code only. For your own terminal, use one of the next two.

The marketplace is the small repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins). It names the release of the plugin that users get, so a plugin changes for you only when a release is published there.

**With a binary.** Download the file for your system from the [releases](https://github.com/Glacius-Labs/Markitect/releases), check it against the checksums file, name it `markitect` and put it on your `PATH`. Then run `markitect install` once: it writes a personal skill, so that a Claude Code session in any folder knows what Markitect is and how to begin. (The plugin brings that skill itself.)

**From source.** With Go 1.27.1 or newer: `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main`, then `markitect install`.

The skills and agents call `markitect` by name, so the folder of the binary must be on your `PATH`; `version`, `init`, `install` and `update` warn when it is not. `markitect version` shows which build you have and whether the personal skill and the current project match it.

To upgrade, run `markitect update`. It installs the newest release over the binary, then lets the new binary bring what Markitect wrote earlier in line: the personal skill and, in a project, the skills, agents, hooks and lock that `init` wrote, and the record of a project from 0.20.0, written from its newest certificate. It commits nothing; read the change of the lock before you commit it. It changes no spec file, and `markitect update --check` only reports. A binary the plugin brought is updated with the plugin instead, and a build from source is replaced only with `markitect update --force`.

### Set up a project

```bash
cd your-project
markitect init --name "Your project"
markitect check
```

`init` creates `.markitect/` with the manifest, the lock, a project anchor and Root's registry page, writes `AGENTS.md` and `CLAUDE.md`, and installs the skills, the builder and reviewer agents, and hooks in `.claude/settings.json` that let only the reviewer record verdicts and keep agents out of `.markitect/` except the registry pages. It works in an empty folder too, makes no repository and commits nothing. It never changes an existing spec file, so it is safe to run again after an upgrade. In a new project, `check` says that nothing is built yet. Open Claude Code and run `/markitect-start`, which asks what the project is and starts the spec with you.

In a repository that already has files, the first `check` fails by design, because every file must belong somewhere:

```text
README.md: file.unregistered: neither registered nor ignored: list it on .markitect/registry.md, the page of its owner Root, or add it to the ignore file
1 error: 0 in the spec, for the owner to settle; 1 in the realization, so spec and code do not match yet, which /markitect-build settles
```

List the file on Root's registry page, `.markitect/registry.md`, with the Definitions it builds:

```markdown
| File | Realizes | Why |
|---|---|---|
| `README.md` | [[project]] | What the project is for |
```

### Write the spec

Ask your coding agent to use the `markitect-modeling` skill and describe the project with you. For an existing code base, the `markitect-extract` skill drafts a first spec from the code. Then look around:

```bash
markitect vet                    # coupling, size, how far a change reaches
markitect list --deps            # the Packages, their Definitions and what each depends on
markitect doc members/member     # one Definition: text, links both ways, files, record
markitect status                 # what is to verify, and what changed
```

None of these commands starts an agent.

### Let an agent build it

Commit what is not part of the spec, then run this in Claude Code, in the project's folder:

```text
/markitect-build
```

Your session starts the builder and waits for it; leave the folder alone meanwhile. The first time it builds the whole spec, since nothing is verified yet; later it builds only what `status` lists. `/markitect-review` verifies what `status` lists without changing code. The [project workflow](docs/guides/project-workflow.md) covers every option, limit and exit code.

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
| Loan period | `14` days | The number of calendar days added to the borrowing date to determine a new loan's due date. … |

For [[lending/borrow]], the resulting current open-loan count of the member must not exceed the loan limit.
Count every open [[lending/loan]], including overdue loans, and exclude closed loans. …
```

You change `5` to `3`. `markitect status` lists the Rule itself and every Definition that links to it, such as borrowing, returning and the overdue reminders, each with the files that build them. The `catalog` and `members` Packages are not reached. `/markitect-build` has the builder work on that list; a conforming result refuses a fourth open loan, updates the tests and the documentation, keeps every existing loan and every return working, and leaves the files that already conform untouched. The reviewer then checks each item against the spec.

## Concepts

| Term | Meaning |
| --- | --- |
| **Spec** | The Markdown files under `.markitect/model/`: the project's documentation and its authority. Where code and spec differ, the spec is right. |
| **Definition** | One Markdown file of the spec, with its Kind in the front matter. Its path is its ID, its heading its title, its body the documentation. |
| **Kind** | The type of a Definition, itself a Definition of Kind `Kind`. A project writes its own Kinds, one small file each. |
| **Library** | A set of Kinds for one area: `std` (Project, Rule, Concept, Feature, Decision), `rest` (Endpoint) and `cli` (Command). |
| **Package** | A folder of the spec. Packages nest, and everything a Package defines reaches its whole subtree. The spec's root folder is the Root Package. |
| **Registry** | One page per Package: the path patterns it owns and every file with the Definitions it builds. |
| **Work** | One `/markitect-build` or `/markitect-review`: a builder and a fresh reviewer, or a reviewer alone, working in your folder. |
| **Record** | `.markitect/review.sum`: for each Definition, the verdict, its evidence and the hash of every input at which it was verified. `markitect status` compares it with the folder. |
| **Verdict** | What the reviewer records: `ok`, `fail`, `skip` or `spec-problem`, with evidence or why. |

A project keeps all of it under `.markitect/`:

```text
.markitect/
├── manifest.yaml     # the Libraries the project uses, with their versions
├── lock.yaml         # the digest of each Library
├── model/            # Packages and Definitions; this folder is Root
├── registry.md       # Root's registry page
├── registry/         # one registry page per Package
├── ignore            # files that deliberately belong to no Package, if any
├── libraries/        # Libraries that do not ship with markitect
├── review.sum        # the record of what was verified
└── .gitattributes    # merges the record by keeping both sides
```

The [model language guide](docs/guides/model-language.md) gives the exact format and every diagnostic.

## Status

Main carries [concept 4.0](docs/concepts/v4-concept.md), the target the owner accepted on 7 November 2026. The [delivery plan](docs/implementation-plan.md) holds the current step.

| Area | State |
| --- | --- |
| The spec language and the commands `init`, `check`, `tidy`, `vet`, `list` and `doc` | Available (`tidy`, `vet`, `list` and `doc` are on Main for the next release; 0.21.0 has `lock`, `lint`, `graph` and `refs`) |
| Work with `/markitect-build` and `/markitect-review`, over `status`, `mark` and the record | Available on Claude Code since 0.21.0, covered by tests. The flow before it, with `gate`, was tried on three changes of one project ([trials](docs/validation/lean-trials-20261107.md)) |
| Codex | Not yet: it needs subagents with their own agent files and a hook that names the agent |
| Release | On GitHub, with the Claude Code plugin ([install](#install)) |

## FAQ

<details>
<summary><b>Does the compiler prove that the code is correct?</b></summary>

It checks structure: Kinds, links, visibility, cycles and file accounting. Agents judge meaning. Every Definition a change reaches needs a verdict from a reviewer with fresh context, with the evidence it read. The bar is today's practice, one strong agent with ordinary documentation.

</details>

<details>
<summary><b>How much do I have to write?</b></summary>

As much as good documentation of the project would state: domain concepts, features, interfaces such as REST endpoints and CLI commands, rules, decisions and the architecture. Loops, data structures and algorithms stay with the agents. One file may build several Definitions, and every file is registered or explicitly ignored.

</details>

<details>
<summary><b>Can agents change my spec?</b></summary>

The agents of the work don't: a hook refuses their edits under `.markitect/`, except the registry pages the builder keeps. Agents propose spec changes in conversation through the modeling skill, and you accept each one. A real problem in the spec is recorded as `spec-problem`, with the question for you, and stays on the list until you settle it.

</details>

<details>
<summary><b>Can I adopt it on an existing code base?</b></summary>

Yes. The `markitect-extract` skill drafts a spec from the code with you, and you accept it like any other change. From then on, `/markitect-review --all` verifies every Definition and records where the code deviates, and `/markitect-build` repairs it.

</details>

<details>
<summary><b>Which agents and models does it use?</b></summary>

Claude Code agents, started by your own session. The builder and the reviewer use Claude Sonnet unless the manifest names another `model`. They work under your session's permission mode; the builder edits files without asking.

</details>

<details>
<summary><b>Where does the name come from?</b></summary>

Markitect = Markdown + Architect. The idea grew out of a project documented in Markdown rules, skills and agent instructions that drifted away from its code. Markitect governs the whole project, and agent instructions are one of its realizations.

</details>

## Documentation

| Guide | Covers |
| --- | --- |
| [Project workflow](docs/guides/project-workflow.md) | Using Markitect on a project, command by command, with troubleshooting |
| [Model language](docs/guides/model-language.md) | Definitions, Kinds, Libraries, links, the registry and every diagnostic |
| [Concept 4.0](docs/concepts/v4-concept.md) | The target, which Main builds: work in your own session, and why |
| [Vision](docs/vision.md) | Why Markitect exists, what it promises and what it leaves to people |
| [Architecture](docs/architecture.md) | Packages, layers and the product flow |
| [Delivery plan](docs/implementation-plan.md) | What is built and what comes next |
| [Documentation map](docs/README.md) | Everything else |

## Contributing

Markitect is developed as an ordinary Go project, with tests and documentation. [CONTRIBUTING.md](CONTRIBUTING.md) has the checks, the import boundaries and the review rules. Report defects and propose changes in [GitHub Issues](https://github.com/Glacius-Labs/Markitect/issues); [SUPPORT.md](SUPPORT.md) says where questions go. Report vulnerabilities privately, as [SECURITY.md](SECURITY.md) describes.

## License

Apache-2.0. See [LICENSE](LICENSE) and the [third-party notices](internal/buildinfo/notices.md).
