# Markitect — Adapter Modularity Contract

## Why this is important

Adapters are the clearest place where Markitect should demonstrate modularity.

The target architecture is:

```text
                    Semantic Model
                         │
                  Adapter Contract
                         │
       ┌─────────────────┼────────────────────┐
       ▼                 ▼                    ▼
   .NET Adapter      GitHub Adapter      Azure Adapter
```

Not:

```text
Adapter A
   ↓
Adapter B
   ↓
Adapter C
```

---

## Strong invariants

> **Adding a new adapter must not require modifying another adapter.**

> **Two adapters targeting unrelated systems should be implementable concurrently from one stable semantic-model/adapter contract.**

> **Provider-specific details belong in adapter/configuration boundaries, not the semantic Core.**

> **An adapter must reconcile against canonical semantics, never another adapter's generated representation.**

---

## Expected generic lifecycle

Where applicable:

```text
Consumes
Observe
Plan
Apply
Verify
Project
```

Not every adapter needs every capability.

Capabilities should be explicit.

---

## Adapter input

Adapters should consume a validated normalized semantic model or another stable provider-neutral contract.

They should not parse arbitrary authoring YAML independently if the semantic model already exists.

---

## Target ownership

Every adapter should declare which external/generated targets it owns.

Two adapters should not silently own the same target.

Conflicting ownership should fail explicitly.

---

## Observation

Observe should state:

```text
what was inspected
which target/system
which inputs
which tool/provider version where relevant
what evidence was complete/incomplete
```

Observation is not canonical truth.

---

## Plan

Plan should be:

```text
deterministic
read-only
bound to source/model/config/observation identity
```

A changed input should stale the plan.

---

## Apply

Apply must be explicit.

No hidden mutation in:

```text
check
model
context
impact
observe
plan
verify
```

Apply should operate only on reviewed plan operations.

---

## Verify

Verify should determine whether the target now matches the adapter's declared desired state/evidence contract.

It must not overclaim broader semantic correctness.

---

## Provider mapping

Bad:

```text
Core Module:
azureAreaPath
githubRepositoryId
```

Preferred:

```text
Canonical:
Module/Survey

Azure adapter mapping:
Modules/Intake/Survey

GitHub adapter mapping:
repository/team/path semantics
```

---

## No adapter chains as truth

Bad:

```text
Claude Adapter
    ↓
reads generated Markdown
    ↓
Docs Adapter
    ↓
canonical data
```

Preferred:

```text
Canonical semantic model
    ├─ Docs Adapter
    └─ Claude Adapter
```

Both consume canonical meaning independently.

---

## Parallelizability acceptance test

Before adapter fan-out, answer:

1. Can `.NET` and `GitHub` be implemented from the same baseline?
2. Can `Azure DevOps` and `Claude` be implemented from the same baseline?
3. Does any adapter require editing another adapter?
4. Does any adapter require provider-specific Core fields?
5. Is there a central switch that must be edited for every adapter?
6. Can adapters register/declare themselves without semantic coupling?
7. Can each adapter test independently?
8. Can each adapter declare evidence limitations independently?
9. Can each adapter own its targets independently?
10. Can an adapter be absent without changing semantic Core behavior?

A "no" is not automatically a defect, but it requires explanation.

---

## Preferred long-term direction

A stable protocol should make it possible, in principle, for adapters to be implemented outside the Core repository or even in other languages.

Conceptually:

```yaml
adapter:
  name: azure-devops
  protocol: v1

consumes:
  - Module
  - Product
  - Release

capabilities:
  observe: true
  plan: true
  apply: true
  verify: true
```

This exact syntax is illustrative, not prescribed.

The design goal is a stable semantic ABI/SPI.

---

## Non-goals

Do not build a large plugin runtime merely to satisfy abstraction aesthetics.

Only introduce infrastructure required by concrete adapter independence, target ownership, evidence or lifecycle needs.

The smallest stable contract that permits safe independent adapters is preferable.
