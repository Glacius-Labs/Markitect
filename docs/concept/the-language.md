# The language

A spec is documentation with a fixed structure, in plain Markdown ([DEC-121](../decisions.md#dec-121-a-spec-is-plain-markdown-with-types-and-modules)). A type is a class, a document of it its object, and a section the type declares a property; modules are folders that group documents and say what they show. A spec renders and links on GitHub, in editors and in note tools alike.

## Documents

A spec is a folder, by default `docs/spec/`, and every `.md` file in it, at any depth, is a document. Folders group documents into [modules](#modules): by feature, by kind, or mixed.

- **Path.** A document is known by its path below the spec folder, such as `lending/borrow.md`. File names use lowercase letters, digits and `-`, so that links stay plain, except a module's `README.md`; the readable name is the title.
- **Title.** The first line with content is a level-1 heading, the title. No other level-1 heading follows.
- **Sections.** Each level-2 heading `## …` opens a section.
- **Links.** A link is an ordinary Markdown link with a relative path to a `.md` file, `[Librarian](../team/librarian.md)`, optionally with a section, `[Librarian](../team/librarian.md#does)`. Every such link must point to a document of the spec that is shown to it ([below](#modules)). Web addresses, `mailto:` and a bare `#section` are not checked, and links in code are text.

A link says that one document concerns another, and where it stands decides what more it says ([below](#types)): the type line, the extends line and a property type line name types, a link in a reference property is a reference, and a link anywhere else in the text is a mention. A section means something to the tool only as a property of a type; a module's README has a form of its own.

## Types

A type is a class, and a document that names it in its type line is its object. A document is a type when another document names it in its type line, its extends line or a property type line, when it extends a type itself, or when it has a property type line; nothing else marks it. Its text before the first section says what the type is. Each of its sections declares a **property** of its objects, and the text under it says what belongs there. A property is required unless its heading ends with `(optional)`. A project defines its own types, and the default ships six general ones in the module `core`: concept, decision, goal, role, rule and term ([§7](the-project.md#the-default)).

**A property names its type** in a line directly under its heading, the property type line ([DEC-157](../decisions.md#dec-157-a-property-names-its-type-and-modules-depend-on-each-other-without-a-cycle)):

- **Text.** Without the line, the property holds text. A link in it is a **mention**: it says only that one document concerns another.
- **References.** With links separated by commas, `Type: [Role](../core/role.md)`, every link in the property is a **reference** and points to an object of a named type or of a type derived from one. With several links, the object may be of any one of their types. The property holds any number of references.
- **Fixed values.** With code spans separated by commas, `` Type: `planned`, `offered`, `withdrawn` ``, the first line of the section in an object is one of the values.

So a type says what each property holds before it has a single object. A property that sometimes holds a reference and sometimes something else has no form: it is a sign that it should be cut more cleanly.

```markdown
# Operation

One thing a caller can do to the system or ask of it.

## Does
What it achieves.

## Called by
Type: [Role](../core/role.md)

The roles that may call it.

## Status
Type: `planned`, `offered`, `withdrawn`

Whether callers can rely on it yet.

## Refused when (optional)
The conditions under which it is refused, each with its reason and the answer the caller gets.
```

A document names its type in the first line after the title, `Type: ` followed by one link, and is then an **object** of that type. It has every required property of its type as a section, in any order, each with some text, and each value in the form the property's type gives; it may add sections of its own. Here the project keeps its own types in a module `types`, whose README shows them to the other modules, and the role Member is an object of `core`'s Role:

```markdown
# Borrow

Type: [Operation](../types/operation.md)

## Does
Lends a copy of a book to a member.

## Called by
[Member](../members/member.md)

## Status
offered

## Refused when
The member already has five open loans: the answer is `limit_reached`.
```

The link under `Called by` is a reference, and `check` reports it when Member is no role. A link under `Does` would be a mention, which no type constrains.

A type may extend another, its base, with a line of its own in the place of the type line, `Extends: ` followed by one link ([DEC-155](../decisions.md#dec-155-a-type-extends-another-with-a-line-of-its-own)). No document has both lines. A type that extends another is never an object of it: it needs none of its base's properties, so a type that has no object yet is still a type. Its properties are its own and those of every type up its chain, the base's first; an object of it has every required property of the whole chain. Here a second type of the project, `types/query.md`, extends the Operation above, and an object of it needs the properties of both:

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

## Called by
[Member](../members/member.md)

## Status
planned

## Answers
Each open loan with its copy and due date.
```

A type may describe a property of its base more precisely, under the same heading, or make an optional one required; it never makes a required one optional, and never gives it another type: declared again without a property type line, it keeps the type it inherits. A chain that comes back to itself is an error. Only an extends line passes properties on: a type that has a type line of its own inherits nothing from the type it names.

Where a property is a reference, each object links what it refers to; where it is text, the object states it directly. A strong statement that every object of a type must keep belongs in the type, so that a contradicting object cannot arise.

## Values

A value, such as the status of an operation or the role responsible for a rule, stands in the first line of its section, and the property's type gives its form: a fixed value is exactly one of the values its type lists, and a reference is a link to an object of a named type, of which a reference property may hold several. In a property without a property type line, the value is text, and a link in it is a mention: `core`'s Rule declares no type for `Responsible`, so a role linked there is a mention, and a project that wants it checked gives its copy of Rule a property type line. There are no lines in the head but the type line, or in a type the extends line.

## Modules

**A module is a folder, and it only groups** ([DEC-156](../decisions.md#dec-156-a-module-is-a-folder-and-its-readme-says-what-it-shows)). It is no type, and there are no kinds of modules. The folder is its namespace. Every folder of the spec that holds a document directly is a module and has a README; a folder without documents of its own, such as one that holds only images or only other folders, is none. Modules nest as their folders do.

**The README is the manifest of its module.** It has no type line, since its form belongs to the language, and it is never a type:

- The text before its first section says what the module is for.
- `Responsible`, optional, links the role that answers for the module. Without it, the role of the module above answers.
- `Realized in`, optional, says where in the project the module is built.
- `Contents` lists what the module shows to the other modules.

A README is never itself the thing it describes: in a module `shop`, the shop is a document `shop/shop.md` of its own, and the README says what the module holds.

**Public and internal.** What a README links under `Contents`, a document of its folder or the README of a module directly below, is public; the README itself is public as well. Everything else in the module is internal, and a link to it from outside the module is an error. Inside a module a document may link everything of its own folder, and what is internal to the modules above it, as with Go's internal packages; a module below shows it only what that module's README lists under `Contents`. This holds for every link, the type line, the extends line and the property type line included: a type that other modules use stands in the `Contents` of its module, as `core`'s README lists its six types.

**Patterns.** A README may also show documents of its folder by a pattern over their file names, written as code: in a module `reports` whose monthly reports keep arriving, `` `report-*.md` `` under `Contents` shows each new one without a change to the README, and `` `*.md` `` would show the whole folder. A pattern never reaches into a module below.

A library has a module `lending` with one public and one internal document:

```text
docs/spec/
  core/         the default's module: README.md and six types
  lending/
    README.md
    borrow.md       public, an Operation
    loan-limit.md   internal, a Rule
  members/
    README.md
    card.md
    member.md       public, a Role
  types/
    README.md       shows both types
    operation.md
    query.md
```

```markdown
# Lending

How members borrow copies and bring them back.

## Contents
- [Borrow](borrow.md)
```

`borrow.md` may link `loan-limit.md`, and so may the README: they are inside the module. In `members/card.md`, `[Borrow](../lending/borrow.md)` is right, and `[Loan limit](../lending/loan-limit.md)` is an error of `check`, `link.internal`: the rule is internal to `lending`. The rule's type line, `Type: [Rule](../core/rule.md)`, is right, since `core` shows Rule.

**No cycle.** Modules do not depend on each other in a cycle ([DEC-157](../decisions.md#dec-157-a-property-names-its-type-and-modules-depend-on-each-other-without-a-cycle)). A module depends on another when one of its objects names a type of the other in its type line, one of its types extends a type of the other or links one in a property type line, or one of its objects has a reference to a document of the other. A parent module and a module below it count like any two, although the module below may link what is internal to the one above. A mention never counts, and neither do the links of a README. The library above has none: `lending` depends on `types`, `members` and `core`, `types` and `members` on `core`, and the link from `members/card.md` to Borrow is a mention. A cycle arises when a type in each of two modules names the other in a property type line:

```text
lending/loan.md         ## Holder   Type: [Membership](../members/membership.md)
members/membership.md   ## Loans    Type: [Loan](../lending/loan.md)
```

`check` reports it once, with the chain of modules, `lending -> members -> lending`, and for each step a link that causes it. Here one fact stands in two places: a membership's loans are the loans that it holds, so `Loans` goes, and the fact stays in the loan. Where two modules truly need each other, they are one module, or what both need moves into a module of its own.

## The writing rule

Every fact stands in exactly one place, and everywhere else links to it. A person's role is stated in that person's document, not again in the role's. This rule is not checked, and it matters more than any check: contradictions come from the same fact standing in several places, and no type system catches them.

An agent writes the structure, the sections and the links, as it writes the prose. Nothing derives the structure from the prose.

## What `check` catches

| Code | Meaning |
|---|---|
| `title` | The title is missing, or there is a second level-1 heading |
| `type.unresolved` | The type line's link points to no document |
| `type.line` | A line starting with `Type:` that is not exactly one link, stands neither in the type line's place nor in a property type line's, or follows an extends line |
| `property.line` | A property type line that is not links or code spans separated by commas, mixes the two, or stands in an object or a README |
| `property.unresolved` | A link of a property type line points to no document |
| `property.retyped` | A derived type gives an inherited property another type |
| `reference.type` | A reference in an object points to a document that is no object of a type its property names, or of a type derived from one |
| `property.value` | The first line of a fixed-values section in an object is none of the values |
| `extends.unresolved` | The extends line's link points to no document |
| `extends.line` | A line starting with `Extends:` that is not exactly one link, is not in the type line's place, or follows a type line; the document is then read without it |
| `extends.cycle` | A chain of extends lines comes back to itself |
| `extends.loosens` | A type marks optional a property that a type up its chain requires |
| `section.missing` | An object lacks the section of a required property of its type, or of a type up its chain |
| `section.empty` | The section of a property of the type, or of a type up its chain, has no text |
| `link.unresolved` | A link to a `.md` path points to no document of the spec |
| `link.internal` | A link, a type line, an extends line or a property type line from outside a module points to a document that the module's README does not show under `Contents` |
| `module.readme` | A folder that holds a document directly has no README |
| `module.cycle` | Modules depend on each other in a cycle |
| `readme.pattern` | A pattern in a README's `Contents` is not valid |
| `readme.type` | A README has a type line or an extends line, or a type line or an extends line names a README |

What the language does not catch, contradictions between values, gaps and wrong values, a reader finds ([§5](is-the-spec-good.md)). Beyond references and fixed values there are no value forms, and there are no check rules: something new joins only when a real project needs it.
