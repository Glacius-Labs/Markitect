---
name: markitect-triage
description: The triage job of a Markitect project. It sorts the new reports, rates them and sets the switch of each - repair by a build, a spec change by an architect job, or a new decision escalated to the owner with two or three unbuilt proposals - and closes each on the branch that answers it. It writes only Rating, Actions and Closing. Use when new reports are waiting, when another job wrote reports on the spec, or when the user or a runner asks for triage.
---
Run the triage job in the frame of a job, `${CLAUDE_PLUGIN_ROOT}/jobs.md`. The job is `triage`. It writes only the Rating, Actions and Closing sections of reports. It runs as a job of its own, or on the branch of another job that wrote reports, which then lands as that job.

1. **The reports.** On another job's branch: the reports that job wrote. As a job of its own: the open reports that `markitect reports --open` lists with no rating. When there are none, say so and end.
2. **Sort.** Start `markitect:triage` with their names and the job's branch. Check its step with `markitect status --step triage --before BEFORE`.
3. **The switch,** for each valid report, on this branch:
   - **build:** the rounds of the `markitect:markitect-build` skill; on a build job's own branch, they are its next round;
   - **architect:** the architect job of the `markitect:markitect-architect` skill for the report, then the rounds of the build, so that the spec change lands with its realization;
   - **escalated:** nothing is built, and the document waits on the owner. Tell the owner: in a session, in your answer, with the proposals and the recommendation; a runner tells them its own way.
4. **Close.** As soon as this branch holds a report's answer, a spec change of the architect or an artifact the build repaired, start `markitect:triage` with the report marked as answered, and check the step again. A closed report no longer makes its document wait, so the build realizes the spec change after it. The branch lands only built and confirmed, so the closing lands only with its answer.
5. **Land.** As a job of its own, land as the frame says. On another job's branch, hand back to that job.
