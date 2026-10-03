# Resolved-target equality: design assessment

**Decision:** Implement one finite, generic `same-target` Domain assertion. This accepted contract is not part of published v0.11.0; subsequent release status must be established by immutable publication evidence. Existing package versions remain immutable; adding the assertion requires new exact-pinned package versions. The published baseline and current-source evidence are described separately in the [pressure report](domain-language-pressure.md) and [experiment notes](software-architecture-experiment.md).

## Question and current-language boundary

Should a finite Domain assertion compare the resource reached through two explicitly named relation paths from one selected subject? The two candidate statements are:

1. For an explicitly selected UseCase, its direct Module and the Module reached through its optional Feature are the same resource.
2. For a Deployment, the Product reached through its Service is the same Product reached through its Environment.

Before `same-target`, the finite assertion language could not express either join. `equal` compares a scalar field on one selected resource; `allowed-targets` restricts target kinds; neither followed a path and compared resolved identities. The [software-architecture pressure tests](../../examples/software-architecture_pressure_test.go) and the independent [Delivery acceptance fixture](../../examples/delivery-target-equality/README.md) establish those gaps in the prior language. Current source now exercises the bounded assertion in these cases. Together, they establish a generic modeling need, not broad market demand or validated customer benefit.

## Alternatives

| Option | UseCase and optional Feature | Deployment, Service and Environment | Assessment |
|---|---|---|---|
| Normalize the model so only one owner is recorded | Removing `UseCase.module` would make the optional Feature the only way to derive a Module; that loses ownership for featureless UseCases. Making an owner/container mandatory for every UseCase changes the model and adds an intermediate concept. Removing `Feature.module` preserves UseCase ownership but loses the canonical Module ownership of a Feature. | Service and Environment are independent catalogs, each assigned to a Product. Removing one Product assignment or deriving it from the other erases the independent assignment the deployment consistency rule is meant to compare. Adding a new shared catalog/owner to bypass the check moves the same decision elsewhere. | Normalization is best when the two values are truly redundant and one can be made authoritative without losing valid cases. Neither proposed case has that property as stated. Do not normalize merely to avoid a small predicate. |
| Project-owned `verify` check | A test can load the frozen project snapshot, follow `UseCase.feature` when present, and compare its Module identity with `UseCase.module`. | A test can follow Deployment→Service→Product and Deployment→Environment→Product and compare Product identities. | This is the lowest-scope choice for a one-project rule and keeps project semantics out of Core. It is executable from explicitly declared checks over the materialized snapshot. Its result is a gate result, not a Domain `PolicyResult`; it is not automatically joined to generated Domain views, subject-level context or the existing exception lifecycle. The check executable and its runtime/toolchain also need their own versioning and review. Prefer this when the rule is local, temporary, or its policy language is still changing. |
| Configured adapter | A project-specific adapter could inspect selected normalized-model data and report the mismatch. | The same adapter could check either owner path without reading .NET or filesystem conventions. | An adapter is appropriate when the assertion depends on an external observed system or provider-specific facts. For a pure canonical-graph invariant, it introduces an adapter protocol/configuration and still leaves the result outside Domain constraints and PolicyResults. Do not present an adapter as a generic join API that Markitect already provides. |
| Bounded Core assertion | One composed Software Domain declares a selected UseCase cohort and compares `[belongsToModule]` with `[realizesFeature, belongsToModule]`. | A Deployment Domain compares `[deploysService, belongsToProduct]` with `[deploysTo, belongsToProduct]`. | This expresses the same finite graph operation in two different vocabularies, can be versioned in exact-pinned Domain packages, and makes results available to model/check/context and generated policy views. Independent acceptance probes establish both prior-language gaps; the bounded operator below is the accepted source design. |

`Project.spec.checks` is therefore the preferred fallback when only one project needs the rule. A check can be declared and executed against the fixed snapshot, but Markitect does not reinterpret its output as a per-resource policy result or prove its executable's continuing semantics. A generic equality assertion is not a source-code analyzer and must not read directories, infer architecture from conventions, or execute arbitrary expressions.

## Optional Feature and selection boundary

Feature remains optional. A UseCase with no Feature remains valid unless the project explicitly places that UseCase in the ownership-check cohort. The current selector can select by exact kind and labels, not by property presence, so use a transparent label such as `feature-ownership: required` as the opt-in scope. Do not infer or add that label automatically from `spec.feature`.

