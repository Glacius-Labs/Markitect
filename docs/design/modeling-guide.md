# Modeling guide: what a well-modeled project contains

**Superseded:** [DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation) replaced the typed YAML form this guide describes with Markdown Definitions without typed fields; step T1 built it. The [model language guide](../guides/model-language.md) describes Main, and the modeling skill shipped in the binary teaches the new form. This page is kept for its reasoning.

**Status:** working version v0.3, 28 October 2026, of workstream W2 of the [canonical model track](model-track-charter.md). v0.2 was independently reviewed and includes the owner's architecture decisions of 26 October; v0.3 names the Kinds of the [standard Libraries](standard-libraries.md) and follows the Library decisions of 27 and 28 October ([source](../concepts/sources/library-questions-20261028.md)): refusals have no order, an Interface exposes refusal codes, and an Invariant lives with its data. It changes with the standard Libraries (W4) and the pilots (W6). It builds on the [founding intent](../concepts/sources/founding-intent-20261026.md), [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains), [DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate) and [Kind interfaces](../concepts/register.md#dec-075-kind-interfaces-and-one-core-120). The [quality criteria](model-quality-criteria.md) (W3) are its checkable side.

## The idea in one paragraph

Write the model the way you would write a program in a typed language.
- **Kinds are the types:** a project defines its own, or takes them from a Library.
- **Typed fields carry the shape:** names, arguments, codes, references.
- **The body is the function body:** short binding prose for the meaning a type cannot carry.
- **Packages are the modules,** and each one becomes a Manager.
- **Every decision the owner made is in the model, once.** Everything the model leaves open is free for the realization.

## 1. Who decides, who writes

- **The owner decides intent.** AI agents write the model through skills and Markitect's tooling (founding intent).
- **What the model states always holds** (DEC-063). A statement in the model is therefore a decision.
- **So an agent writes only decisions the owner made or approved in the model diff.** It never adds an architecture, a policy or a constraint on its own. Everything else stays open.

## 2. What a project model covers

| Concern | What the model states | Library Kinds | Lives in |
|---|---|---|---|
| **Project** | Purpose, scope and non-goals; the technical boundary (language, build and test commands, permitted dependencies); delivery obligations | `Practice.Project`, `Practice.Rule` | The project Package (DEC-061) |
| **Values** | Value formats and ranges, units, money and time rules, enumerations | `Values.Scalar`, `Values.Enumeration`, `Values.Unit` | A shared kernel Package |
| **Domain data** | Entities with their fields (as object lists on the Entity), relationships and invariants | `Data.Entity`, `Data.Invariant` | The Package that owns the Entities. An Invariant lives there too and that Package enforces it; it is not placed |
| **Behavior and API** | Operations with arguments, refusals with their wire codes, results and permissions; events | `Api.Operation`, `Api.RefusalCode`, `Api.Role`; events have no Library Kind yet | The Package that owns the behavior |
| **Cross-cutting policies** | Rules that apply to many Packages, such as authorization or the order of checks | `Practice.Rule`, with its scope in `appliesTo` | A shared Package. It reaches every Manager through modeled scope: a `scope` selector in `execution.yaml` names the Kind and the field ([context_scope.go](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/internal/organization/context_scope.go)) |
| **Architecture** | The module structure **is the Package tree** (section 3). On top of it: allowed dependency directions, each Package's interface, and placement decisions the tree cannot show, such as where a policy is realized | `Architecture.Interface`, `Architecture.DependencyRule`, `Architecture.Placement` | The Package tree itself; rules and placement in the project Package; interfaces in each Package |
| **Way of working** | Test, documentation and convention obligations; required checks | `Practice.TestObligation`, `Practice.DocumentationObligation`, `Practice.ConventionObligation` | The project Package; checks in `execution.yaml` |

**Architecture is the owner's decision, encoded as the model's own structure.** In the owner's words: "Of course the project owner should decide the architecture and encode it in the canonical model" ([source](../concepts/sources/model-architecture-20261026.md)). Encoding it does not mean describing it a second time; the owner approved that refinement in the same source, and its [recorded meaning](../concepts/sources/model-architecture-20261026.md#recorded-meaning) states the reading. The model is code, so it has a structure of its own; a separate component description beside that structure would be a second architecture that drifts. This is in the spirit of concept §7.1, where the Manager organization is derived from the model and not maintained separately; the same holds for the code's module structure. So:
- the Package tree is the module structure, and the ownership areas in `execution.yaml` place each Package's code;
- dependency rules on Packages state the allowed directions, and the model's own references obey them too;
- Interface Definitions are each Package's public surface: the operations, types and refusal codes other Packages may use, as a package exports its functions, types and errors in code;
- placement decisions state what the tree cannot show, for example "authorization is realized once, in `pipeline`";
- layers inside an area become child Packages.

