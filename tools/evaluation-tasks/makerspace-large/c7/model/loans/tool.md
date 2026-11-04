---
kind: std/Concept
---
# Tool

A tool that can be lent out, such as a portable power tool. Every tool belongs to a [[equipment/family|tool family]], and the rules for using a family are the same for its tools as for its machines.

**Fields**, in the formats of [[values]]:

- `toolId`: identifier
- `name`: text
- `familyId`: identifier, naming an existing [[equipment/family]]
- `lateFeeCentsPerDay`: integer from `1` to `100000` cents (the fee for one day overdue)
- `onLoan`: flag, `true` while the tool has an active [[loans/loan]]

A record is identified by `toolId`.
