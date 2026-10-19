# Apply preparation and candidate accounting

Apply owns repository/profile preparation and the consumer ports. It performs no
repository I/O itself: adapters supply the captured inputs.

## Preparation

`PreparationService.CaptureSettings` captures the committed repository, compiles
the model, decodes the registry shards and `.markitect/execution.yaml`, accounts
every file, applies the product defaults and resolves the Execute and Verify
profiles of every Manager. `Passed` is static preparation only: it starts no
session, runs no check and claims no runtime readiness. Accounting must pass
before any Agent starts. Previews have the same stages without a publication
guard.

## Execution settings

`.markitect/execution.yaml` (`apiVersion: markitect.execution/v1`, strategy
`shared-workspace-v1`) is a closed document; unknown, duplicate or
representation-limited fields are refused with their exact source coordinates.

```yaml
budget:
  executionTime: 25m   # whole Apply wall clock
  turnTimeout: 10m     # one Agent turn
  agentTurns: 120      # Execute plus Verify turns
  rounds: 4            # Verify rounds including the initial Verify
profiles: {execute: library-agent, verify: library-agent}
instructions: {execute: [], verify: []}   # optional project instruction files
packages: {}           # exact per-Package overrides of profiles and instructions
layout:                # optional persistent ownership of artifact paths
  roots: [realization]
  exceptions:
    - {path: realization/cmd/library, owner: library, reason: The callable interface fixes the location.}
scope: []              # optional modeled scope selectors [{kind, property}] that add Definitions to a Manager's context
checks: []             # required checks (id, profile, directory, args, requiredBy, timeout)
```

The product defaults (`strategy_defaults.yaml`, embedded) supply the empty
instruction lists an authored file omits. Every value keeps its `Setting`
provenance. After a successful preparation, `Settings()` holds the resolved
per-Manager Execute and Verify profile and instructions, `Budget()` the authored
budget, `Checks()` the required checks and `FileCheck().Layout()` the validated
layout.

## Layout and ownership

Without a layout, the registry shards alone decide who owns a file. With one,
`realization.Layout` assigns every path to a Manager (mirrored roots, explicit
exceptions, deepest wins, everything else Root) and accounting requires every
registration to sit in its owner's shard (`registry.outside_area`). An invalid
layout (unknown owner, nested roots, anything under `.markitect`, spellings that
differ only by letter case) fails preparation.

## Candidates

`NewCandidates` binds one passed preparation. `Candidates.Account(tree)` returns
the blocking structural findings of a candidate tree, each addressed to the
Manager that must fix it (without a declared layout an unexplained new file goes
to Root, which asks its children who created it; the creator registers it in its
own shard): unexplained or duplicate files, invalid or unknown
registry shards, registrations outside the registering Manager's area, and any
canonical or control input that changed, vanished or appeared (a changed or
deleted one quotes its original up to 8 KiB of UTF-8 text, else its SHA-256 and
size). Registry shards
are the exception: Managers maintain them. `OwnerOf`, `Shard` and `Areas` give
the engine and the prompts the ownership facts. An accounted candidate returns no
findings. Trust, not enforcement: nothing here attributes a write to an Agent.

## Publication

`Publisher` preflights and publishes a verified tree against the captured source
guard and reads the result back. Apply never commits. `Reconcile` closes the
shared work volume only after an applied or unchanged receipt; otherwise it keeps
the verified work and reports it in `RetainedWork`.

Each Manager's Brief lists its own Definitions; Root has none, so it receives the
Definitions that the required checks name in `requiredBy`.
