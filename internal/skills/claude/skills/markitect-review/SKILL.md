---
name: markitect-review
description: Verify the project against its spec. The markitect-reviewer agent verifies what markitect status lists, or every Definition when the user asks for all, records each verdict and changes no code; a long list is verified in parallel, one reviewer per Package. Use when the user asks to verify or review the project against its spec, or after a change made around the agents.
allowed-tools: Bash(markitect status), Bash(markitect status *), Bash(markitect config strategy.review)
---
Run `markitect status`, or `markitect status --all` if the user asked to verify everything, and count the items it lists. Run `markitect config strategy.review` for how to review: `sequential`, `parallel`, `auto` or the name of a custom workflow. With `auto`, choose: parallel when the list holds more than about 30 items in at least two Packages (`markitect status --json` counts them per Package), otherwise sequential. Change no file and record no verdict yourself.

- **Sequential:** start one markitect-reviewer agent with the message "Verify what markitect status lists.", or "Verify every Definition: markitect status --all.", and add what the user asked to focus on, if anything.
- **Parallel, or a custom workflow:** start the workflow with the Workflow tool, `markitect-parallel` for parallel or the custom workflow by its name, with the args `{"mode": "review", "all": <true for every Definition, else false>, "focus": "<what the user asked to focus on, or empty>"}`. It starts one reviewer per Package that has items, here in this folder; each verdict is recorded when it is made.

When the reviewers have ended, run markitect status. Commit `.markitect/review.sum` alone in one commit on the current branch if it changed, and push nothing; in a folder that is not a Git repository, commit nothing.

Then tell the user in a line or two what the reviewers found: what failed, and which spec-problems wait for the owner. /markitect-build repairs what failed.
