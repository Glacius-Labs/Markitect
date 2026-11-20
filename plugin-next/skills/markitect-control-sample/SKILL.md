---
name: markitect-control-sample
description: The control-sample job of a Markitect project. A sampler plants known deviations in a throwaway copy, a fresh reviewer checks the copy blind, and the job records what was planted and what the reviewer found in .markitect/samples/, from which markitect-next figures computes the hit rate. The copy never lands. Use when the user or a runner asks for a control sample.
---
Run the control-sample job in the frame of a job, `${CLAUDE_PLUGIN_ROOT}/jobs.md`. The job is `control-sample`; NAME is the sample's name, such as `sample-DAY`. It changes no artifact and no spec: only the sample file lands.

1. **The copy.** Lay out a throwaway worktree outside the project on a branch `markitect/NAME` from the job's last commit, `git worktree add -b markitect/NAME COPY HEAD`.
2. **Plant.** Start `markitect-next:sampler` with the copy's folder and how many deviations to plant, a few by default. Keep its `planted` block from everything that follows, and commit the copy.
3. **Check blind.** Start a fresh `markitect-next:reviewer` with the copy's folder and every `changed` and `suspect` entry that `markitect-next status --json` lists there, as `DOCUMENT@FINGERPRINT`, and nothing about what was planted. Commit what it bound in the copy.
4. **Record.** `markitect-next table` in the copy gives each document's verdict. Write `.markitect/samples/NAME.md` in the job folder, of title `# Sample NAME`, with the sections Source, `job control-sample sampler, DAY`; Copy, the commit of the copy; Planted, the sampler's lines; and Verdicts, one line `PATH: VERDICT` for each document the reviewer judged in the copy, and each planted document it left without one as `skip`, which counts as missed. Run `markitect-next check`, commit it as a step of the role `job`, and write the reports of both answers.
5. **The copy goes.** Remove its worktree and its branch; it never lands.

Then land as the frame says. `markitect-next figures` shows the hit rate with the project's reference.
