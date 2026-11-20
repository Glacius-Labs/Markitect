# The frame of a job

Every job skill runs in this frame. The session is the job: it starts each role as an agent of its own, holds every loop, writes what the agents report, and does every Git step. No agent starts another. A job asks no one while it runs; what stays open becomes a report.

## Folders

- **The project** is the folder that holds `.markitect/version`. The **target** is the branch the job lands on: the one the user names, or else the branch checked out where the job starts.
- **The job folder.** Work on a branch of the job's own, `markitect/JOB-DAY-HHMM` from the target, or on the branch the skill names, in a worktree of its own outside the project folder: `git worktree add -b BRANCH FOLDER TARGET`. A worktree inside the project would be a file the project does not explain. Run every command there, and name that folder in every message that starts an agent.
- **The before-state and the target checkout** are two more detached worktrees outside the project: one at the commit before the current step, one at the target's last commit. The tool compares folders and uses no Git, so the session lays them out.

## Steps

For each role the skill names:

1. Commit everything in the job folder, and move the before-state to that commit: `git -C BEFORE checkout --detach COMMIT`.
2. Start the role's agent, `markitect-next:ROLE`, with the model `markitect-next settings --json` names for the role in `models`, and a message that names the job folder and what the skill gives it. Wait for its answer.
3. Check what the step wrote: `markitect-next status --step ROLE --before BEFORE --target TARGETDIR`. When it names a change the write table denies, the branch does not land; keep the output for a report.
4. Commit the step's changes as `markitect: JOB ROLE`.

## Reports

Agents report what they find in their answer, as fenced `report` blocks:

````text
```report
# The fee for a late book
## Place
docs/spec/lending/return.md
## Finding
The spec says when a book comes back, and not what a late one costs.
## Occurred or risk
risk
## Violated property
complete
```
````

For each block, write `.markitect/reports/NAME.md`: NAME from the title in lowercase words joined by hyphens, and not yet taken; the title, then `## Source` with `job JOB ROLE, DAY`, the job, the agent's role and today as `YYYY-MM-DD`, then the block's sections as they are. A report of the job itself, such as a step the write table denied, names the role of the step it concerns. Then run `markitect-next check`, correct the form of what the job wrote until it finds nothing there, and commit the reports as a step of the role `job`, checked with `markitect-next status --step job --before BEFORE --target TARGETDIR`. Write no Rating, Actions or Closing: they are triage's.

**Reports on the spec.** An open report whose violated property is not `none` and whose place is a document of the spec makes that document wait, and until triage escalates it, it keeps the branch from landing. So when the job wrote such reports, run the `markitect-next:markitect-triage` skill on the job's branch before the next step that builds and before landing. It sets the switch of each on this branch: an escalated report waits on the owner and lands, and the others are answered here before the branch lands.

## Landing

At the end, move the target checkout to the target's last commit, then:

1. `markitect-next status --target TARGETDIR` in the job folder exits 0;
2. the command `markitect-next settings --json` names in `tests` exits 0, run in the job folder; without one, nothing lands;
3. every step check of the job passed.

When all three hold, merge the job's branch into the target with a merge commit, where the target is checked out; leave a checkout with uncommitted changes alone and say so. Otherwise the branch stays where it is: write a report for what keeps it, unless an open report already says it. Push only when the user or the runner asks. Remove the before-state and the target checkout, and the job folder once the branch has landed.

End with a few lines: what the job did, whether it landed, the reports it wrote, and which of them wait on the owner.
