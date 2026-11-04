---
kind: std/Concept
---
# Lockout source

Where a lockout came from. It decides who can release it.

| Value | Meaning |
| --- | --- |
| `manual` | Applied by staff or an admin with `applyLockout`. Released with `releaseLockout`. |
| `inspection` | Applied by a failed inspection. Released by a passed inspection of the same machine. |
| `workOrder` | Applied when a work order is opened. Released when that work order is closed. |
