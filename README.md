# Markitect

**Define the project. Keep its implementation aligned.**

Markitect is model-first development with delegated realization. You describe what your project should be: its concepts, behavior, architecture and rules. Code, tests, configuration and documentation realize that model.

Markitect is being built to organize AI agents around those responsibilities, with independent review of their work and its integration. The goal is to spend more attention on designing the system and making decisions while keeping its implementation consistent through change.

**Development stage:** the current source supports model compilation, file accounting, context preparation and a guarded readable model view. Recursive realization and Apply are still being built. The published v0.14.1 CLI belongs to the earlier product generation; its installation instructions appear below.

## Why Markitect

As a project grows, a change can satisfy the immediate request while missing an existing rule, a related test or a documentation update. Markitect makes the intended project explicit and gives its realization accountable owners.

The target workflow is:

1. **Model the intent.** Define the project's behavior, rules and boundaries in one canonical model.
2. **Organize the work.** A Manager owns each model Package. The Host supplies explicit context and propagates the same instruction unchanged to every local Manager.
3. **Realize and review.** Every Manager assesses its area, makes needed changes and integrates its children's results. Fresh independent reviewers assess local work, including no-change results, and integration.
4. **Verify and Apply.** Required checks and current reviews support guarded publication of the candidate. People retain decisions about intent and acceptance.

For example, changing a library's borrowing limit from five books to three should bring its code, tests and usage guide into agreement. The [two-Apply library demonstration](docs/design/canonical-model/manager-apply-playground.md) is the next complete workflow the project is working toward, rather than an available demo.

