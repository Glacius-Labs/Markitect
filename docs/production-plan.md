# Release decisions

**Scope:** Product and release decisions for Markitect source and distributions. See the [roadmap](implementation-plan.md) for source status, the [production assessment](production-assessment.md) for dated evidence, and [operations](operations.md) for release gates.

## Product decisions

1. **Standalone ownership.** Markitect owns its Go source, YAML vocabulary, schemas, compiler, core authoring resources, generic examples, and release assets. A project using Markitect owns its local policy, declared checks, output adapters, import scripts, and acceptance.
2. **Explicit typed data.** Use YAML for machine-relevant identity, ownership, references, inputs, checks, and output selection. Keep ordinary prose in Markdown. Schemas help editors; strict parsing and graph validation remain authoritative.
3. **No inferred execution.** A Project declares each command to run. The CLI passes argv directly without shell interpretation, and `verify` reports incomplete evidence when no checks are declared. Repository identity and file discovery do not select gates.
4. **No implicit compatibility outputs.** Generic views are available by default. Extra rendering requires a Project declaration. Product core does not carry project-specific import or render policy.
5. **Core authoring ships with the product.** Portable resource modelling, query guidance, and authoring workflows are embedded with the tool; no optional knowledge package or model runtime is needed to validate structure.
6. **Immutable, versioned distribution.** Published releases remain unchanged. Release each new version only after its exact source commit passes the documented gates and its assets verify against an immutable GitHub release. The public repository uses the Apache License 2.0; CLI distribution remains distinct from adopting-project acceptance.
7. **Evidence has limits.** Hashes and fixed snapshots bind bytes, not truth or semantic completeness. No source check claims an agent followed prose or a project made a human decision.

## Deferred

MCP/LSP, runtime operators, and a controlled public API domain remain separate product decisions. They are not implied by the current source version.
