# Markitect roadmap

Updated 2026-09-30. This is the product's own roadmap. Source version `0.3.0` defines the current content-package model. Available distributions and their exact source commits are listed in [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases). [Architecture](architecture.md) describes the implemented product boundaries, [Usage](usage.md) records the project and upgrade contract, and [Operations](operations.md) owns release gates.

## v0.1.0 — historical release

The initial release established a standalone Go/YAML tool, typed resources, explicit dependencies, fixed Git snapshots, context and impact queries, core authoring, generic rendering, and verified private release assets. Its product-source checks and release evidence are summarized in [the v0.1.0 assessment](production-assessment.md). The old Project profile and adopter-specific migration/rendering compatibility must not be carried forward as general product policy.

## v0.2.0 — standalone verification and rendering

The v0.2.0 model makes repository behavior explicit and removes adopter-specific assumptions.

1. **Explicit Project checks.** Remove `Project.spec.profile`. Add optional `Project.spec.checks` entries containing a stable name and `run` argv. The first argv element is a bare executable resolved from `PATH`; arguments are literal and no shell is inserted. `verify` runs the commands from its fixed snapshot, with existing time/output bounds. No declared checks means `incomplete-evidence`, not success.
2. **Explicit outputs.** Keep generic managed views. Render other outputs only through Project-declared `targets` and `ruleAdapters`. Do not load a compatibility renderer implicitly.
3. **Remove migration policy from core.** Remove the built-in migration command and hard-coded conversion adapters. A project that needs to import existing content owns a script and its validation in its own repository.
4. **Keep authoring in core.** Continue shipping portable authoring resources and structural queries with Markitect. They explain the product's resource model without deciding task semantics or requiring a model API.
5. **Prove the neutral boundary.** Test schemas, checks, rendering, fixed snapshots, and the executable example in standalone Windows and Linux CI. Ensure there is no bootstrap, Python, or adopting-project gate inferred from files or repository identity.
6. **Document the v0.1 transition.** Describe removal of the Project profile and explicit checks/outputs using neutral examples. Treat v0.1.0 as an immutable historical release; verify the exact source commit, gates, and assets before publication. The confirmed v0.2.0 release evidence is in the [production assessment](production-assessment.md).

Acceptance: a project can author and structurally check resources without repository-specific setup; `verify` reports incomplete evidence when checks are absent and executes only declared argv on the selected fixed commit; generic views render without a provider; extra outputs require explicit configuration; core authoring still compiles; all release gates identify one source commit and platform.

## v0.3.0 — direct offline content packages

The current source adds exact direct package pins in `Project.spec.packages`, deterministic archives from fixed Git revisions, explicit exports, package-qualified graph identities, and package inputs in compiled context and impact/review eligibility. No second content lock or package resolver is introduced; `markitect.lock.yaml` remains the CLI distribution lock. Imported Rules are not activated implicitly, package resources remain read-only, and package manifests do not declare checks, targets, or rule adapters. See [Content packages](content-packages.md) for the contract and the [production assessment](production-assessment.md) for release status.

## Later product options

- One-time project initialization after package composition is proven.
- MCP, LSP, and graph visualization when measured authoring needs justify them.
- Runtime/operator integration only for a concrete state-reconciliation contract.

These options are not v0.3.0 acceptance promises. Keep the core provider-independent and add abstractions only for a demonstrated invariant.
