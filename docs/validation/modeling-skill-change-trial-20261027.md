# Modeling skill change trial: Makerspace C5 — 27 October 2026

The owner approved one trial of the modeling lane, now tracked on the [topics page](../workstreams/topics.md#modeling-skills): a fresh agent with only the [modeling skill](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/packaging/skills/markitect-modeling/SKILL.md) turns one Makerspace change request into a model change. The two earlier skill trials ([PR #771](https://github.com/Glacius-Labs/Markitect/pull/771)) wrote a new model; this one changes an existing model. It is a bounded source trial: no Apply, no container and no provider run beyond the two Claude Code subagents.

## Setup

- **Source:** Main `eb1cf307`.
- **Starting model:** `makerspace-v2` v2.1 at step C4 ([`c4/model`](../../tools/comparison/makerspace-v2/README.md)), with its manifest, lock, execution configuration and vendored Libraries, committed in a fresh Git repository outside this repository. `markitect check` passed on it: 188 Definitions in 11 Packages.
- **Request:** the neutral delta [`requirements/c5.md`](../../tools/comparison/makerspace-v2/requirements/c5.md), long-overdue invoices that block reservations and purchases. The trial used C5 rather than C3, because C3 changes one module and C5 reaches five, adds a refusal at a fixed place in two check orders and changes the billing Interface. The agent did not have the original requirements document the delta's section numbers refer to; it found the places through the model.
- **Agent:** a fresh Claude Code subagent on Sonnet, given the skill folder, the request, the project, a `markitect` binary built from the source and `tools/model-analysis`. It was told to stop with questions if the request left a decision open.
- **Review:** a second fresh subagent on Opus, given only the request and the repository's before and after commits, judged fidelity. It did not see the reference model.

## Result

| | |
|---|---|
| Owner questions | None; the agent found nothing open |
| Change | 14 files, 3 new Rules and 1 new refusal code; 192 Definitions |
| check / lint | Exit 0 (re-run independently); no strong, `dependency-rule`, `interface` or cycle finding; one new warning: `create-reservation` now has 11 decisions |
| Independent review | **Faithful with notes:** nothing missing, nothing wrong, nothing invented of substance; every "stays as it is" item holds |
| Manual repair | None |
| Cost | About 144 k tokens and 29 tool calls for the author, 105 k tokens for the reviewer. Wall time was not measured reliably |

**Against the reference.** The reference `c5/model`, written by the model-track session, changes the same Definitions with the same refusal positions, the same reservation rule 4 and the same modeled scope (`appliesTo` billing, bookings and materials). It states the block in one Rule; the trial split it into three: the block itself, its effects, and its calendar bound in `policies`.

**Review notes.**
1. The calendar-bound consequence belongs as a bullet in `computed-times`; the agent moved it to a new Rule to keep `computed-times` under the skill's 300-word limit.
2. The request's example sits in `create-reservation`; the block Rule is its natural home.
3. The block Rules restate the request's "stays as it is" items as binding text.

## Reading

- On one cross-cutting step, the skill alone was enough to change an existing model faithfully, without owner questions or repair.
- The size limit pushed the agent to fragment a rule that reads better whole (note 1). The skill should say that a limit is a signal to look for a missing concept, not a reason to split one topic.
- One trial, one step, one model family: no claim about other change types such as renames (C7) or architecture changes (C2).
