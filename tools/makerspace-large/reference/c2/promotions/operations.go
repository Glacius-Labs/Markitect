package promotions

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addPromotion", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.ID("code"), policies.Integer("percent", 1, 100),
				policies.Enum("scope", "course", "material", "both"), policies.Integer("maxRedemptions", 1, 100000),
				policies.Date("validUntil")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				code := c.Str("code")
				if m.byCode[code] != nil {
					return nil, "", policies.Refuse("already_exists", "A promotion with code %s already exists.", code)
				}
				p := &Promotion{Code: code, Percent: c.Num("percent"), Scope: c.Str("scope"),
					MaxRedemptions: c.Num("maxRedemptions"), ValidUntil: c.Num("validUntil")}
				m.byCode[code] = p
				return policies.Rec{"promotion": promotionView(p)}, code, nil
			}},
		{Name: "redeemPromotion", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.IDRef("code", m.PromotionExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, p := c.Str("memberId"), m.byCode[c.Str("code")]
				if !m.members.InGoodStanding(mid) {
					return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
				}
				if p.Ended {
					return nil, "", policies.Refuse("promotion_ended", "The promotion has been ended.")
				}
				if policies.DayOf(c.At) > p.ValidUntil {
					return nil, "", policies.Refuse("promotion_expired", "The promotion is no longer valid.")
				}
				if p.Redemptions >= p.MaxRedemptions {
					return nil, "", policies.Refuse("promotion_exhausted", "The promotion has been redeemed as often as it allows.")
				}
				for _, r := range m.list {
					if r.MemberID == mid && r.Code == p.Code {
						return nil, "", policies.Refuse("already_redeemed", "The member has already redeemed the promotion.")
					}
				}
				r := &Redemption{Code: p.Code, MemberID: mid, Scope: p.Scope, Percent: p.Percent, RedeemedAt: c.At}
				m.list = append(m.list, r)
				p.Redemptions++
				return policies.Rec{"redemption": redemptionView(r), "promotion": promotionView(p)}, mid + "/" + p.Code, nil
			}},
		{Name: "listMemberPromotions", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, r := range m.list {
					if r.MemberID == mid {
						out = append(out, redemptionView(r))
					}
				}
				return policies.Rec{"memberId": mid, "redemptions": out}, "", nil
			}},
		{Name: "endPromotion", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("code", m.PromotionExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				p := m.byCode[c.Str("code")]
				if p.Ended {
					return nil, "", policies.Refuse("invalid_state", "The promotion is already ended.")
				}
				p.Ended = true
				return policies.Rec{"promotion": promotionView(p)}, p.Code, nil
			}},
	}
}
