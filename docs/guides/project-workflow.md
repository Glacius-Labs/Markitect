# Project workflow

This is the one guide for using Markitect on a project. It walks through the commands of [draft concept 4.0](../concepts/v4-concept.md), which this branch builds: the model commands `init`, `check`, `lock`, `refs`, `graph`, `impact` and `lint`, which start no agent, and `apply` and `verify`, which start a Command. The [model language guide](model-language.md) owns the model commands' details: the `.markitect/` layout, Definitions, Kinds and Libraries, links, the registry and every option. The [Command engine](../../internal/run/README.md) owns how a Command works, its Result and landing.

The commands run wherever Go and Git run, on Windows, Linux and macOS. Build them from `cmd/markitect`, or run them from source:

```text
go run ./cmd/markitect help
```

The V1 workflow with Apply, the Linux runner, runtime profiles, run configuration, state and evidence is kept at the base commit of T1: [V1 project workflow](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/guides/project-workflow.md).

## 1. Set up the project

```text
markitect init --repo PATH [--name TITLE]
```

`init` writes what is missing: the manifest with `std`, `execution.yaml` with no checks, the project anchor `model/project.md` if the model folder is empty, Root's registry page, the ignore page and the lock. It installs or updates the modeling and Extract skills in `.claude/skills/` and `.agents/skills/` and lists them on the ignore page. It never changes an existing model file, so it is safe to run again after an upgrade. Commit its result.

## 2. Write the model

Write the model under `.markitect/model/`: one Markdown file with front matter per Definition, one folder per Package. The [model language guide](model-language.md#definitions) gives the form. A project writes its own Kinds for what the standard Libraries do not cover; to use `rest` or `cli`, add it to the manifest and run `markitect lock`.

The modeling skill helps an agent write the model with you and asks first when something is open. For an existing code base, the Extract skill draws a first model from the code. You accept every model change yourself; a Command never changes it.

## 3. Register the files

Every repository file outside `.markitect/` belongs to one Package and must be listed on its registry page or on the ignore page. Give each Package a registry page with the path patterns it owns, and list each file with the Definitions it realizes and why ([realization record](model-language.md#the-realization-record)). Shared files such as the README are owned by the lowest common parent, often Root. List the required checks in `.markitect/execution.yaml`.

## 4. Check

```text
markitect check --repo PATH
```

`check` compiles the model and the realization record and prints every compile error as `file:line: code: message`. It prints errors only and exits 1 when there is one. The compiler sees exactly what Git sees: tracked files and untracked files Git does not ignore.

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

Or use the `/markitect-apply` and `/markitect-verify` skills in your own Claude Code session, which `init` installs and which call these commands. You need Git, `sh` and Claude Code on your own login, with `markitect` on the path. No Docker, token or server is involved.

`apply` compiles the spec, commits it on your branch, starts a background session in a worktree of its own on that commit, and returns at once. You keep modeling: the run works on its frozen copy. The session makes the project match the spec, and calls `markitect submit`, which runs `check` and the required checks in a clean copy, has a fresh reviewer answer the checklist, and then brings your branch up by itself. It asks you nothing. Nothing is pushed; you push when you want to.

`apply` stops at once on an error in the spec, and on a change outside `.markitect/` that is not committed, since a run would silently leave it out. With no landed run yet, the first `apply` covers every Definition. A second `apply` during a run starts nothing; one whose session has stopped working, for example after a reboot or a usage limit, is resumed where it was.

`markitect status` says how the runs of your branch stand, and every other command tells you in one line about a run that closed or stopped working. `claude agents` lists the sessions, and `claude attach ID` opens one. `markitect stop WHY` ends a run without landing.

`verify` starts a reviewer's session on the last commit. It changes nothing you have, does not mind uncommitted work and does not look at it, and ends with a Result and Findings on the branch `markitect/runs/<run>`, the input of `apply --findings`.

A run may edit files and run any shell command as you, on your machine, in `dontAsk` mode, with `git push` and `gh` refused and the provider's checks keeping it out of your checkout. So do not give a run untrusted input, and start on a project whose work is committed and pushed ([draft concept 4.0 §7](../concepts/v4-concept.md#7-how-sure-we-are)). [The process of a Run](../../internal/run/README.md) gives the details, and a run's bounds: two repair rounds, 30 calls of `submit` and, with `--deadline`, 6 hours by default.

Exit code 0 means success, 1 that the command ran and found errors, 2 that it could not run. For `apply` and `verify`, 0 means a run started or there was nothing to do. `--json` is on the spec's commands.

## Troubleshooting

| Observation | Response |
| --- | --- |
| `file.unregistered` | List the file on its owner's registry page, or on the ignore page with a reason. |
| `lock.mismatch` after an upgrade of `markitect` | A standard Library changed. Run `markitect lock`, review the lock diff and commit it. |
| `package.cycle` | Move the shared concept into a common parent, or invert the dependency ([links, scope and visibility](model-language.md#links-scope-and-visibility)). |
| `lint.no_paths` | A Package owns no file. Name its first path patterns on its registry page ([first patterns](model-language.md#registry-pages)). |
| `stale.definition`, `stale.file`, `stale.page` | A registry page still names a removed Definition, file or Package. `markitect apply` puts it on its checklist, and `submit` refuses work that leaves it. |
| `definition.unused` | Nothing links to a Definition and no file realizes it. `apply` settles it, or the spec should not hold it. |
| `impact` refuses to run | The spec has errors; run `check --spec` and fix them first. |
| `apply` says the spec has errors | No agent started. Fix the spec and run it again. |
| `apply` says changes outside the spec are not committed | Commit or remove them first; a run would leave them out. |
| `Claude Code does not trust this folder` | A background session asks for its own trust, even in a folder that opens without asking because a parent is trusted. Run `claude --bg "Reply with one word."` once in a terminal in that folder and accept its question. |
| A run ends as `kept` | The work passed but your branch could not be brought up, for example because your uncommitted edits overlap or a merge conflicts. The Result on `markitect/runs/<run>` says why. The next `apply` tries once more, then starts fresh. |
| A run ends as `refused` | The repair rounds or the calls of `submit` ran out. The Result lists what stays open, and the work is on `markitect/runs/<run>`. |
| A run ends as `incomplete` | A deadline, a required check that could not start because a program is missing, or a session that kept trying to end without `submit`. Nothing landed. |
| A run ends as `spec-problem` | The implementer or the reviewer found that the spec contradicts itself or leaves open something that changes behaviour at the boundary. The Result's decisions for the owner say what to settle; change the spec and run again. |
