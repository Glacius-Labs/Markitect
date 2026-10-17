# Interface

An **Interface** describes an interaction point through which an [Actor](actor.md) uses one or more [UseCases](use-case.md). Examples include a user interface, command line or API.

## Standard-library proposal

Proposal for discussion, 17 October 2026; not an owner-selected contract. The [standard-library discussion](standard-library.md#actor-and-interface-proposal) gives the Library example. Its concrete JSON Lines boundary motivates describing both the interaction point and its contract. Whether Interface should also cover internal technical contracts remains open.

This Definition would have its own file in the proposed Package `markitect/software`:

```yaml
imports:
  - package: markitect/core
    as: Core
id: Interface
kind: Core.Kind
body: >-
  Describes an interaction point and its contract.
  Its body identifies participants, offered capabilities,
  required interaction behavior and observable outcomes.
  It states formats, ordering and failure behavior where these constrain
  a valid realization.
spec: {}
```

The current Library [callable interface](library-example/m1/model/library/callable-interface.yaml) already states its JSON Lines format, operations, ordering and errors. It currently has Kind `software/Rule`. A future Interface Definition could own the interaction contract without a second Rule that repeats it. That is a proposed model change, not a change made by this documentation.

No `actors`, `useCases`, `protocol`, `provider` or `version` Property is proposed yet: body links identify participants and capabilities, and the prose already fixes the exact protocol. A specialized structured contract would be justified by an actual consumer such as an exact schema generator or validator. Generic Interface need not anticipate every protocol's fields.
