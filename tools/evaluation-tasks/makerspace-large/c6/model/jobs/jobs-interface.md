---
kind: Interface
---
# Jobs interface

What other modules may use of jobs: its operations, the job record and the refusal codes below. Other modules ask jobs for nothing else.

- **Operations:** [[jobs/start-job]], [[jobs/complete-job]], [[jobs/record-job-payment]], [[jobs/list-jobs]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]], [[billing/billing-interface]], [[ledger/ledger-interface]], [[training/training-interface]], [[quotes/quotes-interface]].
- **Records:** [[jobs/job]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_started` | A job already exists for the quote. |
| `job_not_completed` | The job is not completed. |
| `maker_not_certified` | The maker has no certification valid on the date of the request time for the family of a machine of the quote. |
| `overpayment` | The payment would bring the job's paid amount above its total. |
| `quote_not_accepted` | The quote is not accepted. |
