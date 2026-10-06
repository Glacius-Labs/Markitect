# Coordinator Addendum — Strict Module Types and AI-First Projection

This addendum sharpens two architectural decisions from the current Markitect redesign.

The previous formulation:

> A Module may provide Schemas, Projectors, or both.

is too permissive.

Replace it with a strict two-type model.

---

# 1. Two and only two Module types

Every Markitect Module has exactly one primary responsibility and exactly one type:

```text
Module
├── Schema Module
└── Projection Module
```

Do not implement a generic Module that can freely accumulate both responsibilities.

The goal is to make architectural boundaries obvious and mechanically enforceable.

---

# 2. Why `Schema Module`

Use **Schema Module** instead of `Vocabulary Module`.

`Schema` is already a Core term and directly describes what this Module type contributes.

A Schema Module answers:

> **What can be expressed canonically in Markitect?**

It provides one or more Schemas containing Kinds and their Properties.

Examples:

```text
DDD Schema Module
Vertical Slice Schema Module
CQRS Schema Module
Clean Architecture Schema Module
Azure DevOps Schema Module
```

Example:

```text
markitect-ddd

type:
schema

provides:
  ddd/v1

Kinds:
  BoundedContext
  Aggregate
  Entity
  ValueObject
  DomainEvent
  Repository
```

A Schema Module:

```text
MAY:
- register Schemas
- define Kinds
- define Properties and semantic purposes
- provide reusable canonical language

MUST NOT:
- project artifacts
- write target repositories
- contain target-specific projection behavior
- synchronize with Projection Modules
```

The Schema Module knows semantics, not target technologies.

---

# 3. Projection Module

A Projection Module answers:

> **How can canonical Markitect intent be represented in one target technology or representation?**

Examples:

```text
.NET Projection Module
Go Projection Module
Markdown Projection Module
Azure DevOps Projection Module
GitHub Projection Module
Codex Projection Module
Claude Projection Module
Git Hooks Projection Module
```

A Projection Module provides target-specific execution knowledge such as:

```text
target semantics
target conventions
allowed target surfaces
agent instructions
available tools
repository/API interaction knowledge
discovery signals
verification guidance
```

A Projection Module:

```text
MAY:
- teach an Executor how to work with its target
- provide target-specific tools
- declare target surfaces
- provide discovery metadata
- provide verification guidance

MUST NOT:
- introduce canonical Kinds
- redefine source ontology semantics
- directly synchronize another Module's projection
- treat another Module's artifacts as canonical authority
```

---

# 4. Technologies that need both responsibilities use two Modules

Do not weaken the boundary because some ecosystems naturally have both semantic concepts and a concrete representation.

Example: Azure DevOps.

Instead of:

```text
Azure DevOps Module
  - Pipeline Kind
  - Stage Kind
  - YAML Projector
```

use:

```text
Azure DevOps Schema Module

Kinds:
  Pipeline
  Stage
  Environment
  Approval
```

and separately:

```text
Azure DevOps Projection Module

Target:
  Azure DevOps / pipeline representation
```

Likewise, if .NET eventually needs canonical technology-specific concepts:

```text
.NET Schema Module
```

and:

```text
.NET Projection Module
```

remain separate packages/responsibilities.

---

# 5. Bundles are installation convenience, not a third Module type

The Registry may expose a user-friendly bundle/recommendation:

```text
Azure DevOps
```

which installs:

```text
azure-devops-schema
azure-devops-projection
```

Likewise a recommended stack may display:

```text
DDD
Vertical Slice
.NET
Azure DevOps
Markdown Documentation
```

while internally resolving to a set of Schema Modules and Projection Modules.

Do not introduce `Bundle Module` as a third semantic Module type.

A bundle is Registry/package-management convenience only.

---

# 6. Why the strict split matters

The previous flexible model allows responsibility creep.

A generic Module could gradually become:

```text
.NET Module
- C# Kinds
- DDD helpers
- code generation
- documentation generation
- analyzers
- special architecture semantics
```

At that point the boundary becomes unclear and Modules start knowing about each other.

The strict split enforces:

```text
Schema Module
= extends canonical language

Projection Module
= represents canonical language in a target
```

This should be enforceable through architecture tests / package dependencies.

---

# 7. Projection should be AI-first

