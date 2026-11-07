# Markitect concept 4.0

**You model. Markitect's tools check the spec and work out what a change reaches, an ordinary agent, started by the owner's own session, makes the project match, and one tool command decides whether the work counts.**

| | |
|---|---|
| **Version** | 4.0 – accepted target |
| **Date** | 4 November 2026, revised on 7 November 2026 for release 0.20.0 and accepted the same day |
| **Basis** | The [owner's statements](sources/fresh-start-20261104.md) of 4 November 2026, concept 3.0, a workflow trial on one real change, the Claude Code documentation and the help of version 2.1.318 as read that day, and independent reviews of this concept's two drafts. The revision adds the [lean plan](lean-tool-plan.md) and its [trials](../validation/lean-trials-20261107.md) of 7 November 2026. |
| **State** | Accepted by the owner as Markitect's target on 7 November 2026 ([DEC-098](register.md#dec-098-concept-40-is-the-target)), replacing [concept 3.0](v3-concept.md), which is now a record. Main carries its build since 6 November 2026 ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)) ([what real sessions showed](../validation/native-version-20261104.md)). On 6 November 2026 the owner chose a leaner tool that computes and checks, leaves the agents to a driver and uses no Git ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git)). Its [plan](lean-tool-plan.md) was built, tried and released as 0.20.0 on 7 November 2026, and this page describes that release. The Run of 0.17.0 to 0.19.0, with `apply`, `verify`, `submit`, `stop`, landing and run branches, is in this page's history. |

It uses the words of [concept 3.0](v3-concept.md#2-words): spec, implementer, reviewer. Code and older documents still say model, maker and checker.

## 1. The aim

Make one clean and simple version work reliably on Sonnet. The aim is no longer to prove that one version is better than another.

The problem has not changed: a project holds what is, and what should be lives in people's heads. [Concept 3.0 §1](v3-concept.md#1-what-markitect-is-for) states it as six problems, and they stand. What changes is how much machinery sits around the agent.

In the owner's words, the point is that "the spec as is gets applied reliably", with no babysitting, review or other slowing down, so that people "concentrate on what agents cant do by themselves: design and model". Every choice below is judged by that.

**The way of working:**

1. A person designs or changes the spec in an ordinary agent session.
2. When the spec looks right, the person uses the apply skill.
3. Agents started by the person's own session apply the spec as it stands, in the person's folder. They ask nothing about the spec, and nobody has to watch.
4. The spec stays as it was while they work: `markitect gate` refuses a change to it, so a change to the spec is its own step, before the work or after it.
5. When `gate` passes, the session commits the work with its certificate. The person reads the Result when they want to, and answers only when the spec has a problem.
6. Other Commands, such as a bug hunt, a cleanup or a check that spec and code match, also run on a schedule.

The owner's first picture, on 4 November, had the work run on a branch and in a folder of its own while the person goes on modeling. The owner then chose agents started by their own session ([§5](#5-what-apply-does)), so the folder is the work's until it ends.

## 2. What Markitect is

| Part | What it is | Changes with the provider? |
|---|---|---|
| **Spec** | Typed documentation in Markdown, with the registry. Unchanged ([concept 3.0 §3 and §4](v3-concept.md#3-the-spec)) | No |
| **Tools** | One program, `markitect`. It holds every fact and every rule that can be checked without judgment, and each of its commands prints the next step. It starts no agent and uses no Git ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git)) | No |
| **Harness** | The default driver: a builder and a reviewer agent, two short skills that have the owner's session start them, a hook, and the instruction files that `markitect init` writes ([§9](#9-the-harness)) | Yes |

**One goal and one door.** The builder is told: make the project match the spec, and you are done when `markitect gate` passes. Each time it is called, `gate` says what is next: put back what the work may not change, start the reviewer, list or put back files that lie in no item, fix what `check` reports, repair these wrong items, stop with a question for the owner or because the repair rounds ran out, or passed, with a certificate to commit.

This is the main difference from the first draft of this concept, which kept the steps in a skill and added hooks to catch what an agent forgot. With the steps in what the tool prints:

- a forgotten step cannot be certified, and the agent is told what is missing;
- compaction cannot lose the process, because the agent gets its next step from the tool each time;
- the prompt stays a task and an answer format, as [DEC-051](register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process) asks;
- only one small hook is needed.

What is left to the agents is what needs judgment: how to build it, and whether the code meets the spec. What is left to the driver is who starts which agent, when, how often, and every Git step.

There is no container, no Docker image and no token. No command stays running while the agents work: `markitect checklist --start` begins the work and says which agent to start, the owner's session starts it, and the agents only call `markitect` like any other tool. [§5](#5-what-apply-does) says why this is the native version the owner asked for.

## 3. The tools

### Commands on the spec

None of these starts an agent or uses Git. `--repo DIR` names a folder of the project, whose root is the nearest folder with `.markitect/manifest.yaml` (default: the current folder), and `--json` prints JSON for scripts.

| Command | Does | Change from today |
|---|---|---|
| `install` | Writes the personal skill, once per machine. Optional: `init` says the same and gives a project its own skills ([§9](#9-the-harness)) | New |
| `init` | Sets up a project: `.markitect/`, the first page of the spec, the harness, and `AGENTS.md` and `CLAUDE.md` if they are missing. An empty folder is enough. Run again, it refreshes the harness and never touches spec pages | Writes the agents and the new skills. Since 0.20.0 it makes no repository and commits nothing. It writes no file that would hold nothing yet, and it ends by saying what Markitect is and pointing to `/markitect-start` |
| `lock` | Pins the Libraries. It holds a fingerprint of each one, as `go.sum` does for modules, and `check` refuses a Library that no longer matches | None |
| `check` | Compiles the project and fails on any error ([below](#what-check-refuses)). `--spec` looks only at the spec itself | Stale entries and unused Definitions become errors, except in a project with nothing built yet. `--spec` is new |
| `refs TARGET` | Who links to a Definition or Package, and which files realize it | None |
| `impact` | What a change reaches ([below](#how-impact-works)) | Since 0.20.0 the base is a certificate, by default the newest. The per-Package lists are to go; 0.20.0 still prints them |
| `analyze` | Measures the spec's design ([§4](#4-is-the-spec-good)). `--severity` filters its findings, and `--graph` draws the Package dependency graph | New. Replaces `lint` and `graph`. Comes after `/markitect-apply` is reliable ([§13](#13-order-of-work)) |
| `update` | Installs the newest release over the binary, then lets the new binary bring what an earlier build wrote in line with it: the personal skill if there is one, and in a project the harness and the lock, which it rewrites; since 0.20.0 it commits nothing, and says to read the lock before committing it. `--check` changes nothing and exits 1 if anything is out of date; `--files` leaves the binary alone; `--force` replaces a development build | New, on the owner's word of 6 November 2026 ([§16](#16-questions-for-the-owner), [DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). It reuses what `init` and `lock` write and replaces neither |
| `version` | The build, and whether the personal skill and the project match it | Adds the status lines |
| `licenses` | As named | None |

The owner proposed `update` to replace `lock` and `init`'s refreshing. An earlier draft of this concept held it back, because the two commands already did that work. On 6 November 2026 the owner asked for it, with a `version` that says what is out of date, since an upgraded binary left the personal skill and each project's harness behind with nothing to say so. So `update` is thin: it runs `init`'s writers and `lock` over what differs, compares files without regard to line endings, commits nothing, and never touches a spec file. Commands that read a project print one line when its skills and agents are old.

The same day the owner added releases and the binary's own upgrade, so that `update` does what `claude update` does ([DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). It reads the newest tag from the redirect of GitHub's releases page, downloads the binary for the system, checks it against the checksums file of that release and replaces the running executable: by renaming over it on Linux and macOS, and on Windows, which lets a program rename its own executable but not overwrite it, by moving the old file aside first. The new binary then finishes the update, since only it knows the files it ships. This is the only place where Markitect uses the network, and only when the owner runs `update`; without a network it says so and still refreshes the files. A development build older than the newest release (`0.0.0-dev`, or `0.15.0-dev` while `0.15.0` is out) is not replaced unless `--force` says so, so a build from source is safe from a stray `update`; one newer than the release (`0.16.0-dev` after `0.15.0`) counts as up to date. The download is checked against checksums from the same release over HTTPS, so it protects against a damaged or swapped file but not against a release published by someone who controls the repository; signed attestations would add that.

### What `check` refuses

`check` passes or fails, with no warnings: a project either compiles or it doesn't. Its errors come in two groups, and it prints them apart. One case passes: when the registry lists no file and the only errors are Definitions nothing realizes, it says that the spec compiles and nothing is built yet. `gate` makes no such exception.

| Group | Errors | Who settles them |
|---|---|---|
| **The spec** | A Kind or link that does not resolve, a Package cycle, a Library that does not match the lock, and the rest of today's errors in the spec | The owner, in the spec |
| **The realization** | A file neither listed nor ignored, a registry entry for a Definition or file that no longer exists, and a Definition that nothing links to and no file realizes | The implementer, in the work |

So `check` goes red the moment the owner deletes a Definition or adds one nothing uses yet, and stays red until the work has realized the change. That is how it should read: spec and code don't match yet. `--spec` answers the narrower question "is the spec itself valid?", which is what a modeling session asks.

`markitect checklist` stops only on errors in the spec. Errors in the realization become checklist items, and `gate` refuses until none is left.

On finished Makerspace projects, at steps c5 and c8, there are no stale entries and one unused Definition, `documentation`, out of about 115. So a finished project can meet this rule. A rule that applies everywhere still needs a file that realizes it, or a link from something that does.

### How `impact` works

Like an incremental build, it works out what a change touches, so that only that gets reworked and reviewed. It compares the newest certificate, or one `--base` names, with the folder as it stands; "changed" means a different hash ([the certificate](../../internal/run/README.md#the-certificate)).

| Distance | Reached | On the checklist |
|---|---|---|
| 0 | Definitions that changed, Definitions whose files changed, and Definitions a registry row gained or lost | Yes |
| 1 | Definitions that link to a changed one or are of a changed Kind | Yes |
| 2 and more | Everything further along the links, with distance and reason | No |

It doesn't follow the code's own imports. A full review now and then, `/markitect-verify --full`, is the safety net. The list is rebuilt after the work, so a file touched outside the first impact adds its Definitions. On Makerspace's change steps, one step reached 33 of the 36 existing files that truly changed, and deeper reach 35 ([measurement](../validation/impact-reach-20261103.md)). How far every link reaches says something about the spec too ([§4](#4-is-the-spec-good)).

### Commands of the work

These are for whoever drives the agents. They start no agent, keep nothing running and use no Git. [The work](../../internal/run/README.md) owns how they behave; this table only names them.

| Command | Called by | Does |
|---|---|---|
| `markitect checklist` | The skills, the builder and the reviewer | Prints the task and the list of the change from the newest certificate to the folder, or of every Definition when there is none, with what a changed Definition said before. `--start` begins the work, or goes on with the work under way, and says which agent to start; `--review` prints the reviewer's task with only the items still to answer; `--base`, `--full` and `--findings` name the change |
| `markitect answer` | The reviewer | Checks its answers and stores them for the files as they are. A wrong location or an unknown item is refused at once, and it prints what is still missing |
| `markitect gate` | The builder and the skills | The one door ([§6](#6-the-one-door-gate)). `--spec-problem WHY` is the builder's report of a spec problem; `--findings` closes a review of finished work; `--verify` checks that the folder matches its newest certificate |
| `markitect status` | The owner | How the work stands for the files as they are, and which files differ from the newest certificate. It changes nothing |
| `markitect hook` | Hooks only | The check of [§9](#9-the-harness) |

**The folder.** Every command reads the project folder by walking it, without what the project's `.gitignore` files exclude, read as plain text ([the folder](../../internal/run/README.md#the-folder)). So ignored build output does not count, and a folder needs no repository. Files that Markitect's own ignore file lists are walked and hashed like any other; they only need no registry row.

**The base** of a change is the newest certificate under `.markitect/results/`, which `gate` writes when it passes. With no certificate yet, the list holds every Definition. A file changed beside the work, by hand or by a commit made around `gate`, differs from the certificate, so the next list reaches it or `gate` names it.

**The work under way** lives outside the project, in the user's cache folder, one file per project folder: the hashes of what the work may not change, the list and the stored answers. An answer is bound to the hashes of the files it covers and falls away when one of them changes. Nothing of it reaches the project, and a crash leaves it where it was: run `gate` again.

**There are no required checks.** The owner took them out on 4 November 2026, to see whether they are needed. A project that wants tests states that in the spec, with the command that runs them, and the reviewer checks it and may run them. So no proof in the work comes from outside an agent any more, apart from `check` and what `gate` refuses, and that is what this trial tests. If work is certified that a test would have caught, the checks come back. `execution.yaml` goes with them, and the agents' `model` and `effort` move to their agent files and the manifest.

**The ignore file** is `.markitect/ignore`, written like a `.gitignore`: one pattern per line and `#` for a comment. Its patterns stay the registry's, and it has no `!`. A reason is a comment now and no longer required, which is safe because `gate` refuses a change to the spec, the ignore file included, during the work. The tool accounts for what `init` installs by itself, so a new project has no ignore file.

## 4. Is the spec good?

A spec is judged like code ([founding intent](sources/founding-intent-20261026.md)), so it gets the same kind of analysis code gets. `check` is the compiler: is it valid? `analyze` is the static analyzer: is it well designed? It only ever warns, since granularity stays the author's choice ([DEC-005](register.md#dec-005-model-granularity-is-the-authors-responsibility)), and its thresholds are provisional.

| Measure | Code analogy | Warns about | State |
|---|---|---|---|
| Efferent and afferent coupling, instability, per Package | Clean Architecture | A Package depending on too many others, or on a less stable one | Exists in `lint` |
| Size of a Definition and a Package | Long functions, large modules | God Definitions | Exists in `lint` |
| Packages with no paths, or no Definitions and one child | Empty namespaces | Structure that holds nothing | Exists in `lint` |
| Fan-in of a Definition, including at Root | God class, `utils` file, interface segregation | One page that too many depend on | New. `lint.reach` skips Root today, so it misses the worst case below |
| Blast radius: how far every link reaches from a Definition, as median, 90th percentile and worst | Transitive dependents | A change that could touch most of the spec | New |
| Links inside a Package against links leaving it | Cohesion | A Package whose parts belong elsewhere | New |
| Change hotspots across certificates | Churn | What keeps changing, with how far it reaches | Later. The tool reads no Git history ([DEC-097](register.md#dec-097-markitect-uses-no-git)), so it would count changes from one certificate to the next |
| Size against what tests cover | CRAP | Large, unverified Definitions | Later: needs tests bound to Definitions ([ENH-003](register.md#enh-003-test-to-statement-binding-and-test-review)) |
| The spec's dependencies against the code's imports | Architecture tests | Code that has drifted from the design | Later |

**Package cycles stay a compile error**, as in Go. The details of `analyze`, its measures, thresholds and output, are still to be discussed with the owner.

**What this finds on Makerspace, measured on 4 November.** Following every link reached over 85 % of the spec on four of its eight change steps. That looked like a ball of mud, and at the Package level it is not one: there are no Package cycles, dependencies point towards a stable core, and a typical Definition reaches 8 of 113. But it does show a real design smell:

| | Makerspace, 113 Definitions | 5× project, 413 Definitions |
|---|---|---|
| Direct links to one page, `values` | 60 | 268 |
| Definitions that page can reach through every link | 97 | 392 |
| Definitions reaching over half the spec | 9, all at Root | 17, of which 15 at Root |
| Worst reach if the top three hubs pass nothing on | 46 | 188 |

`values` is one page of thirteen unrelated value formats, and most links to it name no format. A Definition that uses only dates still depends, on paper, on the format of money. Treating the top three hubs as stopping points brings the four steps that reached 85 to 88 % down to between 9 and 38 %. In code this would be a `utils` file, and the remedy would be to split it. Whether splitting it really narrows reach, and whether that helps agents, is untested: cutting hubs in a calculation is not the same as a better spec.

**What analysis can't say:** whether the spec is right. No threshold has been checked against outcomes yet, so a clean report doesn't show that a spec leads to better work.

## 5. What apply does

The owner models until the spec looks right, then uses `/markitect-apply` in their own Claude Code session. The tool computes and checks; the skill, the agents and the session do the rest ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)). [The project workflow](../guides/project-workflow.md#7-apply-and-verify) says how to use it, and [the work](../../internal/run/README.md) owns the rules.

1. **Begin.** The skill injects `markitect checklist --start`, which compiles the spec and begins the work: the list of what changed since the newest certificate, or of every Definition when there is none. An error in the spec stops here, no agent starts, and the owner sees why. A folder without the builder, the reviewer or the hook begins nothing either. A change that reaches nothing in the spec leaves nothing to build, and the session runs `gate`.
2. **Start the builder.** `checklist --start` prints one step, start the `markitect-builder` agent, and the owner's session does exactly that. The builder is a subagent: it starts with fresh context, in the owner's folder, and takes its task and list from `markitect checklist`, not from the message that started it.
3. **Implement.** The builder makes the project match the spec and keeps the registry current. It does not change the spec. A gap with no effect at the project's boundary is its own to decide.
4. **Report a spec problem.** If the spec contradicts itself or leaves open something that changes behaviour at the boundary, the builder builds nothing for it and calls `markitect gate --spec-problem "WHY"`. `gate` records the report and has a reviewer confirm it before the owner hears of it, so no agent decides about its own work; neither answer uses a repair round ([lean plan, decision 7](lean-tool-plan.md#decisions-for-the-owner)). If the reviewer finds that the spec settles it, the builder builds on. A program or a service missing from the machine has no report of its own ([§16](#16-questions-for-the-owner), question 11).
5. **Gate.** When the builder thinks it is done, it calls `markitect gate` and does what it prints ([§6](#6-the-one-door-gate)).
6. **Review.** `gate` tells it to start the `markitect-reviewer` agent, and the builder starts it as its own subagent and waits, so it keeps its context through the repairs. The reviewer starts with fresh context, has no tool to edit files and cannot start further agents. It takes everything from `markitect checklist --review`, reads the Definitions and the code, and stores one line per item with `markitect answer` ([concept 3.0 §8](v3-concept.md#8-the-checklist)).
7. **Repair.** A `wrong` answer goes back to the builder, which fixes it and calls `gate` again. Answers are bound to the files they cover, so the next review asks only about what the repair touched; other answers stand. At most two repair rounds.
8. **Certify and commit.** When nothing is left, `gate` writes the certificate and its Result under `.markitect/results/` and ends the work. The builder ends, and the owner's session runs `gate`, which finds nothing left to check, then commits the work with the certificate and the Result in one commit on the current branch, pushes nothing, and reports in a line or two. In a folder that is not a Git repository it commits nothing and says the work is ready.

**The certificate** holds the SHA-256 of every accounted file and every Definition, a copy of the spec text, and each item's answer with its pointers ([the certificate](../../internal/run/README.md#the-certificate)). It is the base of the next change. After a merge of two certified branches, the newer certificate counts, and what only the other covered is reviewed again: that costs some work and loses nothing. A certificate is a record, not a proof: during the work `gate` refuses a changed one, but one written by hand while no work is under way is taken as it is.

**Git is the driver's.** Markitect makes no repository, commits nothing, keeps no branches and lands nothing ([DEC-097](register.md#dec-097-markitect-uses-no-git)). The skill has the owner's session commit on the branch that is checked out. A branch that moved, a merge or a pull request is ordinary Git, by the session or the owner, and the next `gate` judges the folder that results.

**The folder is the work's.** The agents work in the owner's folder, so the owner leaves it alone until the work ends; the session that started the work is busy with it anyway. `gate` refuses a change of the spec, the harness, a `.gitignore` file or a certificate since the work began, and a changed file that lies in no item; a changed file that realizes a Definition puts that Definition on the list instead, so it is reviewed. A change to the spec is its own step, before the work: `/markitect-apply` then begins the work again from the newest certificate and says so. No agent can do that, since the hook refuses a subagent's `checklist` that would begin the work or name a change.

**When the builder ends early.** The owner's session runs `gate` each time an agent has ended and starts the agent it names: the reviewer, or the builder again, which sees in its `checklist` what the last review found wrong. The work under way stays in the user's cache folder, so a session that closed can go on with it: `/markitect-apply` takes it up where it stood.

**Why not a background session.** The first build of this concept had `apply` start `claude --bg` in a worktree of its own, with generated settings and hooks, and it landed real changes ([record](../validation/native-version-20261104.md)). On 5 November 2026 the owner chose agents started by their own session instead: Markitect should be just a way of agentic coding, without that machinery. What the choice costs:

- the owner's session is busy until the work ends, and the agents stop if that session closes;
- the agents work under that session's permission mode, settings, plugins and memory, so they may ask for permission, and they differ from one person to the next;
- an agent that ends early is started again by the owner's session, as `gate` says, instead of being sent back by a hook.

What it keeps: fresh context for the builder and for the reviewer, which is the test that the spec alone is enough, and every refusal of `gate`.

## 6. The one door: `gate`

`gate` looks at the folder and the work under way, does the first thing that applies, and prints the one next step. It exits 0 only when it writes a certificate, or when there is nothing to check. The table is a summary; [the work](../../internal/run/README.md#gate) owns the exact rules.

| | It finds | It says |
|---|---|---|
| 1 | The spec (everything under `.markitect/` except the registry pages and the results), the harness, a `.gitignore` file or a certificate changed since the work began, or a link leaves the folder | Put them back; a change to the spec is its own step, before the work |
| 2 | The builder's report of a spec problem that no reviewer has answered | Start the reviewer, which confirms it or not |
| 3 | A `spec problem`, confirmed by the reviewer of a report or found by a finished review | The question for the owner; the Result says so, and nothing is certified |
| 4 | A file that changed since the base certificate and lies in no item of the list | The files: list them on the registry so that they are reviewed, or put them back |
| 5 | An error from `check`, in the spec or the realization | The errors |
| 6 | Items without an answer for the files as they are | Start the reviewer |
| 7 | A `wrong` answer | The items and which repair round of two this is; after two rounds it refuses, and the Result says so |
| 8 | Nothing left | Writes the certificate and its Result under `.markitect/results/`, ends the work, and says to commit both with the work in one commit |

`not shown` may be certified, and the Result lists it. A change after a review is new work: the answers bound to the files it changed fall away, so the next review covers what it touched. `gate` never commits.

**Two modes for other uses.** `gate --findings` closes a review of finished work ([§8](#8-other-commands)). `gate --verify` checks that the folder matches its newest certificate and names the files that do not, for a checkout in CI or anywhere no agent reaches.

**Limits.** `gate` keeps one: two repair rounds, a rule about the state of the work. The builder's reports are bounded too: none on a state a reviewer turned one down on, and none after two were turned down. Time and the number of calls are the driver's ([lean plan, decision 3](lean-tool-plan.md#decisions-for-the-owner)). The default driver sets none of its own: the owner's session goes on until `gate` passes or stops the work, and the owner can stop it at any time ([§16](#16-questions-for-the-owner), question 4).

**The Result** is written by the tool, never an agent, as in [concept 3.0 §10](v3-concept.md#10-the-result): each item's answer and pointers, the builder's reports with the reviewer's word, the repair rounds, the decisions for the owner, and what could not be tested ([the Result](../../internal/run/README.md#the-result)). It lies beside the certificate, or, when `gate` stops the work, under a name of the state and the outcome, uncommitted. It doesn't measure how much checking really happened from the reviewer's transcript, and says so. The provider says that format can change between versions, and reviewers read mostly through the shell, so the measure says little for its cost.

## 7. How sure we are

The tool can't stop a determined agent that runs as the user on the user's machine. This concept guards against forgetting and shortcuts, which is what has been seen, and against deliberate bypass only by making it visible.

| What the owner wants | What secures it | How firm |
|---|---|---|
| The spec compiles before work starts | `checklist --start` refuses before any agent starts | Firm |
| What `gate` certifies is complete: spec unchanged, project compiles, every item answered for the files as they are, every changed file in an item | The tool | Firm for what `gate` certifies |
| What is committed is what was certified | The driver commits after `gate`; the tool commits nothing ([DEC-097](register.md#dec-097-markitect-uses-no-git)) | Seen afterwards, not prevented: `status` and `gate --verify` name every file that differs from the newest certificate, and a CI check can refuse it |
| The work doesn't stop to wait for someone | Nothing: the agents work under the owner's session's permission mode. The builder file's `permissionMode: acceptEdits` did not apply in the trials; in the session's `auto` mode five of six runs had no denial ([trials](../validation/lean-trials-20261107.md#what-the-flow-showed)) | Weak by choice: the owner is there to answer. How often manual mode asks is not known |
| The work doesn't mix with the owner's own | The owner leaves the folder alone while the work is under way. `gate` refuses a change of the spec, the harness, a `.gitignore` file or a certificate during the work, and a changed file that lies in no item | Weak for code: an edit of the owner's in a file that realizes a Definition puts that Definition on the list, and it is reviewed and certified as part of the work. Only a file in no item at all, such as one the ignore file lists, is refused |
| The work doesn't just stop | The skill has the owner's session run `gate` each time an agent has ended and start the agent it names | Prompt text, not structure. An early end certifies nothing, so it costs time, not correctness |
| The agent doesn't review itself | A hook in the project's settings lets only a reviewer call `answer`; it runs in subagents too. A builder's report of a spec problem stops the work only once a reviewer confirms it | Good, not proof: the hook reads the command's text and the agent's name |
| The reviewer judges for itself | Its agent file says to take everything from `markitect checklist --review` | Weak: the builder writes the message that starts it. Reviews given a maker's claims did no worse on planted bugs ([17 of 18 blind, 18 of 18 with claims](../validation/k-checker-test-20261103.md)), though that tested claims beside bugs, not a message written to steer. No hook is added until this is seen |
| Nothing goes to a remote | The tool has no Git, and the skill says to push nothing | Catches ordinary use only: the owner's own settings decide what a shell command may do |
| A change that bypassed `gate` is noticed | It differs from the newest certificate, so the next list reaches it or `gate` names it, and `status` lists it | Seen afterwards, not prevented |
| The reviewer really read the code | Pointers checked for existence in the folder | Not provable. It was not provable in the Docker version either |
| The code is right | The reviewer and the project's tests | Not guaranteed by any version |
| The work has a limit | Two repair rounds in `gate`; time and calls are the driver's, and the default driver sets none ([§6](#6-the-one-door-gate)) | Bounded in repair rounds only, not in time or calls. The provider's own budget and turn flags work only with `-p`, and there is no spending limit |

**Anyone who wants more** can run `markitect gate --verify`, `markitect check` and the project's tests where the agent can't reach them: on a server of their own, in CI or in a hook on a remote. Markitect depends on none of these. Answers stay on the machine, and the certificate committed with the work carries them.

**What we give up against concept 3.0:**

- prevention of a deliberate bypass;
- a commit made by the tool: the driver commits, so what is committed matches what was certified only if the driver does it right, and a mismatch shows afterwards;
- isolation from the user's machine: agents run with the user's rights, so the work must not read untrusted input;
- one operating system: agents work on the user's own, Windows included, with its paths and shells;
- a reviewer in its own session as another user, with a throwaway folder. It now shares the owner's folder. `gate` catches what it changes in files the walk sees, and code it runs can still change its stored answers;
- review of a clean copy: it reads the folder as it stands, ignored leftovers included;
- a reviewer's task written by the Command, and a measure of how much it checked.

## 8. Other Commands

A Command is a skill over the same tools: which agent the owner's session starts, which list `checklist` names, and how `gate` closes it. The table in [concept 3.0 §12](v3-concept.md#12-other-commands) stands for what each Command is for. The tool's commands run anywhere, from a chat, a terminal or a schedule; the agents need a driver, which today is the skills in a Claude Code session.

**`/markitect-verify`** changes nothing in the project but the pages it writes. It runs `markitect checklist --review --start`, whose list holds what changed since the newest certificate, or since the one `--base` names, or with `--full` every Definition; with no certificate it holds every Definition. The owner's session starts the `markitect-reviewer` agent, which answers the list, and then runs `markitect gate --findings`:

| It finds | It does |
|---|---|
| Items without an answer | Says to start the reviewer again |
| Otherwise | A `wrong` answer on a Definition becomes a Finding, and what names no Definition, such as an error from `check`, is listed in the Result. It writes the Result and, with Findings, a Findings page under `.markitect/results/`, writes no certificate, and ends the review |

The session commits those pages. `/markitect-apply --findings FILE` begins work on them.

`--full` starts with one reviewer for the whole list, and the first `/markitect-apply` on a project with no certificate covers every Definition too. That is untested at size: the 5× build used 51 reviewers, up to eight at a time, each in a clean copy. A split per Package may be needed, and the runs of [§13](#13-order-of-work) decide that.

Apply comes first, then Verify; both exist as skills. `cleanup`, `bughunt`, `security` and `improve` follow once those work. Modeling and Extract stay skills in the owner's own session, as they are, and so does `markitect-start`, the guide a new project begins with. It asks what the project is, one question at a time, and starts the spec with the owner.

## 9. The harness

For Claude Code, `markitect init` writes these project files, so they are versioned with the project, and the tool accounts for them itself.

| File | Holds |
|---|---|
| `.claude/skills/markitect-apply/SKILL.md` | One injected command, `` !`markitect checklist --start` ``, which begins the work and names the agent to start, and "do what it printed": run `markitect gate` each time an agent has ended, and commit the work with its certificate when `gate` passes. Its front matter sets `allowed-tools`, so the commands run without a prompt, and `disable-model-invocation: true`, so a modeling agent can't start the work by itself. The provider runs the command before the model sees the skill, and any non-zero exit aborts the skill, so `--start` always exits 0 |
| `.claude/skills/markitect-verify/SKILL.md` | The same for a review of finished work: `checklist --review --start`, the reviewer, and `gate --findings` |
| `.claude/agents/markitect-builder.md` | The builder: take the task from `markitect checklist`, nothing from the message that started it, and do what the command the task names says, including when to start the reviewer. The task names `markitect gate` as that command and `gate --spec-problem` for a spec problem. Tools Read, Edit, Write, Glob, Grep, Bash and Agent; model `sonnet`, effort `high` |
| `.claude/agents/markitect-reviewer.md` | Effort, the reviewer's task, and tools without edit or agent tools |
| `.claude/settings.json` | The hook on `answer` and `checklist`, below, added to whatever settings the project already has |
| `AGENTS.md`, `CLAUDE.md` | What kind of project this is, for any session: the spec is the authority, changes start there, and `markitect` is on the path. `CLAUDE.md` holds only `@AGENTS.md`, so that it works in every session of the provider. They hold no process, since the agents read them too. Written only if missing |

**The agents' settings** come from their files, and the manifest's `model`, if it names one, which the tool names when it says which agent to start. Everything else is the owner's session: its permission mode, settings, plugins and memory. The builder file's `permissionMode: acceptEdits` did not apply in the trials, where the agents ran in the session's mode ([trials](../validation/lean-trials-20261107.md#what-the-flow-showed)). No settings are generated per piece of work.

**The hook.** `markitect hook pretool`, in the project's settings, runs before a shell command that calls `markitect answer` or `markitect checklist`, refuses `answer` unless the agent is the reviewer, and refuses a subagent's `checklist` that would begin the work or name a change. Hooks in project settings run inside subagents too, and their input names the agent. The work may not change the file.

The first draft's seal is gone. `answer` checks and stores the answers itself, so no hook has to catch a reviewer as it finishes.

**One file lies outside the project.** A session that runs `init` started before the project's files existed, so it knows only what the tool prints, and Claude Code doesn't watch a skills folder that wasn't there when the session started. So help and `init` say what Markitect is, and `markitect install` writes a personal skill, `~/.claude/skills/markitect/SKILL.md`, with the same text from the binary, which reaches a session in any folder. It allows no command in advance. `init` never writes to the home folder; it only tips when the skill is missing or old. `markitect update` installs the newest release and refreshes the skill and the project's files, and `markitect version` says whether they match. The skills and agents call `markitect` by name, so `version`, `init`, `install` and `update` warn when the `PATH` finds no `markitect` or another one.

**The plugin does both.** `plugin/` is a Claude Code plugin, `markitect`, in this repository. The marketplace that lists it, `glacius-labs`, is the small repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins), whose entry pins the plugin to the tag of a release: the plugin that a user gets is the one at the tagged commit, whatever `main` holds. It carries the personal skill as `markitect:getting-started` (the same text, pinned by a test, under a name that does not repeat the plugin's, since Claude Code names a plugin's skill `plugin:skill`), a launcher, `bin/markitect`, which Claude Code puts on the `PATH` of its shell, and two hooks. The launcher reads the plugin's version, downloads the binary of the release with that version into the plugin's data folder, checks it against the release's checksums and runs it, with `MARKITECT_PLUGIN` set so that the binary knows who started it: `update` then leaves the binary to the plugin (`claude plugin update markitect@glacius-labs`), and the `PATH` warning stays quiet, since the launcher is what the `PATH` leads to. A session-start hook has the launcher fetch the binary in the background and outside the shell's sandbox, which may block the network, but only in a project that has a `.markitect/` folder: in any other project the plugin stays out of the way, and the binary is fetched at the first call. The other hook runs `markitect hook pretool` before `markitect answer` and before `markitect checklist`, as the project's hook does, because the project's hook runs in a shell that does not see the plugin's `bin/` and would find no `markitect` there. It looks at the command itself and starts the launcher, and so a download, only for a command that contains `markitect answer` or `markitect checklist`; the `if` filter of the hook alone was seen to start it for other commands too, in 0.17.0. After `init` both run, and both give the same answer; if the launcher fails, this one exits 2 and blocks the call, which is right, since `markitect answer` would fail too. The plugin does not replace `init`: the project's skills, agents and hook still come from `markitect init`.

Skills are kept short, and they hold no steps, so nothing depends on what survives compaction. The harness's agents stay in the project rather than in a plugin: the provider ignores hooks and permission modes in plugin agents. The plugin described above brings only the command, the personal skill and two hooks.

## 10. Drivers

What earlier drafts called a strategy is a driver ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)): whatever starts the agents, decides how often and how long, and makes the Git steps. **The default driver is a folder of harness files and nothing else.** There is no code interface to implement. Adding a version of it means adding a folder, and removing one means deleting it. **It may change** the prompt, agents, skills, settings, limits and every Git step. **It may not change** the tools or what `gate` refuses.

The first version ships one driver and no switch for it. The natural second is a saved provider workflow that holds the loop and the limits in code, as in the trial of [§14](#14-what-is-not-yet-shown), added only if the first one fails too often ([§16](#16-questions-for-the-owner)). A person can drive the work too, with the same commands.

## 11. Providers

The first version is Claude only. A provider needs subagents with their own agent files, and a hook before a tool call that names the agent. Codex gets its own driver once Claude's version is reliable.

No token is needed: sessions run on the user's own stored login in the unmodified program, which removes token passing. Each session uses the user's subscription quota. The legal reading of [concept 3.0 §14, question 10](v3-concept.md#14-decided-on-acceptance) still applies: the owner's own use looks allowed, and other users are not settled. So the owner still asks Anthropic before anyone else runs Markitect.

**The app.** The owner said "we can remove desktop for now". This concept reads that as the whole app, with its spec views and chats. Its chats are the only other use of containers, so this takes Docker out of Markitect altogether, along with the provider adapters, the session helper, the runtime image and the token. It stays recoverable in Git and can come back once the tool has settled. The owner confirmed that reading on 6 November 2026, and the app is gone ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)).

## 12. Schedules and continuous work

[DEC-084](register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps) stands: the loop runs outside Markitect, as a scheduled job that calls `markitect`, and there is no daemon in the product. Its budget per task is `gate`'s two repair rounds and the driver's limits; a budget per day belongs to the job. Government stays a later step ([IDEA-002](register.md#idea-002-government-delegated-model-maintenance)): its mandate would live in the spec, and a fresh reviewer would accept proposals within it.

A schedule needs a provider session that runs `/markitect-verify` or `/markitect-apply`, since the tool starts no agent; the trials ran the apply skill headless, with `claude -p` ([trials](../validation/lean-trials-20261107.md)). Two limits stay:

- **The machine must be on and signed in.** A server without a stored login is not covered yet.
- **External signals need isolation.** Issue text can hold instructions aimed at the agent, and a native agent has the user's rights. Work that reads outside input needs a container or a machine with limited rights around it. This wraps the harness and needs no engine.

## 13. Order of work

| Step | Result | Done when |
|---|---|---|
| 1. Probe | The harness by hand on the Library, on Windows, under a bounded dispatch. It tries what [§14](#14-what-is-not-yet-shown) lists | Done on Windows in throwaway projects: real sessions tried the background version ([record](../validation/native-version-20261104.md)), and the lean trials tried agents started by the owner's session ([record](../validation/lean-trials-20261107.md)) |
| 2. Build | Built first as a Run with `apply`, `submit` and `stop`, released up to 0.19.0; then made lean by the [lean plan](lean-tool-plan.md#order-of-work) and released as 0.20.0: `checklist`, `answer`, `gate`, `status`, `hook` and the stricter `check`, with no Git. `init` writes the harness. The engine's container and session code is gone | Done: three Makerspace changes were certified or stopped as expected, with a planted bug and a planted contradiction caught, before the Run was removed |
| 3. Make it reliable | The 5× project's 8 change steps on Sonnet, with their hidden cases, which are at an [archive point](../archive.md#removed-with-the-evaluation-tools) since the evaluation tools left Main. Failures are fixed in the tool, not with more prompt text | Two clean passes in a row: every step is certified without the owner, all hidden cases pass and no refusal was overridden ([§16](#16-questions-for-the-owner)) |
| 4. The rest | The other Commands, `analyze` and the rename; Verify exists as a skill. From the lean plan: a workflow driver, and a CI check with `gate --verify` | The owner's reading |

Before step 3, read test 1's nine finished launches. They are in `%LOCALAPPDATA%\markitect-test1`, not in the repository, and their answers are unread. They may say at no new cost whether one reviewer catches known bugs at 5×.

Concept 3.0's tests 2 and 3 fall away, since the aim is no longer to show that this beats something else.

## 14. What is not yet shown

**The background version was tried on Windows on small changes ([record](../validation/native-version-20261104.md)).** That record covers the launcher, settings and hooks the owner has since dropped. **Agents started by the owner's own session were tried on 7 November 2026**, on three small Makerspace changes, one run per case, in `auto` mode and headless, on 0.19.0 with the lean commands ([trials](../validation/lean-trials-20261107.md)). What a first run had to show:

- **The skill's printed step makes the owner's session start the builder, and start an agent again when one ended early: shown.** The session ran only `gate`, `git status`, the commit and one read-only look, and where the builder ended without passing `gate`, it ran `gate` and started the agent it named.
- **The builder, as a subagent, starts the reviewer as its own subagent and waits for it: shown.** Four nested reviews in the three lean runs, seven in five runs with the old flow, each at spawn depth 2 with the builder as its parent and a foreground start; lean A's two came around one repair. Nothing had to be configured.
- **The hook in the project's settings fires inside a subagent: shown.** In a probe it refused a subagent's `checklist --full`. No agent called `answer` without being the reviewer, so that refusal was not met in a run.
- **The certificate travels with the work: shown.** Each certified run committed the work, the certificate and its Result in one commit, and `gate --verify` then found the folder matching it. A builder that changed a file's line endings left the answers standing, since a text file's hash ignores them. Landing went with the Run.
- **How often the owner's permission mode stops a builder to ask: open.** In `auto` mode five of six runs had no denial, and in the sixth the classifier twice refused a builder's Git undo of its own edits. Manual mode was not tried.

**Not yet shown in a run:** 0.20.0 itself, since the trials ran 0.19.0, whose `check` still read files through Git, so the Git-free walk is tried by tests only; the builder's report of a spec problem, `gate --spec-problem`, built after the trials and covered by tests only; a real branch, a merge, a crash or a second certified branch; two repair rounds used up; a change larger than about 64 items.

**What the evidence does and doesn't cover:**

- **A plain Sonnet session has done the work from the typed spec, once.** Concept 3.0's default built the 5× project in one launch, with one implementer session and one repair round, and passed all 327 hidden cases ([record](../validation/default-arm-20261104.md)). A reviewer found a real bug there. That is one launch and not a rate, in a container, driven by the Command, and its change steps have not run. So the implementation doesn't look like the main risk; the process around it does.
- **How often a native agent skips steps is unmeasured.** The owner has seen one forget things. With `gate` as the door, a skipped step should show as a refusal, not a wrong certificate. Whether that keeps the work from ending in nothing is what step 3 finds out.
- **The workflow trial is weak evidence.** On 4 November a saved script fixed the order, run through `claude -p` in `bypassPermissions`, with a throwaway check on a local repository. Of four attempts, two ended early: a refused worktree, and a parser that stopped on a helper's extra sentence. In the third the planted bug became a build error, so the reviewer never met it. The fourth started at review, on the second attempt's work plus the plant: the reviewer caught it once, and it landed. It used no skills and no background session. Its evidence is in `C:\Users\Consiliari\wf-trial\evidence`, outside the repository.
- **Reviewing at 5× has no read result.** Test 1 stopped when the machine went down, with nine of twelve launches run.

## 15. What this overturns

| Entry | Becomes | What stays from it |
|---|---|---|
| [DEC-091](register.md#dec-091-concept-30-is-the-target) | Superseded as the target on 7 November 2026 ([DEC-098](register.md#dec-098-concept-40-is-the-target)) | Concept 3.0 as the record; its six problems, words, checklist, answers and Result |
| [DEC-037](register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers) | Superseded: agents and checks no longer run in containers | A reviewer gets no write tools |
| [DEC-013](register.md#dec-013-linux-first-for-tests-and-the-playground), [DEC-077](register.md#dec-077-development-pace-and-linux-first-execution) | Amended: agents work on the user's own system, so Windows becomes a main path | Linux tests for Markitect's own code |
| [DEC-078](register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) | No remote and no Git in the tool ([DEC-097](register.md#dec-097-markitect-uses-no-git)): the owner's session commits the work with its certificate on the current branch, and the owner pushes | Work counts without the owner reviewing a diff; no Git host is needed |
| [DEC-080](register.md#dec-080-the-model-is-typed-documentation) | Amended: "Definitions nothing links to and no file realizes" move from warnings of `lint` to errors of `check`, and the command list changes to `check`, `lock`, `refs`, `impact`, `analyze` | The language, and everything else measured staying a warning |
| [DEC-083](register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers) | No containers and no host | Native on Windows, Linux and macOS |
| [DEC-087](register.md#dec-087-required-checks-gate-merged-states-not-each-submission) | Removed for now, by the owner, to see whether they are needed. `execution.yaml` goes too | `gate` certifies only a state that passes `check`; a merged state is checked by the next `gate`, not at the merge |
| [DEC-066](register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply), [DEC-089](register.md#dec-089-markitect-desktop-the-app-over-the-cli) | Parked: no app for now, as the owner confirmed on 6 November 2026. If it returns, it works over the same commands | The idea of an app over the CLI. The owner being told when the work ends moves to the owner's own session, which waits for it |
| [DEC-092](register.md#dec-092-a-command-uses-the-users-own-sign-in), [DEC-093](register.md#dec-093-a-command-passes-the-users-own-token) | No token and nothing passed on: the session uses the stored login. DEC-092's fallback was to be built only if `claude -p` lost subscription login; that route is no longer used | The unmodified CLI, and the question to Anthropic. The pinned CLI goes, so a change of the provider's defaults can arrive unannounced |
| [The guide's build decisions](../guides/model-language.md#build-decisions) on stale realization, confirmed by the owner on 28 October 2026, and on a required check's environment | Stale entries are errors, not warnings, and the `environment` field goes | The Command that cleans them up still starts: they are errors in the realization |
| [Concept 3.0 §14](v3-concept.md#14-decided-on-acceptance), question 1 | The rename no longer waits on test 3. It comes in step 4, once the sealed project's use is settled | One step, with no compatibility kept |
| [Concept 3.0 §14](v3-concept.md#14-decided-on-acceptance), question 7 | The tool knows no "committed", only the folder: the work begins from the spec as it stands, and the session commits spec, work and certificate together | A changed file that lies in no item is still refused, so nothing is certified unseen |
| [Concept 3.0 §14](v3-concept.md#14-decided-on-acceptance), question 9 | Services a project needs come from the user's machine, not an image | A builder that lacks one has no report of its own yet ([§16](#16-questions-for-the-owner), question 11) |
| [Concept 3.0 §14](v3-concept.md#14-decided-on-acceptance), question 12 | No runs and no branches in the tool: a branch that moved is ordinary Git, and after a merge of two certified branches the newer certificate counts | What only the other branch covered is reviewed again, so a merge loses nothing |
| [Concept 3.0 §6, §9, §11](v3-concept.md#6-what-apply-does) and [§13](v3-concept.md#13-how-we-know-it-works) | The Command driving Runs, the sandbox, and tests 2 and 3 go. Skills stay optional reading, so they hold no process | The rule that a driver can't change the check |
| [Concept 3.0 §8 and §10](v3-concept.md#8-the-checklist) | One reviewer first instead of one per Package at `--full`. A pointer no longer has to show as opened in the transcript, and the Result doesn't measure depth | The answers, pointers, and what each answer covers |

**Stays unchanged:** [DEC-051](register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process), now served by the next step that `checklist` and `gate` print, in place of an Orchestrator, and the working rule in `AGENTS.md` on fixing failures in the tool contract; the spec's authority and escalation to the owner ([DEC-063](register.md#dec-063-the-model-is-authoritative-defects-escalate)), explicitness, every file accounted for, removed intent removing its realization, delegated realization, quality measures as warnings ([ENH-002](register.md#enh-002-model-quality-linter-and-metrics)), continuous change as a later loop, and honest results, finite budgets and provider neutrality from concept 1.0.

## 16. Questions for the owner

Each open one has a recommendation. Already decided by the owner: that required checks go for now to see whether they are needed, that the agents are started by the owner's own session, that the app goes, that `update` comes now, and that releases, the plugin and a binary that updates itself come with it ([DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). The name `submit` and its landing, decided earlier, gave way on 6 November 2026 to a tool that computes and checks, with `gate` as its door ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)), and uses no Git ([DEC-097](register.md#dec-097-markitect-uses-no-git)); on 7 November 2026 the owner decided how a builder reports a spec problem ([lean plan, decision 7](lean-tool-plan.md#decisions-for-the-owner)), and accepted this concept as the target ([DEC-098](register.md#dec-098-concept-40-is-the-target)), which left the open questions below open.

1. **Starting the session.** Decided by the owner on 5 November 2026: the owner's own session starts the agents, with no background session ([§5](#5-what-apply-does)).
2. **What the work may do.** Settled with question 1: the agents work under the owner's own permission mode and settings. The builder file asks to accept file edits without asking, but in the trials the session's mode held instead ([§9](#9-the-harness)).
3. **How the owner hears of the end of the work.** Settled with question 1: the owner's session waits for the builder and sees it end.
4. **Limits.** `gate` keeps two repair rounds; time and the number of calls are the driver's, with no spending limit, since the provider offers none here ([lean plan, decision 3](lean-tool-plan.md#decisions-for-the-owner)). The default driver sets no time or call limit of its own. Recommended: keep it so, and let a workflow driver hold time and calls in code if step 3 shows work that goes on too long.
5. **One driver or two.** The owner asked for "the two simplest approaches". Recommended: one now, the workflow as the second only if this one fails too often. The owner also expected a package per version behind an interface; a driver here is a folder of harness files, with no code interface, because the tool does not change between versions.
6. **The old version.** Settled on 6 November 2026: Main carries this version, and Main until then is kept on the branch `archive/main-concept-3-20261106` ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)).
7. **The suite.** Decided by the owner on 7 November 2026, as recommended: the runner and arms in `tools/evaluation` went, and the Makerspace projects and hidden cases, which step 3 runs on, were to stay. Later the same day, after accepting this concept, the owner had the oracles and every other evaluation tool go as well, those projects and hidden cases included; they are at an [archive point](../archive.md#removed-with-the-evaluation-tools).
8. **`update`.** Decided by the owner on 6 November 2026: it comes now, with a `version` that reports what is out of date ([§3](#commands-on-the-spec)). The Claude Code plugin that would carry the personal skill and update it with the provider's own mechanism, first held back, comes too: the owner said on 6 November 2026 that releases come anyway, and that the plugin should set up what a user needs ([DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)).
9. **The app.** Decided by the owner on 6 November 2026: the whole app goes ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)).
10. **What "reliable" means.** Step 3's two clean passes on the 8 change steps is a proposal for the talk on evaluation the owner asked for. Is that the finish line?
11. **What the machine lacks.** In 0.19.0 a builder that lacked a program or a service, such as a compiler or a database, said so with `markitect stop --environment`, which asked the owner nothing about the spec. That door went with `stop`, and none replaced it: the builder's task now says to stop and say so, but `gate` knows nothing of it. The owner's session follows `gate`, so it may start a reviewer, whose `wrong` for missing work uses a repair round, or the builder again, as the trials saw for a spec problem before the builder's report existed ([finding 1](../validation/lean-trials-20261107.md#findings)). Nothing wrong is certified, but nothing bounds the cost either: a builder that stops again leaves the state unchanged, calling `gate` again on the same state uses no repair round, and the default driver sets no limit, so the session may start the builder again and again until someone notices. Three ways:
    1. **Leave it** until it is met: no new code, and a missing program is rare on the owner's own machine, but a failure the trials already showed for the spec would repeat here.
    2. **Use the spec problem's report:** no new code, but the reviewer would have to confirm a fact about the machine as a `spec problem`, and the owner would be told to settle the spec when the fix is to install something.
    3. **Recommended: give the builder's report a second kind for the machine.** `gate` records it, a reviewer confirms that the program or service is missing (it can look), and `gate` then stops the work and names what to provide, using no repair round. It follows decision 7, so no agent decides about its own work, and it fixes the gap in the tool contract rather than in prompt text.
