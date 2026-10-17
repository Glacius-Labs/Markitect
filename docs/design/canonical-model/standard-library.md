# Standard-library design discussion

Date: 17 October 2026. This is the owning discussion record for the proposed standard library. Per-Kind pages own their proposed Definition text. The [language specification](minimal-language/language-contract.md) remains the sole language contract. This record changes neither the compiler nor the Library fixtures and does not establish Apply or runtime evidence.

## Source and selected decisions

Source: the owner's standard-library design discussion in this chat on 17 October 2026. Direct quotations below are owner statements; recommendations elsewhere are assistant proposals unless explicitly selected.

- Initial request: "Schaue dir das aktuelle Konzept und die Arbeiten an um Kontext zu sammeln. Dann lass uns das hier durchsprechen und designen. Erstelle mir einen gut durchdachten und designeten Vorschlag". The owner described reusable building blocks analogous to a programming-language standard library.
- The owner chose to examine the building blocks individually, beginning with Concept, Rule and UseCase in the Library example.
- The owner challenged the proposed UseCase.rules Property: "Welchen konkreten Mehrwert hat die rules-Property gegenüber den Links im Body? Gerade wenn die LendingPolicy dort gar nicht hineinpasst: Wäre es nicht einfacher, alle Beziehungen über die Körperlinks auszudrücken?"
- Selected decision: "Okay, dann bleiben wir bei Körperlinks und spec: {} für die drei." Thus Concept, Rule and UseCase use body links for their domain relationships and have empty structured field contracts. The selection does not approve every sentence of proposed Kind wording or settle other Kinds.
- Next requested subject: "Lass uns als Nächstes Actor und Interface am Bibliotheksbeispiel durchgehen. Welche Properties brauchen die wirklich?" Actor and Interface remain under discussion.
- Recording instruction: "Denke daran, alles festzuhalten, sodass wir das ins Konzept mit aufnehmen können und nichts vergessen." Keep selected decisions, proposals and unresolved questions distinguishable as the discussion continues.

## Current design state

| Subject | State | Owning detail |
| --- | --- | --- |
| Concept, Rule, UseCase: body links and empty spec | Owner-selected direction | [Concept](concept.md), [Rule](rule.md), [UseCase](use-case.md) |
| UseCase.rules and its Property definition | Dropped from the proposal following the owner's challenge and selection | Rationale below |
| Actor and Interface: body links and empty spec | Assistant proposal awaiting discussion | [Actor](actor.md), [Interface](interface.md), example below |
| Standard-library distribution, Package names and remaining Kinds | Assistant proposals; not selected | Initial proposal below |
| Production library, migration and consumer behavior | Not implemented by this record | Existing fixtures remain unchanged |

The initial source assessment used Main `f8c1acaa169c9955d237e17cd245add963c155bd`. Before recording, the isolated documentation branch advanced to integrated Main `f23eaa2072226d905a8d79b8d6c20f419dfdc136`. These are source observations, not compatibility or execution evidence. The canonical v1 compiler and read-only preparation exist; full Apply remains separate delivery work.

## Initial proposal and its boundaries

The initial recommendation was an explicitly imported library of ordinary Definitions built on the unchanged Core. Its purpose is reusable meaning and contracts for authoring, realization and review. Imports confer no execution rights, and foreign library Packages create no local Manager slots.

The proposed first target was eight Kinds:

| Proposed Package | Kinds |
| --- | --- |
| markitect/software | Concept, Rule, Actor, UseCase, Interface, Component |
| markitect/verification | Scenario, Check |

Component would retain its earlier general, nestable meaning, including logical parts without a separate process. Scenario would describe an expected concrete situation, actions and result. Check would describe a verification procedure and success criteria; actual results remain execution evidence.

The initial optional structured relations were UseCase.actors, UseCase.rules, Interface.useCases, Component.parts, Component.interfaces, Scenario.useCases and Check.scenarios. UseCase.rules was subsequently dropped. The other fields are unselected candidates to examine individually, not a remaining implementation list. Reverse relationships would be derived rather than maintained twice.

