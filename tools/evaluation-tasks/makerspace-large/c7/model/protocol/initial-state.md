---
kind: std/Rule
internal: true
---
# Initial state

A new process contains exactly one [[members/user]]: identifier `admin`, name `Administrator`, role `admin`. There are no members, families, machines, courses, materials, reservations, invoices, incidents or audit entries, and no record of any other module. Every family is unsupervised, and the volunteer rate is 0.

The program generates the identifiers of these records:

| Record | Format |
| --- | --- |
| Reservation | `res-1`, `res-2`, … |
| Maintenance window | `mnt-1`, `mnt-2`, … |
| Purchase | `pur-1`, `pur-2`, … |
| Invoice | `inv-1`, `inv-2`, … |
| Payment | `pay-1`, `pay-2`, … |
| Incident | `inc-1`, `inc-2`, … |
| Journal entry | `jrn-1`, `jrn-2`, … |
| Notice | `ntc-1`, `ntc-2`, … |
| Shift | `shf-1`, `shf-2`, … |
| Room hold | `hld-1`, `hld-2`, … |
| Deposit | `dep-1`, `dep-2`, … |
| Pack | `pak-1`, `pak-2`, … |
| Scholarship award | `aws-1`, `aws-2`, … |
| Refund | `rfd-1`, `rfd-2`, … |
| Dunning case | `dun-1`, `dun-2`, … |
| Locker rental | `rnt-1`, `rnt-2`, … |
| Guest pass | `gst-1`, `gst-2`, … |
| Volunteer time entry | `vol-1`, `vol-2`, … |
| Badge | `bdg-1`, `bdg-2`, … |
| Door event | `dor-1`, `dor-2`, … |
| Lockout | `lck-1`, `lck-2`, … |
| Class | `cls-1`, `cls-2`, … |
| Payout | `pyt-1`, `pyt-2`, … |
| Event ticket | `tkt-1`, `tkt-2`, … |
| Purchase order | `po-1`, `po-2`, … |
| Stocktake | `stk-1`, `stk-2`, … |
| Work order | `wko-1`, `wko-2`, … |
| Escalation | `esc-1`, `esc-2`, … |
| Quote | `quo-1`, `quo-2`, … |
| Job | `job-1`, `job-2`, … |
| Tool loan | `lon-1`, `lon-2`, … |

Each kind has its own counter. The first record of a kind gets number 1 and each further record of that kind gets the next number, in decimal without leading zeros. Numbers are never reused. Refused and invalid requests consume no number.
