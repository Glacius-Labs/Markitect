# Modules and static composition

The repository uses one Go module. Packages are organized by concrete responsibility and consumer-owned ports. There is no active legacy product layer or second Model check. [Architecture](../architecture.md) explains behavior; the generated [code map](code-map.md) inventories actual packages.

## Go capability packages

Pure model and snapshot packages consume supplied bytes and values. Organization and realization use those pure values. Project owns structural acquisition, complete accounting and static model-derived contexts. Run consumes Project and organization directly, retaining one complete hierarchy per mandate. `runtimeconfig` owns closed static native/check metadata. Workflow selects complete fresh Assessment, Implement and Verify Runs. Apply owns preflight and stored-candidate publication after closure; Bootstrap composes the concrete native and Git/check/store adapters. Native Apply qualification and owner acceptance remain separate.

Run owns Session, WorkspaceService, CandidateChecks and Records ports. The repository adapter implements real private Git copies and self-contained stored candidates. The check adapter binds declared checks to prepared pinned container profiles and disposable input/home copies. The file record adapter provides immutable terminal results and concurrent project Notes. The native harness and provider adapters remain governed by their separately reviewed source and bounded feasibility evidence.

## Dependency direction

| Consumer | Allowed responsibility |
| --- | --- |
| Model, snapshot | No other Markitect responsibility |
| Organization, realization | Model, snapshot |
| Project | Model, snapshot, organization, realization |
| Run | Project, organization, snapshot |
| Static runtime configuration | Project, model, organization, snapshot |
| Model view | Project, model, snapshot |
| Repository adapter | Project, Run, model, snapshot |
| Check adapter | Project, static runtime configuration, Run, pure values, container API |
| Record adapter | Run |
| Provider adapters and native harness | Their exact inward consumers and physical transport, declared by the gate |
| Container adapter and agentd | Agent frame protocol |
| CLI | Project and static metadata reports, model view, pure report values, build metadata and licenses |
| Document adapter | Model view and reviewed guarded-write API |
| Bootstrap | Its actual services and adapters |
| Product command | Bootstrap |
| Maintenance command | Its specific maintenance CLI |

The real-Git vertical test alone receives a test-only record-adapter edge. No sibling adapter or common tooling parent grants a general import exemption. Maintenance remains explicit: module checks consume structural Project preparation and pure hook/pipeline checkers; release uses source/license values; installation/publication use their own maintenance consumers.

## Apply target dependencies

This heading retains the design's historical link. The current gate covers actual root-layout responsibilities rather than a transitional target beside `src/`. Future packages require an explicit responsibility decision and map entry when implemented; no stub or success gate is required for an absent package. Consumers cannot import an absent producer.

## Responsibility map

Every actual Go package has one exact `codemap.yaml` entry with a compatible responsibility label, purpose and owning document. Generated [code-map.md](code-map.md) is byte-checked. The map is an actual inventory, not proof that an unimplemented design package exists.

## Adding, changing or removing a capability package

Coordinate shared contracts first. Add a concrete responsibility/import decision in [the gate](../../internal/tooling/architecture/target.go), an exact code-map entry and its artifact registrations. Source packaging has its own explicit responsibility roots and data selection; update its positive/negative tests for a new package or embedded asset. Keep one root module and reject nested product modules. Delete obsolete source, commands and gates together.

## Mechanical dependency gate

`go run ./cmd/markitect-check-architecture --repo .` inspects all production, test and platform Go files. It checks unknown/removed packages, exact responsibility edges, absent producers, map completeness, owner-document links, external-test self imports and test-only testkit use. The document adapter may use only the reviewed guarded-write selectors. Third-party imports remain permitted; this is a Markitect dependency check, not a general proof of I/O purity.

`internal/testkit` imports no Markitect package. Only test files may import it. The standalone consumer bootstrap (`packaging/bootstrap`) and source smoke import no Markitect packages. Go fixture directories and independent evaluation assets are not product package ownership exemptions.

Regenerate the view with:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

## Historical module links

Earlier capability/module and old product-layer contracts remain in the [historical guide](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/history/development-modules-legacy-sections-20261010.md) and [preceding source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).
