# Project workflow

This is the one guide for using Markitect on a project. It walks through the commands of [draft concept 4.0](../concepts/v4-concept.md), which this branch builds: the model commands `init`, `check`, `lock`, `refs`, `graph`, `impact` and `lint`, which start no agent, and `apply` and `verify`, which start a Command. The [model language guide](model-language.md) owns the model commands' details: the `.markitect/` layout, Definitions, Kinds and Libraries, links, the registry and every option. The [Command engine](../../internal/run/README.md) owns how a Command works, its Result and landing.

The commands run wherever Go and Git run, on Windows, Linux and macOS. Build them from `cmd/markitect`, or run them from source:

```text
go run ./cmd/markitect help
```

The [README](../../README.md#install) gives the ways to get the `markitect` command: the Claude Code plugin, a binary from the releases, or `go install`. After you install a binary, you can run `markitect install` once. It writes a personal skill, `~/.claude/skills/markitect/SKILL.md`, so a Claude Code session in any folder knows what Markitect is and how to begin; the plugin brings the same skill itself. Runs don't read it, and nothing depends on it: `init` says the same and gives the project its own skills. Whatever the skills and agents call is `markitect` by that name, so the folder of the binary must be on the `PATH`; `version`, `install`, `init` and `update` warn when the `PATH` finds no `markitect`, or another one. [Keep it current](#8-keep-it-current) says what to do after an upgrade.

The V1 workflow with Apply, the Linux runner, runtime profiles, run configuration, state and evidence is kept at the base commit of T1: [V1 project workflow](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/guides/project-workflow.md).

## 1. Set up the project

```text
markitect init --repo PATH [--name TITLE]
```

`init` works in an empty folder too, and makes it a Git repository. It writes what is missing: the manifest with `std`, the project anchor `model/project.md` if the model folder is empty, Root's registry page, the lock, `AGENTS.md` and a `CLAUDE.md` that imports it. It installs or updates the start, modeling and Extract skills in `.claude/skills/` and `.agents/skills/`, and the harness in `.claude/`: the apply and verify skills, the builder and reviewer agents, and the hook on `answer` in `.claude/settings.json`, keeping any settings already there. It commits what it wrote and prints what Markitect is. It never changes an existing model file, `AGENTS.md` or `CLAUDE.md`, so it is safe to run again after an upgrade.

Claude Code reads `AGENTS.md` through that `CLAUDE.md`, and it finds skills only in `.claude/skills/`, so `.agents/skills/` is there for other agents such as Codex.

## 2. Write the model

Open Claude Code in the project and run `/markitect-start`. It asks what the project is, who it is for, what it must do and what it is built with, one question at a time, and starts the spec with you. A repository that already has code uses `/markitect-extract` instead. A session that was open before `init` doesn't see these skills yet, because Claude Code doesn't watch a skills folder that wasn't there when the session started: run `/reload-skills` there, or start a new session.

The spec lives under `.markitect/model/`: one Markdown file with front matter per Definition, one folder per Package. The [model language guide](model-language.md#definitions) gives the form. A project writes its own Kinds for what the standard Libraries do not cover; to use `rest` or `cli`, add it to the manifest and run `markitect lock`.

The modeling skill helps an agent write the model with you and asks first when something is open. For an existing code base, the Extract skill draws a first model from the code. You accept every model change yourself; a Command never changes it.

## 3. Register the files

Every repository file outside `.markitect/` belongs to one Package and must be listed on its registry page or in the [ignore file](model-language.md#ignore-file). Give each Package a registry page with the path patterns it owns, and list each file with the Definitions it realizes and why ([realization record](model-language.md#the-realization-record)). Shared files such as the README are owned by the lowest common parent, often Root.

## 4. Check

```text
markitect check --repo PATH
```

`check` compiles the model and the realization record and prints every compile error as `file:line: code: message`. It prints errors only and exits 1 when there is one. A new project passes: when nothing is registered and all that is missing is the work, it says that the spec compiles and nothing is built yet. The compiler sees exactly what Git sees: tracked files and untracked files Git does not ignore.

## 5. Lint and look around

```text
markitect lint  --repo PATH
markitect refs  --repo PATH TARGET
markitect graph --repo PATH
```

`lint` measures the model like code: coupling, size, unused Definitions, reach and stale realization, as warnings only. `refs` shows who links to a Definition or Package, and `graph` the Package dependency graph as a Mermaid diagram. All three also run over a project with compile errors.

## 6. See what a change reaches

```text
markitect impact --repo PATH --base COMMIT [--full] [--findings FILE]
```

`impact` compares the base commit with the working tree: the changed and reached Definitions with distance and reason, and the reached files with their owners. `--findings` adds Findings from a Findings page. A Command makes its checklist from this ([impact and checklists](model-language.md#impact-and-checklists)).

## 7. Apply and verify

```text
markitect apply  --repo PATH [--base COMMIT] [--full] [--findings FILE] [--deadline D]
markitect verify --repo PATH [--base COMMIT] [--full] [--deadline D]
```

Use them through the `/markitect-apply` and `/markitect-verify` skills in your own Claude Code session, which `init` installs. The commands only open a run and say which agent to start; the skill has your session start it. You need Git, `sh` and Claude Code, with `markitect` on the path. No Docker, token, server or background session is involved.

`apply` compiles the spec, commits it on your branch and opens a run. Your session then starts the `markitect-builder` agent, which makes the project in your folder match the spec, so you see the files change. When it thinks it's done, it calls `markitect submit`, which runs `check`, has a fresh `markitect-reviewer` agent answer the checklist, sends wrong answers back to the builder, and then commits the work on your branch. Nothing is pushed; you push when you want to. Your session is busy until the run ends, so don't change the folder meanwhile.

`apply` stops at once on an error in the spec, and on a change outside `.markitect/` that is not committed, since a run would silently leave it out. With no landed run yet, the first `apply` covers every Definition. One run at a time is open in a folder: a second `apply` or `verify` says how to go on with it. If the builder ends while the run is still open, your session starts it again.

`markitect status` says how the runs of your branch stand, and every other command tells you in one line about a run that closed. `markitect stop WHY` ends a run without landing: its work goes on the branch `markitect/runs/<run>` and also stays in your folder, uncommitted, for you to keep or drop.

`verify` opens a review of the last commit, and your session starts the `markitect-reviewer` agent. It reads your folder, so it refuses one with uncommitted changes. It changes nothing you have, and ends with a Result and Findings on the branch `markitect/runs/<run>`, the input of `apply --findings`.

The agents work under your session's permission mode and settings, so they may ask you for permission; the builder accepts file edits without asking. A hook in the project's `.claude/settings.json` lets only the reviewer store answers. So do not give a run untrusted input, and start on a project whose work is committed and pushed ([draft concept 4.0 §7](../concepts/v4-concept.md#7-how-sure-we-are)). [The process of a Run](../../internal/run/README.md) gives the details, and a run's bounds: two repair rounds, 30 calls of `submit` and, with `--deadline`, 6 hours by default.

Exit code 0 means success, 1 that the command ran and found errors, 2 that it could not run. For `apply` and `verify`, 0 means a run is open or there was nothing to do. `--json` is on the spec's commands.

## 8. Keep it current

```text
markitect version [--repo PATH]
markitect update  [--repo PATH] [--check]
```

`markitect` writes some files outside its own binary: the personal skill, and in each project the skills, the builder and reviewer agents, the hook in `.claude/settings.json` and the lock. After an upgrade of the binary they are from the older build until you update them. `update` does both: it upgrades the binary, then has the new binary bring those files in line.

`version` prints the build, then says for the personal skill and for the project in `--repo` whether it matches this binary: *up to date*, *out of date* or, for the personal skill, *not installed*. It changes nothing. Every command that reads the project (`check`, `lint`, `refs`, `graph`, `impact`, `lock`, `status`) also prints one line on standard error when the project's skills and agents differ.

`update` puts that right, the way `init` does, and never touches a spec file, `AGENTS.md` or `CLAUDE.md`:

- The **binary** is replaced by the newest [release](https://github.com/Glacius-Labs/Markitect/releases), unless it is already the newest. `update` reads the newest tag from GitHub, downloads the binary for your system, checks it against the checksums of that release and puts it in the place of the running one; on Windows the old file steps aside first and goes at the next update. The new binary then carries out the rest, since only it knows the files it ships. Without a network, `update` says so and goes on with the files. It leaves two kinds of binary alone: one the plugin manages (`MARKITECT_PLUGIN` is set), which the plugin updates with `claude plugin update markitect@markitect`, and a development build (`0.0.0-dev` or `-dev` after any version) that is older than the newest release, which only `--force` replaces; one that is newer counts as up to date. `--files` skips this step.
- The **personal skill** is refreshed if it exists. `update` does not create it; `install` does, because `install` is the one command that writes into the home folder for the first time.
- The project's **skills, agents and hook** are rewritten where they differ, files the binary no longer ships are removed, and the change is committed as "Update Markitect". Your settings in `.claude/settings.json` stay.
- The **lock** is rewritten from the manifest and left uncommitted: a changed lock means the Libraries your spec is checked against changed, so you read `git diff .markitect/lock.yaml` and commit it yourself.

`update --check` changes nothing, lists what is out of date, including a newer release, and exits 1 if anything is. Files are compared without regard to line endings, so a checkout with CRLF is not out of date. This is the one command that uses the network, and only the releases page of GitHub; `MARKITECT_RELEASES_URL` points it, and the plugin's launcher, to a mirror with the same layout.

The plugin's launcher downloads the binary of the release with the plugin's own version into the plugin's data folder, `~/.claude/plugins/data/markitect-markitect/`, on the first call of a session or at its start. If a sandboxed shell blocks the network, the session-start hook of the plugin fetches it outside the sandbox.

## Troubleshooting

| Observation | Response |
| --- | --- |
| The skills and agents of this project come from another markitect | Run `markitect update` ([keep it current](#8-keep-it-current)). |
| `markitect` is not on your PATH, or the one on the PATH is not this one | Put the folder of the binary (`go env GOPATH` plus `bin` for `go install`) first on the `PATH`; the skills and agents call `markitect` by name. |
| `update` says it could not look for a newer release | There is no network or no release yet. The files were refreshed all the same; run `update` again with a network. |
| `update` says a development build is not replaced | The binary is built from source. `markitect update --force` replaces it by the newest release; otherwise build it again from source. |
| `markitect (plugin): cannot download ...` | The plugin's launcher could not reach the releases page; a sandboxed shell may block it. Start a new session, whose start hook fetches outside the sandbox, or download the binary of the release by hand ([the README](../../README.md#install)). |
| `markitect` exists in Claude Code but not in your terminal | The plugin puts it on the path of Claude Code's shell only. Install a binary for your terminal ([the README](../../README.md#install)). |
| `file.unregistered` | List the file on its owner's registry page, or in the ignore file. |
| `lock.mismatch` after an upgrade of `markitect` | A standard Library changed. Run `markitect update` (or `markitect lock`), review the lock diff and commit it. |
| `package.cycle` | Move the shared concept into a common parent, or invert the dependency ([links, scope and visibility](model-language.md#links-scope-and-visibility)). |
| `lint.no_paths` | A Package owns no file. Name its first path patterns on its registry page ([first patterns](model-language.md#registry-pages)). |
| `stale.definition`, `stale.file`, `stale.page` | A registry page still names a removed Definition, file or Package. `markitect apply` puts it on its checklist, and `submit` refuses work that leaves it. |
| `definition.unused` | Nothing links to a Definition and no file realizes it. `apply` settles it, or the spec should not hold it. A project with nothing built yet does not report this. |
| `impact` refuses to run | The spec has errors; run `check --spec` and fix them first. |
| `apply` says the spec has errors | No agent started. Fix the spec and run it again. |
| `apply` says changes outside the spec are not committed | Commit or remove them first; a run would leave them out. |
| The builder ended but the run is still open | Your session starts it again with the same message, as `apply` said. `markitect status` shows where it stands, and `markitect stop WHY` ends it. |
| A run ends as `kept` | The work passed but your branch could not be brought up, for example because a file in the folder holds an edit of your own. The Result on `markitect/runs/<run>` says why. The next `apply` tries once more, then starts fresh. |
| A run ends as `refused` | The repair rounds or the calls of `submit` ran out. The Result lists what stays open, and the work is on `markitect/runs/<run>`. |
| A run ends as `incomplete` | A deadline, a session that said the machine lacks something it needs, or a session that kept trying to end without `submit`. Nothing landed. |
| A run ends as `spec-problem` | The implementer or the reviewer found that the spec contradicts itself or leaves open something that changes behaviour at the boundary. The Result's decisions for the owner say what to settle; change the spec and run again. |
