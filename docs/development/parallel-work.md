# Parallel work guide

The [roadmap](../implementation-plan.md#how-we-work) owns how parallel work is organized: sessions, work packages, [branch names](../implementation-plan.md#branch-names), [zones](../implementation-plan.md#zones) and the [integration rules](../implementation-plan.md#integration). The [backlog](../work-items/backlog.yaml) owns each package's status. This guide adds the engineering rules that apply inside a package. Consult [shared contracts](shared-contracts.md) for relevant version-bound seams.

The coordinator (the roadmap's integrator) owns semantic decisions, cross-package compatibility, shared-file changes, integration and release readiness. A session owns its package's bounded scope inside its zone, its tests and its evidence. Adapters should evolve without editing another adapter; adoption and authoring work uses explicit shared handoffs; compiler semantics require joint review. An avoidable central switch is an audit finding, not permission for each agent to refactor it.

## Assignment and execution

Tie the assignment to the [product vision](../vision.md): state which human execution/supervision activity it targets and which canonical decisions remain owner-controlled. Report technical evidence separately from any measured adopter benefit, including new upkeep. Routine implementation may proceed within the assigned authority; genuine architectural ambiguity, shared policy conflict, exception decisions and exceptional risk go to the designated owner. This does not bypass the integration review or mutation gates below.

Each package records its objective, zone, dependencies and acceptance criteria in the backlog; its pull request records the full verified base commit, owned paths, required contracts, agreed new files, tests and evidence. Use an isolated worktree and a branch from current `origin/main` named as the roadmap's [branch names](../implementation-plan.md#branch-names) require. Do not implement in protected `main`, another session's checkout or the consumer's parent directory. Verify working directory before writes. Inventory existing edits and preserve them; never clean another agent's work. Two sessions cannot own the same mutable contract, output or test helper. Disjoint research may proceed while a shared contract is unresolved; dependent durable implementation waits for its decision.

Implement only the assigned slice. Report source-selection/privacy/permissions gaps before reading or moving consumer evidence beyond its reviewed scope. Existing-project adoption keeps ADRs, root instructions, tests and provider configurations authoritative until an explicit owner-reviewed cutover. Frequency, AI confidence, a candidate decision field and green CI are not owner approval. Do not use another adapter's output as canonical input.

## Module assignments

A Module assignment owns one `src/internal/modules/<name>` subtree, including its private tests and bounded design/evidence files. It may consume Core IR and explicit Host-supplied configuration or artifact bytes, but must not edit Core, Host, Infrastructure, Tooling, a sibling Module or shared wiring. Host composition, shared DTOs, `go.mod`, CI workflows, and the import gate remain coordinator-owned. The [Module guide](modules.md) defines when to add or remove a Module and the exact static dependency law.

A request to change Core must describe a generic invariant with concrete cases from at least two distinct vocabularies, the current expression and its failure, a finite normalization/check/adapter alternative, consumers, exact input/version behavior, diagnostics, digest/context/impact consequences, and deterministic tests. Submit the request to the coordinator before editing Core. A Module is never its own authority to add a Core primitive.

## Shared-file requests

README, architecture, roadmap, CLI dispatch, format/schema generation, model types, adapter DTOs, package/state/digest semantics and shared helpers stay with the coordinator or with the zone that holds them. An implementer supplies a small proposed patch or design request rather than editing them opportunistically. Tests local to an owned package/new file are independent; renderer entrypoints/provider configuration and shared ownership helpers are integration hotspots. Coordinator review decides whether a needed edit is legitimate shared semantics or removable coupling.

Shared `go.mod`, `go.sum` and CI workflows are also integration files; CI workflows belong to the `ci` zone. Provider library/tool dependencies require a bounded request with scope, version, licensing and validation effects; parallel implementers must not each change the root dependency contract independently.

Escalation includes the desired engineering statement, two concrete cases if requesting Core expansion, current expression, normalization/check/adapter alternatives, affected consumers, exact input/version behavior, diagnostics, context/impact/digest effects and a finite deterministic test. Do not introduce Pattern/Trait/inheritance, fan-out, arbitrary selectors/queries, provider semantics, source analysis or background execution through an unrelated package.

## Integration and evidence

1. Implement and run focused Module tests and compatibility cases, then the normal contribution gates. The architecture import gate at `src/internal/tooling/architecture` statically checks supported-platform files and tests and exercises negative dependency fixtures. It is included in `go test ./...`, a named CI step, the explicit Project check and the release quality job through reusable CI. These routes do not imply an exact-head pass; the coordinator reports evidence separately.
2. Commit a complete candidate. Open one focused PR per package, with the package ID at the start of its title, stating before/after behavior, owned scope, non-goals, evidence, limitations and any shared-contract request.
3. Run [CI](../../.github/workflows/ci.yaml) on that exact head. Linux is the required gate ([DEC-013](../concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)). Do not push to the head while its required Linux run is in progress; fixes may be pushed while only the Windows job is still running. A changed head invalidates the earlier integration evidence.
4. Obtain independent complete-diff review. The coordinator checks shared contract compatibility, target ownership, conflicts with parallel candidates and the consumer authority boundary.
5. The integrator merges one reviewed candidate at a time, in dependency order, against the current main. Rebase/adapt later candidates, rerun affected tests and current-head CI. Keep preserved historical evidence immutable.
6. Record merged commit and main gates. A source artifact or successful merge is not a release or human acceptance of adopter policy.

Every completion report states what changed and why, owned scope, what remains outside it, tests and exact SHA, evidence limitations, remaining gaps, integration dependencies and requested Core changes (or none). Mark incomplete evidence explicitly. Never infer tokens from bytes or make productivity, defect, runtime-safety or market claims from fixture success.

Release is a separate owner decision using [Operations](../operations.md) and immutable publication gates. Agents do not publish individual package releases, update installed tools, edit release-managed README values or overwrite published assets. Published releases remain immutable; source changes do not alter them or claim a new release.
