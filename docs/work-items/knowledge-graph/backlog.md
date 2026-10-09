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
    status: in_progress
    scope: Twelve-question regression, normal provider-free user journey, privacy negatives, honest code/source/config/evidence handoff.
    owner: parent and independent Reader
  - id: KG07-M
    status: awaiting_exact_accepted_main
    scope: Merge actual accepted main into this branch, resolve conflicts, regressions and independent review.
    owner: parent
  - id: RDF-FIT
    status: assessment_complete_prototype_deferred
    scope: RDF-fit-decision.md records the separate fit assessment and optional RF01-RF06 plan; no canonical migration, engine installation, purchase or study implied.
```


## Completed source slices

KG04 focused tests and vet passed. An independent Luna High reader found a private foreign Decision subject leak and missing Impact routing to the subject owner. Both fixes and targeted regressions passed the reader recheck. KG03 adds explicit bounded bidirectional witnesses so Statement -> Artifact -> Check can be explained even when typed edges face different directions; original edge orientations stay visible. The existing failed full Verify is preserved. Whole variant gates and accepted Main reconciliation remain pending.


The Host reader has closed all five material findings and the direct ExtraFacts seam residual with targeted rereads. All source owners are frozen for the integrated candidate. KG07 fixed-source full Verify and final binary journey are next; accepted Main remains explicitly unprovided.
