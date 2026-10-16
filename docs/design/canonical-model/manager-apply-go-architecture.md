# Go architecture for reliable Apply

Date: 16 October 2026. Initial outline for discussion, requested after [library M1](library-example/README.md). This proposes responsibilities and dependency direction for P01 of the [implementation plan](manager-apply-implementation-plan.md); it does not complete P01's concrete contracts or change current source architecture. The [Apply concept](manager-apply-concept.md) owns product behavior and the clean redesign policy.

## Organize by responsibility

Use Ports and Adapters with ordinary Go packages. Keep deterministic model and realization rules independent of external systems. Coordinate the workflow through small capabilities implemented by concrete adapters. Packages describe what they do; an interface is not required for every internal function or struct.

| Package | Responsibility |
| --- | --- |
| `model` | Parse explicitly supplied bytes, compile complete v1, resolve contracts and links, and calculate canonical content and diagnostics. No repository reads, provider calls or prose interpretation. |
| `organization` | Derive the ordinary root and local Package Managers, resolve project defaults and Manager overrides, and render inspectable context from supplied model/task facts. No Agent startup or workflow execution. |
| `realization` | Registry, artifact ownership, path-claim rules, candidate differences, Definition assessments and file accounting over supplied values. A registered file is not proof of correct behavior. |
| `run` | Bounded tasks, delegation, messages, role sessions, review rounds, upward integration, evidence bindings and run history. It enforces the workflow and validates Agent requests; Managers choose their delegated outcomes and task prompts. |
| `apply` | The artifact reconciliation use case: bind inputs, prepare the organization, invoke `run`, require aggregate verification, then request guarded publication and read-back. The model remains read-only. |
| `snapshot` | Small fixed file-tree values and comparisons shared by the above responsibilities and adapters. No Git commands or storage policy. |

`run` is the common execution mechanism, initially used only by Apply. This leaves room for later model operations and custom commands without putting another Manager engine in each command. It does not require a command framework or implement those extensions now.

A Manager is a logical identity with tasks, configuration and session references. It is not a Go package generated for each authored Package, nor does it require a permanently running goroutine. Initially one writer is active. A parent can yield with recorded child tasks pending; the Host runs those tasks and resumes the parent with their results. Concrete session handles and attempts remain distinct from logical Manager identities.

## Suggested source layout

One Go module can use the following layout. Names are provisional; a directory does not need another nested package for every concern listed in the table.

```text
cmd/markitect/
internal/
  model/
  organization/
  realization/
  snapshot/
  run/
  apply/
  adapters/
    cli/
    mcp/
    gitrepo/
    filestore/
    agent/
    checks/
  bootstrap/
```

`bootstrap` constructs concrete adapters and passes them to their consumers through constructors. The CLI and MCP adapters call the same application operations. Their protocol handling and error presentation do not own review, retry or publication policy. `cmd/markitect` delegates startup to that composition root.

Inside `run`, start with coherent files such as `task.go`, `coordinator.go`, `review.go`, `history.go` and `ports.go`. State transitions and record meaning belong there; physical files and provider protocols belong in adapters. Split further only when a distinct responsibility or consumer makes the boundary useful. Avoid another shared Project DTO, global interface catalogue or untyped property bag spanning all packages.

The proposed top-level `cmd` and `internal` directories remove the current `src` prefix. This shortens paths, but requires deliberate updates to imports, build/release tooling, canonical source ownership and architecture checks. It is a layout choice, not a Go language requirement. The responsibility boundaries matter before the move.

## Dependency direction and ports

At the product level, `apply` uses `run` and the pure packages. `run` uses organization, model, realization and snapshot values, without importing `apply` or concrete adapters. Organization and realization may consume model/snapshot values; they do not import each other or coordinate execution. The model compiler does not depend on the organization or registry. The pure snapshot package imports no higher responsibility. Concrete adapters depend inward; only bootstrap assembles them.

