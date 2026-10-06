# Markitect Minimal Core Language

**Status:** Conceptual design baseline  
**Purpose:** Define the current minimal Core language for Markitect after reducing the model to the smallest set of concepts that appear necessary for the long-term projection/assurance vision.

---

# 1. Design goal

Markitect should keep its Core extremely small.

The Core is not responsible for understanding:

- DDD;
- Clean Architecture;
- Use Cases;
- Commands;
- Validators;
- Rules;
- Workflows;
- Projections;
- Evidence;
- C#;
- Java;
- Go;
- CI systems;
- Claude;
- Codex.

The Core should only provide the generic language required to define arbitrary engineering ontologies in a deterministic, typed, addressable way.

The working principle is:

> **The Core should understand structure and identity. Higher layers describe engineering intent. Modules and agents interpret and verify that intent.**

---

# 2. Minimal mental model

The current candidate model is:

```text
Schema
  contains Kinds

Kind
  describes Definitions

Definition
  contains Properties

Property
  has a type

Reference Property
  points to another Definition

KindReference Property
  points to a Kind
```

From the Reference Properties, the compiler can derive a typed semantic graph.

This is the entire conceptual basis of the current Core candidate.

---

# 3. Definition

A **Definition** is the universal canonical unit in Markitect.

> **A Definition states that something exists in the desired engineering model, what kind of thing it is, what it is called, why it exists, and what is declared about it.**

Examples:

```text
CreateOrder
Orders
Dossier
ReleaseProcess
ModuleIndependence
CreateOrderValidator
```

A Definition has the universal structure:

```text
Definition
├── apiVersion
├── kind
├── metadata
│   ├── namespace
│   └── name
├── purpose
└── spec
```

Example:

```yaml
apiVersion: software.example.org/v1
kind: UseCase

metadata:
  namespace: orders
  name: create-order

purpose: >
  Allows a customer to create an order.

spec:
  ...
```

---

# 4. Definition identity

A Definition is uniquely identified by:

```text
apiVersion
+
kind
+
namespace
+
name
```

Conceptually:

```text
DefinitionIdentity(
  apiVersion,
  kind,
  namespace,
  name
)
```

Example:

```text
software.example.org/v1
UseCase
orders
create-order
```

There is initially no second opaque UUID.

Changing one of these identity components changes the Definition identity.

A rename or move is therefore observed by the Core as:

```text
old Definition removed
+
new Definition added
```

A higher-level migration/change model may later describe semantic continuity explicitly.

Core does not infer continuity.

---

# 5. Mandatory `purpose`

Every Definition must have a `purpose`.

> **If something is important enough to exist canonically in Markitect, its reason for existence should be explicit.**

The purpose answers:

```text
Why does this Definition exist?
What responsibility does it have?
What contribution does it make to the larger system?
```

Good:

```text
Owns the accepted case-level state for one proceeding.
```

Poor:

```text
Implemented in C# and stored in PostgreSQL.
```

The purpose describes intent and responsibility, not incidental implementation.

This is also useful for AI agents: they should know not only what something is, but why it exists.

---

# 6. `apiVersion`

`apiVersion` identifies the semantic schema/version under which a Definition is interpreted.

Example:

```text
software.example.org/v1
```

It is not the revision of the individual Definition.

Keep these concepts separate:

```text
apiVersion
= semantic/schema contract version

Git revision
= source-history state

package version
= distributable package version
```

Git remains the source-history authority.

Markitect binds model state and evidence to Git revisions rather than duplicating version control.

---

# 7. `kind`

`kind` identifies the nominal semantic Kind of a Definition.

The full Kind identity is:

```text
apiVersion + kind
```

Therefore:

```text
software.example.org/v1 / Module
```

and:

```text
delivery.example.org/v1 / Module
```

are different nominal Kinds.

The Core does not know what either means.

That meaning is supplied by the Schema.

---

# 8. `namespace`

Namespace exists for:

```text
identity scoping
name resolution
organization
selection
```

It is not an implicit domain relationship.

Example:

```yaml
metadata:
  namespace: orders
```

does not mean:

```text
belongsTo → Orders
```

