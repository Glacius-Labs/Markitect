# Project workflow

This is the one guide for using Markitect on a project. It walks through the commands of [draft concept 4.0](../concepts/v4-concept.md), as the [lean plan](../concepts/lean-tool-plan.md) changed them: the model commands `init`, `check`, `lock`, `refs`, `graph`, `impact` and `lint`, and the `/markitect-apply` and `/markitect-verify` skills, which have your Claude Code session start the agents while `markitect gate` checks their work. No command starts an agent or uses Git. The [model language guide](model-language.md) owns the model commands' details: the `.markitect/` layout, Definitions, Kinds and Libraries, links, the registry and every option. [The work](../../internal/run/README.md) owns how `checklist`, `answer` and `gate` work, the certificate and the Result.

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

`init` works in an empty folder too, and in a folder of a project it works on that project. It writes what is missing: the manifest with `std`, the project anchor `model/project.md` if the model folder is empty, Root's registry page, the lock, `AGENTS.md` and a `CLAUDE.md` that imports it. It installs or updates the start, modeling and Extract skills in `.claude/skills/` and `.agents/skills/`, and the harness in `.claude/`: the apply and verify skills, the builder and reviewer agents, and the hook on `answer` and `checklist` in `.claude/settings.json`, keeping any settings already there. It makes no repository and commits nothing; commit what it wrote when you like. It prints what Markitect is. It never changes an existing model file, `AGENTS.md` or `CLAUDE.md`, so it is safe to run again after an upgrade.

Claude Code reads `AGENTS.md` through that `CLAUDE.md`, and it finds skills only in `.claude/skills/`, so `.agents/skills/` is there for other agents such as Codex.

## 2. Write the model

Open Claude Code in the project and run `/markitect-start`. It asks what the project is, who it is for, what it must do and what it is built with, one question at a time, and starts the spec with you. A repository that already has code uses `/markitect-extract` instead. A session that was open before `init` doesn't see these skills yet, because Claude Code doesn't watch a skills folder that wasn't there when the session started: run `/reload-skills` there, or start a new session.

