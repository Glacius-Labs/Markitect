---
name: markitect-review
description: Verify the project against its spec. The markitect-reviewer agent verifies what markitect status lists, or every Definition when the user asks for all, records each verdict and changes no code. Use when the user asks to verify or review the project against its spec.
disable-model-invocation: true
allowed-tools: Bash(markitect status), Bash(markitect status *)
---
Start the markitect-reviewer agent with the message "Verify what markitect status lists." If the user asked to verify everything, such as with --all, say "Verify every Definition: markitect status --all." instead. Change no file and record no verdict yourself.

When the reviewer has ended, run markitect status. Commit .markitect/review.sum in one commit on the current branch if it changed, and push nothing; in a folder that is not a Git repository, commit nothing.

Then tell the user in a line or two what the reviewer found: what failed, and which spec-problems wait for the owner. /markitect-build repairs what failed.
