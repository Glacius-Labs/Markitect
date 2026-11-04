# Project workflow

This is the one guide for using Markitect on a project. It walks through the commands of [concept 3.0](../concepts/v3-concept.md): the model commands `init`, `check`, `lock`, `refs`, `graph`, `impact` and `lint`, which start no agent, and `apply` and `verify`, which start a Command. The [model language guide](model-language.md) owns the model commands' details: the `.markitect/` layout, Definitions, Kinds and Libraries, links, the registry and every option. The [Command engine](../../internal/run/README.md) owns how a Command works, its Result and landing.

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
markitect apply  --repo PATH [--base COMMIT] [--full] [--findings FILE] [--remote NAME] [--branch NAME]
markitect verify --repo PATH [--base COMMIT] [--full] [--findings-out FILE] [--remote NAME] [--branch NAME]
```

Commit and push the spec change first: a Command starts from the tip of the target branch, never from the working tree, and lands by pushing to it; `apply` refuses to start when the spec has uncommitted changes. The target is the current branch's upstream by default. `apply` starts one container. An implementer changes a clean copy of the project until it matches the spec; the Command commits that copy and runs its gate with no agent: the spec and `execution.yaml` unchanged, `check` passing, no stale registry entry, the required checks passing. A fresh reviewer then answers every checklist item with pointers, and the Command checks the answers. A refusal goes back to the implementer, at most twice. The work lands when the gate passed and no answer is `wrong`; you then pull it like any colleague's push. `verify` starts the reviewer only and changes nothing. The base is the last commit Markitect landed; name one with `--base` for the first Command, or use `--full`. Both print the Result page (`--json` prints JSON) and commit it under `.markitect/results/`: with the landing, or on the branch `markitect/runs/<run>` when the Command does not land, with the reason. `verify --findings-out FILE` writes its Findings for `apply --findings FILE`. Since impact does not follow the code's own imports, run a full check, `verify --full`, at every milestone you name and when a Result says it is due: every Result counts the changes landed since the last full check, and from ten on says one is due. A landed `apply --full` counts as a full check too; nothing starts one by itself ([the full-check count](../../internal/run/README.md#the-result)).

`--run` names the Command (lower-case letters, digits, `.`, `_`, `-`); by default it is named from the time with a random suffix, such as `r20261029-015416-3fa9c1`, so Commands started in the same second get different names. `--deadline` (6 hours) bounds the whole Command and `--session-timeout` (3 hours) one session; when a bound is reached, the Command stops as incomplete and lands nothing. The host runs natively on Windows, Linux and macOS and needs only Docker 28 or newer (Docker Desktop on Windows and macOS), an image and a token; no folder of yours is mounted into a container. Build the image `FROM` `packaging/runtime` with the project's toolchain and name it in `MARKITECT_MANAGER_IMAGE`. Make a token with `claude setup-token` and keep it in `MARKITECT_CLAUDE_TOKEN`; Markitect gives it only to each session's environment and never to the required checks, and you should not set the CLI's own variable for the whole machine. The provider is Claude Code; Codex has no token route yet. The [agent sessions](../../internal/agents/README.md#composition) guide lists every setting. Commands on one branch need no lock: if the branch moved, landing merges only when the incoming changes touch nothing in the Command's impact, and otherwise keeps the work on its branch ([landing](../../internal/run/README.md#landing)). A kept branch is deleted when a later landing covers its impact. Without an image or a token both commands say so and start nothing.

Every command except `init` and `lock` takes `--json`. Exit code 0 means success, 1 that the command ran and found errors, 2 that it could not run. For `apply` 0 means landed or nothing to do, for `verify` that every item conforms; 1 means the Command ran and did not land, or found something.

## Troubleshooting

| Observation | Response |
| --- | --- |
| `file.unregistered` | List the file on its owner's registry page, or on the ignore page with a reason. |
| `lock.mismatch` after an upgrade of `markitect` | A standard Library changed. Run `markitect lock`, review the lock diff and commit it. |
| `package.cycle` | Move the shared concept into a common parent, or invert the dependency ([links, scope and visibility](model-language.md#links-scope-and-visibility)). |
| `lint.no_paths` | A Package owns no file. Name its first path patterns on its registry page ([first patterns](model-language.md#registry-pages)). |
| `lint.stale` | A registry page still names a removed Definition, file or Package. The next Command puts it on its checklist, and its gate refuses work that leaves it. |
| `impact` refuses to run | The project has compile errors; run `check` and fix them first. |
| `apply` ends as `check-failed` | The start commit does not compile; no agent started. Fix the spec, commit, push and run again. |
| `apply` ends as `kept` | The work passed but did not land; the Result on `markitect/runs/<run>` says why, for example that the target moved over the Command's impact. Run `apply` again from the new state; its landing deletes the kept branch when it covers its impact. |
| `apply` ends as `refused` | The repair rounds ran out. The Result lists what stays open, and the work is on `markitect/runs/<run>`. |
| `apply` or `verify` ends as `incomplete` | A deadline, a provider failure (the Result names its class), a required check that could not start because the image lacks the program or service, or reviewer answers the Command could not accept. Nothing landed. |
| `apply` refuses a name | A container or volume of that `--run` name remains from an earlier Command; remove it or name another. |
| `apply` ends as `spec-problem` | An implementer or a reviewer found that the spec contradicts itself or leaves open something that changes behaviour at the boundary. The Result's decisions for the owner say what to settle; change the spec and run again. |
| "no Run has landed on BRANCH yet" | Name the first base with `--base COMMIT`, or use `--full`. |
