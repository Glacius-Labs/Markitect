<a id="canonical-intent-and-reconciliation-source-only-alpha"></a>
# Canonical intent and reconciliation (experimental alpha bundled with v0.14.1)
For a self-contained public end-to-end entrypoint, use the [Classic Commerce example](../examples/classic-commerce/README.md). It includes every selected policy, an explicit runtime, actual opt-in .NET compilation, an independent finite behavior probe and a compilable negative control. Its actor is a deterministic protocol test double; compiler success does not establish autonomous quality.


The `canonical` command is bundled with v0.14.1 as an explicit experimental alpha preview, not stable support. The published v0.13.0 commands and Domain schemas retain their meaning; the v0.14.1 stable command contract remains additive. [Vision](vision.md) owns the product thesis; [the reset decision](design/canonical-projection-reset.md) owns the accepted semantic change. This guide describes the bounded executable slice, not a continuous agent controller or universal code generator.

Users change canonical intent. Markitect plans the affected representations. An Executor produces candidate artifacts; reviewed Apply materializes them; separate Verification records only what the declared checks prove.

## Inputs and ownership

`examples/canonical-projection/canonical.yaml` explicitly lists Definition files, local Module manifest/files and exact package pins. Installable Modules have exactly one type: `schema` or `projection`. A Schema Module cannot depend on a Projection Module. Installing a capability materializes nothing.

Foundation `Projection` declares desired `representation`, exact Definition scope, target location and selected `ProjectionPolicy` references. Neither Definition names a concrete executor/package. Source configuration `projectionBindings` separately binds the full Projection identity to one installed Module name. Its exact pin and unique compatible execution entrypoint are resolved by Host. Missing, duplicate, mismatched or ambiguous bindings fail; no hidden precedence chooses one.

The example selects three commerce Definitions for two independent targets: Markdown and .NET. `EffectAxis` is a custom Kind; its modeled value is the application boundary. Per-Kind .NET guidance is project-owned representation intent. Presence of guidance proves neither adequacy nor implemented behavior.

The source loader uses the ordinary complete snapshot acquisition path. Exact Definition selection is **not** a selective-Git privacy claim. Owner-selected private evidence must use the separate `prepare` handoff flow.

## Brownfield inference and zero-churn adoption

Brownfield discovery runs in reverse: observed representations can inform candidate intent, but inference never changes canonical state. An owner reviews/corrects the proposal and deliberately accepts canonical intent first. Then match exact existing artifact paths to the desired Projection and verify those bytes against immutable source and target revisions. Valid representations are adopted as a baseline without regeneration or normalization. The selection covers only the supplied artifact scope and does not establish whole-repository completeness.

The source-alpha adoption commands take one exact Projection selector and an owner-supplied review reference. This reference is a traceability claim; the command does not authenticate the owner or prove semantic review. `selection.json` requires this closed input shape, including the explicit `activeRecords` array (use `[]` when the caller supplies no active ownership claims):

```json
{
  "artifacts": ["src/Orders/CreateOrder.cs", "src/Orders/CreateOrderHandler.cs"],
  "reviewReference": "review-123",
  "activeRecords": []
}
```

Replace the illustrative artifact paths with the exact existing paths under the selected Projection target. In the compatibility read-only path, caller-supplied active records are ownership claims for conflict checks and no durable ledger is discovered. Conflicts are checked before artifact verification. Its full target snapshot can show existing files outside the exact selection as unmatched. UNKNOWN does not mean rewrite or delete; ordinary reconciliation escalates it until ownership is resolved.

Create and inspect the read-only adoption plan against full immutable commits:

```powershell
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action adopt-plan --base SOURCE --revision TARGET --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet --report .artifacts/canonical-review/selection.json
```

