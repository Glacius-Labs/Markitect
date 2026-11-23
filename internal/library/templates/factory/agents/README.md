# Agents

The agents of the way of working, one [agent](../../../packages/factory/agent.md) per skill. An agent is a fresh session that runs exactly one skill; each skill is built from its document here and lies in the repository, the same for every runner. Whoever starts an agent reads its model, its schedule and whether it asks from its document, so changing the document changes the next start. The document is the agent's usual way of working; what it does outside it is a [deviation](../rules/deviations/usual-way.md).

An agent that asks runs in a session of its own, which the person switches to; an agent that does not ask runs as a fresh subagent of whoever starts it, and [capture](capture.md), [finish-planning](finish-planning.md) and [create-observer](create-observer.md) run inside the session that calls them and are no agents of their own. `as the session` under Model means that the agent has no model of its own and takes that of the session it runs in. Whoever starts an agent tells it who started it, for whom, and the item it works on, so that it can write its report. The deepest chain of agents starting agents is the Assistant, Housekeeper, Builder, Reviewer.

| Agent | Skill |
|---|---|
| [Assistant](assistant.md) | `markitect-assistant` |
| [Capture](capture.md) | `markitect-capture` |
| [Architect](architect.md) | `markitect-architect` |
| [Finish planning](finish-planning.md) | `markitect-finish-planning` |
| [Builder](build.md) | `markitect-build` |
| [Reviewer](review.md) | `markitect-review` |
| [Analyst](analyze.md) | `markitect-analyze` |
| [Accept](accept.md) | `markitect-accept` |
| [Observer](observe.md) | `markitect-observe` |
| [Create observer](create-observer.md) | `markitect-create-observer` |
| [Triager](triage.md) | `markitect-triage` |
| [Recorder](record.md) | `markitect-record` |
| [Briefer](brief.md) | `markitect-brief` |
| [Housekeeper](housekeeper.md) | `markitect-housekeeper` |

To add an agent, write its document here and link it from this table; to change one, change its document through a finding of type Factory. What every agent does besides its task stands in the rules: [drop what you notice](../rules/work/drop-what-you-notice.md), [one per session](../rules/work/one-per-session.md), [claiming](../rules/plan/claiming.md), [deviations](../rules/deviations/README.md), and, for every session that fetches `main`, telling its person what is critical first ([when shown](../rules/briefing/when-shown.md)).
