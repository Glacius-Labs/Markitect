# Coordinator Steer — Consolidated Latest Architecture Clarifications

Apply these clarifications to the **current architecture implementation before considering the current phase finished**.

This is a consolidation of the latest owner decisions. Prefer this document over earlier addenda where wording conflicts.

Do not restart the implementation. Preserve and generalize useful existing Plan/Apply/Verify, projection, artifact-accounting, provenance and drift-repair mechanics where they fit.

---

# 1. Strict Module types

Replace the permissive model:

> A Module may provide Schemas, Projectors, or both.

with:

```text
Module
├── Schema Module
└── Projection Module
```

Every Module is exactly one type.

## Schema Module

Purpose:

> Extend what can be expressed canonically in Markitect.

May provide:

```text
Schemas
Kinds
Properties
semantic purposes
reusable semantic vocabulary
```

Must not:

```text
project artifacts
write target repositories
contain target-specific projection behavior
synchronize with Projection Modules
```

Examples:

```text
DDD
Vertical Slice
CQRS
Clean Architecture
```

## Projection Module

Purpose:

> Teach Markitect's execution system how to represent canonical intent in one target technology/provider/representation.

May provide:

```text
target expertise
allowed target surfaces
tools
agent guidance
discovery signals
verification guidance
```

Must not:

```text
introduce canonical Kinds
redefine source semantics
treat sibling Module artifacts as authority
synchronize with sibling Projection Modules
```

Examples:

```text
.NET
Go
Markdown
Azure DevOps
Codex
Claude
Git Hooks
```

If one ecosystem needs both semantic vocabulary and target projection, use two Modules.

Registry bundles may install several Modules for convenience; Bundle is not a third Module type.

---

# 2. AI-first projection

Projection should have one conceptual execution model:

```text
bounded canonical scope
      ↓
Executor Agent
      ↓
candidate representation
      ↓
Verifier Agent + independent evidence
```

The Executor may use deterministic tools:

```text
renderers
generators
compilers
formatters
SDKs
CLIs
templates
```

but deterministic projection is not a separate top-level architecture.

Preserve working deterministic renderers as tools/evidence where useful.

Likewise verification is agent-orchestrated but may depend heavily on deterministic evidence.

Do not let Executor-authored tests alone count as independent assurance.

---

# 3. Durable engineering meaning is canonical; provider artifacts are projections

Foundation should model durable concepts such as candidates:

```text
Goal
Rule
Process
Responsibility
Invariant
ProjectionPolicy
```

Only add those proven necessary.

Provider-specific artifacts are normally projections:

```text
AGENTS.md
CLAUDE.md
.codex/skills/*
.claude/*
provider-specific Agent files
provider-specific Workflow files
human Markdown documentation
CI enforcement generated from Rules
```

Use this test:

> Would this concept still make semantic sense if all current providers and file formats disappeared?

If yes, it may be canonical.
If no, it is likely a projection representation.

Examples:

```text
Release Process
= canonical

Codex release Skill
= projection

Claude release Workflow
= projection

Markdown release documentation
= projection

Azure pipeline enforcement
= projection
```

Claude and Codex may have separate Projection Modules initially.

Long-term a thin provider bootstrap may request scoped Markitect context directly.

Support both directions without making provider artifacts authoritative.

---

# 4. Reconcile-first desired-state workflow

The primary user behavior is not:

```text
"Implement CreateOrder."
```

It is:

```text
change canonical intent
      ↓
plan
      ↓
apply reconciliation
```

Use desired-state semantics internally:

```text
reconcile
= bring realized representations toward canonical desired state
```

User-facing Terraform-like commands are preferred:

```text
markitect plan
markitect apply
```

`plan` computes reconciliation.
`apply` executes it.

Do not create a competing second concept if `apply` already means executing reconciliation.

The primary principle is:

> **Users change canonical intent. Markitect reconciles representations.**

---

# 5. Projection capability must not be unnecessarily hard-bound in canonical intent

Pressure-test whether a canonical `Projection` Kind is needed.

Concern:

```text
canonical Projection
→ names concrete Projector implementation
```

makes replacing the implementation mechanism look like an engineering-intent change.

Prefer:

```text
canonical semantic model
+
enabled/configured Projection Modules
+
canonical ProjectionPolicies
+
operational ProjectionRecords
```

