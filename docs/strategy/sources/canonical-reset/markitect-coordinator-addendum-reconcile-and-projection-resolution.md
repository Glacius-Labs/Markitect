# Coordinator Addendum — Reconcile-First Operating Model and Projection Resolution

This addendum corrects and sharpens two parts of the current Markitect design:

1. The primary user/system operation should be **reconciliation**, not explicit implementation commands such as “implement CreateOrder”.
2. Canonical intent should not unnecessarily hard-bind itself to a concrete Projector implementation.

These points should be incorporated into the current architecture delta and implementation plan.

---

# 1. Reconcile is the primary operation

The normal user workflow is not:

```text
Model CreateOrder
      ↓
Tell an agent:
"Implement CreateOrder"
```

The intended workflow is:

```text
User + Markitect-aware Agent
      ↓
change canonical engineering intent
      ↓
Core compiles new canonical snapshot
      ↓
Markitect detects the semantic delta
      ↓
Markitect reconciles all affected representations
```

The user changes **desired state**.

Markitect determines what implementation work follows.

This should feel much closer to Terraform than to a conventional coding-agent command interface.

---

# 2. Desired-state mental model

The system should be understood as:

```text
CANONICAL MODEL
= desired engineering state

PROJECTED ARTIFACTS
= currently realized state

RECONCILE
= bring realized state back into agreement
  with current canonical state
```

The user should not normally need to know:

```text
which C# files need editing
which Markdown pages need regeneration
which tests need adjustment
which agent files need updating
which pipeline fragments need changing
```

That is what reconciliation is for.

---

# 3. Typical user flow

Example:

```text
User:
CreateOrder should now be idempotent.

Agent:
[updates canonical Markitect intent]

Markitect:
canonical model compiles

Reconcile:
- detect canonical diff
- calculate impact
- determine affected Projection Modules
- derive affected work scopes
- execute/verify those scopes
- update projection provenance
```

The user does not issue:

```text
"Implement idempotency in CreateOrderHandler.cs"
```

unless they deliberately bypass the normal Markitect workflow.

The desired abstraction is:

> **Users change intent. Markitect reconciles representations.**

---

# 4. Plan before reconcile

A planning step should naturally fall out of this model.

Conceptually:

```text
markitect plan
```

could report:

```text
Canonical changes:

~ UseCase/orders/create-order
    + idempotency requirement

Semantic impact:

~ CreateOrder
~ Orders Module
~ Application

Projection impact:

.NET
  ~ CreateOrder implementation
  ~ related tests

Markdown
  ~ Orders documentation

Azure DevOps
  no affected representation

Verification:

~ CreateOrder scope
~ Orders scope
~ Application scope
```

Then:

```text
markitect reconcile
```

executes the plan.

Exact CLI naming remains open.

The important architectural point is:

```text
canonical diff
→ impact
→ reconciliation plan
→ execution
→ verification
→ convergence
```

---

# 5. Reconcile inputs

A reconciliation run should conceptually start from:

```text
previous canonical revision/snapshot
current canonical revision/snapshot
current Projection Records
current artifact ownership state
enabled Projection Modules
Projection Module configuration
canonical ProjectionPolicies
current target state
```

It should not require a user to manually select every implementation task.

---

# 6. Reconcile algorithm direction

Pressure-test an internal lifecycle approximately like:

```text
1. compile current canonical model

2. compare previous/current canonical snapshots

3. identify changed Definitions / semantic facts

4. traverse typed semantic graph for impact

5. identify affected projection-relevant scopes

6. ask enabled Projection Modules for applicable work
   within their configured target scope

7. combine work proposals

8. resolve ownership/conflicts

9. derive execution DAG / recursive verification scopes

10. execute affected scopes

11. verify each scope independently

12. repeat verification upward for affected compositions

13. update Projection Records / artifact ownership

14. report convergence / unresolved drift / escalation
```

The exact internal APIs are open.

Preserve this operating model.

---

# 7. Impact should be minimal and explicit

A canonical change should not default to:

```text
re-run all projectors
re-read whole repository
globally re-review everything
```

Instead:

```text
changed canonical intent
      ↓
semantic dependents
      ↓
affected Projection Modules
      ↓
affected owned artifacts
      ↓
required parent verification path
```

Only the necessary subgraph should be reconciled.

This is one of the major expected benefits of Markitect.

---

# 8. Intent change vs projection drift

Preserve the strict distinction.

## Intent change

```text
canonical model changes
      ↓
plan
      ↓
reconcile affected representations
```

## Projection drift

```text
canonical model unchanged
      ↓
artifact no longer matches expected representation
      ↓
reconcile artifact/projection back to existing intent
```

Never canonicalize drift automatically.

---

# 9. Reconsider canonical `Projection`

