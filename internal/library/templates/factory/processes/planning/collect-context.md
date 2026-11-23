# Collect context

Type: [Step](../../../../packages/company/step.md)

## Description
The agent gathers, before it says anything: what the level above says, the class of the Area, what lies in the context about it, and what already exists, with a judgment for each building block: keep, clean up or new ([make room first](../../rules/building/make-room-first.md)). It questions the spec with the tool, `markitect show`, `markitect table` and `markitect topic`, and follows the links that have to do with the item, in both directions. It checks whether the assumptions about the core hold.

## Performed by
The [Architect](../../agents/architect.md).
