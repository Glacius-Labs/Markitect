# Standard Libraries

**Redesign decided, not built:** [DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation) replaces this with Markdown Definitions without typed fields, a single Core `Kind` and Libraries of plain Kind Definitions. This page describes Main today.

**Status:** outline v0.3, accepted by the owner on 26 October 2026; v0.3.1 adds clarifications found while drafting `practice` and `architecture`, marked "v0.3.1" below, accepted by the owner on 27 October 2026. The owner decided the open Library questions on 27 and 28 October 2026 ([source](../concepts/sources/library-questions-20261028.md)): `api` 2.0.0 and `architecture` 1.1.0. Canonical model track, workstream W4, built on [Kind interfaces](kind-interfaces.md) (DEC-075 as amended, Core 1.2.0), the [modeling guide](modeling-guide.md) and charter decision D4. All five Libraries are drafted under `libraries/`.

## Principles

Oriented on what Go and C# ship:
- **As explicit as good documentation, no more** ([DEC-079](../concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation)). A Library supports what good documentation of a project would state that explicitly: domain concepts, operations and their contracts, rules, architecture. Loops, maps, internal data structures and algorithms are left to the realizing agents.
- **A small set of stable Libraries, cut by technical concern, not by project.** Anything domain-specific belongs in the project.
- **Libraries add Kinds and Properties only.** They get no compiler semantics (language contract §11).
- **No inheritance.** A project Kind reuses Library Properties in its own `properties`, and declares `implements` where it must be accepted in place of a Library interface. Every Kind shows its whole contract.
- **Each Library is its own model,** with its own version and lock entry.
- **Package names are namespaced** as `markitect/<library>`, like `markitect/core`. A Package belongs to exactly one model (language contract §8.1).
- **Package references are strings.** Packages are not Definitions. Rules about Packages name them as path strings, as modeled scope requires. `"."` means the Root. The W5 tool checks every named Package before a run, because a mistyped name in a scope field stops context selection for every Manager.
- **No Library ID may equal a Core ID, ignoring case:** `Kind`, `Property`, `name`, `valueType`, `minCount`, `maxCount`, `values`, `target`, `properties` or `implements`. Short references are matched across the own Package and all imports ignoring case. That is why the Kind for simple values is `Scalar`, not `ValueType`.
- **No Kind and Property share an ID within a Package, ignoring case.** Where the field name should read naturally, the Property sets it with `spec.name`.

## The Libraries

Direct dependencies (only direct dependencies open imports, language contract §8):

| Library | Depends on |
|---|---|
| `markitect-values` | markitect-core |
| `markitect-practice` | markitect-core |
| `markitect-data` | markitect-values |
| `markitect-api` | markitect-values |
| `markitect-architecture` | markitect-values, markitect-api, markitect-practice |

### `markitect-values`, Package `markitect/values`

The types of fields, and the shared field-shape Properties.

| Definition | Shape | Notes |
|---|---|---|
| **FieldType** (interface Kind) | no Properties | "Any field type": a reference to FieldType accepts every Kind that implements it (Scalar, Enumeration, Data.Entity, project Kinds). It has no items of its own |
| **Scalar** (Kind) implements FieldType | `base` (enum: string, integer, number, boolean); `lowest`, `highest` (number, optional); `measuredIn` (reference to Unit, optional) | Format rules beyond the range stay in the body |
| **Enumeration** (Kind) implements FieldType | `members` (object list: `memberValue` (field name `value`) string, `meaning` string optional) | Wire values are exact strings |
| **Unit** (Kind) | `symbol` (string) | For example cents or minutes |
| **Field-shape Properties** `fieldName` (field name `name`), `ofType` (field name `fieldType`) → FieldType, `list`, `nullable`, `refersTo` → FieldType (optional) | Shared inner Properties | Each use defines its own object Property from them, because a Property fixes both its field name and its cardinality: `Data.fields`, `Api.arguments`, `Api.results`. `fieldType` embeds (an Entity type embeds its fields); `refersTo` says the value is the identity of an existing item of that type |

### `markitect-data`, Package `markitect/data`

| Definition | Shape | Notes |
|---|---|---|
| **Entity** implements Values.FieldType | `fields` (object list of the field-shape Properties), `key` (string, the identifying field name; optional, absent for records without identity of their own) | Fields are object values on the Entity, not separate Definitions. Drafting note: `key` is optional because records such as an invoice line have no identity of their own |
| **Invariant** | `entities` (references to Entity) | The rule is the body |

### `markitect-api`, Package `markitect/api`

