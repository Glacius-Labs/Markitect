package billing

import "makerspace/policies"

// ChargeView is the charge as it appears in results.
func ChargeView(c *Charge) policies.Rec {
	return policies.Rec{"kind": c.Kind, "ref": c.Ref, "quantity": c.Quantity, "unitPriceCents": c.Unit,
		"discountPercent": c.Discount, "amountCents": c.Amount, "at": policies.FormatTime(c.At)}
}

func invoiceView(i *Invoice, at int) policies.Rec {
	lines := make([]policies.Rec, 0, len(i.Lines))
	for k := range i.Lines {
		lines = append(lines, ChargeView(&i.Lines[k]))
	}
	return policies.Rec{"invoiceId": i.ID, "memberId": i.MemberID, "month": policies.FormatMonth(i.Month),
		"issuedAt": policies.FormatTime(i.IssuedAt), "dueAt": policies.FormatTime(i.DueAt), "lines": lines,
		"totalCents": i.Total, "creditAppliedCents": i.CreditApplied, "paidCents": i.Paid,
		"balanceCents": i.Balance, "status": i.Status(), "overdue": i.Overdue(at)}
}

func paymentView(p *Payment) policies.Rec {
	return policies.Rec{"paymentId": p.ID, "invoiceId": p.InvoiceID, "memberId": p.MemberID,
		"amountCents": p.Amount, "appliedCents": p.Applied, "creditedCents": p.Credit,
		"at": policies.FormatTime(p.At)}
}
