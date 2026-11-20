# Is the spec good?

**A good spec has five properties, and four sources show them** ([DEC-123](../decisions.md#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources)). The properties come from the characteristics of good requirements in ISO/IEC/IEEE 29148.

| Property | Means | Holds for | A bad sign |
|---|---|---|---|
| **Necessary** | Only what is fixed: someone relies on it, or it was decided. Nothing about how it is done | Each statement | A role whose responsibility reads "works conscientiously": nobody relies on that |
| **Complete** | Nothing fixed is missing; the executor never has to guess what counts outside | The whole spec | One step of a process names who may do it, a similar step does not |
| **Consistent** | No two statements exclude each other, a general rule and a single case included, and nothing asks more than the world allows. The writing rule is the best means | The whole spec | A rule says someone never works unsupervised, a task has them work alone after hours |
| **Unambiguous and verifiable** | Two readers understand the same, and a checker can say yes or no. Every statement is as strong as possible: general and without exceptions. A real exception is a statement of its own, not a clause | Each statement | "Only for simple cases": who decides what is simple? A quality without a measure |
| **Structured** | One thing per document. Each fact stands in its section and on the level it holds for. What has facts of its own, or is needed by several, is a document of its own that others link | Each document | Facts in the text before the first section; general planning rules inside one task |

A document should hold a part of its own when that part has facts of its own, is needed by several documents, changes at its own pace, or has a rule of its own. It should not for a value only that document relies on, or the spec falls apart into tiny documents. The type decides: where its section asks for a link, every document links.

**Not in the list.** *Correct*, whether the spec says what is meant, no reader can judge without the owner; it shows only in use. *Ranked* is a priority, which is work. *Traceable* is the state's ([§6](the-state.md)). *Modifiable* is in consistent and structured. Coupling, cohesion, abstractness and link density are no measures: folders carry no rules, a folder sorted by kind is not worse, and a high fan-in is natural for a role everyone names.

## Four sources

No source suffices alone, and the spec shows what is missing only as far as one asks it.

| Source | Shows mainly | Means |
|---|---|---|
| **The spec itself** | Necessary, consistent, unambiguous, structured; complete as far as the questions reach | Check, show, read and question, below |
| **Its changes** | Consistent, structured, unambiguous | Six signals from the history ([below](#its-changes)) |
| **What is built from it** | Complete, unambiguous, consistent, verifiable, necessary, and correct | Findings of builders, reviewers and counter-checks, and the owner looking at a result |
| **The questions it raises** | Complete, unambiguous, consistent, structured | Reports ([§9](reports-and-actions.md)) |

## The spec itself

Four steps, as a person who reviews a spec would take them:

1. **Check.** `check` finds what is technically wrong: titles, types, missing or empty sections and dead links ([§3](the-language.md#what-check-catches)). The type works as a checklist.
2. **Show.** The tool judges nothing; it gathers what has to be compared and says where reading pays off.
3. **Read.** A person or an agent judges the meaning against the five properties. Only a reader sees whether something is right.
4. **Question.** One asks the spec questions that a spec of its kind must answer, and sees how well it does. This is the one means that checks completeness on the spec itself.

**The topic view.** A topic is a document, a type or a folder. `markitect topics` shows it as one text with everything that belongs to it: what links to it, with the section the link stands in; the rules that hold for it, also through the README of a folder; and the places that name it without a link. The strong statements about the topic come first, since the single cases are checked against them. A table per type, `markitect table`, shows a column with the same value everywhere: that fact belongs a level higher.

**Signals** say where reading pays off. They have no fixed thresholds and give no warnings; each counts occurrences or compares with the rest of the spec, and the output is a sorted list for the whole spec or a topic. The reader decides.

| Group | Signal | Points at |
|---|---|---|
| Statements | **Weak statements:** conditions and exceptions ("except", "unless", "only if", "otherwise") and vague words ("reasonable", "usually", "etc.") | Unambiguous, consistent |
| | **Strong statements:** "every", "never", "only", against which a reader holds the single cases | Consistent |
| | **Complexity of a statement:** words, terms and conditions per statement | Unambiguous, structured |
| Structure | **Size:** statements of a document against documents of the same type | Structured |
| | **Text before the first section** of a typed document, where a fact usually sits in the wrong place or the type lacks a section | Structured |
| | **A recurring group of links:** the same three or more links in the same section of many documents | Structured: a missing group |
| | **Scattered rules:** one thing in many conditional statements across many documents, which belongs in one place | Structured, consistent |
| Connections | **Mention without a link:** a document's title in another's text, unlinked | Consistent, structured |

**Two references for every number** ([DEC-123](../decisions.md#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources)). A value is shown beside the reference value of its type, which ships with the package, and beside the rest of the project: "documents of this type usually have 200 to 400 characters; all of yours do, and this one has 900". Both are needed: when every document of a project is too long, nothing stands out within it. Reference values come from proven specs, those the owner names as a reference, and improve with each version of the package. Until there are enough, the package's values hold.

**Competency questions.** A spec is as good as it answers the questions it exists for. Valuable are questions across documents: "who may do this alone?", "what happens when someone fails to show up twice?". Each answer comes only from the spec, with a citation, and is rated answered, only inferred, two answers or not answered; anything without a citation is "not stated". The share of cleanly answered questions measures the spec by its purpose. Catalogs ship with packages and templates; a project's own questions lie beside the spec, not in it, as tests lie beside code. An answered question from use becomes a fixed question, a general one may move into the package, and a question stays while it finds something or the owner wants to keep it.

## Its changes

A good link catches a change: when each fact stands in one place, only that place changes. When a change reaches far all the same, a fact is scattered or misplaced. The history gives the data ([§6](the-state.md#the-history)); each signal is compared with the project's other changes, and mechanical changes, such as a `move`, do not count.

| Signal | Usually means | Points at |
|---|---|---|
| **Wide change:** one change touches far more documents than usual | A fact stands in several places | Consistent |
| **Change without a link:** two documents keep changing together with no link between them | A hidden dependency, a doubled fact or a missing concept | Consistent, structured |
| **Dragged links:** a document changes, and those that link to it change with it | They repeat its content instead of linking | Consistent |
| **Hotspot:** often changed and much linked | Stable and changing facts are mixed and belong apart | Structured |
| **Back and forth:** a statement changes and changes back | Something is not decided; a decision with its reason is missing | Unambiguous |
| **Weakening:** a strong statement gathers conditions and exceptions | The rule is watered down | Unambiguous and verifiable |

A high fan-in alone is no problem: a stable vocabulary may be linked much. Only much linked *and* often changed is costly.

## What is built, and the questions raised

Every finding names its place and the property it violates:

| What happens | Means | Property |
|---|---|---|
| The builder had to decide something someone will rely on | A gap | Complete |
| Builder and reviewer read the same place differently | Unclear | Unambiguous |
| Two places ask for different things | A contradiction | Consistent |
| The reviewer cannot say yes or no to a statement | Not verifiable | Verifiable |
| Nobody realizes or relies on a statement | Superfluous | Necessary |
| An answer was found only in an unexpected place | Misplaced | Structured |
| The owner sees the result and says it was not meant so | Wrongly stated | Correct |

Three signals follow: repeated findings at the same place show a bad place; a report closed without a change of the spec leaves the spec incomplete there; and each answered question becomes a fixed question. These findings are also the touchstone for everything above: whether the signals pointed at these places beforehand, whether a reference spec is really proven, and whether a project converges, with fewer findings per area over time.
