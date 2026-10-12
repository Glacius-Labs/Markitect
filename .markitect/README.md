# Markitect project configuration

[project.yaml](project.yaml) selects this repository's current model under `model/`. `markitect check --repo .` checks the full repository against its explicit Artifact paths.

The tooling Manager owns the [module pin contract](model/tooling/module-pin-contract.yaml) and [exact verification inputs](model/tooling/module-verification.yaml).

- [Git hooks configuration](modules/githooks.config) pins the tracked pre-commit bytes; the check does not install or execute the hook.
- [Pipeline configuration](modules/pipelines.config) pins workflow bytes and selected YAML scalars to literal current Check command arguments; the check does not execute CI.
- [Generated project view](../docs/markitect/project.md) is checked separately for byte freshness.
