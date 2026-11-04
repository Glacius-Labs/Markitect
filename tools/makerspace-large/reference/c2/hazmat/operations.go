package hazmat

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "classifyHazard", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("materialId", m.materials.MaterialExists),
				policies.IDRef("familyId", m.equipment.FamilyExists), policies.Integer("monthlyLimit", 1, 10000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				cl := &Classification{MaterialID: c.Str("materialId"), FamilyID: c.Str("familyId"),
					MonthlyLimit: c.Num("monthlyLimit")}
				m.byMat[cl.MaterialID] = cl
				return policies.Rec{"classification": view(cl)}, cl.MaterialID, nil
			}},
		{Name: "declassifyHazard", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("materialId", m.materials.MaterialExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("materialId")
				cl := m.byMat[id]
				if cl == nil {
					return nil, "", policies.Refuse("not_hazardous", "The material has no classification.")
				}
				delete(m.byMat, id)
				return policies.Rec{"classification": view(cl)}, id, nil
			}},
		{Name: "listHazards", Perms: "aaa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				out := []policies.Rec{}
				for _, k := range policies.SortedKeys(m.byMat) {
					out = append(out, view(m.byMat[k]))
				}
				return policies.Rec{"hazards": out}, "", nil
			}},
	}
}
