---
kind: Kind
fields:
parts:
---
# Condition

A Condition Definition states one condition under which an [[std/Operation]] is allowed, named for what must hold, and nothing else. It lies under `conditions/` of the one Operation that checks it, or, when several check it, of the [[std/Concept]] or Package it is about.

The Operation that checks it gives its place in the checking order and the code that answers it there.

Verified when a call to each such Operation that breaks only this Condition is refused with that Operation's code.
