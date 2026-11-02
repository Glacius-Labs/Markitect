# Makerspace

Makerspace runs the day-to-day operations of a fictional community workshop: members, training, equipment, bookings, materials, billing, safety and reporting. It is one local program that reads requests from standard input and answers each with one line on standard output.

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

Each module of the program is one Go package, named after the module. A module depends only on itself and on the modules the architecture lists for it.

| Package | Content |
| --- | --- |
| `policies` | The project-wide rules: value formats, refusal codes, roles, time and money rules, and the vocabulary in which a module describes its operations |
| `members` | Users, members, tiers and standing |
| `equipment` | Tool families, machines and maintenance windows |
| `billing` | Charges, invoices, payments and credit |
| `training` | Courses, enrollments and certifications |
| `bookings` | Reservations, opening hours and the booking rules |
| `materials` | The shop: materials and purchases |
| `safety` | Safety incidents |
| `reporting` | Utilization and revenue reports |
| `protocol` | The outer layer: request parsing, the checks every operation shares, the audit log and the wiring of the modules |
| `cmd/makerspace` | The entry point |

The tests of the operations send request lines through the protocol and read the response lines, so they live in `protocol`; the pure helpers are tested in `policies` and `reporting`.

## Scope

The program keeps all its state in memory for one process and takes all time from the requests. Out of scope: persistence across runs, network access, user interfaces, concurrency, real payments or refunds, notifications, scheduled jobs, deletion of records and integration with other systems.
