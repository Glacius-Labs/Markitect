# Giving up a story

Type: [Rule](../../core/rule.md)

## Applies to
[Story](story.md), [the Project manager](../team/project-manager.md), [decide](../briefing/decide.md)

## Rule
A story nobody wants any more is given up by a decision, like anything else: on its finding, or, where it has none, as a finding of type Planning that is filed as decided ([what is no finding](../triage/not-a-finding.md)). It goes to `cancelled`, never to `done`: `done` says that it was built. The model says what should hold, so what nobody wants any more does not stay in it. The decision therefore says two more things. For each document that only this story realizes, and for each part of a shared document that only this story adds: another story takes it over, or it leaves the model. For each story that depends on it: it goes on without it, or it is given up too. The agent the person decides with asks for both before it drops the decision into the context ([decide](../briefing/decide.md)).

The Project manager carries that out ([record](../recording/record.md)). It sets the story `cancelled`, releases its claim and writes into its Result that it was given up, why, with a link to the decision. It enters a document that is taken over under Realizes of the story that takes it. For what leaves the model, and for what the story already brought to `main` and is not wanted, it creates one story in `draft` that names those documents under Realizes, in the plan of the module that holds them, its origin the report with the decision; so they stay marked as not built until that story's planning takes them out ([the model runs ahead](the-model-runs-ahead.md), [taking a document out of the model](../planning/taking-a-document-out.md)). When the story that was given up had brought nothing to `main`, that planning leaves nothing to build and sets the new story `done` at its finish. It takes the story out of Depends on of each story that goes on, and gives up each one that does not. A decision that leaves one of these open is dropped into the context as unclear, and the story stays as it is.

## Example
The round gives up the story "Remind by text message". Its document "Reminder by text message" leaves the model: the Project manager creates the story "Take out the reminder by text message", which names it. The story "Choose the reminder channel" depended on it and is given up too.
