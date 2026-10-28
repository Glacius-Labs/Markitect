---
kind: std/Feature
---
# Return

The operator supplies a [[lending/loan]] identifier and a valid return date on or after its borrowing date under [[inputs]].

Return an open loan by closing exactly that loan and recording the date. The member's open-loan count decreases by one and the [[catalog/copy]] becomes available under [[catalog/availability]]. The [[members/member]] status does not block return, as specified by [[members/eligibility]]; neither overdue state nor the current limit in [[lending/loan-policy]] blocks it. There is no fee or penalty.

For a closed loan, a repeated return with a valid date on or after its borrowing date reports already returned and changes nothing, preserving the original return date. If its copy was lent again, the newer loan stays open and the copy stays unavailable. An unknown loan or a date before the borrowing date is refused without changing state. A copy identifier alone is insufficient to identify the transaction being returned.

It is reached through the operation `return` of the [[callable-interface]].
