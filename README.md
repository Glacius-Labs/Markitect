<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/markitect-wordmark-dark.svg">
    <img src="assets/markitect-wordmark-light.svg" alt="Markitect" width="360">
  </picture>
</h1>

**Define how your software should be built. Keep documentation, AI agents, and engineering tooling aligned as it evolves.**

Markitect is a compiler-like system for **canonical engineering knowledge**. It lets a project define rules, workflows, skills, agents, reusable text, contracts, ownership, and explicit relationships as typed resources instead of maintaining the same engineering truth independently across Markdown, Codex, Claude, and project checks.

The v0.10.0 release provides a generic canonical engineering model: Projects load versioned Domains, validate typed resources and relations, evaluate finite deterministic constraints, and expose one normalized semantic model to context, impact, projections, and configured adapters. Markitect does this without a model API. Exact declared project-artifact inputs remain opaque to the kernel; it does not infer their domain-specific meaning. See the [product boundary](docs/architecture.md#project-artifact-boundary).

> **AI should implement your architecture, not reinvent it on every task.**

## Why Markitect?

Engineering rules rarely live in one place.

A release process may be described in documentation, repeated in an agent Skill, partially enforced by CI, and summarized again in provider-specific instructions. Architecture guidance is copied into `AGENTS.md`, Claude rules, review checklists, and project conventions. Eventually one rule changes.

Then the real question becomes:

```text
What else has to change?
```

Without a canonical owner, the usual answer is another global consistency pass:

```text
docs
AGENTS.md
Claude configuration
Skills
Rules
Workflows
CI
project conventions
...
        ↓
read everything again
        ↓
find stale copies and contradictions
        ↓
try to synchronize them
```

Markitect moves the problem toward:

```text
canonical engineering knowledge
        ↓
explicit typed relationships
        ↓
deterministic validation
        ↓
bounded context and impact
```

Projects independently select the Markdown and provider projections they need.

The goal is not to generate more prompt text. The goal is to remove duplicated engineering truth and make more consistency questions answerable structurally.

## One canonical model, multiple views

A Markitect project keeps canonical typed resources separate from their explicitly selected projections. Markdown views under `docs/markitect/` are optional; provider outputs read canonical sources independently.

```text
                    Canonical resources
                  .markitect/areas/...
                           │
             ┌─────────────┼─────────────┐
             ▼             ▼             ▼
      Markdown views     Codex         Claude
      docs/markitect     outputs       outputs
```

Project-owned verification commands run separately against a materialized fixed project snapshot.

Generated output is never the canonical owner. Provider projections can be regenerated from canonical resources, and ordinary project artifacts remain in their normal project-owned locations.

## The current workflow

### Define

Author typed engineering resources with explicit ownership and relationships.

Current resource kinds include:

| Kind | Responsibility |
|---|---|
| `Text` | Reusable prose or context |
| `Rule` | Scoped engineering requirement |
| `Workflow` | Procedure plus dependencies |
| `Skill` | Agent entrypoint to a procedure |
| `Agent` | Responsibility and supported provider settings |
| `Contract` | Required kind and symbolic input/output signature |
| `Project` | Areas, imports, bindings, checks, outputs, and package pins |

### Validate

Markitect checks:

```text
resource structure
identities
references
area access
bindings
contracts
cycles
declared file inputs
documentation routers
managed output drift
```

These checks are deterministic.

### Project

Projects explicitly select outputs.

For example:

```yaml
spec:
  targets:
    - markdown
    - claude
```

`markdown` writes generated resource views under `docs/markitect/`. Codex and Claude outputs use their native paths and route directly to canonical resources.

### Understand change

Markitect works from resolved project snapshots. Fixed Git revisions are acquired through the Git adapter and converted into generic snapshot values before graph/context/impact logic runs.

`impact` compares project states and reports changed paths and affected resources. Review evidence can then be invalidated conservatively when its declared inputs no longer match. Unknown or unmodelled files conservatively broaden impact results.

## A small example

The repository-layout fixture contains canonical engineering knowledge under `.markitect/areas/`.

A Rule:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata:
  name: change-review
  namespace: engineering
spec:
  text: |
    Describe the intended behavior and required verification before accepting a change.
```

A Workflow can apply that Rule, reuse shared typed knowledge, and declare an exact ordinary project artifact:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Workflow
metadata:
  name: review-change
  namespace: engineering
spec:
  text: |
    Apply the change review requirement, shared principles, and project-owned procedure.
  files:
    - docs/engineering/change-procedure.md
  rules:
    - kind: Rule
      name: change-review
  uses:
    - kind: Text
      name: principles
      namespace: shared
```

The Project decides which projections exist:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata:
  name: repository-layout-example
spec:
  targets:
    - claude
  areas:
    - name: shared
      path: .markitect/areas/shared
    - name: engineering
      path: .markitect/areas/engineering
      imports:
        - shared
        - documentation
    - name: documentation
      path: docs
  documentation:
    roots:
      - docs
```

Then:

```sh
markitect check --repo .
markitect context --repo . --namespace engineering --kind Workflow --name review-change
markitect render --repo .
```

The canonical YAML remains the source of truth. Human documentation remains human-owned unless an explicit generated view is selected.

## Why this matters for AI-first engineering

AI dramatically increases implementation throughput. Architectural governance does not automatically scale with it.

If every coding agent must rediscover project structure, architecture rules, process requirements, and ownership from a large collection of prose on every task, two things happen:

```text
context grows
architecture drifts
```

Markitect's direction is the opposite:

```text
explicit engineering knowledge
        ↓
deterministic structure
        ↓
task-relevant context
        ↓
more autonomous agents inside known boundaries
```

The v0.10.0 release provides the typed Domain registry, normalized model, context and impact, artifact inputs, projections, deterministic constraints, configured adapters, packages, and reconciliation foundation for that model.

## Canonical repository layout

New projects default to responsibility-owned Areas under `.markitect/areas/<namespace>`.

```text
project/
├── markitect.yaml
├── .markitect/
│   ├── areas/
│   │   └── engineering/
│   ├── packages/
│   ├── tool/
│   └── bootstrap/
├── docs/
│   ├── ... human-owned documentation ...
│   └── markitect/          # generated only when target: markdown
├── .agents/
├── .codex/
├── .claude/
└── src/                    # ordinary project artifacts stay project-owned
```

The layout is a convention, not hidden semantics. Configured Areas are authoritative. File names do not infer resource types, and directory adjacency does not create dependencies.

See [Repository layout](docs/repository-layout.md) for the exact current contract.

## What Markitect does today

- Models AI-facing engineering knowledge as typed YAML resources.
- Gives each resource explicit identity, ownership, and graph relationships.
- Keeps ordinary project files as explicit opaque inputs instead of pretending to understand their domain semantics.
- Validates resource structure, references, bindings, contracts, cycles, file inputs, and configured documentation routers.
- Compiles a selected resource's bounded context.
- Supports committed fixed-run task context with exact selected project artifacts.
- Compares resolved snapshots to identify changed paths and affected resources.
- Produces only explicitly selected Markdown, Codex, and Claude outputs.
- Supports exact direct offline content packages with explicit exports and package-qualified identities.
- Runs only project-owned verification commands declared in `Project.spec.checks`.
- Records advisory review evidence and determines when its declared basis no longer matches.

## What Markitect deliberately does not do

Markitect does not currently:

- infer architecture from source-code ASTs, symbols, or call graphs;
- decide whether prose is semantically true;
- infer hidden dependencies from Markdown links;
- treat generated provider files as canonical knowledge;
- call a model as part of deterministic `check`, `context`, or `impact`;
- replace your CI platform, work-item system, or source-control system;
- claim that a passing structural check proves human acceptance or software correctness.

Project-owned checks and configured adapters integrate specialized tooling without moving domain-specific analysis into the deterministic core.

## The v0.10.0 canonical engineering model

The v0.10.0 release extends the deterministic foundation with project-owned, versioned Domain definitions: closed resource schemas, typed references within a custom Domain, relation-specific context and invalidation, and a finite set of structural constraints. The bundled AI-working Domain can explicitly use registered custom-domain resources through qualified references. The normalized model feeds configured adapters without exposing the core to technology-specific analysis. Domain definitions can be selected locally or from exact pinned content packages.

The [canonical engineering example](examples/canonical-engineering/README.md) demonstrates software and delivery domains. The same structured policy appears in generated documentation and agent context and determines the check result. `model` exports validated resources, relationships, policy definitions and source identities. `reconcile` observes outputs, captures a plan, applies configured writes with `--write`, and verifies the result; changed inputs and altered plans are rejected.

The separately built, read-only .NET reference adapter checks literal, unconditional `ProjectReference` declarations for explicitly mapped resources. It reports unsupported MSBuild constructs as incomplete and does not claim a complete build-system analysis. See [its setup and evidence limits](cmd/markitect-adapter-dotnet/README.md).

Machine results establish only configured assertions for their fixed inputs. They do not prove semantic truth, confer human acceptance, or establish product-market benefit. The [roadmap](docs/implementation-plan.md) records shipped coverage and its limits; the [strategy sources](docs/strategy/README.md) separate adopted product decisions from benefit hypotheses that still need real-world measurement.

<!-- markitect-release:install:start -->

## Install Markitect

The current [v0.10.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.10.0) provides Windows and Linux amd64 binaries. The commands below download a fixed version from the public release, check its published SHA-256 digest, and install it for your user. No Go installation or GitHub login is needed. Git is needed for Markitect commands that read Git revisions.

**Windows (PowerShell):**

```powershell
if (-not [Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([Runtime.InteropServices.OSPlatform]::Windows) -or [Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne [Runtime.InteropServices.Architecture]::X64) { throw 'Only Windows amd64 is released.' }
$tag = 'v0.10.0'
$asset = "markitect-$tag-windows-amd64.exe"
$sha256 = 'da230c607a8779c882f825944353f629def22ff9f84be21eeb6f39c2e7f0f230'
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
if ($LASTEXITCODE -ne 0 -or $installed -ne 'Markitect 0.10.0 (windows/amd64)') { throw 'Installed CLI version check failed.' }
$installed
Remove-Item -LiteralPath $source, $download
```

**Linux (bash):**

```bash
(
  set -e
  [ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] || { echo 'Only Linux amd64 is released.' >&2; exit 1; }
  tag=v0.10.0
  asset="markitect-$tag-linux-amd64"
  sha256=520b35bee308ce06f3e7d539ea593da25e6b3adf609baa0262b27c130f4e25a6
  download="$(mktemp -d)"
  trap 'rm -rf "$download"' EXIT
  curl -fLsS "https://github.com/Glacius-Labs/Markitect/releases/download/$tag/$asset" -o "$download/$asset"
  printf '%s  %s\n' "$sha256" "$download/$asset" | sha256sum -c -
  install -D -m 0755 "$download/$asset" "$HOME/.local/bin/markitect"
  "$HOME/.local/bin/markitect" version
)
export PATH="$HOME/.local/bin:$PATH"
```

Add `~/.local/bin` to your shell startup file if it is not already on `PATH`. For developers who already use Go 1.27.1 or later, `go install github.com/Glacius-Labs/Markitect/cmd/markitect@v0.10.0` is a shorter source-build option. macOS and arm64 binaries are not currently released. For signed release and asset attestation verification, project pinning, and upgrades, follow the [distribution guide](integration/README.md). The CLI's `install` command installs a **project pin**, not the CLI on your computer.

<!-- markitect-release:install:end -->

<!-- markitect-release:try:start -->

## Try the installed CLI

Clone the v0.10.0 synthetic example and run a structural check with the installed binary. This reads the example without modifying an adopting repository.

Windows PowerShell:

~~~powershell
git clone --depth 1 --branch v0.10.0 https://github.com/Glacius-Labs/Markitect.git markitect-sample-v0.10.0
if ($LASTEXITCODE -ne 0) { throw 'Could not get the v0.10.0 synthetic example.' }
markitect version
if ($LASTEXITCODE -ne 0) { throw 'Version check failed.' }
markitect check --repo .\markitect-sample-v0.10.0\examples\minimal
if ($LASTEXITCODE -ne 0) { throw 'Example check failed.' }
~~~

Linux amd64:

~~~sh
set -e
git clone --depth 1 --branch v0.10.0 https://github.com/Glacius-Labs/Markitect.git markitect-sample-v0.10.0
markitect version
markitect check --repo ./markitect-sample-v0.10.0/examples/minimal
~~~

The [minimal example](examples/minimal/README.md) is a synthetic, executable fixture. To use Markitect in your own repository, follow the [release installation and verification guide](integration/README.md) to preview and install a project pin.

<!-- markitect-release:try:end -->

## Start a project

For a complete first-project exercise, follow the [onboarding walkthrough](docs/onboarding.md).

A new project can preview its minimal Markitect layout before anything is written:

```sh
markitect init --repo . --name my-project --namespace engineering
```

On a named non-protected Git branch, apply the reviewed plan with:

```sh
markitect init --repo . --name my-project --namespace engineering --write
```

The default canonical Area is:

```text
.markitect/areas/engineering/
```

Initialization deliberately does not invent your engineering rules, checks, packages, or provider outputs.

## Try it from a source checkout

From a Markitect source checkout, with Go 1.27.1 or later:

```sh
go run ./cmd/markitect check --repo examples/minimal
go run ./cmd/markitect context --repo examples/minimal --namespace sample --kind Skill --name rollback-review
go run ./cmd/markitect check --repo examples/repository-layout
```

The examples are executable fixtures rather than product-specific policy.

## Design principles

### One canonical owner

Engineering knowledge should not become independently maintained copies across providers.

### Explicit relationships

Ownership and dependency should be modeled rather than reconstructed from prose whenever they matter structurally.

### Projection over duplication

Generated Markdown, Codex, and Claude outputs are projections of canonical sources.

### Deterministic where possible

If structure can be checked deterministically, do not repeatedly ask an LLM to rediscover it.

### Semantic humility

Markitect does not claim that structural validity proves business correctness or semantic truth.

### Provider independence

The canonical model should outlive any one AI provider or output format.

## Documentation

- [Usage and upgrade notes](docs/usage.md): project format, CLI behavior, and schema transitions.
- [Architecture](docs/architecture.md): product model, evidence, verification, and boundaries.
- [Repository layout](docs/repository-layout.md): canonical Areas, generated outputs, and layout conventions.
- [Project artifact inputs](docs/documentation.md): exact ordinary files used by resources and fixed-run context.
- [Provider adapters](docs/provider-adapters.md): Codex, Claude, shared entrypoints, and output ownership.
- [Content packages](docs/content-packages.md): exact direct pins for offline content archives.
- [Source snapshots](docs/source-snapshots.md): generic snapshot values and the Git acquisition boundary.
- [Roadmap](docs/implementation-plan.md): released behavior, current scope, and deferred options.
- [Operations](docs/operations.md): source development, checks, verification, and release handling.
- [Integration](integration/README.md): verified downloads, project pins, upgrades, and release publication.
- [Development](CONTRIBUTING.md): code ownership and contribution checks.
- [Third-party notices](internal/licenses/notices.md): bundled upstream attribution, also available offline with `markitect licenses`.

## License

Markitect is licensed under the [Apache License 2.0](LICENSE). The Glacius Labs and Markitect names and logos are not licensed as trademarks by that license. Third-party notices are listed separately above.
