# Markitect Operating Model

**Status:** Conceptual design baseline  
**Scope:** How Markitect is used, what a Module is, how projection work is executed and verified, and which system capabilities are required above the minimal Core.

---

# 1. Big picture

Markitect is the **single semantic source of truth** for the engineering system.

Code, documentation, CI/CD, agent instructions, hooks, infrastructure configuration, tests, and other technical artifacts are not competing sources of truth.

They are projections of canonical Markitect intent.

```text
                         Markitect
                    canonical intent
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
        Code         Documentation       CI/CD
          ▼               ▼               ▼
        .NET            Markdown      Azure DevOps
```

The normal workflow is therefore not:

```text
change code
→ update docs
→ update AI rules
→ update CI
→ later reconcile everything
```

but:

```text
change canonical intent
→ determine impact
→ project affected representations
→ verify each projection
→ verify composition
→ converge
```

If the canonical intent has not changed but a projection is wrong, Markitect reconciles the projection back to the existing intent.

The projection never silently becomes the new truth.

---

# 2. The agent is the primary user interface

A user should not need to learn Markitect syntax before using Markitect.

The expected entry point is a Markitect-aware AI agent.

Example:

> Initialize this repository as a Markitect project.

The agent should then actively guide the user.

It should ask questions such as:

```text
What are you building?
Which architecture style do you want?
Which technologies are involved?
Which deployment/platform systems are involved?
Which documentation projections do you want?
Which reusable Markitect Modules fit the project?
```

It should recommend available Modules rather than requiring the user to know their names in advance.

Example:

```text
Recommended Modules:

- Foundation
- Software Architecture
- DDD
- Vertical Slice
- .NET
- PostgreSQL
- Azure DevOps
- Markdown Documentation
```

The user can then continue working conversationally:

```text
human intent
    ↓
Markitect-aware agent
    ↓
proposed canonical model changes
    ↓
Core compilation
    ↓
accepted canonical model
```

The user may edit Markitect source directly, but this is not required to be the primary workflow.

---

# 3. Modeling remains iterative

Markitect should not turn software development into a large upfront specification phase.

A normal project can evolve as:

```text
discuss
  ↓
model
  ↓
project
  ↓
learn
  ↓
change model
  ↓
reproject
  ↓
verify
  ↓
repeat
```

The user may begin implementation before the complete product is modeled.

What changes is not the exploratory workflow.

What changes is where accepted engineering intent is recorded.

Accepted intent belongs in Markitect rather than being scattered across:

```text
chat history
Markdown files
source code
skills
agent instructions
CI definitions
architecture notes
```

---

# 4. Documentation is a projection

Markdown documentation is not a second canonical authority.

If human-readable documentation is desired, a Markdown/documentation Module projects selected Markitect Definitions into Markdown.

```text
Markitect
    ↓
Markdown Projector
    ↓
docs/architecture.md
docs/domain.md
docs/workflows.md
...
```

The generated documentation represents Markitect.

It does not compete with Markitect.

If the generated documentation drifts, the projection is repaired or regenerated.

---

# 5. What is a Markitect Module?

A **Module** is an installable extension package that adds reusable language and/or projection capabilities to Markitect.

A Module may provide:

```text
1. Schemas / Kinds
2. Projectors
3. Both
```

Nothing requires every Module to provide both.

This distinction is important.

---

# 6. Language-providing Modules

Some Modules primarily extend what can be expressed in the canonical model.

Examples:

```text
DDD
Vertical Slice
CQRS
Clean Architecture
```

A DDD Module might provide Kinds such as:

```text
BoundedContext
Aggregate
Entity
ValueObject
DomainEvent
Repository
```

A Vertical Slice Module might provide:

```text
UseCase
Command
Query
Handler
Validator
```

Such a Module does not necessarily need to project anything by itself.

Its primary value can be reusable ontology.

---

# 7. Projection-providing Modules

Other Modules primarily project canonical intent into a target representation.

Examples:

```text
Markdown Documentation
.NET
Azure DevOps
Git Hooks
GitHub
Codex
Claude
```

A Markdown Module may project Definitions into human-readable documentation without introducing major new engineering Kinds.

