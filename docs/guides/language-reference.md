# Language reference

The exact rules of the spec language as the tool implements them, and the commands of `markitect`. [The language](../concept/the-language.md) in the concept says what the language is for; this page says precisely what the tool reads and reports. The examples are a small lending library.

## Documents

A spec is a folder. Every file in it, at any depth, whose name ends in `.md` in any case is a document; a file named only `.md` is not, and neither is a folder whose name ends in `.md`. Other files are ignored.

- **Path.** A document is known by its path below the spec folder, with forward slashes, such as `lending/borrow.md`. Paths are compared exactly, case included, on every system.
- **Text.** A document is read as UTF-8. A leading byte order mark is dropped, and bytes that are not UTF-8 are replaced. Lines end with LF, CRLF or CR, and are numbered from 1.
- **Markdown.** A document is read as [CommonMark](https://spec.commonmark.org/) with GitHub's extensions, tables, task lists, strikethrough, autolinks and footnotes, as GitHub renders it. So code is what CommonMark calls code: fenced and indented code blocks, also inside lists, and code spans. Nothing in code or in raw HTML, comments included, is a heading, a type line, an extends line or a link, and a character escaped with a backslash is text.

## Title and sections

Titles and sections come from the headings at the top level of a document; a heading inside a list or a block quote is part of the text around it. Link reference definitions, `[rules]: lending/rules.md`, and footnote definitions are no blocks: they stand before no title and give no section text. Both kinds of CommonMark heading count: `# Borrow`, and `Borrow` underlined with `=====` (level 1) or `-----` (level 2). The text of a heading is as written, without closing `#`, and the lines of an underlined heading are joined by a space.

- **Title.** The first block of a document is a level-1 heading with text, the title. Any further level-1 heading is a second title.
- **Sections.** Each level-2 heading opens a section, which runs to the next level-1 or level-2 heading. Every block in between, level-3 to level-6 headings, code and rules included, belongs to it. A section has text when one of these blocks shows something: a character other than white space outside HTML comments, code, a rule, an image or a link. A section that holds only white space and comments, such as a template's `<!-- fill in -->`, is empty.

## Links

A link is a Markdown link outside code and raw HTML: an inline link, `[text](destination)`, optionally with a title, `[text](destination "title")`, or a reference link, `[text][label]`, whose label is defined in the document. White space inside the parentheses is allowed, `[Loans]( loans.md )`, and a destination in `<` and `>` may hold spaces, `[Loans](<lending/open loans.md>)`. A link holds no link: in `[a [b](b.md)](a.md)` only `b.md` is a link. A link in a footnote counts when the footnote is referred to; GitHub shows no other. An image, `![scan](card.md)`, and an autolink are no link.

A link counts when its destination, without its `#section` and after decoding, ends in `.md` in any case and does not start with a scheme such as `https:`, `mailto:` or `C:`. Web addresses, other files and a bare `#section` are not checked.

- **Resolution.** The path is relative to the folder of the linking document. Backslash escapes and character references, named and numeric such as `&#46;` and `&#x2E;`, are resolved, and percent-encoding is decoded, so `open%20loans.md` names `open loans.md`; a `%` that is not followed by two hexadecimal digits stays as it is. A path that starts with `/`, or that leaves the spec folder with `..`, names no document.
- **Sections.** A link with a section, `[Card](members/ada.md#card)`, points to the document; the section is not checked.
- **Every counted link must point to a document of the spec**, by its exact path.

## Types and the type line

A **type** is a document that another document names in its type line or its extends line, or a document with an extends line that is one link, whether or not its target exists ([below](#extending-a-type)); nothing else marks it. Its text before the first section says what the type is, and its level-2 headings declare the sections of its documents.

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
- **Declared twice.** A name declared more than once, in any case, is one section. It is optional only when every declaration is.

The **type line** is the first line of the block right after the title, when that block is a paragraph and the line starts with exactly `Type:`. What follows `Type:`, without surrounding white space, is exactly one inline link that counts as a link above, with an optional title and `#section`. Any other line of text that starts with `Type:`, also in a list or in a document without a title, is a misplaced type line; a line in code is not. The links of a type line, placed or misplaced, are checked only as the type.

```markdown
# Loan 4711

Type: [Loan](../types/loan.md)

## Borrower
[Ada Lovelace](../members/ada-lovelace.md)

## Due
1 March
```

A document with a type has each required section of its type as a level-2 heading of its own, in any order. A heading matches a declared section when the two are equal without regard to case, by Unicode simple case folding; when the document has two such headings, the first counts. An optional section is written without the marker, `## Renewals`, or left out. Every declared section the document has, required or optional, has text. The document may add sections of its own.

### Extending a type

A type may extend another, its **base**, with an **extends line** ([DEC-155](../decisions.md#dec-155-a-type-extends-another-with-a-line-of-its-own)). The extends line is read exactly as the type line: the first line of the block right after the title, when that block is a paragraph and the line starts with exactly `Extends:`, followed by exactly one inline link that counts as a link, with an optional title and `#section`. Any other line of text that starts with `Extends:`, anywhere in the document, is a misplaced extends line; a line in code is not. The links of an extends line, placed or misplaced, are checked only as the base. An extends line that is not exactly one link is reported, and the document is read as if it had no such line.

```markdown
# Renewable loan

Extends: [Loan](loan.md)

A loan that the member may extend.

## Renewals
How often the loan was renewed.

## Limit
How often it may be renewed.
```

- **One of the two lines.** A document has a type line or an extends line, never both. When both stand in the first block after the title, the first of them counts and the second is an error. A document with an extends line is a type and is never checked as a document of its base, so a type that no document names yet is not mistaken for one. `Type:` passes nothing on: a type that has a type line of its own, as a document of another type, inherits none of its sections.
- **The chain.** A type's chain is the type, its base, its base's base and so on. The sections of a type are those declared along its chain. A section declared at more than one level, in any case, is one section, as one declared twice in a type is: required when any level requires it, and described by the most derived level that declares it. The sections stand in chain order: the base's first, then what each derived type adds.
- **Never looser.** A derived type may describe a section of its base more precisely or make an optional one required, `## Renewals` above. It never marks optional a section that a type up its chain requires.
- **No cycle.** A chain that comes back to itself is an error at each type in the cycle; a type whose chain only leads into a cycle is not. A document of such a type is checked against the sections its chain collects before it would loop.

A document of a derived type has each required section of the type's chain, and every declared section of the chain it has has text, by the rules above. A document of `Renewable loan` needs `Borrower`, `Due`, `Renewals` and `Limit`.

## What check reports

| Code | Reported when | At line |
|---|---|---|
| `title` | The first block is not a level-1 heading with text: `missing level-1 title`. Otherwise, a further level-1 heading: `second level-1 heading` | The first block, or 1 for a document without blocks; the first further level-1 heading |
| `type.line` | The type line is not exactly one counted link: `type line must be exactly one link to a document`. A misplaced type line: `a type line stands only on the first line after the title`. A type line after an extends line in the first block: `a document has a type line or an extends line, not both` | That line |
| `type.unresolved` | The type line's link points to no document: `no document at "PATH"`. Its sections are then not checked | The type line |
| `extends.line` | The extends line is not exactly one counted link: `extends line must be exactly one link to a document`. A misplaced extends line: `an extends line stands only on the first line after the title`. An extends line after a type line in the first block: `a document has a type line or an extends line, not both`. The document is then read as without that line | That line |
| `extends.unresolved` | The extends line's link points to no document: `no document at "PATH"` | The extends line |
| `extends.cycle` | The chain comes back to itself: `the chain of extends lines comes back to itself`, once at each type in the cycle; a type that only leads into a cycle reports nothing | The extends line |
| `extends.loosens` | A derived type marks optional a section that a type up its chain requires: `section "NAME" is required by TYPEPATH`, the nearest type up its chain that requires it | The derived type's heading of that section |
| `section.missing` | A required section of the type's chain is missing: `missing section "NAME" required by TYPEPATH`, the most derived type of the chain that requires it | The type line |
| `section.empty` | A declared section of the type's chain that the document has holds no text: `empty section "NAME"` | The document's heading of that section |
| `link.unresolved` | A counted link points to no document: `no document at "PATH"`, with the path as written, without its section | The link's first line |

A document reports each unresolved target once, at its first link: `[the archive](archive.md)`, `[its rules](archive.md#rules)` and `[again](./archive.md)` are one report. Each declared section is reported at most once, and a document at most one `title`. The diagnostics are sorted by path, then by line.

## Commands

Every command but `version` and `licenses` works in a project and finds its spec through it: the folder `-C PROJECT` names, or the nearest folder from the working directory upward that holds `.markitect/spec` ([project reference](project-reference.md#the-project)). A document, type, topic or folder named on the command line is a path below the spec folder, with or without `.md`, with `/` or `\` between folders; a path that leaves the spec folder, is absolute or starts with a scheme names no document. The `OLD` and `NEW` of `diff` are project folders.

A flag may stand anywhere, before the command as well, as in `markitect -C PROJECT check`, and `-json` is the same as `--json`. `--all` is a flag of `topic` only. `--` ends the flags, so that a name starting with `-` can follow. `markitect help`, or `-h` or `--help` wherever it stands, even after a command, prints the general usage, which groups the commands as this table does; the usage of one command is printed with a wrong number of arguments. A flag that no command has, or one that lacks its value, is an error that names it. Output paths are below the spec folder, with `/`.

| Group | Command | Does |
|---|---|---|
| Look | `markitect show DOCUMENT` | The document's title, its type or, for a derived type, its base, its sections (a type's those of its whole chain, with their `(optional)` marker and, for an inherited one, the type that declares it), the documents it links to, the documents that link to it with the section each link stands in, and for a type the types that extend it and its documents, those of the types that extend it included |
| | `markitect table [TYPE]` | Every document of the type and of every type that extends it as a Markdown table: the title, then one column per section of the type's chain. Without a type, every document with its type and title |
| | `markitect topic [TOPIC] [--all]` | The signals of the spec, or the topic view of a document, a type or a folder ([signals](signals.md)) |
| | `markitect questions` | The question catalogs of the packages of the default that the spec has ([signals](signals.md#question-catalogs)) |
| | `markitect diff OLD [NEW]` | What is new, gone or changed in the spec of NEW, by default this project, against the project OLD, with a diff for what changed ([project reference](project-reference.md#diff)) |
| Check | `markitect check` | Checks the spec by the rules above |
| Tool | `markitect version` | The tool's version and build ([project reference](project-reference.md#version)) |
| | `markitect licenses` | The license notices of the modules the tool is built from |

The commands only read: none writes a file, keeps a state, starts an agent or steers work, and each does one thing on its own.

**Exit codes.** 0 when all is well, 1 when `check` or `diff` found something, and 2 when the command could not run: an unknown command or flag, a flag without its value, no command at all, a wrong number of arguments, no project, a `.markitect/spec` that cannot be read or names a spec folder it may not, a spec folder that does not exist, a document, type or topic that the spec does not have, or a file that cannot be read.

### Commands that read

**`check`** prints one line per diagnostic, `path:line: code: message`, then a summary such as `2 errors in 9 documents`:

```text
loans/borrow.md:11: link.unresolved: no document at "fines.md"
roles/member.md:5: section.empty: empty section "Does"
2 errors in 9 documents
```

**`show`** prints the document's path, `Title:`, `Type:` (the type's path, `(none)`, `(invalid line)` for a type line that is not one link, or the path with `(missing)`), or for a document with an extends line `Extends:` (the base's path, or the path with `(missing)`), then the lists `Sections:`, `Extended by:`, the types that extend it directly, sorted, only for a type that has some, `Links to:`, with `(missing)` after a path that names no document, and `Linked from:`, each `(none)` when empty, and `Documents:` only for a type that has some, with the documents of the types that extend it, directly or further down. A derived type's `Sections:` are those of its whole chain in chain order, an optional one with its marker and an inherited one with the type that declares it: `Borrower (from types/loan.md)`. A section the derived type declares again is its own and stands in its base's place. `Links to:` holds each path once; `Linked from:` holds one entry per linking document and section, `path (Section)`, or the bare path for a link that stands before any section, so a document that links from two sections appears twice. Both are sorted without regard to case.

**`table`** prints a row per document of the type and of every type that extends it, directly or further down, in path order, with one column per section of the type's own chain. A cell holds the first line of the section's text as plain text: markup, list markers, HTML tags and comments are dropped, and links, reference links and images show as their text. A section that starts with code shows its first code line, one that starts with a table its first row, cells separated by ` | `. `|` is escaped, and a cell is empty when the section has no text or the document leaves an optional section out. A row's title is the document's title, or its path. Without a type, `table` prints every document of the spec in path order as `| Document | Type | Title |`: its path, the path of its type line's type, empty without one, as for a derived type, and its title.

### JSON

With `--json`, every command prints one document that starts with `format_version` (1) and `kind`, the command's name. Later fields join without changing these.

| Kind | Fields |
|---|---|
| `check` | `documents`, the number of documents of the spec, and `diagnostics`, a list, empty when nothing is found, of objects with `file`, `line`, `code` and `message` |
| `show` | `path`, `title`, `type`, `type_state` (`none`, `invalid`, `missing` or `ok`), `extends` and `extends_state` (`missing` or `ok`) for a document with an extends line, whose `type` is then empty and `type_state` `none`, `sections` with `heading`, the marker included, and `declared_by` for an inherited one, `extended_by`, `links_to` with `path` and `missing`, `linked_from` with `path` and `section`, and `documents`; an empty `section`, `extends`, `extends_state`, `declared_by` or `extended_by` and a `missing` that is false are left out |
| `table` | `type`, `columns`, and `rows` with `path`, `title` and `cells`; without a type, `type` is empty and the one column is `Type` |
| `version`, `diff` | As the [project reference](project-reference.md) says |
| `licenses` | `notices` |
| `topic`, `questions` | As [signals](signals.md#commands) says |