The plan binds the exact selected artifact paths, Projection, source/target inputs, caller-supplied ownership claims and existing immutable checks. Any `materialized-unverified` prospective state in its preview applies only to these selected bytes; it is not semantic verification or a claim about a larger target prefix. Review its plan digest, matched scope and unmatched UNKNOWN artifacts before adoption:

```powershell
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action adopt --base SOURCE --revision TARGET --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet --report .artifacts/canonical-review/selection.json --expect PLAN_DIGEST
```

`adopt` verifies only the existing selected artifact scope. It performs no generation, target writes, record persistence or active-record selection. It returns a ProjectionRecord with `origin: adopted` only when existing immutable checks pass. A failed or missing required check produces no adopted record. This proves the supplied bytes passed the declared checks for that bounded scope; it does not prove semantic adequacy, authenticate the review reference, or establish whole-repository adoption. Preserve the record through explicit caller-owned persistence and active selection, which remain separate operations.

For durable first adoption, pass the closed canonical controller runtime configuration to both calls. Its `recordStore` and `checkInputs` supply the existing external ledger path and exact additional command inputs; its Executor and Verifier entries are required by the shared runtime format but are not invoked by adoption:

```powershell
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --runtime .artifacts/canonical-review/runtime.json --action adopt-plan --base SOURCE --revision TARGET --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet --report .artifacts/canonical-review/selection.json
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --runtime .artifacts/canonical-review/runtime.json --action adopt --base SOURCE --revision TARGET --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet --report .artifacts/canonical-review/selection.json --expect PLAN_DIGEST --write
```

The durable plan binds the runtime digest and whether the external ledger is absent or its current StoreID, head and active record IDs. The selection must contain `activeRecords: []`; ownership is loaded from the ledger. Apply takes the controller lease, refreshes that binding, runs only the selected fixed commands over the exact selected evidence, and then initializes an absent external store if necessary, appends the retained adopted record and compare-and-swaps the active selection. A conflict, stale plan or failed check refuses before initialization. An append followed by a selection error is reported as partial. Host rereads the authoritative ledger: `activeSelectionStatus` is `observed-selected` or `observed-not-selected` when readable, or `unknown` when readback fails. An error may follow a committed selection, so inactive ownership is never assumed. Unobserved ledger head and active IDs are omitted; the attempted record and original error remain visible. No rollback or automatic retry occurs. No repository artifact, canonical input, index or HEAD is changed.

Durable mode reads only canonical source blobs, selected artifact blobs and runtime `checkInputs`. The output identifies its inventory as selected-evidence-only, so its unmatched list is not a complete target-prefix inventory. The persisted record remains `materialized-unverified`; run a separate fresh Verify to record verification. Fixed command success is bounded technical evidence, not semantic assurance, authenticated approval or human acceptance.

## Read-only inspection and reconcile planning

Run from a source checkout with Go 1.27.1. Set `BASE` and `CURRENT` to full immutable Git revisions containing the example; `CURRENT` is the intended canonical state. Module preview may use the working tree, but projection planning requires fixed canonical inputs.

```powershell
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action modules
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action model --revision CURRENT
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action context --revision CURRENT --api-version commerce.example.org/v1 --kind UseCase --namespace commerce --name create-order
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action impact --base BASE --revision CURRENT
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action reconcile-plan --base BASE --revision CURRENT
```

Context shows the exact selected Definition, its Kind purpose/contract and outgoing reference facts. It does not yet compile the historical contextual closure or infer prose dependencies. Impact follows explicit reverse reference dependencies, terminates on cycles, and shows the exact causing property/edge. It does not discover source-code dependencies.

`reconcile-plan` compares intent, active ownership and observed working target bytes. `--evidence active-records.json` optionally supplies a closed JSON array of caller-selected active ProjectionRecords. It proposes bounded work for not-yet-materialized representations, directly affected scopes, incomplete materializations, changed execution capability or recorded target drift. Unchanged representations can report `noApplicableWork`; this means no materialization work detected, **not** verified semantic convergence.

