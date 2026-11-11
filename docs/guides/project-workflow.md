# Project workflow

This is the one guide for using Markitect on a project. It walks through the commands of the [concept](../concept.md) ([DEC-099](../decisions.md#dec-099-a-record-of-what-was-verified)): the spec commands `init`, `tidy`, `check`, `vet`, `list` and `doc`, `status` and `mark` over the record of what was verified, and the `/markitect-build` and `/markitect-review` skills, which have your Claude Code session start the agents. No command starts an agent or uses Git. The [spec language guide](spec-language.md) owns the spec commands' details: the `.markitect/` layout, Definitions, Kinds and Libraries, links, the registry and every option. [The record](../../internal/run/README.md) owns how `status` and `mark` work, and [the agents' rights](../../internal/guard/README.md) how the hook works.

The commands run wherever Go runs, on Windows, Linux and macOS. Build them from `cmd/markitect`, or run them from source:

```text
go run ./cmd/markitect help
```

The [README](../../README.md#install) gives the ways to get the `markitect` command: the Claude Code plugin, a binary from the releases, or `go install`. After you install a binary, you can run `markitect install` once. It writes a personal skill, `~/.claude/skills/markitect/SKILL.md`, so a Claude Code session in any folder knows what Markitect is and how to begin; the plugin brings the same text as a skill of its own, `markitect:getting-started`. The agents don't read it, and nothing depends on it: `init` says the same and gives the project its own skills. Whatever the skills and agents call is `markitect` by that name, so the folder of the binary must be on the `PATH`; `version`, `install`, `init` and `update` warn when the `PATH` finds no `markitect`, or another one. [Keep it current](#8-keep-it-current) says what to do after an upgrade.

## 1. Set up the project

```text
markitect init --repo PATH [--name TITLE]
```

`init` works in an empty folder too, and in a folder of a project it works on that project. It writes what is missing: the manifest with `std`, the project anchor `spec/project.md` if the spec folder is empty, Root's registry page, the lock, `AGENTS.md` and a `CLAUDE.md` that imports it. It installs or updates the architect skill in `.claude/skills/`, and the harness in `.claude/`: the build, review, cleanup, bughunt, audit, schedule and config skills, the builder, reviewer, architect and auditor agents with the model and effort of the [harness configuration](#strategies-and-configuration), the parallel workflow, and in `.claude/settings.json` the hooks and `worktree.baseRef: "head"`, unless the settings name a base, keeping any settings already there. It writes `.markitect/.gitattributes`, so that Git merges the record of what was verified by keeping both sides, and adds `.claude/worktrees/` to `.gitignore` unless a line already ignores it, since Claude Code keeps the builders' worktrees there. It makes no repository and commits nothing; commit what it wrote when you like. It prints what Markitect is. It never changes an existing spec file, `AGENTS.md` or `CLAUDE.md`, so it is safe to run again.

Claude Code reads `AGENTS.md` through that `CLAUDE.md`. Where a `CLAUDE.md` is already there and does not import it, `init` says so and leaves the file to you.

## 2. Write the spec

Open Claude Code in the project and run `/markitect-architect`. In a new project it asks what the project is, who it is for, what it must do and what it is built with, one question at a time, and starts the spec with you. In a project that already exists it draws a first spec from the project as it is, with you ([below](#a-project-that-already-exists)), and in a project that has a spec it changes it. A session that was open before `init` doesn't see the skill yet, because Claude Code doesn't watch a skills folder that wasn't there when the session started: run `/reload-skills` there, or start a new session.

The spec lives under `.markitect/spec/`: one Markdown file with front matter per Definition, one folder per Package. The [spec language guide](spec-language.md#definitions) gives the form. A project writes its own Kinds for what the standard Libraries do not cover; to use `rest` or `cli`, add it to the manifest and run `markitect tidy`.

The architect skill has your session write the spec with you and ask first when something is open. A large first draft it can hand to the `markitect-architect` agent, which writes only under `.markitect/` and hands its questions back. You accept every change of the spec yourself; the builder never changes it. `/markitect-audit` has a fresh auditor read the spec for contradictions, gaps, unclear and duplicated statements and missing links, and report them to you: it triages the unlinked mentions `markitect vet --strict` lists into missing links and ordinary words, and you add the links, with `/markitect-architect` or by hand.

### A project that already exists

The first spec of an existing project states what is as what should be ([DEC-105](../decisions.md#dec-105-a-first-spec-of-an-existing-project-states-what-is-as-what-should-be)). `/markitect-architect` reads every artifact, not only the code, proposes the Packages to you, and writes the Definitions and the registry. The spec is not a copy of the project: it states what callers rely on, and leaves the rest to the agents.

The draft is finished when a build would change nothing. So, with your word, the session commits the draft on its branch and runs `/markitect-review --all`, which reads the whole project once more. Where a Definition does not hold, the session corrects the spec if the spec misread the project, and reviews again what that put back on the list. You get the spec with `.markitect/questions.md`, which holds every point that waits for you, each with what was found, where, and what the session proposes:

- **Places where the project contradicts itself,** such as its documentation against a script, and behaviour the reviewer found that the spec leaves out. `status` lists their Definitions as `fail` or `spec-problem`. Say which side holds, and whether what is left out stays free, before you run `/markitect-build`, which repairs every `fail` toward the spec as it stands. The spec then states that side, and if the other side is a defect, it stays on the list and `/markitect-build` repairs it.
- **What the spec leaves unstated:** behaviour that nothing in the project states and that looks like an accident. Say for each whether it is intended. If it is, it enters the spec. If it is a defect, the spec states what should be instead, and `/markitect-build` repairs it.

Answer in the file or in the session. The session changes the spec as you said and removes the point, and the file goes with its last point; while it exists, `markitect status` says so. The session also names what the reviewer could not show, which the record holds as `skip`.

## 3. Register the files

Every file of the project folder outside `.markitect/` belongs to one Package and must be listed on its registry page or in the [ignore file](spec-language.md#ignore-file). Give each Package a registry page with the path patterns it owns, and list each file with the Definitions it realizes and why ([realization record](spec-language.md#the-realization-record)). Shared files such as the README are owned by the lowest common parent, often Root.

## 4. Check

```text
markitect check --repo PATH
```

`check` compiles the spec and the realization record and prints every compile error as `file:line: code: message`, the spec's first and then the realization's, and ends with one summary line: `ok: N Definitions in M Packages, K files accounted`, or the number of errors with how many are in the spec, for you to settle, and how many in the realization, which `/markitect-build` settles. It exits 1 when there is an error. A new project passes: when nothing is registered and all that is missing is the work, it says that the spec compiles and nothing is built yet. If the lock does not match the manifest, `check` says to run `markitect tidy`. The compiler sees the project folder, the nearest one with `.markitect/manifest.yaml`, without what its `.gitignore` files exclude, read as text; it uses no Git ([the folder](../../internal/run/README.md#the-folder)).

In a repository that already has files, the first `check` fails by design, because every file must belong somewhere:

```text
README.md: file.unregistered: neither registered nor ignored: list it on .markitect/registry.md, the page of its owner Root, or add it to the ignore file
1 error: 0 in the spec, for the owner to settle; 1 in the realization, so spec and code do not match yet, which /markitect-build settles
```

That is expected: `/markitect-architect` registers the files as it draws the first spec, and has a reviewer verify every Definition against the project before you get it ([§2](#a-project-that-already-exists)). From then on, `/markitect-review` records where the project deviates, and `/markitect-build` repairs it.

## 5. Vet and look around

```text
markitect vet  --repo PATH [--strict]
markitect list --repo PATH [--deps]
markitect doc  --repo PATH TARGET
```

`vet` measures the spec like code ([the warnings and notes](spec-language.md#vet-warnings-and-notes)): coupling, size and hubs as warnings, and as notes the names a Definition's prose uses without a link, long Definitions, Definitions that could move down, and hotspots, widely linked Definitions whose text keeps changing. It prints one diagnostic line per warning and one line that counts the notes, nothing when there is neither, and exits 1 only with `--strict` and a warning; `--strict` also lists the notes. `list` shows each Package with its Definitions, and with `--deps` its coupling, abstractness and distance, what each Package depends on and the links behind it. `doc` shows a Definition or Package: its text, what it links to and what links to it, the files that realize it and its record. All three also run over a project with compile errors. Unused Definitions and stale rows are errors of `check`, not warnings of `vet`.

## 6. See what is to verify

```text
markitect status --repo PATH [--all]
```

`.markitect/review.sum` records, for each Definition, the state at which a reviewer last verified it: its text, its Kind and the Definitions it links to, the files its registry rows name, and the files the reviewer gave as evidence, each by its hash. It is committed with the project, like `go.sum`. `status` compares it with the folder and lists what is to verify: every Definition with a changed input or no record yet, with what changed and where to look; every Definition recorded `fail` or `spec-problem`; every Definition gone from the spec, with the files that realized it; and the errors in the realization, such as a file the registry does not list. Each item is printed as `DEFINITION@FINGERPRINT`, six characters after the `@`, which `mark` takes. A Definition recorded `skip` is a note. `--all` lists every Definition, to verify everything again.

So a change of the spec, of a realizing file or of a test that served as evidence puts its Definitions back on the list, whoever made it: you, an agent, a commit or a merge. When `status` lists nothing, the spec compiles and every Definition is verified at its current inputs. It changes nothing. [The record](../../internal/run/README.md) gives the inputs and the rules.

## 7. Build and review

```text
/markitect-build
/markitect-review [--all]
/markitect-cleanup
/markitect-bughunt
/markitect-audit
/markitect-schedule
/markitect-config
```

Use these skills in your own Claude Code session; `init` installs them. They are plain prompts: your session starts an agent on a task, and the agent runs `markitect` as it needs it. You need Claude Code, with `markitect` on the path. Markitect itself uses no Git; your session makes the commits. No token, server or background session is involved. Only you start build, cleanup, bughunt and schedule; Claude may start review, audit and config, and the architect skill, by itself ([DEC-100](../decisions.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). Each skill that starts agents runs by a strategy: one agent at a time, or per Package in parallel ([below](#strategies-and-configuration)). What follows is the sequential way.

`/markitect-build` has your session start the `markitect-builder` agent in a worktree of its own, in the background where Claude Code allows it, so you can go on architecting in your folder. The worktree starts from the last commit of your branch, because `init` sets `worktree.baseRef` to `"head"`, so commit what should be built first; the skill asks when something is uncommitted. The builder runs `markitect status`, makes the project match the spec for what it lists and keeps the registry current. Then it starts a fresh `markitect-reviewer` agent in the same worktree, which verifies each item and records a verdict with `markitect mark`, such as `markitect mark members/limit@3f9a1c ok -e src/loans.go:42`: `ok` with its evidence, `fail` with why, `skip` when the project cannot show more, or `spec-problem` with the question for you; with an `ok` or a `skip` it counts the Definition's cases and those its evidence demonstrates (`--cases`, `--covered`). The builder repeats until `status` lists nothing or only what needs you. There is no limit on rounds, but it stops when three rounds in a row leave the list no shorter than it has been. Your session then commits the work with `.markitect/review.sum` on the worktree's branch, and when `status` is empty there, or says that only `spec-problem`s are left, which wait for you, it merges that branch into yours and removes the worktree; when anything else is left, it asks you whether to merge the work as it stands. Nothing is pushed; you push when you want to. In a folder that is not a Git repository the builder works in your folder.

`/markitect-review` has your session start only the `markitect-reviewer` agent in your folder, on what `status` lists, or on every Definition with `--all`; by the parallel strategy, one reviewer per Package. It changes no code, and your session commits the record. `/markitect-build` then repairs what failed.

`/markitect-cleanup` and `/markitect-bughunt` work like build, in a worktree of their own, and come back when they leave nothing worse: `status` there lists nothing it did not list here before. Cleanup has the builder refactor without changing what the project does, and the reviewer verify again what that put back on the list. Bughunt has the reviewer hunt beyond the list and record each bug as `fail` on the Definition it breaks, or as `spec-problem` on the nearest Definition when the spec does not cover it; the builder repairs and the reviewer verifies again. `/markitect-audit` has a fresh `markitect-auditor` read the spec, not the code, and report contradictions, gaps, unclear and duplicated statements to you, with the missing links among the unlinked mentions `vet` finds; it records nothing.

What an agent notices beside its task, and every finding of a review, goes into the record as `fail` or `spec-problem` with a reason, and nowhere else. A project that wants tests states them in its spec as an ordinary Rule, with the command that runs them; the reviewer runs them as evidence.

A verdict is bound to what the reviewer read: `mark` refuses when the inputs changed since `status` printed the fingerprint, as when the builder edits a file beside the reviewer, and it refuses evidence that is not a file of the folder. Several reviewers, such as one per Package, can record at once without losing a line. When the spec contradicts itself or leaves open behaviour at the project's boundary, the reviewer records `spec-problem`, and the Definition stays on the list until you settle the spec and it is verified again.

What you ask a skill to focus on or to leave alone, it passes on to its agents, and a builder passes it on to its reviewer. That is a request, not a barrier. What agents must not run belongs in Claude Code's permission rules, such as a `deny` rule in `.claude/settings.json`, which hold for subagents too; a rule matches a command's text, so it is no sandbox, and the Bash and the PowerShell tool each need their own rule. The reviewer runs the project's tests, also those that start services, and installs nothing: a test that needs what the machine lacks is not run.

The agents work under your session's permission mode and settings, so they may ask you for permission. A hook holds the four roles' rights for subagents: the builder writes code and the registry pages; the architect the spec and the registry pages; the reviewer and the auditor nothing; and nobody `review.sum`, which only `mark` writes, or what steers the agents: the project's Claude Code settings with `.claude/hooks/` and `.claude/rules/`, Markitect's skills, agents and workflows, `.markitect/harness.yaml`, `AGENTS.md`, `CLAUDE.md` and `CLAUDE.local.md`. Only the reviewer runs `markitect mark`, so the agent that built does not verify its own work. The builder starts only the reviewer, and only your session starts the builder, the architect or a workflow. Your own session is no subagent and stays free to change the spec. The hook reads the call as text, so it guards against an agent's mistakes, not against one that hides a command. So do not give the work untrusted input, and start on a project whose work is committed and pushed ([concept, §7](../concept.md#7-how-sure-we-are)).

### Strategies and configuration

```text
markitect config [--repo PATH] [--user] KEY VALUE
markitect config [--repo PATH] [--user] --unset KEY
markitect config [--repo PATH] KEY
markitect config [--repo PATH] --list [--json]
```

Build, review, cleanup, bughunt and audit each run by a strategy ([DEC-103](../decisions.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)):

- **`auto`,** the default: for now it works sequentially, since the parallel strategy has passed only small real runs so far; set `parallel` to use it. It is meant to choose parallel when `status` lists more than about 30 items in at least two Packages; cleanup, bughunt and audit count the spec's Definitions instead, since they look at the whole project. Build, cleanup and bughunt work in parallel only in a Git repository, since each strand needs a worktree. A short list stays in your session, where one builder settles it in one context.
- **`sequential`:** one builder, reviewer or auditor at a time, as described above.
- **`parallel`:** the workflow `.claude/workflows/markitect-parallel.js`, which `init` installs. Your session first enters a worktree of its own, so that the run has a branch of its own, then starts the workflow, which plans strands per Package, in waves along the spec's dependencies, what is depended on first. Each strand is a builder in a worktree of its own, which commits its work there, and at most 8 run at once; shared files, which Root owns, such as `go.mod`, the README or CI, and the errors in the realization are left to Root's strand, last and alone. The order follows `list --deps`, where a link to an ancestor is implicit, so no Package waits for Root, and a Package may be built beside the parent it links to. After each batch of strands a plain agent merges their branches by name into the run's branch, a reviewer per strand verifies its Package there and records, and a plain agent commits the verdicts and runs `status` again. The workflow's agents cannot start agents themselves, and an agent in a worktree may run Git only on its own, so the workflow makes these steps itself. The workflow stops when `status` lists nothing a builder can settle, or when three rounds in a row leave the list no shorter than it has been, and says what remains and which strands it could not merge, each with its branch; a step that returns nothing, as when you stop an agent, ends the run with that report too. Then your session leaves its worktree, removes the worktrees of the strands it merged, and merges the run's branch into yours by the same rules as above. Review runs one reviewer per Package in your folder, bughunt one hunter per Package before it repairs, cleanup one round over every Package, and audit one auditor per Package and one that looks for contradictions between them. Architect in another session while a parallel run lasts: this one is in the run's worktree.
- **A custom workflow,** a saved Claude Code workflow of your own in `.claude/workflows/` or `~/.claude/workflows/`, named in `workflows.NAME`. It keeps the same contract: it takes the args `mode`, `into` (the branch the work comes back to), `focus` and `all`, reads `markitect status --json`, starts only the four roles' agents for the work, and stops by the same rule. Copy `markitect-parallel.js` under a name without `markitect-` to start one, since `init` and `update` restore their own files and remove others with that prefix. A workflow in the project is a file of the project like any other: list it on Root's registry page or in the ignore file. The hook does not guard it, as it does not guard your own skills.

Claude Code asks before it starts a workflow, unless you answer "don't ask again" for `markitect-parallel` in this project; in auto mode it asks only the first time. Routines take no strategy: nobody is there to answer.

The strategies, and the agents' model and effort, are set in `harness.yaml`, at two levels: `~/.markitect/harness.yaml` for all of your projects, which is not committed, and the project's `.markitect/harness.yaml`, which is. Key by key, the project's value beats yours, and yours beats the built-in one.

| Key | Values | Built in |
|---|---|---|
| `agents.ROLE.model` | `sonnet`, `opus`, `haiku`, a full model ID or `inherit`; ROLE is `builder`, `reviewer`, `architect` or `auditor` | `sonnet` for the builder and the reviewer, `opus` for the architect and the auditor |
| `agents.ROLE.effort` | `low`, `medium`, `high`, `xhigh` or `max`. A role keeps its effort when you change its model, so set both when you move a role to another model | `high` for the builder and the reviewer, `medium` for the architect and the auditor |
| `strategy.default` | `auto`, `sequential`, `parallel` or a custom workflow's name | `auto` |
| `strategy.SKILL` | the same, for `build`, `review`, `cleanup`, `bughunt` or `audit` | `strategy.default` |
| `workflows.NAME` | the path of a custom workflow's script | |

`markitect config KEY VALUE` sets a value in the project's file, and `--user` in yours; it then writes the agents' files of the project here, whose front matter carries each role's model and effort. `--unset KEY` removes a value, `KEY` alone prints the one in force, and `--list` prints every value with where it comes from. `/markitect-config` does the same from a request in plain words, and asks whether it is for this project or for all of yours. Commit a change of the project's file with the agents' files. A value of yours reaches the project's agent files when you run `init`, `update` or `config` there, since Claude Code has no agent scope that is not committed: a team that wants the same agents for everyone sets them in the project's file.

Your own skills in `.claude/skills/` are left alone by `init` and `update`. They may start the four roles' agents, and the hook holds those agents' rights there too.

### Routines

`/markitect-schedule` sets up cleanup, bughunt and `review --all` as routines with you, through Claude Code's own scheduling: a scheduled task of the desktop app, which runs on your machine while the app is open, or a cloud routine (`/schedule`), which runs on a fresh clone of your repository and needs `markitect` installed there. Markitect schedules nothing itself. You choose which routines run when, and the source branch they start from and land on.

Each run works in a worktree or clone of its own, from the newest commit of the source branch, the remote's when it tracks one. It lands on the source branch only when three things hold. Nothing is worse: `status` lists no Definition that it does not also list on the source branch as it stands. The project's tests pass. And the change holds nothing but the agents' work: nothing under `.markitect/` but the record and the registry pages, and no file that steers the agents. Each scheduled prompt is one line that points at `.claude/skills/markitect-schedule/routine.md`, so `markitect update` reaches the routines you have already set up. If the source branch moved meanwhile, it merges it first and has the reviewer verify again what the merge put on the list. It lands by a push when the source branch tracks a remote, otherwise by moving your local branch, which Git refuses while that branch is checked out in a folder; then the run leaves its branch and says how to land it. Otherwise it leaves its branch and says why. Each run ends with a short note: what landed, what is open, and which spec-problems wait for you. A run that asks for a permission waits until someone answers, so give the task a permission mode that does not ask, or rules that allow `markitect`, Git and your test command.

Exit code 0 means success, 1 that the command ran and found something (an error, an item to verify, a refused verdict, a warning of `vet --strict`), 2 that it could not run. So a CI job can run `markitect status`, and any item fails the build. `--json` is on `check`, `vet`, `list`, `doc`, `status`, `mark` and `config --list`, and prints one document with a `format_version` and a `kind` ([the fields](spec-language.md#diagnostics-json-and-exit-codes)).

## 8. Keep it current

```text
markitect version [--repo PATH]
markitect update  [--repo PATH] [--check]
```

`markitect` writes some files outside its own binary: the personal skill, and in each project the skills, the agents, the workflow, the hooks and the worktree base in `.claude/settings.json`, `.markitect/.gitattributes`, the line for Claude Code's worktrees in `.gitignore` and the lock. After an upgrade of the binary they are from the older build until you update them. `update` does both: it upgrades the binary, then has the new binary bring those files in line.

`version` prints the build, then says for the personal skill and for the project in `--repo` whether it matches this binary: *up to date*, *out of date* or, for the personal skill, *not installed*. It changes nothing. Every command that reads the project (`check`, `vet`, `list`, `doc`, `tidy`, `status`) also prints one note on standard error when the project's skills, agents and hooks differ; it never changes the exit code.

`update` puts that right, the way `init` does, and never touches a spec file, `AGENTS.md` or `CLAUDE.md`:

- The **binary** is replaced by the newest [release](https://github.com/Glacius-Labs/Markitect/releases), unless it is already the newest. `update` reads the newest tag from GitHub, downloads the binary for your system, checks it against the checksums of that release and puts it in the place of the running one; on Windows the old file steps aside first and goes at the next update. The new binary then carries out the rest, since only it knows the files it ships. Without a network, `update` says so and goes on with the files. It leaves two kinds of binary alone: one the plugin manages (`MARKITECT_PLUGIN` is set), which the plugin updates with `claude plugin update markitect@glacius-labs`, and a development build (`0.0.0-dev` or `-dev` after any version) that is older than the newest release, which only `--force` replaces; one that is newer counts as up to date. `--files` skips this step.
- The **personal skill** is refreshed if it exists. `update` does not create it; `install` does, because `install` is the one command that writes into the home folder for the first time.
- The project's **skills, agents, workflow, hooks and `.markitect/.gitattributes`** are rewritten where they differ, the line for Claude Code's worktrees is added to `.gitignore` where it is missing, the agents with the model and effort the harness configuration gives them ([strategies and configuration](#strategies-and-configuration)), and files the binary no longer ships are removed.
- The **lock** is rewritten from the manifest, as `markitect tidy` does: a changed lock means the Libraries your spec is checked against changed, so read its change before you commit it.

`update` commits nothing; commit what it wrote when you like.

`update --check` changes nothing, lists what is out of date, including a newer release, and exits 1 if anything is. Files are compared without regard to line endings, so a checkout with CRLF is not out of date. This is the one command that uses the network, and only the releases page of GitHub; `MARKITECT_RELEASES_URL` points it, and the plugin's launcher, to a mirror with the same layout.

The plugin's launcher downloads the binary of the release with the plugin's own version into the plugin's data folder, `~/.claude/plugins/data/markitect-glacius-labs/`, on the first call, or at the start of a session in a project that has a `.markitect/` folder. In any other project the plugin does not download anything until you call `markitect`. If a sandboxed shell blocks the network, the session-start hook fetches it outside the sandbox; in a new project, where there is no `.markitect/` yet, run `markitect` once in a session without the sandbox.

## Troubleshooting

| Observation | Response |
| --- | --- |
| The skills, agents and hooks of this project come from another markitect | Run `markitect update` ([keep it current](#8-keep-it-current)). |
| `markitect` is not on your PATH, or the one on the PATH is not this one | Put the folder of the binary (`go env GOPATH` plus `bin` for `go install`) first on the `PATH`; the skills and agents call `markitect` by name. |
| `update` says it could not look for a newer release | There is no network or no release yet. The files were refreshed all the same; run `update` again with a network. |
| `update` says a development build is not replaced | The binary is built from source. `markitect update --force` replaces it by the newest release; otherwise build it again from source. |
| `markitect (plugin): cannot download ...` | The plugin's launcher could not reach the releases page; a sandboxed shell may block it. Start a new session, whose start hook fetches outside the sandbox, or download the binary of the release by hand ([the README](../../README.md#install)). |
| `markitect` exists in Claude Code but not in your terminal | The plugin puts it on the path of Claude Code's tools only. Install a binary for your terminal ([the README](../../README.md#install)). |
| `markitect: command not found` in Claude Code's Bash tool on Windows, with the plugin on | The plugin's session-start hook puts it on the Bash tool's path. Update the plugin and start a new session; a session that started before the plugin was installed has no such hook. |
| `markitect (plugin): needs the bash of Git for Windows` in PowerShell | The plugin's `markitect.cmd` runs the launcher with the bash that belongs to the `git.exe` on the `PATH`, and found none. Set `CLAUDE_CODE_GIT_BASH_PATH` to the `bash.exe` of Git for Windows, as Claude Code itself reads it. |
| `file.unregistered` | List the file on its owner's registry page, or in the ignore file. |
| `library.version` after an upgrade of `markitect` | A standard Library changed, so its version moved. Run `markitect update` (or `markitect tidy`): it names the version this binary ships in `.markitect/manifest.yaml` and writes the lock. Review both and commit them. The Definitions of its Kinds are then on the list to verify again. |
| `package.cycle` | Move the shared concept into a common parent, or invert the dependency ([links, scope and visibility](spec-language.md#links-scope-and-visibility)). |
| `vet.no_paths` | A Package owns no file. Name its first path patterns on its registry page ([first patterns](spec-language.md#registry-pages)). |
| `stale.definition`, `stale.file`, `stale.page` | A registry page still names a removed Definition, file or Package. `status` lists it for the builder, and it stays on the list until it is settled. |
| `definition.unused` | Nothing links to a Definition and no file realizes it. `/markitect-build` settles it, or the spec should not hold it. A project with nothing built yet does not report it in `check`. |
| `status` says there is no list until the spec compiles | Run `check --spec` and fix the spec first. |
| A command says the folder is in no Markitect project | No folder there or above holds `.markitect/manifest.yaml`. Run it in the project, or `markitect init` to make one. |
| `mark` says the inputs changed since `status` printed the fingerprint | A file changed after the reviewer read it, perhaps by its own test run. Run `status` again and verify the item as it is now. |
| `mark` says the record has lines it cannot read | A line of `.markitect/review.sum` was written by hand or broken. Fix or remove the lines `status` names; no agent may. |
| `status` lists a conflict | After a merge two lines match the folder with different verdicts. Verify the Definition again; the next `mark` writes the file clean. |
| An agent's call was refused by the hook | Each role may only do its own part: only the `markitect-reviewer` agent records verdicts; the builder writes code and the registry pages and starts only the reviewer; the architect writes only under `.markitect/`; the reviewer and the auditor write nothing; no agent changes the settings, the hook scripts and rules in `.claude/`, the Markitect skills and agents, `AGENTS.md`, `CLAUDE.md` or `CLAUDE.local.md`; and only your session starts the builder and the architect. A change to the spec is yours, in your own session, with `/markitect-architect`. |
| The builder ended while `status` still lists items | Look at what it said: a `spec-problem` waits for you, and when only such are left the work is on your branch already; a list that did not get shorter for three rounds needs a look; anything else you can build again with `/markitect-build`. In those cases its work stays committed on the worktree's branch until you merge it. |
| The builder's work did not come back to your branch | `status` in its worktree listed more than `spec-problem`s, or Git refused the merge because of a conflict or uncommitted changes in your folder. The work is committed on the worktree's branch (`git worktree list`): merge it yourself, or run the skill again. |
| `harness.syntax`, or `init`, `update` and `config` say a `harness.yaml` cannot be read | Fix the key or value the message names by hand: `markitect config` cannot read the file either. A broken `~/.markitect/harness.yaml` stops `init` and `update` in every project until it is fixed. |
| `markitect config --list` reports a problem | A strategy names a custom workflow that no `harness.yaml` defines. Set `workflows.NAME` to its script, or set the strategy to `auto`, `sequential` or `parallel`. |
| The agents' files are out of date after a teammate ran `update` | A value in someone's own `~/.markitect/harness.yaml` reached the committed agent files. Put the values the team shares in `.markitect/harness.yaml`, which beats everyone's own. |
| The parallel workflow stopped with `not-in-a-worktree` | The session started it in your folder. Run the skill again: it enters a worktree of its own first. |
| A parallel run left strands it could not merge | Each stays on its branch, which the run's report names. Merge it into the run's branch yourself, or run the skill again, which builds what `status` still lists. |
| A subagent's workflow call was refused by the hook | Only your own session starts a workflow, since a workflow starts agents of every role. |
| `update` says that `AGENTS.md` names a skill that is gone | An earlier `init` wrote `AGENTS.md` naming a skill that `markitect-architect` replaced. `update` never changes `AGENTS.md`: change that line yourself. |
| The builder built an older state of the spec | A worktree starts from the last commit, so commit what should be built first. If the project's settings set `worktree.baseRef` to `"fresh"`, the worktree starts from the default branch on the remote. |
