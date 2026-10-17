# Rule

A **Rule** describes a condition that must hold within the project. For example, a Task must have a title.

## Standard-library contract

Owner-selected direction, 17 October 2026: use body links for domain relationships and `spec: {}`. This page owns the consolidated Kind wording under the [standard-library contract](standard-library.md); the standard-library package has not been implemented.

This Definition has its own file in the selected Package `markitect/software` when implemented:

```yaml
imports:
  - package: markitect/core
    as: Core
id: Rule
kind: Core.Kind
body: >-
  Describes a condition that must hold in its stated scope.
  Its body identifies what the condition concerns, when it applies
  and what must hold. A reference alone does not extend its scope.
spec: {}
```

The Library's [availability](library-example/m1/model/catalog/availability.yaml) defines availability through open loans and constrains borrowing and returns. Its sentences state scope and meaning; the links identify their subjects. Structure checks establish reference validity, not fulfillment of the condition.
