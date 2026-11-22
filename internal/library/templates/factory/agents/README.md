# Agents

The agents of the way of working, one [agent](../../../packages/factory/agent.md) per skill. An agent is a fresh session that runs exactly one skill; each skill is built from its document here and lies in the repository, the same for every runner. Whoever starts an agent reads its model, its schedule and whether it asks from its document, so changing the document changes the next start. The document is the agent's usual way of working; what it does outside it is a [deviation](../rules/deviations/usual-way.md).

| Agent | Skill | Asks |
|---|---|---|
| [Norman](norman.md) | `norman` | yes |
| [Capture](capture.md) | `capture` | no |
| [Architect](architect.md) | `architect` | yes |
| [Finish planning](finish-planning.md) | `finish-planning` | no |
| [Builder](build.md) | `build` | no |
| [Reviewer](review.md) | `review` | no |
| [Analyst](analyze.md) | `analyze` | no |
| [Accept](accept.md) | `accept` | yes |
| [Observer](observe.md) | `observe` | no |
| [Create observer](create-observer.md) | `create-observer` | yes |
| [Triager](triage.md) | `triage` | no |
| [Recorder](record.md) | `record` | no |
| [Briefer](brief.md) | `brief` | no |
| [Housekeeper](housekeeper.md) | `housekeeper` | no |

To add an agent, write its document here and link it from this table; to change one, change its document through a finding of type Factory. What every agent does besides its task stands in the rules: [drop what you notice](../rules/work/drop-what-you-notice.md), [one per session](../rules/work/one-per-session.md), [claiming](../rules/plan/claiming.md), [deviations](../rules/deviations/README.md), and, for every session that fetches `main`, telling its person what is critical first ([when shown](../rules/briefing/when-shown.md)).
