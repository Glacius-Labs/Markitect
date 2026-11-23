# Collect context

Type: [Step](../../../../packages/company/step.md)

## Description
The agent gathers, before it says anything: what the level above says, the class of the Area where it has one, what lies in the context about it, and what already exists, with a judgment for each building block: keep, clean up or new ([make room first](../../rules/building/make-room-first.md)). It questions the spec with the tool, `markitect show`, `markitect table` and `markitect topic`, and follows the links that have to do with the item, in both directions. It checks whether its assumptions about what the item builds on hold.

## Performed by
The [Architect](../../agents/architect.md).
