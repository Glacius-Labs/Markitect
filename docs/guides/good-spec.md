# What makes a good spec

A Markitect spec is judged like code ([founding intent](../concepts/sources/founding-intent-20261026.md)). This guide says what a well designed spec looks like, borrowing the principles that keep code changeable, and how to tell. It is practical advice in the manner of *Effective Go*, not a rule set: granularity stays the author's choice ([DEC-005](../concepts/register.md#dec-005-model-granularity-is-the-authors-responsibility)), and `markitect check` alone decides whether a spec is valid. The [model language guide](model-language.md) owns the language; this page owns how to use it well. Its numbers come from the [measurements of 7 November 2026](../validation/spec-design-measures-20261107.md) on the two archived Makerspace specs, 115 and 419 Definitions.

## Why design matters here: links are the build graph

Each Definition is a unit of verification, and its inputs are its own text, its Kind, the Definitions it links to, its registry rows and the evidence ([DEC-099](../concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)). So the links say what a change puts back on the list of `markitect status`. A change has two costs:

- **Verification:** the changed Definition and every Definition that links to it, or is of it as Kind, go back on the list. That is one step of links, its **fan-in**.
- **Review by the author:** what may need to change in the spec with it, which is everything that reaches it along the links, its **every-link reach**.

A good spec keeps both small and true. Every link names what the Definition relies on, and nothing else: a link it does not need re-verifies too much, and a link it lacks lets a change pass unverified. A typical Makerspace Definition is linked once, puts 2 Definitions back on the list and reaches about 8 by every link. The design problems all sit in the few that reach far.

The rules below each name their **signal**: a measure of `markitect vet` that exists today, one the [`vet` plan](../concepts/vet-plan.md) proposes, or what the auditor checks. The auditor is whoever reviews the spec: the owner, or an agent in a modeling session.

## The rules at a glance

