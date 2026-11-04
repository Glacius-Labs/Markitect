// Package safety holds the safety incidents recorded per machine, and the
// operations on them.
package safety

import (
	"makerspace/equipment"
	"makerspace/policies"
)

// Incident is a safety incident recorded against a machine.
type Incident struct {
	ID, MachineID string
	Severity      string
	Description   string
	ReportedBy    string
	ReportedAt    int
	Resolved      bool
	Resolution    string
	ResolvedAt    int
}

// Module is the safety module: it owns every incident.
type Module struct {
	equipment    *equipment.Module
	ids          policies.IDs
	incidents    map[string]*Incident
	incidentList []*Incident
}

// New returns the module in the initial state.
func New(eq *equipment.Module) *Module {
	return &Module{equipment: eq, incidents: map[string]*Incident{}}
}

// IncidentExists reports whether the identifier names an incident.
func (s *Module) IncidentExists(id string) bool { _, ok := s.incidents[id]; return ok }

// Incident returns the incident with the identifier, or nil.
func (s *Module) Incident(id string) *Incident { return s.incidents[id] }
