package billing

import (
	"sort"

	"makerspace/members"
	"makerspace/policies"
)

func (b *Module) runGenerateInvoices(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	idx := c.Num("month")
	next := policies.MonthStart(idx + 1)
	if c.At < next {
		return nil, "", policies.Refuse("month_not_ended", "The month has not ended yet.")
	}
	if b.invoiced[idx] {
		return nil, "", policies.Refuse("already_invoiced", "Invoices for the month have already been generated.")
	}
	out := []policies.Rec{}
	for _, mid := range b.members.SortedIDs() {
		m := b.members.Member(mid)
		if m.RegisteredAt >= next {
			continue
		}
		inv := &Invoice{ID: b.ids.Next("inv"), MemberID: mid, Month: idx, IssuedAt: c.At, DueAt: c.At + 14*policies.MinPerDay}
		fee := members.TierFee(m.Tier)
		discount := 0
		if _, group, ok := b.households.OfMember(mid); ok && len(group) >= 3 {
			discount = 10
		}
		inv.Lines = append(inv.Lines, Charge{MemberID: mid, Kind: "membership", Ref: m.Tier, Quantity: 1,
			Unit: fee, Discount: discount, Amount: policies.ChargeAmount(1, fee, discount), At: c.At})
		var due []*Charge
		for _, ch := range b.memberCharges(mid) {
			if !ch.Billed && policies.MonthOf(ch.At) == idx {
				due = append(due, ch)
			}
		}
		sort.SliceStable(due, func(a, b int) bool { return due[a].At < due[b].At })
		for _, ch := range due {
			ch.Billed = true
			inv.Lines = append(inv.Lines, *ch)
		}
		for _, l := range inv.Lines {
			inv.Total += l.Amount
		}
		inv.CreditApplied = min(b.credit[mid], inv.Total)
		b.credit[mid] -= inv.CreditApplied
		inv.Balance = inv.Total - inv.CreditApplied - inv.Paid
		b.invoices[inv.ID] = inv
		b.invoiceList = append(b.invoiceList, inv)
		for _, l := range inv.Lines {
			b.ledger.Post("receivable", revenueAccount[l.Kind], l.Amount, inv.ID, c.At)
		}
		b.ledger.Post("creditLiability", "receivable", inv.CreditApplied, inv.ID, c.At)
		out = append(out, invoiceView(inv, c.At))
	}
	b.invoiced[idx] = true
	return policies.Rec{"month": policies.FormatMonth(idx), "invoices": out}, policies.FormatMonth(idx), nil
}

// revenueAccount is the ledger account an invoice line of a kind is credited to.
var revenueAccount = map[string]string{
	"membership": "revenueMembership", "usage": "revenueUsage", "material": "revenueMaterial",
	"course": "revenueCourse", "deposit": "depositLiability", "pass": "revenuePasses",
	"locker": "revenueLockers", "guest": "revenueGuests", "lateFee": "revenueLateFees",
	"event": "revenueEvents",
}
