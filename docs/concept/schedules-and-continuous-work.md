# Schedules and continuous work

The loop runs outside Markitect, as a scheduled job or routine that calls `markitect`, and there is no daemon in the product ([DEC-084](../decisions.md#dec-084-continuous-work-runs-outside-the-tool-and-autonomy-grows-in-steps)). The tool sets no budget and counts no rounds; a budget per task or per day belongs to the job and the driver.

The cheapest scheduled check needs no agent: `markitect status` in CI or a job fails on any item, so a change made around the agents, a merge or a hand edit is seen.

**Routines** act on it. The harness ships one routine file for three: cleanup, bughunt and `review --all`. A scheduled prompt is one line that names the routine and points at that file, which the run reads when it starts, so `update` reaches a routine that is already scheduled. `/markitect-schedule` sets them up with the owner through Claude Code's own scheduling, a scheduled task of the desktop app on the owner's machine or a cloud routine on a fresh clone; Markitect schedules nothing itself.

- **A branch of its own.** A run works in a worktree or clone of its own, from the newest commit of the source branch, the remote's when the branch tracks one, so that nothing unpushed goes out.
- **It lands only when three things hold.** It leaves nothing worse: `status` lists no Definition that it does not also list on the source branch as it now stands. The project's tests pass. And the change is the agents' work alone: it touches nothing under `.markitect/` but the record and the registry pages, and no file that steers the agents, since the run's own session is no subagent and the hook does not hold it. If the source branch moved, the run merges it first and has the reviewer verify again what the merge put on the list or the builder left unverified. It lands as a fast-forward, by a push when the source branch tracks a remote, otherwise by moving the local branch, which Git refuses while it is checked out in a folder.
- **Otherwise it leaves its branch** and says why; a cloud routine, whose clone ends with the run, pushes that branch so that the work is not lost.
- **Each run ends with a short note:** what landed, what is open, and which spec-problems wait for the owner.

So a review that records a new `fail` does not land: its branch holds the record, and its note names the item. Routines take no strategy: nobody is there to answer the prompt with which Claude Code asks before a workflow starts. Two limits stay: the machine must be on and signed in, so a server without a stored login is not covered yet, and external signals need isolation, since issue text can hold instructions aimed at the agent and a native agent has the user's rights. Work that reads outside input needs an isolated machine with limited rights.

<a id="13-what-is-not-yet-shown"></a><a id="14-what-is-not-yet-shown"></a>
