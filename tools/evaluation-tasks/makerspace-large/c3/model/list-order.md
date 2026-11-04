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
| `listMemberReservations` → `reservations` | `start` ascending, then creation order |
| `chargeNoShows` → `charges` | reservation number ascending: the number `N` of `res-N`, compared as a number |
| `listMaterials` → `materials` | `materialId` ascending |
| `generateInvoices` → `invoices` | `memberId` ascending |
| `listInvoices` → `invoices` | `month` ascending |
| invoice → `lines` | the membership line first, then the other lines by their `at` ascending, then in the order the charges were created |
| `listIncidents` → `incidents` | creation order |
| `utilizationReport` → `machines` | `machineId` ascending |
| `listAuditEntries` → `entries` | `seq` ascending |
| `listJournalEntries` → `entries` | creation order |
| `getTrialBalance` → `rows` | the listing order of the ledger accounts |
| `listNotices` → `notices` | creation order |
| household → `memberIds` | ascending |
| guardianship → `consentFamilyIds` | ascending |
| `listMemberWaivers` → `signatures` | `familyId` ascending |
| `listMemberPpe` → `items` | `itemId` ascending |
| `listLockouts` → `lockouts` | creation order |
| `listInspectionsDue` → `plans` | `machineId` ascending |
| `listShifts` → `shifts` | `start` ascending, then creation order |
| `listParts` → `parts` | `partId` ascending |
| `getRoomSchedule` → `holds` | `start` ascending, then creation order |
| `listDeposits` → `deposits` | creation order |
| `listMemberPasses` → `passes` | creation order |
| `listMemberPromotions` → `redemptions` | creation order |
| `listAwards` → `awards` | creation order |
| `listRefunds` → `refunds` | creation order |
| `runDunning` → `actions` | invoices by ascending number, the steps of one invoice in the order notice, fee, suspend |
| `listDunningCases` → `cases` | creation order |
| `billLockers` → `charges` | `lockerId` ascending |
| `listGuestPasses` → `guestPasses` | creation order |
| `listVolunteerTime` → `entries` | creation order |
| `listConsumption` → `rows` | `materialId` ascending |
| `listDoorEvents` → `events` | creation order |
| `listInstructors` → `instructors` | `userId` ascending |
| `listCourseClasses` → `classes` | `start` ascending, then creation order |
| `listClassAttendance` → `attendance` | `memberId` ascending |
| `listAttempts` → `attempts` | the order the attempts were recorded |
| `listWaitlist` → `entries` | join order |
| `promoteWaitlist` → `enrollments`, `skipped` | join order |
| `generatePayouts` → `payouts` | `userId` ascending |
| `listPayouts` → `payouts` | creation order |
| `listEventTickets` → `tickets` | creation order |
| `cancelEvent` → `tickets` | creation order |
| `listHazards` → `hazards` | `materialId` ascending |
| `listOrders` → `orders` | creation order |
| order → `lines` | the order the lines were added |
| stocktake → `counts` | `materialId` ascending |
| `listWorkOrders` → `workOrders` | creation order |
| `listEscalations` → `escalations` | creation order |
| escalation → `cancelledReservationIds` | reservation `start` ascending, then creation order |
| project → `collaboratorIds` | ascending |
| project → `reservationIds`, `purchaseIds` | the order attached |
| `getProjectCost` → `byMember` | `memberId` ascending |
| quote → `lines` | the order the lines were added |
| `listJobs` → `jobs` | creation order |
| `checkEligibility` → `blockers` | the order of the refusals of `createReservation` |
| `complianceSummary` → `families` | `familyId` ascending |
| `chargesByKind` → `rows` | the kind order `membership`, `usage`, `material`, `course`, `deposit`, `pass`, `locker`, `guest`, `lateFee`, `event`, `noShow` |
| `agedReceivables` → `rows` | invoices by ascending number |
| `householdStatement` → `rows` | `memberId` ascending |

"Creation order" means the order in which records of a kind were created, which is the order of their generated numbers.
