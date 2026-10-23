# Read-only analysis of policy failures

Decision for the source iteration after PR 57, 2026-10-03. v0.12.0 remains the published release; this design does not publish a release or alter the finite Domain language.

## Problem and existing pipeline

The [real-code pilot](../validation/real-project-adoption-pilot.md) preserves a passing v1 architecture contract and the exact v2 pin with two failed Validator policies. At that point ordinary Context and Impact abort before their useful output. An exception must express a deliberate deviation, not grant access to analysis.

`app.Parse` already resolves a single graph, evaluates its constraints and exceptions, and retains PolicyResults. `CompileModel` serializes that same graph. CLI dispatch and `CompileContext` currently reject the combined diagnostic list; `runImpact` additionally rejects any diagnosed base. Reuse these values and compilers. There is no alternate parser, graph, policy evaluator or policy model.

## Four states

| State | Structural status | Policy status | Existing validation status | Analysis |
|---|---|---|---|---|
| Invalid schema, reference, relation, cycle, traversal, digest, input or exception metadata | failed | unknown (partial results are not a validity claim) | failed | blocked |
| Structurally valid, all applicable policies pass | passed | passed | passed | available |
| Structurally valid, at least one ordinary PolicyResult fails | passed | failed | failed | explicit read-only mode only |
| Structurally valid, valid explicit waiver(s), no unwaived failure | passed | waived | passed | available; deviations stay visible |

An empty applicable policy set passes. Mixed waived/failed results remain policy-failing. Structural status is about a trustworthy declared graph, not business correctness or implementation conformance. Model retains its existing `validationStatus` and results, adding the separate status summaries rather than changing the meaning of `failed`.

Ordinary failure diagnostics receive an explicit `policyResult` identity: API version, constraint and subject (empty for a collection result). Only a diagnostic generated for and matching an actual failed result is a policy failure. Every other diagnostic remains structural. Do not infer this from names or code prefixes: a constraint may be named `path`, while `constraint.path` also reports an invalid same-target traversal. Schema/relation bounds and exception failures remain unwaivable. Evaluation, digests, waiver eligibility and dates do not change.

## Explicit CLI contract

Add `--analyze-policy-failures` only to resource `context` and fixed-revision `impact`. Their strict defaults remain unchanged. Reject the option for other commands and for `context --run`; fixed-run authoring/advisory evidence is outside this source slice.

The alternatives were changing default Context/Impact success (ambiguous and incompatible), a new analysis command (duplicates selection/comparison interfaces), or a generic allow-invalid flag (too broad). The longer option names the exact permitted condition and fits the existing per-command allowlist.

In this mode:

- Load the ordinary Project once per snapshot. Block if any structural diagnostic remains, including malformed or stale exceptions.
- Emit a visibly marked `analysis` record with mode, completion, snapshot/config/model identities, structural/policy/validation status and failed-result count. It states that analysis is not verification or acceptance.
- Exit **1** when any analyzed snapshot has failed policies, even though useful analysis YAML was produced. An Impact base that fails and candidate that passes still returns 1; per-side status identifies the repaired candidate.
- Exit **0** only when all analyzed inputs have no unwaived failure and analysis completed. Exit **2** remains an invocation/acquisition/compilation/incomplete-output error. Structural diagnostic reports retain their existing failure exit; no Context/Impact analysis result is emitted for them.

Consumers must retain stdout on exit 1. No successful analysis proves repository gates, human acceptance or semantic truth. Check and Verify continue to reject ordinary failed policies and cannot use this option. Ordinary passing Context/Impact selection and conservative invalidation retain their existing semantics.

## Context

An explicit `AnalyzeContext` entrypoint uses the existing context compiler with a narrowly different validation gate. Follow only existing context edges, preserve `via`, package/Domain inputs, declared files and collection-result behavior. Include only normal closure PolicyResults, using the exact normalized result values. A globally failing candidate may have no failed result in an unrelated selected closure; the analysis header still reports the candidate's overall failed status/count. Do not pull unrelated resources or documentation into it.

