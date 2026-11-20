# Language reference

The exact rules of the spec language as the rebuilt tool implements them, and the commands of `markitect-next`, the rebuilt tool while it grows beside `markitect` ([roadmap](../roadmap.md#current-steps)). [The language](../concept/the-language.md) in the concept says what the language is for; this page says precisely what the tool reads and reports. The examples are a small lending library.

## Documents

A spec is a folder. Every file in it, at any depth, whose name ends in `.md` in any case is a document; a file named only `.md` is not, and neither is a folder whose name ends in `.md`. Other files are ignored.

- **Path.** A document is known by its path below the spec folder, with forward slashes, such as `lending/borrow.md`. Paths are compared exactly, case included, on every system.
- **Text.** A document is read as UTF-8. A leading byte order mark is dropped, and bytes that are not UTF-8 are replaced. Lines end with LF, CRLF or CR, and are numbered from 1. A line has content when it holds a character other than white space.
- **Fenced code.** A line with at most three spaces and then three or more backticks or tildes opens a fenced code block; text may follow the run. The block ends at a line that, without its surrounding white space, consists only of the same character, at least as many times, or at the end of the document. Both fence lines belong to the block. Nothing in a block is a heading, a type line or a link.
- **Code spans.** On one line, a run of backticks opens a code span that ends at the next equal run, with at least one character between. When the whole run never recurs, a shorter run from the same start is tried, down to one backtick. A run that closes nowhere is text. Nothing in a code span is a link.
- **Headings.** A heading is a line outside fenced code with at most three spaces, one to six `#`, a space or tab, and text that starts with a character other than white space. Its text is the rest of the line without trailing spaces and tabs; closing `#` stay part of it, so `## Card ##` is the heading `Card ##`. An underlined heading, `Title` above `=====`, is no heading.

## Title and sections

- **Title.** The first line with content is a level-1 heading, `# Borrow`, the title. Its line is not searched for links. Any further level-1 heading is a second title.
- **Sections.** Each level-2 heading opens a section, which runs to the next level-1 or level-2 heading. Every line in between, level-3 to level-6 headings and code included, is its text. A section has text when one of these lines has content.

## Links

A link is an inline Markdown link on one line, outside code: `[text](destination)`, optionally with a title in double or single quotes after white space, `[text](destination "title")`.

- **Text.** The text in brackets may hold brackets one level deep: in `[the [new] rules](rules.md)` the link is `rules.md`, and nothing inside its text is a link.
- **Destination.** The destination is either a run without white space and without `)`, or enclosed in `<` and `>`, which may hold spaces: `[Loans](<lending/open loans.md>)`.
- **Not links.** An image, `![scan](card.md)`, is no link, nor are reference links, `[text][label]`, and autolinks.

A link counts when its destination, without its `#section` and after decoding, ends in `.md` in any case and does not start with a scheme such as `https:`, `mailto:` or `C:`. Web addresses, other files and a bare `#section` are not checked.

- **Resolution.** The path is relative to the folder of the linking document. Percent-encoding is decoded, so `open%20loans.md` names `open loans.md`; a `%` that is not followed by two hexadecimal digits stays as it is. A path that starts with `/`, or that leaves the spec folder with `..`, names no document.
- **Sections.** A link with a section, `[Card](members/ada.md#card)`, points to the document; the section is not checked.
- **Every counted link must point to a document of the spec**, by its exact path.

## Types and the type line

A **type** is a document that another document names in its type line; nothing else marks it. Its text before the first section says what the type is, and its level-2 headings declare the sections of its documents.

```markdown
# Loan

One copy lent to one member.

## Borrower
The member, as a link to a member.

## Due
The date the copy is due back.

## Renewals (optional)
How often the loan was renewed.
```

- **Optional.** A heading that ends with `(optional)`, in any case, declares an optional section. Its name is the heading without the marker and the white space before it: `Renewals`. Every other declared section is required.

The **type line** is the first line with content after the title, when it starts with exactly `Type:` at the beginning of the line and is not in code. What follows `Type:`, without surrounding white space, is exactly one link that counts as a link above, with an optional title and `#section`. A document without a title has no type line, and a line starting with `Type:` anywhere else is ordinary text whose link is checked as a link. The type line's own link is checked only as the type.

```markdown
# Loan 4711

Type: [Loan](../types/loan.md)

## Borrower
[Ada Lovelace](../members/ada-lovelace.md)

## Due
1 March
```

A document with a type has each required section of its type as a level-2 heading of its own, in any order. A heading matches a declared section when the two are equal without regard to case, by Unicode simple case folding; when the document has two such headings, the first counts. An optional section is written without the marker, `## Renewals`, or left out. Every declared section the document has, required or optional, has text. The document may add sections of its own.

## What check reports

| Code | Reported when | At line |
|---|---|---|
| `title` | The first line with content is not a level-1 heading: `missing level-1 title`. Otherwise, a further level-1 heading: `second level-1 heading` | The first line with content, or 1 for a document without content; the first further level-1 heading |
| `type.line` | The type line is not exactly one counted link: `type line must be exactly one link to a document` | The type line |
| `type.unresolved` | The type line's link points to no document: `no document at "PATH"`. Its sections are then not checked | The type line |
| `section.missing` | A required section of the type is missing: `missing section "NAME" required by TYPEPATH` | The type line |
| `section.empty` | A section of the type that the document has holds no text: `empty section "NAME"` | The document's heading of that section |
| `link.unresolved` | A counted link points to no document: `no document at "PATH"`, with the path as written, without its section | The link's line |

A document reports each unresolved path once, at its first line: `[the archive](archive.md)` and `[its rules](archive.md#rules)` are one report. A document reports at most one `title`. The diagnostics are sorted by path, then by line.

## Commands

`markitect-next` has one command so far. Flags may stand before or after the folder.

| Command | Does |
|---|---|
| `markitect-next check SPECFOLDER [--json]` | Checks the spec in `SPECFOLDER` by the rules above |

**Text output.** One line per diagnostic, `path:line: code: message`, with the path below the spec folder, then a summary such as `2 errors in 14 documents`:

```text
lending/borrow.md:7: link.unresolved: no document at "loans/ghost.md"
lending/loans/4711.md:9: section.empty: empty section "Due"
2 errors in 14 documents
```

**JSON.** With `--json`, one document: `format_version` (1), `kind` (`check`), `documents`, the number of documents, and `diagnostics`, a list, empty when nothing is found, of objects with `file`, the path below the spec folder, `line`, `code` and `message`.

**Exit codes.** 0 when nothing is found, 1 when check found something, and 2 when the command could not run: unknown command or flag, not exactly one folder, a folder that does not exist or a file that cannot be read.
