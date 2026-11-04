---
kind: Operation
---
# Add tool

Creates the tool with `onLoan` `false`.

- **Wire name:** `addTool`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `toolId`: identifier
  - `name`: text
  - `familyId`: identifier, naming an existing [[equipment/family]]
  - `lateFeeCentsPerDay`: integer from `1` to `100000` cents (the fee for one day overdue)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a tool with `toolId` exists.
- **Result:**
  - `tool`: a [[loans/tool]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `toolId`.
