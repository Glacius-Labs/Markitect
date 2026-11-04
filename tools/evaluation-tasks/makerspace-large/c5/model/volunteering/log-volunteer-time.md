---
kind: Operation
---
# Log volunteer time

Creates the entry with the next entry identifier, status `pending` and `creditCents` `null`. Standing is not checked.

- **Wire name:** `logVolunteerTime`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `date`: date
  - `minutes`: integer from `1` to `720` (the minutes volunteered in one entry)
  - `note`: text
- **Refusals** at check 7, in this order:
  1. `date_in_future` when `date` is after the date of the request time; today is allowed.
- **Result:**
  - `entry`: a [[volunteering/volunteer-entry]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `entryId`.
