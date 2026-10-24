# Language history before the normative v1

**Historical and not normative.** This record preserves the earlier derivation from the conversation, variants, examples and the verification states of the time (16 October 2026). Statements such as “open”, “chosen” or “next” refer to the respective state of the conversation. They are not additional v1 rules.

The sole normative source is the [language specification v1](language-contract.md). The [procedure proposals](procedure-proposals.md) that have not yet been chosen are kept separately. Current executable evidence is held with the [Exchange example](exchange-example/README.md).

Sections 1–10 and 12 below come from the language contract immediately before the editorial consolidation; section 11 was preserved separately as a procedure proposal. The substantive conversation sections are preserved; navigation references are pointed at their current owners. SHA-256 of the earlier complete UTF-8 text version: `2f62267a7d359e5f4e395c13e614447f7b960effd2cf822aa4e2c864bc6c2370`.

---

## 1. Purpose and scope

The canonical model describes the intended project world: what should hold and come into being. Binding prose and typed entries carry this description together. The files that have actually come into being, running processes and executed evidence are to be distinguished from it.

The Core provides the general means of the language completely. Projects can define their vocabulary themselves. Libraries supply reusable definitions; they grant imported Kinds no additional language capabilities or execution rights.

System, Container, Component, Command, Exchange, Rule or other domain terms are possible contents of a vocabulary. They are not privileged Core Kinds. The model requires neither a built-in Project nor a predefined software architecture taxonomy.

## 2. One basic form: definition

Type contracts, property contracts and concrete project items are definitions with the same basic form:

| Field | Content |
| --- | --- |
| imports, optional | Foreign packages visible in this definition file; if omitted, no foreign names. |
| id | Stable local identity within the package. |
| kind | Reference to the type definition whose contract this definition fulfils. |
| body | Binding description of its meaning and stipulations. |
| spec | Object with the structured values and relationships permitted according to the Kind. |
| title, optional | Readable label; not a reference target and not a substitute for body. |

id, kind and body are non-empty entries. spec can be empty but must conform to the type contract. Unknown fields are errors; for additional project entries, Properties are declared on the respective Kind. The examples use YAML as the authoring form.

Each definition is contained completely in exactly one YAML file of its own. This applies equally to Kinds, Properties and concrete items and to the definitions of the fixed Core package. Every definition file contains the same envelope with id, kind, body, spec and optionally imports and title. Multiple definitions in one file, embedded definitions and compact property declarations are not permitted. The fields within spec are determined by the respective Kind; structured object values remain ordinary values and are not embedded definitions.

The file name is not an identity. The containing folder, by contrast, determines the Package and thus part of the identity. An explicitly determined model root bounds the folder structure. The binding of external models and their exact contents is stipulated separately from the Packages; model is the shared versioning and distribution unit. From these, each definition file itself selects its visible foreign packages with imports; definitions of its own package are visible without import. Versions and digests are not repeated in every definition file. Section 8 separates these two tasks.

```yaml
imports:
  - package: example/connector/vocabulary
    as: Connector

id: list-documents
kind: Exchange
body: The API requests document metadata from the Connector.
spec:
  requester: api
  receiver: onprem
  command: list-documents-command
  requestChannel: requests
  replyChannel: replies
  timeoutSeconds: 30
  maxAttempts: 2
```

This example uses the imported vocabulary explained below. Runtime requests do not create new canonical Exchange definitions. list-documents describes the desired contract of this operation.

## 3. The fixed starting point: Kind and Property

**User decision of 15 October 2026:** Kind.properties and Property.properties are merged into one common Property properties. The chosen Core consists of **nine definitions: two Kinds and seven Properties**. For the complete simplified presentation, the meta-Properties use short IDs; as a result they need no explicitly set spec.name. These are the current design identities, not an automatic reinterpretation of old package versions.

The Core understands two fixed meta-Kinds:

| Kind | Meaning of its definitions |
| --- | --- |
| Kind | Declare type contracts for further definitions. |
| Property | Declare named attributes or relationships of such a contract. |

In the fixed Core package, Kind itself is a definition of Kind; Property is a definition of Kind. A project definition such as Exchange has the Kind Kind. A concrete Exchange definition has the Kind Exchange. A kind target must itself be a valid definition of the fixed meta-Kind Kind.

| Meta-Property ID | Contract |
| --- | --- |
| name | Optional explicitly deviating field name; if omitted, the complete local Property ID applies. |
| type | Required selection of a known value type. |
| minCount | Required non-negative minimum count. |
| maxCount | Required maximum count, or -1. |
| values | For enum, required non-empty list of distinct permitted string values; otherwise not permitted. |
| target | For reference, required reference to the target Kind; otherwise not permitted. |
| properties | Optional list of references to property definitions: field contract of a Kind spec or of an object value according to Section 3.2. |

The compiler knows the meaning of these fields and checks the supplied self-description against the fixed beginning. The self-description does not create its own validity from prose. The Core identities and the bound Core content cannot be replaced by project definitions. A local Kind named Kind or an arbitrary import alias create no special rights. Arbitrary user-defined metaclasses are not provided for in this contract.

### 3.1 Complete Core in nine definition files

All nine files belong to the package markitect/core. They need no imports because their reference targets lie in their own package. Each file has the same complete envelope. The effective field name of each meta-Property corresponds to its short local ID; spec.name is therefore omitted throughout.

**kind.yaml**

```yaml
id: Kind
kind: Kind
body: >-
  Deklariert einen Typ. properties benennt die erlaubten Felder
  der spec seiner Definitionen. Der body einer Typdefinition beschreibt
  deren gemeinsame Bedeutung und geltende Festlegungen.
spec:
  properties:
    - properties
```

**property.yaml**

```yaml
id: Property
kind: Kind
body: >-
  Deklariert einen Feldvertrag mit Werttyp und Kardinalität.
  Der body beschreibt die Bedeutung seiner Werte oder Beziehungen.
  Bei type object benennt properties die erlaubten Felder des Objektwerts.
spec:
  properties:
    - name
    - type
    - minCount
    - maxCount
    - values
    - target
    - properties
```

**name.yaml**

```yaml
id: name
kind: Property
body: >-
  Optionaler ausdrücklich abweichender nichtleerer Feldname.
  Ohne Angabe gilt die vollständige lokale ID der Property als Feldname.
  Der effektive Feldname muss innerhalb jedes verwendenden Vertrags
  eindeutig sein.
spec:
  type: string
  minCount: 0
  maxCount: 1
```

**type.yaml**

```yaml
id: type
kind: Property
body: Der vom Core verstandene Werttyp dieser Property.
spec:
  type: enum
  minCount: 1
  maxCount: 1
  values:
    - string
    - boolean
    - integer
    - number
    - enum
    - object
    - reference
```

**min-count.yaml**

```yaml
id: minCount
kind: Property
body: Nichtnegative Mindestanzahl vorhandener Werte.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**max-count.yaml**

```yaml
id: maxCount
kind: Property
body: >-
  Höchstanzahl vorhandener Werte. Der Wert -1 erlaubt beliebig viele Werte
  innerhalb der Werkzeuggrenzen. Andere Werte müssen mindestens minCount
  entsprechen. Genau 1 bezeichnet einen Einzelwert; andere Höchstwerte
  eine Liste.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**values.yaml**

```yaml
id: values
kind: Property
body: >-
  Erlaubte Stringwerte einer enum-Property.
  Bei type enum ist eine nichtleere Liste eindeutiger Werte erforderlich.
  Bei anderen Werttypen ist diese Angabe nicht zulässig.
spec:
  type: string
  minCount: 0
  maxCount: -1
```

**target.yaml**

```yaml
id: target
kind: Property
body: >-
  Der verlangte Kind der referenzierten Definitionen.
  Bei type reference ist diese Angabe erforderlich.
  Bei anderen Werttypen ist sie nicht zulässig.
spec:
  type: reference
  target: Kind
  minCount: 0
  maxCount: 1
```

**properties.yaml**

```yaml
id: properties
kind: Property
body: >-
  Referenziert die erlaubten Feldverträge des hier beschriebenen Objekts.
  Die effektiven Feldnamen müssen innerhalb dieses Vertrags eindeutig sein.
spec:
  type: reference
  target: Property
  minCount: 0
  maxCount: -1
```

### 3.2 Common Property on the Kind and on the object

The two former Properties match structurally: effective name properties, type reference, target Property, minCount 0, maxCount -1. The chosen common definition names the permitted field contracts of a described object. The difference is which object is described: the spec of the definitions of a Kind or the object value of an object Property. This mapping follows from the respective fixed meta contract and not from name similarity.

kind.yaml and property.yaml from section 3.1 reference the same Property properties. In particular, property.yaml is itself a definition with kind Kind. Its list properties is therefore the permitted field set of all property definitions; it itself needs no spec.type. Reference cycles of this finite starting point are not unfolded textually without end.

**Application on the Kind and on a nested object.** The four following files lie, conceptually, in an example package with an explicitly provided Core. The Core contract is the now chosen starting point from section 3.1 with nine definitions; the walkthrough claims no existing package publication or real digest binding. A Kind Envelope permits a required object field payload; within it, only messageId is permitted.

**envelope.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: Envelope
kind: Kind
body: Describes a message envelope with structured content.
spec:
  properties:
    - payload
```

**payload.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: payload
kind: Property
body: Structured content of the message envelope.
spec:
  type: object
  minCount: 1
  maxCount: 1
  properties:
    - messageId
```

**message-id.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: messageId
kind: Property
body: Identifier of the message.
spec:
  type: string
  minCount: 1
  maxCount: 1
```

**request.yaml**

```yaml
id: request
kind: Envelope
body: An intended message form for the example.
spec:
  payload:
    messageId: document-request
