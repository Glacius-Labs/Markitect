# Importance decides when something is shown

Type: [Rule](../../core/rule.md)

## Applies to
[The Briefer](../team/briefer.md), [the Assistant](../team/assistant.md), [the views](../views/README.md)

## Rule
Not the Triager decides when and where a finding is shown, but its importance: `critical` at once, `high` daily, `normal` and `low` weekly. At once means: every session that fetches the state of `main` tells its human first, the Assistant in the morning as well as every working session, and the daily briefing has it on top at the latest. What is critical stands at the top of every view until it is done ([critical until done](critical-until-done.md)). Whoever files something critical also sends a message at once to the persons it concerns, those who fill the role responsible for its module, else the Admin, through what each person's document names under Reach: one line and the link to the finding, never customer data or secrets. Such a message only points; what is decided, and where, stays as it is. Everything below `critical` waits for the person's next session. The daily run may be started again at any time, so that what was dropped or decided is carried out the same day ([the daily run](../day/daily-run.md)).
