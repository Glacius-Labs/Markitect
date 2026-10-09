# KG02 and read-only KG03 API handoff

Pure module: `internal/modules/projectknowledge`. Imports are Core plus the standard library; the module does not load repository files, import ProjectModel/Host, execute checks, write ledgers or infer policy.

```go
index, err := projectknowledge.Build(compiledModel, facts, scope)
node, err := index.Node(fullIdentityKey)
incoming, err := index.Incoming(fullIdentityKey)
outgoing, err := index.Outgoing(fullIdentityKey)
result, err := index.Walk(projectknowledge.WalkRequest{
    Start: fullIdentityKey, Reverse: false,
    MaxDepth: 8, MaxSteps: 5000, MaxResults: 500,
})
```

This is an internal Go seam, not an existing CLI/MCP command. The caller supplies trusted compiled Core data and explicit bindings. Supplied snapshot, project and fact digest labels remain caller assertions; the module cannot validate them against repository bytes. Its own graph digest reproducibly binds the selected projection and supplied binding labels.

## Projection and privacy contract

`Scope` requires a nonempty scope ID and node selection list. Each `NodeSelection` explicitly chooses Core/full fact ID, literal top-level fields, Purpose and Source. `Scope.Edges` independently allows exact `(From, To, Property)` keys with both endpoints selected. Zero scope never means the full graph. Core reference/kind-reference values are recursively removed from literal properties, even when their property is selected; only permitted edges expose reference endpoints. Literal lists preserve order/repetition and selected absent fields stay absent. Supplemental JSON preserves number lexemes without converting integers to float64.

The Host adapter owns Manager/public/privacy policy, selected text and neutral fact content. Public foreign contract selection does not authorize all outgoing edges; child-manager selection does not authorize Instructions. Select only the fields/relations existing Context permits. Edge selection includes its provenance; only admit an edge if that provenance is permitted. Build errors are internal trusted-adapter diagnostics, not a manager-facing privacy-safe endpoint. Query lookups on the completed index give the same `ErrNotFound` for unselected and absent IDs.

Neutral `ProjectFacts` supplies `Fact` and `Relation` values plus required snapshot/fact digest labels and optional project digest. File presence, declared ownership, artifact membership, expected selectors and declaration-check relationships must be mapped explicitly from existing authoritative reports outside this module. `Fact.State` is known/partial/unknown completeness of the supplied fact only. It does not say an execution succeeded or evidence is current. Omitted fact properties differ from explicit empty values. Derived Relation source and opaque Basis retain the adapter's derivation; they do not become editable ownership declarations.

Graph bindings carry version, model digest, revision, snapshot/project/facts digests, scope ID and reproducible graph digest. Nodes/edges are stably sorted and accessors return copies. Existing Decisions are ordinary Core nodes: selected reason/decision literals plus independently selected subject/actor edges, without vocabulary changes or authenticated-approval claims.

## Navigation and limits

Walk follows outgoing or incoming selected relations in deterministic breadth-first order and returns one shortest witness per reached node, including a zero-length witness for the start. Edges in reverse witnesses retain their original From/To orientation. This does not enumerate every possible path; use Incoming/Outgoing to inspect parallel typed links such as uses and requires. Result carries graph bindings, query version/digest, inspected edge count and explicit completion/reason. Depth/step/result truncation is partial, never a complete no-op. Complete refers only to reachability in the supplied selected graph; unknown/partial fact content remains unknown/partial even when traversal is complete.

Current hard ceilings: 5,000 nodes, 20,000 edges, 8 MiB graph payload; query depth 32, inspected edges 50,000, results 5,000. Caller supplies positive finite depth, step and result limits; zero limits are rejected. Index construction also rejects excessive supplied model/fact counts. These are initial technical ceilings, not measured production scalability promises.

A one-snapshot walk is not Impact. Removed edges require explicitly selected base/candidate data and existing Impact semantics, which remain untouched. Uses remains a declared context/routing relationship; requires has additional coverage meaning owned by the existing project layer. Semantic graph traversal does not schedule Manager execution. Operational trace, binding freshness, coverage certification, history/rename continuity, CLI/MCP exposure and adapter integration remain separately allocated work.

## Integration responsibilities

Integration owns report-to-neutral-facts mapping, privacy selection, operational adapters, canonical identity/vocabulary additions, shared application services, CLI/MCP/skills, managed-artifact declarations and the eventual P08 path rebase. This module work does not change P01–P10 or introduce a Main gate. RDF export/engine/canonical migration are separate evaluate-then-decide items.
