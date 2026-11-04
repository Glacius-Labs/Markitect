package guests

import (
	"makerspace/members"
	"makerspace/policies"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "setGuestAllowance", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.Enum("tier", members.Tiers()...), policies.Integer("perMonth", 0, 100),
				policies.Integer("feeCents", 0, 1000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				a := &Allowance{Tier: c.Str("tier"), PerMonth: c.Num("perMonth"), FeeCts: c.Num("feeCents")}
				m.allowances[a.Tier] = a
				return policies.Rec{"allowance": allowanceView(a)}, a.Tier, nil
			}},
		{Name: "issueGuestPass", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.Text("guestName"), policies.Date("date")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, date := c.Str("memberId"), c.Num("date")
				if !m.members.InGoodStanding(mid) {
					return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
				}
				if date < policies.DayOf(c.At) {
					return nil, "", policies.Refuse("date_in_past", "The date is before the date of the request time.")
				}
				al := m.allowances[m.members.Member(mid).Tier]
				if al == nil {
					return nil, "", policies.Refuse("guests_not_allowed", "The member's tier has no guest allowance.")
				}
				month, counted := policies.MonthOf(date*policies.MinPerDay), 0
				for _, g := range m.list {
					if g.MemberID == mid && !g.Revoked && policies.MonthOf(g.Date*policies.MinPerDay) == month {
						counted++
					}
				}
				g := &GuestPass{ID: m.ids.Next("gst"), MemberID: mid, GuestName: c.Str("guestName"), Date: date,
					Charged: counted >= al.PerMonth}
				m.byID[g.ID] = g
				m.list = append(m.list, g)
				if g.Charged {
					m.billing.AddCharge(mid, "guest", g.ID, 1, al.FeeCts, 0, c.At)
				}
				return policies.Rec{"guestPass": guestPassView(g)}, g.ID, nil
			}},
		{Name: "revokeGuestPass", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("guestPassId", m.GuestPassExists, m.GuestPassOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				g := m.byID[c.Str("guestPassId")]
				if g.Revoked || g.Date < policies.DayOf(c.At) {
					return nil, "", policies.Refuse("invalid_state", "The guest pass is revoked or its date has passed.")
				}
				g.Revoked = true
				return policies.Rec{"guestPass": guestPassView(g)}, g.ID, nil
			}},
		{Name: "listGuestPasses", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, g := range m.list {
					if g.MemberID == mid {
						out = append(out, guestPassView(g))
					}
				}
				return policies.Rec{"memberId": mid, "guestPasses": out}, "", nil
			}},
	}
}
