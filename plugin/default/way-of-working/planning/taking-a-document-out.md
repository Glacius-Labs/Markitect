# Taking a document out of the model

Type: [Rule](../../core/rule.md)

## Applies to
[Planning](planning.md), [the Architect](../team/architect.md)

## Rule
A document leaves the model only through a planning. Before it goes, the Architect looks up what links to it, which `markitect show` lists for the document. What still needs it, the model itself or work that is open, is settled in that planning: the link goes to what replaces the document, or the sentence is rewritten.

Work that is finished keeps a note that the document was once there: in a story that is `done` or `cancelled`, a closed finding, a report or a briefing, the link becomes the document's title as plain text, followed by `(no longer in the model, was PATH)`, the path it had as a code span. Nothing else in that document changes; this is the one change a report takes after it was written. So `markitect check` stays without errors, the finished work still says what it was about, and the history of the repository shows the document under the path the note names.

## Example
The document "Reminder by text message" leaves the model. In the story that built it, long `done`, the line under Realizes becomes: Reminder by text message (no longer in the model, was `lending/reminder-by-text-message.md`): members are reminded by text message.
