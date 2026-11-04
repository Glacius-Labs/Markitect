---
kind: Operation
---
# List volunteer time

Returns all entries of the member in every status, in creation order.

- **Wire name:** `listVolunteerTime`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `entries`: a list of [[volunteering/volunteer-entry]] records
- **Permissions:** member own, staff all, admin all.
