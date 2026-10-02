# Software architecture package experiment

This is a technology-neutral consumer of one offline, exact-pinned Software Architecture package. Its custom Domain describes a Product with three Modules: Orders, Billing, and Inventory. Billing consumes an Interface published by Orders. Inventory has no direct or Interface relationship to either slice and is included to test that a bounded Orders change need not affect an unrelated Module. `Core` has no outgoing relation in this Domain; `Common` is an explicitly owned shared resource.

`Feature` is optional on a UseCase. Where present, its `module` and the UseCase's own `module` reference are both modeled, but today Markitect cannot prove they identify the same Module. Aggregates also name a Module, while UseCases reference Aggregates; matching those owners is another unverified join. Handlers are scalar references with exactly-one structural bounds per UseCase. This says nothing about exclusivity: two UseCases can share one Handler. An Interface names one provider Module; multiple Modules may consume it. The current language cannot restrict which Module may consume which Interface by identity. It also cannot deduplicate semantically equivalent Common resources with different identities.

The central `module-dependencies-stay-at-shared-boundaries` constraint permits direct `dependsOn` target kinds `Core` and `Common`; direct Module-to-Module dependencies fail policy. Cross-module collaboration is represented through typed Interfaces. That is a kind-level boundary only: the finite policy language cannot restrict which consumer Modules may use a particular Interface by identity. Its `acyclic` checks are per relation name, not over a combined Module→Interface→Module path. These limitations are exercised as negative pressure probes; a passing model in those probes is not evidence of the missing invariant.

The v1 and v2 package Domains have the same API version and resource schema. V1 defines an opt-in validation cohort through the `validation: required` label and checks that labeled UseCases have `intent: Command`. V2 adds a per-subject rule requiring at least one Validator for that same labeled cohort. The selector cannot inspect `spec.intent`, so a Command without the opt-in label is outside this rule by design. A label is not proof of classification or risk. The example tests that a mislabeled Query is rejected when it is in the selected cohort and documents that an unlabeled Command is not selected.

The package-v1 archive is checked in and pinned by exact version, local fixture provenance coordinate, archive path, and SHA-256. Package-v2 is built deterministically from its source fixture during the tests. The migration changes the exact pin, leaves the API version/schema stable, and reports affected resources. It does not migrate application code. Exceptions bind to one failing selected UseCase, and the test adds Validators and removes the exception before claiming a passing policy state.

## Run the consumer

From the Markitect repository root:

```powershell
go run ./cmd/markitect format --repo examples/software-architecture
go run ./cmd/markitect check --repo examples/software-architecture
go run ./cmd/markitect model --repo examples/software-architecture
go run ./cmd/markitect context --repo examples/software-architecture --namespace engineering --kind Skill --name implement-order
go test ./examples -run SoftwareArchitecture -count=1
```

Regenerate the local package archive only when intentionally changing package-v1 sources:

```powershell
./examples/software-architecture/build-package.ps1 -Version 1
```

The script refuses to overwrite an archive and prints the resulting digest and local source coordinate. Package-v2 source is under `architecture-package-v2`; the migration test builds a reproducible archive from those bytes without requiring a network or claiming an upstream commit.

## Architecture-contract evolution

`TestSoftwareArchitectureV2PackagePolicyLifecycleUsesFixedSnapshots` creates an isolated consumer Git repository and captures each state as a commit. It verifies that v2 retains the v1 API version, kinds, relations and existing assertions, adding only `selected-commands-have-validators`.

| Snapshot | Explicit change | Expected policy evidence |
|---|---|---|
| v1 | Pin `1.0.0`, activate its Domain member. | Module boundaries and selected Command classification pass. Exactly-one Handler remains structural. |
| v2 | Pin `2.0.0` with its new archive digest. | `create-order` and `issue-invoice` fail the Validator rule; Queries remain unselected. Pin/configuration impact conservatively includes the whole consumer. |
| Temporary exception | Record one constraint/subject digest-bound decision for `create-order` at fixed `policyDate`. | Orders is explicitly waived; Billing still fails. One exception cannot waive another subject. |
| Partial implementation | Add Billing's Validator and explicit ownership relation. | Billing passes; Orders remains waived. Agent context and generated Domain/UseCase views show the waiver, rationale, owner, decision and dates. |
| Completed model | Add Orders' Validator; remove the exception and policy date. | Both selected Commands pass without waivers. |

Separate probes change an exception's subject bytes or advance the fixed review date past expiry: they produce `policy.exception.stale` or `policy.exception.expired` and leave the original finding failed. Observe and Plan tests remain read-only; changing inputs makes a saved projection plan stale before Apply. This sequence updates the canonical architecture graph only. No application code is generated or migrated.

## Agent context and explanation

The generated `.agents/skills/implement-order/SKILL.md` points to canonical Skill YAML. Its explicit `uses` relations select the Orders UseCase, Orders Module and the exported package Workflow. `context` follows declared context edges to the Handler, optional Feature, Aggregate, approved extensions, shared Core/Common and Product. Billing and Inventory subjects remain outside this closure. The exact selected Domain bytes and package version accompany the context, along with PolicyResults for subjects in that closure. A failed project blocks context generation; `model` still exposes findings for inspection.

To explain applicability, join a PolicyResult's API version and constraint name to the Domain's selector/assertion, then compare the subject's kind and exact labels. In the current source iteration, `model.domainInputs` also identifies the Domain API/name, source member, package/version and digest; `context.inputs[].via` names inclusion edges and their source locations. `impact.causes` distinguishes local changes, reverse invalidation edges and conservative global causes. These explanation additions are not available in the published v0.11.0 executable.

The [experiment notes](../../docs/design/software-architecture-experiment.md) answer all ten exit questions. The [pressure report](../../docs/design/domain-language-pressure.md) records unproven joins, incoming Handler exclusivity, identity-pair allowlists, label-selection limitations and mixed-relation cycles. No Pattern/Composition primitive is introduced.
