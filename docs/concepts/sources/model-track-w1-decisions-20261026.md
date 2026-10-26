# Owner decisions on the W1 fitness review

Date: 26 October 2026. Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The coordinator read both owner messages in that session's transcript. The owner statements are verbatim, including one that the app cut off; assistant questions they answer are quoted or summarized and labeled. The recorded meaning is the recorder's and adds no decision.

The decisions belong to the [canonical model track](../../design/model-track-charter.md) ([approval](model-track-approval-20261026.md)). They answer the track's first fitness review, W1, whose report and the G1 design were drafts in that session and are not in the repository at this point.

## 1. Fixing the W1 gaps

Context (summary by the recorder): the session's message presented the W1 draft report, not yet reviewed. It named four blockers: Library Kinds cannot be extended, which needs a core change; Packages have no public surface, proposed as a Library convention checked by a linter; there is no Package dependency graph, answering D3 with "any typed reference or body link" for analysis; and the modeled scope for cross-cutting rules is unused, proposed as a Policy Kind in a Library. It also named smaller points: parent Managers see none of their children's Definitions, [DEC-028](../register.md#dec-028-context-follows-need-to-know-and-starts-simple)'s "See also" section is not built, Library binding and the lock are manual, and maps, unions, defaults and formats can be handled with Library patterns.

Owner reply (verbatim):

> We should fix all of that then. Can you do that?

## 2. Confirmation of G1, D3 and visibility

Assistant question (verbatim): "Confirm G1 with Option A, D3 as proposed, and visibility convention-first?"

Owner reply (verbatim, exactly as received):

> I confirm the G1 Kind extension design v0.2 with Option A (one Core 1.2.0, migrate everything, software-basics becomes 0.3.0, amending D4); D3: a dependency edge is any typed reference or body link, imports are name resolution only; and Package visibility as a Library convention plus linter first, w

Recorder's note: the reply was a reply button offered by the session. The app cut its text at 300 characters, so it ends mid-word. The offered text continued "…plus linter first, with a core rule only on evidence."

## Recorded meaning

Recorder's interpretation:

- **All W1 points are fixed, not measured first.** The owner's message answered a list that included the smaller points, so this covers two points for which the draft report proposed something else: parent Managers seeing their children's Definitions, which the report would measure in pilot W6 before changing, and DEC-028's "See also" section and files to read, which it treated as a known deviation outside the track unless W6 showed the need.
- **G1, Kind extension, design v0.2 with Option A.** A new Core Property `extends` on Kind lists base Kinds. A Kind's effective fields are its bases' fields plus its own, with no overriding. A value reference to a base Kind accepts Definitions of derived Kinds. The extension graph is acyclic, and `Kind` and `Property` cannot be bases. This creates Core 1.2.0. The compiler knows only that Core edition, and everything checked from Main migrates to it.
- **D4 is amended.** `software-basics` becomes 0.3.0 with identical Definitions, instead of keeping 0.2.0 frozen for the existing fixtures.
- **D3 is decided.** For structure analysis and linting, as the W1 report proposed it, a dependency edge is any typed reference or body link; imports are name resolution only. The language contract's statement that a link alone declares no positive dependency keeps its meaning for semantics.
- **Package visibility** is a Library convention checked by a linter first, with a core rule only on evidence. The second half follows the offered text, not the received one.
