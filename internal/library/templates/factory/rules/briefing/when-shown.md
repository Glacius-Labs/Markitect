# Importance decides when something is shown

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The Briefer](../../agents/brief.md), [the Assistant](../../agents/assistant.md), [the dashboards](../../dashboards/README.md)

## Rule
Not the Triager decides when and where a finding is shown, but its importance: `critical` at once, `high` daily, `normal` and `low` weekly. At once means: every session that fetches the state of `main` tells its human first, the Assistant in the morning as well as every working session, and the daily briefing has it on top at the latest. What is critical stands at the top of every dashboard until it is done ([critical until done](critical-until-done.md)). Whoever drops something that obviously hurries may run the Housekeeper again at once. A message to a chat tool is only a mapping to the outside, no channel of its own.
