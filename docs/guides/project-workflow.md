# Project workflow

This is the one guide for using Markitect on a project. It walks through the commands of [concept 2.0](../concepts/v2-concept.md) that exist today: `init`, `check`, `lock`, `refs`, `graph`, `impact` and `lint`. None of them starts an agent. The [model language guide](model-language.md) owns the details: the `.markitect/` layout, Definitions, Kinds and Libraries, links, the registry and every command's options. `apply` and `verify`, which start agents, come with step T4 of the [delivery plan](../implementation-plan.md#concept-20-build).

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

The modeling skill helps an agent write the model with you and asks first when something is open. For an existing code base, the Extract skill draws a first model from the code. You accept every model change yourself.

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

`impact` compares the base commit with the working tree: the changed and reached Definitions, the reached files with their owners, the Managers that would work, and each Manager's checklist. `--findings` adds Findings from a Findings page. T4's `apply` and `verify` will start their agents from this checklist ([impact and checklists](model-language.md#impact-and-checklists)).

Every command except `init` and `lock` takes `--json`. Exit code 0 means success, 1 that the command ran and found errors, 2 that it could not run.

## Troubleshooting

| Observation | Response |
| --- | --- |
| `file.unregistered` | List the file on its owner's registry page, or on the ignore page with a reason. |
| `lock.mismatch` after an upgrade of `markitect` | A standard Library changed. Run `markitect lock`, review the lock diff and commit it. |
| `package.cycle` | Move the shared concept into a common parent, or invert the dependency ([links, scope and visibility](model-language.md#links-scope-and-visibility)). |
| `lint.stale` | A registry page still names a removed Definition, file or Package. The page's Package cleans it up; `impact` puts it on that Manager's checklist. |
| `impact` refuses to run | The project has compile errors; run `check` and fix them first. |