A .NET Module may project relevant architecture/application Definitions into source code, tests, project structure, and analyzers.

An Azure DevOps Module may project delivery intent into pipelines, environments, policies, or work-management configuration.

A Module may also provide both vocabulary and Projectors when the target itself has meaningful canonical concepts.

---

# 8. Vocabulary and projection should not be unnecessarily coupled

A Projector is allowed to consume Kinds provided by another installed Schema/Module.

For example:

```text
DDD Module
→ defines Aggregate

Vertical Slice Module
→ defines UseCase

.NET Module
→ projects Aggregate and UseCase intent into .NET
```

The .NET Module does not need to own the DDD vocabulary.

Likewise:

```text
Markdown Module
→ can document Definitions from many Schemas
```

This keeps ontology reusable across projection technologies.

A technology Module may still provide its own Kinds when technology-specific intent is genuinely canonical.

---

# 9. Module isolation

Modules do not synchronize with each other.

A Module works against:

```text
1. the canonical Markitect model
2. its own projection target
```

Conceptually:

```text
                 Markitect Model
                /       |        \
               /        |         \
            .NET     Markdown    Azure DevOps
```

Not:

```text
.NET ↔ Markdown ↔ Azure DevOps ↔ Agent Rules
```

This is a core architectural principle:

> **Modules synchronize with Markitect, never with each other.**

This avoids pairwise synchronization between representations.

A Module may depend on shared Schemas or contracts through Markitect, but it must not require direct projector-to-projector synchronization or another Module's representation as semantic authority.

---

# 10. Projector

A **Projector** is a Module-provided capability that materializes a selected part of the canonical model into one target representation.

Examples:

```text
Markitect → .NET source
Markitect → Markdown documentation
Markitect → Azure Pipeline
Markitect → Git hooks
Markitect → Codex instructions
```

A Projector should be able to declare at least:

```text
target / owned projection surface
applicable canonical model scope
supported Kinds or selectors
required canonical context
projection procedure or execution instructions
target-specific verification guidance where needed
```

A Projector may be implemented by:

```text
deterministic code
templates
an AI agent
or a combination
```

The Projector does not invent new canonical engineering intent.

If successful projection requires a new canonical decision, work returns to the Markitect model first.

---

# 11. No separate Observer concept for now

A separate `Observer` abstraction is not currently required.

Projection and verification naturally need access to the current target state.

For example:

```text
.NET Projector / Verifier
→ reads relevant .NET source

Markdown Projector / Verifier
→ reads generated documentation

Azure DevOps Projector / Verifier
→ reads relevant pipeline/configuration state
```

This can be treated as target access or target inspection within the projection/verification capability.

We should introduce a separate Observer concept only if later workflows demonstrate a reusable semantic responsibility that cannot be represented cleanly without it.

For now:

```text
observe target state
= operation needed by projection/verification

Observer
≠ required first-class role
```

---

# 12. Canonical model is a graph

The canonical Markitect model is not required to be a tree.

Definitions can have shared dependencies and cross-cutting relations.

Example:

```text
CreateOrder ─┐
             ├──> Order
CancelOrder ─┘
```

Therefore:

```text
Canonical Model
= typed semantic graph
```

The execution system may derive a hierarchical or DAG-shaped work plan from that graph.

The hierarchy is an execution strategy, not the ontology itself.

---

# 13. Recursive execution and verification

Markitect should avoid one large agent receiving the entire repository and being asked:

> Implement everything and then check whether it looks correct.

Instead, work is decomposed into small scopes.

Every scope uses the same basic roles:

```text
Executor
Verifier
```

There is no special `Integrator` role.

The same process repeats recursively from leaves to higher-level compositions.

---

# 14. Executor

The **Executor** is responsible for bringing one bounded scope toward the canonical desired state.

Depending on scope, this may mean:

```text
creating implementation
repairing implementation
updating a projection
reconciling drift
adjusting composition between already projected children
```

At a leaf scope, the Executor may implement one Handler or one generated document.

At a higher scope, the Executor may make the changes required for several already-valid children to compose correctly.

The role is the same.

Only the scope changes.

---

# 15. Verifier