The target operating model assumes that most implementation work is performed by AI agents.

Humans should primarily review and decide engineering intent, not routinely inspect every generated code diff.

The normal loop is:

```text
human defines / approves intent
        ↓
canonical Markitect model
        ↓
scoped Projection
        ↓
Executor Agent
        ↓
candidate representation
        ↓
independent Verifier Agent
        ↓
PASS / bounded repair loop
```

Human attention returns when:

```text
intent is missing
intent is ambiguous
canonical rules conflict
an architecture/business decision is required
verification remains uncertain
explicit approval is required
```

This is a deliberate product assumption.

---

# 8. One projection execution model

Avoid creating two first-class architectural paths such as:

```text
AI Projector
vs
Deterministic Projector
```

Prefer one execution model:

```text
Projection
    ↓
Executor Agent
    ↓
target representation
```

The Executor Agent may use any appropriate deterministic tools:

```text
compiler
formatter
template renderer
source generator
SDK
CLI
schema validator
filesystem tools
```

A trivial Markdown projection may therefore use a deterministic renderer internally.

A .NET projection may rely heavily on agent reasoning.

Architecturally they remain the same process:

```text
bounded canonical projection task
→ Executor
→ result
→ Verifier
```

Deterministic renderers/generators are tools, not a separate projection architecture.

---

# 9. Reconsider whether `Projector` needs to remain a public first-class noun

The earlier architecture used:

```text
Module
→ Projector
→ Projection
```

With strict Projection Modules and an AI-first Executor model, pressure-test whether a separate user-visible `Projector` abstraction still adds enough value.

A simpler possible model is:

```text
Projection Module
      ↓ selected by
Projection
      ↓ executed by
Executor Agent
```

In that model:

```text
Projection Module
= target adapter / target expertise

Projection
= canonical declaration that a scope should be represented through that module

Executor
= AI role that performs the projection
```

A Projection Module may still internally expose an executable entrypoint/interface.

The question is whether `Projector` needs to remain a separate conceptual layer for users and canonical modeling.

Do not remove it blindly, but explicitly evaluate it during implementation.

Prefer fewer concepts if the contracts remain equally clear.

---

# 10. Projection Modules should be source-ontology agnostic where practical

A `.NET Projection Module` should primarily know `.NET`.

It should not accumulate hard-coded knowledge of every possible source Schema:

```text
DDD
CQRS
Vertical Slice
Event Sourcing
project-specific Kinds
future third-party Schemas
```

Avoid:

```text
if Kind == Aggregate ...
if Kind == UseCase ...
if Kind == Saga ...
```

as the universal architecture.

Instead, the Executor receives:

```text
canonical Definition
Kind purpose and Property semantics
relevant graph context
Rules / Invariants
ProjectionPolicy
target context
.NET Projection Module guidance/tools
```

and uses that information to construct a valid representation.

If the representation intent is insufficient, escalate.

---

# 11. AI is what makes source/target decoupling practical

A deterministic generator usually requires an explicit mapping for every source semantic concept.

That leads to a cross-product:

```text
DDD × .NET
DDD × Java
DDD × Go
Vertical Slice × .NET
Vertical Slice × Go
...
```

Markitect should instead exploit AI:

```text
semantic Definition
+
self-describing Kind/Schema
+
project-specific ProjectionPolicy
+
target-specialized Projection Module
+
target context
        ↓
Executor Agent
        ↓
one valid target representation
```

This allows:

```text
DDD Schema Module
```

to know nothing about `.NET`, while:

```text
.NET Projection Module
```

does not need every DDD Kind hard-coded into it.

This is a central architectural hypothesis and should be tested explicitly.

---

# 12. ProjectionPolicy remains the semantic bridge

When the generic semantics are insufficient to determine the intended target representation, the missing decision belongs in canonical ProjectionPolicy / projection intent.

Example:

```text
Aggregate semantics
+
ProjectionPolicy:
  In this project, Aggregates represented in .NET
  are behavior-rich domain classes with no EF attributes
+
.NET Projection Module
        ↓
Executor Agent
        ↓
C# representation
```

Neither Module owns the cross-technology mapping.

The project does.

---

# 13. Missing representation intent is an escalation, not an invitation to guess

If the Executor understands the source concept but cannot determine a safe target representation:

