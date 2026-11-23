# Concept

**You architect the project in a typed, declarative specification language in Markdown, its way of working included. Around the spec a team works in one circle: what happens is observed, sorted into findings, decided in conversation and planned; agents build, review and confirm; and the tool computes what is built and verified.**

This page is the one description of the product, and the target that Main builds ([DEC-140](../decisions.md#dec-140-one-concept-describes-the-product)). The [decisions](../decisions.md) list what it rests on, the [roadmap](../roadmap.md) the next steps and where Main is, and the [vision](../vision.md) why.

## The aim

Make one clean and simple version that works reliably ([DEC-120](../decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)): the spec is realized and checked without anyone reading the work, so that people concentrate on architecture and on the decisions only they can make.

**The problem.** When agents build a project, trust and overview no longer come with the work, and nothing in the project states what it should be; the [vision](../vision.md#why-markitect-exists) gives the reasoning. Markitect supplies that statement: one spec with authority, and every file of the project mapped to a document of it or ignored ([§2](what-markitect-is.md)). On it, Markitect answers four questions:

| Question | Markitect's answer |
|---|---|
| Was something forgotten? | `status` computes from the spec and the state what is changed, suspect or unexplained, not from anyone's memory ([§6](the-state.md)) |
| Was it understood correctly? | A fresh reviewer checks each story against its plan and the spec and confirms with evidence. Counter-checks and control samples measure that checking, later ([§7](trust.md)) |
| Does the spec say what was meant? | Only use shows it. What anyone notices becomes a finding; what follows from what was decided is recorded without a question, and a new decision is made by people in conversation ([§9](the-work.md)) |
| How does the project stand, and what changed since I last looked? | The briefing, the tool's views and a comparison of two states; the agent explains with a citation for each statement ([§10](overview.md)) |

## The way of working

1. **Observe.** People tell their agent what they notice; agents note what strikes them; observers watch one topic each on a schedule. Everything lands as a report in the context ([§8](the-circle.md)).
2. **Triage.** The Triager turns reports into findings and files each by one of three exits: the spec already says what is right, the answer follows from what was decided, or a new decision is needed ([§9](the-work.md)).
3. **Briefing.** What needs a person stands in the briefing, short and by importance, and people decide in conversation.
4. **Record.** The Recorder carries out the decisions and puts them in the plan: Area, Feature, Story, Task.
5. **Work.** A person plans a Feature or a Story with the Architect; a Builder realizes it on a branch, spec change and code together; a Reviewer checks it and writes `confirm`; it reaches the main branch without a person, and whoever accepts, by default the Admin, tries what there is to accept. The same course holds in every Area, with no exceptions. What shows up goes back to 1.

The way of working itself stands in the spec, in its `factory` part, and the skills in the repository realize it ([§12](the-project.md)).

## The other pages

2. [What Markitect is](what-markitect-is.md)
3. [The language](the-language.md)
4. [What goes in a spec](what-goes-in-a-spec.md)
5. [Is the spec good?](is-the-spec-good.md)
6. [The state](the-state.md)
7. [Trust](trust.md)
8. [The circle](the-circle.md)
9. [The work](the-work.md)
10. [Overview](overview.md)
11. [The tool](the-tool.md)
12. [The project and its skills](the-project.md)
13. [What is shown and what is not](what-is-shown.md)

## Not yet

Not part of this concept until the [roadmap](../roadmap.md) brings them: the general runs, such as cleanup, bug hunt and improve; counter-checks, control samples and spot checks; the blast radius of a story, computed from the spec; and a mapping of the plan to a ticket system.
