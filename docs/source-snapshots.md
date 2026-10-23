# Source snapshots

Product snapshot values, repository acquisition, canonical model digests and maintenance source packaging have different responsibilities. [Architecture](architecture.md) and [dependency rules](development/modules.md) describe their current owners.

## Current value

`internal/snapshot` provides immutable exact repository-relative paths, modes, entries, content identities and fixed trees. Construction rejects invalid spellings and collisions. Deterministic comparisons operate on supplied values without Git or filesystem I/O. A tree digest and file content identity are not canonical model digests.

`internal/model` checks supplied canonical sources and produces model digests with source provenance. `internal/organization` and `internal/realization` consume these pure facts. Neither the Model check nor deterministic values infer physical repository identity or authorize a write.

## Adapter and use-case ownership

| Responsibility | Owner |
| --- | --- |
| Committed clean capture, working/fixed previews, exact source roots and census | `internal/adapters/gitrepo` |
| Acquisition/metadata ports and source-bound preparation | `internal/apply` |
| Pure path/file/tree identities and comparisons | `internal/snapshot` |
| Pure canonical meaning and model digest | `internal/model` |
| Registry, file classes, ownership and candidates | `internal/realization` |
| Generated view plan and independent guarded target write | `internal/modeldocs`, `internal/adapters/documents` |
| Fixed release-source bytes/modes and package provenance | `internal/tooling/source`, `internal/tooling/release` |

A new input closure requires a fresh complete capture. Preparation never extends an old guard or fetches extra bytes into it.

## Identity and compatibility

Committed Source capture returns actual HEAD and an opaque comparison guard after validating a clean committed repository. PreviewSource observes a working draft or an explicitly resolved full commit and returns no publication guard. Fixed preview refuses current external inputs without historical binding. The complete census includes Git-ignored untracked files and excludes only technical Git administration and `.markitect/state/**`.

The maintenance-only `source.Input` retains fixed source files, modes and commit provenance for release packaging and installation. Its retained fingerprint is not the product snapshot.Tree digest, canonical model digest, Manager DTO or a publication guard. Current packaging accepts one root `cmd/` and `internal/` layout. Published archives keep their original bootstrap and identity.

## Original v0.8.1 audit (historical)

Earlier snapshot encodings and audits retain their original scope in [the preceding guide](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/source-snapshots.md). That history is not the current compiler or acquisition contract.

## Security and verification limits

Physical identity, exact source spellings, redirect refusal, bytes and finite acquisition budgets are adapter obligations. The pure Model check establishes none of them. A passed preparation report does not establish a real session, check execution, verified candidate, operating-system sandbox or Apply readiness. Document writes use a separate source/target-bound guard and retain actual partial effects on failure.
