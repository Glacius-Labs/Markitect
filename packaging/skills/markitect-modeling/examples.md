# Examples

Contents: Manifest (`.markitect/model.yaml`); Execution configuration (`.markitect/execution.yaml`); Project anchor; Dependency rule; Placement; Interface; Cross-cutting rule; Rule for some areas; Bounded Scalar with a unit; Format Scalar; Enumeration with meanings; Unit; Role; Refusal code; Record; Project Kind implementing a Library interface; The project Kind's own Property; Operation (a mutation of the project Kind); Operation (a read); Operation that needs a record of another area.

Each block is a complete Definition file from the Makerspace v2 reference model in the Markitect repository (`tools/comparison/makerspace-v2/`), which compiles with the standard Libraries. Copy the shape, not the content.

## Manifest (`.markitect/model.yaml`)

The project model's ID, its root folder and the exact Library versions; `model-lock.yaml` beside it pins their digests (write it with `model_analysis.py lock`).

`model.yaml`

```yaml
formatVersion: 1
id: makerspace
root: model
dependencies:
  markitect-core: "1.2.0"
  markitect-values: "1.0.0"
  markitect-data: "1.0.0"
  markitect-api: "1.0.0"
  markitect-practice: "1.0.0"
  markitect-architecture: "1.0.0"
```

## Execution configuration (`.markitect/execution.yaml`)

Ownership gives each Package its code folder. The `scope` selector makes every `Practice.Rule`, and every Kind implementing it, reach the Managers its `appliesTo` names. Checks bind the build and test commands; `requiredBy` names a Definition.

`execution.yaml`

```yaml
apiVersion: markitect.execution/v2
ownership:
  - {path: ./, owner: .}
  - {path: realization/, owner: makerspace}
  - {path: realization/cmd/makerspace/, owner: makerspace/protocol}
  - {path: realization/protocol/, owner: makerspace/protocol}
  - {path: realization/policies/, owner: makerspace/policies}
  - {path: realization/members/, owner: makerspace/members}
  - {path: realization/equipment/, owner: makerspace/equipment}
  - {path: realization/billing/, owner: makerspace/billing}
  - {path: realization/training/, owner: makerspace/training}
  - {path: realization/bookings/, owner: makerspace/bookings}
  - {path: realization/materials/, owner: makerspace/materials}
  - {path: realization/safety/, owner: makerspace/safety}
  - {path: realization/reporting/, owner: makerspace/reporting}
scope:
  - {kind: markitect/practice/Rule, property: markitect/practice/appliesTo}
checks:
  - id: project-build
    profile: go
    directory: realization
    args: [build, ./cmd/makerspace]
    requiredBy: makerspace/makerspace
    timeout: 2m
  - id: project-tests
    profile: go
    directory: realization
    args: [test, ./...]
    requiredBy: makerspace/tests
    timeout: 5m
```

## Project anchor

In the top-level project Package; Definitions may not sit in the model root.

`model/makerspace/makerspace.yaml`

```yaml
imports:
  - {package: markitect/practice, as: Practice}
  - {package: makerspace/policies, as: Policies}
id: makerspace
kind: Practice.Project
body: |-
  Makerspace runs the day-to-day operations of a fictional community workshop. It is one local program. Front-desk tools send it requests on behalf of the person who acts; the program decides, records and answers.

  - **Members** join on a membership tier and pay a monthly fee that depends on the tier. Staff can suspend a member; a suspended member is not in good standing.
  - **Training:** courses certify members for a tool family, such as laser cutters. A certification is valid for 365 days.
  - **Equipment:** machines belong to a tool family and have a rate per minute. Staff schedule maintenance windows during which a machine cannot be reserved.
  - **Bookings:** members reserve machines in 30-minute steps inside opening hours, check in when they start and check out when they finish. They are charged for every metered minute at the machine's rate, minus a tier discount.
  - **Materials:** a small shop sells materials from stock.
  - **Billing:** once a month staff generate invoices: one per member, with the monthly fee and all charges of that month. An invoice is due 14 days after it is issued. Members pay their invoices; an overpayment becomes credit, and credit reduces the next invoice.
  - **Safety:** staff record safety incidents per machine.
  - **Reporting:** staff and admins read utilization and revenue reports.

  Four policies apply to every operation: [[Policies.authorization|authorization]], [[Policies.audit|audit]], [[Policies.money|money]] and [[Policies.time|time]].

  Out of scope: persistence across runs, network access, user interfaces, concurrency, real payments or refunds, notifications, scheduled jobs, deletion of records and integration with other systems.

  The Go module lives at `realization/go.mod`; the program's main package lives at `realization/cmd/makerspace`. Only the Go standard library is used. The program is divided into the modules of its architecture; each module is one Go package under `realization/`, named after the module, for example `realization/billing`; `realization/protocol` and `realization/cmd/makerspace` belong to the protocol module.
spec:
  language: Go
  moduleRoot: realization
  buildCommand: go build ./cmd/makerspace
  testCommand: go test ./...
```