The current design introduced a canonical `Projection` Kind with fields such as:

```text
source scope
Projector
target
policies
```

This should now be pressure-tested.

The concern is:

> If canonical intent stores the concrete Projector implementation, changing the implementation mechanism becomes a canonical engineering change even when the desired representation has not changed.

Example:

```text
dotnet-projector-v1
→ dotnet-projector-v2
```

should usually be an implementation/configuration change, not a semantic architecture change.

Therefore:

> **Canonical intent should not unnecessarily know who executes it.**

---

# 10. Separate desired representation from implementation binding

Distinguish:

```text
Projection intent / desired representation
= what representation should exist

Projection Module capability
= what installed capability can realize that representation

runtime binding
= which installed capability is selected for this reconcile
```

Only the first is necessarily canonical.

The concrete module/binding belongs to runtime/project configuration unless the choice itself is genuinely engineering intent.

---

# 11. Preferred direction

Prefer this model:

```text
Canonical Model
      +
Enabled / configured Projection Modules
      +
Canonical ProjectionPolicies
      +
Operational ProjectionRecords
      ↓
Reconcile
```

rather than:

```text
Canonical Projection Definition
hard-binds
specific Projector implementation
```

The reconciliation system should resolve appropriate projection capabilities.

---

# 12. Projection Module configuration may be enough

A simpler architecture may be:

```text
Project Configuration
├── enabled Schema Modules
└── enabled Projection Modules
      ├── scope
      ├── target
      └── module-specific configuration
```

Example conceptually:

```yaml
projectionModules:

  dotnet:
    scope:
      - application/backend
    target:
      root: src/

  markdown:
    scope:
      - product
      - architecture
      - operations
    target:
      root: docs/
```

Then reconciliation asks each enabled Projection Module:

> Which parts of the affected canonical graph are relevant to your configured target?

This may eliminate the need for a canonical `Projection` Definition entirely.

Do not assume it must be removed.

Pressure-test it.

---

# 13. If canonical `Projection` remains

If a canonical Projection concept still proves useful, it should describe **desired representation**, not execution implementation.

It may reasonably express:

```text
desired representation type
canonical scope
target/location
applicable ProjectionPolicies
representation requirements
```

It should avoid hard-binding:

```text
specific module package
specific Projector version
specific executor implementation
```

Binding should be resolved during reconcile.

---

# 14. Capability resolution

The architecture should permit:

```text
Desired representation:
.NET source representation

Available Projection Modules:
dotnet-standard
dotnet-experimental
```

The runtime/project configuration resolves one suitable capability.

Conceptually:

```text
canonical requirement
      ↓
capability resolution
      ↓
Projection Module
      ↓
Executor Agent
```

Switching compatible Projection Module implementations should not require changing unrelated canonical intent.

---

# 15. Projection Modules may discover applicable work

An enabled Projection Module should be able to inspect the affected canonical graph and propose the work it can meaningfully reconcile within its configured scope.

Example:

```text
.NET Projection Module

affected canonical graph:
- Order
- DeleteOrder
- Orders Module
- retention Rule

result:
- Order implementation affected
- DeleteOrder implementation affected
- related tests affected
```

Markdown may return:

```text
- Orders behavior documentation affected
```

Azure DevOps may return:

```text
no applicable work
```

These are work proposals for planning.

They are not new canonical intent.

---

# 16. Applicability must not become hidden semantic authority

Projection Modules may determine:

```text
what they can represent
what target artifacts they own
which affected scope belongs to them
```

They must not determine:

```text
new business behavior
new architecture responsibilities
new Rules
new canonical dependencies
```

If target representation requires missing semantic intent:

```text
escalate
```

Do not guess and silently extend the canonical model.

---

# 17. ProjectionPolicy still has a clear role

Keep ProjectionPolicy as canonical representation intent where needed.

It answers:

> **What properties must a representation preserve or follow?**

Example:

```text
Aggregate → .NET policy:

- behavior-rich domain type
- persistence-independent
- no EF attributes
```

A Projection Module can then determine how to materialize that policy in its target.

ProjectionPolicy should not identify a concrete Projector implementation.

---

# 18. Suggested conceptual split

Use this distinction when evaluating the current model:

```text
CANONICAL

Definition
Rule
Invariant
ProjectionPolicy
possibly desired representation requirement

----------------------------

PROJECT CONFIGURATION

enabled Schema Modules
enabled Projection Modules
Projection Module target scopes
capability bindings
local execution configuration

----------------------------

OPERATIONAL STATE

ProjectionRecords
artifact ownership
run IDs
digests
verification results
current convergence state
```

Do not let configuration or operational binding leak into canonical semantics without a reason.

---

# 19. Reconcile and Projection Records

Projection Records become especially important in this model.

