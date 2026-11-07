# Markitect concept 4.0

**You model. Markitect's tools check the spec and say what is left to verify, an ordinary agent, started by the owner's own session, makes the project match, and a fresh reviewer verifies each Definition and records it.**

| | |
|---|---|
| **Version** | 4.0 – accepted target |
| **Date** | 4 November 2026, revised on 7 November 2026 for release 0.20.0 and accepted the same day; §3, §5 to §9 and §12 revised for the record of DEC-099, released as 0.21.0, §3 and §4 for its output and names, and §1, §2, §5, §7 to §9 and §11 to §16 for DEC-100's skills, roles and routines, both released as 0.22.0 |
| **Basis** | The [owner's statements](sources/fresh-start-20261104.md) of 4 November 2026, concept 3.0, a workflow trial on one real change, the Claude Code documentation and the help of version 2.1.318 as read that day, and independent reviews of this concept's two drafts. The revision adds the [lean plan](lean-tool-plan.md) and its [trials](../validation/lean-trials-20261107.md) of 7 November 2026. |
| **State** | Accepted by the owner as Markitect's target on 7 November 2026 ([DEC-098](register.md#dec-098-concept-40-is-the-target)), replacing [concept 3.0](v3-concept.md), which is now a record. Main carries its build since 6 November 2026 ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)) ([what real sessions showed](../validation/native-version-20261104.md)). On 6 November 2026 the owner chose a leaner tool that computes and checks, leaves the agents to a driver and uses no Git ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git)). Its [plan](lean-tool-plan.md) was built, tried and released as 0.20.0 on 7 November 2026. The same day the owner decided that the tool becomes a record of what was verified, modelled on Go's test cache, that steers no work ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache), [tooling plan](tooling-plan.md)). Its first step, the record, was released as 0.21.0; its second, the output and the names `tidy`, `vet`, `list` and `doc`, was released as 0.22.0 with the skills, roles and routines the owner decided the same day ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). §3 to §9 and §12 describe them; `checklist`, `answer`, `gate`, the certificate and the Result of 0.20.0, and the Run of 0.17.0 to 0.19.0, with `apply`, `verify`, `submit`, `stop`, landing and run branches, are in this page's history. |

