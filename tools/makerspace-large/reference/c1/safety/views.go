package safety

import "makerspace/policies"

func incidentView(i *Incident) policies.Rec {
	status := "open"
	var resolution, resolvedAt any
	if i.Resolved {
		status = "resolved"
		resolution = i.Resolution
		resolvedAt = policies.FormatTime(i.ResolvedAt)
	}
	return policies.Rec{"incidentId": i.ID, "machineId": i.MachineID, "severity": i.Severity,
		"description": i.Description, "reportedBy": i.ReportedBy,
		"reportedAt": policies.FormatTime(i.ReportedAt), "status": status,
		"resolution": resolution, "resolvedAt": resolvedAt}
}