The output separates `scopeAffectedProjections` from `conservativeInvalidatedProjections`. Requests still bind full model digest and revision: a local change may stale every prior plan/evidence binding even when only a subset needs edits. `evidenceRefreshRequired` identifies retained representations that need new binding/evidence without inventing materialization edits. This limitation is explicit. Recorded artifacts are attributed to their own Projection, even when two representations share Definition scope.

Conflicting active owners fail. Unknown files within configured target roots, retired ownership and overlapping proposed target scopes escalate before execution. The initial overlap check is deliberately broader than exact output collision. Caller-selected records do not establish a complete append history or full repository inventory. The separate experimental controller alpha runtime supports exact per-file `targetExclusions` with owner-supplied reasons. Its proposal reports exclusion path, reason, present/missing state and available inventory metadata (path, mode and size); it does not report target-content hashes or imply whole-repository inventory. Subtree/pattern exclusions and undeclared background repository discovery remain unsupported. The separate [controller actions in Usage](usage.md#canonical-controller-actions-source-only-alpha) support Module-owned work proposals over explicit Host-supplied scopes. The section below documents the supported proposal/materialization/verification and evidence-refresh flows. The [live pilot](validation/standard-operating-model.md#live-protocol-pilot) demonstrates one bounded semantic repair with fresh local and parent verification and audited closure; repeated practical reliability remains unproven. The [roadmap assessment disposition](implementation-plan.md#astra-assessment-disposition-and-next-method-checkpoint) owns current operational evidence and remaining work.

<a id="bounded-controller-path-source-only-alpha"></a>
## Bounded controller path (experimental alpha bundled with v0.14.1)

The controller actions provide a current, explicit path from a fixed source revision to candidate materialization and fresh verification. They are separate from `reconcile-plan` and the targeted `request`/`plan`/`apply` tools above. Use the [canonical controller runtime and command contract](usage.md#canonical-controller-actions-source-only-alpha) to prepare the closed runtime JSON, select a finite assurance scope, and keep reports and private logs outside the source snapshot.

Use three full immutable commit IDs with distinct roles: `BASE` is the earlier comparison revision, `SOURCE` is the canonical source/target revision to reconcile, and `EVIDENCE` is the object-only commit returned by successful controller Apply. The following PowerShell 7.4+ sequence keeps every report and log outside the repository. Native stdout redirection preserves the CLI JSON bytes in a separate file; stderr is saved separately for diagnostics. Set the external runtime and these commit IDs for the project; the runtime must satisfy the closed schema in [Usage](usage.md#canonical-controller-actions-source-only-alpha).

For fresh-process Verify, save the successful Apply JSON and use the alpha's `--apply-result` alternative instead of copying `SOURCE` and `EVIDENCE`; see [the controller command contract in Usage](usage.md#canonical-controller-actions-source-only-alpha). The explicit revision form below remains supported.

```powershell
$config = "examples/canonical-projection/canonical.yaml"
$runtime = "C:/review/controller-runtime.json"
$review = "C:/review"
New-Item -ItemType Directory -Force $review | Out-Null
$base = "FULL_BASE_COMMIT"
$source = "FULL_SOURCE_COMMIT"

function Invoke-MarkitectJson([string]$name, [string[]]$arguments) {
    $stdout = Join-Path $review "$name.stdout.json"
    $stderr = Join-Path $review "$name.stderr.txt"
    & go @arguments 1> $stdout 2> $stderr
    $exitCode = $LASTEXITCODE
    [pscustomobject]@{ ExitCode = $exitCode; Stdout = $stdout; Stderr = $stderr }
}

$common = @("run", "./src/cmd/markitect", "canonical", "--repo", ".", "--config", $config, "--runtime", $runtime)
$proposal = Invoke-MarkitectJson "controller-propose" ($common + @("--action", "controller-propose", "--base", $base, "--revision", $source))
if ($proposal.ExitCode -ne 0) { Get-Content -Raw $proposal.Stdout; throw "Propose returned $($proposal.ExitCode); preserve its JSON and stderr for review." }
$proposalReport = Get-Content -Raw $proposal.Stdout | ConvertFrom-Json
if ($proposalReport.status -ne "planned") { Get-Content -Raw $proposal.Stdout; throw "Propose status is $($proposalReport.status); resolve its findings before Execute." }
$work = @($proposalReport.plan.proposals | Where-Object { $_.decision -eq "work" })
if ($work.Count -eq 0) { Get-Content -Raw $proposal.Stdout; throw "No materialization work was proposed; do not invoke Executor or expect an evidence revision." }

$execute = Invoke-MarkitectJson "controller-execute" ($common + @("--action", "controller-execute", "--base", $base, "--revision", $source))
if ($execute.ExitCode -ne 0) { Get-Content -Raw $execute.Stdout; throw "Execute returned $($execute.ExitCode); preserve its JSON and stderr for review." }
$runPath = $execute.Stdout
$run = Get-Content -Raw $runPath | ConvertFrom-Json
if ($run.status -ne "planned" -or @($run.work).Count -eq 0) { Get-Content -Raw $runPath; throw "Execute produced no reviewed materialization work; leave this outcome open." }
# Review $run.status, $run.work, candidate outputs, escalations and $run.digest before continuing.

$apply = Invoke-MarkitectJson "controller-apply" ($common + @("--action", "controller-apply", "--base", $base, "--revision", $source, "--plan", $runPath, "--expect", "REVIEWED_RUN_DIGEST", "--write"))
if ($apply.ExitCode -ne 0) { Get-Content -Raw $apply.Stdout; throw "Apply returned $($apply.ExitCode); do not continue with stale or partial evidence." }
$applyReport = Get-Content -Raw $apply.Stdout | ConvertFrom-Json
$evidence = $applyReport.evidenceRevision
if (-not $evidence) { throw "Apply returned no evidenceRevision; preserve the report and do not claim materialization." }

$verify = Invoke-MarkitectJson "controller-verify" ($common + @("--action", "controller-verify", "--base", $source, "--revision", $evidence, "--write"))
if ($verify.ExitCode -ne 0) { Get-Content -Raw $verify.Stdout; throw "Verify returned $($verify.ExitCode); preserve its JSON and stderr and leave the outcome open." }
Get-Content -Raw $verify.Stdout
```

`REVIEWED_RUN_DIGEST` is the digest from the exact saved Execute report, reviewed before Apply. A nonzero action still emits a bounded JSON report; preserve both stdout and stderr and inspect the status rather than treating the exit code alone as the full result. Execute invokes a configured Executor only for proposed work and does not run the fixed checks. Apply revalidates the saved proposal and writes an object-only `EVIDENCE` commit; it does not move `HEAD` or the index. The resulting records are `materialized-unverified`. Verify uses `--base SOURCE` and `--revision EVIDENCE`, runs the configured fixed checks and fresh Verifier across the configured assurance graph, and appends results only because this example supplies explicit `--write`.

Keep the requested assurance roots/scopes explicit. For a content audit of every configured target scope, set `auditAll: true` in the runtime and include each intended projection in the configured assurance scopes; targeted `auditAll: false` planning leaves unchanged, unobserved projections unresolved. Even `auditAll` is bounded by declared targets and available evidence, does not inventory arbitrary repository files, and is not semantic proof. A `no-materialization-work` proposal is not a verification PASS. A technical `passed` Verify result is limited to its exact scope, checks and evidence and is not human acceptance or proof of repository-wide completeness.

Evidence-only refresh is a separate guarded path for an existing complete materialization whose bytes still match but whose evidence binding is stale. For this separate case, set `$source` and `$evidence` to the exact immutable source and evidence revisions for the selected records, and save a JSON array of exact active Projection IDs in `$refreshIds`; the list is caller-selected, not discovered from arbitrary artifacts. Use the same external runtime and immutable source/evidence revisions:

```powershell
$refreshIds = "C:/review/projection-ids.json"
$refresh = Invoke-MarkitectJson "controller-refresh-propose" ($common + @("--action", "controller-refresh-propose", "--base", $source, "--revision", $evidence, "--report", $refreshIds))
if ($refresh.ExitCode -ne 0) { Get-Content -Raw $refresh.Stdout; throw "Refresh Propose returned $($refresh.ExitCode); review findings before proceeding." }
$refreshPath = $refresh.Stdout
$refreshReport = Get-Content -Raw $refreshPath | ConvertFrom-Json
# Review selected IDs, findings, status and $refreshReport.digest. Preserve the exact proposal.
$refreshApply = Invoke-MarkitectJson "controller-refresh-apply" ($common + @("--action", "controller-refresh-apply", "--base", $source, "--revision", $evidence, "--plan", $refreshPath, "--expect", "REVIEWED_REFRESH_DIGEST", "--write"))
if ($refreshApply.ExitCode -ne 0) { Get-Content -Raw $refreshApply.Stdout; throw "Refresh Apply returned $($refreshApply.ExitCode); preserve its report and leave the outcome open." }
$postRefreshVerify = Invoke-MarkitectJson "controller-verify-after-refresh" ($common + @("--action", "controller-verify", "--base", $source, "--revision", $evidence, "--write"))
if ($postRefreshVerify.ExitCode -ne 0) { Get-Content -Raw $postRefreshVerify.Stdout; throw "Post-refresh Verify returned $($postRefreshVerify.ExitCode); preserve its report and leave the outcome open." }
Get-Content -Raw $postRefreshVerify.Stdout
```

Refresh Apply retains eligible owned bytes and appends new `materialized-unverified` records; it invokes no Module, Executor or Verifier and never replays an old PASS. Run fresh `controller-verify` against the same `SOURCE` and `EVIDENCE` revisions after refresh. The full eligibility and refusal conditions are in the [retained evidence refresh contract](usage.md#retained-evidence-refresh-source-only-alpha).

A failed semantic Verifier result remains failed. The experimental alpha controller supports narrowly gated Dotnet repair scheduling on a subsequent reviewed proposal/run; it is not a general repair controller and remains experimental alpha rather than stable v0.14.1 support. After a failed result has been appended, a later explicit `controller-propose` may schedule one repair run only if `auditAll: true` is set in the unchanged runtime, canonical intent is unchanged, the exact previously verified artifact, check, runtime and child evidence are still current, and the latest record/result identifies a validated completed semantic failure with bounded findings. Keep `auditAll: true` from the initial Verify because its result binds the runtime configuration; changing it later invalidates reuse. Missing, stale, incomplete, invocation-error, owner-escalated or fixed-check-failure evidence remains inspection or eligible evidence refresh; deterministic Modules do not gain semantic repair behavior. A scheduled repair still requires one separately reviewed Execute/Apply/Verify run with no hidden retry. The capability alone is not repair evidence. The [live pilot](validation/standard-operating-model.md#live-protocol-pilot) separately records one real Executor repair, fresh local and parent verification, and final audited-scope closure; this bounded result does not establish a general repair success rate. Do not treat rerunning Verify, cached evidence or `no-materialization-work` as repair or closure. Incomplete, escalated, refused, or missing required scope/check/Verifier evidence leaves the configured assurance outcome open; resolve the stated gap and obtain fresh evidence before reporting technical completion.

## Executor tools and explicit mutation

Targeted request/plan/apply actions are tools for a selected reconcile scope, not the primary user request to implement a file. Markdown uses a symbolic renderer inside the same candidate/verification lifecycle. .NET consumes supplied candidate bytes and does not invoke a model itself.

```powershell
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action request --revision CURRENT --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet
```

A .NET candidate is one UTF-8 JSON object: `requestDigest` must match that exact request, and `files` contains exact `path`/`content` pairs. No globs, duplicate paths/object members, implicit scope expansion, source mutation or arbitrary file extensions are accepted. Candidate preparation never creates canonical truth.

Build one fixed executable before reviewing mutable targets, and keep candidate/plan/record files outside admitted snapshot inputs. The following `.artifacts` workspace is excluded by the source acquisition contract; using a new ordinary repository file for the saved plan would change the reviewed input snapshot. Rebuilding a tool between Plan and Apply can also invalidate its tool binding.

```powershell
New-Item -ItemType Directory -Force .artifacts/canonical-review | Out-Null
go build -o .artifacts/canonical-review/markitect.exe ./src/cmd/markitect
$markitectBinary = (Resolve-Path .artifacts/canonical-review/markitect.exe).Path
& $markitectBinary canonical --repo . --config examples/canonical-projection/canonical.yaml --action plan --revision CURRENT --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet --report .artifacts/canonical-review/candidate.json > .artifacts/canonical-review/reviewed-plan.yaml
& $markitectBinary canonical --repo . --config examples/canonical-projection/canonical.yaml --action apply --revision CURRENT --api-version markitect.foundation/v1 --kind Projection --namespace commerce --name application-dotnet --report .artifacts/canonical-review/candidate.json --plan .artifacts/canonical-review/reviewed-plan.yaml --expect REVIEWED_CANDIDATE_DIGEST --write
```

`--expect` is the emitted candidate digest, not an authentication token. Apply rebinds and recomputes the complete saved review report against fixed source, observed bytes, config/check arguments, tool, protected source and candidate inputs. Changing any reviewed input rejects the plan. Named non-protected branch, exact-path, alias/collision, preimage and partial-failure protections reuse the validated writer. Deletion is unsupported in this slice.

Apply emits a `materialized-unverified` ProjectionRecord in its report. It does not silently persist a ledger, run verification or grant acceptance. An already materialized candidate returns `already-materialized`, retained paths and an unverified retained-artifact record. No target write occurred; choosing and appending active ownership remains caller-owned. Persist records explicitly in an owner-controlled ledger/workspace. JSON input preparation from the YAML report is currently authoring friction; executable tests exercise exact record payloads. Records, selected active state and append history are distinct.

## Immutable verification and assurance

Commit materialized outputs to obtain full immutable `TARGET`. Supply one closed JSON ProjectionRecord in `.artifacts/canonical-review/record.json`:

```powershell
go run ./src/cmd/markitect canonical --repo . --config examples/canonical-projection/canonical.yaml --action verify --base CURRENT --revision TARGET --evidence .artifacts/canonical-review/record.json
```

Host checks exact source/capability/scope/policy and recorded target bytes/modes, then executes declared literal check argument arrays against immutable materializations. Each check identity binds its arguments plus the full evidence revision/snapshot, including checker source bytes. It does not hash the PATH executable, sandbox the process, authenticate the verifier or prove check independence/sufficiency. Missing checks are incomplete; failed checks fail. Evidence remains separate from canonical intent and ProjectionRecord.

Pure recursive assurance combines explicit bounded parent/child scopes and current verification evidence. Each parent needs its own declared checks plus passing children; a passing child alone cannot make the composition pass. It validates supplied records but does not schedule agents or execute a recursive project controller.

The executable test is `go test ./src/harness/examples -run '^TestCanonicalProjectionBoundCandidateApplyVerifyAndRepair$' -count=1`. The fixture checker proves selected boundary markers and artifact shape only. A preserved invented enum counterexample passes the old weak markers but fails the current boundary check. Passing either checker does not establish CreateOrder business behavior or general AI reasoning quality. The [validation report](validation/canonical-projection-reset.md) records exact candidate gates and remaining pressure.
