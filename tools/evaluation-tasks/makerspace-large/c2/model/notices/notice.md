---
kind: std/Concept
---
# Notice

A message recorded for a member, generated identifier `ntc-N`. The program only records notices; it sends nothing. `ref` and `acknowledgedAt` are `null` until set.

**Fields**, in the formats of [[values]]:

- `noticeId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `kind`: a [[notices/notice-kind]]: `manual`, `dunning`, `suspension`, `waitlist` or `cancellation`
- `subject`: text
- `ref`: text, or `null` (what the notice is about: an invoice, a course or a reservation identifier)
- `at`: timestamp
- `acknowledgedAt`: timestamp, or `null`

A record is identified by `noticeId`.
