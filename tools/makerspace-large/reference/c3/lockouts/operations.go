package lockouts

import "makerspace/policies"

// LockoutView is the lockout as it appears in results.
func LockoutView(l *Lockout) policies.Rec {
	return policies.Rec{"lockoutId": l.ID, "machineId": l.MachineID, "source": l.Source, "reason": l.Reason,
		"appliedBy": l.AppliedBy, "appliedAt": policies.FormatTime(l.AppliedAt),
		"releasedAt": policies.OptTime(l.ReleasedAt)}
}

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "applyLockout", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists), policies.Text("reason")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				l := m.Apply(c.Str("machineId"), "manual", c.Str("reason"), c.ActorID, c.At)
				return policies.Rec{"lockout": LockoutView(l)}, l.ID, nil
			}},
		{Name: "releaseLockout", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("lockoutId", m.LockoutExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				l := m.byID[c.Str("lockoutId")]
				if !l.Active() {
					return nil, "", policies.Refuse("invalid_state", "The lockout is already released.")
				}
				if l.Source != "manual" {
					return nil, "", policies.Refuse("lockout_not_manual", "The lockout was not applied by hand.")
				}
				if l.AppliedBy != c.ActorID && c.ActorRole != "admin" {
					return nil, "", policies.Refuse("not_lockout_holder", "The actor did not apply the lockout and is not an admin.")
				}
				m.Release(l.ID, c.At)
				return policies.Rec{"lockout": LockoutView(l)}, l.ID, nil
			}},
		{Name: "listLockouts", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.equipment.MachineExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("machineId")
				out := []policies.Rec{}
				for _, l := range m.list {
					if l.MachineID == mid {
						out = append(out, LockoutView(l))
					}
				}
				return policies.Rec{"machineId": mid, "lockouts": out}, "", nil
			}},
	}
}
