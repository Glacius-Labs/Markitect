# KG01 twelve questions

Fixed reference: `1495e1be7b0046711531fa42c8407fba67b8e814`. Every source path/test name below is resolved against this commit. Shop paths are relative to `examples/project-world`; identities use API `project.markitect.example.org/v1alpha1` and the full Core four-part key. Short `namespace/name` labels below retain their kind in the stated path. No supplied operational record or Decision is invented for Shop. Existing fixture tests are source witnesses, not new executed results.

Each expected answer is a declaration or observed binding, not a claim of semantic completeness. Manager query views must select permitted nodes AND edges before indexing. Forbidden and absent identities must have the same lookup behavior; no hidden-node counts, boundary identifiers or text should leak. Host owns that selection policy. Unknown/unprovided data remains explicit.

## 01 — Who owns a declared file and the definition that claims it?

**Input/scope:** Shop Orders Manager; `src/shop/orders/order.py` and Statement `commerce.sales.orders/cancel-order`.
**Expected facts/path:** File → selected ownership prefix `src/shop/orders/` → Manager `commerce.sales.orders/orders`; Statement → nearest namespace Manager Orders; Artifact `commerce.sales.orders/order-lifecycle` → declared path → file, and realizes → cancel-order. These are separate derivations.
**Today:** Analyze computes them; Report files carry Owner/Artifact/Statement/Check IDs. Context contains own artifacts; Impact routes the owner when selected bytes change. Sources: `examples/project-world/.markitect/model/commerce/sales/orders/{manager,artifacts,cancel-order}.yaml`, `internal/modules/projectmodel/analyze.go:fileOwner`, `types.go:FileEntry`.
**Privacy/unknown/next:** Missing inventory is unknown, not an unowned fact; do not infer file ownership from prose or Artifact owner alone. KG02 retains declaration source plus selector/membership derivation. This says who declared responsibility, not who authored every line.

## 02 — Which consumers depend on a changed contract, and why?

**Input/scope:** Both revisions of Shop release contract, explicit change to Statement `commerce.sales.inventory.reservations/release-reservation`; whole-project Impact view supplied by Host.
**Expected facts/path:** Orders cancel-order → uses AND requires → release-reservation; contract owner is `commerce.sales.inventory/inventory` (namespace inheritance, no reservation Manager); cancel-order ← realizes ← order-lifecycle → paths `src/shop/orders/`, `docs/cancellation.md` and checks → `engineering/cancellation-tests`. Requires additionally includes reservation-lifecycle → `src/shop/inventory/` → same Check; uses alone routes context/owner without target coverage.
**Today:** Impact follows reverse consumers transitively over both graphs and routes ancestors. Context selects direct public foreign contracts. Sources: Shop `orders/cancel-order.yaml`, `inventory/{manager.yaml,reservations/artifacts.yaml,reservations/release-reservation.yaml}`, `projectmodel/impact.go:Impact`, `TestImpactUsesAddsContextWhileRequiresAddsCoverage`, `TestImpactFollowsTransitiveConsumers`.
**Privacy/unknown/next:** Impact access is not unrestricted manager Context. KG03 witnesses explain exact typed relations without replacing Impact. Incomplete source broadens Impact. Declared consumer impact does not prove all runtime callers were found.

## 03 — What changes when the relation itself is removed?

**Input/scope:** Synthetic base/candidate pair derived from Shop: remove cancel-order.requires → release-reservation; do not alter the contract or implementing files.
**Expected facts/path:** Removed base edge stays available to change analysis; former required contract owner, unchanged reservation-lifecycle path and cancellation-tests remain in review coverage. Label witness revision as base/removed, not candidate/current.
**Today:** Impact unions base/candidate relations and coverage; source witness `TestImpactRetainsRemovedRequiredObligationAndRoutesOwners` and `TestImpactNewRequirementIncludesUnchangedExpectedPathAndCheck` in `projectmodel_test.go`. Context on candidate alone no longer selects a removed-only dependency (Shop retains a uses edge, so still includes release).
**Privacy/unknown/next:** KG02 indexes individual snapshots; a future Host comparison must supply both labeled graphs. KG03 navigation of candidate alone is not an Impact substitute. A removed reference does not prove that underlying obligation was intentionally retired.

## 04 — Which decision supports this rule?

