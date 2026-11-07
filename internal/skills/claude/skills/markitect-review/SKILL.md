---
name: markitect-review
description: Verify the project against its spec. The markitect-reviewer agent verifies what markitect status lists, or every Definition when the user asks for all, records each verdict and changes no code; a long list is split into one reviewer per Package. Use when the user asks to verify or review the project against its spec, or after a change made around the agents.
allowed-tools: Bash(markitect status), Bash(markitect status *)
---
Run `markitect status`, or `markitect status --all` if the user asked to verify everything, and count the items it lists.

- **Up to about 30 items:** start one markitect-reviewer agent with the message "Verify what markitect status lists.", or "Verify every Definition: markitect status --all."
- **More:** start one markitect-reviewer agent per Package that has items, in parallel, each with the message "Verify what markitect status lists in Package P.", or "Verify every Definition in Package P: markitect status --all.", where P is the Package. An item's Package is its Definition's path without the last part, and Root for a Definition with no `/`. With many Packages, start them in waves; each verdict is recorded when it is made, so verifying in stages loses nothing.

Change no file and record no verdict yourself.

When the reviewers have ended, run markitect status. Commit `.markitect/review.sum` alone in one commit on the current branch if it changed, and push nothing; in a folder that is not a Git repository, commit nothing.

Then tell the user in a line or two what the reviewers found: what failed, and which spec-problems wait for the owner. /markitect-build repairs what failed.
