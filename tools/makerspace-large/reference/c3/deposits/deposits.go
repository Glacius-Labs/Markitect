// Package deposits holds the deposits members leave with the workshop, and the
// operations on them.
package deposits

import (
	"makerspace/billing"
	"makerspace/ledger"
	"makerspace/members"
	"makerspace/policies"
)

// Deposit is a sum a member leaves with the workshop.
type Deposit struct {
	ID, MemberID, Purpose, Status string
	Amount, TakenAt               int
	ClosedAt                      *int
}

// Module is the deposits module: it owns every deposit.
type Module struct {
	members *members.Module
	billing *billing.Module
	ledger  *ledger.Module
	ids     policies.IDs
	byID    map[string]*Deposit
	list    []*Deposit
}

// New returns the module in the initial state.
func New(mem *members.Module, bill *billing.Module, led *ledger.Module) *Module {
	return &Module{members: mem, billing: bill, ledger: led, byID: map[string]*Deposit{}}
}

// DepositExists reports whether the identifier names a deposit.
func (m *Module) DepositExists(id string) bool { _, ok := m.byID[id]; return ok }

// DepositOwner is the member a deposit belongs to.
func (m *Module) DepositOwner(id string) string { return m.byID[id].MemberID }

func depositView(d *Deposit) policies.Rec {
	return policies.Rec{"depositId": d.ID, "memberId": d.MemberID, "purpose": d.Purpose, "amountCents": d.Amount,
		"status": d.Status, "takenAt": policies.FormatTime(d.TakenAt), "closedAt": policies.OptTime(d.ClosedAt)}
}
