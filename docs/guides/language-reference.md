# Language reference

The exact rules of the spec language as the tool implements them, and the commands of `markitect`. [The language](../concept/the-language.md) in the concept says what the language is for; this page says precisely what the tool reads and reports. The examples are a small lending library.

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

The documents of the work in `.markitect/` are checked with the same codes against their types in the spec, with the differences the [project reference](project-reference.md#the-documents-of-the-work) gives.

## Commands

Every command but `init`, `version` and `licenses` works in a project and finds its spec through it: the folder `-C PROJECT` names, or the nearest folder from the working directory upward that holds `.markitect/version` ([project reference](project-reference.md#the-project)). A document, type, topic or folder named on the command line is a path below the spec folder, with or without `.md`, with `/` or `\` between folders; a path that leaves the spec folder, is absolute or starts with a scheme names no document. The `OLD` and `NEW` of `compare` are project folders, and the `OLD` and `NEW` of `move` are documents.

A flag may stand anywhere, and `-json` is the same as `--json`. A flag belongs to the commands that name it: `--spec` is one of `init`, `--all` one of `topic`, and `--file`, `--check`, `-e` and `-m` are flags of `confirm`. `--` ends the flags, so that a name starting with `-` can follow. `markitect help`, `-h` or `--help` prints the usage, which groups the commands as this table does. Output paths are below the spec folder, with `/`, unless they name a file of the work or of the project.

| Group | Command | Does |
|---|---|---|
| Look | `markitect show DOCUMENT` | The document's title, its type, its sections (a type's with their `(optional)` marker), the documents it links to, the documents that link to it with the section each link stands in, for a type its documents, and its row in the state |
| | `markitect table [TYPE]` | Every document of the type as a Markdown table: the title, then one column per section of the type. Without a type, every document with its verdict and artifacts |
| | `markitect topic [TOPIC] [--all]` | The signals of the spec and the change signals of its history, or the topic view of a document, a type or a folder ([signals](signals.md)) |
| | `markitect questions` | The question catalogs of the library packages the spec has ([signals](signals.md#question-catalogs)) |
| | `markitect compare OLD [NEW]` | What is new, gone, moved, changed or suspect in NEW, by default this project, against OLD, with diffs, changed verdicts and the documents of the work that are new, gone or changed ([project reference](project-reference.md#compare)) |
| Check | `markitect check` | Checks the spec by the rules above, and the documents of the work against their types ([project reference](project-reference.md#the-documents-of-the-work)) |
| | `markitect status` | What is changed, suspect or unexplained ([project reference](project-reference.md#status)) |
| Change the spec | `markitect new TYPE PATH [TITLE]` | Creates a document of the type |
| | `markitect apply TEMPLATE [TARGET]` | Copies a template of the [library](library.md) into the spec |
| | `markitect move OLD NEW` | Moves or renames a document, rewrites the links to it, carries its row in the state and records the move in the history ([project reference](project-reference.md#move-and-the-state)) |
| Confirm | `markitect confirm DOCUMENT@FINGERPRINT VERDICT [--file PATH]... [--check PATH]... [-e EVIDENCE]... [-m REASON]` | Binds `ok`, `fail` or `skip`, with artifacts, counter-checks and evidence, to the state `status` showed ([project reference](project-reference.md#confirm)) |
| Set up | `markitect init [--spec PATH]` | Sets up a project: `.markitect/`, the spec folder with the four templates, the skills of the standard and an entry in `AGENTS.md` ([project reference](project-reference.md#init)) |
| | `markitect version` | The tool's build, and whether its version is the one the project pins ([project reference](project-reference.md#version)) |
| | `markitect licenses` | The license notices of the modules the tool is built from |

The commands inform and write on request; none starts an agent, and none steers work: `status` is a list, never a gate.

**Exit codes.** 0 when all is well, 1 when `check`, `status` or `compare` found something, `confirm` refused, or `version` found a project that pins another version, and 2 when the command could not run: an unknown command or flag, a wrong number of arguments, no project, a `.markitect/` that cannot be read, a spec folder that does not exist (`apply` creates it), an unknown document, a file that cannot be read, a refused `init`, `new`, `move` or `apply`, or a `confirm` that is malformed ([project reference](project-reference.md#confirm)). A command that does not succeed changes nothing, also when writing fails partway, unless undoing fails as well, which it reports ([below](#commands-that-write)).

### Commands that read

**`check`** prints one line per diagnostic, `path:line: code: message`, then a summary such as `2 errors in 9 documents`, to which the documents of the work add their number when there are any:

```text
loans/borrow.md:11: link.unresolved: no document at "fines.md"
roles/member.md:5: section.empty: empty section "Does"
2 errors in 9 documents, 1 document of the work
```

A document of the work has its path below the project folder, such as `.markitect/findings/pending/late-fee.md`.

**`show`** prints the document's path and its fingerprint, `lending/borrow.md@fde1ae8e`, which `confirm` takes ([project reference](project-reference.md#status)), `Title:`, `Type:` (the type's path, `(none)`, `(invalid line)` for a type line that is not one link, or the path with `(missing)`), then the lists `Sections:`, `Links to:`, with `(missing)` after a path that names no document, and `Linked from:`, each `(none)` when empty, and `Documents:` only for a type that has some. `Links to:` holds each path once; `Linked from:` holds one entry per linking document and section, `path (Section)`, so a document that links from two sections appears twice. Both are sorted without regard to case. Then comes the document's row in the state, the one `status` takes ([project reference](project-reference.md#the-state)): `Verdict:`, `(no row)` without one and `(rows conflict)` after it for a conflict, the lists `Artifacts:`, `Counter-checks:` and `Evidence:`, each `(none)` when empty, and `Reason:` when the row gives one.

**`table`** prints a row per document of the type, in path order. A cell holds the first line of the section's text as plain text: markup, list markers, HTML tags and comments are dropped, and links, reference links and images show as their text. A section that starts with code shows its first code line, one that starts with a table its first row, cells separated by ` | `. `|` is escaped, and a cell is empty when the section has no text or the document leaves an optional section out. A row's title is the document's title, or its path. Without a type, `table` prints every document of the spec in path order as `| Document | Verdict | Artifacts |`: its path, the verdict of the row `status` takes, empty without one and with `(rows conflict)` for a conflict, and its artifacts.

### Commands that write

They compute every change first and check that nothing stands in the way: no file at a path to create, every file to rewrite writable, and every target folder writable. They then write each new content to a temporary file beside its target, make the moves and rename the temporary files into place, keeping each replaced file aside until the end. A new file is linked into place or created exclusively, so a file that appeared meanwhile is never replaced. When a step fails, every step done is undone, so the spec is as it was; a document of the work that `move` cannot read or write is the one exception ([project reference](project-reference.md#move-and-the-state)). Should an undo fail too, the error names the original failure, what could not be undone and every temporary or backup file left behind (`.markitect-*.tmp`, `.markitect-*.bak`), so that the spec can be restored by hand.

- **`new`** writes `# TITLE`, by default the file name, the type line with a relative link to the type, percent-encoded where the path needs it, and an empty level-2 heading for each section the type requires. It refuses when `PATH` exists, as a document or as anything else. It prints `created PATH`.
- **`move`** moves the document and rewrites every link to it in the spec: links, type lines and link reference definitions. When the document changes folder, it also rewrites every relative destination in it so that it reaches the same file: links to any file, images, link reference definitions, and the `href` and `src` attributes of HTML tags. Text between tags, escaped markup such as `&lt;img src="x.png"&gt;`, comments, other attributes, such as a `title` that mentions `href=`, and the contents of `script`, `style` and `textarea` stay as they are. It refuses when such an attribute's value cannot be placed: in a tag that does not close, or broken across lines inside a block quote or a list. A rewritten destination keeps its `#section` and its `<…>` form, and is percent-encoded when it was, or when the new path holds white space, a parenthesis or an angle bracket; nothing else in a file changes, line ends and byte order mark included. It refuses when `OLD` is no document or `NEW` exists, and while the spec has errors or the state a line it cannot read; errors in the documents of the work do not stop it. It carries the state and appends to the history in the same change set, under the lock of the state ([project reference](project-reference.md#move-and-the-state)). It rewrites the links in the documents of the work too, and names those it leaves as they are ([project reference](project-reference.md#move-and-the-state)). It prints `moved OLD -> NEW`, then `changed PATH` for each document it rewrote, a document of the spec by its path below the spec folder and one of the work by its path below the project folder, and `left as it is PATH: REASON` for each document of the work it left.
- **`apply`** copies the template, with every package it needs, into the spec at `TARGET`, by default the spec's root; the spec folder is created when it does not exist. It prints `copied PATH` for each document and `copied packages/NAME/` for each package. The [library](library.md#apply) gives the rules.

### JSON

With `--json`, every command prints one document that starts with `format_version` (1) and `kind`, the command's name. Later fields join without changing these.

| Kind | Fields |
|---|---|
| `check` | `documents`, the number of documents of the spec, `work`, the number of documents of the work, and `diagnostics`, a list, empty when nothing is found, of objects with `file`, `line`, `code` and `message` |
| `show` | `path`, `fingerprint`, `title`, `type`, `type_state` (`none`, `invalid`, `missing` or `ok`), `sections` with `heading`, `links_to` with `path` and `missing`, `linked_from` with `path` and `section`, `documents`, and from the state `verdict`, empty without a row, `conflict` when rows conflict, `artifacts`, `counter_checks`, `evidence` and `reason` |
| `table` | `type`, `columns`, and `rows` with `path`, `title` and `cells`; without a type, `type` is empty and the columns are `Verdict` and `Artifacts` |
| `init`, `version`, `status`, `confirm`, `compare` | As the [project reference](project-reference.md) says |
| `new` | `path` |
| `licenses` | `notices` |
| `move` | `from`, `to`, `changed`, the documents of the spec it rewrote, `work`, the documents of the work it rewrote, and `left`, with `path` and `reason` for each document of the work it left as it is, sorted by path |
| `apply` | `documents` and `packages` |
| `topic`, `questions` | As [signals](signals.md#commands) says |
