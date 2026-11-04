# Markitect concept 4.0 (draft)

**Markitect is a spec, the tools that work on it and a harness that teaches ordinary agents to use them. Agents follow the process, and a tool decides whether their work counts.**

| | |
|---|---|
| **Version** | 4.0 – draft, not accepted |
| **Date** | 4 November 2026 |
| **Basis** | The [owner's statements](sources/fresh-start-20261104.md) of 4 November 2026, concept 3.0, a workflow trial on one real change, the Claude Code documentation as read that day, and an independent review of an earlier version of this draft. |
| **State** | A draft, not accepted ([OQ-012](register.md#open-questions)). [Concept 3.0](v3-concept.md) stays the target until the owner decides. Nothing here is built. |

It uses the words of [concept 3.0](v3-concept.md#2-words): spec, implementer, reviewer. Code and older documents still say model, maker and checker.

## 1. The aim

Make one clean and simple version work reliably on Sonnet. The aim is no longer to prove that one version is better than another.

The problem has not changed: a project holds what is, and what should be lives in people's heads. [Concept 3.0 §1](v3-concept.md#1-what-markitect-is-for) states it as six problems, and they stand. What changes is how much machinery sits around the agent.

**The owner's way of working:**

1. A person designs or changes the spec in an ordinary agent session.
2. When the spec looks right, the person uses the apply skill.
3. Claude opens a branch and applies the change, in a session of its own, and asks nothing on the way. The person doesn't have to watch.
4. `apply` always compiles the spec, never changes it, and works through the checklist and `impact`.
5. The person models or applies, not both: nobody edits the spec while it is being applied.
6. Other commands, such as a bug hunt, a cleanup or a check that spec and code match, also run on a schedule.

## 2. What Markitect is

| Part | What it is | Changes with the provider? |
|---|---|---|
| **Spec** | Typed documentation in Markdown, with the registry. Unchanged ([concept 3.0 §3 and §4](v3-concept.md#3-the-spec)) | No |
| **Tools** | One program, `markitect`. It holds every fact and every rule that can be checked without judgment | No |
| **Harness** | Files `markitect init` writes into the project and `markitect update` refreshes: skills, an agent, hooks and settings. It holds the process | Yes |

**The rule that divides them:** anything a tool can check goes in the tool, and only what needs judgment goes in a skill's text. A skill says how to do the work. The tool says whether it counts.

There is no host, no container and no session started by Markitect, and no app for now ([§11](#11-providers)). The agent runs in the provider's own program, on the user's own login, from a session the user opened.

## 3. The tools

### Commands on the spec

None of these starts an agent. `--repo DIR` names a folder inside the project's repository (default: the current folder), and `--json` prints JSON for scripts.

| Command | Does | Change from today |
|---|---|---|
| `init` | Sets up a new project: `.markitect/`, a first spec page and the harness | First-time setup only |
| `update` | After a new `markitect` is installed, or a project's own Library changed: refreshes the harness, moves the standard Libraries in the manifest to this version and rewrites the lock. Never touches spec pages. The owner reads the diff and commits it | New. Replaces `lock`, and `init`'s refreshing of the skills |
| `check` | Compiles the project and fails on any error ([below](#what-check-refuses)). `--spec` looks only at the spec itself | Stale entries and unused Definitions become errors. `--spec` is new |
| `refs TARGET` | Who links to a Definition or Package, and which files realize it | None |
| `impact` | What a change reaches ([below](#how-impact-works)) | `--base` defaults to the last Result. The per-Package lists go |
| `analyze` | Measures the spec's design ([§4](#4-is-the-spec-good)). `--severity` filters its findings, and `--graph` draws the Package dependency graph | New. Replaces `lint` and `graph` |
| `version`, `licenses` | As named | None |

**The lock file stays.** It holds a fingerprint of each Library, as `go.sum` does for modules. Kinds shape what a spec means, so `check` refuses a Library that no longer matches, and a change to one shows up in a commit. Only the command goes.

### What `check` refuses

`check` passes or fails, with no warnings: a project either compiles or it doesn't. Its errors come in two groups, and it prints them apart.

| Group | Errors | Who settles them |
|---|---|---|
| **The spec** | A Kind or link that does not resolve, a Package cycle, a Library that does not match the lock, and the rest of today's errors in the spec | The owner, in the spec |
| **The realization** | A file neither listed nor ignored, a registry entry for a Definition or file that no longer exists, and a Definition that nothing links to and no file realizes | The implementer, in the work |

So `check` goes red the moment the owner deletes a Definition or adds one nothing uses yet, and stays red until `apply` has realized the change. That is how it should read: spec and code don't match yet. `--spec` answers the narrower question "is the spec itself valid?", which is what a modeling session asks.

`markitect start` stops only on errors in the spec. Errors in the realization become checklist items, as an unlisted file does in today's engine, and `submit` refuses until none is left.

On finished Makerspace projects, at steps c5 and c8, there are no stale entries and one unused Definition, `documentation`, out of about 115. So a finished project can meet this rule. A rule that applies everywhere still needs a file that realizes it, or a link from something that does.

### How `impact` works

Like an incremental build, it works out what a change touches, so that only that gets reworked and reviewed. It compares a base commit with the working tree.

| Distance | Reached | On the checklist |
|---|---|---|
| 0 | Definitions that changed, Definitions whose files changed, and Definitions a registry row gained or lost | Yes |
| 1 | Definitions that link to a changed one or are of a changed Kind | Yes |
| 2 and more | Everything further along the links, with distance and reason | No |

It doesn't follow the code's own imports. Required checks on the whole project and a full `verify` now and then are the safety nets. The list is rebuilt after the work, so a file touched outside the first impact adds its Definitions. On Makerspace's change steps, one step reached 33 of the 36 existing files that truly changed, and deeper reach 35 ([measurement](../validation/impact-reach-20261103.md)). How far every link reaches says something about the spec too (§4).

### Commands of a run

They are thin, and most of their logic exists today in `internal/run`.

| Command | Called by | Does |
|---|---|---|
| `markitect start` | The skill, before the agent sees it | Compiles the spec, works out the base and opens a run. Commits a spec change that is not committed yet, and switches to a branch of its own, `markitect/<run>`. Prints a short summary, not the list. Fails, which aborts the skill, on an error in the spec, an uncommitted change outside the spec, or nothing to do. On an open run it continues that run with its repair count |
| `markitect checklist` | Implementer and reviewer | Prints the list, rebuilt from the current files and registry, so a file outside the first impact adds its Definitions. For a reviewer it also notes the state of the work |
| `markitect answer` | Reviewer | Stores its answers |
| `markitect submit` | Implementer | Runs the final check (§6). If it passes, commits the work with the Result on the run's branch, brings the branch the owner came from up to it and switches back. `--check` only reports |
| `markitect stop` | Implementer, or the owner | Ends the run without landing: commits what is there with a Result on the run's branch, and switches back to the owner's branch |
| `markitect hook` | Hooks only | Called when a reviewer or the implementer tries to finish (§5) |

**The base.** The last Result on the branch names the work it realized, as today. With no Result yet, the first `apply` is `--full` and says so. Commits made beside `submit` stay in the next `impact`, so a hand fix or a bypass still gets reviewed by the next run.

**State of the work** means a Git tree hash of every file Git would take, made with a temporary index, so ignored build output does not count and nothing is written to the user's repository.

**Run state** lives in `.git/markitect/` and never reaches the working tree. Work that did not land stays on its run's branch, where the owner can look at it, and the Result says why.

**Required checks** run on the user's machine through `sh -c`, in a clean temporary export of that state, in the order of `execution.yaml`. Leftovers can't make them pass, and their output can't change the work. The result is kept per state, so they run once for each. The `environment` field, which names a container image, goes. They run code the agent wrote with the user's rights, as the agent's own shell already does.

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
| Change hotspots from Git history | Churn | What keeps changing, with how far it reaches | Later |
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

## 5. What `apply` does

The owner models until the spec looks right, then uses `/markitect-apply`, from the same session or another. From there it runs without them, and they read the Result afterwards.

1. **Start.** The skill's first line is an injected command, `` !`markitect start` ``, which the provider runs before any agent sees the skill. If it fails, no agent starts and the owner sees why. Otherwise the spec is committed and the work is on its own branch.
2. **Its own session.** The skill runs with `context: fork` in the `markitect-implementer` agent, so the implementer starts without the design discussion, and the owner's session only waits.
3. **Implement.** The implementer takes the list from `markitect checklist` and makes the project match the spec, keeping the registry current. It does not commit and does not change the spec. If the spec contradicts itself or leaves open something that changes behaviour at the project's boundary, it runs `markitect stop` with why: nothing lands, and the owner decides. A gap with no such effect is its own to decide.
4. **Pre-check.** `markitect submit --check` shows what the final check would refuse now, before a reviewer spends anything.
5. **Review.** The implementer starts the `markitect-reviewer` agent and waits. The reviewer starts with fresh context, has no tool to edit files and cannot start further agents. It takes the list from `markitect checklist`, reads the Definitions and the code, and stores its answers with `markitect answer`, one line per item ([concept 3.0 §8](v3-concept.md#8-the-checklist)).
6. **Seal.** When the reviewer tries to finish, a SubagentStop hook calls `markitect hook`. A missing or unreadable answer, a pointer to a file or line that does not exist, or a state of the work that differs from when the reviewer took the list sends it back. Otherwise the answers are sealed with that state, its agent type and its transcript's path. Only sealed answers count.
7. **Repair.** A `wrong` answer goes back to the implementer, which fixes it and starts a new reviewer. At most two repair rounds, counted by the tool; the third refusal ends the run as with `markitect stop`. Every review answers every item, since a repair changes the state. A `spec problem` also ends the run.
8. **Submit.** `markitect submit` runs the final check, writes the Result, commits on the run's branch, brings the owner's branch up to it and switches back.
9. **End.** A SubagentStop hook on the implementer calls `markitect hook`, which only reads whether the run is closed and sends the implementer back if it is not. The owner reads the Result and pushes when they want to.

**It asks nothing.** The agents can't ask the owner a question, and they run in the `dontAsk` permission mode: a command nobody pre-approved is refused instead of waiting, and the agent carries on without it or stops and says why. That keeps a run from sitting there until someone comes back.

**One at a time.** Since nobody models during a run, the owner's branch stays where it was, so bringing it up is a fast-forward. If it did move, nothing merges: the work stays on the run's branch, and the Result says so.

The implementer uses Git to read, not to record. The tool makes the branch and the commits. The implementer and reviewer take turns in one folder, with no worktrees and no bundles.

**After a crash or an abandoned session,** the work is still on the run's branch and the run still open. `start` continues it with its repair count, and `markitect stop` ends it. Only `submit` and `stop` close a run.

**`verify`** runs the same reviewer on the committed state and changes nothing in the owner's work. Its Findings and Result are committed on a branch of its own, where `apply --findings` and the full-check count read them. It needs no clean working tree, since it reads the commit through a temporary export.

## 6. The final check

`markitect submit` refuses when:

- anything under `.markitect/` changed except the registry pages, so the spec, manifest, lock and required checks stay as the owner left them;
- anything in the harness changed, so an agent can't weaken its own rules in the work it submits;
- `markitect check` reports any error, in the spec or the realization;
- a required check fails;
- an item has no sealed answer, or the sealed answers are for another state of the work;
- an answer is `wrong` or `spec problem`, or a pointer no longer holds.

`not shown` may land, and the Result lists it.

**The Result** is written by the tool, as in [concept 3.0 §10](v3-concept.md#10-the-result): each item's answer and pointers, what could not be tested, and how much checking really happened, measured from the reviewer's transcript. A landing's Result goes into its commit under `.markitect/results/`.

## 7. How sure we are

The tool can't stop a determined agent that runs as the user on the user's machine. This concept guards against forgetting and shortcuts, which is what has been seen, and against deliberate bypass only by making it visible.

| What the owner wants | What secures it | How firm |
|---|---|---|
| The spec compiles before work starts | Injected `markitect start` | Firm, unless the agent is started without the skill |
| A commit by `submit` is complete: spec unchanged, spec compiles, checks pass, every item answered for this state | The final check | Firm for what `submit` commits |
| A run doesn't stop to wait for someone | `dontAsk` on both agents | Firm if the owner's session is in Manual mode. A session in auto or `bypassPermissions` mode passes its own mode on to them |
| Work goes through `submit` at all | The hook on the implementer; a hook against direct commits and pushes | Catches forgetting. That hook gives up after eight blocks in a row unless a tool runs between them, and it can't catch a session that is simply closed. Command rules match text, so `git -C . push` gets past a deny rule |
| A commit that bypassed `submit` is noticed | It stays in the next `impact`, and `start` prints how many commits there are since the last Result | Seen afterwards, not prevented |
| A fresh reviewer did the review | The hook seals answers only from an agent of that type, and keeps its transcript | Good, not proof: another agent with that name, or an edit to `.git/markitect/`, could fake it |
| The reviewer really read the code | Pointers, measured depth, planted bugs | Not provable. It was not provable in the Docker version either |
| The code is right | The reviewer and the project's tests | Not guaranteed by any version |

**Anyone who wants more** can run `markitect submit --check` where the agent can't reach it: on a server of their own, in CI or in a hook on a remote. Markitect depends on none of these.

**What we give up against concept 3.0:**

- prevention of a deliberate bypass;
- isolation from the user's machine: agents run with the user's rights;
- a reviewer guaranteed to run in its own session as another user, with a throwaway folder. It now shares the folder, so its test runs can touch the work, which the seal catches for tracked files. Code it runs can still change its stored answers;
- review of a clean copy: it reads the folder as it stands, ignored leftovers included, though required checks run on a clean export.

## 8. Commands are skills

Every command is a skill over the same tools. The owner or a schedule starts it directly, so the agent does not have to decide to use it. A command that only reports needs little certainty, since a missed step costs little. Only commands that change artifacts go through `submit`.

`apply` and `verify` come first. `cleanup`, `bughunt`, `security` and `improve` follow the table in [concept 3.0 §12](v3-concept.md#12-other-commands), as skills, once those two work. Modeling and Extract stay as they are.

Skills stay short and put what matters first: after compaction, the provider keeps only the first 5,000 tokens of each. That is why the list comes from a command and not from the skill's text. A skill says "use the `markitect-reviewer` agent" rather than calling another skill, because the documentation describes agents preloading skills and does not describe skills calling skills.

## 9. The harness

For Claude Code, `markitect init` writes project files, so they are versioned with the project:

| File | Holds |
|---|---|
| `.claude/skills/markitect-apply/SKILL.md` | The steps of §5, starting with the injected command; `context: fork`, `agent: markitect-implementer`, and `background: false` so the owner's session simply waits |
| `.claude/skills/markitect-verify/SKILL.md` | The same for `verify` |
| `.claude/agents/markitect-implementer.md` | Model, effort and `permissionMode: dontAsk` |
| `.claude/agents/markitect-reviewer.md` | Model, effort, `permissionMode: dontAsk` and tools without edit or agent tools |
| `.claude/settings.json` | The hooks of §5, and PreToolUse hooks against edits to the spec and direct commits and pushes; the pre-approved tools; subagents forced onto the chosen model |

**The hooks live in the project settings,** because the documentation doesn't say that hooks in an agent's file fire for a forked skill. `markitect hook` does nothing unless a run is open, so a modeling session stays free to edit the spec and commit.

**Pre-approved** are file edits, `markitect`, and the project's build and test commands, which `init` and `update` take from `execution.yaml`. A modeling session gets those same approvals, since they sit in the project's settings. `bypassPermissions` is not used: the provider means it for isolated environments.

**The model.** Fixing it in an agent's file is not enough, since a call can override it and helpers inherit the main session's. So the project settings force subagents onto it: one earlier launch on Haiku started 68 subagents on Opus.

Plugins do not fit: the provider ignores hooks in plugin agents.

## 10. Strategies

**A strategy is a folder of harness files and nothing else.** There is no code interface to implement. Adding a version means adding a folder, and removing one means deleting it.

**A strategy may change** the skills, agent, hooks and settings. **It may not change** the tools or the final check.

The first version ships one strategy and no switch for it. The natural second is the same skills with a saved provider workflow that fixes the order of steps, as in the trial of §14. It is added only if the first one skips steps too often, and a project's choice is recorded then.

## 11. Providers

The first version is Claude only. Skills use the same `SKILL.md` format at Codex, and `init` already writes them to `.agents/skills/`. Its agents and hooks work differently, and what we know of those comes from third-party pages, so a strategy needs its own version per provider. Codex gets one once Claude's version is reliable.

No token is needed for `apply`: sessions run on the user's own login in the unmodified program, which removes token passing. The legal reading of [concept 3.0 §14, question 10](v3-concept.md#14-decided-on-acceptance) still applies: the owner's own use looks allowed, and other users are not settled, so the owner still asks Anthropic before anyone else runs Markitect.

**The app goes for now.** Markitect Desktop, with its spec views and chats, leaves the product, by the owner's call. Its chats were the only other use of containers, so this takes Docker out of Markitect altogether, along with the provider adapters, the session helper, the runtime image and the token. It stays recoverable in Git, and can come back once the tool has settled.

## 12. Schedules and continuous work

[DEC-084](register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps) stands: the loop runs outside Markitect, as a scheduled job or in CI that calls `markitect`, and there is no daemon in the product. A scheduled job starts a skill. Government stays a later step ([IDEA-002](register.md#idea-002-government-delegated-model-maintenance)): its mandate would live in the spec, and a fresh reviewer would accept proposals within it.

Three things change once nobody is watching, and none of them is in the first version:

- **Nobody opens a session.** A schedule would use `claude -p "/markitect-apply"`, which the provider supports. `apply` itself doesn't need it. The documentation says `--bare` "will become the default for `-p`" in a future release, and `--bare` skips hooks and skills and doesn't use a subscription login, so this route would need a pinned CLI version.
- **External signals need isolation.** Issue text can hold instructions aimed at the agent, and a native agent has the user's rights. A run that reads outside input needs a container or a machine with limited rights around the agent. This wraps the harness and needs no engine.
- **Skills must not set `disable-model-invocation: true`** if a session-scoped schedule is to fire them.

## 13. Order of work

| Step | Result | Done when |
|---|---|---|
| 1. Probe | The harness by hand on one small change, with the trial's throwaway check, to see that the mechanisms behave as the documentation reads, on Windows: injected commands, a forked skill starting the reviewer, hooks in the project settings firing for both agents, and `dontAsk` never stopping to wait | Each works, or the concept is changed |
| 2. Build | The new commands, `update`, the stricter `check` and `analyze` in Go, reusing `internal/run` and `lint`; `apply` and `verify` as skills in `init` | A Library change lands, and planted refusals hold |
| 3. A simple evaluation | A talk with the owner, then a small evaluation used to make the version reliable | The owner's reading |
| 4. Remove | The engine, its adapters, the suite, the app and everything only they use go | No page describes them as current |

This order keeps a working `apply` until the new one works. The owner first asked to remove the engine and the suite up front, then chose to leave them while this is discussed, so the order is a question below. No product code depends on the suite, so it can go at any time.

## 14. What is not yet shown

- **This shape has not run.** A workflow trial on 4 November landed one Makerspace change unattended with no Docker and no token, and its reviewer caught a planted bug once. That was a script fixing the order, with a throwaway check on a local repository, not skills and hooks. Its evidence is in `C:\Users\Consiliari\wf-trial\evidence`, outside the repository.
- **How often a native agent skips steps is unmeasured.** The owner has seen it forget things. The evaluation has to answer this first.
- **The mechanisms in §5 are read from the documentation, not tried.** It doesn't state whether a forked skill's agent can start another agent, or whether SubagentStop hooks fire for it. If a forked skill doesn't work this way, the fallback is the same skill in a fresh session. In the provider's auto mode a subagent's report goes through another tool, which is why answers are stored by command and not taken from the reviewer's last message.
- **A plain agent on the spec is untested.** The earlier runs that went well gave the agent a clean document, not the typed spec.
- **Reviewing at 5× has no result.** Test 1 stopped when the machine went down, with nine of twelve launches run and no answer read. The first version uses one reviewer for the whole list. At hundreds of items, a split per Package may be needed.

## 15. What this overturns

| Entry | Becomes | What stays from it |
|---|---|---|
| [DEC-091](register.md#dec-091-concept-30-is-the-target) | Superseded as the target, once this is accepted | Concept 3.0 as the record; its six problems, words, checklist, answers and Result |
| [DEC-051](register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process) | Process moves into the skills | Rules stay out of prompts: they belong in the tool. The working rule in `AGENTS.md` on fixing failures in host state changes with it |
| [DEC-078](register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) | No remote involved: a run has a local branch, `submit` brings the owner's branch up to it, and the owner pushes | Work lands without the owner reviewing a diff |
| [DEC-083](register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers) | No containers for `apply` | Native on Windows, Linux and macOS |
| [DEC-089](register.md#dec-089-markitect-desktop-the-app-over-the-cli) | Parked: no app for now. If it returns, `apply` works in the user's working tree | The idea of an app over the CLI |
| [DEC-092](register.md#dec-092-a-command-uses-the-users-own-sign-in), [DEC-093](register.md#dec-093-a-command-passes-the-users-own-token) | No Command and no chats, so no token at all. DEC-092's unbuilt fallback, skill and hook, becomes the default | The question to Anthropic |
| [The guide's build decision on stale realization](../guides/model-language.md#build-decisions), confirmed by the owner on 28 October 2026 | Stale entries are errors, not warnings | The Command that cleans them up still starts: they are errors in the realization |
| [Concept 3.0 §6, §9, §11](v3-concept.md#6-what-apply-does) | Landing in the CLI after a session, the sandbox and prompts over skills all go | The rule that a strategy can't change the check |

**Stays unchanged:** the spec's authority and escalation to the owner ([DEC-063](register.md#dec-063-the-model-is-authoritative-defects-escalate)), explicitness, the language, every file accounted for, removed intent removing its realization, delegated realization, quality measures as warnings only ([ENH-002](register.md#enh-002-model-quality-linter-and-metrics)) and continuous change as a later loop.

## 16. Questions for the owner

1. **The old version.** Keep the engine until the new one works, as §13 has it, or remove it first for a clean start? And does the suite go now, or wait for the talk on the evaluation?
