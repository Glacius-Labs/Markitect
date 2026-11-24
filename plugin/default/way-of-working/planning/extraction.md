# Extraction

Type: [Activity](../../core/activity.md)

## Trigger
The way of working is applied to a project that exists already and has no model yet. [The Assistant](../team/assistant.md) starts the Architect for it with a person.

## Steps
1. The Architect asks whether the choice of modules taken from the library fits. A module is struck by removing its folder from the model and every link to it; one is added by copying it from the library.
2. It starts one fresh subagent per module of the model, side by side in its working folder, each in its own folder of the model, on the level of the agent that fills the role Architect. It tells each its module, the form of its report (step 3), that nothing unclear is guessed (step 4), and that it runs `markitect check` on what it wrote. Each finds its context itself, code, documentation, what lies in the context, and fills the module.
3. Each subagent leaves a report in the context: for each document, what it comes from, as links; why it is laid out so; what was unclear; where sources contradict each other, such as documentation against code.
4. Nothing that is unclear or contradictory is guessed: it stays open in the document and becomes a finding.
5. The Architect shows the person the modules one after another, with these details; they decide in conversation, and the Architect writes. Each module goes to `main` once the person has decided it, together with its subagent's report ([straight to main](../work/straight-to-main.md)).

## Result
The model states what is as what should be, module by module, and the person has decided every module. What was unclear or contradictory stands open in its document and is a finding.

## Participants
The [Architect](../team/architect.md), with the person who decides.
