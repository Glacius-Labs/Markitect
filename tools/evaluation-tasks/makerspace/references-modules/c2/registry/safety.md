---
paths:
  - realization/safety/**
---
# Registry: safety

| File | Realizes | Why |
|---|---|---|
| `realization/safety/operations.go` | [[safety/report-incident]] [[safety/resolve-incident]] [[safety/list-incidents]] [[safety/incidents]] | Safety operations: reporting, resolving and listing incidents |
| `realization/safety/safety.go` | [[safety/safety-interface]] [[safety/incident]] [[safety/incidents]] | Safety incidents, and what other modules may ask of them |
| `realization/safety/views.go` | [[safety/incident]] | JSON view of the incident record |
