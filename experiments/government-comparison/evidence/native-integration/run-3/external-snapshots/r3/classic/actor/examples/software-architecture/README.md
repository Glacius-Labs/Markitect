# Software architecture package experiment

This is a technology-neutral consumer of one offline, exact-pinned Software Architecture package. Its custom Domain describes a Product with three Modules: Orders, Billing, and Inventory. Billing consumes an Interface published by Orders. Inventory has no direct or Interface relationship to either slice and is included to test that a bounded Orders change need not affect an unrelated Module. `Core` has no outgoing relation in this Domain; `Common` is an explicitly owned shared resource.

`Feature` remains optional on a UseCase. Package 1.1.0 adds an explicitly labeled `feature-ownership: required` cohort and proves that each selected UseCase's `module` resolves to the same canonical Module as `feature -> Feature.module`. A selected UseCase without a Feature is a structural path error; an unlabeled UseCase remains outside the rule even if it has no Feature or points to a Feature owned elsewhere. The label therefore defines coverage and is not inferred from Feature presence. Aggregates also name a Module, while UseCases reference Aggregates; matching those owners remains unverified. Handlers are scalar references with exactly-one structural bounds per UseCase. This says nothing about exclusivity: two UseCases can share one Handler. An Interface names one provider Module; multiple Modules may consume it. The current language cannot restrict which Module may consume which Interface by identity. It also cannot deduplicate semantically equivalent Common resources with different identities.

The central `module-dependencies-stay-at-shared-boundaries` constraint permits direct `dependsOn` target kinds `Core` and `Common`; direct Module-to-Module dependencies fail policy. Cross-module collaboration is represented through typed Interfaces. That is a kind-level boundary only: the finite policy language cannot restrict which consumer Modules may use a particular Interface by identity. Its `acyclic` checks are per relation name, not over a combined Module→Interface→Module path. These limitations are exercised as negative pressure probes; a passing model in those probes is not evidence of the missing invariant.

The package Domain versions retain the same API version and resource schema. Versions 1.0.0 and 2.0.0 are immutable historical sources: v1 defines the `validation: required` cohort and checks that its UseCases have `intent: Command`; v2 adds the Validator requirement. Version 1.1.0 adds the `feature-ownership: required` same-target rule without changing those definitions. Version 2.1.0 retains both opt-in cohorts and adds the Validator requirement to the historical v2 rules. The selector cannot inspect `spec.intent`, so a Command without `validation: required` is outside that rule by design. A label defines coverage, not proof of classification or risk.

The package 1.0.0 and 1.1.0 archives are checked in and pinned by exact version, local fixture provenance coordinate, archive path, and SHA-256. Packages 2.0.0 and 2.1.0 are built deterministically from their versioned source fixtures during tests. The tests preserve the historical 1.0.0→2.0.0 lifecycle and exercise the new 1.1.0→2.1.0 lifecycle. Each migration changes an exact pin, keeps the API/schema stable, and reports affected resources. It does not migrate application code. Exceptions bind to one failing selected UseCase; stale decisions remain actionable, and the tests implement Validators before claiming a passing policy state.

## Run the consumer

From the Markitect repository root:

```powershell
go run ./cmd/markitect format --repo examples/software-architecture
go run ./cmd/markitect check --repo examples/software-architecture
go run ./cmd/markitect model --repo examples/software-architecture
go run ./cmd/markitect context --repo examples/software-architecture --namespace engineering --kind Skill --name implement-order
go test ./examples -run SoftwareArchitecture -count=1
```

Regenerate the local package archive only when intentionally changing the matching versioned package source:

```powershell
./examples/software-architecture/build-package.ps1 -Version 1.1.0
```

The script refuses to overwrite an archive and prints the resulting digest and local source coordinate. Exact source directories are `architecture-package-v1`, `architecture-package-v1.1.0`, `architecture-package-v2`, and `architecture-package-v2.1.0`; tests build reproducible archives from those bytes without a network or claiming an upstream commit.

## Architecture-contract evolution

`TestSoftwareArchitectureV2PackagePolicyLifecycleUsesFixedSnapshots` retains the original 1.0.0→2.0.0 policy migration proof. `TestSoftwareArchitectureV11ToV21VersionedPolicyLifecycle` proves the active 1.1.0→2.1.0 lifecycle with the new ownership contract. Both use isolated consumer Git repositories and exact snapshots.

| Historical snapshot | Explicit change | Expected policy evidence |
|---|---|---|
| v1 | Pin `1.0.0`, activate its Domain member. | Module boundaries and selected Command classification pass. Exactly-one Handler remains structural. |
| v2 | Pin `2.0.0` with its new archive digest. | `create-order` and `issue-invoice` fail the Validator rule; Queries remain unselected. Pin/configuration impact conservatively includes the whole consumer. |
| Temporary exception | Record one constraint/subject digest-bound decision for `create-order` at fixed `policyDate`. | Orders is explicitly waived; Billing still fails. One exception cannot waive another subject. |
| Partial implementation | Add Billing's Validator and explicit ownership relation. | Billing passes; Orders remains waived. Agent context and generated Domain/UseCase views show the waiver, rationale, owner, decision and dates. |
| Completed model | Add Orders' Validator; remove the exception and policy date. | Both selected Commands pass without waivers. |

The separate current-baseline lifecycle starts at package 1.1.0, where the two opted-in UseCases pass same-target ownership. Pinning 2.1.0 retains those passes and adds the two Validator failures; implementing both Validators returns a clean model. This lifecycle is independently exercised by `TestSoftwareArchitectureV11ToV21VersionedPolicyLifecycle`.

Separate probes change an exception's subject bytes or advance the fixed review date past expiry: they produce `policy.exception.stale` or `policy.exception.expired` and leave the original finding failed. The same-target tests show a selected missing Feature as structural and non-waivable, while an unlabeled mismatch remains outside coverage. Observe and Plan tests remain read-only; changing inputs makes a saved projection plan stale before Apply. This sequence updates the canonical architecture graph only. No application code is generated or migrated.

## Agent context and explanation

The generated `.agents/skills/implement-order/SKILL.md` points to canonical Skill YAML. Its explicit `uses` relations select the Orders UseCase, Orders Module and the exported package Workflow. `context` follows declared context edges to the Handler, optional Feature, Aggregate, approved extensions, shared Core/Common and Product. Billing and Inventory subjects remain outside this closure. The exact selected Domain bytes and package version accompany the context, along with PolicyResults for subjects in that closure. A failed project blocks context generation; `model` still exposes findings for inspection.

To explain applicability, join a PolicyResult's API version and constraint name to the Domain's selector/assertion, then compare the subject's kind and exact labels. In the current source iteration, `model.domainInputs` also identifies the Domain API/name, source member, package/version and digest; `context.inputs[].via` names inclusion edges and their source locations. `impact.causes` distinguishes local changes, reverse invalidation edges and conservative global causes. These explanation additions are not available in the published v0.11.0 executable.

The [experiment notes](../../docs/design/software-architecture-experiment.md) answer all ten exit questions. The [pressure report](../../docs/design/domain-language-pressure.md) records unproven joins, incoming Handler exclusivity, identity-pair allowlists, label-selection limitations and mixed-relation cycles. No Pattern/Composition primitive is introduced.
