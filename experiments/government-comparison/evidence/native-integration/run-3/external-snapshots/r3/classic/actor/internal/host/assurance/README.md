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

## Bounded Host execution

`Execute(ctx, RunInput, Runner)` adds an explicit callback boundary for Host-owned work. It validates the complete declared graph before the first callback, including unreachable nodes, then calls the reachable nodes in deterministic child-first order. A node shared by several parents is scheduled once. The Host must provide a current `records.Freshness` binding for each reachable node; the runner receives a defensive copy and cannot replace the binding used for evaluation.

Each callback receives its own bounded node and only its direct children. A child entry contains the composed child result and validated evidence when available. Verification reasons and cause messages are removed before child information reaches a parent callback. The callback is responsible for invoking the adopting Host's scoped executor and verifier; this package does not invent or execute checks. Parent callbacks still run after failed or incomplete children, so they can perform their own checks or return `RunSkipped` when only a read-only diagnosis is appropriate. `Evaluate` composes the returned evidence afterward, so a passing parent check cannot erase a failed or incomplete child.

Runner errors, explicit skips, invalid callback output, and cancellation remain distinct invocation statuses. A skipped, errored, or invalid invocation contributes no fabricated verification evidence and therefore evaluates as incomplete unless an existing child failure gives the composed node the stronger failed outcome. A valid verification result with failed checks remains `checker-failed`; this distinction keeps missing work separate from a failed check. Retries are explicit additional attempts, limited to two, and apply only to callback errors. Cancellation stops later callbacks and returns a partial report plus the context error.

The report binds the canonical graph, Host freshness bindings, and retry limit in `InputDigest`; invocations are ordered by the actual schedule. `WallTime` is reported separately and does not enter the digest. Callback results are checked for the scheduled node ID, exact node scope, record/result binding, and exact declared check identities before being composed. Freshness is still decided by `records.ValidateVerificationFreshness` through `Evaluate`; this scheduler does not add another evidence policy.
