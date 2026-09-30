# Production delivery plan

Updated 2026-09-30 for the owner's expanded mandate: finish a professionally operated Markitect release, integrate both Konfyra and the AI Cockpit, exercise real work and report evidence. This plan owns this delivery. The general [product roadmap](implementation-plan.md) retains optional future features.

## Supported outcome

A private, versioned Go tool with canonical YAML, core authoring, deterministic checks, fixed Git inputs and explicit review reuse. Windows amd64 and Linux amd64 are the initial supported platforms. Consumers own their policy, provider behavior and delivery decisions. Ordinary Markdown remains available; only machine-relevant relationships become typed inputs.

Production readiness requires successful installation, upgrade, rollback, misuse/failure checks, hosted gates, documentation and practical consumer use. It does not mean a compiler can prove arbitrary prose correct. A Kubernetes operator, public registry, general plugin framework, MCP/LSP and policy-package dependency solver are outside this delivery unless a concrete acceptance failure requires them.

## Ordered work and evidence

1. **Reconcile current owners.** Inspect the active Konfyra Story/process and concurrent work before changing any branch. Preserve the prior RC3 checks as historical evidence. Baseline the currently unversioned Cockpit without importing nested checkouts or caches.
2. **Close proven core defects.** Independently review parser, graph, scope, snapshots, controlled writes and installation. Reproduce actual failures, repair them in the owning package, add focused regressions and review the repair.
3. **Make releases durable.** Define one exact semantic version, a complete deterministic distribution, source provenance and checksums. Use private GitHub Releases with all assets assembled before publication; enforce immutability where available. Publication requires the exact reviewed commit and successful Windows/Linux gates. Keep release artifacts separate from consumer policy.
4. **Make provisioning routine.** Validate the complete distribution before changes. Provide a small Go installation/upgrade path with explicit source and version, collision/drift protection, a reviewable plan and recovery. Keep offline checks and a committed consumer pin. Preserve supported existing locks or provide a deliberate conversion. Never fetch a floating version during a normal check.
5. **Migrate both consumers.** Reconcile Konfyra through its current delivery route. Model Cockpit General, Consiliari, customer and project work explicitly, including tasks without a project and cross-customer isolation. Preserve provider metadata and authority. Existing consumer-owned compatibility tools may remain during a measured migration; new Markitect tooling and tests are Go.
6. **Exercise and measure.** Run correctness scenarios and repeated Go benchmarks. Use isolated real authoring tasks, test same/changed-input review decisions, and capture source versions, environment, commands, durations, context size and exact outcomes. Report unavailable token counts as unavailable. Fix observed usability friction without weakening conservative invalidation.
7. **Deliver and report.** Run the final source and consumer gates, publish the selected release, provision it deliberately, retain immutable evidence and document current supported features, limits, open external gates and next use. Human decisions are never fabricated from technical results.

The new mandate authorizes both migrations. It supersedes the earlier scheduling choice to wait for Konfyra human acceptance before preparing the Cockpit migration; each consumer still retains its own human delivery gates.

## Work ownership

The coordinator integrates changes and owns release decisions, CLI composition, hosted delivery and the final report. Independent Luna High agents inspect consumer state, review core correctness and prepare bounded implementation work in separate files. A writer is assigned a concrete file/package boundary before editing. Reviews identify the exact candidate and actual inputs.

The source development checkout is the existing `markitect-authoring` worktree on `feature/production-release`, based on source `5df93085461cfe4b7944bc631513ca06a52f80bc`. The canonical primary checkout remains `Glacius Labs/Markitect`. Current Konfyra targets must be rediscovered; the historical `48d6711` pilot is not assumed to be the active delivery owner.

## Acceptance record

Keep implementation state and final commands/results in the final delivery report, linked from the documentation router. Every unresolved provider, hosted-service or human gate is named with its actual consequence. A missing external gate must not silently become a passed test, nor stop independent authorized implementation work.

## Decisions from actual integration

- The general Konfyra migration remains on `feature/markitect` and PR 3007. Current `origin/master` was merged normally; its new WorkSync recovery procedure was transferred into the owning YAML source and verified at merge commit `7c9248efb2e3d83fcfe4e0ca45269672ac5764ff`. A separate Survey Story still uses an older pin and remains separately owned.
- The Cockpit now has a local Git baseline and a dedicated managed migration worktree. Its 17 mechanisms are migrated without promoting arbitrary prose links into dependencies. Provider cutover requires an exact inventory and Go-check parity.
- Source candidate `39346a4d6c252ca9e9591d728e3b3d3e4c4930d7` passed hosted Windows/Linux source quality jobs. Run `36758935789` failed overall because the explicit Actions-token capability probe received HTTP 403 on the admin-only immutable-release setting. This is a measured platform boundary.
- Release assembly therefore runs in CI with a read-only token. A separate Go publisher uses the existing owner's local GitHub CLI authentication, verifies the successful exact workflow artifact and repository setting, then creates and verifies the durable immutable release. Personal owner credentials are not copied into Actions secrets.
- Independent review found a CRLF upgrade with unchanged text pins and a branch-switch gap in controlled writes. Integration also exposed duplicate inherited rule adapters. These are release blockers until repaired and rechecked, not accepted exceptions.
