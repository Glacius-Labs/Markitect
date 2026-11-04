// Package refunds holds the refunds given on lines of paid invoices, and the
// operations on them.
package refunds

import (
	"makerspace/billing"
	"makerspace/ledger"
	"makerspace/members"
	"makerspace/policies"
)

// Refund is a refund of part or all of one line of a paid invoice.
type Refund struct {
	ID, InvoiceID, Reason string
	Line, Amount, At      int
}

// Module is the refunds module: it owns every refund.
type Module struct {
	members *members.Module
	billing *billing.Module
	ledger  *ledger.Module
	ids     policies.IDs
	list    []*Refund
}

// New returns the module in the initial state.
func New(mem *members.Module, bill *billing.Module, led *ledger.Module) *Module {
	return &Module{members: mem, billing: bill, ledger: led}
}

// RefundedOnLine is the sum of the refunds of the invoice line (1 is the first
// line), and 0 when there is none.
func (m *Module) RefundedOnLine(invoiceID string, line int) int {
	n := 0
	for _, r := range m.list {
		if r.InvoiceID == invoiceID && r.Line == line {
			n += r.Amount
		}
	}
	return n
}

func refundView(r *Refund) policies.Rec {
	return policies.Rec{"refundId": r.ID, "invoiceId": r.InvoiceID, "lineNumber": r.Line, "amountCents": r.Amount,
		"reason": r.Reason, "at": policies.FormatTime(r.At)}
}