```

envelope.yaml uses properties from the Kind contract and thereby determines the permitted fields of request.spec. payload.yaml uses the same meta-Property from the Property contract and determines the fields of request.spec.payload. In both cases, messageId is a normal complete property definition. request.yaml needs no foreign import because Envelope lies in its own package; payload and messageId in the concrete value are field keys, not definition references.

**The exact compiler boundary remains fixed:**

| Definition considered | Meaning of the shared Property |
| --- | --- |
| kind is the fixed meta-Kind Kind | properties declares the permitted fields of the spec of the definitions of this Kind. This also applies to kind.yaml and property.yaml themselves. |
| kind is the fixed meta-Kind Property and spec.type is object | properties declares the permitted fields of the respective object value. |
| kind is the fixed meta-Kind Property with another spec.type | properties is not permitted as a contract entry, not even as an empty list. The already existing value type condition remains in force. |
| kind is an ordinary project Kind | Its own field contract applies. A field named properties permitted there, even when the shared Property is reused, is ordinary data content and creates no additional type or object contract operator. |

The last row prevents an impermissible global reservation of the field name. A project Kind SchemaCatalog could declare properties as a list of property references and use it to describe concrete catalog entries. Such a catalog entry does not thereby become a Kind; only a definition of the fixed meta-Kind Kind declares a new type contract. The fixed language meaning depends on exactly bound meta-identities, not on written names or an arbitrary import alias.

**Counter-probes of the walkthrough:**

| Case | Expected finding and reason |
| --- | --- |
| request.spec.payload.messageId is set | Structurally consistent with the two-level closed field contract; this does not prove a MessageId that is suitable for the domain. |
| request.spec.messageId stands instead of payload.messageId | Not permitted: messageId belongs to the object contract of payload, not to the outer Envelope spec. |
| payload contains an additional unknown field | Structural error in the closed object contract. |
| messageId with type string additionally sets properties | Structural error: on a property definition, properties is permitted only for type object. |
| A properties entry references a concrete item instead of a property definition | Structural error at the target Kind of the reference. |
| Two different Property IDs yield the same effective field name in the same contract | Structural error regardless of whether the name is set explicitly or taken over from the ID. |
| The same field name occurs in different independent Kind/object contracts | No name conflict between these contracts; uniqueness is checked per contract. The common meta-Property may be reused in Kind and Property. |
| An ordinary project definition carries a data field properties that its Kind permits | Normal value/reference check; no automatic schema execution. |
| The Core starting point references Kind, Property and properties cyclically | Finite definition graph; the references are not unfolded textually without end. |
| An object Property uses itself as its own object field | With minCount=0, finite values are possible and the recursion is permitted; with minCount>0, the mandatory self-cycle is impermissible. Section 5.3 determines the exact boundary. |

**Chosen result and migration:** In the intended Core, the common Property properties replaces the previous separate definitions Kind.properties and Property.properties. The remaining meta-Properties receive the short IDs name, type, minCount, maxCount, values and target. All Core field names are thus determined directly by the respective local ID without spec.name. Existing references, body links, seed control and affected model bindings must be migrated to these explicitly changed identities. There is no silent forwarding and no retroactively changed historical Core content. The condition for object Properties remains a fixed compiler rule.

The merge is a user decision; the examples and counter-probes explain the expected behavior of the chosen contract. This did not implement a new production Core starting point or produce independently executed compiler/runtime evidence. The historical reports on ten Core definitions and their model bindings remain evidence of their state at the time.

## 4. Binding prose and scope of validity

body is part of the applicable model contract and is preserved in the bound content.

- The body of a Kind describes the shared meaning and applicable stipulations of its definitions.
- The body of a Property describes the meaning of its values and relationships in the contract that uses it.
- The individual body supplements the stipulations of the concrete item.
- Individual entries do not silently cancel general obligations. Contradictions are treated as findings to be clarified.

The text can describe model conditions, desired runtime behavior, limits and conditions of its applicability. A statement that cannot be checked formally remains binding. For the time being, the Core has no special requirements, roles or subject sections and no general formal rule language. A project can use such words as its own terms without thereby receiving special compiler semantics.

Relationships obtain their domain meaning from the type and property contract. A statement about requester concerns the application referenced there in the described exchange. Neither global rule applicability nor transitive composition, agent partitioning or write rights follow from a relation name.

A mere word in the body creates no reference edge. Explicit body links use the same target address mechanism. The following section makes concrete the direction already agreed in the [earlier prose draft](../../product-model-refinement/canonical-model-proposal.md#25-definitionskörper-explizite-links-und-linkvorschläge). The user chose this body link contract in the subsequent design conversation. The chosen lexical rules are in section 4.5; no product implementation is claimed.

### 4.1 One link format for all definitions

The authored form is `[[target]]` or `[[target|display text]]`. The target is an id available in the definition's own Package or through direct imports, a complete reference package/id or an explicitly alias-qualified reference Alias.id. Resolution takes place in the package of the definition whose body contains the link and in that definition's file-local imports. It uses the same fixed package bindings as kind and the reference values in spec. The display text does not influence resolution. Without explicit display text, the written target address is displayed.

| Body link in the project package | Target in the example |
| --- | --- |
| `[[Exchange|exchange]]` | The Kind definition Exchange in the imported vocabulary. |
| `[[list-documents|document query]]` | The concrete described exchange in its own package. |
| `[[api|Cloud API]]` | The concrete API definition in its own package. |
| `[[requester]]` | The property definition with the stable id requester and the current field name requester. |

Every Property is addressable via its explicitly declared local id. There is neither a separate property link syntax nor automatic search for a field name. A field label does not automatically become an additional Property ID.

### 4.2 Exchange: contract, field and concrete value

In the vocabulary package, the Kind body can address its own Property:

```yaml
imports:
  - package: markitect/core
    as: Core

id: Exchange
kind: Kind
body: >-
  Describes a Request/Reply contract.
  The application determined by [[requester]] must
  match replies to the original Request by means of the MessageId.
spec:
  properties:
    - requester
    - receiver
    - command
    - requestChannel
    - replyChannel
    - timeoutSeconds
    - maxAttempts
```

This view shows the same Exchange definition with the body considered for the body links; it is not a second authoring source alongside section 6. The referenced Properties each stand in their own files. The link denotes the property declaration. The sentence describes the obligation of the application that is entered as requester in every Exchange definition. This statement comes from the prose, not from an access expression executed by the Core.

In the project package, the concrete excerpt looks like this:

```yaml
imports:
  - package: example/connector/vocabulary
    as: Connector

id: list-documents
kind: Exchange
body: >-
  This [[Exchange|exchange]] serves the document query.
  [[api|The Cloud API]] requests the data from [[onprem|the Connector]].
  The application determined by [[requester]]
  must discard late replies after the waiting period has expired.
spec:
  requester: api
  receiver: onprem
  command: list-documents-command
  requestChannel: requests
  replyChannel: replies
  timeoutSeconds: 30
  maxAttempts: 2
