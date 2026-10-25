# Makerspace: product brief (Phase C)

Purpose: a realistic, mid-sized business system for comparing Markitect with one agent on a project too large to keep in view at once. Fictional; it imitates no real organization.

Domain: a community workshop (makerspace). Members pay membership fees by tier. They take courses that certify them per tool family, with expiry. They reserve machines, check in and out, and are charged per metered minute at the machine's rate. They buy materials from a small shop. Each month they receive an invoice with a due date, and they pay it; overpayments become credit. Machines have maintenance windows during which they cannot be used. Staff record safety incidents. Staff and admins read utilization and revenue reports.

Roles: member, staff and admin. Change C4 adds the role steward. Every request names the acting user and a timestamp; authorization decides per operation.

Business areas: members (registry; memberships with tiers, fees and standing); training (courses; certifications per tool family with expiry); equipment (catalog of machines, families and rates; maintenance windows); bookings (reservations, check-in and check-out, metered minutes); materials (stock and sales); billing (monthly invoices with due dates and overdue status; payments and credit balance); safety (incidents; lockouts arrive with C2); reporting (utilization and revenue, read-only).

Cross-cutting policies on every operation: authorization (a role and permission matrix); audit (every accepted mutation is recorded with actor and time); money (integer cents, rounding defined per line, credits); time (UTC minutes, half-open intervals, opening hours, booking granularity of 30 minutes).

Cross-area rules: a reservation needs a certification valid on the reservation date for the machine's family, a machine not in maintenance, a member in good standing and a slot inside opening hours; usage produces charges in billing, audit entries and report data.

Interface and delivery: one local Go program speaking JSON Lines on standard input and output, like the Library comparison program; every request carries an explicit actor and timestamp; state is in memory and deterministic. No persistence, network, user interface, concurrency or real payments. It builds and tests with the standard Go toolchain.

Size target: about 35 operations; a realization of about 4–6k Go lines.

Backlog after the fresh build W0:
- C1: audit also records refusals for missing permission, with the actor's role.
- C2: safety lockouts: an incident can lock a machine for a window; future reservations in that window are cancelled with a credit; new bookings in it are refused; utilization excludes lockout time.
- C3: materials get bulk price tiers; bookings, memberships and training are unaffected.
- C4: new role steward: may handle maintenance and incidents, has no access to billing.
- C5: an invoice more than 30 days overdue blocks new reservations and purchases until it is paid.
- C6: certifications no longer expire.
- C7: payments are folded into invoices as one billing area, with no behavior change.
- C8: booking granularity changes from 30 to 15 minutes, and opening hours change.
- C9: unchanged re-application.

Non-goals: real organizations or brands, external integrations, persistence across runs.