The **Verifier** independently checks whether one bounded scope satisfies its relevant canonical intent and verification requirements.

The Verifier should receive:

```text
canonical intent for the scope
relevant dependencies/contracts
current projected state
verification criteria
relevant child results/evidence
```

The Verifier should not merely repeat the Executor's reasoning.

The desired pattern is:

```text
Executor
   ↓
Verifier
   ├── PASS
   └── FAIL + bounded feedback
            ↓
         Executor
            ↓
         Verifier
            ↓
           ...
```

The loop ends in:

```text
PASS
```

or an explicit escalation/failure condition.

---

# 16. The same process repeats upward

Suppose:

```text
CreateOrder
├── Handler
├── Validator
└── Endpoint
```

Each child is first executed and verified.

Then the same Executor/Verifier process runs at the `CreateOrder` scope.

The higher-level verification does not merely repeat the child checks.

It checks composition.

For example:

```text
Does validation actually protect the mutation?
Does the endpoint invoke the intended UseCase?
Do the Handler and Aggregate behavior compose correctly?
Do the child contracts fit together?
```

Then the same pattern can repeat again:

```text
leaf
  ↓
UseCase
  ↓
Module
  ↓
Product
  ↓
System
```

No new role is introduced at higher levels.

Only the context and verification responsibility change.

---

# 17. Why recursive verification matters

The following implication is false:

```text
valid(A) && valid(B)
⇒ valid(A + B)
```

Two individually correct parts may compose incorrectly.

Therefore Markitect needs verification at both:

```text
unit scope
and
composition scope
```

The recursive model provides this naturally.

---

# 18. Scoped context

Each Executor and Verifier should receive only the context relevant to its scope.

A leaf Executor might receive:

```text
target Definition
direct dependencies
applicable Rules/Invariants
projection contract
local target files
```

A higher-level Verifier might receive:

```text
parent canonical intent
child contracts
child verification results
integration/composition requirements
```

It should not automatically receive the entire repository or complete global model.

This provides:

```text
smaller context
lower token cost
less distraction
clearer responsibility
easier independent verification
```

---

# 19. Change impact

When canonical intent changes, Markitect should determine the affected semantic and projection scope.

Conceptually:

```text
changed Definitions
      ↓
semantic dependency impact
      ↓
affected Projectors
      ↓
affected execution scopes
      ↓
affected parent scopes
```

Only the affected subgraph and the required higher-level verification path need to run again.

Unrelated parts of the system remain untouched.

---

# 20. Intent change vs projection drift

These are fundamentally different.

## Intent change

The desired engineering system changes.

```text
canonical model changes
      ↓
impact analysis
      ↓
projection plan
      ↓
execution
      ↓
verification
```

## Projection drift

The desired engineering system has not changed, but a representation no longer matches it.

```text
canonical model unchanged
      ↓
affected projection identified
      ↓
reconciliation
      ↓
verification
```

Projection drift must never silently update canonical intent.

Markitect remains the authority.

---

# 21. Greenfield workflow

A normal new-project workflow can look like this.

## 1. Initialize

The user asks a Markitect-aware agent to initialize the project.

```text
"Initialize this repository as a Markitect project."
```

## 2. Guided Module selection

The agent asks about:

```text
architecture
technology
platform
documentation
delivery
agent tooling
```

and proposes relevant Modules.

## 3. Discovery and modeling

The user brainstorms naturally with the agent.

The agent continuously translates accepted decisions into the canonical Markitect model.

## 4. Compile continuously

Every model change is checked by the deterministic Core.

## 5. Reach a sufficient first desired state

The model does not need to be complete forever.

It only needs to be sufficiently explicit for the next implementation scope.

## 6. Determine projections

Installed Projectors identify which target representations are required for the selected canonical scope.

## 7. Analyze dependencies and impact

Markitect derives the relevant semantic and projection graph.

## 8. Build a scoped execution plan

The plan decomposes work into bounded nodes with dependencies.

## 9. Execute and verify leaves

Each leaf runs:

```text
Executor
→ Verifier
→ repair if needed
→ PASS
```

## 10. Repeat recursively upward

