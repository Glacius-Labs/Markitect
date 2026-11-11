---
name: markitect-review
description: Verify the project against its spec. The markitect-reviewer agent verifies what markitect status lists, or every Definition when the user asks for all, records each verdict and changes no code; a long list is verified in parallel, one reviewer per Package. Use when the user asks to verify or review the project against its spec, or after a change made around the agents.
allowed-tools: Bash(markitect status), Bash(markitect status *), Bash(markitect config strategy.review)
---
Run `markitect status`, or `markitect status --all` if the user asked to verify everything. If nothing is to verify, or the spec has errors and so there is no list, say so and start no agent. Run `markitect config strategy.review` for how to review: `sequential`, `parallel`, `auto` or the name of a custom workflow. With `auto`, review sequentially for now. Change no file and record no verdict yourself.

- **Sequential:** start one markitect-reviewer agent with the message "Verify what markitect status lists.", or "Verify every Definition: markitect status --all.", and add what the user asked to focus on or to leave alone, if anything.
- **Parallel, or a custom workflow:** start the workflow with the Workflow tool, `markitect-parallel` for parallel or the custom workflow by its name, with the args `{"mode": "review", "all": <true for every Definition, else false>, "focus": "<what the user asked to focus on or to leave alone, or empty>"}`. Its reviewers work here in this folder.

When the reviewers have ended, run markitect status. If `.markitect/review.sum` changed, commit it alone in one commit on the current branch, and push nothing. The verdicts rest on the files as they are in this folder: if `git status` shows other uncommitted changes, tell the user that the record is ahead of the last commit until they are committed, and ask whether to commit them too. In a folder that is not a Git repository, commit nothing.

Then tell the user in a line or two what the reviewers found: what failed, and which spec-problems wait for the owner. /markitect-build repairs what failed.
