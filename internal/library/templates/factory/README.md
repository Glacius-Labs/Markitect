# Factory

The way of working of this project, as a part of its spec: who works on it, which agents run, how the circle turns, how work is planned, built and accepted, and which rules hold. Humans work on the spec; agents bring it into the artifacts; what strikes on the way flows back through one entrance. The skills of the agents are artifacts built from this part, as code is built from the product's part; the [handbook](handbook.md) and the [dashboards](dashboards/README.md) are described here and built by the project like every other artifact. The work itself, the context, the findings, the plan and the briefings, lies in `.markitect/` ([the work folder](rules/work/work-folder.md)).

## How to read it

Start with the [principles](rules/principles/README.md) and [the circle](processes/circle/README.md): observe, triage, briefing, record, work. Picture a head of state with no time for details: observers report, a staff sifts the reports and writes a briefing, the head decides what matters and is told at once when something is critical. Then read the roles and agents, the processes, and the rules where a step links to them.

| Part | Holds | Filled |
|---|---|---|
| [Roles](roles/README.md) | the roles and their usual way of working; by default one, the Admin | by this template; the project adds its own |
| [People](people/README.md) | who holds which role | by the project |
| [Agents](agents/README.md) | one document per skill, what its skill is built from | by this template |
| [Observers](observers/README.md) | one document per observed topic | by this template; the project adds its own sources |
| [Processes](processes/README.md) | the circle, planning, building, accepting, with their steps | by this template |
| [Rules](rules/README.md) | principles, the plan, planning, building, triage, briefing, deviations, the work folder | by this template |
| [Classes](classes/README.md) | the catalog of classes of Areas | by this template; the product's classes by the project |
| [Areas](areas/README.md) | the Areas of the plan, with owner, what they cover and class | by the project |
| [Meetings](meetings/README.md) | the recurring meetings | by the project |
| [Dashboards](dashboards/README.md) | what each view shows, for whom | by this template; a project with more roles adds its views |
| [Handbook](handbook.md) | what the handbook of the way of working covers | by this template |

The types of these documents and of the work in `.markitect/` stand in the package factory: [Agent](../../packages/factory/agent.md), [Observer](../../packages/factory/observer.md), [Person](../../packages/factory/person.md), [Meeting](../../packages/factory/meeting.md), [Area](../../packages/factory/area.md), [Class](../../packages/factory/class.md), [Dashboard](../../packages/factory/dashboard.md), [Handbook](../../packages/factory/handbook.md), and for the work [Feature](../../packages/factory/feature.md), [Story](../../packages/factory/story.md), [Finding](../../packages/factory/finding.md), [Report](../../packages/factory/report.md) and [Briefing](../../packages/factory/briefing.md). Roles and rules are of the types [Role](../../packages/core/role.md) and [Rule](../../packages/core/rule.md), processes and steps of [Process](../../packages/company/process.md) and [Step](../../packages/company/step.md).

## How to fill it

This part is a complete way of working: a project with one person that applies it can run it as it is. Fill the parts the table names for the project: the people with their roles, the Areas with their owners and what they cover, the classes of the product, the meetings, and the observers for the project's own sources. Then complete the list of what is [critical](rules/deviations/critical.md) for the project when setting up the repository's settings.

## How to change it

The way of working improves itself, never on the side ([improves itself](rules/principles/improves-itself.md)). A change starts as a finding of type Factory, from any agent or person, and runs through the circle like any other: it is planned in the Area that covers the factory part, the Architect writes it into these documents, and the skills are rebuilt from them. A change to an agent's model or schedule needs no build: whoever starts the agent reads it from its document.
