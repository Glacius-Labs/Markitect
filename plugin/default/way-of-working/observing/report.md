# Report

What one session or one run leaves behind: what it did and what struck it. A report is information: it demands nothing and stays unchanged, except that a link to a document that later leaves the model becomes a note ([taking a document out of the model](../planning/taking-a-document-out.md)). Every session and every run leaves one ([one per session](one-per-session.md)); a run of a watch that observed nothing leaves none, only its checkpoint, so there is no "all well". It is no finding: the Triager makes findings from reports.

## Participants
Type: [Actor](../../core/actor.md)

Who took part, each a link to an [Actor](../../core/actor.md): the agent that filled the role, or, for something several did together, such as a meeting, everyone who was there. Who was there without being an actor of the project is named in plain text. Example: `[Builder agent](../team/builder-agent.md)`.

## Started by (optional)
Type: [Actor](../../core/actor.md)

Who started the session or run, as a link to an [Actor](../../core/actor.md): a person, or the agent that started it, such as the Housekeeper's or, for a Reviewer, the Builder's. Left out when a timer started it. Example: `[Housekeeper agent](../team/housekeeper-agent.md)`.

## For (optional)
Type: [Person](../../core/person.md)

The [Person](../../core/person.md) behind it, as a link. Left out for a run nobody asked for. Example: `[Ada](../team/ada.md)`.

## Started
When it started: `YYYY-MM-DD HH:MM`. Example: `2026-11-24 14:32`.

## Finished
When it ended: `YYYY-MM-DD HH:MM`. Example: `2026-11-24 15:05`.

## Outcome
How it ended, in a word or two, such as `merged`, `aborted`, `planned`, `confirmed`, `accepted`, `observed` or `source unreachable`. An aborted session states its reason under Statements. Example: `merged`.

## Item (optional)
The plan item the session worked on, a [Story](../plan/story.md) or a [Feature](../plan/feature.md), as a link. Example: `[Renew a loan](../plan/renew-a-loan.md)`.

## Statements (optional)
What the session or run observed or wants known, bundled by theme and numbered throughout, so that a finding can point at statement 4. Each statement is one sentence, with numbers, and the reference to its place; for a source outside the repository, the excerpt in the source's wording. Example: `3. Renewing failed twice for loans older than a year. [run 412, renew.test.ts]`.

## Not observed (optional)
What the run could not see this time, such as a source without access. Example: `The test logs of run 412: not downloadable.`
