# Release decisions

**Scope:** Product and release decisions for source version `0.2.0`. Publication is established by the exact GitHub Release and its verified assets. See the [roadmap](implementation-plan.md), [source release evidence](production-assessment.md), and [operations](operations.md).

## Product decisions

1. **Standalone ownership.** Markitect owns its Go source, YAML vocabulary, schemas, compiler, core authoring resources, generic examples, and release assets. A project using Markitect owns its local policy, declared checks, output adapters, import scripts, and acceptance.
2. **Explicit typed data.** Use YAML for machine-relevant identity, ownership, references, inputs, checks, and output selection. Keep ordinary prose in Markdown. Schemas help editors; strict parsing and graph validation remain authoritative.
3. **No inferred execution.** A Project declares each command to run. The CLI passes argv directly without shell interpretation, and `verify` reports incomplete evidence when no checks are declared. Repository identity and file discovery do not select gates.
4. **No implicit compatibility outputs.** Generic views are available by default. Extra rendering requires a Project declaration. Product core does not carry project-specific import or render policy.
5. **Core authoring ships with the product.** Portable resource modelling, query guidance, and authoring workflows are embedded with the tool; no optional knowledge package or model runtime is needed to validate structure.
6. **Immutable, versioned distribution.** The v0.1.0 release remains unchanged. A v0.2.0 version may be published only after its exact source commit passes the documented gates and the assets verify against an immutable GitHub release.
7. **Evidence has limits.** Hashes and fixed snapshots bind bytes, not truth or semantic completeness. No source check claims an agent followed prose or a project made a human decision.

## Deferred

Content packages, project initialization, MCP/LSP, runtime operators, public licensing, and a controlled public API domain remain separate product decisions. They are not implied by the current source version.
