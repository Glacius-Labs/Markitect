# The three exits

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The Triager](../../agents/triage.md), [the Recorder](../../agents/record.md), [the types of findings](finding-types.md)

## Rule
A statement aimed at the spec is first held against it: when the spec already says it and nothing deviates, no finding arises. Every other finding takes one of three exits. The spec already says what is right: it is repaired, a Bug in `ready`, without a human. The answer follows from what is decided: the Triager files the finding in `ready` and names the place in the spec it follows from; the Recorder writes it into the spec of the level concerned, the spec as a whole included, and closes the finding; the briefing shows it under Decided for you, and a finding reverses it. This exit holds for Requirement and Question; it is no size rule and no way around the work: the finding passes triage, and only what follows from the spec is added. It needs a new decision: the finding goes to `pending`, to a human.

## Example
A member asks whether renewals also follow the lending period. The spec says every loan does: the Recorder adds the sentence to the rule on lending periods, and the briefing shows it.