During `plan/apply`, Projection Modules evaluate the affected canonical graph and propose the work relevant to their configured target/scope.

If a canonical Projection concept remains useful, it should describe only desired representation semantics such as:

```text
representation type
canonical scope
target/location
semantic representation requirements
applicable ProjectionPolicies
```

It should not unnecessarily name:

```text
specific package
specific Projector version
specific Executor implementation
```

Binding belongs in project/runtime configuration.

Explicitly reconsider whether `Projector` needs to remain a public first-class noun once Projection Modules + Executor exist.

---

# 6. Projection Modules may discover applicable affected work

During plan/reconcile:

```text
canonical diff
      ↓
semantic impact
      ↓
enabled Projection Modules
      ↓
each Module evaluates affected graph
      ↓
work proposals / no-op
      ↓
ownership-conflict resolution
      ↓
execution DAG
```

A Module may return:

```text
work
no applicable work
missing projection intent / escalation
```

It must not create new canonical business/architecture intent.

Missing representation semantics must escalate.

---

# 7. ProjectionPolicy is the project-owned semantic bridge

Avoid:

```text
DDD knows .NET
.NET hard-codes DDD
```

Prefer:

```text
DDD semantics
+
project-specific ProjectionPolicy
+
.NET Projection Module
+
target context
      ↓
Executor Agent
      ↓
one valid C# representation
```

ProjectionPolicy should express representation constraints/decisions, not become a full code-generation DSL.

This preserves:

```text
R ∈ ValidRepresentations(M)
```

rather than forcing:

```text
R = f(M)
```

---

# 8. Brownfield has a reverse inference axis

Greenfield/normal operation is forward:

```text
Canonical Intent
      ↓
plan/apply
      ↓
Representations
```

Brownfield adoption initially needs the reverse direction:

```text
Existing Representations
      ↓
analyze / infer
      ↓
Candidate Intent
      ↓
review / correction
      ↓
Canonical Intent
```

This is **inference**, not authority.

Use the mental model:

```text
Reality ──infer──> Candidate Intent
                    │
                  approve
                    ▼
              Canonical Intent
                    │
                 plan/apply
                    ▼
                  Reality
```

After adoption, reverse inference remains diagnostic/proposal-only:

```text
observed new semantics
→ proposed canonical change / escalation
```

Never silently mutate canonical intent from target artifacts.

---

# 9. Brownfield adoption must establish authority before writes

A non-empty repository must not go directly from `init` to destructive apply.

Required posture:

```text
inventory read-only
→ Module discovery
→ candidate model inference
→ owner review
→ canonical acceptance
→ artifact/scope matching
→ verification
→ adopt valid representations as baseline
→ resolve unknown/drift
→ only then normal plan/apply
```

Existing valid representations should be **adopted, not regenerated**.

A valid existing representation can become MANAGED with operational provenance such as:

```text
origin: adopted
```

without changing its bytes.

Zero-churn adoption should be a first-class proof target.

---

# 10. Artifact safety

Relevant artifacts remain classified as:

```text
MANAGED
IGNORED
EXCLUDED
UNKNOWN
```

UNKNOWN does not mean rewrite/delete.

It means ownership/meaning is unresolved.

Brownfield apply should be conservative:

```text
exact canonical revision
exact Git revision
stale-plan rejection
artifact digest binding
ownership-conflict detection
no unexplained deletion
no broad rewrite for style preference
isolated branch/worktree
Git-backed reversibility
```

Large destructive changes during initial adoption should be surfaced explicitly before execution.

---

# 11. Current architecture principle set

Treat these as current owner decisions:

> **Markitect is the single semantic authority.**

> **Schema Modules extend what can be said.**

> **Projection Modules teach agents how to represent intent in a target.**

> **Modules do not synchronize with each other.**

> **Provider Skills, Workflows, agent definitions and documentation are normally projections of durable canonical meaning.**

> **Users change intent; Markitect plans and applies reconciliation.**

> **Canonical intent should not unnecessarily know its execution mechanism.**

> **Brownfield reverse inference proposes intent; it never silently creates authority.**

> **Existing valid representations are adopted rather than normalized into an Executor's preferred form.**

Incorporate these before declaring the current implementation phase complete.
