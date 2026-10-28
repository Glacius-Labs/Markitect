---
kind: std/Rule
---
# List order

Identifiers are ordered by ASCII byte value (so `-` < `.` < digits < uppercase letters < `_` < lowercase letters, and `M10` < `M9`). Listed results are ordered as follows:

| List | Order |
| --- | --- |
| `listCertifications` → `certifications` | `familyId` ascending |
| `listMachines` → `machines` | `machineId` ascending |
| `getMachineSchedule` → `reservations` | `start` ascending, then creation order |
| `getMachineSchedule` → `maintenance` | `start` ascending, then creation order |
| `getMachineSchedule` → `lockouts` | `start` ascending, then creation order |
| `listMemberReservations` → `reservations` | `start` ascending, then creation order |
| `lockMachine` → `cancelledReservations` and a lockout's `cancelledReservationIds` | creation order |
| `listMaterials` → `materials` | `materialId` ascending |
| `generateInvoices` → `invoices` | `memberId` ascending |
| `listInvoices` → `invoices` | `month` ascending |
| invoice → `lines` | the membership line first, then the other lines by their `at` ascending, then in the order the charges were created |
| `listIncidents` → `incidents` | creation order |
| `utilizationReport` → `machines` | `machineId` ascending |
| `listAuditEntries` → `entries` | `seq` ascending |

"Creation order" means the order in which records of a kind were created, which is the order of their generated numbers.
