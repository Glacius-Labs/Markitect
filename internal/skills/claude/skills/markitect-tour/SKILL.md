---
name: markitect-tour
description: Walk a person through a project's spec, chapter by chapter in the order a spec is designed - why the system exists and its business case, how each Stakeholder uses it, its domain, its views, and how it runs - for someone new to the project, or the owner after a long break. The reader may ask questions at any point. It changes nothing. Use when the user asks for a tour, an overview or an introduction to the spec or the project, or to be walked through it.
allowed-tools: Bash(markitect list), Bash(markitect list *), Bash(markitect doc *), Bash(markitect status), Bash(markitect status *), Bash(markitect vet), Bash(markitect vet *), Bash(markitect inbox)
---
Lead the user through the spec in `.markitect/spec/`, from what `markitect` prints. Change no file of the project. If the user names a chapter or an area to start with, start there.

**The chapters.** `markitect list --json` names every Definition with its file. Each top-level folder of the spec belongs to one chapter, in this order:

1. **Why:** `project.md`, `purpose/` and `context/`: the problem and today's way, each Stakeholder with what they gain and what they give, the Goals, the Conflicts, the Alternatives, what is left out, and the world around the system.
2. **Use:** `scenarios/` and `interfaces/`: for each Stakeholder, their Scenarios told as stories, with the pages, routes and messages they meet on the way.
3. **Domain:** `domain/`: what the system keeps, the states a thing goes through, what can be done with it, what it does by itself, and its Rules.
4. **Views:** `architecture/`, with the context (the Stakeholders and External systems around the system), the runtime (its Components and the Connectors between them), the modules (the Packages and what each depends on, from `markitect list --deps`), the deployment, the strategy, the decisions and the Rules across the code.
5. **Running it:** `quality/` and `administration/`: how well it must work, and what the person who runs it does.
6. **The rest:** each top-level folder no chapter above names, under its own name.

A chapter whose folders the spec does not have is a gap: say so in one line and go on.

**Each chapter.** Read its folders with `markitect doc --read FOLDER`. Tell the chapter as a story in plain words, the important first, and give each statement with the Definition it comes from, as a link to its file. Where the spec leaves something open that the chapter would need, say so; do not fill it in. End the chapter with a short index of what it holds that the story did not tell, by Kind, each Definition by its link, so that every Definition of the chapter is named; a part counts with its whole. Then ask whether to go on. Answer a question at any point from the spec as the markitect-explain skill does, then go back to the tour.

**Pictures.** Draw one where it says more than words: the Stakeholders and External systems around the system, a Scenario as a storyboard of what the Actor does and sees, the states of a Concept with the Operations that move it, the Components and Connectors at runtime, the Packages with their dependencies. Draw only what Definitions state, name each element as the spec names it, and say below it which Definitions it shows. Use what the session can show, such as a Mermaid diagram in the answer or a page.

**The end.** Say how the project stands, from `markitect status`: what is not verified yet, the spec-problems that wait for the owner, and what waits in the inbox, as `markitect inbox` lists it. Name each top-level folder and each Definition of `markitect list --json` that no chapter covered, or say that none was left. Then offer to remember this state as seen, as the markitect-catchup skill does in `.claude/skills/markitect-catchup/SKILL.md`, so that the next catch-up starts here.
