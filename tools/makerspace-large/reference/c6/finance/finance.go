// Package finance holds the reads that report the money of a month, a date or a
// household from billing, the ledger and the refunds. It changes nothing.
package finance

import (
	"makerspace/billing"
	"makerspace/households"
	"makerspace/ledger"
	"makerspace/members"
	"makerspace/refunds"
)

// kinds lists the charge kinds in the order of the reports.
var kinds = []string{"membership", "usage", "material", "course", "deposit", "pass", "locker", "guest",
	"lateFee", "event", "noShow"}

// Module is the finance module: it owns no record.
type Module struct {
	members    *members.Module
	ledger     *ledger.Module
	billing    *billing.Module
	households *households.Module
	refunds    *refunds.Module
}

// New returns the module.
func New(mem *members.Module, led *ledger.Module, bill *billing.Module, hh *households.Module,
	rfd *refunds.Module) *Module {
	return &Module{members: mem, ledger: led, billing: bill, households: hh, refunds: rfd}
}

// refundedOn is the sum of the refunds of all lines of the invoice.
func (m *Module) refundedOn(inv *billing.Invoice) int {
	n := 0
	for i := range inv.Lines {
		n += m.refunds.RefundedOnLine(inv.ID, i+1)
	}
	return n
}
