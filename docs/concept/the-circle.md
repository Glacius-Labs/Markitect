# The circle

**Around the spec, the people of a project work in one circle, the same in every Area** ([DEC-150](../decisions.md#dec-150-the-way-of-working-is-one-circle-the-same-in-every-area-with-one-role-by-default)). People work on the spec, the target; agents bring it into the artifacts; and what shows up on the way flows back through one entry. People's attention is the currency, so one question runs through everything: does this need a person?

This page summarizes the way of working. It stands in the spec of each project, in its `factory` part, and the `factory` template ships it as a standard that the project changes like any other part ([DEC-147](../decisions.md#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)). The template itself holds every type and step in detail; it ships in the tool's [library](../guides/library.md).

| Station | What happens | Who |
|---|---|---|
| **Observe** | What happens lands as a report in the context | People through their agent; every agent; observers |
| **Triage** | Reports become findings, classified and filed | Triager |
| **Briefing** | What needs a person stands there in short; people decide in conversation | Briefer; people |
| **Record** | Decisions are carried out and stand in the plan | Recorder |
| **Work** | Planning with a person, building without one, accepting. What shows up goes back to Observe | Architect; Builder, Reviewer, Analyst; whoever accepts |

There are no exceptions: one rule that holds everywhere rather than three that each fit. A person checks only the plan and the changed spec, when planning; agents build, review and merge; the result is tested in use, and what shows flows back as a finding. No change has a check of its own after it, and no Area has a course of its own. Reports, findings, the plan and the briefing are on the [next page](the-work.md).

## Roles

People hold roles, and who holds which stands in the spec. The standard holds for any project with one person: by default there is one role, the **Admin**, who does everything the way of working gives to a person. It plans with the Architect on every level and in every Area, decides on the findings, accepts, owns the Areas and their Features, and changes the way of working.

A project that splits the work adds a document per role, each saying where the role usually works, which agents it usually uses and what another role is usually there for, and narrows the Admin's document to what stays with it. Roles, classes of Areas and observers of a project's own belong to that project, not to the standard. For an agent, its Agent document is its usual way of working.

## Deviations

Nothing is forbidden. Each role has its usual way of working, and work outside it is a **deviation** with a level, one of the four importances.

| Level | When | What happens |
|---|---|---|
| `low` | Outside one's own role or outside an agent's document: work another role or agent is usually there for | The agent asks for the reason and notes it; a single one waits until it recurs |
| `normal` | What the Triager sorts between `low` and `high` | The Analyst looks at it |
| `high` | It hits what others build on: the spec as a whole, the `factory` part, a Feature someone else claimed, the main branch without a reviewer | Warning, reason, note; the Analyst looks at it at once, and the daily briefing shows it |
| `critical` | Hard to undo or reaching outside: production, messages to the outside, shared services, foreign databases, secrets or customer data in the repository, rewriting the history of the main branch | The agent stops **before** acting, says what is about to happen and asks: really, and why? |

The agent the person is working with notices from the role that the work lies outside the usual, says so in one sentence and asks for the reason; the note lands in the context. Agents that ask no one write the reason themselves, go on at `low` and `high`, and stop at `critical` with a report. An observer of the main branch catches what nobody reported. Each deviation is a reference on a Smell story for that person and that kind of deviation, whose priority is the higher of the level and how often it recurs; when an analysis shows good reasons that pile up, the role is described wrongly, and that is a Factory finding.

## Agents

An agent is a fresh session that runs exactly one skill, named `markitect-` and the agent's name; its folder in the context is named like its Agent document. Its model, one line per provider, its schedule and whether it asks stand at its Agent document in the spec, and whoever starts it reads them there ([§12](the-project.md)). Each agent does one thing and uses the tool wherever it helps ([DEC-152](../decisions.md#dec-152-each-command-run-skill-and-agent-does-one-thing-on-its-own)).

| Skill | The one task | Started by | Asks |
|---|---|---|---|
| `markitect-norman` | Prepare a person in the morning, propose, and start the right session | The person | Yes |
| `markitect-capture` | Put material into the context unchanged | Any agent, at any time | No |
| `markitect-architect` | Work out the spec and the plan of one level with a person | Norman | Yes |
| `markitect-finish-planning` | Close a planning: check, report, state, branch, main | The planning session | No |
| `markitect-build` | Realize a story in `ready` | Norman or the Housekeeper | No |
| `markitect-review` | Check a result against its plan, its Feature and the spec, and confirm | Whoever's result is checked | No |
| `markitect-analyze` | Work through a Smell story and write a report | The Housekeeper | No |
| `markitect-accept` | Polish and accept a Feature or a Story with the person who accepts | Norman | Yes |
| `markitect-observe` | Watch one topic and write a report | The Housekeeper | No |
| `markitect-create-observer` | Set up a new observer, on a story | Norman, for a person | Yes |
| `markitect-triage` | Turn new reports into findings, classify and file them | The Housekeeper | No |
| `markitect-record` | Carry out decisions and turn findings into plan items | The Housekeeper | No |
| `markitect-brief` | Write the briefings: daily, weekly and for each meeting | The Housekeeper | No |
| `markitect-housekeeper` | Run the day | A person or a routine | No |

**Norman** is a person's first contact in the morning. It fetches the main branch, says what is critical first, what happened, what waits for this person, what the others do as far as it touches them and which meeting is today, proposes in one sentence where to start, and starts a session with the right skill for the chosen work.

**The Housekeeper** runs once a day, started by anyone or by a routine, and always asks no one, whoever started it: the observers that are due, the Triager, the Recorder, the Briefer, then a builder for every story that is `ready` and analysts for the smell analyses, by priority, and a short report of what is new, what is critical and what it started. Each agent keeps its checkpoint in its folder of the context, so that a run that broke off is caught up by the next.

**Observers** are general and come with the standard: what was dropped into the context, how the plan moves, deviations on the main branch, hotspots, what is built twice, and `status`, which reports what `markitect status` shows anew. A project adds observers for its own sources with `markitect-create-observer`, each on a story like every other change to the spec.

**Meetings** stand in the spec: rhythm, time, participants and what they cover. The Briefer writes a briefing for each in time, and Norman says in the morning which meeting is today. Decisions are made in conversation: in a meeting, whose transcript goes into the context, or by one person telling their agent, which drops the decision there.

## Planning

Planning is the same at every level, the spec, a Feature and a Story: the `markitect-architect` skill applied to one node, and to each child in a new session. An Area has no spec of its own: it names the parts of the product and the building blocks of the architecture it covers, and a change to them is planned as a Feature or a Story like any other.

1. **Gather context:** the spec, asked with the tool (`show`, `table`, the topic view) and followed along the links that concern the node, in both directions; the level above; the class of the Area; what lies in the context; and what exists already.
2. **Brief** the person as if they knew nothing, abstractly, without the details of the realization.
3. **Collect:** the person tells freely. The agent proposes nothing until nothing more comes.
4. **Order** what was said; a large topic is taken part by part.
5. **Summarize** in the agent's own words; the person checks it.
6. **Check:** a reviewer holds the target picture against every level above: a Story's against its Feature's and the spec, a Feature's against the spec.
7. **Plan:** the agent plans on its own and tells abstractly how. For a Feature that is the cut into Stories with order and dependencies; for a Story, its Tasks. Stories that can only be cut with many dependencies go into its report as coupling.
8. **Accept:** a last summary. The person's yes is their go.
9. **Finish** (`markitect-finish-planning`): a reviewer checks the plan against its target picture; the session leaves its report; the story goes to `ready`; the plan goes to the main branch, and a change to the standing spec goes onto the branch of the story that realizes it ([DEC-146](../decisions.md#dec-146-a-spec-change-lands-with-its-realization-and-status-only-shows)).

A person works on one level at a time and goes through the stories of a Feature one after another; while one is built, the next can be planned. The spec as a whole is usually changed by the role whose document says so, by default the Admin; when someone else does, that is a deviation, while a record from triage is none. A change to the spec lands through a story like every other change.

## Building

What is `ready` is built, once all its predecessors are `done`; what starts without a person depends on its priority.

- A fresh builder realizes the story on a branch of its own, task by task, and does not ask. Within the spec it decides itself and takes the simple or common solution, and it runs `check` and `status` while it builds. Tidying up and changing behaviour never share a story, and a smell it meets is reported, not built around.
- A fresh reviewer per round checks the build against the plan, the story's target picture, its Feature's and the spec, task by task, and returns `comments`. The number of rounds is bounded.
- When builder and reviewer agree, the reviewer runs the targeted tests and writes `confirm` for each place in the spec that the story's files realize, which the story names ([§9](the-work.md#the-trace)); it writes nothing into the story. The story goes to the main branch as one merge, its spec change and its realization together. No person checks it.
- **Stop.** When the spec turns out to be the problem, the structure resists, or the rounds bring no agreement, the builder stops, leaves its report in the context and puts the story back to `draft`. Nobody restarts it on their own; a person reopens it with the Architect.

## Accepting

Each story with something to accept goes to `acceptance` once it is on the main branch, and each Feature ends with its acceptance. Whoever accepts, by default the Admin, works through the list, oldest first, and claims what they take. They polish the interface and its use freely within the spec with `markitect-accept`. Accepting has rounds like building: a fresh reviewer per round checks the result, with a bounded number of rounds; what is not accepted within them is a failed build, which goes back to `draft` with a report and flows back as a finding. When something fundamental must change, the session notices it, recommends discussing the Feature again and writes it into its report, and it enters the circle at the front.

A Feature goes to production only once it is accepted; the release itself stays a person's. How a release relates to the main branch is open ([OQ-014](../decisions.md#open-questions)).

## Starting a project

- **A new project:** `init` lays out the four templates and the skills ([§12](the-project.md)), and a person works through the parts with the Architect, the `factory` part included, which starts as the standard.
- **An existing project:** the Architect extracts the spec, stating what is as what should be. It asks whether the choice of templates fits, and drops or adds. For each part of the spec it starts a subagent that finds its own context, code, documentation and what lies in the context, and fills the part, with a report in its folder of the context: for each document where it comes from, why it is laid out so, what was unclear and where sources contradict each other, such as documentation against code. What is unclear or contradictory is not guessed: it becomes a finding and stays open in the document. The person is shown the parts one after another and decides in conversation; the Architect writes.