If ownership, membership, containment, or dependency is meaningful, it should be represented explicitly in `spec`.

Namespace is also not filesystem structure.

A matching folder layout may be convenient, but file paths are representations, not ontology semantics.

---

# 9. `name`

`name` identifies a Definition inside:

```text
apiVersion + kind + namespace
```

A Definition name is semantic.

It does not require its projected artifacts to use the same exact name.

For example:

```text
create-order
```

may be represented as:

```text
CreateOrder.cs
CreateOrderHandler
POST /orders
"Create order"
```

depending on projection strategy.

---

# 10. `spec`

`spec` contains the Definition's desired state beyond its universal identity and purpose.

Its structure is defined entirely by the Definition's Kind.

Example:

```yaml
spec:
  transactional: true
  belongsTo:
    namespace: commerce
    name: orders
```

The spec is closed-world.

If a Kind does not define a Property, that Property is invalid.

There is no:

```text
maybe some plugin understands this unknown field
```

Extension should happen explicitly through Schema/Kind evolution.

---

# 11. Schema

A **Schema** is a versioned collection of Kind declarations.

It answers:

> **Which Kinds exist under this apiVersion and how are Definitions of those Kinds structured?**

Conceptually:

```text
Schema:
software.example.org/v1

Kinds:
  Module
  UseCase
  Handler
  Validator
```

`Schema` replaces the earlier Core use of `Domain`.

This avoids confusion with Domain-Driven Design.

`Domain` may still be introduced by a DDD ontology if that architecture requires it.

---

# 12. Kind

A **Kind** describes one nominal category of Definition.

It answers:

> **What does a Definition of this kind structurally contain?**

A Kind should itself have a purpose/description explaining its meaning.

Example:

```text
Kind: UseCase

Purpose:
Represents one independently understandable unit
of application behavior.
```

A Kind declares its Properties.

It does not use inheritance.

It does not extend another Kind.

It does not implicitly inherit Properties.

---

# 13. Property

A **Property** defines one allowed piece of information inside a Definition's `spec`.

Example:

```text
UseCase Properties:

transactional
belongsTo
handledBy
validatedBy
```

A Property should minimally define:

```text
name
purpose
type
minCount
maxCount
```

Example:

```text
Property: transactional

Purpose:
States whether successful execution must be atomic.

Type:
boolean

minCount:
1

maxCount:
1
```

Properties are not globally addressable model entities.

They are part of a Kind declaration.

---

# 14. Property `type`

The word `type` is used for the technical form of a Property value.

This is intentionally different from `kind`.

- `kind` is the nominal semantic type of a Definition.
- `type` is the value type of a Property.

The initial built-in Property types are:

```text
string
boolean
integer
number
enum
object
reference
kindReference
```

These names are intentionally conventional and obvious.

---

# 15. `string`

Represents text, including natural-language semantic descriptions.

Example:

```yaml
rationale: >
  Accepted state and proposed state remain separate.
```

The Core understands this as a string.

It does not claim to understand the natural-language meaning.

Higher-level AI verifiers may interpret that meaning later.

---

# 16. `boolean`

Represents a true/false value.

Example:

```yaml
transactional: true
```

---

# 17. `integer`

Represents an integer.

Example:

```yaml
retryCount: 3
```

---

# 18. `number`

Represents a general numeric value when integer is insufficient.

---

# 19. `enum`

Represents one value from a closed set.

Example:

```text
mode:
  type: enum
  values:
    - command
    - query
```

---

# 20. `object`

Represents a local structured value without its own canonical identity.

Example:

```yaml
retry:
  attempts: 3
  delaySeconds: 5
```

Use embedded objects only for local structure.

A strong rule is:

> **If something deserves its own purpose, identity, relationships, reuse, or independent lifecycle, it should probably be a Definition instead of an embedded object.**

---

# 21. `reference`

A `reference` Property points to another concrete Definition.

Example Kind Property:

```text
belongsTo

type:
reference

target:
  apiVersion: architecture.example.org/v1
  kind: Module
```

Then a Definition may contain:

```yaml
spec:
  belongsTo:
    namespace: commerce
    name: orders
```

