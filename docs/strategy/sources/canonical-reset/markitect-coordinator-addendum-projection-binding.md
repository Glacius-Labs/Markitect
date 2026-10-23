# Coordinator Addendum — Projection Binding and Projector Selection

This addendum clarifies one important part of the architecture that was not explicit enough in the main reset brief:

> **Installing a Module does not itself mean that anything should be projected.**

A Module registers capabilities. Canonical Markitect intent decides whether, where, and for which scope those capabilities are used.

This distinction should be reflected in the implementation.

---

# 1. The missing connection

The intended model is:

```text
Module
  ↓ registers
Projector
  ↓ selected by
Projection
  ↓ constrained by
ProjectionPolicy
  ↓ materializes
Target Representation
  ↓ recorded by
ProjectionRecord
```

Short definitions:

```text
Module
= packaging/distribution of reusable capabilities

Projector
= executable projection capability

Projection
= canonical intent saying what scope should be projected
  by which Projector into which target

ProjectionPolicy
= canonical representation constraints/guidance

ProjectionRecord
= operational provenance of what was actually materialized
```

Do not collapse these concepts into one abstraction.

---

# 2. Module

A Module provides reusable capabilities.

The primary extension points are:

```text
Schema registration
Projector registration
```

Optionally:

```text
discovery metadata
reusable recommendations/profiles
```

Example:

```text
DDD Module

Schemas:
  ddd/v1

Projectors:
  none
```

Example:

```text
.NET Module

Schemas:
  optional technology-specific schemas

Projectors:
  dotnet-source
```

Installing `.NET` means:

> This Markitect installation is capable of projecting into .NET.

It does **not** mean:

> Every applicable Definition should now automatically become C#.

---

# 3. Projector

A Projector is an executable capability provided by a Module.

Examples:

```text
dotnet-source
markdown-documentation
azure-pipeline
git-hooks
codex-bootstrap
```

A Projector should know its target technology/representation and how to operate on that target.

It should not decide whether the project wants a given semantic scope represented there.

That decision is canonical project intent.

---

# 4. Projection

`Projection` is now a strong candidate for a Foundation Kind.

A Projection expresses:

> **This canonical scope is intended to have a representation in this target using this Projector.**

Conceptually:

```yaml
apiVersion: markitect.foundation/v1
kind: Projection

metadata:
  namespace: implementation
  name: application-dotnet

purpose: >
  Materializes the application architecture as the primary
  .NET implementation.

spec:
  source:
    root: <canonical definition/scope>

  projector:
    module: markitect-dotnet
    id: dotnet-source

  target:
    repository: .
    path: src/

  policies:
    - general-dotnet
    - vertical-slice-dotnet
    - aggregate-dotnet
```

Do not freeze this exact syntax yet.

Preserve the semantics.

A Projection answers:

```text
WHAT canonical scope?
WHICH Projector?
WHERE should it be represented?
WHICH representation policies apply?
```

This is canonical engineering intent and therefore belongs in Markitect.

---

# 5. Projector selection is explicit

The Projector does not scan the model and decide:

> I see Aggregates, therefore I should create C#.

Instead:

```text
Projection
  ↓
selects Projector
  ↓
selects canonical scope
  ↓
selects target
```

This avoids implicit behavior.

Installed capability:

```text
dotnet-source available
```

is different from desired projection:

```text
Application should be projected by dotnet-source into src/
```

---

# 6. Where the Projector gets the "how"

A Projector should derive its concrete implementation from four inputs:

```text
1. canonical Definitions
2. Kind/Schema semantics
3. applicable ProjectionPolicies
4. target-specific Projector knowledge/context
```

Example:

```text
Order
kind: Aggregate
```

The .NET Projector may receive:

```text
Order Definition
Aggregate Kind purpose/Properties
related canonical Definitions
applicable Rules/Invariants
ProjectionPolicy: Aggregate → .NET
existing target state
```

and then materialize one valid .NET representation.

The .NET Projector should not require hard-coded knowledge of every possible ontology Kind.

---

# 7. Avoid ontology × target coupling

Do not make:

```text
.NET know every DDD concept
DDD know every implementation language
```

Avoid a built-in matrix such as:

```text
DDD × .NET
DDD × Java
DDD × Go
Vertical Slice × .NET
Vertical Slice × Go
...
```

Instead prefer:

```text
semantic ontology
      +
project-specific ProjectionPolicy
      +
target Projector
```

Example:

```text
DDD Aggregate
      +
Aggregate-to-.NET ProjectionPolicy
      +
.NET Projector
      ↓
C# representation
```

This keeps Modules independently reusable.

---

# 8. ProjectionPolicy

`ProjectionPolicy` expresses canonical representation intent.

Example:

```yaml
apiVersion: markitect.foundation/v1
kind: ProjectionPolicy

metadata:
  namespace: architecture
  name: aggregate-to-dotnet

purpose: >
  Defines how Aggregates should normally be represented
  in this project's .NET implementation.

spec:
  sourceKind:
    apiVersion: ddd.example.org/v1
    kind: Aggregate

  target: dotnet

  guidance: >
    Represent Aggregates as behavior-rich domain types.
    Keep persistence concerns outside the domain type.
    Do not use EF attributes in domain classes.
```

This is one concrete reason `kindReference` may be useful.

Pressure-test that Core feature during implementation rather than assuming it is permanent.

---

