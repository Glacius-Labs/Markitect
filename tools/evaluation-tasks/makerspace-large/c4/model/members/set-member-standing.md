---
kind: Operation
---
# Set member standing

Suspends, pauses or reinstates the member: sets the standing. Pausing cancels nothing. Setting the current standing succeeds without a change.

- **Wire name:** `setMemberStanding`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `standing`: `good`, `suspended` or `paused`
- **Result:**
  - `member`: a [[members/member]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId`.
