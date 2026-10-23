# Implement then review: no Explore phase (18 October 2026)

Source of [DEC-052](../register.md#dec-052-implement-then-review-no-explore-phase).

## Evidence

In the bound runs R2-A1-04 (Main 3efbdcd7), R2-A1-05 (Main ddbbd4dc), R2-A1-06 (Main 1d0f31f4), R2-A1-07 (Main 9c7227e5) and R2-A1-08 (Main a9f62311), Explore was 6 steps per run, one for each of the 6 Managers, about 4 % of wall time. All five runs started from an empty repository, where Explore gave no information.

The cost lay elsewhere:

- the write barrier, which held every writing step until the last Explore result was confirmed;
- the prompt rules that explained the phase and the barrier to the agents, against [DEC-051](../register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process).

The intake of unexplained files was never implemented, so it cost nothing in these runs. Removing it is a design simplification: the owner settles unexplained files before Apply, and the file check already fails a run on one.

## Owner instruction

> Eventuell sollten wir auch das Vorgehen überdenken. Explore, Implement, Review könnte too much sein. Eventuell wäre Plan, Implement, Review besser. Oder auch nur Implement, Review.

The Architect recommended Implement, then Review, without a Plan step. The owner answered:

> Dann mache es ab jetzt so