**Input/scope:** Rule release-reservation in Shop model; separately a schema-valid synthetic Decision subject → that rule and actor → Inventory.
**Expected facts/path:** Actual Shop: no declared supporting Decision, answer unknown/not supplied. Fixture: Decision node with reason/decision text/source → subject rule, actor Inventory; preserve exact identity.
**Today:** Schema/Core supports Decision references; Analyze validates them but Report does not expose Decisions. A model-only Decision delta is conservative unknown-impact (`TestImpactDecisionOnlyDeltaRoutesFullDeclaredReview` uses a changed ModelDigest fixture). Source: `projectmodel/projectmodel.go:Schema`, `analyze.go`, `types.go:Report`, `core/types.go`.
**Privacy/unknown/next:** KG02 may project existing Core Decision directly, without new schema. Host must explicitly scope it; actor reference is provenance, not consent/authenticated authority. No decision text proves the rule true.

## 05 — Which accepted work led to an applied change?

**Input/scope:** Explicitly supplied Host record set for a single exploration scope; no actual Shop run supplied.
**Expected facts/path:** Work scope/binding → accepted model/history event or verified resolution → Plan/ManagerTask → candidate/file digest → Check/Review/Verify → ApplyReceipt; show each missing join unknown. Actual Shop answer: operational chain not supplied.
**Today:** `projectexplore/types.go:{Record,Binding,ApplyReceipt}`, `projectrun/types.go:{PlanRecord,RunReport,ReviewRecord,CheckResult,VerifyReport}`, `projectrun/deliver.go`, `projectbriefing/store.go:{VerifiedResolutionEvidence,Resolution}` hold separate records; readiness witness `TestReadinessRequiresAcceptedModelDecisionResolutionAndExactAcknowledgement`.
**Privacy/unknown/next:** Integration-owned KG05 adapters must select exact scope/source; no module reads ledgers. Readiness/Apply success is distinct from authenticated human acceptance. Neither Context nor Impact currently returns this complete chain. KG02 does not claim operational trace coverage.

## 06 — Which required artifact or check is missing?

**Input/scope:** Orders view and explicit file inventory; negative fixture removes `src/shop/orders/order.py` AND the remaining files under that declared prefix, or removes required Artifact checks.
**Expected facts/path:** order-lifecycle required → expected `src/shop/orders/`/`docs/cancellation.md` → absence/incomplete finding; realizes → cancel-order; checks → cancellation-tests. Missing check declaration differs from declared check lacking execution.
**Today:** Analyze checks required paths, keeps missing mappings (`TestAnalyzeKeepsRequiredArtifactWhenMappingIsAbsent`). Required nonempty checks gate is `projectrun/plan.go:planChecksWithImpact`; Analyze alone can accept a required Artifact with no checks. Unexecuted declared Check is unknown, not failed/passed.
**Privacy/unknown/next:** Context filters own findings; unknown inventory prevents asserting absence. KG02 neutral file facts must distinguish known present/missing/unknown; no fabricated check result. KG03 can show declaration links; coverage completeness remains existing validator/Host responsibility.

## 07 — Was a concept renamed or replaced?

**Input/scope:** Fixture rename `commerce.sales.orders/cancel-order` → `cancel-unshipped-order`, with adjusted references, fixed before/after revisions.
**Expected facts/path:** Old identity removed, new identity added; history retains source locations. No declared continuity fact exists, answer to same-concept identity is unknown.
**Today:** Core identity uses version/kind/namespace/name. `projectbriefing/briefing.go:Build` diffs added/removed/modified; `TestImpactTracksNamespaceAndSourcePathMovement` checks movement. Impact considers both identities conservatively. Candidate Context includes new identity only.
**Privacy/unknown/next:** KG04 needs explicitly coordinated continuity vocabulary/decision. KG02 must not merge similar names or rewrite history. Removal-plus-add is the current contract, not evidence of intended conceptual equivalence.

## 08 — Why is a relationship visible to one scope but hidden from another?

**Input/scope:** Existing Context fixture with short namespaces `orders`/`inventory`: public inventory/release-reservation uses private inventory/internal-guard; Orders has direct public dependency.
**Expected facts/path:** Orders sees public release contract and direct dependency, never its outgoing relation to private guard, sibling artifacts or guard ID. Inventory own view may contain guard. Children retain Purpose but omit Instructions.
**Today:** `impact.go:Context/visibleRelations`; `TestContextIncludesDirectPublicContractsAndExcludesSiblingInternals`; Analyze rejects direct private cross-manager references (`TestAnalyzeRejectsPrivateCrossManagerReference`). These fixture namespaces are not Shop namespaces.
**Privacy/unknown/next:** Host must produce an already selected graph including edge redaction; generic traversal cannot admit sibling data from raw Core automatically. Do not report a hidden target or hidden counts as an explanation. Context selection is not a filesystem/OS access barrier; Impact uses its separately authorized project scope.

