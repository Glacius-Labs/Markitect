// Package guests holds the guest allowance of each tier and the guest passes
// members issue, and the operations on them.
package guests

import (
	"makerspace/billing"
	"makerspace/members"
	"makerspace/policies"
)

// Allowance is how many guest passes a month a tier includes, and the fee of
// each further one.
type Allowance struct {
	Tier             string
	PerMonth, FeeCts int
}

// GuestPass is the permission for one guest to visit with a member on a date.
type GuestPass struct {
	ID, MemberID, GuestName string
	Date                    int
	Charged, Revoked        bool
}

// Status is issued or revoked.
func (g *GuestPass) Status() string {
	if g.Revoked {
		return "revoked"
	}
	return "issued"
}

// Module is the guests module: it owns every allowance and guest pass.
type Module struct {
	members    *members.Module
	billing    *billing.Module
	ids        policies.IDs
	allowances map[string]*Allowance
	byID       map[string]*GuestPass
	list       []*GuestPass
}

// New returns the module in the initial state: no tier has an allowance.
func New(mem *members.Module, bill *billing.Module) *Module {
	return &Module{members: mem, billing: bill, allowances: map[string]*Allowance{}, byID: map[string]*GuestPass{}}
}

// GuestPassExists reports whether the identifier names a guest pass.
func (m *Module) GuestPassExists(id string) bool { _, ok := m.byID[id]; return ok }

// GuestPassOwner is the member a guest pass belongs to.
func (m *Module) GuestPassOwner(id string) string { return m.byID[id].MemberID }

// GuestPassOf gives the member, the date (as a day) and the status (issued or
// revoked) of a guest pass. The last result is false when there is none.
func (m *Module) GuestPassOf(id string) (memberID string, date int, status string, ok bool) {
	g := m.byID[id]
	if g == nil {
		return "", 0, "", false
	}
	return g.MemberID, g.Date, g.Status(), true
}

func allowanceView(a *Allowance) policies.Rec {
	return policies.Rec{"tier": a.Tier, "perMonth": a.PerMonth, "feeCents": a.FeeCts}
}

func guestPassView(g *GuestPass) policies.Rec {
	return policies.Rec{"guestPassId": g.ID, "memberId": g.MemberID, "guestName": g.GuestName,
		"date": policies.FormatDay(g.Date), "charged": g.Charged, "status": g.Status()}
}
