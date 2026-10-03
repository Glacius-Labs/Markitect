# Markitect — Parallelization and Modularity Strategy

## Central principle

Use:

> **One coordinator for semantic authority + many bounded implementers for execution.**

This does not mean all development should be sequential.

On the contrary:

> **Parallelizability should be used as an architecture quality test.**

---

## Three classes of work

### Class A — should be independently parallelizable

Examples:

```text
.NET Adapter
GitHub Adapter
Azure DevOps Adapter
Docs Adapter
Codex Adapter
Claude Adapter
```

Expected:

- same stable Semantic Model contract,
- same stable Adapter Contract,
- no adapter-to-adapter dependency,
- no provider-specific Core changes,
- independent tests,
- independent target ownership.

If two such adapters cannot be built concurrently, investigate the coupling.

### Class B — coupled by product workflow, but largely parallelizable

Examples:

```text
Existing-project Init
Copy Me
Konfyra Adoption
Authoring UX
```

These have conceptual dependencies, but should have explicit boundaries.

Possible concurrency:

```text
Init agent:
control-plane/bootstrap

Copy Me agent:
candidate/evidence/decision semantics

Konfyra agent:
inventory/evidence mapping
```

Integration happens later.

### Class C — shared compiler/kernel semantics

Examples:

```text
Semantic IR
Policy evaluation
Impact semantics
Constraint language
Core state model
Reconciliation semantics
```

Stronger coordination is normal here.

Do not confuse legitimate shared-kernel coordination with avoidable adapter coupling.

---

## Coordinator role

Coordinator owns:

```text
shared contracts
Core invariants
cross-workstream decisions
integration order
release readiness
```

Coordinator should not become the only implementer.

---

## Worktree model

Example:

```text
main

codex/init-adoption
codex/copy-me
codex/konfyra-adoption
codex/adapter-dotnet
codex/adapter-github
codex/adapter-azure
codex/adapter-docs
codex/adapter-codex
codex/adapter-claude
codex/risk-triage
```

---

## Integration model

```text
Implementer
    ↓
local validation
    ↓
PR
    ↓
CI
    ↓
independent review
    ↓
Coordinator checks shared-contract compatibility
    ↓
merge
```

---

## Shared hot spots

Potential hot files are:

```text
README.md
docs/architecture.md
docs/implementation-plan.md
CLI command registration
schema generation
semantic model types
adapter interfaces
shared test helpers
```

For each hotspot ask:

> Is this legitimate shared contract ownership, or accidental central coupling?

Avoidable registration/switch hotspots should be treated as modularity findings.

---

## Immediate desired wave

After baseline preparation:

```text
Risk Triage ----------------------------┐
                                        │
Existing-project Init --------┐         │
                              ├─> Copy Me
Konfyra Inventory ------------┘         │
                                        │
Adapter Contract Audit --------┬─> .NET │
                               ├─> Docs │
                               ├─> Codex│
                               ├─> Claude
                               ├─> GitHub
                               └─> Azure DevOps
```

Konfyra can begin inventory/evidence work early.

Full adoption should consume stable Init/Copy Me contracts rather than inventing them locally.

---

## Important modularity test

For every subsystem:

```text
Can two independent implementations be built
from the current contract
without editing shared semantic code?
```

If no:

1. identify why,
2. determine whether the dependency is legitimate,
3. if avoidable, repair the boundary before scaling parallel implementation.

---

## Release strategy

Prefer:

```text
parallel implementation wave
        ↓
integration
        ↓
stabilization
        ↓
cross-workstream verification
        ↓
coherent release
```

rather than release-per-agent.

---

## Architectural mirror

The development organization should reflect the product architecture:

> **Centralize semantic authority; decentralize bounded execution.**
