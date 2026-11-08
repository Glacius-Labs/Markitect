# Plan: Markitect as a tool, with a verification cache

**Status:** decided by the owner on 7 November 2026 ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)), after a discussion of how they work with Markitect. An independent architect reviewed this page against the code. Steps 1 and 2 of the [order of work](#order-of-work) are built. Step 1 was released as 0.21.0 ([the record](../../internal/run/README.md)); step 2 was released as 0.22.0, with the skills, roles and routines the owner decided the same day ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role), [below](#skills-roles-and-routines)). The strategies, the parallel driver and the configuration the owner decided next ([DEC-103](register.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow), [below](#strategies-and-configuration)) are built for 0.23.0. Step 3 is next.

The owner decided:
- Markitect supports agent work and does not steer it;
- a record of what was verified is kept, modelled on Go's test cache;
- the command names follow Go and widely known tools;
- the work needs no limit on rounds; with DEC-100 a loop stops when the list has not got shorter for three rounds.

An earlier version proposed a command line like Terraform's, in which the tool steered the work through phases. This version replaces it, and reuses its parts on output and exit codes. It builds on [concept 4.0](v4-concept.md) ([DEC-098](register.md#dec-098-concept-40-is-the-target)) and on DEC-096 and DEC-097 ([lean plan](lean-tool-plan.md)). It replaces their `gate`, their work state and their default driver's steps.

## How the owner works with Markitect

1. **Model.** In an ordinary session, the owner designs the spec with an agent: Definitions, Packages, links and Kinds. The tool checks that the spec is structurally sound and free of conflicts, as a compiler does. It analyses the spec's design the way code is analysed for coupling, and shows dependencies and what a change reaches.
2. **Build.** When the spec is right, agents make the project match it. That is ordinary agent work in sessions and subagents, with prompts. The tool does not drive it.
3. **Make sure nothing is forgotten.** The tool says what has to be verified for the spec to count as implemented. The agents verify each item and record it. When nothing is left to verify, the spec is implemented.

The tool helps at three points: describing, saying what to do, and showing that it is done. It decides nothing about who works when. The skills are prompts that tell the agents to use the tool.

## The idea: a cache of what was verified

`go test` does not run a test again when nothing it depends on has changed; it prints `(cached)`. Markitect does the same for verification. For each Definition it records the state at which an agent last verified it as correctly implemented. That state is the set of everything the verification depended on, its **inputs**:

- the Definition's own text;
- its Kind and every Definition it links to: their text, or for a Library Definition the Library's digest from the lock, so that a newer binary changes nothing;
- the registry rows that name it, and the content of the files they name;
- the files the reviewer gave as evidence, such as a test that lies in another Definition's rows.

Its fingerprint is the hash of these inputs.

**What is to verify** is every Definition with a changed input or with no record. The tool computes that from the folder and the record alone, with no session and no start. These are the same Definitions today's checklist reaches ([concept 4.0 §3](v4-concept.md#how-status-works)):

- one that changed;
- one that links to a changed Definition, or is of a changed Kind;
- one whose files changed;
- one whose registry row gained or lost it.

In both designs reach stops after one step, and a changed Rule above a Package reaches only itself and what links to it.

Three more kinds of item stay on the list until they are verified again:

- **A Definition recorded as `fail` or `spec-problem`** stays with that note.
- **A Definition that is gone from the spec** stays, marked gone. Its line names the files that realized it, so the item can ask whether they are removed or have a new basis. Reviewing it `ok` removes its line.
- **An error in the realization** stays: an unlisted file, a stale registry row, an unused Definition.

The record holds each input's hash, so `status` can say which inputs changed and where to look. It does not hold what a Definition said before; an agent that wants the old text reads it from Git.

**Recording** binds a verdict to the state the reviewer read. `status` prints each item as `DEFINITION@FINGERPRINT`, and `mark DEFINITION@FINGERPRINT VERDICT` refuses if the inputs have changed since. That catches a builder editing a file beside the reviewer, or the reviewer's own test run changing one. Go does the same: it does not cache a result whose inputs changed while the test ran. The fingerprint `status` prints covers every input but the evidence, which the reviewer chooses later. The evidence must exist in the folder, and its files are hashed when the verdict is recorded.

**Done** is an empty list, whatever happened in between. A file edited by hand, a commit made around the agents, or a change on a merged branch alters the inputs it touches, and those Definitions are on the list again.

## The record

`.markitect/review.sum` is committed with the project, like `go.sum`, and it joins the fixed layout of `.markitect/`. It holds one line per Definition, sorted, each line a JSON object:

```text
{"definition":"fees/waiver","verdict":"ok","inputs":{"self":"sha256:…","link:fees/fee":"sha256:…","file:src/fees.go":"sha256:…","evidence:tests/fees_test.go":"sha256:…"},"evidence":["src/fees.go:40","tests/fees_test.go:12"]}
{"definition":"members/limit","verdict":"fail","inputs":{…},"evidence":["src/loans.go:70"],"why":"allows a sixth loan; the Definition says five"}
```

**Verdicts,** after `go test`:

| Verdict | Means |
|---|---|
| `ok` | The Definition is implemented, as the evidence shows |
| `fail` | It is not; the line says why |
| `skip` | Verified as far as the project can show, with the reason, as `t.Skip` gives one. `status` lists it as a note, and a note alone does not make `status` exit 1 |
| `spec-problem` | The spec contradicts itself or leaves open behaviour at the project's boundary, and only the owner can settle it |

An explanation stays on one line, and hashes are full SHA-256.

**How it is written.** `mark` writes under the lock the tool already uses, as one whole file, so parallel reviewers, for example one per Package, do not lose lines.

**How it merges.** Git also conflicts on changes to neighbouring lines. So `init` writes `.markitect/.gitattributes`, which joins the fixed layout, with `merge=union` for this file: Git keeps both sides. Of two lines for one Definition, `status` takes the one whose inputs match the folder. If both match with different verdicts, `status` lists the Definition as a conflict to review. The next `mark` writes the file clean.

**What it replaces:** today's certificate, the stored answers, the work under way in the cache, the Result pages, the Findings pages and the builder's reports.

**What it is:** a record, not a proof, as the certificate is today. A line written by hand shows in the diff, and the hook keeps agents from writing one ([below](#the-hook)).

## The commands

| Command | Does | Like | Today |
|---|---|---|---|
| `markitect init` | Sets up the project, its harness and its lock | `go mod init` | `init` |
| `markitect check` | Compiles the spec and the registry: Kinds, links, Packages, Libraries against the lock, every file listed or ignored. It produces nothing | `cargo check` | `check` |
| `markitect vet` | Analyses the spec's design: coupling, size, fan-in, how far a change reaches. It warns, and exits 1 only with `--strict` | `go vet` | `lint`, the planned `analyze` |
| `markitect list` | The Definitions and Packages; with `--deps`, what each depends on | `go list -deps` | `graph` |
| `markitect doc DEFINITION` | A Definition with its text, what links to it, what it links to, the files that realize it, and its record | `go doc` | `refs` |
| `markitect status` | What is to verify, with which inputs changed, where to look, and each item's fingerprint. It also notes whether the binary and the harness are current, which never changes its exit code. `--all` lists every Definition, to verify everything again, and so exits 1 whenever the spec has one | `git status`, `npm outdated` | `status`, `impact`, `checklist`, `gate --verify` |
| `markitect mark DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]… [-m "WHY"]` | Records a verdict at the state the reviewer read. Evidence is `path:line` or `path`; a test name in parentheses is dropped | `go test` writing its cache | `answer`, `gate`, `gate --spec-problem` |
| `markitect tidy` | Brings the lock in line with the manifest. `check` only reads the lock and, on a mismatch, says to run `tidy`; `init` and `update` keep it current | `go mod tidy` | `lock` |
| `markitect update`, `install`, `version`, `licenses` | As today | `go version` | the same |

**What goes:**
- `checklist` with all its flags;
- `answer`;
- `gate`, with `--verify`, `--findings` and `--spec-problem`;
- `impact` and `graph`;
- the work state in the cache;
- certificates and Result pages;
- repair rounds;
- the hook's rule on `checklist`.

**What remains of `gate`:** `check` still fails on a project that does not compile, and `status` lists the realization's errors. So an empty `status` means two things: the project compiles, and every Definition is verified at its current inputs.

## Skills, roles and routines

Decided by the owner on 7 November 2026 ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)), and built for 0.22.0. Skills are named after the activity, agents after the role. The skills are prompts that name which agent starts and on what task; the agents hold their tasks and the verdict words in their own files, which outlive compaction.

| Skill | Who may start it | Agent | Does |
|---|---|---|---|
| `markitect-design` | The owner or Claude | The session itself; `markitect-architect` for a delegated draft | Designs the spec with the owner. It sees whether the folder is an empty project, code without a spec or an existing spec, and starts the spec, draws it from the code or changes it. `reference.md` and `examples.md` lie beside it. It replaces `markitect-start`, `markitect-extract` and `markitect-modeling` |
| `/markitect-build` | The owner | `markitect-builder`, in a worktree of its own, which starts `markitect-reviewer` | The builder makes the project match the spec for what `status` lists and has the reviewer verify and record each item, until the list is empty. Then the session commits the work and merges it into the owner's branch |
| `/markitect-review` | The owner or Claude | `markitect-reviewer`; one per Package by the parallel strategy ([below](#strategies-and-configuration)) | Verifies what `status` lists, or every Definition with `--all`, and records each verdict. The session commits the record |
| `/markitect-cleanup` | The owner | `markitect-builder`, in a worktree of its own | The builder refactors without changing behaviour; the reviewer verifies again what `status` lists. It comes back when it leaves nothing worse |
| `/markitect-bughunt` | The owner | `markitect-builder`, in a worktree of its own | The reviewer hunts beyond the list and records each bug as `fail`, or as `spec-problem` where the spec does not cover it; the builder repairs; the reviewer verifies again. It comes back when it leaves nothing worse |
| `/markitect-audit` | The owner or Claude | `markitect-auditor` | Reads the spec fresh for contradictions, gaps, unclear and duplicated statements, and reports to the owner. Nothing is recorded |
| `/markitect-schedule` | The owner | none | Sets up routines through Claude Code's own scheduling ([below](#routines)) |
| `/markitect-config` | The owner or Claude | none | Changes the harness configuration with `markitect config`, for this project or for all of the user's projects ([DEC-103](register.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow), [below](#configuration)) |

`disable-model-invocation` keeps Claude from starting a skill that only the owner starts. `markitect-design` is installed for Claude Code and for other agents (`.claude/skills/` and `.agents/skills/`); the rest is the harness for Claude Code.

### The roles

Four roles in a 2×2 of rights: one writes code and one judges it, one writes the spec and one judges it.

| | Writes | Judges |
|---|---|---|
| **The code** | `markitect-builder`: code and the registry pages; not the spec, no `mark` | `markitect-reviewer`: changes nothing; the only one who runs `mark` |
| **The spec** | `markitect-architect`: the spec, the registry pages and the ignore file; no code, no `mark` | `markitect-auditor`: reads only, and reports to the owner |

The architect is for delegated drafting, such as a first spec drawn from a large code base; it decides nothing the owner leaves open and hands its questions back to the session that started it. The builder may start only the reviewer.

**What an agent notices** goes into the record, `review.sum`, and nowhere else: the builder tells the reviewer what it saw, and the reviewer records a `fail` on the Definition a bug breaks, or a `spec-problem` on the nearest Definition when the spec does not cover it.

**Loops have no limit on rounds,** but the builder stops when three rounds in a row end without the list of `status` getting shorter, and the skill reports what remains. This holds for building and for routines.

**Tests are the project's.** A project that wants tests states them in its spec as an ordinary Rule; the reviewer runs them as evidence; a routine finds them in the spec, the README or the CI configuration. The tool knows nothing of them.

### The hook

The hook holds the roles' rights for subagents. The owner's own session is no subagent and stays free to change anything, but it does not record: only the reviewer runs `mark`.

| Subagent | Code | Spec | Registry pages | `review.sum` | Harness | `mark` | Starts |
|---|---|---|---|---|---|---|---|
| `markitect-builder` | yes | no | yes | no | no | no | only `markitect-reviewer` |
| `markitect-reviewer` | no | no | no | no | no | yes | nothing |
| `markitect-architect` | no | yes | yes | no | no | no | nothing |
| `markitect-auditor` | no | no | no | no | no | no | nothing |
| any other | yes | no | yes | no | no | no | anything but the builder and the architect |

Code is every file outside `.markitect/` but the harness, and the spec every file in it but the registry pages and the record, which only `mark` writes. The harness is what steers the agents: the project's Claude Code settings (`.claude/settings.json` and `settings.local.json`), which hold the hook, Markitect's skills and agents, and `AGENTS.md` and `CLAUDE.md`. Since the builder's work comes back without anyone reading its diff, no subagent may change them. Any other subagent writes as the builder does, so a general-purpose helper cannot get past the rules, and only the owner's session starts the two roles that write. Claude Code ignores the agent types listed in a subagent's own `tools`, so the hook, not the agent file, holds who starts whom. The hook reads tool calls, so a command hidden in a script, or one that names `markitect` by a path, still gets through. It guards against mistakes, as before, not against intent.

### The builder's worktree

`/markitect-build`, `/markitect-cleanup` and `/markitect-bughunt` start the builder in a worktree of its own, through Claude Code's worktree isolation, so that the owner can go on designing in their folder. A reviewer the builder starts works in the same worktree. Claude Code branches such a worktree from the repository's default branch unless the setting `worktree.baseRef` is `"head"`, so `init` and `update` set it when the project's settings name no base. A worktree holds no uncommitted change, so the skill asks the owner to commit first. The tool accounts for Claude Code's worktrees under `.claude/worktrees/` itself, so a worktree is no unregistered file of the owner's folder.

When the builder ends, the session commits the work on the worktree's branch, `review.sum` included. Build merges it into the owner's branch when `status` is empty there; cleanup and bughunt when it lists nothing that was not listed before. Otherwise the session asks the owner whether to merge it as it stands. It pushes nothing. In a folder that is not a Git repository, the builder works in the folder itself.

### Routines

The routines are cleanup, bughunt and `review --all`, run on a schedule with nobody to answer. `/markitect-schedule` sets them up with the owner through Claude Code's own scheduling: a scheduled task of the desktop app, which runs on the owner's machine, or a cloud routine, which runs on a fresh clone of the repository and needs `markitect` installed there. Markitect schedules nothing itself. Each routine's prompt is a frame, `routine.md`, with the routine's work in it, all files of the skill.

- **A branch of its own.** A run works in a worktree or clone of its own, from the newest commit of the source branch; from the remote's when the branch tracks one, so that nothing unpushed goes out.
- **Lands only when nothing is worse:** `status` lists no Definition that it does not also list on the source branch as it now stands, and the project's tests pass. If the source branch moved, the run merges it first and has the reviewer verify again what the merge put on the list. It lands as a fast-forward: by a push when the source branch tracks a remote, otherwise by moving the local branch, which Git refuses while it is checked out in a folder.
- **Otherwise it leaves its branch** and says why.
- **Each run ends with a short note:** what landed, what is open, which spec-problems wait for the owner.

So a review that records a new `fail` does not land: its branch holds the record, and its note names the item.

## Strategies and configuration

Decided by the owner on 7 November 2026 ([DEC-103](register.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)), and built for 0.23.0. Claude decides the parallelism per run, through a Claude Code workflow that Markitect ships. There is no fixed tree of Managers: in the runs of 2 and 3 November 2026 the Managers added cost and the worst failures.

### Strategies

Each skill that starts agents runs by a strategy: build, review, cleanup, bughunt and audit.

| Strategy | Does |
|---|---|
| `auto` | The default. Claude estimates the work and chooses `sequential` or `parallel` |
| `sequential` | One builder, reviewer or auditor at a time, started by the session, as DEC-100 built it |
| `parallel` | The shipped workflow `.claude/workflows/markitect-parallel.js` |
| any other name | A custom workflow, which `workflows.NAME` names |

**`auto` chooses parallel** when `status` lists more than about 30 items in at least two Packages, in a Git repository. Cleanup, bughunt and audit look at the whole project, so they count the spec's Definitions instead. 30 is the size at which DEC-100 already split a review, so one number rules both. In the [lean trials](../validation/lean-trials-20261107.md) one builder and one reviewer settled change steps of 21 to 64 items in one context, while a whole first build, such as Makerspace's 113 Definitions, is where one context runs short. Below it, a parallel run's fixed cost, a worktree of its own, a merge per batch of strands and a reviewer per strand, buys little, and with one Package there is nothing to split.

### The parallel driver

The workflow takes the args `{mode, focus, all}`, the mode being build, cleanup, bughunt, review or audit. A workflow script has no shell and reads no file, so a plain agent reads `markitect status --json` for it, and does its Git steps.

- **Strands per Package.** `status --json` gives each Package's items, spec-problems and Definitions, and the Packages it depends on: those `list --deps` names, and an ancestor its Definitions link to. A Package whose items are all spec-problems gets no strand.
- **Waves along the dependencies,** so that what is depended on comes first. A wave holds the Packages whose dependencies among those with work are done; a spec with a cycle between Packages does not compile.
- **Builders in parallel,** each in a worktree of its own, with the message "Build what markitect status lists in Package P." The builder works only on that Package's items and files, and starts its reviewer with the same Package. At most 8 strands run at once, since a strand is a builder and its reviewer, and Claude Code runs at most 20 subagents of a session at once.
- **Shared files go to Root's strand,** last and alone. A strand that needs a change in a file outside its Package, such as `go.mod`, the README or CI, names it; Root's strand makes it, with Root's own items and the errors in the realization.
- **After each batch** the plain agent commits each strand's worktree on its branch, merges the branches into the run's branch, removes the worktrees and runs `status` again. A strand it cannot merge it leaves on its branch, and the driver reports it.
- **Rounds** repeat until `status` lists nothing a builder can settle, or three rounds in a row leave the list no shorter, as DEC-100's loops do. The driver returns how it stopped, what is left per Package and the strands it did not merge.

| Mode | Strands |
|---|---|
| build | Rounds of builder strands, as above |
| cleanup | One round: every Package with Definitions, "Clean up Package P." |
| bughunt | One reviewer per Package hunts, "Hunt bugs in Package P", and records; the plain agent commits the record; then rounds of builder strands |
| review | One reviewer per Package with items, in the session's folder; no worktree |
| audit | One auditor per Package and one that looks only for contradictions between Packages; the session merges their findings for the owner |

**The session's part.** Before a parallel build, cleanup or bughunt the session enters a worktree of its own, with Claude Code's EnterWorktree tool. Claude Code branches every builder's worktree from the session's `HEAD`, and inside a worktree from that worktree's `HEAD`, so a later wave starts from what the earlier waves merged there, and the owner's branch is untouched until the end. When the workflow has ended, the session leaves the worktree, keeps it, and merges its branch by DEC-100's rules: build when `status` is empty there, cleanup and bughunt when it lists nothing new; otherwise it asks the owner. The owner designs in another session while a parallel run lasts.

**A custom strategy** is a saved Claude Code workflow, in `.claude/workflows/` or `~/.claude/workflows/`, named in `workflows.NAME` and started by that name. It keeps the driver's contract: the same args; it reads `markitect status --json`; the work is done by the four roles' agents only, with a plain agent at most for status and Git; and the same stop rule. Copying `markitect-parallel.js` under another name is the way to start one, since `update` restores the shipped file. Routines keep their own prompts and take no strategy, since nobody is there to answer the prompt with which Claude Code asks before a workflow starts.

### Configuration

| Level | File | Committed |
|---|---|---|
| Built-in | the agent files as the binary ships them: `sonnet`, effort `high`; strategy `auto` | |
| User | `~/.markitect/harness.yaml`, for all of the user's projects | No |
| Project | `.markitect/harness.yaml` | Yes |

Key by key, the project beats the user, and the user beats the built-in values, as `git config` has it.

| Key | Values |
|---|---|
| `agents.ROLE.model` | `sonnet`, `opus`, `haiku`, a full model ID or `inherit`; ROLE is `builder`, `reviewer`, `architect` or `auditor` |
| `agents.ROLE.effort` | `low`, `medium`, `high`, `xhigh` or `max` |
| `strategy.default` | `auto`, `sequential`, `parallel` or a custom workflow's name |
| `strategy.SKILL` | the same, for `build`, `review`, `cleanup`, `bughunt` or `audit`; without it, the skill takes `strategy.default` |
| `workflows.NAME` | the path of a custom workflow's script |

```yaml
# .markitect/harness.yaml
agents:
  builder:
    model: opus
    effort: max
strategy:
  build: parallel
```

- **`markitect config [--user] KEY VALUE`** sets a value in the project's file, or with `--user` in the user's, and rewrites the agents' files of the project here. `--unset KEY` removes one, `KEY` alone prints the value in force, and `--list` prints every value with where it comes from, `--json` too. A strategy that names a workflow no level defines is refused; `--list` reports one left after an unset, and exits 1.
- **`init` and `update` write the agents' front matter** from the configuration, and `update` notes a project whose agent files differ from it. A user's value thus reaches the committed agent files when that user runs them: Claude Code has no agent scope that is not committed and beats the project's. A team that wants the same agents for everyone sets them in the project's file.
- **The manifest's `model`** of earlier releases moved here. `update` moves it, as the model of every role the project's file does not set, and `check` reports `manifest.model` until then.
- **`.markitect/harness.yaml` joins the fixed layout,** and `check` reports a file it cannot read as `harness.syntax`, for the owner to fix.
- **`/markitect-config`** takes the request in plain words, asks whether it applies to this project or to all of the user's projects, and runs `markitect config`.

### The hook on the configuration

No subagent writes `.markitect/harness.yaml`, the architect included, or a shipped workflow, `.claude/workflows/markitect-*`: they steer the agents, like the settings and the skills. No subagent runs a workflow, since a workflow starts agents of every role; only the owner's session does. The user's own skills in `.claude/skills/` are left alone by `init` and `update`; they may start the four roles' agents, and the hook holds those agents' rights there as everywhere.

### Trial

Nothing of this has met a real agent yet: the workflow ran only against stub agents. A bounded trial dispatch is drafted for the owner. It first probes, on Haiku, what the driver assumes of Claude Code: that a skill starts a saved workflow, that a workflow's builder with worktree isolation branches from the session's worktree, that its nested reviewer works in the builder's worktree, and that the hook sees the workflow agents' types. Then it builds a fixture project of a few Packages in parallel, with one planted bug and one planted contradiction.

## Output and exit codes

- **Diagnostics** have one shape everywhere, `file:line: message` as Go's tools print them, with a stable code. With `--json`, each is an object with `severity`, `code`, `message` and the position.
- **JSON documents** share one envelope, with a `format_version` and a `kind`, so programs depend on the format and not on the wording.
- **Exit codes** follow Go's tools. 0 is success. 1 means the command did its job and found something: the spec does not compile, `status` lists something to verify, or `vet --strict` warns. Notes, such as a `skip` or an outdated harness, never change the exit code. 2 is a usage error. So a CI job runs `markitect status`, and any item fails the build.

## What this gives up, and what it keeps

| Today | With the cache |
|---|---|
| Every reached Definition is reviewed against the files as they are | Kept. A change of any input puts it back on the list, and the fingerprint `mark` takes binds the verdict to what the reviewer read |
| A changed test that served as evidence voids the answer | Kept. The evidence files are inputs |
| A changed file that realizes a Definition is reviewed | Kept |
| A changed file the ignore file lists is refused (`gate`, row 4) | **Lost.** Ignored files are not tracked |
| A changed harness or `settings.json` is refused during the work (row 1) | **Lost,** it shows in the diff; with DEC-100 the hook refuses a subagent's edit of them again, since the builder's work comes back without anyone reading that diff |
| The spec does not change during the work | Kept for subagents by the hook: only the architect, which only the owner's session starts, writes the spec. Not prevented for the owner's session or for people: a changed Definition goes back on the list, and the change shows in the diff |
| Only a reviewer records verdicts | Kept by the hook |
| The project compiles | Kept: `check`, and `status` lists the realization's errors |
| A gone Definition's files are dealt with | Kept: it stays on the list, marked gone |
| What a changed Definition said before | **Lost** from the tool; an agent reads it from Git |
| The work has a limit | No limit on rounds, by the owner's decision; a loop stops when the list has not got shorter for three rounds ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)) |
| A record of what was checked | Kept, per Definition, with its evidence |

What it gains:
- no state outside the project;
- no session that must hold a piece of work;
- no order the tool imposes;
- parallel reviewers without coordination;
- verification at any time and in any amount, by any driver, a person included.

## Decisions

Decided by the owner on 7 November 2026 ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)):

1. **The verification cache replaces the certificate, the work state, `gate`, `answer` and the reports.**
2. **The inputs** are the Definition's text, its Kind and the Definitions it links to (a Library one by its digest), its registry rows with their files, and the evidence files. `mark DEFINITION@FINGERPRINT` binds a verdict to what was read.
3. **The record** is `.markitect/review.sum`: one sorted JSON line per Definition with the verdict, the inputs' hashes, the evidence and the reason. It merges with `merge=union`.
4. **The hook** lets only the reviewer record, and lets no subagent write under `.markitect/` except the registry pages.
5. **The names:** `check`, `vet`, `list`, `doc`, `status`, `mark` and `tidy` ([refined below](#refined-after-the-plan-was-merged)); the verdicts `ok`, `fail`, `skip` and `spec-problem`; the skills `/markitect-build` and `/markitect-review`, joined by DEC-100's others ([below](#skills-roles-and-routines)).
6. **Output and exit codes as Go's tools have them**, with `vet` exiting 1 only with `--strict`.
7. **Migration.** `update` writes the record from the newest certificate. It computes each Definition's inputs from the spec and file hashes that the certificate holds, and records every Definition as `ok` at that state, with the certificate's pointers where it has them. A `not shown` answer becomes `skip`. For one release, `checklist`, `answer` and `gate` remain as stubs that say to run `markitect update`, so an old project's skill does not fail silently.

### Refined after the plan was merged

The owner refined three details on 7 November 2026, after this page was merged:

8. **The recording command is `mark`, not `review`:** `markitect mark DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]… [-m "WHY"]`, as in `markitect mark members/limit@3f9a1c ok -e src/loans.go:42 -e tests/loans_test.go:88` or `markitect mark loans/renewal@81be07 fail -e src/renew.go:30 -m "allows a fourth renewal; the Definition says three"`. `status` prints each item as `DEFINITION@FINGERPRINT`, so the reviewer can copy it. Evidence is `path:line` or `path`; a test name in parentheses is dropped. The hook lets only `markitect-reviewer` run `markitect mark`. The skill `/markitect-review` and the agent `markitect-reviewer` keep their names.
9. **The commands stay flat,** with no groups.
10. **`markitect lock` becomes `markitect tidy`,** like `go mod tidy`: it brings the lock in line with the manifest. `check` only reads the lock and, on a mismatch, says to run `markitect tidy`; `init` and `update` keep the lock current. It is built with the other names in step 2.

Concept 4.0 is revised in the same change as the code, since its §3, §5 to §9 and §12 describe the work and `gate`. The project workflow, the model language, the README and [the work](../../internal/run/README.md) change with it.

## Order of work

1. **Build the cache.**
   - Build the inputs, the record, `status`, `mark`, the hook's two rules and the migration.
   - Remove `gate`, `answer`, `checklist`, `impact`, the work state, certificates and Result pages.
   - Rewrite the skills and agents as prompts under their new names, and add the stubs.

   One PR and a release, since the harness and the binary change together.
2. **Output and the other names.** The JSON envelope, the diagnostics, the exit codes, `vet`, `list`, `doc` and `tidy`. One PR.
3. **Try it** on a small project with planted defects, under a bounded dispatch. Include a weakened test that served as evidence, and a builder's edit of the spec. The Makerspace of step 3 can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks).
