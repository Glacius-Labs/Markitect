---
kind: std/Concept
---
# Escalation policy

The least severe incident that is escalated for the machines of one tool family. Severities are ordered `low` < `medium` < `high`. A family has at most one policy; families without one cannot have an incident escalated.

**Fields**, in the formats of [[values]]:

- `familyId`: identifier, naming an existing [[equipment/family]]
- `minSeverity`: one of `low`, `medium` or `high`

A record is identified by `familyId`.
