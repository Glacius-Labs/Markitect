# Makerspace

Makerspace runs the day-to-day operations of a fictional community workshop: members and households, guardians, waivers and protective equipment, training and classes, equipment with its lockouts, inspections and work orders, bookings, materials and purchasing, billing with deposits, passes, refunds and dunning, a double-entry ledger, events, payouts, projects, quotes and jobs for customers, and the reports over all of it. It is one local program that reads requests from standard input and answers each with one line on standard output. It offers 195 operations.

## Build, run and test

From this folder (the Go module `makerspace`; only the Go standard library is used):

```text
go build ./cmd/makerspace
./makerspace
go test ./...
```

## Example session

Send one JSON object per line; each request gets exactly one response line, in order. A member registered by the admin, a read by that member, and a request the member's role may not make:

```text
{"id":"1","op":"registerMember","args":{"actor":"admin","at":"2026-03-02T09:00Z","memberId":"ada","name":"Ada Byron","tier":"plus"}}
{"id":"2","op":"listMachines","args":{"actor":"ada","at":"2026-03-02T09:00Z"}}
{"id":"3","op":"registerMember","args":{"actor":"ada","at":"2026-03-02T09:00Z","memberId":"bob","name":"Bob","tier":"basic"}}
```

```text
{"id":"1","result":{"member":{"memberId":"ada","name":"Ada Byron","registeredAt":"2026-03-02T09:00Z","standing":"good","tier":"plus"}},"status":"ok"}
{"id":"2","result":{"machines":[]},"status":"ok"}
{"code":"forbidden","id":"3","reason":"The role member may not use registerMember.","status":"refused"}
```

## Layout

Each module of the program is one Go package, named after the module. A module depends only on itself and on the modules the architecture lists for it; where a lower module needs an answer from an upper one, the `protocol` package supplies it when it wires the modules together.

| Package | Content |
| --- | --- |
| `policies` | The project-wide rules: value formats, refusal codes, roles, time and money rules, and the vocabulary in which a module describes its operations |
| `members` | Users, members, tiers and standing |
| `equipment` | Tool families, machines and maintenance windows |
| `ledger` | The double-entry journal, its accounts and the trial balance |
| `notices` | Notices to members |
| `households` | Households of members with one payer |
| `guardians` | Minors, their guardians and consent per tool family |
| `waivers` | Waivers per tool family and their signatures |
| `ppe` | Protective equipment items and what a family requires |
| `lockouts` | Lockouts that take a machine out of use |
| `inspections` | Inspection plans and due dates |
| `staffing` | Shifts and supervision |
| `parts` | Spare parts and their stock |
| `rooms` | Rooms and their holds |
| `billing` | Charges, invoices, payments and credit |
| `deposits` | Deposits held and released |
| `passes` | Minute passes |
| `promotions` | Promotions and their redemptions |
| `scholarships` | Funds and awards |
| `refunds` | Refunds of invoice lines |
| `dunning` | Overdue invoices: notices, late fees and suspension |
| `lockers` | Locker rentals |
| `guests` | Guest passes |
| `volunteering` | Volunteer time credited to members |
| `training` | Courses, enrollments and certifications |
| `loans` | Tools members borrow, their loans and the late fee |
| `materials` | The shop: materials and purchases |
| `consumables` | Materials a machine uses up per session |
| `bookings` | Reservations, opening hours and the booking rules |
| `safety` | Safety incidents |
| `reporting` | Utilization and revenue reports |
| `access` | Badges and door events |
| `instructors` | Instructors and their families |
| `classes` | Scheduled classes and attendance |
| `assessments` | Assessment attempts |
| `waitlists` | Course waitlists |
| `payouts` | Instructor payouts |
| `events` | Events and tickets |
| `hazmat` | Hazard classes of materials |
| `purchasing` | Suppliers and orders |
| `stocktake` | Stock counts |
| `workorders` | Work orders on machines |
| `escalations` | Incident escalations |
| `projects` | Projects of members, with collaborators and attached reservations and purchases |
| `quotes` | Customers and quotes for machine time and materials |
| `jobs` | Jobs done for accepted quotes, maker credit and payments |
| `compliance` | Eligibility of a member for a machine and the compliance summary |
| `finance` | Charges by kind, finance summary, aged receivables and household statements |
| `protocol` | The outer layer: request parsing, the checks every operation shares, the audit log and the wiring of the modules |
| `cmd/makerspace` | The entry point |

The tests of the operations send request lines through the protocol and read the response lines, so they live in `protocol`; the pure helpers are tested in `policies` and `reporting`.

## Scope

The program keeps all its state in memory for one process and takes all time from the requests: it has no clock and no scheduled job, so what must happen periodically, such as dunning, invoicing or payouts, is an explicit operation. Out of scope: persistence across runs, network access, user interfaces, concurrency, real payments or refunds, sending notifications (a notice is only a record), deletion of records and integration with other systems.
