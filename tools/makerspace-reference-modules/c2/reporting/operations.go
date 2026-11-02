package reporting

import (
	"makerspace/bookings"
	"makerspace/policies"
)

// Operations lists the operations of the module.
func (r *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "utilizationReport", Perms: "naa",
			Args: []policies.ArgSpec{policies.Date("from"), policies.Date("to")},
			Run:  r.runUtilization},
		{Name: "revenueReport", Perms: "naa",
			Args: []policies.ArgSpec{policies.Month("month")},
			Run:  r.runRevenue},
	}
}

func (r *Module) runUtilization(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	from, to := c.Num("from"), c.Num("to")
	if to <= from || to-from > 366 {
		return nil, "", policies.Refuse("invalid_interval", "The report range is invalid.")
	}
	rangeFrom, rangeTo := from*policies.MinPerDay, to*policies.MinPerDay
	rows := []policies.Rec{}
	var tOpen, tMaint, tLock, tAvail, tUsed int
	for _, mid := range r.equipment.MachineIDs() {
		var maint []interval
		for _, w := range r.equipment.ScheduledWindows(mid, rangeFrom, rangeTo) {
			maint = append(maint, interval{w.Start, w.End})
		}
		var locks, both []interval
		for _, l := range r.bookings.LockoutsOf(mid) {
			locks = append(locks, interval{l.Start, l.End})
		}
		both = append(append(both, maint...), locks...)
		openMin, maintMin, lockMin, blockedMin := 0, 0, 0, 0
		for day := from; day < to; day++ {
			open, close, ok := bookings.OpeningHours(day)
			if !ok {
				continue
			}
			lo, hi := day*policies.MinPerDay+open, day*policies.MinPerDay+close
			openMin += hi - lo
			maintMin += unionMinutes(maint, lo, hi)
			lockMin += unionMinutes(locks, lo, hi)
			blockedMin += unionMinutes(both, lo, hi)
		}
		used := 0
		for _, s := range r.bookings.CompletedSessions(mid) {
			used += max(0, min(s.CheckedOutAt, rangeTo)-max(s.CheckedInAt, rangeFrom))
		}
		avail := openMin - blockedMin
		rows = append(rows, policies.Rec{"machineId": mid, "openMinutes": openMin, "maintenanceMinutes": maintMin,
			"lockoutMinutes": lockMin, "availableMinutes": avail, "usedMinutes": used, "utilizationPercent": utilizationPercent(used, avail)})
		tOpen += openMin
		tMaint += maintMin
		tLock += lockMin
		tAvail += avail
		tUsed += used
	}
	totals := policies.Rec{"openMinutes": tOpen, "maintenanceMinutes": tMaint, "lockoutMinutes": tLock,
		"availableMinutes": tAvail,
		"usedMinutes":      tUsed, "utilizationPercent": utilizationPercent(tUsed, tAvail)}
	return policies.Rec{"from": policies.FormatDay(from), "to": policies.FormatDay(to), "machines": rows, "totals": totals}, "", nil
}

func (r *Module) runRevenue(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
	idx := c.Num("month")
	var count, membership, usage, materials, courses, credit, outstanding, payments, issued int
	for _, i := range r.billing.Invoices() {
		if i.Month != idx {
			continue
		}
		count++
		credit += i.CreditApplied
		outstanding += i.Balance
		for _, l := range i.Lines {
			switch l.Kind {
			case "membership":
				membership += l.Amount
			case "usage":
				usage += l.Amount
			case "material":
				materials += l.Amount
			case "course":
				courses += l.Amount
			}
		}
	}
	for _, p := range r.billing.Payments() {
		if policies.MonthOf(p.At) == idx {
			payments += p.Amount
		}
	}
	for _, e := range r.billing.CreditEvents() {
		if policies.MonthOf(e.At) == idx {
			issued += e.Amount
		}
	}
	return policies.Rec{"month": policies.FormatMonth(idx), "invoicesGenerated": r.billing.Invoiced(idx), "invoiceCount": count,
		"membershipCents": membership, "usageCents": usage, "materialCents": materials,
		"courseCents": courses, "invoicedCents": membership + usage + materials + courses,
		"creditAppliedCents": credit, "outstandingCents": outstanding, "paymentsCents": payments,
		"creditsIssuedCents": issued}, "", nil
}
