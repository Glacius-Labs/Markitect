---
name: markitect-read
description: The read-and-question job of a Markitect project. A reader reads the spec with the tool's signals and topic views, answers the question catalogs from the spec alone, and every finding becomes a report. It changes nothing but reports. Use when the user or a runner asks to read and question the spec, and after larger spec changes.
---
Run the read-and-question job in the frame of a job, `${CLAUDE_PLUGIN_ROOT}/jobs.md`. The job is `read`. The reading changes no spec, no artifact and no verdict: it writes reports, which triage answers or escalates on this branch.

1. **Read.** Start `markitect-next:reader` with the topic the user named, a document, a type or a folder, or the whole spec. After a spec change, name the documents `markitect-next compare OLD` names in the job folder, OLD a checkout of the commit before the change outside the project.
2. **Report.** Write the reports of its answer, and run triage on this branch for them, as the frame says: each is escalated, or answered and closed here. Give the user its rated questions: how many are answered, only inferred, answered two ways or not answered.

Then land as the frame says.
