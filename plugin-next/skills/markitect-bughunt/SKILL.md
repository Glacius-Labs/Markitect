---
name: markitect-bughunt
description: The bug-hunt job of a Markitect project. A fresh reviewer looks beyond what markitect-next status lists, reads the artifacts against the spec, and every bug it finds becomes a report for triage. It changes no artifact, no spec and no verdict. Use when the user or a runner asks for a bug hunt.
---
Run the bug-hunt job in the frame of a job, `${CLAUDE_PLUGIN_ROOT}/jobs.md`. The job is `bughunt`. It changes no artifact, no spec and no verdict: what lands is reports, and the build job repairs what triage sends to it.

1. **Hunt.** Start `markitect-next:reviewer` with the task "Hunt bugs" and the folders or documents the user named, or the whole spec. `markitect-next table` lists every document with its artifacts.
2. **Report.** Write the reports of its answer.

Then land as the frame says.
