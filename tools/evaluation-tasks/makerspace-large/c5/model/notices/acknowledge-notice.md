---
kind: Operation
---
# Acknowledge notice

Sets the notice's `acknowledgedAt` to the request time. A notice belongs to the actor when its member is the actor.

- **Wire name:** `acknowledgeNotice`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `noticeId`: identifier, naming an existing [[notices/notice]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when `acknowledgedAt` is not `null`.
- **Result:**
  - `notice`: a [[notices/notice]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `noticeId`.
