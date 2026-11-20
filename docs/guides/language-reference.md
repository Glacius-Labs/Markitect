# Language reference

The exact rules of the spec language as the rebuilt tool implements them, and the commands of `markitect-next`, the rebuilt tool while it grows beside `markitect` ([roadmap](../roadmap.md#current-steps)). [The language](../concept/the-language.md) in the concept says what the language is for; this page says precisely what the tool reads and reports. The examples are a small lending library.

## Documents

A spec is a folder. Every file in it, at any depth, whose name ends in `.md` in any case is a document; a file named only `.md` is not, and neither is a folder whose name ends in `.md`. Other files are ignored.

- **Path.** A document is known by its path below the spec folder, with forward slashes, such as `lending/borrow.md`. Paths are compared exactly, case included, on every system.
- **Text.** A document is read as UTF-8. A leading byte order mark is dropped, and bytes that are not UTF-8 are replaced. Lines end with LF, CRLF or CR, and are numbered from 1.
- **Markdown.** A document is read as [CommonMark](https://spec.commonmark.org/) with GitHub's extensions, tables, task lists, strikethrough, autolinks and footnotes, as GitHub renders it. So code is what CommonMark calls code: fenced and indented code blocks, also inside lists, and code spans. Nothing in code or in raw HTML, comments included, is a heading, a type line or a link, and a character escaped with a backslash is text.

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

## What check reports

| Code | Reported when | At line |
|---|---|---|
| `title` | The first block is not a level-1 heading with text: `missing level-1 title`. Otherwise, a further level-1 heading: `second level-1 heading` | The first block, or 1 for a document without blocks; the first further level-1 heading |
| `type.line` | The type line is not exactly one counted link: `type line must be exactly one link to a document`. A misplaced type line: `a type line stands only on the first line after the title` | That line |
| `type.unresolved` | The type line's link points to no document: `no document at "PATH"`. Its sections are then not checked | The type line |
| `section.missing` | A required section of the type is missing: `missing section "NAME" required by TYPEPATH` | The type line |
| `section.empty` | A section of the type that the document has holds no text: `empty section "NAME"` | The document's heading of that section |
| `link.unresolved` | A counted link points to no document: `no document at "PATH"`, with the path as written, without its section | The link's first line |

A document reports each unresolved target once, at its first link: `[the archive](archive.md)`, `[its rules](archive.md#rules)` and `[again](./archive.md)` are one report. Each declared section is reported at most once, and a document at most one `title`. The diagnostics are sorted by path, then by line.

## Commands

Every command works in a project and finds its spec through it: the folder `-C PROJECT` names, or the nearest folder from the working directory upward that holds `.markitect/` ([project reference](project-reference.md#the-project)). A `DOCUMENT`, `TYPE`, `OLD`, `NEW` or `PATH` is a path below the spec folder, with or without `.md`, with `/` or `\` between folders; a path that leaves the spec folder, is absolute or starts with a scheme names no document. Flags may stand anywhere, `-json` is the same as `--json`, and `--` ends the flags, so that a name starting with `-` can follow. `markitect-next help`, `-h` or `--help` prints the usage. Output paths are below the spec folder, with `/`.

| Command | Does |
|---|---|
| `markitect-next init [--spec PATH]` | Sets up a project: `.markitect/` and the spec folder ([project reference](project-reference.md#init)) |
| `markitect-next check` | Checks the spec by the rules above |
| `markitect-next show DOCUMENT` | The document's title, its type, its sections (a type's with their `(optional)` marker), the documents it links to, the documents that link to it with the section each link stands in, and for a type, its documents |
| `markitect-next table TYPE` | Every document of the type as a Markdown table: the title, then one column per section of the type |
| `markitect-next new TYPE PATH [TITLE]` | Creates a document of the type |
| `markitect-next move OLD NEW` | Moves or renames a document and rewrites the links to it |
| `markitect-next apply TEMPLATE [TARGET]` | Copies a template of the [library](library.md) into the spec |
| `markitect-next topic [TOPIC]` | The signals of the spec, or the topic view of a document, a type or a folder ([signals](signals.md)) |
| `markitect-next questions` | The question catalogs of the library packages the spec has ([signals](signals.md#question-catalogs)) |
| `markitect-next version` | The tool's version, and whether it is the one the project pins ([project reference](project-reference.md#version)) |

**Exit codes.** 0 when all is well, 1 when `check` found something or `version` a project that pins another version, and 2 when the command could not run: an unknown command or flag, a wrong number of arguments, no project, a `.markitect/` that cannot be read, a spec folder that does not exist (`apply` creates it), an unknown document, a file that cannot be read, or a refused `new`, `move` or `apply`. A command that does not succeed changes nothing, also when writing fails partway, unless undoing fails as well, which it reports ([below](#commands-that-write)).

### Commands that read

**`check`** prints one line per diagnostic, `path:line: code: message`, then a summary such as `2 errors in 14 documents`:

```text
lending/borrow.md:7: link.unresolved: no document at "loans/ghost.md"
lending/loans/4711.md:9: section.empty: empty section "Due"
2 errors in 14 documents
```

**`show`** prints the document's path, `Title:`, `Type:` (the type's path, `(none)`, `(invalid line)` for a type line that is not one link, or the path with `(missing)`), then the lists `Sections:`, `Links to:`, with `(missing)` after a path that names no document, and `Linked from:`, each `(none)` when empty, and `Documents:` only for a type that has some. `Links to:` holds each path once; `Linked from:` holds one entry per linking document and section, `path (Section)`, so a document that links from two sections appears twice. Both are sorted without regard to case.

**`table`** prints a row per document of the type, in path order. A cell holds the first line of the section's text as plain text: markup, list markers, HTML tags and comments are dropped, and links, reference links and images show as their text. A section that starts with code shows its first code line, one that starts with a table its first row, cells separated by ` | `. `|` is escaped, and a cell is empty when the section has no text or the document leaves an optional section out. A row's title is the document's title, or its path.

### Commands that write

They compute every change first and check that nothing stands in the way: no file at a path to create, every file to rewrite writable, and every target folder writable. They then write each new content to a temporary file beside its target, make the moves and rename the temporary files into place, keeping each replaced file aside until the end. A new file is linked into place or created exclusively, so a file that appeared meanwhile is never replaced. When a step fails, every step done is undone, so the spec is as it was. Should an undo fail too, the error names the original failure, what could not be undone and every temporary or backup file left behind (`.markitect-*.tmp`, `.markitect-*.bak`), so that the spec can be restored by hand.

- **`new`** writes `# TITLE`, by default the file name, the type line with a relative link to the type, percent-encoded where the path needs it, and an empty level-2 heading for each section the type requires. It refuses when `PATH` exists, as a document or as anything else.
- **`move`** moves the document and rewrites every link to it in the spec: links, type lines and link reference definitions. When the document changes folder, it also rewrites every relative destination in it so that it reaches the same file: links to any file, images, link reference definitions, and the `href` and `src` attributes of HTML tags. Text between tags, escaped markup such as `&lt;img src="x.png"&gt;`, comments, other attributes, such as a `title` that mentions `href=`, and the contents of `script`, `style` and `textarea` stay as they are. It refuses when such an attribute's value cannot be placed: in a tag that does not close, or broken across lines inside a block quote or a list. A rewritten destination keeps its `#section` and its `<…>` form, and is percent-encoded when it was, or when the new path holds white space, a parenthesis or an angle bracket; nothing else in a file changes, line ends and byte order mark included. It refuses when `OLD` is no document or `NEW` exists. It prints `moved OLD -> NEW`, then `changed PATH` for each file it rewrote.
- **`apply`** copies the template, with every package it needs, into the spec at `TARGET`, by default the spec's root; the spec folder is created when it does not exist. It prints `copied PATH` for each document and `copied packages/NAME/` for each package. The [library](library.md#apply) gives the rules.

### JSON

With `--json`, every command prints one document that starts with `format_version` (1) and `kind`, the command's name. Later fields join without changing these.

| Kind | Fields |
|---|---|
| `check` | `documents`, the number of documents, and `diagnostics`, a list, empty when nothing is found, of objects with `file`, `line`, `code` and `message` |
| `show` | `path`, `title`, `type`, `type_state` (`none`, `invalid`, `missing` or `ok`), `sections` with `heading`, `links_to` with `path` and `missing`, `linked_from` with `path` and `section`, and `documents` |
| `table` | `type`, `columns`, and `rows` with `path`, `title` and `cells` |
| `init`, `version` | As the [project reference](project-reference.md) says |
| `new` | `path` |
| `move` | `from`, `to` and `changed` |
| `apply` | `documents` and `packages` |
| `topic`, `questions` | As [signals](signals.md#commands) says |
