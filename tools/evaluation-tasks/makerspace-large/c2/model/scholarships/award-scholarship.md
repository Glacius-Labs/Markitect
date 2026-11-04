---
kind: Operation
---
# Award scholarship

Awards the member a scholarship from the fund. The award is created with the next award identifier and status `active`. The member's standing is not checked and the fund's amounts do not change.

- **Wire name:** `awardScholarship`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `fundId`: identifier, naming an existing [[scholarships/fund]]
  - `memberId`: identifier, naming an existing [[members/member]]
  - `percent`: integer from `1` to `100` (a discount in percent)
- **Refusals** at check 7, in this order:
  1. `already_awarded` when the member has an award with status `active`.
- **Result:**
  - `award`: a [[scholarships/award]] record
- **Permissions:** admin all.
- **Audit target:** the new `awardId`.
