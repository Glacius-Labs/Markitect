# Actor

An **Actor** describes a role that interacts with the system or participates in work on the project. This role can be fulfilled by a person or another system.

## Standard-library contract

Owner-selected direction, 17 October 2026: body links and `spec: {}`, as for Concept, Rule and UseCase. This page owns the consolidated Kind wording under the [standard-library contract](standard-library.md). The [discussion history](standard-library-history.md#actor-and-interface-proposal) preserves the initial Library example and rationale. The package has not been implemented.

This Definition has its own file in the selected Package `markitect/software` when implemented:

```yaml
imports:
  - package: markitect/core
    as: Core
id: Actor
kind: Core.Kind
body: >-
  Describes a role that interacts with the system or participates
  in project work. Its body states the role's purpose and interactions.
  A person or another system can fulfill the role.
  The role does not by itself define a login account or grant permissions.
spec: {}
```

The Library's trusted operator is an Actor candidate. A registered member is domain data and is not automatically an Actor: the current example gives members no direct system interaction. One person may fulfill several roles. A human/system category, interfaces list or permissions list would require a demonstrated need beyond the existing body description.
