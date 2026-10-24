# Derivation of the minimal Core and Connector counter-probes

**Historical and not normative.** The discussion states, variants and reports below keep their original scope of validity. Later decisions are not applied to them retroactively.

The [v1 language contract](language-contract.md) is the completed design baseline confirmed by the owner on 16 October 2026. This historical record contains no open v1 language decisions. As of 17 October 2026, the current source implements the v1 product compiler. The design probes below show only their recorded scope; the [delivery plan](../../../implementation-plan.md) owns current product status and remaining work.

| Entry point | Role |
| --- | --- |
| [Language specification v1](language-contract.md) | Sole normative source of the chosen canonical form. |
| [Diagnostic contract](diagnostic-contract.md) | Separate tool profile for exact coordinates; not a precondition of the model digest. |
| [Language history](language-history.md) | Earlier coherent derivation, changes and the closing reconciliations of that time. |
| [Procedure proposals](procedure-proposals.md) | Not yet chosen elaborations on context, impact and semantic index. |
| [Exchange example](exchange-example/README.md) | Current executable inputs and bounded check evidence. |

The historical bootstrap, compact and identity probes further below are not v1 conformance evidence. The [shared design state](../consolidation.md) leads to the remaining product questions.

## 1. Kind was already the starting point

The user's recollection is correct. Two existing foundations point in this direction:

- The structural Core (`src/internal/core/README.md`) compiles explicitly supplied schemas and definitions. A schema declares freely named Kinds and their closed, typed Properties. References bind against exact target Kinds; graph edges arise from them. `kindReference` can point to a Kind, but does not define it and does not activate any execution logic. The compiler is not restricted to the five Kinds of the present project vocabulary.
- The [proposal on typed prose of 14 October](../../product-model-refinement/canonical-model-proposal.md#23-programmiernahe-arbeitsrichtung-typisierte-prosa) explicitly describes the Kind as a language construct that determines the interpretation of its individual definition body. Meaning, references and expected realization were already meant to be explained once at the Kind there.

The present project workflow (`src/internal/core/projectmodel/projectmodel.go`) nevertheless uses a fixed composed contract with Manager, Statement, Artifact, Check and Decision. Free schema data in the Core is not the same as a finished project loader, arbitrary context/impact rules or Apply for user-defined Kinds. The [fixed Main comparison](../current-main-comparison.md) keeps its role of that time.

This draft connects the existing structural approach with the extensibility of typed project documentation that has now been commissioned. It does not start a new language implementation and does not bring back a historical plugin/installation system.

## 2. The small foundation

This section describes the starting basis from the existing Core. The uniform definition form with Kind and Property that has since been chosen is summarized in section 14.

| Language term | Function |
| --- | --- |
| **Schema / vocabulary** | Versioned contract of the types available in this model and their meaning. |
| **Kind** | Defines a kind of statement and its permitted Properties. Its meaning explains what a concrete definition declares. |
| **Property** | Named, typed value with cardinality. A Property of type reference defines a directed, named relation to a target Kind. |
| **Definition** | Concrete named target description: identity, Kind, individual binding body and structured values or relationships. |

Identity, namespace, reference resolution and canonical content comparison are shared language mechanisms. They are not further domain project Kinds. The core needs to build in neither System nor Command nor Rule as fixed parts. Even an independent Rule definition wanted later is declared by a vocabulary; the fundamental normative force of canonical statements does not need this special type.

**Minimal proposal:** Custom relations are initially defined on the source Kind as reference Properties. Therefore `Component.provides` is a different declaration from `System.provides`, even if both use the same word. The resolved reference keeps source identity, property path and target identity. Equal names do not automatically lead to globally equal semantics.

An independent Relation Kind is only needed when relationships themselves need an independent identity, own statements or relationships. Attributes of a relationship can partly already be expressed by structured objects with reference Properties. Neither a new primitive for arbitrary edges nor a general graph rule engine is introduced here.

### 2.1 Remit of the Core and role of the libraries

The user's clarification later in the conversation: The Core must completely provide the general means of expression required for Markitect. Libraries serve exclusively to import finished vocabulary. They do not unlock any additional language capabilities. The user now commissions the definition of this Core and its constituents; the composition below remains a discussion proposal.

**Yardstick:** A project can define the vocabulary it needs itself, without an imported vocabulary library, and have it worked on with the same Markitect capabilities. A local Kind has the same linguistic standing as an imported Kind. The Core knows no privileged Component, Rule or Mandate type.

General expressiveness does not mean being able to decide every domain truth formally. Structured declarations and binding prose together carry the project description. Statements that cannot be checked formally remain obligatory and need suitable assessment.

### 2.2 Proposal: four basic building blocks and shared language mechanisms

The four basic building blocks from section 2 are made concrete as follows:

| Building block | Minimum content | Connector example |
| --- | --- | --- |
| Schema | Identity and exact version of a type contract, its meaning and contained Kinds. | A schema declares the project-specific communication vocabulary. |
| Kind | Identity in the schema, general meaning, permitted Properties and applicable requirements. | Exchange describes an agreed message exchange and its roles. |
| Property | Name, meaning, value type, cardinality and, where applicable, reference target or object structure. | Exchange.sender references exactly one Container. |
| Definition | Identity, exact Kind, individual target description and values of the declared Properties. | list-documents determines API, Connector, channels and waiting period. |

Schema here is the language contract. Library denotes its provision or assembly for reuse; it is not a second sort of type definition. A schema can be written locally or obtained from a library. Import selection and file loading must produce an unambiguous input set; transport and package installation are not part of the meaning of a Kind.

For this the Core needs shared mechanisms. They are constituents of the language, but not further domain project Kinds:

| Mechanism | Proposed minimum contract | Status compared with the previous Core |
| --- | --- | --- |
| Binding body | Schema, Kind, Property and Definition carry interpretable prose. Kind and property contracts apply to their uses; individual statements supplement them and must not silently override contradictions. | Purpose fields present; complete authoring and interpretation rules still to be specified more precisely. |
| Identity and references | Unique names in a defined namespace and schema/Kind contract; references are resolved exactly. | Present. The exact authoring form for body links remains open. |
| Values and cardinalities | Simple values, structured objects, single values and lists as well as references to definitions and Kinds. | The existing eight value types are the starting point; further type constructors only on concrete need. |
| Requirements | Binding requirements stand in the bodies. Their subject and scope of validity are explained there. | Structure check present; requirements and roles are deferred for now. Additional formal constraint operators remain a separate open extension question. |
| Bound model state | All type contracts and definitions used are recorded together; a deviation in meaning with unchanged instances is also a change. | Deterministic normalization, graph, digest and provenance present; library selection and product connection remain to be designed. |

A relationship therefore needs no fifth universal declaration format for now: its role is declared as a reference Property, the concrete reference stands at a definition. If a relationship itself needs identity, several participants or own statements, a dedicated Kind can describe it. Exchange is already such an example with several referenced roles. Whether this suffices for all relevant cases remains an explicit counter-probe.

### 2.3 What follows from this for Markitect

The language must make the complete applicable contract of a definition accessible: its own description, its Kind, that Kind's Properties and requirements, and resolved references. Omitting an imported or locally defined Kind meaning must not lead to a silent release from obligation.

Generic context formation, change assessment, realization and independent checking build on this. Their procedures continue to belong to the Markitect product, even if their execution data is not part of the canonical project description. The existing structure compiler alone does not yet fulfil this product remit. The new direction does not incidentally lift the existing obligations of delegation, independent checking or repository assignment; it does not yet stipulate their new linguistic connection.

In particular, **a resolved relationship is not yet a computable rule of effect**. The Core can provide its endpoints, declaration and meaning. Which contexts and checks it affects must either be described by an explicit generic language mechanism or be treated as a justified semantic assessment. The relation name alone must trigger neither tree semantics nor rule applicability nor write authority.

The next counter-probes therefore concern: individual and general requirements including their scope of validity; relationships with their own description; explicit references in the body; and the meaning of a change at the Kind for its definitions. Inheritance, target type unions, an independent Relation primitive and a general formal rule language remain open extension candidates. This inventory does not decide them in advance.

## 3. Meaning belongs to the language contract

The Kind body defines its general meaning; the definition body contains the individual target statements. The description of `Exchange`, for example, explains once the roles sender and receiver and the reply mapping. The concrete definition `list-documents` stipulates its participants, channels, operation and particularities.

The compiler checks whether a sender references an existing definition of the permitted Kind Container. It does not thereby prove that the process concerned sends messages correctly. That belongs to the interpretation, realization and semantic checking of the described project. The language meaning is not an agent role and not a Mandate.

Correspondingly, the Property `Channel.topicPattern: string` initially only means that a string is present. That `{tenantId}` is to be filled in from an authorized context and must not contain MQTT wildcards is a declared obligation of the project vocabulary. A formal template check would be a separately defined validation. A new type name installs no parser, broker adapter or code generator.

All schema, Kind and Property bodies needed for interpretation belong to the bound model state. If the meaning of a Kind changes while the instance files are unchanged, this is still a semantically relevant model change. The existing Core already takes schema/Kind/Property purpose into account in the model digest; the new authoring form must preserve this content.

### 3.1 Two subjects of checking and the quality of the contract

Discussion proposal of 15 October on the user's question of how the meaning and rules of self-defined Kinds should be described and checked. The rules and fields below are a draft, not additionally implemented Core functions.

First the **subject of checking** is to be determined. A rule can concern the valid model description or express an obligation of the software that arises from it. In addition, the Kind contract itself needs a domain assessment of its comprehensibility, freedom from contradiction and suitability.

| Subject | Guiding question | Suitable check |
| --- | --- | --- |
| Model conformance | Does a concrete definition fulfil the declared Kind contract? | Compiler for explicitly formally described conditions; agent for domain prose, appropriateness and completeness. |
| Contract quality | Are Kind, rules and relations sensible, unambiguous and compatible with one another? | Domain model review; formal counter-probes and tools can contribute partial findings. |
| Realization conformance | Does the fixed implementation candidate fulfil the applicable obligations? | Deterministic tests and analyses, where suitable, plus independent agent assessment of their probative value and of the candidate. |

The choice of check method follows after that. A realization obligation can also be tested by machine; this execution, however, does not belong to the pure structure compiler. Conversely, the assessment of a model definition partly needs an agent when its meaning is given in prose. The division into model and realization is therefore not a division into automatic and human or agentic.

### 3.2 Content of a self-defined Kind contract

I would describe four contents at the Kind: meaning and subject of its definitions; structured Properties and relation roles; conditions for valid model definitions; obligations that the realization must fulfil. For the last two groups, distinguishable rule names are proposed if they are to be affected or assessed independently.

Such a rule explains its subject, its applicable statement and, where applicable, the condition of its applicability. A formally described condition can be an additional, exactly checkable constituent of this one rule. The named rules are maintained once at the Kind and apply to its definitions according to their declared scope of validity. A concrete definition supplements its individual obligations; general rules are not copied there again. The Kind body should not duplicate the rule meaning again as an independent second truth.

A built-in project Kind named Rule is not required for this. An optional rule section would be part of the meta contract for Kinds. Whether it gets its own structured fields or is expressed for now as named paragraphs in the body is an authoring format decision. Existing binding prose remains obligatory, even if it has no formal constraint.

### 3.3 Example at the Kind Exchange

| Statement of the Kind contract | Subject | What follows from it |
| --- | --- | --- |
| sender references exactly one Container. | Model | Already checkable in the Core through reference target and cardinality. |
| timeoutSeconds and maxAttempts are positive integers. | Model | The integer type is present; the positive value range condition needs an explicit new constraint function or a model review. minCount limits the number of values, not their numeric value. |
| Sender and receiver are chosen appropriately for the operation. | Model / contract quality | The compiler checks the target types. The domain distribution of roles needs review; a later complete formal mapping could check partial conditions. |
| A reply keeps the MessageId of the request. | Realization | An agent assesses the implementation and suitable mapping checks on a fixed candidate; the compiler cannot prove this from a reference graph. |
| The tenant comes from an authorized context. | Realization | Examine access paths, input provenance and suitable negative tests; an existing tenantId field is not sufficient as evidence. |
| Every receiver replica can process the operation distributed to its pool. | Model and realization with separate expectations | Declared capabilities could be compared in the model. Actual registration, configuration and operation need their own evidence. The declaration does not prove its implementation. |

#### 3.3.1 Meaning and Properties of the Exchange contract

Made concrete at the user's request: The following fields are a coherent contract draft, not syntax that has already been decided or implemented. The existing example files and their bounded structure check report remain unchanged.

The subsequent [check of the added value](#36-do-requirements-and-roles-need-their-own-core-semantics) questions requirements and roles as separate language means. The user then explicitly decides to leave them out for now. The structured variant below is therefore deferred and is kept only for comparison.

The general meaning reads: An Exchange definition describes the desired request/reply contract for an operation. It describes neither a single running exchange nor a message. sender issues the operation; receiver processes it. requestChannel and replyChannel describe the communication paths. The Kind binds these roles to one another; its requirements specify the expected behaviour more precisely.

| Property | Type / count | Meaning |
| --- | --- | --- |
| sender | Container / exactly 1 | Application in the requester role of this exchange. |
| receiver | Container / exactly 1 | Application in the processor role of this exchange. |
| command | Command / exactly 1 | Contract of the requested operation. |
| requestChannel | Channel / exactly 1 | Channel for the request direction. |
| replyChannel | Channel / exactly 1 | Channel for replies to the issuing instance. |
| timeoutSeconds | integer / exactly 1 | Waiting period per attempt in seconds. |
| maxAttempts | integer / exactly 1 | Maximum number of attempts including the first attempt. |

These roles are part of this Kind. The Core knows neither sender nor the MQTT meaning of Channel in advance. Every Property gets its own meaning and its structure contract. That a role points to Container describes no agent responsibility and grants no write authority.

#### 3.3.2 Scope of validity of a requirement

The proposed meta section is called `requirements` here; it makes the earlier illustrative `rules` sketch concrete. A requirement is a substructure of the Kind or definition contract, not an additional, fixed built-in domain Rule Kind.

| Item | Contract |
| --- | --- |
| Place of declaration | At the Kind, the requirement applies to every definition of exactly this Kind. At a definition, it only supplements that definition's individual obligations. No type inheritance is presupposed. |
| id | Stable local name within the declaring contract. Its qualified address includes the schema/Kind or definition identity. An equal local name at a definition does not override a Kind requirement. |
| subject | `model` denotes a requirement on the model description; `realization` denotes an obligation of its implementation. Both are canonical target statements. The selection does not stipulate the check method. |
| roles | Optional: explicit reference Properties of the declaring Kind whose participants are relevant for this requirement. Every role is bound relative to the definition under consideration. Without roles, the subject remains the definition or its entire realization in the described scope. |
| definition | Binding statement including domain conditions of its applicability. A condition that cannot be expressed formally is assessed semantically and treated as open in case of uncertainty. |
| constraint | Optional: formal part of the same requirement. Only known operators and valid bindings are permitted; the sketch below uses Property and a numeric lower bound. |

`roles` denotes the affected participants; the concrete division of obligations stands in the statement. It means neither that every participant takes on every partial obligation, nor does it determine the division of agents. In the first proposal, only direct reference Properties are permitted as roles, no freely formulated selector expressions or transitive graph paths. An unknown role name or a value Property at this place is a structural error of this proposed contract.

In particular, the context remains **the single Exchange definition**: A requirement for its receiver concerns the referenced Container in this role and in this exchange. It does not change that Container's general definition, does not automatically apply to other Exchanges and does not migrate wholesale to its Components. In the realization check, the relevant implementing parts must nevertheless be considered. Domain scope of validity and the needed check context are different.

#### 3.3.3 Contract excerpt with requirements

The excerpt stands within the Kind declaration of Exchange. The property contracts listed completely above stand next to it under properties; they are not repeated here.

```yaml
requirements:
  - id: positive-timeout
    subject: model
    definition: The waiting period per attempt must be at least one second.
    constraint:
      property: timeoutSeconds
      minimum: 1

  - id: positive-attempts
    subject: model
    definition: The attempt budget must allow at least the first attempt.
    constraint:
      property: maxAttempts
      minimum: 1

  - id: register-before-publish
    subject: realization
    roles: [sender]
    definition: >-
      The sender must set up the mapping of a waiting exchange before the
      publication of the request, so that an immediate reply can be
      mapped to the waiting exchange.

  - id: preserve-message-id
    subject: realization
    roles: [sender, receiver]
    definition: >-
      The receiver must send a reply with the MessageId of the request.
      The sender must map the reply within its instance to the waiting
      exchange by this MessageId. The CorrelationId serves cross-cutting
      tracing and does not replace this mapping.

  - id: reply-destination
    subject: realization
    roles: [sender, receiver]
    definition: >-
      The sender must give the request a reply address that is authorized
      for the issuing instance, according to replyChannel. The receiver
      must send the reply to this address. The delivery must not be
      distributed to an arbitrary other sender instance.

  - id: bounded-retries
    subject: realization
    roles: [sender]
    definition: >-
      The sender may trigger at most maxAttempts attempts and must end the
      waiting period per attempt after timeoutSeconds without a reply.
      Further attempts use the same MessageId. An expired waiting period
      does not prove that the operation was not executed.
```

This sketch is not part of the YAML files checked against today's Core. It does not stipulate a general expression language. An existing general integer type cannot learn a minimum operator from it on its own. The compiler must know and validate the permitted operator and its value/type semantics. Unknown or wrongly bound formal constraints are a finding, not silently ignored prose. Likewise, a contradiction between formal condition and rule text is a model error to be clarified; neither representation silently wins.

In a finished authoring form, the shared obligations are maintained once under requirements. The existing broad Exchange body in vocabulary.yaml still describes them partly in one paragraph; this evidence-bound first vertical slice is deliberately not rewritten here. In a later conversion, the meaning and the role description remain in the general body, while the named requirements carry their respective statements. These two development states are not a request to keep two independent truths manually in sync.

#### 3.3.4 Binding to list-documents and checking

For the existing definition list-documents, sender is bound to api, receiver to onprem, command to list-documents-command, requestChannel to requests and replyChannel to replies. The waiting period is 30 seconds per attempt, the budget at most two attempts. Without an additional stipulation, neither a fixed total period of 60 seconds nor a guarantee that the operation is executed exactly once in domain terms follows from this.

In the proposed format, this one definition can additionally declare:

```yaml
requirements:
  - id: late-replies
    subject: realization
    roles: [sender]
    definition: >-
      Replies without a matching open waiting state are discarded and
      must not complete any other waiting exchange.
```

This requirement only supplements list-documents. For this example it is extracted from the already existing definition body, not introduced as a new general Exchange obligation. Kind requirements and concrete requirements apply together. The applicable contracts of the Command and the Channels also remain to be taken into account. Neither the property values nor the individual prose can silently switch off the general obligation of reply mapping. A contradiction is treated as a finding to be clarified; a future explicit exception mechanism is not decided here.

Today's compiler can check the property types, cardinalities and reference targets. A compiler extended by this contract could additionally check role binding, requirement addresses and the two minimum constraints. From these declarations it cannot determine whether the running implementation registers the waiting mapping in time or processes replies correctly. That needs implementation and integration evidence as well as independent assessment against the fixed model and candidate state. MessageId is here a runtime quantity described in the protocol, not a Property to be filled in for every Exchange definition.

### 3.4 Division of labour and evidence

The compiler can check exactly formulated conditions over the completely provided model state. With suitable explicit constraint functions, field comparisons, declared uniqueness or acyclicity of a particular relation would later also be possible. Which relations require this is determined by the declared vocabulary contract. It must neither give `contains` tree semantics on the basis of its name alone nor derive a write or delegation authority from a domain use.

An agent assesses the comprehensibility and sufficient specification of the prose, domain contradictions, missing relationships and the suitability of the chosen rules. An independent Verifier assesses the fixed realization candidate against these obligations and needs for this Kind/Property meaning, concrete definitions, relevant neighbouring contracts, findings and suitable execution evidence. Implementation and successful own tests are not, by themselves, an independent semantic acceptance. If the contract is unclear, a finding or a proposed clarification is reported; the success condition is not reinterpreted to fit the implementation.

Results are limited to what they state: structurally checked, evidenced by a particular procedure on the fixed candidate, violated or still open. Required obligations that were not examined or cannot be decided remain open and are not summarized as fulfilled. The suitability of the test for the obligation must also be assessed. Result and execution data belong to the evidence; they are not new canonical target rules.

### 3.5 Proposal for the first language scope

Initially use the existing structure check, describe all meaning and realization obligations bindingly, and treat model review and independent realization check as separate evidence. Additional formal constraints are only introduced for concrete, recurring description needs; positive number ranges are an obvious first counter-probe here. To start, the approach needs neither arbitrary executable code in the model nor a general rule interpreter.

If a Kind contract is changed, its definitions and the affected realizations that arose from them are to be considered again. An unchanged instance document does not evidence an unchanged obligation. The digest already captures the change structurally; the generic context/impact and check control for it remains a product connection still to be designed.

### 3.6 Do requirements and roles need their own Core semantics?

A further user question: Can meaning, requirements and their scope of validity already be expressed through binding prose and Properties? This counter-probe checks the benefit of the proposed new language means. It is a recommendation, not a removal already decided by the user.

**Subsequent user decision:** requirements and roles are left out for now. The lean direction justified here is chosen for the next vertical slice; this does not yet imply acceptance of a particular file syntax or implementation.

**Result at the Exchange:** The obligations described so far need neither a privileged requirements section nor a role-binding roles field in order to be expressed and assessed against the implementation. The binding description of the Kind applies to every one of its definitions. The property contracts name the participants; the concrete definition binds them through typed references. A statement can explicitly declare its scope of validity.

An excerpt of the Kind body can read:

> Every definition of this Kind describes a request/reply contract for its specified operation. The following obligations apply to the applications referenced under sender and receiver in this exchange.
>
> The sender must set up the waiting mapping before the publication of the request. The receiver must preserve the MessageId of the request in the reply. The sender must map the reply to the waiting exchange by this MessageId; the CorrelationId does not replace this mapping.
>
> timeoutSeconds applies per attempt. In total, at most maxAttempts attempts are permitted, including the first. Further attempts use the same MessageId. A timeout does not prove that the operation was not executed.

The Properties sender and receiver remain reference Properties to Container with exactly one value. timeoutSeconds and maxAttempts remain integer Properties whose meaning and value requirements are described at the Properties. The reply mapping that applies jointly stands once in the Kind body. A separate section need not repeat these statements in parallel.

| Desired benefit | Prose and Properties | What requirements or roles would additionally provide |
| --- | --- | --- |
| Binding obligations | Already stipulated in the language contract for all canonical prose. | No additional normative force. |
| General versus individual obligations | The Kind body applies to its definitions; the individual description supplements the single definition. | Machine-readable subdivision, without by itself providing new domain meaning. |
| Scope of validity for sender and receiver | Reference Properties bind the participants. The text describes their obligations in the context of this Exchange definition. | A roles field can make participant assignment and permitted role names structurally checkable independently of the text. It proves neither the content assignment of the text nor the completeness of the role list. |
| Checking runtime behaviour | Implementation, tests and independent review against the complete contract. | A requirement record with prose does not automatically make the behaviour decidable. |
| Exact addressing of a single obligation | Initially a reference to Kind/Property/definition identity and the bound body with text location. | A dedicated stable requirement identity makes obligation-exact findings, references and evidence assignment easier. A general body anchor would be a different solution, to be defined separately. |
| Targeted contexts and change consequences | Complete Kind contract and relevant property/definition contracts as a safe, possibly broad basis. Semantic relationships need assessment. | Explicit assignment can make declared participants per obligation computable. The concrete processing semantics and the handling of missing declarations would additionally have to be stipulated. |
| Formal value conditions | Known constraints can in principle attach directly to the property contract. | A universal requirement section is not necessary for a minimum operator. Cross-property or graph-wide predicates may need a different point of attachment. |

The last row is not evidence of already implemented value ranges. The present Core does not know the operator minimum. **Formally checkable conditions and structured requirement catalogues are different design decisions.** We can add the former on concrete need, without reshaping all prose into requirement objects for it.

The price of the leaner form is explicit: The compiler cannot reliably compute individual obligations, role binding, completeness or domain conditions of application from a prose paragraph. Without requirements, evidence per individually identified obligation is not guaranteed by the authoring format alone. For a fixed model state, agents can derive a checklist with source references; it is a checkable work result with uncertainty about completeness and not a second canonical contract. In case of doubt, the complete applicable contract is taken into account and a missing check remains open. A narrower context derived solely from an unverified extraction would not be a reliable substitute.

Likewise, a Property named in prose and a formally bound body link are to be distinguished. A bare word sender creates no compiler-checked reference to the property declaration. A later general symbol/link mechanism could check such bindings without requiring a special roles field. Its contract is still open.

**Recommendation for the first Core contract:** Binding bodies at Schema, Kind, Property and Definition; typed Properties and references; Kind obligations apply to every one of its own definitions; individual statements supplement and do not silently lift general obligations. requirements, subject and roles receive no special Core processing for now. Their meanings discussed so far can be expressed in domain terms in the body. The structured variant from section 3.3 remains a candidate in case we evidence stable individual obligations and computable assignments as a necessary product capability.

Additional structure remains possible through self-defined Kinds, for example a project-defined Requirement Kind with relationships and prose. Its content does not thereby automatically receive the meaning of a special compiler operator or a new context control. A normal Property at Exchange moreover concerns its concrete definitions; it is not a substitute for a universal meta section at the Kind. The goal is to stipulate such a special mechanism only when a need is evidenced.

## 4. Concrete authoring form of the candidate

The two example files show the levels separately:

- [vocabulary.yaml](vocabulary.yaml) declares the eight project-specific Kinds and their Properties, including their own relations.
- [connector.yaml](connector.yaml) contains thirteen concrete definitions that use these types. They are project descriptions; individual Commands, tenants and process instances are runtime data.

The vocabulary contains, for example:

```yaml
apiVersion: connector.example.org/v1alpha1
definition: Vocabulary of the Connector project.
kinds:
  Exchange:
    definition: Describes a request/reply contract between two applications.
    properties:
      sender:
        definition: The application in the issuing role.
        type: reference
        target: {apiVersion: connector.example.org/v1alpha1, kind: Container}
        minCount: 1
        maxCount: 1
```

This excerpt shows only one relation. The complete vocabulary contains the target Kind and the further fields. A definition excerpt uses the contract like this:

```yaml
apiVersion: connector.example.org/v1alpha1
kind: Exchange
metadata: {namespace: connector, name: list-documents}
definition: The API requests the document metadata of the authorized customer environment.
spec:
  sender: {namespace: connector, name: api}
```

The two excerpts are not independently complete models. The two linked files together form the worked-out vertical slice. The short reference obtains its target Kind and its exact API version from the property contract. No resolution to whichever version happens to be newest and no selection of the first symbol of the same name.

`definition` denotes, in the example files so far, the body for Schema, Kind, Property and concrete definition. Later in the conversation the user chooses body for this; the old example files are kept as a bound test fixture. In this first vertical slice, a schema is a separate document format with a fixed, known meta structure. Whether Kinds and Properties should themselves become definitions with a shared basic form is now examined in section 11. The first vertical slice does not prove this new loading and bootstrap direction.

The examples use the existing Core cardinality: `maxCount: 1` means a single value, other maximum values a list; `-1` means structurally unbounded within the general compiler limits. The first vertical slice permits exactly one target Kind per reference field. A relation for several target Kinds is still to be designed; the earlier reference union question is not presented as solved.

## 5. Own types and their relations in the Connector example

| Project-defined Kind | What it describes in this vocabulary | Own relation fields |
| --- | --- | --- |
| System | Cohesive solution with a declared boundary. | containers -> Container; exchanges -> Exchange; provides -> Interface |
| Container | Runtime application and its intended location. | components -> Component |
| Component | Functional responsibility within an application. | provides/consumes -> Interface; emits/handles -> Command |
| Interface | Contract of an external or internal access. | No structured relation fields in this small excerpt. |
| MqttService | Required MQTT runtime infrastructure. | No structured relation fields in this small excerpt. |
| Channel | Parameterized address family and delivery intent. | broker -> MqttService |
| Command | Versioned operation contract with inputs and result. | No structured relation fields in this small excerpt. |
| Exchange | Connects participants, operation, request channel and reply channel. | sender/receiver -> Container; command -> Command; requestChannel/replyChannel -> Channel |

These are eight types **of this project**, not eight new Markitect primitives. C4-oriented architecture terms and special communication terms are declared through the same language mechanisms. The choice of MqttService resolves the previously visible broker question without a new universal C4 abstraction: A project may explicitly type the relevant infrastructure item. How a C4 or deployment view is derived from it is a further contract.

The directed relationship reads, in excerpts:

```text
platform:System --containers--> api:Container
api:Container --components--> http-edge:Component
http-edge:Component --emits--> list-documents-command:Command
platform:System --exchanges--> list-documents:Exchange
list-documents:Exchange --sender--> api:Container
list-documents:Exchange --receiver--> onprem:Container
list-documents:Exchange --command--> list-documents-command:Command
list-documents:Exchange --requestChannel--> requests:Channel
list-documents:Exchange --replyChannel--> replies:Channel
requests:Channel --broker--> messaging:MqttService
replies:Channel --broker--> messaging:MqttService
onprem:Container --components--> dms-handler:Component
dms-handler:Component --handles--> list-documents-command:Command
dms-handler:Component --consumes--> dms-access:Interface
dms:System --provides--> dms-access:Interface
```

The graph is derived from the reference values declared once. The reverse view, which definitions use a contract, needs no second maintained list. From a name such as containers or emits alone, the compiler recognizes neither transitive composition nor impact propagation nor an order of agent work.

## 6. Topic selection and replies as typed project statements

The example files deliberately use synthetic addresses. They do not claim a new configuration of the examined DMS project. The local source analysis and its actual topic forms are in the [detailed Connector example](../connector-example.md).

`requests:Channel` declares `bridge/{tenantId}/requests`. The body describes the authorized origin of tenantId and the shared subscription group. `replies:Channel` declares `bridge/replies/{senderInstanceId}` with the role of an issuing API instance. Both reference the same described MQTT service.

`list-documents:Exchange` connects these channels with sender api, receiver onprem and the operation contract `dms.documents.list.v1`. The individual definition sets 30 seconds per attempt and two attempts. The Kind explains once the preservation of the MessageId and the different task of the CorrelationId. The individual body describes repetition and error limits.

In this way the roles, parameterized topic selection, typed operation request and reply mapping can be expressed as a target description. The language itself needs to understand neither MQTT nor HTTP. The technical suitability, security and correct implementation of these statements must nevertheless be examined.

The vertical slice is limited to a reading document list operation. Total period, retry delay, distributed idempotency, file transfer, complete authentication and deployment as well as the UserManagement/Winsolvenz extensions remain outside this small instance. They remain necessary description questions of the larger example; they were not settled by a green structural verdict.

## 7. Which check is already generically possible?

Checkable with the existing structural principle are, among others: known and unique Kinds; known and unique definitions; closed fields; value types and cardinalities; existing, exactly typed reference targets; a deterministic graph derivation and a content digest.

Counter-probes of the draft are a non-existent receiver, a Component on a Container-only field, a missing required reply channel field and an unknown spec field. An own relation name needs no new case in the structure compiler.

Not checked by this alone are the positive range rule for periods, the valid placeholders of a topic template, the unique use of an operationId in the whole project, matching implementations of all offered Interfaces, the permitted number of structural parents and the actual runtime behaviour. For this an explicit semantic check contract is required. The first draft commits itself neither to an expression evaluator nor to executable code in vocabulary files.

**Counter-probe performed:** The two YAML files were read in and their body fields explicitly mapped from `definition` to the existing Core `purpose` fields. The remaining values were preserved; integer values were passed on as such. The unchanged current Core successfully compiled the eight Kinds, thirteen definitions and seventeen reference edges. Six invalid variants were rejected as expected: missing mandatory relation, wrong target Kind, unknown spec field, unresolvable target, unknown definition Kind and duplicate identity. Changes to the definition body, to the Kind meaning and to the relation meaning each changed the digest.

The counter-probe also confirmed the limit: `maxAttempts: 0` is structurally accepted as an integer, although the declared Exchange contract demands a positive value. The [bounded check report](structural-probe.json) binds the YAML inputs through their SHA-256 values and names the unchanged Core source state. The temporary probe changed no implementation files in the repository. It is not a new product function and not a generic validator for the proposed authoring form.

The example files are therefore still not `.markitect` models directly loadable in the current project workflow and not evidence of an executed Connector solution. Context, impact and Apply for these own Kinds were not validated by this.

## 8. Existing foundation and connections still to be designed

| Area | Present or proposed here |
| --- | --- |
| Freely named nominal Kinds and closed Properties | Present in the structural Core. |
| Typed references, cardinalities, definition graph and digest | Present in the Core; Reference binds exactly one target Kind. |
| `kindReference` | Present for referring to a type. It is neither a type declaration nor an execution instruction. |
| Body under body | Field name chosen by the user. The older example files still use definition; the current Core uses purpose at Schema/Kind/Property/Definition. The explicit mapping can preserve the content; a new product authoring form is not yet implemented. |
| Project files choose their vocabulary | Loader/selection contract still to be designed. An existing apiVersion value installs no vocabulary. |
| Own reference names | Structurally expressible in the Core as Properties. Their statement meaning comes from the vocabulary; generic impact or Apply does not yet follow from this. |
| Generic contexts, semantic checking and Apply | Must be aligned with definitions and their Kind/property contracts. The existing Manager/Statement project workflow does not achieve this automatically for arbitrary Kinds. |

A model state must firmly bind the used vocabularies and definitions together. First all selected schemas are captured, then definitions are assigned and references resolved. For the vertical slice, the two files are explicitly considered together. The selection/loading configuration is tool configuration; project content need not be a built-in Project Kind for it.

For the first version I recommend exact API versions, completely known inputs, no implicit inheritance, no silent schema extensions and no automatic plugin execution. In a later split into several vocabularies, their reference targets must be explicitly present in the bound input set. The Core can check explicit cross-schema references; convenient packaged provision and versioning/migration policy are not yet decided by this.

The explicit body links already discussed remain a possible authoring form for usage references. This vertical slice deliberately shows structured, typed relation fields. A later link form must normalize into the same traceable reference representation; duplicate manual uses lists and edges invented from mere word matches do not thereby become necessary. The exact link/qualification contract remains open.

## 9. Next decisions

The smallest coherent proposal reads: **Schema, Kind, Property and Definition with binding prose; a directed relation is initially a typed reference Property.** System, Container, Component, Rule, Command or other terms are contents of a chosen vocabulary.

In the example, the following are now particularly to be decided: Does the relation at the source Kind suffice, or do we need independent relation definitions? Which few structural conditions need declarative checking beyond the existing value types? How do context, change comparison and realization obtain the meaning of a user-defined Kind and its relationships? Only after that should the exact authoring form be fixed as a language contract.

## 10. Complete Exchange walkthrough with the minimal Core

The user's request after the decision to leave out requirements and roles for now. The basis is the complete files [vocabulary.yaml](vocabulary.yaml) and [connector.yaml](connector.yaml). Both already use exclusively Schema, Kind, Property, Definition and binding bodies. They need no conversion to the deferred structured variant. The example is a target description of a reading document list operation, not a complete production specification of the Connector platform.

### 10.1 Determining the input set and the language

For this walkthrough, the two files are explicitly loaded together. vocabulary.yaml provides the schema connector.example.org/v1alpha1 with eight project-specific Kinds. connector.yaml provides thirteen concrete definitions. No library is imported; the same type contracts could later come from a library. The language does not thereby gain other capabilities.

The shared contract reads: Canonical prose is binding. A Kind description applies to every definition of exactly this Kind. The property description explains the meaning of its values and references. Individual descriptions supplement the concrete definition; they cannot silently lift general obligations. Domain conditions and limits are described in this prose. The example needs neither requirements nor roles, subject or a general formal constraint language.

This checked vertical slice still uses definition for the body of Schema, Kind, Property and Definition. Afterwards the user chooses body for the intended authoring form. Today's Core uses purpose at these places; the bounded probe explicitly maps the fields of that time. The product loader is not changed by this demonstration.

### 10.2 Describing Exchange as a Kind

The general body of Exchange in vocabulary.yaml describes a request/reply agreement: sender publishes the operation described under command on requestChannel and receives its replies through an instance-specific address of replyChannel. receiver processes the request and replies to the communicated address. Replies preserve the MessageId of the request, which identifies the waiting exchange. CorrelationId serves cross-cutting tracing. timeoutSeconds applies per attempt; maxAttempts counts the first attempt. Both values must be positive. The definition describes the contract, not a single runtime occurrence.

These statements apply to all definitions of this Kind in the bound schema. The complete Kind declaration is in vocabulary.yaml. Its Properties are:

| Property | Structure contract | Meaning |
| --- | --- | --- |
| sender | reference to Container, exactly 1 | Application in the issuing role of this exchange. |
| receiver | reference to Container, exactly 1 | Application in the processing role of this exchange. |
| command | reference to Command, exactly 1 | Contract of the requested operation. |
| requestChannel | reference to Channel, exactly 1 | Channel of the request direction. |
| replyChannel | reference to Channel, exactly 1 | Channel for replies to the issuing instance. |
| timeoutSeconds | integer, exactly 1 | Positive maximum waiting period of an attempt in seconds. |
| maxAttempts | integer, exactly 1 | Positive attempt budget including the first attempt. |

The following real excerpt from the example file (quoted unchanged) shows a property declaration within Exchange.properties:

```yaml
sender:
  definition: Anwendung in der Rolle des ausstellenden Auftraggebers.
  type: reference
  target: {apiVersion: connector.example.org/v1alpha1, kind: Container}
  minCount: 1
  maxCount: 1
```

The name sender is project-specific. The Core checks the reference target and the count. The meaning as requester comes from the described contract. The positive value requirement of the integer Properties remains binding, although today's compiler does not check it formally.

### 10.3 Defining the referenced items

The following definitions are also in connector.yaml; there are no unresolved placeholder definitions:

| Definition | Kind | Relevant individual contract |
| --- | --- | --- |
| api | Container | Cloud application; authorized tenant context, HTTP access and instance-specific replies. A restart does not take over an earlier waiting state. |
| onprem | Container | Application in the customer environment; outgoing MQTT connection and processing only for the authorized environment. |
| list-documents-command | Command | Operation dms.documents.list.v1; FolderId as well as optional PageSize and ContinuationToken as inputs; document metadata, where applicable NextContinuationToken, or a domain error as result. The request changes no documents. |
| requests | Channel | Topic family bridge/{tenantId}/requests; tenantId from the authorized target environment; shared subscription for the matching connector pool. The operation is selected by its identifier. |
| replies | Channel | Topic family bridge/replies/{senderInstanceId}; replies to the issuing API instance, no competing shared subscription for this address. |
| messaging | MqttService | Separate broker runtime, reachable from cloud and customer environment, with authorized publication and subscription addresses. |

The two Channels reference messaging and require at-least-once as transport intent. That is no guarantee that the operation is executed exactly once in domain terms. Their topic templates are string values with described meaning; Markitect does not execute them. Runtime values such as tenantId, senderInstanceId and MessageId are not separate canonical definitions here.

System, Component and Interface definitions of the complete file give the wider architecture and access frame. For this vertical slice they are neither removed nor treated as new Core terms.

### 10.4 Binding the concrete Exchange definition

The following definition is taken unchanged from connector.yaml:

```yaml
apiVersion: connector.example.org/v1alpha1
kind: Exchange
metadata: {namespace: connector, name: list-documents}
definition: >-
  Die API bestimmt den autorisierten Mandanten, bildet den Command und
  registriert den wartenden Austausch vor der Publikation. Der Connector waehlt
  anhand der Operationskennung die Auftragsbearbeitung. Der Transporterfolg und
  der fachliche Resultstatus werden unterschieden. Wiederholungen verwenden
  dieselbe MessageId; spaete Antworten ohne Wartezustand werden verworfen.
  Dieser lesende Durchstich verlangt keine globale Exactly-once-Garantie.
  Bei schreibenden Operationen ist die erlaubte Wiederholungswirkung gesondert
  zu entscheiden. Die Versuchszahl ist ein endliches Budget; ein Timeout
  beweist keine Nichtausfuehrung. Eine Gesamtfrist und Retry-Verzoegerung sind
  in dieser minimalen Instanz noch offen.
spec:
  sender: {namespace: connector, name: api}
  receiver: {namespace: connector, name: onprem}
  command: {namespace: connector, name: list-documents-command}
  requestChannel: {namespace: connector, name: requests}
  replyChannel: {namespace: connector, name: replies}
  timeoutSeconds: 30
  maxAttempts: 2
```

Its identity is the tuple (connector.example.org/v1alpha1, Exchange, connector, list-documents). Reference values omit Kind and API version because their Properties stipulate these exactly. For example, sender is resolved exactly against (connector.example.org/v1alpha1, Container, connector, api). Another schema with the same name or whichever version happens to be newest is no substitute.

From the reference values maintained once, the compiler generates five outgoing edges of the Exchange definition. The two Channel definitions generate their own edges to the broker. A second manually maintained role, uses or edge list is not required.

### 10.5 Assembling the applicable contract

The assessment of list-documents needs at least the schema body, the Exchange body and its property contracts, the concrete values and the individual body, as well as the pertinent contracts of the referenced participants, operation and channels. The complete input set remains available in bound form. The exact generic context selection for the product is not thereby already implemented.

The Kind describes the general reply mapping. For this exchange, the concrete definition adds among other things registration before publication, reuse of the MessageId and discarding of late replies without a waiting state. The channel contracts explain how target tenant and sender instance determine the addresses. These statements are read together, without copying them into a second binding requirement list.

For the role binding, the described relationship suffices: sender is api here; receiver is onprem. A statement about sender concerns api in this exchange. It becomes neither a global obligation of all API operations nor an automatic rule of all Components. The check context must nonetheless capture the relevant implementing parts.

### 10.6 Tracing a desired runtime case

1. The API accepts an authorized HTTP request and determines the target tenant. Inputs are mapped onto the Command contract.
2. The issuing process instance sets up its reply reception and the waiting mapping. It generates or takes over the MessageId and passes along its authorized reply address according to replies.
3. The request is published via bridge/{tenantId}/requests. The operation identifier stands in the Command or in the message envelope, not in the topic chosen here.
4. A matching connector replica receives the request, selects the request processing by dms.documents.list.v1 and reads the permitted document metadata.
5. It replies with the same MessageId to bridge/replies/{senderInstanceId}. The API maps the reply to its waiting exchange and maps the domain result onto the HTTP contract.
6. Without a reply, it ends the waiting period of this attempt after 30 seconds. A further attempt is possible within the budget of at most two attempts and uses the same MessageId. A timeout does not evidence non-execution. Total period and delay are not yet stipulated.

The description stipulates this desired behaviour; it is not a sequence program of the Core compiler. A concrete runtime call needs no new canonical Exchange record. Agents, implementation, test cases and executed check evidence arise in the realization work.

### 10.7 Checking validity and implementation separately

| Counter-probe | Verdict of the existing structure compiler | Further assessment |
| --- | --- | --- |
| replyChannel is missing | Rejected: mandatory Property missing. | No domain substitute assumption. |
| sender names a different Kind | Rejected: wrong target Kind. | No interpretation based on similar names. |
| A target does not exist | Rejected: unresolvable reference. | Correct the missing definition or reference. |
| maxAttempts is 0 | Structurally accepted integer. | Violates the described positive value range; model review must establish this. |
| replyChannel points to requests | Structurally accepted Channel reference. | The suitability for the instance-specific reply contract is to be checked in domain terms; a matching target type alone is not enough. |
| Implementation registers only after publication | Not determinable from the model structure. | Implementation review and a suitable case with an immediate reply can evidence the violation. |
| Reply contains a different MessageId | Not determinable from the model structure. | Mapping and integration check against the fixed implementation candidate. |

**Probe executed again:** Inputs freshly adapted from the two YAML files were compiled against the unchanged Core. Result: eight Kinds, thirteen definitions, seventeen reference edges; five outgoing edges at list-documents. The six negative structural variants were rejected again. The two semantic borderline cases maxAttempts=0 and replyChannel=requests were structurally accepted. Body, Kind and Property meaning remained contained in the digest. The [bound walkthrough report](exchange-walkthrough-probe.json) names input hashes, source state, adaptation and normalized Exchange references. The absence of the special requirement fields was checked on the structured inputs.

This is not an executed Connector integration and not proof of already generic context, impact or Apply. An agent can work out a checklist and matching evidence against the complete bound contract. Such a list is a derived work result; its completeness is to be assessed, not guaranteed by its format. Results must refer to the fixed model and candidate state. Unchecked obligations remain open.

### 10.8 Following a change through

If timeoutSeconds is changed from 30 to 15, the edges stay the same, but the concrete waiting obligation and the model digest change. If the general reply mapping in the Exchange Kind is changed, the applicable contract of its definitions changes even with unchanged instance files. A change to replies.topicPattern, by contrast, concerns a referenced contract; incoming edges make its direct uses discoverable.

For these cases the Core provides identities, known references and the bound content. It does not automatically determine from this the complete domain change effect or a division of agents. As long as there is no narrower reliable rule, a semantic assessment must not cut off relevant obligations through too narrow a context.

### 10.9 What this walkthrough clarifies and what remains open

As its language foundation, the example needs four declaration building blocks: Schema, Kind, Property and Definition. Binding prose, identity, typed values, references and shared model binding carry the described contract. Exchange, Container, Command and Channel remain contents of this project vocabulary.

What remains open is the exact general body link syntax, further formal value conditions, loader/import convenience as well as generic context, impact and realization procedures. In domain terms, this small vertical slice is deliberately limited to a reading exchange. The still open questions on total period, retry delay, repetition effect of writing operations and further operating conditions can be described through additional prose, Properties or own Kinds; their solution is not claimed by a successful structural verdict.

## 11. Shared basic form and body

**User decision:** The binding body should be called body. The choice applies to the intended authoring form and avoids the double use of Definition as item and as text field. Renaming the concrete model item to Element was previously a recommendation of the assistant, not a language decision accepted by the user.

**Investigation assignment:** Can Kind, Property and concrete elements have the same basic form, so that Kinds are themselves definitions? The following variant is a discussion proposal. It replaces neither today's Core data structures nor the evidenced compiler function with a claim of already existing self-description.

### 11.1 Distinguishing two sorts of unification

A shared document envelope can unify identity, type reference, body and structured values. Going further, Kinds and Properties can themselves become referenceable definitions: Exchange is a definition of the built-in Kind Kind, sender is a definition of the built-in Kind Property, and list-documents is a definition of Exchange.

Both variants share a representation. Only the second makes the type contracts ordinarily addressable nodes of the model. It can reduce special forms for referring to Kinds and for representing their changes. The meaning of type declaration, property contract and concrete use remains different in this.

### 11.2 A shared envelope at the Exchange

The proposed basic form reads: identity, kind as a reference to the type definition, binding body and spec as values according to this type contract. The following IDs such as core.Kind and connector.Exchange are **symbolic shorthand notations for the discussion**, not an already stipulated identity grammar. The final form must preserve namespace, version and unambiguous resolution. In particular, today's identity from apiVersion, kind, namespace and name is not silently reused here: in the candidate, the identity of a type definition is to be distinguished from its own Kind, and the type reference is to be directed explicitly at it.

A deliberately reduced excerpt with sender and timeoutSeconds:

```yaml
id: connector.Exchange
kind: core.Kind
body: >-
  Describes a request/reply contract. sender denotes the
  requesting application. Replies keep the MessageId of the request.
  timeoutSeconds applies as the waiting period per attempt.
spec:
  properties:
    - connector.Exchange.sender
    - connector.Exchange.timeoutSeconds
---
id: connector.Exchange.sender
kind: core.Property
body: Application in the requester role of this exchange.
spec:
  name: sender
  type: reference
  target: connector.Container
  minCount: 1
  maxCount: 1
---
id: connector.Exchange.timeoutSeconds
kind: core.Property
body: Positive maximum waiting period per attempt in seconds.
spec:
  name: timeoutSeconds
  type: integer
  minCount: 1
  maxCount: 1
---
id: connector.list-documents
kind: connector.Exchange
body: >-
  The API requests document metadata from the Connector. It registers
  the waiting exchange before the publication.
spec:
  sender: connector.api
  timeoutSeconds: 30
```

The excerpt is not a complete new model record. core.Kind, core.Property, connector.Container and connector.api must be provided with their remaining contracts; receiver, command, channels and number of attempts of the full example are left out here only to illustrate the envelope. No spec shown is executed. The MessageId remains a described runtime quantity.

| Definition | Its Kind | Interpretation |
| --- | --- | --- |
| connector.Exchange | core.Kind | Declares a type for further definitions. |
| connector.Exchange.sender | core.Property | Declares a named, typed item of information of this type. |
| connector.list-documents | connector.Exchange | Describes a concrete desired exchange. |

A Property gets its place in Exchange through Exchange's properties reference. Its spec.name stipulates the value key in concrete Exchanges; the compiler should not guess it from the last segment of its ID. Duplicate names in a Kind and wrong property targets must be rejected. A sender of the same name in another Kind thereby does not have an automatically equal meaning.

### 11.3 The fixed beginning does not disappear

The Core must bring along a known, versioned basic definition of core.Kind and core.Property. In the self-describing candidate, core.Property has the Kind core.Kind; core.Kind can itself have the Kind core.Kind. This one known self-binding is a stipulated beginning, not a request to load an infinite sequence of meta-Kinds. The basic definitions are provided together and recognized by the compiler with a fixed meaning. Projects can neither replace them with local definitions of the same name nor install new compiler operators through prose.

The self-description documents the meta contract but does not prove it from itself. In particular, the compiler must still know: how a type declaration determines the permitted Properties, which value types exist, how cardinalities apply, how a Reference is bound to a target and how body enters model binding and subsequent assessment.

A possible narrow loading contract reads:

1. Capture the fixed, versioned Core contract and the complete input set; check identities for uniqueness.
2. Check and resolve the types declared as core.Kind and their core.Property contracts with the known meta rules.
3. Check the remaining definitions against their now known Kinds; resolve all references and bind the shared model state.

A kind reference must point to a valid type definition, not to an arbitrary existing definition. Free user-defined metaclasses or an arbitrary chain of changing type interpretations are not needed for this candidate. All needed type contracts must be in the bound input set; file order must not determine their availability.

### 11.4 Simplicity gained and new costs

| Question | Shared definitions | Remaining need / price |
| --- | --- | --- |
| Document form | One envelope for type, Property and concrete item. | spec continues to have different meaning and structure depending on the Kind. |
| References | Kinds and Properties can be referenced with the same basic mechanism as other definitions. | A type reference and an ordinary domain reference continue to have different roles. A replacement for the previous kindReference must be stipulated explicitly. |
| Change graph | Kind and property definitions are addressable nodes; uses and type references can become visible together. | The complete domain change effect is not yet determined by this. Today's Core does not represent Kind references as definition edges. |
| Number of basic forms | One general definition instead of separate document forms. | Kind and Property remain two meta-Kinds understood by the compiler; value types, binding and cardinality remain fixed semantics. |
| Properties as definitions of their own | Own identity, referenceability, body and provenance per Property. | More declarations and references for small contracts; versioning and assignment must be unambiguous. |
| Schema / library | Vocabularies can be provided as versioned collections of definitions. | Selection, namespace, exact version binding and closed type contracts remain necessary. Whether a schema itself becomes a Kind is still open. |

A later embedded short form for Properties would be possible if it normalizes exactly onto the same addressable declarations and invents no second meaning. For the present discussion it is not additionally stipulated. A uniform envelope alone does not force moving every small Property out into a file of its own; several definitions can stand in one document.

### 11.5 Recommendation and required counter-probes

**Recommendation for further checking:** Examine a definition as the general basic form; represent Kind and Property within it as the type definitions understood by the Core. The concrete project description is then a definition of a self-defined Kind. body is in all cases the binding body. The previous term Element can serve to explain concrete items, without creating an additional built-in Kind.

This reduces the number of representation forms, not the number of all required language rules. The next counter-probes are a complete fixed Core beginning, duplicate or contradictory property declarations, wrong kind targets, exact cross-vocabulary versions as well as changes to type and property meaning with unchanged concrete definitions. It must be checked whether the uniform references simplify context and change assessment instead of merely rearranging YAML.

The structural probes so far check separate schemas and definitions with explicit body mapping. They evidence neither the new id/kind envelope nor the self-describing beginning nor a generic meta compiler. In this step only the draft and the chosen field name were documented.

## 12. Fixed beginning and type resolution at the Exchange

The user's request: Play through the fixed beginning with Kind and Property as well as the complete type resolution at the Exchange, and make visible the Core rules that are really needed as fixed. The following files together form a closed structural input set for this candidate:

- [bootstrap/core.yaml](bootstrap/core.yaml): ten fixed Core definitions with body.
- [bootstrap/connector.yaml](bootstrap/connector.yaml): eight project Kinds, twenty project Properties and thirteen concrete items.
- [bootstrap/probe.json](bootstrap/probe.json): results of the bounded independent design probe, input hashes and selected resolved edges.

The new form is a proposal. The probe is not an invocation of the existing Go Core compiler, not a new product function and not an executed Connector solution. It examines exclusively the finite beginning, the type/property resolution, structure check and shared content binding. body is the authoring name already chosen. requirements and roles remain left out.

### 12.1 The fixed document envelope

Every definition has exactly id, kind, body and spec. id is a unique, non-empty identity, kind an exact reference to a type definition, body non-empty binding prose and spec an object with the values permitted according to the type. A duplicated id is invalid.

The probe uses IDs such as core/v1/Kind, connector/v1/Exchange and project/v1/list-documents as exact strings. The version parts are stated explicitly; there is neither a short-name search nor an automatic choice of the newest version. This readable notation is still a candidate for the identity form. The values are not guessed from file names or from the last ID segments. In the probe, core/ denotes the reserved basic contract; a project must not replace or add any definition there.

### 12.2 Two meta-Kinds and their eight own Properties

The fixed beginning comprises not only two names. It must also provide their contract structure. In compact form:

```yaml
id: core/v1/Kind
kind: core/v1/Kind
body: >-
  Declares a type. properties names its permitted entries.
  Its body describes the common meaning of its definitions.
spec:
  properties:
    - core/v1/Kind/properties
---
id: core/v1/Property
kind: core/v1/Kind
body: Declares a named attribute or relation of a type contract.
spec:
  properties:
    - core/v1/Property/name
    - core/v1/Property/type
    - core/v1/Property/minCount
    - core/v1/Property/maxCount
    - core/v1/Property/values
    - core/v1/Property/target
    - core/v1/Property/properties
```

All eight referenced Properties are themselves definitions of core/v1/Property. Their contracts are:

| Property | Value type | Count | Stipulated task |
| --- | --- | --- | --- |
| Kind.properties | reference to Property | 0 to any number | Links a type to its permitted property contracts. |
| Property.name | string | exactly 1 | Names the value key in a concrete spec. |
| Property.type | enum | exactly 1 | Chooses a known value type. |
| Property.minCount | integer | exactly 1 | Determines the minimum number of values. |
| Property.maxCount | integer | exactly 1 | Determines the maximum number and the single-value/list form. |
| Property.values | string | 0 to any number | Stipulates the permitted values of an enum. |
| Property.target | reference to Kind | 0 or 1 | Stipulates the target Kind of a reference. |
| Property.properties | reference to Property | 0 to any number | Describes the fields of an object. |

The ten fixed definitions thus consist of two type definitions and eight property definitions. This is a complete beginning for the range of values examined here, not a proof of the smallest possible number of all necessary declarations.

### 12.3 The decisive cyclic definition

The complete fixed data set contains in particular:

```yaml
id: core/v1/Kind/properties
kind: core/v1/Property
body: The property contracts of this Kind, with names that are unique within the Kind.
spec:
  name: properties
  type: reference
  minCount: 0
  maxCount: -1
  target: core/v1/Property
---
id: core/v1/Property/target
kind: core/v1/Property
body: Required only for reference; determines the exact target Kind.
spec:
  name: target
  type: reference
  minCount: 0
  maxCount: 1
  target: core/v1/Kind
```

Kind.properties points to Properties; Property.target points to Kinds; Kind itself has the Kind Kind. This is a finite reference graph. Reference checking here means comparing identity and declared target Kind. It does not mean recursively unfolding the entire target contract anew for every reference. Therefore this beginning needs no Kind of the Kind of the Kind in an infinite sequence.

The compiler nevertheless does not start with unknown YAML and believe its self-description. It starts with the meta contracts shipped as fixed with the chosen Core version, checks the Core definitions with them and afterwards verifies that the contracts derived from these correspond exactly to this beginning. For this, the probe uses two fixed meta contract tables. The Core seed is an explicitly trusted provided input of the probe; the probe implements no package installation, signature check or production loader hardening.

### 12.4 Connecting the Exchange vocabulary

In the complete Connector data set, Exchange is a definition of core/v1/Kind. Its properties list references the seven definitions sender, receiver, command, requestChannel, replyChannel, timeoutSeconds and maxAttempts. Their bodies describe their meaning; their specs declare the structure contract.

For example:

```yaml
id: connector/v1/Exchange/sender
kind: core/v1/Property
body: Application in the role of the issuing requester.
spec:
  name: sender
  type: reference
  target: connector/v1/Container
  minCount: 1
  maxCount: 1
```

The type definition connector/v1/Container has the Kind core/v1/Kind. The concrete API, by contrast, has the Kind connector/v1/Container. This difference is decisive: sender.target may reference Container, not the concrete API.

The concrete definition project/v1/list-documents has the Kind connector/v1/Exchange. It sets sender to project/v1/api, receiver to project/v1/onprem, command to project/v1/list-documents-command, requestChannel to project/v1/requests and replyChannel to project/v1/replies. Its values timeoutSeconds=30 and maxAttempts=2 remain unchanged. The individual body describes, among other things, registration before publication, retries with the same MessageId and the discarding of late replies without a waiting state.

The new file takes over the eight type bodies, twenty property bodies and thirteen individual bodies from the previous example and represents their references explicitly. It is not evidence of a lossless general migration of the old schema format; in particular, the general package/schema envelope remains a separate design question.

### 12.5 Type resolution without dependence on file order

1. **Choose the basic contract.** Provide the fixed shipped Core contract for core/v1. A project import cannot override it.
2. **Capture identities.** Index all Core and project definitions together. Check envelope, uniqueness and reserved IDs. The order of the documents is irrelevant.
3. **Check the type reference.** Every kind reference must point to an existing definition whose own kind is exactly core/v1/Kind. The fixed self-binding of core/v1/Kind is valid. Arbitrary self-typed project definitions are not thereby permitted.
4. **Check meta contracts.** Check definitions of Kind and Property against the fixed beginning. Then check their additional language invariants, such as valid cardinalities and the obligation of target for reference.
5. **Build type contracts.** For every Kind, collect the referenced property definitions by spec.name. Duplicate field names are errors. Object contracts are built in a bounded way; cyclic object field contracts that would have to be unfolded infinitely are invalid in this candidate. Ordinary reference cycles are different from this.
6. **Check all values.** Check every definition, including the Core definitions, against its Kind as thus known. Reject unknown fields, wrong value forms, missing mandatory values or wrong reference targets.
7. **Bind the model.** Normalize all definitions including bodies and structured contracts, record type and value references as distinguishable edges and form a shared content digest.

For Exchange.sender, the following concrete check chain results:

```text
list-documents.kind -> Exchange
Exchange.kind -> core.Kind
Exchange.spec.properties -> Exchange.sender
Exchange.sender.kind -> core.Property
Exchange.sender.spec.target -> Container
Container.kind -> core.Kind
list-documents.spec.sender -> api
api.kind -> Container
```

This formally checks that sender denotes an existing Container definition. The statement that this application is suitable in domain terms as a sender and handles messages correctly does not follow from this chain.

### 12.6 Which rules really stand fixed in the Core

| Fixed rule | Why it does not follow from the body alone |
| --- | --- |
| Envelope and identity | The compiler must know which fields carry identity, type reference, body and structured values. |
| Meaning of the two meta-Kinds | Only the fixed-bound Kind/Property identities produce type and field contracts. A project Kind with the name Kind has no special rights. |
| Connection of type and Properties | spec.properties of a Kind and spec.name of a Property determine its permitted value keys; ID segments do not. |
| Value type semantics | string, boolean, integer, number, enum, object and reference need implemented check rules. A new enum text installs no new operator. |
| Cardinality semantics | minCount is non-negative; maxCount is -1 or at least minCount. maxCount=1 means a single value, otherwise a list. Null and unknown fields are invalid in the candidate. |
| Conditional meta rules | reference requires target; enum requires unique permitted values; object uses its field contracts. Unsuitable additional fields are errors. These relationships are not expressed in the seed by mere integer or string. |
| Exact reference check | A target must exist and its declared Kind must match the property contract. No implicit inheritance, version completion or search heuristics. |
| Binding and general applicability of bodies | Type and property bodies belong to the contract of their uses; individual statements supplement them. Contradictions are not resolved by silent overriding. The digest must cover the entire bodies. |
| Finiteness and limits | Processing must terminate with fixed size/depth limits. The prototype is not a hardened production implementation of all input limits. |

In particular, the range rule for Property.minCount is a language invariant understood as fixed. No automatic knowledge of the positive value range of Exchange.maxAttempts follows from this. The latter continues to stand in domain prose and needs semantic assessment, as long as no general value constraint is introduced.

A separate kindReference value type is not necessary in the candidate: A reference to a type is a normal reference with target core/v1/Kind. Today's product Core still has its previous kindReference type; its migration is not implemented here.

### 12.7 Design probe actually performed

A temporary independent Python prototype loaded the fixed provided Core contract and the new Connector file. The basis was structurally accepted: ten fixed Core definitions plus forty-one project definitions yield fifty-one nodes, among them ten Kinds and twenty-eight Properties. The derived graph has one hundred and thirteen edges, including kind references and the new meta relationships. It therefore has a different scope from the seventeen purely domain reference edges of the earlier structural example.

Sixteen invalid variants were rejected in a targeted way, among them: Core override, duplicate identity, unknown Kind, concrete element as kind target, concrete element in a property list, duplicate property names, concrete element as Property.target, missing target, invalid cardinality, unknown value type, unknown formal constraint, missing mandatory Property, wrong reference Kind, missing exact version, unknown instance field and arbitrary self-typing. The error codes are in the linked report.

For this record, the probe also confirms the self-consistency of the fixed beginning, order independence and the inclusion of changed Kind, Property and individual bodies in the digest. An additional ordinary project Kind named Kind receives no special language meaning. maxAttempts=0 and replyChannel=requests, by contrast, remain structurally accepted and require the same domain assessment as before.

The report binds the two input files and the temporary prototype through SHA-256 and names the repository source state as context. It is evidence for this bounded design probe, not a result of today's Go Core compiler or a universal language proof. The new probe checks no independent semantic interpretation of prose, no production loader, no generic delegation, no complete impact and no Connector runtime.

### 12.8 Assessment of this candidate

The shared basic form can carry the fixed beginning and the complete structured Exchange example. The type and property references are now in the same addressable graph as the concrete items. For this, the Core must still understand as fixed two meta-Kinds, the interpretation of their fields, value types, binding and cardinalities.

The gain is a shared definition form and a shared reference mechanism. The fixed rules are visible and traceable through the self-description, but have not disappeared from the compiler. Whether this is simpler overall for authors and the subsequent Markitect processing than separate schema/definition forms remains the next assessment. No general sufficiency is derived here from a successful structure check alone.

## 13. Comparison and recommendation for Markitect

The user's request: Compare the shared basic form with the previous Core, name real savings and new complexity and recommend a variant. Assessed are the existing structural Core and the bounded self-describing design probe. The currently firmly accepted decisions body and the provisional omission of requirements and roles apply independently of the choice of variant.

**Subsequent user decision:** The uniform basic form with Kind and Property is chosen. The recommendation in 13.4 is thereby superseded; the factual trade-offs remain the derivation. The current direction and the compact property form are in section 14.

### 13.1 What is not saved

Kind remains a type contract, Property remains a field/relation declaration and a concrete item remains a use of this contract. The shared envelope can express these roles but does not abolish them. Both variants need value types, cardinalities, exact reference binding, naming/version rules as well as binding prose and suitable semantic assessment.

Freely definable project vocabulary is already possible in the previous Core. Neither library imports nor a binding body nor the inclusion of a Kind change presuppose a self-typed Kind. The existing digest already binds the Kind and property meaning. The previous product workflow with fixed Manager/Statement vocabulary is a further layer and must not be confused with the expressiveness of the structural Core.

### 13.2 What the shared basic form really gains

| Area | Previous explicit Core | Self-describing candidate | Actual difference |
| --- | --- | --- | --- |
| Document structure | Schema with Kind/property contracts; concrete definitions next to it. | One envelope of id, kind, body and spec. | A uniform record form. |
| Kind and concrete items | Different addressable contract types; kindReference as a separate value type. | Kinds are definitions of the fixed meta-Kind Kind. | Type targets can be addressed with the general reference mechanism. The special type check remains. |
| Properties | Embedded in the type contract and determined by their name or path. | Properties have explicit definition identities and are referenced by the Kind. | First-class property nodes for display, provenance and references. |
| Shared property contract | In the previous Core, entries of the same sort stand as embedded declarations at their respective Kinds; reuse needs authoring/loader help or an extension. | Several Kinds can explicitly reference the same property definition. | A jointly maintained field contract becomes structurally directly possible. Fields of the same name are not thereby automatically equal; the previous Exchange data set does not need this reuse. |
| Model graph | Today's Core provides edges between concrete definitions; type contracts lie next to them. | The prototype also represents kind references and meta relationships as edges. | A shared graph becomes directly visible. Such a view can also be derived from the explicit Core. |
| Schema | Declares an exact versioned type contract. | Can be treated as a package/import collection outside the uniform definition form. | The special schema document can be dropped; grouping, versioning and selection remain required. |
| Identity | The concrete identity contains the Kind. | The candidate separates id from the kind reference. | A separate identity decision; it is not logically bound to self-description. |

A shared envelope and storage in the same graph are real unifications. The claim of having reduced four necessary semantic concepts to one would, by contrast, be too strong. One basic format still contains two fixed-interpreted meta-Kinds and concrete uses. Likewise, a dropped kindReference is not semantically removed entirely without replacement, because of the still required distinction between type target and concrete target.

### 13.3 What complexity is added

The explicit Core starts with known data structures and checks supplied type contracts. The self-describing candidate additionally needs a fixed versioned beginning, its basic definitions and the reconciliation between their declarative representation and the implemented meta rules. Kind.kind=Kind can be handled in a bounded way, but must be an explicit language rule and understandable for tools.

With a complete unification, the following tasks are added as well:

- Core identities and the supplied basic contract must be distinguishable from project definitions and protected against overriding.
- kind may only reference valid type definitions. Arbitrary existing nodes or freely self-typed project nodes are not sufficient.
- Properties get their own identities and an additional entry in the property list of their Kind. name and id have different tasks.
- A small type contract is split into several records and references. An embedded authoring short form would in turn need an explicit normalization contract.
- Packaging and versioning must be clarified for the new identity form. The previous four-part definition identities are not automatically absorbed into it.

The project domain contents do not become fewer through this: The old vertical slice describes eight Kinds, twenty Properties and thirteen concrete items. The new file describes the same forty-one project constituents now as explicit definitions and adds ten fixed Core definitions. Directly comparing thirteen old definitions with fifty-one new nodes would be misleading; the old embedded contracts must not disappear. Equally, the new node or edge count does not by itself evidence an increase in runtime or context cost. No measurement was performed for this.

### 13.4 Recommendation

**Assistant recommendation given at the time, since superseded by the user's choice:** For Markitect, initially develop the explicit structural Core further. Kind and property contracts remain directly declared type structure; concrete items use these types. All bodies are called body in the intended authoring form. requirements and roles receive no special processing, as decided.

The relevant gain of the new investigation should be preserved: **Kind, property and concrete definitions should be jointly addressable and visible in the model graph.** For this, existing schema/Kind identities and unique property paths can be used or suitably made more precise. Type references and the assignment of the Properties are derived from them. Authors need not additionally maintain every Property as an independent instance of core.Property. Such a graph view and its exact addressing are a planned extension, not an already existing function of today's definition graph.

This separates a uniform view for tools and context formation from a complete self-description of the language. A dedicated reference to a Kind can remain as kindReference in the explicit language contract for now. References to property contracts and body links may need a general symbol address contract; its exact syntax is still open. Neither this addressing nor a new independent element identity is presupposed here incidentally as accepted.

Decisive for the recommendation is that the description needs found so far do not yet compellingly require a complete treatment of type and property declarations as authoring records of the same sort. The fixed self-describing beginning mainly creates uniformity of representation and directly referenceable, possibly jointly usable property contracts. The work on complete applicable contexts, semantic change assessment and independent realization checking remains in both variants. A smaller number of visible document forms alone therefore does not justify the additional meta contract for now. This is a justified design decision, not a measurement of the later implementation or maintenance effort.

The shared definition form remains a serious alternative if the direct editing and linking of type and property definitions turns out to be a central work case and a derived shared view does not suffice for it. The previous prototype proves neither this added value nor the general superiority of one variant. No implementation was converted to the recommended path.

## 14. Chosen basic form and compact Properties

**User decision:** As its new design direction, Markitect uses the uniform basic form with Kind and Property. A type definition, a property definition and the description of a concrete project item are definitions with identity, kind reference, binding body and structured values. Kind and Property are the basic types understood as fixed. Project terms are defined as further Kinds. The choice of body and the provisional omission of requirements and roles remain in place.

**Additional request:** Represent Properties compactly directly at the Kind. The following authoring form and its normalization are a concrete proposal for this assignment. The product compiler and its present authoring format were not changed.

### 14.1 An embedded Property is a short form of its definition

Under spec.properties of a Kind, instead of a list of written-out property references, there can be a mapping from the field name to embedded property contracts. An embedded Property contains its body and its contract fields directly. id, kind and name are supplied unambiguously from the location; they are not maintained a second time.

A shortened Exchange excerpt:

```yaml
id: connector/v1/Exchange
kind: core/v1/Kind
body: >-
  Describes a request/reply contract between sender and receiver.
  Replies keep the MessageId of the request.
  timeoutSeconds applies as the waiting period per attempt.
spec:
  properties:
    sender:
      body: Application in the requester role of this exchange.
      type: reference
      target: connector/v1/Container
      minCount: 1
      maxCount: 1
    receiver:
      body: Application in the handler role of this exchange.
      type: reference
      target: connector/v1/Container
      minCount: 1
      maxCount: 1
    timeoutSeconds:
      body: Positive maximum waiting period per attempt in seconds.
      type: integer
      minCount: 1
      maxCount: 1
```

The excerpt reduces the contract only for explanation. The [complete compact Connector file](bootstrap/connector-compact.yaml) contains all eight Kinds and thirteen concrete items with all seven Exchange Properties and a total of twenty embedded property definitions.

Before the type check, the sender excerpt becomes exactly the following written-out definition:

```yaml
id: connector/v1/Exchange/sender
kind: core/v1/Property
body: Application in the requester role of this exchange.
spec:
  name: sender
  type: reference
  target: connector/v1/Container
  minCount: 1
  maxCount: 1
```

At the written-out Exchange, the reference connector/v1/Exchange/sender then stands under spec.properties, and correspondingly for all further Properties. The model still contains the same property definition with its own identity and meaning. There are no two canonical descriptions to be maintained in parallel; the written-out form is the derived result of the compact input.

### 14.2 Unambiguous normalization rules

1. The short form applies in the known meta contexts Kind.spec.properties and object-Property.spec.properties. An arbitrary domain field named properties does not activate it.
2. The mapping key is the property name. An embedded definition gets its ID from the exact owner ID, a separator / and this name; kind is core/v1/Property. spec.name is set from the key.
3. body is taken over unchanged into the property definition. type, target, minCount, maxCount as well as, where applicable, values and properties are taken over into its spec. No additional cardinality defaults are introduced.
4. The embedded form must not state id, kind and name again in a contradictory way. Unknown fields are errors. The prototype requires a non-empty name without /; this makes the separator unambiguous for the derivation. The final identity grammar remains to be specified.
5. The generated definitions are indexed and type-checked together with the remaining input set. A collision with another generated or written-out ID is an error, even with equal content. Duplicate YAML keys are already rejected when reading in.
6. The previous written-out reference list remains possible. As compact reuse, a mapping value can directly be a property ID; its name must agree with the mapping key. An alias that silently renames the Property is not provided for.
7. For an object, its fields can in turn be declared compactly. Their IDs are derived relative to the enclosing Property, for example Kind/envelope/messageId. The known depth limits continue to apply.

For a later production implementation, provenance must be preserved: An error at the generated sender Property should point to the corresponding embedded entry. For this, the normalization report documents owner, field key and generated ID; it is not finished product diagnostics with file lines.

### 14.3 Referenceability and renaming

The subsequent identity draft in section 15 additionally examines an optional stable local id for embedded Properties. This is an explicit further development of the short syntax checked here, which still rejects inline id. The previous probes remain unchanged and do not evidence this new exception; the new bounded probe in section 16 examines it separately.

An embedded Property remains referenceable through its derived ID. It can be reused by a further Kind without a copy if that Kind's field name stays the same. In the compact form, such an excerpt reads:

```yaml
spec:
  properties:
    sender: connector/v1/Exchange/sender
```

Reusing a shared contract is an explicit authoring decision, not an automatic equating of all fields of the same name. For a Property that is maintained jointly in the long term, its written-out definition can stand at its own stable ID. Moving or renaming an embedded Property, by contrast, changes its derived ID and must be treated as a model change for existing references. There is no silent lookup via old names.

The concrete definition list-documents does not change through the short form. It still references the Kind Exchange and sets sender, receiver, command, channels and waiting values. It needs to repeat neither the property definitions nor their bodies.

### 14.4 Bounded probe executed

A temporary normalization prototype read in the compact file, wrote out its twenty embedded Properties and compared the result with the previous written-out Connector file. After normalization, the twenty-one directly written definitions yield the same forty-one project definitions. Together with the unchanged fixed beginning, there remain fifty-one definitions and one hundred and thirteen edges.

**Result:** Definition contents, model digest and edges are identical between written-out and compact input. Nine invalid authoring variants were rejected: additional inline id, kind or name, missing body, invalid name, invalid form, collision of a generated ID, contradictory name in property reuse and duplicate YAML keys. Reuse without property copy, nested object fields and an unchanged domain data field named properties were also cross-checked.

The [compact check report](bootstrap/compact-probe.json) binds the three input files and both temporary prototypes through SHA-256. The old [bootstrap probe](bootstrap/probe.json) and its input data remain unchanged. This new probe evidences the bounded normalization against the same separate design compiler. It is not already implemented product support and checks neither semantic prose nor the Connector runtime.

## 15. Identities, imports, versions and renamings

**Subsequent user decision:** The identity contract drafted here and played through in section 16 is chosen. Its current consolidated version is in the [language contract](language-contract.md#names). This section remains the earlier elaboration; details still open at the lexical and production level are not treated as implemented because of the choice.

The user's request: Address Kinds, concrete definitions and embedded Properties unambiguously, also with imports, versions and renamings. The following identity contract is a coherent proposal. The uniform basic form, Kind and Property, body and the compact representation remain chosen; the new identity notation, an optional title and the subsequent import policy are not treated as already accepted by this draft.

### 15.1 Three different questions

| Question | Proposed answer |
| --- | --- |
| Which item is meant? | Stable logical identity from package ID and definition-local id. The Kind is not part of this identity. |
| What is it called for readers? | Optional display name title; otherwise the id can be displayed. The display name is not a reference target. |
| Which version currently applies? | The bound model state stipulates the exact package versions and contents. |

Example of a logical identity: (example/connector-project, Exchange). It can persist across several model states. The address of a particular version additionally needs its model or package binding. A reference to a lasting identity alone does not promise unchanged meaning across version changes.

IDs may be readable. An id once assigned, however, is not adjusted with every linguistic renaming and is not deliberately reused for a different item. Universally prescribed UUIDs are not required for this proposal. Whether a change of content develops the same thing further or creates a new item is partly a domain decision, not a purely syntactic check.

### 15.2 Package and import context

A package supplies a cohesive set of definitions with a stable package ID. It can contain project definitions or library vocabulary. Package and import information are the selection/loading envelope around the uniform definitions, not new built-in domain Kinds.

An exemplary project header could read:

```yaml
package: example/connector-project
imports:
  core:
    package: markitect/core
    version: "1.0.0"
  architecture:
    package: markitect/architecture
    version: "1.2.0"
```

These package names and version values are design examples, not a claim of available published libraries. core and architecture are only local import aliases. They do not belong to the lasting identity of the imported definitions. The IDs used in the prototype such as connector/v1/Exchange remain the evidenced previous state; the separation of package, local identity and version binding is a proposed further development.

The project declares its own definitions in example/connector-project. Within a library, that library's own package and import context applies. Import aliases of the project are not carried into libraries. Transitive dependencies can be loaded for the resolution of an imported contract; authors do not thereby obtain an implicit global name search across all dependencies. For an own qualified authoring reference, the import concerned is named explicitly.

The compiler must recognize the Core understood as fixed by its actually bound package/definition identity. An alias core or a local Kind with the name Kind confers no special meaning. The fixed Core contract must still not be replaced by a project import.

### 15.3 One reference notation for all targets

| Authoring reference | Resolution in the project example |
| --- | --- |
| Exchange | Local definition (example/connector-project, Exchange). |
| list-documents | Local definition (example/connector-project, list-documents). |
| Exchange/sender | Local property definition (example/connector-project, Exchange/sender). |
| architecture:Container | Definition (markitect/architecture, Container) from the specified import. |
| core:Kind | Fixed meta-Kind (markitect/core, Kind) from the bound Core. |

The proposed short contract is: an unqualified reference searches exclusively in its own package; alias:id searches exclusively in the named import. Missing targets, unknown aliases, duplicate local IDs or duplicate import aliases are errors. There is no search by display names, no selection by the expected target type, no file path resolution and no substitution by a similar name.

Local IDs are case-sensitive and are compared exactly. The separator : is reserved for import qualification; empty constituents and path segments such as . or .. are not provided for in this short notation. / can be used as an explicit separator of sub-identities, but is not a file system path. A complete grammar and escape/Unicode policy must be stipulated before an implementation; the examples here force no silent normalizations.

After authoring resolution, the canonical package/definition identity stands in the model. For tools it can be represented, for example, in structured form as {package: markitect/architecture, id: Container}. A further long authoring format is not thereby necessarily introduced yet. All subsequent type checks use the bound target identities, not the local alias strings.

kind, Property.target and concrete reference values use the same address mechanism. Their type requirements continue to differ: kind and a Property.target must point to valid Kinds; Exchange.sender requires a Container definition. An explicit body link should later use the same target resolution. Its exact markup syntax is still open, and a normal word match in the body does not become a canonical reference.

### 15.4 Binding versions to the model state

**Proposal for the first scope:** Per package ID, exactly one version is permitted in a bound model. Every library import names an exact version. No version ranges, no latest and no selection of the first package found. If two direct or transitive imports need different versions for the same package ID, a conflict is reported.

The chosen version is bound together with a digest of the concrete package content. A version label alone is not sufficient as evidence of content. The same package ID and version with deviating bytes must not silently replace an already bound content. The selection or lock information must also make the transitive dependency closure unambiguous. The exact file/installation form of this information remains open; the structure compiler should not run a network or package installation program for this.

The local project needs no new release number for every uncommitted change. Its fixed inputs are bound by the respective source/model state. The shared model state captures project content, used Core/library contracts and the selected package bindings. For this, references need not recursively insert the model digest that is still to be computed into their own contents.

A package update, for example, lets the logical identity Container persist but selects a new contract. After that, affected definitions and their realizations are considered against the new bound state. The version number alone proves no domain compatibility. For the assessment of evidence, the concrete model binding is needed, not merely the lasting definition ID.

The restriction to one version per package is a deliberate simplification of this proposal, not a fundamental impossibility of parallel versions. A later multi-version solution would need explicit version/instance qualification and type comparisons within exactly these bindings. In the first scope, two aliases to different versions must not circumvent this rule. Describing different domain software versions as project items remains unaffected by this.

### 15.5 Exchange with stable identity and readable name

Within the package above, a Kind can be declared as follows:

```yaml
id: Exchange
title: Request/reply contract
kind: core:Kind
body: >-
  Describes the exchange of a request and its reply.
  The receiver preserves the MessageId in the reply.
spec:
  properties:
    sender:
      body: Application in the requester role.
      type: reference
      target: architecture:Container
      minCount: 1
      maxCount: 1
```

This is an excerpt of the proposed new identity form, not an input set checked against the existing product compiler. title is an optional proposal for the shared envelope; it replaces no body and no field meaning. A change of title does not change the identity. The stored model state may nevertheless capture the changed representation.

In the normal case, the Property has the logical identity (example/connector-project, Exchange/sender). A concrete definition list-documents in the same package sets kind to Exchange and sender to api. Whether the files are called Exchange.yaml or routing-model.yaml does not change these identities.

### 15.6 Renaming display name, field name and identity

**Display name:** The title Request/reply contract may become Request exchange. id Exchange, type references and property identities stay the same. The authoring name need not permanently mirror the historical ID text.

**Property field:** sender becomes requester. This is a change of the type contract, because concrete specs must now use the new key. If the same property identity is to be preserved, I propose an optional stable local id in the embedded form:

```yaml
spec:
  properties:
    requester:
      id: sender
      body: Application in the requester role.
      type: reference
      target: architecture:Container
      minCount: 1
      maxCount: 1
```

Here sender is a stable local identity building block, not a package address and not an alternative field name. The normalization still produces id Exchange/sender and kind core:Property, but sets spec.name to requester. body and contract fields are taken over as before. Without an explicit id, the building block remains the mapping key. id must not smuggle in a foreign package address or owner hierarchy; the exact segment grammar is still to be stipulated.

**This deliberately extends the previous short syntax proposal.** The probe from section 14 still rejects inline id. It remains unchanged evidence of the syntax of that time and does not evidence this extension. A future normalization must check the optional id and reject collisions. kind and a separate name are still not freely written into the short form.

The stable property identity preserves references to the field contract. It does not automatically migrate existing concrete specs: sender must become requester there. Relevant prose and integration expectations are also to be checked for the contract change. Mere text occurrences are not references completely captured by the compiler.

**Real ID change:** If Exchange itself is also to get a new id, that is an explicit identity migration. Known kind, target and value references and explicit body links must be adjusted. The compiler reports old targets left behind as errors. A permanent automatic redirection or hidden name history is not proposed for the minimal Core.

### 15.7 Moving, renaming imports and developing libraries further

| Change | Expected behaviour |
| --- | --- |
| Move a definition into another file of the same package | Identity and references persist with the same package and id. The loading selection must still capture the file. |
| Change the import alias architecture to arch | Authoring references of this package are adjusted; the canonical target identities stay the same. Other packages keep their own import aliases. |
| Write out an embedded Property | Continue it in the same package with the same complete local id, for example Exchange/sender. It remains the same item. |
| Move a Property under a different Kind | The standard derivation would change the ID. To preserve identity, write out the previous Property and reuse it explicitly; do not invent a secret new owner address. |
| Move a definition into another package | The package ID is part of the identity. This is an explicit migration or a new definition, not a mere file move. |
| Two packages each supply an id Container | No conflict: The package IDs distinguish the targets. Local alias qualification makes the authoring reference unambiguous. |
| Update a library version | Bind changed contents anew and check them against the new contract. Old published states and evidence relating to them remain bound. |
| Delete a definition and create it anew under a similar name | Existing references break; no automatic mapping by name. |

A published library state is not rewritten after the fact for a renaming. A new version can keep IDs, change their display name or need an explicit migration step. Whether old readers continue to work depends on the preserved contract. The Core derives neither development processes nor approval rights from this.

### 15.8 What the Core or the loader can check

Loader and import selection produce an unambiguous input set including package bindings. On this basis, the structural Core checks unique identities, exact reference targets, matching Kinds, valid derived property IDs and non-contradictory field names. All needed contract texts remain in the model binding.

That two freely formulated descriptions mean the same item, that a renaming is harmless in domain terms or that a new version preserves the intent remains a subject of review and suitable evidence. No stable ID is a proof of semantic compatibility.

**Recommended first contract:** Package ID plus stable local ID; local and alias-qualified references; one exactly bound content per package in the model; optional display name; property identities derived by default and explicitly stabilized when needed. No automatic name search, version selection or redirection. The proposal remains for discussion; the following section 16 now checks its resolution and binding in a standalone bounded prototype. A production import loader is not thereby implemented.

## 16. Identity walkthrough with import, update and field renaming

**Subsequent user decision:** The identity contract of this walkthrough is recorded and consolidated with the chosen basic form as a [coherent language contract](language-contract.md). The following sections document the checked draft state and its limits.

The user's request: Play through the identity contract completely at the Exchange with imported vocabulary, version update and sender/requester. The data lies under identities/ and is a closed synthetic design example. The package names do not claim existing published libraries. The chosen basic direction is preserved; the concrete package/version policy and the optional stable inline id remain examined proposals.

### 16.1 Separate package responsibility

| Package | Content |
| --- | --- |
| markitect/core, version 1.0.0 | Fixed beginning of Kind, Property and their eight property definitions. |
| example/connector-vocabulary | Importable vocabulary with the eight Kinds and twenty Properties of the Connector example, among them Exchange. |
| example/connector-project | The thirteen concrete project definitions, among them api, onprem, channels, Command and list-documents. |

The [Core artifact](identities/core-1.0.0.yaml) has a known content. The vocabulary imports core for its type definitions. The project imports core and the vocabulary under connector. The repeated dependency on the same Core version is bound once; the beginning is not instantiated several times.

### 16.2 State A: vocabulary 1.0.0

The project header in [project-a.yaml](identities/project-a.yaml) reads:

```yaml
package: example/connector-project
imports:
  core:
    package: markitect/core
    version: "1.0.0"
  connector:
    package: example/connector-vocabulary
    version: "1.0.0"
```

The [vocabulary file 1.0.0](identities/vocabulary-1.0.0.yaml) in turn imports core and describes Exchange, here shortened to the sender Property:

```yaml
id: Exchange
kind: core:Kind
body: >-
  Describes the request/reply contract between sender and receiver.
  Replies keep the MessageId of the request.
spec:
  properties:
    sender:
      body: Application in the requester role.
      type: reference
      target: Container
      minCount: 1
      maxCount: 1
```

Container is resolved locally in this vocabulary file. The project does not retroactively rename the import names within the library. The concrete definition in the project, by contrast, is of Kind connector:Exchange; it sets sender to api, receiver to onprem, command to list-documents-command, requestChannel to requests, replyChannel to replies and the values to 30 seconds and two attempts respectively.

The canonical target identities are:

| Authoring reference and context | Canonical identity |
| --- | --- |
| core:Kind in the vocabulary | (markitect/core, Kind) |
| connector:Exchange in the project | (example/connector-vocabulary, Exchange) |
| connector:Exchange/sender in the project | (example/connector-vocabulary, Exchange/sender) |
| Container as property target in the vocabulary | (example/connector-vocabulary, Container) |
| api as sender value in the project | (example/connector-project, api) |
| list-documents in the project | (example/connector-project, list-documents) |

The first two package versions are recorded with their actual SHA-256 digests in [locks.yaml, binding a](identities/locks.yaml). The project content and these bound library contents form the fixed state A. The concrete resolution chain is:

```text
Project.list-documents.kind -> Vocabulary.Exchange
Vocabulary.Exchange.kind -> Core.Kind
Vocabulary.Exchange.properties -> Vocabulary.Exchange/sender
Vocabulary.Exchange/sender.target -> Vocabulary.Container
Project.list-documents.sender -> Project.api
Project.api.kind -> Vocabulary.Container
```

All seven Exchange Properties and all referenced definitions are present in the files. The representation above is only the excerpt for the identity check. State A is structurally accepted.

### 16.3 State B: update to 1.1.0 with unchanged project definitions

The [vocabulary file 1.1.0](identities/vocabulary-1.1.0.yaml) keeps all IDs and property contracts. In the Exchange body it adds a new general obligation:

> Der Sender muss die Wartefrist jedes Versuchs anhand monotoner Zeitmessung bestimmen. (English translation: “The sender must determine the waiting period of every attempt by monotonic time measurement.”)

[project-b.yaml](identities/project-b.yaml) changes only the version import to 1.1.0. The associated content binding is under b in locks.yaml. The concrete project definitions remain unchanged and continue to be structurally accepted. The reference edges also stay the same.

The bound model state changes nevertheless: The Kind contract has a different body, the package has a different content and the project explicitly selects this new content. An old content binding for 1.0.0 must not be reused as the binding for 1.1.0.

**The structure check does not confirm the new time measurement.** Implementation and suitable evidence must be considered against the new obligation. The version number here is an exact example label, not an automatically proven compatibility promise. The probe neither changes nor checks any running Connector.

### 16.4 State C: version 2.0.0 renames sender to requester

The [vocabulary file 2.0.0](identities/vocabulary-2.0.0.yaml) takes over the obligation from 1.1.0 and renames the Exchange field key. It explicitly keeps the property identity:

```yaml
spec:
  properties:
    requester:
      id: sender
      body: Application in the requester role.
      type: reference
      target: Container
      minCount: 1
      maxCount: 1
```

In normalization this still becomes the definition (example/connector-vocabulary, Exchange/sender). Its spec.name is now requester. Neither a second Property named requester nor an automatic redirection of old data fields arises.

If only the import is updated, the [outdated project state](identities/project-c-stale.yaml) fails: sender is no longer permitted in the new Exchange spec and requester is required. The prototype first reports spec.unknown-property for the old key. The stable property ID does not keep the old field key valid.

The [migrated project state C](identities/project-c.yaml) sets the new key; shortened to the relevant values:

```yaml
id: list-documents
kind: connector:Exchange
body: >-
  The API requests document metadata and registers
  the waiting exchange before publication.
spec:
  requester: api
  receiver: onprem
  command: list-documents-command
  requestChannel: requests
  replyChannel: replies
  timeoutSeconds: 30
  maxAttempts: 2
```

With binding c this state is structurally accepted. An explicit reference connector:Exchange/sender finds the same logical property identity in A, B and C. The value relationship api stays the same; its field path in the concrete Exchange changes from spec.sender to spec.requester. The general description is adjusted to requester in 2.0.0. Whether every further domain prose has been sensibly adjusted cannot be decided by mere reference resolution.

### 16.5 What stays the same and what changes

| Feature | A | B | C, migrated |
| --- | --- | --- | --- |
| Bound vocabulary | 1.0.0 | 1.1.0 | 2.0.0 |
| Logical Exchange identity | example/connector-vocabulary + Exchange | Same | Same |
| Logical property identity | example/connector-vocabulary + Exchange/sender | Same | Same |
| Concrete identity | example/connector-project + list-documents | Same | Same |
| Field key | sender | sender | requester |
| Time measurement obligation | Not yet added generally | Added in the Kind body | Still applies |
| Structural result | Accepted | Accepted | Accepted after field migration |
| Bound model state | A | New | New again |

All three valid states contain the same fifty-one logical identities and one hundred and thirteen edges each. The equal count means neither equal contents nor equal obligations. The concrete package digests and the three different model digests are in the [check report](identities/probe.json).

### 16.6 Counter-probe: stable ID forgotten during renaming

If requester is declared without id: sender, the standard derivation produces the new property identity Exchange/requester. Exchange/sender then no longer exists. The already migrated concrete Exchange can nevertheless be structurally valid if it references no old property address.

This is important: From the change of field name alone, the compiler cannot know that an identity-preserving renaming was intended. The comparison then shows deleted and added identities; actual old property references would be unresolvable. The explicit stable id distinguishes the desired renaming from the creation of a new field contract.

### 16.7 Resolution and binding probe executed

A temporary standalone package/reference prototype resolved all three valid states with their respective content bindings. It passed the resolved package/ID pairs as unambiguous JSON pairs to the earlier independent structure prototype. The fixed beginning was bound to the known Core identities; the domain vocabularies did not thereby receive any special operators.

The probe confirmed:

- All three valid states have the same logical identities, but different bound model digests.
- State B changes the Kind body without changing the fields or reference edges.
- The property ID is preserved for sender/requester when the explicit stable id is set.
- A change of the local import alias connector to wire with adjustment of the authoring references keeps, for this example, the resolved contents and the binding.
- Without a stable inline id, the old property item is gone and a new one exists; without old references this can nevertheless be structurally valid.

Nine negative variants were rejected: old project field under version 2.0.0, new import with old lock binding, exact version not provided, two versions of the same package ID, changed bytes under the same bound version, unknown alias, unqualified search for a Kind present only by import, use of a project alias within the library, and a type definition instead of a Container instance as a concrete requester value.

The report binds nine YAML files, the temporary resolver and the separate structure prototype used through SHA-256. The previous compiler/normalization probes and their inputs were not changed. This new probe is not a product loader, not a package registry, not signature/trust evidence, not a general body link parser and not an executed realization check. The optional display name title is not needed for this walkthrough and remains a separate format question.

### 16.8 Result for the further language decision

The proposed identity contract carries this imported, updated and renamed Exchange example. Lasting identity, bound contract state and current field name can be distinguished. The necessary field migration becomes visible without breaking all references to the Property.

The version/import policy and the optional stable inline id have thus been examined concretely and within bounds. Their adoption as a general language contract, a final identity grammar as well as production loading, diagnostic and migration procedures remain to be decided. None of the successful structural probes evidences the semantic compatibility of an arbitrary update.

Back to the [shared design state](../consolidation.md) and to the [overview of terms](../README.md).