This only supports the statement “the selected feature-ownership cohort is checked.” It does **not** prove that every Feature-bearing UseCase was selected; an omitted label is outside the policy. For a selected subject, each path must resolve exactly one edge at every step. Missing, ambiguous, unresolved, or wrong-kind paths produce structural `constraint.path` diagnostics with no `PolicyResult` and cannot be waived. A fully resolved pair of different Module GraphKeys produces a failed per-resource `PolicyResult` and may use the existing exact-source waiver mechanism. This keeps ordinary featureless UseCases valid and avoids turning incomplete selected paths into a waiver. The cohort label and its limits must remain visible in Domain views, context and check results.

## Accepted assertion contract

Keep the operation finite and closed. An illustrative assertion shape is:

```yaml
constraints:
  - name: selected-feature-use-cases-share-module-owner
    select:
      kind: UseCase
      labels: {feature-ownership: required}
    assert:
      op: same-target
      left: [belongsToModule]
      right: [realizesFeature, belongsToModule]
```

The accepted syntax has `op: same-target`, `left` and `right` arrays of relation names, each with one or two entries, and a required concrete `select.kind`. Both ordered paths start at each selected subject. Registration verifies that every named relation exists in the same Domain API version, each kind transition is compatible, all target-kind sets are concrete (no `*`), and the terminal kind sets overlap. Runtime requires exactly one raw reference and exactly one resolved edge at every step. It compares exact terminal GraphKeys. The assertion does not silently skip an optional path: explicit selection determines which optional relationship is required for this check.

For both passing and failing comparisons, serialized policy results expose `comparison.left` and `comparison.right`. Each side contains the ordered `relations`, resolved `steps`, and terminal `target`, making the actual operands visible without exposing implementation type names.

### Outcomes and failure semantics

| Case | Result |
|---|---|
| Either side has zero raw references or zero resolved edges at a step | Structural `constraint.path` diagnostic; no PolicyResult and no waiver. |
| Either side has multiple raw references or multiple resolved edges at a step | Structural `constraint.path` diagnostic; no PolicyResult and no waiver. |
| A reference is unresolved | Structural `constraint.path` diagnostic at the failed step; no PolicyResult and no waiver. |
| The path starts from a wrong source kind, or an intermediate/terminal target has a kind/API version not allowed by its relation | Registration rejects statically incompatible named paths where possible; a malformed runtime graph produces a structural `constraint.path` diagnostic, not a waiverable result. |
| A path revisits a resource | The operation follows only its statically declared one- or two-edge path, so evaluation terminates. A cycle is rejected only when an applicable relation declares `acyclic: true`; `same-target` does not add a cross-relation cycle rule. |
| Both paths resolve to the same terminal GraphKey | Passed per-resource PolicyResult with both comparison traces. |
| Both paths resolve, but to different terminal GraphKeys | Failed per-resource PolicyResult with both traces and the differing targets; the existing policy-exception process may waive it if its exact digests remain current. |

The Deployment scenario uses required singleton relations and needs no selector label: the selected subject is `Deployment`; each side must resolve to one Product. It demonstrates that the operation is a relation-path predicate, not Feature logic. This form does **not** implement the UseCase→many Aggregates→Module case; that would need explicit set/fan-out and empty-set semantics and is out of scope.

The source implementation binds provenance and dependency behavior as follows:

- `constraintDigest` binds the operator, selector, both ordered paths, Domain API version, every referenced full relation definition, and the consumed source-property schema that can affect reference interpretation. A path, relation, or relevant schema semantic change therefore stales an old exception.
- The subject evidence digest binds the selected subject, all traversed canonical resource contents, and the semantic path trace. It excludes source line/path metadata. This is deliberately conservative: a content change anywhere in a traversed resource may stale an exception even if the changed field did not alter the comparison. Do not claim minimal staleness.
- The result trace names both paths and each exact resolved GraphKey, with source origin/path/line available to explain each edge. A reviewer can see why the comparison applies and what differed without inferring it from a pass/fail label.
- `PolicyDependencies` record attempted path sources and successfully resolved targets, including valid prefixes when a later path step fails, independently of relation `context` and `invalidate` flags. Impact unions dependencies from base and candidate snapshots: changed path inputs select affected policy subjects, then relevant context consumers and reverse invalidation dependents are included. This does not turn the path into ordinary context. Collection, inventory, configuration and unknown-input changes retain conservative treatment; this is not a generic adapter-dependency graph.
- Separate constraints remain separate results. Contradictory constraints are all visible and make validation fail; there is no precedence, last-wins merge, implicit override or automatic human approval.

## Evidence and scope decision

