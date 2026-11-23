# The language

A spec is documentation with a fixed structure, in plain Markdown ([DEC-121](../decisions.md#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates)). There are types and documents, as there are classes and objects. It renders and links on GitHub, in editors and in note tools alike.

## Documents

A spec is a folder, by default `docs/spec/`, and every `.md` file in it, at any depth, is a document. Folders are free: by feature, by kind, or mixed.

- **Path.** A document is known by its path below the spec folder, such as `lending/borrow.md`. File names use lowercase letters, digits and `-`, so that links stay plain; the readable name is the title.
- **Title.** The first line with content is a level-1 heading, the title. No other level-1 heading follows.
- **Sections.** Each level-2 heading `## …` opens a section.
- **Links.** A link is an ordinary Markdown link with a relative path to a `.md` file, `[Member](../roles/member.md)`, optionally with a section, `[Member](../roles/member.md#who-counts)`. Every such link must point to a document of the spec. Web addresses, `mailto:` and a bare `#section` are not checked, and links in code are text.

A link is no import. It says that one document concerns another. No section means anything special to the tool.

## Types

A type is a document that another document names as its type, that another type extends, or that extends a type itself; nothing else marks it. Its text before the first section says what the type is. Each of its sections is a section of its documents, and the text under it says what belongs there. A section is required unless its heading ends with `(optional)`.

```markdown
# Operation

One thing a caller can do to the system or ask of it.

## Does
What it achieves.

## Given
What the caller supplies.

## Refused when (optional)
The conditions under which it is refused, each with its reason and the answer the caller gets.
```

A document names its type in the first line after the title, `Type: ` followed by one link. It then has every required section of its type, in any order, each with some text; it may add sections of its own.

```markdown
# Borrow

Type: [Operation](../packages/software/operation.md)

## Does
Lends a copy of a book to a member.

## Given
The copy's barcode and the member's card.

## Refused when
The member already has five open loans: the answer is `limit_reached`.
```

A type may extend another, its base, with a line of its own in the place of the type line, `Extends: ` followed by one link ([DEC-155](../decisions.md#dec-155-a-type-extends-another-with-a-line-of-its-own)). `Type:` stands only in a document of a type, and no document has both lines. A type that extends another is never a document of it: it needs none of its base's sections, so a type that no document names yet is still a type. Its sections are its own and those of every type up its chain, the base's first; a document of it has every required section of the whole chain. Here a project's own type, `types/query.md`, extends the Operation above, and a document of it needs the sections of both:

```markdown
# Query

Extends: [Operation](../packages/software/operation.md)

An operation that answers a question and changes nothing.

## Answers
What the caller learns from it.
```

```markdown
# Open loans

Type: [Query](../types/query.md)

## Does
Lists the loans of a member that are not yet returned.

## Given
The member's card.

## Answers
Each open loan with its copy and due date.
```

A type may describe a section of its base more precisely, under the same heading, or make an optional one required; it never makes a required one optional. A chain that comes back to itself is an error. Only an extends line passes sections on: a type that has a type line of its own inherits nothing from the type it names.

Where a type says that a section holds a link to something, each document links it; where it says only what belongs there, the document states it directly. A strong statement that every document of a type must keep belongs in the type, so that a contradicting document cannot arise.

## Packages and templates

- **A package** is a folder of types, such as `core` (concept, decision, goal, role, rule, term), `software` or `company`. A type in one package may link to types of another.
- **A template** is a folder of documents that name their types by links into the packages and leave empty the sections a project has to fill. A folder of a template is a template of its own when its documents link nowhere outside it, so a template applies at any level: a whole project, one part of the domain, or a field such as fire protection. Its README says what the part is for and how to fill it.
- **Copying.** A person copies a template into the spec, together with the packages its documents need, placed so that the template's links find them ([§7](the-project.md#setting-up-a-project)). The spec owns its copy and may change it. Then `check` lists the sections left empty, and that list is the work still to do.

Each type says before its sections what it is and when to use it, and under each section what belongs there, with an example. The packages and templates are plain files in Markitect's repository; the reference values and the question catalogs ship inside the tool, in its version ([§7](the-project.md#the-default)).

## Values

A value, such as the state of a concept or the owner of a process, stands in the first line of its section, `## Owner` with a link to a role under it. There are no lines in the head but the type line, or in a type the extends line.

## Parts

A part of the spec is a folder with a README, which is its table of contents: each entry links the README of a part below or a document the part holds. A rule for one module links that module's README in a section such as "Applies to".

## The writing rule

Every fact stands in exactly one place, and everywhere else links to it. A person's role is stated in that person's document, not again in the role's. This rule is not checked, and it matters more than any check: contradictions come from the same fact standing in several places, and no type system catches them.

An agent writes the structure, the sections and the links, as it writes the prose. Nothing derives the structure from the prose.

## What `check` catches

| Code | Meaning |
|---|---|
| `title` | The title is missing, or there is a second level-1 heading |
| `type.unresolved` | The type line's link points to no document |
| `type.line` | A line starting with `Type:` that is not exactly one link |
| `extends.unresolved` | The extends line's link points to no document |
| `extends.line` | A line starting with `Extends:` that is not exactly one link, is not in the type line's place, or stands beside a type line |
| `extends.cycle` | A chain of extends lines comes back to itself |
| `extends.loosens` | A type marks optional a section that a type up its chain requires |
| `section.missing` | A required section of the type, or of a type up its chain, is missing |
| `section.empty` | A section of the type, or of a type up its chain, has no text |
| `link.unresolved` | A link to a `.md` path points to no document of the spec |

What the language does not catch, contradictions between values, gaps and wrong values, a reader finds ([§5](is-the-spec-good.md)). There are no value forms, check rules or declared scopes: something new joins only when a real project needs it.
