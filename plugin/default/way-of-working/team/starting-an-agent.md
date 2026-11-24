# Starting an agent

Type: [Rule](../../core/rule.md)

## Applies to
[Every role an agent fills](README.md)

## Rule
Every piece of work gets a fresh agent, for exactly one role ([fresh context](../principles/fresh-context.md)). A role that works with a person runs in a session of its own, which the person switches to: the Assistant opens it where the provider's tool allows that, with the level's AI model and the brief, and else gives the person the one line that starts it; a role that works alone runs as a fresh subagent of whoever starts it, except the Housekeeper: the daily run is a session of its own, started by a routine or by the Assistant on a person's yes, and it depends on no person's session. Whoever starts an agent reads from the agent that fills the role the level it runs on, and from the level the AI model of its provider and the effort, at that moment, so a change there holds from the next start. They tell it who started it, for whom, and the item it works on, so that it can write its report. What is done inside the session one is in, such as [capture](../observing/capture.md) or finishing a planning, starts no agent and runs with that session's AI model. The deepest chain of agents starting agents is the Housekeeper, a Builder, a Reviewer.
