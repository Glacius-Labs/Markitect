# Project knowledge

Markitect can build a bounded, read-only graph view from the selected project YAML model and explicitly selected records. The YAML model remains canonical; the graph is derived for answering relationship questions and is not persisted as a second source of truth. It does not infer business meaning from code or prose, replace execution dependencies, establish identity, or prove human acceptance. Standard RDF and SPARQL are not part of this interface.

## Choose visibility first

Each query uses exactly one scope: a Manager view or a whole-project view. A Manager view follows the project model's existing Context visibility rules. Use whole-project scope only when the task and repository policy authorize that visibility. Core definition IDs are the complete JSON tuple `[apiVersion, kind, namespace, name]`; copy the exact `nodes[].id` returned by a visible graph query rather than constructing an ID from a name.

The CLI accepts these actions: `graph`, `relations`, `explain`, `trace`, `history`, and `coverage`. For example, PowerShell commands for a Manager-scoped view are:

```powershell
markitect project knowledge --repo . --manager '["project.markitect.example.org/v1alpha1","Manager","","project-owner"]' --knowledge-action graph
markitect project knowledge --repo . --manager '["project.markitect.example.org/v1alpha1","Manager","","project-owner"]' --knowledge-action coverage
```

Use `graph.nodes` and `graph.edges` in the command response to identify the exact node ID and relation property of interest. The response wraps graph data and evidence together, so the node field path is `graph.nodes[].id`. Then query a node and its incident edges, follow bounded paths, or inspect explicit history:

```powershell
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action explain --node-id NODE_ID
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action relations --node-id NODE_ID
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action trace --node-id NODE_ID --max-depth 6 --max-steps 1000 --max-results 100
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action trace --node-id NODE_ID --bidirectional --max-depth 6 --max-steps 1000 --max-results 100
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action history --node-id NODE_ID
```

Replace `MANAGER_ID` and `NODE_ID` with the exact values from the selected model or preceding response. In particular, copy an ID from `graph.nodes[].id` without reconstructing it. Use `--reverse` with `trace` to follow incoming relationships. `--bidirectional` searches both directions and preserves each witness edge's declared orientation. `explain` returns the selected node and its visible relation witnesses; it does not generate a natural-language conclusion. `history` includes explicit Decision and IdentityChange declarations around the selected node; an IdentityChange is a recorded historical identity claim, not proof that the old definition existed or that the concepts are equivalent.

To request a whole-project view, replace `--manager MANAGER_ID` with `--knowledge-scope project`. Do not pass both selectors. Query output is bound to the project/model/snapshot and visibility scope so consumers can distinguish different views.

## Records are opt-in

The default query uses the selected model and known repository facts only. Operational records remain excluded unless the question requires particular records. Select only the needed records explicitly:

```powershell
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action graph --run RUN_ID
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action graph --exploration EXPLORATION_ID
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action graph --session SESSION_ID
markitect project knowledge --repo . --manager MANAGER_ID --knowledge-action history --node-id NODE_ID --briefing-history
```

The selectors read existing records. They do not start a provider or update a run, exploration, session, or briefing ledger. A query can report facts as current, partial, stale, historical, or unknown according to their actual model and source bindings. Missing record chains remain unknown; an empty or missing result is not evidence that nothing happened. Preserve the returned binding, coverage state, source bindings, and query limits when reporting what the graph supports.

## MCP transport

Start the stdio server from the selected Markitect source build:

```powershell
markitect project knowledge-mcp --repo PATH
```

The server exposes six read-only tools: `knowledge_graph`, `knowledge_relations`, `knowledge_explain`, `knowledge_trace`, `knowledge_history`, and `knowledge_coverage`. Each tool call supplies exactly one `managerId` or `projectScope: true`; target-oriented tools take the exact `targetId` from `graph.nodes` in the graph response. Trace calls may set finite `maxDepth`, `maxSteps`, and `maxResults`, plus `reverse` or `bidirectional` traversal. The CLI and MCP transports use the same project knowledge service and scope rules.

This server currently pins the MCP handshake subset to protocol versions `2025-11-25`, `2025-06-18`, and `2024-11-05`. It requires the normal `initialize` then `initialized` sequence. An unknown requested version falls back to `2025-11-25`; `structuredContent` is emitted for `2025-11-25` and `2025-06-18`, and omitted for `2024-11-05`.

## Limits and ownership

Model structure, artifact and check coverage, and optional record completeness are reported separately. `known`, `partial`, and `unknown` describe available facts; they are not success grades. A trace's `complete` flag describes reachability within the selected graph and requested limits; it does not establish that the project knowledge is complete. A removed relation disappears from a rebuilt graph for the selected model revision. A fixed old model queried with separately selected live records keeps those sources and bindings distinct. There is no automatic discovery of every historical record, no unrestricted repository crawl, and no semantic equivalence inference.

YAML owners keep declarations and explicit Decisions or IdentityChanges at their canonical source. Each canonical definition has one responsible Manager; graph visibility follows those existing strict ownership and privacy rules, and a public record does not expose a foreign private subject. A generated readable project document may present those records with source provenance, but editing that view does not change the model. An actor field records who the model says made a decision; it does not authenticate a person. Graph results, source citations, passing checks, agent reports, and provenance do not establish human acceptance. Follow the [model-first workflow](project-workflow.md) for authority, ownership, impact, and evidence decisions. The [small executable knowledge example](../examples/project-knowledge/README.md) provides a provider-free graph walkthrough.
