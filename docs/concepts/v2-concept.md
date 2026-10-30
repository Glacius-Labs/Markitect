# Markitect concept 2.0

**The model is typed documentation; a Manager organization derived from it realizes, checks and lands every change.**

| | |
|---|---|
| **Version** | 2.0 – accepted target |
| **Date** | 28 October 2026 |
| **Basis** | The owner's decisions of 28 October 2026 in the Topics session's design walk-through: [DEC-079](register.md#dec-079-the-model-is-as-explicit-as-good-documentation) to [DEC-085](register.md#dec-085-markdown-tools-are-the-cockpit-for-now), and [DEC-061](register.md#dec-061-what-an-extracted-model-contains) as confirmed that day. The register entries govern; this page puts them in one readable order. |
| **State** | Accepted by the owner as Markitect's target on 28 October 2026 ([DEC-086](register.md#dec-086-concept-20-is-the-target)). Being built in the steps of the [delivery plan](../implementation-plan.md#concept-20-build). |

## 1. What Markitect is for

When a project is built with AI agents, its design, rules, documentation, code, tests and infrastructure drift apart. Consistency runs cannot tell what is correct, and the owner loses the overview ([vision](../vision.md#markitect-in-brief), [founding intent](../vision.md#founding-intent)).

Markitect keeps one canonical model of the project that a compiler checks for structure, and has it realized by an organization of agents derived from it: broad knowledge at the top, deep knowledge below, every area with its own responsible Manager.

**Priority.** The artifacts must be consistent with the model and of good quality, so that the owner does not have to review code, nothing drifts and the process can be trusted. Time and token cost come second ([DEC-082](register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)).

## 2. The model

### 2.1 Definitions

Everything in a model is a Definition: one Markdown file with YAML front matter ([DEC-080](register.md#dec-080-the-model-is-typed-documentation)).

```markdown
---
kind: std/Feature
---
# Create reservation

A [[members/member|member]] reserves a [[equipment/machine|machine]] for a slot. …
```

- The front matter holds the Kind and, where needed, `internal: true`. Nothing else.
- The file name is the ID; the main heading is the title; the body is documentation with links.
- A Package is a folder. Packages nest. The model's root folder is the Root Package and holds the project itself: the project anchor, project-wide Rules, layout conventions and the architecture, which by location reach every Manager.
- There are no typed fields. Exact values such as wire names and error codes are written in prose, in backticks.

### 2.2 Kinds and Libraries

- Core offers one thing: a Definition has a Kind and a body. A Kind is itself a Definition, of Kind `Kind`, whose body says what it is and how it is used.
- A Library is a set of Kind Definitions for one technical area, so that not every project defines them. No Kind has a special meaning to Markitect.
- The standard Libraries are `std` for every project (Project, Rule, Concept, Feature, Decision), `rest` (Endpoint) and `cli` (Command). More areas come only when a real project needs them. A project writes its own Kinds for anything else, one small file each.
- The standard Libraries ship inside the `markitect` binary. Other Libraries are committed under `.markitect/libraries/<name>/`. The manifest names each Library with its version, and the lock pins its digest. The Libraries are rebuilt as 1.0.0, with no compatibility to earlier versions; existing models are converted, and frozen study material stays at its recorded commits.

### 2.3 Links, scope and visibility

- A link names its target by path: `[[members/member]]` inside the model, `[[std/Rule]]` for a Library. Links may target Packages. There are no imports.
- A link between two Packages is a dependency.
- **Scope is location:** everything a Package defines reaches its whole subtree.
- **Visibility as in Go:** a Definition is public unless it says `internal: true`; only its own Package and the Packages below it may link to an internal Definition. Visibility binds links, not reading: any Manager above and any checker may read a child's internal Definitions when reviewing, and registry entries may link any Definition they realize. Internal Definitions do not enter other Packages' Managers' starting context.
- **Cycles as in Go:** every link between two Packages is an edge, and so is the implicit edge from a child to its parents. A cycle is a compile error, so links point up or sideways, never down. A cycle is broken by moving the shared concept into a common parent, or by inverting the dependency: the needing side describes its need, and the other side links to it.
- A parent may link to a child Package, which names part of the organization and is no edge ("authorization is realized in [[pipeline]]"), but not to a Definition inside it.
- Using a Kind is an edge, so shared Kinds sit at Root or in a Library.

### 2.4 How explicit

The model states what good documentation of the project would state, no more ([DEC-079](register.md#dec-079-the-model-is-as-explicit-as-good-documentation)): domain concepts, features, interfaces such as REST endpoints and CLI commands, rules, decisions and the architecture. Loops, maps, data structures and algorithms are left to the agents. For a model drawn from existing code, the binding part is the contract callers rely on; details a caller can observe but does not rely on stay free ([DEC-061](register.md#dec-061-what-an-extracted-model-contains)).

## 3. Tools

One tool, `markitect`, written in Go, with no Python and no other runtime ([DEC-080](register.md#dec-080-the-model-is-typed-documentation)). Everything it writes is Markdown or has a Markdown form.

| Command | Does | Agents |
|---|---|---|
| `init` | Sets up a project (manifest, lock, project anchor, checks) and installs or updates the skills; never touches the model | none |
| `check` | Compiles: errors only | none |
| `lock` | Writes the lock from the manifest | none |
| `refs`, `graph` | Who links to a Definition; the dependency graph | none |
| `impact --base <commit>` | What a model change reaches: Definitions, files, Packages | none |
| `lint` | Measures the model like code: warnings only | none |

**Compile errors:** a Kind or link that does not resolve; a link to an internal Definition from outside its subtree; a cycle between Packages; a Library version that does not match the lock; a repository file that is neither registered nor ignored; two Packages claiming the same path pattern; a registry entry for a Git-ignored file. A stale registry entry, which links a Definition the owner deleted or names a file that no longer exists, is no compile error: `lint` warns, and the owning Manager gets it as a checklist item, so deleting a Definition never blocks the Run that removes its realization.

**Lint warnings:** coupling and instability per Package; dependencies from stable to unstable Packages; size of Definitions and Packages; Definitions nothing links to and no file realizes; the reach of a change; a Package with no Definitions of its own and only one child; a Package other than Root that claims no path pattern; later, change hotspots from Git history.

## 4. The realization record

Next to the model, outside it ([DEC-081](register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)):

- **Registry:** one Markdown page per Package. It names the path patterns the Package owns (several, for example `src/billing/**`, `tests/billing/**`, `docs/billing*.md`) and lists every file there with links to the Definitions it realizes and a short reason. When the owner adds a Package, the owner also writes its registry page with the first path patterns, just as the owner writes its Definitions. The owning Manager maintains the page from then on: it lists the files and may narrow or extend its patterns. A page that exists at the Run's base with no patterns counts as the owner's first patterns, so its Manager may extend it like any other page. A change that moves files between two Packages changes both pages, each by its own Manager, so it goes through their common parent ([DEC-060](register.md#dec-060-owners-realize-others-use-parents-settle-cross-area-matters)). A change to the patterns takes effect only from the next Run, because the hooks take each Manager's paths from the Run's base; so no Manager can take over a sibling's files within a Run. The compiler checks the page. The most specific pattern wins, as in `CODEOWNERS`; Root owns every file no other Package claims, so every file has exactly one owner. Owning is not accounting: every file must still be listed on its owner's page or be ignored, so a file that enters silently, even inside an owned folder, is an error.
- **Shared files** such as the README, a CI pipeline or a compose file are owned by the lowest common parent of the Packages they serve, often Root, and their entry links every Definition they realize. A child that needs a change there asks its parent ([DEC-060](register.md#dec-060-owners-realize-others-use-parents-settle-cross-area-matters)).
- **Ignore list:** files that deliberately belong to no Package.
- **`execution.yaml`** holds only the required checks: name, command, directory, timeout and environment. They are the one proof in a Run that does not come from an agent.

Layout conventions, such as where tests live, are Rules in Root.

**Markitect's own files** under `.markitect/` (Definitions, registry pages, manifest, lock, `execution.yaml`) need no registry entry; the compiler accounts for them itself. The hooks enforce who changes them: the model's Definitions, the manifest, the lock and `execution.yaml` never change in a Run, only through the owner, `markitect init` or `markitect lock`; in a Run, a registry page changes only through its own Package's Manager; a new Package's first page comes from the owner (§4).

## 5. The organization

- Every Package is a Manager; the Package tree is the Manager tree, with Root at the top as today.
- A Manager knows its own Package's Definitions, everything its parents define, and the public Definitions it links to in other Packages. Of its children it sees a table of contents of their public Definitions (title and first sentence) and reads the full text when it needs to, for example to review their work. Breadth at the top, depth below, comes from where things are placed.
- A parent settles matters that cross its children's areas and owns what they share.

## 6. Runs

### 6.1 Who works

Impact decides, like an incremental build ([DEC-082](register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)). A model change reaches the Definitions it changes; through the registry the files that realize them; through links the Packages that depend on them; and their parents up to Root. Untouched areas do not start. Impact does not follow the code's own imports: the required checks always run on the whole repository and catch what a change breaks there, and a full check catches what impact missed: `verify --full`, at every milestone the owner names and once ten changes have landed since the last full check, or since the first landing. `apply --full` counts as a full check too, since its checkers review the whole tree; Runs that do not land do not count. Every Result says how many changes have landed since then, and from ten on that a full check is due; the host starts nothing by itself, and ten is a starting value to adjust with evidence.

### 6.2 One Run, maker and checker

One Run handles a change. Every working Manager is two agents with their own harness:

- **The checklist** has three parts: the Manager's own Definitions the change reaches (changed, linking to something changed, or realized by files the change touches); inherited Definitions the change itself edited, so a changed Rule at Root becomes an item for every Manager below it, while an ancestor's Definition that is only reached through a touched file or a link does not; and every Finding assigned to it. With `--full` it holds all own and all inherited Definitions. A verdict means "conforms to this item and to the Rules above it", so a change that breaks an unchanged inherited Rule still fails.
- **The maker** gives every item on its checklist a verdict with evidence: conforms, changed, or model problem. It checks that each makes sense, plans, implements and checks again.
- **The checker** is fresh. It sees the model, the repository at the submission and its diff, and nothing its maker wrote besides the commit: it does not read the Shared Notes. It re-checks every verdict and sends defects straight back to the maker.
- **What a verdict covers** ([DEC-088](register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)): the model in full for each item, the code of the Manager's own Package only. It is no review of the whole model: a verdict covers the whole Definition, its examples included, and the Definitions it links to; where they contradict each other, nothing can conform, and the verdict is model problem. The evidence for conforms or changed shows how each example in the Definition is met, for example by a test, since changed also claims that the item conforms after the change. The checker judges its own Package's files against its items; whether the rest of the program uses them yet is the parent's integration, which the parent's checker judges and the required checks gate on the merged state (§6.3). A parent integrates by using its children's work, not by building a child's behavior a second time in its own files, and its checker rejects a file that does; shared files stay as in §4.

A parent with items of its own in the change submits its own files first, and its children start from that approved commit, so shared files such as a module file exist before the children build on them; a parent without own items lets its children start at once. The host enforces this order on every level, by when it starts the children; the required checks block only merged states (§6.3). The parent reviews and integrates its children's approved work. Root's checker is the final independent check before landing. The model is authoritative and is never repaired by a Run ([DEC-063](register.md#dec-063-the-model-is-authoritative-defects-escalate)): a Manager that cannot settle a matter within the model escalates it to its parent, which settles it if the model allows; a gap without effect at the project boundary is realization freedom, decided and documented by the Manager. A real model defect, a contradiction or a gap that changes behavior at the boundary, ends the Run at Root without landing.

### 6.3 Git, hooks and messages

- Each Run has its own Git remote. Every Manager pushes to its own branch. A handoff is a commit; the checker approves with a commit of its own; the parent merges.
- The verdicts are lines in the submission commit's message, like Git trailers. The checker approves or rejects with a commit on the maker's branch that changes no file and only confirms each verdict; a checker never edits.
- A parent's push may contain its own changes in its own paths plus merged child commits, unchanged and approved by that child's checker; it cannot slip edits into a child's area.
- The required checks block only merged states: a parent's submission that holds its children's merged work, and the state that lands ([DEC-087](register.md#dec-087-required-checks-gate-merged-states-not-each-submission)). On any other submission they still run on the branch's full repository state, and their results go to its checker as evidence, but a failure does not refuse the push: a parent's first submission is a partial tree, which a check of the whole repository cannot pass when the build needs a child's files. A child's broken build is caught at its parent's merge. The host also compiles every submission and the state it lands with `markitect check`, so a file that enters unregistered never lands.
- Server-side hooks on the Run's remote carry the guarantees, which agents cannot skip: a Manager pushes only to its own branch and its own paths; a submission compiles; a merged state passes the required checks; every item on its checklist has a verdict. Local hooks in each harness only give early feedback.
- Messages carry assignments, questions and escalations along the tree, never work. Nobody blocks waiting for a message; children wait only for their parent's approved commit (§6.2).
- No question reaches the owner during a Run.

### 6.4 The Result

Every Run ends with one Result page, written by the host from commits, verdicts and checks, never by an agent: the outcome; per checklist item its verdict, evidence and approving checker; what did not take part and why, and whether the required checks ran on the whole repository; decisions only the owner can make, each with a recommendation; and side observations (Shared Notes). Every Command that starts agents returns this Result; Findings and model proposals are part of it.

### 6.5 Landing

A Run starts from a fixed commit of the target branch. If the branch has not moved, the result lands once the required checks pass on it, without the owner's review ([DEC-078](register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)). If it moved, the host merges the new commits in; the result lands only if the merge is clean, the required checks pass on the merged state and the incoming changes touch nothing in the Run's impact. Otherwise it stays on its branch, the Result says why, and a follow-up Run starts from the new state. Runs whose impact overlaps take turns on one branch.

Results live in the repository as Markdown under `.markitect/results/<run>.md`: a landing Result is part of the landing commit; a Result that does not land stays on its Run branch, pushed as `markitect/runs/<run>`. A Run's containers and remote go away when it ends; the branches of landed Runs are deleted, the others stay until a later Run lands over the same impact or the owner deletes them. Markitect never touches the user's working tree: a Run starts from a commit, and landing updates the target branch, which the user pulls like any colleague's push.

## 7. Commands that start agents

| Command | Agents | Changes | Ends with |
|---|---|---|---|
| `apply` | makers and checkers; impact by default, `--full` for the whole tree | artifacts, by landing | the Result |
| `verify` | checkers only; impact or `--full` | nothing | the Result with Findings |
| `improve` | each Manager uses `lint`, `graph` and `refs` on its area | a model proposal | a model diff for the owner |
| `bughunt`, `security` | checkers only, with that focus | nothing | Findings; where the model is silent, model proposals |
| `cleanup` | makers and checkers on the unchanged model; behavior stays | artifacts, by landing | the Result |

Every Command takes a base, by default the last commit Markitect landed. Impact is everything that changed since then, in the model or in the files: a changed file reaches its Definitions through the registry, so `verify` after hand edits checks exactly what they touched, and nothing changed means nothing to do. `--full` covers everything. `bughunt`, `security` and `cleanup` take the whole tree by default, narrowable with `--package`. In `verify`, the checkers write the verdicts as commits without file changes, and the hooks require a complete checklist as in `apply`; the required checks run on the whole repository as in every Run.

`apply` and `verify` come first. The others are short declarations over the same mechanism and follow; projects add their own the same way. `explain` comes later.

**One Finding input for every source.** A Finding names a Definition and says what was observed and what is expected. `apply --findings FILE` takes Findings from `verify`, `bughunt`, `security` or, later, issues and incidents; raw report text never reaches an agent. A Finding whose Definition no longer exists becomes a side note. Each Finding becomes a checklist item of the Manager owning that Definition. A report about something the model does not state is a model proposal, not a Finding.

## 8. Authoring

- The owner accepts every model change. Agents propose them in conversation through the modeling skill, which asks first when something is open; the owner reads the Markdown diff and merges it. A Run never changes the model.
- Extract, drawing a model from existing code, is its own skill and uses the modeling skill.
- The skills ship inside the binary and are installed by `markitect init`.

## 9. Platform

The host runs natively on Windows, Linux and macOS and needs only Docker ([DEC-083](register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers)). Each Run gets one Linux container as its remote, with a bare repository, the server-side hooks and the required checks; the Manager containers clone from it and push to it over the container network. No folder of the user's is mounted into a container. The host's own Git work is putting the starting commit into the Run's remote and landing the result. Providers log in with a normal account, never with an API key.

## 10. Continuous change

Triggers such as issues, incidents and dependency updates pass an intake adapter that rewrites them into tasks; triage turns each into a Finding for `apply` or into a model proposal ([DEC-084](register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps)). The loop runs outside Markitect, as a scheduled job or in CI. Autonomy grows in steps the owner decides: first, repairs run automatically and model proposals wait in a daily inbox; then a mandate lets agents accept certain kinds of model change: a fresh checker accepts proposals within it, and the owner gets a daily briefing. Budgets, undo and a circuit breaker bound the loop. It is built after the core; the first slice is the first step on one pilot project, with GitHub issues as the source.

## 11. Cockpit

No app for now ([DEC-085](register.md#dec-085-markdown-tools-are-the-cockpit-for-now)). Everything is Markdown, so Obsidian or an editor explores the model and its links, and GitHub shows Results. The [model explorer](../design/model-explorer.md) is picked up only if real use shows a gap.

## 12. Kept from concept 1.0

- The model is authoritative; a model problem ends work and goes to the owner ([DEC-063](register.md#dec-063-the-model-is-authoritative-defects-escalate)).
- Agents get tasks, not process; a failure between agent and host is fixed in host state or the tool contract, not with prompt text ([§3.7](v1-concept.md#37-tasks-not-process-d13)).
- Facts that are technically unambiguous are established deterministically; nothing more is formalized than its benefit justifies ([§3.2](v1-concept.md#32-pragmatic-determinism)).
- Conforming artifacts are preserved; optional improvements are noted, not made unasked ([§3.5](v1-concept.md#35-focus-and-preservation)).
- Honest results: approval, technical success and owner acceptance are different statements, and an uncertain state is never reported as success ([§3.6](v1-concept.md#36-honest-results), [§20](v1-concept.md#20-errors-budgets-and-controlled-termination)).
- A command has a finite budget that corrections share; deadlines reach the blocking operation; cleanup is honest ([§20](v1-concept.md#20-errors-budgets-and-controlled-termination)).
- Shared Notes record side observations with their origin and carry no authority ([§19](v1-concept.md#19-shared-notes-and-independent-verification)).
- Providers sit behind a provider-neutral boundary and use account login only ([§16](v1-concept.md#16-providers-and-authentication)).
- The initial gate: if `check` fails, no agent starts ([§5.3](v1-concept.md#53-initial-gate)).
- The host makes no domain decisions; Managers decide within their area ([§3.3](v1-concept.md#33-manager-autonomy-with-clear-responsibility)).
- Delegation runs only through the Manager tree; native provider sub-agents do not bypass it ([§7.4](v1-concept.md#74-no-second-delegation-organization)). Messages go only to the direct parent or children ([§8.2](v1-concept.md#82-communication-along-the-hierarchy-d10)).

## 13. Replaced from concept 1.0

| Concept 1.0 | Concept 2.0 |
|---|---|
| YAML Definitions with typed `spec`, Core 1.2.0 with Properties, value types and interfaces | Markdown Definitions, Core as a single `Kind` (§2) |
| Libraries `values`, `data`, `api`, `practice`, `architecture` | `std`, `rest`, `cli` (§2.2) |
| Imports and aliases; D3's typed references beside body links | Links by path only (§2.3) |
| Modeled scope through `execution.yaml` | Scope is location (§2.3) |
| YAML registry shards and ownership in `execution.yaml` | Registry pages with ownership by path patterns (§4) |
| Assessment, Implement and Verify as full-tree Runs; every Manager in every Run | One Run per change; impact decides who works; maker and checker (§6) |
| Handoffs and coordination as host messages | Git commits; messages for assignments, questions and escalation (§6.3) |
| Publication from stored acceptance into the working tree, or a branch handed back by the runner | Landing as in a merge queue (§6.5) |
| Apply host on Linux only; the Linux runner on Windows | Native host everywhere with Docker (§9) |
| The Python analysis tool | `markitect` subcommands (§3) |
| An invalid child submission still reaches the parent, which rejects it (D11, §8.4) | Server-side hooks refuse it before it lands on the Run's remote (§6.3) |
| Only the exact approved candidate is published (§21.1) | After a clean merge with a moved branch, the merged state lands if the required checks pass and nothing in the Run's impact changed (§6.5) |

## 14. Open

**Questions for the owner before building** (raised by the independent review of this draft; all ten decided on 28 October 2026 and folded into the sections above):

1. **Root and the project Package: decided** 28 October: the project Package is Root itself.
2. **Overlapping path patterns: decided** 28 October: the most specific pattern wins; Root owns what no other Package claims; two Packages with the same pattern is a compile error.
3. **What a parent may name below it: decided** 28 October: a parent may link to a child Package (no edge) but not to Definitions inside it; using a Kind is an edge.
4. **Internal Definitions and the Managers above: decided** 28 October: visibility binds links; reading upward and registry links stay free.
5. **The checklist: decided** 28 October: own Definitions the change reaches, inherited Definitions the change reaches, and assigned Findings; everything with `--full`.
6. **Hooks and integration: decided** 28 October: parents push their own paths plus unchanged, approved child commits; the checker's approval is a commit without file changes; the required checks run on every submission. Changed on 29 October 2026, after the T7 pilot: they block only merged states (§6.3).
7. **Commands without a model change: decided** 28 October: a base (by default the last landed commit) with impact over model and file changes; focus Commands take the whole tree or `--package`; in `verify` the checkers write the verdicts.
8. **Control files: decided** 28 October: accounted automatically; the hooks refuse model, manifest, lock and `execution.yaml` changes in a Run and let only a Package's own Manager change its registry page.
9. **Results and leftovers: decided** 28 October: Results in the repository under `.markitect/results/`; Run branches cleaned up after landing; the working tree is never touched.
10. **Acceptance: decided** 28 October (§15).

**Decided after the first build steps** (29 October 2026, on an independent check of the code against this concept): landing compiles every submission and the state it lands; `verify` runs the required checks; the checker does not read the Shared Notes; impact does not follow the code's own imports (§6.1); how planted defects are scored (§15); after the first native Run, a parent submits its own files before its children start (§6.2); after the first suite runs, the owner sets a new Package's first path patterns, and pattern changes take effect from the next Run (§4); after an independent assessment, the full check is `verify --full` at milestones and after ten landed changes (§6.1), and the counted series claims only what it measures (§15); after the T7 pilot, the required checks block only merged states, not each submission (§6.3, [DEC-087](register.md#dec-087-required-checks-gate-merged-states-not-each-submission)); on 30 October 2026, after the pilot and the Codex smoke, a verdict covers the model in full and the code of its own Package only (§6.2, [DEC-088](register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)).

**Details for the build:** the verdict trailer format; the harness and model of makers and checkers; the format of a mandate; the Result page's exact layout.

**Parked:** the Windows test time ([question 33](../workstreams/topics.md#questions-for-the-owner)), to be judged again after the rebuild.

**Order:** the design is settled first; the evaluation and the implementations are then built against it.

## 15. Acceptance

Concept 2.0 is achieved when the evaluation suite shows, on a small project (the library) and a larger one (Makerspace), each over a series of changes:

1. **Correct:** every landed change passes hidden behavior tests that no agent saw.
2. **No drift:** after the series, an independent `verify --full` finds nothing left over: no stale file, no forgotten rule, every file accounted for.
3. **Nothing skipped:** in every Result, every checklist item has a verdict with evidence.
4. **Stops right:** planted model defects end the Run without landing; harmless gaps do not. A result equal to its start counts as not landed, for every arm alike, and every arm gets the same instruction: when the requirements contradict each other or leave a boundary behavior open, change nothing and say why.
5. **At least as trustworthy as today's practice:** on the same series, the results are at least as consistent as one strong agent with ordinary documentation ([DEC-002](register.md#dec-002-trust-is-measured-against-todays-practice)). Cost is reported, not judged.

**What the series claims.** The counted series claims only what it measures: at least the baseline's quality, and fewer defects left for a person to repair per change, given comparable input. It does not show that a person saves time. That is measured apart, after the series: on the new project, the owner takes three changes per arm to acceptance, blinded, and records the minutes and the corrections. Until then less human attention stays a [promise](register.md#promises). If the effort for the requirements and the model was not recorded, the report names that as a limit.

The same series runs natively on Windows and on Linux, on our own machine. The exact numbers, the change series and the planted defects belong to the evaluation design.
