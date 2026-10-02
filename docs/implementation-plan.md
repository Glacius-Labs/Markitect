# Markitect roadmap

Updated 2026-10-02. This is the canonical owner of current source and planned-work status. The published v0.9.0 release separates optional Markdown views, direct provider outputs and consumer tool pins. It retains explicit Area ownership, the omitted-path initialization default and the resolved snapshot boundary from earlier releases. Dated release evidence is in the [production assessment](production-assessment.md), and available distributions are listed in [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases). [Architecture](architecture.md) describes product boundaries, [Usage](usage.md) records the CLI contract, and [Operations](operations.md) owns release gates.

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

This source version also adds an executable project-artifact input example using a Go file, fixes shared-writer path handling for Windows short/long names while retaining reparse rejection, keeps benchmark fixtures checked in CI, and bundles third-party notices accessible offline through `licenses`. The example exercises explicit context and impact, not source-code analysis; the [architecture boundary](architecture.md#project-artifact-boundary) applies to all artifact types. The [authoring pilot](authoring-pilot.md) records actual measurements and the incomplete model comparison; it makes no savings claim.

## v0.4.1 — licensing and repository presentation

The published v0.4.1 release adds the Apache License 2.0 to the product repository and includes it in source distributions. Earlier immutable tags and their attached archives do not gain a license file retroactively. It also improves the README entry, support and security guidance, and contribution templates. Its product behavior and verification boundaries remain those of v0.4.0. The reviewed source, Windows/Linux gates, exact tag, and immutable release verification are recorded in the [production assessment](production-assessment.md).

## v0.5.0 — explicit adapters, measurements, and assertions

The published v0.5.0 release adds project-configured Codex, Claude, and shared entrypoints with strict inventory; see [Provider adapters](provider-adapters.md). It adds a post-publication Windows/Linux benchmark workflow that measures both the new and prior immutable release on fixture v1, with raw results and a summary as workflow artifacts; see [Measurement](measurement.md). The opt-in [consistency MVP](consistency.md) compares explicit, quoted functional assertions and reports source and owner evidence. It does not interpret free text or change human acceptance.

The tagged source passed Windows/Linux release gates, and the immutable release and four attached assets were verified. The release-triggered benchmark failed during runner setup because an action pin was truncated; the separately dispatched repaired run completed on Windows and Linux with raw measurement artifacts. Those measurements are diagnostic, not an acceptance gate or evidence of a speed gain. Consumer pins, project-specific adapter ownership, candidate verification and human acceptance belong to adopting repositories.

## v0.6.0 — documentation placement and optional routers

The published v0.6.0 release adds portable authoring guidance for locating an existing canonical documentation owner and an opt-in `Project.spec.documentation.roots` router check. An adopting project's documentation taxonomy, artifact-to-document relationships, and provider-adapter gates remain project-owned. [Documentation routers](documentation-routers.md) owns the exact participation, link-normalization and diagnostic contract.

Implementation sequence: (1) define and test participating directories and local link normalization against snapshot paths; (2) add strict Project configuration and regenerate its schema; (3) surface stable router diagnostics through `check` and fixed-revision `verify`; (4) update embedded authoring guidance and validate a placement task where an agent finds and changes an existing canonical source; (5) run source and executable-example gates. The acceptance case must distinguish the agent's actual edit from the structural router check. A passing `check` cannot prove semantic placement or complete repository verification.

Focused tests cover invalid and overlapping roots (`internal/format/yaml_documentation_test.go`); participating directories, direct-child coverage, extra cross-links, missing and unsafe targets, reference links, code examples and normalization (`internal/app/documentation_routers_test.go`); fixed-snapshot `check` and broken-router `verify` (`cmd/markitect/documentation_routers_test.go`); and the executable placement fixture (`examples/example_test.go`). These tests assert structural behavior. The separate agent exercise below probes the authoring decision.

One isolated authoring exercise used a copy of the checked-in [placement example](../examples/documentation-placement/README.md) and the prompt recorded there. The agent changed only `docs/engineering/persistence.md`, the existing page whose README claims schema migration requirements; it created no second page. The router check passed on that candidate. This is one observed placement decision, not a measured reliability rate or proof that the prose is correct.

This slice introduces no Resource Kind, documentation graph, README generation, excludes, or additional Project options. Router navigation and explicit dependency/impact analysis remain separate.

The v0.6.0 source, tag, Windows/Linux release gates, attested assets and post-publication benchmark are recorded in the [production assessment](production-assessment.md).

## Published maintainer tooling (v0.7.0)

The published v0.7.0 release includes `markitect-release distribution` to check or regenerate
marked README installation sections and versioned WinGet manifests from a
verified published immutable release. It validates the exact provenance workflow
attempt, tag, asset bytes, and attestations before generating local files.
[Operations](operations.md) owns the commands and submission checks. This does
not publish a WinGet catalog entry.

## v0.7.0 — fixed task and selected artifact context

The published v0.7.0 release adds a fixed-run context request for one explicitly selected resource, committed work-item snapshot, and bounded exact project artifact paths. The command requires a full Git revision and a manifest from that revision; its report binds included and missing selections, hashes, and completeness to the selected snapshot. It does not infer task relevance or validate acceptance criteria. See [Project artifact inputs](documentation.md) for the run contract. An adopting repository owns the task snapshot, artifact selection, and any expected-resource inventory preflight.

The published source, Windows/Linux gates, versioned assets, provenance and owner publication are recorded in the [production assessment](production-assessment.md). The immutable v0.7.0 release has four verified assets, and its post-publication README used the attested Windows/Linux digests.

## v0.8.0 — repository layout

The published v0.8.0 release recommends `.markitect/areas/<owner>/` for canonical typed knowledge, organized by responsibility. New `init` plans use `.markitect/areas/<namespace>` when `--path` is omitted; explicit paths and existing Projects remain supported. Embedded authoring distinguishes typed Areas from human-owned documentation and uses configured ownership before the new-project convention. Optional kind-suffixed names do not infer YAML type or identity. The executable [layout example](../examples/repository-layout/README.md) checks new paths and generated native outputs while existing fixtures retain earlier layouts.

[Repository layout](repository-layout.md) owns the compatibility assessment. Generic sibling companions, provider output paths, Project schema, root agent instructions and pinned distribution/bootstrap paths retain their contracts. A separate companion-output design, distribution migration, optional layout lint, and migration helper are deferred until demonstrated needs justify their compatibility costs. The reviewed source, Windows/Linux release gates, immutable publication and attested assets are recorded in the [production assessment](production-assessment.md). README installation examples and versioned WinGet metadata use the verified release. Existing consumers must explicitly upgrade their installed tools or project pins.

## v0.8.1 — resolved snapshot boundary

The published v0.8.1 release gives resolved project state a concrete value in `internal/snapshot`: an ID, provisional status, file bytes, and regular-file mode tokens. Parsing, context, review decisions, and impact consume resolved values. Snapshot comparison deterministically reports added, modified, and removed paths from bytes and modes. Git revision resolution, commit-tree reads, and working-tree acquisition stay in `internal/source`; materialization writes the selected value to a filesystem tree without consulting Git. Git-specific write guards remain in the use cases and source adapter that need repository safety. Release bundling and content-package source provenance retain their Git contracts.

The public CLI continues to accept its existing Git revision selectors such as branches and `HEAD`; the fixed-run manifest and stored review-evidence workflows retain their full commit ID requirement. YAML continues to use its existing `revision` field, and package pins produced by the CLI continue to use `git:<full-commit-id>`. Review application logic can compare fixed opaque IDs, while the existing CLI enforces full Git commit evidence before resolving those workflows. Snapshot IDs do not enter the legacy content digest. The existing provisional directory reader remains; this release adds no new fixed directory or archive provider, selector syntax, or generic provider registry. See [Source snapshots](source-snapshots.md) for the complete boundary and compatibility inventory, and the [production assessment](production-assessment.md) for the verified source, tag, platform gates and release assets.

## v0.9.0 — separated outputs and consolidated tool pins

The published v0.9.0 release deliberately replaces the sibling Markdown and consumer distribution contracts. `Project.spec.targets` selects `markdown` explicitly to render deterministic views and local generated navigation under `docs/markitect/<area>/`; a Project without that target requires no generic Markdown output. Codex and Claude projections route directly to canonical YAML or explicitly declared original files and operate independently of generic views. Canonical Area paths remain explicit and configurable; filename suffixes do not infer resource kind or identity.

Consumer bundles place the complete pin set under `.markitect/tool/` (`lock.yaml`, `release.yaml`, `source.zip`) and `.markitect/bootstrap/` (`run.go`, `run_test.go`). The source package command uses the same tool path, and content-package pin suggestions use `.markitect/packages/`. Caches remain outside the declarative tree. Installer, bootstrap and bundle verification retain digest, source provenance, exact-file-set and repository write protections. This release introduces one new bundle contract rather than legacy layout modes or an automatic migration helper.

Generated files retain exact ownership, deterministic links and drift/stale checks. They cannot become ordinary `spec.files` inputs. Existing sibling outputs must be reviewed and removed explicitly when adopting the new contract. Root project instructions and human documentation remain project-owned. Initialization still creates only the Project and one Area README.

Executable fixtures demonstrate both optional Markdown views and provider-only operation. Benchmark fixture v1 remains immutable; v2 covers the new layout. The release-transition benchmark records each binary's compatible fixture and suppresses direct performance-change percentages when fixture versions differ. [Repository layout](repository-layout.md) and [Operations](operations.md) own the implementation and release contracts. The reviewed source, Windows/Linux gates, immutable publication, verified public downloads and versioned transition benchmark are recorded in the [production assessment](production-assessment.md).

## v0.9.1 source candidate — source-relative prose navigation

The patch candidate corrects copied resource prose whose relative links broke when v0.9.0 moved Markdown views. Prose URLs are authored relative to canonical YAML: generic Markdown views map exact local typed YAML destinations to their selected view, while provider inline text keeps canonical destinations. Ordinary document/image links are rebased without changing their target. Query strings, fragments, reference links and code examples are covered by focused tests; link projection uses explicit snapshot files to distinguish ordinary Markdown from an old generated companion. It infers no dependencies or input declarations. [Repository layout](repository-layout.md#source-relative-prose-navigation) owns the contract. This candidate is not yet a published distribution.

## Later product options

Current source adds a neutral [onboarding exercise](onboarding.md), its replay in Windows CI, and structural fixture checks on both supported platforms. [WinGet distribution](winget.md) records the submitted v0.5.0 portable manifests and local installation/upgrade checks; submission is distinct from publication in Microsoft's catalog. The [MCP evaluation](mcp-evaluation.md) records the concluded bounded assessment, the decision to keep MCP experimental, and the deferred second-client criterion. The MCP prototype remains experimental and is excluded from published CLI binaries.

- MCP, LSP, and graph visualization when measured authoring needs justify them.
- Runtime/operator integration only for a concrete state-reconciliation contract.

These options are not v0.8.0 acceptance promises. Keep the core provider-independent and add abstractions only for a demonstrated invariant.
