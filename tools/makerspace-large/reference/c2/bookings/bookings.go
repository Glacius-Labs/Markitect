// Package bookings holds the reservations of machines, their check-in and
// check-out, the opening hours and the booking rules, and the operations on
// them.
package bookings

import (
	"makerspace/billing"
	"makerspace/consumables"
	"makerspace/equipment"
	"makerspace/guardians"
	"makerspace/inspections"
	"makerspace/lockouts"
	"makerspace/members"
	"makerspace/passes"
	"makerspace/policies"
	"makerspace/ppe"
	"makerspace/staffing"
	"makerspace/training"
	"makerspace/waivers"
)

type reservation struct {
	id, memberID, machineID string
	start, end              int
	status                  string // stored: reserved, checkedIn, completed, cancelled
	checkedInAt             *int
	checkedOutAt            *int
	rate                    *int
	metered                 *int
	amount                  *int
}

// reportedStatus applies the time-dependent missed state.
func (r *reservation) reportedStatus(at int) string {
	if r.status == "reserved" && r.end <= at {
		return "missed"
	}
	return r.status
}

func (r *reservation) active(at int) bool {
	st := r.reportedStatus(at)
	return st == "reserved" || st == "checkedIn"
}

// Session is the metered interval of a completed reservation.
type Session struct{ CheckedInAt, CheckedOutAt int }

// Needs is what bookings asks of the program without depending on the module
// that knows the answer: the protocol module supplies it from access.
type Needs struct {
	// BadgeOutside reports whether the member has an active badge that is not
	// inside (premises gate).
	BadgeOutside func(memberID string) bool
}

// Module is the bookings module: it owns every reservation.
type Module struct {
	needs        Needs
	members      *members.Module
	training     *training.Module
	equipment    *equipment.Module
	billing      *billing.Module
	guardians    *guardians.Module
	waivers      *waivers.Module
	ppe          *ppe.Module
	lockouts     *lockouts.Module
	inspections  *inspections.Module
	staffing     *staffing.Module
	passes       *passes.Module
	consumables  *consumables.Module
	ids          policies.IDs
	reservations map[string]*reservation
	resList      []*reservation
}

// New returns the module in the initial state.
func New(needs Needs, mem *members.Module, tr *training.Module, eq *equipment.Module, bill *billing.Module,
	gd *guardians.Module, wv *waivers.Module, pp *ppe.Module, lk *lockouts.Module, ins *inspections.Module,
	stf *staffing.Module, pas *passes.Module, cons *consumables.Module) *Module {
	return &Module{
		needs:        needs,
		members:      mem,
		training:     tr,
		equipment:    eq,
		billing:      bill,
		guardians:    gd,
		waivers:      wv,
		ppe:          pp,
		lockouts:     lk,
		inspections:  ins,
		staffing:     stf,
		passes:       pas,
		consumables:  cons,
		reservations: map[string]*reservation{},
	}
}

// ReservationExists reports whether the identifier names a reservation.
func (b *Module) ReservationExists(id string) bool { _, ok := b.reservations[id]; return ok }

// ReservationOwner is the member a reservation belongs to.
func (b *Module) ReservationOwner(id string) string { return b.reservations[id].memberID }

// ActiveReservationOverlaps reports whether an active reservation of the
// machine overlaps [start,end) at the request time at.
func (b *Module) ActiveReservationOverlaps(machineID string, start, end, at int) bool {
	for _, r := range b.resList {
		if r.machineID == machineID && r.active(at) && policies.Overlaps(start, end, r.start, r.end) {
			return true
		}
	}
	return false
}

// CompletedSessions lists the metered sessions of the machine's completed
// reservations.
func (b *Module) CompletedSessions(machineID string) []Session {
	var out []Session
	for _, r := range b.resList {
		if r.machineID == machineID && r.status == "completed" {
			out = append(out, Session{CheckedInAt: *r.checkedInAt, CheckedOutAt: *r.checkedOutAt})
		}
	}
	return out
}
