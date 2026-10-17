# Modules and static composition

The repository uses one Go module. Packages are organized by concrete responsibility and consumer-owned ports. There is no active legacy product layer or second compiler. [Architecture](../architecture.md) explains behavior; the generated [code map](code-map.md) inventories actual packages.

## Go capability packages

Pure model and snapshot packages consume supplied bytes and values. Organization and realization use those pure values. Apply owns preparation and consumer ports; adapters implement concrete acquisition and metadata protocols. Bootstrap constructs actual services. Product and maintenance commands remain thin entrypoints with separate composition.

Run owns fixed input bindings, deterministic budget and lifecycle transitions, and its narrow journal port. A proposal changes no confirmed state; an exact append acknowledgement commits the transition. The Host supplies clock observations. There is no SQL, repository acquisition, provider protocol or public execution path in this package.

## Dependency direction

| Consumer | Allowed responsibility |
| --- | --- |
| Model, snapshot | No other Markitect responsibility |
| Organization, realization | Model, snapshot |
| Run | Model, snapshot, organization, realization |
| Apply | Model, snapshot, organization, realization, run |
| Model view | Apply, model, snapshot |
| Repository adapter | Apply and required pure values |
| Codex/Check metadata adapters | Apply, model, organization, snapshot |
| CLI adapter | Apply, pure report values, model view, build metadata and licenses |
| Document adapter | Model view and reviewed guarded-write API |
| Product bootstrap | Its actual services and adapters |
| Product command | Product bootstrap |
| Maintenance command | Its specific maintenance CLI |

Maintenance services have exact edges: release uses source/license values; publication uses release/source; installation uses release/source/guarded writes. Module checks consume actual v1 preparation and registry plus pure hook/pipeline checkers. Source, module checkers and architecture tooling have no product-execution privilege. A common parent such as `internal/tooling/` or `internal/adapters/` does not permit arbitrary sibling imports.

## Apply target dependencies

This heading retains the design's historical link. The current gate covers actual root-layout responsibilities rather than a transitional target beside `src/`. Future packages require an explicit responsibility decision and map entry when implemented; no stub or success gate is required for an absent package. Consumers cannot import an absent producer.

## Responsibility map

Every actual Go package has one exact `codemap.yaml` entry with a compatible responsibility label, purpose and owning document. Generated [code-map.md](code-map.md) is byte-checked. The map is an actual inventory, not proof that an unimplemented design package exists.

## Adding, changing or removing a capability package

Coordinate shared contracts first. Add a concrete responsibility/import decision in [the gate](../../internal/tooling/architecture/target.go), an exact code-map entry and its artifact registrations. Source packaging has its own explicit responsibility roots and data selection; update its positive/negative tests for a new package or embedded asset. Keep one root module and reject nested product modules. Delete obsolete source, commands and gates together.

## Mechanical dependency gate

`go run ./cmd/markitect-check-architecture --repo .` inspects all production, test and platform Go files. It checks unknown/removed packages, exact responsibility edges, absent producers, map completeness, owner-document links, external-test self imports and test-only testkit use. The document adapter may use only the reviewed guarded-write selectors. Third-party imports remain permitted; this is a Markitect dependency check, not a general proof of I/O purity.

`internal/testkit` imports no Markitect package. Only test files may import it. The standalone integration bootstrap and source smoke import no Markitect packages. Go fixture directories and independent evaluation assets are not product package ownership exemptions.

Regenerate the view with:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

## Historical module links

Earlier capability/module and old product-layer contracts remain in the [historical guide](../history/development-modules-legacy-sections-20261010.md) and [preceding source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).
