# Scenario

A **Scenario** describes a concrete situation, the actions taken and the expected outcome.

## Standard-library contract

Owner-selected direction, 17 October 2026: body links and `spec: {}`. This page owns the consolidated wording under the [standard-library contract](standard-library.md). The [discussion history](standard-library-history.md#scenario-and-check-proposal) preserves the initial Library example and Property assessment. The package has not been implemented.

This Definition has its own file in the selected Package `markitect/verification` when implemented:

```yaml
imports:
  - package: markitect/core
    as: Core
id: Scenario
kind: Core.Kind
body: >-
  Describes a concrete case of intended behavior.
  Its body states the initial situation, actions and expected outcomes,
  including relevant state that must remain unchanged.
  It connects the applicable Definitions through explicit body links.
spec: {}
```

A Scenario makes an expected case explicit; it is neither an executed test nor a record of observed results. Its conditions and expected outcomes must agree with the governing intent. It does not create an independently maintained copy of that intent or prove complete coverage. Given/when/then headings can help organize the body without becoming separate fields.
