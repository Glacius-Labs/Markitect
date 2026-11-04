---
kind: Operation
---
# Set escalation policy

Sets the policy of the family: a family without a policy gets one, and a family with a policy has it replaced. Setting the current value succeeds without a change. Escalations made earlier are not affected.

- **Wire name:** `setEscalationPolicy`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `familyId`: identifier, naming an existing [[equipment/family]]
  - `minSeverity`: one of `low`, `medium` or `high`
- **Result:**
  - `policy`: an [[escalations/escalation-policy]] record
- **Permissions:** admin all.
- **Audit target:** `familyId`.
