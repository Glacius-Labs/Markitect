## Working method

Before changing code, read the backlog item and explore the relevant code, its callers
and tests. Then choose a small, coherent implementation. Keep WORKLOG.md current so
you can pick up where you left off.

Plan, use subagents and get an independent review when that helps; give every helper
a clear task and clear ownership. You integrate and check the result yourself.

When something fails, find the cause, fix it and rerun the affected checks. Keep
existing behavior working and keep code and docs readable. Work on feature branches and
merge finished, checked work to `main`. When the released work is done, report what is
done and what is open, the commits on `main`, the checks you ran and any real blocker.

Wave S3 needs real parallel teamwork that overlaps in time. Wave S4 must change behavior
and references consistently and keep the compatibility the backlog asks for.
