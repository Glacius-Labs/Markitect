// Package equipment holds the tool families, the machines with their rate and
// the maintenance windows, and the operations on them.
package equipment

import "makerspace/policies"

// Family is a tool family.
type Family struct{ ID, Name string }

// Machine belongs to a family and has a rate per minute.
type Machine struct {
	ID, Name, FamilyID string
	Rate               int
}

// Window is a maintenance window of a machine.
type Window struct {
	ID, MachineID string
	Start, End    int
	Reason        string
	Cancelled     bool
}

// Needs is what equipment asks of the program without depending on the module
// that knows the answer: the protocol module supplies it from bookings.
type Needs struct {
	// ActiveReservationOverlaps reports whether an active reservation of the
	// machine overlaps [start,end) at the request time at.
	ActiveReservationOverlaps func(machineID string, start, end, at int) bool
}

// Module is the equipment module: it owns every family, machine and window.
type Module struct {
	needs      Needs
	ids        policies.IDs
	families   map[string]*Family
	machines   map[string]*Machine
	windows    map[string]*Window
	windowList []*Window
}

// New returns the module in the initial state.
func New(needs Needs) *Module {
	return &Module{
		needs:    needs,
		families: map[string]*Family{},
		machines: map[string]*Machine{},
		windows:  map[string]*Window{},
	}
}

// FamilyExists reports whether the identifier names a family.
func (m *Module) FamilyExists(id string) bool { _, ok := m.families[id]; return ok }

// MachineExists reports whether the identifier names a machine.
func (m *Module) MachineExists(id string) bool { _, ok := m.machines[id]; return ok }

// WindowExists reports whether the identifier names a maintenance window.
func (m *Module) WindowExists(id string) bool { _, ok := m.windows[id]; return ok }

// Machine returns the machine with the identifier, or nil.
func (m *Module) Machine(id string) *Machine { return m.machines[id] }

// MachineIDs lists the identifiers of all machines in ascending order.
func (m *Module) MachineIDs() []string { return policies.SortedKeys(m.machines) }

// ScheduledWindows lists the windows of the machine that are not cancelled and
// overlap [from,to), in creation order.
func (m *Module) ScheduledWindows(machineID string, from, to int) []*Window {
	var out []*Window
	for _, w := range m.windowList {
		if w.MachineID == machineID && !w.Cancelled && policies.Overlaps(w.Start, w.End, from, to) {
			out = append(out, w)
		}
	}
	return out
}

// FamilyIDs lists the identifiers of all families in ascending order.
func (m *Module) FamilyIDs() []string { return policies.SortedKeys(m.families) }
