---
kind: Operation
---
# Take deposit

Takes a deposit from the member. The deposit is created with the next deposit identifier, status `held`, `takenAt` the request time and `closedAt` `null`. A deposit [[billing/charge]] is created for the member: `kind` `deposit`, `ref` the deposit identifier, `quantity` 1, `unitPriceCents` the deposit's `amountCents`, `discountPercent` 0 and `at` the request time. The charge is invoiced like any other; the deposit is not paid by this request.

Rules of other modules that apply here: [[billing/charges-and-invoices]] and [[members/membership]].

- **Wire name:** `takeDeposit`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `purpose`: text
  - `amountCents`: integer from `1` to `100000000` cents (a deposit amount)
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
- **Result:**
  - `deposit`: a [[deposits/deposit]] record
  - `charge`: a [[billing/charge]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `depositId`.
