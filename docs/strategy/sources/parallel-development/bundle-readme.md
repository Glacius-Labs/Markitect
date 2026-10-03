# Markitect Parallelization Prompt Bundle

This bundle is intended to be handed to the current Markitect implementer **after the currently assigned work is complete and integrated**.

The immediate objective is not another product feature.

The objective is to:

1. establish a stable parallel-development baseline,
2. make the repository itself sufficient context for fresh implementers,
3. verify that subsystem boundaries are strong enough for real parallel work,
4. explicitly test adapter independence,
5. prepare bounded workstream contracts for multiple implementation agents,
6. preserve one semantic authority through a coordinator,
7. use Konfyra as a real adoption consumer rather than another source of accidental Core requirements.

## Files

- `00-coordinator-prompt.md`
  - The prompt to give the current implementer.
- `01-core-invariants.md`
  - Shared architectural/product invariants that all future implementers should receive.
- `02-risk-register-and-triage.md`
  - Known risk areas, likely failure modes, and the expected triage discipline.
- `03-parallelization-and-modularity.md`
  - Parallelization strategy with explicit modularity expectations.
- `04-adapter-modularity-contract.md`
  - Strong adapter independence requirements and a parallelizability test.
- `05-existing-project-adoption-and-discovery.md`
  - Existing-project migration/discovery model.
- `06-init-existing-project-adoption.md`
  - Proposed existing-project `init` / adoption bootstrap.
- `07-workstream-map.md`
  - Suggested parallel workstreams, ownership and dependencies.

## Intended operating model

```text
                    Coordinator
                         │
          ┌──────────────┼───────────────┐
          ▼              ▼               ▼
      Init/Adoption   Copy Me      Adapter Contract
          │              │               │
          │              │        ┌──────┼──────────────┐
          │              │        ▼      ▼      ▼       ▼
          │              │      .NET   Docs   Codex   Claude ...
          │              │
          └───────┬──────┘
                  ▼
             Konfyra Adoption
                  │
                  ▼
               Evidence
                  │
                  ▼
             Coordinator
```

The key rule is:

> **Centralize semantic authority; decentralize bounded execution.**

And, specifically for adapters:

> **If two unrelated adapters cannot be implemented concurrently from one stable semantic/adapter contract without touching one another or the Core, treat that as an architectural smell that must be explained.**