Bind the analysis marker/status and normalized model identity into the diagnostic context digest. Diagnostic context must not be confused with ordinary passing context or reused as acceptance evidence. Do not turn same-target policy dependencies into Context edges.

## Impact and policy changes

An explicit `AnalyzeImpact` guards both snapshots, then uses the current `Changes` algorithm. Keep `changed`, conservative `affected`, reasons and causes; do not narrow a package/configuration/inventory fallback merely because two policies fail.

Extend the same Impact DTO with sorted `policyChanges`, `directPolicySubjects` and counts. Compare normalized PolicyResults by `(apiVersion, constraint, subject)`, preserving package-qualified subject identities. Compare their semantic outcomes and constraint/subject/exception digests and decision values. A package version/source relocation alone does not make every unchanged passing policy a direct result change; the existing global impact still explains that update. Both-side provenance is retained for changes.

Each change carries the original result (including failure message, waiver and equality traces), exact Domain input and package pin, and the relevant normalized constraint selector/assertion. These are compact policy descriptors, not a copy of the whole model. Bind both snapshots to the same `CompileModel` identities used by Model. Do not reevaluate policy inside Impact.

A missing side is **not a passed result**. Represent it as analysis-only `not-applicable` with an absence reason: constraint not defined, subject absent, or no result selected. This is not a new PolicyResult status. It explains the pilot's new v2 constraint; selector entry/exit and removal also remain distinct through their reason and source. Support passed/failed/waived transitions and transitions to/from a missing result. A collection result has no direct resource subject; keep the delta without inventing one.

When a constraint retains its identity but changes between resource and collection evaluation, its old result identities are no longer applicable. Report `result-scope-changed` on that absent side and retain the new result identities separately. This explains an existing finite assertion's output scope; it does not reevaluate policy or invent a resource subject for a collection.

`directPolicySubjects` contains only subjects of changed result records, not all failed subjects, invalidation consumers, or the complete true implementation-review set. Status-preserving subject/constraint digest changes also count. Direct deltas are evidence of evaluation changes; `affected` remains the conservative review set. New counts summarize those separate existing meanings.

## Mutation and reconciliation boundaries

No flag is accepted by render, format, review, verify, check or reconciliation. Keep CLI rejection of policy-failing Observe/Plan/Apply/Verify. Although read-only projection planning might be useful later, authorizing projections from a noncompliant model is a separate decision; external adapter planning continues to require passing validation. Apply keeps explicit `--write`, reviewed input binding, branch, target and stale-plan checks. No automatic exception, projection write or application-code migration is introduced.

## Verification and historical evidence

Test classification by identity, including a policy named `path`, alongside unresolved/wrong-kind references, malformed Domain/resource, cardinality, cycles, invalid same-target paths and stale/invalid exceptions. Test strict Check/Verify/write boundaries, passing defaults, waived results, scoped context/via/inputs, unchanged policy-only traversal, exact normalized model/result joins, all status transitions, semantic-versus-origin-only deltas, sorted deterministic YAML and conservative causes.

Replay the preserved bundle in a new `experiments/policy-failure-analysis/` result area. Preserve the entire earlier pilot tree. At its exact failing v2 revision obtain Model, diagnostic Context and v1-to-v2 Impact without a waiver, canonical/code edits or projections. Record the two direct failure subjects separately from the actual conservative set, then replay the original first Validator, optional explicit waiver and final passing revisions. Normal Check must still fail at the failing pin. This tests inspectability, not productivity, safety or token use.

Separate follow-ups remain: matched AGENTS.md/architecture-test validation, ADR/source selection, authoring feedback, duplicated ownership mappings and context breadth. No new operator, expanded same-target, graph query, fan-out, inverse ownership, selector language, Pattern, Trait, composition or inheritance is justified here.

## Release decision

This is release-worthy as an explicit CLI and normalized-output contract. Integration and source quality gates do not publish it. Prepare a separate immutable release after integration, with its own exact source commit, Windows/Linux release gates and verified public assets; do not change published v0.12.0 artifacts or release-managed distribution metadata in this iteration. The replay establishes inspectability at fixed snapshots, not product-benefit claims for a release announcement.