It uses the words of [concept 3.0](v3-concept.md#2-words): spec, implementer, reviewer. Code and older documents still say model, maker and checker.

## 1. The aim

Make one clean and simple version work reliably on Sonnet. The aim is no longer to prove that one version is better than another.

The problem has not changed: a project holds what is, and what should be lives in people's heads. [Concept 3.0 §1](v3-concept.md#1-what-markitect-is-for) states it as six problems, and they stand. What changes is how much machinery sits around the agent.

In the owner's words, the point is that "the spec as is gets applied reliably", with no babysitting, review or other slowing down, so that people "concentrate on what agents cant do by themselves: design and model". Every choice below is judged by that.

**The way of working:**

1. A person designs or changes the spec in an ordinary agent session, with the design skill.
2. When the spec looks right and is committed, the person uses the build skill.
3. Agents started by the person's own session apply the spec as it stands, in a worktree of their own, so the person can go on designing. They ask nothing about the spec, and nobody has to watch.
4. The spec stays as it was while they work: the hook keeps the agents that build out of it, so a change to the spec is the person's own step, and what it touched goes back on the list to verify.
5. When `markitect status` lists nothing, the session commits the work with the record of what was verified and brings it to the person's branch. The person reads the verdicts when they want to, and answers only when the spec has a problem.
6. Other Commands, such as a bug hunt, a cleanup or a check that spec and code match, also run on a schedule, each on a branch of its own.

The owner's first picture, on 4 November, had the work run on a branch and in a folder of its own while the person goes on modeling. The owner then chose agents started by their own session ([§5](#5-what-build-does)), and on 7 November 2026 had the builder work in a worktree of its own again, still started by their session ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)).

## 2. What Markitect is

| Part | What it is | Changes with the provider? |
|---|---|---|
| **Spec** | Typed documentation in Markdown, with the registry. Unchanged ([concept 3.0 §3 and §4](v3-concept.md#3-the-spec)) | No |
| **Tools** | One program, `markitect`. It holds every fact and every rule that can be checked without judgment: it checks the spec, says what is to verify and records what was verified. It starts no agent, steers no work and uses no Git ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git), [DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)) | No |
| **Harness** | The default driver: four agents in a 2×2 of rights, short skills named after their activity that have the owner's session start them, routine prompts, a hook, and the instruction files that `markitect init` writes ([§9](#9-the-harness)) | Yes |

**A record of what was verified.** For each Definition the tool keeps the state at which a reviewer last verified it, as Go's test cache does for tests, and `markitect status` lists every Definition whose state changed since ([§3](#how-status-works), [§6](#6-the-record)). The builder is told: make the project match the spec for what `status` lists, and have the reviewer verify and record each item. When nothing is left to verify, the spec is implemented.

This replaces 0.20.0's one door, `gate`, which printed each next step and so steered the agents more than the owner wanted. With a record instead:

- nothing is held between commands outside the project, so compaction or a closed session loses nothing: `status` computes the list from the folder and the record alone;
- the tool imposes no order, reviewers can work in parallel, and any driver, a person included, can verify at any time and in any amount;
- the prompt stays a task and an answer format, as [DEC-051](register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process) asks, and the agents hold it in their own files;
- the hook guards only the record and the spec.

What is left to the agents is what needs judgment: how to build it, and whether the code meets the spec. What is left to the driver is who starts which agent, when, how often, and every Git step.

There is no container, no Docker image and no token. No command stays running while the agents work: the owner's session starts the builder, and the agents call `markitect` like any other tool. [§5](#5-what-build-does) says why this is the native version the owner asked for.

## 3. The tools

### Commands on the spec

None of these starts an agent or uses Git. `--repo DIR` names a folder of the project, whose root is the nearest folder with `.markitect/manifest.yaml` (default: the current folder), and `--json` prints JSON for scripts.

| Command | Does | Change from today |
|---|---|---|
| `install` | Writes the personal skill, once per machine. Optional: `init` says the same and gives a project its own skills ([§9](#9-the-harness)) | New |
| `init` | Sets up a project: `.markitect/`, the first page of the spec, the harness, `.markitect/.gitattributes`, and `AGENTS.md` and `CLAUDE.md` if they are missing. An empty folder is enough. Run again, it refreshes the harness and never touches spec pages | Writes the agents and the new skills. Since 0.20.0 it makes no repository and commits nothing. It writes no file that would hold nothing yet, and it ends by saying what Markitect is and pointing to `/markitect-design`. It sets `worktree.baseRef` in the project's settings when they name no base ([§9](#9-the-harness)) |
| `tidy` | Brings the lock in line with the manifest, as `go mod tidy` does. The lock holds a fingerprint of each Library, as `go.sum` does for modules; `check` only reads it, refuses a Library that no longer matches and says to run `tidy`. `init` and `update` keep it current | Was `lock` ([tooling plan](tooling-plan.md#refined-after-the-plan-was-merged)); `lock` is gone |
| `check` | Compiles the project and fails on any error ([below](#what-check-refuses)). `--spec` looks only at the spec itself | Stale entries and unused Definitions become errors, except in a project with nothing built yet. `--spec` is new |
| `list [--deps]` | The Packages with their Definitions. With `--deps`, what each Package depends on and the links behind it, with its coupling | Was `graph`, without the Mermaid diagram; `graph` is gone |
| `doc TARGET` | A Definition with its text, what it links to and what links to it, the files that realize it and its record; or a Package with its parts. `doc Root` shows Root | Was `refs`; `refs` is gone ([tooling plan](tooling-plan.md#the-commands)) |
| `vet [--strict]` | Measures the spec's design ([§4](#4-is-the-spec-good)). It warns, and exits 1 only with `--strict` | Was `lint`; `lint` is gone. An unused Definition and a stale row are errors of `check` now, so `vet` no longer reports them |
| `update` | Installs the newest release over the binary, then lets the new binary bring what an earlier build wrote in line with it: the personal skill if there is one, and in a project the harness and the lock, which it rewrites, and the record, which it writes from the newest certificate of 0.20.0; it commits nothing, and says to read the lock before committing it. `--check` changes nothing and exits 1 if anything is out of date; `--files` leaves the binary alone; `--force` replaces a development build | New, on the owner's word of 6 November 2026 ([§16](#16-questions-for-the-owner), [DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). It reuses what `init` and `tidy` write and replaces neither |
| `version` | The build, and whether the personal skill and the project match it | Adds the status lines |
| `licenses` | As named | None |

**Output and exit codes** follow Go's tools. A diagnostic is one line, `file:line: code: message`; `check` prints only diagnostics and one summary line, and `vet` only diagnostics. With `--json`, `check`, `vet`, `list`, `doc`, `status` and `mark` print one document that starts with `format_version` and `kind`. Exit 0 means all is well, 1 that the command found something, and 2 that it could not run; a note never changes the exit code. [The model language guide](../guides/model-language.md#diagnostics-json-and-exit-codes) owns the details.

The owner proposed `update` to replace `lock` (now `tidy`) and `init`'s refreshing. An earlier draft of this concept held it back, because the two commands already did that work. On 6 November 2026 the owner asked for it, with a `version` that says what is out of date, since an upgraded binary left the personal skill and each project's harness behind with nothing to say so. So `update` is thin: it runs `init`'s writers and `tidy` over what differs, compares files without regard to line endings, commits nothing, and never touches a spec file. Commands that read a project print one note when its skills, agents and hooks are old.

The same day the owner added releases and the binary's own upgrade, so that `update` does what `claude update` does ([DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). It reads the newest tag from the redirect of GitHub's releases page, downloads the binary for the system, checks it against the checksums file of that release and replaces the running executable: by renaming over it on Linux and macOS, and on Windows, which lets a program rename its own executable but not overwrite it, by moving the old file aside first. The new binary then finishes the update, since only it knows the files it ships. This is the only place where Markitect uses the network, and only when the owner runs `update`; without a network it says so and still refreshes the files. A development build older than the newest release (`0.0.0-dev`, or `0.15.0-dev` while `0.15.0` is out) is not replaced unless `--force` says so, so a build from source is safe from a stray `update`; one newer than the release (`0.16.0-dev` after `0.15.0`) counts as up to date. The download is checked against checksums from the same release over HTTPS, so it protects against a damaged or swapped file but not against a release published by someone who controls the repository; signed attestations would add that.

### What `check` refuses

`check` passes or fails, with no warnings: a project either compiles or it doesn't. Its errors come in two groups, and it prints them apart. One case passes: when the registry lists no file and the only errors are Definitions nothing realizes, it says that the spec compiles and nothing is built yet. `status` makes no such exception.

| Group | Errors | Who settles them |
|---|---|---|
| **The spec** | A Kind or link that does not resolve, a Package cycle, a Library that does not match the lock, and the rest of today's errors in the spec | The owner, in the spec |
| **The realization** | A file neither listed nor ignored, a registry entry for a Definition or file that no longer exists, and a Definition that nothing links to and no file realizes | The implementer, in the work |

So `check` goes red the moment the owner deletes a Definition or adds one nothing uses yet, and stays red until the work has realized the change. That is how it should read: spec and code don't match yet. `--spec` answers the narrower question "is the spec itself valid?", which is what a modeling session asks.

`markitect status` gives no list while the spec has errors. Errors in the realization are on its list, and it is not empty until none is left.

On finished Makerspace projects, at steps c5 and c8, there are no stale entries and one unused Definition, `documentation`, out of about 115. So a finished project can meet this rule. A rule that applies everywhere still needs a file that realizes it, or a link from something that does.

### How `status` works

Like `go test`, which does not run a test again when nothing it depends on changed, Markitect keeps for each Definition the state at which a reviewer last verified it ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache), [the inputs](../../internal/run/README.md#the-inputs)). That state is the set of everything the verification depended on, its **inputs**, each by its hash: the Definition's own text; its Kind and the Definitions it links to, a Library Definition by its Library's digest in the lock, so that a newer binary changes nothing; the files its registry rows name; and the files the reviewer gave as evidence. **What is to verify** is every Definition with a changed input or no record, computed from the folder and the record alone, with no session and no start.

| Distance | Reached | On the list |
|---|---|---|
| 0 | Definitions that changed, Definitions whose files or evidence changed, and Definitions a registry row gained or lost | Yes |
| 1 | Definitions that link to a changed one or are of a changed Kind | Yes |
| 2 and more | Everything further along the links | No |

These are the Definitions 0.20.0's checklist reached. A changed Rule above a Package reaches only itself and what links to it. A Library counts as one, so a change of one Kind of a committed Library lists the Definitions of all its Kinds: more, never less. Three more kinds of item stay on the list until they are verified again: a Definition recorded `fail` or `spec-problem`, a Definition gone from the spec, with the files that realized it, and an error in the realization.

It doesn't follow the code's own imports. A full review now and then, `/markitect-review --all`, is the safety net. On Makerspace's change steps, one step reached 33 of the 36 existing files that truly changed, and deeper reach 35 ([measurement](../validation/impact-reach-20261103.md)). How far every link reaches says something about the spec too ([§4](#4-is-the-spec-good)).

### Commands of the record

These are for whoever drives the agents, and for CI. They start no agent, keep nothing running, steer nothing and use no Git. [The record](../../internal/run/README.md) owns how they behave; this table only names them.

| Command | Called by | Does |
|---|---|---|
| `markitect status` | The builder, the reviewer, the skills, the owner and CI | What is to verify, with which inputs changed, where to look and each item's fingerprint. `--all` lists every Definition. Exit 1 when it lists something; notes, such as a `skip` or an outdated harness, never change the exit code |
| `markitect mark` | The reviewer | Records `ok`, `fail`, `skip` or `spec-problem` on one Definition at the fingerprint `status` printed, as in `markitect mark members/limit@3f9a1c ok -e src/loans.go:42`, with its evidence or why ([§6](#6-the-record)) |
| `markitect hook` | Hooks only | The check of [§9](#9-the-harness) |

For one release, `checklist`, `answer` and `gate` only say on standard output, with exit 0, that they are gone and to run `markitect update`, so that an old project's skill does not fail silently.

**The folder.** Every command reads the project folder by walking it, without what the project's `.gitignore` files exclude, read as plain text ([the folder](../../internal/run/README.md#the-folder)). So ignored build output does not count, and a folder needs no repository. Files that Markitect's own ignore file lists are walked like any other; they need no registry row and are no input.

**The record** is `.markitect/review.sum`, committed with the project ([§6](#6-the-record)). With no record yet, every Definition is on the list. A file changed beside the work, by hand or by a commit made around the agents, changes the inputs it touches, so those Definitions are on the list again. Nothing of the work lives outside the project: the tool keeps no state between commands, only a short lock in the user's cache folder while `mark` writes.

**There are no required checks.** The owner took them out on 4 November 2026, to see whether they are needed. A project that wants tests states that in the spec, with the command that runs them, and the reviewer checks it and may run them. So no proof in the work comes from outside an agent any more, apart from `check` and the record's inputs, and that is what this trial tests. If a Definition is verified that a test would have caught, the checks come back. `execution.yaml` goes with them, and the agents' `model` and `effort` move to their agent files and the manifest.

**The ignore file** is `.markitect/ignore`, written like a `.gitignore`: one pattern per line and `#` for a comment. Its patterns stay the registry's, and it has no `!`. A reason is a comment now and no longer required, which is safe because the hook keeps subagents out of `.markitect/`, the ignore file included. The tool accounts for what `init` installs by itself, so a new project has no ignore file.

## 4. Is the spec good?

A spec is judged like code ([founding intent](sources/founding-intent-20261026.md)), so it gets the same kind of analysis code gets. `check` is the compiler: is it valid? `vet`, named after Go's, is the static analyzer: is it well designed? It only ever warns, since granularity stays the author's choice ([DEC-005](register.md#dec-005-model-granularity-is-the-authors-responsibility)), and its thresholds are provisional.

| Measure | Code analogy | Warns about | State |
|---|---|---|---|
| Efferent and afferent coupling, instability, per Package | Clean Architecture | A Package depending on too many others, or on a less stable one | Exists in `vet` |
| Size of a Definition and a Package | Long functions, large modules | God Definitions | Exists in `vet` |
| Packages with no paths, or no Definitions and one child | Empty namespaces | Structure that holds nothing | Exists in `vet` |
| Fan-in of a Definition, including at Root | God class, `utils` file, interface segregation | One page that too many depend on | New. `vet.reach` skips Root today, so it misses the worst case below |
| Blast radius: how far every link reaches from a Definition, as median, 90th percentile and worst | Transitive dependents | A change that could touch most of the spec | New |
| Links inside a Package against links leaving it | Cohesion | A Package whose parts belong elsewhere | New |
| Change hotspots across records | Churn | What keeps changing, with how far it reaches | Later. The tool reads no Git history ([DEC-097](register.md#dec-097-markitect-uses-no-git)), so it would count changes from one record to the next |
| Size against what tests cover | CRAP | Large, unverified Definitions | Later: needs tests bound to Definitions ([ENH-003](register.md#enh-003-test-to-statement-binding-and-test-review)) |
| The spec's dependencies against the code's imports | Architecture tests | Code that has drifted from the design | Later |

**Package cycles stay a compile error**, as in Go. The details of `vet`, its further measures and thresholds, are still to be discussed with the owner; [the model language guide](../guides/model-language.md#vet-warnings) lists what it warns about today.

**What this finds on Makerspace, measured on 4 November.** Following every link reached over 85 % of the spec on four of its eight change steps. That looked like a ball of mud, and at the Package level it is not one: there are no Package cycles, dependencies point towards a stable core, and a typical Definition reaches 8 of 113. But it does show a real design smell:

| | Makerspace, 113 Definitions | 5× project, 413 Definitions |
|---|---|---|
| Direct links to one page, `values` | 60 | 268 |
| Definitions that page can reach through every link | 97 | 392 |
| Definitions reaching over half the spec | 9, all at Root | 17, of which 15 at Root |
| Worst reach if the top three hubs pass nothing on | 46 | 188 |

`values` is one page of thirteen unrelated value formats, and most links to it name no format. A Definition that uses only dates still depends, on paper, on the format of money. Treating the top three hubs as stopping points brings the four steps that reached 85 to 88 % down to between 9 and 38 %. In code this would be a `utils` file, and the remedy would be to split it. Whether splitting it really narrows reach, and whether that helps agents, is untested: cutting hubs in a calculation is not the same as a better spec.

**What analysis can't say:** whether the spec is right. No threshold has been checked against outcomes yet, so a clean report doesn't show that a spec leads to better work.

## 5. What build does

The owner models until the spec looks right, then uses `/markitect-build` in their own Claude Code session. The tool computes, checks and records; the skill, the agents and the session do the rest, and nothing steers them ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)). [The project workflow](../guides/project-workflow.md#7-build-and-review) says how to use it, and [the record](../../internal/run/README.md) owns the rules.

1. **Start the builder.** The skill is a plain prompt with no injected command: the owner's session starts the `markitect-builder` agent in a worktree of its own, through Claude Code's worktree isolation ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). The builder is a subagent: it starts with fresh context, on the last commit of the owner's branch, and holds its task in its agent file. The worktree holds no uncommitted change, so the skill asks the owner to commit first.
2. **See what is to verify.** The builder runs `markitect status`: every Definition with a changed input or no record yet, with what changed and where to look, the Definitions recorded `fail` or `spec-problem`, the Definitions gone from the spec, and the errors in the realization. An error in the spec gives no list, and the builder says so.
3. **Implement.** The builder makes the project match the spec for what the list holds and keeps the registry current. It does not change the spec, and the hook refuses its edits under `.markitect/` except the registry pages. A gap with no effect at the project's boundary is its own to decide. Where the spec contradicts itself or leaves open behaviour at the boundary, it builds nothing for that Definition and tells the reviewer what it found.
4. **Review.** The builder starts the `markitect-reviewer` agent as its own subagent, the only agent the hook lets it start, and waits, so it keeps its context. The reviewer works in the builder's worktree, starts with fresh context, has no tool to edit files and starts no further agents. It runs `status`, reads each Definition, the Definitions it links to and the code, and records a verdict with `markitect mark DEFINITION@FINGERPRINT VERDICT -e EVIDENCE -m WHY`: `ok`, `fail`, `skip` or `spec-problem` ([§6](#6-the-record)). It judges a reported spec problem on its own reading.
5. **Repeat.** The builder runs `status` again and works on what it lists, until it lists nothing or only `spec-problem`s for the owner. Each verdict is bound to the inputs the reviewer read, so a repair puts back on the list only what it touched; other verdicts stand. There is no limit on rounds, but the builder stops when three rounds in a row end without the list getting shorter. What the builder notices beside its task it tells the reviewer, which records it as `fail` or `spec-problem`: the record is the one store of findings.
6. **Bring it back.** When the builder has ended, the owner's session runs `status` in the worktree and commits the work there together with `.markitect/review.sum` in one commit on its branch. When `status` is empty there, it merges that branch into the owner's branch and removes the worktree; otherwise it asks the owner whether to merge the work as it stands. It pushes nothing, and reports in a line or two what is left. In a folder that is not a Git repository the builder works in the folder itself, and the session commits nothing and says the work is ready.

**Done** is an empty `status`: the project compiles and every Definition is verified at its current inputs, whatever happened in between. A file edited by hand, a commit made around the agents or a change on a merged branch changes the inputs it touches, and those Definitions are on the list again.

**Git is the driver's.** Markitect makes no repository, commits nothing, keeps no branches and lands nothing ([DEC-097](register.md#dec-097-markitect-uses-no-git)). The skill has the owner's session commit in the builder's worktree and merge its branch into the branch that is checked out. A branch that moved, a merge conflict or a pull request is ordinary Git, by the session or the owner, and the next `status` judges the folder that results.

**The worktree is the work's.** The builder and its reviewer work in a worktree of their own, so the owner can go on designing in their folder while the session that started the work waits for it. Claude Code branches such a worktree from the repository's default branch unless `worktree.baseRef` is `"head"`, which `init` sets ([§9](#9-the-harness)). What the owner changes meanwhile meets the work at the merge: a changed spec puts its Definitions back on the list of the owner's folder, and a conflict stops the merge and is the owner's to settle. A change to the spec is the owner's own step, in their own session, which is no subagent.

**When the builder ends early.** The list is computed from the folder and the record alone, so nothing is lost: the work is committed on the worktree's branch, and `/markitect-build` started again, or `/markitect-review`, takes up what `status` lists. Nothing of the work lives outside the project.

**Why not a background session.** The first build of this concept had `apply` start `claude --bg` in a worktree of its own, with generated settings and hooks, and it landed real changes ([record](../validation/native-version-20261104.md)). On 5 November 2026 the owner chose agents started by their own session instead: Markitect should be just a way of agentic coding, without that machinery. What the choice costs:

- the owner's session is busy until the work ends, and the agents stop if that session closes;
- the agents work under that session's permission mode, settings, plugins and memory, so they may ask for permission, and they differ from one person to the next;
- an agent that ends early is started again by the owner, not sent back by a hook.

What it keeps: fresh context for the builder and for the reviewer, which is the test that the spec alone is enough. The worktree of [DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role) gives back the first picture's separate folder without a background session: Claude Code makes and keeps it, and Markitect writes only the setting that makes it start from the owner's branch.

## 6. The record

`.markitect/review.sum` is the record of what was verified, committed with the project like `go.sum` ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache), [the record](../../internal/run/README.md#the-record-file)). It holds one line per Definition, sorted, each a JSON object: the verdict, the hash of each input ([§3](#how-status-works)), the evidence the reviewer gave, and why. It replaces 0.20.0's certificate, the work under way in the cache, `gate`, the stored answers, the Result and Findings pages and the builder's reports.

| Verdict | Means |
|---|---|
| `ok` | The Definition is implemented, as the evidence shows |
| `fail` | It is not; the line says why |
| `skip` | Verified as far as the project can show, with the reason. A note, which never changes `status`'s exit code |
| `spec-problem` | The spec contradicts itself or leaves open behaviour at the project's boundary, and only the owner can settle it |

**A verdict is bound to what the reviewer read.** `status` prints each item as `DEFINITION@FINGERPRINT`, the fingerprint being the start of the hash of every input but the evidence, and `mark` refuses when the inputs changed since: a builder editing a file beside the reviewer, or the reviewer's own test run changing one. The evidence, `path` or `path:line`, must be files of the folder, and their hashes become inputs when the verdict is recorded, so a weakened test that served as evidence puts its Definition back on the list. Go does the same: it does not cache a result whose inputs changed while the test ran.

**How it is written.** `mark` writes under a short lock in the user's cache folder, as one whole file, so parallel reviewers, for example one per Package, do not lose lines. A gone Definition stays on the list with the files that realized it until a reviewer finds them removed or with a new basis and records `ok`, which removes its line.

**How it merges.** Git also conflicts on changes to neighbouring lines, so `init` writes `.markitect/.gitattributes` with `merge=union` for the record, and Git keeps both sides. Of two lines for one Definition, `status` takes the one whose inputs match the folder; if both match with different verdicts, it lists the Definition as a conflict to verify. The next `mark` writes the file clean.

**What it is:** a record, not a proof, as the certificate was. A line written by hand shows in the diff, and the hook keeps agents from writing one ([§9](#9-the-harness)). It holds hashes, not text: what a changed Definition said before comes from Git.

**What remains of `gate`.** `check` still fails on a project that does not compile, and `status` lists the realization's errors and gives no list for a spec with errors. So an empty `status` means two things: the project compiles, and every Definition is verified at its current inputs. A CI job runs `markitect status`, and any item fails the build. The old projects' certificates are not lost: `markitect update` writes the record from the newest one ([the migration](../../internal/run/README.md#the-migration)).

## 7. How sure we are

The tool can't stop a determined agent that runs as the user on the user's machine. This concept guards against forgetting and shortcuts, which is what has been seen, and against deliberate bypass only by making it visible.

| What the owner wants | What secures it | How firm |
|---|---|---|
| The spec compiles before it is verified | `status` gives no list and `mark` records nothing while the spec has errors | Firm |
| Every Definition is verified at the state it is in | The record's inputs: a change of the Definition, its Kind or links, its files or its evidence puts it back on the list, and `mark` binds a verdict to the fingerprint the reviewer read | Firm for one step of links and the registry; deeper reach and the code's imports are not followed ([§3](#how-status-works)) |
| What is committed is what was verified | The session commits the work with the record; the tool commits nothing ([DEC-097](register.md#dec-097-markitect-uses-no-git)) | Seen, not prevented: `status` on any checkout lists what differs from the record, and a CI check can refuse it |
| The work doesn't stop to wait for someone | Nothing: the agents work under the owner's session's permission mode. The builder file's `permissionMode: acceptEdits` did not apply in the trials; in the session's `auto` mode five of six runs had no denial ([trials](../validation/lean-trials-20261107.md#what-the-flow-showed)) | Weak by choice: the owner is there to answer. How often manual mode asks is not known |
| The work doesn't mix with the owner's own | The builder works in a worktree of its own, from the last commit of the owner's branch, and its work comes back by a merge ([§5](#5-what-build-does)) | Good: Claude Code keeps the worktree's edits out of the owner's folder, and a conflict stops the merge. A worktree holds no uncommitted change, so what the owner did not commit is not built. In a folder without Git the work shares the owner's folder |
| The spec doesn't change during the work | The hook refuses a write under `.markitect/` by any subagent but the architect, except the registry pages; only the owner's session starts the architect | Good for subagents, not proof: it reads the tool call, and a shell command can still write. Not prevented for the owner's session or for people: a changed Definition goes back on the list, and the change shows in the diff |
| The work doesn't just stop | The builder works until `status` lists nothing or only what needs the owner, or the list has not got shorter for three rounds | Prompt text, not structure. An early end verifies nothing, so it costs time, not correctness |
| The agent doesn't review itself | A hook in the project's settings lets only the reviewer run `markitect mark`; it runs in subagents too. The hook lets the builder start only the reviewer, since Claude Code ignores the agent types a subagent's own tools list | Good, not proof: the hook reads the command's text and the agent's name |
| The reviewer judges for itself | Its agent file says to take the list from `markitect status` and to treat the message that started it as claims to check | Weak: the builder writes that message. Reviews given a maker's claims did no worse on planted bugs ([17 of 18 blind, 18 of 18 with claims](../validation/k-checker-test-20261103.md)), though that tested claims beside bugs, not a message written to steer |
| Nothing goes to a remote | The tool has no Git, and the skills say to push nothing. A routine the owner set up lands by a push when its source branch tracks a remote, and only what the remote already had and its own work go out ([§12](#12-schedules-and-continuous-work)) | Catches ordinary use only: the owner's own settings decide what a shell command may do |
| A change made around the agents is noticed | It changes the inputs it touches, so `status` lists those Definitions | Seen afterwards, not prevented |
| The reviewer really read the code | Evidence checked for existence in the folder, and hashed | Not provable. It was not provable in the Docker version either |
| The code is right | The reviewer and the project's tests | Not guaranteed by any version |
| The work has a limit | No limit on rounds, by the owner's decision ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)); the builder stops when three rounds in a row end without the list getting shorter ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)), and the owner can stop the work | Prompt text: a builder that keeps the list shrinking slowly goes on. Unbounded in time and calls. The provider's own budget and turn flags work only with `-p`, and there is no spending limit |

**Anyone who wants more** can run `markitect status`, `markitect check` and the project's tests where the agent can't reach them: on a server of their own, in CI or in a hook on a remote. Markitect depends on none of these. The record committed with the work carries the verdicts and their evidence.

**What we give up against concept 3.0:**

- prevention of a deliberate bypass;
- a commit made by the tool: the driver commits, so what is committed matches what was verified only if the driver does it right, and a mismatch shows afterwards;
- isolation from the user's machine: agents run with the user's rights, so the work must not read untrusted input;
- one operating system: agents work on the user's own, Windows included, with its paths and shells;
- a reviewer in its own session as another user, with a throwaway folder. It now shares the owner's folder, and code it runs can change files; `mark` then refuses its verdict;
- review of a clean copy: it reads the folder as it stands, ignored leftovers included;
- a reviewer's task written by the Command, and a measure of how much it checked.

**What we give up against 0.20.0** ([tooling plan](tooling-plan.md#what-this-gives-up-and-what-it-keeps)): the refusal of a changed file the ignore file lists, of a changed harness, `.gitignore` or `settings.json`, and of a symbolic link that leaves the folder during the work; the spec freeze for the owner's session and for people; the old text of a changed Definition in the tool's output; and a limit on the work.

## 8. Other Commands

A Command is a skill over the same tools: which agent the owner's session starts, on what task, and what it verifies. Skills are named after the activity and agents after the role ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role), [tooling plan](tooling-plan.md#skills-roles-and-routines)). The table in [concept 3.0 §12](v3-concept.md#12-other-commands) stands for what each Command is for. The tool's commands run anywhere, from a chat, a terminal or a schedule; the agents need a driver, which today is the skills in a Claude Code session.

| Skill | Started by | Does |
|---|---|---|
| `markitect-design` | The owner or Claude | Designs the spec with the owner, in their own session. It sees whether the folder is an empty project, code without a spec or an existing spec, and starts the spec one question at a time, draws it from the code, or changes it. A large draft, such as a first spec drawn from a big code base, it can hand to the `markitect-architect` agent, which hands its questions back. It replaces `markitect-start`, `markitect-extract` and `markitect-modeling` |
| `/markitect-build` | The owner | [§5](#5-what-build-does) |
| `/markitect-review` | The owner or Claude | The owner's session starts the `markitect-reviewer` agent, which verifies what `markitect status` lists, or with `--all` every Definition, records each verdict and reports what failed and what is a `spec-problem`. Above about 30 items it starts one reviewer per Package, in parallel and in stages if need be; the record loses no line ([§6](#6-the-record)). The session commits the record. `/markitect-build` then repairs what failed |
| `/markitect-cleanup` | The owner | The builder, in a worktree of its own, refactors without changing behaviour, and the reviewer verifies again what that put back on the list. The work comes back when it leaves nothing worse |
| `/markitect-bughunt` | The owner | The builder, in a worktree of its own, has the reviewer hunt beyond the list and record each bug as `fail`, or as `spec-problem` where the spec does not cover it; it repairs, and the reviewer verifies again. The work comes back when it leaves nothing worse |
| `/markitect-audit` | The owner or Claude | The `markitect-auditor` agent reads the spec fresh for contradictions, gaps, unclear and duplicated statements, and reports to the owner. Nothing is recorded or committed |
| `/markitect-schedule` | The owner | Sets up cleanup, bughunt and `review --all` as routines ([§12](#12-schedules-and-continuous-work)) |

Only the owner starts the skills that change code or set up routines: their files set `disable-model-invocation`. What any agent notices goes into the record as `fail` or `spec-problem` and nowhere else. A project that wants tests states them in its spec as an ordinary Rule; the reviewer runs them as evidence, and a routine finds them in the spec, the README or the CI configuration. `security` and `improve` may follow.

## 9. The harness

For Claude Code, `markitect init` writes these project files, so they are versioned with the project, and the tool accounts for them itself.

| File | Holds |
|---|---|
| `.claude/skills/markitect-design/` | The design skill, with `reference.md` and `examples.md`. Also written to `.agents/skills/`, for other agents |
| `.claude/skills/markitect-build/SKILL.md`, `markitect-cleanup`, `markitect-bughunt` | Plain prompts: start the `markitect-builder` agent in a worktree of its own on its task, and when it has ended commit the work there and merge it back ([§5](#5-what-build-does)). They inject no command, so no exit code can abort them. Their front matter allows `markitect status` without a prompt and sets `disable-model-invocation: true` |
| `.claude/skills/markitect-review/SKILL.md`, `markitect-audit` | Start the `markitect-reviewer` agent, one per Package on a long list, or the `markitect-auditor` agent. Claude may start them |
| `.claude/skills/markitect-schedule/` | The routine setup, with the routine prompts: a frame, `routine.md`, and the work of each routine ([§12](#12-schedules-and-continuous-work)) |
| `.claude/agents/markitect-builder.md` | The builder's three tasks, build, clean up and hunt bugs, and how it works: run `status`, make the project match, keep the registry, start the reviewer, repeat until `status` lists nothing or only what needs the owner, or until three rounds left the list no shorter. Tools Read, Edit, Write, Glob, Grep, Bash and `Agent(markitect-reviewer)`; model `sonnet`, effort `high` |
| `.claude/agents/markitect-reviewer.md` | The reviewer's two tasks, verify and hunt bugs, and the verdict words: run `status`, verify each item, record it with `markitect mark`. Tools without edit or agent tools |
| `.claude/agents/markitect-architect.md` | The architect's task: draft the spec as the design skill says, write only under `.markitect/`, and hand every question back. No agent tool |
| `.claude/agents/markitect-auditor.md` | The auditor's task: read the spec fresh and report what is wrong with it. Tools without edit or agent tools |
| `.claude/settings.json` | The hooks, below, and `worktree.baseRef: "head"` unless the settings name a base, added to whatever settings the project already has |
| `.markitect/.gitattributes` | `review.sum merge=union`, so a merge keeps the lines of both sides |
| `AGENTS.md`, `CLAUDE.md` | What kind of project this is, for any session: the spec is the authority, changes start there, and `markitect` is on the path. `CLAUDE.md` holds only `@AGENTS.md`, so that it works in every session of the provider. They hold no process, since the agents read them too. Written only if missing |

The agents hold their tasks and the verdict words in their own files, which outlive compaction; a skill only names the task.

**The agents' settings** come from their files. The manifest's `model`, if it names one, is written into every agent file by `init` and `update`. Everything else is the owner's session: its permission mode, settings, plugins and memory. The builder file's `permissionMode: acceptEdits` did not apply in the trials, where the agents ran in the session's mode ([trials](../validation/lean-trials-20261107.md#what-the-flow-showed)). No settings are generated per piece of work.

**The worktree.** Claude Code branches a subagent's worktree from the repository's default branch unless `worktree.baseRef` is `"head"`, so `init` and `update` set it when the project's settings name no base; a base the owner chose stays. Claude Code keeps its worktrees under `.claude/worktrees/`, and the tool accounts for that folder itself, so that a builder's worktree is no unregistered file of the owner's folder. A reviewer the builder starts works in the builder's worktree, as a probe showed on 7 November 2026.

**The hook** stays, as a guard of the record and the spec, not of the process. `markitect hook pretool`, in the project's settings, runs before a shell command that calls `markitect mark`, before every tool that writes a file, and before every tool that starts an agent. Hooks in project settings run inside subagents too, and their input names the agent. It holds the four roles' rights, a 2×2 of who writes and who judges the code and the spec ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)), for subagents only:

| Subagent | Writes | Records | Starts |
|---|---|---|---|
| `markitect-builder` | Code and the registry pages | No | Only `markitect-reviewer` |
| `markitect-reviewer` | Nothing | Yes, the only one | Nothing |
| `markitect-architect` | The spec, the registry pages and the ignore file | No | Nothing |
| `markitect-auditor` | Nothing | No | Nothing |
| Any other | As the builder | No | Anything but the builder and the architect |

- **Only the reviewer agent may run `markitect mark`.** So the agent that built does not verify its own work. In the trials, a separate reviewer caught 17 of 18 planted bugs ([checker test](../validation/k-checker-test-20261103.md)). The owner's session does not record either.
- **No subagent writes `.markitect/review.sum`,** which only `mark` writes, and none but the architect writes the rest of `.markitect/` except the registry pages. So a builder cannot weaken a Definition during the work or write lines into the record. The owner's own session, where the spec is designed, is not a subagent and stays free.
- **The builder starts only the reviewer, and only the owner's session starts the builder and the architect.** Claude Code ignores the agent types listed in a subagent's own tools, so the hook holds this, and a general-purpose helper cannot get past the rules.

The hook reads tool calls, so a command hidden in a script, or one that names `markitect` by a path or another spelling, still gets through. It guards against mistakes, not against intent.

**One file lies outside the project.** A session that runs `init` started before the project's files existed, so it knows only what the tool prints, and Claude Code doesn't watch a skills folder that wasn't there when the session started. So help and `init` say what Markitect is, and `markitect install` writes a personal skill, `~/.claude/skills/markitect/SKILL.md`, with the same text from the binary, which reaches a session in any folder. It allows no command in advance. `init` never writes to the home folder; it only tips when the skill is missing or old. `markitect update` installs the newest release and refreshes the skill and the project's files, and `markitect version` says whether they match. The skills and agents call `markitect` by name, so `version`, `init`, `install` and `update` warn when the `PATH` finds no `markitect` or another one.

**The plugin does both.** `plugin/` is a Claude Code plugin, `markitect`, in this repository. The marketplace that lists it, `glacius-labs`, is the small repository [Claude-Plugins](https://github.com/Glacius-Labs/Claude-Plugins), whose entry pins the plugin to the tag of a release: the plugin that a user gets is the one at the tagged commit, whatever `main` holds. It carries the personal skill as `markitect:getting-started` (the same text, pinned by a test, under a name that does not repeat the plugin's, since Claude Code names a plugin's skill `plugin:skill`), a launcher, `bin/markitect`, which Claude Code puts on the `PATH` of its shell, and four hooks. The launcher reads the plugin's version, downloads the binary of the release with that version into the plugin's data folder, checks it against the release's checksums and runs it, with `MARKITECT_PLUGIN` set so that the binary knows who started it: `update` then leaves the binary to the plugin (`claude plugin update markitect@glacius-labs`), and the `PATH` warning stays quiet, since the launcher is what the `PATH` leads to. A session-start hook has the launcher fetch the binary in the background and outside the shell's sandbox, which may block the network, but only in a project that has a `.markitect/` folder: in any other project the plugin stays out of the way, and the binary is fetched at the first call. The other three run `markitect hook pretool` as the project's hooks do, because the project's hook runs in a shell that does not see the plugin's `bin/` and would find no `markitect` there: one before a shell command that contains `markitect mark`, one before a subagent's tool call that writes and names `.markitect` or comes from a role that may not write code, and one before a subagent's start of an agent that names a Markitect agent. Each looks at the call itself and starts the launcher, and so a download, only then; the `if` filter of a hook alone was seen to start it for other commands too, in 0.17.0. After `init` both the project's and the plugin's hooks run, and both give the same answer; if the launcher fails, the plugin's hook exits 2 and blocks the call. The plugin does not replace `init`: the project's skills, agents and hooks still come from `markitect init`.

Skills are kept short, and they hold no steps the agents need to remember, so nothing depends on what survives compaction. The harness's agents stay in the project rather than in a plugin: the provider ignores hooks and permission modes in plugin agents. The plugin described above brings only the command, the personal skill and its hooks.

## 10. Drivers

What earlier drafts called a strategy is a driver ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)): whatever starts the agents, decides how often and how long, and makes the Git steps. **The default driver is a folder of harness files and nothing else.** There is no code interface to implement. Adding a version of it means adding a folder, and removing one means deleting it. **It may change** the prompt, agents, skills, settings, limits and every Git step. **It may not change** the tools, the record's inputs or what the hook refuses.

The first version ships one driver and no switch for it. The natural second is a saved provider workflow that holds the loop and the limits in code, as in the trial of [§14](#14-what-is-not-yet-shown), added only if the first one fails too often ([§16](#16-questions-for-the-owner)). A person can drive the work too, with the same commands.

## 11. Providers

The first version is Claude only. A provider needs subagents with their own agent files, and a hook before a tool call that names the agent. Codex gets its own driver once Claude's version is reliable.

No token is needed: sessions run on the user's own stored login in the unmodified program, which removes token passing. Each session uses the user's subscription quota. The legal question of [concept 3.0 §14, question 10](v3-concept.md#14-decided-on-acceptance), whether other users may run Markitect on their own login, is moot and closed ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)): Markitect starts no session and touches no credentials, so each user simply runs Claude Code on their own login, with Markitect as a tool it calls.

**The app.** The owner said "we can remove desktop for now". This concept reads that as the whole app, with its spec views and chats. Its chats are the only other use of containers, so this takes Docker out of Markitect altogether, along with the provider adapters, the session helper, the runtime image and the token. It stays recoverable in Git and can come back once the tool has settled. The owner confirmed that reading on 6 November 2026, and the app is gone ([DEC-094](register.md#dec-094-main-carries-the-native-version-and-the-app-goes)).

## 12. Schedules and continuous work

[DEC-084](register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps) stands: the loop runs outside Markitect, as a scheduled job that calls `markitect`, and there is no daemon in the product. The tool sets no budget and counts no rounds ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)); a budget per task or per day belongs to the job and the driver. Government stays a later step ([IDEA-002](register.md#idea-002-government-delegated-model-maintenance)): its mandate would live in the spec, and a fresh reviewer would accept proposals within it.

The cheapest scheduled check needs no agent: `markitect status` in CI or a job fails on any item, so a change made around the agents, a merge or a hand edit is seen.

**Routines** act on it ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role), [tooling plan](tooling-plan.md#routines)). The harness ships prompts for three: cleanup, bughunt and `review --all`. `/markitect-schedule` sets them up with the owner through Claude Code's own scheduling, a scheduled task of the desktop app on the owner's machine or a cloud routine on a fresh clone; Markitect schedules nothing itself.

- **A branch of its own.** A run works in a worktree or clone of its own, from the newest commit of the source branch, the remote's when the branch tracks one, so that nothing unpushed goes out.
- **It lands only when it leaves nothing worse:** `status` lists no Definition that it does not also list on the source branch as it now stands, and the project's tests pass. If the source branch moved, the run merges it first and has the reviewer verify again what the merge put on the list. It lands as a fast-forward, by a push when the source branch tracks a remote, otherwise by moving the local branch, which Git refuses while it is checked out in a folder.
- **Otherwise it leaves its branch** and says why.
- **Each run ends with a short note:** what landed, what is open, and which spec-problems wait for the owner.

So a review that records a new `fail` does not land: its branch holds the record, and its note names the item. Two limits stay:

- **The machine must be on and signed in.** A server without a stored login is not covered yet.
- **External signals need isolation.** Issue text can hold instructions aimed at the agent, and a native agent has the user's rights. Work that reads outside input needs a container or a machine with limited rights around it. This wraps the harness and needs no engine.

## 13. Order of work

| Step | Result | Done when |
|---|---|---|
| 1. Probe | The harness by hand on the Library, on Windows, under a bounded dispatch. It tries what [§14](#14-what-is-not-yet-shown) lists | Done on Windows in throwaway projects: real sessions tried the background version ([record](../validation/native-version-20261104.md)), and the lean trials tried agents started by the owner's session ([record](../validation/lean-trials-20261107.md)) |
| 2. Build | Built first as a Run with `apply`, `submit` and `stop`, released up to 0.19.0; then made lean by the [lean plan](lean-tool-plan.md#order-of-work) and released as 0.20.0: `checklist`, `answer`, `gate`, `status`, `hook` and the stricter `check`, with no Git. `init` writes the harness. The engine's container and session code is gone. Then made a record of what was verified by the [tooling plan](tooling-plan.md#order-of-work) and released as 0.21.0: `status`, `mark`, `.markitect/review.sum`, the hook on the record and the spec, and the build and review skills as plain prompts. Then the tooling plan's step 2 and DEC-100, released as 0.22.0: the diagnostics and the JSON envelope, the exit codes, and `tidy`, `vet`, `list` and `doc` in place of `lock`, `lint`, `graph` and `refs`; the skills named after the activity, the four roles held by the hook, the builder in a worktree of its own, and the routines | Done for 0.20.0: three Makerspace changes were certified or stopped as expected, with a planted bug and a planted contradiction caught, before the Run was removed. 0.21.0 and 0.22.0 are covered by tests; the tooling plan's step 3 tries them with planted defects under a bounded dispatch |
| 3. Make it reliable | The 5× project's 8 change steps on Sonnet, with their hidden cases, which are at an [archive point](../archive.md#removed-with-the-evaluation-tools) since the evaluation tools left Main. Failures are fixed in the tool, not with more prompt text | Two clean passes in a row: every step ends with nothing left to verify without the owner, all hidden cases pass and no refusal was overridden ([§16](#16-questions-for-the-owner)) |
| 4. The rest | `security` and `improve`, `vet`'s further measures and the rename; build, review, cleanup, bughunt, audit and the routines exist as skills since 0.22.0. From the lean plan: a workflow driver, and a CI check with `markitect status` | The owner's reading |

Before step 3, read test 1's nine finished launches. They are in `%LOCALAPPDATA%\markitect-test1`, not in the repository, and their answers are unread. They may say at no new cost whether one reviewer catches known bugs at 5×.

Concept 3.0's tests 2 and 3 fall away, since the aim is no longer to show that this beats something else.

## 14. What is not yet shown

**The background version was tried on Windows on small changes ([record](../validation/native-version-20261104.md)).** That record covers the launcher, settings and hooks the owner has since dropped. **Agents started by the owner's own session were tried on 7 November 2026**, on three small Makerspace changes, one run per case, in `auto` mode and headless, on 0.19.0 with the lean commands ([trials](../validation/lean-trials-20261107.md)). What a first run had to show:

- **The skill's printed step makes the owner's session start the builder, and start an agent again when one ended early: shown.** The session ran only `gate`, `git status`, the commit and one read-only look, and where the builder ended without passing `gate`, it ran `gate` and started the agent it named.
- **The builder, as a subagent, starts the reviewer as its own subagent and waits for it: shown.** Four nested reviews in the three lean runs, seven in five runs with the old flow, each at spawn depth 2 with the builder as its parent and a foreground start; lean A's two came around one repair. Nothing had to be configured.
- **The hook in the project's settings fires inside a subagent: shown.** In a probe it refused a subagent's `checklist --full`. No agent called `answer` without being the reviewer, so that refusal was not met in a run.
- **The certificate travels with the work: shown.** Each certified run committed the work, the certificate and its Result in one commit, and `gate --verify` then found the folder matching it. A builder that changed a file's line endings left the answers standing, since a text file's hash ignores them. Landing went with the Run.
- **How often the owner's permission mode stops a builder to ask: open.** In `auto` mode five of six runs had no denial, and in the sixth the classifier twice refused a builder's Git undo of its own edits. Manual mode was not tried.

**Not yet shown in a run:** 0.20.0 itself, since the trials ran 0.19.0, whose `check` still read files through Git, so the Git-free walk is tried by tests only; 0.21.0's record, its hook on writes under `.markitect/` and its plain skills, built after the trials and covered by tests only, which the tooling plan's step 3 tries with a weakened evidence test and a builder's edit of the spec; 0.22.0's skills and roles, the builder in a worktree of its own and its merge back, a bug hunt, a cleanup, an audit and the routines, of which only a probe showed that a reviewer started by a builder in a worktree works in that worktree; a real branch, a merge, a crash or parallel reviewers; a change larger than about 64 items.

**What the evidence does and doesn't cover:**

- **A plain Sonnet session has done the work from the typed spec, once.** Concept 3.0's default built the 5× project in one launch, with one implementer session and one repair round, and passed all 327 hidden cases ([record](../validation/default-arm-20261104.md)). A reviewer found a real bug there. That is one launch and not a rate, in a container, driven by the Command, and its change steps have not run. So the implementation doesn't look like the main risk; the process around it does.
- **How often a native agent skips steps is unmeasured.** The owner has seen one forget things. With the record, a skipped step should show as an item still on the list, not as a wrong verdict. Whether that keeps the work from ending in nothing is what step 3 finds out.
- **The workflow trial is weak evidence.** On 4 November a saved script fixed the order, run through `claude -p` in `bypassPermissions`, with a throwaway check on a local repository. Of four attempts, two ended early: a refused worktree, and a parser that stopped on a helper's extra sentence. In the third the planted bug became a build error, so the reviewer never met it. The fourth started at review, on the second attempt's work plus the plant: the reviewer caught it once, and it landed. It used no skills and no background session. Its evidence is in `C:\Users\Consiliari\wf-trial\evidence`, outside the repository.
- **Reviewing at 5× has no read result.** Test 1 stopped when the machine went down, with nine of twelve launches run.

## 15. What this overturns

| Entry | Becomes | What stays from it |
|---|---|---|
| [DEC-091](register.md#dec-091-concept-30-is-the-target) | Superseded as the target on 7 November 2026 ([DEC-098](register.md#dec-098-concept-40-is-the-target)) | Concept 3.0 as the record; its six problems, words, checklist, answers and Result |
| [DEC-037](register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers) | Superseded: agents and checks no longer run in containers | A reviewer gets no write tools |
| [DEC-013](register.md#dec-013-linux-first-for-tests-and-the-playground), [DEC-077](register.md#dec-077-development-pace-and-linux-first-execution) | Amended: agents work on the user's own system, so Windows becomes a main path | Linux tests for Markitect's own code |
| [DEC-078](register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) | No remote and no Git in the tool ([DEC-097](register.md#dec-097-markitect-uses-no-git)): the owner's session commits the work with its certificate on the current branch, and the owner pushes | Work counts without the owner reviewing a diff; no Git host is needed |
| [DEC-080](register.md#dec-080-the-model-is-typed-documentation) | Amended: "Definitions nothing links to and no file realizes" move from warnings of `lint` to errors of `check`, and the command list changes to `check`, `tidy`, `doc`, `list`, `vet` (named `lock`, `refs`, `impact` and `analyze` when this row was written) | The language, and everything else measured staying a warning |
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

[DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache) amends this page in turn: `gate`, its rows, the certificate and the work state give way to the record of what was verified in §3, §5 to §9 and §12, and the [tooling plan](tooling-plan.md#what-this-gives-up-and-what-it-keeps) says what that gives up. The rows above that name `gate` or a certificate describe 0.20.0.

**Stays unchanged:** [DEC-051](register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process), now served by the agents' own files, which hold their task and answer format, while the tool steers no process ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)), and the working rule in `AGENTS.md` on fixing failures in the tool contract; the spec's authority and escalation to the owner ([DEC-063](register.md#dec-063-the-model-is-authoritative-defects-escalate)), explicitness, every file accounted for, removed intent removing its realization, delegated realization, quality measures as warnings ([ENH-002](register.md#enh-002-model-quality-linter-and-metrics)), continuous change as a later loop, and honest results, finite budgets and provider neutrality from concept 1.0.

## 16. Questions for the owner

Each open one has a recommendation. Already decided by the owner: that required checks go for now to see whether they are needed, that the agents are started by the owner's own session, that the app goes, that `update` comes now, and that releases, the plugin and a binary that updates itself come with it ([DEC-095](register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). The name `submit` and its landing, decided earlier, gave way on 6 November 2026 to a tool that computes and checks, with `gate` as its door ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts)), and uses no Git ([DEC-097](register.md#dec-097-markitect-uses-no-git)); on 7 November 2026 the owner decided how a builder reports a spec problem ([lean plan, decision 7](lean-tool-plan.md#decisions-for-the-owner)), and accepted this concept as the target ([DEC-098](register.md#dec-098-concept-40-is-the-target)), which left the open questions below open.

1. **Starting the session.** Decided by the owner on 5 November 2026: the owner's own session starts the agents, with no background session ([§5](#5-what-build-does)).
2. **What the work may do.** Settled with question 1: the agents work under the owner's own permission mode and settings. The builder file asks to accept file edits without asking, but in the trials the session's mode held instead ([§9](#9-the-harness)).
3. **How the owner hears of the end of the work.** Settled with question 1: the owner's session waits for the builder and sees it end.
4. **Limits.** Settled by the owner on 7 November 2026: the tool keeps no limit and counts no rounds, and the owner stops the work ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)). The same day the owner added that a loop stops when the list of `status` has not got shorter for three rounds in a row, and the skill reports what remains ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). What follows is the earlier recommendation. `gate` kept two repair rounds; time and the number of calls are the driver's, with no spending limit, since the provider offers none here ([lean plan, decision 3](lean-tool-plan.md#decisions-for-the-owner)). The default driver sets no time or call limit of its own. Recommended: keep it so, and let a workflow driver hold time and calls in code if step 3 shows work that goes on too long.
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

    With [DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache) the three ways lapse with `gate` and the builder's report. The builder's agent file says to stop and say so when the machine lacks something; its items stay on the list of `status`, and nothing starts the builder again but the owner, so the cost is bounded by the owner. Open is only whether the record needs a word of its own for it, beside `skip` and `fail`.