Reconcile can use them to answer:

```text
Which canonical scopes were previously materialized?
Which Module produced their representations?
Which artifacts are owned?
Which canonical revision produced them?
Which artifacts are stale after this canonical diff?
```

Projection Records remain operational provenance.

They are not canonical desired state.

---

# 20. Reconcile example

Suppose canonical intent changes:

```text
Order:
+ deletion requires retention check
```

The semantic graph indicates impact:

```text
Order
↓
DeleteOrder
↓
Orders Module
↓
Application
```

Enabled Projection Modules evaluate that impact:

```text
.NET
→ Order behavior
→ DeleteOrder
→ tests

Markdown
→ Orders behavior documentation

Azure DevOps
→ no change

Codex bootstrap
→ no change
```

Reconcile executes only the relevant scopes.

The user never needs to manually enumerate those artifacts.

---

# 21. Product UX implication

The normal high-level operations should increasingly feel like:

```text
initialize
model
inspect
plan
reconcile
verify/explain
```

not:

```text
implement file X
update docs Y
sync agent rule Z
```

Markitect should operate at desired-state level.

---

# 22. Terraform analogy

Use the Terraform analogy carefully but deliberately.

Similarities:

```text
desired state
current/observed state
plan
apply/reconcile
drift
dependency graph
targeted change
```

Differences:

```text
Markitect projections may be non-deterministic
AI searches representation space
verification is evidence-bounded
one canonical concept may map to many artifacts
one artifact may represent many canonical concepts
semantic correctness cannot be fully reduced to state equality
```

Do not force Terraform's implementation model where it does not fit.

Use the desired-state/reconciliation principle.

---

# 23. Updated preferred operating model

```text
                         USER
                          │
                    changes intent
                          │
                          ▼
                  CANONICAL MODEL
                          │
                     compile/diff
                          │
                          ▼
                    IMPACT GRAPH
                          │
                          ▼
                      RECONCILE
                          │
              load configured targets
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
   .NET Projection    Markdown        Azure DevOps
       Module          Module            Module
          │               │               │
          ▼               ▼               ▼
     work proposal    work proposal      no-op
          │               │
          └───────┬───────┘
                  ▼
            execution DAG
                  │
                  ▼
          Executor / Verifier
                  │
            recursively upward
                  │
                  ▼
              CONVERGED
```

---

# 24. Architectural principle

Add this as a first-class principle:

> **Users change canonical intent. Markitect reconciles representations.**

And this companion principle:

> **Canonical intent should describe desired representations and their semantic requirements, not unnecessarily bind itself to the mechanism that realizes them.**

---

# 25. Required implementation pressure tests

While evolving the current design, explicitly test:

```text
Can a canonical change trigger reconciliation without any explicit
"implement this Definition" command?

Can impact analysis identify only the necessary Projection Modules?

Can Projection Modules independently propose applicable affected work?

Can changing one Projection Module implementation leave canonical
intent unchanged?

Can target/scoping configuration live outside canonical semantics
without losing traceability?

Is a canonical Projection Kind still necessary once configured
Projection Modules and ProjectionPolicy exist?

If Projection remains, can it avoid naming a specific implementation?

Can ProjectionRecords provide enough information to reconcile drift
without canonical Projector bindings?

Can a Module safely return "no applicable work"?

Can conflicting Module ownership/work proposals be detected
deterministically before execution?
```

Preserve negative findings.

---

# 26. Update prior examples

Where prior design material says:

```text
User:
"Implement CreateOrder."
```

replace the default mental model with:

```text
User:
changes CreateOrder intent

Markitect:
plans and reconciles the affected projections automatically
```

Explicit task execution may still exist as a debugging/development tool.

It is not the primary product workflow.

---

# 27. Do not overreact

This addendum does not require deleting working projection contracts or the existing Plan/Apply/Verify mechanics.

The previous projection-first implementation already contains useful machinery.

Prefer to generalize:

```text
existing Plan
→ reconcile planning

existing Apply
→ execution/materialization step

existing Verify
→ scoped verification/evidence
```

where those mechanics fit.

The architectural correction is primarily about:

```text
who decides what work exists
what is canonical
where Projector binding lives
and what the user asks Markitect to do
```

---

# 28. Final direction

The intended product loop is now:

```text
DISCUSS / MODEL
      ↓
CANONICAL INTENT CHANGES
      ↓
COMPILE
      ↓
DIFF
      ↓
IMPACT
      ↓
PLAN
      ↓
RECONCILE
      ↓
VERIFY RECURSIVELY
      ↓
CONVERGED
```

Then repeat.

The user should increasingly think:

> **I changed what the engineering system should be. Markitect will reconcile what that implies.**

rather than:

> **I know which implementation artifacts need to change, so I will tell an agent to change them.**
