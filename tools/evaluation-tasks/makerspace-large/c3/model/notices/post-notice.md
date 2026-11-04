---
kind: Operation
---
# Post notice

Creates a notice for the member with the next notice identifier: `kind` `manual`, `subject` as supplied, `ref` `null`, `at` the request time and `acknowledgedAt` `null`. Standing is not checked.

- **Wire name:** `postNotice`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `subject`: text
- **Result:**
  - `notice`: a [[notices/notice]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `noticeId`.
