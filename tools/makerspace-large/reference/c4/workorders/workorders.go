// Package workorders holds the repair jobs on machines, which lock a machine
// out and use spare parts, and the operations on them.
package workorders

import (
	"makerspace/equipment"
	"makerspace/lockouts"
	"makerspace/members"
	"makerspace/parts"
	"makerspace/policies"
	"makerspace/safety"
)

// WorkOrder is a repair job on one machine.
type WorkOrder struct {
	ID, MachineID, Summary string
	Closed                 bool
	IncidentID             string // "" while no incident is linked
	LockoutID              string
	PartsCost              int
	OpenedBy               string
	OpenedAt               int
	ClosedAt               *int
	Note                   *string
}

// Module is the workorders module: it owns every work order.
type Module struct {
	members   *members.Module
	equipment *equipment.Module
	safety    *safety.Module
	parts     *parts.Module
	lockouts  *lockouts.Module
	ids       policies.IDs
	byID      map[string]*WorkOrder
	list      []*WorkOrder
}

// New returns the module in the initial state.
func New(mem *members.Module, eq *equipment.Module, sf *safety.Module, pt *parts.Module,
	lk *lockouts.Module) *Module {
	return &Module{members: mem, equipment: eq, safety: sf, parts: pt, lockouts: lk, byID: map[string]*WorkOrder{}}
}

// WorkOrderExists reports whether the identifier names a work order.
func (m *Module) WorkOrderExists(id string) bool { _, ok := m.byID[id]; return ok }

func (m *Module) open(machineID, summary, openerID string, at int) *WorkOrder {
	l := m.lockouts.Apply(machineID, "workOrder", summary, openerID, at)
	w := &WorkOrder{ID: m.ids.Next("wko"), MachineID: machineID, Summary: summary, LockoutID: l.ID,
		OpenedBy: openerID, OpenedAt: at}
	m.byID[w.ID] = w
	m.list = append(m.list, w)
	return w
}

// OpenLinked opens a work order for the machine with the summary and the
// opener, already linked to the incident of that machine, as openWorkOrder
// followed by linkIncident do, without their refusals and without an audit
// entry. It returns the work order.
func (m *Module) OpenLinked(machineID, summary, openerID, incidentID string, at int) *WorkOrder {
	w := m.open(machineID, summary, openerID, at)
	w.IncidentID = incidentID
	return w
}

// View is the work order as it appears in results.
func View(w *WorkOrder) policies.Rec {
	status := "open"
	if w.Closed {
		status = "closed"
	}
	var incident, note any
	if w.IncidentID != "" {
		incident = w.IncidentID
	}
	if w.Note != nil {
		note = *w.Note
	}
	return policies.Rec{"workOrderId": w.ID, "machineId": w.MachineID, "summary": w.Summary, "status": status,
		"incidentId": incident, "lockoutId": w.LockoutID, "partsCostCents": w.PartsCost, "openedBy": w.OpenedBy,
		"openedAt": policies.FormatTime(w.OpenedAt), "closedAt": policies.OptTime(w.ClosedAt), "note": note}
}
