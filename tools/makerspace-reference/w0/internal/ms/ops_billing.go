package ms

import "sort"

func billingOps() []*opDef {
	return []*opDef{
		{name: "generateInvoices", mutation: true, perms: "naa",
			args: []argSpec{month("month")},
			run:  runGenerateInvoices},
		{name: "listInvoices", perms: "oaa",
			args: []argSpec{idRef("memberId", refMember)},
			run: func(c *call) (rec, string, *refusal) {
				mid := c.str("memberId")
				var list []*invoice
				for _, i := range c.s.invoiceList {
					if i.memberID == mid {
						list = append(list, i)
					}
				}
				sort.SliceStable(list, func(a, b int) bool { return list[a].month < list[b].month })
				out := make([]rec, 0, len(list))
				for _, i := range list {
					out = append(out, invoiceView(i, c.at))
				}
				return rec{"memberId": mid, "invoices": out}, "", nil
			}},
		{name: "recordPayment", mutation: true, perms: "oaa",
			args: []argSpec{idRef("invoiceId", refInvoice), integer("amountCents", 1, 100000000)},
			run: func(c *call) (rec, string, *refusal) {
				inv := c.s.invoices[c.str("invoiceId")]
				if inv.status() == "paid" {
					return nil, "", refuse("invoice_paid", "The invoice is already paid.")
				}
				amount := c.num("amountCents")
				applied := min(amount, inv.balance)
				p := &payment{id: c.s.nextID("pay"), invoiceID: inv.id, memberID: inv.memberID,
					amount: amount, applied: applied, credit: amount - applied, at: c.at}
				inv.paid += applied
				inv.balance -= applied
				m := c.s.members[inv.memberID]
				m.credit += p.credit
				c.s.payments = append(c.s.payments, p)
				return rec{"payment": paymentView(p), "invoice": invoiceView(inv, c.at),
					"creditBalanceCents": m.credit}, p.id, nil
			}},
		{name: "issueCredit", mutation: true, perms: "nna",
			args: []argSpec{idRef("memberId", refMember), integer("amountCents", 1, 100000000), text("reason")},
			run: func(c *call) (rec, string, *refusal) {
				m := c.member("memberId")
				m.credit += c.num("amountCents")
				c.s.creditEvents = append(c.s.creditEvents, creditEvent{at: c.at, amount: c.num("amountCents")})
				return rec{"memberId": m.user.id, "amountCents": c.num("amountCents"),
					"creditBalanceCents": m.credit}, m.user.id, nil
			}},
		{name: "getAccount", perms: "oaa",
			args: []argSpec{idRef("memberId", refMember)},
			run: func(c *call) (rec, string, *refusal) {
				m := c.member("memberId")
				open, overdue, unbilled := 0, 0, 0
				for _, i := range c.s.invoiceList {
					if i.memberID != m.user.id {
						continue
					}
					if i.status() == "open" {
						open += i.balance
					}
					if i.overdue(c.at) {
						overdue += i.balance
					}
				}
				for _, ch := range c.s.memberCharges(m.user.id) {
					if !ch.billed {
						unbilled += ch.amount
					}
				}
				return rec{"account": rec{"memberId": m.user.id, "creditBalanceCents": m.credit,
					"openBalanceCents": open, "overdueBalanceCents": overdue, "unbilledCents": unbilled}}, "", nil
			}},
	}
}

func runGenerateInvoices(c *call) (rec, string, *refusal) {
	idx := c.num("month")
	next := monthStart(idx + 1)
	if c.at < next {
		return nil, "", refuse("month_not_ended", "The month has not ended yet.")
	}
	if c.s.invoiced[idx] {
		return nil, "", refuse("already_invoiced", "Invoices for the month have already been generated.")
	}
	out := []rec{}
	for _, mid := range sortedKeys(c.s.members) {
		m := c.s.members[mid]
		if m.registeredAt >= next {
			continue
		}
		inv := &invoice{id: c.s.nextID("inv"), memberID: mid, month: idx, issuedAt: c.at, dueAt: c.at + 14*minPerDay}
		fee := tierFee(m.tier)
		inv.lines = append(inv.lines, charge{memberID: mid, kind: "membership", ref: m.tier, quantity: 1,
			unit: fee, amount: chargeAmount(1, fee, 0), at: c.at})
		var due []*charge
		for _, ch := range c.s.memberCharges(mid) {
			if !ch.billed && monthOf(ch.at) == idx {
				due = append(due, ch)
			}
		}
		sort.SliceStable(due, func(a, b int) bool { return due[a].at < due[b].at })
		for _, ch := range due {
			ch.billed = true
			inv.lines = append(inv.lines, *ch)
		}
		for _, l := range inv.lines {
			inv.total += l.amount
		}
		inv.creditApplied = min(m.credit, inv.total)
		m.credit -= inv.creditApplied
		inv.balance = inv.total - inv.creditApplied - inv.paid
		c.s.invoices[inv.id] = inv
		c.s.invoiceList = append(c.s.invoiceList, inv)
		out = append(out, invoiceView(inv, c.at))
	}
	c.s.invoiced[idx] = true
	return rec{"month": formatMonth(idx), "invoices": out}, formatMonth(idx), nil
}
