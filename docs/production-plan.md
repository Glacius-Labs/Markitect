# Production delivery decisions

**Recorded:** 2026-09-30; **Status:** Source-release and provisioning decisions are closed for `v0.1.0`. Bounded install/rollback evidence is recorded in the assessment. Final consumer review, PR gates and human acceptance remain owned by each consumer and are not inferred from source delivery.

This document records the decisions that govern this delivery. The [production delivery assessment](production-assessment.md) is the evidence record.

## Closed decisions

1. **Standalone source ownership.** Markitect owns its Go module, YAML vocabulary, schemas, compiler, release bundle and generic examples. Consumer repositories own their policies, provider behavior, migrations and acceptance records. Customer policy content is not copied into the source product.
2. **One typed YAML vocabulary.** Keep the six content kinds and Project envelope. Markdown remains the reading surface; explicit dependencies and signatures live in YAML. Generated schemas are editor aids, not CRDs or semantic validators. Prose links do not become inferred dependencies.
3. **Bounded first release.** `v0.1.0` is a private immutable release for Windows amd64 and Linux amd64. The exact reviewed merge commit, hosted gates, bundle, four release assets and final attestations are recorded in the assessment. A version string alone never signals release acceptance.
4. **Split CI build from owner publication.** CI runs source gates and two-platform bundle/bootstrap smoke tests and retains the exact four-file artifact for 30 days. The Actions `GITHUB_TOKEN` cannot read the admin-only immutable-release setting, so CI does not publish and no PAT is put in workflow secrets. An authorized owner uses the Go publisher and their existing `gh auth` session: inspect its read-only plan, then explicitly select `--publish`. The publisher validates and byte-compares the run artifact, checks release/tag state and asset hashes, and verifies the immutable result.
5. **Explicit provisioning and recovery.** Consumer install uses the complete verified five-file pin, starts with a plan, writes only on a reviewable branch with a committed baseline, and handles individual files atomically rather than claiming a repository transaction. Git history owns integration and rollback.
6. **Separate consumer acceptance.** Local installs and source-profile checks are useful evidence, not a consumer release, provider-runtime approval or human decision. Each consumer closes its own candidate-specific review, hosted gates, rollback and owner acceptance.
7. **Keep future scope outside this release.** Content packages, template initialization, Kubernetes operators/CRDs, a general plugin framework, MCP/LSP, a controlled API domain and public licensing are not shipped by this delivery. Any future format or scope expansion needs its own acceptance and migration.

## Remaining delivery state

The source release and provisioning route are established. Both consumer migrations and practical checks are recorded in the assessment. Konfyra has passing final source and PR-merge gates while its PR remains Draft; the separate Survey Story and human decision remain with the consumer. Cockpit has local Windows/Linux evidence and a prepared CI workflow, with no remote yet. Provider runtime authentication, consumer human acceptance and the explicitly deferred consumer dossier are not inferred from successful technical checks. Consult the assessment for the exact candidate and evidence boundaries.
