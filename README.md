# Markitect

Markitect makes AI-facing engineering knowledge maintainable as a small, typed system. It gives people and agents a canonical place to author rules, workflows, skills, agents, reusable text, and contracts; explicit dependencies connect those resources. A deterministic Go CLI validates structure, compiles selected context, explains relationships, measures change impact, checks declared outputs, and records whether review evidence still applies.

Authoring is part of the product. Its portable guidance is built from the same resource model as user-authored content. Markitect does not require a model to parse, validate, render, or compile context, and it does not claim to decide whether prose is true or whether an agent followed it.

Source and verified distributions are maintained by [Glacius Labs](https://github.com/Glacius-Labs/Markitect). See [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for available versions. See [the roadmap](docs/implementation-plan.md) and [architecture](docs/architecture.md) for the current product model.

## Start from source

Building this checkout requires Git and Go 1.27.1 or later.

```powershell
go test ./...
go build -o bin/markitect.exe ./cmd/markitect
./bin/markitect.exe version
./bin/markitect.exe authoring
./bin/markitect.exe check --repo examples/minimal
./bin/markitect.exe find --repo examples/minimal --query rollback --kind Rule
./bin/markitect.exe explain --repo examples/minimal --namespace sample --kind Rule --name rollback-review
./bin/markitect.exe context --repo examples/minimal --namespace sample --kind Skill --name rollback-review
```

On Linux and macOS, build as `go build -o bin/markitect ./cmd/markitect`. The [minimal example](examples/minimal/README.md) is synthetic, executable product documentation; it demonstrates resources, explicit relationships, generated views, and compiled context.

## Documentation

- [Usage and upgrade notes](docs/usage.md): project format, commands, and the v0.1-to-v0.2 transition.
- [Content packages](docs/content-packages.md): the direct offline package model in Markitect 0.3.0.
- [Documentation inputs](docs/documentation.md): connect exact ordinary file inputs to compiled context and impact.
- [Architecture](docs/architecture.md): product model, evidence, verification, rendering, and boundaries.
- [Roadmap](docs/implementation-plan.md): current source scope and planned work.
- [Operations](docs/operations.md): source development, checks, and release handling.
- [Integration](integration/README.md): verified downloads, installation, upgrades, and release publication.
- [Development](CONTRIBUTING.md): code ownership and validation.
- [Third-party notices](internal/licenses/notices.md): bundled upstream attribution, also available offline with `markitect licenses`.

The current source model builds on typed YAML resources, graph checks, core authoring, fixed-snapshot context and impact, explicit outputs and checks, and direct offline content packages. It adds bounded project initialization. Markitect does not provide an agent runtime, guarantee semantic completeness, or execute undeclared checks. See [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for available distributions and the [roadmap](docs/implementation-plan.md) for current source status.
