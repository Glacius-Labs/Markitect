---
kind: std/Rule
---
# Maker credit

When a [[jobs/job]] is completed, its maker receives 30 percent of the job's `totalCents` as credit: `makerCreditCents` is `(totalCents × 30 + 50) div 100`, rounded half up once as [[money]] states for amounts. The maker's credit balance increases by `makerCreditCents` ([[billing/billing-interface]]). The credit applies to the maker's next invoices, like any other credit.

The credit is posted to the ledger as [[ledger/postings]] states. A `makerCreditCents` of 0 changes no balance and posts nothing.

Example: a job of 3602 cents gives `(3602 × 30 + 50) div 100 = 1081` cents. A job of 1 cent gives 0.
