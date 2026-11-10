# Decisions

The decisions that hold for Markitect, grouped by topic, and the questions still open. The [concept](concept.md) describes the product they add up to, and the [vision](vision.md) says why.

- This page lists only what holds. A decision changes only through a new decision: it takes the next free number, and the entries it replaces leave the page in the same change. Git keeps every earlier wording.
- Only the product owner adds or changes a decision. An agent's proposal enters as an open question and stays one until the owner decides.
- Do not reopen a decision without new evidence. With such evidence, add an open question that cites it and leave the decision as it is.
- An entry keeps its ID, and the IDs it absorbed keep an anchor on it, unless the anchor names a retired design.
- If another page differs from a decision, the decision holds: correct the page.

## Purpose

### DEC-101 People architect their project in Markdown

- People design, define and model their project in Markdown: what its concepts are and mean, how they relate, its interfaces, structure, rules and way of working. Architect is a role that the owner and an agent fill together: the agent drafts, analyses and proposes, and the owner decides.
- The spec is typed prose in Packages, with links that work like imports. A deterministic tool checks and analyses it like code, and agents realize and verify it.
- Code has always been a precise description for a limited but diligent executor. An AI executor is intelligent, but it still has to be told what is meant and what holds, or it guesses. So documentation takes the place of code as what people work on, and Markitect makes its authors define what they mean and what holds and connect those statements. What matters in code stays as relevant there: architecture, consistency, cleanliness, an overview without reading the code, and knowing what should be as against what is.
- Contradictions in a description surface only when someone changes the place where they sit, as with dynamically typed programming languages. So Markitect borrows typing and a compiler as helpers; nothing in the spec is executed. The aim is to meet the preconditions that open the established knowledge of software architecture to the spec: Definitions with a type, declared dependencies, and a structure a tool can check and measure.
- It builds on proven practice: requirements traceability with suspect links (the registry, `status` and `mark`), architecture metrics (`vet`) and Go's conventions (the command line). What is new: the traceability is plain, typed text checked deterministically, agents do the realization and the evidence, and the record of what was verified works like Go's test cache. AI makes affordable what ordinary projects dropped for its manual cost, such as keeping a traceability matrix.
- The honest limit: "complete" means complete for the declared links.

<a id="dec-001-delegated-realization-is-central-to-the-product"></a>

### DEC-001 Delegated realization is the purpose

- People maintain the spec. Agents realize it in code, tests, configuration and documentation, and verify the result. The tool computes and checks, so that the delegated work can be trusted.
- Judge Markitect as a way of developing software, not as a linter or a context tool. Removing delegated realization as scope creep contradicts this decision.

<a id="cpt-001-trust-comes-from-structure"></a><a id="cpt-001-federalism"></a>

### DEC-002 Trust is measured against today's practice

