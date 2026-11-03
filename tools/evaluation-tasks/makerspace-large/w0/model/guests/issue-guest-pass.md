---
kind: Operation
---
# Issue guest pass

Creates the guest pass with the next guest pass identifier, the supplied guest name and date and status `issued`. `charged` is set as [[guests/guest-passes]] states. When the pass is charged, a [[billing/charge]] is created for the member: `kind` `guest`, `ref` the new guest pass identifier, `quantity` 1, `unitPriceCents` the allowance's `feeCents`, `discountPercent` 0, `at` the request time.

Rules of other modules that apply here: [[billing/charges-and-invoices]] and [[members/membership]].

- **Wire name:** `issueGuestPass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `guestName`: text
  - `date`: date
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `date_in_past` when `date` is before the date of the request time; today is allowed.
  3. `guests_not_allowed` when the member's tier has no allowance.
- **Result:**
  - `guestPass`: a [[guests/guest-pass]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `guestPassId`.
