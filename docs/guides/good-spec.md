# What makes a good spec

A Markitect spec is judged like code ([founding intent](../concepts/sources/founding-intent-20261026.md)). This guide says what a well designed spec looks like, borrowing the principles that keep code changeable, and how to tell. It is practical advice in the manner of *Effective Go*, not a rule set: granularity stays the author's choice ([DEC-005](../concepts/register.md#dec-005-model-granularity-is-the-authors-responsibility)), and `markitect check` alone decides whether a spec is valid. The [model language guide](model-language.md) owns the language; this page owns how to use it well. Its numbers come from the [measurements of 7 November 2026](../validation/spec-design-measures-20261107.md) on the two archived Makerspace specs, 115 and 419 Definitions.

## Why design matters here: links are the build graph

Each Definition is a unit of verification, and its inputs are its own text, its Kind, the Definitions it links to, its registry rows and the evidence ([DEC-099](../concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)). So the links say what a change puts back on the list of `markitect status`. A change has two costs:

- **Verification:** the changed Definition and every Definition that links to it, or is of it as Kind, go back on the list. That is one step of links, its **fan-in**.
- **Review by the author:** what may need to change in the spec with it, which is everything that reaches it along the links, its **every-link reach**.

A good spec keeps both small and true. Every link names what the Definition relies on, and nothing else: a link it does not need re-verifies too much, and a link it lacks lets a change pass unverified. A typical Makerspace Definition is linked once, puts 2 Definitions back on the list and reaches about 8 by every link. The design problems all sit in the few that reach far.

