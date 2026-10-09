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
    status: in_progress
    scope: First-class Decisions and explicit Statement identity changes; closed YAML schema and conservative impact.
    owner: model subagent
    files: internal/modules/projectmodel
  - id: KG05-S
    status: in_progress
    scope: Authoritative ProjectModel mapping and exact manager node-field-edge privacy; read-only queries.
    owner: scope service subagent
    files: internal/host/projectgraph
  - id: KG05-E
    status: in_progress
    scope: Explicit selected existing exploration/run/briefing/evidence adapters; current stale historical partial unknown distinctions.
    owner: evidence subagent
    files: internal/host/knowledgeevidence
  - id: KG06
    status: ready_after_service_seam
    scope: Shared projectapp operations plus CLI and MCP graph tools; canonical guidance and executable examples.
    owner: parent and transport subagent
    files: [internal/host/projectapp, internal/host/projectcli, internal/host/knowledgeprotocol, docs]
  - id: KG07
    status: ready_after_features
    scope: Twelve-question regression, normal provider-free user journey, privacy negatives, honest code/source/config/evidence handoff.
    owner: parent and independent Reader
  - id: KG07-M
    status: awaiting_exact_accepted_main
    scope: Merge actual accepted main into this branch, resolve conflicts, regressions and independent review.
    owner: parent
  - id: RDF-FIT
    status: deferred
    scope: Separate justified fit evaluation; no canonical migration, installation, purchase or study implied.
```
