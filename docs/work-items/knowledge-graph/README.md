# Knowledge Graph workstream

Owner: chat `01a12178-5797-7c41-8acc-8e8231084c79`, branch `codex/knowledge-graph`.
Pinned reference: `1495e1be7b0046711531fa42c8407fba67b8e814` (Management/P01/P02 source, not main).

This independently assigned work is outside P01–P10 and does not add a Main gate. Integration reserved only new pure files under `internal/modules/projectknowledge`. Later P08 path changes require an explicit rebase. Existing shared sources, schemas, dependency manifests, Host adapters and CLI/MCP composition remain Integration-owned.

- [KG01 reference](KG01-reference.md): implemented mechanisms, provenance and executed startup checks.
- [KG01 twelve-question acceptance matrix](KG01-question-matrix.md): reproducible examples and explicit gaps.
- [KG02/KG03 API](KG02-KG03-api.md): pure projection, bounded navigation and adapter ownership.
- [Validation ledger](validation.md): executed source-bound checks and remaining coverage gaps.
- [Local handoff](handoff.md): finite work items, API ownership and validation.

KG02 implements a rebuildable model index over Core and explicit neutral DTOs. Read-only KG03 returns bounded witness paths over an explicitly selected view. It does not replace Context or conservative Impact. KG04 identity/vocabulary, KG05 operational evidence and KG06/07 product integration require separate Integration assignments. RDF mapping can reuse KG01; engine installation, canonical RDF migration and study execution are unallocated.
