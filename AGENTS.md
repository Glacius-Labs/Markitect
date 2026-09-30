# Markitect agent entrypoint

Start with [the documentation map](docs/README.md). [Architecture](docs/architecture.md) describes the product model and boundaries; [usage](docs/usage.md) describes the CLI and project format; [the roadmap](docs/implementation-plan.md) owns future work; [development](CONTRIBUTING.md) owns contribution checks.

Markitect is an independent authoring and structure-validation tool for AI-facing engineering resources. Keep its core independent of adopting repositories and model providers. Use YAML for machine-relevant ownership and dependencies, Markdown for readable prose, and explicit inputs for data that affects a result. Preserve deterministic diagnostics, fixed-snapshot evidence, and one owner per generated output.

Change canonical sources and regenerate generated views/schemas. Keep examples executable. Do not infer dependencies from prose links, weaken conservative invalidation, or treat AI evidence as human acceptance. Keep project-local policy in the project that owns it unless a deliberate product decision establishes a reusable general rule.

Work on a non-protected feature branch. Scope tests to concrete risks and required gates. Changes to an adopting project belong in its checkout and follow its instructions. A source change here does not update an installed release; validate the candidate and publish a versioned release through the documented process.
