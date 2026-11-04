# Notes: people modules

Modules: `notices`, `households`, `guardians`, `waivers`, `ppe`, `staffing`, `guests`, `volunteering`, `access`. 38 operations, 9 interfaces, 9 registry pages.

## (a) Generated identifier prefixes

| Prefix | Record | Module |
|---|---|---|
| `ntc-N` | notice | notices |
| `shf-N` | shift | staffing |
| `gst-N` | guest pass | guests |
| `vol-N` | volunteer time entry | volunteering |
| `bdg-N` | badge | access |
| `dor-N` | door event | access |

Posted notices take their numbers from the same `ntc-` counter as `postNotice`. `householdId`, `itemId` and the other records are named by the caller.

## (b) Lists for the list-order page

| List | Order |
|---|---|
| `listNotices` → `notices` | creation order |
| `listMemberWaivers` → `signatures` | `familyId` ascending |
| `listMemberPpe` → `items` | `itemId` ascending |
| `listShifts` → `shifts` | `start` ascending, then creation order |
| `listGuestPasses` → `guestPasses` | creation order |
| `listVolunteerTime` → `entries` | creation order |
| `listDoorEvents` → `events` | creation order |
| household → `memberIds` | ascending |
| guardianship → `consentFamilyIds` | ascending |

## (c) What other modules' interfaces must offer

- billing: "Create a charge" (guests), "Add credit" with account and memo (volunteering), "A member's account" with `overdueBalanceCents` (access). All three are in the billing interface already. The household discount is applied by billing as `households/household-discount` states.
- bookings: "The active reservations of all machines of a family" (staffing, already stated) and "The opening hours of a date" (access, already stated). Bookings declares the need "does a member have an active badge that is not inside" (already stated); protocol supplies it from access.
- members: user role (staffing: member, staff or admin), tier (guests), standing (all).
- equipment: family exists (guardians, waivers, ppe, staffing); family of a machine (the gates).
- ledger: only through billing's "Add credit" (volunteering). `ledger/postings` already has the row.
- Authorization page must add `noticeId`, `guestPassId`, `badgeId` and `guardianId` as DESIGN says. Mine use them with permission `own`: `acknowledgeNotice`, `revokeGuestPass`, `enterDoor`, `leaveDoor`, `admitGuest` (on `badgeId` alone), `grantConsent` and `withdrawConsent` (on `guardianId` alone).
- Initial state: no households, guardianships, waivers, signatures, PPE items or requirements, shifts, guest allowances, passes, entries, badges, door events or notices. Every family is unsupervised. The volunteer rate is 0.
- Refusal codes added by these interfaces: see each interface's table. Gate codes are listed by the module that owns the rule: `guardian_consent_missing`, `waiver_missing`, `ppe_missing`, `supervisor_required`, `not_on_premises`.

## (d) Gaps in DESIGN and what I chose

1. **Result shapes** are not in DESIGN. Mutations return the changed record under its record name (`household`, `guardianship`, `waiver`, `signature`, `item`, `requirement`, `shift`, `supervision`, `allowance`, `guestPass`, `entry`, `badge`, `notice`). `enterDoor` and `leaveDoor` also return `event`; `admitGuest` returns `event`. `setVolunteerRate` returns `centsPerHour`. Lists return `memberId` or `date` and the list.
2. **Notice subjects.** DESIGN gives kind and `ref` but no subject for posted notices. `notices/notice-kind` fixes one per kind and the waitlist text equals the one `waitlists/waitlist-promotion` already states; that page may drop its own sentence.
3. **Guardians.** The guardian and the minor are members (check 5). `not_guardian` also covers a member with no guardianship. `birthDate` may lie after the request date (the member is then a minor). Consent can be granted to an adult who was declared a minor.
4. **Guest allowance.** No allowance for a tier means `guests_not_allowed`; `perMonth` 0 means every pass is charged. A fee of 0 still creates a charge of 0 cents, as for courses.
5. **`shift_needed`** is read as: an active reservation of a supervised family overlaps the shift, is covered now, and would not be covered without it. Reservations that are uncovered already do not stop a cancellation.
6. **`addShift`** keeps DESIGN's order, `not_staff` first. An admin may have shifts.
7. **`approver_not_on_shift`**: the staff actor's own scheduled shift must contain the request time (`start` at or before, `end` after). It applies to rejections too.
8. **PPE:** a member holds at most one unit of an item. `listMemberPpe` returns the item records with current stock.
9. **Access:** `revokeBadge` writes no door event. `leaveDoor` and `admitGuest` check no standing, hours or account. `guest_already_admitted` means a `guest` door event for the pass exists.
10. **Waivers:** the waiver record has `familyId`, `version`, `text` only. A member can hold signatures of several versions; the list shows the latest per family.
11. **`decideVolunteerTime`** credits through billing's "Add credit", which posts the ledger entry; volunteering does not post itself. A rate of 0 approves with `creditCents` 0.
