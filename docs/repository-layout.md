# Repository layout

This page describes the layout of an adopting model-first project and Markitect's own source repository.

## Recommended target-project layout

A Markitect project keeps its control plane in `.markitect/` and its application files where the project chooses. `markitect init` creates the model tree; `markitect onboard` adds guidance for the selected coding client.

```text
.markitect/
  project.yaml                 project identity and policy
  model/                       Managers, concepts, rules and artifact responsibilities
  runtime.yaml                 inner roles, profiles and finite budgets
  workflows/model-first.md     shared workflow installed by onboarding
  records/
    run/                       plans, candidates, receipts and verification evidence
    exploration/               explorations and decisions
    briefing/                  accepted History briefings
    brownfield/                Brownfield sessions
  runs/                        retained transport inputs and private runtime material
  .gitignore                   excludes local state and generated views
.agents/skills/                optional Codex skills installed by onboarding
.claude/skills/                optional Claude Code skills installed by onboarding
AGENTS.md / CLAUDE.md          repository instructions with managed blocks
src/                           project-selected application files
tests/                         project-selected tests
docs/markitect/project.md      generated readable view of the accepted model
```

- Application paths are examples, not conventions. The model tree follows conceptual ownership, not source directories.
- Each ordinary file has one accountable Manager. Shared responsibility and dependencies are explicit in the model.
- Markitect does not infer behavior, dependencies or ownership from paths, names or Markdown links.
- The generated project view is not a source of intent. Commit accepted model changes under the repository's policy before implementation.
- Markitect keeps domain records under `.markitect/records/`; retained transport/private runtime material stays under `.markitect/runs/`. New Init ignores records; existing projects add the anchored rule through a reviewed setup edit. [Cutover](project-operations.md#record-layout-and-cutover) requires quiescent writers and preserves old domain bytes.

## Ownership and boundaries

| Area | Owns |
|---|---|
| `.markitect/project.yaml` and the model paths | Project identity, policy, Managers, concepts, rules, artifact responsibilities and checks |
| `.markitect/runtime.yaml` | Local role profiles, helper policy and finite budgets |
| `.markitect/records/` | Domain-owned Run, Explore, accepted History and Brownfield records |
| `.markitect/runs/` | Retained transport inputs and private native execution/recovery material |
| `AGENTS.md`, `CLAUDE.md`, `.agents/skills/`, `.claude/skills/` | Outer-agent guidance; onboarding keeps custom text outside its managed blocks |
| Application paths and `tests/` | The adopting repository's implementation, tests and acceptance policy |
| `docs/` | Human-owned documents and the generated model view |

The MCP server is fixed to one project root at startup; tool calls cannot redirect it. Inner roles work in owned candidate workspaces, and only guarded Apply updates the checkout. Local execution uses the caller's permissions; workspaces are not an operating-system sandbox.

## Source repository layout

Markitect's Go module sits at the repository root, so package paths start with `src/`. Build and test commands run from the root.

```text
src/
  cmd/                         executable entrypoints
  internal/
    core/                      structural compiler and snapshots
    infrastructure/            Git and working-tree access
    host/                      product use cases, runtime and composition
    modules/                   capability packages, including project model
    tooling/                   import gate, release, publication, notices and providers
    testkit/                   hermetic fixtures for tests
  harness/                     executable scenarios over current project fixtures
docs/                          guides, design, history, validation and work items
examples/                      example projects and fixtures
experiments/                    bounded pilots and the case playground
benchmark/                      historical release-comparison workloads
integration/                    standalone distribution bootstrap
packaging/winget/               versioned portable manifests
scripts/                         maintenance and evaluation helpers
```

- A directory does not give a package its layer. The [code map](development/code-map.md) assigns each package its layer, and [Architecture](architecture.md#layers) describes the layers.
- Go tests live beside their package; cross-package scenarios live in `src/harness/`.
- Add a file to the package that owns the responsibility before you create a package.
- [Modules and static composition](development/modules.md) owns the import rules. The import gate enforces them; directory names do not.

This repository is modelled in [`.markitect/project.yaml`](../.markitect/project.yaml) and `.markitect/model/`, with full coverage; [`docs/markitect/project.md`](markitect/project.md) is its generated readable view. Change canonical YAML and regenerate the view through the digest-bound documentation workflow. The root [AGENTS.md](../AGENTS.md) routes engineering tasks. `.github/` holds hosted workflows and templates, and `.githooks/` the pre-commit hook. Local logs and build outputs belong outside source inputs ([CONTRIBUTING](../CONTRIBUTING.md#release-work)).

Use the [documentation map](README.md), [examples map](../examples/README.md) and [experiments map](../experiments/README.md) to find current guides, fixtures and evaluations. Keep dated evidence and immutable workload paths when reorganizing navigation.
