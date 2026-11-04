// Package scholarships holds the funds and the awards that give members a
// discount on course fees, and the operations on them.
package scholarships

import (
	"makerspace/members"
	"makerspace/policies"
)

// Fund is a budget from which the discounts of awards are paid.
type Fund struct {
	ID            string
	Budget, Spent int
}

// Award is a scholarship of one member from one fund.
type Award struct {
	ID, FundID, MemberID string
	Percent              int
	Revoked              bool
}

// Status is active or revoked.
func (a *Award) Status() string {
	if a.Revoked {
		return "revoked"
	}
	return "active"
}

// Module is the scholarships module: it owns every fund and award.
type Module struct {
	members *members.Module
	ids     policies.IDs
	funds   map[string]*Fund
	list    []*Award
}

// New returns the module in the initial state.
func New(mem *members.Module) *Module {
	return &Module{members: mem, funds: map[string]*Fund{}}
}

// FundExists reports whether the identifier names a fund.
func (m *Module) FundExists(id string) bool { _, ok := m.funds[id]; return ok }

func (m *Module) awardExists(id string) bool {
	for _, a := range m.list {
		if a.ID == id {
			return true
		}
	}
	return false
}

func (m *Module) award(id string) *Award {
	for _, a := range m.list {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// ActiveAward is the active award of the member and the remaining budget of
// its fund (budget less spent). The last result is false when the member has
// no active award.
func (m *Module) ActiveAward(memberID string) (award *Award, remaining int, ok bool) {
	for _, a := range m.list {
		if a.MemberID == memberID && !a.Revoked {
			f := m.funds[a.FundID]
			return a, f.Budget - f.Spent, true
		}
	}
	return nil, 0, false
}

// Spend adds the amount to the spent amount of the fund.
func (m *Module) Spend(fundID string, amount int) { m.funds[fundID].Spent += amount }

func fundView(f *Fund) policies.Rec {
	return policies.Rec{"fundId": f.ID, "budgetCents": f.Budget, "spentCents": f.Spent}
}

func awardView(a *Award) policies.Rec {
	return policies.Rec{"awardId": a.ID, "fundId": a.FundID, "memberId": a.MemberID, "percent": a.Percent,
		"status": a.Status()}
}
