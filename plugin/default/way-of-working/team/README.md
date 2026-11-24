# The team

Who works on the project: the roles, and the people and agents who fill them. Processes, tasks, rules and modules name roles, never a person. Nothing is forbidden: a role describes the usual way, and work outside it is a [deviation](../deviations/usual-way.md).

## Contents
The role a person fills. By default there is one, who does everything a human does here; a project with one person needs no other:

- [Admin](admin.md)

The roles agents fill:

- [Assistant](assistant.md): prepares a person, proposes, gets the right work going.
- [Housekeeper](housekeeper.md): runs the day.
- [Observer](observer.md): runs one watch and writes a report.
- [Triager](triager.md): turns new reports into findings.
- [Project manager](project-manager.md): carries out decisions, turns findings into plan items.
- [Briefer](briefer.md): writes the briefings.
- [Architect](architect.md): works out one level with a person.
- [Builder](builder.md): carries out one story.
- [Reviewer](reviewer.md): checks a result against its plan and every level above.
- [Analyst](analyst.md): works through one smell story.
- [Acceptance helper](acceptance-helper.md): polishes and accepts with the person.

The agents, one per role:

- [Assistant agent](assistant-agent.md), [Housekeeper agent](housekeeper-agent.md), [Observer agent](observer-agent.md), [Project manager agent](project-manager-agent.md), [Briefer agent](briefer-agent.md), [Builder agent](builder-agent.md) and [Acceptance helper agent](acceptance-helper-agent.md), on the level [Workforce](levels/workforce.md).
- [Triager agent](triager-agent.md), [Reviewer agent](reviewer-agent.md) and [Analyst agent](analyst-agent.md), on the level [Specialist](levels/specialist.md).
- [Architect agent](architect-agent.md), on the level [Expert](levels/expert.md).

The [levels and AI models](levels/README.md) they run on.

What holds for all of them:

- [Starting an agent](starting-an-agent.md)
- [A skill per activity](a-skill-per-activity.md)

What everyone does besides their own task stands where it belongs: [drop what you notice](../observing/drop-what-you-notice.md), [one report per session](../observing/one-per-session.md), [claiming](../plan/claiming.md), [deviations](../deviations/README.md), and, for every session that fetches `main`, telling its person what is critical first ([when shown](../briefing/when-shown.md)).

## How to fill it
Write one [person](../../core/person.md) per human here, named after them, such as `ada.md`, with the roles they fill, and link it from this list. A project that splits the work adds a role here and narrows the Admin's description to what stays with it. To give a role a more or a less demanding level, change the link in its agent; to take another AI model for a level, change the level. Entering a person with roles that exist is filling, written straight to `main`; a new role, or a change to one, goes through a finding of type Process, for example when deviations of a role pile up with good reasons.
