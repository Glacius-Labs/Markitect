# Implement then review: no Explore phase (18 October 2026)

Source of [DEC-052](../register.md#dec-052-implement-then-review-no-explore-phase).

## Evidence

In the bound runs R2-A1-04 to R2-A1-08, the Explore phase was 6 steps, about 4 % of wall time. In an empty repository it gave no information.

The cost lay elsewhere:

- the write barrier, which held every writing step until the last Explore result was confirmed;
- the intake of unexplained files, which the barrier waited for;
- the prompt rules that explained the phase and the barrier to the agents, against [DEC-051](../register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process).

## Owner instruction

> Eventuell sollten wir auch das Vorgehen überdenken. Explore, Implement, Review könnte too much sein. Eventuell wäre Plan, Implement, Review besser. Oder auch nur Implement, Review.

The Architect recommended Implement, then Review, without a Plan step. The owner answered:

> Dann mache es ab jetzt so
