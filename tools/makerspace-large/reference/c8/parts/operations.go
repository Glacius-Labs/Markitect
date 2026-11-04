package parts

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addPart", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.ID("partId"), policies.Text("name"), policies.Integer("unitCostCents", 1, 1000000),
				policies.IDRef("familyId", m.equipment.FamilyExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("partId")
				if m.parts[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A part with identifier %s already exists.", id)
				}
				p := &Part{ID: id, Name: c.Str("name"), UnitCost: c.Num("unitCostCents"), FamilyID: c.Str("familyId")}
				m.parts[id] = p
				return policies.Rec{"part": partView(p)}, id, nil
			}},
		{Name: "restockPart", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("partId", m.PartExists), policies.Integer("quantity", 1, 100000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				p := m.parts[c.Str("partId")]
				p.Stock += c.Num("quantity")
				return policies.Rec{"part": partView(p)}, p.ID, nil
			}},
		{Name: "listParts", Perms: "naa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				out := []policies.Rec{}
				for _, k := range policies.SortedKeys(m.parts) {
					out = append(out, partView(m.parts[k]))
				}
				return policies.Rec{"parts": out}, "", nil
			}},
	}
}
