# Dashboards

What the dashboards show, for whom, in which blocks. One [dashboard](../../../packages/factory/dashboard.md) per document. Together they answer in one minute: is anything burning, what waits for me, who does what. Norman says the top of them in the morning; the rest stays on the page to look up.

- [Overview](overview.md), for everyone: where do we stand, who does what?
- [Briefing](briefing.md), for everyone, in the meetings: what is to decide?
- [Plan](plan.md), for everyone: what is there, who owns it, how far is it?
- [My work](my-work.md), for the Admin: what waits for me, where do I start? A project with more roles adds a view per role.

The way of working ships these documents, not a program that builds the page. The page builder is an artifact a project builds from these documents, like any other artifact; it builds the page from the work as it lies on `main`, locally, and the page is never checked in, so everyone sees the same picture of the same state. Until a project has built it, [Norman](../agents/norman.md) tells what the dashboards would show directly from the work, and no agent builds or rebuilds a page. A dashboard shows what is going on through the claims of those who work on an item.

On purpose, no dashboard shows code, diffs, review rounds or single test runs; the tasks of others, other than as a number at the story; the full text of findings and reports, other than their title line with a link; closed findings and history, figures or trends; the way of working itself; or the log of running sessions, since the story shows who works on it.

To add a view, write a document of type Dashboard here and link it from this list; to change one, raise a finding of type Factory.