A single proposed distribution model, `markitect-standard`, would carry both Packages and depend on the existing Core model. Exact manifest dependencies, per-file imports and the complete external lock would bind its contents. Bundled availability with explicit imports is proposed. No registry, automatic update selection or additional language mechanism is required for the first design.

Proposed version policy: preserve stable identities, keep published contents immutable and bind updates explicitly. Changes to Kind bodies can change obligations while preserving structural validity. New required fields and stronger obligations need explicit contract review. One version for both Packages simplifies use but couples their updates; this remains a tradeoff to decide.

HTTP, messaging, storage, deployment and architecture-specific vocabulary were suggested as later focused libraries. Project, System, Module, Workflow, Decision and Artifact were deferred pending their concrete purpose and existing open boundaries. A Project Kind is not required to establish model identity or a root. A Component relationship does not create Manager ownership; local Package topology does.

The existing optional modeled-scope mechanism can route a Rule stated to cover local Packages: execution configuration selects the Kind and Property carrying scope. No import or ordinary body link automatically activates a rule globally. This does not select an appliesTo Property for the three basic Kinds.

## Why the three basic Kinds need no rules list

The proposed Property had two additional capabilities: exact Rule target checking and an explicit relation meaning that a UseCase must satisfy those Rules. It could prove neither completeness of the list nor fulfillment by artifacts. In the Library, LendingPolicy is a separate Kind carrying applicable obligations; it cannot be a value of a reference Property targeting Rule.

The owner selected the simpler form. A sentence such as "Borrowing is allowed only when the member satisfies [[Members.eligibility]], the copy is available under [[Catalog.availability]], and the resulting open-loan count satisfies [[loan-policy]]" states the required relationship directly. Links identify their targets; the sentence states their meaning and applicability. No additional rules or uses list repeats them.

The selection concerns the domain relationships of Concept, Rule and UseCase. It does not eliminate the Core kind reference, typed values or useful custom Property contracts. A Kind can have meaningful common obligations in its body without declaring spec fields.

The existing [loan](library-example/m1/model/lending/loan.yaml), [availability](library-example/m1/model/catalog/availability.yaml), [eligibility](library-example/m1/model/members/eligibility.yaml), [borrow](library-example/m1/model/lending/borrow.yaml) and [return](library-example/m1/model/lending/return.yaml) provide the concrete example. They already use bodies and links with empty specs. A Concept describes intended meaning, not a particular live loan or a new record Kind. A Rule states a condition and its scope. A UseCase states an action, inputs, admission, results, refusals and preserved state.

The separate [LendingPolicy](library-example/m1/model/lending/lending-policy.yaml) declares structured parameters. Its [loan-policy](library-example/m1/model/lending/loan-policy.yaml) has maxOpenLoans 5 and loanPeriodDays 14. It remains directly linked from behavior; no wrapper Rule duplicates its obligations. Changing five to three stays a change to the existing single parameter owner.

Core v1 has exact nominal Kind targets and no inheritance or union reference target. Reusing Property Definitions in another Kind does not make it a subtype or automatically inherit another Kind's body. Generic analysis must not assume all applicable obligations have Kind Rule.

## Actor and Interface proposal

The current [Library scope](library-example/m1/model/library/library.yaml) already describes a trusted operator. Member is domain data, not a login account. The [callable interface](library-example/m1/model/library/callable-interface.yaml) already declares a local Go program with a JSON Lines contract; it currently has Kind software/Rule.

Proposal: give the existing role and interaction point their own Actor and Interface Definitions, with empty specs and body links. This makes the role and boundary explicitly addressable. It adds no authentication, self-service, protocol, permissions or runtime behavior.

### Proposed operator

One proposed Definition file in Package library:

