# UseCase

A **UseCase** describes an action an [Actor](actor.md) can perform with the system and the outcome the action achieves. The system makes the action available through an [Interface](interface.md).

## Standard-library contract

Owner-selected direction, 17 October 2026: use body links for domain relationships and `spec: {}`. The [standard-library discussion](standard-library.md) records why the earlier `rules` Property proposal was dropped. The following wording describes the selected direction; the standard-library package has not been implemented.

This Definition would have its own file in the proposed Package `markitect/software`:

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