| Principle | Rule | Signal |
|---|---|---|
| [Bounded contexts](#bounded-contexts-and-ubiquitous-language) | One Package per area of the domain, in that area's words | Auditor |
| [The Dependency Rule](#the-dependency-rule-and-layers) | Link inward: from what changes often to what changes rarely | `package.cycle`, `link.downward`, `vet.unstable_dependency` |
| [Stable abstractions](#stable-dependencies-and-stable-abstractions) | What many link to states what something is, not what happens | `vet.unstable_dependency`; proposed `vet.hub` |
| [Cohesion and coupling](#cohesion-and-coupling) | A Package's Definitions link each other; it depends on few Packages | `vet.coupling`; proposed `vet.cohesion` |
| [Complexity](#complexity-of-a-definition) | One idea per Definition, a few hundred words, a dozen statements | `vet.definition_size`; proposed `vet.statements` |
| [Evidence for complexity](#evidence-for-complexity-a-crap-analogue) | The larger a Definition, the more evidence its `ok` names | Proposed `vet.risk` |
| [Accidental complexity](#accidental-complexity) | Write only what good documentation would state | `definition.unused`, `vet.thin_package`; auditor |
| [Abstraction levels](#one-level-of-abstraction-per-definition) | One level per Definition | Auditor |
| [Placement](#everything-has-its-place) | Each Definition in the lowest Package that holds all its users; Root minimal | `vet.package_size`; proposed `vet.placement`, `vet.hub` |
| [Say it once](#say-it-once-link-exactly-what-you-use) | One fact in one place; link exactly what you use, directly | Proposed `vet.hub`; auditor |

## Bounded contexts and ubiquitous language

**Rule.** Cut the spec into Packages by area of the domain, as domain-driven design cuts a system into bounded contexts. Each Package uses the area's own words: a term means one thing inside it, and titles and wire names say what the people of that area say. A term that two areas use differently gets a Definition in each. A term both use the same way sits at their lowest common parent.

**Why.** A change in one area stays in one Package, and the links that cross Packages are the only places where areas touch. A reader of one Package learns one vocabulary.

**Bad.** Makerspace's Root is both the project's anchor and a module: its project page names "the shared policies module, which this folder describes". Formats, refusal codes, audit and authorization, with `list-audit-entries`, the only Operation at Root, sit beside the project's anchor and its project-wide Rules. Two areas share one Package, and one of them is the Package every other one inherits.

**Good.** `members`, `equipment`, `billing`, `training`, `bookings`, `materials`, `safety` and `reporting` each hold one area: its Concepts, its Operations, its Interface and the Rules only it follows. `billing/charge` and `materials/purchase` are separate Concepts, each in the words of its area.

**Signal.** The auditor checks that each Package holds one area and that one term means one thing in it. No measure sees vocabulary.

## The Dependency Rule and layers

**Rule.** Dependencies point inward, from what changes often to what changes rarely: from the wire protocol and the interfaces to the operations, from the operations to the domain's Concepts, and from those to the formats and project-wide Rules. In a Markitect spec the tree already has a direction. A child links up freely, and a parent may not link into a child (`link.downward`), so a parent is always inner to its children. Between siblings, the outer one links the inner one, never back. Name the layers of the project in its architecture Decision at Root.

**Why.** What lies inward changes rarely. A change to an outer Definition puts nothing inward back on the list, and an inward change reaches outward only through links that say so.

**Bad.** In the 5× project, `billing` (instability 0.15) depends on `households` (0.33): a core module points at a less stable one, so a change in `households` reaches the core.

**Good.** Makerspace's `protocol`, the outer layer, depends on 8 modules, and no Package depends on it. The area modules link each other's Operations only through their Interfaces.

**Signal.** `package.cycle` and `link.downward` are errors of `check`. `vet.unstable_dependency` warns when a Package depends on a less stable one, which is how a measure can see the Dependency Rule without knowing the layers. The auditor checks the edges against the layers the architecture names. Whether Kinds should declare a layer, so that `vet` can check it, is a [question for the owner](../concepts/vet-plan.md#should-kinds-declare-a-layer).

## Stable dependencies and stable abstractions

**Rule.** What many Definitions link to must be stable, and stable things should be abstract. In a spec, the abstract Definitions are the types and contracts: Kinds, Concepts, formats, Interfaces and Rules. The concrete ones say what happens: Operations, Features, Endpoints and Commands. Link to behaviour only from its own module's Interface. When another module must change an operation's behaviour, let it state its need and link the operation, as an inverted dependency.

**Why.** A change to a Definition puts back on the list everything that links to it. If many link to it, it has to be something that rarely changes, and a type changes more rarely than a procedure.

In code the Package is the unit that is released and depended on, so the principle is read per Package, as a distance from the "main sequence". In Markitect the unit of verification is the Definition: a change to `members/register-member` puts back itself and its Interface, however many depend on `members/member` beside it. So read the principle per Definition. Measured per Package, every Makerspace area Package is half to two-thirds abstract whatever its stability, because it holds its vocabulary beside its operations, and the distance from the main sequence tells good Packages from bad ones no better than instability alone.

**Bad.** A Makerspace field list that links `values`, a page of thirteen formats, depends on the formats of money and of clock times when it uses only identifiers.

**Good.** In Makerspace, Concepts carry the fan-in (median 6) and Operations do not (at most 2). `members/member` is linked by 24 Definitions; `members/register-member` by its Interface alone.

**Signal.** `vet.unstable_dependency` for Packages. For Definitions, the proposed `vet.hub` warns when one change would put more than a quarter of the spec back on the list.

## Cohesion and coupling

**Rule.** A Package's Definitions link each other more than they link out, links up the tree aside. A Package whose Definitions fall into groups with no link between them is two Packages. A Package depends on few others, and on each through its Interface.

**Why.** A change stays inside its Package, and a reader of a Package needs few others to understand it.

**Bad.** The 5× project's `bookings` depends on 12 Packages. Each of them can put `bookings` Definitions back on the list, and a reader of `bookings` reads twelve Interfaces.

**Good.** `equipment` and `members` link only inside and up, and depend on no other Package. Of the links a typical Makerspace Package makes, links up aside, 58 % stay inside. A Rule that binds its Package by scope and links nothing, such as `billing/due-date-and-block`, is fine: scope is location.

**Signal.** `vet.coupling` warns above 5 dependencies. The proposed `vet.cohesion` warns about a Package that falls into two or more groups of two or more Definitions each; no Package of either Makerspace spec does. The outer layer is the exception by design: `protocol` wires other modules and has little cohesion of its own.

## Complexity of a Definition

**Rule.** One idea per Definition. Keep it to a few hundred words and about a dozen statements, a statement being a list item, a table row, a refusal or a state. Each statement is a branch the reviewer verifies, as each condition is a branch of a function. When a Definition grows past that, split it along its ideas.

**Why.** A reviewer verifies a Definition as one item. The more it states, the more a review can miss, and a change to one of its statements puts all the others back on the list too.

**Bad.** The 5× project's `list-order` is one table of the order of every listed result, 67 rows: one item for the reviewer with 67 things to check, and a change to one row puts the whole table back on the list.

**Good.** A typical Makerspace Definition has 88 words and 6 statements; nine in ten stay under 253 words and 12 statements. `billing/record-payment` states its effect, wire name, arguments, refusals, result, permissions and audit target in eight list items.

**Signal.** `vet.definition_size` warns above 1000 words, which no Makerspace Definition reaches (the largest has 430 words, 995 in the 5× project). The proposed `vet.statements` warns, under `--strict`, above 20 statements.

## Evidence for complexity: a CRAP analogue

**Rule.** The larger a Definition, the more evidence its `ok` should name. A large Definition verified on one pointer is a risk, as a complex function without tests is.

CRAP, the change-risk measure for code, is *comp² × (1 − cov)³ + comp*, with *comp* the cyclomatic complexity and *cov* the share covered by tests; above 30 is a risk. Its analogue for a Definition takes *c* = 1 + its statements and *e* = the evidence pointers of its `ok` in `.markitect/review.sum`, divided by *c*, at most 1. So *risk = c² × (1 − e)³ + c*. A Definition with 7 statements (*c* 8) verified on two pointers scores 64 × 0.42 + 8 ≈ 35; on four pointers, 16.

**Why.** It finds what is both hard to verify and thinly verified. As with CRAP, a Definition of more than about 30 statements stays above the line however much evidence it names: the remedy is to split it.

**Bad and good.** No record of a real build exists for Makerspace yet, so there are no measured examples. Without any record, 73 of its 115 Definitions would score above 30, which is why the measure only looks at Definitions recorded `ok`.

**Signal.** The proposed `vet.risk`, under `--strict` only, until a real record calibrates it.

## Accidental complexity

**Rule.** Write what good documentation of the project would state, and nothing more ([DEC-079](../concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation)). No Definition that only points elsewhere, no Kind used once, no Package that only adds a level, no restating of a fact another Definition holds.

**Why.** Every Definition is an item to verify and every link one to keep true. What the domain does not need is cost without value, for the reviewer and the reader.

**Bad.** Restating: 21 of Makerspace's 38 Operations say "Refusals at check 7", a fact `order-of-checks` already holds and the `Operation` Kind holds again. When the order of checks changes, 23 places have to.

**Good.** Makerspace's two project Kinds both earn their place: `Operation` is the form of 38 Definitions and `Interface` of 8. No Package of either spec is thin.

**Signal.** `definition.unused` is an error of `check` and `vet.thin_package` a warning. Restating and pointer-only Definitions are the auditor's.

## One level of abstraction per Definition

**Rule.** A Definition stays at one level. A Kind says what its Definitions contain. A Concept says what a thing is. A Rule states a constraint. An Operation says what happens. The project anchor and the architecture say what the whole is. Detail goes down into a Definition of its own, and a summary goes up.

**Why.** A change at one level then leaves the others alone. A reader finds each fact at the level where it is decided.

**Bad.** The `Operation` Kind states both the form of an Operation (wire name, arguments, refusals, result, permissions, audit target) and rules of the protocol: an argument that names a record must name one that exists, refusals are checked at check 7. To do so it links six Root Definitions. A Kind's link reaches all its Definitions when every link is followed, so after `values` was split, the Kind carried most of what was left of the widest reaches: 44 of 55 Definitions on Makerspace's step c1.

**Good.** The `Interface` Kind says what an Interface Definition contains and links nothing. Each format of the split `values` is one Concept at one level: its JSON type, its range and what it means.

**Signal.** The auditor's. A measure can point at a Kind that links Definitions other than Kinds, but whether such a link mixes levels needs judgment.

## Everything has its place

**Rule.** Put each Definition in the lowest Package that holds every Definition that uses it, its lowest common parent. Root holds the project anchor, the architecture, the project-wide Rules and the Kinds that several Packages use, and little else. A module's vocabulary goes into a Package of its own, even when every other module uses it.

**Why.** Placement is scope: a Rule at Root binds the whole project. It is also visibility. And Root can link nothing below it, so whatever a Root Definition needs must sit at Root too: a module at Root pulls its own dependencies up after it. Links up the tree are free, so coupling to Root shows in no Package's coupling.

**Bad.** Makerspace's Root holds 21 Definitions, among them the formats, the refusal codes, audit and authorization. The Root Kind `Operation` links `audit`, `audit-entry`, `authorization` and `refusal-codes`, so none of them can move down. With the thirteen formats split out at Root, Root would hold 33 Definitions.

**Good.** Each area's Concepts sit in its area Package, and the two shared Kinds sit at Root. Split into a `values/` Package, the formats are visible as one: each area Package now shows an edge to `values`.

**Signal.** `vet.package_size` warns above 25 Definitions, Root included. The proposed `vet.placement` warns, under `--strict`, about a Definition whose users all lie below it in one child Package. The proposed `vet.hub` counts Root, which `vet.reach` skips today.

## Say it once, link exactly what you use

**Rule.** State each fact in one Definition. Every Definition that relies on it links it, directly, and links nothing it does not rely on. Link the smallest Definition that holds what you use. To name an area without depending on all of it, link its Package: a Package link is no input of the record and puts nothing back on the list.

**Why.** The links are the build graph. A link the Definition does not need puts it back on the list for changes it does not care about. A missing link lets a change pass: a statement relied on through a chain of links, or because a Rule binds by scope, is not put back on the list when its source changes, since `status` follows one step. Files that change with the code catch some of that, and `/markitect-review --all` is the safety net.

**Bad: one page for everything.** 60 Makerspace Definitions link `values`, thirteen formats, and use a median of 2 of them; in the 5× project 269 link it and use 1. Any change to `values`, even to the clock time only two Definitions use, put 61 Definitions back on the list, 270 in the 5× project.

```markdown
**Fields**, in the formats of [[values]]:

- `memberId`: identifier, naming an existing [[members/member]]
- `blocked`: flag
```

**Good: one Definition per format, linked where it is used.** After the split, a change to one format lists between 1 and 50 Definitions, a median of 4. Following every link, Makerspace's four widest change steps fall from 85–88 % of the spec to 45–60 %. The header still names the area, as a Package link.

```markdown
**Fields**, in the formats of [[values]]:

- `memberId`: [[values/identifier|identifier]], naming an existing [[members/member]]
- `blocked`: [[values/flag|flag]]
```

**Bad: relying through a chain.** 21 Operations say "Refusals at check 7", but only the `Operation` Kind links `order-of-checks`. A change to the order of checks puts the Kind back on the list, not those 21 Operations. Linking `order-of-checks` from each Operation makes it a true hub, which is right: then keep it small and stable.

**A milder case.** 17 of the 24 Definitions that link `members/member` do so only to say that an identifier names an existing member; a new field of the member puts them all back on the list. Split such a part out only when it changes on its own.

**Signal.** The proposed `vet.hub` points at a Definition many link to. Whether they use all of it, or only part, is the auditor's question, and so is whether a Definition relies on something it does not link.

## Measures at a glance

| Measure | Formula | Makerspace c8 | 5× project c8 | Signal |
|---|---|---|---|---|
| Fan-in | Definitions that link to it or are of it as Kind | median 1, p90 7, max 60 | 1, 7, 269 | Proposed `vet.hub` |
| One-step reach | 1 + fan-in: what `status` lists after a change to it alone | 2, 8, 61 | 2, 8, 270 | Proposed `vet.hub` |
| Every-link reach | Everything that reaches it along the links | 8, 52, 103 | 8, 85, 402 | Read, not warned |
| Fan-out | Definitions it links to | 2, 8, 16 | 4, 9, 24 | Read, not warned |
| Efferent coupling Ce | Packages a Package depends on | at most 8 | at most 12 | `vet.coupling` |
| Instability I | Ce / (Ca + Ce) | 0 to 1 | 0 to 1 | `vet.unstable_dependency` |
| Abstractness A, distance D | Abstract / (abstract + concrete); \|A + I − 1\| | A 0.5 to 1 | A 0.33 to 1 | Read, not warned |
| Cohesion | Groups of a Package joined by its own links | none split | none split | Proposed `vet.cohesion` |
| Size | Words in a Definition's body | median 88, max 430 | 99, 995 | `vet.definition_size` |
| Statements | List items and table rows | median 6, max 25 | 7, 67 | Proposed `vet.statements` |
| Risk | c² × (1 − e)³ + c | no record yet | no record yet | Proposed `vet.risk` |

The [`vet` plan](../concepts/vet-plan.md) proposes the thresholds, and which warnings apply only with `--strict`. [The model language guide](model-language.md#vet-warnings) lists what `vet` warns about today.