The compiler resolves that value to the full Definition identity.

A Reference is not just a string.

It is a typed semantic pointer to another Definition.

---

# 22. References create graph edges

The Property name itself is the semantic relationship.

Example:

```text
CreateOrder.belongsTo = Orders
```

creates:

```text
CreateOrder --belongsTo--> Orders
```

Similarly:

```text
CreateOrder --handledBy--> CreateOrderHandler
CreateOrder --validatedBy--> CreateOrderValidator
```

Therefore the current Core does not need a separate `Relation` declaration.

Reference-valued Properties naturally form the semantic graph.

---

# 23. `kindReference`

A `kindReference` points to a Kind instead of a concrete Definition.

This is useful for higher-level ontologies.

Example:

```text
An Invariant applies to all Definitions of Kind Module.
```

It should not need to list every Module Definition.

Conceptually:

```yaml
spec:
  appliesTo:
    apiVersion: architecture.example.org/v1
    kind: Module
```

The Core only verifies that this Kind exists.

The Core does not know what `Invariant` means.

---

# 24. `minCount` and `maxCount`

Multiplicity is expressed using explicit counts rather than Generics or a separate `Cardinality` concept.

Exactly one:

```text
minCount: 1
maxCount: 1
```

Optional:

```text
minCount: 0
maxCount: 1
```

Zero or more:

```text
minCount: 0
maxCount: unbounded
```

One or more:

```text
minCount: 1
maxCount: unbounded
```

This avoids introducing:

```text
Optional<T>
List<T>
Ref<T>
```

into the Core type system.

---

# 25. No Generics

The initial Core deliberately has no general Generic type system.

Do not model:

```text
Ref<Module>
List<Validator>
Optional<Text>
Map<K,V>
```

Instead:

```text
Property type
+
target Kind where relevant
+
minCount
+
maxCount
```

provide the required semantics.

This keeps the type system simple and predictable.

---

# 26. No inheritance or subtyping

The initial Core has:

```text
no inheritance
no extends
no implements
no nominal subtyping hierarchy
no trait hierarchy
no multiple inheritance
no override rules
```

Composition is preferred.

If several Kinds need similar Properties, that duplication is initially accepted.

This keeps every Kind declaration locally understandable.

If repeated real-world evidence later proves the need for reusable semantic contracts, that problem can be addressed explicitly rather than importing OO complexity preemptively.

Implementation-language inheritance remains possible as a projection concern.

---

# 27. No separate Relation concept

The Core currently does not need:

```text
Relation
```

as a separate language construct.

A Property whose type is `reference` already defines:

```text
source Definition
Property name / semantic relation
target Definition
```

This is sufficient to build the typed semantic graph.

Anything additional such as:

```text
acyclic
context traversal
impact propagation
```

belongs in higher-level intent, verification, or derived behavior unless future evidence proves otherwise.

---

# 28. No Invariant language in the Core

The Core currently does not need a built-in engineering `Invariant` DSL.

An `Invariant` can be modeled later as a normal Kind in a Foundation Schema.

Example:

```yaml
apiVersion: markitect.foundation/v1
kind: Invariant

metadata:
  namespace: architecture
  name: module-independence

purpose: >
  Preserve independent evolution of Modules.

spec:
  appliesTo:
    apiVersion: architecture.example.org/v1
    kind: Module

  statement: >
    A Module must not depend directly on another
    Module's private implementation.
```

The Core validates the structure.

A higher-level Assurance system decides how to verify the statement.

---

# 29. Intent and verification are separate

An Invariant, Rule, Process, or other higher-level Definition should not dictate that AI must verify it.

Instead:

```text
Intent
  ↓
Verification strategy
```

The verification mechanism should be chosen independently.

Examples:

```text
Simple type requirement
→ Core compiler

Dependency rule
→ static analyzer

Behavior requirement
→ executable test

Semantic correspondence
→ scoped AI verifier

Architecture ownership decision
→ human
```

This separation is fundamental.

---

# 30. Core responsibility

The Core compiler should answer only:

> **Is this canonical model structurally valid, unambiguous, and type-safe?**

