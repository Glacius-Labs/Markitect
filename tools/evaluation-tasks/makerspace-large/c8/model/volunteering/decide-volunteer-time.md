---
kind: Operation
---
# Decide volunteer time

Sets the entry's status to the decision. A rejection leaves `creditCents` `null`. An approval sets `creditCents` and credits the member as [[volunteering/volunteer-credit]] states. The `entry` result is the entry after the change.

Rules of other modules that apply here: [[ledger/postings]] and [[money]].

- **Wire name:** `decideVolunteerTime`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `entryId`: identifier, naming an existing [[volunteering/volunteer-entry]]
  - `decision`: `approved` or `rejected`
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the entry's status is not `pending`.
  2. `volunteer_cap_reached` when the decision is `approved` and the cap of [[volunteering/volunteer-credit]] would be exceeded.
- **Result:**
  - `entry`: a [[volunteering/volunteer-entry]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `entryId`.
