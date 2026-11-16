---
name: markitect-explain
description: Explain a part of a project's spec - one thing, an area or a question - from the spec alone, each statement linked to the Definition it comes from, and say plainly what the spec does not state. It changes nothing. Use when the user asks what the spec says about something, how an area or a thing works, who may do what, or why the project is as it is.
allowed-tools: Bash(markitect list), Bash(markitect list *), Bash(markitect doc *), Bash(markitect status), Bash(markitect status *), Bash(markitect vet), Bash(markitect vet *)
---
Answer from the spec in `.markitect/spec/`, not from the code or from what would be usual. Change no file of the project.

- **A thing,** such as an Operation, a Concept or a Rule: `markitect doc TARGET` shows it whole, with its parts, what it links to, what links to it, what binds it, the files that realize it and its verdict. Say what it states, what it relies on and what relies on it.
- **An area,** a folder or a Package: `markitect doc --read FOLDER --bound` prints every Definition at or below the folder, what binds them from outside it and what outside it they link. Explain each Definition of the folder; a part counts with its whole. `markitect list --deps` shows what the area's Packages depend on.
- **A question:** find the Definitions that bear on it in `markitect list --json`, by title and Kind, and by searching the files under `.markitect/spec/`, and read each with `markitect doc DEFINITION`. Say which Definitions you read.

Give each statement with the Definition it comes from, as a link to its file. Where the spec does not state what the user asks, say so plainly and do not fill it in: if it matters, it is a question for the owner and the markitect-architect skill. Where two Definitions contradict each other, show both. If the user asks what the code does, say first what the spec states; the code is another question.

Draw a picture where it says more than words, such as the states of a Concept with the Operations that move it, a Scenario as a storyboard, or the Components and Connectors at runtime. Draw only what Definitions state, name each element as the spec names it, and say below it which Definitions it shows.

After an area, name each Definition of it that you did not explain, or say that none was left. Then ask whether to go deeper somewhere.
