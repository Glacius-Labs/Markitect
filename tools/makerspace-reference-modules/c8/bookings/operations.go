package bookings

import (
	"sort"

	"makerspace/billing"
	"makerspace/equipment"
	"makerspace/members"
	"makerspace/policies"
)

// Operations lists the operations of the module.
func (b *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "createReservation", Mutation: true, Perms: "oaan",
			Args: []policies.ArgSpec{policies.MemberRef(b.members.Exists), policies.IDRef("machineId", b.equipment.MachineExists),
				policies.Stamp("start"), policies.Stamp("end")},
			Run: b.runCreateReservation},
		{Name: "cancelReservation", Mutation: true, Perms: "oaan",
			Args: []policies.ArgSpec{policies.OwnedIDRef("reservationId", b.ReservationExists, b.ReservationOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				r := b.reservations[c.Str("reservationId")]
				if r.reportedStatus(c.At) != "reserved" {
					return nil, "", policies.Refuse("invalid_state", "The reservation is not in state reserved.")
				}
				r.status = "cancelled"
				return policies.Rec{"reservation": reservationView(r, c.At)}, r.id, nil
			}},
		{Name: "checkIn", Mutation: true, Perms: "oaan",
			Args: []policies.ArgSpec{policies.OwnedIDRef("reservationId", b.ReservationExists, b.ReservationOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				r := b.reservations[c.Str("reservationId")]
				if r.reportedStatus(c.At) != "reserved" {
					return nil, "", policies.Refuse("invalid_state", "The reservation is not in state reserved.")
				}
				if c.At < r.start {
					return nil, "", policies.Refuse("check_in_too_early", "The reservation has not started yet.")
				}
				m := b.members.Member(r.memberID)
				if m.Standing != "good" {
					return nil, "", policies.Refuse("member_suspended", "The member is suspended.")
				}
				mach := b.equipment.Machine(r.machineID)
				if !b.training.Certified(m.User.ID, mach.FamilyID, policies.DayOf(c.At)) {
					return nil, "", policies.Refuse("not_certified", "The member is not certified for the machine.")
				}
				at, rate := c.At, mach.Rate
				r.status, r.checkedInAt, r.rate = "checkedIn", &at, &rate
				return policies.Rec{"reservation": reservationView(r, c.At)}, r.id, nil
			}},
		{Name: "checkOut", Mutation: true, Perms: "oaan",
			Args: []policies.ArgSpec{policies.OwnedIDRef("reservationId", b.ReservationExists, b.ReservationOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				r := b.reservations[c.Str("reservationId")]
				if r.status != "checkedIn" {
					return nil, "", policies.Refuse("invalid_state", "The reservation is not checked in.")
				}
				m := b.members.Member(r.memberID)
				metered := c.At - *r.checkedInAt
				ch := b.billing.AddCharge(m.User.ID, "usage", r.id, metered, *r.rate, members.TierDiscount(m.Tier), c.At)
				at, amount := c.At, ch.Amount
				r.status, r.checkedOutAt, r.metered, r.amount = "completed", &at, &metered, &amount
				return policies.Rec{"reservation": reservationView(r, c.At), "charge": billing.ChargeView(ch)}, r.id, nil
			}},
		{Name: "listMemberReservations", Perms: "oaan",
			Args: []policies.ArgSpec{policies.MemberRef(b.members.Exists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				mid := c.Str("memberId")
				var list []*reservation
				for _, r := range b.resList {
					if r.memberID == mid {
						list = append(list, r)
					}
				}
				sort.SliceStable(list, func(i, j int) bool { return list[i].start < list[j].start })
				out := make([]policies.Rec, 0, len(list))
				for _, r := range list {
					out = append(out, reservationView(r, c.At))
				}
				return policies.Rec{"memberId": mid, "reservations": out}, "", nil
			}},
		{Name: "getMachineSchedule", Perms: "aaaa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", b.equipment.MachineExists), policies.Date("date")},
			Run:  b.runMachineSchedule},
		{Name: "lockMachine", Mutation: true, Perms: "naaa",
			Args: []policies.ArgSpec{policies.IDRef("incidentId", b.safety.IncidentExists), policies.Stamp("start"), policies.Stamp("end")},
			Run:  b.runLockMachine},
	}
}

