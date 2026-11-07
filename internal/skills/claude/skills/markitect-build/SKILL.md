---
name: markitect-build
description: Build the spec as it stands. The markitect-builder agent works in a worktree of its own, makes the project match the spec for what markitect status lists, and starts the markitect-reviewer agent, which verifies each item and records it. The work comes back to this branch when nothing is left to verify. Use when the user says the spec is ready to build or apply.
disable-model-invocation: true
allowed-tools: Bash(markitect status), Bash(markitect status *)
---
Start the markitect-builder agent with the message "Build what markitect status lists.", and add what the user asked to focus on, if anything. Change no file and record no verdict yourself.

In a Git repository, start it in a worktree of its own (the Agent tool's worktree isolation), so that the user can go on working here. The worktree starts from the last commit of this branch, so it holds no uncommitted change: if `git status` shows any, tell the user they are not part of the build and ask whether to commit them first. In a folder that is not a Git repository, start it here and ask the user to leave the folder alone until it ends.

The builder works until markitect status lists nothing, or only spec-problems, or the list has not got shorter for three rounds; it starts the reviewer itself. Wait for it. It ends by naming its folder and branch.

When it has ended, run `markitect status --repo <its folder>`. Commit everything in its folder in one commit on its branch, `.markitect/review.sum` included. Then:

- **If status lists nothing there,** merge its branch into the current branch here, then remove its worktree and branch. If Git refuses, because of uncommitted changes in the way or a conflict, abort the merge, leave the branch as it is and tell the user.
- **Otherwise,** leave the work on its branch and ask the user whether to merge it as it stands; `markitect status` here then lists what remains.

Push nothing. In a folder that is not a Git repository, commit nothing and say that the work is ready to commit.

Then tell the user in a line or two how it ended: whether status lists nothing, and if not, what is left and which spec-problems wait for the owner. If status lists errors in the spec, offer to fix them with the user.
