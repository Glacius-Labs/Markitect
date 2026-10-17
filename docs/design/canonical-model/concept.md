# Concept

A **Concept** describes something meaningful in the project's domain, including its meaning and relevant characteristics. Examples include a Task, Priority or TaskStatus.

## Standard-library contract

Owner-selected direction, 17 October 2026: use body links for domain relationships and `spec: {}`. The [standard-library discussion](standard-library.md) records the decision and its boundaries. The following Kind definition is the working wording of that direction; the standard-library package has not been implemented.

Each Definition has its own file. This file would belong to the proposed Package `markitect/software`:

```yaml
imports:
  - package: markitect/core
    as: Core
id: Concept
kind: Core.Kind
body: >-
  Describes something meaningful in the project's domain, its meaning,
  identity, relevant characteristics and relationships.
  A Concept describes intended domain meaning, not a particular live record.
spec: {}
```

The Library's [loan](library-example/m1/model/lending/loan.yaml) describes what a loan means. Actual loan records belong to the realized program. Body links connect that meaning to members, copies, borrowing and returns. A Concept Definition does not declare a new Kind for records; a project declares a separate Kind when it needs its own structured field contract.
