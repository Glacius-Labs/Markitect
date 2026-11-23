# Overview

**The tool shows, the agent explains** ([DEC-131](../decisions.md#dec-131-the-tool-shows-claude-explains)). With agents, code and even the spec change while nobody watches. Overview keeps a person's picture of the project right. It confirms nothing: looking at something is no trust mechanism ([§7](trust.md)).

The image is a building model with revision clouds. Floor plan, section and bill of materials are generated from one model, never drawn twice, and what changed since the last issue carries a cloud. The spec is the model, every view is generated from it, and whoever comes back sees the clouds first.

## Three questions

| Question | Answer | From |
|---|---|---|
| What is the project? | The spec | The spec itself, the topic view and the table per type, explained by the agent |
| How does it stand? | The briefing | Findings and the plan, written up by the Briefer ([§9](the-work.md#the-briefing)), and the state |
| What changed since I last looked? | The revision | A comparison of two states |

A picture goes stale through changes one did not see, so the third question is the core.

## The separation rule

What people should not have to take on an agent's word does not come from an agent. The tool, and Git where branches are concerned, show: completely, deterministically, with numbers that add up. The agent tells and explains, with a citation for each statement, and ends by naming what it did not cover. Inventory, status and changes come from the tool. The content of a report or a finding is agents' text: the tool checks only its form, so it counts only with its source, and the briefing links it.

## Two entries

### Briefing

A person starts the day with the Assistant, who tells them what is critical, what happened, what waits for them and which meeting is today, from the briefing the Briefer wrote ([§9](the-work.md#the-briefing)), and what changed since their last look. Everyone gets the briefing; what needs a decision is decided in conversation.

### Explain

Explain answers the whole, a part, a document or a question, only from the spec, with a citation for each statement, and otherwise says "not stated", as a competency question is answered ([§5](is-the-spec-good.md#the-spec-itself)). "Explain the whole" goes through the parts in the order of their tables of contents. A picture, such as the states of a concept, the agent draws only from what documents say, and names them below it.

## Comparing two states

The comparison of two states, the tool's `compare`, compares two folders, each with a spec, a state and the work, and names what is new, gone, changed or suspect, with the old and the new text. The tool uses no Git; the skill lays out the other state. The comparison carries "since I last looked", the seen state against now, and the preview of a branch against the main branch.

A reader's last look is personal. It is a copy in their profile under `~/.markitect/`, outside the project and outside any one client.

## Views, dashboards and handbooks

Views are generic and come from the language, so they work for every spec: the topic view, a table per type, `show` with a document's artifacts and verdict, `status`, and the comparison ([§11](the-tool.md)). Views particular to a package come with a user interface.

A dashboard or a handbook is different: the way of working describes it in the spec, what it shows, for whom and in which sections, and the project builds it from the spec as an artifact, like any other; the standard ships no program that builds it ([§8](the-circle.md)). A handbook of the way of working comes from the `factory` part, a user handbook from the `product` part, so each fact stands once.

There is no user interface of Markitect for now. The spec is readable Markdown, the tool runs natively, and people ask their agent. A user interface comes once Markitect is proven ([§13](what-is-shown.md)).
