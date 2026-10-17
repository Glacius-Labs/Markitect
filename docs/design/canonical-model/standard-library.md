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
- Further owner decision and question: "Interface sollte auch interne technische Verträge abdecken. Wie würde das am Bibliotheksbeispiel aussehen, und wie grenzen wir Interface dabei von Component ab?" Interface includes internal technical contracts. The concrete architecture and exact Property contracts below remain proposals.
- Further selection: "Okay, dann bleiben Actor, Interface und Component ebenfalls bei Körperlinks und spec: {}." This selects the empty structured contracts and body-link relationships for those three Kinds. It does not select the proposed internal Component split or every sentence of Kind wording.
- Next requested subject: "Lass uns als Nächstes Scenario und Check am Bibliotheksbeispiel durchgehen. Welche Properties brauchen die wirklich?" Their contracts below remain assistant proposals.

## Current design state

| Subject | State | Owning detail |
| --- | --- | --- |
| Concept, Rule, UseCase: body links and empty spec | Owner-selected direction | [Concept](concept.md), [Rule](rule.md), [UseCase](use-case.md) |
| UseCase.rules and its Property definition | Dropped from the proposal following the owner's challenge and selection | Rationale below |
| Interface covers internal technical contracts as well as external interaction | Owner-selected direction | [Interface](interface.md), internal example below |
| Actor, Interface and Component: body links and empty spec | Owner-selected direction | [Actor](actor.md), [Interface](interface.md), [Component](component.md) |
| Scenario and general-purpose Check: body links and empty spec | Assistant proposal awaiting discussion | [Scenario](scenario.md), [Check](check.md), example below |
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

The initial optional structured relations were UseCase.actors, UseCase.rules, Interface.useCases, Component.parts, Component.interfaces, Scenario.useCases and Check.scenarios. The selection of empty specs for the six software Kinds drops their proposed fields, including UseCase.rules. Scenario.useCases and Check.scenarios remain unselected candidates to assess below, not an implementation list. Reverse relationships would be derived rather than maintained twice.

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

The initially proposed empty specs and body links for Actor and Interface have since been selected by the owner. Giving the particular existing role and interaction point their own Definitions remains an illustrative model change. It would make the role and boundary explicitly addressable without adding authentication, self-service, protocol, permissions or runtime behavior.

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
| Interface.provider | The internal example states provider and consumer relationships in Component bodies; it requires no second structured list. |

A Property needs a demonstrated purpose beyond labeling existing prose. Exact schema generation, structured validation or independent parameter maintenance could supply such a purpose. A specialized protocol Kind could then declare its exact fields; it would not automatically inherit Interface behavior under v1.

## Internal contract and Component boundary

Owner-selected scope: Interface includes internal technical contracts. The earlier Actor-only working definition was too narrow. An internal consumer can be a Component without being recast as an Actor. The [Interface page](interface.md) owns the expanded definition; the [Component page](component.md) owns its responsibility distinction.

The new observation is the owner's explicit scope selection. Earlier [Library architecture exploration](../product-model-refinement/library-architecture-example.md) already used service and repository contracts, but its old format, frontend/backend/database choices and provider checks do not become current requirements. Current M1 [realization](library-example/m1/model/library/realization.yaml) deliberately leaves internal architecture free and requires no production database. The following is an illustrative architecture proposal, not a change to M1 or its Apply acceptance target.

| Definition | Proposed Kind | Owns |
| --- | --- | --- |
| operator | Actor | The role that operates the library. |
| callable-interface | Interface | The existing external JSON Lines interaction contract. |
| jsonl-adapter | Component | Translation between the external contract and internal calls. |
| lending-api | Interface | The internal lending call contract. |
| lending-service | Component | Responsibility for realizing borrowing and returns. |
| lending/borrow and lending/return | UseCase | The existing domain behavior, including conditions, effects and refusals. |

Both Components and Interfaces describe intended architecture. Neither is the source implementation itself. An Interface states expectations at an interaction boundary; a Component states which system part fulfills a responsibility. The same Component can offer multiple Interfaces, and its internal representation can change while those contracts stay valid. A separate Interface is useful where a deliberate boundary or consumer/provider agreement needs a stable, independently addressable owner; do not create one for every helper call.

### Proposed internal lending contract

One proposed Definition file in Package library:

```yaml
imports:
  - package: markitect/software
    as: Software
  - package: lending
    as: Lending
id: lending-api
kind: Software.Interface
body: >-
  An in-process contract for borrowing and returns.
  Borrow accepts a member identifier, a copy identifier and a borrowing date.
  It returns the created loan's identifier and due date, or a domain refusal,
  as specified by [[Lending.borrow]].
  Return accepts a loan identifier and a return date.
  It returns the loan result and distinguishes returned from already returned,
  as specified by [[Lending.return]].
  Each call completes before returning its outcome.
  This contract carries domain inputs and outcomes, without a JSON envelope
  or a transport correlation identifier.
spec: {}
```

