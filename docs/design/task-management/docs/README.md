# Documentation

These documents describe the complete intended first version of Tasks. The project is at design stage; implementation and operational validation are still pending. Start with the [project README](../README.md) for setup and a short example.

## Product and use

- [Product](product.md): audience, problem, scope, capabilities, quality goals and limits.
- [Domain](domain.md): task identity, text, state and lifecycle.
- [Command reference](cli.md): syntax, flows, output, errors and exit status.

## Engineering and care

- [Architecture](architecture.md): context, responsibilities, dependencies and runtime flows.
- [Storage](storage.md): JSON format, validation, commit point and compatibility.
- [Operations](operations.md): local setup, file selection, backup, restore, failure recovery and source release/update.
- [Testing](testing.md): automated scenarios, human review, platform evidence and release criteria.
- [Decisions](decisions.md): rationale, alternatives, consequences and deferred capabilities.
- [Contributing](../CONTRIBUTING.md): source layout, commands, changes and review.
- [Agent entrypoint](../AGENTS.md): a short route to the same project guidance.
- [Changelog](../CHANGELOG.md): intended release scope and eventual release notes.

## Maintenance

Product behavior belongs in the domain and command reference; file behavior in Storage; user procedures in Operations; development procedures in Contributing. Architecture explains implementation responsibilities and Decisions explains their rationale. Update affected pages and examples together when behavior changes. Cross-links support reading; no specialized tooling is needed to understand this documentation.

The documentation is English and public command messages are English. There is no localization commitment for the first version. Technical documentation choices do not establish completed implementation or release readiness; actual results must name their source version, platform and outcome.
