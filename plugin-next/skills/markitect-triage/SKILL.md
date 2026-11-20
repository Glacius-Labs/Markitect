---
name: markitect-triage
description: The triage job of a Markitect project. It sorts the new reports, rates them and sets the switch of each - repair by a build, a spec change by an architect job, or a new decision escalated to the owner with two or three unbuilt proposals - and writes only Rating, Actions and Closing. Use when new reports are waiting, or the user or a runner asks for triage.
---
Run the triage job in the [frame of a job](../../jobs.md). The job is `triage`. It writes only the Rating, Actions and Closing sections of reports.

1. **New reports** are the files in `.markitect/reports/` with neither a Rating nor a Closing section. **Answered reports** are the open ones whose Actions name a branch that `git branch --merged TARGET` shows landed. When there are none, say so and end.
2. **Sort.** Start `markitect-next:triage` with their names, the answered ones marked as landed. Its step is checked with `markitect-next status --step triage --before BEFORE --target TARGETDIR`.
3. **Land** as the frame says, so that the target holds each switch.
4. **The switch.** Then, from the target, for each valid report:
   - **build:** run the `markitect-next:markitect-build` skill on the branch its Actions name;
   - **architect:** run the `markitect-next:markitect-architect` skill as a job for the report on the branch its Actions name, and then `markitect-next:markitect-build` on that branch, so that the spec change lands with its realization;
   - **escalated:** nothing is built. Tell the owner: in a session, in your answer, with the proposals and the recommendation; a runner tells them its own way.
