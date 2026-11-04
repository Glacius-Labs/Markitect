---
kind: Operation
---
# Record inspection

Records the outcome of an inspection of the machine.

- `passed`: `lastPassedOn` becomes the date of the request time, `nextDueOn` is computed again, `failed` becomes `false`, and every active lockout of source `inspection` on the machine is released ([[lockouts/lockouts-interface]]). Lockouts of other sources stay.
- `failed`: `failed` becomes `true`; `lastPassedOn` and `nextDueOn` stay. When the machine has no active lockout of source `inspection`, one is applied with `reason` the supplied `note`, `appliedBy` the actor and `appliedAt` the request time; otherwise none is applied.

The `note` is not stored on the plan.

Rules of other modules that apply here: [[lockouts/lockout-gate]].

- **Wire name:** `recordInspection`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `outcome`: `passed` or `failed`
  - `note`: text
- **Refusals** at check 7, in this order:
  1. `no_plan` when the machine has no inspection plan.
- **Result:**
  - `plan`: an [[inspections/inspection-plan]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `machineId`.