```

Here, body references arise to the Kind, two concrete definitions and one property definition. spec.requester, by contrast, explicitly sets the typed requester relationship to api. The body link does not set this value. If the sentence and spec do not agree, that is a semantic contradiction; a successful target resolution does not prove their agreement.

A link to requester does not denote a single property value of list-documents. The first scope needs no additional expression language for dereferencing, filters, field paths or computed links. A sentence can express the connection via the named property contract; if a concrete target must appear explicitly in the graph, this target is also referenced explicitly.

### 4.3 The fixed meaning of the link

The Core knows exactly one additional relationship from this markup: **At this body location, the source definition explicitly refers to the target definition.** For each occurrence, it records the source, canonical package/ID target, optional display text and location. Repeated links remain traceable occurrences; a summarized graph view may present them as one reference with several locations. For this there is no second manual uses/references list and no new built-in domain Kind.

The domain statement arises from the sentence, the Kind contract and the property contract. The following limits follow from this:

- A Kind link does not type the source. The field kind serves that purpose.
- A Property link explains or uses the field contract; it does not enter a value into spec and does not count as fulfilling a Property cardinality.
- A definition link alone declares neither containment, runtime invocation, positive dependency nor global rule applicability. Even “does not use [[dms]]” contains an explicit reference to dms.
- Referencing a rule does not automatically activate it for the whole linked item. The applicable scope must emerge from its contract and the statement.
- Binding unmarked prose remains part of the model. Links help with unambiguous designation but replace neither the statement nor a completeness assessment.

The reference is thus machine-usable for navigation, context and change checks. A change to the target makes the sources candidates for a renewed assessment. It does not prove a necessary code change and creates no execution assignment. In semantic reading, the bound meaning of the target belongs to the understanding of the sentence; mere reachability in the graph does not transitively import all obligations of all reachable definitions.

### 4.4 Checking, renaming and model state

The compiler can recognize active links, resolve their targets unambiguously after the definitions have been loaded completely, and preserve provenance. A missing target or unknown import is an error; title, display text and expected meaning do not serve as a fallback search. Unlike with a typed reference Property, any definition is initially permitted as a target in a body link. Whether the sentence means the right Kind, the right Property or the right concrete definition remains the subject of semantic review.

When an import alias changes from Connector to Wire, only the authored references explicitly qualified with it must be adjusted, for example Connector.Exchange to Wire.Exchange. An unambiguous directly available reference Exchange remains unchanged; the canonical target stays the same. In the historical renaming case, an explicitly preserved ID such as exchange.sender with the field name requester can still be linked. In the current simplified example, the ID itself is already called requester and the link [[requester]] remains unambiguous. An outdated display text sender can still be structurally valid and yet linguistically misleading. A real change of the Property ID requires migration of its links; without migration, old targets can no longer be resolved. A version update binds the same target identity to its explicitly chosen new version.

The fixed model snapshot contains the complete body and the resolved target identities. Changes to prose or display text are preserved even if the target set stays the same. Source positions belong to the diagnostic provenance, not to the logical target identity. Alias changes do not change the canonical link target; a change of the stored body can nevertheless be visible in the content comparison. The exact digest encoding is now in section 8.3.

The compiler checks the existence of the references, not the truth, completeness or correct interpretation of the statements. Agents can propose missing or wrong links; only a change adopted in the source changes the canonical reference. Link proposals from word similarity are not model edges.

### 4.5 Binding body link and Markdown profile

**User decision of 15 October 2026:** body uses [CommonMark 0.31.2](https://spec.commonmark.org/0.31.2/) with the explicit extension by body links. These rules apply exclusively to body. title and other string values are not examined for model links.

Processing begins with YAML decoding according to Section 5.1. Then the decoded body is read as Markdown with the body link extension. YAML escapes can therefore already produce characters of the link notation before Markdown processing. By contrast, no new model links are produced from later Markdown character references such as &#91; or from rendered text. The original body is preserved; normalization for Markdown processing is not an automatic change of the source.

| Context | Behavior |
| --- | --- |
| Ordinary text, heading, list, quotation or formatting outside a link | The unprotected double-bracket notation is active. |
| Inline code and indented or fenced code block | Literal content, no model links or link syntax errors from it. |
| Ordinary Markdown link or image, including text, destination and reference definition | No body link evaluation within this construct. |
| Autolink, HTML tag, HTML comment or raw HTML block | No body link evaluation in it. Normal Markdown text between inline HTML tags continues to follow its normal rules; link elements do not form a nested model link. |

In ordinary text, the active double-bracket notation takes precedence over ordinary Markdown link recognition: `[[api]](Text)` contains the model link api followed by normal text. A link or image already begun with ordinary Markdown syntax, by contrast, protects its content. There is no additional image/embedding form for model definitions.

**Grammar:** An active link has exactly the form `[[target]]` or `[[target|display text]]`. The target is a complete authored reference according to Sections 7 and 8. It must not be empty and contains no surrounding spaces; it is neither trimmed nor rewritten. NFC, declared spelling, import boundaries, collision and ambiguity rules apply unchanged. The link can address any definition category. Field paths, URL targets, title or similarity search are not added.

The optional display text is plain Unicode text. It is preserved after the escapes below and is not evaluated again as Markdown, character reference, model link or expression. It must not be empty or consist exclusively of Unicode White_Space characters of the bound Unicode version. Formatting stands outside, for example `**[[api|Cloud API]]**`. Within an active link, LF, CR, NEL, LINE SEPARATOR and PARAGRAPH SEPARATOR as well as nested unprotected brackets are excluded.

**Escapes:** Outside the link, the CommonMark rules apply. `\[[api]]` makes the notation literal; a backslash protected by a further backslash does not prevent a link that then begins normally. In the display text, exactly `\|`, `\[`, `\]` and `\\` are permitted for the corresponding literal characters. Other backslash sequences inside the link are errors. In the target, no escapes are needed or permitted; its characters follow the name grammar. An additional unprotected |, an unprotected single bracket or an active [[ without matching closure is a markup error. An isolated ]] outside an active link remains ordinary text.

A link must be single-line in the decoded body. The authored source text may be folded by YAML over several file lines as long as no forbidden line break character arises in the decoded link. Invalid active notation is a structural error and is not silently downgraded to text. Character sequences of the same kind in the protected contexts, by contrast, remain literal.

**Locations:** Each occurrence receives a zero-based, end-exclusive UTF-8 byte range [startByte, endByte) in the decoded original body. It covers the complete link notation. The mapping to the original YAML file additionally records start and exclusive end as one-based line and column; columns count Unicode code points, not UTF-8 bytes or screen widths. Quote escapes and YAML folding must be taken into account in doing so. Repeated links remain individual occurrences. Locations are provenance and not an additional target identity.

The bounded example verifier uses an isolated CommonMark parser dependency bound via requirements.txt and checks body link syntax, protected contexts, reference resolution and source mapping with counterexamples. The Go product dependencies are not changed by this. Its technical limit is 2000000 UTF-8 bytes per body and 64 nesting levels; a source mapping that cannot be evidenced or a resource abort reports the check as not completed. Because of a known whitespace deviation of the bound parser version, the probe also reports this for otherwise read bodies with U+001C to U+001F or U+0085. This is an explicitly declared tool limit and not a prohibition of these characters in the language contract. A separately passed body link part proves neither the validity of the whole model nor the truth of the linked prose. The complete canonical byte encoding is now in section 8.3; generic semantic procedures remain a separate task.

## 5. Values and property contracts

| Value type | Meaning |
| --- | --- |
| string | Text value. |
| boolean | Truth value. |
| integer | Arbitrary-precision integer. |
| number | Finite, exactly represented decimal number value; integers are also permitted. |
| enum | One of the explicitly declared string values. |
| object | Object with declared, closed field contracts. |
| reference | Reference to an existing definition of the explicitly required Kind. |

The following fixed language rules apply to Properties:

- spec.name is optional. If omitted, the effective field name is exactly the complete local id of the property definition. Package name, import alias and reference spelling do not belong to this field name; dots or prefixes of the local ID are not cut off.
- An explicitly set spec.name must be a non-empty string; null or an empty entry does not trigger a fallback. The effective field names must be unique within every Kind or object contract.
- minCount is at least 0. maxCount is -1 or at least minCount.
- maxCount=1 denotes a single value; other maximum values denote a list. -1 permits any number of values within finite tool limits.
- An omitted entry counts as zero present values. The data value null is not permitted.
- reference requires target. The target of target is a valid type definition.
- enum requires a non-empty list of unique permitted values.
- object uses the independent property definitions referenced under properties. Undeclared object fields are errors. A concrete object value contains values of these fields, not new property definitions.
- values, target and properties are permitted only for the value types intended for them. A new type name installs no parser or operator.

A separate value type kindReference is not needed in the new contract. A type reference can be described by reference with the target Kind Kind. The type references in the common field kind remain explicitly checkable as such.

Cardinality bounds the number of values, not their numeric value range. A domain statement such as “maxAttempts must be positive” still requires semantic assessment as long as no additional formal value condition has been introduced.

### 5.1 YAML and value profile

**User decision of 15 October 2026:** The authoring form uses a bounded profile of [YAML 1.2.2](https://yaml.org/spec/1.2.2/). The following contract determines the reading form independently of library and implementation language. It is not a choice of PyYAML for the product. Markitect is built in Go; the Python verifier is a separate design probe. Its parser defaults are not normative.

| Area | Binding rule |
| --- | --- |
| File and document | UTF-8, exactly one YAML document and a mapping as root. A single document marker and an explicit YAML 1.2 directive are permitted. Other version directives and document sequences are errors. |
| Keys | Exclusively strings, written exactly and in NFC. Duplicate keys and keys colliding according to Section 7.2 are errors. Non-textual keys are not converted into strings. |
| Explicit contents | Anchors, YAML aliases, merge keys and custom tags are excluded. For a uniform authored form, explicit standard tags such as !!str are also excluded; ordinary and quoted values suffice. |
| Representation | Comments, indentation, block/flow representation and the order of mapping keys do not by themselves produce different model content. YAML decoding must yield the same value in doing so. |
| Strings | Plain, quoted and multi-line strings are permitted. Quoted and block scalars remain strings, including empty strings. Escape characters, line folding and trailing lines are decoded according to YAML; this decoded text is preserved. Required fields can additionally require non-empty content. Unpaired Unicode surrogates are not permitted text values. |
| Booleans | Only the unquoted tokens true and false are read as truth values. yes, no, on, off, True or FALSE remain strings. |
| Date and other text | Date-like tokens such as 2026-10-15 remain strings. There is no automatically created date type. Other unquoted tokens that do not match any of the forms defined below remain text. |
| Missing entries and null | Optional fields may be absent. The data value null is excluded; an empty YAML value entry as well as the unquoted null forms Null, NULL and ~ are also errors. A literal text such as "null" must be quoted. |
| Type checking | No subsequent type conversion based on the Property: "42" does not satisfy an integer Property, 42 does not satisfy a string Property and "true" does not satisfy a boolean Property. |

The file-local import aliases with as remain permitted. The prohibition concerns YAML anchors and their alias references, not our import names.

**Number lexical form and precision:** Numbers use ASCII decimal digits, optionally a leading minus and, for decimal numbers, an ASCII point or an exponent. Unicode in names and texts introduces no further number notations.

- An integer literal has the form `-?(0|[1-9][0-9]*)`, for example 0, -42 or an arbitrarily long digit sequence without leading zeros.
- A decimal literal has the same integer part and either a decimal point with at least one following digit, an exponent, or both. An exponent begins with e or E, optionally allows + or - and contains at least one ASCII digit. Examples: 0.1, -42.0, 1e300, 1.25e-3. A leading +, .5 and 1. are not number literals of this profile.
- Integer values are arbitrarily precise; decimal values are preserved without binary floating-point rounding. A number Property accepts both kinds of literal. An integer Property accepts exclusively integer values; a decimal literal such as 42.0 is not subsequently converted into an integer.
- Hexadecimal, octal and other additional number notations are not read numerically. For example, 0x10 and 012 remain text and do not satisfy a number Property. Quoted numbers likewise remain text.
- Non-finite numeric tokens are excluded. These include .nan, .inf, +.inf, -.inf, NaN, Inf and Infinity, with an optional sign for Inf/Infinity and regardless of upper/lower case. As explicitly quoted text, these character sequences remain permitted.

There is no language-level 64-bit range or fixed rounding context. Finite tool limits are to be reported explicitly; exceeding a limit is not a silently rounded number and no evidence of the domain invalidity of the model. The current probe limits numbers to 100000 coefficient digits and an absolute exponent of 1000000000. The report names these limits; an aborted source run reports the check as not completed.

Mapping order thus has no semantic meaning. The chosen list rules are in section 5.2. The canonical serialization of equivalent number spellings remains part of the separate encoding decision.

The bounded probe uses PyYAML exclusively for syntax decomposition, reads scalars itself according to this profile and then checks the definitions. It neither extends the product dependencies nor implements the new Go compiler. The later Go processing must fulfill the same language contract; the library must not change it through other implicit reading or conversion rules.

### 5.2 List order and missing versus empty lists

**User decision of 15 October 2026:** Ordinary value lists are ordered. Declarative enumerations of the fixed Core and the import list are unordered. A missing optional list and an explicitly empty list have the same canonical value. The original authoring form is retained as source information.

| Use | Meaning of the order |
| --- | --- |
| Ordinary multi-valued Property, including reference and object values | The order counts: [a, b] is different from [b, a]. Repetitions are preserved unless an additional contract excludes them. |
| properties of a definition of the fixed meta-Kind Kind | Unordered declaration of the permitted fields. References are compared by canonical target identity. |
| properties of an object Property of the fixed meta-Kind Property | Unordered declaration of the object field contract; the same target resolution applies. |
| values of an enum Property of the fixed meta-Kind Property | Unordered set of permitted string values. The values remain exact and are not normalized or compared case-insensitively. |
| File-local imports | Unordered selection of visible packages and declared aliases. The order creates no precedence in resolution. |

The special meaning follows exclusively from the fixed meta-contract, not from a written field name. An ordinary Kind with a data field properties or values has an ordered value list there. This also applies when the shared Core Property properties is reused as an ordinary data field. No additional collection types, sort fields or compact authored forms are introduced.

Unordered does not mean that errors are removed: duplicate field declarations, duplicate enum values and duplicate package imports remain impermissible according to their existing contracts. The verifier validates these rules before the comparison and must not make invalid inputs valid by sorting or deduplicating. Added or removed members remain content changes. Only reordering a permissible declarative enumeration does not change its contract. This also applies to the declarative lists of the fixed Core self-description.

**Missing and empty:** For a permissible optional list Property, that is minCount 0 and maxCount not equal to 1, a missing field and [] each mean the same empty sequence. This applies recursively in object values, for an empty Kind/object field contract and for missing versus empty imports. With minCount greater than 0, both forms are invalid. Additional meta-rules remain in force: the values of an enum Property, for example, must still be non-empty. Fields forbidden for a value type are not supplemented by an assumed empty list.

The rule concerns neither optional single values nor empty strings or empty objects. A missing object single value remains distinct from an explicitly present {}; optional lists within the present object are nevertheless compared as the same empty sequence. null remains excluded. A missing list thus does not mean additional information such as unknown, determine automatically or adopt; a needed distinction must be expressed explicitly in the project vocabulary.

For the comparison of ordinary lists, order and repetitions are preserved. Declarative references are compared without order after unambiguous resolution, enum values as exact strings and imports by their resolved bindings. Source text, prose, original order and field presence remain available for presentation and provenance; the source files are not automatically supplemented or sorted.

The example verifier has a bounded comparison probe for this after a successful structural check. It also checks nested lists, deviating ordinary data fields and the fixed Core self-description. This earlier list probe is not a complete canonical byte encoding and does not itself produce a canonical content digest. The complete serialization, defaults and reference projection are now defined in section 8.3 and have their own counter-probes.

### 5.3 Object contract cycles and resource limits

**User decision of 16 October 2026:** Ordinary definition references may form cycles, including the fixed Core starting point. Reference targets are resolved as identities and not replaced by textual or value-side unfolding. For object contracts, the following separate condition applies.

An object contract graph has the object property definitions as nodes. An edge leads from the object contract to every object Property declared in it as a field. An edge is mandatory if the target Property has minCount>0; this applies equally to single values and non-empty lists. The subgraph of exclusively mandatory edges must be acyclic. Thus every permitted object cycle contains at least one optional edge with minCount=0. Unused object contracts are also checked for mandatory cycles; a contract that is impossible to satisfy does not become valid through the absence of concrete instances.

| Case | Language rule |
| --- | --- |
| Optional object Property uses itself | Permitted; the value can end without this field or with an empty optional list. |
| Mutual object contracts with at least one optional edge per cycle | Permitted; finite concrete values are possible. |
| Object cycle exclusively via mandatory fields | Error; a finite value cannot fulfil this obligation. |
| Ordinary reference Property leads to an already referenced item | Permitted definition reference; no object expansion. |
| Empty list with minCount=0 and maxCount=0 | Permitted; no object values are produced or visited. |

Object values remain finite, closed YAML values according to Section 5.1. Value checking and canonical projection follow exclusively values actually present; they do not expand the contract graph in advance and do not produce missing optional objects. The preceding rules for missing optional lists apply unchanged within present objects. The prohibition of YAML anchors and aliases remains in place.

**Resource policy:** The language sets no additional fixed maximums of size, quantity or depth. Tools may use finite processing budgets but must declare their scope and values explicitly and name them in the check report. A mere limit overrun is no evidence of a model violation: the affected check reports not_completed and produces, for this model scope, neither canonical bytes nor a digest nor evidence of completeness. A language error that has already been proven remains an error even if a further check aborts; the abort is additionally declared. An independent, completely checked model scope can still receive a result.

Completely finished runs over the same bound inputs must deliver the same canonical bytes and digests given sufficiently large budgets. Budgets and runtime capacities belong to the tool configuration and check provenance, not to the canonical model projection. There is no silent truncation, skipping or rounding as a successful evaluation. Runtime-caused recursion or memory aborts are likewise declared as incomplete evaluation.

In addition to the body link/number budgets already named, the bounded Python verifier uses an explicitly configurable budget for actually nested object values: --max-object-depth, default 128. Its Python recursion limit is named in the report; no separate fixed memory byte amount is configured. A memory abort is reported as such. The object contract check uses an iterative traversal for its mandatory graph; mere contract depth consumes no object value depth budget. These probe limits are not language maximums. The counter-probes check terminable self-cycles and multi-node cycles, mandatory cycles, closed values, unchanged sources and bytes with different successful budgets, and absent digests on abort.

The rules of this section concern definition references and object contracts. For the separate model.yaml dependency graph, the acyclicity rule from Section 8.1 applies; a cyclic definition reference is not a self-dependency of the model.

## 6. Standalone Properties and one file per definition

**User decision of 15 October 2026:** Each definition receives its own file and the same basic form. A Property is written exclusively as a complete definition with kind Property. Under Kind.spec.properties stands a list of references to such definitions. Embedded property contracts and compact special reference forms are not permitted.

The [complete simplified Exchange example](exchange-example/README.md) contains the actual definition files including Core, vocabulary, concrete project items and a bounded structural probe. These YAML files own the executable example content; the following views explain it. The language contract owns the general rules. Historical probe files are not rewritten by this.

Exchange and its seven Properties lie in the vocabulary package. Container, Command, Channel and MqttService are completely defined there, as are the Properties they need. Short IDs form the respective field name; spec.name is therefore not needed. The file names serve only for storage. In particular, command-kind.yaml and command-property.yaml distinguish the Kind Command from the Property command unambiguously, also on Windows.

**exchange.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: Exchange
kind: Kind
body: >-
  Describes a Request/Reply contract between requester and receiver. Replies keep
  the MessageId of the Request. timeoutSeconds applies per attempt; maxAttempts includes the first
  attempt.
spec:
  properties:
  - requester
  - receiver
  - command
  - requestChannel
  - replyChannel
  - timeoutSeconds
  - maxAttempts
```