The call semantics are a concrete proposed technical choice. Existing UseCases remain the owners of domain admission rules and effects. The contract does not copy the borrowing limit or independently redefine return eligibility. Exact Go signatures, type representation and package layout remain unspecified by this excerpt.

### Proposed consumer and provider

One proposed consumer Definition file in the same Package:

```yaml
imports:
  - package: markitect/software
    as: Software
id: jsonl-adapter
kind: Software.Component
body: >-
  Provides [[callable-interface]] and translates its requests and responses.
  For borrowing and returns, it consumes [[lending-api]].
  It maps request arguments to call inputs and call outcomes to JSON responses,
  preserving the external contract's ordering and error behavior.
  Domain admission and loan state transitions belong to [[lending-service]].
spec: {}
```

One proposed provider Definition file in the same Package:

```yaml
imports:
  - package: markitect/software
    as: Software
  - package: lending
    as: Lending
id: lending-service
kind: Software.Component
body: >-
  Provides [[lending-api]] and realizes [[Lending.borrow]] and [[Lending.return]]
  against the current library state, including their applicable obligations.
  It owns domain admission and loan state transitions for those operations.
  It does not parse JSON Lines or read or write standard input and output.
spec: {}
```

Only the borrowing/returning slice is expanded here. The adapter still has to satisfy every other operation of the external contract; this example does not select their internal providers. It introduces no persistence, extra process, frontend, dependency-injection framework or additional Manager. All of these Definition excerpts are design illustrations, not a complete compiled fixture.

The provider/consumer roles follow from explicit sentences, not from the link notation alone. The Core verifies target identity, not a formal provides/consumes relationship, actual code dependency direction or fulfillment. No universal single-provider rule or automatic provider selection is proposed. Component does not imply a source-language class or process; Interface does not imply a Go or C# interface declaration.

The owner has selected empty specs and body links for Component as well as Interface. The standard-library contract therefore adds no provides or consumes fields. A later demonstrated structured consumer could motivate a future proposal. A useful future observation would be that a consumer can be reviewed against this contract while its provider's internal representation changes, with no lost domain or external behavior. This discussion supplies no execution proof of that result.

## Scenario and Check proposal

