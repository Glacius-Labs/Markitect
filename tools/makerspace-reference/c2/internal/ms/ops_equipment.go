package ms

func equipmentOps() []*opDef {
	return []*opDef{
		{name: "addFamily", mutation: true, perms: "nna",
			args: []argSpec{id("familyId"), text("name")},
			run: func(c *call) (rec, string, *refusal) {
				fid := c.str("familyId")
				if c.s.families[fid] != nil {
					return nil, "", refuse("already_exists", "A family with identifier %s already exists.", fid)
				}
				f := &family{id: fid, name: c.str("name")}
				c.s.families[fid] = f
				return rec{"family": familyView(f)}, fid, nil
			}},
		{name: "addMachine", mutation: true, perms: "nna",
			args: []argSpec{id("machineId"), text("name"), idRef("familyId", refFamily),
				integer("rateCentsPerMinute", 1, 10000)},
			run: func(c *call) (rec, string, *refusal) {
				mid := c.str("machineId")
				if c.s.machines[mid] != nil {
					return nil, "", refuse("already_exists", "A machine with identifier %s already exists.", mid)
				}
				m := &machine{id: mid, name: c.str("name"), familyID: c.str("familyId"),
					rate: c.num("rateCentsPerMinute")}
				c.s.machines[mid] = m
				return rec{"machine": machineView(m)}, mid, nil
			}},
		{name: "setMachineRate", mutation: true, perms: "nna",
			args: []argSpec{idRef("machineId", refMachine), integer("rateCentsPerMinute", 1, 10000)},
			run: func(c *call) (rec, string, *refusal) {
				m := c.s.machines[c.str("machineId")]
				m.rate = c.num("rateCentsPerMinute")
				return rec{"machine": machineView(m)}, m.id, nil
			}},
		{name: "listMachines", perms: "aaa",
			run: func(c *call) (rec, string, *refusal) {
				out := []rec{}
				for _, k := range sortedKeys(c.s.machines) {
					out = append(out, machineView(c.s.machines[k]))
				}
				return rec{"machines": out}, "", nil
			}},
		{name: "scheduleMaintenance", mutation: true, perms: "naa",
			args: []argSpec{idRef("machineId", refMachine), stamp("start"), stamp("end"), text("reason")},
			run: func(c *call) (rec, string, *refusal) {
				mid, start, end := c.str("machineId"), c.num("start"), c.num("end")
				if end <= start {
					return nil, "", refuse("invalid_interval", "The end is not after the start.")
				}
				if start < c.at {
					return nil, "", refuse("starts_in_past", "The window starts before the request time.")
				}
				for _, r := range c.s.resList {
					if r.machineID == mid && r.active(c.at) && overlaps(start, end, r.start, r.end) {
						return nil, "", refuse("reservation_conflict", "An active reservation overlaps the window.")
					}
				}
				w := &window{id: c.s.nextID("mnt"), machineID: mid, start: start, end: end, reason: c.str("reason")}
				c.s.windows[w.id] = w
				c.s.windowList = append(c.s.windowList, w)
				return rec{"window": windowView(w)}, w.id, nil
			}},
		{name: "cancelMaintenance", mutation: true, perms: "naa",
			args: []argSpec{idRef("windowId", refWindow)},
			run: func(c *call) (rec, string, *refusal) {
				w := c.s.windows[c.str("windowId")]
				if w.cancelled || w.start <= c.at {
					return nil, "", refuse("invalid_state", "The window cannot be cancelled.")
				}
				w.cancelled = true
				return rec{"window": windowView(w)}, w.id, nil
			}},
	}
}