## 09 — Which supplied claims explicitly conflict?

**Input/scope:** Brownfield fixed discovery/distillation fixture scope `orders`; selected implementation `src/orders/cancel.go`, documented intent `docs/order.md`, separately submitted runtime `runtime/test.log`.
**Expected facts/path:** Explicit Contradiction `implementation-intent-gap` → its two ClaimIDs → their EvidenceRefs, question `clarify-cancellation`; distinguish static observation, documented intent and submitted runtime record. No assertion of actual contradiction in Shop.
**Today:** `projectadoption/projectadoption_test.go:validDistillation`, `TestDistillationPreservesContradictionsAndSeparatesEvidenceMethods`, `TestResolutionRequiresExplicitPerScopeDecisionAndAnswers`; `types.go:{Claim,Contradiction,Question,Resolution}`. Context/Impact do not expose these Host entities.
**Privacy/unknown/next:** KG05 projects supplied scoped conflicts only, with exact source/line/digest. Unknown absence of records is not proof of consistency; arbitrary semantic disagreement remains undetected. A question resolution is not runtime evidence or universal truth.

## 10 — Is evidence current, stale, historical, partial, or unknown?

**Input/scope:** A supplied verification receipt with exact model/source/runtime/selection/plan binding, compared to explicitly selected candidate; negative fixture changes any bound input. No Shop receipt supplied.
**Expected facts/path:** Check → executed result → bound candidate/model/snapshot/runtime → verification/apply receipt; mismatch = stale for this selection, prior PASS remains historical. Missing checks/telemetry/joins = partial or unknown, never zero usage/current PASS.
**Today:** Host plan/apply/freshness checks and `projectrun/full_verify.go`; `full_verify_test.go` stale binding/unknown coverage cases; `projectbriefing/store.go:ResolveVerified`; model semantic digest cannot replace snapshot bytes. Root startup checks in the reference are historical for their fixed BASE.
**Privacy/unknown/next:** KG05 owns actual binding comparison and status adaptation; generic KG02 stores no authoritative operational verdict. Context/Impact return declarations/routing, not current execution status. Reindexing cannot certify evidence freshness or user acceptance.

## 11 — Why did Context include a contract?

**Input/scope:** Shop Orders Manager at reference; ask about release-reservation.
**Expected facts/path:** own cancel-order → direct uses AND requires → public foreign release-reservation; owner Inventory. Also reservation is a direct public uses contract. Do not take arbitrary transitive closure.
**Today:** `impact.go:Context` iterates both own Uses and Requires, includes foreign Public targets and filters their relations. Shop `orders/cancel-order.yaml` and `inventory/reservations/*.yaml` establish exact inputs.
**Privacy/unknown/next:** KG03 can provide witnesses only over Host-selected edges. Must not turn Context into unrestricted graph dump. This corrects the design table's shorthand 'direct public uses': requires also participates. Included contract is declared context, not proof it was read or obeyed by an agent.

## 12 — Does the model declare full coverage for this obligation?

**Input/scope:** Shop `.markitect/project.yaml` coverageMode full; cancel-order requires release-reservation; both required realization Artifacts and Check plus selected inventory.
**Expected facts/path:** requires → release → realizes inverse → reservation-lifecycle → expected paths/check; orders own realization → order-lifecycle → paths/check. Manifest says full coverage, but that is a declaration of selected model policy. Absence of a realization for an arbitrary Statement is not universally rejected by Analyze.
**Today:** `projectwork/config.go`, `projectmodel/analyze.go`, `projectrun/plan.go`, `projectrun/full_verify.go`; Check limitation in `engineering/checks.yaml` restricts SQLite cancellation suite. Existing checks cannot establish completeness of all code/prose or production/concurrency behavior.
**Privacy/unknown/next:** Do not expose foreign implementation from manager Context. KG03 must not emit generic full-coverage certification; Integration preserves existing required-path/check gates and broadens unknown Impact. Future stronger obligation requirements need explicit KG04 policy, not an implicit graph query assumption. Actual runtime chain and human acceptance remain unprovided.
