// Package lockers holds the lockers members rent by the month and their
// rentals, and the operations on them.
package lockers

import (
	"makerspace/billing"
	"makerspace/members"
	"makerspace/policies"
)

// Locker is a locker that members can rent.
type Locker struct {
	ID      string
	Monthly int
}

// Rental is the rental of one locker by one member.
type Rental struct {
	ID, LockerID, MemberID string
	Active                 bool
	StartedAt              int
	EndedAt                *int
}

// Status is active or ended.
func (r *Rental) Status() string {
	if r.Active {
		return "active"
	}
	return "ended"
}

// Module is the lockers module: it owns every locker and rental.
type Module struct {
	members *members.Module
	billing *billing.Module
	ids     policies.IDs
	lockers map[string]*Locker
	byID    map[string]*Rental
	list    []*Rental
	billed  map[int]bool
}

// New returns the module in the initial state.
func New(mem *members.Module, bill *billing.Module) *Module {
	return &Module{members: mem, billing: bill, lockers: map[string]*Locker{}, byID: map[string]*Rental{},
		billed: map[int]bool{}}
}

// LockerExists reports whether the identifier names a locker.
func (m *Module) LockerExists(id string) bool { _, ok := m.lockers[id]; return ok }

// RentalExists reports whether the identifier names a rental.
func (m *Module) RentalExists(id string) bool { _, ok := m.byID[id]; return ok }

// RentalOwner is the member a rental belongs to.
func (m *Module) RentalOwner(id string) string { return m.byID[id].MemberID }

func lockerView(l *Locker) policies.Rec {
	return policies.Rec{"lockerId": l.ID, "monthlyCents": l.Monthly}
}

func rentalView(r *Rental) policies.Rec {
	return policies.Rec{"rentalId": r.ID, "lockerId": r.LockerID, "memberId": r.MemberID, "status": r.Status(),
		"startedAt": policies.FormatTime(r.StartedAt), "endedAt": policies.OptTime(r.EndedAt)}
}
