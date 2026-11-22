# The circle

**Around the spec, a team works in one circle, with roles, findings and a plan** ([DEC-143](../decisions.md#dec-143-the-way-of-working-is-a-circle-for-a-team-with-roles-findings-and-a-plan)). People work on the spec, the target; agents bring it into the artifacts; and what shows up on the way flows back through one entry. People's attention is the currency, so one question runs through everything: does this need a person?

This page summarizes the way of working. It stands in the spec of each project, in its `factory` part, and the `factory` template ships it as a standard that the project changes like any other part ([DEC-147](../decisions.md#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)). The template, with every type and step, is described with the [library](../guides/library.md).

| Station | What happens | Who |
|---|---|---|
| **Observe** | What happens lands as a report in the context | People through their agent; every agent; observers |
| **Triage** | Reports become findings, classified and filed | Triager |
| **Briefing** | What needs a person stands there in short; people decide in conversation | Briefer; people |
| **Record** | Decisions are carried out and stand in the plan | Recorder |
| **Work** | Planning with a person, building without one, accepting. What shows up goes back to Observe | Architect; Builder, Reviewer, Analyst; QA |

The same steps hold at every level: one rule that holds everywhere rather than three that each fit. Reports, findings, the plan and the briefing are on the [next page](the-work.md).

## Roles

People hold roles. A person may hold several, and who holds which stands in the spec.

| Role | Usually works on | Usually left to another role |
|---|---|---|
| **Admin** | What others build on: the core, the way of working, cross-cutting rules, the global spec and the global architecture; and, as a Developer, their own Areas | Polishing and accepting the surface: QA |
| **Developer** | The Areas they own and the Features they have claimed: planning them, starting the build, reopening a stopped build, accepting what has no surface | The core, the way of working and the global spec: an Admin. Polishing the surface: QA. Other Areas: their owners |
| **QA** | The surface and handling of everything in `acceptance`, and whole Features when nothing new is there | Code outside the surface, planning Features: a Developer |

For an agent, its skill is its usual way of working.

## Deviations

Nothing is forbidden. Each role has its usual way of working, and work outside it is a **deviation** with a level, one of the four importances.

| Level | When | What happens |
|---|---|---|
| `low` | Outside one's own role, such as a Developer polishing the surface | The agent asks for the reason and notes it; a single one waits until it recurs |
| `normal` | What the Triager sorts between `low` and `high` | The Analyst looks at it |
| `high` | It hits what others build on: the core, the way of working, the global spec, a Feature someone else claimed, the main branch without a reviewer | Warning, reason, note; the Analyst looks at it at once, and the daily briefing shows it |
| `critical` | Hard to undo or reaching outside: production, messages to the outside, shared services, foreign databases, secrets or customer data in the repository, rewriting the history of the main branch | The agent stops **before** acting, says what is about to happen and asks: really, and why? |

The agent the person is working with notices from the role that the work lies outside the usual, says so in one sentence and asks for the reason; the note lands in the context. Agents that ask no one write the reason themselves, go on at `low` and `high`, and stop at `critical` with a report. An observer of the main branch catches what nobody reported. Each deviation is a reference on a Smell story for that person and that kind of deviation, whose priority is the higher of the level and how often it recurs; when an analysis shows good reasons that pile up, the role is described wrongly, and that is a Factory finding.

## Agents

An agent is a fresh session that runs exactly one skill. Its model, its schedule and whether it asks stand at its Agent document in the spec, and whoever starts it reads them there ([§12](the-project.md)).

| Skill | The one task | Started by | Asks |
|---|---|---|---|
| `norman` | Prepare a person in the morning, propose, and start the right session | The person | Yes |
| `capture` | Put material into the context unchanged | Any agent, at any time | No |
| `architect` | Work out the spec and the plan of one level with a person | Norman | Yes |
| `finish-planning` | Close a planning: check, report, state, branch, main | The planning session | No |
| `build` | Realize a story in `ready` | Norman or the Housekeeper | No |
| `review` | Check a result against its plan and every level above, and confirm | Whoever's result is checked | No |
| `analyze` | Work through a Smell story and write a report | The Housekeeper | No |
| `accept` | Polish and accept a Feature or a Story with QA | Norman | Yes |
| `observe` | Watch one topic and write a report | The Housekeeper | No |
| `create-observer` | Set up a new observer | A person | Yes |
| `triage` | Turn new reports into findings, classify and file them | The Housekeeper | No |
| `record` | Carry out decisions and turn findings into plan items | The Housekeeper | No |
| `brief` | Write the briefings: daily, weekly and for each meeting | The Housekeeper | No |
| `housekeeper` | Run the day | A person or a routine | No |

**Norman** is a person's first contact in the morning. It fetches the main branch, says what is critical first, what happened, what waits for this person, what the others do as far as it touches them and which meeting is today, proposes in one sentence where to start, and starts a session with the right skill for the chosen work.

**The Housekeeper** runs once a day, started by anyone or by a routine, and asks no one: the observers that are due, the Triager, the Recorder, the Briefer, then builders and analysts for what is `ready`, by priority, and a short report of what is new, what is critical and what it started. Each agent keeps its checkpoint in its folder of the context, so that a run that broke off is caught up by the next.

**Meetings** stand in the spec: rhythm, time, participants and what they cover. The Briefer writes a briefing for each in time, and Norman says in the morning which meeting is today. Decisions are made in the conversation; its transcript goes into the context.

## Planning

Planning is the same at every level, the global spec, an Area, a Feature and a Story: the `architect` skill applied to one node of the plan, and to each child in a new session.

1. **Gather context:** the spec, asked with the tool (`show`, `table`, the topic view) and followed along the links that concern the node, in both directions; the level above; the class of the Area; what lies in the context; and what exists already.
2. **Brief** the person as if they knew nothing, abstractly, without the details of the realization.
3. **Collect:** the person tells freely. The agent proposes nothing until nothing more comes.
4. **Order** what was said; a large topic is taken part by part.
5. **Summarize** in the agent's own words; the person checks it.
6. **Check:** a reviewer holds the target picture against every level above, up to the global spec.
7. **Plan:** the agent plans on its own and tells abstractly how. For a Feature that is the cut into Stories with order and dependencies; for a Story, its Tasks. Stories that can only be cut with many dependencies go into its report as coupling.
8. **Accept:** a last summary. The person's yes is their go.
9. **Finish** (`finish-planning`): a reviewer checks the plan against its target picture; the session leaves its report; the story goes to `ready`; the plan goes to the main branch, and a change to the standing spec goes onto the branch of the story that realizes it ([DEC-146](../decisions.md#dec-146-a-spec-change-lands-with-its-realization-and-status-only-shows)).

A person works on one level at a time and goes through the stories of a Feature one after another; while one is built, the next can be planned. The global spec is usually changed by an Admin; when someone else does, that is a deviation, while a record from triage is none.

## Building

What is `ready` is built, once all its predecessors are `done`; what starts without a person depends on its priority.

- A fresh builder realizes the story on a branch of its own, task by task, and does not ask. Within the spec it decides itself and takes the simple or common solution. Tidying up and changing behaviour never share a story, and a smell it meets is reported, not built around.
- A fresh reviewer per round checks the build against the plan, the target picture and every level above, task by task, and returns `comments`. The number of rounds is bounded.
- When builder and reviewer agree, the reviewer runs the targeted tests, writes `confirm`, and the story goes to the main branch as one merge, its spec change and its realization together. No person checks it.
- **Stop.** When the spec turns out to be the problem, the structure resists, or the rounds bring no agreement, the builder stops, leaves its report in the context and puts the story back to `draft`. Nobody restarts it on their own; a person reopens it with the Architect.

## Accepting

Each story with something to accept goes to `acceptance` once it is on the main branch, and each Feature ends with its acceptance. QA works through the list, oldest first, and claims what it takes. It polishes surface and handling freely within the spec; the result goes through a reviewer and to the main branch. When something fundamental must change, the session notices it, recommends discussing the Feature again and writes it into its report, and it enters the circle at the front. What has no surface is accepted by whoever worked it out.

## Starting a project

- **A new project:** `init` lays out the four templates and the skills ([§12](the-project.md)), and a person works through the parts with the Architect, the `factory` part included, which starts as the standard.
- **An existing project:** the Architect extracts the spec, stating what is as what should be. It asks whether the choice of templates fits, and drops or adds. For each part of the spec it starts a subagent that finds its own context, code, documentation and what lies in the context, and fills the part, with a report in its folder of the context: for each document where it comes from, why it is laid out so, what was unclear and where sources contradict each other, such as documentation against code. What is unclear or contradictory is not guessed: it becomes a finding and stays open in the document. The person is shown the parts one after another and decides in conversation; the Architect writes.
