---
kind: std/Decision
---
# Architecture

The program is divided into the modules below, with one-way dependencies; the owner decided this module cut on 27 October 2026. Each module is one Go package under `realization/`, named after the module, for example `realization/billing`. A module may depend only on itself and on the modules listed for it; the Go standard library is not counted.

| Module | Model | May depend on |
| --- | --- | --- |
| `policies` | this folder, the project itself | nothing |
| `members` | [[members]] | `policies` |
| `equipment` | [[equipment]] | `policies` |
| `ledger` | [[ledger]] | `policies` |
| `notices` | [[notices]] | `members`, `policies` |
| `households` | [[households]] | `members`, `policies` |
| `guardians` | [[guardians]] | `members`, `equipment`, `policies` |
| `waivers` | [[waivers]] | `members`, `equipment`, `guardians`, `policies` |
| `ppe` | [[ppe]] | `members`, `equipment`, `policies` |
| `lockouts` | [[lockouts]] | `equipment`, `policies` |
| `inspections` | [[inspections]] | `equipment`, `lockouts`, `policies` |
| `parts` | [[parts]] | `equipment`, `policies` |
| `rooms` | [[rooms]] | `policies` |
| `billing` | [[billing]] | `members`, `ledger`, `households`, `policies` |
| `deposits` | [[deposits]] | `members`, `billing`, `ledger`, `policies` |
| `packs` | [[packs]] | `members`, `billing`, `policies` |
| `promotions` | [[promotions]] | `members`, `policies` |
| `scholarships` | [[scholarships]] | `members`, `policies` |
| `refunds` | [[refunds]] | `members`, `billing`, `ledger`, `policies` |
| `dunning` | [[dunning]] | `members`, `billing`, `notices`, `policies` |
| `lockers` | [[lockers]] | `members`, `billing`, `policies` |
| `guests` | [[guests]] | `members`, `billing`, `policies` |
| `volunteering` | [[volunteering]] | `members`, `billing`, `ledger`, `policies` |
| `training` | [[training]] | `members`, `equipment`, `billing`, `guardians`, `promotions`, `scholarships`, `policies` |
| `loans` | [[loans]] | `members`, `equipment`, `training`, `guardians`, `waivers`, `billing`, `notices`, `policies` |
| `materials` | [[materials]] | `members`, `billing`, `promotions`, `policies` |
| `consumables` | [[consumables]] | `equipment`, `materials`, `billing`, `policies` |
| `bookings` | [[bookings]] | `members`, `training`, `equipment`, `billing`, `guardians`, `waivers`, `ppe`, `lockouts`, `inspections`, `packs`, `consumables`, `loans`, `policies` |
| `safety` | [[safety]] | `equipment`, `policies` |
| `reporting` | [[reporting]] | `equipment`, `bookings`, `billing`, `policies` |
| `access` | [[access]] | `members`, `billing`, `guests`, `bookings`, `policies` |
| `instructors` | [[instructors]] | `members`, `equipment`, `training`, `policies` |
| `classes` | [[classes]] | `members`, `training`, `instructors`, `rooms`, `policies` |
| `assessments` | [[assessments]] | `members`, `training`, `policies` |
| `waitlists` | [[waitlists]] | `members`, `training`, `notices`, `policies` |
| `payouts` | [[payouts]] | `members`, `ledger`, `instructors`, `classes`, `policies` |
| `events` | [[events]] | `members`, `rooms`, `billing`, `ledger`, `policies` |
| `hazmat` | [[hazmat]] | `equipment`, `materials`, `training`, `policies` |
| `purchasing` | [[purchasing]] | `materials`, `ledger`, `policies` |
| `stocktake` | [[stocktake]] | `materials`, `ledger`, `policies` |
| `workorders` | [[workorders]] | `members`, `equipment`, `safety`, `parts`, `lockouts`, `policies` |
| `escalations` | [[escalations]] | `equipment`, `safety`, `lockouts`, `bookings`, `workorders`, `notices`, `policies` |
| `projects` | [[projects]] | `members`, `bookings`, `materials`, `policies` |
| `quotes` | [[quotes]] | `equipment`, `materials`, `policies` |
| `jobs` | [[jobs]] | `members`, `equipment`, `billing`, `ledger`, `training`, `quotes`, `policies` |
| `compliance` | [[compliance]] | `members`, `equipment`, `guardians`, `waivers`, `ppe`, `training`, `lockouts`, `inspections`, `policies` |
| `finance` | [[finance]] | `members`, `ledger`, `billing`, `households`, `refunds`, `policies` |
| `protocol` | [[protocol]] | `policies`, `members`, `equipment`, `ledger`, `notices`, `households`, `guardians`, `waivers`, `ppe`, `lockouts`, `inspections`, `parts`, `rooms`, `billing`, `deposits`, `packs`, `promotions`, `scholarships`, `refunds`, `dunning`, `lockers`, `guests`, `volunteering`, `training`, `loans`, `materials`, `consumables`, `bookings`, `safety`, `reporting`, `access`, `instructors`, `classes`, `assessments`, `waitlists`, `payouts`, `events`, `hazmat`, `purchasing`, `stocktake`, `workorders`, `escalations`, `projects`, `quotes`, `jobs`, `compliance`, `finance` |

- `policies`: The policies module depends on no other module of the project; every other module may depend on it.
- `equipment`: Where maintenance must know about reservations, equipment declares what it needs and the protocol module supplies it from bookings, so equipment never depends on bookings.
- `training`: Where training must know something of `classes` and `assessments`, training declares what it needs and the protocol module supplies it, so training never depends on them.
- `materials`: Where materials must know something of `hazmat` and `stocktake`, materials declares what it needs and the protocol module supplies it, so materials never depends on them.
- `bookings`: Where bookings must know something of `access`, bookings declares what it needs and the protocol module supplies it, so bookings never depends on access.
- `instructors`: Where instructors must know something of `classes`, instructors declares what it needs and the protocol module supplies it, so instructors never depends on classes.
- `protocol`: The protocol module is the outer layer: it reads requests, runs the checks every operation shares and calls the area modules. It may depend on every other module; no module depends on it.

**Placements.** Some project-wide rules are realized once, in one module:

- The protocol module appends the audit entry after a mutation is answered `ok`, using the operation's audit target; area modules do not write audit entries.
- Checks 2, 4 and 6 of the [[order-of-checks|order of checks]] run once, in the protocol module, before any operation's own rules; area modules do not check roles or ownership themselves. An operation's own refusals at check 7 may still look at who the actor is, where the operation says so, such as `releaseLockout` and `approveOrder`; the protocol module passes the actor's identifier and role to the area module with the request, so neither depends on `members`.
- The protocol module keeps the request-time order (check 3) and passes the request time to the area modules as their only notion of now.
- The protocol module realizes the whole [[order-of-checks]], including the argument checks of [[argument-values]] (check 1) and the referenced records (check 5). These rules sit in the project itself so that every module reads them, not because the `policies` module realizes them.

**Layout.** `realization/go.mod`, `realization/README.md` and the `policies` package `realization/policies/` belong to the project itself; `realization/protocol/` and `realization/cmd/makerspace/` belong to [[protocol]]; every other module owns `realization/<module>/`.
