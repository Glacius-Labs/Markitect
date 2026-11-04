// Package staffing holds the shifts of staff and admins and the supervision
// setting of each tool family, and the operations on them.
package staffing

import (
	"sort"

	"makerspace/equipment"
	"makerspace/members"
	"makerspace/policies"
)

// maxShift is the longest shift in minutes.
const maxShift = 720

// Shift is a period in which a staff user or an admin is on duty.
type Shift struct {
	ID, UserID string
	Start, End int
	Cancelled  bool
}

// Reservation is an active reservation of a machine, as staffing asks for it.
type Reservation struct {
	MachineID  string
	Start, End int
}

// Needs is what staffing asks of the program without depending on the module
// that knows the answer: the protocol module supplies it from bookings.
type Needs struct {
	// ActiveReservations lists the active reservations of all machines of the
	// family at the request time at.
	ActiveReservations func(familyID string, at int) []Reservation
}

// Module is the staffing module: it owns every shift and supervision setting.
type Module struct {
	needs      Needs
	members    *members.Module
	equipment  *equipment.Module
	ids        policies.IDs
	shifts     map[string]*Shift
	shiftList  []*Shift
	supervised map[string]bool
}

// New returns the module in the initial state: every family is unsupervised.
func New(needs Needs, mem *members.Module, eq *equipment.Module) *Module {
	return &Module{needs: needs, members: mem, equipment: eq, shifts: map[string]*Shift{},
		supervised: map[string]bool{}}
}

// ShiftExists reports whether the identifier names a shift.
func (m *Module) ShiftExists(id string) bool { _, ok := m.shifts[id]; return ok }

// Supervised reports whether the family needs a person on duty.
func (m *Module) Supervised(familyID string) bool { return m.supervised[familyID] }

// Covered reports whether every minute of [start,end) lies in at least one
// scheduled shift.
func (m *Module) Covered(start, end int) bool { return m.covered(start, end, "") }

// SupervisorMissing reports whether a reservation of a machine of the family
// in [start,end) is refused supervisor_required: the family is supervised and
// the slot is not covered.
func (m *Module) SupervisorMissing(familyID string, start, end int) bool {
	return m.supervised[familyID] && !m.Covered(start, end)
}

// OnShift reports whether the user has a scheduled shift that contains the
// point in time: start <= at < end.
func (m *Module) OnShift(userID string, at int) bool {
	for _, s := range m.shiftList {
		if !s.Cancelled && s.UserID == userID && s.Start <= at && at < s.End {
			return true
		}
	}
	return false
}

// covered is Covered without the shift with the excluded identifier.
func (m *Module) covered(start, end int, excluded string) bool {
	var iv []*Shift
	for _, s := range m.shiftList {
		if !s.Cancelled && s.ID != excluded && s.End > start && s.Start < end {
			iv = append(iv, s)
		}
	}
	sort.SliceStable(iv, func(i, j int) bool { return iv[i].Start < iv[j].Start })
	at := start
	for _, s := range iv {
		if s.Start > at {
			return false
		}
		if s.End > at {
			at = s.End
		}
		if at >= end {
			return true
		}
	}
	return at >= end
}

// needed reports whether cancelling the shift would leave an active reservation
// of a supervised family without cover: one that overlaps the shift, is
// covered now and would not be covered without the shift.
func (m *Module) needed(sh *Shift, at int) bool {
	for _, fid := range policies.SortedKeys(m.supervised) {
		if !m.supervised[fid] {
			continue
		}
		for _, r := range m.needs.ActiveReservations(fid, at) {
			if policies.Overlaps(r.Start, r.End, sh.Start, sh.End) && m.covered(r.Start, r.End, "") &&
				!m.covered(r.Start, r.End, sh.ID) {
				return true
			}
		}
	}
	return false
}

func shiftView(s *Shift) policies.Rec {
	status := "scheduled"
	if s.Cancelled {
		status = "cancelled"
	}
	return policies.Rec{"shiftId": s.ID, "userId": s.UserID, "start": policies.FormatTime(s.Start),
		"end": policies.FormatTime(s.End), "status": status}
}
