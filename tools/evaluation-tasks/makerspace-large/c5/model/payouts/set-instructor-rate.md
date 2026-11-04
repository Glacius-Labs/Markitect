---
kind: Operation
---
# Set instructor rate

Sets the user's rate and replaces an earlier one. Payouts that are already generated do not change. The rate stays when the user's qualifications are withdrawn later.

- **Wire name:** `setInstructorRate`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `userId`: identifier, naming an existing [[members/user]]
  - `centsPerHour`: integer from `1` to `1000000` cents per hour (an instructor's pay for one hour of teaching)
- **Refusals** at check 7, in this order:
  1. `not_instructor` when the user holds no qualification for any family ([[instructors/instructors-interface]]).
- **Result:**
  - `rate`: a [[payouts/instructor-rate]] record
- **Permissions:** admin all.
- **Audit target:** `userId`.
