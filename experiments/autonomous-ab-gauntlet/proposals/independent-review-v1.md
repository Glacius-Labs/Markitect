# Forward/Backward Gate Proposal — Source-Only Design Review

Review date: 2026-10-05
Reviewed baseline: `d16ee50d138ec3120245ce38628f110ef41bb805`
Scope: supplied `markitect-coordinator-forward-plan-backward-gate.md` and current product source/docs only. This note is design critique, not authorization to change product behavior. No arena, holdout, or outcome contents were inspected.

## Assessment

The proposal fits Markitect's existing separation of responsibilities if treated as workflow orchestration over evidence already produced by the fixed-snapshot compiler, checks, Impact, artifact accounting, and reconciliation. The reusable idea is a bounded, task-specific, input-bound Change Plan followed by a compositional completion report. It does not establish a need for a Core or Domain primitive. First test whether the existing canonical workflow and a small Host/CI wrapper can use current outputs without introducing a persisted plan format.

There is a real usability gap worth validating: current workflow instructions require a fixed BASE, intent-first changes, selected context and impact, final candidate checks, verification and accounting, but they do not themselves create a machine-checkable record of the planned task scope that a later candidate gate compares against. That is a Host/orchestration or adopter-owned workflow gap. The existing `.artifacts/markitect/reconcile/` plan is specifically a renderer plan and cannot stand in for a task plan.

## Reusable mechanisms and boundaries

- `check`, `model`, `context`, `impact`, `verify`, and policy-failure analysis already separate structural validity, policy status, useful context, conservative invalidation, and configured implementation evidence. Reuse their outputs; do not duplicate evaluation or create a second policy model.
- `markitect-artifacts.yaml` plus the `artifactcoverage` module already classifies exact paths under declared managed roots as canonical, input, generated, tooling, or excluded and reports unmanaged/colliding ownership. It intentionally says nothing about paths outside those roots and does not establish semantic correctness. A task gate should use this scoped accounting, not describe it as a complete repository inventory.
- Reconciliation already binds plans to source/config/model/desired state, adapter/tool identity, and observed destination preconditions; it recomputes before Apply, requires an explicit write boundary, and verifies the planned outputs. This is a strong precedent for stale-plan rejection, but its plan authorizes a narrowly defined projection operation, not engineering intent or task acceptance.
- Repository import ownership is checked by project tooling (`internal/tooling/architecture`); source-level truth remains project-owned evidence, not Core semantics. Keep any changed-path/source checks as configured checks or specialist adapters.
- Existing workflow and constitution already say implementation-only work must not fabricate model edits, human-owned intent precedes implementation for intent changes, fixed inputs matter, machine evidence is not human acceptance, and unsupported/unknown evidence remains visible.

## What an input-bound digest proves

A cryptographic digest can show that the bytes of a particular plan, BASE, canonical model/configuration, declared checks, tool versions, and selected evidence match the values used by a later evaluator. Recomputing those values can detect staleness or tampering relative to the supplied identities. A commit ID and parent relation can establish repository ancestry and the content recorded in those commits.

A digest by itself cannot prove that a plan was authored before implementation, that a reviewer saw or approved it, that the reviewer had authority, that the classification was correct, or that the plan was not rewritten after the work and then hashed. A timestamp embedded in the plan is only a claim unless backed by a trusted append-only CI/review record or signature. Git ancestry helps establish ordering only when the accepted plan/intent is itself committed or otherwise recorded by a trusted system before the candidate; a loose local plan file is not chronology or approval evidence. Existing exception owner/decision fields likewise record a decision claim but do not authenticate it.

Keep authority as a distinct input/evidence dimension. If the first increment has no trusted approval mechanism, label the plan as a reviewed declaration supplied to the gate; do not call its digest “approval.”

## Plan identity and scope

