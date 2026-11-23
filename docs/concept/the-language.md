# The language

A spec is documentation with a fixed structure, in plain Markdown ([DEC-121](../decisions.md#dec-121-a-spec-is-plain-markdown-with-types-and-modules)). There are types and documents, as there are classes and objects, and modules, folders that group documents and say what they show. It renders and links on GitHub, in editors and in note tools alike.

## Documents

A spec is a folder, by default `docs/spec/`, and every `.md` file in it, at any depth, is a document. Folders group documents into [modules](#modules): by feature, by kind, or mixed.

- **Path.** A document is known by its path below the spec folder, such as `lending/borrow.md`. File names use lowercase letters, digits and `-`, so that links stay plain, except a module's `README.md`; the readable name is the title.
- **Title.** The first line with content is a level-1 heading, the title. No other level-1 heading follows.
- **Sections.** Each level-2 heading `## …` opens a section.
- **Links.** A link is an ordinary Markdown link with a relative path to a `.md` file, `[Librarian](../team/librarian.md)`, optionally with a section, `[Librarian](../team/librarian.md#does)`. Every such link must point to a document of the spec that it may see ([below](#modules)). Web addresses, `mailto:` and a bare `#section` are not checked, and links in code are text.

A link is no import. It says that one document concerns another. No section of a document means anything special to the tool; only a module's README has a form of its own.

## Types

A type is a document that another document names as its type, that another type extends, or that extends a type itself; nothing else marks it. Its text before the first section says what the type is. Each of its sections is a section of its documents, and the text under it says what belongs there. A section is required unless its heading ends with `(optional)`. A project defines its own types, and the default ships six general ones in the module `core`: concept, decision, goal, role, rule and term ([§7](the-project.md#the-default)).

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

A document names its type in the first line after the title, `Type: ` followed by one link. It then has every required section of its type, in any order, each with some text; it may add sections of its own. Here the project keeps its own types in a module `types`, whose README shows them to the other modules:

```markdown
# Borrow

Type: [Operation](../types/operation.md)

## Does
Lends a copy of a book to a member.

## Given
The copy's barcode and the member's card.

## Refused when
The member already has five open loans: the answer is `limit_reached`.
```

A type may extend another, its base, with a line of its own in the place of the type line, `Extends: ` followed by one link ([DEC-155](../decisions.md#dec-155-a-type-extends-another-with-a-line-of-its-own)). `Type:` stands only in a document of a type, and no document has both lines. A type that extends another is never a document of it: it needs none of its base's sections, so a type that no document names yet is still a type. Its sections are its own and those of every type up its chain, the base's first; a document of it has every required section of the whole chain. Here a second type of the project, `types/query.md`, extends the Operation above, and a document of it needs the sections of both:

```markdown
# Query

Extends: [Operation](operation.md)

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

## Values

A value, such as the state of a concept or the role responsible for a rule, stands in the first line of its section, `## Responsible` with a link to a role under it. There are no lines in the head but the type line, or in a type the extends line.

## Modules

**A module is a folder, and it only groups** ([DEC-156](../decisions.md#dec-156-a-module-is-a-folder-and-its-readme-says-what-it-shows)). It is no type, and there are no kinds of modules. The folder is its namespace. Every folder of the spec that holds documents is a module and has a README; a folder without documents, such as one that holds only images or only other folders, is none. Modules nest as their folders do.

**The README is the manifest of its module.** It has no type line, since its form belongs to the language, and it is never a type:

- The text before its first section says what the module is for.
- `Responsible`, optional, links the role that answers for the module. Without it, the role of the module above answers.
- `Realized in`, optional, says where in the project the module is built.
- `Contents` lists what the module shows to the other modules.

A README is never itself the thing it describes: in a module `shop`, the shop is a document `shop/shop.md` of its own, and the README says what the module holds.

**Public and internal.** What a README links under `Contents`, a document of its folder or the README of a module directly below, is public; the README itself is public as well. Everything else in the module is internal, and a link to it from outside the module is an error. Inside a module everything may link to everything, also to what is internal to the modules above it, as with Go's internal packages. This holds for every link, the type line and the extends line included: a type that other modules use stands in the `Contents` of its module, as `core`'s README lists its six types.

**Patterns.** A README may also show documents of its folder by a pattern over their file names, written as code: in a module `reports` whose monthly reports keep arriving, `` `report-*.md` `` under `Contents` shows each new one without a change to the README, and `` `*.md` `` would show the whole folder. A pattern never reaches into a module below.

A library has a module `lending` with one public and one internal document:

```text
docs/spec/
  core/         the default's module: README.md and six types
  lending/
    README.md
    borrow.md       public
    loan-limit.md   internal, a Rule
  members/
    README.md
    card.md
```

```markdown
# Lending

How members borrow copies and bring them back.

## Contents
- [Borrow](borrow.md)
```

`borrow.md` may link `loan-limit.md`, and so may the README: they are inside the module. In `members/card.md`, `[Borrow](../lending/borrow.md)` is right, and `[Loan limit](../lending/loan-limit.md)` is an error of `check`, `link.internal`: the rule is internal to `lending`. The rule's type line, `Type: [Rule](../core/rule.md)`, is right, since `core` shows Rule.

## The writing rule

Every fact stands in exactly one place, and everywhere else links to it. A person's role is stated in that person's document, not again in the role's. This rule is not checked, and it matters more than any check: contradictions come from the same fact standing in several places, and no type system catches them.

An agent writes the structure, the sections and the links, as it writes the prose. Nothing derives the structure from the prose.

## What `check` catches

| Code | Meaning |
|---|---|
| `title` | The title is missing, or there is a second level-1 heading |
| `type.unresolved` | The type line's link points to no document |
| `type.line` | A line starting with `Type:` that is not exactly one link, is not in the type line's place, or follows an extends line |
| `extends.unresolved` | The extends line's link points to no document |
| `extends.line` | A line starting with `Extends:` that is not exactly one link, is not in the type line's place, or follows a type line; the document is then read without it |
| `extends.cycle` | A chain of extends lines comes back to itself |
| `extends.loosens` | A type marks optional a section that a type up its chain requires |
| `section.missing` | A required section of the type, or of a type up its chain, is missing |
| `section.empty` | A section of the type, or of a type up its chain, has no text |
| `link.unresolved` | A link to a `.md` path points to no document of the spec |
| `link.internal` | A link, a type line or an extends line from outside a module points to a document that the module's README does not show under `Contents` |
| `module.readme` | A folder that holds documents has no README |
| `readme.pattern` | A pattern in a README's `Contents` is not valid |
| `readme.type` | A README has a type line or an extends line, or a type line or an extends line names a README |

What the language does not catch, contradictions between values, gaps and wrong values, a reader finds ([§5](is-the-spec-good.md)). There are no value forms, check rules or declared scopes: something new joins only when a real project needs it.
