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
| `billing` | [[billing]] | `members`, `policies` |
| `training` | [[training]] | `members`, `equipment`, `billing`, `policies` |
| `bookings` | [[bookings]] | `members`, `training`, `equipment`, `billing`, `safety`, `policies` |
| `materials` | [[materials]] | `members`, `billing`, `policies` |
| `safety` | [[safety]] | `equipment`, `policies` |
| `reporting` | [[reporting]] | `equipment`, `bookings`, `billing`, `policies` |
| `protocol` | [[protocol]] | `policies`, `members`, `equipment`, `billing`, `training`, `bookings`, `materials`, `safety`, `reporting` |

- `policies`: The policies module depends on no other module of the project; every other module may depend on it.
- `equipment`: Where maintenance must know about reservations, equipment declares what it needs and the protocol module supplies it from bookings, so equipment never depends on bookings.
- `bookings`: Depends on safety because a lockout belongs to an open incident.
- `protocol`: The protocol module is the outer layer: it reads requests, runs the checks every operation shares and calls the area modules. It may depend on every other module; no module depends on it.

**Placements.** Some project-wide rules are realized once, in one module:

- The protocol module appends the audit entry after a mutation is answered `ok`, using the operation's audit target, and after a `forbidden` refusal; area modules do not write audit entries.
- Checks 2, 4 and 6 of the [[order-of-checks|order of checks]] run once, in the protocol module, before any operation's own rules; area modules do not check roles or ownership themselves.
- The protocol module keeps the request-time order (check 3) and passes the request time to the area modules as their only notion of now.
- The protocol module realizes the whole [[order-of-checks]], including the argument checks of [[argument-values]] (check 1) and the referenced records (check 5). These rules sit in the project itself so that every module reads them, not because the `policies` module realizes them.

**Layout.** `realization/go.mod`, `realization/README.md` and the `policies` package `realization/policies/` belong to the project itself; `realization/protocol/` and `realization/cmd/makerspace/` belong to [[protocol]]; every other module owns `realization/<module>/`.
