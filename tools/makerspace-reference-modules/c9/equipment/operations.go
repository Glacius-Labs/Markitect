package equipment

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addFamily", Mutation: true, Perms: "nnan",
			Args: []policies.ArgSpec{policies.ID("familyId"), policies.Text("name")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				fid := c.Str("familyId")
				if m.families[fid] != nil {
					return nil, "", policies.Refuse("already_exists", "A family with identifier %s already exists.", fid)
				}
				f := &Family{ID: fid, Name: c.Str("name")}
				m.families[fid] = f
				return policies.Rec{"family": familyView(f)}, fid, nil
			}},
		{Name: "addMachine", Mutation: true, Perms: "nnan",
			Args: []policies.ArgSpec{policies.ID("machineId"), policies.Text("name"), policies.IDRef("familyId", m.FamilyExists),
				policies.Integer("rateCentsPerMinute", 1, 10000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("machineId")
				if m.machines[mid] != nil {
					return nil, "", policies.Refuse("already_exists", "A machine with identifier %s already exists.", mid)
				}
				mach := &Machine{ID: mid, Name: c.Str("name"), FamilyID: c.Str("familyId"),
					Rate: c.Num("rateCentsPerMinute")}
				m.machines[mid] = mach
				return policies.Rec{"machine": machineView(mach)}, mid, nil
			}},
		{Name: "setMachineRate", Mutation: true, Perms: "nnan",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.MachineExists), policies.Integer("rateCentsPerMinute", 1, 10000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mach := m.machines[c.Str("machineId")]
				mach.Rate = c.Num("rateCentsPerMinute")
				return policies.Rec{"machine": machineView(mach)}, mach.ID, nil
			}},
		{Name: "listMachines", Perms: "aaaa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				out := []policies.Rec{}
				for _, k := range policies.SortedKeys(m.machines) {
					out = append(out, machineView(m.machines[k]))
				}
				return policies.Rec{"machines": out}, "", nil
			}},
		{Name: "scheduleMaintenance", Mutation: true, Perms: "naaa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", m.MachineExists), policies.Stamp("start"), policies.Stamp("end"), policies.Text("reason")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid, start, end := c.Str("machineId"), c.Num("start"), c.Num("end")
				if end <= start {
					return nil, "", policies.Refuse("invalid_interval", "The end is not after the start.")
				}
				if start < c.At {
					return nil, "", policies.Refuse("starts_in_past", "The window starts before the request time.")
				}
				if m.needs.ActiveReservationOverlaps(mid, start, end, c.At) {
					return nil, "", policies.Refuse("reservation_conflict", "An active reservation overlaps the window.")
				}
				w := &Window{ID: m.ids.Next("mnt"), MachineID: mid, Start: start, End: end, Reason: c.Str("reason")}
				m.windows[w.ID] = w
				m.windowList = append(m.windowList, w)
				return policies.Rec{"window": WindowView(w)}, w.ID, nil
			}},
		{Name: "cancelMaintenance", Mutation: true, Perms: "naaa",
			Args: []policies.ArgSpec{policies.IDRef("windowId", m.WindowExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				w := m.windows[c.Str("windowId")]
				if w.Cancelled || w.Start <= c.At {
					return nil, "", policies.Refuse("invalid_state", "The window cannot be cancelled.")
				}
				w.Cancelled = true
				return policies.Rec{"window": WindowView(w)}, w.ID, nil
			}},
	}
}
