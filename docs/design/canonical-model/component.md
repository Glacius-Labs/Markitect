# Component

A **Component** describes a part of the system, its purpose and its responsibilities.

A Component can be a logical part or have a runtime or deployment role. It can contain other Components; a separate process or deployment is not required. For example, Task Logic is a logical Component, while a backend service can be a runtime Component.

Design decision: the owner chose this general, nestable meaning on 15 October 2026 in the [consolidated design discussion](consolidation.md#1-component-allgemeiner-systemteil-oder-laufzeitbaustein). At that stage the additional purposes of System and Module were still open.

Earlier review status: the owner marked Component versus Module, together with System, as [provisional terms](consolidation.md#arbeitsziel-und-wackelkandidaten). On 17 October the owner selected Component for the first foundation and the [System/Application/Component boundaries](standard-library.md#selected-boundaries). Component retains its broad meaning; System supplies a coherent solution's purpose and external boundary, while Application identifies an executable runtime boundary. Module remains deferred and is not a prerequisite for this first library.

## Boundary with Interface

The owner selected internal technical contracts as part of [Interface](interface.md) on 17 October 2026. The [standard-library discussion](standard-library.md#internal-contract-and-component-boundary) illustrates the distinction without selecting a new architecture for the executable Library model.

A Component describes an intended part of the system: its purpose, responsibility and boundaries. An Interface describes the contract under which parts or external roles interact. Both are model intent; actual source files realize them. A Component is not automatically a class, process, Package or Manager.

In the proposed example, a JSON Lines adapter Component uses an internal lending Interface. A lending-service Component provides that Interface and realizes the existing borrow and return UseCases. The Interface states the calls and results; the Component states responsibility for fulfilling them. A change to a Component's internal representation can preserve its Interfaces.

Body links with explicit sentences state these provider/consumer relationships. The owner subsequently selected body links and `spec: {}` for Component as well as Actor and Interface on 17 October 2026. No automatic formal provides/consumes relation or provider uniqueness check is introduced by this documentation. The particular Component split remains a proposal.

## Standard-library contract

This page owns the consolidated wording for the selected empty-spec contract. It occupies one Definition file in the selected Package `markitect/software` when implemented; the package has not been implemented.

```yaml
imports:
  - package: markitect/core
    as: Core
id: Component
kind: Core.Kind
body: >-
  Describes a part of the system, its purpose, responsibilities,
  boundaries and collaboration with other parts.
  It may be a logical part or have a runtime or deployment role,
  and may contain other Components.
spec: {}
```