## Dependency rule

One per Package, in the project Package.

`model/makerspace/billing-dependencies.yaml`

```yaml
imports:
  - {package: markitect/architecture, as: Architecture}
id: billing-dependencies
kind: Architecture.DependencyRule
body: |-
  The billing module depends only on members, policies.
spec:
  appliesTo: [makerspace/billing]
  mayDependOn: [makerspace/members, makerspace/policies]
```

## Placement

Where a cross-cutting rule is realized once.

`model/makerspace/audit-placement.yaml`

```yaml
imports:
  - {package: markitect/architecture, as: Architecture}
  - {package: makerspace/policies, as: Policies}
id: audit-placement
kind: Architecture.Placement
body: |-
  The protocol module appends the audit entry after a mutation is answered `ok`, using the operation's audit target; area modules do not write audit entries.
spec:
  appliesTo: [makerspace]
  concern: Policies.audit
  realizedIn: makerspace/protocol
```

## Interface

Each area's public surface, in that area.

`model/makerspace/billing/billing-interface.yaml`

```yaml
imports:
  - {package: markitect/architecture, as: Architecture}
  - {package: makerspace/policies, as: Policies}
id: billing-interface
kind: Architecture.Interface
body: |-
  What other modules may use of billing: its operations and the charge, invoice, payment and account records. Other modules create a charge for a member through billing; billing computes its amount by [[Policies.money|money]].
spec:
  appliesTo: [makerspace]
  operations: [generate-invoices, list-invoices, record-payment, issue-credit, get-account]
  types: [charge, invoice, payment, account]
```

## Cross-cutting rule

Lives once in the shared kernel; `appliesTo: [makerspace]` reaches every Manager.

`model/makerspace/policies/money.yaml`

```yaml
imports:
  - {package: markitect/practice, as: Practice}
id: money
kind: Practice.Rule
body: |-
  - All amounts are integer numbers of [[cents|cents]], represented as JSON integers. There are no fractions of a cent, no floating-point amounts and no currency field. Stored and returned amounts are never negative.
  - Every charge is a line with a `quantity`, a `unitPriceCents` and a `discountPercent`. Its amount is
    `amountCents = (quantity × unitPriceCents × (100 − discountPercent) + 50) div 100`,
    where `div` is integer division. This is the exact value rounded half up to a whole cent. Rounding happens once per line. Totals are sums of rounded line amounts and are never rounded again.
    Example: 37 minutes at 45 cents with a 10 % discount: `(37 × 45 × 90 + 50) div 100 = 149900 div 100 = 1499` cents.
  - A charge's amount is fixed when the charge is created. Later changes to a rate, a tier or a price never change existing charges.
  - Each member has a credit balance, initially 0. It increases by credits issued by an admin and by overpayments. It decreases only when credit is applied to a new invoice. It is never negative. Credit is never paid out.
spec:
  appliesTo: [makerspace]
```

## Rule for some areas

`appliesTo` lists the Packages that need it.

`model/makerspace/policies/opening-hours.yaml`

```yaml
imports:
  - {package: markitect/practice, as: Practice}
id: opening-hours
kind: Practice.Rule
body: |-
  **Opening hours** are the same every week:

  | Day | Opens | Closes | Open minutes |
  | --- | --- | --- | --- |
  | Monday to Friday | 09:00 | 21:00 | 720 |
  | Saturday | 10:00 | 18:00 | 480 |
  | Sunday | closed | closed | 0 |

  The opening period of a day is the interval `[opens, closes)` on that date. A reservation is inside opening hours when its whole interval lies in the opening period of its start date: its start is at or after the opening time and its end is at or before the closing time of that same date. A reservation ending exactly at closing time is inside. Sunday has no opening period.
spec:
  appliesTo: [makerspace/bookings, makerspace/reporting]
```

## Bounded Scalar with a unit

`model/makerspace/equipment/rate-cents-per-minute.yaml`

