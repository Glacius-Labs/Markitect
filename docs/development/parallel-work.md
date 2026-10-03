# Parallel work guide

Read [shared contracts](shared-contracts.md), the [audit](parallelizability.md), [risk register](risk-register.md) and your [workstream specification](../workstreams/README.md) after the canonical entry documents. These prepare future assignments; they do not launch product implementation.

The coordinator owns semantic decisions, cross-workstream compatibility, shared-file changes, integration and release readiness. An implementer owns a bounded subsystem, its tests and evidence. Class A adapters should evolve without editing another adapter; Class B adoption/authoring work uses explicit shared handoffs; Class C compiler semantics require joint review. An avoidable central switch is an audit finding, not permission for each agent to refactor it.

## Assignment and execution

Tie the assignment to the [product vision](../vision.md): state which human execution/supervision activity it targets and which canonical decisions remain owner-controlled. Report technical evidence separately from any measured adopter benefit, including new upkeep. Routine implementation may proceed within the assigned authority; genuine architectural ambiguity, shared policy conflict, exception decisions and exceptional risk go to the designated owner. This does not bypass the integration review or mutation gates below.

Each assignment records the full verified baseline commit, objective, owned paths, required contracts, forbidden paths, agreed new files, tests and exit criteria. Use an isolated worktree and `codex/` feature branch. Do not implement in protected `main`, another implementer's checkout or the consumer's parent directory. Verify working directory before writes. Inventory existing edits and preserve them; never clean another agent's work. Two implementers cannot own the same mutable contract, output or test helper. Disjoint research may proceed while a shared contract is unresolved; dependent durable implementation waits for its decision.

Implement only the assigned slice. Report source-selection/privacy/permissions gaps before reading or moving consumer evidence beyond its reviewed scope. Existing-project adoption keeps ADRs, root instructions, tests and provider configurations authoritative until an explicit owner-reviewed cutover. Frequency, AI confidence, a candidate decision field and green CI are not owner approval. Do not use another adapter's output as canonical input.

## Shared-file requests

README, architecture, roadmap, CLI dispatch, format/schema generation, model types, adapter DTOs, package/state/digest semantics and shared helpers stay with the coordinator during a wave. An implementer supplies a small proposed patch or design request rather than editing them opportunistically. Tests local to an owned package/new file are independent; renderer entrypoints/provider configuration and shared ownership helpers are integration hotspots. Coordinator review decides whether a needed edit is legitimate shared semantics or removable coupling.

Shared `go.mod`, `go.sum` and CI workflows are also coordinator-owned integration files. Provider library/tool dependencies require a bounded request with scope, version, licensing and validation effects; parallel implementers must not each change the root dependency contract independently.

Escalation includes the desired engineering statement, two concrete cases if requesting Core expansion, current expression, normalization/check/adapter alternatives, affected consumers, exact input/version behavior, diagnostics, context/impact/digest effects and a finite deterministic test. Do not introduce Pattern/Trait/inheritance, fan-out, arbitrary selectors/queries, provider semantics, source analysis or background execution through an unrelated workstream.

## Integration and evidence

1. Implement and run focused tests for concrete risks, then the normal contribution gates. Keep canonical sources and generated schemas/examples together where legitimately changed.
2. Commit a complete candidate. Open a focused PR stating before/after behavior, owned scope, non-goals, evidence, limitations and any shared-contract request.
3. Run [Windows/Linux CI](../../.github/workflows/ci.yaml) on that exact head; changed head invalidates the earlier integration evidence.
4. Obtain independent complete-diff review. The coordinator checks shared contract compatibility, target ownership, conflicts with parallel candidates and the consumer authority boundary.
5. Integrate one reviewed candidate at a time against the current main. Rebase/adapt later candidates, rerun affected tests and current-head CI. Keep preserved historical evidence immutable.
6. Record merged commit and main gates. A source artifact or successful merge is not a release or human acceptance of adopter policy.

Every completion report states what changed and why, owned scope, what remains outside it, tests and exact SHA, evidence limitations, remaining gaps, integration dependencies and requested Core changes (or none). Mark incomplete evidence explicitly. Never infer tokens from bytes or make productivity, defect, runtime-safety or market claims from fixture success.

Release is a separate owner decision using [Operations](../operations.md) and immutable publication gates. Agents do not publish individual workstream releases, update installed tools, edit release-managed README values or overwrite published assets. v0.12.0 stays current until verified publication establishes another release.