This guide's rule: leave architecture open only where the owner explicitly chooses to, and say so in the model.

**Binding versus free.** A statement is binding when callers rely on it at the boundary, or when the owner decided it (DEC-061: "documented as intended"; DEC-063: what the model states holds). What the model leaves open, by the owner's explicit choice, is a realization choice that parents settle among their children (concept §7.3, DEC-063). Say explicitly when something is free, for example "the internal layout of this component is free".

**Why architecture and way of working matter.** The founding intent names drifting rules, documentation and structure as the original problem. A model that covers only behavior is a requirements document and cannot keep those consistent.

## 3. Packages are modules

- **The Package tree is the architecture.** Do not add Component Definitions or a component list beside it; that is a shadow architecture.
- **One Package per module or area of responsibility,** with an ownership area for its realization. Project Kinds and Properties live with the feature that uses them (language contract §7.3); a Package of Kinds only needs no files of its own. A Package that holds rules but realizes nothing and defines no types is a topic folder: move its rules to where they are realized.
- **Dependencies point inward.** Feature Packages depend on the kernel and the shared policies, never the reverse. A policy states its rule and its scope; it does not list every operation it covers.
- **No cycles between Packages,** as in Go. Under D3 every typed reference and body link is an edge, including "does not use [[x]]". So link only what you depend on, and choose one direction for links between a parent and its children.
- **A shared decision is stated once and reaches each consumer.** Put it in the common parent or a shared Package, and make it reach every Manager that needs it by a reference, a link or modeled scope. A parent's Definitions do not reach child Managers on their own (select_context.go).

## 4. Definitions are small and typed

- **One decision per Definition.** More than about 150 words of prose is a warning sign; above 300 words, a Definition is usually missing. Tables of data are exempt.
- **Type what has shape:** arguments, codes, fields, references, scopes. Do not write "Arguments: …" as a Markdown list when a Kind can carry it.
- **Every shared term is defined:** a value type, a field, or an enum value. A name such as `memberId` used across three Packages must not live only as a string in prose.
- **Link what you depend on.** Context selection follows typed references and body links; an unlinked dependency is invisible to the Manager that needs it.
- **Build on Library Kinds instead of copying them.** Reuse a Library's Properties in your own Kind's `properties`, and declare `implements` when your Kind should be accepted wherever the Library interface is required. The compiler then checks that your Kind declares every Property of the interface. Nothing is inherited, so each Kind shows its complete contract.

```yaml
# A project Kind that implements a Library interface
imports:
  - {package: markitect/core, as: Core}
  - {package: markitect/api, as: Api}
id: LedgerOperation
kind: Core.Kind
body: An operation that also posts its effect to the ledger.
spec:
  implements: [Api.Operation]
  properties: [Api.wireName, Api.mutation, Api.arguments, Api.refusals, Api.results, Api.permissions, ledgerAccount]
```

## 5. How files relate to the model

