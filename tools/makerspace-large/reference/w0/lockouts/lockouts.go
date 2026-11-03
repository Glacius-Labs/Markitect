// Package lockouts holds the lockouts that take machines out of use, and the
// operations on them.
package lockouts

import (
	"makerspace/equipment"
	"makerspace/policies"
)

// Lockout takes one machine out of use while ReleasedAt is nil.
type Lockout struct {
	ID, MachineID, Source, Reason, AppliedBy string
	AppliedAt                                int
	ReleasedAt                               *int
}

// Active reports whether the lockout is not released.
func (l *Lockout) Active() bool { return l.ReleasedAt == nil }

// Module is the lockouts module: it owns every lockout.
type Module struct {
	equipment *equipment.Module
	ids       policies.IDs
	byID      map[string]*Lockout
	list      []*Lockout
}

// New returns the module in the initial state.
func New(eq *equipment.Module) *Module {
	return &Module{equipment: eq, byID: map[string]*Lockout{}}
}

// LockoutExists reports whether the identifier names a lockout.
func (m *Module) LockoutExists(id string) bool { _, ok := m.byID[id]; return ok }

// LockedOut reports whether the machine has an active lockout.
func (m *Module) LockedOut(machineID string) bool { return m.HasActive(machineID, "") }

// HasActive reports whether the machine has an active lockout of the source;
// an empty source means any source.
func (m *Module) HasActive(machineID, source string) bool {
	for _, l := range m.list {
		if l.MachineID == machineID && l.Active() && (source == "" || l.Source == source) {
			return true
		}
	}
	return false
}

// Apply creates an active lockout of the machine with the next lockout
// identifier and returns it.
func (m *Module) Apply(machineID, source, reason, appliedBy string, at int) *Lockout {
	l := &Lockout{ID: m.ids.Next("lck"), MachineID: machineID, Source: source, Reason: reason,
		AppliedBy: appliedBy, AppliedAt: at}
	m.byID[l.ID] = l
	m.list = append(m.list, l)
	return l
}

// Release releases one lockout, whatever its source. A lockout that is
// already released is left as it is.
func (m *Module) Release(lockoutID string, at int) {
	if l := m.byID[lockoutID]; l != nil && l.Active() {
		t := at
		l.ReleasedAt = &t
	}
}

// ReleaseSource releases every active lockout of the source on the machine.
func (m *Module) ReleaseSource(machineID, source string, at int) {
	for _, l := range m.list {
		if l.MachineID == machineID && l.Source == source {
			m.Release(l.ID, at)
		}
	}
}
