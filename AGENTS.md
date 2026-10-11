# Markitect agent entrypoint

Markitect is model-first development with delegated realization. People maintain a canonical model of the project's intended world. Markitect compiles it, derives the affected responsibilities and files, and has each change of intent realized by recursive Managers with independent review, integration, verification and guarded Apply. Before you assess the product direction or propose changing it, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence.

## Every task

1. Work on a branch from current `origin/main`. Its name starts with a prefix that says what it holds, such as `dev/`, `fix/` or `docs/` ([branch names](docs/implementation-plan.md#branch-names)).
2. Check this repository's [project model](.markitect/project.yaml) and read the context of the Manager that owns your change:

   ```text
   go run ./src/cmd/markitect check --repo .
   go run ./src/cmd/markitect context --repo . '["project.markitect.example.org/v1alpha1","Manager","product","product"]'
   ```

   The root Manager `markitect` delegates to `product`, `legacy`, `tooling`, `documentation` and `examples`; copy a Manager's ID from its Identity line in the [readable view](docs/markitect/project.md). In Windows PowerShell 5.1, write each `"` inside the ID as `\"`. The context holds the Manager's paths, instructions, Statements with the repository rules, Artifacts and checks. `check` fails when a file lies outside every Artifact path. [The workflow guide](docs/project-workflow.md) explains how to work with the model. Implementation-only work needs no model change.
3. If you change a file under `.markitect/model/`, preview the readable view with `go run ./src/cmd/markitect docs --repo .` and write it with `go run ./src/cmd/markitect docs --repo . --expect DIGEST --write`, using the preview's digest. Do not edit generated files.
4. Before you open a pull request, run the checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change).

The earlier Project/Domain line still runs its own checks in CI until ARCH-09 removes it.

## Where things are

- [Roadmap](docs/implementation-plan.md): direction, streams, zones and the rules for parallel work.
- [Backlog](docs/work-items/backlog.yaml): the status of each work package.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): every other document.
