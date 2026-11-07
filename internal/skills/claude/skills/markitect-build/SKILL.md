---
name: markitect-build
description: Build the spec as it stands. The markitect-builder agent makes the project in this folder match the spec for what markitect status lists, and starts the markitect-reviewer agent, which verifies each item and records it. Use when the user says the spec is ready to build or apply.
disable-model-invocation: true
allowed-tools: Bash(markitect status), Bash(markitect status *)
---
Start the markitect-builder agent with the message "Build what markitect status lists.", and add what the user asked to focus on, if anything. The builder works until markitect status lists nothing, or only what needs the owner; it starts the reviewer itself. Change no file and record no verdict yourself.

When the builder has ended, run markitect status. Commit the work together with .markitect/review.sum in one commit on the current branch, unless that is done already, and push nothing; in a folder that is not a Git repository, commit nothing and say that the work is ready to commit.

Then tell the user in a line or two how it ended: whether status lists nothing, and if not, what is left and which spec-problems wait for the owner. If status lists errors in the spec, offer to fix them with the user.
