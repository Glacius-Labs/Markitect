# Agents

The agents of the way of working, one [agent](../../../packages/factory/agent.md) per skill. An agent is a fresh session that runs exactly one skill; each skill is built from its document here and lies in the repository, the same for every runner. Whoever starts an agent reads its model, its schedule and whether it asks from its document, so changing the document changes the next start. The document is the agent's usual way of working; what it does outside it is a [deviation](../rules/deviations/usual-way.md).

| Agent | Skill |
|---|---|
| [Norman](norman.md) | `norman` |
| [Capture](capture.md) | `capture` |
| [Architect](architect.md) | `architect` |
| [Finish planning](finish-planning.md) | `finish-planning` |
| [Builder](build.md) | `build` |
| [Reviewer](review.md) | `review` |
| [Analyst](analyze.md) | `analyze` |
| [Accept](accept.md) | `accept` |
| [Observer](observe.md) | `observe` |
| [Create observer](create-observer.md) | `create-observer` |
| [Triager](triage.md) | `triage` |
| [Recorder](record.md) | `record` |
| [Briefer](brief.md) | `brief` |
| [Housekeeper](housekeeper.md) | `housekeeper` |

To add an agent, write its document here and link it from this table; to change one, change its document through a finding of type Factory. What every agent does besides its task stands in the rules: [drop what you notice](../rules/work/drop-what-you-notice.md), [one per session](../rules/work/one-per-session.md), [claiming](../rules/plan/claiming.md), [deviations](../rules/deviations/README.md), and, for every session that fetches `main`, telling its person what is critical first ([when shown](../rules/briefing/when-shown.md)).