**requester.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requester
kind: Property
body: >-
  Application in the requester role of this exchange.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

**receiver.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: receiver
kind: Property
body: >-
  Application in the handler role of this exchange.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

**command-property.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: command
kind: Property
body: Contract of the requested operation.
spec:
  type: reference
  target: Command
  minCount: 1
  maxCount: 1
```

**request-channel.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requestChannel
kind: Property
body: >-
  Communication path for requests.
spec:
  type: reference
  target: Channel
  minCount: 1
  maxCount: 1
```

**reply-channel.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: replyChannel
kind: Property
body: >-
  Communication path for replies to the issuing instance.
spec:
  type: reference
  target: Channel
  minCount: 1
  maxCount: 1
```

**timeout-seconds.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: timeoutSeconds
kind: Property
body: >-
  Positive maximum waiting period per attempt in seconds.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**max-attempts.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: maxAttempts
kind: Property
body: >-
  Positive attempt budget including the first attempt.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

The concrete item is in [list-documents.yaml](exchange-example/model/example/connector/project/list-documents.yaml); its participants, the Command, the Channels and messaging are each also in their own files. All reference targets are present in the example.

**Further user decision of 15 October 2026:** Property IDs need no prescribed prefixes. spec.name is optional and, when absent, uses the complete local Property ID as the field name. A dot in an explicitly structured ID creates neither an owner nor a further namespace or file path. An explicit name can still stipulate a deviating field name; it does not change the identity.

Properties can be referenced by several Kinds or object contracts without a copy. Each field of a nested object also has its own property definition; the object Property refers to their IDs under spec.properties. The effective field names must be unique within every using contract. Duplicate definitions, duplicate authored keys and colliding field names are errors. The diagnostic provenance refers to the respective definition file.

### 6.1 Short Property ID with default field name

A short ID needs no additional name field. The following view corresponds to the Property requester from the complete example. It does not automatically replace an old identity such as exchange.sender; such a switch requires explicitly adjusted references.

**requester.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requester
kind: Property
body: >-
  Application in the requester role of this exchange.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

A Kind that uses this Property references requester instead of exchange.sender. Without a further entry, its concrete spec field is called requester. An explicit entry name: requester would yield the same effective field contract, without thereby already claiming identical source or model digests. The authoring source and provenance are retained; the exact digest encoding is still open.

The default is formed from the local ID of the target definition, not from the written reference: an import alias or a reference such as example/connector/vocabulary/requester also yields the field name requester. With a local ID exchange.sender and a missing name, by contrast, the field name would be exchange.sender in full; neither sender nor requester is guessed. An explicitly set name: requester maintains the previous field naming.

If an ID changes while name is missing, its default field name changes as well. If existing field keys are to be kept, the previous name must be set explicitly during the ID migration. Conversely, with a stable ID, an explicitly set name can change the field key; affected concrete specs must be migrated. Deleting an explicitly deviating name activates the ID default and can therefore also require a field migration. IDs are not automatically renamed by these rules.

## 7. Identities and references

**User decision of 16 October 2026:** Package is exclusively a namespace. Every folder under an explicitly determined model root forms exactly one Package; its relative folder path determines the Package name. Additional namespaces and a namespace field do not exist. The earlier coupling of Package and versioning/distribution unit is lifted. Imports continue to make the complete IDs of the definitions of an explicitly imported Package directly available; kind can, for example, use Kind or Exchange instead of Core.Kind or Connector.Exchange. Package names use / according to the relative folder path. Full Package qualification is package/id; short alias qualification is still Alias.id. Structured local IDs keep dots. The earlier dot notation for Package names is superseded and is not a valid alternative spelling. The model is the common versioning and distribution unit; section 8.1 determines manifest, identity, root and dependency form. Model dependencies must be acyclic according to section 8.1; self-dependencies are thus also forbidden.

The logical identity remains **package ID plus stable local id**. It contains neither Kind nor display name, file name or version number. The folder path does, however, determine the package ID; a different Package folder therefore changes the identity. A specific revision is additionally determined by its model state and that model state's content bindings. The authored reference package/id or Alias.id is resolved to this canonical pair.

| Item | Package | Local id |
| --- | --- | --- |
| Exchange Kind | example/connector/vocabulary | Exchange |
| Requester Property | example/connector/vocabulary | requester |
| Concrete exchange | example/connector/project | list-documents |
| Concrete API | example/connector/project | api |

The Package names are synthetic examples with / between the folder components. An ID such as exchange.sender is a complete local name within a package; the dot defines no additional namespace hierarchy, containment or property dereferencing. An import makes exactly this complete ID available, not automatically the last segment sender. A / is not permitted in a local id: it separates Package components and, in full qualification, the Package from the complete local ID. The chosen switch is made explicitly in imports and references; old dot spellings for Packages are neither accepted nor automatically forwarded.

For identity and collision checking, case makes no difference. Within a package, therefore, IDs of different Kinds also must not be distinguishable solely by case. The permitted authored form is nevertheless strict and must be adhered to exactly: a deviating spelling is an error, not an accepted alternative reference. The verifier does not automatically correct or normalize any source file. Readable IDs are not changed at every change of the preferred designation, nor deliberately reused for other items. An optional title is not an alternative address.

Resolution of authored references uses directly available names and, where needed, explicitly qualified names. Only its own package and the imports declared in the respective definition file are visible; supplying a model alone opens no foreign names:

- A complete id is resolved against the definitions of its own package and of all packages directly imported in this definition file. Exactly one canonical target must match. Example: Kind comes from the imported Core package, Exchange from the imported vocabulary, api from the own project package.
- Several different targets with the same visible id are ambiguous even if they differ only in letter case. There is neither a silent precedence of local definitions nor a selection by the expected Kind or the order of the imports. The reference must be explicitly qualified or the name collision resolved.
- Full qualification uses importedPackagePath/id, for example markitect/core/Kind or example/connector/vocabulary/exchange.sender. The last / separates the Package and the complete local ID. An explicitly declared short alias, by contrast, is used with a dot: Core.Kind or Connector.exchange.sender. Both forms denote exactly one Package imported in this definition file. Dots in a local ID remain part of this ID. Alias qualification can still collide with a directly available structured ID; different targets are an error. There is no step-by-step namespace or file path search.
- Import aliases must not shadow directly available complete IDs with a different meaning. If the same spelling were both an ID and an alias qualification with a different target, that is an error; the alias or the ID must be changed.
- Only direct imports make names available. Further provided Packages and the contents of transitive model dependencies are not automatically included in the visible namespace of a file. Access to them requires a separate explicit import in the using definition file.

All reference positions use this target resolution: kind, Property.target, reference values in spec and body links. kind and Property.target must denote type definitions; a concrete requester value must, for example, denote a Container definition. The expected target type is checked only after unambiguous resolution and is not used as a search heuristic.

Missing or ambiguous names, unknown import aliases and contradictory import entries are errors. There is no search outside its own package and the direct imports of this file, no automatic version selection, no file path interpretation and no resolution via similar names. After resolution, canonical package/ID targets are used. Short and qualified references may denote the same target but do not create different identities.

### 7.1 Binding spelling conventions and collision checking

**User decision of 15 October 2026:** Letter case does not create a different identity. The spelling conventions are enforced as language rules. Violations are rejected; a hint at the expected form is permitted, silent correction or further processing of the deviating form is not.

| Language identifier | Prescribed form | Examples |
| --- | --- | --- |
| Package name | Lowercase components derived from the folder path, with / as separator | markitect/core, example/connector/vocabulary |
| ID of a Kind definition | PascalCase | Kind, Property, Exchange, MqttService |
| ID of a property definition | camelCase | requester, maxAttempts, messageId |
| Effective field name, including explicit spec.name | camelCase; structured names component by component | requester, timeoutSeconds, exchange.sender |
| ID of a concrete item | lowercase kebab-case | api, list-documents |
| Import alias | PascalCase | Core, Connector |
| YAML file name | lowercase kebab-case with .yaml | exchange.yaml, max-attempts.yaml |
| Model ID | Lowercase Unicode segments, separated by . or -; digit start allowed, see Section 8.1 | markitect-core, markitect.core, 2026-modell |
| Version label | The same character grammar as model ID; additionally no range notations or latest | 1.0, 1.0.0-rc.1, 2026.10 |

The category of a definition follows from its resolved typing, not from the case of a written kind value. A Kind definition is a definition of the fixed meta-Kind Kind; a property definition is a definition of the fixed meta-Kind Property. Acronyms follow the same word spelling: MqttService and messageId, not MQTTService or messageID. A reference must use exactly the declared spelling of the unambiguous target. kind: exchange instead of kind: Exchange is thus an error, even if the diagnostics can find the intended target. Likewise, inappropriately spelled package names, aliases, field keys and file names are errors. Language keys such as id, imports, package, as and minCount must be written exactly.

Structured local IDs remain single names: for each component separated by a dot, the convention of the respective definition category applies. A Property ID exchange.sender thus remains possible; without spec.name, exchange.sender is then also the complete literal field key, not an access expression. camelCase per component also applies to such structured field names. A Kind may, for example, be called Messaging.Exchange. This creates no additional namespace.

Package/alias qualifiers and local ID are resolved by the declared imports and actual targets, not by their case or the expected target type. There is no selection by longest prefix or random order.

Collisions are checked independently of case: package IDs, definition IDs within a package, aliases within a definition file, effective field names within a Kind/object contract and file names in the respective package. Kind and Property do not get separate ID namespaces in this. In particular, Command and command in the same package are a conflict, even if each individually satisfies the convention of its category. Different file names do not resolve this definition conflict.

The comparison serves exclusively identity, conflict detection and diagnostics. Source text, stored canonical spelling and references are not automatically rewritten. A wrong reference with an unambiguous target is a spelling error; several target identities that match after normalization remain ambiguous and are not selected by the expected category or an exactly written subset. Data values, enum values, body, title, topic patterns and operation identifiers are not generally compared case-insensitively; the respective data contract applies to them.

The Unicode basis and the exact name comparison are in section 7.2. The earlier ASCII-restricted profile is superseded. The permitted characters follow the Unicode profile bound there; body link escapes and source positions are stipulated in section 4.5. The YAML/value profile is in section 5.1.

The existing Exchange file collection still contains the definition conflict Command/command. The user requested a modeling discussion before any renaming or merging. This conflict therefore remains explicitly visible and must be rejected by the current verifier. The earlier passed reports refer to the previous, still case-sensitive rule; they are not a confirmation of the current state.

### 7.2 Unicode, normal form and name comparison

**User decision of 15 October 2026:** Unicode is the consistent text basis. All files use UTF-8. Unicode is permitted in body, title and string/enum values just as in package names, local IDs, aliases, effective field names and YAML file names. The fixed language keys and structural characters such as /, dot, hyphen, .yaml and body link delimiters keep their exact spelling and meaning.

**Version binding of the current draft profile:** The character tables are fixed to Unicode 15.1.0, corresponding to [UAX #31, Revision 39](https://www.unicode.org/reports/tr31/tr31-39.html). This binding uses the tables of the existing Python 3.13 verifier and is not a claim about the newest Unicode version. A change of the tables is an explicit change of the language profile. An implementation must not silently use the respective operating system, runtime or locale setting. The example verifier checks the table version and rejects a deviation.

The following profile applies to identifiers:

- A name word begins with XID_Start and continues with XID_Continue. Connector_Punctuation, including underscore, is excluded; this preserves the spelling conventions already chosen. Spaces, emoji and arbitrary symbols thus do not become name characters. In normal text values they remain usable according to the YAML profile from section 5.1.
- / structures Package names; dots structure local IDs. Each component must be non-empty. A single folder name contains no dots. Package names are relative logical paths without leading or trailing /, empty components, . or .., backslash, drive prefix or URL decoding. No file system path normalization takes place. Aliases are single PascalCase names without dots. In concrete IDs and file names, the ASCII hyphen separates non-empty lowercase words; after a hyphen, a part may also begin with a Unicode decimal digit, for example größe-2. The first component still begins with XID_Start.
- PascalCase and camelCase are checked by the first cased letter of a component. Titlecase characters count as an uppercase initial. Without such letters, this initial requirement does not apply, for example for 文档. For lowercase names, all cased letters must be lowercase. In PascalCase/camelCase words, consecutive uppercase letters remain excluded; combining marks are skipped in this check. Thus Überweisung, gültigBis and MqttService remain matching examples. These categories still follow from the typing.

**Strict authored form:** Names, file names and written references must already be in [Unicode Normalization Form NFC](https://www.unicode.org/reports/tr15/). A decomposed spelling, for example U+0075 U+0308 instead of U+00FC for ü, is rejected with a diagnostic. Neither a source file nor its decoded name is silently corrected. A reference must furthermore use exactly the declared spelling of the unambiguous target. Equivalence in the collision comparison does not make a deviating authored reference valid.

**Collision key:** For names we use Unicode Canonical Caseless Matching with full Default Case Folding, culture-independent and without Turkish special handling. Precisely, K(s) = NFD(toCasefold(NFD(s))) applies according to definition D145 of the Unicode Standard. Two names collide exactly when their keys are equal, each within the already stipulated scope: package, definition, file-local alias, field contract or file selection. The comparison key is an internal means of checking, not a new stored authored form. In particular, straße and strasse as well as Straße and Strasse collide. A corresponding reference deviation is also rejected; the verifier does not select an exactly written subset from several colliding targets.

NFC and this comparison do not cause general transliteration: ä and ae remain different. An additional NFKC compatibility folding is not chosen. body, title, string and enum values are neither normalized nor compared case-insensitively; there the decoded text is preserved and the respective data contract applies. The NFC requirement also concerns field keys, but not their ordinary text values. Provenance and original file bytes remain available for diagnostics.

The bounded example verifier applies these rules to its loaded packages, names, imports, field contracts and structured references. Its counter-probes cover Unicode names, scripts without letter case, combining marks, unchanged text data, NFC violations, the straße/strasse collision in all name scopes and strict reference spelling. The body link rules from section 4.5 and the reading form from section 5.1 have their own counter-probes; a general parser conformance proof is not claimed by this. The question of a separate Command definition is a vocabulary question and does not extend this language contract.

The shared grammar for model IDs and version labels is in Section 8.1. It uses the same Unicode tables, NFC authored form and collision key, but additionally allows Unicode decimal digits at the beginning of each segment. This extension does not automatically apply to Package folders, definition IDs, import aliases or file names.

### 7.3 Folders and Packages

**User decision of 16 October 2026:** The canonical model is structured in folders like documentation. The explicitly determined model root is itself not a Package and must not directly contain any definition files. Below this root, every folder is a Package, including an intermediate folder without definitions of its own. Every definition thus lies in a named subfolder and belongs to exactly the Package of its immediately containing folder. Subfolders belong to their own Packages; recursive collection does not make their definitions members of the parent Package. The Package of the parent folder is not inherited.

**Chosen authoring convention:** Packages are cut by domain context or feature (package by feature / vertical slices). Kinds, Properties and concrete items that belong together stand together with the respective feature. Their definition type is no reason for separate folders such as kinds/, properties/ or instances/. A further subfolder corresponds to a domain subarea; each definition keeps its own file. This convention is a decision on model organization, not an additional Core concept and not a syntactic prohibition of particular folder names. The structural verifier derives no domain appropriateness from a folder name.

The Package name equals the non-empty path of the immediately containing folder relative to the model root and is not maintained additionally in a package.yaml or in every definition. There is neither an empty Package name nor a Package named . nor a substitute name from the model ID. The chosen spelling joins the folder components with / and thus corresponds directly to the relative folder path:

```text
model/                                  # model root; no Package, no definitions here
  markitect/                            # markitect
    core/                               # markitect/core
      kind.yaml                         # (markitect/core, Kind)
      property.yaml                     # (markitect/core, Property)
  example/                              # example
    connector/                          # example/connector
      exchange.yaml                     # (example/connector, Exchange): Kind
      requester.yaml                    # (example/connector, requester): Property
      list-documents.yaml               # (example/connector, list-documents): concrete item