- **The registry maps every realization file to the Definitions it realizes,** shard by shard, one owner per file. It is maintained by the Managers, not by the author.
- **Tests and documentation are realizations too** ([CPT-002](../concepts/register.md#cpt-002-tests-are-realizations-too)). Binding tests to individual statements is a later direction ([ENH-003](../concepts/register.md#enh-003-test-to-statement-binding-and-test-review)).
- **The generated view** (`docs/markitect/project.md`) is the readable form of the model. It is generated, never edited.

## 6. What does not belong in the model

The test ([DEC-079](../concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation)): would good documentation of the project state it this explicitly? If not, it is too detailed for the model.

- **Consequences of other rules,** restated "so that they are read only one way". If a rule needs that, the rule is unclear: fix the rule.
- **Process descriptions:** how Managers should work. That is the host's and the harness's job.
- **Undecided implementation detail:** private names, internal step order, data structures.
- **Evidence and status:** test results, run outcomes, review notes.

## 7. Starting and changing a model

- **Start small:** a project Package, the kernel and a few areas. Split an area when its Manager owns too much, as you would split a large module.
- **Identities are stable.** Moving a Definition to another Package changes its identity and every reference to it (language contract §9). Makerspace's C7 fold of `billing/payments` into `billing/invoices` is a real example. Plan Packages so that moves stay rare.
- **A change of intent touches the Definitions that state it.** If an ordinary change touches many Packages, the cut is wrong, the code equivalent of shotgun surgery. A genuinely cross-cutting rule change may touch many.
- **The model diff is the change order.** Keep it readable for the owner: one concern per change.
- **Refusals have no order.** An operation lists the refusals it can return, each with its condition. When several can apply to one request and it matters which one is returned, a Rule such as the order of checks, or the operation's body, states the precedence.

## 8. Worked example: Makerspace, remodeled (sketch)

The Phase C requirements leave the architecture to the implementer ("Internal package layout and design are the implementer's choice"). That makes them an incomplete model by this guide. The remodel therefore first needs owner decisions on the architecture, made by the requirements author acting as owner. For W6, the same decisions go into a copy of the frozen model too, so that the comparison still isolates the cut and the typing.

Illustrative architecture decisions, expressed as the Package tree plus rules:
- one Package per area, a `pipeline` Package for the request pipeline and the line protocol, and a `kernel` Package for shared values;
- dependency rules: areas and `pipeline` may depend on `kernel`; areas never depend on `pipeline`; `kernel` depends on nothing;
- placement: the order of checks, authorization and audit are realized once, in `pipeline`;
- interfaces: areas use each other only through their Interface Definitions, for example billing reads booking charges only through `bookings/charges-for`.

| Package (= module) | Owns | Notable Definitions |
|---|---|---|
| `makerspace` (project) | Purpose, technical boundary (Go module, standard library only, line protocol), delivery obligations, the dependency rules and placement decisions above | Project, Rules, DependencyRule, Placement |
| `pipeline` | The central request pipeline and the protocol adapter: order of checks, results, delivery, authorization and audit | Interface; Rules with modeled scope |
| `kernel` | Value types, money, time, records and the refusal codes shared by all areas | Scalars (`identifier`, `month`, `cents`), money and time Rules, RefusalCodes |
| `members`, `training`, `equipment`, `safety`, `bookings`, `materials`, `billing`, `reporting` | One area each, with its entities, operations and its interface | Interface, Entity, Operation |

A typed operation, with imports, links and the unchanged wire codes:

```yaml
imports:
  - {package: markitect/api, as: Api}
  - {package: kernel, as: Kernel}
  - {package: pipeline, as: Access}
id: generate-invoices
kind: Api.Operation
body: >-
  Creates the month's invoices as described in [[invoice|charges and invoices]]
  and marks the month as invoiced, even when there is no member to invoice.
spec:
  wireName: generateInvoices
  mutation: true
  arguments:
    - {name: month, fieldType: Kernel.month}
  refusals:
    - {refusal: month-not-ended, when: "the request time is before the first minute of the following month"}
    - {refusal: already-invoiced, when: "invoices for this month were generated before"}
  results:
    - {name: month, fieldType: Kernel.month}
    - {name: invoices, fieldType: invoice, list: true}
  permissions:
    - {role: Access.staff, scope: all}
    - {role: Access.admin, scope: all}
```

Each refusal code Definition carries its wire value (`code: month_not_ended`), so callers see no change. The pipeline policies reach every operation's Manager through modeled scope, which replaces the per-operation links to request processing, results and delivery. Roles live in `pipeline` with authorization, so the alias stays `Access`.

## 9. Checklist

Before a model is used for Apply or an evaluation, check it against the [quality criteria](model-quality-criteria.md):

1. **Coverage (R1):** project, values, domain, behavior, policies, architecture and way of working are covered. Architecture is decided by the owner and expressed as the Package tree plus dependency rules, interfaces and placement; there is no second, descriptive architecture (R5). Anything left open is stated as free by the owner's explicit choice.
2. **Decisions (H1, R2):** every owner decision is stated once and reaches each consumer; nothing was added that the owner did not decide.
3. **Structure (H2, H3):** no Package cycles; every Package owns an area or only defines types.
4. **Typing (H4, S2, S8):** shapes are typed; shared terms are defined; Library Kinds are extended, not copied without reason.
5. **Size (S1):** Definitions over 150 words of prose are justified; over 300 are split.
6. **Content (R3):** nothing restates another rule, describes Manager process or records evidence.
