---
kind: Operation
---
# Sell pack

Sells the product to the member. A pack is created with the next pack identifier, `minutesLeft` the product's `minutes`, `issuedOn` the date of the request time and `expiresOn` the date `validDays` days after `issuedOn` ([[time]]). A pack [[billing/charge]] is created for the member: `kind` `pack`, `ref` the pack identifier, `quantity` 1, `unitPriceCents` the product's `priceCents`, `discountPercent` 0 and `at` the request time. The pack can be used before its charge is paid.

Rules of other modules that apply here: [[billing/charges-and-invoices]] and [[members/membership]].

- **Wire name:** `sellPack`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `productId`: identifier, naming an existing [[packs/pack-product]]
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
- **Result:**
  - `pack`: a [[packs/pack]] record
  - `charge`: a [[billing/charge]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `packId`.
