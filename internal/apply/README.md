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

`.markitect/execution.yaml` (`apiVersion: markitect.execution/v1`) is a closed document;
unknown, duplicate or representation-limited fields are refused with their exact source
coordinates. Its required `strategy` selects the work composition the Run binds: `direct`
or `staged` (see Apply below); any other value, including the replaced
`recursive-passes-v1`, is refused. A project migrating from `recursive-passes-v1` writes
`staged` for the previous composition or `direct` for the provisional baseline.

```yaml
strategy: direct       # direct or staged
budget:
  executionTime: 60m   # whole run wall clock
  turnTimeout: 8m      # one Agent turn
  agentTurns: 220      # every turn of Managers, reviewers and helpers
  passes: 10           # every work Pass, Verify and every repetition
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

The product defaults (`strategy_defaults.yaml`, embedded, `strategies: [direct, staged]`)
serve both strategies alike: the empty instruction lists an authored file omits and the
standard team, one `worker` role with writes, bound to each Manager's Execute profile, at
most one active helper. Every value keeps its `Setting` provenance. After a successful
preparation, `Settings()` holds the resolved per-Manager roles and helpers, `Budget()` the
authored budget, `Strategy()` the authored strategy, `Checks()` the required checks and
`FileCheck().Layout()` the validated ownership.

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

## Assignment realization caller boundary

`RealizeAssignments(ctx, AssignmentRealizationRequest)` is the callable DEC-067
use case. A passed captured settings preparation supplies the fixed model,
required Manager tree, ownership, profiles and check declarations. Explicit
caller values supply a unique invocation ID, build identity, absolute transcript
directory, one finite context, message timeout and total message allowance.
Assessment, Implement, questions, corrections and independent Verify consume that
same allowance. This API does not reinterpret the legacy Pass/Turn settings.

The use case assesses the model in a separate assignment before opening the
Implement Agent. A material model problem stops affected work; corrections to
intent happen outside execution. Implement realizes the required organization,
preserving conforming artifacts. Root's named Git commit is read through
`ReadAssignmentCommit` for model/accounting validation and mandatory project
checks. A fresh independent reviewer receives the fixed model, exact Root
commit, actual child results, ordinary command observations and check outputs.
Concrete corrections return through Root to their owners in the existing
assignments. Relevant checks and independent review assess the revised commit;
there is no whole-tree work restart or custom probe/disposition protocol.

Inject these concrete ports for a source-bound runner:

- `Workspaces`: the same `gitrepo.Workspaces` instance implements assignment
  preparation, current/exact forks and named commit reads.
- `Integration`: that same instance supplies ordinary Git review/integration.
- `Agents`: an assignment-capable provider adapter, such as the existing Codex
  adapter with an explicitly selected normal CLI profile and login.
- `Checks`: the existing `checks.Runner`, with `CheckRequest` built from each
  selected prepared check profile. The declared project arguments are retained.
- `CreateHome`: exclusively create and check the requested home using the
  engine's `CreateVolumeExclusive` with caller-owned labels. Existing or
  uncertain homes are refused. Credentials and limits remain caller-owned.

`AssignmentRealizationResult` returns the assessment, retained starting copy and
Implement tree, fixed Root handoff, commit-bound candidate/checks, prior reviews,
ordinary result observations and message use. `AcceptedCommit` is set only for
the independently accepted Root result after observed assignment shutdown.
Errors clear acceptance while keeping available work; `TreeCommit` states which
stand the last available candidate represents. The caller owns later workspace
and home cleanup, including inspection of uncertain creation or shutdown.
The Verify prompt includes bounded, explicitly marked excerpts of summaries and
ordinary outputs (at most 32 KiB of result projection in total); the fixed model
context and complete retained result/check/review observations are not clipped.

There is no Publisher parameter, publication permit or legacy `run.Outcome`.
The API transfers nothing to the owner's repository. A4 owns explicit invocation,
CLI/Bootstrap/report migration and the existing guarded publication boundary.
The existing `Apply`, `AssessCaptured`, `VerifyCaptured` and Extract consumers
remain on their source-bound Pass path below until that caller migration.
Source tests of the new seam are controlled local evidence; actual provider
recognition of model gaps, weak tests and realization defects needs a separately
coordinated run.

## Legacy Apply

`Apply(ctx, ApplyRequest)` binds the strategy, the organization (Briefs, own obligations,
write plans, helpers), the required checks, the gates and the budget, requires the container
engine's volume-subpath support, and composes the Passes on `run`. Nobody outside the
Manager tree answers during the run. Every Apply first runs the Model assessment of
`assess.go` (Session C's method; a finding's correction names its class `model_error`,
`model_gap` or `unclassified`): only an accepted assessment releases the work, and a model
problem Root confirms ends the run Failed without publication. The strategy only selects the
work Passes between the assessment and Verify; ownership, gates, evidence, Verify
and publication are the same, and `run.start` and the report record it:

- `direct`, the provisional baseline: Implement, in which every Manager settles its own
  approach and lays its own foundation, submits a basis its children start on, and
  realizes its part, then Verify. Its waits declare its dependencies (`BindWaits`): a wait
  on another Manager outside its own line binds the accepted version, and a revision
  resets or reopens what built on it. Every correction is a new Implement.
- `staged`, kept for comparison: Plan, Prepare and Implement, then Verify; dependencies
  are those of the accepted plans, and a correction restarts at the earliest purpose its
  findings name.

`Apply`, `VerifyCaptured` and `AssessCaptured` (only the Model assessment, without checks or
publication, the evaluation entry of the method's trial) bind through one `applyConfig`, which needs `ApplyRequest.Build`,
the implementation identity of the running product, and returns a `RunBinding`: the entry,
the strategy with its work sequence (led by `assess`; none for `VerifyCaptured`), correction
purposes and the `composition` digest of the assessment, its Pass declarations and Root duty, the build, the content
identities of the execution settings and product defaults, the budget and deadline, every
Manager's resolved profiles, helpers and instructions, and the required checks. `run.start`
records it and `ApplyResult.Binding` returns it for the report. Everything but the work
Passes is shared, Root's duty included: entry and Pass-boundary gates, writer-stop proof, Verify, finding closure
and publication.

A Pass that needs correction restarts the work as its strategy allows and a fresh complete
Verify follows; a Verify that only fresh disputes or disproofs awaiting an independent
assessment keep from being clean is repeated on the unchanged candidate. Only a
Verified outcome is published, through the guarded `Publisher` with read-back; the shared
work volume is closed only after an applied or unchanged receipt and kept as
`RetainedWork` otherwise. Apply never commits. Each required check also passes the number
of its execution profile's own arguments to `run`, so a probe can reuse that profile with
its own arguments.

The mandates ask Plan (staged) or Implement (direct) for the needed dependency stage, to
keep existing conforming artifacts as their own evidence and to report a public behavior the
model leaves open to their parent as a model problem instead of defaulting it; Implement to test the
changes it makes rather than add coverage of conforming behavior, and to wait for another
Manager's basis or result instead of reporting progress; and Verify
to derive claims and permitted counterexamples from the bound model first, probe them at
the public interface and cite receipts or inspected sources for their claims, with parents
challenging their children's assertions.

Implement and artifact review now require model-derived `conditions` and typed,
stand-bound evidence for done/preserved claims. Parents supply explicit `integration`
even without own Definitions; fresh Verify derives its cases before author coverage.
See [the legacy Run submission](../run/README.md#final-assessment-and-host-submission). Implement satisfaction
is a submission to Verify, never verified conformance. `ApplyRequest.WorkDeadline`
optionally binds an earlier cutoff for all non-Verify Passes, preserving the remaining
whole-run window for Verify; its expiry leaves unfinished work Incomplete. Both effective
cutoffs are retained in `BudgetBinding`/run evidence. Setup does not choose this reserve.

`VerifyCaptured(ctx, ApplyRequest)` runs only the independent Verify of the captured stand
with the same configuration and never publishes. It is the evaluation entry for assessing
a known stand through the Pass process; no Command uses it, and the grant-gated
`TestVerifyEvaluationHarness` in `internal/bootstrap` drives it with the real adapters.
`AssessCaptured(ctx, ApplyRequest)` is the evaluation entry of the
[Model assessment trial](../../experiments/model-assessment/README.md): one read-only review
Pass of the captured stand with the product's built-in assessment method, repeated only for
fresh disputes, with the same configuration in a closed Run (no owner channel, answers or
decisions) and without required checks or publication. A confirmed material problem ends
`Failed`; acceptance seals `Satisfied` and ends `Incomplete` with `AssessmentAccepted`,
because this entry realizes and verifies no artifacts. The grant-gated
`TestModelAssessmentHarness` drives it with the real adapters.

Each Brief lists the Manager's own Definitions, included ones with their owners, its
areas and local registry with its entry shape (exact files or directories), the required
checks and its configured instructions. A Definition is stated in full: its body, the
body links as written with the identities they resolve to (`body links:`) and every `spec`
value exactly, decimals in plain notation; foreign ones also with their owner and inclusion
reasons. The binding records the captured `head`, the owner `request`'s digest and per
Manager `context`: the Brief's digest and which identities it states in full and which only
by reference (Root's project inventory and a parent's lines about its children). Root also receives the project obligations: every
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
