# Makerspace requirements

Neutral product requirements for the Makerspace program, written from the [product brief](../brief.md). They are the single source of product truth for every later Phase C input: both writers' representations, the hidden oracle and the hidden reference implementations. They are organized by business area and describe only observable behavior.

| File | Content |
| --- | --- |
| [w0.md](w0.md) | The complete requirements for the fresh build: domain, roles, the program interface, every operation with its arguments, results and refusal codes, the four policies (authorization, audit, money, time) and the boundary rules. |
| [c1.md](c1.md) | Audit also records refusals for missing permission, with the actor's role. |
| [c2.md](c2.md) | Safety lockouts: a new operation, cancellations with credit, a new refusal, schedule and utilization changes. |
| [c3.md](c3.md) | Bulk price tiers for materials, with what must not change. |
| [c4.md](c4.md) | The new role steward. |
| [c5.md](c5.md) | Invoices more than 30 days overdue block reservations and purchases. |
| [c6.md](c6.md) | Certifications no longer expire. |
| [c7.md](c7.md) | Payments folded into invoices: a restatement of the billing text with no behavior change. |
| [c8.md](c8.md) | 15-minute booking granularity and new opening hours. |
| [c9.md](c9.md) | Unchanged re-application. |

## Reading the steps

Each change is a delta against the step before it, not against W0 alone. The requirements of step Cn are W0 with C1 to Cn applied in order. Section numbers in the change documents refer to `w0.md`. Every change states what changes and what stays as it is; C3 and C7 also state explicitly what must not change.

Where the brief left a product choice open, these documents decide it and say so in plain terms, for example the tier fees and discounts, the 14-day due period, the lockout credit of 1000 cents and the bulk discount table.

## Transport

The program speaks JSON Lines on standard input and output with the Library comparison program's envelope (`id`, `op`, `args`) and statuses (`ok`, `refused`, `invalidRequest`), so the Library oracle's process transport can be reused. Two differences: every refusal carries an exact `code`, and every `args` object carries the acting user (`actor`) and the request time (`at`).
