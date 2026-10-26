# Repository layout

This repository holds Markitect's source and its documentation. It is developed as an ordinary software project and keeps no canonical self-model ([DEC-076](../concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)); `.markitect/` paths in code, tests and fixtures belong to target projects. For the layout of a project that uses Markitect, see [Project layout](../guides/project-workflow.md#project-layout).

| Path | Holds |
|---|---|
| `cmd/` | Go entrypoints. `markitect` (the CLI) and `markitect-agentd` (runs inside the runtime container) ship; `markitect-check-architecture` and `markitect-release` are maintenance commands. |
| `internal/` | Private Go packages of the product and its maintenance services. The [code map](code-map.md) lists every Go package of the module, its layer and owning guide; [modules](modules.md) states the allowed imports. |
| `docs/` | Concept and decisions, design, guides, development, current evidence and the archive index [`docs/archive.md`](../archive.md). |
| `libraries/` | The standard Libraries (`values`, `api`) that projects vendor and pass with `--dependency`; see [standard libraries](../design/standard-libraries.md). |
| `testdata/` | Byte-stable fixtures shared by several packages, the smoke and the qualification: the `markitect/core` dependency and the canonical Library model M1. |
| `tools/` | Independent outside checks that import no product code: the packaged-source smoke and the Library oracle. |
| `scripts/` | Maintainer scripts: documentation link check, CI test and flake reports, the native Library Apply operator script. |
| `packaging/` | What ships besides the binary: the runtime image, the consumer bootstrap and the distribution guide. |
| `examples/` | Copyable runtime-profile and run configuration examples; tests keep them valid. |
| `.github/` | Manual CI, release and flake-report workflows. |

**Why there is no `src/`.** Markitect is one Go module. Go keeps `cmd/` and `internal/` at the module root, and the source packager refuses a `src/` layout.

**Where new things go.**

| New item | Place | Also update |
|---|---|---|
| Go package | `internal/<responsibility>/` | Code map entry, import rules, source packaging roots |
| Shipped command | `cmd/<name>/` | Source packaging roots, architecture gate |
| Fixture used by one package | `<package>/testdata/` | Nothing else |
| Fixture shared across packages or used by qualification | `testdata/` | Keep bytes stable; locks pin them |
| Outside check that must not import product code | `tools/` | CI step |
| Maintainer script | `scripts/` | Tests next to it |
| Document | The owning folder under `docs/`, see [documentation maintenance](documentation.md) | Local map |
| Historical material | Not on Main: it stays at an archive point and [`docs/archive.md`](../archive.md) points to it | Index entry |

**What enforces this.** The architecture gate fails on Go code outside a mapped responsibility; the source packager fails on unknown roots inside `cmd/` and `internal/` and on nested modules; `scripts/check-docs.py` fails on broken local links.
