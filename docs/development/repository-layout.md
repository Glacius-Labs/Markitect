# Repository layout

This repository holds Markitect's source and its documentation. It is developed as an ordinary software project and keeps no canonical self-model ([DEC-076](../concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)); `.markitect/` paths in code, tests and fixtures belong to target projects. For the layout of a project that uses Markitect, see [Project layout](../guides/project-workflow.md#project-layout).

| Path | Holds |
|---|---|
| `cmd/` | Go entrypoints. `markitect` (the CLI), `markitect-agentd` (runs inside the runtime container) and `markitect-runner` (the entrypoint of the [Linux runner](../design/linux-runner.md) image) ship; `markitect-release` is the maintenance command. |
| `internal/` | Private Go packages of the product and its maintenance services. The [code map](code-map.md) lists every Go package of the module, its layer and owning guide; the [rule table](../../internal/tooling/architecture/rules.go) it is generated from states the allowed imports, and the [architecture](../architecture.md) describes the layers, contracts and invariants. |
| `docs/` | Concept and decisions, design, guides, development, current evidence and the archive index [`docs/archive.md`](../archive.md). |
| `libraries/` | The standard Libraries (`values`, `data`, `api`, `practice`, `architecture`) that projects vendor and pass with `--dependency`; see [standard libraries](../design/standard-libraries.md). |
| `testdata/` | Byte-stable fixtures shared by several packages, the smoke and the qualification: the `markitect/core` dependency and the canonical Library model M1. |
| `tools/` | Independent tools that import no product code: the packaged-source smoke, the Library and Makerspace oracles, the Makerspace reference modules, the comparison study's runner and assessment, and the model analysis. |
| `scripts/` | Maintainer scripts: documentation link check, CI test and flake reports, the native Library Apply operator script. |
| `packaging/` | What ships besides the binary: the runtime image, the Linux runner image, the consumer bootstrap, the modeling skill for agents and the distribution guide. |
| `examples/` | Copyable runtime-profile and run configuration examples; tests keep them valid. |
| `.github/` | Manual CI, release and flake-report workflows. |

**Why there is no `src/`.** Markitect is one Go module. Go keeps `cmd/` and `internal/` at the module root, and the source packager refuses a `src/` layout.

**Where new things go.**

| New item | Place | Also update |
|---|---|---|
| Go package | `internal/<responsibility>/` | One row in the architecture rule table and a package doc comment; regenerate the code map |
| Shipped command | `cmd/<name>/` | One row in the architecture rule table; the smoke's build list |
| Fixture used by one package | `<package>/testdata/` | Nothing else |
| Fixture shared across packages or used by qualification | `testdata/` | Keep bytes stable; locks pin them |
| Outside check that must not import product code | `tools/` | CI step |
| Maintainer script | `scripts/` | Tests next to it |
| Document | The owning folder under `docs/`, see [documentation maintenance](documentation.md) | Local map |
| Historical material | Not on Main: it stays at an archive point and [`docs/archive.md`](../archive.md) points to it | Index entry |

**What enforces this.** The architecture gate fails on a Go package without a row in its rule table; the source packager fails on unknown roots inside `cmd/` and `internal/` and on nested modules; `scripts/check-docs.py` fails on broken local links.