```yaml
imports:
  - package: markitect/software
    as: Software
id: operator
kind: Software.Actor
body: >-
  A trusted role operating [[library]] through [[callable-interface]].
  The operator supplies requests and observes their responses.
  It performs the supported registrations, membership status changes,
  borrowing, returns and inspections, and requests overdue notices.
  A member record does not grant this role, and this example requires
  no authentication or authorization mechanism.
spec: {}
```

An Actor describes a role, not a specific person, account or automatic grant of authority. A person or another system may fulfill a role. Do not invent a Member Actor without an intended direct interaction. This example assumes no separate human UI around the JSON Lines program.

### Proposed interface

One proposed Definition file in Package library, shown as a shortened contract excerpt:

```yaml
imports:
  - package: markitect/software
    as: Software
  - package: members
    as: Members
  - package: lending
    as: Lending
id: callable-interface
kind: Software.Interface
body: >-
  The local program boundary through which [[operator]] operates [[library]].
  Requests and responses are UTF-8 JSON Lines on standard input and output.
  It processes requests sequentially and writes and flushes exactly one
  response before reading the next request.
  Its operations include [[Members.register-member]], [[Lending.borrow]]
  and [[Lending.return]]. Expected refusals are observable and leave
  domain state unchanged. Diagnostics go to standard error.
spec: {}
```

The actual Library contract also includes every other operation, exact envelope and record fields, correlation, invalid-request handling, process lifetime and remaining constraints. A future model edit must preserve all of them. This excerpt is not a replacement fixture or a tested input set.

UseCases would identify the role through an explicit link, for example "[[Library.operator]] supplies a member identifier, a copy identifier and a borrowing date". The interface owns protocol and operation mapping; the UseCase owns domain effects and refusals. An Actor's link to the interface need not repeat its whole operation catalogue.

Changing callable-interface from Rule to Interface is a proposed Kind change under the same identity, not a new parallel owner. The existing file also contains Go, build and test requirements. Consider moving realization-wide requirements to the existing realization Rule while preserving one canonical owner and explicit links. Do not drop those requirements as part of a reclassification.

### Property assessment

| Candidate | Present reason to omit it |
| --- | --- |
| Actor.interfaces | Body links already identify how the role interacts. |
| Actor category, such as human or system | No current behavior consumes that classification; a role may have different fulfillers. |
| Actor permissions | The Library assumes a trusted operator and no authentication. An Actor declaration does not grant authority. |
| Interface.actors or Interface.useCases | Body links and the existing operation table identify participants and offered behavior. A second list would require maintenance. |
| Interface.protocol or version | The exact JSON Lines protocol and identifier are already binding prose; no structured consumer has been identified. |
| Interface.provider | Component/provider modeling has not been designed in this discussion. |

A Property needs a demonstrated purpose beyond labeling existing prose. Exact schema generation, structured validation or independent parameter maintenance could supply such a purpose. A specialized protocol Kind could then declare its exact fields; it would not automatically inherit Interface behavior under v1.

## Remaining discussion

- Owner review of Actor and Interface meaning, wording and empty specs; especially whether an Interface also covers internal technical contracts.
- Examine Component, Scenario and Check individually before selecting their fields or the complete eight-Kind target.
- Decide distribution model, Package identities and update policy before publishing them.
- Preserve the Library's complete intent when proposing a vocabulary migration; do not alter the M1/M2 Apply acceptance target merely to demonstrate the standard library.
- Examine a small Tasks model after the Library to distinguish shared needs from Library-specific vocabulary. This is a proposed design check, not authorization for a study or proof of benefit.
- A useful result would reduce repeated basic definitions while preserving expressiveness and understandable Manager context. Compiler success and Definition counts do not measure that benefit.

## Recording practice

Continue this owner record and the relevant term pages as the discussion proceeds. Record owner selections explicitly; keep unselected recommendations and superseded proposals labeled. Actual vocabulary fixtures, locks and executable probes remain unchanged in this documentation increment. No generated model view or canonical self-model change is required for this documentation-only record.

Return to the [canonical-model design map](README.md).
