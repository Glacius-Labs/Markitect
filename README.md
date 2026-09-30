# Markitect

Markitect treats the rules, workflows, skills, agents and context around AI-assisted work as an engineered system. Natural language carries meaning; typed YAML makes ownership and dependencies explicit. A Go CLI validates the graph, compiles context, detects affected resources and generates managed views.

Humans describe intent. Agents maintain the resources. Markitect supports authoring and checks their structure. Core authoring guidance ships with the tool; structural queries help agents find the canonical owner and explain dependencies.

This is the independent [Glacius Labs Markitect source repository](https://github.com/Glacius-Labs/Markitect). Konfyra and the AI Cockpit consume the same pinned release; neither repository is required to build or test the tool. The published private release is **[v0.1.0](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.1.0)**. Verify its immutable metadata and successful run against the intended source commit. Consumer integration and acceptance are recorded separately in each consumer's delivery route.

## Start

Building and testing this source checkout requires Git and Go 1.27.1 or later. Consumer bootstrap execution has the same Go prerequisite. Published Windows amd64 and Linux amd64 native binaries can run the CLI without Go; consumer verification still needs the tools required by its gates. [Integration](integration/README.md) documents download verification and installation.

```powershell
# Optional caches when the shared Go cache is not writable
$env:GOCACHE = Join-Path $PWD '.cache/go-build'
$env:GOTMPDIR = Join-Path $PWD '.cache/tmp'
New-Item -ItemType Directory -Force $env:GOCACHE, $env:GOTMPDIR, bin | Out-Null
go test ./...
go build -o bin/markitect.exe ./cmd/markitect
./bin/markitect.exe version
./bin/markitect.exe authoring
./bin/markitect.exe check --repo examples/minimal
./bin/markitect.exe find --repo examples/minimal --query rollback --kind Rule
./bin/markitect.exe explain --repo examples/minimal --namespace sample --kind Rule --name rollback-review
./bin/markitect.exe context --repo examples/minimal --namespace sample --kind Skill --name rollback-review
```

On Linux/macOS, use a writable Go cache and `go build -o bin/markitect ./cmd/markitect`. The CLI flags are identical.

The [minimal example](examples/minimal/README.md) demonstrates all six content kinds, a Contract binding and an ordinary file input. Its generated views and compiled context are covered by Go tests. It is an example to copy, not an installed template or a reusable package dependency.

## Read

- [Documentation](docs/README.md): usage, current architecture, design decisions, roadmap and pilot assessment.
- [Development](CONTRIBUTING.md): checks, boundaries and release preparation.
- [Schemas](schema/Project.yaml): generated YAML schemas; parser and graph checks remain authoritative.
- [Consumer integration](integration/README.md): pinned source packages and bootstrap.
- [Operations and releases](docs/operations.md): parallel reviews, supported platforms, failure recovery and release gates.
- [Production delivery assessment](docs/production-assessment.md): `v0.1.0` release evidence, source support boundary and separate consumer acceptance.

Implemented: strict YAML, six resource kinds, scopes, references, Contracts, immutable Git snapshots, structural queries, bundled core authoring, context compilation, impact, managed rendering, advisory review reuse, fixed-revision release bundles and safe install planning/application. Content packages, template initialization, MCP/LSP and Kubernetes are design work. Markitect does not prove arbitrary prose consistent or enforce an agent's runtime behavior.

The Go module is `github.com/Glacius-Labs/Markitect`. The repository is private. GitHub immutable releases are enabled. Release CI produces and tests a run-scoped artifact; an authenticated release owner publishes it with the local release tool after a read-only preflight. The Actions `GITHUB_TOKEN` cannot read the admin-only immutable-release setting (live API returns 403), so CI does not publish and no PAT is stored in workflow secrets. Server-side branch rules could not be configured under the current private-repository plan; the CLI rejects `main` and `master`, but this is not server-enforced protection. The API group remains the placeholder `markitect.example.org/v1alpha1`; changing it needs an explicit format migration. A controlled API domain and public distribution license remain separate decisions. See [consumer integration](integration/README.md) for release verification and installation.
