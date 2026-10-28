---
kind: std/Rule
internal: true
---
# Initial state

A new process contains exactly one [[members/user]]: identifier `admin`, name `Administrator`, role `admin`. There are no members, families, machines, courses, materials, reservations, invoices, incidents or audit entries.

The program generates the identifiers of these records:

| Record | Format |
| --- | --- |
| Reservation | `res-1`, `res-2`, … |
| Maintenance window | `mnt-1`, `mnt-2`, … |
| Purchase | `pur-1`, `pur-2`, … |
| Invoice | `inv-1`, `inv-2`, … |
| Payment | `pay-1`, `pay-2`, … |
| Incident | `inc-1`, `inc-2`, … |

Each kind has its own counter. The first record of a kind gets number 1 and each further record of that kind gets the next number, in decimal without leading zeros. Numbers are never reused. Refused and invalid requests consume no number.
