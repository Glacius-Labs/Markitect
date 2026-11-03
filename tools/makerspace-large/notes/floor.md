# Floor cluster notes

Modules: lockouts, inspections, parts, consumables, workorders, escalations, compliance; changes to bookings and safety.

## (a) New generated identifiers

| Prefix | Record |
|---|---|
| `lck-N` | lockout (one counter for every source: manual, inspection, workOrder) |
| `wko-N` | work order |
| `esc-N` | escalation |

## (b) New lists, for the list-order page

| List | Order |
|---|---|
| `listLockouts` → `lockouts` | creation order |
| `listInspectionsDue` → `plans` | `machineId` ascending |
| `listParts` → `parts` | `partId` ascending |
| `listConsumption` → `rows` | `materialId` ascending |
| `listWorkOrders` → `workOrders` | creation order |
| `listEscalations` → `escalations` | creation order |
| escalation → `cancelledReservationIds` | the order of cancellation: reservation `start` ascending, then creation order |
| `checkEligibility` → `blockers` | the order of `createReservation`'s refusals |
| `complianceSummary` → `families` | `familyId` ascending |

Initial state page: add the three prefixes above; no floor record exists at the start.

## (c) What I need from other modules

- equipment: the list of all families (ascending `familyId`) for `complianceSummary`; a machine's family (already said).
- materials: already says what consumables needs (stock, price, drawing stock). Charge page: `ref` of a `material` charge may also be a reservation identifier (consumables).
- billing: create a charge for a member (said already). The charge page's `ref` description should mention reservation identifiers for material charges.
- safety: an incident by identifier (done on the safety interface).
- notices: posting a `cancellation` notice (said already by notices). Escalations relies on its subject coming from the kind.
- members: users (`members/user`) for `workorders` `openedBy`; the member's standing.
- training: certification valid on a date (for `compliance`).
- guardians: minor on a date; consent held for a family. waivers: the gate (current-version signature) for a member and family. ppe: the member holds every item the family requires.
- passes: `pass-draw` is linked from `checkOut`; it names `drawn` and `billable`, which `checkOut` uses.
- staffing: the supervision rule; bookings asks it whether a family is supervised and whether an interval is covered.
- access: `access/premises-gate` is linked from `checkIn` but did not exist when I wrote. Bookings' declared need on access is stated in the bookings interface (a member has an active badge that is not inside).
- protocol: supplies bookings' need on access, as it does equipment's need on bookings.
- computed-times page: an inspection plan's `nextDueOn` can lie after 2099 (`lastPassedOn` up to 2099-12-31 plus 3650 days, so up to 2109). Add it to the list of computed dates.
- authorization page: nothing new for the floor modules (they use `memberId` only in compliance).
- project page overview: a line for lockouts/inspections/parts/consumables/workorders/escalations/compliance (the integrator owns it).

## (d) Gaps in DESIGN and my choices

- `linkIncident` `already_linked`: read as "this work order already has an incident". Several work orders may link the same incident. (The other reading would make `escalateIncident` fail for an incident already linked by hand, and it has no refusal for that.)
- `getInspectionStatus` result: the DESIGN names no members. I gave `plan` and `due` (true when the request date is on or after `nextDueOn`).
- `recordInspection`, `setInspectionPlan`: result is `plan`. `note` of `recordInspection` is the lockout's `reason` on a failure and is stored nowhere else.
- Reason of lockouts applied by other modules: inspection = the `note`; work order = the `summary`.
- `clearConsumption` result: `machineId`, `materialId`. `setConsumption`: `row`. `listConsumption`: `machineId`, `rows`.
- `checkOut` keeps its old result shape: the usage `charge` only; material charges are not in the result.
- `checkOut` rule names `drawn` and `billable` are those of `passes/pass-draw`.
- An escalation cancels reservations with reported status `reserved` and `start` strictly after the request time. A reserved reservation that has started but is not checked in is left alone.
- Lockouts, cancellations and notices of an escalation are not audited on their own (one entry for the escalation).
- Rule page names not in DESIGN's list: `lockouts/lockouts`, `inspections/inspection-cycle`, `parts/spare-parts`, `workorders/work-order-lifecycle`, `compliance/eligibility` (all `internal: true`); record pages `lockouts/lockout-source`, `consumables/consumption-row`, `escalations/escalation-policy`, `compliance/family-compliance`.
- `safety/incidents` names `escalateIncident` in plain words, not as a link, so that safety does not link upward.
- Registry pages are empty tables, as the old ones.
