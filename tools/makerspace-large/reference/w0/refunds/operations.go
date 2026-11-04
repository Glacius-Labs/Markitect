package refunds

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	invoiceArg := policies.OwnedIDRef("invoiceId", func(id string) bool { return m.billing.Invoice(id) != nil },
		func(id string) string { return m.billing.Invoice(id).MemberID })
	return []*policies.Operation{
		{Name: "issueRefund", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{invoiceArg, policies.Integer("lineNumber", 1, 1000),
				policies.Integer("amountCents", 1, 100000000), policies.Text("reason")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				inv, line, amount := m.billing.Invoice(c.Str("invoiceId")), c.Num("lineNumber"), c.Num("amountCents")
				if inv.Status() != "paid" {
					return nil, "", policies.Refuse("invoice_not_paid", "The invoice is not paid.")
				}
				if line > len(inv.Lines) {
					return nil, "", policies.Refuse("invalid_line", "The invoice has no line %d.", line)
				}
				if amount > inv.Lines[line-1].Amount-m.RefundedOnLine(inv.ID, line) {
					return nil, "", policies.Refuse("refund_too_large", "The amount is more than is left of the line.")
				}
				r := &Refund{ID: m.ids.Next("rfd"), InvoiceID: inv.ID, Line: line, Amount: amount,
					Reason: c.Str("reason"), At: c.At}
				m.list = append(m.list, r)
				m.billing.AddCredit(inv.MemberID, amount, "revenueRefunds", r.ID, c.At)
				return policies.Rec{"refund": refundView(r),
					"creditBalanceCents": m.billing.CreditBalance(inv.MemberID)}, r.ID, nil
			}},
		{Name: "listRefunds", Perms: "oaa",
			Args: []policies.ArgSpec{invoiceArg},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("invoiceId")
				out := []policies.Rec{}
				for _, r := range m.list {
					if r.InvoiceID == id {
						out = append(out, refundView(r))
					}
				}
				return policies.Rec{"invoiceId": id, "refunds": out}, "", nil
			}},
	}
}