- The baseline is common practice: an agent implements a task, another AI reviews the pull request, and the human approval is often given almost blind. Markitect aims to earn more trust than that, not compiler-grade guarantees.
- Trust comes from structure, not from infallible agents: four eyes, roles with bounded rights, need to know, a list computed from the spec rather than from memory, and a check of every Definition against the spec. The [vision](vision.md#where-trust-comes-from) lists the mechanisms.
- Claims read "more reliable than the baseline", never "correct". Use the smallest process that earns the trust: more report fields, mandatory probes or completeness declarations do not by themselves make work more correct.

<a id="dec-003-the-compiler-is-a-borrowed-concept"></a><a id="cpt-003-two-parts-of-the-promise"></a>

### DEC-003 The compiler is a borrowed idea, and two claims stay apart

- The spec's structure compiles deterministically: Kinds, links, Packages and file accounting. Meaning is judged by agents. Markitect claims no semantic guarantee for what agents build.
- Two claims stay apart. *Nothing in the spec's scope is overlooked* can be computed from declared links and file accounting; undeclared dependencies in code are a known gap. *Everything is realized correctly* can only be supported by evidence: a fresh reviewer, the project's tests and `check`.

<a id="dec-009-the-central-evaluation-question"></a><a id="dec-008-the-delegated-method-has-not-yet-been-tested"></a><a id="dec-010-evaluate-the-method-separately-from-its-execution-runtime"></a>

### DEC-009 How Markitect is judged

- The aim is one clean and simple version that works reliably on Sonnet, with no babysitting, not to prove that one version beats another.
- The long-term questions: when people develop against a spec, can it be realized reliably and can that be checked; does the repository stay consistent and clean; does agentic coding become more trustworthy; are rules followed instead of overlooked; can people move their attention to architecture?
- The method is judged apart from any runtime: a failure of Markitect's own tool is a product finding, a harness or environment failure a limit of the measurement.
- The method has not yet been tested over a long series of changes. The [concept](concept.md#13-what-is-shown-and-what-is-not) says what is shown and what is not.

<a id="dec-014-compatibility-does-not-drive-decisions"></a><a id="dec-016-a-clean-stable-testable-main-and-uniform-structure-first"></a><a id="dec-038-learn-through-complete-workflows"></a>

### DEC-032 Quality, simplicity and stability over effort and compatibility

- Clean architecture, simplicity and stability come first. Effort is no criterion, and neither is compatibility with old versions while Markitect is early. Released files stay immutable; that is a fact, not a promise of support.
- Choose the clean design even when it is more work. Add no aliases, adapters or legacy paths. Delete what is obsolete, source, tests and documents together.
- Main stays clean, stable and testable: tests check Markitect itself, not provider or environment quirks, and the repository has a uniform structure.
- Simpler is more robust, and simplicity is earned through insight. Markitect works with Claude Code's own features rather than building machinery around them.
- A substantial redesign names the new observation behind it and the result that would show it is better, and asks first whether a mechanism is needed at all or something simpler would do.

### DEC-098 One concept describes the product

- The [concept](concept.md) is the single description of the product and the target that Main builds: the spec, a tool that checks it and records what was verified, and a thin default harness in which the user's own Claude Code session starts the agents.
- Earlier concepts are kept on archive branches ([history](history.md)), not on Main.

## The spec language

<a id="dec-080-the-model-is-typed-documentation"></a>

### DEC-080 The spec is typed documentation in Markdown

- A Definition is one Markdown file with YAML front matter that names its Kind. Its path is its ID, its main heading its title, and its body the documentation. There are no typed fields: exact values, such as wire names and error codes, are written in prose.
- A Kind is itself a Definition, of Kind `Kind`. No Kind has a special meaning to Markitect. Projects define their own Kinds, and Libraries supply common ones: `std`, `rest` and `cli` ship in the binary; any other Library is committed under `.markitect/libraries/`. The manifest names each Library with its version, and the lock pins its digest; `tidy` writes it.
- The spec lives in `.markitect/spec/`. A Package is a folder. Everything a Package defines reaches its whole subtree, and the spec's root folder is the Root Package.
- A link, `[[members/member]]`, names its target by path and is a declared dependency, like an import. Links point up the tree or sideways, never down into a child's Definitions; a cycle between Packages is a compile error, as in Go. A Definition with `internal: true` front matter may be linked only from its own subtree.
- There are no explicit layers and no mechanism by which one Kind refines another. The Package tree carries the Dependency Rule: links up are free, links down are forbidden, sideways edges are checked by the cycle ban and `vet.unstable_dependency`, and a project states its own limits with `internal: true` or an ordinary Rule.
- The [spec language guide](guides/spec-language.md) is the canonical description.

<a id="dec-079-the-model-is-as-explicit-as-good-documentation"></a><a id="dec-061-what-an-extracted-model-contains"></a><a id="dec-005-model-granularity-is-the-authors-responsibility"></a>

### DEC-079 The spec is as explicit as good documentation

- A statement belongs in the spec when good documentation of the project would state it that explicitly: domain concepts, features, interfaces such as REST endpoints and CLI commands, rules, decisions and the architecture. Loops, maps and data structures stay with the agents.
- The binding part is what callers rely on: inputs and their validation, data shapes, status and exit codes, stored data. Detail a caller can observe but not reasonably depend on, such as message wording or identifier formats, is free unless the project fixes it. This holds for a spec drawn from existing code too.
- The project's owner decides the architecture, and the spec encodes it as its own structure: Packages, their links and their file ownership.
- Granularity is the author's choice. Markitect measures and warns; it imposes nothing ([DEC-102](#dec-102-what-vet-measures)).

<a id="dec-063-the-model-is-authoritative-defects-escalate"></a><a id="dec-033-binding-intent-lives-only-in-the-model"></a><a id="dec-034-removed-intent-removes-its-realization"></a><a id="dec-026-apply-comes-first-and-changes-only-artifacts"></a>

### DEC-063 The spec is the authority, and its defects go to the owner

- Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines and configuration alike. A project rule found outside the spec, in configuration or agent instructions, belongs in the spec.
- The agents that build never change the spec, and the agent that drafts the spec writes no code ([DEC-100](#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). The project's owner accepts every spec change.
- A gap with no effect at the project's boundary is the builder's to decide. Where the spec contradicts itself or leaves open behaviour at the boundary, nothing is built for that Definition: the reviewer records `spec-problem`, and only the owner settles it.
- Removed intent removes its realization: a file whose only basis was a removed Definition goes, or gets a new basis in the registry.

<a id="dec-006-every-file-is-covered-or-explicitly-ignored"></a><a id="dec-081-the-organization-knows-broad-at-the-top-and-deep-below"></a>

### DEC-006 Every file is accounted for

- Every file of the project is realized by the spec and listed in the registry, is Markitect's own, or is excluded: by the project's `.gitignore` files or by Markitect's ignore file, `.markitect/ignore`. One file may realize several Definitions.
- The registry is one Markdown page per Package: the path patterns it owns, most specific first as in `CODEOWNERS`, and each file with the Definitions it realizes. Root owns what no other Package claims, and a file shared by several Packages sits with their lowest common parent.
- `check` refuses a file neither listed nor ignored, a registry row for a Definition or file that no longer exists, and a Definition that nothing links to and no file realizes. Accounting proves that nothing is unexplained, not that the realization is right.

### DEC-104 A link is never decorative; identity and surface are written as code writes them

- The spec behaves like code, and code has no "see also": every link is a real relation that matters for meaning and, in some direction, for change. A link to a Definition says that the source relies on what the target states; a link to a Package says that the source's module depends on that module or, down the tree, says something about a part of the parent. A link that would only help a reader find something does not belong in the spec; navigation belongs in documentation rendered from it.
- The spec has no relation keywords, and `check` and `status` treat every link alike. What code writes with a type and with visibility, the spec writes the same way:
  - a record that others name by its identifier gets an identity Definition of its own in its Package, such as `members/member-id`; the record and its identity link each other, and each holder links the identity instead of the record, as domain-driven design's typed identity; a holder that relies on more than the identity links that as well;
  - a Package's surface is its visibility: what is not `internal: true` is what it publishes, and the spec does not list it again;
  - a module names the modules it uses by Package, `[[members]]`.
- `doc PACKAGE` shows a Package's public Definitions apart from its internal ones, and every Definition that `list` and `doc` show carries `internal` in their JSON.
- Link keywords such as `id` and `part` were weighed and not chosen; they are added only on new evidence: a real spec whose surface must differ from its visibility, or whose identity Definitions, measured over its history, cost more than the re-verification they save ([OQ-015](#open-questions) is the one point left open).
- [What makes a good spec](guides/good-spec.md#every-link-is-a-relation) states the patterns. On the archived Makerspace specs they put 113 items on the list instead of 133 over eight change steps, and 354 instead of 468 on a project five times as large, with no needed item lost ([measurements](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/spec-relations-20261108.md)).

## The tool

<a id="dec-096-markitect-computes-and-checks-a-driver-acts"></a>

### DEC-096 The tool computes and checks; a driver acts

- The `markitect` program checks and analyses the spec, says what is to verify, and records verdicts. It starts no agent, keeps no run open, imposes no order, counts no rounds and keeps no state between commands outside the project.
- Who starts which agent, when and how often is the driver's: the skills, the person's own session, a workflow or the person ([DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)). A driver may change the prompts, agents, skills, settings, limits and every Git step; it may not change the tool, the record's inputs or what the hook refuses.
- The commands are flat and named after Go's tools and other widely known ones: `init`, `check`, `vet`, `list`, `doc`, `status`, `mark`, `tidy`, `config`, `update`, `install`, `version` and `licenses`, and `hook` for the hooks. Diagnostics read `file:line: code: message`; `--json` prints one document with a `format_version` and a `kind`; exit 0 means all is well, 1 that the command found something, 2 that it could not run, and a note never changes the exit code.

### DEC-097 Markitect uses no Git

- The tool makes no repository, commits nothing, keeps no branches, lands nothing and reads no history. Agents and people do every Git step.
- It reads the project by walking its folder, skipping what the project's `.gitignore` files exclude, read as plain text.
- So it cannot guarantee that what is committed is what was verified. `status` on any checkout shows what differs from the record, and a CI job can refuse it. Markitect's own release tooling still packages its source from Git; that is not part of the product.

<a id="dec-027-strict-links-with-an-optional-optimizer"></a>

### DEC-102 What `vet` measures

- `check` is the compiler; `vet` is the static analyzer. It warns about coupling and instability per Package, dependencies on less stable Packages, the size of Definitions and Packages, Packages that hold nothing, and hubs: a Definition with fan-in + 1 > max(10, N / 5), Root included, Kinds left out. It exits 1 only with `--strict`.
- Notes never change the exit code; plain `vet` counts them on one line and `--strict` lists them. They are unlinked mentions, which the auditor triages into missing links and ordinary words, with the mentions a link to which would go against the Package structure or add a dependency counted apart; long Definitions (more than 20 statements); placement; and hotspots, revisions × fan-in > max(20, N / 10), from how often a Definition's own text changed between verdicts.
- `list --deps` shows Martin's abstractness and distance, as information only. The zone notes and a risk measure wait for real records. Every threshold is provisional, and `vet.unlinked_mention` becomes a warning only once a spec's own measurements show its noise low.
- Links stay strict: the tool reports a missing link and never adds one. A value rule of the project's own is stated once, as a small Definition in the only Package that uses it or at the lowest common parent of those that share it; there is no central page of values.
- [What makes a good spec](guides/good-spec.md) says what each measure is for.

## Verification

<a id="dec-099-markitect-is-a-tool-with-a-verification-cache"></a><a id="cpt-004-the-specification-stays-effective"></a><a id="cpt-002-tests-are-the-projects-and-they-are-realizations-too"></a><a id="cpt-002-tests-are-realizations-too"></a><a id="dec-087-required-checks-gate-merged-states-not-each-submission"></a>

### DEC-099 A record of what was verified

- For each Definition, `.markitect/review.sum` records the inputs at which a reviewer last verified it, by their hashes: its text, its Kind and the Definitions it links to (a Library Definition by its Library's digest), its registry rows with their files, and the evidence. It is committed with the project, like `go.sum`, and merges with `merge=union`.
- `markitect status` lists every Definition with a changed input or no record, those recorded `fail` or `spec-problem`, those gone from the spec, and the errors in the realization. It gives no list while the spec has errors. An empty `status` means that the project compiles and every Definition is verified at its current inputs.
- `markitect mark DEFINITION@FINGERPRINT VERDICT` records `ok`, `fail`, `skip` or `spec-problem` at the state the reviewer read, with its evidence (`path` or `path:line`) or why. It refuses when an input changed since `status` printed the fingerprint.
- A verdict holds until one of its inputs changes, so every obligation keeps a verdict, and a change puts back on the list only what it reaches, one step along the links. It is a record, not a proof, and the tool keeps no limit on rounds.
- Tests are the project's, and they are realizations too: agents write them, so a green test is evidence, not proof. A project that wants tests states them in its spec as an ordinary Rule, with the command that runs them; the reviewer runs them as evidence. The tool knows nothing of them, and there are no required checks: if a Definition is verified that a test would have caught, they come back.

## Skills and roles

### DEC-100 Skills are named after the activity, agents after the role

- The skills are `markitect-architect`, `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit`, `/markitect-schedule` and `/markitect-config`. Only the owner starts build, cleanup, bughunt and schedule; Claude may start architect, review, audit and config. The architect skill has the session talk with the owner and the architect agent write the spec. A skill only names the agent and its task; the agents hold their tasks and verdict words in their own files.
- Four roles in a 2×2 of rights, held by a hook for subagents; the owner's own session stays free but does not record. The **builder** writes code and the registry pages, not the spec, and may start only the reviewer. The **reviewer** changes nothing and is the only one that records with `mark`. The **architect** writes the spec, the registry pages and the ignore file, not code, and hands its questions back. The **auditor** reads only and reports. No subagent writes the record or what steers the agents.
- What any agent notices goes into the record as `fail` or `spec-problem`, and nowhere else.
- Loops have no limit on rounds, but stop when the list of `status` has not got shorter for three rounds in a row. A long list may run in parallel, one builder, reviewer or auditor per Package with a Package-scoped task ([DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)).
- The builder works in a worktree of its own, so the owner can go on architecting; its work comes back by a merge. Routines work on a branch of their own and land only when they leave nothing worse.

<a id="dec-051-agents-get-tasks-the-orchestrator-owns-the-process"></a>

### DEC-051 Agents get tasks, not process

- An agent gets a clear task and an answer format. Order, limits, acceptance and routing to the owner are not explained to agents in prompt text.
- A failure in how agents work with Markitect is fixed in the tool contract, not with more prompt text. Since the tool steers no process, the agents' own files hold their task and answer format, and skills hold no steps an agent must remember.

## Driver and configuration

<a id="dec-094-main-carries-the-native-version-and-the-app-goes"></a><a id="dec-092-a-command-uses-the-users-own-sign-in"></a><a id="dec-093-a-command-passes-the-users-own-token"></a>

### DEC-094 The user's own session runs the agents, natively

- Agents run on the user's own machine, started by the user's own Claude Code session, under its stored account login, permission mode and settings, in the unmodified program. No token, no API key and no background session.
- The tool runs natively on Windows, Linux and macOS, and Windows is a main path. Agents run with the user's rights, so work that reads untrusted input needs an isolated machine with limited rights around it.
- Markitect starts no session and touches no credentials, so each user runs Claude Code on their own login. Claude Code is the only provider for now; another comes once Claude's version is reliable, and the tool does not change for it.
- There is no app. If one comes, it works over the same commands.

### DEC-103 Claude chooses the parallelism per run, through a shipped workflow

- Claude decides per run whether work runs in sequence or in parallel. A skill runs by a **strategy**, set per skill: `auto`, the default, is meant to choose parallel when `status` lists more than about 30 items in at least two Packages (cleanup, bughunt and audit count the spec's Definitions), but chooses sequential until the parallel strategy has proven itself on a long list; so far it has passed two small real runs. `sequential` runs one builder, reviewer or auditor at a time; `parallel` runs the shipped workflow; a custom strategy is a saved Claude Code workflow that keeps the same contract: it reads `markitect status --json`, keeps to the four roles and stops by the same rule.
- The parallel workflow, `.claude/workflows/markitect-parallel.js`, plans strands per Package in waves along the spec's dependencies, what is depended on first. Claude Code gives a workflow's agents no tool to start an agent, and lets an agent in a worktree run Git only on its own, so each strand is steps of the script: a builder in a worktree of its own, which commits its work there; a merge step on the run's branch, which merges the strands' branches by name; and a reviewer for the strand's Package on the run's branch, whose verdicts a record step commits. Shared files, which Root owns, go to Root's strand, last and alone; at most 8 strands run at once. It stops when three rounds in a row leave the list no shorter than it has been, and reports what remains. Review, bughunt, cleanup and audit run per Package the same way. Parallel work needs a Git repository, and the session that starts it waits for it.
- **Configuration** is one file format, `harness.yaml`, at two levels: the user's `~/.markitect/harness.yaml`, not committed, and the project's `.markitect/harness.yaml`, committed. Key by key the project beats the user, and the user beats the built-in values, `sonnet` and effort `high`. It holds the model and effort per role, the strategy by default and per skill, and the custom workflows. `markitect config [--user] KEY VALUE` sets a value and rewrites the agents' files; `config --list` shows each value with its origin; `/markitect-config` takes a request in plain words. `init` and `update` write the agents' front matter from it.
- The hook refuses a subagent's write of `harness.yaml` or of a shipped workflow, and a subagent's start of a workflow. Routines take no strategy, since nobody is there to answer when Claude Code asks before a workflow starts. A user's own skills in `.claude/skills/` are left alone, and the hook holds the roles there too.

<a id="dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps"></a><a id="dec-011-detailed-planning-of-future-ideas-waits"></a>

### DEC-084 Continuous work runs outside the tool, and autonomy grows in steps

- A loop runs as a scheduled job or routine that calls `markitect`; there is no daemon in the product. A budget per task or per day belongs to the job and the driver, never to the tool; when it is used up, the loop stops and says so.
- A trigger, such as an issue, an incident or a dependency update, passes an intake that rewrites it into a task; raw trigger text never reaches an agent. Where a Definition states the expected behaviour, the task becomes a `fail` on it; where the spec is silent, it becomes a spec proposal for the project's owner.
- Autonomy grows in steps, each the project owner's decision. Step 1: repairs run on their own, and spec proposals wait in an inbox. Step 2: the owner grants a mandate, the kinds of spec change agents may accept themselves; a fresh reviewer accepts proposals within it, and the owner gets a daily briefing instead of reviewing each change. Agents maintaining the spec within mandates is the final form ([roadmap](roadmap.md#ideas)).
- A wrong repair is reverted; a wrong spec change is reverted together with its realization. A circuit breaker drops the loop back a step when reverts or incidents rise, until the owner widens it again.
- Detailed planning of such ideas waits until a usable, stable version exists.

## Releases

<a id="dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself"></a>

### DEC-095 Releases, a plugin and a binary that updates itself

- A tag `vX.Y.Z` on a commit of Main whose source and plugin version are `X.Y.Z` publishes binaries for Linux, macOS and Windows on amd64 and arm64, a checksums file and the source bundle as a GitHub release. Between releases the source says `-dev`. The retired versions 0.1.0 to 0.14.1 stay unused.
- `plugin/` is the Claude Code plugin `markitect@glacius-labs`, listed in the marketplace repository `Glacius-Labs/Claude-Plugins`, which pins it to a release. Its launcher fetches the binary of its own version, so a plugin update is a binary update. The plugin works only in Claude Code, not on claude.ai or in Cowork.
- `markitect update` installs the newest release over the binary, then refreshes the personal skill and a project's harness and lock. It commits nothing, never touches the spec, and is the only use of the network.
- The download is checked against checksums of the same release, which guards against a damaged file, not against a release by someone who controls the repository.

## How Markitect itself is developed

<a id="dec-076-markitect-is-developed-as-a-regular-software-project"></a><a id="dec-020-markitect-uses-any-library-that-helps"></a><a id="dec-070-parallel-writers-where-useful"></a><a id="dec-077-linux-ci-run-on-request"></a><a id="dec-077-development-pace-and-linux-first-execution"></a>

### DEC-076 Markitect is developed as an ordinary software project

- Markitect's own repository keeps no spec of itself: agentic coding, tests and ordinary documentation, with one page per topic. Target projects keep their spec under `.markitect/`; product code and tests that handle it use fixtures.
- Markitect uses any library that helps; the architecture gate restricts only imports between its own packages.
- Sessions work in parallel where their files do not overlap; shared files have one writer at a time.
- CI runs on Linux only, with the race detector, started on request. Markitect's own tests run on Linux; for users, Windows is a main path ([DEC-094](#dec-094-the-users-own-session-runs-the-agents-natively)).
- [CONTRIBUTING](../CONTRIBUTING.md) is the development process.

### DEC-069 Documentation is written in English

- All documentation on Main is in English.

<a id="dec-071-history-leaves-main-through-the-archive"></a><a id="dec-071-repository-structure-and-archive"></a>

### DEC-071 Main holds only the current state

- Main's documentation describes what holds today, as one concept, and quotes no one. Superseded designs, plans, decisions and dated records leave Main in the change that supersedes them; Git keeps them, and each earlier concept has an archive branch ([history](history.md)).
- A page that relies on a dated result says it in one sentence and links its record: the PR that reported it, or a file at a full commit SHA.

## Open questions

| ID | Question | Related |
|---|---|---|
| OQ-001 | How can dependency observation work across languages and file types, and how is its own coverage reported? | [Roadmap](roadmap.md#later) |
| OQ-002 | Which thresholds show that a spec is too fine or too coarse? `vet`'s are provisional, and the zones of pain and uselessness and a risk measure wait for real records. | [DEC-102](#dec-102-what-vet-measures) |
| OQ-003 | Which model tiers suffice for which roles, and does the total cost fall? | [DEC-009](#dec-009-how-markitect-is-judged) |
| OQ-004 | How much does a reviewer on a different model add beyond fresh context? | [DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow) |
| OQ-005 | Should Markitect be judged over a series of changes on a long-lived project, and what ground truth makes "nothing forgotten" measurable there, for example predeclared affected sets, seeded cross-cutting rules and hidden cases? | [DEC-009](#dec-009-how-markitect-is-judged) |
| OQ-013 | When the machine lacks a program or a service the work needs, does the record need a verdict of its own beside `skip` and `fail`? Today the builder stops and says so, and its items stay on the list. | [DEC-099](#dec-099-a-record-of-what-was-verified) |
| OQ-014 | What does "reliable" mean? The proposal is two clean passes in a row over the 5× project's eight change steps on Sonnet: every step ends with nothing left to verify without the owner, every hidden case passes, and no refusal of the hook was overridden. | [DEC-009](#dec-009-how-markitect-is-judged), [roadmap](roadmap.md#current-steps) |
| OQ-015 | A downward Package link that only lists a parent's children names what the tree already says, so under [DEC-104](#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them) it is a pointer. Should `check` refuse it, or a guide advise against it? No guide does yet, and the architect skill's example names its Packages by link. | [DEC-080](#dec-080-the-spec-is-typed-documentation-in-markdown), [DEC-104](#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them) |
