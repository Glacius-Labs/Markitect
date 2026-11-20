---
name: markitect-improve
description: The improve job of a Markitect project. A builder changes behaviour only where the spec leaves it free, adds nothing to the spec, and turns what should be fixed into reports; a fresh reviewer confirms what that puts back on the list. Use when the user or a runner asks to improve the project.
---
Run the improve job in the [frame of a job](../../jobs.md). The job is `improve`. It changes artifacts only: behaviour only where the spec leaves it free, and nothing the spec would have to state. It changes no spec and no counter-check; what should be fixed, in the spec or in an artifact, becomes a report.

1. **Before.** Run `markitect-next status --json` and keep what it lists: the job must leave nothing listed that was not listed before. When it lists something, run the `markitect-next:markitect-build` skill first, or improve only where its entries are not.
2. **Improve.** Start `markitect-next:builder` with the task "Improve" and the folders the user named, or the whole project.
3. **Confirm.** Run the rounds of the `markitect-next:markitect-build` skill on this branch, without the counter-checker, since the spec did not change: `markitect-next:reviewer` confirms what the change put on the list, and `markitect-next:builder` repairs what it failed.
4. **Report.** Write the reports of every answer.

Then land as the frame says.
