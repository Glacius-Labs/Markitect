package inspections

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "setInspectionPlan", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists), policies.Integer("intervalDays", 1, 3650)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("machineId")
				p := m.plans[mid]
				if p == nil {
					p = &Plan{MachineID: mid, LastPassedOn: policies.DayOf(c.At)}
					m.plans[mid] = p
				}
				p.IntervalDays = c.Num("intervalDays")
				return policies.Rec{"plan": planView(p)}, mid, nil
			}},
		{Name: "recordInspection", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists), policies.Enum("outcome", "passed", "failed"),
				policies.Text("note")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("machineId")
				p := m.plans[mid]
				if p == nil {
					return nil, "", policies.Refuse("no_plan", "The machine has no inspection plan.")
				}
				if c.Str("outcome") == "passed" {
					p.LastPassedOn, p.Failed = policies.DayOf(c.At), false
					m.lockouts.ReleaseSource(mid, "inspection", c.At)
				} else {
					p.Failed = true
					if !m.lockouts.HasActive(mid, "inspection") {
						m.lockouts.Apply(mid, "inspection", c.Str("note"), c.ActorID, c.At)
					}
				}
				return policies.Rec{"plan": planView(p)}, mid, nil
			}},
		{Name: "getInspectionStatus", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("machineId")
				p := m.plans[mid]
				if p == nil {
					return nil, "", policies.Refuse("no_plan", "The machine has no inspection plan.")
				}
				return policies.Rec{"plan": planView(p), "due": m.DueOn(mid, policies.DayOf(c.At))}, "", nil
			}},
		{Name: "listInspectionsDue", Perms: "naa",
			Args: []policies.ArgSpec{policies.Date("date")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				out := []policies.Rec{}
				for _, mid := range policies.SortedKeys(m.plans) {
					if p := m.plans[mid]; p.NextDueOn() <= c.Num("date") {
						out = append(out, planView(p))
					}
				}
				return policies.Rec{"date": c.Str("date"), "plans": out}, "", nil
			}},
	}
}