The spec lives under `.markitect/model/`: one Markdown file with front matter per Definition, one folder per Package. The [model language guide](model-language.md#definitions) gives the form. A project writes its own Kinds for what the standard Libraries do not cover; to use `rest` or `cli`, add it to the manifest and run `markitect lock`.

The modeling skill helps an agent write the model with you and asks first when something is open. For an existing code base, the Extract skill draws a first model from the code. You accept every model change yourself; the builder never changes it.

## 3. Register the files

Every file of the project folder outside `.markitect/` belongs to one Package and must be listed on its registry page or in the [ignore file](model-language.md#ignore-file). Give each Package a registry page with the path patterns it owns, and list each file with the Definitions it realizes and why ([realization record](model-language.md#the-realization-record)). Shared files such as the README are owned by the lowest common parent, often Root.

## 4. Check

```text
markitect check --repo PATH
```

`check` compiles the model and the realization record and prints every compile error as `file:line: code: message`. It prints errors only and exits 1 when there is one. A new project passes: when nothing is registered and all that is missing is the work, it says that the spec compiles and nothing is built yet. The compiler sees the project folder, the nearest one with `.markitect/manifest.yaml`, without what its `.gitignore` files exclude, read as text; it uses no Git ([the folder](../../internal/run/README.md#the-folder)).

## 5. Lint and look around

```text
markitect lint  --repo PATH
markitect refs  --repo PATH TARGET
markitect graph --repo PATH
```

`lint` measures the model like code: coupling, size, unused Definitions, reach and stale realization, as warnings only. `refs` shows who links to a Definition or Package, and `graph` the Package dependency graph as a Mermaid diagram. All three also run over a project with compile errors.

## 6. See what a change reaches

```text
markitect impact --repo PATH [--base CERTIFICATE] [--full] [--findings FILE]
```

`impact` compares the folder with the newest certificate under `.markitect/results/`, or the one `--base` names by its `.json` or `.md`: the changed and reached Definitions with distance and reason, and the reached files with their owners. A project with no certificate yet has no base, and its first `/markitect-apply` covers every Definition. `--findings` adds Findings from a Findings page. `checklist` makes its list from this ([impact and checklists](model-language.md#impact-and-checklists)).

## 7. Apply and verify

```text
/markitect-apply  [--base CERTIFICATE] [--full] [--findings FILE]
/markitect-verify [--base CERTIFICATE] [--full]
```

Use the `/markitect-apply` and `/markitect-verify` skills in your own Claude Code session, which `init` installs. They pass their arguments to `markitect checklist --start`, which begins the work and says which agent to start, and your session starts it. You need Claude Code, with `markitect` on the path. Markitect itself uses no Git; your session makes the commit. No Docker, token, server or background session is involved.

`/markitect-apply` begins the work with the list of what changed since the newest certificate under `.markitect/results/`, or of every Definition when there is none yet; an error in the spec stops it here. Your session then starts the `markitect-builder` agent, which makes the project in your folder match the spec, so you see the files change. When it thinks it's done, it calls `markitect gate`, which runs `check`, has it start a fresh `markitect-reviewer` agent that answers the checklist, and sends wrong answers back to it. When every item has an answer and none is wrong, `gate` writes a certificate with its Result page, and your session commits the work with both in one commit on your branch. Nothing is pushed; you push when you want to. If the builder ends before that, your session runs `gate` and does what it prints, starting the builder or the reviewer again.

Your session is busy until the work ends, so don't change the folder meanwhile: `gate` refuses a changed file that lies in no item of the list, and a change of the spec, the harness, a `.gitignore` or a certificate during the work. A change to the spec is its own step, before the work. When the builder finds that the spec contradicts itself or leaves open behaviour at the project's boundary, it reports it with `markitect gate --spec-problem "WHY"`; a reviewer confirms it before you hear of it, so no agent decides about its own work, and that uses no repair round. If the reviewer finds the spec settles it, the builder builds on. When `gate` stops the work, on a problem in the spec or when the repair rounds run out, change the spec and run `/markitect-apply` again: after a change to the spec it begins the work anew. `markitect status` says how the work stands.

`/markitect-verify` has a fresh `markitect-reviewer` agent answer the checklist of what changed since the newest certificate, or of every Definition with `--full`, and changes nothing. `markitect gate --findings` then writes its Result and, if it found something, a Findings page under `.markitect/results/`, the input of `/markitect-apply --findings FILE`; your session commits them.

The agents work under your session's permission mode and settings, so they may ask you for permission; A hook lets only the reviewer store answers, and refuses an agent's `checklist` that would begin the work or name a change. It reads the command as text, so it guards against an agent's mistakes, not against one that hides a command. `/markitect-apply` refuses to begin in a project whose agents or hook are missing or old: run `markitect update`. So do not give the work untrusted input, and start on a project whose work is committed and pushed ([draft concept 4.0 §7](../concepts/v4-concept.md#7-how-sure-we-are)). [The work](../../internal/run/README.md#gate) gives gate's rows, the certificate and the one bound the tool keeps, two repair rounds; time and the number of calls are your session's.

The commands of a Run, `markitect apply`, `verify`, `submit` and `stop`, with run branches, landing and the `Run-*` trailers, went in 0.20.0 ([lean plan](../concepts/lean-tool-plan.md#order-of-work)). Old runs' branches and their Result pages stay where they are; nothing reads them any more.

Exit code 0 means success, 1 that the command ran and found errors, 2 that it could not run. `checklist --start` always exits 0, so that the skill's session sees what it printed. `--json` is on the spec's commands.

## 8. Keep it current

```text
markitect version [--repo PATH]
markitect update  [--repo PATH] [--check]
```

`markitect` writes some files outside its own binary: the personal skill, and in each project the skills, the builder and reviewer agents, the hook in `.claude/settings.json` and the lock. After an upgrade of the binary they are from the older build until you update them. `update` does both: it upgrades the binary, then has the new binary bring those files in line.

`version` prints the build, then says for the personal skill and for the project in `--repo` whether it matches this binary: *up to date*, *out of date* or, for the personal skill, *not installed*. It changes nothing. Every command that reads the project (`check`, `lint`, `refs`, `graph`, `impact`, `lock`, `status`) also prints one line on standard error when the project's skills and agents differ.

`update` puts that right, the way `init` does, and never touches a spec file, `AGENTS.md` or `CLAUDE.md`:

- The **binary** is replaced by the newest [release](https://github.com/Glacius-Labs/Markitect/releases), unless it is already the newest. `update` reads the newest tag from GitHub, downloads the binary for your system, checks it against the checksums of that release and puts it in the place of the running one; on Windows the old file steps aside first and goes at the next update. The new binary then carries out the rest, since only it knows the files it ships. Without a network, `update` says so and goes on with the files. It leaves two kinds of binary alone: one the plugin manages (`MARKITECT_PLUGIN` is set), which the plugin updates with `claude plugin update markitect@glacius-labs`, and a development build (`0.0.0-dev` or `-dev` after any version) that is older than the newest release, which only `--force` replaces; one that is newer counts as up to date. `--files` skips this step.
- The **personal skill** is refreshed if it exists. `update` does not create it; `install` does, because `install` is the one command that writes into the home folder for the first time.
- The project's **skills, agents and hook** are rewritten where they differ, and files the binary no longer ships are removed. Your settings in `.claude/settings.json` stay.
- The **lock** is rewritten from the manifest: a changed lock means the Libraries your spec is checked against changed, so read its change before you commit it.

`update` commits nothing; commit what it wrote when you like.

`update --check` changes nothing, lists what is out of date, including a newer release, and exits 1 if anything is. Files are compared without regard to line endings, so a checkout with CRLF is not out of date. This is the one command that uses the network, and only the releases page of GitHub; `MARKITECT_RELEASES_URL` points it, and the plugin's launcher, to a mirror with the same layout.

The plugin's launcher downloads the binary of the release with the plugin's own version into the plugin's data folder, `~/.claude/plugins/data/markitect-glacius-labs/`, on the first call, or at the start of a session in a project that has a `.markitect/` folder. In any other project the plugin does not download anything until you call `markitect`. If a sandboxed shell blocks the network, the session-start hook fetches it outside the sandbox; in a new project, where there is no `.markitect/` yet, run `markitect` once in a session without the sandbox.

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
| `stale.definition`, `stale.file`, `stale.page` | A registry page still names a removed Definition, file or Package. `/markitect-apply` puts it on its checklist, and `gate` refuses work that leaves it. |
| `definition.unused` | Nothing links to a Definition and no file realizes it. `/markitect-apply` settles it, or the spec should not hold it. A project with nothing built yet does not report this. |
| `impact` refuses to run | The spec has errors; run `check --spec` and fix them first. Or there is no certificate yet, so there is no base. |
| `/markitect-apply` says the spec has errors | No agent started. Fix the spec and run it again. |
| A command says the folder is in no Markitect project | No folder there or above holds `.markitect/manifest.yaml`. Run it in the project, or `markitect init` to make one. |
| `gate` names files that lie in no item of the list | A file changed that nobody reviews: your own edit beside the work, or a builder that touched more than its items. Put it back, or list it on the registry so that it is reviewed. |
| The builder ended before `gate` passed | Your session runs `gate` and starts the agent it names, as the skill says. `markitect status` shows where the work stands. |
| `gate` stops as `refused` | The reviewer still found something wrong after two repair rounds. The Result lists what stays open, and the work stays in the folder, uncommitted. |
| `gate` stops as `spec-problem` | A reviewer found, or confirmed the builder's report, that the spec contradicts itself or leaves open something that changes behaviour at the boundary. The Result's decisions for the owner say what to settle; change the spec and run `/markitect-apply` again. |
