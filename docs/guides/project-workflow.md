# Project workflow

This is the one guide for using Markitect on a project. It walks through the commands of [concept 4.0](../concepts/v4-concept.md), as the [tooling plan](../concepts/tooling-plan.md) changed them ([DEC-099](../concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)): the model commands `init`, `tidy`, `check`, `vet`, `list` and `doc`, `status` and `mark` over the record of what was verified, and the `/markitect-build` and `/markitect-review` skills, which have your Claude Code session start the agents. No command starts an agent or uses Git. The [model language guide](model-language.md) owns the model commands' details: the `.markitect/` layout, Definitions, Kinds and Libraries, links, the registry and every option. [The record](../../internal/run/README.md) owns how `status`, `mark` and the hook work.

The commands run wherever Go runs, on Windows, Linux and macOS. Build them from `cmd/markitect`, or run them from source:

```text
go run ./cmd/markitect help
```

The [README](../../README.md#install) gives the ways to get the `markitect` command: the Claude Code plugin, a binary from the releases, or `go install`. After you install a binary, you can run `markitect install` once. It writes a personal skill, `~/.claude/skills/markitect/SKILL.md`, so a Claude Code session in any folder knows what Markitect is and how to begin; the plugin brings the same text as a skill of its own, `markitect:getting-started`. The agents don't read it, and nothing depends on it: `init` says the same and gives the project its own skills. Whatever the skills and agents call is `markitect` by that name, so the folder of the binary must be on the `PATH`; `version`, `install`, `init` and `update` warn when the `PATH` finds no `markitect`, or another one. [Keep it current](#8-keep-it-current) says what to do after an upgrade.

The V1 workflow with Apply, the Linux runner, runtime profiles, run configuration, state and evidence is kept at the base commit of T1: [V1 project workflow](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/guides/project-workflow.md).

## 1. Set up the project

```text
markitect init --repo PATH [--name TITLE]
```

`init` works in an empty folder too, and in a folder of a project it works on that project. It writes what is missing: the manifest with `std`, the project anchor `model/project.md` if the model folder is empty, Root's registry page, the lock, `AGENTS.md` and a `CLAUDE.md` that imports it. It installs or updates the design skill in `.claude/skills/` and `.agents/skills/`, and the harness in `.claude/`: the build, review, cleanup, bughunt, audit and schedule skills, the builder, reviewer, architect and auditor agents, and in `.claude/settings.json` the hooks and `worktree.baseRef: "head"`, unless the settings name a base, keeping any settings already there. It writes `.markitect/.gitattributes`, so that Git merges the record of what was verified by keeping both sides. It makes no repository and commits nothing; commit what it wrote when you like. It prints what Markitect is. It never changes an existing model file, `AGENTS.md` or `CLAUDE.md`, so it is safe to run again after an upgrade.

Claude Code reads `AGENTS.md` through that `CLAUDE.md`, and it finds skills only in `.claude/skills/`, so `.agents/skills/` is there for other agents such as Codex.

## 2. Write the model

Open Claude Code in the project and run `/markitect-design`. In a new project it asks what the project is, who it is for, what it must do and what it is built with, one question at a time, and starts the spec with you. In a repository that already has code it draws a first spec from the code with you, and in a project that has a spec it changes it. A session that was open before `init` doesn't see the skill yet, because Claude Code doesn't watch a skills folder that wasn't there when the session started: run `/reload-skills` there, or start a new session.

The spec lives under `.markitect/model/`: one Markdown file with front matter per Definition, one folder per Package. The [model language guide](model-language.md#definitions) gives the form. A project writes its own Kinds for what the standard Libraries do not cover; to use `rest` or `cli`, add it to the manifest and run `markitect tidy`.

The design skill helps an agent write the spec with you and asks first when something is open. A large first draft it can hand to the `markitect-architect` agent, which writes only under `.markitect/` and hands its questions back. You accept every change of the spec yourself; the builder never changes it. `/markitect-audit` has a fresh auditor read the spec for contradictions, gaps, unclear and duplicated statements, and report them to you.

## 3. Register the files

Every file of the project folder outside `.markitect/` belongs to one Package and must be listed on its registry page or in the [ignore file](model-language.md#ignore-file). Give each Package a registry page with the path patterns it owns, and list each file with the Definitions it realizes and why ([realization record](model-language.md#the-realization-record)). Shared files such as the README are owned by the lowest common parent, often Root.

## 4. Check

```text
markitect check --repo PATH
```

`check` compiles the model and the realization record and prints every compile error as `file:line: code: message`, the spec's first and then the realization's, and ends with one summary line: `ok: N Definitions in M Packages, K files accounted`, or the number of errors with how many are in the spec, for you to settle, and how many in the realization, which `/markitect-build` settles. It exits 1 when there is an error. A new project passes: when nothing is registered and all that is missing is the work, it says that the spec compiles and nothing is built yet. If the lock does not match the manifest, `check` says to run `markitect tidy`. The compiler sees the project folder, the nearest one with `.markitect/manifest.yaml`, without what its `.gitignore` files exclude, read as text; it uses no Git ([the folder](../../internal/run/README.md#the-folder)).

## 5. Vet and look around

```text
markitect vet  --repo PATH [--strict]
markitect list --repo PATH [--deps]
markitect doc  --repo PATH TARGET
```

`vet` measures the model like code: coupling, size and how far a change reaches, as warnings only ([the warnings](model-language.md#vet-warnings)). It prints one diagnostic line per warning, nothing when the spec is clean, and exits 1 only with `--strict`. `list` shows each Package with its Definitions, and with `--deps` what each Package depends on and the links behind it. `doc` shows a Definition or Package: its text, what it links to and what links to it, the files that realize it and its record. All three also run over a project with compile errors. Unused Definitions and stale rows are errors of `check`, not warnings of `vet`.

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
```

Use these skills in your own Claude Code session; `init` installs them. They are plain prompts: your session starts an agent on a task, and the agent runs `markitect` as it needs it. You need Claude Code, with `markitect` on the path. Markitect itself uses no Git; your session makes the commits. No Docker, token, server or background session is involved. Only you start build, cleanup, bughunt and schedule; Claude may start review and audit, and the design skill, by itself ([DEC-100](../concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)).

`/markitect-build` has your session start the `markitect-builder` agent in a worktree of its own, so you can go on designing in your folder. The worktree starts from the last commit of your branch, because `init` sets `worktree.baseRef` to `"head"`, so commit what should be built first; the skill asks when something is uncommitted. The builder runs `markitect status`, makes the project match the spec for what it lists and keeps the registry current. Then it starts a fresh `markitect-reviewer` agent in the same worktree, which verifies each item and records a verdict with `markitect mark`, such as `markitect mark members/limit@3f9a1c ok -e src/loans.go:42`: `ok` with its evidence, `fail` with why, `skip` when the project cannot show more, or `spec-problem` with the question for you. The builder repeats until `status` lists nothing or only what needs you. There is no limit on rounds, but it stops when three rounds in a row end without the list getting shorter. Your session then commits the work with `.markitect/review.sum` on the worktree's branch, and when `status` is empty there it merges that branch into yours and removes the worktree; otherwise it asks you whether to merge the work as it stands. Nothing is pushed; you push when you want to. In a folder that is not a Git repository the builder works in your folder.

`/markitect-review` has your session start only the `markitect-reviewer` agent in your folder, on what `status` lists, or on every Definition with `--all`; above about 30 items it starts one reviewer per Package, in parallel. It changes no code, and your session commits the record. `/markitect-build` then repairs what failed.

`/markitect-cleanup` and `/markitect-bughunt` work like build, in a worktree of their own, and come back when they leave nothing worse: `status` there lists nothing it did not list here before. Cleanup has the builder refactor without changing what the project does, and the reviewer verify again what that put back on the list. Bughunt has the reviewer hunt beyond the list and record each bug as `fail` on the Definition it breaks, or as `spec-problem` on the nearest Definition when the spec does not cover it; the builder repairs and the reviewer verifies again. `/markitect-audit` has a fresh `markitect-auditor` read the spec, not the code, and report contradictions, gaps, unclear and duplicated statements to you; it records nothing.

What an agent notices beside its task, and every finding of a review, goes into the record as `fail` or `spec-problem` with a reason, and nowhere else. A project that wants tests states them in its spec as an ordinary Rule, with the command that runs them; the reviewer runs them as evidence.

A verdict is bound to what the reviewer read: `mark` refuses when the inputs changed since `status` printed the fingerprint, as when the builder edits a file beside the reviewer, and it refuses evidence that is not a file of the folder. Several reviewers, such as one per Package, can record at once without losing a line. When the spec contradicts itself or leaves open behaviour at the project's boundary, the reviewer records `spec-problem`, and the Definition stays on the list until you settle the spec and it is verified again.

The agents work under your session's permission mode and settings, so they may ask you for permission. A hook holds the four roles' rights for subagents: the builder writes code and the registry pages; the architect, for a delegated draft, the spec and the registry pages; the reviewer and the auditor nothing; and nobody `review.sum`, which only `mark` writes. Only the reviewer runs `markitect mark`, so the agent that built does not verify its own work. The builder starts only the reviewer, and only your session starts the builder or the architect. Your own session is no subagent and stays free to change the spec. The hook reads the call as text, so it guards against an agent's mistakes, not against one that hides a command. So do not give the work untrusted input, and start on a project whose work is committed and pushed ([concept 4.0 §7](../concepts/v4-concept.md#7-how-sure-we-are)).

### Routines

`/markitect-schedule` sets up cleanup, bughunt and `review --all` as routines with you, through Claude Code's own scheduling: a scheduled task of the desktop app, which runs on your machine while the app is open, or a cloud routine (`/schedule`), which runs on a fresh clone of your repository and needs `markitect` installed there. Markitect schedules nothing itself. You choose which routines run when, and the source branch they start from and land on.

Each run works in a worktree or clone of its own, from the newest commit of the source branch, the remote's when it tracks one. It lands on the source branch only when it leaves nothing worse: `status` lists no Definition that it does not also list on the source branch as it stands, and the project's tests pass. If the source branch moved meanwhile, it merges it first and has the reviewer verify again what the merge put on the list. It lands by a push when the source branch tracks a remote, otherwise by moving your local branch, which Git refuses while that branch is checked out in a folder; then the run leaves its branch and says how to land it. Otherwise it leaves its branch and says why. Each run ends with a short note: what landed, what is open, and which spec-problems wait for you. A run that asks for a permission waits until someone answers, so give the task a permission mode that does not ask, or rules that allow `markitect`, Git and your test command.

`markitect checklist`, `answer` and `gate` went in 0.21.0 with the certificates and Result pages; for one release they only say to run `markitect update`, which writes the record from the newest certificate ([keep it current](#8-keep-it-current)). The commands of a Run, `markitect apply`, `verify`, `submit` and `stop`, went in 0.20.0 ([lean plan](../concepts/lean-tool-plan.md#order-of-work)).

Exit code 0 means success, 1 that the command ran and found something (an error, an item to verify, a refused verdict, a warning of `vet --strict`), 2 that it could not run. So a CI job can run `markitect status`, and any item fails the build. `--json` is on `check`, `vet`, `list`, `doc`, `status` and `mark`, and prints one document with a `format_version` and a `kind` ([the fields](model-language.md#diagnostics-json-and-exit-codes)). `lock`, `lint`, `refs` and `graph` are gone; they are unknown commands.

## 8. Keep it current

```text
markitect version [--repo PATH]
markitect update  [--repo PATH] [--check]
```

`markitect` writes some files outside its own binary: the personal skill, and in each project the skills, the agents, the hooks and the worktree base in `.claude/settings.json`, `.markitect/.gitattributes` and the lock. After an upgrade of the binary they are from the older build until you update them. `update` does both: it upgrades the binary, then has the new binary bring those files in line.

`version` prints the build, then says for the personal skill and for the project in `--repo` whether it matches this binary: *up to date*, *out of date* or, for the personal skill, *not installed*. It changes nothing. Every command that reads the project (`check`, `vet`, `list`, `doc`, `tidy`, `status`) also prints one note on standard error when the project's skills, agents and hooks differ; it never changes the exit code.

`update` puts that right, the way `init` does, and never touches a spec file, `AGENTS.md` or `CLAUDE.md`:

- The **binary** is replaced by the newest [release](https://github.com/Glacius-Labs/Markitect/releases), unless it is already the newest. `update` reads the newest tag from GitHub, downloads the binary for your system, checks it against the checksums of that release and puts it in the place of the running one; on Windows the old file steps aside first and goes at the next update. The new binary then carries out the rest, since only it knows the files it ships. Without a network, `update` says so and goes on with the files. It leaves two kinds of binary alone: one the plugin manages (`MARKITECT_PLUGIN` is set), which the plugin updates with `claude plugin update markitect@glacius-labs`, and a development build (`0.0.0-dev` or `-dev` after any version) that is older than the newest release, which only `--force` replaces; one that is newer counts as up to date. `--files` skips this step.
- The **personal skill** is refreshed if it exists. `update` does not create it; `install` does, because `install` is the one command that writes into the home folder for the first time.
- The project's **skills, agents, hooks and `.markitect/.gitattributes`** are rewritten where they differ, and files the binary no longer ships, such as the start, modeling and Extract skills that `markitect-design` replaced, are removed. Your settings in `.claude/settings.json` stay, and so does a `worktree.baseRef` you chose.
- The **lock** is rewritten from the manifest, as `markitect tidy` does: a changed lock means the Libraries your spec is checked against changed, so read its change before you commit it. Its header now reads "Written by markitect tidy from the manifest", so the first `update` after 0.21.0 rewrites every lock once.
- The **record**: a project from 0.20.0 has certificates under `.markitect/results/`. `update` writes `.markitect/review.sum` from the newest one, recording every Definition it certified as `ok` at the certified state, with the certificate's pointers as evidence and a `not shown` answer as `skip`, and removes the results folder, which no longer belongs to the layout. A project that already has its record keeps it.

`update` commits nothing; commit what it wrote when you like.

`update --check` changes nothing, lists what is out of date, including a newer release, and exits 1 if anything is. Files are compared without regard to line endings, so a checkout with CRLF is not out of date. This is the one command that uses the network, and only the releases page of GitHub; `MARKITECT_RELEASES_URL` points it, and the plugin's launcher, to a mirror with the same layout.

The plugin's launcher downloads the binary of the release with the plugin's own version into the plugin's data folder, `~/.claude/plugins/data/markitect-glacius-labs/`, on the first call, or at the start of a session in a project that has a `.markitect/` folder. In any other project the plugin does not download anything until you call `markitect`. If a sandboxed shell blocks the network, the session-start hook fetches it outside the sandbox; in a new project, where there is no `.markitect/` yet, run `markitect` once in a session without the sandbox.

## Troubleshooting

| Observation | Response |
| --- | --- |
| The skills, agents and hooks of this project come from another markitect | Run `markitect update` ([keep it current](#8-keep-it-current)). |
| `markitect checklist`, `answer` or `gate` says it is gone | The project's skills come from 0.20.0 or older. Run `markitect update`, then use `/markitect-build` or `/markitect-review`. |
| `project.results` | The certificates of 0.20.0 are retired. `markitect update` writes `.markitect/review.sum` from the newest one and removes `.markitect/results/`. |
| `markitect` is not on your PATH, or the one on the PATH is not this one | Put the folder of the binary (`go env GOPATH` plus `bin` for `go install`) first on the `PATH`; the skills and agents call `markitect` by name. |
| `update` says it could not look for a newer release | There is no network or no release yet. The files were refreshed all the same; run `update` again with a network. |
| `update` says a development build is not replaced | The binary is built from source. `markitect update --force` replaces it by the newest release; otherwise build it again from source. |
| `markitect (plugin): cannot download ...` | The plugin's launcher could not reach the releases page; a sandboxed shell may block it. Start a new session, whose start hook fetches outside the sandbox, or download the binary of the release by hand ([the README](../../README.md#install)). |
| `markitect` exists in Claude Code but not in your terminal | The plugin puts it on the path of Claude Code's shell only. Install a binary for your terminal ([the README](../../README.md#install)). |
| `file.unregistered` | List the file on its owner's registry page, or in the ignore file. |
| `lock.mismatch` after an upgrade of `markitect` | A standard Library changed. Run `markitect update` (or `markitect tidy`), review the lock diff and commit it. The Definitions of its Kinds are then on the list to verify again. |
| `package.cycle` | Move the shared concept into a common parent, or invert the dependency ([links, scope and visibility](model-language.md#links-scope-and-visibility)). |
| `vet.no_paths` | A Package owns no file. Name its first path patterns on its registry page ([first patterns](model-language.md#registry-pages)). |
| `stale.definition`, `stale.file`, `stale.page` | A registry page still names a removed Definition, file or Package. `status` lists it for the builder, and it stays on the list until it is settled. |
| `definition.unused` | Nothing links to a Definition and no file realizes it. `/markitect-build` settles it, or the spec should not hold it. A project with nothing built yet does not report it in `check`. |
| `status` says there is no list until the spec compiles | Run `check --spec` and fix the spec first. |
| A command says the folder is in no Markitect project | No folder there or above holds `.markitect/manifest.yaml`. Run it in the project, or `markitect init` to make one. |
| `mark` says the inputs changed since `status` printed the fingerprint | A file changed after the reviewer read it, perhaps by its own test run. Run `status` again and verify the item as it is now. |
| `mark` says the record has lines it cannot read | A line of `.markitect/review.sum` was written by hand or broken. Fix or remove the lines `status` names; no agent may. |
| `status` lists a conflict | After a merge two lines match the folder with different verdicts. Verify the Definition again; the next `mark` writes the file clean. |
| An agent's call was refused by the hook | Each role may only do its own part: only the `markitect-reviewer` agent records verdicts; the builder writes code and the registry pages and starts only the reviewer; the architect writes only under `.markitect/`; the reviewer and the auditor write nothing; and only your session starts the builder and the architect. A change to the spec is yours, in your own session. |
| The builder ended while `status` still lists items | Look at what it said: a `spec-problem` waits for you; a list that did not get shorter for three rounds needs a look; anything else you can build again with `/markitect-build`. Its work stays committed on the worktree's branch until you merge it. |
| The builder's work did not come back to your branch | `status` in its worktree was not empty, or Git refused the merge because of a conflict or uncommitted changes in your folder. The work is committed on the worktree's branch (`git worktree list`): merge it yourself, or run the skill again. |
| The builder built an older state of the spec | A worktree starts from the last commit, so commit what should be built first. If the project's settings set `worktree.baseRef` to `"fresh"`, the worktree starts from the default branch on the remote. |
