# Markitect

Markitect is model-first development with delegated realization. You maintain one canonical YAML model of your project's intended world under `.markitect/`: its concepts, rules, boundaries, responsibilities, checks and dependencies. Source code, tests, configuration and prose remain ordinary files in the adopting project; they realize the model.

You change the model, usually in conversation with your coding agent, and Markitect has the change applied. It compiles the model, derives the affected Managers and files, and lets recursive Managers implement the change with independent review, integration, fixed-candidate verification and guarded Apply. Markitect reports evidence for the checks performed; project owners retain decisions about semantic acceptance and release. [Markitect in brief](docs/vision.md#markitect-in-brief) explains why and how to judge it.

## Quick navigation

| Goal | Start here |
|---|---|
| Build the model-first workflow from this source checkout | [Run the current source](#run-the-current-source) |
| Create a new adopting project | [Start a model-first project](#start-a-model-first-project) |
| Model an existing repository | [Brownfield workflow](docs/project-workflow.md#existing-repositories) |
| Understand the workflow and tool boundaries | [Project workflow](docs/project-workflow.md), [Project operations](docs/project-operations.md), and [Architecture](docs/architecture.md) |
| Install the published compatibility CLI | [Install Markitect](#install-markitect) and [distribution guidance](integration/README.md) |

## Run the current source

The model-first `markitect project` workflow is available in this development source checkout; it is not included in the published v0.14.1 binary. Building from source does not publish a release or change a project's existing version pin.

For Windows, use Git and Go 1.27.1 or later. From this Markitect source checkout, build a separate development executable:

```powershell
$toolBin = Join-Path $env:LOCALAPPDATA 'Markitect\development'
New-Item -ItemType Directory -Force -Path $toolBin | Out-Null
go build -o (Join-Path $toolBin 'markitect.exe') ./src/cmd/markitect
if ($LASTEXITCODE -ne 0) { throw 'Markitect build failed.' }
$env:PATH = "$toolBin;$env:PATH"
markitect project --help
```

Keep the source checkout for development. Configure the adopting project's runtime with `project setup`, then connect the outer coding client to its repository-local Markitect MCP server. Manager and reviewer execution uses the supported Codex App Server runtime. See [Provider adapters](docs/provider-adapters.md) for connection and execution boundaries.

## Start a model-first project

Initialize your target Git repository on a feature branch, then create its scaffold:

```powershell
# Run in the target project, with the development executable on PATH.
git init -b codex/project-start
markitect project init --repo . --name my-project
markitect project init --repo . --name my-project --write
markitect project onboard --repo . --provider codex
```

Review the onboarding preview and apply it with the returned digest:

```powershell
markitect project onboard --repo . --provider codex --expect ONBOARDING_PLAN_DIGEST --write
```

Choose `claude` or `both` for those contributors. Onboarding previews repository-local entrypoints and discoverable skills, preserves custom content, and points contributors to the shared workflow. It does not register MCP or change global account/provider settings. Connect the selected client using [Provider adapters](docs/provider-adapters.md).

Then give your agent an ordinary request, for example:

> Build a small shop where creating an order reserves stock and cancelling it restores the reserved stock exactly once. Include tests and explain the project through its generated documentation.

Use the [Project workflow](docs/project-workflow.md) to inspect the model, record scope and acceptance, update canonical intent when needed, establish readiness, and continue the same operation through execution, verification, and guarded Apply. The [Project operations reference](docs/project-operations.md) documents exact interfaces and recovery. The [Shop example](examples/project-world/README.md) provides a runnable model and finite tests.

## Example adopting-project layout

The tree below illustrates a repository that uses Markitect. It describes a target adopting project, not this Markitect source repository. In this source checkout, implementation is under `src/`, product documentation is under `docs/`, and adopting-project examples are under `examples/`.

```text
.markitect/
  project.yaml                    project selection and policy
  model/
    manager.yaml                  root responsibility
    sales/
      manager.yaml                Sales responsibility
      orders/                     concepts, rules and use cases together
  runtime.yaml                    pinned local execution and budgets
  workflows/model-first.md        installed contributor workflow
  state/                          exploration and model-change history
  runs/                           candidates, receipts and verification
.agents/skills/                   native Codex skills, when selected
.claude/skills/                   native Claude skills, when selected
AGENTS.md / CLAUDE.md             selected native entrypoints
src/                             application artifacts in the adopting project
tests/                           tests in the adopting project
docs/markitect/project.md         generated readable model view
```

Application paths are choices made by each project, not imposed language conventions. The model tree follows conceptual slices. Each file has one accountable owner and explicit relationships to the concepts or rules it realizes; a shared file can realize several concepts. Managers receive bounded context and public neighbor contracts. Parents receive child results for integration, not private transcripts.

## Change, verify, and recover

Start an intent change in the model. Review its readable view and commit the accepted canonical model under the repository's policy before implementing it. Drafts remain proposals. Markitect records accepted first-parent model transitions and supplies scoped change briefings.

Implementation follows impact through responsible Managers and their ancestors. Leaf implementers and independent reviewers iterate within finite limits; parent Managers integrate child results and can request bounded rework. Final verification evaluates permanent Manager responsibilities and declared checks against one immutable candidate. Coverage accounts for ordinary files, explicit ignores, and known tool files so an unknown file cannot silently disappear.

The MCP-led workflow advances a ready named scope through planning, Manager work and integration, verification, preflight, and guarded Apply. A successful Apply records technical completion, not semantic acceptance. Interrupted work resumes with the same durable IDs; unknown outcomes are inspected rather than replayed. See [Project operations](docs/project-operations.md) for recovery and exact interfaces.

For an existing repository, begin with fixed-source discovery and iterative reverse modeling. Separate observed code behavior from documented intent and future decisions. Managers make bounded proposals; parent Managers integrate them, and explicit owner resolutions precede model-only adoption. Commit the accepted model before a separate cleanup or implementation request. Unresolved or transitional areas remain visible. See the [Brownfield workflow](docs/project-workflow.md#existing-repositories).

Native instructions guide cooperation but do not prevent a process with ordinary filesystem permissions from editing files directly. Guarded Apply, checks, and repository policy enforce their stated boundaries. Commits, digests, and decision references bind evidence but do not authenticate human approval.

## Current source status and compatibility

The combined A01 native smoke passed for source `cdd30b0efc540f151404dabe86b022275dc40d83`. That result is tied to its recorded source and does not establish human semantic acceptance or productivity benefit. The [roadmap](docs/implementation-plan.md) and its [backlog](docs/work-items/backlog.yaml) own current direction and status; the [A01 validation record](docs/validation/a01-native-smoke-20261010.md) records the run and its limits.

Earlier Project/Domain and canonical Projection contracts, released examples, and historical research are preserved. Existing projects need an explicit migration; the model-first workflow does not silently reinterpret old files. Use the [compatibility reference](docs/usage.md#legacy-projectdomain-cli-compatibility) and [distribution guide](integration/README.md) to maintain a released installation.
<details>
<summary>Install and try the published v0.14.1 compatibility CLI</summary>

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

The [minimal example](examples/minimal/README.md) is a synthetic, executable fixture. To use Markitect in your own repository, follow the [release installation and verification guide](integration/README.md) to preview and install a project pin.

<!-- markitect-release:try:end -->
</details>

## Documentation and development

Use the [documentation map](docs/README.md) to find product and development guidance. The [architecture](docs/architecture.md) describes product boundaries; [contribution checks](CONTRIBUTING.md) describes source validation. Adopting projects own their business rules, technologies, and acceptance policy.

## License

Markitect is licensed under [Apache-2.0](LICENSE). The Glacius Labs and Markitect names and logos are not licensed as trademarks by that license. Third-party notices are maintained with the [distribution guidance](integration/README.md).
