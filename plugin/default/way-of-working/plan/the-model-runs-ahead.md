# The model runs ahead

Type: [Rule](../../core/rule.md)

## Applies to
[The model](../../README.md), [planning](../planning/planning.md), [the Builder](../team/builder.md), [the Reviewer](../team/reviewer.md), [the Triager](../team/triager.md)

## Rule
The model says what should hold, also where it is not built yet. A change to the model goes to `main` as soon as it is worked out: at the end of the planning that wrote it, checked by a Reviewer, without waiting for its realization. So a larger idea stands in the model as a whole before its first story is built, and each story builds a part of it. The artifacts follow through stories and reach `main` only through a Reviewer. An addition that only writes down what follows from what is decided goes to `main` at once as well, and so does a statement a person decided in so many words; each comes with the story that brings the artifacts after it ([record](../recording/record.md)).

What is not built yet, the plan says, document by document. Every story names under Realizes the documents it realizes and, for each, how it changes it. A document that a story changes is ahead of the artifacts until that story has landed, that is, until it stands in `acceptance` or is `done`; a document that a story names as `unchanged` is not ahead because of it. A document counts as built when every story that changes it has landed and a Reviewer confirmed it for each of them ([one trace](../review/one-trace.md)). Whether a person has accepted it yet is another question, which the story's state answers.

The planning cuts the model to fit: what a later story builds stands in a document of its own, which that story names, so that no story finds in its documents a sentence that belongs to another. Where a story still only adds to a document that other stories realize too, its sentence under Realizes says which part is its own, and only that part is ahead: the rest of the document counts as built. Some additions cannot be cut out, such as a new line in the list of what a thing consists of. Where a story adds whole sentences of its own to a document, that is a sign that the document holds more than one thing.

Whoever has to judge a difference between the model and the artifacts, the Triager for a statement, the Builder and the Reviewer for the story they work on, looks first at the stories that name the document, which `markitect show` lists for it. When one of them changes the document, or the part in question, and has not landed, that is not built yet: no bug, no finding and no reason to abort. A difference at a document, or a part, whose stories have all landed is a difference like any other. Whoever merely notices something outside their task drops it as always, without looking further ([drop what you notice](../observing/drop-what-you-notice.md)). The Builder builds what the documents of its own story say and leaves the rest to the stories that name it.

## Example
The round decides that members can renew online, but not while another member has reserved the book. In the planning of the feature, the Architect writes into the module Lending that a loan can be renewed once, and, as a document of its own, that a reservation blocks the renewal; both go to `main` together with the plan. The story "Renew a loan" names the first document and builds it. The second is named by the later story "Block the renewal of a reserved book": until that story has landed, nobody treats the missing block as a bug.
