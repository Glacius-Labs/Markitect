# Scenario

A **Scenario** describes a concrete situation, the actions taken and the expected outcome.

## Standard-library proposal

Proposal for discussion, 17 October 2026: body links and `spec: {}`. The [standard-library discussion](standard-library.md#scenario-and-check-proposal) contains the Library example and Property assessment. The owner has not selected this contract yet.

This Definition would have its own file in the proposed Package `markitect/verification`:

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
