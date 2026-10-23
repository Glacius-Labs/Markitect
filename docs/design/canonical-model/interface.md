# Interface

An **Interface** describes a contract for interaction across an external or internal boundary. It states what participants exchange, what they may expect and what they must satisfy. Examples include a command line, a public API or an internal service contract.

Owner decision, 17 October 2026: Interface also covers internal technical contracts. An internal caller can be a [Component](component.md); it need not become an Actor merely to use an Interface. A Component describes a system part and its responsibility, while an Interface describes its interaction contract.

## Standard-library contract

The expanded external/internal scope and body links with `spec: {}` are owner-selected on 17 October 2026. This page owns the consolidated Kind wording under the [standard-library contract](standard-library.md#internal-contract-and-component-boundary). The [discussion history](standard-library-history.md#internal-contract-and-component-boundary) preserves the initial Library example. The package has not been implemented.

This Definition has its own file in the selected Package `markitect/software` when implemented:

```yaml
imports:
  - package: markitect/core
    as: Core
id: Interface
kind: Core.Kind
body: >-
  Describes a contract for interaction across an external or internal boundary.
  Its body states offered operations or exchanged information,
  the obligations of participants and observable outcomes.
  It states formats, ordering and failure behavior where these constrain
  a valid realization. A caller need not be an Actor.
spec: {}
```

The current Library [callable interface](library-example/m1/model/library/callable-interface.yaml) already states its JSON Lines format, operations, ordering and errors. It currently has Kind `software/Rule`. A future Interface Definition could own the interaction contract without a second Rule that repeats it. That is a proposed model change, not a change made by this documentation.

No `actors`, `useCases`, `protocol`, `provider` or `version` Property is proposed yet: body links identify participants and capabilities, and the prose already fixes the exact protocol. A specialized structured contract would be justified by an actual consumer such as an exact schema generator or validator. Generic Interface need not anticipate every protocol's fields.

An Interface Definition does not mandate a Go interface type, a C# interface, a network service or a separate process. A Component can offer several Interfaces. In the example, component bodies state which contract they provide or use; those statements do not prove that the realized code conforms or impose a universal provider-selection rule.
