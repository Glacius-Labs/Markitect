---
kind: Operation
---
# Record job payment

The job's `paidCents` increases by `amountCents`. The customer pays the job directly; no invoice and no credit is involved. A payment that brings `paidCents` exactly to `totalCents` is accepted, and a job that is fully paid takes no further payment. Ledger postings are made.

Rules of other modules that apply here: [[ledger/postings]].

- **Wire name:** `recordJobPayment`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `jobId`: identifier, naming an existing [[jobs/job]]
  - `amountCents`: integer from `1` to `100000000` cents (an amount paid in one request)
- **Refusals** at check 7, in this order:
  1. `job_not_completed` when the job's status is not `completed`.
  2. `overpayment` when `paidCents` plus `amountCents` is above `totalCents`.
- **Result:**
  - `job`: a [[jobs/job]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `jobId`.
