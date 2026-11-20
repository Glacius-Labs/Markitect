---
name: markitect-cleanup
description: The cleanup job of a Markitect project. A builder cleans up the artifacts - structure, doubled code, dead code and names - without changing behaviour, and a fresh reviewer confirms what that puts back on the list. It changes no spec. Use when the user or a runner asks for a cleanup.
---
Run the cleanup job in the [frame of a job](../../jobs.md). The job is `cleanup`. It changes artifacts only, and no behaviour: structure, doubled code, dead code and names. It changes no spec and no counter-check; what should change in behaviour becomes a report.

1. **Before.** Run `markitect-next status --json` and keep what it lists: the cleanup must leave nothing listed that was not listed before. When it lists something, run the `markitect-next:markitect-build` skill first, or clean up only where its entries are not.
2. **Clean up.** Start `markitect-next:builder` with the task "Clean up" and the folders the user named, or the whole project.
3. **Confirm.** Run the rounds of the `markitect-next:markitect-build` skill on this branch, without the counter-checker, since the spec did not change: `markitect-next:reviewer` confirms what the cleanup put on the list, and `markitect-next:builder` repairs what it failed.
4. **Report.** Write the reports of every answer.

Then land as the frame says.
