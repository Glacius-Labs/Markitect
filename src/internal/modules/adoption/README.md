# Adoption Module

This capability owns pure selective-capture and Copy Me review contracts. Its private `capture` package validates owner-supplied scope, selected snapshots, coverage and handoff identities; `review` consumes that immutable handoff plus supplied evidence/candidate/decision bytes. This private dependency stays within one Module. Neither package acquires Git evidence or writes a workspace.

Host owns `prepare`/storage and `copy-me` runtime IO. It supplies only approved selected bytes and exact records. Capture cannot infer policy; review cannot expand evidence, authenticate owners/reviewers or adopt canonical meaning. Changed evidence/candidate/handoff bytes stale decisions. Explicit acceptance remains an unauthenticated supplied review statement and does not mutate a Project.

See the [handoff contract](../../../docs/design/selective-adoption-handoff.md), [review contract](../../../docs/design/copy-me-evidence-review.md), [usage](../../../docs/usage.md) and [Module laws](../../../docs/development/modules.md). Module tests own closed-contract/digest/coverage/freshness cases; Host and examples test selective acquisition, storage refusal and executable lifecycle.