The main external capabilities are:

| Port capability | Consumer | Concrete work |
| --- | --- | --- |
| Source acquisition | `apply` | Capture model bytes and explicitly supplied dependencies, configuration, registry, repository census and the source binding for the run. |
| Agent sessions | `run` | Start a configured role, supply tasks/context, observe messages and tool requests, resume and cancel, and establish known or uncertain termination. |
| Candidate workspaces | `run` | Materialize an owned candidate, observe actual changes, and close or retain it for investigation. |
| Run journal | `run` and the Apply lifecycle | Durably append bound events and store exact payloads under one run identity and writer discipline. |
| Check execution | `run` and final verification | Execute explicit project checks on the exact candidate and return observed results. |
| Guarded publication | `apply` | Recheck the captured source, publish the verified artifact/registry candidate together, and report actual writes and read-back. |

These are responsibilities for small consumer-owned interfaces, not final signatures or a large universal Repository interface. Exact request/result types, journal sequencing and capability lifetimes are the next P01 design step. The same concrete adapter may satisfy several narrow interfaces. Ordinary calls between pure packages use concrete functions and values.

The Host owns the task/message protocol, effective authority and history. An Agent adapter translates provider transport into those records; it cannot accept a child delivery, grant a path or mark a review passed by itself. Manager, reviewer and verifier use the same session boundary with distinct roles and fresh review inputs. Provider wire objects and provider session IDs do not become canonical model types.

Journal observations cover preparation, dispatch, additional input, responses, candidate changes, checks and publication. One run has a coordinated writer and explicit transitions; a generic event bus or full event-sourcing framework is unnecessary. Recording required work must succeed before further dispatch or publication. Prepared, sent, observed and uncertain states stay distinguishable; cancellation does not prove a child process terminated.

The registry belongs to realization, outside canonical Definition roots. The publication capability includes its accepted revision and the matching artifact tree, rather than saving the registry separately and hoping they agree. Guarded multi-file publication can still fail partially; the adapter reports completed writes and uncertainty, and Apply records recovery state instead of promising a transaction.

