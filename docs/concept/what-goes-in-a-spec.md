# What goes in a spec

**The spec declares everything that is fixed about what it describes** ([DEC-122](../decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). Fixed is what someone relies on or what was decided. What could differ without bothering anyone is free and stays out.

The image is an interface: it says what holds and hides how it is done, as a module states its responsibility and keeps its decisions behind it. "Specification" is the word for it, since decisions about the inside, such as modules or technology, belong in the spec too, and no interface would show them. What belongs in it follows from what it describes, not from who reads it.

The test keeps the spec short. For software, what whoever relies on it depends on is fixed: inputs and their validation, data shapes, the answer to a refused call, stored data. Detail one can observe but not reasonably depend on, such as the wording of a message or the format of an identifier, is free unless the project fixes it. Loops and data structures are the agents'. For a business, prices, who may do what and opening hours are fixed.

So it is clear what a defect is. Where something differs from what the spec states, it is a defect of the artifact. Where something someone relies on is missing, it is a defect of the spec, and it becomes a finding ([§9](the-work.md)).

## The end state, not the way

The spec describes the project as it should be now, as if it came from an ideal process; the way there is in Git. It describes what should hold, not the work that changes the spec: a backlog, a plan, the tasks towards a goal and the risks of the project stay out. What a running operation does belongs in: its processes and recurring tasks, the way of working of the project, and goals with a date, which say what should hold by then.

**Plan items are work.** A Feature, a Story, a finding or a report is the occasion for a change, not the target: it changes its state all the time and would make everything it links to suspect, and a finished item is history. So the documents of the work lie in `.markitect/`, while their types stand in the spec, in its `factory` part ([§9](the-work.md)). They are written in the same language, and `check` checks them.

## Four templates

A spec starts from four small templates, which `init` lays out; each is a part of the spec, a folder whose README is its table of contents ([§3](the-language.md#parts)).

| Template | Says |
|---|---|
| **`project`** | Why the project exists and what is given: its purpose, its context, its guiding principles and its quality |
| **`product`** | What the product should do, by topic. It has no fixed chapters, and it is structured, never flat |
| **`architecture`** | How it is built: its building blocks and the rules for building |
| **`factory`** | How the team works: roles and persons, the agents, the circle, meetings, dashboards and handbooks, the planning with its Areas and classes, and the types of the work. It is not empty: it holds the way of working as a standard, which the project changes like any part of its spec ([§8](the-circle.md)) |

Areas belong to the planning, not to the architecture: one may want to plan differently from how the system is built, although they usually follow it. An Area is a document of its own in the `factory` part, with its owner and its class, never a folder of the spec.

A business has the parts of the `company` package instead: its offer, its processes, its rules and its team. A business case, scenarios and operational tasks of a software project wait until a project needs them.

## The content model

The `software` template, with its six chapters, is a checklist for the `project` part, not its outline: it names what a software project has to answer somewhere.

| Chapter | Says |
|---|---|
| **Purpose** | Why the system exists: the problem, its goals and what it leaves out |
| **Context** | What is given from outside: the words of the field, external systems, constraints such as laws and standards, and assumptions |
| **Domain** | What the system does: its concepts with their rules, and what can be done with them |
| **Interfaces** | How it is reached: each interface with its operations |
| **Quality** | How well it must work, along the nine characteristics of ISO/IEC 25010 |
| **Architecture** | How it is built: components, rules across the code, decisions, and deployment |

## What the software package asks for

- **World and system.** Context separates what is given from what is built. An assumption states what the system takes for granted so that it can be simpler, who ensures it and what is therefore not built; without it a missing check looks like a bug. A constraint links the rule that imposes it.
- **Behaviour as contracts.** An operation says what it does, what the caller gives, and when it is refused, each condition with its reason and the answer the caller gets, since the caller relies on which answer it gets. A concept's rules are its invariants, and an example under a rule illustrates without fixing anything.
- **Quality with a measure.** Each quality requirement has a measure that decides whether it is met. The Quality type has the nine characteristics of ISO/IEC 25010 as optional sections, and a template lists them empty, so that `check` names each one until it says what it asks of the system or why it asks nothing.
- **Deployment.** Architecture says where and how the system runs.
- **Decisions.** A decision states the choice, why, and the alternatives weighed, so that it is not reopened without new evidence. Its optional section "Reopen when" names the evidence that would reopen it.
