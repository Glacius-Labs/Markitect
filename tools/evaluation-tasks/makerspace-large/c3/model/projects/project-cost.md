---
kind: std/Rule
---
# Project cost

The cost of a [[projects/project]] is read from the records attached to it, at the request time of the read.

- `usageCents` is the sum of `amountCents` over the attached reservations whose status is `completed`. Reservations in any other status add nothing.
- `materialCents` is the sum of `amountCents` over the attached purchases. The amount of a purchase is the amount after its discount.
- `totalCents` is `usageCents` plus `materialCents`.
- `byMember` has one [[projects/cost-share]] for each member who has a share, in ascending `memberId` order. A member has a share when at least one attached completed reservation or at least one attached purchase belongs to the member. The member's `usageCents` is the sum over the member's attached completed reservations and the member's `materialCents` is the sum over the member's attached purchases. Summed over `byMember`, the shares equal `usageCents` and `materialCents`.

Example: `ada` has an attached completed reservation of 1499 cents and an attached purchase of 600 cents, and `bo`, a collaborator, has an attached purchase of 250 cents. `usageCents` is 1499, `materialCents` is 850 and `totalCents` is 2349. `byMember` is `ada` with 1499 and 600, then `bo` with 0 and 250.
