---
kind: std/Feature
---
# Inspect loans

Support inspection of a known [[lending/loan]] by its identifier, returning its loan, member and copy identifiers, borrowing and due dates, open or closed state, and return date when closed.

Also support inspecting a known [[members/member]] for its currently open loans and count, following [[lending/loan-policy]]. An empty list and zero count are valid.

Unknown or invalid requested identifiers are refused under [[inputs]]. Both queries are read-only and describe current state; returned loan order is unrestricted.

They are reached through the operations `inspectLoan` and `inspectOpenLoans` of the [[callable-interface]].
