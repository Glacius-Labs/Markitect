# Repository layout

This repository holds Markitect's source and its documentation. It is developed as an ordinary software project and keeps no canonical self-model ([DEC-076](../concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)); `.markitect/` paths in code, tests and fixtures belong to target projects. For the layout of a project that uses Markitect, see [the project's Markitect folder](../guides/model-language.md#the-projects-markitect-folder).

| Path | Holds |
|---|---|
| `cmd/` | Go entrypoints. `markitect` (the CLI) and `markitect-agentd` (runs inside the runtime container) ship; `markitect-release` is the maintenance command. |
| `internal/` | Private Go packages of the product and its maintenance services, including the standard Libraries (`internal/libraries`) and the agent skills (`internal/skills`) that the binary embeds. The [code map](code-map.md) lists every Go package of the module, its layer and owning guide; the [rule table](../../internal/tooling/architecture/rules.go) it is generated from states the allowed imports, and the [architecture](../architecture.md) describes the layers and contracts. |
| `docs/` | Concept and decisions, design, guides, development, current evidence and the archive index [`docs/archive.md`](../archive.md). |
| `testdata/` | Fixtures shared by the evaluation: the canonical Library model M1, still in V1's form until step T2 converts it. |
| `tools/` | Independent tools that import no product code: the packaged-source smoke, the Library and Makerspace oracles, the Makerspace reference modules, the comparison study's runner and assessment, and the evaluation suite with its judges and baseline variant. |
| `scripts/` | Maintainer scripts: documentation link check, CI test and flake reports. |
| `packaging/` | What ships besides the binary: the runtime image, the consumer bootstrap and the distribution guide. |
| `.github/` | Manual CI, release and flake-report workflows. |

**Why there is no `src/`.** Markitect is one Go module. Go keeps `cmd/` and `internal/` at the module root, and the source packager refuses a `src/` layout.

**Where new things go.**

| New item | Place | Also update |
|---|---|---|
| Go package | `internal/<responsibility>/` | One row in the architecture rule table and a package doc comment; regenerate the code map |
| Shipped command | `cmd/<name>/` | One row in the architecture rule table; the smoke's build list |
| File the binary embeds | Inside the embedding package, such as `internal/libraries/` | The source packager's data selection and its tests |
| Fixture used by one package | `<package>/testdata/` | Nothing else |
| Fixture shared across packages or used by the evaluation | `testdata/` | Keep bytes stable |
| Outside check that must not import product code | `tools/` | CI step |
| Maintainer script | `scripts/` | Tests next to it |
| Document | The owning folder under `docs/`, see [documentation maintenance](documentation.md) | Local map |
| Historical material | Not on Main: it stays at an archive point and [`docs/archive.md`](../archive.md) points to it | Index entry |

**What enforces this.** The architecture gate fails on a Go package without a row in its rule table; the source packager fails on unknown roots inside `cmd/` and `internal/` and on nested modules; `scripts/check-docs.py` fails on broken local links.
