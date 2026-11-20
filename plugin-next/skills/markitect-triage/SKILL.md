---
name: markitect-triage
description: The triage job of a Markitect project. It sorts the new reports, rates them and sets the switch of each - repair by a build, a spec change by an architect job, or a new decision escalated to the owner with two or three unbuilt proposals - and writes only Rating, Actions and Closing. Use when new reports are waiting, when another job wrote reports on the spec, or when the user or a runner asks for triage.
---
Run the triage job in the frame of a job, `${CLAUDE_PLUGIN_ROOT}/jobs.md`. The job is `triage`. It writes only the Rating, Actions and Closing sections of reports. It runs as a job of its own, or on the branch of another job that wrote reports, which then lands as that job.

1. **The reports.** On another job's branch: the reports that job wrote. As a job of its own: the new reports, the files in `.markitect/reports/` with neither a Rating nor a Closing section, and the answered ones, open reports whose Actions name a branch that `git branch --merged TARGET` shows landed. When there are none, say so and end.
2. **Sort.** Start `markitect-next:triage` with their names, the answered ones marked as landed, and the branch: the job's branch, or a new `markitect/NAME` from the target as a job of its own. Its step is checked with `markitect-next status --step triage --before BEFORE --target TARGETDIR`.
3. **The switch,** for each valid report, on the branch its Actions name:
   - **build:** the rounds of the `markitect-next:markitect-build` skill; on a build job's own branch, they are its next round;
   - **architect:** the architect job of the `markitect-next:markitect-architect` skill for the report, then the rounds of the build, so that the spec change lands with its realization;
   - **escalated:** nothing is built, and the document waits on the owner. Tell the owner: in a session, in your answer, with the proposals and the recommendation; a runner tells them its own way.
4. **Land.** As a job of its own, land the triaged reports as the frame says before the branches of step 3 start from the target. On another job's branch, hand back to that job.
