# Repository layout

A Markitect project keeps its canonical control plane separate from application code. The project selector and policy live in `.markitect/project.yaml`; the canonical model and runtime are selected from that manifest. Managers describe responsibility slices and artifact ownership, while application files remain ordinary project files.

## Recommended target-project layout

```text
.markitect/
  project.yaml                 selected model, policy, checks and coverage
  model/                       Manager and owned concept/rule/workflow files
  runtime.yaml                 local execution roles and finite budgets
  workflows/model-first.md    installed shared workflow
  state/                       typed Work Item explorations and decisions
  runs/                        plans, candidates, receipts and verification
.agents/skills/                optional Codex guidance installed by onboarding
.claude/skills/                optional Claude Code guidance installed by onboarding
AGENTS.md / CLAUDE.md          repository-owned instructions with managed blocks
src/                           project-selected application artifacts
tests/                        project-selected tests
docs/markitect/project.md     generated readable view of accepted model
```

These application paths are examples, not language conventions. The model tree follows conceptual ownership rather than mirroring source directories. Give each ordinary artifact one accountable Manager; express shared responsibility and explicit dependencies in the model. Markitect does not infer behavior, dependencies, or ownership from path names or Markdown links.

The generated `docs/markitect/project.md` is a readable view of the canonical model, not an independent source of intent. Keep the model in YAML, review generated views after changes, and commit accepted model changes under the adopting repository's policy before implementation readiness.

## Ownership and boundaries

| Area | Owns |
|---|---|
| `.markitect/project.yaml` and selected model paths | Project identity, policy, Managers, concepts, rules, artifact responsibilities, and checks |
| `.markitect/runtime.yaml` | Explicit local provider/runtime selection, roles, helper policy, and finite budgets |
| `.markitect/state/` and `.markitect/runs/` | Durable workflow records, proposals, plans, candidates, receipts and verification evidence |
| `AGENTS.md`, `CLAUDE.md`, `.agents/skills/`, `.claude/skills/` | Repository-local outer-agent guidance and discoverable skills; onboarding preserves custom material outside Markitect-managed blocks |
| Application paths and `tests/` | Adopting repository implementation, tests, and local acceptance policy |
| `docs/` | Human-owned docs plus the explicitly configured generated model view |

The MCP server is selected for one explicit project root at startup; tool calls cannot redirect its authority. Local execution uses the caller's permissions. Native workers receive fresh owned candidate workspaces and bounded model/artifact context; only guarded Apply updates the adopting checkout. Candidate workspaces are not an operating-system security boundary.

## Source repository layout

Markitect source keeps its Go module at the repository root. Production entrypoints and packages live under `src/cmd/` and `src/internal/`; external harnesses that import Host packages live under `src/harness/`. The example fixture data remains at the repository root under `examples/`. Build and test commands run from the repository root, so production package paths include the `src/` prefix. This product-source layout is separate from the target-project example above, where each adopting project chooses its own application paths.

```text
src/
  cmd/                         executable entrypoints; delegate to Host
  internal/
    core/                      structural compiler and normalized values
    modules/                   separate capability implementations
    host/                      composition, workflows, CLI and controlled writes
    infrastructure/            source acquisition and platform operations
    tooling/                   import checks, provider harnesses and distribution
  harness/                     executable scenarios and integration tests
docs/                          guides, design, work queues and dated evidence
examples/                      synthetic projects and deterministic fixtures
experiments/                   bounded pilot harnesses and preserved evidence
benchmark/                     immutable release-comparison workloads
integration/                   standalone distribution bootstrap
packaging/winget/              versioned portable manifests
schema/                        generated schema views
scripts/                       repository maintenance and evaluation helpers
```

Go tests live beside the package they exercise; cross-package scenarios live in `src/harness/`. Add a file to the package that owns its responsibility before considering a new package. [Contribution checks](../CONTRIBUTING.md#ownership-and-layout) and the [Module guide](development/modules.md) define the import boundaries; directory names alone do not enforce them.

Markitect's own engineering-resource configuration is `markitect.yaml`, with resources under `.markitect/areas/` and module configuration under `.markitect/modules/`. It uses the structural compiler to check this repository's contributor guidance. It is separate from the model-first adopting-project scaffold above. [Managed-artifact accounting](../markitect-artifacts.yaml) declares a bounded source scope, not every file in the checkout.

The root [AGENTS.md](../AGENTS.md) routes engineering tasks. `.agents/skills/`, `.claude/skills/` and `docs/markitect/` are generated guidance views; change their canonical YAML and regenerate the outputs. `.github/` owns hosted workflows and issue/PR templates; `.githooks/` contains configured hook artifacts. `runs/c11checktools/` retains original/replacement check-program fixtures and is separate from an adopting project's `.markitect/runs/`. New local logs and build outputs belong outside source inputs, as described in [CONTRIBUTING](../CONTRIBUTING.md#release-work).

Use the [documentation map](README.md) for current guides and canonical owners, the [examples map](../examples/README.md) for executable fixtures, and the [experiments map](../experiments/README.md) for bounded evaluations. Preserve dated evidence and immutable workload paths when reorganizing navigation; a maintained index can clarify their role without changing their original bytes.

Historical Project/Domain inputs remain versioned compatibility surfaces; canonical Projection inputs belonged to the v0.14.1 alpha, which current source has removed. The currently published v0.14.1 pin retains its own paths and behavior; no in-place migration or automatic removal is implied by this recommended target layout. See the [compatibility reference](usage.md#legacy-projectdomain-cli-compatibility) and [distribution guide](../integration/README.md).
