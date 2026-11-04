---
kind: Operation
---
# Change tier

Sets the member's tier. Setting the current tier succeeds without a change. The new tier applies to membership fees of invoices generated afterwards and to usage charges created by later check-outs.

- **Wire name:** `changeTier`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `tier`: a [[members/tier]]: `basic`, `plus` or `pro`
- **Result:**
  - `member`: a [[members/member]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId`.
