// Package escalations holds the escalation policies of the tool families and
// the escalations of incidents, and the operations on them.
package escalations

import (
	"makerspace/bookings"
	"makerspace/equipment"
	"makerspace/notices"
	"makerspace/policies"
	"makerspace/safety"
	"makerspace/workorders"
)

type escalation struct {
	id, incidentID, machineID, workOrderID string
	cancelled                              []string
	at                                     int
}

// Module is the escalations module: it owns every policy and escalation.
type Module struct {
	equipment  *equipment.Module
	safety     *safety.Module
	bookings   *bookings.Module
	workorders *workorders.Module
	notices    *notices.Module
	ids        policies.IDs
	policies   map[string]string // familyId -> minSeverity
	byIncident map[string]*escalation
	list       []*escalation
}

// New returns the module in the initial state: no policy, no escalation.
func New(eq *equipment.Module, sf *safety.Module, bk *bookings.Module, wo *workorders.Module,
	ntc *notices.Module) *Module {
	return &Module{equipment: eq, safety: sf, bookings: bk, workorders: wo, notices: ntc,
		policies: map[string]string{}, byIncident: map[string]*escalation{}}
}

var severityRank = map[string]int{"low": 0, "medium": 1, "high": 2}

func view(e *escalation) policies.Rec {
	ids := append([]string{}, e.cancelled...)
	return policies.Rec{"escalationId": e.id, "incidentId": e.incidentID, "machineId": e.machineID,
		"workOrderId": e.workOrderID, "cancelledReservationIds": ids, "at": policies.FormatTime(e.at)}
}
