---
kind: Operation
---
# Add machine

Creates the machine.

- **Wire name:** `addMachine`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier
  - `name`: text
  - `familyId`: identifier, naming an existing [[equipment/family]]
  - `rateCentsPerMinute`: integer from `1` to `10000` cents (a machine's rate per minute)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a machine with `machineId` exists.
- **Result:**
  - `machine`: a [[equipment/machine]] record
- **Permissions:** admin all.
- **Audit target:** `machineId`.