```text
Missing Projection Intent
```

should be returned.

The Markitect-aware modeling agent asks the user for the missing decision.

Accepted guidance becomes canonical intent.

Then projection resumes.

Do not hide cross-ontology conventions inside provider prompts or Module implementation.

---

# 14. Verification is AI-first as well

The recurring roles remain:

```text
Executor Agent
Verifier Agent
```

The Verifier Agent may use deterministic evidence:

```text
compiler output
unit tests
integration tests
static analyzers
architecture checks
schema validation
artifact accounting
```

Think of the Verifier as producing a bounded assurance judgment from both:

```text
semantic reasoning
+
independent evidence
```

Do not let the Executor self-certify solely through tests it authored.

Preserve the existing research constraint that independent evidence and candidate-authored checks are not equivalent.

---

# 15. Recursive process remains unchanged

For each bounded scope:

```text
canonical scope
      ↓
Executor Agent
      ↓
candidate projection
      ↓
Verifier Agent + evidence
      ↓
PASS / FAIL
```

On failure:

```text
FAIL
 ↓
bounded diagnostics
 ↓
same Executor responsibility repairs
 ↓
Verifier reruns
```

After child scopes pass, run the same roles at the parent/composition scope.

No special Integrator role.

---

# 16. Registry / onboarding implication

The Markitect-aware agent should recommend two categories during setup.

Conceptually:

```text
Schema Modules
"What concepts do you want to model?"

Projection Modules
"Which real representations do you want Markitect to maintain?"
```

Example:

```text
Based on your project, I recommend:

Schema Modules:
- DDD
- Vertical Slice

Projection Modules:
- .NET
- Azure DevOps
- Markdown Documentation
```

The UI does not need to use this exact wording if a friendlier presentation works better.

The architecture should still retain the strict underlying split.

---

# 17. Updated architectural summary

Use this as the current preferred model:

```text
                    MARKITECT CORE
                          │
                 Schema / Kind system
                          │
             ┌────────────┴────────────┐
             │                         │
      SCHEMA MODULES           PROJECTION MODULES
             │                         │
       extend canonical          provide target
          language              expertise + tools
             │                         │
             └────────────┬────────────┘
                          ▼
                   CANONICAL MODEL
                          │
                     Projection
                          │
                  ProjectionPolicies
                          │
                          ▼
                    EXECUTOR AGENT
                          │
                          ▼
                      ARTIFACTS
                          │
                          ▼
                    VERIFIER AGENT
                          │
                   evidence + intent
                          │
                     PASS / repair
```

The essential boundary is:

```text
Schema Module
= what can be said

Projection Module
= how intent can be represented in a target

Projection
= what should actually be represented there

ProjectionPolicy
= project-owned representation decisions

Executor
= AI that performs the projection

Verifier
= independent AI that judges the result using scoped evidence
```

---

# 18. Update prior design language

Where current design docs or implementation plans say:

> Modules may provide Schemas, Projectors, or both

replace that direction with:

> **Every Module is exactly one of two types: Schema Module or Projection Module.**

Where prior docs describe deterministic and AI Projectors as parallel architectural models, prefer:

> **Projection is agent-executed. Deterministic renderers and generators are tools available to the Executor, not a separate top-level projection paradigm.**

Preserve existing deterministic mechanics where they provide valuable evidence, reproducibility, or efficient tooling.

Do not throw away working renderers/checkers.

Reframe them underneath the unified agent-execution model.

---

# 19. Pressure tests

While implementing, explicitly test:

```text
Can a .NET Projection Module project a custom third-party Kind
without hard-coded knowledge of its Schema when sufficient
purpose/Properties/ProjectionPolicy are provided?

Can a Schema Module remain completely unaware of every
Projection Module?

Does strict module typing eliminate ambiguous ownership?

Can deterministic Markdown rendering remain useful as an
Executor tool without requiring a second architecture?

Does removing/retaining a separate Projector noun materially
simplify or complicate the contracts?

Does agent-first projection increase ambiguity or cost in cases
where deterministic rendering is obviously sufficient?

Can the Verifier remain independent enough when both Executor
and Verifier are AI agents?
```

Preserve negative findings.

The strict module split and AI-first execution are architectural hypotheses; test them rather than protecting them from evidence.
