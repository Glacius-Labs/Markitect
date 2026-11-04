---
kind: Interface
---
# Loans interface

What other modules may use of loans: its operations, the tool and loan records and the refusal codes below. Other modules also ask whether a member has a loan that is overdue on a date. The rule is [[loans/overdue-loans]].

- **Operations:** [[loans/add-tool]], [[loans/list-tools]], [[loans/borrow-tool]], [[loans/return-tool]], [[loans/renew-loan]], [[loans/list-member-loans]], [[loans/charge-overdue-loans]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]], [[training/training-interface]], [[guardians/guardians-interface]], [[waivers/waivers-interface]], [[billing/billing-interface]], [[notices/notices-interface]].
- **Records:** [[loans/tool]], [[loans/loan]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `loan_limit` | The member already has 3 active loans. |
| `loan_overdue` | The member has an overdue tool loan, or the loan to renew is overdue. |
| `renewal_limit` | The loan has been renewed twice. |
| `tool_on_loan` | The tool has an active loan. |

[[loans/borrow-tool]] and [[loans/renew-loan]] also return codes that other modules add: `member_suspended` (members interface), `guardian_consent_missing` (guardians interface), `not_certified` (training interface) and `waiver_missing` (waivers interface), each with the meaning that interface states unless the operation states another.
