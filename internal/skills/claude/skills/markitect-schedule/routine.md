# A Markitect routine

You are a scheduled run on a Markitect project. The message that started you names the routine, which is `review`, `bughunt` or `cleanup`, the source branch and the project's folder. Nobody can answer a question while you work, so decide nothing that is the owner's: the spec stays as it is. Markitect itself schedules nothing and uses no Git; every Git step below is yours.

## 1. A branch of your own

Never work on the source branch itself or in the owner's folder. Enter a worktree of your own, named `markitect-`, the routine's name and the date and time, and work only there: the agents you start work there too. If this run started in a fresh clone of its own, as a cloud run does, work in that clone instead.

Then set your branch to the source commit. The source commit is the newest commit of the source branch: if the source branch tracks a remote, fetch it and take the remote's commit, so that nothing the owner has not pushed goes out with your work; otherwise take the local branch's.

## 2. Before

Run `markitect status`. If it reports errors in the spec, do nothing but step 5. Find how the project runs its tests: a Rule in the spec, the README, or the CI configuration.

## 3. The work

Start one agent, by the routine, and wait for it:

- **review:** the `markitect-reviewer` agent with the message "Verify every Definition: markitect status --all." It verifies every Definition again, the safety net for what `markitect status` cannot see.
- **bughunt:** the `markitect-builder` agent with the message "Hunt bugs."
- **cleanup:** the `markitect-builder` agent with the message "Clean up."

Then commit what changed, `.markitect/review.sum` included, in one commit on your branch.

## 4. Land, only when all three hold

Find the source commit again, as in step 1. If it has moved since you began, merge it into your branch first; on a conflict, abort the merge and go to step 5. Then start the `markitect-reviewer` agent with the message "Verify what markitect status lists." for what the merge put back on the list, and commit its record.

1. **Nothing is worse:** `markitect status` here lists no Definition that it does not also list at the source commit. Look with `markitect status --repo <folder>` in a temporary checkout of the source commit outside the project's folder (`git worktree add --detach`), and remove that checkout afterwards.
2. **The project's tests pass** here. If you found none, say so.
3. **The change is the agents' work alone:** `git diff --name-only <source commit>...HEAD` names nothing under `.markitect/` but `review.sum`, `registry.md` and the pages under `registry/`, and no file that steers the agents. Those are, anywhere in the tree and whatever the case of their letters: `.claude/settings.json`, `.claude/settings.local.json`, anything under `.claude/hooks/` or `.claude/rules/`, a `markitect-` skill, agent or workflow under `.claude/`, and `AGENTS.md`, `CLAUDE.md` and `CLAUDE.local.md`. The hook refuses such a change to an agent you start, but not to your own session.

Then bring the source branch to your commit, as a fast-forward and never by force. If it tracks a remote, push your commit to it: `git push <remote> HEAD:<source branch>`. Otherwise move the local branch: `git fetch . HEAD:<source branch>`, which Git refuses unless it is a fast-forward, and refuses while the source branch is checked out in a folder: then do not touch that folder, and say how to land it there, `git merge --ff-only <your branch>`.

If one of the three does not hold, or Git refuses, do not land. Leave your branch as it is, push it if it would end with this run's clone, and say why.

## 5. Report

End with a short note of at most three lines, and send it as a notification too if you can:

- what landed on the source branch, or which branch holds the work and why it did not land;
- what `markitect status` still lists;
- which spec-problems wait for the owner, each with its question.

Leave your worktree, and remove it if your work landed.
