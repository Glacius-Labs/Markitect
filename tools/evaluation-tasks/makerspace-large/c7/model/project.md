---
kind: std/Project
---
# Makerspace

Makerspace runs the day-to-day operations of a fictional community workshop. It is one local program. Front-desk tools send it requests on behalf of the person who acts; the program decides, records and answers.

- **Members** join on a membership tier and pay a monthly fee that depends on the tier. Staff can suspend a member or, for a few months away, pause the membership at a small fixed fee; a suspended or paused member is not in good standing.
- **Training:** courses certify members for a tool family, such as laser cutters. A certification is valid for 365 days.
- **Equipment:** machines belong to a tool family and have a rate per minute. Staff schedule maintenance windows during which a machine cannot be reserved.
- **Bookings:** members reserve machines in 15-minute steps inside opening hours, check in when they start and check out when they finish. They are charged for every metered minute that no pack covers, at the machine's rate, minus a tier discount. A member who books a machine and does not come pays half the price of the time blocked, through a run that staff start by hand, unless the machine was locked out during the slot.
- **Materials:** a small shop sells materials from stock.
- **Billing:** once a month staff generate invoices: one per member, with the monthly fee and all charges of that month. An invoice is due 14 days after it is issued. Members pay their invoices; an overpayment becomes credit, and credit reduces the next invoice.
- **Safety:** staff record safety incidents per machine.
- **Reporting:** staff and admins read utilization and revenue reports.
- **Ledger:** every movement of money is posted to a double-entry journal, which admins read with its trial balance.
- **Notices:** the program keeps notices for members, who read and acknowledge them.
- **Households:** members can form a household with one payer; a household of three or more pays less for the monthly fee.
- **Guardians and waivers:** a minor needs a guardian's consent for a tool family, and a family can require a signed waiver of its current version, and a signature counts for a year.
- **Protective equipment:** a family can require items that a member must hold to check in.
- **Lockouts and inspections:** a machine can be locked out, and a machine with an inspection plan cannot be reserved for its due date or later.
- **Staffing:** staff work shifts, and a family can require a supervisor on shift for a whole reservation.
- **Parts, work orders and escalations:** staff repair machines with spare parts under work orders. An incident can be escalated, which locks the machine and cancels what is booked on it.
- **Rooms, instructors and classes:** courses can have classes in rooms, taught by qualified instructors. A course can require attendance and a passed assessment, and it can have a waiting list. Instructors are paid per month.
- **Events:** staff run ticketed events in rooms.
- **Deposits, packs and lockers:** members leave deposits, buy packs of prepaid machine minutes and rent lockers.
- **Promotions and scholarships:** a redeemed promotion or a scholarship award lowers a course or material charge.
- **Refunds and dunning:** admins refund invoice lines as credit, and an explicit run moves overdue invoices through notice, late fee and suspension.
- **Guests, access and volunteering:** members bring guests, and badges open the door. Approved volunteer time becomes credit.
- **Tool library:** staff lend portable power tools to members for 14 days, at the desk and only while someone is on duty. A late tool costs a fee per day, and a member who keeps one too long cannot check in to a machine until it is back. A member can renew a loan twice, for 14 more days each, as long as it is not late.
- **Consumables and hazardous materials:** a machine can draw material from stock while it runs, and hazardous materials need a certification and have a monthly limit.
- **Purchasing and stocktake:** staff order materials from suppliers and count stock.
- **Projects, quotes and jobs:** members attach their work to projects, and the workshop quotes and carries out jobs for outside customers.
- **Compliance and finance:** staff read whether a member may use a machine, and financial summaries.

Four policies apply to every operation: [[authorization]], [[audit]], [[money]] and [[time]].

Out of scope: persistence across runs, network access, user interfaces, concurrency, real payments or refunds, the sending of notifications (a notice is a record in the program), scheduled jobs, deletion of records and integration with other systems.

The Go module lives at `realization/go.mod`; the program's main package lives at `realization/cmd/makerspace`. Only the Go standard library is used. The program is divided into the modules of its architecture; each module is one Go package under `realization/`, named after the module, for example `realization/billing`; `realization/protocol` and `realization/cmd/makerspace` belong to the protocol module.

The program is written in Go. From `realization/`, `go build ./cmd/makerspace` builds it and `go test ./...` runs its tests.

The modules are described in the [[architecture]]: the area modules, each with its own folder here, the outer layer [[protocol]], and the shared policies module, which this folder describes: [[values]], [[refusal-codes]], roles and the project-wide rules.
