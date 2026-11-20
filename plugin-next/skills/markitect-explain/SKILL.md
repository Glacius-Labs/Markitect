---
name: markitect-explain
description: Explain a Markitect project's spec - the whole, an area, a document or a question - from the spec alone, each statement with the document it comes from, and "not stated" where the spec does not say. It changes nothing. Use when someone asks what the spec says, how an area works, who may do what, or why the project is as it is.
allowed-tools: Bash(markitect-next show *), Bash(markitect-next topic *), Bash(markitect-next table *), Bash(markitect-next questions)
---
Answer from the spec alone, not from the code or from what would be usual. The tool shows the spec; you explain it, each statement with the document it comes from, as a link to its file. Change no file of the project.

- **A document:** `markitect-next show DOCUMENT` gives its type, sections, links, what links to it, its artifacts and its verdict; `markitect-next topic DOCUMENT` the strong statements about it, what links to it with the section, and where it is named without a link.
- **An area,** a folder, or a type: `markitect-next topic TOPIC` prints it as one text with everything that belongs to it; `markitect-next table TYPE` every document of a type.
- **A question:** find the documents that bear on it, by title, type and text, and read each. Answer as a competency question is answered: answered, only inferred, two answers, or not answered. What has no citation is not stated; say "not stated" and fill nothing in. Where two documents contradict each other, show both. `markitect-next questions` lists the questions a spec of its kind should answer.
- **The whole:** go through the areas in the order of the template, for a software project purpose, context, domain, interfaces, quality and architecture, each by its README.

Draw a picture where it says more than words, such as the states of a concept, only from what documents state, and name them below it. What matters and is not stated is a question for the owner and the `markitect-next:markitect-architect` skill. End by naming what of the area or question you did not cover.
