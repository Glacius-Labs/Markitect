---
kind: std/Concept
---
# Volunteer time entry

Time a member volunteered, logged by or for the member and decided by staff, generated identifier `vol-N`. `creditCents` is `null` until the entry is approved.

**Fields**, in the formats of [[values]]:

- `entryId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `date`: date
- `minutes`: integer from `1` to `720` (the minutes volunteered in one entry)
- `note`: text
- `status`: `pending`, `approved` or `rejected`
- `creditCents`: cent amount, or `null`

A record is identified by `entryId`.