```yaml
imports:
  - {package: markitect/values, as: Values}
  - {package: makerspace/policies, as: Policies}
id: rate-cents-per-minute
kind: Values.Scalar
body: |-
  A machine's rate per minute.
spec:
  base: integer
  lowest: 1
  highest: 10000
  measuredIn: Policies.cents
```

## Format Scalar

The body states the format; the base type is typed.

`model/makerspace/policies/timestamp.yaml`

```yaml
imports:
  - {package: markitect/values, as: Values}
id: timestamp
kind: Values.Scalar
body: |-
  A JSON string `YYYY-MM-DDTHH:MMZ` naming a valid UTC calendar date and time from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`, for example `2026-03-02T09:00Z`. Hours are `00` to `23`, minutes `00` to `59`; there are no seconds.
spec:
  base: string
```

## Enumeration with meanings

`model/makerspace/members/tier.yaml`

```yaml
imports:
  - {package: markitect/values, as: Values}
id: tier
kind: Values.Enumeration
body: |-
  A membership tier. It sets the monthly fee and the discount on machine usage.
spec:
  members:
    - {value: basic, meaning: "Monthly fee 2500 cents, usage discount 0 %."}
    - {value: plus, meaning: "Monthly fee 4900 cents, usage discount 10 %."}
    - {value: pro, meaning: "Monthly fee 9900 cents, usage discount 25 %."}
```

## Unit

`model/makerspace/policies/cents.yaml`

```yaml
imports:
  - {package: markitect/values, as: Values}
id: cents
kind: Values.Unit
body: |-
  One cent; all amounts are integer numbers of cents.
spec:
  symbol: ct
```

## Role

`model/makerspace/policies/staff-role.yaml`

```yaml
imports:
  - {package: markitect/api, as: Api}
id: staff-role
kind: Api.Role
body: |-
  Staff run the front desk and the workshop floor.
spec: {}
```

## Refusal code

`model/makerspace/policies/invalid-state.yaml`

```yaml
imports:
  - {package: markitect/api, as: Api}
id: invalid-state
kind: Api.RefusalCode
body: |-
  The record is not in a state that allows the operation.
spec:
  code: invalid_state
```

## Record

`key` names the identifying field; `refersTo` points at an existing record; `list` embeds a list of records.

`model/makerspace/billing/invoice.yaml`

```yaml
imports:
  - {package: markitect/data, as: Data}
  - {package: makerspace/policies, as: Policies}
  - {package: makerspace/members, as: Members}
id: invoice
kind: Data.Entity
body: |-
  An invoice of one member for one month, generated identifier `inv-N`.
spec:
  fields:
    - {name: invoiceId, fieldType: Policies.identifier}
    - {name: memberId, fieldType: Policies.identifier, refersTo: Members.member}
    - {name: month, fieldType: Policies.month}
    - {name: issuedAt, fieldType: Policies.timestamp}
    - {name: dueAt, fieldType: Policies.timestamp}
    - {name: lines, fieldType: charge, list: true}
    - {name: totalCents, fieldType: Policies.amount-cents}
    - {name: creditAppliedCents, fieldType: Policies.amount-cents}
    - {name: paidCents, fieldType: Policies.amount-cents}
    - {name: balanceCents, fieldType: Policies.amount-cents}
    - {name: status, fieldType: invoice-status}
    - {name: overdue, fieldType: Policies.flag}
  key: invoiceId
```

## Project Kind implementing a Library interface

It declares every Property of `Api.Operation` plus its own; its Definitions are accepted wherever an `Api.Operation` is required.

`model/makerspace/policies/audited-operation.yaml`

```yaml
imports:
  - {package: markitect/core, as: Core}
  - {package: markitect/api, as: Api}
id: AuditedOperation
kind: Core.Kind
body: |-
  An operation that changes state and therefore names the target of its audit entry.
spec:
  implements: [Api.Operation]
  properties: [Api.wireName, Api.mutation, Api.arguments, Api.refusals, Api.results, Api.permissions, auditTarget]
```

## The project Kind's own Property

`model/makerspace/policies/audit-target.yaml`

```yaml
imports:
  - {package: markitect/core, as: Core}
id: auditTarget
kind: Core.Property
body: |-
  The `target` of the operation's [[audit-entry|audit entry]]: the name of an argument or of a new identifier whose value is the target; `a/b` joins two values with a slash, for example `laser-101/ada` for course `laser-101` and member `ada`; `month` is the month string, for example `2026-03`.
spec:
  valueType: string
  minCount: 1
  maxCount: 1
