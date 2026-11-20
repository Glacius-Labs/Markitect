# The language

A spec is documentation with a fixed structure, in plain Markdown ([DEC-121](../decisions.md#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates)). There are types and documents, as there are classes and objects. It renders and links on GitHub, in editors and in note tools alike.

## Documents

A spec is a folder, by default `docs/spec/`, and every `.md` file in it, at any depth, is a document. Folders are free: by feature, by kind, or mixed.

- **Path.** A document is known by its path below the spec folder, such as `lending/borrow.md`. File names use lowercase letters, digits and `-`, so that links stay plain; the readable name is the title.
- **Title.** The first line with content is a level-1 heading, the title. No other level-1 heading follows.
- **Sections.** Each level-2 heading `## …` opens a section.
- **Links.** A link is an ordinary Markdown link with a relative path to a `.md` file, `[Member](../roles/member.md)`, optionally with a section, `[Member](../roles/member.md#who-counts)`. Every such link must point to a document of the spec. Web addresses, `mailto:` and a bare `#section` are not checked, and links in code are text.

A link is no import. It says that one document concerns another, and a change at either end makes the other suspect ([§6](the-state.md)). No section means anything special to the tool.

## Types

A type is a document that another document names as its type; nothing else marks it. Its text before the first section says what the type is. Each of its sections is a section of its documents, and the text under it says what belongs there. A section is required unless its heading ends with `(optional)`.

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

Where a type says that a section holds a link to something, each document links it; where it says only what belongs there, the document states it directly. A strong statement that every document of a type must keep belongs in the type, so that a contradicting document cannot arise.

## Packages and templates

- **A package** is a folder of types, such as `core` (concept, decision, goal, role, rule, term), `software` or `company`. A type in one package may link to types of another.
- **A template** is a folder of documents that name their types by links into the packages and leave their sections empty. Every folder of a template is a template too, so a template applies at any level: a whole project, one area of the domain, or a field such as fire protection.
- **`apply`** copies a template into the spec, together with every package its documents need. A package the spec already has is never overwritten: the spec owns its copy and may change it. After `apply`, `check` lists the empty sections, and that list is the work still to do.

The library of packages and templates, the type of a report, the reference values and the question catalogs ship inside the tool, in its version ([§12](jobs-and-runners.md#the-project)).

## Areas

An area is a folder of the spec with a README whose type is an area type. The sections of the area type are its table of contents, each linking to the README of an area below or to the documents it holds. For a software project the areas are Purpose, Context, Domain, Interfaces, Quality and Architecture ([§4](what-goes-in-a-spec.md)). A rule for one module links that module's README in a section such as "Applies to".

## The writing rule

Every fact stands in exactly one place, and everywhere else links to it. A person's role is stated in that person's document, not again in the role's. This rule is not checked, and it matters more than any check: contradictions come from the same fact standing in several places, and no type system catches them.

An agent writes the structure, the sections and the links, as it writes the prose. Nothing derives the structure from the prose.

## What `check` catches

| Code | Meaning |
|---|---|
| `title` | The title is missing, or there is a second level-1 heading |
| `type.unresolved` | The type line's link points to no document |
| `type.line` | A line starting with `Type:` that is not exactly one link |
| `section.missing` | A required section of the type is missing |
| `section.empty` | A section of the type has no text |
| `link.unresolved` | A link to a `.md` path points to no document of the spec |

What the language does not catch, contradictions between values, gaps and wrong values, a reader finds ([§5](is-the-spec-good.md)). There are no value forms, check rules or declared scopes: something new joins only when a real project needs it.
