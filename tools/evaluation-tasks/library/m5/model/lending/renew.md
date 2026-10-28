---
kind: std/Feature
---
# Renew

The operator supplies a [[lending/loan]] identifier and a renewal date under [[inputs]].

Admit the renewal only when the loan is open, its [[members/member]] is registered under [[members/eligibility]], the renewal date is on or after the borrowing date and on or before the current due date, the loan has been renewed fewer times than the renewal limit of [[lending/loan-policy]], and the new due date falls on or before 9999-12-31.

Success moves the due date forward by the loan period of the loan's [[catalog/copy]] under [[lending/loan-policy]], counted from the current due date, adds one to the loan's renewal count and returns the loan. The member's open-loan count, the copy's availability and the borrowing date are unchanged.

An unknown loan, a closed loan, a renewal date before the borrowing date or after the current due date, an exhausted renewal count, a new due date after 9999-12-31 or invalid inputs cause refusal with no observable state change.

It is reached through the operation `renew` of the [[callable-interface]].
