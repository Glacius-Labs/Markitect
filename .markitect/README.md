# Markitect project configuration

[project.yaml](project.yaml) selects this repository's project model under `model/`, which `markitect check --repo .` checks. The files below belong to the legacy line until ARCH-09 removes it.

The development Area owns repository-specific checks and their explicit module inputs.

- [Module verification Rule](areas/development/module-verification.rule.yaml) declares the checked inputs and the limits of the hook and pipeline assertions.
- [Git hooks module configuration](modules/githooks.config) selects the tracked pre-commit file. It is not installed automatically.
- [Pipeline module configuration](modules/pipelines.config) binds selected workflow scalars to literal Project check argv renderings.
