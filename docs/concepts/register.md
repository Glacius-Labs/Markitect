# Concept register: decisions in force

This page lists what holds for Markitect today, grouped by topic. Each entry says in plain words what holds, since when, and which earlier entries it consolidates. [Concept 4.0](v4-concept.md) describes the product these decisions add up to, and the [vision](../vision.md) says why. The [record rules](README.md) say how entries are added and changed.

**History.** Until 8 November 2026 this register held 159 entries, 103 of them decisions, most of them superseded by later concepts. That full register, with every entry's original wording, status and sources, is archived at [`d85e0fa1`](https://github.com/Glacius-Labs/Markitect/blob/d85e0fa1302ec95d1c82c6d2c088bf2ff34b6557/docs/concepts/register.md) ([archive index](../archive.md#consolidated-on-8-november-2026)). An ID that is not on this page is history: it was superseded, or it recorded a step on the way. An entry below keeps the ID of the decision it mainly restates, and an old link to any ID it consolidates still lands on it.

"Owner" means Markitect's product owner. "Since" gives the date of the decision and the release that first carried it, where there is one.

## Purpose and thesis

### DEC-101 Markitect is a programming language for documentation

**Since** 7 and 8 November 2026, the owner's statement of how to read the product.

- The spec is typed prose in Packages, with links that work like imports. A deterministic tool checks and analyses it like code, and agents realize and verify it.
- The owner expects coding to move from languages such as Go, C# or Python towards documentation and meaning. What matters in code stays as relevant there: architecture, consistency, cleanliness, an overview without reading the code, and knowing what should be as against what is.
- Code has always been a precise description for a limited but diligent executor. An AI executor is intelligent, but it still has to be told what is meant and what holds, or it guesses. So documentation becomes the programming language, and Markitect makes its authors define what they mean and what holds and connect those statements.
- Contradictions in a description surface only when someone changes the place where they sit, as with weakly typed programming languages. So Markitect borrows typing and a compiler as helpers.
- The aim is to meet the preconditions that open the established knowledge of software architecture to this language: Definitions with a type, declared dependencies, and a structure a tool can check and measure.
- It builds on proven practice: requirements traceability with suspect links (the registry, `status` and `mark`), architecture metrics (`vet`) and Go's conventions (the command line). What is new: the traceability is plain, typed text checked deterministically, agents do the realization and the evidence, and the record of what was verified works like Go's test cache.
- AI makes affordable what ordinary projects dropped for its manual cost, such as keeping a traceability matrix.
- Three problems: nobody keeps track of what was decided, so the spec is the one living statement of what holds; nobody reads the volume of documentation AI produces, so the spec is small, structured and navigable; nobody knows whether realization and idea match, so `status` and the record say per Definition what a fresh reviewer found, with its evidence.
- The honest limit: "complete" means complete for the declared links.

<a id="dec-001-delegated-realization-is-central-to-the-product"></a>

### DEC-001 Delegated realization is the purpose

**Since** 10 October 2026; restated on 6 November 2026 for the lean tool. **Consolidates** DEC-001 and the part of DEC-096 that keeps it.

- People maintain the spec. Agents realize it in code, tests, configuration and documentation, and verify the result. The tool computes and checks, so that the delegated work can be trusted.
- Judge Markitect as a way of developing software, not as a linter or a context tool. Removing delegated realization as scope creep contradicts this decision.

### DEC-002 Trust is measured against today's practice

**Since** 10 October 2026; reaffirmed on 22 October 2026.

- The baseline is common practice: an agent implements a task, another AI reviews the pull request, and the human approval is often given almost blind. Markitect aims to earn more trust than that through structure, not compiler-grade guarantees.
- Claims read "more reliable than the baseline", never "correct". Use the smallest process that earns the trust: more report fields, mandatory probes or completeness declarations do not by themselves make work more correct.

<a id="dec-003-the-compiler-is-a-borrowed-concept"></a><a id="cpt-003-two-parts-of-the-promise"></a>

### DEC-003 The compiler is a borrowed idea, and two claims stay apart

**Since** 10 October 2026. **Consolidates** DEC-003 and CPT-003.

- The spec's structure compiles deterministically: Kinds, links, Packages and file accounting. Meaning is judged by agents. Markitect claims no semantic guarantee for what agents build.
- Two claims stay apart. *Nothing in the spec's scope is overlooked* can be computed from declared links and file accounting; undeclared dependencies in code are a known gap ([ENH-001](#enh-001-observed-code-dependencies-widen-impact)). *Everything is realized correctly* can only be supported by evidence: a fresh reviewer, the project's tests and `check`.

<a id="dec-009-the-central-evaluation-question"></a><a id="dec-008-the-delegated-method-has-not-yet-been-tested"></a><a id="dec-010-evaluate-the-method-separately-from-its-execution-runtime"></a><a id="dec-072-stable-v1-justified-by-data"></a>

### DEC-009 How Markitect is judged

**Since** 10 October 2026; the aim set on 4 November 2026. **Consolidates** DEC-008, DEC-009 and DEC-010, and **supersedes** DEC-072, which asked data to show V1 better than one agent.