func (b *Module) runCreateReservation(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	m := b.members.Member(c.Str("memberId"))
	mach := b.equipment.Machine(c.Str("machineId"))
	start, end := c.Num("start"), c.Num("end")
	if end <= start {
		return nil, "", policies.Refuse("invalid_interval", "The end is not after the start.")
	}
	if start%granularity != 0 || end%granularity != 0 || end-start < granularity || end-start > maxReservation {
		return nil, "", policies.Refuse("invalid_slot", "The slot does not follow the booking granularity or duration limits.")
	}
	if start < c.At {
		return nil, "", policies.Refuse("starts_in_past", "The reservation starts before the request time.")
	}
	if !insideOpeningHours(start, end) {
		return nil, "", policies.Refuse("outside_opening_hours", "The slot is not inside the opening hours.")
	}
	if m.Standing != "good" {
		return nil, "", policies.Refuse("member_suspended", "The member is suspended.")
	}
	if b.billing.Blocked(m.User.ID, c.At) {
		return nil, "", policies.Refuse("account_overdue", "The member has an invoice that is more than 30 days overdue.")
	}
	if !b.training.Certified(m.User.ID, mach.FamilyID, policies.DayOf(start)) {
		return nil, "", policies.Refuse("not_certified", "The member is not certified for the machine.")
	}
	if len(b.equipment.ScheduledWindows(mach.ID, start, end)) > 0 {
		return nil, "", policies.Refuse("machine_in_maintenance", "A maintenance window overlaps the slot.")
	}
	for _, l := range b.lockouts {
		if l.MachineID == mach.ID && policies.Overlaps(start, end, l.Start, l.End) {
			return nil, "", policies.Refuse("machine_locked", "A lockout of the machine overlaps the slot.")
		}
	}
	if b.ActiveReservationOverlaps(mach.ID, start, end, c.At) {
		return nil, "", policies.Refuse("machine_booked", "The machine is booked in the slot.")
	}
	for _, r := range b.resList {
		if r.memberID == m.User.ID && r.active(c.At) && policies.Overlaps(start, end, r.start, r.end) {
			return nil, "", policies.Refuse("member_double_booked", "The member has another reservation in the slot.")
		}
	}
	r := &reservation{id: b.ids.Next("res"), memberID: m.User.ID, machineID: mach.ID,
		start: start, end: end, status: "reserved"}
	b.reservations[r.id] = r
	b.resList = append(b.resList, r)
	return policies.Rec{"reservation": reservationView(r, c.At)}, r.id, nil
}

func (b *Module) runMachineSchedule(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	mid := c.Str("machineId")
	day := c.Num("date")
	from, to := day*policies.MinPerDay, (day+1)*policies.MinPerDay
	var hours any
	if open, close, ok := OpeningHours(day); ok {
		hours = policies.Rec{"open": clock(open), "close": clock(close)}
	}
	var rs []*reservation
	for _, r := range b.resList {
		if r.machineID == mid && policies.DayOf(r.start) == day && r.status != "cancelled" {
			rs = append(rs, r)
		}
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].start < rs[j].start })
	resOut := make([]policies.Rec, 0, len(rs))
	for _, r := range rs {
		resOut = append(resOut, reservationView(r, c.At))
	}
	ws := b.equipment.ScheduledWindows(mid, from, to)
	sort.SliceStable(ws, func(i, j int) bool { return ws[i].Start < ws[j].Start })
	mntOut := make([]policies.Rec, 0, len(ws))
	for _, w := range ws {
		mntOut = append(mntOut, equipment.WindowView(w))
	}
	var ls []*Lockout
	for _, l := range b.lockouts {
		if l.MachineID == mid && policies.Overlaps(l.Start, l.End, from, to) {
			ls = append(ls, l)
		}
	}
	sort.SliceStable(ls, func(i, j int) bool { return ls[i].Start < ls[j].Start })
	lockOut := make([]policies.Rec, 0, len(ls))
	for _, l := range ls {
		lockOut = append(lockOut, lockoutView(l))
	}
	return policies.Rec{"machineId": mid, "date": policies.FormatDay(day), "openingHours": hours,
		"reservations": resOut, "maintenance": mntOut, "lockouts": lockOut}, "", nil
}

// lockoutCreditCents is the credit a member receives for each reservation a
// lockout cancels.
const lockoutCreditCents = 1000

func (b *Module) runLockMachine(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	inc := b.safety.Incident(c.Str("incidentId"))
	start, end := c.Num("start"), c.Num("end")
	if inc.Resolved {
		return nil, "", policies.Refuse("invalid_state", "The incident is resolved.")
	}
	if end <= start {
		return nil, "", policies.Refuse("invalid_interval", "The end is not after the start.")
	}
	if start < c.At {
		return nil, "", policies.Refuse("starts_in_past", "The lockout starts before the request time.")
	}
	l := &Lockout{ID: b.ids.Next("lock"), IncidentID: inc.ID, MachineID: inc.MachineID, Start: start, End: end}
	b.lockouts = append(b.lockouts, l)
	cancelled := []policies.Rec{}
	for _, r := range b.resList {
		if r.machineID != l.MachineID || r.reportedStatus(c.At) != "reserved" || !policies.Overlaps(start, end, r.start, r.end) {
			continue
		}
		r.status = "cancelled"
		l.CancelledIDs = append(l.CancelledIDs, r.id)
		b.billing.GrantCredit(r.memberID, lockoutCreditCents, c.At)
		cancelled = append(cancelled, reservationView(r, c.At))
	}
	return policies.Rec{"lockout": lockoutView(l), "cancelledReservations": cancelled}, l.ID, nil
}
