---
kind: std/Concept
---
# Award

A scholarship of one member from one [[scholarships/fund]], generated identifier `aws-N`. It gives a discount of `percent` on course fees while it is active ([[scholarships/course-discount]]). A member has at most one active award.

**Fields**, in the formats of [[values]]:

- `awardId`: identifier
- `fundId`: identifier, naming an existing [[scholarships/fund]]
- `memberId`: identifier, naming an existing [[members/member]]
- `percent`: integer from `1` to `100` (a discount in percent)
- `status`: `active` or `revoked`

A record is identified by `awardId`.
