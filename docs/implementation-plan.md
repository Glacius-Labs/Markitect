# Markitect roadmap

Updated 2026-09-30. This is the product's own roadmap. Source version `0.2.0` defines the standalone configuration model below. Available distributions and their exact source commits are listed in [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases). [Architecture](architecture.md) describes the implemented product boundaries, [Usage](usage.md) records the project and upgrade contract, and [Operations](operations.md) owns release gates.

## v0.1.0 — historical release

The initial release established a standalone Go/YAML tool, typed resources, explicit dependencies, fixed Git snapshots, context and impact queries, core authoring, generic rendering, and verified private release assets. Its product-source checks and release evidence are summarized in [the v0.1.0 assessment](production-assessment.md). The old Project profile and adopter-specific migration/rendering compatibility must not be carried forward as general product policy.

## v0.2.0 — standalone verification and rendering

The v0.2.0 model makes repository behavior explicit and removes adopter-specific assumptions.

1. **Explicit Project checks.** Remove `Project.spec.profile`. Add optional `Project.spec.checks` entries containing a stable name and `run` argv. The first argv element is a bare executable resolved from `PATH`; arguments are literal and no shell is inserted. `verify` runs the commands from its fixed snapshot, with existing time/output bounds. No declared checks means `incomplete-evidence`, not success.
2. **Explicit outputs.** Keep generic managed views. Render other outputs only through Project-declared `targets` and `ruleAdapters`. Do not load a compatibility renderer implicitly.
3. **Remove migration policy from core.** Remove the built-in migration command and hard-coded conversion adapters. A project that needs to import existing content owns a script and its validation in its own repository.
4. **Keep authoring in core.** Continue shipping portable authoring resources and structural queries with Markitect. They explain the product's resource model without deciding task semantics or requiring a model API.
5. **Prove the neutral boundary.** Test schemas, checks, rendering, fixed snapshots, and the executable example in standalone Windows and Linux CI. Ensure there is no bootstrap, Python, or adopting-project gate inferred from files or repository identity.
6. **Document the v0.1 transition.** Describe removal of the Project profile and explicit checks/outputs using neutral examples. Treat v0.1.0 as an immutable historical release; publish v0.2.0 only after the exact candidate meets release gates.

Acceptance: a project can author and structurally check resources without repository-specific setup; `verify` reports incomplete evidence when checks are absent and executes only declared argv on the selected fixed commit; generic views render without a provider; extra outputs require explicit configuration; core authoring still compiles; all release gates identify one source commit and platform.

## Later product options

- Reusable, versioned content packages with exact identities, exports, offline inputs, and impact/review integration.
- One-time project initialization after package composition is proven.
- MCP, LSP, and graph visualization when measured authoring needs justify them.
- Runtime/operator integration only for a concrete state-reconciliation contract.

These options are not v0.2.0 acceptance promises. Keep the core provider-independent and add abstractions only for a demonstrated invariant.
