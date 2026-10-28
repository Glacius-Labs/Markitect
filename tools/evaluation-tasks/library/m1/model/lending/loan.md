---
kind: std/Concept
---
# Loan

A loan associates one registered [[members/member]] with one registered [[catalog/copy]] in the [[project|library]]. A successful [[lending/borrow]] creates its own unique stable loan identifier, member identifier, copy identifier, borrowing date and due date under [[lending/loan-policy]] and [[inputs]]. It starts open. [[lending/return]] records a return date and closes it. Closed loans stay inspectable, preserve their dates and never have their identifiers reused. Loan records describe current state and retained facts, not a service for reconstructing the entire library at a historical date.