```

This excerpt illustrates the domain grouping. The older executable Exchange vertical slice still separates vocabulary/ and project/ in order to play through imports and reference forms; this division is not a recommendation for authoring organization and not part of the language syntax.

A folder name corresponds to exactly one lowercase Package component as previously defined in Section 7.2; a dot in a single folder name does not create additional components and is impermissible in this form. Case and Unicode collisions are checked independently of the behavior of the file system, including straße/strasse. Names must already be in NFC and in the prescribed form; no automatic correction.

The definitions are discovered from their respective folders. There is no separate definitionFiles list per Package and no additional mandatory definitions/ folder. The local id is still stated explicitly in every definition file; it is not derived from the file name. Renaming a file in the same folder preserves its identity. Moving a file into another Package folder or renaming/moving a Package folder changes the affected qualified identities and requires adjusted references. A different physical storage location of the entire model root preserves the relative Package names.

The folder hierarchy creates no implicit imports, inheritance, containment relationships between definitions or domain relationships. Parent, child and sibling Packages must be explicitly imported in every using file. A Package has no version, dependency list or publication file of its own. Model denotes the shared versioning and distribution unit and can contain several Packages. Section 8.1 determines model identity, root and manifest form. The model identity creates no automatic Package prefix and no additional part of the definition identity. Section 8.2 determines the lock format; Section 8.3 determines the canonical digest bytes.

The model root is explicitly determined in model.yaml. The current vertical slice uses root: model for each of the two models and places all definitions in named subfolders. Definitions directly in this root are language errors; the verifier reports them as failed and does not automatically move or correct the files. The explicitly bound model.yaml and the model-lock.yaml next to it are model metadata and not definition files; even with root: . they create no root Package. Section 8.1 determines the physical path spelling and the prohibition of redirections and overlapping model roots.

## 8. File-local imports and bound models

**Chosen:** Imports stand in the respective definition file and apply exclusively there. Package denotes the name area derived from the folder path from section 7.3. It is not a versioning or distribution unit. The earlier package.yaml with its own Package version and dependencies is superseded. The common manifest form model.yaml is in section 8.1.

**list-documents.yaml in the folder model/example/connector/project/:**

```yaml
imports:
  - package: example/connector/vocabulary
    as: Connector

id: list-documents
kind: Exchange
body: The API requests document metadata from the Connector.
spec:
  requester: api
  receiver: onprem
  command: list-documents-command
  requestChannel: requests
  replyChannel: replies
  timeoutSeconds: 30
  maxAttempts: 2
```

The file imports the provided Package example/connector/vocabulary and can therefore use Exchange directly. api, onprem, list-documents-command and the concrete Channels are the project's own definitions. An additional Core import is not required for this file because it references no Core name directly. exchange.yaml and each of its property files, by contrast, import markitect/core themselves in order to be able to use Kind or Property respectively.

The optional envelope field imports contains the foreign Packages explicitly visible in this file. Without imports or with imports: [], exclusively definitions of its own Package are visible. An unknown Package, or one not supplied in the bound input set, is an error. A foreign Package must come from the own model or from an explicitly declared direct model dependency. Transitive models are loaded for contract checking; direct access to their Packages requires a direct dependency of its own in the using model manifest. Imports select neither a new version nor a physical file system path. The provisioning of an external model and its version alone open no names; conversely, an import is not an installation instruction.

**Chosen alias form:** Each import is an object with a required package and an optional as. A string short form is not kept in parallel. as declares a file-local name in PascalCase and changes neither Package identity nor the set of available definitions.

```yaml
imports:
  - package: markitect/core
    as: Core
  - package: example/connector/vocabulary
    as: Connector
