# Markitect routine: {{NAME}}

You are a scheduled run on the Markitect project in `{{FOLDER}}`. Nobody can answer a question while you work, so decide nothing that is the owner's: the spec stays as it is. Markitect itself schedules nothing and uses no Git; every Git step below is yours.

## 1. A branch of your own

Never work on `{{SOURCE}}` itself or in the owner's folder. Enter a worktree of your own, named `markitect-{{NAME}}` with the date and time, and work only there: the agents you start work there too. If this run started in a fresh clone of its own, as a cloud run does, work in that clone instead.

Then set your branch to the newest commit of `{{SOURCE}}`. If `{{SOURCE}}` tracks a remote, fetch it and take the remote's commit, so that nothing the owner has not pushed goes out with your work; otherwise take the local branch.

## 2. Before

Run `markitect status`. If it reports errors in the spec, do nothing but step 5. Find how the project runs its tests: a Rule in the spec, the README, or the CI configuration.

## 3. The work

{{WORK}}

Commit what changed, `.markitect/review.sum` included, in one commit on your branch.

## 4. Land, only if nothing is worse

1. If `{{SOURCE}}` has moved since you began, merge it into your branch; on a conflict, abort the merge and go to step 5. Then start the `markitect-reviewer` agent with the message "Verify what markitect status lists." for what the merge put back on the list, and commit its record.
2. Nothing is worse when both hold:
   - `markitect status` here lists no Definition that it does not also list for `{{SOURCE}}` as that branch now stands. Look with `markitect status --repo <folder>` in a temporary checkout of it outside the project's folder (`git worktree add --detach`), and remove that checkout afterwards.
   - The project's tests pass here. If you found none, say so.
3. Then bring `{{SOURCE}}` to your commit, as a fast-forward and never by force. If it tracks a remote, push your commit to it: `git push <remote> HEAD:{{SOURCE}}`. Otherwise move the local branch: `git fetch . HEAD:{{SOURCE}}`, which Git refuses unless it is a fast-forward, and refuses while `{{SOURCE}}` is checked out in a folder: then do not touch that folder, and say how to land it there, `git merge --ff-only <your branch>`.

If anything is worse, or Git refuses, do not land. Leave your branch as it is, push it if it would end with this run's clone, and say why.

## 5. Report

End with a short note of at most three lines, and send it as a notification too if you can:

- what landed on `{{SOURCE}}`, or which branch holds the work and why it did not land;
- what `markitect status` still lists;
- which spec-problems wait for the owner, each with its question.

Leave your worktree, and remove it if your work landed.
