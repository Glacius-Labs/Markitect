// Package promotions holds the discount codes members redeem and the
// redemptions they make, and the operations on them.
package promotions

import (
	"makerspace/members"
	"makerspace/policies"
)

// Promotion is a discount code. ValidUntil is a day.
type Promotion struct {
	Code, Scope             string
	Percent, MaxRedemptions int
	ValidUntil, Redemptions int
	Ended                   bool
}

// Redemption is one member's redemption of one promotion.
type Redemption struct {
	Code, MemberID, Scope string
	Percent, RedeemedAt   int
	UsedAt                *int
	UsedOn                *string
}

// Module is the promotions module: it owns every promotion and redemption.
type Module struct {
	members *members.Module
	byCode  map[string]*Promotion
	list    []*Redemption
}

// New returns the module in the initial state.
func New(mem *members.Module) *Module {
	return &Module{members: mem, byCode: map[string]*Promotion{}}
}

// PromotionExists reports whether the identifier names a promotion.
func (m *Module) PromotionExists(code string) bool { _, ok := m.byCode[code]; return ok }

// OldestUnused is the oldest unused redemption of the member whose promotion
// covers the scope (course or material), or nil.
func (m *Module) OldestUnused(memberID, scope string) *Redemption {
	for _, r := range m.list {
		if r.MemberID == memberID && r.UsedAt == nil && (r.Scope == scope || r.Scope == "both") {
			return r
		}
	}
	return nil
}

// Use marks the redemption used at the time for the charge ref.
func (m *Module) Use(r *Redemption, ref string, at int) {
	r.UsedAt, r.UsedOn = &at, &ref
}

func promotionView(p *Promotion) policies.Rec {
	return policies.Rec{"code": p.Code, "percent": p.Percent, "scope": p.Scope, "maxRedemptions": p.MaxRedemptions,
		"validUntil": policies.FormatDay(p.ValidUntil), "redemptions": p.Redemptions, "ended": p.Ended}
}

func redemptionView(r *Redemption) policies.Rec {
	var usedOn any
	if r.UsedOn != nil {
		usedOn = *r.UsedOn
	}
	return policies.Rec{"code": r.Code, "memberId": r.MemberID, "percent": r.Percent,
		"redeemedAt": policies.FormatTime(r.RedeemedAt), "usedAt": policies.OptTime(r.UsedAt), "usedOn": usedOn}
}
