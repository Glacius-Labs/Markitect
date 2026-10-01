# Markitect roadmap

Updated 2026-10-01. This is the canonical owner of current source and planned-work status. The published v0.5.0 release adds explicit provider adapters and opt-in sourced consistency checks to the earlier product behavior. Dated release evidence is in the [production assessment](production-assessment.md), and available distributions are listed in [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases). [Architecture](architecture.md) describes product boundaries, [Usage](usage.md) records the CLI contract, and [Operations](operations.md) owns release gates.

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

This source version added exact direct package pins in `Project.spec.packages`, deterministic archives from fixed Git revisions, explicit exports, package-qualified graph identities, and package inputs in compiled context and impact/review eligibility. No second content lock or package resolver was introduced; `markitect.lock.yaml` remains the CLI distribution lock. Imported Rules are not activated implicitly, package resources remain read-only, and package manifests do not declare checks, targets, or rule adapters. See [Content packages](content-packages.md) for the contract.

## v0.3.1 — binding-cycle correction

This correction rejects binding cycles that include a Contract selected by its own implementation. The dated source and release evidence is in the [production assessment](production-assessment.md).

## v0.4.0 — minimal project initialization

The v0.4.0 release adds `markitect init` for an existing repository. Preview prints the exact minimal Project YAML and area README plan without writing, including outside Git. `--write` recomputes and validates the plan, requires a named non-protected Git branch, and creates only `markitect.yaml` and one README in a previously absent area directory. It does not infer project policy, add resources or checks, select outputs or packages, use custom templates, or edit root documentation or agent instructions. File creation is exclusive and guarded by the shared write lock; a partial failure reports created paths and recovery guidance, because two file creations are not a transaction.

Acceptance is structural: the new Project parses, resolves, and passes output checks. `verify` must remain incomplete until the adopting project owner declares actual checks and commits the candidate. Consult GitHub Releases for currently available distributions.

This source version also adds an executable code-to-documentation example, fixes shared-writer path handling for Windows short/long names while retaining reparse rejection, keeps benchmark fixtures checked in CI, and bundles third-party notices accessible offline through `licenses`. The [authoring pilot](authoring-pilot.md) records actual measurements and the incomplete model comparison; it makes no savings claim.

## v0.4.1 — licensing and repository presentation

The published v0.4.1 release adds the Apache License 2.0 to the product repository and includes it in source distributions. Earlier immutable tags and their attached archives do not gain a license file retroactively. It also improves the README entry, support and security guidance, and contribution templates. Its product behavior and verification boundaries remain those of v0.4.0. The reviewed source, Windows/Linux gates, exact tag, and immutable release verification are recorded in the [production assessment](production-assessment.md).

## v0.5.0 — explicit adapters, measurements, and assertions

The published v0.5.0 release adds project-configured Codex, Claude, and shared entrypoints with strict inventory; see [Provider adapters](provider-adapters.md). It adds a post-publication Windows/Linux benchmark workflow that measures both the new and prior immutable release on fixture v1, with raw results and a summary as workflow artifacts; see [Measurement](measurement.md). The opt-in [consistency MVP](consistency.md) compares explicit, quoted functional assertions and reports source and owner evidence. It does not interpret free text or change human acceptance.

The tagged source passed Windows/Linux release gates, and the immutable release and four attached assets were verified. The release-triggered benchmark failed during runner setup because an action pin was truncated; the separately dispatched repaired run completed on Windows and Linux with raw measurement artifacts. Those measurements are diagnostic, not an acceptance gate or evidence of a speed gain. Konfyra's consumer pin, provider adapter ownership, candidate verification and human acceptance are tracked in its own repository. The Survey pilot can move only after its exact candidate basis and existing work are reconciled.

## Unreleased maintainer tooling

The source now includes `markitect-release distribution` to check or regenerate
marked README installation sections and versioned WinGet manifests from a
verified published immutable release. It validates the exact provenance workflow
attempt, tag, asset bytes, and attestations before generating local files.
[Operations](operations.md) owns the commands and submission checks. This does
not change the published v0.5.0 binary or publish a WinGet catalog entry.

## Later product options

- MCP, LSP, and graph visualization when measured authoring needs justify them.
- Runtime/operator integration only for a concrete state-reconciliation contract.

These options are not v0.5.0 acceptance promises. Keep the core provider-independent and add abstractions only for a demonstrated invariant.
