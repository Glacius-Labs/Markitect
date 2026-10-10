# Repository layout

This page describes two layouts: the recommended layout of a project that adopts Markitect, and the layout of Markitect's own source repository.

## Recommended target-project layout

A Markitect project keeps its control plane in `.markitect/` and its application files where the project chooses. `markitect init` writes the manifest, runtime, root Manager, ignore rules and readable view; `markitect project onboard` adds the workflow, instructions and skills.

```text
.markitect/
  project.yaml                 selected model, policy, checks and coverage
  model/                       Managers and their concepts, rules and artifacts
  runtime.yaml                 inner roles, profiles and finite budgets
  workflows/model-first.md     shared workflow installed by onboarding
  state/                       explorations, decisions and briefings
  runs/                        plans, candidates, receipts and verification
  .gitignore                   keeps cache/, runs/ and views/ out of Git
.agents/skills/                optional Codex skills installed by onboarding
.claude/skills/                optional Claude Code skills installed by onboarding
AGENTS.md / CLAUDE.md          repository instructions with managed blocks
src/                           project-selected application files
tests/                         project-selected tests
docs/markitect/project.md      generated readable view of the accepted model
```

- Application paths are examples, not conventions. The model tree follows conceptual ownership, not the source directories.
- Each ordinary file has one accountable Manager. Shared responsibility and dependencies are explicit in the model.
- Markitect does not infer behavior, dependencies or ownership from paths, names or Markdown links.
- `docs/markitect/project.md` is a view of the model, not a source of intent. Commit accepted model changes under the repository's policy before implementation.
- Markitect also uses `cache/`, `views/`, `drafts/` and `write.lock` under `.markitect/` while it works.

## Ownership and boundaries

| Area | Owns |
|---|---|
| `.markitect/project.yaml` and the model paths | Project identity, policy, Managers, concepts, rules, artifact responsibilities and checks |
| `.markitect/runtime.yaml` | Local role profiles, helper policy and finite budgets |
| `.markitect/state/` and `.markitect/runs/` | Durable workflow records, plans, candidates, receipts and verification evidence |
| `AGENTS.md`, `CLAUDE.md`, `.agents/skills/`, `.claude/skills/` | Outer-agent guidance; onboarding keeps custom text outside its managed blocks |
| Application paths and `tests/` | The adopting repository's implementation, tests and acceptance policy |
| `docs/` | Human-owned documents and the configured generated model view |

The MCP server is fixed to one project root at startup; tool calls cannot redirect it. Inner roles work in owned candidate workspaces, and only guarded Apply updates the checkout. Local execution uses the caller's permissions; workspaces are not an operating-system sandbox.

## Source repository layout

Markitect's Go module sits at the repository root, so package paths start with `src/`. Build and test commands run from the root.

```text
src/
  cmd/                         executable entrypoints
  internal/
    core/                      structural compiler and snapshots
    infrastructure/            Git and working-tree access
    host/                      product use cases and runtime, and the legacy line
    modules/                   capability packages: project model, adoption helpers
    tooling/                   import gate, release, publication, notices, provider runners
    testkit/                   hermetic fixtures for tests
  harness/                     executable scenarios over the examples
docs/                          guides, design, history, validation and work items
examples/                      example projects and fixtures
experiments/                   bounded pilots and the case playground
benchmark/                     release-comparison workloads
integration/                   standalone distribution bootstrap
packaging/winget/              versioned portable manifests
schema/                        generated schema views
scripts/                       maintenance and evaluation helpers
```

- A directory does not give a package its layer. `src/internal/host` holds product packages of every layer next to the legacy line. The [code map](development/code-map.md) assigns each package its layer, and [Architecture](architecture.md#layers) describes the layers.
- Go tests live beside their package; cross-package scenarios live in `src/harness/`.
- Add a file to the package that owns the responsibility before you create a package.
- [Modules and static composition](development/modules.md) owns the import rules. The import gate enforces them; directory names do not.

This repository's own engineering resources still use the legacy line: `markitect.yaml`, `.markitect/areas/` and `.markitect/modules/`. They render `.agents/skills/`, `.claude/skills/` and `docs/markitect/` and run this repository's checks until ARCH-07 moves Markitect onto its own `.markitect/project.yaml` model. Change their canonical YAML and regenerate the views. [Managed-artifact accounting](../markitect-artifacts.yaml) declares a bounded source scope, not every file.

The root [AGENTS.md](../AGENTS.md) routes engineering tasks. `.github/` holds the hosted workflows and templates, and `.githooks/` the pre-commit hook. Local logs and build outputs belong outside source inputs ([CONTRIBUTING](../CONTRIBUTING.md#release-work)).

Use the [documentation map](README.md), the [examples map](../examples/README.md) and the [experiments map](../experiments/README.md) to find current guides, fixtures and evaluations. Keep dated evidence and immutable workload paths when you reorganize navigation.
