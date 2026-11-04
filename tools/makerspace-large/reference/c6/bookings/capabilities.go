package bookings

import (
	"sort"
)

// Reservation is a copy of a reservation as the reservation record shows it,
// with its status as reported at the request time. The pointers are nil for the
// fields that are null.
type Reservation struct {
	ID, MemberID, MachineID, Status string
	Start, End                      int
	CheckedInAt, CheckedOutAt       *int
	RateCentsPerMinute              *int
	MeteredMinutes, AmountCents     *int
}

func copyOf(r *reservation, at int) Reservation {
	return Reservation{ID: r.id, MemberID: r.memberID, MachineID: r.machineID, Status: r.reportedStatus(at),
		Start: r.start, End: r.end, CheckedInAt: r.checkedInAt, CheckedOutAt: r.checkedOutAt,
		RateCentsPerMinute: r.rate, MeteredMinutes: r.metered, AmountCents: r.amount}
}

// ReservationOf is the reservation with the identifier, with the status
// reported at the request time at. The last result is false when the
// identifier names no reservation.
func (b *Module) ReservationOf(id string, at int) (Reservation, bool) {
	r := b.reservations[id]
	if r == nil {
		return Reservation{}, false
	}
	return copyOf(r, at), true
}

// ActiveReservation is an active reservation of a machine.
type ActiveReservation struct {
	MachineID  string
	Start, End int
}

// ActiveReservationsOfFamily lists the active reservations at the request time
// at of all machines of the family, by machine identifier ascending, then in
// creation order.
func (b *Module) ActiveReservationsOfFamily(familyID string, at int) []ActiveReservation {
	var out []ActiveReservation
	for _, id := range b.equipment.MachineIDs() {
		if b.equipment.Machine(id).FamilyID != familyID {
			continue
		}
		for _, iv := range b.ActiveIntervals(id, at) {
			out = append(out, ActiveReservation{MachineID: id, Start: iv.Start, End: iv.End})
		}
	}
	return out
}

// UpcomingReservations lists the reservations of the machine whose reported
// status at the request time at is reserved and whose start is after at, in
// ascending start order, then creation order.
func (b *Module) UpcomingReservations(machineID string, at int) []Reservation {
	var out []Reservation
	for _, r := range b.resList {
		if r.machineID == machineID && r.reportedStatus(at) == "reserved" && r.start > at {
			out = append(out, copyOf(r, at))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

// CancelReserved cancels the reservation as cancelReservation does, with no fee
// and no credit, and answers true. It changes nothing and answers false when
// the reported status at the request time at is not reserved.
func (b *Module) CancelReserved(id string, at int) bool {
	r := b.reservations[id]
	if r == nil || r.reportedStatus(at) != "reserved" {
		return false
	}
	r.status = "cancelled"
	return true
}

// OpeningHoursOfDay is the opening hours of one day: the clock times of the
// record and the same as minutes of the day.
type OpeningHoursOfDay struct {
	Open, Close             string
	OpenMinute, CloseMinute int
}

// OpeningHoursOf is the opening hours of the date. The last result is false on
// a Sunday, which has none.
func (b *Module) OpeningHoursOf(day int) (OpeningHoursOfDay, bool) {
	open, close, ok := OpeningHours(day)
	if !ok {
		return OpeningHoursOfDay{}, false
	}
	return OpeningHoursOfDay{Open: clock(open), Close: clock(close), OpenMinute: open, CloseMinute: close}, true
}
