---
kind: Operation
---
# Set guest allowance

Sets the allowance of the tier, replacing an earlier one. Passes that exist are not changed.

- **Wire name:** `setGuestAllowance`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `tier`: a [[members/tier]]: `basic`, `plus` or `pro`
  - `perMonth`: integer from `0` to `100` (the guest passes of one month that are free)
  - `feeCents`: integer from `0` to `1000000` cents (the fee of a guest pass beyond the free ones)
- **Result:**
  - `allowance`: a [[guests/guest-allowance]] record
- **Permissions:** admin all.
- **Audit target:** `tier`.
