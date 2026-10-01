<h1><img src="assets/markitect-avatar-512.png" alt="" width="48" height="48"> Markitect</h1>

AI-facing engineering guidance often lives in documents whose relationships and owners are hard to see. Markitect makes that guidance explicit: author rules, workflows, skills, agents, reusable text, and contracts as typed resources, then declare how they depend on one another and which ordinary files they need.

The Go CLI checks resource structure and generated views, compiles a selected resource's declared context, and reports affected resources between fixed Git revisions. These operations are deterministic and do not require a model API. Markitect does not decide whether prose is true, complete, or followed.

## Install

The current [v0.4.1 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.4.1) provides Windows and Linux amd64 binaries. The commands below download a fixed version from the public release, check its published SHA-256 digest, and install it for your user. No Go installation or GitHub login is needed. Git is needed for Markitect commands that read Git revisions.

**Windows (PowerShell):**

```powershell
if (-not [Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([Runtime.InteropServices.OSPlatform]::Windows) -or [Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne [Runtime.InteropServices.Architecture]::X64) { throw 'Only Windows amd64 is released.' }
$tag = 'v0.4.1'
$asset = "markitect-$tag-windows-amd64.exe"
$sha256 = '419f61e99a3ee09c4d87624b44755bacbcb796244cffe7a1fa148da4c79a35f1'
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
if ($LASTEXITCODE -ne 0 -or $installed -ne 'Markitect 0.4.1 (windows/amd64)') { throw 'Installed CLI version check failed.' }
$installed
Remove-Item -LiteralPath $source, $download
```

**Linux (bash):**

```bash
(
  set -e
  [ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] || { echo 'Only Linux amd64 is released.' >&2; exit 1; }
  tag=v0.4.1
  asset="markitect-$tag-linux-amd64"
  sha256=a02d0eb0ada42d95012da0965d471e6e31c4f5d607ac849fb8bd29d85b9b65a3
  download="$(mktemp -d)"
  trap 'rm -rf "$download"' EXIT
  curl -fLsS "https://github.com/Glacius-Labs/Markitect/releases/download/$tag/$asset" -o "$download/$asset"
  printf '%s  %s\n' "$sha256" "$download/$asset" | sha256sum -c -
  install -D -m 0755 "$download/$asset" "$HOME/.local/bin/markitect"
  "$HOME/.local/bin/markitect" version
)
export PATH="$HOME/.local/bin:$PATH"
```

Add `~/.local/bin` to your shell startup file if it is not already on `PATH`. For developers who already use Go 1.27.1 or later, `go install github.com/Glacius-Labs/Markitect/cmd/markitect@v0.4.1` is a shorter source-build option. macOS and arm64 binaries are not currently released. The [distribution guide](integration/README.md) covers GitHub's signed release attestations, project pinning, and upgrades. The CLI's `install` command installs a **project pin**, not the CLI on your computer.

## Try it in a minute

From a Markitect source checkout, with Go 1.27.1 or later:

```sh
go run ./cmd/markitect check --repo examples/minimal
go run ./cmd/markitect context --repo examples/minimal --namespace sample --kind Skill --name rollback-review
```

The first command checks the example's declared resources and managed views. The second prints the selected Skill's dependency context. The [minimal example](examples/minimal/README.md) is a synthetic, executable fixture.

To use Markitect in your repository, start with the installed CLI and preview a minimal project setup with `markitect init`; the preview shows the exact files before writing.

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
