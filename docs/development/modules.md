# Modules and static composition

The repository uses one Go module. Packages are organized by concrete responsibility and consumer-owned ports. There is no active legacy product layer or second compiler. [Architecture](../architecture.md) explains behavior; the generated [code map](code-map.md) inventories actual packages.

## Go capability packages

Pure model and snapshot packages consume supplied bytes and values. Organization and realization use those pure values. Apply owns preparation and consumer ports; adapters implement concrete acquisition and metadata protocols. Bootstrap constructs actual services. Product and maintenance commands remain thin entrypoints with separate composition.

The container runtime runs every Agent session in its own Linux container ([DEC-037](../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)). `markitect-agentd`, built from `internal/agentd` by `cmd/markitect-agentd`, runs inside the container: it starts the provider CLI, relays its standard streams, serves the Orchestrator tools over MCP on a Unix socket and reports exits. The container adapter starts hardened session containers and owns the frame transport of `internal/agentd/frame` with agentd. The concrete Codex and Claude session adapters may consume the frame DTOs exposed by that container API while implementing `run.Sessions`. Each provider adapter keeps its own protocol handling and cannot import the other provider or agentd execution. The base image is defined in [packaging/runtime](../../packaging/runtime/README.md).

Run owns fixed input bindings, deterministic budget and lifecycle transitions, and its narrow journal port. A proposal changes no confirmed state; an exact append acknowledgement commits the transition. The Orchestrator supplies clock observations. There is no SQL, repository acquisition, provider protocol or public execution path in this package.

Existing history is reconstructed through bounded, revision-pinned reads and the same transition rules. Required blobs and complete batches are verified before a projection is returned. Reading creates no effect observations; open invocations retain their holds until explicit reconciliation.

The [SQLite journal adapter](../../internal/adapters/journal/README.md) implements Apply control and Run recording. Production imports only Apply and Run; its owned integration fixtures may import model and snapshot values from test files. Repository-local lifetime control remains independent of the external store directory. SQL commits validate the live epoch and required byte references; reopening never implies that previous processes stopped.

The [Codex metadata preparer](../../internal/adapters/codexappserver/README.md) uses a narrow image reader backed by the container adapter. Bootstrap injects its factory; only a selected source-bound profile supplies the engine executable and immutable image ID. Preparation inspects the image and copies its build metadata from a created, never started container. It starts no Session or provider and observes no schema inventory.

## Dependency direction

| Consumer | Allowed responsibility |
| --- | --- |
| Model, snapshot | No other Markitect responsibility |
| Organization, realization | Model, snapshot |
| Run | Model, snapshot, organization, realization |
| Apply | Model, snapshot, organization, realization, run |
| Model view | Apply, model, snapshot |
| Repository, workspace and publication adapter | Apply, Run, required pure values and container API |
| Journal adapter | Apply and Run; model and snapshot only in tests |
| Codex metadata and session adapter | Apply, model, organization, snapshot, Run, container API and its agent frame DTOs |
| Claude session adapter | Run, container API and its agent frame DTOs |
| Check metadata and execution adapter | Apply, model, organization, snapshot, Run and container API |
| Container adapter | Agent frame protocol |
| Session helper (agentd) | Agent frame protocol |
| Agent frame protocol | No other Markitect responsibility |
| CLI adapter | Apply, pure report values, model view, build metadata and licenses |
| Document adapter | Model view and reviewed guarded-write API |
| Product bootstrap | Its actual services and adapters |
| Product command | Product bootstrap |
| Runtime command | Session helper |
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
