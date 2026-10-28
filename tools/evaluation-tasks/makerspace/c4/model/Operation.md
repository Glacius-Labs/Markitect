---
kind: Kind
---
# Operation

An Operation Definition describes one operation of the program's request protocol: what it does, how a caller reaches it and what the caller gets back. Its body states the operation's effect; a list below it states:

- **Wire name:** the exact `op` value, and whether the operation is a mutation, which changes state and is audited by [[audit]], or a read.
- **Arguments** besides `actor` and `at`, which every operation takes ([[argument-values]]). Every argument is required. An argument that names an existing record must name one that exists (check 5 of the [[order-of-checks]]).
- **Refusals:** the operation's own refusals, checked at check 7 of the [[order-of-checks]] in the order listed; the first that applies decides the code. Shared codes are explained in [[refusal-codes]], the others in the module's interface.
- **Result:** the members of a successful `result`, each with its type.
- **Permissions:** for each role, `all` or `own` as [[authorization]] defines them; a role that is not listed may not use the operation.
- **Audit target:** for a mutation, the `target` of its [[audit-entry|audit entry]]: the name of an argument or of a new identifier whose value is the target; `a/b` joins two values with a slash, for example `laser-101/ada` for course `laser-101` and member `ada`; `month` is the month string, for example `2026-03`.
