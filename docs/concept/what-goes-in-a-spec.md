# What goes in a spec

**The spec declares everything that is fixed about what it describes** ([DEC-122](../decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). Fixed is what someone relies on or what was decided. What could differ without bothering anyone is free and stays out.

The image is an interface: it says what holds and hides how it is done, as a component of software states its responsibility and keeps its decisions behind it. "Specification" is the word for it, since decisions about the inside, such as the building blocks of a system or its technology, belong in the spec too, and no interface would show them. What belongs in it follows from what it describes, not from who reads it.

The test keeps the spec short. For software, what whoever relies on it depends on is fixed: inputs and their validation, data shapes, the answer to a refused call, stored data. Detail one can observe but not reasonably depend on, such as the wording of a message or the format of an identifier, is free unless the project fixes it. Loops and data structures are the agents'. For a business, prices, who may do what and opening hours are fixed.

So it is clear what a defect is. Where something differs from what the spec states, it is a defect of the artifact. Where something someone relies on is missing, it is a defect of the spec.

## The end state, not the way

The spec describes the project as it should be now, as if it came from an ideal process; the way there is in the repository's history. It describes what should hold, not the work that changes the spec: a backlog, a plan, the tasks towards a goal and the risks of the project stay out. What a running operation does belongs in: its processes and recurring tasks, the way of working of the project, and goals with a date, which say what should hold by then.

**The work is not the target.** A plan item, a ticket or a report is the occasion for a change: it changes its state all the time, and a finished one is history. How a project keeps its work, in which form and where, is part of its way of working, which its spec states ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).

## What a software project states

A software project's spec answers, in modules of its own choosing ([§3](the-language.md#modules)), why the system exists, what is given from outside, what the system does, how it is reached, how well it must work and how it is built. The default ships general types only, the six of `core` ([§7](the-project.md#the-default)); a project defines the types it needs for the rest, and a new default beyond `core` is open ([roadmap](../roadmap.md#next)).

- **World and system.** What is given from outside stands apart from what is built. An assumption states what the system takes for granted so that it can be simpler, who ensures it and what is therefore not built; without it a missing check looks like a bug. A constraint links the rule that imposes it.
- **Behaviour as contracts.** An operation says what it does, what the caller gives, and when it is refused, each condition with its reason and the answer the caller gets, since the caller relies on which answer it gets. A concept's rules are its invariants, and an example under a rule illustrates without fixing anything.
- **Quality with a measure.** How well the system must work follows the nine characteristics of ISO/IEC 25010, and each quality requirement has a measure that decides whether it is met.
- **Deployment.** The spec says where and how the system runs.
- **Decisions.** A decision states the choice, why, and the alternatives weighed, so that it is not reopened without new evidence. Its optional section "Reopen when" names the evidence that would reopen it.
