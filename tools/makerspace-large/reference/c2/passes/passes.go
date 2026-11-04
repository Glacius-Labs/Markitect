// Package passes holds the pass products and the passes members buy, and the
// operations on them.
package passes

import (
	"sort"

	"makerspace/billing"
	"makerspace/members"
	"makerspace/policies"
)

// Product is a block of machine minutes the workshop sells.
type Product struct {
	ID, Name                 string
	Minutes, Price, ValidDay int
}

// Pass is a pass of one member. IssuedOn and ExpiresOn are days.
type Pass struct {
	ID, MemberID, ProductID string
	MinutesLeft             int
	IssuedOn, ExpiresOn     int
}

// Module is the passes module: it owns every product and pass.
type Module struct {
	members  *members.Module
	billing  *billing.Module
	ids      policies.IDs
	products map[string]*Product
	list     []*Pass
}

// New returns the module in the initial state.
func New(mem *members.Module, bill *billing.Module) *Module {
	return &Module{members: mem, billing: bill, products: map[string]*Product{}}
}

// ProductExists reports whether the identifier names a pass product.
func (m *Module) ProductExists(id string) bool { _, ok := m.products[id]; return ok }

// Draw draws minutes from the passes of the member that are valid on the date
// of the time at, as the pass draw rule states, lowers their minutesLeft and
// returns the minutes drawn, never more than the minutes asked for.
func (m *Module) Draw(memberID string, minutes, at int) int {
	day := policies.DayOf(at)
	var valid []*Pass
	for _, p := range m.list {
		if p.MemberID == memberID && p.MinutesLeft > 0 && day < p.ExpiresOn {
			valid = append(valid, p)
		}
	}
	sort.SliceStable(valid, func(i, j int) bool { return valid[i].ExpiresOn < valid[j].ExpiresOn })
	drawn := 0
	for _, p := range valid {
		if drawn == minutes {
			break
		}
		take := min(p.MinutesLeft, minutes-drawn)
		p.MinutesLeft -= take
		drawn += take
	}
	return drawn
}

func productView(p *Product) policies.Rec {
	return policies.Rec{"productId": p.ID, "name": p.Name, "minutes": p.Minutes, "priceCents": p.Price,
		"validDays": p.ValidDay}
}

func passView(p *Pass) policies.Rec {
	return policies.Rec{"passId": p.ID, "memberId": p.MemberID, "productId": p.ProductID,
		"minutesLeft": p.MinutesLeft, "issuedOn": policies.FormatDay(p.IssuedOn),
		"expiresOn": policies.FormatDay(p.ExpiresOn)}
}
