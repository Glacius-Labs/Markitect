---
kind: Operation
---
# Complete job

The job's status becomes `completed`. `makerCreditCents` is set and added to the maker's credit balance, as [[jobs/maker-credit]] states. The maker's standing is not judged.

Rules of other modules that apply here: [[jobs/maker-credit]], [[ledger/postings]] and [[money]].

- **Wire name:** `completeJob`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `jobId`: identifier, naming an existing [[jobs/job]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the job's status is not `started`.
- **Result:**
  - `job`: a [[jobs/job]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `jobId`.
