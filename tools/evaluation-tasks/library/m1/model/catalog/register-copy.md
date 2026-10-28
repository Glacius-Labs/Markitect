---
kind: std/Feature
---
# Register copy

The operator supplies a new identifier and title under [[inputs]] to register one [[catalog/copy]] in the [[project|library]]. It initially has no loans and is available according to [[catalog/availability]]. Reusing an existing copy identifier is refused without replacing that record or changing any state. Matching an existing title alone is not a duplicate registration.

It is reached through the operation `registerCopy` of the [[callable-interface]].
