package billing

import (
	"sort"

	"makerspace/policies"
)

// Operations lists the operations of the module.
func (b *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "generateInvoices", Mutation: true, Perms: "naan",
			Args: []policies.ArgSpec{policies.Month("month")},
			Run:  b.runGenerateInvoices},
		{Name: "listInvoices", Perms: "oaan",
			Args: []policies.ArgSpec{policies.MemberRef(b.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				var list []*Invoice
				for _, i := range b.invoiceList {
					if i.MemberID == mid {
						list = append(list, i)
					}
				}
				sort.SliceStable(list, func(a, b int) bool { return list[a].Month < list[b].Month })
				out := make([]policies.Rec, 0, len(list))
				for _, i := range list {
					out = append(out, invoiceView(i, c.At))
				}
				return policies.Rec{"memberId": mid, "invoices": out}, "", nil
			}},
		{Name: "recordPayment", Mutation: true, Perms: "oaan",
			Args: []policies.ArgSpec{policies.OwnedIDRef("invoiceId", b.InvoiceExists, b.InvoiceOwner), policies.Integer("amountCents", 1, 100000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				inv := b.invoices[c.Str("invoiceId")]
				if inv.Status() == "paid" {
					return nil, "", policies.Refuse("invoice_paid", "The invoice is already paid.")
				}
				amount := c.Num("amountCents")
				applied := min(amount, inv.Balance)
				p := &Payment{ID: b.ids.Next("pay"), InvoiceID: inv.ID, MemberID: inv.MemberID,
					Amount: amount, Applied: applied, Credit: amount - applied, At: c.At}
				inv.Paid += applied
				inv.Balance -= applied
				b.credit[inv.MemberID] += p.Credit
				b.payments = append(b.payments, p)
				return policies.Rec{"payment": paymentView(p), "invoice": invoiceView(inv, c.At),
					"creditBalanceCents": b.credit[inv.MemberID]}, p.ID, nil
			}},
		{Name: "issueCredit", Mutation: true, Perms: "nnan",
			Args: []policies.ArgSpec{policies.MemberRef(b.members.Exists), policies.Integer("amountCents", 1, 100000000), policies.Text("reason")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				b.GrantCredit(mid, c.Num("amountCents"), c.At)
				return policies.Rec{"memberId": mid, "amountCents": c.Num("amountCents"),
					"creditBalanceCents": b.credit[mid]}, mid, nil
			}},
		{Name: "getAccount", Perms: "oaan",
			Args: []policies.ArgSpec{policies.MemberRef(b.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				open, overdue, unbilled := 0, 0, 0
				for _, i := range b.invoiceList {
					if i.MemberID != mid {
						continue
					}
					if i.Status() == "open" {
						open += i.Balance
					}
					if i.Overdue(c.At) {
						overdue += i.Balance
					}
				}
				for _, ch := range b.memberCharges(mid) {
					if !ch.Billed {
						unbilled += ch.Amount
					}
				}
				return policies.Rec{"account": policies.Rec{"memberId": mid, "creditBalanceCents": b.credit[mid],
					"openBalanceCents": open, "overdueBalanceCents": overdue, "unbilledCents": unbilled,
					"blocked": b.Blocked(mid, c.At)}}, "", nil
			}},
	}
}
