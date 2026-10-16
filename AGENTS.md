# Markitect agent entrypoint

Markitect is model-first development with delegated realization. People maintain a canonical model of the project's intended world. Markitect compiles it, derives the affected responsibilities and files, and has each change of intent realized by recursive Managers with independent review, integration, verification and guarded Apply. Before you assess the product direction or propose changing it, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence.

## Every task

1. Read the current [delivery plan](docs/implementation-plan.md) and your bounded assignment. Use an isolated worktree from its integrated base and a non-protected `codex/apply-<scope>` branch. Verify path, full SHA and local changes before writing. Preserve other sessions' work.
2. Check this repository's [project model](.markitect/project.yaml) and read the context of the Manager that owns your change:

   ```text
   go run ./src/cmd/markitect check --repo .
   go run ./src/cmd/markitect context --repo . product
   ```

   Select the Manager that owns your change. The context holds its paths, instructions, applicable repository rules, Artifacts, and owned or verifying checks. `check` fails when a file lies outside every Artifact path. [The workflow guide](docs/project-workflow.md) explains how to work with the model. Implementation-only work needs no model change.
3. If you change a file under `.markitect/model/`, preview the readable view with `go run ./src/cmd/markitect docs --repo .` and write it with `go run ./src/cmd/markitect docs --repo . --expect DIGEST --write`, using the preview's digest. Do not edit generated files. If `check` reports a path that no Artifact covers, for example a new package directory or top-level file, add it to the Artifact of the Manager that owns that area, or to `.markitect/ignore.yaml` with a reason, in the same pull request.
4. Before you open a pull request, run the checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change).

## Where things are

- [Delivery plan](docs/implementation-plan.md): current scopes, dependencies, path reservations and integration rules. Progress and checks live in the assigned chat/PR; the old backlog/dashboard are retired.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): every other document.
