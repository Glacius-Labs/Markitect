# The three exits

Type: [Rule](../../core/rule.md)

## Applies to
[The Triager](../team/triager.md), [the Project manager](../team/project-manager.md), [the types of findings](finding-types.md)

## Rule
A statement aimed at the model is first held against it: when the model already says it and nothing deviates, no finding arises; nor when the artifacts deviate at a document, or the part of one, that a story changes which has not landed: that is simply not built yet ([the model runs ahead](../plan/the-model-runs-ahead.md)); nor when users cannot reach something that is built and switched off until it is accepted ([off until accepted](../accepting/off-until-accepted.md)). Every other finding takes one of three exits. The model already says what is right: it is repaired, a Bug in `ready`, without a human. The answer follows from what is decided: the Triager files the finding in `ready` and names the place in the model it follows from; the Project manager writes it into the model, creates the story that brings the artifacts after it, and closes the finding; the briefing shows it under Decided for you, and a finding reverses it. This exit holds for Requirement and Question; it is no size rule and no way around the work: the finding passes triage, and only what follows from the model is added. The model already says it when a reader finds the answer in the document as it stands; it follows when a general statement has to be applied to a case the document does not name; and it follows only when one reading is possible: where the Triager would have to choose between two, the finding needs a decision. It needs a new decision: the finding goes to `pending`, to a human. A finding that was decided where it arose takes none of them: it is `ready`, with its decision ([what is no finding](not-a-finding.md)).

## Example
A member asks whether renewals also follow the lending period. The model says every loan does: the Project manager adds the sentence to the rule on lending periods, and the briefing shows it.
