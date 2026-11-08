# Plan: relationship types in the spec

**Status:** decided on 8 November 2026 by the coordinating session, under the owner's delegation of decisions until their test on 9 November 2026 ([DEC-104](register.md#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them)), as this page recommended; built for 0.25.0. This page is the plan's record. It was written the same day as a proposal answering the owner's request: relationship types that open up for the spec what code's relation types open up, grounded in proven concepts, with the core language kept simple, flexible and clean. A first draft proposed one link that only names its target; the owner's rule below ruled that out, a second draft proposed two link keywords, and an independent review found the same gain without new syntax, which this page then recommended. It also answers the question the Ideas chat left open on 9 October 2026, "Which relationships materially improve coverage?" ([IDEA-20261009-002](https://github.com/Glacius-Labs/Markitect/blob/d85e0fa1302ec95d1c82c6d2c088bf2ff34b6557/docs/concepts/sources/ideas-register-20261009.md#idea-20261009-002--typed-relationships-make-ownership-and-impact-visible)).

It keeps to what is settled:

- **The spec behaves like code.** As the coordinating session relayed the owner's rule of 8 November 2026: the spec is Markdown and is, or generates, the documentation, but it behaves like code, and code has no "see also". So every link is a real relationship that matters for meaning and, in some direction, for change, and navigation aids belong in documentation rendered from the spec. The owner favours typing of every kind under that rule. The rule is paraphrased here, as it was relayed.
- **Core is one Definition,** `Kind`, with no `implements` and no typed fields, and a link names its target by path ([DEC-080](register.md#dec-080-the-model-is-typed-documentation)).
- **The tool treats no Kind specially by its name;** there are no layers and no Kind that refines another (the owner's rules for the [`vet` plan](https://github.com/Glacius-Labs/Markitect/blob/d85e0fa1302ec95d1c82c6d2c088bf2ff34b6557/docs/concepts/vet-plan.md)).
- **A Definition's inputs** are its text, its Kind, what it links to, its registry rows and its evidence ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)).

**Basis:** the [measurements of 8 November 2026](../validation/spec-relations-20261108.md) on the archived Makerspace spec and the 5× project, checked by an independent reviewer. [The model language guide](../guides/model-language.md) owns the language as it is.

## In brief

- **Markitect already has the relations the established sets name,** most of them outside the link: a Definition's Kind, the Package tree with its scope and visibility, the plain link, the Package link, the registry and the evidence.
- **Two relations of code are missing from the specs as written,** and both end up as plain links to the whole target: an identifier that names a record ("`memberId`: identifier, naming an existing [[members/member]]") and a module's list of what it publishes (an Interface's operations and records). They are 110 of Makerspace's 360 links. A change to the record or to a listed part puts each of them back on the list, though none relies on what changed.
- **Code expresses neither with a relation keyword.** An identity has a type of its own, as domain-driven design's typed identity (`MemberId`) does; what a module publishes is its visibility, as Go's exported names are. The spec can do the same today: an identity Definition per record that others name, linked by every holder; no list of parts, since `internal: true` already says what a module publishes; and a module's "Uses" list linking Packages.
- **Measured:** Makerspace's eight change steps put 113 items on the list instead of 133, and the 5× project's 354 instead of 468; its step c4 falls from 138 items to 42. No needed item is lost. That is what two new link keywords would give (113 and 348), and every link stays a plain link that relies on its target.
- **Decided: no change to Core.** The rule that a link is never decorative and the three patterns go into the guides and the `markitect-design` skill, and `doc PACKAGE` shows a Package's public Definitions, as `go doc` shows exported names. Two link keywords, `id` and `part`, are designed and costed [below](#the-alternative-two-link-keywords) and kept as the considered alternative.

## What Markitect has today

| Relation | Written as | Used today for | Input of the record | In the Package graph |
|---|---|---|---|---|
| **instance-of** | `kind:` in the front matter | The form a Definition follows and how it is verified; abstractness in `list --deps` ([DEC-102](register.md#dec-102-what-vet-measures)) | A change to the Kind puts its Definitions back on the list | An edge (using a Kind) |
| **containment** | The folder tree | Scope (a Rule binds its subtree), visibility (`internal: true`), the Dependency Rule (no link down the tree) | None: a changed Rule above a Package is one item, with what links to it, not one per Package below | The implicit edge to the parent |
| **relies on** | Every `[[path]]` to a model Definition | The build graph: inputs, coupling, cycles, `vet.hub`, `doc`'s "linked from" | A change to the target's text puts the source back on the list, one step | An edge, if sideways |
| **module dependency, or a part of the organization** | `[[members]]`, a link to a Package | Sideways, a dependency on another module; down the tree, naming a part of the parent's organization | None; the Package must exist | An edge sideways; none down the tree |
| **uses a Library Kind** | `[[std/Rule]]` in a body | Naming a standard form | Through the Library's digest in the lock | None |
| **realizes** | A row of a registry page: file, Definitions, why; the page's path patterns say which Package owns a file | Accounting for every file, `definition.unused`, stale realization | A change to the file, or a row that gains or loses the Definition, puts it back on the list | None |
| **verifies** | The evidence of a verdict in `.markitect/review.sum` | What counts as done; revisions, the basis of `vet.hotspot` | A change to an evidence file puts the Definition back on the list | None |

## What the established sets offer

| Family | UML | ArchiMate | SysML, ReqIF | DDD | C4 | In Markitect |
|---|---|---|---|---|---|---|
| Type and instance | The classifier of an instance | — | — | — | — | The Kind |
| Specialization | Generalization | Specialization | — | — | — | None, by the owner's rule: no Kind refines another |
| Whole and part | Composition, aggregation | Composition, aggregation | Containment | Aggregate | Containers, components | The Package tree |
| Structural link between things | Association | Association | — | — | — | The plain link |
| Realization | Realization | Realization, assignment | «satisfy» | — | — | The registry |
| Verification | — | — | «verify» | — | — | The evidence |
| Dependency | Dependency «use», «call» | Serving, access, influence | «deriveReqt», «refine» | Upstream and downstream | "Uses", with a description | The plain link; a Package link for a module |
| Reference by identity | — | — | — | Typed identity; "reference other aggregates by identity" | — | Missing as written; a typed identity, [below](#typed-identity) |
| A module's published surface | Visibility | — | — | Open host service, published language | — | Visibility (`internal: true`); written today as lists, [below](#the-surface-is-visibility) |
| Traceability without meaning | «trace» | — | «trace» | — | — | None, by the owner's rule: no "see also" |
| Behaviour over time | — | Triggering, flow | — | — | — | Nothing, and no measured need |

Four readings matter for Markitect:

- **A build graph distinguishes how much of a target a source needs.** John Lakos (*Large-Scale C++ Software Design*, 1996) calls it using a type "in name only", through a forward declaration, against using it in the interface or the implementation; only the second forces a rebuild. Domain-driven design gives the domain's form of it: an entity refers to another by its identity (Vaughn Vernon, *Implementing Domain-Driven Design*, 2013), and that identity is best a value type of its own, `MemberId` rather than a bare string. A typed identity is something a holder truly relies on, its form and what it names, and it changes far less often than the record.
- **A module's surface is its visibility.** Go has no export list: a package publishes its exported names, and its documentation shows them. Markitect chose the same with `internal: true`. Haskell's export lists and OCaml's module signatures are the other style, a list beside the definitions; a spec that writes such a list beside its visibility says the same thing twice.
- **SysML's chain of traceability is already complete,** under other names: a Definition relies on the general Definitions it derives from or refines (the plain link, up the tree), files realize it (the registry, «satisfy»), and evidence verifies it (the record, «verify»). «trace», the relation that carries no meaning for change, is the one the owner's rule keeps out.
- **The richer vocabularies serve diagrams and exchange, not a build.** ArchiMate's eleven relationships and their derivation rules, UML's association ends, and ReqIF's relation types, which each exchange defines for itself, let tools draw and query models. C4 shows that one relation, "uses", with its meaning in a line of prose, is enough for architecture diagrams. DDD's context map relates bounded contexts, which are Packages here: upstream and downstream is the edge's direction; an open host service with its published language is what Makerspace's Interfaces and its outer `protocol` module offer; a shared kernel sits in a common parent; and a partnership comes closest to a mutual dependency, which `check` refuses as a cycle.

## Every link is a relation

For the model language guide and the guide to a good spec:

> A link is never decorative. Each link states a relation that matters for what the source means and, in some direction, for change. A plain link says that the source relies on what the target states, so a change to the target puts the source back on the list. A link to a Package says that the source's module depends on that module, or, down the tree, says something about a part of the parent's organization. The Kind, the tree and its visibility, the registry and the evidence state the other relations. A link that would only help a reader find something does not belong in the spec: navigation belongs in documentation rendered from it.

So a pointer such as "the modules are described in the [[architecture]]" is either a sentence that relies on the target, and stays a plain link, or it goes. The same holds for a link from the general back to what uses it, such as `values` pointing to the Rule that checks arguments against it, which the [guide](../guides/good-spec.md#say-it-once-link-exactly-what-you-use) already says to drop. Whether a downward Package link that only lists a parent's children is such a pointer, since the tree already says it, is left open ([below](#decided)).

## The decision: write identity and surface as code writes them

### Typed identity

A record that other Definitions name by an identifier gets an identity Definition of its own, in its Package, next to the record. The identity states its form and what it names:

```markdown
---
kind: std/Concept
---
# Member identifier

The identifier of a [[members/member]]: an [[identifier]] issued when the member is registered, which never changes. A user whose role is not `member`, such as a steward, has an identifier but names no member.
```

The record relies on its identity:

```markdown
# Member

A user with role `member` and a membership. A record is identified by its [[members/member-id|member identifier]].
```

and each holder relies on the identity, which is what it uses, instead of on the record:

```markdown
- `memberId`: a [[members/member-id|member identifier]]
- `machineId`: a [[equipment/machine-id|machine identifier]]
```

The record and its identity link each other, a cycle inside one Package, which the language allows. Both links are real: the record relies on the form of its identity, and the identity on what it names. A link only in words would be a name without a link, which `vet.unlinked_mention` reports and the guide calls a missing link.

- **What the record does:** a new field of the member puts back the record, its identity and what relies on the record's fields, not the 17 Makerspace Definitions that only name a member: the list follows one step. A change to what identifies a member, its form or who counts as one, reaches every holder, which is right: they rely on it.
- **Where the rules of an identity live:** in one place. Today "the identifier of a staff, admin or steward user does not name a member" sits in check 5 of `order-of-checks`, far from the member.
- **Grounded in** DDD's typed identity and Lakos's use in name only. It is [DEC-102](register.md#dec-102-what-vet-measures)'s rule for values applied to an identity: stated once, as a small Definition of its own, where its users meet. There is no `std/Value`, so an identity is a `std/Concept` or of a project Kind.
- **A holder that relies on more than the identity** links that as well. One that relies on the record's fields or status links the record, as borrowing a book links the member to check that the member is active. One that names "the invoice's member" relies on the invoice and links the invoice.

### The surface is visibility

An Interface states what other modules may ask of its module, its capabilities and the codes it adds, and does not list its operations and records: what is public is what it publishes.

```markdown
# Bookings interface

What other modules may use of bookings: its public operations and records. Other modules also ask whether an active reservation of a machine overlaps an interval, for the completed sessions of a machine and for the lockouts of a machine.

- **Uses:** [[members]], [[training]], [[equipment]], [[billing]], [[safety]].

The refusal codes of this module: …
```

- **Measured:** on all nine Makerspace steps, each Interface's lists of operations and records are exactly its Package's public Definitions that are not Rules (72 of 72); in the 5× project on 399 of 417, where two Packages leave out public status Concepts. The lists say twice what visibility says once.
- **What the record does:** a new refusal of an operation no longer reaches its Interface; an operation that joins or leaves the surface changes its own `internal:` line and is on the list itself.
- **What the tool adds:** `doc PACKAGE` lists the Package's public Definitions apart from its internal ones, and its JSON says of each Definition whether it is internal, so that the surface is shown where it is stated, as `go doc` shows exported names. This is the one change to the tool, and no change to the language: every listed Definition carries `internal` in the JSON of `list` and `doc`.

### A module's "Uses" list links Packages

`**Uses:** [[members]], [[billing]]` states that the module depends on those modules, which is what the list says; it makes the edge and puts nothing on the list when the members Interface changes. The language allows it today. Where the Interface's prose relies on what the used Interface states, as "bookings, which access uses, declares its need for that answer" does in the 5× project, that sentence keeps a plain link. Makerspace's own `Interface` Kind says that an Interface "links the interfaces of the modules it uses"; a spec that adopts the pattern changes the Kind's text with it.

### What it gains

The [record](../validation/spec-relations-20261108.md) has the method and every figure.

| Over the eight change steps | Makerspace | 5× project (patterns only) |
|---|---|---|
| Items on the list today | 133 (84 changed, 49 put back by links) | 468 (111 changed, 357 by links) |
| "Uses" lists linking Packages only | 129 | 420 |
| All three patterns | 113 (29 by links) | 354 (243 by links) |
| For comparison, the keywords `id` and `part` | 113 | 348 |
| Dropped items whose source needed the change | none, of 36 pairs | none, of 126 pairs |

The 5× project's six items more than the keywords are three identity Definitions written when new modules first named their records, and three identities put back on the list when their record changed. The largest single drop is the 5× project's step c4, which changed `members/member` and the members Interface: from 138 items to 42. 67 Definitions there only name a member, and 29 Interfaces only list the members Interface among the modules they use.

**What else it opens up:**

- **Hubs that are hubs.** `members/member` falls from fan-in 24 to 6 on Makerspace and from 90 to at most 6 on the 5× project, and is no hub at `vet.hub`'s threshold. Its identity takes the fan-in, as `values/identifier` did after the split: a true hub, small and stable, as the guide asks.
- **Coupling by identity, visible.** `list --deps` already names the links behind each edge. In Makerspace `safety` and `training` depend on `equipment` through the identities of a machine and a family, and through their "Uses" lists: what changes in `equipment` reaches them only when what identifies a machine or a family changes.
- **Cohesion by reliance.** Without the lists, three Makerspace Packages fall into groups that read as sub-areas: the catalog and maintenance in `equipment`, membership and the users in `members`, invoicing and the account in `billing`. Today each Interface links every part of its module and glues them into one group.
- **Traceability.** From a Rule or a Decision along the links to the Definitions that rely on it, their files and their evidence; from a file to the Definitions it realizes and what they rely on. With no pointers in the spec, each step of such a chain is a reliance.
- **Every-link reach for the author** falls on Makerspace's four widest steps from 85–88 % of the spec to 79–82 %, and to 43–60 % once the link back from `values` to its user goes.

### What it costs

- **Files:** 9 identity Definitions in Makerspace, 42 in the 5× project, each a few lines, with their registry rows; and one more whenever a record is first named by another module.
- **Adopting it once:** rewriting a spec this way changes the text of many Definitions, so the next `status` lists most of the spec once: on Makerspace c8, 73 of 115 Definitions and the 9 new ones; on the 5× project, 358 of 419 and 42 new. That is more than the eight steps save. A spec adopts it best from the start, or with a change that touches its Interfaces anyway. The keywords cost the same once (76 of 115; 375 of 419).
- **Reading:** an Interface no longer lists its parts; `doc PACKAGE` and the rendered documentation do.

### What was built

No language change, for 0.25.0:

1. **The [model language guide](../guides/model-language.md#links-scope-and-visibility)** states the rule [above](#every-link-is-a-relation) under links, and that a Package's public Definitions are what it publishes.
2. **The [guide to a good spec](../guides/good-spec.md#every-link-is-a-relation)** states the three patterns, typed identity, the surface as visibility and "Uses" lists as Package links, with their measured effect, and when a holder also links the record.
3. **The `markitect-design` skill** states the rule in its design rules, its reference explains the patterns, and its example names a book's holder by the member's identity, `members/member-id`.
4. **`doc PACKAGE`** lists a Package's public Definitions apart from its internal ones, and every Definition that `list` and `doc` show carries `internal` in their JSON; tests cover the text and the JSON of `doc` and `list`.

## The alternative: two link keywords

Two keywords give the same gain with the relations in the link syntax instead of identity Definitions and visibility. They are kept as the considered alternative, concrete enough to build from. They come back only on new evidence: a real spec whose surface must differ from its visibility, or whose identity Definitions, measured over its history, cost more than the re-verification they save.

| Relation | Means | Written as | Input of the source | Input of the target |
|---|---|---|---|---|
| `id` | The source names a record of the target by its identity | `[[id::path]]`, `[[id::path\|label]]` | Which Kind the target is, not its text | — |
| `part` | The target is part of what the source publishes | `[[part::path]]`, `[[part::path\|label]]` | Which Kind the target is | Being a part of the source: a source that gains or loses the part puts it back on the list, as a registry row that gains or loses a Definition does |

**Why each is a relation under the owner's rule.** `id` is Lakos's use in name only: the source depends on the target's declaration, its path, its visibility and its Kind, and not on what it states; `check` refuses a removed or renamed target, and a change of the target's Kind puts the source back on the list. Most of its force comes from `check`, which every link has. The same argument is what keeps a downward Package link in the language. `part` changes the record in the other direction, from the whole to the part.

**For the build:** a link may start with `id::` or `part::`; any other word before `::`, either prefix on a link to a Package, a Library Kind or a Kind, and either prefix in a registry row's Realizes cell are errors (`link.relation`); today `[[id::…]]` fails as `link.syntax`. Both stay edges, checked for visibility, direction and cycles, and count as links for `definition.unused`; keeping them edges is a choice on principle, so that the architecture sees every named dependency, not something the measurements force, since every edge they carry is also carried by a "Uses" list. The inputs gain `id:<target>` and `part:<target>`, each with the target's Kind ID, in place of `link:<target>`, and `part-of:<whole>` on the part with a constant value; a pair joined by a plain link keeps `link:`, and a pair joined by both `id` and `part` takes `part:`. `status` prints the new keys as it prints the others, such as `part-of:bookings/bookings-interface (new)`. The existing `via` field of `list` and `doc` gains `id` and `part` beside `link` and `kind`; a Definition's `links` in `list --json` stays a list of IDs, and `format_version` stays 1. Measures that count what a change puts back on the list, `vet.hub` among them, count plain links and Kinds. The canonical table of inputs in [the record's README](../../internal/run/README.md), the model language guide and DEC-099's list of inputs change with it.

**Four forms of writing them** were compared on the reservation record and the bookings Interface:

- **(a) Derived from the Kinds at both ends, no syntax.** "`memberId`: identifier, naming an existing [[members/member]]" and "`tier`: a [[members/tier]]" both link a Concept from a Concept, the first by identity and the second for its values. The best rule from the pair of Kinds is right for 308 of Makerspace's 360 written links and misses exactly the 52 identity references; it would also make the tool know Kinds by name, which DEC-080 and the rules for the `vet` plan exclude.
- **(b) Inline, as Semantic MediaWiki writes typed links:** ``- `memberId`: identifier, naming an existing [[id::members/member|member]]`` and ``**Operations:** [[part::bookings/create-reservation]], …``. The relation stands beside the sentence that justifies it, and existing specs are unchanged. Markdown tools that follow `[[path]]`, such as Obsidian, show the label of such a link but do not follow it, since they read `id::members/member` as a page name; DEC-080 chose `[[path]]` so that links work in such tools. The best of the four.
- **(c) A heading convention:** links under **Operations** and **Records** are parts, and identities move to a list of their own (``**Identities:** [[members/member]], [[equipment/machine]]``), away from the field they explain, since an identity reference shares its label, **Fields** or **Arguments**, with field lines whose links the field relies on. It catches all 20 items Makerspace saves but only 53 of the 5× project's 120; prose labels would become grammar, and editing a heading would change what is verified.
- **(d) Front matter lists:** `id: [members/member, equipment/machine]`. Against DEC-080 twice: the front matter holds only `kind` and `internal`, and there are no imports. The list would silently override the prose links to the same targets, and when the text later comes to rely on a listed target, nothing shows it.

**Why not chosen:**

- It changes Core for a gain the language reaches without it.
- The vocabulary would be fixed in Core, as Go's keywords are, because the tool acts on each word; but code has no keyword for either relation, and Go shows the surface by visibility.
- `id` gives an identity's rules no home: the steward rule stays in check 5, far from the member, and a change to what identifies a member reaches no holder.
- `part` writes beside visibility what visibility says, which the guide's "say it once" rules out.
- Markdown tools stop following those links.

## Considered and not chosen

| Relation | Grounded in | Why not |
|---|---|---|
| A link for reading only: "see also", a mention | SysML and UML «trace»; doc links in Go comments | Excluded by the owner's rule: navigation belongs in rendered documentation. Measured for the record: pointers and code terms would have taken 6 more items off Makerspace's list |
| refines, derives | SysML «refine», «deriveReqt» | The plain link from the specific to the general, placed below it in the tree, says it already, with the same change semantics |
| constrains, gates | ArchiMate influence | A Rule that adds a gate to another module's operation links it, and propagates as a plain link should |
| Types that only add meaning: applies a rule, typed by, reads, writes | UML stereotypes, ArchiMate access | They would carry a plain link's change semantics: "where is `money` applied" and a matrix of which module writes which record would become queries. They change nothing the record does, so they wait until an analysis needs them |
| specializes | UML generalization | Settled by the owner on 7 November 2026: no Kind refines another |
| Kind links inherited by the Kind's Definitions | A subclass depends on what its base class uses | Measured: 64 more items on two steps, for checks the architecture realizes once, in the protocol module |
| Section-precise inputs, `[[members/member#Identity]]` | Rebuilding only on interface changes (C++ headers, ABI jars) | Another axis: hash only the linked section. A typed identity does the same with a Definition, which a heading cannot silently undo |
| Project-defined relation names | ReqIF, Semantic MediaWiki | The tool cannot act on them, and prose already carries them |

## Risks

- **A holder may rely on more than the identity** and link only the identity. The guide's rule covers the measured cases, a holder that relies on the record's status and one that names another record's field; the reviewer and the auditor read the links with their sentences, and `/markitect-review --all` stays the safety net. The independent reviewer's blind reading agreed with this session's on 49 of 50 link lines; that supports how existing links were read, not how authors will write.
- **An Interface's capabilities still restate parts of its operations,** such as which codes its module adds; the Interface keeps those, and a change to them changes the Interface's text.
- **The identity Definitions never changed** in the steps, so their own cost on a list is unmeasured.
- **One Kind's text changes:** Makerspace's `Interface` Kind says that an Interface links the interfaces of the modules it uses; a spec that adopts the patterns rewrites its Kinds as well.

## Decided

[DEC-104](register.md#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them) records the decision:

1. **No change to Core.** Identity and a module's surface are written as code writes them: an identity Definition per record that others name, the surface as visibility with no list of parts, and "Uses" lists linking Packages.
2. **A link is never decorative,** in the model language guide, the guide to a good spec and the `markitect-design` skill, with the three patterns.
3. **`doc PACKAGE` shows the surface:** public and internal Definitions apart, in its text and JSON; the only tool change.
4. **Not the keywords `id` and `part`,** kept above as the alternative, with the condition that brings them back.
5. **Not now:** types that only add meaning, refines, constrains, inherited Kind links and section-precise inputs.

**Open for the owner:** a downward Package link that only lists a parent's children is a pointer under the rule. Whether the language refuses such links, or a guide advises against them, is the owner's call; no guide does so yet, and the skill's example names its Packages by link.