# 9. ProjectionPolicy is not a generator template

Do not turn ProjectionPolicy into a complete implementation DSL.

It should express representation decisions and constraints.

For AI Projectors:

```text
Canonical intent
+
ProjectionPolicy
+
target context
      ↓
AI searches implementation space
      ↓
candidate representation
```

The intended model remains:

```text
R ∈ ValidRepresentations(M)
```

not:

```text
R = f(M)
```

Different concrete implementations may be valid.

---

# 10. Missing projection intent must be visible

Suppose a Projector encounters:

```text
kind: EffectAxis
```

and its semantics do not provide enough information to make a safe .NET representation.

It must not silently invent a convention.

It should return an explicit gap/escalation such as:

```text
Missing projection intent:

EffectAxis is part of the selected .NET Projection,
but no sufficiently precise representation policy or
semantic guidance exists.
```

The Markitect-aware agent then discusses the missing representation decision with the user.

Accepted guidance becomes canonical ProjectionPolicy/intent.

Projection resumes afterward.

This turns ambiguity into explicit modeling rather than hidden agent convention.

---

# 11. A Projection operates on a semantic subgraph

Do not assume:

```text
one Definition
→ one artifact
```

A Projection consumes a canonical subgraph.

Example:

```text
CreateOrder UseCase
CreateOrder Handler
Validator
Rules
Invariants
Module
ProjectionPolicies
        ↓
.NET Projector
        ↓
CreateOrderCommand.cs
CreateOrderHandler.cs
CreateOrderValidator.cs
Endpoint.cs
Tests
```

Likewise, one artifact may represent several Definitions.

The relationship is naturally many-to-many.

---

# 12. ProjectionRequest

The Host/Planner should eventually be able to construct a bounded request similar to:

```text
ProjectionRequest

canonicalRevision
Projection Definition
canonical scope
Definitions in scope
relevant Kind schemas
resolved references/dependencies
applicable Rules/Invariants
applicable ProjectionPolicies
currently owned artifacts
relevant target state
```

This is the Projector's working context.

Do not automatically send the whole repository/model.

---

# 13. ProjectionResult

A Projector should return a structured result.

Conceptually:

```text
ProjectionResult

created artifacts
modified artifacts
deleted artifacts
retained artifacts
diagnostics
escalations
```

A Projector should not make opaque/unreported changes.

The exact patch/write mechanism remains an implementation detail.

---

# 14. ProjectionRecord

After materialization, persist operational provenance:

```text
Projection
      ↓
canonical inputs
      ↓
Projector/version
      ↓
actual artifacts
```

This becomes the basis for:

```text
artifact ownership
drift detection
impact analysis
targeted reconciliation
```

Remember:

```text
Projection
= canonical intent

ProjectionRecord
= operational result/provenance
```

Do not merge them.

---

# 15. Example end-to-end binding

```text
Installed Module:
markitect-dotnet

registers:
dotnet-source Projector
```

Then canonical model contains:

```text
Projection:
application-dotnet

source:
Application

projector:
dotnet-source

target:
src/

policies:
general-dotnet
vertical-slice-dotnet
aggregate-dotnet
```

Planner resolves:

```text
Application semantic subgraph
+
Policies
+
owned artifacts
+
target state
```

into:

```text
ProjectionRequest
```

Projector executes.

It returns:

```text
ProjectionResult
```

Markitect persists:

```text
ProjectionRecord
```

Verifier then independently evaluates the resulting scope.

---

# 16. Implication for the Markitect Foundation

When designing the initial Foundation Schema, explicitly evaluate whether the following are now justified:

```text
Project
Goal
Rule
Invariant
Projection
ProjectionPolicy
```

Do not automatically add all of them.

However, `Projection` now has a concrete architectural responsibility that `ProjectionPolicy` alone cannot provide:

```text
Projection
= declares that a representation SHOULD exist

ProjectionPolicy
= constrains HOW that representation should look
```

That distinction should be preserved even if final naming changes.

---

# 17. Implication for Projector descriptors

Be careful with Projector declarations such as:

```text
supportedKinds:
  Aggregate
  UseCase
  Handler
  ...
```

For specialized deterministic Projectors this may be appropriate.

For a generic AI-backed target Projector such as `.NET`, it risks reintroducing coupling to every ontology.

Prefer, where practical:

```text
Projector understands target representation
+
canonical semantic context
+
ProjectionPolicy
```

and allow explicit escalation when representation intent is insufficient.

The architecture should support both:

```text
generic semantic Projector
```

and:

```text
specialized typed deterministic Projector
```

without making either model universal.

---

# 18. Architectural summary

Use this as the current projection binding model:

```text
MODULE
"What reusable capability is installed?"

        ↓ registers

PROJECTOR
"What executable materialization capability exists?"

        ↓ selected by

PROJECTION
"What canonical scope should be represented,
where, and using which Projector?"

        ↓ constrained by

PROJECTION POLICY
"What representation decisions constrain how
that semantic intent should appear?"

        ↓ executed as

PROJECTION REQUEST
"Here is the bounded canonical + target context."

        ↓ returns

PROJECTION RESULT
"These artifacts were created/changed/deleted."

        ↓ persisted as

PROJECTION RECORD
"This is what was actually materialized from
this canonical revision."
```

This should be incorporated into the architecture delta and implementation plan before locking the new Foundation and Projector contracts.
