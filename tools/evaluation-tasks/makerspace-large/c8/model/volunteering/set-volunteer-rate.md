---
kind: Operation
---
# Set volunteer rate

Sets the rate that later approvals pay. Entries that are decided already keep their credit. Setting the current value succeeds without a change.

- **Wire name:** `setVolunteerRate`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `centsPerHour`: integer from `0` to `100000` cents per hour (the credit for one hour of volunteering)
- **Result:**
  - `centsPerHour`: cent amount
- **Permissions:** admin all.
- **Audit target:** the fixed string `rate`.
