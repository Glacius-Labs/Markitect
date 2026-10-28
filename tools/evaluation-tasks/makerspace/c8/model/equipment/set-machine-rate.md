---
kind: Operation
---
# Set machine rate

Sets the rate. The rate is captured at check-in, so the new rate applies to check-ins after the change. Reservations already checked in keep their captured rate, and existing charges do not change.

- **Wire name:** `setMachineRate`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `rateCentsPerMinute`: integer from `1` to `10000` cents per minute (a machine's rate per minute)
- **Result:**
  - `machine`: a [[equipment/machine]] record
- **Permissions:** admin all.
- **Audit target:** `machineId`.
