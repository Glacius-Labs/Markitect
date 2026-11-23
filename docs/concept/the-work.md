# The work

**Reports, findings and plan items hold the work** ([DEC-151](../decisions.md#dec-151-reports-findings-and-plan-items-hold-the-work-and-every-story-names-the-places-its-files-realize)). What happens lands as a report; what deserves attention in it becomes a finding; what is to be done becomes a plan item; and what reaches people is the briefing. The loop is the improvement loop of ISO 9001, clause 10.2, nonconformity and corrective action, without approvals.

All of them are documents of the language, with a title, a type line and sections whose value stands in the first line ([§3](the-language.md#values)). Their types stand in the `factory` part of the spec; their documents lie in `.markitect/` ([§12](the-project.md)), and `check` checks them against their types. They are work, not part of the spec, so a link from them into the spec makes nothing suspect.

## Reports

A report is what an agent or an observer writes about one run or session. It lies in the context, in the folder of its kind of agent, named like its Agent document, one file per run named by date, time and its item. Every report has the same head:

| Section | Holds |
|---|---|
| Item | A link to the plan item it worked on, where it has one |
| Started by | A person, the Assistant, the Housekeeper or a routine |
| For | The person behind it |
| Started, Finished | When |
| Outcome | How it ended, such as merged or stopped |

An observer's report adds its statements, each with its source, and what it could not observe. Every session that works on a plan item, planning, building, analysing or accepting, ends with a short report; what it noticed stands there, and with nothing noteworthy no finding comes of it. A run that observed nothing writes only its checkpoint, so the context stays free of "all is well". What the Housekeeper starts is listed in its own report too, and a session that crashed shows by its claim standing still.

Material that lies nowhere else, such as a transcript, is dropped into the context by `markitect-capture` unchanged and kept hidden; an observer's report carries the relevant statements, grouped by topic, each pointing to its place. No customer data and no secret goes into the repository.

## Findings

A finding is one thing that deserves attention, without a solution. The Triager writes it from the reports and bundles related statements into one finding per topic.

| Points at | Type | Means | Usually filed in |
|---|---|---|---|
| The spec | **Requirement** | A statement of what should hold that is missing from the spec or stands there differently | `pending` |
| | **Idea** | A raw proposal for something new | `pending` |
| | **Question** | Unclear or contradictory, needs an answer | `pending` |
| The product | **Bug** | Does not do what it should | `ready` |
| | **Risk** | Works today, can go wrong | `pending` |
| | **Smell** | A sign that something is not clean | `ready` |
| The plan | **Planning** | Order, scope, date or person | `pending` |
| The way of working | **Factory** | The way of working itself does not work | `pending` |
| None of these | **Unclassified** | Fits no type | `pending` |

Where the answer to a Requirement or a Question follows from what was decided, it goes to `ready` instead ([below](#triage)). Each finding holds its type, its importance, `critical`, `high`, `normal` or `low`, with the reason, its Area, what it is, its source and its context, and for a Bug or a Risk its likelihood and impact. Its folder is its state: `pending` needs a person, `ready` is released for work, and `closed` is done. Every finding ends in `closed`, with its outcome; the Triager files only noise and duplicates there directly.

## Triage

The Triager reads the new reports and looks into a source only when something is unclear. A statement about the spec is held against it: what the spec already says makes no finding. A finding that recurs is bundled into a new finding against the concept one level higher, where the problem may lie. Each finding leaves by one of three exits ([DEC-145](../decisions.md#dec-145-triage-has-three-exits-and-what-follows-from-what-was-decided-is-recorded-without-a-question)):

- **The spec already says what is right:** `ready`. The Recorder makes it a story with fixed tasks, and it is built without a person.
- **The answer follows from what was decided:** `ready`, naming the place it follows from. The Recorder records it in the spec without a question and closes the finding. The record adds nothing new; it writes down what already held, so it reaches the main branch without a build, and where the artifacts differ, that is a Bug like any other. It shows in the briefing under "decided for you", and a finding reverses it.
- **A new decision is needed:** `pending`. People decide in conversation, from the briefing, and the Recorder carries the decision out.

Which exit a finding takes follows from its type and from what the spec says, not from its size. There is no way past the plan: all work is a plan item, so it stays traceable what happened and why.

## The plan

The plan is a tree.

| Level | What it is |
|---|---|
| **Area** | An area of planning, with an owner, what it covers and an optional class. It stays, and has no spec of its own: it names the parts of the product and the building blocks of the architecture it covers |
| **Feature** | An idea to build, with a target picture of its own. It ends with its acceptance |
| **Story** | Functionally complete in itself, given that its predecessors are done |
| **Task** | One step of a story. A planned story always has tasks |

- An Area is a document of the spec, in the `factory` part; Features and Stories are plan items in `.markitect/plan/`. A plan item has no number of its own: its identity is its place in the tree.
- A class groups Areas: its purpose, what it connects to, what it never does, and who usually plans there. Areas usually follow the architecture, and need not.
- Every Area has a standing Feature, `housekeeping`, for stories without a Feature: bugs, maintenance and smell analyses.
- Ownership lies with the Feature; the Story is built. Whoever starts on a plan item claims it first, by entering themselves on the main branch; whoever is first has it, and another session sees that and stops.
- The levels of what should hold are the spec, a Feature's target picture and a Story's target picture; each fits every level above it.

| Story state | Means | Who acts |
|---|---|---|
| `draft` | Created, not planned yet | The Architect with a person |
| `ready` | Nobody has to design anything more | Waits for its start |
| `in progress` | Claimed | Builder and Reviewer, an Analyst or a person |
| `acceptance` | On the main branch, waiting to be accepted | Whoever accepts |
| `done` | Finished, or given up by a decision | Nobody |

A story starts only when it is `ready` and all its predecessors are `done`; a blocked story is one whose predecessor is still open. Stories with fixed tasks, a bug with a clear spec, a smell analysis and maintenance, are created with their tasks and are `ready` at once. A Smell story rises in priority with each further reference, so a single smell waits and only recurrence starts the analysis. The `factory` template holds the tasks and the thresholds.

## The trace

One trace leads from the origin to the file and back ([DEC-151](../decisions.md#dec-151-reports-findings-and-plan-items-hold-the-work-and-every-story-names-the-places-its-files-realize)). A story names its origin, a finding or the context, and every place in the spec that its files realize, with how it changes each, or `unchanged` where it changes nothing there. Where no document of the spec covers its files, the story adds one, and that is a change. The reviewer's `confirm` binds through the tool, for each of these places, the files and the test that shows it, and replaces that document's list of files ([§6](the-state.md#binding-a-verdict)); the reviewer writes nothing into the story, whose Result is the builder's. The commit names the story.

## The briefing

The briefing is what reaches people: short, ordered by importance, each line pointing to its finding.

| Section | Holds |
|---|---|
| **Needs a decision** | Findings in `pending`, by importance |
| **Decided for you** | What was recorded in the spec because it follows from what was decided, with its place; a finding reverses it |
| **Went to work without a decision** | Findings that went to `ready` |
| **Decided yesterday** | What people decided, and what became of it |

The Briefer writes it daily, weekly and for each meeting. Importance decides when something is shown: `critical` at once, until its story is `done`, and in the next meeting; `high` daily; `normal` and `low` weekly. At once means that every session that fetches the main branch tells its person first.
