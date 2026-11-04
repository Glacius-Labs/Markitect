---
kind: Interface
---
# Escalations interface

What other modules may use of escalations: its operations and the escalation and policy records. No other module asks anything of it beyond these.

- **Operations:** [[escalations/set-escalation-policy]], [[escalations/escalate-incident]], [[escalations/list-escalations]].
- **Uses:** [[equipment/equipment-interface]], [[safety/safety-interface]], [[lockouts/lockouts-interface]], [[bookings/bookings-interface]], [[workorders/workorders-interface]], [[notices/notices-interface]].
- **Records:** [[escalations/escalation]], [[escalations/escalation-policy]].
- **Rules that bind other modules:** [[escalations/escalation-effects]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_escalated` | The incident has been escalated before. |
| `below_policy` | The incident's severity is lower than the minimum severity of its family's policy. |
| `no_policy` | The family of the incident's machine has no escalation policy. |
