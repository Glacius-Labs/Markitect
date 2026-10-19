# Parallel work guide

The [delivery plan](../implementation-plan.md) owns current scopes, dependencies, path reservations and integration order. The [working method](working-method.md) governs design learning and owner orientation. This guide owns execution inside a package. Candidate progress and checks live in its chat/PR; there is no separate backlog/dashboard.

## Assignment and execution

Use an isolated worktree and the integrated base named by the coordinator. Read [AGENTS.md](../../AGENTS.md), the plan, technical contract and [CONTRIBUTING](../../CONTRIBUTING.md). Verify checkout, branch, full SHA and dirty state. Preserve other work; never stash or reset another checkout.

One chat owns one bounded package/API slice and tests. Dispatch lists paths, inputs, acceptance and dependency gates. Subagents are permitted within scope. Unresolved shared semantics block dependent implementation; independent work can proceed. No model/provider claim replaces actual verification.

## Shared-file requests

The coordinator owns root dependencies and the plan; the [architect section](../implementation-plan.md#architect-and-coordinator) assigns the Apply concept, its register entries, the ports and the core packages to the architect session, and everything else stays with the coordinator. BOOT/CUTOVER may receive explicit bounded ownership of gates, canonical sources, generated views and wiring. Others request a concrete change with intended behavior, consumer and cases rather than editing shared files.

Producers own value types; consumers own ports. No global types/interface package, old Project DTO bridge or duplicate definitions to unblock an adapter. Follow the [target Go boundaries](../history/apply-design-before-dec-056/manager-apply-go-architecture.md), not obsolete Module/Host rules in historical guides. Until cutover, current source retains its gates and the public CLI does not import partial replacement packages.

Two chats cannot own the same contract, generated output or helper. Tests/fixtures live with their package. External dependencies need coordinator agreement and an exact pin; root module changes have one writer. New paths require project-model coverage before integration.

## Integration and evidence

1. Run focused checks and inspect actual changed files. Preserve failures/unknowns with exact source identity.
2. Open a focused PR with package ID, scope, contracts, base/head and validation. Distinguish implemented behavior from scaffolding/proposed APIs.
3. Obtain independent complete-diff review and run relevant local checks. Freeze the reviewed head. During development, full hosted CI is manual and is required only when the assignment explicitly requests it.
4. Follow the [delivery plan's integration rules](../implementation-plan.md#integration): the delivery merges its approved exact head after the relevant checks pass; recompose for conflicts or substantive overlap and renew relevant review/validation. The coordinator owns dependencies and shared reservations.
5. Regenerate views from sources. Retire obsolete tests/gates with their feature; never exempt a required invariant for green CI.

A source package is not a real Agent run, successful library Apply, release or human acceptance. Provider studies and delivery need separate scope and finite authority.
