# Scoped assurance composition

```go
report, err := assurance.Evaluate(assurance.Input{
    RootIDs: []string{"application"},
    Nodes: []assurance.Node{
        {ID: "application", ScopeIDs: []string{"application"}, Children: []string{"handler"}, RequiredChecks: parentChecks},
        {ID: "handler", ScopeIDs: []string{"application"}, RequiredChecks: handlerChecks},
    },
    Evidence: suppliedEvidence,
})
```

The package composes caller-supplied projection records and verification results over an explicit finite DAG. Callers name the requested roots, declare every node scope and required check identity, and supply the current freshness binding for each evidence item. A node's own record and verification are independent requirements; child results never replace a missing or failed parent verification.

Children may reference only declared nodes, every child scope must be a subset of its parent scope, and cycles, duplicate IDs, dangling links, duplicate evidence, empty check declarations, and excessive graph sizes fail input validation. The limits are 128 nodes, 128 scope IDs and required checks per node, and a longest path of 8 nodes. Results include only nodes reachable from requested roots; other declared nodes do not change the requested outcome.

A record must validate and its scope IDs must equal the node scope exactly. Verification freshness is checked by the host records package against the supplied current revision, model, record, target snapshot, verifier, and exactly the node's declared check identities. A mismatch or stale/invalid result makes that node incomplete and records the cause. Materialization `partial-failure` yields failed, and `escalated` yields escalated. Missing evidence yields incomplete.

Rollup status uses this explicit order: failed if the node or any child failed; otherwise escalated if the node or any child escalated; otherwise incomplete if the node or any child is incomplete; otherwise passed. Reports retain causes at each node and link parent causes to exact child, record, and result IDs when available.

This package performs no I/O, persistence, check execution, policy discovery, artifact union, or graph query. Fixed checks run in the Host through its existing verifier; this package only validates and composes explicit records. The records do not authenticate the caller, prove verifier independence or check sufficiency, establish objective correctness, or constitute human acceptance.