| Definition | Shape | Notes |
|---|---|---|
| **Operation** (interface Kind with fields) | `wireName` (string), `mutation` (boolean), `arguments` (object list of the field-shape Properties; constraints in the body), `refusals` (object list without order: `refusal` → RefusalCode, `when` string; since 2.0.0 no "first one wins", and where precedence matters a Rule or the operation's body states it), `results` (object list of the field-shape Properties), `permissions` (object list: `forRole` (field name `role`) → Role, `scope` enum own/all) | Usable directly, or implemented by project Kinds that add fields, for example an audit target. A role without a permission entry has none |
| **RefusalCode** | `code` (string, the wire value) | One Definition per code; shared codes live in a shared Package. Another Package uses a code only if an Interface exposes it, as an exported error in Go |
| **Role** | none | The body says who holds it |

Event is deferred until a pilot needs it. A transport binding (HTTP route and status, CLI command, line protocol) is commonly needed but not in this outline; for now it stays in Rule bodies.

### `markitect-practice`, Package `markitect/practice`

| Definition | Shape | Notes |
|---|---|---|
| **Project** | `language` (string); `moduleRoot` (string, repository-relative); `permittedDependencies` (strings; absent or empty means nothing beyond the standard library); `buildCommand`, `testCommand` (strings, run in `moduleRoot`; v0.3.1: optional) | The project Package's anchor (DEC-061). The required checks in `execution.yaml` are authoritative; these fields describe them for readers and agents |
| **Rule** (interface Kind with a field) | `appliesTo` (Package strings, optional; v0.3.1: a named Package includes the Packages below it, as modeled scope delivers it; absent or empty, the Rule governs only its own Package) | A binding statement. With `appliesTo` and one `scope` selector on Rule in `execution.yaml`, it reaches every Manager in scope, including items of every Kind that implements Rule |
| **TestObligation**, **DocumentationObligation**, **ConventionObligation** implement Rule | `appliesTo` | Way-of-working duties. A project adds its own, for example a SecurityObligation, by implementing Rule |

### `markitect-architecture`, Package `markitect/architecture`

The Package tree is the architecture (modeling guide §2). These Kinds add only what the tree cannot show. They all implement `Practice.Rule` and declare `appliesTo`, so modeled scope brings them to the Managers they govern.

| Definition | Shape | Notes |
|---|---|---|
| **Interface** implements Practice.Rule | `appliesTo`, `operations` (references to Api.Operation), `types` (references to Values.FieldType), `refusalCodes` (references to Api.RefusalCode; since 1.1.0) | A Package's public surface; the Package is where the Interface lives. Other Packages use only what an Interface exposes (a W5 tool rule). v0.3.1: the Package's own subtree is not bound by it; `appliesTo` names the bound Packages, usually `.` |
| **DependencyRule** implements Practice.Rule | `appliesTo`, `mayDependOn` (Package strings) | An allowlist for the Packages in `appliesTo`, checked against the model's Package graph (D3) and against the code's imports by project checks. v0.3.1: only dependencies between the project's own Packages count, subtrees included on both sides; Libraries, Core and third-party code fall under `Project.permittedDependencies`; absent or empty `mayDependOn` means none |
| **Placement** implements Practice.Rule | `appliesTo`, `concern` (reference to Practice.Rule), `realizedIn` (Package string) | Where a cross-cutting rule is realized, for example authorization in `pipeline`. v0.3.1: `appliesTo` usually equals the concern's. An Invariant is not placed: the Package that owns its Entities enforces it, as code that owns data guards its invariants |

## What this replaces

`software-basics` 0.3.0 stays frozen for the existing fixtures:

| `software-basics` | New |
|---|---|
| Concept | Data.Entity, or a project Kind implementing Values.FieldType |
| Rule | Practice.Rule |
| UseCase | Api.Operation |

## Versioning and compatibility

- Each Library started at 1.0.0 and depends on `markitect-core` 1.2.0. Current: `values`, `practice` and `data` 1.0.0, `api` 2.0.0, `architecture` 1.1.0. Projects pin exact versions; versions are exact labels (language contract §8.1).
- **Authoring policy:** a change to the Properties of an interface Kind breaks its implementers, as adding a method to a Go interface does, so it needs a new major version. New Kinds, new enum values and new optional fields inside object Properties are minor versions; a minor version can still require a project to qualify a short reference (below).
- **Known limit:** an import makes every ID of the imported Package visible unqualified (language contract §8). A new Library Definition can therefore make a project's existing short reference ambiguous; the compiler reports it, and the project qualifies the reference. Imports visible only through their alias, as in Go, would remove this. That is a possible Core change, recorded as a W1 finding, not part of W4.

## Location and use

- `libraries/<library>/model.yaml` and `libraries/<library>/model/markitect/<library>/…` (D4); since R3a ([DEC-076](../concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)) this repository keeps no registry, so the Libraries have no owner shard.
- A project vendors the Libraries it uses and passes each one with `--dependency`. The W5 tool writes the lock entries.
- A Library ships Kinds, not scoped Rule items, because modeled scope selects only the project's own Definitions. The W5 skill adds the `execution.yaml` selector.

## Answered and open questions

| Question | Answer |
|---|---|
| Package paths as strings? | Yes. Modeled scope requires strings anyway; referenceable Packages would be a Core change with no gain now |
| "Any type"? | An empty interface Kind, `FieldType`, implemented by Scalar, Enumeration and Entity |
| How strict is compatibility? | Interface Properties change only in a major version |
| Event now? | No, deferred |
| Where does Project live? | In `practice`, the base Library that `architecture` builds on |
| Alias-only import visibility as a Core change? | Not now. Ambiguities are reported and fixed by qualification until a real model shows harm |
| Must a refusal list be ordered? | No. A refusal is a possible problem, and when it applies belongs to its own entry. Since `api` 2.0.0 the list has no order; precedence, where it matters, is a Rule, as Makerspace's `order-of-checks` states it. This replaced an "unordered" flag before it was built |
| Refusal codes in an Interface? | Yes, as in code, where a package's exported errors are part of its public API. `architecture` 1.1.0 adds `refusalCodes` |
| Placing an invariant? | No. Placement is for cross-cutting concerns; an Invariant lives in, and is enforced by, the Package that owns its Entities |
| Maps, unions, defaults, typed formats? | No ([DEC-079](../concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation)). Good documentation does not state them; uniqueness beyond `key` and constraints across fields stay prose in Rules or Invariants |
| Use cases, REST routes, CLI commands? | Part of the redesign: which Library areas Markitect ships is open ([topics page](../workstreams/topics.md#j-library-areas)) |
| A wire-envelope Kind? | Not yet. Models state envelopes and statuses as protocol Rules until a real model needs a Kind |
