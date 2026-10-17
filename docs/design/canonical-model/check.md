# Check

A **Check** describes how an expectation is verified and what counts as success.

## Standard-library contract

Owner-selected direction, 17 October 2026: body links and `spec: {}` for the general-purpose Kind. This page owns the consolidated wording under the [standard-library contract](standard-library.md). The [discussion history](standard-library-history.md#scenario-and-check-proposal) preserves the initial Library example and Property assessment. The package has not been implemented.

This Definition has its own file in the selected Package `markitect/verification` when implemented:

```yaml
imports:
  - package: markitect/core
    as: Core
id: Check
kind: Core.Kind
body: >-
  Describes how an intended expectation is examined.
  Its body identifies the applicable Definitions, the examination method,
  required observations, success criteria and relevant limits of the method.
  It does not record the outcome of a particular execution.
spec: {}
```

A Check can examine a Scenario or directly examine a Rule, Interface, Component or other applicable Definition. It can describe automated observation, review or another suitable method. A Scenario need not have exactly one Check, and every Check need not have a Scenario.

The method does not by itself establish coverage, independence or current success. Executable tests and review procedures realize the declared Check; actual candidate/model bindings, execution records and findings supply evidence.

The repository's existing [source-maintenance Check](../../../.markitect/model/vocabulary/check.yaml) deliberately requires a structured command. It is a narrower project Kind used for literal maintenance wiring, not the proposed universal Check contract. Preserve that concrete structured consumer. A required command can be canonical intent; a general Check still need not require every method to have a command. A specialized Kind can declare command without becoming a subtype of this Kind under v1.
