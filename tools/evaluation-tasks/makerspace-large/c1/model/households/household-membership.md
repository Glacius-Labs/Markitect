---
kind: std/Rule
internal: true
---
# Household membership

A [[households/household]] has 1 to 6 members. A member belongs to at most one household. The payer is always one of its members: the household is created with the payer as its only member, and the payer cannot be removed. Households are never removed and their payer and name never change. A member removed from a household belongs to none and may be added to any household, including the same one.