The [worked cases](library-example/business-rules.md#worked-cases), [return UseCase](library-example/m1/model/lending/return.yaml), [availability Rule](library-example/m1/model/catalog/availability.yaml) and [independent evaluation contract](library-example/evaluation-contract.md) already distinguish expected behavior, a method of examination and actual evidence. The evaluation contract includes repeated-old-return case E08. It also states observable-state limits and separates runtime observations from source review.

Proposal: Scenario describes a concrete expected case; Check describes how to examine an expectation and decide whether observations satisfy it. The [Scenario](scenario.md) and [Check](check.md) pages own the proposed Kind wording. Both use empty specs and body links. Neither is an execution result. The new observation is that these existing examples can express the distinction without the initially proposed Scenario.useCases or Check.scenarios fields.

The following two proposed Definition files would live in Package lending. They are design illustrations, not changes to M1 or a compiled fixture. The core domain and interface Definitions remain the source of their meaning.

### Proposed repeated-old-return Scenario

```yaml
imports:
  - package: markitect/verification
    as: Verification
  - package: members
    as: Members
  - package: catalog
    as: Catalog
id: repeated-old-return
kind: Verification.Scenario
body: >-
  Start with one registered active [[Members.member]] and one registered
  available [[Catalog.copy]], with no open loans.
  [[borrow]] the copy on 2026-10-01 and retain the resulting loan identifier
  as firstLoan. [[return]] firstLoan on 2026-10-02.
  Borrow the same copy again on 2026-10-03 and retain its new loan identifier
  as secondLoan. Both borrowings and the first return succeed.
  Return firstLoan again with the valid date 2026-10-04.
  The final operation reports already returned and changes no state.
  firstLoan remains closed with its original return date 2026-10-02.
  secondLoan remains open and unchanged, the member has exactly that one
  open loan, and the copy remains unavailable under [[Catalog.availability]].
spec: {}
```

firstLoan and secondLoan are names for captured results, not mandated generated identifier strings. The dates are example inputs; the expected old return date is owned by the case, not another loan policy. The case does not repeat the five/three admission limit and is applicable under both existing policies. It illustrates already accepted return behavior rather than adding a new rule.

### Proposed observation Check

```yaml
imports:
  - package: markitect/verification
    as: Verification
  - package: library
    as: Library
  - package: catalog
    as: Catalog
id: check-repeated-old-return
kind: Verification.Check
body: >-
  Examine [[repeated-old-return]] through [[Library.callable-interface]]
  in a fresh empty program process. Arrange the state using normal
  registration and loan operations, capturing the actual loan identifiers.
  Before and after the repeated old return, inspect both loan records,
  the member's open-loan list and count through [[inspect-loans]],
  and the copy through [[Catalog.inspect-copy]].
  Pass only when setup succeeds, the final response reports alreadyReturned,
  all observations satisfy the Scenario, and the before/after state agrees.
  A released copy, changed return date, closed second loan, incorrect count,
  malformed response or missing required observation prevents a pass.
  Retain the request/response trace and assertions against the exact candidate
  and model basis. Public observations do not establish absence of changes
  in inaccessible internal state.
spec: {}
```

The Check points to the Scenario for the expected behavior and adds the observation method. It does not copy the domain rule into a second independent catalogue. A concrete test, evaluator or review realizes this procedure. The record of a particular run holds actual candidate/model bindings, results and findings. Such a record belongs to evidence handling; no result file, status or execution timestamp is a Property of the intended Check.

A Check can also directly examine a Rule, Interface, Component or other Definition; a Scenario is not a mandatory intermediate node. A Scenario can receive different suitable Checks, for example public-interface observation and a source review examining a different aspect. These are meaningful distinctions only when their methods and claims actually differ. Linking a Check to a Scenario establishes neither test coverage nor successful execution.

### Property assessment

| Candidate | Assessment for the general-purpose Kinds |
| --- | --- |
| Scenario.given, when, then | Useful optional body headings; no structured editor, runner or consumer currently needs separate fields. |
| Scenario.useCases or Check.scenarios | Body links already identify the targets together with the precise statement about them. Typed lists would restrict targets without proving completeness or coverage. |
| Check.command | Not every method has a command. A required exact command may be model intent, but no universal command field is justified. See the existing narrower Kind below. |
| Check.testFiles | Actual realization files and their assignments belong to the registry; a particular required output can still be stated as intent where necessary. |
| Check.status, lastRun or observed result | These describe a particular execution, not the intended procedure. |
| Check.execution profile or operational budget | These are execution choices; a behavioral performance requirement would remain canonical intent. |

There is one concrete reason to retain structure in an existing narrower Kind: the repository's [maintenance Check](../../../.markitect/model/vocabulary/check.yaml) has a required command Property, with a [Library-oracle instance](../../../.markitect/model/tooling/gates/library-oracle.yaml). [Module configurations](../../../.markitect/README.md) bind hook/workflow bytes and selected pipeline scalars to literal commands. This is a demonstrated structured consumer. Do not remove that existing contract or assume that its source-maintenance meaning is universal. A specialized command-bearing Kind can coexist with the proposed general Check; v1 supplies no inheritance between them.

Routine Checks can use parameters from an exact selected model, with semantic review when expectations change. The existing independent Library experiment deliberately freezes separate five/three profiles; this proposal does not change those controls or create a second evaluator. Neither new Scenario nor Check Definitions are prerequisites newly added to the two-Apply target.

Recommendation pending owner selection: both general-purpose Kinds use body links and empty specs. Review their proposed meaning before treating the complete eight-Kind set as selected. No scenario execution, new test implementation or behavioral proof is supplied by this documentation.

## Remaining discussion

- Owner review of Scenario and general-purpose Check meaning and empty specs. The six software Kinds already have selected empty specs and body-link relationships.
- Exact Kind wording remains a working formulation. Interface's internal/external scope is selected; the particular call contract and Component split remain illustrative architecture proposals.
- Decide distribution model, Package identities and update policy before publishing them.
- Preserve the Library's complete intent when proposing a vocabulary migration; do not alter the M1/M2 Apply acceptance target merely to demonstrate the standard library.
- Examine a small Tasks model after the Library to distinguish shared needs from Library-specific vocabulary. This is a proposed design check, not authorization for a study or proof of benefit.
- A useful result would reduce repeated basic definitions while preserving expressiveness and understandable Manager context. Compiler success and Definition counts do not measure that benefit.

## Recording practice

Continue this owner record and the relevant term pages as the discussion proceeds. Record owner selections explicitly; keep unselected recommendations and superseded proposals labeled. Actual vocabulary fixtures, locks and executable probes remain unchanged in this documentation increment. No generated model view or canonical self-model change is required for this documentation-only record.

Return to the [canonical-model design map](README.md).
