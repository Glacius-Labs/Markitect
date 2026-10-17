# Architecture

Markitect is canonical model-first development with delegated realization. The [vision](vision.md#markitect-in-brief) owns that purpose; the [delivery plan](implementation-plan.md) owns the staged rebuild. Current source implements complete v1 compilation and read-only preparation, plus a separately guarded generated view. Execution and Apply remain unavailable.

## Product flow

`cmd/markitect → internal/bootstrap → CLI adapter → PreparationService → consumer-owned repository/profile ports`.

The repository adapter captures actual sources. ModelService compiles only complete acquired inputs. PreparationService composes actual model, registry, complete file accounting, execution declarations and static context selection. Optional metadata/settings preparation uses explicit Host sources and actual profile adapters. The CLI presents those reports without inventing run records or execution readiness.

Generated views use `internal/modeldocs` and its document-writer port, implemented by `internal/adapters/documents`. Their guard is independent of Apply publication. `apply` returns unavailable before acquisition or effects.

## Authority and evidence

The canonical model states intended behavior; exact registry entries state artifact ownership and Definition basis. Execution declarations, product defaults, Host observations, assignments and future runtime records have separate responsibilities. A committed model binds its specification revision, not an authenticated human decision. Source checks, hosted CI, provider execution, releases and owner acceptance remain separate evidence.

## Layers

All maintained Go packages use one root module. [The code map](development/code-map.md) inventories every actual package. [Import rules](development/modules.md#mechanical-dependency-gate) check explicit responsibilities in production, test and platform files.

| Responsibility | Packages | Direction |
| --- | --- | --- |
| Canonical language values/compiler | `internal/model` | No other Markitect responsibility |
| Fixed path/file/tree values | `internal/snapshot` | No other Markitect responsibility |
| Topology, exact settings and context | `internal/organization` | Model and snapshot |
| Ownership, accounting and candidates | `internal/realization` | Model and snapshot |
| Read-only preparation and consumer ports | `internal/apply` | Pure responsibilities |
| Generated model view/write plan | `internal/modeldocs` | Apply, model and snapshot |
| Physical/protocol adapters | Explicit `internal/adapters/*` packages | Their actual inward consumers; no general sibling privilege |
| Composition | `internal/bootstrap` | Actual product services and adapters |
| Product entrypoint | `cmd/markitect` | Bootstrap only |
| Maintenance | Explicit `internal/tooling/*` packages and commands | Exact maintenance edges, separate from product execution |

## Core layer

The old Core/Project DTO compiler has been removed. Current `model`, `snapshot`, `organization` and `realization` packages are pure values and deterministic operations over explicit supplied facts. They have no repository, filesystem, Git or provider responsibility. Canonical model digests, snapshot content/tree identities and acquisition guards remain different values.

There is one ordinary root and one Manager per local canonical Package, including authoritative intermediate and empty Packages. Manager topology is derived; it is not a canonical Manager Kind or a copy of the old DTO.

## Infrastructure layer

`gitrepo` implements Source, PreviewSource and explicit revision resolution. Committed capture requires clean HEAD and returns its comparison guard. Working/fixed previews never return that guard. It checks exact spellings, modes, physical directory identity, manifest roots and the complete census. Native aliases of the already confirmed selected root retain their repository coordinates; redirects and unbound external historical sources remain refused.

Codex and Check adapters parse explicitly selected source-bound metadata. They do not start a process, probe versions, export schemas or execute checks. The document adapter observes and writes one reviewed generated target through the independent guarded-write API.

## Application layer

`internal/apply` owns the acquisition and metadata-preparation ports. It retains original requests, every actual capture attempt, known failures and incomplete observations. New closure inputs require a fresh complete acquisition, never an extended old guard. Static context preserves actual Definitions, resolved values and inclusion reasons. Settings resolution preserves authored absence and the original embedded defaults source.

Bootstrap alone constructs concrete adapters. CLI code owns parsing/presentation and uses the repository port to resolve explicit refs once. It does not invoke Git directly or reconstruct producer semantics.

## Runtime layer

`internal/run` supplies fixed run bindings, assignment budget admission and confirmed journal transitions. Its narrow journal port separates proposals from acknowledged history and retains uncertain reservations. It has no physical journal adapter or public execution entrypoint.

Recursive Manager execution, sessions, workspace creation, executable checks, durable journal storage, verification and guarded artifact publication remain later implementation work. There is no active legacy runtime or fallback execution engine. A structural or metadata pass cannot authorize those effects. The current CLI exposes no MCP server.

## Operational record storage

Read-only preparation creates no run/task/budget/candidate or journal. Document writes retain observed completed paths if a later validation fails. Future durable execution/recovery records belong to actual consumer ports and their implementations; a preview cannot fabricate them.

## Project artifact boundary

Every regular file in the actual census is a protected canonical/control input, an exact registered artifact or an exact explained ignore. Registry shards derive their owner from their physical Package coordinate. Each artifact links existing Definitions and has one owner; a file may realize several Definitions. Gitignore is not accounting, and ignored files cannot satisfy required realization.

## Fixed inputs and evidence

Fixed previews inspect exactly the resolved full commit and refuse current external inputs without historical binding, even when the ref is HEAD. Working previews observe the actual draft. Neither is committed Apply capture. Captured source bytes/modes/provenance, canonical model digest and complete census identity remain visible in reports.

The generated view distinguishes canonical model, rendered content and source/target-bound preview digests. Writes recheck source and target, reject protected inputs and historical writes, and read back actual bytes under an independent guard. Multi-file or interrupted effects are not assumed transactional.

## Distribution and product boundary

Source packaging, release publication, distribution and guarded installation remain independent maintenance services. Current source archives use root `cmd/` and `internal/`; legacy/mixed layouts are refused. Embedded licenses, strategy defaults and selected test data are preserved. Source-smoke checks do not publish a release or demonstrate provider/Apply acceptance. See [distribution](../integration/README.md) and [operations](operations.md#release-operations).

## Historical Project/Domain architecture

Published releases retain their immutable contracts. The removed product generation is reconstructible from [5ff32069](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).

## Historical architecture links

These retained anchors route existing links to the [history record](history/architecture-legacy-sections-20261010.md) of the earlier line.

<a id="historical-projectdomain-and-projection-compatibility-architecture"></a>
The [Project/Domain compatibility architecture](history/architecture-legacy-sections-20261010.md#historical-projectdomain-and-projection-compatibility-architecture) is historical.

<a id="compiler-and-evidence-model"></a>
The [compiler and evidence model](history/architecture-legacy-sections-20261010.md#compiler-and-evidence-model) of the Domain kernel is historical.

<a id="current-implementation-boundary"></a>
The [v0.11.0 implementation boundary](history/architecture-legacy-sections-20261010.md#current-implementation-boundary) is historical.

<a id="v0120-bounded-resolved-target-equality"></a>
The [v0.12.0 resolved-target equality](history/architecture-legacy-sections-20261010.md#v0120-bounded-resolved-target-equality) is historical.

<a id="policy-failure-analysis-in-current-source"></a>
<a id="policy-failure-analysis-in-v0130"></a>
The [v0.13.0 policy failure analysis](history/architecture-legacy-sections-20261010.md#policy-failure-analysis-in-v0130) is historical.

<a id="selective-adoption-preparation-in-current-source"></a>
<a id="selective-adoption-preparation-in-v0130"></a>
The [v0.13.0 selective adoption preparation](history/architecture-legacy-sections-20261010.md#selective-adoption-preparation-in-v0130) is historical.

<a id="projection-first-source-direction-in-progress"></a>
<a id="projection-first-reconciliation-experimental-preview"></a>
The [projection-first reconciliation preview](history/architecture-legacy-sections-20261010.md#projection-first-reconciliation-experimental-preview) is historical.

<a id="resource-model"></a>
The [v0.13 resource model](history/architecture-legacy-sections-20261010.md#resource-model) is historical.

<a id="deterministic-core-and-adapters"></a>
The [deterministic core and adapters](history/architecture-legacy-sections-20261010.md#deterministic-core-and-adapters) of the Domain kernel are historical.

<a id="go-implementation-boundaries"></a>
<a id="go-ownership-and-final-dependency-model"></a>
The [earlier Go ownership table](history/architecture-legacy-sections-20261010.md#go-ownership-and-final-dependency-model) is historical; current ownership is in [Layers](#layers) and the [code map](development/code-map.md).

<a id="authoring-and-queries"></a>
The [v0.13 authoring and queries](history/architecture-legacy-sections-20261010.md#authoring-and-queries) are historical.

<a id="markitect-first-and-artifact-coverage-release-candidate"></a>
<a id="markitect-first-and-artifact-coverage-published-v0130"></a>
The [v0.13.0 Markitect-first workflow and artifact coverage](history/architecture-legacy-sections-20261010.md#markitect-first-and-artifact-coverage-published-v0130) are historical.

<a id="canonical-reset-source-only-vnext-boundary"></a>
<a id="canonical-reset-alpha-removed"></a>
The [canonical reset alpha](history/architecture-legacy-sections-20261010.md#canonical-reset-alpha-removed) was removed from source.