Once children pass, the same roles run for the parent scope.

## 11. Reach convergence

The requested canonical scope is considered converged when required projection and verification work has passed according to the declared process.

## 12. Continue evolving

New learning changes the canonical model first, then the affected projections are rerun.

---

# 22. Markitect-aware agent protocol

A Markitect-aware agent should follow a small universal operating protocol.

Conceptually:

```text
1. Treat Markitect as canonical authority.

2. Guide the user toward suitable installed/reusable Modules.

3. Record accepted engineering intent in Markitect.

4. Do not create a competing source of semantic truth.

5. Before changing engineering intent:
   update or propose canonical Definitions first.

6. Before projection:
   obtain scoped canonical context.

7. Project only within the current Projector/scope.

8. Do not synchronize directly with other Modules.

9. If a projection conflicts with canonical intent:
   canonical intent wins unless the user explicitly changes intent.

10. If projection requires a new architectural/business decision:
    return to canonical modeling instead of inventing the decision locally.

11. Verification returns evidence/results,
    not silent changes to canonical intent.
```

The provider-specific bootstrap for Codex, Claude, or another agent should ideally remain very thin.

Most project knowledge should come from Markitect dynamically.

---

# 23. Concrete system capabilities required

The operating model implies the following capabilities above Core.

These are **capabilities**, not yet necessarily separate services, modules, processes, or ontology Kinds.

## 23.1 Project initialization

Markitect must be able to initialize its project structure and bootstrap agent integration.

## 23.2 Module catalog and resolution

Markitect must be able to:

```text
discover available Modules
inspect what they provide
resolve compatible versions/dependencies
enable selected Modules
```

The AI agent should use this capability to guide Module selection.

## 23.3 Guided modeling

The agent needs access to:

```text
installed Schemas/Kinds
existing Definitions
relevant graph context
compiler diagnostics
```

so it can help the user create and evolve canonical intent.

## 23.4 Core compilation

The Core must:

```text
validate Definitions
validate Kinds/Properties/types
resolve references
normalize the model
build the semantic graph
report diagnostics
bind the result to a source revision
```

## 23.5 Model querying

Higher layers need a generic way to ask:

```text
Which Definitions are related to X?
What is the dependency neighborhood?
Which Rules apply?
Which parents/dependents exist?
Which Projectors are applicable?
```

This should operate over the normalized canonical model.

## 23.6 Model diff

Markitect must be able to compare canonical snapshots and identify semantic changes.

## 23.7 Impact analysis

Given a change, Markitect must determine:

```text
affected Definitions
affected dependencies
affected projections
affected execution scopes
affected higher-level verification scopes
```

## 23.8 Projection discovery

Installed Projectors must be discoverable for a canonical scope.

Markitect must determine:

```text
which Projectors apply
which target surfaces they own
what canonical context they require
```

## 23.9 Work planning

Markitect must transform:

```text
canonical scope
+
dependencies
+
applicable Projectors
+
verification requirements
```

into a scoped execution DAG/hierarchy.

## 23.10 Context building

Each work node needs a bounded context package containing only what its Executor or Verifier needs.

## 23.11 Execution orchestration

Markitect must run the Executor role for ready work nodes, respecting dependencies and parallelism.

The Executor may use:

```text
deterministic Projector logic
AI agents
external tools
or combinations
```

## 23.12 Verification orchestration

Markitect must run an independent Verifier role after execution.

Verification can use:

```text
deterministic checks
tests
analyzers
AI review
external systems
human escalation
```

## 23.13 Recursive composition verification

After child nodes pass, the same Executor/Verifier cycle must run at the parent scope.

This continues upward until the requested scope is verified.

There is no separate Integrator role.

## 23.14 Reconciliation

When verification fails but canonical intent is unchanged, Markitect must provide bounded failure information back to the Executor and retry/reconcile the projection.

## 23.15 Escalation

When the system cannot proceed without new engineering intent, it must stop projection work and ask for a canonical decision.

It must not invent that decision locally.

## 23.16 Run/evidence records

Markitect should retain enough structured information to answer:

```text
what was projected
against which canonical revision
by which Projector
which scope was verified
which checks ran
what passed/failed
what evidence supported the result
```

