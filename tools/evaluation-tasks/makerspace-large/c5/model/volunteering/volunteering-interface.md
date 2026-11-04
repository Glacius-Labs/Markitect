---
kind: Interface
---
# Volunteering interface

What other modules may use of volunteering: its operations, the volunteer time entry record and the refusal codes below. No other module asks anything of it beyond that. Volunteering in turn asks billing to add credit to a member, which posts the entry to the ledger, and staffing whether a user has a scheduled shift that contains a point in time.

- **Operations:** [[volunteering/set-volunteer-rate]], [[volunteering/log-volunteer-time]], [[volunteering/decide-volunteer-time]], [[volunteering/list-volunteer-time]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]], [[ledger/ledger-interface]], [[staffing/staffing-interface]].
- **Records:** [[volunteering/volunteer-entry]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `approver_not_on_shift` | The actor is staff and has no scheduled shift at the request time. |
| `date_in_future` | The date is after the date of the request time. |
| `volunteer_cap_reached` | Approving the entry would take the member's approved minutes of the month above 1200. |
