# Signals and the topic view

How the rebuilt tool `markitect-next` shows where reading a spec pays off: the statements it reads, the marker words, the eight signals of [the spec itself](../concept/is-the-spec-good.md#the-spec-itself), the topic view, the reference values and the question catalogs ([DEC-123](../decisions.md#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources)). The tool judges nothing and has no thresholds: each signal is a list, strongest first, and the reader decides. [The language reference](language-reference.md) gives the rules of the language and the other commands; the [library](library.md) holds the reference values and the catalogs.

## Commands

| Command | Does |
|---|---|
| `markitect-next topic` | The signals of the whole spec. The text shows the first ten findings of each signal; `--all` shows every one, and `--json` always does |
| `markitect-next topic TOPIC` | The topic view of a document, a type or a folder, with every finding |
| `markitect-next questions` | The question catalogs of the library packages the spec has in `packages/` |
| `markitect-next figures` | The signals of the spec's changes, computed from the history ([below](#the-change-signals)) |

A finding reads `path:line: signal: detail`. With `--json`, `topic` prints `topic`, `topic_kind` (`spec`, `document`, `type` or `folder`), `documents`, `strong`, `linked_from` and `mentions` as lists of statements with `file`, `line`, `section` and `text`, and `signals`, each signal's list of findings with `file`, `line` and `detail`; for a type also `typed_documents`. `questions` prints `packages`, each with `package` and `questions`.

## Statements

A statement is the unit of every signal: a sentence of a paragraph, a list item, or a table body row. The tool reads the documents as CommonMark ([language reference](language-reference.md#documents)) and takes the text as plain text, with links and images as their text.

- **Sentences.** A paragraph splits after `.`, `!` or `?`, together with any further `.!?` and closing quotes or brackets, when white space or the end follows. It does not split after `e.g.`, `i.e.`, `etc.` or `vs.`, nor inside a number such as `2.5`.
- **Whole units.** A list item is one statement, whatever its sentences; its nested lists are statements of their own. A table body row is one statement, its cells joined by ` | `; the header row is none. A block quote's paragraphs and lists are statements like any other.
- **Not statements.** Headings, the type line and any other line starting with `Type:`, code, HTML, link reference definitions and footnotes. In a type, the text under its sections tells the writer what belongs there and is no statement of the spec; only its text before the first section, which says what the type is, counts.
- **What a statement keeps.** Its document, its section, or none before the first section, its line, its words, the documents its links point to, and its markers.
- **Words** are runs of letters and digits, joined inside a word by a hyphen or an apostrophe, compared in lower case. `user-friendly` is one word; `e.g.` is two.

## Markers

Markers are English words and phrases, matched as whole words without regard to case. At each word the longest phrase of all three lists counts, and its words count for nothing else: in "only if" the condition counts, not the strong word "only", and in "if possible" the vague phrase, not the condition. Other languages are not supported yet.

| Kind | Words and phrases |
|---|---|
| **Conditions** | if, unless, except, only if, otherwise, provided that, as long as, in case |
| **Vague words**, after the terms to avoid of ISO/IEC/IEEE 29148 | reasonable, appropriate, adequate, sufficient, good, usually, normally, generally, typically, often, sometimes, as needed, as appropriate, as applicable, if possible, etc., and so on, user-friendly, easy, simple, fast, flexible, efficient, approximately, about, several, some, many, few, best, minimal, maximal |
| **Strong words** | every, each, all, always, never, only, none, no one, nobody, must not |

## The eight signals

Each signal is a list sorted strongest first; equal ones stand in path and line order. A document *refers to* another when one of its statements links to it, or names it by its title, as the last row and the paragraph after the table describe.

| Signal | Name | Lists | Strongest first |
|---|---|---|---|
| Weak statements | `weak` | Every statement with conditions or vague words, with the words found | Most conditions and vague words |
| Strong statements | `strong` | Every statement with strong words. Its subject is what it links to or names, or its own document when it does neither | Most documents that link to or name its subject |
| Complexity of a statement | `complexity` | Every statement with its words, beside the project's median and maximum and the reference of its document's type, its terms, the documents it links to, and its conditions | Most words, then conditions, then terms |
| Size | `size` | Every document with a type: its statements, beside the median, range and number of documents of the same type and the type's reference | Highest ratio to the median of its type |
| Text before the first section | `text-before-sections` | Every document with a type that has statements before its first section, after the type line, with their number | Most statements |
| Recurring link group | `link-group` | The same three or more link targets under the same section heading, compared without regard to case, in two or more documents. The groups are the intersections of the sections' targets, of any number of documents, so a group shared by many is found even where each pair shares more. Only the largest group with the same documents is listed | Most documents, then most links |
| Scattered rules | `scattered-rules` | Every document that conditional or restricting statements of other documents refer to, with those documents. A statement is restricting when it holds one of the strong words only, never, must not, no one, nobody or none, or "not" followed by "alone" within three words | Most other documents |
| Mention without a link | `unlinked-mention` | Every pair of a document and another document whose title stands in its statements while it has no link to it anywhere, the type line included | Most mentions |

**Mention without a link.** A title is found as whole words, without regard to case; at each word the longest title counts. One-word titles shorter than four letters do not count, nor do the titles of types, which stand in prose as ordinary words, such as "the role". A document does not mention itself.

**Two references.** Where a signal shows a number, it shows the project's values beside it, and the reference value of the document's type when the type is a type of a library package: `packages/software/operation.md` matches the `software` package's `operation`. A project's own types have no reference value and show only the project's values.

## The topic view

A topic is a document, a type, which is a document that another names in its type line, or a folder below the spec folder. `topic` prints one text:

1. **Strong statements about it** from other documents: those that link to it or name it. They come first, since the single cases are read against them.
2. **The topic itself:** a document's text; a type's text and its documents as the [`table`](language-reference.md#commands-that-read); a folder's documents in path order, its README among them.
3. **Linked from:** every statement of another document that links to it, with its section. A link outside a statement, such as one in a heading, does not appear here; `show` lists it.
4. **Mentioned without a link:** every statement of another document that names it while that document has no link to it.
5. **Signals:** every finding that concerns a document of the topic.

A link to the README of a folder that holds a topic document counts as a link to the topic, so a rule that applies to a folder, and every folder above it, shows for each document in it. Only READMEs of folders below the spec root count this way: a link to the root's `README.md` concerns that document alone.

## Reference values

The reference values ship inside the tool, one file per package in the library, `references/PACKAGE.md`, and are not copied into a spec. Each holds a Markdown table with a row per type: its name, the statements per document and the words per statement, each as low–high. Every type starts with 1–5 statements per document, since a document of more than a few sentences is too big, and 5–25 words per statement. These are first values, to be replaced by values measured on proven reference specs, those the owner names as a reference.

## Question catalogs

Each package has a catalog of competency questions in the library, `questions/PACKAGE.md`: five to ten questions across documents that a spec built on that package should answer, generic and in English. `questions` prints the catalogs of the library packages the spec has. A template may bring a catalog of its own; the shipped templates rely on the catalogs of their packages. Answering them, each from the spec alone with a citation, is the work of a job, and a project's own questions beside the spec come with the layout of the project.

## The change signals

`markitect-next figures` computes the signals of [the spec's changes](../concept/is-the-spec-good.md#its-changes) from the history ([project reference](project-reference.md#the-history)), against the spec as it is. Like the signals of the spec itself, each is a list sorted strongest first, with the project's own reference beside it, and no threshold; the text shows the first ten findings of each, `--all` every one. The figures of reports and control samples join it when step 4 brings them.

**A change.** A verdict event's document, together with the documents its `changed` list names, is one change. A verdict joins the change before it when it follows that change's last verdict within an hour and its documents are among those of the change's first verdict: a reviewer confirming the documents of one change one after the other makes one change, not several. A verdict an hour or more later, or one that names a document the change did not have, starts a new change. The history follows a document through its moves to its current path. `move` events are no changes, and the documents a move rewrote count only when a verdict names them.

The `changed` list holds every document `status` lists as changed, also one recorded `fail`, one whose artifact changed and one gone from the spec. Such a document stays in the list of every later verdict until it is confirmed, and so counts in every change made meanwhile.

| Signal | Name | Lists | Strongest first |
|---|---|---|---|
| Wide change | `wide-change` | Every change, with its documents, beside the median and range of the project's changes | Most documents |
| Change without a link | `unlinked-change` | Every pair of documents of the spec that changed together in two changes or more while no link joins them in either direction | Most changes together |
| Dragged links | `dragged-links` | Every document of the spec whose changes came, two times or more, with changes of documents that link to it, with those documents | Most such changes |
| Hotspot | `hotspot` | Every document of the spec that changed and that documents link to: how often it changed, how many link to it, and the product, beside the medians of both | Highest product |
| Back and forth | `back-and-forth` | Every document whose text returned to an earlier version, by the hashes of its own verdicts and of the `changed` lists | Most returns |
| Weakening | | Not computed: a strong statement gathering conditions shows only in its old and new text, which an agent reads from Git; the history holds hashes | |

The time of an event is the clock of the machine that wrote it, so a wrong clock can reorder events and shift the changes.

```text
### hotspot (1)

reference: changes per document: median 1; documents linking to one: median 0.5
lending/return.md: changed 2 times × linked from 1 document = 2
```

With `--json`, `figures` prints `verdicts` and `moves`, the events read, `changes` with `time`, `docs` and `events`, `signals`, each signal's list with `docs`, `count` and `detail`, `reference` per signal, `weakening`, and `history`, the lines of the history it cannot read, which the text shows as notes. It exits 0.