```

Without as, a Package can be qualified only under its complete Package name. With as, both the complete name and the short alias are permitted. The complete local IDs of the imported Package become directly available in both cases: Kind, Core.Kind and markitect/core/Kind can denote the same target. An alias does not make the import an import that is visible only when qualified.

Aliases apply only in the declaring definition file. Imports of other files, the imports of a referenced Kind/property definition, parent Packages and transitive model dependencies do not extend this view. Reference values and body links are always resolved in the file context in which they are written. Even several Packages of the same distribution unit need their own file-local imports.

Duplicate aliases, duplicate imports of the same Package and shadowed different targets are rejected case-insensitively. Core.Kind and core.Kind are not different addresses; with Core declared, core.Kind is an error. Import selection, full Package qualification and short alias must not together create different target identities for the same normalized reference spelling. There is no precedence by exact spelling, target Kind or prefix length. Section 7.2 owns the Unicode profile.

**Content binding remains required:** External reusable models are bound with an exact version and a concrete content digest. Version ranges, latest and silent content exchange under a bound version are not provided for. The bound input set must supply every canonical Package/ID identity unambiguously; multiple imports or aliases do not circumvent a conflict. The local model is bound by its fixed source/model state; no local change requires a new publication number.

The [complete Exchange example](exchange-example/README.md) now uses two model.yaml manifests, explicit local provision and Packages derived from the respective model roots. The synthetic versions claim no published libraries. Its verifier checks manifests, exact local version bindings and Package membership; it also checks the lock structure and its manifest bindings, and the real canonical content digests of valid external models. The [historical example file](identities/project-c.yaml) and earlier check reports remain evidence of their format at the time and are not rewritten.

### 8.1 model.yaml: identity, root and dependencies

**User decision of 16 October 2026:** Every model, as one unit of Packages that belong together, has a manifest with the fixed file name model.yaml. Manifest denotes the role of this file; model is the common versioning and distribution unit for several Packages. model.yaml is provisioning metadata, not an additional definition and not a new privileged Core Kind. Every actual definition keeps its uniform envelope.

| Field | Chosen contract |
| --- | --- |
| formatVersion | Required integer; binds manifest format and language contract. The current draft uses 1. Unknown versions are not silently interpreted as 1. |
| id | Required stable, non-empty model identity; independent of checkout/installation folder, storage location, model root, Package names and version. |
| version | Exact textual publication version of all contained Packages. Optional for local unpublished work; required for publication and as a supplied dependency. |
| root | Required non-empty relative directory path, relative to the folder of the respective model.yaml. There is no guessed default folder. |
| dependencies | Required mapping from directly needed model ID to exact textual version; {} without dependencies. No Packages, import aliases, version ranges or latest. |

The envelope is closed: unknown or misspelled fields are errors. A dependency must use exactly the declared model ID. Model IDs and version labels are YAML strings; numeric values are not converted. The chosen physical root path spelling is further below in this section.

**Chosen common character grammar for model IDs and version labels:**

```text
Label             = Segment (("." | "-") Segment)*
Segment           = StartCharacter ContinueCharacter*
StartCharacter    = XID_Start | Unicode decimal digit (General_Category=Nd)
ContinueCharacter = XID_Continue
```

The Unicode profile from Section 7.2 applies with the following refinements:

- Connector_Punctuation, including _, is excluded everywhere. Cased characters must be lowercase; scripts without letter case remain permitted. All labels must already be NFC. Violations are rejected, not corrected.
- ASCII dot and ASCII hyphen separate non-empty segments. There are no leading, trailing or immediately consecutive separators, not even mixed as in 1.-0. Whitespace, /, backslash, emoji and other characters not belonging to the profile are impermissible.
- The same grammar applies to model identity, dependency keys, published version, version requirement and lock binding. Model IDs may also begin with digits or contain dots. In these labels, dots create neither Packages nor namespaces; hyphens likewise carry no additional semantics.
- Valid are, for example, markitect-core, überweisungen, 文档, markitect.core, 2026-modell, 1.0, 1.0.0-rc.1 and ٢٠٢٦.١. Invalid are, for example, Markitect-Core, über_weisungen, modell/name, 1..0 and 1.0.0-.
- For version labels, latest and range notations remain excluded. The previously rejected dot-separated range markers x, for example x or 1.x, remain reserved; *, ^, ~, comparison and range operators are not label characters anyway. This reservation does not concern model IDs.

The complete Unicode collision comparison from Section 7.2 applies to model IDs within the supplied set of models and to version labels within the same model. straße and strasse must not denote different identities there. Different models may use the same version label. Requirements and lock entries must nevertheless use exactly the declared authored form; a colliding spelling is not a valid alternative reference. The first language scope still binds only one version per model ID; the collision check opens no multi-version selection.

**Chosen version semantics:** A version is an exact text label. Manifest, direct dependency requirement and lock must use the same decoded text. The language does not interpret its parts as numbers, adds no missing components and derives neither version order nor compatibility from it. "1.0" and "1.0.0" denote different versions; "1.0.1" also does not satisfy a requirement for "1.0.0". Version ranges and latest remain forbidden. The concrete content state is additionally bound by the canonical digest. The common character grammar above is chosen; it defines no SemVer grammar or compatibility evaluation.

**model.yaml**

```yaml
formatVersion: 1
id: connector-example
root: model
dependencies:
  markitect-core: "1.0.0"
```

This local model contains the Packages example/connector/vocabulary and example/connector/project together with their intermediate folders. Its own version is deliberately omitted: local changes are bound by the fixed source/model state and require no new publication number. A version: "1.0.0" could additionally be set for a published version.

**dependencies/core/model.yaml**

```yaml
formatVersion: 1
id: markitect-core
version: "1.0.0"
root: model
dependencies: {}
```

The second manifest belongs to the separately provided Core model. Its folder model/markitect/core/ provides the Package markitect/core with nine definitions. dependencies/core/ is exclusively the chosen local storage location of this vertical slice, not an additional authoring field, registry name or already chosen lock format. The provision of both manifest files is an explicit check input. Package names are each determined under their own root and are prefixed neither with the installation path nor automatically with the model ID.

The dependency connector-example → markitect-core requires the exact identity and version of the provided model version. A missing model, deviating spelling, contradictory versions of direct/transitive requirements or a catalog with several candidates of the same model ID must not be resolved silently. The first language scope binds one version per model ID; multi-version imports are not provided for. Reusable external contents additionally need an unambiguous content binding; the chosen lock format is in section 8.2; the canonical digest bytes are in section 8.3. The manifest verifier evidences only the existing local mapping and version equality, not a publication or an already verified content digest.

A Package occupied by definitions belongs to exactly one model in the bound input set. Two models must not split the same Package, even with different local IDs. Common intermediate folders remain possible: separate models with markitect/core and markitect/http can both contain the definition-free intermediate folder markitect. As soon as a Package contains definitions, they come from exactly one model; an identical name that occurs elsewhere merely as an empty intermediate folder supplies no further definitions. Differently written Package names that collide in the Unicode comparison remain errors. No loading or import order decides membership.

Imports remain file-local. They make exclusively the actually imported Packages visible. A Package from the same model requires an import as soon as it is not the file's own Package. The availability of a transitive model opens no direct import without a direct model dependency of its own. model.yaml is supplemented neither from imports nor from prose; it is not an installation or execution instruction.

A different physical storage location of the entire model, a different installation folder or an identity-preserving relocation of its root do not change its model ID and the relative Package names. An explicit change of the model ID changes the address of the model dependency but does not automatically create new Package/definition identities. root determines which files are loaded under which relative Package names; a change is to be checked against this new inventory.

**Chosen acyclicity rule for model dependencies:** The directed graph of the model.yaml dependencies must be acyclic. Nodes are the unambiguously bound model IDs, edges the explicitly declared direct requirements. In the complete reachable model scope, no dependency path may lead back to the starting model. This forbids self-dependencies as cycles of length one just as it forbids mutual or longer cycles. Even exactly matching versions or digests do not make such a cycle valid. Several models may use the same dependency, provided the common exact version and content binding matches.

A model cycle is a language error and yields failed, not a merely incomplete check. For the affected invalid model scope there are neither canonical bytes nor digest. The rule needs no special handling for self-dependencies. Parts that are inseparably mutually dependent in domain terms can belong to a shared model with several domain Packages. Ordinary definition references and Package imports within the same model may continue to form cycles; the object contract rules from Section 5.3 apply additionally. No dependencies are supplemented from prose or mere file links.

The acyclicity rule is a deliberately chosen boundary between jointly versioned distribution units. It does not follow necessarily from the digest form, which precisely does not embed foreign contents recursively. The check verifies the reachable model graph deterministically; common transitive libraries are not cycles.

**Chosen physical root path rules:**

- root is either . or a non-empty relative folder path, relative to the folder of the respective model.yaml. The target must already exist as a directory; missing targets and files instead of directories are errors. . denotes exclusively the manifest folder itself.
- ASCII / is the only separator, also under Windows. Every relative component follows the existing lowercase Unicode Package folder grammar from Section 7.2: XID_Start at the beginning, then XID_Continue, no Connector_Punctuation. The new model ID/version label grammar does not extend this folder grammar. In particular, dots, hyphens and digit starts are not additionally permitted in individual root folder components.
- There are no leading or trailing /, empty components, embedded . or .., backslashes, drive prefixes or absolute paths. The authored form is checked before any path interpretation; ./model, docs//model and model/ are not silently normalized. There is no variable, home, URL or other substitution.
- The written components must already be NFC and correspond exactly to the actual directory names, including case and Unicode spelling. A Model findable on a tolerant file system does not replace a written model. Several directories matching in the Unicode collision comparison remain an error; the exactly written subset obtains no precedence.
- Symlinks, junctions and comparable reparse redirections in the relative root path or in the discovered model tree are forbidden and are not followed. The explicitly loaded manifest and lock files are not read via such redirections either. The physical manifest folder forms the reference point; the absolute storage path above it obtains no Package naming convention.
- Loaded model roots must neither denote the same physical directory nor lie within one another. The check takes place for the bound set of models before definition discovery. With root: ., further bound models must therefore lie outside this model tree. Shared logical, empty Package prefixes from separate physical roots remain possible.

Valid forms are ., model, docs/model and überweisung/文档, provided the exactly written directories exist. Invalid are, for example, ./model, model/, docs//model, ../shared/model, Model and C:\\models\\connector. The absolute storage locations of the models remain free; an identity-preserving physical relocation changes neither Package names nor canonical digest bytes. The root itself still forms no Package and must not directly contain any definition files (Section 7.3).

The bounded verifier checks these rules, including actual directory names, Unicode collisions, overlaps and a real junction under Windows or a symlink on other supported hosts. Impermissible root paths and redirections are failed language checks, not areas treated as not completed. A complete model conformance or publication proof is not claimed by this.

### 8.2 model-lock.yaml: complete external content binding

**User decision of 16 October 2026:** The generated lock is stored as model-lock.yaml next to the entry manifest and versioned together with the model. It binds all direct and transitive external model dependencies exactly once. The own local model version is bound by the current source/model state and is not repeated as a separate external entry in the lock.

The closed form consists of exactly formatVersion and models. The current formatVersion is the integer 1. models is a mapping from model ID to an object with exactly version and digest. Without external dependencies, models: {} is the complete table. Every key is the exactly written NFC model ID; the Unicode collision comparison from Section 7.2 also applies here. version is the exact textual version. digest has the form sha256: with exactly 64 following lowercase hexadecimal digits. Unknown fields, spelling deviations, missing values and collisions are errors.

The stored example now contains the real Core digest computed against section 8.3:

**model-lock.yaml**

```yaml
formatVersion: 1
models:
  markitect-core:
    version: "1.0.0"
    digest: "sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1"
