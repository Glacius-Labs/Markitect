package ms

import "sort"

type interval struct{ from, to int }

// unionMinutes is the number of minutes covered by at least one interval,
// counting only the part inside [lo,hi).
func unionMinutes(ivs []interval, lo, hi int) int {
	var clipped []interval
	for _, iv := range ivs {
		a, b := max(iv.from, lo), min(iv.to, hi)
		if a < b {
			clipped = append(clipped, interval{a, b})
		}
	}
	sort.Slice(clipped, func(i, j int) bool { return clipped[i].from < clipped[j].from })
	total, end := 0, lo
	for _, iv := range clipped {
		a := max(iv.from, end)
		if iv.to > a {
			total += iv.to - a
			end = iv.to
		}
	}
	return total
}

func reportOps() []*opDef {
	return []*opDef{
		{name: "utilizationReport", perms: "naa",
			args: []argSpec{date("from"), date("to")},
			run:  runUtilization},
		{name: "revenueReport", perms: "naa",
			args: []argSpec{month("month")},
			run:  runRevenue},
	}
}

func utilizationPercent(used, available int) int {
	if available == 0 {
		return 0
	}
	return (used*100*2 + available) / (available * 2)
}

func runUtilization(c *call) (rec, string, *refusal) {
	from, to := c.num("from"), c.num("to")
	if to <= from || to-from > 366 {
		return nil, "", refuse("invalid_interval", "The report range is invalid.")
	}
	rangeFrom, rangeTo := from*minPerDay, to*minPerDay
	rows := []rec{}
	var tOpen, tMaint, tLock, tAvail, tUsed int
	for _, mid := range sortedKeys(c.s.machines) {
		var maint []interval
		for _, w := range c.s.windowList {
			if w.machineID == mid && !w.cancelled {
				maint = append(maint, interval{w.start, w.end})
			}
		}
		var locks, both []interval
		for _, l := range c.s.lockouts {
			if l.machineID == mid {
				locks = append(locks, interval{l.start, l.end})
			}
		}
		both = append(append(both, maint...), locks...)
		openMin, maintMin, lockMin, blockedMin := 0, 0, 0, 0
		for day := from; day < to; day++ {
			open, close, ok := openingHours(day)
			if !ok {
				continue
			}
			lo, hi := day*minPerDay+open, day*minPerDay+close
			openMin += hi - lo
			maintMin += unionMinutes(maint, lo, hi)
			lockMin += unionMinutes(locks, lo, hi)
			blockedMin += unionMinutes(both, lo, hi)
		}
		used := 0
		for _, r := range c.s.resList {
			if r.machineID == mid && r.status == "completed" {
				used += max(0, min(*r.checkedOutAt, rangeTo)-max(*r.checkedInAt, rangeFrom))
			}
		}
		avail := openMin - blockedMin
		rows = append(rows, rec{"machineId": mid, "openMinutes": openMin, "maintenanceMinutes": maintMin,
			"lockoutMinutes": lockMin, "availableMinutes": avail, "usedMinutes": used, "utilizationPercent": utilizationPercent(used, avail)})
		tOpen += openMin
		tMaint += maintMin
		tLock += lockMin
		tAvail += avail
		tUsed += used
	}
	totals := rec{"openMinutes": tOpen, "maintenanceMinutes": tMaint, "lockoutMinutes": tLock,
		"availableMinutes": tAvail,
		"usedMinutes":      tUsed, "utilizationPercent": utilizationPercent(tUsed, tAvail)}
	return rec{"from": formatDay(from), "to": formatDay(to), "machines": rows, "totals": totals}, "", nil
}

func runRevenue(c *call) (rec, string, *refusal) {
	idx := c.num("month")
	var count, membership, usage, materials, courses, credit, outstanding, payments, issued int
	for _, i := range c.s.invoiceList {
		if i.month != idx {
			continue
		}
		count++
		credit += i.creditApplied
		outstanding += i.balance
		for _, l := range i.lines {
			switch l.kind {
			case "membership":
				membership += l.amount
			case "usage":
				usage += l.amount
			case "material":
				materials += l.amount
			case "course":
				courses += l.amount
			}
		}
	}
	for _, p := range c.s.payments {
		if monthOf(p.at) == idx {
			payments += p.amount
		}
	}
	for _, e := range c.s.creditEvents {
		if monthOf(e.at) == idx {
			issued += e.amount
		}
	}
	return rec{"month": formatMonth(idx), "invoicesGenerated": c.s.invoiced[idx], "invoiceCount": count,
		"membershipCents": membership, "usageCents": usage, "materialCents": materials,
		"courseCents": courses, "invoicedCents": membership + usage + materials + courses,
		"creditAppliedCents": credit, "outstandingCents": outstanding, "paymentsCents": payments,
		"creditsIssuedCents": issued}, "", nil
}
