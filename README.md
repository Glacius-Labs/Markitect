<h1><img src="assets/markitect-avatar-512.png" alt="" width="48" height="48"> Markitect</h1>

AI-facing engineering guidance often lives in documents whose relationships and owners are hard to see. Markitect makes that guidance explicit: author rules, workflows, skills, agents, reusable text, and contracts as typed resources, then declare how they depend on one another and which ordinary files they need.

The Go CLI checks resource structure and generated views, compiles a selected resource's declared context, and reports affected resources between fixed Git revisions. These operations are deterministic and do not require a model API. Markitect does not decide whether prose is true, complete, or followed.

## Try it in a minute

From a Markitect source checkout, with Go 1.27.1 or later:

```sh
go run ./cmd/markitect check --repo examples/minimal
go run ./cmd/markitect context --repo examples/minimal --namespace sample --kind Skill --name rollback-review
```

The first command checks the example's declared resources and managed views. The second prints the selected Skill's dependency context. The [minimal example](examples/minimal/README.md) is a synthetic, executable fixture.

To use Markitect in your repository, download the [verified v0.4.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.4.0) and follow the [installation and verification guide](integration/README.md). The CLI can preview a minimal project setup with `markitect init`; the preview shows the exact files before writing.

## What it does

- Defines typed resources and explicit dependencies in YAML; keeps readable prose in Markdown.
- Checks structure and managed output drift without calling a model.
- Compiles context from declared resource and file inputs.
- Compares immutable Git revisions to identify changed paths and affected resources.
- Runs only the verification commands a project owner declares; missing checks remain incomplete evidence.

Authoring guidance ships with Markitect and uses the same resource model as project content. Review evidence is advisory: the CLI can assess whether its declared inputs still match, but cannot authenticate a reviewer or transfer human acceptance.

## Documentation

- [Usage and upgrade notes](docs/usage.md): project format, CLI behavior, and schema transitions.
- [Architecture](docs/architecture.md): product model, evidence, verification, and boundaries.
- [Roadmap](docs/implementation-plan.md): current source scope and planned work.
- [Operations](docs/operations.md): source development, checks, and release handling.
- [Integration](integration/README.md): verified downloads, installation, upgrades, and release publication.
- [Content packages](docs/content-packages.md): exact direct pins for offline content archives.
- [Documentation inputs](docs/documentation.md): declare exact ordinary files used by resources.
- [Development](CONTRIBUTING.md): code ownership and contribution checks.
- [Third-party notices](internal/licenses/notices.md): bundled upstream attribution, also available offline with `markitect licenses`.

## License

Markitect is licensed under the [Apache License 2.0](LICENSE). The Glacius Labs and Markitect names and logos are not licensed as trademarks by that license. Third-party notices are listed separately above.