Before the operator, the UseCase/Feature probe and independent Deployment probe showed valid typed paths to different owners without an equality result. Current-source fixture tests exercise matching and mismatching Product assignments, distinguish non-context ownership from invalidation, and measure affected path dependents while leaving an unrelated cohort out. These are concrete examples for one bounded operator, not a case for a general query language.

**Decision: implement the bounded `same-target` contract above.** Keep the explicit project-check fallback for project-specific rules. Do not add Pattern/Composition semantics, relation fan-out, inverse ownership, identity allowlists, arbitrary field selectors, general path predicates, filesystem access, or source-code interpretation as part of this decision. A design decision or local tests alone do not establish publication.

No inference follows about reduced development effort, lower drift, or demand from these two examples. Measure usefulness and maintenance cost on repeated real change work before making those product claims.

## Answers to the ten decision questions

| Question | Answer |
|---|---|
| 1. Can independent Domains express the same needed rule without Core-specific architecture cases? | Yes for these two independent singleton path equalities: Software UseCase/Feature ownership and Delivery Deployment/Product ownership use the same named-path operation. Neither adds an architecture-specific Core kind. |
| 2. Could normalization represent the ownership more simply? | Normalize when two facts are genuinely redundant and one can be authoritative. Here a Feature is optional, so deriving every UseCase owner from its Feature would lose featureless UseCases; Service and Environment retain independent Product associations. The equality check preserves both facts and compares them. |
| 3. Why Core rather than a project check or adapter? | The assertion reads only canonical typed graph references and recurs in two unrelated Domains; making it a finite Domain result connects it to normal policy evidence and exact package evolution. Use a project check when the rule is unique/local, or an adapter when it depends on an external observed system. |
| 4. What exact finite rule is justified? | Compare two explicit one- or two-relation paths from a selected kind; each hop has exactly one raw and resolved target; terminal GraphKeys must match. Nothing more general is required by the evidence. |
| 5. Can authors see why a result applies? | Yes for a valid comparison: both serialized traces show relation names, ordered source/target steps, target identities, and edge provenance. Invalid paths are structural diagnostics and do not create an exception-eligible PolicyResult. |
| 6. Is impact bounded for a local owner change? | For this operator, current source records path dependencies and considers both snapshots; a changed dependency selects the policy subject, then actual context consumers and reverse invalidation dependents. This does not claim minimal impact for other policies, collection changes, unknown inputs, or external observations. |
| 7. Can old exceptions become stale when meaning changes? | The constraint digest binds ordered paths, relation definitions, and consumed reference-property schema; subject evidence includes the subject, traversed resource contents, and semantic trace. This conservative evidence can stale an exception after an unrelated field changes within a traversed resource; minimal staleness is not promised. |
| 8. Does this permit arbitrary queries? | No. Paths are named, statically typed, same-Domain paths of at most two edges. No expressions, wildcards, fan-out, or source-code/filesystem reads are accepted. |
| 9. Which important cases remain unsolved? | UseCase-to-many-Aggregate owner equality, inverse/exclusive ownership, mixed-relation cycle prohibitions, and identity-pair allowlists remain outside this operator. Source behavior such as Query non-mutation also needs explicit checks or adapters. |
| 10. Is a broader mechanism justified? | No. The evidence supports this one bounded relational assertion, not Pattern/Composition, domain inheritance, or a general query engine. |

## Source verification

Local Windows validation on 2026-10-03 used Go 1.27.1 and passed `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `go mod verify`, generated-schema checks, and the one-iteration authoring benchmark fixture gate. Check and canonical formatting passed for twelve consumer/benchmark fixtures, including Software and Delivery. An independent read-only review found no reproduced correctness defect. These are synthetic source checks, not release or business evidence.

The local candidate executable used for fixed-snapshot replay has SHA-256 `79ad0d4ca6ae83c3ac52f8cad3b6505fa853e724a518b71efc8c046e27985766`. After guarded format/render in separate consumer Git repositories, `check`, `model`, `format` and agent `context` passed for Software revision `be9c2e5e788e59f5bc99b366520b541f095e54c2` and Delivery revision `00c69f17cd17e10370cbe7cd617d8d8bce90169e`; both checks reported `provisional: false`. The executable still displays the repository's v0.11.0 development version; this hash identifies a local source candidate, not the published v0.11.0 binary. Historical Software package sources 1.0.0/2.0.0 and the checked-in 1.0.0 archive remain byte-unchanged from PR #53.

Windows/Linux CI is a separate required commit-bound gate, recorded by the pull request checks. No release tag, published executable, distribution pin or package inheritance is introduced by this iteration.
