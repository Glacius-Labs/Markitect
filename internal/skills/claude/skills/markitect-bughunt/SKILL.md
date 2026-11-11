---
name: markitect-bughunt
description: Hunt for bugs beyond what markitect status lists. In a worktree of its own, the markitect-builder agent has the markitect-reviewer agent hunt and record each bug, repairs them, and has the reviewer verify again; a bug the spec does not cover becomes a spec-problem for the owner. A large project is hunted in parallel, per Package. The work comes back to this branch when it leaves nothing worse. Use when the user asks for a bug hunt on a Markitect project.
disable-model-invocation: true
allowed-tools: Bash(markitect status), Bash(markitect status *), Bash(markitect config strategy.bughunt)
---
Run `markitect status` and keep the Definitions it lists. Run `markitect config strategy.bughunt` for how to hunt: `sequential`, `parallel`, `auto` or the name of a custom workflow. With `auto`, work sequentially for now. Parallel and custom workflows need Git: in a folder that is not a Git repository, work sequentially. Tell the user in a line how you work. Change no file and record no verdict yourself.

In a Git repository the work starts from the last commit of this branch, so it holds no uncommitted change: if `git status` shows any, tell the user they are not part of the hunt and ask whether to commit them first.

**Sequential.** Start the markitect-builder agent with the message "Hunt bugs.", and add what the user asked to focus on or to leave alone, if anything. In a Git repository, start it in a worktree of its own (the Agent tool's worktree isolation), and in the background where the Agent tool can, so that the user can go on architecting, here or in another session. In a folder that is not a Git repository, start it here and ask the user to leave the folder alone until it ends. Wait for it to end. It ends by naming its folder and branch.

**Parallel, or a custom workflow.** Note the current branch (`git branch --show-current`): the run comes back to it. Then enter a worktree of your own with the EnterWorktree tool, named `markitect-bughunt` with the date and time: the run works on its branch, and Claude Code branches every builder's worktree from it. Then start the workflow with the Workflow tool, `markitect-parallel` for parallel or the custom workflow by its name, with the args `{"mode": "bughunt", "into": "<the branch you noted>", "focus": "<what the user asked to focus on or to leave alone, or empty>"}`. Tell the user that it runs, and stay in the worktree until the workflow has ended. Then leave the worktree with ExitWorktree and keep it: its folder and branch hold the work. The workflow returns the strands' worktrees: remove each one it marks merged, with `git worktree remove --force <folder>` and then `git branch -D <branch>`, and leave what is already gone; the others hold work that is left. If Git or Claude Code refuses to remove one, tell the user which folders and branches are left to remove.

**When the work has ended,** run `markitect status --repo <its folder>`. If its worktree is gone, because nothing changed, run `markitect status` here instead, skip the commit and the merge, and report. Commit what is uncommitted in its folder in one commit on its branch, `.markitect/review.sum` included; a parallel run has committed its work already.

If status there lists a Definition as `changed` or with `no record yet`, the builder ended after a change that no reviewer has seen. Start the markitect-reviewer agent once, with the message "Verify what markitect status lists in the folder <its folder>, and work only there.", wait for it, commit its record in that folder, and run status there again. Then:

- **If status there lists no Definition that was not listed here before,** merge its branch into the current branch here, then remove its worktree and branch. If Git refuses, because of uncommitted changes in the way or a conflict, abort the merge, leave the branch as it is and tell the user.
- **Otherwise,** leave the work on its branch, tell the user what the hunt left open, and ask whether to merge it as it stands, so that its spec-problems reach the spec here.

Push nothing. In a folder that is not a Git repository, commit nothing and say that the work is ready to commit.

Then tell the user in a line or two what the hunt found and repaired, whether it was merged, and which spec-problems wait for the owner, each with its question. A workflow returns the strands it could not merge, each with its branch: name them.
