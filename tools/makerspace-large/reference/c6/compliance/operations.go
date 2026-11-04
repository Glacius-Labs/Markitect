package compliance

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "checkEligibility", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists),
				policies.IDRef("machineId", m.equipment.MachineExists), policies.Date("date")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				blockers := m.Blockers(c.Str("memberId"), c.Str("machineId"), c.Num("date"), policies.DayOf(c.At))
				return policies.Rec{"memberId": c.Str("memberId"), "machineId": c.Str("machineId"),
					"date": c.Str("date"), "eligible": len(blockers) == 0, "blockers": blockers}, "", nil
			}},
		{Name: "complianceSummary", Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(m.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, day := c.Str("memberId"), policies.DayOf(c.At)
				rows := []policies.Rec{}
				for _, f := range m.equipment.FamilyIDs() {
					rows = append(rows, m.familyRow(mid, f, day))
				}
				return policies.Rec{"memberId": mid, "standing": m.members.Member(mid).Standing,
					"minor": m.guardians.IsMinor(mid, day), "families": rows}, "", nil
			}},
	}
}
