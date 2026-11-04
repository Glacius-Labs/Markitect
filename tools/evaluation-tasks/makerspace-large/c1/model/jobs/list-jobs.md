---
kind: Operation
---
# List jobs

Returns all jobs of the customer, in creation order, in every status.

- **Wire name:** `listJobs`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `customerId`: identifier, naming an existing [[quotes/customer]]
- **Result:**
  - `customerId`: identifier
  - `jobs`: a list of [[jobs/job]] records
- **Permissions:** staff all, admin all.
