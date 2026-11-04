# Change request: A fee for no-shows

Members book machines, do not come, and the slot is lost to everyone else. I want them to pay half the price of the time they blocked. A missed reservation causes a charge from now on, but only through a run that staff start by hand, and never when the machine could not have been used anyway.

The run takes every reservation whose reported status is `missed` at the request time, in ascending reservation number (the number `N` of `res-N`, compared as a number). For each one that no earlier run has charged and whose machine was not locked out during the slot, it creates a charge for the reservation's member: `kind` `noShow`, `ref` the reservation identifier, `quantity` the reserved minutes (`end − start`), `unitPriceCents` the machine's current rate, `discountPercent` 50, `at` the request time, and the amount by money.

- The machine **was locked out during the slot** when a lockout of the machine, of any source, overlaps `[start, end)`, taking the lockout as the interval `[appliedAt, releasedAt)`. A lockout that is not released overlaps when its `appliedAt` is before `end`. Such a reservation is not charged, by this run or any later one.
- The member's standing is not checked.
- The reservation does not change: it stays `missed` and its `amountCents` stays `null`.

Example: at a rate of 45 cents per minute, a missed reservation of 60 minutes gives `(60 × 45 × 50 + 50) div 100 = 1350` cents, and one of 15 minutes gives `(15 × 45 × 50 + 50) div 100 = 338` cents.

- **Wire name:** `chargeNoShows`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `charges`: a list of charge records, the charges created, in ascending reservation number; empty when nothing is due
- **Permissions:** staff all, admin all.
- **Audit target:** the fixed string `noShows`.

`noShow` is a new kind of charge. It comes last in the kind order, after `event`. Its invoice lines are credited to a new ledger account, `revenueNoShows`, which means "No-show lines invoiced."
