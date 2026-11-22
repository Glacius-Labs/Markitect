# Dashboards

What the dashboards show, for whom, in which blocks. One [dashboard](../../../packages/factory/dashboard.md) per document. Together they answer in one minute: is anything burning, what waits for me, who does what. Norman says the top of them in the morning; the rest stays on the page to look up.

- [Overview](overview.md), for everyone: where do we stand, who does what?
- [Briefing](briefing.md), for everyone, in the meetings: what is to decide?
- [Plan](plan.md), for everyone: what is there, who owns it, how far is it?
- My work, one per role: what waits for me, where do I start? For the [Developer](developer.md), the [QA](qa.md) and the [Admin](admin.md). The Admin's view holds the Developer's blocks too.

The page is built from these documents and from the work as it lies on `main`, locally, and never checked in: the data and the builder lie on `main`, so everyone sees the same picture of the same state. It is rebuilt by whoever needs it: the session that reached a [milestone](../rules/plan/milestones.md), for its person; Norman in the morning; every working session that fetches `main`; the Housekeeper at the end of its run. A dashboard shows what is going on through the claims of those who work on an item.

On purpose, no dashboard shows code, diffs, review rounds or single test runs; the tasks of others, other than as a number at the story; the full text of findings and reports, other than their title line with a link; closed findings and history, figures or trends; the way of working itself; or the log of running sessions, since the story shows who works on it.

To add a view, write a document of type Dashboard here and link it from this list; to change one, raise a finding of type Factory.
