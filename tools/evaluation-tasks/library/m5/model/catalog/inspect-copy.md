---
kind: std/Feature
---
# Inspect copy

Given a known [[catalog/copy]] identifier, return its identifier, title, loan class and current availability under [[catalog/availability]]. Apply [[inputs]] to unknown or invalid identifiers. Inspection changes no state and never creates a copy.

It is reached through the operation `inspectCopy` of the [[callable-interface]].
