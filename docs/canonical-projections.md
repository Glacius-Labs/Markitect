# Canonical intent and reconciliation (source-only alpha)

This is the new `canonical` command in unreleased source. Published v0.13.0 commands, Domain schemas, package versions and distributed artifacts retain their meaning. [Vision](vision.md) owns the product thesis; [the reset decision](design/canonical-projection-reset.md) owns the accepted semantic change. This guide describes the bounded executable slice, not a continuous agent controller or universal code generator.

Users change canonical intent. Markitect plans the affected representations. An Executor produces candidate artifacts; reviewed Apply materializes them; separate Verification records only what the declared checks prove.

## Inputs and ownership

`examples/canonical-projection/canonical.yaml` explicitly lists Definition files, local Module manifest/files and exact package pins. Installable Modules have exactly one type: `schema` or `projection`. A Schema Module cannot depend on a Projection Module. Installing a capability materializes nothing.

Foundation `Projection` declares desired `representation`, exact Definition scope, target location and selected `ProjectionPolicy` references. Neither Definition names a concrete executor/package. Source configuration `projectionBindings` separately binds the full Projection identity to one installed Module name. Its exact pin and unique compatible execution entrypoint are resolved by Host. Missing, duplicate, mismatched or ambiguous bindings fail; no hidden precedence chooses one.

The example selects three commerce Definitions for two independent targets: Markdown and .NET. `EffectAxis` is a custom Kind; its modeled value is the application boundary. Per-Kind .NET guidance is project-owned representation intent. Presence of guidance proves neither adequacy nor implemented behavior.

The source loader uses the ordinary complete snapshot acquisition path. Exact Definition selection is **not** a selective-Git privacy claim. Owner-selected private evidence must use the separate `prepare` handoff flow.

## Read-only inspection and high-level planning

Run from a source checkout with Go 1.27.1. Set `BASE` and `CURRENT` to full immutable Git revisions containing the example; `CURRENT` is the intended canonical state. Module preview may use the working tree, but projection planning requires fixed canonical inputs.

```powershell
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action modules
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action model --revision CURRENT
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action context --revision CURRENT --api-version commerce.example.org/v1 --kind UseCase --namespace commerce --name create-order
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action impact --base BASE --revision CURRENT
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action reconcile-plan --base BASE --revision CURRENT
```

Context shows the exact selected Definition, its Kind purpose/contract and outgoing reference facts. It does not yet compile the historical contextual closure or infer prose dependencies. Impact follows explicit reverse reference dependencies, terminates on cycles, and shows the exact causing property/edge. It does not discover source-code dependencies.

`reconcile-plan` compares intent, active ownership and observed working target bytes. `--evidence active-records.json` optionally supplies a closed JSON array of caller-selected active ProjectionRecords. It proposes bounded work for not-yet-materialized representations, directly affected scopes, incomplete materializations, changed execution capability or recorded target drift. Unchanged representations can report `noApplicableWork`; this means no materialization work detected, **not** verified semantic convergence.

The output separates `scopeAffectedProjections` from `conservativeInvalidatedProjections`. Requests still bind full model digest and revision: a local change may stale every prior plan/evidence binding even when only a subset needs edits. `evidenceRefreshRequired` identifies retained representations that need new binding/evidence without inventing materialization edits. This limitation is explicit. Recorded artifacts are attributed to their own Projection, even when two representations share Definition scope.

Conflicting active owners fail. Unknown files within configured target roots, retired ownership and overlapping proposed target scopes escalate before execution. The initial overlap check is deliberately broader than exact output collision. Caller-selected records do not establish a complete append history or full repository inventory. Arbitrary target exclusions/classification and autonomous per-Module work discovery are later steps.

## Executor tools and explicit mutation

Targeted request/plan/apply actions are tools for a selected reconcile scope, not the primary user request to implement a file. Markdown uses a symbolic renderer inside the same candidate/verification lifecycle. .NET consumes supplied candidate bytes and does not invoke a model itself.

```powershell
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action request --revision CURRENT --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet
```

A .NET candidate is one UTF-8 JSON object: `requestDigest` must match that exact request, and `files` contains exact `path`/`content` pairs. No globs, duplicate paths/object members, implicit scope expansion, source mutation or arbitrary file extensions are accepted. Candidate preparation never creates canonical truth.

```powershell
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action plan --revision CURRENT --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet --report candidate.json > reviewed-plan.yaml
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action apply --revision CURRENT --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet --report candidate.json --plan reviewed-plan.yaml --expect REVIEWED_CANDIDATE_DIGEST --write
```

`--expect` is the emitted candidate digest, not an authentication token. Apply rebinds and recomputes the complete saved review report against fixed source, observed bytes, config/check arguments, tool, protected source and candidate inputs. Changing any reviewed input rejects the plan. Named non-protected branch, exact-path, alias/collision, preimage and partial-failure protections reuse the validated writer. Deletion is unsupported in this slice.

Apply emits a `materialized-unverified` ProjectionRecord in its report. It does not silently persist a ledger, run verification or grant acceptance. An already materialized candidate returns `already-materialized`, retained paths and an unverified retained-artifact record. No target write occurred; choosing and appending active ownership remains caller-owned. Persist records explicitly in an owner-controlled ledger/workspace. JSON input preparation from the YAML report is currently authoring friction; executable tests exercise exact record payloads. Records, selected active state and append history are distinct.

## Immutable verification and assurance

Commit materialized outputs to obtain full immutable `TARGET`. Supply one closed JSON ProjectionRecord in `record.json`:

```powershell
go run ./cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action verify --base CURRENT --revision TARGET --evidence record.json
```

Host checks exact source/capability/scope/policy and recorded target bytes/modes, then executes declared literal check argument arrays against immutable materializations. Each check identity binds its arguments plus the full evidence revision/snapshot, including checker source bytes. It does not hash the PATH executable, sandbox the process, authenticate the verifier or prove check independence/sufficiency. Missing checks are incomplete; failed checks fail. Evidence remains separate from canonical intent and ProjectionRecord.

Pure recursive assurance combines explicit bounded parent/child scopes and current verification evidence. Each parent needs its own declared checks plus passing children; a passing child alone cannot make the composition pass. It validates supplied records but does not schedule agents or execute a recursive project controller.

The executable test is `go test ./examples -run '^TestCanonicalProjectionBoundCandidateApplyVerifyAndRepair$' -count=1`. The fixture checker proves selected boundary markers and artifact shape only. A preserved invented enum counterexample passes the old weak markers but fails the current boundary check. Passing either checker does not establish CreateOrder business behavior or general AI reasoning quality. The [validation report](validation/canonical-projection-reset.md) records exact candidate gates and remaining pressure.