```

The [canonical Core file](exchange-example/canonical/core.json) contains exactly 3769 bytes without BOM and without a trailing line break. The own Core content, including its fixed self-description and dependencies, is valid and was checked before encoding. The open Command/command conflict of the other model does not make this independent Core binding invalid; for the faulty Connector model, by contrast, no digest is output. Earlier reports with a placeholder remain historical findings of their state at the time.

**Completeness:** Exactly the external lock table must arise from the direct requirements of the entry manifest and the bound model.yaml manifests of the dependencies. Missing, surplus or contradictory entries are errors. There is no additional manual list of the direct dependencies and no second description of the graph edges in the lock. One version applies per model ID; neither import aliases nor loading or table order resolve version or content conflicts.

**Checking:** Model ID, exact version and computed content digest must match simultaneously. The verifier checks all reachable manifest requirements and then every content binding. Other contents under the same bound version are rejected. A check run does not automatically write, supplement or update the lock. A missing lock is not replaced by a silently created new binding. Changes of the dependencies require a renewed check of the complete binding; local definition changes alone do not require a new lock. The root lock table determines the shared external binding of the evaluated model state; any further lock file of a dependency model that is shipped along does not silently replace or extend this table.

**Digest scope:** A model digest binds exclusively the own model content, including the own model metadata, declared direct dependency requirements, Packages and definitions. It contains no recursively embedded digests of foreign model contents. The lock table carries the complete concrete binding of the transitive contents. This avoids recursive digest definitions; ordinary definition references remain references to canonical identities and are not textually expanded infinitely. The exact projection, defaults, orderings and byte encoding are in Section 8.3.

**Provisioning:** Installation paths, storage locations and download addresses are provisioning/provenance inputs and not lock fields. The already chosen model ID is not supplemented by a storage location. Provisioning supplies contents whose identity, version and digest must be checked; the lock entry alone is not an installation instruction.

The verifier checks envelope, versions, exact model set and actually computed external model digests. Artificial comparison digests remain restricted to isolated counter-probes. The report separates manifest check, definition check, canonical model projection and lock check: the Core and its content binding pass, while the whole definition graph is still rejected because of Command/command. File hashes additionally remain byte provenance of their respective inputs and are not to be equated with canonical model digests.

### 8.3 Canonical bytes and SHA-256: model-digest/1

**User decision of 16 October 2026:** Every model digest is SHA-256 over the canonical bytes below. The encoding describes the evaluated own model content. Before any projection, the selected model and its complete dependency closure are validated: manifests, spellings/collisions, fixed Core self-description, type contracts, values, cardinalities and body links. Errors detected during loading must not be lost through a later map representation either. An invalid version receives no digest; an uncompleted check is no valid digest evidence. A valid dependency model can be checked and hashed independently of a faulty local model that uses it.

**Projection.** The encoded tree has exactly three fields:

| Field | Content |
| --- | --- |
| format | The fixed string markitect-model-digest/1 as identifier of this projection/encoding. |
| model | Own formatVersion, id, optional version and the complete direct dependencies from the own manifest; root is excluded. |
| packages | Mapping of own occupied Package names to a mapping of their local definition IDs to the following definition records. |

A definition record contains exactly kind, body, bodyLinks, spec and, if set by the author, title. Package and id are carried by the two mapping keys and are not repeated again in the record. kind is the resolved reference form from the following table. body and title are the decoded unchanged Unicode strings. bodyLinks is always a list of canonical target references in the order of the active occurrences in the body; repetitions are preserved. A target is encoded as ["reference", package, id]. Display texts and the written link notation are already contained in the unchanged body. Byte/line locations are not included. spec is a mapping of effective field names to the typed canonical values.

| Value type | Exact tree form |
| --- | --- |
| string | ["string", unchangedString] |
| enum | ["enum", unchangedPermittedString] |
| boolean | ["boolean", true] or ["boolean", false] |
| integer | ["integer", decimalIntegerAsString] |
| number | ["number", decimalCoefficientAsString, decimalExponentAsString] |
| reference | ["reference", exactlyWrittenPackage, completeLocalID] |
| object | ["object", MappingOfEffectiveFieldsToTypedValues] |
| Multiple value | ["list", ListOfSingleValueForms] |

The type form follows from the resolved field contract, not from YAML notation or field name similarity. Integer strings use 0 or an optional negative sign followed by digits without leading zeros; -0 becomes 0. number means coefficient × 10 to the power of exponent. The coefficient has neither leading nor trailing zeros; removed trailing zeros are added to the exponent. The sign appears only with a negative, non-zero coefficient. Zero is always ["number", "0", "0"]. The exponent is an integer string of the same form. Thus 42, 42.0 and 4.2e1 yield the same form ["number", "42", "0"] for a number Property. For integer, a decimal value remains impermissible. There is no binary floating-point number or context-dependent decimal rounding; large exponents are not textually expanded as huge digit sequences.

**Defaults and lists.** Missing optional lists are represented as empty lists according to Section 5.2, also within object values. The fields forbidden for a concrete meta-type are not inserted through defaults. A missing spec.name of a property definition is filled in as the complete local Property ID; an explicitly set equal value produces the same bytes. Other missing optional single values are not invented. A missing title or version remains distinguishable from a present value. Ordinary value lists keep their order and repetitions. The fixed Core declaration lists from Section 5.2 are sorted by the canonical JSON bytes of their individual elements. An ordinary data field with the same name does not receive this sorting. The fixed Core self-description likewise compares explicitly set ID default names by their effective value.

**References and text.** Structural references are resolved to the canonical Package/ID pair. Their short, fully qualified or alias-qualified authored form does not change the bytes. imports and aliases are not stored as separate content fields after successful resolution; an unused, still valid import does not influence the digest. Removed imports, ambiguities or spelling errors are nevertheless rejected before the projection. In the body, by contrast, the written link notation remains part of the text: an alias change in the body changes the bytes, even with the same target. Conversely, the additional bodyLinks targets bind a resolution changed by other imports when the body is unchanged. Body texts are not rendered, reformatted, trimmed, case-folded or subsequently Unicode-normalized. The same applies to title and ordinary string/enum values. YAML comments, indentation, quote choice and the source order of mappings influence only the authored bytes, not the decoded tree.

**JSON/UTF-8 encoding.** The following rules apply recursively to the entire tree:

- JSON object keys are sorted in ascending lexicographic order by their UTF-8 bytes as unsigned bytes, independent of locale settings or file system. Spellings are preserved exactly; the collision key is not a stored name.
- There is no formatting whitespace between tokens. Object/array delimiters are exactly the ASCII characters {}, [], : and comma. true and false are lowercase. The manifest number formatVersion is encoded as a normal JSON integer; typed payload values follow exclusively the table above.
- Strings are in double ASCII quotation marks. Quotation marks and backslash are escaped with a backslash. Backspace, tab, LF, form feed and CR use the short JSON escapes b, t, n, f and r, each after a backslash. All other characters U+0000 to U+001F are encoded as backslash-u and exactly four lowercase hexadecimal digits. All other permitted Unicode scalar values are encoded directly as UTF-8, including /, <, >, &, U+2028 and U+2029. There is no additional HTML escaping, ASCII transliteration or escape selection by runtime library.
- The output contains neither BOM nor trailing line break. Unpaired surrogates and non-finite numbers are not valid inputs. The projection uses no null and no untyped payload values or binary floating-point numbers.

**Excluded data.** File names, root, installation/download paths, source bytes, comments, locations, check reports, lock files and foreign model contents are not fields of this projection. An own digest in particular contains no recursive foreign content digests; the concrete external binding remains in the lock. Empty Packages are not included; intermediate Packages result from the complete Package paths of the contained definitions. This does not change the authoring rule that every folder is a Package, and it permits no circumvention of spelling or import errors in the preceding validation. Git need not preserve empty directories in order to produce the same canonical content bytes.

**Counter-probes and limit.** [canonical/core.json](exchange-example/canonical/core.json) is a fixed byte artifact with 3769 bytes and digest sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1. The probe checks exact escape/sorting vectors, exact large numbers, effective defaults, list rules, reference forms, text changes and preceding validation. An independent standard JSON encoder is additionally compared against the projected tree. This is executable design evidence of the defined profile, not general parser conformance or an already production-ready Go implementation. Resource limits of the probe remain disclosed and justify no secret alternative encoding.

## 9. Renamings and migrations

| Change | Contract |
| --- | --- |
| Change title | Identity remains; the changed presentation can be captured in the new state. |
| Rename file within the same Package folder | Identity remains if id persists and the file continues to be loaded. |
| Move file into another folder or rename/move Package folder | Package name and thus identity change; migrate references explicitly. |
| Change the import alias of a file | Adjust only the authored references of this file qualified with it; other files, unambiguous directly available names and canonical target identities remain. |
| Change property field sender to requester | The new field key applies in the type contract; concrete specs must be migrated. |
| Preserve the property identity in doing so | Keep id in the property file and set spec.name to the desired new field name; for example requester with the stable ID exchange.sender. |
| Omit spec.name | The complete local Property ID is the effective field name. |
| Remove deviating spec.name | The field name falls back to the complete local ID; affected concrete specs must be migrated where necessary. |
| Change ID while spec.name is missing | Identity migration and change of the default field name; an explicitly set previous name can preserve the field key. |
| Use Property under another Kind | Explicitly reference the same standalone property definition if the same field contract is desired. |
| Change local id or package membership | Explicit identity migration and adjustment of known references. |

A stable Property ID preserves references to the field contract but does not keep the old field name valid. id is always explicitly required. An explicitly set spec.name separates the field name from this ID; without name, the ID applies as default field name. If an author additionally changes the ID, that is a separate identity migration; the compiler does not guess a renaming intention and creates no new Property ID on a field rename.

A new jointly versioned model version can contain the same logical identities with changed contracts. The new binding is chosen explicitly, and affected models or realizations are considered against it. Neither a stable ID nor a version number proves semantic compatibility.

Published old versions and evidence referring to them are not changed retroactively. Automatic redirects, hidden name histories and mapping via similar names are not part of the minimal Core. On a genuine identity change, known kind, target and value references and explicit body links must be migrated. Free prose additionally requires review.

## 10. Processing and fixed Core rules

Processing takes place independently of document order:

1. Provide the known Core contract and the explicitly bound packages.
2. Capture exactly one complete definition from each definition file; check the uniform envelope, preserve provenance and reject identity collisions. Embedded property declarations and definition collections are errors.
3. Resolve references unambiguously in its own package and the imports of the respective definition file.
4. Check meta-Kinds and property contracts against the fixed beginning and its language invariants.
5. Form closed type contracts from the valid Kind/property definitions; determine the effective field name from spec.name or the complete local Property ID and reject name collisions.
6. Check all definitions against their Kind, value types, cardinalities and reference targets.
7. Form a fixed model state with complete bodies, type contracts, package bindings and resolved references.

For this, the compiler must have a fixed, built-in understanding of the document envelope, the one-file rule, the two meta-Kinds, their field interpretation, value types, cardinalities and reference rules. This meaning does not arise from its being described again in the body of the fixed beginning.

The self-typed Kind and ordinary reference cycles are finite graph relationships. They require no infinite type expansion. Object contract cycles and tool budgets follow Section 5.3: only mandatory object cycles are impermissible; actually present finite values are checked without advance expansion. Resource aborts are not language maxima and must not yield a successful check.

The result makes Kind, Property and concrete definitions addressable together. A content digest binds the model content and the chosen contracts; source provenance remains traceable. The exact canonical data encoding is chosen in section 8.3. Production error codes and report formats remain separate tool elaboration.

## 12. Evidence and elaborations still open

The [probe overview in the derivation](README.md) and the linked reports document separate scopes:

- The [complete simplified Exchange example](exchange-example/README.md) stores the chosen Core with nine, the vocabulary with seventeen and the project with seven definitions. Its local verifier and report bind the files read and check the explicit example graph. This is neither the new product compiler nor a complete alias/markup/canonicalization proof.
- The existing Go Core was checked with older examples explicitly adapted from definition to purpose. This does not evidence a new authoring form.
- The [bootstrap probe](bootstrap/probe.json) checks the finite starting point and the common definition model in a separate design prototype.
- The [compact probe](bootstrap/compact-probe.json) confirms, for the earlier state, the equality of written-out and embedded Properties. The compact authoring form is superseded by the later user decision; the historical report remains unchanged.
- The [identity walkthrough](identities/probe.json) separately checks imported versions, package contexts and the stable inline id added at the time. Its three valid states have the same 51 logical identities with different model bindings. It does not check the now chosen storage of each definition in its own file; the identity contract remains in place with explicitly set Property IDs.

The now chosen Core has nine definitions with short meta-Property IDs and one common Property properties. The historical bootstrap/identity probes concern their starting point of the time, made of ten Core definitions; their node, edge and digest statements are not transferred to this new state.

None of these reports is evidence of already production-ready support of the whole new contract. In particular, the new runtime obligation of the version example has not been confirmed by an executed Connector.

The completed state of language decisions and the separate implementation/procedure work are in the final reconciliation below. Rules already chosen are not carried again as open language decisions because their Go implementation is missing.

### 12.1 State of the definition of the canonical model

**Final reconciliation of 16 October 2026:** The canonical form of the minimal language scope chosen so far is defined. With the chosen acyclicity rule for model dependencies, the last open manifest question is also decided. The renewed reconciliation has identified no further open fundamental decision in this scope. The following mapping distinguishes language decisions from executable evidence and product implementation.

| Area | Owner and state |
| --- | --- |
| Uniform definitions, one file per Kind/Property/item, closed envelope | Section 2; chosen. No compact or embedded definition form. |
| Fixed Core and field contracts | Section 3; nine definitions, common properties, short Property IDs and optional ID default for spec.name chosen. |
| Binding body and explicit body links | Section 4, in particular 4.5; chosen CommonMark/link profile including source positions. |
| YAML, value types, cardinalities and defaults | Sections 5.1 and 5.2; chosen. |
| Object contract cycles and resource limits | Section 5.3; optional terminating edges, prohibition of mandatory cycles and pure tool budgets chosen. |
| Unicode, spelling, identities and folder Packages | Section 7; chosen, including a root without definitions and domain Package organization. |
| File-local imports and aliases | Section 8; chosen, unambiguous resolution without search priorities. |
| Models, version text labels, common character grammar, physical root paths and model dependencies | Section 8.1; chosen. Model dependencies are acyclic, including the prohibition of self-dependencies. |
| Lock and canonical bytes | Sections 8.2 and 8.3; chosen. The actual Core digest and its binding are tested. |
| Changes and processing | Sections 9 and 10; identity migrations are explicit, references are not unfolded without end. |

The executable probe is bound design evidence, not a general YAML/CommonMark conformance proof and not a production Go implementation. The current Exchange input set remains invalid because of Command/command; its vocabulary problem is not a missing language capability. Its older vocabulary/project division is not a recommended domain Package boundary.

For production, a Go compiler, an implementation-independent valid/invalid reference test corpus, diagnostic/report formats and complete source mapping as well as provisioning and migration remain to be worked out. They must not change the chosen rules through parser defaults, runtime limits or automatic corrections. Error codes and exact output formats are tool contracts, not new mandatory Kinds.

### 12.2 Result of the completeness reconciliation

**No further open language decision identified in the chosen scope.** Section 8.1 now stipulates explicitly that model.yaml dependencies must form an acyclic graph. A → A, A → B → A and longer cycles are errors. Common transitive dependencies and ordinary reference cycles within a model remain permitted; object contracts follow their own rule from section 5.3.

The bounded manifest verifier now rejects model cycles as failed and delivers no digest for them; the earlier provisional classification as not_completed is superseded. Counter-probes distinguish self, mutual, longer and transitive cycles from valid chains and jointly used libraries. The check applies both in the physical manifest binding and before the canonical projection. The resource limits of the tool remain separate from this.

In the rest of the language scope chosen so far, this reconciliation has identified no further open fundamental decision. This is no proof of suitability for every conceivable project or of complete parser conformance. Further formal value conditions, multi-version imports and additional language means remain explicitly optional extensions.

**Separate further work:** Prose and spec are binding together. A contradiction remains a semantic finding even if the structural check passes; there is no automatic precedence rule. Context, semantic index, updating, realization mapping, check planning and Apply from section 11 are elaborated procedure proposals and have not, as a whole, been chosen as parts of the canonical form. Their execution data require no additional Core Kinds. Product implementation, check evidence and domain acceptance remain separate.

An editorially streamlined normative v1 version with reference cases is sensible as a preparation; it should clearly separate the chosen language rules and deliberately excluded extensions from the historical conversation and the proposals from section 11. This is no further open language decision and no precondition for implementing the scope stipulated here.

Back to the [common design state](../consolidation.md) or to the [overview of terms](../README.md).

## Earlier navigation summary of the Core derivation

The status statements below come from the navigation before the v1 consolidation.

As of: 15 October 2026. **Status: conversation derivation, earlier variants and bounded check reports.** The current coherent state is in the [consolidated language contract](language-contract.md). The user has meanwhile also chosen the identity contract and commissioned the consolidation. Earlier proposal and status statements on this page document their respective point in the conversation; they do not override the later decisions. Product support of the new overall contract is not claimed.

**Currently chosen direction:** The user decides on **one uniform definition form with the Kinds Kind and Property understood as fixed**. In the continuation of the conversation, the user stipulates **one file per definition**, including every Property and every concrete item, and excludes compact or embedded special Property forms. [Section 6 of the language contract](language-contract.md#property-names) is authoritative. body remains the binding body; requirements and roles are dropped as special language means for the time being. [Section 14](README.md#14-chosen-basic-form-and-compact-properties) documents the compact authoring form, since superseded, and its probes of the time; a new product implementation is not claimed by this.

**Further chosen direction:** Package is the only namespace; every folder under the model root forms a Package whose name follows from the relative path. An additional namespace field is dropped. Since the user decision of 16 October, Package is no longer a versioning/distribution unit; this unit is called model. Direct imports in the respective definition file make complete definition IDs available without a prefix. The provision and version binding of models that belong together is stipulated separately and opens no names. Package names and full Package qualification use /; structured local IDs and necessary alias qualification still use dots. Ambiguities are rejected. [Sections 7 and 8 of the language contract](language-contract.md#names) own this later decision; the following historical examples and probes keep their earlier notation.

**Further selection on Properties:** No mandatory ID prefixes; spec.name is optional and, when absent, uses the complete local ID as the field name. [Section 6.1 of the language contract](language-contract.md#property-names) owns this rule. The user subsequently adopted the common meta-Property properties. The [complete Core with nine definition files](language-contract.md#core-definitions) uses short meta-Property IDs without explicit spec.name; [section 3.2](language-contract.md#property-contracts) records the use and counter-probes. Historical probes of the earlier ten-part starting point remain unchanged.

**Chosen YAML/value profile:** [Section 5.1](language-contract.md#yaml-values) owns the bounded YAML 1.2.2 authoring form, unambiguous scalar rules without implicit date/yes/on conversion, exact finite numbers and strict structural checking. Python/PyYAML remain aids of the design probe; the Go product is not switched over by this. [Section 5.2](language-contract.md#lists) now records ordered value lists, unordered Core declarations/imports and the canonical equality of missing/empty optional lists. Sources remain unchanged; section 8.3 now determines the complete canonical byte encoding.

**Chosen body link rules:** [Section 4.5](language-contract.md#body-links) binds CommonMark 0.31.2, protected contexts, exact targets, simple display texts, escapes and UTF-8/YAML locations. The bounded verifier has a parser and counter-probes for this; two active links of the example are checked separately from the still invalid whole definition graph. Folder Packages are now in section 7.3. Section 8.1 now chooses model.yaml for stable model identity, relative root, optional local version and exact direct model dependencies. Occupied Packages belong to exactly one model; empty intermediate folders can be shared. The example loads two manifests. Section 8.2 now chooses model-lock.yaml for the complete external model set with version and its own content digest, without a second graph description or storage locations. Section 8.3 now determines model-digest/1 as an exact UTF-8 JSON projection with defaults, resolved references and unchanged texts. The placeholder has been replaced by the real Core digest and checked against 3769 stored canonical bytes. No digest is output for the invalid Connector model. The model root is itself not a Package and must not directly contain any definitions; every definition belongs to a named subfolder Package. Model IDs and version labels now use the same grammar: lowercase NFC Unicode segments, with digit start and . or - as separators. Physical root paths are also chosen: . or exactly written relative /-paths with existing folders according to the Package folder grammar, without normalization or redirections and without overlapping model roots. Object cycle and resource rules are now in section 5.3: only mandatory object cycles are errors; budgets limit the tool and deliver no digest on abort. Model dependencies are now stipulated as acyclic, including the prohibition of self-dependencies. Common transitive libraries remain permitted. The final reconciliation identifies no further open language decision in the chosen minimal scope; product implementation and evidence remain separate.

The [complete simplified Exchange example](exchange-example/README.md) contains the current files for Core, vocabulary and project as well as a bounded reproducible structural probe. Since then, strict spelling conventions without automatic correction, case-insensitive identity/collision checking and file-local import aliases have been chosen. Each import is an object with package and optional as. Subsequently, Unicode as the consistent text basis, NFC for names and references and complete culture-independent name comparison including straße/strasse are chosen; section 7.2 owns the name profile bound to Unicode 15.1.0. The unresolved conflict Command/command is rejected under these rules; its modeling remains a vocabulary question. The earlier reports and examples further below remain historical derivation. [Section 12.2 of the language contract](language-contract.md#conformance) separates this example state from the remaining language decisions.

The [identity and reference contract](README.md#15-identities-imports-versions-and-renamings) distinguishes stable identity, optional display names and the binding of imported versions. After the walkthrough in section 16, the user chose this contract; the binding design rules and the format details still open are recorded separately in the consolidated language contract.

The [complete identity walkthrough](README.md#16-identity-walkthrough-with-import-update-and-field-renaming) now checks this proposal in a bounded way with an imported vocabulary, three bound states and sender/requester. It implements no product loader.

The [investigation of a common basic form](README.md#11-shared-basic-form-and-body) is the basis of the direction chosen since. The previous YAML files and their hash-bound reports remain unchanged evidence of the previous vertical slice with definition. The current implementation name purpose remains unaffected by this.

The subsequently commissioned [fixed starting point and complete type resolution](README.md#12-fixed-beginning-and-type-resolution-at-the-exchange) are worked out as a separate, bounded design probe with body. The probe uses a common definition form and does not replace the product compiler.

The [comparison of both Core variants](README.md#13-comparison-and-recommendation-for-markitect) also documents the assistant's recommendation of the time for the explicit variant. This recommendation is superseded by the subsequent user choice in favor of the uniform form. Earlier sections remain preserved as derivation and comparison.
