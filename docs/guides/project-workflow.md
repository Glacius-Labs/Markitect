# Project workflow

This is the one guide for using Markitect on a project. It walks through the model commands `init`, `check`, `lock`, `refs`, `graph`, `impact` and `lint`, which start no agent. There is no `apply` or `verify` today: how agents realize a model is being redesigned ([DEC-094](../concepts/register.md#dec-094-a-fresh-start-for-realization)). The [model language guide](model-language.md) owns the details of the commands: the `.markitect/` layout, Definitions, Kinds and Libraries, links, the registry and every option.

The commands run wherever Go and Git run, on Windows, Linux and macOS. Build them from `cmd/markitect`, or run them from source:

```text
go run ./cmd/markitect help
```

The earlier workflows with `apply` are kept at archive points: [V1 project workflow](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/docs/guides/project-workflow.md) and the [workflow with `apply` at the pre-native point](https://github.com/Glacius-Labs/Markitect/blob/10b074e4fe82c70c8219fb2fa5a0bacee4986ee1/docs/guides/project-workflow.md), both listed in the [archive index](../archive.md).

## 1. Set up the project

```text
markitect init --repo PATH [--name TITLE]
```

`init` writes what is missing: the manifest with `std`, `execution.yaml` with no checks, the project anchor `model/project.md` if the model folder is empty, Root's registry page, the ignore page and the lock. It installs or updates the modeling and Extract skills in `.claude/skills/` and `.agents/skills/` and lists them on the ignore page. It never changes an existing model file, so it is safe to run again after an upgrade. Commit its result.

## 2. Write the model

Write the model under `.markitect/model/`: one Markdown file with front matter per Definition, one folder per Package. The [model language guide](model-language.md#definitions) gives the form. A project writes its own Kinds for what the standard Libraries do not cover; to use `rest` or `cli`, add it to the manifest and run `markitect lock`.

The modeling skill helps an agent write the model with you and asks first when something is open. For an existing code base, the Extract skill draws a first model from the code. You accept every model change yourself.

## 3. Register the files

Every repository file outside `.markitect/` belongs to one Package and must be listed on its registry page or on the ignore page. Give each Package a registry page with the path patterns it owns, and list each file with the Definitions it realizes and why ([realization record](model-language.md#the-realization-record)). Shared files such as the README are owned by the lowest common parent, often Root. The required checks go in `.markitect/execution.yaml` ([required checks](model-language.md#required-checks)).

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

`impact` compares the base commit with the working tree: the changed and reached Definitions with distance and reason, and the reached files with their owners. `--findings` adds Findings from a Findings page. It prints the checklist ([impact and checklists](model-language.md#impact-and-checklists)).

## 7. Output and exit codes

Every command except `init` and `lock` takes `--json`. Exit code 0 means success, 1 that the command ran and found errors, 2 that it could not run.

## Troubleshooting

| Observation | Response |
| --- | --- |
| `file.unregistered` | List the file on its owner's registry page, or on the ignore page with a reason. |
| `lock.mismatch` after an upgrade of `markitect` | A standard Library changed. Run `markitect lock`, review the lock diff and commit it. |
| `package.cycle` | Move the shared concept into a common parent, or invert the dependency ([links, scope and visibility](model-language.md#links-scope-and-visibility)). |
| `lint.no_paths` | A Package owns no file. Name its first path patterns on its registry page ([first patterns](model-language.md#registry-pages)). |
| `lint.stale` | A registry page still names a removed Definition, file or Package. Remove the entry or page. |
| `impact` refuses to run | The project has compile errors; run `check` and fix them first. |
