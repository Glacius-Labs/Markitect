# Markitect — Core Invariants for Parallel Work

These are the architectural constraints every implementer should receive.

## Semantic kernel

1. Core remains deterministic.
2. Source syntax is not the semantic model.
3. Semantic relationships are explicit and typed.
4. Hidden semantic dependencies are a design smell.
5. The kernel should remain smaller than the domains modeled with it.
6. Composition is preferred over inheritance.
7. Core constraint evaluation must terminate and remain side-effect free.
8. No arbitrary policy/query language should emerge accidentally.

## State separation

Keep these distinct:

```text
Canonical
Observed
Generated
Inferred
Proposed
```

Rules:

- observed state never silently overwrites canonical desired state,
- generated projections are never canonical sources,
- inferred candidates never become canonical without explicit adoption,
- AI is not canonical authority.

## Domain vs technology

- Domain semantics define engineering meaning.
- Adapters translate that meaning into technology-specific state.
- Provider-specific concepts must not leak into Core unless they are genuinely domain-neutral.
- Adapters reconcile against canonical semantics, not another adapter's output.

## Validation semantics

Distinguish:

```text
structurally invalid
structurally valid + policy passing
structurally valid + policy failing
structurally valid + waived policy
```

- structural invalidity remains a hard blocker,
- ordinary policy failure may be analyzed read-only when explicitly requested,
- Check and Verify remain strict,
- diagnostic analysis does not mean acceptance.

## Change and reconciliation

- canonical architecture may evolve explicitly,
- exact package/version changes are review events,
- reconciliation plans must be deterministic,
- Apply is an explicit mutation boundary,
- approval binds exact plan/input identity,
- reconciliation should converge,
- external mutation must never hide inside read-only commands.

## Impact

- unknown impact broadens conservatively,
- local explicit dependencies should produce local impact where sound,
- false precision is worse than an explicit conservative result,
- impact results should explain their causes.

## Exceptions

- exceptions are explicit deviations,
- narrow scope,
- exact subject/constraint binding,
- semantic changes should stale outdated exceptions,
- exceptions are not an analysis-access mechanism,
- exceptions do not make structural errors waiverable.

## Evidence

Every check/adapter must say what it proves.

Examples:

```text
declared dependency
build dependency
runtime dependency
semantic dependency
```

must not be conflated.

## Source code

Core should not claim generic understanding of:

```text
AST
symbols
classes
methods
call graphs
business semantics
```

unless a specialist adapter/check explicitly provides bounded evidence.

## Provider independence

Canonical engineering meaning should outlive:

```text
Codex
Claude
GitHub
Azure DevOps
Jira
.NET
```

Provider-specific representations are projections/adaptations.

## Adapter independence

Strong invariant:

> Adding a new adapter should not require modifying another adapter.

Strong default:

> Adding a provider-specific adapter should not require provider-specific changes to the semantic Core.

Unrelated adapters should be independently parallelizable from the same stable contract.

## Core expansion

Do not add a new primitive because one fixture is awkward.

Require:

```text
repeated concrete need
clear generic semantics
finite deterministic contract
explainable diagnostics
sound impact semantics
versioning behavior
```

before extending Core.

## Claims

Do not infer:

```text
productivity
token savings
fewer defects
safer autonomy
market demand
```

from synthetic fixtures or one uncontrolled pilot.

Separate:

```text
technical proof
workflow observation
human acceptance
business benefit
```
