package consumables

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "setConsumption", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists),
				policies.IDRef("materialId", m.materials.MaterialExists), policies.Integer("unitsPerHour", 1, 1000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mach, mat := c.Str("machineId"), c.Str("materialId")
				if m.rows[mach] == nil {
					m.rows[mach] = map[string]int{}
				}
				m.rows[mach][mat] = c.Num("unitsPerHour")
				return policies.Rec{"row": rowView(mach, mat, c.Num("unitsPerHour"))}, mach + "/" + mat, nil
			}},
		{Name: "clearConsumption", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists),
				policies.IDRef("materialId", m.materials.MaterialExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mach, mat := c.Str("machineId"), c.Str("materialId")
				if _, ok := m.rows[mach][mat]; !ok {
					return nil, "", policies.Refuse("no_consumption", "The machine has no consumption row for the material.")
				}
				delete(m.rows[mach], mat)
				return policies.Rec{"machineId": mach, "materialId": mat}, mach + "/" + mat, nil
			}},
		{Name: "listConsumption", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mach := c.Str("machineId")
				out := []policies.Rec{}
				for _, mat := range policies.SortedKeys(m.rows[mach]) {
					out = append(out, rowView(mach, mat, m.rows[mach][mat]))
				}
				return policies.Rec{"machineId": mach, "rows": out}, "", nil
			}},
	}
}
