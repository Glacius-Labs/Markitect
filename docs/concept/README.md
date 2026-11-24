# Concept

**You architect the project in a typed, declarative specification language in Markdown. Markitect is that language, the default that ships with it, and a tool that checks the spec, analyses it and helps reading it.**

This page is the one description of the product, and the target that Main builds ([DEC-140](../decisions.md#dec-140-one-concept-describes-the-product)). The [decisions](../decisions.md) list what it rests on, the [roadmap](../roadmap.md) the next steps and where Main is, and the [vision](../vision.md) why.

## The aim

Make one clean and simple version that works reliably ([DEC-120](../decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)): a spec that agents can realize and against which their work can be checked without anyone reading it, so that people concentrate on architecture and on the decisions only they can make.

**The problem.** When agents build a project, trust and overview no longer come with the work, and nothing in the project states what it should be; the [vision](../vision.md#why-markitect-exists) gives the reasoning. Markitect supplies the means for that statement: one spec with authority over every artifact of the project ([§2](what-markitect-is.md)). On it, Markitect answers four questions:

| Question | Markitect's answer |
|---|---|
| Is the spec sound in itself? | `check` finds what is technically wrong: titles, types, missing or empty sections, references and values that do not fit their property, dead links, links into what a module keeps internal and cycles between modules ([§3](the-language.md#what-check-catches)) |
| Is the spec good? | Signals say where reading pays off, and competency questions measure the spec by its purpose; a reader judges ([§5](is-the-spec-good.md)) |
| What is the project? | The spec, shown by the tool and explained by an agent with a citation for each statement ([§6](the-tool.md#reading-the-spec)) |
| What changed? | The repository's ordinary diff, like every change, and `diff` between two states of a project: the documents of the spec, and for each module whether its spec, its files or both changed, reported and not judged ([§6](the-tool.md#reading-the-spec)) |

## What Markitect leaves to the project

The way a project works is the project's own ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)). How what happens is noted, how planning goes, which meetings there are, what a story or a report is, how work is reviewed and accepted and where the work lies belong to the project's own way of working. The project describes it in its own spec, with general types, and skills, agents and scripts realize that spec, as code realizes the product. The default ships a way of working to start from, which the project copies and then owns like any part of its spec ([§7](the-project.md#the-way-of-working)).

## The other pages

2. [What Markitect is](what-markitect-is.md)
3. [The language](the-language.md)
4. [What goes in a spec](what-goes-in-a-spec.md)
5. [Is the spec good?](is-the-spec-good.md)
6. [The tool](the-tool.md)
7. [The project](the-project.md)
8. [What is shown and what is not](what-is-shown.md)
