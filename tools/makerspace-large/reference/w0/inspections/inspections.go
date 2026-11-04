// Package inspections holds the inspection plan of each machine and records
// the outcomes of inspections, and the operations on them.
package inspections

import (
	"makerspace/equipment"
	"makerspace/lockouts"
	"makerspace/policies"
)

// Plan is the inspection plan of one machine. Days are days since 2000-01-01.
type Plan struct {
	MachineID    string
	IntervalDays int
	LastPassedOn int
	Failed       bool
}

// NextDueOn is the day IntervalDays days after LastPassedOn.
func (p *Plan) NextDueOn() int { return p.LastPassedOn + p.IntervalDays }

// Module is the inspections module: it owns every inspection plan.
type Module struct {
	equipment *equipment.Module
	lockouts  *lockouts.Module
	plans     map[string]*Plan
}

// New returns the module in the initial state.
func New(eq *equipment.Module, lk *lockouts.Module) *Module {
	return &Module{equipment: eq, lockouts: lk, plans: map[string]*Plan{}}
}

// PlanOf returns a copy of the machine's plan, or nil when it has none.
func (m *Module) PlanOf(machineID string) *Plan {
	p := m.plans[machineID]
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

// DueOn reports whether the machine has a plan and the day is on or after the
// plan's nextDueOn.
func (m *Module) DueOn(machineID string, day int) bool {
	p := m.plans[machineID]
	return p != nil && day >= p.NextDueOn()
}

func planView(p *Plan) policies.Rec {
	return policies.Rec{"machineId": p.MachineID, "intervalDays": p.IntervalDays,
		"lastPassedOn": policies.FormatDay(p.LastPassedOn), "nextDueOn": policies.FormatDay(p.NextDueOn()),
		"failed": p.Failed}
}
