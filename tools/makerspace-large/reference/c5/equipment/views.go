package equipment

import "makerspace/policies"

func familyView(f *Family) policies.Rec {
	return policies.Rec{"familyId": f.ID, "name": f.Name}
}

func machineView(m *Machine) policies.Rec {
	return policies.Rec{"machineId": m.ID, "name": m.Name, "familyId": m.FamilyID,
		"rateCentsPerMinute": m.Rate}
}

// WindowView is the maintenance window as it appears in results.
func WindowView(w *Window) policies.Rec {
	status := "scheduled"
	if w.Cancelled {
		status = "cancelled"
	}
	return policies.Rec{"windowId": w.ID, "machineId": w.MachineID, "start": policies.FormatTime(w.Start),
		"end": policies.FormatTime(w.End), "reason": w.Reason, "status": status}
}
