# Notes: trade cluster

Modules: hazmat, purchasing, stocktake, projects, quotes, jobs, and the changes to materials.

## (a) Generated identifier prefixes

| Prefix | Record |
|---|---|
| `po-N` | purchasing order |
| `stk-N` | stocktake |
| `quo-N` | quote |
| `job-N` | job |

Suppliers, customers, projects and classifications use caller-chosen identifiers.

## (b) New lists for the list-order page

| List | Order |
|---|---|
| `listHazards` → `hazards` | `materialId` ascending |
| `listOrders` → `orders` | creation order |
| order → `lines` | the order the lines were added |
| stocktake → `counts` (`getStocktake`, and every operation returning a stocktake) | `materialId` ascending |
| project → `collaboratorIds` | ascending |
| project → `reservationIds`, `purchaseIds` | the order attached |
| `getProjectCost` → `byMember` | `memberId` ascending |
| quote → `lines` | the order the lines were added |
| `listJobs` → `jobs` | creation order |

Also for the initial-state and generated-identifier pages: the four prefixes above; no trade record exists at the start.

## (c) What I need from other modules' interfaces

- **ledger:** an entry-posting capability for the events of `ledger/postings` (`receiveOrder`, `closeStocktake` shrink and gain, `completeJob`, `recordJobPayment`). Pages used: `ledger/postings`, `ledger/ledger-interface`.
- **billing:** a capability "increase a member's credit balance by an amount" (used by `completeJob`; also by other modules).
- **promotions:** `promotions/promotions-interface` and `promotions/redeemed-promotions`: for a member and a scope, the percent of the oldest unused redeemed promotion, and marking that redemption used by a charge `ref` (here the purchaseId) at the request time.
- **training:** a member's certification for a family valid on a date (hazmat, jobs).
- **equipment:** a machine's existence, family and current rate (quotes, jobs); a family's existence (hazmat).
- **bookings:** a reservation by identifier with its `memberId`, `status` and `amountCents` (projects).
- **protocol:** supplies hazmat's answer and stocktake's answer to materials, as materials' interface declares. The stocktake answer (a material has a count in an open stocktake) also serves `receiveOrder`: purchasing does not depend on stocktake, so it asks materials, which offers that capability.
- **Authorization shared page:** ownership for `ownerId` and `projectId` as DESIGN states; my operation pages say "judged on `projectId` alone / `ownerId`" and link `authorization`.
- **consumables author:** use materials' capability "drawing stock" (stock decreases by the smaller of units and stock; the number drawn is answered). It does not look at stocktake.

## (d) Gaps in the design and what I chose

1. `stocktake_open` at `receiveOrder`: purchasing depends only on materials and ledger, not stocktake. Chosen: materials offers "whether a material has a count in an open stocktake" (answered through its declared need on stocktake); purchasing asks that.
2. A `count`'s `difference` is the program's first signed integer (negative for a shrink). The count page says so. `integer-limits` ("no computed integer is negative") and `values` need a line for it; integrator to decide.
3. Hazmat answer: materials cannot depend on training, so the hazard need is a single question to hazmat ("which refusal applies, if any"), not two separate ones.
4. New supplier status is `active` (design does not say).
5. Quote `subtotalCents` and `totalCents` are 0 while draft; `sentAt` and `validUntil` are `null` until sent.
6. `quote_expired`: valid on `validUntil` itself, refused the day after. Rejection never expires.
7. The same `ref` may appear on several quote lines (design forbids nothing). A machine line's `quantity` is minutes.
8. `getProjectCost`: a member "has a share" when an attached completed reservation or an attached purchase is theirs. The result also echoes `projectId`.
9. `declassifyHazard` returns the removed classification as `classification`; `listOrders` and `listJobs` echo `supplierId` and `customerId`.
10. Record page names chosen: `hazmat/classification`, `stocktake/count`, `projects/cost-share`, `purchasing/order-line`, `quotes/quote-line`, `quotes/customer`, `purchasing/supplier`.
11. Extra rule pages beyond the agreed ones: `purchasing/receiving`, `projects/attachment`, `projects/project-cost`, `quotes/quote-pricing`, `quotes/quote-validity`, `jobs/maker-gate`, `jobs/maker-credit`. Enumeration pages: `purchasing/supplier-status`, `purchasing/order-status`, `quotes/quote-status`.
12. Hazard gate counts all purchases of the material in the month of the request time, whatever the classification was when they were made. Reclassifying changes only the limit and family judged.
13. Registry pages are empty tables, like the old ones.
