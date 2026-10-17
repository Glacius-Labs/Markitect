# Markitect

Markitect is model-first development with delegated realization. People maintain a canonical model of their project's intended world. Code, tests, configuration and documents realize that intent. Recursive Managers are intended to realize model changes with independent review, integration, verification and guarded Apply. [Markitect in brief](docs/vision.md#markitect-in-brief) explains the product purpose and evidence boundary.

## Run the current source

Current source implements canonical language v1 compilation, complete file accounting, execution declarations, static context selection and optional source-bound profile metadata/settings preparation. It also generates a reviewed readable model view. Apply explicitly returns unavailable before acquisition or effects. The [delivery plan](docs/implementation-plan.md) and delivery PRs distinguish integrated execution components from the complete Manager workflow.

Use Git and the Go version in `go.mod`, then run from this checkout:

```text
go run ./cmd/markitect help
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context product/model --repo .
go run ./cmd/markitect docs --repo .
```

Build into an external directory with `go build -o ABSOLUTE_EXTERNAL_PATH/markitect ./cmd/markitect` (use a native path and `markitect.exe` on Windows). Building does not publish a release or update a project pin.

## Start a model-first project

Use an existing Git commit, author a [canonical v1 manifest and model](docs/design/canonical-model/minimal-language/language-contract.md), and register exact artifact ownership. The default manifest is `.markitect/model.yaml`; `--model PATH` and repeated `--dependency PATH` select explicit sources. No automatic scaffold or dependency discovery runs. The [self-model](.markitect/README.md) is one complete current example.

`check` requests structural preparation. `context <package>` reports selected Definitions and inclusion reasons. `--revision REF` resolves once to a fixed commit. `--metadata --host-profile ABSOLUTE_DOCUMENT_PATH` additionally requests static metadata/settings preparation; it starts no process and does not establish runtime support. [Project workflow](docs/project-workflow.md) explains the stages and exits.

## Change, verify, and recover

Maintain intent in canonical Definitions and exact file ownership in the registry. After a model edit, review `docs` and pass its returned `view.previewDigest` to `--write --expect`. Content, canonical model and write-plan digests are distinct. Source or target changes refuse a stale write; protected inputs and fixed revisions cannot be overwritten. Actual completed paths remain visible after partial failure.

[Architecture](docs/architecture.md), [contributor gates](CONTRIBUTING.md) and the [code map](docs/development/code-map.md) describe the implementation. The packaged source smoke exercises actual Library M1/M2 read-only preparation and refusals; the separate [Library oracle](experiments/library-playground/README.md) remains independent. These are source checks, not delegated realization or acceptance proof.

## Distribution status

The previous Project/Domain releases and their binary downloads were retired by the owner on 17 October 2026. There is currently no supported binary release of the new model-first product. Build the current source as described above; an old version tag is a historical source reference.

<!-- markitect-release:install:start -->

## Install Markitect

Build from current reviewed source. A future published release needs its own source-bound verification before installation instructions are added here.

<!-- markitect-release:install:end -->

<!-- markitect-release:try:start -->

## Examples

Use the [canonical Library model](docs/design/canonical-model/library-example/README.md), current [source workflow](docs/project-workflow.md) and [independent Playground](experiments/library-playground/README.md). A complete Apply remains the delivery target.

<!-- markitect-release:try:end -->

Historical Git tags remain unchanged. The [retirement record](docs/history/distribution-retirement-20261017.md) preserves release identities, asset digests and attestations. The [distribution guide](integration/README.md) describes local packaging and future publication.

## Documentation and development

Start with [the documentation map](docs/README.md), [the workflow](docs/project-workflow.md), [operations](docs/operations.md) and [CONTRIBUTING](CONTRIBUTING.md). Accepted product decisions remain in the [concept register](docs/concepts/register.md).

## License

Markitect is licensed under [Apache-2.0](LICENSE). Third-party notices are available through `markitect licenses` and [the canonical notice file](internal/tooling/licenses/notices.md).
