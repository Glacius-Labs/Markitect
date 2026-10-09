<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/markitect-wordmark-dark.svg">
    <img src="assets/markitect-wordmark-light.svg" alt="Markitect" width="360">
  </picture>
</h1>

**Humans shape the project world. AI works within it. Markitect keeps their model and repository aligned.**

Markitect is a model-first engineering workflow. A project records its accepted goals, rules, architecture, responsibilities, expected artifacts and checks as a recursively managed tree under `.markitect/`. Agents receive bounded Manager tasks, produce candidate changes, and return evidence for integration and verification. The owner-readable project document is generated from that model; it is a view, not a second source of truth. See the [product vision](docs/vision.md) and the [model-first workflow](docs/project-workflow.md).

The model in a committed project revision is the accepted repository specification Markitect uses for that revision. A draft, distillation, uncommitted edit, passing check, plan digest, commit identity or supplied provenance string does not authenticate human approval. Project commands run with the caller's local permissions; native agent instructions and scoped payloads are not an operating-system sandbox.

The [current source](docs/project-workflow.md#choose-the-executable) provides the new `markitect project` workflow. It is not included in the published [v0.14.1 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.14.1), whose install instructions below remain bound to that immutable release. Build or run the exact source candidate to try newer commands; no new release is promised here.

Start with the [Shop walkthrough](examples/project-world/README.md) for a runnable vertical-slice example, then follow the [project guide](docs/project-workflow.md) to initialize, model, delegate and verify a repository. Existing Project/Domain commands and canonical projection tools remain available as compatibility contracts; their examples and validation records are preserved in the [documentation index](docs/README.md). Historical comparisons remain evidence, not proof that this model-first workflow is superior; quality, cost and human-attention benefits remain unproven.

## Why Markitect?

Autonomous implementation can produce correct local changes while losing broader intent: another representation remains stale, an affected module is overlooked, or a project rule is ignored. Markitect aims to make the desired world explicit and connect its evolution to the governed project's actual state. Owners should spend attention on architecture and meaningful decisions rather than repeatedly synchronizing every representation. This is a quality and autonomy hypothesis to test, not a measured result.

Engineering rules often have several consumers.

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

The Manager tree assigns documentation, code, configuration and agent guidance to their accountable owners.

These mechanisms aim to remove independently maintained copies of modeled intent and make consistency questions answerable structurally, leaving human attention for engineering decisions. Their upkeep must be included when testing whether that goal is achieved.

## Working method

The current project workflow is organized by the model's recursive Manager tree:

```text
owner explores and decides project intent
    -> accepted model revision and generated readable view
    -> impact-scoped Manager assignments
    -> bounded implementation and integration
    -> full Manager and declared-check verification
    -> guarded Apply, bounded repair or an owner decision
```

The intended user journey begins with conversational Explore and explicit decisions, then commits accepted model changes before implementation. The source supports durable Explore records, scoped readiness, model edits, impact, Manager runs, integration, verification, guarded Apply, recovery, readable documentation, full repository classification, structured Brownfield proposal sessions, and the composed/resumable `project deliver` operation. These structured interfaces do not establish the quality of natural-language exploration or human approval. See the [workflow and status](docs/project-workflow.md), the [native work-item delivery checklist](docs/design/project-world/native-work-item-delivery.md), and [roadmap](docs/implementation-plan.md).

At every scope, child success is not parent integration success. Implementations may vary within their declared freedom; exact bytes are required only for deterministic contracts. Repair drift against unchanged intent without inventing model changes. When intent changes, update its canonical owner, review and commit that model before implementation. The [measurement procedure](docs/measurement.md#staged-method-readiness) defines how to evaluate quality, missed fanout, conformity, maintenance effort and human intervention. No superiority or productivity advantage is established.

Agents working on this repository begin through [AGENTS.md](AGENTS.md) and its fixed Markitect engineering context. The vision, operating methodology and measurement procedure are declared inputs of that context. Adopting projects select their own ontology, checks, ownership and delegation policy.

## The model owns desired project intent

A Markitect project keeps canonical model files and runtime configuration under `.markitect/`. The recursive Manager tree owns conceptual slices, artifact responsibilities and required checks. Source files, tests, configuration and documentation remain project artifacts assigned to their declared owners. The generated readable view under `docs/markitect/project.md` reflects the accepted model; agents and people propose changes to the canonical model rather than editing its generated view as a second authority.

```text
owner decisions -> committed .markitect model -> managers and file owners
                                      |                  |
                                      v                  v
                              generated readable view   candidate changes
                                                         |
                                                         v
                                      integration -> full Verify -> guarded Apply
```

This is the workflow direction, not a claim that Markitect generates or semantically verifies every artifact. Project-owned checks run against a fixed candidate snapshot and establish only their declared assertions. Agent output remains a candidate until it passes the selected evidence and is applied under the caller's authority. A commit supplies a stable model basis; it does not authenticate who approved the model.

The older Project/Domain and canonical Projection formats remain compatibility surfaces. Their versioned contracts, release evidence and examples are retained, but they are not a second active product workflow. See [legacy CLI compatibility](docs/usage.md#legacy-projectdomain-cli-compatibility).

## Historical canonical Projection example

The following Commerce walkthrough preserves evidence for the v0.14.1 canonical Projection alpha. It is a compatibility example, not the current model-first product entrypoint.

The [intent-change walkthrough](examples/classic-commerce/intent-change.md) follows one rule from `UseCase.purpose` through selected .NET and Markdown policies to a handler, readable documentation and an independent behavior probe. It then changes the minimum order quantity from one to two, exposes stale evidence, refuses reuse of the old Apply candidate, and requests fresh verification. Model acceptance and review of exact materialization bytes are separate decisions.

This walkthrough is an unreleased source example on `main` using the published v0.14.1 **experimental alpha** CLI. Obtain the [separate pinned example checkout](examples/classic-commerce/README.md#obtain-the-example-source); the release-tag checkout used by the stable quickstart below does not contain it. A [source-bound Windows replay](docs/validation/classic-commerce-entrypoint.md#corrected-uninterrupted-replay-2026-10-08) completed the initial and changed-intent cycles with real finite .NET checks; this is bounded example evidence. Its fixed actor demonstrates protocol mechanics; it does not translate arbitrary requests, establish semantic acceptance or run an autonomous team. The [example README](examples/classic-commerce/README.md) gives prerequisites and commands.

<a id="the-current-workflow"></a>

## Legacy Project/Domain CLI compatibility

The released v0.14.1 CLI and its Project/Domain resources remain supported compatibility contracts. They are not the recommended workflow for new projects. Keep existing repositories on their current contract unless undertaking an explicit migration; do not mix these resource formats with the new `.markitect/model/` format.

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

The canonical YAML remains the source for the modeled engineering decisions. Human documentation remains human-owned unless an explicit generated view is selected.

## Why this matters for AI-first engineering

Task context, deterministic checks and explicit policy evolution are ways to make human engineering decisions usable by many agents. Markitect's [vision](docs/vision.md) defines their role in executable governance and the desired shift from routine supervision to architecture-level decisions. Relevant context, fewer interventions and sustainable maintenance remain [evaluation questions](docs/measurement.md#human-attention-and-delegated-work); generated instructions do not prove agent compliance.

## Current project layout

New model-first projects keep their canonical model and runtime under `.markitect/`:

```text
project/
├── .markitect/
│   ├── project.yaml
│   ├── runtime.yaml
│   ├── model/
│   │   └── <responsibility-owned vertical slices>/
│   ├── ignore.yaml
│   └── drafts/              # explicit proposals and operational records
├── docs/
│   ├── ... human-owned documentation ...
│   └── markitect/project.md # generated readable model view
└── src/                     # assigned to a declared Manager
```

The project model declares conceptual slices, Managers, statements, expected artifacts, concrete file responsibility and checks. Names or directory adjacency do not infer semantic meaning or dependencies. See the [model-first project guide](docs/project-workflow.md) for the implemented schema and source/release boundary.

The earlier `.markitect/areas/<namespace>` layout belongs to the legacy Project/Domain CLI; see [Repository layout](docs/repository-layout.md) for its exact compatibility contract.

## What the model-first source workflow provides

- Stores a recursively managed project model, manager responsibilities, artifact requirements and checks under `.markitect/`.
- Compiles context and impact from fixed source snapshots, with targeted implementation and complete Manager verification for full-coverage projects.
- Supports exact model proposals, explicit Brownfield selection, bounded Manager execution, integration, independent review, declared checks, recovery and guarded Apply.
- Classifies the full repository in full-coverage mode; every ordinary path must have an explicit model, tool-owner or ignore classification before closure.
- Generates a readable project-model document and can install native Codex/Claude contributor instructions.
- Preserves prior Project/Domain commands and the canonical Projection alpha as compatibility contracts.
- Keeps model truth, proposals, machine evidence and human authority distinct. Checks prove only their declared assertions; project files are not semantically understood by generic Core.

Explore/readiness records and structured Brownfield return to model proposals are available in current source. A natural-language interview's completeness, end-to-end acceptance and productivity effects remain unproven. Native instructions do not prevent an agent with write access from bypassing Markitect. Quality, cost and autonomy benefits remain unmeasured.

## What Markitect deliberately does not do

Markitect does not currently:

- infer architecture from source-code ASTs, symbols, or call graphs;
- decide whether prose is semantically true;
- infer hidden dependencies from Markdown links;
- treat generated provider files as canonical knowledge;
- call a model as part of deterministic `check`, `context`, or `impact`;
- replace your CI platform, work-item system, or source-control system;
- claim that a passing structural check proves human acceptance or software correctness.

Project-owned checks and target tools integrate specialized evidence without moving domain-specific analysis into the generic compiler. A candidate plan binds work and inputs; it is not human approval, and successful evidence proves only its declared checks.

## Historical model background: v0.10.0

The following describes a preserved release contract and examples, not the current model-first project setup.

The v0.10.0 release extends the deterministic foundation with project-owned, versioned Domain definitions: closed resource schemas, typed references within a custom Domain, relation-specific context and invalidation, and a finite set of structural constraints. The bundled AI-working Domain can explicitly use registered custom-domain resources through qualified references. The normalized model feeds configured adapters without exposing the core to technology-specific analysis. Domain definitions can be selected locally or from exact pinned content packages.

The [canonical engineering example](examples/canonical-engineering/README.md) demonstrates software and delivery domains. The same structured policy appears in generated documentation and agent context and determines the check result. `model` exports validated resources, relationships, policy definitions and source identities. `reconcile` observes outputs, captures a plan, applies configured writes with `--write`, and verifies the result; changed inputs and altered plans are rejected.

The separately built, read-only .NET reference adapter checks literal, unconditional `ProjectReference` declarations for explicitly mapped resources. It reports unsupported MSBuild constructs as incomplete and does not claim a complete build-system analysis. See [its setup and evidence limits](cmd/markitect-adapter-dotnet/README.md).

Machine results establish only configured assertions for their fixed inputs. They do not prove semantic truth, confer human acceptance, or establish product-market benefit. The [roadmap](docs/implementation-plan.md) records shipped coverage and its limits; the [strategy sources](docs/strategy/README.md) separate adopted product decisions from benefit hypotheses that still need real-world measurement.

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

## Start a model-first project

The current model-first commands are available from this source checkout, not the published v0.14.1 binary. From the Markitect source root, preview initialization for the target repository:

```powershell
go run ./cmd/markitect project init --repo C:\src\my-project --name my-project
```

Review the returned paths and exact plan digest, then apply the preview on a writable feature branch:

```powershell
go run ./cmd/markitect project init --repo C:\src\my-project --name my-project --write
go run ./cmd/markitect project check --repo C:\src\my-project
go run ./cmd/markitect project index --repo C:\src\my-project
go run ./cmd/markitect project document --repo C:\src\my-project
```

Then follow [Project workflow](docs/project-workflow.md) for onboarding, durable Explore, scoped readiness, model changes, implementation plans, Manager runs, Verify, guarded Apply and recovery. Current source also has the composed, resumable `project deliver` operation and structured Brownfield sessions for repeated model proposals and reviewed adoption. The [native work-item delivery checklist](docs/design/project-world/native-work-item-delivery.md) records the intended user journey and its remaining validation gates; it is not proof that those gates passed. The [Shop example](examples/project-world/README.md) provides the executable model-first project.

## Legacy CLI quick check from a source checkout

These commands are retained for repositories using the released Project/Domain contract; they are not the model-first quick start:

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

### Generated views over duplicated authority

Readable documentation and native agent instructions derive from canonical model sources where supported; they do not become independent authorities.

### Deterministic where possible

If structure can be checked deterministically, do not repeatedly ask an LLM to rediscover it.

### Semantic humility

Markitect does not claim that structural validity proves business correctness or semantic truth.

### Provider independence

The canonical model should outlive any one AI provider or output format.

## Documentation

- [Product vision](docs/vision.md): human intent, agent responsibility, long-term operating model and unproven benefit hypothesis.
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
- [Third-party notices](internal/tooling/licenses/notices.md): bundled upstream attribution, also available offline with `markitect licenses`.

## License

Markitect is licensed under the [Apache License 2.0](LICENSE). The Glacius Labs and Markitect names and logos are not licensed as trademarks by that license. Third-party notices are listed separately above.
