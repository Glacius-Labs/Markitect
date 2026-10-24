# Markitect: normative language specification v1

**Status:** Completed v1 design baseline, confirmed by the user on 16 October 2026. This file is the sole normative source for the canonical form. The chosen v1 language scope is fully designed; there are no open language decisions within this scope.

The design of the full v1 language is complete. As of 17 October 2026, the current source implements its product compiler; the [delivery plan](../implementation-plan.md) owns the remaining product work. The design probes show only the checks recorded for each probe. Compiler and repository checks do not prove real provider execution, Apply, a release or human acceptance. Project vocabulary and the separate procedure proposals are outside this language baseline.

**Normative force:** MUST and MUST NOT denote mandatory rules and prohibitions respectively. MAY denotes a permitted option. The declarative rules, tables, grammars and the nine Core definitions of this specification are also normative. There is no implicit error correction. Examples and check reports outside this file are informative and cannot supplement or override these rules.

[Language history](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/language-history.md), [procedure proposals not yet chosen](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/procedure-proposals.md) and [executable Exchange example with bounded check evidence](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/exchange-example/README.md) are separate documents. They are not alternative language contracts.

| Subject | Normative section |
| --- | --- |
| Uniform file and fixed Core | [Definition](#definition), [Core](#core-definitions), [Field contracts](#property-contracts) |
| Binding prose and links | [body](#body), [Markdown/link profile](#body-links) |
| Values, lists and finite processing | [Values](#values), [YAML](#yaml-values), [Lists](#lists), [Cycles and resources](#cycles-resources) |
| Names and Packages | [Resolution](#names), [Spelling](#spelling), [Unicode](#unicode), [Folders](#packages) |
| Bound models | [Imports](#imports), [Manifest and root](#models), [Lock](#lock) |
| Canonical content and validity | [Bytes](#canonical-bytes), [Changes](#changes), [Processing](#processing), [Conformance](#conformance) |


<a id="scope"></a>

## 1. Purpose and scope

The canonical model describes the intended project world: what should hold and come into being. Binding prose and typed entries carry this description together. The files that have actually come into being, running processes and executed evidence are to be distinguished from it.

The Core provides the general language features completely. Projects can define their vocabulary themselves. Libraries supply reusable definitions; they grant imported Kinds no additional language capabilities or execution rights.

System, Container, Component, Command, Exchange, Rule or other domain terms are possible contents of a vocabulary. They are not privileged Core Kinds. The model requires neither a built-in Project nor a predefined software architecture taxonomy.

<a id="definition"></a>

## 2. One basic form: Definition

Type contracts, property contracts and concrete project items are definitions with the same basic form:

| Field | Content |
| --- | --- |
| imports, optional | Foreign packages visible in this definition file; if omitted, no foreign names. |
| id | Stable local identity within the package. |
| kind | Reference to the type definition whose contract this definition fulfils. |
| body | Binding description of its meaning and stipulations. |
| spec | Object with the structured values and relationships permitted according to the Kind. |
| title, optional | Readable label; not a reference target and not a substitute for body. |

`id`, `kind` and `body` MUST be non-empty strings. `body` MUST additionally contain at least one character outside the Unicode property White_Space in version 15.1.0 (Section 4). `spec` MUST be a mapping; it may be empty if the Kind contract permits this. A present `title` MUST be a string. Unknown envelope fields are errors; additional project entries are declared as Properties on the respective Kind.

Each definition is contained completely in exactly one YAML file of its own. This applies equally to Kinds, Properties and concrete items and to the definitions of the fixed Core package. Every definition file contains the same envelope with id, kind, body, spec and optionally imports and title. Multiple definitions in one file, embedded definitions and compact property declarations are not permitted. The fields within spec are determined by the respective Kind; structured object values remain ordinary values and are not embedded definitions.

The file name is not an identity. The containing folder, by contrast, determines the Package and thus part of the identity. An explicitly determined model root bounds the folder structure. The binding of external models and their exact contents is stipulated separately from the Packages; model is the shared versioning and distribution unit. From these, each definition file itself selects its visible foreign packages with imports; definitions of its own package are visible without import. Versions and digests are not repeated in every definition file. Section 8 separates these two tasks.

<a id="core"></a>

## 3. The fixed beginning: Kind and Property

The fixed Core in the Package `markitect/core` consists of exactly nine definitions: the Kinds `Kind` and `Property` and the seven Properties `name`, `type`, `minCount`, `maxCount`, `values`, `target` and `properties`. Section 3.1 defines their complete contents.

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

<a id="core-definitions"></a>

### 3.1 Complete Core in nine definition files

All nine files belong to the package markitect/core. They need no imports because their reference targets lie in their own package. Each file has the same complete envelope. The effective field name of each meta-Property corresponds to its short local ID; spec.name is therefore omitted throughout.

Language note (DEC-069): the bodies of these nine definitions are the English wording of `markitect-core` 1.1.0, translated with unchanged meaning from the German bodies of 1.0.0 ([DEC-069](../concepts/register.md#dec-069-documentation-is-written-in-english)). Only this wording is the fixed Core. The German wording of 1.0.0 is preserved at the [archive point](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/language-contract.md#core-definitions).

In the comparison with this fixed self-description, too, missing imports and explicitly set `imports: []` are equivalent according to Section 5.2. The authored form is preserved; an erroneous entry or a non-empty import list is not treated as an empty list.

The comparison checks the effective content according to the general language rules: references in kind, target and properties are compared by their resolved Package/ID targets; effective default values and unordered declarations follow Sections 5.2 and 8.3. Short, fully qualified and valid-import-alias-qualified references are equivalent in the Core as well. Imports are checked completely, but after successful resolution they are not an additional content field. A still valid unused import does not change the self-description. Spelling errors, ambiguities, invalid imports, duplicates and forbidden fields remain errors. The complete bodies and all other content values must still correspond to the fixed Core; the source files are not rewritten.

**kind.yaml**

```yaml
id: Kind
kind: Kind
body: >-
  Declares a type. properties names the permitted fields of the spec
  of its definitions. The body of a type definition describes their
  shared meaning and applicable stipulations.
spec:
  properties:
    - properties
```

**property.yaml**

```yaml
id: Property
kind: Kind
body: >-
  Declares a field contract with value type and cardinality.
  The body describes the meaning of its values or relationships.
  For type object, properties names the permitted fields of the object value.
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
  Optional explicitly deviating non-empty field name.
  If omitted, the complete local ID of the Property applies as the field name.
  The effective field name must be unique within every contract
  that uses it.
spec:
  type: string
  minCount: 0
  maxCount: 1
```

**type.yaml**

```yaml
id: type
kind: Property
body: The value type of this Property, as understood by the Core.
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
body: Non-negative minimum number of present values.
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
  Maximum number of present values. The value -1 permits any number of values
  within the tool limits. Other values must be at least minCount.
  Exactly 1 denotes a single value; other maximum values denote
  a list.
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
  Permitted string values of an enum Property.
  For type enum, a non-empty list of distinct values is required.
  For other value types, this entry is not permitted.
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
  The Kind required of the referenced definitions.
  For type reference, this entry is required.
  For other value types, it is not permitted.
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
  References the permitted field contracts of the object described here.
  The effective field names must be unique within this contract.
spec:
  type: reference
  target: Property
  minCount: 0
  maxCount: -1
```

<a id="property-contracts"></a>

### 3.2 Shared Property on the Kind and on the object

The fixed Core Property `properties` declares a list of standalone property definitions. On the Kind it determines the permitted fields of the `spec` of its definitions; on an object Property it determines the permitted fields of its object value. This meaning follows exclusively from the bound meta-identities.

| Definition considered | Meaning of the shared Property |
| --- | --- |
| kind is the fixed meta-Kind Kind | properties declares the permitted fields of the spec of the definitions of this Kind. This also applies to kind.yaml and property.yaml themselves. |
| kind is the fixed meta-Kind Property and spec.type is object | properties declares the permitted fields of the respective object value. |
| kind is the fixed meta-Kind Property with another spec.type | properties is not permitted as a contract entry, not even as an empty list. The already existing value type condition remains in force. |
| kind is an ordinary project Kind | Its own field contract applies. A field named properties permitted there, even when the shared Property is reused, is ordinary data content and creates no additional type or object contract operator. |

The last row prevents an impermissible global reservation of the field name. A project Kind SchemaCatalog could declare properties as a list of property references and use it to describe concrete catalog entries. Such a catalog entry does not thereby become a Kind; only a definition of the fixed meta-Kind Kind declares a new type contract. The fixed language meaning depends on exactly bound meta-identities, not on written names or an arbitrary import alias.

<a id="body"></a>

## 4. Binding prose and scope of validity

body is part of the applicable model contract and is preserved in the bound content.

**Non-empty content:** The body as decoded from YAML MUST contain at least one character that does not belong to the Unicode property White_Space in the bound version 15.1.0. An empty string and a string consisting exclusively of such whitespace characters are errors. The same character set applies as for link display texts in Section 4.5, not a language- or runtime-specific strip/isspace definition. The check does not trim, replace or normalize the text. Leading, inner and trailing whitespace are preserved completely and enter the canonical body value unchanged. This content condition does not extend any Markdown or body link grammar and does not apply generally to ordinary string values or the optional title.

- The body of a Kind describes the shared meaning and applicable stipulations of its definitions.
- The body of a Property describes the meaning of its values and relationships in the contract that uses it.
- The individual body supplements the stipulations of the concrete item.
- Individual entries do not silently cancel general obligations. Contradictions are treated as findings to be clarified.

The text can describe model conditions, desired runtime behavior, boundaries and conditions of its applicability. A statement that cannot be formally checked remains binding. The Core has no special requirements, roles or subject sections and no general formal rule language. A project can use such words as its own terms without thereby obtaining special compiler semantics.

Relationships obtain their domain meaning from the type and property contract. A statement about requester concerns the application referenced there in the described exchange. Neither global rule applicability nor transitive composition, agent partitioning or write rights follow from a relation name.

### 4.1 One link format for all definitions

The authored form is `[[target]]` or `[[target|display text]]`. The target is an id available in the definition's own Package or through direct imports, a complete reference package/id or an explicitly alias-qualified reference Alias.id. Resolution takes place in the package of the definition whose body contains the link and in that definition's file-local imports. It uses the same fixed package bindings as kind and the reference values in spec. The display text does not influence resolution. Without explicit display text, the written target address is displayed.

Every Property is addressable via its explicitly declared local id. There is neither a separate property link syntax nor automatic search for a field name. A field label does not automatically become an additional Property ID.

### 4.2 Declaration and concrete value

A body link to a Property denotes its standalone definition, not a concrete field value and not an access expression. A concrete reference value is set in `spec`. The surrounding sentence carries domain relationships; if prose and `spec` do not agree, this remains a semantic contradiction. There is no automatic precedence rule.


### 4.3 The fixed meaning of the link

The Core knows exactly one additional relationship from this markup: **The source definition explicitly refers to the target definition at this body position.** It retains the ordered occurrences with source definition, canonical package/ID target and optional display text. Repeated links remain individual occurrences; an aggregated graph view may represent them as one reference with multiple occurrences. For this there is no second manual uses/references list and no new built-in domain Kind. Exact diagnostic coordinates are not a prerequisite of this relationship.

The domain statement arises from the sentence, the Kind contract and the property contract. The following limits follow from this:

- A Kind link does not type the source. The field kind serves that purpose.
- A Property link explains or uses the field contract; it does not enter a value into spec and does not count as fulfilling a Property cardinality.
- A definition link alone declares neither containment, runtime invocation, positive dependency nor global rule applicability. Even “does not use [[dms]]” contains an explicit reference to dms.
- Referencing a rule does not automatically activate it for the whole linked item. The applicable scope must emerge from its contract and the statement.
- Binding unmarked prose remains part of the model. Links help with unambiguous designation but replace neither the statement nor a completeness assessment.

The bound meaning of the link target belongs to the understanding of the sentence. Mere reachability in the graph does not transitively import all obligations of all reachable definitions.

### 4.4 Checking, renaming and model state

The compiler can recognize active links, resolve their targets unambiguously after the definitions have been loaded completely, and preserve provenance. A missing target or unknown import is an error; title, display text and expected meaning do not serve as a fallback search. Unlike with a typed reference Property, any definition is initially permitted as a target in a body link. Whether the sentence means the right Kind, the right Property or the right concrete definition remains the subject of semantic review.

The fixed model state contains the complete body and the resolved target identities. Changes to prose or display text are preserved even if the set of targets stays the same. Source positions belong to diagnostic provenance, not to logical target identity. Alias changes do not change the canonical link target; a change of the stored body can nevertheless be visible in the content comparison. The digest encoding is in Section 8.3.

The compiler checks the existence of the references, not the truth, completeness or correct interpretation of the statements. Agents can propose missing or wrong links; only a change adopted in the source changes the canonical reference. Link proposals from word similarity are not model edges.

<a id="body-links"></a>

### 4.5 Binding body link and Markdown profile

body uses [CommonMark 0.31.2](https://spec.commonmark.org/0.31.2/) with the explicit extension for body links. These rules apply exclusively to body. title and other string values are not examined for model links.

Processing begins with YAML decoding according to Section 5.1. Then the decoded body is read as Markdown with the body link extension. YAML escapes can therefore already produce characters of the link notation before Markdown processing. By contrast, no new model links are produced from later Markdown character references such as &#91; or from rendered text. The original body is preserved; normalization for Markdown processing is not an automatic change of the source.

| Context | Behavior |
| --- | --- |
| Ordinary text, heading, list, quotation or formatting outside a link | The unprotected double-bracket notation is active. |
| Inline code and indented or fenced code block | Literal content, no model links or link syntax errors from it. |
| Ordinary Markdown link or image, including text, destination and reference definition | No body link evaluation within this construct. |
| Autolink, HTML tag, HTML comment or raw HTML block | No body link evaluation in it. Normal Markdown text between inline HTML tags continues to follow its normal rules; link elements do not form a nested model link. |

In ordinary text, the active double-bracket notation takes precedence over ordinary Markdown link recognition: `[[api]](Text)` contains the model link api followed by normal text. A link or image already begun with ordinary Markdown syntax, by contrast, protects its content. There is no additional image/embedding form for model definitions.

**Grammar:** An active link has exactly the form `[[target]]` or `[[target|display text]]`. The target is a complete authored reference according to Sections 7 and 8. It must not be empty and contains no surrounding spaces; it is neither trimmed nor rewritten. NFC, declared spelling, import boundaries, collision and ambiguity rules apply unchanged. The link can address any definition category. Field paths, URL targets, title or similarity search are not added.

The optional display text is plain Unicode text. It is preserved after the escapes below and is not evaluated again as Markdown, character reference, model link or expression. It must not be empty or consist exclusively of Unicode White_Space characters of the bound Unicode version. Formatting stands outside, for example `**[[api|Cloud-API]]**`. Within an active link, LF, CR, NEL, LINE SEPARATOR and PARAGRAPH SEPARATOR as well as nested unprotected brackets are excluded.

The preservation requirement explicitly includes U+0000 (NUL) if it is present in the display text after YAML decoding. There it MUST NOT be replaced by U+FFFD or another character. The CommonMark normalization for recognizing the surrounding Markdown context does not change this literal value of the body link extension. An actually written U+FFFD remains distinct from NUL. For example, `body: "[[api|A\0B]]"` decodes to a link with the display text characters U+0041, U+0000 and U+0042. This introduces no additional body link escape; `\0` within an already YAML-decoded link remains impermissible according to the following escape rule.

**Escapes:** Outside the link, the CommonMark rules apply. `\[[api]]` makes the notation literal; a backslash protected by a further backslash does not prevent a link that then begins normally. In the display text, exactly `\|`, `\[`, `\]` and `\\` are permitted for the corresponding literal characters. Other backslash sequences inside the link are errors. In the target, no escapes are needed or permitted; its characters follow the name grammar. An additional unprotected |, an unprotected single bracket or an active [[ without matching closure is a markup error. An isolated ]] outside an active link remains ordinary text.

A link must be single-line in the decoded body. The authored source text may be folded by YAML over several file lines as long as no forbidden line break character arises in the decoded link. Invalid active notation is a structural error and is not silently downgraded to text. Character sequences of the same kind in the protected contexts, by contrast, remain literal.

**Occurrences and diagnostics:** For the language, complete link recognition, unambiguous target resolution and the order and repetition of occurrences are binding. The language prescribes neither UTF-8 byte ranges nor line/column coordinates. The separate [diagnostic contract for tools](diagnostic-contract.md) determines the exact coordinate form for tools that offer this profile. These coordinates are provenance, not target identity and not canonical model content. Missing or unverifiable diagnostic coordinates alone MUST NOT prevent a successfully completed content check and its digest. Incomplete link recognition, target resolution or other content checking continues to prevent canonical output for the affected model scope.

<a id="values"></a>

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

- spec.name is optional. If omitted, the effective field name is exactly the complete local id of the property definition. Package name, import alias and reference spelling do not belong to this field name; prefixes of the local ID are not cut off. Dots are excluded in local IDs and field names.
- An explicitly set spec.name must be a non-empty string; null or an empty entry trigger no fallback. The effective field names must be unique within each Kind or object contract.
- minCount is at least 0. maxCount is -1 or at least minCount.
- maxCount=1 denotes a single value; other maximum values denote a list. -1 sets no language-level maximum count; resource rules are in Section 5.3.
- An omitted entry counts as zero present values. The data value null is not permitted.
- reference requires target. The target of target is a valid type definition.
- enum requires a non-empty list of distinct permitted values.
- object uses the standalone property definitions referenced under properties. Undeclared object fields are errors. A concrete object value contains values of these fields, not new property definitions.
- values, target and properties are permitted only with the value types intended for them. A new type name installs no parser or operator.

A type reference is described by reference with the target Kind Kind; kindReference is not one of the value types. The type references in the shared field kind remain explicitly checkable as such.

Cardinality bounds the number of values, not their numeric value range. A domain statement such as “maxAttempts must be positive” still requires semantic assessment as long as no additional formal value condition has been introduced.

<a id="yaml-values"></a>

### 5.1 YAML and value profile

The authored form uses the following restricted profile of [YAML 1.2.2](https://yaml.org/spec/1.2.2/). This profile determines the reading form independently of library and implementation language.

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

There is no language-level 64-bit range or fixed rounding context. Finite tool limits follow Section 5.3; they must not deliver a rounded or truncated number as a successful evaluation.


<a id="lists"></a>

### 5.2 List order and missing versus empty lists

Ordinary value lists are ordered. Declarative enumerations of the fixed Core and the import list are unordered. A missing optional list and an explicitly empty list have the same canonical value. The original authored form is preserved as source information.

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

<a id="cycles-resources"></a>

### 5.3 Object contract cycles and resource limits

Ordinary definition references may form cycles, including the fixed Core beginning. Reference targets are resolved as identities and are not replaced by textual or value-side expansion. The following separate condition applies to object contracts.

An object contract graph has the object property definitions as nodes. An edge leads from the object contract to every object Property declared in it as a field. An edge is mandatory if the target Property has minCount>0; this applies equally to single values and non-empty lists. The subgraph of exclusively mandatory edges must be acyclic. Thus every permitted object cycle contains at least one optional edge with minCount=0. Unused object contracts are also checked for mandatory cycles; a contract that is impossible to satisfy does not become valid through the absence of concrete instances.

| Case | Language rule |
| --- | --- |
| Optional object Property uses itself | Permitted; the value can end without this field or with an empty optional list. |
| Mutual object contracts with at least one optional edge per cycle | Permitted; finite concrete values are possible. |
| Object cycle exclusively via mandatory fields | Error; a finite value cannot fulfil this obligation. |
| Ordinary reference Property leads to an already referenced item | Permitted definition reference; no object expansion. |
| Empty list with minCount=0 and maxCount=0 | Permitted; no object values are produced or visited. |

Object values remain finite, closed YAML values according to Section 5.1. Value checking and canonical projection follow exclusively values actually present; they do not expand the contract graph in advance and do not produce missing optional objects. The preceding rules for missing optional lists apply unchanged within present objects. The prohibition of YAML anchors and aliases remains in place.

**Resource policy:** The language sets no additional fixed size, quantity or depth maxima. Tools may use finite processing budgets but must explicitly state the scope and values of these budgets and name them in the check report. A mere limit overrun is not evidence of a model violation: the affected content check reports not_completed and produces for this model scope neither canonical bytes nor digest nor completeness evidence. A language error already established remains an error even if a further check aborts; the abort is additionally reported. An independent, completely checked model scope can still obtain a result.

Completely finished runs over the same bound inputs must, with sufficiently large budgets, deliver the same canonical bytes and digests. Budgets and runtime capacities belong to tool configuration and check provenance, not to the canonical model projection. There is no silent truncation, skipping or rounding as a successful evaluation. Runtime-related recursion or memory aborts during the content check are likewise reported as incomplete evaluation. Pure coordinate aborts concern exclusively the separate diagnostic result.

The rules of this section concern definition references and object contracts. For the separate model.yaml dependency graph, the acyclicity rule from Section 8.1 applies; a cyclic definition reference is not a self-dependency of the model.

<a id="property-names"></a>

## 6. Standalone Properties and one file per definition

Every Property is a complete definition according to Section 2. There are no mandatory prefixes for its ID. Property definitions are explicitly referenced from the respective Kind or object contract; field keys of concrete values are not additional definitions.

### 6.1 Short Property ID with default field name

`spec.name` is optional. If omitted, the local Property ID is used exactly as the field name; `requester` yields `requester`. Prefixes are not cut off. A set `spec.name` determines a deviating field name and must satisfy its spelling convention. Neither the ID nor the field name may contain dots. Both forms are subject to the same uniqueness check per field contract. Identity and field name remain distinct concepts; migrations follow Section 9.


<a id="names"></a>

## 7. Identities and references

Package is exclusively a namespace. Every folder under the explicit model root forms a Package; its relative path determines the name. There are no additional namespaces. Packages are structured with `/`. Local IDs are unstructured names; in definition references, dot notation serves exclusively for alias qualification. Complete local IDs of imported Packages are directly available.

The logical identity remains **package ID plus stable local id**. It contains neither Kind nor display name, file name or version number. The folder path does, however, determine the package ID; a different Package folder therefore changes the identity. A specific revision is additionally determined by its model state and that model state's content bindings. The authored reference package/id or Alias.id is resolved to this canonical pair.

| Item | Package | Local id |
| --- | --- | --- |
| Exchange Kind | example/connector/vocabulary | Exchange |
| Requester Property | example/connector/vocabulary | requester |
| Concrete exchange | example/connector/project | list-documents |
| Concrete API | example/connector/project | api |

A local ID is a single name according to the convention of its definition category. It contains neither a dot nor `/`. The Package takes over the domain structuring. An import makes local IDs directly available. In definition references, `/` separates the Package from the local ID; a dot separates an explicitly declared import alias from the local ID. For example, `Core.Kind` denotes the ID `Kind` from the Package imported under `Core`, not a local ID named `Core.Kind`.

For identity and collision checking, case makes no difference. Within a package, therefore, IDs of different Kinds also must not be distinguishable solely by case. The permitted authored form is nevertheless strict and must be adhered to exactly: a deviating spelling is an error, not an accepted alternative reference. The verifier does not automatically correct or normalize any source file. Readable IDs are not changed at every change of the preferred designation, nor deliberately reused for other items. An optional title is not an alternative address.

Resolution of authored references uses directly available names and, where needed, explicitly qualified names. Only the definition's own package and the imports declared in the respective definition file are visible; supplying a model alone opens no foreign names:

- A complete id is resolved against the definitions of the definition's own package and of all packages directly imported in this definition file. Exactly one canonical target must match. Example: Kind comes from the imported Core package, Exchange from the imported vocabulary, api from the project's own package.
- Several different targets with the same visible id are ambiguous even if they differ merely in case. There is neither a silent precedence of local definitions nor a selection by the expected Kind or the order of the imports. The reference must be explicitly qualified or the name collision resolved.
- Full qualification uses `package/id`, for example `markitect/core/Kind` or `example/connector/vocabulary/requester`. The last `/` separates Package and local ID. A definition's own Package is always also available fully qualified, without an import of its own. A foreign Package is available fully qualified only after explicit import. Alias qualification has exactly the form `Alias.id`, for example `Core.Kind` or `Connector.requester`, with exactly one dot. Neither alias nor local ID itself contains dots. There is no stepwise namespace or file path search and no field dereferencing.
- Identical visible short IDs do not by themselves make the import selection invalid. Only the use of a reference spelling with several different target identities is an error. An unambiguous alias or full qualification denotes the desired own or imported target. A local ID cannot syntactically coincide with an alias qualification.
- Only direct imports make names available. Further supplied Packages and the contents of transitive model dependencies are not automatically included in the visible namespace of a file. Access to them requires a separate explicit import in the using definition file.

All reference positions use this target resolution: kind, Property.target, reference values in spec and body links. kind and Property.target must denote type definitions; a concrete requester value must, for example, denote a Container definition. The expected target type is checked only after unambiguous resolution and is not used as a search heuristic.

Missing or ambiguous names, unknown import aliases and contradictory import entries are errors. There is no search outside the definition's own package and the direct imports of this file, no automatic version selection, no file path interpretation and no resolution via similar names. After resolution, canonical package/ID targets are used. Short and qualified references may denote the same target but do not create different identities.

<a id="spelling"></a>

### 7.1 Binding spelling conventions and collision checking

Case does not create a different identity. The spelling conventions are enforced as language rules. Violations are rejected; a hint at the expected form is permitted, a silent correction or further processing of the deviating form is not.

| Language identifier | Prescribed form | Examples |
| --- | --- | --- |
| Package name | Lowercase components derived from the folder path, with / as separator | markitect/core, example/connector/vocabulary |
| ID of a Kind definition | PascalCase | Kind, Property, Exchange, MqttService |
| ID of a property definition | camelCase | requester, maxAttempts, messageId |
| Effective field name, including explicit spec.name | Single camelCase name without dots | requester, timeoutSeconds |
| ID of a concrete item | lowercase kebab-case | api, list-documents |
| Import alias | PascalCase | Core, Connector |
| YAML file name | lowercase kebab-case with .yaml | exchange.yaml, max-attempts.yaml |
| Model ID | Lowercase Unicode segments, separated by . or -; digit start allowed, see Section 8.1 | markitect-core, markitect.core, 2026-modell |
| Version label | The same character grammar as model ID; additionally no range notations or latest | 1.0, 1.0.0-rc.1, 2026.10 |

The category of a definition follows from its resolved typing, not from the case of a written kind value. A Kind definition is a definition of the fixed meta-Kind Kind; a property definition is a definition of the fixed meta-Kind Property. Acronyms follow the same word spelling: MqttService and messageId, not MQTTService or messageID. A reference must use exactly the declared spelling of the unambiguous target. kind: exchange instead of kind: Exchange is thus an error, even if the diagnostics can find the intended target. Likewise, inappropriately spelled package names, aliases, field keys and file names are errors. Language keys such as id, imports, package, as and minCount must be written exactly.

Local IDs of all definition categories and effective field names MUST NOT contain dots. This applies to Kinds, Properties and concrete items as well as to explicitly set `spec.name` and object field contracts. A deviating name entry cannot make an impermissible ID valid. The existing PascalCase, camelCase and kebab-case conventions continue to apply. A domain subdivision is expressed in the Package, not through structured IDs. Violations are rejected; dots are neither removed nor replaced by other characters.

Package/alias qualifiers and local ID are resolved by the declared imports and actual targets, not by their case or the expected target type. There is no selection by longest prefix or random order.

Collisions are checked independently of case: package IDs, definition IDs within a package, aliases within a definition file, effective field names within a Kind/object contract and file names in the respective package. Kind and Property do not get separate ID namespaces in this. In particular, Command and command in the same package are a conflict, even if each individually satisfies the convention of its category. Different file names do not resolve this definition conflict.

The comparison serves exclusively identity, conflict detection and diagnostics. Source text, stored canonical spelling and references are not automatically rewritten. A wrong reference with an unambiguous target is a spelling error; several target identities that match after normalization remain ambiguous and are not selected by the expected category or an exactly written subset. Data values, enum values, body, title, topic patterns and operation identifiers are not generally compared case-insensitively; the respective data contract applies to them.

The Unicode basis and the exact name comparison are in Section 7.2. The earlier, restricted ASCII profile is superseded. The permitted characters follow the Unicode profile bound there; body link escapes are stipulated in Section 4.5; exact source positions follow the separate [diagnostic contract](diagnostic-contract.md). The YAML/value profile is in Section 5.1.

<a id="unicode"></a>

### 7.2 Unicode, normal form and name comparison

Unicode is the text basis throughout. All files use UTF-8. Unicode is permitted in body, title and string/enum values as well as in package names, local IDs, aliases, effective field names and YAML file names. The fixed language keys and structural characters such as /, dot, hyphen, .yaml and body link delimiters keep their exact spelling and meaning.

The character tables are fixed to Unicode 15.1.0, corresponding to [UAX #31, Revision 39](https://www.unicode.org/reports/tr31/tr31-39.html). A change of tables is an explicit change of the language profile. Operating system, runtime or locale settings must not replace this binding.

The following profile applies to identifiers:

- A name word begins with XID_Start and continues with XID_Continue. Connector_Punctuation, including underscore, is excluded; this preserves the spelling conventions already chosen. Spaces, emoji and arbitrary symbols thereby do not become name characters. In normal text values they remain usable according to the YAML profile from Section 5.1.
- `/` structures Package names; every Package component is non-empty. A single folder name, a local ID, an alias and an effective field name contain no dots. Package names are relative logical paths without leading or trailing `/`, empty components, `.` or `..`, backslash, drive prefix or URL decoding. No file system path normalization takes place. Aliases are single PascalCase names. In concrete IDs and file names, the ASCII hyphen separates non-empty lowercase words; after a hyphen, a part may also begin with a Unicode decimal digit, for example `größe-2`. The first word still begins with XID_Start. The dot notation `Alias.id` is a reference form; dots in model IDs, version labels, decimal numbers and ordinary text values follow their own rules.
- PascalCase and camelCase are checked by the first cased letter of the name. Titlecase characters count as an uppercase initial. Without such letters, this initial requirement is dropped, for example with 文档. For lowercase names, all cased letters must be lowercase. In PascalCase/camelCase words, consecutive uppercase letters remain excluded; combining marks are skipped in this check. Thus Überweisung, gültigBis and MqttService remain fitting examples. These categories continue to follow from the typing.

**Strict authored form:** Names, file names and written references must already be in [Unicode Normalization Form NFC](https://www.unicode.org/reports/tr15/). A decomposed spelling, for example U+0075 U+0308 instead of U+00FC for ü, is rejected with a diagnostic. Neither a source file nor its decoded name is silently corrected. A reference must furthermore use exactly the declared spelling of the unambiguous target. Equivalence in the collision comparison does not make a deviating authored reference valid.

**Collision key:** For names we use Unicode Canonical Caseless Matching with full Default Case Folding, culture-independent and without Turkish special handling. Precisely, K(s) = NFD(toCasefold(NFD(s))) applies according to definition D145 of the Unicode Standard. Two names collide exactly when their keys are equal, each within the already stipulated scope: package, definition, file-local alias, field contract or file selection. The comparison key is an internal means of checking, not a new stored authored form. In particular, straße and strasse as well as Straße and Strasse collide. A corresponding reference deviation is also rejected; the verifier does not select an exactly written subset from several colliding targets.

NFC and this comparison do not cause general transliteration: ä and ae remain different. An additional NFKC compatibility folding is not chosen. body, title, string and enum values are neither normalized nor compared case-insensitively; there the decoded text is preserved and the respective data contract applies. The NFC requirement also concerns field keys, but not their ordinary text values. Provenance and original file bytes remain available for diagnostics.

The shared grammar for model IDs and version labels is in Section 8.1. It uses the same Unicode tables, NFC authored form and collision key, but additionally allows Unicode decimal digits at the beginning of each segment. This extension does not automatically apply to Package folders, definition IDs, import aliases or file names.

<a id="packages"></a>

### 7.3 Folders and Packages

The canonical model is structured in folders like documentation. The explicitly determined model root is itself not a Package and must not directly contain any definition files. Below this root, every folder is a Package, including an intermediate folder without definitions of its own. Every definition thus lies in a named subfolder and belongs exactly to the Package of its immediately containing folder. Subfolders belong to their own Packages; recursive discovery does not make their definitions members of the parent Package. The Package of the parent folder is not inherited.

**Authoring convention:** Packages are cut by domain context or feature (package by feature / vertical slices). Kinds, Properties and concrete items that belong together stand together with the respective feature. Their type of definition is no reason for separate folders such as kinds/, properties/ or instances/. A further subfolder corresponds to a domain subarea; every definition keeps its own file. This convention is a decision on model organization, not an additional Core concept and not a syntactic prohibition of certain folder names. The structure verifier does not derive domain appropriateness from a folder name.

The Package name corresponds to the non-empty path of the immediately containing folder relative to the model root and is not additionally maintained in a package.yaml or in each definition. There is neither an empty Package name nor a Package named . nor a substitute name from the model ID. The folder components are joined with `/`; this is directly the relative folder path.

A folder name corresponds to exactly one lowercase Package component as previously defined in Section 7.2; a dot in a single folder name does not create additional components and is impermissible in this form. Case and Unicode collisions are checked independently of the behavior of the file system, including straße/strasse. Names must already be in NFC and in the prescribed form; no automatic correction.

**File discovery:** Discovery MUST traverse the complete model tree including all subfolders; there are no implicit ignore rules. Every subfolder remains a Package according to this section. The following rules apply exhaustively to regular files:

- Every file with the exact extension `.yaml` is a definition file and MUST satisfy the complete definition envelope. An additional YAML data file without a definition envelope is an error and is not silently skipped.
- Excepted are exclusively the explicitly bound `model.yaml` manifest and its `model-lock.yaml` immediately next to it, provided they lie in the model tree. The exception applies to exactly these paths with exactly written file names, not across the board to files of the same name in Packages.
- `.yml` as well as case variants of the extensions `.yaml` and `.yml` are errors. They are neither ignored nor renamed nor accepted as an alternative authored form. For valid definition files, the strict file name rules from Section 7.1 additionally apply.
- Other regular files, for example Markdown, images or JSON, are not read as definitions and do not enter the canonical model content. The redirection prohibition from Section 8.1 continues to apply to the entire discovered model tree.

The definitions are discovered from their respective folders. There is no separate definitionFiles list per Package and no additional mandatory definitions/ folder. The local id is still stated explicitly in every definition file; it is not derived from the file name. Renaming a file in the same folder preserves its identity. Moving a file into another Package folder or renaming/moving a Package folder changes the affected qualified identities and requires adjusted references. A different physical storage location of the entire model root preserves the relative Package names.

The folder hierarchy creates no implicit imports, inheritance, containment relationships between definitions or domain relationships. Parent, child and sibling Packages must be explicitly imported in every using file. A Package has no version, dependency list or publication file of its own. Model denotes the shared versioning and distribution unit and can contain several Packages. Section 8.1 determines model identity, root and manifest form. The model identity creates no automatic Package prefix and no additional part of the definition identity. Section 8.2 determines the lock format; Section 8.3 determines the canonical digest bytes.

The explicitly bound `model.yaml` and the adjacent `model-lock.yaml` are model metadata and not definition files; even with `root: .` they do not create a root Package. Definitions directly in the model root are errors.


<a id="imports"></a>

## 8. File-local imports and bound models

Only Packages with at least one definition of their own are importable. An empty folder or a Package that contains only occupied sub-Packages is not importable; its sub-Packages must each be imported themselves. The physical presence of empty folders therefore opens no names and establishes no import binding.

The optional envelope field imports contains the foreign Packages explicitly visible in this file. Without imports or with imports: [], exclusively definitions of its own Package are visible. An unknown Package, or one not supplied in the bound input set, is an error. A foreign Package must come from the own model or from an explicitly declared direct model dependency. Transitive models are loaded for contract checking; direct access to their Packages requires a direct dependency of its own in the using model manifest. Imports select neither a new version nor a physical file system path. The provisioning of an external model and its version alone open no names; conversely, an import is not an installation instruction.

**Import form:** Every import is a closed object with required `package` and optional `as`; further fields are errors. Both entries are strings. If `as` is present, it must contain a non-empty valid alias. A string short form is not kept in parallel. as declares a file-local name in PascalCase and changes neither Package identity nor the set of available definitions.

Without as, a Package can be qualified only under its complete Package name. With as, both the complete name and the short alias are permitted. The complete local IDs of the imported Package become directly available in both cases: Kind, Core.Kind and markitect/core/Kind can denote the same target. An alias does not make the import an import that is visible only when qualified.

Aliases apply only in the declaring definition file. Imports of other files, the imports of a referenced Kind/property definition, parent Packages and transitive model dependencies do not extend this view. Reference values and body links are always resolved in the file context in which they are written. Even several Packages of the same distribution unit need their own file-local imports.

Duplicate aliases and duplicate imports of the same Package are already rejected as declaration errors, independently of case. Definitions of the same name from different visible Packages, by contrast, are permitted. Every reference actually used must yield exactly one target; competing short, alias or complete spellings obtain no precedence by exact spelling, target Kind or prefix length. `Core.Kind` and `core.Kind` are not different addresses; with `Core` declared, `core.Kind` is a spelling error. Section 7.2 owns the Unicode profile.

**Content binding remains required:** External reusable models are bound with an exact version and a concrete content digest. Version ranges, latest and silent content exchange under a bound version are not provided for. The bound input set must supply every canonical Package/ID identity unambiguously; multiple imports or aliases do not circumvent a conflict. The local model is bound by its fixed source/model state; no local change requires a new publication number.

<a id="models"></a>

### 8.1 model.yaml: identity, root and dependencies

Every model, as one unit of Packages that belong together, has a manifest with the fixed file name model.yaml. Manifest denotes the role of this file; model is the shared versioning and distribution unit for several Packages. model.yaml is provisioning metadata, not an additional definition and not a new privileged Core Kind. Every actual definition keeps its uniform envelope.

| Field | Chosen contract |
| --- | --- |
| formatVersion | Required integer; binds manifest format and language contract. v1 uses 1. Unknown versions are not silently interpreted as 1. |
| id | Required stable, non-empty model identity; independent of checkout/installation folder, storage location, model root, Package names and version. |
| version | Exact textual publication version of all contained Packages. Optional for local unpublished work; required for publication and as a supplied dependency. |
| root | Required non-empty relative directory path, relative to the folder of the respective model.yaml. There is no guessed default folder. |
| dependencies | Required mapping from directly needed model ID to exact textual version; {} without dependencies. No Packages, import aliases, version ranges or latest. |

The envelope is closed: unknown or misspelled fields are errors. A dependency must use exactly the declared model ID. Model IDs and version labels are YAML strings; numeric values are not converted. The physical root path spelling is given further below in this section.

**Shared character grammar for model IDs and version labels:**

```text
Label             = Segment (("." | "-") Segment)*
Segment           = StartCharacter ContinueCharacter*
StartCharacter    = XID_Start | Unicode decimal digit (General_Category=Nd)
ContinueCharacter = XID_Continue
```

The Unicode profile from Section 7.2 applies with the following refinements:

- Connector_Punctuation, including _, is excluded everywhere. Cased characters must be lowercase; scripts without case remain permitted. All labels must already be NFC. Violations are rejected, not corrected.
- ASCII dot and ASCII hyphen separate non-empty segments. There are no leading, trailing or immediately consecutive separators, not even mixed as in 1.-0. Whitespace, /, backslash, emoji and other characters not belonging to the profile are impermissible.
- The same grammar applies to model identity, dependency key, published version, version requirement and lock binding. Model IDs, too, may begin with digits or contain dots. Dots in these labels create neither Packages nor namespaces; hyphens likewise carry no additional semantics.
- Valid are, for example, markitect-core, überweisungen, 文档, markitect.core, 2026-modell, 1.0, 1.0.0-rc.1 and ٢٠٢٦.١. Invalid are, for example, Markitect-Core, über_weisungen, modell/name, 1..0 and 1.0.0-.
- `x` is not reserved in version labels and has no placeholder meaning. `"x"`, `"1.x"` and `"release.x"` are ordinary exact text labels. A requirement for `"1.x"` requires exactly this version and is not satisfied by `"1.0"` or `"1.7"`. `latest` remains forbidden as a version label; `*`, `^`, `~`, comparison and range operators still do not belong to the label grammar. The shared grammar for model IDs remains unchanged.

The complete Unicode collision comparison from Section 7.2 applies to model IDs within the supplied set of models and to version labels within the same model. straße and strasse must not denote different identities there. Different models may use the same version label. Requirements and lock entries must nevertheless use exactly the declared authored form; a colliding spelling is not a valid alternative reference. The first language scope still binds only one version per model ID; the collision check opens no multi-version selection.

A version is an exact text label. Manifest, direct dependency requirement and lock use the same decoded text. Components are not compared as numbers, not supplemented, and establish neither order nor compatibility. `"1.0"` and `"1.0.0"` are different versions. Version ranges and `latest` are forbidden; the exact content is additionally bound by the digest.

A dependency requires the exactly declared identity and version of the supplied model version. Missing models, deviating spellings, contradictory direct or transitive version requirements and several candidates for the same model ID are errors. v1 binds exactly one version per model ID. External contents are additionally bound by the lock according to Section 8.2.

A Package occupied by definitions belongs to exactly one model in the bound input set. Two models must not split the same Package, not even with different local IDs. Shared intermediate folders remain possible: separate models with markitect/core and markitect/http can both contain the definition-less intermediate folder markitect. As soon as a Package contains definitions, these come from exactly one model; an identical name occurring elsewhere merely as an empty intermediate folder supplies no further definitions. No loading or import order decides membership.

**Cross-model name binding:** Only occupied Package paths and all non-empty intermediate paths derivable from them enter the shared Package namespace. Differently spelled names of this set that collide in the Unicode comparison are errors. Purely empty subtrees bind no names across models. For example, an occupied straße/feature in one model does not collide with an empty strasse/empty in another model. If, however, strasse/empty contains a definition, the derivable intermediate paths straße and strasse collide. Within each individual model tree, all folder names including empty subtrees continue to be checked for valid spelling, NFC and Unicode collisions. Every folder remains locally a Package; empty Packages remain non-importable.

Imports remain file-local. They make exclusively the actually imported Packages visible. A Package from the same model requires an import as soon as it is not the file's own Package. The availability of a transitive model opens no direct import without a direct model dependency of its own. model.yaml is supplemented neither from imports nor from prose; it is not an installation or execution instruction.

A different physical storage location of the entire model, a different installation folder or an identity-preserving relocation of its root do not change its model ID and the relative Package names. An explicit change of the model ID changes the address of the model dependency but does not automatically create new Package/definition identities. root determines which files are loaded under which relative Package names; a change is to be checked against this new inventory.

**Model dependencies:** The directed graph of the model.yaml dependencies must be acyclic. Nodes are the unambiguously bound model IDs, edges the explicitly declared direct requirements. In the complete reachable model scope, no dependency path may lead back to the originating model. This prohibits self-dependencies as cycles of length one just as it prohibits mutual or longer cycles. Even exactly matching versions or digests do not make such a cycle valid. Several models may use the same dependency, provided the shared exact version and content binding matches.

A model cycle is a language error and yields failed, not a merely incomplete check. For the affected invalid model scope there are neither canonical bytes nor digest. The rule needs no special handling for self-dependencies. Parts that are inseparably mutually dependent in domain terms can belong to a shared model with several domain Packages. Ordinary definition references and Package imports within the same model may continue to form cycles; the object contract rules from Section 5.3 apply additionally. No dependencies are supplemented from prose or mere file links.

**Physical root path rules:**

- root is either . or a non-empty relative folder path, relative to the folder of the respective model.yaml. The target must already exist as a directory; missing targets and files instead of directories are errors. . denotes exclusively the manifest folder itself.
- ASCII / is the only separator, also under Windows. Every relative component follows the existing lowercase Unicode Package folder grammar from Section 7.2: XID_Start at the beginning, then XID_Continue, no Connector_Punctuation. The new model ID/version label grammar does not extend this folder grammar. In particular, dots, hyphens and digit starts are not additionally permitted in individual root folder components.
- There are no leading or trailing /, empty components, embedded . or .., backslashes, drive prefixes or absolute paths. The authored form is checked before any path interpretation; ./model, docs//model and model/ are not silently normalized. There is no variable, home, URL or other substitution.
- The written components must already be NFC and correspond exactly to the actual directory names, including case and Unicode spelling. A Model findable on a tolerant file system does not replace a written model. Several directories matching in the Unicode collision comparison remain an error; the exactly written subset obtains no precedence.
- Symlinks, junctions and comparable reparse redirections in the relative root path or in the discovered model tree are forbidden and are not followed. The explicitly loaded manifest and lock files are not read via such redirections either. The physical manifest folder forms the reference point; the absolute storage path above it obtains no Package naming convention.
- Loaded model roots must neither denote the same physical directory nor lie within one another. The check takes place for the bound set of models before definition discovery. With root: ., further bound models must therefore lie outside this model tree. Shared logical, empty Package prefixes from separate physical roots remain possible.

Valid forms are ., model, docs/model and überweisung/文档, provided the exactly written directories exist. Invalid are, for example, ./model, model/, docs//model, ../shared/model, Model and C:\\models\\connector. The absolute storage locations of the models remain free; an identity-preserving physical relocation changes neither Package names nor canonical digest bytes. The root itself still forms no Package and must not directly contain any definition files (Section 7.3).

<a id="lock"></a>

### 8.2 model-lock.yaml: complete external content binding

The generated lock is stored as model-lock.yaml next to the entry manifest and versioned together with the model. It binds all direct and transitive external model dependencies exactly once. The own local model version is bound by the current source/model state and is not repeated as a separate external entry in the lock.

The closed form consists of exactly formatVersion and models. The current formatVersion is the integer 1. models is a mapping from model ID to an object with exactly version and digest. Without external dependencies, models: {} is the complete table. Every key is the exactly written NFC model ID; the Unicode collision comparison from Section 7.2 also applies here. version is the exact textual version. digest has the form sha256: with exactly 64 following lowercase hexadecimal digits. Unknown fields, spelling deviations, missing values and collisions are errors.

**Completeness:** Exactly the external lock table must arise from the direct requirements of the entry manifest and the bound model.yaml manifests of the dependencies. Missing, surplus or contradictory entries are errors. There is no additional manual list of the direct dependencies and no second description of the graph edges in the lock. One version applies per model ID; neither import aliases nor loading or table order resolve version or content conflicts.

**Checking:** Model ID, exact version and computed content digest must match simultaneously. The verifier checks all reachable manifest requirements and then every content binding. Other contents under the same bound version are rejected. A check run does not automatically write, supplement or update the lock. A missing lock is not replaced by a silently created new binding. Changes of the dependencies require a renewed check of the complete binding; local definition changes alone do not require a new lock. The root lock table determines the shared external binding of the evaluated model state; any further lock file of a dependency model that is shipped along does not silently replace or extend this table.

**Digest scope:** A model digest binds exclusively the own model content, including the own model metadata, declared direct dependency requirements, Packages and definitions. It contains no recursively embedded digests of foreign model contents. The lock table carries the complete concrete binding of the transitive contents. This avoids recursive digest definitions; ordinary definition references remain references to canonical identities and are not textually expanded infinitely. The exact projection, defaults, orderings and byte encoding are in Section 8.3.

**Provisioning:** Installation paths, storage locations and download addresses are provisioning/provenance inputs and not lock fields. The already chosen model ID is not supplemented by a storage location. Provisioning supplies contents whose identity, version and digest must be checked; the lock entry alone is not an installation instruction.

<a id="canonical-bytes"></a>

### 8.3 Canonical bytes and SHA-256: model-digest/1

Every model digest is SHA-256 over the following canonical bytes. The encoding describes the evaluated own model content. Before any projection, the selected model and its complete dependency closure are validated: manifests, spellings/collisions, fixed Core self-description, type contracts, values, cardinalities and body links. Errors detected during loading must not be lost through a later map representation either. An invalid version obtains no digest; an unfinished content check is not valid digest evidence. A valid dependency model can be checked and hashed independently of an erroneous local model that uses it.

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

**Excluded data.** File names, root, installation/download paths, source bytes, comments, locations, check reports, lock files and foreign model contents are not fields of this projection. In particular, an own digest contains no recursive foreign content digests; the concrete external binding remains in the lock. Empty Packages are not included; intermediate Packages result from the complete Package paths of the contained definitions. The same occupied paths and their derivable intermediate paths also determine the cross-model name binding according to Section 8.1. Empty Packages are not importable; adding or removing locally valid empty subtrees therefore changes neither a permissible import binding nor the cross-model collision check. This does not change the authoring rule that every folder is a Package, and it does not allow local spelling or collision errors to be circumvented in the preceding validation. Git need not preserve empty directories in order to produce the same canonical content bytes.

<a id="changes"></a>

## 9. Renames and migrations

| Change | Contract |
| --- | --- |
| Change title | Identity remains; the changed presentation can be captured in the new state. |
| Rename file within the same Package folder | Identity remains if id persists and the file continues to be loaded. |
| Move file into another folder or rename/move Package folder | Package name and thus identity change; migrate references explicitly. |
| Change the import alias of a file | Adjust only the authored references of this file qualified with it; other files, unambiguous directly available names and canonical target identities remain. |
| Change property field sender to requester | The new field key applies in the type contract; concrete specs must be migrated. |
| Preserve property identity while doing so | Keep id in the property file and set spec.name to the desired new field name; for example requester with stable ID sender. |
| Omit spec.name | The complete local Property ID is the effective field name. |
| Remove deviating spec.name | The field name falls back to the complete local ID; affected concrete specs must be migrated where necessary. |
| Change ID while spec.name is missing | Identity migration and change of the default field name; an explicitly set previous name can preserve the field key. |
| Use Property under another Kind | Explicitly reference the same standalone property definition if the same field contract is desired. |
| Change local id or package membership | Explicit identity migration and adjustment of known references. |

A stable Property ID preserves references to the field contract but does not keep the old field name valid. id is always explicitly required. An explicitly set spec.name separates the field name from this ID; without name, the ID applies as default field name. If an author additionally changes the ID, that is a separate identity migration; the compiler does not guess a renaming intention and creates no new Property ID on a field rename.

A new jointly versioned model version can contain the same logical identities with changed contracts. The new binding is chosen explicitly, and affected models or realizations are considered against it. Neither a stable ID nor a version number proves semantic compatibility.

Published old versions and evidence referring to them are not changed retroactively. Automatic redirects, hidden name histories and mapping via similar names are not part of the minimal Core. On a genuine identity change, known kind, target and value references and explicit body links must be migrated. Free prose additionally requires review.

<a id="processing"></a>

## 10. Processing and fixed Core rules

Processing takes place independently of document order:

1. Provide the known Core contract and the explicitly bound packages.
2. Capture exactly one complete definition from each definition file; check the uniform envelope, preserve provenance and reject identity collisions. Embedded property declarations and definition collections are errors.
3. Resolve references unambiguously in the definition's own package and the imports of the respective definition file.
4. Check meta-Kinds and property contracts against the fixed beginning and its language invariants.
5. Form closed type contracts from the valid Kind/property definitions; determine the effective field name from spec.name or the complete local Property ID and reject name collisions.
6. Check all definitions against their Kind, value types, cardinalities and reference targets.
7. Form a fixed model state with complete bodies, type contracts, package bindings and resolved references.

For this, the compiler must have a fixed, built-in understanding of the document envelope, the one-file rule, the two meta-Kinds, their field interpretation, value types, cardinalities and reference rules. This meaning does not arise from its being described again in the body of the fixed beginning.

The self-typed Kind and ordinary reference cycles are finite graph relationships. They require no infinite type expansion. Object contract cycles and tool budgets follow Section 5.3: only mandatory object cycles are impermissible; actually present finite values are checked without advance expansion. Resource aborts are not language maxima and must not yield a successful check.

The result makes Kind, Property and concrete definitions jointly addressable. A content digest binds the model content and the chosen contracts; source provenance remains traceable. The canonical data encoding is in Section 8.3. Production error codes and report formats remain separate tool work.

<a id="conformance"></a>

## 11. Conformance and boundaries

A complete result MUST check envelope, one-file rule, fixed Core self-description, names and collisions, manifests and roots, model dependency graph, imports, type/object contracts, values, cardinalities, references, body links and the external lock binding. Invalid inputs are `failed`; pure resource aborts are `not_completed` according to Section 5.3. A partial check MUST NOT be output as a completely valid model. For invalid model scopes, or model scopes whose content has been checked incompletely, there are neither canonical bytes nor digest. The separate diagnostic coordinate profile is not an additional content check; its incomplete status changes neither the result of completely checked contents nor their digest. A digest proves only the content of its model scope that has been checked according to Section 8.3.

Structural validity proves neither the truth or completeness of binding prose nor suitable modeling, correct realization, executed runtime evidence or human acceptance. Domain contradictions remain semantic findings. Derived checklists are not a second canonical truth.

Not part of v1 are compact or embedded definitions, additional namespaces, user-defined metaclasses, automatic name corrections or redirects, version ranges, multi-version imports and a general formal value-condition or execution language. Additional domain Kinds from libraries obtain no privileged compiler semantics.

Context, semantic indexing, change assessment, realization mapping, check planning, delegation, integration and Apply are not procedures defined by this specification. The [separate procedure proposals](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/procedure-proposals.md) have not been selected. Diagnostic profiles, output formats, deployment and migration remain separate work. They add no required Kinds and do not change the v1 rules.

<a id="121-stand-der-definition-des-kanonischen-modells"></a>

The chosen minimal language scope is completed as the v1 design baseline. Basic form, fixed Core, values and lists, Unicode and spelling conventions, identities and imports, Packages, models and roots, dependencies and lock binding as well as canonical bytes are stipulated. The known Command/command collision in the informative Exchange example is a vocabulary conflict and not an open language decision. Later implementation findings can require explicitly documented refinements or corrections; further language features require an explicit extension decision. The earlier closing reconciliation remains preserved in the [language history](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/minimal-language/language-history.md#121-state-of-the-definition-of-the-canonical-model).
