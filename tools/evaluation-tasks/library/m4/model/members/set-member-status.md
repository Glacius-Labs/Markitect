---
kind: std/Feature
---
# Set member status

The operator supplies a known [[members/member]] identifier and the requested status, active or suspended, under [[inputs]]. Success sets that status and preserves all other member data and all loans as required by [[members/eligibility]]. Setting the current status again succeeds without another change. An unknown member or a status other than active or suspended is refused without changing state. This operation does not register a missing member.

It is reached through the operation `setMemberStatus` of the [[callable-interface]].
