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
		{Name: "createReservation", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.MemberRef(b.members.Exists), policies.IDRef("machineId", b.equipment.MachineExists),
				policies.Stamp("start"), policies.Stamp("end")},
			Run: b.runCreateReservation},
		{Name: "cancelReservation", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("reservationId", b.ReservationExists, b.ReservationOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				r := b.reservations[c.Str("reservationId")]
				if !b.CancelReserved(r.id, c.At) {
					return nil, "", policies.Refuse("invalid_state", "The reservation is not in state reserved.")
				}
				return policies.Rec{"reservation": reservationView(r, c.At)}, r.id, nil
			}},
		{Name: "checkIn", Mutation: true, Perms: "oaa",
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
				if b.guardians.ConsentMissing(m.User.ID, mach.FamilyID, policies.DayOf(c.At)) {
					return nil, "", policies.Refuse("guardian_consent_missing", "The member is a minor without consent for the family.")
				}
				if !b.training.Certified(m.User.ID, mach.FamilyID, policies.DayOf(c.At)) {
					return nil, "", policies.Refuse("not_certified", "The member is not certified for the machine.")
				}
				if b.waivers.Missing(m.User.ID, mach.FamilyID, policies.DayOf(c.At)) {
					return nil, "", policies.Refuse("waiver_missing", "The member has not signed the current waiver of the family.")
				}
				if b.ppe.Missing(m.User.ID, mach.FamilyID) {
					return nil, "", policies.Refuse("ppe_missing", "The member does not hold every item the family requires.")
				}
				if b.lockouts.LockedOut(mach.ID) {
					return nil, "", policies.Refuse("machine_locked_out", "The machine is locked out.")
				}
				if b.needs.BadgeOutside(m.User.ID) {
					return nil, "", policies.Refuse("not_on_premises", "The member has an active badge that is not inside.")
				}
				if b.loans.HasOverdue(m.User.ID, policies.DayOf(c.At)) {
					return nil, "", policies.Refuse("loan_overdue", "The member has an overdue tool loan.")
				}
				at, rate := c.At, mach.Rate
				r.status, r.checkedInAt, r.rate = "checkedIn", &at, &rate
				return policies.Rec{"reservation": reservationView(r, c.At)}, r.id, nil
			}},
		{Name: "checkOut", Mutation: true, Perms: "oaa",
			Args: []policies.ArgSpec{policies.OwnedIDRef("reservationId", b.ReservationExists, b.ReservationOwner)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				r := b.reservations[c.Str("reservationId")]
				if r.status != "checkedIn" {
					return nil, "", policies.Refuse("invalid_state", "The reservation is not checked in.")
				}
				m := b.members.Member(r.memberID)
				metered := c.At - *r.checkedInAt
				billable := metered - b.packs.Draw(m.User.ID, metered, c.At)
				ch := b.billing.AddCharge(m.User.ID, "usage", r.id, billable, *r.rate, members.TierDiscount(m.Tier), c.At)
				b.consumables.Draw(r.id, m.User.ID, r.machineID, metered, c.At)
				at, amount := c.At, ch.Amount
				r.status, r.checkedOutAt, r.metered, r.amount = "completed", &at, &metered, &amount
				return policies.Rec{"reservation": reservationView(r, c.At), "charge": billing.ChargeView(ch)}, r.id, nil
			}},
		{Name: "listMemberReservations", Perms: "oaa",
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
		{Name: "getMachineSchedule", Perms: "aaa",
			Args: []policies.ArgSpec{policies.IDRef("machineId", b.equipment.MachineExists), policies.Date("date")},
			Run:  b.runMachineSchedule},
		{Name: "chargeNoShows", Mutation: true, Perms: "naa",
			Run: b.runChargeNoShows},
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
	if b.guardians.ConsentMissing(m.User.ID, mach.FamilyID, policies.DayOf(start)) {
		return nil, "", policies.Refuse("guardian_consent_missing", "The member is a minor without consent for the family.")
	}
	if !b.training.Certified(m.User.ID, mach.FamilyID, policies.DayOf(start)) {
		return nil, "", policies.Refuse("not_certified", "The member is not certified for the machine.")
	}
	if b.waivers.Missing(m.User.ID, mach.FamilyID, policies.DayOf(c.At)) {
		return nil, "", policies.Refuse("waiver_missing", "The member has not signed the current waiver of the family.")
	}
	if b.lockouts.LockedOut(mach.ID) {
		return nil, "", policies.Refuse("machine_locked_out", "The machine is locked out.")
	}
	if b.inspections.DueOn(mach.ID, policies.DayOf(start)) {
		return nil, "", policies.Refuse("inspection_due", "The machine is due for inspection on the date of the slot.")
	}
	if len(b.equipment.ScheduledWindows(mach.ID, start, end)) > 0 {
		return nil, "", policies.Refuse("machine_in_maintenance", "A maintenance window overlaps the slot.")
	}
	if b.staffing.SupervisorMissing(mach.FamilyID, start, end) {
		return nil, "", policies.Refuse("supervisor_required", "The slot is not covered by scheduled shifts.")
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

// runChargeNoShows charges half the price of the blocked time for every missed
// reservation that no run has charged and whose machine was not locked out in
// the slot. The reservations are in creation order, which is ascending
// reservation number.
func (b *Module) runChargeNoShows(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	out := []policies.Rec{}
	for _, r := range b.resList {
		if r.reportedStatus(c.At) != "missed" || r.noShowCharged || b.lockouts.OverlapsInterval(r.machineID, r.start, r.end) {
			continue
		}
		ch := b.billing.AddCharge(r.memberID, "noShow", r.id, r.end-r.start, b.equipment.Machine(r.machineID).Rate, 50, c.At)
		r.noShowCharged = true
		out = append(out, billing.ChargeView(ch))
	}
	return policies.Rec{"charges": out}, "noShows", nil
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
	return policies.Rec{"machineId": mid, "date": policies.FormatDay(day), "openingHours": hours,
		"reservations": resOut, "maintenance": mntOut}, "", nil
}