It should validate:

```text
Definition identity
Kind existence
mandatory purpose
closed spec
known Properties
Property types
min/max counts
enum values
reference resolution
reference target Kind
kindReference resolution
duplicate Definition identities
duplicate Kind identities
deterministic normalization
```

Then it should produce:

```text
Normalized Definitions
Typed Reference Graph
Provenance
Diagnostics
```

---

# 31. What Core does not verify

The Core compiler should not attempt to answer:

```text
Does this C# implementation actually satisfy the UseCase?

Does the documentation still match the Dossier meaning?

Does Validation execute before mutation?

Does the pipeline correctly enforce the Release process?

Does this Agent follow the Rule?

Is the architecture a good architecture?
```

These are higher-level Assurance questions.

---

# 32. Three-layer model

The current architecture is best understood as three layers.

## Level 1 — Core / Type Safety

Question:

> Is the canonical model well-formed and unambiguous?

Mechanism:

```text
Definition
Schema
Kind
Property
type
references
counts
compiler
```

Deterministic.

## Level 2 — Engineering Intent

Question:

> What should exist and what should be true?

Possible Foundation / architecture / project Kinds:

```text
Rule
Invariant
Process
Projection
Obligation
EvidenceRequirement
UseCase
Module
Dossier
...
```

These are modeled using the Core.

## Level 3 — Assurance

Question:

> What evidence do we have that the materialized project still matches the intent?

Possible mechanisms:

```text
compiler
static analyzers
architecture tests
unit tests
integration tests
runtime probes
AI verifiers
integration verifiers
human decisions
```

---

# 33. Runtime/compiler concepts

The implementation will also need runtime concepts such as:

```text
DefinitionIdentity
KindIdentity
SchemaRegistry
ResolvedDefinition
Model
Graph
Provenance
Diagnostics
Snapshot
```

These are not necessarily user-facing Core language concepts.

They exist to implement compilation, resolution, graph construction, revision binding, and diagnostics.

---

# 34. Explicitly not Core

The following are not currently Core language concepts:

```text
Domain
Module
UseCase
Command
Query
Handler
Validator
Aggregate
Dossier

Rule
Invariant
Workflow
Skill
Agent
Process

Projection
Obligation
Evidence
Decision
Exception
Authority

C#
Java
Go
GitHub Actions
Azure Pipelines
Claude
Codex
```

These may be expressed as Kinds in Foundation, architecture, technology, provider, or project Schemas.

Importance does not imply kernel status.

---

# 35. Current minimal Core vocabulary

The current proposed user-facing Core vocabulary is:

```text
Definition
Schema
Kind
Property
type
reference
kindReference
minCount
maxCount
```

with built-in scalar/structured Property types:

```text
string
boolean
integer
number
enum
object
reference
kindReference
```

This is intentionally small.

---

# 36. Short mental model

```text
Definition
= a canonical thing that exists for a stated purpose

Schema
= a versioned vocabulary of Kinds

Kind
= describes Definitions of one nominal kind

Property
= one piece of information allowed in that Kind's spec

type
= the technical form of that Property value

reference
= a Property value that points to another Definition

kindReference
= a Property value that points to a Kind

minCount / maxCount
= how many values the Property may/must have
```

---

# 37. Compiler mental model

```text
Schema
  ↓
Kinds
  ↓
Property contracts
  ↓
Definitions
  ↓
Type checking
  ↓
Reference resolution
  ↓
Typed semantic graph
```

Everything above that graph belongs to higher-level Markitect ontologies and modules.

---

# 38. Why this is promising

This Core is small enough to reason about rigorously while still providing:

```text
custom vocabularies
nominal typing
closed models
explicit purpose
typed references
graph structure
multiple abstraction levels
source-format independence
deterministic compilation
```

The richer Markitect vision can then be built in Markitect itself.

For example:

```text
Rule
Invariant
Projection
Obligation
Evidence
Process
```

can be ordinary Kinds rather than hardcoded language primitives.

This preserves the desired architectural direction:

> **The Core provides the language. The Markitect ecosystem defines the engineering ontology.**