- The aim is one clean and simple version that works reliably on Sonnet, with no babysitting. It is no longer to prove that one version beats another ([concept 4.0, §1](v4-concept.md#1-the-aim)).
- The long-term question stays the owner's: when people develop against a spec, can it be realized reliably and can that be checked; does the repository stay consistent and clean; does agentic coding become more trustworthy; are rules followed instead of overlooked; can people move their attention to deciding and modeling?
- The method is judged apart from any runtime: a failure of Markitect's own tool is a product finding, a harness or environment failure a limit of the measurement. The tool is separate from the driver that starts the agents ([DEC-096](#dec-096-the-tool-computes-and-checks-a-driver-acts)).
- The method has not yet been tested over a long series of changes. What is shown, and what is not, is in [concept 4.0, §13](v4-concept.md#13-what-is-not-yet-shown).

<a id="dec-014-compatibility-does-not-drive-decisions"></a><a id="dec-016-a-clean-stable-testable-main-and-uniform-structure-first"></a>

### DEC-032 Quality, simplicity and stability over effort and compatibility

**Since** 16 October 2026. **Consolidates** DEC-014, DEC-016 and DEC-032.

- Clean architecture, simplicity and stability come first. Effort is no criterion, and neither is compatibility with old versions: nobody else uses Markitect yet. Released files stay immutable; that is a fact, not a promise of support.
- Choose the clean design even when it is more work. Add no aliases, adapters or legacy paths. Delete what is obsolete, source, tests and documents together.
- Main stays clean, stable and testable: tests check Markitect itself, not provider or environment quirks, and the repository has a uniform structure, keeping a convention only where it is sensible.
- Simpler is more robust, and simplicity is earned through insight. Markitect works with Claude Code and Anthropic's own features rather than building machinery around them.

<a id="dec-098-concept-40-is-the-target"></a><a id="dec-068-v1-concept-10"></a><a id="dec-086-concept-20-is-the-target"></a><a id="dec-091-concept-30-is-the-target"></a>

### DEC-098 Concept 4.0 is the one concept

**Since** 7 November 2026, revised with each decision since. **Supersedes** DEC-068, DEC-086 and DEC-091, which accepted the earlier concepts.

- [Concept 4.0](v4-concept.md) is the single description of the product and the target that Main builds: the spec, a tool that checks it and records what was verified, and a thin default harness in which the user's own Claude Code session starts the agents.
- V1 concept 1.0, concept 2.0 and concept 3.0 are records in the [archive](../archive.md#consolidated-on-8-november-2026).

## The spec language

<a id="dec-080-the-model-is-typed-documentation"></a>

### DEC-080 The spec is typed documentation in Markdown

**Since** 28 October 2026; amended on 7 November 2026 (unused Definitions are errors) and 8 November 2026 (the folder `.markitect/spec/`). **Consolidates** DEC-080 and the language parts of DEC-098.

- A Definition is one Markdown file with YAML front matter that names its Kind. Its path is its ID, its main heading its title, and its body the documentation. There are no typed fields: exact values, such as wire names and error codes, are written in prose.
- A Kind is itself a Definition, of Kind `Kind`. No Kind has a special meaning to Markitect. Projects define their own Kinds, and Libraries supply common ones: `std`, `rest` and `cli` ship in the binary; any other Library is committed under `.markitect/libraries/`. The manifest names each Library with its version, and the lock pins its digest; `tidy` writes it.
- The spec lives in `.markitect/spec/`, renamed from `.markitect/model/` on 8 November 2026. A Package is a folder. Everything a Package defines reaches its whole subtree, and the spec's root folder is the Root Package.
- A link, `[[members/member]]`, names its target by path and is a declared dependency, like an import. Links point up the tree or sideways, never down into a child's Definitions; a cycle between Packages is a compile error, as in Go. A Definition with `internal: true` front matter may be linked only from its own subtree.
- There are no explicit layers and no mechanism by which one Kind refines another (7 November 2026). The Package tree carries the Dependency Rule: links up are free, links down are forbidden, sideways edges are checked by the cycle ban and `vet.unstable_dependency`, and a project states its own limits with `internal: true` or an ordinary Rule.
- [The model language guide](../guides/model-language.md) is the canonical description.

<a id="dec-079-the-model-is-as-explicit-as-good-documentation"></a><a id="dec-061-what-an-extracted-model-contains"></a><a id="dec-005-model-granularity-is-the-authors-responsibility"></a>

### DEC-079 The spec is as explicit as good documentation

**Since** 28 October 2026. **Consolidates** DEC-005, DEC-061 and DEC-079.

- A statement belongs in the spec when good documentation of the project would state it that explicitly: domain concepts, features, interfaces such as REST endpoints and CLI commands, rules, decisions and the architecture. Loops, maps and data structures stay with the agents.
- The binding part is what callers rely on: inputs and their validation, data shapes, status and exit codes, stored data. Detail a caller can observe but not reasonably depend on, such as message wording or identifier formats, is free unless the project fixes it. This holds for a spec drawn from existing code too.
- The project owner decides the architecture, and the spec encodes it as its own structure: Packages, their links and their file ownership.
- Granularity is the author's choice. Markitect measures and warns; it imposes nothing ([DEC-102](#dec-102-what-vet-measures)).

<a id="dec-063-the-model-is-authoritative-defects-escalate"></a><a id="dec-033-binding-intent-lives-only-in-the-model"></a><a id="dec-034-removed-intent-removes-its-realization"></a><a id="dec-026-apply-comes-first-and-changes-only-artifacts"></a>

### DEC-063 The spec is the authority, and its defects go to the owner

**Since** 16 and 19 October 2026; the `spec-problem` verdict since 7 November 2026. **Consolidates** DEC-026, DEC-033, DEC-034 and DEC-063.

- Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines and configuration alike. A project rule found outside the spec, in configuration or agent instructions, belongs in the spec.
- The agents that build never change the spec, and the agent that drafts the spec writes no code ([DEC-100](#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). The owner accepts every spec change.
- A gap with no effect at the project's boundary is the builder's to decide. Where the spec contradicts itself or leaves open behaviour at the boundary, nothing is built for that Definition: the reviewer records `spec-problem`, and only the owner settles it.
- Removed intent removes its realization: a file whose only basis was a removed Definition goes, or gets a new basis in the registry.

<a id="dec-006-every-file-is-covered-or-explicitly-ignored"></a><a id="dec-081-the-organization-knows-broad-at-the-top-and-deep-below"></a>

### DEC-006 Every file is accounted for

**Since** 10 October 2026; the registry as Markdown since 28 October 2026; stricter `check` since 7 November 2026. **Consolidates** DEC-006, DEC-081 and the accounting parts of DEC-097 and DEC-098.

- Every file of the project is realized by the spec and listed in the registry, is Markitect's own, or is excluded: by the project's `.gitignore` files or by Markitect's ignore file, `.markitect/ignore`. One file may realize several Definitions.
- The registry is one Markdown page per Package: the path patterns it owns, most specific first as in `CODEOWNERS`, and each file with the Definitions it realizes. Root owns what no other Package claims, and a file shared by several Packages sits with their lowest common parent.
- `check` refuses a file neither listed nor ignored, a registry row for a Definition or file that no longer exists, and a Definition that nothing links to and no file realizes. Accounting proves that nothing is unexplained, not that the realization is right.

### DEC-104 A link is never decorative; identity and surface are written as code writes them

**Since** 8 November 2026 (0.25.0), decided under the owner's delegation of that day.

- The spec behaves like code, and code has no "see also": every link is a real relation that matters for meaning and, in some direction, for change. A link to a Definition says that the source relies on what the target states; a link to a Package says that the source's module depends on that module or, down the tree, says something about a part of the parent. A link that would only help a reader find something does not belong in the spec; navigation belongs in documentation rendered from it.
- The language does not change: there are no relation keywords, and `check` and `status` stay as they are. What code writes with a type and with visibility, the spec writes the same way:
  - a record that others name by its identifier gets an identity Definition of its own in its Package, such as `members/member-id`; the record and its identity link each other, and each holder links the identity instead of the record, as domain-driven design's typed identity; a holder that relies on more than the identity links that as well;
  - a Package's surface is its visibility: what is not `internal: true` is what it publishes, and the spec does not list it again;
  - a module names the modules it uses by Package, `[[members]]`.
- `doc PACKAGE` shows a Package's public Definitions apart from its internal ones, and every Definition that `list` and `doc` show carries `internal` in their JSON.
- Two link keywords, `id` and `part`, were weighed and not chosen. They come back only on new evidence: a real spec whose surface must differ from its visibility, or whose identity Definitions, measured over its history, cost more than the re-verification they save ([OQ-015](#open-questions) is the one point left open).
- [What makes a good spec](../guides/good-spec.md#every-link-is-a-relation) states the patterns, the [measurements](../validation/spec-relations-20261108.md) give the evidence, and the archived [relations plan](https://github.com/Glacius-Labs/Markitect/blob/942d04bda68dde1c90be70c23587b6aea21ab008/docs/concepts/relations-plan.md) weighs the alternatives: on the archived Makerspace specs the patterns put 113 items on the list instead of 133 over eight change steps, and 354 instead of 468 on the 5× project, with no needed item lost.

## The tool

<a id="dec-096-markitect-computes-and-checks-a-driver-acts"></a>

### DEC-096 The tool computes and checks; a driver acts

**Since** 6 November 2026 (0.20.0); made a tool that steers no work on 7 November 2026 (0.21.0). **Consolidates** DEC-096 and the tool parts of DEC-099.

- The `markitect` program checks and analyses the spec, says what is to verify, and records verdicts. It starts no agent, keeps no run open, imposes no order, counts no rounds and keeps no state between commands outside the project.
- Who starts which agent, when and how often is the driver's: the skills, the person's own session, a workflow or the person ([DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)). A driver may change the prompts, agents, skills, settings, limits and every Git step; it may not change the tool, the record's inputs or what the hook refuses.
- The commands are flat and named after Go's tools and other widely known ones: `init`, `check`, `vet`, `list`, `doc`, `status`, `mark`, `tidy`, `config`, `update`, `install`, `version` and `licenses`, and `hook` for the hooks. Diagnostics read `file:line: code: message`; `--json` prints one document with a `format_version` and a `kind`; exit 0 means all is well, 1 that the command found something, 2 that it could not run, and a note never changes the exit code.
- [Concept 4.0, §3](v4-concept.md#3-the-tools) describes the commands.

### DEC-097 Markitect uses no Git

**Since** 6 November 2026 (0.20.0).

- The tool makes no repository, commits nothing, keeps no branches, lands nothing and reads no history. Agents and people do every Git step.
- It reads the project by walking its folder, skipping what the project's `.gitignore` files exclude, read as plain text.
- So it cannot guarantee that what is committed is what was verified. `status` on any checkout shows what differs from the record, and a CI job can refuse it. Markitect's own release tooling still packages its source from Git; that is not part of the product.

<a id="dec-027-strict-links-with-an-optional-optimizer"></a>

### DEC-102 What `vet` measures

**Since** 8 November 2026 (0.23.0). **Consolidates** DEC-102, the `vet` part of ENH-002 and the strict links of DEC-027.

- `check` is the compiler; `vet` is the static analyzer. It warns about coupling and instability per Package, dependencies on less stable Packages, the size of Definitions and Packages, Packages that hold nothing, and hubs: a Definition with fan-in + 1 > max(10, N / 5), Root included, Kinds left out. It exits 1 only with `--strict`.
- Notes never change the exit code; plain `vet` counts them on one line and `--strict` lists them. They are unlinked mentions, which the auditor triages into missing links and ordinary words, with the mentions a link to which would go against the Package structure or add a dependency counted apart; long Definitions (more than 20 statements); placement; and hotspots, revisions × fan-in > max(20, N / 10), from how often a Definition's own text changed between verdicts.
- `list --deps` shows Martin's abstractness and distance, as information only. The zone notes and a risk measure wait for real records. Every threshold is provisional, and `vet.unlinked_mention` becomes a warning only once a spec's own measurements show its noise low.
- Links stay strict: the tool reports a missing link and never adds one. A value rule of the project's own is stated once, as a small Definition in the only Package that uses it or at the lowest common parent of those that share it; there is no central page of values.
- [What makes a good spec](../guides/good-spec.md) says what each measure is for.

## Verification

<a id="dec-099-markitect-is-a-tool-with-a-verification-cache"></a><a id="cpt-004-the-specification-stays-effective"></a>

### DEC-099 A record of what was verified

**Since** 7 November 2026 (0.21.0); revisions and case counts since 8 November 2026 (0.23.0). **Consolidates** DEC-099, CPT-004 and the record parts of DEC-102.

- For each Definition, `.markitect/review.sum` records the inputs at which a reviewer last verified it, by their hashes: its text, its Kind and the Definitions it links to (a Library Definition by its Library's digest), its registry rows with their files, and the evidence. It is committed with the project, like `go.sum`, and merges with `merge=union`.
- `markitect status` lists every Definition with a changed input or no record, those recorded `fail` or `spec-problem`, those gone from the spec, and the errors in the realization. It gives no list while the spec has errors. An empty `status` means that the project compiles and every Definition is verified at its current inputs.
- `markitect mark DEFINITION@FINGERPRINT VERDICT` records `ok`, `fail`, `skip` or `spec-problem` at the state the reviewer read, with its evidence (`path` or `path:line`) or why. It refuses when an input changed since `status` printed the fingerprint.
- A verdict holds until one of its inputs changes, so the specification stays effective: every obligation keeps a verdict, and a change puts back on the list only what it reaches, one step along the links.
- It is a record, not a proof. The tool keeps no limit on rounds.

<a id="cpt-002-tests-are-realizations-too"></a><a id="dec-087-required-checks-gate-merged-states-not-each-submission"></a>

### CPT-002 Tests are the project's, and they are realizations too

**Since** 10 October 2026; required checks removed on 4 November 2026, which **supersedes** DEC-087; tests as Rules since 7 November 2026.

- Tests are one form in which the spec is realized. Agents write them, so they can be as wrong as the code: a green test is evidence, not proof.
- A project that wants tests states them in its spec as an ordinary Rule, with the command that runs them. The reviewer runs them as evidence, and a routine finds them in the spec, the README or the CI configuration. The tool knows nothing of them.
- There are no required checks. The owner removed them to see whether they are needed; if a Definition is verified that a test would have caught, they come back.

## Skills and roles

### DEC-100 Skills are named after the activity, agents after the role

**Since** 7 November 2026 (0.22.0). **Consolidates** DEC-100 and the skill parts of DEC-099; amended by DEC-103 (0.24.0).

- The skills are `markitect-design`, `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit`, `/markitect-schedule` and `/markitect-config`. Only the owner starts build, cleanup, bughunt and schedule; Claude may start design, review, audit and config. A skill only names the agent and its task; the agents hold their tasks and verdict words in their own files.
- Four roles in a 2×2 of rights, held by a hook for subagents; the owner's own session stays free but does not record. The **builder** writes code and the registry pages, not the spec, and may start only the reviewer. The **reviewer** changes nothing and is the only one that records with `mark`. The **architect** writes the spec, the registry pages and the ignore file, not code, and hands its questions back. The **auditor** reads only and reports to the owner. No subagent writes the record or what steers the agents.
- What any agent notices goes into the record as `fail` or `spec-problem`, and nowhere else.
- Loops have no limit on rounds, but stop when the list of `status` has not got shorter for three rounds in a row. A long list runs in parallel, one builder, reviewer or auditor per Package with a Package-scoped task ([DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)).
- The builder works in a worktree of its own, so the owner can go on designing; its work comes back by a merge. Routines work on a branch of their own and land only when they leave nothing worse.

<a id="dec-051-agents-get-tasks-the-orchestrator-owns-the-process"></a>

### DEC-051 Agents get tasks, not process

**Since** 18 October 2026.

- An agent gets a clear task and an answer format. Order, limits, acceptance and routing to the owner are not explained to agents in prompt text.
- A failure in how agents work with Markitect is fixed in the tool contract, not with more prompt text. Since the tool steers no process, the agents' own files hold their task and answer format, and skills hold no steps an agent must remember.

## Driver and configuration

<a id="dec-094-main-carries-the-native-version-and-the-app-goes"></a><a id="dec-083-the-host-runs-natively-each-run-lives-in-containers"></a><a id="dec-092-a-command-uses-the-users-own-sign-in"></a><a id="dec-093-a-command-passes-the-users-own-token"></a>

### DEC-094 The user's own session runs the agents, natively

**Since** 6 November 2026 (Main moved to the native version); 7 November 2026 (no containers, no token). **Consolidates** DEC-094 and what stays of DEC-083, DEC-092 and DEC-093, and of DEC-013 for users.

- Agents run on the user's own machine, started by the user's own Claude Code session, under its stored account login, permission mode and settings, in the unmodified program. No container, no Docker, no token, no API key and no background session.
- The tool runs natively on Windows, Linux and macOS, and Windows is a main path. Agents run with the user's rights, so work that reads untrusted input needs a container or a machine with limited rights around the harness.
- Markitect starts no session and touches no credentials, so each user runs Claude Code on their own login. Claude is the first provider; Codex gets its own driver once Claude's is reliable.
- There is no app for now. If one returns, it works over the same commands.

### DEC-103 Claude chooses the parallelism per run, through a shipped workflow

**Since** 7 November 2026, built for 0.24.0.

- Claude decides per run whether work runs in sequence or in parallel. A skill runs by a **strategy**, set per skill: `auto`, the default, chooses parallel when `status` lists more than about 30 items in at least two Packages (cleanup, bughunt and audit count the spec's Definitions); `sequential` runs one builder, reviewer or auditor at a time; `parallel` runs the shipped workflow; a custom strategy is a saved Claude Code workflow that keeps the same contract: it reads `markitect status --json`, keeps to the four roles and stops by the same rule. There is no fixed tree of Managers.
- The parallel workflow, `.claude/workflows/markitect-parallel.js`, plans strands per Package in waves along the spec's dependencies, what is depended on first. Each strand is a builder in a worktree of its own with the reviewer it starts; shared files, which Root owns, go to Root's strand, last and alone; at most 8 strands run at once. It stops when three rounds in a row leave the list no shorter than it has been, and reports what remains. Review, bughunt, cleanup and audit run per Package the same way. Parallel work needs a Git repository, and the session that starts it waits for it.
- **Configuration** is one file format, `harness.yaml`, at two levels: the user's `~/.markitect/harness.yaml`, not committed, and the project's `.markitect/harness.yaml`, committed. Key by key the project beats the user, and the user beats the built-in values, `sonnet` and effort `high`. It holds the model and effort per role, the strategy by default and per skill, and the custom workflows. `markitect config [--user] KEY VALUE` sets a value and rewrites the agents' files; `config --list` shows each value with its origin; `/markitect-config` takes a request in plain words. `init` and `update` write the agents' front matter from it.
- The hook refuses a subagent's write of `harness.yaml` or of a shipped workflow, and a subagent's start of a workflow. Routines take no strategy, since nobody is there to answer when Claude Code asks before a workflow starts. A user's own skills in `.claude/skills/` are left alone, and the hook holds the roles there too.

<a id="dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps"></a><a id="dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis"></a><a id="dec-011-detailed-planning-of-future-ideas-waits"></a>

### DEC-084 Continuous work runs outside the tool, and autonomy grows in steps

**Since** 28 October 2026; routines since 7 November 2026. **Consolidates** DEC-004's later axis, DEC-011's planning hold and DEC-084.

- A loop runs as a scheduled job or routine that calls `markitect`; there is no daemon in the product. A budget per task or per day belongs to the job and the driver, never to the tool; when it is used up, the loop stops and says so.
- A trigger, such as an issue, an incident or a dependency update, passes an intake that rewrites it into a task; raw trigger text never reaches an agent. Where a Definition states the expected behaviour, the task becomes a `fail` on it; where the spec is silent, it becomes a spec proposal for the owner.
- Autonomy grows in steps, each an owner decision. Step 1: repairs run on their own, and spec proposals wait in an inbox for the owner. Step 2: the owner grants a mandate, the kinds of spec change agents may accept themselves; a fresh reviewer accepts proposals within it, and the owner gets a daily briefing instead of reviewing each change. Government, agents maintaining the spec within mandates, is the final form ([IDEA-002](#ideas)).
- A wrong repair is reverted; a wrong spec change is reverted together with its realization. A circuit breaker drops the loop back a step when reverts or incidents rise, until the owner widens it again.
- Detailed planning of such ideas waits until a usable, stable version exists.

## Releases and distribution

<a id="dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself"></a>

### DEC-095 Releases, a plugin and a binary that updates itself

**Since** 6 November 2026 (0.15.0).

- A tag `vX.Y.Z` on a commit of Main whose source and plugin version are `X.Y.Z` publishes binaries for Linux, macOS and Windows on amd64 and arm64, a checksums file and the source bundle as a GitHub release. Between releases the source says `-dev`. The retired versions 0.1.0 to 0.14.1 stay unused.
- `plugin/` is the Claude Code plugin `markitect@glacius-labs`, listed in the marketplace repository `Glacius-Labs/Claude-Plugins`, which pins it to a release. Its launcher fetches the binary of its own version, so a plugin update is a binary update. The plugin works only in Claude Code, not on claude.ai or in Cowork.
- `markitect update` installs the newest release over the binary, then refreshes the personal skill and a project's harness and lock. It commits nothing, never touches the spec, and is the only use of the network.
- The download is checked against checksums of the same release, which guards against a damaged file, not against a release by someone who controls the repository.

## How Markitect itself is developed

<a id="dec-076-markitect-is-developed-as-a-regular-software-project"></a><a id="dec-020-markitect-uses-any-library-that-helps"></a>

### DEC-076 Markitect is developed as an ordinary software project

**Since** 26 October 2026.

- Markitect's own repository keeps no spec of itself: agentic coding, tests and ordinary documentation, with one canonical page per topic. Target projects keep their spec under `.markitect/`.
- Product code and tests that handle `.markitect/` use fixtures.
- Markitect uses any library that helps; the architecture gate restricts only imports between its own packages. This consolidates DEC-020 of 10 October 2026.

### DEC-070 Parallel writers where useful

**Since** 24 October 2026.

- As many writing sessions run in parallel as are useful and actually decoupled, each with its own worktree and exclusive file scope. Shared contracts and generated files have one writer at a time.

### DEC-069 Documentation is written in English

**Since** 24 October 2026.

- All documentation on Main is in English. Verbatim owner statements under `sources/` keep their original wording, with a translation where they serve as a reference. Archived files keep their original bytes.

### DEC-038 Learn through complete workflows

**Since** 17 October 2026.

- Explain a substantial redesign through new observations and a visible criterion for improvement; use the accepted foundation in complete workflows; look early at uncertain boundaries; ask whether a mechanism is needed and whether something simpler would do; give short owner updates at milestones. The [working method](../development/working-method.md) owns it.

<a id="dec-077-development-pace-and-linux-first-execution"></a><a id="dec-013-linux-first-for-tests-and-the-playground"></a>

### DEC-077 Linux CI, run on request

**Since** 26 October 2026; amended on 7 November 2026. **Consolidates** DEC-077 and the test parts of DEC-013.

- CI runs on Linux, with quality checks and the race detector, started on request. Markitect's own code keeps its Linux tests; for users, Windows is a main path ([DEC-094](#dec-094-the-users-own-session-runs-the-agents-natively)).

<a id="dec-071-repository-structure-and-archive"></a>

### DEC-071 History leaves Main through the archive

**Since** 24 October 2026.

- Main holds what holds. Earlier designs, records and plans leave Main in one change: their files are removed, the [archive index](../archive.md) gets an entry, and remaining links point to full-SHA permalinks. Archive refs are protected against deletion and force-push, and the Main-derived archive points also exist as bundles outside the repository.

## Promises

Target outcomes Markitect is built to deliver. Each is to be validated; none is an established result.

<a id="dec-007-more-economical-models-means-capable-mid-tier-models"></a>

| ID | Promise |
|---|---|
| PRM-003 | The repository stays consistent, contradiction-free, uniform and clean through many changes, so cleanups and refactorings become dependable. |
| PRM-004 | People concentrate on deciding and modeling. Their review moves from code diffs to spec changes and `spec-problem`s, while the code reliably reflects the spec. Routine human code review becomes unnecessary because the realization can be trusted; a fresh reviewer stays part of the method, and the project's acceptance policy still applies. |
| PRM-005 | Capable mid-tier models, such as Sonnet, do most of the delegated work, so routine delegation does not depend on the most expensive models. Haiku counts if it proves sufficient. Lower cost is a possible benefit, not the purpose. |
| PRM-006 | Software becomes understandable: the spec states what the system does and should do, and people can reason over it without reading the code. |
| PRM-007 | Order pays off: structure and careful definition are rewarded instead of turning into endless cleanup. |
| PRM-008 | Nothing in the spec's scope is silently forgotten. What is to verify comes from the spec and the record, not from an agent's memory, and unknown impact remains visible. This covers declared links; code dependencies and unlinked mentions are known gaps. |
| PRM-009 | Rules are applied where they matter. A Rule reaches every Package below it, an `ok` means the Rules above hold for that Definition, and a fresh reviewer checks it. |

## Assumptions

| ID | Assumption | Evidence |
|---|---|---|
| ASM-001 | Agents overlook more relevant files and rules as projects grow, and explicit consistency runs miss things, take longer and struggle with contradictions. | The owner's experience; not measured |
| ASM-002 | A reviewer with fresh context is more critical than self-review in the same thread, even on the same model. | Owner observation; a fresh checker caught 17 of 18 planted bugs blind ([record](../validation/k-checker-test-20261103.md)) |
| ASM-003 | With a typed spec and a computed list, capable mid-tier models do most of the implementation and review. | One Sonnet launch built the 5× project from its spec and passed 327 hidden cases ([record](../validation/default-arm-20261104.md)); one launch, not a rate |
| ASM-004 | What works in human teams transfers to agents: four eyes, bounded roles, and clear responsibility. | Owner hypothesis; not yet evaluated over a series of changes |
| ASM-005 | A reviewer on a different model finds more defects, because reviewers on the same model share blind spots. | Untested ([OQ-004](#open-questions)) |
| ASM-006 | Forgetting, rather than misunderstanding, is the main problem. | Hypothesis from an independent assessment; untested |

## Concepts

<a id="cpt-001-federalism"></a>

### CPT-001 Trust comes from structure

**Since** 10 October 2026, the owner's term "federalism"; the Manager tree it first described went on 4 November 2026.

- Markitect does not try to make agents infallible. Trust comes from four eyes, roles with bounded rights, need to know, a list computed from the spec rather than from memory, and a global check of every Definition against the spec. The [vision](../vision.md#where-trust-comes-from) lists the mechanisms.

## Enhancements

Agreed directions that are not scheduled. The [delivery plan](../implementation-plan.md) owns what is scheduled.

### ENH-001 Observed code dependencies widen impact

**Since** 10 October 2026: the problem is accepted, the approach open.

- The spec's links do not contain the code's own dependencies: calls, imports, shared types. Optional, language-specific tools could observe them and widen what `status` lists, never narrow it. Until then, `/markitect-review --all` is the safety net.

<a id="enh-002-model-quality-linter-and-metrics"></a>

### ENH-002 Spec-quality signals beyond `vet`

**Since** 10 October 2026; the rest of its direction is `vet` ([DEC-102](#dec-102-what-vet-measures)).

- Two signals that a spec is too fine or too coarse are not measured yet: files that realize unusually many Definitions, and the size of a spec change against the code change it causes. Like `vet`, they would be findings and recommendations, never gates.

<a id="enh-003-test-to-statement-binding-and-test-review"></a>

### ENH-003 Tests bound to Definitions

**Since** 10 October 2026.

- Bind tests to the Definitions they verify, review whether a test really verifies its Definition, and report Definitions whose only evidence is an unreviewed test. The reviewer's case counts on `mark` are a first step ([DEC-102](#dec-102-what-vet-measures)); a size-against-coverage measure waits for them.

<a id="enh-004-per-role-agent-configuration-and-model-mixing"></a><a id="enh-006-configurable-teams-for-perspectives-and-four-eyes"></a>

### ENH-004 Models and perspectives per role

**Since** 10 and 16 October 2026. **Consolidates** ENH-004 and ENH-006.

- Configure the model per role, and allow mixing, such as one model building and another reviewing, as an experimental factor ([ASM-005](#assumptions)). Distinct perspectives, such as security or architecture, and the four-eyes principle come from configurable teams whose roles differ in question, context, tools and model, with no hierarchy per perspective. Since 0.24.0, `harness.yaml` sets the model and effort per role ([DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)); mixing models as an experimental factor, and teams per perspective, are not built.

<a id="enh-005-exchangeable-executor-for-the-delegated-method"></a>

### ENH-005 A second provider

**Since** 10 October 2026, as an exchangeable executor.

- A provider needs subagents with their own agent files and a hook before a tool call that names the agent. Codex gets its own driver once Claude's version is reliable; the tool does not change for it.

## Open questions

| ID | Question | Related |
|---|---|---|
| OQ-001 | How can dependency observation work across languages and file types, and how is its own coverage reported? | [ENH-001](#enh-001-observed-code-dependencies-widen-impact) |
| OQ-002 | Which thresholds show that a spec is too fine or too coarse? `vet`'s are provisional, and the zones of pain and uselessness and a risk measure wait for real records. | [DEC-102](#dec-102-what-vet-measures) |
| OQ-003 | Which model tiers suffice for which roles, and does the total cost fall? | [PRM-005](#promises) |
| OQ-004 | How much does a reviewer on a different model add beyond fresh context? | [ASM-005](#assumptions) |
| OQ-005 | Should Markitect be judged over a series of changes on a long-lived project, and what ground truth makes "nothing forgotten" measurable there, for example predeclared affected sets, seeded cross-cutting rules and hidden cases? | [DEC-009](#dec-009-how-markitect-is-judged) |
| OQ-013 | When the machine lacks a program or a service the work needs, does the record need a verdict of its own beside `skip` and `fail`? Today the builder stops and says so, and its items stay on the list. | [DEC-099](#dec-099-a-record-of-what-was-verified) |
| OQ-014 | What does "reliable" mean? The proposal is two clean passes in a row over the 5× project's eight change steps on Sonnet: every step ends with nothing left to verify without the owner, every hidden case passes, and no refusal of the hook was overridden. | [DEC-009](#dec-009-how-markitect-is-judged), [delivery plan](../implementation-plan.md#current-steps) |
| OQ-015 | A downward Package link that only lists a parent's children names what the tree already says, so under [DEC-104](#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them) it is a pointer. Should the language refuse it, or a guide advise against it? No guide does yet, and the skill's example names its Packages by link. | [DEC-080](#dec-080-the-spec-is-typed-documentation-in-markdown), [DEC-104](#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them) |

## Ideas

Ideas that no decision has superseded: the owner's, and three proposals the owner has not assessed (IDEA-001 from the review of 10 October 2026, IDEA-014 from the product chat, IDEA-016 from research). They are recorded, not planned: detailed planning waits until a usable, stable version exists ([DEC-084](#dec-084-continuous-work-runs-outside-the-tool-and-autonomy-grows-in-steps)).

<a id="vis-001-cockpit-personal-ai-and-autonomous-project-organization"></a><a id="idea-001-the-model-as-institutional-memory-for-stateless-agents"></a><a id="idea-002-government-delegated-model-maintenance"></a><a id="idea-003-degree-of-freedom-per-manager-and-rule-strictness"></a><a id="idea-004-project-goals-and-non-functional-priorities"></a><a id="idea-005-reasoned-user-decisions-as-project-precedents"></a><a id="idea-006-dispute-severity-and-higher-review-levels"></a><a id="idea-007-managers-escalate-implementation-cases"></a><a id="idea-008-configurable-local-priorities-at-every-level"></a><a id="idea-009-regular-briefings"></a><a id="idea-010-monitoring-logs-and-data-collection"></a><a id="idea-011-kubernetes-or-etcd-as-later-candidates"></a><a id="idea-012-visualize-the-governed-realm"></a><a id="idea-013-gamification"></a><a id="idea-014-shared-evidence-bound-events"></a><a id="idea-015-cockpit-with-a-conversational-personal-agent"></a><a id="idea-016-optional-offline-claim-and-evidence-classifier"></a><a id="idea-017-operational-check-through-the-managers"></a>

| ID | Idea |
|---|---|
| VIS-001 | The owner's long-term picture: a cockpit with a personal AI, and an autonomous project organization in which the owner models, sets rules and develops ideas while agents keep the project realized and tidy. |
| IDEA-001 | Proposal: the spec as institutional memory: agents keep neither memory nor responsibility over time, and the spec and its record supply both. |
| IDEA-002 | Government: agents maintain the spec within explicit mandates, with the owner briefed and asked only about what matters; departments such as security or architecture raise objections on spec changes. It must keep the spec's quality high. |
| IDEA-003 | Rules can carry a strictness, and work a degree of freedom. |
| IDEA-004 | Each project states a target picture or a ranking of its non-functional properties, against which trade-offs are weighed. |
| IDEA-005 | The owner's reasoned decisions kept as precedents that later cases can cite, like an interpretation of the law. |
| IDEA-006 | Disagreements carry a severity, with higher instances to settle them. |
| IDEA-007 | An agent can bring a case it cannot settle to a higher instance while it works, beyond a `spec-problem`. |
| IDEA-008 | Local priorities, goals and settings at every level of the spec, with the inherited ones kept visible. |
| IDEA-009 | Regular briefings: a short summary of what happened and what waits for the owner, as the routines' notes begin to give. |
| IDEA-010 | Monitoring, logs and data collection for more autonomous operation. |
| IDEA-011 | Kubernetes or etcd for a more autonomous operation, once a concrete need is stated. |
| IDEA-012 | A view of the project: responsibilities, activity, decisions and health. |
| IDEA-013 | Gamification that makes verified progress visible, without false incentives. |
| IDEA-014 | Proposal: the same evidence-bound events feed briefings, history, observability and visualization. |
| IDEA-015 | A cockpit with a conversational agent that searches the project, proposes spec changes and shows what interests the owner. |
| IDEA-016 | Proposal: an optional offline classifier of claims and evidence, never a replacement for `check` or the reviewer. |
| IDEA-017 | Logs of a running system checked against the spec, as a kind of `doctor`. |
