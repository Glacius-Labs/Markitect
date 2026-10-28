---
kind: std/Feature
---
# Register member

The operator supplies a new identifier and name under [[inputs]] to register a [[members/member]] in the [[project|library]]. Success creates exactly one member with those values. Reusing an existing member identifier is refused without replacing the existing record or changing other state.

It is reached through the operation `registerMember` of the [[callable-interface]].
