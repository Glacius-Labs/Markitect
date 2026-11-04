package finance

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "chargesByKind", Perms: "naa",
			Args: []policies.ArgSpec{policies.Month("month")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				invoiced, refunded := map[string]int{}, map[string]int{}
				for _, inv := range m.billing.InvoicesOfMonth(c.Num("month")) {
					for i, l := range inv.Lines {
						invoiced[l.Kind] += l.Amount
						refunded[l.Kind] += m.refunds.RefundedOnLine(inv.ID, i+1)
					}
				}
				rows, total := []policies.Rec{}, 0
				for _, k := range kinds {
					rows = append(rows, policies.Rec{"kind": k, "invoicedCents": invoiced[k], "refundedCents": refunded[k]})
					total += invoiced[k]
				}
				return policies.Rec{"month": c.Str("month"), "rows": rows, "totalCents": total}, "", nil
			}},
		{Name: "financeSummary", Perms: "nna",
			Args: []policies.ArgSpec{policies.Month("month")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				month := c.Num("month")
				invoiced, refunded := 0, 0
				for _, inv := range m.billing.InvoicesOfMonth(month) {
					for _, l := range inv.Lines {
						invoiced += l.Amount
					}
					refunded += m.refundedOn(inv)
				}
				return policies.Rec{"month": c.Str("month"), "invoicedCents": invoiced, "refundedCents": refunded,
					"paymentsCents": m.billing.PaymentsInMonth(month), "creditsIssuedCents": m.billing.CreditsIssuedInMonth(month),
					"journalEntries": m.ledger.EntriesInMonth(month), "trialBalanced": m.ledger.Balanced()}, "", nil
			}},
		{Name: "agedReceivables", Perms: "naa",
			Args: []policies.ArgSpec{policies.Date("date")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				day := c.Num("date")
				rows := []policies.Rec{}
				var current, to30, to60, over60 int
				for _, inv := range m.billing.OpenInvoices() {
					days := max(0, day-policies.DayOf(inv.DueAt))
					switch {
					case days == 0:
						current += inv.Balance
					case days <= 30:
						to30 += inv.Balance
					case days <= 60:
						to60 += inv.Balance
					default:
						over60 += inv.Balance
					}
					rows = append(rows, policies.Rec{"invoiceId": inv.ID, "memberId": inv.MemberID,
						"balanceCents": inv.Balance, "daysOverdue": days})
				}
				return policies.Rec{"date": c.Str("date"), "rows": rows, "currentCents": current,
					"days1to30Cents": to30, "days31to60Cents": to60, "over60Cents": over60}, "", nil
			}},
		{Name: "householdStatement", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("householdId", m.households.HouseholdExists), policies.Month("month")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				rows, total, balance := []policies.Rec{}, 0, 0
				for _, mid := range m.households.MemberIDsOf(c.Str("householdId")) {
					var invoiceID any
					t, b := 0, 0
					if inv := m.billing.InvoiceOfMonth(mid, c.Num("month")); inv != nil {
						invoiceID, t, b = inv.ID, inv.Total, inv.Balance
					}
					total, balance = total+t, balance+b
					rows = append(rows, policies.Rec{"memberId": mid, "invoiceId": invoiceID, "totalCents": t, "balanceCents": b})
				}
				return policies.Rec{"householdId": c.Str("householdId"), "month": c.Str("month"), "rows": rows,
					"totalCents": total, "balanceCents": balance}, "", nil
			}},
	}
}
