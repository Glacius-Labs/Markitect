# Markitect agent entrypoint

Begin each engineering task with the canonical [Markitect-first Change workflow](internal/authoring/resources/workflow-markitect-first-change.yaml). The repository entry is `development/Skill/engineering-change` in [the root Project](markitect.yaml). Resolve a fixed BASE commit, then run `go run ./cmd/markitect check --repo . --revision BASE` and `go run ./cmd/markitect context --repo . --revision BASE --namespace development --kind Skill --name engineering-change`. [The workflow guide](docs/markitect-first.md) explains classification, desired intent first, artifact accounting and completion. Implementation-only work does not require a fake model change. Generated Codex/Claude entrypoints derive from these owners.

Start with [the documentation map](docs/README.md). [Vision](docs/vision.md) owns the product thesis and human/agent responsibilities; [Architecture](docs/architecture.md) describes the technical model and boundaries; [usage](docs/usage.md) describes the CLI and project format; [the roadmap](docs/implementation-plan.md) owns future work; [development](CONTRIBUTING.md) owns contribution checks.

Markitect makes project-selected engineering knowledge and policy explicit and versioned for bounded checks, context, impact and configured consumers. Keep its core independent of adopting repositories and model providers. Use YAML for machine-relevant ownership and dependencies, Markdown for readable prose, and explicit inputs for data that affects a result. Preserve deterministic diagnostics, fixed-snapshot evidence, and one owner per generated output.

Treat adopting-project code, schemas, configuration, infrastructure and documentation as declared artifact inputs. The core may track exact paths and bytes but must not infer their domain-specific structure or meaning. [Architecture](docs/architecture.md#project-artifact-boundary) owns this product boundary.

Change canonical sources and regenerate generated views/schemas. Keep examples executable. Do not infer dependencies from prose links, weaken conservative invalidation, or treat AI evidence as human acceptance. Keep project-local policy in the project that owns it unless a deliberate product decision establishes a reusable general rule.

Work on a non-protected feature branch. Scope tests to concrete risks and required gates. Changes to an adopting project belong in its checkout and follow its instructions. A source change here does not update an installed release; validate the candidate and publish a versioned release through the documented process.
