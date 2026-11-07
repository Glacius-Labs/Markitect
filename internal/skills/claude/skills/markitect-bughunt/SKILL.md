---
name: markitect-bughunt
description: Hunt for bugs beyond what markitect status lists. In a worktree of its own, the markitect-builder agent has the markitect-reviewer agent hunt and record each bug, repairs them, and has the reviewer verify again; a bug the spec does not cover becomes a spec-problem for the owner. The work comes back to this branch when it leaves nothing worse. Use when the user asks for a bug hunt on a Markitect project.
disable-model-invocation: true
allowed-tools: Bash(markitect status), Bash(markitect status *)
---
Run `markitect status` and keep the Definitions it lists. Then start the markitect-builder agent with the message "Hunt bugs.", and add what the user asked to focus on, if anything. Change no file and record no verdict yourself.

In a Git repository, start it in a worktree of its own (the Agent tool's worktree isolation), and in the background where the Agent tool can, so that the user can go on designing, here or in another session. The worktree starts from the last commit of this branch, so it holds no uncommitted change: if `git status` shows any, tell the user they are not part of the hunt and ask whether to commit them first. In a folder that is not a Git repository, start it here and ask the user to leave the folder alone until it ends.

The builder starts the reviewer to hunt; the reviewer records each bug it finds as `fail` on the Definition it breaks, and a bug the spec does not cover as `spec-problem` on the nearest Definition. The builder repairs what failed and has the reviewer verify again, until nothing it can settle is left, or the list has not got shorter for three rounds. Wait for it to end, and do the rest when it has. It ends by naming its folder and branch.

When it has ended, run `markitect status --repo <its folder>`. If its worktree is gone, because the builder changed nothing, run `markitect status` here instead, skip the commit and the merge, and report. Commit everything in its folder in one commit on its branch, `.markitect/review.sum` included. Then:

- **If status there lists no Definition that was not listed here before,** merge its branch into the current branch here, then remove its worktree and branch. If Git refuses, because of uncommitted changes in the way or a conflict, abort the merge, leave the branch as it is and tell the user.
- **Otherwise,** leave the work on its branch, tell the user what the hunt left open, and ask whether to merge it as it stands, so that its spec-problems reach the spec here.

Push nothing. In a folder that is not a Git repository, commit nothing and say that the work is ready to commit.

Then tell the user in a line or two what the hunt found and repaired, whether it was merged, and which spec-problems wait for the owner, each with its question.