These choices follow the Go guidance to place interfaces with their consumers and return concrete implementations ([Go code review comments](https://go.dev/wiki/CodeReviewComments#interfaces)). Constructors wire dependencies explicitly. Use `context.Context` at effectful calls for cancellation/deadlines and ordinary errors for failures; pure computations need no interface merely to be testable. Avoid generic `common`, `types` and `interfaces` packages ([package naming](https://go.dev/blog/package-names)). The `cmd`/`internal` layout follows a common documented convention ([Go module layout](https://go.dev/doc/modules/layout)), without making it the only valid layout.

## Treatment of current source

Product source inspected at Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`; the design branch at `2b82ba5a7b7b3f28073857c7e7a8a36e06619c87` has the same product source. The current implementation already has ports, explicit composition and protected I/O boundaries. The redesign changes their ownership and old-model coupling, rather than assuming none exist. These are code-reading findings and reuse candidates, not new execution evidence.

| Current area | Treatment |
| --- | --- |
| [Compiler identities and envelopes](../../../src/internal/core/types.go), fixed [project Schema](../../../src/internal/core/projectmodel/projectmodel.go) and its old project views | Replace with complete v1 and Package-derived organization plus the separate registry. Remove fixed Manager/Statement/Artifact authoring contracts and their obsolete consumers. Reuse individual algorithms only after checking them against v1; preserve no old DTO shape for compatibility. |
| [Snapshot values](../../../src/internal/core/snapshot/snapshot.go) and [source acquisition](../../../src/internal/application/projectwork/project.go) | Reuse useful deterministic comparisons and hardened acquisition mechanics. Replace the old project loader, manifest assumptions and `projectvalue.Project` result with explicit new inputs. Acquisition already has a shared source session; do not recreate separate unbound reads. |
| [Guarded writing](../../../src/internal/infrastructure/guardedwrite/guardedwrite.go) | Retain or directly reuse source/identity checks and honest partial-write behavior behind the publication adapter. Add the accepted registry binding and final read-back required by the concept. Tests must establish these properties in the new path. |
| [Record storage](../../../src/internal/infrastructure/recordstore/store.go) and [runtime store](../../../src/internal/runtime/projectrun/store.go) | Reuse confined file access, publication and lease mechanics where suitable. Replace old run codecs, closed historical record layouts and domain lifecycle coupling when they do not fit. The journal must support the actual prompt/message/candidate history; keeping historical formats readable is not a target. Original evidence retains its identity. |
| [Runtime Host and Invoker](../../../src/internal/runtime/projectrun/types.go), coordinator, review, verify and Apply | Reuse method rules and meaningful counterexamples: bounded rework, fresh review, candidate binding, stale refusal and recovery. Restructure into `run`, `apply` and realization responsibilities. Remove the broad old Host callbacks and Plan/Report coupling rather than wrapping the whole old runtime. The current [Reconcile guidance](../../../src/internal/runtime/projectrun/operation.go) already expresses full responsibility against unchanged intent. |
| [Workspace boundary](../../../src/internal/runtime/projectworkspace/contracts.go) and Git workspace implementation | Reuse materialization, delta harvesting and lifecycle mechanics where they fit. Bind new ownership, path reservations and artifact-only scope to the new registry. A workspace is not an OS sandbox. |
| [Native Agent adapter](../../../src/internal/runtime/codexappserver/session.go) and process execution | Reuse bounded process/session handling and observation/recovery mechanisms selectively. Replace old request/response coupling with the new task and session contracts. Select one real executor first; preserve no adapter merely for compatibility. This outline does not choose a provider API contract. |
| [Application composition](../../../src/internal/application/projectapp/composition.go), CLI and MCP | Keep the shared thin-entrypoint principle, rebuild composition around the new packages and make both surfaces call the same operations. Remove obsolete command wiring together with the behavior and tests it specified; later extensions need a concrete consumer. |

There is no old/new compatibility frontend or permanent second Apply. Reusable code is material for the new architecture, not a constraint on it. Update the repository's own canonical model, generated maps, checks and packaging with each deliberate source replacement. Current contribution rules still govern until that reviewed replacement changes them; an allowlist must not conceal obsolete architecture assumptions.

## Walk the library through the boundaries

Apply acquires M1 and the current realization, compiles through `model`, derives Managers/context through `organization` and obtains registry/coverage facts through `realization`. It starts `run` with the artifact reconciliation instruction for the root only. The root delegates to the local Package Managers; lending can delegate reminders. Host-mediated tasks and results cross the session port and enter the journal. Actual workspace deltas and registry proposals are validated before review and integration.

Local review, integration review and final verification retain their distinct questions and exact candidate bindings. Only a complete verified aggregate reaches guarded publication. The publisher reports actual artifact and registry writes and read-back; the run summary refers to that evidence. M2 follows the same path from existing A1, with only the model's admission limit changed.

The Playground stays outside the product packages. It prepares scenarios, invokes the real product and independently inspects the outcome. It owns neither a Manager engine nor another review/publication loop. A passing fake-port workflow test, a real Agent session test and the two real Applies remain different evidence classes.

## Next design step

Agree on these responsibilities, then specify one root-to-lending task with the smallest useful port contracts: prepared inputs, session exchange, candidate/registry delivery, journal events and publication result. Assign each state and output one owner. Only then finish the concrete keep/rewrite/delete map and implementation slices. P01 remains incomplete until those contracts and the source-removal boundaries are reviewable; the Playground gets its own P02 design before substantial runtime implementation.

Return to the [Apply concept](manager-apply-concept.md), [implementation plan](manager-apply-implementation-plan.md) or [canonical-model overview](README.md).
