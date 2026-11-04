// Package payouts holds the hourly rates of instructors and the monthly
// payouts for the classes they taught, and the operations on them.
package payouts

import (
	"makerspace/classes"
	"makerspace/instructors"
	"makerspace/ledger"
	"makerspace/members"
	"makerspace/policies"
)

// Payout is the pay owed to one instructor for the classes of one month.
type Payout struct {
	ID, UserID      string
	Month           int
	Minutes, Amount int
	Settled         bool
}

// Status is open or settled.
func (p *Payout) Status() string {
	if p.Settled {
		return "settled"
	}
	return "open"
}

// Module is the payouts module: it owns every rate and payout.
type Module struct {
	members     *members.Module
	ledger      *ledger.Module
	instructors *instructors.Module
	classes     *classes.Module
	ids         policies.IDs
	rates       map[string]int
	generated   map[int]bool
	byID        map[string]*Payout
	list        []*Payout
}

// New returns the module in the initial state.
func New(mem *members.Module, led *ledger.Module, inst *instructors.Module, cl *classes.Module) *Module {
	return &Module{members: mem, ledger: led, instructors: inst, classes: cl, rates: map[string]int{},
		generated: map[int]bool{}, byID: map[string]*Payout{}}
}

// PayoutExists reports whether the identifier names a payout.
func (m *Module) PayoutExists(id string) bool { _, ok := m.byID[id]; return ok }

func payoutView(p *Payout) policies.Rec {
	return policies.Rec{"payoutId": p.ID, "userId": p.UserID, "month": policies.FormatMonth(p.Month),
		"minutes": p.Minutes, "amountCents": p.Amount, "status": p.Status()}
}