Do not bind one digest indiscriminately to the whole BASE and then expect it to remain stable after intended edits. Record at least two identities: (1) the task's starting BASE snapshot, and (2) the accepted desired-state/model identity after any intent change. The implementation candidate must be explicitly related to the latter (for example, a declared parent/commit chain or a reviewed intent-candidate identity). For implementation-only work, the canonical desired-state identity should remain unchanged even though source snapshot identity advances.

Represent scope as declarations and evidence, not as a promise that Markitect understands all code changes. A plan can list selected canonical subjects, relevant Impact and its causes, declared managed artifact paths or bounded path categories, configured checks, and explicit owner-decision items. Impact's `affected` set is conservative support for review, not an exact set of source files that should change. Do not turn it into an allowlist or imply that omitted implementation paths are unrelated. Distinguish direct policy-result subjects from conservative affected resources, and preserve unknown/unmanaged findings rather than hiding broad impact.

Classification can route the workflow, but cannot be trusted as semantic truth merely because an agent chose `implementation-only`. Inputs that can alter the required gates (classification, scope, exceptions, check set, selected resources) should be bound and visible. Ambiguity or missing evidence should yield incomplete/owner decision, not a silent pass. Avoid arbitrary reasoning text in gate identity; it is neither deterministic evidence nor authority.

## Generic semantics versus adopter checks

“Desired model matches implementation,” “a UseCase has a source file,” “a Query does not mutate,” and “the observed architecture is complete” cannot be established by generic path accounting or hashes. They require a project-owned test/adapter with a stated proof boundary. The existing rule is sound: canonical intent plus explicit, typed adapter/check evidence yields a bounded verification result. The generic completion gate may require a configured check and report its exact snapshot/tool/result, but must not reinterpret source code or claim more than that check proves.

“Unplanned changed file” is also not universally a semantic violation: generated/tooling files, lockfiles, tests, and ordinary implementation details can legitimately change without a canonical resource edit. Enforce only a project-declared managed surface and explicit task scope. Everything else remains reported as outside that accounting contract or unverified, not presumed acceptable or invalid.

## Smallest useful path and falsification

1. Keep product development frozen until the global fix/no-fix decision. Do not use this design assessment to inspect or reinterpret any reserved outcome.
2. For the next authorized increment, first replay the existing canonical workflow with a small adopter-owned wrapper that consumes fixed BASE/candidate plus current `check`, `impact`, configured `verify`, and artifact-coverage results. No Domain kind, Core semantic, or persisted task resource is needed to test whether composition is currently missing.
3. If the same omission recurs, specify a Host-level Change Plan contract: versioned, deterministic fields for task/change identity; start snapshot; desired-state identity; declared change class; selected subjects and Impact identity; managed path scope; required checks/projections; explicit owner decisions; and evidence references. Define stale-plan and incomplete/unknown behavior. Keep human approval proof outside this object unless a trusted identity/signature source is integrated.
4. Test separately: implementation-only task, intent-first migration, policy-failing but structurally valid candidate, unmanaged changed artifact, stale plan, missing check evidence, and a plan rewritten after implementation. Verify strict acceptance and Apply behavior remain unchanged. Record replan/false-positive overhead as well as omissions.

Falsify the need for a new product mechanism if the existing workflow plus one short CI command reliably collects the same evidence and makes omissions obvious. Also reject the design if the plan mostly duplicates task issue text, requires hand-maintaining a broad list of implementation files, labels conservative Impact as exact scope, frequently invalidates on harmless edits, or provides no trustworthy ordering/authority beyond its own self-asserted fields. Compare it against a short `AGENTS.md` plus existing architecture tests before proposing a persistent Host feature.

## Conclusion

The forward-plan/backward-gate direction is conceptually compatible as a possible workflow layer, not as a new Core language feature. There is a plausible bounded gap—no general task-scope record tying intended scope to final evidence—but it is not yet proven that a new plan artifact is better than composing existing fixed-snapshot commands in CI. First test that smallest path and falsify against the simpler workflow. No broader policy, source semantics, autonomous approval, or product-value claim follows from this design alone.
