# Durable implementation backlog

```yaml
version: 1
workstream: knowledge-graph
branch: codex/knowledge-graph
starting_reference: 1495e1be7b0046711531fa42c8407fba67b8e814
continuation_base: 0a56d1cea65b53aec0152f2de30f8d1fa28600fd
authority: direct user continuation relayed by Overseer on 2026-10-09
canonical_format: YAML
integration_policy: Independent variant; merge exact accepted main when Overseer provides it; never auto merge KG into main.
time_limits_minutes: {focused: 10, packages: 30, suite: 60, outer: 90}
items:
  - {id: KG01, status: complete, output: KG01-question-matrix.md}
  - {id: KG02, status: complete, scope: pure index}
  - {id: KG03, status: complete, scope: pure bounded navigation}
  - id: KG04
    status: complete_reviewed
    scope: First-class Decisions and explicit Statement identity changes; closed YAML schema and conservative impact.
    owner: model subagent
    files: internal/modules/projectmodel
  - id: KG05-S
    status: complete_reviewed
    scope: Authoritative ProjectModel mapping and exact manager node-field-edge privacy; read-only queries.
    owner: scope service subagent
    files: internal/host/projectgraph
  - id: KG05-E
    status: complete_reviewed
    scope: Explicit selected existing exploration/run/briefing/evidence adapters; current stale historical partial unknown distinctions.
    owner: evidence subagent
    files: internal/host/knowledgeevidence
  - id: KG06
    status: complete_reviewed
    scope: Shared projectapp operations plus CLI and MCP graph tools; canonical guidance and executable examples.
    owner: parent and transport subagent
    files: [internal/host/projectapp, internal/host/projectcli, internal/host/knowledgeprotocol, docs]
  - id: KG07
    status: implemented_targeted_validated_reviewed
    scope: Twelve-question regression, normal provider-free user journey, privacy negatives, honest code/source/config/evidence handoff.
    owner: parent and independent Reader
  - id: KG07-M
    status: awaiting_exact_accepted_main
    scope: Merge actual accepted main into this branch, resolve conflicts, regressions and independent review.
    owner: parent
  - id: KG07-E
    status: complete_targeted_validated_reviewed
    scope: Fresh actual stored Plan/Run/candidate/check/review/Verification/guarded Apply chain to real CLI and MCP processes; privacy, stale/partial/unknown and read-only negatives.
    source: f9b8114d91748d52a2f3783e667a461211c75461
    output: fresh-evidence-chain-20261010.md
    owner: parent and independent Luna High Reader
  - id: RDF-FIT
    status: assessment_complete_prototype_deferred
    scope: RDF-fit-decision.md records the separate fit assessment and optional RF01-RF06 plan; no canonical migration, engine installation, purchase or study implied.
```


## Completed source slices

KG04 focused tests and vet passed. An independent Luna High reader found a private foreign Decision subject leak and missing Impact routing to the subject owner. Both fixes and targeted regressions passed the reader recheck. KG03 adds explicit bounded bidirectional witnesses so Statement -> Artifact -> Check can be explained even when typed edges face different directions; original edge orientations stay visible. All failed full Verifies remain preserved. There is no full-suite PASS; accepted Main reconciliation and the combined full gate remain pending.


The Host reader has closed all five material findings and the direct ExtraFacts seam residual with targeted rereads. All implementation owners are complete. Both source-bound actual CLI/MCP journeys passed all six queries without provider calls. The d360ec01 full Verify failed two CLI fixtures; their repairs passed in the ebe51470 full suite. That replay failed a shared four-minute positive Delivery fixture budget; the localized fifteen-minute repair passed the exact targeted test in 217.099 s and independent review. KG07 therefore records implementation/targeted closure, not an overall full Verify PASS. Per Overseer's explicit direction, the next full replay is the combined accepted Main-to-KG candidate. KG07-M is still awaiting the exact accepted Main SHA; the RDF assessment is complete and its optional prototype is deferred.
