# Markitect agent entrypoint

Markitect is model-first development with delegated realization. People maintain a canonical model of the project's intended world. Markitect compiles it, derives the affected responsibilities and files, and has each change of intent realized by recursive Managers with independent review, integration, verification and guarded Apply. Before you assess the product direction or propose changing it, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence.

## Every task

1. Work on a branch from current `origin/main`. Its name starts with a prefix that says what it holds, such as `dev/`, `fix/` or `docs/` ([branch names](docs/implementation-plan.md#branch-names)).
2. Resolve a fixed BASE commit and load this repository's engineering context:

   ```text
   go run ./src/cmd/markitect check --repo . --revision BASE
   go run ./src/cmd/markitect context --repo . --revision BASE --namespace development --kind Skill --name engineering-change
   ```

   The context contains the [engineering-change Skill](.markitect/areas/development/engineering-change.skill.yaml), the [Markitect-first Change workflow](src/internal/host/embedded/resources/workflow-markitect-first-change.yaml), the [repository rules](.markitect/areas/development/repository-boundaries.rule.yaml) and the [document owners](.markitect/areas/development/document-owners.text.yaml). [The workflow guide](docs/markitect-first.md) explains how to classify and complete a change. Implementation-only work needs no model change.
3. If you change a file under `.markitect/`, regenerate its views with `go run ./src/cmd/markitect render --repo . --write`. Do not edit generated files.
4. Before you open a pull request, run the checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change).

These commands belong to the earlier Project/Domain line, which still runs this repository's own checks. ARCH-07 moves Markitect onto its own project model.

## Where things are

- [Roadmap](docs/implementation-plan.md): direction, streams, zones and the rules for parallel work.
- [Backlog](docs/work-items/backlog.yaml): the status of each work package.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): every other document.
