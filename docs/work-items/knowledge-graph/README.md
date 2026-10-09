# Knowledge Graph workstream

Owner: chat `01a12178-5797-7c41-8acc-8e8231084c79`, independent branch `codex/knowledge-graph`.
Reference: Management/P01/P02 source `1495e1be7b0046711531fa42c8407fba67b8e814`; continuation starts at `0a56d1cea65b53aec0152f2de30f8d1fa28600fd`.

The new direct user mandate authorizes full KG implementation in this isolated variant, including necessary schema/Host/CLI/MCP/guidance changes. Historical Integration-only holds in older handoffs no longer apply here. Work does not write the Integration branch or add a Main gate. After Root reports exact accepted Main, merge that source into this branch and validate the combined candidate; no automatic KG-to-Main merge, release or study.

- [Backlog](backlog.md): durable current work and owners.
- [Continuation design](continuation-design.md): accepted local intent and parallel seams.
- [KG01 reference](KG01-reference.md) and [twelve questions](KG01-question-matrix.md): source-bound acceptance contract.
- [Pure index API](KG02-KG03-api.md): existing projection/navigation contract.
- [Validation ledger](validation.md): actual source checks and immutable failed Verify.
- [Historical finite-slice handoff](handoff.md): original ownership and completed KG01–03 evidence.

YAML stays canonical. RDF/SPARQL is evaluated separately if justified; engines, migration, purchases and case-study/provider starts are not authorized by developer checks.
