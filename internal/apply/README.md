# Apply preparation, gates and composition

Apply owns repository/profile preparation, the deterministic gates of a run and the
composition of its Passes. It performs no repository I/O itself: adapters supply the
captured inputs and the ports.

## Preparation

`PreparationService.CaptureSettings` captures the committed repository, runs the Model
check, decodes the registry shards and `.markitect/execution.yaml`, accounts every file,
applies the product defaults and resolves the Execute, Verify and helper profiles of every
Manager. `Passed` is static preparation only: it starts no session, runs no check and
claims no runtime readiness. File accounting is strict: every finding fails the
preparation, and no start drift is left for a run to repair
([DEC-064](../../docs/concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams)).
`Capture` (used by `markitect check`) and previews have the same stages without a
publication guard.

## Execution settings

`.markitect/execution.yaml` (`apiVersion: markitect.execution/v1`, strategy
`recursive-passes-v1`) is a closed document; unknown, duplicate or
representation-limited fields are refused with their exact source coordinates.

```yaml
budget:
  executionTime: 60m   # whole run wall clock
  turnTimeout: 8m      # one Agent turn
  agentTurns: 220      # every turn of Managers, reviewers and helpers
  passes: 10           # Implement, Verify and every repetition
profiles: {execute: library-agent, verify: library-agent}
instructions: {execute: [], verify: []}   # optional project instruction files
packages: {}           # exact per-Package overrides of profiles and instructions
ownership:             # who may write what: exact files, directories ending in /, or ./
  - {path: ./, owner: .}
  - {path: realization/lending/, owner: lending}
  - {path: realization/cmd/library/, owner: library, reason: The callable interface fixes the location.}
  - {path: realization/go.mod, owner: library}   # an exact file inside Root's area
team:                  # optional; replaces the product default team
  maxHelpers: 2
  helpers:
    - {role: worker, writes: true}                       # the Manager's Execute profile
    - {role: architect, profile: review-agent, writes: false, instructions: [docs/architect.md]}
scope: []              # optional modeled scope selectors [{kind, property}]
checks: []             # required checks (id, profile, directory, args, requiredBy, timeout)
```

The product defaults (`strategy_defaults.yaml`, embedded) supply the empty instruction
lists an authored file omits and the standard team: one `worker` role with writes,
bound to each Manager's Execute profile, at most one active helper. Every value keeps
its `Setting` provenance. After a successful preparation, `Settings()` holds the resolved
per-Manager roles and helpers, `Budget()` the authored budget, `Checks()` the required
checks and `FileCheck().Layout()` the validated ownership.

## Ownership

`realization.Layout` holds the declared areas; the most specific area owns a path (an
exact file above a directory, a deeper directory above a shallower one). Two areas on
one path, a path declared both as a file and as a directory, anything under `.markitect`
or `.git`, an owner that is no local Package and letter-case collisions fail the
preparation. A file no area covers fails File check (`layout.no_owner`). Ownership is
required for Apply; `markitect check` without it accounts by registrant only.

## Candidates

`NewCandidates` binds one passed preparation and supplies the gates and facts of a run:
`Account` is the File check of a stand (unexplained or duplicate files, invalid or
unknown registry shards, entries outside their registrant's area, directory entries that
register nothing, files without an owner and changed authority inputs, a changed or
deleted one quoting its original up to 8 KiB of UTF-8 text), `ModelCheck` re-runs the
Model check on the stand's model inputs and requires the bound model, `RegistryView`
assembles the derived read-only registry view, `OwnerOf` returns the owner of any path
("" for none), `Areas` and `Shard` describe a Manager's areas and local registry, and
`WritePlan` returns the enforced `run.WritePlan` of a Manager: directory and exact-file
overlays of its own areas and registry writable, `.markitect`, protected inputs and other
owners' areas read-only. An exact-file area that needs its own overlay but does not exist
in the start stand is refused with `ErrUnsupportedOwnership`.

## Apply

`Apply(ctx, ApplyRequest)` binds the organization (Briefs, own obligations, write plans,
helpers), the required checks, the gates, the owner channel and the budget, requires the
container engine's volume-subpath support, and composes the Passes on `run`: Implement,
in which every Manager plans and prepares its own part, then Verify; a Verify that needs
correction starts a new Implement and a fresh complete Verify follows; a Verify that only
fresh disputes or disproofs awaiting an independent assessment keep from being clean is
repeated on the unchanged candidate. Only a
Verified outcome is published, through the guarded `Publisher` with read-back; the shared
work volume is closed only after an applied or unchanged receipt and kept as
`RetainedWork` otherwise. Apply never commits. Each required check also passes the number
of its execution profile's own arguments to `run`, so a probe can reuse that profile with
its own arguments.

The mandates ask Implement for each Manager's own approach first (distinct claims, input
classes, interfaces, needs and outputs), a basis that holds the approach and the
foundation its children build on, and a wait report for every other Manager's basis or
result its work needs, which records that dependency; and Verify
to derive claims and permitted counterexamples from the bound model first, probe them at
the public interface and cite receipts or inspected sources for their claims, with parents
challenging their children's assertions.

`VerifyCaptured(ctx, ApplyRequest)` runs only the independent Verify of the captured stand
with the same configuration and never publishes. It is the evaluation entry for assessing
a known stand through the Pass process; no Command uses it, and the grant-gated
`TestVerifyEvaluationHarness` in `internal/bootstrap` drives it with the real adapters.

Each Brief lists the Manager's own Definitions, included ones with their owners, its
areas and local registry with its entry shape (exact files or directories), the required
checks and its configured instructions. Root also receives the project obligations: every
local Definition with its owner and the Definitions the required checks name. A parent's
Brief names each child's areas and Definitions, the Managers below it and the role
statement: children realize their own areas, the parent supplies its own foundation,
coordinates and integrates but cannot take over or waive a child's obligations.

## Extract

`PreparationService.CaptureExtract` prepares the start of an Extract (artifacts, no
model) like `CaptureSettings`, except that unregistered artifacts (`file.unexplained`) and
artifacts that no area covers yet (`layout.no_owner`) do not fail file checking.
`Extract(ctx, ExtractRequest)` then runs two phases on the shared composition: a
structure phase with Root alone, which proposes Packages with one overview Definition each,
the ownership areas they need and the checks the artifacts document (a proposal for the
owner's adoption), and a model phase with the Manager tree of the verified proposal, which
derives each Package's model and registers the artifacts. Each phase is a work Pass of its
own purpose (`structure`, `model`) followed by Verify. It refuses a start whose local
model has Definitions or that declares no ownership areas, never changes an artifact, and
publishes a verified model only as the changed `.markitect` model, registry and
`execution.yaml`. `extractAccounting` runs the Model check on each candidate tree from
scratch (pure, safe for concurrent use) and gives each phase its write plan. The model
phase's Verify sets `ArtifactFindingsAdvisory` with a `ResolveDefinition` that runs the
Model check on the candidate, so an artifact that deviates from a justified statement is
a non-blocking finding with subject `artifact` and its Definition. `ExtractDeviations`
groups the open such findings into one `ArtifactDeviation` per Definition. The
[runtime contract](../../docs/design/canonical-model/manager-apply-runtime.md#extract-first-bounded-implementation)
states the plans, accounting, budget and limits.

## Publication

`Publisher` preflights and publishes a verified tree against the captured source guard
and reads the result back. A stale source or a change set the target cannot represent is
a definite receipt without any write; a partial write stays visible and unsuccessful.