Whether `Evidence` becomes a canonical Kind is a later ontology decision.

## 23.17 Convergence evaluation

For a requested scope, Markitect must be able to determine whether all required work and verification has passed sufficiently to call that scope converged.

Convergence means:

> The declared engineering assurance process for this scope has completed successfully.

It does not mean:

> The software is objectively bug-free.

---

# 24. What is deliberately not yet a first-class concept

The current design does not require these as special system primitives yet:

```text
Observer
Integrator
Invariant DSL
Rule engine
general-purpose graph query language
module-to-module synchronization
```

Likewise, names such as:

```text
Planner
Evidence
Obligation
Verifier Definition
Projection Definition
```

should not be promoted into the canonical ontology until the operating model proves they need to be independently modeled.

Capabilities come first.

Ontology is derived from real capability needs.

---

# 25. Current architectural principles

## P1 — Canonical Authority

> **Markitect is the single semantic source of truth. Everything else is a projection or observation of that truth.**

## P2 — Agent-led UX

> **The normal user works with a Markitect-aware agent; the agent guides modeling and Module selection.**

## P3 — Module Isolation

> **A Module works against Markitect and its own target. Modules do not synchronize with each other.**

## P4 — Vocabulary/Projection Decoupling

> **A Module may provide Schemas, Projectors, or both. Projectors may consume reusable Kinds defined elsewhere.**

## P5 — Scoped Delegation

> **Every execution or verification agent receives only the context and responsibility required for its current scope.**

## P6 — Independent Verification

> **Projection work and verification are separate responsibilities.**

## P7 — Recursive Assurance

> **The same Executor/Verifier process repeats from leaves through progressively larger composition scopes.**

## P8 — Graph-first Modeling

> **The canonical model is a graph; hierarchical execution plans are derived from it.**

## P9 — Intent Before Projection

> **Engineering intent changes Markitect first; projections follow.**

## P10 — Reconcile Drift, Do Not Canonicalize It

> **A divergent projection is repaired against canonical intent unless the user explicitly changes that intent.**

---

# 26. Current end-to-end picture

```text
                         USER
                           │
                           ▼
                Markitect-aware Agent
                           │
                 guides / asks / models
                           │
                           ▼
                  CANONICAL MARKITECT
                         MODEL
                           │
                           ▼
                         CORE
                 compile / normalize
                  resolve / graph
                           │
                           ▼
                  change + impact
                           │
                           ▼
                  projection discovery
                           │
                           ▼
                    execution plan
                    (DAG/hierarchy)
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
      Projector A      Projector B      Projector C
          │                │                │
          ▼                ▼                ▼
       Executor         Executor         Executor
          │                │                │
          ▼                ▼                ▼
       Verifier         Verifier         Verifier
          │                │                │
          └──────────┬─────┴─────┬──────────┘
                     ▼           ▼
              parent scope Executor
                     │
                     ▼
              parent scope Verifier
                     │
                     ▼
                repeat upward
                     │
                     ▼
                 CONVERGED
```

On failure:

```text
Verifier FAIL
    ↓
bounded feedback
    ↓
Executor repairs
    ↓
Verifier reruns
```

If a new semantic decision is required:

```text
projection stops
    ↓
escalate to modeling
    ↓
canonical model changes
    ↓
new impact analysis
```

---

# 27. Next design questions

The next useful design work is to define the contracts for the capabilities that now appear necessary.

In particular:

```text
1. What exactly is a Module manifest?
2. How does a Module declare provided Schemas and Projectors?
3. What exactly is a Projector contract?
4. How does a Projector declare applicability and target ownership?
5. What context does a Projector request?
6. How is a work node represented?
7. How is the recursive Executor/Verifier plan derived from the graph?
8. What is the minimal Verifier input/output contract?
9. What information is retained as run/evidence data?
10. What exactly constitutes convergence?
11. How does impact propagation work?
12. Does kindReference prove useful in these contracts, or can it be removed?
```

The guiding method remains:

> **First identify the capability required by the operating model. Only then decide whether it deserves a canonical Kind, a runtime concept, a Module contract, or no new concept at all.**
