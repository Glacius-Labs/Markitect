# UseCase

A **UseCase** describes an action available through the system, its initiator and its outcome. An [Interface](interface.md) can own its interaction contract; an [Actor](actor.md) can identify a participating role. The Kind does not require separate Actor or Interface Definitions for every action.

## Standard-library contract

Owner-selected direction, 17 October 2026: use body links for domain relationships and `spec: {}`. This page owns the consolidated Kind wording under the [standard-library contract](standard-library.md#body-links-and-empty-specs), which explains why the earlier rules Property proposal was dropped. The package has not been implemented.

This Definition has its own file in the selected Package `markitect/software` when implemented:

```yaml
imports:
  - package: markitect/core
    as: Core
id: UseCase
kind: Core.Kind
body: >-
  Describes an action available through the system.
  Its body states who initiates it, its inputs, applicable conditions,
  successful outcome and expected refusals.
  Observable changes and what must remain unchanged are explicit.
spec: {}
```

The Library's [borrow](library-example/m1/model/lending/borrow.yaml) directly names membership eligibility, copy availability and the custom lending policy in its body. All three can carry applicable obligations even though `LendingPolicy` is not the Kind `Rule`. No separate `rules` or `uses` list repeats these references.
