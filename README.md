<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/markitect-wordmark-dark.svg">
    <img src="assets/markitect-wordmark-light.svg" alt="Markitect" width="360">
  </picture>
</h1>

AI-facing engineering guidance often lives in documents whose relationships and owners are hard to see. Markitect makes that guidance explicit: author rules, workflows, skills, agents, reusable text, and contracts as typed resources, then declare how they depend on one another and which ordinary files they need.

The Go CLI checks resource structure and generated views, compiles a selected resource's declared context, and reports affected resources between fixed Git revisions. These operations are deterministic and do not require a model API. Markitect does not decide whether prose is true, complete, or followed.

## Get started with the v0.4.1 release

With Git and an authenticated [GitHub CLI](https://cli.github.com/) session, verify the exact [v0.4.1 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.4.1), download the binary for your platform, and verify that file before running it. The supported native targets are Windows amd64 and Linux amd64.

Clone the tagged synthetic example, then run `version` and `check`. These commands read the checked-in example; they do not initialize or modify an adopting repository.

Windows PowerShell:

```powershell
New-Item -ItemType Directory -Path markitect-v0.4.1-demo -ErrorAction Stop | Out-Null
Set-Location markitect-v0.4.1-demo
$repository = 'github.com/Glacius-Labs/Markitect'
gh release verify v0.4.1 --repo $repository
if ($LASTEXITCODE -ne 0) { throw 'Release verification failed.' }
gh release download v0.4.1 --repo $repository --pattern 'markitect-v0.4.1-windows-amd64.exe' --dir .
if ($LASTEXITCODE -ne 0) { throw 'Binary download failed.' }
gh release verify-asset v0.4.1 .\markitect-v0.4.1-windows-amd64.exe --repo $repository
if ($LASTEXITCODE -ne 0) { throw 'Binary verification failed.' }
git clone --depth 1 --branch v0.4.1 https://github.com/Glacius-Labs/Markitect.git markitect-sample-v0.4.1
if ($LASTEXITCODE -ne 0) { throw 'Could not get the v0.4.1 synthetic example.' }
& .\markitect-v0.4.1-windows-amd64.exe version
if ($LASTEXITCODE -ne 0) { throw 'Version check failed.' }
& .\markitect-v0.4.1-windows-amd64.exe check --repo .\markitect-sample-v0.4.1\examples\minimal
if ($LASTEXITCODE -ne 0) { throw 'Example check failed.' }
```

Linux amd64:

```sh
set -e
mkdir markitect-v0.4.1-demo
cd markitect-v0.4.1-demo
repository=github.com/Glacius-Labs/Markitect
gh release verify v0.4.1 --repo "$repository"
gh release download v0.4.1 --repo "$repository" --pattern 'markitect-v0.4.1-linux-amd64' --dir .
gh release verify-asset v0.4.1 ./markitect-v0.4.1-linux-amd64 --repo "$repository"
git clone --depth 1 --branch v0.4.1 https://github.com/Glacius-Labs/Markitect.git markitect-sample-v0.4.1
chmod +x ./markitect-v0.4.1-linux-amd64
./markitect-v0.4.1-linux-amd64 version
./markitect-v0.4.1-linux-amd64 check --repo ./markitect-sample-v0.4.1/examples/minimal
```

To use Markitect in an adopting repository, follow the [release installation and verification guide](integration/README.md). It verifies the matching versioned release assets and previews the exact pin files before installation; downloading the executable above alone does not install a project pin.

## Try it from a source checkout

From a Markitect source checkout, with Go 1.27.1 or later:

```sh
go run ./cmd/markitect check --repo examples/minimal
go run ./cmd/markitect context --repo examples/minimal --namespace sample --kind Skill --name rollback-review
```

The first command checks the example's declared resources and managed views. The second prints the selected Skill's dependency context. The [minimal example](examples/minimal/README.md) is a synthetic, executable fixture.

To use Markitect in your repository, choose a [published GitHub release](https://github.com/Glacius-Labs/Markitect/releases) and follow the [installation and verification guide](integration/README.md). The CLI can preview a minimal project setup with `markitect init`; the preview shows the exact files before writing.

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
