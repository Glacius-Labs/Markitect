# Overview

**The tool shows, Claude explains** ([DEC-131](../decisions.md#dec-131-the-tool-shows-claude-explains)). With agents, code and even the spec change while nobody watches. Overview keeps a person's picture of the project right. It confirms nothing: looking at something is no trust mechanism ([§7](trust.md)).

The image is a building model with revision clouds. Floor plan, section and bill of materials are generated from one model, never drawn twice, and what changed since the last issue carries a cloud. The spec is the model, every view is generated from it, and whoever comes back sees the clouds first.

## Three questions

| Question | Answer | From |
|---|---|---|
| What is the project? | The plan | The spec itself, the topic view and the table per type, explained by Claude |
| How does it stand? | The briefing | Reports, the state and the factory's figures |
| What changed since I last looked? | The revision | A comparison of two states |

A picture goes stale through changes one did not see, so the third question is the core.

## The separation rule

What the owner should not have to take on Claude's word does not come from Claude. The tool, and Git where branches are concerned, show: completely, deterministically, with numbers that add up. Claude tells and explains, with a citation for each statement, and ends by naming what it did not cover. Inventory, status, changes and figures come from the tool. The content of a report, its finding or the reason for a decision made for the owner, is agents' text: the tool checks only its form, so it counts only with its evidence, and the briefing shows both.

## Two entries

### Briefing

The owner gets up, looks in, and sees what they may decide today, drops in ideas and sees how the factory runs. Every reader gets the briefing; "to decide" is the owner's alone.

| Part | Shows |
|---|---|
| **To decide** | Escalated reports, by importance, each with its prepared proposals and a recommendation |
| **Decided for you** | What was decided with the recommendation, with the reason; reversible by a report |
| **Ideas** | Drop one in; see how an idea would change the spec; discuss or take it |
| **Incidents** | Reports from outside, how they were answered, how it went and whether they came back |
| **Development** | What landed, what runs, which branches stayed and why |
| **How the factory runs** | Findings per area over time, the hit rate of control samples, contradictions of spot checks, jobs without progress |
| **Since you last looked** | What is new, gone, changed or suspect, with the old and the new text, and the report behind each change |

Why a change was made is in the report whose action it is; without one, the briefing says it was not recorded.

### Explain

Explain answers the whole, an area, a document or a question, only from the spec, with a citation for each statement, and otherwise says "not stated", as a competency question is answered ([§5](is-the-spec-good.md#the-spec-itself)). "Explain the whole" goes through the areas in the order of the template. A picture, such as the states of a concept, Claude draws only from what documents say, and names them below it.

## Comparing two states

`markitect diff` compares two folders, each with a spec, a state and reports, and names what is new, gone, changed or suspect, with the old and the new text. The tool uses no Git; the runner or the skill lays out the other state. The comparison carries three things: "since I last looked", the seen state against now; the preview of an idea, its branch against the main branch; and the development in the briefing.

A reader's last look is personal. It is a copy in their profile under `~/.markitect/`, outside the project and outside any one client.

## Views

Views are generic and come from the language, so they work for every spec: the topic view, a table per type, `show` with a document's artifacts and verdict, `status`, and the comparison ([§11](the-tool.md)). Everything a person sees is generated; nothing is kept by hand or stored in the repository. Views particular to a package come with a user interface.

There is no user interface for now. The spec is readable Markdown, the tool runs natively, and people ask in the chat. A user interface and clients other than Claude come once Markitect is proven ([§13](what-is-shown.md)).
