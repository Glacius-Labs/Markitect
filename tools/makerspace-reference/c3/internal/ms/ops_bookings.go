package ms

import "sort"

func bookingOps() []*opDef {
	return []*opDef{
		{name: "createReservation", mutation: true, perms: "oaa",
			args: []argSpec{idRef("memberId", refMember), idRef("machineId", refMachine),
				stamp("start"), stamp("end")},
			run: runCreateReservation},
		{name: "cancelReservation", mutation: true, perms: "oaa",
			args: []argSpec{idRef("reservationId", refReservation)},
			run: func(c *call) (rec, string, *refusal) {
				r := c.s.reservations[c.str("reservationId")]
				if r.reportedStatus(c.at) != "reserved" {
					return nil, "", refuse("invalid_state", "The reservation is not in state reserved.")
				}
				r.status = "cancelled"
				return rec{"reservation": reservationView(r, c.at)}, r.id, nil
			}},
		{name: "checkIn", mutation: true, perms: "oaa",
			args: []argSpec{idRef("reservationId", refReservation)},
			run: func(c *call) (rec, string, *refusal) {
				r := c.s.reservations[c.str("reservationId")]
				if r.reportedStatus(c.at) != "reserved" {
					return nil, "", refuse("invalid_state", "The reservation is not in state reserved.")
				}
				if c.at < r.start {
					return nil, "", refuse("check_in_too_early", "The reservation has not started yet.")
				}
				m := c.s.members[r.memberID]
				if m.standing != "good" {
					return nil, "", refuse("member_suspended", "The member is suspended.")
				}
				mach := c.s.machines[r.machineID]
				if !c.s.certified(m.user.id, mach.familyID, dayOf(c.at)) {
					return nil, "", refuse("not_certified", "The member is not certified for the machine.")
				}
				at, rate := c.at, mach.rate
				r.status, r.checkedInAt, r.rate = "checkedIn", &at, &rate
				return rec{"reservation": reservationView(r, c.at)}, r.id, nil
			}},
		{name: "checkOut", mutation: true, perms: "oaa",
			args: []argSpec{idRef("reservationId", refReservation)},
			run: func(c *call) (rec, string, *refusal) {
				r := c.s.reservations[c.str("reservationId")]
				if r.status != "checkedIn" {
					return nil, "", refuse("invalid_state", "The reservation is not checked in.")
				}
				m := c.s.members[r.memberID]
				metered := c.at - *r.checkedInAt
				ch := c.s.addCharge(m.user.id, "usage", r.id, metered, *r.rate, tierDiscount(m.tier), c.at)
				at, amount := c.at, ch.amount
				r.status, r.checkedOutAt, r.metered, r.amount = "completed", &at, &metered, &amount
				return rec{"reservation": reservationView(r, c.at), "charge": chargeView(ch)}, r.id, nil
			}},
		{name: "listMemberReservations", perms: "oaa",
			args: []argSpec{idRef("memberId", refMember)},
			run: func(c *call) (rec, string, *refusal) {
				mid := c.str("memberId")
				var list []*reservation
				for _, r := range c.s.resList {
					if r.memberID == mid {
						list = append(list, r)
					}
				}
				sort.SliceStable(list, func(i, j int) bool { return list[i].start < list[j].start })
				out := make([]rec, 0, len(list))
				for _, r := range list {
					out = append(out, reservationView(r, c.at))
				}
				return rec{"memberId": mid, "reservations": out}, "", nil
			}},
		{name: "getMachineSchedule", perms: "aaa",
			args: []argSpec{idRef("machineId", refMachine), date("date")},
			run:  runMachineSchedule},
	}
}

func runCreateReservation(c *call) (rec, string, *refusal) {
	m := c.member("memberId")
	mach := c.s.machines[c.str("machineId")]
	start, end := c.num("start"), c.num("end")
	if end <= start {
		return nil, "", refuse("invalid_interval", "The end is not after the start.")
	}
	if start%granularity != 0 || end%granularity != 0 || end-start < granularity || end-start > maxReservation {
		return nil, "", refuse("invalid_slot", "The slot does not follow the booking granularity or duration limits.")
	}
	if start < c.at {
		return nil, "", refuse("starts_in_past", "The reservation starts before the request time.")
	}
	if !insideOpeningHours(start, end) {
		return nil, "", refuse("outside_opening_hours", "The slot is not inside the opening hours.")
	}
	if m.standing != "good" {
		return nil, "", refuse("member_suspended", "The member is suspended.")
	}
	if !c.s.certified(m.user.id, mach.familyID, dayOf(start)) {
		return nil, "", refuse("not_certified", "The member is not certified for the machine.")
	}
	for _, w := range c.s.windowList {
		if w.machineID == mach.id && !w.cancelled && overlaps(start, end, w.start, w.end) {
			return nil, "", refuse("machine_in_maintenance", "A maintenance window overlaps the slot.")
		}
	}
	for _, l := range c.s.lockouts {
		if l.machineID == mach.id && overlaps(start, end, l.start, l.end) {
			return nil, "", refuse("machine_locked", "A lockout of the machine overlaps the slot.")
		}
	}
	for _, r := range c.s.resList {
		if r.machineID == mach.id && r.active(c.at) && overlaps(start, end, r.start, r.end) {
			return nil, "", refuse("machine_booked", "The machine is booked in the slot.")
		}
	}
	for _, r := range c.s.resList {
		if r.memberID == m.user.id && r.active(c.at) && overlaps(start, end, r.start, r.end) {
			return nil, "", refuse("member_double_booked", "The member has another reservation in the slot.")
		}
	}
	r := &reservation{id: c.s.nextID("res"), memberID: m.user.id, machineID: mach.id,
		start: start, end: end, status: "reserved"}
	c.s.reservations[r.id] = r
	c.s.resList = append(c.s.resList, r)
	return rec{"reservation": reservationView(r, c.at)}, r.id, nil
}

func runMachineSchedule(c *call) (rec, string, *refusal) {
	mid := c.str("machineId")
	day := c.num("date")
	from, to := day*minPerDay, (day+1)*minPerDay
	var hours any
	if open, close, ok := openingHours(day); ok {
		hours = rec{"open": clock(open), "close": clock(close)}
	}
	var rs []*reservation
	for _, r := range c.s.resList {
		if r.machineID == mid && dayOf(r.start) == day && r.status != "cancelled" {
			rs = append(rs, r)
		}
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].start < rs[j].start })
	resOut := make([]rec, 0, len(rs))
	for _, r := range rs {
		resOut = append(resOut, reservationView(r, c.at))
	}
	var ws []*window
	for _, w := range c.s.windowList {
		if w.machineID == mid && !w.cancelled && overlaps(w.start, w.end, from, to) {
			ws = append(ws, w)
		}
	}
	sort.SliceStable(ws, func(i, j int) bool { return ws[i].start < ws[j].start })
	mntOut := make([]rec, 0, len(ws))
	for _, w := range ws {
		mntOut = append(mntOut, windowView(w))
	}
	var ls []*lockout
	for _, l := range c.s.lockouts {
		if l.machineID == mid && overlaps(l.start, l.end, from, to) {
			ls = append(ls, l)
		}
	}
	sort.SliceStable(ls, func(i, j int) bool { return ls[i].start < ls[j].start })
	lockOut := make([]rec, 0, len(ls))
	for _, l := range ls {
		lockOut = append(lockOut, lockoutView(l))
	}
	return rec{"machineId": mid, "date": formatDay(day), "openingHours": hours,
		"reservations": resOut, "maintenance": mntOut, "lockouts": lockOut}, "", nil
}
