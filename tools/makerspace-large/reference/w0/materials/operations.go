package materials

import "makerspace/policies"

// Operations lists the operations of the module.
func (s *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addMaterial", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.ID("materialId"), policies.Text("name"), policies.Integer("unitPriceCents", 1, 1000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("materialId")
				if s.materials[mid] != nil {
					return nil, "", policies.Refuse("already_exists", "A material with identifier %s already exists.", mid)
				}
				m := &material{id: mid, name: c.Str("name"), unit: c.Num("unitPriceCents")}
				s.materials[mid] = m
				return policies.Rec{"material": materialView(m)}, mid, nil
			}},
		{Name: "restockMaterial", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("materialId", s.MaterialExists), policies.Integer("quantity", 1, 100000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				m := s.materials[c.Str("materialId")]
				if s.needs.Counted(m.id) {
					return nil, "", policies.Refuse("stocktake_open", "The material has a count in an open stocktake.")
				}
				m.stock += c.Num("quantity")
				return policies.Rec{"material": materialView(m)}, m.id, nil
			}},
		{Name: "purchaseMaterial", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(s.members.Exists), policies.IDRef("materialId", s.MaterialExists),
				policies.Integer("quantity", 1, 10000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mem := s.members.Member(c.Str("memberId"))
				mat := s.materials[c.Str("materialId")]
				qty := c.Num("quantity")
				if mem.Standing != "good" {
					return nil, "", policies.Refuse("member_suspended", "The member is suspended.")
				}
				if code := s.needs.HazardRefusal(mem.User.ID, mat.id, qty, c.At); code != "" {
					return nil, "", policies.Refuse(code, "The hazard gate refuses the purchase.")
				}
				if s.needs.Counted(mat.id) {
					return nil, "", policies.Refuse("stocktake_open", "The material has a count in an open stocktake.")
				}
				if mat.stock < qty {
					return nil, "", policies.Refuse("insufficient_stock", "The stock is lower than the quantity.")
				}
				mat.stock -= qty
				discount := 0
				red := s.promotions.OldestUnused(mem.User.ID, "material")
				if red != nil {
					discount = red.Percent
				}
				p := &purchase{id: s.ids.Next("pur"), memberID: mem.User.ID, materialID: mat.id,
					quantity: qty, unit: mat.unit, discount: discount, at: c.At}
				p.amount = policies.ChargeAmount(p.quantity, p.unit, p.discount)
				if red != nil {
					s.promotions.Use(red, p.id, c.At)
				}
				s.purchases = append(s.purchases, p)
				s.byID[p.id] = p
				s.billing.AddCharge(mem.User.ID, "material", p.id, p.quantity, p.unit, p.discount, c.At)
				return policies.Rec{"purchase": purchaseView(p), "material": materialView(mat)}, p.id, nil
			}},
		{Name: "listMaterials", Perms: "aaa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				out := []policies.Rec{}
				for _, k := range policies.SortedKeys(s.materials) {
					out = append(out, materialView(s.materials[k]))
				}
				return policies.Rec{"materials": out}, "", nil
			}},
	}
}
