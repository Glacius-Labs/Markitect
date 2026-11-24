# Observing

How what happens lands in the context: what a report is, how anyone drops what they notice, and which watches look at their source again and again. Observing only states; it never classifies, interprets, fixes or decides.

## Realized in
`.claude/skills/markitect-observe/`: the skill for observe.
`.claude/skills/markitect-capture/`: the skill for capture.
`.claude/skills/markitect-add-a-watch/`: the skill for add a watch.

## Contents
- [Report](report.md): what a session or a run leaves behind.
- [One per session](one-per-session.md)
- [Drop what you notice](drop-what-you-notice.md)
- [Capture](capture.md): dropping material unchanged.
- [Observe](observe.md): the steps of one run.
- [A watch only states](only-states.md)
- [Add a watch](add-a-watch.md): how a new watch is planned and built.

The watches, each of the type [Watch](watch.md), on one topic in one source:

- [The dropped material](watch-dropped-material.md): what people and agents dropped into the context.
- [The plan items](watch-plan-items.md): how the plan moves.
- [The deviations](watch-deviations.md): what happened on `main` against the usual way of the roles.
- [The hotspots](watch-hotspots.md): where development jams in the structure.
- [Duplication](watch-duplication.md): what is built more than once.
- [The status](watch-status.md): what moved between the model and the files.

## How to fill it
The watches here look at sources every project with this way of working has. A project adds the watches for its own sources through stories like any other change, as [add a watch](add-a-watch.md) says, each as a document here, linked from this list: for example a regular run of the full test suite on `main`, where the project has one ([tests](../building/tests.md)), or later logs, alerts, metrics, dependencies or feedback from users, once they exist and can be read without customer data. A watch is for a topic worth looking at again and again; a single look is a story, and a second topic is a second watch.
