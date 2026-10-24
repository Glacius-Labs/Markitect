# Repository layout

This repository holds Markitect's source, its own canonical self-model and its documentation. For the layout of a project that uses Markitect, see [Project layout](../guides/project-workflow.md#project-layout).

| Path | Holds |
|---|---|
| `cmd/` | Go entrypoints. `markitect` (the CLI) and `markitect-agentd` (runs inside the runtime container) ship; `markitect-check-architecture`, `markitect-check-modules` and `markitect-release` are maintenance commands. |
| `internal/` | Private Go packages of the product and its maintenance services. The [code map](code-map.md) lists every Go package of the module, its layer and owning guide; [modules](modules.md) states the allowed imports. |
| `.markitect/` | Markitect's own canonical model, registry shards, ignore rules, execution declarations and module pins. Every file in this repository is accounted for here. |
| `docs/` | Concept and decisions, design, guides, development, current evidence and the archive index [`docs/archive.md`](../archive.md). |
| `testdata/` | Byte-stable fixtures shared by several packages, the smoke and the qualification: the `markitect/core` dependency and the canonical Library model M1. |
| `tools/` | Independent outside checks that import no product code: the packaged-source smoke and the Library oracle. |
| `scripts/` | Maintainer scripts: documentation link check, CI test and flake reports, the native Library Apply operator script. |
| `packaging/` | What ships besides the binary: the runtime image, the consumer bootstrap and the distribution guide. |
| `examples/` | Copyable runtime-profile and run configuration examples; tests keep them valid. |
| `.github/`, `.githooks/` | Manual CI, release and flake-report workflows; the pre-commit model check. |

**Why there is no `src/`.** Markitect is one Go module. Go keeps `cmd/` and `internal/` at the module root, and the source packager refuses a `src/` layout.

**Why `.markitect/` is here.** Markitect is developed model-first with its own product. `markitect check` runs in CI and before every commit and fails on any file without a registration or an explained ignore entry. The self-model is [model.yaml](../../.markitect/model.yaml), [model-lock.yaml](../../.markitect/model-lock.yaml) and `.markitect/model/`; [project.md](../markitect/project.md) is its officially generated view. Keep new build caches and logs outside governed source inputs; Gitignore does not filter the census.

**Where new things go.**

| New item | Place | Also update |
|---|---|---|
| Go package | `internal/<responsibility>/` | Code map entry, import rules, registry entry |
| Shipped command | `cmd/<name>/` | Source packaging roots, architecture gate |
| Fixture used by one package | `<package>/testdata/` | Registry entry |
| Fixture shared across packages or used by qualification | `testdata/` | Registry entry; keep bytes stable, locks pin them |
| Outside check that must not import product code | `tools/` | Registry entry, CI step |
| Maintainer script | `scripts/` | Registry entry, tests next to it |
| Document | The owning folder under `docs/`, see [documentation maintenance](documentation.md) | Local map, registry entry |
| Historical material | Not on Main: it stays at an archive point and [`docs/archive.md`](../archive.md) points to it | Index entry |

**What enforces this.** The architecture gate fails on Go code outside a mapped responsibility; the source packager fails on unknown roots inside `cmd/` and `internal/` and on nested modules; `markitect check` fails on unaccounted files; `scripts/check-docs.py` fails on broken local links.
