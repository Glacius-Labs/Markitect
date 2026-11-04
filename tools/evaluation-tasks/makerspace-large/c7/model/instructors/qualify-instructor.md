---
kind: Operation
---
# Qualify instructor

Creates the qualification of the user for the family ([[instructors/qualifications]]).

- **Wire name:** `qualifyInstructor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `userId`: identifier, naming an existing [[members/user]]
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Refusals** at check 7, in this order:
  1. `not_eligible` when the user is an admin, or is a member who holds no certification for the family valid on the date of the request time ([[training/training-interface]]).
  2. `already_exists` when the user holds a qualification for the family.
- **Result:**
  - `qualification`: a [[instructors/qualification]] record
- **Permissions:** admin all.
- **Audit target:** `userId/familyId`.
