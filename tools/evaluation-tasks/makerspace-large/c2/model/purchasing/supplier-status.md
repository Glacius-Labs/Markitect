---
kind: std/Concept
---
# Supplier status

Whether orders may be created for a supplier and approved.

| Value | Meaning |
| --- | --- |
| `active` | Orders can be created for the supplier and approved. |
| `blocked` | No order can be created for the supplier and no order of the supplier can be approved. Orders in any other step are not affected. |