The rules below each name their **signal**: a warning or a note of `markitect vet` ([DEC-102](../concepts/register.md#dec-102-what-vet-measures)), a measure that waits for data, or what the auditor checks. The auditor is whoever reads the spec for its quality: the owner, or the read-only auditor that `/markitect-audit` starts ([DEC-100](../concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)), which reports to the owner. The tool treats every Kind alike, a project's own Kinds as much as `std`'s; what a Kind means is for the people and agents who read its text.

**What a measure can see.** Coupling, instability, abstractness counted from Kinds, fan-in, reach and cycles are properties of the link graph. They do not depend on the language the spec is written in, so they carry over from code as they are. Qualities of the prose do not: how hard a Definition is to follow, whether two Definitions contradict each other, whether a statement is vague. `vet` can only approximate them, by counting statements or finding a name used without a link, and the auditor judges the rest. One difference from code matters most: a missing import does not compile, but prose can name a concept without linking it, so a spec's dependencies are only as complete as its author's links. Every threshold here comes from the two Makerspace specs and stays provisional until more specs calibrate it.

## The rules at a glance

| Principle | Rule | Signal |
|---|---|---|
| [Bounded contexts](#bounded-contexts-and-ubiquitous-language) | One Package per area of the domain, in that area's words | Auditor |
| [The Dependency Rule](#the-package-tree-is-the-dependency-rule) | Links go up the tree or sideways towards stability, never down | `link.downward`, `package.cycle`, `vet.unstable_dependency` |
| [Stable abstractions](#stable-dependencies-and-stable-abstractions) | What many depend on is a form or vocabulary that rarely changes | `vet.hub`, `vet.hotspot`; abstractness and distance in `list --deps` |
| [Cohesion and coupling](#cohesion-and-coupling) | A Package's Definitions link each other; it depends on few Packages | `vet.coupling`; auditor |
| [Complexity](#complexity-of-a-definition) | One idea per Definition, a few hundred words, a dozen statements | `vet.definition_size`, `vet.statements` |
| [Evidence for complexity](#evidence-for-complexity-a-crap-analogue) | The larger a Definition, the more evidence its `ok` names | The reviewer's counts on `mark`; `vet.risk` waits for data |
| [Accidental complexity](#accidental-complexity) | Write only what good documentation would state | `definition.unused`, `vet.thin_package`; auditor |
| [Abstraction levels](#one-level-of-abstraction-per-definition) | One level per Definition | Auditor |
| [Placement](#everything-has-its-place) | Each Definition in the lowest Package that holds all its users; Root minimal | `vet.package_size`, `vet.placement`, `vet.hub` |
| [Say it once](#say-it-once-link-exactly-what-you-use) | One fact in one place; link exactly what you use, directly, from the specific to the general; no central page of values | `vet.unlinked_mention`, which the auditor triages; `vet.hub` |

## Bounded contexts and ubiquitous language

**Rule.** Cut the spec into Packages by area of the domain, as domain-driven design cuts a system into bounded contexts. Each Package uses the area's own words: a term means one thing inside it, and titles and wire names say what the people of that area say. A term that two areas use differently gets a Definition in each. A term both use the same way sits at their lowest common parent.

**Why.** A change in one area stays in one Package, and the links that cross Packages are the only places where areas touch. A reader of one Package learns one vocabulary.

**Bad.** Makerspace's Root is both the project's anchor and a module: its project page names "the shared policies module, which this folder describes". Formats, refusal codes, audit and authorization, with `list-audit-entries`, the only Operation at Root, sit beside the project's anchor and its project-wide Rules. Two areas share one Package, and one of them is the Package every other one inherits.

**Good.** `members`, `equipment`, `billing`, `training`, `bookings`, `materials`, `safety` and `reporting` each hold one area: its Concepts, its Operations, its Interface and the Rules only it follows. `billing/charge` and `materials/purchase` are separate Concepts, each in the words of its area.

**Signal.** The auditor checks that each Package holds one area and that one term means one thing in it. No measure sees vocabulary.

## The Package tree is the Dependency Rule

**Rule.** Clean Architecture's Dependency Rule says that dependencies point from the specific to the general, never back. A Markitect spec has it built in, as Go has it without layers: a child links up to its ancestors freely, and a parent may not link into a child (`link.downward`). So put the general above and the specific below: the project-wide Rules and shared Kinds at Root, an area's vocabulary in its Package, and what only one part of the area needs in a Package below it. Between siblings, link towards the more stable one, never back, and never in a circle. A limit of your own project, such as "nothing but `protocol` uses the operations of another module", is written with what the language has: mark what a Package keeps to itself `internal: true`, and state the rest as an ordinary Rule.

**Why.** What lies above changes rarely, and the tree stops a general Definition from depending on a specific one. A change below puts nothing above back on the list, and a change above reaches below only through links that say so.

**Bad.** In the 5× project, `billing` (instability 0.15) depends on its sibling `households` (0.33): a core module points at a less stable one, so a change in `households` reaches the core. And `access/enter-door` links `billing/get-account`, an operation of another module, not that module's Interface: behaviour built on behaviour.

**Good.** Makerspace's `protocol`, the outer part that maps the request protocol onto the modules, depends on 8 of them, and no Package depends on it. In Makerspace the area modules link each other's Operations only through their Interfaces, and 19 Rules that only their own module follows, all six of `protocol`'s among them, are marked `internal: true`.

**Signal.** `link.downward` and `package.cycle` are errors of `check`. `vet.unstable_dependency` checks Martin's Stable Dependencies Principle on the sideways edges, from the graph alone: it warns about an edge from a more stable Package to a less stable one. A project's own limits are checked by `link.internal` when they are written as visibility, and by the reviewer when they are written as a Rule.

## Stable dependencies and stable abstractions

**Rule.** What many depend on must rarely change, and Martin adds that what is stable should be abstract, so that it can stay put while what builds on it varies. A spec's abstractions are its Kinds: a Kind is to Definitions what an interface is to classes, the form others follow. So a Package's abstractness A is its share of Definitions of Kind `Kind`, and its distance from the main sequence is D = |A + I − 1|. A stable Package with no Kinds lies in the zone of pain, where change is costly; an unstable Package of Kinds nobody uses lies in the zone of uselessness. In practice: keep shared forms in Kinds at the stable place, keep what many Definitions link to small and settled, and let what changes often be what nothing links to. Link behaviour only from its own module's Interface; when another module must change an operation, let it state its need and link the operation, as an inverted dependency.

**Why.** A change to a Definition puts back on the list everything that links to it, so the cost of a change is its fan-in times how often it happens. Martin's zone of pain is harmful only where things change: a stable, concrete Package that never changes costs nothing.

In code the Package is the unit that is released and depended on. In Markitect the unit of verification is the Definition: a change to `members/register-member` puts back itself and its Interface, however many depend on `members/member` beside it. So read the Package measures as a pointer to where to look, and the Definition's fan-in as the cost.

**Bad.** In the 5× project's change steps, `members/member` (fan-in 90), `billing/charge` and `members/membership` (20 each) changed, and with six other widely linked Definitions they caused 248 of the 468 items the steps put back on the list, items no other change reached. `members`, `billing` and `ledger` all lie in the zone of pain.

**Good.** In Makerspace none of the five Definitions linked by 10 or more changed in eight steps; what changed at Root was the anchor and Rules nothing links. Concepts carry the fan-in (median 6) and Operations do not (at most 2): `members/member` is linked by 24 Definitions, `members/register-member` by its Interface alone.

**Signal.** `vet.unstable_dependency` for the direction of edges. `list --deps` shows each Package's A and D; notes for the zone of pain and the zone of uselessness wait until `vet` knows how often things change. `vet.hub` warns about a Definition whose change alone puts more than a fifth of the spec back on the list. `vet.hotspot` notes, with `--strict`, the cost of change as it happened: a Definition whose fan-in times the revisions of its text the record counts is above a tenth of the spec. On the 5× project's change steps it names `members/member`, `billing/charge`, `members/membership`, `members/members-interface` and `ledger/postings`, and on Makerspace nothing. Both Makerspace specs keep their Kinds at Root, so A is 0 in every other Package and D is 1 − I there.

## Cohesion and coupling

**Rule.** A Package's Definitions link each other more than they link out, links up the tree aside. A Package whose Definitions fall into groups with no link between them may hold two areas; Rules that bind it by scope form groups of their own and are fine. A Package depends on few others, and on each through its Interface.

**Why.** A change stays inside its Package, and a reader of a Package needs few others to understand it.

**Bad.** The 5× project's `bookings` depends on 12 Packages. Each of them can put `bookings` Definitions back on the list, and a reader of `bookings` reads twelve Interfaces.

**Good.** `equipment` and `members` link only inside and up, and depend on no other Package. Of the links a typical Makerspace Package makes, links up aside, 58 % stay inside. A Rule that binds its Package by scope and links nothing, such as `billing/due-date-and-block`, is fine: scope is location.

**Signal.** `vet.coupling` warns above 5 dependencies. Whether a Package holds two areas is the auditor's call: the one Package of either spec that fell into two linked groups, `bookings` at steps w0 and c1, was three Rules binding it by scope, not a second area. The outer part is the exception by design: `protocol` wires other modules and has little cohesion of its own.

## Complexity of a Definition

**Rule.** One idea per Definition. Keep it to a few hundred words and about a dozen statements, a statement being a list item, a table row, a refusal or a state. Each statement is a branch the reviewer verifies, as each condition is a branch of a function. When a Definition grows past that, split it along its ideas.

**Why.** A reviewer verifies a Definition as one item. The more it states, the more a review can miss, and a change to one of its statements puts all the others back on the list too.

**Bad.** The 5× project's `list-order` is one table of the order of every listed result, 67 rows: one item for the reviewer with 67 things to check, and a change to one row puts the whole table back on the list.

**Good.** A typical Makerspace Definition has 88 words and 6 statements; nine in ten stay under 253 words and 12 statements. `billing/account` says in one sentence what an account is and lists its six fields, each with its format.

**Signal.** `vet.definition_size` warns above 1000 words, which no Makerspace Definition reaches (the largest has 430 words, 995 in the 5× project). `vet.statements` notes, shown with `--strict`, a Definition of more than 20 statements.

## Evidence for complexity: a CRAP analogue

**Rule.** The larger a Definition, the more evidence its `ok` should name. A large Definition verified on one pointer is a risk, as a complex function without tests is.

CRAP, the change-risk measure for code, is *comp² × (1 − cov)³ + comp*, with *comp* the cyclomatic complexity and *cov* the share covered by tests; above 30 is a risk. Its analogue for a Definition recorded `ok` or `skip` in `.markitect/review.sum`: *c* is 1 + the cases, conditions and exceptions the reviewer counted for the Definition, or 1 + its statements until a count exists. *e* is the share of those cases the reviewer found its evidence demonstrates, or, until it records that, the evidence pointers divided by *c*, at most 1. A `skip` has *e* 0. So *risk = c² × (1 − e)³ + c*. A Definition of 7 cases (*c* 8) with 2 of them demonstrated scores 64 × 0.36 + 8 ≈ 31; with 4, about 13.

**Why.** It finds what is both hard to verify and thinly verified. As with CRAP, a Definition of more than about 30 statements stays above the line however much evidence it names: the remedy is to split it.

**Bad and good.** No record of a real build exists for Makerspace yet, so there are no measured examples. Without any record, 73 of its 115 Definitions would score above 30, which is why the measure only looks at Definitions recorded `ok` or `skip`.

**Signal.** `vet.risk` waits until the record of real builds can calibrate it ([DEC-102](../concepts/register.md#dec-102-what-vet-measures)). The reviewer already records the counts it rests on, with `markitect mark … --cases N --covered M` ([the `vet` plan](../concepts/vet-plan.md#ai-assisted-measures)).

## Accidental complexity

**Rule.** Write what good documentation of the project would state, and nothing more ([DEC-079](../concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation)). No Definition that only points elsewhere, no Kind used once, no Package that only adds a level, no restating of a fact another Definition holds.

**Why.** Every Definition is an item to verify and every link one to keep true. What the domain does not need is cost without value, for the reviewer and the reader.

**Bad.** Restating: 21 of Makerspace's 38 Operations say "Refusals at check 7", a fact `order-of-checks` already holds and the `Operation` Kind holds again. When the order of checks changes, 23 places have to.

**Good.** Makerspace's two project Kinds both earn their place: `Operation` is the form of 38 Definitions and `Interface` of 8. No Package of either spec is thin.

**Signal.** `definition.unused` is an error of `check` and `vet.thin_package` a warning. Restating and pointer-only Definitions are the auditor's.

## One level of abstraction per Definition

**Rule.** A Definition stays at one level. A Kind says what its Definitions contain and how one is verified; the builder, the reviewer and the auditor read it, and since a Definition's Kind is an input of its verification, a change to the Kind puts its Definitions back on the list. A Concept says what a thing is. A Rule states a constraint. An Operation says what happens. The project anchor and the architecture say what the whole is. Detail goes down into a Definition of its own, and a summary goes up.

**Why.** A change at one level then leaves the others alone. A reader finds each fact at the level where it is decided.

**Bad.** The `Operation` Kind states both the form of an Operation (wire name, arguments, refusals, result, permissions, audit target) and rules of the protocol: an argument that names a record must name one that exists, refusals are checked at check 7. To do so it links six Root Definitions. A Kind's link reaches all its Definitions when every link is followed, so after `values` was split, the Kind carried most of what was left of the widest reaches: 44 of 55 Definitions on Makerspace's step c1.

**Good.** The `Interface` Kind says what an Interface Definition contains and links nothing. Each format of the split `values` is one Concept at one level: its JSON type, its range and what it means.

**Signal.** The auditor's. A measure can point at a Kind that links Definitions other than Kinds, but whether such a link mixes levels needs judgment.

## Everything has its place

**Rule.** Put each Definition in the lowest Package that holds every Definition that uses it, its lowest common parent. Root holds the project anchor, the architecture, the project-wide Rules, the Kinds that several Packages use and the vocabulary those need, and little else. What only the area modules use, even all of them, goes into a Package of its own; a module's own Definitions never sit at Root. What only a Package and those below it may use is marked `internal: true`.

**Why.** Placement is scope: a Rule at Root binds the whole project. It is also visibility. And Root can link nothing below it, so whatever a Root Definition needs must sit at Root too: a module at Root pulls its own dependencies up after it. Links up the tree are free, so coupling to Root shows in no Package's coupling.

**Bad.** Makerspace's Root holds 21 Definitions, among them a module: the refusal codes, audit and authorization, and `list-audit-entries`, an Operation. The Root Kind `Operation` links `audit`, `audit-entry`, `authorization` and `refusal-codes`, so none of them can move down, and `audit-entry` in turn needs the formats at Root.

**Good.** Each area's Concepts sit in its area Package, and the two shared Kinds sit at Root. The formats are used by Root's own Rules (`time`, `money`) and by `audit-entry`, so their lowest common parent is Root: split there, as thirteen Definitions, every link is exact. Split into a `values/` Package instead, they show as an edge from each module, but Root's five users can no longer link them, and a change to a format stops reaching them. At Root they make Root 33 Definitions, and `vet.package_size` warns: the sign that Root holds a module, which only moving the module and the Kind's links out would cure.

**Signal.** `vet.package_size` warns above 25 Definitions, Root included. `vet.placement` notes, shown with `--strict`, a Definition whose users all lie below it in one Package. `vet.hub` counts Root, which the earlier `vet.reach` skipped.

## Say it once, link exactly what you use

**Rule.** State each fact in one Definition. Every Definition that relies on it links it, directly, and links nothing it does not rely on. Link the smallest Definition that holds what you use, and link from the specific to the general, never back: a format does not link the Rule that uses it. When prose names another Definition's concept, link it there. To name an area without depending on all of it, link its Package: a Package link is no input of the record and puts nothing back on the list.

**Values.** Do not specify general knowledge, such as what a date, an amount or an email address is: the agents know it, and a page that restates it is one more thing to verify and keep true. A value rule of the project's own, its format, its range, its unit or when two values are equal, is stated once, as a small Definition of its own, in the only Package that uses it or at the lowest common parent of the Packages that share it. There is no central page of values, and no Kind for them ([DEC-102](../concepts/register.md#dec-102-what-vet-measures)).

**Why.** The links are the build graph. A link the Definition does not need puts it back on the list for changes it does not care about. A missing link lets a change pass: a statement relied on through a chain of links, or because a Rule binds by scope, is not put back on the list when its source changes, since `status` follows one step. Files that change with the code catch some of that, and `/markitect-review --all` is the safety net.

**Bad: one page for everything.** 60 Makerspace Definitions link `values`, thirteen formats, and use a median of 2 of them; in the 5× project 269 link it and use 1. Any change to `values`, even to the clock time only two Definitions use, put 61 Definitions back on the list, 270 in the 5× project.

```markdown
**Fields**, in the formats of [[values]]:

- `memberId`: identifier, naming an existing [[members/member]]
- `blocked`: flag
```

**Good: one Definition per format, linked where it is used.** Split into thirteen Definitions at Root, the lowest common parent of their users, a change to one format lists between 2 and 52 Definitions, a median of 5. Each format is a `std/Concept`; the owner decided against a Kind for values ([`vet` plan](../concepts/vet-plan.md#a-new-std-kind-value)). A format only one Package uses belongs in that Package.

```markdown
**Fields:**

- `memberId`: [[identifier]], naming an existing [[members/member]]
- `blocked`: [[flag]]
```

**Bad: linking back.** `values` said "Arguments are checked against them as `argument-values` states", a link from the general formats to a Rule that uses them. With `argument-values` linking `computed-times`, that linking the limits, and the limits and `money` linking `values`, six Root Definitions formed a cycle, and every change that reached any of them reached all 60 users of `values` when every link is followed. Dropping that one link halves the every-link reach of Makerspace's four widest change steps, from 85–88 % of the spec to 45–60 %.

**Bad: naming without linking.** `billing/issue-credit` issues a credit and never links `billing/credit`; `bookings/metering` applies "the member's tier discount" and never links `members/tier`. A change to the credit or the tier puts neither back on the list. A text check found roughly 130 to 180 such missing links among Makerspace's 115 Definitions.

**Bad: relying through a chain.** 21 Operations say "Refusals at check 7", but only the `Operation` Kind links `order-of-checks`. A change to the order of checks puts the Kind back on the list, not those 21 Operations. Linking `order-of-checks` from each Operation makes it a true hub, which is right: then keep it small and stable.

**A milder case.** 17 of the 24 Definitions that link `members/member` do so only to say that an identifier names an existing member; a new field of the member puts them all back on the list. Split such a part out only when it changes on its own.

**Signal.** `vet.unlinked_mention` notes a Definition's title or name in another Definition's prose with no link to it; on the Makerspace specs half to three-quarters of what it finds are real missing links, and the rest ordinary words and other meanings of a name. Plain `vet` counts them in its last line, and `/markitect-audit` has the auditor triage them into missing links and ordinary words, so that the architect adds the links. `vet.hub` points at a Definition many link to. Whether they use all of it, or only part, and whether a Definition relies on something it neither links nor names, are the auditor's questions.

## Measures at a glance

| Measure | Formula | Makerspace c8 | 5× project c8 | Signal |
|---|---|---|---|---|
| Fan-in | Definitions that link to it or are of it as Kind | median 1, p90 7, max 60 | 1, 7, 269 | `vet.hub` |
| One-step reach | 1 + fan-in: what `status` lists after a change to it alone | 2, 8, 61 | 2, 8, 270 | `vet.hub` |
| Every-link reach | Everything that reaches it along the links | 8, 52, 103 | 8, 85, 402 | Read, not warned |
| Fan-out | Definitions it links to | 2, 8, 16 | 4, 9, 24 | Read, not warned |
| Efferent coupling Ce | Packages a Package depends on | at most 8 | at most 12 | `vet.coupling` |
| Instability I | Ce / (Ca + Ce) | 0 to 1 | 0 to 1 | `vet.unstable_dependency` |
| Abstractness A, distance D | Share of Definitions of Kind `Kind`; \|A + I − 1\| | A 0.10 at Root, 0 elsewhere; 4 Packages in the zone of pain | 0.10, 0; 9 in the zone of pain | In `list --deps`; zone notes wait |
| Volatility × fan-in | Revisions of a Definition's text times its fan-in | at most 14; no hub changed | 5 above 42, `members/member` 90; 248 of 468 items from 9 hubs | `vet.hotspot`, from the record |
| Cohesion | Groups of a Package joined by its own links | none split at c8; `bookings` at w0 and c1, by scope Rules | none split | Read, not warned |
| Unlinked mentions | A Definition's name in another's prose, with no link | 233 pairs, 52–65 % real in a sample | 863, 57–70 % | `vet.unlinked_mention` |
| Size | Words in a Definition's body | median 88, max 430 | 99, 995 | `vet.definition_size` |
| Statements | List items and table rows | median 6, max 25 | 7, 67 | `vet.statements` |
| Risk | c² × (1 − e)³ + c | no record yet | no record yet | `vet.risk` waits for data |

[The model language guide](model-language.md#vet-warnings-and-notes) lists `vet`'s warnings and notes with their thresholds, and the [`vet` plan](../concepts/vet-plan.md) records why each is a warning, a note, or not built yet.
