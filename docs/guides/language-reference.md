# Language reference

The exact rules of the model language as the tool implements them, and the commands of `markitect`. [The language](../concept/the-language.md) in the concept says what the language is for; this page says precisely what the tool reads and reports. The examples are a small lending library.

## Documents

A model is a folder. Every file in it, at any depth, whose name ends in `.md` in any case is a document; a file named only `.md` is not, and neither is a folder whose name ends in `.md`. Other files are ignored.

- **Path.** A document is known by its path below the model folder, with forward slashes, such as `lending/borrow.md`. Paths are compared exactly, case included, on every system.
- **Text.** A document is read as UTF-8. A leading byte order mark is dropped, and bytes that are not UTF-8 are replaced. Lines end with LF, CRLF or CR, and are numbered from 1.
- **Markdown.** A document is read as [CommonMark](https://spec.commonmark.org/) with GitHub's extensions, tables, task lists, strikethrough, autolinks and footnotes, as GitHub renders it. So code is what CommonMark calls code: fenced and indented code blocks, also inside lists, and code spans. Nothing in code or in raw HTML, comments included, is a heading, a type line, an extends line, a property type line or a link, and a character escaped with a backslash is text.

## Title and sections

Titles and sections come from the headings at the top level of a document; a heading inside a list or a block quote is part of the text around it. Link reference definitions, `[rules]: lending/rules.md`, and footnote definitions are no blocks: they stand before no title and give no section text. Both kinds of CommonMark heading count: `# Borrow`, and `Borrow` underlined with `=====` (level 1) or `-----` (level 2). The text of a heading is as written, without closing `#`, and the lines of an underlined heading are joined by a space.

- **Title.** The first block of a document is a level-1 heading with text, the title. Any further level-1 heading is a second title.
- **Sections.** Each level-2 heading opens a section, which runs to the next level-1 or level-2 heading. Every block in between, level-3 to level-6 headings, code and rules included, belongs to it. A section has text when one of these blocks shows something: a character other than white space outside HTML comments, code, a rule, an image or a link. A section that holds only white space and comments, such as `<!-- fill in -->`, is empty.

## Links

A link is a Markdown link outside code and raw HTML: an inline link, `[text](destination)`, optionally with a title, `[text](destination "title")`, or a reference link, `[text][label]`, whose label is defined in the document. White space inside the parentheses is allowed, `[Loans]( loans.md )`, and a destination in `<` and `>` may hold spaces, `[Loans](<lending/open loans.md>)`. A link holds no link: in `[a [b](b.md)](a.md)` only `b.md` is a link. A link in a footnote counts when the footnote is referred to; GitHub shows no other. An image, `![scan](card.md)`, and an autolink are no link.

A link counts when its destination, without its `#section` and after decoding, ends in `.md` in any case and does not start with a scheme such as `https:`, `mailto:` or `C:`. Web addresses, other files and a bare `#section` are not checked.

- **Resolution.** The path is relative to the folder of the linking document. Backslash escapes and character references, named and numeric such as `&#46;` and `&#x2E;`, are resolved, and percent-encoding is decoded, so `open%20loans.md` names `open loans.md`; a `%` that is not followed by two hexadecimal digits stays as it is. A path that starts with `/`, or that leaves the model folder with `..`, names no document.
- **Sections.** A link with a section, `[Card](members/ada.md#card)`, points to the document; the section is not checked.
- **Every counted link must point to a document of the model**, by its exact path, and to one that is shown across the boundaries of modules ([below](#visibility)).

## Types, objects and properties

A **type** is a document that another document names in its type line, its extends line or a property type line, a document with an extends line that is one link, whether or not its target exists ([below](#extending-a-type)), or a document with a well-formed property type line ([below](#properties-and-their-types)); nothing else marks it. Its text before the first section says what the type is, and each of its level-2 headings declares a **property** of its objects ([DEC-157](../decisions.md#dec-157-a-property-names-its-type-and-modules-depend-on-each-other-without-a-cycle)).

```markdown
# Loan

One copy lent to one member.

## Borrower
Type: [Member](member.md)

The member who holds the copy.

## Due
The date the copy is due back.

## State
Type: `open`, `returned`, `overdue`

Where the loan stands.

## Renewals (optional)
How often the loan was renewed.
```

- **Optional.** A heading that ends with `(optional)`, in any case, declares an optional property. Its name is the heading without the marker and the white space before it: `Renewals`. Every other property is required.
- **Declared twice.** A name declared more than once, in any case, is one property, declared by the first heading of that name, with or without a property type line. It is optional only when every heading of that name is. A `Type:` line under a later heading of that name is a misplaced type line, `type.line`.

The **type line** is the first line of the block right after the title, when that block is a paragraph and the line starts with exactly `Type:`. What follows `Type:`, without surrounding white space, is exactly one inline link that counts as a link above, with an optional title and `#section`. A document with a type line is an **object** of the type it names. Any other line of text that starts with `Type:`, also in a list or in a document without a title, is a misplaced type line, unless it is a property type line in its place ([below](#properties-and-their-types)); a line in code is not. The links of a type line, placed or misplaced, are checked only as the type.

```markdown
# Loan 4711

Type: [Loan](../types/loan.md)

## Borrower
[Ada Lovelace](../members/ada-lovelace.md)

## Due
1 March

## State
open
```

An object has each required property of its type as a level-2 heading of its own, in any order. A heading matches a property when the two are equal without regard to case, by Unicode simple case folding; when the object has two such headings, the first counts. An optional property is written without the marker, `## Renewals`, or left out. Every section of a property the object has, required or optional, has text, and holds a value of the property's type ([below](#properties-and-their-types)). The object may add sections of its own, which hold text.

### Properties and their types

A property names its type with a **property type line**: the first line of the first block after the first level-2 heading of that name in the type, when that block is a paragraph and the line starts with exactly `Type:`. What follows `Type:` is inline links separated by commas, or code spans separated by commas, and nothing else. Any other line there, one that mixes links and code spans or holds a blank code span included, is `property.line`: it gives the property no type, so the section holds text, and it does not make its document a type. A `Type:` line anywhere else stays what it was: in the type line's place a type line, elsewhere a misplaced type line.

- **Text.** A property without a property type line holds text. A link in its section is a **mention**: it says that one document concerns another, and the tool checks it as a link and nothing more.
- **References.** A property type line of links makes a **reference property**, and each link names a type. In an object, every counted link in the property's section is a **reference** and points to an object of a named type or of a type derived from one; several links in the line mean that it may be an object of any one of them. A web address, or any other link that does not count, is no reference, and a link in a footnote stands in no section and is a mention, also when the footnote is referred to from a reference property. The section holds any number of references. A reference to a document of another kind is `reference.type`, once for each wrong target in the section, at its first link. A reference that points to no document is only `link.unresolved`, and one into what a module keeps internal is `link.internal`, as for any link.
- **Fixed values.** A property type line of code spans makes a fixed-values property: the text of each code span is one value. The first line of the section in an object, read as plain text as [`table`](#commands-that-read) reads it, is one of the values, compared exactly, case included, so `` `offered` `` and `offered` are the same value; otherwise it is `property.value`.
- **Where it stands.** A property type line stands only in a type. In an object, a type that has a type line of its own included, and in a README, it is `property.line` and is not read further.
- **Its links.** Each link of a property type line counts as a link above, with an optional title and `#section`, and must point to a document, which is a type by that alone: one that points to no document is `property.unresolved`, and one that names a README is `readme.type`. It passes the boundaries of modules as every link does ([visibility](#visibility)).

A property that sometimes holds a reference and sometimes something else has no form: it is a sign that the property should be cut more cleanly.

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

- **One of the two lines.** A document has a type line or an extends line, never both. When both stand in the first block after the title, the first of them counts and the second is an error. A document with an extends line is a type and is never checked as an object of its base, so a type that has no object yet is not mistaken for one. The type line passes nothing on: a type that has a type line of its own, as an object of another type, inherits none of its properties.
- **The chain.** A type's chain is the type, its base, its base's base and so on. The properties of a type are those declared along its chain. A property declared at more than one level, in any case, is one property, as one declared twice in a type is: required when any level requires it, and described by the most derived level that declares it. The properties stand in chain order: the base's first, then what each derived type adds.
- **Never looser.** A derived type may describe a property of its base more precisely or make an optional one required, `## Renewals` above. It never marks optional a property that a type up its chain requires.
- **Same type.** A derived type never gives an inherited property another type; that is `property.retyped`. Types are compared as sets, so the same types or values in another order are the same type. The derived type's line then counts for its own objects all the same. Declared again without a property type line, the property keeps the type it inherits.
- **No cycle.** A chain that comes back to itself is an error at each type in the cycle; a type whose chain only leads into a cycle is not. An object of such a type is checked against the properties its chain collects before it would loop.

An object of a derived type has each required property of the type's chain, and every property of the chain it has has text and a value of its type, by the rules above. An object of `Renewable loan` needs `Borrower`, `Due`, `State`, `Renewals` and `Limit`.

## Modules

A **module** is a folder of the model that holds a document directly, as a file of its own and not only in a folder below it ([DEC-156](../decisions.md#dec-156-a-module-is-a-folder-and-its-readme-says-what-it-shows)). The model folder itself is a module only when it holds a document directly. A module **holds** the documents of its folder and of every folder below it, those of the modules below included.

- **Transparent folders.** A folder that holds no document directly, only folders or files that are no documents, is no module and needs no README. It shows and hides nothing: the modules in it count as lying directly in the nearest module above it.
- **Nesting.** A module lies within every module whose folder holds its folder at any depth. A module is **directly below** another when it lies within it and within no other module that lies within it, transparent folders passed over. When the module `lending/` holds `fines/late.md`, and a folder `rules/` that holds no document but the folder `renewals/` with `limit.md`, both `lending/fines/` and `lending/rules/renewals/` are directly below `lending/`.
- **README.** Every module has a document named exactly `README.md` in its folder, its README. A module without one is `module.readme`, and every link into it from outside is `link.internal`.

### The README

A README is a document, read and checked by the rules above, with a form of its own:

- **No type line and no extends line.** Either line in a README is `readme.type`, and the README is read without it: a README is never an object and never a type, and `show` prints `Type: (none)` for it. A type line, an extends line or a property type line of another document that names a README is `readme.type` as well. A property type line in a README is `property.line`.
- **Text before the first section** says what the module is for.
- **`Responsible`**, optional, names the role that answers for the module, as a link; without it, the role of the module above answers. `check` does not check it.
- **`Realized in`**, optional, says where in the project the module is built, by path patterns ([below](#realized-in)).
- **`Contents`** is every section of the README whose level-2 heading is `Contents`, without regard to case; when there are two, the links of both count. `Contents` is not required, and a README without it shows nothing but itself.

**What `Contents` shows.** Every counted link in `Contents`, one with a `#section` included, that points to a document of the README's own folder or to the README of a module directly below. Such a document is **public** in the module, and so is the README itself. A link in `Contents` to anything else shows nothing and is checked as any other link of the README: one to a document inside a module below is itself `link.internal` unless that module shows it.

**Patterns.** A code span in `Contents` whose text holds no `/` is a pattern over the file names of the documents directly in the README's folder, and every document it matches is public: `` `*.md` `` shows every document of the folder, `` `report-*.md` `` those whose names start with `report-`. The wildcards are `*` for any run of characters, `?` for one character and a class such as `[a-z]` for one of a set, or `[^a-z]` for one outside it; everything else matches itself, case included. A pattern matches the file name alone, so it never reaches into a folder below: a module below is shown by a link to its README. A code span that holds `/`, such as `` `old/*.md` ``, is plain code and shows nothing. A pattern that matches no document is no error; one that is not a valid pattern, such as `` `report-[1.md` ``, is `readme.pattern` and shows nothing. Code spans count anywhere in `Contents`, in list items and block quotes too; fenced and indented code blocks do not. A pattern is written as code so that a rendered page shows no dead link.

```markdown
# Lending

How members borrow copies and bring them back.

## Responsible
[Librarian](../team/librarian.md)

## Realized in
- `src/lending/`, the loan service
- `db/migrations/*-loans.sql`

## Contents
- [Borrow](borrow.md)
- [Fines](fines/README.md)
- `notice-*.md`
```

### Realized in

`Realized in` is every section of a README whose level-2 heading is `Realized in`, without regard to case; when there are two, the patterns of both count ([DEC-158](../decisions.md#dec-158-a-module-says-where-it-is-built-and-diff-compares-its-model-and-its-files)). Every code span in it is a **path pattern**, in list items and block quotes too; fenced and indented code blocks do not count, and text around a code span, such as the words after `src/lending/` above, is free. `check` reads the patterns; only [`diff`](project-reference.md#diff) uses them.

- **Relative to the project folder.** A path pattern names files by their path below the project folder, the folder that holds `.markitect/model`, with `/` between folders, not relative to the README as a link is. `src/lending/` in the README above is the folder `src/lending` beside `.markitect/`, wherever the model folder lies.
- **Wildcards.** `*` stands for any run of characters, `?` for one character and a class such as `[a-z]` for one of a set, or `[^a-z]` for one outside it, as in the patterns under `Contents`. None of them matches across a `/`: each matches within one folder name or the file name, a dot at its start included, so `*/workflows/` reaches `.github/workflows/`. `**/` at the start matches in any folder, `/**` at the end everything inside, and `/**/` in the middle zero or more folders: `**/*.sql` names `top.sql` and `db/x/q.sql`. Everything else matches itself, case included.
- **Files.** A pattern that does not end with `/` names the files whose whole path matches it: `db/migrations/*-loans.sql` names `db/migrations/0042-loans.sql`, and not `db/migrations/old/0001-loans.sql`.
- **Folders.** A pattern that ends with `/` names a folder and every file below it, at any depth: `src/lending/` names `src/lending/borrow.go` and `src/lending/store/sql.go`. A name that starts with a dot is a name like any other: `src/` names `src/.env` as well.
- **Not valid.** A pattern that is empty or only white space, absolute, starts with a drive such as `C:`, holds a backslash, has an empty folder name or `.` or `..` as a folder name, or is malformed, such as `` `src/[a.go` ``, is `readme.pattern` and names nothing.

`check` does not check whether a path exists: it reads only the model, and a pattern that names no file is no error. A pattern never names a file in the model folder, in the folder `.markitect/` at the top of the project, or what the project's ignore file leaves out ([project reference](project-reference.md#the-ignore-file)).

### Visibility

Every counted link passes the boundaries of modules by one rule: a link from a document A to a document B passes when **every module that holds B but not A shows the way to B**, that is, lists under `Contents` either B itself, when B lies in that module's folder, or else the README of the module directly below it that holds B. A module's own README is always shown. A link that does not pass is `link.internal`. The rule holds for the type line, the extends line and the property type line as for every other link, so a type that other modules name stands in the `Contents` of its module, and of every module above it that the naming document lies outside.

With the README above, `lending/fines/README.md` showing `late.md`, and a further document `lending/loan-limit.md` that `Contents` does not list:

| Direction | From | To | Passes |
|---|---|---|---|
| Inside | `lending/borrow.md` | `lending/loan-limit.md` | Yes: `lending` holds both |
| Upward | `lending/fines/late.md` | `lending/loan-limit.md` | Yes: a module may link what is internal to the modules above it |
| Downward | `lending/borrow.md` | `lending/fines/late.md` | Yes: `fines` shows `late.md` |
| Sideways | `members/card.md` | `lending/borrow.md` | Yes: `lending` shows it |
| Sideways | `members/card.md` | `lending/loan-limit.md` | No: `link.internal` |
| Two levels | `members/card.md` | `lending/fines/late.md` | Yes: `lending` shows `fines/README.md`, and `fines` shows `late.md` |
| Two levels | `members/card.md` | `lending/fines/waiver.md`, not listed in `fines` | No: `link.internal` |

A module that holds both documents is never asked, so the model folder, when it is a module, never hides anything. A link that points to no document is `link.unresolved`, not `link.internal`.

### Dependencies and cycles

Modules do not depend on each other in a cycle ([DEC-157](../decisions.md#dec-157-a-property-names-its-type-and-modules-depend-on-each-other-without-a-cycle)). A document belongs to the module whose folder holds it directly. A module A has a **dependency** on another module B when:

- an object in A has a type line that names a type in B;
- a type in A has an extends line that names a type in B;
- a type in A has a property type line with a link to a type in B; or
- an object in A has a reference, a link in a reference property, to a document in B.

A link from a module to itself is no dependency. A parent module and a module below it count like any two: a link downward and a link upward between them make a cycle, although visibility lets the module below link what is internal to the one above. A mention never counts, and neither does any link of a README, `Contents` and `Responsible` included. A link that points to no document makes no dependency, and neither does a README named in a property type line; a link that is `link.internal` or `reference.type` makes one all the same.

A **cycle** is a chain of dependencies that leads from a module back to itself. `check` reports `module.cycle` once for each group of modules that all reach each other by dependencies, with one shortest cycle in the group as a chain of modules and, for each step, one link that causes it. The diagnostic stands at the link of the first step:

```text
a/x.md:6: module.cycle: module cycle a -> b -> a: a/x.md:6 refers to b/y.md, b/y.md:6 refers to a/x.md
```

Each step names its kind: `is of type` for a type line, `extends` for an extends line, `declares a reference to` for a property type line and `refers to` for a reference, as in `module cycle a -> b -> a: a/x.md:3 is of type b/thing.md, b/y.md:6 refers to a/x.md`. A module is named by its folder below the model folder, such as `lending/fines`, and the model folder as `(model folder)`.

## What check reports

| Code | Reported when | At line |
|---|---|---|
| `title` | The first block is not a level-1 heading with text: `missing level-1 title`. Otherwise, a further level-1 heading: `second level-1 heading` | The first block, or 1 for a document without blocks; the first further level-1 heading |
| `type.line` | The type line is not exactly one counted link: `type line must be exactly one link to a document`. A misplaced type line, one that is neither in the type line's place nor a property type line: `a type line stands only on the first line after the title`. A type line after an extends line in the first block: `a document has a type line or an extends line, not both` | That line |
| `type.unresolved` | The type line's link points to no document: `no document at "PATH"`. Its sections are then not checked | The type line |
| `extends.line` | The extends line is not exactly one counted link: `extends line must be exactly one link to a document`. A misplaced extends line: `an extends line stands only on the first line after the title`. An extends line after a type line in the first block: `a document has a type line or an extends line, not both`. The document is then read as without that line | That line |
| `extends.unresolved` | The extends line's link points to no document: `no document at "PATH"` | The extends line |
| `extends.cycle` | The chain comes back to itself: `the chain of extends lines comes back to itself`, once at each type in the cycle; a type that only leads into a cycle reports nothing | The extends line |
| `extends.loosens` | A derived type marks optional a property that a type up its chain requires: `section "NAME" is required by TYPEPATH`, the nearest type up its chain that requires it | The derived type's heading of that property |
| `property.line` | A property type line is not inline links or code spans separated by commas, mixes the two or holds a blank code span: `property type line must be links to documents or code spans, separated by commas`. In a README: `a README has no property type line`. In an object: `a document with a type line has no property type line` | That line |
| `property.unresolved` | A link of a property type line points to no document: `no document at "PATH"`, with the path as written | That line |
| `property.retyped` | A derived type declares an inherited property with a property type line that gives it another type: `property "NAME" is TYPE in TYPEPATH, not TYPE`, where TYPE is `text`, `a reference to` the types' paths or `one of` the values, and TYPEPATH the type up the chain that declares the property | The derived type's property type line |
| `section.missing` | A required property of the type's chain has no section in the object: `missing section "NAME" required by TYPEPATH`, the most derived type of the chain that requires it | The type line |
| `section.empty` | The section of a property of the type's chain that the object has holds no text: `empty section "NAME"` | The object's heading of that section |
| `reference.type` | A reference in an object points to a document that is no object of a type its property names, or of a type derived from one: `section "NAME" refers to documents of TYPEPATH or a type that extends it, but "PATH" is a document of OTHERPATH`, with several types `documents of one of TYPEPATH, TYPEPATH or a type that extends one of them`; for a target that is no object the end reads `is a type`, `is a README` or `is a document without a type`. Once for each wrong target in a section | The target's first link in the section |
| `property.value` | The first line of a fixed-values section in an object is none of the values: ``the first line of section "NAME" is "TEXT", not one of `A`, `B` ``. A value that holds a backtick prints between as many backticks as keep it unambiguous, ``` ``a`b`` ``` | The line that holds the value, the first line of the section's text |
| `link.unresolved` | A counted link points to no document: `no document at "PATH"`, with the path as written, without its section | The link's first line |
| `link.internal` | A counted link, those of the type line, the extends line and a property type line included, points to a document of a module that does not show the way to it ([visibility](#visibility)): `"PATH" is internal to MODULE: MODULE/README.md does not list it under Contents`, for the first module of the way that does not show its step, or `… does not list MODULE/BELOW/README.md under Contents` when that step is the README of a module below; when a module on the way has no README, `"PATH" is internal to MODULE: BELOW has no README.md`, where BELOW is that module, MODULE itself or one below it. PATH is the target below the model folder | The link's first line |
| `module.readme` | A folder that holds a document directly has no `README.md`: `the folder FOLDER holds documents and has no README.md`, or for the model folder `the model folder holds documents and has no README.md`. Once per folder | Line 1 of the README's path, such as `shop/README.md:1` or `README.md:1` |
| `module.cycle` | Modules depend on each other in a cycle ([dependencies and cycles](#dependencies-and-cycles)): `module cycle MODULE -> MODULE -> MODULE: PATH:LINE STEP PATH, ...`, where STEP names the kind of the step, once per group of modules that all reach each other, with one shortest cycle and one link per step | The link of the first step |
| `readme.pattern` | A code span in a README's `Contents` without `/` is no valid pattern: `bad pattern "TEXT": a pattern uses the wildcards *, ? and [a-z]`. A code span in a README's `Realized in` is no valid path pattern ([Realized in](#realized-in)): `bad pattern "TEXT": REASON`, where REASON is `a pattern is not empty`, `a pattern separates folders with /, not \`, `a pattern is relative to the project folder and has no drive`, `a pattern is relative to the project folder, not absolute`, `a pattern stays below the project folder, without ..`, `a pattern has no empty folder name and no .` or `a pattern uses the wildcards *, ? and [a-z]`, as in `shop/README.md:7: readme.pattern: bad pattern "../shop": a pattern stays below the project folder, without ..` | The code span's line |
| `readme.type` | A README has a type line or an extends line: `a README has no type line` or `a README has no extends line`, once for each line it has. A type line, an extends line or a property type line names a README: `"PATH" is a README, which is no type` | That line |

A document reports each unresolved or internal target once, at its first link, where its type line or extends line comes first: `[the archive](archive.md)`, `[its rules](archive.md#rules)` and `[again](./archive.md)` are one report. Each property is reported at most once, and a document at most one `title`. The diagnostics are sorted by path, then by line.

## Commands

Every command but `version` and `licenses` works in a project and finds its model through it: the folder `-C PROJECT` names, or the nearest folder from the working directory upward that holds `.markitect/model` ([project reference](project-reference.md#the-project)). A document, type, topic or folder named on the command line is a path below the model folder, with or without `.md`, with `/` or `\` between folders; a path that leaves the model folder, is absolute or starts with a scheme names no document. The `OLD` and `NEW` of `diff` are project folders.

A flag may stand anywhere, before the command as well, as in `markitect -C PROJECT check`, and `-json` is the same as `--json`. `--all` is a flag of `topic` only. `--` ends the flags, so that a name starting with `-` can follow. `markitect help`, or `-h` or `--help` wherever it stands, even after a command, prints the general usage, which groups the commands as this table does; the usage of one command is printed with a wrong number of arguments. A flag that no command has, or one that lacks its value, is an error that names it. Output paths are below the model folder, with `/`.

| Group | Command | Does |
|---|---|---|
| Look | `markitect show DOCUMENT` | The document's title, its type or, for a derived type, its base, its sections (a type's those of its whole chain, with their `(optional)` marker, each property's type and, for an inherited one, the type that declares it), the documents it links to, the documents that link to it with the section each link stands in, and for a type the types that extend it and its objects, those of the types that extend it included |
| | `markitect table [TYPE]` | Every object of the type and of every type that extends it as a Markdown table: the title, then one column per section of the type's chain. Without a type, every document with its type and title |
| | `markitect topic [TOPIC] [--all]` | The signals of the model, or the topic view of a document, a type or a folder ([signals](signals.md)) |
| | `markitect questions` | The question catalog of the default's module `core`, when the model has it ([signals](signals.md#question-catalogs)) |
| | `markitect diff OLD [NEW]` | What is new, gone or changed in the model of NEW, by default this project, against the project OLD, with a diff for what changed; for each module whether its model, its files or both changed; and the files that no module names ([project reference](project-reference.md#diff)) |
| Check | `markitect check` | Checks the model by the rules above |
| Tool | `markitect version` | The tool's version and build ([project reference](project-reference.md#version)) |
| | `markitect licenses` | The license notices of the Go modules the tool is built from |

The commands only read: none writes a file, keeps a state, starts an agent or steers work, and each does one thing on its own.

**Exit codes.** 0 when all is well, 1 when `check` or `diff` found something, and 2 when the command could not run: an unknown command or flag, a flag without its value, no command at all, a wrong number of arguments, no project, a `.markitect/model` that cannot be read or names a model folder it may not, a model folder that does not exist, a document, type or topic that the model does not have, an ignore file with a line that is no valid pattern, or a file that cannot be read.

### Commands that read

**`check`** prints one line per diagnostic, `path:line: code: message`, then a summary such as `2 errors in 9 documents`:

```text
loans/borrow.md:11: link.unresolved: no document at "fines.md"
roles/member.md:5: section.empty: empty section "Does"
2 errors in 9 documents
```

**`show`** prints the document's path, `Title:`, `Type:` (the type's path, `(none)`, `(invalid line)` for a type line that is not one link, or the path with `(missing)`), or for a document with an extends line `Extends:` (the base's path, or the path with `(missing)`), then the lists `Sections:`, `Extended by:`, the types that extend it directly, sorted, only for a type that has some, `Links to:`, with `(missing)` after a path that names no document, and `Linked from:`, each `(none)` when empty, and `Documents:` only for a type that has some objects, with the objects of the types that extend it, directly or further down. A derived type's `Sections:` are those of its whole chain in chain order, an optional one with its marker and an inherited one with the type that declares it: `Due (from types/loan.md)`. A section the derived type declares again is its own and stands in its base's place. On a type, a property with a type shows it after a colon, `Borrower (from types/loan.md): reference to types/member.md` or ``State (from types/loan.md): one of `open`, `returned`, `overdue` ``, a value that holds a backtick between as many backticks as keep it unambiguous, with several types separated by commas and `(missing)` after a path that names no document; a text property shows nothing more. `Links to:` holds each path once; `Linked from:` holds one entry per linking document and section, `path (Section)`, or the bare path for a link that stands before any section, so a document that links from two sections appears twice. Both are sorted without regard to case, and both leave out the links of type lines, extends lines and property type lines.

**`table`** prints a row per object of the type and of every type that extends it, directly or further down, in path order, with one column per section of the type's own chain. A cell holds the first line of the section's text as plain text: markup, list markers, HTML tags and comments are dropped, and links, reference links and images show as their text. A section that starts with code shows its first code line, one that starts with a table its first row, cells separated by ` | `. `|` is escaped, and a cell is empty when the section has no text or the object leaves an optional property out. A row's title is the document's title, or its path. Without a type, `table` prints every document of the model in path order as `| Document | Type | Title |`: its path, the path of its type line's type, empty without one, as for a derived type, and its title.

### JSON

With `--json`, every command prints one document that starts with `format_version` (1) and `kind`, the command's name. Later fields join without changing these.

| Kind | Fields |
|---|---|
| `check` | `documents`, the number of documents of the model, and `diagnostics`, a list, empty when nothing is found, of objects with `file`, `line`, `code` and `message` |
| `show` | `path`, `title`, `type`, `type_state` (`none`, `invalid`, `missing` or `ok`), `extends` and `extends_state` (`missing` or `ok`) for a document with an extends line, whose `type` is then empty and `type_state` `none`, `sections` with `heading`, the marker included, `declared_by` for an inherited one, and `references`, objects with `path` and `missing`, for a reference property or `values` for a fixed-values property, both left out for text, `extended_by`, `links_to` with `path` and `missing`, `linked_from` with `path` and `section`, and `documents`; an empty `section`, `extends`, `extends_state`, `declared_by` or `extended_by` and a `missing` that is false are left out |
| `table` | `type`, `columns`, and `rows` with `path`, `title` and `cells`; without a type, `type` is empty and the one column is `Type` |
| `version`, `diff` | As the [project reference](project-reference.md) says |
| `licenses` | `notices` |
| `topic`, `questions` | As [signals](signals.md#commands) says |
