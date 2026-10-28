# Markitect concept 2.0

**The model is typed documentation; a Manager organization derived from it realizes, checks and lands every change.**

| | |
|---|---|
| **Version** | 2.0 – draft for the owner's review |
| **Date** | 28 October 2026 |
| **Basis** | The owner's decisions of 28 October 2026 in the Topics session's design walk-through: [DEC-079](register.md#dec-079-the-model-is-as-explicit-as-good-documentation) to [DEC-085](register.md#dec-085-markdown-tools-are-the-cockpit-for-now), and [DEC-061](register.md#dec-061-what-an-extracted-model-contains) as confirmed that day. The register entries govern; this page puts them in one readable order. |
| **State** | Designed, nothing built. On the owner's acceptance it replaces [concept 1.0](v1-concept.md) as the target. Until then concept 1.0 describes Main. |

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
- **Visibility as in Go:** a Definition is public unless it says `internal: true`; an internal Definition is visible only to its own Package and the Packages below it.
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

**Compile errors:** a Kind or link that does not resolve; a link to an internal Definition from outside its subtree; a cycle between Packages; a Library version that does not match the lock; a repository file that is neither registered nor ignored; two Packages claiming the same path pattern; a registry entry for a Git-ignored file.

**Lint warnings:** coupling and instability per Package; dependencies from stable to unstable Packages; size of Definitions and Packages; Definitions nothing links to and no file realizes; the reach of a change; a Package with no Definitions of its own and only one child; later, change hotspots from Git history.

## 4. The realization record

Next to the model, outside it ([DEC-081](register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)):

- **Registry:** one Markdown page per Package. It names the path patterns the Package owns (several, for example `src/billing/**`, `tests/billing/**`, `docs/billing*.md`) and lists every file there with links to the Definitions it realizes and a short reason. The owning Manager maintains it; the compiler checks it. The most specific pattern wins, as in `CODEOWNERS`; Root owns every file no other Package claims, so every file has exactly one owner. Owning is not accounting: every file must still be listed on its owner's page or be ignored, so a file that enters silently, even inside an owned folder, is an error.
- **Shared files** such as the README, a CI pipeline or a compose file are owned by the lowest common parent of the Packages they serve, often Root, and their entry links every Definition they realize. A child that needs a change there asks its parent ([DEC-060](register.md#dec-060-owners-realize-others-use-parents-settle-cross-area-matters)).
- **Ignore list:** files that deliberately belong to no Package.
- **`execution.yaml`** holds only the required checks: name, command, directory, timeout and environment. They are the one proof in a Run that does not come from an agent.

Layout conventions, such as where tests live, are Rules in Root.

## 5. The organization

- Every Package is a Manager; the Package tree is the Manager tree, with Root at the top as today.
- A Manager knows its own Package's Definitions, everything its parents define, and the public Definitions it links to in other Packages. Of its children it sees a table of contents of their public Definitions (title and first sentence) and reads the full text when it needs to, for example to review their work. Breadth at the top, depth below, comes from where things are placed.
- A parent settles matters that cross its children's areas and owns what they share.

## 6. Runs

### 6.1 Who works

Impact decides, like an incremental build ([DEC-082](register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)). A model change reaches the Definitions it changes; through the registry the files that realize them; through links and the code's own imports the Packages that depend on them; and their parents up to Root. Untouched areas do not start. The required checks always run on the whole repository, and an occasional full Run catches what impact missed.

### 6.2 One Run, maker and checker

One Run handles a change. Every working Manager is two agents with their own harness:

- **The maker** gives every item on its checklist, the Definitions the change reaches in its area and every Finding it received, a verdict with evidence: conforms, changed, or model problem. It checks that each makes sense, plans, implements and checks again.
- **The checker** is fresh. It sees only the model and the diff, re-checks every verdict and sends defects straight back to the maker.

The parent reviews and integrates its children's approved work. Root's checker is the final independent check before landing. The model is authoritative and is never repaired by a Run ([DEC-063](register.md#dec-063-the-model-is-authoritative-defects-escalate)): a Manager that cannot settle a matter within the model escalates it to its parent, which settles it if the model allows; a gap without effect at the project boundary is realization freedom, decided and documented by the Manager. A real model defect, a contradiction or a gap that changes behavior at the boundary, ends the Run at Root without landing.

### 6.3 Git, hooks and messages

- Each Run has its own Git remote. Every Manager pushes to its own branch. A handoff is a commit; the checker approves with a commit of its own; the parent merges.
- The verdicts are lines in the submission commit's message, like Git trailers.
- Server-side hooks on the Run's remote carry the guarantees, which agents cannot skip: a Manager pushes only to its own branch and its own paths; a submission passes the required checks; every item on its checklist has a verdict. Local hooks in each harness only give early feedback.
- Messages carry assignments, questions and escalations along the tree, never work. Nobody blocks waiting for a message.
- No question reaches the owner during a Run.

### 6.4 The Result

Every Run ends with one Result page, written by the host from commits, verdicts and checks, never by an agent: the outcome; per checklist item its verdict, evidence and approving checker; what did not take part and why, and whether the required checks ran on the whole repository; decisions only the owner can make, each with a recommendation; and side observations (Shared Notes). Every Command that starts agents returns this Result; Findings and model proposals are part of it.

### 6.5 Landing

A Run starts from a fixed commit of the target branch. If the branch has not moved, the result lands without the owner's review ([DEC-078](register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)). If it moved, the host merges the new commits in; the result lands only if the merge is clean, the required checks pass on the merged state and the incoming changes touch nothing in the Run's impact. Otherwise it stays on its branch, the Result says why, and a follow-up Run starts from the new state. Runs whose impact overlaps take turns on one branch.

## 7. Commands that start agents

| Command | Agents | Changes | Ends with |
|---|---|---|---|
| `apply` | makers and checkers; impact by default, `--full` for the whole tree | artifacts, by landing | the Result |
| `verify` | checkers only; impact or `--full` | nothing | the Result with Findings |
| `improve` | each Manager uses `lint`, `graph` and `refs` on its area | a model proposal | a model diff for the owner |
| `bughunt`, `security` | checkers only, with that focus | nothing | Findings; where the model is silent, model proposals |
| `cleanup` | makers and checkers on the unchanged model; behavior stays | artifacts, by landing | the Result |

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

**Questions for the owner before building** (raised by the independent review of this draft):

1. **Root and the project Package: decided** 28 October: the project Package is Root itself.
2. **Overlapping path patterns: decided** 28 October: the most specific pattern wins; Root owns what no other Package claims; two Packages with the same pattern is a compile error.
3. **What a parent may name below it: decided** 28 October: a parent may link to a child Package (no edge) but not to Definitions inside it; using a Kind is an edge.
4. **Internal Definitions and the Managers above.** A parent reviewing a child, Root's checker, or a shared-file entry may need a child's internal Definitions. Does visibility bind only sideways, not upwards?
5. **The checklist.** Exactly which Definitions are on a Manager's checklist: its own Package's Definitions that the change reaches, also the inherited Rules, or everything in its area on a `--full` Run?
6. **Hooks and integration.** A parent pushes merged child commits; where does the checker's approval commit go; must a child's submission pass the whole-repository checks before its siblings are integrated?
7. **Commands without a model change.** What `verify`, `cleanup` and `bughunt` take as their impact when the model has not changed, and who writes the verdicts in `verify`, which has no makers.
8. **Control files.** Must the model files, registry pages, manifest, lock and `execution.yaml` be registered or ignored?
9. **Results and leftovers.** Where Results are stored, especially when nothing lands; cleanup of Run branches and retention of evidence (open since DEC-078); landing while the user's local checkout has uncommitted changes.
10. **Acceptance.** What counts as concept 2.0 being achieved; concept 1.0's acceptance (§25) does not carry over.

**Details for the build:** the verdict trailer format; how far impact follows dependents; the harness and model of makers and checkers; when the full Run is due; the format of a mandate; the Result page's exact layout.

**Parked:** the Windows test time ([question 33](../workstreams/topics.md#questions-for-the-owner)), to be judged again after the rebuild.

**Order:** the design is settled first; the evaluation and the implementations are then built against it.
