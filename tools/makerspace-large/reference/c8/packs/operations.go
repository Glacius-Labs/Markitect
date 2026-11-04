package packs

import (
	"makerspace/billing"
	"makerspace/policies"
)

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addPackProduct", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.ID("productId"), policies.Text("name"), policies.Integer("minutes", 1, 100000),
				policies.Integer("priceCents", 1, 100000000), policies.Integer("validDays", 1, 3650)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("productId")
				if m.products[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A pack product with identifier %s already exists.", id)
				}
				p := &Product{ID: id, Name: c.Str("name"), Minutes: c.Num("minutes"), Price: c.Num("priceCents"),
					ValidDay: c.Num("validDays")}
				m.products[id] = p
				return policies.Rec{"product": productView(p)}, id, nil
			}},
		{Name: "sellPack", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists), policies.IDRef("productId", m.ProductExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				if !m.members.InGoodStanding(mid) {
					return nil, "", policies.Refuse("member_suspended", "The member is not in good standing.")
				}
				prod := m.products[c.Str("productId")]
				today := policies.DayOf(c.At)
				p := &Pack{ID: m.ids.Next("pak"), MemberID: mid, ProductID: prod.ID, MinutesLeft: prod.Minutes,
					IssuedOn: today, ExpiresOn: today + prod.ValidDay}
				m.list = append(m.list, p)
				ch := m.billing.AddCharge(mid, "pack", p.ID, 1, prod.Price, 0, c.At)
				return policies.Rec{"pack": packView(p), "charge": billing.ChargeView(ch)}, p.ID, nil
			}},
		{Name: "listMemberPacks", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				out := []policies.Rec{}
				for _, p := range m.list {
					if p.MemberID == mid {
						out = append(out, packView(p))
					}
				}
				return policies.Rec{"memberId": mid, "packs": out}, "", nil
			}},
	}
}
