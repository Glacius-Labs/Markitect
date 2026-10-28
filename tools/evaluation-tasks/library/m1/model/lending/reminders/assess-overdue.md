---
kind: std/Feature
---
# Assess overdue

Given a valid explicit assessment date under [[inputs]], calculate overdue notices for the [[project|library]].

Include exactly the currently open [[lending/loan|loans]] whose due dates, established by [[lending/loan-policy]], are strictly earlier than the assessment date. A loan due on that date is not yet overdue. Returned loans are excluded. A suspended [[members/member]] does not remove an outstanding loan from the result.

Return exactly one notice per qualifying loan, identifying its loan, member, copy and due date, or an empty list when none qualify. Notice order is unrestricted. Send no message, schedule no job and change no library state. The same state and date yield the same notice contents. This is an assessment of current open loans, not a historical-state query; a loan already returned in current state remains excluded even when assessing an earlier date.

It is reached through the operation `assessOverdue` of the [[callable-interface]].