```

## Operation (a mutation of the project Kind)

Refusals in checking order; a role without a permission entry has none; the body states only the effect.

`model/makerspace/bookings/create-reservation.yaml`

```yaml
imports:
  - {package: makerspace/policies, as: Policies}
  - {package: makerspace/members, as: Members}
  - {package: makerspace/training, as: Training}
  - {package: makerspace/equipment, as: Equipment}
id: create-reservation
kind: Policies.AuditedOperation
body: |-
  Creates the reservation with status `reserved` and the next reservation identifier.

  Example: on Saturday `2026-03-07`, the slot `17:30`–`18:00` is inside opening hours and `17:30`–`18:30` is not; on Tuesday `2026-03-03`, `10:15`–`11:00` is refused with `invalid_slot`.
spec:
  wireName: createReservation
  mutation: true
  arguments:
    - {name: memberId, fieldType: Policies.identifier, refersTo: Members.member}
    - {name: machineId, fieldType: Policies.identifier, refersTo: Equipment.machine}
    - {name: start, fieldType: Policies.timestamp}
    - {name: end, fieldType: Policies.timestamp}
  refusals:
    - {refusal: Policies.invalid-interval, when: "`end` is not after `start`"}
    - {refusal: invalid-slot, when: "`start` or `end` is not on a 30-minute boundary or the length is outside 30 to 240 minutes"}
    - {refusal: Policies.starts-in-past, when: "`start` is before the request time"}
    - {refusal: outside-opening-hours, when: the slot is not inside the opening hours of its start date}
    - {refusal: Members.member-suspended, when: the member is not in good standing}
    - {refusal: Training.not-certified, when: "the member has no certification for the machine's family valid on the date of `start`"}
    - {refusal: machine-in-maintenance, when: a scheduled maintenance window of the machine overlaps the slot}
    - {refusal: machine-booked, when: another active reservation of the machine overlaps the slot}
    - {refusal: member-double-booked, when: another active reservation of the member overlaps the slot}
  results:
    - {name: reservation, fieldType: reservation}
  permissions:
    - {role: Policies.member-role, scope: own}
    - {role: Policies.staff-role, scope: all}
    - {role: Policies.admin-role, scope: all}
  auditTarget: new reservationId
```

## Operation (a read)

`model/makerspace/billing/get-account.yaml`

```yaml
imports:
  - {package: markitect/api, as: Api}
  - {package: makerspace/policies, as: Policies}
  - {package: makerspace/members, as: Members}
id: get-account
kind: Api.Operation
body: |-
  Returns the member's account: `creditBalanceCents` is the member's credit balance; `openBalanceCents` is the sum of `balanceCents` over the member's invoices with status `open`; `overdueBalanceCents` is the same sum over the member's overdue invoices; `unbilledCents` is the sum of the amounts of the member's unbilled charges.
spec:
  wireName: getAccount
  mutation: false
  arguments:
    - {name: memberId, fieldType: Policies.identifier, refersTo: Members.member}
  results:
    - {name: account, fieldType: account}
  permissions:
    - {role: Policies.member-role, scope: own}
    - {role: Policies.staff-role, scope: all}
    - {role: Policies.admin-role, scope: all}
```

## Operation that needs a record of another area

The body link and the result type are dependency edges on billing; the dependency rules allow bookings -> billing.

`model/makerspace/bookings/check-out.yaml`

```yaml
imports:
  - {package: makerspace/policies, as: Policies}
  - {package: makerspace/billing, as: Billing}
id: check-out
kind: Policies.AuditedOperation
body: |-
  Status `completed`; `checkedOutAt` is the request time; `meteredMinutes` is `checkedOutAt − checkedInAt` in minutes, which may be 0 and may extend past the reservation's end or closing time. A usage [[Billing.charge|charge]] is created for the member: `kind` `usage`, `ref` the reservation identifier, `quantity` the metered minutes, `unitPriceCents` the captured rate, `discountPercent` the discount of the member's tier at check-out, `at` the request time. A charge of 0 cents is still created. The reservation's `amountCents` is the charge's amount. Standing is not checked.
spec:
  wireName: checkOut
  mutation: true
  arguments:
    - {name: reservationId, fieldType: Policies.identifier, refersTo: reservation}
  refusals:
    - {refusal: Policies.invalid-state, when: "the status is not `checkedIn`"}
  results:
    - {name: reservation, fieldType: reservation}
    - {name: charge, fieldType: Billing.charge}
  permissions:
    - {role: Policies.member-role, scope: own}
    - {role: Policies.staff-role, scope: all}
    - {role: Policies.admin-role, scope: all}
  auditTarget: reservationId
```