Markitect aims to reduce missed obligations and repeated synchronization. Those benefits still need validation. Start with [Markitect in brief](docs/vision.md#markitect-in-brief) for the full product picture, or [origin and rationale](docs/concepts/origin-and-rationale.md) for the experience behind it.

## Explore the project

| Your goal | Start here |
| --- | --- |
| Understand the product and its intended use | [Vision](docs/vision.md#markitect-in-brief) |
| Run what current source supports | [Commands below](#run-the-current-source) and [project workflow](docs/project-workflow.md) |
| Understand the new Manager and Apply concept | [Apply design map](docs/design/canonical-model/README.md#apply-design) |
| Contribute to the rebuild | [Delivery plan](docs/implementation-plan.md) and [CONTRIBUTING](CONTRIBUTING.md) |
| Find guides, evidence or historical records | [Documentation map](docs/README.md) |

## Run the current source

Current source implements canonical language v1 compilation, complete file accounting, execution declarations, static context selection and optional source-bound profile metadata/settings preparation. It also generates a reviewed readable model view. Apply explicitly returns unavailable before acquisition or effects. Sessions, workspaces, executable checks and artifact publication are later stages in the [delivery plan](docs/implementation-plan.md).

Use Git and the Go version in `go.mod`, then run from this checkout:

```text
go run ./cmd/markitect help
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context product/model --repo .
go run ./cmd/markitect docs --repo .
```

Build into an external directory with `go build -o ABSOLUTE_EXTERNAL_PATH/markitect ./cmd/markitect` (use a native path and `markitect.exe` on Windows). This source surface is separate from the published v0.14.1 binary below. Building does not publish or update a project pin.

## Start a model-first project

Use an existing Git commit, author a [canonical v1 manifest and model](docs/design/canonical-model/minimal-language/language-contract.md), and register exact artifact ownership. The default manifest is `.markitect/model.yaml`; `--model PATH` and repeated `--dependency PATH` select explicit sources. No automatic scaffold or dependency discovery runs. The [self-model](.markitect/README.md) is one complete current example.

`check` requests structural preparation. `context <package>` reports selected Definitions and inclusion reasons. `--revision REF` resolves once to a fixed commit. `--metadata --host-profile ABSOLUTE_DOCUMENT_PATH` additionally requests static metadata/settings preparation; it starts no process and does not establish runtime support. [Project workflow](docs/project-workflow.md) explains the stages and exits.

## Change, verify, and recover

Maintain intent in canonical Definitions and exact file ownership in the registry. After a model edit, review `docs` and pass its returned `view.previewDigest` to `--write --expect`. Content, canonical model and write-plan digests are distinct. Source or target changes refuse a stale write; protected inputs and fixed revisions cannot be overwritten. Actual completed paths remain visible after partial failure.

[Architecture](docs/architecture.md), [contributor gates](CONTRIBUTING.md) and the [code map](docs/development/code-map.md) describe the implementation. The packaged source smoke exercises actual Library M1/M2 read-only preparation and refusals; the separate [Library oracle](experiments/library-playground/README.md) remains independent. These are source checks, not delegated realization or acceptance proof.

## Current source status and compatibility

Current source has one root Go module and no legacy product engine. Historical source and published releases retain their identities and evidence. The installation and example blocks below intentionally remain pinned to the existing immutable v0.14.1 release; they do not describe this source increment.

<!-- markitect-release:install:start -->

## Install Markitect

The current [v0.14.1 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.14.1) provides Windows and Linux amd64 binaries. The commands below download a fixed version from the public release, check its published SHA-256 digest, and install it for your user. No Go installation or GitHub login is needed. Git is needed for Markitect commands that read Git revisions.

**Windows (PowerShell):**

```powershell
if (-not [Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([Runtime.InteropServices.OSPlatform]::Windows) -or [Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne [Runtime.InteropServices.Architecture]::X64) { throw 'Only Windows amd64 is released.' }
$tag = 'v0.14.1'
$asset = "markitect-$tag-windows-amd64.exe"
$sha256 = '2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4'
$download = Join-Path $env:TEMP ("markitect-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $download -ErrorAction Stop | Out-Null
$source = Join-Path $download $asset
Invoke-WebRequest "https://github.com/Glacius-Labs/Markitect/releases/download/$tag/$asset" -OutFile $source -UseBasicParsing -ErrorAction Stop
if ((Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash -ne $sha256) { throw 'Release binary SHA-256 mismatch.' }
$bin = Join-Path $env:LOCALAPPDATA 'Programs\Markitect'
New-Item -ItemType Directory -Path $bin -Force -ErrorAction Stop | Out-Null
Copy-Item -LiteralPath $source -Destination (Join-Path $bin 'markitect.exe') -Force -ErrorAction Stop
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($bin -notin ($userPath -split ';')) {
    [Environment]::SetEnvironmentVariable('Path', ((@($userPath, $bin) | Where-Object { $_ }) -join ';'), 'User')
}
$env:Path = "$bin;$env:Path"
$installed = & (Join-Path $bin 'markitect.exe') version
if ($LASTEXITCODE -ne 0 -or $installed -ne 'Markitect 0.14.1 (windows/amd64)') { throw 'Installed CLI version check failed.' }
$installed
Remove-Item -LiteralPath $source, $download
```

**Linux (bash):**

```bash
(
  set -e
  [ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] || { echo 'Only Linux amd64 is released.' >&2; exit 1; }
  tag=v0.14.1
  asset="markitect-$tag-linux-amd64"
  sha256=f515f8fb37335ff5bd36f77cbbd0af227c3558d0d5aa8ba80503bae40314fbdc
  download="$(mktemp -d)"
  trap 'rm -rf "$download"' EXIT
  curl -fLsS "https://github.com/Glacius-Labs/Markitect/releases/download/$tag/$asset" -o "$download/$asset"
  printf '%s  %s\n' "$sha256" "$download/$asset" | sha256sum -c -
  install -D -m 0755 "$download/$asset" "$HOME/.local/bin/markitect"
  "$HOME/.local/bin/markitect" version
)
export PATH="$HOME/.local/bin:$PATH"
```

Add `~/.local/bin` to your shell startup file if it is not already on `PATH`. For developers who already use Go 1.27.1 or later, `go install github.com/Glacius-Labs/Markitect/cmd/markitect@v0.14.1` is a shorter source-build option. macOS and arm64 binaries are not currently released. For signed release and asset attestation verification, project pinning, and upgrades, follow the [distribution guide](integration/README.md). The CLI's `install` command installs a **project pin**, not the CLI on your computer.

<!-- markitect-release:install:end -->

<!-- markitect-release:try:start -->

## Try the published v0.14.1 compatibility CLI

Clone the v0.14.1 synthetic example and run a structural check with the installed binary. This reads the example without modifying an adopting repository.

Windows PowerShell:

~~~powershell
git clone --depth 1 --branch v0.14.1 https://github.com/Glacius-Labs/Markitect.git markitect-sample-v0.14.1
if ($LASTEXITCODE -ne 0) { throw 'Could not get the v0.14.1 synthetic example.' }
markitect version
if ($LASTEXITCODE -ne 0) { throw 'Version check failed.' }
markitect check --repo .\markitect-sample-v0.14.1\examples\minimal
if ($LASTEXITCODE -ne 0) { throw 'Example check failed.' }
~~~

Linux amd64:

~~~sh
set -e
git clone --depth 1 --branch v0.14.1 https://github.com/Glacius-Labs/Markitect.git markitect-sample-v0.14.1
markitect version
markitect check --repo ./markitect-sample-v0.14.1/examples/minimal
~~~

The [minimal example from v0.14.1](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1/examples/minimal) is a synthetic, executable fixture. To use Markitect in your own repository, follow the [release installation and verification guide](integration/README.md) to preview and install a project pin.

<!-- markitect-release:try:end -->

## Documentation and development

Start with [the documentation map](docs/README.md), [the workflow](docs/project-workflow.md), [operations](docs/operations.md) and [CONTRIBUTING](CONTRIBUTING.md). Accepted product decisions remain in the [concept register](docs/concepts/register.md).

## License

Markitect is licensed under [Apache-2.0](LICENSE). Third-party notices are available through `markitect licenses` and [the canonical notice file](internal/tooling/licenses/notices.md).
